package agent

import "strings"

// IntentSuggestion represents a possible interpretation of user's intent
// This is a local copy to avoid circular imports with the main app package
type ExclusionIntentSuggestion struct {
	ID          string `json:"id"`          // Unique identifier
	Title       string `json:"title"`       // Short title (10 chars max)
	Description string `json:"description"` // Detailed description (30 chars max)
	Icon        string `json:"icon"`        // Icon (emoji or icon name)
	Query       string `json:"query"`       // Actual query/analysis request to execute
}

// DefaultSummarizationThreshold is the default number of exclusions that triggers summarization
const DefaultSummarizationThreshold = 6

// DefaultMaxSummaryLength is the default maximum length for the summary in characters
const DefaultMaxSummaryLength = 500

// ExclusionSummarizer handles summarization of excluded intent suggestions
// When the number of excluded suggestions exceeds a threshold, it generates
// a compressed summary to prevent context overload in LLM prompts.
type ExclusionSummarizer struct {
	threshold int // 触发摘要的阈值，默认 6
	maxLength int // 摘要最大长度，默认 500
}

// NewExclusionSummarizer creates a new ExclusionSummarizer with default settings
// Default threshold: 6 exclusions
// Default max length: 500 characters
func NewExclusionSummarizer() *ExclusionSummarizer {
	return &ExclusionSummarizer{
		threshold: DefaultSummarizationThreshold,
		maxLength: DefaultMaxSummaryLength,
	}
}

// NewExclusionSummarizerWithOptions creates a new ExclusionSummarizer with custom settings
func NewExclusionSummarizerWithOptions(threshold, maxLength int) *ExclusionSummarizer {
	// Ensure threshold is at least 1
	if threshold < 1 {
		threshold = DefaultSummarizationThreshold
	}
	// Ensure maxLength is at least 100
	if maxLength < 100 {
		maxLength = DefaultMaxSummaryLength
	}
	return &ExclusionSummarizer{
		threshold: threshold,
		maxLength: maxLength,
	}
}

// NeedsSummarization checks if the exclusions need to be summarized
// Returns true if the number of exclusions exceeds the threshold
func (s *ExclusionSummarizer) NeedsSummarization(exclusions []ExclusionIntentSuggestion) bool {
	return len(exclusions) > s.threshold
}

// GetThreshold returns the current summarization threshold
func (s *ExclusionSummarizer) GetThreshold() int {
	return s.threshold
}

// GetMaxLength returns the current maximum summary length
func (s *ExclusionSummarizer) GetMaxLength() int {
	return s.maxLength
}

// SummarizeExclusions generates a summary of excluded suggestions
// It extracts analysis types and topics, merges similar exclusions,
// and controls the summary length to be within maxLength characters.
// Returns a formatted summary string suitable for LLM prompts.
//
// Requirements:
// - 6.2: Preserve core semantic features (analysis type, target dimensions, key topics)
// - 6.3: Merge similar exclusions into one summary description
// - 6.4: Control summary length (�?00 characters)
func (s *ExclusionSummarizer) SummarizeExclusions(exclusions []ExclusionIntentSuggestion) string {
	if len(exclusions) == 0 {
		return ""
	}

	// Extract and categorize analysis types from exclusions
	categories := s.categorizeExclusions(exclusions)

	// Build the summary
	return s.buildSummary(categories)
}

// analysisCategory represents a category of analysis with its details
type analysisCategory struct {
	name    string   // Category name (e.g., "时间趋势分析")
	details []string // Specific details within this category
}

// categorizeExclusions extracts and categorizes exclusions by analysis type
func (s *ExclusionSummarizer) categorizeExclusions(exclusions []ExclusionIntentSuggestion) []analysisCategory {
	// Use a map to group similar exclusions
	categoryMap := make(map[string][]string)
	categoryOrder := []string{} // Preserve order of first occurrence

	for _, excl := range exclusions {
		// Extract category and detail from the exclusion
		category, detail := s.extractCategoryAndDetail(excl)

		if _, exists := categoryMap[category]; !exists {
			categoryOrder = append(categoryOrder, category)
			categoryMap[category] = []string{}
		}

		// Add detail if not already present and not empty
		if detail != "" && !s.containsString(categoryMap[category], detail) {
			categoryMap[category] = append(categoryMap[category], detail)
		}
	}

	// Convert map to ordered slice
	result := make([]analysisCategory, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		result = append(result, analysisCategory{
			name:    cat,
			details: categoryMap[cat],
		})
	}

	return result
}

