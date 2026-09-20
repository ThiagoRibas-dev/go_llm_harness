# 🧠 Session Event & Memory System — Execution Plan

**Row coverage map** — for each roadmap row this plan claims, which phase owns it and how deep that
coverage actually is. `gap` means the row is claimed but not yet written into the plan body.

| Roadmap row | Phase | Coverage |
|---|---|---|
| `9.1` O(1) range loader + hierarchical memory decay | Phase A (identity/epoch substrate), Phase B (epoch manifests) | **partial** — specifies the substrate the loader needs; the loader itself is not designed here |
| `9.2` Visual memory map dashboard | Slice 4 (provenance API) | **partial** — API only; the dashboard surface is out of scope |
| `9.3` Hierarchical archived-memory retrieval | Phases B–D | **core** |
| `11.6` Session tree / time-travel navigator | Phase E | **core** (projection over current storage) |
| `11.9` Cross-session memory | Phase C stage 1, Slice 3 | **core** |
| `12.2` Session event log as source of truth | Phase F, Slice 5 | **partial** — draft substrate only; no migration spec yet |
| `12.18` Session query with FTS | — | **gap** — not addressed in the plan body |
| `12.29.7` Step-grouped transcript / compaction placement / streaming-tail isolation | Phase E | **partial** — transcript projection only |

Open decision carried from the roadmap: whether Phase F / Slice 5 satisfies the roadmap's requested
*session event log + migration spec*, or whether a separate spec is still owed.

> **Status:** Execution plan
> **System:** Session Event & Memory System
> **Primary roadmap rows:** `9.1`, `9.2`, `9.3`, `11.6`, `11.9`, `12.2`, `12.18`, `12.29.7`

This document turns the Session Event & Memory System from a conceptual roadmap cluster into an **implementation program**.

It is deliberately shaped around three realities:

1. GoHarness already has a lot of durable session state.
2. We should not pause all memory improvements until the perfect event-log substrate exists.
3. We must keep the runtime **pure Go, file-backed, local-first, inspectable, and truthful**.

So this plan is split into:

- a **file-backed v1** that can ship soon,
- and a **canonical event-log v2** that becomes the long-term substrate.

---

# 1. System scope

This system owns:

- canonical session history semantics,
- session/branch graph structure,
- compaction/archive boundaries,
- archived-memory retrieval,
- cross-session recall,
- transcript projections,
- and future memory/trajectory inspection APIs.

It does **not** own:

- general task orchestration,
- approvals/hooks/policy,
- shell/composer UI behavior,
- MCP transport logic,
- or worktree execution isolation.

Those systems will consume memory/session projections, but should not redefine them.

---

# 2. Source inputs and what they contribute

## 2.1 GoHarness source already relevant

### `src/agent_runtime.go`
Current useful substrate:
- per-agent save/load turn model
- session-scoped persistence
- per-session history reconstruction
- concurrency-safe turn numbering

This shows that GoHarness already has a viable file-backed session persistence model, but not a unified event abstraction.

### `src/agent.go`
Current useful substrate:
- compaction boundary handling
- archived turn folders
- summary persistence
- rollback mechanics
- pinned file storage
- BM25 search over workspace/session/uploads

This is the strongest existing base for a **file-backed archived-memory v1**.

### `src/workflow.go`
Current useful substrate:
- structured workflow events
- live node execution lifecycle
- final assistant output persistence

This provides an important lesson: event-like structures already exist conceptually in runtime behavior, even if not yet as the canonical session substrate.

### `src/subagent.go`
Current useful substrate:
- child-session creation
- parent/child linkage
- sub-agent lifecycle signalling

This is relevant to future branch/session graph expansion and cross-session memory.

---

## 2.2 Reference source inspection and lessons

## A. `VictorTaelin/OptMem` (`memo`)
Inspected source:
- append-only log model
- explicit tree summary cache
- rebuildability semantics
- fixed-width record model

Important lessons from actual source:

1. **Separation of truth vs cache**
   - `LOG.txt` is the durable source of truth
   - `TREE/` summaries are rebuildable cache/projections

