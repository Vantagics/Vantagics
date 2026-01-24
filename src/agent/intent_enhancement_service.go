package agent

import (
	"context"
	"fmt"
	"sync"
)

// IntentSuggestion represents a possible interpretation of user's intent
// This is a local copy to avoid circular imports with the main app package
// 保持与现有结构完全一致，确保向后兼容
// Validates: Requirements 7.4
type IntentSuggestion struct {
	ID          string `json:"id"`          // Unique identifier
	Title       string `json:"title"`       // Short title (10 chars max)
	Description string `json:"description"` // Detailed description (30 chars max)
	Icon        string `json:"icon"`        // Icon (emoji or icon name)
	Query       string `json:"query"`       // Actual query/analysis request to execute
}

// IsValid 检查意图建议是否有效
// 验证所有必需字段都非空
// Returns true if all required fields (ID, Title, Description, Icon, Query) are non-empty
// Validates: Requirements 1.2 (意图建议结构完整性)
func (s *IntentSuggestion) IsValid() bool {
	return s.ID != "" &&
		s.Title != "" &&
		s.Description != "" &&
		s.Icon != "" &&
		s.Query != ""
}

// Clone 创建意图建议的深拷贝
// Returns a new IntentSuggestion with the same values
// Useful for avoiding unintended modifications to the original
func (s *IntentSuggestion) Clone() *IntentSuggestion {
	if s == nil {
		return nil
	}
	return &IntentSuggestion{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		Icon:        s.Icon,
		Query:       s.Query,
	}
}

// String 返回意图建议的字符串表示
// Format: "[Icon] Title: Description (ID: xxx)"
// Useful for logging and debugging
func (s *IntentSuggestion) String() string {
	if s == nil {
		return "<nil IntentSuggestion>"
	}
	return fmt.Sprintf("[%s] %s: %s (ID: %s)", s.Icon, s.Title, s.Description, s.ID)
}

// ContextEnhancer 上下文增强器
// Responsible for collecting and integrating historical analysis records
// Validates: Requirements 1.1, 1.2, 1.4
type ContextEnhancer struct {
	memoryService *MemoryService
	historyStore  *AnalysisHistoryStore
	dataDir       string
	mu            sync.RWMutex
	initialized   bool
}

// NewContextEnhancer 创建上下文增强器
// Parameters:
//   - dataDir: the directory where analysis history will be stored
//   - memoryService: optional memory service for additional context (can be nil)
//
// Returns a new ContextEnhancer instance
func NewContextEnhancer(dataDir string, memoryService *MemoryService) *ContextEnhancer {
	return &ContextEnhancer{
		memoryService: memoryService,
		historyStore:  NewAnalysisHistoryStore(dataDir),
		dataDir:       dataDir,
		initialized:   false,
	}
}

// Initialize initializes the context enhancer by loading history from disk
// Returns error if loading fails
func (c *ContextEnhancer) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	if c.historyStore == nil {
		c.historyStore = NewAnalysisHistoryStore(c.dataDir)
	}

	if err := c.historyStore.Load(); err != nil {
		return fmt.Errorf("failed to load analysis history: %w", err)
	}

	c.initialized = true
	return nil
}

// GetHistoryContext 获取历史分析上下文
// Retrieves historical analysis records for a specific data source
// Parameters:
//   - dataSourceID: the ID of the data source to get history for
//   - maxRecords: maximum number of records to return (default 10 as per requirements)
//
// Returns records sorted by timestamp in descending order (newest first)
// Returns empty slice if no records exist or on error (graceful degradation)
// Validates: Requirements 1.2, 1.4
func (c *ContextEnhancer) GetHistoryContext(dataSourceID string, maxRecords int) []AnalysisRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return empty slice if not initialized
	if !c.initialized || c.historyStore == nil {
		return []AnalysisRecord{}
	}

	// Use default max records if not specified
	if maxRecords <= 0 {
		maxRecords = 10 // Default as per Requirements 1.2
	}

	// Get records from history store
	// The store already handles sorting by timestamp (newest first) and limiting
	records, err := c.historyStore.GetRecordsByDataSource(dataSourceID, maxRecords)
	if err != nil {
		// Graceful degradation: return empty slice on error
		return []AnalysisRecord{}
	}

	return records
}

