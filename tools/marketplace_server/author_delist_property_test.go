package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: author-self-delist, Property 1: 下架后状态一致性
// Validates: Requirements 3.2
//
// For any Pack_Listing owned by a user, when the owner performs a delist operation
// on a Pack_Listing with status='published', the status should become 'delisted'.
func TestProperty_AuthorDelistStatusConsistency(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test user (the author/owner)
	res, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email) VALUES ('test', 'delist-prop1', 'DelistAuthor', 'delist@example.com')")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get user id: %v", err)
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	iteration := 0

	f := func(seed int64) bool {
		iteration++
		rng := rand.New(rand.NewSource(seed))

		// Generate a random pack name for this iteration
		packName := fmt.Sprintf("DelistPack_%d_%d", iteration, rng.Intn(100000))

		// Insert a pack_listing with status='published' owned by our user
		insertPack := `INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, meta_info, created_at)
			VALUES (?, 1, X'00', ?, 'test desc', '', 'TestAuthor', 'free', 0, 'published', '{}', datetime('now'))`

		packRes, err := db.Exec(insertPack, userID, packName)
		if err != nil {
			t.Logf("iteration=%d: failed to insert pack: %v", iteration, err)
			return false
		}
		listingID, err := packRes.LastInsertId()
		if err != nil {
			t.Logf("iteration=%d: failed to get listing id: %v", iteration, err)
			return false
		}

		// Verify the listing is in 'published' state before delist
		var beforeStatus string
		err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&beforeStatus)
		if err != nil {
			t.Logf("iteration=%d: failed to read before status: %v", iteration, err)
			return false
		}
		if beforeStatus != "published" {
			t.Logf("iteration=%d: expected before status 'published', got '%s'", iteration, beforeStatus)
			return false
		}

		// Simulate POST request to handleAuthorDelistPack with correct owner
		form := url.Values{}
		form.Set("listing_id", fmt.Sprintf("%d", listingID))
		req := httptest.NewRequest(http.MethodPost, "/user/author/delist-pack", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()

		handleAuthorDelistPack(rr, req)

		// Verify the handler responded with a redirect (302)
		if rr.Code != http.StatusFound {
			t.Logf("iteration=%d: expected HTTP 302, got %d", iteration, rr.Code)
			return false
		}

		// Property: after delist, the status must be 'delisted'
		var afterStatus string
		err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&afterStatus)
		if err != nil {
			t.Logf("iteration=%d: failed to read after status: %v", iteration, err)
			return false
		}
		if afterStatus != "delisted" {
			t.Logf("iteration=%d: expected after status 'delisted', got '%s' (listingID=%d, userID=%d)", iteration, afterStatus, listingID, userID)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (下架后状态一致性) failed: %v", err)
	}
}

// Feature: author-self-delist, Property 2: 非所有者无法下架
// Validates: Requirements 3.4
//
// For any Pack_Listing and any non-owner user, attempting to delist that
// Pack_Listing should be rejected with HTTP 403, and the status remains unchanged.
func TestProperty_AuthorDelistNonOwnerRejected(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create user A (the owner)
	resA, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email) VALUES ('test', 'delist-prop2-owner', 'OwnerUser', 'owner@example.com')")
	if err != nil {
		t.Fatalf("failed to create owner user: %v", err)
	}
	ownerID, err := resA.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get owner user id: %v", err)
	}

	// Create user B (the non-owner)
	resB, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email) VALUES ('test', 'delist-prop2-nonowner', 'NonOwnerUser', 'nonowner@example.com')")
	if err != nil {
		t.Fatalf("failed to create non-owner user: %v", err)
	}
	nonOwnerID, err := resB.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get non-owner user id: %v", err)
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	iteration := 0

	f := func(seed int64) bool {
		iteration++
		rng := rand.New(rand.NewSource(seed))

		// Generate a random pack name for this iteration
		packName := fmt.Sprintf("NonOwnerPack_%d_%d", iteration, rng.Intn(100000))

		// Insert a pack_listing with status='published' owned by user A (owner)
		insertPack := `INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, meta_info, created_at)
			VALUES (?, 1, X'00', ?, 'test desc', '', 'TestAuthor', 'free', 0, 'published', '{}', datetime('now'))`

		packRes, err := db.Exec(insertPack, ownerID, packName)
		if err != nil {
			t.Logf("iteration=%d: failed to insert pack: %v", iteration, err)
			return false
		}
		listingID, err := packRes.LastInsertId()
		if err != nil {
			t.Logf("iteration=%d: failed to get listing id: %v", iteration, err)
			return false
		}

		// Simulate POST request from user B (non-owner) trying to delist
		form := url.Values{}
		form.Set("listing_id", fmt.Sprintf("%d", listingID))
		req := httptest.NewRequest(http.MethodPost, "/user/author/delist-pack", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", nonOwnerID))
		rr := httptest.NewRecorder()

		handleAuthorDelistPack(rr, req)

		// Property: non-owner should receive HTTP 403 Forbidden
		if rr.Code != http.StatusForbidden {
			t.Logf("iteration=%d: expected HTTP 403, got %d (listingID=%d, ownerID=%d, nonOwnerID=%d)", iteration, rr.Code, listingID, ownerID, nonOwnerID)
			return false
		}

		// Property: pack_listing status must remain 'published' (unchanged)
		var afterStatus string
		err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&afterStatus)
		if err != nil {
			t.Logf("iteration=%d: failed to read after status: %v", iteration, err)
			return false
		}
		if afterStatus != "published" {
			t.Logf("iteration=%d: expected status to remain 'published', got '%s' (listingID=%d)", iteration, afterStatus, listingID)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (非所有者无法下架) failed: %v", err)
	}
}

