# Parallel Sub-Agents — Implementation Plan

> Goal: give the main agent a Claude Code–style fan-out: it can call `spawn_sub_agent`
> several times in one response, and those child agents run **concurrently** in
> isolated sessions with their own instructions, returning dense summaries that
> the parent collects before continuing.
>
> This document is a plan only. Nothing here is implemented yet.

---

## 1. Current state

We already have most of the machinery:

- `spawn_sub_agent` tool is declared (`src/tools.go:106`) and implemented as
  `executeSubAgent(prompt)` (`src/agent.go:604`).
- It creates an isolated session `<parent>_sub_agent_<nanos>`, writes a
  `meta.json` with `ParentSessionID`, swaps in that session, runs the full
  tool-capable `runAgentLoop`, restores the parent, and returns a
  `=== SUB-AGENT RESEARCH REPORT ===` block.
- Sessions already model parent/child (`SessionMeta.ParentSessionID`), used by
  fork/branch.
- The DAG executor already runs nodes concurrently and we added **per-profile
  semaphores** (`WorkflowExecutor.throttlers`, `acquireThrottle`) that cap
  concurrent LLM calls to a profile — directly reusable so N parallel
  sub-agents on a 2-RPM profile queue instead of 429ing.
- Tool calls/results now render as collapsed `<details>` in the chat UI, so a
  3-agent fan-out won't flood the transcript.

**What's missing (the actual work):**

1. **Concurrency-safety of the agent loop.** `runAgentLoop`, `executeToolCall`,
   `saveMessageTurn`, and most helpers read/write three **package globals**:
   - `activeSessionID` — which session folder turns are written to
   - `currentTurnNumber` — turn counter for filenames
   - `activeConfig` — the live `*Config` (including `.API`, which gets
     hot-swapped per DAG node and per compaction call)

   `executeSubAgent` works around #1/#2 today by *swapping the globals and
   running synchronously*. That is fundamentally incompatible with two
   sub-agents running at once (or a sub-agent running while the parent thread
   is mid-loop): they would write turns into each other's session folders and
   corrupt turn numbering.
2. **No parallel dispatch.** `executeToolCall` is a straight switch; when the
   model returns N tool calls in one response, `runAgentLoop` iterates them
   sequentially (`for _, toolCall := range ...`).
3. **Thin tool schema.** `spawn_sub_agent` takes only `{prompt}`. Claude Code's
   equivalent takes description/role, model override, and a tool subset.
4. **No progress UI.** A fan-out just looks like three long-running tool calls.
5. **No recursion guard.** A sub-agent can currently spawn sub-sub-agents
   unbounded (it calls the same `runAgentLoop`, which has the same tools).

There is also a **latent existing bug** this plan fixes as a prerequisite:
DAG nodes run in goroutines and `runLLMNode` does
`activeConfig.API = nodeAPI; defer restore`. Two nodes with different profiles
running at once race on `activeConfig.API` — the per-profile semaphore limits
*concurrency* but doesn't make that swap safe. Today it mostly works because
parallel nodes usually share a profile, but it is a data race. The refactor
below removes the pattern entirely.

---

## 2. Design principles

- **Explicit context, no shared mutable globals during a run.** Each agent
  invocation carries its session identity, turn counter, and API connection in
  a struct. Globals remain only for process-wide config and the active UI
  session.
- **Sub-agents are read-only by default.** They may search/read/run commands,
  but `write_file`/`patch_file` are opt-in. This is the safest default for
  parallel children (two writers to the same file would conflict) and matches
  Claude Code's "Explore" agents. Write-capable sub-agents run with an
  exclusive workspace lock so they can't overlap each other or the parent.
- **Fan-out/fan-in within one turn.** No background job/polling system in v1.
  The parent emits parallel `spawn_sub_agent` calls; the harness runs them
  concurrently and returns all results together. This covers the high-value
  cases ("research A, B, C in parallel") without a job queue.
- **Reuse what we built.** Per-profile throttling, isolated sessions,
  foldable tool results, and SSE traces all get reused, not rebuilt.
- **Backwards compatible.** The existing single-sub-agent behavior, the linear
  loop, and all current callers keep working; the signature changes are
  internal.

---

## 3. Phase 0 — Thread context refactor (prerequisite, no behavior change)

This is the hard/risky part and should land on its own, green, before any
concurrency is added.

### 3.1 Introduce an `AgentContext`