2. **Rebuildability matters more than clever compression**
   - summaries can be dropped and rebuilt
   - corruption recovery is part of the model

3. **Hierarchy should be treated as a derived navigation/retrieval structure**
   - not as the only durable truth

For GoHarness, the key takeaway is **not** fixed-width files.
The key takeaway is:

> keep canonical storage simple and append-oriented, and make memory summaries/projections rebuildable.

## B. `Lumi-node/infinite-context`
Inspected source:
- `src/adapters/index/hat.rs`
- `src/adapters/index/mod.rs`
- `src/adapters/index/learnable_routing.rs`
- `src/adapters/python.rs`
- `python/infinite_context/context.py`

Important lessons from actual source:

1. Their “infinite context” is really:
   - hierarchical retrieval
   - plus beam-style branch pruning
   - plus bounded reinjection

2. The hierarchy is fixed:
   - `Global -> Session -> Document -> Chunk`

3. The retrieval layer is approximate:
   - the code itself labels `HatIndex` as approximate

4. Their high-level wrapper has correctness limitations:
   - beam-width exposure looks partially underwired
   - retrieved text mapping in the Python wrapper appears fragile

For GoHarness, the useful lesson is:

> hierarchical cold-memory retrieval is valuable, but must be built around honest provenance and stable storage contracts.

## C. `Prime Agent` session runtime
Inspected source/docs:
- `packages/coding-agent/src/core/session-manager.ts`
- `packages/coding-agent/docs/session-format.md`
- `packages/coding-agent/docs/daemon.md`

Important lessons from actual source:

1. **Typed JSONL session entries** are a strong middle ground
   - appendable
   - inspectable
   - migratable
   - projection-friendly

2. **Versioned session schema** matters
   - Prime explicitly migrates session versions forward
   - this is a strong argument for GoHarness introducing versioned event schemas early

3. **Projection boundaries should be explicit**
   - session header
   - message entries
   - custom/system bookkeeping entries
   - compaction entries
   - tree navigation state

For GoHarness, the useful lesson is:

> event-style typed entries are a practical way to unify history, branching, compaction, and retrieval without giving up file inspectability.

---

# 3. Current GoHarness baseline and gap analysis

## 3.1 What already exists

GoHarness already has:
- turn-by-turn JSON files,
- session metadata,
- compaction boundary tracking,
- archived turn folders,
- summaries,
- rollback/branch semantics,
- uploads,
- deliverables,
- BM25 search,
- workflow event streams.

## 3.2 What is missing

It still lacks:
- a canonical **event abstraction**,
- a unified **session tree projection**,
- a durable **archived memory unit** format,
- cross-session archived retrieval,
- a formal prompt-budgeted archived recall mechanism,
- a stable API for transcript/trajectory/memory projections.

---

# 4. Program strategy

This program should ship in **two layers**.

## Layer 1 — File-backed archived-memory v1
Built on top of current turn files and compaction folders.

Goal:
- make old history searchable and reinjectable **without waiting for full event-log migration**.

## Layer 2 — Canonical event-log substrate
Long-term cleanup and unification.

Goal:
- make transcript, trajectory, session graph, and memory all read from one event substrate.

That means we should not block `9.3` on `12.2`, but we also should not let v1 hard-code itself so deeply into today's file layout that migration becomes painful.

---

# 5. Execution phases

# Phase A — Normalize the file-backed session substrate

## Goal
Make current session storage consistent enough to support derived archived-memory projections.

## Deliverables

1. **Session schema versioning for GoHarness turn/meta layout**
   - Add an explicit version field in session metadata if missing.
   - Version compaction/archive semantics formally.

2. **Stable session/branch identity model**
   - Define:
     - `workspace_id`
     - `session_id`
     - `branch_id`
   - Ensure branches can be referred to independently from filenames.

3. **Compaction epoch identity**
   - Introduce explicit epoch IDs corresponding to compaction boundaries.

## GoHarness files likely touched
- `src/config.go`
- `src/agent.go`
- `src/agent_runtime.go`
- `src/web.go`

## Why now
Without explicit identities and epoch naming, archived-memory units become ad hoc and brittle.

