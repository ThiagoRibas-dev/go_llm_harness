# 🧭 GoHarness Roadmap

This document is the **execution-grade roadmap** for GoHarness. It is the **sole roadmap file** and is meant to be both comprehensive and implementation-aware.

It consolidates:
- product intent
- competitive lessons
- adjacent reference-app lessons
- implementation readiness
- dependency chains
- known spec gaps
- and now, a **system-oriented delivery model** instead of a flat bag of feature rows

Supporting documents remain useful, but they are **reference material**, not parallel roadmap sources.

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

---

## Why the roadmap is organized this way now

The older roadmap structure was useful for discovery, but too many rows were really:

- one backend substrate,
- plus one policy layer,
- plus one UI projection,
- plus one operator surface.

If we implement those rows one-by-one as isolated features, we will repeatedly rebuild the same state models.

So this roadmap now follows a **system-oriented model**.

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

This is a refinement of the earlier consolidation model: two areas that were previously implicit are now called out explicitly because they are large enough to deserve their own implementation systems:

- **Knowledge Ingress & Retrieval**
- **Runtime, Config & Operator Control Plane**

---

## System execution plans

Each of the nine systems gets **exactly one** execution plan. The split of responsibilities is:

- this roadmap is the **map** — what a system owns, which rows belong to it, what blocks what, and which
  research/spec document justifies each row;
- the execution plan is the **program** — source inputs (including reference-project source inspection),
  phases, concrete slices, testing, risks, and the success definition.

| # | System | Execution plan | Plan status | Plan shape |
|---|---|---|---|---|
| 1 | Session Event & Memory | [`docs/SESSION_EVENT_MEMORY_EXECUTION_PLAN.md`](./SESSION_EVENT_MEMORY_EXECUTION_PLAN.md) | **Written** | 6 phases (A–F) · 5 slices |
| 2 | Knowledge Ingress & Retrieval | _pending_ | **Next up** (Wave 1) | — |
| 3 | Task & Background Work | [`docs/TASK_BACKGROUND_WORK_EXECUTION_PLAN.md`](./TASK_BACKGROUND_WORK_EXECUTION_PLAN.md) | **Written** | 8 phases (A–H) · 6 slices |
| 4 | Capabilities & Extensions | _pending_ | Wave 4 | — |
| 5 | Policy & Hooks Engine | _pending_ | Wave 3 | — |
| 6 | Execution & Review | _pending_ | Wave 1 + Wave 3 | — |
| 7 | MCP Runtime Expansion | _pending_ | Wave 3 | — |
| 8 | Runtime, Config & Operator Control Plane | _pending_ | Wave 1 | — |
| 9 | Shell & Interaction | _pending_ | Wave 1 (largest system) | — |

**Plan status** is `Written` or `Pending`. A pending plan is not a roadmap gap: it means the system has
enough specification to keep its `Ready` rows actionable, but the program document (phases, slices,
tests) has not been produced yet.

**Row coverage.** Every plan carries a **row coverage map** near its header stating, for each roadmap row
it claims, which phase owns it and how deep that coverage is (`core` / `partial` / `reference` / `gap`).
That map is the authoritative trace from a roadmap row to a phase, and it is deliberately allowed to say
`gap` — a claimed-but-unwritten row is more useful to know about than a silently implied one.

**Deliberate cross-listings.** One row is intentionally shared, because it spans two systems and
splitting it would lose information:

- `12.28.12` (deliverables / references / message feedback) — **Knowledge Ingress & Retrieval** owns the
  staged evidence (what a deliverable or reference *is* once ingested) and **Shell & Interaction** owns
  its rendering surface. Both table entries say so explicitly.

Every other row appears in exactly one system. If a row turns up in two systems without a note like the
above, that is a defect in this roadmap, not a deliberate overlap.

**How to read a system section below**

- The **Items** table is the master backlog for that system: row, role, status, related docs, dependencies.
- The **Execution plan** blockquote is the program document, the decisions the plan has already made, and
  the honest notes about what it does not yet cover.
- Rows marked **Subsumed** are historical: they are implemented through the newer item named in the table,
  never as a parallel project.

---

## Current shipped baseline

These are the most important foundations already landed and **should not be re-specified as open work**:

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

