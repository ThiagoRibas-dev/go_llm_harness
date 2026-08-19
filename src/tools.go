package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// builtInToolSchemas returns the schema definitions for the native tools that
// can be exposed to an LLM. These are shared by the legacy linear ReAct loop
// and by DAG llm nodes that have tool-calling enabled.
func builtInToolSchemas() []Tool {
	return []Tool{
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
				Description: "Spawn an isolated sub-agent to perform one focused task (research, code analysis, search) and return a dense summary. Call this tool multiple times in ONE response to run several sub-agents in parallel; their results come back together.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"prompt": map[string]interface{}{
							"type":        "string",
							"description": "The precise task or instruction for the sub-agent.",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Short label for this sub-agent shown while it runs (e.g. 'research auth library').",
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
				Description: "Index and search files in the workspace or session chat archives using BM25 lexical ranking. Recommended over raw terminal searches for code/facts/context retrieval.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The keywords or query terms to search for.",
						},
						"scope": map[string]interface{}{
							"type":        "string",
							"description": "Scope: 'workspace' (active repository files) or 'session' (previous chat/tool logs).",
							"enum":        []string{"workspace", "session"},
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of scored results (default 5).",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}
}

// selectTools returns the tool set exposed to an LLM: the named built-in tools
// (or all built-ins when allowed is empty), plus MCP-discovered tools when
// includeMCP is true.
func selectTools(allowed []string, includeMCP bool) []Tool {
	builtins := builtInToolSchemas()

	if len(allowed) == 0 {
		// Empty means "all built-ins".
		allowed = make([]string, 0, len(builtins))
		for _, t := range builtins {
			allowed = append(allowed, t.Function.Name)
		}
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, n := range allowed {
		allowedSet[n] = true
	}

	tools := make([]Tool, 0, len(allowed)+4)
	for _, t := range builtins {
		if allowedSet[t.Function.Name] {
			tools = append(tools, t)
		}
	}

	if includeMCP {
		if mcp := discoverMCPTools(); len(mcp) > 0 {
			tools = append(tools, mcp...)
		}
	}
	return tools
}

// executeToolCall runs a single tool invoked by an LLM and returns its text
// result. It is shared by the linear ReAct loop and tool-enabled DAG nodes.
// turn is used solely for execution-trace logging.
func executeToolCall(a *Agent, turn int, tc ToolCall) string {
	toolStart := time.Now()
	name := tc.Function.Name

	// MCP-discovered tools take priority when the name is registered to a server.
	if mcpServerName, isMCP := mcpToolsMap[name]; isMCP {
		writeDebugLog("[TOOL %s] Routing to MCP server '%s'", name, mcpServerName)
		result := executeMCPToolCall(mcpServerName, name, tc.Function.Arguments)
		LogExecutionTrace(turn, "tool_mcp_call", toolStart, "success", map[string]interface{}{
			"mcp_server": mcpServerName,
			"tool_name":  name,
			"session":    a.SessionID,
		})
		return result
	}

	switch name {
	case "read_file":
		var args struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line"`
			EndLine   int    `json:"end_line"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
			if args.StartLine <= 0 {
				args.StartLine = 1
			}
			result := executeReadFile(args.Path, args.StartLine, args.EndLine)
			LogExecutionTrace(turn, "tool_read_file", toolStart, "success", map[string]interface{}{
				"path":       args.Path,
				"start_line": args.StartLine,
				"end_line":   args.EndLine,
			})
			return result
		}
		return parseToolErr(turn, name, toolStart, tc.Function.Arguments)

	case "write_file":
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
			LogExecutionTrace(turn, "tool_write_file", toolStart, "success", map[string]interface{}{
				"path":          args.Path,
				"bytes_written": len(args.Content),
			})
			return executeWriteFile(a, args.Path, args.Content)
		}
		return parseToolErr(turn, name, toolStart, tc.Function.Arguments)

	case "patch_file":
		var args struct {
			Path    string `json:"path"`
			Search  string `json:"search"`
			Replace string `json:"replace"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
			LogExecutionTrace(turn, "tool_patch_file", toolStart, "success", map[string]interface{}{"path": args.Path})
			return executePatchFile(a, args.Path, args.Search, args.Replace)
		}
		return parseToolErr(turn, name, toolStart, tc.Function.Arguments)

	case "execute_command":
		var args struct{ Command string }
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
			LogExecutionTrace(turn, "tool_execute_command", toolStart, "success", map[string]interface{}{"command": args.Command})
			return executeTerminalCommand(args.Command)
		}
		return parseToolErr(turn, name, toolStart, tc.Function.Arguments)

	case "bm25_search":
		var args struct {
			Query string `json:"query"`
			Scope string `json:"scope"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
			if args.Limit <= 0 {
				args.Limit = 5
			}
			if args.Scope == "" {
				args.Scope = "workspace"
			}
			LogExecutionTrace(turn, "tool_bm25_search", toolStart, "success", map[string]interface{}{
				"query": args.Query,
				"scope": args.Scope,
			})
			return executeBM25Search(a, args.Query, args.Scope, args.Limit)
		}
		return parseToolErr(turn, name, toolStart, tc.Function.Arguments)

	default:
		LogExecutionTrace(turn, "tool_unknown", toolStart, "failed", map[string]interface{}{"tool": name})
		return fmt.Sprintf("Unknown tool name: %s", name)
	}
}

func parseToolErr(turn int, name string, start time.Time, args string) string {
	err := fmt.Errorf("failed to parse arguments")
	LogExecutionTrace(turn, "tool_"+name, start, "failed", map[string]interface{}{"error": err.Error()})
	_ = args
	return fmt.Sprintf("Error parsing tool arguments: %v", err)
}

// truncateResult clips a tool result for console display (the full text is
// still returned to the LLM).
func truncateResult(result string) string {
	if len(result) > 400 {
		return result[:400] + "\n... [TRUNCATED] ..."
	}
	return result
}

var _ = strings.TrimSpace
