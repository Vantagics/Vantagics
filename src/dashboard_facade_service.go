package main

import (
	"context"
	"fmt"
	"vantagics/agent"
	"vantagics/database"
)

// DashboardManager 定义仪表盘管理接�?
type DashboardManager interface {
	GetDashboardData() DashboardData
	SaveLayout(config database.LayoutConfiguration) error
	LoadLayout(userID string) (*database.LayoutConfiguration, error)
	CheckComponentHasData(componentType string, instanceID string) (bool, error)
	GetFilesByCategory(category string) ([]database.FileInfo, error)
	DownloadFile(fileID string) (string, error)
	ExportDashboard(req database.ExportRequest) (*database.ExportResult, error)
}

// DashboardFacadeService 仪表盘服务门面，封装所有仪表盘相关的业务逻辑
type DashboardFacadeService struct {
	ctx               context.Context
	dataSourceService *agent.DataSourceService
	configProvider    ConfigProvider
	layoutService     *database.LayoutService
	dataService       *database.DataService
	fileService       *database.FileService
	exportService     *database.ExportService
	logger            func(string)
}

// NewDashboardFacadeService 创建新的 DashboardFacadeService 实例
func NewDashboardFacadeService(
	dataSourceService *agent.DataSourceService,
	configProvider ConfigProvider,
	layoutService *database.LayoutService,
	dataService *database.DataService,
	fileService *database.FileService,
	exportService *database.ExportService,
	logger func(string),
) *DashboardFacadeService {
	return &DashboardFacadeService{
		dataSourceService: dataSourceService,
		configProvider:    configProvider,
		layoutService:     layoutService,
		dataService:       dataService,
		fileService:       fileService,
		exportService:     exportService,
		logger:            logger,
	}
}

// Name 返回服务名称
func (d *DashboardFacadeService) Name() string {
	return "dashboard"
}

// Initialize 初始化仪表盘门面服务
func (d *DashboardFacadeService) Initialize(ctx context.Context) error {
	d.ctx = ctx
	d.log("DashboardFacadeService initialized")
	return nil
}

// Shutdown 关闭仪表盘门面服�?
func (d *DashboardFacadeService) Shutdown() error {
	return nil
}

// SetContext 设置 Wails 上下�?
func (d *DashboardFacadeService) SetContext(ctx context.Context) {
	d.ctx = ctx
}

// log 记录日志
func (d *DashboardFacadeService) log(msg string) {
	if d.logger != nil {
		d.logger(msg)
	}
}

// --- Dashboard Data Methods ---

// getDashboardTranslations 返回仪表盘翻译字符串
func (d *DashboardFacadeService) getDashboardTranslations(lang string) map[string]string {
	if lang == "简体中�? {
		return map[string]string{
			"Data Sources":  "数据�?,
			"Total":         "总计",
			"Files":         "文件",
			"Local":         "本地",
			"Databases":     "数据�?,
			"Connected":     "已连�?,
			"Tables":        "数据�?,
			"Analyzed":      "已分�?,
			"ConnectPrompt": "连接数据源以开始使用�?,
			"Analyze":       "分析",
		}
	}
	return map[string]string{
		"Data Sources":  "Data Sources",
		"Total":         "Total",
		"Files":         "Files",
		"Local":         "Local",
		"Databases":     "Databases",
		"Connected":     "Connected",
		"Tables":        "Tables",
		"Analyzed":      "Analyzed",
		"ConnectPrompt": "Connect a data source to get started.",
		"Analyze":       "Analyze",
	}
}

// GetDashboardData 返回数据源的摘要统计和洞察信�?
func (d *DashboardFacadeService) GetDashboardData() DashboardData {
	if d.dataSourceService == nil {
		return DashboardData{}
	}

	cfg, _ := d.configProvider.GetConfig()
	tr := d.getDashboardTranslations(cfg.Language)

	sources, _ := d.dataSourceService.LoadDataSources()

	var excelCount, dbCount int
	var totalTables int

	for _, ds := range sources {
		if ds.Type == "excel" || ds.Type == "csv" {
			excelCount++
		} else {
			dbCount++
		}

		if ds.Analysis != nil {
			totalTables += len(ds.Analysis.Schema)
		}
	}

	metrics := []Metric{
		{Title: tr["Data Sources"], Value: fmt.Sprintf("%d", len(sources)), Change: tr["Total"]},
		{Title: tr["Files"], Value: fmt.Sprintf("%d", excelCount), Change: tr["Local"]},
		{Title: tr["Databases"], Value: fmt.Sprintf("%d", dbCount), Change: tr["Connected"]},
		{Title: tr["Tables"], Value: fmt.Sprintf("%d", totalTables), Change: tr["Analyzed"]},
	}

	var insights []Insight
	for _, ds := range sources {
		desc := ds.Name
		if ds.Analysis != nil && ds.Analysis.Summary != "" {
			desc = ds.Analysis.Summary
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
		}

		icon := "info"
		if ds.Type == "excel" {
			icon = "file-text"
		} else if ds.Type == "mysql" {
			icon = "database"
		}

		insights = append(insights, Insight{
			Text:         fmt.Sprintf("%s %s (%s)", tr["Analyze"], ds.Name, ds.Type),
			Icon:         icon,
			DataSourceID: ds.ID,
			SourceName:   ds.Name,
		})
	}

	if len(insights) == 0 {
		insights = append(insights, Insight{Text: tr["ConnectPrompt"], Icon: "info"})
	}

	return DashboardData{
		Metrics:  metrics,
		Insights: insights,
	}
}

