# 🔬 GoHarness v2.0 Verification Audit: Workflow Storage, Switching & Inspection

**Date:** 2026-08-15
**Scope:** Verify three UX questions against the actual code (not the spec):
1. Where do workflows live, and are the two defaults auto-created?
2. Is there an easy UI control to switch workflows on the fly?
3. Is the inspection UI (intermediate node output + metrics) easily accessible?

**Method:** Direct read of `src/workflow.go`, `src/main.go`, `src/web.go`, `src/web/index.html`, `src/llm.go`, `src/agent.go`, and `workflows.json`. Findings cite file and line.

---

## 📊 Executive Summary

| # | Question | Status | Notes |
|---|----------|--------|-------|
| 1 | Workflows live in a config file? | 🟢 **Yes** | Separate `workflows.json` (resolved next to the binary), distinct from `config.json`. |
| 1 | Two defaults auto-created if missing? | 🟢 **Yes (fixed)** | `EnsureDefaultWorkflows()` seeds both defaults at startup if `workflows.json` is absent (non-destructive), and `GET /workflows.json` serves/seed the full defaults instead of the empty stub. |
| 2 | Easy select box to switch on the fly? | 🟢 **Yes (fixed)** | A header **Workflow** `<select>` lists every workflow from `GET /workflows.json` and switches via `POST /api/workflows/activate`; the active name shows in the header. `/workflows` and `/workflow <id>` work in both the web chat input and the TUI. |
| 3 | Inspection UI easily accessible? | 🟢 **Yes (fixed)** | A **live workflow trace card** appears in the chat stream *before* the final reply when a DAG workflow runs, showing each intermediate node light up in real time (queued → running → done/failed) with latency and an expandable output preview. Per-node lifecycle is also still logged to `.goharness/debug.log`. |

---

## 1️⃣ Where Do Workflows Live? Are Defaults Auto-Created?

### What the code does

* **Storage file:** `workflows.json`, resolved through `GetSystemPath("workflows.json")`, i.e. **next to the running binary executable** — same rule as `config.json` (`src/config.go:176`).
  * Loader: `LoadWorkflowConfig()` in `src/workflow.go:69-81` → `os.ReadFile(GetSystemPath("workflows.json"))`.
  * HTTP read: `GET /workflows.json` in `src/web.go:1336`.
  * HTTP write: `POST /api/workflows/save` in `src/web.go:1350` → writes the posted JSON verbatim to `GetSystemPath("workflows.json")`.
* **It is a separate file from `config.json`.** `config.json` holds API/agent/security/compaction settings; `workflows.json` holds the DAG definitions and the `active_workflow` pointer. This is a clean separation and matches the spec.
* **Shipping state:** the repository **does** include a complete top-level `workflows.json` containing both `linear_chat` and `enhanced_cognition` (verified, 141 lines). So a normal clone already has both defaults.

### Auto-creation of defaults (implemented ✅)

A single source of truth, `DefaultWorkflowConfig()` in `src/workflow.go`, returns the built-in set of both workflows. It is used in two places:

* **Startup bootstrap:** `EnsureDefaultWorkflows()` is called right after `config.json` is seeded in `src/main.go`. If `workflows.json` does not exist next to the binary, it writes the full default set (indented JSON) and logs `[SYSTEM] workflows.json not found. Seeded default workflows...`. An existing file is **never overwritten**, so user customizations are preserved.
* **HTTP fallback:** `GET /workflows.json` in `src/web.go` calls `EnsureDefaultWorkflows()` and serves the full `DefaultWorkflowConfig()` (both `linear_chat` and `enhanced_cognition`, with all nodes) when the file is missing, instead of the previous empty `{"active_workflow":"linear_chat"}` stub. The handler also re-seeds the file to disk so it persists.

> **Verdict:** Storage is correct, well-separated, and now self-healing. A missing `workflows.json` is transparently restored with both built-in defaults at startup and via the HTTP endpoint.

### ✅ Verification steps

```bash
# 1. Confirm the shipped file has both defaults
python3 -c "import json;d=json.load(open('workflows.json'));print(list(d['workflows']))"
# Expect: ['linear_chat', 'enhanced_cognition']

# 2. Simulate a missing file (run from a temp dir with the binary copied there)
mkdir -p /tmp/empty && cp bin/agent_linux /tmp/empty/ && cd /tmp/empty
./agent_linux -web &        # no workflows.json next to binary
# Expect startup log: [SYSTEM] workflows.json not found. Seeded default workflows...
curl -s http://localhost:8080/workflows.json | python3 -c "import sys,json;d=json.load(sys.stdin);print(list(d['workflows']))"
# Expect: ['enhanced_cognition', 'linear_chat']   <-- full defaults, not an empty stub
```

