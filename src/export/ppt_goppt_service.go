package export

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	ppt "github.com/VantageDataChat/GoPPT"
)

// GoPPTService handles PowerPoint generation using GoPPT (pure Go, zero dependencies)
type GoPPTService struct{}

// NewGoPPTService creates a new GoPPT service
func NewGoPPTService() *GoPPTService {
	return &GoPPTService{}
}

// PPT布局常量 - 16:9宽屏比例
const (
	emuPerInch = 914400

	// 页面边距 (EMU)
	gopptMarginLeft   = int64(0.4 * emuPerInch)
	gopptMarginRight  = int64(0.4 * emuPerInch)
	gopptMarginTop    = int64(0.4 * emuPerInch)
	gopptMarginBottom = int64(0.3 * emuPerInch)

	// 内容区域 (EMU)
	gopptContentWidth  = int64(9.2 * emuPerInch)
	gopptContentHeight = int64(4.9 * emuPerInch)
	gopptSlideWidth    = int64(10.0 * emuPerInch)
	gopptSlideHeight   = int64(5.625 * emuPerInch)

	// 字体大小 (pt)
	gopptFontTitle     = 36
	gopptFontSubtitle  = 20
	gopptFontHeading   = 28
	gopptFontBody      = 14
	gopptFontSmall     = 12
	gopptFontTableHead = 11
	gopptFontTableCell = 10
	gopptFontFooter    = 9
)

// helper: create a solid fill
func solidFill(argb string) *ppt.Fill {
	return ppt.NewFill().SetSolid(ppt.NewColor(argb))
}

// helper: set paragraph alignment to center
func alignCenter(p *ppt.Paragraph) {
	p.SetAlignment(ppt.NewAlignment().SetHorizontal(ppt.HorizontalCenter))
}

// helper: set paragraph alignment to right
func alignRight(p *ppt.Paragraph) {
	p.SetAlignment(ppt.NewAlignment().SetHorizontal(ppt.HorizontalRight))
}

// ExportDashboardToPPT exports dashboard data to PowerPoint format using GoPPT
func (s *GoPPTService) ExportDashboardToPPT(data DashboardData) ([]byte, error) {
	p := ppt.New()
	p.GetDocumentProperties().Title = "智能仪表盘报告"
	p.GetDocumentProperties().Creator = "VantageData"

	// Add title slide
	s.addTitleSlide(p, "智能仪表盘报告", data.UserRequest)

	// Add metrics slide if present
	if len(data.Metrics) > 0 {
		s.addMetricsSlide(p, data.Metrics)
	}

	// Add chart slides if present
	if len(data.ChartImages) > 0 {
		s.addChartSlides(p, data.ChartImages)
	}

	// Add insights slides if present
	if len(data.Insights) > 0 {
		s.addInsightsSlides(p, data.Insights)
	}

	// Add table slides if present
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTableSlides(p, data.TableData)
	}

	// Add ending slide
	s.addEndingSlide(p)

	// Save to buffer
	w, err := ppt.NewWriter(p, ppt.WriterPowerPoint2007)
	if err != nil {
		return nil, fmt.Errorf("failed to create PPT writer: %w", err)
	}

	var buf bytes.Buffer
	if err := w.(*ppt.PPTXWriter).WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("failed to save PPT: %w", err)
	}

	return buf.Bytes(), nil
}