New file `src/agent_context.go`:

```go
// AgentContext carries per-run identity and connection state so the agent loop
// can run concurrently for multiple sessions without touching package globals.
type AgentContext struct {
    SessionID   string
    Turn        int                 // next turn number for this session
    Workspace   string              // workspace dir (usually activeConfig.Agent.WorkspaceDir)
    API         APIConfig           // the resolved connection this run uses
    ProfileName string              // profile that resolved API (for throttle keying)
    Recursion   int                 // sub-agent depth (0 = main/root agent)

    // Per-run (not global) concurrency throttle. The DAG executor already has
    // one; a standalone run builds its own from profiles on disk. Nil = no
    // limit is ever acquired (treated as unlimited).
    throttlers map[string]chan struct{}
    throttleMu sync.Mutex
}
```

Factory:

```go
func NewRootAgentContext() *AgentContext   // uses activeConfig + activeSessionID
func (c *AgentContext) Sub(sessID string) *AgentContext  // child inherits API/Workspace, Recursion+1
```

`Turn` is seeded from `findMaxTurnNumber(SessionID)` so a sub-agent (or a
resumed run) starts numbering correctly. The session folder is created before
incrementing.

### 3.2 Thread `ctx *AgentContext` through the call graph

Change these from reading globals to taking `ctx`:

| Function | File | Change |
|---|---|---|
| `saveMessageTurn(msg)` | `agent.go` | `saveMessageTurn(ctx, msg)` — writes to `ctx.SessionID`, uses `ctx.Turn++` |
| `loadHistoryFromFiles()` | `agent.go` | `loadHistoryFromFiles(ctx)` (or take `sessionID string`) |
| `runAgentLoop(prompt)` | `main.go` | `runAgentLoop(ctx, prompt) string` — new signature |
| `executeToolCall(turn, tc)` | `tools.go` | `executeToolCall(ctx, turn, tc) string` |
| `executeSubAgent(prompt)` | `agent.go` | replaced in Phase 1 |
| `executeBM25Search(q, scope, limit)` | `agent.go` | `executeBM25Search(ctx, ...)` (session-scoped "session" scope) |
| `getCompactedSummary()` | `agent.go` | takes `sessionID` |
| `backupWorkspaceFile` / `restoreWorkspaceBackups` | `main.go` | take `sessionID` + turn (these are writer/rollback bookkeeping) |
| `executeReadFile/WriteFile/PatchFile/TerminalCommand` | `main.go` | take workspace from `ctx.Workspace`; read-only ones need no session |

The LLM dispatch also takes the connection explicitly:

```go
// llm.go
func SendProviderRequest(api APIConfig, messages []Message, tools []Tool) (*Message, error)
```

`SendMultiProviderRequest(messages, tools)` stays as a **thin wrapper** that
passes `activeConfig.API` — used by compaction and any legacy call site. The
agent loop and DAG nodes call the new explicit version, which removes the
`activeConfig.API = nodeAPI; defer restore` hot-swap entirely (fixing the
latent DAG race from §1).

### 3.3 What stays global

- `activeConfig *Config` — treated as **read-only** during runs. It still
  supplies `Agent`, `Security`, `DirectoryScan`, `Compaction.*` settings. No
  code mutates it mid-run after this refactor (the DAG hot-swap is gone;
  compaction already swaps and restores locally, which we move to a local
  `api := activeConfig.ResolveCompactionConfig()`).
- `mcpToolsMap`, `activeMCPServers` — read-only after bootstrap.
- `sseClients`/`BroadcastSSE` — already mutex-protected.
- The root `activeSessionID`/`currentTurnNumber` — used only by the web/TUI
  entry points to build the root `AgentContext`; never touched inside a run.

### 3.4 Compaction under concurrency

`executeSlidingWindowCompaction` mutates files inside the session folder
(renames turns into an archive). That is only safe for one agent per session,
which is already true. It must take `ctx` and use the explicit
`ResolveCompactionConfig()` for its LLM call. **Sub-agents have auto-compaction
disabled** (Phase 2): they are short-lived research tasks and compaction's
file shuffling plus API hot-swap is needless risk. They still get the parent's
compacted summary injected as context if relevant, but they don't compact
themselves.

### 3.5 Exit criteria for Phase 0

- `go build ./...`, `go vet ./...`, all existing tests pass.
- `go test -race ./...` is clean **for the packages we touched** (some
  pre-existing filesystem tests may need the tolerant cleanup already added in
  `6b7a819`).
