package agent

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LicenseClient handles license activation
type LicenseClient struct {
	serverURL      string
	sn             string
	data           *ActivationData
	mu             sync.RWMutex
	log            func(string)
	analysisCount  int       // Today's analysis count
	analysisDate   string    // Date of analysis count (YYYY-MM-DD)
	dataDir        string    // Directory to store encrypted data
}

const (
	licenseEncryptionKey = "vantagedata-license-2024"
	licenseDataFile      = "license.dat"
)

// ActivationData contains the decrypted configuration from server
type ActivationData struct {
	LLMType       string `json:"llm_type"`
	LLMBaseURL    string `json:"llm_base_url"`
	LLMAPIKey     string `json:"llm_api_key"`
	LLMModel      string `json:"llm_model"`
	SearchType    string `json:"search_type"`
	SearchAPIKey  string `json:"search_api_key"`
	ExpiresAt     string `json:"expires_at"`
	DailyAnalysis int    `json:"daily_analysis"` // Daily analysis limit, 0 = unlimited
}

// ActivationResponse from server
type ActivationResponse struct {
	Success       bool   `json:"success"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	EncryptedData string `json:"encrypted_data,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

// ActivateResult contains the result of activation attempt
type ActivateResult struct {
	Success bool
	Code    string
	Message string
	Data    *ActivationData
}

// NewLicenseClient creates a new license client
func NewLicenseClient(log func(string)) *LicenseClient {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	dataDir := filepath.Join(configDir, "VantageData")
	
	return &LicenseClient{
		log:     log,
		dataDir: dataDir,
	}
}

// SetDataDir sets the data directory for storing encrypted license data
func (c *LicenseClient) SetDataDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dataDir = dir
}

// encryptData encrypts data using AES-GCM with SN as key
func (c *LicenseClient) encryptData(data []byte, sn string) (string, error) {
	hash := sha256.Sum256([]byte(sn + licenseEncryptionKey))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Activate attempts to activate with the given SN
func (c *LicenseClient) Activate(serverURL, sn string) (*ActivateResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.serverURL = serverURL
	c.sn = sn

	// Build request
	reqBody, _ := json.Marshal(map[string]string{"sn": sn})
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(serverURL+"/activate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "CONNECTION_FAILED",
			Message: fmt.Sprintf("连接服务器失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var result ActivationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("解析响应失败: %v", err),
		}, nil
	}

	if !result.Success {
		return &ActivateResult{
			Success: false,
			Code:    result.Code,
			Message: result.Message,
		}, nil
	}

	// Decrypt data
	decrypted, err := c.decryptData(result.EncryptedData, sn)
	if err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "DECRYPT_FAILED",
			Message: fmt.Sprintf("解密失败: %v", err),
		}, nil
	}

	var data ActivationData
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("解析配置失败: %v", err),
		}, nil
	}

	c.data = &data
	
	if c.log != nil {
		c.log(fmt.Sprintf("[LICENSE] Activation successful, expires: %s", data.ExpiresAt))
	}

	return &ActivateResult{
		Success: true,
		Code:    "SUCCESS",
		Message: "激活成功",
		Data:    &data,
	}, nil
}

// decryptData decrypts data using SN as key
func (c *LicenseClient) decryptData(encryptedData string, sn string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(sn))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// decryptLocalData decrypts locally stored data using SN + local key
func (c *LicenseClient) decryptLocalData(encryptedData string, sn string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(sn + licenseEncryptionKey))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// SaveActivationData saves the activation data encrypted to local file
func (c *LicenseClient) SaveActivationData() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil || c.sn == "" {
		return fmt.Errorf("no activation data to save")
	}

	// Create data directory if not exists
	if err := os.MkdirAll(c.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Prepare data to save (include SN for verification)
	saveData := struct {
		SN        string          `json:"sn"`
		Data      *ActivationData `json:"data"`
		SavedAt   string          `json:"saved_at"`
		ServerURL string          `json:"server_url"`
	}{
		SN:        c.sn,
		Data:      c.data,
		SavedAt:   time.Now().Format(time.RFC3339),
		ServerURL: c.serverURL,
	}

	jsonData, err := json.Marshal(saveData)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	// Encrypt
	encrypted, err := c.encryptData(jsonData, c.sn)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %v", err)
	}

	// Save to file
	filePath := filepath.Join(c.dataDir, licenseDataFile)
	if err := os.WriteFile(filePath, []byte(encrypted), 0600); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	if c.log != nil {
		c.log("[LICENSE] Activation data saved to local storage")
	}

	return nil
}

