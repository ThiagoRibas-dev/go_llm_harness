# 🔬 GoHarness: Research Bibliography & Inspiration Ledger

This document acts as an engineering ledger and bibliography recording the open-source projects, architectures, and philosophies researched and referenced during the creation of **GoHarness**. GoHarness stands on the shoulders of these incredible developer platforms, borrowing and refactoring their breakthrough designs into a single, zero-dependency, $6\text{ MB}$ compiled Go module.

---

## 🧬 The 9 Core Agentic Research Vectors

To design a production-grade, local-first agent runner, we must formalize our systems engineering across **9 distinct research vectors**. These categories define the full technical landscape of modern AI Agent Harnesses and Middleware Platforms:

### 1. 🎨 UI/UX & Live Streaming Interfaces
* **The Challenge:** Providing a rich, responsive console interface that monitors background agent thoughts and tool logs without terminal locking, output truncation, or rendering lags.
* **GoHarness Solution:** Serves an embedded Single-Page Application (HTML5, Tailwind, JS) and uses **Server-Sent Events (SSE)** to stream LLM tokens, cost updates, and directory tree changes in real-time. It features a responsive workspace swapper, conversation history selector, and parallel timeline branching dashboard.

### 2. 🔄 Agentic Loops & Autonomous Workflows
* **The Challenge:** Managing autonomous multi-turn reasoning loops. The loop must cleanly handle tool calls, feed stdout/stderr back into context, check for loop-termination states, and handle self-debugging on compiler errors.
* **GoHarness Solution:** Implements a strict, configurable Turn Loop Manager that safely cuts off at `max_turns` (preventing infinite loop API spend), and automatically feeds back raw trace outcomes to let the model self-correct on failures.

### 3. ⚙️ Automation & Local File Manipulation
* **The Challenge:** Manipulating host-system repositories, editing code, and creating folders without bloating the context window or corrupting files.
* **GoHarness Solution:** Implements **Semantic Diff Patching (`patch_file`)**, allowing the model to make precise search-and-replace string substitutions in-memory, rather than re-writing thousand-line files, saving up to $90\%$ of completions token costs.

### 4. 🔌 Tool Execution & Dynamic Routing
* **The Challenge:** Decoupling tools from the core orchestrator, and dynamically routing executions depending on who owns the capability (native functions vs. external subprocesses).
* **GoHarness Solution:** Features a global routing map (`mcpToolsMap`). It parses tool requests and seamlessly forwards them natively to host sandboxes or encodes them as JSON-RPC payloads destined for local MCP child servers.

### 5. 🧠 Memory & State Management
* **The Challenge:** Keeping permanent, multi-session histories across months of work, and managing session rewinds without letting the disk file structure drift out-of-sync with the agent's memory.
* **GoHarness Solution:** Employs **Turn-by-Turn Plain-Text JSON Serialization** (storing turns as sequential files on disk) combined with a **Dual-Engine Workspace Rollback** (Git resets or local backup stashes) to physically restore your folder's files when branching, alongside a **Sliding-Window Context Compactor** that preserves the last $N$ turns fully raw.

### 6. 🔒 Security, Sandbox Isolation & System Protection Shields
* **The Challenge:** Protecting the host system from malicious commands, data deletion, or systemic escapes, while preventing a rogue LLM from overriding its own log history.
* **GoHarness Solution:** Enforces bare-metal kernel-level isolation (Linux Landlock LSM, macOS Apple SBPL `sandbox-exec` profiles, and Windows Job Objects & Low-Integrity SIDs) to restrict execution strictly to `./workspace/`, combined with an in-process **Write-Protection Shield** that blocks any edits to `.goharness` or configuration directories.

### 7. 🔌 Multi-Provider Portability & Payload Translation
* **The Challenge:** Translating your agent's universal conversation state dynamically to match the completely incompatible, non-OpenAI REST payload structures of alternative cloud providers.
* **GoHarness Solution:** Implements **Modular Data Decoupling** inside `src/llm.go`. It parses our universal message history and compiles it fresh on demand, handling Anthropic's top-level system strings and Gemini's strict model/user binary roles and function declarations.

### 8. 🔌 Extensibility Standards (Model Context Protocol)
* **The Challenge:** Allowing the agent's toolset to be infinitely extendable by users without requiring code modification or recompiling the binary.
* **GoHarness Solution:** Acts as a complete **MCP Stdio Client**. It programmatically spawns local tool servers as child processes, conducts standard JSON-RPC 2.0 handshakes, and dynamically imports their schemas, making SQLite, Git, or Brave Search immediately usable on Turn 1.

### 9. 🪙 Telemetry, Auditing, & Observability
* **The Challenge:** Recording a structured, time-stamped, and audit-compliant log of the agent's internal thought latency, tool execution runtimes, and financial costs.
* **GoHarness Solution:** Writes append-only structured logs to **`.goharness/traces.jsonl`** using a thread-safe Go write loop, enabling offline performance diagnostic analysis.

---

## 📚 Refereed Projects & Mapped Inspirations

### 1. 🦙 Llama.cpp & KoboldCPP (`ggerganov/llama.cpp` & `LostRuins/koboldcpp`)
* **Core Research Vector:** *UI/UX (Phase 6) & OpenAI-Compatible API Gateway (Phase 8)*
* **GoHarness Implementation:** Exposing an in-process, zero-dependency, OpenAI-Compatible API Gateway directly from Go's native web server. This includes:
  - `GET /v1/models` (handshake discovery).
  - `POST /v1/chat/completions` (agentic proxy completion loop).
  - `POST /v1/embeddings` (RAG vector proxy).
  - `/v1/tokenize` & `/v1/detokenize` (in-process BPE-approximate round-trip tokenizer).

