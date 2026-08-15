package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
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
			LastActiveSessionID:   "",
			MaxTurns:              15,
			CommandTimeoutSeconds: 30,
			TargetScanDirs:        []string{},
		},
		Security: SecurityConfig{
			SandboxMode:     "host",
			DockerContainer: "agent-workspace",
			SandboxFallback: false,
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
			Provider:         "openai",
			Key:              "",
			BaseURL:          "https://api.openai.com/v1/chat/completions",
			Model:            "gpt-4o-mini",
			Temperature:      0.2,
			ProjectID:        "",
			Region:           "",
			AutoCompactTurns: 6,
			KeepLastN:        2,
			SystemPrompt:     "You are a professional context compaction, research synthesis, and developer handoff engine. Your task is to generate a highly structured, dense, and complete summary of the execution conversation so far. This summary will be injected into a future session as the sole active baseline context, so you MUST preserve critical technical details, architectural decisions, core research data, and constraints while dropping conversational noise.\n\n" +
				"When compacting, you MUST structure your output into these 9 aspects:\n" +
				"1. 📊 CURRENT STATE: High-level active task status, project progress, or current document draft baseline.\n" +
				"2. 🎯 GOALS & INTENT: What the user explicitly requested, the overarching strategic objective, target audience, or desired output tone.\n" +
				"3. 🛠️ RECENT CHANGES: Exact code modifications, files changed, newly configured tools, or primary data points newly extracted.\n" +
				"4. 💡 KEY DECISIONS: Technical/architectural choices, analytical hypotheses selected, and key decisions (and why they were made).\n" +
				"5. 🏗️ ACTIVE WORK: What is currently in progress, partially written drafts, active scripts running, or ongoing research pathways.\n" +
				"6. 📂 KEY FILES & SOURCES: Crucial file paths, routes, configurations, research URLs, citations, or primary documents involved.\n" +
				"7. 🎓 LEARNINGS & FINDINGS: Discovered bug resolutions, platform-specific blocks (like Windows API locks), crucial numbers/facts/statistics found, and analytical findings.\n" +
				"8. 🔒 IMPORTANT CONTEXT & CONSTRAINTS: Active user preferences, structural requirements, legal/operational constraints, or explicit instructions (e.g. zero-dependency rules, security configurations, document limits).\n" +
				"9. 📋 OPTIONAL NEXT STEPS: Clear, chronological, and actionable next steps or next-stage research questions for continuation.\n\n" +
				"PRESERVE VERBATIM: Specific error messages and their corresponding fixes, raw credentials/tokens if any, core data parameters, facts and figures, and explicit user rules.\n" +
				"MAY CONDENSE/DROP: Raw stdout/stderr terminal outputs (summarize the conclusion), raw file contents read or web page HTML scraped (summarize what was learned and extract key data), and dead exploratory steps.",
		},
		MCPServers: map[string]MCPServerConfig{
			"sqlite-demo": {
				Command: "uvx",
				Args:    []string{"mcp-server-sqlite", "--db-path", "./workspace/dev.db"},
			},
		},
		Web: WebConfig{
			Enabled:           false,
			Port:              8080,
			APIGatewayEnabled: true,
		},
		Debug: false,
	}

	// 2. Try to load config.json (Resolve relative to binary path - Phase 8.6)
	configPath := GetSystemPath("config.json")
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

	// 2b. Seed workflows.json with the built-in defaults if it is missing.
	//     This is non-destructive: an existing workflows.json is never overwritten.
	if err := EnsureDefaultWorkflows(); err != nil {
		fmt.Printf("%s[WARNING] Failed to seed default workflows.json: %v%s\n", ColorYellow, err, ColorReset)
	}

	// 2c. Seed providers.json (reusable connection profiles) from config.json on
	//     first run, then apply the active chat profile (if one is set). The
	//     profile is used directly; the inline api block only supplies
	//     max_tokens/key fallbacks.
	if err := EnsureProvidersFile(); err != nil {
		fmt.Printf("%s[WARNING] Failed to seed providers.json: %v%s\n", ColorYellow, err, ColorReset)
	}
	if activeConfig.ProviderProfile != "" {
		if resolved, err := GetProvider(activeConfig.ProviderProfile); err == nil {
			activeConfig.API = resolved
			if activeConfig.API.MaxTokens == 0 {
				activeConfig.API.MaxTokens = defaultCfg.API.MaxTokens
			}
			fmt.Printf("%s[SYSTEM] Active chat connection using provider profile: %s (%s/%s)%s\n",
				ColorGreen, activeConfig.ProviderProfile, resolved.Provider, resolved.Model, ColorReset)
		} else {
			fmt.Printf("%s[WARNING] %v%s\n", ColorYellow, err, ColorReset)
		}
	}

	// 3. Command Line Flags (Overriding config values)
	var webMode bool
	flag.BoolVar(&webMode, "web", activeConfig.Web.Enabled, "Start GoHarness in Web GUI & API Gateway mode")
	flag.IntVar(&activeConfig.Web.Port, "port", activeConfig.Web.Port, "Port to run the Web GUI & API Gateway on")
	flag.StringVar(&activeConfig.API.Provider, "provider", activeConfig.API.Provider, "LLM provider: 'openai', 'anthropic', 'gemini' or 'vertex'")
	flag.StringVar(&activeConfig.API.Key, "api-key", activeConfig.API.Key, "OpenAI API Key (or env OPENAI_API_KEY)")
	flag.StringVar(&activeConfig.API.BaseURL, "url", activeConfig.API.BaseURL, "API endpoint base url")
	flag.StringVar(&activeConfig.API.Model, "model", activeConfig.API.Model, "Model target name")
	flag.StringVar(&activeConfig.Security.SandboxMode, "sandbox", activeConfig.Security.SandboxMode, "Sandbox environment: 'host', 'docker' or 'none'")
	flag.StringVar(&activeConfig.Security.DockerContainer, "container", activeConfig.Security.DockerContainer, "Target Docker container if sandbox is docker")
	flag.BoolVar(&activeConfig.Debug, "debug", activeConfig.Debug, "Enable verbose developer diagnostic logs") // Phase 8.6
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
		_ = SaveConfig(configPath, activeConfig)
	}

	// 5. Initialize or Resume the Turn-by-Turn Session System (Phase 8.6 session persistence)
	sessionID := activeConfig.Agent.LastActiveSessionID
	sessionPath := ""
	if sessionID != "" {
		sessionPath = GetSystemPath(filepath.Join(".goharness", "sessions", sessionID))
	}

	// Verify that the saved last active session directory physically exists on disk
	if sessionID != "" && dirExists(sessionPath) {
		activeSessionID = sessionID
		fmt.Printf("%s[SYSTEM] Resuming last active conversation session: %s%s\n", ColorGreen, activeSessionID, ColorReset)
		_ = loadHistoryFromFiles() // Warm up memory cache
		currentTurnNumber = findMaxTurnNumber(activeSessionID)
	} else {
		// Fallback: Create a brand new session on first boot or if deleted
		activeSessionID = "sess_" + time.Now().Format("20060102-150405")
		sessionPath = GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
		os.MkdirAll(sessionPath, 0755)

		// Write metadata for the initial session
		createSessionMeta(activeSessionID, activeConfig.Agent.WorkspaceDir, "", "Initial Session")
		
		activeConfig.Agent.LastActiveSessionID = activeSessionID
		_ = SaveConfig(configPath, activeConfig)
	}

	// 6. Spawn and initialize all registered Model Context Protocol (MCP) servers
	defer cleanupMCPServers()
	bootstrapMCPServers()

	// 7. If -web mode is enabled, block and start the embedded Web GUI! (Phase 6)
	if webMode {
		StartWebGUI(activeConfig.Web.Port)
		return
	}

	// 8. Print beautiful startup banner (TUI Mode)
	printBanner()

	// 9. CLI Prompt Input Shell Loop (TUI Mode)
	scanner := bufio.NewScanner(os.Stdin)
	// Allow prompts up to 4096 bytes (preserves prior input ceiling).
	scanner.Buffer(make([]byte, 4096), 4096)
	for {
		fmt.Printf("\n%s%sEnter your prompt (or type '/fork <turn>' to rollback history, '/workflows' to list pipelines, 'exit' to quit):%s\n> ", ColorBold, ColorCyan, ColorReset)

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Printf("%sError reading input: %v%s\n", ColorRed, err, ColorReset)
			} else {
				fmt.Println("\nGoodbye!")
			}
			return
		}
		prompt := strings.TrimSpace(scanner.Text())

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

		// /workflows: List all registered pipelines and highlight the active one
		if prompt == "/workflows" {
			activeID, wfs, err := ListWorkflows()
			if err != nil {
				fmt.Printf("%s[ERROR] Failed to load workflows: %v%s\n", ColorRed, err, ColorReset)
				continue
			}
			fmt.Printf("\n%s%s=== Registered Workflows ===%s\n", ColorBold, ColorCyan, ColorReset)
			for id, wf := range wfs {
				marker := "  "
				if id == activeID {
					marker = ColorGreen + "▶ " + ColorReset
				}
				fmt.Printf("  %s%s%s%s %s- %s\n", marker, ColorBold, id, ColorReset, wf.Name, wf.Description)
			}
			fmt.Printf("\n  Switch with: %s/workflow <id>%s (e.g. /workflow enhanced_cognition)\n", ColorYellow, ColorReset)
			continue
		}

		// /workflow <id>: Hot-swap the active conversation pipeline
		if strings.HasPrefix(prompt, "/workflow") {
			targetID := strings.TrimSpace(strings.TrimPrefix(prompt, "/workflow"))
			if targetID == "" {
				fmt.Printf("%sUsage: /workflow <id>   (try /workflows to list available pipelines)%s\n", ColorYellow, ColorReset)
				continue
			}
			if err := ActivateWorkflow(targetID); err != nil {
				fmt.Printf("%s[ERROR] %v%s\n", ColorRed, err, ColorReset)
				continue
			}
			fmt.Printf("%s✅ [WORKFLOW ACTIVATED] Active pipeline switched to '%s'.%s\n", ColorGreen, targetID, ColorReset)
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

func runAgentLoop(userPrompt string) string {
	agentTools := []Tool{
		{
			Type: "function",
			Function: FunctionDescriptor{
				Name:        "read_file",
				Description: "Read the contents of a file inside the workspace. For large files, you can specify specific start and end line ranges to prevent token blowouts.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Relative file path of the file to read (e.g. src/app.js).",
						},
						"start_line": map[string]interface{}{
							"type":        "integer",
							"description": "Optional 1-based line number to start reading from (default: 1).",
						},
						"end_line": map[string]interface{}{
							"type":        "integer",
							"description": "Optional 1-based line number to stop reading at (inclusive, default: end of file).",
						},
					},
					"required": []string{"path"},
				},
			},
		},
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
		{
			Type: "function",
			Function: FunctionDescriptor{
				Name:        "spawn_sub_agent",
				Description: "Spawn a specialized background sub-agent to perform complex file searches, code analysis, or parallel research in the workspace or session archives, returning a dense final summary report.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"prompt": map[string]interface{}{
							"type":        "string",
							"description": "The precise task, file-search query, or instruction for the sub-agent (e.g., 'Find the postgres credentials in the archived turns in compacted_summary_up_to_turn_046/').",
						},
					},
					"required": []string{"prompt"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDescriptor{
				Name:        "bm25_search",
				Description: "Index and search files inside the workspace or session chat archives using standard BM25 lexical ranking. Highly recommended over raw terminal searches for code, facts, or context retrieval in large directories.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The keywords or query terms to search for (e.g. 'postgres database credentials').",
						},
						"scope": map[string]interface{}{
							"type":        "string",
							"description": "Scope of search: 'workspace' (to search active repository files) or 'session' (to search previous chat/tool logs).",
							"enum":        []string{"workspace", "session"},
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of scored results to return (default 5).",
						},
					},
					"required": []string{"query"},
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

	// 1. Try to route execution through GoHarness v2.0 Dynamic Node Workflow Engine!
	workflowsPath := GetSystemPath("workflows.json")
	if _, errStat := os.Stat(workflowsPath); errStat == nil {
		cfg, err := LoadWorkflowConfig()
		if err == nil && cfg.ActiveWorkflow != "" && cfg.ActiveWorkflow != "linear_chat" {
			answer, errExec := ExecuteActiveWorkflow(userPrompt)
			if errExec == nil {
				return answer
			}
			fmt.Printf("%s[WORKFLOW EXECUTION ERROR] %v. Falling back to native tools loop.%s\n", ColorRed, errExec, ColorReset)
		}
	}

	history := loadHistoryFromFiles()

	// Count actual user prompts to decide if we should trigger compaction
	userTurns := 0
	for _, msg := range history {
		if msg.Role == "user" {
			userTurns++
		}
	}

	if userTurns >= activeConfig.Compaction.AutoCompactTurns {
		executeSlidingWindowCompaction(history, false)
		history = loadHistoryFromFiles()
	}

	compactedSummary, compactionBoundary := getCompactedSummary()

	var requestMessages []Message
	requestMessages = append(requestMessages, Message{Role: "system", Content: fullSystemPrompt})

	if compactedSummary != "" {
		summaryState := fmt.Sprintf("=== CONTEXT COMPACTION STATE SUMMARY (Turns 1 to %d) ===\n"+
			"Below is a dense summary of prior turns which have been archived to save tokens. Refer to this as the active task baseline:\n\n%s\n\n"+
			"💡 RETRIEVAL INSTRUCTION: If you need to inspect the exact raw messages, tool calls, or code from these archived turns "+
			"(e.g., to find specific technical details referenced in the summary), you have full access to their raw JSON files. "+
			"They are stored in your active session subfolder under: '.goharness/sessions/%s/compacted_summary_up_to_turn_%03d/'. "+
			"Use the 'execute_command' tool to read/find them if necessary.",
			compactionBoundary, compactedSummary, activeSessionID, compactionBoundary)
		requestMessages = append(requestMessages, Message{Role: "system", Content: summaryState})
	}

	var filteredHistory []Message
	sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
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
		writeDebugLog("Entering Turn %d of %d (Active Session: %s)", turn, maxTurns, activeSessionID)

		fmt.Printf("%s[LLM] Thinking...%s\n", ColorYellow, ColorReset)
		writeDebugLog("Requesting completion from provider: %s, model: %s", activeConfig.API.Provider, activeConfig.API.Model)
		
		startTime := time.Now()
		responseMsg, err := SendMultiProviderRequest(requestMessages, agentTools)
		if err != nil {
			LogExecutionTrace(turn, "llm_completion", startTime, "failed", map[string]interface{}{"error": err.Error()})
			fmt.Printf("%s[ERROR] LLM API Call Failed: %v%s\n", ColorRed, err, ColorReset)
			writeDebugLog("LLM API Call failed on Turn %d: %v", turn, err)
			
			// Broadcast the API error directly to the Web Console!
			BroadcastSSE("turn_secured", map[string]interface{}{
				"turn_number": 0,
				"role":        "system",
				"name":        "system",
				"content":     fmt.Sprintf("❌ [LLM API ERROR] Request failed: %v. Please verify your API Key and Provider Base URL in the Settings modal.", err),
			})
			return "Error: LLM API Call Failed."
		}
		LogExecutionTrace(turn, "llm_completion", startTime, "success", map[string]interface{}{"model": activeConfig.API.Model, "provider": activeConfig.API.Provider})
		writeDebugLog("LLM response received. Content length: %d bytes. Tool calls: %d", len(responseMsg.Content), len(responseMsg.ToolCalls))

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
				if toolCall.Function.Name == "read_file" {
					var args struct {
						Path      string `json:"path"`
						StartLine int    `json:"start_line"`
						EndLine   int    `json:"end_line"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						if args.StartLine <= 0 {
							args.StartLine = 1
						}
						fmt.Printf("  ↳ Reading file: %s (Lines: %d to %d)\n", args.Path, args.StartLine, args.EndLine)
						writeDebugLog("[TOOL] read_file requested for path: %s (start: %d, end: %d)", args.Path, args.StartLine, args.EndLine)
						result = executeReadFile(args.Path, args.StartLine, args.EndLine)
						LogExecutionTrace(turn, "tool_read_file", toolStart, "success", map[string]interface{}{
							"path":       args.Path,
							"start_line": args.StartLine,
							"end_line":   args.EndLine,
						})
					} else {
						result = fmt.Sprintf("Error parsing tool arguments: %v", err)
						LogExecutionTrace(turn, "tool_read_file", toolStart, "failed", map[string]interface{}{"error": err.Error()})
					}
				} else if toolCall.Function.Name == "write_file" {
					var args struct {
						Path    string `json:"path"`
						Content string `json:"content"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						fmt.Printf("  ↳ Writing %d bytes to: %s\n", len(args.Content), args.Path)
						writeDebugLog("[TOOL] write_file requested for path: %s (%d bytes)", args.Path, len(args.Content))
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
				} else if toolCall.Function.Name == "spawn_sub_agent" {
					var args struct {
						Prompt string `json:"prompt"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						fmt.Printf("  ↳ Spawning specialized sub-agent for: %s\n", args.Prompt)
						result = executeSubAgent(args.Prompt)
						LogExecutionTrace(turn, "tool_spawn_sub_agent", toolStart, "success", map[string]interface{}{
							"prompt": args.Prompt,
						})
					} else {
						result = fmt.Sprintf("Error parsing tool arguments: %v", err)
						LogExecutionTrace(turn, "tool_spawn_sub_agent", toolStart, "failed", map[string]interface{}{"error": err.Error()})
					}
				} else if toolCall.Function.Name == "bm25_search" {
					var args struct {
						Query string `json:"query"`
						Scope string `json:"scope"`
						Limit int    `json:"limit"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						if args.Limit <= 0 {
							args.Limit = 5
						}
						if args.Scope == "" {
							args.Scope = "workspace"
						}
						fmt.Printf("  ↳ Executing BM25 Search (Query: '%s', Scope: %s)\n", args.Query, args.Scope)
						result = executeBM25Search(args.Query, args.Scope, args.Limit)
						LogExecutionTrace(turn, "tool_bm25_search", toolStart, "success", map[string]interface{}{
							"query": args.Query,
							"scope": args.Scope,
						})
					} else {
						result = fmt.Sprintf("Error parsing tool arguments: %v", err)
						LogExecutionTrace(turn, "tool_bm25_search", toolStart, "failed", map[string]interface{}{"error": err.Error()})
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

// backupWorkspaceFile stashes a file before we overwrite or patch it.
// If the file doesn't exist, we write an empty ".untracked_new" marker so that the rollback engine knows to delete it! (Phase 8.5)
func backupWorkspaceFile(relativePath string) {
	srcPath := filepath.Join(activeConfig.Agent.WorkspaceDir, relativePath)

	content, err := os.ReadFile(srcPath)
	if err != nil {
		// File does not exist yet. Create a ".untracked_new" marker file (Phase 8.5)
		markerDir := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID, "backups", fmt.Sprintf("turn-%d", currentTurnNumber+1)))
		os.MkdirAll(markerDir, 0755)
		markerPath := filepath.Join(markerDir, relativePath+".untracked_new")
		os.MkdirAll(filepath.Dir(markerPath), 0755)
		_ = os.WriteFile(markerPath, []byte(""), 0644)
		return
	}

	backupDir := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID, "backups", fmt.Sprintf("turn-%d", currentTurnNumber+1)))
	os.MkdirAll(backupDir, 0755)

	destPath := filepath.Join(backupDir, relativePath)
	os.MkdirAll(filepath.Dir(destPath), 0755)

	_ = os.WriteFile(destPath, content, 0644)
}