// addTitleSlide adds a title slide with optional user request
func (s *GoPPTService) addTitleSlide(p *ppt.Presentation, title string, userRequest string) {
	slide := p.GetActiveSlide()

	// 顶部蓝色装饰条
	topBar := slide.CreateRichTextShape()
	topBar.SetOffsetX(0).SetOffsetY(0)
	topBar.SetWidth(gopptSlideWidth).SetHeight(int64(0.15 * emuPerInch))
	topBar.SetFill(solidFill("FF3B82F6"))

	// Title text
	titleShape := slide.CreateRichTextShape()
	titleShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(1.6 * emuPerInch))
	titleShape.SetWidth(gopptContentWidth).SetHeight(int64(1.0 * emuPerInch))
	tr := titleShape.CreateTextRun(title)
	tr.GetFont().SetSize(gopptFontTitle).SetBold(true).SetColor(ppt.NewColor("FF1E40AF"))
	alignCenter(titleShape.GetActiveParagraph())

	// User request subtitle
	if userRequest != "" {
		reqShape := slide.CreateRichTextShape()
		reqShape.SetOffsetX(int64(1.0 * emuPerInch)).SetOffsetY(int64(2.8 * emuPerInch))
		reqShape.SetWidth(int64(8.0 * emuPerInch)).SetHeight(int64(0.8 * emuPerInch))
		reqShape.SetFill(solidFill("FFF8FAFC"))

		displayRequest := userRequest
		if len([]rune(displayRequest)) > 80 {
			displayRequest = string([]rune(displayRequest)[:77]) + "..."
		}
		reqTr := reqShape.CreateTextRun("「" + displayRequest + "」")
		reqTr.GetFont().SetSize(gopptFontSubtitle).SetColor(ppt.NewColor("FF475569"))
		alignCenter(reqShape.GetActiveParagraph())
	}

	// Timestamp
	tsShape := slide.CreateRichTextShape()
	tsShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(4.0 * emuPerInch))
	tsShape.SetWidth(gopptContentWidth).SetHeight(int64(0.4 * emuPerInch))
	tsTr := tsShape.CreateTextRun(time.Now().Format("2006年01月02日 15:04"))
	tsTr.GetFont().SetSize(gopptFontSmall).SetColor(ppt.NewColor("FF94A3B8"))
	alignCenter(tsShape.GetActiveParagraph())

	// Footer
	footerShape := slide.CreateRichTextShape()
	footerShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(4.8 * emuPerInch))
	footerShape.SetWidth(gopptContentWidth).SetHeight(int64(0.3 * emuPerInch))
	ftTr := footerShape.CreateTextRun("由 VantageData 智能分析系统生成")
	ftTr.GetFont().SetSize(gopptFontFooter).SetColor(ppt.NewColor("FF94A3B8"))
	alignCenter(footerShape.GetActiveParagraph())

	// 底部蓝色装饰条
	bottomBar := slide.CreateRichTextShape()
	bottomBar.SetOffsetX(0).SetOffsetY(int64(5.5 * emuPerInch))
	bottomBar.SetWidth(gopptSlideWidth).SetHeight(int64(0.125 * emuPerInch))
	bottomBar.SetFill(solidFill("FF3B82F6"))
}

// addSlideHeader adds a consistent header to content slides
func (s *GoPPTService) addSlideHeader(slide *ppt.Slide, title string) {
	// 顶部蓝色装饰条
	topBar := slide.CreateRichTextShape()
	topBar.SetOffsetX(0).SetOffsetY(0)
	topBar.SetWidth(gopptSlideWidth).SetHeight(int64(0.08 * emuPerInch))
	topBar.SetFill(solidFill("FF3B82F6"))

	// Title
	titleShape := slide.CreateRichTextShape()
	titleShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(0.3 * emuPerInch))
	titleShape.SetWidth(gopptContentWidth).SetHeight(int64(0.6 * emuPerInch))
	tr := titleShape.CreateTextRun(title)
	tr.GetFont().SetSize(gopptFontHeading).SetBold(true).SetColor(ppt.NewColor("FF1E40AF"))
}

