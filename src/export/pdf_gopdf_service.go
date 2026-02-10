package export

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"vantagedata/i18n"

	gopdf "github.com/VantageDataChat/GoPDF2"
)

// GopdfService handles PDF generation using gopdf with better Chinese support
type GopdfService struct{}

// NewGopdfService creates a new gopdf service
func NewGopdfService() *GopdfService {
	return &GopdfService{}
}

// PDF布局常量 - A4纸张 (595.28 x 841.89 points)
// gopdf 使用 point 作为单位，1 point = 1/72 inch
const (
	// 页面尺寸 (points)
	pdfPageWidth  = 595.28 // A4宽度
	pdfPageHeight = 841.89 // A4高度

	// 页面边距 (points) - 窄边距最大化内容区域
	pdfMarginLeft   = 36.0 // 左边距 (~12.7mm)
	pdfMarginRight  = 36.0 // 右边距
	pdfMarginTop    = 45.0 // 上边距 (~15.9mm)
	pdfMarginBottom = 45.0 // 下边距

	// 内容区域 (points)
	pdfContentWidth = 523.28 // 内容宽度 = 595.28 - 36 - 36

	// 字体大小 (points)
	pdfFontTitle     = 24.0 // 报告标题
	pdfFontHeading1  = 16.0 // 一级标题
	pdfFontHeading2  = 14.0 // 二级标题
	pdfFontHeading3  = 12.0 // 三级标题
	pdfFontBody      = 11.0 // 正文
	pdfFontSmall     = 10.0 // 小字
	pdfFontTableHead = 10.0 // 表头
	pdfFontTableCell = 9.0  // 表格单元格
	pdfFontFooter    = 9.0  // 页脚

	// 行高 (points)
	pdfLineHeightTitle   = 30.0 // 标题行高
	pdfLineHeightHeading = 22.0 // 小标题行高
	pdfLineHeightBody    = 16.0 // 正文行高
	pdfLineHeightTable   = 18.0 // 表格行高
)

// ExportDashboardToPDF exports dashboard data to PDF using gopdf
func (s *GopdfService) ExportDashboardToPDF(data DashboardData) ([]byte, error) {
	// 检查是否是纯分析结果导出（只有 Insights，没有其他数据）
	isAnalysisOnly := len(data.Insights) > 0 &&
		len(data.Metrics) == 0 &&
		len(data.ChartImages) == 0 &&
		(data.TableData == nil || len(data.TableData.Columns) == 0) &&
		len(data.AllTableData) == 0

	if isAnalysisOnly {
		return s.exportAnalysisResultToPDF(data)
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	// Add first page
	pdf.AddPage()

	// Try multiple font paths for better compatibility
	fontPaths := []struct {
		name string
		path string
	}{
		{"msyh", "C:\\Windows\\Fonts\\msyh.ttc"},
		{"msyhbd", "C:\\Windows\\Fonts\\msyhbd.ttc"},
		{"simsun", "C:\\Windows\\Fonts\\simsun.ttc"},
		{"simhei", "C:\\Windows\\Fonts\\simhei.ttf"},
		{"arialuni", "C:\\Windows\\Fonts\\ARIALUNI.TTF"},
		{"arial", "C:\\Windows\\Fonts\\arial.ttf"},
	}

	var fontName string
	var err error

	for _, font := range fontPaths {
		err = pdf.AddTTFFont(font.name, font.path)
		if err == nil {
			fontName = font.name
			break
		}
	}

	if fontName == "" {
		return nil, fmt.Errorf("%s", i18n.T("report.font_load_failed"))
	}

	err = pdf.SetFont(fontName, "", 12)
	if err != nil {
		return nil, fmt.Errorf("failed to set font: %w", err)
	}

	// Add cover page with header
	reportTitle := data.GetReportTitle()
	s.addCoverPage(&pdf, reportTitle, data.DataSourceName, data.UserRequest, fontName)

	// Add insights section FIRST (LLM-generated analysis narrative is the main body)
	if len(data.Insights) > 0 {
		s.addInsightsSection(&pdf, data.Insights, fontName)
	}

	// Add metrics section (supporting data)
	if len(data.Metrics) > 0 {
		s.addMetricsSection(&pdf, data.Metrics, fontName)
	}

	// Add chart images (visual evidence)
	if len(data.ChartImages) > 0 {
		s.addChartsSection(&pdf, data.ChartImages, fontName)
	}

	// Add table section (detailed data)
	if len(data.AllTableData) > 0 {
		for _, namedTable := range data.AllTableData {
			tableData := namedTable.Table
			if len(tableData.Columns) > 0 {
				s.addTableSection(&pdf, &tableData, fontName, namedTable.Name)
			}
		}
	} else if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTableSection(&pdf, data.TableData, fontName)
	}

	// Add footer to all pages
	s.addPageFooters(&pdf, fontName)

	// Get PDF bytes
	var buf bytes.Buffer
	_, err = pdf.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// exportAnalysisResultToPDF exports analysis results (insights only) with optimized layout
func (s *GopdfService) exportAnalysisResultToPDF(data DashboardData) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	// Add first page
	pdf.AddPage()

	// Load font
	fontPaths := []struct {
		name string
		path string
	}{
		{"msyh", "C:\\Windows\\Fonts\\msyh.ttc"},
		{"msyhbd", "C:\\Windows\\Fonts\\msyhbd.ttc"},
		{"simsun", "C:\\Windows\\Fonts\\simsun.ttc"},
		{"simhei", "C:\\Windows\\Fonts\\simhei.ttf"},
		{"arialuni", "C:\\Windows\\Fonts\\ARIALUNI.TTF"},
		{"arial", "C:\\Windows\\Fonts\\arial.ttf"},
	}

	var fontName string
	var err error

	for _, font := range fontPaths {
		err = pdf.AddTTFFont(font.name, font.path)
		if err == nil {
			fontName = font.name
			break
		}
	}

	if fontName == "" {
		return nil, fmt.Errorf("无法加载中文字体")
	}

	err = pdf.SetFont(fontName, "", 12)
	if err != nil {
		return nil, fmt.Errorf("failed to set font: %w", err)
	}

	// 添加简洁的页眉
	s.addAnalysisHeader(&pdf, fontName, data.DataSourceName, data.UserRequest, data.GetReportTitle())

	// 添加分析内容
	s.addAnalysisContent(&pdf, data.Insights, fontName)

	// 添加数据表格（使用原始表格数据直接渲染，不经过 LLM）
	if len(data.AllTableData) > 0 {
		for _, namedTable := range data.AllTableData {
			tableData := namedTable.Table
			if len(tableData.Columns) > 0 {
				s.addTableSection(&pdf, &tableData, fontName, namedTable.Name)
			}
		}
	} else if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTableSection(&pdf, data.TableData, fontName)
	}

	// 添加页脚
	s.addPageFooters(&pdf, fontName)

	// Get PDF bytes
	var buf bytes.Buffer
	_, err = pdf.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// addAnalysisHeader adds a simple header for analysis result export
