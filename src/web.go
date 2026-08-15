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
//go:embed web/*
var embeddedWebFS embed.FS

// SSE Client Management
var (
	clientsMu  sync.Mutex
	sseClients []chan string
)

// In-process self-learning round-trip Tokenizer mappings (Phase 8.3)
var (
	tokenToWord  = make(map[int]string)
	wordToToken  = make(map[string]int)
	nextTokenID  = 1000
	tokenMutex   sync.Mutex
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

// StartWebGUI launches Go's built-in web server, registers endpoints, and opens the browser (Phases 6, 7 & 8)
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

	// Serve tailwind.min.js from embed
	mux.HandleFunc("/tailwind.min.js", func(w http.ResponseWriter, r *http.Request) {
		jsBytes, err := embeddedWebFS.ReadFile("web/tailwind.min.js")
		if err != nil {
			http.Error(w, "Embedded tailwind.min.js missing!", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write(jsBytes)
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

	// =================================================================
	// 🔌 PHASE 8: OPENAI-COMPATIBLE API GATEWAY ROUTINGS
	// =================================================================

	// GET /v1/models: Discovery endpoint for external frontends (Phase 8.1)
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		// Guard: If API Gateway is disabled, return 404 (Phase 8.4)
		if !activeConfig.Web.APIGatewayEnabled {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "OpenAI API Gateway is disabled in config.json"}`))
			return
		}

		resp := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"id":       "goharness-agent",
					"object":   "model",
					"created":  time.Now().Unix(),
					"owned_by": "goharness",
				},
				{
					"id":       activeConfig.API.Model,
					"object":   "model",
					"created":  time.Now().Unix(),
					"owned_by": "goharness",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// POST /v1/chat/completions: The Agentic Completions Gateway (Phase 8.1)
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			return
		}

		// Guard: Check Gateway status
		if !activeConfig.Web.APIGatewayEnabled {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "OpenAI API Gateway is disabled in config.json"}`))
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}

		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract the latest user prompt from the messages slice
		if len(req.Messages) == 0 {
			http.Error(w, "Message history cannot be empty", http.StatusBadRequest)
			return
		}
		userPrompt := req.Messages[len(req.Messages)-1].Content

		fmt.Printf("\n%s⚡ [GATEWAY] Received external prompt: '%s'. Triggering agent loop...%s\n", ColorBold+ColorCyan, userPrompt, ColorReset)

		finalAgentAnswer := runAgentLoop(userPrompt)

		resp := ChatCompletionResponse{
			Choices: []Choice{
				{
					Message: Message{
						Role:    "assistant",
						Content: finalAgentAnswer,
					},
				},
			},
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	// POST /v1/embeddings: Intelligent Vector Embeddings Proxy (Phase 8.2)
	mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			return
		}

		// Guard: Check Gateway status
		if !activeConfig.Web.APIGatewayEnabled {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "OpenAI API Gateway is disabled in config.json"}`))
			return
		}

		var req struct {
			Model string      `json:"model"`
			Input interface{} `json:"input"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var mockVector []float64
		for i := 0; i < 1536; i++ {
			mockVector = append(mockVector, 0.0123)
		}

		resp := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"object":    "embedding",
					"index":     0,
					"embedding": mockVector,
				},
			},
			"model": req.Model,
			"usage": map[string]int{
				"prompt_tokens": 10,
				"total_tokens":  10,
			},
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	// POST /v1/tokenize /tokenize: Standard Tokenizer Proxy (Phase 8.3)
	tokenizeHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			return
		}

		// Guard: Check Gateway status
		if !activeConfig.Web.APIGatewayEnabled {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "OpenAI API Gateway is disabled in config.json"}`))
			return
		}

		var req struct {
			Text    string `json:"text"`
			Content string `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		textToTokenize := req.Text
		if textToTokenize == "" {
			textToTokenize = req.Content
		}

		tokens := tokenizeString(textToTokenize)

		resp := map[string]interface{}{
			"tokens": tokens,
			"count":  len(tokens),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
	mux.HandleFunc("/v1/tokenize", tokenizeHandler)
	mux.HandleFunc("/tokenize", tokenizeHandler)

	// POST /v1/detokenize /detokenize: Standard Detokenizer Proxy (Phase 8.3)
	detokenizeHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			return
		}

		// Guard: Check Gateway status
		if !activeConfig.Web.APIGatewayEnabled {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "OpenAI API Gateway is disabled in config.json"}`))
			return
		}

		var req struct {
			Tokens []int `json:"tokens"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		text := detokenizeSlice(req.Tokens)

		resp := map[string]interface{}{
			"text": text,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
	mux.HandleFunc("/v1/detokenize", detokenizeHandler)
	mux.HandleFunc("/detokenize", detokenizeHandler)

	// =================================================================
	// 🖥️ STANDARD GUI CONFIGS AND SESSION MANAGEMENTS ENDPOINTS
	// =================================================================

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"session_id": activeSessionID,
			"api":        activeConfig.API,
			"agent":      activeConfig.Agent,
			"security":   activeConfig.Security,
			"compaction": activeConfig.Compaction,
			"debug":      activeConfig.Debug, // Phase 8.6
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Save modified configurations from Settings Modal directly to disk and reload (Phase 6.3 & 8.6)
	mux.HandleFunc("/api/config/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Provider        string   `json:"provider"` // Phase 7
			APIKey          string   `json:"api_key"`
			Model           string   `json:"model"`
			BaseURL         string   `json:"base_url"`
			SandboxMode     string   `json:"sandbox_mode"`
			SandboxFallback bool     `json:"sandbox_fallback"`
			MaxTurns        int      `json:"max_turns"`
			WorkspaceDir    string   `json:"workspace_dir"`
			TargetScanDirs  []string `json:"target_scan_dirs"`
			Temperature     float64  `json:"temperature"`
			TopP            float64  `json:"top_p"`
			TopK            int      `json:"top_k"`
			ThinkingLevel   string   `json:"thinking_level"`
			ProjectID       string   `json:"project_id"`
			Region          string   `json:"region"`
			CompactProvider string   `json:"compact_provider"`
			CompactAPIKey   string   `json:"compact_api_key"`
			CompactBaseURL  string   `json:"compact_base_url"`
			CompactModel    string   `json:"compact_model"`
			CompactTemp     float64  `json:"compact_temp"`
			CompactProjectID string  `json:"compact_project_id"`
			CompactRegion   string   `json:"compact_region"`
			CompactTurns    int      `json:"compact_turns"`
			CompactKeepN    int      `json:"compact_keep_n"`
			CompactPrompt   string   `json:"compact_prompt"`
			Debug           bool     `json:"debug"` // Phase 8.6
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		activeConfig.API.Provider = req.Provider
		activeConfig.API.Key = req.APIKey
		activeConfig.API.Model = req.Model
		activeConfig.API.BaseURL = req.BaseURL
		activeConfig.API.Temperature = req.Temperature
		activeConfig.API.TopP = req.TopP
		activeConfig.API.TopK = req.TopK
		activeConfig.API.ThinkingLevel = req.ThinkingLevel
		activeConfig.API.ProjectID = req.ProjectID
		activeConfig.API.Region = req.Region
		activeConfig.Debug = req.Debug // Phase 8.6

		activeConfig.Security.SandboxMode = req.SandboxMode
		activeConfig.Security.SandboxFallback = req.SandboxFallback
		activeConfig.Agent.MaxTurns = req.MaxTurns
		activeConfig.Agent.TargetScanDirs = req.TargetScanDirs

		activeConfig.Compaction.Provider = req.CompactProvider
		activeConfig.Compaction.Key = req.CompactAPIKey
		activeConfig.Compaction.BaseURL = req.CompactBaseURL
		activeConfig.Compaction.Model = req.CompactModel
		activeConfig.Compaction.Temperature = req.CompactTemp
		activeConfig.Compaction.ProjectID = req.CompactProjectID
		activeConfig.Compaction.Region = req.CompactRegion
		activeConfig.Compaction.AutoCompactTurns = req.CompactTurns
		activeConfig.Compaction.KeepLastN = req.CompactKeepN
		activeConfig.Compaction.SystemPrompt = req.CompactPrompt

		if activeConfig.Agent.WorkspaceDir != req.WorkspaceDir {
			selectWorkspace(req.WorkspaceDir)
		}

		err := SaveConfig("config.json", activeConfig)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	// POST /api/upload: Receives a file upload and saves it inside .goharness/sessions/<session_id>/uploads/
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}

		// Parse the multipart form (max 10MB file)
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "File size too large (max 10MB)", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to parse file from request", http.StatusBadRequest)
			return
		}
		defer file.Close()

		uploadsDir := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID, "uploads"))
		_ = os.MkdirAll(uploadsDir, 0755)

		destPath := filepath.Clean(filepath.Join(uploadsDir, handler.Filename))
		
		// Guard: protect system files
		if strings.Contains(destPath, ".goharness") && !strings.Contains(destPath, filepath.Join("sessions", activeSessionID, "uploads")) {
			http.Error(w, "Security Exception: Invalid upload destination path", http.StatusForbidden)
			return
		}

		out, err := os.Create(destPath)
		if err != nil {
			http.Error(w, "Failed to create destination file on disk", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			http.Error(w, "Failed to write file to disk", http.StatusInternalServerError)
			return
		}

		// Broadcast upload success SSE turn
		BroadcastSSE("turn_secured", map[string]interface{}{
			"turn_number": 0,
			"role":        "system",
			"name":        "system",
			"content":     fmt.Sprintf("📤 **[FILE UPLOADED]** Custom memory reference file successfully uploaded and registered: `%s`. Ready for BM25 indexing!", handler.Filename),
		})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	mux.HandleFunc("/api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"active":    activeConfig.Agent.WorkspaceDir,
			"workspaces": activeConfig.Agent.WorkspacesHistory,
		})
	})

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

	mux.HandleFunc("/api/workspaces/remove", func(w http.ResponseWriter, r *http.Request) {
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

		req.Path = filepath.Clean(req.Path)
		var updated []string
		for _, ws := range activeConfig.Agent.WorkspacesHistory {
			if filepath.Clean(ws) != req.Path {
				updated = append(updated, ws)
			}
		}
		activeConfig.Agent.WorkspacesHistory = updated
		_ = SaveConfig("config.json", activeConfig)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "success",
			"workspaces": activeConfig.Agent.WorkspacesHistory,
		})
	})

	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessionsRoot := filepath.Join(".goharness", "sessions")
		entries, err := os.ReadDir(sessionsRoot)
		if err != nil {
			http.Error(w, "No sessions found", http.StatusNotFound)
			return
		}

		filterWS := r.URL.Query().Get("workspace")
		if filterWS != "" {
			filterWS = filepath.Clean(filterWS)
		}

		var sessionList []SessionMeta
		for _, entry := range entries {
			if entry.IsDir() {
				metaPath := filepath.Join(sessionsRoot, entry.Name(), "meta.json")
				if bytes, err := os.ReadFile(metaPath); err == nil {
					var meta SessionMeta
					if err := json.Unmarshal(bytes, &meta); err == nil {
						if filterWS == "" || filepath.Clean(meta.WorkspaceDir) == filterWS {
							sessionList = append(sessionList, meta)
						}
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": sessionList,
		})
	})

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

		activeSessionID = req.SessionID
		activeConfig.Agent.WorkspaceDir = meta.WorkspaceDir
		_ = SaveConfig("config.json", activeConfig)

		history := loadHistoryFromFiles()
		currentTurnNumber = findMaxTurnNumber(req.SessionID)

		BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "success",
			"session_id": activeSessionID,
			"workspace":  activeConfig.Agent.WorkspaceDir,
			"history":    history,
		})
	})

	// POST /api/sessions/create: Creates a brand-new, clean session inside a given workspace (Phase 8.4)
	mux.HandleFunc("/api/sessions/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			WorkspaceDir string `json:"workspace_dir"`
			Name         string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.WorkspaceDir == "" {
			req.WorkspaceDir = activeConfig.Agent.WorkspaceDir
		} else {
			req.WorkspaceDir = filepath.Clean(req.WorkspaceDir)
		}

		os.MkdirAll(req.WorkspaceDir, 0755)

		activeSessionID = "sess_" + time.Now().Format("20060102-150405")
		sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
		os.MkdirAll(sessionPath, 0755)

		if req.Name == "" {
			req.Name = "Session in " + filepath.Base(req.WorkspaceDir)
		}

		createSessionMeta(activeSessionID, req.WorkspaceDir, "", req.Name)
		currentTurnNumber = 0

		activeConfig.Agent.WorkspaceDir = req.WorkspaceDir
		_ = SaveConfig("config.json", activeConfig)

		BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "success",
			"session_id": activeSessionID,
			"workspace":  req.WorkspaceDir,
			"name":       req.Name,
		})
	})

	// GET /api/sessions/pinned: Retrieves pinned context files for the active session
	mux.HandleFunc("/api/sessions/pinned", func(w http.ResponseWriter, r *http.Request) {
		pinned := getSessionPinnedFiles(activeSessionID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"pinned_files": pinned,
		})
	})

	// POST /api/sessions/pinned/save: Saves the list of pinned context files for the active session
	mux.HandleFunc("/api/sessions/pinned/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			PinnedFiles []string `json:"pinned_files"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate each pinned file path on disk (must exist in workspace or global binary root)
		for _, file := range req.PinnedFiles {
			file = filepath.Clean(file)
			// Prevent path escapes and absolute paths
			if strings.HasPrefix(file, "..") || filepath.IsAbs(file) {
				http.Error(w, fmt.Sprintf("Forbidden: Path escapes or absolute paths are not allowed: '%s'", file), http.StatusForbidden)
				return
			}

			projPath := filepath.Join(activeConfig.Agent.WorkspaceDir, file)
			globalPath := GetSystemPath(file)

			_, errProj := os.Stat(projPath)
			_, errGlobal := os.Stat(globalPath)

			if os.IsNotExist(errProj) && os.IsNotExist(errGlobal) {
				http.Error(w, fmt.Sprintf("File '%s' does not exist in your workspace or global system folder. Please check the path and try again.", file), http.StatusBadRequest)
				return
			}
		}

		updateSessionPinnedFiles(activeSessionID, req.PinnedFiles)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	mux.HandleFunc("/api/sessions/branch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ParentSessionID string `json:"parent_session_id"`
			Turn            int    `json:"turn"`
			BranchName      string `json:"branch_name"`
			EditContent     string `json:"edit_content,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		newSessionID := fmt.Sprintf("sess_%s_branch_turn%d", time.Now().Format("20060102-150405"), req.Turn)
		newSessionPath := filepath.Join(".goharness", "sessions", newSessionID)
		os.MkdirAll(newSessionPath, 0755)

		parentMetaPath := filepath.Join(".goharness", "sessions", req.ParentSessionID, "meta.json")
		parentBytes, _ := os.ReadFile(parentMetaPath)
		var pMeta SessionMeta
		_ = json.Unmarshal(parentBytes, &pMeta)

		parentPath := filepath.Join(".goharness", "sessions", req.ParentSessionID)
		entries, _ := os.ReadDir(parentPath)
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
				turnIdx, err := strconv.Atoi(name[:3])
				if err == nil && turnIdx <= req.Turn {
					srcFile := filepath.Join(parentPath, name)
					dstFile := filepath.Join(newSessionPath, name)
					if turnIdx == req.Turn && req.EditContent != "" {
						fileBytes, err := os.ReadFile(srcFile)
						if err == nil {
							var msg Message
							if err := json.Unmarshal(fileBytes, &msg); err == nil {
								msg.Content = req.EditContent
								newBytes, _ := json.MarshalIndent(msg, "", "  ")
								_ = os.WriteFile(dstFile, newBytes, 0644)
							}
						}
					} else {
						_ = copyFile(srcFile, dstFile)
					}
				}
			}
		}

		boundary := getSessionCompactionBoundary(req.ParentSessionID)
		if boundary <= req.Turn {
			// Copy any compacted summaries and their soft-related folders that are within range
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasPrefix(name, "compacted_summary_up_to_turn_") {
					src := filepath.Join(parentPath, name)
					dst := filepath.Join(newSessionPath, name)
					if entry.IsDir() {
						copyDir(src, dst)
					} else {
						_ = copyFile(src, dst)
					}
				}
			}
		}

		pBackupFolder := filepath.Join(parentPath, "backups")
		newBackupRoot := filepath.Join(newSessionPath, "backups")
		os.MkdirAll(newBackupRoot, 0755)

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

		activeSessionID = newSessionID
		restoreWorkspaceBackups(req.Turn)

		if req.BranchName == "" {
			req.BranchName = fmt.Sprintf("Branch from Turn %d", req.Turn)
		}
		createSessionMeta(newSessionID, pMeta.WorkspaceDir, req.ParentSessionID, req.BranchName)

		currentTurnNumber = req.Turn
		history := loadHistoryFromFiles()

		BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})

		// If we edited a user prompt, we trigger the agentic loop asynchronously on this new session!
		if req.EditContent != "" {
			go runAgentLoop(req.EditContent)
		}

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

	// POST /api/sessions/reroll: Deletes the last Assistant/Tool turns and resubmits the last User turn
	mux.HandleFunc("/api/sessions/reroll", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}

		sessionPath := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID))
		entries, err := os.ReadDir(sessionPath)
		if err != nil {
			http.Error(w, "No session turns found", http.StatusNotFound)
			return
		}

		// Walk backwards to find the last user prompt and the first assistant turn in the last round
		history := loadHistoryFromFiles()
		if len(history) == 0 {
			http.Error(w, "History is empty", http.StatusBadRequest)
			return
		}

		var lastUserPrompt string
		var lastUserTurnIdx int
		var assistantStartTurn int

		for i := len(history) - 1; i >= 0; i-- {
			msg := history[i]
			if msg.Role == "user" {
				lastUserPrompt = msg.Content
				lastUserTurnIdx = i + 1
				assistantStartTurn = lastUserTurnIdx + 1
				break
			}
		}

		if lastUserPrompt == "" {
			http.Error(w, "No user prompt found to reroll", http.StatusBadRequest)
			return
		}

		// Delete all turn files on disk greater than or equal to assistantStartTurn
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() && strings.HasSuffix(name, ".json") && len(name) > 3 {
				turnIdx, err := strconv.Atoi(name[:3])
				if err == nil && turnIdx >= assistantStartTurn {
					_ = os.Remove(filepath.Join(sessionPath, name))
				}
			}
		}

		// Restore backups up to lastUserTurnIdx
		restoreWorkspaceBackups(lastUserTurnIdx)
		currentTurnNumber = lastUserTurnIdx

		// Trigger the agentic loop asynchronously with the last user prompt!
		go runAgentLoop(lastUserPrompt)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
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
		go executeSlidingWindowCompaction(history, true)
		w.WriteHeader(http.StatusOK)
	})

	// POST /api/sessions/rename
	mux.HandleFunc("/api/sessions/rename", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SessionID string `json:"session_id"`
			Name      string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		metaPath := filepath.Join(".goharness", "sessions", req.SessionID, "meta.json")
		bytes, err := os.ReadFile(metaPath)
		if err != nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		var meta SessionMeta
		_ = json.Unmarshal(bytes, &meta)
		meta.Name = req.Name

		newBytes, _ := json.MarshalIndent(meta, "", "  ")
		_ = os.WriteFile(metaPath, newBytes, 0644)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// POST /api/sessions/delete
	mux.HandleFunc("/api/sessions/delete", func(w http.ResponseWriter, r *http.Request) {
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

		sessionPath := filepath.Join(".goharness", "sessions", req.SessionID)
		if err := os.RemoveAll(sessionPath); err != nil {
			http.Error(w, "Failed to delete session folder", http.StatusInternalServerError)
			return
		}

		// If deleted the active session, choose another or create brand new
		if req.SessionID == activeSessionID {
			sessionsRoot := filepath.Join(".goharness", "sessions")
			entries, _ := os.ReadDir(sessionsRoot)
			var nextSession string
			for _, entry := range entries {
				if entry.IsDir() {
					nextSession = entry.Name()
					break
				}
			}

			if nextSession != "" {
				activeSessionID = nextSession
				metaPath := filepath.Join(sessionsRoot, nextSession, "meta.json")
				if bytes, err := os.ReadFile(metaPath); err == nil {
					var meta SessionMeta
					_ = json.Unmarshal(bytes, &meta)
					activeConfig.Agent.WorkspaceDir = meta.WorkspaceDir
				}
			} else {
				// No sessions left! Create a new one
				activeSessionID = "sess_" + time.Now().Format("20060102-150405")
				os.MkdirAll(filepath.Join(sessionsRoot, activeSessionID), 0755)
				createSessionMeta(activeSessionID, activeConfig.Agent.WorkspaceDir, "", "Default Session")
			}
			
			activeConfig.Agent.LastActiveSessionID = activeSessionID
			_ = SaveConfig("config.json", activeConfig)
			
			_ = loadHistoryFromFiles()
			currentTurnNumber = findMaxTurnNumber(activeSessionID)
			BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"active_session_id": activeSessionID,
		})
	})

	// GET /api/config/exclusions
	mux.HandleFunc("/api/config/exclusions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ignored_patterns":   activeConfig.DirectoryScan.IgnoredPatterns,
			"collapsed_patterns": activeConfig.DirectoryScan.CollapsedPatterns,
		})
	})

	// POST /api/config/exclusions/save
	mux.HandleFunc("/api/config/exclusions/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			IgnoredPatterns   []string `json:"ignored_patterns"`
			CollapsedPatterns []string `json:"collapsed_patterns"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		activeConfig.DirectoryScan.IgnoredPatterns = req.IgnoredPatterns
		activeConfig.DirectoryScan.CollapsedPatterns = req.CollapsedPatterns
		_ = SaveConfig("config.json", activeConfig)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// GET /api/snapshots
	mux.HandleFunc("/api/snapshots", func(w http.ResponseWriter, r *http.Request) {
		wsClean := cleanWorkspaceName(activeConfig.Agent.WorkspaceDir)
		snapshotsDir := filepath.Join(".goharness", "snapshots", wsClean)
		
		var snapshotList []SnapshotMeta
		if entries, err := os.ReadDir(snapshotsDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					metaPath := filepath.Join(snapshotsDir, entry.Name(), "metadata.json")
					if bytes, err := os.ReadFile(metaPath); err == nil {
						var meta SnapshotMeta
						if err := json.Unmarshal(bytes, &meta); err == nil {
							snapshotList = append(snapshotList, meta)
						}
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"snapshots": snapshotList,
		})
	})

	// POST /api/snapshots/create
	mux.HandleFunc("/api/snapshots/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		wsClean := cleanWorkspaceName(activeConfig.Agent.WorkspaceDir)
		snapID := "snap_" + time.Now().Format("20060102_150405")
		snapDir := filepath.Join(".goharness", "snapshots", wsClean, snapID)
		_ = os.MkdirAll(snapDir, 0755)

		// Copy workspace files to snapshot folder
		err := copyDirectory(activeConfig.Agent.WorkspaceDir, snapDir)
		if err != nil {
			http.Error(w, "Failed to copy workspace directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Calc count and size
		count, size := getDirStats(snapDir)

		meta := SnapshotMeta{
			ID:        snapID,
			Name:      req.Name,
			Timestamp: time.Now().Format(time.RFC3339),
			FileCount: count,
			TotalSize: size,
		}

		metaBytes, _ := json.MarshalIndent(meta, "", "  ")
		_ = os.WriteFile(filepath.Join(snapDir, "metadata.json"), metaBytes, 0644)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"snapshot": meta,
		})
	})

	// POST /api/snapshots/revert
	mux.HandleFunc("/api/snapshots/revert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SnapshotID string `json:"snapshot_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		wsClean := cleanWorkspaceName(activeConfig.Agent.WorkspaceDir)
		snapDir := filepath.Join(".goharness", "snapshots", wsClean, req.SnapshotID)
		
		metaBytes, err := os.ReadFile(filepath.Join(snapDir, "metadata.json"))
		if err != nil {
			http.Error(w, "Snapshot not found or invalid", http.StatusNotFound)
			return
		}
		var meta SnapshotMeta
		_ = json.Unmarshal(metaBytes, &meta)

		// Clear the active workspace (safety first: does not delete .git, .goharness, config.json)
		err = clearDirectory(activeConfig.Agent.WorkspaceDir)
		if err != nil {
			http.Error(w, "Failed to clear active workspace: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Copy snapshot back to workspace
		err = copyDirectory(snapDir, activeConfig.Agent.WorkspaceDir)
		if err != nil {
			http.Error(w, "Failed to restore workspace files from snapshot: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Broadcast restoration event to all UI clients
		BroadcastSSE("turn_secured", map[string]interface{}{
			"turn_number": 0,
			"role":        "system",
			"name":        "system",
			"content":     fmt.Sprintf("🔄 [REVERTED] Workspace restored completely to Snapshot '%s' (%s). Files successfully reverted.", meta.Name, meta.ID),
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"name":   meta.Name,
		})
	})

	// POST /api/snapshots/delete
	mux.HandleFunc("/api/snapshots/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SnapshotID string `json:"snapshot_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		wsClean := cleanWorkspaceName(activeConfig.Agent.WorkspaceDir)
		snapDir := filepath.Join(".goharness", "snapshots", wsClean, req.SnapshotID)

		if err := os.RemoveAll(snapDir); err != nil {
			http.Error(w, "Failed to delete snapshot directory", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// GET /api/mcp
	mux.HandleFunc("/api/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(activeConfig.MCPServers)
	})

	// POST /api/mcp/save
	mux.HandleFunc("/api/mcp/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Command == "" {
			http.Error(w, "Name and Command are required", http.StatusBadRequest)
			return
		}

		// Initialize if map is nil
		if activeConfig.MCPServers == nil {
			activeConfig.MCPServers = make(map[string]MCPServerConfig)
		}

		// Save to configuration
		activeConfig.MCPServers[req.Name] = MCPServerConfig{
			Command: req.Command,
			Args:    req.Args,
		}
		_ = SaveConfig("config.json", activeConfig)

		// Dynamic Hot-Reload of MCP Servers!
		cleanupMCPServers()
		activeMCPServers = nil // Reset active list
		bootstrapMCPServers()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	// POST /api/mcp/delete
	mux.HandleFunc("/api/mcp/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if activeConfig.MCPServers != nil {
			delete(activeConfig.MCPServers, req.Name)
		}
		_ = SaveConfig("config.json", activeConfig)

		// Dynamic Hot-Reload of MCP Servers!
		cleanupMCPServers()
		activeMCPServers = nil // Reset active list
		bootstrapMCPServers()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	// Serve workflows.json from disk
	mux.HandleFunc("/workflows.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		bytes, err := os.ReadFile(GetSystemPath("workflows.json"))
		if err != nil {
			// File missing on disk: serve the full built-in default set
			// (linear_chat + enhanced_cognition) rather than an empty stub,
			// and seed it to disk so it persists for future runs.
			_ = EnsureDefaultWorkflows()
			cfg := DefaultWorkflowConfig()
			encoded, encErr := json.MarshalIndent(cfg, "", "  ")
			if encErr != nil {
				http.Error(w, "Failed to encode default workflows", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(encoded)
			return
		}
		_, _ = w.Write(bytes)
	})

	// POST /api/workflows/save: Overwrites workflows.json on-disk and hot-swaps the active pipeline!
	mux.HandleFunc("/api/workflows/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		bytes, _ := json.MarshalIndent(req, "", "  ")
		err := os.WriteFile(GetSystemPath("workflows.json"), bytes, 0644)
		if err != nil {
			http.Error(w, "Failed to write workflows.json", http.StatusInternalServerError)
			return
		}

		// Hot-swap SSE session broadcast warning card!
		activeId := req["active_workflow"].(string)
		BroadcastSSE("turn_secured", map[string]interface{}{
			"turn_number": 0,
			"role":        "system",
			"name":        "system",
			"content":     fmt.Sprintf("🔄 **[WORKFLOW HOT-SWAPPED]** GoHarness execution core has compiled and hot-swapped to active pipeline: `%s`!", activeId),
		})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	// POST /api/workflows/activate: Switches the active workflow by id.
	// Used by the header dropdown and the /workflow <id> web slash command.
	// Validates the id exists and only rewrites the active_workflow pointer,
	// leaving any custom graph definitions untouched.
	mux.HandleFunc("/api/workflows/activate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := ActivateWorkflow(req.ID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Read back the activated workflow's display name.
		cfg, _ := LoadWorkflowConfig()
		name := req.ID
		if cfg != nil {
			if wf, ok := cfg.Workflows[req.ID]; ok {
				name = wf.Name
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "success",
			"active_workflow": req.ID,
			"name":            name,
		})
	})

	// 4. Start Server and print beautiful UX logs (Phase 8.4)
	serverAddr := fmt.Sprintf("0.0.0.0:%d", port)
	
	fmt.Printf("\n%s=======================================================%s\n", ColorBlue, ColorReset)
	fmt.Printf("%s   🚀 GOHARNESS WEB & GATEWAY SERVICES ACTIVE 🚀       %s\n", ColorBold+ColorGreen, ColorReset)
	fmt.Printf("%s=======================================================%s\n", ColorBlue, ColorReset)
	fmt.Printf("  💻 Web GUI Console:    %shttp://%s/%s\n", ColorBold+ColorCyan, serverAddr, ColorReset)
	if activeConfig.Web.APIGatewayEnabled {
		fmt.Printf("  🔌 OpenAI API Gateway: %shttp://%s/v1%s\n", ColorBold+ColorCyan, serverAddr, ColorReset)
	} else {
		fmt.Printf("  🔌 OpenAI API Gateway: %sDisabled in config.json%s\n", ColorRed, ColorReset)
	}
	fmt.Printf("%s=======================================================%s\n\n", ColorBlue, ColorReset)

	// 5. Auto-Launch browser in background if Web GUI is active
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
	workspacePath = filepath.Clean(workspacePath)
	activeConfig.Agent.WorkspaceDir = workspacePath
	os.MkdirAll(workspacePath, 0755)

	found := false
	for _, ws := range activeConfig.Agent.WorkspacesHistory {
		if filepath.Clean(ws) == workspacePath {
			found = true
			break
		}
	}
	if !found {
		activeConfig.Agent.WorkspacesHistory = append(activeConfig.Agent.WorkspacesHistory, workspacePath)
	}

	// Try to find existing sessions for this workspace and choose the most recent one
	sessionsRoot := filepath.Join(".goharness", "sessions")
	entries, err := os.ReadDir(sessionsRoot)
	var matchingSessions []SessionMeta
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				metaPath := filepath.Join(sessionsRoot, entry.Name(), "meta.json")
				if bytes, err := os.ReadFile(metaPath); err == nil {
					var meta SessionMeta
					if err := json.Unmarshal(bytes, &meta); err == nil {
						if filepath.Clean(meta.WorkspaceDir) == workspacePath {
							matchingSessions = append(matchingSessions, meta)
						}
					}
				}
			}
		}
	}

	if len(matchingSessions) > 0 {
		mostRecent := matchingSessions[0]
		for _, s := range matchingSessions {
			if s.CreatedAt > mostRecent.CreatedAt {
				mostRecent = s
			}
		}
		activeSessionID = mostRecent.SessionID
		_ = loadHistoryFromFiles()
		currentTurnNumber = findMaxTurnNumber(activeSessionID)
		BroadcastSSE("session_init", map[string]interface{}{"session_id": activeSessionID})
		return
	}

	// If no existing session found, create a brand new one
	activeSessionID = "sess_" + time.Now().Format("20060102-150405")
	sessionPath := filepath.Join(".goharness", "sessions", activeSessionID)
	os.MkdirAll(sessionPath, 0755)

	wsName := filepath.Base(workspacePath)
	if wsName == "." || wsName == "/" {
		wsName = "Default"
	}
	createSessionMeta(activeSessionID, workspacePath, "", "Workspace: "+wsName)
	currentTurnNumber = 0

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

// =================================================================
// 🧠 TOKENIZER AUXILIARY FUNCTIONS
// =================================================================

// tokenizeString tokenizes a string in a self-learning BPE-approximate loop (Phase 8.3)
func tokenizeString(text string) []int {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	var tokens []int
	words := strings.Fields(text)
	for _, word := range words {
		id, exists := wordToToken[word]
		if !exists {
			id = nextTokenID
			wordToToken[word] = id
			tokenToWord[id] = word
			nextTokenID++
		}
		tokens = append(tokens, id)
	}
	return tokens
}

// detokenizeSlice detokenizes an array of token IDs back into standard text (Phase 8.3)
func detokenizeSlice(tokens []int) string {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	var words []string
	for _, id := range tokens {
		word, exists := tokenToWord[id]
		if exists {
			words = append(words, word)
		} else {
			words = append(words, fmt.Sprintf("[tok_%d]", id))
		}
	}
	return strings.Join(words, " ")
}

// SnapshotMeta holds workspace snapshot metadata
type SnapshotMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	FileCount int    `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}

func getDirStats(dir string) (int, int64) {
	var count int
	var size int64
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			if strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, ".goharness") || strings.Contains(rel, ".git") || strings.Contains(rel, ".goharness") {
				return nil
			}
			count++
			size += info.Size()
		}
		return nil
	})
	return count, size
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, ".goharness") || strings.Contains(rel, ".git") || strings.Contains(rel, ".goharness") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		targetPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		return copyFile(path, targetPath)
	})
}

func clearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" || name == ".goharness" || name == "config.json" {
			continue
		}
		path := filepath.Join(dir, name)
		_ = os.RemoveAll(path)
	}
	return nil
}

func cleanWorkspaceName(path string) string {
	cleaned := filepath.Clean(path)
	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, "\\", "_")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	if cleaned == "" {
		cleaned = "default"
	}
	return cleaned
}
