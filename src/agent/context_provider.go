package agent

import (
	"fmt"
	"strings"
	"sync"
)

// ContextProvider 上下文提供器
// 整合数据源特征和历史分析记录，为意图生成提供上下文信息
// Validates: Requirements 2.1, 2.7
type ContextProvider struct {
	dataSourceService *DataSourceService
	historyStore      *AnalysisHistoryStore
	dataDir           string
	mu                sync.RWMutex
	logger            func(string)
}

// DataSourceContext 数据源上下文
// 包含数据源的所有相关信息，用于意图生成
// Validates: Requirements 2.1, 2.7
type DataSourceContext struct {
	TableName      string              `json:"table_name"`
	Columns        []ContextColumnInfo `json:"columns"`
	AnalysisHints  []string            `json:"analysis_hints"`   // 分析提示
	RecentAnalyses []AnalysisRecord    `json:"recent_analyses"`  // 最近分析记录
}

// ContextColumnInfo 上下文列信息
// 包含列的名称、类型和语义类型，用于意图理解上下文
// Validates: Requirements 2.1
type ContextColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	SemanticType string `json:"semantic_type"` // date, geographic, numeric, categorical, text
}

// NewContextProvider 创建上下文提供器
// Parameters:
//   - dataDir: 数据目录路径
//   - dataSourceService: 数据源服务
//
// Returns: 新的 ContextProvider 实例
// Validates: Requirements 2.1, 2.7
func NewContextProvider(
	dataDir string,
	dataSourceService *DataSourceService,
) *ContextProvider {
	return &ContextProvider{
		dataSourceService: dataSourceService,
		historyStore:      NewAnalysisHistoryStore(dataDir),
		dataDir:           dataDir,
		logger:            nil,
	}
}

// NewContextProviderWithLogger 创建带日志功能的上下文提供器
// Parameters:
//   - dataDir: 数据目录路径
//   - dataSourceService: 数据源服务
//   - logger: 日志函数
//
// Returns: 新的 ContextProvider 实例
func NewContextProviderWithLogger(
	dataDir string,
	dataSourceService *DataSourceService,
	logger func(string),
) *ContextProvider {
	return &ContextProvider{
		dataSourceService: dataSourceService,
		historyStore:      NewAnalysisHistoryStore(dataDir),
		dataDir:           dataDir,
		logger:            logger,
	}
}

// log 记录日志
func (c *ContextProvider) log(msg string) {
	if c.logger != nil {
		c.logger(msg)
	}
}

// GetContext 获取数据源上下文
// 收集表信息、列特征、历史记录
// Parameters:
//   - dataSourceID: 数据源ID
//   - maxHistoryRecords: 最大历史记录数
//
// Returns: 数据源上下文和错误
// Validates: Requirements 2.1, 2.6, 2.7
func (c *ContextProvider) GetContext(
	dataSourceID string,
	maxHistoryRecords int,
) (*DataSourceContext, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	context := &DataSourceContext{
		TableName:      "",
		Columns:        []ContextColumnInfo{},
		AnalysisHints:  []string{},
		RecentAnalyses: []AnalysisRecord{},
	}

	// 获取表信息
	if c.dataSourceService != nil {
		tables, err := c.dataSourceService.GetTables(dataSourceID)
		if err != nil {
			c.log(fmt.Sprintf("[CONTEXT-PROVIDER] Failed to get tables: %v", err))
			// 继续使用空上下文，不返回错误
		} else if len(tables) > 0 {
			// 使用第一个表作为主表
			context.TableName = tables[0]

			// 获取列信息
			columns, err := c.dataSourceService.GetTableColumns(dataSourceID, context.TableName)
			if err != nil {
				c.log(fmt.Sprintf("[CONTEXT-PROVIDER] Failed to get columns: %v", err))
			} else {
				context.Columns = c.analyzeColumns(columns)
				context.AnalysisHints = c.generateHints(context.Columns)
			}
		}
	}

	// 获取历史记录
	if c.historyStore != nil {
		history, err := c.historyStore.GetRecordsByDataSource(dataSourceID, maxHistoryRecords)
		if err != nil {
			c.log(fmt.Sprintf("[CONTEXT-PROVIDER] Failed to get history: %v", err))
		} else {
			context.RecentAnalyses = history
		}
	}

	return context, nil
}

