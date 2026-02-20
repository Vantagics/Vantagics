package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultServicePortalURL is the default service portal address.
const DefaultServicePortalURL = "https://service.vantagics.com"

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
	return DefaultServicePortalURL + "/auth/ticket-login?ticket=" + ticket + "&redirect=" + url.QueryEscape(DefaultServicePortalURL+"/?vantagics")
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
	if a.marketplaceFacadeService == nil {
		return "", WrapError("App", "ServicePortalLogin", fmt.Errorf("marketplace facade service not initialized"))
	}
	return a.marketplaceFacadeService.ServicePortalLogin()
}
