package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"vantagics/agent"
	"vantagics/config"
	"vantagics/database"
	"vantagics/i18n"
	"vantagics/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Metric structure for dashboard
type Metric struct {
	Title  string `json:"title"`
	Value  string `json:"value"`
	Change string `json:"change"`
}

// Insight structure for dashboard
type Insight struct {
	Text         string `json:"text"`
	Icon         string `json:"icon"`
	DataSourceID string `json:"data_source_id,omitempty"`
	SourceName   string `json:"source_name,omitempty"`
}

// IntentSuggestion represents a possible interpretation of user's intent
// 保持与现有结构完全一致，确保向后兼容
// Validates: Requirements 7.4
type IntentSuggestion struct {
	ID          string `json:"id"`          // Unique identifier
	Title       string `json:"title"`       // Short title (10 chars max)
	Description string `json:"description"` // Detailed description (30 chars max)
	Icon        string `json:"icon"`        // Icon (emoji or icon name)
	Query       string `json:"query"`       // Actual query/analysis request to execute
}

// IsValid 检查意图建议是否有�
// 验证所有必需字段都非�
// Returns true if all required fields (ID, Title, Description, Icon, Query) are non-empty
// Validates: Requirements 1.2 (意图建议结构完整�
func (s *IntentSuggestion) IsValid() bool {
	return s.ID != "" &&
		s.Title != "" &&
		s.Description != "" &&
		s.Icon != "" &&
		s.Query != ""
}

// Clone 创建意图建议的深拷贝
// Returns a new IntentSuggestion with the same values
// Useful for avoiding unintended modifications to the original
func (s *IntentSuggestion) Clone() *IntentSuggestion {
	if s == nil {
		return nil
	}
	return &IntentSuggestion{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		Icon:        s.Icon,
		Query:       s.Query,
	}
}

// String 返回意图建议的字符串表示
// Format: "[Icon] Title: Description (ID: xxx)"
// Useful for logging and debugging
func (s *IntentSuggestion) String() string {
	if s == nil {
		return "<nil IntentSuggestion>"
	}
	return fmt.Sprintf("[%s] %s: %s (ID: %s)", s.Icon, s.Title, s.Description, s.ID)
}

// DashboardData structure
type DashboardData struct {
	Metrics  []Metric  `json:"metrics"`
	Insights []Insight `json:"insights"`
}

// contextKey is a typed key for context.WithValue to avoid collisions (Go best practice).
type contextKey string

const appContextKey contextKey = "app"

// mapLicenseLLMType maps license server LLM type strings to the canonical provider names
// used throughout the application. This centralizes the mapping to avoid duplication.
func mapLicenseLLMType(llmType, baseURL string) (mappedType, mappedBaseURL string) {
	mappedBaseURL = baseURL
	switch strings.ToLower(llmType) {
	case "openai":
		mappedType = "OpenAI"
	case "anthropic":
		mappedType = "Anthropic"
	case "gemini":
		mappedType = "Gemini"
	case "deepseek":
		mappedType = "OpenAI-Compatible"
		if mappedBaseURL == "" {
			mappedBaseURL = "https://api.deepseek.com"
		}
	case "openai-compatible":
		mappedType = "OpenAI-Compatible"
	case "claude-compatible":
		mappedType = "Claude-Compatible"
	default:
		mappedType = llmType
	}
	return
}

// applyActivatedLLMConfig merges activated license LLM settings into the given config.
// Returns true if the config was modified.
func applyActivatedLLMConfig(cfg *config.Config, activationData *agent.ActivationData) bool {
	if activationData == nil || activationData.LLMAPIKey == "" {
		return false
	}
	llmType, baseURL := mapLicenseLLMType(activationData.LLMType, activationData.LLMBaseURL)
	cfg.LLMProvider = llmType
	cfg.APIKey = activationData.LLMAPIKey
	cfg.BaseURL = baseURL
	cfg.ModelName = activationData.LLMModel
	return true
}

// App struct
type App struct {
	ctx                      context.Context
	registry                 *ServiceRegistry
	configService            *ConfigService
	chatFacadeService        *ChatFacadeService
	dataSourceFacadeService  *DataSourceFacadeService
	analysisFacadeService    *AnalysisFacadeService
	exportFacadeService      *ExportFacadeService
	dashboardFacadeService   *DashboardFacadeService
	licenseFacadeService     *LicenseFacadeService
	marketplaceFacadeService *MarketplaceFacadeService
	skillFacadeService       *SkillFacadeService
	pythonFacadeService      *PythonFacadeService
	connectionTestService    *ConnectionTestService
	chatService              *ChatService
	pythonService            *agent.PythonService
	dataSourceService        *agent.DataSourceService
	memoryService            *agent.MemoryService
	workingContextManager    *agent.WorkingContextManager
	analysisPathManager      *agent.AnalysisPathManager
	einoService              *agent.EinoService
	searchKeywordsManager    *agent.SearchKeywordsManager
	storageDir               string
	logger                   *logger.Logger
	// Event aggregator for analysis results
	eventAggregator *EventAggregator
	// License client for activation
	licenseClient           *agent.LicenseClient
	licenseActivationFailed bool   // True if license activation/refresh failed
	licenseActivationError  string // Error message to show user
	// Usage license store for local billing enforcement
	usageLicenseStore *UsageLicenseStore
	// Pack passwords from marketplace downloads (filePath -> encryption password)
	packPasswords map[string]string
	// Persistent pack password store (survives app restarts)
	packPasswordStore *PackPasswordStore
	// startupDone is closed when startup() finishes, used by background goroutines
	// to wait for full initialization before calling SaveConfig/reinitializeServices.
	startupDone chan struct{}
}

