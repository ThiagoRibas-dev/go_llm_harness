# 🧠 Session Event & Memory System — Execution Plan

> **Status:** Execution plan
> **System:** Session Event & Memory System
> **Primary roadmap rows:** `9.1`, `9.2`, `9.3`, `11.6`, `11.9`, `12.2`, `12.18`, `12.29.7`

This document turns the Session Event & Memory System from a cluster of related roadmap rows into an actual program of work.

It is shaped by three facts about the project as it stands.

First, GoHarness already stores a great deal of durable session state, so this system is mostly about giving that state a proper shape rather than starting from nothing.

Second, we should not hold back improvements to memory until a perfect event-log substrate exists. Useful recall can be built on what is already on disk.

Third, the runtime must stay pure Go, file-backed, local-first, and inspectable. Nothing here should require a database, a service, or an opaque cache.

For those reasons the plan has two layers. Layer one is a file-backed version that can ship soon. Layer two is a canonical event log that eventually becomes the single source of truth. The plan is explicit about which phase belongs to which layer.

## Row coverage map

For each roadmap row this plan claims, here is the phase that owns it and how complete that coverage actually is. A coverage of `gap` would mean the row is claimed but not yet written into the plan body; after a revision pass, there are none left.

| Roadmap row | Phase | Coverage |
|---|---|---|
| `9.1` O(1) range loader + hierarchical memory decay | Phase A for the identity substrate, Phase B for the epoch manifests | **partial.** The plan specifies the substrate the loader will need, but does not design the loader itself. |
| `9.2` Visual memory map dashboard | Slice 5 | **partial.** The plan delivers the API the dashboard would read from. The dashboard surface itself belongs to the Shell system. |
| `9.3` Hierarchical archived-memory retrieval | Phases B, C, and D | **core.** This is the centre of the file-backed version. |
| `11.6` Session tree / time-travel navigator | Phase E | **core.** Built as a projection over the storage that exists today. |
| `11.9` Cross-session memory | Phase C, stage 1, and Slice 3 | **core.** |
| `12.2` Session event log as source of truth | Phase F, Slice 6 | **partial.** The plan produces a draft substrate. It does not yet contain a migration specification. |
| `12.18` Session query with full-text search | Phase C, deliverable 5, and Slice 4 | **partial.** The plan specifies a query index over whole artifacts. Filtering by individual event is Phase F work and is not promised here. |
| `12.29.7` Step-grouped transcript, compaction placement, streaming-tail isolation | Phase E | **partial.** Covered as a transcript projection only. |

One open question is carried from the roadmap: whether Phase F and Slice 6 together satisfy the roadmap's request for a *session event log and migration spec*, or whether that document is still owed separately.

---

# 1. System scope

This system owns the durable memory of a session. Concretely, that means canonical session history semantics, the structure of the session and branch graph, the boundaries created by compaction and archiving, retrieval over archived memory, recall across sessions, transcript projections, and the inspection APIs that later surfaces will read.

It does not own task orchestration, approvals and hooks and policy, the behaviour of the shell and composer, MCP transport logic, or worktree isolation. Those systems will consume the projections this system produces, but they should not define their own versions of them.

---

# 2. Where the design comes from

## 2.1 What the existing GoHarness source already provides

### `src/agent_runtime.go`

This file gives us per-agent saving and loading of turns, session-scoped persistence, reconstruction of a session's history, and turn numbering that is safe under concurrency.

The lesson is that a workable file-backed session model already exists. What is missing is a common abstraction over it, so that everything reading session history does not have to understand the file layout individually.

### `src/agent.go`

This is the richest existing source. It handles compaction boundaries, archived turn folders, summary persistence, rollback mechanics, pinned file storage, and BM25 search across the workspace, the session, and uploads.

This file is the strongest base for the file-backed version of archived memory, because compaction already produces exactly the kind of boundary that a memory epoch needs.

### `src/workflow.go`

Workflows already emit structured events and track the lifecycle of each node, and they persist the final assistant output.

The lesson is conceptual rather than practical: event-shaped structures already exist in the runtime, even though they are not yet the canonical way sessions are stored. That means the event-log direction is a continuation of something the project already does, not a foreign idea.

### `src/subagent.go`

This gives us child session creation, the link between a parent and a child session, and lifecycle signalling for sub-agents.

This matters later, when the session graph becomes a real structure and when recall has to decide whether a sibling branch is inside scope.

---

## 2.2 What we learned by reading reference implementations