// addMetricsSlide adds a slide with metrics in a visually appealing grid
func (s *GoPPTService) addMetricsSlide(p *ppt.Presentation, metrics []MetricData) {
	slide := p.CreateSlide()
	s.addSlideHeader(slide, "关键指标")

	maxMetrics := 6
	if len(metrics) > maxMetrics {
		metrics = metrics[:maxMetrics]
	}

	cols := 3
	if len(metrics) <= 2 {
		cols = 2
	} else if len(metrics) <= 4 {
		cols = 2
	}

	startY := 1.1
	startX := 0.4
	spacing := 0.15
	boxWidth := (9.2 - float64(cols-1)*spacing) / float64(cols)
	boxHeight := 1.4

	for i, metric := range metrics {
		row := i / cols
		col := i % cols

		x := startX + float64(col)*(boxWidth+spacing)
		y := startY + float64(row)*(boxHeight+spacing)

		metricShape := slide.CreateRichTextShape()
		metricShape.SetOffsetX(int64(x * emuPerInch)).SetOffsetY(int64(y * emuPerInch))
		metricShape.SetWidth(int64(boxWidth * emuPerInch)).SetHeight(int64(boxHeight * emuPerInch))
		metricShape.SetFill(solidFill("FFF8FAFC"))

		// Metric title
		titleTr := metricShape.CreateTextRun(metric.Title)
		titleTr.GetFont().SetSize(gopptFontSmall).SetColor(ppt.NewColor("FF64748B"))
		alignCenter(metricShape.GetActiveParagraph())

		// Metric value
		metricShape.CreateParagraph()
		valueTr := metricShape.CreateTextRun(metric.Value)
		valueTr.GetFont().SetSize(28).SetBold(true).SetColor(ppt.NewColor("FF1E40AF"))
		alignCenter(metricShape.GetActiveParagraph())

		// Metric change
		if metric.Change != "" {
			metricShape.CreateParagraph()
			changeTr := metricShape.CreateTextRun(metric.Change)
			changeTr.GetFont().SetSize(gopptFontSmall)
			alignCenter(metricShape.GetActiveParagraph())

			if strings.HasPrefix(metric.Change, "+") || strings.Contains(metric.Change, "增") || strings.Contains(metric.Change, "升") {
				changeTr.GetFont().SetColor(ppt.NewColor("FF16A34A"))
			} else if strings.HasPrefix(metric.Change, "-") || strings.Contains(metric.Change, "减") || strings.Contains(metric.Change, "降") {
				changeTr.GetFont().SetColor(ppt.NewColor("FFDC2626"))
			} else {
				changeTr.GetFont().SetColor(ppt.NewColor("FF64748B"))
			}
		}
	}
}

// addChartSlides adds slides with chart images
func (s *GoPPTService) addChartSlides(p *ppt.Presentation, chartImages []string) {
	for i, chartImage := range chartImages {
		slide := p.CreateSlide()
		s.addSlideHeader(slide, fmt.Sprintf("数据可视化 %d", i+1))

		// Extract base64 data
		imageData := chartImage
		mimeType := "image/png"
		if strings.HasPrefix(chartImage, "data:image") {
			parts := strings.SplitN(chartImage, ",", 2)
			if len(parts) == 2 {
				imageData = parts[1]
				if strings.Contains(parts[0], "image/jpeg") {
					mimeType = "image/jpeg"
				} else if strings.Contains(parts[0], "image/gif") {
					mimeType = "image/gif"
				}
			}
		}

		// Decode base64
		imgBytes, err := base64.StdEncoding.DecodeString(imageData)
		if err != nil {
			continue
		}

		// Add image using DrawingShape
		imgShape := slide.CreateDrawingShape()
		imgShape.SetImageData(imgBytes, mimeType)
		imgShape.SetOffsetX(int64(0.5 * emuPerInch)).SetOffsetY(int64(1.0 * emuPerInch))
		imgShape.SetWidth(int64(9.0 * emuPerInch)).SetHeight(int64(4.2 * emuPerInch))
	}
}

