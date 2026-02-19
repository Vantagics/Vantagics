package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// testInsertPackListing inserts a pack listing with a share_token and returns (listingID, shareToken).
func testInsertPackListing(userID, categoryID int64, packName, shareMode string, creditsPrice int, status string) (int64, string, error) {
	token := generateShareToken()
	res, err := db.Exec(
		`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, share_token)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, categoryID, []byte("fake-data"), packName, "desc", "Src", "Author", shareMode, creditsPrice, status, token,
	)
	if err != nil {
		return 0, "", err
	}
	id, _ := res.LastInsertId()
	return id, token, nil
}

// testInsertPackListingFull inserts a pack listing with full metadata and a share_token.
func testInsertPackListingFull(userID, categoryID int64, packName, packDesc, sourceName, authorName, shareMode string, creditsPrice, downloadCount int) (int64, string, error) {
	token := generateShareToken()
	res, err := db.Exec(
		`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, download_count, share_token)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', ?, ?)`,
		userID, categoryID, []byte("fake-data"), packName, packDesc, sourceName, authorName, shareMode, creditsPrice, downloadCount, token,
	)
	if err != nil {
		return 0, "", err
	}
	id, _ := res.LastInsertId()
	return id, token, nil
}

// Feature: qap-share-url, Property 10: pack_name 查询 listing_id 一致性
// **Validates: Requirements 7.2**
//
// For any published pack listing, querying /api/packs/listing-id with the pack_name
// should return the same listing_id as the one stored in the pack_listings table.
func TestProperty10_PackNameQueryListingIDConsistency(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES ('ShareURLCat', 'test category')")
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = 'ShareURLCat'").Scan(&categoryID)

		// Create a test user (the pack author)
		username := fmt.Sprintf("author_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword("testpass123")
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-SHARE-%d", seed), username, email, username, hashed, 500.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Create random number of published pack listings (1-5)
		numListings := r.Intn(5) + 1
		type packInfo struct {
			listingID int64
			packName  string
		}
		packs := make([]packInfo, 0, numListings)

		for i := 0; i < numListings; i++ {
			packName := fmt.Sprintf("SharePack_%d_%d_%d", seed, i, r.Intn(10000))
			lid, _, err := testInsertPackListing(userID, categoryID, packName, "free", 0, "published")
			if err != nil {
				t.Logf("seed=%d: failed to create listing %d: %v", seed, i, err)
				return false
			}
			packs = append(packs, packInfo{listingID: lid, packName: packName})
		}

		// For each pack, query the listing-id API and verify consistency
		for _, p := range packs {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/packs/listing-id?pack_name=%s", p.packName), nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleGetListingID(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("seed=%d: expected 200 for pack %s, got %d", seed, p.packName, rr.Code)
				return false
			}

			var resp struct {
				ListingID int64 `json:"listing_id"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("seed=%d: failed to parse response for pack %s: %v", seed, p.packName, err)
				return false
			}

			if resp.ListingID != p.listingID {
				t.Logf("seed=%d: listing_id mismatch for pack %s: expected %d, got %d", seed, p.packName, p.listingID, resp.ListingID)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 10 (pack_name 查询 listing_id 一致性) failed: %v", err)
	}
}


// Feature: qap-share-url, Property 2: 分析包详情 API 返回完整信息
// **Validates: Requirements 2.2**
//
// For any published pack listing, calling /api/packs/{listing_id}/detail should return
// a JSON response containing all required fields: listing_id, pack_name, pack_description,
// source_name, author_name, share_mode, credits_price, download_count, and category_name.
func TestProperty2_PackDetailAPIReturnsCompleteInfo(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category with random name
		catName := fmt.Sprintf("DetailCat_%d", seed)
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, 'test')", catName)
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = ?", catName).Scan(&categoryID)

		// Create a test user
		username := fmt.Sprintf("detail_author_%d", seed)
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-DETAIL-%d", seed), username, fmt.Sprintf("%s@test.com", username), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Generate random pack data
		shareModes := []string{"free", "per_use", "subscription"}
		shareMode := shareModes[r.Intn(len(shareModes))]
		creditsPrice := 0
		if shareMode != "free" {
			creditsPrice = r.Intn(500) + 1
		}
		packName := fmt.Sprintf("DetailPack_%d_%d", seed, r.Intn(10000))
		packDesc := fmt.Sprintf("Description for %s", packName)
		sourceName := fmt.Sprintf("Source_%d", seed)
		authorName := fmt.Sprintf("Author_%d", seed)
		downloadCount := r.Intn(1000)

		// Insert the pack listing
		listingID, _, err := testInsertPackListingFull(userID, categoryID, packName, packDesc, sourceName, authorName, shareMode, creditsPrice, downloadCount)
		if err != nil {
			t.Logf("seed=%d: failed to create listing: %v", seed, err)
			return false
		}

		// Call the detail API
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/packs/%d/detail", listingID), nil)
		rr := httptest.NewRecorder()
		handleGetPackDetail(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d, body: %s", seed, rr.Code, rr.Body.String())
			return false
		}

		// Parse and verify all fields
		var detail struct {
			ListingID       int64  `json:"listing_id"`
			PackName        string `json:"pack_name"`
			PackDescription string `json:"pack_description"`
			SourceName      string `json:"source_name"`
			AuthorName      string `json:"author_name"`
			ShareMode       string `json:"share_mode"`
			CreditsPrice    int    `json:"credits_price"`
			DownloadCount   int    `json:"download_count"`
			CategoryName    string `json:"category_name"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &detail); err != nil {
			t.Logf("seed=%d: failed to parse response: %v", seed, err)
			return false
		}

		// Verify each field matches what was inserted
		if detail.ListingID != listingID {
			t.Logf("seed=%d: listing_id mismatch: expected %d, got %d", seed, listingID, detail.ListingID)
			return false
		}
		if detail.PackName != packName {
			t.Logf("seed=%d: pack_name mismatch: expected %q, got %q", seed, packName, detail.PackName)
			return false
		}
		if detail.PackDescription != packDesc {
			t.Logf("seed=%d: pack_description mismatch: expected %q, got %q", seed, packDesc, detail.PackDescription)
			return false
		}
		if detail.SourceName != sourceName {
			t.Logf("seed=%d: source_name mismatch: expected %q, got %q", seed, sourceName, detail.SourceName)
			return false
		}
		if detail.AuthorName != authorName {
			t.Logf("seed=%d: author_name mismatch: expected %q, got %q", seed, authorName, detail.AuthorName)
			return false
		}
		if detail.ShareMode != shareMode {
			t.Logf("seed=%d: share_mode mismatch: expected %q, got %q", seed, shareMode, detail.ShareMode)
			return false
		}
		if detail.CreditsPrice != creditsPrice {
			t.Logf("seed=%d: credits_price mismatch: expected %d, got %d", seed, creditsPrice, detail.CreditsPrice)
			return false
		}
		if detail.DownloadCount != downloadCount {
			t.Logf("seed=%d: download_count mismatch: expected %d, got %d", seed, downloadCount, detail.DownloadCount)
			return false
		}
		if detail.CategoryName != catName {
			t.Logf("seed=%d: category_name mismatch: expected %q, got %q", seed, catName, detail.CategoryName)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (分析包详情 API 返回完整信息) failed: %v", err)
	}
}

// Feature: qap-share-url, Property 3: 不存在或未发布的分析包返回 404
// **Validates: Requirements 2.3**
//
// For any listing_id that does not exist in the database, or any listing that exists
// but has a status other than 'published', calling /api/packs/{listing_id}/detail
// should return HTTP 404.
func TestProperty3_NonExistentOrUnpublishedPackReturns404(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES ('NotFoundCat', 'test')")
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = 'NotFoundCat'").Scan(&categoryID)

		// Create a test user
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-NF-%d", seed), "nfuser", fmt.Sprintf("nf_%d@test.com", seed), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Test 1: Non-existent listing_id (random large ID)
		nonExistentID := int64(r.Intn(900000)) + 100000
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/packs/%d/detail", nonExistentID), nil)
		rr := httptest.NewRecorder()
		handleGetPackDetail(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Logf("seed=%d: non-existent ID %d: expected 404, got %d", seed, nonExistentID, rr.Code)
			return false
		}

		// Test 2: Create a pack with non-published status and verify 404
		nonPublishedStatuses := []string{"pending", "rejected", "delisted"}
		status := nonPublishedStatuses[r.Intn(len(nonPublishedStatuses))]
		packName := fmt.Sprintf("UnpubPack_%d_%d", seed, r.Intn(10000))
		unpubID, _, err := testInsertPackListing(userID, categoryID, packName, "free", 0, status)
		if err != nil {
			t.Logf("seed=%d: failed to create unpublished listing: %v", seed, err)
			return false
		}

		req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/packs/%d/detail", unpubID), nil)
		rr2 := httptest.NewRecorder()
		handleGetPackDetail(rr2, req2)

		if rr2.Code != http.StatusNotFound {
			t.Logf("seed=%d: unpublished pack (status=%s, id=%d): expected 404, got %d", seed, status, unpubID, rr2.Code)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (不存在或未发布的分析包返回 404) failed: %v", err)
	}
}

// Feature: qap-share-url, Property 11: 无效 listing_id 输入验证
// **Validates: Requirements 9.3**
//
// For any non-positive integer or non-numeric string as listing_id,
// the server should return HTTP 400 error.
func TestProperty11_InvalidListingIDInputValidation(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Generate various invalid listing_id values
		invalidInputs := []string{
			"0",
			"-1",
			fmt.Sprintf("-%d", r.Intn(1000)+1),
			"abc",
			"12.5",
			"",
			"null",
			"true",
			fmt.Sprintf("-%d", r.Int63n(999999)+1),
		}

		// Pick a random invalid input
		input := invalidInputs[r.Intn(len(invalidInputs))]

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/packs/%s/detail", input), nil)
		rr := httptest.NewRecorder()
		handleGetPackDetail(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Logf("seed=%d: invalid input %q: expected 400, got %d", seed, input, rr.Code)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 11 (无效 listing_id 输入验证) failed: %v", err)
	}
}


// Feature: qap-share-url, Property 4: 详情页根据 share_mode 显示正确的按钮类型
// **Validates: Requirements 3.3, 3.4**
//
// For any pack detail data, when share_mode is "free" the rendered HTML should contain
// "免费领取" button text; when share_mode is "per_use" or "subscription" the rendered
// HTML should contain "购买" button text and price info.
func TestProperty4_DetailPageShowsCorrectButtonByShareMode(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		catName := fmt.Sprintf("BtnCat_%d", seed)
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, 'test')", catName)
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = ?", catName).Scan(&categoryID)

		// Create a test user
		username := fmt.Sprintf("btn_author_%d", seed)
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-BTN-%d", seed), username, fmt.Sprintf("%s@test.com", username), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Pick a random share_mode
		shareModes := []string{"free", "per_use", "subscription"}
		shareMode := shareModes[r.Intn(len(shareModes))]
		creditsPrice := 0
		if shareMode != "free" {
			creditsPrice = r.Intn(500) + 1
		}
		packName := fmt.Sprintf("BtnPack_%d_%d", seed, r.Intn(10000))

		// Insert a published pack listing
		_, shareToken, err := testInsertPackListing(userID, categoryID, packName, shareMode, creditsPrice, "published")
		if err != nil {
			t.Logf("seed=%d: failed to create listing: %v", seed, err)
			return false
		}

		// Call handlePackDetailPage (no session cookie → unauthenticated visitor)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pack/%s", shareToken), nil)
		rr := httptest.NewRecorder()
		handlePackDetailPage(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d", seed, rr.Code)
			return false
		}

		body := rr.Body.String()

		if shareMode == "free" {
			// For unauthenticated visitor with free pack, the rendered action button
			// should be "登录后免费领取" (login-to-claim link)
			if !strings.Contains(body, "登录后免费领取") {
				t.Logf("seed=%d: share_mode=free but HTML does not contain '登录后免费领取'", seed)
				return false
			}
			// Should NOT contain the paid action button "登录后购买"
			if strings.Contains(body, "登录后购买") {
				t.Logf("seed=%d: share_mode=free but HTML contains '登录后购买'", seed)
				return false
			}
		} else {
			// share_mode is per_use or subscription
			// For unauthenticated visitor, the rendered action button should be "登录后购买"
			if !strings.Contains(body, "登录后购买") {
				t.Logf("seed=%d: share_mode=%s but HTML does not contain '登录后购买'", seed, shareMode)
				return false
			}
			// Should contain price info (credits_price value)
			priceStr := fmt.Sprintf("%d Credits", creditsPrice)
			if !strings.Contains(body, priceStr) {
				t.Logf("seed=%d: share_mode=%s but HTML does not contain price '%s'", seed, shareMode, priceStr)
				return false
			}
			// Should NOT contain the free action button "登录后免费领取"
			if strings.Contains(body, "登录后免费领取") {
				t.Logf("seed=%d: share_mode=%s but HTML contains '登录后免费领取'", seed, shareMode)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 (详情页根据 share_mode 显示正确的按钮类型) failed: %v", err)
	}
}

// Feature: qap-share-url, Property 5: 登录重定向往返一致性
// **Validates: Requirements 4.1, 4.3**
//
// For any valid share_token, when an unauthenticated user visits /pack/{share_token},
// the page should contain a login link with redirect=/pack/{share_token}, preserving
// the original pack path for post-login redirection.
func TestProperty5_LoginRedirectRoundtripConsistency(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		catName := fmt.Sprintf("RedirCat_%d", seed)
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, 'test')", catName)
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = ?", catName).Scan(&categoryID)

		// Create a test user (pack author)
		username := fmt.Sprintf("redir_author_%d", seed)
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-REDIR-%d", seed), username, fmt.Sprintf("%s@test.com", username), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Pick a random share_mode
		shareModes := []string{"free", "per_use", "subscription"}
		shareMode := shareModes[r.Intn(len(shareModes))]
		creditsPrice := 0
		if shareMode != "free" {
			creditsPrice = r.Intn(500) + 1
		}
		packName := fmt.Sprintf("RedirPack_%d_%d", seed, r.Intn(10000))

		// Insert a published pack listing
		_, shareToken, err := testInsertPackListing(userID, categoryID, packName, shareMode, creditsPrice, "published")
		if err != nil {
			t.Logf("seed=%d: failed to create listing: %v", seed, err)
			return false
		}

		// Call handlePackDetailPage WITHOUT a session cookie (unauthenticated)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pack/%s", shareToken), nil)
		rr := httptest.NewRecorder()
		handlePackDetailPage(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d", seed, rr.Code)
			return false
		}

		body := rr.Body.String()

		// The page should contain a login redirect link that preserves the pack path (using share_token)
		expectedRedirect := fmt.Sprintf("/user/login?redirect=/pack/%s", shareToken)
		if !strings.Contains(body, expectedRedirect) {
			t.Logf("seed=%d: HTML does not contain expected redirect URL %q", seed, expectedRedirect)
			return false
		}

		// Verify the redirect path in the link matches the original pack path
		expectedPackPath := fmt.Sprintf("/pack/%s", shareToken)
		redirectParam := fmt.Sprintf("redirect=%s", expectedPackPath)
		if !strings.Contains(body, redirectParam) {
			t.Logf("seed=%d: redirect param %q not found in HTML", seed, redirectParam)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (登录重定向往返一致性) failed: %v", err)
	}
}


