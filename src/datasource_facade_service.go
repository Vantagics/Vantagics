package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"vantagedata/agent"
	"vantagedata/config"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DataSourceManager 定义数据源管理接口
type DataSourceManager interface {
	GetDataSources() ([]agent.DataSource, error)
	AddDataSource(name, driverType string, config map[string]string) (*agent.DataSource, error)
	DeleteDataSource(id string) error
	RenameDataSource(id, newName string) error
	ImportExcelDataSource(name, filePath string) (*agent.DataSource, error)
	ImportCSVDataSource(name, dirPath string) (*agent.DataSource, error)
	ImportJSONDataSource(name, filePath string) (*agent.DataSource, error)
	GetDataSourceTables(id string) ([]string, error)
	GetDataSourceTableData(id, tableName string) ([]map[string]interface{}, error)
	GetDataSourceTableCount(id, tableName string) (int, error)
	GetDataSourceTableDataWithCount(id, tableName string) (*TableDataWithCount, error)
	DeleteTable(id, tableName string) error
	RenameColumn(id, tableName, oldColumnName, newColumnName string) error
	DeleteColumn(id, tableName, columnName string) error
	UpdateMySQLExportConfig(id, host, port, user, password, database string) error
	RefreshDataSource(id string) (*agent.RefreshResult, error)
	RefreshEcommerceDataSource(id string) (*agent.RefreshResult, error)
	IsEcommerceDataSource(dsType string) bool
	IsRefreshableDataSource(dsType string) bool
	GetDataSourceStatistics() (*agent.DataSourceStatistics, error)
	StartDataSourceAnalysis(dataSourceID string) (string, error)
	GetShopifyOAuthConfig() ShopifyOAuthConfig
	StartShopifyOAuth(shop string) (string, error)
	WaitForShopifyOAuth() (map[string]string, error)
	CancelShopifyOAuth()
	GetJiraProjects(instanceType, baseUrl, username, apiToken string) ([]JiraProject, error)
}

// DataSourceFacadeService 数据源服务门面，封装所有数据源相关的业务逻辑
type DataSourceFacadeService struct {
	ctx             context.Context
	dataSourceService *agent.DataSourceService
	configProvider    ConfigProvider
	chatService       *ChatService
	einoService       *agent.EinoService
	eventAggregator   *EventAggregator
	logger            func(string)

	// sendMessageFn is injected from App to allow import methods to call SendMessage
	sendMessageFn func(threadID, message, userMessageID, requestID string) (string, error)

	// Shopify OAuth state (module-level in original, moved here for encapsulation)
	shopifyOAuthService *agent.ShopifyOAuthService
	shopifyOAuthMutex   sync.Mutex
}

// NewDataSourceFacadeService 创建新的 DataSourceFacadeService 实例
func NewDataSourceFacadeService(
	dataSourceService *agent.DataSourceService,
	configProvider ConfigProvider,
	chatService *ChatService,
	einoService *agent.EinoService,
	eventAggregator *EventAggregator,
	logger func(string),
) *DataSourceFacadeService {
	return &DataSourceFacadeService{
		dataSourceService: dataSourceService,
		configProvider:    configProvider,
		chatService:       chatService,
		einoService:       einoService,
		eventAggregator:   eventAggregator,
		logger:            logger,
	}
}

// Name 返回服务名称
func (d *DataSourceFacadeService) Name() string {
	return "datasource"
}

// Initialize 初始化数据源门面服务
func (d *DataSourceFacadeService) Initialize(ctx context.Context) error {
	d.ctx = ctx
	if d.dataSourceService == nil {
		return WrapError("datasource", "Initialize", fmt.Errorf("dataSourceService dependency is nil"))
	}
	d.log("DataSourceFacadeService initialized")
	return nil
}

// Shutdown 关闭数据源门面服务
func (d *DataSourceFacadeService) Shutdown() error {
	// Cancel any active Shopify OAuth
	d.shopifyOAuthMutex.Lock()
	if d.shopifyOAuthService != nil {
		d.shopifyOAuthService.StopCallbackServer()
		d.shopifyOAuthService = nil
	}
	d.shopifyOAuthMutex.Unlock()
	d.log("DataSourceFacadeService shutdown")
	return nil
}

