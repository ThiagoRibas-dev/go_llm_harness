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
	sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
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
	sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
	if activeConfig.Debug {
		fmt.Printf("[DEBUG] Loading history from directory: %s\n", sessionPath)
	}
	
	entries, err := os.ReadDir(sessionPath)
	if err != nil {
		if activeConfig.Debug {
			fmt.Printf("%s[DEBUG WARNING] Failed to read session directory: %v%s\n", ColorYellow, err, ColorReset)
		}
		return []Message{} // Return empty slice instead of nil (Phase 8.6)
	}

	if activeConfig.Debug {
		fmt.Printf("[DEBUG] Found %d total directory entries inside session folder\n", len(entries))
	}

	var history []Message
	for _, entry := range entries {
		name := entry.Name()
		
		isDir := entry.IsDir()
		isJson := strings.HasSuffix(name, ".json")
		longEnough := len(name) > 4
		hasDash := len(name) > 3 && name[3] == '-'

		if !isDir && isJson && longEnough && hasDash {
			filePath := filepath.Join(sessionPath, name)
			fileBytes, err := os.ReadFile(filePath)
			if err != nil {
				if activeConfig.Debug {
					fmt.Printf("%s[DEBUG WARNING] Failed to read file %s: %v%s\n", ColorYellow, name, err, ColorReset)
				}
				continue
			}

			var msg Message
			if err := json.Unmarshal(fileBytes, &msg); err == nil {
				history = append(history, msg)
				if activeConfig.Debug {
					fmt.Printf("  ➔ Loaded Turn file successfully: %s (Role: %s)\n", name, msg.Role)
				}
			} else {
				fmt.Printf("%s[UNMARSHAL WARNING] Failed to parse turn file %s: %v%s\n", ColorYellow, name, err, ColorReset)
			}
		} else {
			if activeConfig.Debug && name != "meta.json" && name != "compacted_summary.json" && name != "compaction_boundary.txt" && name != "backups" {
				fmt.Printf("  [DEBUG] File '%s' did not match history filter (isDir=%t, isJson=%t, len=%d, hasDash=%t)\n", name, isDir, isJson, len(name), hasDash)
			}
		}
	}

	if history == nil {
		history = []Message{}
	}

	if activeConfig.Debug {
		fmt.Printf("[DEBUG] Completed loading. Loaded %d valid history messages.\n", len(history))
	}
	return history
}

// findMaxTurnNumber scans both the root session path and any compacted folders to find the absolute maximum turn number
func findMaxTurnNumber(sessionID string) int {
	sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", sessionID))
	maxTurn := 0

	// 1. Scan root of session path
	entries, err := os.ReadDir(sessionPath)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
				turnIdx, err := strconv.Atoi(name[:3])
				if err == nil && turnIdx > maxTurn {
					maxTurn = turnIdx
				}
			}
		}
	}

	// 2. Scan inside any compacted subdirectories
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "compacted_summary_up_to_turn_") {
				subDirPath := filepath.Join(sessionPath, entry.Name())
				_ = filepath.Walk(subDirPath, func(path string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".json") && len(info.Name()) > 3 {
						turnIdx, err := strconv.Atoi(info.Name()[:3])
						if err == nil && turnIdx > maxTurn {
							maxTurn = turnIdx
						}
					}
					return nil
				})
			}
		}
	}

	return maxTurn
}

