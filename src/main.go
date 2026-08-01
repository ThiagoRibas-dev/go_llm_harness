package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	mcpToolsMap = make(map[string]string)

	// 1. Set default config values
	defaultCfg := Config{
		API: APIConfig{
			Provider:    "openai",
			Key:         "",
			BaseURL:     "https://api.openai.com/v1/chat/completions",
			Model:       "gpt-4o",
			Temperature: 0.0,
			MaxTokens:   4096,
		},
		Agent: AgentConfig{
			WorkspaceDir:          "./workspace",
			WorkspacesHistory:     []string{"./workspace"},
			MaxTurns:              15,
			CommandTimeoutSeconds: 30,
		},
		Security: SecurityConfig{
			SandboxMode:     "host",
			DockerContainer: "agent-workspace",
			AllowedTools:    []string{"write_file", "patch_file", "execute_command"},
			BlockedPatterns: []string{"rm -rf /", "mkfs", "dd if=", "shutdown", "format"},
		},
		DirectoryScan: DirectoryScanConfig{
			MaxDepth:             4,
			MaxFilesPerDirectory: 15,
			IgnoredPatterns:      []string{".git", ".DS_Store", "__pycache__", ".mypy_cache", ".goharness"},
			CollapsedPatterns:    []string{"node_modules", "venv", ".venv", "dist", "build", "target"},
		},
		Compaction: CompactionConfig{
			AutoCompactTurns: 6,
			KeepLastN:        2,
			Model:            "gpt-4o-mini",
			Temperature:      0.2,
			SystemPrompt:     "You are a rolling context compaction engine. Summarize the files modified, the current bugs resolved, and the active task plan in a highly dense, bulleted summary. Do not lose key context.",
		},
		MCPServers: map[string]MCPServerConfig{
			"sqlite-demo": {
				Command: "uvx",
				Args:    []string{"mcp-server-sqlite", "--db-path", "./workspace/dev.db"},
			},
		},
	}

	// 2. Try to load config.json, or create a default one if it doesn't exist
	configPath := "config.json"
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Printf("%s[SYSTEM] config.json not found or invalid (%v). Generating standard default config.json...%s\n", ColorYellow, err, ColorReset)
		err = SaveConfig(configPath, &defaultCfg)
		if err != nil {
			fmt.Printf("%s[ERROR] Failed to save default config.json: %v%s\n", ColorRed, err, ColorReset)
		}
		loadedCfg = &defaultCfg
	}
	activeConfig = loadedCfg

	// 3. Command Line Flags (Overriding config values)
	var webMode bool
	flag.BoolVar(&webMode, "web", false, "Start GoHarness in Web GUI mode on port 8080")
	flag.StringVar(&activeConfig.API.Provider, "provider", activeConfig.API.Provider, "LLM provider: 'openai', 'anthropic', 'gemini' or 'vertex'")
	flag.StringVar(&activeConfig.API.Key, "api-key", activeConfig.API.Key, "OpenAI API Key (or env OPENAI_API_KEY)")
	flag.StringVar(&activeConfig.API.BaseURL, "url", activeConfig.API.BaseURL, "API endpoint base url")
	flag.StringVar(&activeConfig.API.Model, "model", activeConfig.API.Model, "Model target name")
	flag.StringVar(&activeConfig.Security.SandboxMode, "sandbox", activeConfig.Security.SandboxMode, "Sandbox environment: 'host', 'docker' or 'none'")
	flag.StringVar(&activeConfig.Security.DockerContainer, "container", activeConfig.Security.DockerContainer, "Target Docker container if sandbox is docker")
	flag.Parse()

	// 4. Fallback Environment Variables for API Key
	if activeConfig.API.Key == "" {
		activeConfig.API.Key = os.Getenv("OPENAI_API_KEY")
	}

	// Ensure workspace exists
	os.MkdirAll(activeConfig.Agent.WorkspaceDir, 0755)

	// Populate workspace history if empty
	if len(activeConfig.Agent.WorkspacesHistory) == 0 {
		activeConfig.Agent.WorkspacesHistory = append(activeConfig.Agent.WorkspacesHistory, activeConfig.Agent.WorkspaceDir)
		_ = SaveConfig("config.json", activeConfig)
	}

	// 5. Initialize the Turn-by-Turn Session System
	activeSessionID = "sess_" + time.Now().Format("20060102-150405")
	sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
	os.MkdirAll(sessionPath, 0755)

	// Write metadata for the initial session
	createSessionMeta(activeSessionID, activeConfig.Agent.WorkspaceDir, "", "Initial Session")

	// 6. Spawn and initialize all registered Model Context Protocol (MCP) servers
	defer cleanupMCPServers()
	bootstrapMCPServers()

	// 7. If -web mode is enabled, block and start the embedded Web GUI! (Phase 6)
	if webMode {
		StartWebGUI(8080)
		return
	}

	// 8. Print beautiful startup banner (TUI Mode)
	printBanner()

	// 9. CLI Prompt Input Shell Loop (TUI Mode)
	for {
		fmt.Printf("\n%s%sEnter your prompt (or type '/fork <turn>' to rollback history, 'exit' to quit):%s\n> ", ColorBold, ColorCyan, ColorReset)
		
		reader := io.LimitReader(os.Stdin, 4096)
		buf := make([]byte, 4096)
		n, err := reader.Read(buf)
		if err != nil {
			fmt.Printf("%sError reading input: %v%s\n", ColorRed, err, ColorReset)
			return
		}
		prompt := strings.TrimSpace(string(buf[:n]))

		if prompt == "exit" || prompt == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if prompt == "" {
			continue
		}

		// Check for Rollback Fork command: e.g., `/fork 3`
		if strings.HasPrefix(prompt, "/fork ") {
			turnStr := strings.TrimSpace(strings.TrimPrefix(prompt, "/fork "))
			targetTurn, err := strconv.Atoi(turnStr)
			if err != nil {
				fmt.Printf("%s[ERROR] Invalid turn number: %s%s\n", ColorRed, turnStr, ColorReset)
				continue
			}
			executeSessionRollback(targetTurn)
			continue
		}

		// Execute the Agent Loop for this user request
		_ = runAgentLoop(prompt)
	}
}