// Feature: qap-share-url, Property 6: 免费领取后创建购买和下载记录
// **Validates: Requirements 5.2, 5.3**
//
// For any logged-in user and free pack, after calling POST /pack/{id}/claim,
// user_purchased_packs should have a record (is_hidden=0) and user_downloads
// should have a record.
func TestProperty6_FreeClaimCreatesRecords(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		catName := fmt.Sprintf("ClaimCat_%d", seed)
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, 'test')", catName)
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = ?", catName).Scan(&categoryID)

		// Create a pack author
		authorName := fmt.Sprintf("claim_author_%d", seed)
		authorRes, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-CA-%d", seed), authorName, fmt.Sprintf("ca_%d@test.com", seed), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create author: %v", seed, err)
			return false
		}
		authorID, _ := authorRes.LastInsertId()

		// Create a random number of free pack listings (1-3)
		numPacks := r.Intn(3) + 1
		type listingInfo struct {
			id    int64
			token string
		}
		listings := make([]listingInfo, 0, numPacks)
		for i := 0; i < numPacks; i++ {
			packName := fmt.Sprintf("FreePack_%d_%d_%d", seed, i, r.Intn(10000))
			lid, token, err := testInsertPackListing(authorID, categoryID, packName, "free", 0, "published")
			if err != nil {
				t.Logf("seed=%d: failed to create listing %d: %v", seed, i, err)
				return false
			}
			listings = append(listings, listingInfo{id: lid, token: token})
		}

		// Create a claiming user
		claimerName := fmt.Sprintf("claimer_%d_%d", seed, r.Intn(10000))
		claimerRes, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-CL-%d-%d", seed, r.Intn(10000)), claimerName, fmt.Sprintf("cl_%d_%d@test.com", seed, r.Intn(10000)), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create claimer: %v", seed, err)
			return false
		}
		claimerID, _ := claimerRes.LastInsertId()

		// Claim each free pack
		for _, li := range listings {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/pack/%s/claim", li.token), nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", claimerID))
			rr := httptest.NewRecorder()
			handleClaimFreePack(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("seed=%d: claim listing %d: expected 200, got %d, body: %s", seed, li.id, rr.Code, rr.Body.String())
				return false
			}

			// Verify user_purchased_packs record exists with is_hidden=0
			var isHidden int
			err := db.QueryRow(
				"SELECT is_hidden FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?",
				claimerID, li.id,
			).Scan(&isHidden)
			if err != nil {
				t.Logf("seed=%d: no user_purchased_packs record for user=%d listing=%d: %v", seed, claimerID, li.id, err)
				return false
			}
			if isHidden != 0 {
				t.Logf("seed=%d: user_purchased_packs is_hidden=%d, expected 0 for user=%d listing=%d", seed, isHidden, claimerID, li.id)
				return false
			}

			// Verify user_downloads record exists
			var downloadCount int
			err = db.QueryRow(
				"SELECT COUNT(*) FROM user_downloads WHERE user_id = ? AND listing_id = ?",
				claimerID, li.id,
			).Scan(&downloadCount)
			if err != nil {
				t.Logf("seed=%d: failed to query user_downloads for user=%d listing=%d: %v", seed, claimerID, li.id, err)
				return false
			}
			if downloadCount < 1 {
				t.Logf("seed=%d: no user_downloads record for user=%d listing=%d", seed, claimerID, li.id)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 (免费领取后创建购买和下载记录) failed: %v", err)
	}
}

