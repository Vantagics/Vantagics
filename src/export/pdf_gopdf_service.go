package export

import (
	"bytes"
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

// ExportDashboardToPDF exports dashboard data to PDF using gopdf
func (s *GopdfService) ExportDashboardToPDF(data DashboardData) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	
	// Add a page
	pdf.AddPage()
	
	// Try multiple font paths for better compatibility
	// Priority: Microsoft YaHei (best looking) > SimSun > SimHei > Arial Unicode MS
	fontPaths := []struct {
		name string
		path string
	}{
		{"msyh", "C:\\Windows\\Fonts\\msyh.ttc"},              // 微软雅黑 (推荐，现代美观)
		{"msyhbd", "C:\\Windows\\Fonts\\msyhbd.ttc"},          // 微软雅黑粗体
		{"simsun", "C:\\Windows\\Fonts\\simsun.ttc"},          // 宋体 (经典)
		{"simhei", "C:\\Windows\\Fonts\\simhei.ttf"},          // 黑体
		{"arialuni", "C:\\Windows\\Fonts\\ARIALUNI.TTF"},      // Arial Unicode MS
		{"arial", "C:\\Windows\\Fonts\\arial.ttf"},            // Arial (fallback)
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
	
	// Add header
	s.addHeader(&pdf, "智能仪表盘报告", fontName)
	
	// Add user request
	if data.UserRequest != "" {
		s.addUserRequest(&pdf, data.UserRequest, fontName)
	}
	
	// Add metrics
	if len(data.Metrics) > 0 {
		s.addMetrics(&pdf, data.Metrics, fontName)
	}
	
	// Add insights
	if len(data.Insights) > 0 {
		s.addInsights(&pdf, data.Insights, fontName)
	}
	
	// Add table
	if data.TableData != nil && len(data.TableData.Columns) > 0 {
		s.addTable(&pdf, data.TableData, fontName)
	}
	
	// Add footer
	s.addFooter(&pdf, fontName)
	
	// Get PDF bytes
	var buf bytes.Buffer
	_, err = pdf.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write PDF: %w", err)
	}
	
	return buf.Bytes(), nil
}

func (s *GopdfService) addHeader(pdf *gopdf.GoPdf, title string, fontName string) {
	// Title
	pdf.SetFont(fontName, "B", 18)
	pdf.SetTextColor(59, 130, 246)
	pdf.SetX(50)
	pdf.SetY(30)
	pdf.Cell(nil, title)
	
	// Timestamp
	pdf.SetFont(fontName, "", 9)
	pdf.SetTextColor(100, 116, 139)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	pdf.SetX(70)
	pdf.SetY(45)
	pdf.Cell(nil, fmt.Sprintf("生成时间: %s", timestamp))
	
	pdf.SetY(60)
}

