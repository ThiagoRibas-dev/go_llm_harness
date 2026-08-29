package main

import (
	"fmt"
	"strings"
)

type UIState struct {
	ComposerEnabled bool   `json:"composer_enabled"`
	Status          string `json:"status"`
	BlockedReason   string `json:"blocked_reason,omitempty"`
	CTAAction       string `json:"cta_action,omitempty"`
	CTALabel        string `json:"cta_label,omitempty"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
}

func buildUIState(sessionID string) UIState {
	state := UIState{
		ComposerEnabled: true,
		Status:          "ready",
		Title:           "GoHarness is ready",
		Summary:         "Ask it to inspect code, patch files, run commands, or drive a workflow.",
	}
	if isSessionRunning(sessionID) {
		state.ComposerEnabled = false
		state.Status = "busy"
		state.BlockedReason = "GoHarness is already executing this session. Wait for the current run to finish."
		state.Title = "Current session is running"
		state.Summary = "Live tool output and assistant replies will continue streaming below until the run finishes."
		return state
	}
	if strings.TrimSpace(activeConfig.Agent.WorkspaceDir) == "" {
		state.ComposerEnabled = false
		state.Status = "blocked"
		state.BlockedReason = "Select a workspace first."
		state.CTAAction = "open_workspace"
		state.CTALabel = "Choose workspace"
		state.Title = "Workspace required"
		state.Summary = "GoHarness needs an active workspace before it can read files, patch code, or run commands."
		return state
	}
	if profile := strings.TrimSpace(activeConfig.ProviderProfile); profile != "" {
		if _, err := GetProvider(profile); err != nil {
			state.ComposerEnabled = false
			state.Status = "blocked"
			state.BlockedReason = fmt.Sprintf("Selected provider profile '%s' no longer exists.", profile)
			state.CTAAction = "open_settings"
			state.CTALabel = "Fix provider"
			state.Title = "Provider profile missing"
			state.Summary = "Pick another saved profile or switch back to inline model settings."
			return state
		}
	} else if strings.TrimSpace(activeConfig.API.Model) == "" {
		state.ComposerEnabled = false
		state.Status = "blocked"
		state.BlockedReason = "Select a model first."
		state.CTAAction = "open_settings"
		state.CTALabel = "Choose model"
		state.Title = "Model required"
		state.Summary = "The runtime will not pretend this is configurable if the model is blank. Set one in Settings first."
		return state
	}
	state.Summary = fmt.Sprintf("Workspace: %s", activeConfig.Agent.WorkspaceDir)
	if profile := strings.TrimSpace(activeConfig.ProviderProfile); profile != "" {
		state.Summary += fmt.Sprintf(" · Profile: @%s", profile)
	} else if model := strings.TrimSpace(activeConfig.API.Model); model != "" {
		state.Summary += fmt.Sprintf(" · Model: %s", model)
	}
	return state
}
