package export

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/signintech/gopdf"
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
		(data.TableData == nil || len(data.TableData.Columns) == 0)

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
		return nil, fmt.Errorf("无法加载中文字体。请确保系统安装了以下字体之一：Arial Unicode MS、微软雅黑、黑体、宋体")
	}

	err = pdf.SetFont(fontName, "", 12)
	if err != nil {
		return nil, fmt.Errorf("failed to set font: %w", err)
	}

	// Add cover page with header
	s.addCoverPage(&pdf, "智能仪表盘报告", data.UserRequest, fontName)

	// Add metrics section
	if len(data.Metrics) > 0 {
		s.addMetricsSection(&pdf, data.Metrics, fontName)
	}

	// Add chart images
	if len(data.ChartImages) > 0 {
		s.addChartsSection(&pdf, data.ChartImages, fontName)
	}

	// Add insights section
	if len(data.Insights) > 0 {
		s.addInsightsSection(&pdf, data.Insights, fontName)
	}

	// Add table section
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
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
	s.addAnalysisHeader(&pdf, fontName)

	// 添加分析内容
	s.addAnalysisContent(&pdf, data.Insights, fontName)

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
func (s *GopdfService) addAnalysisHeader(pdf *gopdf.GoPdf, fontName string) {
	// 顶部装饰条
	pdf.SetFillColor(59, 130, 246)
	pdf.RectFromUpperLeftWithStyle(0, 0, pdfPageWidth, 16, "F")

	// 标题
	pdf.SetFont(fontName, "B", 20)
	pdf.SetTextColor(30, 64, 175)
	title := "智能分析报告"
	titleWidth, _ := pdf.MeasureTextWidth(title)
	pdf.SetX((pdfPageWidth - titleWidth) / 2)
	pdf.SetY(50)
	pdf.Cell(nil, title)

	// 生成时间
	pdf.SetFont(fontName, "", pdfFontSmall)
	pdf.SetTextColor(148, 163, 184)
	timestamp := time.Now().Format("2006年01月02日 15:04:05")
	timeText := "生成时间: " + timestamp
	timeWidth, _ := pdf.MeasureTextWidth(timeText)
	pdf.SetX((pdfPageWidth - timeWidth) / 2)
	pdf.SetY(80)
	pdf.Cell(nil, timeText)

	// 分隔线
	pdf.SetStrokeColor(226, 232, 240)
	pdf.Line(pdfMarginLeft, 100, pdfPageWidth-pdfMarginRight, 100)

	pdf.SetY(120)
}

// addAnalysisContent adds the analysis content with proper formatting
func (s *GopdfService) addAnalysisContent(pdf *gopdf.GoPdf, insights []string, fontName string) {
	y := pdf.GetY()

	// 合并所有内容
	allContent := strings.Join(insights, "\n")
	lines := strings.Split(allContent, "\n")

	inCodeBlock := false

	for _, line := range lines {
		// 检查代码块标记
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// 跳过代码块内容
		if inCodeBlock {
			continue
		}

		// 空行添加间距
		if strings.TrimSpace(line) == "" {
			y += 10
			continue
		}

		// 解析 Markdown 格式
		format := s.parseMarkdownLine(line)

		// 根据格式设置字体
		fontSize := pdfFontBody
		lineHeight := pdfLineHeightBody
		leftMargin := pdfMarginLeft

		if format.isHeading == 1 {
			fontSize = 18.0
			lineHeight = 28.0
			pdf.SetFont(fontName, "B", fontSize)
			pdf.SetTextColor(30, 64, 175)
			y += 8 // 标题前额外间距
		} else if format.isHeading == 2 {
			fontSize = 16.0
			lineHeight = 24.0
			pdf.SetFont(fontName, "B", fontSize)
			pdf.SetTextColor(59, 130, 246)
			y += 6
		} else if format.isHeading == 3 {
			fontSize = 14.0
			lineHeight = 22.0
			pdf.SetFont(fontName, "B", fontSize)
			pdf.SetTextColor(71, 85, 105)
			y += 4
		} else if format.isHeading == 4 {
			fontSize = 12.0
			lineHeight = 20.0
			pdf.SetFont(fontName, "B", fontSize)
			pdf.SetTextColor(71, 85, 105)
		} else if format.isBold {
			pdf.SetFont(fontName, "B", fontSize)
			pdf.SetTextColor(51, 65, 85)
		} else {
			pdf.SetFont(fontName, "", fontSize)
			pdf.SetTextColor(51, 65, 85)
		}

		// 列表项缩进
		if format.isList {
			leftMargin += float64(format.indent) * 24
		}

		// 检查分页
		y = s.checkPageBreak(pdf, y, lineHeight)

		// 文本换行
		text := format.text
		maxLineLen := 75 - format.indent*4

		if s.containsChinese(text) {
			maxLineLen = 50 - format.indent*3
		}

		wrappedLines := s.wrapText(text, maxLineLen)

		for _, wrappedLine := range wrappedLines {
			y = s.checkPageBreak(pdf, y, lineHeight)
			pdf.SetX(leftMargin)
			pdf.SetY(y)
			pdf.Cell(nil, wrappedLine)
			y += lineHeight
		}

		// 标题后额外间距
		if format.isHeading > 0 {
			y += 6
		}
	}

	pdf.SetY(y + 20)
}

