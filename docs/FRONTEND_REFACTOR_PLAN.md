# Frontend Refactor & CI Plan

> **Status:** In progress
> **Owner:** Arena.ai Agent Mode
> **Last updated:** 2026-08-16

This document is the single source of truth for two related efforts:

1. **Continuous Integration** — automated HTML/JS linting, Go tests, and
   cross-platform binary builds on GitHub Actions.
2. **The Great `index.html` Decomposition** — refactoring the 3,700-line
   monolithic single-file SPA into ES modules, without introducing a Node
   bundler/toolchain to the runtime.

It is a **living document**: as work lands, check boxes, update statuses, and
record decisions so anyone (human or agent) can pick up where the last session
left off.

---

## 1. Goals & Non-Goals

### Goals

- **Catch layout regressions in CI** — specifically the class of bug that
  repeatedly broke the Settings modal (mismatched/unbalanced HTML tags,
  handlers referencing undefined functions, duplicate IDs).
- **Build all four release binaries automatically** on every push to `main`
  and attach them as workflow artifacts.
- **Make the frontend maintainable** by splitting the inline `<script>` into
  focused ES modules that can be read, tested, and reasoned about
  independently.
- **Preserve the project's "zero runtime dependencies" ethos.** The compiled
  binary stays a single self-contained executable with no external assets.

### Non-Goals

- No npm/Yarn bundler (Vite/esbuild/webpack) in the runtime/build path.
  GoHarness is served by Go's `net/http` + `//go:embed`, and that stays.
- No TypeScript in the first pass (JS modules are already a huge win; TS can
  be revisited once modules are stable and a check step exists).
- No redesign of UI/UX — structure only, behavior preserved.
- No splitting of the *HTML* into multiple pages (it remains a single shell;
  only the JS is modularized).

---

## 2. Current State (pre-refactor baseline)

### Architecture

```
src/web/index.html   (~215 KB, ~3,700 lines)
  ├── <style>        all CSS inline
  ├── <body>         header, sidebar, chat, settings modal, fork modal
  └── <script>       ~150 KB of vanilla JS:
                       - SSE / chat rendering
                       - settings (config, compaction, sandbox, MCP)
                       - providers (CRUD + activate)
                       - workflow graph engine (canvas, nodes, edges,
                         inspector, validation, JSON sync, AI compiler)
                       - sidebar (files, sessions, snapshots)
                       - helpers (fetch wrappers, DOM, formatting)
```

Go embeds everything:

```go
// src/web.go
//go:embed web/*
```

### Pain points that motivated this

- **Fragile layout.** A single misplaced `</div>` (e.g. an inline API field
  sitting outside the `#inline-api-fields` toggle) silently collapsed an
  entire settings panel. Hard to spot in a 900-line form.
- **No automated checks on the HTML/JS.** A typo in an `onclick` handler
  (`toggleAddNodeMenu` / `autoLayoutNodes` were called but never defined)
  shipped to runtime.
- **Everything shares one global scope.** Functions reference each other
  across 150 KB with no explicit dependency graph, making safe edits hard.
- **The workflow editor alone is ~1,200 lines** of intertwined rendering,
  drag handling, validation, and state — a natural module boundary.

---

## 3. CI Specification

### Triggers

| Event    | Behavior                                        |
| -------- | ----------------------------------------------- |
| push to `main` | Run lint + test, then build 4 binaries, upload artifacts |
| PR to `main`  | Run lint + test (no artifact upload)            |

### Pipeline: `lint-test` job

1. Check out repo.
2. Install Go 1.24 (with module cache).
3. Install Node 20 (only for the HTML linter; no `npm install`).
4. `go vet ./...`
5. `go test ./...`
6. `node scripts/lint-html.js`

### Pipeline: `build` job (matrix)

- Depends on `lint-test` passing.
- Matrix: `linux/amd64`, `windows/amd64`, `darwin/amd64`, `darwin/arm64`.
- `CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/<artifact> ./src`
- Upload each binary as a workflow artifact (30-day retention).

### The HTML linter (`scripts/lint-html.js`)

A dependency-free Node script (no `npm install`) that validates
`src/web/index.html`:

| Check | Catches |
| ----- | ------- |
| **Tag balance** | Mismatched/unclosed `<div>` etc. (the Settings-modal class of bug) |
| **Duplicate IDs** | Two elements with the same `id` |
| **Inline handler resolution** | `onclick="undefinedFn()"` referencing a JS function that doesn't exist |
| **Anchor targets** | `href="#missing-id"` pointing at a non-existent element |
| **Inline JS syntax** | `new Function(body)` parses every inline `<script>` without running it |

