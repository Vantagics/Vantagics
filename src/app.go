package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	ctx                  context.Context
	chatService          *ChatService
	pythonService        *agent.PythonService
	dataSourceService    *agent.DataSourceService
	memoryService        *agent.MemoryService
	einoService          *agent.EinoService
	storageDir           string
	logger               *logger.Logger
	isChatGenerating     bool
	isChatOpen           bool
	cancelAnalysisMutex  sync.Mutex
	cancelAnalysis       bool
	activeThreadID       string
}

// AgentMemoryView structure for frontend
type AgentMemoryView struct {
	LongTerm   []string `json:"long_term"`
	MediumTerm []string `json:"medium_term"`
	ShortTerm  []string `json:"short_term"`
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

// onBeforeClose is called when the application is about to close
func (a *App) onBeforeClose(ctx context.Context) (prevent bool) {
	if a.isChatGenerating || a.isChatOpen {
		dialog, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Active Chat Session",
			Message:       "A chat session is currently open. Closing the application might lose context. Are you sure you want to exit?",
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "No",
			CancelButton:  "No",
		})

		if err != nil {
			return false
		}

		return dialog == "No"
	}
	return false
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
		
		es, err := agent.NewEinoService(cfg, a.dataSourceService, a.Log)
		if err != nil {
			a.Log(fmt.Sprintf("Failed to initialize EinoService: %v", err))
		} else {
			a.einoService = es
		}
	}

	// Initialize Logging if enabled
	if cfg.DetailedLog {
		a.logger.Init(dataDir)
	}
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

	// Short-term memory: Last 3 messages (what the AI sees in full detail)
	var shortTerm []string
	shortStart := 0
	if len(messages) > 3 {
		shortStart = len(messages) - 3
	}
	for _, msg := range messages[shortStart:] {
		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		shortTerm = append(shortTerm, fmt.Sprintf("[%s]: %s", msg.Role, content))
	}

	// Medium-term memory: Compressed summaries of older messages (messages 4-20)
	var mediumTerm []string
	if len(messages) > 3 {
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
				// Extract first meaningful line
				content := msg.Content
				if idx := strings.Index(content, "\n"); idx > 0 && idx < 200 {
					content = content[:idx]
				} else if len(content) > 200 {
					content = content[:200] + "..."
				}
				assistantFindings = append(assistantFindings, content)
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
		mediumTerm = []string{"No compressed history yet (conversation is short enough to fit in short-term memory)."}
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
		global, sessionLong, _ := a.memoryService.GetMemories(threadID)
		for _, mem := range global {
			longTerm = append(longTerm, fmt.Sprintf("🌐 %s", mem))
		}
		for _, mem := range sessionLong {
			longTerm = append(longTerm, fmt.Sprintf("📌 %s", mem))
		}
	}

	// If nothing substantive found, show a meaningful message
	if len(longTerm) == 0 {
		longTerm = append(longTerm, "暂无提取到的关键信息，继续对话后将自动提取分析主题、涉及表格、关键发现等。")
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
			MaxTokens:    4096,
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

	return os.WriteFile(path, data, 0644)
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

		respMsg, err := a.einoService.RunAnalysisWithProgress(a.ctx, history, dataSourceID, threadID, sessionDir, progressCallback, fileSavedCallback, a.IsCancelRequested)
		var resp string
		if err != nil {
			resp = fmt.Sprintf("Error: %v", err)
			if cfg.DetailedLog {
				a.logChatToFile(threadID, "SYSTEM ERROR", resp)
			}
			return "", err
		}
		resp = respMsg.Content

		if cfg.DetailedLog {
			a.logChatToFile(threadID, "LLM RESPONSE", resp)
		}

		startPost := time.Now()
		// Detect and store chart data
		var chartData *ChartData

		// Priority order: ECharts > Image > Table > CSV
		// 1. ECharts JSON
		reECharts := regexp.MustCompile("(?s)```[ \\t]*json:echarts\\s*({.*?})\\s*```")
		matchECharts := reECharts.FindStringSubmatch(resp)
		if len(matchECharts) > 1 {
			chartData = &ChartData{Charts: []ChartItem{{Type: "echarts", Data: matchECharts[1]}}}
			runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
				"sessionId": threadID,
				"type":      "echarts",
				"data":      matchECharts[1],
			})
		}

		// 2. Markdown Image (Base64)
		if chartData == nil {
			reImage := regexp.MustCompile(`!\[.*?\]\((data:image\/.*?;base64,.*?)\)`)
			matchImage := reImage.FindStringSubmatch(resp)
			if len(matchImage) > 1 {
				chartData = &ChartData{Charts: []ChartItem{{Type: "image", Data: matchImage[1]}}}
				runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
					"sessionId": threadID,
					"type":      "image",
					"data":      matchImage[1],
				})
			}
		}

		// 3. Check for saved chart files (e.g., chart.png from Python tool)
		// This should be checked BEFORE table data, as saved images are more visual than tables
		if chartData == nil && threadID != "" {
			// Get session files to see if chart images were saved
			sessionFiles, err := a.chatService.GetSessionFiles(threadID)
			if err == nil {
				// Collect ALL chart image files
				var chartItems []ChartItem
				for _, file := range sessionFiles {
					if file.Type == "image" && (file.Name == "chart.png" || strings.HasPrefix(file.Name, "chart")) {
						// Read the image file and encode as base64
						filePath := filepath.Join(sessionDir, "files", file.Name)
						if imageData, err := os.ReadFile(filePath); err == nil {
							base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
							chartItems = append(chartItems, ChartItem{Type: "image", Data: base64Data})
							a.Log(fmt.Sprintf("[CHART] Detected saved chart file: %s", file.Name))
						}
					}
				}

				// If we found any chart files, create ChartData with all of them
				if len(chartItems) > 0 {
					chartData = &ChartData{Charts: chartItems}
					// Emit dashboard update with the first chart (frontend will handle navigation)
					runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
						"sessionId": threadID,
						"type":      "image",
						"data":      chartItems[0].Data,
						"chartData": chartData, // Send full chart data for multi-chart support
					})
				}
			}
		}

		// 4. Dashboard Data Update (Metrics & Insights)
		reDashboard := regexp.MustCompile("(?s)```[ \\t]*json:dashboard\\s*({.*?})\\s*```")
		matchDashboard := reDashboard.FindStringSubmatch(resp)
		if len(matchDashboard) > 1 {
			var data DashboardData
			if err := json.Unmarshal([]byte(matchDashboard[1]), &data); err == nil {
				runtime.EventsEmit(a.ctx, "dashboard-data-update", data)
			} else {
				a.Log(fmt.Sprintf("Failed to unmarshal dashboard data: %v", err))
			}
		}

		// 5. Table Data (JSON array from SQL results or analysis)
		if chartData == nil {
			reTable := regexp.MustCompile("(?s)```[ \\t]*json:table\\s*(\\[.*?\\])\\s*```")
			matchTable := reTable.FindStringSubmatch(resp)
			if len(matchTable) > 1 {
				var tableData []map[string]interface{}
				if err := json.Unmarshal([]byte(matchTable[1]), &tableData); err == nil {
					tableDataJSON, _ := json.Marshal(tableData)
					chartData = &ChartData{Charts: []ChartItem{{Type: "table", Data: string(tableDataJSON)}}}
					runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
						"sessionId": threadID,
						"type":      "table",
						"data":      tableData,
					})
				}
			}
		}

		// 6. CSV Download Link (data URL)
		if chartData == nil {
			reCSV := regexp.MustCompile(`\[.*?\]\((data:text/csv;base64,[A-Za-z0-9+/=]+)\)`)
			matchCSV := reCSV.FindStringSubmatch(resp)
			if len(matchCSV) > 1 {
				chartData = &ChartData{Charts: []ChartItem{{Type: "csv", Data: matchCSV[1]}}}
				runtime.EventsEmit(a.ctx, "dashboard-update", map[string]interface{}{
					"sessionId": threadID,
					"type":      "csv",
					"data":      matchCSV[1],
				})
			}
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

		a.Log(fmt.Sprintf("[TIMING] Post-processing response took: %v", time.Since(startPost)))
		a.Log(fmt.Sprintf("[TIMING] Total SendMessage (Eino) took: %v", time.Since(startTotal)))

		// Auto-extract metrics from analysis response
		if resp != "" && userMessageID != "" {
			go func() {
				// Small delay to ensure frontend has processed the response
				time.Sleep(1 * time.Second)
				
				// Notify frontend that metrics extraction is starting
				runtime.EventsEmit(a.ctx, "metrics-extracting", userMessageID)
				
				if err := a.ExtractMetricsFromAnalysis(userMessageID, resp); err != nil {
					a.Log(fmt.Sprintf("Failed to extract metrics for message %s: %v", userMessageID, err))
				}
			}()
		}

		return resp, nil
	}

	langPrompt := a.getLangPrompt(cfg)
	fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", message, langPrompt)

	llm := agent.NewLLMService(cfg, a.Log)
	startChat := time.Now()
	resp, err := llm.Chat(a.ctx, fullMessage)
	a.Log(fmt.Sprintf("[TIMING] LLM Chat (Standard) took: %v", time.Since(startChat)))

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
			
			if err := a.ExtractMetricsFromAnalysis(messageID, resp); err != nil {
				a.Log(fmt.Sprintf("Failed to extract metrics for standard LLM response: %v", err))
			}
		}()
	}

	a.Log(fmt.Sprintf("[TIMING] Total SendMessage (Standard) took: %v", time.Since(startTotal)))
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
	return a.chatService.DeleteThread(threadID)
}

