package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"vantagics/config"
)

// ConfigProvider 定义配置读取接口
type ConfigProvider interface {
	GetConfig() (config.Config, error)
	GetEffectiveConfig() (config.Config, error)
}

// ConfigPersister 定义配置持久化接�?
type ConfigPersister interface {
	SaveConfig(cfg config.Config) error
}

// ConfigNotifier 定义配置变更通知接口
type ConfigNotifier interface {
	OnConfigChanged(callback func(config.Config))
}

// ConfigService 封装所有配置管理逻辑
// 实现 Service, ConfigProvider, ConfigPersister, ConfigNotifier 接口
type ConfigService struct {
	storageDir string
	logger     func(string)
	callbacks  []func(config.Config)
	mu         sync.RWMutex
}

// NewConfigService 创建新的 ConfigService 实例
func NewConfigService(logger func(string)) *ConfigService {
	return &ConfigService{
		logger:    logger,
		callbacks: make([]func(config.Config), 0),
	}
}

// Name 返回服务名称
func (cs *ConfigService) Name() string {
	return "config"
}

// Initialize 初始化配置服务，确保存储目录存在
func (cs *ConfigService) Initialize(ctx context.Context) error {
	dir, err := cs.GetStorageDir()
	if err != nil {
		return WrapError("config", "Initialize", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return WrapError("config", "Initialize", fmt.Errorf("failed to create storage dir: %w", err))
	}
	cs.log(fmt.Sprintf("ConfigService initialized, storage dir: %s", dir))
	return nil
}

// Shutdown 关闭配置服务（无操作�?
func (cs *ConfigService) Shutdown() error {
	return nil
}

// GetStorageDir 返回存储目录路径（~/Vantagics�?
func (cs *ConfigService) GetStorageDir() (string, error) {
	cs.mu.RLock()
	sd := cs.storageDir
	cs.mu.RUnlock()

	if sd != "" {
		return sd, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", WrapError("config", "GetStorageDir", err)
	}
	return filepath.Join(home, "Vantagics"), nil
}

// SetStorageDir 设置自定义存储目录（主要用于测试�?
func (cs *ConfigService) SetStorageDir(dir string) {
	cs.mu.Lock()
	cs.storageDir = dir
	cs.mu.Unlock()
}

// GetConfigPath 返回配置文件路径
func (cs *ConfigService) GetConfigPath() (string, error) {
	dir, err := cs.GetStorageDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// GetConfig 从磁盘加载配置文�?
func (cs *ConfigService) GetConfig() (config.Config, error) {
	path, err := cs.GetConfigPath()
	if err != nil {
		return config.Config{}, err
	}

	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, "Vantagics")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		defaultCfg := config.Config{
			LLMProvider:       "OpenAI",
			ModelName:         "gpt-4o",
			MaxTokens:         8192,
			LocalCache:        true,
			Language:          "English",
			DataCacheDir:      defaultDataDir,
			MaxPreviewRows:    100,
			IntentEnhancement: config.DefaultIntentEnhancementConfig(),
		}
		return defaultCfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return config.Config{}, WrapError("config", "GetConfig", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.Config{}, WrapError("config", "GetConfig", err)
	}

	// Apply defaults for empty fields
	if cfg.DataCacheDir == "" {
		cfg.DataCacheDir = defaultDataDir
	}
	if cfg.MaxPreviewRows <= 0 {
		cfg.MaxPreviewRows = 100
	}

	// Initialize IntentEnhancement with defaults if nil (backward compatibility)
	if cfg.IntentEnhancement == nil {
		cfg.IntentEnhancement = config.DefaultIntentEnhancementConfig()
	} else {
		cfg.IntentEnhancement.Validate()
	}

	return cfg, nil
}

// GetEffectiveConfig 返回配置（基础实现，不含许可证合并逻辑�?
// 许可�?LLM 配置的合并由 App 门面层处�?
func (cs *ConfigService) GetEffectiveConfig() (config.Config, error) {
	return cs.GetConfig()
}

// SaveConfig 保存配置到磁盘，调用 Validate()，并触发所有回�?
func (cs *ConfigService) SaveConfig(cfg config.Config) error {
	// Initialize MCPServices if nil
	if cfg.MCPServices == nil {
		cfg.MCPServices = []config.MCPService{}
	}

	// Validate DataCacheDir exists if set
	if cfg.DataCacheDir != "" {
		info, err := os.Stat(cfg.DataCacheDir)
		if err != nil {
			if os.IsNotExist(err) {
				return WrapError("config", "SaveConfig", fmt.Errorf("data cache directory does not exist: %s", cfg.DataCacheDir))
			}
			return WrapError("config", "SaveConfig", err)
		}
		if !info.IsDir() {
			return WrapError("config", "SaveConfig", fmt.Errorf("data cache path is not a directory: %s", cfg.DataCacheDir))
		}
	}

	dir, err := cs.GetStorageDir()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return WrapError("config", "SaveConfig", fmt.Errorf("failed to create storage dir: %w", err))
	}

	// Validate config before saving
	cfg.Validate()

	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return WrapError("config", "SaveConfig", fmt.Errorf("failed to marshal config: %w", err))
	}

	// Save with restricted permissions (0600: owner-only read/write since it contains API keys)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return WrapError("config", "SaveConfig", fmt.Errorf("failed to write config file: %w", err))
	}

	cs.log("Configuration saved to disk")

	// Trigger all registered callbacks
	cs.NotifyConfigChanged(cfg)

	return nil
}

// OnConfigChanged 注册配置变更回调函数
func (cs *ConfigService) OnConfigChanged(callback func(config.Config)) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.callbacks = append(cs.callbacks, callback)
}

// NotifyConfigChanged 触发所有已注册的配置变更回�?
func (cs *ConfigService) NotifyConfigChanged(cfg config.Config) {
	cs.mu.RLock()
	cbs := make([]func(config.Config), len(cs.callbacks))
	copy(cbs, cs.callbacks)
	cs.mu.RUnlock()

	for _, cb := range cbs {
		cb(cfg)
	}
}

// log 内部日志辅助方法
func (cs *ConfigService) log(msg string) {
	if cs.logger != nil {
		cs.logger(msg)
	}
}
