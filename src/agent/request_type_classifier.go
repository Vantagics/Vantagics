package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// RequestTypeClassifier uses LLM to classify request types
type RequestTypeClassifier struct {
	chatModel model.ChatModel
	logger    func(string)
	cache     map[string]*ClassificationResult
}

// ClassificationResult represents the LLM classification result
type ClassificationResult struct {
	RequestType       string   `json:"request_type"`
	NeedsVisualization bool    `json:"needs_visualization"`
	NeedsDataExport   bool     `json:"needs_data_export"`
	Confidence        float64  `json:"confidence"`
	Reasoning         string   `json:"reasoning"`
	SuggestedOutputs  []string `json:"suggested_outputs"`
	CachedAt          time.Time
}

// NewRequestTypeClassifier creates a new LLM-based classifier
func NewRequestTypeClassifier(chatModel model.ChatModel, logger func(string)) *RequestTypeClassifier {
	return &RequestTypeClassifier{
		chatModel: chatModel,
		logger:    logger,
		cache:     make(map[string]*ClassificationResult),
	}
}

func (c *RequestTypeClassifier) log(msg string) {
	if c.logger != nil {
		c.logger(msg)
	}
}

// ClassifyRequest uses LLM to classify the request type
func (c *RequestTypeClassifier) ClassifyRequest(ctx context.Context, userQuery string, dataSourceInfo string) (*ClassificationResult, error) {
	// Check cache first (valid for 5 minutes)
	cacheKey := fmt.Sprintf("%s|%s", userQuery, dataSourceInfo)
	if cached, ok := c.cache[cacheKey]; ok {
		if time.Since(cached.CachedAt) < 5*time.Minute {
			c.log("[CLASSIFIER-LLM] Using cached classification result")
			return cached, nil
		}
	}

	startTime := time.Now()
	c.log("[CLASSIFIER-LLM] Classifying request with LLM...")

	prompt := fmt.Sprintf(`你是一个请求分类专家。分析用户请求，判断其类型和所需输出。

## 用户请求
"%s"

## 数据源信息
%s

## 分类规则

### 请求类型 (request_type)
1. **consultation** - 纯咨询/建议请求，用户只是想了解可以做什么，不需要实际执行分析
   - 例如："这个数据可以做什么分析？"、"有什么分析建议？"、"能帮我做什么？"
   
2. **data_analysis** - 数据分析请求，需要查询数据并进行分析
   - 例如："分析销售趋势"、"统计订单数量"、"查看客户分布"
   
3. **visualization** - 明确需要可视化的请求
   - 例如："画一个图表"、"展示趋势图"、"生成饼图"
   
4. **data_export** - 数据导出请求
   - 例如："导出数据"、"生成报告"、"下载Excel"
   
5. **calculation** - 简单计算请求（不需要数据库）
   - 例如："计算2+2"、"现在几点"
   
6. **web_search** - 需要网络搜索的请求
   - 例如："搜索最新新闻"、"查询股价"

### 判断是否需要可视化 (needs_visualization)
- 数据分析请求通常需要可视化来展示结果
- 趋势、分布、对比、排名等分析都应该有图表
- 只有纯数据查询（如"列出所有订单"）不需要可视化

### 判断是否需要数据导出 (needs_data_export)
- 用户明确要求导出、下载、生成报告时为true
- 分析结果较大时建议导出

## 输出格式 (JSON)
{
  "request_type": "consultation|data_analysis|visualization|data_export|calculation|web_search",
  "needs_visualization": true/false,
  "needs_data_export": true/false,
  "confidence": 0.0-1.0,
  "reasoning": "简短解释分类原因",
  "suggested_outputs": ["chart", "table", "insight", "excel", "pdf"]
}

只输出JSON，不要其他内容。`, userQuery, dataSourceInfo)

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是请求分类专家。只输出有效JSON，不要其他内容。",
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	resp, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		c.log(fmt.Sprintf("[CLASSIFIER-LLM] LLM call failed: %v, using fallback", err))
		return c.fallbackClassification(userQuery), nil
	}

	// Parse response
	result := &ClassificationResult{}
	content := strings.TrimSpace(resp.Content)
	content = extractJSON(content)

	if err := json.Unmarshal([]byte(content), result); err != nil {
		c.log(fmt.Sprintf("[CLASSIFIER-LLM] Failed to parse response: %v, using fallback", err))
		return c.fallbackClassification(userQuery), nil
	}

	result.CachedAt = time.Now()
	c.cache[cacheKey] = result

	c.log(fmt.Sprintf("[CLASSIFIER-LLM] Classification complete in %v: type=%s, viz=%v, export=%v, confidence=%.2f",
		time.Since(startTime), result.RequestType, result.NeedsVisualization, result.NeedsDataExport, result.Confidence))

	return result, nil
}

// fallbackClassification provides a fallback when LLM fails
func (c *RequestTypeClassifier) fallbackClassification(query string) *ClassificationResult {
	queryLower := strings.ToLower(query)

	result := &ClassificationResult{
		RequestType:       "data_analysis",
		NeedsVisualization: true,
		NeedsDataExport:   false,
		Confidence:        0.5,
		Reasoning:         "Fallback classification based on keywords",
		SuggestedOutputs:  []string{"chart", "table", "insight"},
		CachedAt:          time.Now(),
	}

	// Simple keyword-based fallback
	if containsAny(queryLower, []string{"建议", "可以做什么", "能做什么", "suggest", "recommend"}) &&
		!containsAny(queryLower, []string{"分析", "统计", "查询", "销售", "订单"}) {
		result.RequestType = "consultation"
		result.NeedsVisualization = false
		result.SuggestedOutputs = []string{"text"}
	} else if containsAny(queryLower, []string{"导出", "下载", "报告", "export", "download"}) {
		result.RequestType = "data_export"
		result.NeedsDataExport = true
		result.SuggestedOutputs = []string{"excel", "pdf"}
	} else if containsAny(queryLower, []string{"计算", "几点", "时间", "calculate", "time"}) &&
		!containsAny(queryLower, []string{"数据", "订单", "销售"}) {
		result.RequestType = "calculation"
		result.NeedsVisualization = false
		result.SuggestedOutputs = []string{"text"}
	}

	return result
}

// ToRequestType converts classification result to RequestType enum
func (r *ClassificationResult) ToRequestType() RequestType {
	switch r.RequestType {
	case "consultation":
		return RequestTypeConsultation
	case "data_analysis":
		if r.NeedsVisualization {
			return RequestTypeVisualization
		}
		return RequestTypeDataQuery
	case "visualization":
		return RequestTypeVisualization
	case "data_export":
		return RequestTypeDataQuery
	case "calculation":
		return RequestTypeCalculation
	case "web_search":
		return RequestTypeWebSearch
	default:
		return RequestTypeDataQuery
	}
}

// ShouldGenerateChart returns whether the request should produce a chart
func (r *ClassificationResult) ShouldGenerateChart() bool {
	return r.NeedsVisualization || r.RequestType == "visualization" || r.RequestType == "data_analysis"
}

// ShouldExportData returns whether the request should export data
func (r *ClassificationResult) ShouldExportData() bool {
	return r.NeedsDataExport || r.RequestType == "data_export"
}