func (s *GopdfService) addAnalysisHeader(pdf *gopdf.GoPdf, fontName string, dataSourceName string, userRequest string, reportTitle string) {
	// 顶部装饰条 - 清新的青绿色
	pdf.SetFillColor(16, 185, 129) // emerald-500
	pdf.RectFromUpperLeftWithStyle(0, 0, pdfPageWidth, 16, "F")

	// 标题 - 深青色
	pdf.SetFont(fontName, "B", 20)
	pdf.SetTextColor(6, 95, 70) // emerald-800
	title := reportTitle
	if title == "" {
		title = i18n.T("report.data_analysis_report")
		if dataSourceName != "" {
			title = dataSourceName + i18n.T("report.data_analysis_report")
		}
	}
	titleWidth, _ := pdf.MeasureTextWidth(title)
	pdf.SetX((pdfPageWidth - titleWidth) / 2)
	pdf.SetY(40)
	pdf.Cell(nil, title)

	nextY := 68.0

	// 数据源名称
	if dataSourceName != "" {
		pdf.SetFont(fontName, "", pdfFontBody)
		pdf.SetTextColor(71, 85, 105)
		dsText := i18n.T("report.data_source_label") + ": " + dataSourceName
		dsWidth, _ := pdf.MeasureTextWidth(dsText)
		pdf.SetX((pdfPageWidth - dsWidth) / 2)
		pdf.SetY(nextY)
		pdf.Cell(nil, dsText)
		nextY += 18
	}

	// 分析请求 - 完整显示
	if userRequest != "" {
		pdf.SetFont(fontName, "", pdfFontBody)
		pdf.SetTextColor(71, 85, 105)

		labelText := i18n.T("report.analysis_request_label") + ":"
		labelWidth, _ := pdf.MeasureTextWidth(labelText)
		pdf.SetX((pdfPageWidth - labelWidth) / 2)
		pdf.SetY(nextY)
		pdf.Cell(nil, labelText)
		nextY += 18

		pdf.SetFont(fontName, "", pdfFontBody)
		pdf.SetTextColor(51, 65, 85)
		maxLineLen := 55
		if !s.containsChinese(userRequest) {
			maxLineLen = 80
		}
		wrappedLines := s.wrapText(userRequest, maxLineLen)
		for _, line := range wrappedLines {
			lineWidth, _ := pdf.MeasureTextWidth(line)
			pdf.SetX((pdfPageWidth - lineWidth) / 2)
			pdf.SetY(nextY)
			pdf.Cell(nil, line)
			nextY += pdfLineHeightBody
		}
		nextY += 8
	}

	// 生成时间
	pdf.SetFont(fontName, "", pdfFontSmall)
	pdf.SetTextColor(148, 163, 184)
	timestamp := s.formatTimestamp(time.Now())
	timeText := i18n.T("report.generated_time_label") + ": " + timestamp
	timeWidth, _ := pdf.MeasureTextWidth(timeText)
	pdf.SetX((pdfPageWidth - timeWidth) / 2)
	pdf.SetY(nextY)
	pdf.Cell(nil, timeText)

	// 分隔线 - 清新的青绿色
	pdf.SetStrokeColor(167, 243, 208) // emerald-200
	pdf.Line(pdfMarginLeft, nextY+20, pdfPageWidth-pdfMarginRight, nextY+20)

	pdf.SetY(nextY + 40)
}

// addAnalysisContent adds the analysis content with proper formatting
func (s *GopdfService) addAnalysisContent(pdf *gopdf.GoPdf, insights []string, fontName string) {
	// Delegate to addInsightsSection which has the same logic
	s.addInsightsSection(pdf, insights, fontName)
}

// addCoverPage adds a professional cover page
func (s *GopdfService) addCoverPage(pdf *gopdf.GoPdf, title string, dataSourceName string, userRequest string, fontName string) {
	// 顶部装饰条 - 清新的青绿色渐变效果
	pdf.SetFillColor(16, 185, 129) // emerald-500
	pdf.RectFromUpperLeftWithStyle(0, 0, pdfPageWidth, 24, "F")

	// 主标题 - 居中显示，使用深青色
	pdf.SetFont(fontName, "B", pdfFontTitle)
	pdf.SetTextColor(6, 95, 70) // emerald-800
	titleWidth, _ := pdf.MeasureTextWidth(title)
	pdf.SetX((pdfPageWidth - titleWidth) / 2)
	pdf.SetY(160)
	pdf.Cell(nil, title)

	nextY := 200.0

	// 数据源名称 - 标注在标题下方
	if dataSourceName != "" {
		pdf.SetFont(fontName, "", pdfFontHeading2)
		pdf.SetTextColor(71, 85, 105) // slate-600
		dsText := i18n.T("report.data_source_label") + ": " + dataSourceName
		dsWidth, _ := pdf.MeasureTextWidth(dsText)
		pdf.SetX((pdfPageWidth - dsWidth) / 2)
		pdf.SetY(nextY)
		pdf.Cell(nil, dsText)
		nextY += 35
	}

	// 用户请求 - 完整显示，支持多行换行
	if userRequest != "" {
		pdf.SetFont(fontName, "", pdfFontBody)
		pdf.SetTextColor(100, 116, 139) // slate-500

		// 先显示标签
		labelText := i18n.T("report.analysis_request_label") + ":"
		labelWidth, _ := pdf.MeasureTextWidth(labelText)
		pdf.SetX((pdfPageWidth - labelWidth) / 2)
		pdf.SetY(nextY)
		pdf.Cell(nil, labelText)
		nextY += 22

		// 完整显示用户请求，支持自动换行
		pdf.SetFont(fontName, "", pdfFontBody)
		pdf.SetTextColor(51, 65, 85)
		maxLineLen := 55
		if !s.containsChinese(userRequest) {
			maxLineLen = 80
		}
		wrappedLines := s.wrapText(userRequest, maxLineLen)
		for _, line := range wrappedLines {
			lineWidth, _ := pdf.MeasureTextWidth(line)
			pdf.SetX((pdfPageWidth - lineWidth) / 2)
			pdf.SetY(nextY)
			pdf.Cell(nil, line)
			nextY += pdfLineHeightBody
		}
		nextY += 20
	}

	// 生成时间
	pdf.SetFont(fontName, "", pdfFontSmall)
	pdf.SetTextColor(148, 163, 184) // slate-400
	timestamp := s.formatTimestamp(time.Now())
	timeText := i18n.T("report.generated_time_label") + ": " + timestamp
	timeWidth, _ := pdf.MeasureTextWidth(timeText)
	pdf.SetX((pdfPageWidth - timeWidth) / 2)
	pdf.SetY(nextY)
	pdf.Cell(nil, timeText)

	// 分隔线 - 清新的青绿色
	pdf.SetStrokeColor(167, 243, 208) // emerald-200
	pdf.Line(pdfMarginLeft, nextY+40, pdfPageWidth-pdfMarginRight, nextY+40)

	pdf.SetY(nextY + 80)
}

// addSectionTitle adds a styled section title
func (s *GopdfService) addSectionTitle(pdf *gopdf.GoPdf, title string, fontName string) float64 {
	y := pdf.GetY()
	y = s.checkPageBreak(pdf, y, 60)

	// 青绿色左边框装饰
	pdf.SetFillColor(16, 185, 129) // emerald-500
	pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, 8, 24, "F")

	// 标题文字 - 深青色
	pdf.SetFont(fontName, "B", pdfFontHeading1)
	pdf.SetTextColor(6, 95, 70) // emerald-800
	pdf.SetX(pdfMarginLeft + 20)
	pdf.SetY(y + 4)
	pdf.Cell(nil, title)

	return y + 40
}

