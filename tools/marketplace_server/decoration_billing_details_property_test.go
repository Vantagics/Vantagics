package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/xuri/excelize/v2"
)

// Feature: decoration-billing-details, Property 1: 列表响应包含完整字段
// **Validates: Requirements 1.1, 2.4**
//
// For any set of decoration billing records, every record returned by
// GET /admin/api/billing/decoration contains id, user_id, display_name,
// store_name, amount, description, created_at fields, and the response
// contains total, total_credits, page, page_size fields.
func TestDecorationBillingDetailsProperty1_ResponseCompleteness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate 1-10 decoration billing records
		numRecords := rng.Intn(10) + 1
		for i := 0; i < numRecords; i++ {
			// Create a user with a unique display name
			displayName := fmt.Sprintf("User_%d_%d", seed, i)
			email := fmt.Sprintf("dbd-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user: %v", err)
				return false
			}
			userID, _ := result.LastInsertId()

			// Randomly create a storefront for some users
			if rng.Intn(2) == 0 {
				storeName := fmt.Sprintf("Store_%d_%d", seed, i)
				storeSlug := fmt.Sprintf("store-%d-%d", seed, i)
				_, err := db.Exec(
					"INSERT INTO author_storefronts (user_id, store_name, store_slug) VALUES (?, ?, ?)",
					userID, storeName, storeSlug,
				)
				if err != nil {
					t.Logf("FAIL: failed to create storefront: %v", err)
					return false
				}
			}

			// Insert a decoration transaction
			amount := -(float64(rng.Intn(500) + 1)) // negative amount for deduction
			description := fmt.Sprintf("装修费用 %d Credits", -int(amount))
			createdAt := time.Now().Add(-time.Duration(rng.Intn(86400)) * time.Second).Format("2006-01-02 15:04:05")
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, 'decoration', ?, ?, ?)",
				userID, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create transaction: %v", err)
				return false
			}
		}

		// Call the handler
		req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration", nil)
		rr := httptest.NewRecorder()
		handleDecorationBillingList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: expected 200, got %d", rr.Code)
			return false
		}

		// Parse response
		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("FAIL: failed to parse response: %v", err)
			return false
		}

		// Verify top-level fields exist
		for _, field := range []string{"total", "total_credits", "page", "page_size"} {
			if _, ok := resp[field]; !ok {
				t.Logf("FAIL: response missing top-level field %q", field)
				return false
			}
		}

		// Verify records array exists
		recordsRaw, ok := resp["records"]
		if !ok {
			t.Logf("FAIL: response missing 'records' field")
			return false
		}
		records, ok := recordsRaw.([]interface{})
		if !ok {
			t.Logf("FAIL: 'records' is not an array")
			return false
		}

		// Verify each record contains all required fields
		requiredFields := []string{"id", "user_id", "display_name", "store_name", "amount", "description", "created_at"}
		for idx, recRaw := range records {
			rec, ok := recRaw.(map[string]interface{})
			if !ok {
				t.Logf("FAIL: record %d is not an object", idx)
				return false
			}
			for _, field := range requiredFields {
				if _, ok := rec[field]; !ok {
					t.Logf("FAIL: record %d missing field %q", idx, field)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// Feature: decoration-billing-details, Property 2: 结果按创建时间降序排列
// **Validates: Requirements 1.2**
//
// For any set of decoration billing records, the records returned by
// GET /admin/api/billing/decoration are sorted by created_at in descending
// order — each record's created_at is greater than or equal to the next
// record's created_at.
func TestDecorationBillingDetailsProperty2_SortedByTimeDesc(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate 2-20 decoration billing records with varying timestamps
		numRecords := rng.Intn(19) + 2
		for i := 0; i < numRecords; i++ {
			displayName := fmt.Sprintf("User_%d_%d", seed, i)
			email := fmt.Sprintf("dbd2-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd2-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user: %v", err)
				return false
			}
			userID, _ := result.LastInsertId()

			// Randomly create a storefront for some users
			if rng.Intn(2) == 0 {
				storeName := fmt.Sprintf("Store_%d_%d", seed, i)
				storeSlug := fmt.Sprintf("store2-%d-%d", seed, i)
				_, err := db.Exec(
					"INSERT INTO author_storefronts (user_id, store_name, store_slug) VALUES (?, ?, ?)",
					userID, storeName, storeSlug,
				)
				if err != nil {
					t.Logf("FAIL: failed to create storefront: %v", err)
					return false
				}
			}

			// Insert a decoration transaction with a random created_at spread over 30 days
			amount := -(float64(rng.Intn(500) + 1))
			description := fmt.Sprintf("装修费用 %d Credits", -int(amount))
			// Spread timestamps across 30 days to ensure meaningful ordering variation
			offsetSeconds := rng.Intn(30 * 24 * 3600)
			createdAt := time.Now().Add(-time.Duration(offsetSeconds) * time.Second).Format("2006-01-02 15:04:05")
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, 'decoration', ?, ?, ?)",
				userID, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create transaction: %v", err)
				return false
			}
		}

		// Call the handler
		req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration", nil)
		rr := httptest.NewRecorder()
		handleDecorationBillingList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: expected 200, got %d", rr.Code)
			return false
		}

		// Parse response
		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("FAIL: failed to parse response: %v", err)
			return false
		}

		recordsRaw, ok := resp["records"]
		if !ok {
			t.Logf("FAIL: response missing 'records' field")
			return false
		}
		records, ok := recordsRaw.([]interface{})
		if !ok {
			t.Logf("FAIL: 'records' is not an array")
			return false
		}

		if len(records) < 2 {
			// Not enough records to verify ordering
			return true
		}

		// Verify each record's created_at >= next record's created_at (descending order)
		for i := 0; i < len(records)-1; i++ {
			curr, ok1 := records[i].(map[string]interface{})
			next, ok2 := records[i+1].(map[string]interface{})
			if !ok1 || !ok2 {
				t.Logf("FAIL: record %d or %d is not an object", i, i+1)
				return false
			}

			currTime, ok1 := curr["created_at"].(string)
			nextTime, ok2 := next["created_at"].(string)
			if !ok1 || !ok2 {
				t.Logf("FAIL: record %d or %d missing created_at string", i, i+1)
				return false
			}

			if currTime < nextTime {
				t.Logf("FAIL: records not sorted DESC — record[%d].created_at=%q < record[%d].created_at=%q",
					i, currTime, i+1, nextTime)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// Feature: decoration-billing-details, Property 3: 总数和总扣费金额准确
// **Validates: Requirements 1.4, 3.4**
//
// For any set of decoration billing records and any search condition,
// total equals the number of matching records and total_credits equals
// the sum of ABS(amount) of matching records.
func TestDecorationBillingDetailsProperty3_TotalAndCreditsAccuracy(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate 1-15 decoration billing records with known amounts
		numRecords := rng.Intn(15) + 1

		type testRecord struct {
			displayName string
			storeName   string
			amount      float64
		}
		var allRecords []testRecord

		for i := 0; i < numRecords; i++ {
			displayName := fmt.Sprintf("User_%d_%d", seed, i)
			email := fmt.Sprintf("dbd3-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd3-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user: %v", err)
				return false
			}
			userID, _ := result.LastInsertId()

			// Randomly create a storefront for some users
			storeName := ""
			if rng.Intn(2) == 0 {
				storeName = fmt.Sprintf("Store_%d_%d", seed, i)
				storeSlug := fmt.Sprintf("store3-%d-%d", seed, i)
				_, err := db.Exec(
					"INSERT INTO author_storefronts (user_id, store_name, store_slug) VALUES (?, ?, ?)",
					userID, storeName, storeSlug,
				)
				if err != nil {
					t.Logf("FAIL: failed to create storefront: %v", err)
					return false
				}
			}

			// Insert a decoration transaction with a known negative amount
			amount := -(float64(rng.Intn(500) + 1))
			description := fmt.Sprintf("装修费用 %d Credits", -int(amount))
			createdAt := time.Now().Add(-time.Duration(rng.Intn(30*24*3600)) * time.Second).Format("2006-01-02 15:04:05")
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, 'decoration', ?, ?, ?)",
				userID, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create transaction: %v", err)
				return false
			}

			allRecords = append(allRecords, testRecord{
				displayName: displayName,
				storeName:   storeName,
				amount:      amount,
			})
		}

		// --- Test 1: Without search parameter ---
		{
			req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration", nil)
			rr := httptest.NewRecorder()
			handleDecorationBillingList(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("FAIL: expected 200, got %d", rr.Code)
				return false
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response: %v", err)
				return false
			}

			// Verify total equals the number of records
			total, ok := resp["total"].(float64)
			if !ok {
				t.Logf("FAIL: total is not a number")
				return false
			}
			if int(total) != numRecords {
				t.Logf("FAIL: total=%d, expected=%d", int(total), numRecords)
				return false
			}

			// Compute expected total_credits = sum of ABS(amount)
			var expectedCredits float64
			for _, rec := range allRecords {
				expectedCredits += math.Abs(rec.amount)
			}

			totalCredits, ok := resp["total_credits"].(float64)
			if !ok {
				t.Logf("FAIL: total_credits is not a number")
				return false
			}
			if math.Abs(totalCredits-expectedCredits) > 0.01 {
				t.Logf("FAIL: total_credits=%.2f, expected=%.2f", totalCredits, expectedCredits)
				return false
			}
		}

		// --- Test 2: With search parameter ---
		{
			// Pick a search term that matches a subset of records
			// Use the index of a random record to build a search term
			targetIdx := rng.Intn(numRecords)
			searchTerm := fmt.Sprintf("_%d_%d", seed, targetIdx)

			// Compute expected filtered results
			var expectedTotal int
			var expectedCredits float64
			for _, rec := range allRecords {
				if strings.Contains(rec.displayName, searchTerm) || strings.Contains(rec.storeName, searchTerm) {
					expectedTotal++
					expectedCredits += math.Abs(rec.amount)
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration?search="+searchTerm, nil)
			rr := httptest.NewRecorder()
			handleDecorationBillingList(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("FAIL: expected 200 with search, got %d", rr.Code)
				return false
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse search response: %v", err)
				return false
			}

			total, ok := resp["total"].(float64)
			if !ok {
				t.Logf("FAIL: total is not a number (search)")
				return false
			}
			if int(total) != expectedTotal {
				t.Logf("FAIL: search total=%d, expected=%d (search=%q)", int(total), expectedTotal, searchTerm)
				return false
			}

			totalCredits, ok := resp["total_credits"].(float64)
			if !ok {
				t.Logf("FAIL: total_credits is not a number (search)")
				return false
			}
			if math.Abs(totalCredits-expectedCredits) > 0.01 {
				t.Logf("FAIL: search total_credits=%.2f, expected=%.2f (search=%q)", totalCredits, expectedCredits, searchTerm)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 violated: %v", err)
	}
}

// Feature: decoration-billing-details, Property 4: 搜索过滤正确性
// **Validates: Requirements 3.1, 3.4**
//
// For any search term, every record returned by
// GET /admin/api/billing/decoration?search=xxx has a display_name or
// store_name that contains the search term (case-insensitive LIKE matching),
// and total reflects the filtered record count.
func TestDecorationBillingDetailsProperty4_SearchFilterCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate 3-15 decoration billing records with varied names
		numRecords := rng.Intn(13) + 3

		type testRecord struct {
			displayName string
			storeName   string
		}
		var allRecords []testRecord

		// Use a set of name fragments so some records share substrings
		nameFragments := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}

		for i := 0; i < numRecords; i++ {
			fragment := nameFragments[rng.Intn(len(nameFragments))]
			displayName := fmt.Sprintf("%s_User_%d_%d", fragment, seed, i)
			email := fmt.Sprintf("dbd4-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd4-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user: %v", err)
				return false
			}
			userID, _ := result.LastInsertId()

			// Randomly create a storefront with a different fragment
			storeName := ""
			if rng.Intn(2) == 0 {
				storeFragment := nameFragments[rng.Intn(len(nameFragments))]
				storeName = fmt.Sprintf("%s_Store_%d_%d", storeFragment, seed, i)
				storeSlug := fmt.Sprintf("store4-%d-%d", seed, i)
				_, err := db.Exec(
					"INSERT INTO author_storefronts (user_id, store_name, store_slug) VALUES (?, ?, ?)",
					userID, storeName, storeSlug,
				)
				if err != nil {
					t.Logf("FAIL: failed to create storefront: %v", err)
					return false
				}
			}

			// Insert a decoration transaction
			amount := -(float64(rng.Intn(500) + 1))
			description := fmt.Sprintf("装修费用 %d Credits", -int(amount))
			createdAt := time.Now().Add(-time.Duration(rng.Intn(30*24*3600)) * time.Second).Format("2006-01-02 15:04:05")
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, 'decoration', ?, ?, ?)",
				userID, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create transaction: %v", err)
				return false
			}

			allRecords = append(allRecords, testRecord{
				displayName: displayName,
				storeName:   storeName,
			})
		}

		// Pick a random search term from the name fragments
		searchTerm := nameFragments[rng.Intn(len(nameFragments))]

		// Compute expected filtered count locally using case-insensitive contains
		searchLower := strings.ToLower(searchTerm)
		var expectedTotal int
		for _, rec := range allRecords {
			if strings.Contains(strings.ToLower(rec.displayName), searchLower) ||
				strings.Contains(strings.ToLower(rec.storeName), searchLower) {
				expectedTotal++
			}
		}

		// Call the handler with search parameter
		req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration?search="+searchTerm, nil)
		rr := httptest.NewRecorder()
		handleDecorationBillingList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: expected 200, got %d", rr.Code)
			return false
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("FAIL: failed to parse response: %v", err)
			return false
		}

		// Verify total reflects the filtered count
		total, ok := resp["total"].(float64)
		if !ok {
			t.Logf("FAIL: total is not a number")
			return false
		}
		if int(total) != expectedTotal {
			t.Logf("FAIL: total=%d, expected=%d (search=%q)", int(total), expectedTotal, searchTerm)
			return false
		}

		// Verify every returned record's display_name or store_name contains the search term
		recordsRaw, ok := resp["records"]
		if !ok {
			t.Logf("FAIL: response missing 'records' field")
			return false
		}
		records, ok := recordsRaw.([]interface{})
		if !ok {
			t.Logf("FAIL: 'records' is not an array")
			return false
		}

		for idx, recRaw := range records {
			rec, ok := recRaw.(map[string]interface{})
			if !ok {
				t.Logf("FAIL: record %d is not an object", idx)
				return false
			}

			displayName, _ := rec["display_name"].(string)
			storeName, _ := rec["store_name"].(string)

			displayNameLower := strings.ToLower(displayName)
			storeNameLower := strings.ToLower(storeName)

			if !strings.Contains(displayNameLower, searchLower) && !strings.Contains(storeNameLower, searchLower) {
				t.Logf("FAIL: record %d display_name=%q store_name=%q does not contain search=%q",
					idx, displayName, storeName, searchTerm)
				return false
			}
		}

		// Verify the number of returned records matches total (within page size)
		if len(records) != int(total) && len(records) != 50 {
			t.Logf("FAIL: records count=%d does not match total=%d (and is not page_size=50)", len(records), int(total))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 violated: %v", err)
	}
}

