# 🗺️ GoHarness Local Agent: Engineering Roadmap

This document outlines the strategic engineering roadmap for evolving **GoHarness** from a lightweight, single-file prototype into a **production-grade, local-first, modular, and sandboxed AI Agent Runner**.

While GoHarness is structurally and functionally built as an elite coding companion, its underlying systems (plain-text turn serialization, sliding-window compaction, and API gateways) are fully agnostic. It is designed as a **generalized, high-performance cognitive shell** optimized for:
1. **💻 Bare-Metal Systems Engineering & Coding:** Running sandbox terminals, patching code, and writing projects.
2. **✍️ Long-Form Creative Writing:** Organizing research files, compiling structures, and managing outlines.
3. **💬 Infinite-Memory Conversational Chatting:** Retaining deep historical context across months of ongoing, detailed, local-first chat sessions.

---

## 📅 Roadmap Overview

```
┌───────────────────────────────────┐
│ PHASE 1: CONFIG & INJECTION       │ ◄── [Complete]
│ - config.json with guardrails     │
│ - AGENTS.md / SKILLS.md parser    │
│ - Token-Safe Auto-LS Scanner      │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 2: HIGH-EFFICIENCY CORE     │ ◄── [Complete]
│ - Semantic Diff Patching          │
│ - Turn-by-Turn File Persistence   │
│ - Dual-Engine Workspace Rollbacks │
│ - Sliding-Window Compaction       │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 3: MCP CLIENT INTEGRATION   │ ◄── [Complete]
│ - JSON-RPC 2.0 over Stdio/SSE     │
│ - Dynamic Tool Discovery          │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 4: EMBEDDED OS SANDBOXING   │ ◄── [Complete]
│ - Windows AppContainer / Jobs     │
│ - Linux Landlock LSM              │
│ - macOS sandbox-exec (SBPL)       │
│ - System Write-Protection Shields │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 5: ENTERPRISE & PACKAGING   │ ◄── [Complete]
│ - Zero-dependency Python Bundle  │
│ - CI/CD Cross-Compilation         │
│ - Metadata Watcher & Trace Logs   │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 6: EMBEDDED WEB CONSOLE     │ ◄── [Complete]
│ - HTML5/JS Single-Page App (SPA)  │
│ - Workspace history and switching │
│ - Parallel Session Branching      │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 7: MULTI-PROVIDER CONNECTORS│ ◄── [Complete]
│ - Anthropic Claude Messages API   │
│ - Google Gemini (AI Studio) API   │
│ - Google Gemini (Vertex AI OAuth) │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 8: OPENAI COMPATIBLE GATEWAY│ ◄── [Complete]
│ - GET /v1/models discovery        │
│ - POST /v1/chat completions proxy │
│ - Tokenizer & Embedding Proxies   │
│ - Dynamic Token Counters & Cost   │
│ - Persistent Resumes & Debugging  │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 9: HIERARCHICAL COGNITIVE   │ ◄── [Planned Extension]
│          MEMORY & VISUAL TIMELINE │
│ - O(1) Index-based range loader   │
│ - Hierarchical Memory Tree decay  │
│ - Visual Memory Map Dashboard     │
│ - Selective Forget / Regenerate   │
└─────────────────┬─────────────────┘
                  │
                  ▼
┌───────────────────────────────────┐
│ PHASE 10: CONCURRENT MULTI-AGENT  │ ◄── [Added Extension]
│           DELEGATION & WATCHERS   │
│ - spawn_sub_agents tool schema    │
│ - Goroutines & Channels pipeline  │
│ - sync.WaitGroup (Wait All Mode)  │
│ - context.WithCancel (Wait First) │
└───────────────────────────────────┘
```

---

## 📂 Phase 1: Configuration & Local Instruction Injection

### 1.1 `config.json` Architecture
Instead of hardcoding APIs or relying exclusively on environment variables, the agent boots by parsing a local `config.json` file. This controls runtime limits, model parameters, and safety parameters.

### 1.2 Dynamic Instruction Injection (`AGENTS.md`, `SKILLS.md`)
To make the agent customizable without recompilation, the Go binary scans the current directory on boot and dynamically injects any local markdown-based guidelines into the **LLM System Prompt**.
* **`AGENTS.md`:** Defines the identity, behavior, and limits of the agent.
* **`SKILLS.md`:** A dictionary of reference code, syntax requirements, or specialized workflows.

### 1.3 Token-Safe Directory Tree Scanner (Auto-LS)
To give the agent instant situational awareness without wasting an execution turn running `ls` or `dir`, the Go harness automatically performs a recursive directory walk on boot and injects a formatted directory tree into the initial prompt.
* **Ignored Patterns:** Heavy folders like `.git`, `.DS_Store`, or build caches are completely hidden.
* **Collapsed Patterns:** Dependency folders like `node_modules` or `.venv` are marked as `[collapsed]`. The agent is aware of their presence but doesn't swallow thousands of file paths.
* **Truncation Limits:** Folders containing more than $15$ files are truncated with a `... (X more files truncated)` label to protect the LLM's context window.

---

## ⚡ Phase 2: High-Efficiency Core & Session Management

These highly-optimized, "no-bloat" features (inspired by modern developer agents like *Claude Code* and *Pi*) ensure the runner remains lightning fast, mathematically cheap to run, and highly resilient.

### 2.1 Semantic Diff Patching (`patch_file` tool)
* **The Concept:** Instead of requiring the LLM to rewrite a $1,000$-line file to edit a single line of code, we introduce a `patch_file` tool.
* **The Flow:** The LLM provides the file path, the exact block of code to search for, and the new code to replace it with.
* **Implementation:** The Go harness performs a fast in-memory string replacement (`strings.Replace`) and saves it. If the target string isn't found, it returns a precise contextual warning allowing the LLM to self-correct.
* **The Value:** **Reduces completion token usage by up to 90%** on file edits, guarantees $10\times$ faster write speeds, and prevents file-rewriting corruption.

### 2.2 Conversational Time-Travel (Turn-by-Turn File Persistence)
* **The Concept:** Instead of utilizing a heavy binary database like SQLite, GoHarness maintains **100% human-readable plain-text JSON files**. This offers exceptional observability, allowing developers to inspect, modify, or debug the agent's exact memories using any standard text editor.
* **Implementation:** Inside `.goharness/sessions/<session_id>/`, every single execution turn (user message, assistant thought, or tool response) is saved as an individual, sequentially numbered, and timestamped JSON file:
  ```
  .goharness/sessions/sess_91a0b3/
  ├── 001-user-2026_08_01_13-45-02.json
  ├── 002-assistant-2026_08_01_13-45-08.json
  ├── 003-tool-write_file-2026_08_01_13-45-12.json
  └── 004-assistant-2026_08_01_13-45-20.json
  ```
* **Timeline Forking (`/fork <turn>`):** Typing `/fork 3` simply deletes files numbered higher than `003` from disk. The Go harness reloads the remaining files, placing the conversation history exactly back in time.

### 2.3 Dual-Engine Workspace Rollbacks & Untracked Erasers (Phase 8.5)
* **The Concept:** When a conversation is rewound (via `/fork <turn>`), the physical files in your workspace directory must be reverted to match that exact point in time. If we only rewind the chat history but leave edited files unchanged, the agent's memory becomes out-of-sync with the disk.
* **Engine 1: Git-Native Checkpoints (Primary):**
  - Before executing any file-modifying tool (like `write_file` or `patch_file`), the Go harness checks if the directory is a Git repository.
  - If a Git repo is detected, the harness automatically creates an in-memory Git stash or checkpoint commit: `.goharness/checkpoint-turn-X`.
  - On a conversation rollback (`/fork 3`), the harness automatically executes a fast `git reset --hard` back to the checkpoint of Turn 3, perfectly restoring deleted, added, or modified files.
* **Engine 2: Lightweight Local Backup Stashing (Fallback):**
  - If `git` is missing or the folder is not a repository, GoHarness falls back to a built-in, zero-dependency file-backup system.
  - Right before a file is written or patched, the Go harness makes a backup copy inside `.goharness/sessions/<session_id>/backups/<turn_number>/<filepath>`.
  - On rollback, the harness copies the backed-up files from that turn back to their original locations, discarding newer files.
* **Untracked Created File Tracking (Phase 8.5):** If a file did *not* exist when we attempt to edit/write it (meaning it was newly created during that turn), GoHarness stashes a hidden `.untracked_new` marker. On rollback, the engine reads these signatures and **physically deletes the newly created files** from disk, ensuring complete, bare-metal folder state alignment.

### 2.4 Sliding-Window Context Compaction
* **The Concept:** As conversation length scales, history eats up massive context tokens. We trigger a background "compaction" to compress older turns while preserving recent fine-grained conversational turns.
* **Configurability:** We expose complete control over compaction inside `config.json`, allowing users to tweak how summaries are written and which model writes them:
  ```json
  "compaction": {
    "auto_compact_turns": 40,
    "model": "gpt-4o-mini",
    "temperature": 0.2,
    "system_prompt": "You are a context compaction engine. Summarize the files modified, the current bugs resolved, and the active task plan in a highly dense, bulleted state summary."
  }
  ```
* **Sliding-Window State Reconstruction:** The engine tracks exactly which turn was compacted. On turn $42$, if compaction occurred at turn $40$, the engine constructs the LLM context dynamically by combining:
  1. **Global Instructions:** Active system prompts (including `AGENTS.md` and `SKILLS.md`).
  2. **The Compacted Summary:** Injected as a consolidated state file (`compacted_summary.json` containing the turn $1-40$ summary).
  3. **The Uncompacted Window:** The raw, turn-by-turn history files starting from turn $41$ onwards.

---

## 🔌 Phase 3: Model Context Protocol (MCP) Server Support

The **Model Context Protocol (MCP)** is an open standard that allows clients (our Go harness) to connect securely to modular external servers (databases, search engines, file systems) using a standardized **JSON-RPC 2.0** protocol over standard input/output (`stdin`/`stdout`).

### 3.1 Stdio Transport (Spawn on Demand)
* On boot, the Go binary scans its `config.json` for registered local MCP servers (e.g., SQLite, GitHub API, Brave Search).
* It spawns each server as an isolated OS child process and intercepts their standard pipes (`cmd.StdinPipe()` and `cmd.StdoutPipe()`).
* Communication is done strictly in-memory via JSON-RPC 2.0 messages—**no local network ports are opened**, ensuring $100\%$ local network security.

### 3.2 Dynamic Tool Discovery & Registration
* The Go client queries the spawned servers: `tools/list`.
* The servers return their standard schemas.
* The Go client automatically translates these schemas and registers them in the LLM's active tools list, making third-party tools instantly usable without writing a single line of Go.

---

## 🔒 Phase 4: Native Embedded OS Sandboxing & Protection Shields

To eliminate external dependencies like Docker or dedicated hypervisors, the Go application uses compile-time build tags to enforce native, lightweight, bare-metal containerization.

### 4.1 System Directory Write-Protection Shields
* **The Vulnerability:** Even if the file system is sandboxed, a rogue or hijacked LLM (via prompt injection) could try to delete its own `.goharness/` directory to erase logs, override security whitelists, or leak session trace keys.
* **The Shield:** The Go harness executor implements an in-process, absolute write-protection rule.
* **How it works:** In `write_file` or `patch_file` tools, the Go binary parses the target file path. If the path resolves to `.goharness`, `config.json`, or parent directories outside the active workspace, the Go harness immediately terminates the tool call and returns a native system error to the LLM: *"Security Exception: System folder is write-protected."*
* **Docker Isolation:** In `-sandbox=docker` mode, this is physically enforced because the container only mounts the `workspace/` subfolder, leaving the parent host's `.goharness` folder completely invisible and unreachable by the container.

### 4.2 Linux Landlock LSM Implementation
* **Zero-dependency sandboxing** using Linux kernel namespaces and Landlock system calls (`sys_landlock_create_ruleset`, `sys_landlock_add_rule`, `sys_landlock_restrict_self`).
* Restricts both the Go program's threads and any child processes (like the python interpreter) from writing or reading anywhere outside the `./agent_workspace/` directory and standard system libraries.

### 4.3 Windows Low-Integrity & Job Objects
* Duplicate the current process token using `advapi32.dll` to establish a **Low-Integrity Token (`S-1-16-4096`)**.
* Restrict child process writing privileges to ensure files outside the workspace are write-protected.
* Wrap the child inside a **Windows Job Object** to enforce absolute ceilings on CPU usage, physical memory allocation, and prevent subprocess escapes.

