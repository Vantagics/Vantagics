package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// GeminiChatModel implements the Eino ChatModel interface for Google Gemini
type GeminiChatModel struct {
	config          *GeminiConfig
	tools           []*schema.ToolInfo
	thoughtSignature string // Store the thought signature from model responses
}

// GeminiConfig holds configuration for Gemini API
type GeminiConfig struct {
	APIKey    string
	BaseURL   string // Optional custom base URL
	Model     string // e.g., "gemini-1.5-pro", "gemini-1.5-flash", "gemini-2.0-flash-exp"
	MaxTokens int
}

// NewGeminiChatModel creates a new Gemini chat model
func NewGeminiChatModel(ctx context.Context, config *GeminiConfig) (*GeminiChatModel, error) {
	if config == nil {
		return nil, fmt.Errorf("gemini config is nil")
	}
	if config.Model == "" {
		return nil, fmt.Errorf("gemini model name is empty - please configure a valid model name")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is empty - please configure your API key")
	}

	return &GeminiChatModel{
		config: config,
	}, nil
}

// BindTools binds tools to the model for function calling
func (m *GeminiChatModel) BindTools(tools []*schema.ToolInfo) error {
	m.tools = tools
	return nil
}

// Generate sends a request to Gemini API and returns the response
func (m *GeminiChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.config.Model == "" {
		return nil, fmt.Errorf("model name is empty - please configure a valid model name in settings")
	}

	// Build request body
	reqBody := m.buildRequestBody(input)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Build URL
	baseURL := m.config.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	fullURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, m.config.Model, m.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return m.parseResponse(respBody)
}

// buildRequestBody constructs the Gemini API request body
func (m *GeminiChatModel) buildRequestBody(input []*schema.Message) map[string]interface{} {
	reqBody := map[string]interface{}{}

	// Build contents array
	var contents []map[string]interface{}
	var systemInstruction string

	for _, msg := range input {
		if msg.Role == schema.System {
			systemInstruction += msg.Content + "\n"
			continue
		}

		role := "user"
		if msg.Role == schema.Assistant {
			role = "model"
		} else if msg.Role == schema.Tool {
			// Tool response - for Gemini 2.0+, we need thought_signature for function responses
			functionResponsePart := map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name": msg.ToolCallID,
					"response": map[string]interface{}{
						"result": msg.Content,
					},
				},
			}
			// Add thought signature if available (required for Gemini 2.0 thinking models)
			if m.thoughtSignature != "" {
				functionResponsePart["thoughtSignature"] = m.thoughtSignature
			}
			contents = append(contents, map[string]interface{}{
				"role":  "user",
				"parts": []interface{}{functionResponsePart},
			})
			continue
		}

		// Build parts
		var parts []interface{}

		if msg.Content != "" {
			parts = append(parts, map[string]interface{}{
				"text": msg.Content,
			})
		}

		// Handle tool calls from assistant - include proper functionCall format
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				functionCallPart := map[string]interface{}{
					"functionCall": map[string]interface{}{
						"name": tc.Function.Name,
						"args": args,
					},
				}
				// Add thought signature if available
				if m.thoughtSignature != "" {
					functionCallPart["thoughtSignature"] = m.thoughtSignature
				}
				parts = append(parts, functionCallPart)
			}
		}

		if len(parts) > 0 {
			contents = append(contents, map[string]interface{}{
				"role":  role,
				"parts": parts,
			})
		}
	}

	reqBody["contents"] = contents

	// Add system instruction if present
	if systemInstruction != "" {
		reqBody["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": strings.TrimSpace(systemInstruction)},
			},
		}
	}

	// Add generation config
	generationConfig := map[string]interface{}{}
	if m.config.MaxTokens > 0 {
		generationConfig["maxOutputTokens"] = m.config.MaxTokens
	} else {
		generationConfig["maxOutputTokens"] = 8192
	}
	reqBody["generationConfig"] = generationConfig

	// Add tools if configured
	if len(m.tools) > 0 {
		var functionDeclarations []map[string]interface{}
		for _, tool := range m.tools {
			funcDecl := map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Desc,
			}
			if tool.ParamsOneOf != nil {
				funcDecl["parameters"] = tool.ParamsOneOf
			}
			functionDeclarations = append(functionDeclarations, funcDecl)
		}
		reqBody["tools"] = []map[string]interface{}{
			{"functionDeclarations": functionDeclarations},
		}
	}

	return reqBody
}

// parseResponse parses the Gemini API response
func (m *GeminiChatModel) parseResponse(respBody []byte) (*schema.Message, error) {
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text             string `json:"text,omitempty"`
					ThoughtSignature string `json:"thoughtSignature,omitempty"`
					FunctionCall     *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall,omitempty"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s (code: %d)", result.Error.Message, result.Error.Code)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}

	responseMsg := &schema.Message{
		Role:    schema.Assistant,
		Content: "",
	}

	candidate := result.Candidates[0]
	for _, part := range candidate.Content.Parts {
		// Capture thought signature for use in subsequent function responses
		if part.ThoughtSignature != "" {
			m.thoughtSignature = part.ThoughtSignature
		}
		if part.Text != "" {
			responseMsg.Content += part.Text
		}
		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			responseMsg.ToolCalls = append(responseMsg.ToolCalls, schema.ToolCall{
				ID: part.FunctionCall.Name, // Gemini doesn't provide separate ID, use name
				Function: schema.FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return responseMsg, nil
}

// Stream implements streaming response (not yet supported)
func (m *GeminiChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("streaming not supported yet for Gemini")
}