- Linear chat, TUI mode, the OpenAI gateway, reroll, branch/edit, and the DAG
  workflows all behave identically. The diff is almost entirely mechanical
  signature threading.
- No `activeSessionID =` / `currentTurnNumber =` assignments remain inside
  `runAgentLoop` or `executeToolCall` (grep gate in review).

---

## 4. Phase 1 — Concurrent `spawn_sub_agent`

### 4.1 New tool shape

`src/tools.go` schema for `spawn_sub_agent`:

```json
{
  "name": "spawn_sub_agent",
  "description": "Spawn an isolated sub-agent to perform a focused task (research, code analysis, search) and return a dense summary. Call this tool multiple times in one response to run sub-agents in parallel.",
  "parameters": {
    "type": "object",
    "properties": {
      "prompt":      { "type": "string", "description": "The precise task/question for the sub-agent." },
      "description": { "type": "string", "description": "Short label shown while the agent runs (e.g. 'research auth libs')." },
      "model":       { "type": "string", "description": "Optional provider profile name to use (defaults to the parent's connection)." },
      "tools":       {
        "type": "array",
        "items": { "type": "string", "enum": ["read","search","execute","write"] },
        "description": "Tool classes to grant. Defaults to read+search+execute. Add 'write' only if the agent must modify files."
      }
    },
    "required": ["prompt"]
  }
}
```

`description` drives the progress UI; `model` lets the parent fan out onto a
cheap profile (e.g. five `haiku` agents); `tools` gates capabilities.

### 4.2 Execution model

`executeToolCall` currently returns one string. To support parallel calls we
introduce a small plan/execute split:

```go
// tools.go
type toolCallResult struct {
    ID     int    // index in the response's tool_calls array
    Name   string
    Result string
}

// runToolCalls executes every tool call from one assistant response. Plain
// tools run sequentially; spawn_sub_agent calls run concurrently and the rest
// run after, preserving result order.
func (a *Agent) runToolCalls(ctx *AgentContext, calls []ToolCall) []toolCallResult
```

Algorithm:

1. Partition `calls` into `spawns` and `plain`.
2. For each spawn, build a child `ctx.Sub(sessID)` and an `agentSpec`
   (prompt, description, allowed tools, optional model profile).
3. Launch all spawns in goroutines:
   ```go
   var wg sync.WaitGroup
   results := make([]toolCallResult, len(calls))
   for i, spawn := range spawns {
       wg.Add(1)
       go func(i int, s spawnSpec) {
           defer wg.Done()
           results[i] = toolCallResult{ID: i, Name: "spawn_sub_agent",
               Result: a.runSubAgent(ctx, s)}  // builds child ctx, calls runAgentLoop
       }(i, spawn)
   }
   wg.Wait()
   ```
4. Then run `plain` tool calls sequentially in their original order (file
   writes/commands must stay deterministic and serialized — the parent's own
   writes shouldn't interleave with sub-agents; see §5).
5. Assemble the ordered `[]toolCallResult` and append all tool messages to the
   conversation. The model gets every result on the next turn.

The main loop changes from "`for _, tc := range response.ToolCalls { ... }`" to
"`results := a.runToolCalls(ctx, response.ToolCalls)`". Result ordering is
preserved by index, so the model can correlate call #2 with result #2.

Sub-agent LLM calls go through the child `ctx.API`, and each
`SendProviderRequest` call acquires the child's throttle bucket (keyed by
`ctx.ProfileName`). A root-level throttler map is created at
`NewRootAgentContext` time from the profiles on disk, so five children on a
profile with `max_concurrency: 2` run 2 at a time automatically.

### 4.3 Isolation and capabilities

- Each sub-agent gets its own `SessionID` with `ParentSessionID = ctx.SessionID`
  in `meta.json` (already how it works).
- It inherits the parent's **workspace** and the environment/system context.
- It does **not** inherit the parent's full transcript — that would blow the
  context budget and defeat isolation. It gets:
  - the environment block,
  - its task prompt as the user message,
  - optionally the parent's compacted summary (a short "current state"
    preamble), and
  - `AGENTS.md`/pinned instructions (it's a fresh session, so
    `LoadLocalInstructions` runs).
- Tool gating: `selectTools(allowed []string)` returns a subset of
  `builtInTools` + MCP tools. The default set excludes `write_file`,
  `patch_file`, and (for v1) `spawn_sub_agent` unless recursion is enabled.