// SetContext sets the Wails runtime context
func (d *DataSourceFacadeService) SetContext(ctx context.Context) {
	d.ctx = ctx
}

// SetSendMessageFn injects the SendMessage function from App
func (d *DataSourceFacadeService) SetSendMessageFn(fn func(threadID, message, userMessageID, requestID string) (string, error)) {
	d.sendMessageFn = fn
}

// SetEinoService updates the EinoService reference (used during reinitializeServices)
func (d *DataSourceFacadeService) SetEinoService(es *agent.EinoService) {
	d.einoService = es
}

// --- Data Source CRUD ---

// GetDataSources returns the list of registered data sources
func (d *DataSourceFacadeService) GetDataSources() ([]agent.DataSource, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "GetDataSources", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.LoadDataSources()
}

// AddDataSource adds a new data source with generic configuration
func (d *DataSourceFacadeService) AddDataSource(name, driverType string, dsConfigMap map[string]string) (*agent.DataSource, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "AddDataSource", fmt.Errorf("data source service not initialized"))
	}

	dsConfig := agent.DataSourceConfig{
		OriginalFile:           dsConfigMap["filePath"],
		Host:                   dsConfigMap["host"],
		Port:                   dsConfigMap["port"],
		User:                   dsConfigMap["user"],
		Password:               dsConfigMap["password"],
		Database:               dsConfigMap["database"],
		StoreLocally:           dsConfigMap["storeLocally"] == "true",
		ShopifyStore:           dsConfigMap["shopifyStore"],
		ShopifyAccessToken:     dsConfigMap["shopifyAccessToken"],
		ShopifyAPIVersion:      dsConfigMap["shopifyAPIVersion"],
		BigCommerceStoreHash:   dsConfigMap["bigcommerceStoreHash"],
		BigCommerceAccessToken: dsConfigMap["bigcommerceAccessToken"],
		EbayAccessToken:        dsConfigMap["ebayAccessToken"],
		EbayEnvironment:        dsConfigMap["ebayEnvironment"],
		EbayApiFulfillment:     dsConfigMap["ebayApiFulfillment"] != "false",
		EbayApiFinances:        dsConfigMap["ebayApiFinances"] != "false",
		EbayApiAnalytics:       dsConfigMap["ebayApiAnalytics"] != "false",
		EtsyShopId:             dsConfigMap["etsyShopId"],
		EtsyAccessToken:        dsConfigMap["etsyAccessToken"],
		JiraInstanceType:       dsConfigMap["jiraInstanceType"],
		JiraBaseUrl:            dsConfigMap["jiraBaseUrl"],
		JiraUsername:           dsConfigMap["jiraUsername"],
		JiraApiToken:           dsConfigMap["jiraApiToken"],
		JiraProjectKey:         dsConfigMap["jiraProjectKey"],
		// Snowflake configuration
		SnowflakeAccount:   dsConfigMap["snowflakeAccount"],
		SnowflakeUser:      dsConfigMap["snowflakeUser"],
		SnowflakePassword:  dsConfigMap["snowflakePassword"],
		SnowflakeWarehouse: dsConfigMap["snowflakeWarehouse"],
		SnowflakeDatabase:  dsConfigMap["snowflakeDatabase"],
		SnowflakeSchema:    dsConfigMap["snowflakeSchema"],
		SnowflakeRole:      dsConfigMap["snowflakeRole"],
		// BigQuery configuration
		BigQueryProjectID:   dsConfigMap["bigqueryProjectId"],
		BigQueryDatasetID:   dsConfigMap["bigqueryDatasetId"],
		BigQueryCredentials: dsConfigMap["bigqueryCredentials"],
		// Financial data source configuration fields
		FinancialProvider:    dsConfigMap["financialProvider"],
		FinancialAPIKey:      dsConfigMap["financialApiKey"],
		FinancialAPISecret:   dsConfigMap["financialApiSecret"],
		FinancialToken:       dsConfigMap["financialToken"],
		FinancialUsername:    dsConfigMap["financialUsername"],
		FinancialPassword:    dsConfigMap["financialPassword"],
		FinancialDatasets:    dsConfigMap["financialDatasets"],
		FinancialSymbols:     dsConfigMap["financialSymbols"],
		FinancialDataType:    dsConfigMap["financialDataType"],
		FinancialDatasetCode: dsConfigMap["financialDatasetCode"],
		FinancialCertPath:    dsConfigMap["financialCertPath"],
		FinancialEnvironment: dsConfigMap["financialEnvironment"],
	}

	headerGen := func(prompt string) (string, error) {
		if d.sendMessageFn != nil {
			return d.sendMessageFn("", prompt, "", "")
		}
		return "", fmt.Errorf("sendMessage function not available")
	}

	ds, err := d.dataSourceService.ImportDataSource(name, driverType, dsConfig, headerGen)
	if err == nil && ds != nil {
		go d.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// DeleteDataSource deletes a data source
func (d *DataSourceFacadeService) DeleteDataSource(id string) error {
	if d.dataSourceService == nil {
		return WrapError("datasource", "DeleteDataSource", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.DeleteDataSource(id)
}

// RenameDataSource renames a data source
func (d *DataSourceFacadeService) RenameDataSource(id, newName string) error {
	if d.dataSourceService == nil {
		return WrapError("datasource", "RenameDataSource", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.RenameDataSource(id, newName)
}

// --- Import Methods ---

// ImportExcelDataSource imports an Excel file as a data source
func (d *DataSourceFacadeService) ImportExcelDataSource(name, filePath string) (*agent.DataSource, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "ImportExcelDataSource", fmt.Errorf("data source service not initialized"))
	}

	headerGen := func(prompt string) (string, error) {
		if d.sendMessageFn != nil {
			return d.sendMessageFn("", prompt, "", "")
		}
		return "", fmt.Errorf("sendMessage function not available")
	}

	ds, err := d.dataSourceService.ImportExcel(name, filePath, headerGen)
	if err == nil && ds != nil {
		go d.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// ImportCSVDataSource imports a CSV directory as a data source
func (d *DataSourceFacadeService) ImportCSVDataSource(name, dirPath string) (*agent.DataSource, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "ImportCSVDataSource", fmt.Errorf("data source service not initialized"))
	}

	headerGen := func(prompt string) (string, error) {
		if d.sendMessageFn != nil {
			return d.sendMessageFn("", prompt, "", "")
		}
		return "", fmt.Errorf("sendMessage function not available")
	}

	ds, err := d.dataSourceService.ImportCSV(name, dirPath, headerGen)
	if err == nil && ds != nil {
		go d.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// ImportJSONDataSource imports a JSON file as a data source
func (d *DataSourceFacadeService) ImportJSONDataSource(name, filePath string) (*agent.DataSource, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "ImportJSONDataSource", fmt.Errorf("data source service not initialized"))
	}

	headerGen := func(prompt string) (string, error) {
		if d.sendMessageFn != nil {
			return d.sendMessageFn("", prompt, "", "")
		}
		return "", fmt.Errorf("sendMessage function not available")
	}

	ds, err := d.dataSourceService.ImportJSON(name, filePath, headerGen)
	if err == nil && ds != nil {
		go d.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// --- Table Operations ---

// GetDataSourceTables returns all table names for a data source
func (d *DataSourceFacadeService) GetDataSourceTables(id string) ([]string, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "GetDataSourceTables", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.GetDataSourceTables(id)
}

// GetDataSourceTableData returns preview data for a table
func (d *DataSourceFacadeService) GetDataSourceTableData(id, tableName string) ([]map[string]interface{}, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "GetDataSourceTableData", fmt.Errorf("data source service not initialized"))
	}
	cfg, err := d.configProvider.GetConfig()
	if err != nil {
		return nil, err
	}
	return d.dataSourceService.GetDataSourceTableData(id, tableName, cfg.MaxPreviewRows)
}

// GetDataSourceTableCount returns the total number of rows in a table
func (d *DataSourceFacadeService) GetDataSourceTableCount(id, tableName string) (int, error) {
	if d.dataSourceService == nil {
		return 0, WrapError("datasource", "GetDataSourceTableCount", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.GetDataSourceTableCount(id, tableName)
}

// GetDataSourceTableDataWithCount returns preview data and row count in a single DB connection
func (d *DataSourceFacadeService) GetDataSourceTableDataWithCount(id, tableName string) (*TableDataWithCount, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "GetDataSourceTableDataWithCount", fmt.Errorf("data source service not initialized"))
	}
	cfg, err := d.configProvider.GetConfig()
	if err != nil {
		return nil, err
	}
	data, count, err := d.dataSourceService.GetDataSourceTableDataWithCount(id, tableName, cfg.MaxPreviewRows)
	if err != nil {
		d.log(fmt.Sprintf("[DataBrowser] Failed to load table data for %s.%s: %v", id, tableName, err))
		return nil, err
	}
	return &TableDataWithCount{Data: data, RowCount: count}, nil
}

// DeleteTable removes a table from a data source
func (d *DataSourceFacadeService) DeleteTable(id, tableName string) error {
	if d.dataSourceService == nil {
		return WrapError("datasource", "DeleteTable", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.DeleteTable(id, tableName)
}

// RenameColumn renames a column in a table
func (d *DataSourceFacadeService) RenameColumn(id, tableName, oldColumnName, newColumnName string) error {
	if d.dataSourceService == nil {
		return WrapError("datasource", "RenameColumn", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.RenameColumn(id, tableName, oldColumnName, newColumnName)
}

// DeleteColumn deletes a column from a table
func (d *DataSourceFacadeService) DeleteColumn(id, tableName, columnName string) error {
	if d.dataSourceService == nil {
		return WrapError("datasource", "DeleteColumn", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.DeleteColumn(id, tableName, columnName)
}

// UpdateMySQLExportConfig updates the MySQL export configuration for a data source
func (d *DataSourceFacadeService) UpdateMySQLExportConfig(id, host, port, user, password, database string) error {
	if d.dataSourceService == nil {
		return WrapError("datasource", "UpdateMySQLExportConfig", fmt.Errorf("data source service not initialized"))
	}
	mysqlConfig := agent.MySQLExportConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
	}
	return d.dataSourceService.UpdateMySQLExportConfig(id, mysqlConfig)
}

// --- Refresh Methods ---

// RefreshDataSource performs incremental update for supported data sources
func (d *DataSourceFacadeService) RefreshDataSource(id string) (*agent.RefreshResult, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "RefreshDataSource", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.RefreshDataSource(id)
}

// RefreshEcommerceDataSource performs incremental update for e-commerce data sources
func (d *DataSourceFacadeService) RefreshEcommerceDataSource(id string) (*agent.RefreshResult, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "RefreshEcommerceDataSource", fmt.Errorf("data source service not initialized"))
	}
	return d.dataSourceService.RefreshEcommerceDataSource(id)
}

// IsEcommerceDataSource checks if a data source type supports incremental refresh
func (d *DataSourceFacadeService) IsEcommerceDataSource(dsType string) bool {
	if d.dataSourceService == nil {
		return false
	}
	return d.dataSourceService.IsEcommerceDataSource(dsType)
}

// IsRefreshableDataSource checks if a data source type supports incremental refresh
func (d *DataSourceFacadeService) IsRefreshableDataSource(dsType string) bool {
	if d.dataSourceService == nil {
		return false
	}
	return d.dataSourceService.IsRefreshableDataSource(dsType)
}

// --- Statistics & Analysis ---

// GetDataSourceStatistics returns aggregated statistics about all data sources
func (d *DataSourceFacadeService) GetDataSourceStatistics() (*agent.DataSourceStatistics, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "GetDataSourceStatistics", fmt.Errorf("data source service not initialized"))
	}

	dataSources, err := d.dataSourceService.LoadDataSources()
	if err != nil {
		return nil, fmt.Errorf("failed to load data sources: %w", err)
	}

	stats := &agent.DataSourceStatistics{
		TotalCount:      len(dataSources),
		BreakdownByType: make(map[string]int),
		DataSources:     make([]agent.DataSourceSummary, 0, len(dataSources)),
	}

	for _, ds := range dataSources {
		stats.BreakdownByType[ds.Type]++
		stats.DataSources = append(stats.DataSources, agent.DataSourceSummary{
			ID:   ds.ID,
			Name: ds.Name,
			Type: ds.Type,
		})
	}

	return stats, nil
}

// StartDataSourceAnalysis initiates analysis for a specific data source
// Returns the analysis session/thread ID
func (d *DataSourceFacadeService) StartDataSourceAnalysis(dataSourceID string) (string, error) {
	if d.dataSourceService == nil {
		return "", WrapError("datasource", "StartDataSourceAnalysis", fmt.Errorf("data source service not initialized"))
	}

	if d.chatService == nil {
		return "", WrapError("datasource", "StartDataSourceAnalysis", fmt.Errorf("chat service not initialized"))
	}

	// Validate data source exists
	dataSources, err := d.dataSourceService.LoadDataSources()
	if err != nil {
		return "", fmt.Errorf("failed to load data sources: %w", err)
	}

	var targetDS *agent.DataSource
	for i := range dataSources {
		if dataSources[i].ID == dataSourceID {
			targetDS = &dataSources[i]
			break
		}
	}

	if targetDS == nil {
		return "", fmt.Errorf("data source not found: %s", dataSourceID)
	}

	sessionTitle := fmt.Sprintf("分析: %s", targetDS.Name)
	thread, err := d.chatService.CreateThread(dataSourceID, sessionTitle)
	if err != nil {
		return "", fmt.Errorf("failed to create chat thread: %w", err)
	}

	threadID := thread.ID

	prompt := fmt.Sprintf("请分析数据源 '%s' (%s)，提供数据概览、关键指标和洞察。",
		targetDS.Name, targetDS.Type)

	userMessageID := fmt.Sprintf("ds-msg-%d", time.Now().UnixNano())

	d.log(fmt.Sprintf("[DATASOURCE-ANALYSIS] Starting analysis for %s (thread: %s, msgId: %s)",
		dataSourceID, threadID, userMessageID))

	// Emit event to notify frontend that analysis is starting
	runtime.EventsEmit(d.ctx, "chat-loading", map[string]interface{}{
		"loading":  true,
		"threadId": threadID,
	})

	// Notify frontend that a new analysis thread was created
	runtime.EventsEmit(d.ctx, "analysis-session-created", map[string]interface{}{
		"threadId":       threadID,
		"dataSourceId":   dataSourceID,
		"dataSourceName": targetDS.Name,
		"title":          sessionTitle,
	})

	// Call SendMessage asynchronously so we can return the threadID immediately
	go func() {
		defer func() {
			if r := recover(); r != nil {
				d.log(fmt.Sprintf("[PANIC] Recovered in async SendMessage goroutine: %v", r))
			}
		}()
		if d.sendMessageFn != nil {
			_, err := d.sendMessageFn(threadID, prompt, userMessageID, "")
			if err != nil {
				d.log(fmt.Sprintf("[DATASOURCE-ANALYSIS] Error: %v", err))
				runtime.EventsEmit(d.ctx, "analysis-error", map[string]interface{}{
					"threadId": threadID,
					"message":  err.Error(),
					"code":     "ANALYSIS_ERROR",
				})
			}
		}
	}()

	return threadID, nil
}

// analyzeDataSource performs background analysis on a data source
func (d *DataSourceFacadeService) analyzeDataSource(dataSourceID string) {
	startTotal := time.Now()
	if d.dataSourceService == nil {
		return
	}

	d.log(fmt.Sprintf("Starting analysis for source %s", dataSourceID))

	// 1. Get Tables
	startTables := time.Now()
	tables, err := d.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		d.log(fmt.Sprintf("Failed to get tables: %v", err))
		return
	}
	d.log(fmt.Sprintf("[TIMING] Getting tables took: %v", time.Since(startTables)))

	// 2. Sample Data & Construct Prompt
	startSample := time.Now()
	cfg, _ := d.configProvider.GetEffectiveConfig()
	langPrompt := getLangPrompt(cfg)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("I am starting a new analysis on this database. Based on the following schema and first row of data, please provide exactly two sentences in %s: the first sentence should describe the industry background of this data, and the second sentence should provide a concise overview of the data source content.\n\n", langPrompt))

	var tableSchemas []agent.TableSchema

	for _, tableName := range tables {
		sb.WriteString(fmt.Sprintf("Table: %s\n", tableName))

		data, err := d.dataSourceService.GetDataSourceTableData(dataSourceID, tableName, 1)
		if err != nil {
			sb.WriteString("(Failed to fetch data)\n")
			continue
		}

		var cols []string
		if len(data) > 0 {
			for k := range data[0] {
				cols = append(cols, k)
			}
			sb.WriteString(fmt.Sprintf("Columns: %s\nData:\n", strings.Join(cols, ", ")))

			for _, row := range data {
				var vals []string
				for _, col := range cols {
					if val, ok := row[col]; ok {
						vals = append(vals, fmt.Sprintf("%v", val))
					} else {
						vals = append(vals, "NULL")
					}
				}
				sb.WriteString(fmt.Sprintf("[%s]\n", strings.Join(vals, ", ")))
			}
		} else {
			sb.WriteString("(Empty table)\n")
		}
		sb.WriteString("\n")

		if len(cols) > 0 {
			tableSchemas = append(tableSchemas, agent.TableSchema{
				TableName: tableName,
				Columns:   cols,
			})
		}
	}
	d.log(fmt.Sprintf("[TIMING] Data sampling and prompt construction took: %v", time.Since(startSample)))

	// 3. Call LLM
	prompt := sb.String()

	if cfg.DetailedLog {
		d.log("Sending Analysis Prompt to LLM...")
	}

	llm := agent.NewLLMService(cfg, d.log)
	startLLM := time.Now()
	description, err := llm.Chat(context.Background(), prompt)
	d.log(fmt.Sprintf("[TIMING] Background LLM Analysis took: %v", time.Since(startLLM)))

	if err != nil {
		d.log(fmt.Sprintf("LLM Analysis failed: %v", err))
		return
	}

	if description == "" {
		d.log("LLM returned empty response during analysis.")
		description = "No description provided by LLM."
	}

	// 4. Save Analysis to DataSource
	startSave := time.Now()
	analysis := agent.DataSourceAnalysis{
		Summary: description,
		Schema:  tableSchemas,
	}

	if err := d.dataSourceService.UpdateAnalysis(dataSourceID, analysis); err != nil {
		d.log(fmt.Sprintf("Failed to update data source analysis: %v", err))
		return
	}
	d.log(fmt.Sprintf("[TIMING] Saving analysis result took: %v", time.Since(startSave)))
	d.log(fmt.Sprintf("[TIMING] Total Background Analysis took: %v", time.Since(startTotal)))

	d.log("Data Source Analysis complete and saved.")
}

// --- Shopify OAuth ---

// GetShopifyOAuthConfig returns the Shopify OAuth configuration
func (d *DataSourceFacadeService) GetShopifyOAuthConfig() ShopifyOAuthConfig {
	cfg, _ := d.configProvider.GetConfig()
	return ShopifyOAuthConfig{
		ClientID:     cfg.ShopifyClientID,
		ClientSecret: cfg.ShopifyClientSecret,
	}
}

// StartShopifyOAuth initiates the Shopify OAuth flow
func (d *DataSourceFacadeService) StartShopifyOAuth(shop string) (string, error) {
	d.shopifyOAuthMutex.Lock()
	defer d.shopifyOAuthMutex.Unlock()

	cfg, err := d.configProvider.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %v", err)
	}

	if cfg.ShopifyClientID == "" || cfg.ShopifyClientSecret == "" {
		return "", fmt.Errorf("Shopify OAuth not configured. Please set Client ID and Client Secret in settings.")
	}

	oauthConfig := agent.ShopifyOAuthConfig{
		ClientID:     cfg.ShopifyClientID,
		ClientSecret: cfg.ShopifyClientSecret,
		Scopes:       "read_orders,read_products,read_customers,read_inventory",
	}
	d.shopifyOAuthService = agent.NewShopifyOAuthService(oauthConfig, d.log)

	authURL, _, err := d.shopifyOAuthService.GetAuthURL(shop)
	if err != nil {
		return "", err
	}

	if err := d.shopifyOAuthService.StartCallbackServer(d.ctx); err != nil {
		return "", err
	}

	d.log(fmt.Sprintf("[SHOPIFY-OAUTH] Started OAuth flow for shop: %s", shop))
	return authURL, nil
}

// WaitForShopifyOAuth waits for the OAuth flow to complete
func (d *DataSourceFacadeService) WaitForShopifyOAuth() (map[string]string, error) {
	d.shopifyOAuthMutex.Lock()
	service := d.shopifyOAuthService
	d.shopifyOAuthMutex.Unlock()

	if service == nil {
		return nil, fmt.Errorf("OAuth flow not started")
	}

	result := service.WaitForResult(5 * time.Minute)

	service.StopCallbackServer()

	d.shopifyOAuthMutex.Lock()
	d.shopifyOAuthService = nil
	d.shopifyOAuthMutex.Unlock()

	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}

	return map[string]string{
		"accessToken": result.AccessToken,
		"shop":        result.Shop,
		"scope":       result.Scope,
	}, nil
}

