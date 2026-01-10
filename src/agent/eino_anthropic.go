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

		if msg.Role == schema.User {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": msg.Content,
			})
		} else if msg.Role == schema.Assistant {
			content := []map[string]interface{}{}
			if msg.Content != "" {
				content = append(content, map[string]interface{}{
					"type": "text",
					"text": msg.Content,
				})
			}
			// Handle Tool Calls in Assistant Message
			for _, tc := range msg.ToolCalls {
				// Parse arguments if they are string
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					// Fallback if not JSON or empty
					args = map[string]interface{}{}
				}
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": args,
				})
			}
			messages = append(messages, map[string]interface{}{
				"role":    "assistant",
				"content": content,
			})
		} else if msg.Role == schema.Tool {
			// Tool Result
			// Anthropic expects tool results in a 'user' message block with type 'tool_result'
			content := []map[string]interface{}{
				{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID, // Eino schema field for ID
					"content":     msg.Content,
				},
			}
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": content,
			})
		}
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

	// Definition for response parsing
	type contentBlock struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	}

	var result struct {
		Content []contentBlock `json:"content"`
		Role    string         `json:"role"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Convert back to schema.Message
	responseMsg := &schema.Message{
		Role:    schema.Assistant,
		Content: "",
	}

	for _, block := range result.Content {
		if block.Type == "text" {
			responseMsg.Content += block.Text
		} else if block.Type == "tool_use" {
			// Append to ToolCalls
			responseMsg.ToolCalls = append(responseMsg.ToolCalls, schema.ToolCall{
				ID: block.ID,
				Function: schema.FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}

	return responseMsg, nil
}

func (m *AnthropicChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("streaming not supported yet")
}
