package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DefaultMarketplaceServerURL is the default marketplace server address.
const DefaultMarketplaceServerURL = "https://market.vantagedata.chat"

// DefaultLicenseServerURL is the default license server address.
const DefaultLicenseServerURL = "https://license.vantagedata.chat"

// maxErrorBodySize limits how much of an error response body we read to prevent memory exhaustion.
const maxErrorBodySize = 4096

// readErrorBody reads a limited amount of the response body for error messages.
func readErrorBody(body io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(body, maxErrorBodySize))
	return string(data)
}

// MarketplaceClient is the HTTP client for communicating with the marketplace server.
type MarketplaceClient struct {
	ServerURL        string
	LicenseServerURL string
	Token            string // JWT token
	client           *http.Client
}

// NewMarketplaceClient creates a new MarketplaceClient with the given server URL.
func NewMarketplaceClient(serverURL string) *MarketplaceClient {
	if serverURL == "" {
		serverURL = DefaultMarketplaceServerURL
	}
	return &MarketplaceClient{
		ServerURL:        serverURL,
		LicenseServerURL: DefaultLicenseServerURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// PackCategory represents a marketplace pack category.
type PackCategory struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPreset    bool   `json:"is_preset"`
	PackCount   int    `json:"pack_count"`
}

// PackListingInfo represents a marketplace pack listing (without file data).
type PackListingInfo struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	CategoryID      int64  `json:"category_id"`
	CategoryName    string `json:"category_name"`
	PackName        string `json:"pack_name"`
	PackDescription string `json:"pack_description"`
	SourceName      string `json:"source_name"`
	AuthorName      string `json:"author_name"`
	ShareMode       string `json:"share_mode"`
	CreditsPrice    int    `json:"credits_price"`
	DownloadCount   int    `json:"download_count"`
	CreatedAt       string `json:"created_at"`
	Purchased       bool   `json:"purchased"`
}

