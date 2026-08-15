package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// WorkflowConfig holds the schema of workflows.json
type WorkflowConfig struct {
	ActiveWorkflow string              `json:"active_workflow"`
	Workflows      map[string]Workflow `json:"workflows"`
}

type Workflow struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Nodes       []WorkflowNode `json:"nodes"`
}

type WorkflowNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Inputs     []NodeConnection       `json:"inputs"`
}

// NodeConnection defines a directed link mapping data between nodes
type NodeConnection struct {
	SourceNode   string `json:"source_node"`
	SourceOutput string `json:"source_output"`
	TargetInput  string `json:"target_input"`
}

// RuntimeNode represents a running instance of a node inside the DAG
type RuntimeNode struct {
	ID         string
	Type       string
	Properties map[string]interface{}
	Inputs     []NodeConnection
	Outputs    map[string]interface{}
	State      WorkflowNodeState
	Error      error
	
	mu         sync.RWMutex
	readyChan  chan struct{}
	doneChan   chan struct{}
}

type WorkflowNodeState string
const (
	StatePending   WorkflowNodeState = "pending"
	StateRunning   WorkflowNodeState = "running"
	StateCompleted WorkflowNodeState = "completed"
	StateFailed    WorkflowNodeState = "failed"
)

// WorkflowExecutor executes a given workflow DAG
type WorkflowExecutor struct {
	SessionID string
	Nodes     map[string]*RuntimeNode
	Timeout   time.Duration
}

// LoadWorkflowConfig reads the workflows.json file from disk
func LoadWorkflowConfig() (*WorkflowConfig, error) {
	bytes, err := os.ReadFile(GetSystemPath("workflows.json"))
	if err != nil {
		return nil, err
	}
	var cfg WorkflowConfig
	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ExecuteActiveWorkflow runs the active workflow DAG inside workflows.json
func ExecuteActiveWorkflow(rawUserPrompt string) (string, error) {
	cfg, err := LoadWorkflowConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load workflows.json: %w", err)
	}

	wf, ok := cfg.Workflows[cfg.ActiveWorkflow]
	if !ok {
		return "", fmt.Errorf("active workflow '%s' not found inside workflows.json", cfg.ActiveWorkflow)
	}

	writeDebugLog("[WORKFLOW ENGINE] Hot-swapping and instantiating DAG pipeline: '%s' (%s)", wf.Name, cfg.ActiveWorkflow)

	// Build the runtime executor DAG
	executor := &WorkflowExecutor{
		SessionID: activeSessionID,
		Nodes:     make(map[string]*RuntimeNode),
		Timeout:   120 * time.Second, // 2-minute hard timeout boundary
	}

	for _, n := range wf.Nodes {
		executor.Nodes[n.ID] = &RuntimeNode{
			ID:         n.ID,
			Type:       n.Type,
			Properties: n.Properties,
			Inputs:     n.Inputs,
			Outputs:    make(map[string]interface{}),
			State:      StatePending,
			readyChan:  make(chan struct{}),
			doneChan:   make(chan struct{}),
		}
	}

	// Verify required anchors exist
	if _, ok := executor.Nodes["start"]; !ok {
		return "", fmt.Errorf("invalid graph: missing start 'user_input' node anchor")
	}
	if _, ok := executor.Nodes["terminal"]; !ok {
		return "", fmt.Errorf("invalid graph: missing terminal 'assistant_response' node anchor")
	}

	// Run execution loop
	ctx := context.Background()
	err = executor.Execute(ctx, rawUserPrompt)
	if err != nil {
		return "", err
	}

	// Fetch final terminal output
	terminal := executor.Nodes["terminal"]
	terminal.mu.RLock()
	finalOut, _ := terminal.Outputs["final_output"].(string)
	terminal.mu.RUnlock()

	return finalOut, nil
}

