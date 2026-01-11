package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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

	respBody, _ := io.ReadAll(resp.Body)

	// Debug logging
	if logPath, ok := ctx.Value("debug_log_path").(string); ok && logPath != "" {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			f.WriteString(fmt.Sprintf("\n--- Request [%s] ---\n", time.Now().Format(time.RFC3339)))
			f.Write(jsonBody)
			f.WriteString("\n--- Response ---\n")
			f.WriteString(fmt.Sprintf("Status: %d\n", resp.StatusCode))
			f.WriteString(fmt.Sprintf("Body Length: %d bytes\n", len(respBody)))
			// Write full response for debugging tool calls
			f.Write(respBody)
			f.WriteString("\n")
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, string(respBody))
	}

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

	if err := json.Unmarshal(respBody, &result); err != nil {
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
			// Convert Input (json.RawMessage) to proper JSON string
			var inputStr string
			if len(block.Input) > 0 {
				// Check if Input is valid JSON first
				var inputObj map[string]interface{}
				if err := json.Unmarshal(block.Input, &inputObj); err == nil {
					// Re-marshal to ensure valid JSON string
					if inputBytes, err := json.Marshal(inputObj); err == nil {
						inputStr = string(inputBytes)
					} else {
						// Marshal failed - log to debug file if available
						if logPath, ok := ctx.Value("debug_log_path").(string); ok && logPath != "" {
							f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
							if f != nil {
								f.WriteString(fmt.Sprintf("\n[WARNING] Failed to re-marshal tool input: %v\n", err))
								f.WriteString(fmt.Sprintf("Raw Input: %s\n", string(block.Input)))
								f.Close()
							}
						}
						inputStr = string(block.Input)
					}
				} else {
					// Unmarshal failed - this is the problem
					if logPath, ok := ctx.Value("debug_log_path").(string); ok && logPath != "" {
						f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
						if f != nil {
							f.WriteString(fmt.Sprintf("\n[ERROR] Failed to unmarshal tool input: %v\n", err))
							f.WriteString(fmt.Sprintf("Tool: %s, ID: %s\n", block.Name, block.ID))
							f.WriteString(fmt.Sprintf("Raw Input Length: %d bytes\n", len(block.Input)))
							f.WriteString(fmt.Sprintf("Raw Input (first 1000 chars): %s\n", string(block.Input[:min(1000, len(block.Input))])))
							f.Close()
						}
					}
					// Try to use it as-is
					inputStr = string(block.Input)
				}
			} else {
				inputStr = "{}"
			}

			responseMsg.ToolCalls = append(responseMsg.ToolCalls, schema.ToolCall{
				ID: block.ID,
				Function: schema.FunctionCall{
					Name:      block.Name,
					Arguments: inputStr,
				},
			})
		}
	}

	return responseMsg, nil
}

func (m *AnthropicChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("streaming not supported yet")
}
