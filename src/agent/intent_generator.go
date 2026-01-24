package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// IntentGenerator 意图生成器
// 负责构建提示词并调用LLM生成意图建议
// 整合 ContextProvider 和 ExclusionManager 的功能
// Validates: Requirements 1.3
type IntentGenerator struct {
	contextProvider *ContextProvider
	exclusionMgr    *ExclusionManager
	logger          func(string)
}

// NewIntentGenerator 创建意图生成器
// Parameters:
//   - contextProvider: 上下文提供器，用于获取数据源上下文
//   - exclusionMgr: 排除项管理器，用于生成排除摘要
//   - logger: 日志函数
//
// Returns: 新的 IntentGenerator 实例
// Validates: Requirements 1.3
func NewIntentGenerator(
	contextProvider *ContextProvider,
	exclusionMgr *ExclusionManager,
	logger func(string),
) *IntentGenerator {
	if logger == nil {
		logger = func(msg string) {
			fmt.Println(msg)
		}
	}

	return &IntentGenerator{
		contextProvider: contextProvider,
		exclusionMgr:    exclusionMgr,
		logger:          logger,
	}
}

// log 记录日志
func (g *IntentGenerator) log(msg string) {
	if g.logger != nil {
		g.logger(msg)
	}
}

// BuildPrompt 构建提示词
// 整合用户消息、数据源上下文、排除项摘要，生成完整的LLM提示词
// Parameters:
//   - userMessage: 用户的原始请求消息
//   - dataSourceContext: 数据源上下文信息（表名、列信息、分析提示等）
//   - exclusionSummary: 排除项摘要（已排除的分析方向）
//   - language: 语言设置 ("zh" 中文, "en" 英文)
//   - maxSuggestions: 最大建议数量
//
// Returns: 完整的LLM提示词
// Validates: Requirements 1.3 (将数据源的列信息和数据特征作为上下文传递给LLM)
func (g *IntentGenerator) BuildPrompt(
	userMessage string,
	dataSourceContext *DataSourceContext,
	exclusionSummary string,
	language string,
	maxSuggestions int,
) string {
	// 确定输出语言指令
	outputLangInstruction := "Respond in English"
	if language == "zh" {
		outputLangInstruction = "用简体中文回复"
	}

	// 构建数据源上下文部分
	contextSection := g.buildDataSourceContextSection(dataSourceContext, language)

	// 构建排除项部分
	exclusionSection := g.buildExclusionSection(exclusionSummary, language)

	// 构建重试指导（如果有排除项）
	retryGuidance := ""
	if exclusionSummary != "" {
		retryGuidance = g.buildRetryGuidance(language)
	}

	// 构建"坚持原始请求"指导
	stickToOriginalGuidance := g.buildStickToOriginalGuidance(language)

	// 构建完整提示词
	prompt := g.buildFullPrompt(
		userMessage,
		contextSection,
		exclusionSection,
		retryGuidance,
		stickToOriginalGuidance,
		outputLangInstruction,
		maxSuggestions,
		language,
	)

	g.log(fmt.Sprintf("[INTENT-GENERATOR] Built prompt for message: %s (language: %s, maxSuggestions: %d)",
		truncateString(userMessage, 50), language, maxSuggestions))

	return prompt
}

