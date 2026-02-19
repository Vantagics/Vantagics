package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vantagedata/agent"
	"vantagedata/i18n"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// MarketplaceManager 定义市场管理接口
type MarketplaceManager interface {
	MarketplaceLoginWithSN() error
	EnsureMarketplaceAuth() error
	IsMarketplaceLoggedIn() bool
	MarketplacePortalLogin() (string, error)
	GetMarketplaceCategories() ([]PackCategory, error)
	SharePackToMarketplace(packFilePath string, categoryID int64, pricingModel string, creditsPrice int, detailedDescription string) error
	BrowseMarketplacePacks(categoryID int64) ([]PackListingInfo, error)
	GetMySharedPackNames() ([]string, error)
	GetMyPublishedPacks(sourceName string) ([]MyPublishedPackInfo, error)
	ReplaceMarketplacePack(localPackPath string, listingID int64) error
	GetMyPurchasedPacks() []PurchasedPackInfo
	DownloadMarketplacePack(listingID int64) (string, error)
	GetMarketplaceCreditsBalance() (float64, error)
	PurchaseAdditionalUses(listingID int64, quantity int) error
	RenewSubscription(listingID int64, months int) error
	GetUsageLicenses() []*UsageLicense
	GetPackLicenseInfo(listingID int64) *UsageLicense
	RefreshPurchasedPackLicenses() error
	FlushPendingUsageReports()
	ReportPackUsage(listingID int64, usedAt string) (*ReportPackUsageResponse, error)
	ValidateSubscriptionLicenseAsync(listingID int64)
	GetMarketplaceNotifications() ([]NotificationInfo, error)
	GetPackListingID(packName string) (int64, error)
	GetPackShareToken(packName string) (string, error)
	GetShareURL(packName string) (string, error)
	ServicePortalLogin() (string, error)
}

// MarketplaceFacadeService 市场服务门面，封装所有市场相关的业务逻辑和并发状态
type MarketplaceFacadeService struct {
	ctx             context.Context
	configProvider  ConfigProvider
	logger          func(string)

	// Marketplace client for API calls
	marketplaceClient *MarketplaceClient

	// License client dependency (for SN/email auth)
	licenseClient *agent.LicenseClient

	// Usage license store for local billing enforcement
	usageLicenseStore *UsageLicenseStore

	// Pending usage queue for offline usage report retry
	pendingUsageQueue *PendingUsageQueue

	// 并发状态（从 App 迁移过来）
	flushUsageMu sync.Mutex

	// Pack passwords from marketplace downloads (filePath -> encryption password)
	packPasswords map[string]string

	// Persistent pack password store (survives app restarts)
	packPasswordStore *PackPasswordStore
}

// NewMarketplaceFacadeService 创建新的 MarketplaceFacadeService 实例
func NewMarketplaceFacadeService(
	configProvider ConfigProvider,
	logger func(string),
) *MarketplaceFacadeService {
	return &MarketplaceFacadeService{
		configProvider: configProvider,
		logger:         logger,
		packPasswords:  make(map[string]string),
	}
}

// Name 返回服务名称
func (m *MarketplaceFacadeService) Name() string {
	return "marketplace"
}

// Initialize 初始化市场门面服务
func (m *MarketplaceFacadeService) Initialize(ctx context.Context) error {
	m.ctx = ctx

	// Initialize usage license store
	uls, err := NewUsageLicenseStore()
	if err != nil {
		m.log(fmt.Sprintf("[MARKETPLACE] Failed to create UsageLicenseStore: %v", err))
	} else {
		if err := uls.Load(); err != nil {
			m.log(fmt.Sprintf("[MARKETPLACE] Failed to load usage licenses: %v", err))
		}
		m.usageLicenseStore = uls
		m.log("[MARKETPLACE] UsageLicenseStore initialized successfully")
	}

	// Initialize pending usage queue
	puq, err := NewPendingUsageQueue()
	if err != nil {
		m.log(fmt.Sprintf("[MARKETPLACE] Failed to create PendingUsageQueue: %v", err))
	} else {
		if err := puq.Load(); err != nil {
			m.log(fmt.Sprintf("[MARKETPLACE] Failed to load pending usage queue: %v", err))
		}
		m.pendingUsageQueue = puq
		m.log("[MARKETPLACE] PendingUsageQueue initialized successfully")

		go func() {
			defer func() {
				if r := recover(); r != nil {
					m.log(fmt.Sprintf("[MARKETPLACE] FlushPendingUsageReports goroutine recovered from panic: %v", r))
				}
			}()
			m.FlushPendingUsageReports()
		}()
	}

	// Initialize pack password store
	pps, err := NewPackPasswordStore()
	if err != nil {
		m.log(fmt.Sprintf("[MARKETPLACE] Failed to create PackPasswordStore: %v", err))
	} else {
		if err := pps.Load(); err != nil {
			m.log(fmt.Sprintf("[MARKETPLACE] Failed to load pack passwords: %v", err))
		}
		m.packPasswordStore = pps
		pps.LoadIntoMap(m.packPasswords)
		m.log("[MARKETPLACE] PackPasswordStore initialized successfully")
	}

	m.log("MarketplaceFacadeService initialized")
	return nil
}