// analyzeColumns 分析列信息，确定语义类型
// 复用 DimensionAnalyzer 的列类型识别逻辑
// Parameters:
//   - columns: 列结构信息列表
//
// Returns: 带语义类型的列信息列表
// Validates: Requirements 2.2, 2.3, 2.4, 2.5
func (c *ContextProvider) analyzeColumns(columns []ColumnSchema) []ContextColumnInfo {
	result := make([]ContextColumnInfo, 0, len(columns))

	for _, col := range columns {
		semanticType := c.identifySemanticType(col.Name, col.Type)
		result = append(result, ContextColumnInfo{
			Name:         col.Name,
			Type:         col.Type,
			SemanticType: semanticType,
		})
	}

	return result
}

// identifySemanticType 识别列的语义类型
// 基于列名和数据库类型判断语义类型
// Parameters:
//   - columnName: 列名
//   - dbType: 数据库类型
//
// Returns: 语义类型 (date, geographic, numeric, categorical, text)
// Validates: Requirements 2.2, 2.3, 2.4, 2.5
func (c *ContextProvider) identifySemanticType(columnName string, dbType string) string {
	// 复用 dimension_analyzer.go 中的关键词匹配逻辑
	if isDateColumn(columnName) {
		return "date"
	}

	if isGeographicColumn(columnName) {
		return "geographic"
	}

	if isNumericColumn(columnName) {
		return "numeric"
	}

	if isCategoricalColumn(columnName) {
		return "categorical"
	}

	// 根据数据库类型推断
	upperDBType := strings.ToUpper(dbType)

	if strings.Contains(upperDBType, "DATE") || strings.Contains(upperDBType, "TIME") ||
		strings.Contains(upperDBType, "TIMESTAMP") {
		return "date"
	}

	if strings.Contains(upperDBType, "INT") || strings.Contains(upperDBType, "REAL") ||
		strings.Contains(upperDBType, "FLOAT") || strings.Contains(upperDBType, "DOUBLE") ||
		strings.Contains(upperDBType, "DECIMAL") || strings.Contains(upperDBType, "NUMERIC") ||
		strings.Contains(upperDBType, "NUMBER") {
		return "numeric"
	}

	return "text"
}

// generateHints 生成分析提示
// 根据列的语义类型生成适合的分析提示
// Parameters:
//   - columns: 列信息列表
//
// Returns: 分析提示列表
// Validates: Requirements 2.2, 2.3, 2.4, 2.5
func (c *ContextProvider) generateHints(columns []ContextColumnInfo) []string {
	hints := []string{}

	hasDate := false
	hasGeographic := false
	hasNumeric := false
	hasCategorical := false

	for _, col := range columns {
		switch col.SemanticType {
		case "date":
			hasDate = true
		case "geographic":
			hasGeographic = true
		case "numeric":
			hasNumeric = true
		case "categorical":
			hasCategorical = true
		}
	}

	// 根据列类型生成分析提示
	// Validates: Requirements 2.2
	if hasDate {
		hints = append(hints, "适合时间序列分析（包含日期列）")
	}

	// Validates: Requirements 2.3
	if hasGeographic {
		hints = append(hints, "适合区域分析（包含地理位置列）")
	}

	// Validates: Requirements 2.4
	if hasNumeric {
		hints = append(hints, "适合统计分析（包含数值列）")
	}

	// Validates: Requirements 2.5
	if hasCategorical {
		hints = append(hints, "适合分组对比分析（包含分类列）")
	}

	return hints
}

// AddAnalysisRecord 添加分析记录
// Parameters:
//   - record: 分析记录
//
// Returns: 错误
func (c *ContextProvider) AddAnalysisRecord(record AnalysisRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.historyStore == nil {
		return fmt.Errorf("history store not initialized")
	}

	return c.historyStore.AddRecord(record)
}

