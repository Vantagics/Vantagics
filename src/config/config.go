package config

// MCPService represents a single MCP service configuration
type MCPService struct {
	ID          string `json:"id"`          // Unique identifier
	Name        string `json:"name"`        // Display name
	Description string `json:"description"` // Service description
	URL         string `json:"url"`         // MCP service URL
	Enabled     bool   `json:"enabled"`     // Whether the service is enabled
	Tested      bool   `json:"tested"`      // Whether the service has been tested successfully
}

// SearchEngine represents a search engine configuration (DEPRECATED - use SearchAPIConfig)
type SearchEngine struct {
	ID      string `json:"id"`      // Unique identifier
	Name    string `json:"name"`    // Display name (e.g., "Google", "Bing")
	URL     string `json:"url"`     // Base URL (e.g., "www.google.com")
	Enabled bool   `json:"enabled"` // Whether this engine is enabled
	Tested  bool   `json:"tested"`  // Whether the engine has been tested successfully
}

// SearchAPIConfig represents a search API service configuration
type SearchAPIConfig struct {
	ID          string `json:"id"`          // Unique identifier: "duckduckgo", "google_custom", "uapi_pro"
	Name        string `json:"name"`        // Display name
	Description string `json:"description"` // Service description
	APIKey      string `json:"apiKey,omitempty"`      // API key (required for Google Custom Search and UAPI Pro)
	CustomID    string `json:"customId,omitempty"`    // Custom Search Engine ID (for Google)
	Enabled     bool   `json:"enabled"`     // Whether this service is enabled
	Tested      bool   `json:"tested"`      // Whether the service has been tested successfully
}

// ProxyConfig represents proxy server configuration
type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`  // Whether proxy is enabled
	Protocol string `json:"protocol"` // Proxy protocol: "http", "https", "socks5"
	Host     string `json:"host"`     // Proxy server IP or hostname
	Port     int    `json:"port"`     // Proxy server port
	Username string `json:"username,omitempty"` // Optional username for authentication
	Password string `json:"password,omitempty"` // Optional password for authentication
	Tested   bool   `json:"tested"`   // Whether the proxy has been tested successfully
}

// LocationConfig represents user location configuration
type LocationConfig struct {
	Country   string  `json:"country"`   // Country name (e.g., "China", "United States")
	City      string  `json:"city"`      // City name (e.g., "Beijing", "New York")
	Latitude  float64 `json:"latitude,omitempty"`  // Optional latitude
	Longitude float64 `json:"longitude,omitempty"` // Optional longitude
}

// UAPIConfig represents UAPI service configuration
type UAPIConfig struct {
	Enabled  bool   `json:"enabled"`  // Whether UAPI is enabled
	APIToken string `json:"apiToken"` // UAPI API token
	BaseURL  string `json:"baseUrl,omitempty"` // Optional custom base URL
	Tested   bool   `json:"tested"`   // Whether UAPI has been tested successfully
}

// Config structure
type Config struct {
	LLMProvider            string       `json:"llmProvider"`
	APIKey                 string       `json:"apiKey"`
	BaseURL                string       `json:"baseUrl"`
	ModelName              string       `json:"modelName"`
	MaxTokens              int          `json:"maxTokens"`
	DarkMode               bool         `json:"darkMode"`
	EnableMemory           bool         `json:"enableMemory"`           // 启用记忆功能
	AutoAnalysisSuggestions bool        `json:"autoAnalysisSuggestions"` // 自动分析建议
	LocalCache             bool         `json:"localCache"`
	Language          string       `json:"language"`
	ClaudeHeaderStyle string       `json:"claudeHeaderStyle"`
	DataCacheDir      string       `json:"dataCacheDir"`
	PythonPath        string         `json:"pythonPath"`
	MaxPreviewRows    int            `json:"maxPreviewRows"`
	MaxConcurrentAnalysis int        `json:"maxConcurrentAnalysis"` // Maximum concurrent analysis tasks (1-10, default 5)
	DetailedLog       bool           `json:"detailedLog"`
	AutoIntentUnderstanding bool     `json:"autoIntentUnderstanding"` // Enable automatic intent understanding before analysis
	MCPServices       []MCPService   `json:"mcpServices"`     // Generic MCP services configuration
	SearchEngines     []SearchEngine `json:"searchEngines,omitempty"`   // DEPRECATED: Legacy search engines
	SearchAPIs        []SearchAPIConfig `json:"searchAPIs"`   // Search API services configuration
	ActiveSearchEngine string        `json:"activeSearchEngine,omitempty"` // DEPRECATED: ID of active search engine
	ActiveSearchAPI   string        `json:"activeSearchAPI,omitempty"` // ID of active search API
	ProxyConfig       *ProxyConfig   `json:"proxyConfig,omitempty"` // Proxy server configuration
	UAPIConfig        *UAPIConfig    `json:"uapiConfig,omitempty"` // DEPRECATED: Use SearchAPIs instead
	
	// Deprecated: Legacy web search fields (kept for backward compatibility)
	WebSearchProvider string `json:"webSearchProvider,omitempty"`
	WebSearchAPIKey   string `json:"webSearchAPIKey,omitempty"`
	WebSearchMCPURL   string `json:"webSearchMCPURL,omitempty"`
	
	// IntentEnhancement 意图增强配置
	// 用于控制意图理解增强功能的各项开关和参数
	IntentEnhancement *IntentEnhancementConfig `json:"intentEnhancement,omitempty"`
	
	// Location 用户位置配置
	// 用于位置相关查询（如天气、附近地点等）
	Location *LocationConfig `json:"location,omitempty"`
}
