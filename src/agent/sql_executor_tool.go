package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type SQLExecutorTool struct {
	dsService         *DataSourceService
	sqlPlanner        *SQLPlanner
	logger            func(string)
	maxRetries        int
	errorKnowledge    *ErrorKnowledge
	sqlCollector      *SQLCollector // Collects successful SQL for training data
	currentQueryLogic string        // Natural language description of current query
	executionRecorder *ExecutionRecorder // Records SQL executions for replay
}

func NewSQLExecutorTool(dsService *DataSourceService) *SQLExecutorTool {
	return &SQLExecutorTool{
		dsService:  dsService,
		maxRetries: 2, // Default: try original + 2 correction attempts
	}
}

// NewSQLExecutorToolWithPlanner creates a tool with self-correction capability
func NewSQLExecutorToolWithPlanner(dsService *DataSourceService, sqlPlanner *SQLPlanner, logger func(string)) *SQLExecutorTool {
	return &SQLExecutorTool{
		dsService:  dsService,
		sqlPlanner: sqlPlanner,
		logger:     logger,
		maxRetries: 2,
	}
}

// SetErrorKnowledge injects the error knowledge system
func (t *SQLExecutorTool) SetErrorKnowledge(ek *ErrorKnowledge) {
	t.errorKnowledge = ek
}

// SetSQLCollector injects the SQL collector for training data collection
func (t *SQLExecutorTool) SetSQLCollector(collector *SQLCollector) {
	t.sqlCollector = collector
}

// SetExecutionRecorder injects the execution recorder
func (t *SQLExecutorTool) SetExecutionRecorder(recorder *ExecutionRecorder) {
	t.executionRecorder = recorder
}

// SetQueryLogic sets the natural language description for the upcoming SQL execution
// This is used to record meaningful intent for training data
func (t *SQLExecutorTool) SetQueryLogic(logic string) {
	t.currentQueryLogic = logic
}

type sqlExecutorInput struct {
	DataSourceID string `json:"data_source_id"`
	Query        string `json:"query"`
}

func (t *SQLExecutorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "execute_sql",
		Desc: "Execute a SQL query against a data source and return results as JSON. Use this to retrieve data for analysis. Results are limited to 1000 rows. Use SELECT statements to query data.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"data_source_id": {
				Type:     schema.String,
				Desc:     "The ID of the data source to query.",
				Required: true,
			},
			"query": {
				Type:     schema.String,
				Desc:     "The SQL query to execute (e.g., 'SELECT * FROM sales WHERE date > 2023-01-01').",
				Required: true,
			},
		}),
	}, nil
}