// restoreWorkspaceBackups restores modified files and physically deletes newly created files on rollbacks (Phase 8.5)
func restoreWorkspaceBackups(targetTurn int) {
	backupRoot := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID, "backups"))

	// 1. Restore original contents of modified files
	targetBackupFolder := filepath.Join(backupRoot, fmt.Sprintf("turn-%d", targetTurn+1))
	if _, err := os.Stat(targetBackupFolder); err == nil {
		fmt.Printf("%s  ↳ Restoring original file contents from backup: turn-%d...%s\n", ColorMagenta, targetTurn+1, ColorReset)
		filepath.Walk(targetBackupFolder, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || strings.HasSuffix(info.Name(), ".untracked_new") {
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

	// 2. Physically delete newly created files that did not exist at the target turn (Phase 8.5)
	entries, err := os.ReadDir(backupRoot)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "turn-") {
				turnNumStr := strings.TrimPrefix(entry.Name(), "turn-")
				turnNum, err := strconv.Atoi(turnNumStr)
				// If this backup was created in a turn higher than our rollback target (e.g. turn 4, 5, etc.)
				if err == nil && turnNum > targetTurn {
					turnFolder := filepath.Join(backupRoot, entry.Name())
					filepath.Walk(turnFolder, func(path string, info os.FileInfo, err error) error {
						if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".untracked_new") {
							rel, _ := filepath.Rel(turnFolder, path)
							relClean := strings.TrimSuffix(rel, ".untracked_new")
							destPath := filepath.Join(activeConfig.Agent.WorkspaceDir, relClean)
							
							// Physically delete the untracked file to completely restore state!
							_ = os.Remove(destPath)
							fmt.Printf("    - Deleted newly created file: %s\n", relClean)
						}
						return nil
					})
				}
			}
		}
	}

	// 3. Clean up backup folders higher than the targetTurn
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