// Feature: decoration-billing-details, Property 5: 空搜索返回全部记录
// **Validates: Requirements 3.3**
//
// For any set of decoration billing records, when the search parameter is
// empty or not provided, total equals the count of all decoration-type
// transaction records.
func TestDecorationBillingDetailsProperty5_EmptySearchReturnsAll(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate 1-15 decoration billing records
		numDecorationRecords := rng.Intn(15) + 1
		for i := 0; i < numDecorationRecords; i++ {
			displayName := fmt.Sprintf("User_%d_%d", seed, i)
			email := fmt.Sprintf("dbd5-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd5-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user: %v", err)
				return false
			}
			userID, _ := result.LastInsertId()

			// Randomly create a storefront for some users
			if rng.Intn(2) == 0 {
				storeName := fmt.Sprintf("Store_%d_%d", seed, i)
				storeSlug := fmt.Sprintf("store5-%d-%d", seed, i)
				_, err := db.Exec(
					"INSERT INTO author_storefronts (user_id, store_name, store_slug) VALUES (?, ?, ?)",
					userID, storeName, storeSlug,
				)
				if err != nil {
					t.Logf("FAIL: failed to create storefront: %v", err)
					return false
				}
			}

			// Insert a decoration transaction
			amount := -(float64(rng.Intn(500) + 1))
			description := fmt.Sprintf("装修费用 %d Credits", -int(amount))
			createdAt := time.Now().Add(-time.Duration(rng.Intn(30*24*3600)) * time.Second).Format("2006-01-02 15:04:05")
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, 'decoration', ?, ?, ?)",
				userID, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create decoration transaction: %v", err)
				return false
			}
		}

		// Also create some non-decoration transactions to ensure they are excluded
		numOtherRecords := rng.Intn(10) + 1
		for i := 0; i < numOtherRecords; i++ {
			displayName := fmt.Sprintf("OtherUser_%d_%d", seed, i)
			email := fmt.Sprintf("dbd5-other-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd5-other-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create other user: %v", err)
				return false
			}
			otherUserID, _ := result.LastInsertId()

			// Insert a non-decoration transaction (e.g., 'purchase', 'topup')
			otherTypes := []string{"purchase", "topup", "refund", "listing"}
			txType := otherTypes[rng.Intn(len(otherTypes))]
			amount := -(float64(rng.Intn(200) + 1))
			description := fmt.Sprintf("其他交易 %d Credits", -int(amount))
			createdAt := time.Now().Add(-time.Duration(rng.Intn(30*24*3600)) * time.Second).Format("2006-01-02 15:04:05")
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, ?, ?, ?, ?)",
				otherUserID, txType, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create other transaction: %v", err)
				return false
			}
		}

		// --- Test 1: Without search parameter ---
		{
			req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration", nil)
			rr := httptest.NewRecorder()
			handleDecorationBillingList(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("FAIL: expected 200, got %d (no search param)", rr.Code)
				return false
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response (no search param): %v", err)
				return false
			}

			total, ok := resp["total"].(float64)
			if !ok {
				t.Logf("FAIL: total is not a number (no search param)")
				return false
			}
			if int(total) != numDecorationRecords {
				t.Logf("FAIL: total=%d, expected=%d (no search param)", int(total), numDecorationRecords)
				return false
			}
		}

		// --- Test 2: With empty search parameter (search=) ---
		{
			req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration?search=", nil)
			rr := httptest.NewRecorder()
			handleDecorationBillingList(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("FAIL: expected 200, got %d (empty search)", rr.Code)
				return false
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response (empty search): %v", err)
				return false
			}

			total, ok := resp["total"].(float64)
			if !ok {
				t.Logf("FAIL: total is not a number (empty search)")
				return false
			}
			if int(total) != numDecorationRecords {
				t.Logf("FAIL: total=%d, expected=%d (empty search)", int(total), numDecorationRecords)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 violated: %v", err)
	}
}

