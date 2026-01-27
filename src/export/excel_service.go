package export

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExcelExportService handles Excel file generation using excelize
type ExcelExportService struct{}

// NewExcelExportService creates a new Excel export service
func NewExcelExportService() *ExcelExportService {
	return &ExcelExportService{}
}

// ExportTableToExcel exports table data to Excel format
func (s *ExcelExportService) ExportTableToExcel(tableData *TableData, sheetName string) ([]byte, error) {
	if tableData == nil || len(tableData.Columns) == 0 {
		return nil, fmt.Errorf("no table data to export")
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create or use default sheet
	if sheetName == "" {
		sheetName = "数据表"
	}
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet: %w", err)
	}
	f.SetActiveSheet(index)

	// Define header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "FFFFFF",
			Family: "Microsoft YaHei",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	// Define data style
	dataStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Family: "Microsoft YaHei",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create data style: %w", err)
	}

	// Write headers
	for i, col := range tableData.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col.Title)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
		
		// Set column width based on title length
		width := float64(len(col.Title)) * 1.5
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, colName, colName, width)
	}

	// Set header row height
	f.SetRowHeight(sheetName, 1, 25)

	// Write data rows
	for rowIdx, rowData := range tableData.Data {
		excelRow := rowIdx + 2 // Excel rows start at 1, header is row 1
		
		for colIdx := 0; colIdx < len(tableData.Columns) && colIdx < len(rowData); colIdx++ {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, excelRow)
			
			// Set cell value based on data type
			cellValue := rowData[colIdx]
			f.SetCellValue(sheetName, cell, cellValue)
			f.SetCellStyle(sheetName, cell, cell, dataStyle)
		}
		
		// Set row height
		f.SetRowHeight(sheetName, excelRow, 20)
	}

	// Add auto-filter
	lastCol, _ := excelize.ColumnNumberToName(len(tableData.Columns))
	lastRow := len(tableData.Data) + 1
	filterRange := fmt.Sprintf("A1:%s%d", lastCol, lastRow)
	f.AutoFilter(sheetName, filterRange, []excelize.AutoFilterOptions{})

	// Freeze header row
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Add metadata
	f.SetDocProps(&excelize.DocProperties{
		Category:       "数据分析",
		ContentStatus:  "Final",
		Created:        time.Now().Format(time.RFC3339),
		Creator:        "VantageData",
		Description:    "由 VantageData 智能分析系统生成",
		Identifier:     "xlsx",
		Keywords:       "数据分析,报表,Excel",
		LastModifiedBy: "VantageData",
		Revision:       "1",
		Subject:        "数据分析报表",
		Title:          sheetName,
		Language:       "zh-CN",
		Version:        "1.0",
	})

	// Save to buffer
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	return buffer.Bytes(), nil
}

// ExportMultipleTablesToExcel exports multiple tables to different sheets in one Excel file
func (s *ExcelExportService) ExportMultipleTablesToExcel(tables map[string]*TableData) ([]byte, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables to export")
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Delete default Sheet1
	f.DeleteSheet("Sheet1")

	sheetIndex := 0
	for sheetName, tableData := range tables {
		if tableData == nil || len(tableData.Columns) == 0 {
			continue
		}

		// Create sheet
		index, err := f.NewSheet(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
		}

		// Set first sheet as active
		if sheetIndex == 0 {
			f.SetActiveSheet(index)
		}
		sheetIndex++

		// Define styles (same as single table export)
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Bold:   true,
				Size:   11,
				Color:  "FFFFFF",
				Family: "Microsoft YaHei",
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{"4472C4"},
				Pattern: 1,
			},
			Alignment: &excelize.Alignment{
				Horizontal: "center",
				Vertical:   "center",
			},
			Border: []excelize.Border{
				{Type: "left", Color: "FFFFFF", Style: 1},
				{Type: "top", Color: "FFFFFF", Style: 1},
				{Type: "bottom", Color: "FFFFFF", Style: 1},
				{Type: "right", Color: "FFFFFF", Style: 1},
			},
		})

		dataStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   10,
				Family: "Microsoft YaHei",
			},
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
				WrapText:   true,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "D9D9D9", Style: 1},
				{Type: "top", Color: "D9D9D9", Style: 1},
				{Type: "bottom", Color: "D9D9D9", Style: 1},
				{Type: "right", Color: "D9D9D9", Style: 1},
			},
		})

		// Write headers
		for i, col := range tableData.Columns {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, col.Title)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)

			width := float64(len(col.Title)) * 1.5
			if width < 10 {
				width = 10
			}
			if width > 50 {
				width = 50
			}
			colName, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheetName, colName, colName, width)
		}

		f.SetRowHeight(sheetName, 1, 25)

		// Write data rows
		for rowIdx, rowData := range tableData.Data {
			excelRow := rowIdx + 2
			for colIdx := 0; colIdx < len(tableData.Columns) && colIdx < len(rowData); colIdx++ {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, excelRow)
				f.SetCellValue(sheetName, cell, rowData[colIdx])
				f.SetCellStyle(sheetName, cell, cell, dataStyle)
			}
			f.SetRowHeight(sheetName, excelRow, 20)
		}

		// Add auto-filter
		lastCol, _ := excelize.ColumnNumberToName(len(tableData.Columns))
		lastRow := len(tableData.Data) + 1
		filterRange := fmt.Sprintf("A1:%s%d", lastCol, lastRow)
		f.AutoFilter(sheetName, filterRange, []excelize.AutoFilterOptions{})

		// Freeze header row
		f.SetPanes(sheetName, &excelize.Panes{
			Freeze:      true,
			Split:       false,
			XSplit:      0,
			YSplit:      1,
			TopLeftCell: "A2",
			ActivePane:  "bottomLeft",
		})
	}

	// Add metadata
	f.SetDocProps(&excelize.DocProperties{
		Category:       "数据分析",
		ContentStatus:  "Final",
		Created:        time.Now().Format(time.RFC3339),
		Creator:        "VantageData",
		Description:    "由 VantageData 智能分析系统生成",
		Identifier:     "xlsx",
		Keywords:       "数据分析,报表,Excel",
		LastModifiedBy: "VantageData",
		Revision:       "1",
		Subject:        "数据分析报表",
		Title:          "多表数据分析",
		Language:       "zh-CN",
		Version:        "1.0",
	})

	// Save to buffer
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	return buffer.Bytes(), nil
}
