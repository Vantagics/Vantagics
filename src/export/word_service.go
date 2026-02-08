package export

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"

	goword "github.com/VantageDataChat/GoWord"
	"github.com/VantageDataChat/GoWord/style"
	"vantagedata/i18n"
)

// WordExportService handles Word document generation using GoWord (pure Go)
type WordExportService struct{}

// NewWordExportService creates a new Word export service
func NewWordExportService() *WordExportService {
	return &WordExportService{}
}

// ExportDashboardToWord exports dashboard data to Word format
func (s *WordExportService) ExportDashboardToWord(data DashboardData) ([]byte, error) {
	// Title - 使用LLM生成的标题或数据源名称生成标题
	reportTitle := data.GetReportTitle()

	doc := goword.New()
	doc.Properties.Title = reportTitle
	doc.Properties.Creator = "VantageData"
	doc.Properties.Description = i18n.T("export.doc_description")

	sec := doc.AddSection()

	// 内容区域宽度 = A4宽度(11906) - 左右边距(1440*2) = 9026 twips
	contentWidth := 9026

	titlePara := sec.AddTitle(reportTitle, 1)
	titlePara.Style.Alignment = style.AlignCenter

	// 数据源名称
	if data.DataSourceName != "" {
		sec.AddText(i18n.T("export.datasource_label")+data.DataSourceName,
			&style.FontStyle{Size: 11, Color: "475569"},
			&style.ParagraphStyle{Alignment: style.AlignCenter, SpaceAfter: 60})
	}

	// 分析请求
	if data.UserRequest != "" {
		sec.AddText(i18n.T("export.analysis_request")+data.UserRequest,
			&style.FontStyle{Size: 11, Color: "475569"},
			&style.ParagraphStyle{Alignment: style.AlignCenter, SpaceAfter: 60})
	}

	// Timestamp
	sec.AddText(time.Now().Format("2006年01月02日 15:04"),
		&style.FontStyle{Size: 10, Color: "94A3B8"},
		&style.ParagraphStyle{Alignment: style.AlignCenter})

	sec.AddTextBreak(1)

	// Insights (LLM-generated analysis narrative) - 作为报告主体内容优先展示
	if len(data.Insights) > 0 {
		for _, insight := range data.Insights {
			s.renderMarkdownContent(sec, insight, contentWidth)
		}

		sec.AddTextBreak(1)
	}

	// Metrics - 作为支撑数据展示
	if len(data.Metrics) > 0 {
		sec.AddText(i18n.T("export.key_metrics"),
			&style.FontStyle{Bold: true, Size: 14, Color: "059669"}, // emerald-600 清新的青绿色
			nil)

		// Create metrics table
		ts := &style.TableStyle{Width: contentWidth, Alignment: "center"}
		ts.SetAllBorders("single", 4, "A7F3D0") // emerald-200 清新的边框
		tbl := sec.AddTable(ts)

		colWidth := contentWidth / 3

		// Header row - 清新的青绿色
		headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "10B981"}, // emerald-500
		}).AddText(i18n.T("export.metric_column"), &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "10B981"}, // emerald-500
		}).AddText(i18n.T("export.value_column"), &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "10B981"}, // emerald-500
		}).AddText(i18n.T("export.change_column"), &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)

		// Data rows
		for _, metric := range data.Metrics {
			row := tbl.AddRow(0, nil)
			row.AddCell(colWidth, nil).AddText(metric.Title, &style.FontStyle{Size: 10}, nil)
			row.AddCell(colWidth, nil).AddText(metric.Value, &style.FontStyle{Size: 10, Bold: true, Color: "047857"}, nil) // emerald-700

			changeColor := "64748B"
			if strings.HasPrefix(metric.Change, "+") || strings.Contains(metric.Change, "增") {
				changeColor = "059669" // emerald-600 更清新的绿色
			} else if strings.HasPrefix(metric.Change, "-") || strings.Contains(metric.Change, "减") {
				changeColor = "EF4444" // red-500 更柔和的红色
			}
			row.AddCell(colWidth, nil).AddText(metric.Change, &style.FontStyle{Size: 10, Color: changeColor}, nil)
		}

		sec.AddTextBreak(1)
	}

	// Chart images - 数据可视化
	s.addChartImages(sec, data.ChartImages, contentWidth)

	// Table data - render all tables after the report text
	s.renderAllTables(sec, data, contentWidth)

	// Footer
	sec.AddTextBreak(1)
	sec.AddText(i18n.T("export.generated_by"),
		&style.FontStyle{Size: 9, Color: "94A3B8"},
		&style.ParagraphStyle{Alignment: style.AlignCenter})

	// Save to bytes
	data2, err := doc.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to write Word file: %w", err)
	}

	return data2, nil
}

