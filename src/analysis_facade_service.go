package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"vantagics/agent"
	"vantagics/config"
	"vantagics/i18n"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AnalysisManager 定义分析管理接口
type AnalysisManager interface {
	GenerateIntentSuggestions(threadID, userMessage string) ([]IntentSuggestion, error)
	GenerateIntentSuggestionsWithExclusions(threadID, userMessage string, excludedSuggestions []IntentSuggestion) ([]IntentSuggestion, error)
	ExtractMetricsFromAnalysis(threadID, messageId, analysisContent string) error
	ExtractSuggestionsFromAnalysis(threadID, userMessageID, analysisContent string) error
	ExtractSuggestionsAsItems(threadID, userMessageID, analysisContent string) []AnalysisResultItem
	SaveMetricsJson(messageId, metricsJson string) error
	LoadMetricsJson(messageId string) (string, error)
	AddAnalysisRecord(dataSourceID string, record agent.AnalysisRecord) error
	RecordIntentSelection(threadID string, intent IntentSuggestion) error
	GetMessageAnalysisData(threadID, messageID string) (map[string]interface{}, error)
	ShowStepResultOnDashboard(threadID, messageID string) error
	ShowAllSessionResults(threadID string) error
	SaveMessageAnalysisResults(threadID, messageID string, results []AnalysisResultItem) error
	SaveSessionRecording(threadID, title, description string) (string, error)
	GetSessionRecordings() ([]agent.AnalysisRecording, error)
	ReplayAnalysisRecording(recordingID, targetSourceID string, autoFixFields bool, maxFieldDiff int) (*agent.ReplayResult, error)
}

// AnalysisFacadeService 分析服务门面，封装所有分析相关的业务逻辑
type AnalysisFacadeService struct {
	ctx                        context.Context
	chatService                *ChatService
	configProvider             ConfigProvider
	dataSourceService          *agent.DataSourceService
	einoService                *agent.EinoService
	eventAggregator            *EventAggregator
	licenseClient              *agent.LicenseClient
	intentEnhancementService   *agent.IntentEnhancementService
	intentUnderstandingService *agent.IntentUnderstandingService
	storageDir                 string
	logger                     func(string)

	// readChartDataFileFn is injected from App to resolve file:// references
	readChartDataFileFn func(threadID, fileRef string) (string, error)
	// attachChartFn is injected from App/ChatFacadeService to attach chart data to messages
	attachChartFn func(threadID, messageID string, chartData *ChartData)
}

// NewAnalysisFacadeService 创建新的 AnalysisFacadeService 实例
func NewAnalysisFacadeService(
	chatService *ChatService,
	configProvider ConfigProvider,
	dataSourceService *agent.DataSourceService,
	einoService *agent.EinoService,
	eventAggregator *EventAggregator,
	storageDir string,
	logger func(string),
) *AnalysisFacadeService {
	return &AnalysisFacadeService{
		chatService:       chatService,
		configProvider:    configProvider,
		dataSourceService: dataSourceService,
		einoService:       einoService,
		eventAggregator:   eventAggregator,
		storageDir:        storageDir,
		logger:            logger,
	}
}

// Name 返回服务名称
func (s *AnalysisFacadeService) Name() string {
	return "analysis"
}

// Initialize 初始化分析门面服�
func (s *AnalysisFacadeService) Initialize(ctx context.Context) error {
	s.ctx = ctx
	s.log("AnalysisFacadeService initialized")
	return nil
}

// Shutdown 关闭分析门面服务
func (s *AnalysisFacadeService) Shutdown() error {
	s.log("AnalysisFacadeService shutdown")
	return nil
}

