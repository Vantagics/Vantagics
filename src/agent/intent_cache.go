package agent

import (
	"container/list"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IntentCache 意图缓存
// Caches intent suggestions for similar requests to reduce LLM calls
// Uses LRU eviction strategy when cache exceeds max entries
// Supports semantic similarity matching for cache lookups
// Validates: Requirements 5.4, 5.5, 5.6
type IntentCache struct {
	cache      map[string]*CacheEntry      // Main cache storage: key -> entry
	lruList    *list.List                  // LRU tracking: front = most recently used
	lruMap     map[string]*list.Element    // Maps cache key to LRU list element
	maxEntries int                         // Maximum number of cache entries
	expiration time.Duration               // Cache entry expiration duration
	similarity *SemanticSimilarityCalculator // Semantic similarity calculator
	dataDir    string                      // Directory for JSON persistence
	mu         sync.RWMutex                // Read-write mutex for thread safety
	
	// Statistics
	hitCount  int64 // Number of cache hits
	missCount int64 // Number of cache misses
}

// CacheEntry 缓存条目
// Represents a single cached intent suggestion result
// Validates: Requirements 5.4
type CacheEntry struct {
	Key          string             `json:"key"`            // Unique cache key (dataSourceID + userMessage hash)
	DataSourceID string             `json:"data_source_id"` // Data source identifier
	UserMessage  string             `json:"user_message"`   // Original user message/request
	Embedding    []float64          `json:"embedding"`      // Text embedding for similarity matching
	Suggestions  []IntentSuggestion `json:"suggestions"`    // Cached intent suggestions
	CreatedAt    time.Time          `json:"created_at"`     // Entry creation timestamp
	AccessCount  int                `json:"access_count"`   // Number of times this entry was accessed
	LastAccessed time.Time          `json:"last_accessed"`  // Last access timestamp
}

// CacheStats 缓存统计信息
// Provides statistics about cache performance
type CacheStats struct {
	TotalEntries int     `json:"total_entries"` // Current number of entries in cache
	MaxEntries   int     `json:"max_entries"`   // Maximum allowed entries
	HitCount     int64   `json:"hit_count"`     // Total cache hits
	MissCount    int64   `json:"miss_count"`    // Total cache misses
	HitRate      float64 `json:"hit_rate"`      // Hit rate percentage (0.0 to 1.0)
}

// CachePersistence 缓存持久化结构
// Used for JSON serialization of cache state
type CachePersistence struct {
	Entries  []*CacheEntry `json:"entries"`   // All cache entries
	Stats    CacheStats    `json:"stats"`     // Cache statistics
	SavedAt  time.Time     `json:"saved_at"`  // Timestamp when cache was saved
}

// Default configuration values
const (
	DefaultMaxCacheEntries    = 1000
	DefaultCacheExpirationHrs = 24
	DefaultSimilarityThreshold = 0.85
	CacheFileName             = "intent_cache.json"
)

// NewIntentCache 创建意图缓存
// Creates a new IntentCache with the specified configuration
// Parameters:
//   - maxEntries: maximum number of cache entries (default 1000 if <= 0)
//   - expirationHours: cache entry expiration in hours (default 24 if <= 0)
//   - similarityThreshold: threshold for semantic similarity matching (default 0.85)
//
// Returns a new IntentCache instance
// Validates: Requirements 5.4, 5.5, 5.6
func NewIntentCache(
	maxEntries int,
	expirationHours int,
	similarityThreshold float64,
) *IntentCache {
	// Apply defaults for invalid values
	if maxEntries <= 0 {
		maxEntries = DefaultMaxCacheEntries
	}
	if expirationHours <= 0 {
		expirationHours = DefaultCacheExpirationHrs
	}
	if similarityThreshold <= 0 || similarityThreshold > 1.0 {
		similarityThreshold = DefaultSimilarityThreshold
	}

	return &IntentCache{
		cache:      make(map[string]*CacheEntry),
		lruList:    list.New(),
		lruMap:     make(map[string]*list.Element),
		maxEntries: maxEntries,
		expiration: time.Duration(expirationHours) * time.Hour,
		similarity: NewSemanticSimilarityCalculator(similarityThreshold),
		hitCount:   0,
		missCount:  0,
	}
}

// NewIntentCacheWithDataDir 创建带持久化目录的意图缓存
// Creates a new IntentCache with persistence support
// Parameters:
//   - maxEntries: maximum number of cache entries
//   - expirationHours: cache entry expiration in hours
//   - similarityThreshold: threshold for semantic similarity matching
//   - dataDir: directory for JSON persistence
//
// Returns a new IntentCache instance with persistence enabled
func NewIntentCacheWithDataDir(
	maxEntries int,
	expirationHours int,
	similarityThreshold float64,
	dataDir string,
) *IntentCache {
	cache := NewIntentCache(maxEntries, expirationHours, similarityThreshold)
	cache.dataDir = dataDir
	return cache
}

// GenerateCacheKey 生成缓存键
// Generates a unique cache key from dataSourceID and userMessage
// The key format ensures different (dataSourceID, userMessage) combinations
// produce different keys
// Parameters:
//   - dataSourceID: the data source identifier
//   - userMessage: the user's message/request
//
// Returns a unique cache key string
// Validates: Requirements 5.4
func GenerateCacheKey(dataSourceID, userMessage string) string {
	// Use a simple concatenation with separator
	// This ensures uniqueness: different combinations produce different keys
	return fmt.Sprintf("%s|%s", dataSourceID, userMessage)
}

// Initialize 初始化缓存
// Loads cache from disk if persistence is enabled
// Returns error if loading fails
func (c *IntentCache) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If no data directory specified, skip loading
	if c.dataDir == "" {
		return nil
	}

	// Try to load from disk
	return c.loadFromDisk()
}

