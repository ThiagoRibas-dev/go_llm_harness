# ⚙️ Task & Background Work System — Execution Plan

**Row coverage map** — for each roadmap row this plan claims, which phase owns it and how deep that
coverage actually is. `gap` means the row is claimed but not yet written into the plan body.

| Roadmap row | Phase | Coverage |
|---|---|---|
| `10.2B` Wait First / Race | Phase E | **gap** — no body text yet |
| `11.2` Message queue while agent is running | Phase A | **core** |
| `11.3` Cancel vs abort distinction | Phase B (+ §5.3 state semantics) | **core** |
| `11.7` Durable / background sub-agent jobs | Phase C | **core** |
| `11.8` Named sub-agent roles & packaged workflows | Phase E | **reference** |
| `11.10` Automatic build/lint/typecheck feedback loop | Phase C | **gap** — the recurring validation loop is not yet written into the body |
| `11.11` Model-initiated Q&A interludes | Phase D | **core** |
| `11.12` Persistent TODO/plan overlay | Phase H | **partial** — projection only |
| `12.7` Plan mode as logged collaboration state | Phase F / Phase H boundary | **partial** — the logged collaboration state is not specified beyond its projection |
| `12.8` Goals with autonomous round-driving | Phase F | **core** |
| `12.9` Dynamic workflows over sub-agents | Phase E | **reference** |
| `12.10` Multiple sub-agent providers | Phase A / Phase C | **gap** — no body text yet |
| `12.11` Structured sub-agent output | Phase E (receipts) | **reference** |
| `12.12` Per-agent tool scoping & personas | Phase E | **gap** — no body text yet |
| `12.13` Per-session agent presets | Phase A | **gap** — no body text yet |
| `12.21` Scheduled / mission runs | Phase G | **core** |
| `13.6` Plan mode + goal mode + progress rows | Phase H | **partial** — projection only |
| `14.5` Subagents as data files | Phase E | **reference** |
| `14.8` Background agents, task roster, queued command semantics | Phase H | **reference** |

The next revision of this plan should fold the five `gap` rows into their phases: `12.10` and `12.13`
into Phase A (per-provider and per-session job configuration), `10.2B`, `11.10` and `12.12` into
Phase E (sub-agent work as a job family).

> **Status:** Execution plan
> **System:** Task & Background Work System
> **Primary roadmap rows:** `10.2B`, `11.2`, `11.3`, `11.7`, `11.8`, `11.10`, `11.11`, `11.12`, `12.7`, `12.8`, `12.9`, `12.10`, `12.11`, `12.12`, `12.13`, `12.21`, `13.6`, `14.5`, `14.8`

This document defines the implementation program for GoHarness's **queued work, background work, waiting states, and long-running autonomous task model**.

The core belief behind this system is:

> queueing, blocked-state recovery, background jobs, plans, goals, schedules, and sub-agent orchestration are not separate features — they are all projections over the same task/runtime state machine.

---

# 1. System scope

This system owns:

- queued user work,
- session-local pending work,
- background/durable agent jobs,
- blocked/waiting-for-user states,
- task receipts and outputs,
- future plan/goal/schedule runtime state,
- and UI rosters/progress projections over that state.

It does **not** own:

- shell chrome itself,
- archived memory retrieval,
- permission/hook policy,
- execution isolation mechanics,
- or MCP transport logic.

Those systems will consume task state, but should not each invent their own queue/job model.

---

# 2. Source inputs and what they contribute

## 2.1 GoHarness source already relevant

### `src/web/js/composer.js`
Current useful substrate:
- session-local queued steering/follow-up logic in browser state
- busy/blocked-state composer semantics
- `/compact`, `/workflow`, `/new`, `/settings` routing patterns

This proves the **interaction contract** users want, but not yet a durable backend queue.

### `src/subagent.go`
Current useful substrate:
- child session creation
- sub-agent lifecycle events
- depth limiting
- process-wide write lock for file mutations

This is the first slice of durable work delegation.

### `src/agent_runtime.go` + `src/agent_run.go`
Current useful substrate:
- explicit per-agent runtime identity
- retry/wait state around provider issues
- session persistence and root/sub-agent distinction

This gives us a strong base for moving from ephemeral queue semantics to host-owned queued work semantics.

### `src/workflow.go`
Current useful substrate:
- typed concurrent workflow execution
- node lifecycle states
- terminal/failure boundaries

This is relevant because background or scheduled work may later be expressed as workflows, but the task system must exist independently of the DAG engine.

---

## 2.2 Reference source inspection and lessons