### 2. 🧠 OptMem (`VictorTaelin/OptMem`)
* **Core Research Vector:** *Memory & State Management (Phases 2 & 9)*
* **GoHarness Implementation:**
  - Decoupling conversation state from heavy databases into **100% human-readable turn-by-turn plain-text JSON files** for exceptional observability.
  - **$O(1)$ Index-Based Range Loader:** Targeting and loading history files directly by sequence range rather than $O(N)$ folder scanning.
  - **Hierarchical Memory Tree Decay (Phase 9):** Collapsing old conversation blocks into multi-level, logarithmic summary nodes while preserving recent turns in raw verbatim detail.

### 3. 🦀 Claude Code (Rust Edition) (`lorryjovens-hub/claude-code-rust`)
* **Core Research Vector:** *Automation (Phase 2) & Bare-Metal Sandboxing (Phase 4)*
* **GoHarness Implementation:**
  - **Semantic Diff Patching (`patch_file`):** Making fast, exact-match string replacements (`strings.Replace`) instead of rewriting entire files. Saves up to $90\%$ of token costs.
  - **Bare-Metal Native Sandboxing:** Platform-specific secure execution cages (Linux Landlock, macOS SBPL, Windows restricted tokens & Job Objects).
  - **Incremental File-Watcher Sync:** Integrating standard `os.Stat` modification time checks (`ModTime()`) directly into the Auto-LS tree scanner.
  - **`CLAUDE.md` Compatibility (Phase 8.6):** Support for reading and injecting standard Claude Code repository instructions dynamically on boot.

### 4. 🎛️ SillyTavern (`Cohee1207/SillyTavern`)
* **Core Research Vector:** *UI/UX (Phase 6) & Multi-Provider Portability (Phase 7)*
* **GoHarness Implementation:**
  - Redesigning our Web settings panel into a **dynamic, reactive modal** that morphs and reveals/hides fields based on your selected provider (e.g. revealing GCP Project ID and Region *only* when Google Vertex is selected).
  - Integrating support for advanced sampling parameters: **Temperature**, **Top-P**, **Top-K**, and **Google Reasoning/Thinking Levels**.

### 5. 🥧 Pi Agent Harness (`earendil-works/pi`)
* **Core Research Vector:** *Agentic Loops & State Management (Phase 2)*
* **GoHarness Implementation:**
  - **Conversational Time-Travel:** Reverting conversation history back in time using a `/fork <turn>` command.
  - **Context Compaction:** Consolidating long history logs into single-message summary states to protect LLM context windows.

### 🦞 6. OpenClaw (`openclaw/openclaw`)
* **Core Research Vector:** *UI/UX & Live Streaming Interfaces (Phase 6)*
* **GoHarness Implementation:**
  - Embedding entire web assets (HTML5, Tailwind, vanilla JS Single-Page App) directly inside the statically compiled executable using `//go:embed web/*`, serving a robust Web GUI Console on Port 8080.

### 🧬 7. Hermes Metaharness (`howdymary/hermes-agent-metaharness`)
* **Core Research Vector:** *Telemetry, Auditing & Observability (Phase 5.3)*
* **GoHarness Implementation:**
  - Writing thread-safe, append-only structured trace logs into **`.goharness/traces.jsonl`** to track LLM latencies, tool execution durations, exit codes, and operational metadata.

---

## 💻 IDE Copilots & Autonomous Extensions (VS Code Integration)

### 8. 💻 Continue Autopilot (`continuedev/continue`)
* **Core Research Vector:** *Automation & Workspace Context Provision (Phase 1)*
* **GoHarness Translation:** Our automatic, dual-scoped instruction loaders (which scan both active project workspaces for `CLAUDE.md` and global binary roots on boot) and our token-safe Auto-LS directory tree walks represent an automated, zero-configuration **Local Context Provider**. It provides the LLM with instant repository-wide awareness on turn 1 without requiring manual user tags.

### 9. 🤖 Cline / Prev-Devins (`cline/cline`)
* **Core Research Vector:** *UI/UX & Secure Agentic Control (Phases 4 & 6)*
* **GoHarness Translation:** Cline proved that developers want absolute control over their local agent's actions. GoHarness implements this by merging an **interactive TUI shell** and a **Web Console** with live streamed SSE tool logs, allowing developers to monitor outputs, configure safety guardrails, blacklist command patterns (e.g. `rm -rf`), and execute timeline rollbacks/branching in a single click.

### 10. 🌐 Zoo Code & OpenCode (`Zoo-Code-Org/Zoo-Code` & `anomalyco/opencode`)
* **Core Research Vector:** *Multi-Provider Portability & Extensibility Standards (Phases 7 & 8)*
* **GoHarness Translation:** By exposing a local, un-sandboxed OpenAI API Gateway (Phase 8), GoHarness decouples its core sandboxed loop capabilities from its native Web UI. It allows you to plug its secure, sandboxed multi-turn capabilities directly into self-hosted, open-source IDE extensions or collaborative git agents without any corporate cloud lock-in.

---

## 📄 License
This document is licensed under the MIT License.