// AddAnalysisRecord 添加分析记录
// Adds a new analysis record to the history store
// Parameters:
//   - record: the analysis record to add
//
// Returns error if the record cannot be added
// Validates: Requirements 1.1
func (c *ContextEnhancer) AddAnalysisRecord(record AnalysisRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize if not already done
	if !c.initialized {
		if c.historyStore == nil {
			c.historyStore = NewAnalysisHistoryStore(c.dataDir)
		}
		if err := c.historyStore.Load(); err != nil {
			return fmt.Errorf("failed to initialize history store: %w", err)
		}
		c.initialized = true
	}

	// Add record to history store
	// The store handles ID generation and timestamp setting if not provided
	return c.historyStore.AddRecord(record)
}

// GetRecordCount returns the total number of records for a data source
func (c *ContextEnhancer) GetRecordCount(dataSourceID string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.initialized || c.historyStore == nil {
		return 0
	}

	return c.historyStore.CountByDataSource(dataSourceID)
}

// IsInitialized returns whether the context enhancer has been initialized
func (c *ContextEnhancer) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// BuildContextSection 构建上下文提示词片段
// Builds a prompt section containing historical analysis information
// Parameters:
//   - records: the analysis records to include in the context
//   - language: the language for output ("zh" for Chinese, "en" for English)
//
// Returns a formatted prompt section string
// Returns empty string if records is empty
// Validates: Requirements 1.5, 8.1
func (c *ContextEnhancer) BuildContextSection(records []AnalysisRecord, language string) string {
	// Return empty string if no records
	if len(records) == 0 {
		return ""
	}

	// Build the prompt section based on language
	var result string

	if language == "zh" {
		result = c.buildChineseContextSection(records)
	} else {
		result = c.buildEnglishContextSection(records)
	}

	return result
}

// buildChineseContextSection builds the context section in Chinese
// Validates: Requirements 1.5, 8.1
func (c *ContextEnhancer) buildChineseContextSection(records []AnalysisRecord) string {
	var result string

	// Header
	result = "## 历史分析记录\n"
	result += "以下是用户在该数据源上的历史分析记录，请参考这些信息生成更相关的建议：\n\n"

	// Build each record entry
	for i, record := range records {
		// Record number and analysis type
		result += fmt.Sprintf("%d. %s\n", i+1, c.translateAnalysisType(record.AnalysisType, "zh"))

		// Target columns/dimensions
		if len(record.TargetColumns) > 0 {
			result += fmt.Sprintf("   - 目标维度: %s\n", c.formatColumns(record.TargetColumns))
		}

		// Key findings
		if record.KeyFindings != "" {
			result += fmt.Sprintf("   - 关键发现: %s\n", record.KeyFindings)
		}

		// Add blank line between records (except for the last one)
		if i < len(records)-1 {
			result += "\n"
		}
	}

	return result
}

// buildEnglishContextSection builds the context section in English
// Validates: Requirements 1.5, 8.1
func (c *ContextEnhancer) buildEnglishContextSection(records []AnalysisRecord) string {
	var result string

	// Header
	result = "## Historical Analysis Records\n"
	result += "The following are the user's historical analysis records on this data source. Please refer to this information to generate more relevant suggestions:\n\n"

	// Build each record entry
	for i, record := range records {
		// Record number and analysis type
		result += fmt.Sprintf("%d. %s\n", i+1, c.translateAnalysisType(record.AnalysisType, "en"))

		// Target columns/dimensions
		if len(record.TargetColumns) > 0 {
			result += fmt.Sprintf("   - Target Dimensions: %s\n", c.formatColumns(record.TargetColumns))
		}

		// Key findings
		if record.KeyFindings != "" {
			result += fmt.Sprintf("   - Key Findings: %s\n", record.KeyFindings)
		}

		// Add blank line between records (except for the last one)
		if i < len(records)-1 {
			result += "\n"
		}
	}

	return result
}

// translateAnalysisType translates the analysis type to the specified language
// Validates: Requirements 8.1
func (c *ContextEnhancer) translateAnalysisType(analysisType string, language string) string {
	// Chinese translations
	zhTranslations := map[string]string{
		"trend":        "趋势分析",
		"comparison":   "对比分析",
		"distribution": "分布分析",
		"correlation":  "相关性分析",
		"aggregation":  "聚合分析",
		"ranking":      "排名分析",
		"time_series":  "时间序列分析",
		"geographic":   "地理分析",
		"statistical":  "统计分析",
		"categorical":  "分类分析",
	}

	// English translations
	enTranslations := map[string]string{
		"trend":        "Trend Analysis",
		"comparison":   "Comparison Analysis",
		"distribution": "Distribution Analysis",
		"correlation":  "Correlation Analysis",
		"aggregation":  "Aggregation Analysis",
		"ranking":      "Ranking Analysis",
		"time_series":  "Time Series Analysis",
		"geographic":   "Geographic Analysis",
		"statistical":  "Statistical Analysis",
		"categorical":  "Categorical Analysis",
	}

	var translations map[string]string
	if language == "zh" {
		translations = zhTranslations
	} else {
		translations = enTranslations
	}

	// Return translated type if found, otherwise return original
	if translated, ok := translations[analysisType]; ok {
		return translated
	}

	// If not found in translations, return the original type
	// This handles custom analysis types
	return analysisType
}

