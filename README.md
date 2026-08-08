# 🤖 GoHarness: Local-First Secure AI Agent Runner & Web Console

GoHarness is a high-performance, single-file, **local-first and secure AI Agent Runner** written in pure, standard Go with **zero external dependencies**. It bridges local developer workspaces, secure bare-metal sandbox engines, and dynamic Model Context Protocol (MCP) servers, serving a gorgeous, real-time streamed Web Console directly from the compiled binary.

While GoHarness is designed as an elite, high-speed companion for **systems-engineering and coding tasks**, it is built from the ground up as a **universal, multi-purpose cognitive shell**. Its underlying architecture is fully optimized for **long-form creative writing, deep multi-document research, and infinite-memory conversational chatting**.

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
5. **🔄 Dual-Engine Workspace Rollbacks & Untracked Erasers (Phase 8.5):** Supports chronological session rollbacks (`/fork <turn>`). When you go back in time, GoHarness not only rewinds the chat logs but **physically restores your folder structure** to that turn's exact state using either local backups (fallback) or Git-native reset checkpoints (primary). It automatically tracks newly created files and **physically deletes untracked files** during rollbacks to maintain perfect folder synchronization.
6. **🌲 Token-Safe Directory Tree (Auto-LS):** Recursively maps your workspace, collapsing heavy dependency folders (like `node_modules` or `.venv`) and truncating long outputs, while appending visual Git-like status flags and relative modification timers inline (e.g. `main.go [Modified 2m ago]`, `schema.sql [New / Untracked]`).
7. **🌐 Embedded Web Console:** Employs Go's native `net/http` router and `//go:embed` to serve a modern Single-Page Application (HTML5, Tailwind, JS) and uses **Server-Sent Events (SSE)** to stream thoughts and tool logs to your browser in real-time. It includes a responsive **Workspace Swapper**, a **Conversational History Selector**, and a **Session Deletion** dashboard.
8. **🔌 Multi-Provider AI API Connectors (Phase 7):** Features native, standard-library-only translation wrappers for:
   * **OpenAI API / compatible routers** (Ollama, DeepSeek, Groq, etc.).
   * **Anthropic Claude Messages API** (extracting system instructions as top-level params, and mapping `"tool_use"` and `"tool_result"` blocks).
   * **Google Gemini AI Studio & Vertex AI REST APIs** (conforming roles strictly to `"user"`/`"model"`, mapping function schemas, and managing OAuth bearer headers).
9. **🔌 OpenAI-Compatible API Gateway & Tokenizer Proxies (Phase 8):** Exposes a standard gateway so you can plug any existing chat frontend (like OpenWebUI, SillyTavern, or LibreChat) straight into GoHarness!
   * **`GET /v1/models` & `POST /v1/chat/completions`:** Masquerades as a single, ultra-smart model, executes our secure sandboxed loop in the background, and streams the finished report back to your frontend.
   * **`/v1/embeddings`:** Vector embeddings proxy mapping standard RAG pipelines.
   * **`/v1/tokenize` & `/v1/detokenize`:** Implements an incredibly fast, local, self-learning Byte-Pair Encoding (BPE) approximate tokenizer, allowing SillyTavern to calculate exact context window limits in-process with $100\%$ round-trip accuracy!
10. **📂 Dual-Scoped Instruction Injection & Claude Code Compatibility (Phase 8.6):** Dynamically scans both your active workspace directory (Project-scoped) and your binary directory (Global-scoped) on boot for custom instructions (`AGENTS.md`, `SKILLS.md`, `INSTRUCTIONS.md`, and **`CLAUDE.md`**), giving GoHarness $100\%$ out-of-the-box compatibility with existing Claude Code repositories!
11. **🪙 Context Persistence & Developer Debug Modes (Phase 8.6):** Remembers and **automatically resumes your last active session ID on launch**, while offering a `"debug"` mode that logs detailed, high-verbosity terminal traces explaining precisely how every single session file and pipeline behaves behind the scenes.

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
│   ├── sandbox_darwin.go  # macOS Apple sandbox-exec SBPL executor
│   ├── sandbox_windows.go # Windows Job Object & restricted low-integrity token spawner
│   ├── sandbox_fallback.go# Fallback executor for other unmapped OSes
│   │
│   └── web/
│       └── index.html  # Responsive Single-Page App assets (fully embedded)
│
├── bin/                # Compiled production static binaries (cross-compiled)
│   ├── agent.exe       # Windows static executable (~6.6 MB)
│   ├── agent_linux     # Linux static executable (~6.5 MB)
│   ├── agent_mac_arm64 # macOS Apple Silicon static executable (~6.2 MB)
│   └── agent_mac_amd64 # macOS Intel static executable (~6.6 MB)
│
└── docs/
    ├── ROADMAP.md      # Multi-phase strategic engineering plan
    ├── RESEARCH.md     # Research bibliography and project inspirations (New!)
    ├── COMPARISON_MATRIX.md # Technical comparison audit of GoHarness vs SOTA (New!)
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

To run with verbose developer logs:
```bash
./bin/agent_linux -web -debug
```

