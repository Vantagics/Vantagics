package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// OpenAICompatibleWrapper wraps an OpenAI-compatible chat model to handle
// non-standard error responses (e.g., Gemini's array-format errors)
type OpenAICompatibleWrapper struct {
	inner   model.ChatModel
	baseURL string
	logger  func(string)
}

// NewOpenAICompatibleWrapper creates a wrapper around an OpenAI-compatible model
func NewOpenAICompatibleWrapper(inner model.ChatModel, baseURL string, logger func(string)) *OpenAICompatibleWrapper {
	return &OpenAICompatibleWrapper{
		inner:   inner,
		baseURL: baseURL,
		logger:  logger,
	}
}

// isGeminiEndpoint checks if the base URL points to Gemini's OpenAI-compatible endpoint
func (w *OpenAICompatibleWrapper) isGeminiEndpoint() bool {
	return strings.Contains(w.baseURL, "generativelanguage.googleapis.com")
}

// Generate wraps the inner model's Generate method with error handling
func (w *OpenAICompatibleWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	resp, err := w.inner.Generate(ctx, input, opts...)
	if err != nil {
		// Try to parse and improve error messages
		improvedErr := w.improveErrorMessage(err)
		return nil, improvedErr
	}
	return resp, nil
}

// Stream wraps the inner model's Stream method with error handling
func (w *OpenAICompatibleWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	reader, err := w.inner.Stream(ctx, input, opts...)
	if err != nil {
		improvedErr := w.improveErrorMessage(err)
		return nil, improvedErr
	}
	return reader, nil
}

// BindTools delegates to the inner model
func (w *OpenAICompatibleWrapper) BindTools(tools []*schema.ToolInfo) error {
	if binder, ok := w.inner.(interface{ BindTools([]*schema.ToolInfo) error }); ok {
		return binder.BindTools(tools)
	}
	return nil
}

// improveErrorMessage attempts to parse and improve error messages,
// especially for Gemini's non-standard error format
func (w *OpenAICompatibleWrapper) improveErrorMessage(err error) error {
	errStr := err.Error()

	// Check for JSON unmarshal errors that indicate Gemini's array format
	if strings.Contains(errStr, "cannot unmarshal array") {
		// Try to extract the actual error from the body
		if idx := strings.Index(errStr, "body:"); idx != -1 {
			bodyStr := strings.TrimSpace(errStr[idx+5:])
			
			// Try to parse as Gemini's array error format
			var geminiErrors []struct {
				Error struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Status  string `json:"status"`
				} `json:"error"`
			}
			
			if jsonErr := json.Unmarshal([]byte(bodyStr), &geminiErrors); jsonErr == nil && len(geminiErrors) > 0 {
				ge := geminiErrors[0].Error
				
				// Provide user-friendly messages for common errors
				switch ge.Code {
				case 503:
					return fmt.Errorf("Gemini service temporarily unavailable (503): %s. Please try again in a few moments", ge.Message)
				case 429:
					return fmt.Errorf("Gemini rate limit exceeded (429): %s. Please wait before retrying", ge.Message)
				case 400:
					// Check for thought_signature error
					if strings.Contains(ge.Message, "thought_signature") {
						return fmt.Errorf("Gemini thinking model requires thought signatures for function calls. Consider using the native 'Gemini' provider instead of 'OpenAI-Compatible' for full feature support. Original error: %s", ge.Message)
					}
					return fmt.Errorf("Gemini bad request (400): %s", ge.Message)
				case 401:
					return fmt.Errorf("Gemini authentication failed (401): Please check your API key")
				case 404:
					return fmt.Errorf("Gemini model not found (404): %s. Please verify the model name", ge.Message)
				default:
					return fmt.Errorf("Gemini API error (%d - %s): %s", ge.Code, ge.Status, ge.Message)
				}
			}
		}
		
		// If we couldn't parse, return a generic improved message
		if w.isGeminiEndpoint() {
			return fmt.Errorf("Gemini API returned an error in non-standard format. This may be a temporary service issue. Original error: %v", err)
		}
	}

	// Check for common error patterns and improve messages
	if strings.Contains(errStr, "overloaded") || strings.Contains(errStr, "UNAVAILABLE") {
		return fmt.Errorf("The model is currently overloaded. Please try again in a few moments. Original: %v", err)
	}

	// Return original error if no improvement possible
	return err
}