// addCoverPage adds a professional cover page
func (s *GopdfService) addCoverPage(pdf *gopdf.GoPdf, title string, userRequest string, fontName string) {
	// 顶部装饰条
	pdf.SetFillColor(59, 130, 246)
	pdf.RectFromUpperLeftWithStyle(0, 0, pdfPageWidth, 24, "F")

	// 主标题 - 居中显示
	pdf.SetFont(fontName, "B", pdfFontTitle)
	pdf.SetTextColor(30, 64, 175)
	titleWidth, _ := pdf.MeasureTextWidth(title)
	pdf.SetX((pdfPageWidth - titleWidth) / 2)
	pdf.SetY(180)
	pdf.Cell(nil, title)

	// 用户请求 - 作为副标题显示
	if userRequest != "" {
		pdf.SetFont(fontName, "", pdfFontHeading2)
		pdf.SetTextColor(71, 85, 105)

		// 截断过长的请求
		displayRequest := userRequest
		if len(displayRequest) > 60 {
			displayRequest = displayRequest[:57] + "..."
		}
		displayRequest = "「" + displayRequest + "」"

		reqWidth, _ := pdf.MeasureTextWidth(displayRequest)
		pdf.SetX((pdfPageWidth - reqWidth) / 2)
		pdf.SetY(230)
		pdf.Cell(nil, displayRequest)
	}

	// 生成时间
	pdf.SetFont(fontName, "", pdfFontSmall)
	pdf.SetTextColor(148, 163, 184)
	timestamp := time.Now().Format("2006年01月02日 15:04:05")
	timeText := "生成时间: " + timestamp
	timeWidth, _ := pdf.MeasureTextWidth(timeText)
	pdf.SetX((pdfPageWidth - timeWidth) / 2)
	pdf.SetY(290)
	pdf.Cell(nil, timeText)

	// 分隔线
	pdf.SetStrokeColor(226, 232, 240)
	pdf.Line(pdfMarginLeft, 340, pdfPageWidth-pdfMarginRight, 340)

	pdf.SetY(380)
}