// Shutdown 关闭市场门面服务
func (m *MarketplaceFacadeService) Shutdown() error {
	m.log("MarketplaceFacadeService shutdown")
	return nil
}

// SetContext 设置 Wails 上下文
func (m *MarketplaceFacadeService) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// SetLicenseClient 注入许可证客户端依赖
func (m *MarketplaceFacadeService) SetLicenseClient(lc *agent.LicenseClient) {
	m.licenseClient = lc
}

// GetPackPasswords 返回 pack 密码映射（供外部使用）
func (m *MarketplaceFacadeService) GetPackPasswords() map[string]string {
	return m.packPasswords
}

// GetPackPasswordStore 返回 pack 密码存储
func (m *MarketplaceFacadeService) GetPackPasswordStore() *PackPasswordStore {
	return m.packPasswordStore
}

// GetUsageLicenseStore 返回使用许可证存储
func (m *MarketplaceFacadeService) GetUsageLicenseStore() *UsageLicenseStore {
	return m.usageLicenseStore
}

// ensureMarketplaceClient initializes the marketplace client if not already set.
func (m *MarketplaceFacadeService) ensureMarketplaceClient() {
	if m.marketplaceClient == nil {
		m.marketplaceClient = NewMarketplaceClient(DefaultMarketplaceServerURL)
	}
}

// MarketplaceLoginWithSN performs automatic login using the locally saved SN and Email.
func (m *MarketplaceFacadeService) MarketplaceLoginWithSN() error {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	if m.licenseClient == nil || !m.licenseClient.IsActivated() {
		return fmt.Errorf("license not activated, cannot login to marketplace")
	}
	sn := m.licenseClient.GetSN()
	if sn == "" {
		return fmt.Errorf("SN not available, cannot login to marketplace")
	}

	cfg, err := m.configProvider.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	email := cfg.LicenseEmail
	if email == "" {
		return fmt.Errorf("email not available, cannot login to marketplace")
	}

	// Step 1: POST to License server /api/marketplace-auth
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

	if authResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(authResp.Body)
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 0 && bodyStr[0] == '<' {
			return fmt.Errorf("license server returned HTML instead of JSON (status %d). The license server may be unavailable or misconfigured at %s", authResp.StatusCode, mc.LicenseServerURL)
		}
		return fmt.Errorf("license server returned status %d: %s", authResp.StatusCode, bodyStr)
	}

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

	// Step 2: POST to Market server /api/auth/sn-login
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

	mc.Token = loginResult.Token
	return nil
}

// EnsureMarketplaceAuth checks if the marketplace token is valid, and if not, performs automatic login.
func (m *MarketplaceFacadeService) EnsureMarketplaceAuth() error {
	m.ensureMarketplaceClient()
	if m.marketplaceClient.Token != "" {
		return nil
	}
	return m.MarketplaceLoginWithSN()
}

// IsMarketplaceLoggedIn returns true if the marketplace client has a valid token.
func (m *MarketplaceFacadeService) IsMarketplaceLoggedIn() bool {
	if m.marketplaceClient == nil {
		return false
	}
	return m.marketplaceClient.Token != ""
}

