package main

import (
	"context"
	"os"
	"path/filepath"
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
// beyond the recursion limit.
func TestMaxSubAgentDepth(t *testing.T) {
	a := &Agent{Depth: MaxSubAgentDepth}
	tools := a.agentTools()
	for _, tl := range tools {
		if tl.Function.Name == "spawn_sub_agent" {
			t.Fatal("spawn_sub_agent should be stripped at max depth")
		}
		if tl.Function.Name == "write_file" || tl.Function.Name == "patch_file" {
			t.Fatal("sub-agents should not have write tools")
		}
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

// guard to ensure filepath import is used
var _ = filepath.Join
