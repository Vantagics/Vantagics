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
	dsService *DataSourceService
}

func NewSQLExecutorTool(dsService *DataSourceService) *SQLExecutorTool {
	return &SQLExecutorTool{
		dsService: dsService,
	}
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

func (t *SQLExecutorTool) InvokableRun(ctx context.Context, input string, opts ...tool.Option) (string, error) {
	var in sqlExecutorInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}

	// Validate query is a safe read-only statement
	// Remove comments and extra whitespace for validation
	cleanQuery := strings.TrimSpace(in.Query)
	// Remove SQL comments (-- and /* */)
	cleanQuery = regexp.MustCompile(`--[^\n]*`).ReplaceAllString(cleanQuery, "")
	cleanQuery = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(cleanQuery, "")
	cleanQuery = strings.TrimSpace(cleanQuery)

	upperQuery := strings.ToUpper(cleanQuery)

	// Allow SELECT and WITH (for CTEs)
	if !strings.HasPrefix(upperQuery, "SELECT") && !strings.HasPrefix(upperQuery, "WITH") {
		return "", fmt.Errorf("only SELECT queries are allowed for safety. Use SELECT to retrieve data.\nReceived query: %s", in.Query)
	}

	// 1. Get Data Source
	sources, err := t.dsService.LoadDataSources()
	if err != nil {
		return "", fmt.Errorf("failed to load data sources: %v", err)
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == in.DataSourceID {
			target = &ds
			break
		}
	}

	if target == nil {
		return "", fmt.Errorf("data source not found: %s", in.DataSourceID)
	}

	// 2. Connect to Database
	var db *sql.DB
	var dbType string
	if target.Config.DBPath != "" {
		// Local SQLite
		dbType = "sqlite"
		dbPath := filepath.Join(t.dsService.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return "", fmt.Errorf("failed to open database: %v", err)
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		// Remote MySQL/Doris
		dbType = target.Type
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return "", fmt.Errorf("failed to connect to database: %v", err)
		}
	} else {
		return "", fmt.Errorf("unsupported data source type: %s", target.Type)
	}
	defer db.Close()

	// 3. Apply SQL dialect conversion if needed
	processedQuery := in.Query
	if dbType == "sqlite" {
		processedQuery = t.convertMySQLToSQLite(processedQuery)
	}

	// 4. Extract table name from query for column case fixing (simple extraction)
	// This handles: SELECT ... FROM tablename ...
	fromRegex := regexp.MustCompile(`(?i)FROM\s+` + "`?" + `([a-zA-Z0-9_]+)` + "`?")
	fromMatches := fromRegex.FindStringSubmatch(processedQuery)
	if len(fromMatches) > 1 && dbType == "sqlite" {
		tableName := fromMatches[1]
		processedQuery = t.fixColumnCaseSensitivity(processedQuery, db, tableName)
	}

	// 5. Add LIMIT to prevent huge result sets if not already present
	// First, strip trailing semicolons to avoid "ORDER BY x; LIMIT 1000" syntax error
	processedQuery = strings.TrimRight(processedQuery, "; \t\n\r")

	queryWithLimit := processedQuery
	upperProcessed := strings.ToUpper(processedQuery)
	if !strings.Contains(upperProcessed, "LIMIT") {
		queryWithLimit = fmt.Sprintf("%s LIMIT 1000", processedQuery)
	}

	// Execute Query
	rows, err := db.Query(queryWithLimit)
	if err != nil {
		// Enhanced error message with the actual query that was executed
		errorMsg := fmt.Sprintf("query execution failed: %v\nOriginal query: %s\nProcessed query: %s", err, in.Query, queryWithLimit)

		// If it's a "no such column" error, try to provide helpful info about available columns
		if strings.Contains(err.Error(), "no such column") {
			if len(fromMatches) > 1 {
				tableName := fromMatches[1]
				if columns, colErr := t.getTableColumns(db, dbType, tableName); colErr == nil && len(columns) > 0 {
					errorMsg += fmt.Sprintf("\n\n❌ Column not found in table '%s'.\n✅ Available columns: %s", tableName, strings.Join(columns, ", "))
					errorMsg += "\n\n💡 Please rewrite your query using only the available columns listed above."
				}
			} else {
				errorMsg += "\n\n💡 The column name in your query doesn't exist. Please check the table schema using get_data_source_context tool first."
			}
		}

		return "", fmt.Errorf("%s", errorMsg)
	}
	defer rows.Close()

	// 7. Get Column Names
	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %v", err)
	}

	// 8. Read Results
	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return "", fmt.Errorf("failed to scan row: %v", err)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			if val != nil && *val != nil {
				// Handle []byte for text columns
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
		return "", fmt.Errorf("error iterating rows: %v", err)
	}

	// 9. Format Response
	if len(results) == 0 {
		return "Query executed successfully. No rows returned.", nil
	}

	// Return as JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %v", err)
	}

	response := fmt.Sprintf("Query executed successfully. Returned %d rows.\n\n%s",
		len(results), string(jsonData))

	// Truncate if too large (keep first 50KB)
	maxSize := 50000
	if len(response) > maxSize {
		response = response[:maxSize] + "\n\n[Output truncated - result set too large. Consider using WHERE clause or LIMIT to reduce result size]"
	}

	return response, nil
}
