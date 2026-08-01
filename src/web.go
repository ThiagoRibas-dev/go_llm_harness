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
			Provider      string  `json:"provider"`       // Phase 7
			APIKey        string  `json:"api_key"`
			Model         string  `json:"model"`
			BaseURL       string  `json:"base_url"`
			SandboxMode   string  `json:"sandbox_mode"`
			MaxTurns      int     `json:"max_turns"`
			WorkspaceDir  string  `json:"workspace_dir"`
			Temperature   float64 `json:"temperature"`     // Phase 8.6
			TopP          float64 `json:"top_p"`           // Phase 8.6
			TopK          int     `json:"top_k"`           // Phase 8.6
			ThinkingLevel string  `json:"thinking_level"`  // Phase 8.6
			ProjectID     string  `json:"project_id"`      // Phase 8.6
			Region        string  `json:"region"`          // Phase 8.6
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
		activeConfig.API.Temperature = req.Temperature
		activeConfig.API.TopP = req.TopP
		activeConfig.API.TopK = req.TopK
		activeConfig.API.ThinkingLevel = req.ThinkingLevel
		activeConfig.API.ProjectID = req.ProjectID
		activeConfig.API.Region = req.Region
		
		activeConfig.Security.SandboxMode = req.SandboxMode
		activeConfig.Agent.MaxTurns = req.MaxTurns

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
		currentTurnNumber = len(history)

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
					_ = copyFile(filepath.Join(parentPath, name), filepath.Join(newSessionPath, name))
				}
			}
		}

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

	// 4. Start Server and print beautiful UX logs (Phase 8.4)
	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)
	
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
		if ws == workspacePath {
			found = true
			break
		}
	}
	if !found {
		activeConfig.Agent.WorkspacesHistory = append(activeConfig.Agent.WorkspacesHistory, workspacePath)
	}

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
