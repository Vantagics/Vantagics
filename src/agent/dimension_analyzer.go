package agent

import (
	"strings"
)

// DimensionAnalyzer 维度分析�?
// Responsible for dynamically adjusting analysis dimensions based on data characteristics
// Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6

// ColumnCharacteristics 列特�?
// Represents the characteristics of a column in a data source
// Used by DimensionAnalyzer to determine appropriate analysis dimensions
type ColumnCharacteristics struct {
	Name         string   `json:"name"`          // Column name
	DataType     string   `json:"data_type"`     // date, numeric, categorical, geographic, text
	SemanticType string   `json:"semantic_type"` // time, location, amount, count, category, etc.
	UniqueRatio  float64  `json:"unique_ratio"`  // Ratio of unique values (0.0 to 1.0)
	SampleValues []string `json:"sample_values"` // Sample values from the column
}

// DimensionRecommendation 维度推荐
// Represents a recommended analysis dimension based on data characteristics
// Used to guide intent suggestions toward appropriate analysis types
type DimensionRecommendation struct {
	DimensionType string   `json:"dimension_type"` // temporal, geographic, statistical, categorical
	Priority      int      `json:"priority"`       // 1-10, higher means more important
	Columns       []string `json:"columns"`        // Columns relevant to this dimension
	Rationale     string   `json:"rationale"`      // Explanation for the recommendation
}

// Column type constants
const (
	ColumnTypeDate        = "date"
	ColumnTypeNumeric     = "numeric"
	ColumnTypeCategorical = "categorical"
	ColumnTypeGeographic  = "geographic"
	ColumnTypeText        = "text"
)

// Semantic type constants
const (
	SemanticTypeTime     = "time"
	SemanticTypeLocation = "location"
	SemanticTypeAmount   = "amount"
	SemanticTypeCount    = "count"
	SemanticTypeCategory = "category"
	SemanticTypeUnknown  = "unknown"
)

// Keyword lists for column type identification
// These keywords are used to identify column types based on column names
// Supports both English and Chinese keywords
// Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5

// dateKeywords contains keywords that indicate a date/time column
var dateKeywords = []string{
	// English keywords
	"date", "time", "datetime", "timestamp", "day", "month", "year", "week",
	"created_at", "updated_at", "deleted_at", "created", "updated", "modified",
	"start_date", "end_date", "start_time", "end_time", "birth", "birthday",
	"expire", "expiry", "expiration", "due", "deadline", "schedule",
	"order_date", "ship_date", "delivery_date", "payment_date", "register_date",
	"login_time", "logout_time", "last_login", "first_login",
	// Chinese keywords
	"日期", "时间", "�?, "�?, "�?, "�?, "星期",
	"创建时间", "更新时间", "修改时间", "删除时间",
	"开始日�?, "结束日期", "开始时�?, "结束时间",
	"出生日期", "生日", "过期时间", "到期时间", "截止日期",
	"订单日期", "发货日期", "交付日期", "付款日期", "注册日期",
	"登录时间", "登出时间", "最后登�?, "首次登录",
	"年份", "月份", "季度", "时刻",
}

// geographicKeywords contains keywords that indicate a geographic/location column
var geographicKeywords = []string{
	// English keywords
	"province", "city", "region", "country", "state", "county", "district",
	"address", "location", "area", "zone", "territory", "place",
	"street", "road", "avenue", "building", "floor", "room",
	"zip", "zipcode", "postal", "postcode", "geo", "latitude", "longitude",
	"lat", "lng", "lon", "coord", "coordinate",
	"nation", "continent", "town", "village", "neighborhood",
	// Chinese keywords
	"�?, "省份", "�?, "城市", "�?, "�?, "�?, "�?, "�?,
	"地区", "区域", "地址", "位置", "地点", "场所",
	"街道", "�?, "大道", "�?, "�?, "�?, "�?,
	"邮编", "邮政编码", "经度", "纬度", "坐标",
	"国家", "�?, "�?, "大洲", "社区", "小区",
	"门店", "店铺", "分店", "网点", "站点",
}