// buildDataSourceContextSection 构建数据源上下文部分
// Validates: Requirements 1.3
func (g *IntentGenerator) buildDataSourceContextSection(context *DataSourceContext, language string) string {
	if context == nil {
		return ""
	}

	// 使用 ContextProvider 的 BuildContextSection 方法
	if g.contextProvider != nil {
		return g.contextProvider.BuildContextSection(context, language)
	}

	// 如果没有 ContextProvider，手动构建
	var sb strings.Builder

	if language == "zh" {
		sb.WriteString("## 数据源上下文\n\n")
	} else {
		sb.WriteString("## Data Source Context\n\n")
	}

	// 表名
	if context.TableName != "" {
		if language == "zh" {
			sb.WriteString(fmt.Sprintf("**表名**: %s\n\n", context.TableName))
		} else {
			sb.WriteString(fmt.Sprintf("**Table Name**: %s\n\n", context.TableName))
		}
	}

	// 列信息
	if len(context.Columns) > 0 {
		if language == "zh" {
			sb.WriteString("**列信息**:\n")
		} else {
			sb.WriteString("**Column Information**:\n")
		}

		for _, col := range context.Columns {
			sb.WriteString(fmt.Sprintf("- %s (%s, %s)\n", col.Name, col.Type, col.SemanticType))
		}
		sb.WriteString("\n")
	}

	// 分析提示
	if len(context.AnalysisHints) > 0 {
		if language == "zh" {
			sb.WriteString("**分析建议**:\n")
		} else {
			sb.WriteString("**Analysis Suggestions**:\n")
		}

		for _, hint := range context.AnalysisHints {
			sb.WriteString(fmt.Sprintf("- %s\n", hint))
		}
		sb.WriteString("\n")
	}

	// 最近分析记录
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
			sb.WriteString(fmt.Sprintf("- %s: %s\n", record.AnalysisType, record.KeyFindings))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildExclusionSection 构建排除项部分
func (g *IntentGenerator) buildExclusionSection(exclusionSummary string, language string) string {
	if exclusionSummary == "" {
		return ""
	}

	var sb strings.Builder

	if language == "zh" {
		sb.WriteString("\n\n## 已排除的分析方向\n")
		sb.WriteString("用户已表示以下分析方向不符合其意图：\n\n")
	} else {
		sb.WriteString("\n\n## Previously Rejected Interpretations\n")
		sb.WriteString("The user has indicated that the following interpretations DO NOT match their intent:\n\n")
	}

	sb.WriteString(exclusionSummary)
	sb.WriteString("\n")

	return sb.String()
}

// buildRetryGuidance 构建重试指导
func (g *IntentGenerator) buildRetryGuidance(language string) string {
	if language == "zh" {
		return `

## 重新理解指导
用户拒绝了之前的所有建议。这意味着：
1. 之前的理解偏离了用户意图
2. 需要从完全不同的角度思考
3. 考虑替代的含义、上下文或分析方法
4. 避免与被拒绝建议相似的模式或主题
5. 更具创造性，探索边缘情况或非常规解释`
	}

	return `

## Critical Instruction for Retry
The user rejected ALL previous suggestions. This means:
1. Your previous interpretations were off-target
2. You need to think from COMPLETELY DIFFERENT angles
3. Consider alternative meanings, contexts, or analysis approaches
4. Avoid similar patterns or themes from rejected suggestions
5. Be more creative and explore edge cases or unconventional interpretations`
}

// buildStickToOriginalGuidance 构建"坚持原始请求"指导
func (g *IntentGenerator) buildStickToOriginalGuidance(language string) string {
	if language == "zh" {
		return `

# 关于"坚持我的请求"选项
用户可以选择"坚持我的请求"来直接使用他们的原始输入进行分析。因此：
1. 你的建议应该提供与原始请求不同的分析角度
2. 如果原始请求已经足够具体，你的建议应该探索相关但不同的分析方向
3. 不要简单地重复或轻微改写用户的原始请求
4. 每个建议都应该为用户提供独特的价值`
	}

	return `

# About "Stick to My Request" Option
The user can choose "Stick to My Request" to use their original input directly for analysis. Therefore:
1. Your suggestions should offer different analytical angles from the original request
2. If the original request is already specific, your suggestions should explore related but different analysis directions
3. Do not simply repeat or slightly rephrase the user's original request
4. Each suggestion should provide unique value to the user`
}

// buildFullPrompt 构建完整的提示词
func (g *IntentGenerator) buildFullPrompt(
	userMessage string,
	contextSection string,
	exclusionSection string,
	retryGuidance string,
	stickToOriginalGuidance string,
	outputLangInstruction string,
	maxSuggestions int,
	language string,
) string {
	// 构建列信息字符串（从上下文中提取）
	columnsStr := "No schema information available"
	tableName := "Unknown"

	// 从 contextSection 中提取信息（如果有的话）
	// 这里我们直接使用 contextSection 作为上下文

	// 确定建议数量范围
	minSuggestions := 3
	if maxSuggestions < minSuggestions {
		maxSuggestions = 5
	}

	var prompt strings.Builder

	// 角色定义
	if language == "zh" {
		prompt.WriteString(`# 角色
你是一位专业的数据分析意图解释专家。你的任务是理解用户的模糊请求，并生成多个可能的解释。

`)
	} else {
		prompt.WriteString(`# Role
You are an expert data analysis intent interpreter. Your task is to understand ambiguous user requests and generate multiple plausible interpretations.

`)
	}

	// 用户请求
	if language == "zh" {
		prompt.WriteString(fmt.Sprintf(`# 用户请求
"%s"

`, userMessage))
	} else {
		prompt.WriteString(fmt.Sprintf(`# User's Request
"%s"

`, userMessage))
	}

	// 数据源上下文
	if contextSection != "" {
		prompt.WriteString(contextSection)
	} else {
		// 如果没有上下文，添加基本信息
		if language == "zh" {
			prompt.WriteString(fmt.Sprintf(`# 可用数据上下文
- **表名**: %s
- **列**: %s

`, tableName, columnsStr))
		} else {
			prompt.WriteString(fmt.Sprintf(`# Available Data Context
- **Table**: %s
- **Columns**: %s

`, tableName, columnsStr))
		}
	}

	// 排除项部分
	if exclusionSection != "" {
		prompt.WriteString(exclusionSection)
	}

	// 重试指导
	if retryGuidance != "" {
		prompt.WriteString(retryGuidance)
	}

	// 坚持原始请求指导
	prompt.WriteString(stickToOriginalGuidance)

	// 任务说明
	if language == "zh" {
		prompt.WriteString(fmt.Sprintf(`

# 任务
生成 %d-%d 个不同的用户意图解释。每个解释应该：
1. 代表不同的分析视角或方法
2. 具体且可执行
3. 与可用的数据结构一致
4. 按可能性排序（最可能的排在前面）

# 考虑的解释维度
- **时间分析**: 时间趋势、周期对比、季节性
- **分类分析**: 按类别、地区、产品、客户类型等
- **聚合级别**: 汇总统计、详细分解、排名
- **对比分析**: 同比、环比、基准对比、A/B测试
- **相关性分析**: 变量间关系、因果分析
- **异常检测**: 异常值、异常模式、例外情况
- **预测分析**: 预测、预估、假设分析

# 输出格式
返回一个包含 %d-%d 个解释的 JSON 数组。每个对象必须包含：

[
  {
    "title": "简短描述性标题（最多10个字）",
    "description": "清晰解释这个解释的含义（最多30个字）",
    "icon": "相关的表情符号（📊, 📈, 📉, 🔍, 💡, 📅, 🎯 等）",
    "query": "具体、详细的分析请求，可以直接执行（明确指标、维度和筛选条件）"
  }
]

# 质量要求
- **具体性**: 每个 query 应该足够详细，可以无歧义地执行
- **多样性**: 解释应该覆盖不同的分析角度
- **可行性**: 只建议可以用可用列执行的分析
- **清晰性**: 描述应该清晰，避免专业术语
- **语言**: %s

# 输出规则
- 只返回 JSON 数组
- 不要使用 markdown 代码块，不要解释，不要额外文本
- 确保 JSON 语法正确
- 以 [ 开始，以 ] 结束

现在生成解释：`, minSuggestions, maxSuggestions, minSuggestions, maxSuggestions, outputLangInstruction))
	} else {
		prompt.WriteString(fmt.Sprintf(`

# Task
Generate %d-%d distinct interpretations of the user's intent. Each interpretation should:
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
Return a JSON array with %d-%d interpretations. Each object must include:

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

Generate the interpretations now:`, minSuggestions, maxSuggestions, minSuggestions, maxSuggestions, outputLangInstruction))
	}

	return prompt.String()
}

// Note: truncateString is defined in utils.go

// LLMCallFunc 定义LLM调用函数类型
// 用于依赖注入，便于测试和灵活配置
// Parameters:
//   - ctx: 上下文，用于取消操作
//   - prompt: 发送给LLM的提示词
//
// Returns:
//   - string: LLM的响应文本
//   - error: 调用失败时的错误
type LLMCallFunc func(ctx context.Context, prompt string) (string, error)

// Generate 生成意图建议
// 构建提示词，调用LLM，解析响应
// Parameters:
//   - ctx: 上下文，用于取消操作
//   - userMessage: 用户的原始请求消息
//   - dataSourceContext: 数据源上下文信息（表名、列信息、分析提示等）
//   - exclusionSummary: 排除项摘要（已排除的分析方向）
//   - language: 语言设置 ("zh" 中文, "en" 英文)
//   - maxSuggestions: 最大建议数量
//   - llmCall: LLM调用函数，用于实际调用LLM服务
//
// Returns:
//   - []IntentSuggestion: 生成的意图建议列表
//   - error: 生成失败时的错误
//
// Validates: Requirements 1.1 (调用LLM生成3-5个意图建议), 1.2 (每个意图建议包含完整字段)
func (g *IntentGenerator) Generate(
	ctx context.Context,
	userMessage string,
	dataSourceContext *DataSourceContext,
	exclusionSummary string,
	language string,
	maxSuggestions int,
	llmCall LLMCallFunc,
) ([]IntentSuggestion, error) {
	// 验证LLM调用函数
	if llmCall == nil {
		return nil, fmt.Errorf("LLM call function is required")
	}

	// 构建提示词
	prompt := g.BuildPrompt(userMessage, dataSourceContext, exclusionSummary, language, maxSuggestions)
	g.log(fmt.Sprintf("[INTENT-GENERATOR] Built prompt, length: %d characters", len(prompt)))

	// 调用LLM
	g.log("[INTENT-GENERATOR] Calling LLM to generate intent suggestions...")
	response, err := llmCall(ctx, prompt)
	if err != nil {
		g.log(fmt.Sprintf("[INTENT-GENERATOR] LLM call failed: %v", err))
		return nil, fmt.Errorf("意图生成失败: %w", err)
	}

	g.log(fmt.Sprintf("[INTENT-GENERATOR] Received LLM response, length: %d characters", len(response)))

	// 解析响应
	suggestions, err := g.ParseResponse(response)
	if err != nil {
		g.log(fmt.Sprintf("[INTENT-GENERATOR] Response parse failed: %v", err))
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	// 验证建议数量
	if len(suggestions) == 0 {
		g.log("[INTENT-GENERATOR] No suggestions generated")
		return nil, fmt.Errorf("未能生成意图建议")
	}

	g.log(fmt.Sprintf("[INTENT-GENERATOR] Successfully generated %d intent suggestions", len(suggestions)))

	return suggestions, nil
}

// ParseResponse 解析LLM响应为IntentSuggestion列表
// 从LLM响应中提取JSON数组并解析为意图建议
// Parameters:
//   - response: LLM的原始响应文本
//
// Returns:
//   - []IntentSuggestion: 解析后的意图建议列表
//   - error: 解析失败时的错误
//
// Validates: Requirements 1.2 (每个意图建议包含完整的title、description、icon和query字段)
func (g *IntentGenerator) ParseResponse(response string) ([]IntentSuggestion, error) {
	// 提取JSON数组
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")

	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no valid JSON array found in response")
	}

	jsonStr := response[start : end+1]

	// 解析JSON
	var rawSuggestions []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawSuggestions); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// 转换为IntentSuggestion
	suggestions := make([]IntentSuggestion, 0, len(rawSuggestions))
	timestamp := time.Now().Unix()

	for i, raw := range rawSuggestions {
		// 生成唯一ID
		id := fmt.Sprintf("intent_%d_%d", timestamp, i)

		// 提取字段
		title := g.getStringField(raw, "title")
		description := g.getStringField(raw, "description")
		icon := g.getStringField(raw, "icon")
		query := g.getStringField(raw, "query")

		// 创建建议
		suggestion := IntentSuggestion{
			ID:          id,
			Title:       title,
			Description: description,
			Icon:        icon,
			Query:       query,
		}

		// 验证必需字段
		// Validates: Requirements 1.2 (每个意图建议包含完整字段)
		if suggestion.Title != "" && suggestion.Query != "" {
			// 如果缺少icon，使用默认值
			if suggestion.Icon == "" {
				suggestion.Icon = "📊"
			}
			// 如果缺少description，使用title作为默认值
			if suggestion.Description == "" {
				suggestion.Description = suggestion.Title
			}
			suggestions = append(suggestions, suggestion)
		} else {
			g.log(fmt.Sprintf("[INTENT-GENERATOR] Skipping invalid suggestion at index %d: missing title or query", i))
		}
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no valid suggestions found in response")
	}

	return suggestions, nil
}

