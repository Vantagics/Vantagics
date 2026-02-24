package config

// IntentEnhancementConfig 增强功能配置
// 用于控制意图理解增强功能的各项开关和参数
type IntentEnhancementConfig struct {
	// EnableContextEnhancement 启用上下文增�?
	// 当启用时，系统会从历史分析记录中检索上下文信息来增强意图建�?
	EnableContextEnhancement bool `json:"enable_context_enhancement"`

	// EnablePreferenceLearning 启用偏好学习
	// 当启用时，系统会记录用户的意图选择并根据历史偏好对建议进行排序
	EnablePreferenceLearning bool `json:"enable_preference_learning"`

	// EnableDynamicDimensions 启用动态维度调�?
	// 当启用时，系统会根据数据特征自动调整分析维度建议
	EnableDynamicDimensions bool `json:"enable_dynamic_dimensions"`

	// EnableFewShotExamples 启用Few-shot示例
	// 当启用时，系统会在提示词中包含领域特定的示例来提高建议质�?
	EnableFewShotExamples bool `json:"enable_few_shot_examples"`

	// EnableCaching 启用缓存机制
	// 当启用时，系统会缓存相似请求的意图建议以减少LLM调用
	EnableCaching bool `json:"enable_caching"`

	// CacheSimilarityThreshold 缓存相似度阈�?
	// 当两个请求的语义相似度超过此阈值时，返回缓存的建议
	// 默认�? 0.85
	CacheSimilarityThreshold float64 `json:"cache_similarity_threshold"`

	// CacheExpirationHours 缓存过期时间（小时）
	// 缓存条目在此时间后将被视为过期并清理
	// 默认�? 24
	CacheExpirationHours int `json:"cache_expiration_hours"`

	// MaxCacheEntries 最大缓存条目数
	// 当缓存条目超过此限制时，使用LRU策略清理最少使用的条目
	// 默认�? 1000
	MaxCacheEntries int `json:"max_cache_entries"`

	// MaxHistoryRecords 最大历史记录数
	// 上下文增强时包含的最大历史分析记录数�?
	// 默认�? 10
	MaxHistoryRecords int `json:"max_history_records"`
}

// DefaultIntentEnhancementConfig 返回默认的意图增强配�?
// 默认情况下所有增强功能都启用，使用推荐的参数�?
func DefaultIntentEnhancementConfig() *IntentEnhancementConfig {
	return &IntentEnhancementConfig{
		EnableContextEnhancement:  true,
		EnablePreferenceLearning:  true,
		EnableDynamicDimensions:   true,
		EnableFewShotExamples:     true,
		EnableCaching:             true,
		CacheSimilarityThreshold:  0.85,
		CacheExpirationHours:      24,
		MaxCacheEntries:           1000,
		MaxHistoryRecords:         10,
	}
}

// DisabledIntentEnhancementConfig 返回禁用所有增强功能的配置
// 用于向后兼容或降级场�?
func DisabledIntentEnhancementConfig() *IntentEnhancementConfig {
	return &IntentEnhancementConfig{
		EnableContextEnhancement:  false,
		EnablePreferenceLearning:  false,
		EnableDynamicDimensions:   false,
		EnableFewShotExamples:     false,
		EnableCaching:             false,
		CacheSimilarityThreshold:  0.85,
		CacheExpirationHours:      24,
		MaxCacheEntries:           1000,
		MaxHistoryRecords:         10,
	}
}

// IsAllDisabled 检查是否所有增强功能都已禁�?
// 当所有功能都禁用时，系统应表现与原始版本一�?
func (c *IntentEnhancementConfig) IsAllDisabled() bool {
	return !c.EnableContextEnhancement &&
		!c.EnablePreferenceLearning &&
		!c.EnableDynamicDimensions &&
		!c.EnableFewShotExamples &&
		!c.EnableCaching
}

// Validate 验证配置参数的有效�?
// 如果参数无效，将其设置为默认�?
func (c *IntentEnhancementConfig) Validate() {
	// 验证缓存相似度阈�?(0.0 - 1.0)
	if c.CacheSimilarityThreshold < 0.0 || c.CacheSimilarityThreshold > 1.0 {
		c.CacheSimilarityThreshold = 0.85
	}

	// 验证缓存过期时间 (至少1小时)
	if c.CacheExpirationHours < 1 {
		c.CacheExpirationHours = 24
	}

	// 验证最大缓存条目数 (至少1�?
	if c.MaxCacheEntries < 1 {
		c.MaxCacheEntries = 1000
	}

	// 验证最大历史记录数 (至少1�?
	if c.MaxHistoryRecords < 1 {
		c.MaxHistoryRecords = 10
	}
}

// Clone 创建配置的深拷贝
func (c *IntentEnhancementConfig) Clone() *IntentEnhancementConfig {
	return &IntentEnhancementConfig{
		EnableContextEnhancement:  c.EnableContextEnhancement,
		EnablePreferenceLearning:  c.EnablePreferenceLearning,
		EnableDynamicDimensions:   c.EnableDynamicDimensions,
		EnableFewShotExamples:     c.EnableFewShotExamples,
		EnableCaching:             c.EnableCaching,
		CacheSimilarityThreshold:  c.CacheSimilarityThreshold,
		CacheExpirationHours:      c.CacheExpirationHours,
		MaxCacheEntries:           c.MaxCacheEntries,
		MaxHistoryRecords:         c.MaxHistoryRecords,
	}
}