// Execute performs concurrent topological execution of the DAG
func (e *WorkflowExecutor) Execute(ctx context.Context, rawUserPrompt string) error {
	ctx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	var wg sync.WaitGroup

	// 1. Initialize start node inputs
	startNode := e.Nodes["start"]
	startNode.Outputs["prompt"] = rawUserPrompt
	close(startNode.readyChan)
	close(startNode.doneChan)
	startNode.State = StateCompleted

	// 2. Spawn concurrent worker goroutines for every downstream node
	for id, node := range e.Nodes {
		if id == "start" {
			continue
		}
		
		wg.Add(1)
		go func(n *RuntimeNode) {
			defer wg.Done()
			
			// Wait for upstreams to signal readiness
			select {
			case <-n.readyChan:
				// Proceed to execution
			case <-ctx.Done():
				n.mu.Lock()
				n.State = StateFailed
				n.Error = ctx.Err()
				n.mu.Unlock()
				close(n.doneChan)
				return
			}

			n.mu.Lock()
			n.State = StateRunning
			n.mu.Unlock()

			// Resolve actual input data from upstream outputs
			inputPayload := e.resolveIncomingData(n)

			// Execute node logic based on type
			err := e.runNodeLogic(ctx, n, inputPayload)
			
			n.mu.Lock()
			if err != nil {
				n.State = StateFailed
				n.Error = err
			} else {
				n.State = StateCompleted
			}
			n.mu.Unlock()

			// Signal all downstreams that this node's outputs are ready
			close(n.doneChan)
		}(node)
	}

	// 3. Spawn a background coordinate goroutine to resolve edge readiness dynamically
	go e.coordinateEdgeReadiness(ctx)

	wg.Wait()

	// Check if terminal node completed successfully
	terminal := e.Nodes["terminal"]
	if terminal.State != StateCompleted {
		return fmt.Errorf("workflow execution failed: %v", terminal.Error)
	}

	return nil
}

// coordinateEdgeReadiness loops to close readyChan when a node's upstreams are done
func (e *WorkflowExecutor) coordinateEdgeReadiness(ctx context.Context) {
	for {
		allResolved := true
		for id, node := range e.Nodes {
			if id == "start" {
				continue
			}
			
			node.mu.RLock()
			isPending := node.State == StatePending
			node.mu.RUnlock()

			if isPending {
				allResolved = false
				
				// Verify if all connecting upstream nodes have completed
				upstreamsReady := true
				for _, conn := range node.Inputs {
					src := e.Nodes[conn.SourceNode]
					src.mu.RLock()
					srcDone := src.State == StateCompleted
					src.mu.RUnlock()
					
					if !srcDone {
						upstreamsReady = false
						break
					}
				}
				
				if upstreamsReady {
					// Toggle standard Go channel safety closures
					node.mu.Lock()
					select {
					case <-node.readyChan:
						// Already closed
					default:
						close(node.readyChan)
					}
					node.mu.Unlock()
				}
			}
		}

		if allResolved {
			break
		}
		
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
			// Small polling tick
		}
	}
}

func (e *WorkflowExecutor) resolveIncomingData(n *RuntimeNode) map[string]interface{} {
	payload := make(map[string]interface{})
	for _, conn := range n.Inputs {
		src := e.Nodes[conn.SourceNode]
		src.mu.RLock()
		val := src.Outputs[conn.SourceOutput]
		src.mu.RUnlock()
		payload[conn.TargetInput] = val
	}
	return payload
}

