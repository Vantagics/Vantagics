package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"rapidbi/config"
)

type LLMService struct {
	Provider          string
	APIKey            string
	BaseURL           string
	ModelName         string
	MaxTokens         int
	ClaudeHeaderStyle string
	Log               func(string)
}

func NewLLMService(cfg config.Config, logFunc func(string)) *LLMService {
	return &LLMService{
		Provider:          cfg.LLMProvider,
		APIKey:            cfg.APIKey,
		BaseURL:           cfg.BaseURL,
		ModelName:         cfg.ModelName,
		MaxTokens:         cfg.MaxTokens,
		ClaudeHeaderStyle: cfg.ClaudeHeaderStyle,
		Log:               logFunc,
	}
}

func (s *LLMService) log(msg string) {
	if s.Log != nil {
		s.Log(msg)
	}
}

func (s *LLMService) Chat(ctx context.Context, message string) (string, error) {
	s.log(fmt.Sprintf("Chat Request [%s]: %s", s.Provider, message))

	if s.APIKey == "" && s.Provider != "OpenAI-Compatible" && s.Provider != "Claude-Compatible" {
		return "Please set your API key in settings.", nil
	}

	var resp string
	var err error

	switch s.Provider {
	case "OpenAI", "OpenAI-Compatible":
		resp, err = s.chatOpenAI(ctx, message)
	case "Anthropic":
		resp, err = s.chatAnthropic(ctx, message)
	case "Claude-Compatible":
		resp, err = s.chatClaudeCompatible(ctx, message)
	default:
		return "Unsupported LLM provider.", nil
	}

	if err != nil {
		s.log(fmt.Sprintf("Chat Error: %v", err))
	} else {
		s.log(fmt.Sprintf("Chat Response: %s", resp))
	}

	return resp, err
}

func (s *LLMService) chatOpenAI(ctx context.Context, message string) (string, error) {
	fullURL := "https://api.openai.com/v1/chat/completions"
	if s.BaseURL != "" {
		u, err := url.Parse(s.BaseURL)
		if err != nil {
			return "", fmt.Errorf("invalid base URL: %v", err)
		}

		// Smart path handling
		path := u.Path
		// Normalize: remove trailing slash for check
		trimmedPath := strings.TrimSuffix(path, "/")
		
		if !strings.HasSuffix(trimmedPath, "/chat/completions") {
			if !strings.HasSuffix(path, "/") {
				path += "/"
			}

			// Check if version is already present in path (e.g., /v1/, /v4/)
			hasVersion := false
			parts := strings.Split(path, "/")
			for _, p := range parts {
				if strings.HasPrefix(p, "v") && len(p) > 1 && p[1] >= '0' && p[1] <= '9' {
					hasVersion = true
					break
				}
			}

			if hasVersion {
				path += "chat/completions"
			} else {
				path += "v1/chat/completions"
			}
		}
		u.Path = path
		fullURL = u.String()
	}

	body := map[string]interface{}{
		"model": s.ModelName,
		"max_tokens": getProviderMaxTokens(s.ModelName, s.MaxTokens),
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
	}

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("API error (404): Not Found. Please check your Base URL and path (e.g., /v1/chat/completions). Full URL used: %s", fullURL)
		}
		if resp.StatusCode == http.StatusBadRequest {
			return "", fmt.Errorf("API error (400): Bad Request. This often means the model name '%s' is incorrect or doesn't exist on the provider. Original error: %s", s.ModelName, string(respBody))
		}
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
	fullURL := "https://api.anthropic.com/v1/messages"
	if s.BaseURL != "" {
		u, err := url.Parse(s.BaseURL)
		if err != nil {
			return "", fmt.Errorf("invalid base URL: %v", err)
		}

		path := u.Path
		if path == "" || path == "/" || path == "/v1" || path == "/v1/" {
			if !strings.HasSuffix(path, "/") {
				path += "/"
			}
			if !strings.HasPrefix(strings.TrimPrefix(path, "/"), "v1") {
				path += "v1/"
			}
			path += "messages"
		}
		u.Path = path
		fullURL = u.String()
	}

	body := map[string]interface{}{
		"model": s.ModelName,
		"max_tokens": getProviderMaxTokens(s.ModelName, s.MaxTokens),
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("API error (404): Not Found. Please check your Base URL and path (e.g., /v1/messages). Full URL used: %s", fullURL)
		}
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

func (s *LLMService) chatClaudeCompatible(ctx context.Context, message string) (string, error) {
	// For Claude Compatible, we assume the user provides the full URL or we append standard paths
	// similar to OpenAI/Anthropic logic but respecting the BaseURL more strictly if provided.
	if s.BaseURL == "" {
		return "", fmt.Errorf("Base URL is required for Claude-Compatible provider")
	}

	u, err := url.Parse(s.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %v", err)
	}

	// Smart path appending: only append if no messages-related path is present
	path := u.Path
	if path == "" || path == "/" || path == "/v1" || path == "/v1/" {
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		if !strings.HasPrefix(strings.TrimPrefix(path, "/"), "v1") {
			path += "v1/"
		}
		path += "messages"
	}
	u.Path = path
	fullURL := u.String()

	body := map[string]interface{}{
		"model":      s.ModelName,
		"max_tokens": getProviderMaxTokens(s.ModelName, s.MaxTokens),
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	// Apply headers based on style preference
	if s.ClaudeHeaderStyle == "OpenAI" {
		if s.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.APIKey)
		}
	} else {
		// Default to Anthropic style if not specified or explicitly "Anthropic"
		if s.APIKey != "" {
			req.Header.Set("x-api-key", s.APIKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("API error (404): Not Found. Please check your Base URL and path (e.g., /v1/messages). Full URL used: %s", fullURL)
		}
		return "", fmt.Errorf("Claude-Compatible API error (%d): %s", resp.StatusCode, string(respBody))
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

	return "", fmt.Errorf("no response from Claude-Compatible API")
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

