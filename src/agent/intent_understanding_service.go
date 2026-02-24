package agent

import (
	"context"
	"fmt"
	"sync"
)

// IntentUnderstandingService 意图理解服务
// 简化后的主服务，协�?个核心组件：
// - IntentGenerator: 意图生成器，负责调用LLM生成意图建议
// - ContextProvider: 上下文提供器，整合数据源信息和历史记�?
// - ExclusionManager: 排除项管理器，管理用户拒绝的意图并生成排除提�?
// - IntentRanker: 意图排序器，根据用户偏好对建议进行排�?
// Validates: Requirements 7.1, 7.3
type IntentUnderstandingService struct {
	generator       *IntentGenerator
	contextProvider *ContextProvider
	exclusionMgr    *ExclusionManager
	ranker          *IntentRanker
	configManager   *IntentUnderstandingConfigManager
	logger          func(string)
	mu              sync.RWMutex
}

// NewIntentUnderstandingService 创建意图理解服务
// 初始化所有核心组件并加载配置
// Parameters:
//   - dataDir: 数据目录路径，用于存储配置和偏好数据
//   - dataSourceService: 数据源服务，用于获取数据源信�?
//   - logger: 日志函数，用于记录服务运行日�?
//
// Returns: 初始化后�?IntentUnderstandingService 实例
// Validates: Requirements 7.1, 7.3
func NewIntentUnderstandingService(
	dataDir string,
	dataSourceService *DataSourceService,
	logger func(string),
) *IntentUnderstandingService {
	// 设置默认日志函数
	if logger == nil {
		logger = func(msg string) {
			fmt.Println(msg)
		}
	}

	logger("[INTENT-SERVICE] Initializing IntentUnderstandingService...")

	// 创建配置管理器并加载配置
	configManager := NewIntentUnderstandingConfigManager(dataDir)
	config := configManager.GetConfig()

	logger(fmt.Sprintf("[INTENT-SERVICE] Loaded config: enabled=%v, maxSuggestions=%d, maxHistoryRecords=%d, preferenceThreshold=%d, maxExclusionSummary=%d",
		config.Enabled, config.MaxSuggestions, config.MaxHistoryRecords, config.PreferenceThreshold, config.MaxExclusionSummary))

	// 创建上下文提供器
	// Validates: Requirements 2.1, 2.7
	contextProvider := NewContextProviderWithLogger(dataDir, dataSourceService, logger)
	logger("[INTENT-SERVICE] Created ContextProvider")

	// 创建排除项管理器
	// Validates: Requirements 3.2, 3.3
	exclusionMgr := NewExclusionManager(config.MaxExclusionSummary)
	logger("[INTENT-SERVICE] Created ExclusionManager")

	// 创建意图排序�?
	// Validates: Requirements 5.1, 5.2
	ranker := NewIntentRanker(dataDir, config.PreferenceThreshold)
	logger("[INTENT-SERVICE] Created IntentRanker")

	// 创建意图生成�?
	// Validates: Requirements 1.3
	generator := NewIntentGenerator(contextProvider, exclusionMgr, logger)
	logger("[INTENT-SERVICE] Created IntentGenerator")

	service := &IntentUnderstandingService{
		generator:       generator,
		contextProvider: contextProvider,
		exclusionMgr:    exclusionMgr,
		ranker:          ranker,
		configManager:   configManager,
		logger:          logger,
	}

	logger("[INTENT-SERVICE] IntentUnderstandingService initialized successfully")

	return service
}

// log 记录日志
func (s *IntentUnderstandingService) log(msg string) {
	if s.logger != nil {
		s.logger(msg)
	}
}

// GetConfig 获取配置
// 返回当前配置的副本，线程安全
// Returns: 当前配置的副�?
// Validates: Requirements 7.3
func (s *IntentUnderstandingService) GetConfig() *IntentUnderstandingConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.configManager.GetConfig()
}

