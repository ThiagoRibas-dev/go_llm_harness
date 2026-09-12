export function createWorkflowLabModule({ state, actions, escapeHtml }) {
  const NODE_W = 200;
  const NODE_H = 78;
  const COL_GAP = 70;
  const ROW_GAP = 28;
  const PIN_R = 6;

  const NODE_DEFS = {
    user_input: {
      title: "Start", icon: "fa-sign-in-alt", color: "amber",
      inputs: [], outputs: [{ port: "prompt", label: "prompt" }],
      anchor: true,
    },
    assistant_response: {
      title: "Response", icon: "fa-sign-out-alt", color: "emerald",
      inputs: [{ port: "final_output", label: "final", required: true }], outputs: [],
      anchor: true,
    },
    llm: {
      title: "LLM", icon: "fa-robot", color: "purple",
      inputs: [], outputs: [{ port: "response", label: "response" }],
      defaults: { temperature: 0.2, system_prompt: "", provider_profile: "", provider: "openai", model: "gpt-4o-mini", tools_enabled: false, mcp_tools: true, max_turns: 15, allowed_tools: [] },
      fields: ["llm_profile", "temperature", "system_prompt"],
    },
    llm_query: { aliasOf: "llm" },
    llm_synthesis: { aliasOf: "llm" },
    bm25_search: {
      title: "BM25 Search", icon: "fa-magnifying-glass", color: "indigo",
      inputs: [{ port: "query", label: "query", required: true }],
      outputs: [{ port: "search_results", label: "results" }],
      defaults: { scope: "workspace", limit: 5 },
      fields: ["scope", "limit"],
    },
    tool_execution: {
      title: "Tool", icon: "fa-terminal", color: "cyan",
      inputs: [{ port: "arguments", label: "args" }],
      outputs: [{ port: "stdout", label: "stdout" }, { port: "exit_code", label: "code" }],
      defaults: { tool_name: "execute_command" },
      fields: ["tool_name"],
    },
    conditional_router: {
      title: "Router", icon: "fa-route", color: "amber",
      inputs: [{ port: "eval_var", label: "eval" }],
      outputs: [{ port: "route_branch", label: "branch" }],
      defaults: { condition: "on_error" },
      fields: ["condition"],
    },
  };

  const NODE_STYLES = {
    amber: "border-amber-500/40 text-amber-400",
    emerald: "border-emerald-500/40 text-emerald-400",
    purple: "border-purple-500/40 text-purple-400",
    indigo: "border-indigo-500/40 text-indigo-400",
    cyan: "border-cyan-500/40 text-cyan-400",
    slate: "border-slate-500/40 text-slate-300",
  };

  let wfModel = state.wfModel || { workflowId: null, workflow: null, nodes: [], selected: null };
  state.wfModel = wfModel;
  let workflowLabEventsBound = false;

  function defFor(type) {
    const d = NODE_DEFS[type] || {};
    return d.aliasOf ? NODE_DEFS[d.aliasOf] : d;
  }

  function nodeColor(type) { return (defFor(type).color || "slate"); }
  function nodeIcon(type) { return (defFor(type).icon || "fa-circle"); }
  function isAnchor(type) { return !!defFor(type).anchor; }

  function inputPortsFor(node) {
    const d = defFor(node.type);
    if (node.type === "llm" || d.aliasOf === "llm") {
      const seen = new Set();
      const ports = [];
      (node.properties && node.properties.input_ports || []).forEach(name => {
        if (name && !seen.has(name)) {
          seen.add(name);
          ports.push({ port: name, label: name });
        }
      });
      (node.inputs || []).forEach(c => {
        if (c.target_input && !seen.has(c.target_input)) {
          seen.add(c.target_input);
          ports.push({ port: c.target_input, label: c.target_input });
        }
      });
      if (ports.length === 0) ports.push({ port: "prompt", label: "prompt" });
      return ports;
    }
    return d.inputs || [];
  }

  function outputPortsFor(node) {
    if (node.type === "llm" || defFor(node.type).aliasOf === "llm") {
      return [{ port: "response", label: "response" }];
    }
    return defFor(node.type).outputs || [];
  }

  function wfCanvas() { return document.getElementById("wf-canvas"); }
  function wfScroll() { return document.getElementById("wf-canvas-scroll"); }
  function wfEdgesSvg() { return document.getElementById("wf-edges"); }
  function nodeById(id) { return wfModel.nodes.find(n => n.id === id); }

  function loadWorkflowIntoModel(schema, workflowId) {
    const ids = Object.keys(schema.workflows || {});
    const activeId = workflowId || schema.active_workflow || ids[0];
    const wf = schema.workflows ? schema.workflows[activeId] : null;
    const rawNodes = (wf && wf.nodes) ? JSON.parse(JSON.stringify(wf.nodes)) : [];
    const nodes = rawNodes.map(n => {
      if (n.type === "llm_query" || n.type === "llm_synthesis") n.type = "llm";
      n.properties = n.properties || {};
      n.inputs = n.inputs || [];
      if (typeof n.properties.x !== "number" || typeof n.properties.y !== "number") {
        n._needsLayout = true;
      } else {
        n.x = n.properties.x;
        n.y = n.properties.y;
      }
      return n;
    });

    wfModel = { workflowId: activeId, workflow: wf, nodes, selected: null };
    state.wfModel = wfModel;
    if (nodes.some(n => n._needsLayout)) autoLayout();
    renderGraph();
    refreshLabSelector(schema);
  }

  function refreshLabSelector(schema) {
    const sel = document.getElementById("wf-lab-selector");
    if (!sel) return;
    const wfs = (schema && schema.workflows) || {};
    const current = wfModel.workflowId;
    let html = "";
    Object.keys(wfs).forEach(id => {
      const name = wfs[id].name || id;
      html += `<option value="${escapeHtml(id)}" ${id === current ? "selected" : ""}>${escapeHtml(name)} (${escapeHtml(id)})</option>`;
    });
    sel.innerHTML = html;
    sel.value = current || "";
  }

  function autoLayout() {
    const nodes = wfModel.nodes;
    if (!nodes.length) return;
    const byId = {};
    nodes.forEach(n => { byId[n.id] = n; });
    const indeg = {}, adj = {};
    nodes.forEach(n => { indeg[n.id] = 0; adj[n.id] = []; });
    nodes.forEach(n => (n.inputs || []).forEach(c => {
      if (byId[c.source_node]) {
        adj[c.source_node].push(n.id);
        indeg[n.id]++;
      }
    }));

    const depth = {}, queue = [];
    nodes.forEach(n => { depth[n.id] = 0; if (indeg[n.id] === 0) queue.push(n.id); });
    while (queue.length) {
      const id = queue.shift();
      adj[id].forEach(nxt => {
        depth[nxt] = Math.max(depth[nxt], depth[id] + 1);
        if (--indeg[nxt] === 0) queue.push(nxt);
      });
    }
    nodes.forEach(n => { if (depth[n.id] === undefined) depth[n.id] = 0; });

    const cols = {};
    nodes.forEach(n => {
      const d = depth[n.id];
      (cols[d] = cols[d] || []).push(n);
    });
    Object.keys(cols).sort((a, b) => a - b).forEach((d, ci) => {
      cols[d].forEach((n, ri) => {
        n.x = 40 + ci * (NODE_W + COL_GAP);
        n.y = 40 + ri * (NODE_H + ROW_GAP);
        n._needsLayout = false;
      });
    });
    persistPositions();
  }

  function persistPositions() {
    wfModel.nodes.forEach(n => {
      n.properties = n.properties || {};
      n.properties.x = Math.round(n.x);
      n.properties.y = Math.round(n.y);
    });
  }

  function renderGraph() {
    const canvas = wfCanvas();
    if (!canvas) return;
    Array.from(canvas.querySelectorAll(".wf-node")).forEach(el => el.remove());

    const activeNameEl = document.getElementById("workflow-active-name");
    if (activeNameEl && wfModel.workflow) {
      activeNameEl.textContent = `${wfModel.workflowId} — ${wfModel.workflow.name || wfModel.workflowId}`;
    }

    let maxX = 0, maxY = 0;
    wfModel.nodes.forEach(n => {
      n.x = typeof n.x === "number" ? n.x : 40;
      n.y = typeof n.y === "number" ? n.y : 40;
      maxX = Math.max(maxX, n.x + NODE_W + 60);
      maxY = Math.max(maxY, n.y + NODE_H + 60);
    });

    const scroll = wfScroll();
    const minW = scroll ? scroll.clientWidth - 2 : 0;
    const minH = scroll ? scroll.clientHeight - 2 : 0;
    canvas.style.width = Math.max(maxX, minW) + "px";
    canvas.style.height = Math.max(maxY, minH) + "px";
    if (wfEdgesSvg()) {
      wfEdgesSvg().setAttribute("width", Math.max(maxX, minW));
      wfEdgesSvg().setAttribute("height", Math.max(maxY, minH));
    }

    wfModel.nodes.forEach(n => canvas.appendChild(renderNodeCard(n)));
    drawEdges();
  }

  function styleFor(type) { return NODE_STYLES[nodeColor(type)] || NODE_STYLES.slate; }

  function renderNodeCard(n) {
    const sty = styleFor(n.type);
    const [borderCls, iconCls] = sty.split(" ");
    const def = defFor(n.type);
    const card = document.createElement("div");
    card.className = "wf-node absolute rounded-lg border bg-slate-900/80 shadow-lg " + borderCls;
    card.style.left = n.x + "px";
    card.style.top = n.y + "px";
    card.style.width = NODE_W + "px";
    card.style.minHeight = NODE_H + "px";
    card.dataset.id = n.id;
    if (wfModel.selected && wfModel.selected.kind === "node" && wfModel.selected.id === n.id) {
      card.classList.add("ring-2", "ring-blue-400");
    }

    const inPorts = inputPortsFor(n);
    const outPorts = outputPortsFor(n);
    const label = n.properties && n.properties.provider_profile
      ? "@" + n.properties.provider_profile
      : (n.properties && n.properties.model ? n.properties.model : (def.title || n.type));

    card.innerHTML = `
      <div class="flex items-center justify-between px-2 py-1 border-b border-slate-700/60 rounded-t-lg bg-slate-800/60 cursor-move">
        <span class="font-bold text-slate-100 text-[11px] truncate flex items-center">
          <i class="fa-solid ${nodeIcon(n.type)} mr-1.5 ${iconCls}"></i>${escapeHtml(n.id)}
        </span>
        <span class="text-[8px] uppercase tracking-wide text-slate-400 bg-slate-900/70 px-1 rounded">${escapeHtml(n.type)}</span>
      </div>
      <div class="px-2 py-1 text-[10px] text-slate-400 truncate">${escapeHtml(label)}</div>
      <div class="relative px-2 pb-1 flex justify-between text-[9px] text-slate-500">
        <div class="wf-inputs space-y-0.5">
          ${inPorts.map(p => `
            <div class="flex items-center">
              <span class="wf-pin wf-pin-in -ml-3.5 mr-1 inline-block rounded-full bg-blue-500 border-2 border-slate-900"
                style="width:${PIN_R*2}px;height:${PIN_R*2}px"
                data-node="${escapeHtml(n.id)}" data-port="${escapeHtml(p.port)}" data-kind="in" title="${escapeHtml(p.label)}"></span>
              <span class="truncate">${escapeHtml(p.label)}</span>
            </div>`).join("")}
        </div>
        <div class="wf-outputs space-y-0.5 text-right">
          ${outPorts.map(p => `
            <div class="flex items-center justify-end">
              <span class="truncate">${escapeHtml(p.label)}</span>
              <span class="wf-pin wf-pin-out -mr-3.5 ml-1 inline-block rounded-full bg-emerald-400 border-2 border-slate-900 cursor-crosshair"
                style="width:${PIN_R*2}px;height:${PIN_R*2}px"
                data-node="${escapeHtml(n.id)}" data-port="${escapeHtml(p.port)}" data-kind="out" title="${escapeHtml(p.label)}"></span>
            </div>`).join("")}
        </div>
      </div>`;

    card.addEventListener("mousedown", (e) => {
      if (e.target.classList.contains("wf-pin")) {
        if (e.target.classList.contains("wf-pin-out")) startPortDrag(e, e.target);
        return;
      }
      selectNode(n.id);
      if (isAnchor(n.type)) return;
      startNodeDrag(e, n, card);
    });
    return card;
  }

  function pinCenter(pinEl) {
    const canvas = wfCanvas();
    const cr = canvas.getBoundingClientRect();
    const r = pinEl.getBoundingClientRect();
    return {
      x: r.left + r.width / 2 - cr.left,
      y: r.top + r.height / 2 - cr.top,
    };
  }

  function bezierPath(x1, y1, x2, y2) {
    const dx = Math.max(40, Math.abs(x2 - x1) * 0.5);
    return `M ${x1},${y1} C ${x1 + dx},${y1} ${x2 - dx},${y2} ${x2},${y2}`;
  }

  function drawEdges() {
    const svg = wfEdgesSvg();
    if (!svg) return;
    svg.innerHTML = "";
    const sel = wfModel.selected;
    wfModel.nodes.forEach(n => {
      (n.inputs || []).forEach((c) => {
        const src = nodeById(c.source_node);
        if (!src) return;
        const outPin = svg.parentElement.querySelector(`.wf-pin-out[data-node="${CSS.escape(c.source_node)}"][data-port="${CSS.escape(c.source_output)}"]`);
        const inPin = svg.parentElement.querySelector(`.wf-pin-in[data-node="${CSS.escape(n.id)}"][data-port="${CSS.escape(c.target_input)}"]`);
        if (!outPin || !inPin) return;
        const a = pinCenter(outPin), b = pinCenter(inPin);
        const isSelected = sel && sel.kind === "edge" && sel.from === c.source_node && sel.fromPort === c.source_output && sel.to === n.id && sel.toPort === c.target_input;
        const d = bezierPath(a.x, a.y, b.x, b.y);
        const group = document.createElementNS("http://www.w3.org/2000/svg", "g");
        group.setAttribute("data-edge", "1");
        group.dataset.from = c.source_node;
        group.dataset.fromPort = c.source_output;
        group.dataset.to = n.id;
        group.dataset.toPort = c.target_input;

        const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
        path.setAttribute("d", d);
        path.setAttribute("fill", "none");
        path.setAttribute("stroke", isSelected ? "#f472b6" : "#818cf8");
        path.setAttribute("stroke-width", isSelected ? "3" : "2");
        path.setAttribute("opacity", isSelected ? "1" : "0.75");
        path.style.pointerEvents = "none";
        group.appendChild(path);

        const hit = document.createElementNS("http://www.w3.org/2000/svg", "path");
        hit.setAttribute("d", d);
        hit.setAttribute("fill", "none");
        hit.setAttribute("stroke", "transparent");
        hit.setAttribute("stroke-width", "14");
        hit.style.cursor = "pointer";
        group.appendChild(hit);
        svg.appendChild(group);
      });
    });
  }

  function findEdgeIndex(fromNode, fromPort, toNode, toPort) {
    const tgt = nodeById(toNode);
    if (!tgt || !tgt.inputs) return -1;
    return tgt.inputs.findIndex(c => c.source_node === fromNode && c.source_output === fromPort && c.target_input === toPort);
  }

  function addEdge(fromNode, fromPort, toNode, toPort) {
    if (fromNode === toNode) return { ok: false, error: "Cannot connect a node to itself." };
    const tgt = nodeById(toNode);
    if (!tgt) return { ok: false, error: "Target node not found." };
    const existing = (tgt.inputs || []).findIndex(c => c.target_input === toPort);
    if (existing !== -1) {
      const old = tgt.inputs[existing];
      if (old.source_node === fromNode && old.source_output === fromPort) {
        return { ok: false, error: "That connection already exists." };
      }
      tgt.inputs.splice(existing, 1);
    }
    tgt.inputs = tgt.inputs || [];
    tgt.inputs.push({ source_node: fromNode, source_output: fromPort, target_input: toPort });
    if (createsCycle(fromNode, toNode)) {
      const idx = tgt.inputs.findIndex(c => c.source_node === fromNode && c.source_output === fromPort && c.target_input === toPort);
      if (idx !== -1) tgt.inputs.splice(idx, 1);
      return { ok: false, error: "Connection would create a cycle." };
    }
    return { ok: true };
  }

  function createsCycle(from, to) {
    const adj = {};
    wfModel.nodes.forEach(n => { adj[n.id] = []; });
    wfModel.nodes.forEach(n => (n.inputs || []).forEach(c => {
      if (adj[c.source_node]) adj[c.source_node].push(n.id);
    }));
    const seen = new Set();
    const stack = [to];
    while (stack.length) {
      const u = stack.pop();
      if (u === from) return true;
      if (seen.has(u)) continue;
      seen.add(u);
      (adj[u] || []).forEach(v => stack.push(v));
    }
    return false;
  }

  function deleteSelectedEdge() {
    const sel = wfModel.selected;
    if (!sel || sel.kind !== "edge") return;
    const tgt = nodeById(sel.to);
    if (tgt && tgt.inputs) {
      const idx = findEdgeIndex(sel.from, sel.fromPort, sel.to, sel.toPort);
      if (idx !== -1) tgt.inputs.splice(idx, 1);
    }
    wfModel.selected = null;
    renderGraph();
    syncJsonFromModel();
  }

  function startPortDrag(e, pinEl) {
    e.stopPropagation();
    e.preventDefault();
    if (pinEl.dataset.kind !== "out") return;
    const fromNode = pinEl.dataset.node;
    const fromPort = pinEl.dataset.port;
    const tempSvg = document.getElementById("wf-temp-edge");
    const canvas = wfCanvas();
    if (!tempSvg || !canvas) return;

    const start = pinCenter(pinEl);
    const tempPath = document.createElementNS("http://www.w3.org/2000/svg", "path");
    tempPath.setAttribute("fill", "none");
    tempPath.setAttribute("stroke", "#38bdf8");
    tempPath.setAttribute("stroke-width", "2");
    tempPath.setAttribute("stroke-dasharray", "5,4");
    tempPath.setAttribute("opacity", "0.9");
    tempSvg.appendChild(tempPath);

    document.body.style.userSelect = "none";

    const highlightPins = (on) => {
      canvas.querySelectorAll(".wf-pin-in").forEach(p => {
        p.style.boxShadow = on ? "0 0 0 3px rgba(56,189,248,0.35)" : "";
      });
    };
    highlightPins(true);

    const eventPos = (ev) => {
      const cr = canvas.getBoundingClientRect();
      return { x: ev.clientX - cr.left, y: ev.clientY - cr.top };
    };

    const onMove = (ev) => {
      const p = eventPos(ev);
      tempPath.setAttribute("d", bezierPath(start.x, start.y, p.x, p.y));
    };

    const onUp = (ev) => {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
      document.body.style.userSelect = "";
      highlightPins(false);
      if (tempPath.parentNode) tempPath.parentNode.removeChild(tempPath);
      const el = document.elementFromPoint(ev.clientX, ev.clientY);
      if (el && el.classList.contains("wf-pin-in")) {
        const toNode = el.dataset.node;
        const toPort = el.dataset.port;
        const res = addEdge(fromNode, fromPort, toNode, toPort);
        if (!res.ok) {
          setValidation(false, res.error);
        } else {
          setValidation(true, "Connected.");
          syncJsonFromModel();
        }
      }
      renderGraph();
    };

    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }

  function startNodeDrag(e, node, cardEl) {
    e.preventDefault();
    const startX = e.clientX, startY = e.clientY;
    const origX = node.x, origY = node.y;
    const onMove = (ev) => {
      node.x = Math.max(0, origX + ev.clientX - startX);
      node.y = Math.max(0, origY + ev.clientY - startY);
      cardEl.style.left = node.x + "px";
      cardEl.style.top = node.y + "px";
      drawEdges();
    };
    const onUp = () => {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
      persistPositions();
      syncJsonFromModel();
      renderGraph();
      selectNode(node.id);
    };
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }

  function selectNode(id) {
    wfModel.selected = { kind: "node", id };
    document.querySelectorAll(".wf-node").forEach(el => {
      el.classList.toggle("ring-2", el.dataset.id === id);
      el.classList.toggle("ring-blue-400", el.dataset.id === id);
    });
    renderInspector(id);
  }

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
    const n = nodeById(id);
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
    syncJsonFromModel();
    renderGraph();
    selectNode(id);
    setValidation(true, profileName ? `Using profile '@${profileName}'.` : "Using inline model settings.");
  }

  function toggleNodeTools(id, enabled) {
    const n = nodeById(id);
    if (!n) return;
    n.properties = n.properties || {};
    n.properties.tools_enabled = !!enabled;
    if (enabled) {
      if (!("mcp_tools" in n.properties)) n.properties.mcp_tools = true;
      if (!("max_turns" in n.properties)) n.properties.max_turns = 15;
      if (!Array.isArray(n.properties.allowed_tools)) n.properties.allowed_tools = [];
    }
    syncJsonFromModel();
    renderGraph();
    selectNode(id);
  }

  function toggleNodeTool(id, toolName, enabled) {
    const n = nodeById(id);
    if (!n) return;
    n.properties = n.properties || {};
    let list = Array.isArray(n.properties.allowed_tools) ? n.properties.allowed_tools.slice() : [];
    if (enabled) {
      if (!list.includes(toolName)) list.push(toolName);
    } else {
      list = list.filter(t => t !== toolName);
    }
    n.properties.allowed_tools = list;
    syncJsonFromModel();
    renderGraph();
    selectNode(id);
  }

  function updateNodeProp(id, key, value) {
    const n = nodeById(id);
    if (!n) return;
    n.properties = n.properties || {};
    n.properties[key] = value;
    syncJsonFromModel();
    const card = wfCanvas() && wfCanvas().querySelector(`.wf-node[data-id="${CSS.escape(id)}"]`);
    if (card) {
      renderGraph();
      selectNode(id);
    }
  }

  function renameNode(id, newId) {
    newId = newId.trim();
    const n = nodeById(id);
    if (!n || !newId || newId === id) return;
    if (nodeById(newId)) { alert("A node with that ID already exists."); return; }
    if (isAnchor(n.type)) return;
    wfModel.nodes.forEach(other => (other.inputs || []).forEach(c => {
      if (c.source_node === id) c.source_node = newId;
    }));
    n.id = newId;
    wfModel.selected = { kind: "node", id: newId };
    renderGraph();
    syncJsonFromModel();
    selectNode(newId);
  }

  function addLlmInput(id) {
    const n = nodeById(id);
    if (!n) return;
    const name = prompt("Input name (used as a labeled section in the prompt):", "context");
    if (!name) return;
    const clean = name.trim().replace(/\s+/g, "_");
    if (!clean) return;
    const ports = inputPortsFor(n);
    if (ports.some(p => p.port === clean)) { alert("An input with that name already exists."); return; }
    n.properties.input_ports = n.properties.input_ports || (ports.length === 1 && ports[0].port === "prompt" ? ["prompt"] : ports.map(p => p.port));
    if (!n.properties.input_ports.includes(clean)) n.properties.input_ports.push(clean);
    renderGraph();
    syncJsonFromModel();
    selectNode(id);
  }

  function removeLlmInput(id, portName) {
    const n = nodeById(id);
    if (!n) return;
    if (!confirm(`Remove input '${portName}' and its connection?`)) return;
    n.inputs = (n.inputs || []).filter(c => c.target_input !== portName);
    n.properties.input_ports = (n.properties.input_ports || []).filter(p => p !== portName);
    renderGraph();
    syncJsonFromModel();
    selectNode(id);
  }

  async function renderInspector(id) {
    const body = document.getElementById("wf-inspector-body");
    if (!body) return;
    if (!id) {
      body.innerHTML = `<p class="text-slate-500 italic">Select a node to edit its properties.</p>`;
      return;
    }
    const n = nodeById(id);
    if (!n) { renderInspector(null); return; }
    n.properties = n.properties || {};
    const p = n.properties;
    const anchor = isAnchor(n.type);
    const profs = await loadProfiles();
    const profileOpts = ['<option value="">— inline —</option>']
      .concat(Object.keys(profs).map(k => `<option value="${escapeHtml(k)}" ${p.provider_profile === k ? "selected" : ""}>${escapeHtml(k)} (${escapeHtml(profs[k].provider)}/${escapeHtml(profs[k].model)})</option>`)).join("");

    let fields = "";
    if (n.type === "llm") {
      const ports = inputPortsFor(n);
      fields = `
        <div>
          <label class="block text-slate-400 mb-1">Connection Profile</label>
          <select onchange="setNodeProfile('${escapeHtml(n.id)}',this.value)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">${profileOpts}</select>
        </div>
        <div id="llm-inline-fields" class="space-y-2" style="${p.provider_profile ? "display:none" : ""}">
          <div>
            <label class="block text-slate-400 mb-1">Model</label>
            <input type="text" value="${escapeHtml(p.model || "")}" onchange="updateNodeProp('${escapeHtml(n.id)}','model',this.value)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
          </div>
          <div>
            <label class="block text-slate-400 mb-1">Temperature</label>
            <input type="number" min="0" max="2" step="0.1" value="${p.temperature ?? 0.2}" onchange="updateNodeProp('${escapeHtml(n.id)}','temperature',parseFloat(this.value)||0)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
          </div>
        </div>
        <div>
          <label class="block text-slate-400 mb-1">System Prompt</label>
          <textarea rows="4" onchange="updateNodeProp('${escapeHtml(n.id)}','system_prompt',this.value)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono text-[10px] leading-snug">${escapeHtml(p.system_prompt || "")}</textarea>
        </div>
        <div class="rounded border border-[#334155] p-2 space-y-2">
          <label class="flex items-center gap-2 text-slate-300">
            <input type="checkbox" ${p.tools_enabled ? "checked" : ""} onchange="toggleNodeTools('${escapeHtml(n.id)}', this.checked)" class="accent-blue-500">
            <span class="font-bold">Enable tool calling (ReAct loop)</span>
          </label>
          ${p.tools_enabled ? `
          <div class="pl-4 space-y-2">
            <label class="flex items-center gap-2 text-slate-400">
              <input type="checkbox" ${p.mcp_tools !== false ? "checked" : ""} onchange="updateNodeProp('${escapeHtml(n.id)}','mcp_tools',this.checked)" class="accent-blue-500">
              <span>Include MCP server tools</span>
            </label>
            <div>
              <label class="block text-slate-400 mb-1">Max turns</label>
              <input type="number" min="1" max="50" value="${p.max_turns ?? 15}" onchange="updateNodeProp('${escapeHtml(n.id)}','max_turns',parseInt(this.value)||15)" class="w-20 bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono text-[10px]">
            </div>
            <div>
              <label class="block text-slate-400 mb-1">Allowed built-in tools (blank = all)</label>
              <div class="grid grid-cols-2 gap-1">
                ${["read_file","read_spill","write_file","patch_file","execute_command","spawn_sub_agent","bm25_search"].map(t => {
                  const list = Array.isArray(p.allowed_tools) ? p.allowed_tools : [];
                  const allowed = list.length === 0 || list.includes(t);
                  return `<label class="flex items-center gap-1 text-slate-400 text-[10px]"><input type="checkbox" ${allowed ? "checked" : ""} onchange="toggleNodeTool('${escapeHtml(n.id)}','${t}',this.checked)" class="accent-blue-500 scale-90"><span>${t}</span></label>`;
                }).join("")}
              </div>
            </div>
          </div>` : ""}
        </div>
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="text-slate-400">Inputs</label>
            <button type="button" onclick="addLlmInput('${escapeHtml(n.id)}')" class="text-blue-400 hover:text-blue-300 text-[10px]"><i class="fa-solid fa-plus"></i> add</button>
          </div>
          <div class="space-y-1">
            ${ports.map(pt => `
              <div class="flex items-center justify-between bg-slate-900/50 rounded px-2 py-1">
                <span class="font-mono text-slate-300 text-[10px]">${escapeHtml(pt.port)}</span>
                ${pt.port === "prompt" && ports.length === 1 ? "" : `<button type="button" onclick="removeLlmInput('${escapeHtml(n.id)}','${escapeHtml(pt.port)}')" class="text-red-400 hover:text-red-300 text-[10px]"><i class="fa-solid fa-xmark"></i></button>`}
              </div>`).join("")}
          </div>
        </div>`;
    } else if (n.type === "bm25_search") {
      fields = `
        <div>
          <label class="block text-slate-400 mb-1">Scope</label>
          <select onchange="updateNodeProp('${escapeHtml(n.id)}','scope',this.value)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
            <option value="workspace" ${p.scope === "session" ? "" : "selected"}>workspace</option>
            <option value="session" ${p.scope === "session" ? "selected" : ""}>session</option>
          </select>
        </div>
        <div>
          <label class="block text-slate-400 mb-1">Result Limit</label>
          <input type="number" min="1" max="50" value="${p.limit ?? 5}" onchange="updateNodeProp('${escapeHtml(n.id)}','limit',parseInt(this.value)||5)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
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
          <select onchange="updateNodeProp('${escapeHtml(n.id)}','tool_name',this.value)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
            ${tools.map(([v, l]) => `<option value="${v}" ${p.tool_name === v ? "selected" : ""}>${l}</option>`).join("")}
          </select>
        </div>
        <p class="text-[10px] text-slate-500 italic">Tool arguments are supplied by an incoming edge to "args", or left to the agent at runtime.</p>`;
    } else if (n.type === "conditional_router") {
      fields = `
        <div>
          <label class="block text-slate-400 mb-1">Condition</label>
          <select onchange="updateNodeProp('${escapeHtml(n.id)}','condition',this.value)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono">
            <option value="on_error" ${p.condition === "on_error" ? "selected" : ""}>on_error</option>
          </select>
        </div>`;
    } else {
      fields = `<p class="text-slate-500 italic text-[10px]">${anchor ? "System anchor — no editable properties." : "No editable properties."}</p>`;
    }

    body.innerHTML = `
      <div>
        <label class="block text-slate-400 mb-1">Node ID</label>
        <input type="text" value="${escapeHtml(n.id)}" ${anchor ? "disabled" : ""} onchange="renameNode('${escapeHtml(n.id)}',this.value)" class="w-full bg-[#0f172a] border border-[#334155] rounded px-2 py-1 text-slate-200 font-mono ${anchor ? "opacity-60" : ""}">
      </div>
      <div class="text-[10px] text-slate-500 flex justify-between">
        <span>type: <span class="text-slate-400">${escapeHtml(n.type)}</span></span>
        <span>${Math.round(n.x)}, ${Math.round(n.y)}</span>
      </div>
      ${fields}
      ${anchor ? "" : `<button type="button" onclick="deleteSelectedNode()" class="w-full mt-2 bg-red-950/40 hover:bg-red-900 border border-red-800 text-red-300 rounded px-2 py-1 text-[10px] font-bold"><i class="fa-solid fa-trash mr-1"></i> Delete node</button>`}`;
  }

  function buildSchemaFromModel() {
    let schema;
    try {
      schema = JSON.parse(document.getElementById("workflow-json-editor").value || "{}");
    } catch {
      schema = { workflows: {} };
    }
    schema.workflows = schema.workflows || {};
    if (!wfModel.workflowId) return schema;

    const nodes = wfModel.nodes.map(n => {
      const out = {
        id: n.id,
        type: n.type,
        properties: Object.assign({}, n.properties),
        inputs: (n.inputs || []).map(c => ({ ...c })),
      };
      out.properties.x = Math.round(n.x);
      out.properties.y = Math.round(n.y);
      delete out._needsLayout;
      return out;
    });
    schema.workflows[wfModel.workflowId] = Object.assign({}, wfModel.workflow, { nodes });
    schema.active_workflow = wfModel.workflowId;
    return schema;
  }

  function syncJsonFromModel() {
    const editor = document.getElementById("workflow-json-editor");
    if (!editor) return;
    editor.value = JSON.stringify(buildSchemaFromModel(), null, 2);
  }

  function reloadGraphFromJson() {
    const editor = document.getElementById("workflow-json-editor");
    if (!editor) return;
    try {
      const schema = JSON.parse(editor.value);
      loadWorkflowIntoModel(schema);
      setValidation(true, "Graph reloaded from JSON.");
    } catch (err) {
      setValidation(false, "JSON error: " + err.message);
    }
  }

  function toggleWorkflowJson() {
    const wrap = document.getElementById("workflow-json-wrap");
    const chev = document.getElementById("workflow-json-chevron");
    if (!wrap || !chev) return;
    const open = !wrap.classList.contains("hidden");
    wrap.classList.toggle("hidden");
    chev.style.transform = open ? "" : "rotate(180deg)";
  }

  function setValidation(ok, msg) {
    const el = document.getElementById("workflow-validation-status");
    if (!el) return;
    el.className = (ok ? "text-emerald-400" : "text-red-400") + " font-mono text-[10px] flex items-center";
    el.innerHTML = `<i class="fa-solid ${ok ? "fa-circle-check" : "fa-triangle-exclamation"} mr-1.5"></i> ${escapeHtml(msg)}`;
  }

  function toggleAddNodeMenu() {
    document.getElementById("add-node-menu")?.classList.toggle("hidden");
  }

  function autoLayoutNodes() {
    autoLayout();
    renderGraph();
    syncJsonFromModel();
    setValidation(true, "Auto-arranged nodes.");
  }

  function uniqueNodeId(type) {
    if (!nodeById(type)) return type;
    let n = 1;
    while (nodeById(`${type}_${n}`)) n++;
    return `${type}_${n}`;
  }

  function nextNodePosition() {
    if (!wfModel.nodes.length) return { x: 40, y: 40 };
    let maxRight = 0;
    wfModel.nodes.forEach(n => { maxRight = Math.max(maxRight, (n.x || 0) + NODE_W); });
    return { x: maxRight + COL_GAP, y: 40 };
  }

  function addNode(type) {
    document.getElementById("add-node-menu")?.classList.add("hidden");
    const def = NODE_DEFS[type];
    if (!def || def.anchor) return;
    const id = uniqueNodeId(type);
    const pos = nextNodePosition();
    const node = {
      id,
      type,
      x: pos.x,
      y: pos.y,
      properties: Object.assign({}, def.defaults || {}),
      inputs: [],
    };
    wfModel.nodes.push(node);
    wfModel.selected = { kind: "node", id };
    persistPositions();
    renderGraph();
    syncJsonFromModel();
    selectNode(id);
    setValidation(true, `Added ${type} node '${id}'.`);
  }

  function deleteSelectedNode() {
    const sel = wfModel.selected;
    if (!sel || sel.kind !== "node") return;
    const n = nodeById(sel.id);
    if (!n || isAnchor(n.type)) return;
    wfModel.nodes = wfModel.nodes.filter(x => x.id !== sel.id);
    wfModel.nodes.forEach(other => {
      other.inputs = (other.inputs || []).filter(c => c.source_node !== sel.id);
    });
    wfModel.selected = null;
    renderGraph();
    syncJsonFromModel();
    renderInspector(null);
    setValidation(true, `Deleted node '${sel.id}'.`);
  }

  function initWorkflowLabEvents() {
    if (workflowLabEventsBound) return;
    workflowLabEventsBound = true;

    document.addEventListener("scroll", () => { if (wfCanvas()) drawEdges(); }, true);
    window.addEventListener("resize", () => { if (wfCanvas()) drawEdges(); });

    document.addEventListener("click", (e) => {
      const canvas = wfCanvas();
      const panel = document.getElementById("workflow-lab-surface");
      if (!canvas || !panel || panel.classList.contains("hidden")) return;
      const g = e.target.closest && e.target.closest("g[data-edge='1']");
      if (g && canvas.contains(g)) {
        wfModel.selected = {
          kind: "edge",
          from: g.dataset.from, fromPort: g.dataset.fromPort,
          to: g.dataset.to, toPort: g.dataset.toPort,
        };
        document.querySelectorAll(".wf-node").forEach(el => {
          el.classList.remove("ring-2", "ring-blue-400");
        });
        renderInspector(null);
        drawEdges();
      } else if (!e.target.closest(".wf-node") && !e.target.closest("#wf-inspector") && !e.target.closest("button") && !e.target.closest("input") && !e.target.closest("textarea") && !e.target.closest("select")) {
        wfModel.selected = null;
        document.querySelectorAll(".wf-node").forEach(el => {
          el.classList.remove("ring-2", "ring-blue-400");
        });
        renderInspector(null);
        drawEdges();
      }
    });

    document.addEventListener("keydown", (e) => {
      const panel = document.getElementById("workflow-lab-surface");
      if (!panel || panel.classList.contains("hidden")) return;
      const tag = (e.target.tagName || "").toLowerCase();
      if (tag === "input" || tag === "textarea" || tag === "select") return;
      if (e.key === "Delete" || e.key === "Backspace") {
        if (wfModel.selected && wfModel.selected.kind === "edge") {
          e.preventDefault();
          deleteSelectedEdge();
        } else if (wfModel.selected && wfModel.selected.kind === "node") {
          const n = nodeById(wfModel.selected.id);
          if (n && !isAnchor(n.type)) {
            e.preventDefault();
            deleteSelectedNode();
          }
        }
      }
    });

    document.addEventListener("click", (e) => {
      const menu = document.getElementById("add-node-menu");
      if (menu && !menu.classList.contains("hidden") && !e.target.closest("#add-node-menu") && !e.target.closest("button[onclick='toggleAddNodeMenu()']")) {
        menu.classList.add("hidden");
      }
    });
  }

  function loadWorkflowsSchema() {
    fetch("/workflows.json")
      .then(res => res.json())
      .then(data => {
        document.getElementById("workflow-json-editor").value = JSON.stringify(data, null, 2);
        loadWorkflowIntoModel(data);
        setValidation(true, "Graph loaded.");
      })
      .catch(() => {
        const defaultSchema = {
          active_workflow: "linear_chat",
          workflows: {
            linear_chat: {
              name: "Standard Linear Chat",
              description: "Standard conversational agent loop mapping user input to a single, high-fidelity LLM response.",
              nodes: [
                { id: "start", type: "user_input", properties: { x: 40, y: 40 }, inputs: [] },
                { id: "query_node", type: "llm", properties: { provider: "openai", model: "gpt-4o", temperature: 0.0, system_prompt: "You are a highly capable agent..." }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
                { id: "terminal", type: "assistant_response", properties: {}, inputs: [{ source_node: "query_node", source_output: "response", target_input: "final_output" }] }
              ]
            }
          }
        };
        document.getElementById("workflow-json-editor").value = JSON.stringify(defaultSchema, null, 2);
        loadWorkflowIntoModel(defaultSchema);
      });
  }

  function compileWorkflowWithAI() {
    const promptInput = document.getElementById("ai-workflow-prompt");
    const prompt = promptInput.value.trim();
    if (!prompt) return;
    setValidation(false, "AI Compiler compiling graph...");

    setTimeout(() => {
      let schema;
      if (prompt.toLowerCase().includes("parallel") || prompt.toLowerCase().includes("decomposed") || prompt.toLowerCase().includes("axis")) {
        schema = {
          active_workflow: "enhanced_cognition",
          workflows: {
            enhanced_cognition: {
              name: "Enhanced Cognition (POADR)",
              description: "Decomposes your query concurrently across 5 parallel cognitive axes to eliminate representational interference in smaller models, merging them in a final synthesis pass.",
              nodes: [
                { id: "start", type: "user_input", properties: {}, inputs: [] },
                { id: "axis_chronological", type: "llm", properties: { provider: "openai", model: "gpt-4o-mini", temperature: 0.1, system_prompt: "You are a chronological state tracking specialist." }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
                { id: "axis_causal_logical", type: "llm", properties: { provider: "openai", model: "gpt-4o-mini", temperature: 0.1, system_prompt: "You are a causal-logical constraint specialist." }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
                { id: "axis_semantic_world", type: "llm", properties: { provider: "openai", model: "gpt-4o-mini", temperature: 0.1, system_prompt: "You are a spatial-ontological world specialist." }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
                { id: "axis_behavioral_psych", type: "llm", properties: { provider: "openai", model: "gpt-4o-mini", temperature: 0.1, system_prompt: "You are a social-behavioral psychology specialist." }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
                { id: "axis_stylistic_prose", type: "llm", properties: { provider: "openai", model: "gpt-4o-mini", temperature: 0.1, system_prompt: "You are a stylistic-prose aesthetics specialist." }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
                { id: "aggregator", type: "llm", properties: { provider: "openai", model: "gpt-4o", temperature: 0.2, input_ports: ["chronological_context","causal_context","semantic_context","behavioral_context","stylistic_context","raw_prompt"], system_prompt: "Synthesize the five parallel cognitive reports into a single, high-fidelity response." }, inputs: [
                  { source_node: "axis_chronological", source_output: "response", target_input: "chronological_context" },
                  { source_node: "axis_causal_logical", source_output: "response", target_input: "causal_context" },
                  { source_node: "axis_semantic_world", source_output: "response", target_input: "semantic_context" },
                  { source_node: "axis_behavioral_psych", source_output: "response", target_input: "behavioral_context" },
                  { source_node: "axis_stylistic_prose", source_output: "response", target_input: "stylistic_context" },
                  { source_node: "start", source_output: "prompt", target_input: "raw_prompt" }
                ] },
                { id: "terminal", type: "assistant_response", properties: {}, inputs: [{ source_node: "aggregator", source_output: "response", target_input: "final_output" }] }
              ]
            }
          }
        };
      } else {
        schema = {
          active_workflow: "linear_chat",
          workflows: {
            linear_chat: {
              name: "Standard Linear Chat",
              description: "Standard conversational agent loop mapping user input to a single, high-fidelity LLM response.",
              nodes: [
                { id: "start", type: "user_input", properties: {}, inputs: [] },
                { id: "query_node", type: "llm", properties: { provider: "openai", model: "gpt-4o", temperature: 0.0, system_prompt: "You are a highly capable agent with access to a local terminal sandbox." }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
                { id: "terminal", type: "assistant_response", properties: {}, inputs: [{ source_node: "query_node", source_output: "response", target_input: "final_output" }] }
              ]
            }
          }
        };
      }

      Object.values(schema.workflows).forEach(wf => (wf.nodes || []).forEach(n => {
        if (n.type === "llm_query" || n.type === "llm_synthesis") n.type = "llm";
        n.properties = n.properties || {};
        delete n.properties.x;
        delete n.properties.y;
      }));
      document.getElementById("workflow-json-editor").value = JSON.stringify(schema, null, 2);
      loadWorkflowIntoModel(schema);
      promptInput.value = "";
      actions.appendSystemAlert && actions.appendSystemAlert("AI Workflow Staged", "Generated a node-graph draft. Review it and click Compile & Apply to save.", "fa-wand-magic-sparkles text-blue-400");
    }, 1200);
  }

  function saveWorkflowConfigurations() {
    persistPositions();
    const schema = buildSchemaFromModel();
    const v = validateGraph(wfModel);
    if (!v.ok) {
      setValidation(false, v.errors.join(" "));
      alert("Cannot apply: " + v.errors.join(" "));
      return;
    }
    if (v.warnings.length) setValidation(true, v.warnings.join(" "));
    else setValidation(true, "Graph valid.");
    fetch("/api/workflows/save", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(schema)
    })
      .then(res => {
        if (!res.ok) throw new Error("Save failed");
        return res.json();
      })
      .then(() => {
        document.getElementById("workflow-json-editor").value = JSON.stringify(schema, null, 2);
        loadWorkflowSelector();
        setValidation(true, "Saved.");
        actions.appendSystemAlert && actions.appendSystemAlert("Active Pipeline Saved", `workflows.json saved and hot-swapped! Active pipeline: '${schema.active_workflow}'.`, "fa-check-double text-green-400");
      })
      .catch(err => alert("Error committing workflow to disk: " + err));
  }

  function readSchemaFromEditor() {
    try {
      return JSON.parse(document.getElementById("workflow-json-editor").value || "{}");
    } catch (err) {
      alert("The Advanced/JSON panel contains invalid JSON:\n" + err.message);
      return null;
    }
  }

  function switchLabWorkflow(id) {
    if (!id) return;
    const schema = readSchemaFromEditor();
    if (!schema) return;
    if (!schema.workflows || !schema.workflows[id]) {
      alert("Workflow '" + id + "' not found in the JSON.");
      return;
    }
    loadWorkflowIntoModel(schema, id);
    setValidation(true, `Editing '${id}'. Click Compile & Apply to save/activate.`);
  }

  function newWorkflow() {
    const schema = readSchemaFromEditor();
    if (!schema) return;
    schema.workflows = schema.workflows || {};
    const id = uniqueWorkflowId(schema, "new_workflow");
    const display = prompt("Name for the new workflow (id):", id);
    if (display === null) return;
    const clean = (display.trim() || id).replace(/\s+/g, "_").replace(/[^a-zA-Z0-9_-]/g, "");
    if (!clean) { alert("Invalid workflow id."); return; }
    if (schema.workflows[clean]) { alert("A workflow with that id already exists."); return; }
    schema.workflows[clean] = {
      name: display.trim() || clean,
      description: "Custom workflow.",
      nodes: [
        { id: "start", type: "user_input", properties: { x: 40, y: 40 }, inputs: [] },
        { id: "query_node", type: "llm", properties: { x: 310, y: 40, provider: "openai", model: "gpt-4o-mini", temperature: 0.2, system_prompt: "" }, inputs: [{ source_node: "start", source_output: "prompt", target_input: "prompt" }] },
        { id: "terminal", type: "assistant_response", properties: { x: 580, y: 40 }, inputs: [{ source_node: "query_node", source_output: "response", target_input: "final_output" }] },
      ],
    };
    document.getElementById("workflow-json-editor").value = JSON.stringify(schema, null, 2);
    loadWorkflowIntoModel(schema, clean);
    setValidation(true, `New workflow '${clean}' staged. Click Compile & Apply to save.`);
  }

  function cloneWorkflow() {
    const schema = readSchemaFromEditor();
    if (!schema) return;
    const srcId = wfModel.workflowId;
    if (!srcId || !schema.workflows || !schema.workflows[srcId]) {
      alert("No workflow selected to clone.");
      return;
    }
    const cloneId = uniqueWorkflowId(schema, srcId + "_clone");
    const display = prompt("Name for the cloned workflow (id):", cloneId);
    if (display === null) return;
    const clean = (display.trim() || cloneId).replace(/\s+/g, "_").replace(/[^a-zA-Z0-9_-]/g, "");
    if (!clean) { alert("Invalid workflow id."); return; }
    if (schema.workflows[clean]) { alert("A workflow with that id already exists."); return; }
    schema.workflows[clean] = JSON.parse(JSON.stringify(schema.workflows[srcId]));
    schema.workflows[clean].name = (schema.workflows[clean].name || srcId) + " (copy)";
    document.getElementById("workflow-json-editor").value = JSON.stringify(schema, null, 2);
    loadWorkflowIntoModel(schema, clean);
    setValidation(true, `Cloned '${srcId}' to '${clean}'. Click Compile & Apply to save.`);
  }

  function deleteWorkflow() {
    const schema = readSchemaFromEditor();
    if (!schema) return;
    const id = wfModel.workflowId;
    if (!id || !schema.workflows || !schema.workflows[id]) {
      alert("No workflow selected.");
      return;
    }
    const ids = Object.keys(schema.workflows);
    if (ids.length <= 1) {
      alert("You cannot delete the last workflow.");
      return;
    }
    if (!confirm(`Delete workflow '${id}'? This cannot be undone after you click Compile & Apply.`)) return;
    delete schema.workflows[id];
    if (schema.active_workflow === id) schema.active_workflow = Object.keys(schema.workflows)[0];
    document.getElementById("workflow-json-editor").value = JSON.stringify(schema, null, 2);
    loadWorkflowIntoModel(schema, schema.active_workflow);
    setValidation(true, `Deleted '${id}'. Click Compile & Apply to save.`);
  }

  function uniqueWorkflowId(schema, base) {
    if (!schema.workflows[base]) return base;
    let n = 2;
    while (schema.workflows[`${base}_${n}`]) n++;
    return `${base}_${n}`;
  }

  function validateGraph(model) {
    const errors = [];
    const warnings = [];
    const nodes = model.nodes || [];
    const ids = new Set(nodes.map(n => n.id));
    let start = null, end = null;

    nodes.forEach(n => {
      if (n.type === "user_input") {
        if (start) errors.push("More than one Start node.");
        start = n;
      }
      if (n.type === "assistant_response") {
        if (end) errors.push("More than one Response node.");
        end = n;
      }
      (n.inputs || []).forEach(c => {
        if (!ids.has(c.source_node)) errors.push(`Node '${n.id}' references missing source '${c.source_node}'.`);
      });
      inputPortsFor(n).forEach(p => {
        const wired = (n.inputs || []).some(c => c.target_input === p.port);
        if (p.required && !wired) errors.push(`'${n.id}' requires input '${p.port}'.`);
      });
      if ((n.type === "llm") && (!n.inputs || n.inputs.length === 0)) errors.push(`LLM node '${n.id}' needs at least one input.`);
    });

    if (!start) errors.push("Graph must have exactly one Start node.");
    if (!end) errors.push("Graph must have exactly one Response node.");
    if (hasGraphCycle(nodes)) errors.push("Graph contains a cycle.");

    if (start && end && errors.length === 0) {
      const reachable = new Set([start.id]);
      const queue = [start.id];
      while (queue.length) {
        const u = queue.shift();
        nodes.forEach(n => (n.inputs || []).forEach(c => {
          if (c.source_node === u && !reachable.has(n.id)) {
            reachable.add(n.id);
            queue.push(n.id);
          }
        }));
      }
      if (!reachable.has(end.id)) errors.push("Response node is not reachable from Start.");
      nodes.forEach(n => {
        if (!reachable.has(n.id) && !isAnchor(n.type)) warnings.push(`Node '${n.id}' is not reachable from Start.`);
      });
    }

    if (end) {
      const wired = (end.inputs || []).some(c => c.target_input === "final_output");
      if (!wired) errors.push("Response node's 'final' input must be connected.");
    }

    return { ok: errors.length === 0, errors, warnings };
  }

  function hasGraphCycle(nodes) {
    const adj = {};
    nodes.forEach(n => { adj[n.id] = []; });
    nodes.forEach(n => (n.inputs || []).forEach(c => {
      if (adj[c.source_node]) adj[c.source_node].push(n.id);
    }));
    const WHITE = 0, GRAY = 1, BLACK = 2;
    const color = {};
    nodes.forEach(n => { color[n.id] = WHITE; });
    function dfs(u) {
      color[u] = GRAY;
      for (const v of adj[u]) {
        if (color[v] === GRAY) return true;
        if (color[v] === WHITE && dfs(v)) return true;
      }
      color[u] = BLACK;
      return false;
    }
    return nodes.some(n => color[n.id] === WHITE && dfs(n.id));
  }

  function loadWorkflowSelector() {
    const selector = document.getElementById("workflow-selector");
    if (!selector) return;
    fetch("/workflows.json")
      .then(res => res.json())
      .then(data => {
        const workflows = data.workflows || {};
        const active = data.active_workflow || "";
        let html = "";
        Object.keys(workflows).forEach(id => {
          const wf = workflows[id];
          const selected = id === active ? "selected" : "";
          html += `<option value="${id}" ${selected}>${wf.name || id}</option>`;
        });
        selector.innerHTML = html;
        selector.value = active;
      })
      .catch(err => console.error("Error loading workflow selector:", err));
  }

  function switchWorkflow(id) {
    if (!id) return;
    fetch("/api/workflows/activate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id })
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "Activation failed");
        return res.json();
      })
      .then(() => {
        const selector = document.getElementById("workflow-selector");
        if (selector) selector.value = id;
        const labPanel = document.getElementById("workflow-lab-surface");
        if (labPanel && !labPanel.classList.contains("hidden")) loadWorkflowsSchema();
      })
      .catch(err => {
        alert("Failed to switch workflow: " + err.message);
        loadWorkflowSelector();
      });
  }

  return {
    defFor,
    nodeColor,
    nodeIcon,
    isAnchor,
    inputPortsFor,
    outputPortsFor,
    loadWorkflowIntoModel,
    refreshLabSelector,
    autoLayout,
    persistPositions,
    renderGraph,
    drawEdges,
    deleteSelectedEdge,
    selectNode,
    setNodeProfile,
    toggleNodeTools,
    toggleNodeTool,
    updateNodeProp,
    renameNode,
    addLlmInput,
    removeLlmInput,
    renderInspector,
    buildSchemaFromModel,
    syncJsonFromModel,
    reloadGraphFromJson,
    toggleWorkflowJson,
    setValidation,
    toggleAddNodeMenu,
    autoLayoutNodes,
    addNode,
    deleteSelectedNode,
    initWorkflowLabEvents,
    loadWorkflowsSchema,
    compileWorkflowWithAI,
    saveWorkflowConfigurations,
    readSchemaFromEditor,
    switchLabWorkflow,
    newWorkflow,
    cloneWorkflow,
    deleteWorkflow,
    validateGraph,
    hasGraphCycle,
    loadWorkflowSelector,
    switchWorkflow,
  };
}
