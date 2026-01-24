package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
)

// ExecutionPath represents the chosen execution path
type ExecutionPath string

const (
	PathQuick        ExecutionPath = "quick"        // No LLM needed
	PathSQLOnly      ExecutionPath = "sql_only"     // Direct SQL execution
	PathUnified      ExecutionPath = "unified"      // Unified Python generation
	PathMultiStep    ExecutionPath = "multi_step"   // Traditional multi-step
	PathConsultation ExecutionPath = "consultation" // Suggestion only
)

// RequestRouter routes analysis requests to the optimal execution path
type RequestRouter struct {
	classifier    *RequestClassifier
	llmClassifier *RequestTypeClassifier
	logger        func(string)
	useLLM        bool // Whether to use LLM for classification
}

// NewRequestRouter creates a new request router
func NewRequestRouter(logger func(string)) *RequestRouter {
	return &RequestRouter{
		classifier: NewRequestClassifier(logger),
		logger:     logger,
		useLLM:     false, // Default to keyword-based for backward compatibility
	}
}

// NewRequestRouterWithLLM creates a router that uses LLM for classification
func NewRequestRouterWithLLM(chatModel model.ChatModel, logger func(string)) *RequestRouter {
	return &RequestRouter{
		classifier:    NewRequestClassifier(logger),
		llmClassifier: NewRequestTypeClassifier(chatModel, logger),
		logger:        logger,
		useLLM:        true,
	}
}

func (r *RequestRouter) log(msg string) {
	if r.logger != nil {
		r.logger(msg)
	}
}

// RouteRequestWithLLM uses LLM to determine the optimal execution path
func (r *RequestRouter) RouteRequestWithLLM(ctx context.Context, userRequest string, dataSourceInfo string) (ExecutionPath, *ClassificationResult) {
	if userRequest == "" {
		return PathQuick, nil
	}

	// Quick path detection (no LLM needed for obvious cases)
	requestLower := strings.ToLower(userRequest)
	if r.isQuickRequest(requestLower) {
		r.log("[ROUTER] Quick path detected (no LLM needed)")
		return PathQuick, nil
	}

	// Use LLM classifier if available
	if r.llmClassifier != nil {
		result, err := r.llmClassifier.ClassifyRequest(ctx, userRequest, dataSourceInfo)
		if err == nil && result != nil {
			path := r.classificationToPath(result, dataSourceInfo)
			r.log(fmt.Sprintf("[ROUTER] LLM classification: type=%s, path=%s", result.RequestType, path))
			return path, result
		}
		r.log("[ROUTER] LLM classification failed, falling back to keyword-based")
	}

	// Fallback to keyword-based routing
	return r.RouteRequest(userRequest, dataSourceInfo), nil
}

// classificationToPath converts LLM classification to execution path
func (r *RequestRouter) classificationToPath(result *ClassificationResult, dataSourceInfo string) ExecutionPath {
	switch result.RequestType {
	case "consultation":
		return PathConsultation
	case "calculation":
		return PathQuick
	case "web_search":
		return PathMultiStep
	case "data_analysis", "visualization", "data_export":
		if dataSourceInfo != "" {
			return PathUnified
		}
		return PathMultiStep
	default:
		if dataSourceInfo != "" {
			return PathUnified
		}
		return PathMultiStep
	}
}

// RouteRequest determines the optimal execution path
func (r *RequestRouter) RouteRequest(userRequest string, dataSourceInfo string) ExecutionPath {
	if userRequest == "" {
		return PathQuick
	}

	requestLower := strings.ToLower(userRequest)

	// 1. Check for quick path (no LLM needed)
	if r.isQuickRequest(requestLower) {
		r.log("[ROUTER] Routing to quick path")
		return PathQuick
	}

	// 2. Check for consultation requests
	if r.isConsultationRequest(requestLower) {
		r.log("[ROUTER] Routing to consultation path")
		return PathConsultation
	}

	// 3. Check for visualization requests -> unified path
	if r.isVisualizationRequest(requestLower) {
		r.log("[ROUTER] Routing to unified path (visualization)")
		return PathUnified
	}

	// 4. Check for complex analysis -> unified path
	if r.isComplexAnalysis(requestLower) {
		r.log("[ROUTER] Routing to unified path (complex analysis)")
		return PathUnified
	}

	// 5. Check for simple queries -> SQL only path
	if r.isSimpleQuery(requestLower) && dataSourceInfo != "" {
		r.log("[ROUTER] Routing to SQL-only path")
		return PathSQLOnly
	}

	// 6. Default to unified path for data analysis
	if dataSourceInfo != "" {
		r.log("[ROUTER] Routing to unified path (default)")
		return PathUnified
	}

	// 7. No data source, use multi-step
	r.log("[ROUTER] Routing to multi-step path")
	return PathMultiStep
}

// ShouldUseUnifiedPath checks if unified Python path is appropriate
func (r *RequestRouter) ShouldUseUnifiedPath(userRequest string) bool {
	path := r.RouteRequest(userRequest, "has_datasource")
	return path == PathUnified
}

