# 🧠 GoHarness Hierarchical Archived Memory Spec

> **Status:** Draft implementation spec
> **Scope:** Roadmap items `9.3`, `11.9`, and part of the foundation for `9.1`, `9.2`, `11.6`, and `12.2`
> **Primary reference input:** [`docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md`](./INFINITE_CONTEXT_MEMORY_RESEARCH.md)

This document defines a **GoHarness-native memory architecture** for archived conversation history.

It is intentionally designed to:

- stay **pure Go**,
- stay **single-binary / embedded-assets** at runtime,
- stay **local-first**,
- avoid fake “infinite context” claims,
- and integrate cleanly with the existing:
  - per-turn JSON persistence,
  - compaction model,
  - BM25 retrieval engine,
  - session/branch history,
  - deliverables and uploads.

The core idea is simple:

> recent turns stay hot, compacted history becomes warm, and older archived history becomes **cold memory** that is retrieved selectively under budget.

---

# 1. Why this spec exists

GoHarness already has several memory-like mechanisms:

- raw turn-by-turn session persistence,
- sliding-window compaction,
- archived compacted turn folders,
- uploaded documents,
- pinned context,
- deliverables,
- BM25 over workspace/session files.

What it lacks is a **coherent archived-memory retrieval model**.

Right now, once history is compacted, the main long-lived memory surface becomes:

- one summary blob,
- plus whatever the user manually restages or the agent re-reads.

That is truthful, but lossy.

This spec introduces a better cold-memory model:

- keep summaries,
- preserve archived raw structure,
- derive retrievable memory units,
- search those units hierarchically,
- inject only what fits the prompt budget,
- and show the user exactly what was recalled.

---

# 2. Goals

## 2.1 Primary goals

1. **Make compacted history retrievable without dumping everything back into prompt context.**
2. **Preserve provenance** for every recalled memory fragment.
3. **Prefer lexical/metadata retrieval first** so the system remains local-first and dependency-light.
4. **Exploit GoHarness's existing structure**:
   - sessions
   - branches
   - compaction epochs
   - uploads
   - deliverables
   - file paths
5. **Stay honest**: retrieval is not infinite context; it is selective recall.

## 2.2 Secondary goals

1. Enable **cross-session memory** within a workspace.
2. Make future embedding/rerank support possible **without requiring it for v1**.
3. Set up a storage and retrieval model that can later be migrated onto an event-log substrate.

---

# 3. Non-goals

This spec explicitly does **not** aim to:

1. make the model truly reason over unlimited tokens,
2. add a Python runtime,
3. require vector embeddings in v1,
4. replace existing per-turn JSON persistence,
5. hide retrieval behavior from the user,
6. solve general agent memory across all workspaces globally in one step,
7. turn GoHarness into a graph-RAG system.

---

# 4. Design principles

## 4.1 Retrieval is not memory magic

All archived recall must be framed as:

- retrieval,
- ranking,
- budgeted reinjection,
- provenance-aware prompting.

Not:

- “the model remembers everything”.

## 4.2 Derived storage, not source-of-truth replacement

The canonical source of conversational state remains:

- per-turn JSON files,
- compaction artifacts,
- session metadata,
- backups,
- uploads,
- deliverables.

Archived memory indexes are **derived projections**.

## 4.3 BM25 first

v1 should use:

- BM25
- metadata filters
- recency weighting
- hierarchical narrowing

Embeddings are explicitly optional future work.

## 4.4 Hierarchy should reflect real agent structure

Do not use arbitrary chunk buckets alone.

Prefer boundaries grounded in actual GoHarness semantics:

- workspace
- session / branch
- compaction epoch
- uploaded document
- deliverable group
- future event/topic projections

## 4.5 User-visible truthfulness

If archived memory was searched or injected, the UI should tell the user:

- what sources were searched,
- what sources were selected,
- what was omitted due to budget.

---

# 5. Memory model: hot, warm, cold

## 5.1 Hot memory

Always close to the active prompt.

Includes:

- system instructions,
- active environment prompt,
- recent raw turns,
- staged files,
- pinned context,
- immediate workflow/tool artifacts.

## 5.2 Warm memory

Session-local compressed state and near-history support.

Includes:

- active compaction summary,
- latest deliverables,
- current uploads,
- recent archived epochs if explicitly referenced.

## 5.3 Cold memory

Durable archived memory requiring retrieval.

Includes:

- archived raw turn chunks,
- older compaction epochs,
- older session/branch histories in the same workspace,
- uploaded documents not currently staged,
- future structured memory notes.

