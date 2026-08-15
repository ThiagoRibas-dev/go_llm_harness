# 🔬 Technical Specification: GoHarness v2.0 Node-Based Workflow Orchestration Engine

This document defines the complete system architecture, data structures, execution lifecycles, and configuration schemas of the **GoHarness v2.0 Dynamic Node-Based Workflow Engine**. 

GoHarness 2.0 evolves from a hardcoded linear agent loop into a fully generalizable, high-performance, and lightweight **Directed Acyclic Graph (DAG) Execution Runtime** written in pure Go (standard library only). It allows developers and users to architect, visualize, and hot-swap complex reasoning pipelines—spanning parallel cognitive axes, recursive tool loops, and multi-agent synthesis—without re-compiling the system.

---

## 🎨 1. Core Architectural Overview

In GoHarness 2.0, every conversation run is modeled as a **Directed Acyclic Graph (DAG)** of execution nodes.

```
┌─────────────────┐
│ User Input Node │ (Start Anchor)
└────────┬────────┘
         │
         ▼ [raw_prompt]
┌────────────────────────┐
├─► Causal Axis LLM      ├─┐ [causal_report]
├────────────────────────┤ │
├─► Semantic Axis LLM    ├─┼─► ┌───────────────────┐     ┌────────────────────────┐
├────────────────────────┤ │   │ Aggregator Node   ├────►│ Assistant Response Node│
├─► Chronological LLM   ├─┘   │ (LLM Synthesis)   │     │ (Terminal Anchor)      │
└────────────────────────┘     └───────────────────┘     └────────────────────────┘
  (Parallel Cached Processing)
```

### 1.1 The Base Anchors
Every pipeline graph, regardless of its complexity or shape, is bounded by two un-deletable system anchors:
1. **`User Input Node` (Start Anchor):** Automatically instantiated as the starting node of the DAG. It captures the user's raw prompt, uploaded files, and session state on execution.
2. **`Assistant Response Node` (Terminal Anchor):** Automatically instantiated as the terminal node of the DAG. It captures the final consolidated output and streams it back to the client Web UI over SSE.

---

## ⚙️ 2. Pipeline Configuration Schema (`workflows.json`)

To make pipelines incredibly easy to write, edit, and share, GoHarness 2.0 represents them as a structured **JSON or YAML** array of node descriptors inside a dedicated `workflows.json` configuration index. It defines two out-of-the-box default workflows:

### 2.1 The Schema Blueprint

