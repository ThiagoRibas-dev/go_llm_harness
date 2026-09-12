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

DeepSeek Harness is the primary reference for **shell ownership, interaction boundaries, and surface maturity**, not because we want to clone its visual styling, but because it makes object interactions more coherent.

The most important DSH lesson is not “add more panels.” It is:

> **Each object has a stable home, and each interaction has a clear owner.**

### 3.1 Navigation model: workspace and session browsing

#### DSH reference behavior
- The sidebar is primarily a **workspace/session browser**, not a miscellaneous tool shelf.
- Workspaces group sessions.
- An open workspace shows **5 sessions by default** with a transient **Show more**.
- Rows surface live status directly:
  - **pending interaction** = amber dot
  - **running** = blue activity
  - descendant sub-agent activity outranks stale completion markers
- Search expands inside the workspace header and behaves in two layers:
  - immediate substring filtering for titles/labels
  - **250 ms debounced content search** with snippets and capped results
- Sub-agent-origin sessions are **hidden from the ordinary sidebar** and reached through the parent context.
- Row actions are contextual and compact: rename, fork, archive, delete.
- DSH is mostly **click / hover / keyboard** oriented here, not swipe-oriented. The useful interaction model is about stable rows, hover actions, focus, and drag order — not touch-first gestures.

#### What to copy
- Workspace/session selection should be **navigation**, not a general settings field.
- Search should be in-place and workspace-aware.
- Session status should be visible where you pick the session, not somewhere else.
- We should prefer one primary interaction path for selecting a workspace or session.

### 3.2 Shell geometry: stable left / center / right ownership

#### DSH reference behavior
- The shell is a **three-column frame**: `sidebar`, `conversation`, `details`.
- When horizontal space shrinks, **only the details panel concedes**. Once too small, it auto-closes.
- A collapsed sidebar is not gone; it becomes a **56 px icon rail** with **36 px controls** at **10 px inset**.
- The details panel collapses to **0 px**.
- The conversation surface remains mounted and does not change identity when switching view tabs.
- Drag handles control geometry; the model is spatial and explicit, not modal-heavy.

#### What to copy
- Left = navigate
- Center = converse / act
- Right = inspect
- Collapse and resize should be explicit pointer interactions, not ad hoc hide/show toggles.
- We should prioritize **drag handles and persistent geometry** over adding more modal layers.

### 3.3 Conversation and composer interaction model

#### DSH reference behavior
- The conversation header has a **view ring**: Chat / Trajectory / other views.
- The shell stays mounted while only the **view body** swaps.
- The composer is a **stack**, not one textarea:
  1. status/stats dock
  2. queued message rows
  3. plan/todo/goal strips
  4. editor bar
  5. access row
  6. trigger overlay
- Approvals and user questions **replace the composer in place**.
- Blocked states render the missing reason directly in the composer and keep exactly **one unblock action** live.
- Pending approval/question states do **not** produce fake placeholder transcript cards.

#### What to copy
- Keep the user in the conversation surface when approval or clarification is needed.
- Put state close to the input, not in a giant header or a detached modal.
- Make `@`, `/`, and `!` part of the interaction contract, not hidden power-user affordances.

### 3.4 Settings and provider management

#### DSH reference behavior
- Settings are a **surface**, not a mega-modal.
- Model/provider entries are structured cards.
- Key entry is validated early (e.g. reject `NAME=value` paste shapes).
- “Fetch available models” interrogates the **currently typed unsaved** endpoint/key.
- Settings writes are revisioned and can reject stale edits with `settings-conflict`.

#### What to copy
- Separate common settings from advanced settings.
- Use cards and pickers before raw text.
- Let users validate the draft config before saving.
- Stop making users understand provider settings via one giant scroll form.

### 3.5 File, deliverable, and inspection behavior

#### DSH reference behavior
- Files are often reached contextually through:
  - `@file` mentions
  - deliverable chips
  - read/search/web blocks
  - details-panel previews
- Deliverables are derived from **successful file mutations**, not parsed from prose.
- Typed blocks exist for terminal, diff, read, search, and web content.
- Non-user/system context injections are usually collapsed disclosures, not loud transcript interruptions.

#### What to copy
- A file tree should either become a real interaction surface or be demoted.
- Deliverables should be clickable and inspectable.
- Inspection belongs in a details surface, not buried in settings.

### 3.6 Useful `tldw_chatbook` adjacent lessons

- truthful blocked states with recovery instructions
- explicit distinction between Search and grounded/RAG Answer modes
- staged evidence handoff into conversation
- clear separation between agent tools and any explicitly user-owned persistent terminal
- artifact / portable-bundle mindset for outputs

### 3.7 Adoption policy for reference behavior