// convertMySQLToSQLite converts common MySQL syntax to SQLite
func (t *SQLExecutorTool) convertMySQLToSQLite(query string) string {
	// Convert YEAR(column) -> strftime('%Y', column)
	// Use word boundary to avoid matching JULIANDAY, etc.
	yearRegex := regexp.MustCompile(`(?i)\bYEAR\s*\(\s*([^)]+)\s*\)`)
	query = yearRegex.ReplaceAllString(query, "strftime('%Y', $1)")

	// Convert MONTH(column) -> strftime('%m', column)
	monthRegex := regexp.MustCompile(`(?i)\bMONTH\s*\(\s*([^)]+)\s*\)`)
	query = monthRegex.ReplaceAllString(query, "strftime('%m', $1)")

	// Convert DAY(column) -> strftime('%d', column)
	// Use word boundary to avoid matching JULIANDAY
	dayRegex := regexp.MustCompile(`(?i)\bDAY\s*\(\s*([^)]+)\s*\)`)
	query = dayRegex.ReplaceAllString(query, "strftime('%d', $1)")

	// Convert DATE_FORMAT(column, '%Y-%m') -> strftime('%Y-%m', column)
	dateFormatRegex := regexp.MustCompile(`(?i)DATE_FORMAT\s*\(\s*([^,]+)\s*,\s*'([^']+)'\s*\)`)
	query = dateFormatRegex.ReplaceAllString(query, "strftime('$2', $1)")

	// Convert NOW() -> datetime('now')
	nowRegex := regexp.MustCompile(`(?i)NOW\s*\(\s*\)`)
	query = nowRegex.ReplaceAllString(query, "datetime('now')")

	// Convert CURDATE() -> date('now')
	curdateRegex := regexp.MustCompile(`(?i)CURDATE\s*\(\s*\)`)
	query = curdateRegex.ReplaceAllString(query, "date('now')")

	// Convert IFNULL(a, b) -> COALESCE(a, b)
	ifnullRegex := regexp.MustCompile(`(?i)IFNULL\s*\(`)
	query = ifnullRegex.ReplaceAllString(query, "COALESCE(")

	// Convert LOCATE(substr, str) -> INSTR(str, substr) - note the reversed order
	locateRegex := regexp.MustCompile(`(?i)LOCATE\s*\(\s*([^,]+)\s*,\s*([^)]+)\s*\)`)
	query = locateRegex.ReplaceAllString(query, "INSTR($2, $1)")

	// Convert INSTR with 3 parameters (MySQL style with position) to 2 parameters (SQLite)
	// MySQL: INSTR(str, substr, pos)
	// SQLite: INSTR(SUBSTR(str, pos), substr) but this changes the result position
	// For simplicity, if pos is 1, just remove it. Otherwise, warn and use 2-param version
	instr3Regex := regexp.MustCompile(`(?i)INSTR\s*\(\s*([^,]+)\s*,\s*([^,]+)\s*,\s*(\d+)\s*\)`)
	query = instr3Regex.ReplaceAllStringFunc(query, func(match string) string {
		parts := instr3Regex.FindStringSubmatch(match)
		if len(parts) == 4 {
			// If position is 1, just use 2-parameter version
			if parts[3] == "1" {
				return fmt.Sprintf("INSTR(%s, %s)", parts[1], parts[2])
			}
			// For other positions, use SUBSTR + INSTR (approximate)
			// Note: This changes the return value offset
			return fmt.Sprintf("INSTR(SUBSTR(%s, %s), %s)", parts[1], parts[3], parts[2])
		}
		return match
	})

	// Convert SUBSTRING(str, pos, len) -> SUBSTR(str, pos, len)
	substringRegex := regexp.MustCompile(`(?i)SUBSTRING\s*\(`)
	query = substringRegex.ReplaceAllString(query, "SUBSTR(")

	// Convert CONCAT(a, b, c) -> (a || b || c)
	// Simple version for basic concatenation
	// Use word boundary to avoid matching GROUP_CONCAT
	concatRegex := regexp.MustCompile(`(?i)\bCONCAT\s*\(([^)]+)\)`)
	matches := concatRegex.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 {
			args := strings.Split(match[1], ",")
			var trimmedArgs []string
			for _, arg := range args {
				trimmedArgs = append(trimmedArgs, strings.TrimSpace(arg))
			}
			replacement := "(" + strings.Join(trimmedArgs, " || ") + ")"
			query = strings.Replace(query, match[0], replacement, 1)
		}
	}

	return query
}

// fixColumnCaseSensitivity attempts to fix column name case issues
func (t *SQLExecutorTool) fixColumnCaseSensitivity(query string, db *sql.DB, tableName string) string {
	// Get actual column names from the table
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM `%s` LIMIT 0", tableName))
	if err != nil {
		return query
	}
	defer rows.Close()

	actualColumns, err := rows.Columns()
	if err != nil {
		return query
	}

	// Create a case-insensitive map of column names
	columnMap := make(map[string]string)
	for _, col := range actualColumns {
		columnMap[strings.ToLower(col)] = col
	}

	// Replace column references in the query
	// This is a simple approach - look for word boundaries
	for lowerCol, actualCol := range columnMap {
		if lowerCol != actualCol {
			// Use word boundary regex to avoid partial matches
			pattern := fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(lowerCol))
			re := regexp.MustCompile(pattern)
			query = re.ReplaceAllString(query, actualCol)
		}
	}

	return query
}

// getTableColumns retrieves column names for a table
func (t *SQLExecutorTool) getTableColumns(db *sql.DB, dbType string, tableName string) ([]string, error) {
	var rows *sql.Rows
	var err error

	if dbType == "sqlite" {
		rows, err = db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", tableName))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var columns []string
		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}
			columns = append(columns, name)
		}
		return columns, nil
	} else {
		// For MySQL/Doris, use DESCRIBE
		rows, err = db.Query(fmt.Sprintf("DESCRIBE `%s`", tableName))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var columns []string
		for rows.Next() {
			var field, colType string
			var null, key, extra sql.NullString
			var dflt interface{}
			if err := rows.Scan(&field, &colType, &null, &key, &dflt, &extra); err != nil {
				continue
			}
			columns = append(columns, field)
		}
		return columns, nil
	}
}

