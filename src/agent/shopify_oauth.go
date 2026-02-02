package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ShopifyOAuthConfig holds the OAuth configuration for Shopify
type ShopifyOAuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scopes       string `json:"scopes"`
}

// ShopifyOAuthResult holds the result of OAuth flow
type ShopifyOAuthResult struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	Shop        string `json:"shop"`
	Error       string `json:"error,omitempty"`
}

// ShopifyOAuthService handles Shopify OAuth flow
type ShopifyOAuthService struct {
	config       ShopifyOAuthConfig
	log          func(string)
	server       *http.Server
	serverMutex  sync.Mutex
	resultChan   chan ShopifyOAuthResult
	state        string
	shop         string
	callbackPort int
}

// NewShopifyOAuthService creates a new Shopify OAuth service
func NewShopifyOAuthService(config ShopifyOAuthConfig, log func(string)) *ShopifyOAuthService {
	return &ShopifyOAuthService{
		config:     config,
		log:        log,
		resultChan: make(chan ShopifyOAuthResult, 1),
	}
}

// generateState generates a random state string for CSRF protection
func (s *ShopifyOAuthService) generateState() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a less secure but functional state if crypto/rand fails
		s.log(fmt.Sprintf("[SHOPIFY-OAUTH] Warning: crypto/rand failed: %v, using fallback", err))
		// Use current timestamp as fallback (less secure but functional)
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// GetAuthURL returns the Shopify OAuth authorization URL
func (s *ShopifyOAuthService) GetAuthURL(shop string) (string, int, error) {
	s.serverMutex.Lock()
	defer s.serverMutex.Unlock()

	// Normalize shop URL
	shop = s.normalizeShop(shop)
	if shop == "" {
		return "", 0, fmt.Errorf("invalid shop URL")
	}
	s.shop = shop

	// Generate state for CSRF protection
	s.state = s.generateState()

	// Use fixed port for callback server (easier to whitelist in Shopify)
	s.callbackPort = 21549

	// Build redirect URI
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", s.callbackPort)

	// Default scopes for BI analysis
	scopes := s.config.Scopes
	if scopes == "" {
		scopes = "read_orders,read_products,read_customers,read_inventory"
	}

	// Build authorization URL
	authURL := fmt.Sprintf(
		"https://%s/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s&state=%s",
		shop,
		url.QueryEscape(s.config.ClientID),
		url.QueryEscape(scopes),
		url.QueryEscape(redirectURI),
		url.QueryEscape(s.state),
	)

	s.log(fmt.Sprintf("[SHOPIFY-OAUTH] Auth URL generated for shop: %s, callback: %s", shop, redirectURI))
	return authURL, s.callbackPort, nil
}

// StartCallbackServer starts the local HTTP server to receive OAuth callback
func (s *ShopifyOAuthService) StartCallbackServer(ctx context.Context) error {
	s.serverMutex.Lock()
	if s.callbackPort == 0 {
		s.serverMutex.Unlock()
		return fmt.Errorf("callback port not set, call GetAuthURL first")
	}
	port := s.callbackPort
	s.serverMutex.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	s.log(fmt.Sprintf("[SHOPIFY-OAUTH] Starting callback server on port %d", port))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log(fmt.Sprintf("[SHOPIFY-OAUTH] Server error: %v", err))
			s.resultChan <- ShopifyOAuthResult{Error: err.Error()}
		}
	}()

	return nil
}

// handleCallback handles the OAuth callback from Shopify
func (s *ShopifyOAuthService) handleCallback(w http.ResponseWriter, r *http.Request) {
	s.log("[SHOPIFY-OAUTH] Received callback")

	// Verify state to prevent CSRF
	state := r.URL.Query().Get("state")
	if state != s.state {
		s.log("[SHOPIFY-OAUTH] State mismatch - possible CSRF attack")
		s.sendErrorResponse(w, "State verification failed")
		s.resultChan <- ShopifyOAuthResult{Error: "state verification failed"}
		return
	}

	// Check for error response
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		s.log(fmt.Sprintf("[SHOPIFY-OAUTH] Error from Shopify: %s - %s", errMsg, errDesc))
		s.sendErrorResponse(w, errDesc)
		s.resultChan <- ShopifyOAuthResult{Error: errDesc}
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	shop := r.URL.Query().Get("shop")
	if code == "" {
		s.log("[SHOPIFY-OAUTH] No authorization code received")
		s.sendErrorResponse(w, "No authorization code received")
		s.resultChan <- ShopifyOAuthResult{Error: "no authorization code"}
		return
	}

	// Exchange code for access token
	accessToken, scope, err := s.exchangeCodeForToken(shop, code)
	if err != nil {
		s.log(fmt.Sprintf("[SHOPIFY-OAUTH] Token exchange failed: %v", err))
		s.sendErrorResponse(w, "Failed to get access token")
		s.resultChan <- ShopifyOAuthResult{Error: err.Error()}
		return
	}

	s.log("[SHOPIFY-OAUTH] Successfully obtained access token")
	s.sendSuccessResponse(w)
	s.resultChan <- ShopifyOAuthResult{
		AccessToken: accessToken,
		Scope:       scope,
		Shop:        shop,
	}
}

