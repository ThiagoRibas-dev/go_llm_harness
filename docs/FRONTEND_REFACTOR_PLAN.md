# Frontend Refactor & UI Maturation Plan

> **Status:** Active
> **Owner:** Arena.ai Agent Mode
> **Last updated:** 2026-09-12

This document is the working plan for maturing the GoHarness web UI **before** we pile on more user-facing features.

It consolidates three things into one source of truth:

1. **Current-state audit** of the shipped embedded web UI.
2. **Target interaction model** informed primarily by **DeepSeek Harness** UI patterns, plus the useful adjacent lessons from `tldw_chatbook`.
3. **Implementation plan** for restructuring the shell and then decomposing the monolithic frontend into ES modules.

This is not just a code-organization plan anymore. It is a **UI architecture plan**.

---

## 1. Why this should happen now

GoHarness already has a surprising amount of capability in the browser:

- workspace switching
- session management
- snapshots
- provider profiles
- MCP management
- workflow editing
- tool-call rendering
- sub-agent progress
- rollback / branching
- cost/token displays

But the UI still behaves like a collection of powerful internal controls rather than a coherent shell.

The biggest risk is not that the UI is broken. The biggest risk is that we keep shipping more visible features into a shell that still has poor surface boundaries:

- too many things live inside the Settings modal
- too many concepts can be changed from multiple places
- too many fields are raw text instead of constrained pickers
- some tabs/panels are visually first-class but not interactionally first-class
- file and workspace interaction feel more like diagnostics than navigation

If we fix the shell now, later features like approvals, `@file`, `/commands`, trajectory, deliverables, plan mode, and background jobs will slot into stable places instead of forcing another full UI rethink.

---

## 2. Constraints and product rules

### Hard constraints

- Keep GoHarness **single-binary** and **embedded-asset** based.
- No runtime Node/Vite/webpack/esbuild dependency.
- The frontend remains browser-delivered from Go `//go:embed` assets.
- Use **ES modules** for the refactor.
- Preserve current product capabilities while reshaping their surfaces.

### Product rules

1. **One important object should have one obvious home.**
   - Workspace selection is navigation, not general settings.
   - Workflow editing is a product surface, not a settings subsection.

2. **Do not fake affordances.**
   - If something is blocked, inert, unavailable, or ignored by runtime, the UI must say so directly.

3. **Prefer constrained controls over raw text when the valid set is knowable.**
   - Pickers before free-text.
   - Disclosures before giant always-visible advanced forms.

4. **Inline takeover beats modal explosion.**
   - Approvals, questions, staged context, queue rows, and plan strips belong in or near the composer.

5. **Inspection should live in an inspection surface.**
   - Trajectory, file preview, sub-agent details, run ledgers, and deliverables belong in a details panel or equivalent, not buried in settings.

6. **The shell should be spatially stable.**
   - Left = navigate
   - Center = converse / act
   - Right = inspect

---

## 3. Reference model: what DeepSeek Harness gets right

DeepSeek Harness is the primary reference for **shell ownership and UI surface boundaries**, not because we want to clone its styling, but because it makes object interactions more coherent.

### Useful DSH patterns to copy

- **Three-column shell**: sidebar / conversation / details
- **Conversation stays mounted** while views swap
- **Settings are a surface**, not a mega-modal
- **Trajectory / inspector** live in a details view, not in transcript clutter
- **Composer is a stack**, not just one textarea
- **Approvals/questions take over the composer in place**
- **Workspace/session browser is a true navigation surface**
- **Deliverables are first-class**
- **Typed renderers** for terminal, diff, read, search, web blocks
- **Visibility rules** are deliberate: what is navigation vs state vs inspection is obvious

### Useful `tldw_chatbook` adjacent lessons

- truthful blocked states with recovery instructions
- explicit distinction between Search and grounded/RAG answer modes
- staged evidence handoff into conversation
- clear separation between agent tools and any explicitly user-owned persistent terminal
- artifact / portable-bundle mindset for outputs

---

## 4. Current UI architecture snapshot

The current frontend still lives mostly inside one file:

- `src/web/index.html`

It currently contains:

- top header
- left sidebar with `Files / Sessions / Snapshots`
- central transcript + composer
- giant Settings modal
- fork/branch modal
- providers editor nested inside Settings
- workflow editor nested inside Settings
- inline CSS
- large inline JS behavior surface

### Major interaction surfaces in the current file

