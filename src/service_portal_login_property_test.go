package main

import (
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: service-portal-sso, Property 5: ticket-login URL 构造正确性
// **Validates: Requirements 3.4**
func TestProperty5_TicketLoginURLConstruction(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Generate random non-empty ticket strings
	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate a random non-empty ticket string
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."
		ticketLen := r.Intn(64) + 1 // 1 to 64 characters
		ticket := make([]byte, ticketLen)
		for i := range ticket {
			ticket[i] = chars[r.Intn(len(chars))]
		}
		ticketStr := string(ticket)

		url := BuildTicketLoginURL(ticketStr)

		// Property: URL must have the exact expected prefix
		expectedPrefix := "https://service.vantagedata.chat/auth/ticket-login?ticket="
		if !strings.HasPrefix(url, expectedPrefix) {
			t.Logf("seed=%d: URL prefix mismatch: got %q", seed, url)
			return false
		}

		// Property: URL must end with the exact ticket string
		if !strings.HasSuffix(url, ticketStr) {
			t.Logf("seed=%d: URL suffix mismatch: expected ticket=%q, got url=%q", seed, ticketStr, url)
			return false
		}

		// Property: URL must be exactly prefix + ticket (no extra characters)
		expected := expectedPrefix + ticketStr
		if url != expected {
			t.Logf("seed=%d: URL mismatch: expected=%q, got=%q", seed, expected, url)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (ticket-login URL construction) failed: %v", err)
	}
}
