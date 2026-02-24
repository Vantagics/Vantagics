package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// IntentUnderstandingConfig 意图理解配置
// 简化为5个核心配置项，用于控制意图理解系统的行为
// Validates: Requirements 7.3, 11.2
type IntentUnderstandingConfig struct {
	// Enabled 是否启用意图理解
	// 当禁用时，系统将跳过意图理解流程，直接使用用户原始请�?
	Enabled bool `json:"enabled"`

	// MaxSuggestions 最大建议数�?
	// 控制每次生成的意图建议数量上限，默认5
	MaxSuggestions int `json:"max_suggestions"`

	// MaxHistoryRecords 最大历史记录数
	// 上下文增强时包含的最大历史分析记录数量，默认5
	MaxHistoryRecords int `json:"max_history_records"`

	// PreferenceThreshold 偏好学习阈�?
	// 用户选择次数达到此阈值后才启用偏好排序，默认3
	PreferenceThreshold int `json:"preference_threshold"`

	// MaxExclusionSummary 排除摘要最大长�?
	// 排除项摘要的最大字符数，默�?00
	MaxExclusionSummary int `json:"max_exclusion_summary"`
}

// Default values for IntentUnderstandingConfig
const (
	DefaultMaxSuggestions      = 5
	DefaultMaxHistoryRecords   = 5
	DefaultPreferenceThreshold = 3
	DefaultMaxExclusionSummary = 300
)

// NewDefaultIntentUnderstandingConfig 返回默认的意图理解配�?
// 默认情况下启用意图理解，使用推荐的参数�?
func NewDefaultIntentUnderstandingConfig() *IntentUnderstandingConfig {
	return &IntentUnderstandingConfig{
		Enabled:             true,
		MaxSuggestions:      DefaultMaxSuggestions,
		MaxHistoryRecords:   DefaultMaxHistoryRecords,
		PreferenceThreshold: DefaultPreferenceThreshold,
		MaxExclusionSummary: DefaultMaxExclusionSummary,
	}
}

// NewDisabledIntentUnderstandingConfig 返回禁用意图理解的配�?
// 用于向后兼容或降级场�?
func NewDisabledIntentUnderstandingConfig() *IntentUnderstandingConfig {
	config := NewDefaultIntentUnderstandingConfig()
	config.Enabled = false
	return config
}

// Validate 验证配置参数的有效�?
// 如果参数无效，将其设置为默认�?
// 返回是否有参数被修正
func (c *IntentUnderstandingConfig) Validate() bool {
	modified := false

	// 验证最大建议数�?(1-10)
	if c.MaxSuggestions < 1 {
		c.MaxSuggestions = DefaultMaxSuggestions
		modified = true
	} else if c.MaxSuggestions > 10 {
		c.MaxSuggestions = 10
		modified = true
	}

	// 验证最大历史记录数 (1-20)
	if c.MaxHistoryRecords < 1 {
		c.MaxHistoryRecords = DefaultMaxHistoryRecords
		modified = true
	} else if c.MaxHistoryRecords > 20 {
		c.MaxHistoryRecords = 20
		modified = true
	}

	// 验证偏好学习阈�?(1-10)
	if c.PreferenceThreshold < 1 {
		c.PreferenceThreshold = DefaultPreferenceThreshold
		modified = true
	} else if c.PreferenceThreshold > 10 {
		c.PreferenceThreshold = 10
		modified = true
	}

	// 验证排除摘要最大长�?(50-1000)
	if c.MaxExclusionSummary < 50 {
		c.MaxExclusionSummary = DefaultMaxExclusionSummary
		modified = true
	} else if c.MaxExclusionSummary > 1000 {
		c.MaxExclusionSummary = 1000
		modified = true
	}

	return modified
}

// Clone 创建配置的深拷贝
func (c *IntentUnderstandingConfig) Clone() *IntentUnderstandingConfig {
	return &IntentUnderstandingConfig{
		Enabled:             c.Enabled,
		MaxSuggestions:      c.MaxSuggestions,
		MaxHistoryRecords:   c.MaxHistoryRecords,
		PreferenceThreshold: c.PreferenceThreshold,
		MaxExclusionSummary: c.MaxExclusionSummary,
	}
}

// IntentUnderstandingConfigManager 配置管理�?
// 负责配置的加载、保存和线程安全访问
type IntentUnderstandingConfigManager struct {
	dataDir string
	config  *IntentUnderstandingConfig
	mu      sync.RWMutex
}

// NewIntentUnderstandingConfigManager 创建配置管理�?
func NewIntentUnderstandingConfigManager(dataDir string) *IntentUnderstandingConfigManager {
	manager := &IntentUnderstandingConfigManager{
		dataDir: dataDir,
		config:  NewDefaultIntentUnderstandingConfig(),
	}

	// 尝试加载已有配置
	if err := manager.Load(); err != nil {
		// 加载失败时使用默认配置，不返回错�?
		manager.config = NewDefaultIntentUnderstandingConfig()
	}

	return manager
}

// getConfigPath 获取配置文件路径
func (m *IntentUnderstandingConfigManager) getConfigPath() string {
	return filepath.Join(m.dataDir, "config", "intent_understanding_config.json")
}

// Load 从文件加载配�?
func (m *IntentUnderstandingConfigManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，使用默认配置
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config IntentUnderstandingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 验证并修正配�?
	config.Validate()
	m.config = &config

	return nil
}

// Save 保存配置到文�?
func (m *IntentUnderstandingConfigManager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saveInternal()
}

// saveInternal 内部保存方法（不加锁�?
func (m *IntentUnderstandingConfigManager) saveInternal() error {
	path := m.getConfigPath()
	dir := filepath.Dir(path)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig 获取当前配置的副�?
func (m *IntentUnderstandingConfigManager) GetConfig() *IntentUnderstandingConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.Clone()
}

// SetConfig 设置新配置并保存
func (m *IntentUnderstandingConfigManager) SetConfig(config *IntentUnderstandingConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证配置
	configCopy := config.Clone()
	configCopy.Validate()

	m.config = configCopy
	return m.saveInternal()
}

// IsEnabled 检查意图理解是否启�?
func (m *IntentUnderstandingConfigManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.Enabled
}

// SetEnabled 设置意图理解启用状�?
func (m *IntentUnderstandingConfigManager) SetEnabled(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Enabled = enabled
	return m.saveInternal()
}
