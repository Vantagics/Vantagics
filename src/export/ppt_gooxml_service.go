package export

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"baliance.com/gooxml/color"
	"baliance.com/gooxml/common"
	"baliance.com/gooxml/measurement"
	"baliance.com/gooxml/presentation"
	"baliance.com/gooxml/schema/soo/dml"
)

// GooxmlPPTService handles PowerPoint generation using gooxml (open source)
type GooxmlPPTService struct{}

// NewGooxmlPPTService creates a new gooxml PPT service
func NewGooxmlPPTService() *GooxmlPPTService {
	return &GooxmlPPTService{}
}

// PPT布局常量 - 16:9宽屏比例 (9144000 EMU x 5143500 EMU = 10" x 5.625")
const (
	// 页面边距
	pptMarginLeft   = 0.4  // 左边距（英寸）
	pptMarginRight  = 0.4  // 右边距（英寸）
	pptMarginTop    = 0.4  // 上边距（英寸）
	pptMarginBottom = 0.3  // 下边距（英寸）

	// 内容区域
	pptContentWidth  = 9.2  // 内容宽度（英寸）= 10 - 0.4 - 0.4
	pptContentHeight = 4.9  // 内容高度（英寸）= 5.625 - 0.4 - 0.3

	// 标题区域
	pptTitleHeight = 0.6 // 标题高度（英寸）
	pptTitleY      = 0.3 // 标题Y位置（英寸）

	// 字体大小
	pptFontTitle     = 36 // 幻灯片标题
	pptFontSubtitle  = 20 // 副标题
	pptFontHeading   = 28 // 内容标题
	pptFontBody      = 14 // 正文
	pptFontSmall     = 12 // 小字
	pptFontTableHead = 11 // 表头
	pptFontTableCell = 10 // 表格单元格
	pptFontFooter    = 9  // 页脚
)

// ExportDashboardToPPT exports dashboard data to PowerPoint format
func (s *GooxmlPPTService) ExportDashboardToPPT(data DashboardData) ([]byte, error) {
	// Create new presentation
	ppt := presentation.New()

	// Add title slide
	s.addTitleSlide(ppt, "智能仪表盘报告", data.UserRequest)

	// Add metrics slide if present (combined with user request for compact layout)
	if len(data.Metrics) > 0 {
		s.addMetricsSlide(ppt, data.Metrics)
	}

	// Add chart slides if present
	if len(data.ChartImages) > 0 {
		s.addChartSlides(ppt, data.ChartImages)
	}

	// Add insights slides if present (may span multiple slides)
	if len(data.Insights) > 0 {
		s.addInsightsSlides(ppt, data.Insights)
	}

	// Add table slides if present (may span multiple slides)
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTableSlides(ppt, data.TableData)
	}

	// Add ending slide
	s.addEndingSlide(ppt)

	// Save to buffer
	var buf bytes.Buffer
	if err := ppt.Save(&buf); err != nil {
		return nil, fmt.Errorf("failed to save PPT: %w", err)
	}

	return buf.Bytes(), nil
}

