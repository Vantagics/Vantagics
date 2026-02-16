package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Feature: pack-usage-tracking, Property 2: 服务器使用次数递增
// For any valid user and per_use pack combination, each new (non-duplicate)
// Usage_Report increments used_count by exactly 1, so N distinct reports yield used_count == N.
// **Validates: Requirements 3.1, 5.5**
func TestProperty2_ServerUsageCountIncrement(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		cleanup := setupTestDB(t)

		// Create user with balance
		userID := createTestUserWithBalance(t, 1000)
		catID := getCategoryID(t)

		// Create a per_use pack listing
		creditsPrice := rng.Intn(50) + 1
		packID := createTestPackListing(t, userID, catID, "per_use", creditsPrice, []byte("test-data"))

		// Generate random N (1-20) distinct usage reports
		n := rng.Intn(20) + 1
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		for j := 0; j < n; j++ {
			usedAt := baseTime.Add(time.Duration(j) * time.Hour).Format(time.RFC3339)
			body, _ := json.Marshal(map[string]interface{}{
				"listing_id": packID,
				"used_at":    usedAt,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/packs/report-usage", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleReportPackUsage(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("iteration %d, report %d: expected 200, got %d; body: %s", i, j, rr.Code, rr.Body.String())
				break
			}
		}

		// Verify used_count == N
		var usedCount int
		err := db.QueryRow(
			"SELECT used_count FROM pack_usage_records WHERE user_id = ? AND listing_id = ?",
			userID, packID,
		).Scan(&usedCount)
		if err != nil {
			t.Errorf("iteration %d: failed to query used_count: %v", i, err)
		} else if usedCount != n {
			t.Errorf("iteration %d: expected used_count=%d, got %d", i, n, usedCount)
		}

		cleanup()
	}
}

// Feature: pack-usage-tracking, Property 3: 服务器重复上报幂等性
// For any Usage_Report, sending the same (user_id, listing_id, used_at) twice
// should only increment used_count by 1 total (second request is idempotent).
// **Validates: Requirements 3.3**
func TestProperty3_ServerDuplicateReportIdempotency(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		cleanup := setupTestDB(t)

		userID := createTestUserWithBalance(t, 1000)
		catID := getCategoryID(t)
		creditsPrice := rng.Intn(50) + 1
		packID := createTestPackListing(t, userID, catID, "per_use", creditsPrice, []byte("test-data"))

		// Generate a random timestamp
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		offset := rng.Intn(365 * 24)
		usedAt := baseTime.Add(time.Duration(offset) * time.Hour).Format(time.RFC3339)

		// Send the same report twice
		for attempt := 0; attempt < 2; attempt++ {
			body, _ := json.Marshal(map[string]interface{}{
				"listing_id": packID,
				"used_at":    usedAt,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/packs/report-usage", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleReportPackUsage(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("iteration %d, attempt %d: expected 200, got %d; body: %s", i, attempt, rr.Code, rr.Body.String())
				break
			}

			// Parse response to verify used_count
			var resp map[string]interface{}
			json.Unmarshal(rr.Body.Bytes(), &resp)
			usedCountFloat, ok := resp["used_count"].(float64)
			if !ok {
				t.Errorf("iteration %d, attempt %d: used_count not in response", i, attempt)
				break
			}
			// Both attempts should report used_count == 1
			if int(usedCountFloat) != 1 {
				t.Errorf("iteration %d, attempt %d: expected used_count=1, got %d", i, attempt, int(usedCountFloat))
			}
		}

		// Verify DB used_count == 1
		var usedCount int
		err := db.QueryRow(
			"SELECT used_count FROM pack_usage_records WHERE user_id = ? AND listing_id = ?",
			userID, packID,
		).Scan(&usedCount)
		if err != nil {
			t.Errorf("iteration %d: failed to query used_count: %v", i, err)
		} else if usedCount != 1 {
			t.Errorf("iteration %d: expected used_count=1 after duplicate, got %d", i, usedCount)
		}

		cleanup()
	}
}

