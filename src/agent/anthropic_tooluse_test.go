package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"vantagedata/config"
)

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... (%d chars total)", len(s))
}

// TestAnthropicToolUse verifies that the Anthropic proxy supports tool use.
func TestAnthropicToolUse(t *testing.T) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, "Vantagics", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Get LLM config from license
	if cfg.LicenseSN != "" && cfg.LicenseServerURL != "" {
		lc := NewLicenseClient(func(msg string) { t.Log(msg) })
		result, err := lc.Activate(cfg.LicenseServerURL, cfg.LicenseSN)
		if err != nil {
			t.Fatalf("License activation failed: %v", err)
		}
		if result != nil && result.Success && result.Data != nil && result.Data.LLMAPIKey != "" {
			ad := result.Data
			if ad.LLMType == "anthropic" {
				cfg.LLMProvider = "Anthropic"
			} else {
				cfg.LLMProvider = ad.LLMType
			}
			cfg.APIKey = ad.LLMAPIKey
			cfg.BaseURL = ad.LLMBaseURL
			cfg.ModelName = ad.LLMModel
		}
	}

	if cfg.LLMProvider != "Anthropic" {
		t.Skip("Not Anthropic provider")
	}

	t.Logf("Provider: %s, Model: %s, BaseURL: %s", cfg.LLMProvider, cfg.ModelName, cfg.BaseURL)

	model, err := NewAnthropicChatModel(context.Background(), &AnthropicConfig{
		APIKey:    cfg.APIKey,
		BaseURL:   cfg.BaseURL,
		Model:     cfg.ModelName,
		MaxTokens: 2048,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Sub-test 1: No tools — basic chat
	t.Run("NoTools", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		msgs := []*schema.Message{
			{Role: schema.User, Content: "Say hello in exactly 3 words."},
		}
		resp, err := model.Generate(ctx, msgs)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		t.Logf("Response: %s", truncateStr(resp.Content, 200))
		t.Logf("ToolCalls: %d", len(resp.ToolCalls))
		if resp.Content == "" {
			t.Error("Expected non-empty text response")
		}
	})

	// Sub-test 2: With tools — check if model calls them
	t.Run("WithTools", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Bind a simple tool
		toolInfo := &schema.ToolInfo{
			Name: "get_weather",
			Desc: "Get current weather for a city. You MUST call this tool to answer weather questions.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"city": {
					Type:     schema.String,
					Desc:     "City name",
					Required: true,
				},
			}),
		}
		if err := model.BindTools([]*schema.ToolInfo{toolInfo}); err != nil {
			t.Fatalf("BindTools failed: %v", err)
		}

		msgs := []*schema.Message{
			{Role: schema.System, Content: "You are a helpful assistant. When asked about weather, you MUST use the get_weather tool. Do not answer without calling the tool first."},
			{Role: schema.User, Content: "What's the weather in Tokyo?"},
		}
		resp, err := model.Generate(ctx, msgs)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		t.Logf("Response text: %s", truncateStr(resp.Content, 200))
		t.Logf("ToolCalls count: %d", len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			t.Logf("  ToolCall[%d]: name=%s, id=%s, args=%s", i, tc.Function.Name, tc.ID, tc.Function.Arguments)
		}

		if len(resp.ToolCalls) == 0 {
			t.Error("FAIL: Model did NOT call any tools — proxy may not support tool_use")
		} else {
			t.Log("SUCCESS: Model called tools — proxy supports tool_use")
		}

		// Unbind tools for other tests
		model.BindTools(nil)
	})

	// Sub-test 3: Verify tool schema serialization
	t.Run("ToolSchemaSerialization", func(t *testing.T) {
		toolInfo := &schema.ToolInfo{
			Name: "test_tool",
			Desc: "A test tool",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {
					Type:     schema.String,
					Desc:     "Search query",
					Required: true,
				},
				"limit": {
					Type:     schema.Integer,
					Desc:     "Max results",
					Required: false,
				},
			}),
		}

		// Convert like our fixed code does
		jsonSchema, err := toolInfo.ParamsOneOf.ToJSONSchema()
		if err != nil {
			t.Fatalf("ToJSONSchema failed: %v", err)
		}
		schemaBytes, err := json.MarshalIndent(jsonSchema, "", "  ")
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		t.Logf("Serialized tool schema:\n%s", string(schemaBytes))

		// Verify it has the expected structure
		var schemaMap map[string]interface{}
		json.Unmarshal(schemaBytes, &schemaMap)
		if schemaMap["type"] != "object" {
			t.Errorf("Expected type=object, got %v", schemaMap["type"])
		}
		if props, ok := schemaMap["properties"].(map[string]interface{}); !ok || len(props) == 0 {
			t.Error("Expected non-empty properties")
		} else {
			t.Logf("Properties: %v", props)
		}

		// Also verify the OLD broken way
		brokenBytes, _ := json.Marshal(toolInfo.ParamsOneOf)
		t.Logf("OLD (broken) serialization: %s", string(brokenBytes))
		if string(brokenBytes) == "{}" {
			t.Log("CONFIRMED: Direct json.Marshal of ParamsOneOf produces '{}' — this was the bug!")
		}
	})
}
