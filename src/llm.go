package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// =================================================================
// 🔌 UNIFIED MULTI-PROVIDER DISPATCHER
// =================================================================

// SendMultiProviderRequest detects the configured provider and routes the request
func SendMultiProviderRequest(messages []Message, tools []Tool) (*Message, error) {
	provider := strings.ToLower(activeConfig.API.Provider)
	if provider == "" {
		provider = "openai"
	}

	var respMsg *Message
	var err error

	switch provider {
	case "anthropic":
		respMsg, err = sendAnthropicRequest(messages, tools)
	case "gemini", "vertex":
		respMsg, err = sendGeminiRequest(messages, tools, provider == "vertex")
	default:
		// Default to OpenAI-compatible payload
		respMsg, err = sendOpenAIRequest(messages, tools)
	}

	// Trigger real-time visual cost trackers and logs (Phase 6.3)
	if err == nil {
		BroadcastSSE("cost_update", map[string]interface{}{
			"cost": 0.0005, // Simulated cost increment per turn
		})
	}

	return respMsg, err
}

// =================================================================
// 🅰️ ANTHROPIC (CLAUDE MESSAGES API) CONNECTOR
// =================================================================

type AnthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"` // System prompts are top-level in Anthropic
	Messages    []AnthropicMessage `json:"messages"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
}

type AnthropicMessage struct {
	Role    string             `json:"role"`
	Content []AnthropicContent `json:"content"`
}

type AnthropicContent struct {
	Type       string           `json:"type"`
	Text       string           `json:"text,omitempty"`
	ID         string           `json:"id,omitempty"`
	Name       string           `json:"name,omitempty"`
	Input      *json.RawMessage `json:"input,omitempty"`
	ToolCallID string           `json:"tool_use_id,omitempty"` // for tool response matching
	Content    string           `json:"content,omitempty"`     // for tool response matching
}

type AnthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
	Role    string             `json:"role"`
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func sendAnthropicRequest(messages []Message, tools []Tool) (*Message, error) {
	var antReq AnthropicRequest
	antReq.Model = activeConfig.API.Model
	antReq.MaxTokens = activeConfig.API.MaxTokens
	antReq.Temperature = activeConfig.API.Temperature

	// 1. Separate System Prompt from core Messages
	var cleanMessages []Message
	for _, m := range messages {
		if m.Role == "system" {
			if antReq.System != "" {
				antReq.System += "\n" + m.Content
			} else {
				antReq.System = m.Content
			}
		} else {
			cleanMessages = append(cleanMessages, m)
		}
	}

	// 2. Translate Message slices
	for _, m := range cleanMessages {
		var antMsg AnthropicMessage
		antMsg.Role = m.Role

		// Anthropic roles mapping: 'tool' becomes 'user' with a 'tool_result' block type
		if m.Role == "tool" {
			antMsg.Role = "user"
			antMsg.Content = append(antMsg.Content, AnthropicContent{
				Type:       "tool_result",
				ToolCallID: m.ToolCallID,
				Content:    m.Content,
			})
		} else if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			// Assistant response calling tools
			if m.Content != "" {
				antMsg.Content = append(antMsg.Content, AnthropicContent{
					Type: "text",
					Text: m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				rawArgs := json.RawMessage(tc.Function.Arguments)
				antMsg.Content = append(antMsg.Content, AnthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: &rawArgs,
				})
			}
		} else {
			// Standard User or Assistant Text Message
			antMsg.Content = append(antMsg.Content, AnthropicContent{
				Type: "text",
				Text: m.Content,
			})
		}
		antReq.Messages = append(antReq.Messages, antMsg)
	}

	// 3. Translate Tool Schemas
	for _, t := range tools {
		antReq.Tools = append(antReq.Tools, AnthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}

	payload, err := json.Marshal(antReq)
	if err != nil {
		return nil, err
	}

	// 4. Dispatch HTTP Call
	url := activeConfig.API.BaseURL
	if url == "" || strings.Contains(url, "api.openai.com") {
		url = "https://api.anthropic.com/v1/messages" // Fallback to official endpoint
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", activeConfig.API.Key)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic returned status %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var antResp AnthropicResponse
	if err := json.Unmarshal(bodyBytes, &antResp); err != nil {
		return nil, err
	}

	if antResp.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s (%s)", antResp.Error.Message, antResp.Error.Type)
	}

	// 5. Translate Anthropic Response back to standard message
	var result Message
	result.Role = "assistant"

	for _, content := range antResp.Content {
		if content.Type == "text" {
			result.Content = content.Text
		} else if content.Type == "tool_use" {
			argsBytes, _ := json.Marshal(content.Input)
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: ToolFunction{
					Name:      content.Name,
					Arguments: string(argsBytes),
				},
			})
		}
	}

	return &result, nil
}

// =================================================================
// ♊ GOOGLE GEMINI (AI STUDIO & VERTEX AI) CONNECTOR
// =================================================================

type GeminiRequest struct {
	Contents          []GeminiContent    `json:"contents"`
	SystemInstruction *GeminiInstruction `json:"systemInstruction,omitempty"`
	Tools             []GeminiToolGroup  `json:"tools,omitempty"`
}

type GeminiInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text             string                 `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall    `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type GeminiToolGroup struct {
	FunctionDeclarations []GeminiTool `json:"functionDeclarations"`
}

type GeminiTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []GeminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
	} `json:"candidates"`
}

func sendGeminiRequest(messages []Message, tools []Tool, isVertex bool) (*Message, error) {
	var gemReq GeminiRequest

	// 1. Separate System instruction
	for _, m := range messages {
		if m.Role == "system" {
			if gemReq.SystemInstruction == nil {
				gemReq.SystemInstruction = &GeminiInstruction{}
			}
			gemReq.SystemInstruction.Parts = append(gemReq.SystemInstruction.Parts, GeminiPart{Text: m.Content})
		}
	}

	// 2. Translate conversation contents
	for _, m := range messages {
		if m.Role == "system" {
			continue // Already separated
		}

		var gemContent GeminiContent
		// Gemini Roles: 'user' or 'model' (for assistant)
		if m.Role == "user" {
			gemContent.Role = "user"
			gemContent.Parts = append(gemContent.Parts, GeminiPart{Text: m.Content})
		} else if m.Role == "assistant" {
			gemContent.Role = "model"
			if m.Content != "" {
				gemContent.Parts = append(gemContent.Parts, GeminiPart{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				gemContent.Parts = append(gemContent.Parts, GeminiPart{
					FunctionCall: &GeminiFunctionCall{
						Name: tc.Function.Name,
						Args: json.RawMessage(tc.Function.Arguments),
					},
				})
			}
		} else if m.Role == "tool" {
			// Tool output maps back to role 'user' with a functionResponse part
			gemContent.Role = "user"
			// Gemini expects function response wrapped inside a JSON object wrapper
			gemContent.Parts = append(gemContent.Parts, GeminiPart{
				FunctionResponse: &GeminiFunctionResponse{
					Name: m.Name,
					Response: map[string]interface{}{
						"output": m.Content,
					},
				},
			})
		}
		gemReq.Contents = append(gemReq.Contents, gemContent)
	}

	// 3. Translate Tool declarations
	if len(tools) > 0 {
		var toolGroup GeminiToolGroup
		for _, t := range tools {
			toolGroup.FunctionDeclarations = append(toolGroup.FunctionDeclarations, GeminiTool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}
		gemReq.Tools = append(gemReq.Tools, toolGroup)
	}

	payload, err := json.Marshal(gemReq)
	if err != nil {
		return nil, err
	}

	// 4. Construct API Endpoint URL
	url := activeConfig.API.BaseURL
	if isVertex {
		// Vertex AI utilizes Bearer OAuth tokens and a Google Cloud REST endpoint
		if url == "" || strings.Contains(url, "googleapis.com") {
			// Fallback placeholder structure
			url = fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/publishers/google/models/%s:generateContent", activeConfig.API.Model)
		}
	} else {
		// Google AI Studio uses standard API key in query parameters
		if url == "" || strings.Contains(url, "api.openai.com") {
			url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", activeConfig.API.Model, activeConfig.API.Key)
		} else if !strings.Contains(url, "?key=") {
			url = fmt.Sprintf("%s?key=%s", strings.TrimSuffix(url, "/"), activeConfig.API.Key)
		}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if isVertex {
		req.Header.Set("Authorization", "Bearer "+activeConfig.API.Key)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini returned status %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var gemResp GeminiResponse
	if err := json.Unmarshal(bodyBytes, &gemResp); err != nil {
		return nil, err
	}

	if len(gemResp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini returned zero content candidates")
	}

	// 5. Translate Gemini response back to standard Message
	var result Message
	result.Role = "assistant"

	for _, part := range gemResp.Candidates[0].Content.Parts {
		if part.Text != "" {
			result.Content = part.Text
		}
		if part.FunctionCall != nil {
			// Generate a unique tool ID
			toolID := fmt.Sprintf("call_%d", time.Now().UnixNano())
			argsBytes, _ := json.Marshal(part.FunctionCall.Args)
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   toolID,
				Type: "function",
				Function: ToolFunction{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsBytes),
				},
			})
		}
	}

	return &result, nil
}

// =================================================================
// 🌐 STANDARD OPENAI (OPENAI-COMPATIBLE API) CONNECTOR
// =================================================================

func sendOpenAIRequest(messages []Message, tools []Tool) (*Message, error) {
	reqBody := ChatCompletionRequest{
		Model:       activeConfig.API.Model,
		Messages:    messages,
		Tools:       tools,
		Temperature: activeConfig.API.Temperature,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", activeConfig.API.BaseURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if activeConfig.API.Key != "" {
		req.Header.Set("Authorization", "Bearer "+activeConfig.API.Key)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("api response contained zero completion choices")
	}

	return &chatResp.Choices[0].Message, nil
}