---

## 2️⃣ Is There an Easy Select Box to Switch Workflows On the Fly?

### Implemented ✅

* **Header workflow dropdown.** A **Workflow** `<select>` in the top bar (`#workflow-selector`, next to the Model card) is populated from `GET /workflows.json` via `loadWorkflowSelector()` and shows the active workflow by name. Selecting an entry calls `switchWorkflow(id)` → `POST /api/workflows/activate`. The selector also refreshes after **Compile & Apply** in the lab, and the active workflow is visible at a glance without opening settings.
* **Dedicated activation endpoint.** `POST /api/workflows/activate` (`src/web.go`) validates that the id exists (returns HTTP 400 with a clear message if not), rewrites **only** the `active_workflow` pointer in `workflows.json` (graph definitions are left untouched), and broadcasts an SSE `WORKFLOW ACTIVATED` confirmation card.
* **Shared backend helpers.** `ListWorkflows()` and `ActivateWorkflow(id)` in `src/workflow.go` are reused by both the web endpoint and the TUI, ensuring one code path. `ActivateWorkflow` is non-destructive and logs to `.goharness/debug.log`.
* **Slash commands (web + TUI), matching `V2_SPECIFICATION.md` §6:**
  * **`/workflows`** lists every registered pipeline, marks the active one with `▶`, and shows descriptions.
  * **`/workflow <id>`** hot-swaps the active pipeline. In the web chat input it calls the activation endpoint and the backend streams the confirmation card; in the TUI it calls `ActivateWorkflow` directly. `/workflow` with no id prints usage.
* **Existing AI Workflow Lab path** remains: editing `active_workflow` in the JSON codeboard and clicking **Compile & Apply** still works via `POST /api/workflows/save`.
* The backend reads `active_workflow` fresh from disk on every prompt, so all switches take effect immediately with no restart.

> **Verdict:** Workflows can now be switched in one click from the header, via slash commands in either interface, or via the AI Workflow Lab. The active workflow is always visible.

### ✅ Verification steps

```bash
# Backend endpoint (bad id -> HTTP 400; good id -> persists and returns name)
curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/api/workflows/activate \
  -H 'Content-Type: application/json' -d '{"id":"nope"}'          # -> 400
curl -s -X POST http://localhost:8080/api/workflows/activate \
  -H 'Content-Type: application/json' -d '{"id":"enhanced_cognition"}'
python3 -c "import json;print(json.load(open('workflows.json'))['active_workflow'])"  # -> enhanced_cognition
```

TUI (each line is its own command; the input loop is now line-oriented via `bufio.Scanner`):
```
/workflows                         # lists pipelines, marks active
/workflow enhanced_cognition       # switches and confirms
/workflow nope                     # error: not registered
/workflow                          # usage hint
```

Web: type `/workflows` or `/workflow enhanced_cognition` into the chat input, or just pick from the **Workflow** dropdown in the header.

---

## 3️⃣ Is the Inspection UI Easily Accessible?

### Implemented ✅ — Live Workflow Trace Card

When a DAG workflow (e.g. `enhanced_cognition`) runs, the executor streams three new SSE events so the browser can show what the parallel branches are doing **before the final reply appears**:

| SSE event | Emitted when | Payload |
|-----------|--------------|---------|
| `workflow_start` | The run begins, before any node executes | `run_id`, `workflow_id`, `name`, `turn_number`, `nodes[]` (non-anchor nodes in declaration order, each with `id`/`type`/`label`) |
| `workflow_node` | A node transitions `running` → `completed`/`failed` | `run_id`, `node_id`, `type`, `label`, `status`, `duration_ms`, plus `started_at` (running) or `preview` (completed) / `error` (failed) |
| `workflow_end` | The run finishes (success **or** failure/timeout) | `run_id`, `workflow_id`, `status` (`completed`/`failed`), optional `error` |

**Backend (`src/workflow.go`):**
* `WorkflowExecutor` gained `RunID`, `WorkflowID`, `WorkflowName`, `TurnNumber`, and `NodeOrder` fields; `ExecuteActiveWorkflow` populates them.
* `Execute()` emits `workflow_start` up front, broadcasts a `workflow_node` event at each node's running/completed/failed transitions (real latency via `time.Since`, plus a truncated output `preview`), and `defer`s a `workflow_end` so failure/timeout paths are always reported.
* A new `nodePreview()` helper extracts the primary output per node type (`response`, `search_results`, `stdout`, `route_branch`), capped at 600 characters. The two anchors (`user_input`, `assistant_response`) are intentionally excluded from the trace.

