package export

import (
	"fmt"
)

// ExcelExportService handles Excel file generation using GoExcel
type ExcelExportService struct {
	service *GoExcelExportService
}

// NewExcelExportService creates a new Excel export service
func NewExcelExportService() *ExcelExportService {
	return &ExcelExportService{
		service: NewGoExcelExportService(),
	}
}

// ExportTableToExcel exports table data to Excel format
func (s *ExcelExportService) ExportTableToExcel(tableData *TableData, sheetName string) ([]byte, error) {
	return s.service.ExportTableToExcel(tableData, sheetName)
}

// ExportMultipleTablesToExcel exports multiple tables to different sheets in one Excel file
func (s *ExcelExportService) ExportMultipleTablesToExcel(tables map[string]*TableData) ([]byte, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables to export")
	}
	return s.service.ExportMultipleTablesToExcel(tables)
}
