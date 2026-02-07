package agent

import (
	"regexp"
	"strings"
)

// CodeValidator validates generated Python code for safety and correctness
type CodeValidator struct {
	allowedImports    []string
	forbiddenPatterns []string
	maxCodeLength     int
}

// ValidationResult represents the result of code validation
type ValidationResult struct {
	Valid      bool     `json:"valid"`
	Errors     []string `json:"errors"`
	Warnings   []string `json:"warnings"`
	SQLQueries []string `json:"sql_queries"`
	HasChart   bool     `json:"has_chart"`
	HasExport  bool     `json:"has_export"`
}

// NewCodeValidator creates a new code validator with default settings
func NewCodeValidator() *CodeValidator {
	return &CodeValidator{
		allowedImports: []string{
			"sqlite3", "pandas", "numpy", "matplotlib", "seaborn",
			"json", "os", "datetime", "math", "re", "collections",
			"itertools", "functools", "csv", "io", "warnings",
		},
		forbiddenPatterns: []string{
			// System command execution
			`os\.system\s*\(`,
			`subprocess\.`,
			`os\.popen\s*\(`,
			`os\.exec`,
			`commands\.`,
			// File deletion
			`os\.remove\s*\(`,
			`os\.unlink\s*\(`,
			`os\.rmdir\s*\(`,
			`shutil\.rmtree\s*\(`,
			`shutil\.remove\s*\(`,
			`pathlib\.Path.*\.unlink`,
			// Network operations (except allowed)
			`requests\.`,
			`urllib\.request`,
			`urllib\.urlopen`,
			`http\.client`,
			`socket\.`,
			`ftplib\.`,
			`smtplib\.`,
			// Code execution
			`exec\s*\(`,
			`eval\s*\(`,
			`compile\s*\(`,
			`__import__\s*\(`,
			// Dangerous operations
			`pickle\.loads`,
			`marshal\.loads`,
			`yaml\.load\s*\([^,]+\)`, // yaml.load without Loader
		},
		maxCodeLength: 50000, // 50KB max
	}
}

// ValidateCode checks the generated code for safety and correctness
func (v *CodeValidator) ValidateCode(code string) *ValidationResult {
	result := &ValidationResult{
		Valid:      true,
		Errors:     []string{},
		Warnings:   []string{},
		SQLQueries: []string{},
	}

	if code == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "代码为空")
		return result
	}

	// Check code length
	if len(code) > v.maxCodeLength {
		result.Valid = false
		result.Errors = append(result.Errors, "代码长度超过限制")
		return result
	}

	// Check for forbidden patterns
	for _, pattern := range v.forbiddenPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(code) {
			result.Valid = false
			result.Errors = append(result.Errors, "检测到不安全的代码模式: "+pattern)
		}
	}

	// Extract and validate SQL queries
	result.SQLQueries = v.ExtractSQLQueries(code)
	if err := v.ValidateSQLQueries(result.SQLQueries); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Check for chart generation
	result.HasChart = v.hasChartGeneration(code)

	// Check for file export
	result.HasExport = v.hasFileExport(code)

	// Check for basic structure
	if !strings.Contains(code, "import") {
		result.Warnings = append(result.Warnings, "代码缺少import语句")
	}

	// Check for error handling
	if !strings.Contains(code, "try:") || !strings.Contains(code, "except") {
		result.Warnings = append(result.Warnings, "代码缺少错误处理")
	}

	// Check for database connection cleanup
	if strings.Contains(code, "sqlite3.connect") || strings.Contains(code, "conn =") {
		if !strings.Contains(code, "conn.close()") && !strings.Contains(code, "finally:") {
			result.Warnings = append(result.Warnings, "代码可能缺少数据库连接清理")
		}
	}

	return result
}

// ExtractSQLQueries extracts SQL queries from Python code
func (v *CodeValidator) ExtractSQLQueries(code string) []string {
	var queries []string

	// Pattern 1: Triple-quoted strings with SQL keywords
	tripleQuotePattern := regexp.MustCompile(`(?s)"""(.*?)"""|'''(.*?)'''`)
	matches := tripleQuotePattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		content := match[1]
		if content == "" {
			content = match[2]
		}
		if v.looksLikeSQL(content) {
			queries = append(queries, strings.TrimSpace(content))
		}
	}

	// Pattern 2: Single-line strings with SQL keywords
	singleQuotePattern := regexp.MustCompile(`(?:pd\.read_sql_query|pd\.read_sql|cursor\.execute|conn\.execute)\s*\(\s*["']([^"']+)["']`)
	matches = singleQuotePattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 && v.looksLikeSQL(match[1]) {
			queries = append(queries, strings.TrimSpace(match[1]))
		}
	}

	// Pattern 3: Variable assignments with SQL
	sqlVarPattern := regexp.MustCompile(`(?i)(?:sql|query)\s*=\s*["']{1,3}([^"']+)["']{1,3}`)
	matches = sqlVarPattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 && v.looksLikeSQL(match[1]) {
			queries = append(queries, strings.TrimSpace(match[1]))
		}
	}

	return queries
}

