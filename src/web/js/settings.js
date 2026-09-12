export function createSettingsModule({ state, actions, escapeHtml, populateKnownModelList }) {
  function suggestBaseURL(provider) {
    const urlInput = document.getElementById("input-base-url");
    const modelInput = document.getElementById("input-model");
    const vFields = document.getElementById("vertex-fields");
    populateKnownModelList("known-models-chat", provider);
    if (!urlInput || !modelInput || !vFields) return;

    if (provider === "vertex") {
      vFields.classList.remove("hidden");
      urlInput.value = "";
      modelInput.value = "gemini-3.1-flash-lite";
      return;
    }

    vFields.classList.add("hidden");
    if (provider === "anthropic") {
      urlInput.value = "https://api.anthropic.com/v1/messages";
      modelInput.value = "claude-3-5-sonnet-latest";
    } else if (provider === "gemini") {
      urlInput.value = "";
      modelInput.value = "gemini-1.5-flash";
    } else {
      urlInput.value = "https://api.openai.com/v1/chat/completions";
      modelInput.value = "gpt-4o";
    }
  }

  function renderSettingsConnectionsSummary(cfg) {
    const host = document.getElementById("settings-connections-summary");
    if (!host) return;
    const activeProfile = cfg.provider_profile ? `@${cfg.provider_profile}` : "inline";
    const compProfile = cfg.compaction && cfg.compaction.provider_profile ? `@${cfg.compaction.provider_profile}` : "inline";
    host.innerHTML = `
      <div class="rounded border border-[#334155]/60 bg-slate-950/30 p-3">
        <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-1">Active chat</div>
        <div class="font-mono text-slate-200 text-xs">${escapeHtml(activeProfile)}</div>
        <div class="mt-1 text-[11px] text-slate-400">${escapeHtml((cfg.api.provider || "openai").toUpperCase())} · ${escapeHtml(cfg.api.model || "unset")}</div>
      </div>
      <div class="rounded border border-[#334155]/60 bg-slate-950/30 p-3">
        <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-1">Compaction</div>
        <div class="font-mono text-slate-200 text-xs">${escapeHtml(compProfile)}</div>
        <div class="mt-1 text-[11px] text-slate-400">${escapeHtml(((cfg.compaction && cfg.compaction.provider) || "openai").toUpperCase())} · ${escapeHtml((cfg.compaction && cfg.compaction.model) || "unset")}</div>
      </div>`;
  }

  function openSettingsModal() {
    fetch("/api/config")
      .then(res => res.json())
      .then(cfg => {
        state.currentSettingsRevision = cfg.settings_revision || 0;
        const provider = cfg.api.provider || "openai";
        document.getElementById("input-provider").value = provider;
        document.getElementById("input-api-key").value = cfg.api.key || "";
        document.getElementById("input-model").value = cfg.api.model || "gpt-4o";
        document.getElementById("input-base-url").value = cfg.api.base_url || "";
        document.getElementById("input-sandbox-mode").value = cfg.security.sandbox_mode || "host";
        document.getElementById("input-sandbox-fallback").value = String(cfg.security.sandbox_fallback || false);
        document.getElementById("input-max-turns").value = cfg.agent.max_turns || 15;
        document.getElementById("input-target-scan-dirs").value = (cfg.agent.target_scan_dirs || []).join(", ");

        if (cfg.compaction) {
          const compProvider = cfg.compaction.provider || "openai";
          document.getElementById("input-compact-provider").value = compProvider;
          document.getElementById("input-compact-api-key").value = cfg.compaction.key || "";
          document.getElementById("input-compact-base-url").value = cfg.compaction.base_url || "";
          document.getElementById("input-compact-model").value = cfg.compaction.model || "gpt-4o-mini";
          document.getElementById("input-compact-temp").value = cfg.compaction.temperature !== undefined ? cfg.compaction.temperature : 0.2;
          document.getElementById("input-compact-project-id").value = cfg.compaction.project_id || "";
          document.getElementById("input-compact-region").value = cfg.compaction.region || "";
          const compVFields = document.getElementById("compact-vertex-fields");
          if (compVFields) compVFields.classList.toggle("hidden", compProvider !== "vertex");
          document.getElementById("input-compact-turns").value = cfg.compaction.auto_compact_turns || 6;
          document.getElementById("input-compact-keep-n").value = cfg.compaction.keep_last_n || 2;
          document.getElementById("input-compact-prompt").value = cfg.compaction.system_prompt || "";
        }

        document.getElementById("input-temperature").value = cfg.api.temperature !== undefined ? cfg.api.temperature : 0.0;
        document.getElementById("input-top-p").value = cfg.api.top_p !== undefined ? cfg.api.top_p : 0.95;
        document.getElementById("input-top-k").value = cfg.api.top_k !== undefined ? cfg.api.top_k : 40;
        document.getElementById("input-thinking").value = cfg.api.thinking_level || "off";
        document.getElementById("input-project-id").value = cfg.api.project_id || "";
        document.getElementById("input-region").value = cfg.api.region || "";
        document.getElementById("input-debug").checked = cfg.debug || false;

        const vFields = document.getElementById("vertex-fields");
        if (vFields) vFields.classList.toggle("hidden", provider !== "vertex");
        populateKnownModelList("known-models-chat", provider);
        populateKnownModelList("known-models-compact", (cfg.compaction && cfg.compaction.provider) || "openai");

        renderSettingsConnectionsSummary(cfg);

        fetch("/api/config/exclusions")
          .then(res => res.json())
          .then(ex => {
            state.currentIgnoredPatterns = ex.ignored_patterns || [];
            state.currentCollapsedPatterns = ex.collapsed_patterns || [];
            renderExclusionChips();
          });

        fetchMCPServers();
        populateProfileSelectors(cfg.provider_profile, cfg.compaction ? cfg.compaction.provider_profile : "");
        document.getElementById("settings-modal")?.classList.remove("hidden");
      })
      .catch(err => alert("Failed to fetch settings: " + err));
  }

  function closeSettingsModal() {
    document.getElementById("settings-modal")?.classList.add("hidden");
  }

  function saveSettings(e) {
    e.preventDefault();
    const payload = {
      provider: document.getElementById("input-provider").value,
      provider_profile: document.getElementById("input-provider-profile").value,
      api_key: document.getElementById("input-api-key").value.trim(),
      model: document.getElementById("input-model").value.trim(),
      base_url: document.getElementById("input-base-url").value.trim(),
      sandbox_mode: document.getElementById("input-sandbox-mode").value,
      sandbox_fallback: document.getElementById("input-sandbox-fallback").value === "true",
      max_turns: parseInt(document.getElementById("input-max-turns").value, 10) || 15,
      target_scan_dirs: document.getElementById("input-target-scan-dirs").value.split(",").map(s => s.trim()).filter(Boolean),
      compact_profile: document.getElementById("input-compact-profile").value,
      compact_provider: document.getElementById("input-compact-provider").value,
      compact_api_key: document.getElementById("input-compact-api-key").value.trim(),
      compact_base_url: document.getElementById("input-compact-base-url").value.trim(),
      compact_model: document.getElementById("input-compact-model").value.trim() || "gpt-4o-mini",
      compact_temp: parseFloat(document.getElementById("input-compact-temp").value) || 0.2,
      compact_project_id: document.getElementById("input-compact-project-id").value.trim(),
      compact_region: document.getElementById("input-compact-region").value.trim(),
      compact_turns: parseInt(document.getElementById("input-compact-turns").value, 10) || 6,
      compact_keep_n: parseInt(document.getElementById("input-compact-keep-n").value, 10) || 2,
      compact_prompt: document.getElementById("input-compact-prompt").value.trim(),
      temperature: parseFloat(document.getElementById("input-temperature").value) || 0.0,
      top_p: parseFloat(document.getElementById("input-top-p").value) || 0.95,
      top_k: parseInt(document.getElementById("input-top-k").value, 10) || 40,
      thinking_level: document.getElementById("input-thinking").value,
      project_id: document.getElementById("input-project-id").value.trim(),
      region: document.getElementById("input-region").value.trim(),
      debug: document.getElementById("input-debug").checked,
      expected_revision: state.currentSettingsRevision,
    };

    fetch("/api/config/exclusions/save", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ignored_patterns: state.currentIgnoredPatterns,
        collapsed_patterns: state.currentCollapsedPatterns,
      }),
    })
      .then(() => fetch("/api/config/save", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      }))
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "Failed to save config");
        return res.json();
      })
      .then(data => {
        state.currentSettingsRevision = data.settings_revision || state.currentSettingsRevision;
        closeSettingsModal();
        fetchConfig();
        actions.appendSystemAlert && actions.appendSystemAlert("System Settings Saved", `Configurations and scan exclusions successfully updated. Now running on provider '${payload.provider.toUpperCase()}' using model '${payload.model}'.`, "fa-check-double text-green-400");
        actions.refreshWorkspaceTree && actions.refreshWorkspaceTree();
        actions.fetchWorkspaces && actions.fetchWorkspaces();
      })
      .catch(err => {
        const msg = String(err && err.message ? err.message : err);
        if (msg.includes("settings-conflict")) {
          alert("Settings changed elsewhere. Please reopen Settings, review the latest values, and try again.");
          openSettingsModal();
          return;
        }
        alert("Error saving settings: " + err);
      });
  }

  function fetchConfig() {
    return fetch("/api/config")
      .then(res => res.json())
      .then(cfg => {
        state.activeWorkspaceDir = cfg.agent.workspace_dir;
        window.__cfgProvider = cfg.api.provider || "openai";
        window.__cfgModel = cfg.api.model || "";
        document.getElementById("model-name").innerText = cfg.api.model + " (" + (cfg.api.provider || "openai").toUpperCase() + ")";
        document.getElementById("workspace-path").innerText = "Path: " + cfg.agent.workspace_dir;
        document.getElementById("session-id").innerText = "Session: " + cfg.session_id;
        state.currentUIState = cfg.ui_state || state.currentUIState;
        actions.applyThemeChrome && actions.applyThemeChrome();
        actions.updateComposerState && actions.updateComposerState();
        actions.renderComposerActionRow && actions.renderComposerActionRow();
        actions.renderQueuedMessages && actions.renderQueuedMessages();
        actions.renderStagedContextStrip && actions.renderStagedContextStrip();
        actions.renderComposerTakeover && actions.renderComposerTakeover();
        actions.renderEmptyHero && actions.renderEmptyHero();
        actions.fetchSessions && actions.fetchSessions();
        return cfg;
      })
      .catch(err => console.error("Error fetching config:", err));
  }

  function renderExclusionChips() {
    const ignoreList = document.getElementById("ignore-chips-list");
    const collapseList = document.getElementById("collapse-chips-list");
    if (ignoreList) {
      const ignoreHtml = state.currentIgnoredPatterns.map((pat, idx) => `
        <span class="inline-flex items-center space-x-1 px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[10px] border border-slate-700">
          <span>${pat}</span>
          <button type="button" onclick="removeIgnorePattern(${idx})" class="text-slate-500 hover:text-red-400 font-bold">&times;</button>
        </span>`).join("");
      ignoreList.innerHTML = ignoreHtml || '<span class="text-slate-500 italic text-[10px]">No patterns ignored.</span>';
    }
    if (collapseList) {
      const collapseHtml = state.currentCollapsedPatterns.map((pat, idx) => `
        <span class="inline-flex items-center space-x-1 px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[10px] border border-slate-700">
          <span>${pat}</span>
          <button type="button" onclick="removeCollapsePattern(${idx})" class="text-slate-500 hover:text-red-400 font-bold">&times;</button>
        </span>`).join("");
      collapseList.innerHTML = collapseHtml || '<span class="text-slate-500 italic text-[10px]">No directories collapsed.</span>';
    }
  }

  function addIgnorePattern() {
    const input = document.getElementById("add-ignore-input");
    const val = input ? input.value.trim() : "";
    if (val && !state.currentIgnoredPatterns.includes(val)) {
      state.currentIgnoredPatterns.push(val);
      input.value = "";
      renderExclusionChips();
    }
  }

  function removeIgnorePattern(idx) {
    state.currentIgnoredPatterns.splice(idx, 1);
    renderExclusionChips();
  }

  function addCollapsePattern() {
    const input = document.getElementById("add-collapse-input");
    const val = input ? input.value.trim() : "";
    if (val && !state.currentCollapsedPatterns.includes(val)) {
      state.currentCollapsedPatterns.push(val);
      input.value = "";
      renderExclusionChips();
    }
  }

  function removeCollapsePattern(idx) {
    state.currentCollapsedPatterns.splice(idx, 1);
    renderExclusionChips();
  }

  function fetchSnapshots() {
    const list = document.getElementById("snapshots-list");
    if (!list) return;
    list.innerHTML = '<div class="text-slate-500 italic text-xs"><i class="fa-solid fa-spinner animate-spin mr-1"></i>Loading snapshots...</div>';
    fetch("/api/snapshots")
      .then(res => res.json())
      .then(data => {
        if (!data.snapshots || data.snapshots.length === 0) {
          list.innerHTML = '<div class="text-slate-500 italic text-xs">No snapshots captured yet.</div>';
          return;
        }
        data.snapshots.sort((a, b) => b.timestamp.localeCompare(a.timestamp));
        list.innerHTML = data.snapshots.map(snap => {
          const date = new Date(snap.timestamp).toLocaleString();
          const sizeKB = (snap.total_size / 1024).toFixed(1);
          return `
            <div class="p-3 bg-slate-900/40 border border-[#334155]/60 rounded-lg space-y-2 relative group hover:border-indigo-500/50 transition">
              <div class="pr-12">
                <div class="font-bold text-slate-100 text-xs truncate">${snap.name}</div>
                <div class="text-[9px] text-slate-500 font-mono mt-0.5">${date}</div>
              </div>
              <div class="flex items-center justify-between text-[10px] text-slate-400 font-mono">
                <span>${snap.file_count} file(s)</span>
                <span>${sizeKB} KB</span>
              </div>
              <div class="flex items-center space-x-2 pt-1 border-t border-[#334155]/30">
                <button onclick="revertToSnapshot('${snap.id}')" class="flex-1 py-1 bg-indigo-950/40 text-indigo-400 hover:bg-indigo-900 hover:text-white border border-indigo-900/60 rounded text-[10px] font-bold transition flex items-center justify-center space-x-1">
                  <i class="fa-solid fa-undo"></i> <span>Revert Workspace</span>
                </button>
                <button onclick="deleteSnapshot('${snap.id}')" class="px-2 py-1 bg-red-950/20 text-red-400 hover:bg-red-900 hover:text-white border border-red-900/40 rounded text-[10px] font-bold transition">
                  <i class="fa-solid fa-trash"></i>
                </button>
              </div>
            </div>`;
        }).join("");
      })
      .catch(err => console.error("Error loading snapshots:", err));
  }

  function createSnapshot() {
    const input = document.getElementById("snapshot-name-input");
    const name = (input && input.value.trim()) || "Snapshot " + new Date().toLocaleTimeString();
    fetch("/api/snapshots/create", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name })
    })
      .then(res => {
        if (!res.ok) throw new Error("Failed to create snapshot");
        return res.json();
      })
      .then(() => {
        if (input) input.value = "";
        fetchSnapshots();
        actions.appendSystemAlert && actions.appendSystemAlert("Workspace Captured", `Successfully created a file snapshot of your active workspace directory: '${name}'. You can revert to this state at any time.`, "fa-camera text-indigo-400");
      })
      .catch(err => alert("Failed to create snapshot: " + err));
  }

  function revertToSnapshot(id) {
    if (!confirm("Are you sure you want to revert the active workspace to this snapshot? ALL untracked or unsaved file modifications in the workspace will be permanently overwritten!")) return;
    fetch("/api/snapshots/revert", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ snapshot_id: id })
    })
      .then(res => {
        if (!res.ok) throw new Error("Failed to revert workspace");
        return res.json();
      })
      .then(data => {
        actions.refreshWorkspaceTree && actions.refreshWorkspaceTree();
        fetchSnapshots();
        actions.appendSystemAlert && actions.appendSystemAlert("Workspace Restored", `Active files reverted completely to match snapshot: '${data.name}'.`, "fa-undo text-emerald-400");
      })
      .catch(err => alert("Failed to revert workspace: " + err));
  }

  function deleteSnapshot(id) {
    if (!confirm("Are you sure you want to delete this snapshot from disk?")) return;
    fetch("/api/snapshots/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ snapshot_id: id })
    })
      .then(res => {
        if (!res.ok) throw new Error("Failed to delete snapshot");
        return res.json();
      })
      .then(() => fetchSnapshots())
      .catch(err => alert("Failed to delete snapshot: " + err));
  }

  function fetchMCPServers() {
    const list = document.getElementById("mcp-servers-list");
    if (!list) return;
    list.innerHTML = '<div class="text-slate-500 italic text-[10px]"><i class="fa-solid fa-spinner animate-spin mr-1"></i>Loading MCP servers...</div>';
    fetch("/api/mcp")
      .then(res => res.json())
      .then(data => {
        const keys = Object.keys(data || {});
        if (keys.length === 0) {
          list.innerHTML = '<div class="text-slate-500 italic text-[10px]">No MCP servers registered.</div>';
          return;
        }
        list.innerHTML = keys.map(key => {
          const srv = data[key];
          const argsStr = (srv.args || []).join(" ");
          const stateBadge = srv.running
            ? '<span class="text-[9px] bg-emerald-950/40 text-emerald-300 px-1.5 py-0.5 rounded font-bold">RUNNING</span>'
            : '<span class="text-[9px] bg-amber-950/40 text-amber-300 px-1.5 py-0.5 rounded font-bold">STOPPED</span>';
          return `
            <div class="p-2 rounded bg-slate-900/30 border border-[#334155]/40 text-[10px] font-mono space-y-2">
              <div class="flex items-start justify-between gap-3">
                <div class="truncate flex-1 pr-2">
                  <div class="font-bold text-slate-200 flex items-center gap-2">${key} ${stateBadge}</div>
                  <div class="text-slate-400 mt-0.5 truncate">${srv.command} ${argsStr}</div>
                </div>
                <button onclick="deleteMCPServer('${key}')" class="text-slate-500 hover:text-red-400 p-1 rounded hover:bg-slate-800 transition" title="Delete MCP Server">
                  <i class="fa-solid fa-trash text-[10px]"></i>
                </button>
              </div>
              <div class="flex flex-wrap items-center gap-3 text-[10px] text-slate-500">
                <span>Transport: <span class="text-slate-300">${escapeHtml(srv.transport || "stdio")}</span></span>
                <span>Auth: <span class="text-slate-300">${escapeHtml(srv.auth_state || "unknown")}</span></span>
                ${srv.last_error ? `<span class="text-red-300">Error: ${escapeHtml(srv.last_error)}</span>` : ""}
              </div>
            </div>`;
        }).join("");
      })
      .catch(err => {
        list.innerHTML = `<div class="text-red-400 text-[10px]">Failed to load: ${err}</div>`;
      });
  }

  function addMCPServer() {
    const nameInput = document.getElementById("mcp-name-input");
    const cmdInput = document.getElementById("mcp-cmd-input");
    const argsInput = document.getElementById("mcp-args-input");
    const name = nameInput.value.trim();
    const command = cmdInput.value.trim();
    const argsRaw = argsInput.value.trim();
    if (!name || !command) {
      alert("Please supply both Server Key/Name and Startup Command.");
      return;
    }
    const args = argsRaw ? argsRaw.split(" ").filter(Boolean) : [];
    fetch("/api/mcp/save", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, command, args })
    })
      .then(res => {
        if (!res.ok) throw new Error("Failed to save MCP server");
        return res.json();
      })
      .then(() => {
        nameInput.value = "";
        cmdInput.value = "";
        argsInput.value = "";
        fetchMCPServers();
        actions.appendSystemAlert && actions.appendSystemAlert("MCP Server Added", `Successfully registered and spawned MCP Server '${name}'. Live handshaking initiated.`, "fa-plug text-green-400");
      })
      .catch(err => alert("Error saving MCP server: " + err));
  }

  function deleteMCPServer(name) {
    if (!confirm(`Are you sure you want to permanently delete and stop MCP server '${name}'?`)) return;
    fetch("/api/mcp/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name })
    })
      .then(res => {
        if (!res.ok) throw new Error("Failed to delete MCP server");
        return res.json();
      })
      .then(() => {
        fetchMCPServers();
        actions.appendSystemAlert && actions.appendSystemAlert("MCP Server Deleted", `Successfully stopped and removed MCP Server '${name}'.`, "fa-plug text-amber-400");
      })
      .catch(err => alert("Error deleting MCP server: " + err));
  }

  function suggestCompactBaseURL(provider) {
    const urlInput = document.getElementById("input-compact-base-url");
    const modelInput = document.getElementById("input-compact-model");
    const vFields = document.getElementById("compact-vertex-fields");
    populateKnownModelList("known-models-compact", provider);
    if (!urlInput || !modelInput || !vFields) return;

    if (provider === "vertex") {
      vFields.classList.remove("hidden");
      urlInput.value = "";
      modelInput.value = "gemini-1.5-flash";
      return;
    }

    vFields.classList.add("hidden");
    if (provider === "anthropic") {
      urlInput.value = "https://api.anthropic.com/v1/messages";
      modelInput.value = "claude-3-5-haiku";
    } else if (provider === "gemini") {
      urlInput.value = "";
      modelInput.value = "gemini-1.5-flash";
    } else {
      urlInput.value = "https://api.openai.com/v1/chat/completions";
      modelInput.value = "gpt-4o-mini";
    }
  }

  function switchSettingsTab(tabName) {
    const panels = {
      standard: document.getElementById("settings-panel-standard"),
      providers: document.getElementById("settings-panel-providers"),
    };
    const buttons = {
      standard: document.getElementById("btn-settings-tab-standard"),
      providers: document.getElementById("btn-settings-tab-providers"),
    };
    const stdFooter = document.getElementById("settings-standard-footer");
    if (stdFooter) stdFooter.classList.toggle("hidden", tabName !== "standard");
    Object.keys(panels).forEach(k => panels[k] && panels[k].classList.add("hidden"));
    Object.keys(buttons).forEach(k => {
      if (buttons[k]) buttons[k].className = "font-bold text-slate-400 border-b-2 border-transparent hover:text-slate-200 pb-1 flex items-center text-xs uppercase tracking-wider transition";
    });
    if (panels[tabName]) panels[tabName].classList.remove("hidden");
    if (buttons[tabName]) buttons[tabName].className = "font-bold text-blue-500 border-b-2 border-blue-500 pb-1 flex items-center text-xs uppercase tracking-wider transition";
    if (tabName === "providers") fetchProviders();
  }

  function fetchProviders() {
    state.wfProfiles = null;
    fetch("/api/providers")
      .then(res => res.json())
      .then(data => {
        state.currentProvidersRevision = data.providers_revision || 0;
        const profiles = data.providers || {};
        const names = Object.keys(profiles);
        const chatSel = document.getElementById("active-chat-profile");
        const compSel = document.getElementById("active-compaction-profile");
        [chatSel, compSel].forEach(sel => {
          if (!sel) return;
          sel.innerHTML = '<option value="">— ' + (sel === chatSel ? 'inline config.api' : 'inline compaction') + ' —</option>';
          names.forEach(n => {
            sel.innerHTML += `<option value="${n}">${n} (${profiles[n].provider}/${profiles[n].model})</option>`;
          });
        });
        if (chatSel) chatSel.value = data.active_profile || "";
        if (compSel) compSel.value = data.compaction_profile || "";

        const list = document.getElementById("providers-list");
        if (!list) return;
        if (names.length === 0) {
          list.innerHTML = '<div class="text-slate-500 italic text-xs">No profiles yet. Click "New Profile" to create one.</div>';
          return;
        }
        list.innerHTML = names.map(name => {
          const p = profiles[name];
          const activeBadge = name === data.active_profile ? '<span class="ml-2 text-[9px] bg-blue-900 text-blue-300 px-1.5 py-0.5 rounded font-bold">CHAT</span>' : '';
          const compBadge = name === data.compaction_profile ? '<span class="ml-2 text-[9px] bg-indigo-900 text-indigo-300 px-1.5 py-0.5 rounded font-bold">COMPACTION</span>' : '';
          const endpoint = p.base_url ? escapeHtml(p.base_url) : '<span class="text-slate-500">auto endpoint</span>';
          return `
            <div class="p-3 bg-slate-900/30 border border-[#334155] rounded-lg space-y-3">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <div class="font-bold text-slate-200 flex items-center">
                    <i class="fa-solid fa-bolt text-amber-400 mr-2 text-xs"></i>${name}
                    ${activeBadge}${compBadge}
                  </div>
                  <div class="text-[10px] text-slate-400 mt-1 font-mono">${p.provider} · ${p.model || "unset"}</div>
                  <div class="text-[10px] text-slate-500 mt-1 break-all font-mono">${endpoint}</div>
                </div>
                <button onclick="editProvider('${name}')" class="px-2.5 py-1 rounded border border-[#334155] hover:bg-slate-800 text-slate-300 hover:text-white text-[10px] font-bold" title="Edit profile">Edit</button>
              </div>
              <div class="flex items-center justify-between text-[10px] text-slate-500">
                <span>Max parallel: <span class="font-mono text-slate-300">${p.max_concurrency || 0}</span></span>
                <button onclick="deleteProvider('${name}')" class="text-red-400 hover:text-red-300 font-bold" title="Delete profile">Delete</button>
              </div>
            </div>`;
        }).join("");
      })
      .catch(err => {
        document.getElementById("providers-list").innerHTML = `<div class="text-red-400 text-xs">Failed to load profiles: ${err}</div>`;
      });
  }

  function newProviderForm() {
    state.editingProviderName = null;
    document.getElementById("provider-editor-title").textContent = "New Profile";
    document.getElementById("provider-editor").setAttribute("data-mode", "new");
    document.getElementById("pe-name").value = "";
    document.getElementById("pe-name").disabled = false;
    document.getElementById("pe-provider").value = "openai";
    document.getElementById("pe-model").value = "";
    document.getElementById("pe-temperature").value = "0.2";
    document.getElementById("pe-max-concurrency").value = "0";
    document.getElementById("pe-key").value = "";
    document.getElementById("pe-base-url").value = "";
    document.getElementById("pe-project-id").value = "";
    document.getElementById("pe-region").value = "";
    document.getElementById("pe-is-active").checked = false;
    suggestProviderBaseUrl("openai");
    document.getElementById("provider-editor")?.classList.remove("hidden");
  }

  function editProvider(name) {
    fetch("/api/providers")
      .then(res => res.json())
      .then(data => {
        state.currentProvidersRevision = data.providers_revision || state.currentProvidersRevision;
        const p = (data.providers || {})[name];
        if (!p) return;
        state.editingProviderName = name;
        document.getElementById("provider-editor-title").textContent = "Edit Profile: " + name;
        document.getElementById("provider-editor").setAttribute("data-mode", "edit");
        document.getElementById("pe-name").value = name;
        document.getElementById("pe-name").disabled = true;
        document.getElementById("pe-provider").value = p.provider || "openai";
        document.getElementById("pe-model").value = p.model || "";
        document.getElementById("pe-temperature").value = p.temperature != null ? p.temperature : 0.2;
        document.getElementById("pe-max-concurrency").value = p.max_concurrency || 0;
        document.getElementById("pe-key").value = "";
        document.getElementById("pe-key").placeholder = "••••••" + (p.key ? p.key.slice(-4) : " (set a new key to replace)");
        document.getElementById("pe-base-url").value = p.base_url || "";
        document.getElementById("pe-project-id").value = p.project_id || "";
        document.getElementById("pe-region").value = p.region || "";
        document.getElementById("pe-is-active").checked = (name === data.active_profile);
        suggestProviderBaseUrl(p.provider || "openai");
        document.getElementById("provider-editor")?.classList.remove("hidden");
      });
  }

  function closeProviderForm() {
    document.getElementById("provider-editor")?.classList.add("hidden");
  }

  function suggestProviderBaseUrl(provider) {
    const urlEl = document.getElementById("pe-base-url");
    const modelEl = document.getElementById("pe-model");
    const vFields = document.getElementById("pe-vertex-fields");
    populateKnownModelList("known-models-provider", provider);
    if (!urlEl || !modelEl || !vFields) return;

    if (provider === "vertex") {
      vFields.classList.remove("hidden");
      urlEl.value = "";
      modelEl.placeholder = "gemini-1.5-flash";
      return;
    }

    vFields.classList.add("hidden");
    if (provider === "anthropic") {
      urlEl.placeholder = "https://api.anthropic.com/v1/messages";
      modelEl.placeholder = "claude-3-5-sonnet-latest";
    } else if (provider === "gemini") {
      urlEl.placeholder = "(auto-built by backend)";
      modelEl.placeholder = "gemini-1.5-flash";
    } else {
      urlEl.placeholder = "https://api.openai.com/v1/chat/completions";
      modelEl.placeholder = "gpt-4o-mini";
    }
  }

  function saveProvider() {
    const name = document.getElementById("pe-name").value.trim();
    if (!name) {
      alert("Profile name is required.");
      return;
    }
    const profile = {
      provider: document.getElementById("pe-provider").value,
      model: document.getElementById("pe-model").value.trim(),
      temperature: parseFloat(document.getElementById("pe-temperature").value) || 0,
      max_concurrency: parseInt(document.getElementById("pe-max-concurrency").value, 10) || 0,
      key: document.getElementById("pe-key").value,
      base_url: document.getElementById("pe-base-url").value.trim(),
      project_id: document.getElementById("pe-project-id").value.trim(),
      region: document.getElementById("pe-region").value.trim(),
      max_tokens: 4096,
    };
    if (state.editingProviderName && profile.key === "") profile.key = "••••";
    fetch("/api/providers/save", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: state.editingProviderName || name,
        profile,
        is_active: document.getElementById("pe-is-active").checked,
        expected_revision: state.currentProvidersRevision,
      }),
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text());
        return res.json();
      })
      .then(data => {
        state.currentProvidersRevision = data.providers_revision || state.currentProvidersRevision;
        closeProviderForm();
        fetchProviders();
        fetchConfig();
        actions.appendSystemAlert && actions.appendSystemAlert("Profile Saved", `Connection profile '${state.editingProviderName || name}' saved.`, "fa-plug-circle-check text-green-400");
      })
      .catch(err => {
        if (String(err.message || err).includes("settings-conflict")) {
          alert("Provider profiles changed elsewhere. Reloading the latest profiles.");
          fetchProviders();
          return;
        }
        alert("Failed to save profile: " + err.message);
      });
  }

  function deleteProvider(name) {
    if (!confirm(`Delete provider profile '${name}'? Nodes referencing it will fall back to inline settings.`)) return;
    fetch("/api/providers/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, expected_revision: state.currentProvidersRevision }),
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "delete failed");
        return res.json();
      })
      .then(data => {
        state.currentProvidersRevision = data.providers_revision || state.currentProvidersRevision;
        fetchProviders();
        fetchConfig();
      })
      .catch(err => {
        if (String(err.message || err).includes("settings-conflict")) {
          alert("Provider profiles changed elsewhere. Reloading the latest profiles.");
          fetchProviders();
          return;
        }
        alert("Failed to delete profile: " + err);
      });
  }

  function setActiveProfile(name, target) {
    fetch("/api/providers/activate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, target, expected_revision: state.currentProvidersRevision }),
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "activate failed");
        return res.json();
      })
      .then(() => {
        fetchProviders();
        fetchConfig();
        const which = target === "compaction" ? "Compaction" : "Active chat";
        actions.appendSystemAlert && actions.appendSystemAlert("Profile Activated", `${which} connection ${name ? "set to '" + name + "'" : "reset to inline config"}.`, "fa-bolt text-amber-400");
      })
      .catch(err => {
        if (String(err.message || err).includes("settings-conflict")) {
          alert("Provider profiles changed elsewhere. Reloading the latest profiles.");
          fetchProviders();
          return;
        }
        alert("Failed to activate profile: " + err);
        fetchProviders();
      });
  }

  function populateProfileSelectors(activeChat, activeCompaction) {
    fetch("/api/providers")
      .then(res => res.json())
      .then(data => {
        const profiles = data.providers || {};
        const opts = '<option value="">— inline (use fields below) —</option>' + Object.keys(profiles).map(n => `<option value="${n}">${n} (${profiles[n].provider}/${profiles[n].model})</option>`).join("");
        const chat = document.getElementById("input-provider-profile");
        const comp = document.getElementById("input-compact-profile");
        if (chat) {
          chat.innerHTML = opts;
          chat.value = activeChat || "";
          onChatProfileChange(chat.value);
        }
        if (comp) {
          comp.innerHTML = opts;
          comp.value = activeCompaction || "";
          onCompactProfileChange(comp.value);
        }
      })
      .catch(err => console.error("Failed to load profiles for settings:", err));
  }

  function onChatProfileChange(value) {
    const inline = document.getElementById("inline-api-fields");
    if (inline) inline.style.display = value ? "none" : "";
  }

  function onCompactProfileChange(value) {
    const inline = document.getElementById("inline-compact-fields");
    if (inline) inline.style.display = value ? "none" : "";
  }

  function triggerCompaction() {
    fetch("/api/compact", { method: "POST" }).then(res => {
      if (res.ok) alert("Compaction requested in background!");
    });
  }

  return {
    suggestBaseURL,
    renderSettingsConnectionsSummary,
    openSettingsModal,
    closeSettingsModal,
    saveSettings,
    fetchConfig,
    renderExclusionChips,
    addIgnorePattern,
    removeIgnorePattern,
    addCollapsePattern,
    removeCollapsePattern,
    fetchSnapshots,
    createSnapshot,
    revertToSnapshot,
    deleteSnapshot,
    fetchMCPServers,
    addMCPServer,
    deleteMCPServer,
    suggestCompactBaseURL,
    switchSettingsTab,
    fetchProviders,
    newProviderForm,
    editProvider,
    closeProviderForm,
    suggestProviderBaseUrl,
    saveProvider,
    deleteProvider,
    setActiveProfile,
    populateProfileSelectors,
    onChatProfileChange,
    onCompactProfileChange,
    triggerCompaction,
  };
}
