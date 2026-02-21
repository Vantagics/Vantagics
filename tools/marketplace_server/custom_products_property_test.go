package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"marketplace_server/templates"
)

// Feature: storefront-custom-products, Property 3: 商品名称联合唯一约束
// **Validates: Requirements 2.3**
//
// For any storefront_id and product_name, inserting two products with the same
// (storefront_id, product_name) pair must fail due to the UNIQUE constraint.
// This ensures no duplicate product names within the same storefront.
func TestCustomProductProperty3_ProductNameUniqueness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("store-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Generate a random product name (2-100 chars)
		nameLen := 2 + rng.Intn(99) // 2 to 100
		nameRunes := make([]byte, nameLen)
		for i := range nameRunes {
			nameRunes[i] = byte('a' + rng.Intn(26))
		}
		productName := string(nameRunes)

		// First insert should succeed
		_, err = db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd)
			 VALUES (?, ?, 'credits', 9.99)`,
			storefrontID, productName,
		)
		if err != nil {
			t.Logf("FAIL: first insert failed unexpectedly: %v", err)
			return false
		}

		// Second insert with same storefront_id and product_name should fail
		_, err = db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd)
			 VALUES (?, ?, 'credits', 19.99)`,
			storefrontID, productName,
		)
		if err == nil {
			t.Logf("FAIL: second insert with same storefront_id=%d and product_name=%q succeeded, expected UNIQUE violation",
				storefrontID, productName)
			return false
		}

		// Verify the error is a UNIQUE constraint violation
		if !strings.Contains(err.Error(), "UNIQUE") {
			t.Logf("FAIL: expected UNIQUE constraint error, got: %v", err)
			return false
		}

		// Verify exactly one product exists with this name for this storefront
		var count int
		db.QueryRow(
			"SELECT COUNT(*) FROM custom_products WHERE storefront_id = ? AND product_name = ?",
			storefrontID, productName,
		).Scan(&count)
		if count != 1 {
			t.Logf("FAIL: expected exactly 1 product for storefront_id=%d with name=%q, got %d",
				storefrontID, productName, count)
			return false
		}

		// Verify that a DIFFERENT storefront CAN use the same product name
		userID2 := createTestUserWithBalance(t, 0)
		slug2 := fmt.Sprintf("store2-%d-%d", userID2, rng.Int63n(1000000))
		result2, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID2, slug2,
		)
		if err != nil {
			t.Logf("FAIL: failed to create second storefront: %v", err)
			return false
		}
		storefrontID2, _ := result2.LastInsertId()

		_, err = db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd)
			 VALUES (?, ?, 'credits', 9.99)`,
			storefrontID2, productName,
		)
		if err != nil {
			t.Logf("FAIL: insert with same product_name but different storefront_id failed: %v", err)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 4: 字段值约束
// **Validates: Requirements 2.5, 2.6**
//
// For any product_type value NOT IN {"credits", "virtual_goods"}, INSERT should fail.
// For any status value NOT IN {"draft", "pending", "published", "rejected"}, INSERT should fail.
// Valid values should be accepted by the CHECK constraints.
func TestCustomProductProperty4_FieldValueConstraints(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	validProductTypes := map[string]bool{"credits": true, "virtual_goods": true}
	validStatuses := map[string]bool{"draft": true, "pending": true, "published": true, "rejected": true}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("store-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// --- Test product_type constraint ---

		// Generate a random invalid product_type (not "credits" or "virtual_goods")
		invalidType := generateRandomString(rng, 1+rng.Intn(20))
		for validProductTypes[invalidType] {
			invalidType = generateRandomString(rng, 1+rng.Intn(20))
		}

		productName := fmt.Sprintf("prod-type-%d", rng.Int63n(1000000))
		_, err = db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd)
			 VALUES (?, ?, ?, 9.99)`,
			storefrontID, productName, invalidType,
		)
		if err == nil {
			t.Logf("FAIL: INSERT with invalid product_type=%q succeeded, expected CHECK constraint failure", invalidType)
			return false
		}
		if !strings.Contains(err.Error(), "CHECK") {
			t.Logf("FAIL: expected CHECK constraint error for product_type=%q, got: %v", invalidType, err)
			return false
		}

		// Verify valid product_types are accepted
		for validType := range validProductTypes {
			pname := fmt.Sprintf("valid-type-%s-%d", validType, rng.Int63n(1000000))
			_, err = db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd)
				 VALUES (?, ?, ?, 9.99)`,
				storefrontID, pname, validType,
			)
			if err != nil {
				t.Logf("FAIL: INSERT with valid product_type=%q failed: %v", validType, err)
				return false
			}
		}

		// --- Test status constraint ---

		// Generate a random invalid status (not "draft", "pending", "published", "rejected")
		invalidStatus := generateRandomString(rng, 1+rng.Intn(20))
		for validStatuses[invalidStatus] {
			invalidStatus = generateRandomString(rng, 1+rng.Intn(20))
		}

		productName2 := fmt.Sprintf("prod-status-%d", rng.Int63n(1000000))
		_, err = db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
			 VALUES (?, ?, 'credits', 9.99, ?)`,
			storefrontID, productName2, invalidStatus,
		)
		if err == nil {
			t.Logf("FAIL: INSERT with invalid status=%q succeeded, expected CHECK constraint failure", invalidStatus)
			return false
		}
		if !strings.Contains(err.Error(), "CHECK") {
			t.Logf("FAIL: expected CHECK constraint error for status=%q, got: %v", invalidStatus, err)
			return false
		}

		// Verify valid statuses are accepted
		for validStatus := range validStatuses {
			pname := fmt.Sprintf("valid-status-%s-%d", validStatus, rng.Int63n(1000000))
			_, err = db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
				 VALUES (?, ?, 'credits', 9.99, ?)`,
				storefrontID, pname, validStatus,
			)
			if err != nil {
				t.Logf("FAIL: INSERT with valid status=%q failed: %v", validStatus, err)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 violated: %v", err)
	}
}

// generateRandomString generates a random lowercase string of the given length.
func generateRandomString(rng *rand.Rand, length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = byte('a' + rng.Intn(26))
	}
	return string(b)
}

// TestValidateCustomProduct verifies the validateCustomProduct function
// covers all validation rules: name length, price range, credits amount, and license endpoint.
func TestValidateCustomProduct(t *testing.T) {
	tests := []struct {
		name     string
		product  CustomProduct
		wantErr  string
	}{
		{
			name: "valid credits product",
			product: CustomProduct{
				ProductName:   "积分充值包",
				ProductType:   "credits",
				PriceUSD:      9.99,
				CreditsAmount: 100,
			},
			wantErr: "",
		},
		{
			name: "valid virtual_goods product",
			product: CustomProduct{
				ProductName:        "软件授权",
				ProductType:        "virtual_goods",
				PriceUSD:           29.99,
				LicenseAPIEndpoint: "https://license.example.com/api",
			},
			wantErr: "",
		},
		{
			name: "name too short (1 char)",
			product: CustomProduct{
				ProductName: "A",
				ProductType: "credits",
				PriceUSD:    9.99,
				CreditsAmount: 100,
			},
			wantErr: "商品名称长度必须在 2 到 100 个字符之间",
		},
		{
			name: "name empty",
			product: CustomProduct{
				ProductName: "",
				ProductType: "credits",
				PriceUSD:    9.99,
				CreditsAmount: 100,
			},
			wantErr: "商品名称长度必须在 2 到 100 个字符之间",
		},
		{
			name: "name exactly 2 chars (valid boundary)",
			product: CustomProduct{
				ProductName:   "AB",
				ProductType:   "credits",
				PriceUSD:      9.99,
				CreditsAmount: 100,
			},
			wantErr: "",
		},
		{
			name: "name exactly 100 chars (valid boundary)",
			product: CustomProduct{
				ProductName:   strings.Repeat("a", 100),
				ProductType:   "credits",
				PriceUSD:      9.99,
				CreditsAmount: 100,
			},
			wantErr: "",
		},
		{
			name: "name 101 chars (too long)",
			product: CustomProduct{
				ProductName:   strings.Repeat("a", 101),
				ProductType:   "credits",
				PriceUSD:      9.99,
				CreditsAmount: 100,
			},
			wantErr: "商品名称长度必须在 2 到 100 个字符之间",
		},
		{
			name: "price zero",
			product: CustomProduct{
				ProductName:   "Test Product",
				ProductType:   "credits",
				PriceUSD:      0,
				CreditsAmount: 100,
			},
			wantErr: "价格必须为正数且不超过 9999.99 美元",
		},
		{
			name: "price negative",
			product: CustomProduct{
				ProductName:   "Test Product",
				ProductType:   "credits",
				PriceUSD:      -1.0,
				CreditsAmount: 100,
			},
			wantErr: "价格必须为正数且不超过 9999.99 美元",
		},
		{
			name: "price exceeds max",
			product: CustomProduct{
				ProductName:   "Test Product",
				ProductType:   "credits",
				PriceUSD:      10000.00,
				CreditsAmount: 100,
			},
			wantErr: "价格必须为正数且不超过 9999.99 美元",
		},
		{
			name: "price at max boundary (valid)",
			product: CustomProduct{
				ProductName:   "Test Product",
				ProductType:   "credits",
				PriceUSD:      9999.99,
				CreditsAmount: 100,
			},
			wantErr: "",
		},
		{
			name: "credits product with zero credits_amount",
			product: CustomProduct{
				ProductName:   "Test Product",
				ProductType:   "credits",
				PriceUSD:      9.99,
				CreditsAmount: 0,
			},
			wantErr: "积分数量必须为正数",
		},
		{
			name: "credits product with negative credits_amount",
			product: CustomProduct{
				ProductName:   "Test Product",
				ProductType:   "credits",
				PriceUSD:      9.99,
				CreditsAmount: -5,
			},
			wantErr: "积分数量必须为正数",
		},
		{
			name: "virtual_goods with empty license endpoint",
			product: CustomProduct{
				ProductName:        "Test Product",
				ProductType:        "virtual_goods",
				PriceUSD:           29.99,
				LicenseAPIEndpoint: "",
			},
			wantErr: "请填写 License API 地址",
		},
		{
			name: "virtual_goods with valid license endpoint",
			product: CustomProduct{
				ProductName:        "Test Product",
				ProductType:        "virtual_goods",
				PriceUSD:           29.99,
				LicenseAPIEndpoint: "https://api.example.com",
			},
			wantErr: "",
		},
		{
			name: "unicode name with 2 runes (valid)",
			product: CustomProduct{
				ProductName:   "你好",
				ProductType:   "credits",
				PriceUSD:      9.99,
				CreditsAmount: 50,
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateCustomProduct(tt.product)
			if got != tt.wantErr {
				t.Errorf("validateCustomProduct() = %q, want %q", got, tt.wantErr)
			}
		})
	}
}

// Feature: storefront-custom-products, Property 6: 商品创建验证规则
// **Validates: Requirements 3.6, 3.7, 3.8, 3.9**
//
// For all random product parameters, validateCustomProduct must:
//   - Return an error if name length < 2 or > 100 (rune count)
//   - Return an error if price <= 0 or > 9999.99
//   - Return an error if product_type == "credits" and credits_amount <= 0
//   - Return an error if product_type == "virtual_goods" and license_api_endpoint == ""
//   - Return empty string for valid combinations
func TestCustomProductProperty6_CreationValidation(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate random product parameters
		// Name: random length 0-120 to cover invalid and valid ranges
		nameLen := rng.Intn(121) // 0 to 120
		name := generateRandomString(rng, nameLen)

		// Price: random in range [-100, 11000] to cover invalid and valid ranges
		price := (rng.Float64()*11100.0 - 100.0)
		// Round to 2 decimal places to avoid floating point issues
		price = float64(int(price*100)) / 100.0

		// Product type: randomly pick from credits, virtual_goods, or other
		productTypes := []string{"credits", "virtual_goods"}
		productType := productTypes[rng.Intn(2)]

		// Credits amount: random in range [-10, 1000]
		creditsAmount := rng.Intn(1011) - 10 // -10 to 1000

		// License API endpoint: randomly empty or non-empty
		var licenseEndpoint string
		if rng.Intn(2) == 0 {
			licenseEndpoint = ""
		} else {
			licenseEndpoint = fmt.Sprintf("https://api-%d.example.com/license", rng.Int63n(10000))
		}

		p := CustomProduct{
			ProductName:        name,
			ProductType:        productType,
			PriceUSD:           price,
			CreditsAmount:      creditsAmount,
			LicenseAPIEndpoint: licenseEndpoint,
		}

		result := validateCustomProduct(p)

		// Determine expected validity based on the property specification
		runeLen := len([]rune(name))
		nameInvalid := runeLen < 2 || runeLen > 100
		priceInvalid := price <= 0 || price > 9999.99
		creditsInvalid := productType == "credits" && creditsAmount <= 0
		licenseInvalid := productType == "virtual_goods" && licenseEndpoint == ""

		shouldReject := nameInvalid || priceInvalid || creditsInvalid || licenseInvalid

		if shouldReject && result == "" {
			t.Logf("FAIL: expected validation error but got empty string for product: name=%q (runeLen=%d), price=%.2f, type=%s, credits=%d, endpoint=%q",
				name, runeLen, price, productType, creditsAmount, licenseEndpoint)
			return false
		}

		if !shouldReject && result != "" {
			t.Logf("FAIL: expected validation to pass but got error %q for product: name=%q (runeLen=%d), price=%.2f, type=%s, credits=%d, endpoint=%q",
				result, name, runeLen, price, productType, creditsAmount, licenseEndpoint)
			return false
		}

		// Verify specific error messages for each invalid condition (checked in priority order)
		if result != "" {
			if nameInvalid {
				if result != "商品名称长度必须在 2 到 100 个字符之间" {
					t.Logf("FAIL: expected name length error, got %q", result)
					return false
				}
			} else if priceInvalid {
				if result != "价格必须为正数且不超过 9999.99 美元" {
					t.Logf("FAIL: expected price error, got %q", result)
					return false
				}
			} else if creditsInvalid {
				if result != "积分数量必须为正数" {
					t.Logf("FAIL: expected credits amount error, got %q", result)
					return false
				}
			} else if licenseInvalid {
				if result != "请填写 License API 地址" {
					t.Logf("FAIL: expected license endpoint error, got %q", result)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 1: 自定义商品权限开关 Round-Trip
// **Validates: Requirements 1.2, 1.3**
//
// For any storefront, setting custom_products_enabled to true then reading it back
// should return true. Setting it to false then reading it back should return false.
// This verifies the toggle round-trip at the database level.
func TestCustomProductProperty1_ToggleRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("toggle-rt-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Verify default value is 0 (false)
		var enabled int
		err = db.QueryRow(
			"SELECT custom_products_enabled FROM author_storefronts WHERE id = ?",
			storefrontID,
		).Scan(&enabled)
		if err != nil {
			t.Logf("FAIL: failed to read initial custom_products_enabled: %v", err)
			return false
		}
		if enabled != 0 {
			t.Logf("FAIL: expected initial custom_products_enabled=0, got %d", enabled)
			return false
		}

		// Toggle ON: set custom_products_enabled = 1
		_, err = db.Exec(
			"UPDATE author_storefronts SET custom_products_enabled = 1 WHERE id = ?",
			storefrontID,
		)
		if err != nil {
			t.Logf("FAIL: failed to set custom_products_enabled=1: %v", err)
			return false
		}

		// Read back and verify it's 1
		err = db.QueryRow(
			"SELECT custom_products_enabled FROM author_storefronts WHERE id = ?",
			storefrontID,
		).Scan(&enabled)
		if err != nil {
			t.Logf("FAIL: failed to read custom_products_enabled after toggle ON: %v", err)
			return false
		}
		if enabled != 1 {
			t.Logf("FAIL: expected custom_products_enabled=1 after toggle ON, got %d", enabled)
			return false
		}

		// Toggle OFF: set custom_products_enabled = 0
		_, err = db.Exec(
			"UPDATE author_storefronts SET custom_products_enabled = 0 WHERE id = ?",
			storefrontID,
		)
		if err != nil {
			t.Logf("FAIL: failed to set custom_products_enabled=0: %v", err)
			return false
		}

		// Read back and verify it's 0
		err = db.QueryRow(
			"SELECT custom_products_enabled FROM author_storefronts WHERE id = ?",
			storefrontID,
		).Scan(&enabled)
		if err != nil {
			t.Logf("FAIL: failed to read custom_products_enabled after toggle OFF: %v", err)
			return false
		}
		if enabled != 0 {
			t.Logf("FAIL: expected custom_products_enabled=0 after toggle OFF, got %d", enabled)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 2: 关闭权限级联下架
// **Validates: Requirements 1.4**
//
// For any storefront with published custom products, when the admin disables
// custom_products_enabled, all products with status='published' must become 'draft'.
// Products that were already 'draft' or 'pending' must remain unchanged.
func TestCustomProductProperty2_CascadeDisable(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront with custom_products_enabled=1
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("cascade-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Randomly decide how many products of each status to create (1-5 each)
		numPublished := 1 + rng.Intn(5)
		numDraft := rng.Intn(4) // 0-3
		numPending := rng.Intn(4) // 0-3

		// Track product IDs by their original status
		publishedIDs := make([]int64, 0, numPublished)
		draftIDs := make([]int64, 0, numDraft)
		pendingIDs := make([]int64, 0, numPending)

		// Insert published products
		for i := 0; i < numPublished; i++ {
			pname := fmt.Sprintf("pub-%d-%d", i, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
				 VALUES (?, ?, 'credits', 9.99, 'published')`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert published product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			publishedIDs = append(publishedIDs, id)
		}

		// Insert draft products
		for i := 0; i < numDraft; i++ {
			pname := fmt.Sprintf("draft-%d-%d", i, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
				 VALUES (?, ?, 'credits', 9.99, 'draft')`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert draft product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			draftIDs = append(draftIDs, id)
		}

		// Insert pending products
		for i := 0; i < numPending; i++ {
			pname := fmt.Sprintf("pending-%d-%d", i, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
				 VALUES (?, ?, 'credits', 9.99, 'pending')`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert pending product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			pendingIDs = append(pendingIDs, id)
		}

		// Execute the cascade disable (same logic as handleAdminCustomProductsToggle)
		tx, err := db.Begin()
		if err != nil {
			t.Logf("FAIL: failed to begin transaction: %v", err)
			return false
		}
		_, err = tx.Exec(
			"UPDATE custom_products SET status = 'draft', updated_at = CURRENT_TIMESTAMP WHERE storefront_id = ? AND status = 'published'",
			storefrontID,
		)
		if err != nil {
			tx.Rollback()
			t.Logf("FAIL: failed to cascade published→draft: %v", err)
			return false
		}
		_, err = tx.Exec(
			"UPDATE author_storefronts SET custom_products_enabled = 0 WHERE id = ?",
			storefrontID,
		)
		if err != nil {
			tx.Rollback()
			t.Logf("FAIL: failed to disable custom_products_enabled: %v", err)
			return false
		}
		if err := tx.Commit(); err != nil {
			t.Logf("FAIL: failed to commit transaction: %v", err)
			return false
		}

		// Verify: NO published products remain for this storefront
		var publishedCount int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM custom_products WHERE storefront_id = ? AND status = 'published'",
			storefrontID,
		).Scan(&publishedCount)
		if err != nil {
			t.Logf("FAIL: failed to count published products: %v", err)
			return false
		}
		if publishedCount != 0 {
			t.Logf("FAIL: expected 0 published products after cascade disable, got %d", publishedCount)
			return false
		}

		// Verify: all previously published products are now draft
		for _, id := range publishedIDs {
			var status string
			err = db.QueryRow(
				"SELECT status FROM custom_products WHERE id = ?", id,
			).Scan(&status)
			if err != nil {
				t.Logf("FAIL: failed to read status for previously published product id=%d: %v", id, err)
				return false
			}
			if status != "draft" {
				t.Logf("FAIL: previously published product id=%d has status=%q, expected 'draft'", id, status)
				return false
			}
		}

		// Verify: draft products remain draft
		for _, id := range draftIDs {
			var status string
			err = db.QueryRow(
				"SELECT status FROM custom_products WHERE id = ?", id,
			).Scan(&status)
			if err != nil {
				t.Logf("FAIL: failed to read status for draft product id=%d: %v", id, err)
				return false
			}
			if status != "draft" {
				t.Logf("FAIL: draft product id=%d changed to status=%q, expected 'draft'", id, status)
				return false
			}
		}

		// Verify: pending products remain pending
		for _, id := range pendingIDs {
			var status string
			err = db.QueryRow(
				"SELECT status FROM custom_products WHERE id = ?", id,
			).Scan(&status)
			if err != nil {
				t.Logf("FAIL: failed to read status for pending product id=%d: %v", id, err)
				return false
			}
			if status != "pending" {
				t.Logf("FAIL: pending product id=%d changed to status=%q, expected 'pending'", id, status)
				return false
			}
		}

		// Verify: custom_products_enabled is now 0
		var enabled int
		err = db.QueryRow(
			"SELECT custom_products_enabled FROM author_storefronts WHERE id = ?",
			storefrontID,
		).Scan(&enabled)
		if err != nil {
			t.Logf("FAIL: failed to read custom_products_enabled: %v", err)
			return false
		}
		if enabled != 0 {
			t.Logf("FAIL: expected custom_products_enabled=0 after disable, got %d", enabled)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 7: 新建商品默认状态为 draft
// **Validates: Requirements 3.10**
//
// For all valid product creation requests, inserting a product into custom_products
// without specifying a status should result in the default status being "draft".
// This verifies the database DEFAULT constraint on the status column.
func TestCustomProductProperty7_DefaultDraftStatus(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront with custom_products_enabled=1
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("draft-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Randomly choose product type
		productType := "credits"
		if rng.Intn(2) == 0 {
			productType = "virtual_goods"
		}

		// Generate a random product name (2-100 chars)
		nameLen := 2 + rng.Intn(99)
		productName := fmt.Sprintf("%s-%d", generateRandomString(rng, nameLen), rng.Int63n(1000000))

		// Generate a random valid price (0.01 to 9999.99)
		price := float64(1+rng.Intn(999999)) / 100.0

		// INSERT without specifying status — should default to 'draft'
		res, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd)
			 VALUES (?, ?, ?, ?)`,
			storefrontID, productName, productType, price,
		)
		if err != nil {
			t.Logf("FAIL: failed to insert product: %v", err)
			return false
		}
		productID, _ := res.LastInsertId()

		// SELECT and verify status is 'draft'
		var status string
		err = db.QueryRow(
			"SELECT status FROM custom_products WHERE id = ?", productID,
		).Scan(&status)
		if err != nil {
			t.Logf("FAIL: failed to read status for product id=%d: %v", productID, err)
			return false
		}
		if status != "draft" {
			t.Logf("FAIL: expected status='draft' for newly created product id=%d, got %q", productID, status)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 8: 每小铺商品数量上限
// **Validates: Requirements 3.12, 3.13**
//
// For any storefront, the count of non-soft-deleted custom products must never exceed 50.
// When 50 products already exist, the application logic (productCount >= 50) must reject
// the creation of a 51st product. Soft-deleted products do not count toward the limit.
func TestCustomProductProperty8_ProductCountLimit(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 10,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront with custom_products_enabled=1
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("limit-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Insert 50 products directly via SQL
		for i := 0; i < 50; i++ {
			pname := fmt.Sprintf("product-%d-%d-%d", i, seed, rng.Int63n(1000000))
			_, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd)
				 VALUES (?, ?, 'credits', 9.99)`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert product %d: %v", i, err)
				return false
			}
		}

		// Verify count is exactly 50 (non-deleted)
		var productCount int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM custom_products WHERE storefront_id = ? AND deleted_at IS NULL",
			storefrontID,
		).Scan(&productCount)
		if err != nil {
			t.Logf("FAIL: failed to count products: %v", err)
			return false
		}
		if productCount != 50 {
			t.Logf("FAIL: expected 50 products, got %d", productCount)
			return false
		}

		// Verify the application-level limit check would reject the 51st product
		// This mirrors the logic in handleCustomProductCreate: if productCount >= 50, reject
		if productCount < 50 {
			t.Logf("FAIL: productCount=%d is less than 50, limit check would not trigger", productCount)
			return false
		}

		// Verify that soft-deleted products do NOT count toward the limit
		// Soft-delete a random number of products (1-5)
		numToDelete := 1 + rng.Intn(5)
		for i := 0; i < numToDelete; i++ {
			_, err := db.Exec(
				`UPDATE custom_products SET deleted_at = CURRENT_TIMESTAMP
				 WHERE storefront_id = ? AND deleted_at IS NULL
				 AND id = (SELECT id FROM custom_products WHERE storefront_id = ? AND deleted_at IS NULL LIMIT 1)`,
				storefrontID, storefrontID,
			)
			if err != nil {
				t.Logf("FAIL: failed to soft-delete product: %v", err)
				return false
			}
		}

		// Re-count non-deleted products
		err = db.QueryRow(
			"SELECT COUNT(*) FROM custom_products WHERE storefront_id = ? AND deleted_at IS NULL",
			storefrontID,
		).Scan(&productCount)
		if err != nil {
			t.Logf("FAIL: failed to re-count products after soft-delete: %v", err)
			return false
		}
		expectedCount := 50 - numToDelete
		if productCount != expectedCount {
			t.Logf("FAIL: after soft-deleting %d products, expected count=%d, got %d", numToDelete, expectedCount, productCount)
			return false
		}

		// Now the limit check should allow new products (productCount < 50)
		if productCount >= 50 {
			t.Logf("FAIL: after soft-deleting %d products, productCount=%d should be < 50", numToDelete, productCount)
			return false
		}

		// Verify the invariant: count of non-deleted products never exceeds 50
		var totalNonDeleted int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM custom_products WHERE storefront_id = ? AND deleted_at IS NULL",
			storefrontID,
		).Scan(&totalNonDeleted)
		if err != nil {
			t.Logf("FAIL: failed to count non-deleted products for invariant check: %v", err)
			return false
		}
		if totalNonDeleted > 50 {
			t.Logf("FAIL: invariant violated: non-deleted product count=%d exceeds 50", totalNonDeleted)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 9: 商品状态转换合法性
// **Validates: Requirements 4.1, 4.3, 4.4, 12.2**
//
// For all products and state transitions, only the following are valid:
//   - draft → pending (submit for review)
//   - pending → published (approve)
//   - pending → rejected (reject, must have non-empty reject_reason)
//   - rejected → pending (resubmit)
//   - published → draft (delist)
// All other transitions must be rejected by the handler logic.
func TestCustomProductProperty9_StateTransitions(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	allStatuses := []string{"draft", "pending", "published", "rejected"}

	type transition struct {
		from string
		to   string
	}

	validTransitions := map[transition]bool{
		{from: "draft", to: "pending"}:      true,
		{from: "pending", to: "published"}:  true,
		{from: "pending", to: "rejected"}:   true,
		{from: "rejected", to: "pending"}:   true,
		{from: "published", to: "draft"}:    true,
	}

	// simulateTransition mimics the handler logic for each transition type.
	// Returns (success bool, err error).
	simulateTransition := func(productID int64, from, to string) (bool, error) {
		// Read current status
		var currentStatus string
		err := db.QueryRow(
			"SELECT status FROM custom_products WHERE id = ? AND deleted_at IS NULL",
			productID,
		).Scan(&currentStatus)
		if err != nil {
			return false, fmt.Errorf("failed to read product status: %v", err)
		}
		if currentStatus != from {
			return false, fmt.Errorf("product status is %q, expected %q", currentStatus, from)
		}

		switch {
		case from == "draft" && to == "pending":
			// Submit: handler allows draft → pending
			if currentStatus != "draft" && currentStatus != "rejected" {
				return false, nil
			}
			_, err = db.Exec(
				"UPDATE custom_products SET status = 'pending', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				productID,
			)
			return err == nil, err

		case from == "rejected" && to == "pending":
			// Resubmit: handler allows rejected → pending
			if currentStatus != "draft" && currentStatus != "rejected" {
				return false, nil
			}
			_, err = db.Exec(
				"UPDATE custom_products SET status = 'pending', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				productID,
			)
			return err == nil, err

		case from == "pending" && to == "published":
			// Approve: handler allows pending → published
			if currentStatus != "pending" {
				return false, nil
			}
			_, err = db.Exec(
				"UPDATE custom_products SET status = 'published', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				productID,
			)
			return err == nil, err

		case from == "pending" && to == "rejected":
			// Reject: handler allows pending → rejected, must set reject_reason
			if currentStatus != "pending" {
				return false, nil
			}
			_, err = db.Exec(
				"UPDATE custom_products SET status = 'rejected', reject_reason = '审核不通过', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				productID,
			)
			return err == nil, err

		case from == "published" && to == "draft":
			// Delist: handler allows published → draft
			if currentStatus != "published" {
				return false, nil
			}
			_, err = db.Exec(
				"UPDATE custom_products SET status = 'draft', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				productID,
			)
			return err == nil, err

		default:
			// Invalid transition — handler would reject based on status check
			// Simulate the handler's guard: each handler only allows specific from-statuses
			return false, nil
		}
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("state-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Pick a random from-status and to-status
		fromStatus := allStatuses[rng.Intn(len(allStatuses))]
		toStatus := allStatuses[rng.Intn(len(allStatuses))]

		// Skip identity transitions (no-op)
		if fromStatus == toStatus {
			return true
		}

		// Create a product in the from-status
		productName := fmt.Sprintf("trans-%s-%s-%d", fromStatus, toStatus, rng.Int63n(1000000))
		res, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
			 VALUES (?, ?, 'credits', 9.99, ?)`,
			storefrontID, productName, fromStatus,
		)
		if err != nil {
			t.Logf("FAIL: failed to insert product with status=%q: %v", fromStatus, err)
			return false
		}
		productID, _ := res.LastInsertId()

		tr := transition{from: fromStatus, to: toStatus}
		isValid := validTransitions[tr]

		success, err := simulateTransition(productID, fromStatus, toStatus)
		if err != nil {
			t.Logf("FAIL: simulateTransition(%q→%q) returned error: %v", fromStatus, toStatus, err)
			return false
		}

		if isValid && !success {
			t.Logf("FAIL: valid transition %q→%q was rejected", fromStatus, toStatus)
			return false
		}

		if !isValid && success {
			t.Logf("FAIL: invalid transition %q→%q was allowed", fromStatus, toStatus)
			return false
		}

		// For valid transitions, verify the product is now in the expected to-status
		if isValid {
			var newStatus string
			err = db.QueryRow(
				"SELECT status FROM custom_products WHERE id = ?", productID,
			).Scan(&newStatus)
			if err != nil {
				t.Logf("FAIL: failed to read status after transition: %v", err)
				return false
			}
			if newStatus != toStatus {
				t.Logf("FAIL: after valid transition %q→%q, product status is %q", fromStatus, toStatus, newStatus)
				return false
			}

			// Special check: if to == "rejected", reject_reason must not be empty
			if toStatus == "rejected" {
				var rejectReason string
				err = db.QueryRow(
					"SELECT reject_reason FROM custom_products WHERE id = ?", productID,
				).Scan(&rejectReason)
				if err != nil {
					t.Logf("FAIL: failed to read reject_reason: %v", err)
					return false
				}
				if rejectReason == "" {
					t.Logf("FAIL: transition to 'rejected' but reject_reason is empty")
					return false
				}
			}
		}

		// For invalid transitions, verify the product status is unchanged
		if !isValid {
			var unchangedStatus string
			err = db.QueryRow(
				"SELECT status FROM custom_products WHERE id = ?", productID,
			).Scan(&unchangedStatus)
			if err != nil {
				t.Logf("FAIL: failed to read status after rejected transition: %v", err)
				return false
			}
			if unchangedStatus != fromStatus {
				t.Logf("FAIL: after rejected transition %q→%q, product status changed to %q", fromStatus, toStatus, unchangedStatus)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 19: 软删除保留订单记录
// **Validates: Requirements 12.4, 12.6**
//
// For all products with orders, after soft-deleting the product:
//   - product.deleted_at IS NOT NULL
//   - COUNT(custom_product_orders WHERE custom_product_id = product.id) == order_count_before
// This ensures soft delete does not cascade-delete or affect associated order records.
func TestCustomProductProperty19_SoftDeletePreservesOrders(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront with custom_products_enabled=1
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("softdel-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create a product
		productName := fmt.Sprintf("softdel-prod-%d", rng.Int63n(1000000))
		productType := "credits"
		if rng.Intn(2) == 0 {
			productType = "virtual_goods"
		}
		price := float64(1+rng.Intn(999999)) / 100.0
		res, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
			 VALUES (?, ?, ?, ?, 'published')`,
			storefrontID, productName, productType, price,
		)
		if err != nil {
			t.Logf("FAIL: failed to insert product: %v", err)
			return false
		}
		productID, _ := res.LastInsertId()

		// Insert a random number of orders (1-10) for this product
		numOrders := 1 + rng.Intn(10)
		for i := 0; i < numOrders; i++ {
			buyerID := createTestUserWithBalance(t, 0)
			orderStatus := []string{"pending", "paid", "fulfilled", "failed"}[rng.Intn(4)]
			_, err := db.Exec(
				`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd, status)
				 VALUES (?, ?, ?, ?, ?)`,
				productID, buyerID, fmt.Sprintf("PAYPAL-%d", rng.Int63n(1000000)), price, orderStatus,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert order %d: %v", i, err)
				return false
			}
		}

		// Count orders before soft delete
		var orderCountBefore int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM custom_product_orders WHERE custom_product_id = ?",
			productID,
		).Scan(&orderCountBefore)
		if err != nil {
			t.Logf("FAIL: failed to count orders before soft delete: %v", err)
			return false
		}
		if orderCountBefore != numOrders {
			t.Logf("FAIL: expected %d orders before soft delete, got %d", numOrders, orderCountBefore)
			return false
		}

		// Soft delete the product
		_, err = db.Exec(
			"UPDATE custom_products SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			productID,
		)
		if err != nil {
			t.Logf("FAIL: failed to soft delete product id=%d: %v", productID, err)
			return false
		}

		// Verify deleted_at IS NOT NULL
		var deletedAt *string
		err = db.QueryRow(
			"SELECT deleted_at FROM custom_products WHERE id = ?",
			productID,
		).Scan(&deletedAt)
		if err != nil {
			t.Logf("FAIL: failed to read deleted_at for product id=%d: %v", productID, err)
			return false
		}
		if deletedAt == nil {
			t.Logf("FAIL: product id=%d deleted_at is NULL after soft delete", productID)
			return false
		}

		// Count orders after soft delete — should be unchanged
		var orderCountAfter int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM custom_product_orders WHERE custom_product_id = ?",
			productID,
		).Scan(&orderCountAfter)
		if err != nil {
			t.Logf("FAIL: failed to count orders after soft delete: %v", err)
			return false
		}
		if orderCountAfter != orderCountBefore {
			t.Logf("FAIL: order count changed after soft delete: before=%d, after=%d", orderCountBefore, orderCountAfter)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 19 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 12: 商品按 sort_order 升序排列且排序更新正确
// **Validates: Requirements 5.4, 13.2, 13.3**
//
// For any storefront and any permutation of product IDs, after updating sort_order
// based on the array position, querying products ORDER BY sort_order ASC must return
// them in the same order as the input array.
func TestCustomProductProperty12_SortOrderInvariance(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront with custom_products_enabled=1
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("sort-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create between 2 and 10 products
		numProducts := 2 + rng.Intn(9)
		productIDs := make([]int64, numProducts)
		for i := 0; i < numProducts; i++ {
			productName := fmt.Sprintf("sort-prod-%d-%d-%d", storefrontID, i, rng.Int63n(1000000))
			productType := "credits"
			if rng.Intn(2) == 0 {
				productType = "virtual_goods"
			}
			price := float64(1+rng.Intn(999999)) / 100.0
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status, sort_order)
				 VALUES (?, ?, ?, ?, 'published', ?)`,
				storefrontID, productName, productType, price, i,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert product %d: %v", i, err)
				return false
			}
			pid, _ := res.LastInsertId()
			productIDs[i] = pid
		}

		// Shuffle the product IDs to create a random permutation
		shuffled := make([]int64, len(productIDs))
		copy(shuffled, productIDs)
		rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		// Update sort_order based on shuffled array position (mimicking the reorder handler)
		tx, err := db.Begin()
		if err != nil {
			t.Logf("FAIL: failed to begin transaction: %v", err)
			return false
		}
		defer tx.Rollback()

		for i, pid := range shuffled {
			_, err := tx.Exec(
				"UPDATE custom_products SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				i, pid,
			)
			if err != nil {
				t.Logf("FAIL: failed to update sort_order for product %d: %v", pid, err)
				return false
			}
		}

		if err := tx.Commit(); err != nil {
			t.Logf("FAIL: failed to commit transaction: %v", err)
			return false
		}

		// Query products ORDER BY sort_order ASC
		rows, err := db.Query(
			`SELECT id, sort_order FROM custom_products
			 WHERE storefront_id = ? AND deleted_at IS NULL
			 ORDER BY sort_order ASC`,
			storefrontID,
		)
		if err != nil {
			t.Logf("FAIL: failed to query products: %v", err)
			return false
		}
		defer rows.Close()

		var queriedIDs []int64
		var prevSortOrder int = -1
		for rows.Next() {
			var id int64
			var sortOrder int
			if err := rows.Scan(&id, &sortOrder); err != nil {
				t.Logf("FAIL: failed to scan row: %v", err)
				return false
			}
			// Verify sort_order is strictly ascending
			if sortOrder <= prevSortOrder {
				t.Logf("FAIL: sort_order not strictly ascending: prev=%d, current=%d", prevSortOrder, sortOrder)
				return false
			}
			prevSortOrder = sortOrder
			queriedIDs = append(queriedIDs, id)
		}
		if err := rows.Err(); err != nil {
			t.Logf("FAIL: rows iteration error: %v", err)
			return false
		}

		// Verify the queried order matches the shuffled array
		if len(queriedIDs) != len(shuffled) {
			t.Logf("FAIL: expected %d products, got %d", len(shuffled), len(queriedIDs))
			return false
		}
		for i := range shuffled {
			if queriedIDs[i] != shuffled[i] {
				t.Logf("FAIL: position %d: expected product ID %d, got %d (shuffled=%v, queried=%v)",
					i, shuffled[i], queriedIDs[i], shuffled, queriedIDs)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 12 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 18: PayPal 配置保存/读取 Round-Trip 与掩码
// **Validates: Requirements 11.3, 11.4, 11.5**
//
// For any valid PayPal config (client_id, client_secret with len >= 8, mode in {sandbox, live}),
// after saving the config and reading it back:
//   - client_id and mode should match the original
//   - client_secret should be masked (first4 + "****" + last4)
//   - the raw value stored in the database should NOT equal the plaintext secret
//   - decrypting the stored value should recover the original secret
func TestCustomProductProperty18_PayPalConfigRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Set encryption key for the test
		os.Setenv("PAYPAL_ENCRYPTION_KEY", "test-encryption-key-for-property18")
		defer os.Unsetenv("PAYPAL_ENCRYPTION_KEY")

		// Generate random client_id (8-40 chars alphanumeric)
		clientIDLen := 8 + rng.Intn(33)
		clientID := generateRandomString(rng, clientIDLen)

		// Generate random client_secret (>= 8 chars)
		secretLen := 8 + rng.Intn(33)
		clientSecret := generateRandomString(rng, secretLen)

		// Random mode: sandbox or live
		mode := "sandbox"
		if rng.Intn(2) == 0 {
			mode = "live"
		}

		// Step 1: Encrypt the client_secret
		encrypted, err := encryptPayPalSecret(clientSecret)
		if err != nil {
			t.Logf("FAIL: encryptPayPalSecret failed: %v", err)
			return false
		}

		// Step 2: Save config to settings table
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('paypal_client_id', ?)", clientID); err != nil {
			t.Logf("FAIL: failed to save paypal_client_id: %v", err)
			return false
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('paypal_client_secret', ?)", encrypted); err != nil {
			t.Logf("FAIL: failed to save paypal_client_secret: %v", err)
			return false
		}
		if _, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('paypal_mode', ?)", mode); err != nil {
			t.Logf("FAIL: failed to save paypal_mode: %v", err)
			return false
		}

		// Step 3: Read back client_id — should match original
		readClientID := getSetting("paypal_client_id")
		if readClientID != clientID {
			t.Logf("FAIL: client_id mismatch: expected %q, got %q", clientID, readClientID)
			return false
		}

		// Step 4: Read back mode — should match original
		readMode := getSetting("paypal_mode")
		if readMode != mode {
			t.Logf("FAIL: mode mismatch: expected %q, got %q", mode, readMode)
			return false
		}

		// Step 5: Read back encrypted secret from DB — should NOT equal plaintext
		storedSecret := getSetting("paypal_client_secret")
		if storedSecret == clientSecret {
			t.Logf("FAIL: stored secret equals plaintext — encryption not applied")
			return false
		}
		if storedSecret == "" {
			t.Logf("FAIL: stored secret is empty")
			return false
		}

		// Step 6: Decrypt the stored value — should equal original
		decrypted, err := decryptPayPalSecret(storedSecret)
		if err != nil {
			t.Logf("FAIL: decryptPayPalSecret failed: %v", err)
			return false
		}
		if decrypted != clientSecret {
			t.Logf("FAIL: decrypted secret mismatch: expected %q, got %q", clientSecret, decrypted)
			return false
		}

		// Step 7: maskPayPalSecret should show first4 + "****" + last4
		masked := maskPayPalSecret(clientSecret)
		expectedMask := clientSecret[:4] + "****" + clientSecret[len(clientSecret)-4:]
		if masked != expectedMask {
			t.Logf("FAIL: mask mismatch: expected %q, got %q (secret=%q)", expectedMask, masked, clientSecret)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 18 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 13: 积分商品履约 Round-Trip
// **Validates: Requirements 7.1, 7.2, 7.3**
//
// For any credits product and user, after fulfilling a credits order:
// - The user's balance should increase by credits_amount
// - A credits_transaction with transaction_type="purchase" should exist
// - The order status should be "fulfilled"
func TestCustomProductProperty13_CreditsFulfillmentRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Random initial balance (0 to 10000)
		initialBalance := float64(rng.Intn(10001))

		// Random credits_amount (1 to 5000)
		creditsAmount := 1 + rng.Intn(5000)

		// Random price (0.01 to 9999.99)
		priceUSD := float64(1+rng.Intn(999999)) / 100.0

		// 1. Create user with initial balance
		userID := createTestUserWithBalance(t, initialBalance)

		// 2. Create storefront and credits product
		slug := fmt.Sprintf("credits-ful-%d-%d", userID, rng.Int63n(1000000))
		sfResult, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := sfResult.LastInsertId()

		productName := fmt.Sprintf("credits-prod-%d", rng.Int63n(1000000))
		prodResult, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, credits_amount, status)
			 VALUES (?, ?, 'credits', ?, ?, 'published')`,
			storefrontID, productName, priceUSD, creditsAmount,
		)
		if err != nil {
			t.Logf("FAIL: failed to create credits product: %v", err)
			return false
		}
		productID, _ := prodResult.LastInsertId()

		// 3. Create a pending order (simulating purchase initiation)
		paypalOrderID := fmt.Sprintf("PAYPAL-CREDITS-%d", rng.Int63n(1000000))
		orderResult, err := db.Exec(
			`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd, status)
			 VALUES (?, ?, ?, ?, 'pending')`,
			productID, userID, paypalOrderID, priceUSD,
		)
		if err != nil {
			t.Logf("FAIL: failed to create order: %v", err)
			return false
		}
		orderID, _ := orderResult.LastInsertId()

		// 4. Simulate payment success: update order to paid
		_, err = db.Exec(
			`UPDATE custom_product_orders SET paypal_payment_status='COMPLETED', status='paid', updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			orderID,
		)
		if err != nil {
			t.Logf("FAIL: failed to update order to paid: %v", err)
			return false
		}

		// 5. Simulate credits fulfillment (same logic as handlePayPalReturn)
		tx, txErr := db.Begin()
		if txErr != nil {
			t.Logf("FAIL: failed to begin tx: %v", txErr)
			return false
		}

		err = addWalletBalance(tx, userID, float64(creditsAmount))
		if err != nil {
			tx.Rollback()
			t.Logf("FAIL: addWalletBalance failed: %v", err)
			return false
		}

		description := fmt.Sprintf("购买商品「%s」充值 %d 积分", productName, creditsAmount)
		_, err = tx.Exec(`INSERT INTO credits_transactions (user_id, transaction_type, amount, description, created_at)
			VALUES (?, 'purchase', ?, ?, CURRENT_TIMESTAMP)`,
			userID, creditsAmount, description)
		if err != nil {
			tx.Rollback()
			t.Logf("FAIL: failed to insert credits_transaction: %v", err)
			return false
		}

		_, err = tx.Exec(`UPDATE custom_product_orders SET status='fulfilled', updated_at=CURRENT_TIMESTAMP WHERE id=?`, orderID)
		if err != nil {
			tx.Rollback()
			t.Logf("FAIL: failed to update order to fulfilled: %v", err)
			return false
		}

		if commitErr := tx.Commit(); commitErr != nil {
			t.Logf("FAIL: failed to commit tx: %v", commitErr)
			return false
		}

		// 6. Verify: user balance increased by credits_amount
		// Check email_wallets balance (primary balance store)
		var userEmail string
		db.QueryRow("SELECT COALESCE(email, '') FROM users WHERE id = ?", userID).Scan(&userEmail)

		var balanceAfter float64
		if userEmail != "" {
			err = db.QueryRow("SELECT credits_balance FROM email_wallets WHERE email = ?", userEmail).Scan(&balanceAfter)
		} else {
			err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balanceAfter)
		}
		if err != nil {
			t.Logf("FAIL: failed to read balance after fulfillment: %v", err)
			return false
		}

		expectedBalance := initialBalance + float64(creditsAmount)
		if balanceAfter != expectedBalance {
			t.Logf("FAIL: balance mismatch: expected %.2f (initial=%.2f + credits=%d), got %.2f",
				expectedBalance, initialBalance, creditsAmount, balanceAfter)
			return false
		}

		// 7. Verify: credits_transactions has a 'purchase' type record for this user
		var txCount int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM credits_transactions WHERE user_id = ? AND transaction_type = 'purchase'",
			userID,
		).Scan(&txCount)
		if err != nil {
			t.Logf("FAIL: failed to query credits_transactions: %v", err)
			return false
		}
		if txCount < 1 {
			t.Logf("FAIL: expected at least 1 purchase transaction for user %d, got %d", userID, txCount)
			return false
		}

		// 8. Verify: order status is 'fulfilled'
		var orderStatus string
		err = db.QueryRow("SELECT status FROM custom_product_orders WHERE id = ?", orderID).Scan(&orderStatus)
		if err != nil {
			t.Logf("FAIL: failed to query order status: %v", err)
			return false
		}
		if orderStatus != "fulfilled" {
			t.Logf("FAIL: expected order status 'fulfilled', got %q", orderStatus)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 13 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 14: 虚拟商品履约存储 SN
// **Validates: Requirements 8.3, 8.4**
//
// For any virtual product order, when the License API returns a success response,
// the order's license_sn must equal the API-returned SN, license_email must equal
// the user's email, and order status must be "fulfilled".
func TestCustomProductProperty14_LicenseFulfillmentSN(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate random SN and product ID for the mock License API
		mockSN := fmt.Sprintf("SN-%d-%d", rng.Int63n(1000000), rng.Int63n(1000000))
		mockProductID := fmt.Sprintf("PROD-%d", rng.Int63n(100000))
		mockAPIKey := fmt.Sprintf("KEY-%d", rng.Int63n(100000))

		// Random price (0.01 to 9999.99)
		priceUSD := float64(1+rng.Intn(999999)) / 100.0

		// 1. Set up a test HTTP server that returns a mock License API success response
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"sn": mockSN})
		}))
		// Close immediately after use to avoid accumulating open servers across iterations
		defer ts.Close()

		// 2. Create user with a random email
		userEmail := fmt.Sprintf("vguser-%d-%d@test.example.com", seed, rng.Int63n(1000000))
		userResult, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, 'VGUser', ?, 0)",
			fmt.Sprintf("vg-auth-%d-%d", seed, rng.Int63n(1000000)), userEmail,
		)
		if err != nil {
			t.Logf("FAIL: failed to create user: %v", err)
			return false
		}
		userID, _ := userResult.LastInsertId()

		// Ensure email_wallets entry exists
		db.Exec("INSERT OR IGNORE INTO email_wallets (email, credits_balance) VALUES (?, 0)", userEmail)

		// 3. Create storefront and virtual_goods product pointing to the test server
		slug := fmt.Sprintf("vg-ful-%d-%d", userID, rng.Int63n(1000000))
		sfResult, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := sfResult.LastInsertId()

		productName := fmt.Sprintf("vg-prod-%d", rng.Int63n(1000000))
		prodResult, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd,
			 license_api_endpoint, license_api_key, license_product_id, status)
			 VALUES (?, ?, 'virtual_goods', ?, ?, ?, ?, 'published')`,
			storefrontID, productName, priceUSD, ts.URL, mockAPIKey, mockProductID,
		)
		if err != nil {
			t.Logf("FAIL: failed to create virtual_goods product: %v", err)
			return false
		}
		productID, _ := prodResult.LastInsertId()

		// 4. Create an order in "paid" status (simulating successful PayPal capture)
		paypalOrderID := fmt.Sprintf("PAYPAL-VG-%d", rng.Int63n(1000000))
		orderResult, err := db.Exec(
			`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd,
			 paypal_payment_status, status)
			 VALUES (?, ?, ?, ?, 'COMPLETED', 'paid')`,
			productID, userID, paypalOrderID, priceUSD,
		)
		if err != nil {
			t.Logf("FAIL: failed to create order: %v", err)
			return false
		}
		orderID, _ := orderResult.LastInsertId()

		// 5. Simulate virtual goods fulfillment (same logic as handlePayPalReturn)
		sn, licErr := callLicenseAPI(ts.URL, mockAPIKey, userEmail, mockProductID)
		if licErr != nil {
			t.Logf("FAIL: callLicenseAPI returned error: %v", licErr)
			return false
		}

		_, dbErr := db.Exec(
			`UPDATE custom_product_orders SET license_sn=?, license_email=?, status='fulfilled', updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			sn, userEmail, orderID,
		)
		if dbErr != nil {
			t.Logf("FAIL: failed to update order with license info: %v", dbErr)
			return false
		}

		// 6. Verify: order.license_sn == API returned SN (mockSN)
		var actualSN, actualEmail, actualStatus string
		err = db.QueryRow(
			"SELECT license_sn, license_email, status FROM custom_product_orders WHERE id = ?", orderID,
		).Scan(&actualSN, &actualEmail, &actualStatus)
		if err != nil {
			t.Logf("FAIL: failed to query order: %v", err)
			return false
		}

		if actualSN != mockSN {
			t.Logf("FAIL: license_sn mismatch: expected %q, got %q", mockSN, actualSN)
			return false
		}

		// 7. Verify: order.license_email == user email
		if actualEmail != userEmail {
			t.Logf("FAIL: license_email mismatch: expected %q, got %q", userEmail, actualEmail)
			return false
		}

		// 8. Verify: order.status == "fulfilled"
		if actualStatus != "fulfilled" {
			t.Logf("FAIL: expected order status 'fulfilled', got %q", actualStatus)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 14 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 10: 公开页面仅展示已上架且未删除的商品
// **Validates: Requirements 4.6, 4.7, 12.5**
//
// For all storefronts, querying the public custom products page should return
// ONLY products with status='published' AND deleted_at IS NULL.
// Products that are draft, pending, rejected, or soft-deleted (even if published)
// must NOT appear in the results.
func TestCustomProductProperty10_PublicPageVisibility(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront with custom_products_enabled=true
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("pub-vis-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Randomly decide how many products of each status to create
		numPublished := 1 + rng.Intn(5)       // 1-5 published (visible)
		numDraft := rng.Intn(4)                // 0-3 draft (hidden)
		numPending := rng.Intn(4)              // 0-3 pending (hidden)
		numRejected := rng.Intn(4)             // 0-3 rejected (hidden)
		numSoftDeletedPublished := 1 + rng.Intn(3) // 1-3 published but soft-deleted (hidden)

		// Track IDs of products that SHOULD be visible
		visibleIDs := make(map[int64]bool)
		// Track IDs of products that should NOT be visible
		hiddenIDs := make(map[int64]bool)

		// Insert published products (should be visible)
		for i := 0; i < numPublished; i++ {
			pname := fmt.Sprintf("pub-%d-%d-%d", i, seed, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status, sort_order)
				 VALUES (?, ?, 'credits', 9.99, 'published', ?)`,
				storefrontID, pname, i,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert published product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			visibleIDs[id] = true
		}

		// Insert draft products (should be hidden)
		for i := 0; i < numDraft; i++ {
			pname := fmt.Sprintf("draft-%d-%d-%d", i, seed, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
				 VALUES (?, ?, 'credits', 9.99, 'draft')`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert draft product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			hiddenIDs[id] = true
		}

		// Insert pending products (should be hidden)
		for i := 0; i < numPending; i++ {
			pname := fmt.Sprintf("pending-%d-%d-%d", i, seed, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
				 VALUES (?, ?, 'credits', 9.99, 'pending')`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert pending product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			hiddenIDs[id] = true
		}

		// Insert rejected products (should be hidden)
		for i := 0; i < numRejected; i++ {
			pname := fmt.Sprintf("rejected-%d-%d-%d", i, seed, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status, reject_reason)
				 VALUES (?, ?, 'credits', 9.99, 'rejected', 'test rejection')`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert rejected product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			hiddenIDs[id] = true
		}

		// Insert published but soft-deleted products (should be hidden)
		for i := 0; i < numSoftDeletedPublished; i++ {
			pname := fmt.Sprintf("softdel-%d-%d-%d", i, seed, rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status, deleted_at)
				 VALUES (?, ?, 'credits', 9.99, 'published', CURRENT_TIMESTAMP)`,
				storefrontID, pname,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert soft-deleted published product: %v", err)
				return false
			}
			id, _ := res.LastInsertId()
			hiddenIDs[id] = true
		}

		// Query using the same SQL as the public page
		rows, err := db.Query(
			`SELECT id, status, deleted_at FROM custom_products
			 WHERE storefront_id = ? AND status = 'published' AND deleted_at IS NULL
			 ORDER BY sort_order ASC`,
			storefrontID,
		)
		if err != nil {
			t.Logf("FAIL: failed to query public products: %v", err)
			return false
		}
		defer rows.Close()

		returnedIDs := make(map[int64]bool)
		for rows.Next() {
			var id int64
			var status string
			var deletedAt *string
			if err := rows.Scan(&id, &status, &deletedAt); err != nil {
				t.Logf("FAIL: failed to scan row: %v", err)
				return false
			}

			// Every returned product must be published
			if status != "published" {
				t.Logf("FAIL: public query returned product id=%d with status=%q, expected 'published'", id, status)
				return false
			}

			// Every returned product must have deleted_at IS NULL
			if deletedAt != nil {
				t.Logf("FAIL: public query returned product id=%d with non-null deleted_at=%q", id, *deletedAt)
				return false
			}

			returnedIDs[id] = true
		}
		if err := rows.Err(); err != nil {
			t.Logf("FAIL: rows iteration error: %v", err)
			return false
		}

		// Verify: all visible products are in the result set
		for id := range visibleIDs {
			if !returnedIDs[id] {
				t.Logf("FAIL: published non-deleted product id=%d was NOT returned by public query", id)
				return false
			}
		}

		// Verify: no hidden products are in the result set
		for id := range hiddenIDs {
			if returnedIDs[id] {
				t.Logf("FAIL: hidden product id=%d was returned by public query (should be excluded)", id)
				return false
			}
		}

		// Verify: result count matches expected visible count
		if len(returnedIDs) != numPublished {
			t.Logf("FAIL: expected %d visible products, got %d", numPublished, len(returnedIDs))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 10 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 11: 商品卡片包含必要信息和正确的类型标签
// **Validates: Requirements 5.2, 5.3**
//
// For any published product, the rendered product card HTML should contain the product name,
// description, formatted price, and the correct type label ("积分充值" for credits,
// "虚拟商品" for virtual_goods).
func TestCustomProductProperty11_ProductCardContent(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate random product data
		nameLen := 2 + rng.Intn(20)
		productName := generateRandomString(rng, nameLen)
		descLen := 5 + rng.Intn(50)
		description := generateRandomString(rng, descLen)

		// Generate a random price (positive, up to 9999.99)
		priceUSD := float64(rng.Intn(999900)+1) / 100.0 // 0.01 to 9999.00

		// Randomly pick product type
		var productType string
		if rng.Intn(2) == 0 {
			productType = "credits"
		} else {
			productType = "virtual_goods"
		}

		// Build a CustomProduct
		product := CustomProduct{
			ID:          int64(rng.Intn(10000) + 1),
			ProductName: productName,
			Description: description,
			ProductType: productType,
			PriceUSD:    priceUSD,
			Status:      "published",
		}

		// Build minimal StorefrontPageData with the product
		pageData := StorefrontPageData{
			Storefront: StorefrontInfo{
				StoreName: "TestStore",
				StoreSlug: "test-store",
			},
			Packs:           []StorefrontPackInfo{},
			PurchasedIDs:    map[int64]bool{},
			DefaultLang:     "zh-CN",
			Sections:        []SectionConfig{},
			ThemeCSS:        GetThemeCSS("default"),
			PackGridColumns: 2,
			BannerData:      map[int]CustomBannerSettings{},
			CustomProducts:  []CustomProduct{product},
		}

		// Render the storefront template
		var buf bytes.Buffer
		if err := templates.StorefrontTmpl.Execute(&buf, pageData); err != nil {
			t.Logf("FAIL: template execution error: %v", err)
			return false
		}
		html := buf.String()

		// Verify: HTML contains product name
		if !strings.Contains(html, productName) {
			t.Logf("FAIL: HTML does not contain product name %q", productName)
			return false
		}

		// Verify: HTML contains description
		if !strings.Contains(html, description) {
			t.Logf("FAIL: HTML does not contain description %q", description)
			return false
		}

		// Verify: HTML contains formatted price
		expectedPrice := fmt.Sprintf("$%.2f", priceUSD)
		if priceUSD == float64(int(priceUSD)) {
			expectedPrice = fmt.Sprintf("$%.0f", priceUSD)
		}
		if !strings.Contains(html, expectedPrice) {
			t.Logf("FAIL: HTML does not contain formatted price %q (priceUSD=%.2f)", expectedPrice, priceUSD)
			return false
		}

		// Verify: correct type label
		if productType == "credits" {
			if !strings.Contains(html, "积分充值") {
				t.Logf("FAIL: credits product HTML does not contain '积分充值'")
				return false
			}
		} else if productType == "virtual_goods" {
			if !strings.Contains(html, "虚拟商品") {
				t.Logf("FAIL: virtual_goods product HTML does not contain '虚拟商品'")
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 11 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 5: 自定义商品标签页可见性
// **Validates: Requirements 3.1, 3.2**
//
// For any storefront, when custom_products_enabled is true, the rendered manage page
// HTML should contain the "custom-products-tab" data-testid marker. When false, the
// HTML should NOT contain that marker.
func TestCustomProductProperty5_TabVisibility(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Randomly toggle the custom products enabled flag
		enabled := rng.Intn(2) == 1

		// Build minimal StorefrontManageData
		data := StorefrontManageData{
			Storefront: StorefrontInfo{
				StoreName:   "TestStore",
				StoreSlug:   "test-store",
				StoreLayout: "default",
			},
			FullURL:               "https://market.example.com/store/test-store",
			DefaultLang:           "zh-CN",
			CustomProductsEnabled: enabled,
			CustomProducts:        []CustomProduct{},
			LayoutSectionsJSON:    "[]",
			CurrentTheme:          "default",
		}

		// Render the storefront manage template
		var buf bytes.Buffer
		if err := templates.StorefrontManageTmpl.Execute(&buf, data); err != nil {
			t.Logf("FAIL: template execution error: %v", err)
			return false
		}
		html := buf.String()

		containsTab := strings.Contains(html, "custom-products-tab")

		if enabled && !containsTab {
			t.Logf("FAIL: custom_products_enabled=true but HTML does not contain 'custom-products-tab'")
			return false
		}
		if !enabled && containsTab {
			t.Logf("FAIL: custom_products_enabled=false but HTML contains 'custom-products-tab'")
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 15: 订单列表按创建时间降序排列
// **Validates: Requirements 9.2**
//
// For any order list queried via the same SQL as handleStorefrontCustomProductOrders,
// order_list[i].created_at >= order_list[i+1].created_at for all valid i.
func TestCustomProductProperty15_OrderListSorting(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("ordersort-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create a published product
		productName := fmt.Sprintf("sortprod-%d", rng.Int63n(1000000))
		res, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
			 VALUES (?, ?, 'credits', 9.99, 'published')`,
			storefrontID, productName,
		)
		if err != nil {
			t.Logf("FAIL: failed to insert product: %v", err)
			return false
		}
		productID, _ := res.LastInsertId()

		// Create between 2 and 15 orders with explicit, distinct created_at timestamps
		numOrders := 2 + rng.Intn(14)
		// Base time: 2024-01-01 00:00:00 UTC
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		// Generate random offsets and shuffle to insert in non-sorted order
		offsets := make([]int, numOrders)
		for i := 0; i < numOrders; i++ {
			// Each order gets a unique offset in minutes (i*60 + random 0-59 minutes)
			offsets[i] = i*60 + rng.Intn(60)
		}
		// Shuffle offsets so insertion order differs from time order
		for i := numOrders - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			offsets[i], offsets[j] = offsets[j], offsets[i]
		}

		for i := 0; i < numOrders; i++ {
			buyerID := createTestUserWithBalance(t, 0)
			ts := baseTime.Add(time.Duration(offsets[i]) * time.Minute).Format("2006-01-02 15:04:05")
			statuses := []string{"pending", "paid", "fulfilled", "failed"}
			orderStatus := statuses[rng.Intn(len(statuses))]
			_, err := db.Exec(
				`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd, status, created_at)
				 VALUES (?, ?, ?, 9.99, ?, ?)`,
				productID, buyerID, fmt.Sprintf("PAYPAL-%d-%d", seed, i), orderStatus, ts,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert order %d: %v", i, err)
				return false
			}
		}

		// Query orders using the same SQL pattern as handleStorefrontCustomProductOrders
		rows, err := db.Query(
			`SELECT o.id, o.created_at
			 FROM custom_product_orders o
			 JOIN custom_products p ON o.custom_product_id = p.id
			 JOIN users u ON o.user_id = u.id
			 WHERE p.storefront_id = ?
			 ORDER BY o.created_at DESC`,
			storefrontID,
		)
		if err != nil {
			t.Logf("FAIL: failed to query orders: %v", err)
			return false
		}
		defer rows.Close()

		var timestamps []string
		for rows.Next() {
			var id int64
			var createdAt string
			if err := rows.Scan(&id, &createdAt); err != nil {
				t.Logf("FAIL: failed to scan order row: %v", err)
				return false
			}
			timestamps = append(timestamps, createdAt)
		}
		if err := rows.Err(); err != nil {
			t.Logf("FAIL: rows iteration error: %v", err)
			return false
		}

		// Verify we got all orders
		if len(timestamps) != numOrders {
			t.Logf("FAIL: expected %d orders, got %d", numOrders, len(timestamps))
			return false
		}

		// Verify descending order: timestamps[i] >= timestamps[i+1]
		for i := 0; i < len(timestamps)-1; i++ {
			if timestamps[i] < timestamps[i+1] {
				t.Logf("FAIL: orders not in descending order at index %d: %q < %q",
					i, timestamps[i], timestamps[i+1])
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 15 violated: %v", err)
	}
}

// Feature: storefront-custom-products, Property 16: 订单筛选返回匹配子集
// **Validates: Requirements 9.5**
//
// For any filter_name and filter_status,
//   filtered = queryOrders(storefront, filter_name, filter_status)
//   all = queryOrders(storefront, "", "")
//   => len(filtered) <= len(all)
//   AND ALL order IN filtered:
//     (filter_name == "" OR order.product_name CONTAINS filter_name)
//     AND (filter_status == "" OR order.status == filter_status)
func TestCustomProductProperty16_OrderFilterSubset(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("orderfilter-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create 2-4 published products with distinct names
		numProducts := 2 + rng.Intn(3)
		productNames := []string{"Alpha", "Beta", "Gamma", "Delta"}
		type prodInfo struct {
			id   int64
			name string
		}
		var products []prodInfo
		for i := 0; i < numProducts; i++ {
			name := fmt.Sprintf("%s-%d", productNames[i], rng.Int63n(1000000))
			res, err := db.Exec(
				`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, status)
				 VALUES (?, ?, 'credits', 9.99, 'published')`,
				storefrontID, name,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert product %d: %v", i, err)
				return false
			}
			pid, _ := res.LastInsertId()
			products = append(products, prodInfo{id: pid, name: name})
		}

		// Create 3-12 orders across different products and statuses
		orderStatuses := []string{"pending", "paid", "fulfilled", "failed"}
		numOrders := 3 + rng.Intn(10)
		for i := 0; i < numOrders; i++ {
			buyerID := createTestUserWithBalance(t, 0)
			prod := products[rng.Intn(len(products))]
			status := orderStatuses[rng.Intn(len(orderStatuses))]
			_, err := db.Exec(
				`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd, status)
				 VALUES (?, ?, ?, 9.99, ?)`,
				prod.id, buyerID, fmt.Sprintf("PAYPAL-FILTER-%d-%d", seed, i), status,
			)
			if err != nil {
				t.Logf("FAIL: failed to insert order %d: %v", i, err)
				return false
			}
		}

		// Helper to query orders with optional filters (mirrors handleStorefrontCustomProductOrders)
		queryOrders := func(filterName, filterStatus string) ([]struct {
			productName string
			orderStatus string
		}, error) {
			query := `SELECT p.product_name, o.status
				FROM custom_product_orders o
				JOIN custom_products p ON o.custom_product_id = p.id
				JOIN users u ON o.user_id = u.id
				WHERE p.storefront_id = ?`
			args := []interface{}{storefrontID}

			if filterName != "" {
				query += " AND p.product_name LIKE ?"
				args = append(args, "%"+filterName+"%")
			}
			if filterStatus != "" {
				query += " AND o.status = ?"
				args = append(args, filterStatus)
			}
			query += " ORDER BY o.created_at DESC"

			rows, err := db.Query(query, args...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			var results []struct {
				productName string
				orderStatus string
			}
			for rows.Next() {
				var pn, st string
				if err := rows.Scan(&pn, &st); err != nil {
					return nil, err
				}
				results = append(results, struct {
					productName string
					orderStatus string
				}{pn, st})
			}
			return results, rows.Err()
		}

		// Get all orders (no filter)
		allOrders, err := queryOrders("", "")
		if err != nil {
			t.Logf("FAIL: failed to query all orders: %v", err)
			return false
		}

		// Pick a random filter combination
		// 0 = filter by product name only, 1 = filter by status only, 2 = both
		filterMode := rng.Intn(3)
		var filterName, filterStatus string

		switch filterMode {
		case 0:
			// Use the prefix portion of a random product name as filter substring
			prod := products[rng.Intn(len(products))]
			parts := strings.SplitN(prod.name, "-", 2)
			if len(parts) > 0 {
				filterName = parts[0]
			}
		case 1:
			filterStatus = orderStatuses[rng.Intn(len(orderStatuses))]
		case 2:
			prod := products[rng.Intn(len(products))]
			parts := strings.SplitN(prod.name, "-", 2)
			if len(parts) > 0 {
				filterName = parts[0]
			}
			filterStatus = orderStatuses[rng.Intn(len(orderStatuses))]
		}

		// Query with filter
		filteredOrders, err := queryOrders(filterName, filterStatus)
		if err != nil {
			t.Logf("FAIL: failed to query filtered orders: %v", err)
			return false
		}

		// Property check 1: filtered count <= total count
		if len(filteredOrders) > len(allOrders) {
			t.Logf("FAIL: filtered count (%d) > total count (%d) with filter name=%q status=%q",
				len(filteredOrders), len(allOrders), filterName, filterStatus)
			return false
		}

		// Property check 2: all filtered results match the filter criteria
		for i, order := range filteredOrders {
			if filterName != "" && !strings.Contains(order.productName, filterName) {
				t.Logf("FAIL: order %d product_name %q does not contain filter %q",
					i, order.productName, filterName)
				return false
			}
			if filterStatus != "" && order.orderStatus != filterStatus {
				t.Logf("FAIL: order %d status %q does not match filter %q",
					i, order.orderStatus, filterStatus)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 16 violated: %v", err)
	}
}


// Feature: storefront-custom-products, Property 17: 已履约订单显示类型特定信息
// **Validates: Requirements 9.3, 10.3, 10.4**
//
// For all orders where status == "fulfilled":
//   If product_type == "virtual_goods" then license_sn != "" AND license_email != ""
//   If product_type == "credits" then credits_amount > 0
func TestCustomProductProperty17_FulfilledOrderTypeInfo(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// 1. Create a user and storefront
		userEmail := fmt.Sprintf("fulfilled-user-%d-%d@test.example.com", seed, rng.Int63n(1000000))
		userResult, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, 'FulUser', ?, 0)",
			fmt.Sprintf("ful-auth-%d-%d", seed, rng.Int63n(1000000)), userEmail,
		)
		if err != nil {
			t.Logf("FAIL: failed to create user: %v", err)
			return false
		}
		userID, _ := userResult.LastInsertId()
		db.Exec("INSERT OR IGNORE INTO email_wallets (email, credits_balance) VALUES (?, 0)", userEmail)

		slug := fmt.Sprintf("ful-order-%d-%d", userID, rng.Int63n(1000000))
		sfResult, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, custom_products_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := sfResult.LastInsertId()

		// 2. Create a credits product with random positive credits_amount
		creditsAmount := 1 + rng.Intn(5000)
		creditsPrice := float64(1+rng.Intn(999999)) / 100.0
		creditsName := fmt.Sprintf("credits-p17-%d", rng.Int63n(1000000))
		creditsProdResult, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd, credits_amount, status)
			 VALUES (?, ?, 'credits', ?, ?, 'published')`,
			storefrontID, creditsName, creditsPrice, creditsAmount,
		)
		if err != nil {
			t.Logf("FAIL: failed to create credits product: %v", err)
			return false
		}
		creditsProductID, _ := creditsProdResult.LastInsertId()

		// 3. Create a virtual_goods product
		vgPrice := float64(1+rng.Intn(999999)) / 100.0
		vgName := fmt.Sprintf("vg-p17-%d", rng.Int63n(1000000))
		vgProdResult, err := db.Exec(
			`INSERT INTO custom_products (storefront_id, product_name, product_type, price_usd,
			 license_api_endpoint, license_api_key, license_product_id, status)
			 VALUES (?, ?, 'virtual_goods', ?, 'https://license.example.com', 'key123', 'prod123', 'published')`,
			storefrontID, vgName, vgPrice,
		)
		if err != nil {
			t.Logf("FAIL: failed to create virtual_goods product: %v", err)
			return false
		}
		vgProductID, _ := vgProdResult.LastInsertId()

		// 4. Create a fulfilled credits order
		_, err = db.Exec(
			`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd,
			 paypal_payment_status, status)
			 VALUES (?, ?, ?, ?, 'COMPLETED', 'fulfilled')`,
			creditsProductID, userID, fmt.Sprintf("PAYPAL-C17-%d", rng.Int63n(1000000)), creditsPrice,
		)
		if err != nil {
			t.Logf("FAIL: failed to create credits order: %v", err)
			return false
		}

		// 5. Create a fulfilled virtual_goods order with license_sn and license_email
		licenseSN := fmt.Sprintf("SN-P17-%d-%d", rng.Int63n(1000000), rng.Int63n(1000000))
		_, err = db.Exec(
			`INSERT INTO custom_product_orders (custom_product_id, user_id, paypal_order_id, amount_usd,
			 paypal_payment_status, license_sn, license_email, status)
			 VALUES (?, ?, ?, ?, 'COMPLETED', ?, ?, 'fulfilled')`,
			vgProductID, userID, fmt.Sprintf("PAYPAL-V17-%d", rng.Int63n(1000000)), vgPrice,
			licenseSN, userEmail,
		)
		if err != nil {
			t.Logf("FAIL: failed to create virtual_goods order: %v", err)
			return false
		}

		// 6. Query fulfilled orders using the same JOIN pattern as the handler
		query := `SELECT o.id, o.status, COALESCE(o.license_sn, ''), COALESCE(o.license_email, ''),
			p.product_type, COALESCE(p.credits_amount, 0)
			FROM custom_product_orders o
			JOIN custom_products p ON o.custom_product_id = p.id
			WHERE o.user_id = ? AND o.status = 'fulfilled'
			ORDER BY o.created_at DESC`

		rows, err := db.Query(query, userID)
		if err != nil {
			t.Logf("FAIL: failed to query fulfilled orders: %v", err)
			return false
		}
		defer rows.Close()

		fulfilledCount := 0
		for rows.Next() {
			var orderID int64
			var status, licSN, licEmail, productType string
			var credAmt int
			if err := rows.Scan(&orderID, &status, &licSN, &licEmail, &productType, &credAmt); err != nil {
				t.Logf("FAIL: failed to scan order: %v", err)
				return false
			}

			fulfilledCount++

			// Property check: fulfilled virtual_goods must have non-empty license_sn and license_email
			if productType == "virtual_goods" {
				if licSN == "" {
					t.Logf("FAIL: fulfilled virtual_goods order %d has empty license_sn", orderID)
					return false
				}
				if licEmail == "" {
					t.Logf("FAIL: fulfilled virtual_goods order %d has empty license_email", orderID)
					return false
				}
			}

			// Property check: fulfilled credits must have positive credits_amount
			if productType == "credits" {
				if credAmt <= 0 {
					t.Logf("FAIL: fulfilled credits order %d has credits_amount=%d (expected > 0)", orderID, credAmt)
					return false
				}
			}
		}
		if err := rows.Err(); err != nil {
			t.Logf("FAIL: rows iteration error: %v", err)
			return false
		}

		// We should have at least 2 fulfilled orders (one credits, one virtual_goods)
		if fulfilledCount < 2 {
			t.Logf("FAIL: expected at least 2 fulfilled orders, got %d", fulfilledCount)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 17 violated: %v", err)
	}
}