// MarketplacePortalLogin performs SSO login to the marketplace user portal.
func (m *MarketplaceFacadeService) MarketplacePortalLogin() (string, error) {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	if m.licenseClient == nil || !m.licenseClient.IsActivated() {
		return "", fmt.Errorf("license not activated")
	}
	sn := m.licenseClient.GetSN()
	if sn == "" {
		return "", fmt.Errorf("SN not available")
	}

	cfg, err := m.configProvider.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	email := cfg.LicenseEmail
	if email == "" {
		return "", fmt.Errorf("email not available")
	}

	// Step 1: POST to License Server /api/marketplace-auth
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

	// Step 2: POST to Marketplace Server /api/auth/sn-login
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

	if loginResult.Token != "" {
		mc.Token = loginResult.Token
	}

	return mc.ServerURL + "/user/ticket-login?ticket=" + loginResult.LoginTicket, nil
}

// GetMarketplaceCategories fetches the list of pack categories from the marketplace server.
func (m *MarketplaceFacadeService) GetMarketplaceCategories() ([]PackCategory, error) {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

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
func (m *MarketplaceFacadeService) SharePackToMarketplace(packFilePath string, categoryID int64, pricingModel string, creditsPrice int, detailedDescription string) error {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	fileInfo, err := os.Stat(packFilePath)
	if err != nil {
		return fmt.Errorf("failed to access pack file: %w", err)
	}
	const maxUploadSize = 500 * 1024 * 1024
	if fileInfo.Size() > maxUploadSize {
		return fmt.Errorf("pack file too large (%dMB), maximum is %dMB", fileInfo.Size()/1024/1024, maxUploadSize/1024/1024)
	}

	file, err := os.Open(packFilePath)
	if err != nil {
		return fmt.Errorf("failed to open pack file: %w", err)
	}
	defer file.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("file", filepath.Base(packFilePath))
		if err != nil {
			errCh <- fmt.Errorf("failed to create form file: %w", err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errCh <- fmt.Errorf("failed to write file data: %w", err)
			return
		}
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

	if writeErr := <-errCh; writeErr != nil {
		return writeErr
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	return nil
}

// BrowseMarketplacePacks fetches pack listings from the marketplace, optionally filtered by category.
func (m *MarketplaceFacadeService) BrowseMarketplacePacks(categoryID int64) ([]PackListingInfo, error) {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	reqURL := mc.ServerURL + "/api/packs"
	if categoryID > 0 {
		reqURL = fmt.Sprintf("%s?category_id=%d", reqURL, categoryID)
	}

	req, err := http.NewRequest("GET", reqURL, nil)
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
func (m *MarketplaceFacadeService) GetMySharedPackNames() ([]string, error) {
	if err := m.EnsureMarketplaceAuth(); err != nil {
		return []string{}, nil
	}
	mc := m.marketplaceClient

	if mc.Token == "" {
		return []string{}, nil
	}

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

	allPacks, err := m.BrowseMarketplacePacks(0)
	if err != nil {
		return []string{}, nil
	}

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

// GetMyPublishedPacks returns the current user's published packs, optionally filtered by source_name.
func (m *MarketplaceFacadeService) GetMyPublishedPacks(sourceName string) ([]MyPublishedPackInfo, error) {
	if err := m.EnsureMarketplaceAuth(); err != nil {
		return []MyPublishedPackInfo{}, nil
	}
	mc := m.marketplaceClient
	if mc.Token == "" {
		return []MyPublishedPackInfo{}, nil
	}

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

	allPacks, err := m.BrowseMarketplacePacks(0)
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
				Version:    0,
			})
		}
	}
	if result == nil {
		result = []MyPublishedPackInfo{}
	}
	return result, nil
}

// ReplaceMarketplacePack replaces a published pack's content with a local pack file.
func (m *MarketplaceFacadeService) ReplaceMarketplacePack(localPackPath string, listingID int64) error {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

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

// GetMyPurchasedPacks fetches the current user's purchased packs from the marketplace.
func (m *MarketplaceFacadeService) GetMyPurchasedPacks() []PurchasedPackInfo {
	if err := m.EnsureMarketplaceAuth(); err != nil {
		return []PurchasedPackInfo{}
	}
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

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

// DownloadMarketplacePack downloads a pack from the marketplace and saves it.
func (m *MarketplaceFacadeService) DownloadMarketplacePack(listingID int64) (string, error) {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	if mc.Token == "" {
		return "", fmt.Errorf("not logged in to marketplace")
	}

	dlURL := fmt.Sprintf("%s/api/packs/%d/download", mc.ServerURL, listingID)
	req, err := http.NewRequest("GET", dlURL, nil)
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

	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("marketplace_pack_%d.qap", listingID)
	filePath := filepath.Join(tmpDir, fileName)

	const maxDownloadSize int64 = 500 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxDownloadSize+1)

	outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	written, err := io.Copy(outFile, limitedReader)
	outFile.Close()
	if err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("failed to write download to disk: %w", err)
	}
	if written > maxDownloadSize {
		os.Remove(filePath)
		return "", fmt.Errorf("downloaded file exceeds maximum size limit (%dMB)", maxDownloadSize/1024/1024)
	}

	cfg, err := m.configProvider.GetConfig()
	if err != nil {
		return filePath, fmt.Errorf("downloaded to temp but failed to get config for QAP directory: %w", err)
	}
	qapDir := filepath.Join(cfg.DataCacheDir, "qap")
	if err := os.MkdirAll(qapDir, 0755); err != nil {
		return filePath, fmt.Errorf("downloaded to temp but failed to create QAP directory: %w", err)
	}
	finalPath := filepath.Join(qapDir, fileName)

	// Clean up old timestamped files for the same listing ID
	oldPattern := fmt.Sprintf("marketplace_pack_%d_*.qap", listingID)
	if matches, err := filepath.Glob(filepath.Join(qapDir, oldPattern)); err == nil {
		for _, oldFile := range matches {
			os.Remove(oldFile)
		}
	}

	if err := os.Rename(filePath, finalPath); err != nil {
		if copyErr := copyFile(filePath, finalPath); copyErr != nil {
			return filePath, fmt.Errorf("downloaded to temp but failed to move to QAP directory: %w", copyErr)
		}
		os.Remove(filePath)
	}

	// Extract encryption password from response header
	if encPassword := resp.Header.Get("X-Encryption-Password"); encPassword != "" {
		if m.packPasswords == nil {
			m.packPasswords = make(map[string]string)
		}
		m.packPasswords[finalPath] = encPassword
		if m.packPasswordStore != nil {
			m.packPasswordStore.SetPassword(finalPath, encPassword)
			_ = m.packPasswordStore.Save()
		}
	}

	// Parse X-Usage-License header if present
	if licenseHeader := resp.Header.Get("X-Usage-License"); licenseHeader != "" {
		var usageLicense UsageLicense
		if err := json.Unmarshal([]byte(licenseHeader), &usageLicense); err == nil {
			if m.usageLicenseStore != nil {
				m.usageLicenseStore.SetLicense(&usageLicense)
				_ = m.usageLicenseStore.Save()
			}
		}
	}

	return finalPath, nil
}

// GetMarketplaceCreditsBalance fetches the current user's credits balance.
func (m *MarketplaceFacadeService) GetMarketplaceCreditsBalance() (float64, error) {
	if err := m.EnsureMarketplaceAuth(); err != nil {
		return 0, fmt.Errorf("marketplace auth failed: %w", err)
	}
	mc := m.marketplaceClient

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
func (m *MarketplaceFacadeService) PurchaseAdditionalUses(listingID int64, quantity int) error {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	payload, _ := json.Marshal(map[string]int{"quantity": quantity})
	reqURL := fmt.Sprintf("%s/api/packs/%d/purchase-uses", mc.ServerURL, listingID)
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(payload))
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
		body := readErrorBody(resp.Body)
		if resp.StatusCode == http.StatusPaymentRequired {
			var errResp struct {
				Error    string  `json:"error"`
				Required int     `json:"required"`
				Balance  float64 `json:"balance"`
			}
			if json.Unmarshal([]byte(body), &errResp) == nil && errResp.Error == "INSUFFICIENT_CREDITS" {
				return fmt.Errorf("%s", i18n.T("marketplace.insufficient_credits", errResp.Required, errResp.Balance))
			}
		}
		return fmt.Errorf("purchase failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		RemainingUses int `json:"remaining_uses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode purchase response: %w", err)
	}

	if m.usageLicenseStore != nil {
		lic := m.usageLicenseStore.GetLicense(listingID)
		if lic != nil {
			lic.RemainingUses += result.RemainingUses
			lic.TotalUses += quantity
			lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			m.usageLicenseStore.SetLicense(lic)
			_ = m.usageLicenseStore.Save()
		}
	}

	return nil
}

// RenewSubscription renews a subscription pack listing.
func (m *MarketplaceFacadeService) RenewSubscription(listingID int64, months int) error {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	if mc.Token == "" {
		return fmt.Errorf("not logged in to marketplace")
	}

	payload, _ := json.Marshal(map[string]int{"months": months})
	reqURL := fmt.Sprintf("%s/api/packs/%d/renew", mc.ServerURL, listingID)
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(payload))
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
		body := readErrorBody(resp.Body)
		if resp.StatusCode == http.StatusPaymentRequired {
			var errResp struct {
				Error    string  `json:"error"`
				Required int     `json:"required"`
				Balance  float64 `json:"balance"`
			}
			if json.Unmarshal([]byte(body), &errResp) == nil && errResp.Error == "INSUFFICIENT_CREDITS" {
				return fmt.Errorf("%s", i18n.T("marketplace.insufficient_credits", errResp.Required, errResp.Balance))
			}
		}
		return fmt.Errorf("renew failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		ExpiresAt          string `json:"expires_at"`
		SubscriptionMonths int    `json:"subscription_months"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode renew response: %w", err)
	}

	if m.usageLicenseStore != nil {
		lic := m.usageLicenseStore.GetLicense(listingID)
		if lic != nil {
			lic.ExpiresAt = result.ExpiresAt
			lic.SubscriptionMonths = result.SubscriptionMonths
			lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			m.usageLicenseStore.SetLicense(lic)
			_ = m.usageLicenseStore.Save()
		}
	}

	return nil
}

// GetUsageLicenses returns all local usage licenses.
func (m *MarketplaceFacadeService) GetUsageLicenses() []*UsageLicense {
	if m.usageLicenseStore == nil {
		return nil
	}
	return m.usageLicenseStore.GetAllLicenses()
}

// GetPackLicenseInfo returns the usage license for a specific listing ID.
func (m *MarketplaceFacadeService) GetPackLicenseInfo(listingID int64) *UsageLicense {
	if m.usageLicenseStore == nil || listingID <= 0 {
		return nil
	}
	return m.usageLicenseStore.GetLicense(listingID)
}

// RefreshPurchasedPackLicenses fetches the latest usage license info from the server.
func (m *MarketplaceFacadeService) RefreshPurchasedPackLicenses() error {
	if err := m.EnsureMarketplaceAuth(); err != nil {
		return fmt.Errorf("marketplace auth failed: %w", err)
	}
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

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

	if m.usageLicenseStore == nil {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, lic := range result.Licenses {
		lic.UpdatedAt = now
		if lic.CreatedAt == "" {
			lic.CreatedAt = now
		}
		if lic.PricingModel == "subscription" || lic.PricingModel == "time_limited" {
			if expiresAt, err := time.Parse(time.RFC3339, lic.ExpiresAt); err == nil {
				lic.Blocked = !time.Now().Before(expiresAt)
			} else {
				lic.Blocked = false
			}
		}
		m.usageLicenseStore.SetLicense(&lic)
	}
	return m.usageLicenseStore.Save()
}

// FlushPendingUsageReports processes all pending usage records in the queue.
func (m *MarketplaceFacadeService) FlushPendingUsageReports() {
	if m.pendingUsageQueue == nil {
		return
	}

	m.flushUsageMu.Lock()
	defer m.flushUsageMu.Unlock()

	records := m.pendingUsageQueue.GetAll()
	for _, rec := range records {
		_ = m.pendingUsageQueue.Dequeue(rec.ListingID, rec.UsedAt)
		_, err := m.ReportPackUsage(rec.ListingID, rec.UsedAt)
		if err != nil {
			fmt.Printf("[FlushPendingUsageReports] failed to report listing %d at %s: %v\n", rec.ListingID, rec.UsedAt, err)
		}
	}

	_ = m.pendingUsageQueue.Save()
}

// ReportPackUsage reports a per_use pack usage to the marketplace server.
func (m *MarketplaceFacadeService) ReportPackUsage(listingID int64, usedAt string) (*ReportPackUsageResponse, error) {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

	if mc.Token == "" {
		if m.pendingUsageQueue != nil {
			_ = m.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("not logged in to marketplace")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"listing_id": listingID,
		"used_at":    usedAt,
	})
	reqURL := fmt.Sprintf("%s/api/packs/report-usage", mc.ServerURL)
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(payload))
	if err != nil {
		if m.pendingUsageQueue != nil {
			_ = m.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("failed to create report-usage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		if m.pendingUsageQueue != nil {
			_ = m.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("failed to report pack usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		errMsg := readErrorBody(resp.Body)
		fmt.Printf("[ReportPackUsage] warning: server returned 400 for listing %d: %s\n", listingID, errMsg)
		return nil, fmt.Errorf("invalid usage report (400): %s", errMsg)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := readErrorBody(resp.Body)
		if m.pendingUsageQueue != nil {
			_ = m.pendingUsageQueue.Enqueue(PendingUsageRecord{ListingID: listingID, UsedAt: usedAt})
		}
		return nil, fmt.Errorf("report-usage failed with status %d: %s", resp.StatusCode, errMsg)
	}

	var result ReportPackUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode report-usage response: %w", err)
	}

	if m.usageLicenseStore != nil {
		lic := m.usageLicenseStore.GetLicense(listingID)
		if lic != nil && lic.PricingModel == "per_use" {
			if result.TotalPurchased > 0 {
				lic.TotalUses = result.TotalPurchased
			}
			lic.RemainingUses = result.RemainingUses
			if lic.RemainingUses < 0 {
				lic.RemainingUses = 0
			}
			lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			m.usageLicenseStore.SetLicense(lic)
			_ = m.usageLicenseStore.Save()
			m.log(fmt.Sprintf("[REPORT-USAGE] Synced license for listing %d: remaining=%d, total=%d, used=%d, exhausted=%v",
				listingID, lic.RemainingUses, lic.TotalUses, result.UsedCount, result.Exhausted))
		}
	}

	return &result, nil
}

// ValidateSubscriptionLicenseAsync refreshes licenses in the background.
func (m *MarketplaceFacadeService) ValidateSubscriptionLicenseAsync(listingID int64) {
	go func() {
		m.log(fmt.Sprintf("[LICENSE-VALIDATE] Starting async subscription validation for listing %d", listingID))
		err := m.RefreshPurchasedPackLicenses()
		if err != nil {
			m.log(fmt.Sprintf("[LICENSE-VALIDATE] Failed to refresh licenses from server: %v", err))
			return
		}
		if m.usageLicenseStore == nil {
			return
		}
		lic := m.usageLicenseStore.GetLicense(listingID)
		if lic == nil {
			return
		}
		if lic.Blocked {
			m.log(fmt.Sprintf("[LICENSE-VALIDATE] Listing %d subscription confirmed expired by server, marked as blocked", listingID))
		} else {
			m.log(fmt.Sprintf("[LICENSE-VALIDATE] Listing %d subscription still valid", listingID))
		}
	}()
}

// GetMarketplaceNotifications fetches active notifications from the marketplace server.
func (m *MarketplaceFacadeService) GetMarketplaceNotifications() ([]NotificationInfo, error) {
	m.ensureMarketplaceClient()
	mc := m.marketplaceClient

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
func (m *MarketplaceFacadeService) GetPackListingID(packName string) (int64, error) {
	if err := m.EnsureMarketplaceAuth(); err != nil {
		return 0, fmt.Errorf("marketplace auth failed: %w", err)
	}
	mc := m.marketplaceClient

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
		ListingID int64  `json:"listing_id"`
		ShareToken string `json:"share_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode listing-id response: %w", err)
	}
	return result.ListingID, nil
}

