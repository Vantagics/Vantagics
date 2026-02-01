package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	gort "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"vantagedata/agent"
	"vantagedata/config"
	"vantagedata/database"
	"vantagedata/logger"

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

// IsValid 检查意图建议是否有效
// 验证所有必需字段都非空
// Returns true if all required fields (ID, Title, Description, Icon, Query) are non-empty
// Validates: Requirements 1.2 (意图建议结构完整性)
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

// App struct
type App struct {
	ctx                      context.Context
	chatService              *ChatService
	pythonService            *agent.PythonService
	dataSourceService        *agent.DataSourceService
	memoryService            *agent.MemoryService
	workingContextManager    *agent.WorkingContextManager
	analysisPathManager      *agent.AnalysisPathManager
	preferenceLearner          *agent.PreferenceLearner
	intentEnhancementService   *agent.IntentEnhancementService
	intentUnderstandingService *agent.IntentUnderstandingService
	einoService                *agent.EinoService
	skillService             *agent.SkillService
	searchKeywordsManager    *agent.SearchKeywordsManager
	db                       *sql.DB
	storageDir               string
	logger                   *logger.Logger
	activeThreads            map[string]bool // Track active analysis sessions by thread ID
	activeThreadsMutex       sync.RWMutex    // Protect activeThreads map
	isChatOpen               bool
	cancelAnalysisMutex      sync.Mutex
	cancelAnalysis           bool
	activeThreadID           string
	// Dashboard drag-drop layout services
	layoutService *database.LayoutService
	dataService   *database.DataService
	fileService   *database.FileService
	exportService *database.ExportService
	// Event aggregator for analysis results
	eventAggregator *EventAggregator
	// License client for activation
	licenseClient *agent.LicenseClient
}

// AgentMemoryView structure for frontend
type AgentMemoryView struct {
	LongTerm   []string `json:"long_term"`
	MediumTerm []string `json:"medium_term"`
	ShortTerm  []string `json:"short_term"`
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		pythonService: agent.NewPythonService(),
		logger:        logger.NewLogger(),
		activeThreads: make(map[string]bool),
		isChatOpen:    false,
	}
}

// SetChatOpen updates the chat open state
func (a *App) SetChatOpen(isOpen bool) {
	a.isChatOpen = isOpen
}

