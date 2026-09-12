export function createSettingsRuntimeModule({ state, actions, escapeHtml, populateKnownModelList }) {
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

  function applySettingsConfig(cfg) {
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
  }

  function openSettingsModal() {
    fetch("/api/config")
      .then(res => res.json())
      .then(cfg => {
        applySettingsConfig(cfg);
        fetch("/api/config/exclusions")
          .then(res => res.json())
          .then(ex => {
            state.currentIgnoredPatterns = ex.ignored_patterns || [];
            state.currentCollapsedPatterns = ex.collapsed_patterns || [];
            actions.renderExclusionChips && actions.renderExclusionChips();
          });
        actions.fetchMCPServers && actions.fetchMCPServers();
        actions.populateProfileSelectors && actions.populateProfileSelectors(cfg.provider_profile, cfg.compaction ? cfg.compaction.provider_profile : "");
        document.getElementById("settings-modal")?.classList.remove("hidden");
      })
      .catch(err => alert("Failed to fetch settings: " + err));
  }

  function closeSettingsModal() {
    document.getElementById("settings-modal")?.classList.add("hidden");
  }

  function buildSettingsPayload() {
    return {
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
  }

  function saveSettings(e) {
    e.preventDefault();
    const payload = buildSettingsPayload();
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
        document.getElementById("workspace-path").title = cfg.agent.workspace_dir;
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
    if (tabName === "providers") actions.fetchProviders && actions.fetchProviders();
  }

  return {
    suggestBaseURL,
    suggestCompactBaseURL,
    renderSettingsConnectionsSummary,
    applySettingsConfig,
    openSettingsModal,
    closeSettingsModal,
    buildSettingsPayload,
    saveSettings,
    fetchConfig,
    switchSettingsTab,
  };
}
