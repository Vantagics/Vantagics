package export

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/unidoc/unioffice/color"
	"github.com/unidoc/unioffice/measurement"
	"github.com/unidoc/unioffice/presentation"
	"github.com/unidoc/unioffice/schema/soo/dml"
)

// PPTService handles PowerPoint generation using unioffice
type PPTService struct{}

// NewPPTService creates a new PPT service
func NewPPTService() *PPTService {
	return &PPTService{}
}

// ExportDashboardToPPT exports dashboard data to PowerPoint format
func (s *PPTService) ExportDashboardToPPT(data DashboardData) ([]byte, error) {
	// Create new presentation
	ppt := presentation.New()
	defer ppt.Close()

	// Add title slide
	s.addTitleSlide(ppt, "智能仪表盘报告")

	// Add user request slide if present
	if data.UserRequest != "" {
		s.addUserRequestSlide(ppt, data.UserRequest)
	}

	// Add metrics slide if present
	if len(data.Metrics) > 0 {
		s.addMetricsSlide(ppt, data.Metrics)
	}

	// Add insights slide if present
	if len(data.Insights) > 0 {
		s.addInsightsSlide(ppt, data.Insights)
	}

	// Add table slide if present
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTableSlide(ppt, data.TableData)
	}

	// Save to buffer
	var buf bytes.Buffer
	if err := ppt.Save(&buf); err != nil {
		return nil, fmt.Errorf("failed to save PPT: %w", err)
	}

	return buf.Bytes(), nil
}

// addTitleSlide adds a title slide
func (s *PPTService) addTitleSlide(ppt *presentation.Presentation, title string) {
	slide := ppt.AddSlide()

	// Add title text box
	titleBox := slide.AddTextBox()
	titleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	titleBox.Properties().SetPosition(
		measurement.Distance(1*measurement.Inch),
		measurement.Distance(2*measurement.Inch),
	)
	titleBox.Properties().SetSize(
		measurement.Distance(8*measurement.Inch),
		measurement.Distance(1.5*measurement.Inch),
	)

	// Set title text
	para := titleBox.AddParagraph()
	para.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	run := para.AddRun()
	run.SetText(title)
	run.Properties().SetSize(44)
	run.Properties().SetBold(true)
	run.Properties().SetSolidFill(color.RGB(59, 130, 246)) // Blue color

	// Add subtitle
	subtitleBox := slide.AddTextBox()
	subtitleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	subtitleBox.Properties().SetPosition(
		measurement.Distance(1*measurement.Inch),
		measurement.Distance(4*measurement.Inch),
	)
	subtitleBox.Properties().SetSize(
		measurement.Distance(8*measurement.Inch),
		measurement.Distance(0.5*measurement.Inch),
	)

	subPara := subtitleBox.AddParagraph()
	subPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	subRun := subPara.AddRun()
	subRun.SetText("由 VantageData 智能分析系统生成")
	subRun.Properties().SetSize(18)
	subRun.Properties().SetSolidFill(color.RGB(100, 116, 139))
}

// addUserRequestSlide adds a slide with user request
func (s *PPTService) addUserRequestSlide(ppt *presentation.Presentation, request string) {
	slide := ppt.AddSlide()

	// Add title
	titleBox := slide.AddTextBox()
	titleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	titleBox.Properties().SetPosition(
		measurement.Distance(0.5*measurement.Inch),
		measurement.Distance(0.5*measurement.Inch),
	)
	titleBox.Properties().SetSize(
		measurement.Distance(9*measurement.Inch),
		measurement.Distance(0.8*measurement.Inch),
	)

	titlePara := titleBox.AddParagraph()
	titleRun := titlePara.AddRun()
	titleRun.SetText("用户请求")
	titleRun.Properties().SetSize(32)
	titleRun.Properties().SetBold(true)

	// Add content
	contentBox := slide.AddTextBox()
	contentBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	contentBox.Properties().SetPosition(
		measurement.Distance(0.5*measurement.Inch),
		measurement.Distance(1.5*measurement.Inch),
	)
	contentBox.Properties().SetSize(
		measurement.Distance(9*measurement.Inch),
		measurement.Distance(4.5*measurement.Inch),
	)

	contentPara := contentBox.AddParagraph()
	contentRun := contentPara.AddRun()
	contentRun.SetText(request)
	contentRun.Properties().SetSize(20)
}