// addInsightsSlides adds slides with insights (may span multiple slides)
func (s *GoPPTService) addInsightsSlides(p *ppt.Presentation, insights []string) {
	allContent := strings.Join(insights, "\n\n")
	lines := strings.Split(allContent, "\n")

	maxLinesPerSlide := 18
	currentLines := []string{}
	slideNum := 1

	for _, line := range lines {
		wrappedLines := s.wrapText(line, 85)
		for _, wrappedLine := range wrappedLines {
			if len(currentLines) >= maxLinesPerSlide {
				s.createInsightSlide(p, currentLines, slideNum)
				currentLines = []string{}
				slideNum++
			}
			currentLines = append(currentLines, wrappedLine)
		}
	}

	if len(currentLines) > 0 {
		s.createInsightSlide(p, currentLines, slideNum)
	}
}

// createInsightSlide creates a single insight slide
func (s *GoPPTService) createInsightSlide(p *ppt.Presentation, lines []string, slideNum int) {
	slide := p.CreateSlide()

	title := "智能洞察"
	if slideNum > 1 {
		title = fmt.Sprintf("智能洞察（续 %d）", slideNum)
	}
	s.addSlideHeader(slide, title)

	contentShape := slide.CreateRichTextShape()
	contentShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(1.0 * emuPerInch))
	contentShape.SetWidth(gopptContentWidth).SetHeight(int64(4.3 * emuPerInch))

	for i, line := range lines {
		if i > 0 {
			contentShape.CreateParagraph()
		}

		if strings.TrimSpace(line) == "" {
			tr := contentShape.CreateTextRun(" ")
			tr.GetFont().SetSize(6)
			continue
		}

		format := s.parseMarkdown(line)
		tr := contentShape.CreateTextRun(format.text)

		if format.isHeading == 1 {
			tr.GetFont().SetSize(18).SetBold(true).SetColor(ppt.NewColor("FF1E40AF"))
		} else if format.isHeading == 2 {
			tr.GetFont().SetSize(16).SetBold(true).SetColor(ppt.NewColor("FF3B82F6"))
		} else if format.isHeading == 3 || format.isHeading == 4 {
			tr.GetFont().SetSize(gopptFontBody).SetBold(true).SetColor(ppt.NewColor("FF475569"))
		} else {
			tr.GetFont().SetSize(gopptFontBody).SetColor(ppt.NewColor("FF334155"))
		}
	}
}

// addTableSlides adds slides with data tables (may span multiple slides)
func (s *GoPPTService) addTableSlides(p *ppt.Presentation, tableData *TableData) {
	maxCols := 6
	cols := tableData.Columns
	if len(cols) > maxCols {
		cols = cols[:maxCols]
	}

	maxRowsPerSlide := 14
	rows := tableData.Data
	totalRows := len(rows)

	slideNum := 1
	for startRow := 0; startRow < totalRows; startRow += maxRowsPerSlide {
		endRow := startRow + maxRowsPerSlide
		if endRow > totalRows {
			endRow = totalRows
		}
		pageRows := rows[startRow:endRow]
		s.createTableSlide(p, cols, pageRows, slideNum, startRow, endRow, totalRows, len(tableData.Columns) > maxCols)
		slideNum++
	}

	if totalRows == 0 {
		s.createTableSlide(p, cols, [][]interface{}{}, 1, 0, 0, 0, false)
	}
}