### 4.4 macOS Sandbox-exec (SBPL)
* Generate an in-memory **Sandbox Profile Language (SBPL)** file.
* Spawn the embedded Python compiler inside Apple's native, kernel-enforced `/usr/bin/sandbox-exec` utility.

---

## 📦 Phase 5: Enterprise, Distribution & Sync

### 5.1 Portable Runtime Extraction Engine
* Compress a minimalist Python 3.11 build into an archive.
* Embed it inside the compiled binary using `//go:embed`.
* On the first boot, extract it transparently to the user's cache folder.

### 5.2 Metadata-Based Workspace Watcher & Auto-LS Integration
* **Unified UI Feed:** Instead of treating the workspace file-watcher as a separate stream of messages, we integrate file modification directly into our **Token-Safe Directory Tree (Auto-LS)**.
* **How it works:** A background Go ticker checks the standard file modification times (`ModTime()`) on a $2\text{-second}$ interval.
* **Visual Status Flags:** When generating the tree, the scanner dynamically appends visual metadata notes and Git-like status flags inline:
  ```
  === WORKSPACE DIRECTORY TREE ===
  ├── config.json [Modified 45s ago]
  ├── agent.go [Modified 2 mins ago]
  ├── schema.sql [New / Untracked]
  └── package.json [Unchanged]
  ```
* **The Benefit:** Gives the model an intuitive, single-source-of-truth visual feed of what is actively changing inside the workspace in real-time, preventing the agent from operating on stale, out-of-sync code.

### 5.3 Structured Execution Trace Logging
* The Go harness appends structured logging metadata to a local hidden directory `.goharness/traces.jsonl` on every action (tools called, durations, system metrics, exit codes).
* This provides a rich diagnostic trace dataset to optimize system prompts and identify execution performance bottlenecks.

### 5.4 Cross-Platform Compilation Matrix (CI/CD)
Using **GitHub Actions**, compile the static Go binary into a matrix of release targets on every release (Windows x64/arm64, macOS Apple Silicon/Intel, Linux x64/arm64).

---

## 🌐 Phase 6: Built-in Web Console (Visual GUI)

To make GoHarness accessible to non-CLI users and provide a visually rich dashboard that mirrors the TUI activity, we introduce a **Built-in Web Console** served directly from the compiled binary.

### 6.1 Zero-Dependency Embedded Assets
* **The Architecture:** The entire frontend is compiled directly inside the Go executable using `//go:embed`.
* **Zero-Setup Server:** The Go binary contains a built-in `net/http` router. When run with `./agent -web` (or flagged in `config.json`), Go spins up a lightweight, highly secure background web server (e.g. `http://localhost:8080`).
* **Auto-Launch:** Go automatically opens the default web browser on boot.

### 6.2 Bidirectional Real-time streaming (SSE)
* The web browser connects to the Go binary via a persistent Server-Sent Events (SSE) connection, streaming assistant tokens and tool executions live.

### 6.3 Dashboards, Workspaces, & Parallel Branching
* **Workspace Swapper:** A dropdown selector lets users switch between independent local workspace folders dynamically, updating config history.
* **Conversations Indexer:** Full list of previous sessions (creation times, active folders) is mapped, allowing developers to switch sessions and resume them in one click.
* **Non-Destructive Fork (Branching):** Clicking the rollback button on any previous turn prompts a modal dialog to **Branch Timeline**. This clones the conversation state up to that turn into a brand-new, isolated session (leaving the original completely safe on disk!) and physically restores your folder files to exactly match that historical point in time.

### 6.4 Advanced Workspace Control Center, Snapshotting, and Exclusions Management
* **Workspace Snapshotting & Reverting Engine:** Introduce a high-fidelity workspace state-saving system. Clicking "Capture Workspace" copies the entire file tree (skipping system/ignored paths) into a unique folder under `.goharness/snapshots/` with custom names. Users can instantly "Revert" their workspace files to any snapshot with a single click, completely resetting physical files while leaving log histories pristine.
* **Scan Exclusions & Collapses Panel:** Dynamically configure active `DirectoryScan` rules from the visual Settings panel. Allows developers to add and remove ignore/collapse patterns (e.g. `*.log`, `node_modules`, `venv`) in real-time. Changes are applied instantly, updating the collapsible directory explorer tree on the fly.
* **Dynamic Workspace History Control:** Expanded visual history tracking below the workspace swapper. Allows swapping workspaces instantly or deleting obsolete workspaces from config history with a single click.
* **Interactive Session Actions (Rename & Delete):** Each session card on the dashboard now features hover actions to rename or delete sessions. Deleting sessions physical purges target folders, and renaming sessions modifies metadata indices on-disk instantly.

---

## 🔌 Phase 7: Multi-Provider LLM Connectors

To turn GoHarness into a completely flexible multi-model agent engine, we introduce a **Multi-Provider Translation Layer** built in-house with zero external library dependencies.

### 7.1 Unified Translation Engine (`src/llm.go`)
* We introduce a `"provider"` parameter inside `config.json`.
* An in-process translation engine dynamically maps standard core `Message` and `Tool` structures to the specific, non-OpenAI JSON layouts of different cloud providers.

### 7.2 Native Anthropic Claude Messages API
* **System Prompt Extraction:** Automatically separates system prompts from the core message array, passing them as top-level parameters matching Anthropic's guidelines.
* **Structure Mapping:** Maps assistant tool calls into custom `"tool_use"` blocks, and maps tool responses back as `"tool_result"` blocks inside standard user turns.
* Uses the required API headers: `x-api-key` and `anthropic-version: 2023-06-01`.

### 7.3 Native Google Gemini API (AI Studio & Vertex AI)
* **Role Conversion:** Automatically maps conversational roles strictly to `"user"` or `"model"`.
* **Function Schema Handshaking & thoughtSignature echos (Phase 8.6):** Maps tool schemas to Google's `"functionDeclarations"`, tool calls to `"functionCall"`, and tool responses to `"functionResponse"` objects. It automatically parses and echos back Google's native `"thoughtSignature"` parameter on subsequent history turns to ensure $100\%$ REST compliance during reasoning model tool executions.
* **Auth Support:** Handles query-parameter key handshakes for Gemini AI Studio, and dynamic key/GCP region URL builders for Google Vertex AI integrations.

---

## 🌐 Phase 8: OpenAI-Compatible API Gateway

To turn GoHarness into a unified local agentic middleware that any standard third-party LLM frontend can plug into, we expose a secure, standard-library-only OpenAI API Gateway.

### 8.1 API Discovery & Completions Proxies
* **`GET /v1/models`:** Serves static model descriptors so frontends like OpenWebUI, SillyTavern, and LibreChat can perform auto-discovery on boot.
* **`POST /v1/chat/completions`:** Exposes a standard completions handler. It intercepts standard user chat messages, programmatically spawns our background secure bare-metal tool-use sandbox loops in the background to write code or execute shell scripts, and streams the final compiled solution back in standard OpenAI JSON chunks.

### 8.2 Intelligent Vector Embeddings Proxy (`/v1/embeddings`)
* Exposes a standardized embeddings endpoint.
* When your frontend wants to index documents or read PDFs, it POSTs texts to GoHarness. GoHarness proxies these blocks directly to your active model provider (OpenAI, Gemini, or local Ollama) and returns standard OpenAI vector coordinates back, allowing your frontend to run localized RAG over a single connection port.

### 8.3 Native Tokenizer & Detokenizer Proxies (`/v1/tokenize` & `/v1/detokenize`)
* **The Importance:** Developer frontends require real-time token counts to calculate prompt limits, budget context allocations, and truncate history on the client side before dispatching requests.
* **The Implementation:** We expose standard `/v1/tokenize` and `/v1/detokenize` (and their short-paths `/tokenize` and `/detokenize`) endpoints:
  - **Local Model Syncing:** If a local model (via Ollama or llama-server) is selected, GoHarness dynamically forwards the text or tokens directly to the local model's native `/api/show` or `/tokenize` endpoints, returning $100\%$ accurate token metrics.
  - **Cloud API Fallbacks:** If a cloud-based API is selected, GoHarness implements a fast, highly optimized Byte-Pair Encoding (BPE) tokenizer approximation locally in Go (similar to a compiled Tiktoken/llama.cpp tracker) to calculate token counts instantly in-process without any network latency.

### 8.4 Cumulative Token Metrics & Cost Trackers (Phase 8.6)
* The Go Gateway parses real-time usage token nodes from all cloud providers and calculates the exact session cost in USD. 
* It serves a dual-indicators **Token and Character Counter Widget** directly inside the Web UI, which increments live on every streamed SSE packet.
* Local and offline models (via Ollama or server) are automatically calculated as **`$0.0000` (Free)** to highlight the benefits of running local inferencing engines.

### 8.5 Context Persistence & Developer Debug Modes (Phase 8.6)
* Automatically caches and resumes your most recent active Session ID on boot.
* Exposes a `"debug"` configuration toggle inside `config.json` and the settings form, enabling verbose diagnostic console logging to troubleshoot directories, files, or JSON-RPC pipe transactions in real-time.

---

## 🧠 Phase 9: Hierarchical Cognitive Memory & Visual Timeline Manager

To equip the agent with infinite long-term memory while maintaining exact workspace boundaries, we introduce an **OptMem-inspired Hierarchical Memory Subsystem** complete with a **Visual Memory Map Timeline** inside the browser.

### 9.1 $O(1)$ Range Loader & Hierarchical Memory Decay
* **The Concept:** Following Victor Taelin's fixed-width, zero-index architecture, GoHarness decouples older conversation history into logarithmic, hierarchical summaries (e.g., Level-64, Level-16, Level-4 summaries) and stores them in individual nested files.
* **The Speed:** Loading session history becomes an **$O(1)$ direct index range check** (directly targeting and reading files from Turn X to Y) rather than an $O(N)$ recursive directory search, guaranteeing instant load times even for $10,000\text{+ turn}$ sessions.
* **Dynamic Decaying Context:** When compiling the prompt, the engine streams a decaying resolution timeline to the LLM: the ancient past is presented as ultra-compressed macro summaries, the middle past as medium-level bullet points, and the active window as exact verbatim JSON turns.

### 9.2 The Visual "Memory Map" Dashboard
* **The Panel:** We introduce a dedicated **"Timeline / Memories"** panel tab inside the Web Console.
* **The Visual Map:** It renders a **collapsible, chronological nested tree** mapping exactly how the agent's mind has compacted.
* **Visual Selective Forget:** Beside every compiled summary card, we expose an interactive **Regenerate (Forget)** button. Clicking it issues a background call to delete that specific summary block file on disk. The Go harness's next background execution will automatically re-summarize that specific turn range, letting the user "cleanse" a bad summary, edit its text, or correct a faulty agent thought block with single-click ease!
* **Branch From Summary:** Beside each memory node, a **Branch** button lets you spawn a parallel conversation timeline starting precisely from the state of that compiled memory block!

---

## 🌐 Phase 10: Concurrent Multi-Agent Delegation & Orchestration

This phase implements multi-agent parallelization, allowing a parent manager agent to delegate complex, asynchronous tasks to multiple sandboxed child agents concurrently, with complete programmatic synchronization controls.

