export function createSettingsOpsModule({ state, actions, escapeHtml }) {
  function renderExclusionChips() {
    const ignoreList = document.getElementById("ignore-chips-list");
    const collapseList = document.getElementById("collapse-chips-list");
    if (ignoreList) {
      const ignoreHtml = state.currentIgnoredPatterns.map((pat, idx) => `
        <span class="inline-flex items-center space-x-1 px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[10px] border border-slate-700">
          <span>${pat}</span>
          <button type="button" data-remove-ignore-index="${idx}" class="text-slate-500 hover:text-red-400 font-bold">&times;</button>
        </span>`).join("");
      ignoreList.innerHTML = ignoreHtml || '<span class="text-slate-500 italic text-[10px]">No patterns ignored.</span>';
    }
    if (collapseList) {
      const collapseHtml = state.currentCollapsedPatterns.map((pat, idx) => `
        <span class="inline-flex items-center space-x-1 px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[10px] border border-slate-700">
          <span>${pat}</span>
          <button type="button" data-remove-collapse-index="${idx}" class="text-slate-500 hover:text-red-400 font-bold">&times;</button>
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
                <button type="button" data-revert-snapshot="${snap.id}" class="flex-1 py-1 bg-indigo-950/40 text-indigo-400 hover:bg-indigo-900 hover:text-white border border-indigo-900/60 rounded text-[10px] font-bold transition flex items-center justify-center space-x-1">
                  <i class="fa-solid fa-undo"></i> <span>Revert Workspace</span>
                </button>
                <button type="button" data-delete-snapshot="${snap.id}" class="px-2 py-1 bg-red-950/20 text-red-400 hover:bg-red-900 hover:text-white border border-red-900/40 rounded text-[10px] font-bold transition">
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
                <button type="button" data-delete-mcp="${escapeHtml(key)}" class="text-slate-500 hover:text-red-400 p-1 rounded hover:bg-slate-800 transition" title="Delete MCP Server">
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

  function triggerCompaction() {
    fetch("/api/compact", { method: "POST" }).then(res => {
      if (res.ok) alert("Compaction requested in background!");
    });
  }

  return {
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
    triggerCompaction,
  };
}