---

# Phase B — Implement archived-memory unit derivation

## Goal
Create a file-backed derived memory layer from current archived sessions.

## Deliverables

1. **Memory Unit schema**
   - implement `MemoryUnit`
   - implement `MemoryEpoch`

2. **Epoch builder on compaction**
   - when compaction succeeds:
     - keep current summary behavior
     - derive memory-unit files
     - write epoch manifest

3. **Workspace memory catalog**
   - register epochs by workspace/session/branch

## Proposed new Go files
- `src/memory_units.go`
- `src/memory_epochs.go`
- `src/memory_catalog.go`

## GoHarness files likely touched
- `src/agent.go` (compaction completion path)
- `src/config.go` (if IDs/settings are needed)

## Notes
v1 should store:
- `.txt` payloads for BM25 indexing
- `.meta.json` sidecars for provenance

This keeps the system aligned with current GoHarness values:
- inspectable
- local
- dependency-light

---

# Phase C — Implement hierarchical archived-memory retrieval API

## Goal
Retrieve relevant cold memory under a strict budget.

## Deliverables

1. **MemoryQuery / MemoryRetrievalResult** API
2. **Coarse narrowing over epochs/groups**
3. **Leaf retrieval over memory units**
4. **Prompt assembly formatter**

## Proposed new Go files
- `src/memory_query.go`
- `src/memory_retrieval.go`
- `src/memory_prompt.go`

## Retrieval behavior

### Stage 1 — group narrowing
Search/select:
- current session's archived epochs
- sibling branches
- other sessions in the workspace
- upload/deliverable groups

### Stage 2 — leaf search
Use BM25 over:
- raw turn unit text
- summary text
- upload chunks
- deliverable memory units

### Stage 3 — budgeted assembly
Produce a bounded reinjection block with:
- provenance labels
- source grouping
- omitted-by-budget counts

## Reference grounding
This phase is justified by:
- `docs/BM25_SCALING_RESEARCH.md` — lexical-first scale evidence
- `docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md` — hierarchical retrieval and bounded reinjection pattern
- `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md` — GoHarness-native structure

---

# Phase D — Integrate archived-memory retrieval into agent/runtime flows

## Goal
Make archived retrieval available to the root agent in a truthful, controllable way.

## Deliverables

1. **Session-local archived recall path**
2. **Workspace-scoped cross-session recall path**
3. **System-prompt / contextual injection policy**
4. **User-visible retrieval provenance**

## Design requirements

- archived memory must not silently replace recent context
- retrieval should be explicit in logs/debug traces
- the assembled prompt must show where the memory came from

## GoHarness files likely touched
- `src/agent_run.go`
- `src/agent_runtime.go`
- `src/llm.go` only if prompt formatting paths need adjustment

## Important boundary
This phase should not yet turn archived retrieval into an always-on magical system.
Start with:
- explicit retrieval calls in known paths
- or conservative automatic use when context is clearly missing

---

# Phase E — Build session-tree and transcript projections on top of current storage

## Goal
Improve inspectability and navigation before full event-log migration.

## Deliverables

1. **Session/branch graph projection**
2. **Transcript grouping projection**
3. **Memory/epoch introspection endpoints**

## Likely APIs
- list epochs
- inspect epoch summary
- inspect epoch units
- query archived memory
- query branch/session ancestry

## Why this phase exists
It lets the Shell & Interaction System build truthful history/memory surfaces sooner, even before `12.2` is complete.

---

# Phase F — Canonical event-log substrate

## Goal
Introduce the long-term source of truth that eventually subsumes the ad hoc turn-file semantics.

## Deliverables

1. **event schema**
2. **append-only event log**
3. **projection builders**
4. **dual-write migration path**
5. **replay/rebuild tooling**

## Proposed new Go files
- `src/session_events.go`
- `src/session_event_log.go`
- `src/session_projection_transcript.go`
- `src/session_projection_tree.go`
- `src/session_projection_memory.go`

## Migration rule
The event log should become canonical, but archived-memory APIs should continue working through a projection interface, so external behavior stays stable while internals migrate.

---