// AgentMemoryView structure for frontend
type AgentMemoryView struct {
	LongTerm   []string `json:"long_term"`
	MediumTerm []string `json:"medium_term"`
	ShortTerm  []string `json:"short_term"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	l := logger.NewLogger()
	ps := agent.NewPythonService()
	return &App{
		configService:       NewConfigService(l.Log),
		pythonService:       ps,
		pythonFacadeService: NewPythonFacadeService(ps, l.Log),
		logger:              l,
		packPasswords:       make(map[string]string),
		startupDone:         make(chan struct{}),
	}
}

// SetChatOpen updates the chat open state
func (a *App) SetChatOpen(isOpen bool) {
	if a.chatFacadeService == nil {
		return
	}
	a.chatFacadeService.SetChatOpen(isOpen)
}

// ShowAbout displays the about dialog
func (a *App) ShowAbout() {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   i18n.T("app.about_title"),
		Message: i18n.T("app.about_message"),
	})
}

// OpenDevTools opens the developer tools/console
func (a *App) OpenDevTools() {
	// Wails v2 doesn't have direct API to open DevTools
	// Show instructions to the user
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   i18n.T("app.devtools_title"),
		Message: i18n.T("app.devtools_message"),
	})
}

// tryLicenseActivationWithRetry attempts license activation with exponential backoff retry.
// operation should be "Activation" or "Refresh" for logging purposes.
func (a *App) tryLicenseActivationWithRetry(cfg *config.Config, operation string) {
	if a.licenseFacadeService == nil {
		a.Log("[STARTUP] License facade service not initialized, skipping license activation")
		return
	}
	a.licenseFacadeService.tryLicenseActivationWithRetry(cfg, operation)
	// Sync activation failure state back to App for startup flow
	a.licenseActivationFailed = a.licenseFacadeService.CheckLicenseActivationFailed()
	a.licenseActivationError = a.licenseFacadeService.GetLicenseActivationError()
}

// onBeforeClose is called when the application is about to close
func (a *App) onBeforeClose(ctx context.Context) (prevent bool) {
	// Check if cancellation was requested - if so, wait a moment for cleanup
	if a.chatFacadeService != nil && a.chatFacadeService.IsCancelRequested() {
		// Wait briefly for the cancellation to complete
		a.Log("[CLOSE-DIALOG] Cancel was requested, waiting for cleanup...")
		time.Sleep(500 * time.Millisecond)
	}

	// Only prevent close if there's an active analysis running
	hasActiveAnalysis := a.chatFacadeService != nil && a.chatFacadeService.HasActiveAnalysis()

	if hasActiveAnalysis {
		title := i18n.T("app.confirm_exit_title")
		message := i18n.T("app.confirm_exit_message")
		yesButton := i18n.T("app.exit_button")
		noButton := i18n.T("app.cancel_button")

		dialog, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         title,
			Message:       message,
			Buttons:       []string{noButton, yesButton}, // 取消按钮在前，退出按钮在�
			DefaultButton: noButton,
			CancelButton:  noButton,
		})

		if err != nil {
			// 如果对话框出错，阻止关闭以保护用户数�
			a.Log(fmt.Sprintf("[CLOSE-DIALOG] Error showing dialog: %v", err))
			return true
		}

		// Log the dialog result for debugging
		a.Log(fmt.Sprintf("[CLOSE-DIALOG] User clicked: '%s' (yesButton='%s', noButton='%s')", dialog, yesButton, noButton))

		// Windows may return standard button values instead of custom text
		// Check for both custom button text and standard Windows values
		// Allow close only if user explicitly clicked "Exit" button
		// Standard Windows values for "Yes" button: "Yes", "OK", "Ok"
		if dialog == yesButton || dialog == "Yes" || dialog == "OK" || dialog == "Ok" {
			a.Log("[CLOSE-DIALOG] Allowing application to close")
			return false // 允许关闭
		}
		a.Log("[CLOSE-DIALOG] Preventing application close")
		return true // 阻止关闭 (user clicked Cancel/No or closed dialog)
	}
	return false // 没有分析任务，允许关�
}

// shutdown is called when the application is closing to clean up resources
func (a *App) shutdown(ctx context.Context) {
	// Final credits usage report before shutdown (synchronous, with timeout)
	if a.licenseClient != nil && a.licenseClient.IsCreditsMode() && a.licenseClient.GetTrustLevel() == "low" && a.licenseClient.ShouldReportNow() {
		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					a.Log(fmt.Sprintf("[SHUTDOWN] Panic in final usage report: %v", r))
				}
				close(done)
			}()
			if err := a.licenseClient.ReportUsage(); err != nil {
				a.Log(fmt.Sprintf("[SHUTDOWN] Final usage report failed: %v", err))
			} else {
				a.Log("[SHUTDOWN] Final usage report sent successfully")
			}
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			a.Log("[SHUTDOWN] Final usage report timed out after 5s")
		}
	}
	// Stop credits usage reporting
	if a.licenseClient != nil {
		a.licenseClient.StopUsageReporting()
	}
	// Close EinoService (which closes Python pool) with timeout
	if a.einoService != nil {
		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					a.Log(fmt.Sprintf("[SHUTDOWN] Panic in EinoService close: %v", r))
				}
				close(done)
			}()
			a.einoService.Close()
		}()
		select {
		case <-done:
			a.Log("[SHUTDOWN] EinoService closed successfully")
		case <-time.After(5 * time.Second):
			a.Log("[SHUTDOWN] EinoService close timed out after 5s, forcing shutdown")
		}
	}
	// Shutdown all registered services via ServiceRegistry (reverse registration order)
	if a.registry != nil {
		a.registry.ShutdownAll()
	}
	// Close logger last - other services may need to log during shutdown
	a.logger.Close()
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	defer close(a.startupDone) // Signal background goroutines that startup is complete

	// Store App instance in context for system tray access
	ctx = context.WithValue(ctx, appContextKey, a)
	a.ctx = ctx

	// Start system tray (Windows/Linux only, handled by build tags)
	runSystray(ctx)

	// Create ServiceRegistry for lifecycle management
	a.registry = NewServiceRegistry(ctx, a.Log)
	a.Log("[STARTUP] ServiceRegistry created")

	// Load config to get DataCacheDir
	cfg, err := a.GetConfig()
	if err != nil {
		fmt.Printf("Error loading config on startup: %v\n", err)
		// Fallback to default storage dir if config fails
		path, _ := a.getStorageDir()
		if path != "" {
			_ = os.MkdirAll(path, 0755)
			sessionsDir := filepath.Join(path, "sessions")
			a.chatService = NewChatService(sessionsDir)
			a.dataSourceService = agent.NewDataSourceService(path, a.Log)
			// Need a basic config for memory service if loading failed, or just skip
			a.memoryService = agent.NewMemoryService(config.Config{DataCacheDir: path})
		}
		return
	}

	// Initialize i18n with language from config
	i18n.SyncLanguageFromConfig(&cfg)
	a.Log(fmt.Sprintf("[STARTUP] i18n initialized with language: %s", i18n.GetLanguageString()))

	// Use configured DataCacheDir
	dataDir := cfg.DataCacheDir
	if dataDir == "" {
		dataDir, _ = a.getStorageDir()
	}

	if dataDir != "" {
		_ = os.MkdirAll(dataDir, 0755)
		sessionsDir := filepath.Join(dataDir, "sessions")
		a.chatService = NewChatService(sessionsDir)
		a.dataSourceService = agent.NewDataSourceService(dataDir, a.Log)
		a.memoryService = agent.NewMemoryService(cfg)

		// Set data cache dir on PythonService for uv environment management
		if a.pythonService != nil {
			a.pythonService.SetDataCacheDir(dataDir)

			// Auto-setup uv environment in background if not ready
			go a.autoSetupUvEnvironment(&cfg)
		}

		// Initialize working context manager for UI state tracking
		a.workingContextManager = agent.NewWorkingContextManager(dataDir)
		a.Log("[STARTUP] Working context manager initialized")

		// Initialize analysis path manager for storyline tracking
		a.analysisPathManager = agent.NewAnalysisPathManager(dataDir)
		a.Log("[STARTUP] Analysis path manager initialized")

		// Initialize preference learner for user behavior tracking
		preferenceLearner := agent.NewPreferenceLearner(dataDir)
		a.Log("[STARTUP] Preference learner initialized")

		// Initialize intent enhancement service for improved intent understanding
		intentEnhancementService := agent.NewIntentEnhancementService(
			dataDir,
			preferenceLearner,
			a.memoryService,
			a.Log,
		)
		if err := intentEnhancementService.Initialize(); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Intent enhancement service initialization warning: %v", err))
		} else {
			a.Log("[STARTUP] Intent enhancement service initialized successfully")
		}

		// Initialize new intent understanding service (simplified architecture)
		// Validates: Requirements 7.1
		intentUnderstandingService := agent.NewIntentUnderstandingService(
			dataDir,
			a.dataSourceService,
			a.Log,
		)
		if err := intentUnderstandingService.Initialize(); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Intent understanding service initialization warning: %v", err))
		} else {
			a.Log("[STARTUP] Intent understanding service initialized successfully")
		}

		// Initialize skill service for skills management
		skillService := agent.NewSkillService(dataDir, a.Log)
		a.Log("[STARTUP] Skill service initialized")

		// Initialize search keywords manager for intelligent web search detection
		a.searchKeywordsManager = agent.NewSearchKeywordsManager(dataDir, a.Log)
		a.Log("[STARTUP] Search keywords manager initialized")

		// Initialize license client and try auto-activation if SN is saved
		a.licenseClient = agent.NewLicenseClient(a.Log)
		if cfg.LicenseSN != "" && cfg.LicenseServerURL != "" {
			a.Log("[STARTUP] Found saved license SN, attempting auto-activation...")
			
			// First try to load from local cache
			loadErr := a.licenseClient.LoadActivationData(cfg.LicenseSN)
			if loadErr != nil {
				a.Log(fmt.Sprintf("[STARTUP] No local cache or invalid: %v, activating from server...", loadErr))
				a.tryLicenseActivationWithRetry(&cfg, "Activation")
			} else {
				a.Log("[STARTUP] Loaded license from local cache")
				
				// Check if refresh is needed based on trust level
				needsRefresh, reason := a.licenseClient.NeedsRefresh()
				if needsRefresh {
					a.Log(fmt.Sprintf("[STARTUP] %s, refreshing...", reason))
					a.tryLicenseActivationWithRetry(&cfg, "Refresh")
					// If refresh failed (not rejected), clear cached data
					if a.licenseActivationFailed {
						a.licenseClient.Clear()
					}
				} else {
					trustLevel := a.licenseClient.GetTrustLevel()
					refreshInterval := a.licenseClient.GetRefreshInterval()
					trustLabel := "试用�"
					if trustLevel == "high" {
						trustLabel = "正式�"
					}
					a.Log(fmt.Sprintf("[STARTUP] License valid (%s, refresh every %d days)", trustLabel, refreshInterval))
				}
			}
			
			// Check if activation/refresh failed - will show error dialog and exit
			if a.licenseActivationFailed {
				a.Log("[STARTUP] License validation failed, application will show error and exit")
				// Don't continue with LLM initialization
				return
			}

			// Start credits usage reporting if applicable
			isCreditsMode := a.licenseClient.IsCreditsMode()
			trustLevel := a.licenseClient.GetTrustLevel()
			a.Log(fmt.Sprintf("[STARTUP] Credits reporting check: credits_mode=%v, trust_level=%s", isCreditsMode, trustLevel))
			if isCreditsMode && trustLevel == "low" {
				shouldReport := a.licenseClient.ShouldReportOnStartup()
				a.Log(fmt.Sprintf("[STARTUP] ShouldReportOnStartup=%v", shouldReport))
				if shouldReport {
					a.Log("[STARTUP] Reporting credits usage on startup (overdue)")
					go func() {
						defer func() {
							if r := recover(); r != nil {
								a.Log(fmt.Sprintf("[STARTUP] ReportUsage goroutine recovered from panic: %v", r))
							}
						}()
						a.licenseClient.ReportUsage()
					}()
				}
			}
			a.licenseClient.StartUsageReporting()
			
			// Update config with activated LLM settings
			if activationData := a.licenseClient.GetData(); activationData != nil && activationData.LLMAPIKey != "" {
				if applyActivatedLLMConfig(&cfg, activationData) {
					a.Log(fmt.Sprintf("[STARTUP] Using activated LLM config: provider=%s, model=%s, baseURL=%s",
						cfg.LLMProvider, cfg.ModelName, cfg.BaseURL))
				}
			}
		}

		a.Log(fmt.Sprintf("[STARTUP] Initializing EinoService with provider: %s, model: %s", cfg.LLMProvider, cfg.ModelName))
		es, err := agent.NewEinoService(cfg, a.dataSourceService, a.memoryService, a.workingContextManager, a.Log)
		if err != nil {
			a.Log(fmt.Sprintf("Failed to initialize EinoService: %v", err))
		} else {
			a.einoService = es
			a.Log("[STARTUP] EinoService initialized successfully")
		}

		// Initialize dashboard drag-drop layout services after all other services are ready
		fileService := database.NewFileService(dataDir)
		a.Log("[STARTUP] FileService initialized successfully")

		dataService := database.NewDataService(dataDir, fileService)
		// Set the data source service to avoid circular dependency
		if a.dataSourceService != nil {
			dataService.SetDataSourceService(a.dataSourceService)
		}
		a.Log("[STARTUP] DataService initialized successfully")

		layoutService := database.NewLayoutService(dataDir)
		a.Log("[STARTUP] LayoutService initialized successfully")

		exportService := database.NewExportService(dataService, layoutService)
		a.Log("[STARTUP] ExportService initialized successfully")

		// Initialize event aggregator for analysis results
		a.eventAggregator = NewEventAggregator(ctx)
		a.eventAggregator.SetLogger(a.Log)
		a.Log("[STARTUP] EventAggregator initialized successfully")

		// === CREATE AND REGISTER SERVICES WITH ServiceRegistry ===
		// Services are registered in dependency order:
		// ConfigService �basic services �business services
		// Requirements: 2.1, 2.2, 2.4

		// 1. Register ConfigService (critical - already created in NewApp)
		if err := a.registry.RegisterCritical(a.configService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register ConfigService: %v", err))
		}

		// 2. Create and register ChatFacadeService (critical)
		a.chatFacadeService = NewChatFacadeService(
			a.chatService,
			a.configService,
			a.einoService,
			a.eventAggregator,
			a.Log,
		)
		a.chatFacadeService.SetContext(ctx)
		if a.licenseClient != nil {
			a.chatFacadeService.SetLicenseClient(a.licenseClient)
		}
		if a.searchKeywordsManager != nil {
			a.chatFacadeService.SetSearchKeywordsManager(a.searchKeywordsManager)
		}
		if a.dataSourceService != nil {
			a.chatFacadeService.SetDataSourceService(a.dataSourceService)
		}
		a.chatFacadeService.SetSaveChartDataToFileFn(a.saveChartDataToFile)
		if err := a.registry.RegisterCritical(a.chatFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register ChatFacadeService: %v", err))
		}

		// 3. Create and register DataSourceFacadeService (critical)
		a.dataSourceFacadeService = NewDataSourceFacadeService(
			a.dataSourceService,
			a, // Use App as ConfigProvider so GetEffectiveConfig() includes license LLM config
			a.chatService,
			a.einoService,
			a.eventAggregator,
			a.Log,
		)
		a.dataSourceFacadeService.SetContext(ctx)
		if err := a.registry.RegisterCritical(a.dataSourceFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register DataSourceFacadeService: %v", err))
		}

		// 4. Create and register LicenseFacadeService (non-critical)
		a.licenseFacadeService = NewLicenseFacadeService(
			a.configService,
			a.configService,
			a.Log,
		)
		a.licenseFacadeService.SetContext(ctx)
		a.licenseFacadeService.SetChatFacadeService(a.chatFacadeService)
		a.licenseFacadeService.SetReinitializeServicesFn(a.reinitializeServices)
		if a.licenseClient != nil {
			a.licenseFacadeService.SetLicenseClient(a.licenseClient)
		}
		if err := a.registry.Register(a.licenseFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register LicenseFacadeService: %v", err))
		}

		// 5. Create and register AnalysisFacadeService (non-critical)
		a.analysisFacadeService = NewAnalysisFacadeService(
			a.chatService,
			a.configService,
			a.dataSourceService,
			a.einoService,
			a.eventAggregator,
			dataDir,
			a.Log,
		)
		a.analysisFacadeService.SetContext(ctx)
		if err := a.registry.Register(a.analysisFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register AnalysisFacadeService: %v", err))
		}

		// 6. Create and register ExportFacadeService (non-critical)
		a.exportFacadeService = NewExportFacadeService(
			a.dataSourceService,
			a.chatService,
			a.einoService,
			a.Log,
		)
		a.exportFacadeService.SetContext(ctx)
		if err := a.registry.Register(a.exportFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register ExportFacadeService: %v", err))
		}

		// 7. Create and register DashboardFacadeService (non-critical)
		a.dashboardFacadeService = NewDashboardFacadeService(
			a.dataSourceService,
			a.configService,
			layoutService,
			dataService,
			fileService,
			exportService,
			a.Log,
		)
		a.dashboardFacadeService.SetContext(ctx)
		if err := a.registry.Register(a.dashboardFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register DashboardFacadeService: %v", err))
		}

		// Initialize usage license store for local billing enforcement
		uls, err := NewUsageLicenseStore()
		if err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to create UsageLicenseStore: %v", err))
		} else {
			if err := uls.Load(); err != nil {
				a.Log(fmt.Sprintf("[STARTUP] Failed to load usage licenses: %v", err))
			}
			a.usageLicenseStore = uls
			a.Log("[STARTUP] UsageLicenseStore initialized successfully")
		}

		// Initialize pack password store (persists marketplace pack passwords across restarts)
		pps, err := NewPackPasswordStore()
		if err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to create PackPasswordStore: %v", err))
		} else {
			if err := pps.Load(); err != nil {
				a.Log(fmt.Sprintf("[STARTUP] Failed to load pack passwords: %v", err))
			}
			a.packPasswordStore = pps
			// Load persisted passwords into the in-memory map for backward compatibility
			pps.LoadIntoMap(a.packPasswords)
			a.Log("[STARTUP] PackPasswordStore initialized successfully")
		}

		// 8. Create and register MarketplaceFacadeService (non-critical)
		a.marketplaceFacadeService = NewMarketplaceFacadeService(
			a.configService,
			a.Log,
		)
		a.marketplaceFacadeService.SetContext(ctx)
		if a.licenseClient != nil {
			a.marketplaceFacadeService.SetLicenseClient(a.licenseClient)
		}
		if err := a.registry.Register(a.marketplaceFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register MarketplaceFacadeService: %v", err))
		}

		// 9. Create and register SkillFacadeService (non-critical)
		a.skillFacadeService = NewSkillFacadeService(
			skillService,
			a.einoService,
			a.chatFacadeService,
			a.Log,
		)
		a.skillFacadeService.SetContext(ctx)
		if err := a.registry.Register(a.skillFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register SkillFacadeService: %v", err))
		}

		// 10. Create and register PythonFacadeService (non-critical)
		a.pythonFacadeService = NewPythonFacadeService(
			a.pythonService,
			a.Log,
		)
		a.pythonFacadeService.SetContext(ctx)
		if err := a.registry.Register(a.pythonFacadeService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register PythonFacadeService: %v", err))
		}

		// 11. Create and register ConnectionTestService (non-critical)
		a.connectionTestService = NewConnectionTestService(
			a.configService,
			a.Log,
		)
		if a.licenseClient != nil {
			a.connectionTestService.SetLicenseClient(a.licenseClient)
		}
		a.connectionTestService.SetContext(ctx)
		if err := a.registry.Register(a.connectionTestService); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to register ConnectionTestService: %v", err))
		}

		// === INITIALIZE ALL REGISTERED SERVICES ===
		// InitializeAll() calls Initialize(ctx) on each service in registration order.
		// Critical services (ConfigService, ChatFacadeService, DataSourceFacadeService)
		// will cause startup failure if they fail. Non-critical services degrade gracefully.
		// Requirements: 2.1, 2.2, 2.4
		if err := a.registry.InitializeAll(); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] ServiceRegistry initialization failed: %v", err))
		} else {
			a.Log("[STARTUP] All services initialized via ServiceRegistry")
		}
	}

	// Always initialize logger directory for log management (compression, cleanup)
	// Set log max size
	maxSizeMB := cfg.LogMaxSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 100 // Default 100MB
	}
	a.logger.SetMaxSizeMB(maxSizeMB)
	
	// Initialize logger (this also handles compression of existing logs)
	if cfg.DetailedLog {
		a.logger.Init(dataDir)
	} else {
		// Just set the log directory for management purposes without enabling logging
		a.logger.SetLogDir(dataDir)
	}

	// Auto-detect location on startup if not configured
	// This runs in background to avoid blocking startup
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.Log(fmt.Sprintf("[STARTUP] autoDetectLocation goroutine recovered from panic: %v", r))
			}
		}()
		a.autoDetectLocation(&cfg)
	}()
}

// UpdateWorkingContext updates the working context from frontend events
// This enables context-aware analysis by capturing UI state (charts, filters, operations)
func (a *App) UpdateWorkingContext(sessionID string, updates map[string]interface{}) error {
	if a.workingContextManager == nil {
		return fmt.Errorf("working context manager not initialized")
	}

	a.Log(fmt.Sprintf("[WORKING-CONTEXT] Update for session %s", sessionID))
	return a.workingContextManager.UpdateContext(sessionID, updates)
}

// GetAnalysisPath retrieves the complete analysis path for a session
func (a *App) GetAnalysisPath(sessionID string) *agent.AnalysisPath {
	if a.analysisPathManager == nil {
		return nil
	}
	return a.analysisPathManager.GetPath(sessionID)
}

// MarkAsFinding marks content as an important finding
func (a *App) MarkAsFinding(sessionID, content string, importance int) error {
	if a.analysisPathManager == nil {
		return fmt.Errorf("analysis path manager not initialized")
	}

	finding := agent.ConfirmedFinding{
		Content:     content,
		Importance:  importance,
		ConfirmedBy: "user_marked",
	}

	a.Log(fmt.Sprintf("[ANALYSIS-PATH] Marked finding with importance %d for session %s", importance, sessionID))
	return a.analysisPathManager.AddFinding(sessionID, finding)
}

// GetAgentMemory retrieves the memory context for a thread
// Shows what the AI "remembers" at different memory tiers during analysis
func (a *App) GetAgentMemory(threadID string) (AgentMemoryView, error) {
	if a.chatService == nil {
		return AgentMemoryView{}, fmt.Errorf("services not initialized")
	}

	// Load only the specific thread instead of all threads (performance optimization)
	thread, err := a.chatService.LoadThread(threadID)
	if err != nil || thread == nil {
		return AgentMemoryView{
			LongTerm:   []string{"No conversation history yet."},
			MediumTerm: []string{"No conversation history yet."},
			ShortTerm:  []string{"No conversation history yet."},
		}, nil
	}
	messages := thread.Messages

	if len(messages) == 0 {
		return AgentMemoryView{
			LongTerm:   []string{"No conversation history yet."},
			MediumTerm: []string{"No conversation history yet."},
			ShortTerm:  []string{"No conversation history yet."},
		}, nil
	}

	// Short-term memory: Last 5 messages (what the AI sees in full detail)
	var shortTerm []string
	shortStart := 0
	if len(messages) > 5 {
		shortStart = len(messages) - 5
	}
	for i, msg := range messages[shortStart:] {
		content := msg.Content
		// Truncate very long messages for display
		if len(content) > 800 {
			content = content[:800] + "...\n[内容已截断]"
		}

		// Format with role and index
		roleIcon := "👤"
		if msg.Role == "assistant" {
			roleIcon = "🤖"
		} else if msg.Role == "system" {
			roleIcon = "⚙️"
		}

		shortTerm = append(shortTerm, fmt.Sprintf("%s %s (消息 #%d):\n%s",
			roleIcon, msg.Role, shortStart+i+1, content))
	}

	// Medium-term memory: Compressed summaries of older messages (messages beyond short-term)
	var mediumTerm []string
	if len(messages) > 5 {
		midEnd := shortStart
		midStart := 0
		if midEnd > 20 {
			midStart = midEnd - 20
		}

		// Extract user questions and key assistant findings
		userQuestions := []string{}
		assistantFindings := []string{}

		for _, msg := range messages[midStart:midEnd] {
			if msg.Role == "user" {
				q := msg.Content
				if len(q) > 150 {
					q = q[:150] + "..."
				}
				userQuestions = append(userQuestions, q)
			} else if msg.Role == "assistant" {
				// Extract meaningful content, filtering out incomplete/meaningless fragments
				content := msg.Content

				// Skip if content is too short to be meaningful
				if len(content) < 50 {
					continue
				}

				// Skip if it's just a prompt or question back to user
				lowerContent := strings.ToLower(content)
				if strings.HasPrefix(lowerContent, "�") ||
					strings.HasPrefix(lowerContent, "�") ||
					strings.HasPrefix(lowerContent, "what") ||
					strings.HasPrefix(lowerContent, "how") ||
					strings.HasPrefix(lowerContent, "could you") ||
					strings.HasPrefix(lowerContent, "can you") {
					continue
				}

				// Skip incomplete JSON blocks
				if strings.HasPrefix(content, "```json") && !strings.Contains(content, "```\n") {
					continue
				}
				if strings.HasPrefix(content, "```") && strings.Count(content, "```") < 2 {
					continue
				}

				// Extract a meaningful summary (first complete sentence or paragraph)
				summary := content

				// Try to find first complete paragraph or sentence
				if idx := strings.Index(content, "\n\n"); idx > 0 && idx < 500 {
					summary = content[:idx]
				} else if idx := strings.Index(content, "�"); idx > 0 && idx < 500 {
					summary = content[:idx+3] // Include the period (3 bytes in UTF-8)
				} else if idx := strings.Index(content, ". "); idx > 0 && idx < 500 {
					summary = content[:idx+1]
				} else if len(content) > 400 {
					summary = content[:400] + "..."
				}

				// Final check: skip if summary is still too short or meaningless
				if len(summary) < 50 || strings.HasPrefix(summary, "```") {
					continue
				}

				assistantFindings = append(assistantFindings, summary)
			}
		}

		if len(userQuestions) > 0 {
			mediumTerm = append(mediumTerm, fmt.Sprintf("📝 User asked about: %d topics", len(userQuestions)))
			for i, q := range userQuestions {
				if i >= 5 {
					mediumTerm = append(mediumTerm, fmt.Sprintf("  ... and %d more questions", len(userQuestions)-5))
					break
				}
				mediumTerm = append(mediumTerm, fmt.Sprintf("  �%s", q))
			}
		}

		if len(assistantFindings) > 0 {
			mediumTerm = append(mediumTerm, fmt.Sprintf("💡 Key findings: %d responses", len(assistantFindings)))
			for i, f := range assistantFindings {
				if i >= 3 {
					mediumTerm = append(mediumTerm, fmt.Sprintf("  ... and %d more findings", len(assistantFindings)-3))
					break
				}
				mediumTerm = append(mediumTerm, fmt.Sprintf("  �%s", f))
			}
		}
	}

	if len(mediumTerm) == 0 {
		mediumTerm = []string{"暂无压缩历史（对话足够短，全部保留在短期记忆中）"}
	}

	// Add persisted medium-term memories from MemoryService (AI-generated summaries)
	if a.memoryService != nil {
		_, _, _, sessionMedium := a.memoryService.GetMemories(threadID)
		if len(sessionMedium) > 0 {
			mediumTerm = append([]string{"📚 AI 自动生成的对话摘�"}, mediumTerm...)
			for _, mem := range sessionMedium {
				mediumTerm = append(mediumTerm, fmt.Sprintf("  📝 %s", mem))
			}
		}
	}

	// Long-term memory: Key facts, entities, and insights extracted from the conversation
	var longTerm []string

	// Extract substantive content from all messages
	var mentionedTables []string
	var keyInsights []string
	var dataPatterns []string

	seenTables := make(map[string]bool)
	for _, msg := range messages {
		content := msg.Content

		// Extract mentioned tables
		tableMatches := reTablePattern.FindAllStringSubmatch(content, -1)
		for _, match := range tableMatches {
			if len(match) > 1 {
				tableName := strings.ToLower(match[1])
				// Filter out common SQL keywords
				if tableName != "select" && tableName != "where" && tableName != "group" &&
					tableName != "order" && tableName != "limit" && !seenTables[tableName] {
					seenTables[tableName] = true
					mentionedTables = append(mentionedTables, match[1])
				}
			}
		}

		// Extract insights from assistant messages
		if msg.Role == "assistant" {
			for _, re := range reInsightPatterns {
				matches := re.FindAllStringSubmatch(content, 2)
				for _, match := range matches {
					if len(match) > 1 && len(keyInsights) < 5 {
						insight := strings.TrimSpace(match[1])
						if len(insight) > 20 {
							keyInsights = append(keyInsights, insight)
						}
					}
				}
			}

			// Extract data patterns (numbers, percentages, trends)
			numMatches := reNumPattern.FindAllString(content, 5)
			for _, num := range numMatches {
				if len(dataPatterns) < 3 {
					dataPatterns = append(dataPatterns, num)
				}
			}
		}
	}

	// Build substantive long-term memory
	if len(mentionedTables) > 0 {
		if len(mentionedTables) > 5 {
			mentionedTables = mentionedTables[:5]
		}
		longTerm = append(longTerm, fmt.Sprintf("📊 涉及数据� %s", strings.Join(mentionedTables, ", ")))
	}

	// Extract the main analysis topic from first user message
	for _, msg := range messages {
		if msg.Role == "user" {
			topic := msg.Content
			if len(topic) > 80 {
				topic = topic[:80] + "..."
			}
			longTerm = append(longTerm, fmt.Sprintf("🎯 分析主题: %s", topic))
			break
		}
	}

	// Add key insights
	for i, insight := range keyInsights {
		if i >= 3 {
			break
		}
		longTerm = append(longTerm, fmt.Sprintf("💡 %s", insight))
	}

	// Add data patterns if found
	if len(dataPatterns) > 0 {
		longTerm = append(longTerm, fmt.Sprintf("📈 关键数据: %s", strings.Join(dataPatterns, ", ")))
	}

	// Add any persisted long-term memories from MemoryService
	if a.memoryService != nil {
		globalDataSources, globalGoals, sessionLong, _ := a.memoryService.GetMemories(threadID)

		// Add a header if we have persistent memories
		if len(globalDataSources) > 0 || len(globalGoals) > 0 || len(sessionLong) > 0 {
			longTerm = append([]string{"🗄�持久化知识库:"}, longTerm...)
		}

		// Global data sources (cross-session knowledge)
		if len(globalDataSources) > 0 {
			longTerm = append(longTerm, "\n📊 全局数据�")
			for _, mem := range globalDataSources {
				longTerm = append(longTerm, fmt.Sprintf("  �%s", mem))
			}
		}

		// Global goals (overall objectives)
		if len(globalGoals) > 0 {
			longTerm = append(longTerm, "\n🎯 全局目标:")
			for _, mem := range globalGoals {
				longTerm = append(longTerm, fmt.Sprintf("  �%s", mem))
			}
		}

		// Session long-term (persistent facts for this session)
		if len(sessionLong) > 0 {
			longTerm = append(longTerm, "\n📌 会话持久化事�")
			for _, mem := range sessionLong {
				longTerm = append(longTerm, fmt.Sprintf("  �%s", mem))
			}
		}
	}

	// If nothing substantive found, show a meaningful message
	if len(longTerm) == 0 {
		longTerm = append(longTerm, "暂无提取到的持久化知识�")
		longTerm = append(longTerm, "")
		longTerm = append(longTerm, "💡 长期记忆会自动从以下内容中提取：")
		longTerm = append(longTerm, "  �数据源架构（表名、字段名�")
		longTerm = append(longTerm, "  �业务规则和定�")
		longTerm = append(longTerm, "  �数据特征（枚举值、状态类型）")
		longTerm = append(longTerm, "")
		longTerm = append(longTerm, "继续对话和分析后，系统将自动提取和保存这些知识�")
	}

	return AgentMemoryView{
		LongTerm:   longTerm,
		MediumTerm: mediumTerm,
		ShortTerm:  shortTerm,
	}, nil
}

