package export

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// PDFExportService handles PDF generation using maroto
type PDFExportService struct{}

// NewPDFExportService creates a new PDF export service
func NewPDFExportService() *PDFExportService {
	return &PDFExportService{}
}

// DashboardData represents dashboard export data
type DashboardData struct {
	UserRequest string
	Metrics     []MetricData
	Insights    []string
	ChartImages []string // base64 encoded images
	TableData   *TableData
}

type MetricData struct {
	Title  string
	Value  string
	Change string
}

type TableData struct {
	Columns []TableColumn
	Data    [][]interface{}
}

type TableColumn struct {
	Title    string
	DataType string
}

// ExportDashboardToPDF exports dashboard data to PDF
// Tries gopdf first (better Chinese support), falls back to maroto if needed
func (s *PDFExportService) ExportDashboardToPDF(data DashboardData) ([]byte, error) {
	// Try gopdf first (has better Chinese font support via TrueType fonts)
	gopdfService := NewGopdfService()
	pdfBytes, err := gopdfService.ExportDashboardToPDF(data)
	if err == nil {
		return pdfBytes, nil
	}
	
	// If gopdf fails (e.g., font not found), try maroto as fallback
	return s.exportWithMaroto(data)
}

// exportWithMaroto uses maroto library (fallback implementation)
func (s *PDFExportService) exportWithMaroto(data DashboardData) ([]byte, error) {
	// Create maroto configuration with Arial font family (better Unicode support)
	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithDefaultFont(&props.Font{
			Family: fontfamily.Arial, // Arial has better Unicode/Chinese support
			Size:   10,
		}).
		Build()

	// Create new maroto instance
	m := maroto.New(cfg)

	// Add header
	s.addHeader(m, "智能仪表盘报告")

	// Add user request section
	if data.UserRequest != "" {
		s.addUserRequest(m, data.UserRequest)
	}

	// Add metrics section
	if len(data.Metrics) > 0 {
		s.addMetrics(m, data.Metrics)
	}

	// Add insights section
	if len(data.Insights) > 0 {
		s.addInsights(m, data.Insights)
	}

	// Add chart images
	if len(data.ChartImages) > 0 {
		s.addCharts(m, data.ChartImages)
	}

	// Add table data
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTable(m, data.TableData)
	}

	// Add footer
	s.addFooter(m)

	// Generate PDF
	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return document.GetBytes(), nil
}

// addHeader adds the report header
func (s *PDFExportService) addHeader(m core.Maroto, title string) {
	m.AddRow(20,
		col.New(12).Add(
			text.New(title, props.Text{
				Family: fontfamily.Arial,
				Size:   18,
				Style:  fontstyle.Bold,
				Align:  align.Center,
				Color:  &props.Color{Red: 59, Green: 130, Blue: 246}, // Blue color
			}),
		),
	)

	// Add timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	m.AddRow(8,
		col.New(12).Add(
			text.New(fmt.Sprintf("生成时间: %s", timestamp), props.Text{
				Family: fontfamily.Arial,
				Size:   9,
				Align:  align.Center,
				Color:  &props.Color{Red: 100, Green: 116, Blue: 139},
			}),
		),
	)

	// Add spacing
	m.AddRow(5)
}

// addUserRequest adds the user request section
func (s *PDFExportService) addUserRequest(m core.Maroto, request string) {
	m.AddRow(8,
		col.New(12).Add(
			text.New("用户请求", props.Text{
				Family: fontfamily.Arial,
				Size:   12,
				Style:  fontstyle.Bold,
			}),
		),
	)

	m.AddRow(10,
		col.New(12).Add(
			text.New(request, props.Text{
				Family: fontfamily.Arial,
				Size:   10,
			}),
		),
	)

	m.AddRow(5)
}

// addMetrics adds the metrics section
func (s *PDFExportService) addMetrics(m core.Maroto, metrics []MetricData) {
	m.AddRow(8,
		col.New(12).Add(
			text.New("关键指标", props.Text{
				Family: fontfamily.Arial,
				Size:   12,
				Style:  fontstyle.Bold,
			}),
		),
	)

	// Add metrics in a grid (2 columns)
	for i := 0; i < len(metrics); i += 2 {
		cols := []core.Col{}

		// First metric
		metric1 := metrics[i]
		cols = append(cols, col.New(6).Add(
			text.New(fmt.Sprintf("%s: %s (%s)", metric1.Title, metric1.Value, metric1.Change), props.Text{
				Family: fontfamily.Arial,
				Size:   9,
			}),
		))

		// Second metric (if exists)
		if i+1 < len(metrics) {
			metric2 := metrics[i+1]
			cols = append(cols, col.New(6).Add(
				text.New(fmt.Sprintf("%s: %s (%s)", metric2.Title, metric2.Value, metric2.Change), props.Text{
					Family: fontfamily.Arial,
					Size:   9,
				}),
			))
		} else {
			cols = append(cols, col.New(6))
		}

		m.AddRow(8, cols...)
	}

	m.AddRow(5)
}