// createTableSlide creates a single table slide
func (s *GoPPTService) createTableSlide(p *ppt.Presentation, cols []TableColumn, rows [][]interface{}, slideNum int, startRow int, endRow int, totalRows int, colsTruncated bool) {
	slide := p.CreateSlide()

	title := "数据表格"
	if slideNum > 1 {
		title = fmt.Sprintf("数据表格（第 %d 页）", slideNum)
	}
	s.addSlideHeader(slide, title)

	tableStartY := 1.0
	tableWidth := 9.2
	colWidth := tableWidth / float64(len(cols))
	headerHeight := 0.35
	rowHeight := 0.28

	// Table header
	headerShape := slide.CreateRichTextShape()
	headerShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(tableStartY * emuPerInch))
	headerShape.SetWidth(int64(tableWidth * emuPerInch)).SetHeight(int64(headerHeight * emuPerInch))
	headerShape.SetFill(solidFill("FF3B82F6"))

	headerText := ""
	for i, col := range cols {
		if i > 0 {
			headerText += "    │    "
		}
		colTitle := col.Title
		colRunes := []rune(colTitle)
		maxColLen := int(colWidth * 3.5)
		if maxColLen < 12 {
			maxColLen = 12
		}
		if len(colRunes) > maxColLen {
			colTitle = string(colRunes[:maxColLen-2]) + ".."
		}
		headerText += colTitle
	}
	headerTr := headerShape.CreateTextRun(headerText)
	headerTr.GetFont().SetSize(gopptFontTableHead).SetBold(true).SetColor(ppt.ColorWhite)
	alignCenter(headerShape.GetActiveParagraph())

	// Data rows
	currentY := tableStartY + headerHeight
	for rowIdx, rowData := range rows {
		rowShape := slide.CreateRichTextShape()
		rowShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(currentY * emuPerInch))
		rowShape.SetWidth(int64(tableWidth * emuPerInch)).SetHeight(int64(rowHeight * emuPerInch))

		if rowIdx%2 == 0 {
			rowShape.SetFill(solidFill("FFF8FAFC"))
		} else {
			rowShape.SetFill(solidFill("FFF1F5F9"))
		}

		rowText := ""
		for i := 0; i < len(cols) && i < len(rowData); i++ {
			if i > 0 {
				rowText += "    │    "
			}
			cellValue := fmt.Sprintf("%v", rowData[i])
			cellRunes := []rune(cellValue)
			maxCellLen := int(colWidth * 3.5)
			if maxCellLen < 12 {
				maxCellLen = 12
			}
			if len(cellRunes) > maxCellLen {
				cellValue = string(cellRunes[:maxCellLen-2]) + ".."
			}
			rowText += cellValue
		}
		rowTr := rowShape.CreateTextRun(rowText)
		rowTr.GetFont().SetSize(gopptFontTableCell).SetColor(ppt.NewColor("FF334155"))
		alignCenter(rowShape.GetActiveParagraph())

		currentY += rowHeight
	}

	// Pagination info
	if totalRows > 0 {
		infoShape := slide.CreateRichTextShape()
		infoShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(5.2 * emuPerInch))
		infoShape.SetWidth(gopptContentWidth).SetHeight(int64(0.25 * emuPerInch))

		infoText := fmt.Sprintf("显示第 %d-%d 行，共 %d 行", startRow+1, endRow, totalRows)
		if colsTruncated {
			infoText += "（列数已截断）"
		}
		infoTr := infoShape.CreateTextRun(infoText)
		infoTr.GetFont().SetSize(gopptFontFooter).SetColor(ppt.NewColor("FF94A3B8"))
		alignRight(infoShape.GetActiveParagraph())
	}
}

