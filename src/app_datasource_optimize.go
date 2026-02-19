package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"vantagedata/agent"
	"vantagedata/i18n"
)

// OptimizeSuggestion represents a DuckDB-specific optimization suggestion
type OptimizeSuggestion struct {
	TableName   string   `json:"table_name"`
	Type        string   `json:"type"` // "column_type", "enum_conversion", "sort_order", "compression"
	Description string   `json:"description"`
	Columns     []string `json:"columns"`
	Reason      string   `json:"reason"`
	SQLCommands []string `json:"sql_commands"`
	Applied     bool     `json:"applied"`
	Error       string   `json:"error,omitempty"`
}

// Keep IndexSuggestion for backward compatibility with existing callers
type IndexSuggestion = OptimizeSuggestion

// OptimizeSuggestionsResult represents the suggestions result
type OptimizeSuggestionsResult struct {
	DataSourceID   string               `json:"data_source_id"`
	DataSourceName string               `json:"data_source_name"`
	Suggestions    []OptimizeSuggestion `json:"suggestions"`
	Success        bool                 `json:"success"`
	Error          string               `json:"error,omitempty"`
}

// OptimizeDataSourceResult represents the optimization result
type OptimizeDataSourceResult struct {
	DataSourceID   string               `json:"data_source_id"`
	DataSourceName string               `json:"data_source_name"`
	Suggestions    []OptimizeSuggestion `json:"suggestions"`
	Summary        string               `json:"summary"`
	Success        bool                 `json:"success"`
	Error          string               `json:"error,omitempty"`
}