- header settings trigger: `openSettingsModal()`
- sidebar tabs: `switchSidebarTab('files'|'sessions'|'snapshots')`
- workspace selector: `workspace-history-select`
- workspace add: `new-workspace-input` + `addNewWorkspace()`
- session list: `selectSession(...)`
- workflow selector in header: `workflow-selector`
- workflow lab editor selector: `wf-lab-selector`
- settings modal tabs: `switchSettingsTab('standard'|'workflow'|'providers')`
- composer: `prompt-form`, `submitPrompt(...)`, `handleInputKeydown(...)`
- timeline fork modal: `triggerFork(...)`, `executeForkAction()`

---

## 5. Current-state audit: structure, content, and interaction problems

## 5.1 Misclassified surfaces

### A. Workflow Lab is incorrectly housed inside Settings
**Current:** `settings-panel-workflow`

This is one of GoHarness's strongest product surfaces and one of its largest UI systems, yet it is treated as a tab inside the Settings modal.

#### Why this is a problem
- It is cramped by modal height/width constraints.
- It is visually categorized as configuration instead of work.
- It creates too much nesting: Settings → Workflow → Graph → Inspector → JSON.
- It discourages a future where the active workflow, live trace, and workflow editing all relate coherently.

#### DSH comparison
DSH gives major surfaces real seats in the shell. It does not hide a product-defining interaction model inside a general settings container.

#### Decision
**Workflow Lab should become a first-class app surface.**

---

### B. Snapshots are over-promoted
**Current:** top-level sidebar tab peer to Files and Sessions.

#### Why this is a problem
Snapshots are useful, but they are not an equal-frequency navigation object compared to:
- workspace/session selection
- active conversation
- file/context inspection

Treating them as a primary sidebar tab increases shell complexity for little gain.

#### Decision
**Demote Snapshots** into a secondary surface:
- details panel section,
- session/history utility,
- or a smaller history tools view.

---

### C. Settings own too many unrelated concepts
The current Standard Settings tab mixes:
- active provider/model settings
- compaction settings
- workspace path
- sandbox settings
- scan dirs
- ignore/collapse patterns
- MCP server management

#### Why this is a problem
The user has to mentally parse whether they are:
- changing current chat behavior
- changing app-wide runtime config
- choosing a connection profile
- editing advanced infrastructure
- editing the workspace itself

#### Decision
Settings need **progressive disclosure** and **stronger grouping**, with several major surfaces moved out.

---

## 5.2 Redundant interactions

### A. Workspace switching happens in too many places
Current workspace-touching interactions:
1. workspace dropdown in Sessions tab
2. clickable workspace history rows
3. add workspace via free-text input
4. `workspace_dir` free-text field in Settings
5. implicit workspace changes via session selection

#### Problem
One concept has too many homes. Navigation and configuration are mixed.

#### DSH comparison
DSH treats workspace/session selection as navigation, not as a general settings field.

#### Decision
Keep:
- one primary workspace/session browser
- one secondary add/validate flow

Remove:
- workspace path editing from standard settings

---

### B. Workflow selection is split across runtime and editing with weak separation
Current workflow interactions:
1. header `workflow-selector` switches active runtime workflow
2. `/workflow` slash behavior also switches active workflow
3. `wf-lab-selector` chooses which workflow to edit
4. JSON editor directly edits workflow schema
5. AI compiler stages another draft

#### Problem
There are really three distinct actions here:
- choose active runtime workflow
- choose workflow to edit
- edit workflow definition

The UI exposes those capabilities, but does not make the distinctions obvious enough.

#### Decision
Keep these as separate concepts, but **separate them by surface**:
- runtime selection in main shell
- editing target in Workflow Lab
- advanced JSON only inside Workflow Lab

---

### C. File pinning is manual despite the app already knowing the filesystem
Current interaction:
- user types into `quick-pin-input`
- tree is visible elsewhere

#### Problem
Typing file paths manually is redundant and fragile when the app already has a workspace tree.

#### Decision
Pinning should come from:
- `@file` picker,
- file browser clicks,
- or search results,
not free-text typing.

---

### D. Recovery actions are fragmented
Current actions include:
- reroll
- edit & fork
- rollback / branch
- destructive truncate inside fork modal

#### Problem
All of these are legitimate, but they do not present one clear mental model for recovery/history.

#### Decision
Future organization should distinguish:
- retry last assistant result
- branch from here
- rewind to here
- inspect turn details

These should be grouped under a coherent history model, not scattered as ad hoc buttons.

---

## 5.3 Fake or weakly-realized interactions

### A. Snapshots tab is not treated as a truly first-class tab
The UI visually presents three peer tabs, but the switching logic is still largely "files vs not-files" in spirit.

### B. Files pane behaves more like a styled report than a browser
The workspace tree is visible, but it is not yet a strong interaction surface for:
- open
- preview
- pin
- mention
- inspect