// CancelShopifyOAuth cancels the ongoing OAuth flow
func (d *DataSourceFacadeService) CancelShopifyOAuth() {
	d.shopifyOAuthMutex.Lock()
	defer d.shopifyOAuthMutex.Unlock()

	if d.shopifyOAuthService != nil {
		d.shopifyOAuthService.StopCallbackServer()
		d.shopifyOAuthService = nil
		d.log("[SHOPIFY-OAUTH] OAuth flow cancelled")
	}
}

// OpenShopifyOAuthInBrowser opens the Shopify OAuth URL in the default browser
func (d *DataSourceFacadeService) OpenShopifyOAuthInBrowser(url string) {
	runtime.BrowserOpenURL(d.ctx, url)
}

// --- Jira ---

// GetJiraProjects fetches available projects from Jira using provided credentials
func (d *DataSourceFacadeService) GetJiraProjects(instanceType, baseUrl, username, apiToken string) ([]JiraProject, error) {
	if d.dataSourceService == nil {
		return nil, WrapError("datasource", "GetJiraProjects", fmt.Errorf("data source service not initialized"))
	}
	projects, err := d.dataSourceService.GetJiraProjects(instanceType, baseUrl, username, apiToken)
	if err != nil {
		return nil, err
	}
	result := make([]JiraProject, len(projects))
	for i, p := range projects {
		result[i] = JiraProject{
			Key:  p.Key,
			Name: p.Name,
			ID:   p.ID,
		}
	}
	return result, nil
}