// addTitleSlide adds a title slide with optional user request
func (s *GooxmlPPTService) addTitleSlide(ppt *presentation.Presentation, title string, userRequest string) {
	slide := ppt.AddSlide()

	// 添加背景装饰条（顶部蓝色条）
	topBar := slide.AddTextBox()
	topBar.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	topBar.Properties().SetPosition(0, 0)
	topBar.Properties().SetSize(10*measurement.Inch, measurement.Distance(0.15)*measurement.Inch)
	topBar.Properties().SetSolidFill(color.RGB(59, 130, 246))

	// Add title text box - 居中显示
	titleBox := slide.AddTextBox()
	titleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	titleBox.Properties().SetPosition(
		measurement.Distance(pptMarginLeft)*measurement.Inch,
		measurement.Distance(1.6)*measurement.Inch,
	)
	titleBox.Properties().SetSize(
		measurement.Distance(pptContentWidth)*measurement.Inch,
		measurement.Distance(1.0)*measurement.Inch,
	)

	// Set title text
	para := titleBox.AddParagraph()
	para.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	run := para.AddRun()
	run.SetText(title)
	run.Properties().SetSize(pptFontTitle)
	run.Properties().SetBold(true)
	run.Properties().SetSolidFill(color.RGB(30, 64, 175)) // 深蓝色

	// Add user request as subtitle if present
	if userRequest != "" {
		requestBox := slide.AddTextBox()
		requestBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
		requestBox.Properties().SetPosition(
			measurement.Distance(1.0)*measurement.Inch,
			measurement.Distance(2.8)*measurement.Inch,
		)
		requestBox.Properties().SetSize(
			measurement.Distance(8.0)*measurement.Inch,
			measurement.Distance(0.8)*measurement.Inch,
		)
		requestBox.Properties().SetSolidFill(color.RGB(248, 250, 252)) // 浅灰背景

		reqPara := requestBox.AddParagraph()
		reqPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
		reqRun := reqPara.AddRun()
		// 截断过长的请求文本
		displayRequest := userRequest
		if len(displayRequest) > 80 {
			displayRequest = displayRequest[:77] + "..."
		}
		reqRun.SetText("「" + displayRequest + "」")
		reqRun.Properties().SetSize(pptFontSubtitle)
		reqRun.Properties().SetSolidFill(color.RGB(71, 85, 105))
	}

	// Add timestamp
	timestampBox := slide.AddTextBox()
	timestampBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	timestampBox.Properties().SetPosition(
		measurement.Distance(pptMarginLeft)*measurement.Inch,
		measurement.Distance(4.0)*measurement.Inch,
	)
	timestampBox.Properties().SetSize(
		measurement.Distance(pptContentWidth)*measurement.Inch,
		measurement.Distance(0.4)*measurement.Inch,
	)

	timePara := timestampBox.AddParagraph()
	timePara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	timeRun := timePara.AddRun()
	timeRun.SetText(time.Now().Format("2006年01月02日 15:04"))
	timeRun.Properties().SetSize(pptFontSmall)
	timeRun.Properties().SetSolidFill(color.RGB(148, 163, 184))

	// Add footer
	footerBox := slide.AddTextBox()
	footerBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	footerBox.Properties().SetPosition(
		measurement.Distance(pptMarginLeft)*measurement.Inch,
		measurement.Distance(4.8)*measurement.Inch,
	)
	footerBox.Properties().SetSize(
		measurement.Distance(pptContentWidth)*measurement.Inch,
		measurement.Distance(0.3)*measurement.Inch,
	)

	footerPara := footerBox.AddParagraph()
	footerPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	footerRun := footerPara.AddRun()
	footerRun.SetText("由 VantageData 智能分析系统生成")
	footerRun.Properties().SetSize(pptFontFooter)
	footerRun.Properties().SetSolidFill(color.RGB(148, 163, 184))

	// 添加底部装饰条
	bottomBar := slide.AddTextBox()
	bottomBar.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	bottomBar.Properties().SetPosition(0, measurement.Distance(5.5)*measurement.Inch)
	bottomBar.Properties().SetSize(10*measurement.Inch, measurement.Distance(0.125)*measurement.Inch)
	bottomBar.Properties().SetSolidFill(color.RGB(59, 130, 246))
}

// addSlideHeader adds a consistent header to content slides
func (s *GooxmlPPTService) addSlideHeader(slide presentation.Slide, title string) {
	// 顶部蓝色装饰条
	topBar := slide.AddTextBox()
	topBar.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	topBar.Properties().SetPosition(0, 0)
	topBar.Properties().SetSize(10*measurement.Inch, measurement.Distance(0.08)*measurement.Inch)
	topBar.Properties().SetSolidFill(color.RGB(59, 130, 246))

	// Add title
	titleBox := slide.AddTextBox()
	titleBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	titleBox.Properties().SetPosition(
		measurement.Distance(pptMarginLeft)*measurement.Inch,
		measurement.Distance(pptTitleY)*measurement.Inch,
	)
	titleBox.Properties().SetSize(
		measurement.Distance(pptContentWidth)*measurement.Inch,
		measurement.Distance(pptTitleHeight)*measurement.Inch,
	)

	titlePara := titleBox.AddParagraph()
	titleRun := titlePara.AddRun()
	titleRun.SetText(title)
	titleRun.Properties().SetSize(pptFontHeading)
	titleRun.Properties().SetBold(true)
	titleRun.Properties().SetSolidFill(color.RGB(30, 64, 175))
}