// ShowAbout displays the about dialog
func (a *App) ShowAbout() {
	cfg, _ := a.GetConfig()

	var title, message string
	if cfg.Language == "简体中文" {
		title = "关于 观界"
		message = "观界 (VantageData)\n\n" +
			"观数据之界，见商业全貌。\n\n" +
			"版本：1.0.0\n" +
			"© 2026 VantageData. All rights reserved."
	} else {
		title = "About VantageData"
		message = "VantageData\n\n" +
			"See Beyond Data. Master Your Vantage.\n\n" +
			"Version: 1.0.0\n" +
			"© 2026 VantageData. All rights reserved."
	}

	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

// OpenDevTools opens the developer tools/console
func (a *App) OpenDevTools() {
	// Wails v2 doesn't have direct API to open DevTools
	// Show instructions to the user
	cfg, _ := a.GetConfig()

	var title, message string
	if cfg.Language == "简体中文" {
		title = "打开开发者工具"
		message = "请使用以下方法打开开发者工具：\n\n" +
			"方法1：按 F12 键\n" +
			"方法2：按 Ctrl+Shift+I\n" +
			"方法3：按 Ctrl+Shift+J\n" +
			"方法4：在空白区域右键点击，选择\"检查\"\n\n" +
			"如果以上方法都不行，请在开发模式下运行：\n" +
			"wails dev"
	} else {
		title = "Open Developer Tools"
		message = "Please use one of the following methods to open DevTools:\n\n" +
			"Method 1: Press F12\n" +
			"Method 2: Press Ctrl+Shift+I\n" +
			"Method 3: Press Ctrl+Shift+J\n" +
			"Method 4: Right-click in empty area and select 'Inspect'\n\n" +
			"If none of these work, run in development mode:\n" +
			"wails dev"
	}

	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

// onBeforeClose is called when the application is about to close
func (a *App) onBeforeClose(ctx context.Context) (prevent bool) {
	// Check if cancellation was requested - if so, wait a moment for cleanup
	a.cancelAnalysisMutex.Lock()
	cancelRequested := a.cancelAnalysis
	a.cancelAnalysisMutex.Unlock()

	if cancelRequested {
		// Wait briefly for the cancellation to complete
		a.Log("[CLOSE-DIALOG] Cancel was requested, waiting for cleanup...")
		time.Sleep(500 * time.Millisecond)
	}

	// Only prevent close if there's an active analysis running
	a.activeThreadsMutex.RLock()
	hasActiveAnalysis := len(a.activeThreads) > 0
	a.activeThreadsMutex.RUnlock()

	if hasActiveAnalysis {
		// Get current language configuration
		cfg, _ := a.GetConfig()

		var title, message, yesButton, noButton string
		if cfg.Language == "简体中文" {
			title = "确认退出"
			message = "当前有正在进行的分析任务，确定要退出吗？\n\n退出将中断分析过程。"
			yesButton = "退出"
			noButton = "取消"
		} else {
			title = "Confirm Exit"
			message = "There is an analysis task in progress. Are you sure you want to exit?\n\nExiting will interrupt the analysis."
			yesButton = "Exit"
			noButton = "Cancel"
		}

		dialog, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         title,
			Message:       message,
			Buttons:       []string{noButton, yesButton}, // 取消按钮在前，退出按钮在后
			DefaultButton: noButton,
			CancelButton:  noButton,
		})

		if err != nil {
			// 如果对话框出错，阻止关闭以保护用户数据
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
	return false // 没有分析任务，允许关闭
}

// shutdown is called when the application is closing to clean up resources
func (a *App) shutdown(ctx context.Context) {
	// Close database connection
	if a.db != nil {
		a.db.Close()
		a.Log("[SHUTDOWN] Database connection closed")
	}
	// Close EinoService (which closes Python pool)
	if a.einoService != nil {
		a.einoService.Close()
	}
	// Close logger
	a.logger.Close()
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Store App instance in context for system tray access
	ctx = context.WithValue(ctx, "app", a)
	a.ctx = ctx

	// Start system tray (Windows/Linux only, handled by build tags)
	runSystray(ctx)

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

		// Initialize database with migrations
		db, err := database.InitDB(dataDir)
		if err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Failed to initialize database: %v", err))
		} else {
			a.db = db
			a.Log("[STARTUP] Database initialized successfully")
		}

		// Initialize working context manager for UI state tracking
		a.workingContextManager = agent.NewWorkingContextManager(dataDir)
		a.Log("[STARTUP] Working context manager initialized")

		// Initialize analysis path manager for storyline tracking
		a.analysisPathManager = agent.NewAnalysisPathManager(dataDir)
		a.Log("[STARTUP] Analysis path manager initialized")

		// Initialize preference learner for user behavior tracking
		a.preferenceLearner = agent.NewPreferenceLearner(dataDir)
		a.Log("[STARTUP] Preference learner initialized")

		// Initialize intent enhancement service for improved intent understanding
		a.intentEnhancementService = agent.NewIntentEnhancementService(
			dataDir,
			a.preferenceLearner,
			a.memoryService,
			a.Log,
		)
		if err := a.intentEnhancementService.Initialize(); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Intent enhancement service initialization warning: %v", err))
		} else {
			a.Log("[STARTUP] Intent enhancement service initialized successfully")
		}

		// Initialize new intent understanding service (simplified architecture)
		// Validates: Requirements 7.1
		a.intentUnderstandingService = agent.NewIntentUnderstandingService(
			dataDir,
			a.dataSourceService,
			a.Log,
		)
		if err := a.intentUnderstandingService.Initialize(); err != nil {
			a.Log(fmt.Sprintf("[STARTUP] Intent understanding service initialization warning: %v", err))
		} else {
			a.Log("[STARTUP] Intent understanding service initialized successfully")
		}

		// Initialize skill service for skills management
		a.skillService = agent.NewSkillService(dataDir, a.Log)
		a.Log("[STARTUP] Skill service initialized")

		// Initialize search keywords manager for intelligent web search detection
		a.searchKeywordsManager = agent.NewSearchKeywordsManager(dataDir, a.Log)
		a.Log("[STARTUP] Search keywords manager initialized")

		// Initialize license client and try auto-activation if SN is saved
		a.licenseClient = agent.NewLicenseClient(a.Log)
		if cfg.LicenseSN != "" && cfg.LicenseServerURL != "" {
			a.Log("[STARTUP] Found saved license SN, attempting auto-activation...")
			_, err := a.licenseClient.Activate(cfg.LicenseServerURL, cfg.LicenseSN)
			if err != nil {
				a.Log(fmt.Sprintf("[STARTUP] Auto-activation failed: %v", err))
			} else {
				a.Log("[STARTUP] License auto-activated successfully")
				// Update config with activated LLM settings
				if activationData := a.licenseClient.GetData(); activationData != nil && activationData.LLMAPIKey != "" {
					// Map license server LLM types to the expected provider names
					llmType := activationData.LLMType
					baseURL := activationData.LLMBaseURL
					switch strings.ToLower(llmType) {
					case "openai":
						llmType = "OpenAI"
					case "anthropic":
						llmType = "Anthropic"
					case "gemini":
						llmType = "Gemini"
					case "deepseek":
						llmType = "OpenAI-Compatible"
						if baseURL == "" {
							baseURL = "https://api.deepseek.com"
						}
					case "openai-compatible":
						llmType = "OpenAI-Compatible"
					case "claude-compatible":
						llmType = "Claude-Compatible"
					}
					cfg.LLMProvider = llmType
					cfg.APIKey = activationData.LLMAPIKey
					cfg.BaseURL = baseURL
					cfg.ModelName = activationData.LLMModel
					a.Log(fmt.Sprintf("[STARTUP] Using activated LLM config: provider=%s (mapped from %s), model=%s, baseURL=%s",
						cfg.LLMProvider, activationData.LLMType, cfg.ModelName, cfg.BaseURL))
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
		if a.db != nil {
			a.fileService = database.NewFileService(a.db, dataDir)
			a.Log("[STARTUP] FileService initialized successfully")

			a.dataService = database.NewDataService(a.db, dataDir, a.fileService)
			// Set the data source service to avoid circular dependency
			if a.dataSourceService != nil {
				a.dataService.SetDataSourceService(a.dataSourceService)
			}
			a.Log("[STARTUP] DataService initialized successfully")

			a.layoutService = database.NewLayoutService(a.db)
			a.Log("[STARTUP] LayoutService initialized successfully")

			a.exportService = database.NewExportService(a.dataService, a.layoutService)
			a.Log("[STARTUP] ExportService initialized successfully")
		}

		// Initialize event aggregator for analysis results
		a.eventAggregator = NewEventAggregator(ctx)
		a.Log("[STARTUP] EventAggregator initialized successfully")
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

	// Get the full conversation history
	threads, _ := a.chatService.LoadThreads()
	var messages []ChatMessage
	for _, t := range threads {
		if t.ID == threadID {
			messages = t.Messages
			break
		}
	}

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
				if strings.HasPrefix(lowerContent, "请") ||
					strings.HasPrefix(lowerContent, "您") ||
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
				} else if idx := strings.Index(content, "。"); idx > 0 && idx < 500 {
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
				mediumTerm = append(mediumTerm, fmt.Sprintf("  • %s", q))
			}
		}

		if len(assistantFindings) > 0 {
			mediumTerm = append(mediumTerm, fmt.Sprintf("💡 Key findings: %d responses", len(assistantFindings)))
			for i, f := range assistantFindings {
				if i >= 3 {
					mediumTerm = append(mediumTerm, fmt.Sprintf("  ... and %d more findings", len(assistantFindings)-3))
					break
				}
				mediumTerm = append(mediumTerm, fmt.Sprintf("  • %s", f))
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
			mediumTerm = append([]string{"📚 AI 自动生成的对话摘要:"}, mediumTerm...)
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

	tablePattern := regexp.MustCompile(`(?i)(?:table|from|join)\s+["\x60]?(\w+)["\x60]?`)
	insightPatterns := []string{
		`(?i)(?:发现|found|shows?|indicates?|suggests?|reveals?)[：:\s]+(.{20,100})`,
		`(?i)(?:结论|conclusion|result|总结)[：:\s]+(.{20,100})`,
		`(?i)(?:趋势|trend|pattern|规律)[：:\s]+(.{20,100})`,
	}

	seenTables := make(map[string]bool)
	for _, msg := range messages {
		content := msg.Content

		// Extract mentioned tables
		tableMatches := tablePattern.FindAllStringSubmatch(content, -1)
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
			for _, pattern := range insightPatterns {
				re := regexp.MustCompile(pattern)
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
			numPattern := regexp.MustCompile(`(\d+(?:\.\d+)?%|\d{1,3}(?:,\d{3})+|\d+(?:\.\d+)?\s*(?:万|亿|million|billion|k|M|B))`)
			numMatches := numPattern.FindAllString(content, 5)
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
		longTerm = append(longTerm, fmt.Sprintf("📊 涉及数据表: %s", strings.Join(mentionedTables, ", ")))
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
			longTerm = append([]string{"🗄️ 持久化知识库:"}, longTerm...)
		}

		// Global data sources (cross-session knowledge)
		if len(globalDataSources) > 0 {
			longTerm = append(longTerm, "\n📊 全局数据源:")
			for _, mem := range globalDataSources {
				longTerm = append(longTerm, fmt.Sprintf("  • %s", mem))
			}
		}

		// Global goals (overall objectives)
		if len(globalGoals) > 0 {
			longTerm = append(longTerm, "\n🎯 全局目标:")
			for _, mem := range globalGoals {
				longTerm = append(longTerm, fmt.Sprintf("  • %s", mem))
			}
		}

		// Session long-term (persistent facts for this session)
		if len(sessionLong) > 0 {
			longTerm = append(longTerm, "\n📌 会话持久化事实:")
			for _, mem := range sessionLong {
				longTerm = append(longTerm, fmt.Sprintf("  • %s", mem))
			}
		}
	}

	// If nothing substantive found, show a meaningful message
	if len(longTerm) == 0 {
		longTerm = append(longTerm, "暂无提取到的持久化知识。")
		longTerm = append(longTerm, "")
		longTerm = append(longTerm, "💡 长期记忆会自动从以下内容中提取：")
		longTerm = append(longTerm, "  • 数据源架构（表名、字段名）")
		longTerm = append(longTerm, "  • 业务规则和定义")
		longTerm = append(longTerm, "  • 数据特征（枚举值、状态类型）")
		longTerm = append(longTerm, "")
		longTerm = append(longTerm, "继续对话和分析后，系统将自动提取和保存这些知识。")
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
	defer zipWriter.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, srcFile)
	return err
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
	return filepath.Join(home, "VantageData"), nil
}

func (a *App) getConfigPath() (string, error) {
	dir, err := a.getStorageDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// GetConfig loads the config from the ~/VantageData/config.json
func (a *App) GetConfig() (config.Config, error) {
	path, err := a.getConfigPath()
	if err != nil {
		return config.Config{}, err
	}

	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, "VantageData")

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
		if activationData != nil && activationData.LLMAPIKey != "" {
			// Map license server LLM types to the expected provider names
			llmType := activationData.LLMType
			baseURL := activationData.LLMBaseURL
			switch strings.ToLower(llmType) {
			case "openai":
				llmType = "OpenAI"
			case "anthropic":
				llmType = "Anthropic"
			case "gemini":
				llmType = "Gemini"
			case "deepseek":
				// DeepSeek uses OpenAI-compatible API
				llmType = "OpenAI-Compatible"
				// Set default BaseURL for DeepSeek if not provided
				if baseURL == "" {
					baseURL = "https://api.deepseek.com"
				}
			case "openai-compatible":
				llmType = "OpenAI-Compatible"
			case "claude-compatible":
				llmType = "Claude-Compatible"
			}
			cfg.LLMProvider = llmType
			cfg.APIKey = activationData.LLMAPIKey
			cfg.BaseURL = baseURL
			cfg.ModelName = activationData.LLMModel
		}

		// Merge activated search config if available
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

// SaveConfig saves the config to the ~/VantageData/config.json
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

	// Save the configuration file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	// Reinitialize services that depend on configuration
	a.reinitializeServices(cfg)

	// Update window title based on language
	a.updateWindowTitle(cfg.Language)

	// Update application menu based on language
	a.UpdateApplicationMenu(cfg.Language)

	// Notify frontend that configuration has been updated
	runtime.EventsEmit(a.ctx, "config-updated")

	a.Log("Configuration saved and services reinitialized")
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
	
	// Get stats from logger (vantagedata_*.log files)
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
	
	// Cleanup vantagedata_*.log files
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
	if language == "简体中文" {
		title = "观界 - 智能数据分析"
	} else {
		title = "VantageData - Smart Data Analysis"
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

// reinitializeServices reinitializes services that depend on configuration
func (a *App) reinitializeServices(cfg config.Config) {
	// Check if we have activated license with LLM config
	if a.licenseClient != nil && a.licenseClient.IsActivated() {
		activationData := a.licenseClient.GetData()
		if activationData != nil && activationData.LLMAPIKey != "" {
			// Use activated LLM config
			a.Log("[REINIT] Using activated license LLM configuration")
			// Map license server LLM types to the expected provider names
			llmType := activationData.LLMType
			baseURL := activationData.LLMBaseURL
			switch strings.ToLower(llmType) {
			case "openai":
				llmType = "OpenAI"
			case "anthropic":
				llmType = "Anthropic"
			case "gemini":
				llmType = "Gemini"
			case "deepseek":
				llmType = "OpenAI-Compatible"
				if baseURL == "" {
					baseURL = "https://api.deepseek.com"
				}
			case "openai-compatible":
				llmType = "OpenAI-Compatible"
			case "claude-compatible":
				llmType = "Claude-Compatible"
			}
			cfg.LLMProvider = llmType
			cfg.APIKey = activationData.LLMAPIKey
			cfg.BaseURL = baseURL
			cfg.ModelName = activationData.LLMModel
			a.Log(fmt.Sprintf("[REINIT] Mapped LLM config: Provider=%s, Model=%s, BaseURL=%s", cfg.LLMProvider, cfg.ModelName, cfg.BaseURL))
		}
	}
	
	// Reinitialize MemoryService if configuration changed
	if a.memoryService != nil { // Original condition, keeping it as the provided `oldPath != path` is out of context.
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
		}
	}

	// Note: LLMService is created fresh for each request in SendMessage, so no reinitialization needed
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
	// Check if we have activated license with LLM config
	if a.licenseClient != nil && a.licenseClient.IsActivated() {
		activationData := a.licenseClient.GetData()
		if activationData != nil && activationData.LLMAPIKey != "" {
			// Use activated LLM config
			a.Log("[TEST-LLM] Using activated license LLM configuration")
			cfg.LLMProvider = activationData.LLMType
			cfg.APIKey = activationData.LLMAPIKey
			cfg.BaseURL = activationData.LLMBaseURL
			cfg.ModelName = activationData.LLMModel
		}
	}
	
	llm := agent.NewLLMService(cfg, a.Log)
	_, err := llm.Chat(a.ctx, "hi LLM, I'm just test the connection. Just answer ok to me without other infor.")
	if err != nil {
		return ConnectionResult{
			Success: false,
			Message: err.Error(),
		}
	}

	return ConnectionResult{
		Success: true,
		Message: "Connection successful!",
	}
}

// TestMCPService tests the connection to an MCP service
func (a *App) TestMCPService(url string) ConnectionResult {
	if url == "" {
		return ConnectionResult{
			Success: false,
			Message: "MCP service URL is required",
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try to make a simple GET request to check if the service is reachable
	resp, err := client.Get(url)
	if err != nil {
		return ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to connect: %v", err),
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ConnectionResult{
			Success: true,
			Message: fmt.Sprintf("MCP service is reachable (HTTP %d)", resp.StatusCode),
		}
	}

	// Even if status is not 2xx, if we got a response, the service is reachable
	return ConnectionResult{
		Success: true,
		Message: fmt.Sprintf("MCP service responded (HTTP %d)", resp.StatusCode),
	}
}

// TestSearchEngine tests if a search engine is accessible
func (a *App) TestSearchEngine(url string) ConnectionResult {
	if url == "" {
		return ConnectionResult{
			Success: false,
			Message: "Search engine URL is required",
		}
	}

	// Ensure URL has protocol
	testURL := url
	if !strings.HasPrefix(testURL, "http://") && !strings.HasPrefix(testURL, "https://") {
		testURL = "https://" + testURL
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects
			return nil
		},
	}

	// Try to make a HEAD request first (faster)
	resp, err := client.Head(testURL)
	if err != nil {
		// Try GET if HEAD fails
		resp, err = client.Get(testURL)
		if err != nil {
			return ConnectionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to connect: %v", err),
			}
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return ConnectionResult{
			Success: true,
			Message: fmt.Sprintf("Search engine is accessible (HTTP %d)", resp.StatusCode),
		}
	}

	return ConnectionResult{
		Success: false,
		Message: fmt.Sprintf("Search engine returned HTTP %d", resp.StatusCode),
	}
}

// TestSearchTools tests web_search and web_fetch tools with a sample query
// DEPRECATED: This function used chromedp-based tools which have been removed.
// Search functionality now uses Search API (search_api_tool.go)
// Web fetch functionality now uses HTTP client (web_fetch_tool.go)
func (a *App) TestSearchTools(engineURL string) ConnectionResult {
	// Get user's language preference
	cfg, _ := a.GetConfig()
	lang := cfg.Language
	isChinese := lang == "简体中文"

	msg := "Search tools test is deprecated. Please use Search API configuration instead."
	if isChinese {
		msg = "搜索工具测试已弃用。请改用搜索API配置。"
	}

	return ConnectionResult{
		Success: false,
		Message: msg,
	}
}

// TestProxy tests if a proxy server is accessible
func (a *App) TestProxy(proxyConfig config.ProxyConfig) ConnectionResult {
	if proxyConfig.Host == "" {
		return ConnectionResult{
			Success: false,
			Message: "Proxy host is required",
		}
	}

	if proxyConfig.Port <= 0 || proxyConfig.Port > 65535 {
		return ConnectionResult{
			Success: false,
			Message: "Invalid proxy port",
		}
	}

	// Determine protocol
	protocol := strings.ToLower(proxyConfig.Protocol)
	if protocol == "" {
		protocol = "http"
	}

	// Test proxy by making a request through it
	// Use a reliable test URL
	testURL := "https://www.google.com"

	// Build proxy URL for http.Transport
	var proxyUser *url.Userinfo
	if proxyConfig.Username != "" {
		if proxyConfig.Password != "" {
			proxyUser = url.UserPassword(proxyConfig.Username, proxyConfig.Password)
		} else {
			proxyUser = url.User(proxyConfig.Username)
		}
	}

	// Create HTTP client with proxy
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: protocol,
				Host:   fmt.Sprintf("%s:%d", proxyConfig.Host, proxyConfig.Port),
				User:   proxyUser,
			}),
		},
	}

	// Try to make a HEAD request
	resp, err := client.Head(testURL)
	if err != nil {
		return ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Proxy connection failed: %v", err),
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return ConnectionResult{
			Success: true,
			Message: fmt.Sprintf("Proxy is working (HTTP %d)", resp.StatusCode),
		}
	}

	return ConnectionResult{
		Success: true,
		Message: fmt.Sprintf("Proxy connected but returned HTTP %d", resp.StatusCode),
	}
}

// TestUAPIConnection tests the connection to UAPI service
func (a *App) TestUAPIConnection(apiToken, baseURL string) ConnectionResult {
	if apiToken == "" {
		return ConnectionResult{
			Success: false,
			Message: "API Token is required",
		}
	}

	a.logger.Log("[UAPI-TEST] Starting UAPI connection test...")

	// Create UAPI search tool
	uapiTool, err := agent.NewUAPISearchTool(a.logger.Log, apiToken)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create UAPI tool: %v", err)
		a.logger.Log(fmt.Sprintf("[UAPI-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	// Test with a simple search query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testInput := `{"query": "test", "max_results": 1, "source": "general"}`
	result, err := uapiTool.InvokableRun(ctx, testInput)
	if err != nil {
		errMsg := fmt.Sprintf("UAPI search test failed: %v", err)
		a.logger.Log(fmt.Sprintf("[UAPI-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	a.logger.Log(fmt.Sprintf("[UAPI-TEST] Test successful, result: %s", result))

	return ConnectionResult{
		Success: true,
		Message: "UAPI connection successful",
	}
}

// TestSearchAPI tests a search API configuration
func (a *App) TestSearchAPI(apiConfig config.SearchAPIConfig) ConnectionResult {
	a.logger.Log(fmt.Sprintf("[SEARCH-API-TEST] Testing %s...", apiConfig.Name))

	// Create search API tool
	searchTool, err := agent.NewSearchAPITool(a.logger.Log, &apiConfig)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create search tool: %v", err)
		a.logger.Log(fmt.Sprintf("[SEARCH-API-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	// Test with a simple search query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testInput := `{"query": "test search", "max_results": 3}`
	result, err := searchTool.InvokableRun(ctx, testInput)
	if err != nil {
		errMsg := fmt.Sprintf("%s test failed: %v", apiConfig.Name, err)
		a.logger.Log(fmt.Sprintf("[SEARCH-API-TEST] %s", errMsg))
		return ConnectionResult{
			Success: false,
			Message: errMsg,
		}
	}

	a.logger.Log(fmt.Sprintf("[SEARCH-API-TEST] %s test successful, result: %s", apiConfig.Name, result))

	return ConnectionResult{
		Success: true,
		Message: fmt.Sprintf("%s connection successful", apiConfig.Name),
	}
}

func (a *App) getDashboardTranslations(lang string) map[string]string {
	if lang == "简体中文" {
		return map[string]string{
			"Data Sources":  "数据源",
			"Total":         "总计",
			"Files":         "文件",
			"Local":         "本地",
			"Databases":     "数据库",
			"Connected":     "已连接",
			"Tables":        "数据表",
			"Analyzed":      "已分析",
			"ConnectPrompt": "连接数据源以开始使用。",
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

// GetDashboardData returns summary statistics and insights about data sources
func (a *App) GetDashboardData() DashboardData {
	if a.dataSourceService == nil {
		return DashboardData{}
	}

	cfg, _ := a.GetConfig()
	tr := a.getDashboardTranslations(cfg.Language)

	sources, _ := a.dataSourceService.LoadDataSources()

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

func (a *App) getLangPrompt(cfg config.Config) string {
	if cfg.Language == "简体中文" {
		return "Simplified Chinese"
	}
	return "English"
}

// GenerateIntentSuggestions generates possible interpretations of user's intent
func (a *App) GenerateIntentSuggestions(threadID, userMessage string) ([]IntentSuggestion, error) {
	return a.GenerateIntentSuggestionsWithExclusions(threadID, userMessage, nil)
}

// GenerateIntentSuggestionsWithExclusions generates possible interpretations of user's intent,
// excluding previously generated suggestions
// Validates: Requirements 5.1, 5.2, 5.3, 2.3, 6.5, 2.2, 7.1
func (a *App) GenerateIntentSuggestionsWithExclusions(threadID, userMessage string, excludedSuggestions []IntentSuggestion) ([]IntentSuggestion, error) {
	cfg, err := a.GetEffectiveConfig()
	if err != nil {
		return nil, err
	}

	// Get data source information for context
	var dataSourceID string
	var tableName string
	var columns []string

	if threadID != "" && a.chatService != nil {
		threads, _ := a.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				dataSourceID = t.DataSourceID
				break
			}
		}
	}

	if dataSourceID != "" && a.dataSourceService != nil {
		// Get data source
		dataSources, err := a.dataSourceService.LoadDataSources()
		if err == nil {
			for _, ds := range dataSources {
				if ds.ID == dataSourceID {
					if ds.Analysis != nil && len(ds.Analysis.Schema) > 0 {
						tableName = ds.Analysis.Schema[0].TableName
						columns = ds.Analysis.Schema[0].Columns
					}
					break
				}
			}
		}

		// If no analysis available, try to get table info directly
		if tableName == "" {
			tables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
			if err == nil && len(tables) > 0 {
				tableName = tables[0]
				cols, err := a.dataSourceService.GetDataSourceTableColumns(dataSourceID, tableName)
				if err == nil {
					columns = cols
				}
			}
		}
	}

	// Try to use the new IntentUnderstandingService if available and enabled
	// Validates: Requirements 7.1, 7.2 - Use new service with fallback to old implementation
	if a.intentUnderstandingService != nil && a.intentUnderstandingService.IsEnabled() {
		a.Log("[INTENT] Using new IntentUnderstandingService")

		// Convert language setting
		language := "en"
		if cfg.Language == "简体中文" {
			language = "zh"
		}

		// Convert excluded suggestions to agent.IntentSuggestion
		agentExclusions := convertToAgentSuggestions(excludedSuggestions)

		// Create LLM call function
		llmCall := func(ctx context.Context, prompt string) (string, error) {
			llm := agent.NewLLMService(cfg, a.Log)
			return llm.Chat(ctx, prompt)
		}

		// Generate suggestions using the new service
		agentSuggestions, err := a.intentUnderstandingService.GenerateSuggestions(
			a.ctx,
			threadID,
			userMessage,
			dataSourceID,
			language,
			agentExclusions,
			llmCall,
		)
		if err != nil {
			a.Log(fmt.Sprintf("[INTENT] IntentUnderstandingService failed: %v, falling back to old implementation", err))
			// Fall through to old implementation
		} else {
			// Convert agent.IntentSuggestion to IntentSuggestion
			suggestions := convertAgentSuggestions(agentSuggestions)
			a.Log(fmt.Sprintf("[INTENT] Generated %d suggestions using new service", len(suggestions)))
			return suggestions, nil
		}
	}

	// Fallback to old implementation using IntentEnhancementService
	// Validates: Requirements 7.2 - Fallback when new service is disabled or fails
	a.Log("[INTENT] Using legacy IntentEnhancementService")

	// Check cache for similar requests (Requirement 5.1, 5.2, 5.3)
	// IMPORTANT: Skip cache when there are exclusions - we need fresh suggestions from LLM
	// This ensures "Retry Understanding" actually generates new suggestions
	if a.intentEnhancementService != nil && dataSourceID != "" && len(excludedSuggestions) == 0 {
		cachedSuggestions, cacheHit := a.intentEnhancementService.GetCachedSuggestions(dataSourceID, userMessage)
		if cacheHit && len(cachedSuggestions) > 0 {
			a.Log("[INTENT] Cache hit for intent suggestions (no exclusions)")
			// Convert agent.IntentSuggestion to IntentSuggestion
			suggestions := make([]IntentSuggestion, len(cachedSuggestions))
			for i, s := range cachedSuggestions {
				suggestions[i] = IntentSuggestion{
					ID:          s.ID,
					Title:       s.Title,
					Description: s.Description,
					Icon:        s.Icon,
					Query:       s.Query,
				}
			}
			// Apply preference ranking even for cached results (Requirement 2.3)
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
			rankedSuggestions := a.intentEnhancementService.RankSuggestions(dataSourceID, agentSuggestions)
			for i, s := range rankedSuggestions {
				suggestions[i] = IntentSuggestion{
					ID:          s.ID,
					Title:       s.Title,
					Description: s.Description,
					Icon:        s.Icon,
					Query:       s.Query,
				}
			}
			return suggestions, nil
		}
	} else if len(excludedSuggestions) > 0 {
		a.Log(fmt.Sprintf("[INTENT] Skipping cache - retry with %d exclusions, will call LLM for fresh suggestions", len(excludedSuggestions)))
	}

	// Create ExclusionSummarizer and check if summarization is needed
	// Validates: Requirements 6.5, 2.2 - Use summary when exclusions exceed threshold
	summarizer := agent.NewExclusionSummarizer()
	var exclusionSummary string

	// Convert IntentSuggestion to ExclusionIntentSuggestion for the summarizer
	exclusionIntents := make([]agent.ExclusionIntentSuggestion, len(excludedSuggestions))
	for i, s := range excludedSuggestions {
		exclusionIntents[i] = agent.ExclusionIntentSuggestion{
			ID:          s.ID,
			Title:       s.Title,
			Description: s.Description,
			Icon:        s.Icon,
			Query:       s.Query,
		}
	}

	if summarizer.NeedsSummarization(exclusionIntents) {
		// Generate summary when exclusions exceed threshold (Requirement 6.5)
		exclusionSummary = summarizer.SummarizeExclusions(exclusionIntents)
		a.Log(fmt.Sprintf("[INTENT] Using exclusion summary for %d excluded suggestions (threshold: %d)",
			len(excludedSuggestions), summarizer.GetThreshold()))
	}

	// Build prompt for LLM - pass summary if available, otherwise pass full list
	// Validates: Requirement 2.2 - Pass exclusion list to LLM
	prompt := a.buildIntentUnderstandingPrompt(userMessage, tableName, columns, cfg.Language, excludedSuggestions, dataSourceID, exclusionSummary)

	// Call LLM
	llm := agent.NewLLMService(cfg, a.Log)
	response, err := llm.Chat(a.ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate intent suggestions: %v", err)
	}

	// Parse response
	suggestions, err := a.parseIntentSuggestions(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent suggestions: %v", err)
	}

	// Apply preference ranking (Requirement 2.3)
	if a.intentEnhancementService != nil && dataSourceID != "" && len(suggestions) > 0 {
		// Convert to agent.IntentSuggestion for ranking
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

		// Rank suggestions based on user preferences
		rankedSuggestions := a.intentEnhancementService.RankSuggestions(dataSourceID, agentSuggestions)

		// Convert back to IntentSuggestion
		for i, s := range rankedSuggestions {
			suggestions[i] = IntentSuggestion{
				ID:          s.ID,
				Title:       s.Title,
				Description: s.Description,
				Icon:        s.Icon,
				Query:       s.Query,
			}
		}

		// Cache the suggestions for future similar requests
		a.intentEnhancementService.CacheSuggestions(dataSourceID, userMessage, agentSuggestions)
	}

	return suggestions, nil
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
	outputLangInstruction := "Respond in English"
	langCode := "en"
	if language == "简体中文" {
		outputLangInstruction = "用简体中文回复"
		langCode = "zh"
	}

	columnsStr := strings.Join(columns, ", ")
	if columnsStr == "" {
		columnsStr = "No schema information available"
	}
	if tableName == "" {
		tableName = "Unknown"
	}

	// Build excluded suggestions section
	// Validates: Requirement 6.5 - Use summary when available to prevent context overload
	excludedSection := ""
	retryGuidance := ""
	if len(excludedSuggestions) > 0 {
		excludedSection = "\n\n## Previously Rejected Interpretations\n"
		excludedSection += "The user has indicated that the following interpretations DO NOT match their intent:\n\n"

		// Use summary if available (when exclusions exceed threshold), otherwise use full list
		// Validates: Requirement 6.5 - Use summarized exclusion description instead of full list
		if exclusionSummary != "" {
			// Use the compressed summary to prevent context overload
			excludedSection += exclusionSummary + "\n"
		} else {
			// Use full list when under threshold (Requirement 2.2)
			for i, suggestion := range excludedSuggestions {
				excludedSection += fmt.Sprintf("%d. **%s**: %s\n", i+1, suggestion.Title, suggestion.Description)
			}
		}
		retryGuidance = `

## Critical Instruction for Retry
The user rejected ALL previous suggestions. This means:
1. Your previous interpretations were off-target
2. You need to think from COMPLETELY DIFFERENT angles
3. Consider alternative meanings, contexts, or analysis approaches
4. Avoid similar patterns or themes from rejected suggestions
5. Be more creative and explore edge cases or unconventional interpretations`
	}

	// Build "stick to original" guidance based on language
	// Validates: Requirements 1.4, 9.1 - Bilingual support for prompt guidance
	stickToOriginalGuidance := ""
	if language == "简体中文" {
		stickToOriginalGuidance = `

# 关于"坚持我的请求"选项
用户可以选择"坚持我的请求"来直接使用他们的原始输入进行分析。因此：
1. 你的建议应该提供与原始请求不同的分析角度
2. 如果原始请求已经足够具体，你的建议应该探索相关但不同的分析方向
3. 不要简单地重复或轻微改写用户的原始请求
4. 每个建议都应该为用户提供独特的价值`
	} else {
		stickToOriginalGuidance = `

# About "Stick to My Request" Option
The user can choose "Stick to My Request" to use their original input directly for analysis. Therefore:
1. Your suggestions should offer different analytical angles from the original request
2. If the original request is already specific, your suggestions should explore related but different analysis directions
3. Do not simply repeat or slightly rephrase the user's original request
4. Each suggestion should provide unique value to the user`
	}

	basePrompt := fmt.Sprintf(`# Role
You are an expert data analysis intent interpreter. Your task is to understand ambiguous user requests and generate multiple plausible interpretations.

# User's Request
"%s"

# Available Data Context
- **Table**: %s
- **Columns**: %s%s%s%s

# Task
Generate 3-5 distinct interpretations of the user's intent. Each interpretation should:
1. Represent a different analytical perspective or approach
2. Be specific and actionable
3. Align with the available data structure
4. Be sorted by likelihood (most probable first)

# Interpretation Dimensions to Consider
- **Temporal Analysis**: Trends over time, period comparisons, seasonality
- **Segmentation**: By category, region, product, customer type, etc.
- **Aggregation Level**: Summary statistics, detailed breakdowns, rankings
- **Comparison**: Year-over-year, benchmarking, A/B testing
- **Correlation**: Relationships between variables, cause-effect analysis
- **Anomaly Detection**: Outliers, unusual patterns, exceptions
- **Forecasting**: Predictions, projections, what-if scenarios

# Output Format
Return a JSON array with 3-5 interpretations. Each object must include:

[
  {
    "title": "Short descriptive title (max 10 words)",
    "description": "Clear explanation of what this interpretation means (max 30 words)",
    "icon": "Relevant emoji (📊, 📈, 📉, 🔍, 💡, 📅, 🎯, etc.)",
    "query": "Specific, detailed analysis request that can be executed (be explicit about metrics, dimensions, and filters)"
  }
]

# Quality Requirements
- **Specificity**: Each query should be detailed enough to execute without ambiguity
- **Diversity**: Interpretations should cover different analytical angles
- **Feasibility**: Only suggest analyses that can be performed with the available columns
- **Clarity**: Descriptions should be clear and jargon-free
- **Language**: %s

# Output Rules
- Return ONLY the JSON array
- No markdown code blocks, no explanations, no additional text
- Ensure valid JSON syntax
- Start with [ and end with ]

Generate the interpretations now:`, userMessage, tableName, columnsStr, excludedSection, retryGuidance, stickToOriginalGuidance, outputLangInstruction)

	// Enhance prompt with context, dimensions, and examples using IntentEnhancementService
	// Validates: Requirements 6.1, 6.4 - backward compatibility maintained
	if a.intentEnhancementService != nil && dataSourceID != "" {
		// Convert columns to ColumnSchema for enhanced analysis
		columnSchemas := make([]agent.ColumnSchema, len(columns))
		for i, col := range columns {
			columnSchemas[i] = agent.ColumnSchema{Name: col}
		}

		enhancedPrompt, err := a.intentEnhancementService.EnhancePromptWithColumns(
			a.ctx,
			basePrompt,
			dataSourceID,
			userMessage,
			langCode,
			columnSchemas,
			tableName,
		)
		if err != nil {
			a.Log(fmt.Sprintf("[INTENT] Failed to enhance prompt: %v, using base prompt", err))
			return basePrompt
		}
		return enhancedPrompt
	}

	return basePrompt
}

func (a *App) parseIntentSuggestions(response string) ([]IntentSuggestion, error) {
	// Extract JSON from response
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")

	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no valid JSON array found in response")
	}

	jsonStr := response[start : end+1]

	// Parse JSON
	var rawSuggestions []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawSuggestions); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Convert to IntentSuggestion
	suggestions := make([]IntentSuggestion, 0, len(rawSuggestions))
	for i, raw := range rawSuggestions {
		suggestion := IntentSuggestion{
			ID:          fmt.Sprintf("intent_%d_%d", time.Now().Unix(), i),
			Title:       a.getString(raw, "title"),
			Description: a.getString(raw, "description"),
			Icon:        a.getString(raw, "icon"),
			Query:       a.getString(raw, "query"),
		}

		// Validate
		if suggestion.Title != "" && suggestion.Query != "" {
			suggestions = append(suggestions, suggestion)
		}
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no valid suggestions generated")
	}

	return suggestions, nil
}

func (a *App) getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SendMessage sends a message to the AI
// Task 3.1: Added requestId parameter for request tracking (Requirements 1.3, 4.3, 4.4)
func (a *App) SendMessage(threadID, message, userMessageID, requestID string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	cfg, err := a.GetEffectiveConfig()
	if err != nil {
		return "", err
	}

	startTotal := time.Now()

	// Log user message if threadID provided
	if threadID != "" && cfg.DetailedLog {
		a.logChatToFile(threadID, "USER REQUEST", message)
	}

	// Save user message to thread file BEFORE processing
	// This ensures the message is visible when frontend reloads the thread
	// Check if message already exists to prevent duplicates
	if threadID != "" && userMessageID != "" {
		// Load thread to check if message already exists
		threads, err := a.chatService.LoadThreads()
		if err == nil {
			messageExists := false
			for _, t := range threads {
				if t.ID == threadID {
					for _, m := range t.Messages {
						if m.ID == userMessageID {
							messageExists = true
							a.Log(fmt.Sprintf("[CHAT] User message already exists in thread: %s", userMessageID))
							break
						}
					}
					break
				}
			}

			// Only add message if it doesn't exist
			if !messageExists {
				userMsg := ChatMessage{
					ID:        userMessageID,
					Role:      "user",
					Content:   message,
					Timestamp: time.Now().Unix(),
				}
				if err := a.chatService.AddMessage(threadID, userMsg); err != nil {
					a.Log(fmt.Sprintf("[ERROR] Failed to save user message: %v", err))
					// Continue anyway - this is not a fatal error
				} else {
					a.Log(fmt.Sprintf("[CHAT] Saved user message to thread: %s", userMessageID))
				}
			}
		}
	}

	// Wait for concurrent analysis slot if needed
	// This implements queuing behavior instead of rejecting requests
	// Re-fetch config to get latest maxConcurrentAnalysis setting
	cfg, _ = a.GetConfig()
	maxConcurrent := cfg.MaxConcurrentAnalysis
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default to 5
	}
	if maxConcurrent > 10 {
		maxConcurrent = 10 // Cap at 10
	}

	// Check if we need to wait for a slot
	waitStartTime := time.Now()
	maxWaitTime := 5 * time.Minute // Maximum wait time before giving up
	checkInterval := 500 * time.Millisecond
	notifiedWaiting := false

	// First check if we need to wait
	a.activeThreadsMutex.RLock()
	activeCount := len(a.activeThreads)
	a.activeThreadsMutex.RUnlock()

	if activeCount >= maxConcurrent {
		// Immediately notify frontend that we're entering waiting state
		// This ensures the loading indicator shows up right away
		a.Log(fmt.Sprintf("[CONCURRENT] Need to wait for slot. Active: %d, Max: %d, Thread: %s", activeCount, maxConcurrent, threadID))

		// First, emit chat-loading to show the loading indicator
		if threadID != "" {
			a.Log(fmt.Sprintf("[LOADING-DEBUG] Backend emitting chat-loading true (waiting) for threadId: %s", threadID))
			runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
				"loading":  true,
				"threadId": threadID,
			})
		}

		// Then emit the queue status with waiting message
		var waitMessage string
		if cfg.Language == "简体中文" {
			waitMessage = fmt.Sprintf("等待分析队列中...（当前 %d/%d 个任务进行中）", activeCount, maxConcurrent)
		} else {
			waitMessage = fmt.Sprintf("Waiting in analysis queue... (%d/%d tasks in progress)", activeCount, maxConcurrent)
		}
		runtime.EventsEmit(a.ctx, "analysis-queue-status", map[string]interface{}{
			"threadId": threadID,
			"status":   "waiting",
			"message":  waitMessage,
			"position": activeCount - maxConcurrent + 1,
		})
		notifiedWaiting = true
	}

	for {
		a.activeThreadsMutex.RLock()
		activeCount = len(a.activeThreads)
		a.activeThreadsMutex.RUnlock()

		if activeCount < maxConcurrent {
			// Slot available, proceed
			if notifiedWaiting {
				a.Log(fmt.Sprintf("[CONCURRENT] Slot available after waiting, proceeding with analysis for thread: %s", threadID))
				// Notify frontend that waiting is over
				runtime.EventsEmit(a.ctx, "analysis-queue-status", map[string]interface{}{
					"threadId": threadID,
					"status":   "starting",
					"message":  "开始分析",
				})
			}
			break
		}

		// Check if we've waited too long
		if time.Since(waitStartTime) > maxWaitTime {
			a.Log(fmt.Sprintf("[CONCURRENT] Timeout waiting for analysis slot for thread: %s", threadID))
			// Clear loading state on timeout
			if threadID != "" {
				runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
					"loading":  false,
					"threadId": threadID,
				})
			}
			var errorMessage string
			if cfg.Language == "简体中文" {
				errorMessage = fmt.Sprintf("等待分析队列超时（已等待 %v）。当前有 %d 个分析任务进行中。请稍后重试。", time.Since(waitStartTime).Round(time.Second), activeCount)
			} else {
				errorMessage = fmt.Sprintf("Timeout waiting for analysis queue (waited %v). There are currently %d analysis tasks in progress. Please try again later.", time.Since(waitStartTime).Round(time.Second), activeCount)
			}
			return "", fmt.Errorf("%s", errorMessage)
		}

		// Check if cancellation was requested
		if a.IsCancelRequested() {
			a.Log(fmt.Sprintf("[CONCURRENT] Cancellation requested while waiting for slot, aborting for thread: %s", threadID))
			// Clear loading state on cancellation
			if threadID != "" {
				runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
					"loading":  false,
					"threadId": threadID,
				})
			}
			return "", fmt.Errorf("analysis cancelled while waiting in queue")
		}

		// Update waiting message periodically (every 5 seconds)
		if notifiedWaiting && int(time.Since(waitStartTime).Seconds())%5 == 0 {
			var waitMessage string
			waitedTime := time.Since(waitStartTime).Round(time.Second)
			if cfg.Language == "简体中文" {
				waitMessage = fmt.Sprintf("等待分析队列中...（已等待 %v，当前 %d/%d 个任务进行中）", waitedTime, activeCount, maxConcurrent)
			} else {
				waitMessage = fmt.Sprintf("Waiting in analysis queue... (waited %v, %d/%d tasks in progress)", waitedTime, activeCount, maxConcurrent)
			}
			runtime.EventsEmit(a.ctx, "analysis-queue-status", map[string]interface{}{
				"threadId": threadID,
				"status":   "waiting",
				"message":  waitMessage,
				"position": activeCount - maxConcurrent + 1,
			})
		}

		time.Sleep(checkInterval)
	}

	// Mark this thread as having active analysis
	a.activeThreadsMutex.Lock()
	a.activeThreads[threadID] = true
	a.activeThreadsMutex.Unlock()

	// Check license analysis limit before proceeding
	if a.licenseClient != nil && a.licenseClient.IsActivated() {
		canAnalyze, limitMsg := a.licenseClient.CanAnalyze()
		if !canAnalyze {
			// Remove from active threads since we're not proceeding
			a.activeThreadsMutex.Lock()
			delete(a.activeThreads, threadID)
			a.activeThreadsMutex.Unlock()
			
			// Clear loading state
			if threadID != "" {
				runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
					"loading":  false,
					"threadId": threadID,
				})
			}
			
			return "", fmt.Errorf(limitMsg)
		}
		// Increment analysis count
		a.licenseClient.IncrementAnalysis()
		a.Log("[LICENSE] Analysis count incremented")
	}

	// Notify frontend that loading has started
	if threadID != "" {
		a.Log(fmt.Sprintf("[LOADING-DEBUG] Backend emitting chat-loading true for threadId: %s", threadID))
		runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
			"loading":  true,
			"threadId": threadID,
		})
		
		// Notify frontend to clear current dashboard display for new analysis
		// This ensures the dashboard shows fresh results for the new request
		// Note: We emit analysis-result-loading instead of clearing all data,
		// so historical data is preserved and can be restored when user clicks old messages
		if a.eventAggregator != nil {
			a.Log(fmt.Sprintf("[DASHBOARD] Setting loading state for thread: %s, requestId: %s", threadID, requestID))
			a.eventAggregator.SetLoading(threadID, true, requestID)
		}
	}

	defer func() {
		a.activeThreadsMutex.Lock()
		delete(a.activeThreads, threadID)
		a.activeThreadsMutex.Unlock()

		// Notify frontend that loading is complete
		if threadID != "" {
			a.Log(fmt.Sprintf("[LOADING-DEBUG] Backend emitting chat-loading false for threadId: %s", threadID))
			runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
				"loading":  false,
				"threadId": threadID,
			})
		}
	}()

	// Set active thread and reset cancel flag
	a.cancelAnalysisMutex.Lock()
	a.activeThreadID = threadID
	a.cancelAnalysis = false
	a.cancelAnalysisMutex.Unlock()

	// Check if we should use Eino (if thread has DataSourceID)
	var useEino bool
	var dataSourceID string
	if threadID != "" && a.einoService != nil {
		startCheck := time.Now()
		threads, _ := a.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID && t.DataSourceID != "" {
				useEino = true
				dataSourceID = t.DataSourceID
				break
			}
		}
		a.Log(fmt.Sprintf("[TIMING] Checking Eino eligibility took: %v", time.Since(startCheck)))
	} else if threadID != "" && a.einoService == nil {
		a.Log("[ERROR] EinoService is nil - cannot use advanced analysis features")
		// Log current configuration for debugging
		if cfg, err := a.GetConfig(); err == nil {
			a.Log(fmt.Sprintf("[DEBUG] Current config - Provider: %s, Model: %s", cfg.LLMProvider, cfg.ModelName))
		}
	}

	if useEino {
		// Load history
		startHist := time.Now()
		var history []*schema.Message
		threads, _ := a.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				for _, m := range t.Messages {
					role := schema.User
					if m.Role == "assistant" {
						role = schema.Assistant
					}
					history = append(history, &schema.Message{
						Role:    role,
						Content: m.Content,
					})
				}
				break
			}
		}
		a.Log(fmt.Sprintf("[TIMING] Loading history took: %v", time.Since(startHist)))

		// Add current message (Eino expects the new user message in the input list for the chain we built)
		history = append(history, &schema.Message{Role: schema.User, Content: message})

		// Create progress callback to emit events to frontend with threadId
		progressCallback := func(update agent.ProgressUpdate) {
			// Include threadId in progress events for multi-session support
			progressWithThread := map[string]interface{}{
				"threadId":    threadID,
				"stage":       update.Stage,
				"progress":    update.Progress,
				"message":     update.Message,
				"step":        update.Step,
				"total":       update.Total,
				"tool_name":   update.ToolName,
				"tool_output": update.ToolOutput,
			}
			runtime.EventsEmit(a.ctx, "analysis-progress", progressWithThread)
		}

		// Get session directory for file storage
		sessionDir := a.chatService.GetSessionDirectory(threadID)

		// Capture existing session files before analysis (to identify new files later)
		existingFiles := make(map[string]bool)
		if preAnalysisFiles, err := a.chatService.GetSessionFiles(threadID); err == nil {
			for _, file := range preAnalysisFiles {
				existingFiles[file.Name] = true
			}
			a.Log(fmt.Sprintf("[CHART] Pre-analysis: %d existing files in session", len(existingFiles)))
		}

		// Create file saved callback to track generated files
		fileSavedCallback := func(fileName, fileType string, fileSize int64) {
			// Register the file in the chat thread
			file := SessionFile{
				Name:      fileName,
				Path:      fmt.Sprintf("files/%s", fileName),
				Type:      fileType,
				Size:      fileSize,
				CreatedAt: time.Now().Unix(),
			}
			if err := a.chatService.AddSessionFile(threadID, file); err != nil {
				a.Log(fmt.Sprintf("[ERROR] Failed to register session file: %v", err))
			} else {
				a.Log(fmt.Sprintf("[SESSION] Registered file: %s (%s, %d bytes)", fileName, fileType, fileSize))
			}
		}

		// Double-check EinoService is still available before using it (prevent race condition)
		if a.einoService == nil {
			a.Log("[WARNING] EinoService became nil during request processing, falling back to standard LLM")
			// Fall through to standard LLM processing
		} else {
			a.Log(fmt.Sprintf("[EINO-CHECK] EinoService is available, proceeding with analysis for thread: %s, dataSource: %s", threadID, dataSourceID))

			// Start timing for analysis
			analysisStartTime := time.Now()

			respMsg, err := a.einoService.RunAnalysisWithProgress(a.ctx, history, dataSourceID, threadID, sessionDir, userMessageID, progressCallback, fileSavedCallback, a.IsCancelRequested)

			// Calculate analysis duration
			analysisDuration := time.Since(analysisStartTime)
			minutes := int(analysisDuration.Minutes())
			seconds := int(analysisDuration.Seconds()) % 60

			var resp string
			if err != nil {
				resp = fmt.Sprintf("Error: %v", err)
				if cfg.DetailedLog {
					a.logChatToFile(threadID, "SYSTEM ERROR", resp)
				}

				// Determine error type and emit appropriate event
				errStr := err.Error()
				
				// Check if this was a cancellation
				if strings.Contains(errStr, "cancelled by user") || strings.Contains(errStr, "cancelled while waiting") {
					a.Log(fmt.Sprintf("[CANCEL] Analysis cancelled for thread: %s", threadID))
					// Use event aggregator for consistent event emission
					if a.eventAggregator != nil {
						a.eventAggregator.EmitCancelled(threadID, requestID)
						a.eventAggregator.SetLoading(threadID, false, requestID)
					} else {
						runtime.EventsEmit(a.ctx, "analysis-cancelled", map[string]interface{}{
							"threadId":  threadID,
							"requestId": requestID,
							"message":   "分析已取消",
							"timestamp": time.Now().UnixMilli(),
						})
						runtime.EventsEmit(a.ctx, "analysis-result-loading", map[string]interface{}{
							"sessionId": threadID,
							"loading":   false,
							"requestId": requestID,
						})
					}
					// Emit chat-loading false to update App.tsx loading state
					runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
						"loading":  false,
						"threadId": threadID,
					})
				} else {
					// Determine error code based on error message
					var errorCode string
					var userFriendlyMessage string
					
					switch {
					case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "Timeout"):
						errorCode = "ANALYSIS_TIMEOUT"
						userFriendlyMessage = fmt.Sprintf("分析超时（已运行 %d分%d秒）。请尝试简化查询或稍后重试。", minutes, seconds)
					case strings.Contains(errStr, "context canceled") || strings.Contains(errStr, "context deadline exceeded"):
						errorCode = "ANALYSIS_TIMEOUT"
						userFriendlyMessage = "分析请求超时。请尝试简化查询或稍后重试。"
					case strings.Contains(errStr, "connection") || strings.Contains(errStr, "network"):
						errorCode = "NETWORK_ERROR"
						userFriendlyMessage = "网络连接错误。请检查网络连接后重试。"
					case strings.Contains(errStr, "database") || strings.Contains(errStr, "sqlite") || strings.Contains(errStr, "SQL"):
						errorCode = "DATABASE_ERROR"
						userFriendlyMessage = "数据库查询错误。请检查数据源配置或查询条件。"
					case strings.Contains(errStr, "Python") || strings.Contains(errStr, "python"):
						errorCode = "PYTHON_ERROR"
						userFriendlyMessage = "Python 执行错误。请检查分析代码或数据格式。"
					case strings.Contains(errStr, "LLM") || strings.Contains(errStr, "API") || strings.Contains(errStr, "model"):
						errorCode = "LLM_ERROR"
						userFriendlyMessage = "AI 模型调用错误。请检查 API 配置或稍后重试。"
					default:
						errorCode = "ANALYSIS_ERROR"
						userFriendlyMessage = fmt.Sprintf("分析过程中发生错误: %s", errStr)
					}
					
					a.Log(fmt.Sprintf("[ERROR] Analysis error for thread %s: code=%s, message=%s", threadID, errorCode, errStr))
					
					// Emit error event to frontend with detailed information
					if a.eventAggregator != nil {
						a.eventAggregator.EmitErrorWithCode(threadID, requestID, errorCode, userFriendlyMessage)
						a.eventAggregator.SetLoading(threadID, false, requestID)
					} else {
						runtime.EventsEmit(a.ctx, "analysis-error", map[string]interface{}{
							"threadId":  threadID,
							"sessionId": threadID,
							"requestId": requestID,
							"code":      errorCode,
							"error":     userFriendlyMessage,
							"message":   userFriendlyMessage,
							"timestamp": time.Now().UnixMilli(),
						})
						runtime.EventsEmit(a.ctx, "analysis-result-loading", map[string]interface{}{
							"sessionId": threadID,
							"loading":   false,
							"requestId": requestID,
						})
					}
					// Emit chat-loading false to update App.tsx loading state
					runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
						"loading":  false,
						"threadId": threadID,
					})
				}

				return "", err
			}
			resp = respMsg.Content

			// Check if timing information is already present before adding
			if !strings.Contains(resp, "⏱️ 分析耗时:") {
				// Append timing information to response
				timingInfo := fmt.Sprintf("\n\n---\n⏱️ 分析耗时: %d分%d秒", minutes, seconds)
				resp = resp + timingInfo
				a.Log(fmt.Sprintf("[TIMING] Analysis completed in: %d分%d秒 (%v)", minutes, seconds, analysisDuration))
			} else {
				a.Log(fmt.Sprintf("[TIMING] Timing info already present in response, skipping addition. Duration: %d分%d秒 (%v)", minutes, seconds, analysisDuration))
			}

			if cfg.DetailedLog {
				a.logChatToFile(threadID, "LLM RESPONSE", resp)
			}

			// Detect and emit images from the response
			a.detectAndEmitImages(resp, threadID, userMessageID, requestID)

			// Filter out false file generation claims when ECharts is used
			// LLM sometimes hallucinates file generation when using ECharts
			resp = a.filterFalseFileClaimsIfECharts(resp)

			startPost := time.Now()
			// Detect and store chart data
			var chartData *ChartData
			var chartItems []ChartItem // Collect all chart types

			// Collect all chart types (ECharts, Images, Tables, CSV)
			// Changed from priority-based to collection-based approach

			// 1. ECharts JSON
			// Match until closing ``` to handle deeply nested objects
			// Allow optional newline after json:echarts
			reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
			matchECharts := reECharts.FindStringSubmatch(resp)
			if len(matchECharts) > 1 {
				jsonStr := strings.TrimSpace(matchECharts[1])
				// Validate it's valid JSON before using
				var testJSON map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &testJSON); err == nil {
					// Check if data is large and should be saved to file
					fileRef, saveErr := a.saveChartDataToFile(threadID, "echarts", jsonStr)

					var chartDataStr string
					if saveErr != nil {
						// Log error but continue with inline storage as fallback
						a.Log(fmt.Sprintf("[CHART-FILE] Failed to save to file, using inline storage: %v", saveErr))
						chartDataStr = jsonStr
					} else if fileRef != "" {
						// Use file reference (large data saved to file)
						chartDataStr = fileRef
						a.Log(fmt.Sprintf("[CHART-FILE] Using file reference: %s", fileRef))
					} else {
						// Small data, use inline storage
						chartDataStr = jsonStr
					}

					chartItems = append(chartItems, ChartItem{Type: "echarts", Data: chartDataStr})
					a.Log("[CHART] Detected ECharts JSON")
					// Use event aggregator for new unified event system
					if a.eventAggregator != nil {
						a.eventAggregator.AddECharts(threadID, userMessageID, requestID, jsonStr)
					}
				} else {
					maxLen := 500
					if len(jsonStr) < maxLen {
						maxLen = len(jsonStr)
					}
					a.Log(fmt.Sprintf("[CHART] Failed to parse echarts JSON: %v\nJSON string (first 500 chars): %s", err, jsonStr[:maxLen]))
				}
			}

			// 2. Markdown Image (Base64) - always check, don't skip if ECharts exists
			reImage := regexp.MustCompile(`!\[.*?\]\((data:image\/.*?;base64,.*?)\)`)
			matchImage := reImage.FindStringSubmatch(resp)
			if len(matchImage) > 1 {
				chartItems = append(chartItems, ChartItem{Type: "image", Data: matchImage[1]})
				a.Log("[CHART] Detected inline base64 image")
				// Use event aggregator for new unified event system
				if a.eventAggregator != nil {
					a.eventAggregator.AddImage(threadID, userMessageID, requestID, matchImage[1], "")
				}
			}

			// 3. Check for saved chart files (e.g., chart_timestamp.png from Python tool)
			// Always check, don't skip if ECharts exists
			if threadID != "" {
				// Get session files to see if chart images were saved
				sessionFiles, err := a.chatService.GetSessionFiles(threadID)
				if err == nil {
					// Collect ONLY NEWLY CREATED chart image files (not pre-existing ones)
					newFileCount := 0
					for _, file := range sessionFiles {
						// Skip files that existed before the analysis started
						if existingFiles[file.Name] {
							continue
						}

						if file.Type == "image" && (file.Name == "chart.png" || strings.HasPrefix(file.Name, "chart")) {
							// Read the image file and encode as base64
							filePath := filepath.Join(sessionDir, "files", file.Name)
							if imageData, err := os.ReadFile(filePath); err == nil {
								base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
								chartItems = append(chartItems, ChartItem{Type: "image", Data: base64Data})
								newFileCount++
								a.Log(fmt.Sprintf("[CHART] Detected NEW chart file from this analysis: %s", file.Name))
							}
						}
					}

					if newFileCount > 0 {
						a.Log(fmt.Sprintf("[CHART] Added %d new chart file(s) to chart items", newFileCount))
					} else {
						a.Log("[CHART] No new chart files generated in this analysis")
					}
				}
			}

			// NOTE: Don't create chartData here yet - wait until all chart types are collected
			// Table and CSV data are processed below and need to be included
			a.Log(fmt.Sprintf("[CHART] Charts collected so far (ECharts + Images): %d", len(chartItems)))

			// 4. Dashboard Data Update (Metrics & Insights)
			// Match until closing ``` to handle nested objects (same fix as echarts/table)
			reDashboard := regexp.MustCompile("(?s)```\\s*json:dashboard\\s*\\n([\\s\\S]+?)\\n\\s*```")
			matchDashboard := reDashboard.FindStringSubmatch(resp)
			if len(matchDashboard) > 1 {
				jsonStr := strings.TrimSpace(matchDashboard[1])
				var data DashboardData
				if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
					// Use event aggregator for new unified event system
					if a.eventAggregator != nil {
						for _, metric := range data.Metrics {
							a.eventAggregator.AddMetric(threadID, userMessageID, requestID, metric)
						}
						for _, insight := range data.Insights {
							a.eventAggregator.AddInsight(threadID, userMessageID, requestID, insight)
						}
					}
				} else {
					a.Log(fmt.Sprintf("[DASHBOARD] Failed to unmarshal dashboard data: %v\nJSON (first 500 chars): %s", err, jsonStr[:min(500, len(jsonStr))]))
				}
			}

			// 5. Table Data (JSON array from SQL results or analysis) - always check
			// Use [\s\S] instead of . to match newlines, match until closing ``` not first ]
			// Allow optional newline after json:table
			reTable := regexp.MustCompile("(?s)```\\s*json:table\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
			matchTable := reTable.FindStringSubmatch(resp)
			if len(matchTable) > 1 {
				jsonStr := strings.TrimSpace(matchTable[1])
				
				// Try to parse as object array first (standard format)
				var tableData []map[string]interface{}
				var parseErr error
				if parseErr = json.Unmarshal([]byte(jsonStr), &tableData); parseErr != nil {
					// Try to parse as 2D array (first row is headers)
					var arrayData [][]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &arrayData); err == nil && len(arrayData) > 1 {
						// Convert 2D array to object array
						// First row is headers
						headers := make([]string, len(arrayData[0]))
						for i, h := range arrayData[0] {
							headers[i] = fmt.Sprintf("%v", h)
						}
						
						// Remaining rows are data
						tableData = make([]map[string]interface{}, 0, len(arrayData)-1)
						for _, row := range arrayData[1:] {
							rowMap := make(map[string]interface{})
							for i, val := range row {
								if i < len(headers) {
									rowMap[headers[i]] = val
								}
							}
							tableData = append(tableData, rowMap)
						}
						parseErr = nil
						a.Log(fmt.Sprintf("[CHART] Converted 2D array table: %d columns, %d rows", len(headers), len(tableData)))
					}
				}
				
				if parseErr == nil && len(tableData) > 0 {
					tableDataJSON, _ := json.Marshal(tableData)
					tableDataStr := string(tableDataJSON)

					// Check if table data is large and should be saved to file
					fileRef, saveErr := a.saveChartDataToFile(threadID, "table", tableDataStr)

					var finalTableData string
					if saveErr != nil {
						// Log error but continue with inline storage as fallback
						a.Log(fmt.Sprintf("[CHART-FILE] Failed to save table data to file, using inline storage: %v", saveErr))
						finalTableData = tableDataStr
					} else if fileRef != "" {
						// Use file reference (large data saved to file)
						finalTableData = fileRef
						a.Log(fmt.Sprintf("[CHART-FILE] Using file reference for table data: %s", fileRef))
					} else {
						// Small data, use inline storage
						finalTableData = tableDataStr
					}

					chartItems = append(chartItems, ChartItem{Type: "table", Data: finalTableData})
					a.Log("[CHART] Detected table data")

					// Use event aggregator for new unified event system
					if a.eventAggregator != nil {
						a.eventAggregator.AddTable(threadID, userMessageID, requestID, tableData)
					}
				} else {
					maxLen := 500
					if len(jsonStr) < maxLen {
						maxLen = len(jsonStr)
					}
					a.Log(fmt.Sprintf("[CHART] Failed to parse table JSON: %v\nJSON string (first 500 chars): %s", parseErr, jsonStr[:maxLen]))
				}
			}

			// 6. CSV Download Link (data URL) - always check
			reCSV := regexp.MustCompile(`\[.*?\]\((data:text/csv;base64,[A-Za-z0-9+/=]+)\)`)
			matchCSV := reCSV.FindStringSubmatch(resp)
			if len(matchCSV) > 1 {
				chartItems = append(chartItems, ChartItem{Type: "csv", Data: matchCSV[1]})
				a.Log("[CHART] Detected CSV data")
				// Use event aggregator for new unified event system
				if a.eventAggregator != nil {
					a.eventAggregator.AddCSV(threadID, userMessageID, requestID, matchCSV[1], "")
				}
			}

			// Create chartData with ALL collected items (ECharts, Images, Tables, CSV)
			// This is done AFTER all chart types are processed to ensure nothing is missed
			if len(chartItems) > 0 {
				chartData = &ChartData{Charts: chartItems}
				a.Log(fmt.Sprintf("[CHART] Final total charts: %d (ECharts + Images + Tables + CSV)", len(chartItems)))
			}

			// Attach chart data to the user's request message (specific user message ID)
			if chartData != nil && threadID != "" {
				if userMessageID != "" {
					a.attachChartToUserMessage(threadID, userMessageID, chartData)
				} else {
					// Fallback to old behavior (last user message) only if ID is missing (backward compatibility)
					a.Log("[WARNING] SendMessage called without userMessageID, falling back to last user message")
					a.attachChartToUserMessage(threadID, "", chartData)
				}
			}

			// Create and save assistant message with chart data BEFORE returning
			// This ensures chart_data is available when frontend reloads the thread
			if threadID != "" {
				// Prepare timing data with stage breakdown
				// Use the same analysisDuration that was used for the response timing info
				totalSecs := analysisDuration.Seconds()

				// Estimate stage durations (rough approximation)
				// Typical breakdown: AI ~60%, SQL ~20%, Python ~15%, Other ~5%
				aiTime := totalSecs * 0.60
				sqlTime := totalSecs * 0.20
				pythonTime := totalSecs * 0.15
				otherTime := totalSecs * 0.05

				timingData := map[string]interface{}{
					"total_seconds":           totalSecs,
					"total_minutes":           minutes,
					"total_seconds_remainder": seconds,
					"analysis_type":           "eino_service",
					"timestamp":               analysisStartTime.Add(analysisDuration).Unix(), // Use analysis end time, not current time
					"stages": []map[string]interface{}{
						{
							"name":        "AI 分析",
							"duration":    aiTime,
							"percentage":  60.0,
							"description": "LLM 理解需求、生成代码和分析结果",
						},
						{
							"name":        "SQL 查询",
							"duration":    sqlTime,
							"percentage":  20.0,
							"description": "数据库查询和数据提取",
						},
						{
							"name":        "Python 处理",
							"duration":    pythonTime,
							"percentage":  15.0,
							"description": "数据处理和图表生成",
						},
						{
							"name":        "其他",
							"duration":    otherTime,
							"percentage":  5.0,
							"description": "初始化和后处理",
						},
					},
				}

				assistantMsg := ChatMessage{
					ID:         strconv.FormatInt(time.Now().UnixNano(), 10),
					Role:       "assistant",
					Content:    resp,
					Timestamp:  time.Now().Unix(),
					ChartData:  chartData,  // Attach chart data to assistant message
					TimingData: timingData, // Attach timing data
				}

				if err := a.chatService.AddMessage(threadID, assistantMsg); err != nil {
					a.Log(fmt.Sprintf("[CHART] Failed to save assistant message: %v", err))
				} else {
					a.Log(fmt.Sprintf("[CHART] Saved assistant message with chart_data: %v, timing_data: %v", chartData != nil, timingData != nil))

					// Associate newly created files with the USER message (not assistant message)
					// This makes more sense as files are generated in response to the user's analysis request
					if userMessageID != "" {
						if err := a.associateNewFilesWithMessage(threadID, userMessageID, existingFiles); err != nil {
							a.Log(fmt.Sprintf("[SESSION] Failed to associate files with user message: %v", err))
						} else {
							a.Log(fmt.Sprintf("[SESSION] Associated new files with user message: %s", userMessageID))
						}
					} else {
						a.Log("[WARNING] No userMessageID available, cannot associate files")
					}

					// Emit analysis-completed event to trigger automatic dashboard update
					// Task 3.1: Added requestId to event payload (Requirements 1.3, 4.3, 4.4)

					// Flush all pending analysis results before emitting completion
					var flushedItems []AnalysisResultItem
					if a.eventAggregator != nil {
						flushedItems = a.eventAggregator.FlushNow(threadID, true)
					}

					// Save analysis results to the user message for persistence
					if len(flushedItems) > 0 && userMessageID != "" {
						if err := a.chatService.SaveAnalysisResults(threadID, userMessageID, flushedItems); err != nil {
							a.Log(fmt.Sprintf("[PERSISTENCE] Failed to save analysis results: %v", err))
						} else {
							a.Log(fmt.Sprintf("[PERSISTENCE] Saved %d analysis results to message %s", len(flushedItems), userMessageID))
						}
					}

					runtime.EventsEmit(a.ctx, "analysis-completed", map[string]interface{}{
						"threadId":       threadID,
						"userMessageId":  userMessageID,
						"assistantMsgId": assistantMsg.ID,
						"hasChartData":   chartData != nil,
						"requestId":      requestID, // Task 3.1: Include requestId for frontend validation
					})
					a.Log(fmt.Sprintf("[DASHBOARD] Emitted analysis-completed event for message %s with requestId %s", userMessageID, requestID))

					// Record analysis completion for intent enhancement (Requirement 1.1)
					if a.intentEnhancementService != nil && dataSourceID != "" {
						go func(dsID string, respContent string) {
							// Get available columns from the data source
							var availableColumns []string
							if a.dataSourceService != nil {
								// Get all tables for the data source
								if tables, err := a.dataSourceService.GetDataSourceTables(dsID); err == nil {
									// Get columns from all tables
									for _, tableName := range tables {
										if cols, err := a.dataSourceService.GetDataSourceTableColumns(dsID, tableName); err == nil {
											availableColumns = append(availableColumns, cols...)
										}
									}
								}
							}

							// Extract analysis type and key findings from the response
							analysisType := a.detectAnalysisType(respContent)
							keyFindings := a.extractKeyFindings(respContent)
							targetColumns := a.extractTargetColumns(respContent, availableColumns)

							record := agent.AnalysisRecord{
								DataSourceID:  dsID,
								AnalysisType:  analysisType,
								TargetColumns: targetColumns,
								KeyFindings:   keyFindings,
							}

							// Record the analysis history
							a.recordAnalysisHistory(dsID, record)
						}(dataSourceID, resp)
					}
				}
			}

			a.Log(fmt.Sprintf("[TIMING] Post-processing response took: %v", time.Since(startPost)))
			a.Log(fmt.Sprintf("[TIMING] Total SendMessage (Eino) took: %v", time.Since(startTotal)))

			// Auto-extract metrics from analysis response
			if resp != "" && userMessageID != "" {
				go func() {
					// Small delay to ensure frontend has processed the response
					time.Sleep(1 * time.Second)

					// Notify frontend that metrics extraction is starting
					runtime.EventsEmit(a.ctx, "metrics-extracting", userMessageID)

					if err := a.ExtractMetricsFromAnalysis(threadID, userMessageID, resp); err != nil {
						a.Log(fmt.Sprintf("Failed to extract metrics for message %s: %v", userMessageID, err))
					}
					// Extract and emit suggestions to dashboard
					if err := a.ExtractSuggestionsFromAnalysis(threadID, userMessageID, resp); err != nil {
						a.Log(fmt.Sprintf("Failed to extract suggestions for message %s: %v", userMessageID, err))
					}
				}()
			}

			a.Log(fmt.Sprintf("[DEBUG-TRUNCATION] Returning response length: %d characters", len(resp)))
			if len(resp) > 500 {
				a.Log(fmt.Sprintf("[DEBUG-TRUNCATION] Response preview (first 200 chars): %s", resp[:200]))
				a.Log(fmt.Sprintf("[DEBUG-TRUNCATION] Response preview (last 200 chars): %s", resp[len(resp)-200:]))
			}
			return resp, nil
		}
	}

	langPrompt := a.getLangPrompt(cfg)
	fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", message, langPrompt)

	llm := agent.NewLLMService(cfg, a.Log)

	// Start timing for standard LLM chat
	chatStartTime := time.Now()
	startChat := time.Now()
	resp, err := llm.Chat(a.ctx, fullMessage)

	// Calculate chat duration
	chatDuration := time.Since(chatStartTime)
	minutes := int(chatDuration.Minutes())
	seconds := int(chatDuration.Seconds()) % 60

	a.Log(fmt.Sprintf("[TIMING] LLM Chat (Standard) took: %v", time.Since(startChat)))

	// Append timing information to response if successful
	if err == nil && resp != "" {
		// Check if timing information is already present (from EinoService fallback)
		if !strings.Contains(resp, "⏱️ 分析耗时:") {
			timingInfo := fmt.Sprintf("\n\n---\n⏱️ 分析耗时: %d分%d秒", minutes, seconds)
			resp = resp + timingInfo
		}
		a.Log(fmt.Sprintf("[TIMING] Chat completed in: %d分%d秒 (%v)", minutes, seconds, chatDuration))
	}

	// Log LLM response if threadID provided
	if threadID != "" && cfg.DetailedLog {
		if err != nil {
			a.logChatToFile(threadID, "SYSTEM ERROR", fmt.Sprintf("Error: %v", err))
		} else {
			a.logChatToFile(threadID, "LLM RESPONSE", resp)
		}
	}

	// Auto-extract metrics from analysis response (for standard LLM path)
	if resp != "" && err == nil && threadID != "" {
		// For standard path, we don't have userMessageID, so we'll use a generated one
		// This is less ideal but provides fallback functionality
		go func() {
			// Small delay to ensure frontend has processed the response
			time.Sleep(1 * time.Second)

			// Generate a message ID based on timestamp and thread
			messageID := fmt.Sprintf("%s_%d", threadID, time.Now().UnixNano())

			// Notify frontend that metrics extraction is starting
			runtime.EventsEmit(a.ctx, "metrics-extracting", messageID)

			if err := a.ExtractMetricsFromAnalysis(threadID, messageID, resp); err != nil {
				a.Log(fmt.Sprintf("Failed to extract metrics for standard LLM response: %v", err))
			}
		}()
	}

	a.Log(fmt.Sprintf("[TIMING] Total SendMessage (Standard) took: %v", time.Since(startTotal)))
	a.Log(fmt.Sprintf("[DEBUG-TRUNCATION] Returning response length: %d characters", len(resp)))
	if len(resp) > 500 {
		a.Log(fmt.Sprintf("[DEBUG-TRUNCATION] Response preview (first 200 chars): %s", resp[:200]))
		a.Log(fmt.Sprintf("[DEBUG-TRUNCATION] Response preview (last 200 chars): %s", resp[len(resp)-200:]))
	}
	return resp, err
}

