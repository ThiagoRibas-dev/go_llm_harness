package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
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
		activeConfig.API.Key = req.APIKey
		activeConfig.API.Model = req.Model
		activeConfig.API.BaseURL = req.BaseURL
		activeConfig.Security.SandboxMode = req.SandboxMode
		activeConfig.Agent.MaxTurns = req.MaxTurns
		activeConfig.Agent.WorkspaceDir = req.WorkspaceDir

		// Overwrite config.json on host disk
		err := SaveConfig("config.json", activeConfig)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
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
