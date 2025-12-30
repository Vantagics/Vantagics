package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LLMService struct {
	Provider  string
	APIKey    string
	BaseURL   string
	ModelName string
	MaxTokens int
}

func NewLLMService(config Config) *LLMService {
	return &LLMService{
		Provider:  config.LLMProvider,
		APIKey:    config.APIKey,
		BaseURL:   config.BaseURL,
		ModelName: config.ModelName,
		MaxTokens: config.MaxTokens,
	}
}

func (s *LLMService) Chat(ctx context.Context, message string) (string, error) {
	if s.APIKey == "" && s.Provider != "OpenAI-Compatible" {
		return "Please set your API key in settings.", nil
	}

	switch s.Provider {
	case "OpenAI", "OpenAI-Compatible":
		return s.chatOpenAI(ctx, message)
	case "Anthropic":
		return s.chatAnthropic(ctx, message)
	default:
		return "Unsupported LLM provider.", nil
	}
}

func (s *LLMService) chatOpenAI(ctx context.Context, message string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	if s.BaseURL != "" {
		url = s.BaseURL
		// If it's just a base URL like http://localhost:11434, append the OpenAI path
		// We can't be sure, but a common pattern is that users provide the base
		if !contains(url, "/v1/chat/completions") && !contains(url, "/chat/completions") {
			if url[len(url)-1] == '/' {
				url += "v1/chat/completions"
			} else {
				url += "/v1/chat/completions"
			}
		}
	}

	body := map[string]interface{}{
		"model": s.ModelName,
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI-compatible API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from OpenAI-compatible API")
}

func (s *LLMService) chatAnthropic(ctx context.Context, message string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"
	if s.BaseURL != "" {
		url = s.BaseURL
		if !contains(url, "/v1/messages") && !contains(url, "/messages") {
			if url[len(url)-1] == '/' {
				url += "v1/messages"
			} else {
				url += "/v1/messages"
			}
		}
	}

	body := map[string]interface{}{
		"model": s.ModelName,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}

	return "", fmt.Errorf("no response from Anthropic")
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

