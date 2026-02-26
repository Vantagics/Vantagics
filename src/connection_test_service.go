package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vantagics/agent"
	"vantagics/config"
)

// ConnectionTester 定义连接测试接口
type ConnectionTester interface {
	TestLLMConnection(cfg config.Config) ConnectionResult
	TestMCPService(url string) ConnectionResult
	TestSearchEngine(url string) ConnectionResult
	TestSearchTools(engineURL string) ConnectionResult
	TestProxy(proxyConfig config.ProxyConfig) ConnectionResult
	TestUAPIConnection(apiToken, baseURL string) ConnectionResult
	TestSearchAPI(apiConfig config.SearchAPIConfig) ConnectionResult
}

// ConnectionTestService 连接测试服务，封装所有连接测试相关的业务逻辑
type ConnectionTestService struct {
	ctx            context.Context
	configProvider ConfigProvider
	logger         func(string)

	// License client for activated license LLM config
	licenseClient *agent.LicenseClient
}

// NewConnectionTestService 创建新的 ConnectionTestService 实例
func NewConnectionTestService(
	configProvider ConfigProvider,
	logger func(string),
) *ConnectionTestService {
	return &ConnectionTestService{
		configProvider: configProvider,
		logger:         logger,
	}
}

// Name 返回服务名称
func (s *ConnectionTestService) Name() string {
	return "connectionTest"
}

// Initialize 初始化连接测试服�
func (s *ConnectionTestService) Initialize(ctx context.Context) error {
	s.ctx = ctx
	s.log("ConnectionTestService initialized")
	return nil
}

// Shutdown 关闭连接测试服务
func (s *ConnectionTestService) Shutdown() error {
	return nil
}

// SetContext 设置 Wails 上下�
func (s *ConnectionTestService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetLicenseClient 设置 LicenseClient（用于延迟注入或重新初始化）
func (s *ConnectionTestService) SetLicenseClient(lc *agent.LicenseClient) {
	s.licenseClient = lc
}

// log 记录日志
func (s *ConnectionTestService) log(msg string) {
	if s.logger != nil {
		s.logger(msg)
	}
}

// --- Connection Test Methods ---

// TestLLMConnection tests the connection to an LLM provider
func (s *ConnectionTestService) TestLLMConnection(cfg config.Config) ConnectionResult {
	// Check if we have activated license with LLM config
	if s.licenseClient != nil && s.licenseClient.IsActivated() {
		activationData := s.licenseClient.GetData()
		if activationData != nil && activationData.LLMAPIKey != "" {
			// Use activated LLM config
			s.log("[TEST-LLM] Using activated license LLM configuration")
			cfg.LLMProvider = activationData.LLMType
			cfg.APIKey = activationData.LLMAPIKey
			cfg.BaseURL = activationData.LLMBaseURL
			cfg.ModelName = activationData.LLMModel
		}
	}

	llm := agent.NewLLMService(cfg, s.log)
	_, err := llm.Chat(s.ctx, "hi LLM, I'm just test the connection. Just answer ok to me without other infor.")
	if err != nil {
		return ConnectionResult{
			Success: false,
			Message: err.Error(),
		}
	}

	return ConnectionResult{
		Success: true,
		Message: "Connection successful!",
	}
}

// TestMCPService tests the connection to an MCP service
func (s *ConnectionTestService) TestMCPService(url string) ConnectionResult {
	if url == "" {
		return ConnectionResult{
			Success: false,
			Message: "MCP service URL is required",
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try to make a simple GET request to check if the service is reachable
	resp, err := client.Get(url)
	if err != nil {
		return ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to connect: %v", err),
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ConnectionResult{
			Success: true,
			Message: fmt.Sprintf("MCP service is reachable (HTTP %d)", resp.StatusCode),
		}
	}

	// Even if status is not 2xx, if we got a response, the service is reachable
	return ConnectionResult{
		Success: true,
		Message: fmt.Sprintf("MCP service responded (HTTP %d)", resp.StatusCode),
	}
}

// TestSearchEngine tests if a search engine is accessible
func (s *ConnectionTestService) TestSearchEngine(urlStr string) ConnectionResult {
	if urlStr == "" {
		return ConnectionResult{
			Success: false,
			Message: "Search engine URL is required",
		}
	}

	// Ensure URL has protocol
	testURL := urlStr
	if !strings.HasPrefix(testURL, "http://") && !strings.HasPrefix(testURL, "https://") {
		testURL = "https://" + testURL
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects
			return nil
		},
	}

	// Try to make a HEAD request first (faster)
	resp, err := client.Head(testURL)
	if err != nil {
		// Try GET if HEAD fails
		resp, err = client.Get(testURL)
		if err != nil {
			return ConnectionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to connect: %v", err),
			}
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return ConnectionResult{
			Success: true,
			Message: fmt.Sprintf("Search engine is accessible (HTTP %d)", resp.StatusCode),
		}
	}

	return ConnectionResult{
		Success: false,
		Message: fmt.Sprintf("Search engine returned HTTP %d", resp.StatusCode),
	}
}

// TestSearchTools tests web_search and web_fetch tools with a sample query
// DEPRECATED: This function used chromedp-based tools which have been removed.
// Search functionality now uses Search API (search_api_tool.go)
// Web fetch functionality now uses HTTP client (web_fetch_tool.go)
func (s *ConnectionTestService) TestSearchTools(engineURL string) ConnectionResult {
	// Get user's language preference
	cfg, _ := s.configProvider.GetConfig()
	lang := cfg.Language
	isChinese := lang == "简体中�"

	msg := "Search tools test is deprecated. Please use Search API configuration instead."
	if isChinese {
		msg = "搜索工具测试已弃用。请改用搜索API配置�"
	}

	return ConnectionResult{
		Success: false,
		Message: msg,
	}
}

// TestProxy tests if a proxy server is accessible
func (s *ConnectionTestService) TestProxy(proxyConfig config.ProxyConfig) ConnectionResult {
	if proxyConfig.Host == "" {
		return ConnectionResult{
			Success: false,
			Message: "Proxy host is required",
		}
	}

	if proxyConfig.Port <= 0 || proxyConfig.Port > 65535 {
		return ConnectionResult{
			Success: false,
			Message: "Invalid proxy port",
		}
	}

	// Determine protocol
	protocol := strings.ToLower(proxyConfig.Protocol)
	if protocol == "" {
		protocol = "http"
	}

	// Test proxy by making a request through it
	// Use a reliable test URL
	testURL := "https://www.google.com"

	// Build proxy URL for http.Transport
	var proxyUser *url.Userinfo
	if proxyConfig.Username != "" {
		if proxyConfig.Password != "" {
			proxyUser = url.UserPassword(proxyConfig.Username, proxyConfig.Password)
		} else {
			proxyUser = url.User(proxyConfig.Username)
		}
	}

	// Create HTTP client with proxy
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: protocol,
				Host:   fmt.Sprintf("%s:%d", proxyConfig.Host, proxyConfig.Port),
				User:   proxyUser,
			}),
		},
	}

	// Try to make a HEAD request
	resp, err := client.Head(testURL)
	if err != nil {
		return ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Proxy connection failed: %v", err),
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return ConnectionResult{
			Success: true,
			Message: fmt.Sprintf("Proxy is working (HTTP %d)", resp.StatusCode),
		}
	}

	return ConnectionResult{
		Success: true,
		Message: fmt.Sprintf("Proxy connected but returned HTTP %d", resp.StatusCode),
	}
}

