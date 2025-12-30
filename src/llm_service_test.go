package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLLMServiceFactory(t *testing.T) {
	// Test OpenAI config
	openAIConfig := Config{
		LLMProvider: "OpenAI",
		APIKey:      "sk-test",
		ModelName:   "gpt-4",
	}
	
	service := NewLLMService(openAIConfig)
	if service.Provider != "OpenAI" {
		t.Errorf("Expected provider OpenAI, got %s", service.Provider)
	}

	// Test Anthropic config
	anthropicConfig := Config{
		LLMProvider: "Anthropic",
		APIKey:      "sk-ant-test",
		ModelName:   "claude-3",
	}
	
	service = NewLLMService(anthropicConfig)
	if service.Provider != "Anthropic" {
		t.Errorf("Expected provider Anthropic, got %s", service.Provider)
	}
}

func TestLLMServiceChat_OpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Hello from mock OpenAI",
					},
				},
			},
		})
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "OpenAI",
		APIKey:      "sk-test",
		ModelName:   "gpt-4",
		BaseURL:     server.URL,
	}
	service := NewLLMService(config)
	
	resp, err := service.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	
	if resp != "Hello from mock OpenAI" {
		t.Errorf("Expected mock response, got %s", resp)
	}
}

func TestLLMServiceChat_Anthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-ant-test" {
			t.Errorf("Expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"text": "Hello from mock Anthropic",
				},
			},
		})
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "Anthropic",
		APIKey:      "sk-ant-test",
		ModelName:   "claude-3",
		BaseURL:     server.URL,
	}
	service := NewLLMService(config)
	
	resp, err := service.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	
	if resp != "Hello from mock Anthropic" {
		t.Errorf("Expected mock response, got %s", resp)
	}
}

func TestLLMServiceChat_OpenAICompatible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Hello from mock Compatible",
					},
				},
			},
		})
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "OpenAI-Compatible",
		APIKey:      "optional-key",
		ModelName:   "local-model",
		BaseURL:     server.URL,
	}
	service := NewLLMService(config)
	
	resp, err := service.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	
	if resp != "Hello from mock Compatible" {
		t.Errorf("Expected mock response, got %s", resp)
	}
}

func TestLLMServiceChat_OpenAICompatible_BaseOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected path /v1/chat/completions, got %s", r.URL.Path)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Hello from base URL",
					},
				},
			},
		})
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "OpenAI-Compatible",
		BaseURL:     server.URL, // No trailing path
	}
	service := NewLLMService(config)
	
	resp, err := service.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	
	if resp != "Hello from base URL" {
		t.Errorf("Expected mock response, got %s", resp)
	}
}

func TestLLMServiceChat_OpenAIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "OpenAI",
		APIKey:      "test-key",
		BaseURL:     server.URL,
	}
	service := NewLLMService(config)
	
	_, err := service.Chat(context.Background(), "Hello")
	if err == nil {
		t.Error("Expected error for non-200 status")
	}
}

func TestLLMServiceChat_AnthropicError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "Anthropic",
		APIKey:      "test-key",
		BaseURL:     server.URL,
	}
	service := NewLLMService(config)
	
	_, err := service.Chat(context.Background(), "Hello")
	if err == nil {
		t.Error("Expected error for non-200 status")
	}
}

func TestLLMServiceChat_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json}"))
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "OpenAI",
		APIKey:      "test-key",
		BaseURL:     server.URL,
	}
	service := NewLLMService(config)
	
	_, err := service.Chat(context.Background(), "Hello")
	if err == nil {
		t.Error("Expected error for malformed JSON")
	}
}