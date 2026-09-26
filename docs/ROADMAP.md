# 🧭 GoHarness Roadmap

This document is the execution-grade roadmap for GoHarness, and it is the only roadmap file in the project. It is meant to be both complete and aware of what implementation actually requires.

It brings together the information that used to be scattered across separate notes:

- what the product is trying to do,
- what we learned from competing and adjacent tools,
- how ready each piece of work really is,
- which pieces depend on which other pieces,
- which areas still need a written specification,
- how the work groups into systems rather than a flat list of features,
- and which research or specification document justifies each item.

Other documents under `docs/` remain useful, but they are reference material. They are not competing sources of roadmap truth.

---

## Status key

- **Shipped** — already implemented.
- **Partial** — first cut shipped; follow-on work remains.
- **Ready** — enough information exists to implement a solid v1 now.
- **Blocked** — enough information exists, but another roadmap item should land first.
- **Spec needed** — concept is clear, but the data model / lifecycle semantics / migration plan are not ready.
- **Deferred** — intentionally lower priority even if technically specifiable.
- **Subsumed** — do not implement as a separate project; fold it into the newer item named in the notes.

## Implementation role key

This roadmap now groups rows by **implementation system** and labels each row by what kind of work it really is:

- **Substrate** — storage/runtime primitives other features sit on top of.
- **Retrieval** — search/memory/query logic.
- **Policy** — permissions, trust, hooks, approvals, safety rules.
- **Projection/UI** — user-facing surfaces built on other systems.
- **Operator** — control-plane, CLI, headless, status, or maintenance surfaces.
- **Integration** — bounded feature that depends on multiple substrates but is not itself a new platform.

## Reference document key

The roadmap now explicitly tracks how items relate to research/spec documents.