// SetContext sets the Wails runtime context
func (s *AnalysisFacadeService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetLicenseClient sets the license client dependency
func (s *AnalysisFacadeService) SetLicenseClient(lc *agent.LicenseClient) {
	s.licenseClient = lc
}

// SetIntentEnhancementService sets the intent enhancement service dependency
func (s *AnalysisFacadeService) SetIntentEnhancementService(ies *agent.IntentEnhancementService) {
	s.intentEnhancementService = ies
}

// SetIntentUnderstandingService sets the intent understanding service dependency
func (s *AnalysisFacadeService) SetIntentUnderstandingService(ius *agent.IntentUnderstandingService) {
	s.intentUnderstandingService = ius
}

// SetReadChartDataFileFn sets the chart data file reader function
func (s *AnalysisFacadeService) SetReadChartDataFileFn(fn func(threadID, fileRef string) (string, error)) {
	s.readChartDataFileFn = fn
}

// SetAttachChartFn sets the chart attachment function
func (s *AnalysisFacadeService) SetAttachChartFn(fn func(threadID, messageID string, chartData *ChartData)) {
	s.attachChartFn = fn
}

// SetEinoService updates the EinoService reference (used during reinitializeServices)
func (s *AnalysisFacadeService) SetEinoService(es *agent.EinoService) {
	s.einoService = es
}

// --- Intent Suggestions ---

// GenerateIntentSuggestions generates possible interpretations of user's intent
func (s *AnalysisFacadeService) GenerateIntentSuggestions(threadID, userMessage string) ([]IntentSuggestion, error) {
	return s.GenerateIntentSuggestionsWithExclusions(threadID, userMessage, nil)
}

// GenerateIntentSuggestionsWithExclusions generates possible interpretations of user's intent,
// excluding previously generated suggestions
func (s *AnalysisFacadeService) GenerateIntentSuggestionsWithExclusions(threadID, userMessage string, excludedSuggestions []IntentSuggestion) ([]IntentSuggestion, error) {
	// Check analysis limit before proceeding
	if s.licenseClient != nil && s.licenseClient.IsActivated() {
		canAnalyze, limitMsg := s.licenseClient.CanAnalyze()
		if !canAnalyze {
			return nil, fmt.Errorf("%s", limitMsg)
		}
	}

	cfg, err := s.configProvider.GetEffectiveConfig()
	if err != nil {
		return nil, err
	}

	// Get data source information for context
	var dataSourceID string
	var tableName string
	var columns []string

	if threadID != "" && s.chatService != nil {
		threads, _ := s.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				dataSourceID = t.DataSourceID
				break
			}
		}
	}

	if dataSourceID != "" && s.dataSourceService != nil {
		dataSources, err := s.dataSourceService.LoadDataSources()
		if err == nil {
			for _, ds := range dataSources {
				if ds.ID == dataSourceID {
					if ds.Analysis != nil && len(ds.Analysis.Schema) > 0 {
						tableName = ds.Analysis.Schema[0].TableName
						columns = ds.Analysis.Schema[0].Columns
					}
					break
				}
			}
		}

		// If no analysis available, try to get table info directly
		if tableName == "" {
			tables, err := s.dataSourceService.GetDataSourceTables(dataSourceID)
			if err == nil && len(tables) > 0 {
				tableName = tables[0]
				cols, err := s.dataSourceService.GetDataSourceTableColumns(dataSourceID, tableName)
				if err == nil {
					columns = cols
				}
			}
		}
	}

	// Try to use the new IntentUnderstandingService if available and enabled
	if s.intentUnderstandingService != nil && s.intentUnderstandingService.IsEnabled() {
		s.log("[INTENT] Using new IntentUnderstandingService")

		language := "en"
		if cfg.Language == "简体中�" {
			language = "zh"
		}

		agentExclusions := convertToAgentSuggestions(excludedSuggestions)

		llmCall := func(ctx context.Context, prompt string) (string, error) {
			llm := agent.NewLLMService(cfg, s.log)
			return llm.Chat(ctx, prompt)
		}

		agentSuggestions, err := s.intentUnderstandingService.GenerateSuggestions(
			s.ctx,
			threadID,
			userMessage,
			dataSourceID,
			language,
			agentExclusions,
			llmCall,
		)
		if err != nil {
			s.log(fmt.Sprintf("[INTENT] IntentUnderstandingService failed: %v, falling back to old implementation", err))
		} else {
			suggestions := convertAgentSuggestions(agentSuggestions)
			s.log(fmt.Sprintf("[INTENT] Generated %d suggestions using new service", len(suggestions)))
			if s.licenseClient != nil && s.licenseClient.IsActivated() {
				s.licenseClient.IncrementAnalysis()
				s.log("[LICENSE] Analysis count incremented for successful intent suggestion generation")
			}
			return suggestions, nil
		}
	}

	// Fallback to old implementation using IntentEnhancementService
	s.log("[INTENT] Using legacy IntentEnhancementService")

	// Check cache for similar requests - skip cache when there are exclusions
	if s.intentEnhancementService != nil && dataSourceID != "" && len(excludedSuggestions) == 0 {
		cachedSuggestions, cacheHit := s.intentEnhancementService.GetCachedSuggestions(dataSourceID, userMessage)
		if cacheHit && len(cachedSuggestions) > 0 {
			s.log("[INTENT] Cache hit for intent suggestions (no exclusions)")
			suggestions := make([]IntentSuggestion, len(cachedSuggestions))
			for i, cs := range cachedSuggestions {
				suggestions[i] = IntentSuggestion{
					ID:          cs.ID,
					Title:       cs.Title,
					Description: cs.Description,
					Icon:        cs.Icon,
					Query:       cs.Query,
				}
			}
			// Apply preference ranking even for cached results
			agentSuggestions := make([]agent.IntentSuggestion, len(suggestions))
			for i, ss := range suggestions {
				agentSuggestions[i] = agent.IntentSuggestion{
					ID:          ss.ID,
					Title:       ss.Title,
					Description: ss.Description,
					Icon:        ss.Icon,
					Query:       ss.Query,
				}
			}
			rankedSuggestions := s.intentEnhancementService.RankSuggestions(dataSourceID, agentSuggestions)
			for i, rs := range rankedSuggestions {
				suggestions[i] = IntentSuggestion{
					ID:          rs.ID,
					Title:       rs.Title,
					Description: rs.Description,
					Icon:        rs.Icon,
					Query:       rs.Query,
				}
			}
			return suggestions, nil
		}
	} else if len(excludedSuggestions) > 0 {
		s.log(fmt.Sprintf("[INTENT] Skipping cache - retry with %d exclusions, will call LLM for fresh suggestions", len(excludedSuggestions)))
	}

	// Create ExclusionSummarizer and check if summarization is needed
	summarizer := agent.NewExclusionSummarizer()
	var exclusionSummary string

	exclusionIntents := make([]agent.ExclusionIntentSuggestion, len(excludedSuggestions))
	for i, es := range excludedSuggestions {
		exclusionIntents[i] = agent.ExclusionIntentSuggestion{
			ID:          es.ID,
			Title:       es.Title,
			Description: es.Description,
			Icon:        es.Icon,
			Query:       es.Query,
		}
	}

	if summarizer.NeedsSummarization(exclusionIntents) {
		exclusionSummary = summarizer.SummarizeExclusions(exclusionIntents)
		s.log(fmt.Sprintf("[INTENT] Using exclusion summary for %d excluded suggestions (threshold: %d)",
			len(excludedSuggestions), summarizer.GetThreshold()))
	}

	// Build prompt for LLM
	prompt := s.buildIntentUnderstandingPrompt(userMessage, tableName, columns, cfg.Language, excludedSuggestions, dataSourceID, exclusionSummary)

	// Call LLM
	llm := agent.NewLLMService(cfg, s.log)
	response, err := llm.Chat(s.ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate intent suggestions: %v", err)
	}

	// Parse response
	suggestions, err := s.parseIntentSuggestions(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent suggestions: %v", err)
	}

	// Apply preference ranking
	if s.intentEnhancementService != nil && dataSourceID != "" && len(suggestions) > 0 {
		agentSuggestions := make([]agent.IntentSuggestion, len(suggestions))
		for i, ss := range suggestions {
			agentSuggestions[i] = agent.IntentSuggestion{
				ID:          ss.ID,
				Title:       ss.Title,
				Description: ss.Description,
				Icon:        ss.Icon,
				Query:       ss.Query,
			}
		}

		rankedSuggestions := s.intentEnhancementService.RankSuggestions(dataSourceID, agentSuggestions)

		for i, rs := range rankedSuggestions {
			suggestions[i] = IntentSuggestion{
				ID:          rs.ID,
				Title:       rs.Title,
				Description: rs.Description,
				Icon:        rs.Icon,
				Query:       rs.Query,
			}
		}

		s.intentEnhancementService.CacheSuggestions(dataSourceID, userMessage, agentSuggestions)
	}

	// Increment analysis count on successful suggestion generation
	if s.licenseClient != nil && s.licenseClient.IsActivated() {
		s.licenseClient.IncrementAnalysis()
		s.log("[LICENSE] Analysis count incremented for successful intent suggestion generation (legacy)")
	}

	return suggestions, nil
}

