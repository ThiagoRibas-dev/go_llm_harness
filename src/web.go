package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Global assets embedding (Phase 6.1)
//go:embed web/index.html
var embeddedWebFS embed.FS

// SSE Client Management
var (
	clientsMu  sync.Mutex
	sseClients []chan string
)

// RegisterSSEClient adds a client channel to the active broadcast list
func RegisterSSEClient() chan string {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	ch := make(chan string, 50)
	sseClients = append(sseClients, ch)
	return ch
}

// UnregisterSSEClient removes a client channel
func UnregisterSSEClient(ch chan string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for i, c := range sseClients {
		if c == ch {
			sseClients = append(sseClients[:i], sseClients[i+1:]...)
			close(ch)
			break
		}
	}
}

// BroadcastSSE pushes an event and its structured data to all listening browsers (Phase 6.2)
func BroadcastSSE(event string, data interface{}) {
	packet := map[string]interface{}{
		"event": event,
		"data":  data,
	}
	jsonBytes, err := json.Marshal(packet)
	if err != nil {
		return
	}

	clientsMu.Lock()
	for _, ch := range sseClients {
		select {
		case ch <- string(jsonBytes):
		default:
			// Skip slow or blocked clients
		}
	}
	clientsMu.Unlock()
}

// StartWebGUI launches Go's built-in web server and opens the browser (Phase 6)
func StartWebGUI(port int) {
	mux := http.NewServeMux()

	// 1. Root: Serve the embedded Single-Page App index.html
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		htmlBytes, err := embeddedWebFS.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "Embedded GUI index.html missing!", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(htmlBytes)
	})

	// 2. SSE Streaming endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		clientChan := RegisterSSEClient()
		defer UnregisterSSEClient(clientChan)

		// Send initial boot confirmation packet
		initMsg := map[string]interface{}{
			"event": "session_init",
			"data": map[string]interface{}{
				"session_id": activeSessionID,
			},
		}
		initBytes, _ := json.Marshal(initMsg)
		fmt.Fprintf(w, "data: %s\n\n", string(initBytes))
		flusher.Flush()

		// Stream loop
		for {
			select {
			case msg := <-clientChan:
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	// 3. API endpoints
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"session_id": activeSessionID,
			"api":        activeConfig.API,
			"agent":      activeConfig.Agent,
			"security":   activeConfig.Security,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Save modified configurations from Settings Modal directly to disk and reload (Phase 6.3)
	mux.HandleFunc("/api/config/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Provider     string `json:"provider"` // Phase 7
			APIKey       string `json:"api_key"`
			Model        string `json:"model"`
			BaseURL      string `json:"base_url"`
			SandboxMode  string `json:"sandbox_mode"`
			MaxTurns     int    `json:"max_turns"`
			WorkspaceDir string `json:"workspace_dir"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Update global memory settings immediately
		activeConfig.API.Provider = req.Provider
		activeConfig.API.Key = req.APIKey
		activeConfig.API.Model = req.Model
		activeConfig.API.BaseURL = req.BaseURL
		activeConfig.Security.SandboxMode = req.SandboxMode
		activeConfig.Agent.MaxTurns = req.MaxTurns
		
		// If workspace changed, set up the new workspace and add to history
		if activeConfig.Agent.WorkspaceDir != req.WorkspaceDir {
			selectWorkspace(req.WorkspaceDir)
		}

		// Overwrite config.json on host disk
		err := SaveConfig("config.json", activeConfig)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	// GET /api/workspaces: returns list of known workspaces & active
	mux.HandleFunc("/api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"active":    activeConfig.Agent.WorkspaceDir,
			"workspaces": activeConfig.Agent.WorkspacesHistory,
		})
	})

	// POST /api/workspaces/select: switches workspace
	mux.HandleFunc("/api/workspaces/select", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		selectWorkspace(req.Path)
		_ = SaveConfig("config.json", activeConfig)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "success",
			"active":     activeConfig.Agent.WorkspaceDir,
			"session_id": activeSessionID,
		})
	})

	// GET /api/sessions: Lists all sessions on disk via their meta.json (Phase 6.3)
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessionsRoot := filepath.Join(".goharness", "sessions")
		entries, err := os.ReadDir(sessionsRoot)
		if err != nil {
			http.Error(w, "No sessions found", http.StatusNotFound)
			return
		}

		var sessionList []SessionMeta
		for _, entry := range entries {
			if entry.IsDir() {
				metaPath := filepath.Join(sessionsRoot, entry.Name(), "meta.json")
				if bytes, err := os.ReadFile(metaPath); err == nil {
					var meta SessionMeta
					if err := json.Unmarshal(bytes, &meta); err == nil {
						sessionList = append(sessionList, meta)
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": sessionList,
		})
	})

	// POST /api/sessions/select: Switch active session and sync its workspace (Phase 6.3)
	mux.HandleFunc("/api/sessions/select", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		metaPath := filepath.Join(".goharness", "sessions", req.SessionID, "meta.json")
		bytes, err := os.ReadFile(metaPath)
		if err != nil {
			http.Error(w, "Session metadata not found", http.StatusNotFound)
			return
		}

		var meta SessionMeta
		_ = json.Unmarshal(bytes, &meta)

		// Sync settings
		activeSessionID = req.SessionID
		activeConfig.Agent.WorkspaceDir = meta.WorkspaceDir
		_ = SaveConfig("config.json", activeConfig)

		// Recalculate turn numbers from disk
		history := loadHistoryFromFiles()
		currentTurnNumber = len(history)

		// Broadcast new session ID
		BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "success",
			"session_id": activeSessionID,
			"workspace":  activeConfig.Agent.WorkspaceDir,
			"history":    history,
		})
	})

	// POST /api/sessions/branch: Non-destructive timeline branching! (Phase 6.3)
	mux.HandleFunc("/api/sessions/branch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ParentSessionID string `json:"parent_session_id"`
			Turn            int    `json:"turn"`
			BranchName      string `json:"branch_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 1. Generate new Session ID
		newSessionID := fmt.Sprintf("sess_%s_branch_turn%d", time.Now().Format("20060102-150405"), req.Turn)
		newSessionPath := filepath.Join(".goharness", "sessions", newSessionID)
		os.MkdirAll(newSessionPath, 0755)

		// 2. Read parent metadata
		parentMetaPath := filepath.Join(".goharness", "sessions", req.ParentSessionID, "meta.json")
		parentBytes, _ := os.ReadFile(parentMetaPath)
		var pMeta SessionMeta
		_ = json.Unmarshal(parentBytes, &pMeta)

		// 3. Copy turn files up to the target turn from parent directory
		parentPath := filepath.Join(".goharness", "sessions", req.ParentSessionID)
		entries, _ := os.ReadDir(parentPath)
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
				turnIdx, err := strconv.Atoi(name[:3])
				if err == nil && turnIdx <= req.Turn {
					_ = copyFile(filepath.Join(parentPath, name), filepath.Join(newSessionPath, name))
				}
			}
		}

		// Copy compaction summary and boundary files if they are within range
		pSummaryPath := filepath.Join(parentPath, "compacted_summary.json")
		pBoundaryPath := filepath.Join(parentPath, "compaction_boundary.txt")
		boundaryBytes, err := os.ReadFile(pBoundaryPath)
		if err == nil {
			boundary, _ := strconv.Atoi(string(boundaryBytes))
			if boundary <= req.Turn {
				_ = copyFile(pSummaryPath, filepath.Join(newSessionPath, "compacted_summary.json"))
				_ = copyFile(pBoundaryPath, filepath.Join(newSessionPath, "compaction_boundary.txt"))
			}
		}

		// 4. Copy Backups folder for the fork point (to restore the file state on disk)
		pBackupFolder := filepath.Join(parentPath, "backups")
		newBackupRoot := filepath.Join(newSessionPath, "backups")
		os.MkdirAll(newBackupRoot, 0755)

		// Copy parent backup folders <= targetTurn
		bEntries, err := os.ReadDir(pBackupFolder)
		if err == nil {
			for _, bEntry := range bEntries {
				if bEntry.IsDir() && strings.HasPrefix(bEntry.Name(), "turn-") {
					turnNumStr := strings.TrimPrefix(bEntry.Name(), "turn-")
					turnNum, err := strconv.Atoi(turnNumStr)
					if err == nil && turnNum <= req.Turn+1 {
						copyDir(filepath.Join(pBackupFolder, bEntry.Name()), filepath.Join(newBackupRoot, bEntry.Name()))
					}
				}
			}
		}

		// 5. Restore physical files on disk for the fork point!
		activeSessionID = newSessionID
		restoreWorkspaceBackups(req.Turn)

		// 6. Write the new metadata
		if req.BranchName == "" {
			req.BranchName = fmt.Sprintf("Branch from Turn %d", req.Turn)
		}
		createSessionMeta(newSessionID, pMeta.WorkspaceDir, req.ParentSessionID, req.BranchName)

		currentTurnNumber = req.Turn
		history := loadHistoryFromFiles()

		// Broadcast new session
		BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "success",
			"session_id": activeSessionID,
			"history":    history,
		})
	})

	mux.HandleFunc("/api/workspace/tree", func(w http.ResponseWriter, r *http.Request) {
		tree, _ := GenerateWorkspaceTree(activeConfig.Agent.WorkspaceDir, activeConfig.DirectoryScan)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tree": tree})
	})

	mux.HandleFunc("/api/prompt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		go runAgentLoop(req.Prompt)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"queued"}`))
	})

	mux.HandleFunc("/api/fork", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Turn int `json:"turn"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		executeSessionRollback(req.Turn)
		history := loadHistoryFromFiles()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"history": history,
		})
	})

	mux.HandleFunc("/api/compact", func(w http.ResponseWriter, r *http.Request) {
		history := loadHistoryFromFiles()
		go executeSlidingWindowCompaction(history)
		w.WriteHeader(http.StatusOK)
	})

	// 4. Start Server
	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("\n%s🚀 [WEB GUI] Server starting on http://%s/ %s\n", ColorBold+ColorGreen, serverAddr, ColorReset)

	// 5. Auto-Launch browser in background
	go func() {
		time.Sleep(500 * time.Millisecond)
		launchBrowser(fmt.Sprintf("http://%s/", serverAddr))
	}()

	_ = http.ListenAndServe(serverAddr, mux)
}

func launchBrowser(url string) {
	var cmd *exec.Cmd
	switch runtimeOS() {
	case "windows":
		cmd = exec.Command("cmd.exe", "/C", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	}

	if cmd != nil {
		_ = cmd.Start()
	}
}

// =================================================================
// 📁 INTERNAL CONVENIENCE FILES/DIRECTORIES UTILITIES
// =================================================================

// selectWorkspace handles active workspace switching, creating paths and maintaining config history (Phase 6.3)
func selectWorkspace(workspacePath string) {
	// Clean the path
	workspacePath = filepath.Clean(workspacePath)
	activeConfig.Agent.WorkspaceDir = workspacePath
	os.MkdirAll(workspacePath, 0755)

	// Add to WorkspacesHistory if not already present
	found := false
	for _, ws := range activeConfig.Agent.WorkspacesHistory {
		if ws == workspacePath {
			found = true
			break
		}
	}
	if !found {
		activeConfig.Agent.WorkspacesHistory = append(activeConfig.Agent.WorkspacesHistory, workspacePath)
	}

	// Spin up a brand new, clean session for this workspace!
	activeSessionID = "sess_" + time.Now().Format("20060102-150405")
	sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
	os.MkdirAll(sessionPath, 0755)

	// Write session metadata
	wsName := filepath.Base(workspacePath)
	if wsName == "." || wsName == "/" {
		wsName = "Default"
	}
	createSessionMeta(activeSessionID, workspacePath, "", "Workspace: "+wsName)
	currentTurnNumber = 0

	// Broadcast session update to GUI
	BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})
}

// createSessionMeta creates the meta.json descriptor inside each session
func createSessionMeta(sessionID, workspaceDir, parentID, name string) {
	meta := SessionMeta{
		SessionID:       sessionID,
		WorkspaceDir:    workspaceDir,
		CreatedAt:       time.Now().Format(time.RFC3339),
		Name:            name,
		ParentSessionID: parentID,
	}
	metaPath := filepath.Join(".goharness", "sessions", sessionID, "meta.json")
	os.MkdirAll(filepath.Dir(metaPath), 0755)

	bytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaPath, bytes, 0644)
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies a directory
func copyDir(src string, dst string) {
	os.MkdirAll(dst, 0755)
	entries, err := os.ReadDir(src)
	if err != nil {
		return
	}
	for _, entry := range entries {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(s, d)
		} else {
			_ = copyFile(s, d)
		}
	}
}