// Get 获取缓存
// Retrieves cached suggestions for a request using semantic similarity matching
// Parameters:
//   - dataSourceID: the data source identifier
//   - userMessage: the user's message/request
//
// Returns the cached suggestions and true if found, nil and false otherwise
// Also updates LRU order and access statistics on hit
// Validates: Requirements 5.1, 5.2
func (c *IntentCache) Get(dataSourceID, userMessage string) ([]IntentSuggestion, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// First, try exact match
	key := GenerateCacheKey(dataSourceID, userMessage)
	if entry, exists := c.cache[key]; exists {
		// Check if entry is expired
		if c.isExpired(entry) {
			c.removeEntry(key)
			c.missCount++
			return nil, false
		}

		// Update LRU and access stats
		c.updateLRU(key)
		entry.AccessCount++
		entry.LastAccessed = time.Now()
		c.hitCount++

		return entry.Suggestions, true
	}

	// Try semantic similarity matching for entries with same dataSourceID
	for k, entry := range c.cache {
		// Skip entries from different data sources
		if entry.DataSourceID != dataSourceID {
			continue
		}

		// Check if entry is expired
		if c.isExpired(entry) {
			c.removeEntry(k)
			continue
		}

		// Check semantic similarity
		if c.similarity.IsSimilar(userMessage, entry.UserMessage) {
			// Update LRU and access stats
			c.updateLRU(k)
			entry.AccessCount++
			entry.LastAccessed = time.Now()
			c.hitCount++

			return entry.Suggestions, true
		}
	}

	c.missCount++
	return nil, false
}

// Set 设置缓存
// Stores intent suggestions in the cache
// Parameters:
//   - dataSourceID: the data source identifier
//   - userMessage: the user's message/request
//   - suggestions: the intent suggestions to cache
//
// Automatically evicts LRU entries if cache exceeds max size
// Validates: Requirements 5.3, 5.6
func (c *IntentCache) Set(dataSourceID, userMessage string, suggestions []IntentSuggestion) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := GenerateCacheKey(dataSourceID, userMessage)
	now := time.Now()

	// Check if entry already exists
	if _, exists := c.cache[key]; exists {
		// Update existing entry
		c.cache[key].Suggestions = suggestions
		c.cache[key].LastAccessed = now
		c.updateLRU(key)
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:          key,
		DataSourceID: dataSourceID,
		UserMessage:  userMessage,
		Embedding:    c.similarity.GetEmbedding(userMessage),
		Suggestions:  suggestions,
		CreatedAt:    now,
		AccessCount:  0,
		LastAccessed: now,
	}

	// Evict LRU entries if necessary
	for len(c.cache) >= c.maxEntries {
		c.evictLRU()
	}

	// Add to cache
	c.cache[key] = entry

	// Add to LRU list (front = most recently used)
	element := c.lruList.PushFront(key)
	c.lruMap[key] = element
}

// Clear 清除缓存
// Removes all entries from the cache
func (c *IntentCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheEntry)
	c.lruList = list.New()
	c.lruMap = make(map[string]*list.Element)
	// Note: We don't reset hit/miss counts as they represent historical stats
}

// GetStats 获取缓存统计
// Returns current cache statistics
func (c *IntentCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalRequests := c.hitCount + c.missCount
	var hitRate float64
	if totalRequests > 0 {
		hitRate = float64(c.hitCount) / float64(totalRequests)
	}

	return CacheStats{
		TotalEntries: len(c.cache),
		MaxEntries:   c.maxEntries,
		HitCount:     c.hitCount,
		MissCount:    c.missCount,
		HitRate:      hitRate,
	}
}

// Size 获取缓存大小
// Returns the current number of entries in the cache
func (c *IntentCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// GetMaxEntries 获取最大条目数
// Returns the maximum number of entries allowed in the cache
func (c *IntentCache) GetMaxEntries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxEntries
}

// GetExpiration 获取过期时间
// Returns the cache entry expiration duration
func (c *IntentCache) GetExpiration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.expiration
}

