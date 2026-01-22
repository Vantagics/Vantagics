package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
)

// MemoryItem represents a piece of extractable memory
type MemoryItem struct {
	Content  string
	Tier     MemoryTier // LongTerm, MidTerm, or ShortTerm
	Category string     // "schema", "business_rule", "data_characteristic", "finding"
}

// MemoryTier defines the memory tier for storage
type MemoryTier string

const (
	LongTermTier  MemoryTier = "long_term"  // Persistent facts: data source schemas, business rules
	MidTermTier   MemoryTier = "mid_term"   // Compressed summaries of past conversations
	ShortTermTier MemoryTier = "short_term" // Current conversation context (not persisted)
)

// MemoryExtractor extracts valuable information from analysis results
type MemoryExtractor struct {
	chatModel model.ChatModel
	logger    func(string)
}

// NewMemoryExtractor creates a new memory extractor
func NewMemoryExtractor(chatModel model.ChatModel, logger func(string)) *MemoryExtractor {
	return &MemoryExtractor{
		chatModel: chatModel,
		logger:    logger,
	}
}

// ExtractKeyFindings extracts memorable information from analysis results
func (e *MemoryExtractor) ExtractKeyFindings(
	ctx context.Context,
	userQuery string,
	assistantResponse string,
	sqlQueries []string,
	dataResults []map[string]interface{},
) []MemoryItem {
	
	var memories []MemoryItem
	
	// 1. Extract schema information from SQL queries
	schemaMemories := e.extractSchemaFromSQL(sqlQueries)
	memories = append(memories, schemaMemories...)
	
	// 2. Extract data characteristics from results
	dataMemories := e.extractDataCharacteristics(dataResults, userQuery)
	memories = append(memories, dataMemories...)
	
	// 3. Extract user-declared rules from query
	ruleMemories := e.extractUserRules(userQuery)
	memories = append(memories, ruleMemories...)
	
	// 4. Extract key findings from assistant response (with filtering)
	findingMemories := e.extractFilteredFindings(assistantResponse)
	memories = append(memories, findingMemories...)
	
	// Apply quality filter to all memories
	filtered := []MemoryItem{}
	for _, mem := range memories {
		if e.filterQuality(mem) {
			filtered = append(filtered, mem)
		}
	}
	
	if e.logger != nil && len(filtered) > 0 {
		e.logger(fmt.Sprintf("[MEMORY] Extracted %d memorable items (filtered from %d)", len(filtered), len(memories)))
	}
	
	return filtered
}

// extractSchemaFromSQL extracts table and column information from SQL queries
// Schema information is LONG-TERM memory (persistent, reusable across sessions)
func (e *MemoryExtractor) extractSchemaFromSQL(sqlQueries []string) []MemoryItem {
	var memories []MemoryItem
	seenTables := make(map[string]bool)
	
	for _, query := range sqlQueries {
		upperQuery := strings.ToUpper(query)
		
		// Extract table names from FROM and JOIN clauses
		tables := extractTableNames(query)
		for _, table := range tables {
			if !seenTables[table] {
				seenTables[table] = true
				
				// Extract columns used in this query for this table
				columns := extractColumnsForTable(query, table)
				if len(columns) > 0 {
					columnList := strings.Join(columns, ", ")
					memories = append(memories, MemoryItem{
						Content:  fmt.Sprintf("表 %s 包含字段: %s", table, columnList),
						Tier:     LongTermTier, // Schema is long-term knowledge
						Category: "schema",
					})
				}
			}
		}
		
		// Detect aggregations and calculations
		if strings.Contains(upperQuery, "SUM(") || strings.Contains(upperQuery, "AVG(") || 
		   strings.Contains(upperQuery, "COUNT(") || strings.Contains(upperQuery, "MAX(") || 
		   strings.Contains(upperQuery, "MIN(") {
			// This is a metric calculation, could be a business rule
			// But we'll be conservative and not auto-save unless it's named
		}
	}
	
	return memories
}

// extractDataCharacteristics extracts notable data patterns from results
// Data characteristics are LONG-TERM memory (persistent facts about the data)
func (e *MemoryExtractor) extractDataCharacteristics(results []map[string]interface{}, userQuery string) []MemoryItem {
	var memories []MemoryItem
	
	if len(results) == 0 {
		return memories
	}
	
	// For now, only extract if result is small and meaningful
	// Avoid storing large result sets
	if len(results) <= 10 {
		// Check if this looks like a status/category enumeration
		if len(results) > 0 {
			// Look for fields that might be enumerations
			for key := range results[0] {
				lowerKey := strings.ToLower(key)
				if strings.Contains(lowerKey, "status") || 
				   strings.Contains(lowerKey, "type") || 
				   strings.Contains(lowerKey, "category") ||
				   strings.Contains(lowerKey, "state") {
					// Collect unique values
					values := []string{}
					for _, row := range results {
						if val, ok := row[key]; ok {
							valStr := fmt.Sprintf("%v", val)
							if !stringInSlice(valStr, values) && len(values) < 10 {
								values = append(values, valStr)
							}
						}
					}
					if len(values) > 0 && len(values) <= 5 {
						memories = append(memories, MemoryItem{
							Content:  fmt.Sprintf("字段 %s 的可能值: %s", key, strings.Join(values, ", ")),
							Tier:     LongTermTier, // Data characteristics are long-term
							Category: "data_characteristic",
						})
					}
				}
			}
		}
	}
	
	return memories
}