// Feature: qap-share-url, Property 7: 免费领取幂等性
// **Validates: Requirements 5.5**
//
// For any logged-in user and free pack, calling claim twice should still result
// in only one record in user_purchased_packs (due to UNIQUE constraint / upsert),
// and is_hidden should remain 0.
func TestProperty7_FreeClaimIdempotency(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		catName := fmt.Sprintf("IdempCat_%d", seed)
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, 'test')", catName)
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = ?", catName).Scan(&categoryID)

		// Create a pack author
		authorRes, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-IA-%d", seed), fmt.Sprintf("idemp_author_%d", seed), fmt.Sprintf("ia_%d@test.com", seed), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create author: %v", seed, err)
			return false
		}
		authorID, _ := authorRes.LastInsertId()

		// Create a free pack listing
		packName := fmt.Sprintf("IdempPack_%d_%d", seed, r.Intn(10000))
		listingID, shareToken, err := testInsertPackListing(authorID, categoryID, packName, "free", 0, "published")
		if err != nil {
			t.Logf("seed=%d: failed to create listing: %v", seed, err)
			return false
		}

		// Create a claiming user
		claimerRes, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-IC-%d-%d", seed, r.Intn(10000)), fmt.Sprintf("idemp_claimer_%d", seed), fmt.Sprintf("ic_%d@test.com", seed), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create claimer: %v", seed, err)
			return false
		}
		claimerID, _ := claimerRes.LastInsertId()

		// Claim the pack multiple times (2-5 times)
		numClaims := r.Intn(4) + 2
		for i := 0; i < numClaims; i++ {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/pack/%s/claim", shareToken), nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", claimerID))
			rr := httptest.NewRecorder()
			handleClaimFreePack(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("seed=%d: claim attempt %d: expected 200, got %d", seed, i+1, rr.Code)
				return false
			}
		}

		// Verify only ONE record in user_purchased_packs (UNIQUE constraint)
		var purchaseCount int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?",
			claimerID, listingID,
		).Scan(&purchaseCount)
		if err != nil {
			t.Logf("seed=%d: failed to count user_purchased_packs: %v", seed, err)
			return false
		}
		if purchaseCount != 1 {
			t.Logf("seed=%d: expected 1 user_purchased_packs record, got %d (after %d claims)", seed, purchaseCount, numClaims)
			return false
		}

		// Verify is_hidden remains 0
		var isHidden int
		err = db.QueryRow(
			"SELECT is_hidden FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?",
			claimerID, listingID,
		).Scan(&isHidden)
		if err != nil {
			t.Logf("seed=%d: failed to query is_hidden: %v", seed, err)
			return false
		}
		if isHidden != 0 {
			t.Logf("seed=%d: is_hidden=%d after %d claims, expected 0", seed, isHidden, numClaims)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 (免费领取幂等性) failed: %v", err)
	}
}