For this plan we read the actual source of three reference projects rather than summaries of them. Each one contributes a different lesson.

### A. `VictorTaelin/OptMem`, specifically the `memo` module

What we inspected: an append-only log model, an explicit tree of summaries, rules about what can be rebuilt, and a fixed-width record format.

The lessons that matter are these.

There is a clean separation between truth and cache. `LOG.txt` is the durable source of truth, while everything under `TREE/` is a summary that can be regenerated from it.

Rebuildability matters more than clever compression. Summaries can be deleted and rebuilt, and recovering from a corrupted summary is part of the design rather than an emergency.

Hierarchy should be treated as a derived structure for navigation and retrieval, not as the only durable record of what happened.

The important thing for GoHarness is *not* the fixed-width file format. It is the principle underneath it:

> Keep canonical storage simple and append-oriented, and make the summaries and projections rebuildable from it.

### B. `Lumi-node/infinite-context`

What we inspected: `src/adapters/index/hat.rs`, `src/adapters/index/mod.rs`, `src/adapters/index/learnable_routing.rs`, `src/adapters/python.rs`, and `python/infinite_context/context.py`.

Four observations came out of reading the source.

What the project calls "infinite context" is really three ordinary techniques combined: hierarchical retrieval, beam-style pruning of branches, and bounded reinjection of what was found.

The hierarchy is fixed rather than learned. It runs from Global to Session to Document to Chunk.

The retrieval layer is approximate, and the code says so. `HatIndex` is labelled as approximate in the source itself.

The high-level wrapper has correctness limits. The beam-width setting appears partly unwired, and the way retrieved text is mapped back in the Python wrapper looks fragile.

The lesson for GoHarness is a caution rather than a technique:

> Hierarchical retrieval of cold memory is worth building, but only around honest provenance and storage contracts that do not move underneath it.

This is also why the plan refuses the phrase "infinite context" and describes bounded retrieval instead.

### C. `Prime Agent` session runtime

What we inspected: `packages/coding-agent/src/core/session-manager.ts`, `packages/coding-agent/docs/session-format.md`, and `packages/coding-agent/docs/daemon.md`.

Three lessons came out of it.

Typed JSONL session entries are a good middle ground. They can be appended to, they can be read by a human, they can be migrated, and they can be projected into other shapes.

A versioned session schema matters in practice. Prime Agent explicitly migrates sessions forward between versions, which is a strong argument for GoHarness introducing versioned schemas early rather than after the format has drifted.

Projection boundaries should be named explicitly: the session header, the message entries, bookkeeping entries that are not conversation, compaction entries, and the navigation state of the tree.

The lesson for GoHarness:

> Event-style typed entries are a practical way to unify history, branching, compaction, and retrieval without giving up the ability to inspect the files by hand.

---

# 3. Where GoHarness stands today

## 3.1 What already exists

Session turns are already stored as JSON files, one per turn. Session metadata exists. Compaction boundaries are tracked, and archived turns are moved into folders named for the boundary. Summaries are written to disk. Rollback and branching have some semantics already. Uploads and deliverables exist as artifacts. BM25 search already covers the workspace, the session, and uploads. Workflows already produce event streams.

## 3.2 What is missing

There is no canonical event abstraction, so nothing can describe what happened in a session without referring to files. There is no unified projection of the session tree. There is no durable format for an archived memory unit. Retrieval across sessions does not exist. There is no mechanism for injecting archived memory under a prompt budget. And there is no stable API that transcript, trajectory, and memory surfaces can all read from.

---

# 4. Program strategy

This program ships in two layers.

## Layer 1 — file-backed archived memory, version one

This layer is built on the turn files and compaction folders that already exist. Its goal is to make older history searchable and reinjectable without waiting for the event-log migration to finish. This layer is where the real user value is, and it is deliberately designed so that it can later be re-pointed at a better substrate.

## Layer 2 — the canonical event log

This layer is the long-term cleanup. Its goal is for transcript, trajectory, session graph, and memory to all read from one substrate instead of each reading the file layout directly.

The consequence for sequencing is that row `9.3` should not be blocked on row `12.2`. At the same time, version one must not hard-code itself so deeply into today's file layout that migration becomes painful later. That is why Phase A exists at all: it introduces stable identities before anything is derived from them.

---

# 5. Execution phases

# Phase A — Normalize the file-backed session substrate

## Goal

Make the existing session storage consistent enough that derived memory projections can be built on top of it without guessing.

