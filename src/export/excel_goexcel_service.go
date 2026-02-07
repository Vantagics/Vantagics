package export

import (
	"bytes"
	"fmt"

	gospreadsheet "github.com/VantageDataChat/GoExcel"
)

// GoExcelExportService handles Excel file generation using GoExcel (pure Go)
type GoExcelExportService struct{}

// NewGoExcelExportService creates a new GoExcel export service
func NewGoExcelExportService() *GoExcelExportService {
	return &GoExcelExportService{}
}

// ExportTableToExcel exports table data to Excel format using GoExcel
func (s *GoExcelExportService) ExportTableToExcel(tableData *TableData, sheetName string) ([]byte, error) {
	if tableData == nil || len(tableData.Columns) == 0 {
		return nil, fmt.Errorf("no table data to export")
	}

	wb := gospreadsheet.New()
	ws := wb.GetActiveSheet()

	if sheetName == "" {
		sheetName = "数据表"
	}
	ws.SetTitle(sheetName)

	// Header style
	headerStyle := gospreadsheet.NewStyle().
		SetFont(&gospreadsheet.Font{
			Bold:  true,
			Size:  11,
			Color: "FFFFFF",
			Name:  "Microsoft YaHei",
		}).
		SetFill(&gospreadsheet.Fill{
			Type:  "solid",
			Color: "4472C4",
		}).
		SetAlignment(&gospreadsheet.Alignment{
			Horizontal: gospreadsheet.AlignCenter,
			Vertical:   gospreadsheet.AlignMiddle,
		}).
		SetBorders(&gospreadsheet.Borders{
			Left:   gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
			Top:    gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
			Bottom: gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
			Right:  gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
		})

	// Data style
	dataStyle := gospreadsheet.NewStyle().
		SetFont(&gospreadsheet.Font{
			Size: 10,
			Name: "Microsoft YaHei",
		}).
		SetAlignment(&gospreadsheet.Alignment{
			Horizontal: gospreadsheet.AlignLeft,
			Vertical:   gospreadsheet.AlignMiddle,
			WrapText:   true,
		}).
		SetBorders(&gospreadsheet.Borders{
			Left:   gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
			Top:    gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
			Bottom: gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
			Right:  gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
		})

	// Write headers
	for i, col := range tableData.Columns {
		cellName, _ := gospreadsheet.CellName(0, i)
		ws.SetCellValue(cellName, col.Title)
		ws.SetCellStyle(cellName, headerStyle)

		// Set column width
		runeLen := len([]rune(col.Title))
		width := float64(runeLen) * 2.5
		if width < 12 {
			width = 12
		}
		if width > 60 {
			width = 60
		}
		ws.SetColumnWidth(i, width)
	}

	// Set header row height
	ws.SetRowHeight(0, 25)

	// Write data rows
	for rowIdx, rowData := range tableData.Data {
		excelRow := rowIdx + 1

		for colIdx := 0; colIdx < len(tableData.Columns) && colIdx < len(rowData); colIdx++ {
			cellName, _ := gospreadsheet.CellName(excelRow, colIdx)
			ws.SetCellValue(cellName, rowData[colIdx])
			ws.SetCellStyle(cellName, dataStyle)
		}

		ws.SetRowHeight(excelRow, 20)
	}

	// Freeze header row
	ws.FreezePane("A2")

	// Add metadata
	wb.Properties.Title = sheetName
	wb.Properties.Creator = "VantageData"
	wb.Properties.Description = "由 VantageData 智能分析系统生成"
	wb.Properties.Subject = "数据分析报表"
	wb.Properties.Keywords = "数据分析,报表,Excel"
	wb.Properties.Category = "数据分析"
	wb.Properties.LastModifiedBy = "VantageData"

	// Save to bytes
	var buf bytes.Buffer
	writer := gospreadsheet.NewXLSXWriter()
	if err := writer.Write(wb, &buf); err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportMultipleTablesToExcel exports multiple tables to different sheets in one Excel file
func (s *GoExcelExportService) ExportMultipleTablesToExcel(tables map[string]*TableData) ([]byte, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables to export")
	}

	wb := gospreadsheet.New()

	sheetIndex := 0
	for sheetName, tableData := range tables {
		if tableData == nil || len(tableData.Columns) == 0 {
			continue
		}

		var ws *gospreadsheet.Worksheet
		if sheetIndex == 0 {
			ws = wb.GetActiveSheet()
			ws.SetTitle(sheetName)
		} else {
			var err error
			ws, err = wb.AddSheet(sheetName)
			if err != nil {
				return nil, fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
			}
		}
		sheetIndex++

		// Header style
		headerStyle := gospreadsheet.NewStyle().
			SetFont(&gospreadsheet.Font{
				Bold:  true,
				Size:  11,
				Color: "FFFFFF",
				Name:  "Microsoft YaHei",
			}).
			SetFill(&gospreadsheet.Fill{
				Type:  "solid",
				Color: "4472C4",
			}).
			SetAlignment(&gospreadsheet.Alignment{
				Horizontal: gospreadsheet.AlignCenter,
				Vertical:   gospreadsheet.AlignMiddle,
			}).
			SetBorders(&gospreadsheet.Borders{
				Left:   gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
				Top:    gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
				Bottom: gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
				Right:  gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "FFFFFF"},
			})

		// Data style
		dataStyle := gospreadsheet.NewStyle().
			SetFont(&gospreadsheet.Font{
				Size: 10,
				Name: "Microsoft YaHei",
			}).
			SetAlignment(&gospreadsheet.Alignment{
				Horizontal: gospreadsheet.AlignLeft,
				Vertical:   gospreadsheet.AlignMiddle,
				WrapText:   true,
			}).
			SetBorders(&gospreadsheet.Borders{
				Left:   gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
				Top:    gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
				Bottom: gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
				Right:  gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D9D9D9"},
			})

		// Write headers
		for i, col := range tableData.Columns {
			cellName, _ := gospreadsheet.CellName(0, i)
			ws.SetCellValue(cellName, col.Title)
			ws.SetCellStyle(cellName, headerStyle)

			runeLen := len([]rune(col.Title))
			width := float64(runeLen) * 2.5
			if width < 12 {
				width = 12
			}
			if width > 60 {
				width = 60
			}
			ws.SetColumnWidth(i, width)
		}

		ws.SetRowHeight(0, 25)

		// Write data rows
		for rowIdx, rowData := range tableData.Data {
			excelRow := rowIdx + 1
			for colIdx := 0; colIdx < len(tableData.Columns) && colIdx < len(rowData); colIdx++ {
				cellName, _ := gospreadsheet.CellName(excelRow, colIdx)
				ws.SetCellValue(cellName, rowData[colIdx])
				ws.SetCellStyle(cellName, dataStyle)
			}
			ws.SetRowHeight(excelRow, 20)
		}

		// Freeze header row
		ws.FreezePane("A2")
	}

	// Add metadata
	wb.Properties.Title = "多表数据分析"
	wb.Properties.Creator = "VantageData"
	wb.Properties.Description = "由 VantageData 智能分析系统生成"
	wb.Properties.Subject = "数据分析报表"
	wb.Properties.Keywords = "数据分析,报表,Excel"
	wb.Properties.Category = "数据分析"
	wb.Properties.LastModifiedBy = "VantageData"

	// Save to bytes
	var buf bytes.Buffer
	writer := gospreadsheet.NewXLSXWriter()
	if err := writer.Write(wb, &buf); err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	return buf.Bytes(), nil
}
