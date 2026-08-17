package main

import (
	"fmt"
	"runtime"
	"strings"
)

// executionEnvironment describes the OS/shell surface that the model's
// `execute_command` tool calls will actually land in. Models are trained on
// POSIX conventions (ls, cat, /-paths) and will blindly emit them even on a
// Windows host, so we detect the real target and tell them explicitly.
type executionEnvironment struct {
	OS          string // "linux", "windows", "darwin"
	Arch        string // "amd64", "arm64", ...
	Shell       string // "cmd.exe", "sh", "docker:sh"
	PathSep     string // "\\" or "/"
	LineEnding  string // "\r\n" or "\n"
	ListCmd     string // "dir" or "ls"
	ReadCmd     string // "type" or "cat"
	Workspace   string
	SandboxMode string
	DockerName  string
}

// detectEnvironment inspects the host + active sandbox configuration to work
// out where commands actually execute. The key subtlety: when sandbox_mode is
// "docker" the commands run inside a Linux container (sh -c) even if the host
// is Windows, so the model must target Linux there.
func detectEnvironment() executionEnvironment {
	env := executionEnvironment{
		Arch:        runtime.GOARCH,
		Workspace:   activeConfig.Agent.WorkspaceDir,
		SandboxMode: activeConfig.Security.SandboxMode,
		DockerName:  activeConfig.Security.DockerContainer,
	}

	switch {
	case env.SandboxMode == "docker":
		// docker exec <container> sh -c "<command>" -> always Linux/POSIX.
		env.OS = "linux"
		env.Shell = "docker:sh"
		env.PathSep = "/"
		env.LineEnding = "\n"
		env.ListCmd = "ls"
		env.ReadCmd = "cat"
	case runtime.GOOS == "windows":
		env.OS = "windows"
		// Both the unsandboxed path and the Windows Job-Object sandbox use cmd.exe /C.
		env.Shell = "cmd.exe"
		env.PathSep = "\\"
		env.LineEnding = "\r\n"
		env.ListCmd = "dir"
		env.ReadCmd = "type"
	case runtime.GOOS == "darwin":
		env.OS = "darwin"
		env.Shell = "sh"
		env.PathSep = "/"
		env.LineEnding = "\n"
		env.ListCmd = "ls"
		env.ReadCmd = "cat"
	default:
		env.OS = "linux"
		env.Shell = "sh"
		env.PathSep = "/"
		env.LineEnding = "\n"
		env.ListCmd = "ls"
		env.ReadCmd = "cat"
	}

	return env
}

// environmentSystemPrompt renders a concise, instruction-oriented block that is
// injected into the system prompt for both the linear agent loop and DAG llm
// nodes. It deliberately tells the model which commands/idioms to use AND which
// to avoid, rather than just naming the OS.
func (e executionEnvironment) systemPrompt() string {
	var b strings.Builder
	b.WriteString("=== EXECUTION ENVIRONMENT (you MUST adapt commands to this target) ===\n")
	fmt.Fprintf(&b, "Operating system: %s (%s)\n", e.OS, e.Arch)
	fmt.Fprintf(&b, "Workspace directory: %s\n", e.Workspace)

	switch e.Shell {
	case "docker:sh":
		fmt.Fprintf(&b, "Commands run via `docker exec %s sh -c \"<command>\"` — this is a Linux container regardless of the host OS.\n", e.DockerName)
		b.WriteString("Use POSIX shell syntax: `ls`, `cat`, `grep`, forward-slash paths. Do NOT use `dir`, `type`, or backslash paths.\n")
	case "cmd.exe":
		b.WriteString("Commands run via `cmd.exe /C <command>` on Windows.\n")
		b.WriteString("Use Windows commands: `dir` (not ls), `type` (not cat), `findstr` (not grep), backslash path separators, and `&`/`&&` to chain. Do NOT emit POSIX-only commands like `ls`, `cat`, `chmod`, or forward-slash absolute paths.\n")
	case "sh":
		if e.OS == "darwin" {
			b.WriteString("Commands run via `sh -c` on macOS (BSD userland). `ls`, `cat`, `grep` work; note BSD variants for `sed`/`cp` flags differ from GNU/Linux.\n")
		} else {
			b.WriteString("Commands run via `sh -c` on Linux. Use POSIX/GNU commands (`ls`, `cat`, `grep`) and forward-slash paths.\n")
		}
	}

	fmt.Fprintf(&b, "Path separator: %q. Line endings: %q.\n", e.PathSep, e.eolCRLF())
	b.WriteString("When you need to inspect the workspace, use `")
	b.WriteString(e.ListCmd)
	b.WriteString("`; to print a file, use `")
	b.WriteString(e.ReadCmd)
	b.WriteString("`. Match the casing of real file/directory names; lookups are case-sensitive on Linux/macOS.\n")
	b.WriteString("=== END EXECUTION ENVIRONMENT ===")
	return b.String()
}

func (e executionEnvironment) eolCRLF() string {
	if e.LineEnding == "\r\n" {
		return "CRLF"
	}
	return "LF"
}

// buildEnvironmentSystemPrompt is the package-level convenience used by the
// agent loop and workflow nodes.
func buildEnvironmentSystemPrompt() string {
	return detectEnvironment().systemPrompt()
}
