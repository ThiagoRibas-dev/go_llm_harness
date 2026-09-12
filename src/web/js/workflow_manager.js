export function createWorkflowManagerModule({ state, actions }) {
  function loadWorkflowsSchema() {
    fetch("/workflows.json")
      .then(res => res.json())
      .then(data => {
        document.getElementById("workflow-json-editor").value = JSON.stringify(data, null, 2);
        actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(data);
        actions.setValidation && actions.setValidation(true, "Graph loaded.");
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
        actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(defaultSchema);
      });
  }

  function compileWorkflowWithAI() {
    const promptInput = document.getElementById("ai-workflow-prompt");
    const prompt = promptInput.value.trim();
    if (!prompt) return;
    actions.setValidation && actions.setValidation(false, "AI Compiler compiling graph...");

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
      actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(schema);
      promptInput.value = "";
      actions.appendSystemAlert && actions.appendSystemAlert("AI Workflow Staged", "Generated a node-graph draft. Review it and click Compile & Apply to save.", "fa-wand-magic-sparkles text-blue-400");
    }, 1200);
  }

  function saveWorkflowConfigurations() {
    actions.persistPositions && actions.persistPositions();
    const schema = actions.buildSchemaFromModel ? actions.buildSchemaFromModel() : {};
    const v = actions.validateGraph ? actions.validateGraph(state.wfModel) : { ok: true, warnings: [], errors: [] };
    if (!v.ok) {
      actions.setValidation && actions.setValidation(false, v.errors.join(" "));
      alert("Cannot apply: " + v.errors.join(" "));
      return;
    }
    if (v.warnings.length) {
      actions.setValidation && actions.setValidation(true, v.warnings.join(" "));
    } else {
      actions.setValidation && actions.setValidation(true, "Graph valid.");
    }
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
        actions.setValidation && actions.setValidation(true, "Saved.");
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
    actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(schema, id);
    actions.setValidation && actions.setValidation(true, `Editing '${id}'. Click Compile & Apply to save/activate.`);
  }

  function newWorkflow() {
    const schema = readSchemaFromEditor();
    if (!schema) return;
    schema.workflows = schema.workflows || {};
    const id = uniqueWorkflowId(schema, "new_workflow");
    const display = prompt("Name for the new workflow (id):", id);
    if (display === null) return;
    const clean = (display.trim() || id).replace(/\s+/g, "_").replace(/[^a-zA-Z0-9_-]/g, "");
    if (!clean) {
      alert("Invalid workflow id.");
      return;
    }
    if (schema.workflows[clean]) {
      alert("A workflow with that id already exists.");
      return;
    }
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
    actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(schema, clean);
    actions.setValidation && actions.setValidation(true, `New workflow '${clean}' staged. Click Compile & Apply to save.`);
  }

  function cloneWorkflow() {
    const schema = readSchemaFromEditor();
    if (!schema) return;
    const srcId = state.wfModel && state.wfModel.workflowId;
    if (!srcId || !schema.workflows || !schema.workflows[srcId]) {
      alert("No workflow selected to clone.");
      return;
    }
    const cloneId = uniqueWorkflowId(schema, srcId + "_clone");
    const display = prompt("Name for the cloned workflow (id):", cloneId);
    if (display === null) return;
    const clean = (display.trim() || cloneId).replace(/\s+/g, "_").replace(/[^a-zA-Z0-9_-]/g, "");
    if (!clean) {
      alert("Invalid workflow id.");
      return;
    }
    if (schema.workflows[clean]) {
      alert("A workflow with that id already exists.");
      return;
    }
    schema.workflows[clean] = JSON.parse(JSON.stringify(schema.workflows[srcId]));
    schema.workflows[clean].name = (schema.workflows[clean].name || srcId) + " (copy)";
    document.getElementById("workflow-json-editor").value = JSON.stringify(schema, null, 2);
    actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(schema, clean);
    actions.setValidation && actions.setValidation(true, `Cloned '${srcId}' to '${clean}'. Click Compile & Apply to save.`);
  }

  function deleteWorkflow() {
    const schema = readSchemaFromEditor();
    if (!schema) return;
    const id = state.wfModel && state.wfModel.workflowId;
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
    actions.loadWorkflowIntoModel && actions.loadWorkflowIntoModel(schema, schema.active_workflow);
    actions.setValidation && actions.setValidation(true, `Deleted '${id}'. Click Compile & Apply to save.`);
  }

  function uniqueWorkflowId(schema, base) {
    if (!schema.workflows[base]) return base;
    let n = 2;
    while (schema.workflows[`${base}_${n}`]) n++;
    return `${base}_${n}`;
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
    loadWorkflowsSchema,
    compileWorkflowWithAI,
    saveWorkflowConfigurations,
    readSchemaFromEditor,
    switchLabWorkflow,
    newWorkflow,
    cloneWorkflow,
    deleteWorkflow,
    uniqueWorkflowId,
    loadWorkflowSelector,
    switchWorkflow,
  };
}
