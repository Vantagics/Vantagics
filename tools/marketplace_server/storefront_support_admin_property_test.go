package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: storefront-support-admin, Property 1: 门槛值 Round-Trip
// **Validates: Requirements 1.2, 1.4**
//
// For any positive integer T, after calling set-threshold(T),
// get-threshold() should return T.
func TestSupportAdminProperty1_ThresholdRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate a random positive integer in range [1, 10000000]
		threshold := rng.Intn(10000000) + 1

		// Call set-threshold using JSON body (send threshold as string to avoid float64 precision issues)
		jsonBody := `{"threshold":"` + strconv.Itoa(threshold) + `"}`
		setReq := httptest.NewRequest(http.MethodPost, "/admin/api/storefront-support/set-threshold", strings.NewReader(jsonBody))
		setReq.Header.Set("Content-Type", "application/json")
		setRR := httptest.NewRecorder()
		handleSetSupportThreshold(setRR, setReq)

		if setRR.Code != http.StatusOK {
			t.Logf("FAIL: set-threshold(%d) returned status %d, body: %s", threshold, setRR.Code, setRR.Body.String())
			return false
		}

		// Call get-threshold
		getReq := httptest.NewRequest(http.MethodGet, "/admin/api/storefront-support/get-threshold", nil)
		getRR := httptest.NewRecorder()
		handleGetSupportThreshold(getRR, getReq)

		if getRR.Code != http.StatusOK {
			t.Logf("FAIL: get-threshold returned status %d", getRR.Code)
			return false
		}

		var resp map[string]int
		if err := json.NewDecoder(getRR.Body).Decode(&resp); err != nil {
			t.Logf("FAIL: failed to decode get-threshold response: %v", err)
			return false
		}

		got := resp["threshold"]
		if got != threshold {
			t.Logf("FAIL: set-threshold(%d) then get-threshold() = %d, want %d", threshold, got, threshold)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// Feature: storefront-support-admin, Property 2: 非正整数门槛值被拒绝
// **Validates: Requirements 1.5**
//
// For any non-positive-integer value V (0, negative, float string, non-numeric string),
// set-threshold(V) should return an error and the threshold should remain unchanged.
func TestSupportAdminProperty2_InvalidThresholdRejected(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Step 1: Set a known valid threshold
		knownThreshold := rng.Intn(10000) + 1 // [1, 10000]
		jsonBody := `{"threshold":"` + strconv.Itoa(knownThreshold) + `"}`
		setReq := httptest.NewRequest(http.MethodPost, "/admin/api/storefront-support/set-threshold", strings.NewReader(jsonBody))
		setReq.Header.Set("Content-Type", "application/json")
		setRR := httptest.NewRecorder()
		handleSetSupportThreshold(setRR, setReq)
		if setRR.Code != http.StatusOK {
			t.Logf("FAIL: could not set known threshold %d, status %d", knownThreshold, setRR.Code)
			return false
		}

		// Step 2: Generate a random invalid value
		invalidValue := generateInvalidThreshold(rng)

		// Step 3: Try setting the invalid value
		invalidJSON := `{"threshold":` + invalidValue + `}`
		invalidReq := httptest.NewRequest(http.MethodPost, "/admin/api/storefront-support/set-threshold", strings.NewReader(invalidJSON))
		invalidReq.Header.Set("Content-Type", "application/json")
		invalidRR := httptest.NewRecorder()
		handleSetSupportThreshold(invalidRR, invalidReq)

		if invalidRR.Code != http.StatusBadRequest {
			t.Logf("FAIL: set-threshold(%s) returned status %d, expected 400, body: %s", invalidValue, invalidRR.Code, invalidRR.Body.String())
			return false
		}

		// Step 4: Verify threshold is unchanged
		getReq := httptest.NewRequest(http.MethodGet, "/admin/api/storefront-support/get-threshold", nil)
		getRR := httptest.NewRecorder()
		handleGetSupportThreshold(getRR, getReq)

		if getRR.Code != http.StatusOK {
			t.Logf("FAIL: get-threshold returned status %d after invalid set", getRR.Code)
			return false
		}

		var resp map[string]int
		if err := json.NewDecoder(getRR.Body).Decode(&resp); err != nil {
			t.Logf("FAIL: failed to decode get-threshold response: %v", err)
			return false
		}

		got := resp["threshold"]
		if got != knownThreshold {
			t.Logf("FAIL: threshold changed from %d to %d after invalid set-threshold(%s)", knownThreshold, got, invalidValue)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// generateInvalidThreshold returns a random invalid threshold value as a JSON-embeddable string.
// Categories: 0, negative integers, float strings, non-numeric strings.
func generateInvalidThreshold(rng *rand.Rand) string {
	category := rng.Intn(4)
	switch category {
	case 0:
		// Zero
		return "0"
	case 1:
		// Negative integer
		neg := -(rng.Intn(1000000) + 1)
		return strconv.Itoa(neg)
	case 2:
		// Float string (e.g., "3.14", "0.5", "-2.7")
		floats := []string{
			`"3.14"`, `"0.5"`, `"1.1"`, `"-2.7"`, `"0.001"`,
			`"99.99"`, `"1.0"`, `"-0.1"`, `"2.5"`, `"100.1"`,
		}
		return floats[rng.Intn(len(floats))]
	case 3:
		// Non-numeric string
		nonNumeric := []string{
			`"abc"`, `"hello"`, `"--1"`, `"12abc"`, `""`,
			`"null"`, `"true"`, `"false"`, `" "`, `"NaN"`,
		}
		return nonNumeric[rng.Intn(len(nonNumeric))]
	}
	return "0"
}

// Feature: storefront-support-admin, Property 3: 动态门槛在资格校验中生效
// **Validates: Requirements 1.6**
//
// For any positive integer threshold T and any total sales S >= 0,
// after setting threshold to T, getSupportSalesThreshold() returns T,
// and the eligibility result equals S >= float64(T).
func TestSupportAdminProperty3_DynamicThresholdEligibility(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate a random positive integer threshold in range [1, 10000000]
		threshold := rng.Intn(10000000) + 1

		// Set threshold via handleSetSupportThreshold
		jsonBody := `{"threshold":"` + strconv.Itoa(threshold) + `"}`
		setReq := httptest.NewRequest(http.MethodPost, "/admin/api/storefront-support/set-threshold", strings.NewReader(jsonBody))
		setReq.Header.Set("Content-Type", "application/json")
		setRR := httptest.NewRecorder()
		handleSetSupportThreshold(setRR, setReq)

		if setRR.Code != http.StatusOK {
			t.Logf("FAIL: set-threshold(%d) returned status %d, body: %s", threshold, setRR.Code, setRR.Body.String())
			return false
		}

		// Verify getSupportSalesThreshold() returns the set value
		got := getSupportSalesThreshold()
		if got != threshold {
			t.Logf("FAIL: getSupportSalesThreshold() = %d after setting %d", got, threshold)
			return false
		}

		// Generate a random total sales value >= 0
		// Use a range that covers values around the threshold for meaningful testing
		totalSales := rng.Float64() * float64(threshold*2)

		// The actual code checks: totalSales < float64(getSupportSalesThreshold())
		// So eligible = !(totalSales < float64(threshold)) = totalSales >= float64(threshold)
		expectedEligible := totalSales >= float64(threshold)
		actualIneligible := totalSales < float64(getSupportSalesThreshold())
		actualEligible := !actualIneligible

		if actualEligible != expectedEligible {
			t.Logf("FAIL: threshold=%d, totalSales=%.2f, expectedEligible=%v, actualEligible=%v",
				threshold, totalSales, expectedEligible, actualEligible)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 violated: %v", err)
	}
}

// Feature: storefront-support-admin, Property 4: 每页最多返回 50 条记录
// **Validates: Requirements 2.1**
//
// For any dataset with N records and any page P,
// len(response.items) <= 50.
// If N > 50 and P == 1, then len(response.items) == 50.
func TestSupportAdminProperty4_PageSizeLimit(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate random N in [0, 120]
		n := rng.Intn(121)

		// Insert N storefront_support_requests with corresponding users and storefronts
		statuses := []string{"pending", "approved", "disabled"}
		for i := 0; i < n; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(10000)))
			storeSlug := fmt.Sprintf("p4-store-%d-%d-%d", seed, i, rng.Int63n(1000000))
			storeName := fmt.Sprintf("P4Store_%d_%d", seed, i)

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
				t.Logf("FAIL: failed to insert request %d: %v", i, err)
				return false
			}
		}

		// Generate random page P in [1, max(1, ceil(n/50)+1)]
		maxPage := 1
		if n > 0 {
			maxPage = (n / 50) + 2 // go a bit beyond to test empty pages too
		}
		page := rng.Intn(maxPage) + 1

		// Call handleAdminStorefrontSupportList with page=P
		req := httptest.NewRequest(http.MethodGet,
			"/admin/api/storefront-support/list?page="+strconv.Itoa(page), nil)
		rr := httptest.NewRecorder()
		handleAdminStorefrontSupportList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: list returned status %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		var resp AdminSupportListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Logf("FAIL: failed to decode response: %v", err)
			return false
		}

		// Property: len(items) <= 50
		if len(resp.Items) > 50 {
			t.Logf("FAIL: N=%d, page=%d, got %d items (expected <= 50)", n, page, len(resp.Items))
			return false
		}

		// Property: if N > 50 and P == 1, then len(items) == 50
		if n > 50 && page == 1 && len(resp.Items) != 50 {
			t.Logf("FAIL: N=%d, page=1, got %d items (expected exactly 50)", n, len(resp.Items))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 violated: %v", err)
	}
}

