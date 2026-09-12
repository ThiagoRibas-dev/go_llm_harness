import { appState } from "./state.js";
import { escapeHtml, toolCallTitle, toolResultTitle, populateKnownModelList } from "./helpers.js";
import { deliverablePathsFromTool, renderDeliverableChips, renderToolCallArgumentsBody, renderToolBody } from "./renderers.js";
import { createShellModule } from "./shell.js";
import { createChatModule } from "./chat.js";
import { createComposerModule } from "./composer.js";
import { createSessionsModule } from "./sessions.js";
import { createSettingsModule } from "./settings.js";
import { createWorkflowGraphModule } from "./workflow_graph.js";
import { createWorkflowManagerModule } from "./workflow_manager.js";
import { createEventsModule } from "./events.js";

const actions = {};

Object.assign(actions, createShellModule({
  state: appState,
  actions,
  escapeHtml,
  deliverablePathsFromTool,
  renderDeliverableChips,
  renderToolBody,
}));

Object.assign(actions, createChatModule({
  state: appState,
  actions,
  escapeHtml,
  toolCallTitle,
  toolResultTitle,
  deliverablePathsFromTool,
  renderDeliverableChips,
  renderToolCallArgumentsBody,
  renderToolBody,
}));

Object.assign(actions, createComposerModule({
  state: appState,
  actions,
  escapeHtml,
}));

Object.assign(actions, createSessionsModule({
  state: appState,
  actions,
  escapeHtml,
}));

Object.assign(actions, createSettingsModule({
  state: appState,
  actions,
  escapeHtml,
  populateKnownModelList,
}));

Object.assign(actions, createWorkflowGraphModule({
  state: appState,
  actions,
  escapeHtml,
}));

Object.assign(actions, createWorkflowManagerModule({
  state: appState,
  actions,
}));

Object.assign(actions, createEventsModule({
  state: appState,
  actions,
}));

window.addEventListener("DOMContentLoaded", () => {
  actions.applyThemeChrome();
  actions.fetchConfig();
  actions.refreshWorkspaceTree();
  actions.fetchWorkspaces();
  actions.fetchSessions();
  actions.fetchPinnedFiles();
  actions.loadWorkflowSelector();
  actions.connectSSE();
  actions.appendGreeting();
  actions.switchPrimarySurface("console");
  actions.switchSidebarTab("sessions");
  actions.switchConversationView("chat");
  actions.switchDetailsTab("trajectory");
  actions.initShellResizers();
  actions.initWorkflowLabEvents();
  actions.syncRailButtons();
});

function exposeInlineHandlerFunctions() {
  const names = [
    "toggleSidebar", "openSettingsModal", "activateRailSection", "switchPrimarySurface", "toggleDetailsPanel", "triggerNewSession", "triggerFileUpload", "uploadSelectedFile", "addNewWorkspace", "createSnapshot", "triggerCompaction", "switchWorkflow", "switchConversationView", "switchDetailsTab", "submitPrompt", "handleComposerShellClick", "handleInputKeydown", "handleComposerInput", "triggerReroll", "runComposerCta", "compileWorkflowWithAI", "toggleAddNodeMenu", "addNode", "autoLayoutNodes", "reloadGraphFromJson", "switchLabWorkflow", "newWorkflow", "cloneWorkflow", "deleteWorkflow", "toggleWorkflowJson", "saveWorkflowConfigurations", "switchSettingsTab", "closeSettingsModal", "saveSettings", "onChatProfileChange", "suggestBaseURL", "onCompactProfileChange", "suggestCompactBaseURL", "addIgnorePattern", "addCollapsePattern", "addMCPServer", "newProviderForm", "setActiveProfile", "closeProviderForm", "suggestProviderBaseUrl", "saveProvider", "closeForkModal", "toggleForkFields", "executeForkAction", "stageWorkspaceFile", "switchSidebarTab", "openWorkspaceFile", "editQueuedMessage", "removeQueuedMessage", "clearStagedContext", "removeStagedContext", "changeWorkspaceFromSelector", "removeWorkspaceFromHistory", "selectSession", "renameSessionPrompt", "deleteSessionConfirm", "seedPromptExample", "enableCardEdit", "triggerFork", "toggleCardMetrics", "toggleWfPreview", "setNodeProfile", "updateNodeProp", "toggleNodeTools", "toggleNodeTool", "addLlmInput", "removeLlmInput", "renameNode", "deleteSelectedNode", "removeIgnorePattern", "removeCollapsePattern", "revertToSnapshot", "deleteSnapshot", "deleteMCPServer", "removeContextPin", "saveAndBranchCard", "cancelCardEdit", "editProvider", "deleteProvider"
  ];
  for (const name of names) {
    if (typeof actions[name] === "function") window[name] = actions[name];
  }
}

exposeInlineHandlerFunctions();