// Log writes a detailed log entry if logging is enabled
func (a *App) Log(message string) {
	a.logger.Log(message)
}

// WriteSystemLog writes a log entry to system.log in the user cache directory
// This is exposed to frontend for debugging purposes
func (a *App) WriteSystemLog(level, source, message string) error {
	cfg, err := a.GetConfig()
	if err != nil {
		return err
	}

	// Get log file path
	logPath := filepath.Join(cfg.DataCacheDir, "system.log")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}

	// Check file size and rotate if needed
	maxSizeMB := cfg.LogMaxSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 100 // Default 100MB
	}
	maxBytes := int64(maxSizeMB) * 1024 * 1024

	if info, err := os.Stat(logPath); err == nil && info.Size() >= maxBytes {
		// Rotate: compress old log and create new one
		a.rotateSystemLog(logPath, cfg.DataCacheDir)
	}

	// Open file in append mode
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Format: [timestamp] [level] [source] message
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logEntry := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, source, message)

	_, err = f.WriteString(logEntry)
	return err
}

// rotateSystemLog compresses the system.log file and creates a new one
func (a *App) rotateSystemLog(logPath, cacheDir string) {
	// Create archive filename with timestamp
	timestamp := time.Now().Format("2006-01-02_150405")
	archivePath := filepath.Join(cacheDir, fmt.Sprintf("system_%s.log.zip", timestamp))

	// Compress the log file
	if err := compressFile(logPath, archivePath); err != nil {
		a.Log(fmt.Sprintf("[SYSTEM_LOG] Failed to compress system.log: %v", err))
		return
	}

	// Remove original file
	os.Remove(logPath)
	a.Log(fmt.Sprintf("[SYSTEM_LOG] Rotated system.log to %s", archivePath))

	// Cleanup old archives (keep last 10)
	cleanupOldSystemLogArchives(cacheDir, 10)
}

// compressFile compresses a single file to a zip archive
func compressFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	zipFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	info, err := srcFile.Stat()
	if err != nil {
		zipWriter.Close()
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		zipWriter.Close()
		return err
	}
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		zipWriter.Close()
		return err
	}

	if _, err = io.Copy(writer, srcFile); err != nil {
		zipWriter.Close()
		return err
	}

	// Close the zip writer explicitly to flush the central directory.
	// A deferred Close() would silently swallow this error, producing corrupt archives.
	return zipWriter.Close()
}

// cleanupOldSystemLogArchives removes old system log archives, keeping only the most recent ones
func cleanupOldSystemLogArchives(cacheDir string, keepCount int) {
	pattern := filepath.Join(cacheDir, "system_*.log.zip")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) <= keepCount {
		return
	}

	// Sort by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	files := make([]fileInfo, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: path, modTime: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	// Remove oldest files
	toRemove := len(files) - keepCount
	for i := 0; i < toRemove; i++ {
		os.Remove(files[i].path)
	}
}

func (a *App) getStorageDir() (string, error) {
	if a.storageDir != "" {
		return a.storageDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Vantagics"), nil
}

func (a *App) getConfigPath() (string, error) {
	dir, err := a.getStorageDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// autoDetectLocation attempts to detect location via IP and save to config if not already set
func (a *App) autoDetectLocation(cfg *config.Config) {
	// Skip if location is already configured
	if cfg.Location != nil && cfg.Location.City != "" {
		a.Log("[STARTUP] Location already configured, skipping auto-detection")
		return
	}

	a.Log("[STARTUP] Attempting to auto-detect location via IP...")

	// Try to get IP-based location
	ipLoc, err := agent.GetIPBasedLocation()
	if err != nil {
		a.Log(fmt.Sprintf("[STARTUP] IP location detection failed: %v", err))
		return
	}

	if ipLoc == nil || !ipLoc.Available || ipLoc.City == "" {
		a.Log("[STARTUP] IP location returned no valid data")
		return
	}

	// Update config with detected location
	cfg.Location = &config.LocationConfig{
		Country:   ipLoc.Country,
		City:      ipLoc.City,
		Latitude:  ipLoc.Latitude,
		Longitude: ipLoc.Longitude,
	}

	// Save the updated config
	if err := a.SaveConfig(*cfg); err != nil {
		a.Log(fmt.Sprintf("[STARTUP] Failed to save auto-detected location: %v", err))
		return
	}

	a.Log(fmt.Sprintf("[STARTUP] Auto-detected location saved: %s, %s (%.4f, %.4f)",
		ipLoc.City, ipLoc.Country, ipLoc.Latitude, ipLoc.Longitude))
}

// GetConfig loads the config from the ~/Vantagics/config.json
func (a *App) GetConfig() (config.Config, error) {
	path, err := a.getConfigPath()
	if err != nil {
		return config.Config{}, err
	}

	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, "Vantagics")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		defaultCfg := config.Config{
			LLMProvider:       "OpenAI",
			ModelName:         "gpt-4o",
			MaxTokens:         8192, // Safe default, will be adjusted per provider
			LocalCache:        true,
			Language:          "English",
			DataCacheDir:      defaultDataDir,
			MaxPreviewRows:    100,
			IntentEnhancement: config.DefaultIntentEnhancementConfig(),
		}
		// Fill Shopify credentials from embedded appdata
		if shopifyConfig, err := a.GetShopifyConfigFromAppData(); err == nil && shopifyConfig != nil {
			defaultCfg.ShopifyClientID = shopifyConfig.ClientID
			defaultCfg.ShopifyClientSecret = shopifyConfig.ClientSecret
		}
		return defaultCfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return config.Config{}, err
	}

	var cfg config.Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return config.Config{}, err
	}

	// Ensure DataCacheDir has a default if empty in existing config
	if cfg.DataCacheDir == "" {
		cfg.DataCacheDir = defaultDataDir
	}

	if cfg.MaxPreviewRows <= 0 {
		cfg.MaxPreviewRows = 100
	}

	// Initialize SearchAPIs with defaults if empty or nil
	if cfg.SearchAPIs == nil || len(cfg.SearchAPIs) == 0 {
		cfg.SearchAPIs = []config.SearchAPIConfig{
			{
				ID:          "serper",
				Name:        "Serper (Google Search)",
				Description: "Google Search API via Serper.dev (requires API key)",
				APIKey:      "",
				Enabled:     false,
				Tested:      false,
			},
			{
				ID:          "uapi_pro",
				Name:        "UAPI Pro",
				Description: "UAPI Pro search service with structured data (API key optional)",
				APIKey:      "",
				Enabled:     true, // Default enabled
				Tested:      false,
			},
		}
	} else {
		// Remove DuckDuckGo if it exists (deprecated)
		var filteredAPIs []config.SearchAPIConfig
		for _, api := range cfg.SearchAPIs {
			if api.ID != "duckduckgo" {
				filteredAPIs = append(filteredAPIs, api)
			}
		}
		cfg.SearchAPIs = filteredAPIs
		// Reset active API if it was DuckDuckGo
		if cfg.ActiveSearchAPI == "duckduckgo" {
			cfg.ActiveSearchAPI = ""
		}
	}

	// Set default active search API to uapi_pro if not set
	if cfg.ActiveSearchAPI == "" {
		cfg.ActiveSearchAPI = "uapi_pro"
		// Ensure uapi_pro is enabled
		for i := range cfg.SearchAPIs {
			if cfg.SearchAPIs[i].ID == "uapi_pro" {
				cfg.SearchAPIs[i].Enabled = true
				break
			}
		}
	}

	// Initialize IntentEnhancement with defaults if nil (backward compatibility)
	if cfg.IntentEnhancement == nil {
		cfg.IntentEnhancement = config.DefaultIntentEnhancementConfig()
	} else {
		// Validate existing configuration
		cfg.IntentEnhancement.Validate()
	}

	// Fill Shopify credentials from embedded appdata if not set in config
	if cfg.ShopifyClientID == "" || cfg.ShopifyClientSecret == "" {
		if shopifyConfig, err := a.GetShopifyConfigFromAppData(); err == nil && shopifyConfig != nil {
			if cfg.ShopifyClientID == "" {
				cfg.ShopifyClientID = shopifyConfig.ClientID
			}
			if cfg.ShopifyClientSecret == "" {
				cfg.ShopifyClientSecret = shopifyConfig.ClientSecret
			}
		}
	}

	return cfg, nil
}

// GetEffectiveConfig returns the config with activated license settings merged in
// This should be used when creating LLM services to ensure activated LLM config is used
func (a *App) GetEffectiveConfig() (config.Config, error) {
	cfg, err := a.GetConfig()
	if err != nil {
		return cfg, err
	}

	// Merge activated license LLM config if available
	if a.licenseClient != nil && a.licenseClient.IsActivated() {
		activationData := a.licenseClient.GetData()
		applyActivatedLLMConfig(&cfg, activationData)

		// Also merge activated search config if available
		if activationData != nil && activationData.SearchAPIKey != "" && activationData.SearchType != "" {
			// Update the search API config based on activation data
			found := false
			for i := range cfg.SearchAPIs {
				if cfg.SearchAPIs[i].ID == activationData.SearchType {
					cfg.SearchAPIs[i].APIKey = activationData.SearchAPIKey
					cfg.SearchAPIs[i].Enabled = true
					cfg.ActiveSearchAPI = activationData.SearchType
					found = true
					break
				}
			}
			// If search type not found in existing config, add it
			if !found {
				cfg.ActiveSearchAPI = activationData.SearchType
			}
		}
	}

	return cfg, nil
}

// GetActiveSearchAPIInfo returns information about the currently active search API
// Returns: apiName, apiID, isEnabled, error
func (a *App) GetActiveSearchAPIInfo() (string, string, bool, error) {
	cfg, err := a.GetConfig()
	if err != nil {
		return "", "", false, err
	}

	cfg.InitializeSearchAPIs()
	activeAPI := cfg.GetActiveSearchAPI()

	if activeAPI == nil {
		return "", "", false, nil
	}

	return activeAPI.Name, activeAPI.ID, activeAPI.Enabled, nil
}

// UpdateDeviceLocation updates the device location from frontend Geolocation API
// This is called by the frontend when it gets a location update
func (a *App) UpdateDeviceLocation(latitude, longitude, accuracy float64, timestamp int64, city, country, address string, available bool, errorMsg string) {
	a.Log(fmt.Sprintf("[LOCATION] Updating device location: lat=%.6f, lng=%.6f, city=%s, country=%s, available=%v",
		latitude, longitude, city, country, available))

	agent.UpdateLocation(agent.LocationData{
		Latitude:  latitude,
		Longitude: longitude,
		Accuracy:  accuracy,
		Timestamp: timestamp,
		City:      city,
		Country:   country,
		Address:   address,
		Available: available,
		Error:     errorMsg,
	})
}

// GetDeviceLocation returns the current stored device location
func (a *App) GetDeviceLocation() map[string]interface{} {
	loc := agent.GetCurrentLocation()
	return map[string]interface{}{
		"latitude":  loc.Latitude,
		"longitude": loc.Longitude,
		"accuracy":  loc.Accuracy,
		"timestamp": loc.Timestamp,
		"city":      loc.City,
		"country":   loc.Country,
		"address":   loc.Address,
		"available": loc.Available,
		"error":     loc.Error,
	}
}

// SaveConfig saves the config to the ~/Vantagics/config.json
func (a *App) SaveConfig(cfg config.Config) error {
	// Migrate legacy web search configuration to new MCP services format
	if cfg.WebSearchProvider != "" && cfg.WebSearchAPIKey != "" {
		// Check if this legacy config has already been migrated
		migrated := false
		for _, svc := range cfg.MCPServices {
			if svc.Name == "Web Search ("+cfg.WebSearchProvider+")" {
				migrated = true
				break
			}
		}

		// If not migrated, add it to MCPServices
		if !migrated {
			var mcpURL string
			switch cfg.WebSearchProvider {
			case "Tavily":
				mcpURL = fmt.Sprintf("https://mcp.tavily.com/mcp/?tavilyApiKey=%s", cfg.WebSearchAPIKey)
			case "Bright":
				mcpURL = fmt.Sprintf("https://mcp.brightdata.com/mcp?token=%s", cfg.WebSearchAPIKey)
			}

			if mcpURL != "" {
				newService := config.MCPService{
					ID:          fmt.Sprintf("websearch-%s", strings.ToLower(cfg.WebSearchProvider)),
					Name:        fmt.Sprintf("Web Search (%s)", cfg.WebSearchProvider),
					Description: fmt.Sprintf("Web search powered by %s", cfg.WebSearchProvider),
					URL:         mcpURL,
					Enabled:     true,
				}
				cfg.MCPServices = append(cfg.MCPServices, newService)
			}

			// Clear legacy fields after migration
			cfg.WebSearchProvider = ""
			cfg.WebSearchAPIKey = ""
			cfg.WebSearchMCPURL = ""
		}
	}

	// Initialize MCPServices if nil
	if cfg.MCPServices == nil {
		cfg.MCPServices = []config.MCPService{}
	}

	// Initialize SearchEngines if empty
	cfg.InitializeSearchEngines()

	// Validate DataCacheDir exists if it's set
	if cfg.DataCacheDir != "" {
		info, err := os.Stat(cfg.DataCacheDir)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("data cache directory does not exist: %s", cfg.DataCacheDir)
			}
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("data cache path is not a directory: %s", cfg.DataCacheDir)
		}
	}

	dir, err := a.getStorageDir()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Handle Logging State Change
	logDir := cfg.DataCacheDir
	if logDir == "" {
		logDir = dir // fallback to storage dir
	}
	
	// Always update log max size
	maxSizeMB := cfg.LogMaxSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 100 // Default 100MB
	}
	a.logger.SetMaxSizeMB(maxSizeMB)
	
	if cfg.DetailedLog {
		// Enable detailed logging
		a.logger.Init(logDir)
	} else {
		// Disable detailed logging but keep log directory for management
		a.logger.Close()
		a.logger.SetLogDir(logDir)
	}

	// Save the configuration file (0600: owner-only read/write since it contains API keys)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	// Reinitialize services that depend on configuration
	a.reinitializeServices(cfg)

	// Sync i18n language setting
	i18n.SyncLanguageFromConfig(&cfg)
	a.Log(fmt.Sprintf("[CONFIG] Language updated to: %s", i18n.GetLanguageString()))

	// Update window title based on language
	a.updateWindowTitle(cfg.Language)

	// Update application menu based on language
	a.UpdateApplicationMenu(cfg.Language)

	// Notify frontend that configuration has been updated
	runtime.EventsEmit(a.ctx, "config-updated")

	a.Log("Configuration saved and services reinitialized")
	return nil
}