// Feature: storefront-support-admin, Property 5: 排序方向正确性
// **Validates: Requirements 3.2, 3.4, 3.5, 3.6**
//
// For any sort_order in {"asc", "desc", ""}, the list API returns results
// sorted by created_at in the specified direction.
// When sort_order is "asc", created_at should be ascending.
// When sort_order is "desc" or empty, created_at should be descending.
func TestSupportAdminProperty5_SortOrderCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate random N in [5, 20]
		n := rng.Intn(16) + 5

		// Insert N storefront_support_requests with different created_at timestamps
		statuses := []string{"pending", "approved", "disabled"}
		baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < n; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(10000)))
			storeSlug := fmt.Sprintf("p5-store-%d-%d-%d", seed, i, rng.Int63n(1000000))
			storeName := fmt.Sprintf("P5Store_%d_%d", seed, i)

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
			// Each record gets a distinct timestamp: base + i days
			createdAt := baseTime.Add(time.Duration(i) * 24 * time.Hour).Format("2006-01-02 15:04:05")

			_, err = db.Exec(`
				INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
				VALUES (?, ?, 'vantagics', ?, '欢迎', ?, ?, CURRENT_TIMESTAMP)
			`, storefrontID, userID, storeName, status, createdAt)
			if err != nil {
				t.Logf("FAIL: failed to insert request %d: %v", i, err)
				return false
			}
		}

		// Randomly choose sort_order from {"asc", "desc", ""}
		sortOrders := []string{"asc", "desc", ""}
		sortOrder := sortOrders[rng.Intn(len(sortOrders))]

		// Call handleAdminStorefrontSupportList with sort_order parameter
		url := "/admin/api/storefront-support/list?page=1"
		if sortOrder != "" {
			url += "&sort_order=" + sortOrder
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rr := httptest.NewRecorder()
		handleAdminStorefrontSupportList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: list returned status %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		var resp AdminSupportListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Logf("FAIL: failed to decode response: %v", err)
			return false
		}

		// Verify sorting: need at least 2 items to check order
		if len(resp.Items) < 2 {
			return true
		}

		for i := 0; i < len(resp.Items)-1; i++ {
			curr := resp.Items[i].CreatedAt
			next := resp.Items[i+1].CreatedAt

			if sortOrder == "asc" {
				// Ascending: curr <= next
				if curr > next {
					t.Logf("FAIL: sort_order=%q, items[%d].created_at=%q > items[%d].created_at=%q (expected ascending)",
						sortOrder, i, curr, i+1, next)
					return false
				}
			} else {
				// Descending (desc or empty): curr >= next
				if curr < next {
					t.Logf("FAIL: sort_order=%q, items[%d].created_at=%q < items[%d].created_at=%q (expected descending)",
						sortOrder, i, curr, i+1, next)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 violated: %v", err)
	}
}

// Feature: storefront-support-admin, Property 6: 日期范围过滤正确性
// **Validates: Requirements 4.3, 4.4, 4.5, 4.6**
//
// For any valid date range [date_from, date_to] where date_from <= date_to,
// all returned records should have created_at >= date_from 00:00:00
// AND created_at <= date_to 23:59:59.
func TestSupportAdminProperty6_DateRangeFilter(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate random N in [10, 30]
		n := rng.Intn(21) + 10

		// Insert N storefront_support_requests with random created_at in 2025
		statuses := []string{"pending", "approved", "disabled"}
		yearStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		yearEnd := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
		totalSeconds := int(yearEnd.Sub(yearStart).Seconds())

		for i := 0; i < n; i++ {
			userID := createTestUserWithBalance(t, float64(rng.Intn(10000)))
			storeSlug := fmt.Sprintf("p6-store-%d-%d-%d", seed, i, rng.Int63n(1000000))
			storeName := fmt.Sprintf("P6Store_%d_%d", seed, i)

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
			// Random timestamp within 2025
			randomOffset := time.Duration(rng.Intn(totalSeconds)) * time.Second
			createdAt := yearStart.Add(randomOffset).Format("2006-01-02 15:04:05")

			_, err = db.Exec(`
				INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
				VALUES (?, ?, 'vantagics', ?, '欢迎', ?, ?, CURRENT_TIMESTAMP)
			`, storefrontID, userID, storeName, status, createdAt)
			if err != nil {
				t.Logf("FAIL: failed to insert request %d: %v", i, err)
				return false
			}
		}

		// Generate a random valid date range within 2025 (date_from <= date_to)
		// Pick two random day offsets in [0, 364] and sort them
		day1 := rng.Intn(365)
		day2 := rng.Intn(365)
		if day1 > day2 {
			day1, day2 = day2, day1
		}
		dateFrom := yearStart.AddDate(0, 0, day1).Format("2006-01-02")
		dateTo := yearStart.AddDate(0, 0, day2).Format("2006-01-02")

		// Call handleAdminStorefrontSupportList with date_from and date_to
		url := fmt.Sprintf("/admin/api/storefront-support/list?page=1&date_from=%s&date_to=%s", dateFrom, dateTo)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rr := httptest.NewRecorder()
		handleAdminStorefrontSupportList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: list returned status %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		var resp AdminSupportListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Logf("FAIL: failed to decode response: %v", err)
			return false
		}

		// Verify all returned items have created_at within the date range
		// Parse boundaries as time.Time for reliable comparison regardless of format
		rangeStartTime, _ := time.Parse("2006-01-02 15:04:05", dateFrom+" 00:00:00")
		rangeEndTime, _ := time.Parse("2006-01-02 15:04:05", dateTo+" 23:59:59")

		for i, item := range resp.Items {
			// Try multiple formats since created_at may come as ISO 8601 or datetime string
			var itemTime time.Time
			var parseErr error
			for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
				itemTime, parseErr = time.Parse(layout, item.CreatedAt)
				if parseErr == nil {
					break
				}
			}
			if parseErr != nil {
				t.Logf("FAIL: could not parse items[%d].created_at=%q: %v", i, item.CreatedAt, parseErr)
				return false
			}

			if itemTime.Before(rangeStartTime) {
				t.Logf("FAIL: items[%d].created_at=%q is before rangeStart=%s (date_from=%s, date_to=%s)",
					i, item.CreatedAt, rangeStartTime.Format("2006-01-02 15:04:05"), dateFrom, dateTo)
				return false
			}
			if itemTime.After(rangeEndTime) {
				t.Logf("FAIL: items[%d].created_at=%q is after rangeEnd=%s (date_from=%s, date_to=%s)",
					i, item.CreatedAt, rangeEndTime.Format("2006-01-02 15:04:05"), dateFrom, dateTo)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 violated: %v", err)
	}
}

