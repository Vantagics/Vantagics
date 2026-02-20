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

		// Property: URL must contain the ticket parameter
		expectedPrefix := "https://service.vantagics.com/auth/ticket-login?ticket="
		if !strings.HasPrefix(url, expectedPrefix) {
			t.Logf("seed=%d: URL prefix mismatch: got %q", seed, url)
			return false
		}

		// Property: URL must contain the ticket string followed by redirect parameter
		if !strings.Contains(url, "ticket="+ticketStr+"&redirect=") {
			t.Logf("seed=%d: URL does not contain expected ticket param: got %q", seed, url)
			return false
		}

		// Property: URL must contain the redirect to /?vantagics
		if !strings.Contains(url, "redirect=") {
			t.Logf("seed=%d: URL missing redirect param: got %q", seed, url)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (ticket-login URL construction) failed: %v", err)
	}
}