// Feature: qap-share-url, Property 8: 购买余额检查正确性
// **Validates: Requirements 6.4, 6.7**
//
// For any logged-in user, paid pack, and purchase parameters: when credits_balance >= total_cost,
// purchase should succeed; when credits_balance < total_cost, purchase should return insufficient_balance.
func TestProperty8_PurchaseBalanceCheckCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		catName := fmt.Sprintf("BalCat_%d", seed)
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, 'test')", catName)
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = ?", catName).Scan(&categoryID)

		// Create a pack author
		authorRes, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-BA-%d", seed), fmt.Sprintf("bal_author_%d", seed), fmt.Sprintf("ba_%d@test.com", seed), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create author: %v", seed, err)
			return false
		}
		authorID, _ := authorRes.LastInsertId()

		// Pick a random paid share_mode
		paidModes := []string{"per_use", "subscription"}
		shareMode := paidModes[r.Intn(len(paidModes))]
		creditsPrice := r.Intn(100) + 1 // 1-100

		// Create a paid pack listing
		packName := fmt.Sprintf("BalPack_%d_%d", seed, r.Intn(10000))
		listingID, shareToken, err := testInsertPackListing(authorID, categoryID, packName, shareMode, creditsPrice, "published")
		_ = listingID
		if err != nil {
			t.Logf("seed=%d: failed to create listing: %v", seed, err)
			return false
		}

		// Generate purchase parameters
		var quantity, months int
		var totalCost int
		var bodyStr string
		if shareMode == "per_use" {
			quantity = r.Intn(10) + 1 // 1-10
			totalCost = creditsPrice * quantity
			bodyStr = fmt.Sprintf(`{"quantity": %d}`, quantity)
		} else {
			months = r.Intn(12) + 1 // 1-12
			totalCost = creditsPrice * months
			bodyStr = fmt.Sprintf(`{"months": %d}`, months)
		}

		// Test case 1: sufficient balance (balance >= totalCost)
		sufficientBalance := float64(totalCost + r.Intn(500)) // totalCost to totalCost+499
		buyerRes1, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-BB1-%d-%d", seed, r.Intn(10000)), fmt.Sprintf("buyer1_%d", seed), fmt.Sprintf("bb1_%d_%d@test.com", seed, r.Intn(10000)), sufficientBalance,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create buyer1: %v", seed, err)
			return false
		}
		buyerID1, _ := buyerRes1.LastInsertId()

		req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/pack/%s/purchase", shareToken), strings.NewReader(bodyStr))
		req1.Header.Set("X-User-ID", fmt.Sprintf("%d", buyerID1))
		req1.Header.Set("Content-Type", "application/json")
		rr1 := httptest.NewRecorder()
		handlePurchaseFromDetail(rr1, req1)

		if rr1.Code != http.StatusOK {
			t.Logf("seed=%d: sufficient balance purchase: expected 200, got %d, body: %s", seed, rr1.Code, rr1.Body.String())
			return false
		}

		var resp1 map[string]interface{}
		if err := json.Unmarshal(rr1.Body.Bytes(), &resp1); err != nil {
			t.Logf("seed=%d: failed to parse purchase response: %v", seed, err)
			return false
		}
		if resp1["success"] != true {
			if resp1["insufficient_balance"] == true {
				t.Logf("seed=%d: got insufficient_balance with balance=%.0f >= totalCost=%d", seed, sufficientBalance, totalCost)
				return false
			}
			t.Logf("seed=%d: purchase did not succeed: %v", seed, resp1)
			return false
		}

		// Test case 2: insufficient balance (balance < totalCost)
		insufficientBalance := float64(r.Intn(totalCost)) // 0 to totalCost-1
		if totalCost <= 1 {
			insufficientBalance = 0
		}
		buyerRes2, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-BB2-%d-%d", seed, r.Intn(10000)), fmt.Sprintf("buyer2_%d", seed), fmt.Sprintf("bb2_%d_%d@test.com", seed, r.Intn(10000)), insufficientBalance,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create buyer2: %v", seed, err)
			return false
		}
		buyerID2, _ := buyerRes2.LastInsertId()

		req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/pack/%s/purchase", shareToken), strings.NewReader(bodyStr))
		req2.Header.Set("X-User-ID", fmt.Sprintf("%d", buyerID2))
		req2.Header.Set("Content-Type", "application/json")
		rr2 := httptest.NewRecorder()
		handlePurchaseFromDetail(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Logf("seed=%d: insufficient balance purchase: expected 200, got %d", seed, rr2.Code)
			return false
		}

		var resp2 map[string]interface{}
		if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil {
			t.Logf("seed=%d: failed to parse insufficient response: %v", seed, err)
			return false
		}
		if resp2["insufficient_balance"] != true {
			t.Logf("seed=%d: expected insufficient_balance=true with balance=%.0f < totalCost=%d, got: %v", seed, insufficientBalance, totalCost, resp2)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 (购买余额检查正确性) failed: %v", err)
	}
}