// buildIntentUnderstandingPrompt builds the prompt for intent understanding
func (s *AnalysisFacadeService) buildIntentUnderstandingPrompt(userMessage, tableName string, columns []string, language string, excludedSuggestions []IntentSuggestion, dataSourceID string, exclusionSummary string) string {
	outputLangInstruction := "Respond in English"
	langCode := "en"
	if language == "简体中�" {
		outputLangInstruction = "用简体中文回�"
		langCode = "zh"
	}

	columnsStr := strings.Join(columns, ", ")
	if columnsStr == "" {
		columnsStr = "No schema information available"
	}
	if tableName == "" {
		tableName = "Unknown"
	}

	// Build excluded suggestions section
	excludedSection := ""
	retryGuidance := ""
	if len(excludedSuggestions) > 0 {
		excludedSection = "\n\n## Previously Rejected Interpretations\n"
		excludedSection += "The user has indicated that the following interpretations DO NOT match their intent:\n\n"

		if exclusionSummary != "" {
			excludedSection += exclusionSummary + "\n"
		} else {
			for i, suggestion := range excludedSuggestions {
				excludedSection += fmt.Sprintf("%d. **%s**: %s\n", i+1, suggestion.Title, suggestion.Description)
			}
		}
		retryGuidance = `

## Critical Instruction for Retry
The user rejected ALL previous suggestions. This means:
1. Your previous interpretations were off-target
2. You need to think from COMPLETELY DIFFERENT angles
3. Consider alternative meanings, contexts, or analysis approaches
4. Avoid similar patterns or themes from rejected suggestions
5. Be more creative and explore edge cases or unconventional interpretations`
	}

	stickToOriginalGuidance := ""
	if language == "简体中�" {
		stickToOriginalGuidance = `

# 关于"坚持我的请求"选项
用户可以选择"坚持我的请求"来直接使用他们的原始输入进行分析。因此：
1. 你的建议应该提供与原始请求不同的分析角度
2. 如果原始请求已经足够具体，你的建议应该探索相关但不同的分析方�
3. 不要简单地重复或轻微改写用户的原始请求
4. 每个建议都应该为用户提供独特的价值`
	} else {
		stickToOriginalGuidance = `

# About "Stick to My Request" Option
The user can choose "Stick to My Request" to use their original input directly for analysis. Therefore:
1. Your suggestions should offer different analytical angles from the original request
2. If the original request is already specific, your suggestions should explore related but different analysis directions
3. Do not simply repeat or slightly rephrase the user's original request
4. Each suggestion should provide unique value to the user`
	}

	basePrompt := fmt.Sprintf(`# Role
You are an expert data analysis intent interpreter. Your task is to understand ambiguous user requests and generate multiple plausible interpretations.

# User's Request
"%s"

# Available Data Context
- **Table**: %s
- **Columns**: %s%s%s%s

# Task
Generate 3-5 distinct interpretations of the user's intent. Each interpretation should:
1. Represent a different analytical perspective or approach
2. Be specific and actionable
3. Align with the available data structure
4. Be sorted by likelihood (most probable first)

# Interpretation Dimensions to Consider
- **Temporal Analysis**: Trends over time, period comparisons, seasonality
- **Segmentation**: By category, region, product, customer type, etc.
- **Aggregation Level**: Summary statistics, detailed breakdowns, rankings
- **Comparison**: Year-over-year, benchmarking, A/B testing
- **Correlation**: Relationships between variables, cause-effect analysis
- **Anomaly Detection**: Outliers, unusual patterns, exceptions
- **Forecasting**: Predictions, projections, what-if scenarios

# Output Format
Return a JSON array with 3-5 interpretations. Each object must include:

[
  {
    "title": "Short descriptive title (max 10 words)",
    "description": "Clear explanation of what this interpretation means (max 30 words)",
    "icon": "Relevant emoji (📊, 📈, 📉, 🔍, 💡, 📅, 🎯, etc.)",
    "query": "Specific, detailed analysis request that can be executed (be explicit about metrics, dimensions, and filters)"
  }
]

# Quality Requirements
- **Specificity**: Each query should be detailed enough to execute without ambiguity
- **Diversity**: Interpretations should cover different analytical angles
- **Feasibility**: Only suggest analyses that can be performed with the available columns
- **Clarity**: Descriptions should be clear and jargon-free
- **Language**: %s

# Output Rules
- Return ONLY the JSON array
- No markdown code blocks, no explanations, no additional text
- Ensure valid JSON syntax
- Start with [ and end with ]

Generate the interpretations now:`, userMessage, tableName, columnsStr, excludedSection, retryGuidance, stickToOriginalGuidance, outputLangInstruction)

	// Enhance prompt with context, dimensions, and examples using IntentEnhancementService
	if s.intentEnhancementService != nil && dataSourceID != "" {
		columnSchemas := make([]agent.ColumnSchema, len(columns))
		for i, col := range columns {
			columnSchemas[i] = agent.ColumnSchema{Name: col}
		}

		enhancedPrompt, err := s.intentEnhancementService.EnhancePromptWithColumns(
			s.ctx,
			basePrompt,
			dataSourceID,
			userMessage,
			langCode,
			columnSchemas,
			tableName,
		)
		if err != nil {
			s.log(fmt.Sprintf("[INTENT] Failed to enhance prompt: %v, using base prompt", err))
			return basePrompt
		}
		return enhancedPrompt
	}

	return basePrompt
}

// parseIntentSuggestions parses LLM response into IntentSuggestion slice
func (s *AnalysisFacadeService) parseIntentSuggestions(response string) ([]IntentSuggestion, error) {
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")

	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no valid JSON array found in response")
	}

	jsonStr := response[start : end+1]

	var rawSuggestions []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawSuggestions); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	suggestions := make([]IntentSuggestion, 0, len(rawSuggestions))
	for i, raw := range rawSuggestions {
		suggestion := IntentSuggestion{
			ID:          fmt.Sprintf("intent_%d_%d", time.Now().Unix(), i),
			Title:       getStringFromMap(raw, "title"),
			Description: getStringFromMap(raw, "description"),
			Icon:        getStringFromMap(raw, "icon"),
			Query:       getStringFromMap(raw, "query"),
		}

		if suggestion.Title != "" && suggestion.Query != "" {
			suggestions = append(suggestions, suggestion)
		}
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no valid suggestions generated")
	}

	return suggestions, nil
}

// getStringFromMap safely extracts a string value from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// --- Metrics Extraction ---

// SaveMetricsJson saves metrics JSON data for a specific message
func (s *AnalysisFacadeService) SaveMetricsJson(messageId string, metricsJson string) error {
	metricsDir := filepath.Join(s.storageDir, "data", "metrics")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		return fmt.Errorf("failed to create metrics directory: %w", err)
	}

	filePath := filepath.Join(metricsDir, fmt.Sprintf("%s.json", messageId))

	if err := os.WriteFile(filePath, []byte(metricsJson), 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	s.log(fmt.Sprintf("Metrics JSON saved for message %s: %s", messageId, filePath))
	return nil
}

// LoadMetricsJson loads metrics JSON data for a specific message
func (s *AnalysisFacadeService) LoadMetricsJson(messageId string) (string, error) {
	filePath := filepath.Join(s.storageDir, "data", "metrics", fmt.Sprintf("%s.json", messageId))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("metrics file not found for message: %s", messageId)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read metrics file: %w", err)
	}

	s.log(fmt.Sprintf("Metrics JSON loaded for message %s: %s", messageId, filePath))
	return string(data), nil
}