// SetExpiration 设置过期时间
// Updates the cache entry expiration duration
func (c *IntentCache) SetExpiration(expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expiration = expiration
}

// CleanExpired 清理过期条目
// Removes all expired entries from the cache
// Returns the number of entries removed
// Validates: Requirements 5.5
func (c *IntentCache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for key, entry := range c.cache {
		if c.isExpired(entry) {
			c.removeEntry(key)
			removed++
		}
	}

	return removed
}

// Save 保存缓存到磁盘
// Persists the cache to a JSON file
// Returns error if saving fails
func (c *IntentCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.dataDir == "" {
		return nil // No persistence configured
	}

	return c.saveToDisk()
}

// Load 从磁盘加载缓存
// Loads the cache from a JSON file
// Returns error if loading fails
func (c *IntentCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dataDir == "" {
		return nil // No persistence configured
	}

	return c.loadFromDisk()
}

// isExpired checks if a cache entry has expired
// Validates: Requirements 5.5
func (c *IntentCache) isExpired(entry *CacheEntry) bool {
	return time.Since(entry.CreatedAt) > c.expiration
}

// updateLRU moves an entry to the front of the LRU list
func (c *IntentCache) updateLRU(key string) {
	if element, exists := c.lruMap[key]; exists {
		c.lruList.MoveToFront(element)
	}
}

// evictLRU removes the least recently used entry
// Validates: Requirements 5.6
func (c *IntentCache) evictLRU() {
	// Get the back element (least recently used)
	back := c.lruList.Back()
	if back == nil {
		return
	}

	key := back.Value.(string)
	c.removeEntry(key)
}

// removeEntry removes an entry from the cache and LRU structures
func (c *IntentCache) removeEntry(key string) {
	// Remove from cache
	delete(c.cache, key)

	// Remove from LRU list
	if element, exists := c.lruMap[key]; exists {
		c.lruList.Remove(element)
		delete(c.lruMap, key)
	}
}

// saveToDisk saves the cache to a JSON file
func (c *IntentCache) saveToDisk() error {
	// Ensure directory exists
	if err := os.MkdirAll(c.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Collect all entries
	entries := make([]*CacheEntry, 0, len(c.cache))
	for _, entry := range c.cache {
		entries = append(entries, entry)
	}

	// Create persistence structure
	persistence := CachePersistence{
		Entries: entries,
		Stats: CacheStats{
			TotalEntries: len(c.cache),
			MaxEntries:   c.maxEntries,
			HitCount:     c.hitCount,
			MissCount:    c.missCount,
		},
		SavedAt: time.Now(),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(persistence, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to file
	filePath := filepath.Join(c.dataDir, CacheFileName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// loadFromDisk loads the cache from a JSON file
func (c *IntentCache) loadFromDisk() error {
	filePath := filepath.Join(c.dataDir, CacheFileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No cache file, start fresh
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	// Unmarshal JSON
	var persistence CachePersistence
	if err := json.Unmarshal(data, &persistence); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Restore cache entries (skip expired ones)
	for _, entry := range persistence.Entries {
		if !c.isExpired(entry) {
			c.cache[entry.Key] = entry
			element := c.lruList.PushBack(entry.Key)
			c.lruMap[entry.Key] = element
		}
	}

	// Restore statistics
	c.hitCount = persistence.Stats.HitCount
	c.missCount = persistence.Stats.MissCount

	return nil
}

// GetAllEntries 获取所有缓存条目
// Returns a copy of all cache entries (for debugging/testing)
func (c *IntentCache) GetAllEntries() []*CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]*CacheEntry, 0, len(c.cache))
	for _, entry := range c.cache {
		// Create a copy to avoid external modification
		entryCopy := *entry
		entries = append(entries, &entryCopy)
	}

	return entries
}

// GetEntry 获取指定键的缓存条目
// Returns the cache entry for the given key (for debugging/testing)
func (c *IntentCache) GetEntry(dataSourceID, userMessage string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := GenerateCacheKey(dataSourceID, userMessage)
	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Return a copy
	entryCopy := *entry
	return &entryCopy, true
}

// Contains 检查缓存是否包含指定键
// Returns true if the cache contains an entry for the given key
func (c *IntentCache) Contains(dataSourceID, userMessage string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := GenerateCacheKey(dataSourceID, userMessage)
	_, exists := c.cache[key]
	return exists
}

// Remove 移除指定键的缓存条目
// Removes the cache entry for the given key
// Returns true if an entry was removed, false if not found
func (c *IntentCache) Remove(dataSourceID, userMessage string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := GenerateCacheKey(dataSourceID, userMessage)
	if _, exists := c.cache[key]; !exists {
		return false
	}

	c.removeEntry(key)
	return true
}

// ResetStats 重置统计信息
// Resets hit and miss counters to zero
func (c *IntentCache) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hitCount = 0
	c.missCount = 0
}