// executeQueryInternal executes a SQL query and returns results or error details
func (t *SQLExecutorTool) executeQueryInternal(ctx context.Context, dataSourceID, query string) (string, string, error) {
	// Validate query is a safe read-only statement
	cleanQuery := strings.TrimSpace(query)
	cleanQuery = regexp.MustCompile(`--[^\n]*`).ReplaceAllString(cleanQuery, "")
	cleanQuery = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(cleanQuery, "")
	cleanQuery = strings.TrimSpace(cleanQuery)

	upperQuery := strings.ToUpper(cleanQuery)
	if !strings.HasPrefix(upperQuery, "SELECT") && !strings.HasPrefix(upperQuery, "WITH") {
		return "", "", fmt.Errorf("only SELECT queries are allowed for safety")
	}

	// Get Data Source
	sources, err := t.dsService.LoadDataSources()
	if err != nil {
		return "", "", fmt.Errorf("failed to load data sources: %v", err)
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == dataSourceID {
			target = &ds
			break
		}
	}
	if target == nil {
		return "", "", fmt.Errorf("data source not found: %s", dataSourceID)
	}

	// Connect to Database
	var db *sql.DB
	var dbType string
	if target.Config.DBPath != "" {
		dbType = "sqlite"
		dbPath := filepath.Join(t.dsService.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return "", dbType, fmt.Errorf("failed to open database: %v", err)
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		dbType = target.Type
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return "", dbType, fmt.Errorf("failed to connect to database: %v", err)
		}
	} else {
		return "", "", fmt.Errorf("unsupported data source type: %s", target.Type)
	}
	defer db.Close()

	// Apply SQL dialect conversion if needed
	processedQuery := query
	if dbType == "sqlite" {
		processedQuery = t.convertMySQLToSQLite(processedQuery)
	}

	// Extract table name for column case fixing
	fromRegex := regexp.MustCompile(`(?i)FROM\s+` + "`?" + `([a-zA-Z0-9_]+)` + "`?")
	fromMatches := fromRegex.FindStringSubmatch(processedQuery)
	if len(fromMatches) > 1 && dbType == "sqlite" {
		tableName := fromMatches[1]
		processedQuery = t.fixColumnCaseSensitivity(processedQuery, db, tableName)
	}

	// Add LIMIT if not present
	processedQuery = strings.TrimRight(processedQuery, "; \t\n\r")
	queryWithLimit := processedQuery
	upperProcessed := strings.ToUpper(processedQuery)
	if !strings.Contains(upperProcessed, "LIMIT") {
		queryWithLimit = fmt.Sprintf("%s LIMIT 1000", processedQuery)
	}

	// Execute Query
	rows, err := db.Query(queryWithLimit)
	if err != nil {
		return "", dbType, err
	}
	defer rows.Close()

	// Get Column Names
	cols, err := rows.Columns()
	if err != nil {
		return "", dbType, fmt.Errorf("failed to get columns: %v", err)
	}

	// Read Results
	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return "", dbType, fmt.Errorf("failed to scan row: %v", err)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			if val != nil && *val != nil {
				if b, ok := (*val).([]byte); ok {
					rowMap[colName] = string(b)
				} else {
					rowMap[colName] = *val
				}
			} else {
				rowMap[colName] = nil
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return "", dbType, fmt.Errorf("error iterating rows: %v", err)
	}

	// Format Response
	if len(results) == 0 {
		return "✅ Query executed successfully. No rows returned.", dbType, nil
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", dbType, fmt.Errorf("failed to marshal results: %v", err)
	}

	response := fmt.Sprintf("✅ Query executed successfully. Returned %d rows.\n\n%s",
		len(results), string(jsonData))

	maxSize := 50000
	if len(response) > maxSize {
		response = response[:maxSize] + "\n\n[Output truncated - result set too large. Consider using WHERE clause or LIMIT to reduce result size]"
	}

	return response, dbType, nil
}

