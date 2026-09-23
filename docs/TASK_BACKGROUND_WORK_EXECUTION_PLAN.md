# ⚙️ Task & Background Work System — Execution Plan

> **Status:** Execution plan
> **System:** Task & Background Work System
> **Primary roadmap rows:** `10.2B`, `11.2`, `11.3`, `11.7`, `11.8`, `11.10`, `11.11`, `11.12`, `12.7`, `12.8`, `12.9`, `12.10`, `12.11`, `12.12`, `12.13`, `12.21`, `13.6`, `14.5`, `14.8`

This document defines the program of work for queued work, background work, waiting states, and the long-running task model in GoHarness.

The belief that shapes the whole design is this:

> Queueing, recovery from a blocked state, background jobs, plans, goals, schedules, and sub-agent orchestration are not separate features. They are all different views onto one task state machine.

If we build them as separate features, each one will grow its own version of "a thing that is running", and we will spend the following months reconciling them.

## Row coverage map

For each roadmap row this plan claims, here is the phase that owns it and how complete that coverage actually is.

| Roadmap row | Phase | Coverage |
|---|---|---|
| `10.2B` Wait First / Race | Phase E, deliverable 5 | **core.** The three completion modes are specified, including what happens to the children that lose. |
| `11.2` Message queue while the agent is running | Phase A | **core.** |
| `11.3` Cancel versus abort | Phase B, plus the state semantics in section 5.3 | **core.** |
| `11.7` Durable background sub-agent jobs | Phase C | **core.** |
| `11.8` Named sub-agent roles and packaged workflows | Phase E | **reference.** The plan treats these as metadata on a task rather than a separate feature. |
| `11.10` Automatic build, lint, and typecheck feedback loop | Phase C, deliverable 5, and Slice 4 | **core.** Defined as a validation job kind with a bounded number of retries. |
| `11.11` Model-initiated questions to the user | Phase D | **core.** |
| `11.12` Persistent TODO and plan overlay | Phase H | **partial.** Covered as a projection only. |
| `12.7` Plan mode as logged collaboration state | Phase F and Phase H together | **partial.** The projection is specified, but the underlying logged state is not described in detail. |
| `12.8` Goals with autonomous round-driving | Phase F | **core.** |
| `12.9` Dynamic workflows over sub-agents | Phase E | **reference.** |
| `12.10` Multiple sub-agent providers | Phase A, deliverable 4 | **core.** A task can name the connection profile it should run on, resolved when the task is admitted. |
| `12.11` Structured sub-agent output | Phase E, through receipts | **reference.** |
| `12.12` Per-agent tool scoping and personas | Phase E, deliverable 6 | **core.** The plan records the scope on the task. Enforcement is deferred to the policy engine at row `12.1`. |
| `12.13` Per-session agent presets | Phase A, deliverable 5 | **core.** Session-scoped defaults, resolved when the task is admitted. |
| `12.21` Scheduled and mission runs | Phase G | **core.** |
| `13.6` Plan mode, goal mode, and progress rows | Phase H | **partial.** Covered as a projection only. |
| `14.5` Sub-agents as data files | Phase E | **reference.** |
| `14.8` Background agents, task roster, queued command semantics | Phase H | **reference.** |

This table was revised once, and it is worth recording why. Five rows previously had no home in the plan body. They were folded in as follows: `12.10` and `12.13` into Phase A, `10.2B` and `12.12` into Phase E, and `11.10` into Phase C. Row `11.10` had earlier been suggested for Phase E, but the validation loop is really about job lifecycle, meaning retries, receipts, and classifying failures, rather than about sub-agents. That earlier note also contradicted this table, and both are now consistent.

---

# 1. System scope

This system owns work that is waiting to happen or already happening without the user watching it. Concretely: queued work from the user, pending work local to a session, background agent jobs that survive beyond a single turn, states where the system is waiting on a person, the receipts and outputs a task produces, the runtime state behind plans, goals, and schedules, and the rosters and progress rows that display all of it.

It does not own the shell itself, retrieval from archived memory, permission and hook policy, the mechanics of execution isolation, or MCP transport. Those systems will read task state, but they should not each invent their own queue or job model.

---