// ExtractMetricsFromAnalysis automatically extracts key metrics from analysis results
func (s *AnalysisFacadeService) ExtractMetricsFromAnalysis(threadID string, messageId string, analysisContent string) error {
	cfg, err := s.configProvider.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	var prompt string
	if cfg.Language == "简体中�" {
		prompt = fmt.Sprintf(`请从以下分析结果中提取最重要的数值型关键指标，以JSON格式返回�

要求�
1. 只返回JSON数组，不要其他文字说�
2. 每个指标必须包含：name（指标名称）、value（数值）、unit（单位，可选）
3. **重要**：只提取数值型指标，value必须是数字或包含数字的字符串
4. **重要**：如果分析结果中没有明确的数值型指标，返回空数组 []
5. 最多提�个最重要的业务指�
6. 优先提取：总量、增长率、平均值、比率、金额、数量等核心业务指标
7. 数值要准确，来源于分析内容
8. 单位要合适（如：个、%%、元、次/年、天等）
9. 指标名称要简洁明�
10. 不要提取非数值型的描述性内�

示例格式（有数值指标时）：
[
  {"name":"总销售额","value":"1,234,567","unit":"�"},
  {"name":"增长�","value":"+15.5","unit":"%%"},
  {"name":"平均订单价�","value":"89.50","unit":"�"}
]

示例格式（无数值指标时）：
[]

分析内容�
%s

请返回JSON：`, analysisContent)
	} else {
		prompt = fmt.Sprintf(`Please extract the most important numerical key metrics from the following analysis results in JSON format.

Requirements:
1. Return only JSON array, no other text
2. Each metric must include: name, value, unit (optional)
3. **Important**: Only extract numerical metrics, value must be a number or string containing numbers
4. **Important**: If there are no clear numerical metrics in the analysis, return empty array []
5. Extract at most 6 most important business metrics
6. Prioritize: totals, growth rates, averages, ratios, amounts, quantities and other core business metrics
7. Values must be accurate from the analysis content
8. Use appropriate units (e.g., items, %%, $, times/year, days, etc.)
9. Metric names should be concise and clear
10. Do not extract non-numerical descriptive content

Example format (with numerical metrics):
[
  {"name":"Total Sales","value":"1,234,567","unit":"$"},
  {"name":"Growth Rate","value":"+15.5","unit":"%%"},
  {"name":"Average Order Value","value":"89.50","unit":"$"}
]

Example format (without numerical metrics):
[]

Analysis content:
%s

Please return JSON:`, analysisContent)
	}

	// Try extraction up to 3 times
	for attempt := 1; attempt <= 3; attempt++ {
		err := s.tryExtractMetrics(threadID, messageId, prompt, attempt)
		if err == nil {
			return nil
		}

		s.log(fmt.Sprintf("Metrics extraction attempt %d failed: %v", attempt, err))

		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	// If all attempts fail, use fallback text extraction
	return s.fallbackTextExtraction(messageId, analysisContent)
}

// tryExtractMetrics attempts to extract metrics using LLM
func (s *AnalysisFacadeService) tryExtractMetrics(threadID string, messageId string, prompt string, attempt int) error {
	llm := agent.NewLLMService(s.getConfigForExtraction(), s.log)
	response, err := llm.Chat(s.ctx, prompt)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	jsonStr := s.extractJSONFromResponse(response)
	if jsonStr == "" {
		return fmt.Errorf("no valid JSON found in LLM response")
	}

	var metrics []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &metrics); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Allow empty array - no numerical metrics found
	if len(metrics) == 0 {
		s.log("No numerical metrics found in analysis, skipping metrics extraction")
		return nil
	}

	// Validate each metric has required fields and contains numerical value
	validMetrics := []map[string]interface{}{}
	for i, metric := range metrics {
		name, hasName := metric["name"]
		value, hasValue := metric["value"]

		if !hasName {
			s.log(fmt.Sprintf("Metric %d missing 'name' field, skipping", i))
			continue
		}
		if !hasValue {
			s.log(fmt.Sprintf("Metric %d missing 'value' field, skipping", i))
			continue
		}

		valueStr := fmt.Sprintf("%v", value)
		if !containsNumber(valueStr) {
			s.log(fmt.Sprintf("Metric %d (%s) value '%s' does not contain numbers, skipping", i, name, valueStr))
			continue
		}

		validMetrics = append(validMetrics, metric)
	}

	if len(validMetrics) == 0 {
		s.log("No valid numerical metrics after validation, skipping metrics extraction")
		return nil
	}

	validMetricsJSON, err := json.Marshal(validMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal valid metrics: %w", err)
	}
	jsonStr = string(validMetricsJSON)

	// Save metrics JSON
	if err := s.SaveMetricsJson(messageId, jsonStr); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	// Mark the user message with chart_data so frontend knows it has data
	if threadID != "" && s.attachChartFn != nil {
		s.attachChartFn(threadID, messageId, &ChartData{
			Charts: []ChartItem{{Type: "metrics", Data: ""}},
		})
	}

	// Use event aggregator for unified event system
	if s.eventAggregator != nil {
		for _, metric := range validMetrics {
			m := Metric{
				Title:  fmt.Sprintf("%v", metric["name"]),
				Value:  fmt.Sprintf("%v", metric["value"]),
				Change: "",
			}
			if unit, ok := metric["unit"]; ok {
				m.Value = fmt.Sprintf("%v%v", metric["value"], unit)
			}
			if change, ok := metric["change"]; ok {
				m.Change = fmt.Sprintf("%v", change)
			}
			s.eventAggregator.AddMetric(threadID, messageId, "", m)
		}
		s.eventAggregator.FlushNow(threadID, false)
	}

	s.log(fmt.Sprintf("Metrics extracted and saved for message %s (attempt %d)", messageId, attempt))
	return nil
}

// getConfigForExtraction gets config for metrics extraction
func (s *AnalysisFacadeService) getConfigForExtraction() config.Config {
	cfg, _ := s.configProvider.GetEffectiveConfig()
	return cfg
}

// extractJSONFromResponse extracts JSON array from LLM response
func (s *AnalysisFacadeService) extractJSONFromResponse(response string) string {
	jsonPattern := regexp.MustCompile(`\[[\s\S]*?\]`)
	matches := jsonPattern.FindAllString(response, -1)

	for _, match := range matches {
		var test []interface{}
		if json.Unmarshal([]byte(match), &test) == nil {
			return match
		}
	}

	return ""
}

