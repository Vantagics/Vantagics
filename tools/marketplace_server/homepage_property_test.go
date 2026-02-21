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

// Feature: marketplace-homepage, Property 12: storefront_id 唯一约束
// **Validates: Requirements 4.4, 4.6**
//
// For any storefront_id in featured_storefronts, inserting a duplicate
// storefront_id must fail due to the UNIQUE constraint on storefront_id.
// This ensures the same storefront cannot be featured more than once.
func TestProperty12_StorefrontIDUniqueConstraint(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront (FK requirement)
		userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
		slug := fmt.Sprintf("hp-store-%d-%d", userID, rng.Int63n(1000000))

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
			userID, slug, "Test Store", "A test store",
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// First insert into featured_storefronts should succeed
		_, err = db.Exec(
			"INSERT INTO featured_storefronts (storefront_id, sort_order) VALUES (?, ?)",
			storefrontID, 0,
		)
		if err != nil {
			t.Logf("FAIL: first insert into featured_storefronts failed: %v", err)
			return false
		}

		// Second insert with same storefront_id should fail due to UNIQUE constraint
		_, err = db.Exec(
			"INSERT INTO featured_storefronts (storefront_id, sort_order) VALUES (?, ?)",
			storefrontID, 1,
		)
		if err == nil {
			t.Logf("FAIL: second insert with same storefront_id=%d succeeded, expected UNIQUE violation", storefrontID)
			return false
		}

		// Verify exactly one record exists for this storefront_id
		var count int
		db.QueryRow("SELECT COUNT(*) FROM featured_storefronts WHERE storefront_id = ?", storefrontID).Scan(&count)
		if count != 1 {
			t.Logf("FAIL: expected exactly 1 featured record for storefront_id=%d, got %d", storefrontID, count)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 12 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 8: 热销店铺按销售额降序排列
// **Validates: Requirements 5.2, 5.3**
//
// For any set of stores with sales records, queryTopSalesStorefronts must return
// stores sorted by total sales (sum of ABS(amount) for purchase-type transactions)
// in descending order, and the result must not exceed 16 entries.
func TestProperty8_TopSalesStoresSortedCorrectly(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a buyer user for transactions
		buyerID := createTestUserWithBalance(t, 100000)

		numStores := rng.Intn(4) + 2 // 2-5 stores

		// Track expected sales per storefront ID for independent verification
		type storeRecord struct {
			storefrontID int64
			expectedSales float64
		}
		var storeRecords []storeRecord

		for i := 0; i < numStores; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
			slug := fmt.Sprintf("sales-store-%d-%d-%d", i, userID, rng.Int63n(1000000))
			storeName := fmt.Sprintf("SalesStore_%d_%d", i, rng.Int63n(100000))

			result, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
				userID, slug, storeName, fmt.Sprintf("Description for %s", storeName),
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}
			storefrontID, _ := result.LastInsertId()

			var totalSales float64

			// Create 1-3 published pack listings per store
			numPacks := rng.Intn(3) + 1
			for j := 0; j < numPacks; j++ {
				packID := createTestPackListing(t, userID, 1, "per_use", rng.Intn(50)+1, []byte("data"))

				// Insert 1-5 purchase transactions per listing
				numTxns := rng.Intn(5) + 1
				txnTypes := []string{"purchase", "purchase_uses", "renew_subscription"}
				for k := 0; k < numTxns; k++ {
					txnType := txnTypes[rng.Intn(len(txnTypes))]
					amount := -(float64(rng.Intn(100) + 1)) // negative amount = purchase
					_, err := db.Exec(
						"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id) VALUES (?, ?, ?, ?)",
						buyerID, txnType, amount, packID,
					)
					if err != nil {
						t.Logf("FAIL: failed to insert transaction: %v", err)
						return false
					}
					totalSales += -amount // ABS of negative amount
				}
			}

			storeRecords = append(storeRecords, storeRecord{
				storefrontID: storefrontID,
				expectedSales: totalSales,
			})
		}

		// Call the function under test
		stores, err := queryTopSalesStorefronts(16)
		if err != nil {
			t.Logf("FAIL: queryTopSalesStorefronts failed: %v", err)
			return false
		}

		// Verify: result count <= 16
		if len(stores) > 16 {
			t.Logf("FAIL: expected at most 16 stores, got %d", len(stores))
			return false
		}

		// Build a map of expected sales per storefront for independent verification
		expectedSalesMap := make(map[int64]float64)
		for _, sr := range storeRecords {
			expectedSalesMap[sr.storefrontID] = sr.expectedSales
		}

		// Independently calculate sales from DB to verify ordering
		// Query each returned store's actual total sales from credits_transactions
		var prevSales float64 = -1
		for i, store := range stores {
			var actualSales float64
			err := db.QueryRow(`
				SELECT COALESCE(SUM(ABS(ct.amount)), 0)
				FROM author_storefronts s
				JOIN pack_listings pl ON pl.user_id = s.user_id AND pl.status = 'published'
				JOIN credits_transactions ct ON ct.listing_id = pl.id
					AND ct.transaction_type IN ('purchase', 'purchase_uses', 'renew_subscription')
				WHERE s.id = ?`, store.StorefrontID).Scan(&actualSales)
			if err != nil {
				t.Logf("FAIL: failed to query actual sales for store %d: %v", store.StorefrontID, err)
				return false
			}

			// Verify descending order
			if i > 0 && actualSales > prevSales {
				t.Logf("FAIL: stores not sorted by sales descending: store[%d] sales=%.2f > store[%d] sales=%.2f",
					i, actualSales, i-1, prevSales)
				return false
			}
			prevSales = actualSales
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 9: 热门下载店铺按下载量降序排列
// **Validates: Requirements 6.2, 6.3**
//
// For any set of stores with download records, queryTopDownloadsStorefronts must return
// stores sorted by total downloads (sum of download_count for published pack_listings)
// in descending order, and the result must not exceed 16 entries.
func TestProperty9_TopDownloadsStoresSortedCorrectly(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		numStores := rng.Intn(4) + 2 // 2-5 stores

		type storeRecord struct {
			storefrontID   int64
			expectedDownloads int64
		}
		var storeRecords []storeRecord

		for i := 0; i < numStores; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
			slug := fmt.Sprintf("dl-store-%d-%d-%d", i, userID, rng.Int63n(1000000))
			storeName := fmt.Sprintf("DLStore_%d_%d", i, rng.Int63n(100000))

			result, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
				userID, slug, storeName, fmt.Sprintf("Description for %s", storeName),
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}
			storefrontID, _ := result.LastInsertId()

			var totalDownloads int64

			// Create 1-3 published pack listings per store
			numPacks := rng.Intn(3) + 1
			for j := 0; j < numPacks; j++ {
				packID := createTestPackListing(t, userID, 1, "per_use", rng.Intn(50)+1, []byte("data"))

				// Set a random download_count for this listing
				dlCount := int64(rng.Intn(500) + 1)
				_, err := db.Exec("UPDATE pack_listings SET download_count = ? WHERE id = ?", dlCount, packID)
				if err != nil {
					t.Logf("FAIL: failed to update download_count: %v", err)
					return false
				}
				totalDownloads += dlCount
			}

			storeRecords = append(storeRecords, storeRecord{
				storefrontID:      storefrontID,
				expectedDownloads: totalDownloads,
			})
		}

		// Call the function under test
		stores, err := queryTopDownloadsStorefronts(16)
		if err != nil {
			t.Logf("FAIL: queryTopDownloadsStorefronts failed: %v", err)
			return false
		}

		// Verify: result count <= 16
		if len(stores) > 16 {
			t.Logf("FAIL: expected at most 16 stores, got %d", len(stores))
			return false
		}

		// Independently query each returned store's actual total downloads and verify descending order
		var prevDownloads float64 = -1
		for i, store := range stores {
			var actualDownloads float64
			err := db.QueryRow(`
				SELECT COALESCE(SUM(pl.download_count), 0)
				FROM author_storefronts s
				JOIN pack_listings pl ON pl.user_id = s.user_id AND pl.status = 'published'
				WHERE s.id = ?`, store.StorefrontID).Scan(&actualDownloads)
			if err != nil {
				t.Logf("FAIL: failed to query actual downloads for store %d: %v", store.StorefrontID, err)
				return false
			}

			// Verify descending order
			if i > 0 && actualDownloads > prevDownloads {
				t.Logf("FAIL: stores not sorted by downloads descending: store[%d] downloads=%.0f > store[%d] downloads=%.0f",
					i, actualDownloads, i-1, prevDownloads)
				return false
			}
			prevDownloads = actualDownloads
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 10: 热销产品按销售额降序排列且仅含已发布产品
// **Validates: Requirements 7.2, 7.3**
//
// For any set of published and unpublished products with sales records,
// queryTopSalesProducts must return products sorted by total sales
// (sum of ABS(amount) for purchase-type transactions) in descending order,
// the result must not exceed 128 entries, and all returned products must
// have status='published'. No unpublished product should appear in the result.
func TestProperty10_TopSalesProductsSortedAndPublished(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a buyer user for transactions
		buyerID := createTestUserWithBalance(t, 100000)

		// Create a seller user with a storefront
		sellerID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
		slug := fmt.Sprintf("p10-store-%d-%d", sellerID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
			sellerID, slug, "P10 Store", "Property 10 test store",
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		// Create 3-6 published pack listings
		numPublished := rng.Intn(4) + 3
		var publishedIDs []int64
		for i := 0; i < numPublished; i++ {
			packID := createTestPackListing(t, sellerID, 1, "per_use", rng.Intn(50)+1, []byte("data"))
			publishedIDs = append(publishedIDs, packID)
		}

		// Create 1-3 unpublished pack listings (create as published, then update to draft)
		numUnpublished := rng.Intn(3) + 1
		var unpublishedIDs []int64
		for i := 0; i < numUnpublished; i++ {
			packID := createTestPackListing(t, sellerID, 1, "per_use", rng.Intn(50)+1, []byte("data"))
			_, err := db.Exec("UPDATE pack_listings SET status = 'draft' WHERE id = ?", packID)
			if err != nil {
				t.Logf("FAIL: failed to update listing to draft: %v", err)
				return false
			}
			unpublishedIDs = append(unpublishedIDs, packID)
		}

		// Insert random purchase transactions for ALL listings (both published and unpublished)
		allIDs := append(publishedIDs, unpublishedIDs...)
		txnTypes := []string{"purchase", "purchase_uses", "renew_subscription"}
		for _, packID := range allIDs {
			numTxns := rng.Intn(5) + 1
			for j := 0; j < numTxns; j++ {
				txnType := txnTypes[rng.Intn(len(txnTypes))]
				amount := -(float64(rng.Intn(100) + 1))
				_, err := db.Exec(
					"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id) VALUES (?, ?, ?, ?)",
					buyerID, txnType, amount, packID,
				)
				if err != nil {
					t.Logf("FAIL: failed to insert transaction: %v", err)
					return false
				}
			}
		}

		// Call the function under test
		products, err := queryTopSalesProducts(128)
		if err != nil {
			t.Logf("FAIL: queryTopSalesProducts failed: %v", err)
			return false
		}

		// Verify: result count <= 128
		if len(products) > 128 {
			t.Logf("FAIL: expected at most 128 products, got %d", len(products))
			return false
		}

		// Build a set of unpublished IDs for quick lookup
		unpublishedSet := make(map[int64]bool)
		for _, id := range unpublishedIDs {
			unpublishedSet[id] = true
		}

		// Verify: no unpublished product appears in the result
		for _, p := range products {
			if unpublishedSet[p.ListingID] {
				t.Logf("FAIL: unpublished product %d appeared in results", p.ListingID)
				return false
			}
		}

		// Verify: all returned products have status='published' in the DB
		for _, p := range products {
			var status string
			err := db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", p.ListingID).Scan(&status)
			if err != nil {
				t.Logf("FAIL: failed to query status for listing %d: %v", p.ListingID, err)
				return false
			}
			if status != "published" {
				t.Logf("FAIL: product %d has status=%q, expected 'published'", p.ListingID, status)
				return false
			}
		}

		// Verify: result is sorted by total sales descending
		var prevSales float64 = -1
		for i, p := range products {
			var actualSales float64
			err := db.QueryRow(`
				SELECT COALESCE(SUM(ABS(ct.amount)), 0)
				FROM credits_transactions ct
				WHERE ct.listing_id = ?
					AND ct.transaction_type IN ('purchase', 'purchase_uses', 'renew_subscription')`,
				p.ListingID).Scan(&actualSales)
			if err != nil {
				t.Logf("FAIL: failed to query actual sales for product %d: %v", p.ListingID, err)
				return false
			}

			if i > 0 && actualSales > prevSales {
				t.Logf("FAIL: products not sorted by sales descending: product[%d] sales=%.2f > product[%d] sales=%.2f",
					i, actualSales, i-1, prevSales)
				return false
			}
			prevSales = actualSales
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 10 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 1: 根路径返回首页 HTML
// **Validates: Requirements 1.1, 1.2**
//
// For any GET request to the root path "/", the server must return HTTP 200
// with text/html content, not a 302 redirect. The homepage is publicly accessible
// to all visitors regardless of authentication state.
func TestProperty1_RootPathServesHomepage(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		// Create an HTTP GET request to "/"
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		// Call handleHomepage directly
		handleHomepage(rr, req)

		// Verify: response status code is 200 (not 302 redirect)
		if rr.Code != http.StatusOK {
			t.Logf("FAIL: expected status 200, got %d", rr.Code)
			return false
		}

		// Verify: Content-Type header contains "text/html"
		contentType := rr.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Logf("FAIL: expected Content-Type containing 'text/html', got %q", contentType)
			return false
		}

		// Verify: response is NOT a redirect (no Location header)
		location := rr.Header().Get("Location")
		if location != "" {
			t.Logf("FAIL: unexpected redirect Location header: %q", location)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 2: 导航栏反映认证状态
// **Validates: Requirements 1.3, 2.6, 2.7**
//
// For any request to GET /, when the user is not logged in, the homepage HTML
// must contain login ("/user/login") and register ("/user/register") links.
// When the user is logged in (valid JWT in Authorization header), the homepage
// HTML must contain a user center link ("/user/") and must NOT contain the
// login form link.
func TestProperty2_NavigationReflectsAuthState(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// --- Unauthenticated test ---
		reqUnauth := httptest.NewRequest(http.MethodGet, "/", nil)
		rrUnauth := httptest.NewRecorder()
		handleHomepage(rrUnauth, reqUnauth)

		if rrUnauth.Code != http.StatusOK {
			t.Logf("FAIL: unauthenticated request returned status %d, expected 200", rrUnauth.Code)
			return false
		}

		unauthBody := rrUnauth.Body.String()

		// Unauthenticated: must contain login and register links
		if !strings.Contains(unauthBody, "/user/login") {
			t.Logf("FAIL: unauthenticated homepage does not contain '/user/login'")
			return false
		}
		if !strings.Contains(unauthBody, "/user/register") {
			t.Logf("FAIL: unauthenticated homepage does not contain '/user/register'")
			return false
		}

		// --- Authenticated test ---
		userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))

		// Generate a random display name
		nameLen := rng.Intn(10) + 2
		chars := []rune("abcdefghijklmnopqrstuvwxyz")
		nameRunes := make([]rune, nameLen)
		for i := range nameRunes {
			nameRunes[i] = chars[rng.Intn(len(chars))]
		}
		displayName := string(nameRunes)

		// Update user display_name in DB
		_, err := db.Exec("UPDATE users SET display_name = ? WHERE id = ?", displayName, userID)
		if err != nil {
			t.Logf("FAIL: failed to update display_name: %v", err)
			return false
		}

		// Generate a valid JWT token for the authenticated request
		token, err := generateJWT(userID, displayName)
		if err != nil {
			t.Logf("FAIL: failed to generate JWT: %v", err)
			return false
		}

		reqAuth := httptest.NewRequest(http.MethodGet, "/", nil)
		reqAuth.Header.Set("Authorization", "Bearer "+token)
		rrAuth := httptest.NewRecorder()
		handleHomepage(rrAuth, reqAuth)

		if rrAuth.Code != http.StatusOK {
			t.Logf("FAIL: authenticated request returned status %d, expected 200", rrAuth.Code)
			return false
		}

		authBody := rrAuth.Body.String()

		// Authenticated: must contain user center link
		if !strings.Contains(authBody, "/user/") {
			t.Logf("FAIL: authenticated homepage does not contain '/user/' link")
			return false
		}

		// Authenticated: must NOT contain login form link
		if strings.Contains(authBody, "/user/login") {
			t.Logf("FAIL: authenticated homepage still contains '/user/login' link")
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 3: 下载链接来自管理员配置
// **Validates: Requirements 2.3, 2.4**
//
// For any non-empty download URL string saved to the settings table,
// the rendered homepage HTML must contain that URL. When a download URL
// is empty, the corresponding download button must not appear in the HTML.
func TestProperty3_DownloadURLsFromSettings(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate random non-empty URLs for Windows and macOS
		genURL := func() string {
			pathLen := rng.Intn(20) + 5
			chars := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
			path := make([]byte, pathLen)
			for i := range path {
				path[i] = chars[rng.Intn(len(chars))]
			}
			return fmt.Sprintf("https://example.com/download/%s", string(path))
		}

		winURL := genURL()
		macURL := genURL()

		// Insert both download URLs into settings
		_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "download_url_windows", winURL)
		if err != nil {
			t.Logf("FAIL: failed to insert download_url_windows: %v", err)
			return false
		}
		_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "download_url_macos", macURL)
		if err != nil {
			t.Logf("FAIL: failed to insert download_url_macos: %v", err)
			return false
		}

		// Render homepage with both URLs set
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handleHomepage(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: expected status 200, got %d", rr.Code)
			return false
		}

		body := rr.Body.String()

		// Verify: HTML contains both download URLs
		if !strings.Contains(body, winURL) {
			t.Logf("FAIL: homepage HTML does not contain Windows download URL %q", winURL)
			return false
		}
		if !strings.Contains(body, macURL) {
			t.Logf("FAIL: homepage HTML does not contain macOS download URL %q", macURL)
			return false
		}

		// Now set Windows URL to empty and re-render
		_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "download_url_windows", "")
		if err != nil {
			t.Logf("FAIL: failed to clear download_url_windows: %v", err)
			return false
		}

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		rr2 := httptest.NewRecorder()
		handleHomepage(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Logf("FAIL: expected status 200 after clearing Windows URL, got %d", rr2.Code)
			return false
		}

		body2 := rr2.Body.String()

		// Verify: Windows URL no longer appears (button hidden)
		if strings.Contains(body2, winURL) {
			t.Logf("FAIL: homepage still contains Windows download URL %q after clearing it", winURL)
			return false
		}
		// Verify: macOS URL still present
		if !strings.Contains(body2, macURL) {
			t.Logf("FAIL: homepage does not contain macOS download URL %q after clearing Windows URL", macURL)
			return false
		}

		// Now set macOS URL to empty and re-render
		_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "download_url_macos", "")
		if err != nil {
			t.Logf("FAIL: failed to clear download_url_macos: %v", err)
			return false
		}

		req3 := httptest.NewRequest(http.MethodGet, "/", nil)
		rr3 := httptest.NewRecorder()
		handleHomepage(rr3, req3)

		if rr3.Code != http.StatusOK {
			t.Logf("FAIL: expected status 200 after clearing both URLs, got %d", rr3.Code)
			return false
		}

		body3 := rr3.Body.String()

		// Verify: neither URL appears when both are empty
		if strings.Contains(body3, macURL) {
			t.Logf("FAIL: homepage still contains macOS download URL %q after clearing it", macURL)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 4: 明星店铺数量上限
// **Validates: Requirements 3.3, 4.3**
//
// For any sequence of add-featured-storefront operations, the featured_storefronts
// table must never contain more than 16 records. When 16 storefronts are already
// featured, adding a 17th must be rejected with an error response.
func TestProperty4_FeaturedStoreCountLimit(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create 17 users and storefronts
		var storefrontIDs []int64
		for i := 0; i < 17; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
			slug := fmt.Sprintf("p4-store-%d-%d-%d", i, userID, rng.Int63n(1000000))
			storeName := fmt.Sprintf("P4Store_%d_%d", i, rng.Int63n(100000))

			result, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
				userID, slug, storeName, fmt.Sprintf("Description for %s", storeName),
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront %d: %v", i, err)
				return false
			}
			sfID, _ := result.LastInsertId()
			storefrontIDs = append(storefrontIDs, sfID)
		}

		// Add the first 16 storefronts via the API handler - all should succeed
		for i := 0; i < 16; i++ {
			body := strings.NewReader(fmt.Sprintf("storefront_id=%d", storefrontIDs[i]))
			req := httptest.NewRequest(http.MethodPost, "/api/admin/featured-storefronts", body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()

			handleAdminFeaturedStorefronts(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("FAIL: adding storefront %d (id=%d) returned status %d, expected 200. Body: %s",
					i, storefrontIDs[i], rr.Code, rr.Body.String())
				return false
			}
		}

		// Verify exactly 16 records in featured_storefronts
		var countBefore int
		if err := db.QueryRow("SELECT COUNT(*) FROM featured_storefronts").Scan(&countBefore); err != nil {
			t.Logf("FAIL: failed to count featured_storefronts: %v", err)
			return false
		}
		if countBefore != 16 {
			t.Logf("FAIL: expected 16 featured storefronts after adding 16, got %d", countBefore)
			return false
		}

		// Try to add the 17th storefront - should be rejected
		body17 := strings.NewReader(fmt.Sprintf("storefront_id=%d", storefrontIDs[16]))
		req17 := httptest.NewRequest(http.MethodPost, "/api/admin/featured-storefronts", body17)
		req17.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr17 := httptest.NewRecorder()

		handleAdminFeaturedStorefronts(rr17, req17)

		if rr17.Code != http.StatusBadRequest {
			t.Logf("FAIL: adding 17th storefront returned status %d, expected 400. Body: %s",
				rr17.Code, rr17.Body.String())
			return false
		}

		// Verify the count is still <= 16 after the rejected attempt
		var countAfter int
		if err := db.QueryRow("SELECT COUNT(*) FROM featured_storefronts").Scan(&countAfter); err != nil {
			t.Logf("FAIL: failed to count featured_storefronts after rejection: %v", err)
			return false
		}
		if countAfter > 16 {
			t.Logf("FAIL: featured_storefronts count exceeded 16 after rejected add: got %d", countAfter)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 6: 明星店铺排序不变性
// **Validates: Requirements 4.2**
//
// For any set of featured storefronts, after executing any reorder permutation,
// the sort_order values in the database must be strictly increasing, and the
// order of IDs must match the requested permutation.
func TestProperty6_FeaturedStoreReorderInvariance(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create 3-8 storefronts and add them as featured
		numStores := rng.Intn(6) + 3 // 3-8
		var featuredIDs []int64

		for i := 0; i < numStores; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
			slug := fmt.Sprintf("p6-store-%d-%d-%d", i, userID, rng.Int63n(1000000))
			storeName := fmt.Sprintf("P6Store_%d_%d", i, rng.Int63n(100000))

			result, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
				userID, slug, storeName, fmt.Sprintf("Description for %s", storeName),
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront %d: %v", i, err)
				return false
			}
			storefrontID, _ := result.LastInsertId()

			// Insert directly into featured_storefronts
			res, err := db.Exec(
				"INSERT INTO featured_storefronts (storefront_id, sort_order) VALUES (?, ?)",
				storefrontID, i,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert featured storefront %d: %v", i, err)
				return false
			}
			featuredID, _ := res.LastInsertId()
			featuredIDs = append(featuredIDs, featuredID)
		}

		// Generate a random permutation of the featured IDs
		permuted := make([]int64, len(featuredIDs))
		copy(permuted, featuredIDs)
		rng.Shuffle(len(permuted), func(i, j int) {
			permuted[i], permuted[j] = permuted[j], permuted[i]
		})

		// Build comma-separated IDs string for the reorder API
		idStrs := make([]string, len(permuted))
		for i, id := range permuted {
			idStrs[i] = fmt.Sprintf("%d", id)
		}
		idsParam := strings.Join(idStrs, ",")

		// Call the reorder API
		body := strings.NewReader("ids=" + idsParam)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/featured-storefronts/reorder", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		handleAdminFeaturedStorefronts(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: reorder API returned status %d, expected 200. Body: %s", rr.Code, rr.Body.String())
			return false
		}

		// Query featured_storefronts ordered by sort_order ASC
		rows, err := db.Query("SELECT id, sort_order FROM featured_storefronts ORDER BY sort_order ASC")
		if err != nil {
			t.Logf("FAIL: failed to query featured_storefronts after reorder: %v", err)
			return false
		}
		defer rows.Close()

		var resultIDs []int64
		var sortOrders []int
		for rows.Next() {
			var id int64
			var sortOrder int
			if err := rows.Scan(&id, &sortOrder); err != nil {
				t.Logf("FAIL: failed to scan row: %v", err)
				return false
			}
			resultIDs = append(resultIDs, id)
			sortOrders = append(sortOrders, sortOrder)
		}
		if err := rows.Err(); err != nil {
			t.Logf("FAIL: rows iteration error: %v", err)
			return false
		}

		// Verify: sort_order values are strictly increasing
		for i := 1; i < len(sortOrders); i++ {
			if sortOrders[i] <= sortOrders[i-1] {
				t.Logf("FAIL: sort_order not strictly increasing: sortOrders[%d]=%d <= sortOrders[%d]=%d",
					i, sortOrders[i], i-1, sortOrders[i-1])
				return false
			}
		}

		// Verify: the order of IDs matches the requested permutation
		if len(resultIDs) != len(permuted) {
			t.Logf("FAIL: expected %d results, got %d", len(permuted), len(resultIDs))
			return false
		}
		for i := range permuted {
			if resultIDs[i] != permuted[i] {
				t.Logf("FAIL: ID mismatch at position %d: expected %d, got %d", i, permuted[i], resultIDs[i])
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 7: 明星店铺添加/移除 Round-Trip
// **Validates: Requirements 4.4, 4.5**
//
// For any storefront, adding it as a featured store and then querying should
// include that storefront; removing it and then querying should no longer
// include that storefront.
func TestProperty7_FeaturedStoreAddRemoveRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a random user and storefront
		userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
		slug := fmt.Sprintf("p7-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P7Store_%d_%d", userID, rng.Int63n(100000))

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
			userID, slug, storeName, fmt.Sprintf("Description for %s", storeName),
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		sfID, _ := result.LastInsertId()

		// Step 1: Add the storefront as featured via the API
		addBody := strings.NewReader(fmt.Sprintf("storefront_id=%d", sfID))
		addReq := httptest.NewRequest(http.MethodPost, "/api/admin/featured-storefronts", addBody)
		addReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		addRR := httptest.NewRecorder()

		handleAdminFeaturedStorefronts(addRR, addReq)

		if addRR.Code != http.StatusOK {
			t.Logf("FAIL: adding storefront (id=%d) returned status %d, expected 200. Body: %s",
				sfID, addRR.Code, addRR.Body.String())
			return false
		}

		// Step 2: Query featured storefronts and verify the storefront is present
		stores, err := queryFeaturedStorefronts()
		if err != nil {
			t.Logf("FAIL: queryFeaturedStorefronts after add failed: %v", err)
			return false
		}

		found := false
		for _, s := range stores {
			if s.StorefrontID == sfID {
				found = true
				break
			}
		}
		if !found {
			t.Logf("FAIL: storefront (id=%d) not found in featured list after adding", sfID)
			return false
		}

		// Step 3: Remove the storefront via the API
		removeBody := strings.NewReader(fmt.Sprintf("storefront_id=%d", sfID))
		removeReq := httptest.NewRequest(http.MethodPost, "/api/admin/featured-storefronts/remove", removeBody)
		removeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		removeRR := httptest.NewRecorder()

		handleAdminFeaturedStorefronts(removeRR, removeReq)

		if removeRR.Code != http.StatusOK {
			t.Logf("FAIL: removing storefront (id=%d) returned status %d, expected 200. Body: %s",
				sfID, removeRR.Code, removeRR.Body.String())
			return false
		}

		// Step 4: Query featured storefronts and verify the storefront is NOT present
		storesAfter, err := queryFeaturedStorefronts()
		if err != nil {
			t.Logf("FAIL: queryFeaturedStorefronts after remove failed: %v", err)
			return false
		}

		for _, s := range storesAfter {
			if s.StorefrontID == sfID {
				t.Logf("FAIL: storefront (id=%d) still found in featured list after removal", sfID)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 5: 店铺卡片包含必要信息
// **Validates: Requirements 3.4, 3.6, 5.5, 6.5**
//
// For any storefront with a name and description, when that storefront appears
// on the homepage (as a featured store), the rendered HTML must contain the
// store name, description (or its truncated version), and a link to
// "/store/{store_slug}".
func TestProperty5_StoreCardContainsRequiredInfo(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate a random store name (3-20 alphanumeric chars)
		nameLen := rng.Intn(18) + 3
		nameChars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		nameRunes := make([]rune, nameLen)
		for i := range nameRunes {
			nameRunes[i] = nameChars[rng.Intn(len(nameChars))]
		}
		storeName := string(nameRunes)

		// Generate a random slug (lowercase + digits + hyphens)
		slugLen := rng.Intn(15) + 5
		slugChars := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
		slugRunes := make([]rune, slugLen)
		for i := range slugRunes {
			slugRunes[i] = slugChars[rng.Intn(len(slugChars))]
		}
		storeSlug := fmt.Sprintf("p5-%s-%d", string(slugRunes), rng.Int63n(1000000))

		// Generate a random description (5-120 chars to test both truncated and non-truncated)
		descLen := rng.Intn(116) + 5
		descChars := []rune("abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		descRunes := make([]rune, descLen)
		for i := range descRunes {
			descRunes[i] = descChars[rng.Intn(len(descChars))]
		}
		description := string(descRunes)

		// Create a user and storefront
		userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
			userID, storeSlug, storeName, description,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		sfID, _ := result.LastInsertId()

		// Add the storefront as featured so it appears on the homepage
		_, err = db.Exec(
			"INSERT INTO featured_storefronts (storefront_id, sort_order) VALUES (?, 0)",
			sfID,
		)
		if err != nil {
			t.Logf("FAIL: failed to add storefront as featured: %v", err)
			return false
		}

		// Render the homepage
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handleHomepage(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: homepage returned status %d, expected 200", rr.Code)
			return false
		}

		body := rr.Body.String()

		// Verify: HTML contains the store name
		if !strings.Contains(body, storeName) {
			t.Logf("FAIL: homepage HTML does not contain store name %q", storeName)
			return false
		}

		// Verify: HTML contains the description (or truncated version)
		// The template uses truncateDesc with maxLen=80, appending "..." if truncated
		descRunesCheck := []rune(description)
		if len(descRunesCheck) <= 80 {
			if !strings.Contains(body, description) {
				t.Logf("FAIL: homepage HTML does not contain full description %q", description)
				return false
			}
		} else {
			truncated := string(descRunesCheck[:80]) + "..."
			if !strings.Contains(body, truncated) {
				t.Logf("FAIL: homepage HTML does not contain truncated description %q", truncated)
				return false
			}
		}

		// Verify: HTML contains the /store/{slug} link
		expectedLink := "/store/" + storeSlug
		if !strings.Contains(body, expectedLink) {
			t.Logf("FAIL: homepage HTML does not contain store link %q", expectedLink)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 violated: %v", err)
	}
}

// Feature: marketplace-homepage, Property 11: 产品卡片包含必要信息
// **Validates: Requirements 7.4, 7.6**
func TestProperty11_ProductCardContainsRequiredInfo(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate a random pack name (3-20 alphanumeric chars)
		packNameLen := rng.Intn(18) + 3
		nameChars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		packNameRunes := make([]rune, packNameLen)
		for i := range packNameRunes {
			packNameRunes[i] = nameChars[rng.Intn(len(nameChars))]
		}
		packName := string(packNameRunes)

		// Generate a random author name (3-20 alphanumeric chars)
		authorNameLen := rng.Intn(18) + 3
		authorNameRunes := make([]rune, authorNameLen)
		for i := range authorNameRunes {
			authorNameRunes[i] = nameChars[rng.Intn(len(nameChars))]
		}
		authorName := string(authorNameRunes)

		// Create a seller user and storefront
		sellerID := createTestUserWithBalance(t, float64(rng.Intn(1000)))
		slug := fmt.Sprintf("p11-store-%d-%d", sellerID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
			sellerID, slug, "P11 Store", "Property 11 test store",
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		// Create a published pack listing with default names
		packID := createTestPackListing(t, sellerID, 1, "per_use", rng.Intn(50)+1, []byte("data"))

		// Update pack_name and author_name to random values
		_, err = db.Exec("UPDATE pack_listings SET pack_name = ?, author_name = ? WHERE id = ?",
			packName, authorName, packID)
		if err != nil {
			t.Logf("FAIL: failed to update pack listing names: %v", err)
			return false
		}

		// Create a buyer and insert purchase transactions so the product appears in top sales
		buyerID := createTestUserWithBalance(t, 100000)
		txnTypes := []string{"purchase", "purchase_uses", "renew_subscription"}
		numTxns := rng.Intn(5) + 1
		for j := 0; j < numTxns; j++ {
			txnType := txnTypes[rng.Intn(len(txnTypes))]
			amount := -(float64(rng.Intn(100) + 1))
			_, err := db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id) VALUES (?, ?, ?, ?)",
				buyerID, txnType, amount, packID,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert transaction: %v", err)
				return false
			}
		}

		// Render the homepage
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handleHomepage(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: homepage returned status %d, expected 200", rr.Code)
			return false
		}

		body := rr.Body.String()

		// Verify: HTML contains the product pack name
		if !strings.Contains(body, packName) {
			t.Logf("FAIL: homepage HTML does not contain pack name %q", packName)
			return false
		}

		// Verify: HTML contains the product author name
		if !strings.Contains(body, authorName) {
			t.Logf("FAIL: homepage HTML does not contain author name %q", authorName)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 11 violated: %v", err)
	}
}
