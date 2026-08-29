package main

import "sync"

var (
	sessionRunMu     sync.Mutex
	sessionRunCounts = map[string]int{}
)

func beginSessionRun(sessionID string) func() {
	if sessionID == "" {
		return func() {}
	}
	sessionRunMu.Lock()
	sessionRunCounts[sessionID]++
	count := sessionRunCounts[sessionID]
	sessionRunMu.Unlock()
	if count == 1 {
		BroadcastSSE("run_state", map[string]interface{}{
			"session_id": sessionID,
			"running":    true,
		})
	}
	return func() {
		sessionRunMu.Lock()
		count := sessionRunCounts[sessionID]
		if count <= 1 {
			delete(sessionRunCounts, sessionID)
			count = 0
		} else {
			sessionRunCounts[sessionID] = count - 1
			count = sessionRunCounts[sessionID]
		}
		sessionRunMu.Unlock()
		if count == 0 {
			BroadcastSSE("run_state", map[string]interface{}{
				"session_id": sessionID,
				"running":    false,
			})
		}
	}
}

func isSessionRunning(sessionID string) bool {
	sessionRunMu.Lock()
	defer sessionRunMu.Unlock()
	return sessionRunCounts[sessionID] > 0
}
