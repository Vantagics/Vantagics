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
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	"vantagedata/i18n"
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
	lastRefreshAt  time.Time     // Last time we refreshed from server
	lastReportAt       time.Time     // Last time we reported usage to server
	lastReportedCredits float64      // UsedCredits value at last successful report
	reportTicker       *time.Ticker  // Periodic usage reporting ticker
	reportStopCh       chan struct{} // Stop channel for usage reporting goroutine
}

const (
	licenseEncryptionKey = "vantagedata-license-2024"
	licenseDataFile      = "license.dat"
	CreditsPerAnalysis   = 1.5
)

// ActivationData contains the decrypted configuration from server
type ActivationData struct {
	LLMType         string                 `json:"llm_type"`
	LLMBaseURL      string                 `json:"llm_base_url"`
	LLMAPIKey       string                 `json:"llm_api_key"`
	LLMModel        string                 `json:"llm_model"`
	SearchType      string                 `json:"search_type"`
	SearchAPIKey    string                 `json:"search_api_key"`
	ExpiresAt       string                 `json:"expires_at"`
	ActivatedAt     string                 `json:"activated_at"`     // When the license was activated
	DailyAnalysis   int                    `json:"daily_analysis"`   // Daily analysis limit, 0 = unlimited
	ProductID       int                    `json:"product_id"`       // Product ID
	ProductName     string                 `json:"product_name"`     // Product name
	TrustLevel      string                 `json:"trust_level"`      // "high" (正式) or "low" (试用)
	RefreshInterval int                    `json:"refresh_interval"` // SN refresh interval in days (1=daily, 30=monthly)
	ExtraInfo       map[string]interface{} `json:"extra_info,omitempty"` // Product-specific extra info
	TotalCredits    float64                `json:"total_credits"`       // Total credits, 0 = unlimited in credits mode
	CreditsMode     bool                   `json:"credits_mode"`        // true = credits mode, false = daily limit mode
	UsedCredits     float64                `json:"used_credits"`        // Used credits
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
	reqBody, err := json.Marshal(map[string]string{"sn": sn})
	if err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "INTERNAL_ERROR",
			Message: i18n.T("license_client.build_request_failed", err),
		}, nil
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(serverURL+"/activate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "CONNECTION_FAILED",
			Message: i18n.T("license_client.connect_failed", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024)) // 1MB limit for activation response
	if err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "READ_ERROR",
			Message: i18n.T("license_client.read_response_failed", err),
		}, nil
	}
	
	var result ActivationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "PARSE_ERROR",
			Message: i18n.T("license_client.parse_response_failed", err),
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
			Message: i18n.T("license_client.decrypt_failed", err),
		}, nil
	}

	var data ActivationData
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return &ActivateResult{
			Success: false,
			Code:    "PARSE_ERROR",
			Message: i18n.T("license_client.parse_config_failed", err),
		}, nil
	}

	// Merge: take max of server and local used_credits
	if c.data != nil {
		data.UsedCredits = math.Max(data.UsedCredits, c.data.UsedCredits)
	}

	c.data = &data
	c.lastRefreshAt = time.Now() // Update last refresh time
	
	if c.log != nil {
		trustLabel := "试用版"
		if data.TrustLevel == "high" {
			trustLabel = "正式版"
		}
		c.log(fmt.Sprintf("[LICENSE] Activation successful, expires: %s, type: %s", data.ExpiresAt, trustLabel))
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
		SN            string          `json:"sn"`
		Data          *ActivationData `json:"data"`
		SavedAt       string          `json:"saved_at"`
		ServerURL     string          `json:"server_url"`
		LastRefreshAt string          `json:"last_refresh_at"` // Last time we refreshed from server
		AnalysisCount int             `json:"analysis_count"`  // Today's analysis count
		AnalysisDate  string          `json:"analysis_date"`   // Date of analysis count
		UsedCredits   float64         `json:"used_credits"`    // Used credits
		LastReportAt  string          `json:"last_report_at"`  // Last time we reported usage
		LastReportedCredits float64   `json:"last_reported_credits"` // UsedCredits at last report
	}{
		SN:            c.sn,
		Data:          c.data,
		SavedAt:       time.Now().Format(time.RFC3339),
		ServerURL:     c.serverURL,
		LastRefreshAt: time.Now().Format(time.RFC3339),
		AnalysisCount: c.analysisCount,
		AnalysisDate:  c.analysisDate,
		UsedCredits:   c.data.UsedCredits,
		LastReportAt:  c.lastReportAt.Format(time.RFC3339),
		LastReportedCredits: c.lastReportedCredits,
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
		SN            string          `json:"sn"`
		Data          *ActivationData `json:"data"`
		SavedAt       string          `json:"saved_at"`
		ServerURL     string          `json:"server_url"`
		LastRefreshAt string          `json:"last_refresh_at"`
		AnalysisCount int             `json:"analysis_count"`
		AnalysisDate  string          `json:"analysis_date"`
		UsedCredits   float64         `json:"used_credits"`
		LastReportAt  string          `json:"last_report_at"`
		LastReportedCredits float64   `json:"last_reported_credits"`
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
		expiresAt, err := time.Parse(time.RFC3339, saveData.Data.ExpiresAt)
		if err != nil {
			// Try date-only format
			expiresAt, err = time.Parse("2006-01-02", saveData.Data.ExpiresAt)
		}
		if err == nil && time.Now().After(expiresAt) {
			return fmt.Errorf("license expired on %s", saveData.Data.ExpiresAt)
		}
	}

	c.sn = sn
	c.data = saveData.Data
	c.serverURL = saveData.ServerURL

	// Restore used credits from save data
	c.data.UsedCredits = saveData.UsedCredits
	
	// Parse last refresh time
	if saveData.LastRefreshAt != "" {
		if t, err := time.Parse(time.RFC3339, saveData.LastRefreshAt); err == nil {
			c.lastRefreshAt = t
		}
	}

	// Parse last report time
	if saveData.LastReportAt != "" {
		if t, err := time.Parse(time.RFC3339, saveData.LastReportAt); err == nil {
			c.lastReportAt = t
		}
	}

	// Restore last reported credits value
	c.lastReportedCredits = saveData.LastReportedCredits

	// Restore analysis count (only if same day)
	today := time.Now().Format("2006-01-02")
	if saveData.AnalysisDate == today {
		c.analysisCount = saveData.AnalysisCount
		c.analysisDate = saveData.AnalysisDate
	} else {
		c.analysisCount = 0
		c.analysisDate = today
	}

	if c.log != nil {
		c.log(fmt.Sprintf("[LICENSE] Loaded activation data from local storage, expires: %s, trust_level: %s", c.data.ExpiresAt, c.data.TrustLevel))
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
	reqBody, err := json.Marshal(map[string]interface{}{
		"email":      email,
		"product_id": 0,
	})
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("license_client.build_request_failed", err))
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(serverURL+"/request-sn", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("license_client.connect_failed", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("license_client.read_response_failed", err))
	}
	
	var result RequestSNResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("%s", i18n.T("license_client.parse_response_failed", err))
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

	// Credits mode: check credits instead of daily limit
	if c.data.CreditsMode {
		// TotalCredits == 0 means unlimited in credits mode
		if c.data.TotalCredits == 0 {
			return true, ""
		}
		remaining := c.data.TotalCredits - c.data.UsedCredits
		if remaining < 0 {
			remaining = 0
		}
		if remaining < CreditsPerAnalysis {
			return false, i18n.T("license_client.credits_insufficient", remaining, CreditsPerAnalysis)
		}
		return true, ""
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
		return false, i18n.T("license_client.daily_limit_reached", c.data.DailyAnalysis)
	}
	
	return true, ""
}

