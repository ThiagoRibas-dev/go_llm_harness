const WF_NODE_META = {
  llm: { icon: "fa-robot", accent: "text-purple-400" },
  llm_query: { icon: "fa-robot", accent: "text-purple-400" },
  llm_synthesis: { icon: "fa-layer-group", accent: "text-blue-400" },
  bm25_search: { icon: "fa-magnifying-glass", accent: "text-indigo-400" },
  tool_execution: { icon: "fa-terminal", accent: "text-cyan-400" },
  conditional_router: { icon: "fa-route", accent: "text-amber-400" },
};

export function createChatModule({
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

  function toggleCardMetrics(e, turnNum) {
    if (e) e.stopPropagation();
    const panel = document.getElementById(`card-metrics-${turnNum}`);
    if (panel) panel.classList.toggle("hidden");
  }

  return {
    removeEmptyHero,
    seedPromptExample,
    renderEmptyHero,
    appendGreeting,
    appendTurnToChat,
    appendSystemAlert,
    renderSubAgentRow,
    renderWorkflowTrace,
    updateWorkflowNode,
    finalizeWorkflowTrace,
    toggleWfPreview,
    toggleCardMetrics,
  };
}