- **Frontend plan** — `docs/FRONTEND_REFACTOR_PLAN.md`
- **Mockup** — `docs/mockups/ui-mockup.html`
- **Research ledger** — `docs/RESEARCH.md`
- **Comparison matrix** — `docs/COMPARISON_MATRIX.md`
- **BM25 research** — `docs/BM25_SCALING_RESEARCH.md`
- **Infinite-context memory research** — `docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md`
- **Archived memory spec** — `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md`
- **Character card design** — `docs/CHARACTER_CARD_PERSONA_AND_MEMORY.md`
- **Character card guide** — `docs/character-cards/IMPLEMENTATION_GUIDE.md`
- **Subagent plan** — `docs/SUBAGENT_PARALLELISM_PLAN.md`
- **V2 spec** — `docs/V2_SPECIFICATION.md`
- **V2 visual editor** — `docs/V2_VISUAL_EDITOR.md`
- **Execution plans** — `docs/<SYSTEM_NAME>_EXECUTION_PLAN.md` (indexed under [System execution plans](#system-execution-plans))
- **Roadmap synthesis** — backlog/architecture synthesis kept in this roadmap itself
- **Code** — current implementation under `src/`

---

## Supporting documents (reference material, not roadmap files)

- `docs/mockups/ui-mockup.html` — post-roadmap UI direction mockup
- `docs/FRONTEND_REFACTOR_PLAN.md` — web UI maturation + ES Modules frontend split plan
- `docs/RESEARCH.md` — bibliography / inspiration ledger
- `docs/COMPARISON_MATRIX.md` — older broad comparative notes
- `docs/GIT_HISTORY.md` — implementation and documentation chronology
- `docs/SUBAGENT_PARALLELISM_PLAN.md` — concurrency design notes for the shipped sub-agent runtime
- `docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md` — source-level evaluation of `Lumi-node/infinite-context` and its applicability to GoHarness memory architecture
- `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md` — implementation spec for GoHarness cold-memory retrieval over archived turns, summaries, uploads, and deliverables
- `docs/CHARACTER_CARD_PERSONA_AND_MEMORY.md` — design proposal for using Character Card V3 as a persona source, an evidence kind, and a lorebook-shaped retrieval and memory layer
- `docs/character-cards/` — an implementation-agnostic guide to reading Character Card V1/V2/V3 files, and a working reference implementation of it

---

## Why the roadmap is organized this way now

The older roadmap was useful while we were still discovering the shape of the work, but many of its rows described the same underlying thing. A typical row was really three or four projects sharing one name: some storage or runtime primitive, plus a layer of policy, plus a user-facing surface, plus a surface for operators and maintenance.

If we build those rows one at a time as independent features, we will keep rebuilding the same state models, and then spend later effort reconciling them. So this roadmap is organized by implementation system instead of by feature idea.

### The nine implementation systems

1. **Session Event & Memory System**
2. **Knowledge Ingress & Retrieval System**
3. **Task & Background Work System**
4. **Capabilities & Extensions System**
5. **Policy & Hooks Engine**
6. **Execution & Review System**
7. **MCP Runtime Expansion**
8. **Runtime, Config & Operator Control Plane**
9. **Shell & Interaction System**

This is a refinement of an earlier consolidation pass. Two areas that used to be folded into other systems are now listed separately, because each is large enough to justify its own program of work:

- **Knowledge Ingress & Retrieval** covers everything about getting knowledge from outside the session into the session, and making it findable again.
- **Runtime, Config & Operator Control Plane** covers how configuration layers together and how operators interact with the tool.

---

## System execution plans

Each of the nine systems gets exactly one execution plan. The two kinds of document have different jobs:

- This roadmap is the map. It says what a system owns, which rows belong to it, what blocks what, and which research or specification document justifies each row.
- The execution plan is the program. It records where the design came from, including which reference projects were read at source level. It then lays out phases, concrete slices of work, tests, risks, and a definition of what success looks like.

| # | System | Execution plan | Plan status | Plan shape |
|---|---|---|---|---|
| 1 | Session Event & Memory | [`docs/SESSION_EVENT_MEMORY_EXECUTION_PLAN.md`](./SESSION_EVENT_MEMORY_EXECUTION_PLAN.md) | **Written** | 6 phases (A–F) · 6 slices |
| 2 | Knowledge Ingress & Retrieval | [`docs/KNOWLEDGE_INGRESS_RETRIEVAL_EXECUTION_PLAN.md`](./KNOWLEDGE_INGRESS_RETRIEVAL_EXECUTION_PLAN.md) | **Written** | 14 phases (A–N) · 12 slices |
| 3 | Task & Background Work | [`docs/TASK_BACKGROUND_WORK_EXECUTION_PLAN.md`](./TASK_BACKGROUND_WORK_EXECUTION_PLAN.md) | **Written** | 8 phases (A–H) · 7 slices |
| 4 | Capabilities & Extensions | _pending_ | Wave 4 | — |
| 5 | Policy & Hooks Engine | _pending_ | Wave 3 | — |
| 6 | Execution & Review | _pending_ | Wave 1 + Wave 3 | — |
| 7 | MCP Runtime Expansion | _pending_ | Wave 3 | — |
| 8 | Runtime, Config & Operator Control Plane | _pending_ | Wave 1 | — |
| 9 | Shell & Interaction | _pending_ | Wave 1 (largest system) | — |

A plan's status is either **Written** or **Pending**. A pending plan is not a gap in the roadmap. It means the system is specified well enough that its `Ready` rows can still be worked on, but nobody has written the program document yet.

**Row coverage.** Each plan begins with a coverage map. For every roadmap row the plan claims, the map names the phase that owns it and states how deep the coverage actually is. The possible values are `core`, `partial`, `reference`, and `gap`. This map is the authoritative trace from a roadmap row to a phase, and it is allowed to say `gap`. Knowing that a row was claimed but never written up is more useful than assuming it was handled.

**Deliberate cross-listings.** One row appears in two systems on purpose, because splitting it would lose information. Row `12.28.12` covers deliverables, references, and message feedback. Knowledge Ingress & Retrieval owns the staged evidence, which is what a deliverable or reference becomes once it has been ingested. Shell & Interaction owns the surface that renders it. Both table entries say this explicitly.

Every other row appears in exactly one system. If you find a row in two systems without a note like the one above, that is a defect in this roadmap rather than a deliberate overlap.

**How to read a system section**

- The **Items** table is that system's master backlog. Each row records its role, its status, the documents it relates to, and what it depends on.
- The **Execution plan** blockquote points at the program document. It repeats the decisions the plan has already made and the honest notes about what the plan does not yet cover.
- Rows marked **Subsumed** are historical. They are implemented through the newer item named in the table, never as a separate project.

---

## Current shipped baseline

These foundations have already landed. They should not be re-specified as open work:

- Visual DAG workflow editor
- Every active workflow routed through the DAG executor
- `llm` workflow node support for tool-calling ReAct loops
- Provider profiles in `providers.json`
- Per-profile concurrency throttling
- Environment-aware shell guidance
- Foldable tool calls/results
- Workflow CRUD in the Lab
- Parallel sub-agents with depth cap, write lock, and progress SSE
- Comparative roadmap research for Pi / DeepSeek Harness / Codex / Claude Code
- UI mockup in `docs/mockups/ui-mockup.html`
- Embedded HTML partial composition + ES module web client refactor
- No inline HTML event-handler wiring in the current web shell

### Partially landed foundations with follow-on work still open

| ID | Item | Role | Status | Related docs | Notes |
|---|---|---|---|---|---|
| 12.6 | Tool-output spill to disk | Substrate | **Partial** | Frontend plan, Roadmap synthesis, Code | Storage exists; richer viewers and retrieval integration remain open. |
| 12.28.14 | Truthful block/empty states | Projection/UI | **Partial** | Frontend plan, Mockup, Roadmap synthesis | Truthful UI pattern shipped in parts; needs broader coverage. |
| 12.29.12 | Theme token foundation | Projection/UI | **Partial** | Frontend plan, Mockup | Token groundwork exists; full theme/operator story remains open. |

---

## System dependency backbone

These are the major cross-system truths that should shape implementation order.

### A. Event/memory first for durable introspection
The **Session Event & Memory System** is the long-term substrate for:
- archived-memory retrieval,
- session tree/time travel,
- transcript grouping,
- history query,
- memory dashboards.

### B. Task state first for goals/schedules/background work
The **Task & Background Work System** must exist before:
- goal mode,
- schedule/mission runs,
- durable background workers,
- roster/progress UI can become truthful.

### C. Policy engine first for approvals/hooks/permissions
The **Policy & Hooks Engine** is the clean backbone for:
- approvals,
- execpolicy,
- trust review,
- matcher hooks,
- capability restrictions.

### D. Execution substrate first for review/isolation flows
The **Execution & Review System** should own:
- persistent terminals,
- validation loops,
- code/review modes,
- worktree-aware isolation.

### E. Shell/UI is mostly projection, not backend truth
The **Shell & Interaction System** should consume stable projections from the other systems instead of inventing its own state models.

---

# 1) Session Event & Memory System

> **Execution plan:** [`docs/SESSION_EVENT_MEMORY_EXECUTION_PLAN.md`](./SESSION_EVENT_MEMORY_EXECUTION_PLAN.md) · 6 phases (A–F) · 6 slices
> **Decisions the plan has already made:** the file-backed derived projection ships first, through phases B to D, rather than waiting for the canonical event log to exist. Compaction writes epoch manifests. Recalled memory always carries provenance. The plan also refuses the phrase "infinite context" and describes what it actually does instead, which is bounded frontier retrieval.
> **Honest coverage:** every row the plan claims now has a phase that owns it. Rows `9.3`, `11.6`, and `11.9` are covered in full. Rows `9.1`, `9.2`, `12.2`, `12.29.7`, and `12.18` are covered only partly: the plan defines the storage or the API, but not the user-facing surface. Row `12.18` is specified only at the level of whole artifacts, so filtering by event still has to wait for Phase F.
> **Open decision:** whether Phase F and Slice 6 also satisfy this roadmap's request for a *session event log and migration spec*, or whether that document is still owed separately.

## System intent

This system owns:
- canonical session persistence semantics,
- time-travel and branch graph structure,
- compaction/archive boundaries,
- archived-memory retrieval,
- cross-session recall,
- transcript/event projections.

## Why these items belong together

The archived-memory work is now informed by:
- **OptMem-style range and hierarchy ideas** in the older research ledger,
- **BM25-first retrieval evidence** in `docs/BM25_SCALING_RESEARCH.md`,
- the source-level cautionary evaluation in `docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md`,
- and the concrete GoHarness design in `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md`.

That means these rows are no longer isolated “memory ideas”; they are one implementation system with staged substrates and projections.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 9.1 | O(1) range loader + hierarchical memory decay | Substrate | **Spec needed** | Research ledger, Infinite-context memory research, Archived memory spec | 11.6, 12.2 improve it later | Needs canonical storage/projection contract and prompt reconstruction rules. Archived-memory spec now defines the cold-memory direction, but the exact substrate still needs design if done on top of an event log instead of files. |
| 9.2 | Visual memory map dashboard | Projection/UI | **Spec needed** | Archived memory spec, Frontend plan, Mockup | 9.1, 12.2, shell placement | Needs backing API, retrieval provenance model, and user actions like inspect/forget/branch. |
| 9.3 | Hierarchical archived-memory retrieval (BM25-first, optional embeddings later) | Retrieval | **Ready** | Infinite-context memory research, Archived memory spec, BM25 research | 9.1 strongly related; 11.6 and 12.2 improve it later but are not required for a file-backed v1 | Implement as a file-backed derived projection first. Embeddings are optional follow-on work. |
| 11.6 | Session tree / time-travel navigator | Projection/UI | **Spec needed** | Roadmap synthesis, Frontend plan | 12.2 strongly related | Needs canonical branch/session graph semantics, projection format, and migration path from plain turn files. |
| 11.9 | Cross-session memory | Retrieval | **Ready** | Archived memory spec, BM25 research, Infinite-context memory research | — | Good v1 path: workspace-scoped BM25/metadata retrieval over archived memory units. Do not overclaim this as infinite context. |
| 12.2 | Session event log as source of truth | Substrate | **Spec needed** | Archived memory spec, Frontend plan, Roadmap synthesis | 11.6 strongly related | Needs exact event schema, projections, migration/dual-write plan. This is the long-term substrate for the rest of the system. |
| 12.18 | Session query with FTS | Retrieval | **Blocked** | Archived memory spec, Frontend plan | 11.6, 12.2 | Easy once event/query projections exist. |
| 12.29.7 | Full step-grouped transcript / compaction placement / streaming-tail isolation | Projection/UI | **Spec needed** | Frontend plan, Archived memory spec | 12.2 | Wants a typed event/transcript model instead of ad hoc chat card stacking. |

**Cross-system note:** `12.23` (human feedback out of model context) and `12.25` (telemetry / observability
contracts) are operator-side rows and are listed **only** in the Runtime, Config & Operator Control Plane.
The event/transcript schema defined here must stay aligned with `12.25`, and feedback must remain a sidecar
projection per `12.23` — not model-visible conversation text.

---

# 2) Knowledge Ingress & Retrieval System

> **Execution plan:** [`docs/KNOWLEDGE_INGRESS_RETRIEVAL_EXECUTION_PLAN.md`](./KNOWLEDGE_INGRESS_RETRIEVAL_EXECUTION_PLAN.md) · 14 phases (A–N) · 12 slices
> **Decisions the plan has already made:** every source produces the same kind of record, an `EvidenceRecord` that carries where the content came from, its content hash, and how far it should be trusted. This means the surfaces that bring knowledge in can differ, while the storage underneath stays the same. Row `12.19` generalizes the sha256 addressing that `src/spill.go` already uses, rather than introducing a second way of storing things. Staging and injection are separate states: nothing reaches the prompt unless something explicitly puts it there, and injected evidence is delimited and attributed. Content fetched from the internet is marked `untrusted_external` and can never take the position of an instruction. Retrieval returns provenance, offsets, and trust instead of just a path and a score, and the search index is cached and invalidated by modification time, replacing today's full re-walk on every call. Row `11.16` ports the instruction-discovery behaviour from Codex: walk up to a project-root marker, read files from the root down to the working directory, honour an override file, respect a byte budget, and never walk past the root. Images are addressed by their path on disk and delivered to the model as renderings, with an explicit detail setting.
> **Honest coverage:** rows `12.30.1`, `12.30.2`, `12.30.4`, and `12.30.5` are covered by phases J, K, M, and N. Row `12.30.3` is covered by phase L and stays `Blocked` until phase A builds the catalog it registers into. Row `12.30.1` has no dependency on the substrate at all, so it can be built first. Rows `11.1`, `11.4`, `11.16`, `12.19`, and `12.6` are covered by core phases. Row `12.5` is also core but deliberately sequenced last, because it is the largest item and nothing else waits on it. Row `11.19` is fully covered for getting images in, while the model's ability to see them is gated honestly per provider. Row `12.28.12` is partly covered: this plan owns the structured record, and the shell owns the rendering.
> **Open decisions recorded in the plan:** which web-search backend to use for `11.1`, whether uploaded PDFs and office documents get their text extracted, and where in the interface injected instructions are shown.
> **The card series is in the plan as phases J to N**, and it came from the design study in [`docs/CHARACTER_CARD_PERSONA_AND_MEMORY.md`](./CHARACTER_CARD_PERSONA_AND_MEMORY.md). The plan now says how a card gets in without a runtime parser, namely an offline converter with a manifest and the imported file kept in a `source/` directory beside the output. A persona is one labelled block in the system prompt, and it has a working position whether or not the prompt composer in row `12.30.8` exists yet. Two evidence kinds are added, `character_card` and `lorebook`. The lorebook contributes a matcher, a scan window, and a complete drop order for when the budget is exceeded, which is an answer this plan did not previously have. The agent gains a read and write memory surface whose content lives in ordinary files under an index that carries only the injection parameters. The guide that all of this rests on is in [`docs/character-cards/`](./character-cards/).

## System intent

This system owns:
- getting external knowledge into GoHarness,
- making local knowledge surfaces queryable,
- staging evidence for prompts,
- keeping retrieval truthful and inspectable.

## Why these items belong together

The research story here is consistent:
- `docs/BM25_SCALING_RESEARCH.md` argues strongly for lexical-first retrieval at scale,
- the frontend plan explains why staged evidence and typed search/read blocks belong near the composer and details surfaces,
- the current code already has uploads, BM25, pinned context, and workspace tree interaction.

These rows are different ingest/retrieval surfaces over the same knowledge system.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 11.1 | Web search & fetch | Retrieval | **Ready** | Research ledger, Frontend plan, Roadmap synthesis | Spill foundation already landed | One of the most ready items. Should feed the same evidence staging and typed renderer system as other sources. |
| 11.4 | `@file` mentions in composer | Projection/UI | **Ready** | Frontend plan, Mockup | — | UI typeahead + file resolver + staging contract. |
| 11.16 | Parent-directory context walking + overrides | Retrieval | **Ready** | Research ledger, current instruction loader, Roadmap synthesis | — | Natural extension of scoped instructions and workspace context discovery. |
| 11.19 | Image paste/drag into composer | Integration | **Ready** | Frontend plan, current upload plumbing | 12.19 improves backend | Can ship before content-addressing, but 12.19 cleans it up. |
| 12.5 | LSP seam | Substrate | **Ready** | Roadmap synthesis, Comparison matrix | — | Fits here because it is a knowledge/analysis source as much as an execution aid. |
| 12.6 | Tool-output spill to disk | Substrate | **Partial** | Frontend plan, Code | — | Should become one of the standard ingestable evidence forms. |
| 12.19 | Content-addressed attachments | Substrate | **Ready** | Roadmap synthesis, Frontend plan | — | Strong backend cleanup for uploads, images, and future evidence references. |
| 12.28.12 | Deliverables, references, message feedback | Projection/UI | **Ready** | Frontend plan, Mockup | 11.4 helps | This belongs here because deliverables and references are evidence surfaces, even though their UI sits in the shell system. **Deliberate cross-listing** with Shell & Interaction, which owns the rendering surface. |
| 12.30.1 | Card repository and per-session imports | Substrate | **Ready** | Character card design | — | One repository for the installation, and a session-level list of what it imported. Everything else in the `12.30.x` series depends on it. |
| 12.30.2 | Character cards as a persona source | Integration | **Ready** | Character card design, Character card guide | 12.30.1 | One card, delivered as standing instructions of the same kind as `AGENTS.md`. The persona is the block `12.30.8` orders rather than a prompt rule of its own, and it has a working default position when that row is absent. |
| 12.30.3 | Cards and lorebooks as evidence kinds | Substrate | **Blocked** | Character card design | Evidence substrate in this system's plan | Two values added to the existing kind enumeration. |
| 12.30.4 | Lorebook as a retrieval and injection layer | Retrieval | **Ready** | Character card design, Character card guide | 12.30.1 | The matching algorithm, the scan window, and the token budget. The first part of the series worth building. |
| 12.30.5 | Agent-maintained memory, files plus an index | Retrieval | **Ready** | Character card design | 12.30.4 | Read and write tools, every call logged. The content lives in ordinary files and a JSON index carries the injection parameters. |

---

# 3) Task & Background Work System

> **Execution plan:** [`docs/TASK_BACKGROUND_WORK_EXECUTION_PLAN.md`](./TASK_BACKGROUND_WORK_EXECUTION_PLAN.md) · 8 phases (A–H) · 7 slices
> **Decisions the plan has already made:** the queue becomes owned by the host, and the browser becomes a view onto it rather than the place where queued work actually lives. A single task state machine covers jobs, goals, and schedules, with states running from `queued` through `admitted` and `running` to `waiting_user` or `waiting_dependency`, and finally to a terminal state. For version one the job store is local to the session rather than a separate daemon. Scheduled work claims its slot before running and recovers properly if it is interrupted mid-claim.
> **Honest coverage:** every row the plan claims now has a phase that owns it. Rows `11.2`, `11.3`, `11.7`, `11.11`, `12.8`, `12.21`, `12.10`, `12.13`, `11.10`, `12.12`, and `10.2B` are covered by core phases or deliverables. Rows `11.8`, `12.9`, `12.11`, and `14.5` are referenced through Phase E. Rows `12.7`, `11.12`, `13.6`, and `14.8` land in Phase H as user-facing projections of the state underneath.

## System intent

This system owns:
- queued user work,
- agent/sub-agent task execution,
- durable/background jobs,
- blocked/waiting states,
- plans/goals/schedules as actual runtime state.

## Why these items belong together

The current sub-agent runtime already gave us the first slice of this system.
The next rows all depend on having one truthful model of:

- queued work,
- running work,
- waiting-user work,
- resumed work,
- cancelled/aborted work,
- and outputs/receipts.

That makes this one of the strongest consolidation candidates in the roadmap.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 10.2B | Wait First / Race | Integration | **Ready** | Subagent plan, Roadmap synthesis | — | Implement through current sub-agent/task runtime. |
| 10.2C | Async / Fire-and-Forget | Integration | **Subsumed** | Roadmap synthesis | 11.7 | Do not build separately; fold into durable background jobs. |
| 11.2 | Message queue while agent is running | Substrate | **Ready** | Frontend plan, Roadmap synthesis | — | This is one of the first visible faces of the task system. |
| 11.3 | Cancel vs abort distinction | Policy | **Ready** | Roadmap synthesis | — | Part of the task/job state model, not a standalone hack. |
| 11.7 | Durable/background sub-agent jobs | Substrate | **Ready** | Subagent plan, Roadmap synthesis, Prime references consolidated in roadmap | 11.2 helpful | Central substrate for async agent work. |
| 11.8 | Named sub-agent roles & packaged workflows | Integration | **Ready** | Subagent plan, Roadmap synthesis | — | Better once task metadata and role files are unified. |
| 11.10 | Automatic build/lint/typecheck feedback loop | Integration | **Ready** | Comparison matrix, Roadmap synthesis | — | Sits here because it is a recurring job/validation loop, not just a one-shot command. |
| 11.11 | Model-initiated Q&A interludes | Policy | **Ready** | Frontend plan, Roadmap synthesis | — | Should enter the same waiting-user task state used by approvals and blocked work. |
| 11.12 | Persistent TODO/plan overlay | Projection/UI | **Ready** | Frontend plan, Mockup | — | UI projection of task/plan state. |
| 12.7 | Plan mode as logged collaboration state | Integration | **Blocked** | Frontend plan, Mockup, Roadmap synthesis | 12.3, 11.11, waiting-state substrate | Behavior is clear, but it wants task and policy substrate first. |
| 12.8 | Goals with autonomous round-driving | Integration | **Blocked** | Roadmap synthesis, Prime references consolidated in roadmap | 11.2, 11.7 | Best built on a truthful durable job model. |
| 12.9 | Dynamic workflows over sub-agents | Integration | **Blocked** | Roadmap synthesis, current workflow runtime | 11.7, 12.10, 12.11 | Wants the durable sub-agent/task substrate first. |
| 12.10 | Multiple sub-agent providers | Integration | **Ready** | Roadmap synthesis, current provider/profile work | — | A task-system extension: job execution over different connection profiles/providers. |
| 12.11 | Structured sub-agent output | Integration | **Ready** | Subagent plan, Roadmap synthesis | — | Makes task receipts and downstream composition more reliable. |
| 12.12 | Per-agent tool scoping & personas | Policy | **Ready** | Roadmap synthesis, current agent runtime | — | Most truthful once roles and task contexts are unified. |
| 12.13 | Per-session agent presets | Operator | **Ready** | Roadmap synthesis | — | Better seen as session-scoped task runtime configuration. |
| 12.21 | Scheduled / mission runs | Integration | **Blocked** | Task execution plan, Roadmap synthesis | 11.7 | Scheduled work is a durable-job family, not a second async system; needs claim-before-run semantics and interrupted-claim recovery. Row was missing from the roadmap after the reorg while the execution plan already claimed it. |
| 13.6 | Plan mode + goal mode + progress rows | Projection/UI | **Blocked** | Frontend plan, Mockup | 12.7, 12.8, 11.7 | Integration layer over task state, not a separate substrate. |
| 14.5 | Subagents as data files | Substrate | **Ready** | Subagent plan, Roadmap synthesis | — | Strong fit for unifying named sub-agent roles and packaged definitions. |
| 14.8 | Background agents, task roster, queued command semantics | Projection/UI | **Blocked** | Frontend plan, Roadmap synthesis | 11.2, 11.7 | UI/operator surface over the same task substrate. |

---

# 4) Capabilities & Extensions System

> **Execution plan:** _pending._
> **What the plan still has to decide:** what a capability manifest contains and how scopes work, which covers row `12.14`. How an extension is trusted, installed, and updated, which covers rows `11.13C` and `13.4B`. And where the boundary sits between this capability registry and the trust decisions made by the policy engine.

## System intent

This system owns:
- prompt templates,
- skills,
- commands,
- lazy dynamic context injection,
- extension packaging,
- provider/model catalogs as data,
- scope/lifetime semantics for capabilities.

## Why these items belong together

These rows all want one coherent answer to:

- what a capability is,
- how it is declared,
- where it can be loaded,
- how it injects context,
- how it is trusted/installed,
- and how users discover it.

This should not become three separate mini-platforms for templates, skills, and plugins.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 11.13A | Prompt templates | Substrate | **Ready** | Roadmap synthesis, Research ledger | — | Smallest viable slice of the broader capability system. |
| 11.13B | Skills (description + lazy load) | Substrate | **Ready** | Roadmap synthesis, Research ledger | — | Good first real capability abstraction. |
| 11.13C | Full extension protocol / install surface | Substrate | **Spec needed** | Roadmap synthesis, Comparison matrix | 12.3, 12.14, trust model | Needs registration, trust, install/update semantics. |
| 11.14 | Provider/model catalog as data | Operator | **Ready** | Roadmap synthesis, current provider code | — | This is capability metadata and discovery, not just settings UI. |
| 12.14 | Scope primitive | Substrate | **Blocked** | Roadmap synthesis | 12.3, capability registry | Needed so capabilities have truthful lifetimes and visibility rules. |
| 12.24 | Skills as on-demand capability packages | Integration | **Subsumed** | Roadmap synthesis | 11.13, 13.4 | Implement through the unified skills/plugin track. |
| 13.4A | Skills with progressive disclosure | Projection/UI | **Ready** | Frontend plan, Roadmap synthesis | 11.13B helpful | UI/UX projection of the same capability substrate. |
| 13.4B | Full plugin packaging/distribution | Operator | **Spec needed** | Roadmap synthesis, Comparison matrix | 11.13C, 11.15 | Needs install/update/trust/marketplace semantics. |
| 14.4 | Skills / commands / dynamic context injection | Integration | **Ready** | Roadmap synthesis, Research ledger | 11.13B helpful | High-value implementation path once capabilities are first-class. |

---

# 5) Policy & Hooks Engine

> **Execution plan:** _pending._
> **What the plan still has to decide:** the contract for the hook bus and how hook trust is reviewed, covering rows `12.3`, `13.3`, and `14.3`. What approval and execution policy actually mean in practice, covering rows `12.16`, `13.7`, and `14.2`. And where enforcement happens when a capability seam is crossed, which is row `12.1`. This is Gap B in this roadmap, and it gates approvals, hooks, permissions, and sandbox behaviour in three other systems.

## System intent

This system owns:
- host-owned policy decisions,
- hook bus semantics,
- approvals,
- trust review,
- execpolicy,
- permission rules,
- guard/secret scanning.

## Why these items belong together

All of these rows answer one question:

> what is allowed, when, by whom, and under what user-visible rules?

If hooks, permissions, approvals, and guards become separate one-off systems, GoHarness will end up with multiple competing sources of authority.

They should instead form one **host policy engine**.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 11.15 | Project trust & local config policy | Policy | **Ready** | Roadmap synthesis, Research ledger | — | Best treated as an early surface of the unified policy engine. |
| 12.1 | Capability seams (FS / Runner / Terminals / LSP) | Substrate | **Blocked** | Roadmap synthesis | Refactor bandwidth | Foundational for consistent enforcement. |
| 12.3 | Waterfall event hooks | Substrate | **Blocked** | Roadmap synthesis, Comparison matrix | 12.2 ideal but narrow v1 can precede it | Core hook/event substrate. |
| 12.16 | Approval policy & sandbox as services | Policy | **Blocked** | Roadmap synthesis, current sandbox code | 12.1, 12.3, 14.2 | Should not be built as isolated modal logic. |
| 12.17 | Hooks bridge (`hooks.json`) | Operator | **Blocked** | Roadmap synthesis | 12.3 | Bridge surface over the hook bus. |
| 12.22 | Guard / secret scanning | Policy | **Ready** | Roadmap synthesis, current guardrails | — | Good near-term policy interceptor. |
| 12.30.6 | Mode-scoped tool sets | Policy | **Ready** | Character card design | — | Extends `12.12`. A node declares named modes, and the agent may enter one it declared, which swaps the tool list at runtime. A mechanism rather than a memory feature, and useful whether or not the memory work proceeds. |
| 13.3 | Lifecycle hooks with trust review | Policy | **Blocked** | Roadmap synthesis, Comparison matrix | 12.3 | Hook trust/approval is the same engine, not a separate feature. |
| 13.7 | Approval policy + sandbox presets + execpolicy | Policy | **Ready** | Roadmap synthesis, current sandbox code | — | One of the clearest policy/control-plane rows right now. |
| 14.2 | Permission rules as first-class user contract | Policy | **Ready** | Roadmap synthesis, Comparison matrix | — | Strongly related to execpolicy and approvals. |
| 14.3 | Claude-style matcher hooks + trust | Policy | **Blocked** | Roadmap synthesis | 12.3 | Same engine, more advanced matcher surface. |

---

# 6) Execution & Review System

> **Execution plan:** _pending._
> **What the plan still has to decide:** what a persistent execution session is, and whether ownership of it sits with a PTY layer or somewhere else. This covers rows `12.4` and `13.9`. Which system owns the validation loop is another open question, though the Task system has already claimed the loop and its state while leaving the actual command execution here. Worktree and branch isolation is row `14.6`, and it is Gap C in this roadmap. Finally, the Code Mode runtime and its safety contract is row `12.20`.

## System intent

This system owns:
- agent and user execution surfaces,
- persistent terminals,
- validation loops,
- code mode,
- review/security review,
- and worktree-aware isolation.

## Why these items belong together

These rows all depend on the same runtime substrate:

- persistent execution identity,
- durable process/PTY ownership,
- validation and recheck loops,
- isolated execution target selection,
- operator-visible results.

They are different faces of one execution system, not isolated features.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 11.5 | `!command` shell injection | Integration | **Ready** | Frontend plan, Roadmap synthesis | — | Useful immediately, but conceptually part of the execution surface. |
| 12.4 | First-class persistent terminals (PTY) | Substrate | **Ready** | Frontend plan, Comparison matrix | 12.1 helpful but not required | This is the real substrate replacing older rough shell-session ideas. |
| 12.20 | Code Mode / model-written programs | Integration | **Spec needed** | Roadmap synthesis | 12.1 | Needs runtime choice, safety model, and output contract. |
| 13.5 | Dedicated code-review mode | Integration | **Ready** | Roadmap synthesis, Comparison matrix | — | One of the clearest review-mode opportunities now. |
| 13.9 | Persistent terminal sessions + integrated validation workflow | Integration | **Blocked** | Roadmap synthesis | 12.4 first | Strong second-wave PTY feature. |
| 14.6 | Worktree-aware isolation | Substrate | **Spec needed** | Roadmap synthesis | review/branch/job semantics | Must be designed with rollback/branch/task systems, not bolted on. |
| 14.7 | Review + security-review as first-class flows | Integration | **Ready** | Roadmap synthesis | — | Best built on top of the same execution/review substrate as 13.5. |

---

# 7) MCP Runtime Expansion

> **Execution plan:** _pending._
> **What the plan still has to decide:** how the transport and authentication work roll out in stages, which is row `13.8`, and how prompts and resources from an MCP server are exposed to the user. Elicitation also depends on the task system's `waiting_user` state, which is row `11.11`, and that dependency is the reason this system lands after the task substrate rather than alongside it.

## System intent

This system owns:
- MCP transport breadth,
- auth/OAuth,
- prompts/resources surfaces,
- remote server management,
- elicitation support.

## Why these items belong together

The current MCP client already proves the core idea.
The next two rows are just a staged runtime expansion of the same subsystem.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 13.8 | MCP breadth: HTTP, OAuth, richer management | Substrate | **Ready** | Roadmap synthesis, Research ledger | — | Stage it carefully; no need to ship every transport/auth mode at once. |
| 14.9 | MCP prompts, resources, remote transports, OAuth, elicitation | Integration | **Blocked / staged** | Roadmap synthesis | 11.11 for elicitation; 13.8 for transport breadth | Prompt/resource surfacing can land earlier than full OAuth+elicitation stack, but it is one runtime family. |

---

# 8) Runtime, Config & Operator Control Plane

> **Execution plan:** _pending._
> **What the plan still has to decide:** how configuration layers and which layer wins when they disagree, covering rows `14.1` and `12.15`. What the headless mode and its JSONL output schema look like, which is row `13.1`. How the RPC seam depends on the session event substrate, which is row `13.2`. And how rows `13.10` and `14.10` merge into one operator surface instead of staying two separate lists of polish.

## System intent

This system owns:
- repo/local config layering,
- operator commands and diagnostics,
- headless runner surfaces,
- RPC/server seam,
- telemetry contracts,
- config/profile bundles.

## Why these items belong together

These are all control-plane features over the rest of GoHarness.
They should be treated as one operator/runtime program instead of miscellaneous polish rows.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 12.15 | Profiles & bundles as layered patchable config | Substrate | **Spec needed** | Roadmap synthesis | 14.1 helps | Needs merge semantics, precedence, and file format rules. |
| 12.23 | Human feedback out of model context | Operator | **Ready** | Roadmap synthesis, Archived memory spec | — | Sidecar/projection on session state; must not enter model-visible conversation text. Moved here from the memory system to keep one row in one system. |
| 12.25 | Telemetry / observability contracts | Operator | **Ready** | Research ledger, Comparison matrix, Code | — | This system owns the observability contract; the Session Event & Memory System owns the event schema it reads. Cross-referenced there instead of duplicated. |
| 12.26 | Headless / RPC / SDK modes | Integration | **Subsumed** | Roadmap synthesis | 13.1, 13.2 | Do not implement separately. |
| 13.1 | Headless non-interactive runner + JSONL + schema output | Operator | **Ready** | Roadmap synthesis, Comparison matrix | — | Strategically important and cleanly scoped. |
| 13.2 | App-server / RPC seam | Substrate | **Blocked** | Roadmap synthesis, Comparison matrix | 12.2 recommended | Narrow v1 possible sooner, but the clean version wants better event/log substrate. |
| 13.10 | Operator polish: doctor, completions, theme, status, init | Operator | **Ready** | Roadmap synthesis | — | High-value control-plane polish cluster. |
| 14.1 | Repo-scoped configuration hierarchy | Substrate | **Ready** | Roadmap synthesis | — | Strong structural cleanup item. |
| 14.10 | Operator polish: init / doctor / status / memory / permissions / tasks | Operator | **Ready** | Roadmap synthesis | — | Overlaps with 13.10; implement as one operator surface. |
| 12.30.8 | System prompt composition blocks | Substrate | **Ready** | Character card design | — | The system prompt becomes an ordered list of named blocks that a person can reorder, enable, disable, and budget. The default order reproduces the prompt assembled today, and the blocks that describe the runtime cannot be removed. Lands with or without the card work, and the persona is one of its blocks. |

---

# 9) Shell & Interaction System

> **Execution plan:** _pending._
> **What the plan still has to decide:** the contract for slots and regions in the interface, covering rows `12.28.1` and `12.29.2`, and the typed transcript model that this system consumes from the memory system, covering rows `12.29.7` and `12.28.2`. It also has to sort its rows into two groups: those that only display state owned elsewhere, and those where the shell genuinely owns the state. Rows `12.29.10` and `12.29.13` are the ones in question. This is the largest system in the roadmap, and the one most likely to be mistaken for a long list of separate features.

## System intent

This system owns:
- shell frame,
- sidebar/browser/settings surfaces,
- transcript model,
- composer behavior,
- trajectory/details panes,
- typed renderers,
- theme and attachment UX.

## Why these items belong together

The frontend refactor already proved the right direction:
- the shell is spatially stable,
- the composer is the interaction hub,
- transcript, details, settings, and browser are different projections of one UI architecture.

These rows should not be treated as dozens of unrelated frontend tasks.
They are one shell/interactions program with shared DOM/state contracts.

## Items

| ID | Item | Role | Status | Related docs | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|---|
| 11.17 | Richer live status & cost footer / status chips | Projection/UI | **Ready** | Frontend plan, Mockup | — | First pass exists; future shell polish can evolve the placement. |
| 11.18 | Collapsible thinking blocks | Projection/UI | **Ready** | Frontend plan, Roadmap synthesis | — | Provider plumbing can be messy, but UI intent is clear. |
| 11.20 | Editor modal & slash command palette | Projection/UI | **Ready** | Frontend plan, Mockup | — | Clear shell enhancement. |
| 11.21 | In-chat slash commands | Projection/UI | **Ready** | Frontend plan | 11.20 helps | Implementation path is straightforward. |
| 11.22 | Persistent shell sessions / background shells | Integration | **Subsumed** | Frontend plan, Roadmap synthesis | 12.4 | Historical note only; implement via PTY/runtime work. |
| 12.28.1 | Slot / region model for web client | Substrate | **Blocked** | Frontend plan, Mockup | shell refactor start | Clear direction; still the conceptual shell spine. |
| 12.28.2 | Step-grouped conversation + sticky composer | Projection/UI | **Blocked** | Frontend plan, Mockup | 12.28.1, 12.29.7 ideally | Wants a better transcript/event model. |
| 12.28.3 | Composer takeover for approvals/questions | Projection/UI | **Blocked** | Frontend plan, Mockup | 11.11, 12.3 | UI is clear; backend waiting semantics first. |
| 12.28.4 | Trajectory / inspector view | Projection/UI | **Blocked** | Frontend plan, Mockup | 12.2 | Wants reliable event substrate. |
| 12.28.5 | `@` and `/` trigger system | Projection/UI | **Ready** | Frontend plan, Mockup | — | One of the most direct UI items. |
| 12.28.6 | Tool call tree with nested sub-calls | Projection/UI | **Blocked** | Frontend plan, Mockup | richer event lineage | Wants deeper event/tool lineage than we persist today. |
| 12.28.7 | Sub-agent navigation and fleet UI | Projection/UI | **Blocked** | Frontend plan, Mockup, Subagent plan | 11.7 | Needs durable task/session metadata first. |
| 12.28.8 | Plan chip / todo / goals / jobs badge | Projection/UI | **Blocked** | Frontend plan, Mockup | 11.12, 12.7, 12.8, 11.7 | Projection over task system, not a separate substrate. |
| 12.28.9 | Workspace/session sidebar with grouped searchable rows | Projection/UI | **Ready** | Frontend plan, Mockup | 12.18 for full content search | Strong first pass now; full-text search can layer in later. |
| 12.28.10 | Settings as plugin cards with live model testing | Projection/UI | **Ready** | Frontend plan, current provider UI | revisioned settings write contract later | Good incremental UI/control-plane improvement. |
| 12.28.11 | First-class theme system | Projection/UI | **Ready** | Frontend plan, Mockup | — | Foundation exists. |
| 12.28.12 | Message feedback / deliverables / references | Projection/UI | **Ready** | Frontend plan, Mockup | 11.4 helps | **Deliberate cross-listing:** Knowledge Ingress & Retrieval owns the staged evidence; this system owns the rendering surface. Feedback routing is operator-side (`12.23`). |
| 12.28.13 | Drag-and-drop attachments | Projection/UI | **Ready** | Frontend plan | 12.19 improves backend | Strong v1 UI. |
| 12.28.14 | Empty/hero state and block reasons | Projection/UI | **Partial** | Frontend plan, Mockup | — | Truthfulness pattern exists; keep broadening it. |
| 12.28.15 | Reliability details worth copying | Projection/UI | **Ready as checklist** | Frontend plan | attach to related rows | Not a standalone milestone; use as acceptance criteria. |
| 12.29.1 | Three-column shell with concession chain | Substrate | **Ready** | Frontend plan, Mockup | 12.28.1 recommended | Strong spec and partially landed direction. |
| 12.29.2 | Slot ownership of chrome | Substrate | **Blocked** | Frontend plan, Mockup | 12.28.1 | Straight dependency. |
| 12.29.3 | Conversation view tabs | Projection/UI | **Blocked** | Frontend plan, Mockup | 12.28.1, 12.29.1 | Shell projection once slot model is firm. |
| 12.29.4 | Sidebar anatomy and collapse motion | Projection/UI | **Ready** | Frontend plan, Mockup | — | Strongly spec'd. |
| 12.29.5 | Settings as sidebar surface | Projection/UI | **Blocked** | Frontend plan | 12.29.1, 12.28.10 | Shell refactor first. |
| 12.29.6 | Composer regions and in-place takeover | Substrate | **Ready** | Frontend plan, Mockup | backend features vary | UI structure is already clear. |
| 12.29.8 | Layer / z-index contract | Substrate | **Ready** | Frontend plan | — | Good low-risk cleanup. |
| 12.29.9 | Typed blocks (Terminal, Diff, Read, Search, Web) | Projection/UI | **Ready** | Frontend plan, Mockup | — | Implement incrementally per block type. |
| 12.29.10 | Persistent shell identity across session switches | Projection/UI | **Blocked** | Frontend plan | shell refactor | Wants shell state hoisting. |
| 12.29.11 | Workspace/session browser details | Projection/UI | **Ready as checklist** | Frontend plan | 12.28.9 | Acceptance criteria, not a standalone initiative. |
| 12.29.12 | Branding and theming mechanics | Projection/UI | **Partial** | Frontend plan, Mockup | — | Token foundation shipped; broader system still open. |
| 12.29.13 | Boot/rendering lifecycle | Substrate | **Blocked** | Frontend plan | ESM/frontend split | Wait for further shell modularization maturity. |
| 12.30.7 | Showcase workflow: persona, modes, lorebooks | Integration | **Ready** | Character card design, `workflows.json` | 12.30.4, 12.30.6 | A third workflow beside `linear_chat` and `enhanced_cognition`, both of which stay unchanged. It is where the feature is demonstrated end to end. |

---

## Consolidations and “don’t duplicate this work” notes

These items should be treated as **implementation inputs to newer systems**, not separate parallel projects:

| Older item | Implement through | Reason |
|---|---|---|
| 10.2C Async / Fire-and-Forget | 11.7 durable background jobs | Same capability, stronger task-system spec |
| 11.22 Persistent shell sessions | 12.4 + 13.9 | PTY/runtime substrate is the real system |
| 12.24 Skills as packages | 11.13 + 13.4 + 14.4 | Unified capability system is clearer |
| 12.26 Headless / RPC / SDK modes | 13.1 + 13.2 | Control-plane framing is stronger |
| 13.6 Plan/goal/progress rows | 12.7 + 12.8 + 14.8 + 12.29.6 | Projection over task state, not first substrate |
| 12.28.15 Reliability details | related feature acceptance criteria | Not a standalone milestone |
| 12.29.11 Browser details | 12.28.9 acceptance criteria | Same surface |

---

## Recommended delivery waves (system-first)

The waves below are ordered by leverage rather than by row number. A wave can start before the previous one finishes, but the ordering matters where a later system depends on an earlier one.

### Wave 1 — systems whose first slices are already specified

1. **Knowledge Ingress & Retrieval System**, beginning with rows `11.1`, `11.4`, `11.16`, `11.19`, and `12.19`.
2. **Task & Background Work System**, beginning with rows `11.2`, `11.3`, `11.7`, `11.11`, and `11.12`.
3. **Execution & Review System**, beginning with rows `11.5`, `11.10`, `13.5`, and `14.7`.
4. **Runtime, Config & Operator Control Plane**, beginning with rows `13.1`, `13.10`, `14.1`, and `14.10`.
5. **Shell & Interaction System**, by continuing the shell, composer, and browser work that the frontend refactor plan already specifies.

### Wave 2 — file-backed archived memory, version one

The first item is row `9.3`, built the way `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md` describes it. We should not wait for the full event-log migration before shipping useful cold-memory retrieval. Once retrieval works, it should be cross-linked into the Knowledge and Shell systems so that recalled memory shows up where knowledge already appears.

### Wave 3 — the policy backbone and richer execution

1. **Policy & Hooks Engine**, beginning with rows `12.1`, `12.3`, `13.7`, and `14.2`. This is the backbone the rest of the wave depends on.
2. **Execution & Review System**, beginning with rows `12.4`, `13.9`, and `14.6`.
3. **MCP Runtime Expansion**, beginning with row `13.8` and its staged expansion of transports and authentication.

### Wave 4 — long-term substrate work

1. **Session Event & Memory System**, covering rows `11.6`, `12.2`, `12.18`, `12.29.7`, `9.1`, and `9.2`. These are the pieces that need the canonical event log to exist.
2. **Capabilities & Extensions System**, covering rows `11.13C` and `13.4B`.
3. **Policy & Hooks Engine**, covering rows `12.16`, `12.17`, `13.3`, and `14.3`.

---

## The biggest remaining design gaps

### Gap A — canonical session and event storage

There is still no agreed storage model for a session as an append-only log of events. Three areas need it: the Session Event & Memory System itself, the richer transcript projections planned for the Shell system, and a clean seam for RPC and the control plane. Until it exists, each of those areas will invent its own read model, and they will drift apart.

### Gap B — a unified host policy engine

Several features need one place that decides what is allowed: approvals, execution policy, the hooks bridge, matcher hooks, trust review, and restrictions on capabilities. Today there is no such place, which is why every one of those features would otherwise grow its own partial version of the same authority.

### Gap C — execution isolation semantics

We have not decided what isolation means for a piece of execution. Four features wait on that answer: worktree-aware isolation, review-and-fix flows, persistent validation workflows, and background experimental runs. Without a shared answer, each will define isolation slightly differently.

### Gap D — extension packaging lifecycle

There is no lifecycle for extensions yet. That covers how a skill or plugin is installed, updated, and trusted, how its scope and lifetime work, and how it loads dynamic context or capabilities. Skills and plugins cannot ship safely until these rules exist, because the rules are most of what makes them safe.

---

## Practical rules for implementation

1. **Build each substrate once.** When several rows want the same state machine or the same storage model, that is a sign they belong to one system rather than several features. Build the shared piece first, then let the features sit on top of it.

2. **Keep user-facing rows subordinate to backend truth.** A shell feature should display state that the task, memory, or policy systems already own. It should not invent its own parallel version of that state, because the two will disagree.

3. **Keep the link to research and specification visible.** Every substantial implementation should say which research or specification document defines it or justifies the approach.

4. **Do not show controls that do nothing.** If a mode or setting is not genuinely honoured by the runtime, it should not appear in the interface or the configuration file. A switch that pretends to work is worse than a missing switch.

5. **Prefer honest local retrieval over claims about unlimited memory.** This matters most for the archived-memory work, now that hierarchical retrieval is a real part of the plan rather than an aspiration.

---

## Suggested immediate next document work

The remaining ambiguity in this roadmap sits in the specifications for the system substrates, not in the feature lists. Three documents would remove most of it:

1. **A session event log and migration specification.** This supports the long-term Session Event & Memory System, including how existing session files migrate to the new substrate.
2. **A policy and hooks specification.** This would unify approvals, hooks, trust review, and execution policy into one design instead of four partial ones.
3. **An execution isolation and worktree specification.** This unblocks the stronger review and runtime model described in the Execution & Review system.

The archived-memory area is no longer on this list. It has both research grounding in `docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md` and an implementation spec in `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md`, so it is no longer the most uncertain part of the roadmap.