// --- Analysis Suggestions (background) ---

// GenerateAnalysisSuggestions generates analysis suggestions for a data source
func (d *DataSourceFacadeService) GenerateAnalysisSuggestions(threadID string, analysis *agent.DataSourceAnalysis) {
	if d.chatService == nil {
		return
	}

	// Notify frontend that background task started
	runtime.EventsEmit(d.ctx, "chat-loading", map[string]interface{}{
		"loading":  true,
		"threadId": threadID,
	})
	defer runtime.EventsEmit(d.ctx, "chat-loading", map[string]interface{}{
		"loading":  false,
		"threadId": threadID,
	})

	cfg, _ := d.configProvider.GetEffectiveConfig()
	langPrompt := getLangPrompt(cfg)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Based on the following data source summary and schema, please suggest 3-5 distinct business analysis questions that would provide valuable insights for decision-making. Please answer in %s.\n\nIMPORTANT GUIDELINES:\n- Focus on BUSINESS VALUE and INSIGHTS, not technical implementation\n- Use simple, non-technical language that any business user can understand\n- Frame suggestions as business questions or outcomes (e.g., \"Understand customer purchasing patterns\" instead of \"Run RFM analysis\")\n- DO NOT mention SQL, Python, data processing, or any technical terms\n- Focus on what insights can be discovered, not how to discover them\n\nProvide the suggestions as a clear, structured, numbered list (1., 2., 3...). Each suggestion should include:\n- A clear, business-focused title\n- A one-sentence description of what business insights this would reveal\n\nEnd your response by telling the user (in %s) that they can select one or more analysis questions by replying with the corresponding number(s).", langPrompt, langPrompt))
	sb.WriteString(fmt.Sprintf("Summary: %s\n\n", analysis.Summary))
	sb.WriteString("Schema:\n")
	for _, table := range analysis.Schema {
		sb.WriteString(fmt.Sprintf("- Table: %s, Columns: %s\n", table.TableName, strings.Join(table.Columns, ", ")))
	}

	prompt := sb.String()
	llm := agent.NewLLMService(cfg, d.log)

	resp, err := llm.Chat(context.Background(), prompt)
	if err != nil {
		d.log(fmt.Sprintf("Failed to generate suggestions: %v", err))
		return
	}

	msg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   resp,
		Timestamp: time.Now().Unix(),
	}

	if err := d.chatService.AddMessage(threadID, msg); err != nil {
		d.log(fmt.Sprintf("Failed to add suggestion message: %v", err))
		return
	}

	insights := d.parseSuggestionsToInsights(resp, "", "")
	if len(insights) > 0 {
		d.log(fmt.Sprintf("Emitting %d suggestions to dashboard insights", len(insights)))
		if d.eventAggregator != nil {
			for _, insight := range insights {
				d.eventAggregator.AddInsight(threadID, msg.ID, "", insight)
			}
			d.eventAggregator.FlushNow(threadID, true)
		}
	}

	runtime.EventsEmit(d.ctx, "thread-updated", threadID)
}