// GetPackShareToken queries the marketplace server for the share_token of a pack by its name.
func (m *MarketplaceFacadeService) GetPackShareToken(packName string) (string, error) {
	if err := m.EnsureMarketplaceAuth(); err != nil {
		return "", fmt.Errorf("marketplace auth failed: %w", err)
	}
	mc := m.marketplaceClient

	reqURL := mc.ServerURL + "/api/packs/listing-id?pack_name=" + url.QueryEscape(packName)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create listing-id request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+mc.Token)

	resp, err := mc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch share token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var result struct {
		ListingID  int64  `json:"listing_id"`
		ShareToken string `json:"share_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode share token response: %w", err)
	}
	if result.ShareToken == "" {
		return "", fmt.Errorf("share token not available for pack %q", packName)
	}
	return result.ShareToken, nil
}

// GetShareURL generates a share URL for the given pack and copies it to the clipboard.
func (m *MarketplaceFacadeService) GetShareURL(packName string) (string, error) {
	shareToken, err := m.GetPackShareToken(packName)
	if err != nil {
		return "", fmt.Errorf("failed to get share token: %w", err)
	}

	shareURL := fmt.Sprintf("https://market.vantagics.com/pack/%s", shareToken)

	runtime.ClipboardSetText(m.ctx, shareURL)

	return shareURL, nil
}

// ServicePortalLogin performs the SSO login flow for the service portal.
func (m *MarketplaceFacadeService) ServicePortalLogin() (string, error) {
	if m.licenseClient == nil || !m.licenseClient.IsActivated() {
		return "", fmt.Errorf("license not activated")
	}
	sn := m.licenseClient.GetSN()
	if sn == "" {
		return "", fmt.Errorf("SN not available")
	}

	cfg, err := m.configProvider.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	email := cfg.LicenseEmail
	if email == "" {
		return "", fmt.Errorf("email not available")
	}

	spc := NewServicePortalClient("")

	// Step 1: POST to License Server /api/marketplace-auth
	authPayload, _ := json.Marshal(map[string]string{
		"sn":    sn,
		"email": email,
	})
	authResp, err := spc.client.Post(
		spc.LicenseServerURL+"/api/marketplace-auth",
		"application/json",
		bytes.NewReader(authPayload),
	)
	if err != nil {
		return "", fmt.Errorf("failed to contact license server: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(authResp.Body)
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 0 && bodyStr[0] == '<' {
			return "", fmt.Errorf("license server returned HTML instead of JSON (status %d). The license server may be unavailable or misconfigured at %s", authResp.StatusCode, spc.LicenseServerURL)
		}
		return "", fmt.Errorf("license server returned status %d: %s", authResp.StatusCode, bodyStr)
	}

	bodyBytes, readErr := io.ReadAll(authResp.Body)
	if readErr != nil {
		return "", fmt.Errorf("failed to read license server response: %w", readErr)
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
			return "", fmt.Errorf("license server returned HTML instead of JSON. The license server may be unavailable or misconfigured at %s. Response preview: %.200s", spc.LicenseServerURL, bodyStr)
		}
		return "", fmt.Errorf("failed to decode license server response: %w. Response body: %s", err, bodyStr)
	}
	if !authResult.Success {
		return "", fmt.Errorf("license authentication failed: %s (%s)", authResult.Message, authResult.Code)
	}

	// Step 2: POST to Service Portal /api/auth/sn-login
	loginPayload, _ := json.Marshal(map[string]string{
		"token": authResult.Token,
	})
	loginResp, err := spc.client.Post(
		spc.ServerURL+"/api/auth/sn-login",
		"application/json",
		bytes.NewReader(loginPayload),
	)
	if err != nil {
		return "", fmt.Errorf("failed to contact service portal: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		loginBodyBytes, _ := io.ReadAll(loginResp.Body)
		loginBodyStr := string(loginBodyBytes)
		if len(loginBodyStr) > 0 && loginBodyStr[0] == '<' {
			return "", fmt.Errorf("service portal returned HTML instead of JSON (status %d)", loginResp.StatusCode)
		}
		return "", fmt.Errorf("service portal returned status %d: %s", loginResp.StatusCode, loginBodyStr)
	}

	loginBodyBytes, readErr := io.ReadAll(loginResp.Body)
	if readErr != nil {
		return "", fmt.Errorf("failed to read service portal response: %w", readErr)
	}

	var loginResult LoginResult
	if err := json.Unmarshal(loginBodyBytes, &loginResult); err != nil {
		loginBodyStr := string(loginBodyBytes)
		if len(loginBodyStr) > 0 && loginBodyStr[0] == '<' {
			return "", fmt.Errorf("service portal returned HTML instead of JSON. Response preview: %.200s", loginBodyStr)
		}
		return "", fmt.Errorf("failed to decode service portal response: %w", err)
	}
	if !loginResult.Success {
		return "", fmt.Errorf("service portal login failed: %s", loginResult.Message)
	}

	return BuildTicketLoginURL(loginResult.LoginTicket), nil
}

// log 记录日志
func (m *MarketplaceFacadeService) log(msg string) {
	if m.logger != nil {
		m.logger(msg)
	}
}
