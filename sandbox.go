package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RunCommandInSandbox executes a terminal command under the active sandbox rules.
// If sandbox_mode is "host", it triggers platform-specific bare-metal locks (Landlock/SBPL/Win32 Job Objects).
// If "docker", it routes via docker exec. If "none", it runs in plain host environment.
func RunCommandInSandbox(command, workspaceDir string) (string, error) {
	// 1. Docker Sandbox routing
	if activeConfig.Security.SandboxMode == "docker" {
		cmd := exec.Command("docker", "exec", activeConfig.Security.DockerContainer, "sh", "-c", command)
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		return formatCmdResult(err, out.String(), stderr.String()), nil
	}

	// 2. Direct plain host run (no sandboxing)
	if activeConfig.Security.SandboxMode == "none" {
		return executeNativeCommandPlain(command, workspaceDir)
	}

	// 3. Platform-specific Bare-Metal Sandbox (Landlock / SBPL / Job Objects)
	return ExecutePlatformSandbox(command, workspaceDir)
}

// executeNativeCommandPlain runs a standard, unsandboxed command on the host (fallback/none mode)
func executeNativeCommandPlain(command, workspaceDir string) (string, error) {
	var cmd *exec.Cmd
	if runtimeOS() == "windows" {
		cmd = exec.Command("cmd.exe", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Dir = workspaceDir

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()

	return formatCmdResult(err, out.String(), stderr.String()), nil
}

func formatCmdResult(err error, stdout, stderr string) string {
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return fmt.Sprintf("Error running process: %v", err)
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Exit Code: %d\n", exitCode))
	if stdout != "" {
		result.WriteString(fmt.Sprintf("Stdout:\n%s\n", stdout))
	}
	if stderr != "" {
		result.WriteString(fmt.Sprintf("Stderr:\n%s\n", stderr))
	}
	if stdout == "" && stderr == "" {
		result.WriteString("(No terminal output generated)\n")
	}
	return result.String()
}

// Simple helper to bridge OS checking without standard package runtime imports in all files
func runtimeOS() string {
	return getOSName()
}