```json
{
  "active_workflow": "linear_chat",
  "workflows": {
    "linear_chat": {
      "name": "Standard Linear Chat",
      "description": "Standard conversational agent loop mapping user input to a single, high-fidelity LLM response.",
      "nodes": [
        {
          "id": "start",
          "type": "user_input",
          "properties": {},
          "inputs": []
        },
        {
          "id": "query_node",
          "type": "llm_query",
          "properties": {
            "provider": "openai",
            "model": "gpt-4o",
            "temperature": 0.0,
            "system_prompt": "You are a highly capable agent with access to a local terminal sandbox. Use your tools to solve the user's request."
          },
          "inputs": [
            { "source_node": "start", "source_output": "prompt", "target_input": "prompt" }
          ]
        },
        {
          "id": "terminal",
          "type": "assistant_response",
          "properties": {},
          "inputs": [
            { "source_node": "query_node", "source_output": "response", "target_input": "final_output" }
          ]
        }
      ]
    },
    "enhanced_cognition": {
      "name": "Enhanced Cognition (POADR)",
      "description": "Decomposes your query concurrently across 5 parallel cognitive axes to eliminate representational interference in smaller models, merging them in a final synthesis pass.",
      "nodes": [
        {
          "id": "start",
          "type": "user_input",
          "properties": {},
          "inputs": []
        },
        {
          "id": "axis_chronological",
          "type": "llm_query",
          "properties": {
            "provider": "openai",
            "model": "gpt-4o-mini",
            "temperature": 0.1,
            "system_prompt": "You are a chronological state tracking specialist. Analyze the query and history, tracking timelines, mutable states, and temporal continuity."
          },
          "inputs": [
            { "source_node": "start", "source_output": "prompt", "target_input": "prompt" }
          ]
        },
        {
          "id": "axis_causal_logical",
          "type": "llm_query",
          "properties": {
            "provider": "openai",
            "model": "gpt-4o-mini",
            "temperature": 0.1,
            "system_prompt": "You are a causal-logical constraint specialist. Obey rules, compile bounds, check for logical fallacies and conditional requirements."
          },
          "inputs": [
            { "source_node": "start", "source_output": "prompt", "target_input": "prompt" }
          ]
        },
        {
          "id": "axis_semantic_world",
          "type": "llm_query",
          "properties": {
            "provider": "openai",
            "model": "gpt-4o-mini",
            "temperature": 0.1,
            "system_prompt": "You are a spatial-ontological world specialist. Analyze directory hierarchies, visual layout environments, and ontological coordinates."
          },
          "inputs": [
            { "source_node": "start", "source_output": "prompt", "target_input": "prompt" }
          ]
        },
        {
          "id": "axis_behavioral_psych",
          "type": "llm_query",
          "properties": {
            "provider": "openai",
            "model": "gpt-4o-mini",
            "temperature": 0.1,
            "system_prompt": "You are a social-behavioral psychology specialist. Track characters motivations, parent/sub-agent objectives, secrets, and Theory of Mind."
          },
          "inputs": [
            { "source_node": "start", "source_output": "prompt", "target_input": "prompt" }
          ]
        },
        {
          "id": "axis_stylistic_prose",
          "type": "llm_query",
          "properties": {
            "provider": "openai",
            "model": "gpt-4o-mini",
            "temperature": 0.1,
            "system_prompt": "You are a stylistic-prose aesthetics specialist. Analyze dialogue dialect, grammar flow, syntax conventions, and output formatting."
          },
          "inputs": [
            { "source_node": "start", "source_output": "prompt", "target_input": "prompt" }
          ]
        },
        {
          "id": "aggregator",
          "type": "llm_synthesis",
          "properties": {
            "provider": "openai",
            "model": "gpt-4o",
            "temperature": 0.2,
            "system_prompt": "You are the master aggregator. Synthesize the five parallel cognitive reports (Chronological, Causal, Semantic, Behavioral, Stylistic) into a single, high-fidelity response answering the user prompt."
          },
          "inputs": [
            { "source_node": "axis_chronological", "source_output": "response", "target_input": "chronological_context" },
            { "source_node": "axis_causal_logical", "source_output": "response", "target_input": "causal_context" },
            { "source_node": "axis_semantic_world", "source_output": "response", "target_input": "semantic_context" },
            { "source_node": "axis_behavioral_psych", "source_output": "response", "target_input": "behavioral_context" },
            { "source_node": "axis_stylistic_prose", "source_output": "response", "target_input": "stylistic_context" },
            { "source_node": "start", "source_output": "prompt", "target_input": "raw_prompt" }
          ]
        },
        {
          "id": "terminal",
          "type": "assistant_response",
          "properties": {},
          "inputs": [
            { "source_node": "aggregator", "source_output": "response", "target_input": "final_output" }
          ]
        }
      ]
    }
  }
}
```

---

## 🚀 3. Node Specification & Types

GoHarness 2.0 features a rich collection of modular, single-responsibility node types:

### 3.1 `user_input` (Start Anchor)
* **Purpose:** Captures the starting payload of the execution cycle.
* **Outputs:**
  * `prompt` (string): The raw text submitted by the user.
  * `uploaded_files` (array of strings): Paths to session-specific uploaded reference files.

### 3.2 `llm_query` (Specialized LLM Task)
* **Purpose:** Dispatches an LLM request using custom credentials, prompts, and temperatures.
* **Properties:** `provider`, `key`, `base_url`, `model`, `temperature`, `system_prompt`.
* **Inputs:** `prompt` (string), `optional_system_override` (string).
* **Outputs:** `response` (string), `token_usage` (object).

### 3.3 `tool_execution` (Native Sandboxed Tool)
* **Purpose:** Runs a native GoHarness tool (`write_file`, `patch_file`, `execute_command`, `read_file`).
* **Properties:** `tool_name` (string), `sandbox_mode` (string).
* **Inputs:** `arguments` (JSON string).
* **Outputs:** `stdout` (string), `exit_code` (integer).

### 3.4 `bm25_search` (Lexical Candidate Discovery)
* **Purpose:** Queries our custom Go BM25 engine for rapid, cached files retrieval.
* **Properties:** `scope` (string), `limit` (integer).
* **Inputs:** `query` (string), `target_scan_dirs` (array of strings).
* **Outputs:** `search_results` (markdown formatted string).

### 3.5 `conditional_router` (Logical Branching)
* **Purpose:** Evaluates logical operations to branch execution dynamically based on upstream outputs.
* **Properties:** `conditions` (array of logical expressions, e.g. `exit_code != 0`).
* **Inputs:** `eval_variables` (object).
* **Outputs:** `route_branch` (string identifier determining which downstream path to wake).