> **Execution plan:** [`docs/SESSION_EVENT_MEMORY_EXECUTION_PLAN.md`](./SESSION_EVENT_MEMORY_EXECUTION_PLAN.md) · 6 phases (A–F) · 5 slices
> **Decisions already made by the plan:** the file-backed derived projection ships first (`9.3` via phases B–D) instead of waiting on the canonical event log; compaction writes epoch manifests; recall is always provenance-labelled; the plan explicitly refuses "infinite context" framing and markets bounded frontier retrieval instead.
> **Honest coverage:** `11.9`, `11.6`, `9.3` are core to the plan; `9.1`, `9.2`, `12.2` and `12.29.7` are partial (substrate, API, or projection only); `12.18` is not yet addressed in the plan body.
> **Open decision:** whether Phase F / Slice 5 also satisfies this roadmap's requested *session event log + migration spec*, or whether a separate spec is still owed.

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

> **Execution plan:** _pending._
> **What the plan must decide:** the evidence-staging contract shared by web/read/search/upload/spill sources (`11.1`, `11.4`, `12.6`); how LSP results enter that same path (`12.5`); the content-addressing migration for attachments (`12.19`) that `11.19` and `12.28.13` depend on.

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

---

# 3) Task & Background Work System

> **Execution plan:** [`docs/TASK_BACKGROUND_WORK_EXECUTION_PLAN.md`](./TASK_BACKGROUND_WORK_EXECUTION_PLAN.md) · 8 phases (A–H) · 6 slices
> **Decisions already made by the plan:** the queue becomes host-owned and the browser becomes a projection (`11.2`); one task state machine covers jobs, goals and schedules (`queued` → `admitted` → `running` → `waiting_user` / `waiting_dependency` → terminal); the v1 job store is session-local rather than a daemon (`11.7`); schedules use claim-before-run with interrupted-claim recovery (`12.21`).
> **Honest coverage:** `11.2`, `11.3`, `11.7`, `11.11`, `12.8`, `12.21` are core phases; `11.8`, `12.9`, `12.11`, `14.5` are referenced through Phase E; `12.7`, `11.12`, `13.6`, `14.8` land as Phase H projections; `10.2B`, `11.10`, `12.10`, `12.12`, `12.13` are claimed rows with no body text yet.

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
> **What the plan must decide:** the capability manifest and scope model (`12.14`); the trust/install/update lifecycle for extensions (`11.13C`, `13.4B`); the seam between this capability registry and the policy engine's trust decisions.

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
> **What the plan must decide:** the hook bus contract and trust review (`12.3`, `13.3`, `14.3`); approval and execpolicy semantics (`12.16`, `13.7`, `14.2`); the capability-seam enforcement points (`12.1`). This is roadmap **Gap B** and it gates approvals, hooks, permissions and sandbox behaviour in three other systems.

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
| 13.3 | Lifecycle hooks with trust review | Policy | **Blocked** | Roadmap synthesis, Comparison matrix | 12.3 | Hook trust/approval is the same engine, not a separate feature. |
| 13.7 | Approval policy + sandbox presets + execpolicy | Policy | **Ready** | Roadmap synthesis, current sandbox code | — | One of the clearest policy/control-plane rows right now. |
| 14.2 | Permission rules as first-class user contract | Policy | **Ready** | Roadmap synthesis, Comparison matrix | — | Strongly related to execpolicy and approvals. |
| 14.3 | Claude-style matcher hooks + trust | Policy | **Blocked** | Roadmap synthesis | 12.3 | Same engine, more advanced matcher surface. |

---

# 6) Execution & Review System

> **Execution plan:** _pending._
> **What the plan must decide:** PTY / persistent execution identity and who owns it (`12.4`, `13.9`); validation-loop ownership (`11.10`); worktree and branch isolation semantics (`14.6`, roadmap **Gap C**); the Code Mode runtime and safety contract (`12.20`).

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
> **What the plan must decide:** the staged transport/auth rollout (`13.8`); prompts and resources surfacing; and elicitation's dependency on the task system's `waiting_user` state (`11.11`) — which is why this system lands after the task substrate rather than beside it.

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
> **What the plan must decide:** config layering and merge precedence (`14.1`, `12.15`); the headless / JSONL schema contract (`13.1`); the RPC seam's dependence on the event substrate (`13.2`); and merging `13.10` + `14.10` into a single operator surface instead of two polish lists.

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

