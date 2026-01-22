package config

// GetDefaultSearchEngines returns the default search engines
func GetDefaultSearchEngines() []SearchEngine {
	return []SearchEngine{
		{
			ID:      "google",
			Name:    "Google",
			URL:     "www.google.com",
			Enabled: true,  // Available but not necessarily active
			Tested:  true,  // Pre-tested
		},
		{
			ID:      "bing",
			Name:    "Bing",
			URL:     "www.bing.com",
			Enabled: true,  // Available but not necessarily active
			Tested:  true,  // Pre-tested
		},
		{
			ID:      "baidu",
			Name:    "Baidu (百度)",
			URL:     "www.baidu.com",
			Enabled: false, // Disabled by default
			Tested:  true,  // Pre-tested
		},
	}
}

// InitializeSearchEngines initializes search engines if not present
func (c *Config) InitializeSearchEngines() {
	if len(c.SearchEngines) == 0 {
		c.SearchEngines = GetDefaultSearchEngines()
	}
	
	// If user hasn't made a choice (ActiveSearchEngine is empty), set default based on language
	if c.ActiveSearchEngine == "" {
		if c.Language == "简体中文" {
			// Chinese users default to Bing
			c.ActiveSearchEngine = "bing"
			// Enable Bing, disable others
			for i := range c.SearchEngines {
				c.SearchEngines[i].Enabled = (c.SearchEngines[i].ID == "bing")
			}
		} else {
			// English users default to Google
			c.ActiveSearchEngine = "google"
			// Enable Google, disable others
			for i := range c.SearchEngines {
				c.SearchEngines[i].Enabled = (c.SearchEngines[i].ID == "google")
			}
		}
	} else {
		// User has made a choice, ensure the selected engine is enabled
		for i := range c.SearchEngines {
			if c.SearchEngines[i].ID == c.ActiveSearchEngine {
				c.SearchEngines[i].Enabled = true
			}
		}
	}
	
	// Ensure at least one engine is enabled (fallback)
	hasEnabled := false
	for _, engine := range c.SearchEngines {
		if engine.Enabled {
			hasEnabled = true
			break
		}
	}
	
	if !hasEnabled && len(c.SearchEngines) > 0 {
		// Enable the first engine as fallback
		c.SearchEngines[0].Enabled = true
		c.ActiveSearchEngine = c.SearchEngines[0].ID
	}
}

// GetActiveSearchEngine returns the currently active search engine
func (c *Config) GetActiveSearchEngine() *SearchEngine {
	for i := range c.SearchEngines {
		if c.SearchEngines[i].ID == c.ActiveSearchEngine {
			return &c.SearchEngines[i]
		}
	}
	
	// Fallback to first enabled engine
	for i := range c.SearchEngines {
		if c.SearchEngines[i].Enabled {
			return &c.SearchEngines[i]
		}
	}
	
	return nil
}

// SetActiveSearchEngine sets the active search engine (user choice)
func (c *Config) SetActiveSearchEngine(engineID string) bool {
	// Find and enable the selected engine
	found := false
	for i := range c.SearchEngines {
		if c.SearchEngines[i].ID == engineID {
			c.SearchEngines[i].Enabled = true
			c.ActiveSearchEngine = engineID
			found = true
			break
		}
	}
	
	return found
}

// IsUserSelectedEngine returns true if user has explicitly selected a search engine
func (c *Config) IsUserSelectedEngine() bool {
	// If ActiveSearchEngine is set and doesn't match language default, user has made a choice
	if c.ActiveSearchEngine == "" {
		return false
	}
	
	// Check if it matches the language default
	if c.Language == "简体中文" {
		return c.ActiveSearchEngine != "bing"
	}
	
	return c.ActiveSearchEngine != "google"
}