// addMetricsSlide adds a slide with metrics
func (s *PPTService) addMetricsSlide(ppt *presentation.Presentation, metrics []MetricData) {
	slide := ppt.AddSlide()

	// Add title
	titleBox := slide.AddTextBox()
	titleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	titleBox.Properties().SetPosition(
		measurement.Distance(0.5*measurement.Inch),
		measurement.Distance(0.5*measurement.Inch),
	)
	titleBox.Properties().SetSize(
		measurement.Distance(9*measurement.Inch),
		measurement.Distance(0.8*measurement.Inch),
	)

	titlePara := titleBox.AddParagraph()
	titleRun := titlePara.AddRun()
	titleRun.SetText("关键指标")
	titleRun.Properties().SetSize(32)
	titleRun.Properties().SetBold(true)

	// Add metrics in a grid (2 columns)
	startY := 1.5
	startX := 0.5
	boxWidth := 4.25
	boxHeight := 1.2
	spacing := 0.25

	for i, metric := range metrics {
		row := i / 2
		col := i % 2

		x := startX + float64(col)*(boxWidth+spacing)
		y := startY + float64(row)*(boxHeight+spacing)

		// Create metric box
		metricBox := slide.AddTextBox()
		metricBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
		metricBox.Properties().SetPosition(
			measurement.Distance(measurement.Distance(x)*measurement.Inch),
			measurement.Distance(measurement.Distance(y)*measurement.Inch),
		)
		metricBox.Properties().SetSize(
			measurement.Distance(boxWidth*measurement.Inch),
			measurement.Distance(boxHeight*measurement.Inch),
		)

		// Set background color
		metricBox.Properties().SetSolidFill(color.RGB(240, 248, 255))

		// Add metric title
		titlePara := metricBox.AddParagraph()
		titleRun := titlePara.AddRun()
		titleRun.SetText(metric.Title)
		titleRun.Properties().SetSize(16)
		titleRun.Properties().SetBold(true)

		// Add metric value
		valuePara := metricBox.AddParagraph()
		valueRun := valuePara.AddRun()
		valueRun.SetText(metric.Value)
		valueRun.Properties().SetSize(24)
		valueRun.Properties().SetBold(true)
		valueRun.Properties().SetSolidFill(color.RGB(59, 130, 246))

		// Add metric change
		changePara := metricBox.AddParagraph()
		changeRun := changePara.AddRun()
		changeRun.SetText(metric.Change)
		changeRun.Properties().SetSize(14)

		// Color based on change direction
		if strings.HasPrefix(metric.Change, "+") {
			changeRun.Properties().SetSolidFill(color.RGB(34, 197, 94)) // Green
		} else if strings.HasPrefix(metric.Change, "-") {
			changeRun.Properties().SetSolidFill(color.RGB(239, 68, 68)) // Red
		}
	}
}

// addInsightsSlide adds a slide with insights
func (s *PPTService) addInsightsSlide(ppt *presentation.Presentation, insights []string) {
	slide := ppt.AddSlide()

	// Add title
	titleBox := slide.AddTextBox()
	titleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	titleBox.Properties().SetPosition(
		measurement.Distance(0.5*measurement.Inch),
		measurement.Distance(0.5*measurement.Inch),
	)
	titleBox.Properties().SetSize(
		measurement.Distance(9*measurement.Inch),
		measurement.Distance(0.8*measurement.Inch),
	)

	titlePara := titleBox.AddParagraph()
	titleRun := titlePara.AddRun()
	titleRun.SetText("智能洞察")
	titleRun.Properties().SetSize(32)
	titleRun.Properties().SetBold(true)

	// Add insights
	contentBox := slide.AddTextBox()
	contentBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	contentBox.Properties().SetPosition(
		measurement.Distance(0.5*measurement.Inch),
		measurement.Distance(1.5*measurement.Inch),
	)
	contentBox.Properties().SetSize(
		measurement.Distance(9*measurement.Inch),
		measurement.Distance(5*measurement.Inch),
	)

	for i, insight := range insights {
		para := contentBox.AddParagraph()
		run := para.AddRun()
		run.SetText(fmt.Sprintf("%d. %s", i+1, insight))
		run.Properties().SetSize(16)
	}
}

