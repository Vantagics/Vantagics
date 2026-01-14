package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type DataSourceContextTool struct {
	dsService    *DataSourceService
	sqlCollector *SQLCollector // Optional: for tracking schema context
}

func NewDataSourceContextTool(dsService *DataSourceService) *DataSourceContextTool {
	return &DataSourceContextTool{
		dsService: dsService,
	}
}

// SetSQLCollector injects the SQL collector for schema context tracking
func (t *DataSourceContextTool) SetSQLCollector(collector *SQLCollector) {
	t.sqlCollector = collector
}

type dataSourceContextInput struct {
	DataSourceID string   `json:"data_source_id"`
	TableNames   []string `json:"table_names,omitempty"`
}

func (t *DataSourceContextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_data_source_context",
		Desc: `Get the database schema with DDL-like format, sample data, and detected relationships.
IMPORTANT: Always call this FIRST before writing any SQL to understand:
1. Exact column names (case-sensitive!)
2. Data types and formats (especially dates)
3. Table relationships for JOINs
4. Sample data to understand value patterns`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"data_source_id": {
				Type:     schema.String,
				Desc:     "The ID of the data source to inspect.",
				Required: true,
			},
			"table_names": {
				Type:     schema.Array,
				Desc:     "Optional list of table names to inspect details for (max 5).",
				Required: false,
			},
		}),
	}, nil
}

func (t *DataSourceContextTool) InvokableRun(ctx context.Context, input string, opts ...tool.Option) (string, error) {
	var in dataSourceContextInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}

	// 0. Get data source info and determine database type
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

	// Determine database type
	dbType := "sqlite"
	if target.Config.DBPath != "" {
		dbType = "sqlite"
	} else if target.Type == "mysql" || target.Type == "doris" {
		dbType = target.Type
	}

	// 1. Get Tables
	allTables, err := t.dsService.GetDataSourceTables(in.DataSourceID)
	if err != nil {
		return "", err
	}

	// 2. Determine target tables
	var targetTables []string
	if len(in.TableNames) > 0 {
		// Limit to max 5 tables per request
		targetTables = in.TableNames
		if len(targetTables) > 5 {
			targetTables = targetTables[:5]
		}
	} else {
		// If explicit tables not provided, check count
		if len(allTables) > 5 {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("## Database Schema (ID: %s, Type: %s)\n\n", in.DataSourceID, strings.ToUpper(dbType)))
			sb.WriteString(t.getSQLDialectHints(dbType))
			sb.WriteString(fmt.Sprintf("\n⚠️ Database has %d tables. Call this tool with specific 'table_names' to see details.\n\n", len(allTables)))
			sb.WriteString("### Available Tables:\n")
			for _, tbl := range allTables {
				sb.WriteString(fmt.Sprintf("- %s\n", tbl))
			}
			return sb.String(), nil
		}
		targetTables = allTables
	}

	// 3. Detect relationships between tables
	relationships := t.detectRelationships(in.DataSourceID, allTables)

	// 4. Build Enhanced Context String
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Database Schema (ID: %s, Type: %s)\n\n", in.DataSourceID, strings.ToUpper(dbType)))
	sb.WriteString(t.getSQLDialectHints(dbType))
	sb.WriteString("\n")

	// Add detected relationships first (important for JOINs)
	if len(relationships) > 0 {
		sb.WriteString("### 🔗 Detected Relationships (for JOIN)\n")
		for _, rel := range relationships {
			sb.WriteString(fmt.Sprintf("- %s\n", rel))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("### 📊 Table Details\n\n")
	for _, tableName := range targetTables {
		sb.WriteString(fmt.Sprintf("#### Table: `%s`\n", tableName))

		// Get Columns with type hints
		cols, err := t.dsService.GetDataSourceTableColumns(in.DataSourceID, tableName)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- Error fetching columns: %v\n", err))
			continue
		}

		// Get row count
		count, countErr := t.dsService.GetDataSourceTableCount(in.DataSourceID, tableName)

		// DDL-like column listing
		sb.WriteString("```\n")
		sb.WriteString(fmt.Sprintf("-- Table: %s", tableName))
		if countErr == nil {
			sb.WriteString(fmt.Sprintf(" (~%d rows)", count))
		}
		sb.WriteString("\n")
		for _, col := range cols {
			marker := "   "
			lowerCol := strings.ToLower(col)
			if lowerCol == "id" {
				marker = "PK "
			} else if strings.HasSuffix(lowerCol, "_id") {
				marker = "FK "
			}
			sb.WriteString(fmt.Sprintf("[%s] %s\n", marker, col))
		}
		sb.WriteString("```\n")

		// Get sample data (3 rows)
		data, err := t.dsService.GetDataSourceTableData(in.DataSourceID, tableName, 3)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- Error fetching sample: %v\n", err))
			continue
		}

		if len(data) > 0 {
			// Truncate long strings and format nicely
			for i := range data {
				for k, v := range data[i] {
					if str, ok := v.(string); ok {
						if len(str) > 50 {
							data[i][k] = str[:47] + "..."
						}
					}
				}
			}

			sb.WriteString("**Sample Data:**\n```json\n")
			sampleJSON, _ := json.MarshalIndent(data, "", "  ")
			sb.WriteString(string(sampleJSON))
			sb.WriteString("\n```\n")

			// Add data format hints
			sb.WriteString(t.inferDataFormats(data))
		} else {
			sb.WriteString("*(Table is empty)*\n")
		}
		sb.WriteString("\n")
	}

	// Track schema context in SQL collector if available
	if t.sqlCollector != nil {
		schemaCtx := SchemaContext{
			Tables:  targetTables,
			Columns: make(map[string][]string),
		}
		
		// Populate columns for tracked tables
		for _, tableName := range targetTables {
			if cols, err := t.dsService.GetDataSourceTableColumns(in.DataSourceID, tableName); err == nil {
				schemaCtx.Columns[tableName] = cols
			}
		}
		
		t.sqlCollector.SetSchemaContext(schemaCtx)
	}

	result := sb.String()
	// Final safety check: if result is too large, truncate it
	if len(result) > 15000 {
		result = result[:15000] + "\n\n[Output truncated - request fewer tables for full details]"
	}

	return result, nil
}

