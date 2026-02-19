package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"testing/quick"
	mrand "math/rand"
	"time"
)

// generateShareToken creates a random URL-safe token (same logic as server).
func generateShareToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateShareURL generates a Share URL from a share_token.
func generateShareURL(shareToken string) string {
	return fmt.Sprintf("https://market.vantagics.com/pack/%s", shareToken)
}

// parseShareTokenFromURL extracts the share_token from a Share URL path.
func parseShareTokenFromURL(url string) (string, error) {
	const prefix = "/pack/"
	idx := strings.Index(url, prefix)
	if idx == -1 {
		return "", fmt.Errorf("URL does not contain /pack/ path")
	}
	token := url[idx+len(prefix):]
	// Remove any trailing slash or query string
	if i := strings.IndexAny(token, "/?#"); i != -1 {
		token = token[:i]
	}
	if token == "" {
		return "", fmt.Errorf("empty share token")
	}
	return token, nil
}

// Feature: qap-share-url, Property 1: Share URL 往返一致性
// **Validates: Requirements 9.1, 9.2, 1.4**
func TestProperty1_ShareURLRoundtripConsistency(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 200,
		Rand:     mrand.New(mrand.NewSource(time.Now().UnixNano())),
	}

	f := func(_ uint64) bool {
		// Generate a random share token
		shareToken := generateShareToken()

		// Generate the Share URL
		shareURL := generateShareURL(shareToken)

		// Parse the share_token back from the URL
		parsed, err := parseShareTokenFromURL(shareURL)
		if err != nil {
			t.Logf("shareToken=%s: parse failed: %v", shareToken, err)
			return false
		}

		// Property: roundtrip must preserve the share_token
		if parsed != shareToken {
			t.Logf("shareToken=%s: roundtrip mismatch: parsed=%s", shareToken, parsed)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (Share URL roundtrip consistency) failed: %v", err)
	}
}
