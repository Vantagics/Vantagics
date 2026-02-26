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
	MaxAnalysisSteps  int            `json:"maxAnalysisSteps"`      // Maximum agent graph steps per analysis (10-50, default 25)
	DetailedLog       bool           `json:"detailedLog"`
	SoundNotification bool           `json:"soundNotification"` // 分析完成声音提示（默认开启）
	LogMaxSizeMB      int            `json:"logMaxSizeMB"`          // Maximum log file size in MB before compression (default 100)
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
	// 用于位置相关查询（如天气、附近地点等�
	Location *LocationConfig `json:"location,omitempty"`
	
	// Shopify OAuth configuration (for developers)
	// These are set by the developer who registered the app with Shopify
	ShopifyClientID     string `json:"shopifyClientId,omitempty"`
	ShopifyClientSecret string `json:"shopifyClientSecret,omitempty"`
	
	// License configuration (for commercial mode)
	LicenseSN        string `json:"licenseSN,omitempty"`        // Saved license serial number
	LicenseServerURL string `json:"licenseServerURL,omitempty"` // License server URL
	LicenseEmail     string `json:"licenseEmail,omitempty"`     // Bound email address for license activation
	
	// Extra license info from server (product-specific key-value pairs)
	LicenseExtraInfo map[string]interface{} `json:"licenseExtraInfo,omitempty"`
	
	// Layout persistence
	SidebarWidth    int     `json:"sidebarWidth,omitempty"`    // Sidebar panel width in pixels
	PanelRightRatio float64 `json:"panelRightRatio,omitempty"` // Right panel width as ratio of available space (0-1)
	PanelRightWidth int     `json:"panelRightWidth,omitempty"` // DEPRECATED: kept for migration only
	
	// Other settings
	AuthorSignature string `json:"authorSignature,omitempty"` // Default author signature for quick analysis pack export
}

// Validate checks and corrects Config field values, applying safe defaults where needed.
// Call this after loading config from file/JSON to ensure all values are within valid ranges.
func (c *Config) Validate() {
	// MaxTokens: must be positive, default 4096
	if c.MaxTokens <= 0 {
		c.MaxTokens = 4096
	}

	// MaxPreviewRows: must be positive, default 100
	if c.MaxPreviewRows <= 0 {
		c.MaxPreviewRows = 100
	} else if c.MaxPreviewRows > 10000 {
		c.MaxPreviewRows = 10000
	}

	// MaxConcurrentAnalysis: 1-10, default 5
	if c.MaxConcurrentAnalysis < 1 {
		c.MaxConcurrentAnalysis = 5
	} else if c.MaxConcurrentAnalysis > 10 {
		c.MaxConcurrentAnalysis = 10
	}

	// MaxAnalysisSteps: 10-50, default 25
	if c.MaxAnalysisSteps < 10 {
		c.MaxAnalysisSteps = 25
	} else if c.MaxAnalysisSteps > 50 {
		c.MaxAnalysisSteps = 50
	}

	// LogMaxSizeMB: at least 1, default 100
	if c.LogMaxSizeMB < 1 {
		c.LogMaxSizeMB = 100
	}

	// Language: default to "zh-CN" if empty
	if c.Language == "" {
		c.Language = "zh-CN"
	}

	// LLMProvider: default to "OpenAI" if empty
	if c.LLMProvider == "" {
		c.LLMProvider = "OpenAI"
	}

	// ProxyConfig port validation
	if c.ProxyConfig != nil && c.ProxyConfig.Enabled {
		if c.ProxyConfig.Port <= 0 || c.ProxyConfig.Port > 65535 {
			c.ProxyConfig.Enabled = false
		}
		if c.ProxyConfig.Host == "" {
			c.ProxyConfig.Enabled = false
		}
		// Validate proxy protocol
		switch c.ProxyConfig.Protocol {
		case "http", "https", "socks5":
			// valid
		default:
			c.ProxyConfig.Protocol = "http"
		}
	}

	// IntentEnhancement: validate sub-config if present
	if c.IntentEnhancement != nil {
		c.IntentEnhancement.Validate()
	}

	// PanelRightRatio: 0-1 range
	if c.PanelRightRatio < 0 {
		c.PanelRightRatio = 0
	} else if c.PanelRightRatio > 1 {
		c.PanelRightRatio = 1
	}
}