# 2. Where the design comes from

## 2.1 What the existing GoHarness source already provides

### `src/web/js/composer.js`

The composer already queues steering and follow-up messages, though it does so entirely in browser memory. It also already understands a busy or blocked state, and it routes commands such as `/compact`, `/workflow`, `/new`, and `/settings`.

The useful part is that this proves the interaction users actually want. The limit is that none of it survives a page reload, because the state lives in the tab.

### `src/subagent.go`

This gives us child sessions, lifecycle events for sub-agents, a depth limit, and a process-wide lock that serialises file writes.

This is the first real piece of delegated, durable-ish work in the codebase, and it is the natural starting point for the wider task model.

### `src/agent_runtime.go` and `src/agent_run.go`

Between them these provide an explicit identity for each agent at runtime, retry and wait behaviour around provider failures, session persistence, and a clear distinction between the root agent and a sub-agent.

This is a solid base for moving from queue semantics that live in a browser tab to queue semantics the host owns.

### `src/workflow.go`

Workflows already execute concurrently, track the lifecycle of each node, and distinguish terminal states from failures.

This matters because background and scheduled work may eventually be expressed as workflows. It also sets a boundary: the task system has to work on its own, independently of the workflow engine, rather than being a thin wrapper around it.

---

## 2.2 What we learned by reading reference implementations

### A. `Prime Agent`, specifically its queue, scheduling, and goal model

We read `packages/coding-agent/src/core/agent-session.ts`, `packages/coding-agent/src/core/cron-jobs.ts`, `packages/coding-agent/src/core/goals.ts`, `packages/coding-agent/src/core/session-manager.ts`, `packages/coding-agent/docs/daemon.md`, and `packages/coding-agent/test/suite/agent-session-queue.test.ts`.

Four lessons came out of it.

**Queue and admission are owned by the host.** The queue tests and `agent-session.ts` together describe a model where prompts and actions are admitted deliberately, where queued work appears as structured state rather than as text, and where pausing, aborting, and restarting are explicit transitions. Queued items are not simply messages that happen to be sitting somewhere.

For GoHarness the implication is direct: queueing should move from a browser convenience to a backend-owned session action system.

**Scheduled jobs are persisted per session.** `cron-jobs.ts` stores each scheduled job with an identifier, the session that owns it, its status, the next time it should run, how many times it has run, what to do with its output, and how to recover if it was interrupted.

The lesson is not to copy that implementation. It is to copy the principle:

> Scheduled and background work should be durable records with an explicit lifecycle, not timers that exist only in memory.

**Goal state is a real state machine.** `goals.ts` tracks goals as active, paused, complete, errored, or limited by budget. It accounts for what was spent, counts how many times it has continued, records timestamps, and returns explicit responses to the host.

This is exactly the kind of thing that should not be implemented in GoHarness as a loose convention inside a prompt.

**The daemon architecture keeps clients separate from durable workers.** `docs/daemon.md` draws a clear line between clients, a supervisor, workers, schedulers, and whoever owns a session.

GoHarness does not need to become a daemon, now or soon. But the underlying lesson applies:

> Durable work should not be modelled as something that only exists while a user interface is open.

### B. `Pi` and the wider execution philosophy

The top-level documentation is broader than what this plan needs, but the practical takeaway matches the previous section. Session and runtime control, and the continuation of work, should be owned by the host rather than inferred from chat turns.

### C. GoHarness's own sub-agent plan

`docs/SUBAGENT_PARALLELISM_PLAN.md` already establishes child runtime boundaries, recursion limits, structured handling of sub-agents, and the importance of explicit lifecycle events. It is the strongest internal bridge from what exists today into a fuller task system.

---

# 3. Where GoHarness stands today

## 3.1 What already exists

Queued steering and follow-up messages exist on the browser side. The runtime distinguishes the root agent from a sub-agent. Child sessions exist. Lifecycle events are broadcast over SSE. Workflows run concurrently. There are retry semantics around provider and model failures. The interface already has basic support for a blocked state.

## 3.2 What is missing