// addMetricsSection adds metrics in a card-style grid layout
func (s *GopdfService) addMetricsSection(pdf *gopdf.GoPdf, metrics []MetricData, fontName string) {
	y := s.addSectionTitle(pdf, "关键指标", fontName)

	// 计算卡片布局 - 每行3个
	cols := 3
	if len(metrics) <= 2 {
		cols = 2
	}

	cardWidth := (pdfContentWidth - float64(cols-1)*12) / float64(cols)
	cardHeight := 70.0
	spacing := 12.0

	for i, metric := range metrics {
		row := i / cols
		col := i % cols

		x := pdfMarginLeft + float64(col)*(cardWidth+spacing)
		cardY := y + float64(row)*(cardHeight+spacing)

		// 检查分页
		cardY = s.checkPageBreak(pdf, cardY, cardHeight+spacing)
		if cardY < y {
			y = cardY
		}

		// 卡片背景 - 清新的浅青色
		pdf.SetFillColor(236, 253, 245) // emerald-50
		pdf.RectFromUpperLeftWithStyle(x, cardY, cardWidth, cardHeight, "F")

		// 卡片边框 - 青绿色
		pdf.SetStrokeColor(167, 243, 208) // emerald-200
		pdf.RectFromUpperLeftWithStyle(x, cardY, cardWidth, cardHeight, "D")

		// 指标标题
		pdf.SetFont(fontName, "", pdfFontSmall)
		pdf.SetTextColor(100, 116, 139) // slate-500
		pdf.SetX(x + 12)
		pdf.SetY(cardY + 12)
		pdf.Cell(nil, metric.Title)

		// 指标值 - 大字体突出，深青色
		pdf.SetFont(fontName, "B", pdfFontHeading1)
		pdf.SetTextColor(6, 95, 70) // emerald-800
		pdf.SetX(x + 12)
		pdf.SetY(cardY + 35)
		pdf.Cell(nil, metric.Value)

		// 变化值 - 带颜色
		if metric.Change != "" {
			pdf.SetFont(fontName, "", pdfFontSmall)
			if strings.HasPrefix(metric.Change, "+") || strings.Contains(metric.Change, "增") || strings.Contains(metric.Change, "升") {
				pdf.SetTextColor(5, 150, 105) // emerald-600 更清新的绿色
			} else if strings.HasPrefix(metric.Change, "-") || strings.Contains(metric.Change, "减") || strings.Contains(metric.Change, "降") {
				pdf.SetTextColor(239, 68, 68) // red-500 更柔和的红色
			} else {
				pdf.SetTextColor(100, 116, 139) // slate-500
			}
			// 计算值的宽度，将变化值放在右侧
			pdf.SetX(x + cardWidth - 80)
			pdf.SetY(cardY + 35)
			pdf.Cell(nil, metric.Change)
		}
	}

	// 计算总行数
	totalRows := (len(metrics) + cols - 1) / cols
	pdf.SetY(y + float64(totalRows)*(cardHeight+spacing) + 20)
}

// checkPageBreak checks if we need a new page and adds one if necessary
// Returns the current Y position after potential page break
func (s *GopdfService) checkPageBreak(pdf *gopdf.GoPdf, y float64, requiredSpace float64) float64 {
	// 只有当剩余空间真的不够时才分页
	if y+requiredSpace > pdfPageHeight-pdfMarginBottom {
		pdf.AddPage()
		return pdfMarginTop
	}
	return y
}

// parseMarkdownLine parses a single line and returns formatting info
type lineFormat struct {
	text      string
	isBold    bool
	isHeading int // 0=normal, 1=h1, 2=h2, 3=h3, 4=h4
	isList    bool
	indent    int
}

func (s *GopdfService) parseMarkdownLine(line string) lineFormat {
	result := lineFormat{text: line}
	
	// Check for headings (order matters - check longer prefixes first)
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
	} else if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && trimmed[1] == '.' {
		result.isList = true
		result.text = trimmed // Keep numbered list as is
		result.indent = indent / 2
	}
	
	// Handle inline bold (**text** or __text__)
	// Detect if line contains bold markers — set isBold if it starts with **
	if strings.Contains(result.text, "**") {
		result.isBold = true
	}
	result.text = s.stripMarkdownBold(result.text)
	
	return result
}
// parseMarkdownTable converts markdown table lines into [][]string for renderInlineTable
func (s *GopdfService) parseMarkdownTable(lines []string) [][]string {
	if len(lines) < 2 {
		return nil
	}

	var result [][]string

	for idx, line := range lines {
		// Skip separator line (|---|---|)
		if idx == 1 && (strings.Contains(line, "---") || strings.Contains(line, ":-")) {
			continue
		}
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "|")
		parts := strings.Split(line, "|")
		row := make([]string, 0, len(parts))
		for _, p := range parts {
			row = append(row, strings.TrimSpace(p))
		}
		if len(row) > 0 {
			result = append(result, row)
		}
	}

	return result
}

// stripMarkdownBold removes ** and __ markers but keeps the text
func (s *GopdfService) stripMarkdownBold(text string) string {
	// Remove **bold** markers
	for strings.Contains(text, "**") {
		start := strings.Index(text, "**")
		end := strings.Index(text[start+2:], "**")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+2:start+2+end] + text[start+2+end+2:]
	}
	// Remove __bold__ markers
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

// markdownBoldToHTML converts **bold** markers to <b>bold</b> HTML tags
// and escapes basic HTML entities in the rest of the text.
func markdownBoldToHTML(text string) string {
	var sb strings.Builder
	remaining := text
	for {
		start := strings.Index(remaining, "**")
		if start == -1 {
			sb.WriteString(escapeHTMLBasic(remaining))
			break
		}
		end := strings.Index(remaining[start+2:], "**")
		if end == -1 {
			sb.WriteString(escapeHTMLBasic(remaining))
			break
		}
		sb.WriteString(escapeHTMLBasic(remaining[:start]))
		sb.WriteString("<b>")
		sb.WriteString(escapeHTMLBasic(remaining[start+2 : start+2+end]))
		sb.WriteString("</b>")
		remaining = remaining[start+2+end+2:]
	}
	return sb.String()
}

