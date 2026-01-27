package export

// PPTExportService handles PowerPoint generation using gooxml (open source)
type PPTExportService struct {
	service *GooxmlPPTService
}

// NewPPTExportService creates a new PPT export service
func NewPPTExportService() *PPTExportService {
	return &PPTExportService{
		service: NewGooxmlPPTService(),
	}
}

// ExportDashboardToPPT exports dashboard data to PowerPoint format
func (s *PPTExportService) ExportDashboardToPPT(data DashboardData) ([]byte, error) {
	return s.service.ExportDashboardToPPT(data)
}