// formatColumns formats a slice of column names into a comma-separated string
func (c *ContextEnhancer) formatColumns(columns []string) string {
	if len(columns) == 0 {
		return ""
	}

	result := columns[0]
	for i := 1; i < len(columns); i++ {
		result += ", " + columns[i]
	}
	return result
}

// DimensionAnalyzer 维度分析器 (placeholder - will be implemented in Task 3)
// Responsible for dynamically adjusting analysis dimensions based on data characteristics
type DimensionAnalyzer struct {
	initialized bool
}

// ExampleProvider 示例提供器 (placeholder - will be implemented in Task 5)
// Responsible for providing domain-specific Few-shot examples
type ExampleProvider struct {
	initialized bool
}

// Note: IntentCache is now implemented in intent_cache.go
// The full implementation includes:
// - Memory cache with LRU eviction
// - JSON persistence
// - Semantic similarity matching
// - Cache expiration

// IntentEnhancementService 意图增强服务
// Main service that coordinates all enhancement components
type IntentEnhancementService struct {
	contextEnhancer       *ContextEnhancer
	dimensionAnalyzer     *DimensionAnalyzer
	dimensionAnalyzerImpl *DimensionAnalyzerImpl // Full implementation for actual use
	exampleProvider       *ExampleProvider
	exampleProviderImpl   *ExampleProviderImpl // Full implementation for actual use
	intentCache           *IntentCache
	preferenceLearner     *PreferenceLearner
	memoryService         *MemoryService
	config                *IntentEnhancementConfig
	dataDir               string
	logger                func(string)
	mu                    sync.RWMutex

	// Component availability flags for graceful degradation
	contextEnhancerAvailable   bool
	dimensionAnalyzerAvailable bool
	exampleProviderAvailable   bool
	intentCacheAvailable       bool
	preferenceLearnerAvailable bool
}

// NewIntentEnhancementService 创建意图增强服务
// Creates a new intent enhancement service with the provided dependencies
func NewIntentEnhancementService(
	dataDir string,
	preferenceLearner *PreferenceLearner,
	memoryService *MemoryService,
	logger func(string),
) *IntentEnhancementService {
	// Use default logger if none provided
	if logger == nil {
		logger = func(msg string) {
			fmt.Println(msg)
		}
	}

	service := &IntentEnhancementService{
		preferenceLearner: preferenceLearner,
		memoryService:     memoryService,
		config:            DefaultIntentEnhancementConfig(),
		dataDir:           dataDir,
		logger:            logger,
	}

	return service
}

