package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DefaultMarketplaceServerURL is the default marketplace server address.
const DefaultMarketplaceServerURL = "https://market.vantagics.com"

// DefaultLicenseServerURL is the default license server address.
const DefaultLicenseServerURL = "https://license.vantagics.com"

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

// MyPublishedPackInfo represents a published pack owned by the current user (for replacement selection).
type MyPublishedPackInfo struct {
	ListingID  int64  `json:"id"`
	PackName   string `json:"pack_name"`
	SourceName string `json:"source_name"`
	Version    int    `json:"version"`
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

// ReportPackUsageResponse 服务器上报响�?
type ReportPackUsageResponse struct {
	Success        bool `json:"success"`
	UsedCount      int  `json:"used_count"`
	TotalPurchased int  `json:"total_purchased"`
	RemainingUses  int  `json:"remaining_uses"`
	Exhausted      bool `json:"exhausted"`
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

// --- Marketplace Facade Delegation Methods ---

func (a *App) ensureMarketplaceClient() {
	if a.marketplaceFacadeService == nil {
		return
	}
	a.marketplaceFacadeService.ensureMarketplaceClient()
}

func (a *App) MarketplaceLoginWithSN() error {
	if a.marketplaceFacadeService == nil {
		return WrapError("App", "MarketplaceLoginWithSN", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.MarketplaceLoginWithSN()
}

func (a *App) EnsureMarketplaceAuth() error {
	if a.marketplaceFacadeService == nil {
		return WrapError("App", "EnsureMarketplaceAuth", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.EnsureMarketplaceAuth()
}

func (a *App) IsMarketplaceLoggedIn() bool {
	if a.marketplaceFacadeService == nil {
		return false
	}
	return a.marketplaceFacadeService.IsMarketplaceLoggedIn()
}

func (a *App) MarketplacePortalLogin() (string, error) {
	if a.marketplaceFacadeService == nil {
		return "", WrapError("App", "MarketplacePortalLogin", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.MarketplacePortalLogin()
}

func (a *App) GetMarketplaceCategories() ([]PackCategory, error) {
	if a.marketplaceFacadeService == nil {
		return nil, WrapError("App", "GetMarketplaceCategories", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.GetMarketplaceCategories()
}

func (a *App) SharePackToMarketplace(packFilePath string, categoryID int64, pricingModel string, creditsPrice int, detailedDescription string) error {
	if a.marketplaceFacadeService == nil {
		return WrapError("App", "SharePackToMarketplace", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.SharePackToMarketplace(packFilePath, categoryID, pricingModel, creditsPrice, detailedDescription)
}

func (a *App) BrowseMarketplacePacks(categoryID int64) ([]PackListingInfo, error) {
	if a.marketplaceFacadeService == nil {
		return nil, WrapError("App", "BrowseMarketplacePacks", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.BrowseMarketplacePacks(categoryID)
}

func (a *App) GetMySharedPackNames() ([]string, error) {
	if a.marketplaceFacadeService == nil {
		return nil, WrapError("App", "GetMySharedPackNames", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.GetMySharedPackNames()
}

func (a *App) GetMyPublishedPacks(sourceName string) ([]MyPublishedPackInfo, error) {
	if a.marketplaceFacadeService == nil {
		return nil, WrapError("App", "GetMyPublishedPacks", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.GetMyPublishedPacks(sourceName)
}

func (a *App) ReplaceMarketplacePack(localPackPath string, listingID int64) error {
	if a.marketplaceFacadeService == nil {
		return WrapError("App", "ReplaceMarketplacePack", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.ReplaceMarketplacePack(localPackPath, listingID)
}

func (a *App) GetMyPurchasedPacks() []PurchasedPackInfo {
	if a.marketplaceFacadeService == nil {
		return []PurchasedPackInfo{}
	}
	return a.marketplaceFacadeService.GetMyPurchasedPacks()
}

func (a *App) DownloadMarketplacePack(listingID int64) (string, error) {
	if a.marketplaceFacadeService == nil {
		return "", WrapError("App", "DownloadMarketplacePack", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.DownloadMarketplacePack(listingID)
}

func (a *App) GetMarketplaceCreditsBalance() (float64, error) {
	if a.marketplaceFacadeService == nil {
		return 0, WrapError("App", "GetMarketplaceCreditsBalance", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.GetMarketplaceCreditsBalance()
}

func (a *App) PurchaseAdditionalUses(listingID int64, quantity int) error {
	if a.marketplaceFacadeService == nil {
		return WrapError("App", "PurchaseAdditionalUses", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.PurchaseAdditionalUses(listingID, quantity)
}

func (a *App) RenewSubscription(listingID int64, months int) error {
	if a.marketplaceFacadeService == nil {
		return WrapError("App", "RenewSubscription", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.RenewSubscription(listingID, months)
}

func (a *App) GetUsageLicenses() []*UsageLicense {
	if a.marketplaceFacadeService == nil {
		return nil
	}
	return a.marketplaceFacadeService.GetUsageLicenses()
}

func (a *App) GetPackLicenseInfo(listingID int64) *UsageLicense {
	if a.marketplaceFacadeService == nil {
		return nil
	}
	return a.marketplaceFacadeService.GetPackLicenseInfo(listingID)
}

func (a *App) RefreshPurchasedPackLicenses() error {
	if a.marketplaceFacadeService == nil {
		return WrapError("App", "RefreshPurchasedPackLicenses", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.RefreshPurchasedPackLicenses()
}

func (a *App) FlushPendingUsageReports() {
	if a.marketplaceFacadeService == nil {
		return
	}
	a.marketplaceFacadeService.FlushPendingUsageReports()
}

func (a *App) ReportPackUsage(listingID int64, usedAt string) (*ReportPackUsageResponse, error) {
	if a.marketplaceFacadeService == nil {
		return nil, WrapError("App", "ReportPackUsage", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.ReportPackUsage(listingID, usedAt)
}

func (a *App) ValidateSubscriptionLicenseAsync(listingID int64) {
	if a.marketplaceFacadeService == nil {
		return
	}
	a.marketplaceFacadeService.ValidateSubscriptionLicenseAsync(listingID)
}

func (a *App) GetMarketplaceNotifications() ([]NotificationInfo, error) {
	if a.marketplaceFacadeService == nil {
		return nil, WrapError("App", "GetMarketplaceNotifications", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.GetMarketplaceNotifications()
}

func (a *App) GetPackListingID(packName string) (int64, error) {
	if a.marketplaceFacadeService == nil {
		return 0, WrapError("App", "GetPackListingID", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.GetPackListingID(packName)
}

func (a *App) GetShareURL(packName string) (string, error) {
	if a.marketplaceFacadeService == nil {
		return "", WrapError("App", "GetShareURL", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.GetShareURL(packName)
}