// addMetricsSlide adds a slide with metrics in a visually appealing grid
func (s *GooxmlPPTService) addMetricsSlide(ppt *presentation.Presentation, metrics []MetricData) {
	slide := ppt.AddSlide()
	s.addSlideHeader(slide, "关键指标")

	// 计算网格布局 - 最多显示6个指标（2行3列）
	maxMetrics := 6
	if len(metrics) > maxMetrics {
		metrics = metrics[:maxMetrics]
	}

	// 根据指标数量决定布局
	cols := 3
	if len(metrics) <= 2 {
		cols = 2
	} else if len(metrics) <= 4 {
		cols = 2
	}

	// 计算每个卡片的尺寸
	startY := 1.1
	startX := pptMarginLeft
	spacing := 0.15
	boxWidth := (pptContentWidth - float64(cols-1)*spacing) / float64(cols)
	boxHeight := 1.4

	for i, metric := range metrics {
		row := i / cols
		col := i % cols

		x := startX + float64(col)*(boxWidth+spacing)
		y := startY + float64(row)*(boxHeight+spacing)

		// Create metric card with rounded appearance
		metricBox := slide.AddTextBox()
		metricBox.Properties().SetGeometry(dml.ST_ShapeTypeRoundRect)
		metricBox.Properties().SetPosition(
			measurement.Distance(x)*measurement.Inch,
			measurement.Distance(y)*measurement.Inch,
		)
		metricBox.Properties().SetSize(
			measurement.Distance(boxWidth)*measurement.Inch,
			measurement.Distance(boxHeight)*measurement.Inch,
		)

		// 设置渐变背景色
		metricBox.Properties().SetSolidFill(color.RGB(248, 250, 252))

		// Add metric title
		titlePara := metricBox.AddParagraph()
		titlePara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
		titleRun := titlePara.AddRun()
		titleRun.SetText(metric.Title)
		titleRun.Properties().SetSize(pptFontSmall)
		titleRun.Properties().SetSolidFill(color.RGB(100, 116, 139))

		// Add metric value - 大字体突出显示
		valuePara := metricBox.AddParagraph()
		valuePara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
		valueRun := valuePara.AddRun()
		valueRun.SetText(metric.Value)
		valueRun.Properties().SetSize(28)
		valueRun.Properties().SetBold(true)
		valueRun.Properties().SetSolidFill(color.RGB(30, 64, 175))

		// Add metric change with color coding
		if metric.Change != "" {
			changePara := metricBox.AddParagraph()
			changePara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
			changeRun := changePara.AddRun()
			changeRun.SetText(metric.Change)
			changeRun.Properties().SetSize(pptFontSmall)

			// Color based on change direction
			if strings.HasPrefix(metric.Change, "+") || strings.Contains(metric.Change, "增") || strings.Contains(metric.Change, "升") {
				changeRun.Properties().SetSolidFill(color.RGB(22, 163, 74)) // Green
			} else if strings.HasPrefix(metric.Change, "-") || strings.Contains(metric.Change, "减") || strings.Contains(metric.Change, "降") {
				changeRun.Properties().SetSolidFill(color.RGB(220, 38, 38)) // Red
			} else {
				changeRun.Properties().SetSolidFill(color.RGB(100, 116, 139))
			}
		}
	}
}

