# 🧭 GoHarness Roadmap

This document is the **execution-grade roadmap** for GoHarness. It is the **sole roadmap file** and is meant to be both comprehensive and implementation-aware.

It consolidates:
- product intent
- competitive lessons
- adjacent reference-app lessons
- implementation readiness
- dependency chains
- known spec gaps

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

## Source basis key

- **Roadmap synthesis** — backlog decisions and scope definitions consolidated in this file
- **Code** — current implementation under `src/`
- **Mockup** — `docs/mockups/ui-mockup.html`
- **Frontend plan** — `docs/FRONTEND_REFACTOR_PLAN.md`
- **Pi / DSH / Codex / Claude** — comparative analysis consolidated into this roadmap from prior investigation
- **Crawl4AI / tldw / Prime** — adjacent reference-app lessons consolidated into this roadmap from prior investigation
- **Research docs** — e.g. `docs/RESEARCH.md`, `docs/COMPARISON_MATRIX.md`

## Supporting documents (reference material, not roadmap files)

- `docs/mockups/ui-mockup.html` — post-roadmap UI direction mockup
- `docs/FRONTEND_REFACTOR_PLAN.md` — ES Modules frontend split plan
- `docs/RESEARCH.md` — bibliography / inspiration ledger
- `docs/COMPARISON_MATRIX.md` — older broad comparative notes
- `docs/GIT_HISTORY.md` — implementation and documentation chronology
- `docs/SUBAGENT_PARALLELISM_PLAN.md` — concurrency design notes for the shipped sub-agent runtime

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

### Partially landed foundations with follow-on work still open

| ID | Item | Current state | Remaining work |
|---|---|---|---|
| 12.6 | Tool-output spill to disk | **Partial** | Web-fetch integration, richer typed viewers, broader retrieval UX |
| 12.28.14 | Truthful block/empty states | **Partial** | Extend the same truthfulness to approvals, RAG, MCP, workflow, and terminal surfaces |
| 12.29.12 | Theme token foundation | **Partial** | Persisted theme picker, full semantic-token pass through deeper UI surfaces |

---

## Dependency backbone

These chains explain why some items are ready and some are not.

### 1. Session log / memory backbone
`11.6 session tree` → `12.2 event log as source of truth` → `9.1/9.2 hierarchical memory` + `12.29.7 transcript semantics`

### 2. Policy / hooks backbone
`12.1 capability seams` + `12.3 hook bus` → `12.16 approval+sandbox services` → `12.17 hooks bridge` + `14.2 permission DSL` + `14.3 Claude-style matcher hooks` + `13.7 execpolicy`

### 3. Long-running work backbone
`11.2 message queue` + `11.7 durable background jobs` → `12.8 goals` + `12.21 schedules` + `14.8 background task roster` + `10.2C async fire-and-forget`

### 4. Frontend shell backbone
`11.20 editor modal / slash palette` + `12.28.1 slot model` → `12.29.1 three-column shell` + `12.29.10 persistent shell identity` → higher-order views like trajectory, fleet, staged context, settings sidebars

### 5. PTY / terminal backbone
`12.4 persistent terminals` → `13.9 validation workflow` and supersedes the older rougher `11.22` shell-session idea

### 6. Skills / extension backbone
`11.13 prompt templates + skills` → `14.4 lazy dynamic context skills` → `13.4 / 12.24 full plugin packaging`

---

## Recommended implementation waves

### Wave 1 — highest leverage, ready now
1. **11.1** web search & fetch
2. **11.2** message queue while agent is running
3. **11.3** cancel vs abort distinction
4. **11.4** `@file` mentions
5. **11.10** automatic build/lint/typecheck feedback
6. **11.11** model-initiated Q&A (`ask_user`)
7. **11.17** richer status row / status chips
8. **11.20** editor modal + slash palette
9. **11.21** in-chat slash commands
10. **13.1** headless runner with JSONL events
11. **13.5** dedicated review mode
12. **13.10 / 14.10** doctor, init, status, permissions/tasks operator polish

### Wave 2 — ready, but best after Wave 1 foundations
1. **11.7** durable background jobs
2. **11.8 / 14.5** named agent roles / declarative subagent files
3. **12.4** persistent terminals
4. **12.11** structured sub-agent output
5. **12.19** content-addressed attachments
6. **12.22** guard / secret scanning
7. **12.23** human feedback sidecar
8. **12.25** telemetry contracts
9. **13.7 / 14.2** unified permission DSL + execpolicy
10. **13.8 / 14.9** richer MCP transports/resources