// SendFreeChatMessage sends a message to the LLM without data source context (free chat mode)
// This allows users to have a direct conversation with the LLM like web ChatGPT
// Uses streaming for better user experience
// Supports web search and fetch tools for information retrieval (e.g., weather queries)
func (a *App) SendFreeChatMessage(threadID, message, userMessageID string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	cfg, err := a.GetEffectiveConfig()
	if err != nil {
		return "", err
	}

	startTotal := time.Now()

	// Log user message if threadID provided
	if threadID != "" && cfg.DetailedLog {
		a.logChatToFile(threadID, "FREE CHAT USER", message)
	}

	// Save user message to thread file BEFORE processing
	if threadID != "" && userMessageID != "" {
		threads, err := a.chatService.LoadThreads()
		if err == nil {
			messageExists := false
			for _, t := range threads {
				if t.ID == threadID {
					for _, m := range t.Messages {
						if m.ID == userMessageID {
							messageExists = true
							break
						}
					}
					break
				}
			}

			if !messageExists {
				userMsg := ChatMessage{
					ID:        userMessageID,
					Role:      "user",
					Content:   message,
					Timestamp: time.Now().Unix(),
				}
				if err := a.chatService.AddMessage(threadID, userMsg); err != nil {
					a.Log(fmt.Sprintf("[ERROR] Failed to save free chat user message: %v", err))
				}
			}
		}
	}

	// NOTE: For free chat with streaming, we don't emit chat-loading events
	// because the streaming output itself serves as progress feedback

	// Build conversation history for context
	var historyContext strings.Builder
	if threadID != "" {
		threads, _ := a.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				// Include last 10 messages for context
				startIdx := 0
				if len(t.Messages) > 10 {
					startIdx = len(t.Messages) - 10
				}
				for _, m := range t.Messages[startIdx:] {
					if m.Role == "user" {
						historyContext.WriteString(fmt.Sprintf("User: %s\n", m.Content))
					} else if m.Role == "assistant" {
						// Truncate long assistant responses
						content := m.Content
						if len(content) > 500 {
							content = content[:500] + "..."
						}
						historyContext.WriteString(fmt.Sprintf("Assistant: %s\n", content))
					}
				}
				break
			}
		}
	}

	// Use smart tool router to determine if tools are needed
	toolRouter := agent.NewToolRouter(a.Log)
	routerResult := toolRouter.Route(message)
	
	// Also check legacy keyword detection as fallback
	legacyNeedsSearch := a.detectWebSearchNeed(message)
	
	// Combine both methods: use tools if either method suggests it
	needsTools := routerResult.NeedsTools || legacyNeedsSearch
	
	a.Log(fmt.Sprintf("[FREE-CHAT] Tool routing: router=%v (confidence=%.2f, reason=%s), legacy=%v, final=%v",
		routerResult.NeedsTools, routerResult.Confidence, routerResult.Reason, legacyNeedsSearch, needsTools))

	// Build the prompt with conversation history
	langPrompt := a.getLangPrompt(cfg)
	var fullMessage string
	if historyContext.Len() > 0 {
		fullMessage = fmt.Sprintf("Previous conversation:\n%s\nUser: %s\n\n(Please answer in %s)", historyContext.String(), message, langPrompt)
	} else {
		fullMessage = fmt.Sprintf("%s\n\n(Please answer in %s)", message, langPrompt)
	}

	// Create assistant message ID for streaming updates
	assistantMsgID := fmt.Sprintf("assistant_%d", time.Now().UnixNano())

	// Emit initial empty message for streaming
	if threadID != "" {
		runtime.EventsEmit(a.ctx, "free-chat-stream-start", map[string]interface{}{
			"threadId":  threadID,
			"messageId": assistantMsgID,
		})
	}

	chatStartTime := time.Now()

	// Stream callback to emit chunks to frontend
	onChunk := func(content string) {
		if threadID != "" {
			runtime.EventsEmit(a.ctx, "free-chat-stream-chunk", map[string]interface{}{
				"threadId":  threadID,
				"messageId": assistantMsgID,
				"content":   content,
			})
		}
	}

	var resp string

	// Two-path approach for optimal user experience:
	// 1. For queries needing tools: Use agent mode with tools (slower, no real streaming, but can use tools)
	// 2. For general conversation: Use streaming LLM chat (fast, real streaming)
	if needsTools && a.einoService != nil {
		// Path 1: Agent mode with tools for queries that need external information
		a.Log("[FREE-CHAT] Tool router detected tool need, using agent with tools (non-streaming)")
		
		// Emit chat-loading event to show loading indicator in chat area
		if threadID != "" {
			a.Log(fmt.Sprintf("[LOADING-DEBUG] Free chat emitting chat-loading true for threadId: %s", threadID))
			runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
				"loading":  true,
				"threadId": threadID,
			})
		}
		
		// Emit search status event to frontend (will show spinner)
		if threadID != "" {
			runtime.EventsEmit(a.ctx, "free-chat-search-status", map[string]interface{}{
				"threadId":  threadID,
				"messageId": assistantMsgID,
				"searching": true,
			})
			// Also emit progress update for the loading indicator
			runtime.EventsEmit(a.ctx, "analysis-progress", map[string]interface{}{
				"threadId": threadID,
				"stage":    "analyzing",
				"progress": 0,
				"message":  "正在搜索网络信息...",
				"step":     1,
				"total":    1,
			})
		}
		resp, err = a.runFreeChatWithTools(a.ctx, message, historyContext.String(), langPrompt, onChunk)
		
		// If tool-based chat failed, try falling back to simple streaming chat
		if err != nil {
			a.Log(fmt.Sprintf("[FREE-CHAT] Tool-based chat failed: %v, falling back to streaming chat", err))
			llm := agent.NewLLMService(cfg, a.Log)
			resp, err = llm.ChatStream(a.ctx, fullMessage, onChunk)
		}
		
		// Emit search complete event
		if threadID != "" {
			runtime.EventsEmit(a.ctx, "free-chat-search-status", map[string]interface{}{
				"threadId":  threadID,
				"messageId": assistantMsgID,
				"searching": false,
			})
			// Emit chat-loading false to hide loading indicator
			a.Log(fmt.Sprintf("[LOADING-DEBUG] Free chat emitting chat-loading false for threadId: %s", threadID))
			runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
				"loading":  false,
				"threadId": threadID,
			})
		}
	} else {
		// Path 2: Streaming LLM chat for general conversation (fast, real streaming)
		a.Log("[FREE-CHAT] No search keyword detected, using streaming LLM chat for better UX")
		llm := agent.NewLLMService(cfg, a.Log)
		resp, err = llm.ChatStream(a.ctx, fullMessage, onChunk)
	}

	chatDuration := time.Since(chatStartTime)

	// Emit stream end
	if threadID != "" {
		runtime.EventsEmit(a.ctx, "free-chat-stream-end", map[string]interface{}{
			"threadId":  threadID,
			"messageId": assistantMsgID,
		})
	}

	if err != nil {
		if threadID != "" && cfg.DetailedLog {
			a.logChatToFile(threadID, "FREE CHAT ERROR", fmt.Sprintf("Error: %v", err))
		}
		return "", err
	}

	// Save assistant response to thread
	if threadID != "" && resp != "" {
		assistantMsg := ChatMessage{
			ID:        assistantMsgID,
			Role:      "assistant",
			Content:   resp,
			Timestamp: time.Now().Unix(),
		}
		if err := a.chatService.AddMessage(threadID, assistantMsg); err != nil {
			a.Log(fmt.Sprintf("[ERROR] Failed to save free chat assistant message: %v", err))
		}

		// Emit thread-updated event to refresh the UI
		runtime.EventsEmit(a.ctx, "thread-updated", threadID)
	}

	// Log response
	if threadID != "" && cfg.DetailedLog {
		a.logChatToFile(threadID, "FREE CHAT RESPONSE", resp)
	}

	a.Log(fmt.Sprintf("[FREE-CHAT] Completed in %v", chatDuration))
	a.Log(fmt.Sprintf("[TIMING] Total SendFreeChatMessage took: %v", time.Since(startTotal)))

	return resp, nil
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
		"新闻", "最新", "今天", "现在", "实时", "当前",
		"股票", "股价", "汇率", "价格", "多少钱",
		"搜索", "查询", "查一下", "帮我查", "帮我搜",
		"网上", "网络", "互联网",
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
		return "抱歉，未能获取到有效信息。"
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
		searchTool, err := agent.NewSearchAPITool(a.Log, activeAPI)
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

	var systemPrompt string
	if hasSearchTool {
		systemPrompt = fmt.Sprintf(`You are a helpful AI assistant with web search capability. You MUST use tools to get real-time information.

🔧 AVAILABLE TOOLS:
%s

⚡ CRITICAL: TOOL SELECTION RULES

🌤️ WEATHER → web_fetch with wttr.in (NOT web_search!)
   - "天气怎样?" → get_device_location → web_fetch("https://wttr.in/{city}?format=3")
   - "北京天气" → web_fetch("https://wttr.in/Beijing?format=3")

✈️ FLIGHTS/机票 → web_search (MUST use web_search, NOT web_fetch!)
   - "去成都的机票" → get_device_location → web_search("{出发城市} 到 成都 机票")
   - "北京到上海航班" → web_search("北京 到 上海 航班 机票")
   - "flights to Tokyo" → web_search("flights to Tokyo from {city}")

📰 NEWS/新闻 → web_search
   - "最新新闻" → web_search("今日新闻 头条")

📈 STOCKS/股票 → web_search
   - "苹果股价" → web_search("苹果股票价格 AAPL")

💱 EXCHANGE/汇率 → web_search
   - "美元汇率" → web_search("美元 人民币 汇率")

🏨 HOTELS/酒店 → web_search
   - "附近酒店" → get_device_location → web_search("{city} 酒店推荐")

⏰ TIME/时间 → get_local_time (NO internet needed!)
   - "现在几点?" → get_local_time(query_type="current_time")

📍 LOCATION/位置 → get_device_location
   - "我在哪?" → get_device_location()

🚨 CRITICAL RULES:
1. ⚠️ web_fetch is ONLY for:
   - Weather via wttr.in API
   - Reading full content from URLs found in web_search results
2. ⚠️ web_fetch CANNOT be used for flights, stocks, news, hotels - these sites need JavaScript!
3. ✅ For flights/stocks/news/hotels → ALWAYS use web_search first!
4. NEVER say "I cannot search" - YOU HAVE web_search!
5. NEVER tell user to visit websites - get the info yourself!

📋 WORKFLOW EXAMPLES:

Example 1: "天气怎样?" / "今天几度?"
→ Step 1: get_device_location (get city)
→ Step 2: web_fetch(url="https://wttr.in/{city}?format=3")
→ Step 3: Analyze and answer

Example 2: "去成都的机票" / "今天还有去成都的机票吗?"
→ Step 1: get_device_location (get departure city, e.g., "San Jose")
→ Step 2: web_search("San Jose 到 成都 机票 航班") ← MUST use web_search!
→ Step 3: Summarize flight options from search results

Example 3: "北京到上海航班"
→ Step 1: web_search("北京 到 上海 航班 机票 今天")
→ Step 2: Summarize flight options

Example 4: "苹果股价"
→ Step 1: web_search("苹果股票价格 AAPL 实时")
→ Step 2: Report stock price from results

Example 5: "最新新闻"
→ Step 1: web_search("今日新闻 头条 最新")
→ Step 2: Summarize top news

🎯 SUMMARY:
- Weather → web_fetch with wttr.in
- Flights/Stocks/News/Hotels → web_search (NEVER web_fetch!)
- Time → get_local_time
- Location → get_device_location

Please respond in %s.`, toolDescriptions.String(), langPrompt)
	} else {
		// No web search available - but time and location tools are always available
		systemPrompt = fmt.Sprintf(`You are a helpful AI assistant with local tools and limited web access.

⚠️ IMPORTANT: No search API is configured. You CANNOT search the web for real-time information.

CRITICAL RULES:
1. For TIME/DATE questions → Use get_local_time tool (instant, accurate!)
2. For LOCATION questions → Use get_device_location tool
3. For WEATHER questions → Use web_fetch with wttr.in API (FREE, works without search API!)
4. For other real-time info (news, stocks, flights, etc.) → Politely explain search API is needed
5. ⚠️ DO NOT try to use web_fetch for flights, stocks, news - these sites require JavaScript and won't work!

Available tools:
%s

=== WHAT YOU CAN DO (NO SEARCH API NEEDED) ===

✅ TIME/DATE: Use get_local_time
   - "现在几点?" → get_local_time(query_type="current_time")
   - "今天星期几?" → get_local_time(query_type="weekday")
   - "今天几号?" → get_local_time(query_type="current_date")

✅ LOCATION: Use get_device_location
   - "我在哪?" → get_device_location()

✅ WEATHER: Use web_fetch with wttr.in (FREE API - plain text, no JavaScript!)
   WORKFLOW:
   1. get_device_location → get city
   2. If unavailable, use Beijing as default
   3. web_fetch(url="https://wttr.in/{city}?format=3")
   
   Examples:
   - "天气怎样?" → get_device_location, then web_fetch("https://wttr.in/{city}?format=3")
   - "北京天气" → web_fetch("https://wttr.in/Beijing?format=3")
   - "上海今天几度?" → web_fetch("https://wttr.in/Shanghai?format=3")

=== WHAT YOU CANNOT DO (NEEDS SEARCH API) ===

❌ The following queries require a search API to be configured:
   - 航班/Flights: "北京到上海的航班", "明天飞深圳", "去成都的机票"
   - 股票/Stocks: "苹果股价", "茅台股票多少钱"
   - 新闻/News: "最新新闻", "今天有什么新闻"
   - 酒店/Hotels: "附近酒店", "三亚酒店推荐"
   - 比赛/Sports: "今天有什么比赛", "NBA比分"
   - 汇率/Exchange: "美元汇率", "人民币兑日元"

⚠️ DO NOT try to use web_fetch for these queries! Most flight/stock/news websites require JavaScript to render content, and web_fetch can only read static HTML.

When user asks for flights, stocks, news, etc., respond like this:
- Chinese: "抱歉，查询航班/股票/新闻等实时信息需要配置搜索引擎。请在「设置」→「搜索API」中启用 Serper 或 UAPI Pro 后再试。目前我只能帮您查询天气、时间和位置信息。"
- English: "Sorry, querying flights/stocks/news requires a search API. Please enable Serper or UAPI Pro in Settings → Search API. Currently I can only help with weather, time, and location queries."

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
						errorMsg := "抱歉，处理请求时遇到问题。请稍后重试。"
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
					errorMsg := "抱歉，无法生成回复。请尝试重新提问。"
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
	a.cancelAnalysisMutex.Lock()

	// Check if there's any active analysis
	a.activeThreadsMutex.RLock()
	hasActiveAnalysis := len(a.activeThreads) > 0
	a.activeThreadsMutex.RUnlock()

	if !hasActiveAnalysis {
		a.cancelAnalysisMutex.Unlock()
		return fmt.Errorf("no analysis is currently running")
	}

	a.cancelAnalysis = true
	a.Log(fmt.Sprintf("[CANCEL] Analysis cancellation requested for thread: %s", a.activeThreadID))
	a.cancelAnalysisMutex.Unlock()

	// Wait for the analysis to actually stop (with timeout)
	// This ensures activeThreads is properly cleaned up before returning
	maxWaitTime := 5 * time.Second
	checkInterval := 100 * time.Millisecond
	startTime := time.Now()

	for {
		a.activeThreadsMutex.RLock()
		stillActive := len(a.activeThreads) > 0
		a.activeThreadsMutex.RUnlock()

		if !stillActive {
			a.Log("[CANCEL] Analysis successfully cancelled and cleaned up")
			return nil
		}

		if time.Since(startTime) > maxWaitTime {
			a.Log("[CANCEL] Timeout waiting for analysis to stop, forcing cleanup")
			// Force cleanup of activeThreads
			a.activeThreadsMutex.Lock()
			for threadID := range a.activeThreads {
				delete(a.activeThreads, threadID)
				a.Log(fmt.Sprintf("[CANCEL] Force removed thread from activeThreads: %s", threadID))
			}
			a.activeThreadsMutex.Unlock()
			return nil
		}

		time.Sleep(checkInterval)
	}
}

// IsCancelRequested checks if analysis cancellation has been requested
func (a *App) IsCancelRequested() bool {
	a.cancelAnalysisMutex.Lock()
	defer a.cancelAnalysisMutex.Unlock()
	return a.cancelAnalysis
}

// GetActiveThreadID returns the currently active thread ID
func (a *App) GetActiveThreadID() string {
	a.cancelAnalysisMutex.Lock()
	defer a.cancelAnalysisMutex.Unlock()
	return a.activeThreadID
}

// GetActiveAnalysisCount returns the current number of active analysis sessions
func (a *App) GetActiveAnalysisCount() int {
	a.activeThreadsMutex.RLock()
	defer a.activeThreadsMutex.RUnlock()
	return len(a.activeThreads)
}

// CanStartNewAnalysis checks if a new analysis can be started based on concurrent limit
func (a *App) CanStartNewAnalysis() (bool, string) {
	cfg, _ := a.GetConfig()
	maxConcurrent := cfg.MaxConcurrentAnalysis
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default to 5
	}
	if maxConcurrent > 10 {
		maxConcurrent = 10 // Cap at 10
	}

	a.activeThreadsMutex.RLock()
	activeCount := len(a.activeThreads)
	a.activeThreadsMutex.RUnlock()

	if activeCount >= maxConcurrent {
		// Get current language configuration
		var errorMessage string
		if cfg.Language == "简体中文" {
			errorMessage = fmt.Sprintf("当前已有 %d 个分析会话进行中（最大并发数：%d）。请等待部分分析完成后再开始新的分析，或在设置中增加最大并发分析任务数。", activeCount, maxConcurrent)
		} else {
			errorMessage = fmt.Sprintf("There are currently %d analysis sessions in progress (max concurrent: %d). Please wait for some analyses to complete before starting a new analysis, or increase the max concurrent analysis limit in settings.", activeCount, maxConcurrent)
		}
		return false, errorMessage
	}

	return true, ""
}

// attachChartToUserMessage attaches chart data to a specific user message in a thread
func (a *App) attachChartToUserMessage(threadID, messageID string, chartData *ChartData) {
	if a.chatService == nil {
		return
	}

	threads, err := a.chatService.LoadThreads()
	if err != nil {
		a.Log(fmt.Sprintf("attachChartToUserMessage: Failed to load threads: %v", err))
		return
	}

	// Find the target thread
	var targetThread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			targetThread = &threads[i]
			break
		}
	}

	if targetThread == nil {
		a.Log(fmt.Sprintf("attachChartToUserMessage: Thread %s not found", threadID))
		return
	}

	// Find the target message
	found := false
	if messageID != "" {
		// Strict mode: Find exact message ID
		for i := range targetThread.Messages {
			if targetThread.Messages[i].ID == messageID {
				targetThread.Messages[i].ChartData = chartData
				a.Log(fmt.Sprintf("[CHART] Attached chart to specific user message: %s", messageID))
				found = true
				break
			}
		}
		if !found {
			a.Log(fmt.Sprintf("attachChartToUserMessage: Message %s not found in thread %s", messageID, threadID))
		}
	} else {
		// Legacy mode: Find last user message
		for i := len(targetThread.Messages) - 1; i >= 0; i-- {
			if targetThread.Messages[i].Role == "user" {
				targetThread.Messages[i].ChartData = chartData
				a.Log(fmt.Sprintf("[CHART] Attached chart to last user message: %s (Fallback)", targetThread.Messages[i].ID))
				found = true
				break
			}
		}
	}

	// Save the updated thread
	if found {
		if err := a.chatService.SaveThreads([]ChatThread{*targetThread}); err != nil {
			a.Log(fmt.Sprintf("attachChartToUserMessage: Failed to save thread: %v", err))
		}
	}
}

// attachChartToLastAssistantMessage attaches chart data to the last assistant message in a thread
func (a *App) attachChartToLastAssistantMessage(threadID string, chartData *ChartData) {
	if a.chatService == nil {
		return
	}

	threads, err := a.chatService.LoadThreads()
	if err != nil {
		a.Log(fmt.Sprintf("attachChartToLastAssistantMessage: Failed to load threads: %v", err))
		return
	}

	// Find the target thread
	var targetThread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			targetThread = &threads[i]
			break
		}
	}

	if targetThread == nil {
		a.Log(fmt.Sprintf("attachChartToLastAssistantMessage: Thread %s not found", threadID))
		return
	}

	// Find the last assistant message
	found := false
	for i := len(targetThread.Messages) - 1; i >= 0; i-- {
		if targetThread.Messages[i].Role == "assistant" {
			targetThread.Messages[i].ChartData = chartData
			a.Log(fmt.Sprintf("[CHART] Attached chart to last assistant message: %s", targetThread.Messages[i].ID))
			found = true
			break
		}
	}

	if !found {
		a.Log(fmt.Sprintf("attachChartToLastAssistantMessage: No assistant message found in thread %s", threadID))
		return
	}

	// Save the updated thread
	if err := a.chatService.SaveThreads([]ChatThread{*targetThread}); err != nil {
		a.Log(fmt.Sprintf("attachChartToLastAssistantMessage: Failed to save thread: %v", err))
	}
}

// detectAndEmitImages detects images in the response and emits analysis-result-update events
// It uses the ImageDetector to find images in various formats (base64, markdown, file references)
// and emits separate events for each detected image
func (a *App) detectAndEmitImages(response string, threadID string, userMessageID string, requestID string) {
	if response == "" || threadID == "" {
		return
	}

	// Create a new ImageDetector
	detector := agent.NewImageDetector()

	// Detect all images in the response
	images := detector.DetectAllImages(response)

	if len(images) == 0 {
		a.Log("[CHART] No images detected in response")
		return
	}

	a.Log(fmt.Sprintf("[CHART] Detected %d image(s) in response", len(images)))

	// Emit separate events for each detected image
	for i, img := range images {
		// Extract the image data based on type
		var imageData string

		switch img.Type {
		case "base64":
			// For base64 images, use the full data URL
			imageData = img.Data
			a.Log(fmt.Sprintf("[CHART] Detected inline base64 image (%d/%d)", i+1, len(images)))

		case "markdown":
			// For markdown images, the data is the path
			// Check if it's already a data URL or needs conversion
			if strings.HasPrefix(img.Data, "data:") {
				imageData = img.Data
			} else if strings.HasPrefix(img.Data, "http://") || strings.HasPrefix(img.Data, "https://") {
				// HTTP URL - use directly
				imageData = img.Data
				a.Log(fmt.Sprintf("[CHART] Detected markdown image with HTTP URL (%d/%d)", i+1, len(images)))
			} else {
				// File path - will be handled by frontend
				imageData = img.Data
				a.Log(fmt.Sprintf("[CHART] Detected markdown image with file path (%d/%d): %s", i+1, len(images), img.Data))
			}

		case "file_reference":
			// For file references, the data is the filename
			// Construct a file reference that the frontend can use
			imageData = "files/" + img.Data
			a.Log(fmt.Sprintf("[CHART] Detected file reference image (%d/%d): %s", i+1, len(images), img.Data))

		case "sandbox":
			// For sandbox paths (OpenAI code interpreter format), the data is the filename
			// Construct a file reference that the frontend can use
			imageData = "files/" + img.Data
			a.Log(fmt.Sprintf("[CHART] Detected sandbox path image (%d/%d): %s", i+1, len(images), img.Data))

		default:
			a.Log(fmt.Sprintf("[CHART] Unknown image type: %s", img.Type))
			continue
		}

		// Use event aggregator for new unified event system
		if a.eventAggregator != nil {
			a.eventAggregator.AddImage(threadID, userMessageID, requestID, imageData, "")
		}

		a.Log(fmt.Sprintf("[CHART] Emitted analysis-result-update event for image (%d/%d)", i+1, len(images)))
	}
}

// filterFalseFileClaimsIfECharts filters out false file generation claims when ECharts is used
// LLM sometimes hallucinates file generation (e.g., "图表已生成: xxx.pdf") when using ECharts,
// but ECharts only renders in the frontend and doesn't generate any files.
func (a *App) filterFalseFileClaimsIfECharts(response string) string {
	// Check if response contains ECharts
	hasECharts := strings.Contains(response, "json:echarts")
	
	if !hasECharts {
		return response // No ECharts, no filtering needed
	}
	
	// Check if response also contains python_executor output (actual file generation)
	// If python_executor was used, files might be real
	hasPythonOutput := strings.Contains(response, "✅ 图表已保存") || 
		strings.Contains(response, "✅ Chart saved") ||
		strings.Contains(response, "plt.savefig") ||
		strings.Contains(response, "FILES_DIR")
	
	if hasPythonOutput {
		return response // Python was used, files might be real
	}
	
	// ECharts is used but no Python execution - filter false file claims
	a.Log("[FILTER] Detected ECharts without Python execution, filtering false file claims")
	
	// Patterns that indicate false file generation claims
	// These patterns match common LLM hallucinations about file generation
	falseClaimPatterns := []string{
		// Chinese patterns - match file generation claims with file sizes
		"(?i)图表文件已生成[：:]\\s*`?[^`\\n]+\\.(pdf|png|jpg|jpeg|xlsx|csv)`?\\s*\\([^)]*\\)",
		"(?i)✅\\s*[^：:\\n]+[：:]\\s*`?[^`\\n]+\\.(pdf|png|jpg|jpeg)`?\\s*\\([^)]*\\)",
		"(?i)图表已生成[：:]\\s*`?[^`\\n]+\\.(pdf|png|jpg|jpeg)`?",
		"(?i)已保存[到至]\\s*`?[^`\\n]+\\.(pdf|png|jpg|jpeg)`?",
		"(?i)文件已生成[：:]\\s*`?[^`\\n]+\\.(pdf|png|jpg|jpeg|xlsx|csv)`?",
		// English patterns
		"(?i)chart\\s+(?:file\\s+)?generated[：:]\\s*`?[^`\\n]+\\.(pdf|png|jpg|jpeg)`?",
		"(?i)saved\\s+(?:to|as)[：:]\\s*`?[^`\\n]+\\.(pdf|png|jpg|jpeg)`?",
	}
	
	result := response
	for _, pattern := range falseClaimPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(result) {
			a.Log(fmt.Sprintf("[FILTER] Removing false file claim matching pattern: %s", pattern))
			result = re.ReplaceAllString(result, "")
		}
	}
	
	// Also remove lines that look like file size claims without actual files
	// e.g., "(32.18 KB)" or "(28.47 KB)" standalone
	// fileSizePattern := regexp.MustCompile(`\s*\(\d+\.?\d*\s*[KMG]?B\)\s*`)
	
	// Only remove file size if it appears after a filename pattern that was removed
	// This is a more conservative approach
	
	// Clean up any double newlines created by removal
	result = regexp.MustCompile("\\n{3,}").ReplaceAllString(result, "\n\n")
	
	if result != response {
		a.Log("[FILTER] False file claims were filtered from response")
	}
	
	return result
}