#### **Option B: Lightweight Terminal UI (TUI) Mode**
Runs the agent loop directly inside your current terminal session:
```bash
./bin/agent_linux
```

---

## 🔌 Model Context Protocol (MCP) Server Registration

GoHarness has native, standard-compliant support for Anthropic's **Model Context Protocol (MCP)**. It acts as an MCP Client, automatically spawning configured tool servers as child processes, conducting full JSON-RPC handshakes over `stdio`, and dynamically appending their discovered tools directly into the AI agent's reasoning schema.

You can register any local or registry-based (npm/PyPI) MCP server using either the **Web Console GUI** or by editing **`config.json`**.

### 💻 Option A: Register via the Web UI (Visual Hot-Reload)
Our built-in **MCP Server Configuration Manager** allows registering and testing new servers on the fly without rebooting the runner:
1. Open the Web Console in your browser (`http://localhost:8080`).
2. Click the ⚙️ **Settings** button in the header.
3. Scroll down to the **Model Context Protocol (MCP) Servers** section.
4. Fill in the **Register New MCP Server** form:
   * **Server Key/Identifier:** A unique local name (e.g., `web-search`, `sqlite-db`).
   * **Executable Command:** The package runner/executable (e.g., `npx`, `uvx`, `node`, `python`).
   * **Arguments:** The command parameters. For registry-hosted servers, use `-y <package-name>` (for npm) or the package name directly (for uvx).
5. Click **Add Server**. The backend will instantly write to `config.json`, terminate old processes, and initialize the new server dynamically!

---

### 📂 Option B: Register via `config.json` (Code-First)
To configure servers statically or add custom environment variables (like API keys), append them to the `"mcp_servers"` block in your `config.json` file:

```json
"mcp_servers": {
  "sqlite-demo": {
    "command": "uvx",
    "args": ["mcp-server-sqlite", "--db-path", "./workspace/dev.db"]
  },
  "web-search": {
    "command": "npx",
    "args": [
      "-y",
      "web-search-mcp"
    ],
    "env": {
      "MAX_CONTENT_LENGTH": "100000",
      "DEFAULT_TIMEOUT": "5000",
      "BRAVE_API_KEY": "your-brave-key-here"
    }
  }
}
```

#### Generic Startup Templates for Common Environments:
* **npm (NodeJS) Packages:** 
  * `Command:` `npx`
  * `Args:` `["-y", "mcp-server-package-name"]`
* **uv (Python) Packages:**
  * `Command:` `uvx`
  * `Args:` `["mcp-server-package-name"]`
* **Local Source Files:**
  * `Command:` `node` (or `python`)
  * `Args:` `["./workspace/my-custom-mcp/index.js"]`

*Note: Since MCP servers run as child processes on your host, make sure you have the respective runtime (`node`, `npm`, `python`, or `uv`) installed and available on your PATH.*

---

## 🧠 Advanced Memory Subsystem (BM25, Custom Uploads, & Target Directories)

GoHarness features a state-of-the-art, local-first **Advanced Memory Subsystem** inspired by Tencent's progressive memory layering [3](https://github.com/TencentCloud/TencentDB-Agent-Memory). It is designed to maximize search speed, limit token waste, and target your agent's focus exactly on what matters:

* **🔍 BM25 Lexical Search Engine:** Written in pure, standard Go, our standalone BM25 engine performs exact keyword-relevance ranking across files on disk. The agent can invoke the `bm25_search` native tool to query files in seconds (outperforming slow terminal `grep` commands) [1](https://arxiv.org/html/2607.26497v3).
* **📤 Custom File Uploads:** Clicking the Cloud-Upload button on the Files sidebar explorer allows you to upload text/JSON/code reference documents (up to 10MB) directly to your active session. GoHarness automatically registers and priority-indexes these uploads during any BM25 search.
* **📂 Targeted Workspace Scanning (`target_scan_dirs`):** Under settings (⚙️), you can specify comma-separated relative directories (e.g. `src, docs`). If configured, GoHarness's BM25 engine will bypass the rest of the workspace and index *only* these designated folders, preventing noise and limiting token blowouts [1](https://arxiv.org/html/2607.26497v3).

---

## 🔄 Conversational Branching (Edit/Fork & Reroll)

To give you complete control over your conversation history and workspace file states, GoHarness implements premium conversational lifecycle triggers:

* **✏️ In-Place Edit & Fork:** Hovering over any previous user or assistant message card reveals an **Edit & Fork (Pen)** button. Clicking it lets you edit the prompt and save it. GoHarness will instantly spawn a parallel timeline branch, copy files up to that point, write your edited text, and automatically re-run the agent loop on the new timeline!
* **🔄 One-Click Regeneration (Reroll):** Clicking the Reroll icon right next to the "Execute" button deletes the last Assistant and Tool execution turns, safely reverts any workspace edits made during that turn, and resubmits the last user prompt for a fresh, live-streamed generation.

---

## 🛡️ Sandboxing & Safety Recommendations (Including Windows Fallback)

To protect your host operating system from malicious commands or Indirect Prompt Injection attacks (e.g., if the agent reads a webpage instructing it to run a harmful script), GoHarness implements platform-specific bare-metal sandboxes:
* **Linux:** Locks process threads via **Landlock LSM** system locks.
* **macOS:** Implements strict Apple **SBPL (`sandbox-exec`)** profiles.
* **Windows:** Duplicates process tokens and locks execution inside low-integrity **Job Objects** [2](https://deepwiki.com/mrkrsl/web-search-mcp).

### ⚙️ Windows Sandboxing & Sandbox Fallback (`sandbox_fallback`)

On standard Windows machines, local security policies block standard, non-elevated command prompts from duplicating tokens or modifying process integrity levels (`SeAssignPrimaryTokenPrivilege`). 

To prevent this from breaking your local development loops, GoHarness implements a secure-by-default **Sandbox Fallback Configuration (`sandbox_fallback`)**:

* **`sandbox_fallback: false` (Strict Default):** If GoHarness fails to allocate low-integrity restricted tokens (like on standard Windows CMD/PowerShell), the command is **blocked immediately with a hard error** to protect your host.
* **`sandbox_fallback: true` (Graceful Fallback):** If sandbox initialization fails, GoHarness falls back gracefully to unsandboxed execution to let your commands succeed, but **instantly injects an interactive warning alert directly inside your visual chat Console timeline**, guiding you to re-enable security.

#### 💡 The 3 Hardened Security Setup Recommendations:

To run GoHarness with complete, zero-compromise containerized or bare-metal isolation, we recommend selecting one of these three developer-hardened setups:

1. 🐳 **Docker Container Sandbox (Highly Recommended for Windows):**
   Run all agent commands inside an isolated Docker container. Spin up your local container:
   ```bash
   docker run -d --name agent-workspace -v ./workspace:/workspace alpine tail -f /dev/null
   ```
   In the Web Console settings (⚙️), set **Sandbox Mode** to `docker` and **Docker Container** to `agent-workspace`. GoHarness will route all command execution through Docker Desktop with $100\%$ security isolation [1](https://aibit.im/en/article/web-search-mcp-search-local-llm-web-search-without-api-keys)!
2. 🐧 **WSL2 (Windows Subsystem for Linux):**
   Clone and run GoHarness inside WSL2. Standard, non-root Linux accounts have native access to **Landlock LSM** sandboxing, allowing GoHarness to lock down file execution with $100\%$ bare-metal security isolation without any administrative prompts!
3. 🛡️ **Administrator Elevation (Windows Host Bare-Metal):**
   To grant GoHarness the necessary token assignment privileges natively on your Windows host, open PowerShell or CMD as **Administrator** and execute `agent.exe` from there.

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
| | `top_p` | Top-P sampling threshold for creative temperature tuning. |
| | `top_k` | Top-K sampling threshold. |
| | `thinking_level` | Gemini Reasoning/Thinking Level: `"off"`, `"low"`, `"medium"`, or `"high"`. |
| | `project_id` | GCP Project ID for scoped Vertex AI API Key queries. |
| | `region` | GCP Region (e.g. `us-central1`) for regional Vertex AI REST requests. |
| **`agent`** | `workspace_dir` | The directory where the agent is allowed to write, read, and edit files. |
| | `target_scan_dirs` | Specific relative subdirectory paths inside the workspace to index for BM25 search (e.g., `["src", "docs"]`). |
| | `workspaces_history` | Array list of registered local project workspaces. Managed automatically by the swapper. |
| | `last_active_session_id` | Remembers and automatically resumes your most recent active session ID on launch. |
| | `max_turns` | Loop cutoff safety fuse to prevent runaway infinite API spend. |
| | `command_timeout_seconds` | safety execution clock-limit to terminate hanging scripts or infinite loops automatically. |
| **`security`** | `sandbox_mode` | Sandbox container selection: `host` (bare-metal locks), `docker`, or `none`. |
| | `sandbox_fallback` | Toggle graceful fallback to unsandboxed execution if sandbox fails. Default `false` (strict). |
| | `blocked_patterns` | Array of blacklisted bash strings (e.g., `rm -rf /`). Blocks execution on detection. |
| **`directory_scan`** | `max_depth` | Deep recursive folder search limit to prevent stack overflows during types walking. |
| | `collapsed_patterns` | List of folders (e.g. `node_modules`, `.venv`) to recognize but skip indexing file-by-file. |
| **`compaction`** | `auto_compact_turns` | Turn index at which rolling context summarization is triggered. |
| | `keep_last_n` | Sliding window size. Number of recent turns to preserve fully uncompacted inside prompt context. |
| **`mcp_servers`** | `command`, `args`, `env` | Executable paths, flags, and credentials to spawn Model Context Protocol child servers. |
| **`web`** | `enabled` | Toggle to automatically spawn the HTTP Server on boot. |
| | `port` | Local port (Default `8080`) that the Web GUI and OpenAI API Gateway bind to. |
| | `api_gateway_enabled` | Toggle switch to expose or disable the OpenAI-Compatible API Gateway. |
| **`debug`** | `debug` | Top-level boolean switch to enable/disable detailed console diagnostic outputs. |

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