// renderMarkdownContent renders markdown text content into Word paragraphs
func (s *WordExportService) renderMarkdownContent(sec *goword.Section, content string, contentWidth int) {
	// Extract json:table blocks and standalone JSON arrays first
	processedContent, jsonTables := s.extractJsonTables(content)

	lines := strings.Split(processedContent, "\n")

	inCodeBlock := false

	// 检测并渲染 markdown 表格
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 检查代码块标记（跳过）
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			i++
			continue
		}

		// 跳过代码块内容
		if inCodeBlock {
			i++
			continue
		}

		// 检测 markdown 表格（连续的 | 开头行）
		if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "|") {
			tableLines := []string{trimmed}
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
				s.renderMarkdownTable(sec, tableLines, contentWidth)
				i = j
				continue
			}
		}

		if trimmed == "" {
			sec.AddTextBreak(1)
			i++
			continue
		}

		// Parse markdown headings
		if strings.HasPrefix(trimmed, "#### ") {
			sec.AddText(strings.TrimPrefix(trimmed, "#### "),
				&style.FontStyle{Bold: true, Size: 11, Color: "475569"},
				&style.ParagraphStyle{SpaceBefore: 80, SpaceAfter: 40})
		} else if strings.HasPrefix(trimmed, "### ") {
			sec.AddText(strings.TrimPrefix(trimmed, "### "),
				&style.FontStyle{Bold: true, Size: 12, Color: "047857"}, // emerald-700
				&style.ParagraphStyle{SpaceBefore: 120, SpaceAfter: 60})
		} else if strings.HasPrefix(trimmed, "## ") {
			sec.AddText(strings.TrimPrefix(trimmed, "## "),
				&style.FontStyle{Bold: true, Size: 14, Color: "059669"}, // emerald-600
				&style.ParagraphStyle{Alignment: style.AlignCenter, SpaceBefore: 200, SpaceAfter: 100})
		} else if strings.HasPrefix(trimmed, "# ") {
			sec.AddText(strings.TrimPrefix(trimmed, "# "),
				&style.FontStyle{Bold: true, Size: 16, Color: "065F46"}, // emerald-800
				&style.ParagraphStyle{Alignment: style.AlignCenter, SpaceBefore: 240, SpaceAfter: 120})
		} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			bulletText := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			bulletText = stripMarkdownBold(bulletText)
			sec.AddText("• "+bulletText,
				&style.FontStyle{Size: 11, Color: "334155"},
				&style.ParagraphStyle{Indent: 360, Hanging: 360})
		} else if matched, numText := parseNumberedItem(trimmed); matched {
			// 有序列表项：使用悬挂缩进确保换行对齐
			numText = stripMarkdownBold(numText)
			sec.AddText(numText,
				&style.FontStyle{Size: 11, Color: "334155"},
				&style.ParagraphStyle{Indent: 480, Hanging: 480, SpaceAfter: 60})
		} else {
			text := stripMarkdownBold(trimmed)
			sec.AddText(text,
				&style.FontStyle{Size: 11, Color: "334155"},
				&style.ParagraphStyle{Alignment: style.AlignBoth})
		}
		i++
	}

	// Render extracted JSON tables
	for _, table := range jsonTables {
		s.renderJsonTable(sec, table, contentWidth)
	}
}

