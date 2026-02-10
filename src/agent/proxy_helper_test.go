package agent

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"vantagedata/config"
)

// TestProxyHTTPClient_WithRealProxy tests that the proxy HTTP client works with a real proxy.
// Run with: go test -v -run TestProxyHTTPClient_WithRealProxy ./agent/
// Requires a running proxy at 127.0.0.1:10808
func TestProxyHTTPClient_WithRealProxy(t *testing.T) {
	proxyCfg := &config.ProxyConfig{
		Enabled:  true,
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     10808,
		Tested:   true,
	}

	client := NewProxyHTTPClient(15*time.Second, proxyCfg)

	// Test 1: Check that transport has proxy configured
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport, got different type or nil")
	}
	if transport.Proxy == nil {
		t.Fatal("Proxy function is nil — proxy was not configured")
	}
	t.Log("✓ Transport has proxy function configured")

	// Test 2: Try to reach an external URL through the proxy
	testURLs := []string{
		"https://httpbin.org/ip",
		"https://api.openai.com",       // OpenAI API endpoint
		"https://api.anthropic.com",     // Anthropic API endpoint
	}

	for _, testURL := range testURLs {
		t.Run(testURL, func(t *testing.T) {
			req, err := http.NewRequest("HEAD", testURL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Logf("✗ Failed to reach %s through proxy: %v", testURL, err)
				t.Logf("  This may mean the proxy at %s:%d is not running", proxyCfg.Host, proxyCfg.Port)
				return
			}
			defer resp.Body.Close()
			t.Logf("✓ Reached %s through proxy — HTTP %d", testURL, resp.StatusCode)
		})
	}

	// Test 3: Verify that without proxy, the same client would be different
	noproxyCfg := &config.ProxyConfig{
		Enabled: false,
	}
	clientNoProxy := NewProxyHTTPClient(15*time.Second, noproxyCfg)
	if clientNoProxy.Transport != nil {
		// Default client has nil transport (uses DefaultTransport)
		t.Log("⚠ No-proxy client has custom transport (unexpected but not fatal)")
	} else {
		t.Log("✓ No-proxy client uses default transport (no proxy)")
	}

	// Test 4: Verify Tested=false prevents proxy usage
	untestedCfg := &config.ProxyConfig{
		Enabled:  true,
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     10808,
		Tested:   false, // Not tested!
	}
	clientUntested := NewProxyHTTPClient(15*time.Second, untestedCfg)
	if clientUntested.Transport != nil {
		t.Fatal("✗ Untested proxy config should NOT configure proxy transport")
	}
	t.Log("✓ Untested proxy config correctly skips proxy configuration")
}

// TestProxyHTTPClient_NilConfig tests that nil proxy config doesn't panic
func TestProxyHTTPClient_NilConfig(t *testing.T) {
	client := NewProxyHTTPClient(10*time.Second, nil)
	if client == nil {
		t.Fatal("Client should not be nil")
	}
	if client.Transport != nil {
		t.Fatal("Nil proxy config should not set custom transport")
	}
	t.Log("✓ Nil proxy config returns plain client")
}

// TestProxyHTTPClient_AllProviders tests that all LLM provider HTTP clients use proxy correctly
func TestProxyHTTPClient_AllProviders(t *testing.T) {
	proxyCfg := &config.ProxyConfig{
		Enabled:  true,
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     10808,
		Tested:   true,
	}

	cfg := config.Config{
		LLMProvider: "OpenAI",
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com",
		ModelName:   "gpt-4o",
		MaxTokens:   4096,
		ProxyConfig: proxyCfg,
	}

	// Test LLMService
	llm := NewLLMService(cfg, func(s string) { fmt.Println(s) })
	if llm.httpClient == nil {
		t.Fatal("LLMService httpClient is nil")
	}
	transport, ok := llm.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatal("LLMService httpClient does not have proxy configured")
	}
	t.Log("✓ LLMService uses proxy")

	// Test AnthropicChatModel
	anthropic, err := NewAnthropicChatModel(nil, &AnthropicConfig{
		APIKey:      "test-key",
		Model:       "claude-3-5-sonnet-20241022",
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		t.Fatalf("Failed to create AnthropicChatModel: %v", err)
	}
	transport, ok = anthropic.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatal("AnthropicChatModel httpClient does not have proxy configured")
	}
	t.Log("✓ AnthropicChatModel uses proxy")

	// Test GeminiChatModel
	gemini, err := NewGeminiChatModel(nil, &GeminiConfig{
		APIKey:      "test-key",
		Model:       "gemini-2.0-flash",
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		t.Fatalf("Failed to create GeminiChatModel: %v", err)
	}
	transport, ok = gemini.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatal("GeminiChatModel httpClient does not have proxy configured")
	}
	t.Log("✓ GeminiChatModel uses proxy")

	// Test WebFetchTool
	webFetch := NewWebFetchTool(func(s string) { fmt.Println(s) }, proxyCfg)
	if webFetch.httpClient == nil {
		t.Fatal("WebFetchTool httpClient is nil")
	}
	transport, ok = webFetch.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatal("WebFetchTool httpClient does not have proxy configured")
	}
	t.Log("✓ WebFetchTool uses proxy")
}