## Deliverables

1. **Schema versioning for the turn and metadata layout.** Session metadata should carry an explicit version field. The meaning of compaction and archive should be written down as a versioned contract rather than existing only in the behaviour of the code.

2. **Stable identity for sessions and branches.** We need defined identifiers for `workspace_id`, `session_id`, and `branch_id`. Branches should be referable by identity rather than by the name of the folder they happen to live in.

3. **Identity for compaction epochs.** Each compaction boundary should have an explicit epoch identifier, so a memory unit can say which epoch it came from.

## Files likely to change

`src/config.go`, `src/agent.go`, `src/agent_runtime.go`, `src/web.go`.

## Why this comes first

Without stable identities and named epochs, memory units become ad hoc structures tied to filenames. Every later phase would then inherit that fragility.

---

# Phase B — Derive archived memory units

## Goal

Create a derived memory layer on disk, built from the archived sessions that already exist.

## Deliverables

1. **A memory unit schema.** This means implementing `MemoryUnit` and `MemoryEpoch` as real types with defined fields.

2. **An epoch builder that runs on compaction.** When compaction succeeds, the system keeps doing what it does today, which is writing a summary. It then additionally derives memory-unit files and writes an epoch manifest describing them.

3. **A workspace memory catalog.** Epochs are registered by workspace, session, and branch, so retrieval can find them without scanning.

## Proposed new files

`src/memory_units.go`, `src/memory_epochs.go`, `src/memory_catalog.go`.

## Files likely to change

`src/agent.go`, at the point where compaction completes. `src/config.go`, if identifiers or settings are needed.

## Notes on storage

Version one stores two things per unit: a `.txt` payload that BM25 can index, and a `.meta.json` sidecar holding provenance.

This keeps the design consistent with the values the rest of the project already follows. The files stay inspectable by a human, everything stays local, and no new dependency is introduced.

---

# Phase C — Retrieval over archived memory

## Goal

Retrieve relevant cold memory under a strict budget, so that archive retrieval cannot crowd out the conversation that is actually happening.

## Deliverables

1. **A `MemoryQuery` and `MemoryRetrievalResult` API.**

2. **Coarse narrowing over epochs and groups,** so that leaf-level search only runs over a plausible subset.

3. **Leaf retrieval over individual memory units.**

4. **A prompt assembly formatter,** which turns retrieved units into a block that can be inserted into a prompt.

5. **A session query projection for full-text search**, covering roadmap row `12.18`. This deserves more detail because it is a separate concern from cold-memory retrieval.

   The projection is a queryable index over session artifacts: the turn files, the `compacted_summary_up_to_turn_%03d.json` summaries, uploads, and deliverables. It should reuse the lexical engine that already exists in `src/bm25.go`, specifically `BM25Engine.AddDocument` and `Search`. There is no reason to add a dependency here. One thing worth noting about the current behaviour: `executeBM25Search` in `src/agent.go` builds a throwaway index on every call, and it only ever indexes the session's uploads folder. The projection replaces both of those limitations.

   Every hit should carry provenance: the session, the branch, the epoch, the turn range, and the kind of artifact it came from. Callers should be able to filter by scope, choosing this session, this workspace, or every known workspace.

   This surface exists so a user can search their own history, which is different from the cold-memory retrieval above. That retrieval exists to fill a prompt; this one exists to answer a question.

   One limitation should be stated plainly rather than discovered later. Results are accurate at the level of whole artifacts, not individual events. A filter such as "show me only approvals" or "only calls to this tool" needs the Phase F substrate, and this plan does not promise it before then.

## Proposed new files

`src/memory_query.go`, `src/memory_retrieval.go`, `src/memory_prompt.go`.

## How retrieval works

Retrieval runs in three stages.

**Stage 1 is group narrowing.** The query selects candidate groups rather than searching everything: the current session's archived epochs, sibling branches, other sessions in the same workspace, and the groups formed by uploads and deliverables.

**Stage 2 is leaf search.** BM25 runs over the text inside those groups, which includes the raw text of turn units, summary text, upload chunks, and deliverable memory units.

**Stage 3 is budgeted assembly.** The output is a bounded block that names where each piece came from, groups results by source, and reports how many items were left out because of the budget.

## What justifies this phase

Three documents support it. `docs/BM25_SCALING_RESEARCH.md` provides evidence that lexical retrieval scales well and should come first. `docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md` describes the hierarchical retrieval and bounded reinjection pattern. `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md` gives the GoHarness-specific structure.