### C. Settings looks like configuration, but contains major product workspaces
Workflow Lab is the clearest example.

### D. Composer looks like the main control hub, but lacks the real command vocabulary
Missing first-class live interactions include:
- `@file`
- `/commands`
- `!command`
- queue rows
- staged evidence strip
- plan/todo strip
- approvals takeover
- details-linked inspection

---

## 5.4 Interaction model gaps versus DSH

### DSH is stronger because it has better object ownership

| Concept | DSH tendency | Current GoHarness tendency |
|---|---|---|
| Workspace/session | navigation | split across tabs, settings, history widgets |
| Settings | settings surface | mega-modal containing multiple product worlds |
| Workflow editing | dedicated surface | nested settings tab |
| File interaction | contextual references + details | static tree + manual typing |
| Approvals/questions | composer takeover | mostly future work / modal-ish mental model |
| Inspection | details panel | mixed between transcript and settings |
| Transcript structure | step-aware | mostly stacked cards |

---

## 6. Target interaction architecture

## 6.1 Shell model

### Left rail / sidebar
Owns:
- workspaces
- sessions
- New Session
- search sessions/workspaces
- Settings entry
- secondary utilities

### Center conversation column
Owns:
- active session header
- conversation view tabs
- transcript
- composer stack

### Right details panel
Owns:
- trajectory / event ledger
- file preview
- selected deliverable details
- active tool / approval / sub-agent details
- snapshot/history utilities where appropriate

---

## 6.2 Surface reassignment matrix

| Current thing | Current container | Better container | Why |
|---|---|---|---|
| Workflow Lab | Settings modal tab | top-level surface / main shell view | It is core product work, not settings |
| Providers | Settings modal tab | settings surface with cards, maybe side panel | Still configuration, but needs less nesting |
| Snapshots | sidebar top tab | details/history utility surface | not frequent enough to deserve equal primary-nav weight |
| File preview | none / indirect | details panel | inspection belongs in inspection surface |
| Approvals | future modal tendency | composer takeover | lower context switching, stronger locality |
| Search result staging | mostly absent | composer-adjacent strip | user should see what will enter context |
| Trajectory | future hidden surface | details tab / conversation tab | inspection should not compete with settings |

---

## 6.3 Primary object interaction rules

### Workspace
- primary interaction: select in sidebar workspace browser
- secondary interaction: add/import workspace via explicit action
- not editable as a generic settings field

### Session
- primary interaction: select row in sidebar
- row should expose status, rename, delete, maybe branch/fork affordances
- current session state should be visible without switching surfaces

### File
- primary interactions:
  - mention via `@file`
  - open from details/deliverables/search result
  - pin/stage from picker or browser
- not primarily manual path entry

### Workflow
- runtime selection lives in main shell
- editing lives in Workflow Lab
- JSON/schema editing lives inside Workflow Lab only

### Approval / question
- appears inline in composer takeover
- should not force leaving the conversation surface

### Deliverable
- clickable chip opens details preview
- can be exported or staged into follow-up work

---

## 7. Required interaction improvements

## 7.1 Immediate cleanup pass in the current shell

These are high-value fixes that do not require the final three-column architecture first.

### Must fix now
1. Remove `workspace_dir` editing from standard settings.
2. Demote Snapshots from the top-level sidebar tab strip.
3. Replace free-text pinned-file entry with selection-driven interaction.
4. Promote **New Session** into stable visible chrome.
5. Clarify active workflow selector vs workflow-to-edit selector.
6. Reduce top-header overload.
7. Move raw advanced fields behind disclosure blocks.

### Expected payoff
- fewer duplicated paths
- fewer accidental invalid states
- lower cognitive load
- better match between what surfaces look like and what they actually are

---

## 7.2 Mature shell pass

### New shell structure to implement
- rail: global nav / stable actions
- sidebar: workspaces + sessions
- center: conversation
- details: trajectory / preview / inspection

### Key interactions to add
- resizable sidebar/details handles
- conversation tabs: Chat / Trajectory / Sub-agents
- staged evidence strip above composer
- queue rows above composer
- typed details targets (file preview, run ledger, deliverables)

### DSH alignment
This is the point where we move from “many controls exist” to “each control has a stable home.”

---

## 7.3 Composer maturity pass

### Composer should become a stack
1. status/stats row
2. queue rows
3. staged context strip
4. plan/todo strip
5. textarea
6. access row
7. trigger overlay

### Trigger system
The live product should support:
- `/` commands
- `@` files / agents / sessions / deliverables
- `!` shell shortcuts

