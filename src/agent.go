package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// saveMessageTurn writes an individual conversational turn to a numbered, timestamped JSON file.
// It also broadcasts the turn to any listening browser GUI.
func saveMessageTurn(msg Message) {
	currentTurnNumber++
	timestamp := time.Now().Format("2006_01_02_15-04-05")
	filename := fmt.Sprintf("%03d-%s-%s.json", currentTurnNumber, msg.Role, timestamp)
	sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
	filePath := filepath.Join(sessionPath, filename)

	jsonBytes, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		fmt.Printf("%s[ERROR] Failed to serialize message turn: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	_ = os.WriteFile(filePath, jsonBytes, 0644)
	fmt.Printf("%s[TURN SECURED] Saved turn %d to disk: %s%s\n", ColorBlue, currentTurnNumber, filename, ColorReset)

	// Broadcast to Web Console (Phase 6.2)
	BroadcastSSE("turn_secured", map[string]interface{}{
		"turn_number": currentTurnNumber,
		"role":        msg.Role,
		"name":        msg.Name,
		"content":     msg.Content,
		"tool_calls":  msg.ToolCalls,
	})
}

// loadHistoryFromFiles reads all uncompacted turns from disk inside the active session directory.
func loadHistoryFromFiles() []Message {
	sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
	entries, err := os.ReadDir(sessionPath)
	if err != nil {
		return nil
	}

	var history []Message
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 4 && name[3] == '-' {
			filePath := filepath.Join(sessionPath, name)
			fileBytes, err := os.ReadFile(filePath)
			if err == nil {
				var msg Message
				if err := json.Unmarshal(fileBytes, &msg); err == nil {
					history = append(history, msg)
				}
			}
		}
	}
	return history
}

// executeSessionRollback rewinds the workspace state and message history back to a specific target turn.
func executeSessionRollback(targetTurn int) {
	fmt.Printf("\n%s🔄 [ROLLBACK] Rewinding state and folder structure back to Turn %d...%s\n", ColorBold+ColorMagenta, targetTurn, ColorReset)

	sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
	entries, err := os.ReadDir(sessionPath)
	if err != nil {
		fmt.Printf("%s[ERROR] Failed to read session path: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	deletedTurnsCount := 0
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
			turnIdxStr := name[:3]
			turnIdx, err := strconv.Atoi(turnIdxStr)
			if err == nil && turnIdx > targetTurn {
				_ = os.Remove(filepath.Join(sessionPath, name))
				deletedTurnsCount++
			}
		}
	}

	restoreWorkspaceBackups(targetTurn)

	currentTurnNumber = targetTurn
	fmt.Printf("%s✅ [SUCCESS] Deleted %d newer turn log(s). Workspace files rolled back. Ready at Turn %d.%s\n", ColorGreen, deletedTurnsCount, targetTurn, ColorReset)
}

// backupWorkspaceFile stashes a file before we overwrite or patch it
func backupWorkspaceFile(relativePath string) {
	srcPath := filepath.Join(activeConfig.Agent.WorkspaceDir, relativePath)
	
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return // File doesn't exist yet, nothing to back up
	}

	backupDir := filepath.Join(".goharness", "sessions", activeSessionID, "backups", fmt.Sprintf("turn-%d", currentTurnNumber+1))
	os.MkdirAll(backupDir, 0755)

	destPath := filepath.Join(backupDir, relativePath)
	os.MkdirAll(filepath.Dir(destPath), 0755)

	_ = os.WriteFile(destPath, content, 0644)
}

