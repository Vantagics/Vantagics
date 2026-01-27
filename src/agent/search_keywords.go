package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SearchKeywordsConfig holds the configuration for web search keyword detection
type SearchKeywordsConfig struct {
	// Default keywords that are always included
	DefaultKeywords []string `json:"default_keywords"`
	// User-added keywords that are learned from usage
	LearnedKeywords []string `json:"learned_keywords"`
	// Keywords that have been used and their usage count
	KeywordUsage map[string]int `json:"keyword_usage"`
	// Last updated timestamp
	LastUpdated time.Time `json:"last_updated"`
}

// SearchKeywordsManager manages web search keywords with learning capability
type SearchKeywordsManager struct {
	config     *SearchKeywordsConfig
	configPath string
	mu         sync.RWMutex
	logFunc    func(string)
}

// NewSearchKeywordsManager creates a new search keywords manager
func NewSearchKeywordsManager(dataDir string, logFunc func(string)) *SearchKeywordsManager {
	configPath := filepath.Join(dataDir, "search_keywords.json")
	
	manager := &SearchKeywordsManager{
		configPath: configPath,
		logFunc:    logFunc,
	}
	
	// Load existing config or create default
	manager.loadOrCreateConfig()
	
	return manager
}

// loadOrCreateConfig loads the config from file or creates a default one
func (m *SearchKeywordsManager) loadOrCreateConfig() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Try to load existing config
	data, err := os.ReadFile(m.configPath)
	if err == nil {
		var config SearchKeywordsConfig
		if err := json.Unmarshal(data, &config); err == nil {
			m.config = &config
			m.log("[SEARCH-KEYWORDS] Loaded config with %d default and %d learned keywords", 
				len(config.DefaultKeywords), len(config.LearnedKeywords))
			return
		}
	}
	
	// Create default config
	m.config = &SearchKeywordsConfig{
		DefaultKeywords: getDefaultSearchKeywords(),
		LearnedKeywords: []string{},
		KeywordUsage:    make(map[string]int),
		LastUpdated:     time.Now(),
	}
	
	// Save the default config
	m.saveConfigLocked()
	m.log("[SEARCH-KEYWORDS] Created default config with %d keywords", len(m.config.DefaultKeywords))
}

// getDefaultSearchKeywords returns the default set of search keywords
func getDefaultSearchKeywords() []string {
	return []string{
		// Chinese keywords - Weather
		"天气", "气温", "温度", "下雨", "下雪", "晴天", "阴天", "预报",
		// Chinese keywords - News & Time
		"新闻", "最新", "今天", "现在", "实时", "当前", "最近",
		// Chinese keywords - Finance
		"股票", "股价", "汇率", "价格", "多少钱", "行情", "涨跌",
		// Chinese keywords - Search intent
		"搜索", "查询", "查一下", "帮我查", "帮我搜", "搜一下",
		// Chinese keywords - Web
		"网上", "网络", "互联网", "在线",
		// Chinese keywords - Location
		"哪里", "地址", "位置", "怎么走", "路线", "城市", "在哪",
		// Chinese keywords - Events
		"比赛", "赛事", "比分", "结果",
		// Chinese keywords - Travel & Transportation
		"航班", "机票", "火车", "高铁", "酒店", "旅游", "出行", "订票", "飞机",
		// Chinese keywords - Time & Date (for local time tool)
		"几点", "时间", "日期", "几号", "星期几", "周几", "几月", "年份",
		
		// English keywords - Weather
		"weather", "temperature", "rain", "snow", "sunny", "cloudy", "forecast",
		// English keywords - News & Time
		"news", "latest", "today", "now", "current", "real-time", "recent",
		// English keywords - Finance
		"stock", "price", "exchange rate", "how much", "market", "trading",
		// English keywords - Search intent
		"search", "look up", "find", "google", "search for",
		// English keywords - Web
		"online", "internet", "web",
		// English keywords - Location
		"where", "address", "location", "directions", "route",
		// English keywords - Events
		"score", "match", "game", "result",
		// English keywords - Travel & Transportation
		"flight", "flights", "ticket", "train", "hotel", "travel", "booking", "airline",
		// English keywords - Time & Date (for local time tool)
		"time", "date", "day", "what day", "what time", "clock", "hour", "minute",
	}
}

// saveConfigLocked saves the config to file (must be called with lock held)
func (m *SearchKeywordsManager) saveConfigLocked() error {
	m.config.LastUpdated = time.Now()
	
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(m.configPath, data, 0644)
}

// SaveConfig saves the config to file
func (m *SearchKeywordsManager) SaveConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveConfigLocked()
}

// GetAllKeywords returns all keywords (default + learned)
func (m *SearchKeywordsManager) GetAllKeywords() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Combine default and learned keywords
	keywords := make([]string, 0, len(m.config.DefaultKeywords)+len(m.config.LearnedKeywords))
	keywords = append(keywords, m.config.DefaultKeywords...)
	keywords = append(keywords, m.config.LearnedKeywords...)
	
	return keywords
}

