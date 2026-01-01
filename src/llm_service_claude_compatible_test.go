package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLLMServiceChat_ClaudeCompatible_AnthropicStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Anthropic headers
		if r.Header.Get("x-api-key") != "sk-ant-test" {
			t.Errorf("Expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header, got %s", r.Header.Get("anthropic-version"))
		}
		// Authorization header should NOT be present (or typically not used like Bearer)
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Did not expect Authorization header, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"text": "Hello from mock Claude Compatible (Anthropic Style)",
				},
			},
		})
	}))
	defer server.Close()

	config := Config{
		LLMProvider:       "Claude-Compatible",
		APIKey:            "sk-ant-test",
		ModelName:         "claude-3-custom",
		BaseURL:           server.URL,
		ClaudeHeaderStyle: "Anthropic", // New field
	}
	service := NewLLMService(config)

	resp, err := service.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp != "Hello from mock Claude Compatible (Anthropic Style)" {
		t.Errorf("Expected mock response, got %s", resp)
	}
}

func TestLLMServiceChat_ClaudeCompatible_OpenAIStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for OpenAI-style Bearer token
		if r.Header.Get("Authorization") != "Bearer sk-test-key" {
			t.Errorf("Expected Authorization header 'Bearer sk-test-key', got %s", r.Header.Get("Authorization"))
		}
		// Anthropic headers should NOT be present
		if r.Header.Get("x-api-key") != "" {
			t.Errorf("Did not expect x-api-key header, got %s", r.Header.Get("x-api-key"))
		}

		w.Header().Set("Content-Type", "application/json")
		// Note: Claude-Compatible proxies usually return Anthropic-format JSON even if auth is OpenAI-style,
		// OR they might transform it.
		// However, "Claude-Compatible" usually implies the *API Contract* (Request/Response body) is Claude's,
		// but the *Transport/Auth* might be flexible.
		// Let's assume the response body is still Anthropic format (content[0].text).
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"text": "Hello from mock Claude Compatible (OpenAI Style)",
				},
			},
		})
	}))
	defer server.Close()

	config := Config{
		LLMProvider:       "Claude-Compatible",
		APIKey:            "sk-test-key",
		ModelName:         "claude-3-custom",
		BaseURL:           server.URL,
		ClaudeHeaderStyle: "OpenAI", // New field
	}
	service := NewLLMService(config)

	resp, err := service.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp != "Hello from mock Claude Compatible (OpenAI Style)" {
		t.Errorf("Expected mock response, got %s", resp)
	}
}