// Initialize 初始化所有增强组件
// Initializes all enhancement components with graceful degradation
// If a component fails to initialize, that feature is disabled and the service continues
// Returns error only if ALL components fail to initialize
func (s *IntentEnhancementService) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var initErrors []error
	componentsInitialized := 0

	// Initialize ContextEnhancer
	if s.config.EnableContextEnhancement {
		if err := s.initContextEnhancer(); err != nil {
			s.logger(fmt.Sprintf("[INTENT-ENHANCEMENT] Context enhancer init failed: %v", err))
			s.config.EnableContextEnhancement = false
			initErrors = append(initErrors, fmt.Errorf("context enhancer: %w", err))
		} else {
			s.contextEnhancerAvailable = true
			componentsInitialized++
			s.logger("[INTENT-ENHANCEMENT] Context enhancer initialized successfully")
		}
	}

	// Initialize DimensionAnalyzer
	if s.config.EnableDynamicDimensions {
		if err := s.initDimensionAnalyzer(); err != nil {
			s.logger(fmt.Sprintf("[INTENT-ENHANCEMENT] Dimension analyzer init failed: %v", err))
			s.config.EnableDynamicDimensions = false
			initErrors = append(initErrors, fmt.Errorf("dimension analyzer: %w", err))
		} else {
			s.dimensionAnalyzerAvailable = true
			componentsInitialized++
			s.logger("[INTENT-ENHANCEMENT] Dimension analyzer initialized successfully")
		}
	}

	// Initialize ExampleProvider
	if s.config.EnableFewShotExamples {
		if err := s.initExampleProvider(); err != nil {
			s.logger(fmt.Sprintf("[INTENT-ENHANCEMENT] Example provider init failed: %v", err))
			s.config.EnableFewShotExamples = false
			initErrors = append(initErrors, fmt.Errorf("example provider: %w", err))
		} else {
			s.exampleProviderAvailable = true
			componentsInitialized++
			s.logger("[INTENT-ENHANCEMENT] Example provider initialized successfully")
		}
	}

	// Initialize IntentCache
	if s.config.EnableCaching {
		if err := s.initIntentCache(); err != nil {
			s.logger(fmt.Sprintf("[INTENT-ENHANCEMENT] Intent cache init failed: %v", err))
			s.config.EnableCaching = false
			initErrors = append(initErrors, fmt.Errorf("intent cache: %w", err))
		} else {
			s.intentCacheAvailable = true
			componentsInitialized++
			s.logger("[INTENT-ENHANCEMENT] Intent cache initialized successfully")
		}
	}

	// Initialize PreferenceLearner (check if provided)
	if s.config.EnablePreferenceLearning {
		if s.preferenceLearner == nil {
			s.logger("[INTENT-ENHANCEMENT] Preference learner not provided, disabling preference learning")
			s.config.EnablePreferenceLearning = false
			initErrors = append(initErrors, fmt.Errorf("preference learner: not provided"))
		} else {
			s.preferenceLearnerAvailable = true
			componentsInitialized++
			s.logger("[INTENT-ENHANCEMENT] Preference learner initialized successfully")
		}
	}

	// If all components failed to initialize, return error
	if componentsInitialized == 0 && len(initErrors) > 0 {
		return fmt.Errorf("all enhancement components failed to initialize: %v", initErrors)
	}

	s.logger(fmt.Sprintf("[INTENT-ENHANCEMENT] Service initialized with %d/%d components available",
		componentsInitialized, 5))

	return nil
}

// initContextEnhancer initializes the context enhancer component
func (s *IntentEnhancementService) initContextEnhancer() error {
	s.contextEnhancer = NewContextEnhancer(s.dataDir, s.memoryService)
	return s.contextEnhancer.Initialize()
}

// initDimensionAnalyzer initializes the dimension analyzer component
func (s *IntentEnhancementService) initDimensionAnalyzer() error {
	s.dimensionAnalyzer = &DimensionAnalyzer{
		initialized: true,
	}
	// Also initialize the full implementation
	s.dimensionAnalyzerImpl = NewDimensionAnalyzer(nil) // DataSourceService can be nil for graceful degradation
	if err := s.dimensionAnalyzerImpl.Initialize(); err != nil {
		return err
	}
	return nil
}

// initExampleProvider initializes the example provider component
func (s *IntentEnhancementService) initExampleProvider() error {
	s.exampleProvider = &ExampleProvider{
		initialized: true,
	}
	// Also initialize the full implementation
	s.exampleProviderImpl = NewExampleProviderImpl()
	return nil
}

// initIntentCache initializes the intent cache component
func (s *IntentEnhancementService) initIntentCache() error {
	s.intentCache = NewIntentCacheWithDataDir(
		s.config.MaxCacheEntries,
		s.config.CacheExpirationHours,
		s.config.CacheSimilarityThreshold,
		s.dataDir,
	)
	return s.intentCache.Initialize()
}

// SetConfig 设置配置
// Updates the service configuration
func (s *IntentEnhancementService) SetConfig(config *IntentEnhancementConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config != nil {
		config.Validate()
		s.config = config
	}
}

// GetConfig 获取当前配置
// Returns a copy of the current configuration
func (s *IntentEnhancementService) GetConfig() *IntentEnhancementConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config.Clone()
}

// IsAvailable 检查服务是否可用
// Returns true if at least one enhancement component is available
func (s *IntentEnhancementService) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.contextEnhancerAvailable ||
		s.dimensionAnalyzerAvailable ||
		s.exampleProviderAvailable ||
		s.intentCacheAvailable ||
		s.preferenceLearnerAvailable
}

