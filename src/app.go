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
	pythonService     *PythonService
	dataSourceService *DataSourceService
	memoryService     *agent.MemoryService
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
		pythonService:    NewPythonService(),
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
			chatPath := filepath.Join(path, "chat_history.json")
			a.chatService = NewChatService(chatPath)
			a.dataSourceService = NewDataSourceService(path, a.Log)
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
		chatPath := filepath.Join(dataDir, "chat_history.json")
		a.chatService = NewChatService(chatPath)
		a.dataSourceService = NewDataSourceService(dataDir, a.Log)
		a.memoryService = agent.NewMemoryService(cfg)
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

	llm := agent.NewLLMService(cfg, a.Log)
	resp, err := llm.Chat(a.ctx, message)

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
	// 1. Get Thread details
	threads, err := a.chatService.LoadThreads()
	if err != nil {
		a.Log(fmt.Sprintf("logChatToFile: Failed to load threads: %v", err))
		return
	}

	var thread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			thread = &threads[i]
			break
		}
	}
	if thread == nil {
		// Log warning to main log
		a.Log(fmt.Sprintf("logChatToFile: Thread %s not found in history", threadID))
		return
	}

	// 2. Get Data Source details
	sources, _ := a.dataSourceService.LoadDataSources()
	var dsName string = "Global"
	for _, ds := range sources {
		if ds.ID == thread.DataSourceID {
			dsName = ds.Name
			break
		}
	}

	// 3. Construct filename
	safeDSName := agent.SanitizeFilename(dsName)
	safeSessionName := agent.SanitizeFilename(thread.Title)
	filename := fmt.Sprintf("%s_%s.log", safeDSName, safeSessionName)
	
	// Use DataCacheDir for logs
	cfg, _ := a.GetConfig()
	logPath := filepath.Join(cfg.DataCacheDir, "chat_logs", filename)
	
	// Ensure dir exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		a.Log(fmt.Sprintf("logChatToFile: Failed to create log directory: %v", err))
		return
	}

	// 4. Append log
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
func (a *App) GetPythonEnvironments() []PythonEnvironment {
	return a.pythonService.ProbePythonEnvironments()
}

// ValidatePython checks the given Python path
func (a *App) ValidatePython(path string) PythonValidationResult {
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

	// If data source is selected, perform initial analysis asynchronously
	if dataSourceID != "" {
		go a.analyzeDataSourceAndMemorize(thread.ID, dataSourceID)
	}

	return thread, nil
}

func (a *App) analyzeDataSourceAndMemorize(threadID, dataSourceID string) {
	if a.dataSourceService == nil || a.memoryService == nil {
		return
	}

	a.Log(fmt.Sprintf("Starting analysis for thread %s, source %s", threadID, dataSourceID))

	// 1. Get Tables
	tables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		a.Log(fmt.Sprintf("Failed to get tables: %v", err))
		return
	}

	// 2. Sample Data & Construct Prompt
	var sb strings.Builder
	sb.WriteString("I am starting a new analysis on this database. Please describe this database based on the following schema and sample data.\n\n")
	
	schemaInfo := "Tables:\n"

	for _, tableName := range tables {
		sb.WriteString(fmt.Sprintf("Table: %s\n", tableName))
		schemaInfo += fmt.Sprintf("- %s: ", tableName)

		// Get 3 rows
		data, err := a.dataSourceService.GetDataSourceTableData(dataSourceID, tableName, 3)
		if err != nil {
			sb.WriteString("(Failed to fetch data)\n")
			continue
		}

		if len(data) > 0 {
			// Extract columns from first row keys
			var cols []string
			for k := range data[0] {
				cols = append(cols, k)
			}
			sb.WriteString(fmt.Sprintf("Columns: %s\nData:\n", strings.Join(cols, ", ")))
			schemaInfo += strings.Join(cols, ", ") + "\n"

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
	}

	// 3. Call LLM
	prompt := sb.String()
	
	// Use SendMessage logic but manually to control logging role/visibility if needed
	// For simplicity, reusing SendMessage but we might want to log this as 'System Analysis'
	// Since SendMessage hardcodes 'User', let's call llm directly and use logChatToFile manually
	
	cfg, _ := a.GetConfig()
	
	if cfg.DetailedLog {
		a.logChatToFile(threadID, "SYSTEM ANALYSIS PROMPT", prompt)
	}

	llm := agent.NewLLMService(cfg, a.Log)
	description, err := llm.Chat(context.Background(), prompt) // Use background context for async
	
	if err != nil {
		a.Log(fmt.Sprintf("LLM Analysis failed: %v", err))
		if cfg.DetailedLog {
			a.logChatToFile(threadID, "SYSTEM ANALYSIS ERROR", err.Error())
		}
		
		// Report error in chat window
		if a.chatService != nil {
			a.chatService.AddMessage(threadID, ChatMessage{
				ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
				Role:      "assistant",
				Content:   fmt.Sprintf("⚠️ Data Source Analysis Failed:\n%v\n\n(Long-term memory could not be constructed)", err),
				Timestamp: time.Now().Unix(),
			})
			runtime.EventsEmit(a.ctx, "thread-updated", threadID)
		}
		return
	}

	if description == "" {
		msg := "LLM returned empty response during analysis."
		a.Log(msg)
		if cfg.DetailedLog {
			a.logChatToFile(threadID, "SYSTEM ANALYSIS ERROR", msg)
		}
		
		// Report warning in chat window
		if a.chatService != nil {
			a.chatService.AddMessage(threadID, ChatMessage{
				ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
				Role:      "assistant",
				Content:   fmt.Sprintf("⚠️ Data Source Analysis Warning:\n%s", msg),
				Timestamp: time.Now().Unix(),
			})
			runtime.EventsEmit(a.ctx, "thread-updated", threadID)
		}
		description = "No description provided by LLM."
	}

	if cfg.DetailedLog {
		a.logChatToFile(threadID, "SYSTEM ANALYSIS RESPONSE", description)
	}

	// 4. Save to Memory
	// Combine Schema Info + Description
	finalMemory := fmt.Sprintf("Data Source Schema:\n%s\n\nDatabase Description:\n%s", schemaInfo, description)
	
	// Save to Session Long Term Memory
	a.memoryService.AddSessionLongTermMemory(threadID, finalMemory)
	a.Log("Analysis and memory update complete.")
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
func (a *App) GetDataSources() ([]DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}
	return a.dataSourceService.LoadDataSources()
}

// ImportExcelDataSource imports an Excel file as a data source
func (a *App) ImportExcelDataSource(name string, filePath string) (*DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt)
	}

	return a.dataSourceService.ImportExcel(name, filePath, headerGen)
}

// ImportCSVDataSource imports a CSV directory as a data source
func (a *App) ImportCSVDataSource(name string, dirPath string) (*DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	headerGen := func(prompt string) (string, error) {
		return a.SendMessage("", prompt)
	}

	return a.dataSourceService.ImportCSV(name, dirPath, headerGen)
}

// AddDataSource adds a new data source with generic configuration
func (a *App) AddDataSource(name string, driverType string, config map[string]string) (*DataSource, error) {
	if a.dataSourceService == nil {
		return nil, fmt.Errorf("data source service not initialized")
	}

	dsConfig := DataSourceConfig{
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

	return a.dataSourceService.ImportDataSource(name, driverType, dsConfig, headerGen)
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
	config := MySQLExportConfig{
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

	config := DataSourceConfig{

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

