package export

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	gospreadsheet "github.com/VantageDataChat/GoExcel"
	"vantagedata/i18n"
)

// GoExcelExportService handles Excel file generation using GoExcel (pure Go)
type GoExcelExportService struct{}

// NewGoExcelExportService creates a new GoExcel export service
func NewGoExcelExportService() *GoExcelExportService {
	return &GoExcelExportService{}
}

// isNumeric checks if a string value looks like a number
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return err == nil
}

// dateFormats lists the date formats checked by isDateLike.
var dateFormats = []string{
	"2006-01-02",
	"2006/01/02",
	"2006-01-02 15:04:05",
	"2006/01/02 15:04:05",
	"01/02/2006",
	"02-Jan-2006",
}

// isDateLike checks if a string looks like a date (yyyy-mm-dd or similar)
func isDateLike(s string) bool {
	trimmed := strings.TrimSpace(s)
	for _, f := range dateFormats {
		if _, err := time.Parse(f, trimmed); err == nil {
			return true
		}
	}
	return false
}

// isPercentLike checks if a string looks like a percentage (e.g. "12.5%")
func isPercentLike(s string) bool {
	trimmed := strings.TrimSpace(s)
	if strings.HasSuffix(trimmed, "%") {
		numPart := strings.TrimSuffix(trimmed, "%")
		_, err := strconv.ParseFloat(strings.TrimSpace(numPart), 64)
		return err == nil
	}
	return false
}

// detectColumnType analyzes data to determine the best number format for a column.
// Samples up to 200 rows to avoid performance issues with large datasets.
func detectColumnType(data [][]interface{}, colIdx int) string {
	if len(data) == 0 || colIdx < 0 {
		return "text"
	}

	maxSample := 200
	sampleSize := len(data)
	if sampleSize > maxSample {
		sampleSize = maxSample
	}

	numericCount := 0
	dateCount := 0
	percentCount := 0
	total := 0

	for i := 0; i < sampleSize; i++ {
		row := data[i]
		if colIdx >= len(row) {
			continue
		}
		val := cellToString(row[colIdx])
		if val == "" {
			continue
		}
		total++
		if isPercentLike(val) {
			percentCount++
		} else if isNumeric(val) {
			numericCount++
		} else if isDateLike(val) {
			dateCount++
		}
	}

	if total == 0 {
		return "text"
	}

	threshold := float64(total) * 0.8
	if float64(percentCount) >= threshold {
		return "percent"
	}
	if float64(dateCount) >= threshold {
		return "date"
	}
	if float64(numericCount) >= threshold {
		return "numeric"
	}
	return "text"
}

// cellToString converts a cell value to string without fmt.Sprintf overhead.
func cellToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

// calcColumnWidth calculates optimal column width based on header and data content
func calcColumnWidth(title string, data [][]interface{}, colIdx int, maxSample int) float64 {
	maxLen := len([]rune(title))

	sampleSize := len(data)
	if sampleSize > maxSample {
		sampleSize = maxSample
	}

	for i := 0; i < sampleSize; i++ {
		if colIdx < len(data[i]) {
			val := cellToString(data[i][colIdx])
			runeLen := len([]rune(val))
			if runeLen > maxLen {
				maxLen = runeLen
			}
		}
	}

	width := float64(maxLen)*2.2 + 4
	if width < 10 {
		width = 10
	}
	if width > 50 {
		width = 50
	}
	return width
}