# 6. Reference-project implementation lessons to apply directly

## 6.1 From OptMem: truth vs derived cache
Direct lesson from source:
- logs are truth
- summaries are rebuildable

Apply in GoHarness:
- turn/event storage must be canonical
- epoch summaries and memory units must be rebuildable projections

## 6.2 From Prime Agent: typed, versioned session artifacts
Direct lesson from source:
- session JSONL entries are typed and versioned
- migration is a first-class concern

Apply in GoHarness:
- do not keep adding silent file-shape drift
- version the session/event/memory formats
- make migration explicit

## 6.3 From infinite-context: bounded frontier retrieval, not fake infinite context
Direct lesson from source:
- hierarchy + beam pruning helps reduce search cost
- but is still approximate

Apply in GoHarness:
- use hierarchical narrowing conceptually
- do not market it as true infinite context
- do not expose routing knobs unless runtime really honors them

---

# 7. Concrete GoHarness implementation slices

## Slice 1 — Epoch manifests on compaction
Smallest shippable backend improvement.

### Add
- epoch manifest write on successful compaction
- unit derivation from archived turn files
- workspace catalog update

### Success criterion
After compaction, the session contains:
- summary file
- epoch manifest
- unit text/meta files

---

## Slice 2 — Archived-memory query over current session

### Add
- query archived memory in current session only
- return top units with provenance
- no cross-session traversal yet

### Success criterion
A user prompt can retrieve relevant archived memory from the same session after compaction.

---

## Slice 3 — Workspace-scoped archived recall

### Add
- workspace catalog lookup
- query across sibling sessions/branches
- scope control

### Success criterion
Cross-session recall works within one workspace without requiring embeddings.

---

## Slice 4 — Memory provenance UI support

### Add
- API endpoints for retrieval metadata
- expose selected source summaries and omitted counts

### Success criterion
Frontend can show what memory was searched/selected.

---

## Slice 5 — Event-log draft substrate

### Add
- draft event schema
- event append path in parallel with current writes
- internal replay test harness

### Success criterion
The project can start migrating transcript/tree/memory projections onto one canonical substrate.

---

# 8. Testing plan

## 8.1 Storage tests

- epoch manifests created on compaction
- unit files created with valid provenance
- workspace catalog rebuild deterministic

## 8.2 Retrieval tests

- session-local archived retrieval returns expected units
- cross-session retrieval respects workspace scope
- retrieval deduplicates overlapping units
- retrieval obeys budget

## 8.3 Prompt assembly tests

- archive text is bounded
- provenance labels retained
- recent turns not displaced by cold memory overrun

## 8.4 Migration tests

- old sessions still load
- event-log dual-write can rebuild same projections
- rollback/branch semantics preserved

---

# 9. Operational/UI consequences

This system should eventually power UI surfaces such as:

- details panel history/memory tab
- session tree / time travel navigator
- transcript grouping around compaction boundaries
- archived-memory provenance strips

But those are **projections**.
The backend system should ship before their UI gets too clever.

---

# 10. Risks and anti-goals

## Risks

1. building retrieval directly against today's file layout too rigidly
2. confusing compaction summary with complete long-term memory
3. overstuffing prompts with archived evidence
4. leaking unrelated workspace/session data into the wrong scope

## Anti-goals

1. do not add embeddings as a requirement for v1
2. do not claim infinite context
3. do not hide retrieval behavior from the user
4. do not replace canonical session truth with opaque memory caches

---

# 11. Recommended first implementation order

1. **Slice 1** — epoch manifests + memory-unit derivation on compaction
2. **Slice 2** — session-local archived retrieval API
3. **Slice 3** — workspace-scoped archived recall
4. **Slice 4** — provenance API for UI
5. **Slice 5** — canonical event-log draft substrate

That sequence keeps the system useful early while still pointing toward the cleaner long-term architecture.

---

# 12. Success definition

This system is succeeding when GoHarness can truthfully say:

- old context is compacted and archived,
- archived context is searchable,
- recalled memory is provenance-aware,
- cross-session recall is possible under scope rules,
- and none of that required abandoning pure-Go, file-backed, local-first design.
