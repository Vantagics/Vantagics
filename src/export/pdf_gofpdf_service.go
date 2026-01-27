package export

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// GofpdfService handles PDF generation using gofpdf with Chinese font support
type GofpdfService struct{}

// NewGofpdfService creates a new gofpdf service
func NewGofpdfService() *GofpdfService {
	return &GofpdfService{}
}

// ExportDashboardToPDF exports dashboard data to PDF using gofpdf
func (s *GofpdfService) ExportDashboardToPDF(data DashboardData) ([]byte, error) {
	// Create PDF with UTF-8 support
	pdf := gofpdf.New("P", "mm", "A4", "")
	
	// Use Arial Unicode which has better Chinese support
	// gofpdf's Arial implementation supports Unicode
	pdf.SetFont("Arial", "", 12)
	
	// Add page
	pdf.AddPage()
	
	// Add header
	s.addHeader(pdf, "智能仪表盘报告")
	
	// Add user request
	if data.UserRequest != "" {
		s.addUserRequest(pdf, data.UserRequest)
	}
	
	// Add metrics
	if len(data.Metrics) > 0 {
		s.addMetrics(pdf, data.Metrics)
	}
	
	// Add insights
	if len(data.Insights) > 0 {
		s.addInsights(pdf, data.Insights)
	}
	
	// Add table
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTable(pdf, data.TableData)
	}
	
	// Add footer
	s.addFooter(pdf)
	
	// Check for errors
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("PDF generation error: %w", err)
	}
	
	// Get PDF bytes using a buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to output PDF: %w", err)
	}
	
	return buf.Bytes(), nil
}

func (s *GofpdfService) addHeader(pdf *gofpdf.Fpdf, title string) {
	// Title
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(59, 130, 246)
	pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
	
	// Timestamp
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 116, 139)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	pdf.CellFormat(0, 6, fmt.Sprintf("生成时间: %s", timestamp), "", 1, "C", false, 0, "")
	
	pdf.Ln(5)
}

func (s *GofpdfService) addUserRequest(pdf *gofpdf.Fpdf, request string) {
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 8, "用户请求", "", 1, "L", false, 0, "")
	
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(0, 6, request, "", "L", false)
	
	pdf.Ln(3)
}

func (s *GofpdfService) addMetrics(pdf *gofpdf.Fpdf, metrics []MetricData) {
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 8, "关键指标", "", 1, "L", false, 0, "")
	
	pdf.SetFont("Arial", "", 9)
	
	// Add metrics in 2 columns
	for i := 0; i < len(metrics); i += 2 {
		x := pdf.GetX()
		y := pdf.GetY()
		
		// First metric
		metric1 := metrics[i]
		text1 := fmt.Sprintf("%s: %s (%s)", metric1.Title, metric1.Value, metric1.Change)
		pdf.MultiCell(90, 6, text1, "", "L", false)
		
		// Second metric (if exists)
		if i+1 < len(metrics) {
			pdf.SetXY(x+100, y)
			metric2 := metrics[i+1]
			text2 := fmt.Sprintf("%s: %s (%s)", metric2.Title, metric2.Value, metric2.Change)
			pdf.MultiCell(90, 6, text2, "", "L", false)
		}
		
		pdf.Ln(2)
	}
	
	pdf.Ln(3)
}

func (s *GofpdfService) addInsights(pdf *gofpdf.Fpdf, insights []string) {
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 8, "智能洞察", "", 1, "L", false, 0, "")
	
	pdf.SetFont("Arial", "", 9)
	
	for i, insight := range insights {
		text := fmt.Sprintf("%d. %s", i+1, insight)
		pdf.MultiCell(0, 6, text, "", "L", false)
		pdf.Ln(1)
	}
	
	pdf.Ln(3)
}

func (s *GofpdfService) addTable(pdf *gofpdf.Fpdf, tableData *TableData) {
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 8, "数据表格", "", 1, "L", false, 0, "")
	
	// Limit columns
	maxCols := 6
	if len(tableData.Columns) > maxCols {
		tableData.Columns = tableData.Columns[:maxCols]
	}
	
	// Calculate column width
	colWidth := 190.0 / float64(len(tableData.Columns))
	
	// Table header
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(68, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	
	for _, col := range tableData.Columns {
		pdf.CellFormat(colWidth, 7, col.Title, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	
	// Table data
	pdf.SetFont("Arial", "", 7)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFillColor(248, 250, 252)
	
	fill := false
	maxRows := 50
	if len(tableData.Data) > maxRows {
		tableData.Data = tableData.Data[:maxRows]
	}
	
	for _, row := range tableData.Data {
		for i := 0; i < len(tableData.Columns) && i < len(row); i++ {
			cellValue := fmt.Sprintf("%v", row[i])
			if len(cellValue) > 30 {
				cellValue = cellValue[:27] + "..."
			}
			pdf.CellFormat(colWidth, 6, cellValue, "1", 0, "L", fill, 0, "")
		}
		pdf.Ln(-1)
		fill = !fill
	}
	
	pdf.Ln(3)
}

func (s *GofpdfService) addFooter(pdf *gofpdf.Fpdf) {
	pdf.SetY(-15)
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 10, "由 VantageData 智能分析系统生成", "", 0, "C", false, 0, "")
}