// IncrementAnalysis increments the analysis count for today and persists it
func (c *LicenseClient) IncrementAnalysis() {
	c.mu.Lock()

	// Credits mode branch: deduct credits and return early (skip daily count logic)
	// Note: check c.data.CreditsMode directly instead of calling IsCreditsMode()
	// because we already hold c.mu.Lock() and IsCreditsMode() uses RLock (would deadlock)
	if c.data != nil && c.data.CreditsMode {
		// TotalCredits == 0 means unlimited, no need to deduct
		if c.data.TotalCredits > 0 {
			c.data.UsedCredits += CreditsPerAnalysis
			shouldReport := c.data.TrustLevel == "low"
			if c.log != nil {
				c.log(fmt.Sprintf("[LICENSE] Credits used: %.1f/%.1f", c.data.UsedCredits, c.data.TotalCredits))
			}
			c.mu.Unlock()

			// Persist updated credits to disk (outside lock to avoid deadlock with SaveActivationData)
			if err := c.SaveActivationData(); err != nil {
				if c.log != nil {
					c.log(fmt.Sprintf("[LICENSE] Failed to persist credits data: %v", err))
				}
			}

			// Report usage to server immediately after credits consumption (async, non-blocking)
			// Throttle: only report if at least 30 seconds since last report
			if shouldReport {
				c.mu.RLock()
				timeSinceLastReport := time.Since(c.lastReportAt)
				c.mu.RUnlock()
				if timeSinceLastReport >= 30*time.Second {
					go func() {
						defer func() {
							if r := recover(); r != nil {
								if c.log != nil {
									c.log(fmt.Sprintf("[LICENSE] Usage report goroutine recovered from panic: %v", r))
								}
							}
						}()
						if err := c.ReportUsage(); err != nil {
							if c.log != nil {
								c.log(fmt.Sprintf("[LICENSE] Immediate usage report after analysis failed: %v", err))
							}
						}
					}()
				}
			}
		} else {
			if c.log != nil {
				c.log("[LICENSE] Credits mode: unlimited (total_credits=0)")
			}
			c.mu.Unlock()
		}
		return
	}

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

	c.mu.Unlock()

	// Persist updated count to disk (outside lock to avoid deadlock with SaveActivationData)
	if err := c.SaveActivationData(); err != nil {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] Failed to persist analysis count: %v", err))
		}
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

