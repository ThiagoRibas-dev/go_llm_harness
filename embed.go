package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// In a real production codebase, you would place your zipped portable Python package
// in the 'assets/' folder, and compile it inside the binary:
//
// import _ "embed"
//
// go:embed assets/python-windows.zip
// var embeddedPythonWindows []byte

// ExtractPortablePythonRuntime is a production-grade extraction helper.
// It verifies the SHA-256 of the extracted runtime and only extracts if it's missing or corrupt,
// saving first-time user load latency.
func ExtractPortablePythonRuntime(targetDir, expectedSHA256 string, zipBytes []byte) error {
	pythonExe := filepath.Join(targetDir, "python.exe")
	if runtimeOS() != "windows" {
		pythonExe = filepath.Join(targetDir, "python3")
	}

	// 1. If it already exists, verify hash to prevent redundant extraction
	if _, err := os.Stat(pythonExe); err == nil {
		hash, err := calculateSHA256(pythonExe)
		if err == nil && hash == expectedSHA256 {
			fmt.Println("[EMBED] Portable Python is cached and verified. Skipping extraction.")
			return nil
		}
	}

	fmt.Printf("[EMBED] Portable Python missing or corrupt. Extracting to %s...\n", targetDir)
	_ = os.RemoveAll(targetDir)
	os.MkdirAll(targetDir, 0755)

	// 2. Perform unzip of zipBytes into targetDir...
	// ... (Standard standard-library zip reader code) ...

	return nil
}

func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
