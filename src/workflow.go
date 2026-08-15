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
	SessionID    string
	WorkflowID   string
	WorkflowName string
	RunID        string       // Unique id for this execution run (groups live node events)
	TurnNumber   int          // Final assistant turn number this run maps to in the UI
	NodeOrder    []string     // Node ids in declaration order (for stable UI rendering)
	Nodes        map[string]*RuntimeNode
	Timeout      time.Duration
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

// DefaultWorkflowConfig returns the built-in set of workflows shipped with
// GoHarness v2.0: Standard Linear Chat and Enhanced Cognition (POADR). It is
// the single source of truth used both to bootstrap workflows.json on first
// run and as the HTTP fallback when the file is missing on disk, ensuring the
// two defaults never drift between the loader, the seeder, and the web UI.
func DefaultWorkflowConfig() *WorkflowConfig {
	return &WorkflowConfig{
		ActiveWorkflow: "linear_chat",
		Workflows: map[string]Workflow{
			"linear_chat": {
				Name:        "Standard Linear Chat",
				Description: "Standard conversational agent loop mapping user input to a single, high-fidelity LLM response.",
				Nodes: []WorkflowNode{
					{
						ID:         "start",
						Type:       "user_input",
						Properties: map[string]interface{}{},
						Inputs:     []NodeConnection{},
					},
					{
						ID:   "query_node",
						Type: "llm_query",
						Properties: map[string]interface{}{
							"provider":      "openai",
							"model":         "gpt-4o",
							"temperature":   0.0,
							"system_prompt": "You are a highly capable agent with access to a local terminal sandbox. Use your tools to solve the user's request.",
						},
						Inputs: []NodeConnection{
							{SourceNode: "start", SourceOutput: "prompt", TargetInput: "prompt"},
						},
					},
					{
						ID:         "terminal",
						Type:       "assistant_response",
						Properties: map[string]interface{}{},
						Inputs: []NodeConnection{
							{SourceNode: "query_node", SourceOutput: "response", TargetInput: "final_output"},
						},
					},
				},
			},
			"enhanced_cognition": {
				Name:        "Enhanced Cognition (POADR)",
				Description: "Decomposes your query concurrently across 5 parallel cognitive axes to eliminate representational interference in smaller models, merging them in a final synthesis pass.",
				Nodes: []WorkflowNode{
					{
						ID:         "start",
						Type:       "user_input",
						Properties: map[string]interface{}{},
						Inputs:     []NodeConnection{},
					},
					{
						ID:   "axis_chronological",
						Type: "llm_query",
						Properties: map[string]interface{}{
							"provider":      "openai",
							"model":         "gpt-4o-mini",
							"temperature":   0.1,
							"system_prompt": "You are a chronological state tracking specialist. Analyze the query and history, tracking timelines, mutable states, and temporal continuity.",
						},
						Inputs: []NodeConnection{
							{SourceNode: "start", SourceOutput: "prompt", TargetInput: "prompt"},
						},
					},
					{
						ID:   "axis_causal_logical",
						Type: "llm_query",
						Properties: map[string]interface{}{
							"provider":      "openai",
							"model":         "gpt-4o-mini",
							"temperature":   0.1,
							"system_prompt": "You are a causal-logical constraint specialist. Obey rules, compile bounds, check for logical fallacies and conditional requirements.",
						},
						Inputs: []NodeConnection{
							{SourceNode: "start", SourceOutput: "prompt", TargetInput: "prompt"},
						},
					},
					{
						ID:   "axis_semantic_world",
						Type: "llm_query",
						Properties: map[string]interface{}{
							"provider":      "openai",
							"model":         "gpt-4o-mini",
							"temperature":   0.1,
							"system_prompt": "You are a spatial-ontological world specialist. Analyze directory hierarchies, visual layout environments, and ontological coordinates.",
						},
						Inputs: []NodeConnection{
							{SourceNode: "start", SourceOutput: "prompt", TargetInput: "prompt"},
						},
					},
					{
						ID:   "axis_behavioral_psych",
						Type: "llm_query",
						Properties: map[string]interface{}{
							"provider":      "openai",
							"model":         "gpt-4o-mini",
							"temperature":   0.1,
							"system_prompt": "You are a social-behavioral psychology specialist. Track characters motivations, parent/sub-agent objectives, secrets, and Theory of Mind.",
						},
						Inputs: []NodeConnection{
							{SourceNode: "start", SourceOutput: "prompt", TargetInput: "prompt"},
						},
					},
					{
						ID:   "axis_stylistic_prose",
						Type: "llm_query",
						Properties: map[string]interface{}{
							"provider":      "openai",
							"model":         "gpt-4o-mini",
							"temperature":   0.1,
							"system_prompt": "You are a stylistic-prose aesthetics specialist. Analyze dialogue dialect, grammar flow, syntax conventions, and output formatting.",
						},
						Inputs: []NodeConnection{
							{SourceNode: "start", SourceOutput: "prompt", TargetInput: "prompt"},
						},
					},
					{
						ID:   "aggregator",
						Type: "llm_synthesis",
						Properties: map[string]interface{}{
							"provider":      "openai",
							"model":         "gpt-4o",
							"temperature":   0.2,
							"system_prompt": "You are the master aggregator. Synthesize the five parallel cognitive reports (Chronological, Causal, Semantic, Behavioral, Stylistic) into a single, high-fidelity response answering the user prompt.",
						},
						Inputs: []NodeConnection{
							{SourceNode: "axis_chronological", SourceOutput: "response", TargetInput: "chronological_context"},
							{SourceNode: "axis_causal_logical", SourceOutput: "response", TargetInput: "causal_context"},
							{SourceNode: "axis_semantic_world", SourceOutput: "response", TargetInput: "semantic_context"},
							{SourceNode: "axis_behavioral_psych", SourceOutput: "response", TargetInput: "behavioral_context"},
							{SourceNode: "axis_stylistic_prose", SourceOutput: "response", TargetInput: "stylistic_context"},
							{SourceNode: "start", SourceOutput: "prompt", TargetInput: "raw_prompt"},
						},
					},
					{
						ID:         "terminal",
						Type:       "assistant_response",
						Properties: map[string]interface{}{},
						Inputs: []NodeConnection{
							{SourceNode: "aggregator", SourceOutput: "response", TargetInput: "final_output"},
						},
					},
				},
			},
		},
	}
}