### Wave 3 — implementation-ready but blocked on shared architecture
1. **12.3 / 13.3 / 14.3 / 12.17** hook bus and trusted hooks
2. **12.8 / 13.6 / 14.8 / 12.21** long-running goal/schedule/task system
3. **12.28.1 / 12.29.1 / 12.29.10** frontend shell refactor backbone
4. **12.18** session query after event-log work

### Wave 4 — requires design work first
1. **11.6 / 12.2** canonical event log + session tree
2. **9.1 / 9.2** hierarchical memory subsystem
3. **12.15** layered patchable config
4. **12.20** code mode / model-written programs
5. **14.6** worktree-aware isolation
6. **13.4 / 11.13 / 12.24** full plugin packaging/distribution model

---

# 1) Session, persistence, memory, and export

| ID | Item | Status | Source basis | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|
| 9.1 | O(1) range loader + hierarchical memory decay | **Spec needed** | Roadmap synthesis, OptMem references in `docs/RESEARCH.md` | 11.6, 12.2 | Need storage model, hierarchy geometry, invalidation rules, prompt reconstruction budgets |
| 9.2 | Visual memory map dashboard | **Spec needed** | Roadmap synthesis, newer UI direction in 12.28/12.29 | 9.1, 12.2, shell placement | Need backing API, UI placement, semantics for forget/regenerate/branch-from-summary |
| 11.6 | Session tree / time-travel navigator | **Spec needed** | Roadmap synthesis, Pi references, DSH event-log pressure | 12.2 strongly related | Need canonical event schema, branch semantics, dual-write migration plan |
| 11.9 | Cross-session memory | **Ready** | Roadmap synthesis, Pi memory refs, Code/BM25 | — | Good v1 path: BM25-backed dated memory notes; embedding/rerank can remain optional |
| 12.2 | Session event log as source of truth | **Spec needed** | DSH, Codex/Prime event-stream references, Roadmap synthesis | 11.6 strongly related | Need exact event schema, projections, migration from per-turn JSON files |
| 12.18 | Session query with FTS | **Blocked** | DSH, Roadmap synthesis, current BM25 engine | 11.6, 12.2 | Straightforward after canonical event log exists |
| 12.29.7 | Full step-grouped transcript / compaction placement / streaming-tail isolation | **Spec needed** | DSH, Roadmap synthesis | 12.2 | Partial transcript cleanup is possible now, but the full model needs a typed event stream |
| 12.28.12 | Deliverables, references, message feedback | **Ready** | DSH, tldw, Roadmap synthesis | 11.4 helps | Deliverables row is implementable now; exportable artifact bundles come later |

---

# 2) Search, web fetching, ingestion, and local knowledge surfaces

| ID | Item | Status | Source basis | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|
| 11.1 | Web search & fetch | **Ready** | Pi, Crawl4AI, tldw, Roadmap synthesis | Spill foundation already landed | One of the best-specified items; default pure-Go fetch, optional JS-heavy fallback later |
| 11.4 | `@file` mentions in composer | **Ready** | Pi, Code, Roadmap synthesis | — | UI typeahead + backend file query |
| 11.16 | Parent-directory context walking + overrides | **Ready** | Pi, current `LoadLocalInstructions()`, Roadmap synthesis | — | Small, clear extension |
| 11.19 | Image paste/drag into composer | **Ready** | Pi, current uploads plumbing, Roadmap synthesis | 12.19 improves storage | Ready even before content-addressing, though 12.19 gives the cleaner backend |
| 12.5 | LSP seam | **Ready** | DSH, Roadmap synthesis | — | Clear four-op seam; just medium effort |
| 12.19 | Content-addressed attachments | **Ready** | DSH, current uploads path, Roadmap synthesis | — | Good bounded v1 |
| 14.9 | MCP prompts, resources, remote transports, OAuth, elicitation | **Blocked / staged** | Claude MCP docs, Roadmap synthesis | 11.11 for elicitation; 13.8 for transport breadth | Prompt/resource surfacing is ready sooner than full OAuth+elicitation stack |
| 13.8 | MCP breadth: streamable HTTP, OAuth, richer management | **Ready** | Codex + Claude MCP docs, Roadmap synthesis | — | Staged implementation strongly recommended |

---