// renderMarkdownTable renders a markdown table into a Word table
func (s *WordExportService) renderMarkdownTable(sec *goword.Section, lines []string, contentWidth int) {
	// Parse header
	headers := parseMarkdownTableRow(lines[0])
	if len(headers) == 0 {
		return
	}

	// Skip separator line (line with |---|---|)
	dataStartIdx := 1
	if dataStartIdx < len(lines) {
		sep := lines[dataStartIdx]
		if strings.Contains(sep, "---") || strings.Contains(sep, ":-") {
			dataStartIdx = 2
		}
	}

	maxCols := len(headers)
	if maxCols > 8 {
		maxCols = 8
		headers = headers[:maxCols]
	}

	colWidth := contentWidth / maxCols

	ts := &style.TableStyle{Width: contentWidth, Alignment: "center"}
	ts.SetAllBorders("single", 4, "A7F3D0") // emerald-200 清新的边框
	tbl := sec.AddTable(ts)
	tbl.Grid = make([]int, maxCols)
	for k := range tbl.Grid {
		tbl.Grid[k] = colWidth
	}

	// Header row - 清新的青绿色
	headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
	for _, h := range headers {
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "10B981"}, // emerald-500
		}).AddText(strings.TrimSpace(h), &style.FontStyle{Bold: true, Size: 9, Color: "FFFFFF"}, nil)
	}

	// Data rows
	for idx := dataStartIdx; idx < len(lines); idx++ {
		cells := parseMarkdownTableRow(lines[idx])
		row := tbl.AddRow(0, nil)
		for c := 0; c < maxCols; c++ {
			cellVal := ""
			if c < len(cells) {
				cellVal = strings.TrimSpace(cells[c])
			}
			row.AddCell(colWidth, nil).AddText(cellVal, &style.FontStyle{Size: 9}, nil)
		}
	}

	sec.AddTextBreak(1)
}

// parseMarkdownTableRow splits a markdown table row into cells
func parseMarkdownTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

// renderAllTables renders all table data into the Word document
func (s *WordExportService) renderAllTables(sec *goword.Section, data DashboardData, contentWidth int) {
	// Prefer AllTableData (multiple tables) over single TableData
	if len(data.AllTableData) > 0 {
		sec.AddText(i18n.T("export.data_tables"),
			&style.FontStyle{Bold: true, Size: 14, Color: "059669"}, // emerald-600
			nil)

		for _, namedTable := range data.AllTableData {
			tableData := &namedTable.Table
			if len(tableData.Columns) == 0 {
				continue
			}

			// Table name as sub-heading
			if namedTable.Name != "" {
				sec.AddText(namedTable.Name,
					&style.FontStyle{Bold: true, Size: 12, Color: "047857"}, // emerald-700
					&style.ParagraphStyle{SpaceBefore: 120, SpaceAfter: 60})
			}

			s.renderSingleTable(sec, tableData, contentWidth)
			sec.AddTextBreak(1)
		}
	} else if data.TableData != nil && len(data.TableData.Columns) > 0 {
		sec.AddText(i18n.T("export.data_tables"),
			&style.FontStyle{Bold: true, Size: 14, Color: "059669"}, // emerald-600
			nil)
		s.renderSingleTable(sec, data.TableData, contentWidth)
	}
}

// renderSingleTable renders one table into the Word document
func (s *WordExportService) renderSingleTable(sec *goword.Section, tableData *TableData, contentWidth int) {
	maxCols := 8
	cols := tableData.Columns
	if len(cols) > maxCols {
		cols = cols[:maxCols]
	}

	colWidth := contentWidth / len(cols)

	ts := &style.TableStyle{Width: contentWidth, Alignment: "center"}
	ts.SetAllBorders("single", 4, "A7F3D0") // emerald-200 清新的边框
	tbl := sec.AddTable(ts)
	tbl.Grid = make([]int, len(cols))
	for i := range tbl.Grid {
		tbl.Grid[i] = colWidth
	}

	// Header row - 清新的青绿色
	headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
	for _, col := range cols {
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "10B981"}, // emerald-500
		}).AddText(col.Title, &style.FontStyle{Bold: true, Size: 9, Color: "FFFFFF"}, nil)
	}

	// Data rows (limit to 50)
	maxRows := 50
	rows := tableData.Data
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	for _, rowData := range rows {
		row := tbl.AddRow(0, nil)
		for i := 0; i < len(cols) && i < len(rowData); i++ {
			cellValue := fmt.Sprintf("%v", rowData[i])
			if len([]rune(cellValue)) > 50 {
				cellValue = string([]rune(cellValue)[:47]) + "..."
			}
			row.AddCell(colWidth, nil).AddText(cellValue, &style.FontStyle{Size: 9}, nil)
		}
	}

	if len(tableData.Data) > maxRows {
		sec.AddText(i18n.T("export.table_note", maxRows, len(tableData.Data)),
			&style.FontStyle{Size: 9, Color: "94A3B8", Italic: true},
			nil)
	}
}