// GetOptimizeSuggestions generates DuckDB-specific optimization suggestions
func (a *App) GetOptimizeSuggestions(dataSourceID string) (*OptimizeSuggestionsResult, error) {
	a.Log(fmt.Sprintf("[OPTIMIZE] Generating suggestions for data source: %s", dataSourceID))

	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	sources, err := a.dataSourceService.LoadDataSources()
	if err != nil {
		return nil, fmt.Errorf("failed to load data sources: %w", err)
	}

	var dataSource *agent.DataSource
	for _, ds := range sources {
		if ds.ID == dataSourceID {
			dataSource = &ds
			break
		}
	}

	if dataSource == nil {
		return nil, fmt.Errorf("data source not found: %s", dataSourceID)
	}

	// Remote databases cannot be optimized
	if dataSource.Type == "mysql" || dataSource.Type == "doris" || dataSource.Type == "postgresql" {
		if !dataSource.Config.StoreLocally {
			return &OptimizeSuggestionsResult{
				DataSourceID:   dataSourceID,
				DataSourceName: dataSource.Name,
				Success:        false,
				Error:          i18n.T("optimize.remote_not_allowed", dataSource.Type),
			}, nil
		}
	}

	if dataSource.Config.DBPath == "" {
		return &OptimizeSuggestionsResult{
			DataSourceID:   dataSourceID,
			DataSourceName: dataSource.Name,
			Success:        false,
			Error:          i18n.T("optimize.no_local_storage"),
		}, nil
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] Data source: %s, DBPath: %s", dataSource.Name, dataSource.Config.DBPath))

	cfg, err := a.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	dbPath := filepath.Join(cfg.DataCacheDir, dataSource.Config.DBPath)

	db, err := a.dataSourceService.DB.OpenReadOnly(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	suggestions, err := a.generateOptimizeSuggestions(db, dataSourceID)
	if err != nil {
		return nil, err
	}

	result := &OptimizeSuggestionsResult{
		DataSourceID:   dataSourceID,
		DataSourceName: dataSource.Name,
		Suggestions:    suggestions,
		Success:        len(suggestions) > 0,
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] Generated %d suggestions", len(suggestions)))
	return result, nil
}

// ApplyOptimizeSuggestions applies the given DuckDB optimization suggestions
func (a *App) ApplyOptimizeSuggestions(dataSourceID string, suggestions []OptimizeSuggestion) (*OptimizeDataSourceResult, error) {
	a.Log(fmt.Sprintf("[OPTIMIZE] Applying %d suggestions for data source: %s", len(suggestions), dataSourceID))

	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	sources, err := a.dataSourceService.LoadDataSources()
	if err != nil {
		return nil, fmt.Errorf("failed to load data sources: %w", err)
	}

	var dataSource *agent.DataSource
	for _, ds := range sources {
		if ds.ID == dataSourceID {
			dataSource = &ds
			break
		}
	}

	if dataSource == nil {
		return nil, fmt.Errorf("data source not found: %s", dataSourceID)
	}

	if dataSource.Type == "mysql" || dataSource.Type == "doris" || dataSource.Type == "postgresql" {
		if !dataSource.Config.StoreLocally {
			return &OptimizeDataSourceResult{
				DataSourceID:   dataSourceID,
				DataSourceName: dataSource.Name,
				Success:        false,
				Error:          i18n.T("optimize.remote_not_allowed", dataSource.Type),
			}, nil
		}
	}

	if dataSource.Config.DBPath == "" {
		return &OptimizeDataSourceResult{
			DataSourceID:   dataSourceID,
			DataSourceName: dataSource.Name,
			Success:        false,
			Error:          i18n.T("optimize.no_local_storage"),
		}, nil
	}

	cfg, err := a.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	dbPath := filepath.Join(cfg.DataCacheDir, dataSource.Config.DBPath)

	db, err := a.dataSourceService.DB.OpenWritable(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	appliedCount := 0
	for i := range suggestions {
		// Validate SQL commands are safe (only ALTER TABLE, CREATE INDEX, COPY allowed)
		allSafe := true
		for _, sqlCmd := range suggestions[i].SQLCommands {
			if !isAllowedOptimizeSQL(sqlCmd) {
				suggestions[i].Applied = false
				suggestions[i].Error = "rejected: SQL command not in allowed list"
				a.Log(fmt.Sprintf("[OPTIMIZE] Rejected SQL: %s", sqlCmd[:min(len(sqlCmd), 80)]))
				allSafe = false
				break
			}
		}
		if !allSafe {
			continue
		}

		// Execute all SQL commands for this suggestion
		allSucceeded := true
		for _, sqlCmd := range suggestions[i].SQLCommands {
			_, err := db.Exec(sqlCmd)
			if err != nil {
				suggestions[i].Applied = false
				suggestions[i].Error = err.Error()
				a.Log(fmt.Sprintf("[OPTIMIZE] Failed to execute: %s, error: %v", sqlCmd[:min(len(sqlCmd), 80)], err))
				allSucceeded = false
				break
			}
		}

		if allSucceeded {
			suggestions[i].Applied = true
			appliedCount++
			a.Log(fmt.Sprintf("[OPTIMIZE] Applied optimization: %s on %s", suggestions[i].Type, suggestions[i].TableName))
		}
	}

	// Mark data source as optimized
	if appliedCount > 0 {
		dataSource.Config.Optimized = true
		sources, err := a.dataSourceService.LoadDataSources()
		if err != nil {
			a.Log(fmt.Sprintf("[OPTIMIZE] Failed to load data sources: %v", err))
		} else {
			found := false
			for j := range sources {
				if sources[j].ID == dataSource.ID {
					sources[j].Config.Optimized = true
					found = true
					break
				}
			}
			if found {
				if err := a.dataSourceService.SaveDataSources(sources); err != nil {
					a.Log(fmt.Sprintf("[OPTIMIZE] Failed to save optimized status: %v", err))
				}
			}
		}
	}

	summary := i18n.T("optimize.summary", appliedCount, len(suggestions))

	result := &OptimizeDataSourceResult{
		DataSourceID:   dataSourceID,
		DataSourceName: dataSource.Name,
		Suggestions:    suggestions,
		Summary:        summary,
		Success:        appliedCount > 0,
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] Optimization complete: %s", summary))
	return result, nil
}

// isAllowedOptimizeSQL checks if a SQL command is safe for optimization
func isAllowedOptimizeSQL(sqlCmd string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sqlCmd))
	allowedPrefixes := []string{
		"ALTER TABLE",
		"CREATE INDEX",
		"CREATE UNIQUE INDEX",
		"COPY",
		"CREATE TABLE",
		"INSERT INTO",
		"DROP TABLE",
	}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

// generateOptimizeSuggestions generates DuckDB-specific optimization suggestions using LLM
func (a *App) generateOptimizeSuggestions(db *sql.DB, dataSourceID string) ([]OptimizeSuggestion, error) {
	tables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Build schema description with column stats for LLM
	var schemaDesc strings.Builder
	schemaDesc.WriteString("DuckDB Database Schema and Statistics:\n\n")

	for _, tableName := range tables {
		columnTypes, err := a.getColumnTypes(db, tableName)
		if err != nil {
			a.Log(fmt.Sprintf("[OPTIMIZE] Failed to get column types for table %s: %v", tableName, err))
			continue
		}

		// Get row count
		rowCount := a.getTableRowCount(db, tableName)

		schemaDesc.WriteString(fmt.Sprintf("Table: %s (rows: %d)\n", tableName, rowCount))
		schemaDesc.WriteString("Columns:\n")

		for colName, colType := range columnTypes {
			// Get cardinality for string/categorical columns
			card := a.getColumnCardinality(db, tableName, colName)
			if card > 0 {
				schemaDesc.WriteString(fmt.Sprintf("  - %s (%s) [distinct values: %d]\n", colName, colType, card))
			} else {
				schemaDesc.WriteString(fmt.Sprintf("  - %s (%s)\n", colName, colType))
			}
		}
		schemaDesc.WriteString("\n")
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] Schema description:\n%s", schemaDesc.String()))

	cfg, err := a.GetEffectiveConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	prompt := fmt.Sprintf(`You are a DuckDB optimization expert. Analyze the following DuckDB database schema and statistics, then suggest optimizations.

%s

DuckDB is a columnar analytical database. Traditional B-tree indexes are NOT useful for analytical workloads. Instead, focus on these DuckDB-specific optimizations:

1. **Column Type Optimization**: Suggest more efficient types. For example:
   - VARCHAR columns with low cardinality (few distinct values relative to row count) should be converted to ENUM type
   - Overly wide numeric types (e.g., DOUBLE for integer data, BIGINT for small numbers) can be narrowed
   - Date/time stored as VARCHAR should be converted to DATE/TIMESTAMP

2. **Sort Order Optimization**: Suggest sorting tables by frequently filtered columns (dates, categories) to improve zone map effectiveness. This requires recreating the table with sorted data.

3. **Compression Hints**: For very large tables, suggest explicit compression if beneficial.

IMPORTANT RULES:
- Only suggest changes that will genuinely improve performance or reduce storage
- For ENUM conversion, only suggest when distinct values < 1000 AND distinct values < 10%% of total rows
- For sort order, only suggest for tables with > 10000 rows
- Each suggestion must include executable DuckDB SQL commands
- For column type changes, use ALTER TABLE ... ALTER COLUMN ... SET DATA TYPE ...
- For ENUM conversions, first CREATE TYPE, then ALTER TABLE
- For sort order, use: CREATE TABLE new AS SELECT * FROM old ORDER BY col; DROP TABLE old; ALTER TABLE new RENAME TO old;

Return ONLY a JSON array (no other text). Each element:
{
  "table_name": "orders",
  "type": "enum_conversion",
  "description": "Convert status column to ENUM type",
  "columns": ["status"],
  "reason": "Column has only 5 distinct values in 100000 rows, ENUM saves storage and speeds up filtering",
  "sql_commands": ["CREATE TYPE status_enum AS ENUM ('active','inactive','pending','completed','cancelled')", "ALTER TABLE orders ALTER COLUMN status SET DATA TYPE status_enum"]
}

If no optimizations are needed, return an empty array: []`, schemaDesc.String())

	llm := agent.NewLLMService(cfg, a.Log)
	response, err := llm.Chat(context.Background(), prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM suggestions: %w", err)
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] LLM response: %s", response))

	suggestions, err := a.parseOptimizeSuggestions(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM suggestions: %w", err)
	}

	// Validate suggestions against actual schema
	tableColumns := make(map[string]map[string]bool)
	for _, tableName := range tables {
		columnTypes, err := a.getColumnTypes(db, tableName)
		if err != nil {
			continue
		}
		tableColumns[tableName] = make(map[string]bool)
		for col := range columnTypes {
			tableColumns[tableName][col] = true
		}
	}

	validSuggestions := []OptimizeSuggestion{}
	for _, sug := range suggestions {
		colMap, tableExists := tableColumns[sug.TableName]
		if !tableExists {
			a.Log(fmt.Sprintf("[OPTIMIZE] Skipping: table %s does not exist", sug.TableName))
			continue
		}

		allColumnsExist := true
		for _, col := range sug.Columns {
			if !colMap[col] {
				a.Log(fmt.Sprintf("[OPTIMIZE] Skipping: column '%s' does not exist in table '%s'", col, sug.TableName))
				allColumnsExist = false
				break
			}
		}
		if !allColumnsExist {
			continue
		}

		if len(sug.SQLCommands) == 0 {
			a.Log(fmt.Sprintf("[OPTIMIZE] Skipping: no SQL commands for %s", sug.Description))
			continue
		}

		validSuggestions = append(validSuggestions, sug)
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] Filtered from %d to %d valid suggestions", len(suggestions), len(validSuggestions)))
	return validSuggestions, nil
}

// getTableRowCount returns the approximate row count for a table
func (a *App) getTableRowCount(db *sql.DB, tableName string) int64 {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, strings.ReplaceAll(tableName, `"`, `""`))
	var count int64
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		a.Log(fmt.Sprintf("[OPTIMIZE] Failed to get row count for %s: %v", tableName, err))
		return 0
	}
	return count
}