func (a *App) logChatToFile(threadID, role, content string) {
	// Use DataCacheDir for logs
	cfg, _ := a.GetConfig()

	// Construct path: sessions/<threadID>/chat.log
	logPath := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "chat.log")

	// Ensure dir exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		a.Log(fmt.Sprintf("logChatToFile: Failed to create log directory: %v", err))
		return
	}

	// Append log
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		a.Log(fmt.Sprintf("logChatToFile: Failed to open log file %s: %v", logPath, err))
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] [%s]\n%s\n\n--------------------------------------------------\n\n", timestamp, role, content)
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
	return a.pythonService.ProbePythonEnvironments()
}

// ValidatePython checks the given Python path
func (a *App) ValidatePython(path string) agent.PythonValidationResult {
	return a.pythonService.ValidatePythonEnvironment(path)
}

// InstallPythonPackages installs missing packages for the given Python environment
func (a *App) InstallPythonPackages(pythonPath string, packages []string) error {
	return a.pythonService.InstallMissingPackages(pythonPath, packages)
}

// CreateVantageDataEnvironment creates a dedicated virtual environment for VantageData
func (a *App) CreateVantageDataEnvironment() (string, error) {
	return a.pythonService.CreateVantageDataEnvironment()
}

// CheckVantageDataEnvironmentExists checks if a vantagedata environment already exists
func (a *App) CheckVantageDataEnvironmentExists() bool {
	return a.pythonService.CheckVantageDataEnvironmentExists()
}