- **Recursion guard:** child contexts start with `Recursion: ctx.Recursion+1`;
  if it exceeds `MaxSubAgentDepth = 2`, the `spawn_sub_agent` tool is omitted
  from the child's tool list (and returns a friendly error if somehow called).
  So: root → child → grandchild, no deeper. This is one line in
  `selectTools` and prevents exponential fan-out accidents.

### 4.4 Write safety / workspace lock

Two sub-agents (or a sub-agent and the parent) must not write the same file
concurrently. Introduce a process-wide `workspaceLock` in a new
`src/workspace_lock.go`:

```go
var workspaceGate struct {
    sync.Mutex                          // exclusive writer lock
    activeWriters int                   // ...
}
// WithWriteLock runs fn while holding the exclusive write lock. Read/search/
// execute_command don't acquire it. write_file/patch_file do.
func WithWriteLock(fn func())
```

Reads are lock-free; `write_file`/`patch_file` (parent or write-enabled
sub-agent) take the exclusive lock around the actual write + backup. This is
coarse but correct and the writes are fast. If we later want parallel writers
to disjoint files, we can upgrade to per-file locks, but v1 doesn't need it.

The parent agent's own turn always runs its plain (non-spawn) tool calls
**after** `wg.Wait()`, so even without the lock, parent writes happen before
or after children, never interleaved mid-turn. The lock protects against
two write-enabled children.

---

## 5. Phase 2 — UI: sub-agent progress card

Reuse the workflow-trace card pattern (`renderWorkflowTrace`/`updateWorkflowNode`)
rather than inventing something new.

When the model returns ≥1 `spawn_sub_agent` call:

1. The assistant turn renders the usual text + a new **"Sub-agents" card**
   above the (already folded) tool results:
   ```
   ⚡ Sub-agents (3)                running
     ● research-auth-libs     running…
     ○ audit-crypto-impl      queued
     ○ find-test-gaps         queued
   ```
2. The backend emits new SSE events from `runToolCalls`:
   - `subagent_start` `{parent_turn, index, description, session_id}`
   - `subagent_update` `{index, status: "running"|"done"|"failed", duration_ms?}`
   - (We could stream partial child output later; v1 just reports state.)
3. As each goroutine finishes, its row flips to ✓ (with duration) or ✗ with
   the error. When all are done the card header shows `completed`.
4. The individual `spawn_sub_agent` tool results still render as collapsed
   `<details>` (we already built this), so the full report is one click away
   per agent, and the card itself stays compact.

Event handlers in `handleIncomingEvent` mirror the `workflow_*` handlers.
This is ~80 lines of JS and no backend state machine beyond the SSE sends
inside the already-running goroutines.

---

## 6. Phase 3 — Tests

### 6.1 Unit (no network)

- `AgentContext` turn numbering: two contexts writing to different sessions
  interleave calls and produce correctly numbered, non-overlapping files.
- `runToolCalls` ordering: with a fake tool that records an index, results
  come back in call order even when goroutines finish out of order.
- Concurrency cap: spin up N child contexts against a profile with
  `max_concurrency: 2`; assert no more than 2 are inside the simulated LLM
  call at once (channel/counter test, same shape as
  `TestAcquireThrottleCapsConcurrency`).
- Recursion guard: a child at `Recursion == MaxSubAgentDepth` has no
  `spawn_sub_agent` in its tool list.
- Tool gating: `selectTools(["read"])` excludes write/spawn tools.
- `WithWriteLock`: a writer blocks a second writer; readers don't block each
  other.

### 6.2 Race detector

```
go test -race ./...
```
is a hard gate for the Phase 0 + Phase 1 merge. The whole point is to remove
the global-stomping races, so CI should run `-race` for these packages.

### 6.3 Integration

- A fake/model-stub `SendProviderRequest` that echoes a script (turn 1 returns
  two `spawn_sub_agent` calls; the stubbed child returns a canned summary)
  drives `runAgentLoop` end-to-end; assert both child session folders exist,
  both summaries appear in the parent's next-turn messages, and the parent
  session has no stray child turns.
- Cross-platform build (linux/windows/darwin) stays green — the new code is
  pure Go concurrency, no OS-specific bits.

---

## 7. File-by-file change summary

**New**
- `src/agent_context.go` — `AgentContext`, factories, throttle helper.
- `src/workspace_lock.go` — `WithWriteLock`.
- `src/agent_context_test.go`, `src/subagent_dispatch_test.go`.