// Feature: pack-usage-tracking, Property 4: 服务器错误处理
// For any invalid report request (missing listing_id, invalid timestamp format,
// or non-per_use listing_id), the server should return HTTP 400.
// **Validates: Requirements 5.2, 5.4**
func TestProperty4_ServerErrorHandling(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		cleanup := setupTestDB(t)

		userID := createTestUserWithBalance(t, 1000)
		catID := getCategoryID(t)

		// Create a non-per_use pack (free) for testing invalid share_mode
		freePackID := createTestPackListing(t, userID, catID, "free", 0, []byte("free-data"))

		// Pick a random error scenario
		scenario := rng.Intn(4)
		var body []byte
		switch scenario {
		case 0:
			// Missing listing_id (listing_id = 0)
			body, _ = json.Marshal(map[string]interface{}{
				"listing_id": 0,
				"used_at":    time.Now().Format(time.RFC3339),
			})
		case 1:
			// Negative listing_id
			body, _ = json.Marshal(map[string]interface{}{
				"listing_id": -(rng.Intn(1000) + 1),
				"used_at":    time.Now().Format(time.RFC3339),
			})
		case 2:
			// Invalid timestamp format
			invalidFormats := []string{
				"2024-01-01",
				"not-a-date",
				"2024/01/01 12:00:00",
				fmt.Sprintf("random-%d", rng.Int()),
				"",
			}
			body, _ = json.Marshal(map[string]interface{}{
				"listing_id": freePackID + 1000, // use a valid-looking ID
				"used_at":    invalidFormats[rng.Intn(len(invalidFormats))],
			})
		case 3:
			// Non-per_use listing_id (free pack)
			body, _ = json.Marshal(map[string]interface{}{
				"listing_id": freePackID,
				"used_at":    time.Now().Format(time.RFC3339),
			})
		}

		req := httptest.NewRequest(http.MethodPost, "/api/packs/report-usage", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleReportPackUsage(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("iteration %d (scenario %d): expected 400, got %d; body: %s",
				i, scenario, rr.Code, rr.Body.String())
		}

		cleanup()
	}
}

// Feature: pack-usage-tracking, Property 7: 续费后 total_purchased 更新
// For any random renewal quantity N, after purchasing additional uses,
// total_purchased in pack_usage_records should increase by N.
// **Validates: Requirements 3.4**
func TestProperty7_RenewalTotalPurchasedUpdate(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		cleanup := setupTestDB(t)

		// Create user with large balance to afford purchases
		userID := createTestUserWithBalance(t, 100000)
		catID := getCategoryID(t)

		// Create a per_use pack with a small price
		creditsPrice := rng.Intn(10) + 1
		packID := createTestPackListing(t, userID, catID, "per_use", creditsPrice, []byte("test-data"))

		// Generate random quantity (1-50)
		quantity := rng.Intn(50) + 1

		// Call handlePurchaseAdditionalUses
		body, _ := json.Marshal(map[string]interface{}{
			"quantity": quantity,
		})
		url := fmt.Sprintf("/api/packs/%d/purchase-uses", packID)
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handlePurchaseAdditionalUses(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("iteration %d: purchase failed with status %d; body: %s", i, rr.Code, rr.Body.String())
			cleanup()
			continue
		}

		// Verify total_purchased == quantity
		var totalPurchased int
		err := db.QueryRow(
			"SELECT total_purchased FROM pack_usage_records WHERE user_id = ? AND listing_id = ?",
			userID, packID,
		).Scan(&totalPurchased)
		if err != nil {
			t.Errorf("iteration %d: failed to query total_purchased: %v", i, err)
			cleanup()
			continue
		}
		if totalPurchased != quantity {
			t.Errorf("iteration %d: expected total_purchased=%d, got %d", i, quantity, totalPurchased)
		}

		// Now do a second purchase with another random quantity to verify accumulation
		quantity2 := rng.Intn(50) + 1
		body2, _ := json.Marshal(map[string]interface{}{
			"quantity": quantity2,
		})
		req2 := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr2 := httptest.NewRecorder()
		handlePurchaseAdditionalUses(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Errorf("iteration %d: second purchase failed with status %d; body: %s", i, rr2.Code, rr2.Body.String())
			cleanup()
			continue
		}

		// Verify total_purchased == quantity + quantity2
		err = db.QueryRow(
			"SELECT total_purchased FROM pack_usage_records WHERE user_id = ? AND listing_id = ?",
			userID, packID,
		).Scan(&totalPurchased)
		if err != nil {
			t.Errorf("iteration %d: failed to query total_purchased after second purchase: %v", i, err)
		} else if totalPurchased != quantity+quantity2 {
			t.Errorf("iteration %d: expected total_purchased=%d, got %d", i, quantity+quantity2, totalPurchased)
		}

		cleanup()
	}
}
