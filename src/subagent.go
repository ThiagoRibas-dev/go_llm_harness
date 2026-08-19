package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// workspaceGate is a process-wide exclusive lock for file-writing tools.
// Reads and execute_command don't take it; write_file/patch_file do, so a
// write-enabled sub-agent can't overlap a parent write or another writer.
var workspaceGate chan struct{}

func init() {
	workspaceGate = make(chan struct{}, 1)
	workspaceGate <- struct{}{}
}

// withWriteLock runs fn while holding the exclusive workspace write lock.
func withWriteLock(fn func() string) string {
	<-workspaceGate
	defer func() { workspaceGate <- struct{}{} }()
	return fn()
}

// subAgentSpec is the parsed shape of a spawn_sub_agent tool call.
type subAgentSpec struct {
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
}

// runSubAgent executes one spawn_sub_agent call: it creates an isolated
// child session, builds a child Agent, runs the ReAct loop in that session,
// and returns a structured report to the parent.
func (a *Agent) runSubAgent(ctx context.Context, tc ToolCall) string {
	var spec subAgentSpec
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &spec); err != nil || spec.Prompt == "" {
		return fmt.Sprintf("spawn_sub_agent error: invalid arguments: %v", err)
	}
	if a.Depth >= MaxSubAgentDepth {
		return "spawn_sub_agent error: maximum sub-agent depth reached"
	}

	childSessID := fmt.Sprintf("%s_sub_%d", a.SessionID, time.Now().UnixNano())
	childPath := GetSystemPath(filepath.Join(".goharness", "sessions", childSessID))
	_ = os.MkdirAll(childPath, 0755)
	createSessionMeta(childSessID, a.Workspace, a.SessionID, "Sub-agent: "+spec.Description)

	child := a.Sub(childSessID)

	// Announce to the UI (only the root session's UI is live, but the event
	// is harmless for children).
	BroadcastSSE("subagent_start", map[string]interface{}{
		"parent_session": a.SessionID,
		"session_id":     childSessID,
		"description":    spec.Description,
		"depth":          child.Depth,
	})

	start := time.Now()
	answer := child.Run(ctx, spec.Prompt)
	duration := time.Since(start).Milliseconds()

	BroadcastSSE("subagent_done", map[string]interface{}{
		"parent_session": a.SessionID,
		"session_id":     childSessID,
		"description":    spec.Description,
		"duration_ms":    duration,
	})

	return fmt.Sprintf("=== SUB-AGENT REPORT (%s) ===\nTask: %s\n\n%s\n=================================", childSessID, spec.Prompt, answer)
}
