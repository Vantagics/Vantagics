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