// DiagnosePythonInstallation provides detailed diagnostic information about Python installations
func (a *App) DiagnosePythonInstallation() map[string]interface{} {
	return a.pythonService.DiagnosePythonInstallation()
}

// GetChatHistory loads the chat history
func (a *App) GetChatHistory() ([]ChatThread, error) {
	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}
	return a.chatService.LoadThreads()
}

// GetChatHistoryByDataSource loads chat history for a specific data source
func (a *App) GetChatHistoryByDataSource(dataSourceID string) ([]ChatThread, error) {
	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}
	return a.chatService.GetThreadsByDataSource(dataSourceID)
}

// CheckSessionNameExists checks if a session name already exists for a data source
func (a *App) CheckSessionNameExists(dataSourceID string, sessionName string) (bool, error) {
	if a.chatService == nil {
		return false, fmt.Errorf("chat service not initialized")
	}
	return a.chatService.CheckSessionNameExists(dataSourceID, sessionName)
}

// SaveChatHistory saves the chat history
func (a *App) SaveChatHistory(threads []ChatThread) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}
	return a.chatService.SaveThreads(threads)
}

// DeleteThread deletes a specific chat thread
func (a *App) DeleteThread(threadID string) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	// Check if this thread is currently running analysis
	a.cancelAnalysisMutex.Lock()
	isActiveThread := a.activeThreadID == threadID

	a.activeThreadsMutex.RLock()
	isGenerating := a.activeThreads[threadID]
	a.activeThreadsMutex.RUnlock()

	if isActiveThread && isGenerating {
		// Cancel the ongoing analysis for this thread
		a.cancelAnalysis = true
		a.Log(fmt.Sprintf("[DELETE-THREAD] Cancelling ongoing analysis for thread: %s", threadID))
	}
	a.cancelAnalysisMutex.Unlock()

	// Wait a moment for cancellation to take effect if needed
	if isActiveThread && isGenerating {
		time.Sleep(100 * time.Millisecond)
		a.Log(fmt.Sprintf("[DELETE-THREAD] Waited for analysis cancellation"))
	}

	// Delete the thread
	err := a.chatService.DeleteThread(threadID)
	if err != nil {
		return err
	}

	// If the deleted thread was active, clear dashboard data
	if isActiveThread {
		a.Log(fmt.Sprintf("[DELETE-THREAD] Clearing dashboard data for deleted active thread: %s", threadID))
		// Use new unified event system
		if a.eventAggregator != nil {
			a.eventAggregator.Clear(threadID)
		}
	}

	return nil
}

// CreateChatThread creates a new chat thread with a unique title
func (a *App) CreateChatThread(dataSourceID, title string) (ChatThread, error) {
	if a.chatService == nil {
		return ChatThread{}, fmt.Errorf("chat service not initialized")
	}

	// Note: We no longer check concurrent analysis limit here.
	// The limit is enforced in SendMessage, where the analysis will wait in queue
	// if the limit is reached. This allows users to create sessions freely,
	// and the waiting indicator will be shown in the chat area.

	thread, err := a.chatService.CreateThread(dataSourceID, title)
	if err != nil {
		return ChatThread{}, err
	}

	// If data source is selected, check for existing analysis and inject into memory
	/*
		if dataSourceID != "" {
			sources, _ := a.dataSourceService.LoadDataSources()
			var target *agent.DataSource
			for _, ds := range sources {
				if ds.ID == dataSourceID {
					target = &ds
					break
				}
			}

			if target != nil && target.Analysis != nil {
				// Generate suggestions based on this analysis
				go a.generateAnalysisSuggestions(thread.ID, target.Analysis)
			}
		}
	*/

	return thread, nil
}