// createHeaderStyle creates the header row style
func createHeaderStyle() *gospreadsheet.Style {
	return gospreadsheet.NewStyle().
		SetFont(&gospreadsheet.Font{
			Bold:  true,
			Size:  11,
			Color: "FFFFFF",
			Name:  "Microsoft YaHei",
		}).
		SetFill(&gospreadsheet.Fill{
			Type:  "solid",
			Color: "10B981", // emerald-500
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
}

// createDataStyle creates the data cell style with optional alternating row color
func createDataStyle(isEvenRow bool, colType string) *gospreadsheet.Style {
	bgColor := "FFFFFF"
	if isEvenRow {
		bgColor = "F0FDF4" // emerald-50 淡绿色交替行
	}

	hAlign := gospreadsheet.AlignLeft
	if colType == "numeric" || colType == "percent" {
		hAlign = gospreadsheet.AlignRight
	} else if colType == "date" {
		hAlign = gospreadsheet.AlignCenter
	}

	style := gospreadsheet.NewStyle().
		SetFont(&gospreadsheet.Font{
			Size: 10,
			Name: "Microsoft YaHei",
		}).
		SetFill(&gospreadsheet.Fill{
			Type:  "solid",
			Color: bgColor,
		}).
		SetAlignment(&gospreadsheet.Alignment{
			Horizontal: hAlign,
			Vertical:   gospreadsheet.AlignMiddle,
			WrapText:   true,
		}).
		SetBorders(&gospreadsheet.Borders{
			Left:   gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D1FAE5"}, // emerald-100
			Top:    gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D1FAE5"},
			Bottom: gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D1FAE5"},
			Right:  gospreadsheet.Border{Style: gospreadsheet.BorderThin, Color: "D1FAE5"},
		})

	// Apply number format based on column type
	switch colType {
	case "numeric":
		style.SetNumberFormat(&gospreadsheet.NumberFormat{FormatCode: "#,##0.##"})
	case "percent":
		style.SetNumberFormat(&gospreadsheet.NumberFormat{FormatCode: "0.00%"})
	case "date":
		style.SetNumberFormat(&gospreadsheet.NumberFormat{FormatCode: "yyyy-mm-dd"})
	}

	return style
}

// writeSheetData writes table data to a worksheet with full formatting
func writeSheetData(ws *gospreadsheet.Worksheet, tableData *TableData) {
	if tableData == nil || len(tableData.Columns) == 0 {
		return
	}

	colCount := len(tableData.Columns)

	// Detect column types from data
	colTypes := make([]string, colCount)
	for i, col := range tableData.Columns {
		// Use DataType hint if available, otherwise detect from data
		switch strings.ToLower(col.DataType) {
		case "number", "numeric", "int", "float", "decimal":
			colTypes[i] = "numeric"
		case "percent", "percentage":
			colTypes[i] = "percent"
		case "date", "datetime", "time":
			colTypes[i] = "date"
		default:
			colTypes[i] = detectColumnType(tableData.Data, i)
		}
	}

	// Write headers
	headerStyle := createHeaderStyle()
	for i, col := range tableData.Columns {
		cellName, _ := gospreadsheet.CellName(0, i)
		ws.SetCellValue(cellName, col.Title)
		ws.SetCellStyle(cellName, headerStyle)

		// Calculate column width from header + data
		width := calcColumnWidth(col.Title, tableData.Data, i, 100)
		ws.SetColumnWidth(i, width)
	}
	ws.SetRowHeight(0, 28)

	// Write data rows with alternating colors
	// Pre-create style cache: [even/odd][colType] to avoid per-cell allocation
	type styleKey struct {
		isEven  bool
		colType string
	}
	styleCache := make(map[styleKey]*gospreadsheet.Style)
	getStyle := func(isEven bool, colType string) *gospreadsheet.Style {
		k := styleKey{isEven, colType}
		if s, ok := styleCache[k]; ok {
			return s
		}
		s := createDataStyle(isEven, colType)
		styleCache[k] = s
		return s
	}

	for rowIdx, rowData := range tableData.Data {
		excelRow := rowIdx + 1
		isEven := rowIdx%2 == 0

		for colIdx := 0; colIdx < colCount && colIdx < len(rowData); colIdx++ {
			cellName, _ := gospreadsheet.CellName(excelRow, colIdx)
			val := rowData[colIdx]

			// Set cell value with proper type handling
			switch colTypes[colIdx] {
			case "numeric":
				switch v := val.(type) {
				case float64:
					ws.SetCellValue(cellName, v)
				case int:
					ws.SetCellValue(cellName, float64(v))
				case int64:
					ws.SetCellValue(cellName, float64(v))
				default:
					strVal := cellToString(val)
					if f, err := strconv.ParseFloat(strVal, 64); err == nil {
						ws.SetCellValue(cellName, f)
					} else {
						ws.SetCellValue(cellName, val)
					}
				}
			case "percent":
				strVal := cellToString(val)
				if strings.HasSuffix(strVal, "%") {
					numPart := strings.TrimSuffix(strVal, "%")
					if f, err := strconv.ParseFloat(strings.TrimSpace(numPart), 64); err == nil {
						ws.SetCellValue(cellName, f/100.0) // Excel stores percent as decimal
					} else {
						ws.SetCellValue(cellName, val)
					}
				} else {
					ws.SetCellValue(cellName, val)
				}
			default:
				ws.SetCellValue(cellName, val)
			}

			ws.SetCellStyle(cellName, getStyle(isEven, colTypes[colIdx]))
		}
		ws.SetRowHeight(excelRow, 22)
	}

	// Freeze header row
	ws.FreezePane("A2")

	// Add auto-filter on header row if there's data
	if len(tableData.Data) > 0 {
		lastColName, _ := gospreadsheet.ColumnIndexToName(colCount - 1)
		lastRow := len(tableData.Data) + 1
		filterRange := fmt.Sprintf("A1:%s%d", lastColName, lastRow)
		af := gospreadsheet.NewAutoFilter(filterRange)
		ws.SetAutoFilter(af)
	}

	// Page setup for printing
	ps := gospreadsheet.NewPageSetup().
		SetOrientation(gospreadsheet.OrientationLandscape).
		SetPaperSize(gospreadsheet.PaperA4).
		SetFitToPage(1, 0). // Fit to 1 page wide, auto height
		SetRepeatRows("1:1") // Repeat header row on each printed page
	ws.SetPageSetup(ps)
}

// setWorkbookMetadata sets common metadata on the workbook
func setWorkbookMetadata(wb *gospreadsheet.Workbook, title string) {
	wb.Properties.Title = title
	wb.Properties.Creator = "VantageData"
	wb.Properties.Description = i18n.T("export.doc_description")
	wb.Properties.Subject = i18n.T("excel.report_subject")
	wb.Properties.Keywords = i18n.T("excel.report_keywords")
	wb.Properties.Category = i18n.T("excel.report_category")
	wb.Properties.LastModifiedBy = "VantageData"
}

// writeWorkbookToBytes serializes the workbook to xlsx bytes
func writeWorkbookToBytes(wb *gospreadsheet.Workbook) ([]byte, error) {
	var buf bytes.Buffer
	writer := gospreadsheet.NewXLSXWriter()
	if err := writer.Write(wb, &buf); err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}
	return buf.Bytes(), nil
}

// ExportTableToExcel exports table data to Excel format using GoExcel
func (s *GoExcelExportService) ExportTableToExcel(tableData *TableData, sheetName string) ([]byte, error) {
	if tableData == nil || len(tableData.Columns) == 0 {
		return nil, fmt.Errorf("no table data to export")
	}

	wb := gospreadsheet.New()
	ws := wb.GetActiveSheet()

	if sheetName == "" {
		sheetName = i18n.T("excel.default_sheet_name")
	}
	ws.SetTitle(sheetName)

	writeSheetData(ws, tableData)

	setWorkbookMetadata(wb, sheetName)

	return writeWorkbookToBytes(wb)
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

		writeSheetData(ws, tableData)
	}

	if sheetIndex == 0 {
		return nil, fmt.Errorf("no valid tables to export")
	}

	setWorkbookMetadata(wb, i18n.T("excel.multi_table_title"))

	return writeWorkbookToBytes(wb)
}

// ExportOrderedTablesToExcel exports multiple tables to different sheets preserving insertion order
func (s *GoExcelExportService) ExportOrderedTablesToExcel(tables []NamedTable) ([]byte, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables to export")
	}

	wb := gospreadsheet.New()

	sheetIndex := 0
	for _, namedTable := range tables {
		tableData := namedTable.Table
		sheetName := namedTable.Name

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

		writeSheetData(ws, tableData)
	}

	if sheetIndex == 0 {
		return nil, fmt.Errorf("no valid tables to export")
	}

	setWorkbookMetadata(wb, i18n.T("excel.multi_table_title"))

	return writeWorkbookToBytes(wb)
}