// NeedsRefresh checks if the license needs to be refreshed based on trust level
// Returns true if refresh is needed, along with the reason
func (c *LicenseClient) NeedsRefresh() (bool, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return false, ""
	}

	// Get refresh interval (default to 1 day for safety)
	refreshInterval := c.data.RefreshInterval
	if refreshInterval <= 0 {
		// Default based on trust level
		if c.data.TrustLevel == "high" {
			refreshInterval = 30 // Monthly for high trust
		} else {
			refreshInterval = 1 // Daily for low trust
		}
	}

	// Check if refresh is needed
	if c.lastRefreshAt.IsZero() {
		return true, i18n.T("license_client.first_use")
	}

	daysSinceRefresh := int(time.Since(c.lastRefreshAt).Hours() / 24)
	if daysSinceRefresh >= refreshInterval {
		trustLabel := i18n.T("license_client.trial_label")
		if c.data.TrustLevel == "high" {
			trustLabel = i18n.T("license_client.official_label")
		}
		return true, i18n.T("license_client.refresh_needed", trustLabel, refreshInterval)
	}

	return false, ""
}

// RefreshIfNeeded checks if refresh is needed and performs it if necessary
// Returns (refreshed, error)
func (c *LicenseClient) RefreshIfNeeded() (bool, error) {
	needsRefresh, reason := c.NeedsRefresh()
	if !needsRefresh {
		return false, nil
	}

	if c.log != nil {
		c.log(fmt.Sprintf("[LICENSE] %s", reason))
	}

	// Get current SN and server URL
	c.mu.RLock()
	sn := c.sn
	serverURL := c.serverURL
	c.mu.RUnlock()

	if sn == "" || serverURL == "" {
		return false, fmt.Errorf("no SN or server URL configured")
	}

	// Attempt to refresh
	result, err := c.Activate(serverURL, sn)
	if err != nil {
		return false, err
	}

	if !result.Success {
		return false, fmt.Errorf("refresh failed: %s", result.Message)
	}

	// Save updated data
	if err := c.SaveActivationData(); err != nil {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] Warning: failed to save refreshed data: %v", err))
		}
	}

	if c.log != nil {
		c.log("[LICENSE] License refreshed successfully")
	}

	return true, nil
}