// fallbackTextExtraction uses regex patterns as fallback when LLM extraction fails
func (s *AnalysisFacadeService) fallbackTextExtraction(messageId string, content string) error {
	metrics := []map[string]interface{}{}

	patterns := []struct {
		regex *regexp.Regexp
		name  string
		unit  string
	}{
		{regexp.MustCompile(`�*?[�]?\s*(\d+(?:,\d{3})*(?:\.\d+)?)`), "总计", ""},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)%`), "百分�", "%"},
		{regexp.MustCompile(`\$(\d+(?:,\d{3})*(?:\.\d+)?)`), "金额", "$"},
		{regexp.MustCompile(`平均.*?[�]?\s*(\d+(?:\.\d+)?)`), "平均�", ""},
		{regexp.MustCompile(`增长.*?[�]?\s*([+\-]?\d+(?:\.\d+)?)%`), "增长�", "%"},
	}

	for _, pattern := range patterns {
		matches := pattern.regex.FindAllStringSubmatch(content, -1)
		for i, match := range matches {
			if len(match) > 1 && len(metrics) < 6 {
				metrics = append(metrics, map[string]interface{}{
					"name":  fmt.Sprintf("%s%d", pattern.name, i+1),
					"value": match[1],
					"unit":  pattern.unit,
				})
			}
		}
	}

	if len(metrics) > 0 {
		jsonStr, _ := json.Marshal(metrics)
		err := s.SaveMetricsJson(messageId, string(jsonStr))
		if err == nil {
			if s.eventAggregator != nil {
				for _, metric := range metrics {
					m := Metric{
						Title:  fmt.Sprintf("%v", metric["name"]),
						Value:  fmt.Sprintf("%v", metric["value"]),
						Change: "",
					}
					if unit, ok := metric["unit"]; ok {
						m.Value = fmt.Sprintf("%v%v", metric["value"], unit)
					}
					s.eventAggregator.AddMetric("", messageId, "", m)
				}
				s.eventAggregator.FlushNow("", false)
			}
			s.log(fmt.Sprintf("Fallback metrics extracted for message %s", messageId))
		}
		return err
	}

	return fmt.Errorf("no metrics could be extracted using fallback method")
}

// --- Suggestion Extraction ---

// ExtractSuggestionsFromAnalysis extracts next-step suggestions from analysis response
// and emits them to the dashboard insights area
func (s *AnalysisFacadeService) ExtractSuggestionsFromAnalysis(threadID, userMessageID, analysisContent string) error {
	insights := s.extractSuggestionInsights(analysisContent)

	if len(insights) > 0 {
		s.log(fmt.Sprintf("[SUGGESTIONS] Extracted %d suggestions from analysis for message %s", len(insights), userMessageID))

		if s.eventAggregator != nil {
			for _, insight := range insights {
				s.eventAggregator.AddInsight(threadID, userMessageID, "", insight)
			}
			s.eventAggregator.FlushNow(threadID, false)
		}
	}

	return nil
}

// extractSuggestionInsights is the shared extraction logic for suggestions.
func (s *AnalysisFacadeService) extractSuggestionInsights(analysisContent string) []Insight {
	if analysisContent == "" {
		return nil
	}

	var insights []Insight
	lines := strings.Split(analysisContent, "\n")

	numberPattern := regexp.MustCompile(`^\s*\*{0,2}(\d+)[.�]\*{0,2}\s*(.+)`)
	listPattern := regexp.MustCompile(`^\s*[-•]\s+(.+)`)
	boldTitlePattern := regexp.MustCompile(`^\s*\*\*(.+?)\*\*\s*[�\-–—]\s*(.+)`)

	suggestionPattern := regexp.MustCompile(`(?i)(建议|suggest|recommend|next|further|深入|可以进一步|后续|下一步|洞察|insight|分析方向|可以从|希望从哪)`)

	inCodeBlock := false
	foundSuggestionSection := false
	consecutiveBoldItems := 0

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock || trimmedLine == "" {
			continue
		}

		if suggestionPattern.MatchString(trimmedLine) {
			foundSuggestionSection = true
		}

		var suggestionText string

		// Strategy 1: Numbered items (only in suggestion section)
		if matches := numberPattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			if foundSuggestionSection {
				suggestionText = strings.TrimSpace(matches[2])
			}
		}

		// Strategy 2: Bold title with colon/dash
		if suggestionText == "" {
			if matches := boldTitlePattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
				title := strings.TrimSpace(matches[1])
				desc := strings.TrimSpace(matches[2])
				if desc != "" {
					suggestionText = title + "：" + desc
				} else {
					suggestionText = title
				}
				consecutiveBoldItems++
				if consecutiveBoldItems >= 3 {
					foundSuggestionSection = true
				}
			} else {
				consecutiveBoldItems = 0
			}
		}

		// Strategy 3: Markdown list items (only in suggestion section)
		if suggestionText == "" && foundSuggestionSection {
			if matches := listPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
				suggestionText = strings.TrimSpace(matches[1])
			}
		}

		// Clean up markdown formatting
		if suggestionText != "" {
			suggestionText = strings.ReplaceAll(suggestionText, "**", "")
			suggestionText = strings.TrimSpace(suggestionText)
		}

		if suggestionText != "" && len([]rune(suggestionText)) > 5 {
			insights = append(insights, Insight{
				Text: suggestionText,
				Icon: "lightbulb",
			})
		}
	}

	if len(insights) > 9 {
		insights = insights[:9]
	}

	return insights
}

// ExtractSuggestionsAsItems extracts suggestions from analysis response,
// emits them to the frontend, and returns them as AnalysisResultItems for persistence.
func (s *AnalysisFacadeService) ExtractSuggestionsAsItems(threadID, userMessageID, analysisContent string) []AnalysisResultItem {
	insights := s.extractSuggestionInsights(analysisContent)

	if len(insights) == 0 {
		return nil
	}

	s.log(fmt.Sprintf("[SUGGESTIONS] Extracted %d suggestions from analysis for message %s", len(insights), userMessageID))

	var items []AnalysisResultItem
	if s.eventAggregator != nil {
		for _, insight := range insights {
			s.eventAggregator.AddInsight(threadID, userMessageID, "", insight)
		}
		items = s.eventAggregator.FlushNow(threadID, true)
	}

	return items
}

// --- Analysis Type Detection ---

// detectAnalysisType detects the type of analysis from the response content
func (s *AnalysisFacadeService) detectAnalysisType(response string) string {
	responseLower := strings.ToLower(response)

	if strings.Contains(responseLower, "trend") || strings.Contains(responseLower, "趋势") ||
		strings.Contains(responseLower, "over time") || strings.Contains(responseLower, "随时�") {
		return "trend"
	}
	if strings.Contains(responseLower, "comparison") || strings.Contains(responseLower, "对比") ||
		strings.Contains(responseLower, "compare") || strings.Contains(responseLower, "比较") {
		return "comparison"
	}
	if strings.Contains(responseLower, "distribution") || strings.Contains(responseLower, "分布") ||
		strings.Contains(responseLower, "breakdown") || strings.Contains(responseLower, "构成") {
		return "distribution"
	}
	if strings.Contains(responseLower, "correlation") || strings.Contains(responseLower, "相关") ||
		strings.Contains(responseLower, "relationship") || strings.Contains(responseLower, "关系") {
		return "correlation"
	}
	if strings.Contains(responseLower, "total") || strings.Contains(responseLower, "sum") ||
		strings.Contains(responseLower, "average") || strings.Contains(responseLower, "汇�") ||
		strings.Contains(responseLower, "平均") {
		return "aggregation"
	}
	if strings.Contains(responseLower, "ranking") || strings.Contains(responseLower, "排名") ||
		strings.Contains(responseLower, "top") || strings.Contains(responseLower, "�") {
		return "ranking"
	}
	if strings.Contains(responseLower, "time series") || strings.Contains(responseLower, "时间序列") ||
		strings.Contains(responseLower, "forecast") || strings.Contains(responseLower, "预测") {
		return "time_series"
	}
	if strings.Contains(responseLower, "geographic") || strings.Contains(responseLower, "地理") ||
		strings.Contains(responseLower, "region") || strings.Contains(responseLower, "区域") ||
		strings.Contains(responseLower, "province") || strings.Contains(responseLower, "省份") {
		return "geographic"
	}

	return "statistical"
}

// extractKeyFindings extracts key findings from the analysis response
func (s *AnalysisFacadeService) extractKeyFindings(response string) string {
	findingsKeywords := []string{
		"关键发现", "主要发现", "结论", "总结",
		"Key Findings", "Key findings", "Conclusion", "Summary",
		"发现", "结果", "insights", "Insights",
	}

	for _, keyword := range findingsKeywords {
		idx := strings.Index(response, keyword)
		if idx != -1 {
			start := idx
			end := start + 200
			if end > len(response) {
				end = len(response)
			}

			excerpt := response[start:end]
			excerpt = strings.TrimSpace(excerpt)
			if len(excerpt) > 150 {
				lastPeriod := strings.LastIndex(excerpt[:150], "�")
				if lastPeriod == -1 {
					lastPeriod = strings.LastIndex(excerpt[:150], ".")
				}
				if lastPeriod > 50 {
					excerpt = excerpt[:lastPeriod+1]
				} else {
					excerpt = excerpt[:150] + "..."
				}
			}

			return excerpt
		}
	}

	if len(response) > 150 {
		excerpt := response[:150]
		lastPeriod := strings.LastIndex(excerpt, "�")
		if lastPeriod == -1 {
			lastPeriod = strings.LastIndex(excerpt, ".")
		}
		if lastPeriod > 30 {
			return excerpt[:lastPeriod+1]
		}
		return excerpt + "..."
	}

	return response
}

// extractTargetColumns extracts target columns mentioned in the analysis
func (s *AnalysisFacadeService) extractTargetColumns(response string, availableColumns []string) []string {
	responseLower := strings.ToLower(response)
	targetColumns := []string{}

	for _, col := range availableColumns {
		colLower := strings.ToLower(col)
		if strings.Contains(responseLower, colLower) {
			targetColumns = append(targetColumns, col)
		}
	}

	if len(targetColumns) > 5 {
		targetColumns = targetColumns[:5]
	}

	return targetColumns
}

// --- Analysis History ---

// recordAnalysisHistory records analysis completion for intent enhancement
func (s *AnalysisFacadeService) recordAnalysisHistory(dataSourceID string, record agent.AnalysisRecord) {
	if s.intentEnhancementService == nil {
		return
	}
	s.AddAnalysisRecord(dataSourceID, record)
}

// AddAnalysisRecord adds an analysis record for intent enhancement
func (s *AnalysisFacadeService) AddAnalysisRecord(dataSourceID string, record agent.AnalysisRecord) error {
	if s.intentEnhancementService == nil {
		return fmt.Errorf("intent enhancement service not initialized")
	}

	if record.DataSourceID == "" {
		record.DataSourceID = dataSourceID
	}

	err := s.intentEnhancementService.AddAnalysisRecord(record)
	if err != nil {
		s.log(fmt.Sprintf("[INTENT-HISTORY] Failed to record analysis: %v", err))
		return err
	}

	s.log(fmt.Sprintf("[INTENT-HISTORY] Successfully recorded analysis: type=%s, columns=%v, findings=%s",
		record.AnalysisType, record.TargetColumns, record.KeyFindings))

	return nil
}

// RecordIntentSelection records user's intent selection for preference learning
func (s *AnalysisFacadeService) RecordIntentSelection(threadID string, intent IntentSuggestion) error {
	var dataSourceID string
	if threadID != "" && s.chatService != nil {
		threads, _ := s.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				dataSourceID = t.DataSourceID
				break
			}
		}
	}

	if dataSourceID == "" {
		return fmt.Errorf("no data source associated with thread")
	}

	agentIntent := agent.IntentSuggestion{
		ID:          intent.ID,
		Title:       intent.Title,
		Description: intent.Description,
		Icon:        intent.Icon,
		Query:       intent.Query,
	}

	if s.intentUnderstandingService != nil {
		if err := s.intentUnderstandingService.RecordSelection(dataSourceID, agentIntent); err != nil {
			s.log(fmt.Sprintf("[INTENT] Failed to record selection in IntentUnderstandingService: %v", err))
		}
	}

	if s.intentEnhancementService != nil {
		s.intentEnhancementService.RecordSelection(dataSourceID, agentIntent)
	}

	s.log(fmt.Sprintf("[INTENT] Recorded intent selection: %s for data source: %s", intent.Title, dataSourceID))

	return nil
}

// --- Dashboard Display ---

// GetMessageAnalysisData retrieves analysis data for a specific message (for dashboard restoration)
func (s *AnalysisFacadeService) GetMessageAnalysisData(threadID, messageID string) (map[string]interface{}, error) {
	if s.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}
	result, err := s.chatService.GetMessageAnalysisData(threadID, messageID)
	if err != nil {
		return nil, err
	}

	// Resolve file:// references in legacy ChartData
	if chartData, ok := result["chartData"]; chartData != nil && ok {
		if cd, ok := chartData.(*ChartData); ok && cd != nil {
			for i := range cd.Charts {
				if strings.HasPrefix(cd.Charts[i].Data, "file://") {
					if s.readChartDataFileFn != nil {
						resolved, readErr := s.readChartDataFileFn(threadID, cd.Charts[i].Data)
						if readErr != nil {
							s.log(fmt.Sprintf("[RESTORE] Failed to resolve file ref %s: %v", cd.Charts[i].Data, readErr))
						} else {
							s.log(fmt.Sprintf("[RESTORE] Resolved file ref %s (%d bytes)", cd.Charts[i].Data, len(resolved)))
							cd.Charts[i].Data = resolved
						}
					}
				}
			}
		}
	}

	// Resolve file:// references in AnalysisResults
	if items, ok := result["analysisResults"]; items != nil && ok {
		if resultItems, ok := items.([]AnalysisResultItem); ok {
			for i := range resultItems {
				if strData, ok := resultItems[i].Data.(string); ok && strings.HasPrefix(strData, "file://") {
					if s.readChartDataFileFn != nil {
						resolved, readErr := s.readChartDataFileFn(threadID, strData)
						if readErr != nil {
							s.log(fmt.Sprintf("[RESTORE] Failed to resolve analysis file ref %s: %v", strData, readErr))
						} else {
							s.log(fmt.Sprintf("[RESTORE] Resolved analysis file ref %s (%d bytes)", strData, len(resolved)))
							resultItems[i].Data = resolved
						}
					}
				}
			}
			result["analysisResults"] = resultItems
		}
	}

	return result, nil
}

// ShowStepResultOnDashboard re-pushes a step's analysis results to the dashboard via EventAggregator.
func (s *AnalysisFacadeService) ShowStepResultOnDashboard(threadID string, messageID string) error {
	if s.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}
	if s.eventAggregator == nil {
		return fmt.Errorf("event aggregator not initialized")
	}

	analysisData, err := s.GetMessageAnalysisData(threadID, messageID)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("dashboard.message_not_found", err))
	}

	pushed := false

	if items, ok := analysisData["analysisResults"]; items != nil && ok {
		if resultItems, ok := items.([]AnalysisResultItem); ok {
			for _, item := range resultItems {
				switch item.Type {
				case "table", "echarts", "image", "metric", "insight":
					s.eventAggregator.AddItem(threadID, messageID, "", item.Type, item.Data, item.Metadata)
					pushed = true
				}
			}
		}
	}

	// Also check for legacy chart data
	if chartData, ok := analysisData["chartData"]; chartData != nil && ok {
		if cd, ok := chartData.(*ChartData); ok && cd != nil {
			for _, chart := range cd.Charts {
				if chart.Data != "" {
					s.eventAggregator.AddECharts(threadID, messageID, "", chart.Data)
					pushed = true
				}
			}
		}
	}

	// Fallback: extract from message content directly
	if !pushed {
		thread, err := s.chatService.LoadThread(threadID)
		if err == nil && thread != nil {
			for _, msg := range thread.Messages {
				if msg.ID == messageID && msg.Role == "assistant" {
					stepDesc := extractStepDescriptionFromContent(msg.Content)
					extracted := s.chatService.extractAnalysisItemsFromContent(msg.Content, threadID, messageID)
					for _, item := range extracted {
						if stepDesc != "" {
							if item.Metadata == nil {
								item.Metadata = map[string]interface{}{}
							}
							item.Metadata["step_description"] = stepDesc
						}
						s.eventAggregator.AddItem(threadID, messageID, "", item.Type, item.Data, item.Metadata)
						pushed = true
					}
					break
				}
			}
		}
	}

	if !pushed {
		return fmt.Errorf("%s", i18n.T("dashboard.step_no_results"))
	}

	s.eventAggregator.FlushNow(threadID, false)
	s.log(fmt.Sprintf("[SHOW-RESULT] Re-pushed results for message %s in thread %s", messageID, threadID))
	return nil
}

// ShowAllSessionResults pushes all analysis results from a session to the dashboard.
func (s *AnalysisFacadeService) ShowAllSessionResults(threadID string) error {
	if s.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}
	if s.eventAggregator == nil {
		return fmt.Errorf("event aggregator not initialized")
	}

	thread, err := s.chatService.LoadThread(threadID)
	if err != nil {
		return fmt.Errorf("failed to load thread: %w", err)
	}
	if thread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	unifiedMessageID := "all_results_" + threadID

	pushed := 0
	for _, msg := range thread.Messages {
		if msg.Role != "assistant" {
			continue
		}
		if strings.Contains(msg.Content, "�") || strings.Contains(msg.Content, "⏭️") {
			continue
		}
		if !strings.Contains(msg.Content, "�") {
			continue
		}

		msgPushed := 0
		analysisData, err := s.GetMessageAnalysisData(threadID, msg.ID)
		if err == nil && analysisData != nil {
			if items, ok := analysisData["analysisResults"]; items != nil && ok {
				if resultItems, ok := items.([]AnalysisResultItem); ok {
					for _, item := range resultItems {
						s.eventAggregator.AddItem(threadID, unifiedMessageID, "", item.Type, item.Data, item.Metadata)
						msgPushed++
					}
				}
			}
			if chartData, ok := analysisData["chartData"]; chartData != nil && ok {
				if cd, ok := chartData.(*ChartData); ok && cd != nil {
					for _, chart := range cd.Charts {
						if chart.Data != "" {
							s.eventAggregator.AddECharts(threadID, unifiedMessageID, "", chart.Data)
							msgPushed++
						}
					}
				}
			}
		}

		// Fallback: extract from message content
		if msgPushed == 0 {
			stepDesc := extractStepDescriptionFromContent(msg.Content)
			extracted := s.chatService.extractAnalysisItemsFromContent(msg.Content, threadID, msg.ID)
			for _, item := range extracted {
				if stepDesc != "" {
					if item.Metadata == nil {
						item.Metadata = map[string]interface{}{}
					}
					item.Metadata["step_description"] = stepDesc
				}
				s.eventAggregator.AddItem(threadID, unifiedMessageID, "", item.Type, item.Data, item.Metadata)
				msgPushed++
			}
		}
		pushed += msgPushed
	}

	if pushed == 0 {
		return fmt.Errorf("该会话没有可显示的结�")
	}

	s.eventAggregator.FlushNow(threadID, true)
	s.log(fmt.Sprintf("[SHOW-ALL-RESULTS] Pushed %d results for thread %s (unified messageID: %s)", pushed, threadID, unifiedMessageID))
	return nil
}

// SaveMessageAnalysisResults saves analysis results for a specific message
func (s *AnalysisFacadeService) SaveMessageAnalysisResults(threadID, messageID string, results []AnalysisResultItem) error {
	if s.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}
	return s.chatService.SaveAnalysisResults(threadID, messageID, results)
}

// --- Session Recording ---

// SaveSessionRecording saves the current session's analysis recording to a file
func (s *AnalysisFacadeService) SaveSessionRecording(threadID, title, description string) (string, error) {
	if s.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	threads, err := s.chatService.LoadThreads()
	if err != nil {
		return "", fmt.Errorf("failed to get threads: %w", err)
	}

	var thread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			thread = &threads[i]
			break
		}
	}

	if thread == nil {
		return "", fmt.Errorf("thread not found: %s", threadID)
	}

	// Extract data source schema
	var schemas []agent.ReplayTableSchema
	if thread.DataSourceID != "" && s.dataSourceService != nil {
		tables, err := s.dataSourceService.GetDataSourceTables(thread.DataSourceID)
		if err == nil {
			for _, tableName := range tables {
				data, err := s.dataSourceService.GetDataSourceTableData(thread.DataSourceID, tableName, 1)
				if err != nil {
					continue
				}
				var cols []string
				if len(data) > 0 {
					for k := range data[0] {
						cols = append(cols, k)
					}
				}
				schemas = append(schemas, agent.ReplayTableSchema{
					TableName: tableName,
					Columns:   cols,
				})
			}
		}
	}

	// Get data source name
	var sourceName string
	if thread.DataSourceID != "" && s.dataSourceService != nil {
		sources, err := s.dataSourceService.LoadDataSources()
		if err == nil {
			for _, ds := range sources {
				if ds.ID == thread.DataSourceID {
					sourceName = ds.Name
					break
				}
			}
		}
	}

	// Create recorder
	recorder := agent.NewAnalysisRecorder(thread.DataSourceID, sourceName, schemas)
	recorder.SetMetadata(title, description)

	// Parse messages to extract tool calls
	stepID := 0
	for _, msg := range thread.Messages {
		if msg.Role != "assistant" {
			continue
		}

		recorder.RecordConversation("assistant", msg.Content)

		// Try to extract SQL queries from message content
		if strings.Contains(msg.Content, "```sql") {
			startSQL := strings.Index(msg.Content, "```sql")
			endSQL := strings.Index(msg.Content[startSQL+6:], "```")
			if endSQL > 0 {
				sqlQuery := strings.TrimSpace(msg.Content[startSQL+6 : startSQL+6+endSQL])
				stepID++
				recorder.RecordStep("execute_sql", fmt.Sprintf("SQL Query Step %d", stepID), sqlQuery, "", "", "")
			}
		}

		// Try to extract Python code from message content
		if strings.Contains(msg.Content, "```python") {
			startPy := strings.Index(msg.Content, "```python")
			endPy := strings.Index(msg.Content[startPy+9:], "```")
			if endPy > 0 {
				pythonCode := strings.TrimSpace(msg.Content[startPy+9 : startPy+9+endPy])
				stepID++
				recorder.RecordStep("python_executor", fmt.Sprintf("Python Analysis Step %d", stepID), pythonCode, "", "", "")
			}
		}
	}

	// Save recording
	recordingDir := filepath.Join(s.storageDir, "recordings")
	filePath, err := recorder.SaveRecording(recordingDir)
	if err != nil {
		return "", fmt.Errorf("failed to save recording: %w", err)
	}

	s.log(fmt.Sprintf("Session recording saved: %s", filePath))
	return filePath, nil
}

