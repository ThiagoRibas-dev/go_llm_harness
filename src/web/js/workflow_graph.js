import { createWorkflowCanvasModule } from "./workflow_canvas.js";
import { createWorkflowInspectorModule } from "./workflow_inspector.js";
import { createWorkflowValidationModule } from "./workflow_validation.js";

export function createWorkflowGraphModule({ state, actions, escapeHtml }) {
  const canvas = createWorkflowCanvasModule({ state, actions, escapeHtml });
  Object.assign(actions, canvas);

  const validation = createWorkflowValidationModule({ state, actions, escapeHtml });
  Object.assign(actions, validation);

  const inspector = createWorkflowInspectorModule({ state, actions, escapeHtml });
  Object.assign(actions, inspector);

  let bindingsInitialized = false;

  function parseInspectorValue(el) {
    const type = el.getAttribute("data-value-type") || "string";
    if (type === "checked-bool") return !!el.checked;
    if (type === "int") return parseInt(el.value, 10) || 0;
    if (type === "float") return parseFloat(el.value) || 0;
    return el.value;
  }

  function initWorkflowBindings() {
    if (bindingsInitialized) return;
    bindingsInitialized = true;

    document.getElementById("workflow-ai-compile-btn")?.addEventListener("click", () => actions.compileWorkflowWithAI && actions.compileWorkflowWithAI());
    document.getElementById("workflow-add-node-toggle")?.addEventListener("click", canvas.toggleAddNodeMenu);
    document.getElementById("workflow-auto-layout-btn")?.addEventListener("click", canvas.autoLayoutNodes);
    document.getElementById("workflow-reload-json-btn")?.addEventListener("click", validation.reloadGraphFromJson);
    document.getElementById("wf-lab-selector")?.addEventListener("change", (e) => actions.switchLabWorkflow && actions.switchLabWorkflow(e.target.value));
    document.getElementById("workflow-new-btn")?.addEventListener("click", () => actions.newWorkflow && actions.newWorkflow());
    document.getElementById("workflow-clone-btn")?.addEventListener("click", () => actions.cloneWorkflow && actions.cloneWorkflow());
    document.getElementById("workflow-delete-btn")?.addEventListener("click", () => actions.deleteWorkflow && actions.deleteWorkflow());
    document.getElementById("workflow-json-toggle")?.addEventListener("click", validation.toggleWorkflowJson);
    document.getElementById("workflow-apply-btn")?.addEventListener("click", () => actions.saveWorkflowConfigurations && actions.saveWorkflowConfigurations());
    document.getElementById("workflow-selector")?.addEventListener("change", (e) => actions.switchWorkflow && actions.switchWorkflow(e.target.value));

    document.getElementById("add-node-menu")?.addEventListener("click", (e) => {
      const btn = e.target.closest("[data-add-node-kind]");
      if (!btn) return;
      canvas.addNode(btn.getAttribute("data-add-node-kind"));
    });

    document.getElementById("wf-inspector-body")?.addEventListener("change", (e) => {
      const el = e.target.closest("[data-action]");
      if (!el) return;
      const action = el.getAttribute("data-action");
      const nodeId = el.getAttribute("data-node-id");
      if (action === "rename-node") inspector.renameNode(nodeId, el.value);
      else if (action === "set-node-profile") inspector.setNodeProfile(nodeId, el.value);
      else if (action === "update-node-prop") inspector.updateNodeProp(nodeId, el.getAttribute("data-prop"), parseInspectorValue(el));
      else if (action === "toggle-node-tools") inspector.toggleNodeTools(nodeId, el.checked);
      else if (action === "toggle-node-tool") inspector.toggleNodeTool(nodeId, el.getAttribute("data-tool-name"), el.checked);
    });

    document.getElementById("wf-inspector-body")?.addEventListener("click", (e) => {
      const el = e.target.closest("[data-action]");
      if (!el) return;
      const action = el.getAttribute("data-action");
      const nodeId = el.getAttribute("data-node-id");
      if (action === "add-llm-input") inspector.addLlmInput(nodeId);
      else if (action === "remove-llm-input") inspector.removeLlmInput(nodeId, el.getAttribute("data-port-name"));
      else if (action === "delete-selected-node") canvas.deleteSelectedNode();
    });
  }

  return {
    ...canvas,
    ...validation,
    ...inspector,
    initWorkflowBindings,
  };
}