// IsAllDisabled 检查是否所有增强功能都已禁用
// Returns true if all enhancement features are disabled
func (s *IntentEnhancementService) IsAllDisabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config.IsAllDisabled()
}


// EnhancePrompt 增强意图理解提示词
// Enhances the base prompt with context, dimensions, and examples
// Parameters:
//   - ctx: context for cancellation
//   - basePrompt: the original prompt to enhance
//   - dataSourceID: the ID of the data source being analyzed
//   - userMessage: the user's original message/request
//   - language: the language for output ("zh" for Chinese, "en" for English)
//
// Returns the enhanced prompt string and any error
// If all enhancements are disabled, returns the base prompt unchanged
// Validates: Requirements 1.5, 3.6, 4.1
func (s *IntentEnhancementService) EnhancePrompt(
	ctx context.Context,
	basePrompt string,
	dataSourceID string,
	userMessage string,
	language string,
) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If all enhancements are disabled, return base prompt unchanged (backward compatibility)
	if s.config.IsAllDisabled() {
		return basePrompt, nil
	}

	enhancedPrompt := basePrompt
	var enhancementSections []string

	// Add context enhancement if enabled and available
	// Validates: Requirements 1.5 - Include historical analysis type, target dimensions, and key findings
	if s.config.EnableContextEnhancement && s.contextEnhancerAvailable && s.contextEnhancer != nil {
		// Get historical context records
		maxRecords := s.config.MaxHistoryRecords
		if maxRecords <= 0 {
			maxRecords = 10 // Default as per Requirements 1.2
		}
		records := s.contextEnhancer.GetHistoryContext(dataSourceID, maxRecords)
		
		// Build context section if we have records
		if len(records) > 0 {
			contextSection := s.contextEnhancer.BuildContextSection(records, language)
			if contextSection != "" {
				enhancementSections = append(enhancementSections, contextSection)
				s.logger("[INTENT-ENHANCEMENT] Added context enhancement with " + formatIntForLog(len(records)) + " historical records")
			}
		} else {
			s.logger("[INTENT-ENHANCEMENT] No historical records found for data source: " + dataSourceID)
		}
	}

	// Add dimension recommendations if enabled and available
	// Validates: Requirements 3.6 - Sort dimension recommendations by relevance weight
	if s.config.EnableDynamicDimensions && s.dimensionAnalyzerAvailable && s.dimensionAnalyzerImpl != nil {
		// For now, we'll use column information if available
		// In a full implementation, this would get columns from the data source
		// The DimensionAnalyzerImpl.GetDimensionRecommendations already sorts by priority (Req 3.6)
		
		// Try to get dimension recommendations
		// Note: In production, you would pass actual column characteristics from the data source
		// For now, we'll check if we can analyze the data source
		characteristics, err := s.dimensionAnalyzerImpl.AnalyzeDataSource(dataSourceID)
		if err == nil && len(characteristics) > 0 {
			// Get recommendations sorted by priority (Validates Req 3.6)
			recommendations := s.dimensionAnalyzerImpl.GetDimensionRecommendations(characteristics)
			if len(recommendations) > 0 {
				dimensionSection := s.dimensionAnalyzerImpl.BuildDimensionSection(recommendations, language)
				if dimensionSection != "" {
					enhancementSections = append(enhancementSections, dimensionSection)
					s.logger("[INTENT-ENHANCEMENT] Added dimension recommendations with " + formatIntForLog(len(recommendations)) + " dimensions")
				}
			}
		} else {
			s.logger("[INTENT-ENHANCEMENT] Could not analyze data source dimensions: " + dataSourceID)
		}
	}

	// Add few-shot examples if enabled and available
	// Validates: Requirements 4.1 - Include 2-3 high-quality Few-shot examples
	if s.config.EnableFewShotExamples && s.exampleProviderAvailable && s.exampleProviderImpl != nil {
		// Detect domain from user message (simplified - in production would use column info)
		// For now, use general domain as we don't have column info in this context
		domain := DomainGeneral
		
		// Get 2-3 examples as per Requirements 4.1
		exampleCount := 3 // Default to 3 examples
		examples := s.exampleProviderImpl.GetExamples(domain, language, exampleCount)
		
		if len(examples) > 0 {
			exampleSection := s.exampleProviderImpl.BuildExampleSection(examples, language)
			if exampleSection != "" {
				enhancementSections = append(enhancementSections, exampleSection)
				s.logger("[INTENT-ENHANCEMENT] Added " + formatIntForLog(len(examples)) + " few-shot examples for domain: " + domain)
			}
		} else {
			s.logger("[INTENT-ENHANCEMENT] No examples found for language: " + language)
		}
	}

	// Combine all enhancement sections with the base prompt
	if len(enhancementSections) > 0 {
		for _, section := range enhancementSections {
			enhancedPrompt = enhancedPrompt + "\n\n" + section
		}
		s.logger("[INTENT-ENHANCEMENT] Enhanced prompt with " + formatIntForLog(len(enhancementSections)) + " sections")
	}

	return enhancedPrompt, nil
}