// parseSuggestionsToInsights extracts numbered suggestions from LLM response and converts to Insight objects
func (d *DataSourceFacadeService) parseSuggestionsToInsights(llmResponse, dataSourceID, dataSourceName string) []Insight {
	var insights []Insight
	lines := strings.Split(llmResponse, "\n")

	numberPattern := regexp.MustCompile(`^\s*\*{0,2}(\d+)[.、)]\*{0,2}\s*(.+)`)
	listPattern := regexp.MustCompile(`^\s*[-*•]\s+(.+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		var suggestionText string
		if matches := numberPattern.FindStringSubmatch(line); len(matches) > 2 {
			suggestionText = strings.TrimSpace(matches[2])
		} else if matches := listPattern.FindStringSubmatch(line); len(matches) > 1 {
			suggestionText = strings.TrimSpace(matches[1])
		}
		if suggestionText != "" {
			suggestionText = strings.TrimPrefix(suggestionText, "**")
			suggestionText = strings.TrimSuffix(suggestionText, "**")
			suggestionText = strings.TrimSpace(suggestionText)
		}
		if suggestionText != "" {
			insights = append(insights, Insight{
				Text:         suggestionText,
				Icon:         "lightbulb",
				DataSourceID: dataSourceID,
				SourceName:   dataSourceName,
			})
		}
	}

	return insights
}

// --- Helper ---

// log writes a log message using the configured logger
func (d *DataSourceFacadeService) log(msg string) {
	if d.logger != nil {
		d.logger(msg)
	}
}

// getLangPrompt returns the language prompt string based on config
func getLangPrompt(cfg config.Config) string {
	if cfg.Language == "简体中文" {
		return "Simplified Chinese"
	}
	return "English"
}
