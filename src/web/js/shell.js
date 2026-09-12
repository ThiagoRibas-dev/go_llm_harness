import { currentSessionId } from "./state.js";

export function createShellModule({ state, actions, escapeHtml, deliverablePathsFromTool, renderDeliverableChips, renderToolBody }) {
  function toggleSidebar() {
    const sidebar = document.getElementById("sidebar");
    const resizer = document.getElementById("sidebar-resizer");
    state.isSidebarCollapsed = !state.isSidebarCollapsed;
    if (!sidebar) return;
    if (state.isSidebarCollapsed) {
      sidebar.style.width = "0px";
      sidebar.style.opacity = "0";
      sidebar.style.pointerEvents = "none";
      sidebar.style.borderRightWidth = "0px";
      if (resizer) {
        resizer.style.width = "0px";
        resizer.style.pointerEvents = "none";
      }
    } else {
      sidebar.style.width = "320px";
      sidebar.style.opacity = "1";
      sidebar.style.pointerEvents = "auto";
      sidebar.style.borderRightWidth = "1px";
      if (resizer) {
        resizer.style.width = "6px";
        resizer.style.pointerEvents = "auto";
      }
    }
    enforceShellConcession();
  }

  function workflowLabVisible() {
    const lab = document.getElementById("workflow-lab-surface");
    return !!lab && !lab.classList.contains("hidden");
  }

  function switchPrimarySurface(surface) {
    const target = surface === "workflow" ? "workflow" : "console";
    state.currentPrimarySurface = target;
    const consoleSurface = document.getElementById("console-surface");
    const workflowSurface = document.getElementById("workflow-lab-surface");
    const consoleBtn = document.getElementById("view-console-btn");
    const workflowBtn = document.getElementById("view-workflow-btn");
    if (!consoleSurface || !workflowSurface || !consoleBtn || !workflowBtn) return;

    const consoleActive = "px-2 py-1 rounded bg-slate-800 text-slate-100 font-bold";
    const consoleIdle = "px-2 py-1 rounded text-slate-400 hover:text-slate-200";
    const workflowActive = "px-2 py-1 rounded bg-purple-950/50 text-purple-200 font-bold border border-purple-700/60";
    const workflowIdle = "px-2 py-1 rounded text-slate-400 hover:text-slate-200";

    if (target === "workflow") {
      consoleSurface.classList.add("hidden");
      workflowSurface.classList.remove("hidden");
      consoleBtn.className = consoleIdle;
      workflowBtn.className = workflowActive;
      if (actions.loadWorkflowsSchema) actions.loadWorkflowsSchema();
    } else {
      workflowSurface.classList.add("hidden");
      consoleSurface.classList.remove("hidden");
      consoleBtn.className = consoleActive;
      workflowBtn.className = workflowIdle;
    }
    syncRailButtons();
    enforceShellConcession();
  }

  function syncRailButtons() {
    const map = {
      sessions: document.getElementById("rail-sessions-btn"),
      files: document.getElementById("rail-files-btn"),
      snapshots: document.getElementById("rail-history-btn"),
      workflow: document.getElementById("rail-workflow-btn"),
    };
    Object.entries(map).forEach(([key, el]) => {
      if (!el) return;
      const active = (key === state.currentSidebarSection && state.currentPrimarySurface === "console") || (key === "workflow" && state.currentPrimarySurface === "workflow");
      el.classList.toggle("active", active);
    });
    const detailsBtn = document.getElementById("rail-details-btn");
    if (detailsBtn) detailsBtn.classList.toggle("active", state.detailsPanelOpen);
  }

  function activateRailSection(section) {
    if (section === "workflow") {
      switchPrimarySurface("workflow");
      return;
    }
    switchPrimarySurface("console");
    switchSidebarTab(section);
  }

  function switchConversationView(view) {
    state.currentConversationView = view === "trajectory" || view === "subagents" ? view : "chat";
    const views = {
      chat: document.getElementById("chat-view"),
      trajectory: document.getElementById("trajectory-view"),
      subagents: document.getElementById("subagents-view"),
    };
    const buttons = {
      chat: document.getElementById("conversation-tab-chat"),
      trajectory: document.getElementById("conversation-tab-trajectory"),
      subagents: document.getElementById("conversation-tab-subagents"),
    };
    Object.values(views).forEach(el => el && el.classList.add("hidden"));
    Object.values(buttons).forEach(el => el && el.classList.remove("active"));
    if (views[state.currentConversationView]) views[state.currentConversationView].classList.remove("hidden");
    if (buttons[state.currentConversationView]) buttons[state.currentConversationView].classList.add("active");
    if (state.currentConversationView === "trajectory") switchDetailsTab("trajectory");
    if (state.currentConversationView === "subagents") renderSubagentsView();
  }

  function detailsEmptyState(title, body) {
    return `<div class="rounded-lg border border-[#334155] bg-slate-900/25 p-4"><div class="font-bold text-slate-200">${escapeHtml(title)}</div><div class="mt-1 text-[11px] leading-relaxed text-slate-400">${escapeHtml(body)}</div></div>`;
  }

  function recordTrajectoryEvent(kind, title, detail) {
    state.trajectoryEvents.unshift({
      at: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" }),
      kind,
      title,
      detail,
    });
    if (state.trajectoryEvents.length > 80) state.trajectoryEvents.length = 80;
    renderTrajectoryPanels();
  }

  function rememberDeliverable(path) {
    if (!path) return;
    if (!state.deliverableRegistry.includes(path)) state.deliverableRegistry.unshift(path);
    if (state.deliverableRegistry.length > 20) state.deliverableRegistry.length = 20;
    renderDeliverablesPanel();
  }

  function extractDeliverablesFromTool(turn) {
    if (!turn || turn.role !== "tool") return;
    deliverablePathsFromTool(turn).forEach(rememberDeliverable);
  }

  function updateSubagentRegistry(data, stateName) {
    const key = data.session_id || data.description || String(Date.now());
    state.subagentRegistry[key] = {
      label: data.description || data.session_id || "sub-agent",
      state: stateName,
      duration: data.duration_ms || 0,
    };
    renderSubagentsView();
  }

  function renderTrajectoryPanels() {
    const html = state.trajectoryEvents.length === 0
      ? detailsEmptyState("No trajectory events yet", "Run the agent, tools, or workflow lab to populate a live event ledger.")
      : state.trajectoryEvents.map(ev => `<div class="rounded-lg border border-[#334155] bg-slate-900/25 p-3"><div class="flex items-center gap-2 text-[11px]"><span class="font-mono text-slate-500">${escapeHtml(ev.at)}</span><span class="font-bold text-slate-200">${escapeHtml(ev.title)}</span></div><div class="mt-1 text-[11px] text-slate-400">${escapeHtml(ev.detail || "")}</div><div class="mt-2 text-[10px] uppercase tracking-wider text-slate-500">${escapeHtml(ev.kind)}</div></div>`).join("");
    const detail = document.getElementById("details-tab-trajectory");
    const view = document.getElementById("trajectory-view-body");
    if (detail) detail.innerHTML = html;
    if (view) view.innerHTML = html;
  }

  function renderSubagentsView() {
    const entries = Object.values(state.subagentRegistry);
    const html = entries.length === 0
      ? detailsEmptyState("No sub-agent activity yet", "Spawned sub-agents and background workers will appear here.")
      : entries.map(sa => `<div class="rounded-lg border border-cyan-900/40 bg-slate-900/25 p-3 flex items-center justify-between gap-3"><div><div class="font-mono text-slate-200">${escapeHtml(sa.label)}</div><div class="text-[11px] text-slate-500">${sa.state === "done" ? "Completed" : "Running"}</div></div><div class="text-[11px] font-mono ${sa.state === "done" ? "text-emerald-400" : "text-cyan-400"}">${sa.duration ? escapeHtml(String(sa.duration)) + " ms" : "live"}</div></div>`).join("");
    const view = document.getElementById("subagents-view-body");
    if (view) view.innerHTML = html;
  }

  function renderFilePanel() {
    const panel = document.getElementById("details-tab-file");
    if (!panel) return;
    if (!state.currentFilePreview) {
      panel.innerHTML = detailsEmptyState("No file selected", "Choose Files in the rail, then click a file row to preview it here.");
      return;
    }
    panel.innerHTML = `<div class="rounded-lg border border-[#334155] bg-slate-900/25 overflow-hidden"><div class="px-3 py-2 border-b border-[#334155]/60 flex items-center justify-between gap-2"><span class="font-mono text-xs text-slate-200 truncate">${escapeHtml(state.currentFilePreview.path)}</span><div class="flex items-center gap-2"><button type="button" data-stage-workspace-file="${escapeHtml(state.currentFilePreview.path)}" class="text-[10px] text-cyan-400 hover:text-cyan-300">Stage for next prompt</button><button type="button" data-switch-sidebar-tab="files" class="text-[10px] text-blue-400 hover:text-blue-300">Back to files</button></div></div><pre class="max-h-[420px] overflow-auto p-3 text-[11px] leading-relaxed text-slate-300 font-mono whitespace-pre-wrap">${escapeHtml(state.currentFilePreview.content)}</pre></div>`;
  }

  function renderToolPanel() {
    const panel = document.getElementById("details-tab-tool");
    if (!panel) return;
    if (!state.currentToolPreview) {
      panel.innerHTML = detailsEmptyState("No tool selected", "Open or expand a tool result in the transcript to inspect it here.");
      return;
    }
    const paths = deliverablePathsFromTool({ role: "tool", name: state.currentToolPreview.name, content: state.currentToolPreview.content });
    panel.innerHTML = `<div class="rounded-lg border border-[#334155] bg-slate-900/25 overflow-hidden"><div class="px-3 py-2 border-b border-[#334155]/60"><span class="font-mono text-xs text-slate-200">${escapeHtml(state.currentToolPreview.name || "tool")}</span></div><div class="typed-tool-body">${renderToolBody(state.currentToolPreview.name, state.currentToolPreview.content || "")}${renderDeliverableChips(paths)}</div></div>`;
  }

  function renderDeliverablesPanel() {
    const panel = document.getElementById("details-tab-deliverable");
    if (!panel) return;
    if (state.deliverableRegistry.length === 0) {
      panel.innerHTML = detailsEmptyState("No deliverables yet", "Files written or patched by successful tool calls will be listed here.");
      return;
    }
    panel.innerHTML = state.deliverableRegistry.map(path => `<div class="rounded-lg border border-[#334155] bg-slate-900/25 p-3"><div class="font-mono text-xs text-slate-200 truncate">${escapeHtml(path)}</div><div class="mt-2 flex items-center gap-3 text-[11px]"><button type="button" data-open-workspace-file="${escapeHtml(path)}" class="text-blue-400 hover:text-blue-300">Open preview</button><button type="button" data-stage-workspace-file="${escapeHtml(path)}" class="text-cyan-400 hover:text-cyan-300">Stage for next prompt</button></div></div>`).join("");
  }

  function renderHistoryPanel() {
    const panel = document.getElementById("details-tab-history");
    if (!panel) return;
    panel.innerHTML = `
      <div class="rounded-lg border border-[#334155] bg-slate-900/25 p-4 space-y-3">
        <div>
          <div class="font-bold text-slate-200">History utilities</div>
          <div class="mt-1 text-[11px] leading-relaxed text-slate-400">Open snapshots or run a manual compaction.</div>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" data-activate-rail-section="snapshots" class="px-3 py-1.5 rounded border border-[#334155] hover:bg-slate-800 text-slate-300 text-xs font-bold">Open Snapshots</button>
          <button type="button" data-trigger-compaction="1" class="px-3 py-1.5 rounded border border-indigo-900 bg-indigo-950/40 text-indigo-300 text-xs font-bold">Run Compaction</button>
        </div>
        <div class="text-[10px] font-mono text-slate-500">Session: ${escapeHtml(currentSessionId() || "loading")}</div>
      </div>`;
  }

  function switchDetailsTab(tab) {
    state.currentDetailsTab = ["trajectory", "file", "tool", "deliverable", "history"].includes(tab) ? tab : "trajectory";
    const tabs = ["trajectory", "file", "tool", "deliverable", "history"];
    const subtitles = {
      trajectory: "Live session and workflow event ledger",
      file: "Preview a selected workspace file",
      tool: "Inspect the currently selected tool output",
      deliverable: "Files produced by successful mutations",
      history: "Snapshots, compaction, and recovery utilities",
    };
    tabs.forEach(name => {
      const body = document.getElementById(`details-tab-${name}`);
      const btn = document.getElementById(`details-tab-${name}-btn`);
      if (body) body.classList.toggle("hidden", name !== state.currentDetailsTab);
      if (btn) {
        btn.className = name === state.currentDetailsTab
          ? "px-2.5 py-1 rounded bg-slate-800 text-white font-bold"
          : "px-2.5 py-1 rounded text-slate-400 hover:text-slate-200";
      }
    });
    const subtitle = document.getElementById("details-subtitle");
    if (subtitle) subtitle.textContent = subtitles[state.currentDetailsTab];
    if (!state.detailsPanelOpen) toggleDetailsPanel(true);
    renderTrajectoryPanels();
    renderSubagentsView();
    renderFilePanel();
    renderToolPanel();
    renderDeliverablesPanel();
    renderHistoryPanel();
    syncRailButtons();
  }

  function toggleDetailsPanel(forceOpen) {
    if (typeof forceOpen === "boolean") state.detailsPanelOpen = forceOpen;
    else state.detailsPanelOpen = !state.detailsPanelOpen;
    const panel = document.getElementById("details-panel");
    const resizer = document.getElementById("details-resizer");
    if (!panel || !resizer) return;
    panel.dataset.open = String(state.detailsPanelOpen);
    resizer.dataset.open = String(state.detailsPanelOpen);
    if (state.detailsPanelOpen) panel.style.width = state.detailsPanelWidth + "px";
    syncRailButtons();
  }

  function enforceShellConcession() {
    const minConversation = 760;
    const occupied = 56 + (state.isSidebarCollapsed ? 0 : 320) + 6 + 6 + state.detailsPanelWidth;
    if (window.innerWidth < occupied + minConversation && state.detailsPanelOpen) {
      toggleDetailsPanel(false);
    }
  }

  function initShellResizers() {
    const sidebarResizer = document.getElementById("sidebar-resizer");
    const detailsResizer = document.getElementById("details-resizer");
    const sidebar = document.getElementById("sidebar");
    const details = document.getElementById("details-panel");

    if (sidebarResizer && sidebar) {
      sidebarResizer.addEventListener("mousedown", (e) => {
        if (state.isSidebarCollapsed) return;
        e.preventDefault();
        sidebarResizer.classList.add("dragging");
        const startX = e.clientX;
        const startW = sidebar.getBoundingClientRect().width;
        const onMove = (ev) => {
          const next = Math.max(240, Math.min(460, startW + (ev.clientX - startX)));
          sidebar.style.width = next + "px";
        };
        const onUp = () => {
          sidebarResizer.classList.remove("dragging");
          document.removeEventListener("mousemove", onMove);
          document.removeEventListener("mouseup", onUp);
          enforceShellConcession();
        };
        document.addEventListener("mousemove", onMove);
        document.addEventListener("mouseup", onUp);
      });
    }

    if (detailsResizer && details) {
      detailsResizer.addEventListener("mousedown", (e) => {
        if (!state.detailsPanelOpen) return;
        e.preventDefault();
        detailsResizer.classList.add("dragging");
        const startX = e.clientX;
        const startW = details.getBoundingClientRect().width;
        const onMove = (ev) => {
          const next = Math.max(220, Math.min(520, startW - (ev.clientX - startX)));
          state.detailsPanelWidth = next;
          details.style.width = next + "px";
        };
        const onUp = () => {
          detailsResizer.classList.remove("dragging");
          document.removeEventListener("mousemove", onMove);
          document.removeEventListener("mouseup", onUp);
          if (state.detailsPanelWidth < 240) toggleDetailsPanel(false);
          enforceShellConcession();
        };
        document.addEventListener("mousemove", onMove);
        document.addEventListener("mouseup", onUp);
      });
    }

    window.addEventListener("resize", enforceShellConcession);
  }

  async function openWorkspaceFile(path) {
    if (!path) return;
    try {
      const res = await fetch("/api/workspace/file?path=" + encodeURIComponent(path));
      if (!res.ok) throw new Error("Failed to read file");
      const data = await res.json();
      state.currentFilePreview = data;
      switchDetailsTab("file");
    } catch (err) {
      state.currentFilePreview = { path, content: "Failed to read file preview: " + err.message };
      switchDetailsTab("file");
    }
  }

  function switchSidebarTab(tabName) {
    state.currentSidebarSection = ["files", "snapshots"].includes(tabName) ? tabName : "sessions";
    const filesTab = document.getElementById("tab-files");
    const sessionsTab = document.getElementById("tab-sessions");
    const snapshotsTab = document.getElementById("tab-snapshots");
    const title = document.getElementById("sidebar-surface-title");
    const subtitle = document.getElementById("sidebar-surface-subtitle");
    if (!filesTab || !sessionsTab || !snapshotsTab) return;

    filesTab.classList.add("hidden");
    sessionsTab.classList.add("hidden");
    snapshotsTab.classList.add("hidden");

    if (state.currentSidebarSection === "files") {
      filesTab.classList.remove("hidden");
      if (title) title.textContent = "Files";
      if (subtitle) subtitle.textContent = "Browse workspace files and staged prompt context";
      if (actions.refreshWorkspaceTree) actions.refreshWorkspaceTree();
    } else if (state.currentSidebarSection === "snapshots") {
      snapshotsTab.classList.remove("hidden");
      if (title) title.textContent = "History Utilities";
      if (subtitle) subtitle.textContent = "Snapshots and workspace recovery tools";
      if (actions.fetchSnapshots) actions.fetchSnapshots();
      switchDetailsTab("history");
    } else {
      sessionsTab.classList.remove("hidden");
      if (title) title.textContent = "Workspaces & Sessions";
      if (subtitle) subtitle.textContent = "Navigate local workspaces and session threads";
      if (actions.fetchSessions) actions.fetchSessions();
      if (actions.fetchWorkspaces) actions.fetchWorkspaces();
    }
    syncRailButtons();
  }

  let shellBindingsInitialized = false;
  function initShellBindings() {
    if (shellBindingsInitialized) return;
    shellBindingsInitialized = true;

    document.getElementById("toggle-sidebar-btn")?.addEventListener("click", toggleSidebar);
    document.getElementById("header-settings-btn")?.addEventListener("click", () => actions.openSettingsModal && actions.openSettingsModal());
    document.getElementById("rail-sessions-btn")?.addEventListener("click", () => activateRailSection("sessions"));
    document.getElementById("rail-files-btn")?.addEventListener("click", () => activateRailSection("files"));
    document.getElementById("rail-history-btn")?.addEventListener("click", () => activateRailSection("snapshots"));
    document.getElementById("rail-workflow-btn")?.addEventListener("click", () => switchPrimarySurface("workflow"));
    document.getElementById("rail-details-btn")?.addEventListener("click", () => toggleDetailsPanel());
    document.getElementById("rail-settings-btn")?.addEventListener("click", () => actions.openSettingsModal && actions.openSettingsModal());
    document.getElementById("manual-compact-btn")?.addEventListener("click", () => actions.triggerCompaction && actions.triggerCompaction());
    document.getElementById("view-console-btn")?.addEventListener("click", () => switchPrimarySurface("console"));
    document.getElementById("view-workflow-btn")?.addEventListener("click", () => switchPrimarySurface("workflow"));
    document.getElementById("conversation-tab-chat")?.addEventListener("click", () => switchConversationView("chat"));
    document.getElementById("conversation-tab-trajectory")?.addEventListener("click", () => switchConversationView("trajectory"));
    document.getElementById("conversation-tab-subagents")?.addEventListener("click", () => switchConversationView("subagents"));
    document.getElementById("workflow-console-back-btn-top")?.addEventListener("click", () => switchPrimarySurface("console"));
    document.getElementById("workflow-settings-btn")?.addEventListener("click", () => actions.openSettingsModal && actions.openSettingsModal());
    document.getElementById("workflow-console-back-btn-bottom")?.addEventListener("click", () => switchPrimarySurface("console"));
    document.getElementById("details-close-btn")?.addEventListener("click", () => toggleDetailsPanel(false));
    document.getElementById("details-tab-trajectory-btn")?.addEventListener("click", () => switchDetailsTab("trajectory"));
    document.getElementById("details-tab-file-btn")?.addEventListener("click", () => switchDetailsTab("file"));
    document.getElementById("details-tab-tool-btn")?.addEventListener("click", () => switchDetailsTab("tool"));
    document.getElementById("details-tab-deliverable-btn")?.addEventListener("click", () => switchDetailsTab("deliverable"));
    document.getElementById("details-tab-history-btn")?.addEventListener("click", () => switchDetailsTab("history"));

    document.getElementById("details-panel")?.addEventListener("click", (event) => {
      const openBtn = event.target.closest("[data-open-workspace-file]");
      if (openBtn) {
        openWorkspaceFile(openBtn.getAttribute("data-open-workspace-file"));
        return;
      }
      const stageBtn = event.target.closest("[data-stage-workspace-file]");
      if (stageBtn) {
        actions.stageWorkspaceFile && actions.stageWorkspaceFile(stageBtn.getAttribute("data-stage-workspace-file"));
        return;
      }
      const tabBtn = event.target.closest("[data-switch-sidebar-tab]");
      if (tabBtn) {
        switchSidebarTab(tabBtn.getAttribute("data-switch-sidebar-tab"));
        return;
      }
      const railBtn = event.target.closest("[data-activate-rail-section]");
      if (railBtn) {
        activateRailSection(railBtn.getAttribute("data-activate-rail-section"));
        return;
      }
      if (event.target.closest("[data-trigger-compaction]")) {
        actions.triggerCompaction && actions.triggerCompaction();
      }
    });
  }

  return {
    toggleSidebar,
    workflowLabVisible,
    switchPrimarySurface,
    syncRailButtons,
    activateRailSection,
    switchConversationView,
    detailsEmptyState,
    recordTrajectoryEvent,
    rememberDeliverable,
    extractDeliverablesFromTool,
    updateSubagentRegistry,
    renderTrajectoryPanels,
    renderSubagentsView,
    renderFilePanel,
    renderToolPanel,
    renderDeliverablesPanel,
    renderHistoryPanel,
    switchDetailsTab,
    toggleDetailsPanel,
    enforceShellConcession,
    initShellResizers,
    initShellBindings,
    openWorkspaceFile,
    switchSidebarTab,
  };
}