There is no backend-owned queue model, so nothing about queued work is true outside one browser tab. There are no durable task records. There is no single state machine covering waiting, running, cancelled, and aborted work. There are no job receipts. Schedules and goals are not host state. There is no truthful roster of background work. And there is no shared substrate for the plan, goal, and async work that later phases will need.

---

# 4. Program strategy

The system is built in three layers.

**Layer 1 is the backend session action queue.** This moves the queue semantics that exist today in the browser into state the host owns.

**Layer 2 is durable jobs and waiting states.** This turns asynchronous work and states that wait on a person into explicit records scoped to a session.

**Layer 3 is goals, schedules, and orchestration projections.** This builds the plan, goal, schedule, and fleet interfaces on top of the same task substrate, so they display state rather than defining it.

---

# 5. The core task model

## 5.1 What identifies a task

Every queued or background piece of work should carry an identifier, the session that owns it, where it came from, its current state, timestamps, the request that created it, a summary of its result or error, and optional references to whatever it produced.

A task record looks like this:

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

## 5.2 The state machine

The states a task can be in are:

- `queued`
- `admitted`
- `running`
- `waiting_user`
- `waiting_dependency`
- `completed`
- `failed`
- `cancelled`
- `aborted`

## 5.3 What the states mean

**`cancelled`** means the user or the system removed the task before it finished and before it produced a result.

**`aborted`** means a task that was already running was interrupted by force.

**`waiting_user`** means the task cannot continue until a person answers something.

**`waiting_dependency`** means the task is paused until another task, sub-agent, or workflow finishes.

These distinctions are not decoration. They drive which controls the interface offers, whether a task should be retried, how scheduling treats it, and what the policy engine will eventually decide about it.

---

# 6. Execution phases

# Phase A — A session action queue owned by the backend

## Goal

Replace the current situation, where the browser holds the only truth about queued work, with queue state the host owns.

## Deliverables

1. **A session action schema.** A queued prompt or action needs a representation, a stable identifier, and a stated delivery policy such as `steer` or `follow_up`.

2. **A persistence model.** Queued actions are stored in the session area. For version one, plain JSON files on disk are sufficient.

3. **An API for changing the queue.** This covers enqueueing an action, listing what is queued, removing one, editing one, clearing the queue, and admitting the next item.

4. **A provider binding per task**, which is roadmap row `12.10`. A task record can name the connection profile it should run on. That reference is resolved when the task is admitted, using the path that already exists: `ResolveAPIConfig` and `Agent.ProfileName` in `src/agent_runtime.go`. The practical effect is that background work can target a different provider than the foreground conversation. It also means the per-profile throttling that already exists, through `throttleKey` and `acquireThrottle`, applies to background work without any new mechanism. This matters because a fan-out of jobs onto a cheap profile should not trip that provider's concurrency limit.

5. **Session task presets**, which is roadmap row `12.13`. These are the defaults a session applies to background work: which profile to use, which tools are in scope, and how deep recursion may go. They are stored alongside the session metadata created by `createSessionMeta`, so a session can be configured once as "cheap profile, read-only, no recursion" instead of repeating that on every spawn. Presets are resolved once, when a task is admitted, and recorded on the task. They are never re-resolved partway through a run, because a task that changes provider or permissions mid-flight is very hard to reason about.

## Proposed new files

`src/session_actions.go`, `src/session_action_store.go`, `src/session_action_api.go`.

## Files likely to change

`src/web.go`, `src/agent_runtime.go`, `src/agent_run.go`, and `src/web/js/composer.js`.

## Why this comes first

It is the smallest amount of substrate that makes queue semantics true outside one browser tab. Everything else in this plan assumes that queue state is real.

---

# Phase B — Unify busy, blocked, and waiting states

## Goal

Turn the current ad hoc handling of busy and blocked states into one shared model of what the host is doing.

## Deliverables

1. **A session runtime state structure** covering running, blocked, `waiting_user`, `waiting_dependency`, and retrying.

2. **A contract for what gets emitted**, covering queue changes, changes in wait state, and transitions to completion or failure.

3. **Integration with the configuration endpoint the interface already reads**, so the current state of a session is visible to the web shell rather than inferred by it.

## Proposed new files

`src/session_runtime_state.go`, `src/session_runtime_events.go`.

## Why this comes first in its layer

