package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveNodeAPIMissingProfileFallsBack verifies that when an llm node
// references a provider profile that does not exist in providers.json, the
// resolver does NOT return an error (which would abort the whole workflow run)
// but instead falls back to the node's inline provider/model and then the
// global connection. This makes the Providers-tab delete promise truthful.
func TestResolveNodeAPIMissingProfileFallsBack(t *testing.T) {
	// Point providers.json at a temp dir that does not define "ghost" profile.
	tmp := t.TempDir()
	t.Chdir(tmp)
	// GetSystemPath resolves relative to the executable; write an empty registry
	// next to the test binary so GetProvider finds no profiles.
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("cannot resolve test executable: %v", err)
	}
	provPath := filepath.Join(filepath.Dir(exe), "providers.json")
	if err := os.WriteFile(provPath, []byte(`{"providers":{}}`), 0644); err != nil {
		t.Fatalf("write providers.json: %v", err)
	}
	t.Cleanup(func() { os.Remove(provPath) })

	activeConfig = &Config{
		API: APIConfig{
			Provider:    "openai",
			Model:       "global-model",
			Temperature: 0.5,
			Key:         "global-key",
			MaxTokens:   4096,
		},
	}

	e := &WorkflowExecutor{}
	n := &RuntimeNode{
		ID:   "query_node",
		Type: "llm",
		Properties: map[string]interface{}{
			"provider_profile": "ghost",       // does not exist
			"provider":         "openai",      // inline fallback
			"model":            "inline-model",
			"temperature":      0.1,
		},
	}

	cfg, err := e.resolveNodeAPI(n)
	if err != nil {
		t.Fatalf("resolveNodeAPI returned an error for a missing profile: %v", err)
	}
	if cfg.Model != "inline-model" {
		t.Errorf("expected inline model 'inline-model', got %q", cfg.Model)
	}
	if cfg.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %q", cfg.Provider)
	}
	if cfg.Temperature != 0.1 {
		t.Errorf("expected inline temperature 0.1, got %v", cfg.Temperature)
	}
	if cfg.Key != "global-key" {
		t.Errorf("expected global key fallback, got %q", cfg.Key)
	}
}

// TestResolveNodeAPIExistingProfileWins confirms a valid profile is used and
// its inline model does NOT silently override it (mergeProfileWithFills only
// applies non-empty inline overrides).
func TestResolveNodeAPIExistingProfileWins(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("cannot resolve test executable: %v", err)
	}
	provPath := filepath.Join(filepath.Dir(exe), "providers.json")
	if err := os.WriteFile(provPath, []byte(
		`{"providers":{"smart":{"provider":"anthropic","model":"claude-x","key":"prof-key","max_tokens":2048}}}`),
		0644); err != nil {
		t.Fatalf("write providers.json: %v", err)
	}
	t.Cleanup(func() { os.Remove(provPath) })

	activeConfig = &Config{
		API: APIConfig{Provider: "openai", Model: "global", Key: "global-key", MaxTokens: 4096},
	}

	e := &WorkflowExecutor{}
	n := &RuntimeNode{
		ID:         "aggregator",
		Type:       "llm",
		Properties: map[string]interface{}{"provider_profile": "smart"},
	}
	cfg, err := e.resolveNodeAPI(n)
	if err != nil {
		t.Fatalf("resolveNodeAPI error: %v", err)
	}
	if cfg.Provider != "anthropic" || cfg.Model != "claude-x" {
		t.Errorf("expected profile anthropic/claude-x, got %s/%s", cfg.Provider, cfg.Model)
	}
	if cfg.Key != "prof-key" {
		t.Errorf("expected profile key, got %q", cfg.Key)
	}
}