// GetSessionRecordings returns all available session recordings
func (s *AnalysisFacadeService) GetSessionRecordings() ([]agent.AnalysisRecording, error) {
	recordingDir := filepath.Join(s.storageDir, "recordings")

	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recordings directory: %w", err)
	}

	files, err := os.ReadDir(recordingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read recordings directory: %w", err)
	}

	recordings := []agent.AnalysisRecording{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(recordingDir, file.Name())
		recording, err := agent.LoadRecording(filePath)
		if err != nil {
			s.log(fmt.Sprintf("Failed to load recording %s: %v", file.Name(), err))
			continue
		}

		recordings = append(recordings, *recording)
	}

	return recordings, nil
}

// ReplayAnalysisRecording replays a recorded analysis on a target data source
func (s *AnalysisFacadeService) ReplayAnalysisRecording(recordingID, targetSourceID string, autoFixFields bool, maxFieldDiff int) (*agent.ReplayResult, error) {
	if s.einoService == nil {
		return nil, fmt.Errorf("eino service not initialized")
	}
	if s.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	// Load recording
	recordingDir := filepath.Join(s.storageDir, "recordings")
	recordingPath := filepath.Join(recordingDir, fmt.Sprintf("recording_%s.json", recordingID))

	recording, err := agent.LoadRecording(recordingPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load recording: %w", err)
	}

	// Get target data source name
	var targetSourceName string
	sources, err := s.dataSourceService.LoadDataSources()
	if err == nil {
		for _, ds := range sources {
			if ds.ID == targetSourceID {
				targetSourceName = ds.Name
				break
			}
		}
	}

	// Create replay config
	replayConfig := &agent.ReplayConfig{
		RecordingID:      recordingID,
		TargetSourceID:   targetSourceID,
		TargetSourceName: targetSourceName,
		AutoFixFields:    autoFixFields,
		MaxFieldDiff:     maxFieldDiff,
		TableMappings:    []agent.TableMapping{},
	}

	// Create SQL and Python tools
	sqlTool := agent.NewSQLExecutorTool(s.dataSourceService)

	cfg, _ := s.configProvider.GetEffectiveConfig()
	pythonTool := agent.NewPythonExecutorTool(cfg)

	// Create LLM service for intelligent field matching
	llmService := agent.NewLLMService(cfg, s.log)

	// Create replayer
	replayer := agent.NewAnalysisReplayer(
		recording,
		replayConfig,
		s.dataSourceService,
		sqlTool,
		pythonTool,
		llmService,
		s.log,
	)

	// Execute replay
	result, err := replayer.Replay()
	if err != nil {
		return nil, fmt.Errorf("replay failed: %w", err)
	}

	return result, nil
}

