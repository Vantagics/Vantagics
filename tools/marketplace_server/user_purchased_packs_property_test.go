package main

import (
	"math/rand"
	"testing"
	"time"
)

// Feature: marketplace-billing-management, Property 6: 购买-删除-再购买往返一致性
// For any user and any pack listing:
// 1. After first purchase (upsertUserPurchasedPack), user_purchased_packs should have a record with is_hidden=0
// 2. After soft delete (softDeleteUserPurchasedPack), is_hidden should be 1
// 3. After re-purchase (upsertUserPurchasedPack again), is_hidden should be back to 0
// 4. Throughout the cycle, (user_id, listing_id) should always have exactly ONE record (no duplicates)
// Validates: Requirements 5.2, 5.3, 6.1, 6.2, 6.3
func TestProperty6_PurchaseDeleteRepurchaseRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Generate random balance and price
		balance := float64(rng.Intn(1000) + 100)
		userID := createTestUserWithBalance(t, balance)

		modes := []string{"free", "per_use", "subscription"}
		mode := modes[rng.Intn(len(modes))]
		var price int
		switch mode {
		case "free":
			price = 0
		case "per_use":
			price = rng.Intn(100) + 1
		case "subscription":
			price = rng.Intn(901) + 100
		}

		packID := createTestPackListing(t, userID, 1, mode, price, []byte("test-data"))

		// Step 1: First purchase — upsert should create record with is_hidden=0
		err := upsertUserPurchasedPack(userID, packID)
		if err != nil {
			t.Errorf("iteration %d: upsert (first purchase) failed: %v", i, err)
			cleanup()
			continue
		}

		isHidden, count := queryPurchasedPackState(t, userID, packID)
		if count != 1 {
			t.Errorf("iteration %d: after first purchase, expected 1 record, got %d", i, count)
			cleanup()
			continue
		}
		if isHidden != 0 {
			t.Errorf("iteration %d: after first purchase, expected is_hidden=0, got %d", i, isHidden)
			cleanup()
			continue
		}

		// Step 2: Soft delete — is_hidden should become 1
		err = softDeleteUserPurchasedPack(userID, packID)
		if err != nil {
			t.Errorf("iteration %d: soft delete failed: %v", i, err)
			cleanup()
			continue
		}

		isHidden, count = queryPurchasedPackState(t, userID, packID)
		if count != 1 {
			t.Errorf("iteration %d: after soft delete, expected 1 record, got %d", i, count)
			cleanup()
			continue
		}
		if isHidden != 1 {
			t.Errorf("iteration %d: after soft delete, expected is_hidden=1, got %d", i, isHidden)
			cleanup()
			continue
		}

		// Step 3: Re-purchase — is_hidden should be restored to 0
		err = upsertUserPurchasedPack(userID, packID)
		if err != nil {
			t.Errorf("iteration %d: upsert (re-purchase) failed: %v", i, err)
			cleanup()
			continue
		}

		isHidden, count = queryPurchasedPackState(t, userID, packID)
		if count != 1 {
			t.Errorf("iteration %d: after re-purchase, expected 1 record, got %d", i, count)
			cleanup()
			continue
		}
		if isHidden != 0 {
			t.Errorf("iteration %d: after re-purchase, expected is_hidden=0, got %d", i, isHidden)
			cleanup()
			continue
		}

		cleanup()
	}
}

// Feature: marketplace-billing-management, Property 5: 软删除隐藏与查询过滤
// For any user and any purchased pack:
// 1. After soft delete, the pack's is_hidden field should be 1
// 2. The pack should NOT appear in query results that filter by (is_hidden IS NULL OR is_hidden = 0)
// 3. The database record should still exist (not physically deleted)
// Validates: Requirements 4.3, 4.5, 5.4, 5.5
func TestProperty5_SoftDeleteHidesFromQuery(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		// Generate random balance and price
		balance := float64(rng.Intn(1000) + 100)
		userID := createTestUserWithBalance(t, balance)

		modes := []string{"free", "per_use", "subscription"}
		mode := modes[rng.Intn(len(modes))]
		var price int
		switch mode {
		case "free":
			price = 0
		case "per_use":
			price = rng.Intn(100) + 1
		case "subscription":
			price = rng.Intn(901) + 100
		}

		packID := createTestPackListing(t, userID, 1, mode, price, []byte("test-data"))

		// Step 1: Purchase the pack (upsert creates record with is_hidden=0)
		err := upsertUserPurchasedPack(userID, packID)
		if err != nil {
			t.Errorf("iteration %d: upsert failed: %v", i, err)
			cleanup()
			continue
		}

		// Step 2: Soft delete the pack
		err = softDeleteUserPurchasedPack(userID, packID)
		if err != nil {
			t.Errorf("iteration %d: soft delete failed: %v", i, err)
			cleanup()
			continue
		}

		// Verify 1: is_hidden should be 1
		isHidden, count := queryPurchasedPackState(t, userID, packID)
		if count != 1 {
			t.Errorf("iteration %d: after soft delete, expected 1 record, got %d", i, count)
			cleanup()
			continue
		}
		if isHidden != 1 {
			t.Errorf("iteration %d: after soft delete, expected is_hidden=1, got %d", i, isHidden)
			cleanup()
			continue
		}

		// Verify 2: Record still exists in DB (not physically deleted)
		var totalRecords int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?",
			userID, packID,
		).Scan(&totalRecords)
		if err != nil {
			t.Errorf("iteration %d: failed to count records: %v", i, err)
			cleanup()
			continue
		}
		if totalRecords != 1 {
			t.Errorf("iteration %d: expected record to still exist in DB, got count=%d", i, totalRecords)
			cleanup()
			continue
		}

		// Verify 3: Filtered query (same as handleUserDashboard) returns 0 results
		// This uses the same LEFT JOIN + is_hidden filter pattern as the dashboard query
		var filteredCount int
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM user_purchased_packs upp
			WHERE upp.user_id = ? AND upp.listing_id = ?
			  AND (upp.is_hidden IS NULL OR upp.is_hidden = 0)
		`, userID, packID).Scan(&filteredCount)
		if err != nil {
			t.Errorf("iteration %d: filtered query failed: %v", i, err)
			cleanup()
			continue
		}
		if filteredCount != 0 {
			t.Errorf("iteration %d: expected filtered query to return 0 results for soft-deleted pack, got %d", i, filteredCount)
		}

		cleanup()
	}
}

// queryPurchasedPackState returns the is_hidden value and record count for a (user_id, listing_id) pair.
func queryPurchasedPackState(t *testing.T, userID, listingID int64) (isHidden int, count int) {
	t.Helper()
	err := db.QueryRow(
		"SELECT COUNT(*) FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?",
		userID, listingID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query record count: %v", err)
	}
	if count > 0 {
		err = db.QueryRow(
			"SELECT is_hidden FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?",
			userID, listingID,
		).Scan(&isHidden)
		if err != nil {
			t.Fatalf("failed to query is_hidden: %v", err)
		}
	}
	return
}