It strips comments and `<script>`/`<style>` contents before structural checks
so template-literal HTML in JS doesn't cause false positives, and locates the
repo by walking up from `cwd` looking for `go.mod` (so it works from any
directory or CI).

### Local commands

```bash
make lint          # go vet + HTML lint
make test          # go test ./...
make build         # cross-compile all 4 release binaries
make all           # lint + test + build
```

---

## 4. ES Module Refactor Plan

### Approach

- Load the main module with `<script type="module" src="/js/app.js">`.
- Serve JS files from `src/web/js/` — Go already embeds and serves
  everything under `web/`, so **no backend changes are needed**.
- ES modules run in strict mode, support `import`/`export`, and work in all
  evergreen browsers over HTTP (GoHarness is always served over HTTP).
- Keep the HTML shell static; modules attach behavior by element ID, exactly
  as today. Incremental migration: move one logical chunk at a time, verify
  after each.

### Target module layout

```
src/web/
├── index.html                 (thin shell: markup + one <script type=module>)
└── js/
    ├── app.js                 # bootstrap: init DOM, wire top-level events
    ├── config.js              # constants, API paths, shared state
    ├── util/
    │   ├── dom.js             # $, $$, escapeHtml, debounce
    │   ├── fetch.js           # get/post JSON helpers with error handling
    │   └── format.js          # token/cost/time formatting
    ├── sse.js                 # EventSource connection, event dispatch
    ├── chat/
    │   ├── chat.js            # message rendering, greeting, submit
    │   ├── turn-cards.js      # assistant/tool/user card markup
    │   ├── metrics.js         # inspect-execution-metrics panel
    │   └── edits.js           # edit-and-fork / reroll
    ├── sidebar/
    │   ├── sidebar.js         # tab switching, init
    │   ├── files.js           # workspace tree, uploads, pinned files
    │   ├── sessions.js        # history list, select, rename, delete
    │   └── snapshots.js       # create/revert/delete snapshots
    ├── settings/
    │   ├── settings.js        # modal open/close, tab routing, save/cancel
    │   ├── api-config.js      # API provider, profile selector, vertex fields
    │   ├── compaction.js      # compaction form
    │   ├── sandbox.js         # runtime/safety form
    │   └── mcp.js             # MCP server CRUD
    ├── providers/
    │   └── providers.js       # providers.json CRUD, activate, editor form
    └── workflow/
        ├── workflow.js        # graph state, load/save, public API
        ├── nodes.js           # NODE_DEFS, node factory, port resolution
        ├── canvas.js          # render cards, drag-to-move, auto-layout
        ├── edges.js           # SVG bezier rendering, edge selection
        ├── wiring.js          # port drag-to-connect, addEdge/cycle checks
        ├── inspector.js       # right-panel typed forms per node type
        ├── validation.js      # anchors, cycles, required inputs, reachability
        ├── json-sync.js       # buildSchemaFromModel / loadWorkflowIntoModel
        ├── toolbar.js         # add-node menu, auto-layout, reload
        ├── ai-compiler.js     # natural-language -> schema stub
        └── live-trace.js      # workflow_start/node/end SSE card
```

### Module interaction rules

- **No circular imports.** If A and B need each other, extract shared
  state/functions to a third module or use an event.
- **State lives in one place per domain** (`workflow.js` owns the wfModel;
  `config.js` owns global UI state). Other modules import getters/actions,
  not mutate state directly.
- **DOM lookup happens once at init** and elements are passed in, not
  re-queried on every event.
- **Each module exports an `init(root)` function** where it queries its
  container and attaches listeners. `app.js` calls them in dependency order.
- Keep the same CSS class names and element IDs during the move so styling
  is unaffected.

### Migration order (incremental, each step shippable)

1. **Scaffold** — create `js/app.js`, add `<script type="module">`, move a
   trivial helper (e.g. `escapeHtml`) to prove the loading works.
2. **Util layer** — `dom.js`, `fetch.js`, `format.js`, `sse.js`.
3. **Providers + settings panels** — self-contained forms, low coupling.
4. **Sidebar** — files, sessions, snapshots.
5. **Chat** — rendering and turn cards.
6. **Workflow editor (last, biggest)** — split into the 9 sub-modules above,
   porting the existing validated logic (nodes, edges, validation,
   json-sync) without changing behavior.
7. **Final cleanup** — delete the inline `<script>`, run the linter, verify
   every settings tab and workflow interaction end-to-end.

### Validation gates for each migration step