// --- Analysis Suggestions (background) ---

// generateAnalysisSuggestions generates analysis suggestions for a data source
func (s *AnalysisFacadeService) generateAnalysisSuggestions(threadID string, analysis *agent.DataSourceAnalysis) {
	if s.chatService == nil {
		return
	}

	// Notify frontend that background task started
	runtime.EventsEmit(s.ctx, "chat-loading", map[string]interface{}{
		"loading":  true,
		"threadId": threadID,
	})
	defer runtime.EventsEmit(s.ctx, "chat-loading", map[string]interface{}{
		"loading":  false,
		"threadId": threadID,
	})

	cfg, _ := s.configProvider.GetEffectiveConfig()
	langPrompt := getLangPrompt(cfg)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Based on the following data source summary and schema, please suggest 3-5 distinct business analysis questions that would provide valuable insights for decision-making. Please answer in %s.\n\nIMPORTANT GUIDELINES:\n- Focus on BUSINESS VALUE and INSIGHTS, not technical implementation\n- Use simple, non-technical language that any business user can understand\n- Frame suggestions as business questions or outcomes (e.g., \"Understand customer purchasing patterns\" instead of \"Run RFM analysis\")\n- DO NOT mention SQL, Python, data processing, or any technical terms\n- Focus on what insights can be discovered, not how to discover them\n\nProvide the suggestions as a clear, structured, numbered list (1., 2., 3...). Each suggestion should include:\n- A clear, business-focused title\n- A one-sentence description of what business insights this would reveal\n\nEnd your response by telling the user (in %s) that they can select one or more analysis questions by replying with the corresponding number(s).", langPrompt, langPrompt))
	sb.WriteString(fmt.Sprintf("Summary: %s\n\n", analysis.Summary))
	sb.WriteString("Schema:\n")
	for _, table := range analysis.Schema {
		sb.WriteString(fmt.Sprintf("- Table: %s, Columns: %s\n", table.TableName, strings.Join(table.Columns, ", ")))
	}

	prompt := sb.String()
	llm := agent.NewLLMService(cfg, s.log)

	resp, err := llm.Chat(context.Background(), prompt)
	if err != nil {
		s.log(fmt.Sprintf("Failed to generate suggestions: %v", err))
		return
	}

	msg := ChatMessage{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Role:      "assistant",
		Content:   resp,
		Timestamp: time.Now().Unix(),
	}

	if err := s.chatService.AddMessage(threadID, msg); err != nil {
		s.log(fmt.Sprintf("Failed to add suggestion message: %v", err))
		return
	}

	insights := s.parseSuggestionsToInsights(resp, "", "")
	if len(insights) > 0 {
		s.log(fmt.Sprintf("Emitting %d suggestions to dashboard insights", len(insights)))
		if s.eventAggregator != nil {
			for _, insight := range insights {
				s.eventAggregator.AddInsight(threadID, msg.ID, "", insight)
			}
			s.eventAggregator.FlushNow(threadID, true)
		}
	}

	runtime.EventsEmit(s.ctx, "thread-updated", threadID)
}

