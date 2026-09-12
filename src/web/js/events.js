import { currentSessionId } from "./state.js";

export function createEventsModule({ state, actions }) {
  function connectSSE() {
    if (state.sseSource) state.sseSource.close();
    state.sseSource = new EventSource("/stream");

    state.sseSource.onmessage = (event) => {
      const packet = JSON.parse(event.data);
      handleIncomingEvent(packet);
    };

    state.sseSource.onerror = () => {
      const dot = document.getElementById("status-dot");
      const text = document.getElementById("status-text");
      if (dot) dot.className = "w-2.5 h-2.5 rounded-full bg-amber-500 animate-pulse";
      if (text) text.innerText = "Reconnecting...";
      setTimeout(connectSSE, 2000);
    };

    state.sseSource.onopen = () => {
      const dot = document.getElementById("status-dot");
      const text = document.getElementById("status-text");
      if (dot) dot.className = "w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse";
      if (text) text.innerText = "Connected";
    };
  }

  function handleIncomingEvent(packet) {
    if (packet.event === "session_init") {
      document.getElementById("session-id").innerText = "Session: " + packet.data.session_id;
      state.trajectoryEvents = [];
      state.subagentRegistry = {};
      state.deliverableRegistry = [];
      state.currentFilePreview = null;
      state.currentToolPreview = null;
      actions.restoreSessionUIState && actions.restoreSessionUIState(packet.data.session_id);
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent("session", "session_init", `Active session is now ${packet.data.session_id}.`);
      actions.renderSubagentsView && actions.renderSubagentsView();
      actions.renderDeliverablesPanel && actions.renderDeliverablesPanel();
      actions.renderFilePanel && actions.renderFilePanel();
      actions.renderToolPanel && actions.renderToolPanel();
      actions.fetchConfig && actions.fetchConfig();
    }

    if (packet.event === "run_state") {
      if (packet.data && packet.data.session_id === currentSessionId()) {
        actions.recordTrajectoryEvent && actions.recordTrajectoryEvent(
          "run",
          packet.data.running ? "run_state:start" : "run_state:end",
          packet.data.running ? "Session execution started." : "Session execution completed."
        );
        actions.fetchConfig && actions.fetchConfig();
      }
    }

    if (packet.event === "turn_secured") {
      const turnData = packet.data;
      actions.appendTurnToChat && actions.appendTurnToChat(turnData);
      actions.extractDeliverablesFromTool && actions.extractDeliverablesFromTool(turnData);
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent(
        turnData.role || "turn",
        `turn_secured:${turnData.role || "message"}`,
        (turnData.name ? `${turnData.name} · ` : "") + ((turnData.content || "").slice(0, 140) || "(no content)")
      );
      actions.refreshWorkspaceTree && actions.refreshWorkspaceTree();
    }

    if (packet.event === "compaction") {
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent("compaction", "compaction", `Compacted history up to turn ${packet.data.boundary_turn}.`);
      actions.appendSystemAlert && actions.appendSystemAlert("Compaction Engaged", `Context compacted up to Turn ${packet.data.boundary_turn}. History older than this turn has been summarized to save tokens.`, "fa-compress text-indigo-400");
    }

    if (packet.event === "cost_update") {
      state.activeCost += parseFloat(packet.data.cost);
      document.getElementById("session-cost").innerText = "$" + state.activeCost.toFixed(4);

      let currentTokens = parseInt(document.getElementById("session-tokens").innerText, 10) || 0;
      currentTokens += packet.data.total_tokens || 0;
      document.getElementById("session-tokens").innerText = currentTokens;
      document.getElementById("session-chars").innerText = `(${packet.data.char_count} chars)`;

      const lastTurnCostId = `metric-cost-${state.turnCounter}`;
      const lastTokensCardId = `metric-tokens-${state.turnCounter}`;
      if (document.getElementById(lastTurnCostId)) {
        document.getElementById(lastTurnCostId).innerText = "$" + parseFloat(packet.data.cost).toFixed(6);
        document.getElementById(lastTokensCardId).innerText = `${packet.data.prompt_tokens} In / ${packet.data.completion_tokens} Out (${packet.data.total_tokens} total)`;
      }
    }

    if (packet.event === "workflow_start") {
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent("workflow", "workflow_start", packet.data.name || packet.data.workflow_id || "workflow started");
      actions.renderWorkflowTrace && actions.renderWorkflowTrace(packet.data);
    }
    if (packet.event === "workflow_node") {
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent("workflow_node", `${packet.data.node_id || "node"} · ${packet.data.status || "update"}`, packet.data.preview || packet.data.error || packet.data.label || packet.data.type || "");
      actions.updateWorkflowNode && actions.updateWorkflowNode(packet.data);
    }
    if (packet.event === "workflow_end") {
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent("workflow", `workflow_${packet.data.status || "end"}`, packet.data.name || packet.data.workflow_id || "workflow finished");
      actions.finalizeWorkflowTrace && actions.finalizeWorkflowTrace(packet.data);
    }

    if (packet.event === "subagent_start") {
      actions.updateSubagentRegistry && actions.updateSubagentRegistry(packet.data, "running");
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent("subagent", "subagent_start", packet.data.description || packet.data.session_id || "sub-agent started");
      actions.renderSubAgentRow && actions.renderSubAgentRow(packet.data, "running");
    }
    if (packet.event === "subagent_done") {
      actions.updateSubagentRegistry && actions.updateSubagentRegistry(packet.data, "done");
      actions.recordTrajectoryEvent && actions.recordTrajectoryEvent("subagent", "subagent_done", packet.data.description || packet.data.session_id || "sub-agent finished");
      actions.renderSubAgentRow && actions.renderSubAgentRow(packet.data, "done");
    }
  }

  return {
    connectSSE,
    handleIncomingEvent,
  };
}
