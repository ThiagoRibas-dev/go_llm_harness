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

// broadcastSandboxWarning streams a highly detailed, security recommendations markdown block to the visual Web Console
func broadcastSandboxWarning(reason string) {
	warningMarkdown := fmt.Sprintf("⚠️ **[SECURITY ISOLATION WARNING] %s**\n\n" +
		"The terminal command was executed directly on your host operating system without sandbox restrictions. " +
		"To secure your host against prompt injections or malicious instructions, we highly recommend setting up one of these three secure sandboxing configurations:\n\n" +
		"1. 🐳 **Docker Container Sandboxing (Highly Recommended):**\n" +
		"   Run all shell commands inside an isolated Docker container. Run this locally to spawn your container workspace:\n" +
		"   ```bash\n" +
		"   docker run -d --name agent-workspace -v ./workspace:/workspace alpine tail -f /dev/null\n" +
		"   ```\n" +
		"   Then click **Settings** (⚙️) in the Web UI, change **Sandbox Mode** to **Docker**, and set **Docker Container** to `agent-workspace`.\n\n" +
		"2. 🐧 **WSL2 / Linux Virtualization:**\n" +
		"   Run GoHarness inside WSL2 or a Linux VM. Linux standard accounts have native access to **Landlock LSM** sandboxing, allowing GoHarness to lock down files with $100%%$ bare-metal isolation without administrative prompts.\n\n" +
		"3. 🛡️ **Administrative Elevation (Windows Bare-Metal):**\n" +
		"   If running natively on Windows host, open PowerShell or CMD as **Administrator** and launch `agent.exe` from there. This grants GoHarness the necessary security privileges to duplicate tokens and lock child processes into low-integrity Job Objects.", reason)

	BroadcastSSE("turn_secured", map[string]interface{}{
		"turn_number": 0,
		"role":        "system",
		"name":        "system",
		"content":     warningMarkdown,
	})
}
