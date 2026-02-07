package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// RequestTypeClassifier uses LLM to classify request types
type RequestTypeClassifier struct {
	chatModel model.ChatModel
	logger    func(string)
	cache     map[string]*ClassificationResult
	cacheMu   sync.RWMutex
}

// ClassificationResult represents the LLM classification result
type ClassificationResult struct {
	RequestType        string   `json:"request_type"`
	NeedsVisualization bool     `json:"needs_visualization"`
	NeedsDataExport    bool     `json:"needs_data_export"`
	Confidence         float64  `json:"confidence"`
	Reasoning          string   `json:"reasoning"`
	SuggestedOutputs   []string `json:"suggested_outputs"`
	SuggestedChartType string   `json:"suggested_chart_type"` // Recommended chart type: line, bar, pie, etc.
	CachedAt           time.Time
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
	c.cacheMu.RLock()
	if cached, ok := c.cache[cacheKey]; ok {
		if time.Since(cached.CachedAt) < 5*time.Minute {
			c.cacheMu.RUnlock()
			c.log("[CLASSIFIER-LLM] Using cached classification result")
			return cached, nil
		}
	}
	c.cacheMu.RUnlock()

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

### 判断是否需要可视化 (needs_visualization) - 重要！
⭐ **默认应该生成可视化图表**，除非是以下情况：
- 纯咨询请求（consultation）
- 简单计算请求（calculation）
- 用户明确只要文字/数字结果

✅ 以下情况**必须**设置 needs_visualization = true：
- 任何涉及"分析"、"统计"、"趋势"、"分布"、"对比"、"排名"的请求
- 涉及时间序列数据（按月、按年、按季度等）
- 涉及分类数据（按产品、按地区、按客户等）
- 涉及占比、比例、百分比的分析
- 涉及 Top N、排行榜类分析
- 涉及增长、下降、变化趋势分析
- 涉及销售、收入、利润、订单等业务指标分析

❌ 以下情况可以设置 needs_visualization = false：
- 纯粹列出数据（如"列出所有订单"）
- 查询单个数值（如"总共有多少订单"）
- 咨询建议类请求

### 判断是否需要数据导出 (needs_data_export)
- 用户明确要求导出、下载、生成报告时为true
- 分析结果较大时建议导出

### 建议的图表类型 (suggested_chart_type)
根据分析类型推荐最合适的图表：
- 时间趋势 → "line"（折线图）
- 分类对比 → "bar"（柱状图）
- 占比分布 → "pie"（饼图）
- 多维对比 → "grouped_bar"（分组柱状图）
- 相关性分析 → "scatter"（散点图）
- 热力分析 → "heatmap"（热力图）

## 输出格式 (JSON)
{
  "request_type": "consultation|data_analysis|visualization|data_export|calculation|web_search",
  "needs_visualization": true/false,
  "needs_data_export": true/false,
  "confidence": 0.0-1.0,
  "reasoning": "简短解释分类原因",
  "suggested_outputs": ["chart", "table", "insight", "excel", "pdf"],
  "suggested_chart_type": "line|bar|pie|grouped_bar|scatter|heatmap"
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
	c.cacheMu.Lock()
	c.cache[cacheKey] = result
	c.cacheMu.Unlock()

	c.log(fmt.Sprintf("[CLASSIFIER-LLM] Classification complete in %v: type=%s, viz=%v, export=%v, confidence=%.2f",
		time.Since(startTime), result.RequestType, result.NeedsVisualization, result.NeedsDataExport, result.Confidence))

	return result, nil
}

// fallbackClassification provides a fallback when LLM fails
// Default to visualization for most analysis requests
func (c *RequestTypeClassifier) fallbackClassification(query string) *ClassificationResult {
	queryLower := strings.ToLower(query)

	result := &ClassificationResult{
		RequestType:        "data_analysis",
		NeedsVisualization: true, // Default to true for better user experience
		NeedsDataExport:    false,
		Confidence:         0.5,
		Reasoning:          "Fallback classification - defaulting to visualization",
		SuggestedOutputs:   []string{"chart", "table", "insight"},
		SuggestedChartType: "bar", // Default chart type
		CachedAt:           time.Now(),
	}

	// Only disable visualization for specific non-analysis requests
	if containsAny(queryLower, []string{"建议", "可以做什么", "能做什么", "suggest", "recommend"}) &&
		!containsAny(queryLower, []string{"分析", "统计", "查询", "销售", "订单"}) {
		result.RequestType = "consultation"
		result.NeedsVisualization = false
		result.SuggestedOutputs = []string{"text"}
		result.SuggestedChartType = ""
	} else if containsAny(queryLower, []string{"导出", "下载", "报告", "export", "download"}) {
		result.RequestType = "data_export"
		result.NeedsDataExport = true
		result.SuggestedOutputs = []string{"excel", "pdf"}
	} else if containsAny(queryLower, []string{"计算", "几点", "时间", "calculate", "time"}) &&
		!containsAny(queryLower, []string{"数据", "订单", "销售"}) {
		result.RequestType = "calculation"
		result.NeedsVisualization = false
		result.SuggestedOutputs = []string{"text"}
		result.SuggestedChartType = ""
	}

	// Suggest chart type based on keywords
	if result.NeedsVisualization {
		if containsAny(queryLower, []string{"趋势", "变化", "时间", "月", "年", "季度", "trend"}) {
			result.SuggestedChartType = "line"
		} else if containsAny(queryLower, []string{"占比", "比例", "分布", "百分比", "pie"}) {
			result.SuggestedChartType = "pie"
		} else if containsAny(queryLower, []string{"对比", "比较", "排名", "top", "前"}) {
			result.SuggestedChartType = "bar"
		}
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
