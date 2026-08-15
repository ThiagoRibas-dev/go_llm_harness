package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// providersFileName is the reusable connection-profile registry. It lives next
// to the binary (same rule as config.json/workflows.json) and is git-ignored
// because profiles hold API keys.
const providersFileName = "providers.json"

// ProvidersFile is the on-disk schema of providers.json: a named map of full
// API connection profiles that workflow nodes, the compaction engine, and the
// active chat connection can all reference by name.
type ProvidersFile struct {
	Providers map[string]APIConfig `json:"providers"`
}

func providersPath() string {
	return GetSystemPath(providersFileName)
}

// LoadProviders reads and parses providers.json from disk.
func LoadProviders() (*ProvidersFile, error) {
	bytes, err := os.ReadFile(providersPath())
	if err != nil {
		return nil, err
	}
	var pf ProvidersFile
	if err := json.Unmarshal(bytes, &pf); err != nil {
		return nil, err
	}
	if pf.Providers == nil {
		pf.Providers = make(map[string]APIConfig)
	}
	return &pf, nil
}

// SaveProviders serializes and writes the providers registry to disk.
func SaveProviders(pf *ProvidersFile) error {
	bytes, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(providersPath(), bytes, 0644)
}

// GetProvider resolves a single named profile. An empty name resolves to
// "default". Returns an error if the profile does not exist.
func GetProvider(name string) (APIConfig, error) {
	if name == "" {
		name = "default"
	}
	pf, err := LoadProviders()
	if err != nil {
		return APIConfig{}, fmt.Errorf("failed to load providers.json: %w", err)
	}
	profile, ok := pf.Providers[name]
	if !ok {
		return APIConfig{}, fmt.Errorf("provider profile %q is not defined in providers.json", name)
	}
	return profile, nil
}

// EnsureProvidersFile seeds providers.json on first run from the active
// config, so an existing installation migrates transparently:
//   - "default"     -> the current global chat API connection (config.api)
//   - "compaction"  -> the current summary/compaction connection
//
// An existing providers.json is never overwritten.
func EnsureProvidersFile() error {
	if _, err := os.Stat(providersPath()); err == nil {
		return nil
	}
	if activeConfig == nil {
		// Nothing to migrate yet; create an empty registry.
		return SaveProviders(&ProvidersFile{Providers: map[string]APIConfig{}})
	}

	profiles := map[string]APIConfig{
		"default": activeConfig.API,
	}

	comp := activeConfig.Compaction
	compProvider := comp.Provider
	if compProvider == "" {
		compProvider = "openai"
	}
	profiles["compaction"] = APIConfig{
		Provider:    compProvider,
		Key:         comp.Key,
		BaseURL:     comp.BaseURL,
		Model:       comp.Model,
		Temperature: comp.Temperature,
		ProjectID:   comp.ProjectID,
		Region:      comp.Region,
		MaxTokens:   activeConfig.API.MaxTokens,
	}

	fmt.Printf("%s[SYSTEM] providers.json not found. Seeded 'default' and 'compaction' profiles from config.json.%s\n", ColorYellow, ColorReset)
	return SaveProviders(&ProvidersFile{Providers: profiles})
}

// maskSecret returns a redacted placeholder for an API key safe to send to the
// browser: a fixed prefix plus the last 4 characters.
func maskSecret(s string) string {
	if len(s) <= 4 {
		return "••••"
	}
	return "••••…" + s[len(s)-4:]
}

// isMaskedSecret reports whether s is one of our masked placeholders (used to
// decide whether to preserve an existing key when the form is saved).
func isMaskedSecret(s string) bool {
	return s == "••••" || strings.HasPrefix(s, "••••…")
}

// mergeProfileWithFills layers a resolved provider profile under inline node
// overrides and the global connection, producing the APIConfig a node should
// actually execute with. Precedence (highest first):
//  1. per-node inline overrides (model, temperature, key, base_url) if non-empty
//  2. the named provider profile from providers.json
//  3. the global/parent API config (for max_tokens and key fallback)
func mergeProfileWithFills(profile APIConfig, props map[string]interface{}, parent APIConfig) APIConfig {
	if profile.MaxTokens == 0 {
		profile.MaxTokens = parent.MaxTokens
	}
	if profile.Key == "" {
		profile.Key = parent.Key
	}

	if m, ok := props["model"].(string); ok && m != "" {
		profile.Model = m
	}
	if t, ok := props["temperature"].(float64); ok {
		profile.Temperature = t
	}
	if k, ok := props["key"].(string); ok && k != "" {
		profile.Key = k
	}
	if b, ok := props["base_url"].(string); ok && b != "" {
		profile.BaseURL = b
	}
	return profile
}