// TestUAPIConnection tests the connection to UAPI service
func (s *ConnectionTestService) TestUAPIConnection(apiToken, baseURL string) ConnectionResult {
	if apiToken == "" {
		return ConnectionResult{
			Success: false,
			Message: "API Token is required",
		}
	}

	s.log("[UAPI-TEST] Starting UAPI connection test...")

	// Create UAPI search tool
	uapiTool, err := agent.NewUAPISearchTool(s.log, apiToken)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create UAPI tool: %v", err)
		s.log(fmt.Sprintf("[UAPI-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	// Test with a simple search query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testInput := `{"query": "test", "max_results": 1, "source": "general"}`
	result, err := uapiTool.InvokableRun(ctx, testInput)
	if err != nil {
		errMsg := fmt.Sprintf("UAPI search test failed: %v", err)
		s.log(fmt.Sprintf("[UAPI-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	s.log(fmt.Sprintf("[UAPI-TEST] Test successful, result: %s", result))

	return ConnectionResult{
		Success: true,
		Message: "UAPI connection successful",
	}
}

// TestSearchAPI tests a search API configuration
func (s *ConnectionTestService) TestSearchAPI(apiConfig config.SearchAPIConfig) ConnectionResult {
	s.log(fmt.Sprintf("[SEARCH-API-TEST] Testing %s...", apiConfig.Name))

	// Load proxy config for search API testing
	currentCfg, _ := s.configProvider.GetConfig()

	// Create search API tool
	searchTool, err := agent.NewSearchAPITool(s.log, &apiConfig, currentCfg.ProxyConfig)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create search tool: %v", err)
		s.log(fmt.Sprintf("[SEARCH-API-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	// Test with a simple search query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testInput := `{"query": "test search", "max_results": 3}`
	result, err := searchTool.InvokableRun(ctx, testInput)
	if err != nil {
		errMsg := fmt.Sprintf("%s test failed: %v", apiConfig.Name, err)
		s.log(fmt.Sprintf("[SEARCH-API-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	s.log(fmt.Sprintf("[SEARCH-API-TEST] %s test successful, result: %s", apiConfig.Name, result))

	return ConnectionResult{
		Success: true,
		Message: fmt.Sprintf("%s connection successful", apiConfig.Name),
	}
}