// parseSuggestionsToInsights extracts numbered suggestions from LLM response and converts to Insight objects
func (s *AnalysisFacadeService) parseSuggestionsToInsights(llmResponse, dataSourceID, dataSourceName string) []Insight {
	var insights []Insight
	lines := strings.Split(llmResponse, "\n")

	numberPattern := regexp.MustCompile(`^\s*\*{0,2}(\d+)[.�]\*{0,2}\s*(.+)`)
	listPattern := regexp.MustCompile(`^\s*[-*•]\s+(.+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		var suggestionText string
		if matches := numberPattern.FindStringSubmatch(line); len(matches) > 2 {
			suggestionText = strings.TrimSpace(matches[2])
		} else if matches := listPattern.FindStringSubmatch(line); len(matches) > 1 {
			suggestionText = strings.TrimSpace(matches[1])
		}
		if suggestionText != "" {
			suggestionText = strings.TrimPrefix(suggestionText, "**")
			suggestionText = strings.TrimSuffix(suggestionText, "**")
			suggestionText = strings.TrimSpace(suggestionText)
		}
		if suggestionText != "" {
			insights = append(insights, Insight{
				Text:         suggestionText,
				Icon:         "lightbulb",
				DataSourceID: dataSourceID,
				SourceName:   dataSourceName,
			})
		}
	}

	return insights
}

// --- Helper ---

// log writes a log message using the configured logger
func (s *AnalysisFacadeService) log(msg string) {
	if s.logger != nil {
		s.logger(msg)
	}
}