// parseNumberedItem checks if a line is a numbered list item (e.g. "1. text" or "1、text")
func parseNumberedItem(line string) (bool, string) {
	for i, r := range line {
		if r >= '0' && r <= '9' {
			continue
		}
		if i > 0 {
			rest := line[i:]
			if strings.HasPrefix(rest, ". ") || strings.HasPrefix(rest, "、") || strings.HasPrefix(rest, ") ") {
				return true, line
			}
		}
		break
	}
	return false, ""
}

// stripMarkdownBold removes ** bold markers from text
func stripMarkdownBold(text string) string {
	for strings.Contains(text, "**") {
		start := strings.Index(text, "**")
		end := strings.Index(text[start+2:], "**")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+2:start+2+end] + text[start+2+end+2:]
	}
	return text
}

// addChartImages adds chart images to the Word document
func (s *WordExportService) addChartImages(sec *goword.Section, chartImages []string, contentWidth int) {
	if len(chartImages) == 0 {
		return
	}

	sec.AddText(i18n.T("export.data_visualization"),
		&style.FontStyle{Bold: true, Size: 14, Color: "059669"}, // emerald-600 清新的青绿色
		&style.ParagraphStyle{SpaceBefore: 200, SpaceAfter: 100})

	for i, chartImage := range chartImages {
		// 图表子标题
		sec.AddText(i18n.T("export.chart_number", i+1, len(chartImages)),
			&style.FontStyle{Size: 10, Color: "64748B"},
			&style.ParagraphStyle{SpaceAfter: 60})

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

		// Determine MIME type
		mimeType := "image/png"
		if strings.Contains(chartImage, "image/jpeg") {
			mimeType = "image/jpeg"
		} else if strings.Contains(chartImage, "image/gif") {
			mimeType = "image/gif"
		}

		// GoWord ImageStyle Width/Height are in pixels (96 DPI), converted via PixelToEmu internally.
		// contentWidth is in twips. 1 inch = 1440 twips = 96 pixels, so pixels = twips / 15.
		imgWidthPx := contentWidth * 9 / (10 * 15) // 90% of content width in pixels

		// Read actual image dimensions to preserve aspect ratio
		imgHeightPx := imgWidthPx * 9 / 16 // default 16:9 fallback
		imgCfg, _, decErr := image.DecodeConfig(bytes.NewReader(imgBytes))
		if decErr == nil && imgCfg.Width > 0 && imgCfg.Height > 0 {
			ratio := float64(imgCfg.Height) / float64(imgCfg.Width)
			imgHeightPx = int(float64(imgWidthPx) * ratio)
		}

		sec.AddImageFromBytes(imgBytes, mimeType, &style.ImageStyle{
			Width:     imgWidthPx,
			Height:    imgHeightPx,
			Alignment: "center",
		})

		sec.AddTextBreak(1)
	}
}

// extractJsonTables extracts json:table code blocks and standalone JSON arrays from text
// Returns cleaned text and table data (same logic as PDF service)
func (s *WordExportService) extractJsonTables(text string) (string, [][][]string) {
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

		afterStart := result[startIdx+len(startMarker):]
		endIdx := strings.Index(afterStart, endMarker)
		if endIdx == -1 {
			break
		}

		jsonContent := strings.TrimSpace(afterStart[:endIdx])
		tableData := s.parseJsonTable(jsonContent)
		if len(tableData) > 0 {
			tables = append(tables, tableData)
		}

		result = result[:startIdx] + result[startIdx+len(startMarker)+endIdx+len(endMarker):]
	}

	// Method 2: Find standalone JSON arrays
	result = s.extractStandaloneJsonArrays(result, &tables)

	return result, tables
}