// escapeHTMLBasic escapes &, <, > for safe HTML embedding
func escapeHTMLBasic(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// containsMarkdownBold checks if text contains **bold** markers
func containsMarkdownBold(text string) bool {
	start := strings.Index(text, "**")
	if start == -1 {
		return false
	}
	end := strings.Index(text[start+2:], "**")
	return end >= 0
}

func (s *GopdfService) addInsightsSection(pdf *gopdf.GoPdf, insights []string, fontName string) {
	y := pdf.GetY()
	// 不再添加"智能洞察"标题，因为 LLM 生成的内容已经包含了结构化的章节标题
	pdf.SetY(y)

	for _, insight := range insights {
		// Pre-process: extract and render json:table blocks
		processedInsight, tables := s.extractJsonTables(insight)

		// Split insight into lines (handle multi-line content)
		lines := strings.Split(processedInsight, "\n")

		inCodeBlock := false

		i := 0
		for i < len(lines) {
			line := lines[i]
			trimmedLine := strings.TrimSpace(line)

			// Check for code block markers (skip them)
			if strings.HasPrefix(trimmedLine, "```") {
				inCodeBlock = !inCodeBlock
				i++
				continue
			}

			// Skip content inside code blocks (already processed json:table)
			if inCodeBlock {
				i++
				continue
			}

			// 检测 markdown 表格（连续的 | 开头行）
			if strings.HasPrefix(trimmedLine, "|") && strings.Contains(trimmedLine, "|") {
				tableLines := []string{trimmedLine}
				j := i + 1
				for j < len(lines) {
					nextTrimmed := strings.TrimSpace(lines[j])
					if strings.HasPrefix(nextTrimmed, "|") && strings.Contains(nextTrimmed, "|") {
						tableLines = append(tableLines, nextTrimmed)
						j++
					} else {
						break
					}
				}
				if len(tableLines) >= 2 {
					mdTable := s.parseMarkdownTable(tableLines)
					if len(mdTable) > 0 {
						y = s.renderInlineTable(pdf, mdTable, fontName, y)
					}
					i = j
					continue
				}
			}

			// 检测 key=value 结构化文本行，转为表格渲染
			if strings.Contains(trimmedLine, "=") {
				consumed, kvTable := s.parseKeyValueLines(lines, i)
				if consumed > 0 && len(kvTable) > 0 {
					y = s.renderInlineTable(pdf, kvTable, fontName, y)
					i += consumed
					continue
				}
			}

			// Skip empty lines but add some spacing
			if trimmedLine == "" {
				y += 8
				i++
				continue
			}

			// Parse markdown formatting
			format := s.parseMarkdownLine(line)

			// Set font based on formatting
			fontSize := pdfFontBody
			lineHeight := pdfLineHeightBody
			leftMargin := pdfMarginLeft

			if format.isHeading == 1 {
				fontSize = 18.0
				lineHeight = pdfLineHeightHeading + 4
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(6, 78, 59) // emerald-900 深青色
			} else if format.isHeading == 2 {
				fontSize = pdfFontHeading1
				lineHeight = pdfLineHeightHeading + 2
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(6, 95, 70) // emerald-800
			} else if format.isHeading == 3 {
				fontSize = pdfFontHeading3
				lineHeight = pdfLineHeightHeading
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(4, 120, 87) // emerald-700
			} else if format.isHeading == 4 {
				fontSize = pdfFontBody
				lineHeight = pdfLineHeightBody
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(5, 150, 105) // emerald-600
			} else if format.isBold {
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(51, 65, 85)
			} else {
				pdf.SetFont(fontName, "", fontSize)
				pdf.SetTextColor(51, 65, 85)
			}

			// Add indent for list items
			if format.isList {
				leftMargin += float64(format.indent) * 20
			}

			// Check page break before rendering
			y = s.checkPageBreak(pdf, y, lineHeight)

			// Use InsertHTMLBox for non-heading lines with inline **bold** markers
			if format.isHeading == 0 && containsMarkdownBold(line) {
				htmlContent := markdownBoldToHTML(format.text)
				availableWidth := pdfContentWidth - (leftMargin - pdfMarginLeft)
				// Use a large box height; InsertHTMLBox returns actual Y after rendering
				newY, err := pdf.InsertHTMLBox(leftMargin, y, availableWidth, 2000, htmlContent, gopdf.HTMLBoxOption{
					DefaultFontFamily: fontName,
					DefaultFontSize:   fontSize,
					DefaultColor:      [3]uint8{51, 65, 85},
					BoldFontFamily:    fontName,
					LineSpacing:       lineHeight - fontSize*1.2,
				})
				if err == nil {
					y = newY
				}
			} else {
				// Render text with word wrapping - 使用像素宽度精确换行
				text := format.text
				availableWidth := pdfContentWidth - (leftMargin - pdfMarginLeft)
				wrappedLines := s.wrapTextByWidth(pdf, text, availableWidth)

				for _, wrappedLine := range wrappedLines {
					y = s.checkPageBreak(pdf, y, lineHeight)
					// 一级和二级标题居中对齐
					if format.isHeading == 1 || format.isHeading == 2 {
						textWidth, _ := pdf.MeasureTextWidth(wrappedLine)
						pdf.SetX((pdfPageWidth - textWidth) / 2)
					} else {
						pdf.SetX(leftMargin)
					}
					pdf.SetY(y)
					pdf.Cell(nil, wrappedLine)
					y += lineHeight
				}
			}

			// Add extra spacing after headings
			if format.isHeading > 0 {
				y += 8
			}
			i++
		}

		// Render extracted tables
		for _, table := range tables {
			y = s.renderInlineTable(pdf, table, fontName, y)
		}

		// Add spacing between insights
		y += 12
	}

	pdf.SetY(y + 10)
}

// addInsights is kept for backward compatibility
func (s *GopdfService) addInsights(pdf *gopdf.GoPdf, insights []string, fontName string) {
	s.addInsightsSection(pdf, insights, fontName)
}

// extractJsonTables extracts json:table code blocks and standalone JSON arrays from text
// Returns cleaned text and table data
func (s *GopdfService) extractJsonTables(text string) (string, [][][]string) {
	var tables [][][]string
	result := text
	
	// Method 1: Find all ```json:table ... ``` blocks
	for {
		startMarker := "```json:table"
		endMarker := "```"
		
		startIdx := strings.Index(result, startMarker)
		if startIdx == -1 {
			break
		}
		
		// Find the end of the code block
		afterStart := result[startIdx+len(startMarker):]
		endIdx := strings.Index(afterStart, endMarker)
		if endIdx == -1 {
			break
		}
		
		// Extract JSON content
		jsonContent := strings.TrimSpace(afterStart[:endIdx])
		
		// Parse JSON array
		tableData := s.parseJsonTable(jsonContent)
		if len(tableData) > 0 {
			tables = append(tables, tableData)
		}
		
		// Remove the code block from result (replace with placeholder marker)
		result = result[:startIdx] + result[startIdx+len(startMarker)+endIdx+len(endMarker):]
	}
	
	// Method 2: Find standalone JSON arrays like [ ["col1", "col2"], ["val1", "val2"] ]
	// These are 2D arrays that look like table data
	result = s.extractStandaloneJsonArrays(result, &tables)
	
	return result, tables
}

// extractStandaloneJsonArrays finds and extracts standalone JSON 2D arrays from text
func (s *GopdfService) extractStandaloneJsonArrays(text string, tables *[][][]string) string {
	result := text
	
	// Look for patterns like [ [ ... ], [ ... ] ] that span multiple elements
	// We need to find balanced brackets
	for {
		// Find potential start of a 2D array
		startIdx := s.findJson2DArrayStart(result)
		if startIdx == -1 {
			break
		}
		
		// Find the matching end bracket
		endIdx := s.findMatchingBracket(result, startIdx)
		if endIdx == -1 {
			break
		}
		
		// Extract the JSON content
		jsonContent := result[startIdx : endIdx+1]

		// Skip if the extracted content is too large (likely not a real JSON table)
		// or if it spans too many lines (likely regular text with brackets)
		if len(jsonContent) > 10000 || strings.Count(jsonContent, "\n") > 100 {
			// Skip this occurrence
			result = result[:startIdx] + result[startIdx+1:]
			continue
		}

		// Validate it's actually valid JSON before trying to parse
		jsonTrimmed := strings.TrimSpace(jsonContent)
		if !strings.HasPrefix(jsonTrimmed, "[[") {
			result = result[:startIdx] + result[startIdx+1:]
			continue
		}

		// Validate it's a 2D array (at least 2 rows with same column count)
		tableData := s.parseJsonTable(jsonContent)
		if len(tableData) >= 2 && len(tableData[0]) >= 2 {
			// Check if all rows have similar column count (allow some variance)
			headerCols := len(tableData[0])
			isValidTable := true
			for _, row := range tableData[1:] {
				if len(row) < headerCols-1 || len(row) > headerCols+1 {
					isValidTable = false
					break
				}
			}
			
			if isValidTable {
				*tables = append(*tables, tableData)
				// Remove the JSON array from result, replace with a marker
				result = result[:startIdx] + "[表格数据已提取]" + result[endIdx+1:]
				continue
			}
		}
		
		// If not a valid table, skip this occurrence and continue searching
		// Replace the opening bracket to avoid finding it again (prevents infinite loop)
		result = result[:startIdx] + result[startIdx+1:]
	}
	
	return result
}

// parseKeyValueLines detects consecutive lines with repeated key=value patterns
// and converts them into table data. Returns the number of lines consumed and the table.
// Example input lines:
//   "<50字符：中性比例=35.8%，占比=0.7%，平均情感得分=2.85"
//   "50-100字符：中性比例=28.5%，占比=1.5%，平均情感得分=3.42"
// Each line has a label prefix (before ：or :) and key=value pairs separated by ，or ,
func (s *GopdfService) parseKeyValueLines(lines []string, startIdx int) (int, [][]string) {
	if startIdx >= len(lines) {
		return 0, nil
	}

	// Try to parse the first line as key=value format
	firstKeys, _, _ := s.extractKeyValuePairs(strings.TrimSpace(lines[startIdx]))
	if len(firstKeys) < 2 {
		return 0, nil
	}

	// Collect consecutive lines with the same key pattern
	type parsedLine struct {
		label  string
		values []string
	}
	var parsed []parsedLine

	for i := startIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			break
		}
		keys, values, label := s.extractKeyValuePairs(trimmed)
		if len(keys) < 2 {
			break
		}
		// Check keys match the first line
		if len(keys) != len(firstKeys) {
			break
		}
		match := true
		for k := range keys {
			if keys[k] != firstKeys[k] {
				match = false
				break
			}
		}
		if !match {
			break
		}
		parsed = append(parsed, parsedLine{label: label, values: values})
	}

	if len(parsed) < 2 {
		return 0, nil
	}

	// Build table: header = [label_column, key1, key2, ...]
	// Determine label column name
	labelCol := ""
	if parsed[0].label != "" {
		labelCol = i18n.T("report.category_label")
	}

	var table [][]string
	header := make([]string, 0, len(firstKeys)+1)
	if labelCol != "" {
		header = append(header, labelCol)
	}
	header = append(header, firstKeys...)
	table = append(table, header)

	for _, p := range parsed {
		row := make([]string, 0, len(p.values)+1)
		if labelCol != "" {
			row = append(row, p.label)
		}
		row = append(row, p.values...)
		table = append(table, row)
	}

	return len(parsed), table
}