// Helper: Checks if a physical directory exists on disk (Phase 8.6)
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return true
	}
	return false
}

// executeReadFile reads a specified line-range from a file in the workspace, enforcing safety truncation rules
func executeReadFile(path string, startLine, endLine int) string {
	cleanPath := filepath.Clean(path)

	// Security Shield: protect system files
	if strings.Contains(cleanPath, ".goharness") || strings.Contains(cleanPath, "config.json") || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return fmt.Sprintf("Security Exception: Systemic or out-of-workspace directories are read-protected. Access denied to path: %s", path)
	}

	fullPath := filepath.Join(activeConfig.Agent.WorkspaceDir, cleanPath)
	fileBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: File not found or could not be read at path '%s'. Ensure the file path is correct and relative.", cleanPath)
	}

	lines := strings.Split(string(fileBytes), "\n")
	totalLines := len(lines)

	if startLine > totalLines {
		return fmt.Sprintf("Error: Requested start_line (%d) exceeds total lines in file (%d).", startLine, totalLines)
	}

	// Calculate end line
	actualEndLine := totalLines
	if endLine > 0 && endLine < totalLines {
		actualEndLine = endLine
	}

	if actualEndLine < startLine {
		return fmt.Sprintf("Error: Requested end_line (%d) is less than start_line (%d).", actualEndLine, startLine)
	}

	// Smart Size Truncation Safeguard:
	// If the requested range (or entire file) is too large (e.g. over 200 lines) and the user/agent didn't specify end_line,
	// we automatically truncate it to protect the LLM's context window!
	isTruncated := false
	maxLinesToReturn := 200
	if endLine <= 0 && (actualEndLine-startLine+1) > maxLinesToReturn {
		actualEndLine = startLine + maxLinesToReturn - 1
		isTruncated = true
	}

	// Slice the lines
	var sb strings.Builder
	for idx := startLine - 1; idx < actualEndLine; idx++ {
		// Line numbering is 1-based
		sb.WriteString(fmt.Sprintf("%d | %s\n", idx+1, lines[idx]))
	}

	if isTruncated {
		sb.WriteString(fmt.Sprintf("\n⚠️ [LINE-RANGE WARNING] File is too large (%d total lines). Output has been truncated to lines %d-%d to prevent token window blowouts. If you need to read subsequent sections, please invoke 'read_file' again with specific 'start_line' and 'end_line' parameters.", totalLines, startLine, actualEndLine))
	}

	return sb.String()
}
