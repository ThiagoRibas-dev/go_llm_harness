package main

import (
	"context"
	"strings"
	"testing"
)

// TestLLMNodeTypeAccepted is a regression test for the visual editor shipping
// nodes with type "llm" (the frontend normalizes the legacy "llm_query"/
// "llm_synthesis" aliases to "llm"). The executor's runNodeLogic switch must
// accept the plain "llm" type; otherwise the default linear_chat workflow
// (whose query_node is type "llm") fails with "unknown node type: llm".
func TestLLMNodeTypeAccepted(t *testing.T) {
	activeConfig = &Config{}
	e := &WorkflowExecutor{
		Nodes: map[string]*RuntimeNode{
			"start": {
				ID: "start", Type: "user_input",
				Outputs:   map[string]interface{}{"prompt": "hi"},
				readyChan: make(chan struct{}),
				doneChan:  make(chan struct{}),
			},
			"llm": {
				ID: "llm", Type: "llm",
				Properties: map[string]interface{}{
					"provider":      "openai",
					"model":         "gpt-4o-mini",
					"system_prompt": "you are a test",
				},
				Inputs: []NodeConnection{
					{SourceNode: "start", SourceOutput: "prompt", TargetInput: "prompt"},
				},
				Outputs:   map[string]interface{}{},
				readyChan: make(chan struct{}),
				doneChan:  make(chan struct{}),
			},
		},
	}
	close(e.Nodes["start"].readyChan)
	close(e.Nodes["start"].doneChan)
	e.Nodes["start"].State = StateCompleted

	err := e.runNodeLogic(context.Background(), e.Nodes["llm"],
		map[string]interface{}{"prompt": "hello"})

	// We do not expect an API call to succeed in the test environment, but it
	// MUST NOT be rejected as an unknown node type.
	if err == nil {
		t.Skip("llm node ran without error (unexpected in unit test); accepting 'llm' type is confirmed")
	}
	if strings.Contains(err.Error(), "unknown node type") {
		t.Fatalf("plain 'llm' node type was rejected by runNodeLogic: %v", err)
	}
	// Any other error (network, missing key, etc.) proves the type was accepted
	// and the LLM branch was entered.
	t.Logf("llm node reached the LLM branch (error as expected without API: %v)", err)
}
