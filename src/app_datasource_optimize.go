package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"vantagedata/agent"
	"strings"

	_ "modernc.org/sqlite"
)

// IndexSuggestion represents a suggested index
type IndexSuggestion struct {
	TableName  string   `json:"table_name"`
	IndexName  string   `json:"index_name"`
	Columns    []string `json:"columns"`
	Reason     string   `json:"reason"`
	SQLCommand string   `json:"sql_command"`
	Applied    bool     `json:"applied"`
	Error      string   `json:"error,omitempty"`
}

// OptimizeSuggestionsResult represents the suggestions result
type OptimizeSuggestionsResult struct {
	DataSourceID   string            `json:"data_source_id"`
	DataSourceName string            `json:"data_source_name"`
	Suggestions    []IndexSuggestion `json:"suggestions"`
	Success        bool              `json:"success"`
	Error          string            `json:"error,omitempty"`
}

// OptimizeDataSourceResult represents the optimization result
type OptimizeDataSourceResult struct {
	DataSourceID   string            `json:"data_source_id"`
	DataSourceName string            `json:"data_source_name"`
	Suggestions    []IndexSuggestion `json:"suggestions"`
	Summary        string            `json:"summary"`
	Success        bool              `json:"success"`
	Error          string            `json:"error,omitempty"`
}

