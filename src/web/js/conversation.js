import { DEFAULT_PROMPT_PLACEHOLDER, currentSessionKey, storageKey } from "./state.js";

const SLASH_COMMAND_SUGGESTIONS = [
  { label: "/workflows", insert: "/workflows", description: "List registered workflows" },
  { label: "/workflow ", insert: "/workflow ", description: "Switch active runtime workflow" },
  { label: "/compact", insert: "/compact", description: "Run manual context compaction" },
  { label: "/new", insert: "/new", description: "Start a new session" },
  { label: "/settings", insert: "/settings", description: "Open Settings" },
  { label: "/workflow-lab", insert: "/workflow-lab", description: "Open Workflow Lab" },
];

const WF_NODE_META = {
  llm: { icon: "fa-robot", accent: "text-purple-400" },
  llm_query: { icon: "fa-robot", accent: "text-purple-400" },
  llm_synthesis: { icon: "fa-layer-group", accent: "text-blue-400" },
  bm25_search: { icon: "fa-magnifying-glass", accent: "text-indigo-400" },
  tool_execution: { icon: "fa-terminal", accent: "text-cyan-400" },
  conditional_router: { icon: "fa-route", accent: "text-amber-400" },
};

export function createConversationModule({
  state,
  actions,
  escapeHtml,
  toolCallTitle,
  toolResultTitle,
  deliverablePathsFromTool,
  renderDeliverableChips,
  renderToolCallArgumentsBody,
  renderToolBody,
}) {
  function persistSessionUIState() {
    try {
      const key = currentSessionKey();
      sessionStorage.setItem(storageKey("queue", key), JSON.stringify(state.sessionQueues[key] || []));
      sessionStorage.setItem(storageKey("staged", key), JSON.stringify(state.stagedContextBySession[key] || []));
    } catch {}
  }

  function restoreSessionUIState(sessionKey) {
    const key = sessionKey || currentSessionKey();
    try {
      const q = sessionStorage.getItem(storageKey("queue", key));
      const s = sessionStorage.getItem(storageKey("staged", key));
      state.sessionQueues[key] = q ? JSON.parse(q) : (state.sessionQueues[key] || []);
      state.stagedContextBySession[key] = s ? JSON.parse(s) : (state.stagedContextBySession[key] || []);
    } catch {
      state.sessionQueues[key] = state.sessionQueues[key] || [];
      state.stagedContextBySession[key] = state.stagedContextBySession[key] || [];
    }
  }

  function isSessionBusy() {
    return state.currentUIState && state.currentUIState.status === "busy";
  }

  function queueForCurrentSession() {
    const key = currentSessionKey();
    if (!state.sessionQueues[key]) restoreSessionUIState(key);
    state.sessionQueues[key] = state.sessionQueues[key] || [];
    return state.sessionQueues[key];
  }

  function stagedContextForCurrentSession() {
    const key = currentSessionKey();
    if (!state.stagedContextBySession[key]) restoreSessionUIState(key);
    state.stagedContextBySession[key] = state.stagedContextBySession[key] || [];
    return state.stagedContextBySession[key];
  }

  function queuePromptText(text, kind) {
    const clean = String(text || "").trim();
    if (!clean) return;
    const queue = queueForCurrentSession();
    const item = { id: "q-" + (state.queueItemSeq++), text: clean, kind: kind === "follow_up" ? "follow_up" : "steering" };
    if (item.kind === "follow_up") {
      queue.push(item);
    } else {
      const firstFollow = queue.findIndex(entry => entry.kind === "follow_up");
      if (firstFollow === -1) queue.push(item);
      else queue.splice(firstFollow, 0, item);
    }
    persistSessionUIState();
    renderQueuedMessages();
    renderComposerActionRow();
  }

  function removeQueuedMessage(id) {
    state.sessionQueues[currentSessionKey()] = queueForCurrentSession().filter(item => item.id !== id);
    persistSessionUIState();
    renderQueuedMessages();
    renderComposerActionRow();
  }

  function editQueuedMessage(id) {
    const queue = queueForCurrentSession();
    const idx = queue.findIndex(item => item.id === id);
    if (idx === -1) return;
    const [item] = queue.splice(idx, 1);
    const input = document.getElementById("prompt-input");
    if (input) {
      input.value = item.text;
      input.style.height = "auto";
      input.style.height = input.scrollHeight + "px";
      input.focus();
    }
    persistSessionUIState();
    renderQueuedMessages();
    renderComposerActionRow();
    updateTriggerOverlay();
  }

  function renderQueuedMessages() {
    const el = document.getElementById("queued-messages-strip");
    if (!el) return;
    const queue = queueForCurrentSession();
    if (queue.length === 0) {
      el.classList.add("hidden");
      el.innerHTML = "";
      return;
    }
    el.classList.remove("hidden");
    el.innerHTML = `<div class="text-[10px] uppercase tracking-wider text-slate-500 mb-2">Queued messages</div>` + queue.map(item => `
      <div class="flex items-center gap-2 rounded border border-[#334155] bg-slate-950/40 px-3 py-2 text-xs mb-2 last:mb-0">
        <span class="px-1.5 py-0.5 rounded ${item.kind === "follow_up" ? "bg-purple-950/50 text-purple-300" : "bg-amber-950/40 text-amber-300"} font-mono">${item.kind === "follow_up" ? "follow-up" : "steering"}</span>
        <span class="flex-1 text-slate-300 truncate">${escapeHtml(item.text)}</span>
        <button type="button" onclick="editQueuedMessage('${item.id}')" class="text-slate-400 hover:text-white">Edit</button>
        <button type="button" onclick="removeQueuedMessage('${item.id}')" class="text-slate-500 hover:text-red-400">✕</button>
      </div>`).join("");
  }

  async function stageWorkspaceFile(path) {
    if (!path) return;
    const staged = stagedContextForCurrentSession();
    if (staged.some(item => item.path === path)) {
      renderStagedContextStrip();
      return;
    }
    try {
      const res = await fetch("/api/workspace/file?path=" + encodeURIComponent(path));
      if (!res.ok) throw new Error("Failed to read file");
      const data = await res.json();
      staged.push({ id: "ctx-" + (state.queueItemSeq++), type: "file", path: data.path, content: data.content });
      state.currentFilePreview = data;
      persistSessionUIState();
      renderStagedContextStrip();
      renderComposerActionRow();
      if (actions.switchDetailsTab) actions.switchDetailsTab("file");
    } catch (err) {
      appendSystemAlert("Stage Context Failed", "Could not stage file: " + err.message, "fa-triangle-exclamation text-red-400");
    }
  }

  function removeStagedContext(id) {
    state.stagedContextBySession[currentSessionKey()] = stagedContextForCurrentSession().filter(item => item.id !== id);
    persistSessionUIState();
    renderStagedContextStrip();
    renderComposerActionRow();
  }

  function clearStagedContext() {
    state.stagedContextBySession[currentSessionKey()] = [];
    persistSessionUIState();
    renderStagedContextStrip();
    renderComposerActionRow();
  }

  function renderStagedContextStrip() {
    const el = document.getElementById("staged-context-strip");
    if (!el) return;
    const staged = stagedContextForCurrentSession();
    if (staged.length === 0) {
      el.classList.add("hidden");
      el.innerHTML = "";
      return;
    }
    el.classList.remove("hidden");
    el.innerHTML = `
      <div class="flex items-center justify-between gap-2 mb-2">
        <div class="text-[10px] uppercase tracking-wider text-slate-500">Staged context for next send</div>
        <button type="button" onclick="clearStagedContext()" class="text-[10px] text-slate-500 hover:text-red-400">Clear all</button>
      </div>
      <div class="flex flex-wrap gap-2">` + staged.map(item => `
        <div class="inline-flex items-center gap-2 rounded border border-cyan-900/40 bg-cyan-950/10 px-2.5 py-1.5 text-xs">
          <span class="text-cyan-300 font-mono truncate max-w-[260px]">${escapeHtml(item.path)}</span>
          <button type="button" onclick="openWorkspaceFile(${JSON.stringify(item.path)})" class="text-cyan-400 hover:text-cyan-300">Open</button>
          <button type="button" onclick="removeStagedContext('${item.id}')" class="text-slate-500 hover:text-red-400">✕</button>
        </div>`).join("") + `</div>`;
  }

  function buildPromptWithStagedContext(prompt) {
    const staged = stagedContextForCurrentSession();
    if (staged.length === 0) return prompt;
    const blocks = staged.map(item => `\n\n## Staged file: ${item.path}\n\n\`\`\`text\n${item.content}\n\`\`\``).join("");
    state.stagedContextBySession[currentSessionKey()] = [];
    persistSessionUIState();
    renderStagedContextStrip();
    renderComposerActionRow();
    return prompt + blocks;
  }

  function renderComposerActionRow() {
    const providerChip = document.getElementById("composer-provider-chip");
    const modelChip = document.getElementById("composer-model-chip");
    const toolsChip = document.getElementById("composer-tools-chip");
    const approvalsChip = document.getElementById("composer-approvals-chip");
    const sourcesChip = document.getElementById("composer-sources-chip");
    const scopeChip = document.getElementById("composer-scope-chip");
    const queueChip = document.getElementById("composer-queue-chip");
    if (!providerChip) return;
    providerChip.textContent = `Provider · ${(state.currentUIState && state.currentUIState.status === "blocked" && !state.activeWorkspaceDir) ? "unset" : ((window.__cfgProvider || "openai").toUpperCase())}`;
    modelChip.textContent = `Model · ${window.__cfgModel || "unset"}`;
    toolsChip.textContent = `Tools · ${state.currentToolPreview ? "latest output" : "inspect"}`;
    approvalsChip.textContent = `Approvals · ${isSessionBusy() ? "none pending" : "idle"}`;
    sourcesChip.textContent = `Sources · ${stagedContextForCurrentSession().length} staged`;
    scopeChip.textContent = `Scope · ${stagedContextForCurrentSession().length > 0 ? "staged files" : "workspace-wide"}`;
    queueChip.textContent = `Queue · ${queueForCurrentSession().length}`;
  }

  function renderComposerTakeover() {
    const el = document.getElementById("composer-takeover");
    if (!el) return;
    if (isSessionBusy()) {
      el.classList.add("hidden");
      el.innerHTML = "";
      return;
    }
    const blocked = state.currentUIState && state.currentUIState.status === "blocked";
    if (!blocked || !state.currentUIState.cta_action) {
      el.classList.add("hidden");
      el.innerHTML = "";
      return;
    }
    el.classList.remove("hidden");
    el.innerHTML = `
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="font-bold text-amber-300">${escapeHtml(state.currentUIState.title || "Blocked")}</div>
          <div class="text-[11px] text-slate-300">${escapeHtml(state.currentUIState.blocked_reason || state.currentUIState.summary || "GoHarness is blocked.")}</div>
        </div>
        <button type="button" onclick="runComposerCta()" class="px-3 py-2 rounded bg-amber-600 hover:bg-amber-500 text-white text-xs font-bold">${escapeHtml(state.currentUIState.cta_label || "Resolve")}</button>
      </div>`;
  }

  function updateTriggerOverlay() {
    const overlay = document.getElementById("trigger-overlay");
    const input = document.getElementById("prompt-input");
    if (!overlay || !input) return;
    const value = input.value || "";
    const trimmed = value.trimStart();
    let items = [];
    let mode = null;

    if (trimmed.startsWith("/")) {
      mode = "slash";
      const q = trimmed.slice(1).toLowerCase();
      items = SLASH_COMMAND_SUGGESTIONS
        .filter(item => item.label.slice(1).toLowerCase().includes(q))
        .slice(0, 6)
        .map(item => ({ ...item, action: () => applySlashSuggestion(item.insert) }));
    } else {
      const atMatch = value.match(/(?:^|\s)@([^\s]*)$/);
      if (atMatch) {
        mode = "file";
        const q = atMatch[1].toLowerCase();
        items = state.workspaceTreeEntries
          .filter(entry => !entry.isDir && entry.path.toLowerCase().includes(q))
          .slice(0, 8)
          .map(entry => ({ label: "@" + entry.path, description: "Stage file into the next prompt", action: () => applyFileSuggestion(entry.path) }));
      } else if (/^!!?/.test(trimmed)) {
        mode = "shell";
        items = [
          { label: "!command", description: "Run a sandboxed shell command, then send its output to the model", action: () => applyShellSuggestion("!") },
          { label: "!!command", description: "Run a sandboxed shell command without sending its output to the model", action: () => applyShellSuggestion("!!") },
        ];
      }
    }

    if (!mode || items.length === 0) {
      overlay.classList.add("hidden");
      overlay.innerHTML = "";
      return;
    }

    overlay.classList.remove("hidden");
    overlay.innerHTML = `<div class="text-[10px] uppercase tracking-wider text-slate-500 px-3 py-2 border-b border-[#334155]/60">${mode === "slash" ? "Commands" : mode === "file" ? "Stage file" : "Shell shortcuts"}</div>` + items.map((item, idx) => `
      <button type="button" data-trigger-index="${idx}" class="w-full text-left px-3 py-2 hover:bg-slate-800/60 border-b border-[#334155]/40 last:border-b-0">
        <div class="text-xs font-mono text-slate-200">${escapeHtml(item.label)}</div>
        <div class="text-[11px] text-slate-500">${escapeHtml(item.description || "")}</div>
      </button>`).join("");

    overlay.querySelectorAll("[data-trigger-index]").forEach((btn, idx) => {
      btn.addEventListener("click", () => items[idx].action());
    });
  }

  function hideTriggerOverlay() {
    const overlay = document.getElementById("trigger-overlay");
    if (!overlay) return;
    overlay.classList.add("hidden");
    overlay.innerHTML = "";
  }

  function handleComposerInput() {
    updateTriggerOverlay();
  }

  function applySlashSuggestion(insert) {
    const input = document.getElementById("prompt-input");
    if (!input) return;
    input.value = insert;
    input.focus();
    input.style.height = "auto";
    input.style.height = input.scrollHeight + "px";
    hideTriggerOverlay();
  }

  function applyFileSuggestion(path) {
    stageWorkspaceFile(path);
    const input = document.getElementById("prompt-input");
    if (input) {
      input.value = input.value.replace(/(?:^|\s)@([^\s]*)$/, "").trimStart();
      input.focus();
      input.style.height = "auto";
      input.style.height = input.scrollHeight + "px";
    }
    hideTriggerOverlay();
  }

  function applyShellSuggestion(prefix) {
    const input = document.getElementById("prompt-input");
    if (!input) return;
    if (!input.value.trim()) input.value = prefix;
    else if (!input.value.trimStart().startsWith("!")) input.value = prefix + " ";
    input.focus();
    input.style.height = "auto";
    input.style.height = input.scrollHeight + "px";
    hideTriggerOverlay();
  }

  function maybeHandleSlashCommand(prompt) {
    const trimmed = prompt.trim();
    if (trimmed === "/compact") {
      if (actions.triggerCompaction) actions.triggerCompaction();
      appendSystemAlert("Compaction Requested", "Queued a manual context compaction in the background.", "fa-compress text-indigo-400");
      return true;
    }
    if (trimmed === "/settings") {
      if (actions.openSettingsModal) actions.openSettingsModal();
      setTimeout(() => actions.switchSettingsTab && actions.switchSettingsTab("standard"), 0);
      return true;
    }
    if (trimmed === "/workflow-lab") {
      if (actions.switchPrimarySurface) actions.switchPrimarySurface("workflow");
      return true;
    }
    if (trimmed === "/new") {
      if (actions.triggerNewSession) actions.triggerNewSession();
      return true;
    }
    return false;
  }

  async function executeDirectCommand(command) {
    const res = await fetch("/api/command", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ command }),
    });
    if (!res.ok) throw new Error("Command execution failed");
    const data = await res.json();
    state.currentToolPreview = { role: "tool", name: `shell · ${command}`, content: data.result || "", meta: { artifacts: data.artifacts || [] } };
    if (actions.recordTrajectoryEvent) actions.recordTrajectoryEvent("tool", "direct_shell_command", command);
    if (actions.switchDetailsTab) actions.switchDetailsTab("tool");
    appendSystemAlert("Shell Command Completed", `Executed sandboxed command: ${command}`, "fa-terminal text-cyan-400");
    return data.result || "";
  }

  async function dispatchPromptText(prompt) {
    const text = String(prompt || "").trim();
    if (!text) return;
    if (maybeHandleSlashCommand(text)) return;

    const directOnly = text.trimStart().startsWith("!!");
    const shellToModel = !directOnly && text.trimStart().startsWith("!");
    if (directOnly || shellToModel) {
      const command = text.trimStart().replace(/^!!?\s*/, "");
      if (!command) return;
      const result = await executeDirectCommand(command);
      if (directOnly) return;
      const forwarded = `Direct shell command executed before this prompt:\n$ ${command}\n\n${result}\n\nPlease analyze the command output and continue from there.`;
      return dispatchPromptText(forwarded);
    }

    const finalPrompt = buildPromptWithStagedContext(text);
    state.turnCounter++;
    appendTurnToChat({ role: "user", content: text, turn_number: state.turnCounter });
    setLocalBusyState();
    const res = await fetch("/api/prompt", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ prompt: finalPrompt }),
    });
    if (!res.ok) throw new Error("API call failed");
  }

  async function processQueuedMessages() {
    if (state.queueDispatchInFlight || isSessionBusy()) return;
    if (state.currentUIState && state.currentUIState.status === "blocked") return;
    const queue = queueForCurrentSession();
    if (queue.length === 0) return;
    const item = queue.shift();
    persistSessionUIState();
    renderQueuedMessages();
    renderComposerActionRow();
    state.queueDispatchInFlight = true;
    try {
      await dispatchPromptText(item.text);
    } catch (err) {
      appendSystemAlert("Queued Message Failed", err.message || String(err), "fa-triangle-exclamation text-red-400");
      if (actions.fetchConfig) actions.fetchConfig();
    } finally {
      state.queueDispatchInFlight = false;
      setTimeout(processQueuedMessages, 0);
    }
  }

  function applyThemeChrome() {
    if (!document.body.dataset.theme) document.body.dataset.theme = "dark";
    window.requestAnimationFrame(() => {
      const bg = getComputedStyle(document.body).getPropertyValue("--gh-bg-app").trim() || "#0f172a";
      const meta = document.querySelector('meta[name="theme-color"]');
      if (meta) meta.setAttribute("content", bg);
      document.documentElement.style.colorScheme = document.body.dataset.theme === "light" ? "light" : "dark";
    });
  }

  function removeEmptyHero() {
    const hero = document.getElementById("empty-hero");
    if (hero) hero.remove();
  }

  function seedPromptExample(prompt) {
    const input = document.getElementById("prompt-input");
    if (!input || input.disabled) return;
    input.value = prompt;
    input.style.height = "auto";
    input.style.height = input.scrollHeight + "px";
    input.focus();
  }

  function runComposerCta() {
    const action = state.currentUIState && state.currentUIState.cta_action;
    if (action === "open_settings") {
      if (actions.openSettingsModal) actions.openSettingsModal();
      setTimeout(() => actions.switchSettingsTab && actions.switchSettingsTab("standard"), 0);
      return;
    }
    if (action === "open_workspace") {
      if (actions.switchSidebarTab) actions.switchSidebarTab("sessions");
      setTimeout(() => {
        const input = document.getElementById("new-workspace-input");
        if (input) input.focus();
      }, 0);
      return;
    }
    const input = document.getElementById("prompt-input");
    if (input && !input.disabled) input.focus();
  }

  function handleComposerShellClick(event) {
    if (event.target.closest("button")) return;
    if (isSessionBusy()) {
      const input = document.getElementById("prompt-input");
      if (input) input.focus();
      return;
    }
    if (state.currentUIState && state.currentUIState.composer_enabled) return;
    if (!state.currentUIState || !state.currentUIState.cta_action) return;
    runComposerCta();
  }

  function setLocalBusyState() {
    state.currentUIState = {
      composer_enabled: false,
      status: "busy",
      blocked_reason: "GoHarness is already executing this session. Queue a steering message or wait for the current run to finish.",
      cta_action: "",
      cta_label: "",
      title: "Current session is running",
      summary: "Live tool output and assistant replies will continue streaming below until the run finishes."
    };
    updateComposerState();
    renderComposerActionRow();
    renderComposerTakeover();
    renderEmptyHero();
  }

  function updateComposerState() {
    const appUiState = state.currentUIState || { composer_enabled: true, status: "ready" };
    const form = document.getElementById("prompt-form");
    const shell = document.getElementById("composer-shell");
    const input = document.getElementById("prompt-input");
    const submit = document.getElementById("prompt-submit");
    const submitLabel = document.getElementById("prompt-submit-label");
    const reroll = document.getElementById("reroll-btn");
    const cta = document.getElementById("composer-cta");
    const ctaLabel = document.getElementById("composer-cta-label");
    const hint = document.getElementById("composer-state-hint");
    if (!form || !shell || !input || !submit || !submitLabel || !reroll || !cta || !ctaLabel || !hint) return;

    const busy = appUiState.status === "busy";
    const enabled = !!appUiState.composer_enabled || busy;
    input.disabled = !enabled;
    input.readOnly = !enabled;
    input.placeholder = busy
      ? "Queue a steering message for after the current run, or use Alt+Enter for a follow-up..."
      : (enabled ? DEFAULT_PROMPT_PLACEHOLDER : (appUiState.blocked_reason || "GoHarness is not ready yet."));
    submit.disabled = !enabled;
    submitLabel.textContent = busy ? "Queue" : "Execute";
    reroll.disabled = !appUiState.composer_enabled;
    shell.classList.toggle("is-disabled", !enabled);
    shell.classList.toggle("is-clickable", !enabled && !!appUiState.cta_action);
    form.setAttribute("aria-disabled", enabled ? "false" : "true");

    if (busy) {
      cta.classList.add("hidden");
      cta.classList.remove("inline-flex");
      hint.innerHTML = 'Current run active. Press <kbd class="bg-slate-800 px-1.5 py-0.5 rounded text-slate-400">Enter</kbd> to queue steering, <kbd class="bg-slate-800 px-1.5 py-0.5 rounded text-slate-400">Alt+Enter</kbd> to queue follow-up, and <kbd class="bg-slate-800 px-1.5 py-0.5 rounded text-slate-400">Shift+Enter</kbd> for new line.';
    } else if (enabled) {
      cta.classList.add("hidden");
      cta.classList.remove("inline-flex");
      hint.innerHTML = 'GoHarness Sandbox Mode is active. System write-protection active. Press <kbd class="bg-slate-800 px-1.5 py-0.5 rounded text-slate-400">Enter</kbd> to execute, <kbd class="bg-slate-800 px-1.5 py-0.5 rounded text-slate-400">Shift+Enter</kbd> for new line.';
    } else {
      if (appUiState.cta_label) {
        ctaLabel.textContent = appUiState.cta_label;
        cta.classList.remove("hidden");
        cta.classList.add("inline-flex");
      } else {
        cta.classList.add("hidden");
        cta.classList.remove("inline-flex");
      }
      hint.textContent = appUiState.blocked_reason || "GoHarness is not ready yet.";
    }

    renderComposerTakeover();
    renderComposerActionRow();
    if (!busy) setTimeout(processQueuedMessages, 0);
  }

  function renderEmptyHero() {
    const chatContainer = document.getElementById("chat-messages");
    if (!chatContainer) return;
    const hasOtherContent = Array.from(chatContainer.children).some(el => el.id !== "empty-hero");
    if (hasOtherContent) {
      removeEmptyHero();
      return;
    }
    removeEmptyHero();
    const hero = document.createElement("div");
    hero.id = "empty-hero";
    hero.className = "gh-hero max-w-4xl mx-auto";
    hero.dataset.state = (state.currentUIState && state.currentUIState.status) || "ready";
    const title = escapeHtml((state.currentUIState && state.currentUIState.title) || "GoHarness is ready");
    const summary = escapeHtml((state.currentUIState && state.currentUIState.summary) || "");
    const blocked = state.currentUIState && state.currentUIState.status !== "ready";
    const cta = state.currentUIState && state.currentUIState.cta_label
      ? `<button type="button" onclick="runComposerCta()" class="gh-hero-action mt-5 inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-bold text-white shadow-md"><i class="fa-solid fa-arrow-right"></i><span>${escapeHtml(state.currentUIState.cta_label)}</span></button>`
      : "";
    const examples = blocked ? "" : `
      <div class="mt-6 grid gap-2 sm:grid-cols-3">
        <button type="button" onclick="seedPromptExample('Summarize this repository architecture and call out the risky parts.')" class="gh-hero-example rounded-lg px-3 py-2 text-left text-xs transition">Summarize this repository architecture and call out the risky parts.</button>
        <button type="button" onclick="seedPromptExample('Run the tests, explain the failures, and propose the minimal fix.')" class="gh-hero-example rounded-lg px-3 py-2 text-left text-xs transition">Run the tests, explain the failures, and propose the minimal fix.</button>
        <button type="button" onclick="seedPromptExample('Inspect workflows.json and explain how the active DAG executes this task.')" class="gh-hero-example rounded-lg px-3 py-2 text-left text-xs transition">Inspect workflows.json and explain how the active DAG executes this task.</button>
      </div>`;
    hero.innerHTML = `
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="max-w-2xl">
          <div class="inline-flex items-center gap-2 rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-[0.18em] gh-hero-chip">
            <i class="fa-solid ${state.currentUIState && state.currentUIState.status === "busy" ? "fa-spinner fa-spin" : state.currentUIState && state.currentUIState.status === "blocked" ? "fa-triangle-exclamation" : "fa-sparkles"}"></i>
            <span>${escapeHtml((state.currentUIState && state.currentUIState.status) || "ready")}</span>
          </div>
          <h2 class="mt-4 text-2xl font-bold text-slate-100">${title}</h2>
          <p class="mt-2 text-sm leading-relaxed text-slate-300">${summary}</p>
          ${cta}
        </div>
        <div class="grid gap-2 text-xs sm:min-w-[220px]">
          <div class="gh-hero-chip rounded-lg px-3 py-2"><span class="block text-slate-500 uppercase tracking-wider text-[10px]">Workspace</span><span class="mt-1 block font-mono text-slate-200">${escapeHtml(state.activeWorkspaceDir || "unset")}</span></div>
          <div class="gh-hero-chip rounded-lg px-3 py-2"><span class="block text-slate-500 uppercase tracking-wider text-[10px]">Connection</span><span class="mt-1 block font-mono text-slate-200">${escapeHtml((document.getElementById("model-name") && document.getElementById("model-name").innerText) || "unset")}</span></div>
        </div>
      </div>
      ${examples}
    `;
    chatContainer.appendChild(hero);
  }

  function appendGreeting() {
    renderEmptyHero();
  }

  function appendTurnToChat(turn) {
    const chatContainer = document.getElementById("chat-messages");
    if (!chatContainer) return;
    removeEmptyHero();

    let avatarChar = "U";
    let avatarBg = "bg-slate-700";
    let roleName = "You";
    let cardBg = "bg-[#0b0f19]";
    let borderStyle = "";

    if (turn.role === "assistant") {
      avatarChar = "🤖";
      avatarBg = "bg-yellow-600";
      roleName = "Assistant";
      cardBg = "bg-slate-900/30";
      borderStyle = "border-l-2 border-yellow-500";
    } else if (turn.role === "tool") {
      avatarChar = "🛠️";
      avatarBg = "bg-cyan-700";
      roleName = `Tool: ${turn.name}`;
      cardBg = "bg-slate-900/10";
      borderStyle = "border-l-2 border-cyan-500";
    }

    const messageId = `turn-${turn.turn_number}`;
    if (document.getElementById(messageId)) return;

    const msgDiv = document.createElement("div");
    msgDiv.id = messageId;
    msgDiv.className = `flex space-x-4 items-start p-4 rounded-lg transition duration-150 group ${cardBg} ${borderStyle}`;

    let rollbackButton = "";
    if (turn.turn_number > 0) {
      const canEdit = turn.role === "user" || turn.role === "assistant";
      rollbackButton = `
        <div class="opacity-0 group-hover:opacity-100 transition duration-150 flex items-center space-x-1.5">
          ${canEdit ? `
          <button onclick="enableCardEdit(event, ${turn.turn_number})" class="text-[10px] bg-slate-800 hover:bg-slate-700 border border-[#334155] text-slate-300 font-mono px-2 py-0.5 rounded transition" title="Edit and Fork Conversation">
            <i class="fa-solid fa-pen text-[9px] mr-1"></i> Edit &amp; Fork
          </button>` : ""}
          <button onclick="triggerFork(${turn.turn_number})" class="text-[10px] bg-red-950/40 hover:bg-red-900 border border-red-800 text-red-400 font-mono px-2 py-0.5 rounded transition">
            <i class="fa-solid fa-code-fork mr-1"></i> Rollback / Branch
          </button>
        </div>`;
    }

    let textHtml = "";
    if (turn.role === "assistant" && turn.tool_calls) {
      textHtml += `<div id="turn-text-${turn.turn_number}" class="text-slate-300 text-sm font-sans mb-2">${turn.content || "Calling tools..."}</div>`;
      turn.tool_calls.forEach((tc) => {
        const title = toolCallTitle(tc);
        textHtml += `
          <details class="tool-call-group mt-2 group/details bg-slate-950 border border-[#334155] rounded-md overflow-hidden">
            <summary class="cursor-pointer select-none px-3 py-1.5 text-cyan-400 font-bold font-mono text-[11px] hover:bg-slate-900/60 flex items-center">
              <i class="fa-solid fa-chevron-right fa-2xs mr-2 transition-transform details-arrow"></i>
              <i class="fa-solid fa-code mr-2"></i>
              <span class="truncate">${escapeHtml(title)}</span>
            </summary>
            <div class="px-3 pb-2 pt-1 border-t border-slate-800/70">
              <div class="text-[9px] uppercase tracking-wider text-slate-500 mb-1">Arguments</div>
              ${renderToolCallArgumentsBody(tc)}
            </div>
          </details>`;
      });
    } else if (turn.role === "tool") {
      const full = turn.content || "";
      const title = toolResultTitle(turn.name, full);
      const deliverables = deliverablePathsFromTool(turn);
      textHtml = `
        <details class="tool-result-group bg-slate-950 border border-slate-800 rounded-md overflow-hidden">
          <summary class="cursor-pointer select-none px-3 py-1.5 text-emerald-400 font-mono text-[11px] hover:bg-slate-900/60 flex items-center">
            <i class="fa-solid fa-chevron-right fa-2xs mr-2 transition-transform details-arrow"></i>
            <i class="fa-solid fa-terminal mr-2"></i>
            <span class="truncate">${escapeHtml(title)}</span>
          </summary>
          <div class="typed-tool-body">${renderToolBody(turn.name, full)}${renderDeliverableChips(deliverables)}</div>
        </details>`;
    } else {
      textHtml = `<div id="turn-text-${turn.turn_number}" class="text-slate-300 text-sm whitespace-pre-wrap font-sans">${turn.content}</div>`;
    }

    let metricsHtml = "";
    if (turn.role === "assistant") {
      metricsHtml = `
        <div class="mt-2 text-[10px] text-slate-500 font-mono">
          <button onclick="toggleCardMetrics(event, ${turn.turn_number})" class="text-slate-400 hover:text-white bg-slate-800/40 hover:bg-slate-800 border border-[#334155]/60 px-2 py-0.5 rounded transition">
            <i class="fa-solid fa-microscope mr-1 text-[9px]"></i> Inspect Execution Metrics
          </button>
          <div id="card-metrics-${turn.turn_number}" class="hidden mt-1.5 p-2.5 bg-slate-950/40 rounded border border-slate-800/80 space-y-1">
            <div class="grid grid-cols-2 gap-2 text-[9px] text-slate-400">
              <div><span class="text-slate-500">Node Target:</span> <span class="text-cyan-400 font-bold">agent</span></div>
              <div><span class="text-slate-500">Latency:</span> <span class="text-emerald-400 font-bold">Active (Online Streaming)</span></div>
              <div><span class="text-slate-500">Tokens In/Out:</span> <span class="text-indigo-400 font-bold" id="metric-tokens-${turn.turn_number}">Calculating...</span></div>
              <div><span class="text-slate-500">USD Spent:</span> <span class="text-amber-400 font-bold" id="metric-cost-${turn.turn_number}">Metering...</span></div>
            </div>
          </div>
        </div>`;
    }

    msgDiv.innerHTML = `
      <div class="w-8 h-8 rounded ${avatarBg} text-white flex items-center justify-center font-bold text-sm shrink-0 shadow-md">
        ${avatarChar}
      </div>
      <div class="space-y-1 flex-1 min-w-0">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <span class="font-bold text-slate-200 text-xs">${roleName}</span>
            <span class="text-[10px] text-slate-500 font-mono">#${turn.turn_number}</span>
          </div>
          ${rollbackButton}
        </div>
        <div id="turn-body-${turn.turn_number}" class="space-y-2">
          ${textHtml}
        </div>
        ${metricsHtml}
      </div>`;

    chatContainer.appendChild(msgDiv);
    if (turn.role === "tool") {
      const summary = msgDiv.querySelector(".tool-result-group > summary");
      if (summary) {
        summary.addEventListener("click", () => {
          state.currentToolPreview = { role: "tool", name: turn.name || "tool", content: turn.content || "", meta: turn.meta || null };
          if (actions.switchDetailsTab) actions.switchDetailsTab("tool");
        });
      }
    }
    chatContainer.scrollTop = chatContainer.scrollHeight;
  }

  function appendSystemAlert(title, message, iconClass) {
    const chatContainer = document.getElementById("chat-messages");
    if (!chatContainer) return;
    removeEmptyHero();
    const alertDiv = document.createElement("div");
    alertDiv.className = "flex space-x-3 items-center bg-indigo-950/20 border border-indigo-900/60 rounded-lg p-4 max-w-4xl mx-auto text-xs text-indigo-300 font-mono";
    alertDiv.innerHTML = `
      <i class="fa-solid ${iconClass} text-lg shrink-0"></i>
      <div class="flex-1">
        <div class="font-bold uppercase tracking-wider">${title}</div>
        <div class="mt-0.5 text-indigo-400/80">${message}</div>
      </div>`;
    chatContainer.appendChild(alertDiv);
    chatContainer.scrollTop = chatContainer.scrollHeight;
  }

  function subAgentCardId(data) {
    return "subagents-" + (data.parent_session || "root");
  }

  function ensureSubAgentCard(data) {
    const chat = document.getElementById("chat-messages");
    if (!chat) return null;
    removeEmptyHero();
    let card = document.getElementById(subAgentCardId(data));
    if (card) return card;
    card = document.createElement("div");
    card.id = subAgentCardId(data);
    card.className = "max-w-4xl mx-auto rounded-lg border border-cyan-500/30 bg-slate-900/40 overflow-hidden";
    card.innerHTML = `
      <div class="flex items-center justify-between px-4 py-2 bg-cyan-950/30 border-b border-cyan-500/20">
        <div class="flex items-center gap-2 text-cyan-300 font-bold text-xs uppercase tracking-wider">
          <i class="fa-solid fa-diagram-project text-sm"></i>
          <span class="sa-title">Sub-agents</span>
        </div>
        <span class="sa-status text-[10px] font-mono text-cyan-400 flex items-center">
          <i class="fa-solid fa-spinner fa-spin mr-1.5"></i> running
        </span>
      </div>
      <div class="px-4 py-2 sa-rows space-y-1.5"></div>`;
    chat.appendChild(card);
    chat.scrollTop = chat.scrollHeight;
    return card;
  }

  function renderSubAgentRow(data, rowState) {
    const card = ensureSubAgentCard(data);
    if (!card) return;
    const rows = card.querySelector(".sa-rows");
    const rowId = "sa-" + (data.session_id || data.description || Date.now());
    let row = document.getElementById(rowId);
    const label = data.description || data.session_id || "sub-agent";
    if (!row) {
      row = document.createElement("div");
      row.id = rowId;
      row.className = "flex items-center gap-2 text-[11px]";
      rows.appendChild(row);
    }
    const icon = rowState === "done"
      ? '<i class="fa-solid fa-circle-check text-emerald-400"></i>'
      : '<i class="fa-solid fa-spinner fa-spin text-cyan-400"></i>';
    const dur = data.duration_ms ? ` · ${data.duration_ms} ms` : "";
    row.innerHTML = `${icon}<span class="font-mono text-slate-300 truncate">${escapeHtml(label)}</span><span class="text-slate-500 text-[10px]">${dur}</span>`;
    if (rowState === "done") {
      const pending = rows.querySelectorAll(".fa-spinner").length;
      if (pending === 0) {
        const st = card.querySelector(".sa-status");
        st.innerHTML = '<i class="fa-solid fa-circle-check mr-1.5 text-emerald-400"></i> completed';
        st.classList.remove("text-cyan-400");
        st.classList.add("text-emerald-400");
      }
    }
    const chat = document.getElementById("chat-messages");
    if (chat) chat.scrollTop = chat.scrollHeight;
  }

  function renderWorkflowTrace(data) {
    const chatContainer = document.getElementById("chat-messages");
    if (!chatContainer) return;
    removeEmptyHero();
    const cardId = `wf-trace-${data.run_id}`;
    if (document.getElementById(cardId)) return;

    const nodes = data.nodes || [];
    let rows = "";
    nodes.forEach(n => {
      const meta = WF_NODE_META[n.type] || { icon: "fa-circle-node", accent: "text-slate-400" };
      const label = n.label ? `<span class="text-slate-500 font-mono ml-1">[${n.label}]</span>` : "";
      rows += `
        <div id="wf-node-${data.run_id}-${n.id}" class="wf-node-row flex items-start space-x-2 py-1.5 border-b border-slate-800/50 last:border-0" data-node="${n.id}">
          <span class="wf-node-status mt-0.5 text-slate-600"><i class="fa-regular fa-circle text-[10px]"></i></span>
          <span class="${meta.accent} mt-0.5"><i class="fa-solid ${meta.icon} text-[11px]"></i></span>
          <div class="flex-1 min-w-0">
            <div class="text-[11px] text-slate-300 font-mono truncate">${n.id}${label}</div>
            <div class="wf-node-detail text-[10px] text-slate-500 mt-0.5">queued…</div>
          </div>
        </div>`;
    });

    const card = document.createElement("div");
    card.id = cardId;
    card.className = "max-w-4xl mx-auto rounded-lg border border-purple-500/30 bg-slate-900/40 overflow-hidden";
    card.innerHTML = `
      <div class="flex items-center justify-between px-4 py-2 bg-purple-950/30 border-b border-purple-500/20">
        <div class="flex items-center space-x-2 text-purple-300">
          <i class="fa-solid fa-diagram-project text-sm"></i>
          <span class="font-bold text-xs uppercase tracking-wider">Workflow: ${data.name || data.workflow_id}</span>
        </div>
        <span class="wf-trace-status text-[10px] text-purple-400 font-mono flex items-center">
          <i class="fa-solid fa-spinner fa-spin mr-1.5"></i> running
        </span>
      </div>
      <div class="px-4 py-2">${rows}</div>`;
    chatContainer.appendChild(card);
    chatContainer.scrollTop = chatContainer.scrollHeight;
  }

  function updateWorkflowNode(data) {
    const row = document.getElementById(`wf-node-${data.run_id}-${data.node_id}`);
    if (!row) return;
    const statusEl = row.querySelector(".wf-node-status");
    const detailEl = row.querySelector(".wf-node-detail");

    if (data.status === "running") {
      statusEl.className = "wf-node-status mt-0.5 text-blue-400";
      statusEl.innerHTML = '<i class="fa-solid fa-spinner fa-spin text-[10px]"></i>';
      detailEl.className = "wf-node-detail text-[10px] text-blue-400/80 mt-0.5";
      detailEl.textContent = "running…";
    } else if (data.status === "completed") {
      statusEl.className = "wf-node-status mt-0.5 text-emerald-400";
      statusEl.innerHTML = '<i class="fa-solid fa-circle-check text-[10px]"></i>';
      const dur = data.duration_ms ? ` · ${data.duration_ms} ms` : "";
      const preview = data.preview ? escapeHtml(data.preview) : "";
      detailEl.className = "wf-node-detail text-[10px] text-slate-400 mt-0.5";
      detailEl.innerHTML = `<span class="text-emerald-400">done${dur}</span>` + (preview ? `<button onclick="toggleWfPreview(this)" class="ml-2 text-purple-400 hover:text-purple-300 underline">show output</button><pre class="wf-preview hidden mt-1.5 p-2 bg-slate-950/60 border border-slate-800 rounded text-[10px] text-slate-300 whitespace-pre-wrap break-words max-h-64 overflow-y-auto">${preview}</pre>` : "");
    } else if (data.status === "failed") {
      statusEl.className = "wf-node-status mt-0.5 text-red-400";
      statusEl.innerHTML = '<i class="fa-solid fa-circle-xmark text-[10px]"></i>';
      detailEl.className = "wf-node-detail text-[10px] text-red-400 mt-0.5";
      detailEl.textContent = "failed: " + (data.error || "unknown error");
    }

    const chatContainer = document.getElementById("chat-messages");
    if (chatContainer) chatContainer.scrollTop = chatContainer.scrollHeight;
  }

  function finalizeWorkflowTrace(data) {
    const card = document.getElementById(`wf-trace-${data.run_id}`);
    if (!card) return;
    const statusEl = card.querySelector(".wf-trace-status");
    if (data.status === "completed") {
      statusEl.className = "wf-trace-status text-[10px] text-emerald-400 font-mono flex items-center";
      statusEl.innerHTML = '<i class="fa-solid fa-circle-check mr-1.5"></i> completed';
    } else {
      statusEl.className = "wf-trace-status text-[10px] text-red-400 font-mono flex items-center";
      statusEl.innerHTML = '<i class="fa-solid fa-circle-xmark mr-1.5"></i> failed';
    }
  }

  function toggleWfPreview(btn) {
    const pre = btn.parentElement.querySelector(".wf-preview");
    if (!pre) return;
    if (pre.classList.contains("hidden")) {
      pre.classList.remove("hidden");
      btn.textContent = "hide output";
    } else {
      pre.classList.add("hidden");
      btn.textContent = "show output";
    }
  }

  function submitPrompt(e, queuedMode) {
    if (e) e.preventDefault();
    const input = document.getElementById("prompt-input");
    if (!input) return;
    const prompt = input.value.trim();
    if (!prompt) return;

    if (!isSessionBusy() && state.currentUIState && !state.currentUIState.composer_enabled) {
      runComposerCta();
      return;
    }

    if (prompt === "/workflows" || prompt.startsWith("/workflow")) {
      const parts = prompt.split(/\s+/);
      if (parts[0] === "/workflows") {
        fetch("/workflows.json")
          .then(res => res.json())
          .then(data => {
            const wfs = data.workflows || {};
            const active = data.active_workflow || "";
            let body = `Currently active: **${active}**\n\n`;
            Object.keys(wfs).forEach(id => {
              const marker = id === active ? "▶ " : "  ";
              body += `${marker}\`${id}\` — **${wfs[id].name || id}**\n   _${wfs[id].description || ""}_\n`;
            });
            body += `\n_Switch with_ \`/workflow <id>\` _or use the Workflow dropdown in the status row._`;
            appendSystemAlert("Registered Workflows", body, "fa-diagram-project text-purple-400");
          })
          .catch(err => alert("Failed to list workflows: " + err));
        input.value = "";
        input.style.height = "auto";
        hideTriggerOverlay();
        return;
      }

      const targetId = (parts[1] || "").trim();
      if (!targetId) {
        appendSystemAlert("Workflow Switch", "Usage: `/workflow <id>` — run `/workflows` to list available pipelines.", "fa-circle-info text-amber-400");
        input.value = "";
        input.style.height = "auto";
        hideTriggerOverlay();
        return;
      }

      fetch("/api/workflows/activate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: targetId })
      })
        .then(async res => {
          if (!res.ok) throw new Error(await res.text() || "Activation failed");
          return res.json();
        })
        .then(() => {
          if (actions.loadWorkflowSelector) actions.loadWorkflowSelector();
        })
        .catch(err => appendSystemAlert("Workflow Switch Failed", err.message, "fa-triangle-exclamation text-red-400"));
      input.value = "";
      input.style.height = "auto";
      hideTriggerOverlay();
      return;
    }

    if (maybeHandleSlashCommand(prompt)) {
      input.value = "";
      input.style.height = "auto";
      hideTriggerOverlay();
      return;
    }

    if (isSessionBusy()) {
      queuePromptText(prompt, queuedMode === "follow_up" ? "follow_up" : "steering");
      input.value = "";
      input.style.height = "auto";
      hideTriggerOverlay();
      return;
    }

    input.value = "";
    input.style.height = "auto";
    hideTriggerOverlay();
    dispatchPromptText(prompt).catch(err => {
      state.queueDispatchInFlight = false;
      if (actions.fetchConfig) actions.fetchConfig();
      appendSystemAlert("API Execution Error", err.message || "Failed to connect to the GoHarness execution API. Please ensure the backend is running.", "fa-triangle-exclamation text-red-400");
    });
  }

  function toggleCardMetrics(e, turnNum) {
    if (e) e.stopPropagation();
    const panel = document.getElementById(`card-metrics-${turnNum}`);
    if (panel) panel.classList.toggle("hidden");
  }

  function handleInputKeydown(e) {
    const input = document.getElementById("prompt-input");
    if (!input) return;
    if (e.key === "Escape") hideTriggerOverlay();
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (isSessionBusy()) {
        submitPrompt(null, e.altKey ? "follow_up" : "steering");
      } else {
        submitPrompt();
      }
    } else {
      setTimeout(() => {
        input.style.height = "auto";
        input.style.height = input.scrollHeight + "px";
        updateTriggerOverlay();
      }, 0);
    }
  }

  return {
    persistSessionUIState,
    restoreSessionUIState,
    isSessionBusy,
    queueForCurrentSession,
    stagedContextForCurrentSession,
    queuePromptText,
    removeQueuedMessage,
    editQueuedMessage,
    renderQueuedMessages,
    stageWorkspaceFile,
    removeStagedContext,
    clearStagedContext,
    renderStagedContextStrip,
    buildPromptWithStagedContext,
    renderComposerActionRow,
    renderComposerTakeover,
    updateTriggerOverlay,
    hideTriggerOverlay,
    handleComposerInput,
    dispatchPromptText,
    processQueuedMessages,
    applyThemeChrome,
    removeEmptyHero,
    seedPromptExample,
    runComposerCta,
    handleComposerShellClick,
    setLocalBusyState,
    updateComposerState,
    renderEmptyHero,
    appendGreeting,
    appendTurnToChat,
    appendSystemAlert,
    renderSubAgentRow,
    renderWorkflowTrace,
    updateWorkflowNode,
    finalizeWorkflowTrace,
    toggleWfPreview,
    submitPrompt,
    toggleCardMetrics,
    handleInputKeydown,
  };
}
