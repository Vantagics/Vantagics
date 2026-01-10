package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	Text string `json:"text"`
	Icon string `json:"icon"`
}

// DashboardData structure
type DashboardData struct {
	Metrics  []Metric  `json:"metrics"`
	Insights []Insight `json:"insights"`
}

// App struct
type App struct {
	ctx               context.Context
	chatService       *ChatService
	pythonService     *agent.PythonService
	dataSourceService *agent.DataSourceService
	memoryService     *agent.MemoryService
	einoService       *agent.EinoService
	storageDir        string
	logger            *logger.Logger
	isChatGenerating  bool
	isChatOpen        bool
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
		
		es, err := agent.NewEinoService(cfg, a.dataSourceService)
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
func (a *App) GetAgentMemory(threadID string) (AgentMemoryView, error) {
	if a.memoryService == nil || a.chatService == nil {
		return AgentMemoryView{}, fmt.Errorf("services not initialized")
	}

	global, sessionLong, sessionMedium := a.memoryService.GetMemories(threadID)
	
	// Combine Global and Session Long for the "Long Term" view
	longTerm := append(global, sessionLong...)

	// Get Short Term (Recent messages)
	// We'll fetch the thread and take the last 10 messages as "Short Term Memory"
	threads, _ := a.chatService.LoadThreads()
	var short []string
	
	for _, t := range threads {
		if t.ID == threadID {
			// Take last 10 messages
			start := 0
			if len(t.Messages) > 10 {
				start = len(t.Messages) - 10
			}
			for _, msg := range t.Messages[start:] {
				short = append(short, fmt.Sprintf("[%s]: %s", msg.Role, msg.Content))
			}
			break
		}
	}

	return AgentMemoryView{
		LongTerm:   longTerm,
		MediumTerm: sessionMedium,
		ShortTerm:  short,
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

// GetDashboardData returns mock data for the dashboard
func (a *App) GetDashboardData() DashboardData {
	return DashboardData{
		Metrics: []Metric{
			{Title: "Total Sales", Value: "$45,231", Change: "+12.5%"},
			{Title: "Active Users", Value: "2,845", Change: "+8.2%"},
			{Title: "Conversion Rate", Value: "3.4%", Change: "-1.2%"},
			{Title: "Avg. Session", Value: "4m 32s", Change: "+2.1%"},
		},
		Insights: []Insight{
			{Text: "Sales are trending up this week! Consider increasing your ad spend.", Icon: "trending-up"},
			{Text: "You have a high user retention rate. Keep up the good work!", Icon: "user-check"},
			{Text: "Conversion rate dropped slightly. Check your checkout flow.", Icon: "alert-circle"},
		},
	}
}

func (a *App) getLangPrompt(cfg config.Config) string {
	if cfg.Language == "简体中文" {
		return "Simplified Chinese"
	}
	return "English"
}

// SendMessage sends a message to the LLM and returns the response
func (a *App) SendMessage(threadID string, message string) (string, error) {
	cfg, err := a.GetConfig()
	if err != nil {
		return "", err
	}

	// Log user message if threadID provided
	if threadID != "" && cfg.DetailedLog {
		a.logChatToFile(threadID, "USER REQUEST", message)
	}

	a.isChatGenerating = true
	defer func() { a.isChatGenerating = false }()

	// Check if we should use Eino (if thread has DataSourceID)
	var useEino bool
	if threadID != "" && a.einoService != nil {
		threads, _ := a.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID && t.DataSourceID != "" {
				useEino = true
				break
			}
		}
	}

	if useEino {
		// Load history
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

		// Add current message (Eino expects the new user message in the input list for the chain we built)
		history = append(history, &schema.Message{Role: schema.User, Content: message})

		respMsg, err := a.einoService.RunAnalysis(a.ctx, history)
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
		return resp, nil
	}

	langPrompt := a.getLangPrompt(cfg)
	fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", message, langPrompt)

	llm := agent.NewLLMService(cfg, a.Log)
	resp, err := llm.Chat(a.ctx, fullMessage)

	// Log LLM response if threadID provided
	if threadID != "" && cfg.DetailedLog {
		if err != nil {
			a.logChatToFile(threadID, "SYSTEM ERROR", fmt.Sprintf("Error: %v", err))
		} else {
			a.logChatToFile(threadID, "LLM RESPONSE", resp)
		}
	}

	return resp, err
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
	
	thread, err := a.chatService.CreateThread(dataSourceID, title)
	if err != nil {
		return ChatThread{}, err
	}

	// If data source is selected, check for existing analysis and inject into memory
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
			// Found existing analysis, inject into session memory
			jsonBytes, err := json.MarshalIndent(target.Analysis, "", "  ")
			if err == nil {
				a.memoryService.AddSessionLongTermMemory(thread.ID, string(jsonBytes))
			}
			
			// Generate suggestions based on this analysis
			go a.generateAnalysisSuggestions(thread.ID, target.Analysis)
		}
	}

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
	sb.WriteString(fmt.Sprintf("Based on the following data source summary and schema, please suggest 3-5 distinct data analysis angles or questions that would be valuable for a business user. Please answer in %s. Provide the suggestions as a clear, structured, numbered list (1., 2., 3...). Each suggestion should include a brief, catchy title and a clear, one-sentence description of the analysis goal. End your response by telling the user (in %s) that they can select one or more analysis angles by replying with the corresponding number(s).", langPrompt, langPrompt))
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
	if a.dataSourceService == nil {
		return
	}

	a.Log(fmt.Sprintf("Starting analysis for source %s", dataSourceID))

	// 1. Get Tables
	tables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("Failed to get tables: %v", err))
		return
	}

	// 2. Sample Data & Construct Prompt
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
	description, err := llm.Chat(context.Background(), prompt) 
	
	if err != nil {
		a.Log(fmt.Sprintf("LLM Analysis failed: %v", err))
		return
	}

	if description == "" {
		a.Log("LLM returned empty response during analysis.")
		description = "No description provided by LLM."
	}

	// 4. Save Analysis to DataSource
	analysis := agent.DataSourceAnalysis{
		Summary: description,
		Schema:  tableSchemas,
	}

	if err := a.dataSourceService.UpdateAnalysis(dataSourceID, analysis); err != nil {
		a.Log(fmt.Sprintf("Failed to update data source analysis: %v", err))
		return
	}

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
		return a.SendMessage("", prompt)
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
		return a.SendMessage("", prompt)
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
		return a.SendMessage("", prompt)
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

// ShowMessage displays a message dialog
func (a *App) ShowMessage(typeStr string, title string, message string) {
	var dialogType runtime.DialogType
	switch typeStr {
	case "info":
		dialogType = runtime.InfoDialog
	case "warning":
		dialogType = runtime.WarningDialog
	case "error":
		dialogType = runtime.ErrorDialog
	default:
		dialogType = runtime.InfoDialog
	}

	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    dialogType,
		Title:   title,
		Message: message,
	})
}

