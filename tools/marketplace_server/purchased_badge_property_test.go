package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"
)

// Feature: marketplace-purchased-badge, Property 1: 已购买集合正确性
// **Validates: Requirements 1.3, 1.4**
//
// For any user and any set of published pack listings, the set of listing IDs
// where `purchased` is true should equal the union of that user's listing_id set
// from the user_downloads table and the listing_id set (where listing_id IS NOT NULL)
// from the credits_transactions table.
func TestProperty1_PurchasedSetCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 10,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES ('BadgeTestCat', 'badge test category')")
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = 'BadgeTestCat'").Scan(&categoryID)

		// Create a test user
		username := fmt.Sprintf("badge_user_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword("test123")
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-BADGE-%d", seed), username, email, username, hashed, 500.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Generate a random number of pack listings (1-8)
		numListings := r.Intn(8) + 1
		listingIDs := make([]int64, 0, numListings)
		for i := 0; i < numListings; i++ {
			packName := fmt.Sprintf("BadgePack_%d_%d", seed, i)
			listRes, err := db.Exec(
				`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, valid_days)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', 0)`,
				userID, categoryID, []byte("fake-data"), packName, "desc", "Src", "Author", "per_use", r.Intn(100)+1,
			)
			if err != nil {
				t.Logf("seed=%d: failed to create listing %d: %v", seed, i, err)
				return false
			}
			lid, _ := listRes.LastInsertId()
			listingIDs = append(listingIDs, lid)
		}

		// Build expected purchased set by randomly assigning downloads and transactions
		expectedPurchased := make(map[int64]bool)

		// Randomly add some listings to user_downloads
		downloadSet := make(map[int64]bool)
		for _, lid := range listingIDs {
			if r.Intn(3) == 0 { // ~33% chance
				_, err := db.Exec(
					"INSERT INTO user_downloads (user_id, listing_id, downloaded_at) VALUES (?, ?, datetime('now'))",
					userID, lid,
				)
				if err != nil {
					t.Logf("seed=%d: failed to insert download for listing %d: %v", seed, lid, err)
					return false
				}
				downloadSet[lid] = true
				expectedPurchased[lid] = true
			}
		}

		// Randomly add some listings to credits_transactions
		transactionSet := make(map[int64]bool)
		txTypes := []string{"download", "purchase_uses", "renew"}
		for _, lid := range listingIDs {
			if r.Intn(3) == 0 { // ~33% chance
				txType := txTypes[r.Intn(len(txTypes))]
				_, err := db.Exec(
					"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
					userID, txType, -10.0, lid, fmt.Sprintf("purchase %d", lid),
				)
				if err != nil {
					t.Logf("seed=%d: failed to insert transaction for listing %d: %v", seed, lid, err)
					return false
				}
				transactionSet[lid] = true
				expectedPurchased[lid] = true
			}
		}

		// Also add some credits_transactions with listing_id = NULL (should NOT appear in purchased set)
		if r.Intn(2) == 0 {
			_, _ = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, created_at) VALUES (?, ?, ?, NULL, ?, datetime('now'))",
				userID, "topup", 100.0, "credits topup",
			)
		}

		// Call getUserPurchasedListingIDs
		actualPurchased := getUserPurchasedListingIDs(userID)

		// Verify: actual purchased set == expected purchased set (union of downloads and transactions)
		// Check no extra IDs in actual
		for lid := range actualPurchased {
			if !expectedPurchased[lid] {
				t.Logf("seed=%d: listing %d in actual purchased set but not expected", seed, lid)
				return false
			}
		}
		// Check no missing IDs from expected
		for lid := range expectedPurchased {
			if !actualPurchased[lid] {
				t.Logf("seed=%d: listing %d expected in purchased set but not found", seed, lid)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (已购买集合正确性) failed: %v", err)
	}
}

// Feature: marketplace-purchased-badge, Property 2: 未认证用户全部为未购买
// **Validates: Requirements 1.2**
//
// For any set of published pack listings, when the request does NOT carry an
// Authorization header, ALL Pack_Listing entries in the response should have
// `purchased` set to false.
func TestProperty2_UnauthenticatedUsersAllUnpurchased(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 10,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES ('UnauthCat', 'unauth test category')")
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = 'UnauthCat'").Scan(&categoryID)

		// Create a user who has purchases (to ensure records exist in DB)
		username := fmt.Sprintf("unauth_owner_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword("test123")
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-UNAUTH-%d", seed), username, email, username, hashed, 500.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Generate random published listings (1-10)
		numListings := r.Intn(10) + 1
		listingIDs := make([]int64, 0, numListings)
		for i := 0; i < numListings; i++ {
			packName := fmt.Sprintf("UnauthPack_%d_%d", seed, i)
			listRes, err := db.Exec(
				`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, valid_days)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', 0)`,
				userID, categoryID, []byte("fake-data"), packName, "desc", "Src", "Author", "per_use", r.Intn(100)+1,
			)
			if err != nil {
				t.Logf("seed=%d: failed to create listing %d: %v", seed, i, err)
				return false
			}
			lid, _ := listRes.LastInsertId()
			listingIDs = append(listingIDs, lid)
		}

		// Randomly add download and transaction records for the user
		// (these should NOT affect the unauthenticated response)
		txTypes := []string{"download", "purchase_uses", "renew"}
		for _, lid := range listingIDs {
			if r.Intn(2) == 0 {
				db.Exec(
					"INSERT INTO user_downloads (user_id, listing_id, downloaded_at) VALUES (?, ?, datetime('now'))",
					userID, lid,
				)
			}
			if r.Intn(2) == 0 {
				txType := txTypes[r.Intn(len(txTypes))]
				db.Exec(
					"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
					userID, txType, -10.0, lid, fmt.Sprintf("purchase %d", lid),
				)
			}
		}

		// Make request WITHOUT Authorization header
		req := httptest.NewRequest(http.MethodGet, "/api/packs", nil)
		rr := httptest.NewRecorder()
		handleListPacks(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d", seed, rr.Code)
			return false
		}

		// Parse response
		var resp struct {
			Packs []struct {
				ID        int64  `json:"id"`
				PackName  string `json:"pack_name"`
				Purchased bool   `json:"purchased"`
			} `json:"packs"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("seed=%d: failed to parse response: %v", seed, err)
			return false
		}

		// Verify we got listings back
		if len(resp.Packs) != numListings {
			t.Logf("seed=%d: expected %d listings, got %d", seed, numListings, len(resp.Packs))
			return false
		}

		// Property: ALL listings must have purchased == false
		for _, pack := range resp.Packs {
			if pack.Purchased {
				t.Logf("seed=%d: listing %d (%s) has purchased=true for unauthenticated request", seed, pack.ID, pack.PackName)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (未认证用户全部为未购买) failed: %v", err)
	}
}