**Frontend (`src/web/index.html`):**
* `handleIncomingEvent` routes the three new events to `renderWorkflowTrace`, `updateWorkflowNode`, and `finalizeWorkflowTrace`.
* `renderWorkflowTrace` inserts a dedicated **"Workflow: <name>"** card into the chat stream the moment a run starts — this is the live view of ongoing parallel/sub-agents that appears before the final assistant message.
* Each node row shows a status icon (queued circle → blue spinner → green check / red X), the node id, its model/tool label, and a live status line. Completed rows get a **show output** toggle that expands the preview inline.
* The card header shows a spinner while running and flips to a green "completed" or red "failed" badge on `workflow_end`.
* Node-type icons/colors are mapped in `WF_NODE_META` for `llm_query`, `llm_synthesis`, `bm25_search`, `tool_execution`, and `conditional_router`.

**Still in place:** the per-card **🔬 Inspect Execution Metrics** button (tokens in/out + USD from `cost_update` events), header aggregate token/cost counters, and high-verbosity per-node logging to `.goharness/debug.log` in debug mode.

> **Verdict:** Ongoing parallel branches and intermediate nodes are now visible in the UI as they execute, with latency and output previews, before the final reply is rendered. Server-side events are covered by an automated test (`src/workflow_trace_test.go`) asserting the start → node(running→completed) → end sequence.

### ✅ Verification steps

1. Switch to the POADR workflow: `/workflow enhanced_cognition` (or pick it from the header dropdown).
2. Send a prompt. A **"Workflow: Enhanced Cognition (POADR)"** card appears immediately.
3. Watch the five `axis_*` rows spin up in parallel and flip to ✅ with a **show output** link; then the `aggregator` row completes; finally the header badge turns green and the normal assistant reply card renders beneath.
4. Click **show output** on any axis to expand its specialist report.
5. Run the backend event test:
   ```bash
   go test ./src -run TestWorkflowLiveTraceEvents -v
   ```
6. For raw server-side detail, run with `-debug` and `tail -f .goharness/debug.log | grep WORKFLOW`.

---

## 🛠️ Recommended Fixes (attack these one at a time)

1. ~~**Bootstrap default workflows**~~ — ✅ **Done.** `DefaultWorkflowConfig()` + `EnsureDefaultWorkflows()` in `src/workflow.go`, called at startup (`src/main.go`) and from the `GET /workflows.json` handler (`src/web.go`). Non-destructive; existing files are never overwritten.
2. ~~**Workflow switcher dropdown**~~ — ✅ **Done.** Header `<select>` populated from `GET /workflows.json`, switching via `POST /api/workflows/activate`; active workflow shown by name. `/workflows` and `/workflow <id>` implemented in both the web chat input and the TUI (TUI input loop also upgraded from a one-shot `io.LimitReader.Read` to line-oriented `bufio.Scanner`).
3. ~~**Per-node inspection UI**~~ — ✅ **Done.** The executor emits `workflow_start` / `workflow_node` (running→completed/failed, with real latency and a truncated output preview) / `workflow_end` SSE events; the Web Console renders a live trace card in the chat stream *before* the final reply, showing parallel sub-agents/nodes light up and letting the user expand each node's output. Covered by `src/workflow_trace_test.go`.
4. **(Optional) Persist per-node traces** — write each node's inputs/outputs into the session folder (or `traces.jsonl`) so the inspection panel can replay them after a reload.

---

## 📁 Reference: Files Inspected

| File | Relevance |
|------|-----------|
| `workflows.json` | Shipped defaults (`linear_chat`, `enhanced_cognition`). |
| `src/workflow.go` | DAG loader, `DefaultWorkflowConfig()`/`EnsureDefaultWorkflows()`, `ListWorkflows()`, `ActivateWorkflow()`, executor, node runners, SSE broadcasts. |
| `src/main.go` | Config + workflow bootstrap, workflow routing gate, TUI `/fork`, `/workflows`, `/workflow` commands (line-oriented `bufio.Scanner`). |
| `src/web.go` | `GET /workflows.json` (full-default fallback), `POST /api/workflows/save`, `POST /api/workflows/activate`. |
| `src/web/index.html` | Header workflow `<select>`, web `/workflows` & `/workflow` parsing, AI Workflow Lab, metrics card, `cost_update` handler. |
| `src/llm.go` | `cost_update` SSE emission (`:72`). |
| `src/telemetry.go` | `traces.jsonl` + `writeDebugLog` to `.goharness/debug.log`. |
