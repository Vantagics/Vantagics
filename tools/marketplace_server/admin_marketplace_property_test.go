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

// Feature: marketplace-delist-restore, Property 1: 状态筛选正确性
// **Validates: Requirements 2.1, 2.2**
//
// For any database containing a mix of published and delisted pack listings,
// and for any valid status parameter ("published" or "delisted"), the API should
// return ONLY listings whose status matches the requested status parameter.
func TestProperty1_StatusFilterCorrectness(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// initDB pre-seeds categories (id=1 Shopify, id=2 BigCommerce, etc.)
	// Use category_id=1 for all test pack listings.

	// Insert a mix of published and delisted pack listings
	insertStmt := `INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, download_count, status, meta_info, created_at)
		VALUES (?, 1, X'00', ?, '', '', '', 'free', 0, 0, ?, '{}', datetime('now'))`

	statuses := []string{"published", "delisted"}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Insert 20 packs with random statuses to ensure a good mix
	for i := 0; i < 20; i++ {
		status := statuses[r.Intn(2)]
		_, err := db.Exec(insertStmt, i+1, "Pack_"+status+"_"+string(rune('A'+i)), status)
		if err != nil {
			t.Fatalf("failed to insert pack listing %d: %v", i, err)
		}
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Pick a random valid status to query
		queryStatus := statuses[rng.Intn(2)]

		req := httptest.NewRequest(http.MethodGet, "/api/admin/marketplace?status="+queryStatus, nil)
		rr := httptest.NewRecorder()
		handleAdminMarketplaceList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d", seed, rr.Code)
			return false
		}

		var resp struct {
			Packs []PackListingInfo `json:"packs"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("seed=%d: failed to unmarshal response: %v", seed, err)
			return false
		}

		// Property: every returned pack must have the requested status
		for _, pack := range resp.Packs {
			if pack.Status != queryStatus {
				t.Logf("seed=%d: expected status %q, got %q for pack %q (id=%d)",
					seed, queryStatus, pack.Status, pack.PackName, pack.ID)
				return false
			}
		}

		// Also verify we got at least some results (since we inserted both statuses)
		if len(resp.Packs) == 0 {
			// Count how many packs exist with this status in DB
			var count int
			db.QueryRow("SELECT COUNT(*) FROM pack_listings WHERE status = ?", queryStatus).Scan(&count)
			if count > 0 {
				t.Logf("seed=%d: expected packs with status %q but got none (db has %d)", seed, queryStatus, count)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (状态筛选正确性) failed: %v", err)
	}
}

// Feature: marketplace-delist-restore, Property 2: 无效状态参数拒绝
// **Validates: Requirements 2.4**
//
// For any string that is NOT "published" and NOT "delisted", calling the
// Marketplace_List_API with that string as the status parameter should return HTTP 400.
func TestProperty2_InvalidStatusRejection(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(status string) bool {
		// Skip the two valid values — we only test invalid ones.
		// Also skip empty string: per Requirement 2.3, absent/empty status
		// defaults to "published", which is correct behaviour tested elsewhere.
		if status == "published" || status == "delisted" || status == "" {
			return true
		}

		req := httptest.NewRequest(http.MethodGet, "/api/admin/marketplace?status="+status, nil)
		rr := httptest.NewRecorder()
		handleAdminMarketplaceList(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Logf("status=%q: expected 400, got %d", status, rr.Code)
			return false
		}

		var resp map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("status=%q: failed to unmarshal error response: %v", status, err)
			return false
		}

		if resp["error"] != "invalid_status" {
			t.Logf("status=%q: expected error 'invalid_status', got %q", status, resp["error"])
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (无效状态参数拒绝) failed: %v", err)
	}
}

// Feature: marketplace-delist-restore, Property 3: 恢复在售状态转换与审计
// **Validates: Requirements 3.2, 3.3**
//
// For any Pack_Listing with status "delisted" and any admin ID, calling the
// Relist_API should change the status to "published", set reviewed_by to the
// admin ID, and set reviewed_at to a non-empty value.
func TestProperty3_RelistStateTransitionAndAudit(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStmt := `INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, download_count, status, meta_info, created_at)
		VALUES (?, 1, X'00', ?, '', '', '', 'free', 0, 0, 'delisted', '{}', datetime('now'))`

	nextID := 1

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(adminIDRaw uint16) bool {
		// Use adminIDRaw+1 to ensure admin ID is always >= 1
		adminID := int(adminIDRaw) + 1

		// Insert a new delisted pack listing for this iteration
		packName := fmt.Sprintf("RelistPack_%d", nextID)
		res, err := db.Exec(insertStmt, nextID, packName)
		if err != nil {
			t.Logf("failed to insert pack listing: %v", err)
			return false
		}
		listingID, err := res.LastInsertId()
		if err != nil {
			t.Logf("failed to get last insert id: %v", err)
			return false
		}
		nextID++

		// Call handleAdminRelistPack
		url := fmt.Sprintf("/api/admin/marketplace/%d/relist", listingID)
		req := httptest.NewRequest(http.MethodPost, url, nil)
		req.Header.Set("X-Admin-ID", fmt.Sprintf("%d", adminID))
		rr := httptest.NewRecorder()
		handleAdminRelistPack(rr, req)

		// Verify response is 200
		if rr.Code != http.StatusOK {
			t.Logf("listingID=%d adminID=%d: expected 200, got %d, body=%s",
				listingID, adminID, rr.Code, rr.Body.String())
			return false
		}

		// Verify response body
		var resp map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("listingID=%d: failed to unmarshal response: %v", listingID, err)
			return false
		}
		if resp["status"] != "ok" {
			t.Logf("listingID=%d: expected status 'ok', got %q", listingID, resp["status"])
			return false
		}

		// Query DB to verify state transition and audit fields
		var status string
		var reviewedBy int64
		var reviewedAt string
		err = db.QueryRow("SELECT status, reviewed_by, reviewed_at FROM pack_listings WHERE id = ?", listingID).
			Scan(&status, &reviewedBy, &reviewedAt)
		if err != nil {
			t.Logf("listingID=%d: failed to query pack listing: %v", listingID, err)
			return false
		}

		// Property checks:
		// 1. Status must be "published"
		if status != "published" {
			t.Logf("listingID=%d: expected status 'published', got %q", listingID, status)
			return false
		}

		// 2. reviewed_by must equal the admin ID
		if reviewedBy != int64(adminID) {
			t.Logf("listingID=%d: expected reviewed_by=%d, got %d", listingID, adminID, reviewedBy)
			return false
		}

		// 3. reviewed_at must be non-empty
		if reviewedAt == "" {
			t.Logf("listingID=%d: expected reviewed_at to be non-empty", listingID)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (恢复在售状态转换与审计) failed: %v", err)
	}
}

// Feature: marketplace-delist-restore, Property 4: 恢复在售前置条件
// **Validates: Requirements 3.5**
//
// For any Pack_Listing whose current status is NOT "delisted" (e.g., "published",
// "pending", "rejected"), calling the Relist_API should return HTTP 409 and the
// status should remain unchanged.
func TestProperty4_RelistPrecondition(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStmt := `INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, download_count, status, meta_info, created_at)
		VALUES (?, 1, X'00', ?, '', '', '', 'free', 0, 0, ?, '{}', datetime('now'))`

	nonDelistedStatuses := []string{"published", "pending", "rejected"}
	nextID := 1

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Pick a random non-delisted status
		originalStatus := nonDelistedStatuses[rng.Intn(len(nonDelistedStatuses))]

		// Insert a pack listing with that status
		packName := fmt.Sprintf("PrecondPack_%d", nextID)
		res, err := db.Exec(insertStmt, nextID, packName, originalStatus)
		if err != nil {
			t.Logf("failed to insert pack listing: %v", err)
			return false
		}
		listingID, err := res.LastInsertId()
		if err != nil {
			t.Logf("failed to get last insert id: %v", err)
			return false
		}
		nextID++

		// Call handleAdminRelistPack
		url := fmt.Sprintf("/api/admin/marketplace/%d/relist", listingID)
		req := httptest.NewRequest(http.MethodPost, url, nil)
		req.Header.Set("X-Admin-ID", "1")
		rr := httptest.NewRecorder()
		handleAdminRelistPack(rr, req)

		// Verify response is 409
		if rr.Code != http.StatusConflict {
			t.Logf("seed=%d listingID=%d status=%q: expected 409, got %d, body=%s",
				seed, listingID, originalStatus, rr.Code, rr.Body.String())
			return false
		}

		// Verify error message
		var resp map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("seed=%d listingID=%d: failed to unmarshal response: %v", seed, listingID, err)
			return false
		}
		if resp["error"] != "can_only_relist_delisted" {
			t.Logf("seed=%d listingID=%d: expected error 'can_only_relist_delisted', got %q",
				seed, listingID, resp["error"])
			return false
		}

		// Query DB to verify status is unchanged
		var currentStatus string
		err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
		if err != nil {
			t.Logf("seed=%d listingID=%d: failed to query status: %v", seed, listingID, err)
			return false
		}
		if currentStatus != originalStatus {
			t.Logf("seed=%d listingID=%d: status changed from %q to %q after rejected relist",
				seed, listingID, originalStatus, currentStatus)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 (恢复在售前置条件) failed: %v", err)
	}
}