// EnsureDefaultWorkflows writes the built-in default workflows.json next to the
// binary if it does not already exist. An existing file is never overwritten,
// so user customizations are always preserved. Safe to call on every startup.
func EnsureDefaultWorkflows() error {
	path := GetSystemPath("workflows.json")
	if _, err := os.Stat(path); err == nil {
		return nil // File already exists; leave user customizations untouched.
	}

	cfg := DefaultWorkflowConfig()
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return err
	}

	fmt.Printf("%s[SYSTEM] workflows.json not found. Seeded default workflows (linear_chat, enhanced_cognition).%s\n", ColorYellow, ColorReset)
	return nil
}

// ListWorkflows returns the active workflow id and the map of registered
// workflows, ensuring defaults are seeded first. Used by the /workflows slash
// command and the header dropdown.
func ListWorkflows() (active string, workflows map[string]Workflow, err error) {
	if err = EnsureDefaultWorkflows(); err != nil {
		return "", nil, err
	}
	cfg, err := LoadWorkflowConfig()
	if err != nil {
		return "", nil, err
	}
	return cfg.ActiveWorkflow, cfg.Workflows, nil
}

// ActivateWorkflow sets the named workflow as active in workflows.json after
// validating it exists. It is non-destructive: only the active_workflow field
// changes (the rest of the graph is preserved). On success it broadcasts an
// SSE hot-swap notification so all connected Web Consoles update instantly.
func ActivateWorkflow(id string) error {
	if id == "" {
		return fmt.Errorf("workflow id cannot be empty")
	}
	if err := EnsureDefaultWorkflows(); err != nil {
		return err
	}

	cfg, err := LoadWorkflowConfig()
	if err != nil {
		return err
	}
	wf, ok := cfg.Workflows[id]
	if !ok {
		return fmt.Errorf("workflow '%s' is not registered in workflows.json (use /workflows to list available)", id)
	}

	if cfg.ActiveWorkflow != id {
		cfg.ActiveWorkflow = id
		bytes, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(GetSystemPath("workflows.json"), bytes, 0644); err != nil {
			return err
		}
	}

	writeDebugLog("[WORKFLOW ENGINE] Active workflow set to '%s' (%s)", id, wf.Name)

	// Broadcast hot-swap to every connected Web Console (also doubles as the
	// confirmation card for /workflow issued from the browser chat input).
	BroadcastSSE("turn_secured", map[string]interface{}{
		"turn_number": 0,
		"role":        "system",
		"name":        "system",
		"content":     fmt.Sprintf("🔄 **[WORKFLOW ACTIVATED]** Conversation pipeline switched to **%s** (`%s`) dynamically!", wf.Name, id),
	})
	return nil
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
		SessionID:    activeSessionID,
		WorkflowID:   cfg.ActiveWorkflow,
		WorkflowName: wf.Name,
		RunID:        fmt.Sprintf("run_%d", time.Now().UnixNano()),
		TurnNumber:   currentTurnNumber + 1, // Final assistant turn this run produces
		Nodes:        make(map[string]*RuntimeNode),
		Timeout:      120 * time.Second, // 2-minute hard timeout boundary
	}

	for _, n := range wf.Nodes {
		executor.NodeOrder = append(executor.NodeOrder, n.ID)
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

	// Run execution loop. Execute() itself emits the workflow_start/node/end
	// live-trace SSE events.
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

	// 2. Announce the run so the Web Console can render a live trace card
	//    BEFORE the final assistant reply streams in.
	type nodeSummary struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Label string `json:"label"`
	}
	var traceNodes []nodeSummary
	for _, id := range e.NodeOrder {
		n := e.Nodes[id]
		if n.Type == "user_input" || n.Type == "assistant_response" {
			continue
		}
		label := ""
		if p, ok := n.Properties["provider_profile"].(string); ok && p != "" {
			label = "@" + p
		} else if model, ok := n.Properties["model"].(string); ok {
			label = model
		} else if tool, ok := n.Properties["tool_name"].(string); ok {
			label = tool
		} else if scope, ok := n.Properties["scope"].(string); ok {
			label = scope
		}
		traceNodes = append(traceNodes, nodeSummary{ID: n.ID, Type: n.Type, Label: label})
	}
	BroadcastSSE("workflow_start", map[string]interface{}{
		"run_id":      e.RunID,
		"workflow_id": e.WorkflowID,
		"name":        e.WorkflowName,
		"turn_number": e.TurnNumber,
		"nodes":       traceNodes,
	})

	// Defer a terminal event so failure paths (timeout/ctx cancel) are also reported.
	runErr := error(nil)
	defer func() {
		status := "completed"
		payload := map[string]interface{}{
			"run_id":      e.RunID,
			"workflow_id": e.WorkflowID,
			"status":      status,
		}
		if runErr != nil {
			payload["status"] = "failed"
			payload["error"] = runErr.Error()
		}
		BroadcastSSE("workflow_end", payload)
	}()

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
			startTime := time.Now()
			n.mu.Unlock()

			// Announce that this node has started (parallel nodes light up live).
			e.broadcastNode(n, "running", startTime, 0, "")

			// Resolve actual input data from upstream outputs
			inputPayload := e.resolveIncomingData(n)

			// Execute node logic based on type
			err := e.runNodeLogic(ctx, n, inputPayload)

			duration := time.Since(startTime).Milliseconds()

			n.mu.Lock()
			if err != nil {
				n.State = StateFailed
				n.Error = err
			} else {
				n.State = StateCompleted
			}
			n.mu.Unlock()

			// Announce completion/failure with a preview of the node's output.
			status := "completed"
			errMsg := ""
			if err != nil {
				status = "failed"
				errMsg = err.Error()
			}
			e.broadcastNode(n, status, startTime, duration, errMsg)

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
		runErr = fmt.Errorf("workflow execution failed: %v", terminal.Error)
		return runErr
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