func (a *App) generateAnalysisSuggestions(threadID string, analysis *agent.DataSourceAnalysis) {
	if a.chatService == nil {
		return
	}

	// Notify frontend that background task started
	runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
		"loading":  true,
		"threadId": threadID,
	})
	defer runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
		"loading":  false,
		"threadId": threadID,
	})

	cfg, _ := a.GetEffectiveConfig()
	langPrompt := a.getLangPrompt(cfg)

	// Construct prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Based on the following data source summary and schema, please suggest 3-5 distinct business analysis questions that would provide valuable insights for decision-making. Please answer in %s.\n\nIMPORTANT GUIDELINES:\n- Focus on BUSINESS VALUE and INSIGHTS, not technical implementation\n- Use simple, non-technical language that any business user can understand\n- Frame suggestions as business questions or outcomes (e.g., \"Understand customer purchasing patterns\" instead of \"Run RFM analysis\")\n- DO NOT mention SQL, Python, data processing, or any technical terms\n- Focus on what insights can be discovered, not how to discover them\n\nProvide the suggestions as a clear, structured, numbered list (1., 2., 3...). Each suggestion should include:\n- A clear, business-focused title\n- A one-sentence description of what business insights this would reveal\n\nEnd your response by telling the user (in %s) that they can select one or more analysis questions by replying with the corresponding number(s).", langPrompt, langPrompt))
	sb.WriteString(fmt.Sprintf("Summary: %s\n\n", analysis.Summary))
	sb.WriteString("Schema:\n")
	for _, table := range analysis.Schema {
		sb.WriteString(fmt.Sprintf("- Table: %s, Columns: %s\n", table.TableName, strings.Join(table.Columns, ", ")))
	}

	prompt := sb.String()
	llm := agent.NewLLMService(cfg, a.Log)

	resp, err := llm.Chat(context.Background(), prompt)
	if err != nil {
		a.Log(fmt.Sprintf("Failed to generate suggestions: %v", err))
		return
	}

	// Add message to chat
	msg := ChatMessage{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Role:      "assistant",
		Content:   resp, // LLM response already contains the suggestions and instructions
		Timestamp: time.Now().Unix(),
	}

	if err := a.chatService.AddMessage(threadID, msg); err != nil {
		a.Log(fmt.Sprintf("Failed to add suggestion message: %v", err))
		return
	}

	// Parse suggestions and emit to dashboard insights area
	// Note: analysis struct doesn't have ID/Name fields, insights will be generic
	insights := a.parseSuggestionsToInsights(resp, "", "")
	if len(insights) > 0 {
		a.Log(fmt.Sprintf("Emitting %d suggestions to dashboard insights", len(insights)))
		// Use event aggregator for new unified event system
		if a.eventAggregator != nil {
			for _, insight := range insights {
				a.eventAggregator.AddInsight(threadID, msg.ID, "", insight)
			}
			a.eventAggregator.FlushNow(threadID, true)
		}
	}

	runtime.EventsEmit(a.ctx, "thread-updated", threadID)
}

