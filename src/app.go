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

// App struct
type App struct {
	ctx context.Context
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
	}
}

func (a *App) getStorageDir() (string, error) {
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