func printBanner() {
	fmt.Printf("%s=======================================================%s\n", ColorBlue, ColorReset)
	fmt.Printf("%s   🤖 GOLANG LOCAL AGENT HARNESS (PHASE 8) 🤖       %s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("%s=======================================================%s\n", ColorBlue, ColorReset)
	fmt.Printf("  Runtime OS   : %s (%s)\n", runtimeOS(), "amd64")
	fmt.Printf("  Model Target : %s (%s, Temp: %.1f)\n", activeConfig.API.Model, activeConfig.API.Provider, activeConfig.API.Temperature)
	fmt.Printf("  Sandbox Mode : %s\n", activeConfig.Security.SandboxMode)
	fmt.Printf("  Workspace Dir: %s\n", activeConfig.Agent.WorkspaceDir)
	fmt.Printf("  Session ID   : %s\n", activeSessionID)
	fmt.Printf("  MCP Servers  : %d running\n", len(activeMCPServers))
	fmt.Printf("  API Endpoint : %s\n", activeConfig.API.BaseURL)
	fmt.Printf("%s=======================================================%s\n\n", ColorBlue, ColorReset)
}

// runAgentLoop executes the core agent loop and returns the final assistant response text (Phase 8)
func runAgentLoop(userPrompt string) string {
	agentTools := []Tool{
		{
			Type: "function",
			Function: FunctionDescriptor{
				Name:        "write_file",
				Description: "Write or overwrite a file in the workspace directory. Paths must be relative.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "The relative file path inside the workspace (e.g., script.py).",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The complete text content of the file.",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDescriptor{
				Name:        "patch_file",
				Description: "Perform a fast semantic search-and-replace edit on an existing file. Avoids rewriting the entire file.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Relative file path of the code file to patch.",
						},
						"search": map[string]interface{}{
							"type":        "string",
							"description": "The EXACT lines of original code to find. Keep spacing, tabs, and lines identical.",
						},
						"replace": map[string]interface{}{
							"type":        "string",
							"description": "The replacement lines of code to write in its place.",
						},
					},
					"required": []string{"path", "search", "replace"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDescriptor{
				Name:        "execute_command",
				Description: "Run a terminal bash/cmd command inside the workspace directory.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "The command line string to run.",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}

	mcpTools := discoverMCPTools()
	if len(mcpTools) > 0 {
		agentTools = append(agentTools, mcpTools...)
	}

	localInstructions := LoadLocalInstructions()

	workspaceTree, err := GenerateWorkspaceTree(activeConfig.Agent.WorkspaceDir, activeConfig.DirectoryScan)
	if err != nil {
		fmt.Printf("%s[WARNING] Failed to scan workspace tree: %v%s\n", ColorYellow, err, ColorReset)
	}

	systemBase := "You are a highly capable agent with access to a local terminal sandbox. Use your tools to write files, patch code, run scripts, compile binaries, and solve the user's request. When you run a script, check its output to ensure it succeeded. If it failed, fix it and try again."
	
	fullSystemPrompt := strings.Join([]string{
		systemBase,
		localInstructions,
		"\n" + workspaceTree,
		"\nIMPORTANT SAFETY RESTRICTION: You cannot write or modify files starting with '.goharness' or any system directories outside the authorized workspace.",
	}, "\n")

	userMsg := Message{Role: "user", Content: userPrompt}
	saveMessageTurn(userMsg)

	history := loadHistoryFromFiles()

	if len(history) >= activeConfig.Compaction.AutoCompactTurns {
		executeSlidingWindowCompaction(history)
		history = loadHistoryFromFiles()
	}

	compactedSummary, compactionBoundary := getCompactedSummary()

	var requestMessages []Message
	requestMessages = append(requestMessages, Message{Role: "system", Content: fullSystemPrompt})

	if compactedSummary != "" {
		summaryState := fmt.Sprintf("=== CONTEXT COMPACTION STATE SUMMARY (Turns 1 to %d) ===\nBelow is a dense summary of prior turns which have been purged to save tokens. Refer to this as the active task baseline:\n\n%s", compactionBoundary, compactedSummary)
		requestMessages = append(requestMessages, Message{Role: "system", Content: summaryState})
	}

	var filteredHistory []Message
	sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
	entries, _ := os.ReadDir(sessionPath)
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
			turnIdx, _ := strconv.Atoi(name[:3])
			if turnIdx > compactionBoundary {
				fileBytes, err := os.ReadFile(filepath.Join(sessionPath, name))
				if err == nil {
					var msg Message
					if err := json.Unmarshal(fileBytes, &msg); err == nil {
						filteredHistory = append(filteredHistory, msg)
					}
				}
			}
		}
	}

	requestMessages = append(requestMessages, filteredHistory...)

	var lastTextAnswer string
	maxTurns := activeConfig.Agent.MaxTurns
	for turn := 1; turn <= maxTurns; turn++ {
		fmt.Printf("\n%s--- TURN LOOP (%d / %d) ---%s\n", ColorBold+ColorWhite, turn, maxTurns, ColorReset)

		fmt.Printf("%s[LLM] Thinking...%s\n", ColorYellow, ColorReset)
		
		startTime := time.Now()
		responseMsg, err := SendMultiProviderRequest(requestMessages, agentTools)
		if err != nil {
			LogExecutionTrace(turn, "llm_completion", startTime, "failed", map[string]interface{}{"error": err.Error()})
			fmt.Printf("%s[ERROR] LLM API Call Failed: %v%s\n", ColorRed, err, ColorReset)
			return "Error: LLM API Call Failed."
		}
		LogExecutionTrace(turn, "llm_completion", startTime, "success", map[string]interface{}{"model": activeConfig.API.Model, "provider": activeConfig.API.Provider})

		saveMessageTurn(*responseMsg)
		requestMessages = append(requestMessages, *responseMsg)

		if responseMsg.Content != "" {
			fmt.Printf("\n%s🤖 Assistant:%s\n%s\n", ColorBold+ColorYellow, ColorReset, responseMsg.Content)
			lastTextAnswer = responseMsg.Content
		}

		if len(responseMsg.ToolCalls) == 0 {
			fmt.Printf("\n%s[SUCCESS] Task complete. No more tools requested by the LLM.%s\n", ColorBold+ColorGreen, ColorReset)
			return lastTextAnswer
		}

		for _, toolCall := range responseMsg.ToolCalls {
			fmt.Printf("\n%s🛠️ [TOOL CALL] Invoking: %s%s\n", ColorBold+ColorCyan, toolCall.Function.Name, ColorReset)

			var result string
			
			toolStart := time.Now()
			
			if mcpServerName, isMCP := mcpToolsMap[toolCall.Function.Name]; isMCP {
				fmt.Printf("  ↳ Routing tool execution to MCP Server: '%s'\n", mcpServerName)
				result = executeMCPToolCall(mcpServerName, toolCall.Function.Name, toolCall.Function.Arguments)
				LogExecutionTrace(turn, "tool_mcp_call", toolStart, "success", map[string]interface{}{
					"mcp_server": mcpServerName,
					"tool_name":  toolCall.Function.Name,
				})
			} else {
				if toolCall.Function.Name == "write_file" {
					var args struct {
						Path    string `json:"path"`
						Content string `json:"content"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						fmt.Printf("  ↳ Writing %d bytes to: %s\n", len(args.Content), args.Path)
						result = executeWriteFile(args.Path, args.Content)
						LogExecutionTrace(turn, "tool_write_file", toolStart, "success", map[string]interface{}{
							"path":          args.Path,
							"bytes_written": len(args.Content),
						})
					} else {
						result = fmt.Sprintf("Error parsing tool arguments: %v", err)
						LogExecutionTrace(turn, "tool_write_file", toolStart, "failed", map[string]interface{}{"error": err.Error()})
					}
				} else if toolCall.Function.Name == "patch_file" {
					var args struct {
						Path    string `json:"path"`
						Search  string `json:"search"`
						Replace string `json:"replace"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						fmt.Printf("  ↳ Patching file: %s\n", args.Path)
						result = executePatchFile(args.Path, args.Search, args.Replace)
						LogExecutionTrace(turn, "tool_patch_file", toolStart, "success", map[string]interface{}{
							"path": args.Path,
						})
					} else {
						result = fmt.Sprintf("Error parsing tool arguments: %v", err)
						LogExecutionTrace(turn, "tool_patch_file", toolStart, "failed", map[string]interface{}{"error": err.Error()})
					}
				} else if toolCall.Function.Name == "execute_command" {
					var args struct {
						Command string `json:"command"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						fmt.Printf("  ↳ Running command: %s\n", args.Command)
						result = executeTerminalCommand(args.Command)
						LogExecutionTrace(turn, "tool_execute_command", toolStart, "success", map[string]interface{}{
							"command": args.Command,
						})
					} else {
						result = fmt.Sprintf("Error parsing tool arguments: %v", err)
						LogExecutionTrace(turn, "tool_execute_command", toolStart, "failed", map[string]interface{}{"error": err.Error()})
					}
				} else {
					result = fmt.Sprintf("Unknown tool name: %s", toolCall.Function.Name)
					LogExecutionTrace(turn, "tool_unknown", toolStart, "failed", map[string]interface{}{"tool": toolCall.Function.Name})
				}
			}

			snippet := result
			if len(snippet) > 400 {
				snippet = snippet[:400] + "\n... [TRUNCATED] ..."
			}
			fmt.Printf("%s[RESULT]%s\n%s\n", ColorGreen, ColorReset, snippet)

			toolResponseMsg := Message{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
				Content:    result,
			}
			saveMessageTurn(toolResponseMsg)
			requestMessages = append(requestMessages, toolResponseMsg)
		}
	}
	return lastTextAnswer
}

func executeWriteFile(path, content string) string {
	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, ".goharness") || strings.Contains(cleanPath, "config.json") || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		fmt.Printf("%s[SECURITY SHIELD] Blocked attempt to write/overwrite systemic path: %s%s\n", ColorRed, path, ColorReset)
		return fmt.Sprintf("Security Exception: Systemic or out-of-workspace directories are write-protected. Access denied to path: %s", path)
	}

	backupWorkspaceFile(cleanPath)

	if activeConfig.Security.SandboxMode == "docker" {
		cmd := exec.Command("docker", "exec", "-i", activeConfig.Security.DockerContainer, "tee", path)
		cmd.Stdin = strings.NewReader(content)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Sprintf("Docker write failed: %v. Stderr: %s", err, stderr.String())
		}
		return fmt.Sprintf("Successfully wrote file inside Docker at %s", path)
	}

	fullPath := filepath.Join(activeConfig.Agent.WorkspaceDir, cleanPath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	err := os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("Failed to write file to host disk: %v", err)
	}
	return fmt.Sprintf("Successfully wrote file to host disk at %s", cleanPath)
}

func executePatchFile(path, search, replace string) string {
	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, ".goharness") || strings.Contains(cleanPath, "config.json") || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return fmt.Sprintf("Security Exception: Systemic or out-of-workspace directories are write-protected. Access denied to path: %s", path)
	}

	fullPath := filepath.Join(activeConfig.Agent.WorkspaceDir, cleanPath)

	originalBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Patch Error: File not found or could not be read: %s. Use write_file to create new files.", cleanPath)
	}
	originalContent := string(originalBytes)

	backupWorkspaceFile(cleanPath)

	if !strings.Contains(originalContent, search) {
		fmt.Printf("%s[PATCH ERROR] Block to search not found in file: %s%s\n", ColorRed, cleanPath, ColorReset)
		return fmt.Sprintf("Patch Error: The search block was not found in the file '%s'. Ensure your search string matches exactly, including spaces and indentations. File remains unmodified.", cleanPath)
	}

	patchedContent := strings.Replace(originalContent, search, replace, 1)

	err = os.WriteFile(fullPath, []byte(patchedContent), 0644)
	if err != nil {
		return fmt.Sprintf("Patch Error: Failed to write patched content to host disk: %v", err)
	}

	diffLines := len(strings.Split(replace, "\n")) - len(strings.Split(search, "\n"))
	fmt.Printf("%s[PATCH SUCCESS] Patched %s successfully (+%d lines changed)%s\n", ColorGreen, cleanPath, diffLines, ColorReset)
	return fmt.Sprintf("Successfully patched file '%s'. Changes applied seamlessly.", cleanPath)
}

func executeTerminalCommand(command string) string {
	for _, pattern := range activeConfig.Security.BlockedPatterns {
		if strings.Contains(command, pattern) {
			fmt.Printf("%s[SECURITY GUARDRAIL] Blocked execution of command containing pattern: '%s'%s\n", ColorRed, pattern, ColorReset)
			return fmt.Sprintf("Security Exception: Command execution blocked. The command contains blacklisted structural pattern: '%s'", pattern)
		}
	}

	result, err := RunCommandInSandbox(command, activeConfig.Agent.WorkspaceDir)
	if err != nil {
		return fmt.Sprintf("Sandbox Execution Error: %v", err)
	}
	return result
}
