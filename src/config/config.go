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

// SearchEngine represents a search engine configuration
type SearchEngine struct {
	ID      string `json:"id"`      // Unique identifier
	Name    string `json:"name"`    // Display name (e.g., "Google", "Bing")
	URL     string `json:"url"`     // Base URL (e.g., "www.google.com")
	Enabled bool   `json:"enabled"` // Whether this engine is enabled
	Tested  bool   `json:"tested"`  // Whether the engine has been tested successfully
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

// Config structure
type Config struct {
	LLMProvider       string       `json:"llmProvider"`
	APIKey            string       `json:"apiKey"`
	BaseURL           string       `json:"baseUrl"`
	ModelName         string       `json:"modelName"`
	MaxTokens         int          `json:"maxTokens"`
	DarkMode          bool         `json:"darkMode"`
	LocalCache        bool         `json:"localCache"`
	Language          string       `json:"language"`
	ClaudeHeaderStyle string       `json:"claudeHeaderStyle"`
	DataCacheDir      string       `json:"dataCacheDir"`
	PythonPath        string         `json:"pythonPath"`
	MaxPreviewRows    int            `json:"maxPreviewRows"`
	DetailedLog       bool           `json:"detailedLog"`
	MCPServices       []MCPService   `json:"mcpServices"`     // Generic MCP services configuration
	SearchEngines     []SearchEngine `json:"searchEngines"`   // Search engines configuration
	ActiveSearchEngine string        `json:"activeSearchEngine,omitempty"` // ID of active search engine
	ProxyConfig       *ProxyConfig   `json:"proxyConfig,omitempty"` // Proxy server configuration
	
	// Deprecated: Legacy web search fields (kept for backward compatibility)
	WebSearchProvider string `json:"webSearchProvider,omitempty"`
	WebSearchAPIKey   string `json:"webSearchAPIKey,omitempty"`
	WebSearchMCPURL   string `json:"webSearchMCPURL,omitempty"`
}
