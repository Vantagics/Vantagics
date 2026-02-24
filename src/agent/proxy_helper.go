package agent

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"vantagics/config"
)

// NewProxyHTTPClient creates an HTTP client with optional proxy support.
// If proxyConfig is nil, disabled, or untested, returns a plain client.
func NewProxyHTTPClient(timeout time.Duration, proxyConfig *config.ProxyConfig) *http.Client {
	client := &http.Client{Timeout: timeout}

	if proxyConfig == nil {
		return client
	}

	if !proxyConfig.Enabled {
		return client
	}

	if !proxyConfig.Tested {
		return client
	}

	if proxyConfig.Host == "" || proxyConfig.Port <= 0 {
		return client
	}

	protocol := proxyConfig.Protocol
	if protocol == "" {
		protocol = "http"
	}
	proxyURL := &url.URL{
		Scheme: protocol,
		Host:   fmt.Sprintf("%s:%d", proxyConfig.Host, proxyConfig.Port),
	}
	if proxyConfig.Username != "" {
		if proxyConfig.Password != "" {
			proxyURL.User = url.UserPassword(proxyConfig.Username, proxyConfig.Password)
		} else {
			proxyURL.User = url.User(proxyConfig.Username)
		}
	}
	// Clone DefaultTransport to preserve sensible defaults (TLS timeouts, keep-alive, etc.)
	// and only override the Proxy function.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	client.Transport = transport

	return client
}