---

# 9) Shell & Interaction System

> **Execution plan:** _pending._
> **What the plan must decide:** the slot/region contract (`12.28.1`, `12.29.2`); the typed transcript model consumed from the memory system (`12.29.7`, `12.28.2`); and which rows are pure projection versus shell-owned state (`12.29.10`, `12.29.13`). This is the largest system in the roadmap and the one most likely to be mistaken for many separate features.

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

## Wave 1 — highest leverage systems with ready slices

1. **Knowledge Ingress & Retrieval System**
   - `11.1`, `11.4`, `11.16`, `11.19`, `12.19`
2. **Task & Background Work System**
   - `11.2`, `11.3`, `11.7`, `11.11`, `11.12`
3. **Execution & Review System**
   - `11.5`, `11.10`, `13.5`, `14.7`
4. **Runtime, Config & Operator Control Plane**
   - `13.1`, `13.10`, `14.1`, `14.10`
5. **Shell & Interaction System**
   - continue implementing the already-specified shell/composer/browser work from the frontend plan

## Wave 2 — file-backed archived memory v1

1. **Session Event & Memory System**
   - implement `9.3` first as the file-backed v1 defined in `docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md`
   - do not wait for full event-log migration before landing useful cold-memory retrieval
2. Cross-link that retrieval into the Knowledge and Shell systems

## Wave 3 — policy backbone and richer execution

1. **Policy & Hooks Engine**
   - `12.1`, `12.3`, `13.7`, `14.2`
2. **Execution & Review System**
   - `12.4`, `13.9`, `14.6`
3. **MCP Runtime Expansion**
   - `13.8` staged transport/auth expansion

## Wave 4 — long-term canonical substrate work

1. **Session Event & Memory System**
   - `11.6`, `12.2`, `12.18`, `12.29.7`, `9.1`, `9.2`
2. **Capabilities & Extensions System**
   - `11.13C`, `13.4B`
3. **Policy & Hooks Engine**
   - `12.16`, `12.17`, `13.3`, `14.3`

---

## The biggest remaining design gaps

### Gap A — canonical session/event storage
Needed primarily by:
- Session Event & Memory System
- advanced Shell transcript projections
- clean RPC/control-plane seams

### Gap B — unified host policy engine
Needed by:
- approvals
- execpolicy
- hooks bridge
- matcher hooks
- trust review
- capability restrictions

### Gap C — execution isolation semantics
Needed by:
- worktree-aware isolation
- review/fix flows
- persistent validation workflows
- background experimental execution

### Gap D — extension packaging lifecycle
Needed by:
- install/update/trust model for skills/plugins
- scope/lifetime semantics
- dynamic context/capability loading

---

## Practical rules for implementation

1. **Build substrates once.**
   If multiple rows want the same state machine or storage model, that means there is one system, not many features.

2. **Keep UI rows subordinate to backend truth.**
   Shell features should project real state from the task/memory/policy systems, not invent their own parallel representations.

3. **Keep research/spec traceability explicit.**
   Every major implementation should cite the research/spec document that defines or justifies it.

4. **Do not surface fake controls.**
   If a runtime knob or mode is not truly honored, keep it out of the UI/config surface.

5. **Prefer truthful local-first retrieval over magical memory claims.**
   This is especially important for the archived-memory roadmap now that hierarchical retrieval is entering the plan.

---

## Suggested immediate next document work

The biggest roadmap ambiguity now sits in system-substrate specs. The next document work should therefore be:

1. **Session event log + migration spec**
   - to support the long-term Session Event & Memory System
2. **Policy & hooks spec**
   - to unify approvals, hooks, trust, and execpolicy
3. **Execution isolation / worktree spec**
   - to unblock the stronger review/runtime model

The archived-memory side now has both:
- research grounding (`docs/INFINITE_CONTEXT_MEMORY_RESEARCH.md`)
- and an implementation spec (`docs/HIERARCHICAL_ARCHIVED_MEMORY_SPEC.md`)

So it is no longer the most ambiguous area in the roadmap.