// --- Layout Methods ---

// SaveLayout 保存布局配置到数据库
func (d *DashboardFacadeService) SaveLayout(config database.LayoutConfiguration) error {
	if d.layoutService == nil {
		return WrapError("dashboard", "SaveLayout", fmt.Errorf("layout service not initialized"))
	}

	d.log(fmt.Sprintf("[LAYOUT] Saving layout configuration for user: %s", config.UserID))
	err := d.layoutService.SaveLayout(config)
	if err != nil {
		d.log(fmt.Sprintf("[LAYOUT] Failed to save layout: %v", err))
		return WrapError("dashboard", "SaveLayout", err)
	}

	d.log("[LAYOUT] Layout configuration saved successfully")
	return nil
}

// LoadLayout 从数据库加载布局配置
func (d *DashboardFacadeService) LoadLayout(userID string) (*database.LayoutConfiguration, error) {
	if d.layoutService == nil {
		return nil, WrapError("dashboard", "LoadLayout", fmt.Errorf("layout service not initialized"))
	}

	d.log(fmt.Sprintf("[LAYOUT] Loading layout configuration for user: %s", userID))
	config, err := d.layoutService.LoadLayout(userID)
	if err != nil {
		// If no layout found, return default layout instead of error
		if err.Error() == fmt.Sprintf("no layout found for user: %s", userID) {
			d.log("[LAYOUT] No saved layout found, returning default layout")
			defaultConfig := d.layoutService.GetDefaultLayout()
			defaultConfig.UserID = userID
			return &defaultConfig, nil
		}

		d.log(fmt.Sprintf("[LAYOUT] Failed to load layout: %v", err))
		return nil, WrapError("dashboard", "LoadLayout", err)
	}

	d.log("[LAYOUT] Layout configuration loaded successfully")
	return config, nil
}

// --- Data Availability Methods ---

// CheckComponentHasData 检查组件是否有可用数据
func (d *DashboardFacadeService) CheckComponentHasData(componentType string, instanceID string) (bool, error) {
	if d.dataService == nil {
		return false, WrapError("dashboard", "CheckComponentHasData", fmt.Errorf("data service not initialized"))
	}

	d.log(fmt.Sprintf("[DATA] Checking data availability for component: %s (%s)", instanceID, componentType))
	hasData, err := d.dataService.CheckComponentHasData(componentType, instanceID)
	if err != nil {
		d.log(fmt.Sprintf("[DATA] Failed to check component data: %v", err))
		return false, WrapError("dashboard", "CheckComponentHasData", err)
	}

	d.log(fmt.Sprintf("[DATA] Component %s has data: %v", instanceID, hasData))
	return hasData, nil
}

// --- File Methods ---

// GetFilesByCategory 按类别获取文件列�?
func (d *DashboardFacadeService) GetFilesByCategory(category string) ([]database.FileInfo, error) {
	if d.fileService == nil {
		return nil, WrapError("dashboard", "GetFilesByCategory", fmt.Errorf("file service not initialized"))
	}

	// Convert string to FileCategory type
	var fileCategory database.FileCategory
	switch category {
	case "all_files":
		fileCategory = database.AllFiles
	case "user_request_related":
		fileCategory = database.UserRequestRelated
	default:
		return nil, WrapError("dashboard", "GetFilesByCategory", fmt.Errorf("invalid file category: %s", category))
	}

	d.log(fmt.Sprintf("[FILES] Getting files for category: %s", category))
	files, err := d.fileService.GetFilesByCategory(fileCategory)
	if err != nil {
		d.log(fmt.Sprintf("[FILES] Failed to get files: %v", err))
		return nil, WrapError("dashboard", "GetFilesByCategory", err)
	}

	d.log(fmt.Sprintf("[FILES] Retrieved %d files for category %s", len(files), category))
	return files, nil
}

// DownloadFile 返回文件下载路径
func (d *DashboardFacadeService) DownloadFile(fileID string) (string, error) {
	if d.fileService == nil {
		return "", WrapError("dashboard", "DownloadFile", fmt.Errorf("file service not initialized"))
	}

	d.log(fmt.Sprintf("[FILES] Downloading file: %s", fileID))
	filePath, err := d.fileService.DownloadFile(fileID)
	if err != nil {
		d.log(fmt.Sprintf("[FILES] Failed to download file: %v", err))
		return "", WrapError("dashboard", "DownloadFile", err)
	}

	d.log(fmt.Sprintf("[FILES] File download path: %s", filePath))
	return filePath, nil
}

// --- Export Methods ---

// ExportDashboard 导出仪表盘数据，支持组件过滤
func (d *DashboardFacadeService) ExportDashboard(req database.ExportRequest) (*database.ExportResult, error) {
	if d.exportService == nil {
		return nil, WrapError("dashboard", "ExportDashboard", fmt.Errorf("export service not initialized"))
	}

	d.log(fmt.Sprintf("[EXPORT] Exporting dashboard for user: %s, format: %s", req.UserID, req.Format))
	result, err := d.exportService.ExportDashboard(req)
	if err != nil {
		d.log(fmt.Sprintf("[EXPORT] Failed to export dashboard: %v", err))
		return nil, WrapError("dashboard", "ExportDashboard", err)
	}

	d.log(fmt.Sprintf("[EXPORT] Dashboard exported successfully: %s", result.FilePath))
	d.log(fmt.Sprintf("[EXPORT] Included components: %d, Excluded components: %d",
		len(result.IncludedComponents), len(result.ExcludedComponents)))
	return result, nil
}
