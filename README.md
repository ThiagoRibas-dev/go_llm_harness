# 🤖 GoHarness: Local-First Secure AI Agent Runner & Node Workflow Engine

GoHarness is a high-performance, **local-first and secure AI Agent Runner** written in pure, standard Go with **zero external dependencies**. It bridges local developer workspaces, bare-metal sandbox engines, dynamic Model Context Protocol (MCP) servers, and a **Directed Acyclic Graph (DAG) workflow runtime**, serving a gorgeous, real-time streamed Web Console directly from the compiled binary.

While GoHarness is designed as an elite, high-speed companion for **systems-engineering and coding tasks**, it is built from the ground up as a **universal, multi-purpose cognitive shell**. Its underlying architecture is fully optimized for **long-form creative writing, deep multi-document research, and infinite-memory conversational chatting**.

> **🆕 v2.0 — Node-Based Workflow Orchestration.** GoHarness 2.0 evolves from a hardcoded linear agent loop into a fully generalizable, concurrent **DAG Execution Runtime** (`src/workflow.go`). You can now architect arbitrary reasoning pipelines — linear steps, fan-out/fan-in parallel branches, tool meshes, and conditional routing — as plain JSON in `workflows.json`, design them visually in the browser, or compile them from natural language in the **AI Workflow Lab**. Ship with two defaults: **Standard Linear Chat** and **Enhanced Cognition (POADR)**, a 5-axis parallel reasoner.

---

## 🎨 System Architecture & Flow

![GoHarness Architecture](docs/assets/harness_architecture.svg)

At a high level, every request flows through one of two engines:

* **Linear Agent Loop** — the classic, tool-capable ReAct loop (read/write/patch files, run commands, BM25 search, spawn sub-agents, call MCP tools) used by the `linear_chat` workflow.
* **DAG Workflow Engine** — a concurrent topological executor that schedules independent nodes in parallel goroutines, synchronizes inputs over channels, and enforces a hard timeout. Used by `enhanced_cognition` and any custom workflow you define.

---

## 🚀 Standout Features (Zero-Bloat Engineering)

GoHarness is engineered to be as **lightweight and secure** as possible, avoiding the dependency fatigue of heavy Python or Node-based agent frameworks:

1. **🕸️ Node-Based DAG Workflow Engine (v2.0):** A pure-Go, standard-library concurrent DAG runtime. Independent nodes execute in parallel goroutines; edges carry typed data between ports; every branch is wrapped in a `context.WithTimeout` so a single hanging node can never bottleneck a response.
2. **🧠 Enhanced Cognition (POADR):** The flagship default workflow decomposes a query across **5 parallel cognitive axes** — ⏳ Chronological, ⛓️ Causal-Logical, 🗺️ Semantic-World, 🧠 Behavioral-Psychological, 🎭 Stylistic-Prose — then synthesizes the reports. This eliminates representational interference in smaller models and is designed to maximize prompt-cache prefix sharing across the parallel branches. See the research in [`docs/COGNITIVE_AXES_ANALYSIS.md`](docs/COGNITIVE_AXES_ANALYSIS.md).
3. **🧪 AI Workflow Lab:** Describe a pipeline in plain English and compile it to a valid `workflows.json` schema; review a live topological node graph alongside an editable JSON codeboard; validate for cycles, mismatched ports, and missing anchors; then hot-swap it into the running engine with one click.
4. **🔒 Bare-Metal Native Sandboxing:** Enforces security at the operating-system kernel level, with no Docker or VM required:
   * **Linux:** **Landlock LSM** (kernel 5.13+) to isolate filesystem access without root/sudo.
   * **macOS:** Dynamic **Sandbox Profile Language (SBPL)** profiles via Apple's native `sandbox-exec`.
   * **Windows:** Duplicated process security tokens, **Low-Integrity SIDs (`S-1-16-4096`)**, and CPU/memory-capped **Job Objects** (same model as Chrome's renderer sandbox).
5. **🛡️ System Write-Protection Shields:** Detects and blocks any LLM attempt (even under malicious prompt injection) to overwrite, read, or manipulate system files, `.goharness/` logs, or `.git/` databases.
6. **🔌 Model Context Protocol (MCP) Client:** Implements Anthropic's open-standard MCP. It spawns local tool servers (SQLite, GitHub API, Search, etc.) as child processes, performs the full JSON-RPC-over-stdio handshake, and dynamically registers/routes their tools to the LLM on turn 1.
7. **🧠 Sliding-Window Context Compaction:** Automatically compresses history when user-turn count crosses a threshold, using a cheap/fast model while **preserving the last $N$ turns fully raw**. Compacted turns are physically evicted into named sibling folders (`compacted_summary_up_to_turn_%03d/`) with the boundary tracked in `meta.json`, and rollbacks crossing a boundary are archive-aware.
8. **🔄 Dual-Engine Workspace Rollbacks & Untracked Erasers:** Supports chronological session rollbacks and branching. Going back in time rewinds chat logs and **physically restores your folder structure** (file contents via per-turn backups, plus physical deletion of newly-created untracked files) using Git-native checkpoints where available or local backups as a fallback.
9. **🌲 Token-Safe Directory Tree (Auto-LS):** Recursively maps your workspace, collapsing heavy dependency folders (`node_modules`, `.venv`) and truncating long outputs, while appending Git-like status flags and relative modification timers inline (e.g. `main.go [Modified 2m ago]`).
10. **🌐 Embedded Web Console:** Go's native `net/http` plus `//go:embed` serve a modern Single-Page Application (HTML5, Tailwind, vanilla JS) and stream thoughts/tool logs to your browser over **Server-Sent Events (SSE)**. Includes a responsive **Workspace Swapper**, **Conversational History Selector**, **Session Deletion**, and the **AI Workflow Lab**.
11. **🔌 Multi-Provider AI API Connectors:** Native, standard-library-only wrappers for:
    * **OpenAI API / compatible routers** (Ollama, DeepSeek, Groq, etc.).
    * **Anthropic Claude Messages API** (system instructions as top-level params, `tool_use`/`tool_result` mapping).
    * **Google Gemini AI Studio & Vertex AI REST APIs** (`"user"`/`"model"` roles, function schemas, OAuth bearer headers, reasoning/thinking levels).
12. **🔌 OpenAI-Compatible API Gateway & Tokenizer Proxies:** Expose a standard gateway so any existing chat frontend (OpenWebUI, SillyTavern, LibreChat) plugs straight into GoHarness:
    * **`GET /v1/models` & `POST /v1/chat/completions`** — masquerades as a single smart model and runs the sandboxed loop in the background.
    * **`/v1/embeddings`** — vector embeddings proxy for RAG pipelines.
    * **`/v1/tokenize` & `/v1/detokenize`** — fast local, self-learning BPE approximate tokenizer with round-trip accuracy.
13. **📂 Dual-Scoped Instruction Injection & Claude Code Compatibility:** On boot, scans both the active workspace (project-scoped) and the binary directory (global-scoped) for `AGENTS.md`, `SKILLS.md`, `INSTRUCTIONS.md`, and **`CLAUDE.md`**, giving GoHarness out-of-the-box compatibility with existing Claude Code repositories. Pinned files are remembered per session in `meta.json`.
14. **🔍 BM25 Lexical Memory Engine:** A custom, zero-dependency BM25 search engine ranks files in the workspace, session logs, and uploads — avoiding semantic traps and saving up to ~95% of context tokens versus brute-force stuffing. Supports `target_scan_dirs` to index only designated subfolders.
15. **🪙 Persistence & Deep Observability:** Remembers and **auto-resumes your last active session** on launch. In debug mode, every subsystem writes a high-verbosity, thread-safe, append-only log to `.goharness/debug.log`, and an in-app **Inspect Execution Metrics** panel streams latency, tokens in/out, and spend per turn over SSE.

---

## 🕸️ The v2.0 Workflow Engine

Workflows are declared in **`workflows.json`** next to the binary. Each workflow is a DAG of typed nodes connected by directed edges that route named output ports to named input ports. Every graph is bounded by two anchors:

* **`start`** (`user_input`) — captures the raw user prompt.
* **`terminal`** (`assistant_response`) — emits the final answer to the Web Console and persists it to the session.

### Node Types

| Node Type | Purpose | Key Properties | Inputs → Outputs |
| :-- | :-- | :-- | :-- |
| `user_input` | Start anchor. Captures the prompt. | — | → `prompt` |
| `llm_query` | Single-purpose LLM call. | `provider`, `model`, `temperature`, `system_prompt` | `prompt` → `response` |
| `llm_synthesis` | Aggregator LLM call (merges many inputs). | same as `llm_query` | many named contexts → `response` |
| `tool_execution` | Runs a native sandboxed tool. | `tool_name` | `arguments` (JSON) → `stdout`, `exit_code` |
| `bm25_search` | Queries the local BM25 engine. | `scope`, `limit` | `query` → `search_results` |
| `conditional_router` | Branches on an evaluated condition. | `condition` (e.g. `on_error`) | `eval_var` → `route_branch` |
| `assistant_response` | Terminal anchor. Streams and saves output. | — | `final_output` → |

Per-node `provider`/`model` settings are hot-swapped into the global API config for the duration of that node's execution and restored immediately on return, so a single graph can mix providers and models freely (e.g. cheap `gpt-4o-mini` specialists feeding a `claude-3-5-sonnet` aggregator).

### Default Workflows

1. **`linear_chat`** — `start → llm_query → terminal`. The standard conversational agent loop with full tool access. Active by default.
2. **`enhanced_cognition`** (POADR) — `start` fans out to **five parallel `llm_query` specialists** (chronological, causal-logical, semantic-world, behavioral-psych, stylistic-prose); all five reports plus the raw prompt feed an `llm_synthesis` aggregator, which produces the final answer at `terminal`.

The active workflow is selected by the top-level `"active_workflow"` key. Setting it to anything other than `linear_chat` routes execution through `ExecuteActiveWorkflow()` in `src/workflow.go`; if the workflow engine errors, the system transparently falls back to the native linear tools loop.

### Designing Workflows

* **By hand** — edit `workflows.json` directly (the schema is documented in [`docs/V2_SPECIFICATION.md`](docs/V2_SPECIFICATION.md)).
* **In the browser** — open **Settings → AI Workflow Lab (v2.0)**. Edit JSON on the right and watch the topological graph re-render on the left, or type a plain-English description and click **Compile with AI** to generate a draft. The lab validates for cycles, port mismatches, and missing anchors; **Compile & Apply** writes to disk and hot-swaps the running engine.
* The LLM-assisted compiler prompt and staging lifecycle are specified in [`docs/V2_LLM_ASSISTED_WORKFLOW_SPEC.md`](docs/V2_LLM_ASSISTED_WORKFLOW_SPEC.md).

> **Prompt-cache note:** to maximize prefix caching across parallel LLM nodes, keep shared system/context instructions identical at the top of each node's prompt and append the axis-specific instruction at the bottom.

---

## 📁 Repository Directory Structure

```
.
├── .git/               # Git version-control database
├── .gitignore          # Version-control exclusions
├── README.md           # This documentation guide
├── config.example.json # Public version-controlled configuration template
├── workflows.json      # v2.0 DAG workflow index (linear_chat + enhanced_cognition)
├── go.mod              # Go module descriptor (standard library only; no go.sum deps)
│
├── src/                # All Go source files & embedded web assets
│   ├── main.go         # CLI shell, flag parser, linear agent loop, tool dispatch
│   ├── workflow.go     # v2.0 concurrent DAG workflow executor & node runners
│   ├── config.go       # Configuration structures, session meta, load/save helpers
│   ├── agent.go        # History persistence, compaction, rollbacks, BM25, sub-agents
│   ├── bm25.go         # Zero-dependency BM25 lexical ranking engine
│   ├── mcp.go          # Model Context Protocol client (JSON-RPC 2.0 over stdio)
│   ├── llm.go          # OpenAI, Anthropic, Gemini, Vertex API translation wrappers
│   ├── web.go          # Embedded HTTP server, SSE router, OpenAI-compatible gateway
│   ├── telemetry.go    # Thread-safe traces.jsonl + .goharness/debug.log writers
│   ├── embed.go        # Portable runtime extraction helpers
│   │
│   ├── sandbox.go            # Unified bare-metal sandbox router
│   ├── sandbox_linux.go      # Linux Landlock LSM executor
│   ├── sandbox_darwin.go     # macOS Apple sandbox-exec SBPL executor
│   ├── sandbox_windows.go    # Windows Job Object & restricted low-integrity token spawner
│   ├── sandbox_fallback.go   # Fallback executor for other unmapped OSes
│   │
│   └── web/
│       ├── index.html    # Responsive SPA (chat, settings, AI Workflow Lab, fork modal)
│       └── tailwind.min.js
│
└── docs/
    ├── V2_SPECIFICATION.md               # DAG engine, node types, schema, UI spec
    ├── V2_LLM_ASSISTED_WORKFLOW_SPEC.md  # AI Workflow Lab & staged-commit compiler
    ├── COGNITIVE_AXES_ANALYSIS.md        # The 5 cognitive axes & model-scale mapping
    ├── COGNITIVE_THEORIES_BIBLIOGRAPHY.md# Cross-disciplinary cognitive-science refs
    ├── COGNITIVE_LITMUS_TEST.md          # 5-axis audit of coding/writing/D&D/chat
    ├── ORCHESTRATOR_COMPARATIVE_RESEARCH.md  # LangFlow, LlamaIndex, Windmill, Temporal
    ├── BM25_SCALING_RESEARCH.md          # BM25/memory scaling research
    ├── COMPARISON_MATRIX.md              # GoHarness vs. state-of-the-art
    ├── RESEARCH.md                       # Research bibliography & inspirations
    ├── ROADMAP.md                        # Multi-phase engineering plan
    ├── GIT_HISTORY.md                    # Development ledger
    └── assets/                           # harness_architecture.svg, scale_comparison.svg
```

> Binaries are produced under `bin/` on build but are intentionally git-ignored. Cross-compile targets are listed below.

---

## 🚀 Getting Started

### 1. Build the Binary

Compile for your current machine:

```bash
go build -o bin/agent ./src
```

Cross-compile for all targets (pure Go, no CGO required):

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/agent.exe ./src

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/agent_mac_arm64 ./src

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/agent_mac_amd64 ./src

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/agent_linux ./src
```

### 2. Configure Your Environment

On first run, GoHarness generates a default **`config.json`** next to the executable (see `config.example.json` for the template):

1. Open `config.json`.
2. Add your API credentials and set `"api.provider"` (`openai`, `anthropic`, `gemini`, or `vertex`).
3. For a local model, point `"base_url"` at your router (e.g. `http://localhost:11434/v1` for Ollama).
4. (Optional) Review `workflows.json` — it ships with `linear_chat` active. Switch `"active_workflow"` to `"enhanced_cognition"` to enable parallel 5-axis reasoning.

### 3. Run the App

**Option A — Web GUI (recommended):** launches the HTTP server, exposes the API gateway, and opens your browser:

```bash
./bin/agent_linux -web
# with verbose developer logs:
./bin/agent_linux -web -debug
```

**Option B — Terminal UI (TUI):** runs the agent loop directly in your terminal:

```bash
./bin/agent_linux
# Type '/fork <turn>' to rollback/branch, or 'exit' to quit.
```

Useful flags: `-web`, `-port`, `-provider`, `-api-key`, `-url`, `-model`, `-sandbox`, `-container`, `-debug`.

---

## 🔌 Model Context Protocol (MCP) Server Registration

GoHarness is a native MCP client: it spawns configured tool servers as child processes, performs the JSON-RPC handshake over `stdio`, and appends their discovered tools to the agent's schema. Register servers via the **Web Console (Settings → MCP Servers)** for instant hot-reload, or statically in `config.json`:

```json
"mcp_servers": {
  "sqlite-demo": {
    "command": "uvx",
    "args": ["mcp-server-sqlite", "--db-path", "./workspace/dev.db"]
  },
  "web-search": {
    "command": "npx",
    "args": ["-y", "web-search-mcp"],
    "env": { "BRAVE_API_KEY": "your-brave-key-here" }
  }
}
```

Common templates: npm packages use `npx` + `["-y", "<pkg>"]`; uv packages use `uvx` + `["<pkg>"]`; local scripts use `node`/`python` + a script path. MCP servers run as host child processes, so ensure the relevant runtime is on your `PATH`.

---

## 🧠 Advanced Memory Subsystem (BM25, Uploads & Target Directories)

Inspired by progressive memory layering, GoHarness maximizes recall while minimizing token waste:

* **🔍 BM25 lexical search** — pure-Go keyword-relevance ranking across workspace files and session logs; invoke via the `bm25_search` tool.
* **📤 Custom uploads** — upload text/JSON/code documents (up to 10 MB) to the active session via the Files sidebar; they are priority-indexed.
* **📂 `target_scan_dirs`** — restrict indexing to designated subdirectories (e.g. `["src", "docs"]`) to skip build artifacts and dependencies.

---

## 🔄 Conversational Branching (Edit/Fork & Reroll)

* **✏️ Edit & Fork** — hover any message card, edit the text, and save. GoHarness clones the session up to that turn, applies your edit, and re-runs the loop on a new parallel timeline.
* **🔄 Reroll** — delete the last assistant/tool turns, revert workspace edits from that turn, and resubmit.
* **🌿 `/fork <turn>`** (TUI) or the **Rewind or Branch Timeline** modal (Web) — choose between cloning into an isolated branch (original untouched) or permanently truncating future turns.

---

## 🛡️ Sandboxing & Safety

| Platform | Mechanism |
| :-- | :-- |
| 🐧 Linux | **Landlock LSM** filesystem locks (kernel 5.13+, no root) |
| 🍎 macOS | **SBPL** profiles via `sandbox-exec` |
| 🪟 Windows | Low-integrity restricted tokens + **Job Objects** |

**Windows fallback (`sandbox_fallback`):** standard non-elevated Windows accounts can't assign tokens, so by default GoHarness blocks the command (`false`, strict). Set `sandbox_fallback: true` to fall back to unsandboxed execution with an in-console warning. For zero-compromise isolation on Windows, prefer **Docker** (`"sandbox_mode": "docker"` + a `"docker_container"`), **WSL2** (native Landlock), or an elevated Administrator shell.

---

## ⚙️ Configuration Reference (`config.json`)

| Block | Parameter | Description |
| :-- | :-- | :-- |
| **`api`** | `provider` | `openai`, `anthropic`, `gemini`, or `vertex`. |
| | `key` | API secret key (or GCP access token for Vertex). |
| | `base_url` | Completions endpoint override (defaults to the provider's). |
| | `model` | Target model (e.g. `gpt-4o`, `claude-3-5-sonnet-latest`). |
| | `temperature` | Sampling temperature (default `0.0`). |
| | `max_tokens` | Output token ceiling. |
| | `top_p`, `top_k` | Sampling thresholds. |
| | `thinking_level` | Gemini reasoning: `off`/`low`/`medium`/`high`. |
| | `project_id`, `region` | GCP coordinates for Vertex AI. |
| **`agent`** | `workspace_dir` | Directory the agent may read/write/edit. |
| | `target_scan_dirs` | Subdirectories to index for BM25 (e.g. `["src","docs"]`). |
| | `workspaces_history` | Registered workspaces (managed by the swapper). |
| | `last_active_session_id` | Auto-resumed session on launch. |
| | `max_turns` | Loop cutoff safety fuse. |
| | `command_timeout_seconds` | Hanging-script termination limit. |
| **`security`** | `sandbox_mode` | `host`, `docker`, or `none`. |
| | `sandbox_fallback` | Graceful unsandboxed fallback if sandbox init fails (default `false`). |
| | `docker_container` | Target container when `sandbox_mode` is `docker`. |
| | `allowed_tools`, `blocked_patterns` | Tool allowlist and blacklisted command substrings. |
| **`directory_scan`** | `max_depth`, `max_files_per_directory` | Tree-walk bounds. |
| | `ignored_patterns`, `collapsed_patterns` | Folders to skip or collapse in the auto-LS tree. |
| **`compaction`** | `provider`/`key`/`base_url`/`model`/`temperature` | Dedicated, cheap model for summarization. |
| | `auto_compact_turns` | User-turn count that triggers compaction. |
| | `keep_last_n` | Recent turns preserved fully uncompacted. |
| | `system_prompt` | Compaction synthesis instructions. |
| **`mcp_servers`** | `command`, `args`, `env` | MCP child-process definitions. |
| **`web`** | `enabled` | Auto-start the HTTP server on boot. |
| | `port` | Web GUI + gateway port (default `8080`). |
| | `api_gateway_enabled` | Toggle the OpenAI-compatible gateway. |
| **(root)** | `debug` | Enable high-verbosity `.goharness/debug.log` output. |

> Workflow topology lives in the separate **`workflows.json`** file, not in `config.json`.

---

## 🪙 Observability: Telemetry & Debug Logging

* **`.goharness/traces.jsonl`** — structured, append-only JSON entries for every LLM completion and tool call (timestamp, session, turn, action, duration, status, metadata).
* **`.goharness/debug.log`** — when `-debug`/`"debug": true` is set, a mutex-guarded writer records high-verbosity subsystem events (turn boundaries, API dispatch, tool paths/args, compaction triggers, workflow node lifecycle). Tail it live with `tail -f .goharness/debug.log`.
* **In-app metrics** — each assistant card exposes a collapsible **🔬 Inspect Execution Metrics** panel showing latency, tokens in/out, and USD spent, streamed in real time over SSE.

---

## 📚 Documentation Index

| Document | Topic |
| :-- | :-- |
| [`V2_SPECIFICATION.md`](docs/V2_SPECIFICATION.md) | DAG engine architecture, `workflows.json` schema, node types, concurrency model, and visual editor design. |
| [`V2_LLM_ASSISTED_WORKFLOW_SPEC.md`](docs/V2_LLM_ASSISTED_WORKFLOW_SPEC.md) | AI Workflow Lab, cookbook compiler prompt, staged-commit verification screen. |
| [`COGNITIVE_AXES_ANALYSIS.md`](docs/COGNITIVE_AXES_ANALYSIS.md) | The 5 cognitive axes and how model scale (width/depth/attention) governs cognitive load. |
| [`COGNITIVE_THEORIES_BIBLIOGRAPHY.md`](docs/COGNITIVE_THEORIES_BIBLIOGRAPHY.md) | Mapping the axes to Baddeley, Kahneman, Kant/Kosslyn, Dunbar, Chomsky/Fodor. |
| [`COGNITIVE_LITMUS_TEST.md`](docs/COGNITIVE_LITMUS_TEST.md) | Decomposing coding, story-writing, D&D, and casual chat across the 5 axes. |
| [`ORCHESTRATOR_COMPARATIVE_RESEARCH.md`](docs/ORCHESTRATOR_COMPARATIVE_RESEARCH.md) | Code-level lessons from LangFlow, Flowise, LlamaIndex Workflows, Windmill, Temporal. |
| [`BM25_SCALING_RESEARCH.md`](docs/BM25_SCALING_RESEARCH.md) | BM25 and memory-scaling research. |
| [`COMPARISON_MATRIX.md`](docs/COMPARISON_MATRIX.md) | GoHarness vs. state-of-the-art agent frameworks. |
| [`RESEARCH.md`](docs/RESEARCH.md) | Bibliography and project inspirations. |
| [`ROADMAP.md`](docs/ROADMAP.md) | Multi-phase engineering plan. |

---

## 🤝 Attribution & Development

This repository, along with its cross-platform bare-metal sandboxes, embedded web console, and v2.0 DAG workflow engine, was co-engineered, refactored, and compiled from scratch by **Arena.ai's Agent Mode**.

Arena.ai's Agent Mode orchestrates multiple state-of-the-art LLM engines (including, but not limited to, Claude, ChatGPT, Gemini, Grok, Qwen, and Kimi) to provide developers with fully autonomous systems-engineering, software packaging, and coding capabilities right from their workspaces.

## 📄 License

This project is open-source and licensed under the MIT License.
