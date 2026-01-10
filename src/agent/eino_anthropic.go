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

type AnthropicChatModel struct {
	config *AnthropicConfig
	tools  []*schema.ToolInfo
}

type AnthropicConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func NewAnthropicChatModel(ctx context.Context, config *AnthropicConfig) (*AnthropicChatModel, error) {
	return &AnthropicChatModel{
		config: config,
	},
	nil
}

func (m *AnthropicChatModel) BindTools(tools []*schema.ToolInfo) error {
	m.tools = tools
	return nil
}

func (m *AnthropicChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"model":      m.config.Model,
		"max_tokens": 4096,
	}

	var messages []map[string]interface{}
	var systemPrompt string

	for _, msg := range input {
		if msg.Role == schema.System {
			systemPrompt += msg.Content + "\n"
			continue
		}

		role := "user"
		if msg.Role == schema.Assistant {
			role = "assistant"
		}
		// TODO: Handle Tool messages (role "tool" in Eino? schema usually uses specialized roles or types)
		// For now, simple text content
		messages = append(messages, map[string]interface{}{
			"role":    role,
			"content": msg.Content,
		})
	}

	if systemPrompt != "" {
		reqBody["system"] = strings.TrimSpace(systemPrompt)
	}
	reqBody["messages"] = messages

	// Add tools if m.tools is set
	if len(m.tools) > 0 {
		var tools []map[string]interface{}
		for _, tool := range m.tools {
			t := map[string]interface{}{
				"name":         tool.Name,
				"description":  tool.Desc,
				"input_schema": tool.ParamsOneOf,
			}
			tools = append(tools, t)
		}
		reqBody["tools"] = tools
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Prepare URL
	fullURL := "https://api.anthropic.com/v1/messages"
	if m.config.BaseURL != "" {
		// Simple URL handling, assuming standard Anthropic compatible endpoint
		fullURL = strings.TrimSuffix(m.config.BaseURL, "/") + "/v1/messages"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", m.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"content"`
		Role string `json:"role"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Convert back to schema.Message
	responseMsg := &schema.Message{
		Role:    schema.Assistant, // Result role should be assistant
		Content: "",
	}

	for _, block := range result.Content {
		if block.Type == "text" {
			responseMsg.Content += block.Text
		}
	}

	return responseMsg, nil
}

func (m *AnthropicChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("streaming not supported yet")
}
