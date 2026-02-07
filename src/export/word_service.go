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
	doc := goword.New()
	doc.Properties.Title = "智能仪表盘报告"
	doc.Properties.Creator = "VantageData"
	doc.Properties.Description = "由 VantageData 智能分析系统生成"

	sec := doc.AddSection()

	// Title
	sec.AddTitle("智能仪表盘报告", 1)

	// Timestamp
	sec.AddText(time.Now().Format("2006年01月02日 15:04"),
		&style.FontStyle{Size: 10, Color: "94A3B8"},
		&style.ParagraphStyle{Alignment: style.AlignCenter})

	sec.AddTextBreak(1)

	// User request
	if data.UserRequest != "" {
		sec.AddText("用户请求",
			&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
			nil)
		sec.AddText(data.UserRequest,
			&style.FontStyle{Size: 11, Color: "334155"},
			&style.ParagraphStyle{SpaceAfter: 200})
		sec.AddTextBreak(1)
	}

	// Metrics
	if len(data.Metrics) > 0 {
		sec.AddText("关键指标",
			&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
			nil)

		// Create metrics table
		ts := &style.TableStyle{Width: 9000, Alignment: "center"}
		ts.SetAllBorders("single", 4, "D9D9D9")
		tbl := sec.AddTable(ts)

		// Header row
		headerRow := tbl.AddRow(0, &style.RowStyle{IsHeader: true})
		headerRow.AddCell(3000, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
		}).AddText("指标", &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)
		headerRow.AddCell(3000, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
		}).AddText("数值", &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)
		headerRow.AddCell(3000, &style.CellStyle{
			Shading: &style.Shading{Fill: "4472C4"},
		}).AddText("变化", &style.FontStyle{Bold: true, Size: 10, Color: "FFFFFF"}, nil)

		// Data rows
		for _, metric := range data.Metrics {
			row := tbl.AddRow(0, nil)
			row.AddCell(3000, nil).AddText(metric.Title, &style.FontStyle{Size: 10}, nil)
			row.AddCell(3000, nil).AddText(metric.Value, &style.FontStyle{Size: 10, Bold: true, Color: "1E40AF"}, nil)

			changeColor := "64748B"
			if strings.HasPrefix(metric.Change, "+") || strings.Contains(metric.Change, "增") {
				changeColor = "16A34A"
			} else if strings.HasPrefix(metric.Change, "-") || strings.Contains(metric.Change, "减") {
				changeColor = "DC2626"
			}
			row.AddCell(3000, nil).AddText(metric.Change, &style.FontStyle{Size: 10, Color: changeColor}, nil)
		}

		sec.AddTextBreak(1)
	}

	// Insights
	if len(data.Insights) > 0 {
		sec.AddText("智能洞察",
			&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
			nil)

		for _, insight := range data.Insights {
			lines := strings.Split(insight, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					sec.AddTextBreak(1)
					continue
				}

				// Parse markdown headings
				if strings.HasPrefix(trimmed, "## ") {
					sec.AddText(strings.TrimPrefix(trimmed, "## "),
						&style.FontStyle{Bold: true, Size: 13, Color: "3B82F6"},
						nil)
				} else if strings.HasPrefix(trimmed, "# ") {
					sec.AddText(strings.TrimPrefix(trimmed, "# "),
						&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
						nil)
				} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
					bulletText := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
					sec.AddText("• "+bulletText,
						&style.FontStyle{Size: 11, Color: "334155"},
						&style.ParagraphStyle{Indent: 360})
				} else {
					// Strip bold markers
					text := trimmed
					for strings.Contains(text, "**") {
						start := strings.Index(text, "**")
						end := strings.Index(text[start+2:], "**")
						if end == -1 {
							break
						}
						text = text[:start] + text[start+2:start+2+end] + text[start+2+end+2:]
					}
					sec.AddText(text,
						&style.FontStyle{Size: 11, Color: "334155"},
						nil)
				}
			}
		}

		sec.AddTextBreak(1)
	}

	// Table data
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		sec.AddText("数据表格",
			&style.FontStyle{Bold: true, Size: 14, Color: "1E40AF"},
			nil)

		maxCols := 6
		cols := data.TableData.Columns
		if len(cols) > maxCols {
			cols = cols[:maxCols]
		}

		colWidthTotal := 9000
		colWidth := colWidthTotal / len(cols)

		ts := &style.TableStyle{Width: colWidthTotal, Alignment: "center"}
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
		rows := data.TableData.Data
		if len(rows) > maxRows {
			rows = rows[:maxRows]
		}

		for _, rowData := range rows {
			row := tbl.AddRow(0, nil)
			for i := 0; i < len(cols) && i < len(rowData); i++ {
				cellValue := fmt.Sprintf("%v", rowData[i])
				if len([]rune(cellValue)) > 40 {
					cellValue = string([]rune(cellValue)[:37]) + "..."
				}
				row.AddCell(colWidth, nil).AddText(cellValue, &style.FontStyle{Size: 9}, nil)
			}
		}

		if len(data.TableData.Data) > maxRows {
			sec.AddText(fmt.Sprintf("注：仅显示前 %d 行数据，共 %d 行", maxRows, len(data.TableData.Data)),
				&style.FontStyle{Size: 9, Color: "94A3B8", Italic: true},
				nil)
		}
	}

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