// ValidateSQLQueries ensures all SQL queries are read-only
func (v *CodeValidator) ValidateSQLQueries(queries []string) error {
	dangerousKeywords := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER",
		"TRUNCATE", "REPLACE", "MERGE", "GRANT", "REVOKE",
	}

	for _, query := range queries {
		queryUpper := strings.ToUpper(strings.TrimSpace(query))

		// Skip empty queries
		if queryUpper == "" {
			continue
		}

		// Check for dangerous keywords at the start of the query
		for _, keyword := range dangerousKeywords {
			if strings.HasPrefix(queryUpper, keyword) {
				return &SQLValidationError{
					Query:   query,
					Keyword: keyword,
				}
			}
		}

		// Also check for dangerous keywords after common prefixes like WITH
		// WITH ... INSERT/UPDATE/DELETE
		if strings.HasPrefix(queryUpper, "WITH") {
			for _, keyword := range dangerousKeywords {
				if strings.Contains(queryUpper, " "+keyword+" ") {
					return &SQLValidationError{
						Query:   query,
						Keyword: keyword,
					}
				}
			}
		}
	}

	return nil
}

// SQLValidationError represents a SQL validation error
type SQLValidationError struct {
	Query   string
	Keyword string
}

func (e *SQLValidationError) Error() string {
	return "检测到非只读SQL操作: " + e.Keyword + " (只允许SELECT查询)"
}

// looksLikeSQL checks if a string looks like a SQL query
func (v *CodeValidator) looksLikeSQL(s string) bool {
	sUpper := strings.ToUpper(strings.TrimSpace(s))
	sqlKeywords := []string{"SELECT", "FROM", "WHERE", "JOIN", "GROUP BY", "ORDER BY", "LIMIT", "WITH"}
	for _, keyword := range sqlKeywords {
		if strings.Contains(sUpper, keyword) {
			return true
		}
	}
	return false
}

// hasChartGeneration checks if the code generates and SAVES charts
// Only returns true if the code contains actual chart saving code (savefig)
// plt.show() alone is not sufficient as it doesn't produce output files
func (v *CodeValidator) hasChartGeneration(code string) bool {
	// Primary check: code must save the chart to a file
	chartSavePatterns := []string{
		"plt.savefig",
		"fig.savefig",
		"savefig(",
	}
	
	for _, pattern := range chartSavePatterns {
		if strings.Contains(code, pattern) {
			return true
		}
	}
	
	return false
}

// hasChartCode checks if the code contains any chart-related code (for informational purposes)
// This is different from hasChartGeneration which only checks for saved charts
func (v *CodeValidator) hasChartCode(code string) bool {
	chartPatterns := []string{
		"plt.savefig",
		"fig.savefig",
		"plt.show",
		"plt.plot",
		"plt.bar",
		"plt.pie",
		"plt.scatter",
		"plt.hist",
		"sns.barplot",
		"sns.lineplot",
		"sns.scatterplot",
		"sns.heatmap",
	}
	for _, pattern := range chartPatterns {
		if strings.Contains(code, pattern) {
			return true
		}
	}
	return false
}

// hasFileExport checks if the code exports data files (CSV, Excel, JSON)
// Note: savefig is for charts, not data export
func (v *CodeValidator) hasFileExport(code string) bool {
	exportPatterns := []string{
		".to_csv(",
		".to_excel(",
		".to_json(",
		".to_parquet(",
		".to_html(",
	}
	for _, pattern := range exportPatterns {
		if strings.Contains(code, pattern) {
			return true
		}
	}
	return false
}

// ValidateImports checks if all imports are allowed
func (v *CodeValidator) ValidateImports(code string) []string {
	var disallowedImports []string

	// Extract import statements
	importPattern := regexp.MustCompile(`(?m)^(?:import|from)\s+(\w+)`)
	matches := importPattern.FindAllStringSubmatch(code, -1)

	for _, match := range matches {
		if len(match) > 1 {
			moduleName := match[1]
			allowed := false
			for _, allowedModule := range v.allowedImports {
				if moduleName == allowedModule || strings.HasPrefix(moduleName, allowedModule+".") {
					allowed = true
					break
				}
			}
			if !allowed {
				disallowedImports = append(disallowedImports, moduleName)
			}
		}
	}

	return disallowedImports
}

// SetMaxCodeLength sets the maximum allowed code length
func (v *CodeValidator) SetMaxCodeLength(length int) {
	v.maxCodeLength = length
}

// AddAllowedImport adds an import to the allowed list
func (v *CodeValidator) AddAllowedImport(module string) {
	v.allowedImports = append(v.allowedImports, module)
}

// AddForbiddenPattern adds a pattern to the forbidden list
func (v *CodeValidator) AddForbiddenPattern(pattern string) {
	v.forbiddenPatterns = append(v.forbiddenPatterns, pattern)
}
