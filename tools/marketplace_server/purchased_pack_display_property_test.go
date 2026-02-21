package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: purchased-pack-display-fix, Property 1: 付费下载交易在个人中心可见
// **Validates: Requirements 1.1, 1.2**
//
// For any user and any paid analysis pack, when the user's credits_transactions table
// contains a record with transaction_type = 'download', the dashboard's purchased pack
// list query should return that pack.
func TestProperty1_PaidDownloadTransactionVisibleInDashboard(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 10,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES ('PropTestCat', 'property test category')")
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = 'PropTestCat'").Scan(&categoryID)

		// Create a test user with some credits balance
		username := fmt.Sprintf("pbt_user_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword("test123")
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-PBT-%d", seed), username, email, username, hashed, 500.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Randomly choose transaction_type: 'download' or 'purchase'
		// The property specifically tests that 'download' type is visible,
		// but we also verify 'purchase' still works.
		txTypes := []string{"download", "purchase"}
		txType := txTypes[r.Intn(len(txTypes))]

		// Generate a random pack name
		packName := fmt.Sprintf("PaidPack_%d_%d", seed, r.Intn(10000))

		// Random credits price (1-100)
		creditsPrice := r.Intn(100) + 1

		// Random share mode for paid packs
		paidModes := []string{"per_use", "subscription", "time_limited"}
		shareMode := paidModes[r.Intn(len(paidModes))]

		validDays := 0
		if shareMode == "time_limited" || shareMode == "subscription" {
			validDays = r.Intn(365) + 1
		}

		// Create a pack listing
		listRes, err := db.Exec(
			`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, valid_days)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', ?)`,
			userID, categoryID, []byte("fake-qap-data"), packName, "test description", "TestSource", "TestAuthor", shareMode, creditsPrice, validDays,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create pack listing: %v", seed, err)
			return false
		}
		listingID, _ := listRes.LastInsertId()

		// Insert a credits_transaction with the chosen transaction_type
		_, err = db.Exec(
			"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
			userID, txType, -float64(creditsPrice), listingID, fmt.Sprintf("购买 %s", packName),
		)
		if err != nil {
			t.Logf("seed=%d: failed to insert credits_transaction: %v", seed, err)
			return false
		}

		// Call handleUserDashboard and check the response contains the pack
		req := httptest.NewRequest(http.MethodGet, "/user/dashboard", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleUserDashboard(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d; body: %s", seed, rr.Code, rr.Body.String()[:min(300, rr.Body.Len())])
			return false
		}

		body := rr.Body.String()

		// The pack should be visible in the dashboard regardless of whether
		// transaction_type is 'download' or 'purchase'
		if !strings.Contains(body, packName) {
			t.Logf("seed=%d: pack %q with transaction_type=%q not found in dashboard HTML", seed, packName, txType)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (付费下载交易在个人中心可见) failed: %v", err)
	}
}

// Feature: purchased-pack-display-fix, Property 2: 个人中心按 listing_id 去重
// **Validates: Requirements 1.3**
//
// For any user, when the same listing_id has multiple records in credits_transactions
// and user_downloads tables, the dashboard's purchased pack list should show that
// listing_id only once.
func TestProperty2_DashboardDeduplicatesByListingID(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 10,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES ('DedupeTestCat', 'dedup test category')")
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = 'DedupeTestCat'").Scan(&categoryID)

		// Create a test user
		username := fmt.Sprintf("dedup_user_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword("test123")
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-DEDUP-%d", seed), username, email, username, hashed, 500.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Create a pack listing
		packName := fmt.Sprintf("DedupPack_%d", seed)
		creditsPrice := r.Intn(100) + 1
		shareMode := "per_use"

		listRes, err := db.Exec(
			`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, valid_days)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', 0)`,
			userID, categoryID, []byte("fake-qap-data"), packName, "test description", "TestSource", "TestAuthor", shareMode, creditsPrice,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create pack listing: %v", seed, err)
			return false
		}
		listingID, _ := listRes.LastInsertId()

		// Insert multiple duplicate records for the same listing_id.
		// Randomly choose how many duplicates (2-5) and which tables to use.
		numDuplicates := r.Intn(4) + 2 // 2 to 5 duplicates
		for i := 0; i < numDuplicates; i++ {
			// Randomly insert into credits_transactions or user_downloads
			if r.Intn(2) == 0 {
				txType := []string{"download", "purchase"}[r.Intn(2)]
				_, err = db.Exec(
					"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, created_at) VALUES (?, ?, ?, ?, ?, datetime('now', ?||' seconds'))",
					userID, txType, -float64(creditsPrice), listingID,
					fmt.Sprintf("购买 %s (dup %d)", packName, i),
					fmt.Sprintf("%d", i),
				)
				if err != nil {
					t.Logf("seed=%d: failed to insert credits_transaction dup %d: %v", seed, i, err)
					return false
				}
			} else {
				_, err = db.Exec(
					"INSERT INTO user_downloads (user_id, listing_id, downloaded_at) VALUES (?, ?, datetime('now', ?||' seconds'))",
					userID, listingID, fmt.Sprintf("%d", i),
				)
				if err != nil {
					t.Logf("seed=%d: failed to insert user_download dup %d: %v", seed, i, err)
					return false
				}
			}
		}

		// Call handleUserDashboard
		req := httptest.NewRequest(http.MethodGet, "/user/dashboard", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleUserDashboard(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d; body: %s", seed, rr.Code, rr.Body.String()[:min(300, rr.Body.Len())])
			return false
		}

		body := rr.Body.String()

		// The pack name should appear in the dashboard
		if !strings.Contains(body, packName) {
			t.Logf("seed=%d: pack %q not found in dashboard HTML at all", seed, packName)
			return false
		}

		// Count occurrences of the pack card in the HTML.
		// The pack name appears multiple times per card (title, data-pack-name attributes),
		// so we count the unique card marker: class="pack-name"> followed by the pack name.
		cardMarker := fmt.Sprintf(`<div class="pack-name">%s</div>`, packName)
		occurrences := strings.Count(body, cardMarker)
		if occurrences != 1 {
			t.Logf("seed=%d: pack %q card appeared %d times (expected 1) with %d duplicate records", seed, packName, occurrences, numDuplicates)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (个人中心按 listing_id 去重) failed: %v", err)
	}
}