// extractStandaloneJsonArrays finds and extracts standalone JSON 2D arrays from text
func (s *WordExportService) extractStandaloneJsonArrays(text string, tables *[][][]string) string {
	result := text

	for {
		startIdx := s.findJson2DArrayStart(result)
		if startIdx == -1 {
			break
		}

		endIdx := s.findMatchingBracket(result, startIdx)
		if endIdx == -1 {
			break
		}

		jsonContent := result[startIdx : endIdx+1]
		tableData := s.parseJsonTable(jsonContent)
		if len(tableData) >= 2 && len(tableData[0]) >= 2 {
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
				result = result[:startIdx] + i18n.T("export.table_extracted") + result[endIdx+1:]
				continue
			}
		}

		if startIdx+1 < len(result) {
			nextStart := s.findJson2DArrayStart(result[startIdx+1:])
			if nextStart == -1 {
				break
			}
			continue
		}
		break
	}

	return result
}

// findJson2DArrayStart finds the start of a potential 2D JSON array
func (s *WordExportService) findJson2DArrayStart(text string) int {
	runes := []rune(text)
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] == '[' {
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

// findMatchingBracket finds the matching closing bracket
func (s *WordExportService) findMatchingBracket(text string, startIdx int) int {
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
func (s *WordExportService) parseJsonTable(jsonContent string) [][]string {
	var result [][]string

	jsonContent = strings.TrimSpace(jsonContent)
	if !strings.HasPrefix(jsonContent, "[") || !strings.HasSuffix(jsonContent, "]") {
		return result
	}

	jsonContent = strings.TrimPrefix(jsonContent, "[")
	jsonContent = strings.TrimSuffix(jsonContent, "]")
	jsonContent = strings.TrimSpace(jsonContent)

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
func (s *WordExportService) parseJsonRow(rowContent string) []string {
	var result []string

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

	if cellStart < len(runes) {
		cell := strings.TrimSpace(string(runes[cellStart:]))
		cell = strings.Trim(cell, "\"'")
		result = append(result, cell)
	}

	return result
}

// renderJsonTable renders a JSON-extracted table into a Word table
func (s *WordExportService) renderJsonTable(sec *goword.Section, tableData [][]string, contentWidth int) {
	if len(tableData) == 0 {
		return
	}

	numCols := len(tableData[0])
	if numCols == 0 {
		return
	}

	maxCols := 8
	if numCols > maxCols {
		numCols = maxCols
	}

	colWidth := contentWidth / numCols

	ts := &style.TableStyle{Width: contentWidth, Alignment: "center"}
	ts.SetAllBorders("single", 4, "A7F3D0") // emerald-200 清新的边框
	tbl := sec.AddTable(ts)
	tbl.Grid = make([]int, numCols)
	for k := range tbl.Grid {
		tbl.Grid[k] = colWidth
	}

	// Header row (first row of JSON table) - 清新的青绿色
	headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
	for c := 0; c < numCols && c < len(tableData[0]); c++ {
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "10B981"}, // emerald-500
		}).AddText(strings.TrimSpace(tableData[0][c]), &style.FontStyle{Bold: true, Size: 9, Color: "FFFFFF"}, nil)
	}

	// Data rows
	for rowIdx := 1; rowIdx < len(tableData); rowIdx++ {
		row := tbl.AddRow(0, nil)
		for c := 0; c < numCols; c++ {
			cellVal := ""
			if c < len(tableData[rowIdx]) {
				cellVal = strings.TrimSpace(tableData[rowIdx][c])
			}
			row.AddCell(colWidth, nil).AddText(cellVal, &style.FontStyle{Size: 9}, nil)
		}
	}

	sec.AddTextBreak(1)
}