// Feature: decoration-billing-details, Property 7: 分页正确性
// **Validates: Requirements 2.1, 2.2**
//
// For any set of decoration billing records with count > 50, requesting
// different page numbers yields non-overlapping records per page, and
// merging all pages produces the complete result set.
func TestDecorationBillingDetailsProperty7_PaginationCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 20,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate 51-80 decoration billing records to ensure pagination kicks in (pageSize=50)
		numRecords := rng.Intn(30) + 51

		// Track all inserted transaction IDs
		var allTransactionIDs []float64

		for i := 0; i < numRecords; i++ {
			displayName := fmt.Sprintf("User_%d_%d", seed, i)
			email := fmt.Sprintf("dbd7-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd7-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user: %v", err)
				return false
			}
			userID, _ := result.LastInsertId()

			// Randomly create a storefront for some users
			if rng.Intn(2) == 0 {
				storeName := fmt.Sprintf("Store_%d_%d", seed, i)
				storeSlug := fmt.Sprintf("store7-%d-%d", seed, i)
				_, err := db.Exec(
					"INSERT INTO author_storefronts (user_id, store_name, store_slug) VALUES (?, ?, ?)",
					userID, storeName, storeSlug,
				)
				if err != nil {
					t.Logf("FAIL: failed to create storefront: %v", err)
					return false
				}
			}

			// Insert a decoration transaction with spread timestamps
			amount := -(float64(rng.Intn(500) + 1))
			description := fmt.Sprintf("装修费用 %d Credits", -int(amount))
			offsetSeconds := rng.Intn(30 * 24 * 3600)
			createdAt := time.Now().Add(-time.Duration(offsetSeconds) * time.Second).Format("2006-01-02 15:04:05")
			txResult, err := db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, 'decoration', ?, ?, ?)",
				userID, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create transaction: %v", err)
				return false
			}
			txID, _ := txResult.LastInsertId()
			allTransactionIDs = append(allTransactionIDs, float64(txID))
		}

		// Request page 1
		req1 := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration?page=1", nil)
		rr1 := httptest.NewRecorder()
		handleDecorationBillingList(rr1, req1)

		if rr1.Code != http.StatusOK {
			t.Logf("FAIL: page 1 expected 200, got %d", rr1.Code)
			return false
		}

		var resp1 map[string]interface{}
		if err := json.Unmarshal(rr1.Body.Bytes(), &resp1); err != nil {
			t.Logf("FAIL: failed to parse page 1 response: %v", err)
			return false
		}

		// Verify total matches the number of records we inserted
		total, ok := resp1["total"].(float64)
		if !ok {
			t.Logf("FAIL: total is not a number")
			return false
		}
		if int(total) != numRecords {
			t.Logf("FAIL: total=%d, expected=%d", int(total), numRecords)
			return false
		}

		records1Raw, ok := resp1["records"].([]interface{})
		if !ok {
			t.Logf("FAIL: page 1 'records' is not an array")
			return false
		}

		// Page 1 should have exactly 50 records (pageSize=50)
		if len(records1Raw) != 50 {
			t.Logf("FAIL: page 1 records count=%d, expected=50", len(records1Raw))
			return false
		}

		// Request page 2
		req2 := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration?page=2", nil)
		rr2 := httptest.NewRecorder()
		handleDecorationBillingList(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Logf("FAIL: page 2 expected 200, got %d", rr2.Code)
			return false
		}

		var resp2 map[string]interface{}
		if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil {
			t.Logf("FAIL: failed to parse page 2 response: %v", err)
			return false
		}

		records2Raw, ok := resp2["records"].([]interface{})
		if !ok {
			t.Logf("FAIL: page 2 'records' is not an array")
			return false
		}

		// Page 2 should have the remaining records (numRecords - 50)
		expectedPage2Count := numRecords - 50
		if len(records2Raw) != expectedPage2Count {
			t.Logf("FAIL: page 2 records count=%d, expected=%d", len(records2Raw), expectedPage2Count)
			return false
		}

		// Collect IDs from page 1
		page1IDs := make(map[float64]bool)
		for idx, recRaw := range records1Raw {
			rec, ok := recRaw.(map[string]interface{})
			if !ok {
				t.Logf("FAIL: page 1 record %d is not an object", idx)
				return false
			}
			id, ok := rec["id"].(float64)
			if !ok {
				t.Logf("FAIL: page 1 record %d missing 'id'", idx)
				return false
			}
			page1IDs[id] = true
		}

		// Collect IDs from page 2 and verify no overlap with page 1
		page2IDs := make(map[float64]bool)
		for idx, recRaw := range records2Raw {
			rec, ok := recRaw.(map[string]interface{})
			if !ok {
				t.Logf("FAIL: page 2 record %d is not an object", idx)
				return false
			}
			id, ok := rec["id"].(float64)
			if !ok {
				t.Logf("FAIL: page 2 record %d missing 'id'", idx)
				return false
			}
			if page1IDs[id] {
				t.Logf("FAIL: record id=%.0f appears on both page 1 and page 2", id)
				return false
			}
			page2IDs[id] = true
		}

		// Verify that merging all pages gives the complete result set
		allPageIDs := make(map[float64]bool)
		for id := range page1IDs {
			allPageIDs[id] = true
		}
		for id := range page2IDs {
			allPageIDs[id] = true
		}

		if len(allPageIDs) != numRecords {
			t.Logf("FAIL: merged page IDs count=%d, expected=%d", len(allPageIDs), numRecords)
			return false
		}

		// Verify every inserted transaction ID appears in the merged set
		for _, txID := range allTransactionIDs {
			if !allPageIDs[txID] {
				t.Logf("FAIL: transaction id=%.0f not found in any page", txID)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 violated: %v", err)
	}
}

