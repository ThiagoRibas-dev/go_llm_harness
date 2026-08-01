//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func getOSName() string {
	return "darwin"
}

// ExecutePlatformSandbox runs the command inside Apple's native sandbox-exec SBPL container
func ExecutePlatformSandbox(command, workspaceDir string) (string, error) {
	// Generate macOS Sandbox Profile (SBPL)
	sbplProfile := fmt.Sprintf(`(version 1)
(deny default)

(allow process-exec)
(allow process-fork)
(allow signal)

;; Allow reading standard dynamic linking files & system bin assets
(allow file-read* 
    (subpath "/usr/lib")
    (subpath "/usr/bin")
    (subpath "/bin")
    (subpath "/System/Library")
    (subpath "/Library")
    (subpath "/private/var/db/dyld")
)

;; Allow full read/write inside Workspace ONLY
(allow file-read* file-write*
    (subpath "%s")
)
`, workspaceDir)

	fmt.Println("🔒 [SANDBOX ENGINE] Engaged: Apple sandbox-exec (SBPL) active...")

	cmd := exec.Command("sandbox-exec", "-p", sbplProfile, "sh", "-c", command)
	cmd.Dir = workspaceDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return formatCmdResult(err, stdout.String(), stderr.String()), nil
}