- [ ] `node scripts/lint-html.js` passes (IDs, handlers, syntax).
- [ ] `go vet ./...` + `go test ./...` pass.
- [ ] Manual click-through: open Settings, each tab saves; open Workflow Lab,
      drag a node, wire two ports, add/delete a node, Compile & Apply; send a
      chat message and confirm streaming + tool cards render.
- [ ] No console errors on load.

---

## 5. Progress Checklist

### Phase A — CI (done)

- [x] `scripts/lint-html.js` (tag balance, duplicate IDs, handler
      resolution, href targets, inline JS syntax).
- [x] Fixed two real bugs the linter immediately found
      (`toggleAddNodeMenu`, `autoLayoutNodes` were called but undefined).
- [x] Fixed `process.exitCode` initialization bug in the linter.
- [x] Robust repo-root resolution (works from any cwd).
- [x] Verified linter **fails** on broken HTML and **passes** on clean.
- [x] `Makefile` with `lint` / `test` / `build` / `all` targets.
- [x] `.github/workflows/ci.yml` with `lint-test` + matrix `build` jobs.
- [ ] First green CI run on GitHub after push (verify in Actions tab).

### Phase B — ES module scaffold

- [ ] Create `src/web/js/` directory.
- [ ] Add `<script type="module" src="/js/app.js"></script>` to `index.html`.
- [ ] Move `escapeHtml` (and prove module loading works end-to-end).
- [ ] Confirm Go `//go:embed web/*` picks up the new `js/` folder.

### Phase C — Utility modules

- [ ] `js/util/dom.js`
- [ ] `js/util/fetch.js`
- [ ] `js/util/format.js`
- [ ] `js/sse.js`

### Phase D — Settings & providers

- [ ] `js/providers/providers.js`
- [ ] `js/settings/settings.js` (modal + tabs)
- [ ] `js/settings/api-config.js`
- [ ] `js/settings/compaction.js`
- [ ] `js/settings/sandbox.js`
- [ ] `js/settings/mcp.js`

### Phase E — Sidebar

- [ ] `js/sidebar/sidebar.js`
- [ ] `js/sidebar/files.js`
- [ ] `js/sidebar/sessions.js`
- [ ] `js/sidebar/snapshots.js`

### Phase F — Chat

- [ ] `js/chat/chat.js`
- [ ] `js/chat/turn-cards.js`
- [ ] `js/chat/metrics.js`
- [ ] `js/chat/edits.js`

### Phase G — Workflow editor

- [ ] `js/workflow/nodes.js`
- [ ] `js/workflow/validation.js`
- [ ] `js/workflow/json-sync.js`
- [ ] `js/workflow/canvas.js`
- [ ] `js/workflow/edges.js`
- [ ] `js/workflow/wiring.js`
- [ ] `js/workflow/inspector.js`
- [ ] `js/workflow/toolbar.js`
- [ ] `js/workflow/ai-compiler.js`
- [ ] `js/workflow/live-trace.js`
- [ ] `js/workflow/workflow.js` (orchestrator + public API)

### Phase H — Wrap-up

- [ ] Delete the inline `<script>` block from `index.html`.
- [ ] Run all validation gates (lint, vet, test, manual click-through).
- [ ] Update `docs/V2_VISUAL_EDITOR.md` if module names/locations changed.
- [ ] Tag/push and confirm CI is green.

---

## 6. Decisions Log

| Date | Decision | Rationale |
| ---- | -------- | --------- |
| 2026-08-16 | Use ES modules, no bundler | Zero-dependency ethos; Go already embeds/serves `web/`; ES modules work over HTTP in all target browsers. |
| 2026-08-16 | External npm dep allowed in **CI only** | A real HTML parser can be added later if the custom linter outgrows itself; the runtime stays pure. |
| 2026-08-16 | Custom dependency-free HTML linter first | Catches the specific bugs we've hit with zero install and trivial CI. |
| 2026-08-16 | Build artifacts on every push (not just tags) | Fast feedback, free on GitHub runners, 30-day retention; binaries are small (~7 MB). |
| 2026-08-16 | Migrate workflow editor last | It's the largest/most coupled area; doing smaller modules first de-risks the mechanics of the split. |

---

## 7. Open Questions

- Should HTML linting eventually use a real parser (e.g. `parse5` /
  `htmlhint`) as an npm dev dependency? The custom checker covers our known
  failure modes but isn't a full validator.
- Do we want a `jsconfig.json` for editor IntelliSense on the modules, even
  without a build step?
- Should the AI Workflow compiler (currently a canned two-template stub) be
  replaced with a real LLM call as part of this work, or left for later?