// numericKeywords contains keywords that indicate a numeric/amount column
var numericKeywords = []string{
	// English keywords
	"amount", "count", "price", "quantity", "total", "sum", "avg", "average",
	"number", "num", "value", "rate", "ratio", "percent", "percentage",
	"cost", "fee", "charge", "payment", "revenue", "profit", "loss",
	"sales", "income", "expense", "budget", "balance", "credit", "debit",
	"score", "rating", "rank", "level", "grade", "point", "points",
	"weight", "height", "width", "length", "size", "volume", "area",
	"age", "duration", "distance", "speed", "velocity",
	"stock", "inventory", "capacity", "limit", "max", "min", "threshold",
	"discount", "tax", "commission", "bonus", "salary", "wage",
	// Chinese keywords
	"金额", "数量", "价格", "总计", "合计", "总额", "总数",
	"数目", "数�?, "�?, "比率", "比例", "百分�?, "占比",
	"成本", "费用", "收费", "付款", "收入", "利润", "亏损",
	"销售额", "销售量", "营收", "支出", "预算", "余额",
	"分数", "评分", "评级", "排名", "等级", "积分",
	"重量", "身高", "宽度", "长度", "尺寸", "体积", "面积",
	"年龄", "时长", "距离", "速度",
	"库存", "存量", "容量", "限额", "上限", "下限", "阈�?,
	"折扣", "�?, "税额", "佣金", "奖金", "工资", "薪资",
	"单价", "均价", "总价", "原价", "现价", "售价",
}

// categoricalKeywords contains keywords that indicate a categorical column
var categoricalKeywords = []string{
	// English keywords
	"category", "type", "status", "level", "grade", "class", "kind",
	"group", "segment", "tier", "classification", "tag", "label",
	"gender", "sex", "department", "division", "team", "unit",
	"brand", "model", "version", "edition", "series", "line",
	"channel", "source", "medium", "platform", "device",
	"priority", "severity", "importance", "urgency",
	"stage", "phase", "step", "state", "mode", "method",
	"industry", "sector", "field", "domain", "specialty",
	"role", "position", "title", "job", "occupation", "profession",
	"membership", "subscription", "plan", "package", "tier",
	// Chinese keywords
	"类型", "类别", "分类", "种类", "品类",
	"状�?, "状况", "情况",
	"等级", "级别", "层级", "档次",
	"�?, "组别", "分组", "群组", "群体",
	"标签", "标记", "标识",
	"性别", "部门", "科室", "团队", "单位",
	"品牌", "型号", "版本", "系列", "产品�?,
	"渠道", "来源", "媒介", "平台", "设备",
	"优先�?, "严重程度", "重要�?, "紧急程�?,
	"阶段", "步骤", "环节", "模式", "方式", "方法",
	"行业", "领域", "专业", "方向",
	"角色", "职位", "职称", "岗位", "职业",
	"会员", "会员等级", "订阅", "套餐", "方案",
}

// DimensionAnalyzerImpl 维度分析器实�?
// Analyzes data source characteristics and provides dimension recommendations
// This is the full implementation that replaces the placeholder in intent_enhancement_service.go
type DimensionAnalyzerImpl struct {
	dataSourceService *DataSourceService
	initialized       bool
}

// NewDimensionAnalyzer 创建维度分析�?
// Creates a new DimensionAnalyzer with the provided DataSourceService
// Parameters:
//   - dataSourceService: the service used to access data source information
//
// Returns a new DimensionAnalyzerImpl instance
func NewDimensionAnalyzer(dataSourceService *DataSourceService) *DimensionAnalyzerImpl {
	return &DimensionAnalyzerImpl{
		dataSourceService: dataSourceService,
		initialized:       false,
	}
}

// Initialize 初始化维度分析器
// Initializes the dimension analyzer
// Returns error if initialization fails
func (d *DimensionAnalyzerImpl) Initialize() error {
	if d.dataSourceService == nil {
		// Allow nil dataSourceService for graceful degradation
		// The analyzer will return empty results in this case
	}
	d.initialized = true
	return nil
}