# 3) Agent loop, sub-agents, long-running work, and orchestration

| ID | Item | Status | Source basis | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|
| 10.2B | Wait First / Race | **Ready** | Roadmap synthesis, current subagent runtime, Prime references | — | Implement through current subagent engine; define cancellation and winner semantics clearly |
| 10.2C | Async / Fire-and-Forget | **Subsumed** | Roadmap synthesis, Prime, 11.7 | 11.7 | Do not build separately; fold into durable background jobs |
| 11.2 | Message queue while agent is running | **Ready** | Pi, Code, Roadmap synthesis | — | Good v1 spec already exists |
| 11.3 | Cancel vs abort distinction | **Ready** | Pi, Code, Roadmap synthesis | — | Small and clear |
| 11.5 | `!command` shell injection | **Ready** | Pi, Code, Roadmap synthesis | — | Small, careful UX needed |
| 11.7 | Durable/background sub-agent jobs | **Ready** | Pi, Prime, current subagent runtime, Roadmap synthesis | 11.2 helpful | Prefer explicit mailboxes / receipts / artifact exchange, not magical shared mutable context |
| 11.8 | Named sub-agent roles & packaged workflows | **Ready** | Pi, Claude 14.5 references, current subagent runtime | — | File format choice is open but not blocker-level |
| 11.10 | Automatic build/lint/typecheck feedback loop | **Ready** | Pi, Code, Roadmap synthesis | — | Heuristic per-language v1 is good enough |
| 11.11 | Model-initiated Q&A interludes | **Ready** | Pi ask-user ref, DSH/Codex/Claude approval patterns | — | Async wait and resume behavior are understood well enough |
| 11.12 | Persistent TODO/plan overlay | **Ready** | Pi, DSH composer references, Roadmap synthesis | — | Straightforward first pass |
| 12.4 | First-class persistent terminals (PTY) | **Ready** | DSH, tldw, Roadmap synthesis | 12.1 helpful but not required | Important distinction between model-invoked PTY tools and any explicitly user-owned persistent terminal surface |
| 12.7 | Plan mode as logged collaboration state | **Blocked** | DSH, Codex, Claude, Roadmap synthesis | 12.3, 11.11, UI takeover plumbing | Behavior is clear; interception and UI surfaces should exist first |
| 12.8 | Goals with autonomous round-driving | **Blocked** | DSH, Prime, Codex, Roadmap synthesis | 11.2, 11.7 | Enough info exists, but it wants durable continuation machinery |
| 12.9 | Dynamic workflows over sub-agents | **Blocked** | DSH, current subagents, Roadmap synthesis | 11.7, 12.10, 12.11 | Better after background jobs and structured output land |
| 12.10 | Multiple sub-agent providers | **Ready** | DSH, Codex/Claude command/runtime references | — | Interface-first implementation is ready |
| 12.11 | Structured sub-agent output | **Ready** | DSH, current subagent schema | — | Tight, bounded item |
| 12.12 | Per-agent tool scoping & personas | **Ready** | DSH, 11.8, 14.5 | — | Enforced filtering, not just hidden prompt text |
| 12.13 | Per-session agent presets | **Ready** | DSH, current `Agent` runtime | — | Good config + UI task |
| 12.14 | Scope primitive | **Blocked** | DSH, Roadmap synthesis | 12.3, 12.13 | Small internal spec still needed once scoped registrations exist |
| 12.20 | Code Mode / model-written programs | **Spec needed** | DSH, Roadmap synthesis | 12.1 | Need runtime choice, security model, SDK/binding surface |
| 12.21 | Scheduled / mission runs | **Blocked** | DSH, Prime, Roadmap synthesis | 11.7, 12.8 | Good once jobs exist |
| 13.5 | Dedicated code-review mode | **Ready** | Codex, Claude, Roadmap synthesis | — | Strongly spec'd and high value |
| 13.6 | Plan mode + goal mode + progress rows | **Blocked** | Codex, DSH, Mockup | 12.7, 12.8, 12.29.6 | Use as integration item, not as the first implementation entry |
| 13.9 | Persistent terminal sessions + integrated validation workflow | **Blocked** | Codex, tldw, 12.4, 11.10 | 12.4 first | Good second-wave PTY feature |
| 14.5 | Subagents as data files | **Ready** | Claude docs, current runtime, Roadmap synthesis | — | Strong improvement path over ad hoc role config |
| 14.6 | Worktree-aware isolation | **Spec needed** | Claude docs, Roadmap synthesis | Review/branch/job semantics | Need interaction with current rollback model, worktree cleanup and collision policy |
| 14.7 | Review + security-review as first-class flows | **Ready** | Claude + Codex, Roadmap synthesis | — | Very implementable |
| 14.8 | Background agents, task roster, queued command semantics | **Blocked** | Claude, Pi, Prime, Roadmap synthesis | 11.2, 11.7 | Clear destination; durable task machinery first |

