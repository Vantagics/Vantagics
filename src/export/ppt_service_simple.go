package export

// PPTExportService handles PowerPoint generation using GoPPT (pure Go, zero dependencies)
type PPTExportService struct {
	service *GoPPTService
}

// NewPPTExportService creates a new PPT export service
func NewPPTExportService() *PPTExportService {
	return &PPTExportService{
		service: NewGoPPTService(),
	}
}

// ExportDashboardToPPT exports dashboard data to PowerPoint format
func (s *PPTExportService) ExportDashboardToPPT(data DashboardData) ([]byte, error) {
	return s.service.ExportDashboardToPPT(data)
}
