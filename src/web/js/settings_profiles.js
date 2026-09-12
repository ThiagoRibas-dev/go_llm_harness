export function createSettingsProfilesModule({ state, actions, escapeHtml, populateKnownModelList }) {
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
                <button type="button" data-edit-provider="${escapeHtml(name)}" class="px-2.5 py-1 rounded border border-[#334155] hover:bg-slate-800 text-slate-300 hover:text-white text-[10px] font-bold" title="Edit profile">Edit</button>
              </div>
              <div class="flex items-center justify-between text-[10px] text-slate-500">
                <span>Max parallel: <span class="font-mono text-slate-300">${p.max_concurrency || 0}</span></span>
                <button type="button" data-delete-provider="${escapeHtml(name)}" class="text-red-400 hover:text-red-300 font-bold" title="Delete profile">Delete</button>
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
        actions.fetchConfig && actions.fetchConfig();
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
        actions.fetchConfig && actions.fetchConfig();
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
        actions.fetchConfig && actions.fetchConfig();
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

  return {
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
  };
}