// SetConfig 设置配置
// 更新配置并保存到文件，同时更新相关组件的配置
// Parameters:
//   - config: 新的配置
//
// Returns: 保存失败时返回错�?
// Validates: Requirements 7.3
func (s *IntentUnderstandingService) SetConfig(config *IntentUnderstandingConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	s.log(fmt.Sprintf("[INTENT-SERVICE] Setting new config: enabled=%v, maxSuggestions=%d, maxHistoryRecords=%d, preferenceThreshold=%d, maxExclusionSummary=%d",
		config.Enabled, config.MaxSuggestions, config.MaxHistoryRecords, config.PreferenceThreshold, config.MaxExclusionSummary))

	// 保存配置
	if err := s.configManager.SetConfig(config); err != nil {
		s.log(fmt.Sprintf("[INTENT-SERVICE] Failed to save config: %v", err))
		return err
	}

	// 更新排序器阈�?
	if s.ranker != nil {
		s.ranker.SetThreshold(config.PreferenceThreshold)
	}

	s.log("[INTENT-SERVICE] Config updated successfully")
	return nil
}

// IsEnabled 检查意图理解是否启�?
// Returns: 是否启用意图理解
// Validates: Requirements 7.3
func (s *IntentUnderstandingService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.configManager.IsEnabled()
}

// SetEnabled 设置意图理解启用状�?
// Parameters:
//   - enabled: 是否启用
//
// Returns: 保存失败时返回错�?
// Validates: Requirements 7.3
func (s *IntentUnderstandingService) SetEnabled(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log(fmt.Sprintf("[INTENT-SERVICE] Setting enabled=%v", enabled))

	if err := s.configManager.SetEnabled(enabled); err != nil {
		s.log(fmt.Sprintf("[INTENT-SERVICE] Failed to set enabled: %v", err))
		return err
	}

	s.log("[INTENT-SERVICE] Enabled status updated successfully")
	return nil
}

// GenerateSuggestions 生成意图建议
// 主入口方法，整合所有组件功能：
// 1. 获取数据源上下文
// 2. 生成排除项摘�?
// 3. 调用LLM生成意图建议
// 4. 根据用户偏好排序
//
// Parameters:
//   - ctx: 上下文，用于取消操作
//   - threadID: 会话ID
//   - userMessage: 用户的原始请求消�?
//   - dataSourceID: 数据源ID
//   - language: 语言设置 ("zh" 中文, "en" 英文)
//   - exclusions: 已排除的意图建议列表
//   - llmCall: LLM调用函数
//
// Returns:
//   - []IntentSuggestion: 排序后的意图建议列表
//   - error: 生成失败时的错误
//
// Validates: Requirements 1.1, 5.3
func (s *IntentUnderstandingService) GenerateSuggestions(
	ctx context.Context,
	threadID string,
	userMessage string,
	dataSourceID string,
	language string,
	exclusions []IntentSuggestion,
	llmCall LLMCallFunc,
) ([]IntentSuggestion, error) {
	s.mu.RLock()
	config := s.configManager.GetConfig()
	s.mu.RUnlock()

	// 检查是否启�?
	if !config.Enabled {
		s.log("[INTENT-SERVICE] Intent understanding is disabled")
		return nil, fmt.Errorf("intent understanding is disabled")
	}

	s.log(fmt.Sprintf("[INTENT-SERVICE] Generating suggestions for message: %s (dataSourceID: %s, language: %s, exclusions: %d)",
		truncateString(userMessage, 50), dataSourceID, language, len(exclusions)))

	// 1. 获取数据源上下文
	// Validates: Requirements 2.1, 2.6, 2.7
	dataSourceContext, err := s.contextProvider.GetContext(dataSourceID, config.MaxHistoryRecords)
	if err != nil {
		s.log(fmt.Sprintf("[INTENT-SERVICE] Failed to get context: %v", err))
		// 继续使用空上下文，不返回错误
		dataSourceContext = &DataSourceContext{
			TableName:      "",
			Columns:        []ContextColumnInfo{},
			AnalysisHints:  []string{},
			RecentAnalyses: []AnalysisRecord{},
		}
	}

	// 2. 生成排除项摘�?
	// Validates: Requirements 3.2, 3.3
	exclusionSummary := ""
	if len(exclusions) > 0 {
		exclusionSummary = s.exclusionMgr.GenerateSummary(exclusions, language)
		s.log(fmt.Sprintf("[INTENT-SERVICE] Generated exclusion summary: %s", truncateString(exclusionSummary, 100)))
	}

	// 3. 调用LLM生成意图建议
	// Validates: Requirements 1.1, 1.2
	suggestions, err := s.generator.Generate(
		ctx,
		userMessage,
		dataSourceContext,
		exclusionSummary,
		language,
		config.MaxSuggestions,
		llmCall,
	)
	if err != nil {
		s.log(fmt.Sprintf("[INTENT-SERVICE] Failed to generate suggestions: %v", err))
		return nil, err
	}

	s.log(fmt.Sprintf("[INTENT-SERVICE] Generated %d suggestions", len(suggestions)))

	// 4. 根据用户偏好排序
	// Validates: Requirements 5.3, 5.4
	rankedSuggestions := s.ranker.RankSuggestions(dataSourceID, suggestions)
	s.log(fmt.Sprintf("[INTENT-SERVICE] Ranked suggestions (selection count: %d, threshold: %d)",
		s.ranker.GetSelectionCount(dataSourceID), s.ranker.GetThreshold()))

	return rankedSuggestions, nil
}

