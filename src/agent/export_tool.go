package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"vantagedata/export"
)

// ExportTool provides PDF, Excel, and PPT export capabilities
type ExportTool struct {
	logger         func(string)
	sessionDir     string
	requestID      string
	onFileSaved    func(fileName, fileType string, fileSize int64)
	pdfService       *export.PDFExportService
	excelService     *export.ExcelExportService
	pptService       *export.GoPPTService
	wordService      *export.WordExportService
}

// ExportInput represents the input for export operations
type ExportInput struct {
	Format      string                   `json:"format" jsonschema:"description=Export format: pdf, excel, ppt, or word"`
	Data        map[string]interface{}   `json:"data" jsonschema:"description=Data to export (structure depends on format)"`
	FileName    string                   `json:"file_name,omitempty" jsonschema:"description=Optional custom filename (without extension)"`
}

// NewExportTool creates a new export tool
func NewExportTool(logger func(string)) *ExportTool {
	return &ExportTool{
		logger:       logger,
		pdfService:   export.NewPDFExportService(),
		excelService: export.NewExcelExportService(),
		pptService:   export.NewGoPPTService(),
		wordService:  export.NewWordExportService(),
	}
}

// SetSessionDirectory sets the session directory for file output
func (t *ExportTool) SetSessionDirectory(dir string) {
	t.sessionDir = dir
}

// SetRequestID sets the request ID for unique file naming
func (t *ExportTool) SetRequestID(requestID string) {
	t.requestID = requestID
}

// SetFileSavedCallback sets the callback for file saved events
func (t *ExportTool) SetFileSavedCallback(callback func(fileName, fileType string, fileSize int64)) {
	t.onFileSaved = callback
}

