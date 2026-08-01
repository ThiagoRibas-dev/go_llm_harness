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

// SendMultiProviderRequest detects the configured provider and routes the request.
// It also parses usage statistics and broadcasts real-time financial tracking to the UI.
func SendMultiProviderRequest(messages []Message, tools []Tool) (*Message, error) {
	provider := strings.ToLower(activeConfig.API.Provider)
	if provider == "" {
		provider = "openai"
	}

	var respMsg *Message
	var err error

	// Track cumulative character count of all raw content (as a backup metric if tokenizers are missing)
	var charCount int
	for _, m := range messages {
		charCount += len(m.Content)
	}

	var promptTokens, completionTokens, totalTokens int

	switch provider {
	case "anthropic":
		// Custom Anthropic request
		var antResp AnthropicResponse
		respMsg, antResp, err = sendAnthropicRequestWithUsage(messages, tools)
		if err == nil {
			promptTokens = antResp.Usage.InputTokens
			completionTokens = antResp.Usage.OutputTokens
			totalTokens = promptTokens + completionTokens
		}
	case "gemini", "vertex":
		// Custom Gemini/Vertex request
		var gemResp GeminiResponse
		respMsg, gemResp, err = sendGeminiRequestWithUsage(messages, tools, provider == "vertex")
		if err == nil {
			promptTokens = gemResp.UsageMetadata.PromptTokenCount
			completionTokens = gemResp.UsageMetadata.CandidatesTokenCount
			totalTokens = gemResp.UsageMetadata.TotalTokenCount
		}
	default:
		// Default OpenAI compatible request
		var oaiResp ChatCompletionResponse
		respMsg, oaiResp, err = sendOpenAIRequestWithUsage(messages, tools)
		if err == nil {
			promptTokens = oaiResp.Usage.PromptTokens
			completionTokens = oaiResp.Usage.CompletionTokens
			totalTokens = oaiResp.Usage.TotalTokens
		}
	}

	// Calculate and broadcast real-time cost and token diagnostics (Phase 8.6)
	if err == nil {
		charCount += len(respMsg.Content)
		cost := calculateActiveCost(activeConfig.API.Model, promptTokens, completionTokens)
		
		// Broadcast metrics to the Web Console SSE channel
		BroadcastSSE("cost_update", map[string]interface{}{
			"cost":              cost,
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      totalTokens,
			"char_count":        charCount,
		})
	}

	return respMsg, err
}

// =================================================================
// 💰 DYNAMIC MONETARY PRICING ENGINE
// =================================================================

