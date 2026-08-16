# Visual Workflow Editor

The **AI Workflow Lab** (Settings → AI Workflow Lab) is a visual node-graph
editor for authoring `workflows.json` without hand-editing JSON.

## Concepts

A workflow is a DAG of **typed nodes**:

| Type | Purpose | Inputs | Outputs |
|------|---------|--------|---------|
| `user_input` | Start anchor (exactly one) | — | `prompt` |
| `assistant_response` | Terminal anchor (exactly one) | `final_output` | — |
| `llm` | An LLM call (specialist or aggregator) | one or more named inputs | `response` |
| `bm25_search` | Keyword search over workspace/session | `query` | `search_results` |
| `tool_execution` | Run a tool (command/read/write/patch) | `arguments` | `stdout`, `exit_code` |
| `conditional_router` | Route a branch | `eval_var` | `route_branch` |

`llm_query` and `llm_synthesis` are legacy aliases; the editor normalizes them
to `llm`. An `llm` node can have any number of **named input ports** (managed
in the inspector); each connected input becomes a labeled section in the
assembled prompt. The aggregation node in the `enhanced_cognition` workflow
uses five named context inputs plus `raw_prompt`.

### Connections

* Edges always run from an **output pin** (right, green) to an **input pin**
  (left, blue).
* Each input port accepts **one** incoming edge; dropping a new edge on an
  occupied port replaces the old connection.
* Self-loops and cycles are rejected.
* Edges can be clicked to select, then deleted with `Delete`/`Backspace`.

### Positions

Node coordinates are stored in `properties.x` / `properties.y` and round-trip
through `workflows.json`. Nodes without coordinates are auto-arranged by a
topological layered layout on load.

## Connections & provider profiles

Each `llm`/`bm25_search`/etc. node can reference a reusable connection
profile via `properties.provider_profile` (a name from `providers.json`). When
set, the node executes with that profile's provider/model/key. When empty, it
uses inline `provider`/`model`/`temperature` properties. The inspector's
**Connection Profile** dropdown lists all profiles; choosing one hides the
inline model/temperature fields.

> If a node references a profile name that does not exist in `providers.json`,
> the workflow run returns an error. Either create the profile or clear the
> reference.

## Editing

* **Drag a node** by its header to move it.
* **Drag from an output pin** to an input pin to connect.
* **Click a node** to edit it in the Inspector (right panel): rename, set
  profile/model/temperature/system prompt, add/remove LLM inputs, or delete it.
* **Add node** (toolbar) inserts one of the four user-creatable types and
  opens the inspector.
* **Auto-layout** re-runs the layered layout.
* **Reload JSON** re-parses the Advanced/JSON textarea into the graph.
* **Compile & Apply** validates the graph (anchors, required inputs, cycles,
  reachability), writes `workflows.json`, and hot-swaps the active workflow.

## Validation (before save)

Hard errors block saving:

* exactly one `user_input` and one `assistant_response`,
* every `llm` node has at least one incoming edge,
* the response node's `final` input is connected,
* no cycles,
* no dangling edge references,
* the response node is reachable from start.

Unreachable non-anchor nodes produce **warnings** (shown in the footer) but do
not block saving.

## JSON model

```jsonc
{
  "active_workflow": "linear_chat",
  "workflows": {
    "linear_chat": {
      "name": "Standard Linear Chat",
      "description": "...",
      "nodes": [
        { "id": "start", "type": "user_input",
          "properties": { "x": 40, "y": 60 }, "inputs": [] },
        { "id": "query_node", "type": "llm",
          "properties": { "x": 310, "y": 60, "model": "gpt-4o",
                          "temperature": 0, "system_prompt": "..." },
          "inputs": [
            { "source_node": "start", "source_output": "prompt",
              "target_input": "prompt" }
          ] },
        { "id": "terminal", "type": "assistant_response",
          "properties": { "x": 580, "y": 60 },
          "inputs": [
            { "source_node": "query_node", "source_output": "response",
              "target_input": "final_output" }
          ] }
      ]
    }
  }
}
```

The Advanced/JSON panel is an escape hatch: edits there are parsed on
**Reload JSON** (or on save), and changes made on the canvas are written back
to it automatically.

## Adding a new node type (developers)

1. Add an entry to `NODE_DEFS` in `src/web/index.html` with `title`, `icon`,
   `color`, `inputs`, `outputs`, `defaults`, and `fields`.
2. Add a `case` in `runNodeLogic` in `src/workflow.go` implementing execution,
   populating `n.Outputs[...]`.
3. Add an inspector form branch in `renderInspector()` for any type-specific
   fields, and a label branch in `nodePreview()` (workflow.go) for the live
   trace card.