---

# Phase D — Connect archived retrieval to the agent at runtime

## Goal

Make archived memory available to the root agent in a way that is controllable and visible, rather than automatic and mysterious.

## Deliverables

1. **A recall path scoped to the current session.**

2. **A cross-session recall path scoped to the workspace.**

3. **A policy for when and how retrieved memory is injected into the system prompt or context.**

4. **Visible provenance for anything that was retrieved**, so a user can see that memory was used and where it came from.

## Design requirements

Archived memory must never quietly displace recent context. Retrieval should appear in logs and debug traces, so that a surprising answer can be explained afterwards. And the assembled prompt should show which parts of it came from memory rather than from the current conversation.

## Files likely to change

`src/agent_run.go` and `src/agent_runtime.go`. `src/llm.go` only if the prompt formatting path needs adjusting.

## An important boundary

This phase should not turn archived retrieval into an always-on system that runs behind the user's back. Start with explicit retrieval calls in the places where we know retrieval helps, and allow conservative automatic use only in cases where context is obviously missing.

---

# Phase E — Session tree and transcript projections

## Goal

Improve how a session can be inspected and navigated, using the storage that exists today rather than waiting for the event log.

## Deliverables

1. **A projection of the session and branch graph.**

2. **A transcript grouping projection.**

3. **Introspection endpoints for memory and epochs.**

## Likely API surface

Listing epochs, inspecting an epoch summary, inspecting the units inside an epoch, querying archived memory, and querying the ancestry of a session or branch.

## Why this phase exists

It lets the Shell & Interaction System build truthful history and memory surfaces earlier. Without it, those surfaces either wait for the event log or invent their own reading of the files, and the second option is how two systems end up disagreeing.

---

# Phase F — The canonical event log

## Goal

Introduce the long-term source of truth that eventually replaces the ad hoc turn-file semantics.

## Deliverables

1. **An event schema.**

2. **An append-only event log.**

3. **Projection builders**, starting with the three that matter most: transcript, tree, and memory.

4. **A dual-write migration path**, so the event log can be filled while the current storage remains authoritative.

5. **Replay and rebuild tooling**, so projections can be regenerated from the log and verified against today's output.

## Proposed new files

`src/session_events.go`, `src/session_event_log.go`, `src/session_projection_transcript.go`, `src/session_projection_tree.go`, `src/session_projection_memory.go`.

## The migration rule

The event log should become canonical, but the archived-memory APIs should keep working through a projection interface. That way the behaviour other systems depend on stays stable while the internals change underneath.

---

# 6. Lessons from the reference projects, applied directly

## 6.1 From OptMem: truth and derived cache are different things

The lesson from reading the source is that logs are the truth and summaries can be rebuilt at any time.

Applied to GoHarness, this means turn and event storage must be canonical, and epoch summaries and memory units must be rebuildable projections rather than the only copy of anything.

## 6.2 From Prime Agent: typed, versioned session artifacts

The lesson is that session entries are typed and versioned, and that migrating between versions is treated as a normal part of the design.

Applied here, this means we should stop letting the shape of session files drift silently. The session, event, and memory formats should be versioned, and migration should be something we do deliberately and can test.

## 6.3 From infinite-context: bounded retrieval, not unlimited context

The lesson is that hierarchy and beam pruning genuinely reduce search cost, but the result is still approximate rather than complete.

Applied here, this means using hierarchical narrowing because it is a sound technique, refusing to describe the result as unlimited context, and not exposing tuning controls unless the runtime genuinely honours them.

---

# 7. Concrete slices of work

Each slice below is meant to be independently shippable and independently useful.

## Slice 1 — Epoch manifests on compaction

This is the smallest backend change that produces something real.

**What it adds.** An epoch manifest written when compaction succeeds, memory units derived from the archived turn files, and an update to the workspace catalog.

**How we know it worked.** After compaction, the session folder contains the summary it always had, plus an epoch manifest and the derived unit files with their text and metadata.

---

## Slice 2 — Archived memory queries within one session

**What it adds.** The ability to query archived memory for the current session only, returning the top units with their provenance. No cross-session traversal yet.

**How we know it worked.** A prompt can retrieve relevant archived memory from the same session after compaction has happened.

---

## Slice 3 — Workspace-scoped recall

**What it adds.** Catalog lookup by workspace, querying across sibling sessions and branches, and a scope control so the user can widen or narrow the search deliberately.

