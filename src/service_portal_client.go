package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultServicePortalURL is the default service portal address.
const DefaultServicePortalURL = "https://service.vantagedata.chat"

// ServicePortalClient is the HTTP client for communicating with the service portal.
type ServicePortalClient struct {
	ServerURL        string
	LicenseServerURL string
	client           *http.Client
}

// NewServicePortalClient creates a new ServicePortalClient with the given server URL.
func NewServicePortalClient(serverURL string) *ServicePortalClient {
	if serverURL == "" {
		serverURL = DefaultServicePortalURL
	}
	return &ServicePortalClient{
		ServerURL:        serverURL,
		LicenseServerURL: DefaultLicenseServerURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}
// BuildTicketLoginURL constructs the ticket-login URL for the given login ticket.
func BuildTicketLoginURL(ticket string) string {
	return DefaultServicePortalURL + "/auth/ticket-login?ticket=" + ticket
}

// LoginResult represents the response from the service portal sn-login endpoint.
type LoginResult struct {
	Success     bool   `json:"success"`
	LoginTicket string `json:"login_ticket,omitempty"`
	Message     string `json:"message,omitempty"`
}

// ServicePortalLogin performs the SSO login flow for the service portal.
// It returns the full ticket-login URL to be opened in the browser.
func (a *App) ServicePortalLogin() (string, error) {
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

	spc := NewServicePortalClient("")

	// Step 1: POST to License Server /api/marketplace-auth to get auth_token
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

	// Step 2: POST to Service Portal /api/auth/sn-login to get login_ticket
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

	// Step 3: Construct and return the ticket-login URL
	return BuildTicketLoginURL(loginResult.LoginTicket), nil
}