// CreateChatThread creates a new chat thread with a unique title
func (a *App) CreateChatThread(dataSourceID, title string) (ChatThread, error) {
	if a.chatService == nil {
		return ChatThread{}, fmt.Errorf("chat service not initialized")
	}

	// Check if there's an active analysis session running
	if a.isChatGenerating {
		return ChatThread{}, fmt.Errorf("当前有分析会话进行中，创建新的会话将影响现有分析会话。请等待当前分析完成后再创建新会话。")
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
	runtime.EventsEmit(a.ctx, "chat-loading", true)
	defer runtime.EventsEmit(a.ctx, "chat-loading", false)

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

	runtime.EventsEmit(a.ctx, "thread-updated", threadID)
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
	return a.chatService.ClearHistory()
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
			Timestamp:    rec.Timestamp.Format("2006-01-02 15:04:05"),
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
func (a *App) ExtractMetricsFromAnalysis(messageId string, analysisContent string) error {
	cfg, err := a.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Build metrics extraction prompt
	var prompt string
	if cfg.Language == "简体中文" {
		prompt = fmt.Sprintf(`请从以下分析结果中提取最重要的关键指标，以JSON格式返回。

要求：
1. 只返回JSON数组，不要其他文字说明
2. 每个指标必须包含：name（指标名称）、value（数值）、unit（单位，可选）
3. 最多提取6个最重要的业务指标
4. 优先提取：总量、增长率、平均值、比率等核心业务指标
5. 数值要准确，来源于分析内容
6. 单位要合适（如：个、%%、$、次/年、天等）
7. 指标名称要简洁明了

示例格式：
[
  {"name":"总销售额","value":"1,234,567","unit":"$"},
  {"name":"增长率","value":"+15.5","unit":"%%"},
  {"name":"平均订单价值","value":"89.50","unit":"$"}
]

分析内容：
%s

请返回JSON：`, analysisContent)
	} else {
		prompt = fmt.Sprintf(`Please extract the most important key metrics from the following analysis results in JSON format.

Requirements:
1. Return only JSON array, no other text
2. Each metric must include: name, value, unit (optional)
3. Extract at most 6 most important business metrics
4. Prioritize: totals, growth rates, averages, ratios and other core business metrics
5. Values must be accurate from the analysis content
6. Use appropriate units (e.g., items, %%, $, times/year, days, etc.)
7. Metric names should be concise and clear

Example format:
[
  {"name":"Total Sales","value":"1,234,567","unit":"$"},
  {"name":"Growth Rate","value":"+15.5","unit":"%%"},
  {"name":"Average Order Value","value":"89.50","unit":"$"}
]

Analysis content:
%s

Please return JSON:`, analysisContent)
	}

	// Try extraction up to 3 times
	for attempt := 1; attempt <= 3; attempt++ {
		err := a.tryExtractMetrics(messageId, prompt, attempt)
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
func (a *App) tryExtractMetrics(messageId string, prompt string, attempt int) error {
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

	// Validate metrics content
	if len(metrics) == 0 {
		return fmt.Errorf("no metrics found in JSON")
	}

	// Validate each metric has required fields
	for i, metric := range metrics {
		if _, hasName := metric["name"]; !hasName {
			return fmt.Errorf("metric %d missing 'name' field", i)
		}
		if _, hasValue := metric["value"]; !hasValue {
			return fmt.Errorf("metric %d missing 'value' field", i)
		}
	}

	// Save metrics JSON
	if err := a.SaveMetricsJson(messageId, jsonStr); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	// Notify frontend
	runtime.EventsEmit(a.ctx, "metrics-extracted", map[string]interface{}{
		"messageId": messageId,
		"metrics":   metrics,
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