// extractKeyValuePairs parses a line like "label：key1=val1，key2=val2" or "key1=val1, key2=val2"
// Returns keys, values, and the optional label prefix.
func (s *GopdfService) extractKeyValuePairs(line string) ([]string, []string, string) {
	label := ""
	content := line

	// Strip leading list markers (-, *, numbered)
	content = strings.TrimLeft(content, " \t")
	if strings.HasPrefix(content, "- ") || strings.HasPrefix(content, "* ") {
		content = content[2:]
	} else if len(content) > 2 && content[0] >= '0' && content[0] <= '9' && (content[1] == '.' || (len(content) > 2 && content[1] >= '0' && content[1] <= '9' && content[2] == '.')) {
		idx := strings.Index(content, ".")
		if idx > 0 && idx < 4 {
			content = strings.TrimLeft(content[idx+1:], " ")
		}
	}

	// Check for label prefix (text before ： or : followed by key=value)
	for _, sep := range []string{"：", ": "} {
		idx := strings.Index(content, sep)
		if idx > 0 && idx < 60 {
			afterSep := content[idx+len(sep):]
			// Verify what follows looks like key=value
			if strings.Contains(afterSep, "=") {
				label = strings.TrimSpace(content[:idx])
				content = afterSep
				break
			}
		}
	}

	// Split by Chinese comma or regular comma
	var parts []string
	// Replace Chinese commas with regular commas for uniform splitting
	normalized := strings.ReplaceAll(content, "，", ",")
	parts = strings.Split(normalized, ",")

	var keys, values []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx <= 0 {
			// Not a key=value pair, abort
			return nil, nil, ""
		}
		keys = append(keys, strings.TrimSpace(part[:eqIdx]))
		values = append(values, strings.TrimSpace(part[eqIdx+1:]))
	}

	return keys, values, label
}

// findJson2DArrayStart finds the start of a potential 2D JSON array
func (s *GopdfService) findJson2DArrayStart(text string) int {
	// Look for pattern: [ [ which indicates start of 2D array
	// Allow whitespace between brackets
	runes := []rune(text)
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] == '[' {
			// Check if next non-whitespace is also [
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == ' ' || runes[j] == '\n' || runes[j] == '\r' || runes[j] == '\t' {
					continue
				}
				if runes[j] == '[' {
					return i
				}
				break
			}
		}
	}
	return -1
}

