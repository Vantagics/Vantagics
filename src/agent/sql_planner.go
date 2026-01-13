package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/model"
	einoSchema "github.com/cloudwego/eino/schema"
)

// SQLPlanner implements a three-phase SQL generation workflow:
// Phase 1: Schema Linking - Identify relevant tables and columns
// Phase 2: Logic Planning - Generate query logic before SQL
// Phase 3: SQL Coding - Generate and validate SQL
type SQLPlanner struct {
	chatModel model.ChatModel
	dsService *DataSourceService
	logger    func(string)
}

// SQLPlan represents the result of SQL planning
type SQLPlan struct {
	// Phase 1: Schema Linking
	RelevantTables  []string            `json:"relevant_tables"`
	RelevantColumns map[string][]string `json:"relevant_columns"` // table -> columns
	Relationships   []string            `json:"relationships"`    // detected join relationships

	// Phase 2: Logic Planning
	QueryLogic    string `json:"query_logic"`     // Natural language description
	JoinStrategy  string `json:"join_strategy"`   // How tables will be joined
	FilterLogic   string `json:"filter_logic"`    // Filtering conditions
	AggregateLogic string `json:"aggregate_logic"` // Aggregation strategy

	// Phase 3: SQL Generation
	GeneratedSQL string `json:"generated_sql"`
	SQLDialect   string `json:"sql_dialect"`
	Complexity   string `json:"complexity"` // simple, moderate, complex
}

// SchemaInfo holds enhanced schema information for SQL planning
type SchemaInfo struct {
	DataSourceID  string                   `json:"data_source_id"`
	DatabaseType  string                   `json:"database_type"`
	Tables        []TableSchemaInfo        `json:"tables"`
	Relationships []RelationshipInfo       `json:"relationships"`
}

// TableSchemaInfo holds detailed table information
type TableSchemaInfo struct {
	Name        string       `json:"name"`
	Columns     []ColumnInfo `json:"columns"`
	SampleData  []map[string]interface{} `json:"sample_data"`
	RowCount    int          `json:"row_count,omitempty"`
	DDL         string       `json:"ddl,omitempty"`
}

// ColumnInfo holds column details
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Nullable bool   `json:"nullable,omitempty"`
	IsPK     bool   `json:"is_primary_key,omitempty"`
	IsFK     bool   `json:"is_foreign_key,omitempty"`
	FKRef    string `json:"fk_reference,omitempty"` // e.g., "orders.customer_id"
}

// RelationshipInfo describes table relationships
type RelationshipInfo struct {
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
	Type       string `json:"type"` // "one-to-many", "many-to-one", "many-to-many"
}

// NewSQLPlanner creates a new SQL planner
func NewSQLPlanner(chatModel model.ChatModel, dsService *DataSourceService, logger func(string)) *SQLPlanner {
	return &SQLPlanner{
		chatModel: chatModel,
		dsService: dsService,
		logger:    logger,
	}
}