// restoreWorkspaceBackups restores files to their Turn-specific states on fork rollbacks
func restoreWorkspaceBackups(targetTurn int) {
	backupRoot := filepath.Join(".goharness", "sessions", activeSessionID, "backups")
	
	targetBackupFolder := filepath.Join(backupRoot, fmt.Sprintf("turn-%d", targetTurn+1))
	if _, err := os.Stat(targetBackupFolder); err == nil {
		fmt.Printf("%s  ↳ Restoring original file contents from backup: turn-%d...%s\n", ColorMagenta, targetTurn+1, ColorReset)
		filepath.Walk(targetBackupFolder, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(targetBackupFolder, path)
			destPath := filepath.Join(activeConfig.Agent.WorkspaceDir, rel)
			
			content, err := os.ReadFile(path)
			if err == nil {
				_ = os.WriteFile(destPath, content, 0644)
				fmt.Printf("    - Restored: %s\n", rel)
			}
			return nil
		})
	}

	entries, err := os.ReadDir(backupRoot)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "turn-") {
				turnNumStr := strings.TrimPrefix(entry.Name(), "turn-")
				turnNum, err := strconv.Atoi(turnNumStr)
				if err == nil && turnNum > targetTurn+1 {
					_ = os.RemoveAll(filepath.Join(backupRoot, entry.Name()))
				}
			}
		}
	}
}

// executeSlidingWindowCompaction triggers a background LLM call to compress history.
func executeSlidingWindowCompaction(history []Message) {
	limit := activeConfig.Compaction.AutoCompactTurns
	keepLastN := activeConfig.Compaction.KeepLastN

	if len(history) < limit || len(history) <= keepLastN {
		return
	}

	compactionEndIndex := len(history) - keepLastN
	boundaryTurnNumber := compactionEndIndex

	fmt.Printf("\n%s⚡ [COMPACTION] Turn limit reached (%d turns). Compacting turns 1 to %d while keeping the last %d turns untouched...%s\n", 
		ColorBold+ColorMagenta, len(history), boundaryTurnNumber, keepLastN, ColorReset)

	var historyToCompact strings.Builder
	for i := 0; i < compactionEndIndex; i++ {
		msg := history[i]
		if msg.Role == "system" {
			continue
		}
		historyToCompact.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, msg.Role, msg.Content))
		for _, tc := range msg.ToolCalls {
			historyToCompact.WriteString(fmt.Sprintf("    Tool Request: %s(%s)\n", tc.Function.Name, tc.Function.Arguments))
		}
	}

	compactionPrompt := fmt.Sprintf("Please summarize the following conversational execution history. Maintain a tight, bulleted summary listing (1) files created or modified, (2) current bugs resolved, and (3) the outstanding task list. Keep it highly dense.\n\nConversation Logs:\n%s", historyToCompact.String())

	reqBody := ChatCompletionRequest{
		Model: activeConfig.Compaction.Model,
		Messages: []Message{
			{Role: "system", Content: activeConfig.Compaction.SystemPrompt},
			{Role: "user", Content: compactionPrompt},
		},
		Temperature: activeConfig.Compaction.Temperature,
	}

	jsonBytes, _ := json.Marshal(reqBody)

	respMsg, err := sendLLMRequest(jsonBytes)
	if err != nil {
		fmt.Printf("%s[WARNING] Context Compaction API call failed: %v. Skipping compaction.%s\n", ColorYellow, err, ColorReset)
		return
	}

	summaryPath := filepath.Join(".goharness", "sessions", activeSessionID, "compacted_summary.json")
	_ = os.WriteFile(summaryPath, []byte(respMsg.Content), 0644)

	boundaryPath := filepath.Join(".goharness", "sessions", activeSessionID, "compaction_boundary.txt")
	_ = os.WriteFile(boundaryPath, []byte(strconv.Itoa(boundaryTurnNumber)), 0644)

	fmt.Printf("%s✅ [SUCCESS] Rolling Context Compaction completed. Saved state up to Turn %d.%s\n", ColorGreen, boundaryTurnNumber, ColorReset)

	// Broadcast to Web Console (Phase 6.2)
	BroadcastSSE("compaction", map[string]interface{}{
		"boundary_turn": boundaryTurnNumber,
	})
}

