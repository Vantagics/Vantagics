package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// DefaultMarketplaceServerURL is the default marketplace server address.
const DefaultMarketplaceServerURL = "http://localhost:8090"

// MarketplaceClient is the HTTP client for communicating with the marketplace server.
type MarketplaceClient struct {
	ServerURL string
	Token     string // JWT token
	client    *http.Client
}

// NewMarketplaceClient creates a new MarketplaceClient with the given server URL.
func NewMarketplaceClient(serverURL string) *MarketplaceClient {
	if serverURL == "" {
		serverURL = DefaultMarketplaceServerURL
	}
	return &MarketplaceClient{
		ServerURL: serverURL,
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

// openBrowser opens the given URL in the system's default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// MarketplaceLogin starts the OAuth login flow by opening the system browser
// and waiting for the callback with a JWT token.
func (a *App) MarketplaceLogin(provider string) error {
	a.ensureMarketplaceClient()
	mc := a.marketplaceClient

	// Start a temporary local HTTP server to receive the OAuth callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local callback server: %w", err)
	}
	callbackPort := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", callbackPort)

	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Set up the callback handler
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			errCh <- fmt.Errorf("OAuth callback missing token")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h2>Login successful! You can close this window.</h2></body></html>"))
		tokenCh <- token
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Ensure server is always shut down gracefully
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Open the browser to the marketplace OAuth page
	oauthURL := fmt.Sprintf("%s/api/auth/oauth?provider=%s&callback=%s", mc.ServerURL, provider, callbackURL)
	if err := openBrowser(oauthURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for the token or timeout
	select {
	case token := <-tokenCh:
		mc.Token = token
		return nil
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("OAuth login timed out")
	}
}

// IsMarketplaceLoggedIn returns true if the marketplace client has a valid token.
func (a *App) IsMarketplaceLoggedIn() bool {
	if a.marketplaceClient == nil {
		return false
	}
	return a.marketplaceClient.Token != ""
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
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
func (a *App) SharePackToMarketplace(packFilePath string, categoryID int64, shareMode string, creditsPrice int) error {
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

	// Read the .qap file
	fileData, err := os.ReadFile(packFilePath)
	if err != nil {
		return fmt.Errorf("failed to read pack file: %w", err)
	}

	// Build multipart form request
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file field
	part, err := writer.CreateFormFile("file", filepath.Base(packFilePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	// Add form fields
	writer.WriteField("category_id", fmt.Sprintf("%d", categoryID))
	writer.WriteField("share_mode", shareMode)
	writer.WriteField("credits_price", fmt.Sprintf("%d", creditsPrice))

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", mc.ServerURL+"/api/packs/upload", &buf)
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
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
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Save to temp directory with size limit to prevent memory exhaustion
	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("marketplace_pack_%d_%d.qap", listingID, time.Now().UnixMilli())
	filePath := filepath.Join(tmpDir, fileName)

	const maxDownloadSize = 500 * 1024 * 1024 // 500MB limit
	limitedReader := io.LimitReader(resp.Body, maxDownloadSize+1)
	fileData, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read download response: %w", err)
	}
	if int64(len(fileData)) > maxDownloadSize {
		return "", fmt.Errorf("downloaded file exceeds maximum size limit (%dMB)", maxDownloadSize/1024/1024)
	}

	if err := os.WriteFile(filePath, fileData, 0600); err != nil {
		return "", fmt.Errorf("failed to save downloaded pack: %w", err)
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
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Balance float64 `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode balance response: %w", err)
	}
	return result.Balance, nil
}