// getColumnCardinality returns the number of distinct values in a column
func (a *App) getColumnCardinality(db *sql.DB, tableName, columnName string) int64 {
	query := fmt.Sprintf(`SELECT COUNT(DISTINCT "%s") FROM "%s"`,
		strings.ReplaceAll(columnName, `"`, `""`),
		strings.ReplaceAll(tableName, `"`, `""`))
	var count int64
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// getColumnTypes retrieves column types for a table
func (a *App) getColumnTypes(db *sql.DB, tableName string) (map[string]string, error) {
	query := fmt.Sprintf(`PRAGMA table_info("%s")`, strings.ReplaceAll(tableName, `"`, `""`))
	rows, err := db.Query(query)
	if err != nil {
		query = fmt.Sprintf("DESCRIBE `%s`", strings.ReplaceAll(tableName, "`", "``"))
		rows, err = db.Query(query)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	columnTypes := make(map[string]string)
	cols, _ := rows.Columns()
	numCols := len(cols)

	for rows.Next() {
		if numCols >= 6 {
			var name, colType, nullable, key, defaultValue, extra sql.NullString
			if err := rows.Scan(&name, &colType, &nullable, &key, &defaultValue, &extra); err != nil {
				continue
			}
			columnTypes[name.String] = colType.String
		} else {
			var cid int
			var name, colType string
			var notnull int
			var dfltValue sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err != nil {
				continue
			}
			columnTypes[name] = colType
		}
	}
	if err := rows.Err(); err != nil {
		return columnTypes, fmt.Errorf("error iterating column rows: %w", err)
	}

	return columnTypes, nil
}

// parseOptimizeSuggestions parses LLM response into optimization suggestions
func (a *App) parseOptimizeSuggestions(response string) ([]OptimizeSuggestion, error) {
	jsonStr := response

	// Remove markdown code blocks if present
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json")
		end := strings.LastIndex(response, "```")
		if start != -1 && end != -1 && end > start {
			jsonStr = response[start+7 : end]
		}
	} else if strings.Contains(response, "```") {
		start := strings.Index(response, "```")
		end := strings.LastIndex(response, "```")
		if start != -1 && end != -1 && end > start {
			jsonStr = response[start+3 : end]
		}
	}

	jsonStr = strings.TrimSpace(jsonStr)

	// Find JSON array
	startIdx := strings.Index(jsonStr, "[")
	endIdx := strings.LastIndex(jsonStr, "]")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		jsonStr = jsonStr[startIdx : endIdx+1]
	}

	var suggestions []OptimizeSuggestion
	if err := json.Unmarshal([]byte(jsonStr), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w, JSON: %s", err, jsonStr)
	}

	return suggestions, nil
}