// nodePreview extracts a short, human-readable preview of a node's main output.
// Preview length is capped so parallel node reports never overwhelm the UI.
func (e *WorkflowExecutor) nodePreview(n *RuntimeNode) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var raw string
	switch n.Type {
	case "llm_query", "llm_synthesis":
		if v, ok := n.Outputs["response"].(string); ok {
			raw = v
		}
	case "bm25_search":
		if v, ok := n.Outputs["search_results"].(string); ok {
			raw = v
		}
	case "tool_execution":
		if v, ok := n.Outputs["stdout"].(string); ok {
			raw = v
		}
	case "conditional_router":
		if v, ok := n.Outputs["route_branch"].(string); ok {
			return "→ " + v
		}
	case "assistant_response":
		if v, ok := n.Outputs["final_output"].(string); ok {
			raw = v
		}
	}

	const maxPreview = 600
	if len(raw) > maxPreview {
		return raw[:maxPreview] + "\n… [truncated]"
	}
	return raw
}

// broadcastNode pushes a node lifecycle event (running/completed/failed) to
// all connected Web Consoles. These events render the live parallel trace.
func (e *WorkflowExecutor) broadcastNode(n *RuntimeNode, status string, startTime time.Time, durationMs int64, errMsg string) {
	// Anchors are part of the graph but not interesting as execution steps.
	if n.Type == "user_input" || n.Type == "assistant_response" {
		return
	}

	label := ""
	if p, ok := n.Properties["provider_profile"].(string); ok && p != "" {
		label = "@" + p
	} else if model, ok := n.Properties["model"].(string); ok {
		label = model
	} else if tool, ok := n.Properties["tool_name"].(string); ok {
		label = tool
	} else if scope, ok := n.Properties["scope"].(string); ok {
		label = scope
	}

	payload := map[string]interface{}{
		"run_id":     e.RunID,
		"node_id":    n.ID,
		"type":       n.Type,
		"label":      label,
		"status":     status,
		"duration_ms": durationMs,
	}
	if status == "running" {
		payload["started_at"] = startTime.Format(time.RFC3339)
	} else {
		payload["preview"] = e.nodePreview(n)
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	BroadcastSSE("workflow_node", payload)
}

func (e *WorkflowExecutor) runNodeLogic(ctx context.Context, n *RuntimeNode, inputs map[string]interface{}) error {
	switch n.Type {
	case "user_input":
		return nil

	case "llm_query", "llm_synthesis":
		// Resolve the connection: a named provider_profile from providers.json
		// wins, with any inline provider/model/temperature overrides layered on
		// top. Falls back to the legacy inline fields for backward compatibility.
		parentAPI := activeConfig.API
		var nodeAPI APIConfig

		if profileName, ok := n.Properties["provider_profile"].(string); ok && profileName != "" {
			profile, err := GetProvider(profileName)
			if err != nil {
				return fmt.Errorf("node %s: %w", n.ID, err)
			}
			nodeAPI = mergeProfileWithFills(profile, n.Properties, parentAPI)
			writeDebugLog("[WORKFLOW NODE %s] Using provider profile '%s' (%s/%s)", n.ID, profileName, nodeAPI.Provider, nodeAPI.Model)
		} else {
			provider, _ := n.Properties["provider"].(string)
			model, _ := n.Properties["model"].(string)
			tempVal, _ := n.Properties["temperature"]
			temperature := 0.2
			if t, ok := tempVal.(float64); ok {
				temperature = t
			}
			nodeAPI = APIConfig{
				Provider:    provider,
				Key:         parentAPI.Key, // Falls back to global API Key
				BaseURL:     "",            // Auto built by backend
				Model:       model,
				Temperature: temperature,
				MaxTokens:   parentAPI.MaxTokens,
			}
			if provider == "openai" && nodeAPI.Key == "" {
				nodeAPI.Key = parentAPI.Key
			}
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

		writeDebugLog("[WORKFLOW NODE %s] LLM query requested: provider: %s, model: %s", n.ID, nodeAPI.Provider, nodeAPI.Model)

		// Temporarily hot-swap global API settings for this node execution.
		activeConfig.API = nodeAPI
		reqMessages := []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: promptText},
		}

		respMsg, err := SendMultiProviderRequest(reqMessages, nil)

		// Restore parent global settings immediately on return.
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