// IsInitialized 检查是否已初始�?
// Returns whether the dimension analyzer has been initialized
func (d *DimensionAnalyzerImpl) IsInitialized() bool {
	return d.initialized
}

// isDateColumn checks if a column name indicates a date/time column
// Parameters:
//   - columnName: the name of the column to check
//
// Returns true if the column name matches date/time keywords
// Validates: Requirements 3.1
func isDateColumn(columnName string) bool {
	return matchesKeywords(columnName, dateKeywords)
}

// isGeographicColumn checks if a column name indicates a geographic/location column
// Parameters:
//   - columnName: the name of the column to check
//
// Returns true if the column name matches geographic keywords
// Validates: Requirements 3.2
func isGeographicColumn(columnName string) bool {
	return matchesKeywords(columnName, geographicKeywords)
}

// isNumericColumn checks if a column name indicates a numeric/amount column
// Parameters:
//   - columnName: the name of the column to check
//
// Returns true if the column name matches numeric keywords
// Validates: Requirements 3.3
func isNumericColumn(columnName string) bool {
	return matchesKeywords(columnName, numericKeywords)
}

// isCategoricalColumn checks if a column name indicates a categorical column
// Parameters:
//   - columnName: the name of the column to check
//
// Returns true if the column name matches categorical keywords
// Validates: Requirements 3.4
func isCategoricalColumn(columnName string) bool {
	return matchesKeywords(columnName, categoricalKeywords)
}

// matchesKeywords checks if a column name contains any of the given keywords
// The matching is case-insensitive and supports partial matching
// Parameters:
//   - columnName: the name of the column to check
//   - keywords: the list of keywords to match against
//
// Returns true if the column name contains any of the keywords
func matchesKeywords(columnName string, keywords []string) bool {
	lowerName := strings.ToLower(columnName)
	for _, keyword := range keywords {
		lowerKeyword := strings.ToLower(keyword)
		if strings.Contains(lowerName, lowerKeyword) {
			return true
		}
	}
	return false
}

// identifyColumnType determines the data type and semantic type of a column
// based on its name and database type
// Parameters:
//   - columnName: the name of the column
//   - dbType: the database type of the column (e.g., "INTEGER", "TEXT", "REAL")
//
// Returns the data type and semantic type of the column
// Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5
func identifyColumnType(columnName string, dbType string) (dataType string, semanticType string) {
	// First, check by column name (semantic analysis)
	// Priority: date > geographic > numeric > categorical > text
	
	if isDateColumn(columnName) {
		return ColumnTypeDate, SemanticTypeTime
	}
	
	if isGeographicColumn(columnName) {
		return ColumnTypeGeographic, SemanticTypeLocation
	}
	
	if isNumericColumn(columnName) {
		// Determine if it's an amount or count based on keywords
		lowerName := strings.ToLower(columnName)
		if strings.Contains(lowerName, "count") || strings.Contains(lowerName, "数量") ||
			strings.Contains(lowerName, "num") || strings.Contains(lowerName, "数目") ||
			strings.Contains(lowerName, "quantity") {
			return ColumnTypeNumeric, SemanticTypeCount
		}
		return ColumnTypeNumeric, SemanticTypeAmount
	}
	
	if isCategoricalColumn(columnName) {
		return ColumnTypeCategorical, SemanticTypeCategory
	}
	
	// If no keyword match, infer from database type
	upperDBType := strings.ToUpper(dbType)
	
	// Check for date/time types in database
	if strings.Contains(upperDBType, "DATE") || strings.Contains(upperDBType, "TIME") ||
		strings.Contains(upperDBType, "TIMESTAMP") {
		return ColumnTypeDate, SemanticTypeTime
	}
	
	// Check for numeric types in database
	if strings.Contains(upperDBType, "INT") || strings.Contains(upperDBType, "REAL") ||
		strings.Contains(upperDBType, "FLOAT") || strings.Contains(upperDBType, "DOUBLE") ||
		strings.Contains(upperDBType, "DECIMAL") || strings.Contains(upperDBType, "NUMERIC") ||
		strings.Contains(upperDBType, "NUMBER") {
		return ColumnTypeNumeric, SemanticTypeAmount
	}
	
	// Default to text type
	return ColumnTypeText, SemanticTypeUnknown
}