**Edited**
- `src/main.go` — `runAgentLoop(ctx, prompt)`; no global swaps; uses
  `ctx.API`/`ctx.SessionID`; calls `a.runToolCalls`.
- `src/agent.go` — `saveMessageTurn(ctx, ...)`, `loadHistoryFromFiles(ctx)`,
  `executeBM25Search(ctx, ...)`, compaction takes ctx; `executeSubAgent`
  becomes `(a *Agent) runSubAgent(parentCtx, spec)` and no longer touches
  globals.
- `src/tools.go` — `executeToolCall(ctx, ...)`; richer `spawn_sub_agent`
  schema; `selectTools`; `runToolCalls` scheduler.
- `src/llm.go` — add `SendProviderRequest(api, ...)`; keep wrapper.
- `src/workflow.go` — `runLLMNode` builds an `AgentContext` per node and calls
  the explicit-API request (removes the `activeConfig.API` hot-swap, fixing
  the latent race); reuses `acquireThrottle`.
- `src/web.go` — entry points (`/api/prompt`, reroll, branch, gateway) build a
  root `AgentContext` and pass it through.
- `src/web/index.html` — sub-agent progress card + SSE handlers; folded tool
  results already in place.
- `docs/` — this plan + a short user-facing note in the editor/tools docs.

**Unchanged**
- Provider clients (`openai.go`/`anthropic.go`/`gemini.go`) — they already take
  an `APIConfig`? If not, they're called only from `SendProviderRequest` so
  the change is localized.
- MCP server lifecycle, BM25 engine, snapshot/session storage, sandbox.

---

## 8. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Global-removal refactor introduces regressions | Land Phase 0 alone, full test suite + `-race`, behavior-preserving; grep gate for global writes in the loop |
| Sub-agents overwhelm a provider / 429 | Reuse per-profile `max_concurrency` semaphores; children share the parent's throttle map |
| Parallel writers corrupt files | Read-only default for sub-agents; `WithWriteLock` for write-enabled children; parent writes only after join |
| Context-budget blowup from N full reports | Each child returns a *summary* (already the pattern); parent gets concise blocks; reports are folded in UI; consider a result `max_tokens` cap on children |
| Unbounded recursion | `MaxSubAgentDepth = 2`, spawn tool stripped beyond it |
| Cost runaway from many children | The model chooses fan-out (same as Claude Code); we surface a clear "Spawning N sub-agents" card; future: a global max-children-per-turn cap (e.g. 6) |
| Compaction file shuffling races | Sub-agents don't auto-compact; compaction stays single-agent per session |
| SSE event ordering across goroutines | Per-card updates are keyed by stable `index`; `BroadcastSSE` is already mutex-protected; UI updates are idempotent |

---

## 9. Out of scope for v1 (future)

- **Background/persistent jobs** with a job ID you poll later (Claude Code's
  long-running tasks). Fan-out/fan-in within a turn covers most value with a
  fraction of the state machine.
- A dedicated **"sub-agent" DAG node type**. Phase 1 lets any tool-enabled LLM
  node fan out internally; a first-class graph node is a v2 design.
- **Streaming child output** into the card (v1 shows running/done; full report
  is one click away).
- **Per-file write locks** for parallel writers (coarse lock is fine for v1).
- **Sub-agent result caching/dedup** across identical tasks.
- Letting the user configure `max_concurrency`, depth, and default tools in the
  Settings UI (sensible hard-coded defaults for v1; the profile concurrency
  field already exists).

---

## 10. Suggested sequencing / commit plan

1. **`refactor(agent): thread AgentContext through loop and tools`** — Phase 0.
   Pure mechanical change, no behavior change, tests + `-race` green.
2. **`fix(workflow): pass API config explicitly instead of hot-swapping global`**
   — folded into #1; removes the latent DAG race.
3. **`feat(subagent): concurrent spawn_sub_agent with tool gating and recursion guard`**
   — Phase 1 (backend), no UI yet; parallel results verified by tests.
4. **`feat(subagent): live progress card in web console`** — Phase 2.
5. **`test(subagent): race detector and integration coverage`** — Phase 3
   (tests can land alongside 3–4, but call it out as its own gate).

Each step is independently shippable and revertible. Phase 0 is the only one
that touches a lot of files; the feature commits on top are small and focused.