### Questions and approvals
- takeover in place
- no placeholder clutter in transcript
- exactly one visible unblock action when blocked

---

## 7.4 File and workspace interaction maturity pass

### File system navigation
We should not try to become a full IDE tree first.

Instead:
- keep a lightweight file browser/search surface
- make file interactions contextual and deliberate
- use details preview instead of permanent giant explorer behavior

### Minimum viable file interactions
- click file row → preview in details panel
- pin/stage from file row
- `@file` picker as primary reference mechanism
- deliverable file chips open directly
- read blocks and search blocks open source files predictably

---

## 8. ES module refactor, reshaped around the new shell

The old module split plan is still directionally right, but it was organized around the current monolithic page. We should instead split modules along the **target shell boundaries**.

## 8.1 Target module layout

```text
src/web/
├── index.html
└── js/
    ├── app.js
    ├── state/
    │   ├── app-state.js
    │   ├── session-state.js
    │   ├── ui-state.js
    │   └── workflow-state.js
    ├── util/
    │   ├── dom.js
    │   ├── fetch.js
    │   ├── format.js
    │   └── events.js
    ├── shell/
    │   ├── shell.js
    │   ├── rail.js
    │   ├── sidebar.js
    │   ├── details.js
    │   └── layout.js
    ├── chat/
    │   ├── transcript.js
    │   ├── turn-cards.js
    │   ├── typed-blocks.js
    │   ├── approvals.js
    │   ├── composer.js
    │   ├── triggers.js
    │   ├── queue.js
    │   └── staged-context.js
    ├── sessions/
    │   ├── workspaces.js
    │   ├── sessions.js
    │   ├── branching.js
    │   └── snapshots.js
    ├── files/
    │   ├── browser.js
    │   ├── preview.js
    │   ├── pinned-context.js
    │   └── uploads.js
    ├── settings/
    │   ├── settings-surface.js
    │   ├── providers.js
    │   ├── runtime.js
    │   ├── compaction.js
    │   └── mcp.js
    ├── workflow/
    │   ├── lab.js
    │   ├── canvas.js
    │   ├── edges.js
    │   ├── nodes.js
    │   ├── inspector.js
    │   ├── validation.js
    │   ├── json-sync.js
    │   └── ai-compiler.js
    └── sse/
        ├── stream.js
        └── event-handlers.js
```

## 8.2 Refactor rule
Do **not** split the code according to the old accidental modal/tab boundaries.

Split it according to the new ownership model:
- shell
- chat
- sessions/workspaces
- files
- workflow lab
- settings
- SSE/event plumbing

---

## 9. Phased implementation plan

## Phase 0 — audit-driven cleanup in current `index.html`

### Goals
- reduce duplicated interactions
- remove obviously misclassified controls
- improve UX honesty without requiring the whole new shell first

### Tasks
- [ ] Remove `workspace_dir` from standard settings
- [ ] Demote/remove Snapshots from top-level sidebar tab strip
- [ ] Replace manual pinned-file entry with selection-driven flow or temporarily hide it until `@file` exists
- [ ] Promote New Session to stable visible chrome
- [ ] Collapse raw advanced provider fields behind disclosure UI
- [ ] Clarify labels between runtime workflow selection and workflow editing
- [ ] Reduce header metrics clutter where a status row would work better

---

## Phase 1 — promote Workflow Lab to a first-class surface

### Goals
- stop treating workflow editing as settings
- give graph editing real space

### Tasks
- [ ] Remove Workflow Lab from Settings modal tabs
- [ ] Create a dedicated app-level Workflow Lab surface
- [ ] Keep active runtime workflow selector in the main shell
- [ ] Keep editing-target selector inside Workflow Lab only
- [ ] Move advanced JSON editor into Workflow Lab-owned disclosure/panel

---

## Phase 2 — establish the shell skeleton

### Goals
- create stable left/center/right ownership
- stop overloading transcript and settings with inspection work

### Tasks
- [ ] Add rail + sidebar + conversation + details shell structure
- [ ] Add optional/resizable details panel
- [ ] Make workspace/session browser primary left sidebar content
- [ ] Define details tabs: Trajectory / File / Tool / Deliverable
- [ ] Add conversation header tab strip for Chat / Trajectory / Sub-agents

---

## Phase 3 — composer maturity

### Goals
- make the composer the true interaction hub

### Tasks
- [ ] Add status/action row near composer
- [ ] Add staged evidence strip
- [ ] Add queued steering/follow-up rows
- [ ] Add `/`, `@`, and `!` trigger overlay
- [ ] Add in-place approvals/questions takeover
- [ ] Add clear blocked-state recovery action patterns

