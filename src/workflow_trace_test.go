package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// drainChannel reads buffered SSE packets until idle.
func drainChannel(ch chan string) []string {
	var out []string
	for {
		select {
		case msg := <-ch:
			out = append(out, msg)
		case <-time.After(300 * time.Millisecond):
			return out
		}
	}
}

func eventsOfType(msgs []string, want string) []map[string]interface{} {
	var events []map[string]interface{}
	for _, m := range msgs {
		var pkt struct {
			Event string                 `json:"event"`
			Data  map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal([]byte(m), &pkt); err != nil {
			continue
		}
		if pkt.Event == want {
			events = append(events, pkt.Data)
		}
	}
	return events
}

// TestWorkflowLiveTraceEvents verifies that executing a DAG emits the three
// live-trace SSE events (workflow_start, per-node running/completed,
// workflow_end). It uses a tool_execution node so no LLM API key is required.
func TestWorkflowLiveTraceEvents(t *testing.T) {
	activeConfig = &Config{}
	activeSessionID = "test_session"
	currentTurnNumber = 0
	activeConfig.Agent.WorkspaceDir = t.TempDir()

	executor := &WorkflowExecutor{
		SessionID:    activeSessionID,
		WorkflowID:   "test_wf",
		WorkflowName: "Test Workflow",
		RunID:        "run_test_1",
		TurnNumber:   1,
		NodeOrder:    []string{"start", "echo", "terminal"},
		Nodes: map[string]*RuntimeNode{
			"start": {
				ID: "start", Type: "user_input",
				Properties: map[string]interface{}{},
				Inputs:     []NodeConnection{},
				Outputs:    map[string]interface{}{},
				State:      StatePending,
				readyChan:  make(chan struct{}),
				doneChan:   make(chan struct{}),
			},
			"echo": {
				ID: "echo", Type: "tool_execution",
				Properties: map[string]interface{}{"tool_name": "execute_command"},
				Inputs: []NodeConnection{
					{SourceNode: "start", SourceOutput: "prompt", TargetInput: "arguments"},
				},
				Outputs:   map[string]interface{}{},
				State:     StatePending,
				readyChan: make(chan struct{}),
				doneChan:  make(chan struct{}),
			},
			"terminal": {
				ID: "terminal", Type: "assistant_response",
				Properties: map[string]interface{}{},
				Inputs: []NodeConnection{
					{SourceNode: "echo", SourceOutput: "stdout", TargetInput: "final_output"},
				},
				Outputs:   map[string]interface{}{},
				State:     StatePending,
				readyChan: make(chan struct{}),
				doneChan:  make(chan struct{}),
			},
		},
		Timeout: 10 * time.Second,
	}

	// Intercept SSE broadcasts.
	clientsMu.Lock()
	ch := make(chan string, 64)
	sseClients = []chan string{ch}
	clientsMu.Unlock()
	defer func() {
		clientsMu.Lock()
		sseClients = nil
		clientsMu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = executor.Execute(ctx, "echo hello")

	msgs := drainChannel(ch)

	// workflow_start
	starts := eventsOfType(msgs, "workflow_start")
	if len(starts) != 1 {
		t.Fatalf("expected 1 workflow_start, got %d: %v", len(starts), msgs)
	}
	if starts[0]["run_id"] != "run_test_1" {
		t.Errorf("workflow_start run_id = %v", starts[0]["run_id"])
	}
	if nodes, ok := starts[0]["nodes"].([]interface{}); !ok || len(nodes) != 1 {
		t.Errorf("expected 1 non-anchor node listed, got %v", starts[0]["nodes"])
	}

	// workflow_node lifecycle for echo (running then completed)
	nodeEvents := eventsOfType(msgs, "workflow_node")
	sawRunning, sawCompleted := false, false
	for _, ne := range nodeEvents {
		if ne["node_id"] != "echo" {
			continue
		}
		switch ne["status"] {
		case "running":
			sawRunning = true
			if _, ok := ne["started_at"].(string); !ok {
				t.Errorf("running event missing started_at: %v", ne)
			}
		case "completed":
			sawCompleted = true
			if _, ok := ne["duration_ms"].(float64); !ok {
				t.Errorf("completed event missing duration_ms: %v", ne)
			}
			if _, ok := ne["preview"].(string); !ok {
				t.Errorf("completed event missing preview: %v", ne)
			}
		}
	}
	if !sawRunning || !sawCompleted {
		t.Fatalf("node lifecycle incomplete (running=%v completed=%v): %v", sawRunning, sawCompleted, nodeEvents)
	}

	// workflow_end
	ends := eventsOfType(msgs, "workflow_end")
	if len(ends) != 1 {
		t.Fatalf("expected 1 workflow_end, got %d", len(ends))
	}
	if ends[0]["status"] != "completed" {
		t.Errorf("workflow_end status = %v, want completed", ends[0]["status"])
	}
}
