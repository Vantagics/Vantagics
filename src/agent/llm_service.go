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
		return "", fmt.Errorf("API key not configured. Please set your API key in settings")
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

// StreamCallback is called for each chunk of streamed content
type LLMStreamCallback func(content string)

// ChatStream sends a message and streams the response
func (s *LLMService) ChatStream(ctx context.Context, message string, onChunk LLMStreamCallback) (string, error) {
	s.log(fmt.Sprintf("ChatStream Request [%s]: %s", s.Provider, message))

	if s.APIKey == "" && s.Provider != "OpenAI-Compatible" && s.Provider != "Claude-Compatible" {
		return "", fmt.Errorf("API key not configured. Please set your API key in settings")
	}

	var resp string
	var err error

	switch s.Provider {
	case "OpenAI", "OpenAI-Compatible":
		resp, err = s.chatOpenAIStream(ctx, message, onChunk)
	case "Anthropic":
		resp, err = s.chatAnthropicStream(ctx, message, onChunk)
	case "Claude-Compatible":
		resp, err = s.chatClaudeCompatibleStream(ctx, message, onChunk)
	default:
		return "Unsupported LLM provider.", nil
	}

	if err != nil {
		s.log(fmt.Sprintf("ChatStream Error: %v", err))
	} else {
		s.log(fmt.Sprintf("ChatStream Response length: %d", len(resp)))
	}

	return resp, err
}

func (s *LLMService) chatOpenAIStream(ctx context.Context, message string, onChunk LLMStreamCallback) (string, error) {
	fullURL := "https://api.openai.com/v1/chat/completions"
	if s.BaseURL != "" {
		u, err := url.Parse(s.BaseURL)
		if err != nil {
			return "", fmt.Errorf("invalid base URL: %v", err)
		}

		path := u.Path
		trimmedPath := strings.TrimSuffix(path, "/")
		
		if !strings.HasSuffix(trimmedPath, "/chat/completions") {
			if !strings.HasSuffix(path, "/") {
				path += "/"
			}

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
		"model":      s.ModelName,
		"max_tokens": getProviderMaxTokens(s.ModelName, s.MaxTokens),
		"stream":     true,
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
		return "", fmt.Errorf("OpenAI-compatible API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	reader := resp.Body

	// Read SSE stream
	buf := make([]byte, 4096)
	var lineBuf strings.Builder

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			lineBuf.Write(buf[:n])
			
			// Process complete lines
			for {
				content := lineBuf.String()
				idx := strings.Index(content, "\n")
				if idx == -1 {
					break
				}
				
				line := content[:idx]
				lineBuf.Reset()
				lineBuf.WriteString(content[idx+1:])
				
				line = strings.TrimSpace(line)
				if line == "" || line == "data: [DONE]" {
					continue
				}
				
				if strings.HasPrefix(line, "data: ") {
					jsonData := strings.TrimPrefix(line, "data: ")
					var chunk struct {
						Choices []struct {
							Delta struct {
								Content string `json:"content"`
							} `json:"delta"`
						} `json:"choices"`
					}
					
					if err := json.Unmarshal([]byte(jsonData), &chunk); err == nil {
						if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
							content := chunk.Choices[0].Delta.Content
							fullContent.WriteString(content)
							if onChunk != nil {
								onChunk(content)
							}
						}
					}
				}
			}
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return fullContent.String(), err
		}
	}

	return fullContent.String(), nil
}

func (s *LLMService) chatAnthropicStream(ctx context.Context, message string, onChunk LLMStreamCallback) (string, error) {
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
		"model":      s.ModelName,
		"max_tokens": getProviderMaxTokens(s.ModelName, s.MaxTokens),
		"stream":     true,
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
		return "", fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	reader := resp.Body

	buf := make([]byte, 4096)
	var lineBuf strings.Builder

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			lineBuf.Write(buf[:n])
			
			for {
				content := lineBuf.String()
				idx := strings.Index(content, "\n")
				if idx == -1 {
					break
				}
				
				line := content[:idx]
				lineBuf.Reset()
				lineBuf.WriteString(content[idx+1:])
				
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				
				if strings.HasPrefix(line, "data: ") {
					jsonData := strings.TrimPrefix(line, "data: ")
					var chunk struct {
						Type  string `json:"type"`
						Delta struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"delta"`
					}
					
					if err := json.Unmarshal([]byte(jsonData), &chunk); err == nil {
						if chunk.Type == "content_block_delta" && chunk.Delta.Text != "" {
							fullContent.WriteString(chunk.Delta.Text)
							if onChunk != nil {
								onChunk(chunk.Delta.Text)
							}
						}
					}
				}
			}
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return fullContent.String(), err
		}
	}

	return fullContent.String(), nil
}

func (s *LLMService) chatClaudeCompatibleStream(ctx context.Context, message string, onChunk LLMStreamCallback) (string, error) {
	if s.BaseURL == "" {
		return "", fmt.Errorf("Base URL is required for Claude-Compatible provider")
	}

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
	fullURL := u.String()

	body := map[string]interface{}{
		"model":      s.ModelName,
		"max_tokens": getProviderMaxTokens(s.ModelName, s.MaxTokens),
		"stream":     true,
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

	if s.ClaudeHeaderStyle == "OpenAI" {
		if s.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.APIKey)
		}
	} else {
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
		return "", fmt.Errorf("Claude-Compatible API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	reader := resp.Body

	buf := make([]byte, 4096)
	var lineBuf strings.Builder

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			lineBuf.Write(buf[:n])
			
			for {
				content := lineBuf.String()
				idx := strings.Index(content, "\n")
				if idx == -1 {
					break
				}
				
				line := content[:idx]
				lineBuf.Reset()
				lineBuf.WriteString(content[idx+1:])
				
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				
				if strings.HasPrefix(line, "data: ") {
					jsonData := strings.TrimPrefix(line, "data: ")
					
					// Try Anthropic format first
					var anthropicChunk struct {
						Type  string `json:"type"`
						Delta struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"delta"`
					}
					
					if err := json.Unmarshal([]byte(jsonData), &anthropicChunk); err == nil {
						if anthropicChunk.Type == "content_block_delta" && anthropicChunk.Delta.Text != "" {
							fullContent.WriteString(anthropicChunk.Delta.Text)
							if onChunk != nil {
								onChunk(anthropicChunk.Delta.Text)
							}
							continue
						}
					}
					
					// Try OpenAI format
					var openaiChunk struct {
						Choices []struct {
							Delta struct {
								Content string `json:"content"`
							} `json:"delta"`
						} `json:"choices"`
					}
					
					if err := json.Unmarshal([]byte(jsonData), &openaiChunk); err == nil {
						if len(openaiChunk.Choices) > 0 && openaiChunk.Choices[0].Delta.Content != "" {
							content := openaiChunk.Choices[0].Delta.Content
							fullContent.WriteString(content)
							if onChunk != nil {
								onChunk(content)
							}
						}
					}
				}
			}
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return fullContent.String(), err
		}
	}

	return fullContent.String(), nil
}