// GetOptimizeSuggestions generates optimization suggestions without applying them
func (a *App) GetOptimizeSuggestions(dataSourceID string) (*OptimizeSuggestionsResult, error) {
	a.Log(fmt.Sprintf("[OPTIMIZE] Generating suggestions for data source: %s", dataSourceID))

	// Check if dataSourceService is initialized
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	// Load data source
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

	// Check if it's a remote database (not allowed to optimize)
	if dataSource.Type == "mysql" || dataSource.Type == "doris" || dataSource.Type == "postgresql" {
		if !dataSource.Config.StoreLocally {
			return &OptimizeSuggestionsResult{
				DataSourceID:   dataSourceID,
				DataSourceName: dataSource.Name,
				Success:        false,
				Error:          fmt.Sprintf("无法优化远程数据库（%s）。为了安全起见，只能优化已导入到本地的数据源。", dataSource.Type),
			}, nil
		}
	}

	// Check if it's a local SQLite database
	if dataSource.Config.DBPath == "" {
		return &OptimizeSuggestionsResult{
			DataSourceID:   dataSourceID,
			DataSourceName: dataSource.Name,
			Success:        false,
			Error:          "数据源没有本地存储，无法优化。请先将数据导入到本地。",
		}, nil
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] Data source: %s, DBPath: %s", dataSource.Name, dataSource.Config.DBPath))

	// Get data cache directory from config
	cfg, err := a.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Get full database path
	dbPath := filepath.Join(cfg.DataCacheDir, dataSource.Config.DBPath)

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get schema information and generate suggestions
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

// ApplyOptimizeSuggestions applies the given optimization suggestions
func (a *App) ApplyOptimizeSuggestions(dataSourceID string, suggestions []IndexSuggestion) (*OptimizeDataSourceResult, error) {
	a.Log(fmt.Sprintf("[OPTIMIZE] Applying %d suggestions for data source: %s", len(suggestions), dataSourceID))

	// Check if dataSourceService is initialized
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	// Load data source
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

	// Check if it's a remote database (not allowed to optimize)
	if dataSource.Type == "mysql" || dataSource.Type == "doris" || dataSource.Type == "postgresql" {
		if !dataSource.Config.StoreLocally {
			return &OptimizeDataSourceResult{
				DataSourceID:   dataSourceID,
				DataSourceName: dataSource.Name,
				Success:        false,
				Error:          fmt.Sprintf("无法优化远程数据库（%s）。为了安全起见，只能优化已导入到本地的数据源。", dataSource.Type),
			}, nil
		}
	}

	// Check if it's a local SQLite database
	if dataSource.Config.DBPath == "" {
		return &OptimizeDataSourceResult{
			DataSourceID:   dataSourceID,
			DataSourceName: dataSource.Name,
			Success:        false,
			Error:          "数据源没有本地存储，无法优化。",
		}, nil
	}

	// Get data cache directory from config
	cfg, err := a.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Get full database path
	dbPath := filepath.Join(cfg.DataCacheDir, dataSource.Config.DBPath)

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Apply suggestions
	appliedCount := 0
	for i := range suggestions {
		// Execute SQL
		_, err := db.Exec(suggestions[i].SQLCommand)
		if err != nil {
			suggestions[i].Applied = false
			suggestions[i].Error = err.Error()
			a.Log(fmt.Sprintf("[OPTIMIZE] Failed to create index %s: %v", suggestions[i].IndexName, err))
		} else {
			suggestions[i].Applied = true
			appliedCount++
			a.Log(fmt.Sprintf("[OPTIMIZE] Created index: %s", suggestions[i].IndexName))
		}
	}

	// Mark data source as optimized
	if appliedCount > 0 {
		dataSource.Config.Optimized = true
		
		// Load all sources, update this one, and save
		sources, err := a.dataSourceService.LoadDataSources()
		if err != nil {
			a.Log(fmt.Sprintf("[OPTIMIZE] Failed to load data sources: %v", err))
		} else {
			found := false
			for i := range sources {
				if sources[i].ID == dataSource.ID {
					sources[i].Config.Optimized = true
					found = true
					break
				}
			}
			
			if found {
				if err := a.dataSourceService.SaveDataSources(sources); err != nil {
					a.Log(fmt.Sprintf("[OPTIMIZE] Failed to save optimized status: %v", err))
				} else {
					a.Log(fmt.Sprintf("[OPTIMIZE] Marked data source as optimized"))
				}
			}
		}
	}

	// Generate summary
	summary := fmt.Sprintf("优化完成：成功创建 %d 个索引，共 %d 个建议", appliedCount, len(suggestions))

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

// generateOptimizeSuggestions generates optimization suggestions using LLM
func (a *App) generateOptimizeSuggestions(db *sql.DB, dataSourceID string) ([]IndexSuggestion, error) {
	// Get tables
	tables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Build schema description for LLM
	var schemaDesc strings.Builder
	schemaDesc.WriteString("Database Schema:\n\n")

	for _, tableName := range tables {
		columns, err := a.dataSourceService.GetDataSourceTableColumns(dataSourceID, tableName)
		if err != nil {
			a.Log(fmt.Sprintf("[OPTIMIZE] Failed to get columns for table %s: %v", tableName, err))
			continue
		}

		// Get column types
		columnTypes, err := a.getColumnTypes(db, tableName)
		if err != nil {
			a.Log(fmt.Sprintf("[OPTIMIZE] Failed to get column types for table %s: %v", tableName, err))
		}

		schemaDesc.WriteString(fmt.Sprintf("Table: %s\n", tableName))
		schemaDesc.WriteString("Columns:\n")
		for _, col := range columns {
			colType := "UNKNOWN"
			if ct, ok := columnTypes[col]; ok {
				colType = ct
			}
			schemaDesc.WriteString(fmt.Sprintf("  - %s (%s)\n", col, colType))
		}
		schemaDesc.WriteString("\n")
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] Schema description:\n%s", schemaDesc.String()))

	// Get existing indexes
	existingIndexes, err := a.getExistingIndexes(db)
	if err != nil {
		a.Log(fmt.Sprintf("[OPTIMIZE] Failed to get existing indexes: %v", err))
	} else {
		schemaDesc.WriteString("Existing Indexes:\n")
		for _, idx := range existingIndexes {
			schemaDesc.WriteString(fmt.Sprintf("  - %s\n", idx))
		}
		schemaDesc.WriteString("\n")
	}

	// Ask LLM for index suggestions
	cfg, err := a.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	prompt := fmt.Sprintf(`You are a database optimization expert. Analyze the following SQLite database schema and suggest indexes to improve query performance.

%s

Please provide index suggestions in JSON format. For each suggestion, include:
- table_name: The table name
- index_name: A descriptive index name (e.g., idx_tablename_column)
- columns: Array of column names to include in the index
- reason: Why this index would improve performance

Consider:
1. Primary keys and foreign keys (if identifiable from column names like *_id)
2. Columns likely used in WHERE clauses (dates, status, categories)
3. Columns used in JOIN operations
4. Columns used in ORDER BY or GROUP BY
5. Avoid creating indexes on very small tables (< 1000 rows estimated)
6. Avoid duplicate indexes

Return ONLY a JSON array of suggestions, no other text:
[
  {
    "table_name": "orders",
    "index_name": "idx_orders_date",
    "columns": ["order_date"],
    "reason": "Improve performance for date range queries"
  }
]`, schemaDesc.String())

	llm := agent.NewLLMService(cfg, a.Log)
	response, err := llm.Chat(context.Background(), prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM suggestions: %w", err)
	}

	a.Log(fmt.Sprintf("[OPTIMIZE] LLM response: %s", response))

	// Parse LLM response
	suggestions, err := a.parseIndexSuggestions(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM suggestions: %w", err)
	}

	// Generate SQL commands for each suggestion
	for i := range suggestions {
		suggestions[i].SQLCommand = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
			suggestions[i].IndexName,
			suggestions[i].TableName,
			strings.Join(suggestions[i].Columns, ", "))
	}

	return suggestions, nil
}

// getColumnTypes retrieves column types for a table
func (a *App) getColumnTypes(db *sql.DB, tableName string) (map[string]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnTypes := make(map[string]string)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}

		columnTypes[name] = colType
	}

	return columnTypes, nil
}

// getExistingIndexes retrieves existing indexes
func (a *App) getExistingIndexes(db *sql.DB) ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_%'"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		indexes = append(indexes, name)
	}

	return indexes, nil
}

// parseIndexSuggestions parses LLM response into index suggestions
func (a *App) parseIndexSuggestions(response string) ([]IndexSuggestion, error) {
	// Extract JSON from response (might be wrapped in markdown code blocks)
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

	var suggestions []IndexSuggestion
	if err := json.Unmarshal([]byte(jsonStr), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w, JSON: %s", err, jsonStr)
	}

	return suggestions, nil
}