// addSectionTitle adds a styled section title
func (s *GopdfService) addSectionTitle(pdf *gopdf.GoPdf, title string, fontName string) float64 {
	y := pdf.GetY()
	y = s.checkPageBreak(pdf, y, 60)

	// 蓝色左边框装饰
	pdf.SetFillColor(59, 130, 246)
	pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, 8, 24, "F")

	// 标题文字
	pdf.SetFont(fontName, "B", pdfFontHeading1)
	pdf.SetTextColor(30, 64, 175)
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

		// 卡片背景
		pdf.SetFillColor(248, 250, 252)
		pdf.RectFromUpperLeftWithStyle(x, cardY, cardWidth, cardHeight, "F")

		// 卡片边框
		pdf.SetStrokeColor(226, 232, 240)
		pdf.RectFromUpperLeftWithStyle(x, cardY, cardWidth, cardHeight, "D")

		// 指标标题
		pdf.SetFont(fontName, "", pdfFontSmall)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetX(x + 12)
		pdf.SetY(cardY + 12)
		pdf.Cell(nil, metric.Title)

		// 指标值 - 大字体突出
		pdf.SetFont(fontName, "B", pdfFontHeading1)
		pdf.SetTextColor(30, 64, 175)
		pdf.SetX(x + 12)
		pdf.SetY(cardY + 35)
		pdf.Cell(nil, metric.Value)

		// 变化值 - 带颜色
		if metric.Change != "" {
			pdf.SetFont(fontName, "", pdfFontSmall)
			if strings.HasPrefix(metric.Change, "+") || strings.Contains(metric.Change, "增") || strings.Contains(metric.Change, "升") {
				pdf.SetTextColor(22, 163, 74) // 绿色
			} else if strings.HasPrefix(metric.Change, "-") || strings.Contains(metric.Change, "减") || strings.Contains(metric.Change, "降") {
				pdf.SetTextColor(220, 38, 38) // 红色
			} else {
				pdf.SetTextColor(100, 116, 139)
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
	result.text = s.stripMarkdownBold(result.text)
	
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

func (s *GopdfService) addInsightsSection(pdf *gopdf.GoPdf, insights []string, fontName string) {
	y := s.addSectionTitle(pdf, "智能洞察", fontName)
	pdf.SetY(y)

	for _, insight := range insights {
		// Pre-process: extract and render json:table blocks
		processedInsight, tables := s.extractJsonTables(insight)

		// Split insight into lines (handle multi-line content)
		lines := strings.Split(processedInsight, "\n")

		inCodeBlock := false

		for _, line := range lines {
			// Check for code block markers (skip them)
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inCodeBlock = !inCodeBlock
				continue
			}

			// Skip content inside code blocks (already processed json:table)
			if inCodeBlock {
				continue
			}

			// Skip empty lines but add some spacing
			if strings.TrimSpace(line) == "" {
				y += 8
				continue
			}

			// Parse markdown formatting
			format := s.parseMarkdownLine(line)

			// Set font based on formatting
			fontSize := pdfFontBody
			lineHeight := pdfLineHeightBody
			leftMargin := pdfMarginLeft

			if format.isHeading == 1 {
				fontSize = pdfFontHeading1
				lineHeight = pdfLineHeightHeading
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(30, 64, 175)
			} else if format.isHeading == 2 {
				fontSize = pdfFontHeading2
				lineHeight = pdfLineHeightHeading
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(59, 130, 246)
			} else if format.isHeading == 3 {
				fontSize = pdfFontHeading3
				lineHeight = pdfLineHeightHeading
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(71, 85, 105)
			} else if format.isHeading == 4 {
				fontSize = pdfFontBody
				lineHeight = pdfLineHeightBody
				pdf.SetFont(fontName, "B", fontSize)
				pdf.SetTextColor(71, 85, 105)
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

			// Render text with word wrapping
			text := format.text
			maxLineLen := 80 - format.indent*4

			// For Chinese text, count characters differently
			if s.containsChinese(text) {
				maxLineLen = 55 - format.indent*3
			}

			// Word wrap the text
			wrappedLines := s.wrapText(text, maxLineLen)

			for _, wrappedLine := range wrappedLines {
				y = s.checkPageBreak(pdf, y, lineHeight)
				pdf.SetX(leftMargin)
				pdf.SetY(y)
				pdf.Cell(nil, wrappedLine)
				y += lineHeight
			}

			// Add extra spacing after headings
			if format.isHeading > 0 {
				y += 8
			}
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
		// Move past this bracket to avoid infinite loop
		if startIdx+1 < len(result) {
			nextStart := s.findJson2DArrayStart(result[startIdx+1:])
			if nextStart == -1 {
				break
			}
			// Adjust for the offset
			continue
		}
		break
	}
	
	return result
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
	maxCols := 6
	if numCols > maxCols {
		numCols = maxCols
	}
	
	// Calculate column width
	colWidth := pdfContentWidth / float64(numCols)
	
	// Render header row
	headerHeight := 24.0
	y = s.checkPageBreak(pdf, y, headerHeight)
	pdf.SetFont(fontName, "B", pdfFontTableHead)
	pdf.SetFillColor(59, 130, 246)
	pdf.SetTextColor(255, 255, 255)
	
	x := pdfMarginLeft
	for i := 0; i < numCols && i < len(tableData[0]); i++ {
		cellValue := tableData[0][i]
		maxLen := int(colWidth / 7)
		if maxLen < 8 {
			maxLen = 8
		}
		if len(cellValue) > maxLen {
			cellValue = cellValue[:maxLen-2] + ".."
		}
		pdf.SetX(x)
		pdf.SetY(y)
		pdf.CellWithOption(&gopdf.Rect{W: colWidth, H: headerHeight}, cellValue, gopdf.CellOption{
			Align:  gopdf.Center | gopdf.Middle,
			Border: gopdf.AllBorders,
		})
		x += colWidth
	}
	y += headerHeight
	
	// Render data rows
	rowHeight := 20.0
	pdf.SetFont(fontName, "", pdfFontTableCell)
	pdf.SetTextColor(51, 65, 85)
	
	for rowIdx := 1; rowIdx < len(tableData); rowIdx++ {
		row := tableData[rowIdx]
		
		// Check for page break
		y = s.checkPageBreak(pdf, y, rowHeight)
		
		// Re-draw header on new page
		if y < pdfMarginTop + 30 {
			pdf.SetFont(fontName, "B", pdfFontTableHead)
			pdf.SetFillColor(59, 130, 246)
			pdf.SetTextColor(255, 255, 255)
			x = pdfMarginLeft
			for i := 0; i < numCols && i < len(tableData[0]); i++ {
				cellValue := tableData[0][i]
				maxLen := int(colWidth / 7)
				if maxLen < 8 {
					maxLen = 8
				}
				if len(cellValue) > maxLen {
					cellValue = cellValue[:maxLen-2] + ".."
				}
				pdf.SetX(x)
				pdf.SetY(y)
				pdf.CellWithOption(&gopdf.Rect{W: colWidth, H: headerHeight}, cellValue, gopdf.CellOption{
					Align:  gopdf.Center | gopdf.Middle,
					Border: gopdf.AllBorders,
				})
				x += colWidth
			}
			y += headerHeight
			pdf.SetFont(fontName, "", pdfFontTableCell)
			pdf.SetTextColor(51, 65, 85)
		}
		
		// Alternating row colors
		if rowIdx%2 == 0 {
			pdf.SetFillColor(248, 250, 252)
		} else {
			pdf.SetFillColor(241, 245, 249)
		}
		
		x = pdfMarginLeft
		for i := 0; i < numCols && i < len(row); i++ {
			cellValue := row[i]
			maxLen := int(colWidth / 7)
			if maxLen < 8 {
				maxLen = 8
			}
			if len(cellValue) > maxLen {
				cellValue = cellValue[:maxLen-2] + ".."
			}
			pdf.SetX(x)
			pdf.SetY(y)
			pdf.CellWithOption(&gopdf.Rect{W: colWidth, H: rowHeight}, cellValue, gopdf.CellOption{
				Align:  gopdf.Left | gopdf.Middle,
				Border: gopdf.AllBorders,
			})
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

// wrapText wraps text to fit within maxLen characters per line
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
		
		// Try to break at space for non-Chinese text
		for i := maxLen; i > maxLen/2; i-- {
			if runes[i] == ' ' || runes[i] == '，' || runes[i] == '。' || runes[i] == '、' {
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

// addChartsSection adds chart images with professional styling
func (s *GopdfService) addChartsSection(pdf *gopdf.GoPdf, chartImages []string, fontName string) {
	if len(chartImages) == 0 {
		return
	}

	for i, chartImage := range chartImages {
		// 每个图表单独一页或检查空间
		y := pdf.GetY()
		chartHeight := 350.0 // 图表高度 (points)

		// 第一个图表添加章节标题
		if i == 0 {
			y = s.addSectionTitle(pdf, "数据可视化", fontName)
		} else {
			// 检查是否需要新页
			y = s.checkPageBreak(pdf, y, chartHeight+60)
		}

		// 图表子标题
		pdf.SetFont(fontName, "", pdfFontSmall)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetX(pdfMarginLeft)
		pdf.SetY(y)
		pdf.Cell(nil, fmt.Sprintf("图表 %d / %d", i+1, len(chartImages)))
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

		// 图表容器背景
		pdf.SetFillColor(250, 251, 252)
		pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, pdfContentWidth, chartHeight+12, "F")

		// 图表边框
		pdf.SetStrokeColor(226, 232, 240)
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
func (s *GopdfService) addTableSection(pdf *gopdf.GoPdf, tableData *TableData, fontName string) {
	y := s.addSectionTitle(pdf, "数据表格", fontName)

	// 限制列数以保证可读性
	maxCols := 6
	cols := tableData.Columns
	if len(cols) > maxCols {
		cols = cols[:maxCols]
	}

	// 计算列宽 - 根据列数动态调整
	colWidth := pdfContentWidth / float64(len(cols))

	// 表头行高和数据行高
	headerHeight := 24.0
	rowHeight := 20.0

	// 每页最大行数
	maxRowsPerPage := 32

	// 绘制表头
	y = s.checkPageBreak(pdf, y, headerHeight+rowHeight*3)

	// 表头背景
	pdf.SetFillColor(59, 130, 246)
	pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, pdfContentWidth, headerHeight, "F")

	// 表头文字
	pdf.SetFont(fontName, "B", pdfFontTableHead)
	pdf.SetTextColor(255, 255, 255)

	x := pdfMarginLeft
	for _, col := range cols {
		colTitle := col.Title
		maxLen := int(colWidth / 7)
		if maxLen < 6 {
			maxLen = 6
		}
		if len(colTitle) > maxLen {
			colTitle = colTitle[:maxLen-2] + ".."
		}
		pdf.SetX(x + 4)
		pdf.SetY(y + 6)
		pdf.CellWithOption(&gopdf.Rect{W: colWidth - 8, H: headerHeight - 12}, colTitle, gopdf.CellOption{
			Align: gopdf.Center | gopdf.Middle,
		})
		x += colWidth
	}

	y += headerHeight
	headerY := y - headerHeight // 记录表头位置用于分页重绘

	// 绘制数据行
	pdf.SetFont(fontName, "", pdfFontTableCell)
	rowCount := 0
	totalRows := len(tableData.Data)

	for rowIdx, rowData := range tableData.Data {
		// 检查分页
		if y+rowHeight > pdfPageHeight-pdfMarginBottom || rowCount >= maxRowsPerPage {
			pdf.AddPage()
			y = pdfMarginTop

			// 重绘表头
			pdf.SetFillColor(59, 130, 246)
			pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, pdfContentWidth, headerHeight, "F")

			pdf.SetFont(fontName, "B", pdfFontTableHead)
			pdf.SetTextColor(255, 255, 255)

			x = pdfMarginLeft
			for _, col := range cols {
				colTitle := col.Title
				maxLen := int(colWidth / 7)
				if maxLen < 6 {
					maxLen = 6
				}
				if len(colTitle) > maxLen {
					colTitle = colTitle[:maxLen-2] + ".."
				}
				pdf.SetX(x + 4)
				pdf.SetY(y + 6)
				pdf.CellWithOption(&gopdf.Rect{W: colWidth - 8, H: headerHeight - 12}, colTitle, gopdf.CellOption{
					Align: gopdf.Center | gopdf.Middle,
				})
				x += colWidth
			}

			y += headerHeight
			headerY = y - headerHeight
			rowCount = 0
			pdf.SetFont(fontName, "", pdfFontTableCell)
		}

		// 交替行背景色
		if rowIdx%2 == 0 {
			pdf.SetFillColor(248, 250, 252)
		} else {
			pdf.SetFillColor(241, 245, 249)
		}
		pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, pdfContentWidth, rowHeight, "F")

		// 绘制单元格边框
		pdf.SetStrokeColor(226, 232, 240)
		pdf.RectFromUpperLeftWithStyle(pdfMarginLeft, y, pdfContentWidth, rowHeight, "D")

		// 绘制数据
		pdf.SetTextColor(51, 65, 85)
		x = pdfMarginLeft
		for i := 0; i < len(cols) && i < len(rowData); i++ {
			cellValue := fmt.Sprintf("%v", rowData[i])
			maxLen := int(colWidth / 7)
			if maxLen < 6 {
				maxLen = 6
			}
			if len(cellValue) > maxLen {
				cellValue = cellValue[:maxLen-2] + ".."
			}
			pdf.SetX(x + 4)
			pdf.SetY(y + 5)
			pdf.CellWithOption(&gopdf.Rect{W: colWidth - 8, H: rowHeight - 10}, cellValue, gopdf.CellOption{
				Align: gopdf.Left | gopdf.Middle,
			})
			x += colWidth
		}

		y += rowHeight
		rowCount++
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

	// 避免未使用变量警告
	_ = headerY
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
	
	footerText := "由 VantageData 智能分析系统生成"
	footerWidth, _ := pdf.MeasureTextWidth(footerText)
	
	pdf.SetX((pdfPageWidth - footerWidth) / 2)
	pdf.SetY(pdfPageHeight - pdfMarginBottom + 15)
	pdf.Cell(nil, footerText)
}

// addFooter is kept for backward compatibility
func (s *GopdfService) addFooter(pdf *gopdf.GoPdf, fontName string) {
	s.addPageFooters(pdf, fontName)
}