// executeSessionRollback rewinds the workspace state and message history back to a specific target turn.
func executeSessionRollback(targetTurn int) {
	fmt.Printf("\n%s🔄 [ROLLBACK] Rewinding state and folder structure back to Turn %d...%s\n", ColorBold+ColorMagenta, targetTurn, ColorReset)

	sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))

	// 1. Move files back from any matching compacted summary folders if they are less than or equal to the targetTurn
	entries, err := os.ReadDir(sessionPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "compacted_summary_up_to_turn_") {
				subDirPath := filepath.Join(sessionPath, entry.Name())
				// We recursively walk this subdirectory and pull matching files back to root
				_ = filepath.Walk(subDirPath, func(path string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".json") && len(info.Name()) > 3 {
						turnIdx, err := strconv.Atoi(info.Name()[:3])
						if err == nil && turnIdx <= targetTurn {
							_ = os.Rename(path, filepath.Join(sessionPath, info.Name()))
						}
					}
					return nil
				})
			}
		}
	}

	// 2. Delete main session files that are greater than targetTurn
	entries, err = os.ReadDir(sessionPath)
	if err != nil {
		fmt.Printf("%s[ERROR] Failed to read session path: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	deletedMainCount := 0
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
			turnIdxStr := name[:3]
			turnIdx, err := strconv.Atoi(turnIdxStr)
			if err == nil && turnIdx > targetTurn {
				_ = os.Remove(filepath.Join(sessionPath, name))
				deletedMainCount++
			}
		}
	}

	// 3. Delete any compacted subfolders and summary files that are greater than targetTurn
	entries, err = os.ReadDir(sessionPath)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "compacted_summary_up_to_turn_") {
				boundaryStr := strings.TrimPrefix(name, "compacted_summary_up_to_turn_")
				boundaryStr = strings.TrimSuffix(boundaryStr, ".json")
				boundary, err := strconv.Atoi(boundaryStr)
				if err == nil && boundary > targetTurn {
					_ = os.RemoveAll(filepath.Join(sessionPath, name))
				}
			}
		}
	}

	// 4. Update the compaction boundary on rollback inside meta.json
	boundary := getSessionCompactionBoundary(activeSessionID)
	if targetTurn <= boundary {
		// Rollback has entered the compacted zone! We must reset the boundary in meta.json
		updateSessionCompactionBoundary(activeSessionID, 0)
		fmt.Printf("%s  ↳ Rollback crossed compaction boundary. Reset compaction tracking in meta.json.%s\n", ColorYellow, ColorReset)
	}

	restoreWorkspaceBackups(targetTurn)

	currentTurnNumber = targetTurn
	fmt.Printf("%s✅ [SUCCESS] Deleted newer turn log(s). Workspace files rolled back. Ready at Turn %d.%s\n", ColorGreen, targetTurn, ColorReset)
}

// executeSlidingWindowCompaction triggers a background LLM call to compress history.
func executeSlidingWindowCompaction(history []Message) {
	limit := activeConfig.Compaction.AutoCompactTurns
	keepLastN := activeConfig.Compaction.KeepLastN

	// Count actual user-submitted prompts (conversational rounds)
	userTurns := 0
	for _, msg := range history {
		if msg.Role == "user" {
			userTurns++
		}
	}

	if userTurns < limit || len(history) <= keepLastN {
		return
	}

	compactionEndIndex := len(history) - keepLastN
	boundaryTurnNumber := compactionEndIndex

	fmt.Printf("\n%s⚡ [COMPACTION] User turn limit reached (%d / %d prompts). Compacting turns 1 to %d while keeping the last %d turns untouched...%s\n", 
		ColorBold+ColorMagenta, userTurns, limit, boundaryTurnNumber, keepLastN, ColorReset)

	// 1. Read previous summary if it exists to accumulate knowledge recursively
	prevSummary, _ := getCompactedSummary()
	
	var historyToCompact strings.Builder
	if prevSummary != "" {
		historyToCompact.WriteString("=== PREVIOUS CONTEXT SUMMARY (BASELINE) ===\n")
		historyToCompact.WriteString(prevSummary)
		historyToCompact.WriteString("\n\n=== NEW CHAT LOGS TO CONSOLIDATE ===\n")
	}

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

	respMsg, err := SendMultiProviderRequest(reqBody.Messages, nil)
	if err != nil {
		fmt.Printf("%s[WARNING] Context Compaction API call failed: %v. Skipping compaction.%s\n", ColorYellow, err, ColorReset)
		return
	}

	sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
	
	// Create named compacted output file
	summaryFilename := fmt.Sprintf("compacted_summary_up_to_turn_%03d.json", boundaryTurnNumber)
	summaryPath := filepath.Join(sessionPath, summaryFilename)
	_ = os.WriteFile(summaryPath, []byte(respMsg.Content), 0644)

	// Save compaction boundary directly into meta.json (replaces compaction_boundary.txt)
	updateSessionCompactionBoundary(activeSessionID, boundaryTurnNumber)

	// Create sibling archive folder with exact same name base (creates a soft relation)
	archiveFolderName := fmt.Sprintf("compacted_summary_up_to_turn_%03d", boundaryTurnNumber)
	archivePath := filepath.Join(sessionPath, archiveFolderName)
	os.MkdirAll(archivePath, 0755)

	// Move uncompacted JSON logs of compacted turns into this folder
	entries, err := os.ReadDir(sessionPath)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
				turnIdx, err := strconv.Atoi(name[:3])
				if err == nil && turnIdx <= boundaryTurnNumber {
					oldFile := filepath.Clean(filepath.Join(sessionPath, name))
					newFile := filepath.Clean(filepath.Join(archivePath, name))
					errRename := os.Rename(oldFile, newFile)
					if errRename != nil && activeConfig.Debug {
						fmt.Printf("%s[RENAME ERROR] Failed to move turn log to archive: %v%s\n", ColorRed, errRename, ColorReset)
					}
				}
			}
		}
	}

	// Also move any older summaries and their sibling folders inside the new archive folder
	// to keep the root clean and build a recursive hierarchy!
	if entries, err = os.ReadDir(sessionPath); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if name != summaryFilename && name != "compaction_boundary.txt" && name != "meta.json" && name != "backups" && name != archiveFolderName {
				if strings.HasPrefix(name, "compacted_summary_up_to_turn_") {
					_ = os.Rename(filepath.Join(sessionPath, name), filepath.Join(archivePath, name))
				}
			}
		}
	}

	fmt.Printf("%s✅ [SUCCESS] Rolling Context Compaction completed. Archived in sibling folder and saved state up to Turn %d.%s\n", ColorGreen, boundaryTurnNumber, ColorReset)

	// Broadcast to Web Console (Phase 6.2)
	BroadcastSSE("compaction", map[string]interface{}{
		"boundary_turn": boundaryTurnNumber,
	})
}

