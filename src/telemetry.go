package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TraceEntry represents a structured telemetry log entry
type TraceEntry struct {
	Timestamp string                 `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	Turn      int                    `json:"turn"`
	Action    string                 `json:"action"`
	Duration  int64                  `json:"duration_ms"`
	Status    string                 `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogExecutionTrace appends a structured trace entry to .goharness/traces.jsonl
func LogExecutionTrace(turn int, action string, startTime time.Time, status string, metadata map[string]interface{}) {
	duration := time.Since(startTime).Milliseconds()
	
	entry := TraceEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		SessionID: activeSessionID,
		Turn:      turn,
		Action:    action,
		Duration:  duration,
		Status:    status,
		Metadata:  metadata,
	}

	// 1. Ensure telemetry folder exists
	telemetryPath := filepath.Join(".goharness", "traces.jsonl")
	os.MkdirAll(filepath.Dir(telemetryPath), 0755)

	// 2. Open file in APPEND mode (create if not exists)
	file, err := os.OpenFile(telemetryPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("%s[WARNING] Failed to write telemetry trace log: %v%s\n", ColorYellow, err, ColorReset)
		return
	}
	defer file.Close()

	// 3. Marshal entry to single-line JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		return
	}

	// 4. Append log to file followed by newline
	_, _ = file.Write(append(jsonBytes, '\n'))
}