// addChartSlides adds slides with chart images
func (s *GooxmlPPTService) addChartSlides(ppt *presentation.Presentation, chartImages []string) {
	for i, chartImage := range chartImages {
		slide := ppt.AddSlide()
		s.addSlideHeader(slide, fmt.Sprintf("数据可视化 %d", i+1))

		// Extract base64 data
		imageData := chartImage
		if strings.HasPrefix(chartImage, "data:image") {
			parts := strings.SplitN(chartImage, ",", 2)
			if len(parts) == 2 {
				imageData = parts[1]
			}
		}

		// Decode base64
		imgBytes, err := base64.StdEncoding.DecodeString(imageData)
		if err != nil {
			continue
		}

		// 创建临时文件保存图片
		tmpFile, err := s.createTempImageFile(imgBytes)
		if err != nil {
			continue
		}
		defer func(path string) {
			// 清理临时文件（延迟到PPT保存后）
			// 注意：由于gooxml在Save时才真正写入图片，这里不能立即删除
		}(tmpFile)

		// Add image to presentation from file
		imgRef, err := common.ImageFromFile(tmpFile)
		if err != nil {
			continue
		}

		img, err := ppt.AddImage(imgRef)
		if err != nil {
			continue
		}

		// Add image to slide - 居中显示，最大化利用空间
		imgBox := slide.AddImage(img)
		imgBox.Properties().SetPosition(
			measurement.Distance(0.5)*measurement.Inch,
			measurement.Distance(1.0)*measurement.Inch,
		)
		// 图表尺寸：宽9英寸，高4.2英寸（保持16:9比例）
		imgBox.Properties().SetSize(
			measurement.Distance(9.0)*measurement.Inch,
			measurement.Distance(4.2)*measurement.Inch,
		)
	}
}

// createTempImageFile creates a temporary file with image data
func (s *GooxmlPPTService) createTempImageFile(imgBytes []byte) (string, error) {
	// 检测图片格式
	ext := ".png"
	if len(imgBytes) > 2 {
		if imgBytes[0] == 0xFF && imgBytes[1] == 0xD8 {
			ext = ".jpg"
		} else if imgBytes[0] == 0x47 && imgBytes[1] == 0x49 {
			ext = ".gif"
		}
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "ppt_chart_*"+ext)
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// 写入图片数据
	_, err = tmpFile.Write(imgBytes)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// addInsightsSlides adds slides with insights (may span multiple slides)
func (s *GooxmlPPTService) addInsightsSlides(ppt *presentation.Presentation, insights []string) {
	// 合并所有洞察内容
	allContent := strings.Join(insights, "\n\n")
	lines := strings.Split(allContent, "\n")

	// 每页最多显示的行数（根据字体大小和页面高度计算）
	maxLinesPerSlide := 18
	currentLines := []string{}
	slideNum := 1

	for _, line := range lines {
		// 处理长行，进行换行
		wrappedLines := s.wrapTextForPPT(line, 85)
		
		for _, wrappedLine := range wrappedLines {
			if len(currentLines) >= maxLinesPerSlide {
				// 创建新幻灯片
				s.createInsightSlide(ppt, currentLines, slideNum, len(insights))
				currentLines = []string{}
				slideNum++
			}
			currentLines = append(currentLines, wrappedLine)
		}
	}

	// 处理剩余内容
	if len(currentLines) > 0 {
		s.createInsightSlide(ppt, currentLines, slideNum, len(insights))
	}
}

// createInsightSlide creates a single insight slide
func (s *GooxmlPPTService) createInsightSlide(ppt *presentation.Presentation, lines []string, slideNum int, totalInsights int) {
	slide := ppt.AddSlide()
	
	title := "智能洞察"
	if slideNum > 1 {
		title = fmt.Sprintf("智能洞察（续 %d）", slideNum)
	}
	s.addSlideHeader(slide, title)

	// Add content box
	contentBox := slide.AddTextBox()
	contentBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	contentBox.Properties().SetPosition(
		measurement.Distance(pptMarginLeft)*measurement.Inch,
		measurement.Distance(1.0)*measurement.Inch,
	)
	contentBox.Properties().SetSize(
		measurement.Distance(pptContentWidth)*measurement.Inch,
		measurement.Distance(4.3)*measurement.Inch,
	)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			// 空行添加小间距
			emptyPara := contentBox.AddParagraph()
			emptyRun := emptyPara.AddRun()
			emptyRun.SetText(" ")
			emptyRun.Properties().SetSize(6)
			continue
		}

		// 解析Markdown格式
		format := s.parseMarkdownForPPT(line)

		para := contentBox.AddParagraph()
		run := para.AddRun()
		run.SetText(format.text)

		// 根据格式设置样式
		if format.isHeading == 1 {
			run.Properties().SetSize(18)
			run.Properties().SetBold(true)
			run.Properties().SetSolidFill(color.RGB(30, 64, 175))
		} else if format.isHeading == 2 {
			run.Properties().SetSize(16)
			run.Properties().SetBold(true)
			run.Properties().SetSolidFill(color.RGB(59, 130, 246))
		} else if format.isHeading == 3 || format.isHeading == 4 {
			run.Properties().SetSize(pptFontBody)
			run.Properties().SetBold(true)
			run.Properties().SetSolidFill(color.RGB(71, 85, 105))
		} else if format.isList {
			run.Properties().SetSize(pptFontBody)
			run.Properties().SetSolidFill(color.RGB(51, 65, 85))
		} else {
			run.Properties().SetSize(pptFontBody)
			run.Properties().SetSolidFill(color.RGB(51, 65, 85))
		}
	}
}

