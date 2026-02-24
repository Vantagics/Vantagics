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

	prompt := fmt.Sprintf(`You are a request classification expert. Analyze the user request and determine its type and required outputs.

## User Request
"%s"

## Data Source Info
%s

## Classification Rules

### Request Type (request_type)
1. **consultation** - Pure consultation/advice, user just wants to know what can be done
2. **data_analysis** - Data analysis request, needs querying and analyzing data
3. **visualization** - Explicitly needs visualization/charts
4. **data_export** - Data export request
5. **calculation** - Simple calculation (no database needed)
6. **web_search** - Needs web search

### Visualization Decision (needs_visualization)
**Default to generating charts**, unless:
- Pure consultation request
- Simple calculation request
- User explicitly only wants text/numbers

Must set needs_visualization = true when:
- Any request involving analysis, statistics, trends, distribution, comparison, ranking
- Time series data (by month, year, quarter, etc.)
- Categorical data (by product, region, customer, etc.)
- Proportions, percentages, ratios
- Top N, leaderboard analysis
- Growth, decline, trend analysis
- Business metrics (sales, revenue, profit, orders)

### Data Export (needs_data_export)
- true when user explicitly asks to export, download, or generate reports

### Chart Type (suggested_chart_type)
- Time trends �?"line"
- Category comparison �?"bar"
- Proportions �?"pie"
- Multi-dimensional �?"grouped_bar"
- Correlation �?"scatter"
- Heat analysis �?"heatmap"

## Output Format (JSON)
{
  "request_type": "consultation|data_analysis|visualization|data_export|calculation|web_search",
  "needs_visualization": true/false,
  "needs_data_export": true/false,
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation",
  "suggested_outputs": ["chart", "table", "insight", "excel", "pdf"],
  "suggested_chart_type": "line|bar|pie|grouped_bar|scatter|heatmap"
}

Output only JSON.`, userQuery, dataSourceInfo)

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "You are a request classification expert. Output only valid JSON.",
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
	if containsAny(queryLower, []string{"建议", "可以做什�?, "能做什�?, "suggest", "recommend"}) &&
		!containsAny(queryLower, []string{"分析", "统计", "查询", "销�?, "订单"}) {
		result.RequestType = "consultation"
		result.NeedsVisualization = false
		result.SuggestedOutputs = []string{"text"}
		result.SuggestedChartType = ""
	} else if containsAny(queryLower, []string{"导出", "下载", "报告", "export", "download"}) {
		result.RequestType = "data_export"
		result.NeedsDataExport = true
		result.SuggestedOutputs = []string{"excel", "pdf"}
	} else if containsAny(queryLower, []string{"计算", "几点", "时间", "calculate", "time"}) &&
		!containsAny(queryLower, []string{"数据", "订单", "销�?}) {
		result.RequestType = "calculation"
		result.NeedsVisualization = false
		result.SuggestedOutputs = []string{"text"}
		result.SuggestedChartType = ""
	}

	// Suggest chart type based on keywords
	if result.NeedsVisualization {
		if containsAny(queryLower, []string{"趋势", "变化", "时间", "�?, "�?, "季度", "trend"}) {
			result.SuggestedChartType = "line"
		} else if containsAny(queryLower, []string{"占比", "比例", "分布", "百分�?, "pie"}) {
			result.SuggestedChartType = "pie"
		} else if containsAny(queryLower, []string{"对比", "比较", "排名", "top", "�?}) {
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