---

# 4) Runtime architecture, hooks, policy, permissions, and config

| ID | Item | Status | Source basis | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|
| 11.13A | Prompt templates | **Ready** | Pi, Codex, Claude, Roadmap synthesis | — | Tiny first slice |
| 11.13B | Skills (description + lazy load) | **Ready** | Pi, Codex, Claude, Roadmap synthesis | — | Enough for a bounded first pass |
| 11.13C | Full extension protocol / install surface | **Spec needed** | Pi, DSH, Codex, Claude | 12.3, 12.14, trust model | Need wire contract, registration lifecycle, install/update/trust model |
| 11.14 | Provider/model catalog as data | **Ready** | Roadmap synthesis, current provider/profile code | — | Medium, straightforward |
| 11.15 | Project trust & local config policy | **Ready** | Pi, Claude trust docs, Roadmap synthesis | — | Enough for a basic trust gate |
| 12.1 | Capability seams (FS / Runner / Terminals / LSP) | **Blocked** | DSH, Code, Roadmap synthesis | Refactor bandwidth | Concept and interfaces are clear; implement incrementally |
| 12.3 | Waterfall event hooks | **Blocked** | DSH, Codex, Claude, Roadmap synthesis | 12.2 ideal, but narrow v1 can precede it | Enough for a v1 bus; full fidelity wants better logging semantics |
| 12.15 | Profiles & bundles as layered patchable config | **Spec needed** | DSH, Roadmap synthesis | 14.1 helps | Need merge/patch semantics, precedence rules, file format |
| 12.16 | Approval policy & sandbox as services | **Blocked** | DSH, current sandbox code, Claude/Codex policy docs | 12.1, 12.3, 14.2 | Strong direction; wants unified services underneath |
| 12.17 | Hooks bridge (`hooks.json`) | **Blocked** | DSH, Codex, Claude | 12.3 | High value once hook bus exists |
| 12.22 | Guard / secret scanning | **Ready** | DSH, current guardrails, Roadmap synthesis | — | Easy high-value guard item |
| 12.23 | Human feedback out of model context | **Ready** | DSH, Roadmap synthesis | — | Clean sidecar feature |
| 12.24 | Skills as on-demand capability packages | **Subsumed** | DSH, Pi, Roadmap synthesis | 11.13, 13.4 | Do not spec separately; implement through the unified skills/plugin track |
| 12.25 | Telemetry / observability contracts | **Ready** | DSH, current trace logging, Roadmap synthesis | — | Good medium-scope cleanup |
| 12.26 | Headless / RPC / SDK modes | **Subsumed** | DSH, current `Agent` runtime | 13.1, 13.2 | Implement via the newer Codex/Prime-informed items |
| 13.1 | Headless non-interactive runner + JSONL + schema output | **Ready** | Codex, Prime, current `Agent` runtime | — | Very ready and strategically important |
| 13.2 | App-server / RPC seam | **Blocked** | Codex, Prime, Roadmap synthesis | 12.2 recommended | A narrow v1 is possible sooner; clean long-term version wants a better event/log substrate |
| 13.3 | Lifecycle hooks with trust review | **Blocked** | Codex, Claude, DSH | 12.3 | Clear once hook bus exists |
| 13.4A | Skills with progressive disclosure | **Ready** | Codex, Claude, Pi | 11.13B | Strong v1 available |
| 13.4B | Full plugin packaging/distribution | **Spec needed** | Codex, Claude, Pi | 11.13C, 11.15 | Needs install/update/trust/marketplace semantics |
| 13.7 | Approval policy + sandbox presets + execpolicy | **Ready** | Codex, Claude, current sandbox code | — | One of the clearest policy items now |
| 13.8 | MCP breadth: HTTP, OAuth, richer management | **Ready** | Codex + Claude MCP docs | — | Stage it; do not try to ship every transport/auth mode at once |
| 13.10 | Operator polish: doctor, completions, theme, status, init | **Ready** | Codex, Claude, current app state | — | Very implementable and high-frequency UX |
| 14.1 | Repo-scoped configuration hierarchy | **Ready** | Claude docs, Roadmap synthesis | — | Good structural cleanup item |
| 14.2 | Permission rules as first-class user contract | **Ready** | Claude docs, Codex, current guard/sandbox code | — | Strong spec now exists |
| 14.3 | Claude-style matcher hooks + trust | **Blocked** | Claude, Codex, DSH | 12.3 | Hook bus first |
| 14.4 | Skills / commands / dynamic context injection | **Ready** | Claude, Codex, Roadmap synthesis | 11.13B helpful | Very good v1 spec |
| 14.10 | Operator polish: init / doctor / status / memory / permissions / tasks | **Ready** | Claude docs, Codex | — | Overlaps with 13.10; implement as one operator-surface program |