// exchangeCodeForToken exchanges the authorization code for an access token
func (s *ShopifyOAuthService) exchangeCodeForToken(shop, code string) (string, string, error) {
	tokenURL := fmt.Sprintf("https://%s/admin/oauth/access_token", shop)

	// Prepare request body
	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("code", code)

	// Make POST request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to request token: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("token request failed: %s", string(body))
	}

	// Parse response
	var result struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %v", err)
	}

	return result.AccessToken, result.Scope, nil
}

// WaitForResult waits for the OAuth result with timeout
func (s *ShopifyOAuthService) WaitForResult(timeout time.Duration) ShopifyOAuthResult {
	select {
	case result := <-s.resultChan:
		return result
	case <-time.After(timeout):
		return ShopifyOAuthResult{Error: "OAuth timeout - user did not complete authorization"}
	}
}

// StopCallbackServer stops the callback server
func (s *ShopifyOAuthService) StopCallbackServer() {
	s.serverMutex.Lock()
	defer s.serverMutex.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
		s.server = nil
		s.log("[SHOPIFY-OAUTH] Callback server stopped")
	}
}

// normalizeShop normalizes the shop URL to just the domain
func (s *ShopifyOAuthService) normalizeShop(shop string) string {
	shop = strings.TrimSpace(shop)
	shop = strings.TrimPrefix(shop, "https://")
	shop = strings.TrimPrefix(shop, "http://")
	shop = strings.TrimSuffix(shop, "/")

	// Ensure it ends with .myshopify.com
	if !strings.Contains(shop, ".myshopify.com") {
		if !strings.Contains(shop, ".") {
			shop = shop + ".myshopify.com"
		}
	}

	return shop
}

// sendSuccessResponse sends a success HTML response to the browser
func (s *ShopifyOAuthService) sendSuccessResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Authorization Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
               display: flex; justify-content: center; align-items: center; height: 100vh; 
               margin: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
        .container { text-align: center; background: white; padding: 40px 60px; border-radius: 16px; 
                     box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
        .icon { font-size: 64px; margin-bottom: 20px; }
        h1 { color: #1a1a2e; margin: 0 0 10px 0; }
        p { color: #666; margin: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✅</div>
        <h1>Authorization Successful!</h1>
        <p>You can close this window and return to VantageData.</p>
    </div>
    <script>setTimeout(function() { window.close(); }, 3000);</script>
</body>
</html>`
	w.Write([]byte(html))
}

// sendErrorResponse sends an error HTML response to the browser
func (s *ShopifyOAuthService) sendErrorResponse(w http.ResponseWriter, message string) {
	// Escape HTML special characters to prevent XSS
	escapedMessage := strings.ReplaceAll(message, "&", "&amp;")
	escapedMessage = strings.ReplaceAll(escapedMessage, "<", "&lt;")
	escapedMessage = strings.ReplaceAll(escapedMessage, ">", "&gt;")
	escapedMessage = strings.ReplaceAll(escapedMessage, "\"", "&quot;")
	escapedMessage = strings.ReplaceAll(escapedMessage, "'", "&#39;")
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authorization Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
               display: flex; justify-content: center; align-items: center; height: 100vh; 
               margin: 0; background: linear-gradient(135deg, #ff6b6b 0%, #ee5a5a 100%); }
        .container { text-align: center; background: white; padding: 40px 60px; border-radius: 16px; 
                     box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
        .icon { font-size: 64px; margin-bottom: 20px; }
        h1 { color: #1a1a2e; margin: 0 0 10px 0; }
        p { color: #666; margin: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">❌</div>
        <h1>Authorization Failed</h1>
        <p>%s</p>
        <p style="margin-top: 20px;">Please close this window and try again.</p>
    </div>
</body>
</html>`, escapedMessage)
	w.Write([]byte(html))
}