// findMatchingBracket finds the matching closing bracket for an opening bracket
func (s *GopdfService) findMatchingBracket(text string, startIdx int) int {
	runes := []rune(text)
	if startIdx >= len(runes) || runes[startIdx] != '[' {
		return -1
	}
	
	depth := 0
	inString := false
	escapeNext := false
	
	for i := startIdx; i < len(runes); i++ {
		ch := runes[i]
		
		if escapeNext {
			escapeNext = false
			continue
		}
		
		if ch == '\\' && inString {
			escapeNext = true
			continue
		}
		
		if ch == '"' {
			inString = !inString
			continue
		}
		
		if inString {
			continue
		}
		
		if ch == '[' {
			depth++
		} else if ch == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	
	return -1
}

// parseJsonTable parses a JSON array into table data
func (s *GopdfService) parseJsonTable(jsonContent string) [][]string {
	var result [][]string
	
	// Simple JSON array parser for [[...], [...], ...] format
	jsonContent = strings.TrimSpace(jsonContent)
	if !strings.HasPrefix(jsonContent, "[") || !strings.HasSuffix(jsonContent, "]") {
		return result
	}
	
	// Remove outer brackets
	jsonContent = strings.TrimPrefix(jsonContent, "[")
	jsonContent = strings.TrimSuffix(jsonContent, "]")
	jsonContent = strings.TrimSpace(jsonContent)
	
	// Split by rows (each row is [...])
	depth := 0
	rowStart := -1
	
	for i, ch := range jsonContent {
		if ch == '[' {
			if depth == 0 {
				rowStart = i
			}
			depth++
		} else if ch == ']' {
			depth--
			if depth == 0 && rowStart >= 0 {
				rowContent := jsonContent[rowStart+1 : i]
				row := s.parseJsonRow(rowContent)
				if len(row) > 0 {
					result = append(result, row)
				}
				rowStart = -1
			}
		}
	}
	
	return result
}

// parseJsonRow parses a single row of JSON array
func (s *GopdfService) parseJsonRow(rowContent string) []string {
	var result []string
	
	// Split by comma, handling quoted strings
	inQuote := false
	quoteChar := rune(0)
	cellStart := 0
	
	runes := []rune(rowContent)
	for i, ch := range runes {
		if !inQuote && (ch == '"' || ch == '\'') {
			inQuote = true
			quoteChar = ch
		} else if inQuote && ch == quoteChar {
			inQuote = false
		} else if !inQuote && ch == ',' {
			cell := strings.TrimSpace(string(runes[cellStart:i]))
			cell = strings.Trim(cell, "\"'")
			result = append(result, cell)
			cellStart = i + 1
		}
	}
	
	// Add last cell
	if cellStart < len(runes) {
		cell := strings.TrimSpace(string(runes[cellStart:]))
		cell = strings.Trim(cell, "\"'")
		result = append(result, cell)
	}
	
	return result
}

// renderInlineTable renders a table extracted from json:table block
func (s *GopdfService) renderInlineTable(pdf *gopdf.GoPdf, tableData [][]string, fontName string, startY float64) float64 {
	if len(tableData) == 0 {
		return startY
	}

	y := startY
	y = s.checkPageBreak(pdf, y, 80)

	// Determine number of columns from first row (header)
	numCols := len(tableData[0])
	if numCols == 0 {
		return y
	}

	// Limit columns
	maxCols := 8
	if numCols > maxCols {
		numCols = maxCols
	}

	// Calculate column width
	colWidth := pdfContentWidth / float64(numCols)
	cellPadding := 4.0
	cellTextWidth := colWidth - cellPadding*2
	cellLineHeight := 12.0 // line height within table cells

	// Helper: compute row height based on wrapped cell content
	computeRowHeight := func(row []string, fontSize float64) (float64, [][]string) {
		pdf.SetFont(fontName, "", fontSize)
		maxLines := 1
		wrappedCells := make([][]string, numCols)
		for i := 0; i < numCols; i++ {
			cellText := ""
			if i < len(row) {
				cellText = s.stripMarkdownBold(row[i])
			}
			wrapped := s.wrapTextByWidth(pdf, cellText, cellTextWidth)
			if len(wrapped) == 0 {
				wrapped = []string{""}
			}
			wrappedCells[i] = wrapped
			if len(wrapped) > maxLines {
				maxLines = len(wrapped)
			}
		}
		height := float64(maxLines)*cellLineHeight + cellPadding*2
		if height < 20.0 {
			height = 20.0
		}
		return height, wrappedCells
	}

	// Helper: draw header row
	drawInlineHeader := func(atY float64) float64 {
		pdf.SetFont(fontName, "B", pdfFontTableHead)
		headerHeight, wrappedCells := computeRowHeight(tableData[0], pdfFontTableHead)
		pdf.SetFont(fontName, "B", pdfFontTableHead)

		// Draw all cell backgrounds first
		pdf.SetFillColor(16, 185, 129) // emerald-500
		pdf.SetStrokeColor(167, 243, 208) // emerald-200
		x := pdfMarginLeft
		for i := 0; i < numCols; i++ {
			pdf.RectFromUpperLeftWithStyle(x, atY, colWidth, headerHeight, "FD")
			x += colWidth
		}

		// Then draw all text on top
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont(fontName, "B", pdfFontTableHead)
		x = pdfMarginLeft
		for i := 0; i < numCols; i++ {
			for li, line := range wrappedCells[i] {
				pdf.SetX(x + cellPadding)
				pdf.SetY(atY + cellPadding + float64(li)*cellLineHeight)
				pdf.Cell(nil, line)
			}
			x += colWidth
		}
		return atY + headerHeight
	}

	// Draw header
	y = drawInlineHeader(y)

	// Render data rows
	for rowIdx := 1; rowIdx < len(tableData); rowIdx++ {
		row := tableData[rowIdx]

		// Compute row height with wrapping
		pdf.SetFont(fontName, "", pdfFontTableCell)
		rowHeight, wrappedCells := computeRowHeight(row, pdfFontTableCell)

		// Check for page break
		y = s.checkPageBreak(pdf, y, rowHeight)

		// Re-draw header on new page
		if y < pdfMarginTop+30 {
			y = drawInlineHeader(y)
		}

		// Alternating row colors
		if rowIdx%2 == 0 {
			pdf.SetFillColor(236, 253, 245) // emerald-50
		} else {
			pdf.SetFillColor(209, 250, 229) // emerald-100
		}
		pdf.SetStrokeColor(167, 243, 208) // emerald-200

		// Draw all cell backgrounds first
		x := pdfMarginLeft
		for i := 0; i < numCols; i++ {
			pdf.RectFromUpperLeftWithStyle(x, y, colWidth, rowHeight, "FD")
			x += colWidth
		}

		// Then draw all text on top — use InsertHTMLBox for cells with **bold**
		pdf.SetFont(fontName, "", pdfFontTableCell)
		pdf.SetTextColor(51, 65, 85)
		x = pdfMarginLeft
		for i := 0; i < numCols; i++ {
			rawText := ""
			if i < len(row) {
				rawText = row[i]
			}
			if containsMarkdownBold(rawText) {
				htmlContent := markdownBoldToHTML(rawText)
				pdf.InsertHTMLBox(x+cellPadding, y+cellPadding, cellTextWidth, rowHeight-cellPadding*2, htmlContent, gopdf.HTMLBoxOption{
					DefaultFontFamily: fontName,
					DefaultFontSize:   pdfFontTableCell,
					DefaultColor:      [3]uint8{51, 65, 85},
					BoldFontFamily:    fontName,
				})
			} else {
				for li, line := range wrappedCells[i] {
					pdf.SetFont(fontName, "", pdfFontTableCell)
					pdf.SetTextColor(51, 65, 85)
					pdf.SetX(x + cellPadding)
					pdf.SetY(y + cellPadding + float64(li)*cellLineHeight)
					pdf.Cell(nil, line)
				}
			}
			x += colWidth
		}
		y += rowHeight
	}

	return y + 16
}

// containsChinese checks if text contains Chinese characters
func (s *GopdfService) containsChinese(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// wrapText wraps text to fit within maxLen rune characters per line (fallback, no PDF context)
func (s *GopdfService) wrapText(text string, maxLen int) []string {
	if len(text) == 0 {
		return []string{}
	}
	
	var lines []string
	runes := []rune(text)
	
	for len(runes) > 0 {
		if len(runes) <= maxLen {
			lines = append(lines, string(runes))
			break
		}
		
		// Find a good break point
		breakPoint := maxLen
		
		// Try to break at space or punctuation
		for i := maxLen; i > maxLen/2; i-- {
			if runes[i] == ' ' || runes[i] == '，' || runes[i] == '。' || runes[i] == '、' || runes[i] == '；' || runes[i] == '：' {
				breakPoint = i + 1
				break
			}
		}
		
		lines = append(lines, string(runes[:breakPoint]))
		runes = runes[breakPoint:]
		
		// Trim leading spaces
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}
	
	return lines
}

// wrapTextByWidth wraps text using actual pixel-width measurement from the PDF engine.
// This produces accurate line breaks that respect the available content width.
// Falls back to rune-based wrapText if measurement fails.
func (s *GopdfService) wrapTextByWidth(pdf *gopdf.GoPdf, text string, maxWidth float64) []string {
	if len(text) == 0 {
		return []string{}
	}

	// Quick check: if the whole text fits, return as-is
	w, err := pdf.MeasureTextWidth(text)
	if err != nil {
		// Fallback to rune-based wrapping
		maxLen := 50
		if !s.containsChinese(text) {
			maxLen = 80
		}
		return s.wrapText(text, maxLen)
	}
	if w <= maxWidth {
		return []string{text}
	}

	var lines []string
	runes := []rune(text)

	for len(runes) > 0 {
		// Check if remaining text fits
		remaining := string(runes)
		w, err := pdf.MeasureTextWidth(remaining)
		if err != nil || w <= maxWidth {
			lines = append(lines, remaining)
			break
		}

		// Binary search for the maximum number of runes that fit within maxWidth
		lo, hi := 1, len(runes)
		for lo < hi {
			mid := (lo + hi + 1) / 2
			candidate := string(runes[:mid])
			cw, e := pdf.MeasureTextWidth(candidate)
			if e != nil {
				hi = mid - 1
				continue
			}
			if cw <= maxWidth {
				lo = mid
			} else {
				hi = mid - 1
			}
		}

		breakPoint := lo

		// Try to find a better break point at a word/punctuation boundary
		// Search backwards from breakPoint for a natural break character
		bestBreak := -1
		searchStart := breakPoint
		if searchStart > len(runes) {
			searchStart = len(runes)
		}
		minBreak := breakPoint / 2
		if minBreak < 1 {
			minBreak = 1
		}
		for i := searchStart; i >= minBreak; i-- {
			ch := runes[i-1]
			if ch == ' ' || ch == '，' || ch == '。' || ch == '、' || ch == '；' || ch == '：' ||
				ch == '）' || ch == '」' || ch == '"' || ch == '\'' {
				bestBreak = i
				break
			}
		}
		if bestBreak > 0 {
			breakPoint = bestBreak
		}

		if breakPoint <= 0 {
			breakPoint = 1 // Ensure progress
		}

		lines = append(lines, string(runes[:breakPoint]))
		runes = runes[breakPoint:]

		// Trim leading spaces
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}

	return lines
}

// truncateTextToWidth truncates text to fit within a given pixel width using font measurement.
// If pdf is nil or measurement fails, falls back to rune-based truncation with runeLimit.
func (s *GopdfService) truncateTextToWidth(pdf *gopdf.GoPdf, text string, maxWidth float64, runeLimit int) string {
	runes := []rune(text)

	// Try pixel-based measurement first
	if pdf != nil {
		textWidth, err := pdf.MeasureTextWidth(text)
		if err == nil && textWidth <= maxWidth {
			return text
		}
		if err == nil {
			// Binary search for the longest fitting substring
			lo, hi := 0, len(runes)
			for lo < hi {
				mid := (lo + hi + 1) / 2
				candidate := string(runes[:mid]) + ".."
				w, e := pdf.MeasureTextWidth(candidate)
				if e != nil {
					break
				}
				if w <= maxWidth {
					lo = mid
				} else {
					hi = mid - 1
				}
			}
			if lo > 0 && lo < len(runes) {
				return string(runes[:lo]) + ".."
			}
			// If lo == len(runes), the full text fits (edge case with ".." overhead)
			if lo >= len(runes) {
				return text
			}
		}
	}

	// Fallback: rune-based truncation
	if len(runes) > runeLimit {
		return string(runes[:runeLimit-2]) + ".."
	}
	return text
}

// addChartsSection adds chart images with professional styling
func (s *GopdfService) addChartsSection(pdf *gopdf.GoPdf, chartImages []string, fontName string) {
	if len(chartImages) == 0 {
		return
	}

	for i, chartImage := range chartImages {
		chartHeight := 350.0 // 图表高度 (points)
		// 图表总需空间：子标题(20) + 容器(chartHeight+12) + 间距(30)
		totalChartSpace := 20 + chartHeight + 12 + 30

		// 第一个图表添加章节标题
		if i == 0 {
			y := s.addSectionTitle(pdf, "数据可视化", fontName)
			// 章节标题后检查剩余空间是否够放图表
			y = s.checkPageBreak(pdf, y, totalChartSpace)
			pdf.SetY(y)
		} else {
			y := pdf.GetY()
			y = s.checkPageBreak(pdf, y, totalChartSpace)
			pdf.SetY(y)
		}

		y := pdf.GetY()

		// 图表子标题
		pdf.SetFont(fontName, "", pdfFontSmall)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetX(pdfMarginLeft)
		pdf.SetY(y)
		pdf.Cell(nil, i18n.T("report.chart_label", i+1, len(chartImages)))
		y += 20

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

		// Create image holder from bytes
		imgHolder, err := gopdf.ImageHolderByBytes(imgBytes)
		if err != nil {
			continue
		}

		// 图表容器背景 - 清新的浅青色
		pdf.SetFillColor(236, 253, 245) // emerald-50
		pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, pdfContentWidth, chartHeight+12, "F")

		// 图表边框 - 青绿色
		pdf.SetStrokeColor(167, 243, 208) // emerald-200
		pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, pdfContentWidth, chartHeight+12, "D")

		// Add image to PDF - 居中显示
		imgWidth := pdfContentWidth - 24
		imgHeight := chartHeight - 12
		imgX := pdfMarginLeft + 12
		imgY := y + 6

		pdf.ImageByHolder(imgHolder, imgX, imgY, &gopdf.Rect{W: imgWidth, H: imgHeight})

		y += chartHeight + 30
		pdf.SetY(y)
	}
}

