import { DEFAULT_PROMPT_PLACEHOLDER, currentSessionKey, storageKey } from "./state.js";

const SLASH_COMMAND_SUGGESTIONS = [
  { label: "/workflows", insert: "/workflows", description: "List registered workflows" },
  { label: "/workflow ", insert: "/workflow ", description: "Switch active runtime workflow" },
  { label: "/compact", insert: "/compact", description: "Run manual context compaction" },
  { label: "/new", insert: "/new", description: "Start a new session" },
  { label: "/settings", insert: "/settings", description: "Open Settings" },
  { label: "/workflow-lab", insert: "/workflow-lab", description: "Open Workflow Lab" },
];

export function createComposerModule({ state, actions, escapeHtml }) {
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
      actions.appendSystemAlert && actions.appendSystemAlert("Stage Context Failed", "Could not stage file: " + err.message, "fa-triangle-exclamation text-red-400");
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
      actions.appendSystemAlert && actions.appendSystemAlert("Compaction Requested", "Queued a manual context compaction in the background.", "fa-compress text-indigo-400");
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
    actions.appendSystemAlert && actions.appendSystemAlert("Shell Command Completed", `Executed sandboxed command: ${command}`, "fa-terminal text-cyan-400");
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
    actions.appendTurnToChat && actions.appendTurnToChat({ role: "user", content: text, turn_number: state.turnCounter });
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
      actions.appendSystemAlert && actions.appendSystemAlert("Queued Message Failed", err.message || String(err), "fa-triangle-exclamation text-red-400");
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
    actions.renderEmptyHero && actions.renderEmptyHero();
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
            actions.appendSystemAlert && actions.appendSystemAlert("Registered Workflows", body, "fa-diagram-project text-purple-400");
          })
          .catch(err => alert("Failed to list workflows: " + err));
        input.value = "";
        input.style.height = "auto";
        hideTriggerOverlay();
        return;
      }

      const targetId = (parts[1] || "").trim();
      if (!targetId) {
        actions.appendSystemAlert && actions.appendSystemAlert("Workflow Switch", "Usage: `/workflow <id>` — run `/workflows` to list available pipelines.", "fa-circle-info text-amber-400");
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
        .catch(err => actions.appendSystemAlert && actions.appendSystemAlert("Workflow Switch Failed", err.message, "fa-triangle-exclamation text-red-400"));
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
      actions.appendSystemAlert && actions.appendSystemAlert("API Execution Error", err.message || "Failed to connect to the GoHarness execution API. Please ensure the backend is running.", "fa-triangle-exclamation text-red-400");
    });
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
    runComposerCta,
    handleComposerShellClick,
    setLocalBusyState,
    updateComposerState,
    submitPrompt,
    handleInputKeydown,
  };
}
