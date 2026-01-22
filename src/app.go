package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	gort "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"rapidbi/agent"
	"rapidbi/config"
	"rapidbi/logger"

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

// DashboardData structure
type DashboardData struct {
	Metrics  []Metric  `json:"metrics"`
	Insights []Insight `json:"insights"`
}

// App struct
type App struct {
	ctx                   context.Context
	chatService           *ChatService
	pythonService         *agent.PythonService
	dataSourceService     *agent.DataSourceService
	memoryService         *agent.MemoryService
	workingContextManager *agent.WorkingContextManager
	analysisPathManager   *agent.AnalysisPathManager
	preferenceLearner     *agent.PreferenceLearner
	einoService           *agent.EinoService
	storageDir            string
	logger                *logger.Logger
	isChatGenerating      bool
	isChatOpen            bool
	cancelAnalysisMutex   sync.Mutex
	cancelAnalysis        bool
	activeThreadID        string
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
		pythonService:    agent.NewPythonService(),
		logger:           logger.NewLogger(),
		isChatGenerating: false,
		isChatOpen:       false,
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
	// Only prevent close if there's an active analysis running
	if a.isChatGenerating {
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
		
		// Initialize working context manager for UI state tracking
		a.workingContextManager = agent.NewWorkingContextManager(dataDir)
		a.Log("[STARTUP] Working context manager initialized")
		
		// Initialize analysis path manager for storyline tracking
		a.analysisPathManager = agent.NewAnalysisPathManager(dataDir)
		a.Log("[STARTUP] Analysis path manager initialized")
		
		// Initialize preference learner for user behavior tracking
		a.preferenceLearner = agent.NewPreferenceLearner(dataDir)
		a.Log("[STARTUP] Preference learner initialized")
		
		a.Log(fmt.Sprintf("[STARTUP] Initializing EinoService with provider: %s, model: %s", cfg.LLMProvider, cfg.ModelName))
		es, err := agent.NewEinoService(cfg, a.dataSourceService, a.memoryService, a.workingContextManager, a.Log)
		if err != nil {
			a.Log(fmt.Sprintf("Failed to initialize EinoService: %v", err))
		} else {
			a.einoService = es
			a.Log("[STARTUP] EinoService initialized successfully")
		}
	}

	// Initialize Logging if enabled
	if cfg.DetailedLog {
		a.logger.Init(dataDir)
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

func (a *App) getStorageDir() (string, error) {
	if a.storageDir != "" {
		return a.storageDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "RapidBI"), nil
}

func (a *App) getConfigPath() (string, error) {
	dir, err := a.getStorageDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// GetConfig loads the config from the ~/rapidbi/config.json
func (a *App) GetConfig() (config.Config, error) {
	path, err := a.getConfigPath()
	if err != nil {
		return config.Config{}, err
	}

	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, "RapidBI")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return config.Config{
			LLMProvider:  "OpenAI",
			ModelName:    "gpt-4o",
			MaxTokens:    8192, // Safe default, will be adjusted per provider
			LocalCache:   true,
			Language:     "English",
			DataCacheDir: defaultDataDir,
			MaxPreviewRows: 100,
		},
		nil
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

	return cfg, nil
}

// SaveConfig saves the config to the ~/rapidbi/config.json
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
	if cfg.DetailedLog {
		// Enable logging if not already enabled (checked inside Init usually, but here we can force re-init or check if active)
		// Our logger handles re-init gracefully by closing old file.
		// However, check if we need to switch on.
		logDir := cfg.DataCacheDir
		if logDir == "" {
			logDir = dir // fallback to storage dir
		}
		a.logger.Init(logDir)
	} else {
		// Disable logging
		a.logger.Close()
	}

	// Save the configuration file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	// Reinitialize services that depend on configuration
	a.reinitializeServices(cfg)

	// Notify frontend that configuration has been updated
	runtime.EventsEmit(a.ctx, "config-updated")

	a.Log("Configuration saved and services reinitialized")
	return nil
}

// reinitializeServices reinitializes services that depend on configuration
func (a *App) reinitializeServices(cfg config.Config) {
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
				"type":    "eino_service",
				"message": fmt.Sprintf("Failed to initialize analysis service: %v", err),
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
	llm := agent.NewLLMService(cfg, a.Log)
	resp, err := llm.Chat(a.ctx, "hi LLM, I'm just test the connection. Just answer ok to me without other infor.")
	if err != nil {
		return ConnectionResult{
			Success: false,
			Message: err.Error(),
		}
	}

	// If it returns the "Please set your API key" message, it's a soft failure
	if resp == "Please set your API key in settings." {
		return ConnectionResult{
			Success: false,
			Message: "API key is missing",
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
func (a *App) TestSearchTools(engineURL string) ConnectionResult {
	if engineURL == "" {
		return ConnectionResult{
			Success: false,
			Message: "Search engine URL is required",
		}
	}

	a.logger.Log("[SEARCH-TEST] Starting search tools test with query: whitehouse")

	// Create search engine config
	searchEngine := &config.SearchEngine{
		ID:      "test-engine",
		Name:    "Test Engine",
		URL:     engineURL,
		Enabled: true,
		Tested:  true,
	}

	// Get proxy config from eino service if available
	var proxyConfig *config.ProxyConfig
	if a.einoService != nil {
		cfg := a.einoService.GetConfig()
		if cfg.ProxyConfig != nil && cfg.ProxyConfig.Enabled && cfg.ProxyConfig.Tested {
			proxyConfig = cfg.ProxyConfig
		}
	}

	// Test 1: Web Search
	a.logger.Log("[SEARCH-TEST] Step 1: Testing web_search tool...")
	searchTool := agent.NewWebSearchTool(a.logger.Log, searchEngine, proxyConfig)
	
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	searchInput := `{"query": "whitehouse", "max_results": 3}`
	searchResult, err := searchTool.InvokableRun(ctx, searchInput)
	if err != nil {
		a.logger.Log(fmt.Sprintf("[SEARCH-TEST] web_search failed: %v", err))
		return ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("web_search failed: %v", err),
		}
	}

	a.logger.Log(fmt.Sprintf("[SEARCH-TEST] web_search succeeded, got %d bytes of results", len(searchResult)))

	// Parse search results to get first URL
	var searchResults []agent.SearchResult
	if err := json.Unmarshal([]byte(searchResult), &searchResults); err != nil {
		a.logger.Log(fmt.Sprintf("[SEARCH-TEST] Failed to parse search results: %v", err))
		return ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse search results: %v", err),
		}
	}

	if len(searchResults) == 0 {
		a.logger.Log("[SEARCH-TEST] No search results found")
		return ConnectionResult{
			Success: false,
			Message: "No search results found for 'whitehouse'",
		}
	}

	// Test 2: Web Fetch
	firstURL := searchResults[0].URL
	a.logger.Log(fmt.Sprintf("[SEARCH-TEST] Step 2: Testing web_fetch tool with URL: %s", firstURL))
	
	fetchTool := agent.NewWebFetchTool(a.logger.Log, proxyConfig)
	
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	fetchInput := fmt.Sprintf(`{"url": "%s", "mode": "truncated"}`, firstURL)
	fetchResult, err := fetchTool.InvokableRun(ctx2, fetchInput)
	if err != nil {
		a.logger.Log(fmt.Sprintf("[SEARCH-TEST] web_fetch failed: %v", err))
		return ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("web_fetch failed: %v", err),
		}
	}

	a.logger.Log(fmt.Sprintf("[SEARCH-TEST] web_fetch succeeded, got %d bytes of content", len(fetchResult)))

	// Success message with details
	successMsg := fmt.Sprintf(
		"✓ Search tools test passed!\n\n"+
		"web_search: Found %d results for 'whitehouse'\n"+
		"First result: %s\n\n"+
		"web_fetch: Successfully fetched %d bytes from:\n%s",
		len(searchResults),
		searchResults[0].Title,
		len(fetchResult),
		firstURL,
	)

	a.logger.Log("[SEARCH-TEST] All tests passed successfully")

	return ConnectionResult{
		Success: true,
		Message: successMsg,
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

// SendMessage sends a message to the AI
func (a *App) SendMessage(threadID, message, userMessageID string) (string, error) {
	if a.chatService == nil {
		return "", fmt.Errorf("chat service not initialized")
	}

	cfg, err := a.GetConfig()
	if err != nil {
		return "", err
	}

	startTotal := time.Now()

	// Log user message if threadID provided
	if threadID != "" && cfg.DetailedLog {
		a.logChatToFile(threadID, "USER REQUEST", message)
	}

	a.isChatGenerating = true
	defer func() { a.isChatGenerating = false }()

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

		// Create progress callback to emit events to frontend
		progressCallback := func(update agent.ProgressUpdate) {
			runtime.EventsEmit(a.ctx, "analysis-progress", update)
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
					runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
						"sessionId": threadID,
						"type":      "echarts",
						"data":      jsonStr, // Always send full data for real-time display
					})
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
				runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
					"sessionId": threadID,
					"type":      "image",
					"data":      matchImage[1],
				})
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
			
			// Create ChartData with all collected charts
			if len(chartItems) > 0 {
				chartData = &ChartData{Charts: chartItems}
				a.Log(fmt.Sprintf("[CHART] Total charts collected: %d (ECharts + Images)", len(chartItems)))
			}

			// 4. Dashboard Data Update (Metrics & Insights)
			// Match until closing ``` to handle nested objects (same fix as echarts/table)
			reDashboard := regexp.MustCompile("(?s)```\\s*json:dashboard\\s*\\n([\\s\\S]+?)\\n\\s*```")
			matchDashboard := reDashboard.FindStringSubmatch(resp)
			if len(matchDashboard) > 1 {
				jsonStr := strings.TrimSpace(matchDashboard[1])
				var data DashboardData
				if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
					runtime.EventsEmit(a.ctx, "dashboard-data-update", data)
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
				var tableData []map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &tableData); err == nil {
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
					
					runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
						"sessionId": threadID,
						"type":      "table",
						"data":      tableData, // Always send full data for real-time display
					})
				} else {
					maxLen := 500
					if len(jsonStr) < maxLen {
						maxLen = len(jsonStr)
					}
					a.Log(fmt.Sprintf("[CHART] Failed to parse table JSON: %v\nJSON string (first 500 chars): %s", err, jsonStr[:maxLen]))
				}
			}

			// 6. CSV Download Link (data URL) - always check
			reCSV := regexp.MustCompile(`\[.*?\]\((data:text/csv;base64,[A-Za-z0-9+/=]+)\)`)
			matchCSV := reCSV.FindStringSubmatch(resp)
			if len(matchCSV) > 1 {
				chartItems = append(chartItems, ChartItem{Type: "csv", Data: matchCSV[1]})
				a.Log("[CHART] Detected CSV data")
				runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
					"sessionId": threadID,
					"type":      "csv",
					"data":      matchCSV[1],
				})
			}
			
			// Update chartData with all collected items (if not already set)
			if chartData == nil && len(chartItems) > 0 {
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
				// Note: These are estimates based on typical analysis patterns
				// In future, we can collect actual timing from eino service
				totalSecs := analysisDuration.Seconds()
				
				// Estimate stage durations (rough approximation)
				// Typical breakdown: AI ~60%, SQL ~20%, Python ~15%, Other ~5%
				aiTime := totalSecs * 0.60
				sqlTime := totalSecs * 0.20
				pythonTime := totalSecs * 0.15
				otherTime := totalSecs * 0.05
				
				timingData := map[string]interface{}{
					"total_seconds":            totalSecs,
					"total_minutes":            minutes,
					"total_seconds_remainder":  seconds,
					"analysis_type":            "eino_service",
					"timestamp":                time.Now().Unix(),
					"stages": []map[string]interface{}{
						{
							"name":       "AI 分析",
							"duration":   aiTime,
							"percentage": 60.0,
							"description": "LLM 理解需求、生成代码和分析结果",
						},
						{
							"name":       "SQL 查询",
							"duration":   sqlTime,
							"percentage": 20.0,
							"description": "数据库查询和数据提取",
						},
						{
							"name":       "Python 处理",
							"duration":   pythonTime,
							"percentage": 15.0,
							"description": "数据处理和图表生成",
						},
						{
							"name":       "其他",
							"duration":   otherTime,
							"percentage": 5.0,
							"description": "初始化和后处理",
						},
					},
				}
				
				assistantMsg := ChatMessage{
					ID:         strconv.FormatInt(time.Now().UnixNano(), 10),
					Role:       "assistant",
					Content:    resp,
					Timestamp:  time.Now().Unix(),
					ChartData:  chartData, // Attach chart data to assistant message
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
					runtime.EventsEmit(a.ctx, "analysis-completed", map[string]interface{}{
						"threadId":       threadID,
						"userMessageId":  userMessageID,
						"assistantMsgId": assistantMsg.ID,
						"hasChartData":   chartData != nil,
					})
					a.Log(fmt.Sprintf("[DASHBOARD] Emitted analysis-completed event for message %s", userMessageID))
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

// CancelAnalysis cancels the ongoing analysis for the active thread
func (a *App) CancelAnalysis() error {
	a.cancelAnalysisMutex.Lock()
	defer a.cancelAnalysisMutex.Unlock()

	if !a.isChatGenerating {
		return fmt.Errorf("no analysis is currently running")
	}

	a.cancelAnalysis = true
	a.Log(fmt.Sprintf("[CANCEL] Analysis cancellation requested for thread: %s", a.activeThreadID))
	return nil
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

// CreateRapidBIEnvironment creates a dedicated virtual environment for RapidBI
func (a *App) CreateRapidBIEnvironment() (string, error) {
	return a.pythonService.CreateRapidBIEnvironment()
}

// CheckRapidBIEnvironmentExists checks if a rapidbi environment already exists
func (a *App) CheckRapidBIEnvironmentExists() bool {
	return a.pythonService.CheckRapidBIEnvironmentExists()
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
	isGenerating := a.isChatGenerating
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
		runtime.EventsEmit(a.ctx, "clear-dashboard-data", map[string]interface{}{
			"reason": "thread_deleted",
			"threadID": threadID,
		})
	}
	
	return nil
}

// CreateChatThread creates a new chat thread with a unique title
func (a *App) CreateChatThread(dataSourceID, title string) (ChatThread, error) {
	if a.chatService == nil {
		return ChatThread{}, fmt.Errorf("chat service not initialized")
	}

	// Check if there's an active analysis session running
	if a.isChatGenerating {
		// Get current language configuration
		cfg, _ := a.GetConfig()
		
		var errorMessage string
		if cfg.Language == "简体中文" {
			errorMessage = "当前有分析会话进行中，创建新的会话将影响现有分析会话。请等待当前分析完成后再创建新会话。"
		} else {
			errorMessage = "An analysis session is currently in progress. Creating a new session will affect the existing analysis session. Please wait for the current analysis to complete before creating a new session."
		}
		
		return ChatThread{}, fmt.Errorf(errorMessage)
	}

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

	cfg, _ := a.GetConfig()
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
		runtime.EventsEmit(a.ctx, "dashboard-data-update", DashboardData{
			Insights: insights,
		})
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
	cfg, _ := a.GetConfig()
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
	isGenerating := a.isChatGenerating
	if isGenerating {
		// Cancel any ongoing analysis
		a.cancelAnalysis = true
		a.Log("[CLEAR-HISTORY] Cancelling ongoing analysis before clearing history")
	}
	a.cancelAnalysisMutex.Unlock()
	
	// Wait for cancellation to take effect if needed
	if isGenerating {
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
	runtime.EventsEmit(a.ctx, "clear-dashboard-data", map[string]interface{}{
		"reason": "history_cleared",
	})
	
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

// ImportExcelDataSource imports an Excel file as a data source
func (a *App) ImportExcelDataSource(name string, filePath string) (*agent.DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt, "")
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
		return a.SendMessage("", prompt, "")
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
		return a.SendMessage("", prompt, "")
	}

	ds, err := a.dataSourceService.ImportJSON(name, filePath, headerGen)
	if err == nil && ds != nil {
		go a.analyzeDataSource(ds.ID)
	}
	return ds, err
}

// AddDataSource adds a new data source with generic configuration
func (a *App) AddDataSource(name string, driverType string, config map[string]string) (*agent.DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	dsConfig := agent.DataSourceConfig{
		OriginalFile: config["filePath"],
		Host:         config["host"],
		Port:         config["port"],
		User:         config["user"],
		Password:     config["password"],
		Database:     config["database"],
		StoreLocally: config["storeLocally"] == "true",
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt, "")
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

// RenameDataSource renames a data source
func (a *App) RenameDataSource(id string, newName string) error {
	if a.dataSourceService == nil {
		return fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.RenameDataSource(id, newName)
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

		Host:     host,

		Port:     port,

		User:     user,

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
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Version         string            `json:"version"`
	Author          string            `json:"author"`
	Category        string            `json:"category"`
	Keywords        []string          `json:"keywords"`
	RequiredColumns []string          `json:"required_columns"`
	Tools           []string          `json:"tools"`
	Enabled         bool              `json:"enabled"`
	Icon            string            `json:"icon"`
	Tags            []string          `json:"tags"`
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
	if a.einoService == nil {
		return fmt.Errorf("eino service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return fmt.Errorf("skill manager not available")
	}

	return skillManager.EnableSkill(skillID)
}

// DisableSkill disables a skill by ID
func (a *App) DisableSkill(skillID string) error {
	if a.einoService == nil {
		return fmt.Errorf("eino service not initialized")
	}

	skillManager := a.einoService.GetSkillManager()
	if skillManager == nil {
		return fmt.Errorf("skill manager not available")
	}

	return skillManager.DisableSkill(skillID)
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

	// Notify frontend
	runtime.EventsEmit(a.ctx, "metrics-extracted", map[string]interface{}{
		"messageId": messageId,
		"metrics":   validMetrics,
	})

	a.Log(fmt.Sprintf("Metrics extracted and saved for message %s (attempt %d)", messageId, attempt))
	return nil
}

// getConfigForExtraction gets config for metrics extraction
func (a *App) getConfigForExtraction() config.Config {
	cfg, _ := a.GetConfig()
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
	
	// Limit to 5 suggestions
	if len(insights) > 5 {
		insights = insights[:5]
	}
	
	if len(insights) > 0 {
		a.Log(fmt.Sprintf("[SUGGESTIONS] Extracted %d suggestions from analysis for message %s", len(insights), userMessageID))
		
		// Emit to dashboard
		runtime.EventsEmit(a.ctx, "dashboard-data-update", DashboardData{
			Insights: insights,
		})
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
			// Notify frontend
			runtime.EventsEmit(a.ctx, "metrics-extracted", map[string]interface{}{
				"messageId": messageId,
				"metrics":   metrics,
			})
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

	cfg, _ := a.GetConfig()
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
