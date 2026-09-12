package main

import (
	"regexp"
	"strings"
)

var (
	writeHostPathRE   = regexp.MustCompile(`(?m)^Successfully wrote file to host disk at\s+(.+?)\s*$`)
	writeDockerPathRE = regexp.MustCompile(`(?m)^Successfully wrote file inside Docker at\s+(.+?)\s*$`)
	patchPathRE       = regexp.MustCompile(`(?m)^Successfully patched file '([^']+)'`)
)

// inferArtifactsFromToolResult extracts workspace file paths materially produced
// or modified by a tool result. This gives the UI a structured artifact list so
// deliverables do not depend on parsing transcript prose client-side.
func inferArtifactsFromToolResult(toolName, result string) []string {
	_ = toolName
	var out []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		for _, existing := range out {
			if existing == path {
				return
			}
		}
		out = append(out, path)
	}
	if m := writeHostPathRE.FindStringSubmatch(result); len(m) > 1 {
		add(m[1])
	}
	if m := writeDockerPathRE.FindStringSubmatch(result); len(m) > 1 {
		add(m[1])
	}
	if m := patchPathRE.FindStringSubmatch(result); len(m) > 1 {
		add(m[1])
	}
	return out
}