// getStringField 从map中获取字符串字段
// Parameters:
//   - m: 包含字段的map
//   - key: 字段名
//
// Returns: 字段值，如果不存在或不是字符串则返回空字符串
func (g *IntentGenerator) getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ValidateSuggestions 验证意图建议列表
// 检查每个建议是否包含所有必需字段
// Parameters:
//   - suggestions: 要验证的意图建议列表
//
// Returns:
//   - []IntentSuggestion: 有效的意图建议列表
//   - []string: 验证错误信息列表
//
// Validates: Requirements 1.2 (每个意图建议包含完整字段)
func (g *IntentGenerator) ValidateSuggestions(suggestions []IntentSuggestion) ([]IntentSuggestion, []string) {
	valid := make([]IntentSuggestion, 0, len(suggestions))
	errors := make([]string, 0)

	for i, s := range suggestions {
		if !s.IsValid() {
			var missing []string
			if s.ID == "" {
				missing = append(missing, "id")
			}
			if s.Title == "" {
				missing = append(missing, "title")
			}
			if s.Description == "" {
				missing = append(missing, "description")
			}
			if s.Icon == "" {
				missing = append(missing, "icon")
			}
			if s.Query == "" {
				missing = append(missing, "query")
			}
			errors = append(errors, fmt.Sprintf("suggestion %d missing fields: %s", i, strings.Join(missing, ", ")))
		} else {
			valid = append(valid, s)
		}
	}

	return valid, errors
}