---

# 6. Core abstraction: Memory Units and Epochs

## 6.1 Memory Unit

A **Memory Unit** is the smallest retrievable archived-memory fragment.

It must have:

- stable ID,
- searchable text,
- metadata,
- provenance,
- bounded size.

### Proposed schema

```json
{
  "memory_id": "mem_ws123_sess456_epoch042_unit0007",
  "workspace_id": "ws_abc123",
  "session_id": "sess_20260912-170452",
  "branch_id": "sess_20260912-170452",
  "epoch_id": "epoch_000042",
  "source_kind": "raw_turn",
  "role": "assistant",
  "turn_start": 37,
  "turn_end": 37,
  "created_at": "2026-09-12T17:05:00Z",
  "path_refs": ["src/workflow.go"],
  "artifact_refs": [],
  "summary_text": "Assistant explained why workflow terminal output was duplicated.",
  "lexical_text": "Full retrievable text body goes here...",
  "provenance": {
    "session_file": ".goharness/sessions/sess_20260912-170452/compacted_summary_up_to_turn_042/037-assistant-2026_09_12_17-05-00.json"
  }
}
```

## 6.2 Epoch

An **Epoch** is a bounded archived-memory group.

For v1, the natural epoch boundary is:

- one compaction boundary.

So when turns `1..42` are compacted, that becomes:

- one archived epoch,
- one epoch summary,
- many derived memory units.

### Epoch responsibilities

An epoch must contain:

- manifest metadata,
- a summary artifact,
- the derived unit files,
- links back to original archived raw turns.

### Proposed epoch schema

```json
{
  "epoch_id": "epoch_000042",
  "workspace_id": "ws_abc123",
  "session_id": "sess_20260912-170452",
  "branch_id": "sess_20260912-170452",
  "turn_start": 1,
  "turn_end": 42,
  "summary_path": "summary.md",
  "unit_count": 31,
  "source_paths": [
    ".goharness/sessions/sess_20260912-170452/compacted_summary_up_to_turn_042/"
  ],
  "created_at": "2026-09-12T17:06:12Z"
}
```

---

# 7. Proposed storage layout

This spec intentionally keeps storage **file-based** and human-inspectable.

## 7.1 Session-local archived memory

```text
.goharness/sessions/<session_id>/
├── meta.json
├── 001-user-....json
├── 002-assistant-....json
├── compacted_summary_up_to_turn_042.json
├── compacted_summary_up_to_turn_042/
│   ├── 001-user-....json
│   ├── 002-assistant-....json
│   └── ...
└── memory/
    ├── manifest.json
    └── epochs/
        └── epoch_000042/
            ├── manifest.json
            ├── summary.md
            ├── summary.meta.json
            └── units/
                ├── 0001-user.txt
                ├── 0001-user.meta.json
                ├── 0002-assistant.txt
                ├── 0002-assistant.meta.json
                └── ...
```

## 7.2 Workspace-level catalog

For cross-session retrieval, add a workspace catalog:

```text
.goharness/memory/workspaces/<workspace_hash>/
├── catalog.json
└── sessions/
    ├── <session_id>.json
    └── ...
```

The catalog should reference:

- known sessions/branches for that workspace,
- which epochs exist,
- timestamps,
- session names,
- summary paths.

This does **not** duplicate leaf content. It indexes references.

---

# 8. File formats

## 8.1 Why use `.txt` plus `.meta.json`

The current GoHarness BM25 engine already indexes text files well.

So for v1, each retrievable memory unit should be stored as:

- one text file for lexical indexing,
- one adjacent metadata JSON file.

Example:

```text
units/
├── 0007-assistant.txt
└── 0007-assistant.meta.json
```

This is deliberately simple:

- no database required,
- no binary vector index required,
- inspectable on disk,
- compatible with existing indexing patterns.

## 8.2 Summary representation

Each epoch summary should also be stored in searchable text form:

- `summary.md`
- `summary.meta.json`

That allows coarse retrieval over summaries before descending to units.

---

# 9. Unit derivation rules

## 9.1 Raw user/assistant turns

Default rule:

- one memory unit per turn
- unless the turn is too large

If a turn exceeds a configured size threshold, split it into subunits by:

- paragraph
- logical block
- or line window (for large tool-like assistant content)

## 9.2 Tool turns

Tool turns should **not** be naively injected wholesale.

Rules:

- store them as units if they carry real evidence
- prefer shorter derived text over raw huge output
- prioritize:
  - read results
  - search results
  - patch/write outcomes
  - artifact references
