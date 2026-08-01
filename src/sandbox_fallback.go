//go:build !linux && !darwin && !windows

package main

func getOSName() string {
	return "unknown"
}

// ExecutePlatformSandbox defaults to plain host execution on unrecognized operating systems
func ExecutePlatformSandbox(command, workspaceDir string) (string, error) {
	return executeNativeCommandPlain(command, workspaceDir)
}