---

## Phase 4 — file and transcript interaction maturity

### Goals
- make files and results actionable, not just visible
- make transcript structure more inspectable

### Tasks
- [ ] Add file preview in details panel
- [ ] Add contextual pin/stage/open actions from file/search/deliverable surfaces
- [ ] Implement typed blocks for Terminal / Diff / Read / Search / Web
- [ ] Move tool-result inspection out of giant raw `<pre>` dependence
- [ ] Add deliverable chips that open previews directly

---

## Phase 5 — settings and providers polish

### Goals
- reduce nested editor feeling
- expose advanced config progressively

### Tasks
- [ ] Convert providers to clearer card/editor model
- [ ] Keep active chat / compaction connection assignments visible
- [ ] Move low-frequency infra fields into Advanced sections
- [ ] Improve settings revision/conflict handling
- [ ] Give MCP management a more explicit connection-state presentation

---

## Phase 6 — ES module decomposition

### Goals
- make the new shell maintainable
- stop growing the monolith

### Tasks
- [ ] Add `<script type="module" src="/js/app.js">`
- [ ] Scaffold new module tree
- [ ] Move util/state/sse first
- [ ] Move shell modules next
- [ ] Move sessions/files/chat/settings/workflow modules incrementally
- [ ] Delete inline script only after behavior parity is verified

---

## 10. Validation and acceptance criteria

## 10.1 Interaction acceptance criteria

### Navigation
- [ ] Workspace switching happens in one obvious primary surface
- [ ] New Session is always easy to find
- [ ] No top-level tab is visually first-class while behaviorally second-class

### Workflow
- [ ] Active workflow selection is clearly distinct from workflow editing
- [ ] Workflow Lab no longer lives in Settings

### Files
- [ ] Files can be previewed, staged, or referenced without manual path typing
- [ ] The file surface behaves like an interaction surface, not a decorative tree dump

### Composer
- [ ] Block reasons are explicit and actionable
- [ ] `/`, `@`, and `!` are discoverable
- [ ] Staged context is visible before send
- [ ] Approvals/questions can happen without leaving the conversation surface

### Settings
- [ ] Common settings are easy; advanced settings are available but not noisy
- [ ] Raw endpoint/path text entry is no longer the default first-line interaction where a picker/list is possible

---

## 10.2 Technical acceptance criteria

- [ ] `node scripts/lint-html.js` passes
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] No console errors on load
- [ ] No duplicate IDs / broken handlers introduced by shell restructuring
- [ ] Workflow Lab still supports drag, connect, validate, and save/apply
- [ ] Session switch preserves shell state more cleanly than the current full rerender behavior

---

## 11. Recommended immediate implementation order

If we want the highest return without boiling the ocean:

1. **Phase 0** cleanup in current shell
2. **Phase 1** promote Workflow Lab out of Settings
3. **Phase 2** establish shell skeleton with details panel
4. **Phase 3** composer maturity (`@`, `/`, queue, staged context)
5. **Phase 4** typed blocks + file preview interactions
6. **Phase 5** settings/provider polish
7. **Phase 6** ES module extraction along the new boundaries

That order is deliberate.

Do **not** fully decompose the old monolith first and then redesign the shell around the decomposed pieces. That risks hardening the wrong boundaries.

Instead:
- clean the interaction model,
- establish the right surface ownership,
- then split the frontend along those new seams.

---

## 12. Decisions log

| Date | Decision | Rationale |
|---|---|---|
| 2026-08-16 | Use ES modules, no bundler | Preserves embedded-asset / zero-runtime-dependency ethos |
| 2026-09-12 | Treat this document as UI architecture + refactor plan, not just code decomposition | The real frontend problem is shell maturity, not only file size |
| 2026-09-12 | Promote Workflow Lab out of Settings before heavier UI feature accretion | Workflow editing is a primary product surface |
| 2026-09-12 | Prefer a DSH-style left / center / right ownership model | Reduces modal overload and duplicated interaction paths |
| 2026-09-12 | Keep explicitness as a product rule | UI must not present ignored or redundant controls as if they matter |

---

## 13. Open questions

- Should file browsing stay a lightweight tree plus search, or should we add a richer browser only after details-panel preview exists?
- Should Settings become a sidebar surface, a slide-over, or a dedicated top-level surface after Workflow Lab leaves it?
- Should Snapshots become a details-panel tool, a history surface, or remain accessible only from a utility menu?
- What is the minimum acceptable first pass for shell persistence across session switches before the ESM refactor?
- Should provider model lists be fetched lazily only when a provider/profile is selected, or preloaded for better picker UX?