// extractCategoryAndDetail extracts the category and detail from an exclusion
// It analyzes the title and description to determine the analysis type
func (s *ExclusionSummarizer) extractCategoryAndDetail(excl ExclusionIntentSuggestion) (category, detail string) {
	title := excl.Title
	desc := excl.Description

	// Keywords for categorization
	timeKeywords := []string{"趋势", "时间", "月度", "季度", "年度", "�?, "�?, "对比", "变化", "增长", "trend", "time", "monthly", "quarterly", "yearly", "growth"}
	dimensionKeywords := []string{"分类", "维度", "�?, "分组", "类型", "地区", "产品", "客户", "部门", "category", "dimension", "group", "by", "type", "region", "product"}
	statisticsKeywords := []string{"统计", "汇�?, "总量", "平均", "排名", "最�?, "最�?, "求和", "计数", "statistics", "summary", "total", "average", "ranking", "max", "min", "sum", "count"}
	correlationKeywords := []string{"关联", "相关", "关系", "影响", "因素", "correlation", "relationship", "impact", "factor"}
	predictionKeywords := []string{"预测", "预估", "未来", "forecast", "prediction", "future", "estimate"}
	comparisonKeywords := []string{"比较", "对比", "差异", "compare", "comparison", "difference", "vs"}

	combined := title + " " + desc

	// Determine category based on keywords
	switch {
	case s.containsAnyKeyword(combined, timeKeywords):
		category = "时间趋势分析"
		detail = s.extractDetail(title, desc)
	case s.containsAnyKeyword(combined, dimensionKeywords):
		category = "分类维度分析"
		detail = s.extractDetail(title, desc)
	case s.containsAnyKeyword(combined, statisticsKeywords):
		category = "统计汇�?
		detail = s.extractDetail(title, desc)
	case s.containsAnyKeyword(combined, correlationKeywords):
		category = "关联分析"
		detail = s.extractDetail(title, desc)
	case s.containsAnyKeyword(combined, predictionKeywords):
		category = "预测分析"
		detail = s.extractDetail(title, desc)
	case s.containsAnyKeyword(combined, comparisonKeywords):
		category = "对比分析"
		detail = s.extractDetail(title, desc)
	default:
		// Use title as category for uncategorized items
		category = "其他分析"
		detail = title
	}

	return category, detail
}

// extractDetail extracts a meaningful detail from title and description
func (s *ExclusionSummarizer) extractDetail(title, desc string) string {
	// Prefer title as it's more concise
	if len(title) > 0 && len(title) <= 15 {
		return title
	}
	// If title is too long, try to use a shortened version
	if len(title) > 15 {
		// Take first 15 characters
		runes := []rune(title)
		if len(runes) > 15 {
			return string(runes[:15])
		}
		return title
	}
	// Fall back to description if title is empty
	if len(desc) > 0 {
		runes := []rune(desc)
		if len(runes) > 20 {
			return string(runes[:20])
		}
		return desc
	}
	return ""
}

// containsAnyKeyword checks if the text contains any of the keywords
func (s *ExclusionSummarizer) containsAnyKeyword(text string, keywords []string) bool {
	lowerText := strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(lowerText, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// containsString checks if a slice contains a string
func (s *ExclusionSummarizer) containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// buildSummary builds the final summary string from categories
func (s *ExclusionSummarizer) buildSummary(categories []analysisCategory) string {
	if len(categories) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Excluded analysis directions:\n")

	for _, cat := range categories {
		line := "- " + cat.name
		if len(cat.details) > 0 {
			// Merge details, limit to 3 for brevity
			detailsToShow := cat.details
			if len(detailsToShow) > 3 {
				detailsToShow = detailsToShow[:3]
			}
			line += " (" + strings.Join(detailsToShow, ", ") + ")"
		}
		line += "\n"

		// Check if adding this line would exceed maxLength
		if builder.Len()+len(line)+len("Please interpret the user's intent from other angles.") > s.maxLength {
			// Truncate: add ellipsis and break
			builder.WriteString("- ...(more excluded)\n")
			break
		}
		builder.WriteString(line)
	}

	builder.WriteString("Please interpret the user's intent from other angles.")

	result := builder.String()

	// Final length check and truncation if needed
	if len(result) > s.maxLength {
		runes := []rune(result)
		if len(runes) > s.maxLength {
			// Find a good truncation point
			truncated := string(runes[:s.maxLength-20])
			// Find last newline to truncate cleanly
			lastNewline := strings.LastIndex(truncated, "\n")
			if lastNewline > 50 {
				truncated = truncated[:lastNewline]
			}
			result = truncated + "\n请从其他角度理解用户意图�?
		}
	}

	return result
}
