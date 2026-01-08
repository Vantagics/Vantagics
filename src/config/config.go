package config

// Config structure
type Config struct {
	LLMProvider       string `json:"llmProvider"`
	APIKey            string `json:"apiKey"`
	BaseURL           string `json:"baseUrl"`
	ModelName         string `json:"modelName"`
	MaxTokens         int    `json:"maxTokens"`
	DarkMode          bool   `json:"darkMode"`
	LocalCache        bool   `json:"localCache"`
	Language          string `json:"language"`
	ClaudeHeaderStyle string `json:"claudeHeaderStyle"`
	DataCacheDir      string `json:"dataCacheDir"`
	PythonPath        string `json:"pythonPath"`
	MaxPreviewRows    int    `json:"maxPreviewRows"`
	DetailedLog       bool   `json:"detailedLog"`
}
