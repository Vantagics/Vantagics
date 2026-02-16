package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Feature: marketplace-billing-management, Property 1: 帐单记录按时间降序排列
// For any user and any number of transaction records, querying billing records
// should return them sorted by created_at in descending order (newest first).
// Validates: Requirements 1.1
func TestProperty1_BillingRecordsSortedByTimeDesc(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Create a user with random balance
		balance := float64(rng.Intn(10000) + 100)
		userID := createTestUserWithBalance(t, balance)

		// Generate a random number of transaction records (1 to 20)
		numRecords := rng.Intn(20) + 1

		// Create a category for pack listings
		_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", "test-category")
		if err != nil {
			t.Fatalf("iteration %d: failed to create category: %v", i, err)
		}

		// Insert transaction records with varying timestamps
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		txTypes := []string{"purchase", "download", "purchase_uses", "renew", "topup"}

		for j := 0; j < numRecords; j++ {
			// Random offset in seconds (0 to 365 days) to create varied timestamps
			offsetSeconds := rng.Intn(365 * 24 * 3600)
			txTime := baseTime.Add(time.Duration(offsetSeconds) * time.Second)
			txType := txTypes[rng.Intn(len(txTypes))]
			amount := float64(rng.Intn(1000)+1) * -1
			if txType == "topup" {
				amount = float64(rng.Intn(1000) + 1)
			}
			description := fmt.Sprintf("test transaction %d", j)

			_, err := db.Exec(`
				INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at)
				VALUES (?, ?, ?, ?, ?)
			`, userID, txType, amount, description, txTime.Format("2006-01-02 15:04:05"))
			if err != nil {
				t.Fatalf("iteration %d: failed to insert transaction %d: %v", i, j, err)
			}
		}

		// Query using the same SQL as handleUserBilling
		rows, err := db.Query(`
			SELECT ct.id, ct.transaction_type, ct.amount, ct.description, ct.created_at,
			       COALESCE(pl.pack_name, '') as pack_name
			FROM credits_transactions ct
			LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
			WHERE ct.user_id = ?
			ORDER BY ct.created_at DESC
		`, userID)
		if err != nil {
			t.Fatalf("iteration %d: failed to query billing records: %v", i, err)
		}

		var records []BillingRecord
		for rows.Next() {
			var rec BillingRecord
			var desc sql.NullString
			if err := rows.Scan(&rec.ID, &rec.TransactionType, &rec.Amount, &desc, &rec.CreatedAt, &rec.PackName); err != nil {
				t.Fatalf("iteration %d: failed to scan row: %v", i, err)
			}
			if desc.Valid {
				rec.Description = desc.String
			}
			records = append(records, rec)
		}
		rows.Close()

		// Verify we got the expected number of records
		if len(records) != numRecords {
			t.Errorf("iteration %d: expected %d records, got %d", i, numRecords, len(records))
			cleanup()
			continue
		}

		// Verify records are sorted by created_at DESC (newest first)
		for k := 1; k < len(records); k++ {
			if records[k-1].CreatedAt < records[k].CreatedAt {
				t.Errorf("iteration %d: records not sorted DESC at index %d: %q < %q",
					i, k, records[k-1].CreatedAt, records[k].CreatedAt)
				break
			}
		}

		cleanup()
	}
}

// Feature: marketplace-billing-management, Property 2: 按次收费续费扣费一致性
// For any per_use pack with credits_price (1-100) and any quantity (1-10),
// the renewal operation should deduct exactly credits_price × quantity from user balance
// and record a 'purchase_uses' type transaction with amount = -(credits_price × quantity).
// Validates: Requirements 2.3, 2.5
func TestProperty2_PerUseRenewalDeductionConsistency(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Random credits_price (1-100) and quantity (1-10)
		creditsPrice := rng.Intn(100) + 1
		quantity := rng.Intn(10) + 1
		expectedCost := creditsPrice * quantity

		// Create user with enough balance (add extra to ensure sufficient)
		initialBalance := float64(expectedCost + rng.Intn(1000) + 1)
		userID := createTestUserWithBalance(t, initialBalance)

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", "test-category")
		if err != nil {
			t.Fatalf("iteration %d: failed to create category: %v", i, err)
		}

		// Create a per_use pack listing with the random credits_price
		listingID := createTestPackListing(t, userID, 1, "per_use", creditsPrice, []byte("test-data"))

		// Build POST form request to handleUserRenewPerUse
		form := fmt.Sprintf("listing_id=%d&quantity=%d", listingID, quantity)
		req := httptest.NewRequest(http.MethodPost, "/user/pack/renew-uses", strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()

		handleUserRenewPerUse(rr, req)

		// Verify the handler redirected with success
		if rr.Code != http.StatusFound {
			t.Errorf("iteration %d: expected status 302, got %d", i, rr.Code)
			cleanup()
			continue
		}
		location := rr.Header().Get("Location")
		if !strings.Contains(location, "success=renew_uses") {
			t.Errorf("iteration %d: expected success redirect, got location: %s", i, location)
			cleanup()
			continue
		}

		// Verify: balance decreased by exactly credits_price × quantity
		var newBalance float64
		err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&newBalance)
		if err != nil {
			t.Fatalf("iteration %d: failed to query balance: %v", i, err)
		}
		expectedBalance := initialBalance - float64(expectedCost)
		if newBalance != expectedBalance {
			t.Errorf("iteration %d: balance mismatch: price=%d, qty=%d, initial=%.2f, expected=%.2f, got=%.2f",
				i, creditsPrice, quantity, initialBalance, expectedBalance, newBalance)
		}

		// Verify: credits_transactions has a 'purchase_uses' record with correct amount
		var txAmount float64
		var txType string
		err = db.QueryRow(`
			SELECT transaction_type, amount FROM credits_transactions
			WHERE user_id = ? AND listing_id = ? AND transaction_type = 'purchase_uses'
			ORDER BY created_at DESC LIMIT 1
		`, userID, listingID).Scan(&txType, &txAmount)
		if err != nil {
			t.Fatalf("iteration %d: failed to query transaction: %v", i, err)
		}
		if txType != "purchase_uses" {
			t.Errorf("iteration %d: expected transaction_type 'purchase_uses', got '%s'", i, txType)
		}
		expectedAmount := -float64(expectedCost)
		if txAmount != expectedAmount {
			t.Errorf("iteration %d: transaction amount mismatch: price=%d, qty=%d, expected=%.2f, got=%.2f",
				i, creditsPrice, quantity, expectedAmount, txAmount)
		}

		cleanup()
	}
}


