package agent

import (
	"strings"
)

// RequestClassifier classifies user requests into different types
type RequestClassifier struct {
	logger func(string)
}

// NewRequestClassifier creates a new request classifier
func NewRequestClassifier(logger func(string)) *RequestClassifier {
	return &RequestClassifier{
		logger: logger,
	}
}

// ClassifyRequest analyzes the request and returns its type
func (c *RequestClassifier) ClassifyRequest(query string, dataSourceInfo string) RequestType {
	if query == "" {
		return RequestTypeTrivial
	}

	queryLower := strings.ToLower(query)

	// Check for consultation requests first (highest priority)
	if c.IsConsultationRequest(queryLower) {
		if c.logger != nil {
			c.logger("[CLASSIFIER] Request classified as: consultation")
		}
		return RequestTypeConsultation
	}

	// Check for multi-step analysis patterns
	if c.IsMultiStepAnalysis(queryLower) {
		if c.logger != nil {
			c.logger("[CLASSIFIER] Request classified as: multi_step_analysis")
		}
		return RequestTypeMultiStepAnalysis
	}

	// Check for web search patterns
	if c.IsWebSearchRequest(queryLower) {
		if c.logger != nil {
			c.logger("[CLASSIFIER] Request classified as: web_search")
		}
		return RequestTypeWebSearch
	}

	// Check for visualization patterns
	if c.IsVisualizationRequest(queryLower) {
		if c.logger != nil {
			c.logger("[CLASSIFIER] Request classified as: visualization")
		}
		return RequestTypeVisualization
	}

	// Check for calculation patterns
	if c.IsCalculationRequest(queryLower) {
		if c.logger != nil {
			c.logger("[CLASSIFIER] Request classified as: calculation")
		}
		return RequestTypeCalculation
	}

	// Default to data_query for database-related requests
	if c.logger != nil {
		c.logger("[CLASSIFIER] Request classified as: data_query (default)")
	}
	return RequestTypeDataQuery
}

// IsConsultationRequest checks if the request is asking for suggestions/advice
func (c *RequestClassifier) IsConsultationRequest(queryLower string) bool {
	// Check for consultation keywords
	for _, keyword := range ConsultationPatterns {
		if strings.Contains(queryLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// IsMultiStepAnalysis checks if the request requires multi-step analysis
func (c *RequestClassifier) IsMultiStepAnalysis(queryLower string) bool {
	// Check for multi-step analysis keywords
	for _, keyword := range MultiStepPatterns {
		if strings.Contains(queryLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// IsWebSearchRequest checks if the request requires web search
func (c *RequestClassifier) IsWebSearchRequest(queryLower string) bool {
	webSearchKeywords := []string{
		"搜索", "查询", "最新", "新闻", "股价", "天气", "实时",
		"search", "latest", "news", "stock", "weather", "real-time",
	}
	for _, keyword := range webSearchKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}
	return false
}

// IsVisualizationRequest checks if the request requires visualization
func (c *RequestClassifier) IsVisualizationRequest(queryLower string) bool {
	vizKeywords := []string{
		"图", "图表", "可视化", "趋势", "分布", "对比", "排名",
		"chart", "visualization", "trend", "distribution", "comparison", "ranking",
	}
	for _, keyword := range vizKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}
	return false
}

// IsCalculationRequest checks if the request is a simple calculation
func (c *RequestClassifier) IsCalculationRequest(queryLower string) bool {
	calcKeywords := []string{
		"计算", "等于多少", "加", "减", "乘", "除", "平方", "开方",
		"calculate", "compute", "plus", "minus", "multiply", "divide",
	}
	for _, keyword := range calcKeywords {
		if strings.Contains(queryLower, keyword) {
			// Make sure it's not a data query
			if !strings.Contains(queryLower, "订单") && !strings.Contains(queryLower, "销售") &&
				!strings.Contains(queryLower, "数据") && !strings.Contains(queryLower, "查询") {
				return true
			}
		}
	}
	return false
}

// GetSchemaLevel determines the appropriate schema level for a request type
func (c *RequestClassifier) GetSchemaLevel(requestType RequestType) SchemaLevel {
	if level, ok := SchemaLevelMapping[requestType]; ok {
		return level
	}
	// Default to detailed for unknown types
	return SchemaLevelDetailed
}