// extractUserRules extracts explicit rules from user statements
// Business rules are LONG-TERM memory (persistent definitions and conventions)
func (e *MemoryExtractor) extractUserRules(userQuery string) []MemoryItem {
	var memories []MemoryItem
	
	// Look for definition patterns
	definitionPatterns := []string{
		"定义为", "定义成", "是指", "指的是",
		"规则是", "规则为", "约定",
	}
	
	for _, pattern := range definitionPatterns {
		if strings.Contains(userQuery, pattern) {
			// User is defining something, this should be remembered
			memories = append(memories, MemoryItem{
				Content:  userQuery,
				Tier:     LongTermTier, // Business rules are long-term
				Category: "business_rule",
			})
			break // Only add once
		}
	}
	
	return memories
}

// extractFilteredFindings extracts key findings while filtering out suggestions
// Findings are SHORT-TERM memory (current conversation context, not persisted)
func (e *MemoryExtractor) extractFilteredFindings(assistantResponse string) []MemoryItem {
	var memories []MemoryItem
	
	// Split by lines or sentences
	lines := strings.Split(assistantResponse, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check if this line contains a valuable finding
		if e.isValuableFinding(line) && !e.isSuggestion(line) {
			memories = append(memories, MemoryItem{
				Content:  line,
				Tier:     ShortTermTier, // Findings are short-term (current context)
				Category: "finding",
			})
		}
	}
	
	return memories
}

// isValuableFinding checks if a line contains valuable information
func (e *MemoryExtractor) isValuableFinding(line string) bool {
	// Indicators of valuable findings
	valueIndicators := []string{
		"发现", "异常", "趋势", "显示",
		"总计", "平均", "最大", "最小",
		"比例", "占比", "增长", "下降",
	}
	
	for _, indicator := range valueIndicators {
		if strings.Contains(line, indicator) {
			return true
		}
	}
	
	return false
}

// isSuggestion checks if a line is a suggestion (should be filtered)
func (e *MemoryExtractor) isSuggestion(line string) bool {
	// Patterns that indicate suggestions
	suggestionPatterns := []string{
		"可以分析", "可以查看", "可以进一步",
		"建议", "推荐", "不妨", "还可以",
		"或者", "也可以", "尝试", "考虑",
		"如果需要", "想要", "希望",
	}
	
	for _, pattern := range suggestionPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	
	return false
}

// filterQuality applies final quality check to memory items
func (e *MemoryExtractor) filterQuality(item MemoryItem) bool {
	// Minimum length check
	if len(item.Content) < 10 {
		return false
	}
	
	// Maximum length check (avoid storing entire responses)
	if len(item.Content) > 500 {
		return false
	}
	
	// Double-check for suggestion patterns
	if e.isSuggestion(item.Content) {
		return false
	}
	
	// Ensure it has substantive content
	// Reject if it's mostly numbers or symbols
	alphaCount := 0
	for _, r := range item.Content {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
		   (r >= '\u4e00' && r <= '\u9fa5') { // Chinese characters
			alphaCount++
		}
	}
	
	if alphaCount < 5 {
		return false
	}
	
	return true
}

// Helper functions

func extractTableNames(query string) []string {
	var tables []string
	seen := make(map[string]bool)
	
	upperQuery := strings.ToUpper(query)
	
	// Simple pattern matching for FROM and JOIN
	patterns := []string{" FROM ", " JOIN "}
	
	for _, pattern := range patterns {
		idx := 0
		for {
			idx = strings.Index(upperQuery[idx:], pattern)
			if idx == -1 {
				break
			}
			idx += len(pattern)
			
			// Extract table name (until space or comma)
			end := idx
			for end < len(upperQuery) && upperQuery[end] != ' ' && upperQuery[end] != ',' && upperQuery[end] != '\n' {
				end++
			}
			
			if end > idx {
				tableName := query[idx:end]
				tableName = strings.TrimSpace(tableName)
				lowerTable := strings.ToLower(tableName)
				if !seen[lowerTable] && isValidIdentifier(lowerTable) {
					seen[lowerTable] = true
					tables = append(tables, tableName)
				}
			}
			
			idx = end
		}
	}
	
	return tables
}

func extractColumnsForTable(query string, table string) []string {
	var columns []string
	seen := make(map[string]bool)
	
	// Simple heuristic: extract column names from SELECT clause
	upperQuery := strings.ToUpper(query)
	selectIdx := strings.Index(upperQuery, "SELECT")
	fromIdx := strings.Index(upperQuery, "FROM")
	
	if selectIdx != -1 && fromIdx != -1 && selectIdx < fromIdx {
		selectClause := query[selectIdx+6 : fromIdx]
		
		// Split by comma
		parts := strings.Split(selectClause, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			
			// Remove AS aliases
			if asIdx := strings.Index(strings.ToUpper(part), " AS "); asIdx != -1 {
				part = part[:asIdx]
			}
			
			// Extract column name (after last dot if qualified)
			if dotIdx := strings.LastIndex(part, "."); dotIdx != -1 {
				part = part[dotIdx+1:]
			}
			
			// Clean up
			part = strings.TrimSpace(part)
			lowerPart := strings.ToLower(part)
			
			// Skip aggregations and keywords
			if !strings.Contains(strings.ToUpper(part), "(") && 
			   lowerPart != "*" && 
			   isValidIdentifier(lowerPart) &&
			   !seen[lowerPart] {
				seen[lowerPart] = true
				columns = append(columns, part)
			}
		}
	}
	
	return columns
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
		     (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func stringInSlice(item string, slice []string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