// buildErrorMessage creates a detailed error message with hints
func (t *SQLExecutorTool) buildErrorMessage(err error, query, dbType string) string {
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("❌ SQL Error: %v\n\n", err))

	errStr := err.Error()
	if strings.Contains(errStr, "no such column") || strings.Contains(errStr, "Unknown column") {
		hasSubquery := strings.Contains(strings.ToUpper(query), "FROM (") || strings.Contains(strings.ToUpper(query), "FROM(")

		if hasSubquery {
			errorMsg.WriteString("⚠️ SUBQUERY COLUMN SCOPE ERROR!\n\n")
			errorMsg.WriteString("Your query has a subquery, but the outer SELECT references a column not in the subquery result.\n\n")
			errorMsg.WriteString("🔧 FIX: Make sure ALL columns used in the outer query are included in the subquery's SELECT.\n\n")
		} else {
			errorMsg.WriteString("💡 FIX: The column name might be misspelled or doesn't exist. Check schema and retry.\n")
		}
		errorMsg.WriteString("🔄 Please rewrite the query with the correct column references and try again.")
	} else if strings.Contains(errStr, "syntax error") {
		errorMsg.WriteString("💡 FIX: Check SQL syntax. If using SQLite, remember:\n")
		errorMsg.WriteString("   - Use strftime('%Y', col) instead of YEAR(col)\n")
		errorMsg.WriteString("   - Use col1 || col2 instead of CONCAT(col1, col2)\n")
		errorMsg.WriteString("   - Use COALESCE instead of IFNULL\n")
	} else {
		errorMsg.WriteString(fmt.Sprintf("Original query:\n%s\n", query))
	}

	return errorMsg.String()
}

func (t *SQLExecutorTool) InvokableRun(ctx context.Context, input string, opts ...tool.Option) (string, error) {
	var in sqlExecutorInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("❌ Error: Invalid input format: %v\n\n💡 Please provide valid JSON with 'data_source_id' and 'query' fields.", err), nil
	}

	currentQuery := in.Query
	var lastError error
	var dbType string
	var executionContext string

	// Try execution with self-correction loop
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		result, detectedDbType, err := t.executeQueryInternal(ctx, in.DataSourceID, currentQuery)
		if detectedDbType != "" {
			dbType = detectedDbType
		}

		if err == nil {
			// Success!
			// Record successful SQL execution for training data
			if t.sqlCollector != nil && currentQuery != "" {
				// Use natural language query logic as intent if available
				intent := t.currentQueryLogic
				if intent == "" {
					// Fallback to SQL if no natural language description
					intent = currentQuery
				}
				t.sqlCollector.SetSQLIntent(intent)
				t.sqlCollector.RecordSuccessfulSQL(currentQuery, 0)
				
				// Clear query logic after recording
				t.currentQueryLogic = ""
				
				if t.logger != nil {
					t.logger(fmt.Sprintf("[SQL-COLLECTOR] Recorded SQL with intent (len=%d)", len(intent)))
				}
			}
			
			// Record execution for replay
			if t.executionRecorder != nil {
				// Extract table names from query
				tables := t.extractTablesFromQuery(currentQuery, in.DataSourceID)
				// Generate step description from query
				stepDescription := t.generateStepDescription(currentQuery)
				t.executionRecorder.RecordSQL(currentQuery, tables, true, "", result, stepDescription)
			}
			
			// If this was a retry (attempt > 0), record the successful solution
			if attempt > 0 && t.errorKnowledge != nil && lastError != nil {
				solution := fmt.Sprintf("Corrected SQL:\n%s", currentQuery)
				t.errorKnowledge.RecordError("sql", lastError.Error(), executionContext, solution, true)
				if t.logger != nil {
					t.logger(fmt.Sprintf("[ERROR-KNOWLEDGE] Recorded successful SQL correction"))
				}
			}

			if attempt > 0 && t.logger != nil {
				t.logger(fmt.Sprintf("[SQL-SELF-CORRECT] Query succeeded after %d correction(s)", attempt))
			}
			return result, nil
		}

		lastError = err
		executionContext = fmt.Sprintf("Executing SQL query (attempt %d/%d): %s", attempt+1, t.maxRetries+1, truncateString(currentQuery, 200))

		// Check error knowledge for hints BEFORE attempting correction
		if t.errorKnowledge != nil {
			hints := t.errorKnowledge.FormatHintsForLLM("sql", err.Error())
			if hints != "" && t.logger != nil {
				t.logger(fmt.Sprintf("[ERROR-KNOWLEDGE] Found similar past errors:%s", hints))
			}
		}

		// Check if we have a planner for self-correction
		if t.sqlPlanner == nil || attempt >= t.maxRetries {
			// No planner or max retries reached
			// Record the failed attempt if this is the last try
			if attempt >= t.maxRetries && t.errorKnowledge != nil {
				t.errorKnowledge.RecordError("sql", err.Error(), executionContext, "Max retries reached, no solution found", false)
			}
			break
		}

		// Attempt self-correction
		if t.logger != nil {
			t.logger(fmt.Sprintf("[SQL-SELF-CORRECT] Attempt %d failed: %v. Trying to correct...", attempt+1, err))
		}

		// Get schema for correction
		schema, schemaErr := t.sqlPlanner.GetEnhancedSchema(ctx, in.DataSourceID)
		if schemaErr != nil {
			if t.logger != nil {
				t.logger(fmt.Sprintf("[SQL-SELF-CORRECT] Failed to get schema for correction: %v", schemaErr))
			}
			break
		}

		// Try to correct the SQL
		correctedSQL, corrErr := t.sqlPlanner.ValidateAndCorrectSQL(ctx, currentQuery, err.Error(), schema)
		if corrErr != nil {
			if t.logger != nil {
				t.logger(fmt.Sprintf("[SQL-SELF-CORRECT] Correction failed: %v", corrErr))
			}
			break
		}

		// Check if the corrected SQL is different
		if strings.TrimSpace(correctedSQL) == strings.TrimSpace(currentQuery) {
			if t.logger != nil {
				t.logger("[SQL-SELF-CORRECT] Corrected SQL is identical to original, stopping retry")
			}
			break
		}

		if t.logger != nil {
			t.logger(fmt.Sprintf("[SQL-SELF-CORRECT] Corrected SQL:\n%s", correctedSQL))
		}

		currentQuery = correctedSQL
	}

	// All retries exhausted, record the final failure
	if t.errorKnowledge != nil && lastError != nil {
		t.errorKnowledge.RecordError("sql", lastError.Error(), executionContext, "Failed after all retry attempts", false)
	}

	// Build error message with hints from error knowledge
	errorMsg := t.buildErrorMessage(lastError, in.Query, dbType)
	if t.errorKnowledge != nil {
		hints := t.errorKnowledge.FormatHintsForLLM("sql", lastError.Error())
		if hints != "" {
			errorMsg += hints
		}
	}

	return errorMsg, nil
}