// parseSuggestionsToInsights extracts numbered suggestions from LLM response and converts to Insight objects
func (a *App) parseSuggestionsToInsights(llmResponse, dataSourceID, dataSourceName string) []Insight {
	var insights []Insight
	lines := strings.Split(llmResponse, "\n")

	// Match lines starting with "1.", "2.", etc
	numberPattern := regexp.MustCompile(`^\s*(\d+)\.\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := numberPattern.FindStringSubmatch(line); len(matches) > 2 {
			// Extract the suggestion text (everything after the number)
			suggestionText := strings.TrimSpace(matches[2])
			if suggestionText != "" {
				insights = append(insights, Insight{
					Text:         suggestionText,
					Icon:         "lightbulb",
					DataSourceID: dataSourceID,
					SourceName:   dataSourceName,
				})
			}
		}
	}

	return insights
}

func (a *App) analyzeDataSource(dataSourceID string) {
	startTotal := time.Now()
	if a.dataSourceService == nil {
		return
	}

	a.Log(fmt.Sprintf("Starting analysis for source %s", dataSourceID))

	// 1. Get Tables
	startTables := time.Now()
	tables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("Failed to get tables: %v", err))
		return
	}
	a.Log(fmt.Sprintf("[TIMING] Getting tables took: %v", time.Since(startTables)))

	// 2. Sample Data & Construct Prompt
	startSample := time.Now()
	cfg, _ := a.GetEffectiveConfig()
	langPrompt := a.getLangPrompt(cfg)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("I am starting a new analysis on this database. Based on the following schema and first row of data, please provide exactly two sentences in %s: the first sentence should describe the industry background of this data, and the second sentence should provide a concise overview of the data source content.\n\n", langPrompt))

	var tableSchemas []agent.TableSchema

	for _, tableName := range tables {
		sb.WriteString(fmt.Sprintf("Table: %s\n", tableName))

		// Get 1 row
		data, err := a.dataSourceService.GetDataSourceTableData(dataSourceID, tableName, 1)
		if err != nil {
			sb.WriteString("(Failed to fetch data)\n")
			continue
		}

		var cols []string
		if len(data) > 0 {
			// Extract columns from first row keys
			for k := range data[0] {
				cols = append(cols, k)
			}
			sb.WriteString(fmt.Sprintf("Columns: %s\nData:\n", strings.Join(cols, ", ")))

			for _, row := range data {
				// Simple values formatting
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

		// Add to schema list
		if len(cols) > 0 {
			tableSchemas = append(tableSchemas, agent.TableSchema{
				TableName: tableName,
				Columns:   cols,
			})
		}
	}
	a.Log(fmt.Sprintf("[TIMING] Data sampling and prompt construction took: %v", time.Since(startSample)))

	// 3. Call LLM
	prompt := sb.String()

	// Log prompt to system log if detailed logging is on (or creates a special log file?)
	// Since we don't have a threadID, logChatToFile needs a path.
	// We can log to "system_analysis.log" or similar?
	// Or just skip file logging for background tasks and use main log.
	if cfg.DetailedLog {
		a.Log("Sending Analysis Prompt to LLM...")
	}

	llm := agent.NewLLMService(cfg, a.Log)
	startLLM := time.Now()
	description, err := llm.Chat(context.Background(), prompt)
	a.Log(fmt.Sprintf("[TIMING] Background LLM Analysis took: %v", time.Since(startLLM)))

	if err != nil {
		a.Log(fmt.Sprintf("LLM Analysis failed: %v", err))
		return
	}

	if description == "" {
		a.Log("LLM returned empty response during analysis.")
		description = "No description provided by LLM."
	}

	// 4. Save Analysis to DataSource
	startSave := time.Now()
	analysis := agent.DataSourceAnalysis{
		Summary: description,
		Schema:  tableSchemas,
	}

	if err := a.dataSourceService.UpdateAnalysis(dataSourceID, analysis); err != nil {
		a.Log(fmt.Sprintf("Failed to update data source analysis: %v", err))
		return
	}
	a.Log(fmt.Sprintf("[TIMING] Saving analysis result took: %v", time.Since(startSave)))
	a.Log(fmt.Sprintf("[TIMING] Total Background Analysis took: %v", time.Since(startTotal)))

	a.Log("Data Source Analysis complete and saved.")
}

// UpdateThreadTitle updates the title of a chat thread
func (a *App) UpdateThreadTitle(threadID, newTitle string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}
	return a.chatService.UpdateThreadTitle(threadID, newTitle)
}

// ClearHistory clears all chat history
func (a *App) ClearHistory() error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	// Check if there's an ongoing analysis and cancel it
	a.cancelAnalysisMutex.Lock()
	a.activeThreadsMutex.RLock()
	hasActiveAnalysis := len(a.activeThreads) > 0
	a.activeThreadsMutex.RUnlock()

	if hasActiveAnalysis {
		// Cancel any ongoing analysis
		a.cancelAnalysis = true
		a.Log("[CLEAR-HISTORY] Cancelling ongoing analysis before clearing history")
	}
	a.cancelAnalysisMutex.Unlock()

	// Wait for cancellation to take effect if needed
	if hasActiveAnalysis {
		time.Sleep(100 * time.Millisecond)
		a.Log("[CLEAR-HISTORY] Waited for analysis cancellation")
	}

	// Clear all history
	err := a.chatService.ClearHistory()
	if err != nil {
		return err
	}

	// Clear dashboard data since all threads are deleted
	a.Log("[CLEAR-HISTORY] Clearing dashboard data after clearing all history")
	// Use new unified event system
	if a.eventAggregator != nil {
		a.eventAggregator.Clear("")
	}

	return nil
}

// --- Data Source Management ---

// GetDataSources returns the list of registered data sources
func (a *App) GetDataSources() ([]agent.DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.LoadDataSources()
}

// GetDataSourceStatistics returns aggregated statistics about all data sources
// Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5
func (a *App) GetDataSourceStatistics() (*agent.DataSourceStatistics, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	// Load all data sources
	dataSources, err := a.dataSourceService.LoadDataSources()
	if err != nil {
		return nil, fmt.Errorf("failed to load data sources: %w", err)
	}

	// Calculate statistics
	stats := &agent.DataSourceStatistics{
		TotalCount:      len(dataSources),
		BreakdownByType: make(map[string]int),
		DataSources:     make([]agent.DataSourceSummary, 0, len(dataSources)),
	}

	// Group by type and build summaries
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
// Validates: Requirements 4.1, 4.2, 4.5
func (a *App) StartDataSourceAnalysis(dataSourceID string) (string, error) {
	if a.dataSourceService == nil {
		return "", fmt.Errorf("data source service not initialized")
	}

	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	// Validate data source exists
	dataSources, err := a.dataSourceService.LoadDataSources()
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

	// Create a new chat thread for this data source analysis
	// Use data source name as the session title
	sessionTitle := fmt.Sprintf("分析: %s", targetDS.Name)
	thread, err := a.chatService.CreateThread(dataSourceID, sessionTitle)
	if err != nil {
		return "", fmt.Errorf("failed to create chat thread: %w", err)
	}

	threadID := thread.ID

	// Construct analysis prompt in Chinese (mention data source name and type)
	prompt := fmt.Sprintf("请分析数据源 '%s' (%s)，提供数据概览、关键指标和洞察。", 
		targetDS.Name, targetDS.Type)

	// Generate unique message ID for tracking
	userMessageID := fmt.Sprintf("ds-msg-%d", time.Now().UnixNano())

	// Log analysis initiation
	a.Log(fmt.Sprintf("[DATASOURCE-ANALYSIS] Starting analysis for %s (thread: %s, msgId: %s)", 
		dataSourceID, threadID, userMessageID))

	// Emit event to notify frontend that analysis is starting
	runtime.EventsEmit(a.ctx, "chat-loading", map[string]interface{}{
		"loading":  true,
		"threadId": threadID,
	})

	// Call SendMessage asynchronously so we can return the threadID immediately
	go func() {
		_, err := a.SendMessage(threadID, prompt, userMessageID, "")
		if err != nil {
			a.Log(fmt.Sprintf("[DATASOURCE-ANALYSIS] Error: %v", err))
			// Emit error event to frontend
			runtime.EventsEmit(a.ctx, "analysis-error", map[string]interface{}{
				"threadId": threadID,
				"message":  err.Error(),
				"code":     "ANALYSIS_ERROR",
			})
		}
	}()

	// Return thread ID immediately (analysis runs in background)
	return threadID, nil
}

// ImportExcelDataSource imports an Excel file as a data source
func (a *App) ImportExcelDataSource(name string, filePath string) (*agent.DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt, "", "") // Task 3.1: Added empty requestId for internal call
	}

	ds, err := a.dataSourceService.ImportExcel(name, filePath, headerGen)
	if err == nil && ds != nil {
		go a.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// ImportCSVDataSource imports a CSV directory as a data source
func (a *App) ImportCSVDataSource(name string, dirPath string) (*agent.DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt, "", "") // Task 3.1: Added empty requestId for internal call
	}

	ds, err := a.dataSourceService.ImportCSV(name, dirPath, headerGen)
	if err == nil && ds != nil {
		go a.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// ImportJSONDataSource imports a JSON file as a data source
func (a *App) ImportJSONDataSource(name string, filePath string) (*agent.DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt, "", "") // Task 3.1: Added empty requestId for internal call
	}

	ds, err := a.dataSourceService.ImportJSON(name, filePath, headerGen)
	if err == nil && ds != nil {
		go a.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// ShopifyOAuthConfig holds the Shopify OAuth configuration
type ShopifyOAuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// shopifyOAuthService holds the active OAuth service instance
var shopifyOAuthService *agent.ShopifyOAuthService
var shopifyOAuthMutex sync.Mutex

// GetShopifyOAuthConfig returns the Shopify OAuth configuration
// Developer should set these values
func (a *App) GetShopifyOAuthConfig() ShopifyOAuthConfig {
	// These should be configured by the developer
	// For now, return empty - developer needs to set these
	cfg, _ := a.GetConfig()
	return ShopifyOAuthConfig{
		ClientID:     cfg.ShopifyClientID,
		ClientSecret: cfg.ShopifyClientSecret,
	}
}

// StartShopifyOAuth initiates the Shopify OAuth flow
// Returns the authorization URL that should be opened in browser
func (a *App) StartShopifyOAuth(shop string) (string, error) {
	shopifyOAuthMutex.Lock()
	defer shopifyOAuthMutex.Unlock()

	// Get OAuth config
	cfg, err := a.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %v", err)
	}

	if cfg.ShopifyClientID == "" || cfg.ShopifyClientSecret == "" {
		return "", fmt.Errorf("Shopify OAuth not configured. Please set Client ID and Client Secret in settings.")
	}

	// Create OAuth service
	oauthConfig := agent.ShopifyOAuthConfig{
		ClientID:     cfg.ShopifyClientID,
		ClientSecret: cfg.ShopifyClientSecret,
		Scopes:       "read_orders,read_products,read_customers,read_inventory",
	}
	shopifyOAuthService = agent.NewShopifyOAuthService(oauthConfig, a.Log)

	// Get authorization URL
	authURL, _, err := shopifyOAuthService.GetAuthURL(shop)
	if err != nil {
		return "", err
	}

	// Start callback server
	if err := shopifyOAuthService.StartCallbackServer(a.ctx); err != nil {
		return "", err
	}

	a.Log(fmt.Sprintf("[SHOPIFY-OAUTH] Started OAuth flow for shop: %s", shop))
	return authURL, nil
}

// WaitForShopifyOAuth waits for the OAuth flow to complete
// Returns the access token and shop URL on success
func (a *App) WaitForShopifyOAuth() (map[string]string, error) {
	shopifyOAuthMutex.Lock()
	service := shopifyOAuthService
	shopifyOAuthMutex.Unlock()

	if service == nil {
		return nil, fmt.Errorf("OAuth flow not started")
	}

	// Wait for result with 5 minute timeout
	result := service.WaitForResult(5 * time.Minute)

	// Stop the callback server
	service.StopCallbackServer()

	// Clear the service
	shopifyOAuthMutex.Lock()
	shopifyOAuthService = nil
	shopifyOAuthMutex.Unlock()

	if result.Error != "" {
		return nil, fmt.Errorf(result.Error)
	}

	return map[string]string{
		"accessToken": result.AccessToken,
		"shop":        result.Shop,
		"scope":       result.Scope,
	}, nil
}

// CancelShopifyOAuth cancels the ongoing OAuth flow
func (a *App) CancelShopifyOAuth() {
	shopifyOAuthMutex.Lock()
	defer shopifyOAuthMutex.Unlock()

	if shopifyOAuthService != nil {
		shopifyOAuthService.StopCallbackServer()
		shopifyOAuthService = nil
		a.Log("[SHOPIFY-OAUTH] OAuth flow cancelled")
	}
}

// OpenShopifyOAuthInBrowser opens the Shopify OAuth URL in the default browser
func (a *App) OpenShopifyOAuthInBrowser(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// AddDataSource adds a new data source with generic configuration
func (a *App) AddDataSource(name string, driverType string, config map[string]string) (*agent.DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	dsConfig := agent.DataSourceConfig{
		OriginalFile:           config["filePath"],
		Host:                   config["host"],
		Port:                   config["port"],
		User:                   config["user"],
		Password:               config["password"],
		Database:               config["database"],
		StoreLocally:           config["storeLocally"] == "true",
		ShopifyStore:           config["shopifyStore"],
		ShopifyAccessToken:     config["shopifyAccessToken"],
		ShopifyAPIVersion:      config["shopifyAPIVersion"],
		BigCommerceStoreHash:   config["bigcommerceStoreHash"],
		BigCommerceAccessToken: config["bigcommerceAccessToken"],
		EbayAccessToken:        config["ebayAccessToken"],
		EbayEnvironment:        config["ebayEnvironment"],
		EbayApiFulfillment:     config["ebayApiFulfillment"] != "false",
		EbayApiFinances:        config["ebayApiFinances"] != "false",
		EbayApiAnalytics:       config["ebayApiAnalytics"] != "false",
		EtsyShopId:             config["etsyShopId"],
		EtsyAccessToken:        config["etsyAccessToken"],
		JiraInstanceType:       config["jiraInstanceType"],
		JiraBaseUrl:            config["jiraBaseUrl"],
		JiraUsername:           config["jiraUsername"],
		JiraApiToken:           config["jiraApiToken"],
		JiraProjectKey:         config["jiraProjectKey"],
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt, "", "") // Task 3.1: Added empty requestId for internal call
	}

	ds, err := a.dataSourceService.ImportDataSource(name, driverType, dsConfig, headerGen)
	if err == nil && ds != nil {
		go a.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// DeleteDataSource deletes a data source
func (a *App) DeleteDataSource(id string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.DeleteDataSource(id)
}

// RefreshEcommerceDataSource performs incremental update for e-commerce data sources
// Returns the refresh result with information about new data fetched
func (a *App) RefreshEcommerceDataSource(id string) (*agent.RefreshResult, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.RefreshEcommerceDataSource(id)
}

// IsEcommerceDataSource checks if a data source type supports incremental refresh
func (a *App) IsEcommerceDataSource(dsType string) bool {
	if a.dataSourceService == nil {
		return false
	}
	return a.dataSourceService.IsEcommerceDataSource(dsType)
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
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	projects, err := a.dataSourceService.GetJiraProjects(instanceType, baseUrl, username, apiToken)
	if err != nil {
		return nil, err
	}
	// Convert agent.JiraProject to JiraProject
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

// IsRefreshableDataSource checks if a data source type supports incremental refresh
// This includes both e-commerce platforms and project management tools like Jira
func (a *App) IsRefreshableDataSource(dsType string) bool {
	if a.dataSourceService == nil {
		return false
	}
	return a.dataSourceService.IsRefreshableDataSource(dsType)
}

// RefreshDataSource performs incremental update for supported data sources
// Works for both e-commerce platforms and Jira
func (a *App) RefreshDataSource(id string) (*agent.RefreshResult, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.RefreshDataSource(id)
}

// RenameDataSource renames a data source
func (a *App) RenameDataSource(id string, newName string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.RenameDataSource(id, newName)
}

// DeleteTable removes a table from a data source
func (a *App) DeleteTable(id string, tableName string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.DeleteTable(id, tableName)
}

// RenameColumn renames a column in a table
func (a *App) RenameColumn(id string, tableName string, oldColumnName string, newColumnName string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.RenameColumn(id, tableName, oldColumnName, newColumnName)
}

// DeleteColumn deletes a column from a table
func (a *App) DeleteColumn(id string, tableName string, columnName string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.DeleteColumn(id, tableName, columnName)
}

// UpdateMySQLExportConfig updates the MySQL export configuration for a data source
func (a *App) UpdateMySQLExportConfig(id string, host, port, user, password, database string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	config := agent.MySQLExportConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
	}
	return a.dataSourceService.UpdateMySQLExportConfig(id, config)
}

// GetDataSourceTables returns all table names for a data source
func (a *App) GetDataSourceTables(id string) ([]string, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.GetDataSourceTables(id)
}

// GetDataSourceTableData returns preview data for a table
func (a *App) GetDataSourceTableData(id string, tableName string) ([]map[string]interface{}, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	cfg, err := a.GetConfig()
	if err != nil {
		return nil, err
	}
	return a.dataSourceService.GetDataSourceTableData(id, tableName, cfg.MaxPreviewRows)
}

// GetDataSourceTableCount returns the total number of rows in a table
func (a *App) GetDataSourceTableCount(id string, tableName string) (int, error) {
	return a.dataSourceService.GetDataSourceTableCount(id, tableName)
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

		Title: "Save File",

		DefaultFilename: filename,

		Filters: []runtime.FileFilter{

			{DisplayName: "Files", Pattern: filterPattern},
		},
	})

}

// ExportToCSV exports one or more data source tables to CSV

func (a *App) ExportToCSV(id string, tableNames []string, outputPath string) error {

	if a.dataSourceService == nil {

		return fmt.Errorf("data source service not initialized")

	}

	return a.dataSourceService.ExportToCSV(id, tableNames, outputPath)

}

// ExportToJSON exports one or more data source tables to JSON

func (a *App) ExportToJSON(id string, tableNames []string, outputPath string) error {

	if a.dataSourceService == nil {

		return fmt.Errorf("data source service not initialized")

	}

	return a.dataSourceService.ExportToJSON(id, tableNames, outputPath)

}

// ExportToSQL exports one or more data source tables to SQL

func (a *App) ExportToSQL(id string, tableNames []string, outputPath string) error {

	if a.dataSourceService == nil {

		return fmt.Errorf("data source service not initialized")

	}

	return a.dataSourceService.ExportToSQL(id, tableNames, outputPath)

}

// ExportToMySQL exports one or more data source tables to MySQL

func (a *App) ExportToMySQL(id string, tableNames []string, host, port, user, password, database string) error {

	if a.dataSourceService == nil {

		return fmt.Errorf("data source service not initialized")

	}

	config := agent.DataSourceConfig{

		Host: host,

		Port: port,

		User: user,

		Password: password,

		Database: database,
	}

	return a.dataSourceService.ExportToMySQL(id, tableNames, config)

}

// TestMySQLConnection tests the connection to a MySQL server
func (a *App) TestMySQLConnection(host, port, user, password string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.TestMySQLConnection(host, port, user, password)
}

// GetMySQLDatabases returns a list of databases from the MySQL server
func (a *App) GetMySQLDatabases(host, port, user, password string) ([]string, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.GetMySQLDatabases(host, port, user, password)
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
	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}
	return a.chatService.GetSessionFiles(threadID)
}

// GetSessionFilePath returns the full path to a session file
func (a *App) GetSessionFilePath(threadID, fileName string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	filesDir := a.chatService.GetSessionFilesDirectory(threadID)
	filePath := filepath.Join(filesDir, fileName)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fileName)
	}

	return filePath, nil
}

// OpenSessionFile opens a session file in the default application
func (a *App) OpenSessionFile(threadID, fileName string) error {
	filePath, err := a.GetSessionFilePath(threadID, fileName)
	if err != nil {
		return err
	}

	runtime.BrowserOpenURL(a.ctx, "file://"+filePath)
	return nil
}

// OpenExternalURL opens a URL in the system's default browser
func (a *App) OpenExternalURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// DeleteSessionFile deletes a specific file from a session
func (a *App) DeleteSessionFile(threadID, fileName string) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	filePath, err := a.GetSessionFilePath(threadID, fileName)
	if err != nil {
		return err
	}

	// Delete the physical file
	if err := os.Remove(filePath); err != nil {
		return err
	}

	// Update the thread to remove the file from the list
	// Load the thread
	threads, err := a.chatService.LoadThreads()
	if err != nil {
		return err
	}

	for _, t := range threads {
		if t.ID == threadID {
			// Remove file from list
			var updatedFiles []SessionFile
			for _, f := range t.Files {
				if f.Name != fileName {
					updatedFiles = append(updatedFiles, f)
				}
			}
			t.Files = updatedFiles

			// Save the updated thread
			return a.chatService.SaveThreads([]ChatThread{t})
		}
	}

	return fmt.Errorf("thread not found")
}

// associateNewFilesWithMessage updates newly created files to associate them with a specific message
func (a *App) associateNewFilesWithMessage(threadID, messageID string, existingFiles map[string]bool) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	// Get current session files
	sessionFiles, err := a.chatService.GetSessionFiles(threadID)
	if err != nil {
		return err
	}

	// Find new files (not in existingFiles map) and update their MessageID
	updated := false
	for i := range sessionFiles {
		// Skip files that existed before this analysis
		if existingFiles[sessionFiles[i].Name] {
			continue
		}

		// Skip files that already have a MessageID
		if sessionFiles[i].MessageID != "" {
			continue
		}

		// Associate this new file with the message
		sessionFiles[i].MessageID = messageID
		updated = true
		a.Log(fmt.Sprintf("[SESSION] Associated file '%s' with message %s", sessionFiles[i].Name, messageID))
	}

	// Save updated thread if any files were modified
	if updated {
		// Load the thread
		threads, err := a.chatService.LoadThreads()
		if err != nil {
			return err
		}

		for _, t := range threads {
			if t.ID == threadID {
				t.Files = sessionFiles
				return a.chatService.SaveThreads([]ChatThread{t})
			}
		}
		return fmt.Errorf("thread not found")
	}

	return nil
}

// OpenSessionResultsDirectory opens the session's results directory in the file explorer
func (a *App) OpenSessionResultsDirectory(threadID string) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}

	// Get the session directory (where files are saved)
	sessionDir := a.chatService.GetSessionDirectory(threadID)

	// Check if directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return fmt.Errorf("session directory does not exist")
	}

	// Open the directory in the file explorer using platform-specific commands
	var cmd *exec.Cmd
	switch gort.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", sessionDir)
	case "darwin":
		cmd = exec.Command("open", sessionDir)
	case "linux":
		cmd = exec.Command("xdg-open", sessionDir)
	default:
		return fmt.Errorf("unsupported platform: %s", gort.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}

	return nil
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
	if a.einoService == nil {
		return nil, fmt.Errorf("eino service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return nil, fmt.Errorf("skill manager not available")
	}

	skills := skillManager.ListSkills()
	result := make([]SkillInfo, 0, len(skills))

	for _, skill := range skills {
		result = append(result, SkillInfo{
			ID:              skill.Manifest.ID,
			Name:            skill.Manifest.Name,
			Description:     skill.Manifest.Description,
			Version:         skill.Manifest.Version,
			Author:          skill.Manifest.Author,
			Category:        skill.Manifest.Category,
			Keywords:        skill.Manifest.Keywords,
			RequiredColumns: skill.Manifest.RequiredColumns,
			Tools:           skill.Manifest.Tools,
			Enabled:         skill.Manifest.Enabled,
			Icon:            skill.Manifest.Icon,
			Tags:            skill.Manifest.Tags,
		})
	}

	return result, nil
}

// GetEnabledSkills returns only enabled skills
func (a *App) GetEnabledSkills() ([]SkillInfo, error) {
	if a.einoService == nil {
		return nil, fmt.Errorf("eino service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return nil, fmt.Errorf("skill manager not available")
	}

	skills := skillManager.ListEnabledSkills()
	result := make([]SkillInfo, 0, len(skills))

	for _, skill := range skills {
		result = append(result, SkillInfo{
			ID:              skill.Manifest.ID,
			Name:            skill.Manifest.Name,
			Description:     skill.Manifest.Description,
			Version:         skill.Manifest.Version,
			Author:          skill.Manifest.Author,
			Category:        skill.Manifest.Category,
			Keywords:        skill.Manifest.Keywords,
			RequiredColumns: skill.Manifest.RequiredColumns,
			Tools:           skill.Manifest.Tools,
			Enabled:         skill.Manifest.Enabled,
			Icon:            skill.Manifest.Icon,
			Tags:            skill.Manifest.Tags,
		})
	}

	return result, nil
}

// GetSkillCategories returns all skill categories
func (a *App) GetSkillCategories() ([]string, error) {
	if a.einoService == nil {
		return nil, fmt.Errorf("eino service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return nil, fmt.Errorf("skill manager not available")
	}

	return skillManager.GetCategories(), nil
}

// EnableSkill enables a skill by ID
func (a *App) EnableSkill(skillID string) error {
	// Check if analysis is in progress
	a.activeThreadsMutex.RLock()
	hasActiveAnalysis := len(a.activeThreads) > 0
	a.activeThreadsMutex.RUnlock()

	if hasActiveAnalysis {
		return fmt.Errorf("cannot enable skill while analysis is in progress")
	}

	// Use skillService if available (for new skill management)
	if a.skillService != nil {
		if err := a.skillService.EnableSkill(skillID); err != nil {
			return err
		}
		// Reload skills in agent after enabling
		return a.ReloadSkills()
	}

	// Fallback to einoService for backward compatibility
	if a.einoService == nil {
		return fmt.Errorf("skill service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return fmt.Errorf("skill manager not available")
	}

	return skillManager.EnableSkill(skillID)
}

// DisableSkill disables a skill by ID
func (a *App) DisableSkill(skillID string) error {
	// Check if analysis is in progress
	a.activeThreadsMutex.RLock()
	hasActiveAnalysis := len(a.activeThreads) > 0
	a.activeThreadsMutex.RUnlock()

	if hasActiveAnalysis {
		return fmt.Errorf("cannot disable skill while analysis is in progress")
	}

	// Use skillService if available (for new skill management)
	if a.skillService != nil {
		if err := a.skillService.DisableSkill(skillID); err != nil {
			return err
		}
		// Reload skills in agent after disabling
		return a.ReloadSkills()
	}

	// Fallback to einoService for backward compatibility
	if a.einoService == nil {
		return fmt.Errorf("skill service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return fmt.Errorf("skill manager not available")
	}

	return skillManager.DisableSkill(skillID)
}

// DeleteSkill deletes a skill by ID (removes directory and config)
func (a *App) DeleteSkill(skillID string) error {
	// Check if analysis is in progress
	a.activeThreadsMutex.RLock()
	isGenerating := len(a.activeThreads) > 0
	a.activeThreadsMutex.RUnlock()

	if isGenerating {
		return fmt.Errorf("cannot delete skill while analysis is in progress")
	}

	// Use skillService if available (for new skill management)
	if a.skillService != nil {
		if err := a.skillService.DeleteSkill(skillID); err != nil {
			return err
		}
		// Try to reload skills in agent after deleting, but don't fail if it errors
		if err := a.ReloadSkills(); err != nil {
			a.Log(fmt.Sprintf("[SKILLS] Warning: Failed to reload skills after deletion: %v", err))
		}
		return nil
	}

	return fmt.Errorf("skill service not initialized")
}

// ReloadSkills reloads all skills from disk
func (a *App) ReloadSkills() error {
	if a.einoService == nil {
		return fmt.Errorf("eino service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return fmt.Errorf("skill manager not available")
	}

	return skillManager.ReloadSkills()
}

// --- Metrics JSON Management ---

// SaveMetricsJson saves metrics JSON data for a specific message
func (a *App) SaveMetricsJson(messageId string, metricsJson string) error {
	// Get storage directory
	storageDir, err := a.getStorageDir()
	if err != nil {
		return fmt.Errorf("failed to get storage directory: %w", err)
	}

	// Create metrics directory path
	metricsDir := filepath.Join(storageDir, "data", "metrics")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		return fmt.Errorf("failed to create metrics directory: %w", err)
	}

	// Create file path
	filePath := filepath.Join(metricsDir, fmt.Sprintf("%s.json", messageId))

	// Write JSON file
	if err := os.WriteFile(filePath, []byte(metricsJson), 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	a.Log(fmt.Sprintf("Metrics JSON saved for message %s: %s", messageId, filePath))
	return nil
}

// LoadMetricsJson loads metrics JSON data for a specific message
func (a *App) LoadMetricsJson(messageId string) (string, error) {
	// Get storage directory
	storageDir, err := a.getStorageDir()
	if err != nil {
		return "", fmt.Errorf("failed to get storage directory: %w", err)
	}

	// Build file path
	filePath := filepath.Join(storageDir, "data", "metrics", fmt.Sprintf("%s.json", messageId))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("metrics file not found for message: %s", messageId)
	}

	// Read JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read metrics file: %w", err)
	}

	a.Log(fmt.Sprintf("Metrics JSON loaded for message %s: %s", messageId, filePath))
	return string(data), nil
}

// ExtractMetricsFromAnalysis automatically extracts key metrics from analysis results
func (a *App) ExtractMetricsFromAnalysis(threadID string, messageId string, analysisContent string) error {
	cfg, err := a.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Build metrics extraction prompt
	var prompt string
	if cfg.Language == "简体中文" {
		prompt = fmt.Sprintf(`请从以下分析结果中提取最重要的数值型关键指标，以JSON格式返回。

要求：
1. 只返回JSON数组，不要其他文字说明
2. 每个指标必须包含：name（指标名称）、value（数值）、unit（单位，可选）
3. **重要**：只提取数值型指标，value必须是数字或包含数字的字符串
4. **重要**：如果分析结果中没有明确的数值型指标，返回空数组 []
5. 最多提取6个最重要的业务指标
6. 优先提取：总量、增长率、平均值、比率、金额、数量等核心业务指标
7. 数值要准确，来源于分析内容
8. 单位要合适（如：个、%%、元、$、次/年、天等）
9. 指标名称要简洁明了
10. 不要提取非数值型的描述性内容

示例格式（有数值指标时）：
[
  {"name":"总销售额","value":"1,234,567","unit":"元"},
  {"name":"增长率","value":"+15.5","unit":"%%"},
  {"name":"平均订单价值","value":"89.50","unit":"元"}
]

示例格式（无数值指标时）：
[]

分析内容：
%s

请返回JSON：`, analysisContent)
	} else {
		prompt = fmt.Sprintf(`Please extract the most important numerical key metrics from the following analysis results in JSON format.

Requirements:
1. Return only JSON array, no other text
2. Each metric must include: name, value, unit (optional)
3. **Important**: Only extract numerical metrics, value must be a number or string containing numbers
4. **Important**: If there are no clear numerical metrics in the analysis, return empty array []
5. Extract at most 6 most important business metrics
6. Prioritize: totals, growth rates, averages, ratios, amounts, quantities and other core business metrics
7. Values must be accurate from the analysis content
8. Use appropriate units (e.g., items, %%, $, times/year, days, etc.)
9. Metric names should be concise and clear
10. Do not extract non-numerical descriptive content

Example format (with numerical metrics):
[
  {"name":"Total Sales","value":"1,234,567","unit":"$"},
  {"name":"Growth Rate","value":"+15.5","unit":"%%"},
  {"name":"Average Order Value","value":"89.50","unit":"$"}
]

Example format (without numerical metrics):
[]

Analysis content:
%s

Please return JSON:`, analysisContent)
	}

	// Try extraction up to 3 times
	for attempt := 1; attempt <= 3; attempt++ {
		err := a.tryExtractMetrics(threadID, messageId, prompt, attempt)
		if err == nil {
			return nil
		}

		a.Log(fmt.Sprintf("Metrics extraction attempt %d failed: %v", attempt, err))

		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second) // Incremental delay
		}
	}

	// If all attempts fail, use fallback text extraction
	return a.fallbackTextExtraction(messageId, analysisContent)
}

// tryExtractMetrics attempts to extract metrics using LLM
func (a *App) tryExtractMetrics(threadID string, messageId string, prompt string, attempt int) error {
	// Call LLM to extract metrics
	llm := agent.NewLLMService(a.getConfigForExtraction(), a.Log)
	response, err := llm.Chat(a.ctx, prompt)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Clean response and extract JSON part
	jsonStr := a.extractJSONFromResponse(response)
	if jsonStr == "" {
		return fmt.Errorf("no valid JSON found in LLM response")
	}

	// Validate JSON format
	var metrics []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &metrics); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Allow empty array - no numerical metrics found
	if len(metrics) == 0 {
		a.Log("No numerical metrics found in analysis, skipping metrics extraction")
		return nil // Not an error, just no metrics to display
	}

	// Validate each metric has required fields and contains numerical value
	validMetrics := []map[string]interface{}{}
	for i, metric := range metrics {
		name, hasName := metric["name"]
		value, hasValue := metric["value"]

		if !hasName {
			a.Log(fmt.Sprintf("Metric %d missing 'name' field, skipping", i))
			continue
		}
		if !hasValue {
			a.Log(fmt.Sprintf("Metric %d missing 'value' field, skipping", i))
			continue
		}

		// Validate that value contains numbers
		valueStr := fmt.Sprintf("%v", value)
		if !containsNumber(valueStr) {
			a.Log(fmt.Sprintf("Metric %d (%s) value '%s' does not contain numbers, skipping", i, name, valueStr))
			continue
		}

		validMetrics = append(validMetrics, metric)
	}

	// If no valid metrics after filtering, don't save or display
	if len(validMetrics) == 0 {
		a.Log("No valid numerical metrics after validation, skipping metrics extraction")
		return nil
	}

	// Re-marshal valid metrics
	validMetricsJSON, err := json.Marshal(validMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal valid metrics: %w", err)
	}
	jsonStr = string(validMetricsJSON)

	// Save metrics JSON
	if err := a.SaveMetricsJson(messageId, jsonStr); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	// Mark the user message with chart_data so frontend knows it has data
	if threadID != "" {
		a.attachChartToUserMessage(threadID, messageId, &ChartData{
			Charts: []ChartItem{{Type: "metrics", Data: ""}},
		})
	}

	// Use event aggregator for new unified event system
	if a.eventAggregator != nil {
		// Get threadID from context if available
		threadID := "" // Will be set by caller context
		for _, metric := range validMetrics {
			m := Metric{
				Title:  fmt.Sprintf("%v", metric["name"]),
				Value:  fmt.Sprintf("%v", metric["value"]),
				Change: "",
			}
			if unit, ok := metric["unit"]; ok {
				m.Value = fmt.Sprintf("%v%v", metric["value"], unit)
			}
			if change, ok := metric["change"]; ok {
				m.Change = fmt.Sprintf("%v", change)
			}
			a.eventAggregator.AddMetric(threadID, messageId, "", m)
		}
		a.eventAggregator.FlushNow(threadID, false)
	}

	a.Log(fmt.Sprintf("Metrics extracted and saved for message %s (attempt %d)", messageId, attempt))
	return nil
}

// getConfigForExtraction gets config for metrics extraction
func (a *App) getConfigForExtraction() config.Config {
	cfg, _ := a.GetEffectiveConfig()
	// Return config as-is since Temperature field doesn't exist
	return cfg
}

// ExtractSuggestionsFromAnalysis extracts next-step suggestions from analysis response
// and emits them to the dashboard insights area
func (a *App) ExtractSuggestionsFromAnalysis(threadID, userMessageID, analysisContent string) error {
	if analysisContent == "" {
		return nil
	}

	// Look for patterns that indicate next steps or suggestions in the analysis
	// Common patterns: numbered lists, "建议", "next steps", "you can", "可以", etc.
	var insights []Insight
	lines := strings.Split(analysisContent, "\n")

	// Patterns for next-step suggestions
	numberPattern := regexp.MustCompile(`^\s*(\d+)[.、]\s+(.+)$`)
	suggestionPattern := regexp.MustCompile(`(?i)(建议|suggest|recommend|next|further|深入|可以进一步)`)

	foundSuggestionSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering a suggestion/next-steps section
		if suggestionPattern.MatchString(trimmedLine) {
			foundSuggestionSection = true
		}

		// Extract numbered items (likely suggestions) - prefer items after suggestion markers
		if matches := numberPattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			suggestionText := strings.TrimSpace(matches[2])
			if suggestionText != "" && len(suggestionText) > 10 { // Filter out very short items
				// Prioritize items found in/after suggestion sections
				_ = foundSuggestionSection // Use variable to avoid compiler error
				insights = append(insights, Insight{
					Text:         suggestionText,
					Icon:         "lightbulb",
					DataSourceID: "",
					SourceName:   "",
				})
			}
		}
	}

	// Limit to 9 suggestions (auto insights)
	if len(insights) > 9 {
		insights = insights[:9]
	}

	if len(insights) > 0 {
		a.Log(fmt.Sprintf("[SUGGESTIONS] Extracted %d suggestions from analysis for message %s", len(insights), userMessageID))

		// Use event aggregator for new unified event system
		if a.eventAggregator != nil {
			for _, insight := range insights {
				a.eventAggregator.AddInsight(threadID, userMessageID, "", insight)
			}
			a.eventAggregator.FlushNow(threadID, false)
		}
	}

	return nil
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
	// Try to extract JSON array
	jsonPattern := regexp.MustCompile(`\[[\s\S]*?\]`)
	matches := jsonPattern.FindAllString(response, -1)

	for _, match := range matches {
		// Validate if it's valid JSON
		var test []interface{}
		if json.Unmarshal([]byte(match), &test) == nil {
			return match
		}
	}

	return ""
}

// fallbackTextExtraction uses regex patterns as fallback when LLM extraction fails
func (a *App) fallbackTextExtraction(messageId string, content string) error {
	metrics := []map[string]interface{}{}

	// Extract common metric patterns
	patterns := []struct {
		regex *regexp.Regexp
		name  string
		unit  string
	}{
		{regexp.MustCompile(`总.*?[：:]?\s*(\d+(?:,\d{3})*(?:\.\d+)?)`), "总计", ""},
		{regexp.MustCompile(`(\d+(?:\.\d+)?)%`), "百分比", "%"},
		{regexp.MustCompile(`\$(\d+(?:,\d{3})*(?:\.\d+)?)`), "金额", "$"},
		{regexp.MustCompile(`平均.*?[：:]?\s*(\d+(?:\.\d+)?)`), "平均值", ""},
		{regexp.MustCompile(`增长.*?[：:]?\s*([+\-]?\d+(?:\.\d+)?)%`), "增长率", "%"},
	}

	for _, pattern := range patterns {
		matches := pattern.regex.FindAllStringSubmatch(content, -1)
		for i, match := range matches {
			if len(match) > 1 && len(metrics) < 6 {
				metrics = append(metrics, map[string]interface{}{
					"name":  fmt.Sprintf("%s%d", pattern.name, i+1),
					"value": match[1],
					"unit":  pattern.unit,
				})
			}
		}
	}

	if len(metrics) > 0 {
		jsonStr, _ := json.Marshal(metrics)
		err := a.SaveMetricsJson(messageId, string(jsonStr))
		if err == nil {
			// Use event aggregator for new unified event system
			if a.eventAggregator != nil {
				for _, metric := range metrics {
					m := Metric{
						Title:  fmt.Sprintf("%v", metric["name"]),
						Value:  fmt.Sprintf("%v", metric["value"]),
						Change: "",
					}
					if unit, ok := metric["unit"]; ok {
						m.Value = fmt.Sprintf("%v%v", metric["value"], unit)
					}
					a.eventAggregator.AddMetric("", messageId, "", m)
				}
				a.eventAggregator.FlushNow("", false)
			}
			a.Log(fmt.Sprintf("Fallback metrics extracted for message %s", messageId))
		}
		return err
	}

	return fmt.Errorf("no metrics could be extracted using fallback method")
}

// SaveSessionRecording saves the current session's analysis recording to a file
func (a *App) SaveSessionRecording(threadID, title, description string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	// Get thread
	threads, err := a.chatService.LoadThreads()
	if err != nil {
		return "", fmt.Errorf("failed to get threads: %w", err)
	}

	var thread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			thread = &threads[i]
			break
		}
	}

	if thread == nil {
		return "", fmt.Errorf("thread not found: %s", threadID)
	}

	// Extract data source schema
	var schemas []agent.ReplayTableSchema
	if thread.DataSourceID != "" {
		tables, err := a.dataSourceService.GetDataSourceTables(thread.DataSourceID)
		if err == nil {
			for _, tableName := range tables {
				data, err := a.dataSourceService.GetDataSourceTableData(thread.DataSourceID, tableName, 1)
				if err != nil {
					continue
				}
				var cols []string
				if len(data) > 0 {
					for k := range data[0] {
						cols = append(cols, k)
					}
				}
				schemas = append(schemas, agent.ReplayTableSchema{
					TableName: tableName,
					Columns:   cols,
				})
			}
		}
	}

	// Get data source name
	var sourceName string
	if thread.DataSourceID != "" {
		sources, err := a.dataSourceService.LoadDataSources()
		if err == nil {
			for _, ds := range sources {
				if ds.ID == thread.DataSourceID {
					sourceName = ds.Name
					break
				}
			}
		}
	}

	// Create recorder
	recorder := agent.NewAnalysisRecorder(thread.DataSourceID, sourceName, schemas)
	recorder.SetMetadata(title, description)

	// Parse messages to extract tool calls
	// We need to extract SQL and Python tool executions from the conversation
	// This is a simplified version - in a real implementation, we would track these during execution
	stepID := 0
	for _, msg := range thread.Messages {
		if msg.Role != "assistant" {
			continue
		}

		// Record conversation
		recorder.RecordConversation("assistant", msg.Content)

		// Try to extract SQL queries from message content
		if strings.Contains(msg.Content, "```sql") {
			startSQL := strings.Index(msg.Content, "```sql")
			endSQL := strings.Index(msg.Content[startSQL+6:], "```")
			if endSQL > 0 {
				sqlQuery := strings.TrimSpace(msg.Content[startSQL+6 : startSQL+6+endSQL])
				stepID++
				recorder.RecordStep("execute_sql", fmt.Sprintf("SQL Query Step %d", stepID), sqlQuery, "", "", "")
			}
		}

		// Try to extract Python code from message content
		if strings.Contains(msg.Content, "```python") {
			startPy := strings.Index(msg.Content, "```python")
			endPy := strings.Index(msg.Content[startPy+9:], "```")
			if endPy > 0 {
				pythonCode := strings.TrimSpace(msg.Content[startPy+9 : startPy+9+endPy])
				stepID++
				recorder.RecordStep("python_executor", fmt.Sprintf("Python Analysis Step %d", stepID), pythonCode, "", "", "")
			}
		}
	}

	// Save recording
	recordingDir := filepath.Join(a.storageDir, "recordings")
	filePath, err := recorder.SaveRecording(recordingDir)
	if err != nil {
		return "", fmt.Errorf("failed to save recording: %w", err)
	}

	a.Log(fmt.Sprintf("Session recording saved: %s", filePath))
	return filePath, nil
}