// getCompactedSummary reads the summary from disk if it exists
func getCompactedSummary() (string, int) {
	boundary := getSessionCompactionBoundary(activeSessionID)
	if boundary == 0 {
		// Fallback: check legacy compaction_boundary.txt file
		sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
		boundaryPath := filepath.Join(sessionPath, "compaction_boundary.txt")
		if boundaryBytes, err := os.ReadFile(boundaryPath); err == nil {
			boundary, _ = strconv.Atoi(string(boundaryBytes))
		}
	}
	if boundary == 0 {
		return "", 0
	}

	sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
	summaryFilename := fmt.Sprintf("compacted_summary_up_to_turn_%03d.json", boundary)
	summaryBytes, err := os.ReadFile(filepath.Join(sessionPath, summaryFilename))
	if err != nil {
		// Fallback to legacy filename if new one doesn't exist
		summaryBytes, err = os.ReadFile(filepath.Join(sessionPath, "compacted_summary.json"))
		if err != nil {
			return "", 0
		}
	}

	return string(summaryBytes), boundary
}

func updateSessionCompactionBoundary(sessionID string, boundary int) {
	metaPath := filepath.Join(".goharness", "sessions", sessionID, "meta.json")
	bytes, err := os.ReadFile(metaPath)
	if err == nil {
		var meta SessionMeta
		if err := json.Unmarshal(bytes, &meta); err == nil {
			meta.CompactionBoundary = boundary
			newBytes, _ := json.MarshalIndent(meta, "", "  ")
			_ = os.WriteFile(metaPath, newBytes, 0644)
		}
	}
}

func getSessionCompactionBoundary(sessionID string) int {
	metaPath := filepath.Join(".goharness", "sessions", sessionID, "meta.json")
	bytes, err := os.ReadFile(metaPath)
	if err == nil {
		var meta SessionMeta
		if err := json.Unmarshal(bytes, &meta); err == nil {
			return meta.CompactionBoundary
		}
	}
	return 0
}