---

# 5) Web UI shell, transcript, and interaction surfaces

| ID | Item | Status | Source basis | Depends on | Missing decisions / notes |
|---|---|---|---|---|---|
| 11.17 | Richer live status & cost footer / status chips | **Ready** | Pi, tldw, Code, Roadmap synthesis | — | Strong first pass exists; later can graduate from footer to clickable chips |
| 11.18 | Collapsible thinking blocks | **Ready** | Pi, current provider translation code | — | Provider plumbing is messy; feature intent is clear |
| 11.20 | Editor modal & slash command palette | **Ready** | Pi, Frontend plan, Roadmap synthesis | — | Good shell-level enhancement |
| 11.21 | In-chat slash commands | **Ready** | Pi, current workflow command handling | 11.20 helps | Implementation path is clear |
| 11.22 | Persistent shell sessions / background shells | **Subsumed** | Pi, DSH 12.4, tldw | 12.4 | Keep as historical note; implement through 12.4 and 13.9 instead |
| 12.28.1 | Slot / region model for web client | **Blocked** | DSH UI docs, Mockup, Frontend plan | Frontend refactor start | Clear direction; wants shell modularization first |
| 12.28.2 | Step-grouped conversation + sticky composer | **Blocked** | DSH, Mockup | 12.28.1, 12.29.7 ideally | First cut possible, full version wants better event model |
| 12.28.3 | Composer takeover for approvals/questions | **Blocked** | DSH, Mockup | 11.11, 12.3 | UI is clear; backend wait logic first |
| 12.28.4 | Trajectory / inspector view | **Blocked** | DSH, Mockup | 12.2 | Needs reliable event substrate |
| 12.28.5 | `@` and `/` trigger system | **Ready** | Pi, DSH, Mockup | — | One of the clearest UI items |
| 12.28.6 | Tool call tree with nested sub-calls | **Blocked** | DSH, Mockup | Richer event structure | Full nested tree wants better tool/event lineage than we store today |
| 12.28.7 | Sub-agent navigation and fleet UI | **Blocked** | DSH, Mockup, 11.7 | 11.7 | Needs durable task/session metadata |
| 12.28.8 | Plan chip / todo / goals / jobs badge | **Blocked** | DSH, Mockup | 11.12, 12.7, 12.8, 11.7 | Pure UI is clear; data sources are not all present yet |
| 12.28.9 | Workspace/session sidebar with grouped searchable rows | **Ready** | DSH, Mockup, tldw | 12.18 for full content search | Good first pass now; later full-text content search can layer in |
| 12.28.10 | Settings as plugin cards with live model testing | **Ready** | DSH, current provider UI | Revisioned settings write contract later | Good incremental item |
| 12.28.11 | First-class theme system | **Ready** | DSH, current theme-token foundation | — | Foundation already in place |
| 12.28.12 | Message feedback / deliverables / references | **Ready** | DSH, tldw | 11.4 helps | Strong item, bounded v1 |
| 12.28.13 | Drag-and-drop attachments | **Ready** | DSH, current upload plumbing | 12.19 improves backend | Clear v1 |
| 12.28.14 | Empty/hero state and block reasons | **Partial** | DSH, tldw, Code | — | Continue broadening this truthfulness across other surfaces |
| 12.28.15 | Reliability details worth copying | **Ready as checklist** | DSH | Attach to related items | Not a standalone feature; use as acceptance criteria |
| 12.29.1 | Three-column shell with concession chain | **Ready** | DSH, Mockup | 12.28.1 recommended | Strong spec |
| 12.29.2 | Slot ownership of chrome | **Blocked** | DSH, Mockup | 12.28.1 | Straight dependency |
| 12.29.3 | Conversation view tabs | **Blocked** | DSH, Mockup | 12.28.1, 12.29.1 | Clear once shell exists |
| 12.29.4 | Sidebar anatomy and collapse motion | **Ready** | DSH, Mockup | — | Strongly spec'd |
| 12.29.5 | Settings as sidebar surface | **Blocked** | DSH, current settings UI | 12.29.1, 12.28.10 | Shell refactor first |
| 12.29.6 | Composer regions and in-place takeover | **Ready** | DSH, Mockup, tldw | Backend features vary | UI structure is clear even if some regions remain empty initially |
| 12.29.8 | Layer / z-index contract | **Ready** | DSH | — | Good low-risk cleanup |
| 12.29.9 | Typed blocks (Terminal, Diff, Read, Search, Web) | **Ready** | DSH, Mockup | — | Implement incrementally per block type |
| 12.29.10 | Persistent shell identity across session switches | **Blocked** | DSH, Frontend plan | shell refactor | Hoist shell state first |
| 12.29.11 | Workspace/session browser details | **Ready as checklist** | DSH | 12.28.9 | Acceptance criteria, not a standalone initiative |
| 12.29.12 | Branding and theming mechanics | **Partial** | DSH, Code | — | Token foundation shipped; full system remains open |
| 12.29.13 | Boot/rendering lifecycle | **Blocked** | DSH, Frontend plan | ESM frontend refactor | Wait for the frontend split |

