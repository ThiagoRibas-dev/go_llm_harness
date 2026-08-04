//go:build linux

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

// Landlock System Call Numbers for Linux x86_64
const (
	sysLandlockCreateRuleset = 444
	sysLandlockAddRule       = 445
	sysLandlockRestrictSelf  = 446
)

// Linux constants
const (
	prSetNoNewPrivs = 38
	oPath           = 0x200000 // O_PATH
)

// Landlock Filesystem Access Rights
const (
	accessFSWriteFile   = 1 << 0
	accessFSReadFile    = 1 << 2
	accessFSReadDir     = 1 << 3
	accessFSRemoveDir   = 1 << 4
	accessFSRemoveFile  = 1 << 5
	accessFSMakeChar    = 1 << 6
	accessFSMakeDir     = 1 << 7
	accessFSMakeReg     = 1 << 8
	accessFSMakeSock    = 1 << 9
	accessFSMakeFifo    = 1 << 10
	accessFSMakeBlock   = 1 << 11
	accessFSMakeSym     = 1 << 12
	accessFSRefer       = 1 << 13
	accessFSTruncate    = 1 << 14
)

type landlockRulesetAttr struct {
	HandledAccessFS uint64
}

type landlockPathBeneathAttr struct {
	AllowedAccess uint64
	ParentFd      int32
}

const (
	landlockRulePathBeneath = 1
)

func getOSName() string {
	return "linux"
}

// ExecutePlatformSandbox locks down the thread via Linux Landlock LSM before running commands
func ExecutePlatformSandbox(command, workspaceDir string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = workspaceDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Println("🔒 [SANDBOX ENGINE] Engaged: Linux Landlock LSM active for child command...")

	allReadAccess := uint64(accessFSReadFile | accessFSReadDir)
	allWriteAccess := uint64(accessFSWriteFile | accessFSRemoveFile | accessFSRemoveDir | accessFSMakeDir | accessFSMakeReg | accessFSTruncate)
	allFSAccess := allReadAccess | allWriteAccess

	// Create Ruleset
	rulesetAttr := landlockRulesetAttr{
		HandledAccessFS: allFSAccess,
	}

	r1, _, sysErr := syscall.Syscall(
		sysLandlockCreateRuleset,
		uintptr(unsafe.Pointer(&rulesetAttr)),
		uintptr(unsafe.Sizeof(rulesetAttr)),
		0,
	)
	if sysErr != 0 {
		if !activeConfig.Security.SandboxFallback {
			return "", fmt.Errorf("Landlock system calls not supported by host kernel (Sandbox Fallback is disabled in config.json)")
		}
		// Fallback gracefully if the kernel doesn't support Landlock yet (e.g. WSL1 or older kernels)
		fmt.Printf("%s[SANDBOX WARNING] Landlock system calls not supported by host kernel. Falling back to unsandboxed execution.%s\n", ColorYellow, ColorReset)
		broadcastSandboxWarning("Linux Landlock LSM is unsupported on this kernel")
		return executeNativeCommandPlain(command, workspaceDir)
	}
	defer syscall.Close(int(r1))

	// Allow full read/write to the Workspace Directory
	err := addLandlockPathRule(r1, workspaceDir, allFSAccess)
	if err != nil {
		if !activeConfig.Security.SandboxFallback {
			return "", fmt.Errorf("failed to restrict workspace directory path: %w", err)
		}
		broadcastSandboxWarning("Linux Landlock add rule failed: " + err.Error())
		return executeNativeCommandPlain(command, workspaceDir)
	}

	// Read-only access to critical system folders so command (like 'ls', 'python3', etc) can resolve dynamic linkings
	_ = addLandlockPathRule(r1, "/usr", allReadAccess)
	_ = addLandlockPathRule(r1, "/bin", allReadAccess)
	_ = addLandlockPathRule(r1, "/lib", allReadAccess)
	_ = addLandlockPathRule(r1, "/lib64", allReadAccess)
	_ = addLandlockPathRule(r1, "/etc", allReadAccess)

	// Lock privileges
	_, _, sysErr = syscall.Syscall(syscall.SYS_PRCTL, prSetNoNewPrivs, 1, 0)
	if sysErr != 0 {
		return "", fmt.Errorf("failed to set no_new_privs: %v", sysErr)
	}

	// Restrict ourselves! Thread locked.
	_, _, sysErr = syscall.Syscall(sysLandlockRestrictSelf, r1, 0, 0)
	if sysErr != 0 {
		return "", fmt.Errorf("failed to restrict thread: %v", sysErr)
	}

	runErr := cmd.Run()

	return formatCmdResult(runErr, stdout.String(), stderr.String()), nil
}

func addLandlockPathRule(rulesetFd uintptr, path string, allowedAccess uint64) error {
	fd, err := syscall.Open(path, oPath|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil // Ignore path if it doesn't exist
	}
	defer syscall.Close(fd)

	ruleAttr := landlockPathBeneathAttr{
		AllowedAccess: allowedAccess,
		ParentFd:      int32(fd),
	}

	_, _, sysErr := syscall.Syscall(
		sysLandlockAddRule,
		rulesetFd,
		landlockRulePathBeneath,
		uintptr(unsafe.Pointer(&ruleAttr)),
	)
	if sysErr != 0 {
		return fmt.Errorf("landlock_add_rule failed for %s: %v", path, sysErr)
	}
	return nil
}
