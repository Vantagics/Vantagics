package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLLMServiceChat_404Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	tests := []struct {
		name     string
		provider string
	}{
		{"OpenAI-Compatible", "OpenAI-Compatible"},
		{"Claude-Compatible", "Claude-Compatible"},
		{"Anthropic", "Anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				LLMProvider: tt.provider,
				APIKey:      "test",
				BaseURL:     server.URL,
			}
			service := NewLLMService(config)
			_, err := service.Chat(context.Background(), "test")
			if err == nil {
				t.Fatal("Expected error for 404 status")
			}

			if !strings.Contains(err.Error(), "API error (404): Not Found") {
				t.Errorf("Expected specific 404 error message, got: %v", err)
			}

			if !strings.Contains(err.Error(), server.URL) {
				t.Errorf("Expected error message to contain the URL, got: %v", err)
			}
		})
	}
}

func TestLLMServiceChat_400Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("model does not exist"))
	}))
	defer server.Close()

	config := Config{
		LLMProvider: "OpenAI-Compatible",
		APIKey:      "test",
		BaseURL:     server.URL,
		ModelName:   "non-existent-model",
	}
	service := NewLLMService(config)
	_, err := service.Chat(context.Background(), "test")
	if err == nil {
		t.Fatal("Expected error for 400 status")
	}

	if !strings.Contains(err.Error(), "API error (400): Bad Request") {
		t.Errorf("Expected specific 400 error message, got: %v", err)
	}

	if !strings.Contains(err.Error(), "non-existent-model") {
		t.Errorf("Expected error message to contain the model name, got: %v", err)
	}
}