// AnalyzeDataSource 分析数据源特�?
// Analyzes the characteristics of columns in a data source
// Parameters:
//   - dataSourceID: the ID of the data source to analyze
//
// Returns a slice of ColumnCharacteristics and any error
// Validates: Requirements 3.5
func (d *DimensionAnalyzerImpl) AnalyzeDataSource(dataSourceID string) ([]ColumnCharacteristics, error) {
	if d.dataSourceService == nil {
		// Graceful degradation: return empty results if no data source service
		return []ColumnCharacteristics{}, nil
	}
	
	// Get all tables in the data source
	tables, err := d.dataSourceService.GetTables(dataSourceID)
	if err != nil {
		return nil, err
	}
	
	if len(tables) == 0 {
		return []ColumnCharacteristics{}, nil
	}
	
	var allCharacteristics []ColumnCharacteristics
	
	// Analyze columns from all tables
	for _, tableName := range tables {
		columns, err := d.dataSourceService.GetTableColumns(dataSourceID, tableName)
		if err != nil {
			// Skip tables that can't be analyzed
			continue
		}
		
		for _, col := range columns {
			dataType, semanticType := identifyColumnType(col.Name, col.Type)
			
			characteristic := ColumnCharacteristics{
				Name:         col.Name,
				DataType:     dataType,
				SemanticType: semanticType,
				UniqueRatio:  0.0, // Would require data sampling to calculate
				SampleValues: []string{}, // Would require data sampling to populate
			}
			
			allCharacteristics = append(allCharacteristics, characteristic)
		}
	}
	
	return allCharacteristics, nil
}

// AnalyzeColumns 分析列特征（直接从列信息�?
// Analyzes column characteristics directly from column schema information
// This is useful when you already have column information and don't need to query the data source
// Parameters:
//   - columns: slice of ColumnSchema containing column name and type information
//
// Returns a slice of ColumnCharacteristics
// Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5
func (d *DimensionAnalyzerImpl) AnalyzeColumns(columns []ColumnSchema) []ColumnCharacteristics {
	var characteristics []ColumnCharacteristics
	
	for _, col := range columns {
		dataType, semanticType := identifyColumnType(col.Name, col.Type)
		
		characteristic := ColumnCharacteristics{
			Name:         col.Name,
			DataType:     dataType,
			SemanticType: semanticType,
			UniqueRatio:  0.0,
			SampleValues: []string{},
		}
		
		characteristics = append(characteristics, characteristic)
	}
	
	return characteristics
}

// Dimension type constants
const (
	DimensionTypeTemporal    = "temporal"
	DimensionTypeGeographic  = "geographic"
	DimensionTypeStatistical = "statistical"
	DimensionTypeCategorical = "categorical"
)

