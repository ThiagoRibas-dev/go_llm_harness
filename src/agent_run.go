package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Run executes the agent's ReAct loop: build the system prompt, then
// alternate between LLM completion and tool execution until the model
// returns no tool calls (answer) or max turns is reached. It is safe to
// call concurrently for different Agents (each has its own session, turn
// counter, and connection).
func (a *Agent) Run(ctx context.Context, userPrompt string) string {
	// Sub-agents don't route through DAG workflows; the root agent does.
	if a.Depth == 0 {
		if answer, ok := a.tryWorkflowRoute(userPrompt); ok {
			return answer
		}
	}

	// Tool set for this agent. Sub-agents get a reduced set (no writes,
	// no deeper spawning beyond the recursion limit).
	tools := a.agentTools()

	instructions := LoadLocalInstructions()
	workspaceTree, err := GenerateWorkspaceTree(a.Workspace, activeConfig.DirectoryScan)
	if err != nil {
		fmt.Printf("%s[WARNING] Failed to scan workspace tree: %v%s\n", ColorYellow, err, ColorReset)
	}

	systemBase := "You are a highly capable agent with access to a local terminal sandbox. Use your tools to read files, search, run commands, and solve the user's request. Check command output; if something failed, fix it and try again."
	if a.Depth > 0 {
		systemBase = "You are a focused sub-agent spawned by the parent agent to complete one specific task. Be concise and return a dense summary of your findings/actions. Do not ask the parent clarifying questions; use your best judgment."
	}

	fullSystemPrompt := strings.Join([]string{
		systemBase,
		buildEnvironmentSystemPrompt(),
		instructions,
		"\n" + workspaceTree,
		"\nIMPORTANT SAFETY RESTRICTION: You cannot write or modify files starting with '.goharness' or any system directories outside the authorized workspace.",
	}, "\n")

	a.saveTurn(Message{Role: "user", Content: userPrompt})

	// Sub-agents skip auto-compaction (it shuffles session files and isn't
	// worth the risk for short research tasks). The root agent compacts as
	// configured.
	history := a.loopHistory()
	if a.Depth == 0 && activeConfig.Compaction.AutoCompactTurns > 0 {
		userTurns := 0
		for _, m := range history {
			if m.Role == "user" {
				userTurns++
			}
		}
		if userTurns >= activeConfig.Compaction.AutoCompactTurns {
			a.compact(history, false)
			history = a.loopHistory()
		}
	}

	requestMessages := []Message{{Role: "system", Content: fullSystemPrompt}}
	if a.Depth == 0 {
		if summary, boundary := a.compactionSummary(); summary != "" {
			requestMessages = append(requestMessages, Message{
				Role: "user",
				Content: fmt.Sprintf(
					"=== PRIOR CONTEXT (archived turns 1-%d) ===\n%s\n(Raw archived turns are under .goharness/sessions/%s/compacted_summary_up_to_turn_%03d/ if you need to inspect them.)",
					boundary, summary, a.SessionID, boundary),
			})
		}
	}
	requestMessages = append(requestMessages, history...)

	maxTurns := activeConfig.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 15
	}
	var lastTextAnswer string

	for turn := 1; turn <= maxTurns; turn++ {
		writeDebugLog("[%s] turn %d/%d", a.SessionID, turn, maxTurns)
		startTime := time.Now()
		responseMsg, err := a.request(ctx, requestMessages, tools)
		if err != nil {
			LogExecutionTrace(turn, "llm_completion", startTime, "failed", map[string]interface{}{"error": err.Error(), "session": a.SessionID})
			if a.Depth == 0 {
				BroadcastSSE("turn_secured", map[string]interface{}{
					"turn_number": 0, "role": "system", "name": "system",
					"content": fmt.Sprintf("❌ [LLM API ERROR] %v", err),
				})
			}
			return fmt.Sprintf("Error: LLM API call failed: %v", err)
		}
		LogExecutionTrace(turn, "llm_completion", startTime, "success", map[string]interface{}{
			"model": a.API.Model, "provider": a.API.Provider, "session": a.SessionID,
		})

		a.saveTurn(*responseMsg)
		requestMessages = append(requestMessages, *responseMsg)

		if responseMsg.Content != "" {
			lastTextAnswer = responseMsg.Content
		}
		if len(responseMsg.ToolCalls) == 0 {
			return lastTextAnswer
		}

		// Execute all tool calls. spawn_sub_agent calls run concurrently;
		// all other tools run sequentially afterward, preserving order.
		results := a.runToolCalls(ctx, turn, responseMsg.ToolCalls)
		for _, tr := range results {
			toolMsg := Message{
				Role:       "tool",
				ToolCallID: tr.ToolCallID,
				Name:       tr.Name,
				Content:    tr.Result,
			}
			a.saveTurn(toolMsg)
			requestMessages = append(requestMessages, toolMsg)
		}
	}

	if lastTextAnswer == "" {
		lastTextAnswer = "Reached maximum turns without a final answer."
	}
	return lastTextAnswer
}

