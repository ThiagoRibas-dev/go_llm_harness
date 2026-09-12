export const DEFAULT_PROMPT_PLACEHOLDER = "Type a prompt to solve (e.g., 'Write a python calculation script and test it')....";

export const appState = {
  sseSource: null,
  activeCost: 0.0,
  turnCounter: 0,
  isSidebarCollapsed: false,
  selectedForkTurn: 0,
  activeWorkspaceDir: "",
  currentPrimarySurface: "console",
  currentSidebarSection: "sessions",
  currentConversationView: "chat",
  currentDetailsTab: "trajectory",
  detailsPanelOpen: true,
  detailsPanelWidth: 340,
  currentUIState: { composer_enabled: true, status: "ready", title: "GoHarness is ready", summary: "" },
  workspaceTreeEntries: [],
  workspaceCollapsedDirs: {},
  trajectoryEvents: [],
  subagentRegistry: {},
  deliverableRegistry: [],
  currentFilePreview: null,
  currentToolPreview: null,
  sessionQueues: {},
  stagedContextBySession: {},
  queueItemSeq: 1,
  queueDispatchInFlight: false,
  currentSettingsRevision: 0,
  currentProvidersRevision: 0,
  currentIgnoredPatterns: [],
  currentCollapsedPatterns: [],
  activePinnedFiles: [],
  originalCardHtmls: {},
  editingProviderName: null,
  wfProfiles: null,
  wfModel: { workflowId: null, workflow: null, nodes: [], selected: null },
};

export function currentSessionId() {
  const el = document.getElementById("session-id");
  return el ? el.innerText.replace("Session: ", "").trim() : "";
}

export function currentSessionKey() {
  return currentSessionId() || "pending-session";
}

export function storageKey(kind, sessionKey) {
  return `goharness:${kind}:${sessionKey}`;
}