---

## 🧠 4. The Go DAG Execution Engine (`src/workflow.go`)

The core Go runtime resolves, schedules, and executes nodes concurrently using **topological sorting**, **Go goroutines**, and **concurrency channels**.

```go
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkflowNodeState represents the execution lifecycle of a single node
type WorkflowNodeState string
const (
	StatePending   WorkflowNodeState = "pending"
	StateRunning   WorkflowNodeState = "running"
	StateCompleted WorkflowNodeState = "completed"
	StateFailed    WorkflowNodeState = "failed"
)

// PipelineNode represents a runtime instance of a node inside the DAG
type PipelineNode struct {
	ID         string
	Type       string
	Properties map[string]interface{}
	State      WorkflowNodeState
	Inputs     []NodeConnection
	Outputs    map[string]interface{}
	Error      error
	
	// Synchronization locks & channels
	mu         sync.RWMutex
	readyChan  chan struct{} // Closed when all upstream inputs are resolved
	doneChan   chan struct{} // Closed when this node completes execution
}

// NodeConnection defines a directed edge mapping data between two nodes
type NodeConnection struct {
	SourceNode   string `json:"source_node"`
	SourceOutput string `json:"source_output"`
	TargetInput  string `json:"target_input"`
}

// WorkflowExecutor parses, validates, and runs an entire DAG concurrently
type WorkflowExecutor struct {
	SessionID string
	Nodes     map[string]*PipelineNode
	Timeout   time.Duration
}

// Execute performs concurrent topological execution of the DAG
func (e *WorkflowExecutor) Execute(ctx context.Context, rawUserPrompt string) error {
	ctx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel = cancel

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
		go func(n *PipelineNode) {
			defer wg.Done()
			
			// Wait for upstreams to signal readiness
			select {
			case <-n.readyChan:
				// Proceed to execution
			case <-ctx.Done():
				n.State = StateFailed
				n.Error = ctx.Err()
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
					close(node.readyChan)
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
```

---

## 🎨 5. The Visual Web UI Node Graph Editor

GoHarness 2.0 embeds a beautiful, interactive **HTML5/Tailwind Node Graph canvas** directly inside the Web Console, eliminating raw text config manipulation.

```
┌────────────────────────────────────────────────────────┐
│ ⚙️ WORKFLOW: Parallel 5-Axis Reasoner       [Save] [Load]│
├────────────────────────────────────────────────────────┤
│  ┌──────────────┐     ┌──────────────┐                 │
│  │  User Input  │────►│  Causal LLM  │───┐             │
│  └──────────────┘     └──────────────┘   │             │
│                       ┌──────────────┐   ▼ ┌─────────┐ │
│                       │ Semantic LLM │────►│ Synthes │ │
│                       └──────────────┘     └─────────┘ │
└────────────────────────────────────────────────────────┘
```

### 5.1 Interactive Features:
1. **Draggable Nodes:** Nodes are rendered as absolute-positioned absolute Tailwind cards.
2. **Dynamic Edge Connections:** Clicking an output port on Node A and dragging it to an input port on Node B draws a beautiful SVG Bezuer connector line in real-time.
3. **Reactive Properties Inspector:** Clicking on any node opens an inspector panel on the right, dynamically morphing its input fields based on the selected node type (e.g. showing Project ID and Region *only* if a `vertex` provider is selected inside an `llm_query` node).
4. **Instant Validation:** The frontend automatically checks for **cycles** (circular dependencies) or **unconnected inputs** on save, preventing runtime execution hangs.

---

### 💬 6. Slash Command Pipeline Hot-Swaps

To make pipeline transitions instant, GoHarness 2.0 parses two new slash commands directly from your interactive chat input:

1. **`/workflows` (List Pipelines):**
   Renders a beautiful system summary card listing all registered pipelines from `workflows.json`, indicating which one is currently **active**.
2. **`/workflow <id>` (Switch Active Pipeline):**
   Instantly swaps the active workflow to `<id>` and hot-reloads the execution graph on-disk!
   * **The UX Feedback:** Streams a beautiful SSE success notification:
     `🔄 [WORKFLOW ACTIVATED] Conversation pipeline has been switched to 'Parallel 5-Axis Reasoner' (cognitive_axis_reasoning) dynamically!`

---

This specification establishes **GoHarness v2.0** as a highly-engineered, modular, and dynamic node-based orchestration hub, ready for cutting-edge development!
