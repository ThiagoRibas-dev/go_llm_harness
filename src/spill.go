package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	toolSpillThresholdChars = 20 * 1024
	toolSpillPreviewHead    = 4000
	toolSpillPreviewTail    = 2000
	toolSpillReadDefault    = 4000
	toolSpillReadMax        = 20000
)

type SpillMeta struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	ToolName  string `json:"tool_name"`
	FileName  string `json:"file_name"`
	CharCount int    `json:"char_count"`
	ByteCount int    `json:"byte_count"`
	CreatedAt string `json:"created_at"`
}

func spillDirForSession(sessionID string) string {
	return GetSystemPath(filepath.Join(".goharness", "sessions", sessionID, "spill"))
}

func spillDataPath(sessionID, id string) string {
	return filepath.Join(spillDirForSession(sessionID), id+".txt")
}

func spillMetaPath(sessionID, id string) string {
	return filepath.Join(spillDirForSession(sessionID), id+".json")
}

func hashSpillID(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func maybeSpillToolResult(a *Agent, toolName, result string) string {
	if a == nil || a.SessionID == "" {
		return result
	}
	runes := []rune(result)
	if len(runes) <= toolSpillThresholdChars {
		return result
	}
	meta, err := saveSpill(a, toolName, result)
	if err != nil {
		preview := buildSpillPreview(result)
		return fmt.Sprintf("=== TOOL OUTPUT TRUNCATED (spill write failed) ===\ntool: %s\nerror: %v\ntotal_chars: %d\n\n%s", toolName, err, len(runes), preview)
	}
	preview := buildSpillPreview(result)
	return fmt.Sprintf("=== TOOL OUTPUT SPILLED TO DISK ===\ntool: %s\nspill_id: %s\ntotal_chars: %d\ntotal_bytes: %d\npreview: head %d chars + tail %d chars\nhint: call read_spill with this spill_id to inspect the full output in pages.\n\n%s", meta.ToolName, meta.ID, meta.CharCount, meta.ByteCount, toolSpillPreviewHead, toolSpillPreviewTail, preview)
}

func saveSpill(a *Agent, toolName, content string) (*SpillMeta, error) {
	id := hashSpillID(content)
	dir := spillDirForSession(a.SessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	meta := &SpillMeta{
		ID:        id,
		SessionID: a.SessionID,
		ToolName:  toolName,
		FileName:  id + ".txt",
		CharCount: len([]rune(content)),
		ByteCount: len(content),
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	dataPath := spillDataPath(a.SessionID, id)
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		if err := os.WriteFile(dataPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(spillMetaPath(a.SessionID, id), metaBytes, 0644); err != nil {
		return nil, err
	}
	return meta, nil
}

func buildSpillPreview(content string) string {
	runes := []rune(content)
	if len(runes) <= toolSpillPreviewHead+toolSpillPreviewTail {
		return content
	}
	headEnd := toolSpillPreviewHead
	tailStart := len(runes) - toolSpillPreviewTail
	if headEnd > len(runes) {
		headEnd = len(runes)
	}
	if tailStart < headEnd {
		tailStart = headEnd
	}
	omitted := len(runes) - headEnd - (len(runes) - tailStart)
	return string(runes[:headEnd]) + fmt.Sprintf("\n\n... [SPILL OMITTED %d chars] ...\n\n", omitted) + string(runes[tailStart:])
}

func readSpill(a *Agent, id string, offset, limit int, findText string) string {
	if a == nil || a.SessionID == "" {
		return "Spill read failed: missing session context."
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "Spill read failed: id is required."
	}
	meta, _ := loadSpillMeta(a.SessionID, id)
	bytes, err := os.ReadFile(spillDataPath(a.SessionID, id))
	if err != nil {
		return fmt.Sprintf("Spill not found for id '%s'.", id)
	}
	content := string(bytes)
	runes := []rune(content)
	total := len(runes)
	if total == 0 {
		return fmt.Sprintf("=== SPILL CONTENT ===\nspill_id: %s\ntotal_chars: 0\n\n(empty spill)", id)
	}
	if limit <= 0 {
		limit = toolSpillReadDefault
	}
	if limit > toolSpillReadMax {
		limit = toolSpillReadMax
	}
	if offset < 0 {
		offset = 0
	}
	searchNote := ""
	if ft := strings.TrimSpace(findText); ft != "" {
		idx := strings.Index(strings.ToLower(content), strings.ToLower(ft))
		if idx >= 0 {
			offset = len([]rune(content[:idx]))
			searchNote = fmt.Sprintf("match_for: %q at char %d\n", ft, offset)
		} else {
			return fmt.Sprintf("Spill text not found.\nspill_id: %s\nfind_text: %q", id, ft)
		}
	}
	if offset >= total {
		offset = maxInt(0, total-limit)
	}
	end := offset + limit
	if end > total {
		end = total
	}
	nextOffset := end
	if nextOffset >= total {
		nextOffset = -1
	}
	prevOffset := offset - limit
	if prevOffset < 0 {
		prevOffset = 0
	}
	toolName := "unknown"
	if meta != nil && meta.ToolName != "" {
		toolName = meta.ToolName
	}
	return fmt.Sprintf("=== SPILL CONTENT ===\ntool: %s\nspill_id: %s\ntotal_chars: %d\nshowing_chars: %d-%d\nnext_offset: %d\nprev_offset: %d\n%s\n%s", toolName, id, total, offset, end, nextOffset, prevOffset, searchNote, string(runes[offset:end]))
}

func loadSpillMeta(sessionID, id string) (*SpillMeta, error) {
	bytes, err := os.ReadFile(spillMetaPath(sessionID, id))
	if err != nil {
		return nil, err
	}
	var meta SpillMeta
	if err := json.Unmarshal(bytes, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