// addTableSlide adds a slide with a data table (as formatted text)
func (s *PPTService) addTableSlide(ppt *presentation.Presentation, tableData *TableData) {
	slide := ppt.AddSlide()

	// Add title
	titleBox := slide.AddTextBox()
	titleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	titleBox.Properties().SetPosition(
		measurement.Distance(0.5*measurement.Inch),
		measurement.Distance(0.5*measurement.Inch),
	)
	titleBox.Properties().SetSize(
		measurement.Distance(9*measurement.Inch),
		measurement.Distance(0.6*measurement.Inch),
	)

	titlePara := titleBox.AddParagraph()
	titleRun := titlePara.AddRun()
	titleRun.SetText("数据表格")
	titleRun.Properties().SetSize(28)
	titleRun.Properties().SetBold(true)

	// Limit columns and rows for PPT
	maxCols := 5
	maxRows := 12

	cols := tableData.Columns
	if len(cols) > maxCols {
		cols = cols[:maxCols]
	}

	rows := tableData.Data
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	// Create table as formatted text
	tableBox := slide.AddTextBox()
	tableBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	tableBox.Properties().SetPosition(
		measurement.Distance(0.5*measurement.Inch),
		measurement.Distance(1.5*measurement.Inch),
	)
	tableBox.Properties().SetSize(
		measurement.Distance(9*measurement.Inch),
		measurement.Distance(5*measurement.Inch),
	)

	// Add header row
	headerPara := tableBox.AddParagraph()
	headerRun := headerPara.AddRun()
	headerText := ""
	for i, col := range cols {
		if i > 0 {
			headerText += "  |  "
		}
		// Truncate long column names
		colTitle := col.Title
		if len(colTitle) > 15 {
			colTitle = colTitle[:12] + "..."
		}
		headerText += colTitle
	}
	headerRun.SetText(headerText)
	headerRun.Properties().SetSize(11)
	headerRun.Properties().SetBold(true)
	headerRun.Properties().SetSolidFill(color.RGB(68, 114, 196))

	// Add separator
	sepPara := tableBox.AddParagraph()
	sepRun := sepPara.AddRun()
	sepRun.SetText(strings.Repeat("─", 80))
	sepRun.Properties().SetSize(8)
	sepRun.Properties().SetSolidFill(color.RGB(148, 163, 184))

	// Add data rows
	for _, rowData := range rows {
		rowPara := tableBox.AddParagraph()
		rowRun := rowPara.AddRun()
		rowText := ""
		for i := 0; i < len(cols) && i < len(rowData); i++ {
			if i > 0 {
				rowText += "  |  "
			}
			cellValue := fmt.Sprintf("%v", rowData[i])
			// Truncate long values
			if len(cellValue) > 15 {
				cellValue = cellValue[:12] + "..."
			}
			rowText += cellValue
		}
		rowRun.SetText(rowText)
		rowRun.Properties().SetSize(10)
	}

	// Add note if data was truncated
	if len(tableData.Data) > maxRows || len(tableData.Columns) > maxCols {
		noteBox := slide.AddTextBox()
		noteBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
		noteBox.Properties().SetPosition(
			measurement.Distance(0.5*measurement.Inch),
			measurement.Distance(6.5*measurement.Inch),
		)
		noteBox.Properties().SetSize(
			measurement.Distance(9*measurement.Inch),
			measurement.Distance(0.3*measurement.Inch),
		)

		notePara := noteBox.AddParagraph()
		noteRun := notePara.AddRun()
		noteRun.SetText(fmt.Sprintf("注：仅显示前 %d 列 × %d 行数据", len(cols), len(rows)))
		noteRun.Properties().SetSize(10)
		noteRun.Properties().SetSolidFill(color.RGB(100, 116, 139))
	}
}

