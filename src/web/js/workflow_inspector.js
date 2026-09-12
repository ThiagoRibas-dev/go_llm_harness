export function createWorkflowInspectorModule({ state, actions, escapeHtml }) {
  async function loadProfiles() {
    if (state.wfProfiles) return state.wfProfiles;
    try {
      const res = await fetch("/api/providers");
      const data = await res.json();
      state.wfProfiles = data.providers || {};
    } catch {
      state.wfProfiles = {};
    }
    return state.wfProfiles;
  }

  function setNodeProfile(id, profileName) {
    const n = actions.nodeById(id);
    if (!n) return;
    n.properties = n.properties || {};
    n.properties.provider_profile = profileName;
    if (profileName) {
      delete n.properties.provider;
      delete n.properties.model;
      delete n.properties.temperature;
    } else {
      n.properties.provider = "openai";
      n.properties.model = "gpt-4o-mini";
      n.properties.temperature = 0.2;
    }
    actions.syncJsonFromModel();
    actions.renderGraph();
    actions.selectNode(id);
    actions.setValidation(true, profileName ? `Using profile '@${profileName}'.` : "Using inline model settings.");
  }

  function toggleNodeTools(id, enabled) {
    const n = actions.nodeById(id);
    if (!n) return;
    n.properties = n.properties || {};
    n.properties.tools_enabled = !!enabled;
    if (enabled) {
      if (!("mcp_tools" in n.properties)) n.properties.mcp_tools = true;
      if (!("max_turns" in n.properties)) n.properties.max_turns = 15;
      if (!Array.isArray(n.properties.allowed_tools)) n.properties.allowed_tools = [];
    }
    actions.syncJsonFromModel();
    actions.renderGraph();
    actions.selectNode(id);
  }

  function toggleNodeTool(id, toolName, enabled) {
    const n = actions.nodeById(id);
    if (!n) return;
    n.properties = n.properties || {};
    let list = Array.isArray(n.properties.allowed_tools) ? n.properties.allowed_tools.slice() : [];
    if (enabled) {
      if (!list.includes(toolName)) list.push(toolName);
    } else {
      list = list.filter(t => t !== toolName);
    }
    n.properties.allowed_tools = list;
    actions.syncJsonFromModel();
    actions.renderGraph();
    actions.selectNode(id);
  }

  function updateNodeProp(id, key, value) {
    const n = actions.nodeById(id);
    if (!n) return;
    n.properties = n.properties || {};
    n.properties[key] = value;
    actions.syncJsonFromModel();
    actions.renderGraph();
    actions.selectNode(id);
  }

  function renameNode(id, newId) {
    newId = newId.trim();
    const n = actions.nodeById(id);
    if (!n || !newId || newId === id) return;
    if (actions.nodeById(newId)) { alert("A node with that ID already exists."); return; }
    if (actions.isAnchor(n.type)) return;
    (state.wfModel.nodes || []).forEach(other => (other.inputs || []).forEach(c => {
      if (c.source_node === id) c.source_node = newId;
    }));
    n.id = newId;
    state.wfModel.selected = { kind: "node", id: newId };
    actions.renderGraph();
    actions.syncJsonFromModel();
    actions.selectNode(newId);
  }

  function addLlmInput(id) {
    const n = actions.nodeById(id);
    if (!n) return;
    const name = prompt("Input name (used as a labeled section in the prompt):", "context");
    if (!name) return;
    const clean = name.trim().replace(/\s+/g, "_");
    if (!clean) return;
    const ports = actions.inputPortsFor(n);
    if (ports.some(p => p.port === clean)) { alert("An input with that name already exists."); return; }
    n.properties.input_ports = n.properties.input_ports || (ports.length === 1 && ports[0].port === "prompt" ? ["prompt"] : ports.map(p => p.port));
    if (!n.properties.input_ports.includes(clean)) n.properties.input_ports.push(clean);
    actions.renderGraph();
    actions.syncJsonFromModel();
    actions.selectNode(id);
  }

  function removeLlmInput(id, portName) {
    const n = actions.nodeById(id);
    if (!n) return;
    if (!confirm(`Remove input '${portName}' and its connection?`)) return;
    n.inputs = (n.inputs || []).filter(c => c.target_input !== portName);
    n.properties.input_ports = (n.properties.input_ports || []).filter(p => p !== portName);
    actions.renderGraph();
    actions.syncJsonFromModel();
    actions.selectNode(id);
  }

  async function renderInspector(id) {
    const body = document.getElementById("wf-inspector-body");
    if (!body) return;
    if (!id) {
      body.innerHTML = `<p class="text-slate-500 italic">Select a node to edit its properties.</p>`;
      return;
    }
    const n = actions.nodeById(id);
    if (!n) { renderInspector(null); return; }
    n.properties = n.properties || {};
    const p = n.properties;
    const anchor = actions.isAnchor(n.type);
    const profs = await loadProfiles();
    const profileOpts = ['<option value="">— inline —</option>']
      .concat(Object.keys(profs).map(k => `<option value="${escapeHtml(k)}" ${p.provider_profile === k ? "selected" : ""}>${escapeHtml(k)} (${escapeHtml(profs[k].provider)}/${escapeHtml(profs[k].model)})</option>`)).join("");

    let fields = "";
    if (n.type === "llm") {
      const ports = actions.inputPortsFor(n);
      fields = `
        <div>
          <label class="block text-slate-400 mb-1">Connection Profile</label>
          <select data-action="set-node-profile" data-node-id="${escapeHtml(n.id)}" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">${profileOpts}</select>
        </div>
        <div id="llm-inline-fields" class="space-y-2" style="${p.provider_profile ? "display:none" : ""}">
          <div>
            <label class="block text-slate-400 mb-1">Model</label>
            <input type="text" value="${escapeHtml(p.model || "")}" data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="model" data-value-type="string" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
          </div>
          <div>
            <label class="block text-slate-400 mb-1">Temperature</label>
            <input type="number" min="0" max="2" step="0.1" value="${p.temperature ?? 0.2}" data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="temperature" data-value-type="float" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
          </div>
        </div>
        <div>
          <label class="block text-slate-400 mb-1">System Prompt</label>
          <textarea rows="4" data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="system_prompt" data-value-type="string" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono text-[10px] leading-snug">${escapeHtml(p.system_prompt || "")}</textarea>
        </div>
        <div class="rounded border border-[#334155] p-2 space-y-2">
          <label class="flex items-center gap-2 text-slate-300">
            <input type="checkbox" ${p.tools_enabled ? "checked" : ""} data-action="toggle-node-tools" data-node-id="${escapeHtml(n.id)}" class="accent-blue-500">
            <span class="font-bold">Enable tool calling (ReAct loop)</span>
          </label>
          ${p.tools_enabled ? `
          <div class="pl-4 space-y-2">
            <label class="flex items-center gap-2 text-slate-400">
              <input type="checkbox" ${p.mcp_tools !== false ? "checked" : ""} data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="mcp_tools" data-value-type="checked-bool" class="accent-blue-500">
              <span>Include MCP server tools</span>
            </label>
            <div>
              <label class="block text-slate-400 mb-1">Max turns</label>
              <input type="number" min="1" max="50" value="${p.max_turns ?? 15}" data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="max_turns" data-value-type="int" class="w-20 bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono text-[10px]">
            </div>
            <div>
              <label class="block text-slate-400 mb-1">Allowed built-in tools (blank = all)</label>
              <div class="grid grid-cols-2 gap-1">
                ${["read_file","read_spill","write_file","patch_file","execute_command","spawn_sub_agent","bm25_search"].map(t => {
                  const list = Array.isArray(p.allowed_tools) ? p.allowed_tools : [];
                  const allowed = list.length === 0 || list.includes(t);
                  return `<label class="flex items-center gap-1 text-slate-400 text-[10px]"><input type="checkbox" ${allowed ? "checked" : ""} data-action="toggle-node-tool" data-node-id="${escapeHtml(n.id)}" data-tool-name="${t}" class="accent-blue-500 scale-90"><span>${t}</span></label>`;
                }).join("")}
              </div>
            </div>
          </div>` : ""}
        </div>
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="text-slate-400">Inputs</label>
            <button type="button" data-action="add-llm-input" data-node-id="${escapeHtml(n.id)}" class="text-blue-400 hover:text-blue-300 text-[10px]"><i class="fa-solid fa-plus"></i> add</button>
          </div>
          <div class="space-y-1">
            ${ports.map(pt => `
              <div class="flex items-center justify-between bg-slate-900/50 rounded px-2 py-1">
                <span class="font-mono text-slate-300 text-[10px]">${escapeHtml(pt.port)}</span>
                ${pt.port === "prompt" && ports.length === 1 ? "" : `<button type="button" data-action="remove-llm-input" data-node-id="${escapeHtml(n.id)}" data-port-name="${escapeHtml(pt.port)}" class="text-red-400 hover:text-red-300 text-[10px]"><i class="fa-solid fa-xmark"></i></button>`}
              </div>`).join("")}
          </div>
        </div>`;
    } else if (n.type === "bm25_search") {
      fields = `
        <div>
          <label class="block text-slate-400 mb-1">Scope</label>
          <select data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="scope" data-value-type="string" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
            <option value="workspace" ${p.scope === "session" ? "" : "selected"}>workspace</option>
            <option value="session" ${p.scope === "session" ? "selected" : ""}>session</option>
          </select>
        </div>
        <div>
          <label class="block text-slate-400 mb-1">Result Limit</label>
          <input type="number" min="1" max="50" value="${p.limit ?? 5}" data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="limit" data-value-type="int" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
        </div>`;
    } else if (n.type === "tool_execution") {
      const tools = [
        ["execute_command", "Run command"],
        ["read_file", "Read file"],
        ["read_spill", "Read spilled output"],
        ["write_file", "Write file"],
        ["patch_file", "Patch file"],
      ];
      fields = `
        <div>
          <label class="block text-slate-400 mb-1">Tool</label>
          <select data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="tool_name" data-value-type="string" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
            ${tools.map(([v, l]) => `<option value="${v}" ${p.tool_name === v ? "selected" : ""}>${l}</option>`).join("")}
          </select>
        </div>
        <p class="text-[10px] text-slate-500 italic">Tool arguments are supplied by an incoming edge to "args", or left to the agent at runtime.</p>`;
    } else if (n.type === "conditional_router") {
      fields = `
        <div>
          <label class="block text-slate-400 mb-1">Condition</label>
          <select data-action="update-node-prop" data-node-id="${escapeHtml(n.id)}" data-prop="condition" data-value-type="string" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
            <option value="on_error" ${p.condition === "on_error" ? "selected" : ""}>on_error</option>
          </select>
        </div>`;
    } else {
      fields = `<p class="text-slate-500 italic text-[10px]">${anchor ? "System anchor — no editable properties." : "No editable properties."}</p>`;
    }

    body.innerHTML = `
      <div>
        <label class="block text-slate-400 mb-1">Node ID</label>
        <input type="text" value="${escapeHtml(n.id)}" ${anchor ? "disabled" : ""} data-action="rename-node" data-node-id="${escapeHtml(n.id)}" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono ${anchor ? "opacity-60" : ""}">
      </div>
      <div class="text-[10px] text-slate-500 flex justify-between">
        <span>type: <span class="text-slate-400">${escapeHtml(n.type)}</span></span>
        <span>${Math.round(n.x)}, ${Math.round(n.y)}</span>
      </div>
      ${fields}
      ${anchor ? "" : `<button type="button" data-action="delete-selected-node" class="w-full mt-2 bg-red-950/40 hover:bg-red-900 border border-red-800 text-red-300 rounded px-2 py-1 text-[10px] font-bold"><i class="fa-solid fa-trash mr-1"></i> Delete node</button>`}`;
  }

  return {
    loadProfiles,
    setNodeProfile,
    toggleNodeTools,
    toggleNodeTool,
    updateNodeProp,
    renameNode,
    addLlmInput,
    removeLlmInput,
    renderInspector,
  };
}