---

## Consolidations and “don’t duplicate this work” notes

These older roadmap items should now be treated as **implementation inputs to newer items**, not independent parallel projects:

| Older item | Implement through | Reason |
|---|---|---|
| 10.2C Async / Fire-and-Forget | 11.7 durable background jobs | Same capability, stronger spec |
| 11.22 Persistent shell sessions | 12.4 + 13.9 | Newer PTY references are better |
| 12.24 Skills as packages | 11.13 + 13.4 + 14.4 | Unified skill/plugin track is clearer |
| 12.26 Headless / RPC / SDK modes | 13.1 + 13.2 | Newer Codex/Prime framing is stronger |
| 13.6 Plan/goal/progress rows | 12.7 + 12.8 + 12.29.6 | Integration layer, not first implementation target |
| 12.28.15 Reliability details | related feature acceptance criteria | Not a standalone milestone |
| 12.29.11 Browser details | 12.28.9 acceptance criteria | Same surface |

---

## The biggest known design gaps

These are the places where the roadmap still needs real engineering specs before implementation starts.

### Gap A — canonical session/event storage
Needed by:
- 11.6
- 12.2
- 9.1
- 9.2
- 12.18
- 12.29.7
- cleaner 13.2

### Gap B — unified policy model
Needed by:
- 12.16
- 12.17
- 13.7
- 14.2
- 14.3
- 12.7 approvals
- 14.9 elicitation / MCP interaction

### Gap C — worktree semantics for git-backed isolation
Needed by:
- 14.6
- future review/fix flows
- background experimental subagents on repos
- any “branch from summary” concept in 9.2

### Gap D — plugin / extension packaging lifecycle
Needed by:
- 11.13C
- 12.24
- 13.4B
- 11.15 trust model
- 12.14 scoped lifetime handling

---

## Practical reading of this roadmap

If you want the roadmap to stay honest, apply these rules while implementing:

1. **Do not start spec-needed items straight from high-level prose.** Write a small design spec first.
2. **Do not implement blocked items as isolated hacks** just because they are tempting UI work; land the shared foundation once.
3. **Prefer consolidation over parallel tracks.** The newer comparative work often superseded older vague items.
4. **Keep explicitness as a product rule.** If runtime cannot honor a setting, mode, or permission, the UI must say so directly.
5. **Prefer host-owned state transitions over model-declared success.** This especially applies to approvals, goals, schedules, and quality gates.

---

## Suggested immediate next document work

Before attempting Phase 9 or the event-log-heavy UI items, write focused specs for:

1. **Session event log + migration** (`11.6` + `12.2`)
2. **Hook bus + policy DSL** (`12.3` + `14.2` + `13.7`)
3. **Worktree isolation semantics** (`14.6`)

Those three specs would eliminate most of the roadmap ambiguity that still remains.
