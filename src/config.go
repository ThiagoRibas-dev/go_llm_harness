package main

import (
	"encoding/json"
	"os"
)

// ANSI Color Codes for beautiful terminal outputs (shared globally)
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorBold    = "\033[1m"
)

// Config holds the complete GoHarness configuration layout
type Config struct {
	API           APIConfig                  `json:"api"`
	Agent         AgentConfig                `json:"agent"`
	Security      SecurityConfig             `json:"security"`
	DirectoryScan DirectoryScanConfig        `json:"directory_scan"`
	Compaction    CompactionConfig           `json:"compaction"`
	MCPServers    map[string]MCPServerConfig `json:"mcp_servers"`
	Web           WebConfig                  `json:"web"`
}

type APIConfig struct {
	Provider      string  `json:"provider"`       // "openai", "anthropic", "gemini", "vertex"
	Key           string  `json:"key"`            // API Key or Access Token
	BaseURL       string  `json:"base_url"`       // Manual Override URL (if non-empty)
	Model         string  `json:"model"`          // Target model name
	Temperature   float64 `json:"temperature"`     // LLM Temperature
	MaxTokens     int     `json:"max_tokens"`      // Maximum output token ceiling
	TopP          float64 `json:"top_p"`          // Top-P sampling (Phase 8.6)
	TopK          int     `json:"top_k"`          // Top-K sampling (Phase 8.6)
	ThinkingLevel string  `json:"thinking_level"` // Thinking / Reasoning level: "off", "low", "medium", "high" (Phase 8.6)
	ProjectID     string  `json:"project_id"`     // GCP Project ID for Vertex AI (Phase 8.6)
	Region        string  `json:"region"`         // GCP Region for Vertex AI (Phase 8.6)
}

type AgentConfig struct {
	WorkspaceDir          string   `json:"workspace_dir"`
	WorkspacesHistory     []string `json:"workspaces_history"` // Workspace History (Phase 6.3)
	MaxTurns              int      `json:"max_turns"`
	CommandTimeoutSeconds int      `json:"command_timeout_seconds"`
}

type SecurityConfig struct {
	SandboxMode     string   `json:"sandbox_mode"`
	DockerContainer string   `json:"docker_container"`
	AllowedTools    []string `json:"allowed_tools"`
	BlockedPatterns []string `json:"blocked_patterns"`
}

type DirectoryScanConfig struct {
	MaxDepth             int      `json:"max_depth"`
	MaxFilesPerDirectory int      `json:"max_files_per_directory"`
	IgnoredPatterns      []string `json:"ignored_patterns"`
	CollapsedPatterns    []string `json:"collapsed_patterns"`
}

type CompactionConfig struct {
	AutoCompactTurns int     `json:"auto_compact_turns"`
	KeepLastN        int     `json:"keep_last_n"`
	Model            string  `json:"model"`
	Temperature      float64 `json:"temperature"`
	SystemPrompt     string  `json:"system_prompt"`
}

type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type WebConfig struct {
	Enabled           bool `json:"enabled"`             // Auto-start Web server on boot
	Port              int  `json:"port"`                // Port to bind Web GUI and Gateway
	APIGatewayEnabled bool `json:"api_gateway_enabled"` // Toggle OpenAI API Gateway endpoints
}

// SessionMeta holds metadata for each conversation session
type SessionMeta struct {
	SessionID       string `json:"session_id"`
	WorkspaceDir    string `json:"workspace_dir"`
	CreatedAt       string `json:"created_at"`
	Name            string `json:"name"`
	ParentSessionID string `json:"parent_session_id,omitempty"`
}

// OpenAI Chat Completion API Schema structures
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID               string       `json:"id"`
	Type             string       `json:"type"`
	Function         ToolFunction `json:"function"`
	ThoughtSignature string       `json:"thought_signature,omitempty"` // Google Gemini reasoning trace (Phase 8.6)
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDescriptor `json:"function"`
}

type FunctionDescriptor struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Temperature float64   `json:"temperature"`
}

type ChatCompletionResponse struct {
	Choices []Choice    `json:"choices"`
	Usage   OpenAIUsage `json:"usage,omitempty"` // Captured for real-time tracking (Phase 8.6)
}

type Choice struct {
	Message Message `json:"message"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Global runtime variables
var (
	activeConfig      *Config
	activeSessionID   string
	currentTurnNumber int
	mcpToolsMap       map[string]string // Maps ToolName ➔ MCPServerName
)

// LoadConfig reads and parses the config.json file
func LoadConfig(filename string) (*Config, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(fileBytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveConfig serializes our configuration to a formatted JSON file
func SaveConfig(filename string, config *Config) error {
	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, jsonBytes, 0644)
}