| Reference behavior | GoHarness action | Rationale |
|---|---|---|
| Three-column shell | **Adopt** | Best fit for our current complexity |
| Workspace/session grouped browser | **Adopt** | Cleans up duplicated navigation |
| Composer takeover for approvals/questions | **Adopt** | Strong locality and honesty |
| Settings as surface, not mega-modal | **Adopt** | Major usability win |
| Typed blocks for tool output | **Adopt** | High clarity / low ambiguity |
| 56px rail + concession chain | **Adapt** | Keep the idea, but we may persist widths where DSH resets them |
| DSH transient geometry rules | **Adapt** | We likely want stronger persistence than DSH |
| tldw staged evidence strip | **Adopt** | Great match for grounded workflows |
| tldw user-owned persistent terminal semantics | **Adopt** | Prevents muddling human shells and model tools |
| Swipe-heavy mobile behavior | **Reject / not relevant** | This is a desktop/keyboard/pointer-first app |

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

## 6.4 Detailed change definitions against the current DOM

These are the concrete UI reassignments we should make from the current `src/web/index.html`.

| Current element / behavior | Current purpose | Target behavior | Reference basis |
|---|---|---|---|
| `#settings-modal` | giant modal for settings + workflow lab + providers | becomes settings-only or is replaced by a dedicated settings surface / slide-over | DSH settings surface |
| `#settings-panel-workflow` | workflow editor in Settings | removed from Settings and promoted to top-level Workflow Lab surface | DSH shell ownership |
| `#workflow-selector` | active runtime workflow selector in header | keep, but clearly label as **Active workflow** only; do not use it as edit target | current shell + DSH separation of runtime vs inspection |
| `#wf-lab-selector` | choose workflow to edit inside modal | keep, but only inside Workflow Lab surface | current workflow tooling |
| `#tab-files-btn`, `#tab-sessions-btn`, `#tab-snapshots-btn` | mixed sidebar navigation | replace with stable workspace/session navigation; move file preview and snapshots elsewhere | DSH workspace sidebar |
| `#workspace-history-select` | workspace dropdown in Sessions tab | demote or remove once sidebar browser becomes primary | DSH navigation ownership |
| `#input-workspace-dir` | workspace path in standard settings | remove from standard settings | DSH + redundancy analysis |
| `#quick-pin-input` | manual pinned-file text entry | replace with picker or remove until `@file` / file-row staging exists | DSH file references + tldw staged evidence |
| `#sessions-list` | session selection list | expand into richer grouped workspace/session browser with status/search/actions | DSH workspace browser |
| `#workspace-tree` | styled tree dump | either become a real interaction surface (open/pin/stage/preview) or move behind details/search | DSH contextual file access |
| `#chat-messages` | flat transcript stack | evolve into step-grouped transcript with typed blocks and deliverables | DSH message flow |
| `#prompt-form` / `#prompt-input` | simple composer | become stack owner: status row, queue rows, staged context, plan strip, editor, access row, triggers | DSH composer stack |
| header metrics cards | top-level cost/token chrome | move closer to composer as status chips or dock | DSH stats dock + tldw status rows |
| fork/branch modal | mixed safe/destructive history actions | keep temporarily, then split into clearer history model actions | current recovery UX audit |

---

## 7. Required interaction improvements

## 7.1 Immediate cleanup pass in the current shell

These are high-value fixes that do not require the final three-column architecture first.

### Must fix now
1. Remove `#input-workspace-dir` from standard settings.
2. Demote `#tab-snapshots-btn` from the top-level sidebar tab strip.
3. Replace `#quick-pin-input` free-text pinning with a selection-driven flow, or temporarily hide it until `@file` exists.
4. Promote **New Session** out of the Sessions sub-panel into stable visible chrome.
5. Clarify `#workflow-selector` vs `#wf-lab-selector` with explicit labels and surface separation.
6. Reduce top-header overload by moving cost/token/runtime state toward a lower status row.
7. Move raw advanced fields (base URL overrides, scan-dir comma lists, etc.) behind disclosure blocks.

### Specific current-element changes
- `#tab-files`, `#tab-sessions`, `#tab-snapshots` should stop pretending to be equal product destinations.
- `switchSidebarTab(...)` should either become a real three-surface router or be removed in favor of the new shell model.
- `openSettingsModal()` should stop being the entry point to Workflow Lab.
- `saveSettings(event)` should no longer serialize workspace navigation state as though it were app config.

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

### Detailed shell definitions
- Sidebar collapse target: **56 px icon rail**
- Rail controls: **36 px** targets
- Sidebar details concession: details panel shrinks first, then auto-closes
- Settings seat: bottom-pinned in sidebar or slide-over, not header-only modal
- Search interaction: expand in-place, outside click collapses empty state, content search debounced

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

### Detailed interaction definitions
- **Enter** while idle: submit
- **Enter** while running: queue steering
- **Alt+Enter** while running: queue follow-up
- **Shift+Enter**: newline
- **Escape** with queue focus: cancel queued row or clear draft
- blocked composer: clicking the card triggers exactly one unblock path if one exists
- approval/question takeover: no modal jump, no fake placeholder transcript row

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
- click pin/stage action → add to staged context strip
- `@file` picker as primary reference mechanism
- deliverable file chips open directly
- read blocks and search blocks open source files predictably
- uploaded files surface as explicit staged/attached objects, not silent hidden state