// GetEnhancedSchema retrieves enhanced schema information for SQL planning
func (p *SQLPlanner) GetEnhancedSchema(ctx context.Context, dataSourceID string) (*SchemaInfo, error) {
	sources, err := p.dsService.LoadDataSources()
	if err != nil {
		return nil, err
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == dataSourceID {
			target = &ds
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("data source not found: %s", dataSourceID)
	}

	// Determine database type
	dbType := "sqlite"
	if target.Config.DBPath != "" {
		dbType = "sqlite"
	} else if target.Type == "mysql" || target.Type == "doris" {
		dbType = target.Type
	}

	// Get all tables
	tables, err := p.dsService.GetDataSourceTables(dataSourceID)
	if err != nil {
		return nil, err
	}

	schemaInfo := &SchemaInfo{
		DataSourceID: dataSourceID,
		DatabaseType: dbType,
		Tables:       make([]TableSchemaInfo, 0),
		Relationships: make([]RelationshipInfo, 0),
	}

	// Collect table information
	for _, tableName := range tables {
		tableInfo := TableSchemaInfo{
			Name:    tableName,
			Columns: make([]ColumnInfo, 0),
		}

		// Get columns
		cols, err := p.dsService.GetDataSourceTableColumns(dataSourceID, tableName)
		if err == nil {
			for _, col := range cols {
				colInfo := ColumnInfo{Name: col}
				// Detect potential foreign keys by naming convention
				lowerCol := strings.ToLower(col)
				if strings.HasSuffix(lowerCol, "_id") && lowerCol != "id" {
					colInfo.IsFK = true
					// Try to infer reference table
					refTable := strings.TrimSuffix(lowerCol, "_id")
					for _, t := range tables {
						if strings.ToLower(t) == refTable || strings.ToLower(t) == refTable+"s" {
							colInfo.FKRef = t + ".id"
							schemaInfo.Relationships = append(schemaInfo.Relationships, RelationshipInfo{
								FromTable:  tableName,
								FromColumn: col,
								ToTable:    t,
								ToColumn:   "id",
								Type:       "many-to-one",
							})
							break
						}
					}
				}
				if lowerCol == "id" {
					colInfo.IsPK = true
				}
				tableInfo.Columns = append(tableInfo.Columns, colInfo)
			}
		}

		// Get sample data (3 rows)
		data, err := p.dsService.GetDataSourceTableData(dataSourceID, tableName, 3)
		if err == nil {
			// Truncate long values
			for i := range data {
				for k, v := range data[i] {
					if str, ok := v.(string); ok && len(str) > 50 {
						data[i][k] = str[:47] + "..."
					}
				}
			}
			tableInfo.SampleData = data
		}

		// Get row count (approximate)
		count, err := p.dsService.GetDataSourceTableCount(dataSourceID, tableName)
		if err == nil {
			tableInfo.RowCount = count
		}

		schemaInfo.Tables = append(schemaInfo.Tables, tableInfo)
	}

	return schemaInfo, nil
}

// FormatSchemaForLLM formats schema info for LLM consumption
func (p *SQLPlanner) FormatSchemaForLLM(schema *SchemaInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Database Schema (Type: %s)\n\n", strings.ToUpper(schema.DatabaseType)))

	// Add dialect-specific hints
	sb.WriteString(p.getSQLDialectHints(schema.DatabaseType))
	sb.WriteString("\n")

	// Add tables with DDL-like format
	sb.WriteString("### Tables\n\n")
	for _, table := range schema.Tables {
		sb.WriteString(fmt.Sprintf("**Table: %s**", table.Name))
		if table.RowCount > 0 {
			sb.WriteString(fmt.Sprintf(" (~%d rows)", table.RowCount))
		}
		sb.WriteString("\n```sql\n")

		// Generate DDL-like description
		sb.WriteString(fmt.Sprintf("-- Table: %s\n", table.Name))
		sb.WriteString("Columns:\n")
		for _, col := range table.Columns {
			marker := "  "
			if col.IsPK {
				marker = "PK"
			} else if col.IsFK {
				marker = "FK"
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s", marker, col.Name))
			if col.FKRef != "" {
				sb.WriteString(fmt.Sprintf(" -> %s", col.FKRef))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("```\n")

		// Add sample data
		if len(table.SampleData) > 0 {
			sb.WriteString("Sample data:\n```json\n")
			sampleJSON, _ := json.MarshalIndent(table.SampleData, "", "  ")
			sb.WriteString(string(sampleJSON))
			sb.WriteString("\n```\n")
		}
		sb.WriteString("\n")
	}

	// Add relationships
	if len(schema.Relationships) > 0 {
		sb.WriteString("### Detected Relationships (for JOIN)\n")
		for _, rel := range schema.Relationships {
			sb.WriteString(fmt.Sprintf("- %s.%s → %s.%s (%s)\n",
				rel.FromTable, rel.FromColumn, rel.ToTable, rel.ToColumn, rel.Type))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Phase1SchemaLinking performs schema linking - identifies relevant tables and columns
func (p *SQLPlanner) Phase1SchemaLinking(ctx context.Context, userQuery string, schema *SchemaInfo) (*SQLPlan, error) {
	if p.logger != nil {
		p.logger("[SQL-PLANNER] Phase 1: Schema Linking")
	}

	schemaText := p.FormatSchemaForLLM(schema)

	prompt := fmt.Sprintf(`You are a database expert performing SCHEMA LINKING.

## Task
Analyze the user's query and identify which tables and columns are needed.

## User Query
"%s"

## Available Schema
%s

## Instructions
1. List ONLY the tables needed for this query
2. For each table, list ONLY the columns that will be used
3. Identify any JOIN relationships needed
4. Do NOT write SQL yet - just identify the schema elements

## Output Format (JSON)
{
  "relevant_tables": ["table1", "table2"],
  "relevant_columns": {
    "table1": ["col1", "col2"],
    "table2": ["col3", "col4"]
  },
  "relationships": ["table1.fk_col = table2.id"],
  "reasoning": "Brief explanation of why these tables/columns are needed"
}

Output ONLY valid JSON, no other text.`, userQuery, schemaText)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: "You are a SQL expert specializing in schema analysis. Output only valid JSON."},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("schema linking failed: %v", err)
	}

	// Parse response
	plan := &SQLPlan{}
	content := strings.TrimSpace(resp.Content)
	// Extract JSON from markdown code blocks if present
	if idx := strings.Index(content, "```json"); idx >= 0 {
		content = content[idx+7:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	} else if idx := strings.Index(content, "```"); idx >= 0 {
		content = content[idx+3:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	}
	content = strings.TrimSpace(content)

	var linkResult struct {
		RelevantTables  []string            `json:"relevant_tables"`
		RelevantColumns map[string][]string `json:"relevant_columns"`
		Relationships   []string            `json:"relationships"`
		Reasoning       string              `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(content), &linkResult); err != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[SQL-PLANNER] Failed to parse schema linking result: %v", err))
		}
		// Fallback: use all tables
		for _, t := range schema.Tables {
			plan.RelevantTables = append(plan.RelevantTables, t.Name)
		}
	} else {
		plan.RelevantTables = linkResult.RelevantTables
		plan.RelevantColumns = linkResult.RelevantColumns
		plan.Relationships = linkResult.Relationships
	}

	plan.SQLDialect = schema.DatabaseType
	return plan, nil
}

// Phase2LogicPlanning generates query logic before SQL
func (p *SQLPlanner) Phase2LogicPlanning(ctx context.Context, userQuery string, plan *SQLPlan, schema *SchemaInfo) error {
	if p.logger != nil {
		p.logger("[SQL-PLANNER] Phase 2: Logic Planning")
	}

	// Build focused schema with only relevant tables
	var relevantSchema strings.Builder
	for _, table := range schema.Tables {
		for _, relTable := range plan.RelevantTables {
			if table.Name == relTable {
				relevantSchema.WriteString(fmt.Sprintf("Table %s: ", table.Name))
				var cols []string
				for _, col := range table.Columns {
					cols = append(cols, col.Name)
				}
				relevantSchema.WriteString(strings.Join(cols, ", "))
				relevantSchema.WriteString("\n")
				break
			}
		}
	}

	prompt := fmt.Sprintf(`You are a database expert creating a QUERY PLAN.

## User Query
"%s"

## Selected Tables and Columns
%s

## Detected Relationships
%s

## Instructions
Before writing SQL, describe your query logic in plain language:
1. How will you JOIN the tables? (which columns?)
2. What filtering conditions are needed?
3. What aggregations or calculations?
4. What is the expected output format?

## Output Format (JSON)
{
  "query_logic": "Step by step description of what the query will do",
  "join_strategy": "How tables will be joined (e.g., 'orders INNER JOIN customers ON orders.customer_id = customers.id')",
  "filter_logic": "Filtering conditions (e.g., 'Filter orders from last 30 days')",
  "aggregate_logic": "Aggregation strategy (e.g., 'GROUP BY customer_id, SUM(amount)')",
  "complexity": "simple|moderate|complex"
}

Output ONLY valid JSON, no other text.`,
		userQuery,
		relevantSchema.String(),
		strings.Join(plan.Relationships, "\n"))

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: "You are a SQL expert specializing in query planning. Output only valid JSON."},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return fmt.Errorf("logic planning failed: %v", err)
	}

	// Parse response
	content := strings.TrimSpace(resp.Content)
	// Extract JSON from markdown code blocks if present
	if idx := strings.Index(content, "```json"); idx >= 0 {
		content = content[idx+7:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	} else if idx := strings.Index(content, "```"); idx >= 0 {
		content = content[idx+3:]
		if endIdx := strings.Index(content, "```"); endIdx >= 0 {
			content = content[:endIdx]
		}
	}
	content = strings.TrimSpace(content)

	var logicResult struct {
		QueryLogic     string `json:"query_logic"`
		JoinStrategy   string `json:"join_strategy"`
		FilterLogic    string `json:"filter_logic"`
		AggregateLogic string `json:"aggregate_logic"`
		Complexity     string `json:"complexity"`
	}

	if err := json.Unmarshal([]byte(content), &logicResult); err != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[SQL-PLANNER] Failed to parse logic planning result: %v", err))
		}
		plan.QueryLogic = "Direct query generation"
		plan.Complexity = "simple"
	} else {
		plan.QueryLogic = logicResult.QueryLogic
		plan.JoinStrategy = logicResult.JoinStrategy
		plan.FilterLogic = logicResult.FilterLogic
		plan.AggregateLogic = logicResult.AggregateLogic
		plan.Complexity = logicResult.Complexity
	}

	return nil
}

// Phase3SQLGeneration generates the final SQL query
func (p *SQLPlanner) Phase3SQLGeneration(ctx context.Context, userQuery string, plan *SQLPlan, schema *SchemaInfo) error {
	if p.logger != nil {
		p.logger("[SQL-PLANNER] Phase 3: SQL Generation")
	}

	// Build focused schema with only relevant tables
	var relevantSchema strings.Builder
	for _, table := range schema.Tables {
		for _, relTable := range plan.RelevantTables {
			if table.Name == relTable {
				relevantSchema.WriteString(fmt.Sprintf("Table: %s\n", table.Name))
				relevantSchema.WriteString("Columns: ")
				var cols []string
				for _, col := range table.Columns {
					cols = append(cols, col.Name)
				}
				relevantSchema.WriteString(strings.Join(cols, ", "))
				relevantSchema.WriteString("\n")
				if len(table.SampleData) > 0 {
					sampleJSON, _ := json.Marshal(table.SampleData[0])
					relevantSchema.WriteString(fmt.Sprintf("Sample: %s\n", string(sampleJSON)))
				}
				relevantSchema.WriteString("\n")
				break
			}
		}
	}

	dialectHints := p.getSQLDialectHints(schema.DatabaseType)

	prompt := fmt.Sprintf(`You are a database expert writing SQL.

## Database Type
%s

## SQL Dialect Rules
%s

## User Query
"%s"

## Query Plan
Logic: %s
Join Strategy: %s
Filters: %s
Aggregation: %s

## Available Schema (ONLY use these tables and columns)
%s

## Critical Rules
1. ONLY use columns that exist in the schema above - NO hallucination!
2. Use correct SQL dialect for %s
3. Include appropriate LIMIT clause (default 1000)
4. Handle NULL values with COALESCE where needed
5. Use proper date formatting for %s

## Output Format
Output ONLY the SQL query, wrapped in sql code block:
` + "```sql\nYOUR SQL HERE\n```",
		strings.ToUpper(schema.DatabaseType),
		dialectHints,
		userQuery,
		plan.QueryLogic,
		plan.JoinStrategy,
		plan.FilterLogic,
		plan.AggregateLogic,
		relevantSchema.String(),
		strings.ToUpper(schema.DatabaseType),
		strings.ToUpper(schema.DatabaseType))

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: p.getSQLExpertSystemPrompt(schema.DatabaseType)},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return fmt.Errorf("SQL generation failed: %v", err)
	}

	// Extract SQL from response
	content := resp.Content
	sqlRegex := regexp.MustCompile("(?s)```sql\\s*(.+?)\\s*```")
	matches := sqlRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		plan.GeneratedSQL = strings.TrimSpace(matches[1])
	} else {
		// Try without language tag
		sqlRegex = regexp.MustCompile("(?s)```\\s*(.+?)\\s*```")
		matches = sqlRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			plan.GeneratedSQL = strings.TrimSpace(matches[1])
		} else {
			// Use entire content
			plan.GeneratedSQL = strings.TrimSpace(content)
		}
	}

	return nil
}

// PlanAndGenerateSQL performs the complete three-phase SQL generation
func (p *SQLPlanner) PlanAndGenerateSQL(ctx context.Context, userQuery string, dataSourceID string) (*SQLPlan, error) {
	// Get enhanced schema
	schema, err := p.GetEnhancedSchema(ctx, dataSourceID)
	if err != nil {
		return nil, err
	}

	// Phase 1: Schema Linking
	plan, err := p.Phase1SchemaLinking(ctx, userQuery, schema)
	if err != nil {
		return nil, err
	}

	// Phase 2: Logic Planning
	if err := p.Phase2LogicPlanning(ctx, userQuery, plan, schema); err != nil {
		// Non-fatal, continue with SQL generation
		if p.logger != nil {
			p.logger(fmt.Sprintf("[SQL-PLANNER] Logic planning warning: %v", err))
		}
	}

	// Phase 3: SQL Generation
	if err := p.Phase3SQLGeneration(ctx, userQuery, plan, schema); err != nil {
		return nil, err
	}

	return plan, nil
}

// ValidateAndCorrectSQL validates SQL and attempts to correct errors
func (p *SQLPlanner) ValidateAndCorrectSQL(ctx context.Context, sql string, errorMsg string, schema *SchemaInfo) (string, error) {
	if p.logger != nil {
		p.logger("[SQL-PLANNER] Self-correction: Fixing SQL error")
	}

	schemaText := p.FormatSchemaForLLM(schema)
	dialectHints := p.getSQLDialectHints(schema.DatabaseType)

	prompt := fmt.Sprintf(`You are a SQL expert fixing a query error.

## Original SQL
` + "```sql\n%s\n```" + `

## Error Message
%s

## SQL Dialect
%s
%s

## Available Schema
%s

## Instructions
1. Analyze the error
2. Check if column names match the schema exactly
3. Check SQL dialect syntax
4. Output the CORRECTED SQL only

## Output Format
Output ONLY the corrected SQL, wrapped in sql code block:
` + "```sql\nCORRECTED SQL HERE\n```",
		sql, errorMsg, strings.ToUpper(schema.DatabaseType), dialectHints, schemaText)

	msgs := []*einoSchema.Message{
		{Role: einoSchema.System, Content: p.getSQLExpertSystemPrompt(schema.DatabaseType)},
		{Role: einoSchema.User, Content: prompt},
	}

	resp, err := p.chatModel.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}

	// Extract SQL from response
	content := resp.Content
	sqlRegex := regexp.MustCompile("(?s)```sql\\s*(.+?)\\s*```")
	matches := sqlRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	sqlRegex = regexp.MustCompile("(?s)```\\s*(.+?)\\s*```")
	matches = sqlRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	return strings.TrimSpace(content), nil
}

// getSQLExpertSystemPrompt returns the system prompt for SQL generation
func (p *SQLPlanner) getSQLExpertSystemPrompt(dbType string) string {
	return fmt.Sprintf(`## Role
You are a senior database expert, proficient in %s SQL syntax.

## Constraints
1. NO HALLUCINATION: Only use columns and tables that exist in the provided schema.
2. PERFORMANCE: Prefer JOIN over subqueries, always include reasonable LIMIT.
3. SYNTAX: All string literals must use single quotes.
4. SAFETY: Check for NULL handling and division by zero.
5. DIALECT: Use only %s-compatible functions and syntax.

## Date Handling
- Always check sample data to understand date format (YYYY-MM-DD vs YYYYMMDD vs timestamp)
- Use appropriate date functions for %s

## Output
- Output clean, executable SQL
- Include comments for complex logic
- Use proper indentation`, strings.ToUpper(dbType), strings.ToUpper(dbType), strings.ToUpper(dbType))
}

// getSQLDialectHints returns dialect-specific hints
func (p *SQLPlanner) getSQLDialectHints(dbType string) string {
	switch dbType {
	case "sqlite":
		return `SQLite Syntax Rules:
- Date: strftime('%Y', col), strftime('%m', col), strftime('%d', col)
- Concat: col1 || ' ' || col2 (NOT CONCAT())
- COALESCE(a, b) instead of IFNULL()
- SUBSTR(str, start, len)
- NO YEAR(), MONTH(), DAY() functions!
- Current: date('now'), datetime('now')`
	case "mysql", "doris":
		return `MySQL/Doris Syntax Rules:
- Date: YEAR(col), MONTH(col), DAY(col)
- Date format: DATE_FORMAT(col, '%Y-%m')
- Concat: CONCAT(col1, ' ', col2)
- IFNULL(a, b) or COALESCE(a, b)
- SUBSTRING(str, start, len)
- Current: NOW(), CURDATE()`
	default:
		return ""
	}
}