// pptLineFormat represents parsed markdown line format for PPT
type pptLineFormat struct {
	text      string
	isBold    bool
	isHeading int
	isList    bool
	indent    int
}

// parseMarkdownForPPT parses markdown formatting for PPT
func (s *GooxmlPPTService) parseMarkdownForPPT(line string) pptLineFormat {
	result := pptLineFormat{text: line}

	// Check for headings
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

	// Check for list items
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		result.isList = true
		result.text = "• " + strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
		result.indent = indent / 2
	} else if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && (trimmed[1] == '.' || (len(trimmed) > 2 && trimmed[2] == '.')) {
		result.isList = true
		result.indent = indent / 2
	}

	// Strip inline bold markers
	result.text = s.stripMarkdownBoldPPT(result.text)

	return result
}

// stripMarkdownBoldPPT removes ** and __ markers
func (s *GooxmlPPTService) stripMarkdownBoldPPT(text string) string {
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

// wrapTextForPPT wraps text to fit within maxLen characters
func (s *GooxmlPPTService) wrapTextForPPT(text string, maxLen int) []string {
	if len(text) == 0 {
		return []string{""}
	}

	var lines []string
	runes := []rune(text)

	// 中文文本使用更短的行长度
	if s.containsChinesePPT(text) {
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

// containsChinesePPT checks if text contains Chinese characters
func (s *GooxmlPPTService) containsChinesePPT(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// addInsightsSlide adds a slide with insights (legacy single slide version)
func (s *GooxmlPPTService) addInsightsSlide(ppt *presentation.Presentation, insights []string) {
	s.addInsightsSlides(ppt, insights)
}

// addTableSlides adds slides with data tables (may span multiple slides)
func (s *GooxmlPPTService) addTableSlides(ppt *presentation.Presentation, tableData *TableData) {
	// 限制列数
	maxCols := 6
	cols := tableData.Columns
	if len(cols) > maxCols {
		cols = cols[:maxCols]
	}

	// 每页最多显示的行数
	maxRowsPerSlide := 14
	rows := tableData.Data
	totalRows := len(rows)

	// 分页处理
	slideNum := 1
	for startRow := 0; startRow < totalRows; startRow += maxRowsPerSlide {
		endRow := startRow + maxRowsPerSlide
		if endRow > totalRows {
			endRow = totalRows
		}

		pageRows := rows[startRow:endRow]
		s.createTableSlide(ppt, cols, pageRows, slideNum, startRow, endRow, totalRows, len(tableData.Columns) > maxCols)
		slideNum++
	}

	// 如果没有数据，至少创建一个空表格幻灯片
	if totalRows == 0 {
		s.createTableSlide(ppt, cols, [][]interface{}{}, 1, 0, 0, 0, false)
	}
}

// createTableSlide creates a single table slide
func (s *GooxmlPPTService) createTableSlide(ppt *presentation.Presentation, cols []TableColumn, rows [][]interface{}, slideNum int, startRow int, endRow int, totalRows int, colsTruncated bool) {
	slide := ppt.AddSlide()

	title := "数据表格"
	if slideNum > 1 {
		title = fmt.Sprintf("数据表格（第 %d 页）", slideNum)
	}
	s.addSlideHeader(slide, title)

	// 计算表格布局
	tableStartY := 1.0
	tableWidth := pptContentWidth
	colWidth := tableWidth / float64(len(cols))

	// 表头行高和数据行高
	headerHeight := 0.35
	rowHeight := 0.28

	// 创建表头
	headerBox := slide.AddTextBox()
	headerBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	headerBox.Properties().SetPosition(
		measurement.Distance(pptMarginLeft)*measurement.Inch,
		measurement.Distance(tableStartY)*measurement.Inch,
	)
	headerBox.Properties().SetSize(
		measurement.Distance(tableWidth)*measurement.Inch,
		measurement.Distance(headerHeight)*measurement.Inch,
	)
	headerBox.Properties().SetSolidFill(color.RGB(59, 130, 246))

	// 表头文本
	headerPara := headerBox.AddParagraph()
	headerPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	headerText := ""
	for i, col := range cols {
		if i > 0 {
			headerText += "    │    "
		}
		colTitle := col.Title
		maxColLen := int(colWidth * 5)
		if maxColLen < 8 {
			maxColLen = 8
		}
		if len(colTitle) > maxColLen {
			colTitle = colTitle[:maxColLen-2] + ".."
		}
		headerText += colTitle
	}
	headerRun := headerPara.AddRun()
	headerRun.SetText(headerText)
	headerRun.Properties().SetSize(pptFontTableHead)
	headerRun.Properties().SetBold(true)
	headerRun.Properties().SetSolidFill(color.RGB(255, 255, 255))

	// 创建数据行
	currentY := tableStartY + headerHeight

	for rowIdx, rowData := range rows {
		// 交替行背景色
		rowBox := slide.AddTextBox()
		rowBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
		rowBox.Properties().SetPosition(
			measurement.Distance(pptMarginLeft)*measurement.Inch,
			measurement.Distance(currentY)*measurement.Inch,
		)
		rowBox.Properties().SetSize(
			measurement.Distance(tableWidth)*measurement.Inch,
			measurement.Distance(rowHeight)*measurement.Inch,
		)

		if rowIdx%2 == 0 {
			rowBox.Properties().SetSolidFill(color.RGB(248, 250, 252))
		} else {
			rowBox.Properties().SetSolidFill(color.RGB(241, 245, 249))
		}

		// 行数据文本
		rowPara := rowBox.AddParagraph()
		rowPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
		rowText := ""
		for i := 0; i < len(cols) && i < len(rowData); i++ {
			if i > 0 {
				rowText += "    │    "
			}
			cellValue := fmt.Sprintf("%v", rowData[i])
			maxCellLen := int(colWidth * 5)
			if maxCellLen < 8 {
				maxCellLen = 8
			}
			if len(cellValue) > maxCellLen {
				cellValue = cellValue[:maxCellLen-2] + ".."
			}
			rowText += cellValue
		}
		rowRun := rowPara.AddRun()
		rowRun.SetText(rowText)
		rowRun.Properties().SetSize(pptFontTableCell)
		rowRun.Properties().SetSolidFill(color.RGB(51, 65, 85))

		currentY += rowHeight
	}

	// 添加分页信息
	if totalRows > 0 {
		infoBox := slide.AddTextBox()
		infoBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
		infoBox.Properties().SetPosition(
			measurement.Distance(pptMarginLeft)*measurement.Inch,
			measurement.Distance(5.2)*measurement.Inch,
		)
		infoBox.Properties().SetSize(
			measurement.Distance(pptContentWidth)*measurement.Inch,
			measurement.Distance(0.25)*measurement.Inch,
		)

		infoPara := infoBox.AddParagraph()
		infoPara.Properties().SetAlign(dml.ST_TextAlignTypeR)
		infoRun := infoPara.AddRun()

		infoText := fmt.Sprintf("显示第 %d-%d 行，共 %d 行", startRow+1, endRow, totalRows)
		if colsTruncated {
			infoText += "（列数已截断）"
		}
		infoRun.SetText(infoText)
		infoRun.Properties().SetSize(pptFontFooter)
		infoRun.Properties().SetSolidFill(color.RGB(148, 163, 184))
	}
}

// addTableSlide adds a slide with a data table (legacy single slide version)
func (s *GooxmlPPTService) addTableSlide(ppt *presentation.Presentation, tableData *TableData) {
	s.addTableSlides(ppt, tableData)
}

// addEndingSlide adds a closing slide
func (s *GooxmlPPTService) addEndingSlide(ppt *presentation.Presentation) {
	slide := ppt.AddSlide()

	// 添加背景装饰
	topBar := slide.AddTextBox()
	topBar.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	topBar.Properties().SetPosition(0, 0)
	topBar.Properties().SetSize(10*measurement.Inch, measurement.Distance(0.15)*measurement.Inch)
	topBar.Properties().SetSolidFill(color.RGB(59, 130, 246))

	// Thank you text
	thankBox := slide.AddTextBox()
	thankBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	thankBox.Properties().SetPosition(
		measurement.Distance(1.0)*measurement.Inch,
		measurement.Distance(2.0)*measurement.Inch,
	)
	thankBox.Properties().SetSize(
		measurement.Distance(8.0)*measurement.Inch,
		measurement.Distance(1.0)*measurement.Inch,
	)

	thankPara := thankBox.AddParagraph()
	thankPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	thankRun := thankPara.AddRun()
	thankRun.SetText("感谢查阅")
	thankRun.Properties().SetSize(36)
	thankRun.Properties().SetBold(true)
	thankRun.Properties().SetSolidFill(color.RGB(30, 64, 175))

	// Subtitle
	subBox := slide.AddTextBox()
	subBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	subBox.Properties().SetPosition(
		measurement.Distance(1.0)*measurement.Inch,
		measurement.Distance(3.2)*measurement.Inch,
	)
	subBox.Properties().SetSize(
		measurement.Distance(8.0)*measurement.Inch,
		measurement.Distance(0.5)*measurement.Inch,
	)

	subPara := subBox.AddParagraph()
	subPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	subRun := subPara.AddRun()
	subRun.SetText("数据驱动决策，智能赋能未来")
	subRun.Properties().SetSize(18)
	subRun.Properties().SetSolidFill(color.RGB(100, 116, 139))

	// Footer
	footerBox := slide.AddTextBox()
	footerBox.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	footerBox.Properties().SetPosition(
		measurement.Distance(pptMarginLeft)*measurement.Inch,
		measurement.Distance(4.8)*measurement.Inch,
	)
	footerBox.Properties().SetSize(
		measurement.Distance(pptContentWidth)*measurement.Inch,
		measurement.Distance(0.3)*measurement.Inch,
	)

	footerPara := footerBox.AddParagraph()
	footerPara.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	footerRun := footerPara.AddRun()
	footerRun.SetText("VantageData 智能分析系统 · " + time.Now().Format("2006"))
	footerRun.Properties().SetSize(pptFontFooter)
	footerRun.Properties().SetSolidFill(color.RGB(148, 163, 184))

	// 底部装饰条
	bottomBar := slide.AddTextBox()
	bottomBar.Properties().SetGeometry(dml.ST_ShapeTypeRect)
	bottomBar.Properties().SetPosition(0, measurement.Distance(5.5)*measurement.Inch)
	bottomBar.Properties().SetSize(10*measurement.Inch, measurement.Distance(0.125)*measurement.Inch)
	bottomBar.Properties().SetSolidFill(color.RGB(59, 130, 246))
}
