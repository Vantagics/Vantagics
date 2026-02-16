package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
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

	resp, err := mc.client.Get(url)
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
	fileName := fmt.Sprintf("marketplace_pack_%d_%d.qap", listingID, time.Now().UnixMilli())
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

	// Extract encryption password from response header (for paid packs)
	if encPassword := resp.Header.Get("X-Encryption-Password"); encPassword != "" {
		if a.packPasswords == nil {
			a.packPasswords = make(map[string]string)
		}
		a.packPasswords[filePath] = encPassword
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

	return filePath, nil
}

// GetMarketplaceCreditsBalance fetches the current user's credits balance from the marketplace.
func (a *App) GetMarketplaceCreditsBalance() (float64, error) {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	if mc.Token == "" {
		return 0, fmt.Errorf("not logged in to marketplace")
	}

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