// addCharts is kept for backward compatibility
func (s *GopdfService) addCharts(pdf *gopdf.GoPdf, chartImages []string, fontName string) {
	s.addChartsSection(pdf, chartImages, fontName)
}

// addTableSection adds data table with professional card-style layout
func (s *GopdfService) addTableSection(pdf *gopdf.GoPdf, tableData *TableData, fontName string, tableName ...string) {
	title := "数据表格"
	if len(tableName) > 0 && tableName[0] != "" {
		title = s.stripMarkdownBold(tableName[0])
	}
	y := s.addSectionTitle(pdf, title, fontName)

	// 限制列数以保证可读性
	maxCols := 8
	cols := tableData.Columns
	if len(cols) > maxCols {
		cols = cols[:maxCols]
	}

	numCols := len(cols)
	colWidth := pdfContentWidth / float64(numCols)
	cellPadding := 4.0
	cellTextWidth := colWidth - cellPadding*2
	cellLineHeight := 12.0

	// Helper: compute row height with text wrapping
	computeStructuredRowHeight := func(rowData []interface{}, fontSize float64) (float64, [][]string) {
		pdf.SetFont(fontName, "", fontSize)
		maxLines := 1
		wrappedCells := make([][]string, numCols)
		for i := 0; i < numCols && i < len(rowData); i++ {
			cellText := s.stripMarkdownBold(fmt.Sprintf("%v", rowData[i]))
			wrapped := s.wrapTextByWidth(pdf, cellText, cellTextWidth)
			if len(wrapped) == 0 {
				wrapped = []string{""}
			}
			wrappedCells[i] = wrapped
			if len(wrapped) > maxLines {
				maxLines = len(wrapped)
			}
		}
		// Fill remaining columns with empty
		for i := len(rowData); i < numCols; i++ {
			wrappedCells[i] = []string{""}
		}
		height := float64(maxLines)*cellLineHeight + cellPadding*2
		if height < 20.0 {
			height = 20.0
		}
		return height, wrappedCells
	}

	// 绘制表头的辅助函数
	drawHeader := func(atY float64) float64 {
		pdf.SetFont(fontName, "B", pdfFontTableHead)
		// Compute header height with wrapping
		maxLines := 1
		wrappedHeaders := make([][]string, numCols)
		for i, col := range cols {
			wrapped := s.wrapTextByWidth(pdf, col.Title, cellTextWidth)
			if len(wrapped) == 0 {
				wrapped = []string{""}
			}
			wrappedHeaders[i] = wrapped
			if len(wrapped) > maxLines {
				maxLines = len(wrapped)
			}
		}
		headerHeight := float64(maxLines)*cellLineHeight + cellPadding*2
		if headerHeight < 24.0 {
			headerHeight = 24.0
		}

		pdf.SetFillColor(16, 185, 129) // emerald-500
		pdf.SetTextColor(255, 255, 255)
		pdf.SetStrokeColor(167, 243, 208) // emerald-200

		// Draw all cell backgrounds first
		hx := pdfMarginLeft
		for i := 0; i < numCols; i++ {
			pdf.RectFromUpperLeftWithStyle(hx, atY, colWidth, headerHeight, "FD")
			hx += colWidth
		}

		// Then draw all text on top
		pdf.SetFont(fontName, "B", pdfFontTableHead)
		pdf.SetTextColor(255, 255, 255)
		hx = pdfMarginLeft
		for i := 0; i < numCols; i++ {
			for li, line := range wrappedHeaders[i] {
				pdf.SetX(hx + cellPadding)
				pdf.SetY(atY + cellPadding + float64(li)*cellLineHeight)
				pdf.Cell(nil, line)
			}
			hx += colWidth
		}
		return atY + headerHeight
	}

	// 绘制表头
	y = s.checkPageBreak(pdf, y, 80)
	y = drawHeader(y)

	// 绘制数据行
	totalRows := len(tableData.Data)

	for rowIdx, rowData := range tableData.Data {
		// Compute row height
		pdf.SetFont(fontName, "", pdfFontTableCell)
		rowHeight, wrappedCells := computeStructuredRowHeight(rowData, pdfFontTableCell)

		// 检查分页
		if y+rowHeight > pdfPageHeight-pdfMarginBottom {
			pdf.AddPage()
			y = pdfMarginTop
			y = drawHeader(y)
		}

		// 交替行背景色
		if rowIdx%2 == 0 {
			pdf.SetFillColor(236, 253, 245) // emerald-50
		} else {
			pdf.SetFillColor(209, 250, 229) // emerald-100
		}
		pdf.SetStrokeColor(167, 243, 208) // emerald-200

		// Draw all cell backgrounds first
		x := pdfMarginLeft
		for i := 0; i < numCols; i++ {
			pdf.RectFromUpperLeftWithStyle(x, y, colWidth, rowHeight, "FD")
			x += colWidth
		}

		// Then draw all text on top — use InsertHTMLBox for cells with **bold**
		pdf.SetFont(fontName, "", pdfFontTableCell)
		pdf.SetTextColor(51, 65, 85)
		x = pdfMarginLeft
		for i := 0; i < numCols; i++ {
			rawText := ""
			if i < len(rowData) {
				rawText = fmt.Sprintf("%v", rowData[i])
			}
			if containsMarkdownBold(rawText) {
				htmlContent := markdownBoldToHTML(rawText)
				pdf.InsertHTMLBox(x+cellPadding, y+cellPadding, cellTextWidth, rowHeight-cellPadding*2, htmlContent, gopdf.HTMLBoxOption{
					DefaultFontFamily: fontName,
					DefaultFontSize:   pdfFontTableCell,
					DefaultColor:      [3]uint8{51, 65, 85},
					BoldFontFamily:    fontName,
				})
			} else {
				for li, line := range wrappedCells[i] {
					pdf.SetFont(fontName, "", pdfFontTableCell)
					pdf.SetTextColor(51, 65, 85)
					pdf.SetX(x + cellPadding)
					pdf.SetY(y + cellPadding + float64(li)*cellLineHeight)
					pdf.Cell(nil, line)
				}
			}
			x += colWidth
		}

		y += rowHeight
	}

	// 添加表格信息
	y += 10
	pdf.SetFont(fontName, "", pdfFontFooter)
	pdf.SetTextColor(148, 163, 184)
	pdf.SetX(pdfMarginLeft)
	pdf.SetY(y)

	infoText := fmt.Sprintf("共 %d 行数据", totalRows)
	if len(tableData.Columns) > maxCols {
		infoText += fmt.Sprintf("（显示前 %d 列）", maxCols)
	}
	pdf.Cell(nil, infoText)

	pdf.SetY(y + 24)
}