## A. `Prime Agent` queue/admission/runtime model
Inspected sources:
- `packages/coding-agent/src/core/agent-session.ts`
- `packages/coding-agent/src/core/cron-jobs.ts`
- `packages/coding-agent/src/core/goals.ts`
- `packages/coding-agent/src/core/session-manager.ts`
- `packages/coding-agent/docs/daemon.md`
- `packages/coding-agent/test/suite/agent-session-queue.test.ts`

Important source-level lessons:

### 1. Queue/admission semantics are host-owned
The queue tests and `agent-session.ts` show a strong model where:
- prompts/actions are admitted deliberately,
- queued work is visible as structured state,
- pause/abort/restart semantics are explicit,
- and queued actions are not just "messages sitting around".

This is crucial for GoHarness: queueing should move from a **browser convenience** to a **backend-owned session action system**.

### 2. Scheduled jobs are persisted per session, not just globally
In `cron-jobs.ts`, Prime Agent persists scheduled jobs with:
- IDs,
- session ownership,
- status,
- next run time,
- run counts,
- delivery behavior,
- interruption recovery.

The key lesson is not to copy the exact implementation, but to copy the principle:

> scheduled/background work should be represented as durable records with explicit lifecycle, not as ad hoc timers.

### 3. Goal state is a real state machine
In `goals.ts`, goal state includes:
- active/paused/complete/error/budget-limited states,
- accounting,
- continuation counts,
- timestamps,
- and explicit host responses.

This is exactly the sort of thing that should not be implemented as a loose prompt convention in GoHarness.

### 4. Daemon architecture separates clients from durable workers
In `docs/daemon.md`, Prime Agent makes a strong architectural distinction between:
- clients,
- supervisor,
- workers,
- schedulers,
- session ownership.

GoHarness does not need to become a daemon tomorrow, but the lesson is important:

> durable work should not be modeled as a UI-only phenomenon.

## B. `Pi` / related execution philosophy
While Pi's top-level README is broader, the practical relevant takeaway is the same:
- session/runtime control and work continuation should be host-owned,
- not just inferred from chat turns.

## C. Existing GoHarness sub-agent plan
`docs/SUBAGENT_PARALLELISM_PLAN.md` already establishes:
- child runtime boundaries,
- recursion limits,
- structured sub-agent handling,
- and the importance of explicit lifecycle events.

This is the strongest internal bridge into a fuller task system.

---

# 3. Current GoHarness baseline and gap analysis

## 3.1 What already exists

GoHarness already has:
- browser-side queued steering/follow-up messages,
- root vs sub-agent runtime identity,
- child sessions,
- SSE lifecycle events,
- workflow concurrency,
- retry semantics around model/provider failures,
- basic blocked-state UI.

## 3.2 What is missing

It still lacks:
- a **backend-owned queue model**,
- durable task records,
- a unified state machine for waiting/running/cancelled/aborted work,
- job receipts,
- schedules/goals as host state,
- a truthful roster of background work,
- and a shared substrate for future plan/goal/async systems.

---

# 4. Program strategy

This system should be built in **three layers**.

## Layer 1 — backend session action queue
Move current queue semantics from browser-only state into backend-owned state.

## Layer 2 — durable jobs and waiting states
Represent asynchronous work and human-waiting states as explicit session-local records.

## Layer 3 — goals, schedules, and orchestration projections
Build plan/goal/schedule/fleet UI and operator surfaces on top of the same task substrate.

---

# 5. Core task model

## 5.1 Task identity

Every queued/background work item should have:

- task ID
- session ownership
- source kind
- state
- timestamps
- prompt/request payload
- result/error summary
- optional linkage to sub-agent/workflow/artifact outputs

### Proposed schema

```json
{
  "task_id": "task_01JABC...",
  "session_id": "sess_20260912-170452",
  "source": "queued_prompt",
  "kind": "turn",
  "state": "queued",
  "priority": "user",
  "created_at": "2026-09-12T17:05:00Z",
  "updated_at": "2026-09-12T17:05:00Z",
  "payload": {
    "prompt": "Fix the failing tests.",
    "queue_kind": "follow_up"
  },
  "result": null,
  "error": null
}
```

## 5.2 State machine

Minimum shared states:

- `queued`
- `admitted`
- `running`
- `waiting_user`
- `waiting_dependency`
- `completed`
- `failed`
- `cancelled`
- `aborted`

## 5.3 State semantics

### `cancelled`
User/system removed a task before completion and before it produced a terminal result.

### `aborted`
A currently running task was forcefully interrupted.

### `waiting_user`
Task cannot proceed until explicit human input arrives.

