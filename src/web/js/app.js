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
  actions.initShellBindings();
  actions.initChatBindings();
  actions.initComposerBindings();
  actions.initSessionsBindings();
  actions.initSettingsBindings();
  actions.initWorkflowBindings();

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