// LoadLocalInstructions recursively scans both the Global Binary Directory and the Active Project Workspace Folder
// for AGENTS.md, SKILLS.md, INSTRUCTIONS.md, and standard CLAUDE.md files, injecting them dynamically on Turn 1 (Phase 8.6)
func LoadLocalInstructions() string {
	var instructions []string
	targets := []string{"AGENTS.md", "SKILLS.md", "INSTRUCTIONS.md", "CLAUDE.md"}

	// Set up path map to prevent reading duplicate files if workspace matches binary root
	loadedPaths := make(map[string]bool)

	// 1. PROJECT SCOPE (Highest priority): Read instructions inside the active workspace directory
	for _, filename := range targets {
		projectPath := filepath.Join(activeConfig.Agent.WorkspaceDir, filename)
		if content, err := os.ReadFile(projectPath); err == nil {
			loadedPaths[projectPath] = true
			fmt.Printf("%s[INJECTION] Injecting PROJECT guideline from %s (%d bytes)%s\n", ColorMagenta, projectPath, len(content), ColorReset)
			header := fmt.Sprintf("\n=== PROJECT GUIDELINE: %s ===\n", filename)
			instructions = append(instructions, header+string(content))
		}
	}

	// 2. GLOBAL SCOPE (Fallback): Read instructions sitting directly next to our binary executable
	for _, filename := range targets {
		globalPath := GetSystemPath(filename)
		// Skip if it was already loaded from the workspace folder
		if loadedPaths[globalPath] {
			continue
		}
		if content, err := os.ReadFile(globalPath); err == nil {
			fmt.Printf("%s[INJECTION] Injecting GLOBAL guideline from %s (%d bytes)%s\n", ColorMagenta, globalPath, len(content), ColorReset)
			header := fmt.Sprintf("\n=== GLOBAL GUIDELINE: %s ===\n", filename)
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

// executeSubAgent spawns a recursively isolated background sub-agent to execute specialized research or historical searches
func executeSubAgent(prompt string) string {
	// Temporarily preserve parent thread context state
	parentSessionID := activeSessionID
	parentTurnNumber := currentTurnNumber

	// Create unique sub-agent session directory
	subAgentSessionID := fmt.Sprintf("%s_sub_agent_%d", parentSessionID, time.Now().UnixNano())
	subAgentPath := GetSystemPath(filepath.Join(".goharness", "sessions", subAgentSessionID))
	_ = os.MkdirAll(subAgentPath, 0755)

	wsName := filepath.Base(activeConfig.Agent.WorkspaceDir)
	createSessionMeta(subAgentSessionID, activeConfig.Agent.WorkspaceDir, parentSessionID, "Sub-Agent research: "+wsName)

	// Swap globally scoped thread context to sub-agent
	activeSessionID = subAgentSessionID
	currentTurnNumber = 0

	fmt.Printf("\n%s🤖 [SUB-AGENT] Initializing recursive execution...%s\n", ColorBold+ColorMagenta, ColorReset)
	
	// Broadcast sub-agent spawning activity to visual browser timeline
	BroadcastSSE("turn_secured", map[string]interface{}{
		"turn_number": 0,
		"role":        "system",
		"name":        "system",
		"content":     fmt.Sprintf("🤖 **[SUB-AGENT ENTRUSTED]** Spawning isolated background sub-agent to search/resolve task:\n\n*\"%s\"*", prompt),
	})

	// Execute full, multi-turn tool-capable loop recursively inside the isolated folder!
	answer := runAgentLoop(prompt)

	fmt.Printf("%s🤖 [SUB-AGENT SUCCESS] Sub-agent completed task. Restoring parent thread.%s\n", ColorBold+ColorGreen, ColorReset)

	// Restore parent thread context state
	activeSessionID = parentSessionID
	currentTurnNumber = parentTurnNumber

	// Save parent session configuration persistence
	activeConfig.Agent.LastActiveSessionID = activeSessionID
	_ = SaveConfig("config.json", activeConfig)

	// Return a beautifully structured markdown report to the parent LLM!
	return fmt.Sprintf("=== SUB-AGENT RESEARCH REPORT ===\n\n**Requested Task:** %s\n\n**Specialized Findings:**\n%s\n\n=================================", prompt, answer)
}