// Info returns tool information for the LLM
func (t *ExportTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "export_data",
		Desc: `Export data to PDF, Excel, or PowerPoint format.

Use this tool to create professional reports and presentations from analysis results.

**Supported Formats:**
- **excel**: Data tables with formatting and multiple sheets (⭐ PREFERRED for data export)
- **pdf**: Dashboard reports with metrics, insights, charts, and tables
- **ppt**: Presentation slides with metrics, insights, and visualizations

**IMPORTANT: Default Export Format**
- When exporting data/tables, ALWAYS use "excel" format (NOT CSV)
- Excel provides better formatting, multiple sheets, and data type preservation
- Only use PDF for visual reports with charts and insights
- Only use PPT for presentation slides

**When to use:**
- User requests "导出数据" or "导出表格" → Use excel format
- User requests "导出为PDF/Excel/PPT" or "生成报告/演示文稿"
- After completing analysis and user wants to save results
- Creating professional documents for sharing

**Excel Export Data Structure (⭐ PREFERRED for data):**
{
  "sheet_name": "工作表名称",
  "table_data": {
    "columns": [{"title": "列名", "data_type": "string"}],
    "data": [[value1, value2, ...]]
  }
}

**PDF Export Data Structure:**
{
  "user_request": "用户的原始请求",
  "metrics": [
    {"title": "指标名称", "value": "123", "change": "+5%"}
  ],
  "insights": ["洞察1", "洞察2"],
  "chart_images": ["base64_image_data"],
  "table_data": {
    "columns": [{"title": "列名", "data_type": "string"}],
    "data": [[value1, value2, ...]]
  }
}

**PPT Export Data Structure:**
Same as PDF structure

**Example (Data Export - Use Excel):**
{
  "format": "excel",
  "data": {
    "sheet_name": "销售数据",
    "table_data": {
      "columns": [{"title": "日期", "data_type": "string"}, {"title": "销售额", "data_type": "number"}],
      "data": [["2024-01", 12345], ["2024-02", 23456]]
    }
  },
  "file_name": "sales_data"
}

**Example (Visual Report - Use PDF):**
{
  "format": "pdf",
  "data": {
    "user_request": "分析销售趋势",
    "metrics": [{"title": "总销售额", "value": "¥1,234,567", "change": "+15%"}],
    "insights": ["销售额持续增长", "Q4表现最佳"],
    "table_data": {...}
  },
  "file_name": "sales_report"
}

The tool will save the file to the session directory and return the file path.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"format": {
				Type:     schema.String,
				Desc:     "Export format: 'pdf', 'excel', or 'ppt'",
				Required: true,
			},
			"data": {
				Type:     schema.Object,
				Desc:     "Data to export (structure depends on format)",
				Required: true,
			},
			"file_name": {
				Type:     schema.String,
				Desc:     "Optional custom filename (without extension)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the export operation
func (t *ExportTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Parse input
	var input ExportInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	// Validate format
	format := input.Format
	if format != "pdf" && format != "excel" && format != "ppt" {
		return "", fmt.Errorf("unsupported format: %s (must be pdf, excel, or ppt)", format)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[EXPORT] Starting %s export", format))
	}

	// Generate filename with request ID for uniqueness
	// If request ID is available, use it; otherwise fall back to timestamp
	var fileName string
	if input.FileName != "" {
		fileName = input.FileName
	} else if t.requestID != "" {
		fileName = fmt.Sprintf("%s_export", t.requestID)
	} else {
		timestamp := time.Now().Format("20060102_150405")
		fileName = fmt.Sprintf("export_%s", timestamp)
	}

	var filePath string
	var fileBytes []byte
	var err error

	// Route to appropriate export function
	// Files are saved to the "files" subdirectory within the session directory
	filesDir := filepath.Join(t.sessionDir, "files")
	switch format {
	case "pdf":
		fileBytes, err = t.exportPDF(input.Data)
		filePath = filepath.Join(filesDir, fileName+".pdf")
	case "excel":
		fileBytes, err = t.exportExcel(input.Data)
		filePath = filepath.Join(filesDir, fileName+".xlsx")
	case "ppt":
		fileBytes, err = t.exportPPT(input.Data)
		filePath = filepath.Join(filesDir, fileName+".pptx")
	}

	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[EXPORT] Failed: %v", err))
		}
		return "", fmt.Errorf("export failed: %v", err)
	}

	// Save file
	if t.sessionDir != "" {
		// Ensure files directory exists
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %v", err)
		}

		// Write file
		if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %v", err)
		}

		// Notify frontend
		if t.onFileSaved != nil {
			fileInfo, _ := os.Stat(filePath)
			t.onFileSaved(filepath.Base(filePath), format, fileInfo.Size())
		}

		if t.logger != nil {
			t.logger(fmt.Sprintf("[EXPORT] Saved to: %s (%.2f KB)", filePath, float64(len(fileBytes))/1024))
		}

		return fmt.Sprintf("✅ %s文件已生成: %s (%.2f KB)\n\n文件已保存到会话目录，可以在界面中下载。",
			map[string]string{"pdf": "PDF", "excel": "Excel", "ppt": "PPT"}[format],
			filepath.Base(filePath),
			float64(len(fileBytes))/1024), nil
	}

	return "", fmt.Errorf("session directory not set")
}

// exportPDF exports data to PDF format
func (t *ExportTool) exportPDF(data map[string]interface{}) ([]byte, error) {
	// Convert map to DashboardData structure
	dashboardData := export.DashboardData{}

	// Extract user request
	if userRequest, ok := data["user_request"].(string); ok {
		dashboardData.UserRequest = userRequest
	}

	// Extract metrics
	if metricsData, ok := data["metrics"].([]interface{}); ok {
		for _, m := range metricsData {
			if metricMap, ok := m.(map[string]interface{}); ok {
				metric := export.MetricData{}
				if title, ok := metricMap["title"].(string); ok {
					metric.Title = title
				}
				if value, ok := metricMap["value"].(string); ok {
					metric.Value = value
				}
				if change, ok := metricMap["change"].(string); ok {
					metric.Change = change
				}
				dashboardData.Metrics = append(dashboardData.Metrics, metric)
			}
		}
	}

	// Extract insights
	if insightsData, ok := data["insights"].([]interface{}); ok {
		for _, insight := range insightsData {
			if insightStr, ok := insight.(string); ok {
				dashboardData.Insights = append(dashboardData.Insights, insightStr)
			}
		}
	}

	// Extract chart images
	if chartImages, ok := data["chart_images"].([]interface{}); ok {
		for _, img := range chartImages {
			if imgStr, ok := img.(string); ok {
				dashboardData.ChartImages = append(dashboardData.ChartImages, imgStr)
			}
		}
	}

	// Extract table data
	if tableDataMap, ok := data["table_data"].(map[string]interface{}); ok {
		tableData := &export.TableData{}

		// Extract columns
		if columnsData, ok := tableDataMap["columns"].([]interface{}); ok {
			for _, col := range columnsData {
				if colMap, ok := col.(map[string]interface{}); ok {
					column := export.TableColumn{}
					if title, ok := colMap["title"].(string); ok {
						column.Title = title
					}
					if dataType, ok := colMap["data_type"].(string); ok {
						column.DataType = dataType
					}
					tableData.Columns = append(tableData.Columns, column)
				}
			}
		}

		// Extract data rows
		if rowsData, ok := tableDataMap["data"].([]interface{}); ok {
			for _, row := range rowsData {
				if rowArray, ok := row.([]interface{}); ok {
					tableData.Data = append(tableData.Data, rowArray)
				}
			}
		}

		dashboardData.TableData = tableData
	}

	// Generate PDF
	return t.pdfService.ExportDashboardToPDF(dashboardData)
}

// exportExcel exports data to Excel format
func (t *ExportTool) exportExcel(data map[string]interface{}) ([]byte, error) {
	// Extract sheet name
	sheetName := "数据表"
	if name, ok := data["sheet_name"].(string); ok {
		sheetName = name
	}

	// Extract table data
	tableDataMap, ok := data["table_data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("table_data is required for Excel export")
	}

	tableData := &export.TableData{}

	// Extract columns
	if columnsData, ok := tableDataMap["columns"].([]interface{}); ok {
		for _, col := range columnsData {
			if colMap, ok := col.(map[string]interface{}); ok {
				column := export.TableColumn{}
				if title, ok := colMap["title"].(string); ok {
					column.Title = title
				}
				if dataType, ok := colMap["data_type"].(string); ok {
					column.DataType = dataType
				}
				tableData.Columns = append(tableData.Columns, column)
			}
		}
	}

	// Extract data rows
	if rowsData, ok := tableDataMap["data"].([]interface{}); ok {
		for _, row := range rowsData {
			if rowArray, ok := row.([]interface{}); ok {
				tableData.Data = append(tableData.Data, rowArray)
			}
		}
	}

	if len(tableData.Columns) == 0 {
		return nil, fmt.Errorf("no columns found in table_data")
	}

	// Generate Excel
	return t.excelService.ExportTableToExcel(tableData, sheetName)
}

// exportPPT exports data to PowerPoint format
func (t *ExportTool) exportPPT(data map[string]interface{}) ([]byte, error) {
	// Convert map to DashboardData structure (same as PDF)
	dashboardData := export.DashboardData{}

	// Extract user request
	if userRequest, ok := data["user_request"].(string); ok {
		dashboardData.UserRequest = userRequest
	}

	// Extract metrics
	if metricsData, ok := data["metrics"].([]interface{}); ok {
		for _, m := range metricsData {
			if metricMap, ok := m.(map[string]interface{}); ok {
				metric := export.MetricData{}
				if title, ok := metricMap["title"].(string); ok {
					metric.Title = title
				}
				if value, ok := metricMap["value"].(string); ok {
					metric.Value = value
				}
				if change, ok := metricMap["change"].(string); ok {
					metric.Change = change
				}
				dashboardData.Metrics = append(dashboardData.Metrics, metric)
			}
		}
	}

	// Extract insights
	if insightsData, ok := data["insights"].([]interface{}); ok {
		for _, insight := range insightsData {
			if insightStr, ok := insight.(string); ok {
				dashboardData.Insights = append(dashboardData.Insights, insightStr)
			}
		}
	}

	// Extract chart images
	if chartImages, ok := data["chart_images"].([]interface{}); ok {
		for _, img := range chartImages {
			if imgStr, ok := img.(string); ok {
				dashboardData.ChartImages = append(dashboardData.ChartImages, imgStr)
			}
		}
	}

	// Extract table data
	if tableDataMap, ok := data["table_data"].(map[string]interface{}); ok {
		tableData := &export.TableData{}

		// Extract columns
		if columnsData, ok := tableDataMap["columns"].([]interface{}); ok {
			for _, col := range columnsData {
				if colMap, ok := col.(map[string]interface{}); ok {
					column := export.TableColumn{}
					if title, ok := colMap["title"].(string); ok {
						column.Title = title
					}
					if dataType, ok := colMap["data_type"].(string); ok {
						column.DataType = dataType
					}
					tableData.Columns = append(tableData.Columns, column)
				}
			}
		}

		// Extract data rows
		if rowsData, ok := tableDataMap["data"].([]interface{}); ok {
			for _, row := range rowsData {
				if rowArray, ok := row.([]interface{}); ok {
					tableData.Data = append(tableData.Data, rowArray)
				}
			}
		}

		dashboardData.TableData = tableData
	}

	// Use GoPPT (pure Go, zero dependencies)
	pptBytes, err := t.pptService.ExportDashboardToPPT(dashboardData)
	if err != nil {
		return nil, fmt.Errorf("PPT export failed: %w", err)
	}

	return pptBytes, nil
}