// GetTrustLevel returns the current trust level
func (c *LicenseClient) GetTrustLevel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return ""
	}
	return c.data.TrustLevel
}

// GetRefreshInterval returns the refresh interval in days
func (c *LicenseClient) GetRefreshInterval() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return 0
	}

	if c.data.RefreshInterval > 0 {
		return c.data.RefreshInterval
	}

	// Default based on trust level
	if c.data.TrustLevel == "high" {
		return 30
	}
	return 1
}

// IsCreditsMode returns true if the current license is in credits mode
func (c *LicenseClient) IsCreditsMode() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data != nil && c.data.CreditsMode
}

// GetCreditsStatus returns the credits status: totalCredits, usedCredits, isCreditsMode
func (c *LicenseClient) GetCreditsStatus() (float64, float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data == nil {
		return 0, 0, false
	}
	return c.data.TotalCredits, c.data.UsedCredits, c.data.CreditsMode
}

// GetLastRefreshAt returns the last refresh time
func (c *LicenseClient) GetLastRefreshAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRefreshAt
}

// GetLicenseStatus returns a summary of the license status
func (c *LicenseClient) GetLicenseStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]interface{})
	
	if c.data == nil {
		status["activated"] = false
		return status
	}

	status["activated"] = true
	status["sn"] = c.sn
	status["expires_at"] = c.data.ExpiresAt
	status["trust_level"] = c.data.TrustLevel
	status["refresh_interval"] = c.GetRefreshInterval()
	status["product_name"] = c.data.ProductName
	status["daily_analysis"] = c.data.DailyAnalysis

	if !c.lastRefreshAt.IsZero() {
		status["last_refresh_at"] = c.lastRefreshAt.Format(time.RFC3339)
		daysSinceRefresh := int(time.Since(c.lastRefreshAt).Hours() / 24)
		status["days_since_refresh"] = daysSinceRefresh
	}

	needsRefresh, reason := c.NeedsRefresh()
	status["needs_refresh"] = needsRefresh
	if needsRefresh {
		status["refresh_reason"] = reason
	}

	return status
}

// ReportUsage sends the current used_credits to the license server
func (c *LicenseClient) ReportUsage() error {
	c.mu.RLock()
	serverURL := c.serverURL
	sn := c.sn
	lastReport := c.lastReportAt
	var usedCredits float64
	if c.data != nil {
		usedCredits = c.data.UsedCredits
	}
	c.mu.RUnlock()

	if c.log != nil {
		c.log(fmt.Sprintf("[LICENSE] ReportUsage: serverURL=%s, sn=%s, usedCredits=%.1f", serverURL, sn, usedCredits))
	}

	if serverURL == "" || sn == "" {
		return fmt.Errorf("not activated, cannot report usage")
	}

	// 间隔守卫：非首次上报时，距上次上报不足1小时则静默跳过
	if !lastReport.IsZero() && time.Since(lastReport) < time.Hour {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] ReportUsage: skipped, only %v since last report", time.Since(lastReport)))
		}
		return nil
	}

	// Build request body
	reqBody, err := json.Marshal(map[string]interface{}{
		"sn":           sn,
		"used_credits": usedCredits,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal report request: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(serverURL+"/report-usage", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] Failed to report usage: %v", err))
		}
		return fmt.Errorf("failed to report usage: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024)) // 1MB limit
	if err != nil {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] Failed to read report response: %v", err))
		}
		return fmt.Errorf("failed to read report response: %v", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] Failed to parse report response: %v", err))
		}
		return fmt.Errorf("failed to parse report response: %v", err)
	}

	if !result.Success {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] Report usage failed: code=%s, message=%s", result.Code, result.Message))
		}
		return fmt.Errorf("report usage failed: %s", result.Code)
	}

	// Success: update lastReportAt, lastReportedCredits and persist
	c.mu.Lock()
	c.lastReportAt = time.Now()
	c.lastReportedCredits = usedCredits
	c.mu.Unlock()

	if err := c.SaveActivationData(); err != nil {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] Failed to save after report: %v", err))
		}
	}

	if c.log != nil {
		c.log(fmt.Sprintf("[LICENSE] Usage reported successfully: %.1f credits", usedCredits))
	}

	return nil
}

