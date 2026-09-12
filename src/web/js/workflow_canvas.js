export function createWorkflowCanvasModule({ state, actions, escapeHtml }) {
  const NODE_W = 200;
  const NODE_H = 78;
  const COL_GAP = 70;
  const ROW_GAP = 28;
  const PIN_R = 6;

  const NODE_DEFS = {
    user_input: {
      title: "Start", icon: "fa-sign-in-alt", color: "amber",
      inputs: [], outputs: [{ port: "prompt", label: "prompt" }], anchor: true,
    },
    assistant_response: {
      title: "Response", icon: "fa-sign-out-alt", color: "emerald",
      inputs: [{ port: "final_output", label: "final", required: true }], outputs: [], anchor: true,
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

  let workflowLabEventsBound = false;
  state.wfModel = state.wfModel || { workflowId: null, workflow: null, nodes: [], selected: null };

  function model() { return state.wfModel; }
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
      ((node.properties && node.properties.input_ports) || []).forEach(name => {
        if (name && !seen.has(name)) { seen.add(name); ports.push({ port: name, label: name }); }
      });
      (node.inputs || []).forEach(c => {
        if (c.target_input && !seen.has(c.target_input)) { seen.add(c.target_input); ports.push({ port: c.target_input, label: c.target_input }); }
      });
      if (ports.length === 0) ports.push({ port: "prompt", label: "prompt" });
      return ports;
    }
    return d.inputs || [];
  }

  function outputPortsFor(node) {
    if (node.type === "llm" || defFor(node.type).aliasOf === "llm") return [{ port: "response", label: "response" }];
    return defFor(node.type).outputs || [];
  }

  function wfCanvas() { return document.getElementById("wf-canvas"); }
  function wfScroll() { return document.getElementById("wf-canvas-scroll"); }
  function wfEdgesSvg() { return document.getElementById("wf-edges"); }
  function nodeById(id) { return (model().nodes || []).find(n => n.id === id); }

  function loadWorkflowIntoModel(schema, workflowId) {
    const ids = Object.keys(schema.workflows || {});
    const activeId = workflowId || schema.active_workflow || ids[0];
    const wf = schema.workflows ? schema.workflows[activeId] : null;
    const rawNodes = (wf && wf.nodes) ? JSON.parse(JSON.stringify(wf.nodes)) : [];
    const nodes = rawNodes.map(n => {
      if (n.type === "llm_query" || n.type === "llm_synthesis") n.type = "llm";
      n.properties = n.properties || {};
      n.inputs = n.inputs || [];
      if (typeof n.properties.x !== "number" || typeof n.properties.y !== "number") n._needsLayout = true;
      else { n.x = n.properties.x; n.y = n.properties.y; }
      return n;
    });
    state.wfModel = { workflowId: activeId, workflow: wf, nodes, selected: null };
    if (nodes.some(n => n._needsLayout)) autoLayout();
    renderGraph();
    refreshLabSelector(schema);
  }

  function refreshLabSelector(schema) {
    const sel = document.getElementById("wf-lab-selector");
    if (!sel) return;
    const wfs = (schema && schema.workflows) || {};
    const current = model().workflowId;
    let html = "";
    Object.keys(wfs).forEach(id => {
      const name = wfs[id].name || id;
      html += `<option value="${escapeHtml(id)}" ${id === current ? "selected" : ""}>${escapeHtml(name)} (${escapeHtml(id)})</option>`;
    });
    sel.innerHTML = html;
    sel.value = current || "";
  }

  function autoLayout() {
    const nodes = model().nodes;
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
    nodes.forEach(n => { const d = depth[n.id]; (cols[d] = cols[d] || []).push(n); });
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
    (model().nodes || []).forEach(n => {
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
    if (activeNameEl && model().workflow) activeNameEl.textContent = `${model().workflowId} — ${model().workflow.name || model().workflowId}`;

    let maxX = 0, maxY = 0;
    (model().nodes || []).forEach(n => {
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
    (model().nodes || []).forEach(n => canvas.appendChild(renderNodeCard(n)));
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
    if (model().selected && model().selected.kind === "node" && model().selected.id === n.id) card.classList.add("ring-2", "ring-blue-400");

    const inPorts = inputPortsFor(n);
    const outPorts = outputPortsFor(n);
    const label = n.properties && n.properties.provider_profile ? "@" + n.properties.provider_profile : (n.properties && n.properties.model ? n.properties.model : (def.title || n.type));

    card.innerHTML = `
      <div class="flex items-center justify-between px-2 py-1 border-b border-slate-700/60 rounded-t-lg bg-slate-800/60 cursor-move">
        <span class="font-bold text-slate-100 text-[11px] truncate flex items-center"><i class="fa-solid ${nodeIcon(n.type)} mr-1.5 ${iconCls}"></i>${escapeHtml(n.id)}</span>
        <span class="text-[8px] uppercase tracking-wide text-slate-400 bg-slate-900/70 px-1 rounded">${escapeHtml(n.type)}</span>
      </div>
      <div class="px-2 py-1 text-[10px] text-slate-400 truncate">${escapeHtml(label)}</div>
      <div class="relative px-2 pb-1 flex justify-between text-[9px] text-slate-500">
        <div class="wf-inputs space-y-0.5">
          ${inPorts.map(p => `
            <div class="flex items-center">
              <span class="wf-pin wf-pin-in -ml-3.5 mr-1 inline-block rounded-full bg-blue-500 border-2 border-slate-900" style="width:${PIN_R*2}px;height:${PIN_R*2}px" data-node="${escapeHtml(n.id)}" data-port="${escapeHtml(p.port)}" data-kind="in" title="${escapeHtml(p.label)}"></span>
              <span class="truncate">${escapeHtml(p.label)}</span>
            </div>`).join("")}
        </div>
        <div class="wf-outputs space-y-0.5 text-right">
          ${outPorts.map(p => `
            <div class="flex items-center justify-end">
              <span class="truncate">${escapeHtml(p.label)}</span>
              <span class="wf-pin wf-pin-out -mr-3.5 ml-1 inline-block rounded-full bg-emerald-400 border-2 border-slate-900 cursor-crosshair" style="width:${PIN_R*2}px;height:${PIN_R*2}px" data-node="${escapeHtml(n.id)}" data-port="${escapeHtml(p.port)}" data-kind="out" title="${escapeHtml(p.label)}"></span>
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
    return { x: r.left + r.width / 2 - cr.left, y: r.top + r.height / 2 - cr.top };
  }

  function bezierPath(x1, y1, x2, y2) {
    const dx = Math.max(40, Math.abs(x2 - x1) * 0.5);
    return `M ${x1},${y1} C ${x1 + dx},${y1} ${x2 - dx},${y2} ${x2},${y2}`;
  }

  function drawEdges() {
    const svg = wfEdgesSvg();
    if (!svg) return;
    svg.innerHTML = "";
    const sel = model().selected;
    (model().nodes || []).forEach(n => {
      (n.inputs || []).forEach(c => {
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
      if (old.source_node === fromNode && old.source_output === fromPort) return { ok: false, error: "That connection already exists." };
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
    (model().nodes || []).forEach(n => { adj[n.id] = []; });
    (model().nodes || []).forEach(n => (n.inputs || []).forEach(c => {
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
    const sel = model().selected;
    if (!sel || sel.kind !== "edge") return;
    const tgt = nodeById(sel.to);
    if (tgt && tgt.inputs) {
      const idx = findEdgeIndex(sel.from, sel.fromPort, sel.to, sel.toPort);
      if (idx !== -1) tgt.inputs.splice(idx, 1);
    }
    model().selected = null;
    renderGraph();
    actions.syncJsonFromModel();
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
      canvas.querySelectorAll(".wf-pin-in").forEach(p => { p.style.boxShadow = on ? "0 0 0 3px rgba(56,189,248,0.35)" : ""; });
    };
    highlightPins(true);
    const eventPos = (ev) => { const cr = canvas.getBoundingClientRect(); return { x: ev.clientX - cr.left, y: ev.clientY - cr.top }; };
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
        if (!res.ok) actions.setValidation(false, res.error);
        else {
          actions.setValidation(true, "Connected.");
          actions.syncJsonFromModel();
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
      actions.syncJsonFromModel();
      renderGraph();
      selectNode(node.id);
    };
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }

  function selectNode(id) {
    model().selected = { kind: "node", id };
    document.querySelectorAll(".wf-node").forEach(el => {
      el.classList.toggle("ring-2", el.dataset.id === id);
      el.classList.toggle("ring-blue-400", el.dataset.id === id);
    });
    actions.renderInspector(id);
  }

  function toggleAddNodeMenu() {
    document.getElementById("add-node-menu")?.classList.toggle("hidden");
  }

  function autoLayoutNodes() {
    autoLayout();
    renderGraph();
    actions.syncJsonFromModel();
    actions.setValidation(true, "Auto-arranged nodes.");
  }

  function uniqueNodeId(type) {
    if (!nodeById(type)) return type;
    let n = 1;
    while (nodeById(`${type}_${n}`)) n++;
    return `${type}_${n}`;
  }

  function nextNodePosition() {
    if (!model().nodes.length) return { x: 40, y: 40 };
    let maxRight = 0;
    model().nodes.forEach(n => { maxRight = Math.max(maxRight, (n.x || 0) + NODE_W); });
    return { x: maxRight + COL_GAP, y: 40 };
  }

  function addNode(type) {
    document.getElementById("add-node-menu")?.classList.add("hidden");
    const def = NODE_DEFS[type];
    if (!def || def.anchor) return;
    const id = uniqueNodeId(type);
    const pos = nextNodePosition();
    const node = { id, type, x: pos.x, y: pos.y, properties: Object.assign({}, def.defaults || {}), inputs: [] };
    model().nodes.push(node);
    model().selected = { kind: "node", id };
    persistPositions();
    renderGraph();
    actions.syncJsonFromModel();
    selectNode(id);
    actions.setValidation(true, `Added ${type} node '${id}'.`);
  }

  function deleteSelectedNode() {
    const sel = model().selected;
    if (!sel || sel.kind !== "node") return;
    const n = nodeById(sel.id);
    if (!n || isAnchor(n.type)) return;
    state.wfModel.nodes = model().nodes.filter(x => x.id !== sel.id);
    model().nodes.forEach(other => { other.inputs = (other.inputs || []).filter(c => c.source_node !== sel.id); });
    model().selected = null;
    renderGraph();
    actions.syncJsonFromModel();
    actions.renderInspector(null);
    actions.setValidation(true, `Deleted node '${sel.id}'.`);
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
        model().selected = { kind: "edge", from: g.dataset.from, fromPort: g.dataset.fromPort, to: g.dataset.to, toPort: g.dataset.toPort };
        document.querySelectorAll(".wf-node").forEach(el => { el.classList.remove("ring-2", "ring-blue-400"); });
        actions.renderInspector(null);
        drawEdges();
      } else if (!e.target.closest(".wf-node") && !e.target.closest("#wf-inspector") && !e.target.closest("button") && !e.target.closest("input") && !e.target.closest("textarea") && !e.target.closest("select")) {
        model().selected = null;
        document.querySelectorAll(".wf-node").forEach(el => { el.classList.remove("ring-2", "ring-blue-400"); });
        actions.renderInspector(null);
        drawEdges();
      }
    });
    document.addEventListener("keydown", (e) => {
      const panel = document.getElementById("workflow-lab-surface");
      if (!panel || panel.classList.contains("hidden")) return;
      const tag = (e.target.tagName || "").toLowerCase();
      if (tag === "input" || tag === "textarea" || tag === "select") return;
      if (e.key === "Delete" || e.key === "Backspace") {
        if (model().selected && model().selected.kind === "edge") {
          e.preventDefault();
          deleteSelectedEdge();
        } else if (model().selected && model().selected.kind === "node") {
          const n = nodeById(model().selected.id);
          if (n && !isAnchor(n.type)) {
            e.preventDefault();
            deleteSelectedNode();
          }
        }
      }
    });
    document.addEventListener("click", (e) => {
      const menu = document.getElementById("add-node-menu");
      if (menu && !menu.classList.contains("hidden") && !e.target.closest("#add-node-menu") && !e.target.closest("#workflow-add-node-toggle")) menu.classList.add("hidden");
    });
  }

  return {
    defFor,
    nodeColor,
    nodeIcon,
    isAnchor,
    inputPortsFor,
    outputPortsFor,
    wfCanvas,
    wfScroll,
    wfEdgesSvg,
    nodeById,
    loadWorkflowIntoModel,
    refreshLabSelector,
    autoLayout,
    persistPositions,
    renderGraph,
    drawEdges,
    findEdgeIndex,
    addEdge,
    createsCycle,
    deleteSelectedEdge,
    selectNode,
    toggleAddNodeMenu,
    autoLayoutNodes,
    uniqueNodeId,
    nextNodePosition,
    addNode,
    deleteSelectedNode,
    initWorkflowLabEvents,
  };
}