The composer already tries to express a busy or blocked state, but it is guessing. This phase gives it something true to read.

---

# Phase C — Durable background jobs

## Goal

Represent long-running asynchronous work as first-class jobs rather than as a call that happens to take a while.

## Deliverables

1. **A job record model.**

2. **A job store scoped to the session.**

3. **Rules for recovering from a crash or an interruption.**

4. **A contract for task receipts**, meaning a consistent summary of what a job produced.

5. **A validation job kind**, which is roadmap row `11.10`. This is the first job kind that is not a sub-agent. It runs a build, lint, or typecheck, classifies the failure, retries within a budget, and stops when the check passes. The job record carries how many attempts have been made, what class of failure the last one was, and the exact command that produced it.

   Ownership of this needs stating precisely, because two systems are involved. This system owns the loop, its state, and its receipts. Actually running the command belongs to the Execution & Review System, through the persistent execution surface described by rows `12.4` and `13.9`.

   One risk to guard against: this must not become an implicit retry storm. Retries are bounded, and every attempt appends its own receipt, so a summary of "we tried eleven times" is visible rather than hidden.

   The reason this is a job kind rather than a special case is that goals and schedules will want to request validation too, and they should not need new machinery to do it.

## Proposed new files

`src/jobs.go`, `src/job_store.go`, `src/job_recovery.go`.

## Design rules

Jobs are local to a session first. No daemon is required for version one. The system must not silently replay side effects it is not certain about. And resumed or interrupted jobs need semantics that are stated clearly rather than inferred from what happens to be on disk.

## Where this design comes from

Prime Agent's `cron-jobs.ts` and its daemon and session ownership model informed this phase. The idea is borrowed; the implementation should be lighter and native to GoHarness rather than a port.

---

# Phase D — Waiting on the user as a first-class state

## Goal

Make a question to the user a concern of the task system rather than only of the interface.

## Deliverables

1. **The `waiting_user` task state.**

2. **A record of the pending question**, so the system knows what it is waiting for.

3. **Defined semantics for resuming** when the answer arrives.

4. **Support for the interface to take over the composer** while the question is outstanding, which is a projection of this state rather than its source.

## Files likely to change

`src/web.go`, `src/agent_run.go`, and the frontend shell and composer modules.

## Why now

This is the clean bridge from row `11.11` into the wider task substrate. Approvals and blocked work will use the same waiting state later, so getting it right here avoids a second waiting mechanism appearing.

---

# Phase E — Sub-agent work as a job family

## Goal

Bring sub-agent execution into the durable task model instead of leaving it as a parallel mechanism.

## Deliverables

1. **A link from a parent task to its child tasks.**

2. **Structured records for a child's result**, so the parent does not have to parse prose.

3. **Metadata describing a task family**, so a group of related tasks can be understood together.

4. **An API for the roster and for progress.**

5. **Completion modes for fan-out**, which is roadmap row `10.2B`. There are three, and the differences between them matter:

   - `wait_all` is what ships today. The description of `spawn_sub_agent` already promises that calling it several times in one response runs the tasks concurrently and returns the results together, so this mode is existing behaviour rather than something new.
   - `wait_first` returns as soon as the first child reaches a terminal state. The siblings keep running and remain visible as tasks. They are not silently dropped.
   - `race` also returns at the first terminal state, but it cancels the losers, using the `cancelled` and `aborted` states from row `11.3`. Each loser emits a terminal receipt, so nothing is left running without a record.

   Whichever mode was used is recorded on the parent task, so the interface can explain why some children were cancelled instead of leaving the user to guess.

6. **Capability scope per task**, which is roadmap row `12.12`. This means the sub-agent specification accepts a set of tool classes (read, search, execute, write) and an optional role or persona.

   This is a restoration rather than an invention. Section 4.1 of `docs/SUBAGENT_PARALLELISM_PLAN.md` already designed `model` and `tools` fields for `spawn_sub_agent`. The specification that shipped was reduced to `{task, context, expect, description}`, which dropped both.

   The scope is resolved when the task is admitted and recorded on the child task. Enforcement stays where it belongs, with the capability seams the policy engine will provide under row `12.1`, rather than being implemented as ad hoc checks inside the task runtime.