// GetDimensionRecommendations 获取维度推荐
// Generates dimension recommendations based on column characteristics
// Parameters:
//   - characteristics: the column characteristics to analyze
//
// Returns a slice of DimensionRecommendation sorted by priority (highest first)
// Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.6
func (d *DimensionAnalyzerImpl) GetDimensionRecommendations(
	characteristics []ColumnCharacteristics,
) []DimensionRecommendation {
	if len(characteristics) == 0 {
		return []DimensionRecommendation{}
	}

	// Collect columns by dimension type
	var dateColumns []string
	var geographicColumns []string
	var numericColumns []string
	var categoricalColumns []string

	for _, col := range characteristics {
		switch col.DataType {
		case ColumnTypeDate:
			dateColumns = append(dateColumns, col.Name)
		case ColumnTypeGeographic:
			geographicColumns = append(geographicColumns, col.Name)
		case ColumnTypeNumeric:
			numericColumns = append(numericColumns, col.Name)
		case ColumnTypeCategorical:
			categoricalColumns = append(categoricalColumns, col.Name)
		}
	}

	var recommendations []DimensionRecommendation

	// Generate temporal dimension recommendation (Priority 8-10)
	// Date columns are highly valuable for time series analysis
	if len(dateColumns) > 0 {
		priority := calculateTemporalPriority(dateColumns)
		recommendations = append(recommendations, DimensionRecommendation{
			DimensionType: DimensionTypeTemporal,
			Priority:      priority,
			Columns:       dateColumns,
			Rationale:     generateTemporalRationale(dateColumns),
		})
	}

	// Generate statistical dimension recommendation (Priority 7-9)
	// Numeric columns enable statistical analysis
	if len(numericColumns) > 0 {
		priority := calculateStatisticalPriority(numericColumns)
		recommendations = append(recommendations, DimensionRecommendation{
			DimensionType: DimensionTypeStatistical,
			Priority:      priority,
			Columns:       numericColumns,
			Rationale:     generateStatisticalRationale(numericColumns),
		})
	}

	// Generate geographic dimension recommendation (Priority 6-8)
	// Geographic columns enable regional analysis
	if len(geographicColumns) > 0 {
		priority := calculateGeographicPriority(geographicColumns)
		recommendations = append(recommendations, DimensionRecommendation{
			DimensionType: DimensionTypeGeographic,
			Priority:      priority,
			Columns:       geographicColumns,
			Rationale:     generateGeographicRationale(geographicColumns),
		})
	}

	// Generate categorical dimension recommendation (Priority 5-7)
	// Categorical columns enable grouping and comparison analysis
	if len(categoricalColumns) > 0 {
		priority := calculateCategoricalPriority(categoricalColumns)
		recommendations = append(recommendations, DimensionRecommendation{
			DimensionType: DimensionTypeCategorical,
			Priority:      priority,
			Columns:       categoricalColumns,
			Rationale:     generateCategoricalRationale(categoricalColumns),
		})
	}

	// Sort recommendations by priority (highest first)
	// Validates: Requirements 3.6
	sortRecommendationsByPriority(recommendations)

	return recommendations
}

// calculateTemporalPriority calculates priority for temporal dimension (8-10)
// More date columns and specific date types increase priority
func calculateTemporalPriority(columns []string) int {
	basePriority := 8
	
	// Increase priority based on number of date columns
	if len(columns) >= 3 {
		basePriority = 10
	} else if len(columns) >= 2 {
		basePriority = 9
	}
	
	return basePriority
}

// calculateStatisticalPriority calculates priority for statistical dimension (7-9)
// More numeric columns increase priority
func calculateStatisticalPriority(columns []string) int {
	basePriority := 7
	
	// Increase priority based on number of numeric columns
	if len(columns) >= 5 {
		basePriority = 9
	} else if len(columns) >= 3 {
		basePriority = 8
	}
	
	return basePriority
}

// calculateGeographicPriority calculates priority for geographic dimension (6-8)
// More geographic columns increase priority
func calculateGeographicPriority(columns []string) int {
	basePriority := 6
	
	// Increase priority based on number of geographic columns
	if len(columns) >= 3 {
		basePriority = 8
	} else if len(columns) >= 2 {
		basePriority = 7
	}
	
	return basePriority
}

// calculateCategoricalPriority calculates priority for categorical dimension (5-7)
// More categorical columns increase priority
func calculateCategoricalPriority(columns []string) int {
	basePriority := 5
	
	// Increase priority based on number of categorical columns
	if len(columns) >= 4 {
		basePriority = 7
	} else if len(columns) >= 2 {
		basePriority = 6
	}
	
	return basePriority
}

// generateTemporalRationale generates rationale for temporal dimension recommendation
func generateTemporalRationale(columns []string) string {
	if len(columns) == 1 {
		return "数据包含日期�?'" + columns[0] + "'，适合进行时间序列分析和趋势分�?
	}
	return "数据包含多个日期�?(" + strings.Join(columns, ", ") + ")，非常适合进行时间维度分析、趋势分析和周期性分�?
}

// generateStatisticalRationale generates rationale for statistical dimension recommendation
func generateStatisticalRationale(columns []string) string {
	if len(columns) == 1 {
		return "数据包含数值列 '" + columns[0] + "'，适合进行统计分析和数值计�?
	}
	return "数据包含多个数值列 (" + strings.Join(columns, ", ") + ")，适合进行统计分析、聚合计算和数值对�?
}

