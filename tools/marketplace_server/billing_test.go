package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// Feature: marketplace-sn-auth-billing, Property 4: 计费参数验证完整性
// For any pricing_model and parameter combination, validatePricingParams correctly
// accepts complete valid parameters and rejects incomplete/invalid parameters.
// Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5, 4.6
func TestProperty4_PricingParamsValidation(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		// Generate random pricing parameters
		modes := []string{"free", "per_use", "subscription", "invalid_mode", "time_limited", ""}
		mode := modes[rng.Intn(len(modes))]
		price := rng.Intn(2000) - 500 // range: -500 to 1499

		errMsg := validatePricingParams(mode, price)

		switch mode {
		case "free":
			// free mode: should always succeed regardless of price
			if errMsg != "" {
				t.Errorf("iteration %d: free mode should accept any params, got error: %s", i, errMsg)
			}
		case "per_use":
			// per_use: needs credits_price between 1 and 100
			if price >= 1 && price <= 100 {
				if errMsg != "" {
					t.Errorf("iteration %d: per_use with valid price %d should succeed, got error: %s", i, price, errMsg)
				}
			} else {
				if errMsg == "" {
					t.Errorf("iteration %d: per_use with invalid price %d should fail", i, price)
				}
			}
		case "subscription":
			// subscription: needs credits_price between 100 and 1000
			if price >= 100 && price <= 1000 {
				if errMsg != "" {
					t.Errorf("iteration %d: subscription with valid price %d should succeed, got error: %s", i, price, errMsg)
				}
			} else {
				if errMsg == "" {
					t.Errorf("iteration %d: subscription with invalid price %d should fail", i, price)
				}
			}
		default:
			// Unknown modes should be rejected
			if errMsg == "" {
				t.Errorf("iteration %d: unknown mode %q should be rejected", i, mode)
			}
		}
	}
}

// Feature: marketplace-sn-auth-billing, Property 5: 下载 Credits 扣除一致性
// For any download operation, the user's credits balance decreases by exactly credits_price
// (0 for free), and a corresponding credits_transaction record is generated.
// Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.6
func TestProperty5_DownloadCreditsDeductionConsistency(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Generate random pricing scenario
		modes := []string{"free", "per_use", "subscription"}
		mode := modes[rng.Intn(len(modes))]
		var price int
		switch mode {
		case "free":
			price = 0
		case "per_use":
			price = rng.Intn(100) + 1 // 1-100
		case "subscription":
			price = rng.Intn(901) + 100 // 100-1000
		}

		// Create user with sufficient balance
		balance := float64(price) + float64(rng.Intn(1000))
		userID := createTestUserWithBalance(t, balance)

		// Create a published pack listing
		packID := createTestPackListing(t, userID, 1, mode, price, []byte("test-data"))

		// Download the pack
		rr := makeDownloadRequest(t, packID, userID)

		if rr.Code != 200 {
			t.Errorf("iteration %d: download failed with status %d for mode=%s price=%d balance=%f",
				i, rr.Code, mode, price, balance)
			cleanup()
			continue
		}

		// Check balance after download
		var newBalance float64
		err := db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&newBalance)
		if err != nil {
			t.Errorf("iteration %d: failed to query balance: %v", i, err)
			cleanup()
			continue
		}

		expectedBalance := balance - float64(price)
		if fmt.Sprintf("%.2f", newBalance) != fmt.Sprintf("%.2f", expectedBalance) {
			t.Errorf("iteration %d: balance mismatch for mode=%s: expected %.2f, got %.2f (initial=%.2f, price=%d)",
				i, mode, expectedBalance, newBalance, balance, price)
		}

		// Check transaction record for paid packs
		if mode != "free" {
			var txCount int
			err = db.QueryRow(
				"SELECT COUNT(*) FROM credits_transactions WHERE user_id = ? AND listing_id = ? AND transaction_type = 'download'",
				userID, packID,
			).Scan(&txCount)
			if err != nil {
				t.Errorf("iteration %d: failed to query transactions: %v", i, err)
			} else if txCount != 1 {
				t.Errorf("iteration %d: expected 1 transaction record for mode=%s, got %d", i, mode, txCount)
			}

			// Verify transaction amount
			var txAmount float64
			err = db.QueryRow(
				"SELECT amount FROM credits_transactions WHERE user_id = ? AND listing_id = ? AND transaction_type = 'download'",
				userID, packID,
			).Scan(&txAmount)
			if err != nil {
				t.Errorf("iteration %d: failed to query transaction amount: %v", i, err)
			} else if txAmount != -float64(price) {
				t.Errorf("iteration %d: transaction amount mismatch: expected %f, got %f", i, -float64(price), txAmount)
			}
		}

		cleanup()
	}
}

// Feature: marketplace-sn-auth-billing, Property 6: 余额不足拒绝下载
// For any user and paid pack, when balance < credits_price, download is rejected
// and balance remains unchanged.
// Validates: Requirements 5.5
func TestProperty6_InsufficientBalanceRejectsDownload(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Generate a paid pack with price > 0
		modes := []string{"per_use", "subscription"}
		mode := modes[rng.Intn(len(modes))]
		var price int
		switch mode {
		case "per_use":
			price = rng.Intn(100) + 1
		case "subscription":
			price = rng.Intn(901) + 100
		}

		// Create user with insufficient balance (0 to price-1)
		balance := float64(rng.Intn(price))
		userID := createTestUserWithBalance(t, balance)

		packID := createTestPackListing(t, userID, 1, mode, price, []byte("test-data"))

		rr := makeDownloadRequest(t, packID, userID)

		// Should be rejected with 402
		if rr.Code != 402 {
			t.Errorf("iteration %d: expected 402 for insufficient balance (balance=%.0f, price=%d, mode=%s), got %d",
				i, balance, price, mode, rr.Code)
		}

		// Balance should remain unchanged
		var newBalance float64
		err := db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&newBalance)
		if err != nil {
			t.Errorf("iteration %d: failed to query balance: %v", i, err)
		} else if fmt.Sprintf("%.2f", newBalance) != fmt.Sprintf("%.2f", balance) {
			t.Errorf("iteration %d: balance changed after rejected download: was %.2f, now %.2f", i, balance, newBalance)
		}

		// No transaction should be recorded
		var txCount int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM credits_transactions WHERE user_id = ? AND listing_id = ?",
			userID, packID,
		).Scan(&txCount)
		if err != nil {
			t.Errorf("iteration %d: failed to query transactions: %v", i, err)
		} else if txCount != 0 {
			t.Errorf("iteration %d: expected 0 transactions after rejected download, got %d", i, txCount)
		}

		cleanup()
	}
}
