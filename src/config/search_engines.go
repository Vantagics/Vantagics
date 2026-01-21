package config

// GetDefaultSearchAPIs returns the default search API services
func GetDefaultSearchAPIs() []SearchAPIConfig {
	return []SearchAPIConfig{
		{
			ID:          "duckduckgo",
			Name:        "DuckDuckGo",
			Description: "Free search API with no API key required",
			Enabled:     true,
			Tested:      false,
		},
		{
			ID:          "serper",
			Name:        "Serper (Google Search)",
			Description: "Google Search API via Serper.dev (requires API key)",
			APIKey:      "",
			Enabled:     false,
			Tested:      false,
		},
		{
			ID:          "uapi_pro",
			Name:        "UAPI Pro",
			Description: "UAPI Pro search service with structured data (requires API key)",
			APIKey:      "",
			Enabled:     false,
			Tested:      false,
		},
	}
}

// InitializeSearchAPIs initializes search API services if not present
func (c *Config) InitializeSearchAPIs() {
	if len(c.SearchAPIs) == 0 {
		c.SearchAPIs = GetDefaultSearchAPIs()
	}
	
	// Migrate from old UAPIConfig if present
	if c.UAPIConfig != nil && c.UAPIConfig.Enabled && c.UAPIConfig.APIToken != "" {
		for i := range c.SearchAPIs {
			if c.SearchAPIs[i].ID == "uapi_pro" {
				c.SearchAPIs[i].APIKey = c.UAPIConfig.APIToken
				c.SearchAPIs[i].Enabled = c.UAPIConfig.Enabled
				c.SearchAPIs[i].Tested = c.UAPIConfig.Tested
				break
			}
		}
		// Clear old config
		c.UAPIConfig = nil
	}
	
	// Set default active search API if not set
	if c.ActiveSearchAPI == "" {
		// Default to DuckDuckGo (free, no API key required)
		c.ActiveSearchAPI = "duckduckgo"
		for i := range c.SearchAPIs {
			if c.SearchAPIs[i].ID == "duckduckgo" {
				c.SearchAPIs[i].Enabled = true
			}
		}
	}
	
	// Ensure at least one API is enabled
	hasEnabled := false
	for _, api := range c.SearchAPIs {
		if api.Enabled {
			hasEnabled = true
			break
		}
	}
	
	if !hasEnabled && len(c.SearchAPIs) > 0 {
		// Enable DuckDuckGo as fallback
		for i := range c.SearchAPIs {
			if c.SearchAPIs[i].ID == "duckduckgo" {
				c.SearchAPIs[i].Enabled = true
				c.ActiveSearchAPI = "duckduckgo"
				break
			}
		}
	}
}

// GetActiveSearchAPI returns the currently active search API
func (c *Config) GetActiveSearchAPI() *SearchAPIConfig {
	for i := range c.SearchAPIs {
		if c.SearchAPIs[i].ID == c.ActiveSearchAPI {
			return &c.SearchAPIs[i]
		}
	}
	
	// Fallback to first enabled API
	for i := range c.SearchAPIs {
		if c.SearchAPIs[i].Enabled {
			return &c.SearchAPIs[i]
		}
	}
	
	return nil
}

// SetActiveSearchAPI sets the active search API (user choice)
func (c *Config) SetActiveSearchAPI(apiID string) bool {
	// Find and enable the selected API
	found := false
	for i := range c.SearchAPIs {
		if c.SearchAPIs[i].ID == apiID {
			c.SearchAPIs[i].Enabled = true
			c.ActiveSearchAPI = apiID
			found = true
			break
		}
	}
	
	return found
}

// UpdateSearchAPIConfig updates a search API configuration
func (c *Config) UpdateSearchAPIConfig(apiID string, apiKey string, customID string, enabled bool, tested bool) bool {
	for i := range c.SearchAPIs {
		if c.SearchAPIs[i].ID == apiID {
			if apiKey != "" {
				c.SearchAPIs[i].APIKey = apiKey
			}
			if customID != "" {
				c.SearchAPIs[i].CustomID = customID
			}
			c.SearchAPIs[i].Enabled = enabled
			c.SearchAPIs[i].Tested = tested
			return true
		}
	}
	return false
}

// Legacy functions for backward compatibility

// GetDefaultSearchEngines returns the default search engines (DEPRECATED)
func GetDefaultSearchEngines() []SearchEngine {
	return []SearchEngine{
		{
			ID:      "google",
			Name:    "Google",
			URL:     "www.google.com",
			Enabled: true,
			Tested:  true,
		},
		{
			ID:      "bing",
			Name:    "Bing",
			URL:     "www.bing.com",
			Enabled: true,
			Tested:  true,
		},
		{
			ID:      "baidu",
			Name:    "Baidu (百度)",
			URL:     "www.baidu.com",
			Enabled: false,
			Tested:  true,
		},
	}
}

// InitializeSearchEngines initializes search engines if not present (DEPRECATED)
func (c *Config) InitializeSearchEngines() {
	// Migrate to new SearchAPIs if old SearchEngines exist
	if len(c.SearchEngines) > 0 && len(c.SearchAPIs) == 0 {
		c.SearchAPIs = GetDefaultSearchAPIs()
		c.SearchEngines = nil
	}
	
	// Use new API-based initialization
	c.InitializeSearchAPIs()
}

// GetActiveSearchEngine returns the currently active search engine (DEPRECATED)
func (c *Config) GetActiveSearchEngine() *SearchEngine {
	// Return nil to force migration to new API system
	return nil
}

// SetActiveSearchEngine sets the active search engine (DEPRECATED)
func (c *Config) SetActiveSearchEngine(engineID string) bool {
	// Redirect to new API system
	return c.SetActiveSearchAPI(engineID)
}

// IsUserSelectedEngine returns true if user has explicitly selected a search engine (DEPRECATED)
func (c *Config) IsUserSelectedEngine() bool {
	return c.ActiveSearchAPI != ""
}