// LoadActivationData loads and decrypts activation data from local file
func (c *LicenseClient) LoadActivationData(sn string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := filepath.Join(c.dataDir, licenseDataFile)
	
	// Read file
	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no saved activation data")
		}
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Decrypt
	decrypted, err := c.decryptLocalData(string(encrypted), sn)
	if err != nil {
		return fmt.Errorf("failed to decrypt data (wrong SN?): %v", err)
	}

	// Parse
	var saveData struct {
		SN        string          `json:"sn"`
		Data      *ActivationData `json:"data"`
		SavedAt   string          `json:"saved_at"`
		ServerURL string          `json:"server_url"`
	}
	if err := json.Unmarshal(decrypted, &saveData); err != nil {
		return fmt.Errorf("failed to parse data: %v", err)
	}

	// Verify SN matches
	if saveData.SN != sn {
		return fmt.Errorf("SN mismatch")
	}

	// Check expiration
	if saveData.Data.ExpiresAt != "" {
		expiresAt, err := time.Parse("2006-01-02", saveData.Data.ExpiresAt)
		if err == nil && time.Now().After(expiresAt) {
			return fmt.Errorf("license expired on %s", saveData.Data.ExpiresAt)
		}
	}

	c.sn = sn
	c.data = saveData.Data
	c.serverURL = saveData.ServerURL

	if c.log != nil {
		c.log(fmt.Sprintf("[LICENSE] Loaded activation data from local storage, expires: %s", c.data.ExpiresAt))
	}

	return nil
}

// ClearSavedData removes the saved activation data file
func (c *LicenseClient) ClearSavedData() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := filepath.Join(c.dataDir, licenseDataFile)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	c.data = nil
	c.sn = ""
	c.serverURL = ""

	if c.log != nil {
		c.log("[LICENSE] Cleared saved activation data")
	}

	return nil
}

// GetData returns the activation data (nil if not activated)
func (c *LicenseClient) GetData() *ActivationData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// IsActivated returns true if license is activated
func (c *LicenseClient) IsActivated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data != nil
}

// Clear clears the activation data
func (c *LicenseClient) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = nil
	c.sn = ""
}

// RequestSNResponse from server
type RequestSNResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	SN      string `json:"sn,omitempty"`
	Code    string `json:"code,omitempty"`
}

// RequestSN requests a serial number from the license server
func (c *LicenseClient) RequestSN(serverURL, email string) (*RequestSNResponse, error) {
	// VantageData product_id is 0
	reqBody, _ := json.Marshal(map[string]interface{}{
		"email":      email,
		"product_id": 0,
	})
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(serverURL+"/request-sn", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("连接服务器失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var result RequestSNResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if c.log != nil {
		if result.Success {
			c.log(fmt.Sprintf("[LICENSE] SN requested successfully for email: %s", email))
		} else {
			c.log(fmt.Sprintf("[LICENSE] SN request failed: %s", result.Message))
		}
	}

	return &result, nil
}

// GetSN returns the current SN
func (c *LicenseClient) GetSN() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sn
}

// GetServerURL returns the current server URL
func (c *LicenseClient) GetServerURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverURL
}

// CanAnalyze checks if analysis is allowed (returns true if allowed, false if limit reached)
func (c *LicenseClient) CanAnalyze() (bool, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.data == nil {
		return true, "" // Not activated, allow (will use user's own config)
	}
	
	// 0 means unlimited
	if c.data.DailyAnalysis == 0 {
		return true, ""
	}
	
	today := time.Now().Format("2006-01-02")
	
	// Reset count if it's a new day
	if c.analysisDate != today {
		c.analysisDate = today
		c.analysisCount = 0
	}
	
	if c.analysisCount >= c.data.DailyAnalysis {
		return false, fmt.Sprintf("今日分析次数已达上限（%d次），请明天再试", c.data.DailyAnalysis)
	}
	
	return true, ""
}

// IncrementAnalysis increments the analysis count for today
func (c *LicenseClient) IncrementAnalysis() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	today := time.Now().Format("2006-01-02")
	
	// Reset count if it's a new day
	if c.analysisDate != today {
		c.analysisDate = today
		c.analysisCount = 0
	}
	
	c.analysisCount++
	
	if c.log != nil && c.data != nil && c.data.DailyAnalysis > 0 {
		c.log(fmt.Sprintf("[LICENSE] Analysis count: %d/%d", c.analysisCount, c.data.DailyAnalysis))
	}
}

// GetAnalysisStatus returns current analysis count and limit
func (c *LicenseClient) GetAnalysisStatus() (count int, limit int, date string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.data == nil {
		return 0, 0, ""
	}
	
	today := time.Now().Format("2006-01-02")
	if c.analysisDate != today {
		return 0, c.data.DailyAnalysis, today
	}
	
	return c.analysisCount, c.data.DailyAnalysis, c.analysisDate
}