// EnhancePromptWithColumns 增强意图理解提示词（带列信息）
// Enhanced version of EnhancePrompt that accepts column information for better dimension analysis
// Parameters:
//   - ctx: context for cancellation
//   - basePrompt: the original prompt to enhance
//   - dataSourceID: the ID of the data source being analyzed
//   - userMessage: the user's original message/request
//   - language: the language for output ("zh" for Chinese, "en" for English)
//   - columns: column schema information for dimension analysis
//   - tableName: name of the table for domain detection
//
// Returns the enhanced prompt string and any error
// Validates: Requirements 1.5, 3.6, 4.1
func (s *IntentEnhancementService) EnhancePromptWithColumns(
	ctx context.Context,
	basePrompt string,
	dataSourceID string,
	userMessage string,
	language string,
	columns []ColumnSchema,
	tableName string,
) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If all enhancements are disabled, return base prompt unchanged (backward compatibility)
	if s.config.IsAllDisabled() {
		return basePrompt, nil
	}

	enhancedPrompt := basePrompt
	var enhancementSections []string

	// Add context enhancement if enabled and available
	// Validates: Requirements 1.5 - Include historical analysis type, target dimensions, and key findings
	if s.config.EnableContextEnhancement && s.contextEnhancerAvailable && s.contextEnhancer != nil {
		maxRecords := s.config.MaxHistoryRecords
		if maxRecords <= 0 {
			maxRecords = 10
		}
		records := s.contextEnhancer.GetHistoryContext(dataSourceID, maxRecords)
		
		if len(records) > 0 {
			contextSection := s.contextEnhancer.BuildContextSection(records, language)
			if contextSection != "" {
				enhancementSections = append(enhancementSections, contextSection)
				s.logger("[INTENT-ENHANCEMENT] Added context enhancement with " + formatIntForLog(len(records)) + " historical records")
			}
		}
	}

	// Add dimension recommendations if enabled and available
	// Validates: Requirements 3.6 - Sort dimension recommendations by relevance weight
	if s.config.EnableDynamicDimensions && s.dimensionAnalyzerAvailable && s.dimensionAnalyzerImpl != nil && len(columns) > 0 {
		// Analyze columns directly
		characteristics := s.dimensionAnalyzerImpl.AnalyzeColumns(columns)
		if len(characteristics) > 0 {
			// Get recommendations sorted by priority (Validates Req 3.6)
			recommendations := s.dimensionAnalyzerImpl.GetDimensionRecommendations(characteristics)
			if len(recommendations) > 0 {
				dimensionSection := s.dimensionAnalyzerImpl.BuildDimensionSection(recommendations, language)
				if dimensionSection != "" {
					enhancementSections = append(enhancementSections, dimensionSection)
					s.logger("[INTENT-ENHANCEMENT] Added dimension recommendations with " + formatIntForLog(len(recommendations)) + " dimensions")
				}
			}
		}
	}

	// Add few-shot examples if enabled and available
	// Validates: Requirements 4.1 - Include 2-3 high-quality Few-shot examples
	if s.config.EnableFewShotExamples && s.exampleProviderAvailable && s.exampleProviderImpl != nil {
		// Detect domain from columns and table name
		columnNames := make([]string, len(columns))
		for i, col := range columns {
			columnNames[i] = col.Name
		}
		domain := s.exampleProviderImpl.DetectDomain(columnNames, tableName)
		
		// Get 2-3 examples as per Requirements 4.1
		exampleCount := 3
		examples := s.exampleProviderImpl.GetExamples(domain, language, exampleCount)
		
		if len(examples) > 0 {
			exampleSection := s.exampleProviderImpl.BuildExampleSection(examples, language)
			if exampleSection != "" {
				enhancementSections = append(enhancementSections, exampleSection)
				s.logger("[INTENT-ENHANCEMENT] Added " + formatIntForLog(len(examples)) + " few-shot examples for domain: " + domain)
			}
		}
	}

	// Combine all enhancement sections with the base prompt
	if len(enhancementSections) > 0 {
		for _, section := range enhancementSections {
			enhancedPrompt = enhancedPrompt + "\n\n" + section
		}
		s.logger("[INTENT-ENHANCEMENT] Enhanced prompt with " + formatIntForLog(len(enhancementSections)) + " sections")
	}

	return enhancedPrompt, nil
}