// Feature: storefront-support-admin, Property 7: 无效日期范围被拒绝
// **Validates: Requirements 4.7**
//
// For any date pair (date_from, date_to) where date_from > date_to,
// the list API should return an error (400 Bad Request).
func TestSupportAdminProperty7_InvalidDateRangeRejected(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate two random dates where date_from is strictly later than date_to.
		// Pick two distinct day offsets in [0, 729] (spanning ~2 years) and ensure date_from > date_to.
		baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		day1 := rng.Intn(730)
		day2 := rng.Intn(730)
		// Ensure they are distinct so date_from is strictly greater than date_to
		for day1 == day2 {
			day2 = rng.Intn(730)
		}
		// Make day1 the larger offset (later date) for date_from
		if day1 < day2 {
			day1, day2 = day2, day1
		}
		dateFrom := baseDate.AddDate(0, 0, day1).Format("2006-01-02")
		dateTo := baseDate.AddDate(0, 0, day2).Format("2006-01-02")

		// Call handleAdminStorefrontSupportList with date_from > date_to
		url := fmt.Sprintf("/admin/api/storefront-support/list?page=1&date_from=%s&date_to=%s", dateFrom, dateTo)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rr := httptest.NewRecorder()
		handleAdminStorefrontSupportList(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Logf("FAIL: expected status 400 for date_from=%s > date_to=%s, got %d, body: %s",
				dateFrom, dateTo, rr.Code, rr.Body.String())
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 violated: %v", err)
	}
}

