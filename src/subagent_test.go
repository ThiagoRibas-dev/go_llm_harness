package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAgentTurnIsolation verifies two agents writing turns concurrently do
// not stomp each other's session folders or turn numbers (the original
// global-state bug that made sub-agents sequential).
func TestAgentTurnIsolation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	activeConfig = &Config{}

	a1 := &Agent{SessionID: "sess_a", Workspace: root, throttlers: map[string]chan struct{}{}}
	a2 := &Agent{SessionID: "sess_b", Workspace: root, throttlers: map[string]chan struct{}{}}
	for _, a := range []*Agent{a1, a2} {
		if err := os.MkdirAll(a.sessionPath(), 0755); err != nil {
			t.Fatal(err)
		}
	}

	done := make(chan struct{}, 2)
	for _, a := range []*Agent{a1, a2} {
		go func(agent *Agent) {
			for i := 0; i < 20; i++ {
				agent.saveTurn(Message{Role: "user", Content: "x"})
			}
			done <- struct{}{}
		}(a)
	}
	<-done
	<-done

	for _, a := range []*Agent{a1, a2} {
		files, _ := os.ReadDir(a.sessionPath())
		if len(files) != 20 {
			t.Errorf("%s: expected 20 turn files, got %d", a.SessionID, len(files))
		}
		if a.turn != 20 {
			t.Errorf("%s: expected turn counter 20, got %d", a.SessionID, a.turn)
		}
	}
}

// TestRunToolCallsOrdering ensures tool results are returned in the same
// order as the input calls even when sub-agents finish out of order.
func TestRunToolCallsOrdering(t *testing.T) {
	a := &Agent{SessionID: "ord", Workspace: t.TempDir(), throttlers: map[string]chan struct{}{}}

	// Two sub-agent calls; make the second finish first by sleeping in a
	// custom runner. We can't easily inject the LLM, so test the sequential
	// path ordering deterministically with a fake tool name.
	calls := []ToolCall{
		{ID: "1", Function: ToolFunction{Name: "read_file", Arguments: `{}`}},
		{ID: "2", Function: ToolFunction{Name: "read_file", Arguments: `{}`}},
	}
	results := a.runToolCalls(context.Background(), 1, calls)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ToolCallID != "1" || results[1].ToolCallID != "2" {
		t.Errorf("results out of order: %+v", results)
	}
}

// TestAgentRequestThrottle verifies a sub-agent's request() respects the
// shared parent throttle map.
func TestAgentRequestThrottle(t *testing.T) {
	a := &Agent{
		SessionID:   "throttle",
		Workspace:   t.TempDir(),
		API:         APIConfig{MaxConcurrency: 1},
		ProfileName: "p",
		throttlers:  map[string]chan struct{}{},
	}
	ctx := context.Background()
	r1 := a.acquireThrottle(ctx, a.ProfileName, a.API)
	defer r1()

	acquired := make(chan struct{})
	go func() {
		r2 := a.acquireThrottle(ctx, a.ProfileName, a.API)
		defer r2()
		close(acquired)
	}()
	select {
	case <-acquired:
		t.Fatal("second acquire succeeded before first release")
	case <-time.After(40 * time.Millisecond):
		// expected
	}
}

// TestMaxSubAgentDepth ensures the agent tool set strips spawn_sub_agent
// beyond the recursion limit. Sub-agents keep write tools (serialized by
// the workspace lock); only deeper spawning is cut off.
func TestMaxSubAgentDepth(t *testing.T) {
	a := &Agent{Depth: MaxSubAgentDepth}
	tools := a.agentTools()
	for _, tl := range tools {
		if tl.Function.Name == "spawn_sub_agent" {
			t.Fatal("spawn_sub_agent should be stripped at max depth")
		}
	}
	// A child below the limit can still spawn and write.
	child := &Agent{Depth: 1}
	var hasSpawn, hasWrite bool
	for _, tl := range child.agentTools() {
		if tl.Function.Name == "spawn_sub_agent" {
			hasSpawn = true
		}
		if tl.Function.Name == "write_file" {
			hasWrite = true
		}
	}
	if !hasSpawn {
		t.Error("child at depth 1 should be able to spawn sub-agents")
	}
	if !hasWrite {
		t.Error("sub-agents should retain write tools (serialized by the lock)")
	}
}

// TestWorkspaceLockIsExclusive confirms two writers block each other but
// don't deadlock.
func TestWorkspaceLockIsExclusive(t *testing.T) {
	got := make(chan int, 2)
	go func() {
		withWriteLock(func() string { time.Sleep(30 * time.Millisecond); got <- 1; return "" })
	}()
	time.Sleep(10 * time.Millisecond)
	go func() {
		withWriteLock(func() string { got <- 2; return "" })
	}()
	first := <-got
	second := <-got
	if first != 1 || second != 2 {
		t.Errorf("expected ordered writes 1 then 2, got %d then %d", first, second)
	}
}

// TestSubAgentComposePrompt verifies the structured task/context/expect
// fields are assembled into labeled sections and optional sections are
// omitted when empty.
func TestSubAgentComposePrompt(t *testing.T) {
	full := subAgentSpec{
		Task:    "audit auth.go",
		Context: "we use session tokens",
		Expect:  "a bullet list",
	}.composePrompt()
	for _, want := range []string{"## Task", "audit auth.go", "## Context", "## Return", "bullet list"} {
		if !strings.Contains(full, want) {
			t.Errorf("composed prompt missing %q:\n%s", want, full)
		}
	}

	minimal := subAgentSpec{Task: "just do it"}.composePrompt()
	if strings.Contains(minimal, "## Context") || strings.Contains(minimal, "## Return") {
		t.Errorf("optional sections should be omitted when empty:\n%s", minimal)
	}
	if !strings.Contains(minimal, "just do it") {
		t.Errorf("task missing from minimal prompt")
	}
}

// guard to ensure filepath import is used
var _ = filepath.Join