### `waiting_dependency`
Task is paused until some other task/sub-agent/workflow result completes.

This distinction matters because these are not just labels; they drive:
- UI affordances,
- retry behavior,
- scheduling,
- and future policy/hook decisions.

---

# 6. Execution phases

# Phase A — Introduce backend-owned session action queue

## Goal
Replace browser-only queue truth with backend-owned session action state.

## Deliverables

1. **Session action schema**
   - queued prompt/action representation
   - stable action IDs
   - delivery policy (`steer`, `follow_up`, etc.)

2. **Persistence model**
   - store queued actions in the session area
   - file-backed JSON is fine for v1

3. **Queue mutation API**
   - enqueue
   - list
   - remove
   - edit
   - clear
   - admit next

## Proposed new Go files
- `src/session_actions.go`
- `src/session_action_store.go`
- `src/session_action_api.go`

## GoHarness files likely touched
- `src/web.go`
- `src/agent_runtime.go`
- `src/agent_run.go`
- `src/web/js/composer.js`

## Why now
This is the minimal substrate needed so queue semantics are truthful outside the current browser tab.

---

# Phase B — Unify busy/blocked/waiting semantics in runtime

## Goal
Turn current ad hoc busy/blocked logic into a shared host state model.

## Deliverables

1. **Session runtime state struct**
   - running
   - blocked
   - waiting_user
   - waiting_dependency
   - retrying

2. **SSE/event emission contract**
   - queue updates
   - wait-state updates
   - completion/failure transitions

3. **UI config endpoint integration**
   - expose current truthful state to web shell

## Proposed new Go files
- `src/session_runtime_state.go`
- `src/session_runtime_events.go`

## Why now
It aligns the backend with what the composer/UI is already trying to express.

---

# Phase C — Durable background jobs

## Goal
Represent long-running async work as first-class durable jobs.

## Deliverables

1. **Job record model**
2. **Session-local job store**
3. **Crash/interruption recovery rules**
4. **Task receipts / result summary contract**

## Proposed new Go files
- `src/jobs.go`
- `src/job_store.go`
- `src/job_recovery.go`

## Design rules

- session-local first
- no daemon required in v1
- no hidden replays of uncertain side effects
- clear resumed/interrupted semantics

## Reference grounding
This phase is strongly informed by Prime Agent's `cron-jobs.ts` and daemon/session ownership model, but should be implemented in a GoHarness-native, lighter-weight way.

---

# Phase D — Waiting-user / ask-user runtime integration

## Goal
Make user questions and clarifications a task-system concern, not just a frontend prompt.

## Deliverables

1. **waiting_user task state**
2. **pending question record**
3. **resume semantics on answer**
4. **UI takeover/projection support**

## GoHarness files likely touched
- `src/web.go`
- `src/agent_run.go`
- frontend shell/composer modules

## Why now
This is the clean bridge from `11.11` into the broader task substrate.

---

# Phase E — Sub-agent work as durable job family

## Goal
Unify sub-agent execution with the durable task model.

## Deliverables

1. **parent task ↔ child task linkage**
2. **structured child result records**
3. **task-family metadata**
4. **roster/progress API**

## GoHarness files likely touched
- `src/subagent.go`
- `src/agent_runtime.go`
- `src/web.go`

## Why this matters
This is how `11.7`, `11.8`, `12.9`, `12.11`, and `14.5` stop being separate features and become one durable work graph.

---

# Phase F — Goal runtime

## Goal
Make goals a first-class host-owned runtime state instead of just prompt convention.

## Deliverables

1. **Goal state record**
2. **goal status transitions**
3. **budget/time accounting**
4. **continuation counter / stop conditions**
5. **goal-context prompt injection contract**

## Proposed new Go files
- `src/goals.go`
- `src/goal_accounting.go`
- `src/goal_runtime.go`

## Reference grounding
Prime Agent's `goals.ts` demonstrates why this needs to be a real state machine.
GoHarness should borrow the idea, not the TS implementation.

---

# Phase G — Schedule / mission jobs

## Goal
Build scheduled work on top of durable jobs rather than inventing a second async system.

## Deliverables

1. **session-local schedule records**
2. **next-run computation**
3. **claim-before-run semantics**
4. **interrupted-claim recovery**

## Proposed new Go files
- `src/schedules.go`
- `src/scheduler.go`

## Why this matters
This is where `12.21` and parts of `14.8` should land.

The system should not build scheduling before durable jobs exist.

---

# Phase H — Task roster, plan chips, and progress rows

## Goal
Expose the task system through truthful UI/operator projections.