- de-prioritize giant stdout blocks unless explicitly requested

## 9.3 Uploads

Uploads should become memory units with:

- `source_kind = upload`
- file path provenance
- optional chunking by paragraphs/sections

## 9.4 Deliverables

Deliverables should be represented as retrievable memory units with:

- output file path
- originating session/turn
- artifact type
- possibly extracted summary text

---

# 10. Retrieval pipeline

This is the core behavioral spec.

## 10.1 Inputs

A retrieval request must accept at least:

```json
{
  "workspace_id": "ws_abc123",
  "session_id": "sess_20260912-170452",
  "query_text": "What did we decide about workflow terminal duplicates?",
  "scope": "session_or_workspace",
  "max_epochs": 4,
  "max_units": 8,
  "char_budget": 12000
}
```

## 10.2 Stage 1: coarse narrowing

Search/select likely relevant groups first.

Possible groups:

- current session's archived epochs
- sibling branches of the same session family
- other sessions in the same workspace
- upload groups
- deliverable groups

Scoring factors:

1. summary BM25 score
2. metadata/path match
3. recency
4. session affinity
5. branch affinity

Output:

- shortlist of candidate epochs / groups

## 10.3 Stage 2: leaf retrieval

Within shortlisted groups:

- BM25 search over unit `.txt` files
- rank units by lexical relevance
- optionally blend with metadata boosts

### Suggested v1 scoring

```text
unit_score =
  bm25_score
  + recency_boost
  + session_affinity_boost
  + path_ref_boost
  + artifact_ref_boost
```

## 10.4 Stage 3: dedup + budget assembly

The retrieval result is not just “top k units”.

It must:

- deduplicate overlapping units,
- prefer diversity across sources,
- prevent one giant artifact from consuming everything,
- obey a hard prompt budget.

---

# 11. Beam search analogue for GoHarness

## 11.1 What we borrow conceptually

From `infinite-context`, the valuable idea is:

- prune the search frontier at each hierarchy level,
- descend only into promising branches.

## 11.2 What we should do in v1

For GoHarness v1, we do **not** need a literal HAT implementation.

Instead, implement a simpler **hierarchical frontier search**:

1. score candidate epochs/groups,
2. keep top N groups,
3. search only those groups' units.

That already captures most of the practical benefit.

## 11.3 Future v2

If needed later, we can formalize a real beam-style search over:

- workspace
- session/branch
- epoch
- unit

At that point we can expose an internal parameter like:

- `frontier_width`

But do **not** expose it in user-facing config/UI until runtime truly needs and honors it.

---

# 12. Prompt assembly contract

## 12.1 Output format

Retrieved memory should be assembled into a structured block like:

```markdown
## Retrieved archived memory

### Source 1
- workspace: game_proj
- session: sess_20260912-170452
- epoch: 1-42
- type: raw_turn
- turns: 37-37

Assistant explained why workflow terminal output was duplicated.

### Source 2
- workspace: game_proj
- session: sess_20260912-170452
- epoch: 1-42
- type: compacted_summary

Summary said the terminal node was broadcasting twice and the root user turn was not persisted.
```

## 12.2 Budget policy

Suggested default budget split:

1. recent raw turns — highest priority
2. active compaction summary — next priority
3. retrieved archived memory — bounded pool
4. staged files/uploads — bounded pool

Example policy:

- total augmentation budget: `12k chars`
- archived memory sub-budget: `4k–6k chars`

## 12.3 Truthfulness requirement

Whenever archived memory is injected, the system should know and be able to surface:

- how many sources were searched
- which were selected
- how many were omitted by budget

---

# 13. Internal Go APIs

This spec proposes the following internal shapes.

## 13.1 Core types

```go
type MemoryUnit struct {
    MemoryID      string
    WorkspaceID   string
    SessionID     string
    BranchID      string
    EpochID       string
    SourceKind    string
    Role          string
    TurnStart     int
    TurnEnd       int
    CreatedAt     time.Time
    PathRefs      []string
    ArtifactRefs  []string
    SummaryText   string
    LexicalText   string
    Provenance    map[string]string
}

type MemoryEpoch struct {
    EpochID      string
    WorkspaceID  string
    SessionID    string
    BranchID     string
    TurnStart    int
    TurnEnd      int
    SummaryPath  string
    UnitCount    int
    CreatedAt    time.Time
}

type MemoryQuery struct {
    WorkspaceID   string
    SessionID     string
    QueryText     string
    Scope         string
    MaxEpochs     int
    MaxUnits      int
    CharBudget    int
}

type MemoryRetrievalResult struct {
    EpochsSearched int
    UnitsSearched  int
    UnitsSelected  []MemoryUnit
    OmittedCount   int
    AssembledText  string
}
```