func calculateActiveCost(model string, promptTokens, completionTokens int) float64 {
	model = strings.ToLower(model)
	baseURL := strings.ToLower(activeConfig.API.BaseURL)

	// If running fully locally via Ollama / llama-server, cost is ABSOLUTELY FREE ($0.00)!
	if strings.Contains(baseURL, "localhost") || strings.Contains(baseURL, "127.0.0.1") || strings.Contains(baseURL, "11434") {
		return 0.0
	}

	// Default Fallback Rates: highly affordable gpt-4o-mini style rates
	inputRate := 0.15 / 1000000.0  // $0.15 per Million tokens
	outputRate := 0.60 / 1000000.0 // $0.60 per Million tokens

	if strings.Contains(model, "gpt-4o-mini") {
		inputRate = 0.15 / 1000000.0
		outputRate = 0.60 / 1000000.0
	} else if strings.Contains(model, "gpt-4o") {
		inputRate = 2.50 / 1000000.0
		outputRate = 10.00 / 1000000.0
	} else if strings.Contains(model, "claude-3-5-sonnet") {
		inputRate = 3.00 / 1000000.0
		outputRate = 15.00 / 1000000.0
	} else if strings.Contains(model, "claude-3-5-haiku") {
		inputRate = 0.80 / 1000000.0
		outputRate = 4.00 / 1000000.0
	} else if strings.Contains(model, "gemini-1.5-flash") || strings.Contains(model, "gemini-3.1-flash") {
		inputRate = 0.075 / 1000000.0
		outputRate = 0.30 / 1000000.0
	} else if strings.Contains(model, "gemini-1.5-pro") || strings.Contains(model, "gemini-3.1-pro") {
		inputRate = 1.25 / 1000000.0
		outputRate = 5.00 / 1000000.0
	}

	return (float64(promptTokens) * inputRate) + (float64(completionTokens) * outputRate)
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
	Usage   AnthropicUsage     `json:"usage,omitempty"` // Captured for real-time tracking
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func sendAnthropicRequestWithUsage(messages []Message, tools []Tool) (*Message, AnthropicResponse, error) {
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

		if m.Role == "tool" {
			antMsg.Role = "user"
			antMsg.Content = append(antMsg.Content, AnthropicContent{
				Type:       "tool_result",
				ToolCallID: m.ToolCallID,
				Content:    m.Content,
			})
		} else if m.Role == "assistant" && len(m.ToolCalls) > 0 {
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
		return nil, AnthropicResponse{}, err
	}

	// 4. Dispatch HTTP Call
	url := activeConfig.API.BaseURL
	if url == "" || strings.Contains(url, "api.openai.com") {
		url = "https://api.anthropic.com/v1/messages"
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, AnthropicResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", activeConfig.API.Key)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, AnthropicResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, AnthropicResponse{}, fmt.Errorf("anthropic returned status %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var antResp AnthropicResponse
	if err := json.Unmarshal(bodyBytes, &antResp); err != nil {
		return nil, AnthropicResponse{}, err
	}

	if antResp.Error != nil {
		return nil, AnthropicResponse{}, fmt.Errorf("anthropic error: %s (%s)", antResp.Error.Message, antResp.Error.Type)
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

	return &result, antResp, nil
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
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	Name             string          `json:"name"`
	Args             json.RawMessage `json:"args"`
	ThoughtSignature string          `json:"thoughtSignature,omitempty"`
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
	UsageMetadata GeminiUsage `json:"usageMetadata,omitempty"` // Captured for real-time tracking
}

type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func sendGeminiRequestWithUsage(messages []Message, tools []Tool, isVertex bool) (*Message, GeminiResponse, error) {
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
			continue
		}

		var gemContent GeminiContent
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
						Name:             tc.Function.Name,
						Args:             json.RawMessage(tc.Function.Arguments),
						ThoughtSignature: tc.ThoughtSignature,
					},
				})
			}
		} else if m.Role == "tool" {
			gemContent.Role = "user"
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
		return nil, GeminiResponse{}, err
	}

	// 4. Construct API Endpoint URL
	url := activeConfig.API.BaseURL
	if isVertex {
		if url == "" || strings.Contains(url, "googleapis.com") {
			endpoint := "aiplatform.googleapis.com"
			if activeConfig.API.Region != "" {
				endpoint = fmt.Sprintf("%s-aiplatform.googleapis.com", activeConfig.API.Region)
			}

			if activeConfig.API.ProjectID != "" && activeConfig.API.Region != "" {
				url = fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent?key=%s", 
					endpoint, activeConfig.API.ProjectID, activeConfig.API.Region, activeConfig.API.Model, activeConfig.API.Key)
			} else {
				url = fmt.Sprintf("https://%s/v1/publishers/google/models/%s:generateContent?key=%s", 
					endpoint, activeConfig.API.Model, activeConfig.API.Key)
			}
		} else if !strings.Contains(url, "?key=") && activeConfig.API.Key != "" {
			url = fmt.Sprintf("%s?key=%s", strings.TrimSuffix(url, "/"), activeConfig.API.Key)
		}
	} else {
		if url == "" || strings.Contains(url, "api.openai.com") {
			url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", activeConfig.API.Model, activeConfig.API.Key)
		} else if !strings.Contains(url, "?key=") && activeConfig.API.Key != "" {
			url = fmt.Sprintf("%s?key=%s", strings.TrimSuffix(url, "/"), activeConfig.API.Key)
		}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, GeminiResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	
	if isVertex && !strings.Contains(url, "?key=") {
		req.Header.Set("Authorization", "Bearer "+activeConfig.API.Key)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, GeminiResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, GeminiResponse{}, fmt.Errorf("gemini returned status %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var gemResp GeminiResponse
	if err := json.Unmarshal(bodyBytes, &gemResp); err != nil {
		return nil, GeminiResponse{}, err
	}

	if len(gemResp.Candidates) == 0 {
		return nil, GeminiResponse{}, fmt.Errorf("gemini returned zero content candidates")
	}

	// 5. Translate Gemini response back to standard Message
	var result Message
	result.Role = "assistant"

	for _, part := range gemResp.Candidates[0].Content.Parts {
		if part.Text != "" {
			result.Content = part.Text
		}
		if part.FunctionCall != nil {
			toolID := fmt.Sprintf("call_%d", time.Now().UnixNano())
			argsBytes, _ := json.Marshal(part.FunctionCall.Args)
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   toolID,
				Type: "function",
				Function: ToolFunction{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsBytes),
				},
				ThoughtSignature: part.FunctionCall.ThoughtSignature,
			})
		}
	}

	return &result, gemResp, nil
}

// =================================================================
// 🌐 STANDARD OPENAI (OPENAI-COMPATIBLE API) CONNECTOR
// =================================================================

func sendOpenAIRequestWithUsage(messages []Message, tools []Tool) (*Message, ChatCompletionResponse, error) {
	reqBody := ChatCompletionRequest{
		Model:       activeConfig.API.Model,
		Messages:    messages,
		Tools:       tools,
		Temperature: activeConfig.API.Temperature,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, ChatCompletionResponse{}, err
	}

	req, err := http.NewRequest("POST", activeConfig.API.BaseURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, ChatCompletionResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	if activeConfig.API.Key != "" {
		req.Header.Set("Authorization", "Bearer "+activeConfig.API.Key)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ChatCompletionResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ChatCompletionResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ChatCompletionResponse{}, fmt.Errorf("api returned status %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, ChatCompletionResponse{}, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, ChatCompletionResponse{}, fmt.Errorf("api response contained zero completion choices")
	}

	return &chatResp.Choices[0].Message, chatResp, nil
}
