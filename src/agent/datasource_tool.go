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
	DataSourceID string `json:"data_source_id"`
}

func (t *DataSourceContextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_data_source_context",
		Desc: "Get the schema and a sample of data for a specific data source. This helps you understand what tables and columns are available.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"data_source_id": {
				Type:     schema.String,
				Desc:     "The ID of the data source to inspect.",
				Required: true,
			},
		}),
	},
    nil
}

func (t *DataSourceContextTool) InvokableRun(ctx context.Context, input string, opts ...tool.Option) (string, error) {
	var in dataSourceContextInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}

	// 1. Get Tables
	tables, err := t.dsService.GetDataSourceTables(in.DataSourceID)
	if err != nil {
		return "", err
	}

	// 2. Build Context String
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Data Source Context (ID: %s)\n\n", in.DataSourceID))

	for _, tableName := range tables {
		sb.WriteString(fmt.Sprintf("Table: %s\n", tableName))
		
		// Get sample data (5 rows)
		data, err := t.dsService.GetDataSourceTableData(in.DataSourceID, tableName, 5)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- Error fetching sample: %v\n", err))
			continue
		}

		if len(data) > 0 {
			// Extract columns
			var cols []string
			for k := range data[0] {
				cols = append(cols, k)
			}
			sb.WriteString(fmt.Sprintf("- Columns: %s\n", strings.Join(cols, ", ")))
			
			// Add sample rows as JSON
			sampleJSON, _ := json.Marshal(data)
			sb.WriteString(fmt.Sprintf("- Sample Data: %s\n", string(sampleJSON)))
		} else {
			sb.WriteString("- (Table is empty)\n")
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