// addEndingSlide adds a closing slide
func (s *GoPPTService) addEndingSlide(p *ppt.Presentation) {
	slide := p.CreateSlide()

	// 顶部装饰
	topBar := slide.CreateRichTextShape()
	topBar.SetOffsetX(0).SetOffsetY(0)
	topBar.SetWidth(gopptSlideWidth).SetHeight(int64(0.15 * emuPerInch))
	topBar.SetFill(solidFill("FF3B82F6"))

	// Thank you
	thankShape := slide.CreateRichTextShape()
	thankShape.SetOffsetX(int64(1.0 * emuPerInch)).SetOffsetY(int64(2.0 * emuPerInch))
	thankShape.SetWidth(int64(8.0 * emuPerInch)).SetHeight(int64(1.0 * emuPerInch))
	thankTr := thankShape.CreateTextRun("感谢查阅")
	thankTr.GetFont().SetSize(36).SetBold(true).SetColor(ppt.NewColor("FF1E40AF"))
	alignCenter(thankShape.GetActiveParagraph())

	// Subtitle
	subShape := slide.CreateRichTextShape()
	subShape.SetOffsetX(int64(1.0 * emuPerInch)).SetOffsetY(int64(3.2 * emuPerInch))
	subShape.SetWidth(int64(8.0 * emuPerInch)).SetHeight(int64(0.5 * emuPerInch))
	subTr := subShape.CreateTextRun("数据驱动决策，智能赋能未来")
	subTr.GetFont().SetSize(18).SetColor(ppt.NewColor("FF64748B"))
	alignCenter(subShape.GetActiveParagraph())

	// Footer
	footerShape := slide.CreateRichTextShape()
	footerShape.SetOffsetX(gopptMarginLeft).SetOffsetY(int64(4.8 * emuPerInch))
	footerShape.SetWidth(gopptContentWidth).SetHeight(int64(0.3 * emuPerInch))
	ftTr := footerShape.CreateTextRun("VantageData 智能分析系统 · " + time.Now().Format("2006"))
	ftTr.GetFont().SetSize(gopptFontFooter).SetColor(ppt.NewColor("FF94A3B8"))
	alignCenter(footerShape.GetActiveParagraph())

	// 底部装饰
	bottomBar := slide.CreateRichTextShape()
	bottomBar.SetOffsetX(0).SetOffsetY(int64(5.5 * emuPerInch))
	bottomBar.SetWidth(gopptSlideWidth).SetHeight(int64(0.125 * emuPerInch))
	bottomBar.SetFill(solidFill("FF3B82F6"))
}

// gopptLineFormat represents parsed markdown line format
type gopptLineFormat struct {
	text      string
	isBold    bool
	isHeading int
	isList    bool
}

// parseMarkdown parses markdown formatting for PPT
func (s *GoPPTService) parseMarkdown(line string) gopptLineFormat {
	result := gopptLineFormat{text: line}

	if strings.HasPrefix(line, "#### ") {
		result.isHeading = 4
		result.text = strings.TrimPrefix(line, "#### ")
		result.isBold = true
	} else if strings.HasPrefix(line, "### ") {
		result.isHeading = 3
		result.text = strings.TrimPrefix(line, "### ")
		result.isBold = true
	} else if strings.HasPrefix(line, "## ") {
		result.isHeading = 2
		result.text = strings.TrimPrefix(line, "## ")
		result.isBold = true
	} else if strings.HasPrefix(line, "# ") {
		result.isHeading = 1
		result.text = strings.TrimPrefix(line, "# ")
		result.isBold = true
	}

	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		result.isList = true
		result.text = "• " + strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
	}

	result.text = s.stripMarkdownBold(result.text)
	return result
}

// stripMarkdownBold removes ** and __ markers
func (s *GoPPTService) stripMarkdownBold(text string) string {
	for strings.Contains(text, "**") {
		start := strings.Index(text, "**")
		end := strings.Index(text[start+2:], "**")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+2:start+2+end] + text[start+2+end+2:]
	}
	for strings.Contains(text, "__") {
		start := strings.Index(text, "__")
		end := strings.Index(text[start+2:], "__")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+2:start+2+end] + text[start+2+end+2:]
	}
	return text
}

// wrapText wraps text to fit within maxLen characters
func (s *GoPPTService) wrapText(text string, maxLen int) []string {
	if len(text) == 0 {
		return []string{""}
	}

	var lines []string
	runes := []rune(text)

	if s.containsChinese(text) {
		maxLen = maxLen * 2 / 3
	}

	for len(runes) > 0 {
		if len(runes) <= maxLen {
			lines = append(lines, string(runes))
			break
		}

		breakPoint := maxLen
		for i := maxLen; i > maxLen/2; i-- {
			if runes[i] == ' ' || runes[i] == '，' || runes[i] == '。' || runes[i] == '、' || runes[i] == '；' {
				breakPoint = i + 1
				break
			}
		}

		lines = append(lines, string(runes[:breakPoint]))
		runes = runes[breakPoint:]

		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}

	return lines
}

// containsChinese checks if text contains Chinese characters
func (s *GoPPTService) containsChinese(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