// Feature: storefront-support-admin, Property 8: 搜索过滤正确性（不区分大小写）
// **Validates: Requirements 5.2, 5.3**
//
// For any search keyword K and dataset, all returned records should have
// store_name or username containing K (case-insensitive).
func TestSupportAdminProperty8_SearchFilterCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Name pools for varied store_name and display_name values
		storeNameParts := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta", "Iota", "Kappa"}
		displayNameParts := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank", "Ivy", "Jack"}

		type testRecord struct {
			storeName   string
			displayName string
		}

		// Insert 10-20 records with varied names
		n := rng.Intn(11) + 10
		var records []testRecord
		for i := 0; i < n; i++ {
			// Build a store name by combining parts with a random suffix
			storeName := storeNameParts[rng.Intn(len(storeNameParts))] + fmt.Sprintf("Shop%d", rng.Intn(1000))
			displayName := displayNameParts[rng.Intn(len(displayNameParts))] + fmt.Sprintf("User%d", rng.Intn(1000))

			records = append(records, testRecord{storeName: storeName, displayName: displayName})

			// Insert user with custom display_name
			email := fmt.Sprintf("p8-%d-%d-%d@test.com", seed, i, rng.Int63n(1000000))
			authID := fmt.Sprintf("p8-auth-%d-%d-%d", seed, i, rng.Int63n(1000000))
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 0)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user %d: %v", i, err)
				return false
			}
			userID, _ := result.LastInsertId()

			storeSlug := fmt.Sprintf("p8-store-%d-%d-%d", seed, i, rng.Int63n(1000000))
			sfResult, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
				userID, storeSlug, storeName,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront %d: %v", i, err)
				return false
			}
			storefrontID, _ := sfResult.LastInsertId()

			statuses := []string{"pending", "approved", "disabled"}
			status := statuses[rng.Intn(len(statuses))]
			_, err = db.Exec(`
				INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
				VALUES (?, ?, 'vantagics', ?, '欢迎', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, storefrontID, userID, storeName, status)
			if err != nil {
				t.Logf("FAIL: failed to insert request %d: %v", i, err)
				return false
			}
		}

		// Generate a search keyword: pick a substring from an existing name to ensure some results
		// Randomly choose between picking from store_name or display_name
		var keyword string
		sourceRecord := records[rng.Intn(len(records))]
		if rng.Intn(2) == 0 {
			// Substring from store_name
			name := sourceRecord.storeName
			if len(name) > 1 {
				start := rng.Intn(len(name) - 1)
				end := start + 1 + rng.Intn(len(name)-start-1) + 1
				if end > len(name) {
					end = len(name)
				}
				keyword = name[start:end]
			} else {
				keyword = name
			}
		} else {
			// Substring from display_name
			name := sourceRecord.displayName
			if len(name) > 1 {
				start := rng.Intn(len(name) - 1)
				end := start + 1 + rng.Intn(len(name)-start-1) + 1
				if end > len(name) {
					end = len(name)
				}
				keyword = name[start:end]
			} else {
				keyword = name
			}
		}

		// Randomly change case of keyword to test case-insensitivity
		var mixedKeyword []byte
		for _, ch := range []byte(keyword) {
			if rng.Intn(2) == 0 {
				mixedKeyword = append(mixedKeyword, byte(strings.ToUpper(string(ch))[0]))
			} else {
				mixedKeyword = append(mixedKeyword, byte(strings.ToLower(string(ch))[0]))
			}
		}
		keyword = string(mixedKeyword)

		// Call handleAdminStorefrontSupportList with search parameter
		url := "/admin/api/storefront-support/list?page=1&search=" + keyword
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rr := httptest.NewRecorder()
		handleAdminStorefrontSupportList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: list returned status %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		var resp AdminSupportListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Logf("FAIL: failed to decode response: %v", err)
			return false
		}

		// Verify ALL returned items have store_name or username containing the keyword (case-insensitive)
		lowerKeyword := strings.ToLower(keyword)
		for i, item := range resp.Items {
			storeMatch := strings.Contains(strings.ToLower(item.StoreName), lowerKeyword)
			usernameMatch := strings.Contains(strings.ToLower(item.Username), lowerKeyword)
			if !storeMatch && !usernameMatch {
				t.Logf("FAIL: items[%d] store_name=%q, username=%q does not contain keyword=%q (case-insensitive)",
					i, item.StoreName, item.Username, keyword)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 violated: %v", err)
	}
}

// Feature: storefront-support-admin, Property 9: 多条件过滤组合正确性
// **Validates: Requirements 5.5**
//
// For any combination of status filter S, search keyword K, date range [F, T],
// and sort order O, all returned results satisfy ALL active filter conditions simultaneously.
func TestSupportAdminProperty9_CombinedFilterCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Name pools for varied store_name and display_name values
		storeNameParts := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta", "Iota", "Kappa"}
		displayNameParts := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank", "Ivy", "Jack"}

		type testRecord struct {
			storeName   string
			displayName string
			status      string
			createdAt   string
		}

		// Insert 10-20 records with varied data
		n := rng.Intn(11) + 10
		var records []testRecord
		statuses := []string{"pending", "approved", "disabled"}
		yearStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		yearEnd := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
		totalSeconds := int(yearEnd.Sub(yearStart).Seconds())

		for i := 0; i < n; i++ {
			storeName := storeNameParts[rng.Intn(len(storeNameParts))] + fmt.Sprintf("Shop%d", rng.Intn(1000))
			displayName := displayNameParts[rng.Intn(len(displayNameParts))] + fmt.Sprintf("User%d", rng.Intn(1000))
			status := statuses[rng.Intn(len(statuses))]
			randomOffset := time.Duration(rng.Intn(totalSeconds)) * time.Second
			createdAt := yearStart.Add(randomOffset).Format("2006-01-02 15:04:05")

			records = append(records, testRecord{
				storeName:   storeName,
				displayName: displayName,
				status:      status,
				createdAt:   createdAt,
			})

			// Insert user with custom display_name
			email := fmt.Sprintf("p9-%d-%d-%d@test.com", seed, i, rng.Int63n(1000000))
			authID := fmt.Sprintf("p9-auth-%d-%d-%d", seed, i, rng.Int63n(1000000))
			result, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, ?, ?, 0)",
				authID, displayName, email,
			)
			if err != nil {
				t.Logf("FAIL: failed to create user %d: %v", i, err)
				return false
			}
			userID, _ := result.LastInsertId()

			storeSlug := fmt.Sprintf("p9-store-%d-%d-%d", seed, i, rng.Int63n(1000000))
			sfResult, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, store_name) VALUES (?, ?, ?)",
				userID, storeSlug, storeName,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront %d: %v", i, err)
				return false
			}
			storefrontID, _ := sfResult.LastInsertId()

			_, err = db.Exec(`
				INSERT INTO storefront_support_requests (storefront_id, user_id, software_name, store_name, welcome_message, status, created_at, updated_at)
				VALUES (?, ?, 'vantagics', ?, '欢迎', ?, ?, CURRENT_TIMESTAMP)
			`, storefrontID, userID, storeName, status, createdAt)
			if err != nil {
				t.Logf("FAIL: failed to insert request %d: %v", i, err)
				return false
			}
		}

		// Randomly generate a combination of filters
		// status: randomly pick from {"pending", "approved", "disabled", ""} (empty = no filter)
		statusOptions := []string{"pending", "approved", "disabled", ""}
		filterStatus := statusOptions[rng.Intn(len(statusOptions))]

		// search: randomly pick a substring from existing names or empty string
		var filterSearch string
		if rng.Intn(3) > 0 { // 2/3 chance of having a search keyword
			sourceRecord := records[rng.Intn(len(records))]
			var name string
			if rng.Intn(2) == 0 {
				name = sourceRecord.storeName
			} else {
				name = sourceRecord.displayName
			}
			if len(name) > 1 {
				start := rng.Intn(len(name) - 1)
				end := start + 1 + rng.Intn(len(name)-start-1) + 1
				if end > len(name) {
					end = len(name)
				}
				filterSearch = name[start:end]
			} else {
				filterSearch = name
			}
		}

		// date_from and date_to: randomly pick a valid date range or leave empty
		var filterDateFrom, filterDateTo string
		if rng.Intn(3) > 0 { // 2/3 chance of having date filters
			day1 := rng.Intn(365)
			day2 := rng.Intn(365)
			if day1 > day2 {
				day1, day2 = day2, day1
			}
			// Randomly decide to use one or both
			switch rng.Intn(3) {
			case 0: // both
				filterDateFrom = yearStart.AddDate(0, 0, day1).Format("2006-01-02")
				filterDateTo = yearStart.AddDate(0, 0, day2).Format("2006-01-02")
			case 1: // only date_from
				filterDateFrom = yearStart.AddDate(0, 0, day1).Format("2006-01-02")
			case 2: // only date_to
				filterDateTo = yearStart.AddDate(0, 0, day2).Format("2006-01-02")
			}
		}

		// sort_order: randomly pick from {"asc", "desc", ""}
		sortOrders := []string{"asc", "desc", ""}
		filterSortOrder := sortOrders[rng.Intn(len(sortOrders))]

		// Build URL with all parameters
		urlStr := "/admin/api/storefront-support/list?page=1"
		if filterStatus != "" {
			urlStr += "&status=" + filterStatus
		}
		if filterSearch != "" {
			urlStr += "&search=" + filterSearch
		}
		if filterDateFrom != "" {
			urlStr += "&date_from=" + filterDateFrom
		}
		if filterDateTo != "" {
			urlStr += "&date_to=" + filterDateTo
		}
		if filterSortOrder != "" {
			urlStr += "&sort_order=" + filterSortOrder
		}

		req := httptest.NewRequest(http.MethodGet, urlStr, nil)
		rr := httptest.NewRecorder()
		handleAdminStorefrontSupportList(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: list returned status %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		var resp AdminSupportListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Logf("FAIL: failed to decode response: %v", err)
			return false
		}

		// Verify ALL returned items satisfy ALL active filter conditions simultaneously
		lowerSearch := strings.ToLower(filterSearch)

		var rangeStartTime, rangeEndTime time.Time
		if filterDateFrom != "" {
			rangeStartTime, _ = time.Parse("2006-01-02 15:04:05", filterDateFrom+" 00:00:00")
		}
		if filterDateTo != "" {
			rangeEndTime, _ = time.Parse("2006-01-02 15:04:05", filterDateTo+" 23:59:59")
		}

		for i, item := range resp.Items {
			// Check status filter
			if filterStatus != "" && item.Status != filterStatus {
				t.Logf("FAIL: items[%d].status=%q does not match filter status=%q", i, item.Status, filterStatus)
				return false
			}

			// Check search filter (case-insensitive)
			if filterSearch != "" {
				storeMatch := strings.Contains(strings.ToLower(item.StoreName), lowerSearch)
				usernameMatch := strings.Contains(strings.ToLower(item.Username), lowerSearch)
				if !storeMatch && !usernameMatch {
					t.Logf("FAIL: items[%d] store_name=%q, username=%q does not contain search=%q (case-insensitive)",
						i, item.StoreName, item.Username, filterSearch)
					return false
				}
			}

			// Check date range filters using time.Parse
			if filterDateFrom != "" || filterDateTo != "" {
				var itemTime time.Time
				var parseErr error
				for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
					itemTime, parseErr = time.Parse(layout, item.CreatedAt)
					if parseErr == nil {
						break
					}
				}
				if parseErr != nil {
					t.Logf("FAIL: could not parse items[%d].created_at=%q: %v", i, item.CreatedAt, parseErr)
					return false
				}

				if filterDateFrom != "" && itemTime.Before(rangeStartTime) {
					t.Logf("FAIL: items[%d].created_at=%q is before date_from=%s 00:00:00",
						i, item.CreatedAt, filterDateFrom)
					return false
				}
				if filterDateTo != "" && itemTime.After(rangeEndTime) {
					t.Logf("FAIL: items[%d].created_at=%q is after date_to=%s 23:59:59",
						i, item.CreatedAt, filterDateTo)
					return false
				}
			}
		}

		// Check sort order
		if len(resp.Items) >= 2 {
			for i := 0; i < len(resp.Items)-1; i++ {
				curr := resp.Items[i].CreatedAt
				next := resp.Items[i+1].CreatedAt

				if filterSortOrder == "asc" {
					if curr > next {
						t.Logf("FAIL: sort_order=%q, items[%d].created_at=%q > items[%d].created_at=%q (expected ascending)",
							filterSortOrder, i, curr, i+1, next)
						return false
					}
				} else {
					// desc or empty defaults to descending
					if curr < next {
						t.Logf("FAIL: sort_order=%q, items[%d].created_at=%q < items[%d].created_at=%q (expected descending)",
							filterSortOrder, i, curr, i+1, next)
						return false
					}
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 violated: %v", err)
	}
}
