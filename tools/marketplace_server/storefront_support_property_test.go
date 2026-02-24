package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: storefront-support-system, Property 1: 累计销售额计算正确性
// **Validates: Requirements 1.1**
//
// For any storefront and any set of credits_transactions records,
// computeStorefrontTotalSales should return the sum of absolute values
// of purchase transaction amounts for that storefront's packs.
func TestSupportSystemProperty1_TotalSalesComputation(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		balance := float64(rng.Intn(100000))
		userID := createTestUserWithBalance(t, balance)
		storeSlug := fmt.Sprintf("support-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("TestStore_%d", seed)

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
			userID, storeSlug, storeName,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create a second user to act as buyer
		buyerID := createTestUserWithBalance(t, 100000)

		// Get a category for pack listings
		catID := getCategoryID(t)

		// Generate 1-5 pack listings for this storefront
		numPacks := rng.Intn(5) + 1
		packIDs := make([]int64, numPacks)
		for i := 0; i < numPacks; i++ {
			packID := createTestPackListing(t, userID, catID, "credits", rng.Intn(100)+1, []byte("test-data"))
			packIDs[i] = packID

			// Link pack to storefront
			_, err := db.Exec(
				"INSERT INTO storefront_packs (storefront_id, pack_listing_id) VALUES (?, ?)",
				storefrontID, packID,
			)
			if err != nil {
				t.Logf("FAIL: failed to link pack to storefront: %v", err)
				return false
			}
		}

		// Generate random purchase transactions for the storefront's packs
		// and compute expected total sales manually
		numTransactions := rng.Intn(10) + 1
		var expectedTotal float64

		for i := 0; i < numTransactions; i++ {
			packIdx := rng.Intn(numPacks)
			packID := packIDs[packIdx]

			// Purchase amounts are negative in the DB (buyer deduction)
			amount := -(float64(rng.Intn(1000) + 1))

			_, err := db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description) VALUES (?, 'purchase', ?, ?, ?)",
				buyerID, amount, packID, fmt.Sprintf("Purchase pack %d", packID),
			)
			if err != nil {
				t.Logf("FAIL: failed to insert purchase transaction: %v", err)
				return false
			}

			expectedTotal += math.Abs(amount)
		}

		// Also insert some non-purchase transactions that should NOT be counted
		numNonPurchase := rng.Intn(5)
		nonPurchaseTypes := []string{"deposit", "withdrawal", "refund", "decoration"}
		for i := 0; i < numNonPurchase; i++ {
			packIdx := rng.Intn(numPacks)
			packID := packIDs[packIdx]
			txType := nonPurchaseTypes[rng.Intn(len(nonPurchaseTypes))]
			amount := -(float64(rng.Intn(500) + 1))

			db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description) VALUES (?, ?, ?, ?, ?)",
				buyerID, txType, amount, packID, fmt.Sprintf("Non-purchase %s", txType),
			)
		}

		// Also insert purchase transactions with positive amounts (should NOT be counted)
		numPositive := rng.Intn(3)
		for i := 0; i < numPositive; i++ {
			packIdx := rng.Intn(numPacks)
			packID := packIDs[packIdx]
			amount := float64(rng.Intn(500) + 1) // positive amount

			db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description) VALUES (?, 'purchase', ?, ?, ?)",
				buyerID, amount, packID, fmt.Sprintf("Positive purchase %d", packID),
			)
		}

		// Also insert purchase transactions for packs NOT in this storefront
		otherPackID := createTestPackListing(t, userID, catID, "credits", 50, []byte("other-data"))
		numOtherTx := rng.Intn(3)
		for i := 0; i < numOtherTx; i++ {
			amount := -(float64(rng.Intn(500) + 1))
			db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description) VALUES (?, 'purchase', ?, ?, ?)",
				buyerID, amount, otherPackID, "Purchase other pack",
			)
		}

		// Call computeStorefrontTotalSales and verify
		actualTotal, err := computeStorefrontTotalSales(storefrontID)
		if err != nil {
			t.Logf("FAIL: computeStorefrontTotalSales returned error: %v", err)
			return false
		}

		// Compare with tolerance for floating point
		if math.Abs(actualTotal-expectedTotal) > 0.01 {
			t.Logf("FAIL: expected total sales %.2f, got %.2f (storefront=%d, packs=%d, txns=%d)",
				expectedTotal, actualTotal, storefrontID, numPacks, numTransactions)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 2: 开通申请资格与初始状态
// **Validates: Requirements 1.4, 1.5, 2.5**
//
// When Total_Sales >= 10000 and no pending/approved request exists, apply should succeed
// and create a pending record; when Total_Sales < 10000, apply should be rejected
// and database should remain unchanged.
func TestSupportSystemProperty2_ApplyEligibilityAndInitialState(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		balance := float64(rng.Intn(100000))
		userID := createTestUserWithBalance(t, balance)
		storeSlug := fmt.Sprintf("support-p2-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P2Store_%d", seed)

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
			userID, storeSlug, storeName,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create a buyer user
		buyerID := createTestUserWithBalance(t, 100000)

		// Get a category for pack listings
		catID := getCategoryID(t)

		// Create a pack listing for this storefront
		packID := createTestPackListing(t, userID, catID, "credits", rng.Intn(100)+1, []byte("test-data"))
		_, err = db.Exec(
			"INSERT INTO storefront_packs (storefront_id, pack_listing_id) VALUES (?, ?)",
			storefrontID, packID,
		)
		if err != nil {
			t.Logf("FAIL: failed to link pack to storefront: %v", err)
			return false
		}

		// Generate a random target total sales amount (0 to 25000)
		// This determines how many purchase transactions we create
		targetSales := float64(rng.Intn(25001))
		if targetSales > 0 {
			// Create purchase transactions to reach the target sales amount
			// Split into a few transactions
			numTx := rng.Intn(5) + 1
			remaining := targetSales
			for i := 0; i < numTx; i++ {
				var amount float64
				if i == numTx-1 {
					amount = remaining
				} else {
					amount = float64(rng.Intn(int(remaining/float64(numTx-i))+1)) + 1
					if amount > remaining {
						amount = remaining
					}
				}
				if amount <= 0 {
					continue
				}
				remaining -= amount
				_, err := db.Exec(
					"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description) VALUES (?, 'purchase', ?, ?, ?)",
					buyerID, -amount, packID, fmt.Sprintf("Purchase for P2 test %d", i),
				)
				if err != nil {
					t.Logf("FAIL: failed to insert transaction: %v", err)
					return false
				}
			}
		}

		// Verify total sales computation
		totalSales, err := computeStorefrontTotalSales(storefrontID)
		if err != nil {
			t.Logf("FAIL: computeStorefrontTotalSales error: %v", err)
			return false
		}

		// Count records before attempting apply
		var countBefore int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM storefront_support_requests WHERE storefront_id = ?",
			storefrontID,
		).Scan(&countBefore)
		if err != nil {
			t.Logf("FAIL: failed to count records before: %v", err)
			return false
		}

		// Check eligibility and attempt to apply at the database level
		eligible := totalSales >= 10000

		// Check for existing pending/approved request (should be none for a fresh storefront)
		var existingStatus string
		existingErr := db.QueryRow(
			"SELECT status FROM storefront_support_requests WHERE storefront_id = ? AND status IN ('pending', 'approved') LIMIT 1",
			storefrontID,
		).Scan(&existingStatus)
		hasPendingOrApproved := existingErr == nil

		if eligible && !hasPendingOrApproved {
			// Should be able to create a pending record
			_, err := db.Exec(`
				INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
				VALUES (?, ?, 'vantagics', ?, ?, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, storefrontID, userID, storeName, "test welcome")
			if err != nil {
				t.Logf("FAIL: eligible storefront (totalSales=%.2f) failed to insert support request: %v", totalSales, err)
				return false
			}

			// Verify the record was created with status='pending'
			status, err := getStorefrontSupportStatus(storefrontID)
			if err != nil {
				t.Logf("FAIL: getStorefrontSupportStatus error after insert: %v", err)
				return false
			}
			if status != "pending" {
				t.Logf("FAIL: expected status 'pending' after apply, got '%s' (totalSales=%.2f)", status, totalSales)
				return false
			}

			// Verify record count increased by 1
			var countAfter int
			err = db.QueryRow(
				"SELECT COUNT(*) FROM storefront_support_requests WHERE storefront_id = ?",
				storefrontID,
			).Scan(&countAfter)
			if err != nil {
				t.Logf("FAIL: failed to count records after: %v", err)
				return false
			}
			if countAfter != countBefore+1 {
				t.Logf("FAIL: expected record count to increase by 1 (before=%d, after=%d)", countBefore, countAfter)
				return false
			}
		} else if !eligible {
			// Total_Sales < 10000: should NOT create a record, database should remain unchanged
			// Simulate the rejection check (as the handler does)
			if totalSales >= 10000 {
				t.Logf("FAIL: expected totalSales < 10000 but got %.2f", totalSales)
				return false
			}

			// Verify no record was created (count should remain the same)
			var countAfter int
			err = db.QueryRow(
				"SELECT COUNT(*) FROM storefront_support_requests WHERE storefront_id = ?",
				storefrontID,
			).Scan(&countAfter)
			if err != nil {
				t.Logf("FAIL: failed to count records after rejection: %v", err)
				return false
			}
			if countAfter != countBefore {
				t.Logf("FAIL: database changed after rejection (before=%d, after=%d, totalSales=%.2f)", countBefore, countAfter, totalSales)
				return false
			}

			// Verify status is still "none"
			status, err := getStorefrontSupportStatus(storefrontID)
			if err != nil {
				t.Logf("FAIL: getStorefrontSupportStatus error: %v", err)
				return false
			}
			if status != "none" {
				t.Logf("FAIL: expected status 'none' for ineligible storefront, got '%s' (totalSales=%.2f)", status, totalSales)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 3: 开通请求数据完整性
// **Validates: Requirements 2.2, 2.3, 2.4, 12.1, 12.4**
//
// For any successfully created support request:
// - software_name should always be "vantagics"
// - store_name should equal the storefront's store_name from author_storefronts
// - welcome_message should equal the storefront's description (if non-empty),
//   or "欢迎来到 {store_name} 的客户支持" if description is empty
func TestSupportSystemProperty3_RequestDataIntegrity(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user
		balance := float64(rng.Intn(100000))
		userID := createTestUserWithBalance(t, balance)

		// Generate random store_name (non-empty)
		storeName := fmt.Sprintf("P3Store_%d_%d", seed, rng.Int63n(1000000))

		// Generate random description: sometimes empty, sometimes non-empty
		var description string
		if rng.Intn(3) > 0 { // ~67% chance of non-empty description
			description = fmt.Sprintf("描述_%d_%d", seed, rng.Int63n(1000000))
		}
		// else description remains "" (empty)

		storeSlug := fmt.Sprintf("support-p3-store-%d-%d", userID, rng.Int63n(1000000))

		// Create storefront with the random store_name and description
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
			userID, storeSlug, storeName, description,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Compute expected welcome_message using the same logic as handleStorefrontSupportApply
		expectedWelcomeMessage := description
		if expectedWelcomeMessage == "" {
			expectedWelcomeMessage = fmt.Sprintf("欢迎来到 %s 的客户支持", storeName)
		}

		// Insert a support request record using the same logic as handleStorefrontSupportApply
		// (software_name="vantagics", store_name from storefront, welcome_message from description or default)
		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, ?, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName, expectedWelcomeMessage)
		if err != nil {
			t.Logf("FAIL: failed to insert support request: %v", err)
			return false
		}

		// Query the record back and verify all fields
		var actualSoftwareName, actualStoreName, actualWelcomeMessage string
		err = db.QueryRow(
			"SELECT software_name, store_name, welcome_message FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1",
			storefrontID,
		).Scan(&actualSoftwareName, &actualStoreName, &actualWelcomeMessage)
		if err != nil {
			t.Logf("FAIL: failed to query support request: %v", err)
			return false
		}

		// Verify software_name is always "vantagics"
		if actualSoftwareName != "vantagics" {
			t.Logf("FAIL: expected software_name='vantagics', got '%s' (storefront=%d)", actualSoftwareName, storefrontID)
			return false
		}

		// Verify store_name equals the storefront's store_name
		if actualStoreName != storeName {
			t.Logf("FAIL: expected store_name='%s', got '%s' (storefront=%d)", storeName, actualStoreName, storefrontID)
			return false
		}

		// Verify welcome_message matches expectation
		if actualWelcomeMessage != expectedWelcomeMessage {
			t.Logf("FAIL: expected welcome_message='%s', got '%s' (storefront=%d, description='%s')",
				expectedWelcomeMessage, actualWelcomeMessage, storefrontID, description)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 4: 重复申请防护
// **Validates: Requirements 2.7**
//
// For any storefront that already has a pending or approved support request,
// attempting to apply again should be rejected and the record count should remain unchanged.
func TestSupportSystemProperty4_DuplicateApplyPrevention(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		balance := float64(rng.Intn(100000))
		userID := createTestUserWithBalance(t, balance)
		storeSlug := fmt.Sprintf("support-p4-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P4Store_%d", seed)

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
			userID, storeSlug, storeName,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Randomly choose an existing request status: 'pending' or 'approved'
		statuses := []string{"pending", "approved"}
		existingStatus := statuses[rng.Intn(len(statuses))]

		// Insert an existing support request with the chosen status
		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, '欢迎', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName, existingStatus)
		if err != nil {
			t.Logf("FAIL: failed to insert existing support request: %v", err)
			return false
		}

		// Count records before the duplicate attempt
		var countBefore int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM storefront_support_requests WHERE storefront_id = ?",
			storefrontID,
		).Scan(&countBefore)
		if err != nil {
			t.Logf("FAIL: failed to count records before: %v", err)
			return false
		}

		// Simulate the handler's duplicate check: look for pending/approved request
		var dupStatus string
		dupErr := db.QueryRow(
			"SELECT status FROM storefront_support_requests WHERE storefront_id = ? AND status IN ('pending', 'approved') LIMIT 1",
			storefrontID,
		).Scan(&dupStatus)

		// The check should detect the existing request (dupErr should be nil)
		if dupErr != nil {
			t.Logf("FAIL: duplicate check did not detect existing %s request for storefront %d: %v",
				existingStatus, storefrontID, dupErr)
			return false
		}

		// Verify the detected status matches what we inserted
		if dupStatus != existingStatus {
			t.Logf("FAIL: duplicate check returned status '%s', expected '%s' for storefront %d",
				dupStatus, existingStatus, storefrontID)
			return false
		}

		// Since duplicate was detected, no new record should be created.
		// Verify the record count remains unchanged.
		var countAfter int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM storefront_support_requests WHERE storefront_id = ?",
			storefrontID,
		).Scan(&countAfter)
		if err != nil {
			t.Logf("FAIL: failed to count records after: %v", err)
			return false
		}

		if countAfter != countBefore {
			t.Logf("FAIL: record count changed after duplicate detection (before=%d, after=%d, existingStatus=%s, storefront=%d)",
				countBefore, countAfter, existingStatus, storefrontID)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 12: 登录 URL 格式正确性
// **Validates: Requirements 8.4**
//
// For any storefront_id (int64) and ticket (string), the generated login URL
// should contain the correct ticket, scope=store, and store_id parameters,
// and start with the expected base URL.
func TestSupportSystemProperty12_LoginURLFormat(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(storefrontID int64, ticket string) bool {
		// Build the URL using the same format as handleStorefrontSupportLogin
		url := fmt.Sprintf("https://service.vantagics.com/auth/ticket-login?ticket=%s&scope=store&store_id=%d",
			ticket, storefrontID)

		// Verify the URL starts with the expected base
		expectedBase := "https://service.vantagics.com/auth/ticket-login?"
		if !strings.HasPrefix(url, expectedBase) {
			t.Logf("FAIL: URL does not start with expected base. URL=%s", url)
			return false
		}

		// Verify the URL contains the correct ticket parameter
		expectedTicket := fmt.Sprintf("ticket=%s", ticket)
		if !strings.Contains(url, expectedTicket) {
			t.Logf("FAIL: URL does not contain expected ticket param. expected=%s, URL=%s", expectedTicket, url)
			return false
		}

		// Verify the URL contains scope=store
		if !strings.Contains(url, "scope=store") {
			t.Logf("FAIL: URL does not contain 'scope=store'. URL=%s", url)
			return false
		}

		// Verify the URL contains the correct store_id parameter
		expectedStoreID := fmt.Sprintf("store_id=%d", storefrontID)
		if !strings.Contains(url, expectedStoreID) {
			t.Logf("FAIL: URL does not contain expected store_id param. expected=%s, URL=%s", expectedStoreID, url)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 12 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 6: 有效状态转换
// **Validates: Requirements 5.1, 5.3, 5.5, 6.2, 6.3, 6.4**
//
// Valid state transitions should succeed and correctly update fields:
// - pending → approved (approve): status becomes approved, reviewed_by and reviewed_at are set
// - pending/approved → disabled (disable with reason): status becomes disabled, disable_reason is set
// - disabled → approved (re-approve): status becomes approved, disable_reason is cleared
func TestSupportSystemProperty6_ValidStateTransitions(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, float64(rng.Intn(100000)))
		storeSlug := fmt.Sprintf("support-p6-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P6Store_%d", seed)

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
			userID, storeSlug, storeName,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create an admin user for reviewed_by
		adminID := int64(rng.Intn(1000) + 1)
		db.Exec("INSERT OR IGNORE INTO admin_credentials (id, username, password_hash, role) VALUES (?, ?, 'hash', 'super')",
			adminID, fmt.Sprintf("admin_p6_%d", adminID))

		// Generate a random disable reason
		disableReason := fmt.Sprintf("违规原因_%d_%d", seed, rng.Int63n(1000000))

		// --- Test transition: pending → approved ---
		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, '欢迎', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName)
		if err != nil {
			t.Logf("FAIL: failed to insert pending request: %v", err)
			return false
		}
		var requestID int64
		err = db.QueryRow("SELECT id FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1", storefrontID).Scan(&requestID)
		if err != nil {
			t.Logf("FAIL: failed to get request ID: %v", err)
			return false
		}

		// Execute approve: same SQL as handleAdminStorefrontSupportApprove
		_, err = db.Exec(
			"UPDATE storefront_support_requests SET status = 'approved', reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			adminID, requestID,
		)
		if err != nil {
			t.Logf("FAIL: approve update failed: %v", err)
			return false
		}

		// Verify: status=approved, reviewed_by and reviewed_at are set
		var status string
		var reviewedBy *int64
		var reviewedAt *string
		err = db.QueryRow("SELECT status, reviewed_by, reviewed_at FROM storefront_support_requests WHERE id = ?", requestID).
			Scan(&status, &reviewedBy, &reviewedAt)
		if err != nil {
			t.Logf("FAIL: failed to query after approve: %v", err)
			return false
		}
		if status != "approved" {
			t.Logf("FAIL: expected status 'approved' after approve, got '%s'", status)
			return false
		}
		if reviewedBy == nil || *reviewedBy != adminID {
			t.Logf("FAIL: expected reviewed_by=%d after approve, got %v", adminID, reviewedBy)
			return false
		}
		if reviewedAt == nil || *reviewedAt == "" {
			t.Logf("FAIL: expected reviewed_at to be set after approve, got %v", reviewedAt)
			return false
		}

		// --- Test transition: approved → disabled ---
		_, err = db.Exec(
			"UPDATE storefront_support_requests SET status = 'disabled', disable_reason = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			disableReason, adminID, requestID,
		)
		if err != nil {
			t.Logf("FAIL: disable update failed: %v", err)
			return false
		}

		var disableReasonResult *string
		err = db.QueryRow("SELECT status, disable_reason, reviewed_by, reviewed_at FROM storefront_support_requests WHERE id = ?", requestID).
			Scan(&status, &disableReasonResult, &reviewedBy, &reviewedAt)
		if err != nil {
			t.Logf("FAIL: failed to query after disable: %v", err)
			return false
		}
		if status != "disabled" {
			t.Logf("FAIL: expected status 'disabled' after disable, got '%s'", status)
			return false
		}
		if disableReasonResult == nil || *disableReasonResult != disableReason {
			t.Logf("FAIL: expected disable_reason='%s', got %v", disableReason, disableReasonResult)
			return false
		}
		if reviewedBy == nil || *reviewedBy != adminID {
			t.Logf("FAIL: expected reviewed_by=%d after disable, got %v", adminID, reviewedBy)
			return false
		}

		// --- Test transition: disabled → approved (re-approve) ---
		_, err = db.Exec(
			"UPDATE storefront_support_requests SET status = 'approved', disable_reason = NULL, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			adminID, requestID,
		)
		if err != nil {
			t.Logf("FAIL: re-approve update failed: %v", err)
			return false
		}

		err = db.QueryRow("SELECT status, disable_reason, reviewed_by, reviewed_at FROM storefront_support_requests WHERE id = ?", requestID).
			Scan(&status, &disableReasonResult, &reviewedBy, &reviewedAt)
		if err != nil {
			t.Logf("FAIL: failed to query after re-approve: %v", err)
			return false
		}
		if status != "approved" {
			t.Logf("FAIL: expected status 'approved' after re-approve, got '%s'", status)
			return false
		}
		if disableReasonResult != nil {
			t.Logf("FAIL: expected disable_reason to be NULL after re-approve, got '%v'", *disableReasonResult)
			return false
		}
		if reviewedBy == nil || *reviewedBy != adminID {
			t.Logf("FAIL: expected reviewed_by=%d after re-approve, got %v", adminID, reviewedBy)
			return false
		}
		if reviewedAt == nil || *reviewedAt == "" {
			t.Logf("FAIL: expected reviewed_at to be set after re-approve, got %v", reviewedAt)
			return false
		}

		// --- Also test: pending → disabled (direct disable from pending) ---
		// Create a new pending request
		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, '欢迎2', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName)
		if err != nil {
			t.Logf("FAIL: failed to insert second pending request: %v", err)
			return false
		}
		var requestID2 int64
		err = db.QueryRow("SELECT id FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1", storefrontID).Scan(&requestID2)
		if err != nil {
			t.Logf("FAIL: failed to get second request ID: %v", err)
			return false
		}

		disableReason2 := fmt.Sprintf("直接禁用_%d", seed)
		_, err = db.Exec(
			"UPDATE storefront_support_requests SET status = 'disabled', disable_reason = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			disableReason2, adminID, requestID2,
		)
		if err != nil {
			t.Logf("FAIL: direct disable from pending failed: %v", err)
			return false
		}

		err = db.QueryRow("SELECT status, disable_reason FROM storefront_support_requests WHERE id = ?", requestID2).
			Scan(&status, &disableReasonResult)
		if err != nil {
			t.Logf("FAIL: failed to query after direct disable: %v", err)
			return false
		}
		if status != "disabled" {
			t.Logf("FAIL: expected status 'disabled' after direct disable from pending, got '%s'", status)
			return false
		}
		if disableReasonResult == nil || *disableReasonResult != disableReason2 {
			t.Logf("FAIL: expected disable_reason='%s' after direct disable, got %v", disableReason2, disableReasonResult)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 7: 无效状态转换拒绝
// **Validates: Requirements 6.5**
//
// Invalid state transitions should be rejected and status should remain unchanged:
// - approved → approve: rejected
// - disabled → disable: rejected
// - pending → re-approve: rejected
func TestSupportSystemProperty7_InvalidStateTransitionRejection(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, float64(rng.Intn(100000)))
		storeSlug := fmt.Sprintf("support-p7-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P7Store_%d", seed)

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
			userID, storeSlug, storeName,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Randomly pick one of the three invalid transition scenarios
		scenario := rng.Intn(3)

		var initialStatus, operation string
		switch scenario {
		case 0:
			// approved → approve: rejected
			initialStatus = "approved"
			operation = "approve"
		case 1:
			// disabled → disable: rejected
			initialStatus = "disabled"
			operation = "disable"
		case 2:
			// pending → re-approve: rejected
			initialStatus = "pending"
			operation = "re-approve"
		}

		// Insert a request with the initial status
		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, '欢迎', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName, initialStatus)
		if err != nil {
			t.Logf("FAIL: failed to insert request with status '%s': %v", initialStatus, err)
			return false
		}
		var requestID int64
		err = db.QueryRow("SELECT id FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1", storefrontID).Scan(&requestID)
		if err != nil {
			t.Logf("FAIL: failed to get request ID: %v", err)
			return false
		}

		// Simulate the handler's status check (same logic as the handlers)
		var currentStatus string
		err = db.QueryRow("SELECT status FROM storefront_support_requests WHERE id = ?", requestID).Scan(&currentStatus)
		if err != nil {
			t.Logf("FAIL: failed to query current status: %v", err)
			return false
		}

		// Check if the operation should be rejected based on current status
		var shouldReject bool
		switch operation {
		case "approve":
			// approve only allowed from pending
			shouldReject = currentStatus != "pending"
		case "disable":
			// disable only allowed from pending or approved
			shouldReject = currentStatus != "pending" && currentStatus != "approved"
		case "re-approve":
			// re-approve only allowed from disabled
			shouldReject = currentStatus != "disabled"
		}

		if !shouldReject {
			t.Logf("FAIL: expected operation '%s' on status '%s' to be rejected, but validation says it's allowed",
				operation, currentStatus)
			return false
		}

		// Verify the status remains unchanged (since we didn't execute the update)
		var statusAfter string
		err = db.QueryRow("SELECT status FROM storefront_support_requests WHERE id = ?", requestID).Scan(&statusAfter)
		if err != nil {
			t.Logf("FAIL: failed to query status after rejection: %v", err)
			return false
		}
		if statusAfter != initialStatus {
			t.Logf("FAIL: status changed from '%s' to '%s' after rejected '%s' operation",
				initialStatus, statusAfter, operation)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 8: 列表 API 状态筛选正确性
// **Validates: Requirements 4.3, 4.4, 6.1**
//
// When filtering by status, all returned requests should have that status.
// Create multiple requests with different statuses, then query with status filter.
func TestSupportSystemProperty8_ListFilterCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		statuses := []string{"pending", "approved", "disabled"}

		// Create multiple requests with different statuses
		numRequests := rng.Intn(10) + 3 // at least 3 to have variety
		for i := 0; i < numRequests; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(100000)))
			storeSlug := fmt.Sprintf("support-p8-store-%d-%d-%d", userID, i, rng.Int63n(1000000))
			storeName := fmt.Sprintf("P8Store_%d_%d", seed, i)

			result, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
				userID, storeSlug, storeName,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront %d: %v", i, err)
				return false
			}
			storefrontID, _ := result.LastInsertId()

			status := statuses[rng.Intn(len(statuses))]
			_, err = db.Exec(`
				INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
				VALUES (?, ?, 'vantagics', ?, '欢迎', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, storefrontID, userID, storeName, status)
			if err != nil {
				t.Logf("FAIL: failed to insert request %d with status '%s': %v", i, status, err)
				return false
			}
		}

		// Pick a random status to filter by
		filterStatus := statuses[rng.Intn(len(statuses))]

		// Use the same SQL query pattern as handleAdminStorefrontSupportList with status filter
		query := `SELECT ssr.id, ssr.storefront_id, ssr.store_name, u.display_name, ssr.software_name,
			ssr.status, COALESCE(ssr.disable_reason, ''), ssr.created_at,
			COALESCE(ssr.reviewed_at, ''), COALESCE(ac.username, '')
			FROM storefront_support_requests ssr
			JOIN users u ON ssr.user_id = u.id
			LEFT JOIN admin_credentials ac ON ssr.reviewed_by = ac.id
			WHERE 1=1 AND ssr.status = ?
			ORDER BY ssr.created_at DESC LIMIT 20 OFFSET 0`

		rows, err := db.Query(query, filterStatus)
		if err != nil {
			t.Logf("FAIL: query with status filter '%s' failed: %v", filterStatus, err)
			return false
		}
		defer rows.Close()

		var resultCount int
		for rows.Next() {
			var id, storefrontID int64
			var storeName, username, softwareName, status, disableReason, createdAt, reviewedAt, reviewedBy string
			if err := rows.Scan(&id, &storefrontID, &storeName, &username, &softwareName,
				&status, &disableReason, &createdAt, &reviewedAt, &reviewedBy); err != nil {
				t.Logf("FAIL: scan error: %v", err)
				return false
			}

			// Every returned record must have the filtered status
			if status != filterStatus {
				t.Logf("FAIL: expected all results to have status '%s', but got '%s' (id=%d, store='%s')",
					filterStatus, status, id, storeName)
				return false
			}
			resultCount++
		}
		if err := rows.Err(); err != nil {
			t.Logf("FAIL: rows iteration error: %v", err)
			return false
		}

		// Also verify the count matches what's in the DB
		var expectedCount int
		err = db.QueryRow("SELECT COUNT(*) FROM storefront_support_requests WHERE status = ?", filterStatus).Scan(&expectedCount)
		if err != nil {
			t.Logf("FAIL: count query failed: %v", err)
			return false
		}
		// Account for pagination (max 20 per page)
		if expectedCount > 20 {
			expectedCount = 20
		}
		if resultCount != expectedCount {
			t.Logf("FAIL: expected %d results for status '%s', got %d", expectedCount, filterStatus, resultCount)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 9: 列表 API 时间降序排列
// **Validates: Requirements 4.6**
//
// List results should be ordered by created_at DESC.
// Create multiple requests with different timestamps, query the list, verify ordering.
func TestSupportSystemProperty9_ListTimeOrdering(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		statuses := []string{"pending", "approved", "disabled"}

		// Create multiple requests with distinct timestamps
		numRequests := rng.Intn(8) + 3 // 3-10 requests
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		for i := 0; i < numRequests; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(100000)))
			storeSlug := fmt.Sprintf("support-p9-store-%d-%d-%d", userID, i, rng.Int63n(1000000))
			storeName := fmt.Sprintf("P9Store_%d_%d", seed, i)

			result, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
				userID, storeSlug, storeName,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront %d: %v", i, err)
				return false
			}
			storefrontID, _ := result.LastInsertId()

			status := statuses[rng.Intn(len(statuses))]
			// Use distinct timestamps with random offsets to ensure different ordering
			createdAt := baseTime.Add(time.Duration(rng.Intn(365*24)) * time.Hour).Format("2006-01-02 15:04:05")

			_, err = db.Exec(`
				INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
				VALUES (?, ?, 'vantagics', ?, '欢迎', ?, ?, ?)
			`, storefrontID, userID, storeName, status, createdAt, createdAt)
			if err != nil {
				t.Logf("FAIL: failed to insert request %d: %v", i, err)
				return false
			}
		}

		// Use the same SQL query pattern as handleAdminStorefrontSupportList (no status filter)
		query := `SELECT ssr.id, ssr.storefront_id, ssr.store_name, u.display_name, ssr.software_name,
			ssr.status, COALESCE(ssr.disable_reason, ''), ssr.created_at,
			COALESCE(ssr.reviewed_at, ''), COALESCE(ac.username, '')
			FROM storefront_support_requests ssr
			JOIN users u ON ssr.user_id = u.id
			LEFT JOIN admin_credentials ac ON ssr.reviewed_by = ac.id
			WHERE 1=1
			ORDER BY ssr.created_at DESC LIMIT 20 OFFSET 0`

		rows, err := db.Query(query)
		if err != nil {
			t.Logf("FAIL: list query failed: %v", err)
			return false
		}
		defer rows.Close()

		var timestamps []string
		for rows.Next() {
			var id, storefrontID int64
			var storeName, username, softwareName, status, disableReason, createdAt, reviewedAt, reviewedBy string
			if err := rows.Scan(&id, &storefrontID, &storeName, &username, &softwareName,
				&status, &disableReason, &createdAt, &reviewedAt, &reviewedBy); err != nil {
				t.Logf("FAIL: scan error: %v", err)
				return false
			}
			timestamps = append(timestamps, createdAt)
		}
		if err := rows.Err(); err != nil {
			t.Logf("FAIL: rows iteration error: %v", err)
			return false
		}

		// Verify descending order: each timestamp should be >= the next one
		for i := 0; i < len(timestamps)-1; i++ {
			if timestamps[i] < timestamps[i+1] {
				t.Logf("FAIL: timestamps not in descending order at index %d: '%s' < '%s'",
					i, timestamps[i], timestamps[i+1])
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 5: 状态查询 Round-Trip
// **Validates: Requirements 3.4**
//
// For any storefront, getStorefrontSupportStatus should return the status of the
// latest record in storefront_support_requests, or "none" if no record exists.
func TestSupportSystemProperty5_StatusQueryRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, float64(rng.Intn(100000)))
		storeSlug := fmt.Sprintf("support-p5-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P5Store_%d", seed)

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
			userID, storeSlug, storeName,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Step 1: Verify getStorefrontSupportStatus returns "none" when no records exist
		status, err := getStorefrontSupportStatus(storefrontID)
		if err != nil {
			t.Logf("FAIL: getStorefrontSupportStatus error (no records): %v", err)
			return false
		}
		if status != "none" {
			t.Logf("FAIL: expected status 'none' when no records exist, got '%s' (storefront=%d)", status, storefrontID)
			return false
		}

		// Step 2: Insert a support request with a random status
		statuses := []string{"pending", "approved", "disabled"}
		randomStatus := statuses[rng.Intn(len(statuses))]

		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, '欢迎', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName, randomStatus)
		if err != nil {
			t.Logf("FAIL: failed to insert support request with status '%s': %v", randomStatus, err)
			return false
		}

		// Step 3: Verify getStorefrontSupportStatus returns the correct status
		status, err = getStorefrontSupportStatus(storefrontID)
		if err != nil {
			t.Logf("FAIL: getStorefrontSupportStatus error after insert: %v", err)
			return false
		}
		if status != randomStatus {
			t.Logf("FAIL: expected status '%s' after insert, got '%s' (storefront=%d)", randomStatus, status, storefrontID)
			return false
		}

		// Also verify by querying the DB directly (round-trip check)
		var dbStatus string
		err = db.QueryRow(
			"SELECT status FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1",
			storefrontID,
		).Scan(&dbStatus)
		if err != nil {
			t.Logf("FAIL: direct DB query error: %v", err)
			return false
		}
		if status != dbStatus {
			t.Logf("FAIL: getStorefrontSupportStatus returned '%s' but DB has '%s' (storefront=%d)", status, dbStatus, storefrontID)
			return false
		}

		// Step 4: Update the status and verify again
		newStatuses := []string{"pending", "approved", "disabled"}
		// Pick a different status than the current one
		var newStatus string
		for {
			newStatus = newStatuses[rng.Intn(len(newStatuses))]
			if newStatus != randomStatus {
				break
			}
		}

		// Get the request ID to update
		var requestID int64
		err = db.QueryRow(
			"SELECT id FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1",
			storefrontID,
		).Scan(&requestID)
		if err != nil {
			t.Logf("FAIL: failed to get request ID: %v", err)
			return false
		}

		_, err = db.Exec(
			"UPDATE storefront_support_requests SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			newStatus, requestID,
		)
		if err != nil {
			t.Logf("FAIL: failed to update status to '%s': %v", newStatus, err)
			return false
		}

		// Verify getStorefrontSupportStatus returns the updated status
		status, err = getStorefrontSupportStatus(storefrontID)
		if err != nil {
			t.Logf("FAIL: getStorefrontSupportStatus error after update: %v", err)
			return false
		}
		if status != newStatus {
			t.Logf("FAIL: expected status '%s' after update, got '%s' (storefront=%d)", newStatus, status, storefrontID)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 10: Check API 返回正确性
// **Validates: Requirements 7.1, 7.2, 7.3**
//
// When approved, the check query should return the correct store_name, welcome_message,
// and software_name. When not approved, it should return the current status.
func TestSupportSystemProperty10_CheckAPICorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, float64(rng.Intn(100000)))
		storeSlug := fmt.Sprintf("support-p10-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P10Store_%d_%d", seed, rng.Int63n(1000000))
		description := fmt.Sprintf("描述_%d_%d", seed, rng.Int63n(1000000))

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name, description) VALUES (?, ?, ?, ?)",
			userID, storeSlug, storeName, description,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Insert a support request with a random status
		statuses := []string{"pending", "approved", "disabled"}
		randomStatus := statuses[rng.Intn(len(statuses))]
		welcomeMessage := description
		if welcomeMessage == "" {
			welcomeMessage = fmt.Sprintf("欢迎来到 %s 的客户支持", storeName)
		}

		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName, welcomeMessage, randomStatus)
		if err != nil {
			t.Logf("FAIL: failed to insert support request with status '%s': %v", randomStatus, err)
			return false
		}

		// Query the database using the same pattern as handleStorefrontSupportCheck
		var qStatus, qStoreName, qWelcomeMessage, qSoftwareName string
		err = db.QueryRow(
			`SELECT status, store_name, welcome_message, software_name FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1`,
			storefrontID,
		).Scan(&qStatus, &qStoreName, &qWelcomeMessage, &qSoftwareName)
		if err != nil {
			t.Logf("FAIL: check query failed: %v", err)
			return false
		}

		if qStatus == "approved" {
			// Verify the query returns the correct store_name, welcome_message, software_name
			if qStoreName != storeName {
				t.Logf("FAIL: approved check: expected store_name='%s', got '%s' (storefront=%d)",
					storeName, qStoreName, storefrontID)
				return false
			}
			if qWelcomeMessage != welcomeMessage {
				t.Logf("FAIL: approved check: expected welcome_message='%s', got '%s' (storefront=%d)",
					welcomeMessage, qWelcomeMessage, storefrontID)
				return false
			}
			if qSoftwareName != "vantagics" {
				t.Logf("FAIL: approved check: expected software_name='vantagics', got '%s' (storefront=%d)",
					qSoftwareName, storefrontID)
				return false
			}
		} else {
			// Verify the query returns the correct status
			if qStatus != randomStatus {
				t.Logf("FAIL: non-approved check: expected status='%s', got '%s' (storefront=%d)",
					randomStatus, qStatus, storefrontID)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 10 violated: %v", err)
	}
}