## Files likely to change

`src/subagent.go`, `src/agent_runtime.go`, `src/web.go`.

## Why this matters

This is how rows `11.7`, `11.8`, `12.9`, `12.11`, and `14.5` stop being separate features and become one record of durable work with a shape.

---

# Phase F — Goal runtime

## Goal

Make goals real runtime state owned by the host, rather than a convention expressed in a prompt.

## Deliverables

1. **A record for goal state.**

2. **Defined transitions between goal statuses.**

3. **Accounting for budget and time.**

4. **A counter for continuations and the conditions that stop the goal.**

5. **A contract for how goal context is injected into a prompt.**

## Proposed new files

`src/goals.go`, `src/goal_accounting.go`, `src/goal_runtime.go`.

## Where this design comes from

Prime Agent's `goals.ts` is the reason this needs to be a real state machine with accounting rather than a prompt convention. The idea is worth borrowing. The TypeScript implementation is not something to port.

---

# Phase G — Scheduled and mission jobs

## Goal

Build scheduled work on top of durable jobs, rather than inventing a second asynchronous system beside them.

## Deliverables

1. **Schedule records scoped to a session.**

2. **A computation of when a schedule should next run.**

3. **Claim-before-run semantics**, so a job is marked as claimed before it starts and a crash mid-run does not cause it to fire twice.

4. **Recovery for interrupted claims.**

## Proposed new files

`src/schedules.go`, `src/scheduler.go`.

## Why this matters

This is where row `12.21` and parts of `14.8` should land. It is also the clearest case of a rule that applies across the whole plan: scheduling must not be built before durable jobs exist, because otherwise the schedule becomes its own job store.

---

# Phase H — Task roster, plan chips, and progress rows

## Goal

Expose the task system through projections that are truthful, for both the interface and operators.

## Deliverables

1. **An endpoint for the task roster.**

2. **Counts of active and pending work.**

3. **Progress rows and chips.**

4. **Badges for goals, plans, and schedules.**

## A boundary worth stating

This phase is projection only. It must not introduce state of its own. Everything it displays has to come from the task, job, and goal records underneath, because a roster that keeps its own tally will eventually disagree with reality.

---

# 7. Lessons from reference projects, applied directly

## 7.1 Queue semantics, from Prime Agent

The lesson is that admission and queue state belong to the host, that queued work is structured rather than being chat text, and that interrupts, pauses, and clearing are explicit transitions rather than side effects.

Applied here: move the truth about the queue into backend state and keep the frontend as a view of it.

## 7.2 Scheduled jobs, from Prime Agent

The lesson is that scheduled jobs are durable records owned by a session, that due work is claimed and advanced before any prompt is delivered, and that interrupted work can be recovered without replaying side effects we are unsure about.

Applied here: jobs and schedules are file-backed state, not bare timers.

## 7.3 Goals, from Prime Agent

The lesson is that a goal is not a string in a prompt. It has a status, a budget, timestamps, counters, and explicit responses from the host.

Applied here: if we implement goal mode, it has to be a genuine state machine owned by the host.

## 7.4 Sub-agents, from GoHarness's own runtime

The lesson is that child sessions and explicit lifecycle events are already valuable, and that the next step is not to invent a new background model but to make what exists durable and queryable.

---

# 8. Concrete slices of work

## Slice 1 — A durable session queue

**What it adds.** Persisted session action records, an API for changing the queue, and admission rules owned by the backend.

**How we know it worked.** A queued follow-up survives a browser refresh, and it looks the same to every client watching that session.

---

## Slice 2 — Runtime waiting states

**What it adds.** The `waiting_user`, `waiting_dependency`, and retrying states, updates over SSE, and truthful exposure to the interface.

**How we know it worked.** The blocked-state interface reflects backend state rather than something JavaScript guessed.

---

## Slice 3 — Background sub-agent jobs

**What it adds.** Task records for sub-agents, receipts that survive the turn, and an API for progress and results.

**How we know it worked.** Long-running sub-agent work can outlive the foreground turn and still be inspected afterwards.

---

## Slice 4 — Validation loop jobs