// generateGeographicRationale generates rationale for geographic dimension recommendation
func generateGeographicRationale(columns []string) string {
	if len(columns) == 1 {
		return "数据包含地理位置�?'" + columns[0] + "'，适合进行区域分析和地理分布分�?
	}
	return "数据包含多个地理位置�?(" + strings.Join(columns, ", ") + ")，适合进行多层级区域分析和地理分布对比"
}

// generateCategoricalRationale generates rationale for categorical dimension recommendation
func generateCategoricalRationale(columns []string) string {
	if len(columns) == 1 {
		return "数据包含分类�?'" + columns[0] + "'，适合进行分组对比分析"
	}
	return "数据包含多个分类�?(" + strings.Join(columns, ", ") + ")，适合进行多维度分组对比和交叉分析"
}

// sortRecommendationsByPriority sorts recommendations by priority in descending order
// Uses stable sort to maintain relative order of equal priorities
// Validates: Requirements 3.6
func sortRecommendationsByPriority(recommendations []DimensionRecommendation) {
	// Simple bubble sort for small slices (typically 4 or fewer items)
	n := len(recommendations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if recommendations[j].Priority < recommendations[j+1].Priority {
				recommendations[j], recommendations[j+1] = recommendations[j+1], recommendations[j]
			}
		}
	}
}