// BuildContextSection 构建上下文提示词片段
// 将数据源上下文转换为LLM可理解的提示词格式
// Parameters:
//   - context: 数据源上下文
//   - language: 语言 ("zh" 中文, "en" 英文)
//
// Returns: 格式化的上下文提示词片段
// Validates: Requirements 2.7
func (c *ContextProvider) BuildContextSection(
	context *DataSourceContext,
	language string,
) string {
	if context == nil {
		return ""
	}

	var sb strings.Builder

	// 写入标题
	if language == "zh" {
		sb.WriteString("## 数据源上下文\n\n")
	} else {
		sb.WriteString("## Data Source Context\n\n")
	}

	// 写入表名
	if context.TableName != "" {
		if language == "zh" {
			sb.WriteString(fmt.Sprintf("**表名**: %s\n\n", context.TableName))
		} else {
			sb.WriteString(fmt.Sprintf("**Table Name**: %s\n\n", context.TableName))
		}
	}

	// 写入列信息
	if len(context.Columns) > 0 {
		if language == "zh" {
			sb.WriteString("**列信息**:\n")
		} else {
			sb.WriteString("**Column Information**:\n")
		}

		for _, col := range context.Columns {
			semanticLabel := c.getSemanticTypeLabel(col.SemanticType, language)
			sb.WriteString(fmt.Sprintf("- %s (%s, %s)\n", col.Name, col.Type, semanticLabel))
		}
		sb.WriteString("\n")
	}

	// 写入分析提示
	if len(context.AnalysisHints) > 0 {
		if language == "zh" {
			sb.WriteString("**分析建议**:\n")
		} else {
			sb.WriteString("**Analysis Suggestions**:\n")
		}

		for _, hint := range context.AnalysisHints {
			if language == "zh" {
				sb.WriteString(fmt.Sprintf("- %s\n", hint))
			} else {
				// 翻译提示为英文
				sb.WriteString(fmt.Sprintf("- %s\n", c.translateHint(hint)))
			}
		}
		sb.WriteString("\n")
	}

	// 写入最近分析记录
	if len(context.RecentAnalyses) > 0 {
		if language == "zh" {
			sb.WriteString("**最近分析记录**:\n")
		} else {
			sb.WriteString("**Recent Analysis Records**:\n")
		}

		for i, record := range context.RecentAnalyses {
			if i >= 5 { // 最多显示5条
				break
			}
			if language == "zh" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", record.AnalysisType, record.KeyFindings))
			} else {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", record.AnalysisType, record.KeyFindings))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// getSemanticTypeLabel 获取语义类型的显示标签
func (c *ContextProvider) getSemanticTypeLabel(semanticType string, language string) string {
	if language == "zh" {
		switch semanticType {
		case "date":
			return "日期"
		case "geographic":
			return "地理位置"
		case "numeric":
			return "数值"
		case "categorical":
			return "分类"
		case "text":
			return "文本"
		default:
			return semanticType
		}
	}

	// English
	switch semanticType {
	case "date":
		return "date"
	case "geographic":
		return "geographic"
	case "numeric":
		return "numeric"
	case "categorical":
		return "categorical"
	case "text":
		return "text"
	default:
		return semanticType
	}
}

// translateHint 将中文提示翻译为英文
func (c *ContextProvider) translateHint(hint string) string {
	translations := map[string]string{
		"适合时间序列分析（包含日期列）":   "Suitable for time series analysis (contains date columns)",
		"适合区域分析（包含地理位置列）":   "Suitable for regional analysis (contains geographic columns)",
		"适合统计分析（包含数值列）":     "Suitable for statistical analysis (contains numeric columns)",
		"适合分组对比分析（包含分类列）":   "Suitable for grouping and comparison analysis (contains categorical columns)",
	}

	if translated, ok := translations[hint]; ok {
		return translated
	}
	return hint
}

// GetHistoryStore 获取历史记录存储
// 用于外部访问历史记录功能
func (c *ContextProvider) GetHistoryStore() *AnalysisHistoryStore {
	return c.historyStore
}

// Initialize 初始化上下文提供器
// 加载历史记录等初始化操作
func (c *ContextProvider) Initialize() error {
	if c.historyStore != nil {
		return c.historyStore.Load()
	}
	return nil
}
