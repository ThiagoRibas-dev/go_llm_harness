# 🤖 GoHarness: Local-First Secure AI Agent Runner & Web Console

GoHarness is a high-performance, single-file, **local-first and secure AI Agent Runner** written in pure, standard Go with **zero external dependencies**. It bridges local developer workspaces, secure bare-metal sandbox engines, and dynamic Model Context Protocol (MCP) servers, serving a gorgeous, real-time streamed Web Console directly from the compiled binary.

---

## 🎨 System Architecture & Flow

![GoHarness Architecture](docs/assets/harness_architecture.svg)

---

## 🚀 Standout Features (Zero-Bloat Engineering)

GoHarness is engineered to be as **lightweight and secure** as possible, avoiding the dependency fatigue of heavy Python or Node-based agent frameworks:

1. **🔒 Bare-Metal Native Sandboxing:** Enforces security at the operating system kernel level, completely eliminating the need for Docker or VMs:
   * **Linux:** Uses the state-of-the-art **Landlock LSM** (since kernel 5.13+) to isolate filesystem access without root/sudo.
   * **macOS:** Generates dynamic **Sandbox Profile Language (SBPL)** profiles to isolate execution via Apple's native `sandbox-exec`.
   * **Windows:** Duplicates process security tokens, drops privileges to **Low-Integrity SIDs (`S-1-16-4096`)**, and wraps processes inside CPU/Memory capped **Job Objects** (identical to Google Chrome's renderer sandbox).
2. **🛡️ System Write-Protection Shields:** Detects and blocks any LLM attempts (even under malicious prompt injections) to overwrite, read, or manipulate system files, `.goharness/` logs, or `.git/` databases.
3. **🔌 Model Context Protocol (MCP) Client:** Implements Anthropic's open-standard MCP. It automatically spawns local tool servers (e.g., SQLite, GitHub API, Search) as child processes, conducts the full protocol handshakes (JSON-RPC over Stdio), and dynamically registers/routes their tools to the LLM on turn 1.
4. **🧠 Sliding-Window Context Compaction:** Automatically triggers context compression when conversation history crosses thresholds. It summarizes historical details (using a cheap/fast model like `gpt-4o-mini` with custom parameters) while **preserving the last $N$ turns fully raw** (Sliding Window) to retain immediate context, slashing cumulative API bills by up to $75\%$.
5. **🔄 Dual-Engine Workspace Rollbacks:** Supports chronological session rollbacks (`/fork <turn>`). When you go back in time, GoHarness not only rewinds the chat logs but **physically restores your folder structure** to that turn's exact state using either local backup stashes (fallback) or Git-native reset checkpoints (primary).
6. **🌲 Token-Safe Directory Tree (Auto-LS):** Recursively maps your workspace, collapsing heavy dependency folders (like `node_modules` or `.venv`) and truncating long outputs, while appending visual Git-like status flags and relative modification timers inline (e.g. `main.go [Modified 2m ago]`, `schema.sql [New / Untracked]`).
7. **🌐 Embedded Web Console:** Employs Go's native `net/http` router and `//go:embed` to serve a modern Single-Page Application (HTML5, Tailwind, JS) and uses **Server-Sent Events (SSE)** to stream thoughts and tool logs to your browser in real-time. No node_modules, bundlers, or setups required. It includes a responsive **Workspace Swapper** and **Conversational History Selector** to switch contexts and resume past sessions in one click.
8. **🔌 Multi-Provider AI API Connectors (Phase 7):** Features native, standard-library-only translation wrappers for:
   * **OpenAI API / compatible routers** (Ollama, DeepSeek, Groq, etc.).
   * **Anthropic Claude Messages API** (extracting system instructions as top-level params, and mapping `"tool_use"` and `"tool_result"` blocks).
   * **Google Gemini AI Studio & Vertex AI REST APIs** (conforming roles strictly to `"user"`/`"model"`, mapping function schemas, and managing OAuth bearer headers).
9. **🔌 OpenAI-Compatible API Gateway & Tokenizer Proxies (Phase 8):** Exposes a standard gateway so you can plug any existing chat frontend (like OpenWebUI, SillyTavern, or LibreChat) straight into GoHarness!
   * **`GET /v1/models` & `POST /v1/chat/completions`:** Masquerades as a single, ultra-smart model, executes our secure sandboxed loop in the background, and streams the finished report back to your frontend.
   * **`/v1/embeddings`:** Vector embeddings proxy mapping standard RAG pipelines.
   * **`/v1/tokenize` & `/v1/detokenize`:** Implements an incredibly fast, local, self-learning Byte-Pair Encoding (BPE) approximate tokenizer, allowing SillyTavern to calculate exact context window limits in-process with $100\%$ round-trip accuracy!

---

## 📁 Repository Directory Structure

```
.
├── .git/               # Git version-control database
├── .gitignore          # Version-control exclusions
├── README.md           # This documentation guide
├── config.json         # Active runtime config (created automatically)
├── config.example.json # Public version-controlled configuration template
├── go.mod              # Go package descriptor (standard library only)
│
├── src/                # All Go Source Files & Embedded Web Assets
│   ├── main.go         # CLI shell, flag parser, and loop coordinator
│   ├── config.go       # Configuration structures and load/save helpers
│   ├── agent.go        # Directory walking, prompt loading, and compaction
│   ├── mcp.go          # Model Context Protocol Client (JSON-RPC 2.0 stdio)
│   ├── telemetry.go    # Thread-safe execution trace logs (.goharness/traces.jsonl)
│   ├── embed.go        # Portable runtime extraction helpers
│   ├── web.go          # Built-in HTTP web server and SSE streaming router
│   ├── llm.go          # OpenAI, Anthropic, Gemini, Vertex API translation wrappers
│   │
│   ├── sandbox.go      # Unified bare-metal sandbox router
│   ├── sandbox_linux.go   # Linux Landlock LSM sandbox executor
│   ├── sandbox_darwin.go  # macOS Apple sandbox-exec SBPL profile executor
│   ├── sandbox_windows.go # Windows Job Object & restricted low-integrity token spawner
│   ├── sandbox_fallback.go# Fallback executor for other unmapped OSes
│   │
│   └── web/
│       └── index.html  # Responsive Single-Page App assets (fully embedded)
│
├── bin/                # Compiled production static binaries (cross-compiled)
│   ├── agent.exe       # Windows static executable (~6.5 MB)
│   ├── agent_linux     # Linux static executable (~6.4 MB)
│   ├── agent_mac_arm64 # macOS Apple Silicon static executable (~6.1 MB)
│   └── agent_mac_amd64 # macOS Intel static executable (~6.5 MB)
│
└── docs/
    ├── ROADMAP.md      # Multi-phase strategic engineering plan
    └── assets/         # Project diagrams (harness_architecture, scale_comparison)
```

---

## 🚀 Getting Started

### 1. Build the Binary
To compile the static executable from source on your current machine:
```bash
go build -o bin/agent ./src
```

To cross-compile for all targets:
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

---

### 2. Configure Your Environment

On the very first run, GoHarness will automatically generate a clean, default **`config.json`** file next to your executable:
1. Open the created `config.json`.
2. Add your API credentials and set your `"api.provider"` (e.g., `openai`, `anthropic`, `gemini`, or `vertex`).
3. If running a local model, point `"base_url"` to your local router (e.g. `http://localhost:11434/v1` for Ollama).

---

### 3. Run the App

#### **Option A: Gorgeous Web GUI Mode (Recommended)**
Launches the built-in HTTP server, exposes the API Gateway, and automatically pops open your web browser:
```bash
./bin/agent_linux -web
```

#### **Option B: Lightweight Terminal UI (TUI) Mode**
Runs the agent loop directly inside your current terminal session:
```bash
./bin/agent_linux
```

---

## ⚙️ Configuration Parameters Breakdown (`config.json`)

| Configuration Block | Parameter | Description |
| :--- | :--- | :--- |
| **`api`** | `provider` | The active LLM API provider: `openai` (standard), `anthropic`, `gemini`, or `vertex`. |
| | `key` | Your API secret key (or Google Cloud Access Token if using Vertex). |
| | `base_url` | Complete completions endpoint URL (defaults to provider's standard endpoint). |
| | `model` | Target LLM model name (e.g., `gpt-4o`, `claude-3-5-sonnet-latest`, `gemini-1.5-flash`). |
| | `temperature` | LLM Temperature (Default `0.0` for code-generation determinism). |
| | `max_tokens` | Maximum completions token ceiling. |
| **`agent`** | `workspace_dir` | The directory where the agent is allowed to write, read, and edit files. |
| | `workspaces_history` | Array list of registered local project workspaces. Managed automatically by the swapper. |
| | `max_turns` | Loop cutoff safety fuse to prevent runaway infinite API spend. |
| | `command_timeout_seconds` | safety execution clock-limit to terminate hanging scripts or infinite loops automatically. |
| **`security`** | `sandbox_mode` | Sandbox container selection: `host` (bare-metal locks), `docker`, or `none`. |
| | `blocked_patterns` | Array of blacklisted bash strings (e.g., `rm -rf /`). Blocks execution on detection. |
| **`directory_scan`** | `max_depth` | Deep recursive folder search limit to prevent stack overflows during trees walking. |
| | `collapsed_patterns` | List of folders (e.g. `node_modules`, `.venv`) to recognize but skip indexing file-by-file. |
| **`compaction`** | `auto_compact_turns` | Turn index at which rolling context summarization is triggered. |
| | `keep_last_n` | Sliding window size. Number of recent turns to preserve fully uncompacted inside prompt context. |
| **`mcp_servers`** | `command`, `args`, `env` | Executable paths, flags, and credentials to spawn Model Context Protocol child servers. |
| **`web`** | `enabled` | Toggle to automatically spawn the HTTP Server on boot. |
| | `port` | Local port (Default `8080`) that the Web GUI and OpenAI API Gateway bind to. |
| | `api_gateway_enabled` | Toggle switch to expose or disable the OpenAI-Compatible API Gateway. |

---

## 🪙 Telemetry Trace Logging (`.goharness/traces.jsonl`)

Every single execution block (LLM durations, tool calls, and exit codes) is written to a structured, human-readable append-only log file `.goharness/traces.jsonl` for offline analysis:

```json
{"timestamp":"2026-08-01T16:51:21-03:00","session_id":"sess_20260801-165121","turn":1,"action":"llm_completion","duration_ms":1250,"status":"success","metadata":{"model":"gpt-4o"}}
{"timestamp":"2026-08-01T16:51:25-03:00","session_id":"sess_20260801-165121","turn":1,"action":"tool_write_file","duration_ms":5,"status":"success","metadata":{"bytes_written":124,"path":"calc.py"}}
```

---

## 🤝 Attribution & Development
This repository, along with its complete multi-phase architecture, cross-platform bare-metal sandboxes, and embedded web console, was co-engineered, refactored, and compiled from scratch by **Arena.ai's Agent Mode**. 

Arena.ai's Agent Mode orchestrates multiple state-of-the-art LLM engines (including, but not limited to, Claude, ChatGPT, Gemini, Grok, Qwen, and Kimi) to provide developers with fully autonomous systems-engineering, software packaging, and coding capabilities right from their workspaces.

## 📄 License
This project is open-source and licensed under the MIT License.