## Deliverables

1. **task roster endpoint**
2. **active/pending counts**
3. **progress rows / chips**
4. **goal/plan/schedule badges**

## This phase is projection-only
Do not invent separate state here.
All data must come from the task/job/goal substrate.

---

# 7. Reference-project implementation lessons to apply directly

## 7.1 From Prime Agent queue semantics

Useful lesson:
- admission and queue state are host-owned
- queued work is structured, not merely chat text
- interrupts, pause, and clear are explicit lifecycle transitions

Apply in GoHarness:
- move queue truth into backend state
- keep frontend as a projection

## 7.2 From Prime Agent cron jobs

Useful lesson:
- scheduled jobs are session-local durable records
- due work is claimed and advanced before prompt delivery
- interrupted work is recoverable without replaying uncertain side effects

Apply in GoHarness:
- jobs/schedules should be file-backed state, not naked timers

## 7.3 From Prime Agent goals

Useful lesson:
- goals are not just a prompt string
- they have status, budget, timestamps, counters, and host responses

Apply in GoHarness:
- if we implement goal mode, it must be a real host-owned state machine

## 7.4 From GoHarness's own subagent runtime

Useful lesson:
- child sessions and explicit lifecycle events are already valuable
- the next step is not inventing a new background model, but making this durable and queryable

---

# 8. Concrete GoHarness implementation slices

## Slice 1 — Durable session queue

### Add
- persisted session action records
- queue mutation API
- backend admission rules

### Success criterion
A queued follow-up survives browser refresh and remains visible/truthful across clients.

---

## Slice 2 — Runtime waiting states

### Add
- waiting_user / waiting_dependency / retrying states
- SSE updates
- truthful UI exposure

### Success criterion
Blocked-state UI is backend-truthful, not merely inferred in JS.

---

## Slice 3 — Background sub-agent jobs

### Add
- sub-agent task records
- durable child receipts
- progress/result API

### Success criterion
Long-running sub-agent work can outlive the immediate foreground turn and still be inspectable.

---

## Slice 4 — Goal state machine

### Add
- goal records
- accounting
- completion/error/pause states

### Success criterion
Goal mode becomes host-owned and resumable instead of purely prompt-shaped behavior.

---

## Slice 5 — Scheduled jobs

### Add
- schedule persistence
- claim-before-run behavior
- interrupted claim recovery

### Success criterion
Scheduled prompts/jobs are durable and do not replay uncertain actions after interruption.

---

## Slice 6 — Task roster and progress projections

### Add
- task roster endpoint
- active / pending counts
- progress rows and chips consumed by the shell
- goal / plan / schedule badges

### Success criterion
The shell can render queued, running, waiting and finished work directly from task-system state,
without inventing its own representation.

---

# 9. Testing plan

## 9.1 Queue tests

- enqueue/edit/remove/clear
- browser refresh and reload persistence
- multiple clients observing same queue state
- ordering and priority semantics

## 9.2 Runtime state tests

- cancel vs abort
- busy vs waiting_user vs waiting_dependency
- retry transitions

## 9.3 Job tests

- durable background job lifecycle
- crash/interruption recovery
- no duplicate completion emission

## 9.4 Goal tests

- budget accounting
- pause/resume
- completion correctness
- error transitions

## 9.5 Schedule tests

- due calculation
- recurring/one-shot semantics
- interrupted claim cleanup

---

# 10. Risks and anti-goals

## Risks

1. implementing frontend queue polish without backend truth first
2. creating one runtime state model for goals and another for jobs
3. allowing scheduled/background work to mutate state without durable receipts
4. mixing policy concerns into the task substrate too early

## Anti-goals

1. do not build a daemon-only solution first
2. do not make the browser the source of truth for queued work
3. do not ship goal/schedule UI before the host state machine exists
4. do not let sub-agent durability fork into a separate architecture from the task system

---

# 11. Recommended first implementation order

1. **Slice 1** — Durable session queue
2. **Slice 2** — Runtime waiting states
3. **Slice 3** — Background sub-agent jobs
4. **Slice 4** — Goal state machine
5. **Slice 5** — Scheduled jobs
6. **Slice 6** — UI/operator roster projections

That order gives us visible value early while preserving one coherent system.

---

# 12. Success definition

This system is succeeding when GoHarness can truthfully say:

- queued work is durable,
- blocked work is classified correctly,
- background jobs are explicit and inspectable,
- goals and schedules are host-owned state machines,
- sub-agents fit into the same lifecycle,
- and the UI is only projecting runtime truth instead of inventing it.