// DetectSearchNeed checks if the message requires web search
func (m *SearchKeywordsManager) DetectSearchNeed(message string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	lowerMessage := strings.ToLower(message)
	
	// Check all keywords
	allKeywords := append(m.config.DefaultKeywords, m.config.LearnedKeywords...)
	for _, keyword := range allKeywords {
		if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
			return true, keyword
		}
	}
	
	return false, ""
}

// RecordKeywordUsage records that a keyword was used for search
func (m *SearchKeywordsManager) RecordKeywordUsage(keyword string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.config.KeywordUsage == nil {
		m.config.KeywordUsage = make(map[string]int)
	}
	
	m.config.KeywordUsage[keyword]++
	m.saveConfigLocked()
}

// LearnKeyword adds a new keyword to the learned list
func (m *SearchKeywordsManager) LearnKeyword(keyword string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if keyword already exists
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	
	lowerKeyword := strings.ToLower(keyword)
	
	// Check in default keywords
	for _, k := range m.config.DefaultKeywords {
		if strings.ToLower(k) == lowerKeyword {
			return false // Already exists
		}
	}
	
	// Check in learned keywords
	for _, k := range m.config.LearnedKeywords {
		if strings.ToLower(k) == lowerKeyword {
			return false // Already exists
		}
	}
	
	// Add to learned keywords
	m.config.LearnedKeywords = append(m.config.LearnedKeywords, keyword)
	m.saveConfigLocked()
	m.log("[SEARCH-KEYWORDS] Learned new keyword: %s", keyword)
	
	return true
}

// LearnFromSuccessfulSearch learns keywords from a successful search query
// It extracts potential keywords from the query that led to a successful search
func (m *SearchKeywordsManager) LearnFromSuccessfulSearch(query string) {
	// Extract potential keywords from the query
	// Look for patterns that might indicate search intent
	potentialKeywords := extractPotentialKeywords(query)
	
	for _, keyword := range potentialKeywords {
		m.LearnKeyword(keyword)
	}
}

// extractPotentialKeywords extracts potential search keywords from a query
func extractPotentialKeywords(query string) []string {
	var keywords []string
	
	// Common patterns that indicate search intent
	patterns := []struct {
		prefix string
		suffix string
	}{
		{"查", ""},
		{"搜", ""},
		{"找", ""},
		{"", "是什么"},
		{"", "怎么样"},
		{"", "多少"},
		{"what is ", ""},
		{"how ", ""},
		{"where ", ""},
		{"when ", ""},
	}
	
	lowerQuery := strings.ToLower(query)
	
	for _, p := range patterns {
		if p.prefix != "" && strings.Contains(lowerQuery, p.prefix) {
			// Extract word after prefix
			idx := strings.Index(lowerQuery, p.prefix)
			if idx >= 0 {
				rest := query[idx+len(p.prefix):]
				// Take first word or phrase
				words := strings.Fields(rest)
				if len(words) > 0 && len(words[0]) >= 2 {
					keywords = append(keywords, words[0])
				}
			}
		}
		if p.suffix != "" && strings.Contains(lowerQuery, p.suffix) {
			// Extract word before suffix
			idx := strings.Index(lowerQuery, p.suffix)
			if idx > 0 {
				before := query[:idx]
				words := strings.Fields(before)
				if len(words) > 0 {
					lastWord := words[len(words)-1]
					if len(lastWord) >= 2 {
						keywords = append(keywords, lastWord)
					}
				}
			}
		}
	}
	
	return keywords
}

// RemoveLearnedKeyword removes a keyword from the learned list
func (m *SearchKeywordsManager) RemoveLearnedKeyword(keyword string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	lowerKeyword := strings.ToLower(keyword)
	
	for i, k := range m.config.LearnedKeywords {
		if strings.ToLower(k) == lowerKeyword {
			m.config.LearnedKeywords = append(m.config.LearnedKeywords[:i], m.config.LearnedKeywords[i+1:]...)
			m.saveConfigLocked()
			m.log("[SEARCH-KEYWORDS] Removed learned keyword: %s", keyword)
			return true
		}
	}
	
	return false
}

// GetLearnedKeywords returns only the learned keywords
func (m *SearchKeywordsManager) GetLearnedKeywords() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]string, len(m.config.LearnedKeywords))
	copy(result, m.config.LearnedKeywords)
	return result
}

// GetKeywordStats returns usage statistics for keywords
func (m *SearchKeywordsManager) GetKeywordStats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]int)
	for k, v := range m.config.KeywordUsage {
		result[k] = v
	}
	return result
}

// log logs a message using the provided log function
func (m *SearchKeywordsManager) log(format string, args ...interface{}) {
	if m.logFunc != nil {
		m.logFunc(strings.ReplaceAll(format, "%s", "%v"))
	}
}