// addTable is kept for backward compatibility
func (s *GopdfService) addTable(pdf *gopdf.GoPdf, tableData *TableData, fontName string) {
	s.addTableSection(pdf, tableData, fontName)
}

// addPageFooters adds footer to all pages (called at the end)
func (s *GopdfService) addPageFooters(pdf *gopdf.GoPdf, fontName string) {
	// gopdf doesn't have a built-in way to iterate pages after creation
	// The footer is added to the current page only
	// For multi-page documents, we rely on the footer being added during content generation
	// This function adds a final footer to the last page if needed
	
	pdf.SetFont(fontName, "", pdfFontFooter)
	pdf.SetTextColor(148, 163, 184)
	
	footerText := i18n.T("export.generated_by")
	footerWidth, _ := pdf.MeasureTextWidth(footerText)
	
	pdf.SetX((pdfPageWidth - footerWidth) / 2)
	pdf.SetY(pdfPageHeight - pdfMarginBottom + 15)
	pdf.Cell(nil, footerText)
}

// addFooter is kept for backward compatibility
func (s *GopdfService) addFooter(pdf *gopdf.GoPdf, fontName string) {
	s.addPageFooters(pdf, fontName)
}


// formatTimestamp formats a timestamp according to the current language
func (s *GopdfService) formatTimestamp(t time.Time) string {
	lang := i18n.GetLanguage()
	if lang == i18n.Chinese {
		return t.Format("2006年01月02日 15:04:05")
	}
	// English format
	return t.Format("January 02, 2006 15:04:05")
}
