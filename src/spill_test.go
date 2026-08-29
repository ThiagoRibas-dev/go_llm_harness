package main

import (
	"os"
	"strings"
	"testing"
)

func TestMaybeSpillToolResultWritesSessionSpill(t *testing.T) {
	sessionID := "spill_test_case"
	t.Cleanup(func() { _ = os.RemoveAll(GetSystemPath(".goharness/sessions/" + sessionID)) })
	a := &Agent{SessionID: sessionID, Workspace: t.TempDir()}
	big := strings.Repeat("alpha beta gamma\n", 1800)

	result := maybeSpillToolResult(a, "execute_command", big)
	if !strings.Contains(result, "=== TOOL OUTPUT SPILLED TO DISK ===") {
		t.Fatalf("expected spill header, got:\n%s", result)
	}
	id := hashSpillID(big)
	if _, err := os.Stat(spillDataPath(sessionID, id)); err != nil {
		t.Fatalf("expected spill data file for %s: %v", id, err)
	}
	if _, err := os.Stat(spillMetaPath(sessionID, id)); err != nil {
		t.Fatalf("expected spill meta file for %s: %v", id, err)
	}

	page := readSpill(a, id, 0, 80, "")
	if !strings.Contains(page, "spill_id: "+id) {
		t.Fatalf("expected page to reference spill id, got:\n%s", page)
	}
	if !strings.Contains(page, "alpha beta gamma") {
		t.Fatalf("expected spilled content page, got:\n%s", page)
	}
}

func TestReadSpillFindText(t *testing.T) {
	sessionID := "spill_find_case"
	t.Cleanup(func() { _ = os.RemoveAll(GetSystemPath(".goharness/sessions/" + sessionID)) })
	a := &Agent{SessionID: sessionID, Workspace: t.TempDir()}
	content := strings.Repeat("prefix\n", 2000) + "needle target\n" + strings.Repeat("suffix\n", 2000)
	meta, err := saveSpill(a, "read_file", content)
	if err != nil {
		t.Fatalf("saveSpill failed: %v", err)
	}

	page := readSpill(a, meta.ID, 0, 60, "needle target")
	if !strings.Contains(page, `match_for: "needle target"`) {
		t.Fatalf("expected match note, got:\n%s", page)
	}
	if !strings.Contains(page, "needle target") {
		t.Fatalf("expected matching content, got:\n%s", page)
	}
}

func TestBuildUIStateBlocksMissingModelAndBusy(t *testing.T) {
	activeConfig = &Config{Agent: AgentConfig{WorkspaceDir: t.TempDir()}, API: APIConfig{Model: ""}}
	state := buildUIState("ui_state_missing_model")
	if state.ComposerEnabled || state.BlockedReason != "Select a model first." {
		t.Fatalf("expected missing-model block, got %+v", state)
	}

	done := beginSessionRun("ui_state_busy")
	defer done()
	activeConfig.API.Model = "gpt-4o"
	busy := buildUIState("ui_state_busy")
	if busy.Status != "busy" || busy.ComposerEnabled {
		t.Fatalf("expected busy state, got %+v", busy)
	}
}