func (e *WorkflowExecutor) runNodeLogic(ctx context.Context, n *RuntimeNode, inputs map[string]interface{}) error {
	switch n.Type {
	case "user_input":
		return nil

	case "llm_query", "llm_synthesis":
		provider, _ := n.Properties["provider"].(string)
		model, _ := n.Properties["model"].(string)
		tempVal, _ := n.Properties["temperature"]
		temperature := 0.2
		if t, ok := tempVal.(float64); ok {
			temperature = t
		}
		systemPrompt, _ := n.Properties["system_prompt"].(string)

		promptText := ""
		// Accumulate inputs into the final prompt text
		var inputParts []string
		for k, v := range inputs {
			inputParts = append(inputParts, fmt.Sprintf("%s:\n%v", k, v))
		}
		if len(inputParts) > 0 {
			promptText = strings.Join(inputParts, "\n\n")
		} else {
			promptText = "Process baseline context."
		}

		writeDebugLog("[WORKFLOW NODE %s] LLM query requested: provider: %s, model: %s", n.ID, provider, model)

		// Temporarily hot-swap global API settings for this node execution!
		parentAPI := activeConfig.API
		activeConfig.API = APIConfig{
			Provider:    provider,
			Key:         activeConfig.API.Key, // Falls back to global API Key
			BaseURL:     "",                    // Auto built by backend
			Model:       model,
			Temperature: temperature,
			MaxTokens:   activeConfig.API.MaxTokens,
		}

		// Check if specialized compaction credentials are set
		if provider == "openai" && activeConfig.API.Key == "" {
			activeConfig.API.Key = parentAPI.Key
		}

		reqMessages := []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: promptText},
		}

		respMsg, err := SendMultiProviderRequest(reqMessages, nil)
		
		// Restore parent global settings immediately on return
		activeConfig.API = parentAPI

		if err != nil {
			writeDebugLog("[WORKFLOW NODE %s ERROR] LLM Query failed: %v", n.ID, err)
			return err
		}

		n.mu.Lock()
		n.Outputs["response"] = respMsg.Content
		n.mu.Unlock()

		writeDebugLog("[WORKFLOW NODE %s] LLM Query completed successfully (%d bytes returned)", n.ID, len(respMsg.Content))
		return nil

	case "bm25_search":
		query, _ := inputs["query"].(string)
		limitVal, _ := n.Properties["limit"]
		limit := 5
		if l, ok := limitVal.(float64); ok {
			limit = int(l)
		}
		scope, _ := n.Properties["scope"].(string)
		if scope == "" {
			scope = "workspace"
		}

		writeDebugLog("[WORKFLOW NODE %s] BM25 Search requested: query: '%s', scope: %s", n.ID, query, scope)
		result := executeBM25Search(query, scope, limit)

		n.mu.Lock()
		n.Outputs["search_results"] = result
		n.mu.Unlock()

		writeDebugLog("[WORKFLOW NODE %s] BM25 Search completed successfully", n.ID)
		return nil

	case "tool_execution":
		toolName, _ := n.Properties["tool_name"].(string)
		args, _ := inputs["arguments"].(string)
		writeDebugLog("[WORKFLOW NODE %s] Executing tool: %s", n.ID, toolName)

		var result string
		var exitCode int

		if toolName == "write_file" {
			var m map[string]string
			_ = json.Unmarshal([]byte(args), &m)
			result = executeWriteFile(m["path"], m["content"])
		} else if toolName == "patch_file" {
			var m map[string]string
			_ = json.Unmarshal([]byte(args), &m)
			result = executePatchFile(m["path"], m["search"], m["replace"])
		} else if toolName == "execute_command" {
			var m map[string]string
			_ = json.Unmarshal([]byte(args), &m)
			result = executeTerminalCommand(m["command"])
			if strings.Contains(result, "Sandbox Execution Error") || strings.Contains(result, "Security Exception") {
				exitCode = 1
			}
		} else if toolName == "read_file" {
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(args), &m)
			start, _ := m["start_line"].(float64)
			end, _ := m["end_line"].(float64)
			path, _ := m["path"].(string)
			result = executeReadFile(path, int(start), int(end))
		} else {
			return fmt.Errorf("unknown workflow tool name: %s", toolName)
		}

		n.mu.Lock()
		n.Outputs["stdout"] = result
		n.Outputs["exit_code"] = exitCode
		n.mu.Unlock()

		writeDebugLog("[WORKFLOW NODE %s] Tool execution completed successfully", n.ID)
		return nil

	case "conditional_router":
		// Standard evaluation of logic parameters
		condition, _ := n.Properties["condition"].(string)
		evalVar := ""
		if v, ok := inputs["eval_var"].(string); ok {
			evalVar = v
		}

		writeDebugLog("[WORKFLOW NODE %s] Evaluating routing condition: '%s'", n.ID, condition)
		targetRoute := "default"
		
		if condition == "on_error" && (strings.Contains(evalVar, "Error") || strings.Contains(evalVar, "Exception")) {
			targetRoute = "error_branch"
		}

		n.mu.Lock()
		n.Outputs["route_branch"] = targetRoute
		n.mu.Unlock()

		writeDebugLog("[WORKFLOW NODE %s] Routed execution dynamically to branch: %s", n.ID, targetRoute)
		return nil

	case "assistant_response":
		finalOutput, _ := inputs["final_output"].(string)
		writeDebugLog("[WORKFLOW NODE %s] Rending final terminal assistant response.", n.ID)

		// Broadcast final compiled answer over SSE
		BroadcastSSE("turn_secured", map[string]interface{}{
			"turn_number": currentTurnNumber + 1, // Visual sequence update
			"role":        "assistant",
			"name":        "assistant",
			"content":     finalOutput,
		})

		// Write turn file to disk to preserve session liveness
		finalMsg := Message{
			Role:    "assistant",
			Content: finalOutput,
		}
		saveMessageTurn(finalMsg)

		return nil

	default:
		return fmt.Errorf("unknown node type: %s", n.Type)
	}
}
