package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// generateShareURL generates a Share URL from a listing_id.
func generateShareURL(listingID int64) string {
	return fmt.Sprintf("https://market.vantagics.com/pack/%d", listingID)
}

// parseListingIDFromURL extracts the listing_id from a Share URL path.
// Returns the parsed listing_id and an error if the path is invalid.
func parseListingIDFromURL(url string) (int64, error) {
	const prefix = "/pack/"
	idx := strings.Index(url, prefix)
	if idx == -1 {
		return 0, fmt.Errorf("URL does not contain /pack/ path")
	}
	idStr := url[idx+len(prefix):]
	// Remove any trailing slash or query string
	if i := strings.IndexAny(idStr, "/?#"); i != -1 {
		idStr = idStr[:i]
	}
	return strconv.ParseInt(idStr, 10, 64)
}

// Feature: qap-share-url, Property 1: Share URL 往返一致性
// **Validates: Requirements 9.1, 9.2, 1.4**
func TestProperty1_ShareURLRoundtripConsistency(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(raw int64) bool {
		// Constrain to positive integers (valid listing_id)
		if raw <= 0 {
			raw = -raw
		}
		if raw <= 0 {
			raw = 1
		}
		listingID := raw

		// Generate the Share URL
		shareURL := generateShareURL(listingID)

		// Parse the listing_id back from the URL
		parsed, err := parseListingIDFromURL(shareURL)
		if err != nil {
			t.Logf("listingID=%d: parse failed: %v", listingID, err)
			return false
		}

		// Property: roundtrip must preserve the listing_id
		if parsed != listingID {
			t.Logf("listingID=%d: roundtrip mismatch: parsed=%d", listingID, parsed)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (Share URL roundtrip consistency) failed: %v", err)
	}
}