func (s *GopdfService) addUserRequest(pdf *gopdf.GoPdf, request string, fontName string) {
	y := pdf.GetY()
	
	pdf.SetFont(fontName, "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetX(20)
	pdf.SetY(y)
	pdf.Cell(nil, "用户请求")
	
	pdf.SetFont(fontName, "", 10)
	pdf.SetX(20)
	pdf.SetY(y + 10)
	
	// Split text into lines manually since gopdf MultiCell API is different
	words := strings.Split(request, " ")
	line := ""
	lineY := y + 10
	for _, word := range words {
		testLine := line + word + " "
		if len(testLine) > 80 { // Approximate line length
			pdf.SetX(20)
			pdf.SetY(lineY)
			pdf.Cell(nil, line)
			line = word + " "
			lineY += 6
		} else {
			line = testLine
		}
	}
	if line != "" {
		pdf.SetX(20)
		pdf.SetY(lineY)
		pdf.Cell(nil, line)
		lineY += 6
	}
	
	pdf.SetY(lineY + 5)
}

func (s *GopdfService) addMetrics(pdf *gopdf.GoPdf, metrics []MetricData, fontName string) {
	y := pdf.GetY()
	
	pdf.SetFont(fontName, "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetX(20)
	pdf.SetY(y)
	pdf.Cell(nil, "关键指标")
	
	pdf.SetFont(fontName, "", 9)
	y += 10
	
	for i := 0; i < len(metrics); i += 2 {
		// First metric
		metric1 := metrics[i]
		text1 := fmt.Sprintf("%s: %s (%s)", metric1.Title, metric1.Value, metric1.Change)
		pdf.SetX(20)
		pdf.SetY(y)
		pdf.Cell(nil, text1)
		
		// Second metric (if exists)
		if i+1 < len(metrics) {
			metric2 := metrics[i+1]
			text2 := fmt.Sprintf("%s: %s (%s)", metric2.Title, metric2.Value, metric2.Change)
			pdf.SetX(120)
			pdf.SetY(y)
			pdf.Cell(nil, text2)
		}
		
		y += 8
	}
	
	pdf.SetY(y + 5)
}

func (s *GopdfService) addInsights(pdf *gopdf.GoPdf, insights []string, fontName string) {
	y := pdf.GetY()
	
	pdf.SetFont(fontName, "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetX(20)
	pdf.SetY(y)
	pdf.Cell(nil, "智能洞察")
	
	pdf.SetFont(fontName, "", 9)
	y += 10
	
	for i, insight := range insights {
		text := fmt.Sprintf("%d. %s", i+1, insight)
		
		// Split text into lines
		words := strings.Split(text, " ")
		line := ""
		lineY := y
		for _, word := range words {
			testLine := line + word + " "
			if len(testLine) > 80 {
				pdf.SetX(20)
				pdf.SetY(lineY)
				pdf.Cell(nil, line)
				line = word + " "
				lineY += 6
			} else {
				line = testLine
			}
		}
		if line != "" {
			pdf.SetX(20)
			pdf.SetY(lineY)
			pdf.Cell(nil, line)
			lineY += 6
		}
		
		y = lineY + 3
	}
	
	pdf.SetY(y + 5)
}

func (s *GopdfService) addTable(pdf *gopdf.GoPdf, tableData *TableData, fontName string) {
	y := pdf.GetY()
	
	pdf.SetFont(fontName, "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetX(20)
	pdf.SetY(y)
	pdf.Cell(nil, "数据表格")
	
	y += 10
	
	// Limit columns
	maxCols := 6
	if len(tableData.Columns) > maxCols {
		tableData.Columns = tableData.Columns[:maxCols]
	}
	
	// Calculate column width
	colWidth := 170.0 / float64(len(tableData.Columns))
	
	// Table header
	pdf.SetFont(fontName, "B", 8)
	pdf.SetFillColor(68, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	
	x := 20.0
	for _, col := range tableData.Columns {
		pdf.SetX(x)
		pdf.SetY(y)
		pdf.CellWithOption(&gopdf.Rect{W: colWidth, H: 7}, col.Title, gopdf.CellOption{
			Align:  gopdf.Center | gopdf.Middle,
			Border: gopdf.AllBorders,
		})
		x += colWidth
	}
	
	y += 7
	
	// Table data
	pdf.SetFont(fontName, "", 7)
	pdf.SetTextColor(0, 0, 0)
	
	maxRows := 50
	if len(tableData.Data) > maxRows {
		tableData.Data = tableData.Data[:maxRows]
	}
	
	for _, row := range tableData.Data {
		x = 20.0
		for i := 0; i < len(tableData.Columns) && i < len(row); i++ {
			cellValue := fmt.Sprintf("%v", row[i])
			if len(cellValue) > 30 {
				cellValue = cellValue[:27] + "..."
			}
			pdf.SetX(x)
			pdf.SetY(y)
			pdf.CellWithOption(&gopdf.Rect{W: colWidth, H: 6}, cellValue, gopdf.CellOption{
				Align:  gopdf.Left | gopdf.Middle,
				Border: gopdf.AllBorders,
			})
			x += colWidth
		}
		y += 6
	}
	
	pdf.SetY(y + 5)
}

func (s *GopdfService) addFooter(pdf *gopdf.GoPdf, fontName string) {
	pdf.SetFont(fontName, "", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.SetX(70)
	pdf.SetY(280)
	pdf.Cell(nil, "由 VantageData 智能分析系统生成")
}