// NotificationInfo represents a marketplace notification message.
type NotificationInfo struct {
	ID                  int64  `json:"id"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	TargetType          string `json:"target_type"`
	EffectiveDate       string `json:"effective_date"`
	DisplayDurationDays int    `json:"display_duration_days"`
	CreatedAt           string `json:"created_at"`
}

// ensureMarketplaceClient initializes the marketplace client on the App if not already set.
func (a *App) ensureMarketplaceClient() {
	if a.marketplaceClient == nil {
		a.marketplaceClient = NewMarketplaceClient(DefaultMarketplaceServerURL)
	}
}

// MarketplaceLoginWithSN performs automatic login using the locally saved SN and Email.
// Two-step flow: first get a license_token from the License server, then exchange it for a market JWT.
func (a *App) MarketplaceLoginWithSN() error {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	// Step 1: Get SN and Email from the app's config/license info
	if a.licenseClient == nil || !a.licenseClient.IsActivated() {
		return fmt.Errorf("license not activated, cannot login to marketplace")
	}
	sn := a.licenseClient.GetSN()
	if sn == "" {
		return fmt.Errorf("SN not available, cannot login to marketplace")
	}

	cfg, err := a.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	email := cfg.LicenseEmail
	if email == "" {
		return fmt.Errorf("email not available, cannot login to marketplace")
	}

	// Step 2: POST to License server /api/marketplace-auth with {sn, email}
	authPayload, _ := json.Marshal(map[string]string{
		"sn":    sn,
		"email": email,
	})
	authResp, err := mc.client.Post(
		mc.LicenseServerURL+"/api/marketplace-auth",
		"application/json",
		bytes.NewReader(authPayload),
	)
	if err != nil {
		return fmt.Errorf("failed to contact license server: %w", err)
	}
	defer authResp.Body.Close()

	// Check response status first
	if authResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(authResp.Body)
		bodyStr := string(bodyBytes)
		// If body looks like HTML (starts with <), provide a clearer error
		if len(bodyStr) > 0 && bodyStr[0] == '<' {
			return fmt.Errorf("license server returned HTML instead of JSON (status %d). The license server may be unavailable or misconfigured at %s", authResp.StatusCode, mc.LicenseServerURL)
		}
		return fmt.Errorf("license server returned status %d: %s", authResp.StatusCode, bodyStr)
	}

	// Read response body first to provide better error messages
	bodyBytes, readErr := io.ReadAll(authResp.Body)
	if readErr != nil {
		return fmt.Errorf("failed to read license server response: %w", readErr)
	}

	var authResult struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &authResult); err != nil {
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 0 && bodyStr[0] == '<' {
			return fmt.Errorf("license server returned HTML instead of JSON. The license server may be unavailable or misconfigured at %s. Response preview: %.200s", mc.LicenseServerURL, bodyStr)
		}
		return fmt.Errorf("failed to decode license server response: %w. Response body: %s", err, bodyStr)
	}
	if !authResult.Success {
		return fmt.Errorf("license authentication failed: %s (%s)", authResult.Message, authResult.Code)
	}

	// Step 3: POST to Market server /api/auth/sn-login with {license_token}
	loginPayload, _ := json.Marshal(map[string]string{
		"license_token": authResult.Token,
	})
	loginResp, err := mc.client.Post(
		mc.ServerURL+"/api/auth/sn-login",
		"application/json",
		bytes.NewReader(loginPayload),
	)
	if err != nil {
		return fmt.Errorf("failed to contact marketplace server: %w", err)
	}
	defer loginResp.Body.Close()

	var loginResult struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		return fmt.Errorf("failed to decode marketplace login response: %w", err)
	}
	if !loginResult.Success {
		return fmt.Errorf("marketplace login failed: %s", loginResult.Message)
	}

	// Step 4: Store JWT in marketplaceClient.Token
	mc.Token = loginResult.Token
	return nil
}

// EnsureMarketplaceAuth checks if the marketplace token is valid, and if not, performs automatic SN+Email login.
func (a *App) EnsureMarketplaceAuth() error {
	a.ensureMarketplaceClient()
	if a.marketplaceClient.Token != "" {
		return nil
	}
	return a.MarketplaceLoginWithSN()
}

// IsMarketplaceLoggedIn returns true if the marketplace client has a valid token.
func (a *App) IsMarketplaceLoggedIn() bool {
	if a.marketplaceClient == nil {
		return false
	}
	return a.marketplaceClient.Token != ""
}

// MarketplacePortalLogin performs SSO login to the marketplace user portal.
// It uses the same SN+Email flow as MarketplaceLoginWithSN, but returns a
// ticket-login URL for opening in the browser (like ServicePortalLogin).
func (a *App) MarketplacePortalLogin() (string, error) {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	// Check license activation
	if a.licenseClient == nil || !a.licenseClient.IsActivated() {
		return "", fmt.Errorf("license not activated")
	}
	sn := a.licenseClient.GetSN()
	if sn == "" {
		return "", fmt.Errorf("SN not available")
	}

	cfg, err := a.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	email := cfg.LicenseEmail
	if email == "" {
		return "", fmt.Errorf("email not available")
	}

	// Step 1: POST to License Server /api/marketplace-auth to get auth_token
	authPayload, _ := json.Marshal(map[string]string{
		"sn":    sn,
		"email": email,
	})
	authResp, err := mc.client.Post(
		mc.LicenseServerURL+"/api/marketplace-auth",
		"application/json",
		bytes.NewReader(authPayload),
	)
	if err != nil {
		return "", fmt.Errorf("failed to contact license server: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(authResp.Body)
		return "", fmt.Errorf("license server returned status %d: %s", authResp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(authResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read license server response: %w", err)
	}

	var authResult struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &authResult); err != nil {
		return "", fmt.Errorf("failed to decode license server response: %w", err)
	}
	if !authResult.Success {
		return "", fmt.Errorf("license authentication failed: %s (%s)", authResult.Message, authResult.Code)
	}

	// Step 2: POST to Marketplace Server /api/auth/sn-login to get login_ticket
	loginPayload, _ := json.Marshal(map[string]string{
		"license_token": authResult.Token,
	})
	loginResp, err := mc.client.Post(
		mc.ServerURL+"/api/auth/sn-login",
		"application/json",
		bytes.NewReader(loginPayload),
	)
	if err != nil {
		return "", fmt.Errorf("failed to contact marketplace server: %w", err)
	}
	defer loginResp.Body.Close()

	loginBodyBytes, err := io.ReadAll(loginResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read marketplace response: %w", err)
	}

	var loginResult struct {
		Success     bool   `json:"success"`
		Token       string `json:"token"`
		LoginTicket string `json:"login_ticket"`
		Message     string `json:"message"`
	}
	if err := json.Unmarshal(loginBodyBytes, &loginResult); err != nil {
		return "", fmt.Errorf("failed to decode marketplace response: %w", err)
	}
	if !loginResult.Success {
		return "", fmt.Errorf("marketplace login failed: %s", loginResult.Message)
	}

	// Also store the JWT for API calls
	if loginResult.Token != "" {
		mc.Token = loginResult.Token
	}

	// Step 3: Construct ticket-login URL
	return mc.ServerURL + "/user/ticket-login?ticket=" + loginResult.LoginTicket, nil
}

// GetMarketplaceCategories fetches the list of pack categories from the marketplace server.
func (a *App) GetMarketplaceCategories() ([]PackCategory, error) {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	resp, err := mc.client.Get(mc.ServerURL + "/api/categories")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		Categories []PackCategory `json:"categories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode categories response: %w", err)
	}
	return result.Categories, nil
}