// formatIntForLog converts an integer to string for logging
func formatIntForLog(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatIntForLog(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// GetCachedSuggestions 获取缓存的建议
// Retrieves cached intent suggestions for a similar request
// Parameters:
//   - dataSourceID: the ID of the data source
//   - userMessage: the user's message to match against cache
//
// Returns the cached suggestions and a boolean indicating cache hit
// If caching is disabled or cache miss, returns nil and false
func (s *IntentEnhancementService) GetCachedSuggestions(
	dataSourceID string,
	userMessage string,
) ([]IntentSuggestion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If caching is disabled or not available, return cache miss
	if !s.config.EnableCaching || !s.intentCacheAvailable {
		return nil, false
	}

	// Use the intent cache to look up cached suggestions
	if s.intentCache != nil {
		return s.intentCache.Get(dataSourceID, userMessage)
	}

	return nil, false
}

// CacheSuggestions 缓存建议
// Stores intent suggestions in the cache for future similar requests
// Parameters:
//   - dataSourceID: the ID of the data source
//   - userMessage: the user's message as cache key
//   - suggestions: the suggestions to cache
//
// Does nothing if caching is disabled or not available
func (s *IntentEnhancementService) CacheSuggestions(
	dataSourceID string,
	userMessage string,
	suggestions []IntentSuggestion,
) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If caching is disabled or not available, do nothing
	if !s.config.EnableCaching || !s.intentCacheAvailable {
		return
	}

	// Use the intent cache to store suggestions
	if s.intentCache != nil {
		s.intentCache.Set(dataSourceID, userMessage, suggestions)
	}
}

// RankSuggestions 根据用户偏好重新排序建议
// Re-ranks intent suggestions based on user's historical preferences
// Parameters:
//   - dataSourceID: the ID of the data source (preferences are per-datasource)
//   - suggestions: the original suggestions from LLM
//
// Returns the re-ranked suggestions
// If preference learning is disabled or insufficient data, returns original order
// Validates: Requirements 2.3
func (s *IntentEnhancementService) RankSuggestions(
	dataSourceID string,
	suggestions []IntentSuggestion,
) []IntentSuggestion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If preference learning is disabled or not available, return original order
	if !s.config.EnablePreferenceLearning || !s.preferenceLearnerAvailable {
		return suggestions
	}

	// If no suggestions or only one, no need to rank
	if len(suggestions) <= 1 {
		return suggestions
	}

	// If preference learner is nil, return original order
	if s.preferenceLearner == nil {
		return suggestions
	}

	// Create a copy of suggestions to avoid modifying the original
	rankedSuggestions := make([]IntentSuggestion, len(suggestions))
	copy(rankedSuggestions, suggestions)

	// Calculate boost values for each suggestion
	type suggestionWithBoost struct {
		suggestion IntentSuggestion
		boost      float64
		origIndex  int // Original index for stable sorting
	}

	suggestionsWithBoost := make([]suggestionWithBoost, len(rankedSuggestions))
	hasBoost := false

	for i, suggestion := range rankedSuggestions {
		// Use the Title as the intent type for preference lookup
		intentType := suggestion.Title
		boost := s.preferenceLearner.GetIntentRankingBoost(dataSourceID, intentType)
		suggestionsWithBoost[i] = suggestionWithBoost{
			suggestion: suggestion,
			boost:      boost,
			origIndex:  i,
		}
		if boost > 0 {
			hasBoost = true
		}
	}

	// If no boost values (insufficient preference data), return original order
	// This implements Requirement 2.6 - graceful degradation when data is insufficient
	if !hasBoost {
		s.logger("[INTENT-ENHANCEMENT] No preference data available for ranking, using original order")
		return suggestions
	}

	// Sort by boost value (descending), with original index as tiebreaker for stability
	// Simple bubble sort is fine for small lists (typically 3-5 suggestions)
	for i := 0; i < len(suggestionsWithBoost); i++ {
		for j := i + 1; j < len(suggestionsWithBoost); j++ {
			// Sort by boost descending, then by original index ascending for stability
			if suggestionsWithBoost[j].boost > suggestionsWithBoost[i].boost ||
				(suggestionsWithBoost[j].boost == suggestionsWithBoost[i].boost &&
					suggestionsWithBoost[j].origIndex < suggestionsWithBoost[i].origIndex) {
				suggestionsWithBoost[i], suggestionsWithBoost[j] = suggestionsWithBoost[j], suggestionsWithBoost[i]
			}
		}
	}

	// Extract the sorted suggestions
	for i, swb := range suggestionsWithBoost {
		rankedSuggestions[i] = swb.suggestion
	}

	s.logger("[INTENT-ENHANCEMENT] Ranked " + formatIntForLog(len(rankedSuggestions)) + " suggestions based on user preferences")

	return rankedSuggestions
}