// Feature: author-self-delist, Property 3: 仅 published 状态可下架
// Validates: Requirements 3.5
//
// For any Pack_Listing whose status is not 'published' (e.g. pending, rejected, delisted),
// attempting to delist it should fail, and the status should remain unchanged.
func TestProperty_AuthorDelistOnlyPublished(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test user (the owner)
	res, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email) VALUES ('test', 'delist-prop3', 'DelistProp3Author', 'delist-prop3@example.com')")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get user id: %v", err)
	}

	nonPublishedStatuses := []string{"pending", "rejected", "delisted"}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	iteration := 0

	f := func(seed int64) bool {
		iteration++
		rng := rand.New(rand.NewSource(seed))

		// Randomly pick a non-published status
		chosenStatus := nonPublishedStatuses[rng.Intn(len(nonPublishedStatuses))]

		// Generate a random pack name for this iteration
		packName := fmt.Sprintf("NonPubPack_%d_%d", iteration, rng.Intn(100000))

		// Insert a pack_listing with the chosen non-published status, owned by our user
		insertPack := `INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, meta_info, created_at)
			VALUES (?, 1, X'00', ?, 'test desc', '', 'TestAuthor', 'free', 0, ?, '{}', datetime('now'))`

		packRes, err := db.Exec(insertPack, userID, packName, chosenStatus)
		if err != nil {
			t.Logf("iteration=%d: failed to insert pack: %v", iteration, err)
			return false
		}
		listingID, err := packRes.LastInsertId()
		if err != nil {
			t.Logf("iteration=%d: failed to get listing id: %v", iteration, err)
			return false
		}

		// Simulate POST request from the owner trying to delist a non-published pack
		form := url.Values{}
		form.Set("listing_id", fmt.Sprintf("%d", listingID))
		req := httptest.NewRequest(http.MethodPost, "/user/author/delist-pack", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()

		handleAuthorDelistPack(rr, req)

		// Property: should get a redirect (302) with error parameter (not 403, since user IS the owner)
		if rr.Code != http.StatusFound {
			t.Logf("iteration=%d: expected HTTP 302, got %d (status=%s, listingID=%d)", iteration, rr.Code, chosenStatus, listingID)
			return false
		}
		location := rr.Header().Get("Location")
		if !strings.Contains(location, "error=") {
			t.Logf("iteration=%d: expected redirect with error param, got Location=%q (status=%s)", iteration, location, chosenStatus)
			return false
		}

		// Property: pack_listing status must remain unchanged (same as the original non-published status)
		var afterStatus string
		err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&afterStatus)
		if err != nil {
			t.Logf("iteration=%d: failed to read after status: %v", iteration, err)
			return false
		}
		if afterStatus != chosenStatus {
			t.Logf("iteration=%d: expected status to remain '%s', got '%s' (listingID=%d)", iteration, chosenStatus, afterStatus, listingID)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (仅 published 状态可下架) failed: %v", err)
	}
}