// getCompactedSummary reads the summary from disk if it exists
func getCompactedSummary() (string, int) {
	summaryPath := filepath.Join(".goharness", "sessions", activeSessionID, "compacted_summary.json")
	boundaryPath := filepath.Join(".goharness", "sessions", activeSessionID, "compaction_boundary.txt")

	summaryBytes, err := os.ReadFile(summaryPath)
	if err != nil {
		return "", 0
	}

	boundaryBytes, err := os.ReadFile(boundaryPath)
	if err != nil {
		return "", 0
	}

	boundary, _ := strconv.Atoi(string(boundaryBytes))
	return string(summaryBytes), boundary
}

// LoadLocalInstructions scans the current directory for AGENTS.md or SKILLS.md
func LoadLocalInstructions() string {
	var instructions []string
	targets := []string{"AGENTS.md", "SKILLS.md", "INSTRUCTIONS.md"}
	
	for _, filename := range targets {
		if content, err := os.ReadFile(filename); err == nil {
			fmt.Printf("%s[INJECTION] Injecting custom guidelines from %s (%d bytes)%s\n", ColorMagenta, filename, len(content), ColorReset)
			header := fmt.Sprintf("\n=== SYSTEM GUIDELINE (%s) ===\n", filename)
			instructions = append(instructions, header+string(content))
		}
	}
	return strings.Join(instructions, "\n")
}

// GenerateWorkspaceTree walks the workspace and generates a tree string.
func GenerateWorkspaceTree(rootPath string, config DirectoryScanConfig) (string, error) {
	var sb strings.Builder
	sb.WriteString("=== ACTIVE WORKSPACE DIRECTORY TREE ===\n")
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		sb.WriteString("(Directory empty or untracked)\n")
		return sb.String(), nil
	}
	err := scanDir(rootPath, "", 0, config, &sb)
	return sb.String(), err
}

func scanDir(currentPath, prefix string, depth int, config DirectoryScanConfig, sb *strings.Builder) error {
	if depth > config.MaxDepth {
		sb.WriteString(fmt.Sprintf("%s└── ... [Max Scan Depth %d Reached]\n", prefix, config.MaxDepth))
		return nil
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	var dirs []os.DirEntry
	var files []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if shouldIgnore(name, config.IgnoredPatterns) {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	for i, dir := range dirs {
		name := dir.Name()
		isLast := (i == len(dirs)-1) && (len(files) == 0)
		
		char := "├── "
		nextPrefix := prefix + "│   "
		if isLast {
			char = "└── "
			nextPrefix = prefix + "    "
		}

		if shouldCollapse(name, config.CollapsedPatterns) {
			sb.WriteString(fmt.Sprintf("%s%s%s/ [collapsed]\n", prefix, char, name))
			continue
		}

		sb.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, char, name))
		err := scanDir(filepath.Join(currentPath, name), nextPrefix, depth+1, config, sb)
		if err != nil {
			return err
		}
	}

	fileLimit := config.MaxFilesPerDirectory
	for i, file := range files {
		name := file.Name()
		isLast := i == len(files)-1

		if i >= fileLimit {
			truncatedCount := len(files) - fileLimit
			sb.WriteString(fmt.Sprintf("%s└── ... (%d more files truncated)\n", prefix, truncatedCount))
			break
		}

		char := "├── "
		if isLast {
			char = "└── "
		}

		metadataNote := ""
		if info, err := file.Info(); err == nil {
			modTime := info.ModTime()
			diff := time.Since(modTime)
			if diff < 1*time.Minute {
				metadataNote = " [Modified < 1m ago]"
			} else if diff < 60*time.Minute {
				metadataNote = fmt.Sprintf(" [Modified %dm ago]", int(diff.Minutes()))
			} else if diff < 24*time.Hour {
				metadataNote = fmt.Sprintf(" [Modified %dh ago]", int(diff.Hours()))
			}
		}

		sb.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, char, name, metadataNote))
	}

	return nil
}

func shouldIgnore(name string, patterns []string) bool {
	for _, p := range patterns {
		if strings.EqualFold(name, p) {
			return true
		}
	}
	return false
}

func shouldCollapse(name string, patterns []string) bool {
	for _, p := range patterns {
		if strings.EqualFold(name, p) {
			return true
		}
	}
	return false
}