// RecordSelection 记录用户的意图选择
// Records the user's intent selection for preference learning
// Parameters:
//   - dataSourceID: the ID of the data source
//   - selectedIntent: the intent that the user selected
//
// Does nothing if preference learning is disabled or not available
// Validates: Requirements 2.1
func (s *IntentEnhancementService) RecordSelection(
	dataSourceID string,
	selectedIntent IntentSuggestion,
) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If preference learning is disabled or not available, do nothing
	if !s.config.EnablePreferenceLearning || !s.preferenceLearnerAvailable {
		return
	}

	// If preference learner is nil, do nothing
	if s.preferenceLearner == nil {
		return
	}

	// Track the intent selection using the preference learner
	// This delegates to PreferenceLearner.TrackIntentSelection which handles:
	// - Recording the selection (Requirement 2.1)
	// - Incrementing selection count (Requirement 2.2)
	// - Per-datasource tracking (Requirement 2.5)
	err := s.preferenceLearner.TrackIntentSelection(dataSourceID, selectedIntent)
	if err != nil {
		s.logger("[INTENT-ENHANCEMENT] Failed to record intent selection: " + err.Error())
		return
	}

	s.logger("[INTENT-ENHANCEMENT] Recorded intent selection: " + selectedIntent.Title + " for data source: " + dataSourceID)
}

// GetComponentStatus 获取组件状态
// Returns the availability status of all enhancement components
func (s *IntentEnhancementService) GetComponentStatus() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]bool{
		"context_enhancer":   s.contextEnhancerAvailable,
		"dimension_analyzer": s.dimensionAnalyzerAvailable,
		"example_provider":   s.exampleProviderAvailable,
		"intent_cache":       s.intentCacheAvailable,
		"preference_learner": s.preferenceLearnerAvailable,
	}
}

// AddAnalysisRecord 添加分析记录到历史存储
// Records a completed analysis for future context enhancement
// Parameters:
//   - record: the analysis record to add
//
// Returns error if the record cannot be added
// Does nothing if context enhancement is disabled or not available
// Validates: Requirements 1.1
func (s *IntentEnhancementService) AddAnalysisRecord(record AnalysisRecord) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If context enhancement is disabled or not available, do nothing
	if !s.config.EnableContextEnhancement || !s.contextEnhancerAvailable {
		s.logger("[INTENT-ENHANCEMENT] Context enhancement disabled, skipping analysis record")
		return nil
	}

	// If context enhancer is nil, return error
	if s.contextEnhancer == nil {
		return fmt.Errorf("context enhancer not initialized")
	}

	// Add the record to the context enhancer
	err := s.contextEnhancer.AddAnalysisRecord(record)
	if err != nil {
		s.logger("[INTENT-ENHANCEMENT] Failed to add analysis record: " + err.Error())
		return err
	}

	s.logger(fmt.Sprintf("[INTENT-ENHANCEMENT] Recorded analysis: type=%s, columns=%v, findings=%s",
		record.AnalysisType, record.TargetColumns, record.KeyFindings))

	return nil
}

// Shutdown 关闭服务
// Gracefully shuts down the service and releases resources
func (s *IntentEnhancementService) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger("[INTENT-ENHANCEMENT] Shutting down service...")

	// Reset availability flags
	s.contextEnhancerAvailable = false
	s.dimensionAnalyzerAvailable = false
	s.exampleProviderAvailable = false
	s.intentCacheAvailable = false
	s.preferenceLearnerAvailable = false

	// Clear component references
	s.contextEnhancer = nil
	s.dimensionAnalyzer = nil
	s.exampleProvider = nil
	s.intentCache = nil

	s.logger("[INTENT-ENHANCEMENT] Service shutdown complete")

	return nil
}
