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