// Feature: decoration-billing-details, Property 6: 导出包含所有过滤记录
// **Validates: Requirements 4.1, 4.3**
//
// For any search condition and any set of decoration billing records,
// the exported Excel file contains a number of data rows equal to the
// total filtered record count (not limited by pagination), and every
// row's username or store name contains the search term.
func TestDecorationBillingDetailsProperty6_ExportContainsAllFiltered(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate 3-20 decoration billing records with varied names
		numRecords := rng.Intn(18) + 3

		type testRecord struct {
			displayName string
			storeName   string
		}
		var allRecords []testRecord

		nameFragments := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}

		for i := 0; i < numRecords; i++ {
			fragment := nameFragments[rng.Intn(len(nameFragments))]
			displayName := fmt.Sprintf("%s_User_%d_%d", fragment, seed, i)
			email := fmt.Sprintf("dbd6-%d-%d@test.com", seed, i)
			authID := fmt.Sprintf("dbd6-auth-%d-%d", seed, i)
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 1000)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user: %v", err)
				return false
			}
			userID, _ := result.LastInsertId()

			// Randomly create a storefront with a different fragment
			storeName := ""
			if rng.Intn(2) == 0 {
				storeFragment := nameFragments[rng.Intn(len(nameFragments))]
				storeName = fmt.Sprintf("%s_Store_%d_%d", storeFragment, seed, i)
				storeSlug := fmt.Sprintf("store6-%d-%d", seed, i)
				_, err := db.Exec(
					"INSERT INTO author_storefronts (user_id, store_name, store_slug) VALUES (?, ?, ?)",
					userID, storeName, storeSlug,
				)
				if err != nil {
					t.Logf("FAIL: failed to create storefront: %v", err)
					return false
				}
			}

			// Insert a decoration transaction
			amount := -(float64(rng.Intn(500) + 1))
			description := fmt.Sprintf("装修费用 %d Credits", -int(amount))
			createdAt := time.Now().Add(-time.Duration(rng.Intn(30*24*3600)) * time.Second).Format("2006-01-02 15:04:05")
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at) VALUES (?, 'decoration', ?, ?, ?)",
				userID, amount, description, createdAt,
			)
			if err != nil {
				t.Logf("FAIL: failed to create transaction: %v", err)
				return false
			}

			allRecords = append(allRecords, testRecord{
				displayName: displayName,
				storeName:   storeName,
			})
		}

		// Pick a random search term from the name fragments
		searchTerm := nameFragments[rng.Intn(len(nameFragments))]

		// Compute expected filtered count locally using case-insensitive contains
		searchLower := strings.ToLower(searchTerm)
		var expectedTotal int
		for _, rec := range allRecords {
			if strings.Contains(strings.ToLower(rec.displayName), searchLower) ||
				strings.Contains(strings.ToLower(rec.storeName), searchLower) {
				expectedTotal++
			}
		}

		// Call the export handler with search parameter
		req := httptest.NewRequest(http.MethodGet, "/admin/api/billing/decoration/export?search="+searchTerm, nil)
		rr := httptest.NewRecorder()
		handleDecorationBillingExport(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: expected 200, got %d", rr.Code)
			return false
		}

		// Parse the Excel response body
		xlFile, err := excelize.OpenReader(bytes.NewReader(rr.Body.Bytes()))
		if err != nil {
			t.Logf("FAIL: failed to open Excel from response: %v", err)
			return false
		}
		defer xlFile.Close()

		sheetName := "装修计费明细"
		rows, err := xlFile.GetRows(sheetName)
		if err != nil {
			t.Logf("FAIL: failed to get rows from sheet %q: %v", sheetName, err)
			return false
		}

		// rows[0] is the header row; data rows start from rows[1]
		dataRowCount := 0
		if len(rows) > 0 {
			dataRowCount = len(rows) - 1
		}

		// Verify the number of data rows equals the expected filtered count
		if dataRowCount != expectedTotal {
			t.Logf("FAIL: Excel data rows=%d, expected=%d (search=%q)", dataRowCount, expectedTotal, searchTerm)
			return false
		}

		// Verify each data row's username (col index 1) or store name (col index 2) contains the search term
		// Excel columns: 交易 ID(0), 用户名(1), 店铺名(2), 扣费金额(3), 交易描述(4), 创建时间(5)
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			userName := ""
			storeNameVal := ""
			if len(row) > 1 {
				userName = row[1]
			}
			if len(row) > 2 {
				storeNameVal = row[2]
			}

			userNameLower := strings.ToLower(userName)
			storeNameLower := strings.ToLower(storeNameVal)

			if !strings.Contains(userNameLower, searchLower) && !strings.Contains(storeNameLower, searchLower) {
				t.Logf("FAIL: Excel row %d userName=%q storeName=%q does not contain search=%q",
					i, userName, storeNameVal, searchTerm)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 violated: %v", err)
	}
}