## 13.2 Core functions

```go
func BuildSessionMemoryEpoch(sessionID string, boundaryTurn int) error
func RebuildWorkspaceMemoryCatalog(workspaceDir string) error
func RetrieveArchivedMemory(q MemoryQuery) (MemoryRetrievalResult, error)
func AssembleArchivedMemoryPrompt(result MemoryRetrievalResult, budget int) string
```

---

# 14. Lifecycle hooks

## 14.1 On compaction

When compaction succeeds:

1. write the normal compacted summary,
2. archive raw turns as today,
3. derive/update the corresponding `memory/epochs/epoch_xxx/` directory,
4. update the workspace memory catalog.

## 14.2 On session switch

No expensive rebuild by default.

Just ensure the workspace catalog is discoverable and lazy-load retrieval metadata when needed.

## 14.3 On upload

Optionally derive upload memory units immediately, or lazily on first retrieval.

## 14.4 On deliverable creation

Optionally register/update deliverable memory units for future recall.

---

# 15. Migration plan

## 15.1 v1 substrate

Use the **current per-turn JSON + compaction folder** model as the source of truth.

This means `9.3` can ship **before** canonical event-log migration if we keep the archived-memory projection derived.

## 15.2 v2 substrate

When `11.6` + `12.2` land:

- keep the same external retrieval contract,
- change the projection builder to derive units from the event log instead of raw turn folders.

So the archived-memory feature should depend on a **projection interface**, not hardwire itself forever to the current turn-file layout.

---

# 16. UI implications

## 16.1 Retrieval visibility

The UI should later expose:

- retrieved archived-memory chips,
- source provenance,
- selected session/epoch/turn ranges,
- omitted-by-budget counts.

## 16.2 No fake control surfaces

Do not expose controls like:

- beam width
- routing mode
- “infinite memory” toggle

unless runtime really supports clear, meaningful differences.

## 16.3 Good future surfaces

Likely homes:

- Details panel → History / Memory tab
- Composer source strip
- Session/trajectory drilldown surfaces

---

# 17. Security and trust rules

Archived memory retrieval must respect the same safety principles as the rest of GoHarness.

## 17.1 Read scope

It may read:

- session turn files
- compacted summaries
- upload artifacts
- deliverable artifacts
- memory projection files

It must not silently expand into unrelated system paths.

## 17.2 Provenance integrity

Every injected memory fragment must be traceable back to:

- session ID
- epoch ID
- turn range or file origin

## 17.3 No hidden mutation

Retrieval is read-only.

Memory projection updates happen only through:

- compaction lifecycle
- explicit rebuild operations
- upload/deliverable registration flows

---

# 18. Testing and acceptance criteria

## 18.1 Storage tests

- compaction produces epoch manifests and unit files
- unit metadata points back to valid provenance
- workspace catalog rebuild is deterministic

## 18.2 Retrieval tests

- query returns expected epoch shortlist
- query returns relevant units within budget
- no duplicate units in final assembly
- cross-session retrieval respects scope

## 18.3 Prompt assembly tests

- budget is never exceeded
- provenance labels remain intact
- recent-turn context is not evicted by archived memory

## 18.4 UX truthfulness tests

- if archived memory is used, result metadata exists
- omitted-by-budget counts are surfaced correctly

---

# 19. Suggested implementation waves

## Wave 1 — foundational v1

- memory unit schema
- epoch builder on compaction
- workspace memory catalog
- BM25 search over summaries and units
- archived-memory prompt assembly

## Wave 2 — better selection quality

- metadata boosts
- recency weighting
- session/branch affinity weighting
- upload/deliverable inclusion

## Wave 3 — richer UX

- visible archived-memory source strip
- memory/history inspector
- explicit retrieval diagnostics

## Wave 4 — optional advanced retrieval

- embedding or rerank path
- explicit hierarchical frontier pruning knobs internally
- event-log-backed projection rebuilds

---

# 20. Final recommendation

GoHarness should adopt the **hierarchical archived-memory retrieval pattern** from adjacent research, but implement it in a way that matches GoHarness values:

- pure Go
- local-first
- single binary
- file-backed and inspectable
- BM25-first
- explicit provenance
- strict budgeting
- no fake “infinite context” story

That gives us the actual benefit we want:

> better cold-memory recall over archived work, without bloating runtime dependencies or lying about what the model truly sees.