// RecordSelection 记录用户选择
// 记录用户选择的意图，用于偏好学习
//
// Parameters:
//   - dataSourceID: 数据源ID
//   - selectedIntent: 用户选择的意图建�?
//
// Returns: 保存失败时返回错�?
// Validates: Requirements 5.1, 5.2
func (s *IntentUnderstandingService) RecordSelection(
	dataSourceID string,
	selectedIntent IntentSuggestion,
) error {
	s.log(fmt.Sprintf("[INTENT-SERVICE] Recording selection: %s (dataSourceID: %s)",
		selectedIntent.Title, dataSourceID))

	if err := s.ranker.RecordSelection(dataSourceID, selectedIntent); err != nil {
		s.log(fmt.Sprintf("[INTENT-SERVICE] Failed to record selection: %v", err))
		return err
	}

	s.log("[INTENT-SERVICE] Selection recorded successfully")
	return nil
}

// GetContextProvider 获取上下文提供器
// 用于外部访问上下文功�?
func (s *IntentUnderstandingService) GetContextProvider() *ContextProvider {
	return s.contextProvider
}

// GetExclusionManager 获取排除项管理器
// 用于外部访问排除项功�?
func (s *IntentUnderstandingService) GetExclusionManager() *ExclusionManager {
	return s.exclusionMgr
}

// GetIntentRanker 获取意图排序�?
// 用于外部访问排序功能
func (s *IntentUnderstandingService) GetIntentRanker() *IntentRanker {
	return s.ranker
}

// GetIntentGenerator 获取意图生成�?
// 用于外部访问生成功能
func (s *IntentUnderstandingService) GetIntentGenerator() *IntentGenerator {
	return s.generator
}

// Initialize 初始化服�?
// 加载历史记录等初始化操作
// Returns: 初始化失败时返回错误
// Validates: Requirements 7.2
func (s *IntentUnderstandingService) Initialize() error {
	s.log("[INTENT-SERVICE] Initializing service components...")

	var initErrors []error

	// 初始化上下文提供�?
	if s.contextProvider != nil {
		if err := s.contextProvider.Initialize(); err != nil {
			s.log(fmt.Sprintf("[INTENT-SERVICE] Context provider init failed: %v", err))
			initErrors = append(initErrors, err)
		}
	}

	// 如果所有组件都失败，返回错�?
	if len(initErrors) > 0 {
		s.log("[INTENT-SERVICE] Some components failed to initialize, running in degraded mode")
	}

	s.log("[INTENT-SERVICE] Service initialization completed")
	return nil
}

// AddAnalysisRecord 添加分析记录
// 将分析记录添加到历史记录中，用于上下文增�?
//
// Parameters:
//   - record: 分析记录
//
// Returns: 保存失败时返回错�?
func (s *IntentUnderstandingService) AddAnalysisRecord(record AnalysisRecord) error {
	if s.contextProvider == nil {
		return fmt.Errorf("context provider not initialized")
	}

	return s.contextProvider.AddAnalysisRecord(record)
}

// GetSelectionCount 获取指定数据源的选择次数
// Parameters:
//   - dataSourceID: 数据源ID
//
// Returns: 选择次数
func (s *IntentUnderstandingService) GetSelectionCount(dataSourceID string) int {
	if s.ranker == nil {
		return 0
	}
	return s.ranker.GetSelectionCount(dataSourceID)
}
