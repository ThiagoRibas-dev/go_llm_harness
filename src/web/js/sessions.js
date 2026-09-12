export function createSessionsModule({ state, actions, escapeHtml }) {
  function fetchWorkspaces() {
    return fetch("/api/workspaces")
      .then(res => res.json())
      .then(data => {
        const summary = document.getElementById("workspace-active-summary");
        if (summary) {
          summary.textContent = data.active || "./workspace";
          summary.title = data.active || "./workspace";
        }

        let wsHtml = "";
        data.workspaces.forEach(ws => {
          const isActive = ws === data.active;
          const activeClass = isActive
            ? "bg-blue-950/40 border-blue-500 text-blue-400 font-bold font-mono"
            : "bg-slate-900/30 border-transparent text-slate-400 hover:bg-slate-800/40 hover:text-slate-200 font-mono";
          wsHtml += `
            <div class="flex items-center justify-between p-1.5 rounded border border-[#334155]/60 text-[10px] ${activeClass}">
              <button type="button" data-change-workspace="${escapeHtml(ws)}" class="truncate cursor-pointer flex-1 text-left" title="${escapeHtml(ws)}">${escapeHtml(ws)}</button>
              ${!isActive ? `
              <button type="button" data-remove-workspace="${escapeHtml(ws)}" class="text-slate-500 hover:text-red-400 p-0.5 ml-1" title="Remove from history">
                <i class="fa-solid fa-times text-[10px]"></i>
              </button>` : ""}
            </div>`;
        });
        const list = document.getElementById("workspaces-history-list");
        if (list) list.innerHTML = wsHtml || '<div class="text-slate-500 italic text-[10px]">No workspace history.</div>';
        return data;
      })
      .catch(err => console.error("Error loading workspaces:", err));
  }

  function removeWorkspaceFromHistory(e, path) {
    if (e) e.stopPropagation();
    if (!confirm(`Are you sure you want to remove workspace '${path}' from history?`)) return;
    fetch("/api/workspaces/remove", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path })
    })
      .then(res => res.json())
      .then(() => fetchWorkspaces())
      .catch(err => alert("Error removing workspace: " + err));
  }

  function addNewWorkspace() {
    const input = document.getElementById("new-workspace-input");
    if (!input) return;
    const path = input.value.trim();
    if (!path) return;
    fetch("/api/workspaces/select", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path })
    })
      .then(res => res.json())
      .then(() => {
        input.value = "";
        actions.fetchConfig && actions.fetchConfig();
        refreshWorkspaceTree();
        fetchWorkspaces();
        fetchSessions();
        const chatContainer = document.getElementById("chat-messages");
        if (chatContainer) chatContainer.innerHTML = "";
        actions.appendGreeting && actions.appendGreeting();
        actions.appendSystemAlert && actions.appendSystemAlert("Workspace Swapped", `Active directory is now aligned to: ${path}. Spun up a new thread.`, "fa-folder-open text-cyan-400");
        state.turnCounter = 0;
      })
      .catch(err => alert("Error adding workspace: " + err));
  }

  function changeWorkspaceFromSelector(path) {
    if (!path) return;
    fetch("/api/workspaces/select", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path })
    })
      .then(res => res.json())
      .then(() => {
        actions.fetchConfig && actions.fetchConfig();
        refreshWorkspaceTree();
        fetchWorkspaces();
        fetchSessions();
        fetchPinnedFiles();
        const chatContainer = document.getElementById("chat-messages");
        if (chatContainer) chatContainer.innerHTML = "";
        actions.appendGreeting && actions.appendGreeting();
        actions.appendSystemAlert && actions.appendSystemAlert("Workspace Swapped", `Active directory is now aligned to: ${path}. Spun up a new thread.`, "fa-folder-open text-cyan-400");
        state.turnCounter = 0;
      });
  }

  function fetchSessions() {
    let url = "/api/sessions";
    if (state.activeWorkspaceDir) url += "?workspace=" + encodeURIComponent(state.activeWorkspaceDir);
    return fetch(url)
      .then(res => res.json())
      .then(data => {
        const listContainer = document.getElementById("sessions-list");
        if (!listContainer) return data;
        if (!data.sessions || data.sessions.length === 0) {
          listContainer.innerHTML = '<div class="text-slate-500 italic">No recorded sessions.</div>';
          return data;
        }

        data.sessions.sort((a, b) => b.created_at.localeCompare(a.created_at));
        const activeSessionId = (document.getElementById("session-id")?.innerText || "").replace("Session: ", "").trim();

        let html = "";
        data.sessions.forEach(sess => {
          const isActive = sess.session_id === activeSessionId;
          const activeClass = isActive
            ? "bg-slate-800 border-blue-500/80 text-white font-bold"
            : "bg-slate-900/30 border-transparent text-slate-400 hover:bg-slate-800/40 hover:text-slate-200";
          const date = new Date(sess.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
          html += `
            <div data-select-session="${escapeHtml(sess.session_id)}" title="${escapeHtml(sess.name)}" class="group relative p-2.5 rounded-lg border-l-4 cursor-pointer transition flex flex-col justify-between space-y-1 ${activeClass}">
              <div class="flex items-center justify-between text-[11px]">
                <span class="truncate pr-8 text-slate-200 font-medium" id="sess-display-${sess.session_id}">${escapeHtml(sess.name)}</span>
                <span class="text-[10px] text-slate-500 shrink-0 font-mono">${date}</span>
              </div>
              <div class="text-[10px] text-slate-500 truncate font-mono">Dir: ${escapeHtml(sess.workspace_dir)}</div>
              <div class="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition duration-150 flex items-center space-x-1">
                <button type="button" data-rename-session="${escapeHtml(sess.session_id)}" data-session-name="${escapeHtml(sess.name)}" class="p-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-white" title="Rename Session">
                  <i class="fa-solid fa-pen text-[9px]"></i>
                </button>
                <button type="button" data-delete-session="${escapeHtml(sess.session_id)}" class="p-1 rounded bg-red-950/60 hover:bg-red-900 text-red-400 hover:text-white" title="Delete Session">
                  <i class="fa-solid fa-trash text-[9px]"></i>
                </button>
              </div>
            </div>`;
        });
        listContainer.innerHTML = html;
        return data;
      })
      .catch(err => console.error("Error loading sessions:", err));
  }

  function renameSessionPrompt(e, id, currentName) {
    if (e) e.stopPropagation();
    const newName = prompt(`Enter a new name for session '${currentName}':`, currentName);
    if (newName === null || newName.trim() === "" || newName.trim() === currentName) return;
    fetch("/api/sessions/rename", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: id, name: newName.trim() })
    })
      .then(res => res.json())
      .then(() => fetchSessions())
      .catch(err => alert("Error renaming session: " + err));
  }

  function deleteSessionConfirm(e, id) {
    if (e) e.stopPropagation();
    if (!confirm("Are you sure you want to permanently delete this conversation session and its turn logs? This cannot be undone.")) return;
    fetch("/api/sessions/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: id })
    })
      .then(res => res.json())
      .then(() => {
        const activeId = (document.getElementById("session-id")?.innerText || "").replace("Session: ", "").trim();
        if (id === activeId) {
          actions.fetchConfig && actions.fetchConfig();
          refreshWorkspaceTree();
          fetchSessions();
          const chatContainer = document.getElementById("chat-messages");
          if (chatContainer) chatContainer.innerHTML = "";
          actions.appendGreeting && actions.appendGreeting();
          state.turnCounter = 0;
        } else {
          fetchSessions();
        }
      })
      .catch(err => alert("Error deleting session: " + err));
  }

  function selectSession(id) {
    fetch("/api/sessions/select", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: id })
    })
      .then(res => res.json())
      .then(data => {
        actions.fetchConfig && actions.fetchConfig();
        refreshWorkspaceTree();
        fetchWorkspaces();
        fetchSessions();
        fetchPinnedFiles();
        const chatContainer = document.getElementById("chat-messages");
        if (chatContainer) chatContainer.innerHTML = "";
        actions.appendGreeting && actions.appendGreeting();
        if (data.history) {
          data.history.forEach((turn, index) => {
            turn.turn_number = index + 1;
            actions.appendTurnToChat && actions.appendTurnToChat(turn);
          });
        }
        state.turnCounter = data.history ? data.history.length : 0;
      })
      .catch(err => alert("Failed to switch session: " + err));
  }

  function triggerNewSession() {
    const name = prompt("Enter a name for this new conversation thread (optional):", "New Session");
    if (name === null) return;
    fetch("/api/sessions/create", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ workspace_dir: state.activeWorkspaceDir, name: name.trim() || "New Session" })
    })
      .then(res => res.json())
      .then(data => {
        actions.fetchConfig && actions.fetchConfig();
        fetchSessions();
        refreshWorkspaceTree();
        fetchPinnedFiles();
        const chatContainer = document.getElementById("chat-messages");
        if (chatContainer) chatContainer.innerHTML = "";
        actions.appendGreeting && actions.appendGreeting();
        actions.appendSystemAlert && actions.appendSystemAlert("New Session Started", `Spun up a fresh, clean conversation thread '${data.name}' inside your active workspace.`, "fa-comment-medical text-green-400");
        state.turnCounter = 0;
      })
      .catch(err => alert("Failed to start session: " + err));
  }

  function triggerFork(turn) {
    state.selectedForkTurn = turn;
    const display = document.getElementById("fork-turn-display");
    if (display) display.innerText = turn;
    const subs = document.getElementsByClassName("fork-turn-sub");
    for (const s of subs) s.innerText = turn;
    const branchInput = document.getElementById("input-branch-name");
    if (branchInput) branchInput.value = `Branch_from_Turn_${turn}`;
    document.getElementById("fork-modal")?.classList.remove("hidden");
  }

  function closeForkModal() {
    document.getElementById("fork-modal")?.classList.add("hidden");
  }

  function toggleForkFields(type) {
    const bFields = document.getElementById("branch-fields");
    if (!bFields) return;
    if (type === "branch") bFields.classList.remove("hidden");
    else bFields.classList.add("hidden");
  }

  function executeForkAction() {
    const isBranch = document.getElementById("fork-type-branch")?.checked;
    if (isBranch) {
      const branchName = document.getElementById("input-branch-name")?.value.trim() || `Branch Turn ${state.selectedForkTurn}`;
      const payload = {
        parent_session_id: (document.getElementById("session-id")?.innerText || "").replace("Session: ", "").trim(),
        turn: state.selectedForkTurn,
        branch_name: branchName
      };
      fetch("/api/sessions/branch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      })
        .then(res => res.json())
        .then(data => {
          closeForkModal();
          actions.fetchConfig && actions.fetchConfig();
          refreshWorkspaceTree();
          fetchSessions();
          const chatContainer = document.getElementById("chat-messages");
          if (chatContainer) chatContainer.innerHTML = "";
          actions.appendGreeting && actions.appendGreeting();
          actions.appendSystemAlert && actions.appendSystemAlert("New Timeline Created", `Spun up parallel timeline branch '${branchName}' from Turn ${state.selectedForkTurn}. The original timeline is completely preserved.`, "fa-code-branch text-green-400");
          if (data.history) {
            data.history.forEach((turn, index) => {
              turn.turn_number = index + 1;
              actions.appendTurnToChat && actions.appendTurnToChat(turn);
            });
          }
          state.turnCounter = state.selectedForkTurn;
          actions.switchSidebarTab && actions.switchSidebarTab("sessions");
        })
        .catch(err => alert("Failed to create branch: " + err));
      return;
    }

    fetch("/api/fork", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ turn: state.selectedForkTurn })
    })
      .then(res => res.json())
      .then(data => {
        closeForkModal();
        refreshWorkspaceTree();
        fetchSessions();
        const chatContainer = document.getElementById("chat-messages");
        if (chatContainer) chatContainer.innerHTML = "";
        actions.appendGreeting && actions.appendGreeting();
        actions.appendSystemAlert && actions.appendSystemAlert("Session Truncated", `Timeline rolled back to Turn ${state.selectedForkTurn}. All future turn files deleted.`, "fa-scissors text-amber-400");
        if (data.history) {
          data.history.forEach((turn, index) => {
            turn.turn_number = index + 1;
            actions.appendTurnToChat && actions.appendTurnToChat(turn);
          });
        }
        state.turnCounter = state.selectedForkTurn;
      })
      .catch(err => alert("Fork failed: " + err));
  }

  function currentWorkspaceTreeCollapseMap() {
    const key = state.activeWorkspaceDir || "__workspace__";
    state.workspaceCollapsedDirs = state.workspaceCollapsedDirs || {};
    state.workspaceCollapsedDirs[key] = state.workspaceCollapsedDirs[key] || {};
    return state.workspaceCollapsedDirs[key];
  }

  function isWorkspaceEntryDir(entry) {
    return !!(entry && (entry.isDir === true || entry.is_dir === true));
  }

  function workspaceEntryDepth(entry) {
    return Number((entry && entry.depth) || 0);
  }

  function workspaceEntryModifiedNote(entry) {
    return String((entry && (entry.modifiedNote || entry.modified_note)) || "");
  }

  function hasCollapsedAncestor(path, collapsedMap) {
    const parts = String(path || "").split("/");
    let prefix = "";
    for (let i = 0; i < parts.length - 1; i++) {
      prefix = prefix ? `${prefix}/${parts[i]}` : parts[i];
      if (collapsedMap[prefix]) return true;
    }
    return false;
  }

  function renderWorkspaceTreeEntries() {
    const treeContainer = document.getElementById("workspace-tree");
    if (!treeContainer) return;
    if (!Array.isArray(state.workspaceTreeEntries) || state.workspaceTreeEntries.length === 0) {
      treeContainer.innerHTML = '<div class="text-slate-500 italic">No files detected yet...</div>';
      return;
    }

    const collapsedMap = currentWorkspaceTreeCollapseMap();
    let html = "";

    state.workspaceTreeEntries.forEach((entry) => {
      const path = String((entry && entry.path) || "");
      const label = String((entry && entry.label) || path.split("/").pop() || path || "entry");
      const isDir = isWorkspaceEntryDir(entry);
      const depth = workspaceEntryDepth(entry);
      const defaultCollapsed = !!(entry && entry.collapsed);
      if (isDir && !(path in collapsedMap)) collapsedMap[path] = defaultCollapsed;
      if (hasCollapsedAncestor(path, collapsedMap)) return;
      const indent = depth * 16;

      if (isDir) {
        const isCollapsed = !!collapsedMap[path];
        html += `
          <div class="workspace-tree-row group" style="padding-left:${indent}px">
            <button type="button" data-toggle-workspace-dir="${escapeHtml(path)}" class="workspace-tree-dir" title="${escapeHtml(path)}">
              <i class="workspace-tree-expander fa-solid ${isCollapsed ? "fa-chevron-right" : "fa-chevron-down"}"></i>
              <i class="fa-solid ${isCollapsed ? "fa-folder" : "fa-folder-open"} text-slate-400"></i>
              <span class="workspace-tree-label">${escapeHtml(label)}</span>
            </button>
          </div>`;
        return;
      }

      const modifiedNote = workspaceEntryModifiedNote(entry);
      html += `
        <div class="workspace-tree-row workspace-tree-file-row group" style="padding-left:${indent}px">
          <button type="button" data-workspace-file="${escapeHtml(path)}" class="workspace-tree-file-btn" title="${escapeHtml(path)}">
            <i class="fa-regular fa-file-lines text-slate-500"></i>
            <span class="workspace-tree-label">${escapeHtml(label)}</span>
            ${modifiedNote ? `<span class="workspace-tree-meta">${escapeHtml(modifiedNote)}</span>` : ""}
          </button>
          <div class="workspace-file-row-actions opacity-0 group-hover:opacity-100 transition">
            <button type="button" data-stage-file="${escapeHtml(path)}" class="typed-action-btn">Stage</button>
          </div>
        </div>`;
    });

    treeContainer.innerHTML = html || '<div class="text-slate-500 italic">No files detected yet...</div>';
  }

  function toggleWorkspaceDir(path) {
    if (!path) return;
    const collapsedMap = currentWorkspaceTreeCollapseMap();
    const current = !!collapsedMap[path];
    collapsedMap[path] = !current;
    renderWorkspaceTreeEntries();
  }

  function refreshWorkspaceTree() {
    const treeContainer = document.getElementById("workspace-tree");
    if (!treeContainer) return;
    treeContainer.innerHTML = '<div class="text-slate-500 italic"><i class="fa-solid fa-spinner animate-spin mr-2"></i>Scanning workspace...</div>';

    fetch("/api/workspace/tree")
      .then(res => res.json())
      .then(data => {
        state.workspaceTreeEntries = Array.isArray(data.entries) ? data.entries : [];
        renderWorkspaceTreeEntries();
      })
      .catch(err => {
        treeContainer.innerHTML = `<div class="text-red-400">Failed to load directory tree: ${err}</div>`;
      });
  }

  function fetchPinnedFiles() {
    const list = document.getElementById("pinned-chips-list");
    if (!list) return;
    fetch("/api/sessions/pinned")
      .then(res => res.json())
      .then(data => {
        state.activePinnedFiles = data.pinned_files || [];
        if (state.activePinnedFiles.length === 0) {
          list.innerHTML = '<div class="text-slate-500 italic text-[10px]">No pinned context files.</div>';
          return;
        }
        list.innerHTML = state.activePinnedFiles.map((file, idx) => `
          <span class="inline-flex items-center space-x-1 px-1.5 py-0.5 rounded bg-blue-950/40 text-blue-400 border border-blue-900/60 text-[9px] font-mono leading-none truncate max-w-[120px]">
            <span class="truncate" title="${escapeHtml(file)}">${escapeHtml(file)}</span>
            <button type="button" data-remove-context-pin="${idx}" class="text-slate-500 hover:text-red-400 font-bold text-[10px]">&times;</button>
          </span>`).join("");
      })
      .catch(err => {
        list.innerHTML = `<div class="text-red-400 text-[10px]">Failed to load: ${err}</div>`;
      });
  }

  function removeContextPin(idx) {
    const file = state.activePinnedFiles[idx];
    state.activePinnedFiles.splice(idx, 1);
    fetch("/api/sessions/pinned/save", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pinned_files: state.activePinnedFiles })
    })
      .then(res => {
        if (!res.ok) throw new Error("Unpin failed");
        return res.json();
      })
      .then(() => {
        fetchPinnedFiles();
        actions.appendSystemAlert && actions.appendSystemAlert("Context File Unpinned", `Successfully unpinned file \`${file}\` from active session context.`, "fa-eraser text-amber-400");
      })
      .catch(err => alert("Failed to unpin file: " + err));
  }

  function triggerFileUpload() {
    document.getElementById("hidden-file-input")?.click();
  }

  function uploadSelectedFile(input) {
    const file = input.files[0];
    if (!file) return;
    const formData = new FormData();
    formData.append("file", file);
    actions.appendSystemAlert && actions.appendSystemAlert("Uploading Memory File", `Uploading reference document '${file.name}' to active session...`, "fa-cloud-arrow-up text-indigo-400 animate-pulse");
    fetch("/api/upload", { method: "POST", body: formData })
      .then(res => {
        if (!res.ok) throw new Error("Upload failed");
        return res.json();
      })
      .then(() => {
        input.value = "";
      })
      .catch(err => alert("File upload failed: " + err));
  }

  function enableCardEdit(e, turnNum) {
    if (e) e.stopPropagation();
    const bodyDiv = document.getElementById(`turn-body-${turnNum}`);
    const textDiv = document.getElementById(`turn-text-${turnNum}`);
    if (!bodyDiv || !textDiv) return;
    state.originalCardHtmls[turnNum] = bodyDiv.innerHTML;
    const originalText = textDiv.innerText;
    bodyDiv.innerHTML = `
      <div class="space-y-2 mt-1">
        <textarea id="edit-textarea-${turnNum}" rows="4" class="w-full bg-[#0f172a] border border-[#334155] rounded-md px-3 py-2 text-slate-100 text-sm outline-none focus:border-blue-500 font-mono leading-normal">${escapeHtml(originalText)}</textarea>
        <div class="flex items-center space-x-2">
          <button type="button" data-save-and-branch-card="${turnNum}" class="bg-blue-600 hover:bg-blue-500 text-white font-bold py-1 px-3 rounded text-[11px] transition">Save &amp; Branch</button>
          <button type="button" data-cancel-card-edit="${turnNum}" class="bg-slate-800 hover:bg-slate-700 border border-[#334155] text-slate-300 py-1 px-3 rounded text-[11px] transition">Cancel</button>
        </div>
      </div>`;
  }

  function cancelCardEdit(e, turnNum) {
    if (e) e.stopPropagation();
    const bodyDiv = document.getElementById(`turn-body-${turnNum}`);
    if (!bodyDiv) return;
    if (state.originalCardHtmls[turnNum]) {
      bodyDiv.innerHTML = state.originalCardHtmls[turnNum];
      delete state.originalCardHtmls[turnNum];
    }
  }

  function saveAndBranchCard(e, turnNum) {
    if (e) e.stopPropagation();
    const textarea = document.getElementById(`edit-textarea-${turnNum}`);
    const textVal = textarea ? textarea.value.trim() : "";
    if (!textVal) return;
    const payload = {
      parent_session_id: (document.getElementById("session-id")?.innerText || "").replace("Session: ", "").trim(),
      turn: turnNum,
      branch_name: `Branch Edit Turn ${turnNum}`,
      edit_content: textVal
    };
    fetch("/api/sessions/branch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    })
      .then(res => res.json())
      .then(data => {
        actions.fetchConfig && actions.fetchConfig();
        refreshWorkspaceTree();
        fetchSessions();
        const chatContainer = document.getElementById("chat-messages");
        if (chatContainer) chatContainer.innerHTML = "";
        actions.appendGreeting && actions.appendGreeting();
        actions.appendSystemAlert && actions.appendSystemAlert("Timeline Edited & Branched", `Spun up parallel timeline branch from edited Turn ${turnNum}. The original timeline is completely preserved.`, "fa-code-fork text-green-400");
        if (data.history) {
          data.history.forEach((turn, index) => {
            turn.turn_number = index + 1;
            actions.appendTurnToChat && actions.appendTurnToChat(turn);
          });
        }
        state.turnCounter = turnNum;
        actions.switchSidebarTab && actions.switchSidebarTab("sessions");
      })
      .catch(err => alert("Failed to branch edit: " + err));
  }

  function triggerReroll() {
    if (!confirm("Are you sure you want to delete the last Assistant response and regenerate it? Any changes made in the workspace during the last turn will be safely reverted.")) return;
    actions.appendSystemAlert && actions.appendSystemAlert("Regenerating Last Response", "Reverting last workspace edits and requesting a new Assistant generation...", "fa-rotate-right text-indigo-400 animate-spin");
    fetch("/api/sessions/reroll", {
      method: "POST",
      headers: { "Content-Type": "application/json" }
    })
      .then(res => {
        if (!res.ok) throw new Error("Reroll failed");
        return res.json();
      })
      .catch(err => alert("Reroll failed: " + err));
  }

  let sessionsBindingsInitialized = false;
  function initSessionsBindings() {
    if (sessionsBindingsInitialized) return;
    sessionsBindingsInitialized = true;

    document.getElementById("sidebar-new-session-btn")?.addEventListener("click", triggerNewSession);
    document.getElementById("file-upload-btn")?.addEventListener("click", triggerFileUpload);
    document.getElementById("hidden-file-input")?.addEventListener("change", (event) => uploadSelectedFile(event.target));
    document.getElementById("add-workspace-btn")?.addEventListener("click", addNewWorkspace);
    document.getElementById("create-snapshot-btn")?.addEventListener("click", () => actions.createSnapshot && actions.createSnapshot());
    document.getElementById("fork-close-btn")?.addEventListener("click", closeForkModal);
    document.getElementById("fork-cancel-btn")?.addEventListener("click", closeForkModal);
    document.getElementById("fork-execute-btn")?.addEventListener("click", executeForkAction);
    document.getElementById("fork-type-branch")?.addEventListener("change", () => toggleForkFields("branch"));
    document.getElementById("fork-type-truncate")?.addEventListener("change", () => toggleForkFields("truncate"));

    document.getElementById("workspaces-history-list")?.addEventListener("click", (event) => {
      const changeBtn = event.target.closest("[data-change-workspace]");
      if (changeBtn) {
        changeWorkspaceFromSelector(changeBtn.getAttribute("data-change-workspace"));
        return;
      }
      const removeBtn = event.target.closest("[data-remove-workspace]");
      if (removeBtn) removeWorkspaceFromHistory(event, removeBtn.getAttribute("data-remove-workspace"));
    });

    document.getElementById("sessions-list")?.addEventListener("click", (event) => {
      const renameBtn = event.target.closest("[data-rename-session]");
      if (renameBtn) {
        renameSessionPrompt(event, renameBtn.getAttribute("data-rename-session"), renameBtn.getAttribute("data-session-name") || "");
        return;
      }
      const deleteBtn = event.target.closest("[data-delete-session]");
      if (deleteBtn) {
        deleteSessionConfirm(event, deleteBtn.getAttribute("data-delete-session"));
        return;
      }
      const card = event.target.closest("[data-select-session]");
      if (card) selectSession(card.getAttribute("data-select-session"));
    });

    document.getElementById("pinned-chips-list")?.addEventListener("click", (event) => {
      const btn = event.target.closest("[data-remove-context-pin]");
      if (!btn) return;
      removeContextPin(Number(btn.getAttribute("data-remove-context-pin")));
    });

    document.getElementById("workspace-tree")?.addEventListener("click", (event) => {
      const toggleBtn = event.target.closest("[data-toggle-workspace-dir]");
      if (toggleBtn) {
        toggleWorkspaceDir(toggleBtn.getAttribute("data-toggle-workspace-dir"));
        return;
      }
      const openBtn = event.target.closest("[data-workspace-file]");
      if (openBtn) {
        actions.openWorkspaceFile && actions.openWorkspaceFile(openBtn.getAttribute("data-workspace-file"));
        return;
      }
      const stageBtn = event.target.closest("[data-stage-file]");
      if (stageBtn) {
        actions.stageWorkspaceFile && actions.stageWorkspaceFile(stageBtn.getAttribute("data-stage-file"));
      }
    });

    document.getElementById("chat-messages")?.addEventListener("click", (event) => {
      const saveBtn = event.target.closest("[data-save-and-branch-card]");
      if (saveBtn) {
        saveAndBranchCard(event, Number(saveBtn.getAttribute("data-save-and-branch-card")));
        return;
      }
      const cancelBtn = event.target.closest("[data-cancel-card-edit]");
      if (cancelBtn) cancelCardEdit(event, Number(cancelBtn.getAttribute("data-cancel-card-edit")));
    });
  }

  return {
    fetchWorkspaces,
    removeWorkspaceFromHistory,
    addNewWorkspace,
    changeWorkspaceFromSelector,
    fetchSessions,
    renameSessionPrompt,
    deleteSessionConfirm,
    selectSession,
    triggerNewSession,
    triggerFork,
    closeForkModal,
    toggleForkFields,
    executeForkAction,
    renderWorkspaceTreeEntries,
    toggleWorkspaceDir,
    refreshWorkspaceTree,
    fetchPinnedFiles,
    removeContextPin,
    triggerFileUpload,
    uploadSelectedFile,
    enableCardEdit,
    cancelCardEdit,
    saveAndBranchCard,
    triggerReroll,
    initSessionsBindings,
  };
}