// Feature: marketplace-billing-management, Property 3: 订阅制续费扣费一致性
// For any subscription pack with credits_price (100-1000) and renewal duration (1 or 12 months):
// - Monthly renewal: deduct credits_price × months
// - Yearly renewal: deduct credits_price × 12 (but grant 14 months)
// - Record a 'renew' type transaction with amount = -(credits_price × months)
// Validates: Requirements 3.3, 3.4, 3.6
func TestProperty3_SubscriptionRenewalDeductionConsistency(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Random credits_price (100-1000) and months (1 or 12)
		creditsPrice := rng.Intn(901) + 100
		months := 1
		if rng.Intn(2) == 1 {
			months = 12
		}
		expectedCost := creditsPrice * months

		// Create user with enough balance
		initialBalance := float64(expectedCost + rng.Intn(5000) + 1)
		userID := createTestUserWithBalance(t, initialBalance)

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", "test-category")
		if err != nil {
			t.Fatalf("iteration %d: failed to create category: %v", i, err)
		}

		// Create a subscription pack listing with the random credits_price
		listingID := createTestPackListing(t, userID, 1, "subscription", creditsPrice, []byte("test-data"))

		// Build POST form request to handleUserRenewSubscription
		form := fmt.Sprintf("listing_id=%d&months=%d", listingID, months)
		req := httptest.NewRequest(http.MethodPost, "/user/pack/renew-subscription", strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()

		handleUserRenewSubscription(rr, req)

		// Verify the handler redirected with success
		if rr.Code != http.StatusFound {
			t.Errorf("iteration %d: expected status 302, got %d", i, rr.Code)
			cleanup()
			continue
		}
		location := rr.Header().Get("Location")
		if !strings.Contains(location, "success=renew_subscription") {
			t.Errorf("iteration %d: expected success redirect, got location: %s", i, location)
			cleanup()
			continue
		}

		// Verify: balance decreased by exactly credits_price × months
		var newBalance float64
		err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&newBalance)
		if err != nil {
			t.Fatalf("iteration %d: failed to query balance: %v", i, err)
		}
		expectedBalance := initialBalance - float64(expectedCost)
		if newBalance != expectedBalance {
			t.Errorf("iteration %d: balance mismatch: price=%d, months=%d, initial=%.2f, expected=%.2f, got=%.2f",
				i, creditsPrice, months, initialBalance, expectedBalance, newBalance)
		}

		// Verify: credits_transactions has a 'renew' record with correct amount
		var txAmount float64
		var txType string
		err = db.QueryRow(`
			SELECT transaction_type, amount FROM credits_transactions
			WHERE user_id = ? AND listing_id = ? AND transaction_type = 'renew'
			ORDER BY created_at DESC LIMIT 1
		`, userID, listingID).Scan(&txType, &txAmount)
		if err != nil {
			t.Fatalf("iteration %d: failed to query transaction: %v", i, err)
		}
		if txType != "renew" {
			t.Errorf("iteration %d: expected transaction_type 'renew', got '%s'", i, txType)
		}
		expectedAmount := -float64(expectedCost)
		if txAmount != expectedAmount {
			t.Errorf("iteration %d: transaction amount mismatch: price=%d, months=%d, expected=%.2f, got=%.2f",
				i, creditsPrice, months, expectedAmount, txAmount)
		}

		cleanup()
	}
}

