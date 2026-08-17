package main

import (
	"runtime"
	"strings"
	"testing"
)

// TestDetectEnvironmentShell verifies the shell target follows sandbox mode:
// docker mode is always Linux/sh regardless of host; host/none follow GOOS.
func TestDetectEnvironmentShell(t *testing.T) {
	activeConfig = &Config{}
	savedMode := activeConfig.Security.SandboxMode
	savedContainer := activeConfig.Security.DockerContainer
	t.Cleanup(func() {
		activeConfig.Security.SandboxMode = savedMode
		activeConfig.Security.DockerContainer = savedContainer
	})

	// Docker => always Linux container (sh), even on a Windows/macOS host.
	activeConfig.Security.SandboxMode = "docker"
	activeConfig.Security.DockerContainer = "agent-workspace"
	env := detectEnvironment()
	if env.OS != "linux" || env.Shell != "docker:sh" {
		t.Errorf("docker mode: expected linux/docker:sh, got %s/%s", env.OS, env.Shell)
	}
	if env.ListCmd != "ls" {
		t.Errorf("docker mode: expected list cmd ls, got %q", env.ListCmd)
	}
	if !strings.Contains(env.systemPrompt(), "docker exec") {
		t.Error("docker prompt should mention docker exec")
	}

	// Host/none => follow the actual host OS.
	activeConfig.Security.SandboxMode = "host"
	env = detectEnvironment()
	if runtime.GOOS == "windows" {
		if env.Shell != "cmd.exe" || env.ListCmd != "dir" {
			t.Errorf("windows host: expected cmd.exe/dir, got %s/%s", env.Shell, env.ListCmd)
		}
		if !strings.Contains(env.systemPrompt(), "cmd.exe") {
			t.Error("windows host prompt should mention cmd.exe")
		}
	} else {
		if env.Shell != "sh" || env.ListCmd != "ls" {
			t.Errorf("unix host: expected sh/ls, got %s/%s", env.Shell, env.ListCmd)
		}
	}
}

// TestEnvironmentPromptMentionsWorkspace ensures the workspace path is
// surfaced to the model (helps it avoid path mistakes).
func TestEnvironmentPromptMentionsWorkspace(t *testing.T) {
	activeConfig.Agent.WorkspaceDir = "/home/user/project"
	activeConfig.Security.SandboxMode = "none"
	prompt := detectEnvironment().systemPrompt()
	if !strings.Contains(prompt, "/home/user/project") {
		t.Errorf("prompt should include workspace path, got:\n%s", prompt)
	}
}