// SharePackToMarketplace uploads a .qap file to the marketplace server.
func (a *App) SharePackToMarketplace(packFilePath string, categoryID int64, pricingModel string, creditsPrice int, detailedDescription string) error {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	// Validate file exists and check size before reading
	fileInfo, err := os.Stat(packFilePath)
	if err != nil {
		return fmt.Errorf("failed to access pack file: %w", err)
	}
	const maxUploadSize = 500 * 1024 * 1024 // 500MB limit
	if fileInfo.Size() > maxUploadSize {
		return fmt.Errorf("pack file too large (%dMB), maximum is %dMB", fileInfo.Size()/1024/1024, maxUploadSize/1024/1024)
	}

	// Open the file for streaming (avoid reading entire file into memory)
	file, err := os.Open(packFilePath)
	if err != nil {
		return fmt.Errorf("failed to open pack file: %w", err)
	}
	defer file.Close()

	// Build multipart form request using pipe to avoid buffering entire file
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write multipart form in a goroutine
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()

		// Add the file field
		part, err := writer.CreateFormFile("file", filepath.Base(packFilePath))
		if err != nil {
			errCh <- fmt.Errorf("failed to create form file: %w", err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errCh <- fmt.Errorf("failed to write file data: %w", err)
			return
		}

		// Add form fields
		writer.WriteField("category_id", fmt.Sprintf("%d", categoryID))
		writer.WriteField("share_mode", pricingModel)
		writer.WriteField("credits_price", fmt.Sprintf("%d", creditsPrice))
		writer.WriteField("detailed_description", detailedDescription)

		if err := writer.Close(); err != nil {
			errCh <- fmt.Errorf("failed to close multipart writer: %w", err)
			return
		}
		errCh <- nil
	}()

	req, err := http.NewRequest("POST", mc.ServerURL+"/api/packs/upload", pr)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload pack: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors from the multipart writer goroutine
	if writeErr := <-errCh; writeErr != nil {
		return writeErr
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	return nil
}

// BrowseMarketplacePacks fetches pack listings from the marketplace, optionally filtered by category.
func (a *App) BrowseMarketplacePacks(categoryID int64) ([]PackListingInfo, error) {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	url := mc.ServerURL + "/api/packs"
	if categoryID > 0 {
		url = fmt.Sprintf("%s?category_id=%d", url, categoryID)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if mc.Token != "" {
		req.Header.Set("Authorization", "Bearer "+mc.Token)
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to browse packs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		Packs []PackListingInfo `json:"packs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode packs response: %w", err)
	}
	return result.Packs, nil
}

// GetMySharedPackNames returns the pack names that the current user has shared to the marketplace.
// It decodes the JWT token to extract user_id, fetches all marketplace listings,
// and returns pack names where the listing's user_id matches the current user.
func (a *App) GetMySharedPackNames() ([]string, error) {
	// Ensure auth so the token is available even if MarketBrowsePage was never opened.
	if err := a.EnsureMarketplaceAuth(); err != nil {
		// Auth is best-effort; if it fails, return empty list without blocking.
		return []string{}, nil
	}
	mc := a.marketplaceClient

	if mc.Token == "" {
		return []string{}, nil
	}

	// Decode JWT payload to get user_id (JWT format: header.payload.signature)
	parts := bytes.SplitN([]byte(mc.Token), []byte("."), 3)
	if len(parts) < 2 {
		return []string{}, nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(string(parts[1]))
	if err != nil {
		return []string{}, nil
	}

	var claims struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil || claims.UserID == 0 {
		return []string{}, nil
	}

	// Fetch all marketplace listings
	allPacks, err := a.BrowseMarketplacePacks(0)
	if err != nil {
		return []string{}, nil
	}

	// Filter by current user's ID
	var names []string
	for _, p := range allPacks {
		if p.UserID == claims.UserID {
			names = append(names, p.PackName)
		}
	}
	if names == nil {
		names = []string{}
	}
	return names, nil
}

// MyPublishedPackInfo represents a published pack owned by the current user (for replacement selection).
type MyPublishedPackInfo struct {
	ListingID   int64  `json:"id"`
	PackName    string `json:"pack_name"`
	SourceName  string `json:"source_name"`
	Version     int    `json:"version"`
}

// GetMyPublishedPacks returns the current user's published packs, optionally filtered by source_name.
func (a *App) GetMyPublishedPacks(sourceName string) ([]MyPublishedPackInfo, error) {
	if err := a.EnsureMarketplaceAuth(); err != nil {
		return []MyPublishedPackInfo{}, nil
	}
	mc := a.marketplaceClient
	if mc.Token == "" {
		return []MyPublishedPackInfo{}, nil
	}

	// Decode JWT to get user_id
	parts := bytes.SplitN([]byte(mc.Token), []byte("."), 3)
	if len(parts) < 2 {
		return []MyPublishedPackInfo{}, nil
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(string(parts[1]))
	if err != nil {
		return []MyPublishedPackInfo{}, nil
	}
	var claims struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil || claims.UserID == 0 {
		return []MyPublishedPackInfo{}, nil
	}

	// Fetch all published packs
	allPacks, err := a.BrowseMarketplacePacks(0)
	if err != nil {
		return []MyPublishedPackInfo{}, nil
	}

	var result []MyPublishedPackInfo
	for _, p := range allPacks {
		if p.UserID == claims.UserID {
			if sourceName != "" && p.SourceName != sourceName {
				continue
			}
			result = append(result, MyPublishedPackInfo{
				ListingID:  p.ID,
				PackName:   p.PackName,
				SourceName: p.SourceName,
				Version:    0, // version not available from list API
			})
		}
	}
	if result == nil {
		result = []MyPublishedPackInfo{}
	}
	return result, nil
}

// ReplaceMarketplacePack replaces a published pack's content with a local pack file.
// The server will bump the version and reset status to pending for re-review.
func (a *App) ReplaceMarketplacePack(localPackPath string, listingID int64) error {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	fileInfo, err := os.Stat(localPackPath)
	if err != nil {
		return fmt.Errorf("failed to access pack file: %w", err)
	}
	const maxUploadSize = 500 * 1024 * 1024
	if fileInfo.Size() > maxUploadSize {
		return fmt.Errorf("pack file too large (%dMB), maximum is %dMB", fileInfo.Size()/1024/1024, maxUploadSize/1024/1024)
	}

	file, err := os.Open(localPackPath)
	if err != nil {
		return fmt.Errorf("failed to open pack file: %w", err)
	}
	defer file.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("file", filepath.Base(localPackPath))
		if err != nil {
			errCh <- fmt.Errorf("failed to create form file: %w", err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errCh <- fmt.Errorf("failed to write file data: %w", err)
			return
		}
		writer.WriteField("listing_id", fmt.Sprintf("%d", listingID))
		if err := writer.Close(); err != nil {
			errCh <- fmt.Errorf("failed to close multipart writer: %w", err)
			return
		}
		errCh <- nil
	}()

	req, err := http.NewRequest("POST", mc.ServerURL+"/api/packs/replace", pr)
	if err != nil {
		return fmt.Errorf("failed to create replace request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to replace pack: %w", err)
	}
	defer resp.Body.Close()

	if writeErr := <-errCh; writeErr != nil {
		return writeErr
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("replace failed with status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	return nil
}

// PurchasedPackInfo represents a purchased pack from the marketplace.
type PurchasedPackInfo struct {
	ListingID       int64  `json:"listing_id"`
	PackName        string `json:"pack_name"`
	PackDescription string `json:"pack_description"`
	SourceName      string `json:"source_name"`
	AuthorName      string `json:"author_name"`
	ShareMode       string `json:"share_mode"`
	CreditsPrice    int    `json:"credits_price"`
	CreatedAt       string `json:"created_at"`
}

// GetMyPurchasedPacks fetches the current user's purchased packs from the marketplace.
// Returns empty slice (not error) on auth or network failure to avoid blocking UI.
func (a *App) GetMyPurchasedPacks() []PurchasedPackInfo {
	if err := a.EnsureMarketplaceAuth(); err != nil {
		return []PurchasedPackInfo{}
	}
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return []PurchasedPackInfo{}
	}

	req, err := http.NewRequest("GET", mc.ServerURL+"/api/packs/purchased", nil)
	if err != nil {
		return []PurchasedPackInfo{}
	}
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return []PurchasedPackInfo{}
	}
	defer resp.Body.Close()

	var result struct {
		Packs []PurchasedPackInfo `json:"packs"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Packs == nil {
		result.Packs = []PurchasedPackInfo{}
	}
	return result.Packs
}


// DownloadMarketplacePack downloads a pack from the marketplace and saves it to a temp directory.
// Returns the local file path of the downloaded .qap file.
func (a *App) DownloadMarketplacePack(listingID int64) (string, error) {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return "", fmt.Errorf("not logged in to marketplace")
	}

	url := fmt.Sprintf("%s/api/packs/%d/download", mc.ServerURL, listingID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download pack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	// Save to temp directory with size limit, streaming to disk to avoid memory exhaustion
	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("marketplace_pack_%d.qap", listingID)
	filePath := filepath.Join(tmpDir, fileName)

	const maxDownloadSize int64 = 500 * 1024 * 1024 // 500MB limit
	limitedReader := io.LimitReader(resp.Body, maxDownloadSize+1)

	outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	written, err := io.Copy(outFile, limitedReader)
	outFile.Close()
	if err != nil {
		os.Remove(filePath) // Clean up partial file
		return "", fmt.Errorf("failed to write download to disk: %w", err)
	}
	if written > maxDownloadSize {
		os.Remove(filePath) // Clean up oversized file
		return "", fmt.Errorf("downloaded file exceeds maximum size limit (%dMB)", maxDownloadSize/1024/1024)
	}

	// Move downloaded file from temp directory to {DataCacheDir}/qap/
	cfg, err := a.GetConfig()
	if err != nil {
		// Cannot get config; return temp path as fallback (Requirement 2.4)
		return filePath, fmt.Errorf("downloaded to temp but failed to get config for QAP directory: %w", err)
	}
	qapDir := filepath.Join(cfg.DataCacheDir, "qap")
	if err := os.MkdirAll(qapDir, 0755); err != nil {
		// Cannot create QAP directory; return temp path (Requirement 2.4)
		return filePath, fmt.Errorf("downloaded to temp but failed to create QAP directory: %w", err)
	}
	finalPath := filepath.Join(qapDir, fileName)

	// Clean up old timestamped files for the same listing ID (legacy format: marketplace_pack_{id}_{timestamp}.qap)
	oldPattern := fmt.Sprintf("marketplace_pack_%d_*.qap", listingID)
	if matches, err := filepath.Glob(filepath.Join(qapDir, oldPattern)); err == nil {
		for _, oldFile := range matches {
			os.Remove(oldFile)
		}
	}

	// Try os.Rename first; fallback to copy+delete for cross-filesystem moves
	if err := os.Rename(filePath, finalPath); err != nil {
		if copyErr := copyFile(filePath, finalPath); copyErr != nil {
			// Copy failed; preserve temp file (Requirement 2.4)
			return filePath, fmt.Errorf("downloaded to temp but failed to move to QAP directory: %w", copyErr)
		}
		// Copy succeeded; clean up temp file (Requirement 2.2)
		os.Remove(filePath)
	}

	// Extract encryption password from response header (for paid packs)
	if encPassword := resp.Header.Get("X-Encryption-Password"); encPassword != "" {
		if a.packPasswords == nil {
			a.packPasswords = make(map[string]string)
		}
		a.packPasswords[finalPath] = encPassword
		// Persist password to disk so it survives app restarts
		if a.packPasswordStore != nil {
			a.packPasswordStore.SetPassword(finalPath, encPassword)
			_ = a.packPasswordStore.Save()
		}
	}

	// Parse X-Usage-License header if present and save to local store
	if licenseHeader := resp.Header.Get("X-Usage-License"); licenseHeader != "" {
		var usageLicense UsageLicense
		if err := json.Unmarshal([]byte(licenseHeader), &usageLicense); err == nil {
			if a.usageLicenseStore != nil {
				a.usageLicenseStore.SetLicense(&usageLicense)
				_ = a.usageLicenseStore.Save()
			}
		}
	}

	return finalPath, nil
}

// copyFile copies src to dst, used as fallback when os.Rename fails across filesystems.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// GetMarketplaceCreditsBalance fetches the current user's credits balance from the marketplace.
func (a *App) GetMarketplaceCreditsBalance() (float64, error) {
	if err := a.EnsureMarketplaceAuth(); err != nil {
		return 0, fmt.Errorf("marketplace auth failed: %w", err)
	}
	mc := a.marketplaceClient

	req, err := http.NewRequest("GET", mc.ServerURL+"/api/credits/balance", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create balance request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch credits balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		Balance float64 `json:"credits_balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode balance response: %w", err)
	}
	return result.Balance, nil
}

// PurchaseAdditionalUses purchases additional uses for a per_use pack listing.
// It POSTs to /api/packs/{id}/purchase-uses and updates the local UsageLicenseStore.
func (a *App) PurchaseAdditionalUses(listingID int64, quantity int) error {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	payload, _ := json.Marshal(map[string]int{"quantity": quantity})
	url := fmt.Sprintf("%s/api/packs/%d/purchase-uses", mc.ServerURL, listingID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create purchase request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to purchase additional uses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("purchase failed with status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		RemainingUses int `json:"remaining_uses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode purchase response: %w", err)
	}

	// Update local UsageLicenseStore
	if a.usageLicenseStore != nil {
		lic := a.usageLicenseStore.GetLicense(listingID)
		if lic != nil {
			lic.RemainingUses += result.RemainingUses
			lic.TotalUses += quantity
			lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			a.usageLicenseStore.SetLicense(lic)
			_ = a.usageLicenseStore.Save()
		}
	}

	return nil
}

// RenewSubscription renews a subscription pack listing.
// It POSTs to /api/packs/{id}/renew and updates the local UsageLicenseStore.
// months: 1 for monthly, 12 for yearly (pays 12 months, gets 14 months).
func (a *App) RenewSubscription(listingID int64, months int) error {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	payload, _ := json.Marshal(map[string]int{"months": months})
	url := fmt.Sprintf("%s/api/packs/%d/renew", mc.ServerURL, listingID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create renew request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to renew subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("renew failed with status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		ExpiresAt          string `json:"expires_at"`
		SubscriptionMonths int    `json:"subscription_months"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode renew response: %w", err)
	}

	// Update local UsageLicenseStore
	if a.usageLicenseStore != nil {
		lic := a.usageLicenseStore.GetLicense(listingID)
		if lic != nil {
			lic.ExpiresAt = result.ExpiresAt
			lic.SubscriptionMonths = result.SubscriptionMonths
			lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			a.usageLicenseStore.SetLicense(lic)
			_ = a.usageLicenseStore.Save()
		}
	}

	return nil
}

// GetUsageLicenses returns all local usage licenses for display in the PackManagerPage.
func (a *App) GetUsageLicenses() []*UsageLicense {
	if a.usageLicenseStore == nil {
		return nil
	}
	return a.usageLicenseStore.GetAllLicenses()
}

// RefreshPurchasedPackLicenses fetches the latest usage license info from the marketplace server
// and updates the local UsageLicenseStore. This ensures the local license data is in sync with
// the server's authoritative records (e.g., after purchases made on another device or via web).
func (a *App) RefreshPurchasedPackLicenses() error {
	if err := a.EnsureMarketplaceAuth(); err != nil {
		return fmt.Errorf("marketplace auth failed: %w", err)
	}
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	req, err := http.NewRequest("GET", mc.ServerURL+"/api/packs/my-licenses", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch licenses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		Licenses []UsageLicense `json:"licenses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode licenses response: %w", err)
	}

	if a.usageLicenseStore == nil {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, lic := range result.Licenses {
		lic.UpdatedAt = now
		if lic.CreatedAt == "" {
			lic.CreatedAt = now
		}
		// For subscription/time_limited packs, update Blocked flag based on server expiry
		if lic.PricingModel == "subscription" || lic.PricingModel == "time_limited" {
			if expiresAt, err := time.Parse(time.RFC3339, lic.ExpiresAt); err == nil {
				lic.Blocked = !time.Now().Before(expiresAt)
			} else {
				lic.Blocked = false // Cannot parse — be permissive
			}
		}
		a.usageLicenseStore.SetLicense(&lic)
	}
	return a.usageLicenseStore.Save()
}

// ReportPackUsageResponse 服务器上报响应
type ReportPackUsageResponse struct {
	Success        bool `json:"success"`
	UsedCount      int  `json:"used_count"`
	TotalPurchased int  `json:"total_purchased"`
	RemainingUses  int  `json:"remaining_uses"`
	Exhausted      bool `json:"exhausted"`
}

// FlushPendingUsageReports processes all pending usage records in the queue.
// For each record, it dequeues first then calls ReportPackUsage. If ReportPackUsage
// fails, it will automatically re-enqueue the record. After processing all records,
// the queue is persisted to disk via Save.
// Uses flushUsageMu to prevent concurrent execution which could cause duplicate reports.
func (a *App) FlushPendingUsageReports() {
	if a.pendingUsageQueue == nil {
		return
	}

	// Prevent concurrent flush calls from causing duplicate reports
	a.flushUsageMu.Lock()
	defer a.flushUsageMu.Unlock()

	records := a.pendingUsageQueue.GetAll()
	for _, rec := range records {
		// Dequeue before calling ReportPackUsage to avoid duplicates.
		// If ReportPackUsage fails, it will re-enqueue automatically.
		_ = a.pendingUsageQueue.Dequeue(rec.ListingID, rec.UsedAt)

		_, err := a.ReportPackUsage(rec.ListingID, rec.UsedAt)
		if err != nil {
			fmt.Printf("[FlushPendingUsageReports] failed to report listing %d at %s: %v\n", rec.ListingID, rec.UsedAt, err)
		}
	}

	_ = a.pendingUsageQueue.Save()
}


// ReportPackUsage reports a per_use pack usage to the marketplace server.
// On success, it updates the local UsageLicenseStore with the server's used_count.
// On network error or non-2xx (except 400), it enqueues the record to PendingUsageQueue.
// On HTTP 400 (invalid request), it logs a warning but does NOT enqueue.
func (a *App) ReportPackUsage(listingID int64, usedAt string) (*ReportPackUsageResponse, error) {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		// No JWT token — enqueue for later retry
		if a.pendingUsageQueue != nil {
			_ = a.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("not logged in to marketplace")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"listing_id": listingID,
		"used_at":    usedAt,
	})
	url := fmt.Sprintf("%s/api/packs/report-usage", mc.ServerURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		if a.pendingUsageQueue != nil {
			_ = a.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("failed to create report-usage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		// Network error — enqueue for retry
		if a.pendingUsageQueue != nil {
			_ = a.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("failed to report pack usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		// HTTP 400: invalid request — log but do NOT enqueue (retrying won't help)
		errMsg := readErrorBody(resp.Body)
		fmt.Printf("[ReportPackUsage] warning: server returned 400 for listing %d: %s\n", listingID, errMsg)
		return nil, fmt.Errorf("invalid usage report (400): %s", errMsg)
	}

	if resp.StatusCode != http.StatusOK {
		// Non-2xx (5xx, etc.) — enqueue for retry
		errMsg := readErrorBody(resp.Body)
		if a.pendingUsageQueue != nil {
			_ = a.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("report-usage failed with status %d: %s", resp.StatusCode, errMsg)
	}

	// Success — parse response
	var result ReportPackUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode report-usage response: %w", err)
	}

	// Update local UsageLicenseStore with server's authoritative counts
	if a.usageLicenseStore != nil {
		lic := a.usageLicenseStore.GetLicense(listingID)
		if lic != nil && lic.PricingModel == "per_use" {
			// Sync TotalUses from server (handles purchases from other devices/web)
			if result.TotalPurchased > 0 {
				lic.TotalUses = result.TotalPurchased
			}
			lic.RemainingUses = result.RemainingUses
			if lic.RemainingUses < 0 {
				lic.RemainingUses = 0
			}
			lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			a.usageLicenseStore.SetLicense(lic)
			_ = a.usageLicenseStore.Save()
			a.Log(fmt.Sprintf("[REPORT-USAGE] Synced license for listing %d: remaining=%d, total=%d, used=%d, exhausted=%v",
				listingID, lic.RemainingUses, lic.TotalUses, result.UsedCount, result.Exhausted))
		}
	}

	return &result, nil
}

// ValidateSubscriptionLicenseAsync refreshes licenses from the server in the background
// and marks the subscription as blocked if the server confirms it has expired.
// This is called after optimistic pack execution for subscription-type packs.
func (a *App) ValidateSubscriptionLicenseAsync(listingID int64) {
	go func() {
		a.Log(fmt.Sprintf("[LICENSE-VALIDATE] Starting async subscription validation for listing %d", listingID))
		err := a.RefreshPurchasedPackLicenses()
		if err != nil {
			a.Log(fmt.Sprintf("[LICENSE-VALIDATE] Failed to refresh licenses from server: %v", err))
			// Network error — don't block the user; next time we'll try again
			return
		}
		if a.usageLicenseStore == nil {
			return
		}
		lic := a.usageLicenseStore.GetLicense(listingID)
		if lic == nil {
			return
		}
		if lic.Blocked {
			a.Log(fmt.Sprintf("[LICENSE-VALIDATE] Listing %d subscription confirmed expired by server, marked as blocked", listingID))
		} else {
			a.Log(fmt.Sprintf("[LICENSE-VALIDATE] Listing %d subscription still valid", listingID))
		}
	}()
}

// GetMarketplaceNotifications fetches active notifications from the marketplace server.
// If the user is logged in (has JWT token), the token is included so the server can
// return both broadcast and targeted notifications. Otherwise only broadcasts are returned.
func (a *App) GetMarketplaceNotifications() ([]NotificationInfo, error) {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	req, err := http.NewRequest("GET", mc.ServerURL+"/api/notifications", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create notifications request: %w", err)
	}
	if mc.Token != "" {
		req.Header.Set("Authorization", "Bearer "+mc.Token)
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		Notifications []NotificationInfo `json:"notifications"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode notifications response: %w", err)
	}

	if result.Notifications == nil {
		return []NotificationInfo{}, nil
	}
	return result.Notifications, nil
}

// GetPackListingID queries the marketplace server for the listing_id of a pack by its name.
func (a *App) GetPackListingID(packName string) (int64, error) {
	if err := a.EnsureMarketplaceAuth(); err != nil {
		return 0, fmt.Errorf("marketplace auth failed: %w", err)
	}
	mc := a.marketplaceClient

	reqURL := mc.ServerURL + "/api/packs/listing-id?pack_name=" + url.QueryEscape(packName)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create listing-id request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch listing-id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		ListingID int64 `json:"listing_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode listing-id response: %w", err)
	}
	return result.ListingID, nil
}

// GetShareURL generates a share URL for the given pack and copies it to the clipboard.
func (a *App) GetShareURL(packName string) (string, error) {
	listingID, err := a.GetPackListingID(packName)
	if err != nil {
		return "", fmt.Errorf("failed to get listing ID: %w", err)
	}

	shareURL := fmt.Sprintf("https://market.vantagedata.chat/pack/%d", listingID)

	runtime.ClipboardSetText(a.ctx, shareURL)

	return shareURL, nil
}