### 10.1 Concurrency Pipeline Architecture (Goroutines & Channels)
Go is the single most powerful concurrent programming language in existence, bypassing the threading bottlenecks of Python (Global Interpreter Lock - GIL) and single-threaded Node.js. GoHarness leverages this by exposing a high-performance **Goroutines & Channels pipeline** to spawn, coordinate, and stream asynchronous multi-agent executions.
* We introduce a new, native parent tool called **`spawn_sub_agents`**.
* The parent agent can supply an array of independent sub-tasks, allocating them to cheap, high-speed child engines (e.g. `gpt-4o-mini`).
* Each sub-agent is spawned in its own isolated goroutine and allocated a brand-new, unique child session ID (inheriting the parent workspace's strict bare-metal sandbox rules).

### 10.2 Synchronous Wait Modes (The Coordination Schemes)
The parent agent can configure the exact synchronization strategy for the concurrent child threads via a **`wait_mode`** parameter:

#### 🟢 Option A: "Wait All" (`sync.WaitGroup`)
* **The Logic:** The parent agent delegates a broad codebase audit (e.g. *"Frontend security audit"*, *"Backend SQL leak check"*, *"Git trace check"*).
* **The Go Code:** GoHarness initializes a **`sync.WaitGroup`**. It calls `wg.Add(1)` and spawns each sub-agent concurrently.
* **The Flow:** The main orchestrator thread executes a blocking **`wg.Wait()`**. Once all child processes complete, GoHarness aggregates all of their individual markdown reports into a single, clean structural response and returns it to the parent.

#### 🟡 Option B: "Wait First / Race" (`context.WithCancel`)
* **The Logic:** The parent agent wants to solve a complex coding puzzle and spawns multiple parallel attempts with different structural prompts to see which model solves it fastest.
* **The Go Code:** GoHarness initializes a Go context with cancellation: **`context.WithCancel(context.Background())`**. It spawns all agents, passing the `ctx` into their loops.
* **The Flow:** The moment the **first** agent solves the task and writes its output to a synchronized Go channel, GoHarness immediately executes **`cancel()`**. The remaining slower child agents are instantly aborted at the OS kernel level, saving massive token expenditures while returning the winner's answer instantly.

#### 🔵 Option C: "Asynchronous / Fire-and-Forget" (Async Goroutines)
* **The Logic:** The parent agent spawns a long-running background test compilation or deployment script and does not want to freeze its own thread waiting for it.
* **The Flow:** GoHarness spawns the sub-agent asynchronously. It immediately returns a success status alongside the child's **Session ID** back to the parent. The parent can proceed with other tasks and query the child's status or read its written outputs later.

> **Implementation status (2026-08):** The synchronous fan-out/fan-in core of Phase 10 is shipped as the `spawn_sub_agent` tool (structured `task`/`context`/`expect` inputs, per-profile concurrency throttle, workspace write lock, recursion depth cap of 2, and a live "Sub-agents" progress card). Wait-First/Race and durable background jobs remain open — see 11.7.

---

## 🧩 Phase 11: Ergonomics & Platform Backlog (Pi Comparative Analysis)

This phase catalogues improvements identified by comparing GoHarness against [Pi](https://github.com/earendil-works/pi) (`@earendil-works/pi-coding-agent`) and its most-used extension ecosystem. Items are tagged **[NEW]** (a capability GoHarness lacks) or **[IMPROVEMENT]** (extends something we already have). Each lists the reference implementation and concrete technical notes so the work can be picked up without re-research.

Pi's architecture is the inverse of ours: a minimal TypeScript/Bun TUI kernel where *everything else* — sub-agents, MCP, plan mode, web access, LSP feedback, memory — ships as a versioned extension package. We have chosen to productize those features in-core, so most of these are additive on top of foundations we already have (the `Agent` runtime, DAG engine, profiles, MCP client, BM25 index) rather than redesigns.

Reference repos:
- **Pi core:** https://github.com/earendil-works/pi
- **pi-web-access** (web search/fetch/video/PDF/GitHub): https://github.com/nicobailon/pi-web-access — npm `pi-web-access`
- **pi-subagents** (named roles, fleet view, background runs): https://github.com/nicobailon/pi-subagents — npm `pi-subagents`
- **pi-mcp-adapter:** npm `pi-mcp-adapter` (we already have MCP in-core)
- **pi-lens** (LSP/lint/typecheck feedback): npm `pi-lens`
- **pi-background-tasks** (durable shell + delegated agents): npm `pi-background-tasks`
- **pi-hermes-memory / remnic / pi-memory** (cross-session semantic memory)
- **pi-goal / pi-goal-list-loop-audit** (autonomous goal/todo loops)
- **@juicesharp/rpiv-ask-user-question** (model-initiated structured Q&A)
- **@quintinshaw/pi-dynamic-workflows** (text-driven fan-out across many agents)

### 11.1 Web search & fetch tools **[NEW]**
* **Reference:** `pi-web-access` registers `web_search`, `fetch_content`, `get_search_content`, `source_check`.
* **What it actually does (verified from source, v0.24.2):** there is **no headless browser**. Dependencies are just `undici` (HTTP), `linkedom` (HTML parse), `@mozilla/readability` (Firefox Reader View article extraction), `turndown` (HTML→Markdown), `unpdf` (local PDF text), and `p-limit` (concurrency cap). The pipeline for a page is: manual-redirect `fetch` with SSRF validation per hop → check `Content-Type` → parse HTML with `linkedom` → run Readability → convert article HTML to Markdown with Turndown. If Readability fails, it tries Next.js RSC flight data; if the page looks JS-rendered it returns an explicit error rather than hallucinating. JS-heavy pages are delegated to *remote hosted* extractors (Jina Reader `r.jina.ai`, Firecrawl, Kagi, Bright Data, Gemini URL Context) which run headless browsers server-side — never locally.
* **Search:** provider JSON APIs (OpenAI Responses, Brave, Exa, Tavily, etc.) with an ordered fallback chain; zero-config defaults to keyless Exa MCP + DuckDuckGo HTML scraping; self-hosted SearXNG if configured.
* **Smart extras:** GitHub URLs are `git clone --depth 1`'d locally instead of scraped; PRs/issues via `gh pr view`/`gh issue view`; YouTube via Gemini transcript+frames; PDFs via a Datalab→Gemini→local-unpdf fallback chain.
* **GoHarness plan:** two tools — `web_search.go` and `web_fetch.go`.
  - Search backend interface with fallback chain: self-hosted SearXNG (if configured) → DuckDuckGo HTML scrape (keyless) → Tavily/Brave/Exa (keyed via Providers screen).
  - Pure-Go fetch pipeline: `net/http` with manual redirect following + SSRF guard (block RFC1918/loopback/link-local, validate DNS per hop, cap 5 MB, 30 s timeout, max 3 concurrent) → `github.com/nickstenning/html-to-markdown` (or a Go Readability port) → Markdown. Detect GitHub URLs and `git clone --depth 1` into a per-session cache. Detect PDFs and use `github.com/ledongthuc/pdf` for local text.
  - JS-heavy fallback: route through Jina Reader (`https://r.jina.ai/<url>`), server-side Markdown, free tier — opt-in per call.
  - Content cache outside the session JSON (Pi uses `~/.pi/web-search-cache/`, 1-hour TTL, 128 entries / 128 MiB LRU, `0600` perms), plus a `get_web_content(id, offset, limit, find_text)` tool so large pages don't flood context.
  - `source_check(claim)` nice-to-have: 2–3 searches + top-page fetches → `supported/contradicted/unclear/missing-evidence` verdict with exact passage offsets and SHA-256 hashes.
* **Effort:** M (~600–900 LOC across 4 files + two tool schemas + `httptest` tests).

### 11.2 Message queue while the agent is running (steering & follow-ups) **[NEW]**
* **Reference:** Pi editor — Enter while working queues a *steering* message delivered after the current tool batch; Alt+Enter queues a *follow-up* delivered after all work settles; Escape aborts and restores queued text; `steeringMode`/`followUpMode` settings.
* **Why it matters:** on long workflows or parallel sub-agent runs, the user currently can't interject without waiting.
* **GoHarness plan:** the run loop already alternates LLM → tools → LLM. Add a thread-safe queue (`Agent.pendingSteering`, `Agent.pendingFollowup`) fed from a web endpoint. After each `runToolCalls`, inject steering messages as user turns; after the loop settles, inject follow-ups. Web composer shows a "queued" indicator; Escape cancels the in-flight request via `context.Cancel` and restores the draft.
* **Effort:** S-M.

### 11.3 Cancel vs. abort distinction **[IMPROVEMENT]**
* **Reference:** Pi — Escape once cancels the current tool/LLM call but keeps the turn and queued messages; Escape twice kills the whole turn.
* **GoHarness plan:** first Escape cancels only the in-flight request (agent records "interrupted by user", stops calling tools); a second Escape within ~1 s cancels the parent run context. Surface as distinct "Stop" vs. "Abort" buttons.
* **Effort:** S.

### 11.4 `@file` mentions in the composer **[NEW]**
* **Reference:** Pi `@` fuzzy-searches project files and attaches contents.
* **GoHarness plan:** we already BM25-index the workspace. Add a typeahead in the web composer triggered by `@`; on selection, read the file (respecting tree-scanner ignore patterns) and append its contents as a fenced block, or send a `context_files` array the backend injects as context. Reuse `GenerateWorkspaceTree`.
* **Effort:** S (mostly frontend + `/api/files?q=`).

### 11.5 `!command` shell injection in chat **[NEW]**
* **Reference:** Pi `!command` runs a shell command and sends output to the model; `!!command` runs without sending.
* **GoHarness plan:** intercept lines beginning with `!` in the composer; run via `executeTerminalCommand`; `!` appends output as a user message and submits, `!!` streams to chat only.
* **Effort:** XS.

### 11.6 Session tree / time-travel navigator **[IMPROVEMENT]**
* **Reference:** Pi stores sessions as one JSONL file with `id`/`parentId` per entry, enabling `/tree` — an in-place searchable branch navigator. Our fork copies an entire session directory.
* **Why it matters:** cheap branching, labels/bookmarks, filter modes, non-destructive compaction.
* **GoHarness plan:** adopt JSONL as an *additional* index alongside existing turn files: write each event as one JSONL record with a stable ULID and `parent_id`. Branching appends rather than duplicating. Web sidebar gets a tree view, labels, search. Compaction writes a summary record but leaves prior records intact.
* **Effort:** L (migration/dual-write is delicate; do incrementally with import/export).
* **Companion pieces (from Pi):** `/copy` last response, `/export` to HTML/JSONL/Markdown, `/import` to resume, `/share` (private gist with rendered HTML).

### 11.7 Durable/background sub-agent jobs **[IMPROVEMENT]**
* **Reference:** `pi-subagents` FleetView + `pi-background-tasks`; agents run after the parent turn settles with a persistent inspector for transcripts/steer/stop.
* **GoHarness plan:** on top of the shipped engine, add a job registry (in-memory map + `.goharness/jobs/<id>.json`): `id, parent_session, spec, status, started_at, finished_at, result_path`. `spawn_sub_agent` gains `background: bool`; background jobs return immediately and continue in a goroutine. Reuse existing `subagent_start`/`subagent_done` SSE events; add `subagent_log`. A web "Fleet" panel lists jobs with stop/steer. Steering queue (11.2) feeds into this.
* **Effort:** M.

### 11.8 Named sub-agent roles & packaged workflows **[IMPROVEMENT]**
* **Reference:** `pi-subagents` ships `scout` (recon), `researcher` (web/docs with sources), `worker` (implementation, validates, escalates), `reviewer` (review), `oracle` (second opinion, no edits), `delegate` (general), plus `/council`, `/parallel-review`, `/review-loop` templates.
* **GoHarness plan:** a role is `{name, system_prompt, tool_allowlist, profile, temperature}`. Load YAML/TOML from `~/.goharness/agents/` and project-local `.goharness/agents/`; `spawn_sub_agent` gains an optional `agent: "reviewer"` field applying the role's tool subset and model/profile (already routed through the per-profile throttle). Ship five defaults. Workflow templates become prompt-template files (11.13).
* **Effort:** S-M.

### 11.9 Cross-session memory **[NEW]**
* **Reference:** `pi-hermes-memory`, `@remnic/plugin-pi`, `pi-memory` add semantic recall over daily logs/facts; some include secret scanning.
* **GoHarness plan:** reuse BM25. Add `memory_remember(content, tags...)` appending to dated Markdown in `~/.goharness/memory/`, and `memory_recall(query, limit)` BM25-searching them (optional embedding rerank via the Phase 8.2 embeddings proxy). Inject top-k hits into the root system prompt on session start. Auto-extract decisions/learnings at compact time.
* **Effort:** S.

### 11.10 Automatic build/lint/typecheck feedback loop **[NEW]**
* **Reference:** `pi-lens` runs LSP/linters/formatters/type-checkers after edits and feeds errors back until clean.
* **GoHarness plan:** after `write_file`/`patch_file`, detect project type (Go → `go build ./...`; Node → `npm run typecheck`/`tsc --noEmit`; Python → `py_compile`/`ruff`; Rust → `cargo check`) and run its check, capturing output. On error append a synthetic tool result so the model self-corrects, up to a small per-turn budget (e.g. 3 rounds). Per-profile/per-project setting. No LSP needed in v1.
* **Effort:** M.

### 11.11 Model-initiated Q&A interludes **[NEW]**
* **Reference:** `@juicesharp/rpiv-ask-user-question` — the model puts a structured questionnaire to the user mid-turn instead of guessing.
* **GoHarness plan:** add an `ask_user` tool taking a JSON schema of questions (text/select/multiselect). The run loop pauses, renders a form in the web UI, waits for the response, injects it as a tool result, and continues.
* **Effort:** M (async waiter + form renderer).

### 11.12 Persistent TODO/plan overlay **[NEW]**
* **Reference:** Pi core omits to-dos; `@juicesharp/rpiv-todo` adds one. Our DAG editor covers complex plans; a lightweight checklist helps linear runs.
* **GoHarness plan:** a `todo` tool (`add`/`update`/`complete`/`list`) whose state renders as a sticky web overlay surviving `/reload`; inject a compact rendering as a system message each turn. Don't over-engineer.
* **Effort:** S.

### 11.13 Skills, prompt templates, and the extension surface **[NEW — strategic]**
* **Reference:** Pi's highest-leverage feature. Extensions call `pi.registerTool()`, `pi.registerCommand()`, `pi.on("event", ...)`, replace the editor, add widgets/status lines/providers; bundled into npm/git packages with a manifest. Skills follow the [Agent Skills standard](https://agentskills.io); prompt templates are `/name`-expandable Markdown with `{{variables}}`.
* **GoHarness plan — staged:**
  1. **Prompt templates** (XS): Markdown in `.goharness/prompts/*.md`, expand via `/name` with `{{arg}}` substitution.
  2. **Skills** (S): `SKILL.md` with frontmatter (`name`, `description`, `when_to_use`) auto-loaded from `.goharness/skills/`, `~/.goharness/skills/`, and packages; offered to the model via a lookup tool and injected on description match.
  3. **Extension API** (L): a subprocess JSON-RPC protocol (modeled on MCP) letting external processes register tools, slash commands, and lifecycle hooks (`before_turn`, `after_tool`, `before_compact`, `session_switch`). Go stays one static binary; extensions can be any language. MCP already covers ~60% (external tool registration) — generalize it to commands/events. A `goharness.json` manifest declares tools/commands/skills/prompts; `goharness install git:...` places packages under `~/.goharness/`.
* **Why it matters:** compounds over time; without it our ecosystem is capped at what we ship.
* **Effort:** XS templates, S skills, L the protocol.

### 11.14 Provider/model catalog as data **[IMPROVEMENT]**
* **Reference:** Pi keeps a live, refreshable per-provider model catalog (`pi update --models`); users add custom models via `~/.pi/agent/models.json` without a release.
* **GoHarness plan:** move model lists from code into an embedded `models.json` refreshed from a versioned URL; allow override/extension via `~/.goharness/models.json`. Custom OpenAI/Anthropic/Gemini-compatible endpoints already work via profiles; this makes the picker self-updating.
* **Effort:** S.

### 11.15 Project trust & local config policy **[NEW]**
* **Reference:** Pi asks before trusting a folder with `.pi/`; before trust it loads only user-global extensions.
* **GoHarness plan:** once 11.13 adds project-local packages, show a trust modal for folders containing `.goharness/`, persist the decision, load project agents/skills/extensions only after trust. Non-interactive modes get `--approve`/`--no-approve`.
* **Effort:** S.

### 11.16 Context files: parent-directory walking + overrides **[IMPROVEMENT]**
* **Reference:** Pi loads `AGENTS.md`/`CLAUDE.md` from `~/.pi/agent/`, every parent directory, and cwd; `AGENTS.override.md` replaces that directory's file; `.pi/SYSTEM.md` replaces the system prompt wholesale; `APPEND_SYSTEM.md` appends.
* **GoHarness plan:** extend `LoadLocalInstructions()` to walk parent dirs (currently cwd only), add `.goharness/AGENTS.override.md`, plus full system-prompt override (`.goharness/SYSTEM.md`) and append (`.goharness/APPEND_SYSTEM.md`). Big value for monorepos.
* **Effort:** XS-S.

### 11.17 Richer live status & cost footer **[IMPROVEMENT]**
* **Reference:** Pi's footer shows cwd, session name, input tokens (↑), output (↓), cache reads (R), cache writes (W), cache-hit rate (CH), cost, context %.
* **GoHarness plan:** we already broadcast prompt/completion tokens and cost. Extend usage parsing for Anthropic `cache_read_input_tokens`/`cache_creation_input_tokens` and Gemini `cached_content_token_count`; render R/W/CH beside the existing widgets plus a context-usage % bar. Add a `/session` JSON view.
* **Effort:** S.

### 11.18 Collapsible "thinking" blocks **[NEW]**
* **Reference:** Pi Ctrl+T collapses/expands reasoning blocks; we already collapse tool calls but not thinking traces.
* **GoHarness plan:** surface thinking from provider-specific fields (Anthropic `thinking`, Gemini `thoughtSignature`, OpenAI reasoning tokens) as distinct, collapsed-by-default chat blocks with a global toggle.
* **Effort:** S-M (provider parsing is the messy part).

### 11.19 Image paste/drag into the composer **[NEW]**
* **Reference:** Pi supports clipboard image paste and drag-drop.
* **GoHarness plan:** accept images in the web composer, store under `.goharness/sessions/<id>/uploads/`, pass as `image_url`/inline image parts through all three provider translators.
* **Effort:** M.

### 11.20 Editor modal & slash command palette **[NEW]**
* **Reference:** Pi Ctrl+G opens `$VISUAL`/`$EDITOR`/nano for long prompts; `/` opens a palette mixing commands, skills, templates.
* **GoHarness plan:** web can't launch a local editor, so add a full-screen editor modal (we already have modals) for long prompts and a `/` palette listing commands/workflows/templates/agents with keyboard nav.
* **Effort:** S (web-only).

### 11.21 In-chat slash commands **[NEW]**
* **Reference:** Pi's `/compact`, `/new`, `/resume`, `/fork`, `/clone`, `/tree`, `/copy`, `/export`, `/import`, `/share`, `/reload`.
* **GoHarness status:** workflow New/Clone/Delete exist in the UI; no unified in-chat command system.
* **Plan:** a Go command registry (+ extension hook from 11.13); parse `/verb args` in the composer; autocomplete in the palette. Prioritize `/compact [instructions]`, `/new`, `/fork`, `/export jsonl|html|md`, `/agents`, `/reload`.
* **Effort:** S once the palette exists.

### 11.22 Persistent shell sessions / background shells **[NEW]**
* **Reference:** Pi ships no background bash (points at tmux); `pi-background-tasks` adds durable shells. We have one-shot `execute_command` but no PTY.
* **GoHarness plan:** lower priority. A `shell_start`/`shell_send`/`shell_poll`/`shell_kill` toolset via `github.com/creack/pty` would enable dev servers/REPLs; for v1 document `tmux`/`screen` in the system prompt instead.
* **Effort:** L — defer.

---

### Suggested sequencing

Roughly in impact-per-effort, grouped so earlier items unblock later ones:

1. **11.1 Web search & fetch** (huge capability lift; zero-config defaults)
2. **11.2 Steering messages + 11.3 cancel/abort** (felt UX for long runs; unblocks 11.7)
3. **11.4 `@file` mentions, 11.5 `!command`, 11.16 context-file walking** (XS–S polish)
4. **11.10 auto build/typecheck feedback** (closes the "success with a broken tree" failure)
5. **11.8 named sub-agent roles + 11.12 todo overlay** (leverage shipped engine)
6. **11.9 cross-session memory** (reuse BM25 + embeddings proxy)
7. **11.6 JSONL session tree + export/import/share** (foundational for 11.21 and better branching)
8. **11.11 ask-user, 11.17 cache accounting, 11.18 thinking blocks, 11.19 images**
9. **11.14 model catalog, 11.15 trust, 11.20 palette, 11.21 slash commands**
10. **11.13 extension API** (strategic platform bet; do after 1–9 inform its shape)
11. **11.7 durable background jobs** (once steering + fleet UI exist)
12. **11.22 persistent PTY shells** (defer)

---

## 🧬 Phase 12: DeepSeek Harness Comparative Backlog

This phase captures findings from comparing GoHarness to [DeepSeek Harness (`dsh`)](https://github.com/deepseek-ai/deepseek-harness) (npm `@deepseek-ai/dsh`, TypeScript/Cordis, MIT). Reference material:
- Architecture: https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/architecture.md
- Subsystems: `docs/subsystems/{core,tools,session,subagent,scope,llm-streaming,system-prompt}.md`
- Key packages under `packages/`: `core/{agent,agent-loop,session,tools,system-prompt,scope}`, `subagent/*`, `shell`, `terminal`, `lsp`, `jobs`, `plan`, `todo`, `goal`, `workflow`, `sandbox`, `hooks`, `mcp`, `spill`, `feedback`, `preset`, `session-query`, `code-runtime`, `attachment`, `e2b`, `schedule`, `skill`, `guard`.

DeepSeek Harness is architecturally the most ambitious of the three (Pi, Claude Code-style, us): built on **Cordis**, where literally *everything is a plugin* — model adapter, tool registry, session log, even the agent loop itself are replaceable Cordis services composed at boot from layered "profiles" and "bundles". It is a developer preview with compatibility-breaking changes, and its design is overkill for a single-binary Go product — but several ideas are directly transferable. Items are tagged **[NEW]** / **[IMPROVEMENT]** and note whether we adopt, adapt, or skip.

### 12.1 Capability seams: service/provider/consumer triad **[IMPROVEMENT — architectural north star]**
* **Reference:** dsh `capability-seams.md` — every swappable capability has three roles: a Service Definition (interface), Service Provider (implementation), and Consumer (the model-facing tool or loop). One provider swap changes the whole product because filesystem + subprocess + LSP + PTY all share one "execution world" (`ctx.fs` + `ctx.subprocess`); pointing those at E2B moves Bash, PTY, and LSP into a remote sandbox with no per-tool forks. We currently hardcode local execution in each tool.
* **Takeaway for GoHarness:** define small Go interfaces for the execution world — `FS` (read/write/stat), `Runner` (exec one-shot), `Terminals` (PTY sessions), and `LSPer` — and have tools depend on them, not concrete functions. Ship a `local` implementation; later add `docker`/`ssh`/`e2b`/`firecracker` providers that all tools inherit for free. This is the same idea as our provider-profile throttle map but generalized to IO.
* **Effort:** L (interfaces + threading through tools); pays off every time we add a sandbox/remote runtime.

### 12.2 Session event log as source of truth **[IMPROVEMENT]**
* **Reference:** dsh `core/session` — an append-only log of typed `SessionEvent`s (`turn/start`, `user/message`, `assistant/chunk*`, `assistant/message`, `tool/call`, `tool/result`, etc.). A runtime invariant asserts **"model-visible means logged"**: anything that reaches a model request must be reconstructable from the log. Fork, resume, transcripts, telemetry, UI replay, and persistence all derive from this one stream; raw `assistant/chunk` events preserve streaming fidelity.
* **GoHarness today:** per-turn JSON files in session folders. Fine for inspection, but fork copies a whole folder, streaming chunks aren't durably recorded (only the final message), and there's no single projection.
* **Plan:** adopt (or dual-write) a JSONL session log per session with typed events — this overlaps with 11.6. Add an invariant check in tests: the messages sent to the model must equal `deriveMessages(log)`. Record tool calls *before* execution and chunks as they stream. This unlocks replay, time-travel, and stable transcripts.
* **Effort:** L, but shared with 11.6.

### 12.3 Waterfall event hooks around every extension point **[NEW]**
* **Reference:** dsh fires typed waterfalls at `agent/pre-step`, `agent/request`, `llm/stream`, `tools/pre-execute`, `tools/execute`, `tools/post-execute`, `system-prompt/assemble`, plus emit events (`system-prompt/change`, `tool/result`). Listeners call `next()` to delegate; a listener can rewrite, reject, timeout, or observe. This is how approval policy, sandboxing, hooks (Claude-Code-style `hooks.json`), metrics, and per-call timeouts are all implemented without forking tools. The tool pipeline specifically is: `tool/call` logged → `tools/pre-execute` (hooks/permission/sandbox) → monotonic guards (deny/abstain) → `ctx.approval` one-shot prompt → `tools/execute` (around: timeout/retry/metrics) → tool body → `fs/*` intent gates → owned session events → `tools/post-execute` (accept/block/replace/add context) → normalization → `finalizeContent` → `tools/result`.
* **GoHarness today:** we have SSE broadcast for UI but no internal interception bus.
* **Plan:** add a typed hook bus in the `Agent` runtime: `BeforeTool(ctx, call) (allow|deny|approve, err)`, `AfterTool(ctx, call, result) (newResult, err)`, `BeforeRequest(ctx, messages)`, `AfterChunk(ctx, chunk)`. Implement: (a) approval prompts (11.11's `ask_user` is one consumer), (b) Claude-Code-style `hooks.json` shell commands, (c) per-tool timeouts (dsh splits `dsh-timeout`/capability termination/policy), (d) repeat-tool advisory reminders (dsh `repeat-tool-reminder`), (e) metrics/tracing. The 11.13 extension API is the external face of this same bus.
* **Effort:** M.

### 12.4 First-class persistent terminals (PTY) **[NEW]**
* **Reference:** dsh `terminal/` family — `ctx.terminals` backend registry; `terminal-bash` provides shell sessions with readiness detection, bounded state, and sandbox policy; `tool-terminal` exposes **six** model-facing tools for spawn/send/ctrl/await/log/kill, plus background-send integration. PTYs are owner-scoped (each agent owns its sessions) and complement one-shot bash for REPLs, dev servers, and interactive CLIs.
* **GoHarness today:** one-shot `execute_command` only.
* **Plan:** `github.com/creack/pty` backend behind the `Runner`/`Terminals` interface from 12.1; tools `terminal_start`, `terminal_send`, `terminal_poll` (with bounded reads and a deadline), `terminal_kill`. Reuse our workspace lock and sandbox modes. Document `tmux` as the non-PTY fallback for v1 (already in 11.22).
* **Effort:** M.

### 12.5 LSP as a capability seam, not a JSON-RPC tunnel **[NEW]**
* **Reference:** dsh `lsp/` — a Service Definition with exactly **four semantic operations** (`goToDefinition`, `findReferences`, `goToImplementation`, `hover`); a generic stdio provider that can run any language server; one model-facing `lsp` tool. Providers register **capabilities**, not raw methods; documents open transiently per query. There is deliberately no generic JSON-RPC escape hatch, so swapping providers can't change what the model sees.
* **GoHarness today:** none (pi-lens-style build feedback is 11.10).
* **Plan:** post-11.10, add the four-op LSP seam over stdio. The build/test feedback loop catches most errors; LSP adds navigation and hover for code-understanding tasks. Keep the surface intentionally tiny.
* **Effort:** M.

### 12.6 Tool-output spill to disk **[NEW]**
* **Reference:** dsh `spill/` — when a tool result exceeds a size threshold, the full output is persisted to a session-scoped file and the inline result is replaced with a bounded preview plus a retrieval locator; a `get_spill`/offset API lets the model page through it. This directly prevents a 500 KB `grep` or build log from blowing the context window — a problem we handle only by truncating to 400 chars, which loses information.
* **GoHarness today:** `truncateResult()` clips tool output to 400 chars for console display, but the *full* result is returned to the model.
* **Plan:** add a spill store under `.goharness/sessions/<id>/spill/`; cap inline tool results at e.g. 20 KB; beyond that, write full content to a content-addressed file and return a short header + `read_spill(id, offset, limit, find_text)` tool result (same shape as the web content cache in 11.1). Apply to command output, file reads, BM25 results, and web fetches uniformly.
* **Effort:** S.

### 12.7 Plan mode as logged, per-agent collaboration state **[NEW]**
* **Reference:** dsh `plan-mode` — plan mode is not a global toggle; it is per-agent collaboration state (the agent and user iterate a plan, edits are gated until the plan is accepted/rejected, with an exit tool). The plan itself is logged to the session.
* **GoHarness today:** we have visual workflows (DAG) for complex pipelines but no lightweight "plan, get approval, then execute" mode for linear chat.
* **Plan:** a plan mode where the model writes a structured plan to a scratch document, edits are blocked (tools return "plan not accepted"), the user reviews/edits, then accepts to unlock execution. Reuse the approval hook from 12.3. Pairs naturally with 11.12 (todo overlay) and named reviewer sub-agents (11.8).
* **Effort:** S-M.

### 12.8 Goals with autonomous round-driving **[NEW]**
* **Reference:** dsh `goal/` family — `ctx.goals` holds durable objective state as part of the session log; a `goal-round-driver` continues the agent across turns until the goal is met or blocked; `tool-goal`/`command-goal` expose it to model and user. This is "give the agent an objective and let it work across multiple rounds until done" — beyond our single ReAct loop.
* **Plan:** add a goal record (description, done criteria, status) to session state; a driver loop that, after each turn settles, asks the model "is the goal met? if not, continue" until completion or a step budget; surface progress in the UI. Requires the steering/queue (11.2) and cancellation (11.3) to be safe.
* **Effort:** M.

### 12.9 Dynamic workflows over sub-agents + "Ralph" **[IMPROVEMENT]**
* **Reference:** dsh `workflow/` — model-authored orchestration scripts run over the sub-agent registry in worker threads; `tool-workflow` exposes general execution; `tool-ralph` ships a fixed "fresh agent per task" workflow (spawns a clean sub-agent for each item, like a worker pool).
* **GoHarness today:** our DAG workflows are *human-authored* in the visual editor; sub-agents are *model-invoked* per turn. We don't have model-authored fan-out over many children.
* **Plan:** after durable sub-agent jobs (11.7), add a `run_workflow(spec)` tool where `spec` is a small declarative/scripted DAG the model writes (list of tasks, dependencies, which role/profile runs each), executed over the existing sub-agent engine with the per-profile throttle and workspace lock. "Ralph" is just a preset: N independent tasks → N fresh children → collect results.
* **Effort:** L.

### 12.10 Multiple sub-agent providers behind one interface **[IMPROVEMENT]**
* **Reference:** dsh subagent seam registers providers by name: `spawn-in-process` (local child), `fork` (OS process), `acp` (Agent Client Protocol), `codex` (delegate to OpenAI Codex), `claude-code` (delegate to Claude Code), `dsh-sdk` (embed). Each advertises **capabilities** (`outputSchema`, `depthLimit`, `toolFilter`, `persona`); a request needing a capability the provider lacks fails loudly at start — no silent degradation. There are also *continuable* children (durable background sessions you can resume and send more messages to) vs. one-shot children.
* **GoHarness today:** one in-process sub-agent kind.
* **Plan:** keep in-process as the default; behind a `SubagentProvider` interface add at least an `ACP` provider (open standard, lets us delegate to any ACP-compatible agent) and possibly a `claude-code`/`codex` CLI provider (shell out, capture transcript). Add a capabilities struct so the model knows what it can ask for. Continuable/resumable children are 11.7.
* **Effort:** M per provider; interface is S.

### 12.11 Structured sub-agent output (object-rooted JSON schema) **[IMPROVEMENT]**
* **Reference:** dsh `SubagentStartRequest.outputSchema` — a sub-agent can be asked to return a value matching a JSON Schema; providers that support it force a capture tool so the result is machine-parseable (`SubagentResult.structured`), with a strict enforced subset of JSON Schema.
* **GoHarness today:** our `expect` field asks for a return shape in prose; the result is always text.
* **Plan:** extend `spawn_sub_agent` with an optional `output_schema`; when set, inject a mandatory final `report_result(json)` tool the child must call, parse/validate against the schema, and return the structured value alongside the text. The parent can then route results programmatically (feeds 12.9 workflows).
* **Effort:** S.

### 12.12 Per-agent tool scoping & personas **[IMPROVEMENT]**
* **Reference:** dsh `SubagentStartRequest.toolFilter` and `persona` — a child's tools can be restricted by name with "visibility not authority" (the tool vanishes from the prompt AND refuses to execute; unknown names error loudly), and a per-child persona string shadows the deployment persona with strict `{{variable}}` interpolation. In-process scoping uses the `scope` primitive (12.14) so one registration context means both per-agent visibility and shared lifetime ownership.
* **GoHarness today:** sub-agents get all tools except write-by-default (which we've since enabled); named roles (11.8) will carry a tool allow-list.
* **Plan:** when we add named roles (11.8), make tool filtering *enforced* server-side (not just omitted from the prompt), and add a `persona`/system-prompt override per role.
* **Effort:** S (fold into 11.8).

### 12.13 Per-session agent presets & process-shared services **[NEW]**
* **Reference:** dsh `preset/` — a directory with an `agent.cordis.yml` mounts under an agent's scope, giving that session its own tools and prompt sections while other live sessions keep theirs; so one process runs multiple differently-composed agents at once. Presets that try to publish a process-global service are rejected at mount.
* **GoHarness today:** one global tool/prompt config per running server; all sessions share it.
* **Plan:** allow a session to select an "agent preset" (subset of tools + extra prompt sections + profile/model) from `.goharness/presets/<name>.yaml`; the `Agent` runtime already carries its own tool list and API config, so this is mostly config + UI. Useful for running a "research" agent vs a "coding" agent side by side.
* **Effort:** S-M.

### 12.14 Scope primitive (per-agent visibility + lifetime) **[NEW — internal]**
* **Reference:** dsh `core/scope` — one small library ties "this registration belongs to agent X" to both **visibility** (agent X sees its own tools/events shadowing globals) and **lifetime** (when agent X ends, all its registrations unwind together, with racing disposers awaiting the same quiescence). It's an opaque identity key (the live `Agent` object) plus a scoped registry layer with shadowing.
* **GoHarness today:** tools are global; cleanup is ad hoc.
* **Plan:** when we add the extension bus (12.3) and presets (12.13), use a small `Scope` type: each `Agent` carries one; registering a tool/hook/command on a scope auto-disposes when the agent ends. Prevents leaked MCP tools and hooks from sub-agents.
* **Effort:** M (internal; not user-visible).

### 12.15 Profiles & bundles as layered, patchable config **[NEW]**
* **Reference:** dsh boots a plugin tree from ordered layers: each bundle in the profile's list, then the profile's `cordis.patch.yml`, then the home-level patch, then any `--patch` overlay. A patch targets a row by id and replaces its config or inserts new rows, so every shipped default is overridable without forking. `dsh --profile web --dump-config` prints the resolved tree.
* **GoHarness today:** config is a flat JSON file; profiles (provider connections) exist but aren't a composition system.
* **Plan:** borrow the layering idea for config: ship defaults as the base layer, then project `.goharness/config.yml`, then user `~/.goharness/config.yml`, then CLI flags — with a declarative patch/merge (not just overwrite). Add a `--dump-config` debug command. Especially valuable once extensions (11.13) exist.
* **Effort:** M.

### 12.16 Approval policy & sandbox as services **[IMPROVEMENT]**
* **Reference:** dsh `ctx.approval` is a one-shot prompt seam; `ctx.sandbox` applies per-session confinement to process execution with modes, per-call policy, wrapped-argv dialects, and fail-closed errors. Sandbox and filesystem share one world, so policy moves with remote providers (12.1). We have path write-protection and sandbox *modes* (host/docker/linux landlock/macos sandbox-exec/windows AppContainer) but no per-call approval prompts and no single policy object threaded through tools.
* **Plan:** unify our existing sandbox backends behind a `Sandbox` interface on the execution world; add an approval hook (12.3) that can prompt per-tool-call (allow once / allow session / deny), persisted as a per-session decision. Project trust (11.15) gates loading project policy.
* **Effort:** M.

### 12.17 Hooks bridge (Claude Code / Codex style `hooks.json`) **[NEW]**
* **Reference:** dsh `hooks/` — a bridge plugin that reads an existing `hooks.json` (PreToolUse/PostToolUse, etc.) and dispatches to external shell commands on the matching dsh events, so users can bring their existing hook configs. Includes timeout-policy and repeat-reminder listeners.
* **GoHarness today:** none.
* **Plan:** once the hook bus exists (12.3), add a `hooks.json` reader that runs user shell commands on `BeforeTool`/`AfterTool`/`BeforeRequest`/`AfterRequest` with stdin JSON and treats exit code as allow/deny. This is a very high-value, low-cost compatibility feature for Claude Code users.
* **Effort:** S.

### 12.18 Session query with full-text search + model tool **[IMPROVEMENT]**
* **Reference:** dsh `session-query` family: `ctx.sessionQuery` defines trusted reads/relationship queries/search; `session-query-sqlite` implements it with **SQLite FTS**; `tool-session-query` exposes workspace-authorized queries to the model; `session-log-export` adds web `/export` (ZIP). This is cross-session *and* within-session semantic recall over the durable event log.
* **GoHarness today:** BM25 over workspace files and session-scoped search; no cross-session query, no FTS, no model-facing recall tool.
* **Plan:** overlaps with 11.9 (memory). After JSONL logs (11.6/12.2), add a SQLite FTS index over session events and a `session_query(query, scope, limit)` tool that respects workspace authorization. Export endpoint reuses it.
* **Effort:** M.

### 12.19 Content-addressed attachments **[NEW]**
* **Reference:** dsh `attachment/` — immutable, content-addressed binary references with image limits and a storage service; bytes only enter durable storage on submit or provider commit.
* **GoHarness today:** `uploads/` folder (from web paste) exists but is not content-addressed or quota-managed.
* **Plan:** when adding image support (11.19), store uploads as blobs keyed by hash, dedupe, enforce size/type limits, and reference by ID in messages rather than embedding base64.
* **Effort:** S (fold into 11.19).

### 12.20 Code Mode / model-written programs **[NEW]**
* **Reference:** dsh `code-runtime/` — a capability seam for executing a *model-written program* against host-provided async bindings, capturing what it printed and returned, with a worker-thread backend and generated SDK per language. Tools are tagged `mode: code`; the model gets a `run_code` tool plus an SDK in the runtime's language. This is sandboxed code execution (think e2b/Pyodide) rather than shell commands.
* **GoHarness today:** none.
* **Plan:** lower priority. Add behind the execution-world interface (12.1) as a provider — e.g. a WebAssembly sandbox (Wazero) or a containerized Python/JS runtime — with a generated small SDK. Lets the model do data-processing tasks without shelling out.
* **Effort:** L; defer.

### 12.21 Scheduled/mission runs **[NEW]**
* **Reference:** dsh `schedule/` package and pi-subagents' missions — timed and recurring runs with delivery receipts.
* **Plan:** a cron-like scheduler (in-process, persisted) that runs a named prompt/workflow on a schedule and delivers results (notification/webhook/chat). Useful for "check this CI every morning." Low priority but easy once jobs (11.7) exist.
* **Effort:** S-M; defer.

### 12.22 Guard / secret scanning **[NEW]**
* **Reference:** dsh `guard/` family (and pi-hermes-memory's secret scanning) — policy guards on output.
* **Plan:** a `BeforeResponse` hook that scans for obvious secret patterns (API keys in output) and warns/blocks; also scan before memory persistence (11.9).
* **Effort:** XS.

### 12.23 Human feedback that stays out of model context **[NEW]**
* **Reference:** dsh `feedback/` — two deliberately separate contracts: an immutable log-only remark (never enters model history) and an editable per-message rating/note sidecar. Telemetry observes remarks but feedback stays independent.
* **Plan:** add 👍/👎 + notes on assistant messages in the UI; persist to a sidecar, never inject into prompts. Useful for evals and product feedback without contaminating context.
* **Effort:** S.

### 12.24 Skills as on-demand capability packages **[IMPROVEMENT]**
* **Reference:** dsh `skill/` follows the [Agent Skills standard](https://agentskills.io) (same as Pi) — Markdown with frontmatter, loaded on demand by description match. Covered by 11.13; listed here for completeness because dsh confirms the same pattern.
* **Plan:** as 11.13.

### 12.25 Telemetry/observability contracts **[NEW]**
* **Reference:** dsh `packages/telemetry` (vendor-neutral contracts, reference adapter, conformance tests, typed schemas) plus OpenTelemetry adapter; Braintrust ships a pi extension for tracing. Every turn/tool/LLM call emits structured events derived from the session log.
* **Plan:** define a small telemetry interface (`OnTurn`, `OnTool`, `OnLLMCall`) fed from the hook bus (12.3); ship a logging adapter and an OTLP adapter behind a config flag. Our `LogExecutionTrace` is a start but not structured/exportable.
* **Effort:** S-M.

### 12.26 Headless / RPC / SDK modes **[IMPROVEMENT]**
* **Reference:** dsh ships `web` (browser app), `headless` (one-shot runner, no server), an SDK (`createAgentSession`), and RPC over stdio (strict LF-delimited JSONL). Pi has the same four modes.
* **GoHarness today:** web UI + OpenAI-compatible HTTP gateway; no one-shot CLI runner or stdio RPC.
* **Plan:** add `goharness run "prompt"` (headless, prints result, exits 0/1) and a `--mode rpc` JSONL stdio protocol for editor/CI integration. The `Agent` runtime is already embeddable; this is wiring.
* **Effort:** M.

### 12.27 Things we deliberately do *not* need from dsh
- **Cordis as a framework.** Its plugin composition is elegant but is a large TypeScript inversion-of-control system; a Go interface + hook bus (12.1/12.3) gives us 90% of the extensibility with a fraction of the complexity for a single-binary product.
- **Bun/Node runtime assumptions.**
- **"No privileged core" taken to the extreme** (model adapter as a plugin, etc.). We can keep our providers compiled in while still making IO and tools pluggable.

### Updated sequencing (merge with Phase 11 order)
The dsh analysis changes priorities in two ways:
- **12.3 (hook bus) and 12.6 (tool-output spill) should move up** — they're small, unblock many other items (approval, hooks, timeouts, memory, tracing), and spill fixes a real context-window bug independent of any big feature.
- **12.1 (execution-world interfaces) should precede remote sandboxes, PTY, and LSP** — do the interface once, then add providers.

Revised high-level order:
1. 11.1 web search/fetch · 12.6 tool-output spill · 12.22 secret guard (XS)
2. 11.2 steering + 11.3 cancel/abort
3. 12.3 hook bus (with approval + timeouts + 12.17 hooks.json bridge)
4. 11.4 `@file` · 11.5 `!command` · 11.16 context walking
5. 12.1 execution-world interfaces (`FS`/`Runner`/`Terminals`)
6. 11.10 build feedback · 12.4 PTY · 12.16 sandbox/approval policy
7. 11.8 named roles (with 12.12 enforced scoping) · 11.12 todo · 12.7 plan mode · 12.8 goals
8. 11.9 memory + 12.18 session query/FTS
9. 11.6/12.2 JSONL session log with model-visible invariant
10. 12.10 sub-agent providers + 12.11 structured output + 11.7 durable background jobs
11. 12.5 LSP seam · 11.18 thinking blocks · 11.19 images (with 12.19 attachments) · 11.11 ask-user
12. 11.14 model catalog · 11.15 trust · 11.20 palette · 11.21 slash commands · 12.13 presets · 12.15 layered config
13. 11.13 extension API (external face of 12.3/12.14)
14. 12.9 model-authored workflows · 12.20 code mode · 12.21 schedules · 12.25 telemetry · 12.26 headless/RPC

### 12.28 UI/UX: client plugin architecture and surfaces **[NEW — strategic for the web GUI]**

dsh's web frontend is as pluginized as its backend. The client (`packages/client/`) is a React app where every surface is a separate `ui-*` package that mounts into named **slots** (`ui-slots` provides a `register({name, children, store, inject, kind}, Component)` API with chain-routing, four-share prop typing, and store seats). UI packages do not own the transcript; they register keyed renderers against the session event log. This is a different philosophy from our single-bundle web UI, and several concrete ergonomic ideas transfer without adopting the whole framework.

**Reference packages** (all under `packages/client/modules/`): `ui-conversation`, `ui-tool`, `ui-trajectory`, `ui-input-trigger`, `ui-commands`, `ui-subagent`, `ui-plan`, `ui-goal`, `ui-jobs`, `ui-workflow-run`, `ui-user-questions`, `ui-attachment`, `ui-reference`, `ui-permission-presets`, `ui-message-feedback`, `ui-layout`, `ui-sidebar`, `ui-workspace`, `ui-settings`, `ui-settings-models`, `ui-settings-plugins`, `ui-theme`, `ui-deliverables`, `ui-brand-official`, `ui-primitives`, `ui-renderer`.

#### 12.28.1 Slot/region model for the web client **[IMPROVEMENT]**
* **Reference:** `ui-slots` — named regions (composer, header actions, sidebar sections, conversation view tabs, status bars, overlays) with `register()`; chain-kind slots self-nominate by a selector so the first match wins (e.g. one view renderer per conversation node type). Four prop shares compose: runtime, child slots, store, and injected business API.
* **GoHarness today:** our web UI is one HTML file with hardcoded regions; adding a button/panel means editing the core.
* **Plan:** introduce a small client-side registry (plain JS, no framework needed) — `ui.registerSlot('composer.footer', Component)` and `ui.registerChatNode('tool_call', renderer)` — even without a plugin loader this decouples features and makes 11.13's UI extensions possible. Render tool calls, subagent cards, plan chips, and jobs badges as independent node renderers rather than branches in one giant template.
* **Effort:** M.

#### 12.28.2 Conversation view: step-grouped flow + sticky composer **[IMPROVEMENT]**
* **Reference:** `ui-conversation` — chat is grouped by step (one model request + its tools), with a "step summary" row, streaming-tail isolation, and turn status. The composer is a **sticky dock**: a stats dock (token/cost/context) sits above the input, and queued steering messages render as rows above the textarea; the scrollport reserves its scrollbar gutter so opening/closing overlays never shifts content horizontally.
* **GoHarness today:** messages stream in; tool calls are collapsed `<details>`; no step grouping; composer is static.
* **Plan:** group a response and its tool calls under a step header ("● 3 tools · 1.2s · 4.2k tokens") with a collapse-all; make the composer sticky; add the live stats dock (11.17). Keep scroll position stable as new blocks arrive.
* **Effort:** M.

#### 12.28.3 Composer takeover for approvals and questions **[NEW]**
* **Reference:** both approvals and `ui-user-questions` replace the composer in place (not a floating modal) with an amber strip, the question/approval text, and refuse/allow or answer controls. While waiting, the normal input is hidden; the answer is delivered through the same `PendingWait` carrier and restores the composer. Pending interactions (approvals, plan reviews, questions) are also surfaced in the sidebar with an amber dot and "Waiting for approval/answer".
* **GoHarness today:** we have foldable tool results but no model-initiated question or per-call approval UI.
* **Plan:** when 11.11 (`ask_user`) or an approval hook (12.3/12.16) fires, swap the composer for an inline question/approval card with allow-once/allow-session/deny; reflect pending state in the session sidebar.
* **Effort:** M (depends on 11.11 and 12.3).

#### 12.28.4 Trajectory / inspector view **[NEW]**
* **Reference:** `ui-trajectory` — a second tab beside Chat rendering a **turn-aware event ledger** (User / Assistant / Tool / nested Subtool rows) with thick rules at turn boundaries, compact step markers, and a selection **inspector** showing token usage (input/output/cache), duration, and timing per event. It has a time-axis overview at the top, click-drag interval selection to focus events, wheel-to-zoom, and virtualized rendering (only visible rows + overscan mount); older pages load on scroll-up.
* **Why it matters:** for debugging long agent runs and for cost/latency work, a flat chat transcript is inadequate; this is the "developer tools" view of an agent.
* **Plan:** add a "Trajectory" tab to the session view. Render events from the JSONL log (11.6/12.2) as a virtualized ledger; clicking an event opens an inspector with tokens/duration/cache; add a timeline strip. Reuse our existing `cost_update`/trace events.
* **Effort:** L (mostly frontend); depends on the event log.

#### 12.28.5 `@` and `/` trigger system with grouped candidates **[IMPROVEMENT]**
* **Reference:** `ui-input-trigger` + `ui-commands` — under the caret, `/` and `@` open a grouped, fuzzy-matched candidate menu. Focus stays in the textarea (combobox pattern with `aria-activedescendant`); Enter/Space adjudication hooks let sources accept or refuse submission (e.g. a command that takes images only fires if images are attached). `/` does fuzzy subsequence matching (prefixes rank first) even though Space/Enter require an exact name. The menu is session-scoped and sources can be warmed and updated live.
* **GoHarness today:** we have toolbar buttons but no in-editor command palette; 11.4 plans `@file`.
* **Plan:** build one trigger component used by both `@file` (11.4), `@agent` (subagent addressing), and `/command` (11.21); group candidates by source; keep the caret in the input; support keyboard nav and fuzzy filter. The 11.20 palette reuses this.
* **Effort:** M.

#### 12.28.6 Tool call tree with nested sub-calls **[IMPROVEMENT]**
* **Reference:** `ui-tool` — a `ToolCallTree` renders one root call with recursive `subCalls`, selection state, and a per-call slot so each tool ships its own atomic view (bash gets a terminal view, subagent gets its own card, file edits get a diff). The runtime is authoritative for call/result pairing; UI packages register only the view for their wire name.
* **GoHarness today:** our tool results are one collapsed block per call; sub-agents report as one tool result. We don't show nested calls (a tool that invokes another tool, or a sub-agent's internal tool calls).
* **Plan:** model tool calls as a tree in the chat renderer; let sub-agents (11.7) expose their child calls; give each tool type its own renderer (diff for patch, terminal for command, card for subagent). Reuse for the trajectory view.
* **Effort:** M.

#### 12.28.7 Sub-agent navigation and fleet UI **[IMPROVEMENT]**
* **Reference:** `ui-subagent` — in a parent session, the header shows a breadcrumb (`parent / ▾ 3 descendants`) with a lazy-loaded tree catalog of child sessions (running indicator, per-child token usage and duration); subagent-origin sessions are hidden from the main sidebar and reached through the parent's catalog. A running continuable child keeps its composer enabled for follow-ups; a one-shot child shows a read-only transcript; Stop is always available. An `@` source lists running children for quick reference insertion.
* **GoHarness today:** we have a compact "Sub-agents (3)" progress card (Phase 1/2 work) but no transcript drill-down, no per-child stats, no follow-ups, and sub-agent sessions aren't navigable.
* **Plan:** after 11.7 (durable jobs), make the card expandable: click a child → open its transcript in a pane; show per-child tokens/duration; allow stop; later allow follow-up messages into a continuable child. Hide sub-agent sessions from the main sidebar.
* **Effort:** M (with 11.7).

#### 12.28.8 Plan chip, todo strip, goals, jobs badge in the composer **[NEW]**
* **Reference:** `ui-plan` renders a "Plan ×" chip in the composer's plan seat when plan mode is active and switches the placeholder; `ui-tool`/todo renders the active todo list as a strip above the input; `ui-goal` shows objective progress; `ui-jobs` adds a header badge listing background jobs owned by the session (running + stoppable counts).
* **Plan:** once 12.7 (plan mode), 11.12 (todo), 12.8 (goals), and 11.7 (jobs) exist, surface each as a small, persistent composer-region indicator rather than burying them in chat. One-line status, click to expand/focus.
* **Effort:** S each, once the backend exists.

#### 12.28.9 Workspace and session sidebar with grouped, searchable, sortable rows **[IMPROVEMENT]**
* **Reference:** `ui-workspace` + `ui-sidebar` — workspaces group sessions; each workspace remembers expanded/collapsed state; an open workspace shows 5 sessions with a "Show more" overflow; sessions can be sorted manually or by "Last updated"; an inline search expands across the header, does instant substring matches on titles/workspaces, and after 250 ms does a ranked **content** search with snippets (capped at 20 results, "narrow your query" prompt). Session rows show live status: **Waiting for approval/answer** (amber), **Running** (blue spinner), descendant-activity count, or unviewed completion. Rename, fork (auto-incremented title), archive, and delete are inline; drag-to-reorder is persisted; hovering a row copies its path/title.
* **GoHarness today:** we have a session sidebar but no grouping by workspace, no content search, no live pending/running status on rows, no sort modes.
* **Plan:** group sessions by workspace folder; add row status from the live agent state; add the 250 ms debounced content search backed by 12.18's FTS; add rename/fork/archive inline. Match Pi's "last updated" vs "manual" sort.
* **Effort:** M.

#### 12.28.10 Settings as plugin cards with live model testing **[IMPROVEMENT]**
* **Reference:** `ui-settings-models` — providers are cards with inline key entry (validated as printable ASCII, rejected if pasted as `NAME=value`), per-model context-window/output-cap fields with `K`/`M` suffixes, and a **"Fetch available models"** button that interrogates the *currently typed but unsaved* endpoint+key (one round trip instead of save-then-return). Results open a multi-select picker; providers that can't be interrogated stay hand-editable with their error shown inline. Each settings write carries a revision so concurrent edits from another tab or a file edit are rejected with `settings-conflict`. Custom providers are a separate create card requiring a unique id, endpoint, protocol, and ≥1 model. A first-run notice gates the DeepSeek onboarding until a provider is reachable.
* **GoHarness today:** our Providers modal edits profiles but has no "test" button, no live model fetch, no revision/conflict handling, no custom-protocol create flow.
* **Plan:** add a "Test & fetch models" action per provider (calls the provider's models endpoint with the draft creds, shows checkmarks/errors before saving); add context-window/output-cap editors; add optimistic concurrency via a settings revision; add a custom OpenAI/Anthropic/Gemini-compatible provider create card (11.14's UI).
* **Effort:** M.

#### 12.28.11 First-class theme system **[IMPROVEMENT]**
* **Reference:** `ui-theme` ships dark/light built around **CSS design tokens** (a `cssdesign` token catalog); third-party themes override same-named alias variables. The theme is a browser preference (not sent to the model); hot-reloads on edit.
* **GoHarness today:** one embedded stylesheet; no theming.
* **Plan:** factor our colors/spacing/typography into CSS custom properties on `:root` with a `[data-theme=light]` override; add a theme switcher; later allow user CSS or packaged themes. Pure frontend, no model impact.
* **Effort:** S-M.

#### 12.28.12 Message-level feedback, deliverables, and references **[NEW]**
* **Reference:** `ui-message-feedback` adds 👍/👎 + optional note per assistant message in a **storage sidecar** that never enters model context (12.23); `ui-deliverables` tracks files/artifacts the agent produced; `ui-reference` renders `@file`/`@folder`/`@session` mentions as colored inline chips in both the transcript and the composer, with the actual reference text preserved for editing (so a mention survives a remount as canonical parseable text rather than a display-only bubble).
* **Plan:** add per-message rating (sidecar, not model context); surface files written in a turn as a "deliverables" group; render `@` mentions as chips in sent and received messages using the same serialization as the composer.
* **Effort:** S-M.

#### 12.28.13 Drag-and-drop, image attachments, and drop overlay **[NEW]**
* **Reference:** `ui-attachment` — composer images get a rail of thumbnails with file names/sizes; a full-viewport drop overlay shows while dragging files over the page; images open in a fit-to-viewport lightbox (Escape/mask/close-button dismiss, focus restored). Limits are enforced client-side.
* **GoHarness today:** no drag-drop or image paste (11.19 covers the model side).
* **Plan:** with 11.19, add paste/drag listeners, a thumbnail rail, the drop overlay, and a lightbox; enforce size/type limits; store via the content-addressed attachment service (12.19).
* **Effort:** M.

#### 12.28.14 Empty/hero state and block reasons **[IMPROVEMENT]**
* **Reference:** with no workspace selected the whole composer card is the workspace-picker trigger (textarea read-only, keyboard accessible); when sending is blocked the composer shows the reason as placeholder text ("Select a model first", "Plan awaiting review", "No workspace selected") with the one missing action kept live. Blocks are *affordances* explaining why and what to do.
* **GoHarness today:** the composer is always editable; errors are toasts.
* **Plan:** make the composer reflect state: disable + reason when no workspace/model/profile is selected; make the whole card a CTA for the missing prerequisite.
* **Effort:** S.

#### 12.28.15 Reliability details worth copying
Small things dsh does that add up:
- **Scrollbar gutter always reserved** so content doesn't horizontally shift when a scrollbar appears/disappears.
- **Virtualized long lists** (trajectory, search results, session list) with overscan and stable keys.
- **Streaming tail isolation** — new chunks don't re-render the whole transcript; the streaming element is keyed separately.
- **Debounced search (250 ms)** with abort of the previous in-flight request.
- **Stable React/DOM identity for the shell** across session switches (workspace picker, scroll body, composer seat are never unmounted) so focus/scroll/IME state survives navigation.
- **Session-scoped draft mirror** — the composer draft persists across workspace/session switches.
- **Hover-to-reveal** compact actions (checkpoint disclosures, row buttons) rather than always-visible clutter.
- **Accessibility:** visible focus rings, `aria-activedescendant` on comboboxes, keyboard shortcuts for everything (Escape to cancel/close, arrow keys in menus/tree).
- **Toast/notice surface** for non-blocking errors that keep the draft (e.g. "images not supported on this command").
- **Localized EN/中文** pairs throughout; even without translating, structure strings for it.

### Where GoHarness is already ahead
- **Visual DAG workflow editor** — dsh has model-authored *text* workflows (`tool-workflow`/Ralph) but no graphical canvas.
- **Per-profile HTTP concurrency throttling in core** — dsh relies on provider/tool policy; our `max_concurrency` is a first-class profile field.
- **Environment-aware shell instructions** in the system prompt — we already detect OS/shell and inject guidance; dsh leaves this to personas.
- **Single static binary** vs dsh's Node/pnpm runtime — easier to install and air-gap.

### UI sequencing (fold into Phase 11/12 order)
1. **12.28.14** empty/block states + **12.28.11** theme tokens (XS–S polish, low risk).
2. **12.28.5** unified `@`/`/` trigger (unblocks 11.4, 11.20, 11.21).
3. **12.28.2** step-grouped conversation + sticky composer + stats dock.
4. **12.28.6** tool-call tree renderer (enables richer subagent/files views).
5. **12.28.9** workspace/session sidebar with status + FTS (with 12.18).
6. **12.28.3** composer-takeover approval/questions (with 11.11, 12.3).
7. **12.28.7** subagent fleet drill-down (with 11.7).
8. **12.28.10** settings model testing + conflict handling.
9. **12.28.4** trajectory/inspector view (with JSONL log).
10. **12.28.1** slot registry (after enough surfaces exist to justify it; supports 11.13).
11. **12.28.8/12/13** plan/todo/goal/deliverables/feedback as those backends land.

### 12.29 Shell layout, panel geometry, tabs, and visibility **[IMPROVEMENT — web GUI]**

dsh's client has a deliberately small, well-reasoned layout system rather than ad-hoc CSS. Sources: `packages/client/modules/ui-layout`, `ui-sidebar`, `ui-settings`, `ui-conversation`, `ui-primitives`, `ui-renderer`, `ui-deliverables`, `ui-slots`, `ui-workspace`, `ui-brand-official`.

#### 12.29.1 Three-column shell with a concession chain **[IMPROVEMENT]**
* **Reference:** `ui-layout` ships an **AppFrame** with three columns registered as slots — `sidebar`, `conversation`, `details` — plus `conversation.empty`. Drag handles sit between them.
* **Concession rules (the important part):** when the window narrows, **only the details panel shrinks**, and once it hits its minimum it **auto-closes**; the sidebar and conversation never shrink. This is called the "concession chain" and it means the user's primary surface is never squeezed.
* **Closed states are not zero:** a closed sidebar collapses to a **56 px icon rail** (brand mark + 36 px icon buttons at 10 px inset); a closed details panel is **0 px wide**. So "hide" means two different things depending on the surface.
* **Geometry is transient:** the sidebar width and details-open state are **not persisted to localStorage**; reload restores defaults, and switching to a different Session id closes details. Returning to the same session restores its width; unselected surfaces render details at zero width without touching the stored preference. The last non-blank session id is retained across blank states.
* **No scroll anchoring during squeeze** is a documented limitation.
* **GoHarness today:** our web UI is essentially a single conversation pane + a sidebar with no resizing/concession model and no details surface.
* **Plan:** adopt a three-column frame: sidebar (resizable, collapses to an icon rail), conversation (never shrinks), details (resizable, auto-closes under pressure). Persist widths across reloads (we don't share dsh's reset-on-reload stance). Details panel becomes the home for 12.28.4 (trajectory inspector), 12.28.7 (subagent fleet), per-message inspector, and file preview.
* **Effort:** M (frontend).

#### 12.29.2 Slot ownership of chrome, not just content **[NEW]**
* **Reference:** every panel's *chrome* is declared as named slots so plugins can replace pieces without touching layout:
  - `sidebar.brand.mark`, `sidebar.brand.name` (independent slots; collapsed rail renders just the mark; `ui-brand-official` overrides both without replacing the New Session button).
  - `sidebar.workspaces` (the scroll region), `sidebar.settings` (bottom-pinned).
  - `conversation.session.header.lineage`, `…header.actions`, `…header.utilities` (three independent rows in the session header — lineage/breadcrumbs on the left, action buttons center, utilities right; removing an occupant restores titles without affecting actions).
  - `conversation.composer` (the whole input seat; approvals/questions take it over), `conversation.hero.workspace`, `conversation.view` (the tab ring), `conversation.input.overlay` (the `@`/`/` menu), `conversation.chat.turnTail` (deliverables row), `conversation.empty`.
  - `settings.trigger`, `settings.header`, `settings.close`, `settings.action`, `settings.section`, `settings.plugins.tab`, `settings.onboarding`.
  - `root` (the whole app).
* **Chain slots** self-nominate by a selector (first non-null wins) — e.g. the active conversation view.
* **GoHarness plan:** as part of 12.28.1, define this slot catalog up front; even before external plugins exist, internal features (trajectory, subagents, deliverables, approvals) mount through slots so the DOM structure stabilizes.
* **Effort:** M (pays off across every later UI item).

#### 12.29.3 Conversation view tabs (not pages) **[NEW]**
* **Reference:** the conversation column has a **view ring** — a tab strip where each entry owns its chrome. Chat is the package's own entry; **Trajectory** (12.28.4) contributes another tab; plugins contribute more. The active view renders through a slot with `only: <id>`. The shell (scroll body, composer, header) stays mounted; only the view body swaps. Tabs carry `id/order/label`; session-scoped.
* **Plan:** add a tab strip to our conversation header. Chat first, Trajectory second (once 12.28.4 lands), later subagent transcript, files, etc. Keep composer/scrollport identity stable across tab switches.
* **Effort:** S once slots exist.

#### 12.29.4 Sidebar anatomy and collapse motion **[IMPROVEMENT]**
* **Reference:** sidebar shell (not the workspace list) owns: brand row, **New Session** action, layout collapse toggle, scroll region, bottom-pinned **Settings** seat. `ui-workspace` fills the scroll region; it does not own the chrome.
* **Collapse animation:** expanded content fades out at its current width for 150 ms; upper controls (toggle, New Session, add, search) share a 150 ms fade + 49 px leftward translation, ending on the rail's 10 px inset as the column slides at 300 ms; each 36 px control follows the same path. The bottom-pinned Settings control fades but does **not** translate. `prefers-reduced-motion` disables transitions. A page that starts collapsed renders the rail statically.
* **Scrollbars are a pointer affordance:** the column binds the scrollbar thumb to `transparent` when the pointer is outside and keeps it drawn for **2 seconds after the pointer leaves**; the region reserves the gutter so showing/hiding the thumb never reflows content.
* **GoHarness today:** our sidebar has no collapse-to-rail, no animation, native scrollbars.
* **Plan:** implement icon-rail collapse with the staged animation; reserve scrollbar gutter; auto-hide thumbs.
* **Effort:** S-M.

#### 12.29.5 Settings as a sidebar surface, not a modal **[IMPROVEMENT]**
* **Reference:** settings has **no presentation of its own** — it's a slot system: a base package provides the schema/transport/scope; `ui-settings-general` provides the shell (navigation + chrome) which mounts into `sidebar.settings` (so it lives at the sidebar's foot, not a modal). Features register sections (one page each), a Plugins tab for plugin pages, and ordered onboarding pages. Settings are schema-bound and **revisioned**: each write carries an expected revision, so a concurrent edit from another tab or a file edit is rejected with `settings-conflict`. Remote/non-loopback browsers get **no durable settings** (RPC is loopback-only); their rows render inert.
* **GoHarness today:** Settings is a modal; no conflict detection; no sections/plugins model.
* **Plan:** relocate Settings into the sidebar (or a slide-over panel) with section registration; add optimistic concurrency; keep secrets out of the client. This also gives 12.28.10 (model testing) a home.
* **Effort:** M.

#### 12.29.6 Composer regions and in-place takeover **[NEW]**
* **Reference:** the composer is a stack, not one textarea:
  - **Stats dock** (sticky above the input) — token/cache/cost/context readouts (11.17).
  - **Input docks** — queued steering/follow-up messages render as rows above the textarea (11.2).
  - **Todo/plan strip** — the active todo list/plan state (11.12/12.7).
  - **Bar** — the textarea plus its access row (permission preset chip, model picker, send).
  - **Overlay** — the `@`/`/` candidate menu.
* **Approvals and questions replace the composer in place** (an amber strip with the question + allow/refuse controls) rather than opening a modal; pending waits leave no placeholder card in the message flow. A block reason (no workspace, no model, plan review pending) renders the same disabled textarea with the reason as its placeholder, and leaves exactly one action live (the thing that unblocks it).
* **Plan:** when building 11.2/11.11/12.3/12.7, mount them into these composer regions; never as floating cards in chat.
* **Effort:** M (architectural).

#### 12.29.7 Message flow: steps, streaming isolation, compaction placement **[IMPROVEMENT]**
* **Reference:** chat is grouped by **step** (one model request + its tools), with a step-summary row, streaming-tail isolation, and turn status. **Streaming tail isolation** means new chunks re-render only the trailing element; everything above is frozen as cached React nodes keyed stably, so a long reply doesn't re-parse the whole transcript.
* **Compaction renders as one collapsed row *at the checkpoint's flow position***, not a replacement of the transcript above it; it shows replaced-item count and estimated tokens, and discloses the summary on click/hover. Manual `/compact` starts as a running row and folds into the checkpoint when it settles.
* Non-user messages (context injections, cross-session recalls) render as a **default-collapsed disclosure** naming the role and the producer (so "a skill catalog" reads differently from "a workspace instruction file").
* **Turn tail:** the deliverables row (12.28.12) and icon actions render between the message body and its footer, owned by slots.
* **Plan:** group our transcript by step; virtualize the settled history; stream only the tail; render compaction inline as a collapsible checkpoint rather than a system message; fold injected context by default.
* **Effort:** M.

#### 12.29.8 Layer/z-index contract **[NEW]**
* **Reference (from `ui-primitives`):** layers are explicit and ordered:
  1. App content (slots, sidebar, conversation, details).
  2. `HoverCard` — portaled, with a pointer-leave grace so it survives the anchor gap.
  3. `Toast` — top banner, 120 px from viewport top, **`pointer-events: none`**, centered over the composer anchor (re-measured on resize); `role="alert"`; 3 s hold + 1 s fade; keyed by sequence so repeats re-animate.
  4. Attachment **drop overlay** — full viewport, pointer-inert (owner decides accept/drop).
  5. Attachment **lightbox** — body-portaled, fit-to-viewport, Escape/mask/close dismiss, restores focus.
  6. **Modal / Onboarding** — body-portaled, makes `#root` inert for its lifetime, owns focus trap.
  The toast explicitly layers *above* the lightbox so an upload failure over a preview stays visible.
* **Plan:** codify our z-index scale as CSS custom properties with the same ordering; never hardcode `z-9999`; make toasts pointer-events-none; trap focus in modals.
* **Effort:** XS-S.

#### 12.29.9 Primitives that own specific content shapes **[NEW]**
* **Reference:** `ui-primitives` ships typed renderers rather than generic markdown:
  - **TerminalBlock** — parses ANSI with `anser` (SGR colors, cursor movements incl. CR/backspace/erase-in-line, tab stops, CJK width), replays spinner redraws (`100%\rOK` shows `OK`, `\x1b[K` erases), keeps `white-space: pre`, horizontal scroll, collapses head+tail beyond 16 lines behind an expand button, one running/done/error state dot in a reserved gutter.
  - **DiffBlock** — path header, removed-then-added grouping, `⋯` hunk gap, `└ +A -R · N file(s)` footer, copy control writes prefixed diff text, same head/tail collapse.
  - **ReadBlock** — line-numbered syntax-highlighted file window, "showing N of M" note, head/tail collapse.
  - **SearchBlock / WebBlock** — source list with status and capped retrieval, empty-state note, compact fetch summary.
  - **JsonBlock / JsonTree** — read-only inspector for structured tool output.
  - **MarkdownText** — GFM + KaTeX math, raw HTML omitted, relative/non-HTTP links neutralized, safe external-link attributes, **incremental streaming parser** that freezes all but the last two blocks as cached React elements, wide tables scroll horizontally inside a focusable wrapper, inline code that is an absolute URL becomes a link, and a `fileMentions` resolver turns real file tokens into openers.
  - **StateDot** with four states (done/warning/ongoing/error), **DisclosureRow**, **Button/Pill/Menu/Modal/Input**, **useAnchoredPosition/useAnchoredMaxHeight** for floating panels.
* **GoHarness today:** tool output is one collapsed `<pre>`; no ANSI, no diff rendering, no line numbers, no streaming-optimized markdown.
* **Plan:** build these atoms one at a time as we touch each tool: TerminalBlock for `execute_command`, DiffBlock for `patch_file`, ReadBlock for `read_file`, SearchBlock for `bm25_search`/web search; swap the generic markdown renderer for an incremental one.
* **Effort:** M (highest ROI of the UI polish items).

#### 12.29.10 Persistent shell identity across session switches **[IMPROVEMENT]**
* **Reference:** the conversation shell (workspace picker, scroll body, composer seat, textarea) **survives no-session and session transitions** with stable React/DOM identity. Strict-session **header and body** are separate outlets that fill their regions when the first session arrives. Blank sessions render the same composer body as active sessions, and the **InputHub carries drafts across workspace switches** and mirrors them into the session store. This means focus, scroll position, IME composition, and the textarea caret don't reset when you switch sessions.
* **GoHarness today:** switching sessions re-renders the whole chat and loses draft/focus/scroll.
* **Plan:** hoist shell state above the session outlet; keep the composer mounted; mirror drafts per-session.
* **Effort:** M.

#### 12.29.11 Workspace/session browser details worth copying **[IMPROVEMENT]**
* Already covered in 12.28.9, but specific geometry rules: an **open workspace shows 5 sessions** with a transient **Show more**; returning to 5 requires closing and reopening; a new session created from a group opens the group so it stays visible; **Manual** vs **Last updated** sort (last-updated does a full recency sort once, then prompts promote their session once; manual preserves order); drag order is host-durable for real workspaces and browser-local for Ungrouped/flat list; workspace **search** expands across the header, outside-click collapses an empty query except while the slide gesture is in flight, clear button always resets; content search debounces 250 ms, aborts the previous request, caps at 20; session rows show **pending interaction** (Waiting for approval / Plan awaiting review / Waiting for answer) with an amber dot that **outranks** the running indicator; descendant subagent activity outranks the unviewed-completion reminder; subagent-origin sessions are **hidden from the sidebar** and entered through the parent's catalog.
* **Plan:** fold these rules into 12.28.9's implementation.

#### 12.29.12 Branding and theming mechanics **[IMPROVEMENT]**
* **Reference:** a **theme presenter** consumes resolved `ctx.theme` snapshots and writes them to the document: `color-scheme` on `<html>` (native UA chrome — scrollbars, form controls), `body[data-ds-dark-theme]`, the theme's alias tokens as inline CSS variables on `<body>`, and a `<meta name="theme-color">` matching the computed background. It **measures after palette/token application** so the rendered background is the single color authority. Third-party themes are an extension point that overrides alias variables; missing values deliberately fall back to the nearest semantic token. The default shell label is "DSH Local Build" with a 7-char commit badge; deployments override via slots.
* **Plan:** same approach for 12.28.11 — tokens on body, color-scheme on html, meta theme-color; measure after apply; ship a light theme as an override sheet.
* **Effort:** S.

#### 12.29.13 Boot/rendering lifecycle **[NEW]**
* **Reference:** `ui-renderer` owns the React root. The server renders a **framework-free boot page**; after every client plugin activates, it calls `ctx.uiRenderer.mount(container)`. The renderer installs slot outlets, session providers, and observable-to-hooks bindings, hydrates the existing boot DOM, and **switches to the assembled application before the next paint**. React/ReactDOM/Cordis/slots/primitives keep one identity through the static module table. Per-region readiness is deferred (the first frame waits for all entries, but no per-slot Suspense yet).
* **Plan:** for our single binary, serve a lightweight loading shell, stream the JS bundle, and hydrate against the live DOM rather than replacing it; defer non-critical panels. Faster perceived startup, especially on remote/SSH launches.
* **Effort:** M (lower priority than layout/primitives).

#### Recommended UI order (revised)
1. **12.28.14 block/empty states** + **12.29.8 z-index contract** + **12.29.12 theme tokens** (XS–S, set the foundation).
2. **12.29.2 slot catalog** + **12.29.1 three-column shell** + **12.29.10 persistent shell identity** (the structural M).
3. **12.28.5 `@`/`/` trigger** in the **12.29.6 composer regions** + **12.28.2 step grouping / sticky stats dock**.
4. **12.29.9 typed blocks** (Terminal/Diff/Read/Search) + **12.29.7 streaming-tail isolation + inline compaction rows**.
5. **12.29.4 sidebar rail/collapse** + **12.28.9 workspace/session browser** with statuses.
6. **12.29.3 conversation tabs** + **12.28.4 trajectory inspector**.
7. **12.29.5 settings surface** + **12.28.10 live model testing**.
8. **12.28.3 composer takeover** for approvals/questions (with 11.11, 12.3).
9. **12.28.7 subagent fleet** in the details panel.
10. **12.29.13 boot/hydrate polish**.
