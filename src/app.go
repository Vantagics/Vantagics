package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config structure
type Config struct {
	LLMProvider  string `json:"llmProvider"`
	APIKey       string `json:"apiKey"`
	BaseURL      string `json:"baseUrl"`
	ModelName    string `json:"modelName"`
	MaxTokens    int    `json:"maxTokens"`
	DarkMode     bool   `json:"darkMode"`
	LocalCache   bool   `json:"localCache"`
	Language     string `json:"language"`
}

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
	ctx         context.Context
	chatService *ChatService
	storageDir  string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// Start system tray (Windows/Linux only, handled by build tags)
	runSystray(ctx)

	// Ensure the storage directory exists on startup
	path, _ := a.getStorageDir()
	if path != "" {
		_ = os.MkdirAll(path, 0755)
		chatPath := filepath.Join(path, "chat_history.json")
		a.chatService = NewChatService(chatPath)
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
	return filepath.Join(home, "rapidbi"), nil
}

func (a *App) getConfigPath() (string, error) {
	dir, err := a.getStorageDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// GetConfig loads the config from the ~/rapidbi/config.json
func (a *App) GetConfig() (Config, error) {
	path, err := a.getConfigPath()
	if err != nil {
		return Config{}, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return Config{
			LLMProvider: "OpenAI",
			ModelName:   "gpt-4o",
			MaxTokens:   4096,
			LocalCache:  true,
			Language:    "English",
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	return config, err
}

// SaveConfig saves the config to the ~/rapidbi/config.json
func (a *App) SaveConfig(config Config) error {
	dir, err := a.getStorageDir()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
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
func (a *App) SendMessage(message string) (string, error) {
	config, err := a.GetConfig()
	if err != nil {
		return "", err
	}

	llm := NewLLMService(config)
	return llm.Chat(a.ctx, message)
}

// GetChatHistory loads the chat history
func (a *App) GetChatHistory() ([]ChatThread, error) {
	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}
	return a.chatService.LoadThreads()
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

// ClearHistory clears all chat history
func (a *App) ClearHistory() error {
	if a.chatService == nil {
		return fmt.Errorf("chat service not initialized")
	}
	return a.chatService.ClearHistory()
}