// tryWorkflowRoute runs the active DAG workflow if one is configured and
// on disk. Returns (answer, true) if a workflow produced an answer.
func (a *Agent) tryWorkflowRoute(userPrompt string) (string, bool) {
	wfPath := GetSystemPath("workflows.json")
	if _, err := os.Stat(wfPath); err != nil {
		return "", false
	}
	cfg, err := LoadWorkflowConfig()
	if err != nil || cfg.ActiveWorkflow == "" {
		return "", false
	}
	answer, execErr := ExecuteActiveWorkflow(userPrompt)
	if execErr == nil {
		return answer, true
	}
	fmt.Printf("%s[WORKFLOW EXECUTION ERROR] %v. Falling back to native tools loop.%s\n", ColorRed, execErr, ColorReset)
	return "", false
}

// agentTools returns the tool set for this agent, applying sub-agent gating.
func (a *Agent) agentTools() []Tool {
	// All built-in tools + MCP tools (same as the root loop).
	tools := selectTools(nil, true)
	if a.Depth == 0 {
		return tools
	}
	// Sub-agents: strip write_file and patch_file by default, and strip
	// spawn_sub_agent once we're at the recursion limit.
	allowSpawn := a.Depth < MaxSubAgentDepth
	filtered := make([]Tool, 0, len(tools))
	for _, t := range tools {
		name := t.Function.Name
		if name == "write_file" || name == "patch_file" {
			continue // read-only sub-agents by default
		}
		if name == "spawn_sub_agent" && !allowSpawn {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

// toolCallResult pairs a tool's output with the call it responded to.
type toolCallResult struct {
	Index      int
	ToolCallID string
	Name       string
	Result     string
}

// runToolCalls executes the tool calls from one assistant response.
// spawn_sub_agent calls run concurrently in goroutines; other tools run
// sequentially afterward. Results are returned in the original call order so
// the model can correlate tool_call_id with its result.
func (a *Agent) runToolCalls(ctx context.Context, turn int, calls []ToolCall) []toolCallResult {
	results := make([]toolCallResult, len(calls))
	var spawnIdx []int
	for i, tc := range calls {
		if tc.Function.Name == "spawn_sub_agent" {
			spawnIdx = append(spawnIdx, i)
		}
	}

	// Fan out sub-agents concurrently.
	if len(spawnIdx) > 0 {
		var wg sync.WaitGroup
		for _, i := range spawnIdx {
			wg.Add(1)
			go func(idx int, tc ToolCall) {
				defer wg.Done()
				results[idx] = toolCallResult{
					Index:      idx,
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Result:     a.runSubAgent(ctx, tc),
				}
			}(i, calls[i])
		}
		wg.Wait()
	}

	// Sequential tools (reads, writes, commands) in original order.
	for i, tc := range calls {
		if tc.Function.Name == "spawn_sub_agent" {
			continue // already done
		}
		results[i] = toolCallResult{
			Index:      i,
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
			Result:     executeToolCall(a, turn, tc),
		}
	}
	return results
}

// compactionSummary reads the latest compacted summary for this agent's
// session, if any, returning its text and boundary turn number.
func (a *Agent) compactionSummary() (string, int) {
	boundary := getSessionCompactionBoundary(a.SessionID)
	if boundary == 0 {
		return "", 0
	}
	path := filepath.Join(a.sessionPath(), fmt.Sprintf("compacted_summary_up_to_turn_%03d.json", boundary))
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", 0
	}
	return string(bytes), boundary
}

// compact runs sliding-window compaction for this agent's session.
func (a *Agent) compact(history []Message, force bool) {
	executeSlidingWindowCompaction(a, history, force)
}

// loopHistory returns this agent's turns from disk after the compaction
// boundary (matching the old root-loop filter).
func (a *Agent) loopHistory() []Message {
	_, boundary := a.compactionSummary()
	entries, err := os.ReadDir(a.sessionPath())
	if err != nil {
		return nil
	}
	var out []Message
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || len(name) <= 3 {
			continue
		}
		turnIdx, err := strconv.Atoi(name[:3])
		if err != nil || turnIdx <= boundary {
			continue
		}
		if bytes, rerr := os.ReadFile(filepath.Join(a.sessionPath(), name)); rerr == nil {
			var msg Message
			if json.Unmarshal(bytes, &msg) == nil {
				out = append(out, msg)
			}
		}
	}
	return out
}
