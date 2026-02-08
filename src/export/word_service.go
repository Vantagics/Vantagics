package export

import (
	"fmt"
	"strings"
	"time"

	goword "github.com/VantageDataChat/GoWord"
	"github.com/VantageDataChat/GoWord/style"
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
	doc.Properties.Description = "由 VantageData 智能分析系统生成"

	sec := doc.AddSection()

	// 内容区域宽度 = A4宽度(11906) - 左右边距(1440*2) = 9026 twips
	contentWidth := 9026

	sec.AddTitle(reportTitle, 1)

	// 数据源名称
	if data.DataSourceName != "" {
		sec.AddText("数据源: "+data.DataSourceName,
			&style.FontStyle{Size: 11, Color: "475569"},
			&style.ParagraphStyle{Alignment: style.AlignCenter, SpaceAfter: 60})
	}

	// 分析请求
	if data.UserRequest != "" {
		sec.AddText("分析请求: "+data.UserRequest,
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
		sec.AddText("关键指标",
			&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
			nil)

		// Create metrics table
		ts := &style.TableStyle{Width: contentWidth, Alignment: "center"}
		ts.SetAllBorders("single", 4, "D9D9D9")
		tbl := sec.AddTable(ts)

		colWidth := contentWidth / 3

		// Header row
		headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
		}).AddText("指标", &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
		}).AddText("数值", &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
		}).AddText("变化", &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)

		// Data rows
		for _, metric := range data.Metrics {
			row := tbl.AddRow(0, nil)
			row.AddCell(colWidth, nil).AddText(metric.Title, &style.FontStyle{Size: 10}, nil)
			row.AddCell(colWidth, nil).AddText(metric.Value, &style.FontStyle{Size: 10, Bold: true, Color: "1E40AF"}, nil)

			changeColor := "64748B"
			if strings.HasPrefix(metric.Change, "+") || strings.Contains(metric.Change, "增") {
				changeColor = "16A34A"
			} else if strings.HasPrefix(metric.Change, "-") || strings.Contains(metric.Change, "减") {
				changeColor = "DC2626"
			}
			row.AddCell(colWidth, nil).AddText(metric.Change, &style.FontStyle{Size: 10, Color: changeColor}, nil)
		}

		sec.AddTextBreak(1)
	}

	// Table data - render all tables (AllTableData takes priority over single TableData)
	s.renderAllTables(sec, data, contentWidth)

	// Footer
	sec.AddTextBreak(1)
	sec.AddText("由 VantageData 智能分析系统生成",
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
	lines := strings.Split(content, "\n")

	// 检测并渲染 markdown 表格
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

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
		if strings.HasPrefix(trimmed, "### ") {
			sec.AddText(strings.TrimPrefix(trimmed, "### "),
				&style.FontStyle{Bold: true, Size: 12, Color: "3B82F6"},
				&style.ParagraphStyle{SpaceBefore: 120, SpaceAfter: 60})
		} else if strings.HasPrefix(trimmed, "## ") {
			sec.AddText(strings.TrimPrefix(trimmed, "## "),
				&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
				&style.ParagraphStyle{Alignment: style.AlignCenter, SpaceBefore: 200, SpaceAfter: 100})
		} else if strings.HasPrefix(trimmed, "# ") {
			sec.AddText(strings.TrimPrefix(trimmed, "# "),
				&style.FontStyle{Bold: true, Size: 16, Color: "1E3A5F"},
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
	ts.SetAllBorders("single", 4, "D9D9D9")
	tbl := sec.AddTable(ts)
	tbl.Grid = make([]int, maxCols)
	for k := range tbl.Grid {
		tbl.Grid[k] = colWidth
	}

	// Header row
	headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
	for _, h := range headers {
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
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
		sec.AddText("数据表格",
			&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
			nil)

		for _, namedTable := range data.AllTableData {
			tableData := &namedTable.Table
			if len(tableData.Columns) == 0 {
				continue
			}

			// Table name as sub-heading
			if namedTable.Name != "" {
				sec.AddText(namedTable.Name,
					&style.FontStyle{Bold: true, Size: 12, Color: "3B82F6"},
					&style.ParagraphStyle{SpaceBefore: 120, SpaceAfter: 60})
			}

			s.renderSingleTable(sec, tableData, contentWidth)
			sec.AddTextBreak(1)
		}
	} else if data.TableData != nil && len(data.TableData.Columns) > 0 {
		sec.AddText("数据表格",
			&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
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
	ts.SetAllBorders("single", 4, "D9D9D9")
	tbl := sec.AddTable(ts)
	tbl.Grid = make([]int, len(cols))
	for i := range tbl.Grid {
		tbl.Grid[i] = colWidth
	}

	// Header row
	headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
	for _, col := range cols {
		headerRow.AddCell(colWidth, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
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
		sec.AddText(fmt.Sprintf("注：仅显示前 %d 行数据，共 %d 行", maxRows, len(tableData.Data)),
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
