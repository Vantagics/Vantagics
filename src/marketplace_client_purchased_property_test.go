package main

import (
	"encoding/json"
	"math/rand"
	"testing"
	"testing/quick"
	"time"
)

// Feature: marketplace-purchased-badge, Property 3: PackListingInfo 反序列化保持 purchased 字段
// **Validates: Requirements 2.2**
//
// For any PackListingInfo with a random purchased boolean value, marshaling to JSON
// and unmarshaling back should preserve the Purchased field value exactly.
func TestProperty3_PackListingInfoDeserializationPreservesPurchased(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate a random PackListingInfo with random field values
		original := PackListingInfo{
			ID:              r.Int63n(100000) + 1,
			UserID:          r.Int63n(100000) + 1,
			CategoryID:      r.Int63n(100) + 1,
			CategoryName:    randomString(r, 1+r.Intn(30)),
			PackName:        randomString(r, 1+r.Intn(50)),
			PackDescription: randomString(r, r.Intn(200)),
			SourceName:      randomString(r, 1+r.Intn(20)),
			AuthorName:      randomString(r, 1+r.Intn(30)),
			ShareMode:       []string{"free", "per_use", "subscription"}[r.Intn(3)],
			CreditsPrice:    r.Intn(10000),
			DownloadCount:   r.Intn(100000),
			CreatedAt:       time.Now().Add(-time.Duration(r.Intn(365*24)) * time.Hour).Format(time.RFC3339),
			Purchased:       r.Intn(2) == 1,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Logf("seed=%d: marshal failed: %v", seed, err)
			return false
		}

		// Unmarshal back
		var decoded PackListingInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Logf("seed=%d: unmarshal failed: %v", seed, err)
			return false
		}

		// Property: Purchased field must match
		if decoded.Purchased != original.Purchased {
			t.Logf("seed=%d: Purchased mismatch: original=%v, decoded=%v", seed, original.Purchased, decoded.Purchased)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (PackListingInfo 反序列化保持 purchased 字段) failed: %v", err)
	}
}

// TestProperty3_RawJSONPurchasedFieldPreserved tests that when a raw JSON response
// contains a "purchased" boolean field, deserializing into PackListingInfo preserves it.
// This simulates receiving JSON from the server rather than round-tripping our own struct.
func TestProperty3_RawJSONPurchasedFieldPreserved(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		purchased := r.Intn(2) == 1

		// Build a raw JSON object simulating a server response
		raw := map[string]interface{}{
			"id":               r.Int63n(100000) + 1,
			"user_id":          r.Int63n(100000) + 1,
			"category_id":      r.Int63n(100) + 1,
			"category_name":    randomString(r, 1+r.Intn(20)),
			"pack_name":        randomString(r, 1+r.Intn(30)),
			"pack_description": randomString(r, r.Intn(100)),
			"source_name":      randomString(r, 1+r.Intn(15)),
			"author_name":      randomString(r, 1+r.Intn(20)),
			"share_mode":       "free",
			"credits_price":    r.Intn(5000),
			"download_count":   r.Intn(50000),
			"created_at":       time.Now().Format(time.RFC3339),
			"purchased":        purchased,
		}

		data, err := json.Marshal(raw)
		if err != nil {
			t.Logf("seed=%d: marshal raw JSON failed: %v", seed, err)
			return false
		}

		var decoded PackListingInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Logf("seed=%d: unmarshal failed: %v", seed, err)
			return false
		}

		if decoded.Purchased != purchased {
			t.Logf("seed=%d: Purchased mismatch: expected=%v, got=%v", seed, purchased, decoded.Purchased)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (Raw JSON purchased field preserved) failed: %v", err)
	}
}

// randomString generates a random alphanumeric string of the given length.
func randomString(r *rand.Rand, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
