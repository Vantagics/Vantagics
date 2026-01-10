package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestAnthropicChatModel_Generate_ToolUse(t *testing.T) {
	// Mock Anthropic API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Request
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Check system prompt
		if reqBody["system"] != "System Prompt" {
			t.Errorf("Expected system prompt 'System Prompt', got %v", reqBody["system"])
		}

		// Check messages
		msgs := reqBody["messages"].([]interface{})
		if len(msgs) != 1 {
			t.Errorf("Expected 1 message, got %d", len(msgs))
		}

		// Verify User Message
		msg0 := msgs[0].(map[string]interface{})
		if msg0["role"] != "user" || msg0["content"] != "User Input" {
			t.Errorf("Message 0 mismatch")
		}

		// Respond with Tool Use
		resp := map[string]interface{}{
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "I will call a tool.",
				},
				{
					"type": "tool_use",
					"id":   "tool_123",
					"name": "test_tool",
					"input": map[string]interface{}{
						"arg": "val",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	model, _ := NewAnthropicChatModel(context.Background(), &AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3",
	})

	input := []*schema.Message{
		{Role: schema.System, Content: "System Prompt"},
		{Role: schema.User, Content: "User Input"},
	}

	resp, err := model.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp.Content != "I will call a tool." {
		t.Errorf("Expected content 'I will call a tool.', got '%s'", resp.Content)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].ID != "tool_123" {
		t.Errorf("Expected tool ID 'tool_123', got '%s'", resp.ToolCalls[0].ID)
	}
	
	if resp.ToolCalls[0].Function.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", resp.ToolCalls[0].Function.Name)
	}
}

func TestAnthropicChatModel_Generate_ToolResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		msgs := reqBody["messages"].([]interface{})
		// Expecting: User, Assistant (Tool Use), User (Tool Result)
		if len(msgs) != 3 {
			t.Fatalf("Expected 3 messages, got %d", len(msgs))
		}

		// Check Tool Result Message (Last one)
		lastMsg := msgs[2].(map[string]interface{})
		if lastMsg["role"] != "user" {
			t.Errorf("Expected tool result to be role 'user', got '%s'", lastMsg["role"])
		}
		
		content := lastMsg["content"].([]interface{})
		block := content[0].(map[string]interface{})
		
		if block["type"] != "tool_result" {
			t.Errorf("Expected type 'tool_result', got '%v'", block["type"])
		}
		if block["tool_use_id"] != "call_1" {
			t.Errorf("Expected tool_use_id 'call_1', got '%v'", block["tool_use_id"])
		}
		if block["content"] != "Result 1" {
			t.Errorf("Expected content 'Result 1', got '%v'", block["content"])
		}

		// Respond
		json.NewEncoder(w).Encode(map[string]interface{}{
			"role": "assistant",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Final Answer"},
			},
		})
	}))
	defer server.Close()

	model, _ := NewAnthropicChatModel(context.Background(), &AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	input := []*schema.Message{
		{Role: schema.User, Content: "Start"},
		{
			Role: schema.Assistant, 
			ToolCalls: []schema.ToolCall{
				{ID: "call_1", Function: schema.FunctionCall{Name: "tool", Arguments: "{}"}},
			},
		},
		{Role: schema.Tool, Content: "Result 1", ToolCallID: "call_1"},
	}

	_, err := model.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}