// GetSessionRecordings returns all available session recordings
func (a *App) GetSessionRecordings() ([]agent.AnalysisRecording, error) {
	recordingDir := filepath.Join(a.storageDir, "recordings")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recordings directory: %w", err)
	}

	// List all recording files
	files, err := os.ReadDir(recordingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read recordings directory: %w", err)
	}

	recordings := []agent.AnalysisRecording{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(recordingDir, file.Name())
		recording, err := agent.LoadRecording(filePath)
		if err != nil {
			a.Log(fmt.Sprintf("Failed to load recording %s: %v", file.Name(), err))
			continue
		}

		recordings = append(recordings, *recording)
	}

	return recordings, nil
}

// ReplayAnalysisRecording replays a recorded analysis on a target data source
func (a *App) ReplayAnalysisRecording(recordingID, targetSourceID string, autoFixFields bool, maxFieldDiff int) (*agent.ReplayResult, error) {
	if a.einoService == nil {
		return nil, fmt.Errorf("eino service not initialized")
	}

	// Load recording
	recordingDir := filepath.Join(a.storageDir, "recordings")
	recordingPath := filepath.Join(recordingDir, fmt.Sprintf("recording_%s.json", recordingID))

	recording, err := agent.LoadRecording(recordingPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load recording: %w", err)
	}

	// Get target data source name
	var targetSourceName string
	sources, err := a.dataSourceService.LoadDataSources()
	if err == nil {
		for _, ds := range sources {
			if ds.ID == targetSourceID {
				targetSourceName = ds.Name
				break
			}
		}
	}

	// Create replay config
	config := &agent.ReplayConfig{
		RecordingID:      recordingID,
		TargetSourceID:   targetSourceID,
		TargetSourceName: targetSourceName,
		AutoFixFields:    autoFixFields,
		MaxFieldDiff:     maxFieldDiff,
		TableMappings:    []agent.TableMapping{},
	}

	// Create SQL and Python tools
	sqlTool := agent.NewSQLExecutorTool(a.dataSourceService)

	cfg, _ := a.GetEffectiveConfig()
	pythonTool := agent.NewPythonExecutorTool(cfg)

	// Create LLM service for intelligent field matching
	llmService := agent.NewLLMService(cfg, a.Log)

	// Create replayer
	replayer := agent.NewAnalysisReplayer(
		recording,
		config,
		a.dataSourceService,
		sqlTool,
		pythonTool,
		llmService,
		a.Log,
	)

	// Execute replay
	result, err := replayer.Replay()
	if err != nil {
		return nil, fmt.Errorf("replay failed: %w", err)
	}

	return result, nil
}

// --- Dashboard Drag-Drop Layout Wails Bridge Methods ---

// SaveLayout saves a layout configuration to the database (Task 5.1)
func (a *App) SaveLayout(config database.LayoutConfiguration) error {
	if a.layoutService == nil {
		return fmt.Errorf("layout service not initialized")
	}

	a.Log(fmt.Sprintf("[LAYOUT] Saving layout configuration for user: %s", config.UserID))
	err := a.layoutService.SaveLayout(config)
	if err != nil {
		a.Log(fmt.Sprintf("[LAYOUT] Failed to save layout: %v", err))
		return err
	}

	a.Log("[LAYOUT] Layout configuration saved successfully")
	return nil
}

// LoadLayout loads a layout configuration from the database (Task 5.2)
func (a *App) LoadLayout(userID string) (*database.LayoutConfiguration, error) {
	if a.layoutService == nil {
		return nil, fmt.Errorf("layout service not initialized")
	}

	a.Log(fmt.Sprintf("[LAYOUT] Loading layout configuration for user: %s", userID))
	config, err := a.layoutService.LoadLayout(userID)
	if err != nil {
		// If no layout found, return default layout instead of error
		if err.Error() == fmt.Sprintf("no layout found for user: %s", userID) {
			a.Log("[LAYOUT] No saved layout found, returning default layout")
			defaultConfig := a.layoutService.GetDefaultLayout()
			defaultConfig.UserID = userID
			return &defaultConfig, nil
		}

		a.Log(fmt.Sprintf("[LAYOUT] Failed to load layout: %v", err))
		return nil, err
	}

	a.Log("[LAYOUT] Layout configuration loaded successfully")
	return config, nil
}

// CheckComponentHasData checks if a component has data available (Task 5.3)
func (a *App) CheckComponentHasData(componentType string, instanceID string) (bool, error) {
	if a.dataService == nil {
		return false, fmt.Errorf("data service not initialized")
	}

	a.Log(fmt.Sprintf("[DATA] Checking data availability for component: %s (%s)", instanceID, componentType))
	hasData, err := a.dataService.CheckComponentHasData(componentType, instanceID)
	if err != nil {
		a.Log(fmt.Sprintf("[DATA] Failed to check component data: %v", err))
		return false, err
	}

	a.Log(fmt.Sprintf("[DATA] Component %s has data: %v", instanceID, hasData))
	return hasData, nil
}

// GetFilesByCategory retrieves files for a specific category (Task 5.4)
func (a *App) GetFilesByCategory(category string) ([]database.FileInfo, error) {
	if a.fileService == nil {
		return nil, fmt.Errorf("file service not initialized")
	}

	// Convert string to FileCategory type
	var fileCategory database.FileCategory
	switch category {
	case "all_files":
		fileCategory = database.AllFiles
	case "user_request_related":
		fileCategory = database.UserRequestRelated
	default:
		return nil, fmt.Errorf("invalid file category: %s", category)
	}

	a.Log(fmt.Sprintf("[FILES] Getting files for category: %s", category))
	files, err := a.fileService.GetFilesByCategory(fileCategory)
	if err != nil {
		a.Log(fmt.Sprintf("[FILES] Failed to get files: %v", err))
		return nil, err
	}

	a.Log(fmt.Sprintf("[FILES] Retrieved %d files for category %s", len(files), category))
	return files, nil
}

// DownloadFile returns the file path for download (Task 5.5)
func (a *App) DownloadFile(fileID string) (string, error) {
	if a.fileService == nil {
		return "", fmt.Errorf("file service not initialized")
	}

	a.Log(fmt.Sprintf("[FILES] Downloading file: %s", fileID))
	filePath, err := a.fileService.DownloadFile(fileID)
	if err != nil {
		a.Log(fmt.Sprintf("[FILES] Failed to download file: %v", err))
		return "", err
	}

	a.Log(fmt.Sprintf("[FILES] File download path: %s", filePath))
	return filePath, nil
}

// ExportDashboard exports dashboard data with component filtering (Task 5.6)
func (a *App) ExportDashboard(req database.ExportRequest) (*database.ExportResult, error) {
	if a.exportService == nil {
		return nil, fmt.Errorf("export service not initialized")
	}

	a.Log(fmt.Sprintf("[EXPORT] Exporting dashboard for user: %s, format: %s", req.UserID, req.Format))
	result, err := a.exportService.ExportDashboard(req)
	if err != nil {
		a.Log(fmt.Sprintf("[EXPORT] Failed to export dashboard: %v", err))
		return nil, err
	}

	a.Log(fmt.Sprintf("[EXPORT] Dashboard exported successfully: %s", result.FilePath))
	a.Log(fmt.Sprintf("[EXPORT] Included components: %d, Excluded components: %d",
		len(result.IncludedComponents), len(result.ExcludedComponents)))
	return result, nil
}

// ListSkills returns all installed skills
func (a *App) ListSkills() ([]agent.Skill, error) {
	if a.skillService == nil {
		return nil, fmt.Errorf("skill service not initialized")
	}
	return a.skillService.ListSkills()
}

// InstallSkillsFromZip installs skills from a ZIP file
// Opens a file dialog for the user to select a ZIP file
func (a *App) InstallSkillsFromZip() ([]string, error) {
	if a.skillService == nil {
		return nil, fmt.Errorf("skill service not initialized")
	}

	// Open file dialog to select ZIP file
	zipPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Skills ZIP Package",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "ZIP Files (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open file dialog: %v", err)
	}

	if zipPath == "" {
		return nil, fmt.Errorf("no file selected")
	}

	a.Log(fmt.Sprintf("[SKILLS] Installing from: %s", zipPath))

	// Install skills from ZIP
	installed, err := a.skillService.InstallFromZip(zipPath)
	if err != nil {
		a.Log(fmt.Sprintf("[SKILLS] Installation failed: %v", err))
		return nil, err
	}

	a.Log(fmt.Sprintf("[SKILLS] Successfully installed: %v", installed))
	return installed, nil
}

// detectAnalysisType detects the type of analysis from the response
// Used for recording analysis history (Requirement 1.1)
func (a *App) detectAnalysisType(response string) string {
	responseLower := strings.ToLower(response)

	// Check for trend analysis keywords
	if strings.Contains(responseLower, "trend") || strings.Contains(responseLower, "趋势") ||
		strings.Contains(responseLower, "over time") || strings.Contains(responseLower, "随时间") {
		return "trend"
	}

	// Check for comparison analysis keywords
	if strings.Contains(responseLower, "comparison") || strings.Contains(responseLower, "对比") ||
		strings.Contains(responseLower, "compare") || strings.Contains(responseLower, "比较") {
		return "comparison"
	}

	// Check for distribution analysis keywords
	if strings.Contains(responseLower, "distribution") || strings.Contains(responseLower, "分布") ||
		strings.Contains(responseLower, "breakdown") || strings.Contains(responseLower, "构成") {
		return "distribution"
	}

	// Check for correlation analysis keywords
	if strings.Contains(responseLower, "correlation") || strings.Contains(responseLower, "相关") ||
		strings.Contains(responseLower, "relationship") || strings.Contains(responseLower, "关系") {
		return "correlation"
	}

	// Check for aggregation analysis keywords
	if strings.Contains(responseLower, "total") || strings.Contains(responseLower, "sum") ||
		strings.Contains(responseLower, "average") || strings.Contains(responseLower, "汇总") ||
		strings.Contains(responseLower, "平均") {
		return "aggregation"
	}

	// Check for ranking analysis keywords
	if strings.Contains(responseLower, "ranking") || strings.Contains(responseLower, "排名") ||
		strings.Contains(responseLower, "top") || strings.Contains(responseLower, "前") {
		return "ranking"
	}

	// Check for time series analysis keywords
	if strings.Contains(responseLower, "time series") || strings.Contains(responseLower, "时间序列") ||
		strings.Contains(responseLower, "forecast") || strings.Contains(responseLower, "预测") {
		return "time_series"
	}

	// Check for geographic analysis keywords
	if strings.Contains(responseLower, "geographic") || strings.Contains(responseLower, "地理") ||
		strings.Contains(responseLower, "region") || strings.Contains(responseLower, "区域") ||
		strings.Contains(responseLower, "province") || strings.Contains(responseLower, "省份") {
		return "geographic"
	}

	// Default to statistical analysis
	return "statistical"
}

// extractKeyFindings extracts key findings from the analysis response
// Used for recording analysis history (Requirement 1.1)
func (a *App) extractKeyFindings(response string) string {
	// Look for key findings section
	findingsKeywords := []string{
		"关键发现", "主要发现", "结论", "总结",
		"Key Findings", "Key findings", "Conclusion", "Summary",
		"发现", "结果", "insights", "Insights",
	}

	for _, keyword := range findingsKeywords {
		idx := strings.Index(response, keyword)
		if idx != -1 {
			// Extract up to 200 characters after the keyword
			start := idx
			end := start + 200
			if end > len(response) {
				end = len(response)
			}

			// Find the end of the sentence or paragraph
			excerpt := response[start:end]

			// Clean up the excerpt
			excerpt = strings.TrimSpace(excerpt)
			if len(excerpt) > 150 {
				// Truncate at the last complete sentence
				lastPeriod := strings.LastIndex(excerpt[:150], "。")
				if lastPeriod == -1 {
					lastPeriod = strings.LastIndex(excerpt[:150], ".")
				}
				if lastPeriod > 50 {
					excerpt = excerpt[:lastPeriod+1]
				} else {
					excerpt = excerpt[:150] + "..."
				}
			}

			return excerpt
		}
	}

	// If no key findings section found, extract the first meaningful sentence
	if len(response) > 150 {
		excerpt := response[:150]
		lastPeriod := strings.LastIndex(excerpt, "。")
		if lastPeriod == -1 {
			lastPeriod = strings.LastIndex(excerpt, ".")
		}
		if lastPeriod > 30 {
			return excerpt[:lastPeriod+1]
		}
		return excerpt + "..."
	}

	return response
}

// extractTargetColumns extracts target columns mentioned in the analysis
// Used for recording analysis history (Requirement 1.1)
func (a *App) extractTargetColumns(response string, availableColumns []string) []string {
	responseLower := strings.ToLower(response)
	targetColumns := []string{}

	for _, col := range availableColumns {
		colLower := strings.ToLower(col)
		if strings.Contains(responseLower, colLower) {
			targetColumns = append(targetColumns, col)
		}
	}

	// Limit to top 5 columns
	if len(targetColumns) > 5 {
		targetColumns = targetColumns[:5]
	}

	return targetColumns
}

// recordAnalysisHistory records analysis completion for intent enhancement
// Used for recording analysis history (Requirement 1.1)
func (a *App) recordAnalysisHistory(dataSourceID string, record agent.AnalysisRecord) {
	if a.intentEnhancementService == nil {
		return
	}

	// Get the context enhancer from the service and add the record
	// Note: We need to access the context enhancer through a public method
	// For now, we'll add a method to IntentEnhancementService to handle this
	a.AddAnalysisRecord(dataSourceID, record)
}

// AddAnalysisRecord adds an analysis record for intent enhancement
// This is a wrapper that delegates to the IntentEnhancementService
// Validates: Requirement 1.1
func (a *App) AddAnalysisRecord(dataSourceID string, record agent.AnalysisRecord) error {
	if a.intentEnhancementService == nil {
		return fmt.Errorf("intent enhancement service not initialized")
	}

	// Ensure the data source ID is set in the record
	if record.DataSourceID == "" {
		record.DataSourceID = dataSourceID
	}

	// Delegate to the IntentEnhancementService to add the record
	err := a.intentEnhancementService.AddAnalysisRecord(record)
	if err != nil {
		a.Log(fmt.Sprintf("[INTENT-HISTORY] Failed to record analysis: %v", err))
		return err
	}

	a.Log(fmt.Sprintf("[INTENT-HISTORY] Successfully recorded analysis: type=%s, columns=%v, findings=%s",
		record.AnalysisType, record.TargetColumns, record.KeyFindings))

	return nil
}

// RecordIntentSelection records user's intent selection for preference learning
// This is called from the frontend when a user selects an intent
// Validates: Requirement 2.1, 5.1
func (a *App) RecordIntentSelection(threadID string, intent IntentSuggestion) error {
	// Get data source ID from thread
	var dataSourceID string
	if threadID != "" && a.chatService != nil {
		threads, _ := a.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				dataSourceID = t.DataSourceID
				break
			}
		}
	}

	if dataSourceID == "" {
		return fmt.Errorf("no data source associated with thread")
	}

	// Convert to agent.IntentSuggestion
	agentIntent := agent.IntentSuggestion{
		ID:          intent.ID,
		Title:       intent.Title,
		Description: intent.Description,
		Icon:        intent.Icon,
		Query:       intent.Query,
	}

	// Record the selection using the new IntentUnderstandingService if available
	// Validates: Requirement 5.1 - Record user intent selection for preference learning
	if a.intentUnderstandingService != nil {
		if err := a.intentUnderstandingService.RecordSelection(dataSourceID, agentIntent); err != nil {
			a.Log(fmt.Sprintf("[INTENT] Failed to record selection in IntentUnderstandingService: %v", err))
		}
	}

	// Also record in the legacy IntentEnhancementService for backward compatibility
	if a.intentEnhancementService != nil {
		a.intentEnhancementService.RecordSelection(dataSourceID, agentIntent)
	}

	a.Log(fmt.Sprintf("[INTENT] Recorded intent selection: %s for data source: %s", intent.Title, dataSourceID))

	return nil
}

// GetMessageAnalysisData retrieves analysis data for a specific message (for dashboard restoration)
func (a *App) GetMessageAnalysisData(threadID, messageID string) (map[string]interface{}, error) {
	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}
	return a.chatService.GetMessageAnalysisData(threadID, messageID)
}

// SaveMessageAnalysisResults saves analysis results for a specific message
func (a *App) SaveMessageAnalysisResults(threadID, messageID string, results []AnalysisResultItem) error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}
	return a.chatService.SaveAnalysisResults(threadID, messageID, results)
}

// ============ License Activation Methods ============

// ActivationResult represents the result of license activation
type ActivationResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// ActivateLicense activates the application with a license server
func (a *App) ActivateLicense(serverURL, sn string) (*ActivationResult, error) {
	if a.licenseClient == nil {
		a.licenseClient = agent.NewLicenseClient(a.Log)
	}

	data, err := a.licenseClient.Activate(serverURL, sn)
	if err != nil {
		return &ActivationResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Save encrypted activation data to local storage
	if err := a.licenseClient.SaveActivationData(); err != nil {
		a.Log(fmt.Sprintf("[LICENSE] Warning: Failed to save activation data: %v", err))
	}

	// Reinitialize services with the new license configuration
	cfg, _ := a.GetConfig()
	a.reinitializeServices(cfg)

	return &ActivationResult{
		Success:   true,
		Message:   "激活成功",
		ExpiresAt: data.ExpiresAt,
	}, nil
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
	if a.licenseClient == nil {
		a.licenseClient = agent.NewLicenseClient(a.Log)
	}

	result, err := a.licenseClient.RequestSN(serverURL, email)
	if err != nil {
		return &RequestSNResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &RequestSNResult{
		Success: result.Success,
		Message: result.Message,
		SN:      result.SN,
		Code:    result.Code,
	}, nil
}

// GetActivationStatus returns the current activation status
func (a *App) GetActivationStatus() map[string]interface{} {
	if a.licenseClient == nil || !a.licenseClient.IsActivated() {
		return map[string]interface{}{
			"activated": false,
		}
	}

	data := a.licenseClient.GetData()
	count, limit, date := a.licenseClient.GetAnalysisStatus()
	
	return map[string]interface{}{
		"activated":            true,
		"expires_at":           data.ExpiresAt,
		"has_llm":              data.LLMAPIKey != "",
		"has_search":           data.SearchAPIKey != "",
		"llm_type":             data.LLMType,
		"search_type":          data.SearchType,
		"sn":                   a.licenseClient.GetSN(),
		"server_url":           a.licenseClient.GetServerURL(),
		"daily_analysis_limit": limit,
		"daily_analysis_count": count,
		"daily_analysis_date":  date,
	}
}

// LoadSavedActivation attempts to load saved activation data from local storage
func (a *App) LoadSavedActivation(sn string) (*ActivationResult, error) {
	if a.licenseClient == nil {
		a.licenseClient = agent.NewLicenseClient(a.Log)
	}

	err := a.licenseClient.LoadActivationData(sn)
	if err != nil {
		return &ActivationResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	data := a.licenseClient.GetData()
	return &ActivationResult{
		Success:   true,
		Message:   "从本地加载激活数据成功",
		ExpiresAt: data.ExpiresAt,
	}, nil
}

// GetActivatedLLMConfig returns the LLM config from activation (for internal use)
func (a *App) GetActivatedLLMConfig() *agent.ActivationData {
	if a.licenseClient == nil || !a.licenseClient.IsActivated() {
		return nil
	}
	return a.licenseClient.GetData()
}

// DeactivateLicense clears the activation
func (a *App) DeactivateLicense() {
	if a.licenseClient != nil {
		a.licenseClient.ClearSavedData()
	}
}

// RefreshLicense refreshes the license from server using stored SN
func (a *App) RefreshLicense() (*ActivationResult, error) {
	if a.licenseClient == nil || !a.licenseClient.IsActivated() {
		return &ActivationResult{
			Success: false,
			Message: "未激活，无法刷新",
		}, nil
	}

	sn := a.licenseClient.GetSN()
	if sn == "" {
		return &ActivationResult{
			Success: false,
			Message: "未找到序列号",
		}, nil
	}

	serverURL := a.licenseClient.GetServerURL()
	if serverURL == "" {
		// Try from config
		cfg, _ := a.GetConfig()
		serverURL = cfg.LicenseServerURL
	}
	if serverURL == "" {
		return &ActivationResult{
			Success: false,
			Message: "未找到授权服务器地址",
		}, nil
	}

	a.Log(fmt.Sprintf("[LICENSE] Refreshing license with SN: %s, Server: %s", sn, serverURL))

	// Re-activate with the same SN
	data, err := a.licenseClient.Activate(serverURL, sn)
	if err != nil {
		a.Log(fmt.Sprintf("[LICENSE] Refresh failed: %v", err))
		return &ActivationResult{
			Success: false,
			Message: fmt.Sprintf("刷新失败: %v", err),
		}, nil
	}

	// Save updated activation data
	if err := a.licenseClient.SaveActivationData(); err != nil {
		a.Log(fmt.Sprintf("[LICENSE] Warning: Failed to save refreshed data: %v", err))
	}

	// Reinitialize services with updated config
	cfg, _ := a.GetConfig()
	a.reinitializeServices(cfg)

	a.Log(fmt.Sprintf("[LICENSE] License refreshed successfully, expires: %s", data.ExpiresAt))

	return &ActivationResult{
		Success:   true,
		Message:   "授权刷新成功",
		ExpiresAt: data.ExpiresAt,
	}, nil
}

// IsLicenseActivated returns true if license is activated
func (a *App) IsLicenseActivated() bool {
	return a.licenseClient != nil && a.licenseClient.IsActivated()
}