// Feature: marketplace-billing-management, Property 4: 余额不足拒绝续费
// For any user and any paid pack, when user Credits balance < total renewal cost:
// 1. The renewal operation should be rejected
// 2. User balance should remain unchanged
// 3. No transaction record should be created
// Validates: Requirements 2.4, 3.5
func TestProperty4_InsufficientBalance(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Randomly choose per_use or subscription mode
		isPerUse := rng.Intn(2) == 0

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", "test-category")
		if err != nil {
			t.Fatalf("iteration %d: failed to create category: %v", i, err)
		}

		if isPerUse {
			// per_use: random price (1-100) and quantity (1-10)
			creditsPrice := rng.Intn(100) + 1
			quantity := rng.Intn(10) + 1
			totalCost := creditsPrice * quantity

			// Create user with balance strictly less than totalCost
			var userBalance float64
			if totalCost > 1 {
				userBalance = float64(rng.Intn(totalCost))
			} else {
				userBalance = 0
			}
			userID := createTestUserWithBalance(t, userBalance)
			listingID := createTestPackListing(t, userID, 1, "per_use", creditsPrice, []byte("test-data"))

			// Count existing transactions before the call
			var txCountBefore int
			err = db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ?", userID).Scan(&txCountBefore)
			if err != nil {
				t.Fatalf("iteration %d: failed to count transactions: %v", i, err)
			}

			// Call handleUserRenewPerUse
			form := fmt.Sprintf("listing_id=%d&quantity=%d", listingID, quantity)
			req := httptest.NewRequest(http.MethodPost, "/user/pack/renew-uses", strings.NewReader(form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()

			handleUserRenewPerUse(rr, req)

			// Verify 1: redirect contains "error=insufficient_credits"
			if rr.Code != http.StatusFound {
				t.Errorf("iteration %d (per_use): expected status 302, got %d", i, rr.Code)
				cleanup()
				continue
			}
			location := rr.Header().Get("Location")
			if !strings.Contains(location, "error=insufficient_credits") {
				t.Errorf("iteration %d (per_use): expected insufficient_credits error, got location: %s (balance=%.2f, cost=%d)",
					i, location, userBalance, totalCost)
				cleanup()
				continue
			}

			// Verify 2: balance unchanged
			var newBalance float64
			err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&newBalance)
			if err != nil {
				t.Fatalf("iteration %d (per_use): failed to query balance: %v", i, err)
			}
			if newBalance != userBalance {
				t.Errorf("iteration %d (per_use): balance changed: expected=%.2f, got=%.2f", i, userBalance, newBalance)
			}

			// Verify 3: no new transaction records
			var txCountAfter int
			err = db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ?", userID).Scan(&txCountAfter)
			if err != nil {
				t.Fatalf("iteration %d (per_use): failed to count transactions: %v", i, err)
			}
			if txCountAfter != txCountBefore {
				t.Errorf("iteration %d (per_use): transaction count changed: before=%d, after=%d", i, txCountBefore, txCountAfter)
			}
		} else {
			// subscription: random price (100-1000) and months (1 or 12)
			creditsPrice := rng.Intn(901) + 100
			months := 1
			if rng.Intn(2) == 1 {
				months = 12
			}
			totalCost := creditsPrice * months

			// Create user with balance strictly less than totalCost
			var userBalance float64
			if totalCost > 1 {
				userBalance = float64(rng.Intn(totalCost))
			} else {
				userBalance = 0
			}
			userID := createTestUserWithBalance(t, userBalance)
			listingID := createTestPackListing(t, userID, 1, "subscription", creditsPrice, []byte("test-data"))

			// Count existing transactions before the call
			var txCountBefore int
			err = db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ?", userID).Scan(&txCountBefore)
			if err != nil {
				t.Fatalf("iteration %d: failed to count transactions: %v", i, err)
			}

			// Call handleUserRenewSubscription
			form := fmt.Sprintf("listing_id=%d&months=%d", listingID, months)
			req := httptest.NewRequest(http.MethodPost, "/user/pack/renew-subscription", strings.NewReader(form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()

			handleUserRenewSubscription(rr, req)

			// Verify 1: redirect contains "error=insufficient_credits"
			if rr.Code != http.StatusFound {
				t.Errorf("iteration %d (subscription): expected status 302, got %d", i, rr.Code)
				cleanup()
				continue
			}
			location := rr.Header().Get("Location")
			if !strings.Contains(location, "error=insufficient_credits") {
				t.Errorf("iteration %d (subscription): expected insufficient_credits error, got location: %s (balance=%.2f, cost=%d)",
					i, location, userBalance, totalCost)
				cleanup()
				continue
			}

			// Verify 2: balance unchanged
			var newBalance float64
			err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&newBalance)
			if err != nil {
				t.Fatalf("iteration %d (subscription): failed to query balance: %v", i, err)
			}
			if newBalance != userBalance {
				t.Errorf("iteration %d (subscription): balance changed: expected=%.2f, got=%.2f", i, userBalance, newBalance)
			}

			// Verify 3: no new transaction records
			var txCountAfter int
			err = db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ?", userID).Scan(&txCountAfter)
			if err != nil {
				t.Fatalf("iteration %d (subscription): failed to count transactions: %v", i, err)
			}
			if txCountAfter != txCountBefore {
				t.Errorf("iteration %d (subscription): transaction count changed: before=%d, after=%d", i, txCountBefore, txCountAfter)
			}
		}

		cleanup()
	}
}
