package agent

import (
	"fmt"
	"strings"
	"unicode/utf8"
	"vantagics/i18n"
)

// DefaultMaxExclusionSummaryLength is the default maximum length for exclusion summary
// Validates: Requirements 3.3 (排除列表摘要不超�?00字符)
const DefaultMaxExclusionSummaryLength = 300

// ExclusionManager 排除项管理器
// 管理用户拒绝的意图并生成排除摘要
// 简化自现有�?ExclusionSummarizer，专注于核心功能
// Validates: Requirements 3.2, 3.3
type ExclusionManager struct {
	maxSummaryLength int
}

// NewExclusionManager 创建排除项管理器
// maxSummaryLength: 摘要最大长度，默认300字符
func NewExclusionManager(maxSummaryLength int) *ExclusionManager {
	if maxSummaryLength <= 0 {
		maxSummaryLength = DefaultMaxExclusionSummaryLength
	}
	return &ExclusionManager{
		maxSummaryLength: maxSummaryLength,
	}
}

// CategoryMergeThreshold 分类合并阈�?
// 当排除项超过此数量时，使用分类合并模�?
// Validates: Requirements 3.4
const CategoryMergeThreshold = 10

// GenerateSummary 生成排除项摘�?
// 将排除项列表转换为简洁的摘要文本
// exclusions: 被排除的意图建议列表
// language: 语言设置 ("zh" �?"en")
// Returns: 简洁的摘要文本，不超过 maxSummaryLength 字符
// Validates: Requirements 3.2, 3.3, 3.4
func (m *ExclusionManager) GenerateSummary(exclusions []IntentSuggestion, language string) string {
	if len(exclusions) == 0 {
		return ""
	}

	// 分类排除�?
	categories := m.CategorizeExclusions(exclusions)

	// 根据排除项数量选择摘要模式
	// Validates: Requirements 3.4 (当排除项超过10个时进行分类合并)
	if len(exclusions) > CategoryMergeThreshold {
		return m.buildCategorizedSummary(categories, len(exclusions), language)
	}

	// 根据语言构建摘要
	return m.buildSummary(categories, language)
}

// CategorizeExclusions 分类排除�?
// 将排除项按分析类型分�?
// Returns: map[分析类型][]具体描述
// Validates: Requirements 3.4
func (m *ExclusionManager) CategorizeExclusions(exclusions []IntentSuggestion) map[string][]string {
	categories := make(map[string][]string)

	for _, excl := range exclusions {
		category := m.detectCategory(excl.Title, excl.Description)
		detail := m.extractDetail(excl.Title)

		if _, exists := categories[category]; !exists {
			categories[category] = []string{}
		}

		// 避免重复
		if !m.containsString(categories[category], detail) && detail != "" {
			categories[category] = append(categories[category], detail)
		}
	}

	return categories
}

// GetMaxSummaryLength 获取最大摘要长�?
func (m *ExclusionManager) GetMaxSummaryLength() int {
	return m.maxSummaryLength
}

