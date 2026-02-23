package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: storefront-decoration-billing, Property 1: 装修费用上限范围约束
// **Validates: Requirements 1.1, 1.2, 1.3**
//
// For any integer value, handleSetDecorationFeeMax succeeds only when 0 ≤ value ≤ 1000,
// otherwise returns 400 and decoration_fee_max remains unchanged.
func TestDecorationBillingProperty1_FeeMaxRangeConstraint(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate a random value in range [-500, 1500]
		value := rng.Intn(2001) - 500

		// Record original value
		original := getSetting("decoration_fee_max")

		form := url.Values{}
		form.Set("value", strconv.Itoa(value))
		req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/decoration-fee-max", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handleSetDecorationFeeMax(rr, req)

		if value >= 0 && value <= 1000 {
			if rr.Code != http.StatusOK {
				t.Logf("FAIL: value=%d should succeed, got status %d", value, rr.Code)
				return false
			}
			stored := getSetting("decoration_fee_max")
			if stored != strconv.Itoa(value) {
				t.Logf("FAIL: value=%d stored as %q", value, stored)
				return false
			}
		} else {
			if rr.Code != http.StatusBadRequest {
				t.Logf("FAIL: value=%d should fail, got status %d", value, rr.Code)
				return false
			}
			stored := getSetting("decoration_fee_max")
			if stored != original {
				t.Logf("FAIL: value=%d changed setting from %q to %q", value, original, stored)
				return false
			}
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 2: 装修费用不超过上限
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4**
//
// For any fee and max, handleSetDecorationFee succeeds only when 0 ≤ fee ≤ max,
// otherwise returns 400 and decoration_fee remains unchanged.
func TestDecorationBillingProperty2_FeeNotExceedMax(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Set a random max first (valid range)
		maxVal := rng.Intn(1001) // 0-1000
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee_max', ?)", strconv.Itoa(maxVal))

		// Try to set a fee in range [-100, maxVal+200]
		fee := rng.Intn(maxVal+301) - 100

		original := getSetting("decoration_fee")

		form := url.Values{}
		form.Set("value", strconv.Itoa(fee))
		req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/decoration-fee", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handleSetDecorationFee(rr, req)

		if fee >= 0 && fee <= maxVal {
			if rr.Code != http.StatusOK {
				t.Logf("FAIL: fee=%d max=%d should succeed, got status %d", fee, maxVal, rr.Code)
				return false
			}
			stored := getSetting("decoration_fee")
			if stored != strconv.Itoa(fee) {
				t.Logf("FAIL: fee=%d stored as %q", fee, stored)
				return false
			}
		} else {
			if rr.Code != http.StatusBadRequest {
				t.Logf("FAIL: fee=%d max=%d should fail, got status %d", fee, maxVal, rr.Code)
				return false
			}
			stored := getSetting("decoration_fee")
			if stored != original {
				t.Logf("FAIL: fee=%d changed setting from %q to %q", fee, original, stored)
				return false
			}
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 3: 上限调低时费用级联调整
// **Validates: Requirements 1.5**
//
// When newMax < currentFee, setting the max cascades decoration_fee down to newMax.
func TestDecorationBillingProperty3_CascadeAdjustDown(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Set initial max and fee where fee > 0
		initialMax := rng.Intn(1000) + 1 // 1-1000
		fee := rng.Intn(initialMax) + 1   // 1 to initialMax
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee_max', ?)", strconv.Itoa(initialMax))
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', ?)", strconv.Itoa(fee))

		// Set new max lower than fee
		newMax := rng.Intn(fee) // 0 to fee-1

		form := url.Values{}
		form.Set("value", strconv.Itoa(newMax))
		req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/decoration-fee-max", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handleSetDecorationFeeMax(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: setting max=%d should succeed, got status %d", newMax, rr.Code)
			return false
		}

		storedMax := getSetting("decoration_fee_max")
		storedFee := getSetting("decoration_fee")
		if storedMax != strconv.Itoa(newMax) {
			t.Logf("FAIL: max should be %d, got %q", newMax, storedMax)
			return false
		}
		if storedFee != strconv.Itoa(newMax) {
			t.Logf("FAIL: fee should cascade to %d, got %q", newMax, storedFee)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 4: 上限调高时费用不变
// **Validates: Requirements 1.6**
//
// When newMax ≥ currentFee, setting the max does not change decoration_fee.
func TestDecorationBillingProperty4_NoChangeWhenMaxIncreased(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Set initial fee
		fee := rng.Intn(501) // 0-500
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', ?)", strconv.Itoa(fee))
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee_max', ?)", strconv.Itoa(fee+rng.Intn(501)))

		// Set new max >= fee
		newMax := fee + rng.Intn(1001-fee) // fee to 1000

		form := url.Values{}
		form.Set("value", strconv.Itoa(newMax))
		req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/decoration-fee-max", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handleSetDecorationFeeMax(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: setting max=%d should succeed, got status %d", newMax, rr.Code)
			return false
		}

		storedFee := getSetting("decoration_fee")
		if storedFee != strconv.Itoa(fee) {
			t.Logf("FAIL: fee should remain %d, got %q", fee, storedFee)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 5: 获取装修费用 API 返回完整信息
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4**
//
// GET /api/decoration-fee returns fee and max consistent with settings table, using defaults when unset.
func TestDecorationBillingProperty5_GetFeeReturnsComplete(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Randomly decide whether to set fee and max
		setFee := rng.Intn(2) == 1
		setMax := rng.Intn(2) == 1

		expectedFee := "0"
		expectedMax := "1000"

		if setMax {
			maxVal := rng.Intn(1001)
			db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee_max', ?)", strconv.Itoa(maxVal))
			expectedMax = strconv.Itoa(maxVal)
		}
		if setFee {
			maxVal, _ := strconv.Atoi(expectedMax)
			feeVal := rng.Intn(maxVal + 1)
			db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', ?)", strconv.Itoa(feeVal))
			expectedFee = strconv.Itoa(feeVal)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/decoration-fee", nil)
		rr := httptest.NewRecorder()
		handleGetDecorationFee(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: GET should return 200, got %d", rr.Code)
			return false
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("FAIL: invalid JSON response: %v", err)
			return false
		}

		if fmt.Sprintf("%v", resp["fee"]) != expectedFee {
			t.Logf("FAIL: fee expected %q, got %v", expectedFee, resp["fee"])
			return false
		}
		if fmt.Sprintf("%v", resp["max"]) != expectedMax {
			t.Logf("FAIL: max expected %q, got %v", expectedMax, resp["max"])
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 6: 发布装修扣费一致性
// **Validates: Requirements 4.1, 4.2**
//
// When fee > 0 and balance >= fee, publishing deducts exactly fee from balance
// and creates a decoration transaction record.
func TestDecorationBillingProperty6_PublishDeductionConsistency(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		fee := rng.Intn(1000) + 1 // 1-1000
		balance := float64(fee + rng.Intn(500)) // balance >= fee

		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', ?)", strconv.Itoa(fee))

		userID := createTestUserWithBalance(t, balance)

		req := httptest.NewRequest(http.MethodPost, "/user/storefront/decoration/publish", nil)
		req.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		rr := httptest.NewRecorder()
		handlePublishDecoration(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["ok"] != true {
			t.Logf("FAIL: publish should succeed, got %v", resp)
			return false
		}

		// Check balance decreased by exactly fee
		newBalance := getWalletBalance(userID)
		expected := balance - float64(fee)
		if newBalance != expected {
			t.Logf("FAIL: balance should be %.0f, got %.0f (was %.0f, fee=%d)", expected, newBalance, balance, fee)
			return false
		}

		// Check transaction record exists
		var count int
		db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ? AND transaction_type = 'decoration'", userID).Scan(&count)
		if count != 1 {
			t.Logf("FAIL: expected 1 decoration transaction, got %d", count)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 7: 余额不足拒绝发布
// **Validates: Requirements 4.3**
//
// When fee > 0 and balance < fee, publish is rejected, balance unchanged, no transaction.
func TestDecorationBillingProperty7_InsufficientBalanceRejection(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		fee := rng.Intn(1000) + 1 // 1-1000
		balance := float64(rng.Intn(fee)) // 0 to fee-1

		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', ?)", strconv.Itoa(fee))

		userID := createTestUserWithBalance(t, balance)

		req := httptest.NewRequest(http.MethodPost, "/user/storefront/decoration/publish", nil)
		req.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		rr := httptest.NewRecorder()
		handlePublishDecoration(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["ok"] != false {
			t.Logf("FAIL: publish should be rejected, got %v", resp)
			return false
		}
		if resp["error"] != "insufficient_balance" {
			t.Logf("FAIL: error should be insufficient_balance, got %v", resp["error"])
			return false
		}

		// Balance unchanged
		newBalance := getWalletBalance(userID)
		if newBalance != balance {
			t.Logf("FAIL: balance should remain %.0f, got %.0f", balance, newBalance)
			return false
		}

		// No transaction
		var count int
		db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ? AND transaction_type = 'decoration'", userID).Scan(&count)
		if count != 0 {
			t.Logf("FAIL: expected 0 decoration transactions, got %d", count)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 8: 免费装修不扣费
// **Validates: Requirements 4.4**
//
// When fee is 0, publishing does not change balance and fee_charged is 0.
func TestDecorationBillingProperty8_FreeDecorationNoCharge(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Fee is 0 (either unset or explicitly 0)
		if rng.Intn(2) == 1 {
			db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee', '0')")
		}
		// else leave unset (default 0)

		balance := float64(rng.Intn(2000))
		userID := createTestUserWithBalance(t, balance)

		req := httptest.NewRequest(http.MethodPost, "/user/storefront/decoration/publish", nil)
		req.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		rr := httptest.NewRecorder()
		handlePublishDecoration(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["ok"] != true {
			t.Logf("FAIL: free publish should succeed, got %v", resp)
			return false
		}

		feeCharged, _ := resp["fee_charged"].(float64)
		if feeCharged != 0 {
			t.Logf("FAIL: fee_charged should be 0, got %v", resp["fee_charged"])
			return false
		}

		newBalance := getWalletBalance(userID)
		if newBalance != balance {
			t.Logf("FAIL: balance should remain %.0f, got %.0f", balance, newBalance)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 9: 装修费用设置 Round-Trip
// **Validates: Requirements 2.1, 8.3**
//
// For any valid fee (0 ≤ fee ≤ max), saving then reading returns the same value.
func TestDecorationBillingProperty9_FeeSettingRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		maxVal := rng.Intn(1001) // 0-1000
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('decoration_fee_max', ?)", strconv.Itoa(maxVal))

		fee := rng.Intn(maxVal + 1) // 0 to maxVal

		// Set fee via handler
		form := url.Values{}
		form.Set("value", strconv.Itoa(fee))
		req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/decoration-fee", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handleSetDecorationFee(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: set fee=%d max=%d should succeed, got %d", fee, maxVal, rr.Code)
			return false
		}

		// Read via GET handler
		req2 := httptest.NewRequest(http.MethodGet, "/api/decoration-fee", nil)
		rr2 := httptest.NewRecorder()
		handleGetDecorationFee(rr2, req2)

		var resp map[string]interface{}
		json.Unmarshal(rr2.Body.Bytes(), &resp)

		if fmt.Sprintf("%v", resp["fee"]) != strconv.Itoa(fee) {
			t.Logf("FAIL: round-trip fee expected %d, got %v", fee, resp["fee"])
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 violated: %v", err)
	}
}

// Feature: storefront-decoration-billing, Property 10: 装修费用上限设置 Round-Trip
// **Validates: Requirements 1.1, 8.4**
//
// For any valid max (0 ≤ max ≤ 1000), saving then reading returns the same value.
func TestDecorationBillingProperty10_FeeMaxSettingRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		maxVal := rng.Intn(1001) // 0-1000

		form := url.Values{}
		form.Set("value", strconv.Itoa(maxVal))
		req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/decoration-fee-max", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handleSetDecorationFeeMax(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: set max=%d should succeed, got %d", maxVal, rr.Code)
			return false
		}

		// Read via GET handler
		req2 := httptest.NewRequest(http.MethodGet, "/api/decoration-fee", nil)
		rr2 := httptest.NewRecorder()
		handleGetDecorationFee(rr2, req2)

		var resp map[string]interface{}
		json.Unmarshal(rr2.Body.Bytes(), &resp)

		if fmt.Sprintf("%v", resp["max"]) != strconv.Itoa(maxVal) {
			t.Logf("FAIL: round-trip max expected %d, got %v", maxVal, resp["max"])
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 10 violated: %v", err)
	}
}
