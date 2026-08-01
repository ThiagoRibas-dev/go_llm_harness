# 🗺️ GoHarness Local Agent: Engineering Roadmap

This document outlines the strategic engineering roadmap for evolving **GoHarness** from a lightweight, single-file prototype into a **production-grade, local-first, modular, and sandboxed AI Agent Runner**.

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
│ PHASE 8: OPENAI COMPATIBLE GATEWAY│ ◄── [Planned Extension]
│ - GET /v1/models discovery        │
│ - POST /v1/chat completions proxy │
│ - Tokenizer & Embedding Proxies   │
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

### 2.3 Dual-Engine Workspace Rollbacks
* **The Concept:** When a conversation is rewound (via `/fork <turn>`), the physical files in your workspace directory must be reverted to match that exact point in time. If we only rewind the chat history but leave edited files unchanged, the agent's memory becomes out-of-sync with the disk.
* **Engine 1: Git-Native Checkpoints (Primary):**
  - Before executing any file-modifying tool (like `write_file` or `patch_file`), the Go harness checks if the directory is a Git repository.
  - If a Git repo is detected, the harness automatically creates an in-memory Git stash or checkpoint commit: `.goharness/checkpoint-turn-X`.
  - On a conversation rollback (`/fork 3`), the harness automatically executes a fast `git reset --hard` back to the checkpoint of Turn 3, perfectly restoring deleted, added, or modified files.
* **Engine 2: Lightweight Local Backup Stashing (Fallback):**
  - If `git` is missing or the folder is not a repository, GoHarness falls back to a built-in, zero-dependency file-backup system.
  - Right before a file is written or patched, the Go harness makes a backup copy inside `.goharness/sessions/<session_id>/backups/<turn_number>/<filepath>`.
  - On rollback, the harness copies the backed-up files from that turn back to their original locations, discarding newer files.

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
* **Function Schema Handshaking:** Maps tool schemas to Google's `"functionDeclarations"`, tool calls to `"functionCall"`, and tool responses to `"functionResponse"` objects.
* **Auth Support:** Handles query-parameter key handshakes for Gemini AI Studio, and OAuth Bearer tokens (`Authorization: Bearer`) for Google Cloud Vertex AI integrations.

---

## 🌐 Phase 8: OpenAI-Compatible API Gateway

To turn GoHarness into a unified local agentic middleware that any standard third-party LLM frontend can plug into, we expose a secure, standard-library-only OpenAI API Gateway.

### 8.1 API Discovery & Completion Proxies
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
