export function createWorkflowValidationModule({ state, actions, escapeHtml }) {
  function buildSchemaFromModel() {
    let schema;
    try {
      schema = JSON.parse(document.getElementById("workflow-json-editor").value || "{}");
    } catch {
      schema = { workflows: {} };
    }
    schema.workflows = schema.workflows || {};
    if (!state.wfModel || !state.wfModel.workflowId) return schema;

    const nodes = (state.wfModel.nodes || []).map(n => {
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
    schema.workflows[state.wfModel.workflowId] = Object.assign({}, state.wfModel.workflow, { nodes });
    schema.active_workflow = state.wfModel.workflowId;
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
      actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(schema);
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
      actions.inputPortsFor(n).forEach(p => {
        const wired = (n.inputs || []).some(c => c.target_input === p.port);
        if (p.required && !wired) errors.push(`'${n.id}' requires input '${p.port}'.`);
      });
      if (n.type === "llm" && (!n.inputs || n.inputs.length === 0)) errors.push(`LLM node '${n.id}' needs at least one input.`);
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
        if (!reachable.has(n.id) && !actions.isAnchor(n.type)) warnings.push(`Node '${n.id}' is not reachable from Start.`);
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

  return {
    buildSchemaFromModel,
    syncJsonFromModel,
    reloadGraphFromJson,
    toggleWorkflowJson,
    setValidation,
    validateGraph,
    hasGraphCycle,
  };
}