// detectRelationships infers table relationships based on column naming conventions
func (t *DataSourceContextTool) detectRelationships(dataSourceID string, tables []string) []string {
	var relationships []string
	tableSet := make(map[string]bool)
	for _, tbl := range tables {
		tableSet[strings.ToLower(tbl)] = true
	}

	for _, tableName := range tables {
		cols, err := t.dsService.GetDataSourceTableColumns(dataSourceID, tableName)
		if err != nil {
			continue
		}

		for _, col := range cols {
			lowerCol := strings.ToLower(col)
			if strings.HasSuffix(lowerCol, "_id") && lowerCol != "id" {
				// Extract potential reference table name
				refTable := strings.TrimSuffix(lowerCol, "_id")

				// Check if reference table exists (singular or plural)
				if tableSet[refTable] {
					relationships = append(relationships,
						fmt.Sprintf("%s.%s → %s.id (FK)", tableName, col, refTable))
				} else if tableSet[refTable+"s"] {
					relationships = append(relationships,
						fmt.Sprintf("%s.%s → %ss.id (FK)", tableName, col, refTable))
				} else if tableSet[refTable+"es"] {
					relationships = append(relationships,
						fmt.Sprintf("%s.%s → %ses.id (FK)", tableName, col, refTable))
				}
			}
		}
	}

	return relationships
}

// inferDataFormats analyzes sample data to provide hints about data formats
func (t *DataSourceContextTool) inferDataFormats(data []map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}

	var hints []string
	sample := data[0]

	for col, val := range sample {
		lowerCol := strings.ToLower(col)
		strVal := fmt.Sprintf("%v", val)

		// Detect date formats
		if strings.Contains(lowerCol, "date") || strings.Contains(lowerCol, "time") ||
			strings.Contains(lowerCol, "created") || strings.Contains(lowerCol, "updated") {
			if len(strVal) == 8 && isNumeric(strVal) {
				hints = append(hints, fmt.Sprintf("⚠️ `%s`: Date format is YYYYMMDD (e.g., %s)", col, strVal))
			} else if strings.Contains(strVal, "-") {
				hints = append(hints, fmt.Sprintf("ℹ️ `%s`: Date format is YYYY-MM-DD", col))
			} else if strings.Contains(strVal, "/") {
				hints = append(hints, fmt.Sprintf("⚠️ `%s`: Date format uses slashes", col))
			}
		}
	}

	if len(hints) > 0 {
		return "**Data Format Hints:**\n" + strings.Join(hints, "\n") + "\n"
	}
	return ""
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// getSQLDialectHints returns SQL syntax hints for the specific database type
func (t *DataSourceContextTool) getSQLDialectHints(dbType string) string {
	switch dbType {
	case "sqlite":
		return `
⚠️ SQL Dialect: SQLite - Use these syntax rules:
• Date functions: strftime('%Y', date_col), strftime('%m', date_col), strftime('%d', date_col)
• Date format: strftime('%Y-%m', date_col) for YYYY-MM format
• String concat: col1 || ' ' || col2 (NOT CONCAT())
• INSTR(str, substr) - only 2 parameters!
• COALESCE(a, b) instead of IFNULL()
• No YEAR(), MONTH(), DAY() functions - use strftime()
• SUBSTR(str, start, len) for substring
• Current date: date('now'), datetime('now')
• CAST(col AS INTEGER/REAL/TEXT) for type conversion
`
	case "mysql", "doris":
		return `
⚠️ SQL Dialect: MySQL/Doris - Use these syntax rules:
• Date functions: YEAR(date_col), MONTH(date_col), DAY(date_col)
• Date format: DATE_FORMAT(date_col, '%Y-%m') for YYYY-MM format
• String concat: CONCAT(col1, ' ', col2)
• IFNULL(a, b) or COALESCE(a, b)
• SUBSTRING(str, start, len) for substring
• Current date: NOW(), CURDATE()
• CAST(col AS SIGNED/DECIMAL/CHAR) for type conversion
• GROUP_CONCAT() for aggregating strings
`
	default:
		return ""
	}
}