// ShouldReportOnStartup checks if a usage report is needed at startup.
// Returns true if:
// 1. Never reported before (lastReportAt is zero) AND used_credits > 0 (has consumption)
// 2. Last report was >= 1 hour ago AND used_credits has changed since last report
func (c *LicenseClient) ShouldReportOnStartup() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return false
	}

	usedCredits := c.data.UsedCredits

	// Case 1: Never reported, but has credits consumption
	if c.lastReportAt.IsZero() {
		if c.log != nil {
			c.log(fmt.Sprintf("[LICENSE] ShouldReportOnStartup: never reported, usedCredits=%.1f", usedCredits))
		}
		return usedCredits > 0
	}

	// Case 2: Over 1 hour since last report AND credits have changed
	elapsed := time.Since(c.lastReportAt)
	changed := usedCredits != c.lastReportedCredits
	if c.log != nil {
		c.log(fmt.Sprintf("[LICENSE] ShouldReportOnStartup: lastReportAt=%v, elapsed=%v, usedCredits=%.1f, lastReportedCredits=%.1f, changed=%v",
			c.lastReportAt.Format(time.RFC3339), elapsed, usedCredits, c.lastReportedCredits, changed))
	}
	return elapsed >= time.Hour && changed
}

// ShouldReportNow 检查当前是否满足上报条件（间隔 + 变化量）
func (c *LicenseClient) ShouldReportNow() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return false
	}

	// 首次上报且有消耗
	if c.lastReportAt.IsZero() {
		return c.data.UsedCredits > 0
	}

	// 间隔已满且有变化
	return time.Since(c.lastReportAt) >= time.Hour && c.data.UsedCredits != c.lastReportedCredits
}


// StartUsageReporting starts a background goroutine that reports usage every hour
func (c *LicenseClient) StartUsageReporting() {
	c.mu.Lock()
	// Don't start if already running
	if c.reportTicker != nil {
		c.mu.Unlock()
		return
	}
	c.reportTicker = time.NewTicker(1 * time.Hour)
	c.reportStopCh = make(chan struct{})
	ticker := c.reportTicker
	stopCh := c.reportStopCh
	c.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if c.log != nil {
					c.log(fmt.Sprintf("[LICENSE] Usage reporting goroutine recovered from panic: %v", r))
				}
			}
		}()
		for {
			select {
			case <-ticker.C:
				// Only report if in credits mode, trial (low trust), and credits have changed
				if c.IsCreditsMode() && c.GetTrustLevel() == "low" {
					c.mu.RLock()
					hasChange := c.data != nil && c.data.UsedCredits != c.lastReportedCredits
					c.mu.RUnlock()
					if hasChange {
						if err := c.ReportUsage(); err != nil {
							if c.log != nil {
								c.log(fmt.Sprintf("[LICENSE] Periodic usage report failed: %v", err))
							}
						}
					}
				}
			case <-stopCh:
				return
			}
		}
	}()

	if c.log != nil {
		c.log("[LICENSE] Started periodic usage reporting (1 hour interval)")
	}
}

// StopUsageReporting stops the background usage reporting goroutine
func (c *LicenseClient) StopUsageReporting() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.reportTicker != nil {
		c.reportTicker.Stop()
		c.reportTicker = nil
	}
	if c.reportStopCh != nil {
		close(c.reportStopCh)
		c.reportStopCh = nil
	}

	if c.log != nil {
		c.log("[LICENSE] Stopped periodic usage reporting")
	}
}

// GetLastReportAt returns the last report time
func (c *LicenseClient) GetLastReportAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastReportAt
}