// extractTablesFromQuery extracts table names and their columns from a SQL query
func (t *SQLExecutorTool) extractTablesFromQuery(query string, dataSourceID string) []TableMetadata {
	tables := []TableMetadata{}
	
	// Simple regex to extract table names from FROM and JOIN clauses
	// This is a basic implementation - could be enhanced with a proper SQL parser
	tableRegex := regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := tableRegex.FindAllStringSubmatch(query, -1)
	
	tableNames := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			tableName := match[1]
			tableNames[tableName] = true
		}
	}
	
	// Get columns for each table
	for tableName := range tableNames {
		columns, err := t.dsService.GetDataSourceTableColumns(dataSourceID, tableName)
		if err != nil {
			if t.logger != nil {
				t.logger(fmt.Sprintf("[EXECUTION-RECORDER] Failed to get columns for table %s: %v", tableName, err))
			}
			columns = []string{} // Empty if failed
		}
		
		tables = append(tables, TableMetadata{
			Name:    tableName,
			Columns: columns,
		})
	}
	
	return tables
}


// generateStepDescription generates a human-readable description of what the SQL does
func (t *SQLExecutorTool) generateStepDescription(query string) string {
	query = strings.ToUpper(strings.TrimSpace(query))
	
	// Simple heuristics to describe the query
	if strings.HasPrefix(query, "SELECT") {
		if strings.Contains(query, "GROUP BY") {
			return "Query and aggregate data"
		} else if strings.Contains(query, "JOIN") {
			return "Query data from multiple tables"
		} else if strings.Contains(query, "WHERE") {
			return "Query filtered data"
		} else {
			return "Query data"
		}
	} else if strings.HasPrefix(query, "INSERT") {
		return "Insert data"
	} else if strings.HasPrefix(query, "UPDATE") {
		return "Update data"
	} else if strings.HasPrefix(query, "DELETE") {
		return "Delete data"
	} else if strings.HasPrefix(query, "CREATE") {
		return "Create table or view"
	}
	
	return "Execute SQL query"
}
