//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")
	modadvapi32 = syscall.NewLazyDLL("advapi32.dll")

	procCreateJobObjectW         = modkernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject = modkernel32.NewProc("AssignProcessToJobObject")
	procCreateRestrictedToken    = modadvapi32.NewProc("CreateRestrictedToken")
	procSetTokenInformation      = modadvapi32.NewProc("SetTokenInformation")
	procConvertStringSidToSidW   = modadvapi32.NewProc("ConvertStringSidToSidW")
)

const (
	lowIntegritySid     = "S-1-16-4096"
	tokenIntegrityLevel = 25
)

type tokenMandatoryLabel struct {
	Label sidAndAttributes
}

type sidAndAttributes struct {
	Sid        uintptr
	Attributes uint32
}

func getOSName() string {
	return "windows"
}

// ExecutePlatformSandbox spawns a child command with a Windows Low-Integrity restricted token inside a Job Object
func ExecutePlatformSandbox(command, workspaceDir string) (string, error) {
	cmd := exec.Command("cmd.exe", "/C", command)
	cmd.Dir = workspaceDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Println("🔒 [SANDBOX ENGINE] Engaged: Windows Low-Integrity restricted Job Object active...")

	// 1. Load active process token
	var currentToken syscall.Token
	currentProcess, _ := syscall.GetCurrentProcess()
	err := syscall.OpenProcessToken(currentProcess, syscall.TOKEN_DUPLICATE|syscall.TOKEN_QUERY|syscall.TOKEN_ASSIGN_PRIMARY, &currentToken)
	if err != nil {
		return fallbackWindowsPlain(command, workspaceDir, fmt.Sprintf("OpenProcessToken failed: %v", err))
	}
	defer currentToken.Close()

	// 2. Create restricted token duplicate
	var restrictedToken syscall.Handle
	r1, _, err := procCreateRestrictedToken.Call(
		uintptr(currentToken),
		0,
		0, 0, // SIDs to Disable (Count, Array)
		0, 0, // Privileges to Delete (Count, Array)
		0, 0, // SIDs to Restrict (Count, Array)
		uintptr(unsafe.Pointer(&restrictedToken)),
	)
	if r1 == 0 {
		return fallbackWindowsPlain(command, workspaceDir, fmt.Sprintf("CreateRestrictedToken failed: %v", err))
	}
	restrictedTokenVal := syscall.Token(restrictedToken)
	defer restrictedTokenVal.Close()

	// 3. Drop token integrity SID to low-integrity S-1-16-4096
	var lowSid uintptr
	sidStrPtr, _ := syscall.UTF16PtrFromString(lowIntegritySid)
	r1, _, err = procConvertStringSidToSidW.Call(
		uintptr(unsafe.Pointer(sidStrPtr)),
		uintptr(unsafe.Pointer(&lowSid)),
	)
	if r1 == 0 {
		return fallbackWindowsPlain(command, workspaceDir, fmt.Sprintf("ConvertStringSidToSidW failed: %v", err))
	}

	var label tokenMandatoryLabel
	label.Label.Sid = lowSid
	label.Label.Attributes = 0x00000020 // SE_GROUP_INTEGRITY

	r1, _, err = procSetTokenInformation.Call(
		uintptr(restrictedTokenVal),
		uintptr(tokenIntegrityLevel),
		uintptr(unsafe.Pointer(&label)),
		uintptr(unsafe.Sizeof(label)),
	)
	if r1 == 0 {
		return fallbackWindowsPlain(command, workspaceDir, fmt.Sprintf("SetTokenInformation failed (Access is denied): %v", err))
	}

	// 4. Set cmd execution with the low-integrity token
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Token: restrictedTokenVal,
	}

	// 5. Create active Job Object limits
	r1, _, err = procCreateJobObjectW.Call(0, 0)
	if r1 == 0 {
		return fallbackWindowsPlain(command, workspaceDir, fmt.Sprintf("CreateJobObjectW failed: %v", err))
	}
	job := syscall.Handle(r1)
	defer syscall.CloseHandle(job)

	// 6. Spawn process
	err = cmd.Start()
	if err != nil {
		return fallbackWindowsPlain(command, workspaceDir, fmt.Sprintf("cmd.Start failed: %v", err))
	}

	// 7. Lock process into Job Object
	r1, _, err = procAssignProcessToJobObject.Call(uintptr(job), uintptr(cmd.Process.Pid))
	if r1 == 0 {
		_ = cmd.Process.Kill()
		return fallbackWindowsPlain(command, workspaceDir, fmt.Sprintf("AssignProcessToJobObject failed: %v", err))
	}

	runErr := cmd.Wait()

	return formatCmdResult(runErr, stdout.String(), stderr.String()), nil
}

func fallbackWindowsPlain(command, workspaceDir, errDetail string) (string, error) {
	if !activeConfig.Security.SandboxFallback {
		// Hard Block (Secure Default!)
		return "", fmt.Errorf("failed to restrict token integrity class: %s (Sandbox Fallback is disabled in config.json)", errDetail)
	}
	fmt.Printf("%s[SANDBOX WARNING] Windows restricted Job Object setup failed (%s). Falling back to unsandboxed execution.%s\n", ColorYellow, errDetail, ColorReset)
	broadcastSandboxWarning("Windows Privilege Restriction failed (" + errDetail + ")")
	return executeNativeCommandPlain(command, workspaceDir)
}