**How we know it worked.** Recall works across sessions inside one workspace without needing embeddings of any kind.

---

## Slice 4 — Session query projection

**What it adds.** A derived index over turns, summaries, uploads, and deliverables; a query API that returns provenance with every hit; a scope filter for session or workspace; and the endpoint the sidebar will call.

**How we know it worked.** A user can search archived content across a workspace and get hits that point at a specific session, epoch, and turn range, all without waiting for the canonical event log.

**A limitation we are stating up front.** Results are accurate at the level of whole artifacts. Filtering by event, by tool, or by approval is Phase F work and is not claimed by this slice.

---

## Slice 5 — Provenance for user interfaces

**What it adds.** API endpoints that expose retrieval metadata, including which source summaries were selected and how many items were omitted.

**How we know it worked.** The frontend can show what memory was searched and what was chosen, instead of presenting retrieved memory as though it appeared from nowhere.

---

## Slice 6 — A draft event-log substrate

**What it adds.** A draft event schema, an append path that runs alongside the current writes, and an internal harness for replaying events in tests.

**How we know it worked.** Transcript, tree, and memory projections can begin migrating onto one substrate, and the replay harness can show that the projections still produce what they produce today.

---

# 8. Testing plan

## 8.1 Storage

The tests should confirm that epoch manifests appear when compaction runs, that unit files are written with valid provenance, and that rebuilding the workspace catalog produces the same result every time.

## 8.2 Retrieval

Retrieval tests cover the behaviour we are actually promising. Session-local retrieval returns the units we expect. Cross-session retrieval respects workspace scope and never leaks across workspaces. Overlapping units are deduplicated rather than returned twice. The budget is obeyed. The session query index is deterministic and filtered by scope, and every hit carries full provenance including session, epoch, turn range, and artifact kind.

## 8.3 Prompt assembly

These tests check the boundaries of injection. Archive text stays within its budget. Provenance labels survive assembly. And recent turns are never displaced by cold memory that has overrun its allowance.

## 8.4 Migration

Migration tests check that old sessions still load, that dual-writing to the event log can rebuild the same projections as today's path, and that rollback and branching semantics survive the transition.

---

# 9. What this means for the interface

This system will eventually power several surfaces: a history and memory tab in the details panel, the session tree and time-travel navigator, transcript grouping around compaction boundaries, and strips that show where recalled memory came from.

All of those are projections of state this system owns. The backend should be in place before the interface around it becomes elaborate, otherwise the interface ends up defining the behaviour by accident.

---

# 10. Risks and things we are deliberately not doing

## Risks

1. **Building retrieval directly against today's file layout, too rigidly.** The layout will change when the event log arrives, so the retrieval layer should depend on identities and projections rather than on filenames.
2. **Confusing a compaction summary with complete long-term memory.** A summary is a compressed view of a window of conversation, not a record of everything that happened.
3. **Overfilling prompts with archived evidence.** Retrieval needs a hard budget, and the budget needs a test.
4. **Leaking data across scopes.** Retrieval that crosses sessions must respect the workspace boundary, because the contents of one project should never appear in another.

## Anti-goals

1. Do not make embeddings a requirement for version one. Lexical retrieval is enough to prove the structure, and the research document argues it scales better at this size.
2. Do not claim unlimited context. The plan retrieves a bounded amount of relevant history, which is a different and more useful promise.
3. Do not hide retrieval from the user. If memory was used, it should be visible.
4. Do not replace canonical session truth with opaque memory caches. Caches can be rebuilt; the truth cannot be reconstructed from them.

---

# 11. Recommended order of implementation

1. **Slice 1** — epoch manifests and memory-unit derivation on compaction
2. **Slice 2** — archived retrieval within a session
3. **Slice 3** — workspace-scoped recall
4. **Slice 4** — session query projection
5. **Slice 5** — provenance API for the interface
6. **Slice 6** — draft event-log substrate

This order delivers something useful early while still moving toward the cleaner long-term architecture. Each step is useful on its own, which means the sequence can be paused at any point without leaving the system half-built.

---

# 12. What success looks like

This system is working when GoHarness can honestly say the following. Old context is compacted and archived rather than lost. Archived context can be searched. Recalled memory says where it came from. Recall across sessions is possible within a defined scope. And none of that required giving up the pure-Go, file-backed, local-first design.

The last point matters as much as the others. A memory system that only works because it became a service with a database would be a different project, not a better version of this one.