// Feature: qap-share-url, Property 9: 购买后余额扣减正确性
// **Validates: Requirements 6.5**
//
// For any successful purchase, the user's credits_balance after purchase should equal
// balance_before - (credits_price × quantity) for per_use, or
// balance_before - (credits_price × months) for subscription.
func TestProperty9_PurchaseBalanceDeductionCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Create a category
		catName := fmt.Sprintf("DeductCat_%d", seed)
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, 'test')", catName)
		if err != nil {
			t.Logf("seed=%d: failed to create category: %v", seed, err)
			return false
		}
		var categoryID int64
		db.QueryRow("SELECT id FROM categories WHERE name = ?", catName).Scan(&categoryID)

		// Create a pack author
		authorRes, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-DA-%d", seed), fmt.Sprintf("deduct_author_%d", seed), fmt.Sprintf("da_%d@test.com", seed), 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create author: %v", seed, err)
			return false
		}
		authorID, _ := authorRes.LastInsertId()

		// Pick a random paid share_mode
		paidModes := []string{"per_use", "subscription"}
		shareMode := paidModes[r.Intn(len(paidModes))]
		creditsPrice := r.Intn(100) + 1 // 1-100

		// Create a paid pack listing
		packName := fmt.Sprintf("DeductPack_%d_%d", seed, r.Intn(10000))
		_, shareToken, err := testInsertPackListing(authorID, categoryID, packName, shareMode, creditsPrice, "published")
		if err != nil {
			t.Logf("seed=%d: failed to create listing: %v", seed, err)
			return false
		}

		// Generate purchase parameters and calculate expected cost
		var quantity, months int
		var totalCost int
		var bodyStr string
		if shareMode == "per_use" {
			quantity = r.Intn(10) + 1 // 1-10
			totalCost = creditsPrice * quantity
			bodyStr = fmt.Sprintf(`{"quantity": %d}`, quantity)
		} else {
			months = r.Intn(12) + 1 // 1-12
			totalCost = creditsPrice * months
			bodyStr = fmt.Sprintf(`{"months": %d}`, months)
		}

		// Create a buyer with enough balance (totalCost + random extra 0-500)
		initialBalance := float64(totalCost + r.Intn(500))
		buyerRes, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-DB-%d-%d", seed, r.Intn(10000)), fmt.Sprintf("deduct_buyer_%d", seed), fmt.Sprintf("db_%d_%d@test.com", seed, r.Intn(10000)), initialBalance,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create buyer: %v", seed, err)
			return false
		}
		buyerID, _ := buyerRes.LastInsertId()

		// Execute purchase
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/pack/%s/purchase", shareToken), strings.NewReader(bodyStr))
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", buyerID))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handlePurchaseFromDetail(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: purchase: expected 200, got %d, body: %s", seed, rr.Code, rr.Body.String())
			return false
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("seed=%d: failed to parse response: %v", seed, err)
			return false
		}
		if resp["success"] != true {
			t.Logf("seed=%d: purchase did not succeed: %v", seed, resp)
			return false
		}

		// Verify balance after purchase
		var balanceAfter float64
		err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", buyerID).Scan(&balanceAfter)
		if err != nil {
			t.Logf("seed=%d: failed to query balance after purchase: %v", seed, err)
			return false
		}

		expectedBalance := initialBalance - float64(totalCost)
		if balanceAfter != expectedBalance {
			t.Logf("seed=%d: balance mismatch: initial=%.0f, totalCost=%d, expected=%.0f, got=%.0f (mode=%s, price=%d, qty=%d, months=%d)",
				seed, initialBalance, totalCost, expectedBalance, balanceAfter, shareMode, creditsPrice, quantity, months)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 (购买后余额扣减正确性) failed: %v", err)
	}
}