### Workspace interactions
- workspace selection belongs in the sidebar browser
- Add Workspace is a secondary explicit action with validation
- session switching must not fully destroy shell identity
- snapshots, if retained, should be reachable from details/history utilities rather than top-level navigation

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
- [x] Remove `#input-workspace-dir` from standard settings and stop persisting workspace navigation through `saveSettings(event)`.
- [x] Demote/remove `#tab-snapshots-btn` from the top-level sidebar tab strip; if snapshot actions remain, move them under a history/details utility.
- [x] Replace `#quick-pin-input` free-text pinning with a selection-driven flow or temporarily hide it until `@file` exists.
- [x] Promote New Session from the Sessions sub-panel into stable visible chrome (rail or sidebar head).
- [x] Collapse raw advanced provider fields (`#input-model`, `#input-base-url`, compaction endpoint/model overrides, scan-dir comma lists) behind disclosure UI where a picker/list cannot replace them yet.
- [x] Clarify labels between `#workflow-selector` (**active runtime workflow**) and `#wf-lab-selector` (**workflow being edited**).
- [x] Reduce header metrics clutter by moving token/cost/runtime state toward a lower status row closer to `#prompt-form`.
- [x] Make `switchSidebarTab(...)` either truly support each visible destination or remove the misleading destination if it is not first-class.

---

## Phase 1 — promote Workflow Lab to a first-class surface

### Goals
- stop treating workflow editing as settings
- give graph editing real space

### Tasks
- [x] Remove `#settings-panel-workflow` and `#btn-settings-tab-workflow` from the Settings modal contract.
- [x] Create a dedicated app-level Workflow Lab surface with full-height canvas, inspector, and advanced JSON drawer owned by the lab itself.
- [x] Keep `#workflow-selector` in the main shell as the runtime selector only.
- [x] Keep `#wf-lab-selector` inside Workflow Lab as the editing-target selector only.
- [x] Move `#workflow-json-editor` into a Workflow Lab-owned disclosure/panel, not a settings sub-scroll.
- [x] Preserve `Compile & Apply` semantics, but stop closing Settings as a side-effect of saving workflow edits because workflow editing should no longer live there.

---

## Phase 2 — establish the shell skeleton

### Goals
- create stable left/center/right ownership
- stop overloading transcript and settings with inspection work

### Tasks
- [x] Add rail + sidebar + conversation + details shell structure.
- [x] Add explicit resize handles and implement the concession rule: details shrinks first, then closes.
- [x] Convert the left sidebar into the primary workspace/session browser; remove the current split-brain between dropdowns, history widgets, and settings fields.
- [x] Define details tabs: Trajectory / File / Tool / Deliverable / History utilities.
- [x] Add conversation header tabs for Chat / Trajectory / Sub-agents while keeping the conversation shell mounted.
- [x] Ensure collapsed sidebar behavior converges toward a 56 px rail target and does not merely set width to zero.

---

## Phase 3 — composer maturity

### Goals
- make the composer the true interaction hub

### Tasks
- [ ] Add status/action row near composer for provider, model, tools, approvals, sources, and scope.
- [ ] Add staged evidence strip so retrieved files/search hits/web pages are visible before send.
- [ ] Add queued steering/follow-up rows with clear remove/edit interactions.
- [ ] Add `/`, `@`, and `!` trigger overlay anchored to the caret or composer seat.
- [ ] Add in-place approvals/questions takeover instead of routing these through modals.
- [ ] Add clear blocked-state recovery action patterns with exactly one primary unblock action.
- [ ] Preserve submit/newline semantics while adding queue semantics: idle Enter submits, running Enter queues steering, Alt+Enter queues follow-up.

---

## Phase 4 — file and transcript interaction maturity

### Goals
- make files and results actionable, not just visible
- make transcript structure more inspectable

### Tasks
- [ ] Add file preview in details panel with a stable open target.
- [ ] Add contextual pin/stage/open actions from file rows, search results, and deliverable chips.
- [ ] Implement typed blocks for Terminal / Diff / Read / Search / Web.
- [ ] Move tool-result inspection out of giant raw `<pre>` dependence and into typed block renderers with collapse/expand behavior.
- [ ] Add deliverable chips that open previews directly and derive from real successful file mutations.
- [ ] Make file interactions consistent: click row = preview, explicit secondary action = stage/pin, `@file` = mention.

---

## Phase 5 — settings and providers polish

### Goals
- reduce nested editor feeling
- expose advanced config progressively

### Tasks
- [ ] Convert providers to clearer card/editor model with one obvious edit path.
- [ ] Keep active chat / compaction connection assignments visible without forcing users through nested tabs.
- [ ] Move low-frequency infra fields into Advanced sections, and replace raw text with lists/pickers wherever the valid set is knowable.
- [ ] Improve settings revision/conflict handling so stale edits are caught explicitly.
- [ ] Give MCP management explicit connection-state, auth-state, and error-state presentation.
- [ ] Ensure Settings remains a settings surface, not a dumping ground for workflow, file, or history interactions.

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
