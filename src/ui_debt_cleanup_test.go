package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferArtifactsFromToolResult(t *testing.T) {
	cases := []struct {
		name   string
		result string
		want   []string
	}{
		{"write host", "Successfully wrote file to host disk at src/app.go", []string{"src/app.go"}},
		{"write docker", "Successfully wrote file inside Docker at docs/out.md", []string{"docs/out.md"}},
		{"patch", "Successfully patched file 'src/main.go'. Changes applied seamlessly.", []string{"src/main.go"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferArtifactsFromToolResult("tool", tc.result)
			if len(got) != len(tc.want) {
				t.Fatalf("want %d artifacts, got %d (%v)", len(tc.want), len(got), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("artifact[%d]: want %q, got %q", i, tc.want[i], got[i])
				}
			}
		})
	}
}

func TestGenerateWorkspaceEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := DirectoryScanConfig{
		MaxDepth:             4,
		MaxFilesPerDirectory: 10,
		IgnoredPatterns:      []string{".git"},
		CollapsedPatterns:    []string{"node_modules"},
	}
	entries, err := GenerateWorkspaceEntries(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var sawSrcDir, sawMainFile, sawCollapsed bool
	for _, entry := range entries {
		switch entry.Path {
		case "src":
			sawSrcDir = entry.IsDir
		case "src/main.go":
			sawMainFile = !entry.IsDir
		case "node_modules":
			sawCollapsed = entry.IsDir && entry.Collapsed
		}
	}
	if !sawSrcDir || !sawMainFile || !sawCollapsed {
		t.Fatalf("expected src dir=%v main file=%v collapsed node_modules=%v; entries=%+v", sawSrcDir, sawMainFile, sawCollapsed, entries)
	}
}