// BuildDimensionSection 构建维度提示词片�?
// Builds a prompt section containing dimension recommendations
// Parameters:
//   - recommendations: the dimension recommendations to include
//   - language: the language for output ("zh" for Chinese, "en" for English)
//
// Returns a formatted prompt section string
// Returns empty string if recommendations is empty
// Validates: Requirements 8.1
func (d *DimensionAnalyzerImpl) BuildDimensionSection(
	recommendations []DimensionRecommendation,
	language string,
) string {
	if len(recommendations) == 0 {
		return ""
	}

	var sb strings.Builder

	// Write header based on language
	if language == "zh" {
		sb.WriteString("## 数据维度分析\n")
		sb.WriteString("根据数据源的列特征，建议关注以下分析维度：\n\n")
	} else {
		sb.WriteString("## Data Dimension Analysis\n")
		sb.WriteString("Based on the column characteristics of the data source, the following analysis dimensions are recommended:\n\n")
	}

	// Write each recommendation
	for i, rec := range recommendations {
		// Get dimension type display name
		dimTypeName := getDimensionTypeName(rec.DimensionType, language)
		
		// Write recommendation header with number, dimension type, and priority
		if language == "zh" {
			sb.WriteString(formatInt(i+1) + ". " + dimTypeName + " (优先�? " + formatInt(rec.Priority) + ")\n")
		} else {
			sb.WriteString(formatInt(i+1) + ". " + dimTypeName + " (Priority: " + formatInt(rec.Priority) + ")\n")
		}

		// Write related columns
		columnsStr := strings.Join(rec.Columns, ", ")
		if language == "zh" {
			sb.WriteString("   - 相关�? " + columnsStr + "\n")
		} else {
			sb.WriteString("   - Related columns: " + columnsStr + "\n")
		}

		// Write rationale
		rationale := getRationaleForLanguage(rec, language)
		if language == "zh" {
			sb.WriteString("   - 建议原因: " + rationale + "\n")
		} else {
			sb.WriteString("   - Rationale: " + rationale + "\n")
		}

		// Add blank line between recommendations (except for the last one)
		if i < len(recommendations)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// getDimensionTypeName returns the display name for a dimension type in the specified language
// Parameters:
//   - dimType: the dimension type constant
//   - language: the language for output ("zh" for Chinese, "en" for English)
//
// Returns the localized dimension type name
func getDimensionTypeName(dimType string, language string) string {
	if language == "zh" {
		switch dimType {
		case DimensionTypeTemporal:
			return "时间序列分析"
		case DimensionTypeGeographic:
			return "区域分析"
		case DimensionTypeStatistical:
			return "统计分析"
		case DimensionTypeCategorical:
			return "分组对比分析"
		default:
			return dimType
		}
	}
	
	// English
	switch dimType {
	case DimensionTypeTemporal:
		return "Time Series Analysis"
	case DimensionTypeGeographic:
		return "Geographic Analysis"
	case DimensionTypeStatistical:
		return "Statistical Analysis"
	case DimensionTypeCategorical:
		return "Categorical Comparison Analysis"
	default:
		return dimType
	}
}

// getRationaleForLanguage returns the rationale in the specified language
// If the stored rationale is in Chinese and English is requested (or vice versa),
// it generates an appropriate rationale based on the dimension type and columns
// Parameters:
//   - rec: the dimension recommendation
//   - language: the language for output ("zh" for Chinese, "en" for English)
//
// Returns the localized rationale string
func getRationaleForLanguage(rec DimensionRecommendation, language string) string {
	// If the rationale is already in the target language, return it
	// Check if rationale contains Chinese characters
	hasChineseChars := containsChineseCharacters(rec.Rationale)
	
	if language == "zh" && hasChineseChars {
		return rec.Rationale
	}
	
	if language != "zh" && !hasChineseChars {
		return rec.Rationale
	}
	
	// Generate rationale in the target language
	columnsStr := strings.Join(rec.Columns, ", ")
	
	if language == "zh" {
		switch rec.DimensionType {
		case DimensionTypeTemporal:
			if len(rec.Columns) == 1 {
				return "数据包含日期/时间列，适合进行趋势分析和时间序列分�?
			}
			return "数据包含多个日期/时间列，非常适合进行时间维度分析、趋势分析和周期性分�?
		case DimensionTypeGeographic:
			if len(rec.Columns) == 1 {
				return "数据包含地理位置列，适合进行区域分析和地理分布分�?
			}
			return "数据包含多个地理位置列，适合进行多层级区域分析和地理分布对比"
		case DimensionTypeStatistical:
			if len(rec.Columns) == 1 {
				return "数据包含数值列，适合进行统计分析和数值计�?
			}
			return "数据包含多个数值列，适合进行统计分析、聚合计算和数值对�?
		case DimensionTypeCategorical:
			if len(rec.Columns) == 1 {
				return "数据包含分类列，适合进行分组对比分析"
			}
			return "数据包含多个分类列，适合进行多维度分组对比和交叉分析"
		default:
			return rec.Rationale
		}
	}
	
	// English
	switch rec.DimensionType {
	case DimensionTypeTemporal:
		if len(rec.Columns) == 1 {
			return "Data contains date/time column (" + columnsStr + "), suitable for trend analysis and time series analysis"
		}
		return "Data contains multiple date/time columns (" + columnsStr + "), ideal for temporal analysis, trend analysis, and periodicity analysis"
	case DimensionTypeGeographic:
		if len(rec.Columns) == 1 {
			return "Data contains geographic column (" + columnsStr + "), suitable for regional analysis and geographic distribution analysis"
		}
		return "Data contains multiple geographic columns (" + columnsStr + "), suitable for multi-level regional analysis and geographic distribution comparison"
	case DimensionTypeStatistical:
		if len(rec.Columns) == 1 {
			return "Data contains numeric column (" + columnsStr + "), suitable for statistical analysis and numerical calculations"
		}
		return "Data contains multiple numeric columns (" + columnsStr + "), suitable for statistical analysis, aggregation calculations, and numerical comparison"
	case DimensionTypeCategorical:
		if len(rec.Columns) == 1 {
			return "Data contains categorical column (" + columnsStr + "), suitable for grouping and comparison analysis"
		}
		return "Data contains multiple categorical columns (" + columnsStr + "), suitable for multi-dimensional grouping comparison and cross-analysis"
	default:
		return rec.Rationale
	}
}

// containsChineseCharacters checks if a string contains Chinese characters
func containsChineseCharacters(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// formatInt converts an integer to a string
// This is a simple helper to avoid importing strconv for just this purpose
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	
	if n < 0 {
		return "-" + formatInt(-n)
	}
	
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