// SaveLayoutConfig saves only layout-related config fields (sidebarWidth, panelRightRatio)
// without triggering config-updated event or reinitializing services.
func (a *App) SaveLayoutConfig(sidebarWidth int, panelRightRatio float64) error {
	cfg, err := a.GetConfig()
	if err != nil {
		return err
	}

	cfg.SidebarWidth = sidebarWidth
	cfg.PanelRightRatio = panelRightRatio
	cfg.PanelRightWidth = 0 // Clear deprecated field

	configPath, err := a.getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LogStats represents statistics about log files
type LogStats struct {
	TotalSizeMB  float64 `json:"totalSizeMB"`
	LogCount     int     `json:"logCount"`
	ArchiveCount int     `json:"archiveCount"`
	LogDir       string  `json:"logDir"`
}

// GetLogStats returns statistics about log files (including system.log)
func (a *App) GetLogStats() (LogStats, error) {
	cfg, _ := a.GetConfig()
	
	// Get stats from logger (vantagics_*.log files)
	totalSizeMB, logCount, archiveCount, err := a.logger.GetLogStats()
	if err != nil {
		// If logger not initialized, just count system.log
		totalSizeMB = 0
		logCount = 0
		archiveCount = 0
	}
	
	// Add system.log stats
	if cfg.DataCacheDir != "" {
		// Count system.log
		systemLogPath := filepath.Join(cfg.DataCacheDir, "system.log")
		if info, err := os.Stat(systemLogPath); err == nil {
			totalSizeMB += float64(info.Size()) / (1024 * 1024)
			logCount++
		}
		
		// Count system.log archives
		pattern := filepath.Join(cfg.DataCacheDir, "system_*.log.zip")
		if matches, err := filepath.Glob(pattern); err == nil {
			archiveCount += len(matches)
			for _, path := range matches {
				if info, err := os.Stat(path); err == nil {
					totalSizeMB += float64(info.Size()) / (1024 * 1024)
				}
			}
		}
	}
	
	logDir := a.logger.GetLogDir()
	if logDir == "" {
		logDir = cfg.DataCacheDir
	}
	
	return LogStats{
		TotalSizeMB:  totalSizeMB,
		LogCount:     logCount,
		ArchiveCount: archiveCount,
		LogDir:       logDir,
	}, nil
}

// CleanupLogs compresses all log files and removes old archives
func (a *App) CleanupLogs() error {
	a.Log("Starting manual log cleanup...")
	
	// Cleanup vantagics_*.log files
	err := a.logger.CleanupAllLogs()
	if err != nil {
		a.Log(fmt.Sprintf("Logger cleanup failed: %v", err))
	}
	
	// Cleanup system.log
	cfg, _ := a.GetConfig()
	if cfg.DataCacheDir != "" {
		systemLogPath := filepath.Join(cfg.DataCacheDir, "system.log")
		if info, err := os.Stat(systemLogPath); err == nil && info.Size() > 1024*1024 { // > 1MB
			a.rotateSystemLog(systemLogPath, cfg.DataCacheDir)
		}
		// Cleanup old system.log archives
		cleanupOldSystemLogArchives(cfg.DataCacheDir, 10)
	}
	
	a.Log("Manual log cleanup completed")
	return nil
}

// updateWindowTitle updates the window title based on language
func (a *App) updateWindowTitle(language string) {
	var title string
	if language == "简体中�" {
		title = "万策"
	} else {
		title = "Vantagics"
	}
	runtime.WindowSetTitle(a.ctx, title)
	a.Log(fmt.Sprintf("Window title updated to: %s", title))
}

// UpdateApplicationMenu updates the application menu based on language
// This is called from SaveConfig when language changes
func (a *App) UpdateApplicationMenu(language string) {
	// Rebuild the menu with new language
	newMenu := createApplicationMenu(a, language)
	
	// Update the global menu reference
	appMenu = newMenu
	
	// Apply the new menu to the application
	runtime.MenuSetApplicationMenu(a.ctx, newMenu)
	runtime.MenuUpdateApplicationMenu(a.ctx)
	
	a.Log(fmt.Sprintf("Application menu updated to language: %s", language))
}

// reinitializeServices reinitializes services that depend on configuration.
// Uses the ServiceRegistry to look up and update affected facade services.
// Requirements: 2.1
func (a *App) reinitializeServices(cfg config.Config) {
	// Check if we have activated license with LLM config
	if a.licenseClient != nil && a.licenseClient.IsActivated() {
		activationData := a.licenseClient.GetData()
		if applyActivatedLLMConfig(&cfg, activationData) {
			a.Log(fmt.Sprintf("[REINIT] Using activated license LLM config: Provider=%s, Model=%s, BaseURL=%s",
				cfg.LLMProvider, cfg.ModelName, cfg.BaseURL))
		}
	}

	// Update PythonService dataCacheDir when config changes
	if a.pythonService != nil && cfg.DataCacheDir != "" {
		a.pythonService.SetDataCacheDir(cfg.DataCacheDir)
	}

	// Reinitialize MemoryService if configuration changed
	if a.memoryService != nil {
		a.memoryService = agent.NewMemoryService(cfg)
		a.Log("MemoryService reinitialized with new configuration")
	}

	// Reinitialize EinoService if it exists and dataSourceService is available
	if a.dataSourceService != nil {
		// Store reference to old service in case reinitialization fails
		oldEinoService := a.einoService

		// Create new EinoService with updated configuration, passing memoryService
		es, err := agent.NewEinoService(cfg, a.dataSourceService, a.memoryService, a.workingContextManager, a.Log)
		if err != nil {
			a.Log(fmt.Sprintf("Failed to reinitialize EinoService: %v", err))
			// Keep the old service if reinitialization fails
			if oldEinoService != nil {
				a.Log("Keeping previous EinoService due to reinitialization failure")
			} else {
				a.Log("No EinoService available - analysis features will be disabled until configuration is fixed")
			}
			// Emit configuration error event to frontend
			runtime.EventsEmit(a.ctx, "config-error", map[string]interface{}{
				"type":       "eino_service",
				"message":    fmt.Sprintf("Failed to initialize analysis service: %v", err),
				"suggestion": "Please check your LLM configuration, especially the model name field",
			})
		} else {
			// Close old service only after successful creation of new one
			if oldEinoService != nil {
				oldEinoService.Close()
				a.Log("Previous EinoService closed")
			}
			a.einoService = es
			a.Log("EinoService reinitialized with new configuration")

			// Update all facade services that depend on EinoService via the registry
			a.updateRegisteredServicesEino(es)
		}
	}

	// Note: LLMService is created fresh for each request in SendMessage, so no reinitialization needed
}

// updateRegisteredServicesEino updates all registered facade services that hold
// an EinoService reference. Uses the ServiceRegistry to look up services by name.
func (a *App) updateRegisteredServicesEino(es *agent.EinoService) {
	if a.registry == nil {
		a.Log("[REINIT] ServiceRegistry not available, skipping facade service updates")
		return
	}

	// Update ChatFacadeService
	if svc, ok := a.registry.Get("chat"); ok {
		if chatSvc, ok := svc.(*ChatFacadeService); ok {
			chatSvc.SetEinoService(es)
			a.Log("[REINIT] ChatFacadeService updated with new EinoService")
		}
	}

	// Update DataSourceFacadeService
	if svc, ok := a.registry.Get("datasource"); ok {
		if dsSvc, ok := svc.(*DataSourceFacadeService); ok {
			dsSvc.SetEinoService(es)
			a.Log("[REINIT] DataSourceFacadeService updated with new EinoService")
		}
	}

	// Update AnalysisFacadeService
	if svc, ok := a.registry.Get("analysis"); ok {
		if analysisSvc, ok := svc.(*AnalysisFacadeService); ok {
			analysisSvc.SetEinoService(es)
			a.Log("[REINIT] AnalysisFacadeService updated with new EinoService")
		}
	}

	// Update ExportFacadeService
	if svc, ok := a.registry.Get("export"); ok {
		if exportSvc, ok := svc.(*ExportFacadeService); ok {
			exportSvc.SetEinoService(es)
			a.Log("[REINIT] ExportFacadeService updated with new EinoService")
		}
	}

	// Update SkillFacadeService
	if svc, ok := a.registry.Get("skill"); ok {
		if skillSvc, ok := svc.(*SkillFacadeService); ok {
			skillSvc.SetEinoService(es)
			a.Log("[REINIT] SkillFacadeService updated with new EinoService")
		}
	}
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// ConnectionResult represents the result of a connection test
type ConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestLLMConnection tests the connection to an LLM provider
func (a *App) TestLLMConnection(cfg config.Config) ConnectionResult {
	if a.connectionTestService == nil {
		return ConnectionResult{Success: false, Message: "connection test service not initialized"}
	}
	return a.connectionTestService.TestLLMConnection(cfg)
}

// TestMCPService tests the connection to an MCP service
func (a *App) TestMCPService(url string) ConnectionResult {
	if a.connectionTestService == nil {
		return ConnectionResult{Success: false, Message: "connection test service not initialized"}
	}
	return a.connectionTestService.TestMCPService(url)
}

// TestSearchEngine tests if a search engine is accessible
func (a *App) TestSearchEngine(url string) ConnectionResult {
	if a.connectionTestService == nil {
		return ConnectionResult{Success: false, Message: "connection test service not initialized"}
	}
	return a.connectionTestService.TestSearchEngine(url)
}

// TestSearchTools tests web_search and web_fetch tools with a sample query
// DEPRECATED: This function used chromedp-based tools which have been removed.
// Search functionality now uses Search API (search_api_tool.go)
// Web fetch functionality now uses HTTP client (web_fetch_tool.go)
func (a *App) TestSearchTools(engineURL string) ConnectionResult {
	if a.connectionTestService == nil {
		return ConnectionResult{Success: false, Message: "connection test service not initialized"}
	}
	return a.connectionTestService.TestSearchTools(engineURL)
}

// TestProxy tests if a proxy server is accessible
func (a *App) TestProxy(proxyConfig config.ProxyConfig) ConnectionResult {
	if a.connectionTestService == nil {
		return ConnectionResult{Success: false, Message: "connection test service not initialized"}
	}
	return a.connectionTestService.TestProxy(proxyConfig)
}

// TestUAPIConnection tests the connection to UAPI service
func (a *App) TestUAPIConnection(apiToken, baseURL string) ConnectionResult {
	if a.connectionTestService == nil {
		return ConnectionResult{Success: false, Message: "connection test service not initialized"}
	}
	return a.connectionTestService.TestUAPIConnection(apiToken, baseURL)
}

// TestSearchAPI tests a search API configuration
func (a *App) TestSearchAPI(apiConfig config.SearchAPIConfig) ConnectionResult {
	if a.connectionTestService == nil {
		return ConnectionResult{Success: false, Message: "connection test service not initialized"}
	}
	return a.connectionTestService.TestSearchAPI(apiConfig)
}

func (a *App) getDashboardTranslations(lang string) map[string]string {
	if a.dashboardFacadeService == nil {
		return map[string]string{}
	}
	return a.dashboardFacadeService.getDashboardTranslations(lang)
}

// GetDashboardData returns summary statistics and insights about data sources
func (a *App) GetDashboardData() DashboardData {
	if a.dashboardFacadeService == nil {
		return DashboardData{}
	}
	return a.dashboardFacadeService.GetDashboardData()
}

func (a *App) getLangPrompt(cfg config.Config) string {
	if cfg.Language == "简体中�" {
		return "Simplified Chinese"
	}
	return "English"
}

// getLangPromptFromMessage detects the language from the user's message
// and returns the appropriate language prompt string.
// This ensures the LLM responds in the same language as the user's question.
func (a *App) getLangPromptFromMessage(message string) string {
	return "the same language as the user's message"
}

// GenerateIntentSuggestions generates possible interpretations of user's intent
func (a *App) GenerateIntentSuggestions(threadID, userMessage string) ([]IntentSuggestion, error) {
	if a.analysisFacadeService == nil {
		return nil, WrapError("App", "GenerateIntentSuggestions", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.GenerateIntentSuggestions(threadID, userMessage)
}

// GenerateIntentSuggestionsWithExclusions generates possible interpretations of user's intent,
// excluding previously generated suggestions
// Validates: Requirements 5.1, 5.2, 5.3, 2.3, 6.5, 2.2, 7.1
func (a *App) GenerateIntentSuggestionsWithExclusions(threadID, userMessage string, excludedSuggestions []IntentSuggestion) ([]IntentSuggestion, error) {
	if a.analysisFacadeService == nil {
		return nil, WrapError("App", "GenerateIntentSuggestionsWithExclusions", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.GenerateIntentSuggestionsWithExclusions(threadID, userMessage, excludedSuggestions)
}

// convertAgentSuggestions converts agent.IntentSuggestion to IntentSuggestion
// Used for converting suggestions from the new IntentUnderstandingService
// Validates: Requirements 7.1, 7.4 - Maintain API compatibility
func convertAgentSuggestions(agentSuggestions []agent.IntentSuggestion) []IntentSuggestion {
	suggestions := make([]IntentSuggestion, len(agentSuggestions))
	for i, s := range agentSuggestions {
		suggestions[i] = IntentSuggestion{
			ID:          s.ID,
			Title:       s.Title,
			Description: s.Description,
			Icon:        s.Icon,
			Query:       s.Query,
		}
	}
	return suggestions
}

// convertToAgentSuggestions converts IntentSuggestion to agent.IntentSuggestion
// Used for passing exclusions to the new IntentUnderstandingService
// Validates: Requirements 7.1, 7.4 - Maintain API compatibility
func convertToAgentSuggestions(suggestions []IntentSuggestion) []agent.IntentSuggestion {
	agentSuggestions := make([]agent.IntentSuggestion, len(suggestions))
	for i, s := range suggestions {
		agentSuggestions[i] = agent.IntentSuggestion{
			ID:          s.ID,
			Title:       s.Title,
			Description: s.Description,
			Icon:        s.Icon,
			Query:       s.Query,
		}
	}
	return agentSuggestions
}

// buildIntentUnderstandingPrompt builds the prompt for intent understanding
// Validates: Requirements 6.5, 2.2 - Use summary when available to prevent context overload
func (a *App) buildIntentUnderstandingPrompt(userMessage, tableName string, columns []string, language string, excludedSuggestions []IntentSuggestion, dataSourceID string, exclusionSummary string) string {
	if a.analysisFacadeService == nil {
		return ""
	}
	return a.analysisFacadeService.buildIntentUnderstandingPrompt(userMessage, tableName, columns, language, excludedSuggestions, dataSourceID, exclusionSummary)
}

func (a *App) parseIntentSuggestions(response string) ([]IntentSuggestion, error) {
	if a.analysisFacadeService == nil {
		return nil, WrapError("App", "parseIntentSuggestions", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.parseIntentSuggestions(response)
}

func (a *App) getString(m map[string]interface{}, key string) string {
	return getStringFromMap(m, key)
}

// saveErrorToChatThread saves an error message as an assistant chat message so the user can see it in the chat area.
func (a *App) saveErrorToChatThread(threadID, errorCode, message string) {
	if a.chatFacadeService == nil || threadID == "" {
		return
	}
	a.chatFacadeService.SaveErrorToChatThread(threadID, errorCode, message)
}

// SendMessage sends a message to the AI
// Task 3.1: Added requestId parameter for request tracking (Requirements 1.3, 4.3, 4.4)
func (a *App) SendMessage(threadID, message, userMessageID, requestID string) (string, error) {
	if a.chatFacadeService == nil {
		return "", WrapError("App", "SendMessage", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.SendMessage(threadID, message, userMessageID, requestID)
}

// SendFreeChatMessage sends a message to the LLM without data source context (free chat mode)
// This allows users to have a direct conversation with the LLM like web ChatGPT
// Uses streaming for better user experience
// Supports web search and fetch tools for information retrieval (e.g., weather queries)
func (a *App) SendFreeChatMessage(threadID, message, userMessageID string) (string, error) {
	if a.chatFacadeService == nil {
		return "", WrapError("App", "SendFreeChatMessage", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.SendFreeChatMessage(threadID, message, userMessageID)
}

// detectWebSearchNeed checks if the user's message requires web search
func (a *App) detectWebSearchNeed(message string) bool {
	// Use the search keywords manager if available
	if a.searchKeywordsManager != nil {
		needsSearch, keyword := a.searchKeywordsManager.DetectSearchNeed(message)
		if needsSearch {
			a.Log(fmt.Sprintf("[FREE-CHAT] Detected search keyword: %s", keyword))
			// Record keyword usage for learning
			a.searchKeywordsManager.RecordKeywordUsage(keyword)
			return true
		}
		return false
	}

	// Fallback to hardcoded keywords if manager not initialized
	searchKeywords := []string{
		// Chinese keywords
		"天气", "气温", "温度", "下雨", "下雪", "晴天", "阴天",
		"新闻", "最�", "今天", "现在", "实时", "当前",
		"股票", "股价", "汇率", "价格", "多少�",
		"搜索", "查询", "查一�", "帮我�", "帮我�",
		"网上", "网络", "互联�",
		// English keywords
		"weather", "temperature", "rain", "snow", "sunny", "cloudy",
		"news", "latest", "today", "now", "current", "real-time",
		"stock", "price", "exchange rate", "how much",
		"search", "look up", "find", "google",
		"online", "internet", "web",
	}

	lowerMessage := strings.ToLower(message)
	for _, keyword := range searchKeywords {
		if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
			a.Log(fmt.Sprintf("[FREE-CHAT] Detected search keyword: %s", keyword))
			return true
		}
	}
	return false
}

// formatToolResultsForUser formats raw tool results into a user-friendly message
func (a *App) formatToolResultsForUser(results []string, langPrompt string) string {
	if len(results) == 0 {
		return "抱歉，未能获取到有效信息�"
	}

	var formatted strings.Builder
	isChinese := strings.Contains(langPrompt, "中文") || strings.Contains(langPrompt, "Chinese")

	if isChinese {
		formatted.WriteString("根据查询，获取到以下信息：\n\n")
	} else {
		formatted.WriteString("Based on the query, here's what I found:\n\n")
	}

	for i, result := range results {
		// Try to parse as JSON and extract meaningful content
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(result), &jsonData); err == nil {
			// It's JSON, try to extract meaningful fields
			if city, ok := jsonData["city"].(string); ok {
				if isChinese {
					formatted.WriteString(fmt.Sprintf("📍 位置: %s", city))
				} else {
					formatted.WriteString(fmt.Sprintf("📍 Location: %s", city))
				}
				if country, ok := jsonData["country"].(string); ok {
					formatted.WriteString(fmt.Sprintf(", %s", country))
				}
				formatted.WriteString("\n")
			}
			if title, ok := jsonData["title"].(string); ok {
				formatted.WriteString(fmt.Sprintf("📄 %s\n", title))
			}
			if mainContent, ok := jsonData["main_content"].(string); ok {
				if mainContent != "" && mainContent != "You need to enable JavaScript to run this app." {
					formatted.WriteString(fmt.Sprintf("%s\n", mainContent))
				}
			}
			if url, ok := jsonData["url"].(string); ok {
				formatted.WriteString(fmt.Sprintf("🔗 %s\n", url))
			}
		} else {
			// Not JSON, just include the text
			if len(result) > 500 {
				result = result[:500] + "..."
			}
			formatted.WriteString(fmt.Sprintf("%d. %s\n", i+1, result))
		}
		formatted.WriteString("\n")
	}

	return formatted.String()
}

// runFreeChatWithTools runs free chat with web search and fetch tools using Eino agent
func (a *App) runFreeChatWithTools(ctx context.Context, userMessage, historyContext, langPrompt string, onChunk func(string)) (string, error) {
	cfg, err := a.GetEffectiveConfig()
	if err != nil {
		return "", err
	}

	// Log proxy config status for diagnostics
	if cfg.ProxyConfig != nil {
		a.Log(fmt.Sprintf("[FREE-CHAT] Proxy config: enabled=%v, tested=%v, host=%s, port=%d, protocol=%s",
			cfg.ProxyConfig.Enabled, cfg.ProxyConfig.Tested, cfg.ProxyConfig.Host, cfg.ProxyConfig.Port, cfg.ProxyConfig.Protocol))
	} else {
		a.Log("[FREE-CHAT] Proxy config: nil (no proxy configured)")
	}

	// Initialize tools
	var webSearchTool *agent.SearchAPITool
	var webFetchTool *agent.WebFetchTool
	var timeTool *agent.TimeTool
	var locationTool *agent.LocationTool

	// Initialize time tool (always available - no external dependencies)
	timeTool = agent.NewTimeTool(a.Log)
	a.Log("[FREE-CHAT] Initialized local time tool")

	// Initialize location tool with configured fallback location
	var configLoc *agent.ConfiguredLocation
	if cfg.Location != nil && cfg.Location.City != "" {
		configLoc = &agent.ConfiguredLocation{
			Country:   cfg.Location.Country,
			City:      cfg.Location.City,
			Latitude:  cfg.Location.Latitude,
			Longitude: cfg.Location.Longitude,
		}
		a.Log(fmt.Sprintf("[FREE-CHAT] Using configured location: %s, %s", cfg.Location.City, cfg.Location.Country))
	}
	locationTool = agent.NewLocationToolWithConfig(a.Log, configLoc)
	a.Log("[FREE-CHAT] Initialized device location tool")

	// Initialize search API
	cfg.InitializeSearchAPIs()
	activeAPI := cfg.GetActiveSearchAPI()

	a.Log(fmt.Sprintf("[FREE-CHAT] Search API check: activeAPI=%v, SearchAPIs count=%d, ActiveSearchAPI=%s",
		activeAPI != nil, len(cfg.SearchAPIs), cfg.ActiveSearchAPI))

	if activeAPI != nil {
		a.Log(fmt.Sprintf("[FREE-CHAT] Active API details: ID=%s, Name=%s, Enabled=%v",
			activeAPI.ID, activeAPI.Name, activeAPI.Enabled))
	}

	if activeAPI != nil && activeAPI.Enabled {
		searchTool, err := agent.NewSearchAPITool(a.Log, activeAPI, cfg.ProxyConfig)
		if err != nil {
			a.Log(fmt.Sprintf("[FREE-CHAT] Failed to initialize search tool: %v", err))
		} else {
			webSearchTool = searchTool
			a.Log(fmt.Sprintf("[FREE-CHAT] Initialized %s search API", activeAPI.Name))
		}
	} else {
		a.Log("[FREE-CHAT] No active search API available or not enabled")
	}

	// Initialize web fetch tool
	webFetchTool = agent.NewWebFetchTool(a.Log, cfg.ProxyConfig)

	// Build tools list - time tool first (local, fast), then location, then search, then fetch
	var tools []tool.BaseTool
	if timeTool != nil {
		tools = append(tools, timeTool)
	}
	if locationTool != nil {
		tools = append(tools, locationTool)
	}
	if webSearchTool != nil {
		tools = append(tools, webSearchTool)
	}
	if webFetchTool != nil {
		tools = append(tools, webFetchTool)
	}

	// Initialize start_datasource_analysis tool (always available if data source service exists)
	var startAnalysisTool *agent.StartAnalysisTool
	if a.dataSourceService != nil {
		// Emit start-new-chat event to trigger the existing frontend analysis flow
		emitStartChat := func(dataSourceID, dataSourceName string) {
			sessionName := fmt.Sprintf("分析: %s", dataSourceName)
			a.Log(fmt.Sprintf("[FREE-CHAT] Emitting start-new-chat for datasource %s (%s)", dataSourceID, dataSourceName))
			runtime.EventsEmit(a.ctx, "start-new-chat", map[string]interface{}{
				"dataSourceId":   dataSourceID,
				"dataSourceName": dataSourceName,
				"sessionName":    sessionName,
				"keepChatOpen":   true,
			})
		}
		startAnalysisTool = agent.NewStartAnalysisTool(a.Log, a.dataSourceService, emitStartChat)
		tools = append(tools, startAnalysisTool)
		a.Log("[FREE-CHAT] Initialized start_datasource_analysis tool")
	}

	// If no tools available, fall back to simple chat
	if len(tools) == 0 {
		a.Log("[FREE-CHAT] No tools available, using simple LLM chat")
		llm := agent.NewLLMService(cfg, a.Log)
		var fullMessage string
		if historyContext != "" {
			fullMessage = fmt.Sprintf("Previous conversation:\n%s\nUser: %s\n\n(Please answer in %s)", historyContext, userMessage, langPrompt)
		} else {
			fullMessage = fmt.Sprintf("%s\n\n(Please answer in %s)", userMessage, langPrompt)
		}
		return llm.ChatStream(ctx, fullMessage, onChunk)
	}

	a.Log(fmt.Sprintf("[FREE-CHAT] Running with %d tools", len(tools)))

	// Build system prompt for agent based on available tools
	var toolDescriptions strings.Builder
	hasTimeTool := timeTool != nil
	hasLocationTool := locationTool != nil
	hasSearchTool := webSearchTool != nil
	hasFetchTool := webFetchTool != nil

	if hasTimeTool {
		toolDescriptions.WriteString("- get_local_time: Get current local time, date, weekday, or timezone. Use this for time/date questions - NO internet needed!\n")
	}
	if hasLocationTool {
		toolDescriptions.WriteString("- get_device_location: Get user's current location (city, country, coordinates). Use this for location-based queries like local weather, nearby places, etc.\n")
	}
	if hasSearchTool {
		toolDescriptions.WriteString("- web_search: Search the web for current information. Returns search results with titles, snippets, and URLs.\n")
	}
	if hasFetchTool {
		toolDescriptions.WriteString("- web_fetch: Fetch full content from a specific URL. Use this to get detailed information from URLs found in search results.\n")
	}
	hasAnalysisTool := startAnalysisTool != nil
	if hasAnalysisTool {
		toolDescriptions.WriteString("- start_datasource_analysis: List available data sources or start a data analysis session. Use when user wants to analyze data or mentions a data source name.\n")
	}

	var systemPrompt string
	if hasSearchTool {
		systemPrompt = fmt.Sprintf(`You are a helpful AI assistant with web search capability. You MUST use tools to get real-time information.

🔧 AVAILABLE TOOLS:
%s

�CRITICAL: TOOL SELECTION RULES

🌤�WEATHER �web_fetch with wttr.in (NOT web_search!)
   - "天气怎样?" �get_device_location �web_fetch("https://wttr.in/{city}?format=3")
   - "北京天气" �web_fetch("https://wttr.in/Beijing?format=3")

✈️ FLIGHTS/机票 �web_search (MUST use web_search, NOT web_fetch!)
   - "去成都的机票" �get_device_location �web_search("{出发城市} �成都 机票")
   - "北京到上海航� �web_search("北京 �上海 航班 机票")
   - "flights to Tokyo" �web_search("flights to Tokyo from {city}")

📰 NEWS/新闻 �web_search
   - "最新新� �web_search("今日新闻 头条")

📈 STOCKS/股票 �web_search
   - "苹果股价" �web_search("苹果股票价格 AAPL")

💱 EXCHANGE/汇率 �web_search
   - "美元汇率" �web_search("美元 人民�汇率")

🏨 HOTELS/酒店 �web_search
   - "附近酒店" �get_device_location �web_search("{city} 酒店推荐")

�TIME/时间 �get_local_time (NO internet needed!)
   - "现在几点?" �get_local_time(query_type="current_time")

📍 LOCATION/位置 �get_device_location
   - "我在�" �get_device_location()

🚨 CRITICAL RULES:
1. ⚠️ web_fetch is ONLY for:
   - Weather via wttr.in API
   - Reading full content from URLs found in web_search results
2. ⚠️ web_fetch CANNOT be used for flights, stocks, news, hotels - these sites need JavaScript!
3. �For flights/stocks/news/hotels �ALWAYS use web_search first!
4. NEVER say "I cannot search" - YOU HAVE web_search!
5. NEVER tell user to visit websites - get the info yourself!

📋 WORKFLOW EXAMPLES:

Example 1: "天气怎样?" / "今天几度?"
�Step 1: get_device_location (get city)
�Step 2: web_fetch(url="https://wttr.in/{city}?format=3")
�Step 3: Analyze and answer

Example 2: "去成都的机票" / "今天还有去成都的机票�"
�Step 1: get_device_location (get departure city, e.g., "San Jose")
�Step 2: web_search("San Jose �成都 机票 航班") �MUST use web_search!
�Step 3: Summarize flight options from search results

Example 3: "北京到上海航�"
�Step 1: web_search("北京 �上海 航班 机票 今天")
�Step 2: Summarize flight options

Example 4: "苹果股价"
�Step 1: web_search("苹果股票价格 AAPL 实时")
�Step 2: Report stock price from results

Example 5: "最新新�"
�Step 1: web_search("今日新闻 头条 最�")
�Step 2: Summarize top news

🎯 SUMMARY:
- Weather �web_fetch with wttr.in
- Flights/Stocks/News/Hotels �web_search (NEVER web_fetch!)
- Time �get_local_time
- Location �get_device_location
- Data analysis �start_datasource_analysis

📊 DATA ANALYSIS (HIGHEST PRIORITY):
🚨 CRITICAL: When user mentions "分析", "analyze", "analysis", or any word meaning "analyze" in ANY language, 
you MUST call start_datasource_analysis tool IMMEDIATELY. Do NOT respond with text asking for clarification.
The user is referring to a data source registered in this application.

MANDATORY WORKFLOW:
1. FIRST call start_datasource_analysis(action="list") to get available data sources
2. Match the user's request to a data source by name (fuzzy match is OK)
3. If a match is found, call start_datasource_analysis(action="start", data_source_id="<id>") to begin analysis
4. If multiple possible matches, ask user to choose from the list
5. If no match at all, show the available data sources and ask which one
6. If no data sources exist, tell the user to add a data source first

🚨 AFTER CALLING start action: The analysis will be launched in a NEW dedicated analysis session automatically.
Your response MUST be VERY SHORT, e.g. "正在为您启动�Bookshop 的分析，请稍候� or "Starting analysis for Bookshop, please wait."
Do NOT describe what the analysis will do. Do NOT list analysis steps. Do NOT explain the process.
The dedicated analysis session will handle everything �just confirm it's starting and STOP.

Examples:
- "分析bookshop2" �list �match "Bookshop2" �start analysis
- "我想分析销售数� �list �find matching one �start analysis  
- "analyze user data" �list �find matching one �start analysis
- "帮我看看bookshop2" �list �match "Bookshop2" �start analysis

⚠️ NEVER respond with generic text like "请提供更多信� when user says "分析xxx". ALWAYS call the tool first!

Please respond in %s.`, toolDescriptions.String(), langPrompt)
	} else {
		// No web search available - but time and location tools are always available
		systemPrompt = fmt.Sprintf(`You are a helpful AI assistant with local tools and limited web access.

⚠️ IMPORTANT: No search API is configured. You CANNOT search the web for real-time information.

CRITICAL RULES:
1. For TIME/DATE questions �Use get_local_time tool (instant, accurate!)
2. For LOCATION questions �Use get_device_location tool
3. For WEATHER questions �Use web_fetch with wttr.in API (FREE, works without search API!)
4. For other real-time info (news, stocks, flights, etc.) �Politely explain search API is needed
5. ⚠️ DO NOT try to use web_fetch for flights, stocks, news - these sites require JavaScript and won't work!

Available tools:
%s

=== WHAT YOU CAN DO (NO SEARCH API NEEDED) ===

�TIME/DATE: Use get_local_time
   - "现在几点?" �get_local_time(query_type="current_time")
   - "今天星期�" �get_local_time(query_type="weekday")
   - "今天几号?" �get_local_time(query_type="current_date")

�LOCATION: Use get_device_location
   - "我在�" �get_device_location()

�WEATHER: Use web_fetch with wttr.in (FREE API - plain text, no JavaScript!)
   WORKFLOW:
   1. get_device_location �get city
   2. If unavailable, use Beijing as default
   3. web_fetch(url="https://wttr.in/{city}?format=3")
   
   Examples:
   - "天气怎样?" �get_device_location, then web_fetch("https://wttr.in/{city}?format=3")
   - "北京天气" �web_fetch("https://wttr.in/Beijing?format=3")
   - "上海今天几度?" �web_fetch("https://wttr.in/Shanghai?format=3")

=== WHAT YOU CANNOT DO (NEEDS SEARCH API) ===

�The following queries require a search API to be configured:
   - 航班/Flights: "北京到上海的航班", "明天飞深�", "去成都的机票"
   - 股票/Stocks: "苹果股价", "茅台股票多少�"
   - 新闻/News: "最新新�", "今天有什么新�"
   - 酒店/Hotels: "附近酒店", "三亚酒店推荐"
   - 比赛/Sports: "今天有什么比�", "NBA比分"
   - 汇率/Exchange: "美元汇率", "人民币兑日元"

⚠️ DO NOT try to use web_fetch for these queries! Most flight/stock/news websites require JavaScript to render content, and web_fetch can only read static HTML.

When user asks for flights, stocks, news, etc., respond like this:
- Chinese: "抱歉，查询航�股票/新闻等实时信息需要配置搜索引擎。请在「设置」→「搜索API」中启用 Serper �UAPI Pro 后再试。目前我只能帮您查询天气、时间和位置信息�"
- English: "Sorry, querying flights/stocks/news requires a search API. Please enable Serper or UAPI Pro in Settings �Search API. Currently I can only help with weather, time, and location queries."

=== DATA ANALYSIS (ALWAYS AVAILABLE - HIGHEST PRIORITY) ===

🚨 CRITICAL: When user mentions "分析", "analyze", "analysis", or any word meaning "analyze" in ANY language,
you MUST call start_datasource_analysis tool IMMEDIATELY. Do NOT respond with text asking for clarification.
The user is referring to a data source registered in this application.

MANDATORY WORKFLOW:
1. FIRST call start_datasource_analysis(action="list") to get available data sources
2. Match the user's request to a data source by name (fuzzy match is OK)
3. If a match is found, call start_datasource_analysis(action="start", data_source_id="<id>") to begin analysis
4. If multiple possible matches, ask user to choose from the list
5. If no match at all, show the available data sources and ask which one
6. If no data sources exist, tell the user to add a data source first

🚨 AFTER CALLING start action: The analysis will be launched in a NEW dedicated analysis session automatically.
Your response MUST be VERY SHORT, e.g. "正在为您启动�Bookshop 的分析，请稍候� or "Starting analysis for Bookshop, please wait."
Do NOT describe what the analysis will do. Do NOT list analysis steps. Do NOT explain the process.
The dedicated analysis session will handle everything �just confirm it's starting and STOP.

Examples:
- "分析bookshop2" �list �match "Bookshop2" �start analysis
- "我想分析销售数� �list �find matching one �start analysis
- "analyze user data" �list �find matching one �start analysis

⚠️ NEVER respond with generic text like "请提供更多信� when user says "分析xxx". ALWAYS call the tool first!

Please respond in %s.`, toolDescriptions.String(), langPrompt)
	}

	// Build messages
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: systemPrompt,
		},
	}

	// Add history if available
	if historyContext != "" {
		messages = append(messages, &schema.Message{
			Role:    schema.User,
			Content: "Previous conversation context:\n" + historyContext,
		})
		messages = append(messages, &schema.Message{
			Role:    schema.Assistant,
			Content: "I understand the context. How can I help you?",
		})
	}

	// Add current user message
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: userMessage,
	})

	// Create tools node
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: tools,
	})
	if err != nil {
		a.Log(fmt.Sprintf("[FREE-CHAT] Failed to create tools node: %v", err))
		// Fallback to simple chat
		llm := agent.NewLLMService(cfg, a.Log)
		fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", userMessage, langPrompt)
		return llm.ChatStream(ctx, fullMessage, onChunk)
	}

	// Bind tools to chat model
	var toolInfos []*schema.ToolInfo
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		toolInfos = append(toolInfos, info)
		a.Log(fmt.Sprintf("[FREE-CHAT] Tool registered: %s", info.Name))
	}

	if err := a.einoService.ChatModel.BindTools(toolInfos); err != nil {
		a.Log(fmt.Sprintf("[FREE-CHAT] Failed to bind tools: %v", err))
		// Fallback to simple chat
		llm := agent.NewLLMService(cfg, a.Log)
		fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", userMessage, langPrompt)
		return llm.ChatStream(ctx, fullMessage, onChunk)
	}

	// Run agent loop (max 5 iterations to prevent infinite loops)
	maxIterations := 5
	var finalResponse strings.Builder
	var lastToolResults []string // Track last tool results for fallback

	for i := 0; i < maxIterations; i++ {
		a.Log(fmt.Sprintf("[FREE-CHAT] Agent iteration %d", i+1))

		// Call LLM
		resp, err := a.einoService.ChatModel.Generate(ctx, messages)
		if err != nil {
			a.Log(fmt.Sprintf("[FREE-CHAT] LLM generation failed at iteration %d: %v", i+1, err))
			// If we have tool results, try to summarize them
			if len(lastToolResults) > 0 {
				summaryPrompt := fmt.Sprintf("Based on the following tool results, provide a helpful summary in %s:\n\n%s\n\nUser's original question: %s",
					langPrompt, strings.Join(lastToolResults, "\n\n"), userMessage)
				llm := agent.NewLLMService(cfg, a.Log)
				summaryResp, summaryErr := llm.ChatStream(ctx, summaryPrompt, onChunk)
				if summaryErr == nil && summaryResp != "" {
					finalResponse.WriteString(summaryResp)
				} else {
					summary := a.formatToolResultsForUser(lastToolResults, langPrompt)
					onChunk(summary)
					finalResponse.WriteString(summary)
				}
				a.einoService.ChatModel.BindTools(nil)
				return finalResponse.String(), nil
			}
			// Unbind tools and fallback to simple chat
			a.einoService.ChatModel.BindTools(nil)
			a.Log("[FREE-CHAT] Falling back to simple streaming chat")
			llm := agent.NewLLMService(cfg, a.Log)
			fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", userMessage, langPrompt)
			return llm.ChatStream(ctx, fullMessage, onChunk)
		}

		a.Log(fmt.Sprintf("[FREE-CHAT] LLM response: content=%d chars, tool_calls=%d", len(resp.Content), len(resp.ToolCalls)))

		// Check if there are tool calls
		if len(resp.ToolCalls) == 0 {
			// No tool calls, this is the final response
			if resp.Content != "" {
				// Stream the final response
				onChunk(resp.Content)
				finalResponse.WriteString(resp.Content)
			} else if len(lastToolResults) > 0 {
				// LLM returned empty but we have tool results, summarize them
				summaryPrompt := fmt.Sprintf("Based on the following tool results, provide a helpful summary in %s:\n\n%s\n\nUser's original question: %s",
					langPrompt, strings.Join(lastToolResults, "\n\n"), userMessage)
				llm := agent.NewLLMService(cfg, a.Log)
				summaryResp, summaryErr := llm.ChatStream(ctx, summaryPrompt, onChunk)
				if summaryErr == nil && summaryResp != "" {
					finalResponse.WriteString(summaryResp)
				} else {
					summary := a.formatToolResultsForUser(lastToolResults, langPrompt)
					onChunk(summary)
					finalResponse.WriteString(summary)
				}
			}
			break
		}

		// If this is the last iteration and we still have tool calls,
		// we need to get a final response without tools
		if i == maxIterations-1 {
			a.Log("[FREE-CHAT] Max iterations reached, getting final response without tools")
			// Unbind tools to force a text response
			a.einoService.ChatModel.BindTools(nil)

			// Add the tool call response to messages
			messages = append(messages, resp)

			// Add a message indicating tools are no longer available
			messages = append(messages, &schema.Message{
				Role:    schema.User,
				Content: "Please provide a summary based on the information gathered so far. Do not use any tools.",
			})

			// Get final response
			finalResp, err := a.einoService.ChatModel.Generate(ctx, messages)
			if err != nil {
				a.Log(fmt.Sprintf("[FREE-CHAT] Failed to get final response: %v", err))
				// If we have tool results, try to summarize them using LLM
				if len(lastToolResults) > 0 {
					summaryPrompt := fmt.Sprintf("Based on the following tool results, provide a helpful summary in %s:\n\n%s\n\nUser's original question: %s",
						langPrompt, strings.Join(lastToolResults, "\n\n"), userMessage)
					llm := agent.NewLLMService(cfg, a.Log)
					summaryResp, summaryErr := llm.ChatStream(ctx, summaryPrompt, onChunk)
					if summaryErr == nil && summaryResp != "" {
						finalResponse.WriteString(summaryResp)
					} else {
						// Fallback to formatted tool results
						summary := a.formatToolResultsForUser(lastToolResults, langPrompt)
						onChunk(summary)
						finalResponse.WriteString(summary)
					}
				} else {
					// Fallback to simple chat
					a.Log("[FREE-CHAT] No tool results, falling back to simple chat")
					llm := agent.NewLLMService(cfg, a.Log)
					fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", userMessage, langPrompt)
					streamResp, streamErr := llm.ChatStream(ctx, fullMessage, onChunk)
					if streamErr == nil {
						finalResponse.WriteString(streamResp)
					} else {
						errorMsg := "抱歉，处理请求时遇到问题。请稍后重试�"
						onChunk(errorMsg)
						finalResponse.WriteString(errorMsg)
					}
				}
			} else if finalResp.Content != "" {
				onChunk(finalResp.Content)
				finalResponse.WriteString(finalResp.Content)
			} else {
				// LLM returned empty content
				a.Log("[FREE-CHAT] LLM returned empty content in final response")
				if len(lastToolResults) > 0 {
					// Try to generate a summary using LLM
					summaryPrompt := fmt.Sprintf("Based on the following tool results, provide a helpful summary in %s:\n\n%s\n\nUser's original question: %s",
						langPrompt, strings.Join(lastToolResults, "\n\n"), userMessage)
					llm := agent.NewLLMService(cfg, a.Log)
					summaryResp, summaryErr := llm.ChatStream(ctx, summaryPrompt, onChunk)
					if summaryErr == nil && summaryResp != "" {
						finalResponse.WriteString(summaryResp)
					} else {
						// Fallback to formatted tool results
						summary := a.formatToolResultsForUser(lastToolResults, langPrompt)
						onChunk(summary)
						finalResponse.WriteString(summary)
					}
				} else {
					errorMsg := "抱歉，无法生成回复。请尝试重新提问�"
					onChunk(errorMsg)
					finalResponse.WriteString(errorMsg)
				}
			}
			break
		}

		// Process tool calls
		messages = append(messages, resp)

		// Log tool calls (only to backend log, not to user)
		for _, toolCall := range resp.ToolCalls {
			a.Log(fmt.Sprintf("[FREE-CHAT] Executing tool: %s with args: %s", toolCall.Function.Name, toolCall.Function.Arguments))
		}

		// Execute all tools at once
		toolResults, err := toolsNode.Invoke(ctx, resp)
		if err != nil {
			a.Log(fmt.Sprintf("[FREE-CHAT] Tool execution failed: %v", err))
			// Add error as tool result for each tool call
			for _, toolCall := range resp.ToolCalls {
				messages = append(messages, &schema.Message{
					Role:       schema.Tool,
					Content:    fmt.Sprintf("Tool execution failed: %v", err),
					ToolCallID: toolCall.ID,
				})
			}
			// Don't show error to user, let LLM handle it gracefully
		} else {
			// Add all tool results to messages and track them
			for _, result := range toolResults {
				if result != nil {
					messages = append(messages, result)
					a.Log(fmt.Sprintf("[FREE-CHAT] Tool result received (length: %d)", len(result.Content)))
					// Track tool results for fallback
					if len(result.Content) > 0 && len(result.Content) < 2000 {
						lastToolResults = append(lastToolResults, result.Content)
					}
				}
			}
		}
	}

	// Unbind tools after use to avoid affecting other operations
	a.einoService.ChatModel.BindTools(nil)

	return finalResponse.String(), nil
}

// CancelAnalysis cancels the ongoing analysis for the active thread
// It sets the cancel flag and waits for the analysis to actually stop
func (a *App) CancelAnalysis() error {
	if a.chatFacadeService == nil {
		return WrapError("App", "CancelAnalysis", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.CancelAnalysis()
}

// IsCancelRequested checks if analysis cancellation has been requested
func (a *App) IsCancelRequested() bool {
	if a.chatFacadeService == nil {
		return false
	}
	return a.chatFacadeService.IsCancelRequested()
}

// GetActiveThreadID returns the currently active thread ID
func (a *App) GetActiveThreadID() string {
	if a.chatFacadeService == nil {
		return ""
	}
	return a.chatFacadeService.GetActiveThreadID()
}

// GetActiveAnalysisCount returns the current number of active analysis sessions
func (a *App) GetActiveAnalysisCount() int {
	if a.chatFacadeService == nil {
		return 0
	}
	return a.chatFacadeService.GetActiveAnalysisCount()
}

// CanStartNewAnalysis checks if a new analysis can be started based on concurrent limit
func (a *App) CanStartNewAnalysis() (bool, string) {
	if a.chatFacadeService == nil {
		return false, "chat facade service not initialized"
	}
	return a.chatFacadeService.CanStartNewAnalysis()
}

// attachChartToUserMessage attaches chart data to a specific user message in a thread
func (a *App) attachChartToUserMessage(threadID, messageID string, chartData *ChartData) {
	if a.chatFacadeService == nil {
		return
	}
	a.chatFacadeService.attachChartToUserMessage(threadID, messageID, chartData)
}



// cleanEChartsJSON removes JavaScript function expressions from ECharts JSON strings
// so they can be parsed by Go's json.Unmarshal. LLMs sometimes generate ECharts configs
// with function(params){...} for formatter/color/label which is valid JS but not valid JSON.
func cleanEChartsJSON(jsonStr string) string {
	// Match patterns like: "formatter": function(params) { ... }
	// Need to handle nested braces inside function bodies
	result := jsonStr

	// Use a manual approach to handle nested braces properly
	// Find "function" keyword that appears as a value (after : )
	for {
		// Find pattern: "key": function or , "key": function
		idx := strings.Index(result, "function")
		if idx < 0 {
			break
		}

		// Check if this looks like a JSON value (preceded by : with optional whitespace)
		// Walk backwards to find the colon
		prefixStart := idx - 1
		for prefixStart >= 0 && (result[prefixStart] == ' ' || result[prefixStart] == '\t' || result[prefixStart] == '\n' || result[prefixStart] == '\r') {
			prefixStart--
		}
		if prefixStart < 0 || result[prefixStart] != ':' {
			// Not a JSON value context, skip past this "function"
			// Replace this occurrence temporarily to avoid infinite loop
			result = result[:idx] + "FUNC_SKIP" + result[idx+8:]
			continue
		}

		// Find the opening brace of the function body
		braceStart := strings.Index(result[idx:], "{")
		if braceStart < 0 {
			break
		}
		braceStart += idx

		// Count braces to find the matching closing brace
		depth := 0
		braceEnd := -1
		for i := braceStart; i < len(result); i++ {
			if result[i] == '{' {
				depth++
			} else if result[i] == '}' {
				depth--
				if depth == 0 {
					braceEnd = i
					break
				}
			}
		}
		if braceEnd < 0 {
			break
		}

		// Now walk backwards from the colon to find the key start (including comma if present)
		// We want to remove: , "key": function(...){...} or "key": function(...){...},
		removeStart := prefixStart // at the colon
		// Walk back past the key name and quotes
		keyStart := removeStart - 1
		for keyStart >= 0 && (result[keyStart] == ' ' || result[keyStart] == '\t' || result[keyStart] == '\n' || result[keyStart] == '\r') {
			keyStart--
		}
		// Walk past the quoted key name
		if keyStart >= 0 && result[keyStart] == '"' {
			keyStart-- // past closing quote
			for keyStart >= 0 && result[keyStart] != '"' {
				keyStart--
			}
			if keyStart > 0 {
				keyStart-- // past opening quote
				// Check for leading comma
				for keyStart >= 0 && (result[keyStart] == ' ' || result[keyStart] == '\t' || result[keyStart] == '\n' || result[keyStart] == '\r') {
					keyStart--
				}
				if keyStart >= 0 && result[keyStart] == ',' {
					removeStart = keyStart
				} else {
					removeStart = keyStart + 1
				}
			}
		}

		// Remove the entire key-value pair
		after := result[braceEnd+1:]
		// If the next non-whitespace char after removal is a comma, and we didn't remove a leading comma, clean it
		trimmedAfter := strings.TrimLeft(after, " \t\n\r")
		if len(trimmedAfter) > 0 && trimmedAfter[0] == ',' && removeStart > 0 && result[removeStart] != ',' {
			// Remove the trailing comma too
			after = trimmedAfter[1:]
		}
		result = result[:removeStart] + after
	}

	// Restore any FUNC_SKIP markers (shouldn't normally happen in valid echarts)
	result = strings.ReplaceAll(result, "FUNC_SKIP", "function")

	// Clean up trailing commas before } or ]
	result = reTrailingComma.ReplaceAllString(result, "$1")

	return result
}


// SelectDirectory opens a directory dialog
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Data Cache Directory",
	})
}

// SelectFolder opens a directory dialog with a custom title
func (a *App) SelectFolder(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

// GetPythonEnvironments returns detected Python environments
func (a *App) GetPythonEnvironments() []agent.PythonEnvironment {
	if a.pythonFacadeService == nil {
		return nil
	}
	return a.pythonFacadeService.GetPythonEnvironments()
}

// ValidatePython checks the given Python path
func (a *App) ValidatePython(path string) agent.PythonValidationResult {
	if a.pythonFacadeService == nil {
		return agent.PythonValidationResult{Valid: false, Version: "", MissingPackages: []string{}}
	}
	return a.pythonFacadeService.ValidatePython(path)
}

// InstallPythonPackages installs missing packages for the given Python environment
func (a *App) InstallPythonPackages(pythonPath string, packages []string) error {
	if a.pythonFacadeService == nil {
		return WrapError("App", "InstallPythonPackages", fmt.Errorf("python facade service not initialized"))
	}
	return a.pythonFacadeService.InstallPythonPackages(pythonPath, packages)
}

// CreateVantagicsEnvironment creates a dedicated virtual environment for Vantagics
func (a *App) CreateVantagicsEnvironment() (string, error) {
	if a.pythonFacadeService == nil {
		return "", WrapError("App", "CreateVantagicsEnvironment", fmt.Errorf("python facade service not initialized"))
	}
	return a.pythonFacadeService.CreateVantagicsEnvironment()
}

// CheckVantagicsEnvironmentExists checks if a vantagics environment already exists
func (a *App) CheckVantagicsEnvironmentExists() bool {
	if a.pythonFacadeService == nil {
		return false
	}
	return a.pythonFacadeService.CheckVantagicsEnvironmentExists()
}

// DiagnosePythonInstallation provides detailed diagnostic information about Python installations
func (a *App) DiagnosePythonInstallation() map[string]interface{} {
	if a.pythonFacadeService == nil {
		return map[string]interface{}{"error": "python facade service not initialized"}
	}
	return a.pythonFacadeService.DiagnosePythonInstallation()
}

// SetupUvEnvironment creates a uv virtual environment and installs required packages.
// It also persists the pythonPath to config so the agent can find it.
func (a *App) SetupUvEnvironment() (string, error) {
	if a.pythonFacadeService == nil {
		return "", WrapError("App", "SetupUvEnvironment", fmt.Errorf("python facade service not initialized"))
	}
	pythonPath, err := a.pythonFacadeService.SetupUvEnvironment()
	if err != nil {
		return "", err
	}
	// Auto-persist pythonPath to config
	if pythonPath != "" {
		if cfg, cfgErr := a.GetConfig(); cfgErr == nil && cfg.PythonPath != pythonPath {
			cfg.PythonPath = pythonPath
			_ = a.SaveConfig(cfg)
		}
	}
	return pythonPath, nil
}

// autoSetupUvEnvironment checks if the uv environment is ready and sets it up in background if not.
// It fully manages the Python environment: create venv, install missing packages, and persist config.
func (a *App) autoSetupUvEnvironment(cfg *config.Config) {
	if a.pythonService == nil {
		return
	}

	var pythonPath string

	if a.pythonService.IsUvEnvironmentReady() {
		a.Log("[STARTUP] uv environment already ready")
		pythonPath = a.pythonService.GetUvEnvironmentPythonPath()

		// Check and auto-install missing packages for existing environment
		status := a.pythonService.GetUvEnvironmentStatus()
		if len(status.MissingPackages) > 0 {
			a.Log(fmt.Sprintf("[STARTUP] Auto-installing missing packages: %v", status.MissingPackages))
			if err := a.pythonService.InstallMissingPackages(pythonPath, status.MissingPackages); err != nil {
				a.Log(fmt.Sprintf("[STARTUP] Auto-install packages failed: %v", err))
			} else {
				a.Log("[STARTUP] All missing packages installed successfully")
			}
		}
	} else if !a.pythonService.IsUvAvailable() {
		a.Log("[STARTUP] uv not available, skipping auto-setup")
		return
	} else {
		a.Log("[STARTUP] Auto-setting up uv environment in background...")
		var err error
		pythonPath, err = a.pythonService.SetupUvEnvironment()
		if err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Auto uv environment setup failed: %v", err))
			return
		}
		a.Log(fmt.Sprintf("[STARTUP] Auto uv environment setup complete: %s", pythonPath))
	}

	// Ensure PythonPath is persisted in config so agent can find it
	if pythonPath == "" {
		return
	}
	// Wait for startup to finish before calling SaveConfig, which rebuilds EinoService
	<-a.startupDone
	currentCfg, err := a.GetConfig()
	if err != nil {
		a.Log(fmt.Sprintf("[STARTUP] Failed to get config for PythonPath update: %v", err))
		return
	}
	if currentCfg.PythonPath == pythonPath {
		return
	}
	currentCfg.PythonPath = pythonPath
	if err := a.SaveConfig(currentCfg); err != nil {
		a.Log(fmt.Sprintf("[STARTUP] Failed to save PythonPath to config: %v", err))
	} else {
		a.Log(fmt.Sprintf("[STARTUP] PythonPath auto-configured: %s", pythonPath))
	}
}


// GetUvEnvironmentStatus returns the current status of the uv environment
func (a *App) GetUvEnvironmentStatus() agent.UvEnvironmentStatus {
	if a.pythonFacadeService == nil {
		return agent.UvEnvironmentStatus{}
	}
	return a.pythonFacadeService.GetUvEnvironmentStatus()
}

// GetChatHistory loads the chat history
func (a *App) GetChatHistory() ([]ChatThread, error) {
	if a.chatFacadeService == nil {
		return nil, WrapError("App", "GetChatHistory", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.GetChatHistory()
}

// GetChatHistoryByDataSource loads chat history for a specific data source
func (a *App) GetChatHistoryByDataSource(dataSourceID string) ([]ChatThread, error) {
	if a.chatFacadeService == nil {
		return nil, WrapError("App", "GetChatHistoryByDataSource", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.GetChatHistoryByDataSource(dataSourceID)
}

// CheckSessionNameExists checks if a session name already exists for a data source
func (a *App) CheckSessionNameExists(dataSourceID string, sessionName string, excludeThreadID string) (bool, error) {
	if a.chatFacadeService == nil {
		return false, WrapError("App", "CheckSessionNameExists", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.CheckSessionNameExists(dataSourceID, sessionName, excludeThreadID)
}

// SaveChatHistory saves the chat history
func (a *App) SaveChatHistory(threads []ChatThread) error {
	if a.chatFacadeService == nil {
		return WrapError("App", "SaveChatHistory", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.SaveChatHistory(threads)
}

// DeleteThread deletes a specific chat thread
func (a *App) DeleteThread(threadID string) error {
	if a.chatFacadeService == nil {
		return WrapError("App", "DeleteThread", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.DeleteThread(threadID)
}

// CreateChatThread creates a new chat thread with a unique title
func (a *App) CreateChatThread(dataSourceID, title string) (ChatThread, error) {
	if a.chatFacadeService == nil {
		return ChatThread{}, WrapError("App", "CreateChatThread", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.CreateChatThread(dataSourceID, title)
}

func (a *App) generateAnalysisSuggestions(threadID string, analysis *agent.DataSourceAnalysis) {
	if a.dataSourceFacadeService == nil {
		return
	}
	a.dataSourceFacadeService.GenerateAnalysisSuggestions(threadID, analysis)
}

// parseSuggestionsToInsights extracts numbered suggestions from LLM response and converts to Insight objects
func (a *App) parseSuggestionsToInsights(llmResponse, dataSourceID, dataSourceName string) []Insight {
	if a.dataSourceFacadeService == nil {
		return nil
	}
	return a.dataSourceFacadeService.parseSuggestionsToInsights(llmResponse, dataSourceID, dataSourceName)
}

func (a *App) analyzeDataSource(dataSourceID string) {
	if a.dataSourceFacadeService == nil {
		return
	}
	a.dataSourceFacadeService.analyzeDataSource(dataSourceID)
}

// UpdateThreadTitle updates the title of a chat thread
func (a *App) UpdateThreadTitle(threadID, newTitle string) (string, error) {
	if a.chatFacadeService == nil {
		return "", WrapError("App", "UpdateThreadTitle", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.UpdateThreadTitle(threadID, newTitle)
}

// ClearHistory clears all chat history
func (a *App) ClearHistory() error {
	if a.chatFacadeService == nil {
		return WrapError("App", "ClearHistory", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.ClearHistory()
}

// ClearThreadMessages clears all messages from a thread but keeps the thread itself
func (a *App) ClearThreadMessages(threadID string) error {
	if a.chatFacadeService == nil {
		return WrapError("App", "ClearThreadMessages", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.ClearThreadMessages(threadID)
}

// --- Data Source Management ---

// GetDataSources returns the list of registered data sources
func (a *App) GetDataSources() ([]agent.DataSource, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "GetDataSources", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.GetDataSources()
}

// GetDataSourceStatistics returns aggregated statistics about all data sources
// Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5
func (a *App) GetDataSourceStatistics() (*agent.DataSourceStatistics, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "GetDataSourceStatistics", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.GetDataSourceStatistics()
}

// StartDataSourceAnalysis initiates analysis for a specific data source
// Returns the analysis session/thread ID
// Validates: Requirements 4.1, 4.2, 4.5
func (a *App) StartDataSourceAnalysis(dataSourceID string) (string, error) {
	if a.dataSourceFacadeService == nil {
		return "", WrapError("App", "StartDataSourceAnalysis", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.StartDataSourceAnalysis(dataSourceID)
}

// ImportExcelDataSource imports an Excel file as a data source
func (a *App) ImportExcelDataSource(name string, filePath string) (*agent.DataSource, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "ImportExcelDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.ImportExcelDataSource(name, filePath)
}

// ImportCSVDataSource imports a CSV directory as a data source
func (a *App) ImportCSVDataSource(name string, dirPath string) (*agent.DataSource, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "ImportCSVDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.ImportCSVDataSource(name, dirPath)
}

// ImportJSONDataSource imports a JSON file as a data source
func (a *App) ImportJSONDataSource(name string, filePath string) (*agent.DataSource, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "ImportJSONDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.ImportJSONDataSource(name, filePath)
}

// ShopifyOAuthConfig holds the Shopify OAuth configuration
type ShopifyOAuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// GetShopifyOAuthConfig returns the Shopify OAuth configuration
// Developer should set these values
func (a *App) GetShopifyOAuthConfig() ShopifyOAuthConfig {
	if a.dataSourceFacadeService == nil {
		return ShopifyOAuthConfig{}
	}
	return a.dataSourceFacadeService.GetShopifyOAuthConfig()
}

// StartShopifyOAuth initiates the Shopify OAuth flow
// Returns the authorization URL that should be opened in browser
func (a *App) StartShopifyOAuth(shop string) (string, error) {
	if a.dataSourceFacadeService == nil {
		return "", WrapError("App", "StartShopifyOAuth", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.StartShopifyOAuth(shop)
}

// WaitForShopifyOAuth waits for the OAuth flow to complete
// Returns the access token and shop URL on success
func (a *App) WaitForShopifyOAuth() (map[string]string, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "WaitForShopifyOAuth", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.WaitForShopifyOAuth()
}

// CancelShopifyOAuth cancels the ongoing OAuth flow
func (a *App) CancelShopifyOAuth() {
	if a.dataSourceFacadeService == nil {
		return
	}
	a.dataSourceFacadeService.CancelShopifyOAuth()
}

// OpenShopifyOAuthInBrowser opens the Shopify OAuth URL in the default browser
func (a *App) OpenShopifyOAuthInBrowser(url string) {
	if a.dataSourceFacadeService == nil {
		return
	}
	a.dataSourceFacadeService.OpenShopifyOAuthInBrowser(url)
}

// AddDataSource adds a new data source with generic configuration
func (a *App) AddDataSource(name string, driverType string, config map[string]string) (*agent.DataSource, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "AddDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.AddDataSource(name, driverType, config)
}

// DeleteDataSource deletes a data source
func (a *App) DeleteDataSource(id string) error {
	if a.dataSourceFacadeService == nil {
		return WrapError("App", "DeleteDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.DeleteDataSource(id)
}

// RefreshEcommerceDataSource performs incremental update for e-commerce data sources
// Returns the refresh result with information about new data fetched
func (a *App) RefreshEcommerceDataSource(id string) (*agent.RefreshResult, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "RefreshEcommerceDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.RefreshEcommerceDataSource(id)
}

// IsEcommerceDataSource checks if a data source type supports incremental refresh
func (a *App) IsEcommerceDataSource(dsType string) bool {
	if a.dataSourceFacadeService == nil {
		return false
	}
	return a.dataSourceFacadeService.IsEcommerceDataSource(dsType)
}

// JiraProject represents a Jira project for selection
type JiraProject struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	ID   string `json:"id"`
}

// GetJiraProjects fetches available projects from Jira using provided credentials
// This allows users to select which project(s) to import
func (a *App) GetJiraProjects(instanceType, baseUrl, username, apiToken string) ([]JiraProject, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "GetJiraProjects", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.GetJiraProjects(instanceType, baseUrl, username, apiToken)
}

// IsRefreshableDataSource checks if a data source type supports incremental refresh
// This includes both e-commerce platforms and project management tools like Jira
func (a *App) IsRefreshableDataSource(dsType string) bool {
	if a.dataSourceFacadeService == nil {
		return false
	}
	return a.dataSourceFacadeService.IsRefreshableDataSource(dsType)
}

// RefreshDataSource performs incremental update for supported data sources
// Works for both e-commerce platforms and Jira
func (a *App) RefreshDataSource(id string) (*agent.RefreshResult, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "RefreshDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.RefreshDataSource(id)
}

// RenameDataSource renames a data source
func (a *App) RenameDataSource(id string, newName string) error {
	if a.dataSourceFacadeService == nil {
		return WrapError("App", "RenameDataSource", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.RenameDataSource(id, newName)
}

// DeleteTable removes a table from a data source
func (a *App) DeleteTable(id string, tableName string) error {
	if a.dataSourceFacadeService == nil {
		return WrapError("App", "DeleteTable", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.DeleteTable(id, tableName)
}

// RenameColumn renames a column in a table
func (a *App) RenameColumn(id string, tableName string, oldColumnName string, newColumnName string) error {
	if a.dataSourceFacadeService == nil {
		return WrapError("App", "RenameColumn", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.RenameColumn(id, tableName, oldColumnName, newColumnName)
}

// DeleteColumn deletes a column from a table
func (a *App) DeleteColumn(id string, tableName string, columnName string) error {
	if a.dataSourceFacadeService == nil {
		return WrapError("App", "DeleteColumn", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.DeleteColumn(id, tableName, columnName)
}

// UpdateMySQLExportConfig updates the MySQL export configuration for a data source
func (a *App) UpdateMySQLExportConfig(id string, host, port, user, password, database string) error {
	if a.dataSourceFacadeService == nil {
		return WrapError("App", "UpdateMySQLExportConfig", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.UpdateMySQLExportConfig(id, host, port, user, password, database)
}

// GetDataSourceTables returns all table names for a data source
func (a *App) GetDataSourceTables(id string) ([]string, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "GetDataSourceTables", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.GetDataSourceTables(id)
}

// GetDataSourceTableData returns preview data for a table
func (a *App) GetDataSourceTableData(id string, tableName string) ([]map[string]interface{}, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "GetDataSourceTableData", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.GetDataSourceTableData(id, tableName)
}

// GetDataSourceTableCount returns the total number of rows in a table
func (a *App) GetDataSourceTableCount(id string, tableName string) (int, error) {
	if a.dataSourceFacadeService == nil {
		return 0, WrapError("App", "GetDataSourceTableCount", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.GetDataSourceTableCount(id, tableName)
}

// TableDataWithCount holds table preview data and total row count
type TableDataWithCount struct {
	Data     []map[string]interface{} `json:"data"`
	RowCount int                      `json:"rowCount"`
}

// GetDataSourceTableDataWithCount returns preview data and row count in a single DB connection.
// This avoids DuckDB concurrent access lock conflicts.
func (a *App) GetDataSourceTableDataWithCount(id string, tableName string) (*TableDataWithCount, error) {
	if a.dataSourceFacadeService == nil {
		return nil, WrapError("App", "GetDataSourceTableDataWithCount", fmt.Errorf("datasource facade service not initialized"))
	}
	return a.dataSourceFacadeService.GetDataSourceTableDataWithCount(id, tableName)
}

// SelectExcelFile opens a file dialog to select an Excel file

func (a *App) SelectExcelFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Excel File",
		Filters: []runtime.FileFilter{
			// Support both .xlsx (Excel 2007+) and .xls (Excel 97-2003) formats
			{DisplayName: "Excel Files", Pattern: "*.xlsx;*.xls;*.xlsm"},
		},
	})
}

// SelectCSVFile opens a file dialog to select a CSV file
func (a *App) SelectCSVFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select CSV File",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
}

// SelectJSONFile opens a file dialog to select a JSON file
func (a *App) SelectJSONFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select JSON File",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
}

// SelectSaveFile opens a save file dialog

func (a *App) SelectSaveFile(filename string, filterPattern string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: filename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Files", Pattern: filterPattern},
		},
	})
}

// ExportToCSV exports one or more data source tables to CSV

func (a *App) ExportToCSV(id string, tableNames []string, outputPath string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportToCSV", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportToCSV(id, tableNames, outputPath)
}

// ExportToJSON exports one or more data source tables to JSON
func (a *App) ExportToJSON(id string, tableNames []string, outputPath string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportToJSON", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportToJSON(id, tableNames, outputPath)
}

// ExportToSQL exports one or more data source tables to SQL
func (a *App) ExportToSQL(id string, tableNames []string, outputPath string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportToSQL", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportToSQL(id, tableNames, outputPath)
}

// ExportToExcel exports one or more data source tables to Excel (.xlsx)
func (a *App) ExportToExcel(id string, tableNames []string, outputPath string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportToExcel", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportToExcel(id, tableNames, outputPath)
}

// ExportToMySQL exports one or more data source tables to MySQL
func (a *App) ExportToMySQL(id string, tableNames []string, host, port, user, password, database string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "ExportToMySQL", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.ExportToMySQL(id, tableNames, host, port, user, password, database)
}

// TestMySQLConnection tests the connection to a MySQL server
func (a *App) TestMySQLConnection(host, port, user, password string) error {
	if a.exportFacadeService == nil {
		return WrapError("App", "TestMySQLConnection", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.TestMySQLConnection(host, port, user, password)
}

// GetMySQLDatabases returns a list of databases from the MySQL server
func (a *App) GetMySQLDatabases(host, port, user, password string) ([]string, error) {
	if a.exportFacadeService == nil {
		return nil, WrapError("App", "GetMySQLDatabases", fmt.Errorf("export facade service not initialized"))
	}
	return a.exportFacadeService.GetMySQLDatabases(host, port, user, password)
}

// ShowMessage displays a message dialog (non-modal via frontend)
func (a *App) ShowMessage(typeStr string, title string, message string) {
	runtime.EventsEmit(a.ctx, "show-message-modal", map[string]string{
		"type":    typeStr,
		"title":   title,
		"message": message,
	})
}

// --- Error Knowledge Management ---

// ErrorKnowledgeSummary represents the summary of error knowledge
type ErrorKnowledgeSummary struct {
	TotalRecords    int            `json:"total_records"`
	SuccessfulCount int            `json:"successful_count"`
	SuccessRate     float64        `json:"success_rate"`
	ByType          map[string]int `json:"by_type"`
	RecentErrors    []ErrorRecord  `json:"recent_errors"`
}

// ErrorRecord represents an error record for frontend display
type ErrorRecord struct {
	Timestamp    string `json:"timestamp"`
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
	Context      string `json:"context"`
	Solution     string `json:"solution"`
	Successful   bool   `json:"successful"`
}

// GetErrorKnowledgeSummary returns a summary of the error knowledge base
func (a *App) GetErrorKnowledgeSummary() (*ErrorKnowledgeSummary, error) {
	if a.einoService == nil {
		return nil, fmt.Errorf("eino service not initialized")
	}

	ek := a.einoService.GetErrorKnowledge()
	if ek == nil {
		return nil, fmt.Errorf("error knowledge not available")
	}

	// Get recent errors
	recentRecords := ek.GetRecentErrors(10)

	// Convert to frontend format
	frontendRecords := make([]ErrorRecord, len(recentRecords))
	typeCounts := make(map[string]int)
	successCount := 0

	for i, rec := range recentRecords {
		frontendRecords[i] = ErrorRecord{
			Timestamp:    time.UnixMilli(rec.Timestamp).Format("2006-01-02 15:04:05"),
			ErrorType:    rec.ErrorType,
			ErrorMessage: rec.ErrorMessage,
			Context:      rec.Context,
			Solution:     rec.Solution,
			Successful:   rec.Successful,
		}
		typeCounts[rec.ErrorType]++
		if rec.Successful {
			successCount++
		}
	}

	totalRecords := len(recentRecords)
	successRate := 0.0
	if totalRecords > 0 {
		successRate = float64(successCount) / float64(totalRecords) * 100.0
	}

	summary := &ErrorKnowledgeSummary{
		TotalRecords:    totalRecords,
		SuccessfulCount: successCount,
		SuccessRate:     successRate,
		ByType:          typeCounts,
		RecentErrors:    frontendRecords,
	}

	return summary, nil
}

// --- Session File Management ---

// GetSessionFiles returns the list of files generated during a session
func (a *App) GetSessionFiles(threadID string) ([]SessionFile, error) {
	if a.chatFacadeService == nil {
		return nil, WrapError("App", "GetSessionFiles", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.GetSessionFiles(threadID)
}

// GetSessionFilePath returns the full path to a session file
func (a *App) GetSessionFilePath(threadID, fileName string) (string, error) {
	if a.chatFacadeService == nil {
		return "", WrapError("App", "GetSessionFilePath", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.GetSessionFilePath(threadID, fileName)
}

// OpenSessionFile opens a session file in the default application
func (a *App) OpenSessionFile(threadID, fileName string) error {
	if a.chatFacadeService == nil {
		return WrapError("App", "OpenSessionFile", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.OpenSessionFile(threadID, fileName)
}

// OpenExternalURL opens a URL in the system's default browser
func (a *App) OpenExternalURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// DeleteSessionFile deletes a specific file from a session
func (a *App) DeleteSessionFile(threadID, fileName string) error {
	if a.chatFacadeService == nil {
		return WrapError("App", "DeleteSessionFile", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.DeleteSessionFile(threadID, fileName)
}

// associateNewFilesWithMessage updates newly created files to associate them with a specific message
func (a *App) associateNewFilesWithMessage(threadID, messageID string, existingFiles map[string]bool) error {
	if a.chatFacadeService == nil {
		return WrapError("App", "associateNewFilesWithMessage", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.AssociateNewFilesWithMessage(threadID, messageID, existingFiles)
}

// OpenSessionResultsDirectory opens the session's results directory in the file explorer
func (a *App) OpenSessionResultsDirectory(threadID string) error {
	if a.chatFacadeService == nil {
		return WrapError("App", "OpenSessionResultsDirectory", fmt.Errorf("chat facade service not initialized"))
	}
	return a.chatFacadeService.OpenSessionResultsDirectory(threadID)
}

// --- Skills Management ---

// SkillInfo represents skill information for frontend
type SkillInfo struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Version         string   `json:"version"`
	Author          string   `json:"author"`
	Category        string   `json:"category"`
	Keywords        []string `json:"keywords"`
	RequiredColumns []string `json:"required_columns"`
	Tools           []string `json:"tools"`
	Enabled         bool     `json:"enabled"`
	Icon            string   `json:"icon"`
	Tags            []string `json:"tags"`
}

// GetSkills returns all loaded skills
func (a *App) GetSkills() ([]SkillInfo, error) {
	if a.skillFacadeService == nil {
		return nil, WrapError("App", "GetSkills", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.GetSkills()
}

// GetEnabledSkills returns only enabled skills
func (a *App) GetEnabledSkills() ([]SkillInfo, error) {
	if a.skillFacadeService == nil {
		return nil, WrapError("App", "GetEnabledSkills", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.GetEnabledSkills()
}

// GetSkillCategories returns all skill categories
func (a *App) GetSkillCategories() ([]string, error) {
	if a.skillFacadeService == nil {
		return nil, WrapError("App", "GetSkillCategories", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.GetSkillCategories()
}

// EnableSkill enables a skill by ID
func (a *App) EnableSkill(skillID string) error {
	if a.skillFacadeService == nil {
		return WrapError("App", "EnableSkill", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.EnableSkill(skillID)
}

// DisableSkill disables a skill by ID
func (a *App) DisableSkill(skillID string) error {
	if a.skillFacadeService == nil {
		return WrapError("App", "DisableSkill", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.DisableSkill(skillID)
}

// DeleteSkill deletes a skill by ID (removes directory and config)
func (a *App) DeleteSkill(skillID string) error {
	if a.skillFacadeService == nil {
		return WrapError("App", "DeleteSkill", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.DeleteSkill(skillID)
}

// ReloadSkills reloads all skills from disk
func (a *App) ReloadSkills() error {
	if a.skillFacadeService == nil {
		return WrapError("App", "ReloadSkills", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.ReloadSkills()
}

// --- Metrics JSON Management ---

// SaveMetricsJson saves metrics JSON data for a specific message
func (a *App) SaveMetricsJson(messageId string, metricsJson string) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "SaveMetricsJson", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.SaveMetricsJson(messageId, metricsJson)
}

// LoadMetricsJson loads metrics JSON data for a specific message
func (a *App) LoadMetricsJson(messageId string) (string, error) {
	if a.analysisFacadeService == nil {
		return "", WrapError("App", "LoadMetricsJson", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.LoadMetricsJson(messageId)
}

// ExtractMetricsFromAnalysis automatically extracts key metrics from analysis results
func (a *App) ExtractMetricsFromAnalysis(threadID string, messageId string, analysisContent string) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "ExtractMetricsFromAnalysis", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.ExtractMetricsFromAnalysis(threadID, messageId, analysisContent)
}

// tryExtractMetrics attempts to extract metrics using LLM
func (a *App) tryExtractMetrics(threadID string, messageId string, prompt string, attempt int) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "tryExtractMetrics", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.tryExtractMetrics(threadID, messageId, prompt, attempt)
}

// getConfigForExtraction gets config for metrics extraction
func (a *App) getConfigForExtraction() config.Config {
	if a.analysisFacadeService == nil {
		cfg, _ := a.GetEffectiveConfig()
		return cfg
	}
	return a.analysisFacadeService.getConfigForExtraction()
}

// ExtractSuggestionsFromAnalysis extracts next-step suggestions from analysis response
// and emits them to the dashboard insights area
func (a *App) ExtractSuggestionsFromAnalysis(threadID, userMessageID, analysisContent string) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "ExtractSuggestionsFromAnalysis", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.ExtractSuggestionsFromAnalysis(threadID, userMessageID, analysisContent)
}

// extractSuggestionInsights is the shared extraction logic for suggestions.
func (a *App) extractSuggestionInsights(analysisContent string) []Insight {
	if a.analysisFacadeService == nil {
		return nil
	}
	return a.analysisFacadeService.extractSuggestionInsights(analysisContent)
}

// ExtractSuggestionsAsItems extracts suggestions from analysis response,
// emits them to the frontend, and returns them as AnalysisResultItems for persistence.
func (a *App) ExtractSuggestionsAsItems(threadID, userMessageID, analysisContent string) []AnalysisResultItem {
	if a.analysisFacadeService == nil {
		return nil
	}
	return a.analysisFacadeService.ExtractSuggestionsAsItems(threadID, userMessageID, analysisContent)
}

// containsNumber checks if a string contains any digit
func containsNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// extractJSONFromResponse extracts JSON array from LLM response
func (a *App) extractJSONFromResponse(response string) string {
	if a.analysisFacadeService == nil {
		return ""
	}
	return a.analysisFacadeService.extractJSONFromResponse(response)
}

// fallbackTextExtraction uses regex patterns as fallback when LLM extraction fails
func (a *App) fallbackTextExtraction(messageId string, content string) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "fallbackTextExtraction", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.fallbackTextExtraction(messageId, content)
}

// SaveSessionRecording saves the current session's analysis recording to a file
func (a *App) SaveSessionRecording(threadID, title, description string) (string, error) {
	if a.analysisFacadeService == nil {
		return "", WrapError("App", "SaveSessionRecording", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.SaveSessionRecording(threadID, title, description)
}

// GetSessionRecordings returns all available session recordings
func (a *App) GetSessionRecordings() ([]agent.AnalysisRecording, error) {
	if a.analysisFacadeService == nil {
		return nil, WrapError("App", "GetSessionRecordings", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.GetSessionRecordings()
}

// ReplayAnalysisRecording replays a recorded analysis on a target data source
func (a *App) ReplayAnalysisRecording(recordingID, targetSourceID string, autoFixFields bool, maxFieldDiff int) (*agent.ReplayResult, error) {
	if a.analysisFacadeService == nil {
		return nil, WrapError("App", "ReplayAnalysisRecording", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.ReplayAnalysisRecording(recordingID, targetSourceID, autoFixFields, maxFieldDiff)
}

// --- Dashboard Drag-Drop Layout Wails Bridge Methods ---

// SaveLayout saves a layout configuration to the database (Task 5.1)
func (a *App) SaveLayout(config database.LayoutConfiguration) error {
	if a.dashboardFacadeService == nil {
		return WrapError("App", "SaveLayout", fmt.Errorf("dashboard facade service not initialized"))
	}
	return a.dashboardFacadeService.SaveLayout(config)
}

// LoadLayout loads a layout configuration from the database
func (a *App) LoadLayout(userID string) (*database.LayoutConfiguration, error) {
	if a.dashboardFacadeService == nil {
		return nil, WrapError("App", "LoadLayout", fmt.Errorf("dashboard facade service not initialized"))
	}
	return a.dashboardFacadeService.LoadLayout(userID)
}

// CheckComponentHasData checks if a component has data available
func (a *App) CheckComponentHasData(componentType string, instanceID string) (bool, error) {
	if a.dashboardFacadeService == nil {
		return false, WrapError("App", "CheckComponentHasData", fmt.Errorf("dashboard facade service not initialized"))
	}
	return a.dashboardFacadeService.CheckComponentHasData(componentType, instanceID)
}

// GetFilesByCategory retrieves files for a specific category
func (a *App) GetFilesByCategory(category string) ([]database.FileInfo, error) {
	if a.dashboardFacadeService == nil {
		return nil, WrapError("App", "GetFilesByCategory", fmt.Errorf("dashboard facade service not initialized"))
	}
	return a.dashboardFacadeService.GetFilesByCategory(category)
}

// DownloadFile returns the file path for download
func (a *App) DownloadFile(fileID string) (string, error) {
	if a.dashboardFacadeService == nil {
		return "", WrapError("App", "DownloadFile", fmt.Errorf("dashboard facade service not initialized"))
	}
	return a.dashboardFacadeService.DownloadFile(fileID)
}

// ExportDashboard exports dashboard data with component filtering
func (a *App) ExportDashboard(req database.ExportRequest) (*database.ExportResult, error) {
	if a.dashboardFacadeService == nil {
		return nil, WrapError("App", "ExportDashboard", fmt.Errorf("dashboard facade service not initialized"))
	}
	return a.dashboardFacadeService.ExportDashboard(req)
}

// ListSkills returns all installed skills
func (a *App) ListSkills() ([]agent.Skill, error) {
	if a.skillFacadeService == nil {
		return nil, WrapError("App", "ListSkills", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.ListSkills()
}

// InstallSkillsFromZip installs skills from a ZIP file
// Opens a file dialog for the user to select a ZIP file
func (a *App) InstallSkillsFromZip() ([]string, error) {
	if a.skillFacadeService == nil {
		return nil, WrapError("App", "InstallSkillsFromZip", fmt.Errorf("skill facade service not initialized"))
	}
	return a.skillFacadeService.InstallSkillsFromZip()
}

// detectAnalysisType detects the type of analysis from the response
// Used for recording analysis history (Requirement 1.1)
func (a *App) detectAnalysisType(response string) string {
	if a.analysisFacadeService == nil {
		return "statistical"
	}
	return a.analysisFacadeService.detectAnalysisType(response)
}

// extractKeyFindings extracts key findings from the analysis response
// Used for recording analysis history (Requirement 1.1)
func (a *App) extractKeyFindings(response string) string {
	if a.analysisFacadeService == nil {
		return ""
	}
	return a.analysisFacadeService.extractKeyFindings(response)
}

// extractTargetColumns extracts target columns mentioned in the analysis
// Used for recording analysis history (Requirement 1.1)
func (a *App) extractTargetColumns(response string, availableColumns []string) []string {
	if a.analysisFacadeService == nil {
		return nil
	}
	return a.analysisFacadeService.extractTargetColumns(response, availableColumns)
}

// recordAnalysisHistory records analysis completion for intent enhancement
// Used for recording analysis history (Requirement 1.1)
func (a *App) recordAnalysisHistory(dataSourceID string, record agent.AnalysisRecord) {
	if a.analysisFacadeService == nil {
		return
	}
	a.analysisFacadeService.recordAnalysisHistory(dataSourceID, record)
}

// AddAnalysisRecord adds an analysis record for intent enhancement
// This is a wrapper that delegates to the AnalysisFacadeService
// Validates: Requirement 1.1
func (a *App) AddAnalysisRecord(dataSourceID string, record agent.AnalysisRecord) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "AddAnalysisRecord", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.AddAnalysisRecord(dataSourceID, record)
}

// RecordIntentSelection records user's intent selection for preference learning
// This is called from the frontend when a user selects an intent
// Validates: Requirement 2.1, 5.1
func (a *App) RecordIntentSelection(threadID string, intent IntentSuggestion) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "RecordIntentSelection", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.RecordIntentSelection(threadID, intent)
}

// GetMessageAnalysisData retrieves analysis data for a specific message (for dashboard restoration)
// Resolves any file:// references in chart data and analysis results before returning
func (a *App) GetMessageAnalysisData(threadID, messageID string) (map[string]interface{}, error) {
	if a.analysisFacadeService == nil {
		return nil, WrapError("App", "GetMessageAnalysisData", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.GetMessageAnalysisData(threadID, messageID)
}

// ShowStepResultOnDashboard re-pushes a step's analysis results to the dashboard via EventAggregator.
func (a *App) ShowStepResultOnDashboard(threadID string, messageID string) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "ShowStepResultOnDashboard", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.ShowStepResultOnDashboard(threadID, messageID)
}

// ShowAllSessionResults 将整个会话的所有分析结果一次性推送到仪表盘�
func (a *App) ShowAllSessionResults(threadID string) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "ShowAllSessionResults", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.ShowAllSessionResults(threadID)
}

// Pre-compiled regexes for extractStepDescriptionFromContent
var (
	reAnalysisRequestLine = regexp.MustCompile(`📋 分析请求[：:](.+)`)
	reStepHeader          = regexp.MustCompile(`步骤\s+\d+\s+\(([^)]+)\)`)
)

// Pre-compiled regexes for GetAgentMemory long-term extraction (avoid recompiling per call)
var (
	reTablePattern   = regexp.MustCompile(`(?i)(?:table|from|join)\s+["\x60]?(\w+)["\x60]?`)
	reInsightPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:发现|found|shows?|indicates?|suggests?|reveals?)[:：\s]+(.{20,100})`),
		regexp.MustCompile(`(?i)(?:结论|conclusion|result|总结)[�\s]+(.{20,100})`),
		regexp.MustCompile(`(?i)(?:趋势|trend|pattern|规律)[�\s]+(.{20,100})`),
	}
	reNumPattern      = regexp.MustCompile(`(\d+(?:\.\d+)?%|\d{1,3}(?:,\d{3})+|\d+(?:\.\d+)?\s*(?:万|亿|million|billion|k|M|B))`)
	reBoldListItem    = regexp.MustCompile(`^\d*[.�]\s*\*\*(.+?)\*\*`)
	reTrailingComma   = regexp.MustCompile(`,(\s*[}\]])`)
)


// extractStepDescriptionFromContent extracts step description from message content.
// It first tries to extract from "📋 分析请求� line, then falls back to step header "步骤 N (描述)".
func extractStepDescriptionFromContent(content string) string {
	// First try to extract from "📋 分析请求� line
	matches := reAnalysisRequestLine.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	// Fallback: extract from step header "步骤 N (描述)"
	matches = reStepHeader.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// SaveMessageAnalysisResults saves analysis results for a specific message
func (a *App) SaveMessageAnalysisResults(threadID, messageID string, results []AnalysisResultItem) error {
	if a.analysisFacadeService == nil {
		return WrapError("App", "SaveMessageAnalysisResults", fmt.Errorf("analysis facade service not initialized"))
	}
	return a.analysisFacadeService.SaveMessageAnalysisResults(threadID, messageID, results)
}

// ============ License Activation Methods ============

// ActivationResult represents the result of license activation
type ActivationResult struct {
	Success          bool   `json:"success"`
	Code             string `json:"code"`
	Message          string `json:"message"`
	ExpiresAt        string `json:"expires_at,omitempty"`
	SwitchedToOSS    bool   `json:"switched_to_oss,omitempty"`    // True if switched to open source mode
}

// ActivateLicense activates the application with a license server
func (a *App) ActivateLicense(serverURL, sn string) (*ActivationResult, error) {
	if a.licenseFacadeService == nil {
		return nil, WrapError("App", "ActivateLicense", fmt.Errorf("license facade service not initialized"))
	}
	return a.licenseFacadeService.ActivateLicense(serverURL, sn)
}

// RequestSNResult represents the result of SN request
type RequestSNResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	SN      string `json:"sn,omitempty"`
	Code    string `json:"code,omitempty"`
}

// RequestSN requests a serial number from the license server
func (a *App) RequestSN(serverURL, email string) (*RequestSNResult, error) {
	if a.licenseFacadeService == nil {
		return nil, WrapError("App", "RequestSN", fmt.Errorf("license facade service not initialized"))
	}
	return a.licenseFacadeService.RequestSN(serverURL, email)
}

// RequestFreeSN requests a permanent free serial number from the license server
func (a *App) RequestFreeSN(serverURL, email string) (*RequestSNResult, error) {
	if a.licenseFacadeService == nil {
		return nil, WrapError("App", "RequestFreeSN", fmt.Errorf("license facade service not initialized"))
	}
	return a.licenseFacadeService.RequestFreeSN(serverURL, email)
}

// RequestOpenSourceSN requests an open source serial number from the license server
func (a *App) RequestOpenSourceSN(serverURL, email string) (*RequestSNResult, error) {
	if a.licenseFacadeService == nil {
		return nil, WrapError("App", "RequestOpenSourceSN", fmt.Errorf("license facade service not initialized"))
	}
	return a.licenseFacadeService.RequestOpenSourceSN(serverURL, email)
}

// IsPermanentFreeMode returns true if the current activation has trust_level "permanent_free"
func (a *App) IsPermanentFreeMode() bool {
	if a.licenseFacadeService == nil {
		return false
	}
	return a.licenseFacadeService.IsPermanentFreeMode()
}

// IsOpenSourceMode returns true if the current activation has trust_level "open_source"
func (a *App) IsOpenSourceMode() bool {
	if a.licenseFacadeService == nil {
		return false
	}
	return a.licenseFacadeService.IsOpenSourceMode()
}

// GetActivationStatus returns the current activation status
func (a *App) GetActivationStatus() map[string]interface{} {
	if a.licenseFacadeService == nil {
		return map[string]interface{}{
			"activated": false,
		}
	}
	return a.licenseFacadeService.GetActivationStatus()
}

// CheckLicenseActivationFailed returns true if license activation failed during startup
func (a *App) CheckLicenseActivationFailed() bool {
	if a.licenseFacadeService == nil {
		return a.licenseActivationFailed
	}
	return a.licenseFacadeService.CheckLicenseActivationFailed()
}

// GetLicenseActivationError returns the license activation error message
func (a *App) GetLicenseActivationError() string {
	if a.licenseFacadeService == nil {
		return a.licenseActivationError
	}
	return a.licenseFacadeService.GetLicenseActivationError()
}

// LoadSavedActivation attempts to load saved activation data from local storage
func (a *App) LoadSavedActivation(sn string) (*ActivationResult, error) {
	if a.licenseFacadeService == nil {
		return nil, WrapError("App", "LoadSavedActivation", fmt.Errorf("license facade service not initialized"))
	}
	return a.licenseFacadeService.LoadSavedActivation(sn)
}

// GetActivatedLLMConfig returns the LLM config from activation (for internal use)
func (a *App) GetActivatedLLMConfig() *agent.ActivationData {
	if a.licenseFacadeService == nil {
		return nil
	}
	return a.licenseFacadeService.GetActivatedLLMConfig()
}

// HasActiveAnalysis checks if there are any active analysis sessions
func (a *App) HasActiveAnalysis() bool {
	if a.licenseFacadeService == nil {
		if a.chatFacadeService == nil {
			return false
		}
		return a.chatFacadeService.HasActiveAnalysis()
	}
	return a.licenseFacadeService.HasActiveAnalysis()
}

// DeactivateLicense clears the activation
func (a *App) DeactivateLicense() error {
	if a.licenseFacadeService == nil {
		return WrapError("App", "DeactivateLicense", fmt.Errorf("license facade service not initialized"))
	}
	return a.licenseFacadeService.DeactivateLicense()
}

// RefreshLicense refreshes the license from server using stored SN
func (a *App) RefreshLicense() (*ActivationResult, error) {
	if a.licenseFacadeService == nil {
		return nil, WrapError("App", "RefreshLicense", fmt.Errorf("license facade service not initialized"))
	}
	return a.licenseFacadeService.RefreshLicense()
}

// IsLicenseActivated returns true if license is activated
func (a *App) IsLicenseActivated() bool {
	if a.licenseFacadeService == nil {
		return a.licenseClient != nil && a.licenseClient.IsActivated()
	}
	return a.licenseFacadeService.IsLicenseActivated()
}

// MarkdownTableData represents a parsed markdown table
type MarkdownTableData struct {
	Title   string                   `json:"title"`
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

// extractMarkdownTablesFromText extracts standard markdown tables from text
// Markdown tables have the format:
// | Header1 | Header2 |
// |---------|---------|
// | Value1  | Value2  |
// It also extracts table titles from preceding lines (headers or bold text)
func extractMarkdownTablesFromText(text string) []MarkdownTableData {
	var tables []MarkdownTableData

	lines := strings.Split(text, "\n")
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check if this line looks like a markdown table header (starts and ends with |)
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			// Check if next line is a separator (|---|---|)
			if i+1 < len(lines) {
				sepLine := strings.TrimSpace(lines[i+1])
				if isMarkdownTableSeparator(sepLine) {
					// Look for table title in preceding lines
					title := extractTableTitle(lines, i)
					
					// Found a table, parse it
					table := parseMarkdownTableFromLines(lines, i)
					table.Title = title
					if len(table.Rows) > 0 {
						tables = append(tables, table)
					}
					// Skip past the table
					for i < len(lines) {
						l := strings.TrimSpace(lines[i])
						if !strings.HasPrefix(l, "|") || !strings.HasSuffix(l, "|") {
							break
						}
						i++
					}
					continue
				}
			}
		}
		i++
	}

	return tables
}

// extractTableTitle looks for a table title in the lines preceding the table
// It searches for markdown headers (###, ##, #) or bold text (**title**)
func extractTableTitle(lines []string, tableStartIdx int) string {
	// Search up to 3 lines before the table for a title
	for j := tableStartIdx - 1; j >= 0 && j >= tableStartIdx-3; j-- {
		line := strings.TrimSpace(lines[j])
		if line == "" {
			continue
		}
		
		// Skip if it's a table line (shouldn't happen, but safety check)
		if strings.HasPrefix(line, "|") {
			continue
		}
		
		// Check for markdown headers: ### Title, ## Title, # Title
		if strings.HasPrefix(line, "#") {
			// Remove leading # and spaces
			title := strings.TrimLeft(line, "#")
			title = strings.TrimSpace(title)
			if title != "" {
				return title
			}
		}
		
		// Check for bold text: **Title** or **Title**：description
		if strings.HasPrefix(line, "**") {
			// Extract text between ** markers
			endIdx := strings.Index(line[2:], "**")
			if endIdx > 0 {
				title := line[2 : 2+endIdx]
				title = strings.TrimSpace(title)
				if title != "" {
					return title
				}
			}
		}
		
		// Check for numbered list with bold: 1. **Title** or **1.** Title
		if matches := reBoldListItem.FindStringSubmatch(line); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
		
		// If we found a non-empty, non-table line that's not a recognized title format, stop searching
		// This prevents picking up unrelated text as titles
		break
	}
	
	return ""
}

// isMarkdownTableSeparator checks if a line is a markdown table separator (|---|---|)
func isMarkdownTableSeparator(line string) bool {
	// Trim spaces first to handle trailing spaces
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return false
	}
	// Remove pipes and check if remaining content is dashes, colons, and spaces
	inner := strings.Trim(line, "|")
	parts := strings.Split(inner, "|")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Each part should be like "---", ":---", "---:", or ":---:"
		cleaned := strings.Trim(part, ":-")
		if cleaned != "" {
			return false
		}
	}
	return true
}

// parseMarkdownTableFromLines parses a markdown table starting at the given line index
func parseMarkdownTableFromLines(lines []string, startIdx int) MarkdownTableData {
	table := MarkdownTableData{
		Title:   "",
		Columns: []string{},
		Rows:    []map[string]interface{}{},
	}

	if startIdx >= len(lines) {
		return table
	}

	// Parse header row
	headerLine := strings.TrimSpace(lines[startIdx])
	headers := parseMarkdownTableRowCells(headerLine)
	if len(headers) == 0 {
		return table
	}

	table.Columns = headers

	// Skip separator line
	dataStartIdx := startIdx + 2

	// Parse data rows
	for i := dataStartIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			break
		}
		// Skip if it's another separator line
		if isMarkdownTableSeparator(line) {
			continue
		}

		cells := parseMarkdownTableRowCells(line)
		row := make(map[string]interface{})
		for j, header := range headers {
			if j < len(cells) {
				row[header] = cells[j]
			} else {
				row[header] = ""
			}
		}
		table.Rows = append(table.Rows, row)
	}

	return table
}

// parseMarkdownTableRowCells splits a markdown table row into cells
func parseMarkdownTableRowCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

// extractJSONObjectKeysOrdered extracts keys from the first JSON object in an array,
// preserving the original order from the JSON string.
// For input like `[{"name":"a","age":1},...]`, returns ["name","age"].
func extractJSONObjectKeysOrdered(jsonStr string) []string {
	dec := json.NewDecoder(strings.NewReader(jsonStr))
	// Expect opening '['
	t, err := dec.Token()
	if err != nil {
		return nil
	}
	if delim, ok := t.(json.Delim); !ok || delim != '[' {
		return nil
	}
	// Expect opening '{' of first object
	t, err = dec.Token()
	if err != nil {
		return nil
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil
	}
	// Read key-value pairs from first object
	var keys []string
	for dec.More() {
		t, err = dec.Token()
		if err != nil {
			break
		}
		if key, ok := t.(string); ok {
			keys = append(keys, key)
			// Skip the value
			var v json.RawMessage
			if err := dec.Decode(&v); err != nil {
				break
			}
		}
	}
	return keys
}
