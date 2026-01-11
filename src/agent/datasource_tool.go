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
	dsService *DataSourceService
}

func NewDataSourceContextTool(dsService *DataSourceService) *DataSourceContextTool {
	return &DataSourceContextTool{
		dsService: dsService,
	}
}

type dataSourceContextInput struct {
	DataSourceID string   `json:"data_source_id"`
	TableNames   []string `json:"table_names,omitempty"`
}

func (t *DataSourceContextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_data_source_context",
		Desc: "Get the schema and a sample of data for a specific data source. Provide 'table_names' to inspect specific tables. If 'table_names' is omitted, it lists all tables (summary only if too many).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"data_source_id": {
				Type:     schema.String,
				Desc:     "The ID of the data source to inspect.",
				Required: true,
			},
			"table_names": {
				Type:     schema.Array,
				Desc:     "Optional list of table names to inspect details for.",
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
		// Limit to max 3 tables per request to control output size
		targetTables = in.TableNames
		if len(targetTables) > 3 {
			targetTables = targetTables[:3]
		}
	} else {
		// If explicit tables not provided, check count
		if len(allTables) > 3 {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Data Source Context (ID: %s, Database Type: %s)\n", in.DataSourceID, strings.ToUpper(dbType)))
			sb.WriteString(t.getSQLDialectHints(dbType))
			sb.WriteString(fmt.Sprintf("\nDatabase has %d tables. To see details/samples, call this tool again with specific 'table_names' (max 3 at a time).\n\n", len(allTables)))
			sb.WriteString("Tables: " + strings.Join(allTables, ", "))
			return sb.String(), nil
		}
		targetTables = allTables
	}

	// 3. Build Context String for target tables
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Data Source Context (ID: %s, Database Type: %s)\n", in.DataSourceID, strings.ToUpper(dbType)))
	sb.WriteString(t.getSQLDialectHints(dbType))
	sb.WriteString("\n")

	for _, tableName := range targetTables {
		sb.WriteString(fmt.Sprintf("Table: %s\n", tableName))
		
		// 1. Get Columns
		cols, err := t.dsService.GetDataSourceTableColumns(in.DataSourceID, tableName)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- Error fetching columns: %v\n", err))
		} else {
			sb.WriteString(fmt.Sprintf("- Columns: %s\n", strings.Join(cols, ", ")))
		}

		// 2. Get sample data (3 rows to limit context size)
		data, err := t.dsService.GetDataSourceTableData(in.DataSourceID, tableName, 3)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- Error fetching sample: %v\n", err))
			continue
		}

		if len(data) > 0 {
			// Truncate long strings in data to prevent context explosion
			for i := range data {
				for k, v := range data[i] {
					if str, ok := v.(string); ok {
						if len(str) > 50 {
							data[i][k] = str[:47] + "..."
						}
					}
				}
			}

			// Add sample rows as JSON
			sampleJSON, _ := json.Marshal(data)
			sb.WriteString(fmt.Sprintf("- Sample Data: %s\n", string(sampleJSON)))
		} else {
			sb.WriteString("- (Table is empty)\n")
		}
		sb.WriteString("\n")
	}

	result := sb.String()
	// Final safety check: if result is too large, truncate it
	if len(result) > 10000 {
		result = result[:10000] + "\n\n[Output truncated due to length - request fewer tables or specific columns]"
	}

	return result, nil
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