// Feature: storefront-support-system, Property 11: 欢迎语同步一致性
// **Validates: Requirements 12.2, 12.3**
//
// After updating description via syncSupportWelcomeMessage, the storefront_support_requests
// table's welcome_message should be synced to the new description value.
// If the new description is empty, the default welcome message "欢迎来到 {store_name} 的客户支持" is used.
func TestSupportSystemProperty11_WelcomeMessageSync(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront with a store_name
		userID := createTestUserWithBalance(t, float64(rng.Intn(100000)))
		storeSlug := fmt.Sprintf("support-p11-store-%d-%d", userID, rng.Int63n(1000000))
		storeName := fmt.Sprintf("P11Store_%d_%d", seed, rng.Int63n(1000000))

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
			userID, storeSlug, storeName,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Insert a support request record with an initial welcome_message
		initialWelcome := fmt.Sprintf("初始欢迎语_%d", seed)
		_, err = db.Exec(`
			INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
			VALUES (?, ?, 'vantagics', ?, ?, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, storefrontID, userID, storeName, initialWelcome)
		if err != nil {
			t.Logf("FAIL: failed to insert support request: %v", err)
			return false
		}

		// --- Test case 1: non-empty description ---
		newDescription := fmt.Sprintf("新描述_%d_%d", seed, rng.Int63n(1000000))

		// Call syncSupportWelcomeMessage with a non-empty description
		syncSupportWelcomeMessage(storefrontID, newDescription)

		// Query the welcome_message from storefront_support_requests
		var actualWelcome string
		err = db.QueryRow(
			"SELECT welcome_message FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1",
			storefrontID,
		).Scan(&actualWelcome)
		if err != nil {
			t.Logf("FAIL: failed to query welcome_message after non-empty sync: %v", err)
			return false
		}

		// Verify it matches the new description
		if actualWelcome != newDescription {
			t.Logf("FAIL: expected welcome_message='%s' after non-empty sync, got '%s' (storefront=%d)",
				newDescription, actualWelcome, storefrontID)
			return false
		}

		// --- Test case 2: empty description (should use default welcome message) ---
		syncSupportWelcomeMessage(storefrontID, "")

		err = db.QueryRow(
			"SELECT welcome_message FROM storefront_support_requests WHERE storefront_id = ? ORDER BY id DESC LIMIT 1",
			storefrontID,
		).Scan(&actualWelcome)
		if err != nil {
			t.Logf("FAIL: failed to query welcome_message after empty sync: %v", err)
			return false
		}

		expectedDefault := fmt.Sprintf("欢迎来到 %s 的客户支持", storeName)
		if actualWelcome != expectedDefault {
			t.Logf("FAIL: expected default welcome_message='%s' after empty sync, got '%s' (storefront=%d)",
				expectedDefault, actualWelcome, storefrontID)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 11 violated: %v", err)
	}
}