// addInsights adds the insights section
func (s *PDFExportService) addInsights(m core.Maroto, insights []string) {
	m.AddRow(8,
		col.New(12).Add(
			text.New("智能洞察", props.Text{
				Family: fontfamily.Arial,
				Size:   12,
				Style:  fontstyle.Bold,
			}),
		),
	)

	for i, insight := range insights {
		m.AddRow(8,
			col.New(12).Add(
				text.New(fmt.Sprintf("%d. %s", i+1, insight), props.Text{
					Family: fontfamily.Arial,
					Size:   9,
				}),
			),
		)
	}

	m.AddRow(5)
}

// addCharts adds chart images
func (s *PDFExportService) addCharts(m core.Maroto, chartImages []string) {
	m.AddRow(8,
		col.New(12).Add(
			text.New("数据可视化", props.Text{
				Family: fontfamily.Arial,
				Size:   12,
				Style:  fontstyle.Bold,
			}),
		),
	)

	for i, chartImage := range chartImages {
		// Extract base64 data
		imageData := chartImage
		if strings.HasPrefix(chartImage, "data:image") {
			// Remove data URL prefix
			parts := strings.SplitN(chartImage, ",", 2)
			if len(parts) == 2 {
				imageData = parts[1]
			}
		}

		// Decode base64
		imgBytes, err := base64.StdEncoding.DecodeString(imageData)
		if err != nil {
			// Skip invalid images
			continue
		}

		// Add chart title
		m.AddRow(6,
			col.New(12).Add(
				text.New(fmt.Sprintf("图表 %d", i+1), props.Text{
					Family: fontfamily.Arial,
					Size:   10,
					Style:  fontstyle.Bold,
				}),
			),
		)

		// Add image (auto-fit to page width)
		m.AddRow(80,
			col.New(12).Add(
				image.NewFromBytes(imgBytes, extension.Png),
			),
		)

		m.AddRow(5)
	}
}

// addTable adds table data
func (s *PDFExportService) addTable(m core.Maroto, tableData *TableData) {
	m.AddRow(8,
		col.New(12).Add(
			text.New("数据表格", props.Text{
				Family: fontfamily.Arial,
				Size:   12,
				Style:  fontstyle.Bold,
			}),
		),
	)

	// Limit columns to fit page width (max 6 columns)
	maxCols := 6
	if len(tableData.Columns) > maxCols {
		tableData.Columns = tableData.Columns[:maxCols]
	}

	// Calculate column width
	colWidth := 12 / len(tableData.Columns)

	// Add table header
	headerCols := []core.Col{}
	for _, column := range tableData.Columns {
		headerCols = append(headerCols, col.New(colWidth).Add(
			text.New(column.Title, props.Text{
				Family: fontfamily.Arial,
				Size:   8,
				Style:  fontstyle.Bold,
				Align:  align.Center,
			}),
		))
	}
	m.AddRow(7, headerCols...)

	// Add table rows (limit to 50 rows for PDF)
	maxRows := 50
	if len(tableData.Data) > maxRows {
		tableData.Data = tableData.Data[:maxRows]
	}

	for _, rowData := range tableData.Data {
		dataCols := []core.Col{}
		for i := 0; i < len(tableData.Columns) && i < len(rowData); i++ {
			cellValue := fmt.Sprintf("%v", rowData[i])
			// Truncate long text
			if len(cellValue) > 30 {
				cellValue = cellValue[:27] + "..."
			}
			dataCols = append(dataCols, col.New(colWidth).Add(
				text.New(cellValue, props.Text{
					Family: fontfamily.Arial,
					Size:   7,
					Align:  align.Left,
				}),
			))
		}
		m.AddRow(6, dataCols...)
	}

	// Add note if data was truncated
	if len(tableData.Data) > maxRows {
		m.AddRow(6,
			col.New(12).Add(
				text.New(fmt.Sprintf("注：仅显示前%d行数据", maxRows), props.Text{
					Family: fontfamily.Arial,
					Size:   7,
					Style:  fontstyle.Italic,
					Align:  align.Center,
					Color:  &props.Color{Red: 100, Green: 116, Blue: 139},
				}),
			),
		)
	}
}

// addFooter adds the report footer
func (s *PDFExportService) addFooter(m core.Maroto) {
	m.AddRow(10,
		col.New(12).Add(
			text.New("由 VantageData 智能分析系统生成", props.Text{
				Family: fontfamily.Arial,
				Size:   8,
				Align:  align.Center,
				Color:  &props.Color{Red: 148, Green: 163, Blue: 184},
			}),
		),
	)
}
