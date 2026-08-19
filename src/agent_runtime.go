package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MaxSubAgentDepth caps recursion when an agent spawns sub-agents.
// 0 = root agent, 1 = its children, 2 = grandchildren (hard limit).
const MaxSubAgentDepth = 2

// Agent carries all per-run identity and connection state so the ReAct loop
// can execute concurrently for multiple sessions without touching package
// globals. A root agent is created by the TUI/web entry points via
// NewRootAgent; a sub-agent is created by the spawn_sub_agent tool via Sub.
type Agent struct {
	SessionID   string
	Workspace   string
	API         APIConfig // resolved connection this agent uses
	ProfileName string    // profile that resolved API (for throttle keying); "" for inline/global
	Depth       int       // 0 for the root/main agent

	turn      int                      // next turn number for this session (local, not global)
	throttlers map[string]chan struct{} // per-profile concurrency semaphores
	mu        sync.Mutex
}

// NewRootAgent builds the agent for the active UI/TUI session using the
// process-wide activeConfig and activeSessionID. It is the one place that
// reads the package-globals that represent "the current user session".
func NewRootAgent() *Agent {
	a := &Agent{
		SessionID:   activeSessionID,
		Workspace:   activeConfig.Agent.WorkspaceDir,
		API:         activeConfig.ResolveAPIConfig(),
		ProfileName: activeConfig.ProviderProfile,
		throttlers:  make(map[string]chan struct{}),
	}
	a.turn = findMaxTurnNumber(a.SessionID)
	return a
}

// Sub creates a child agent that runs in its own isolated session. The child
// inherits the parent's workspace, connection, and throttle map so that
// fan-out is capped by the profile's MaxConcurrency.
func (a *Agent) Sub(sessionID string) *Agent {
	return &Agent{
		SessionID:   sessionID,
		Workspace:   a.Workspace,
		API:         a.API,
		ProfileName: a.ProfileName,
		Depth:       a.Depth + 1,
		throttlers:  a.throttlers,
	}
}

// sessionPath is the on-disk folder for this agent's turns.
func (a *Agent) sessionPath() string {
	return GetSystemPath(filepath.Join(".goharness", "sessions", a.SessionID))
}

// nextTurnNumber allocates and returns the next 1-based turn number for this
// agent. Safe for concurrent agents because each has its own counter.
func (a *Agent) nextTurnNumber() int {
	a.turn++
	return a.turn
}

// saveTurn writes a message to this agent's session folder as a numbered JSON
// file and broadcasts it to attached Web consoles. It replaces the global
// saveMessageTurn for agent-driven runs.
func (a *Agent) saveTurn(msg Message) {
	turn := a.nextTurnNumber()
	ts := time.Now().Format("2006_01_02_15-04-05")
	filename := fmt.Sprintf("%03d-%s-%s.json", turn, msg.Role, ts)
	path := filepath.Join(a.sessionPath(), filename)

	if bytes, err := json.MarshalIndent(msg, "", "  "); err == nil {
		_ = os.WriteFile(path, bytes, 0644)
		if activeConfig.Debug {
			fmt.Printf("[TURN SECURED] %s turn %d -> %s\n", a.SessionID, turn, filename)
		}
	}

	BroadcastSSE("turn_secured", map[string]interface{}{
		"turn_number": turn,
		"session_id":  a.SessionID,
		"role":        msg.Role,
		"name":        msg.Name,
		"content":     msg.Content,
		"tool_calls":  msg.ToolCalls,
	})
}

// loadHistory reads uncompacted turns from this agent's session folder.
func (a *Agent) loadHistory() []Message {
	entries, err := os.ReadDir(a.sessionPath())
	if err != nil {
		return []Message{}
	}
	var history []Message
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || len(name) <= 4 || name[3] != '-' {
			continue
		}
		if bytes, err := os.ReadFile(filepath.Join(a.sessionPath(), name)); err == nil {
			var msg Message
			if err := json.Unmarshal(bytes, &msg); err == nil {
				history = append(history, msg)
			}
		}
	}
	if history == nil {
		history = []Message{}
	}
	return history
}

// throttleKey identifies the concurrency bucket for a connection. Named
// profiles share a bucket; inline connections share by provider/model.
func throttleKey(profileName string, api APIConfig) string {
	if profileName != "" {
		return "profile:" + profileName
	}
	return "inline:" + api.Provider + "/" + api.Model
}

// acquireThrottle blocks until a concurrency slot is free for this agent's
// connection, returning a release function. MaxConcurrency<=0 is unlimited.
// Agents sharing a throttlers map (a parent and its children) respect one
// another, so fan-out onto a rate-limited profile queues instead of 429ing.
func (a *Agent) acquireThrottle(ctx context.Context, profileName string, api APIConfig) func() {
	limit := api.MaxConcurrency
	if limit <= 0 {
		return func() {}
	}
	key := throttleKey(profileName, api)
	a.mu.Lock()
	ch, ok := a.throttlers[key]
	if !ok {
		ch = make(chan struct{}, limit)
		a.throttlers[key] = ch
	}
	a.mu.Unlock()

	select {
	case ch <- struct{}{}:
		return func() { <-ch }
	case <-ctx.Done():
		return func() {}
	}
}

// request performs one LLM completion using this agent's connection,
// respecting its per-profile concurrency limit. This is the explicit-API
// replacement for SendMultiProviderRequest and removes the need to hot-swap
// the global activeConfig.API during runs.
func (a *Agent) request(ctx context.Context, messages []Message, tools []Tool) (*Message, error) {
	release := a.acquireThrottle(ctx, a.ProfileName, a.API)
	defer release()
	return sendProviderRequest(a.API, messages, tools)
}
