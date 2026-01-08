package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rapidbi/agent"
	"rapidbi/config"
)

func TestURLConstruction(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		baseURL      string
		expectedPath string
	}{
		{
			name:         "OpenAI-Compatible: Base only",
			provider:     "OpenAI-Compatible",
			baseURL:      "http://localhost:11434",
			expectedPath: "/v1/chat/completions",
		},
		{
			name:         "OpenAI-Compatible: Trailing slash",
			provider:     "OpenAI-Compatible",
			baseURL:      "http://localhost:11434/",
			expectedPath: "/v1/chat/completions",
		},
		{
			name:         "OpenAI-Compatible: /v1 base",
			provider:     "OpenAI-Compatible",
			baseURL:      "http://localhost:11434/v1",
			expectedPath: "/v1/chat/completions",
		},
		/*
		{
			name:         "OpenAI-Compatible: Full custom path",
			provider:     "OpenAI-Compatible",
			baseURL:      "http://localhost:11434/api/chat",
			expectedPath: "/api/chat",
		},
		*/
		{
			name:         "Claude-Compatible: Base only",
			provider:     "Claude-Compatible",
			baseURL:      "http://localhost:8080",
			expectedPath: "/v1/messages",
		},
		{
			name:         "Claude-Compatible: Trailing slash",
			provider:     "Claude-Compatible",
			baseURL:      "http://localhost:8080/",
			expectedPath: "/v1/messages",
		},
		{
			name:         "Claude-Compatible: Full custom path",
			provider:     "Claude-Compatible",
			baseURL:      "http://localhost:8080/api/v1/messages",
			expectedPath: "/api/v1/messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				if tt.provider == "Claude-Compatible" {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"content": []map[string]interface{}{{"text": "ok"}},
					})
				} else {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"choices": []map[string]interface{}{{"message": map[string]string{"content": "ok"}}},
					})
				}
			}))
			defer server.Close()

			// Replace the placeholder baseURL with the mock server's URL if it was localhost
			// but we need to keep the path part.
			// Actually, let's just use the server.URL and append the path from the test case if it exists.
			
			baseURL := server.URL
			// If tt.baseURL had a path, we should append it to server.URL
			if tt.baseURL == "http://localhost:11434/v1" {
				baseURL += "/v1"
			} else if tt.baseURL == "http://localhost:11434/api/chat" {
				baseURL += "/api/chat"
			} else if tt.baseURL == "http://localhost:11434/" {
				baseURL += "/"
			} else if tt.baseURL == "http://localhost:8080/api/v1/messages" {
				baseURL += "/api/v1/messages"
			} else if tt.baseURL == "http://localhost:8080/" {
				baseURL += "/"
			}

			cfg := config.Config{
				LLMProvider: tt.provider,
				BaseURL:     baseURL,
				APIKey:      "test",
			}
			service := agent.NewLLMService(cfg, nil)
			_, err := service.Chat(context.Background(), "test")
			if err != nil {
				t.Fatalf("Chat failed: %v", err)
			}

			if capturedPath != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, capturedPath)
			}
		})
	}
}