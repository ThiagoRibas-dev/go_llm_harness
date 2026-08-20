package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Task        string `json:"task"`
	Context     string `json:"context"`
	Expect      string `json:"expect"`
	Description string `json:"description"`
}

// composePrompt builds the user message the child agent receives from the
// structured inputs. The child's own system prompt (in Agent.Run) already
// covers the environment, tools, and workspace; this just frames the job.
func (s subAgentSpec) composePrompt() string {
	var b strings.Builder
	b.WriteString("## Task\n")
	b.WriteString(s.Task)
	if strings.TrimSpace(s.Context) != "" {
		b.WriteString("\n\n## Context\n")
		b.WriteString(s.Context)
	}
	if strings.TrimSpace(s.Expect) != "" {
		b.WriteString("\n\n## Return\n")
		b.WriteString(s.Expect)
	}
	return b.String()
}

// runSubAgent executes one spawn_sub_agent call: it creates an isolated
// child session, builds a child Agent, runs the ReAct loop in that session,
// and returns the child's final answer to the parent.
func (a *Agent) runSubAgent(ctx context.Context, tc ToolCall) string {
	var spec subAgentSpec
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &spec); err != nil || strings.TrimSpace(spec.Task) == "" {
		return fmt.Sprintf("spawn_sub_agent error: 'task' is required: %v", err)
	}
	if a.Depth >= MaxSubAgentDepth {
		return "spawn_sub_agent error: maximum sub-agent depth reached"
	}

	childSessID := fmt.Sprintf("%s_sub_%d", a.SessionID, time.Now().UnixNano())
	childPath := GetSystemPath(filepath.Join(".goharness", "sessions", childSessID))
	_ = os.MkdirAll(childPath, 0755)
	label := spec.Description
	if label == "" {
		label = spec.Task
	}
	createSessionMeta(childSessID, a.Workspace, a.SessionID, "Sub-agent: "+label)

	child := a.Sub(childSessID)

	BroadcastSSE("subagent_start", map[string]interface{}{
		"parent_session": a.SessionID,
		"session_id":     childSessID,
		"description":    label,
		"depth":          child.Depth,
	})

	start := time.Now()
	answer := child.Run(ctx, spec.composePrompt())
	duration := time.Since(start).Milliseconds()

	BroadcastSSE("subagent_done", map[string]interface{}{
		"parent_session": a.SessionID,
		"session_id":     childSessID,
		"description":    label,
		"duration_ms":    duration,
	})

	return answer
}