// detectCategory 检测排除项的分析类�?
func (m *ExclusionManager) detectCategory(title, description string) string {
	combined := strings.ToLower(title + " " + description)

	// 时间趋势分析关键�?
	timeKeywords := []string{"趋势", "时间", "月度", "季度", "年度", "�?, "�?, "变化", "增长", "trend", "time", "monthly", "quarterly", "yearly", "growth", "历史"}

	// 维度分析关键�?
	dimensionKeywords := []string{"分类", "维度", "�?, "分组", "类型", "地区", "产品", "客户", "部门", "category", "dimension", "group", "by", "type", "region", "product"}

	// 统计分析关键�?
	statisticsKeywords := []string{"统计", "汇�?, "总量", "平均", "排名", "最�?, "最�?, "求和", "计数", "statistics", "summary", "total", "average", "ranking", "max", "min", "sum", "count", "top"}

	// 关联分析关键�?
	correlationKeywords := []string{"关联", "相关", "关系", "影响", "因素", "correlation", "relationship", "impact", "factor"}

	// 预测分析关键�?
	predictionKeywords := []string{"预测", "预估", "未来", "forecast", "prediction", "future", "estimate"}

	// 对比分析关键�?
	comparisonKeywords := []string{"比较", "对比", "差异", "compare", "comparison", "difference", "vs"}

	switch {
	case m.containsAnyKeyword(combined, timeKeywords):
		return "时间趋势分析"
	case m.containsAnyKeyword(combined, dimensionKeywords):
		return "分类维度分析"
	case m.containsAnyKeyword(combined, statisticsKeywords):
		return "统计汇�?
	case m.containsAnyKeyword(combined, correlationKeywords):
		return "关联分析"
	case m.containsAnyKeyword(combined, predictionKeywords):
		return "预测分析"
	case m.containsAnyKeyword(combined, comparisonKeywords):
		return "对比分析"
	default:
		return "其他分析"
	}
}

// extractDetail 从标题中提取简短描�?
func (m *ExclusionManager) extractDetail(title string) string {
	if title == "" {
		return ""
	}

	// 限制长度�?5个字�?
	runes := []rune(title)
	if len(runes) > 15 {
		return string(runes[:15])
	}
	return title
}

// containsAnyKeyword 检查文本是否包含任意关键词
func (m *ExclusionManager) containsAnyKeyword(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// containsString 检查切片是否包含字符串
func (m *ExclusionManager) containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// buildSummary 构建摘要文本
func (m *ExclusionManager) buildSummary(categories map[string][]string, language string) string {
	if len(categories) == 0 {
		return ""
	}

	var builder strings.Builder

	// 根据语言选择标题和结�?
	var header, footer string
	if language == "en" {
		header = "Excluded analysis directions:\n"
		footer = "Please understand user intent from other perspectives."
	} else {
		header = "已排除的分析方向：\n"
		footer = "请从其他角度理解用户意图�?
	}

	builder.WriteString(header)

	// 按类别构建摘�?
	categoryCount := 0
	for category, details := range categories {
		line := "- " + category
		if len(details) > 0 {
			// 限制每个类别最多显�?个详�?
			detailsToShow := details
			if len(detailsToShow) > 3 {
				detailsToShow = detailsToShow[:3]
			}
			line += "�? + strings.Join(detailsToShow, "�?) + "�?
		}
		line += "\n"

		// 检查是否会超过最大长�?
		potentialLength := utf8.RuneCountInString(builder.String()) + utf8.RuneCountInString(line) + utf8.RuneCountInString(footer)
		if potentialLength > m.maxSummaryLength {
			// 添加省略提示
			if language == "en" {
				builder.WriteString("- ...(more excluded)\n")
			} else {
				builder.WriteString("- ...（更多已排除）\n")
			}
			break
		}

		builder.WriteString(line)
		categoryCount++
	}

	builder.WriteString(footer)

	result := builder.String()

	// 最终长度检查和截断
	if utf8.RuneCountInString(result) > m.maxSummaryLength {
		result = m.truncateToLength(result, m.maxSummaryLength, footer)
	}

	return result
}

// buildCategorizedSummary 构建分类合并摘要
// 当排除项超过10个时使用此方法，只显示分类和数量，不列出具体项目
// Validates: Requirements 3.4
func (m *ExclusionManager) buildCategorizedSummary(categories map[string][]string, totalCount int, language string) string {
	if len(categories) == 0 {
		return ""
	}

	var builder strings.Builder

	// 根据语言选择标题和结�?
	var header, footer, countFormat string
	header = i18n.T("exclusion.header", totalCount, len(categories))
	footer = i18n.T("exclusion.footer")
	countFormat = i18n.T("exclusion.count_format")

	builder.WriteString(header)

	// 按类别数量排序（从多到少�?
	type categoryInfo struct {
		name  string
		count int
	}
	sortedCategories := make([]categoryInfo, 0, len(categories))
	for name, details := range categories {
		sortedCategories = append(sortedCategories, categoryInfo{name: name, count: len(details)})
	}
	// 简单排序：按数量降�?
	for i := 0; i < len(sortedCategories)-1; i++ {
		for j := i + 1; j < len(sortedCategories); j++ {
			if sortedCategories[j].count > sortedCategories[i].count {
				sortedCategories[i], sortedCategories[j] = sortedCategories[j], sortedCategories[i]
			}
		}
	}

	// 构建分类摘要（只显示类别名和数量�?
	for _, cat := range sortedCategories {
		line := fmt.Sprintf(countFormat, cat.name, cat.count)

		// 检查是否会超过最大长�?
		potentialLength := utf8.RuneCountInString(builder.String()) + utf8.RuneCountInString(line) + utf8.RuneCountInString(footer)
		if potentialLength > m.maxSummaryLength {
			// 添加省略提示
			if language == "en" {
				builder.WriteString("- ...(more categories)\n")
			} else {
				builder.WriteString("- ...（更多类别）\n")
			}
			break
		}

		builder.WriteString(line)
	}

	builder.WriteString(footer)

	result := builder.String()

	// 最终长度检查和截断
	if utf8.RuneCountInString(result) > m.maxSummaryLength {
		result = m.truncateToLength(result, m.maxSummaryLength, footer)
	}

	return result
}

// truncateToLength 截断文本到指定长�?
func (m *ExclusionManager) truncateToLength(text string, maxLength int, footer string) string {
	runes := []rune(text)
	if len(runes) <= maxLength {
		return text
	}

	// 计算可用长度（减去footer长度和换行符�?
	footerLen := utf8.RuneCountInString(footer)
	availableLen := maxLength - footerLen - 1

	if availableLen <= 0 {
		return footer
	}

	// 截断并找到最后一个换行符
	truncated := string(runes[:availableLen])
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > 20 {
		truncated = truncated[:lastNewline]
	}

	return truncated + "\n" + footer
}