**What it adds.** The validation job kind for build, lint, and typecheck; bounded retry with failure classification; one receipt per attempt; and the family link so a validation job reports against whichever task asked for it.

**How we know it worked.** An agent can ask for a check, watch it fail, fix the problem, and check again, with every attempt visible as its own record instead of one opaque command result.

---

## Slice 5 — Goal state machine

**What it adds.** Goal records, accounting, and the completion, error, and pause states.

**How we know it worked.** Goal mode becomes state the host owns and can resume, rather than behaviour that exists only in the shape of a prompt.

---

## Slice 6 — Scheduled jobs

**What it adds.** Persistence for schedules, claim-before-run behaviour, and recovery for a claim that was interrupted.

**How we know it worked.** Scheduled work is durable, and an interruption does not cause an uncertain action to be repeated.

---

## Slice 7 — Task roster and progress projections

**What it adds.** The roster endpoint, active and pending counts, progress rows and chips for the shell, and badges for goals, plans, and schedules.

**How we know it worked.** The shell can display queued, running, waiting, and finished work directly from task state, without maintaining its own version of that information.

---

# 9. Testing plan

## 9.1 The queue

Tests cover enqueueing, editing, removing, and clearing. They check that the queue survives a browser refresh and reload. They check that several clients observing one session see the same queue. They check ordering and priority.

## 9.2 Runtime state

Tests cover the difference between cancelling and aborting. They cover the difference between busy, `waiting_user`, and `waiting_dependency`, which are easy to confuse and expensive to confuse. They cover the transitions into and out of retrying.

## 9.3 Jobs

Tests cover the full lifecycle of a durable background job, recovery from a crash or interruption, and the absence of duplicate completion events. They also check that a validation loop stops at its retry budget instead of looping indefinitely, and that every validation attempt produces its own receipt.

## 9.4 Goals

Tests cover budget accounting, pausing and resuming, correct completion, and transitions into error states.

## 9.5 Schedules

Tests cover the calculation of due times, the difference between recurring and one-shot schedules, and the cleanup of an interrupted claim.

## 9.6 Fan-out completion modes

Tests cover all three modes. `wait_all` still returns every child result, which guards the behaviour that already ships. `wait_first` returns at the first terminal child while leaving siblings running and visible. `race` cancels the losers, and every loser emits a terminal receipt, so no child is left unaccounted for. One further test matters more than it looks: a cancelled loser must not still be writing to the workspace after the cancellation call returns.

---

# 10. Risks and things we are deliberately not doing

## Risks

1. **Polishing the queue in the interface before the backend is true.** A beautiful queue that exists only in a tab is worse than an honest one.
2. **Ending up with two state models**, one for goals and another for jobs. They describe the same kind of thing and must not diverge.
3. **Letting scheduled or background work change state without leaving a receipt.** If it happened, there should be a record of it.
4. **Mixing policy concerns into the task substrate too early.** What a task may do is the policy engine's question. Whether it is running is this system's question.

## Anti-goals

1. Do not start by building a daemon.
2. Do not let the browser remain the source of truth for queued work.
3. Do not ship goal or schedule interfaces before the host state machine behind them exists.
4. Do not let sub-agent durability become a separate architecture from the rest of the task system.
5. Do not ship `wait_first` or `race` without explicit cancellation semantics. A child that keeps writing to the workspace after we believe it stopped is worse than simply waiting for it.

---

# 11. Recommended order of implementation

1. **Slice 1** — the durable session queue
2. **Slice 2** — runtime waiting states
3. **Slice 3** — background sub-agent jobs
4. **Slice 4** — validation loop jobs
5. **Slice 5** — the goal state machine
6. **Slice 6** — scheduled jobs
7. **Slice 7** — roster and progress projections

This order produces something visible early while keeping the system coherent. Each slice is useful on its own, so the sequence can pause without leaving anything half-built.

---

# 12. What success looks like

This system is working when GoHarness can honestly say the following. Queued work is durable. Blocked work is classified correctly rather than guessed at. Background jobs are explicit and can be inspected. Goals and schedules are state machines the host owns. Sub-agents belong to the same lifecycle as everything else. And the interface only displays runtime truth, instead of inventing a version of its own.