// isQuickRequest checks if the request can be answered without LLM
func (r *RequestRouter) isQuickRequest(requestLower string) bool {
	quickPatterns := []string{
		"现在几点", "什么时间", "今天日期", "当前时间",
		"what time", "current time", "today's date",
		"你好", "hello", "hi", "帮助", "help",
	}
	for _, pattern := range quickPatterns {
		if strings.Contains(requestLower, pattern) {
			return true
		}
	}
	return false
}

// isConsultationRequest checks if the request is asking for suggestions
// More strict: only pure consultation requests, not analysis requests
func (r *RequestRouter) isConsultationRequest(requestLower string) bool {
	// First check if it's an actual analysis request (should NOT be consultation)
	analysisIndicators := []string{
		"分析", "统计", "查询", "计算", "对比", "趋势", "分布", "排名",
		"销售", "订单", "客户", "产品", "收入", "利润", "数量",
		"analyze", "query", "calculate", "compare", "trend", "distribution",
		"图", "表", "chart", "table", "可视化", "visualization",
	}
	for _, indicator := range analysisIndicators {
		if strings.Contains(requestLower, indicator) {
			return false // This is an analysis request, not consultation
		}
	}
	
	// Only pure consultation patterns
	consultPatterns := []string{
		"可以做什么分析", "分析方向", "怎么分析",
		"what analysis", "how to analyze",
		"能做什么", "应该怎么",
	}
	for _, pattern := range consultPatterns {
		if strings.Contains(requestLower, pattern) {
			return true
		}
	}
	return false
}

// isVisualizationRequest checks if the request requires visualization
// More inclusive: most data analysis benefits from visualization
func (r *RequestRouter) isVisualizationRequest(requestLower string) bool {
	// Explicit visualization patterns
	vizPatterns := []string{
		"图", "图表", "可视化", "趋势", "分布", "对比", "排名",
		"chart", "visualization", "trend", "distribution", "comparison",
		"柱状图", "折线图", "饼图", "散点图", "热力图",
		"bar chart", "line chart", "pie chart", "scatter", "heatmap",
		"画", "绘制", "展示", "显示趋势",
	}
	for _, pattern := range vizPatterns {
		if strings.Contains(requestLower, pattern) {
			return true
		}
	}
	
	// Implicit visualization: analysis requests that would benefit from charts
	// These are common analysis patterns that should produce visualizations
	implicitVizPatterns := []string{
		"分析", "统计", "销售", "收入", "利润", "增长",
		"top", "前", "最", "排行", "占比", "比例",
		"按月", "按年", "按季度", "按周", "时间",
		"analysis", "sales", "revenue", "growth", "monthly", "yearly",
	}
	matchCount := 0
	for _, pattern := range implicitVizPatterns {
		if strings.Contains(requestLower, pattern) {
			matchCount++
		}
	}
	// If multiple analysis indicators, likely needs visualization
	return matchCount >= 2
}

// isComplexAnalysis checks if the request requires complex analysis
func (r *RequestRouter) isComplexAnalysis(requestLower string) bool {
	complexPatterns := []string{
		"分析", "预测", "相关性", "回归", "聚类", "分类",
		"analysis", "predict", "correlation", "regression", "cluster",
		"rfm", "cohort", "漏斗", "funnel", "留存", "retention",
		"同比", "环比", "增长率", "占比",
	}
	for _, pattern := range complexPatterns {
		if strings.Contains(requestLower, pattern) {
			return true
		}
	}
	return false
}

// isSimpleQuery checks if the request is a simple data query
// More strict: only truly simple queries that don't need visualization
func (r *RequestRouter) isSimpleQuery(requestLower string) bool {
	// Simple queries typically just ask for raw data without analysis
	simplePatterns := []string{
		"列出", "显示所有", "查看所有", "有多少条",
		"list all", "show all", "view all", "count records",
	}
	
	// Keywords that indicate analysis/visualization is needed
	analysisKeywords := []string{
		"分析", "趋势", "对比", "图", "预测", "统计", "汇总",
		"排名", "top", "前", "最", "占比", "比例", "增长",
		"按", "分组", "group", "aggregate",
		"analysis", "trend", "compare", "chart", "predict", "summary",
	}
	
	hasSimple := false
	for _, pattern := range simplePatterns {
		if strings.Contains(requestLower, pattern) {
			hasSimple = true
			break
		}
	}
	
	if !hasSimple {
		return false
	}
	
	// Check if it also has analysis keywords - if so, not simple
	for _, keyword := range analysisKeywords {
		if strings.Contains(requestLower, keyword) {
			return false
		}
	}
	
	return true
}

// GetPathDescription returns a human-readable description of the path
func (r *RequestRouter) GetPathDescription(path ExecutionPath) string {
	switch path {
	case PathQuick:
		return "快速响应（无需LLM）"
	case PathSQLOnly:
		return "SQL直接查询"
	case PathUnified:
		return "统一Python分析"
	case PathMultiStep:
		return "多步骤分析"
	case PathConsultation:
		return "咨询建议"
	default:
		return "未知路径"
	}
}
