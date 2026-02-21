package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/xuri/excelize/v2"
)

// Feature: user-payment-settings, Property 1: 收款类型枚举验证
// **Validates: Requirements 1.1**
//
// For any string as payment_type input, the validation function should accept
// the input if and only if the string belongs to {"paypal", "bank_card", "wechat", "alipay", "check"}.
func TestProperty1_PaymentTypeEnumValidation(t *testing.T) {
	validTypes := map[string]bool{
		"paypal":        true,
		"bank_card":     true,
		"wechat":        true,
		"alipay":        true,
		"check":         true,
		"wire_transfer": true,
		"bank_card_us":  true,
		"bank_card_eu":  true,
		"bank_card_cn":  true,
	}

	// Provide valid payment_details for each type so that only the type check matters
	validDetailsForType := map[string]json.RawMessage{
		"paypal":        json.RawMessage(`{"account":"a","username":"u"}`),
		"bank_card":     json.RawMessage(`{"bank_name":"b","card_number":"c","account_holder":"h"}`),
		"wechat":        json.RawMessage(`{"account":"a","username":"u"}`),
		"alipay":        json.RawMessage(`{"account":"a","username":"u"}`),
		"check":         json.RawMessage(`{"full_legal_name":"n","province":"p","city":"c","district":"d","street_address":"s","postal_code":"z","phone":"t"}`),
		"wire_transfer": json.RawMessage(`{"beneficiary_name":"n","beneficiary_address":"a","bank_name":"b","swift_code":"s","account_number":"x"}`),
		"bank_card_us":  json.RawMessage(`{"legal_name":"n","routing_number":"r","account_number":"a","account_type":"checking"}`),
		"bank_card_eu":  json.RawMessage(`{"legal_name":"n","iban":"i","bic_swift":"b"}`),
		"bank_card_cn":  json.RawMessage(`{"real_name":"n","card_number":"c","bank_branch":"b"}`),
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate a random string (0 to 32 chars from printable ASCII)
		strLen := r.Intn(33)
		bs := make([]byte, strLen)
		for i := range bs {
			bs[i] = byte(r.Intn(94) + 33) // ASCII 33-126
		}
		paymentType := string(bs)

		isValid := validTypes[paymentType]

		// Use valid details if the type is valid, otherwise use a dummy valid JSON object
		details := json.RawMessage(`{"account":"a","username":"u"}`)
		if d, ok := validDetailsForType[paymentType]; ok {
			details = d
		}

		errMsg := validatePaymentInfo(paymentType, details)

		if isValid {
			// Valid type with valid details should pass
			if errMsg != "" {
				t.Logf("seed=%d: valid type %q rejected: %s", seed, paymentType, errMsg)
				return false
			}
		} else {
			// Invalid type should be rejected
			if errMsg == "" {
				t.Logf("seed=%d: invalid type %q was accepted", seed, paymentType)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (收款类型枚举验证) failed: %v", err)
	}

	// Also explicitly test all 5 valid types to ensure they are always accepted
	for pt, details := range validDetailsForType {
		errMsg := validatePaymentInfo(pt, details)
		if errMsg != "" {
			t.Errorf("valid payment type %q with valid details was rejected: %s", pt, errMsg)
		}
	}
}

// Feature: user-payment-settings, Property 2: 收款详情必填字段验证
// **Validates: Requirements 2.2, 3.2, 4.2, 5.2, 6.2, 9.1, 9.3**
//
// For any payment_type and payment_details combination, the validation function
// should accept the input if and only if all required fields for that type are
// non-empty strings after trimming whitespace.
// paypal/wechat/alipay: account, username; bank_card: bank_name, card_number, account_holder; check: address
func TestProperty2_RequiredFieldsValidation(t *testing.T) {
	type paymentTypeInfo struct {
		name   string
		fields []string
	}

	paymentTypes := []paymentTypeInfo{
		{"paypal", []string{"account", "username"}},
		{"wechat", []string{"account", "username"}},
		{"alipay", []string{"account", "username"}},
		{"bank_card", []string{"bank_name", "card_number", "account_holder"}},
		{"check", []string{"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"}},
		{"wire_transfer", []string{"beneficiary_name", "beneficiary_address", "bank_name", "swift_code", "account_number"}},
		{"bank_card_us", []string{"legal_name", "routing_number", "account_number", "account_type"}},
		{"bank_card_eu", []string{"legal_name", "iban", "bic_swift"}},
		{"bank_card_cn", []string{"real_name", "card_number", "bank_branch"}},
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Helper: generate a non-empty-after-trim string
	genNonEmptyValue := func(r *rand.Rand) string {
		// At least 1 non-whitespace char, optionally surrounded by spaces
		n := r.Intn(10) + 1
		bs := make([]byte, n)
		for i := range bs {
			bs[i] = byte(r.Intn(94) + 33) // printable non-space ASCII
		}
		// Optionally add leading/trailing spaces
		prefix := ""
		suffix := ""
		if r.Intn(2) == 0 {
			prefix = "  "
		}
		if r.Intn(2) == 0 {
			suffix = "  "
		}
		return prefix + string(bs) + suffix
	}

	// Helper: generate a whitespace-only or empty string
	genEmptyValue := func(r *rand.Rand) string {
		options := []string{"", " ", "  ", "\t", " \t ", "   "}
		return options[r.Intn(len(options))]
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Pick a random valid payment type
		pt := paymentTypes[r.Intn(len(paymentTypes))]

		// Decide: generate all-valid details or inject at least one invalid field
		allValid := r.Intn(2) == 0

		details := make(map[string]string)

		if allValid {
			// Fill all required fields with non-empty values
			for _, field := range pt.fields {
				details[field] = genNonEmptyValue(r)
			}
			// Optionally add extra fields (should not affect validation)
			if r.Intn(2) == 0 {
				details["extra_field"] = genNonEmptyValue(r)
			}

			detailsJSON, _ := json.Marshal(details)
			errMsg := validatePaymentInfo(pt.name, detailsJSON)
			if errMsg != "" {
				t.Logf("seed=%d: type=%s all-valid details rejected: %s, details=%v", seed, pt.name, errMsg, details)
				return false
			}
		} else {
			// Fill all fields with valid values first
			for _, field := range pt.fields {
				details[field] = genNonEmptyValue(r)
			}

			// Pick a random strategy for making it invalid
			strategy := r.Intn(3)
			targetIdx := r.Intn(len(pt.fields))
			targetField := pt.fields[targetIdx]

			switch strategy {
			case 0:
				// Set one required field to empty/whitespace
				details[targetField] = genEmptyValue(r)
			case 1:
				// Remove one required field entirely
				delete(details, targetField)
			case 2:
				// Set one required field to whitespace-only
				details[targetField] = "   \t  "
			}

			detailsJSON, _ := json.Marshal(details)
			errMsg := validatePaymentInfo(pt.name, detailsJSON)
			if errMsg == "" {
				t.Logf("seed=%d: type=%s invalid details accepted: details=%v", seed, pt.name, details)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (收款详情必填字段验证) failed: %v", err)
	}

	// Explicit edge cases: each type with all fields empty
	for _, pt := range paymentTypes {
		details := make(map[string]string)
		for _, field := range pt.fields {
			details[field] = ""
		}
		detailsJSON, _ := json.Marshal(details)
		errMsg := validatePaymentInfo(pt.name, detailsJSON)
		if errMsg == "" {
			t.Errorf("type %s with all empty fields should be rejected", pt.name)
		}
	}

	// Explicit edge cases: each type with all fields valid
	for _, pt := range paymentTypes {
		details := make(map[string]string)
		for _, field := range pt.fields {
			details[field] = "valid_value"
		}
		detailsJSON, _ := json.Marshal(details)
		errMsg := validatePaymentInfo(pt.name, detailsJSON)
		if errMsg != "" {
			t.Errorf("type %s with all valid fields was rejected: %s", pt.name, errMsg)
		}
	}
}

// Feature: user-payment-settings, Property 4: 收款信息序列化往返一致性
// **Validates: Requirements 7.4, 7.5**
//
// For any valid PaymentInfo object (with a legal payment_type and corresponding
// payment_details), serializing it to JSON and then deserializing it back should
// produce an object equivalent to the original.
func TestProperty4_PaymentInfoSerializationRoundTrip(t *testing.T) {
	type paymentTypeSpec struct {
		name   string
		fields []string
	}

	paymentTypes := []paymentTypeSpec{
		{"paypal", []string{"account", "username"}},
		{"wechat", []string{"account", "username"}},
		{"alipay", []string{"account", "username"}},
		{"bank_card", []string{"bank_name", "card_number", "account_holder"}},
		{"check", []string{"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"}},
		{"wire_transfer", []string{"beneficiary_name", "beneficiary_address", "bank_name", "swift_code", "account_number"}},
		{"bank_card_us", []string{"legal_name", "routing_number", "account_number", "account_type"}},
		{"bank_card_eu", []string{"legal_name", "iban", "bic_swift"}},
		{"bank_card_cn", []string{"real_name", "card_number", "bank_branch"}},
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// genNonEmptyString generates a random non-empty printable string.
	genNonEmptyString := func(r *rand.Rand) string {
		n := r.Intn(20) + 1
		bs := make([]byte, n)
		for i := range bs {
			// Use printable ASCII that is safe in JSON (avoid backslash and quote)
			ch := byte(r.Intn(90) + 33) // ASCII 33-122
			if ch == '\\' || ch == '"' {
				ch = 'x'
			}
			bs[i] = ch
		}
		return string(bs)
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Pick a random valid payment type
		spec := paymentTypes[r.Intn(len(paymentTypes))]

		// Build valid payment_details
		details := make(map[string]string)
		for _, field := range spec.fields {
			details[field] = genNonEmptyString(r)
		}

		detailsJSON, err := json.Marshal(details)
		if err != nil {
			t.Logf("seed=%d: failed to marshal details: %v", seed, err)
			return false
		}

		// Create original PaymentInfo
		original := PaymentInfo{
			PaymentType:    spec.name,
			PaymentDetails: json.RawMessage(detailsJSON),
		}

		// Serialize to JSON
		serialized, err := json.Marshal(original)
		if err != nil {
			t.Logf("seed=%d: failed to serialize PaymentInfo: %v", seed, err)
			return false
		}

		// Deserialize back
		var restored PaymentInfo
		if err := json.Unmarshal(serialized, &restored); err != nil {
			t.Logf("seed=%d: failed to deserialize PaymentInfo: %v", seed, err)
			return false
		}

		// Check PaymentType equality
		if original.PaymentType != restored.PaymentType {
			t.Logf("seed=%d: PaymentType mismatch: %q vs %q", seed, original.PaymentType, restored.PaymentType)
			return false
		}

		// Check PaymentDetails equality by comparing unmarshalled maps
		var origDetails, restoredDetails map[string]string
		if err := json.Unmarshal(original.PaymentDetails, &origDetails); err != nil {
			t.Logf("seed=%d: failed to unmarshal original details: %v", seed, err)
			return false
		}
		if err := json.Unmarshal(restored.PaymentDetails, &restoredDetails); err != nil {
			t.Logf("seed=%d: failed to unmarshal restored details: %v", seed, err)
			return false
		}

		if len(origDetails) != len(restoredDetails) {
			t.Logf("seed=%d: details length mismatch: %d vs %d", seed, len(origDetails), len(restoredDetails))
			return false
		}

		for k, v := range origDetails {
			if restoredDetails[k] != v {
				t.Logf("seed=%d: details field %q mismatch: %q vs %q", seed, k, v, restoredDetails[k])
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 (收款信息序列化往返一致性) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 3: 单一有效收款方式
// **Validates: Requirements 1.3, 7.3**
//
// For any user and any two consecutive payment info save operations,
// querying after the second save should return only the second save's content.
// The first save's information is completely overwritten.
func TestProperty3_SingleActivePaymentMethod(t *testing.T) {
	// Set up a temporary SQLite database
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	// Create the user_payment_info table
	_, err = tmpDB.Exec(`
		CREATE TABLE IF NOT EXISTS user_payment_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			payment_type TEXT NOT NULL,
			payment_details TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Save the original global db and replace with our temp db
	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	type paymentTypeSpec struct {
		name   string
		fields []string
	}

	paymentTypes := []paymentTypeSpec{
		{"paypal", []string{"account", "username"}},
		{"wechat", []string{"account", "username"}},
		{"alipay", []string{"account", "username"}},
		{"bank_card", []string{"bank_name", "card_number", "account_holder"}},
		{"check", []string{"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"}},
		{"wire_transfer", []string{"beneficiary_name", "beneficiary_address", "bank_name", "swift_code", "account_number"}},
		{"bank_card_us", []string{"legal_name", "routing_number", "account_number", "account_type"}},
		{"bank_card_eu", []string{"legal_name", "iban", "bic_swift"}},
		{"bank_card_cn", []string{"real_name", "card_number", "bank_branch"}},
	}

	genNonEmptyString := func(r *rand.Rand) string {
		n := r.Intn(15) + 1
		bs := make([]byte, n)
		for i := range bs {
			ch := byte(r.Intn(90) + 33)
			if ch == '\\' || ch == '"' {
				ch = 'x'
			}
			bs[i] = ch
		}
		return string(bs)
	}

	genValidPaymentInfo := func(r *rand.Rand) (string, map[string]string) {
		spec := paymentTypes[r.Intn(len(paymentTypes))]
		details := make(map[string]string)
		for _, field := range spec.fields {
			details[field] = genNonEmptyString(r)
		}
		return spec.name, details
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Use a unique user ID per iteration to avoid cross-iteration interference
		userID := r.Int63n(1000000) + 1

		// Clean up any existing record for this user
		tmpDB.Exec("DELETE FROM user_payment_info WHERE user_id = ?", userID)

		// Generate two different valid payment infos
		type1, details1 := genValidPaymentInfo(r)
		type2, details2 := genValidPaymentInfo(r)

		userIDStr := strconv.FormatInt(userID, 10)

		// --- First save ---
		details1JSON, _ := json.Marshal(details1)
		body1 := PaymentInfo{
			PaymentType:    type1,
			PaymentDetails: json.RawMessage(details1JSON),
		}
		bodyBytes1, _ := json.Marshal(body1)

		req1 := httptest.NewRequest(http.MethodPost, "/user/payment-info", bytes.NewReader(bodyBytes1))
		req1.Header.Set("X-User-ID", userIDStr)
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		handleSavePaymentInfo(w1, req1)

		if w1.Code != http.StatusOK {
			t.Logf("seed=%d: first save failed with status %d: %s", seed, w1.Code, w1.Body.String())
			return false
		}

		// --- Second save ---
		details2JSON, _ := json.Marshal(details2)
		body2 := PaymentInfo{
			PaymentType:    type2,
			PaymentDetails: json.RawMessage(details2JSON),
		}
		bodyBytes2, _ := json.Marshal(body2)

		req2 := httptest.NewRequest(http.MethodPost, "/user/payment-info", bytes.NewReader(bodyBytes2))
		req2.Header.Set("X-User-ID", userIDStr)
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		handleSavePaymentInfo(w2, req2)

		if w2.Code != http.StatusOK {
			t.Logf("seed=%d: second save failed with status %d: %s", seed, w2.Code, w2.Body.String())
			return false
		}

		// --- Query and verify only second save's content is returned ---
		reqGet := httptest.NewRequest(http.MethodGet, "/user/payment-info", nil)
		reqGet.Header.Set("X-User-ID", userIDStr)
		wGet := httptest.NewRecorder()
		handleGetPaymentInfo(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Logf("seed=%d: get failed with status %d: %s", seed, wGet.Code, wGet.Body.String())
			return false
		}

		var result struct {
			PaymentType    string          `json:"payment_type"`
			PaymentDetails json.RawMessage `json:"payment_details"`
		}
		if err := json.Unmarshal(wGet.Body.Bytes(), &result); err != nil {
			t.Logf("seed=%d: failed to parse get response: %v", seed, err)
			return false
		}

		// Verify payment_type matches second save
		if result.PaymentType != type2 {
			t.Logf("seed=%d: payment_type mismatch: got %q, want %q (first was %q)", seed, result.PaymentType, type2, type1)
			return false
		}

		// Verify payment_details matches second save
		var gotDetails map[string]string
		if err := json.Unmarshal(result.PaymentDetails, &gotDetails); err != nil {
			t.Logf("seed=%d: failed to parse payment_details: %v", seed, err)
			return false
		}

		if len(gotDetails) != len(details2) {
			t.Logf("seed=%d: details field count mismatch: got %d, want %d", seed, len(gotDetails), len(details2))
			return false
		}

		for k, v := range details2 {
			if gotDetails[k] != v {
				t.Logf("seed=%d: details field %q mismatch: got %q, want %q", seed, k, gotDetails[k], v)
				return false
			}
		}

		// Also verify there's only one record in the DB for this user
		var count int
		tmpDB.QueryRow("SELECT COUNT(*) FROM user_payment_info WHERE user_id = ?", userID).Scan(&count)
		if count != 1 {
			t.Logf("seed=%d: expected 1 record for user %d, got %d", seed, userID, count)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (单一有效收款方式) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 6: 手续费计算不变量
// **Validates: Requirements 11.3, 12.4**
//
// For any positive credits_amount, positive cash_rate, and non-negative fee_rate
// (0 ≤ fee_rate < 1), the calculation results should satisfy the invariants:
//   cash_amount = credits_amount × cash_rate
//   fee_amount = cash_amount × fee_rate
//   net_amount = cash_amount - fee_amount
func TestProperty6_FeeCalculationInvariants(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	const epsilon = 1e-9

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate positive credits_amount (0, 1_000_000]
		creditsAmount := r.Float64()*999999.0 + 1.0

		// Generate positive cash_rate (0, 100]
		cashRate := r.Float64()*99.0 + 1.0

		// Generate fee_rate as percentage in [0, 99) — e.g. 5.0 means 5%
		feeRate := r.Float64() * 99.0

		cashAmount, feeAmount, netAmount := calculateWithdrawalFee(creditsAmount, cashRate, feeRate)

		// Invariant 1: cash_amount = credits_amount × cash_rate
		expectedCash := creditsAmount * cashRate
		if diff := cashAmount - expectedCash; diff > epsilon || diff < -epsilon {
			t.Logf("seed=%d: cash_amount mismatch: got %f, want %f (diff=%e)", seed, cashAmount, expectedCash, diff)
			return false
		}

		// Invariant 2: fee_amount = cash_amount × fee_rate / 100
		expectedFee := cashAmount * feeRate / 100
		if diff := feeAmount - expectedFee; diff > epsilon || diff < -epsilon {
			t.Logf("seed=%d: fee_amount mismatch: got %f, want %f (diff=%e)", seed, feeAmount, expectedFee, diff)
			return false
		}

		// Invariant 3: net_amount = cash_amount - fee_amount
		expectedNet := cashAmount - feeAmount
		if diff := netAmount - expectedNet; diff > epsilon || diff < -epsilon {
			t.Logf("seed=%d: net_amount mismatch: got %f, want %f (diff=%e)", seed, netAmount, expectedNet, diff)
			return false
		}

		// Additional sanity checks
		if cashAmount <= 0 {
			t.Logf("seed=%d: cash_amount should be positive, got %f", seed, cashAmount)
			return false
		}
		if feeAmount < 0 {
			t.Logf("seed=%d: fee_amount should be non-negative, got %f", seed, feeAmount)
			return false
		}
		if netAmount <= 0 {
			t.Logf("seed=%d: net_amount should be positive for fee_rate < 1, got %f", seed, netAmount)
			return false
		}
		if netAmount > cashAmount {
			t.Logf("seed=%d: net_amount (%f) should not exceed cash_amount (%f)", seed, netAmount, cashAmount)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 (手续费计算不变量) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 5: 提现前置条件——收款信息检查
// **Validates: Requirements 10.1, 10.2, 10.3**
//
// For any user, when the user has NOT set payment info, a withdrawal request
// should be rejected (redirect with error=no_payment_info). When the user HAS
// set payment info, the payment info check should pass (no redirect for missing
// payment info).
func TestProperty5_WithdrawalPreconditionPaymentInfoCheck(t *testing.T) {
	// Set up a temporary SQLite database with all required tables
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	// Create required tables
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_type TEXT NOT NULL,
			auth_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(auth_type, auth_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_payment_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			payment_type TEXT NOT NULL,
			payment_details TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS pack_listings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			file_data BLOB NOT NULL,
			pack_name TEXT NOT NULL,
			pack_description TEXT,
			source_name TEXT,
			author_name TEXT,
			share_mode TEXT NOT NULL,
			credits_price INTEGER DEFAULT 0,
			status TEXT DEFAULT 'pending',
			download_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS credits_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			transaction_type TEXT NOT NULL,
			amount REAL NOT NULL,
			listing_id INTEGER,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawal_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			credits_amount REAL NOT NULL,
			cash_rate REAL NOT NULL,
			cash_amount REAL NOT NULL,
			payment_type TEXT DEFAULT '',
			payment_details TEXT DEFAULT '{}',
			fee_rate REAL DEFAULT 0,
			fee_amount REAL DEFAULT 0,
			net_amount REAL DEFAULT 0,
			status TEXT DEFAULT 'paid',
			display_name TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	} {
		if _, err := tmpDB.Exec(stmt); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	// Insert default settings needed by handleAuthorWithdraw
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('credit_cash_rate', '0.1')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_paypal', '0.03')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card', '0.01')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_wechat', '0.02')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_alipay', '0.02')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_check', '0.05')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_wire_transfer', '0.02')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card_us', '0.01')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card_eu', '0.01')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card_cn', '0.01')")

	// Save the original global db and replace with our temp db
	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	type paymentTypeSpec struct {
		name   string
		fields []string
	}

	paymentTypes := []paymentTypeSpec{
		{"paypal", []string{"account", "username"}},
		{"wechat", []string{"account", "username"}},
		{"alipay", []string{"account", "username"}},
		{"bank_card", []string{"bank_name", "card_number", "account_holder"}},
		{"check", []string{"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"}},
		{"wire_transfer", []string{"beneficiary_name", "beneficiary_address", "bank_name", "swift_code", "account_number"}},
		{"bank_card_us", []string{"legal_name", "routing_number", "account_number", "account_type"}},
		{"bank_card_eu", []string{"legal_name", "iban", "bic_swift"}},
		{"bank_card_cn", []string{"real_name", "card_number", "bank_branch"}},
	}

	genNonEmptyString := func(r *rand.Rand) string {
		n := r.Intn(15) + 1
		bs := make([]byte, n)
		for i := range bs {
			ch := byte(r.Intn(90) + 33)
			if ch == '\\' || ch == '"' {
				ch = 'x'
			}
			bs[i] = ch
		}
		return string(bs)
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	var userCounter int64

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		userCounter++

		// Create a unique user for this iteration
		authID := strconv.FormatInt(userCounter, 10) + "_" + strconv.FormatInt(seed, 10)
		res, err := tmpDB.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, credits_balance) VALUES ('test', ?, ?, ?)",
			authID, "TestUser_"+authID, 10000.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()
		userIDStr := strconv.FormatInt(userID, 10)

		// Create a pack listing so the user is considered an author
		_, err = tmpDB.Exec(
			"INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, share_mode) VALUES (?, 1, X'00', 'test_pack', 'free')",
			userID,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create pack listing: %v", seed, err)
			return false
		}

		// Add credits revenue so the user has unwithdrawn credits
		_, err = tmpDB.Exec(
			"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id) VALUES (?, 'purchase', -100.0, (SELECT id FROM pack_listings WHERE user_id = ? LIMIT 1))",
			userID, userID,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create credits transaction: %v", seed, err)
			return false
		}

		// --- Case 1: No payment info set → withdrawal should be rejected ---
		form1 := "credits_amount=10"
		req1 := httptest.NewRequest(http.MethodPost, "/user/author/withdraw", bytes.NewBufferString(form1))
		req1.Header.Set("X-User-ID", userIDStr)
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w1 := httptest.NewRecorder()
		handleAuthorWithdraw(w1, req1)

		// Should redirect with error=no_payment_info
		if w1.Code != http.StatusFound {
			t.Logf("seed=%d: case1 expected redirect (302), got %d", seed, w1.Code)
			return false
		}
		location1 := w1.Header().Get("Location")
		if !strings.Contains(location1, "error=no_payment_info") {
			t.Logf("seed=%d: case1 expected error=no_payment_info in redirect, got Location=%q", seed, location1)
			return false
		}

		// --- Case 2: Set payment info, then withdraw → payment info check should pass ---
		spec := paymentTypes[r.Intn(len(paymentTypes))]
		details := make(map[string]string)
		for _, field := range spec.fields {
			details[field] = genNonEmptyString(r)
		}
		detailsJSON, _ := json.Marshal(details)

		// Insert payment info directly into DB
		_, err = tmpDB.Exec(
			"INSERT OR REPLACE INTO user_payment_info (user_id, payment_type, payment_details) VALUES (?, ?, ?)",
			userID, spec.name, string(detailsJSON),
		)
		if err != nil {
			t.Logf("seed=%d: failed to insert payment info: %v", seed, err)
			return false
		}

		form2 := "credits_amount=10"
		req2 := httptest.NewRequest(http.MethodPost, "/user/author/withdraw", bytes.NewBufferString(form2))
		req2.Header.Set("X-User-ID", userIDStr)
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w2 := httptest.NewRecorder()
		handleAuthorWithdraw(w2, req2)

		// Should redirect, but NOT with error=no_payment_info
		if w2.Code != http.StatusFound {
			t.Logf("seed=%d: case2 expected redirect (302), got %d", seed, w2.Code)
			return false
		}
		location2 := w2.Header().Get("Location")
		if strings.Contains(location2, "error=no_payment_info") {
			t.Logf("seed=%d: case2 should NOT have error=no_payment_info, got Location=%q", seed, location2)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (提现前置条件——收款信息检查) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 7: 新提现申请初始状态
// **Validates: Requirements 11.2**
//
// For any successfully created withdrawal request record, its status field should be "pending".
func TestProperty7_NewWithdrawalRequestInitialStatus(t *testing.T) {
	// Set up a temporary SQLite database with all required tables
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	// Create required tables
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_type TEXT NOT NULL,
			auth_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(auth_type, auth_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_payment_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			payment_type TEXT NOT NULL,
			payment_details TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS pack_listings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			file_data BLOB NOT NULL,
			pack_name TEXT NOT NULL,
			pack_description TEXT,
			source_name TEXT,
			author_name TEXT,
			share_mode TEXT NOT NULL,
			credits_price INTEGER DEFAULT 0,
			status TEXT DEFAULT 'pending',
			download_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS credits_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			transaction_type TEXT NOT NULL,
			amount REAL NOT NULL,
			listing_id INTEGER,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawal_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			credits_amount REAL NOT NULL,
			cash_rate REAL NOT NULL,
			cash_amount REAL NOT NULL,
			payment_type TEXT DEFAULT '',
			payment_details TEXT DEFAULT '{}',
			fee_rate REAL DEFAULT 0,
			fee_amount REAL DEFAULT 0,
			net_amount REAL DEFAULT 0,
			status TEXT DEFAULT 'paid',
			display_name TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	} {
		if _, err := tmpDB.Exec(stmt); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	// Insert default settings needed by handleAuthorWithdraw
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('credit_cash_rate', '0.1')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_paypal', '0.03')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card', '0.01')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_wechat', '0.02')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_alipay', '0.02')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_check', '0.05')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_wire_transfer', '0.02')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card_us', '0.01')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card_eu', '0.01')")
	tmpDB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('fee_rate_bank_card_cn', '0.01')")

	// Save the original global db and replace with our temp db
	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	type paymentTypeSpec struct {
		name   string
		fields []string
	}

	paymentTypes := []paymentTypeSpec{
		{"paypal", []string{"account", "username"}},
		{"wechat", []string{"account", "username"}},
		{"alipay", []string{"account", "username"}},
		{"bank_card", []string{"bank_name", "card_number", "account_holder"}},
		{"check", []string{"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"}},
		{"wire_transfer", []string{"beneficiary_name", "beneficiary_address", "bank_name", "swift_code", "account_number"}},
		{"bank_card_us", []string{"legal_name", "routing_number", "account_number", "account_type"}},
		{"bank_card_eu", []string{"legal_name", "iban", "bic_swift"}},
		{"bank_card_cn", []string{"real_name", "card_number", "bank_branch"}},
	}

	genNonEmptyString := func(r *rand.Rand) string {
		n := r.Intn(15) + 1
		bs := make([]byte, n)
		for i := range bs {
			ch := byte(r.Intn(90) + 33)
			if ch == '\\' || ch == '"' {
				ch = 'x'
			}
			bs[i] = ch
		}
		return string(bs)
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	var userCounter int64

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		userCounter++

		// Create a unique user for this iteration
		authID := strconv.FormatInt(userCounter, 10) + "_p7_" + strconv.FormatInt(seed, 10)
		res, err := tmpDB.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, credits_balance) VALUES ('test', ?, ?, ?)",
			authID, "TestUser_"+authID, 10000.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()
		userIDStr := strconv.FormatInt(userID, 10)

		// Create a pack listing so the user is considered an author
		_, err = tmpDB.Exec(
			"INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, share_mode) VALUES (?, 1, X'00', 'test_pack', 'free')",
			userID,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create pack listing: %v", seed, err)
			return false
		}

		// Add credits revenue so the user has unwithdrawn credits (large enough for withdrawal)
		_, err = tmpDB.Exec(
			"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id) VALUES (?, 'purchase', -50000.0, (SELECT id FROM pack_listings WHERE user_id = ? LIMIT 1))",
			userID, userID,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create credits transaction: %v", seed, err)
			return false
		}

		// Set payment info for the user (randomly pick a payment type)
		spec := paymentTypes[r.Intn(len(paymentTypes))]
		details := make(map[string]string)
		for _, field := range spec.fields {
			details[field] = genNonEmptyString(r)
		}
		detailsJSON, _ := json.Marshal(details)

		_, err = tmpDB.Exec(
			"INSERT OR REPLACE INTO user_payment_info (user_id, payment_type, payment_details) VALUES (?, ?, ?)",
			userID, spec.name, string(detailsJSON),
		)
		if err != nil {
			t.Logf("seed=%d: failed to insert payment info: %v", seed, err)
			return false
		}

		// Generate a withdrawal amount large enough to pass minimum net_amount threshold (100)
		// With cash_rate=0.1 and max fee_rate ~5%, need at least ~1100 credits
		// Use range 1100-5000 to ensure we always pass the minimum
		creditsAmount := float64(r.Intn(3900) + 1100)

		// Submit withdrawal request via handleAuthorWithdraw
		form := "credits_amount=" + strconv.FormatFloat(creditsAmount, 'f', 2, 64)
		req := httptest.NewRequest(http.MethodPost, "/user/author/withdraw", bytes.NewBufferString(form))
		req.Header.Set("X-User-ID", userIDStr)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handleAuthorWithdraw(w, req)

		// Should redirect with success
		if w.Code != http.StatusFound {
			t.Logf("seed=%d: expected redirect (302), got %d", seed, w.Code)
			return false
		}
		location := w.Header().Get("Location")
		if !strings.Contains(location, "success=withdraw") {
			t.Logf("seed=%d: expected success=withdraw in redirect, got Location=%q", seed, location)
			return false
		}

		// Query the withdrawal_records table for the newly created record and verify status = "pending"
		var status string
		err = tmpDB.QueryRow(
			"SELECT status FROM withdrawal_records WHERE user_id = ? ORDER BY id DESC LIMIT 1",
			userID,
		).Scan(&status)
		if err != nil {
			t.Logf("seed=%d: failed to query withdrawal record: %v", seed, err)
			return false
		}

		if status != "pending" {
			t.Logf("seed=%d: expected status='pending', got status=%q", seed, status)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 (新提现申请初始状态) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 8: 手续费率设置往返一致性
// **Validates: Requirements 12.2, 12.3**
//
// For any fee rate configuration of the five payment types (all non-negative floats),
// saving to the settings table and then reading back should return equivalent values.
func TestProperty8_FeeRateSettingsRoundTrip(t *testing.T) {
	// Set up a temporary SQLite database with the settings table
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	_, err = tmpDB.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create settings table: %v", err)
	}

	// Save the original global db and replace with our temp db
	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	feeKeys := []string{
		"fee_rate_paypal",
		"fee_rate_wechat",
		"fee_rate_alipay",
		"fee_rate_check",
		"fee_rate_wire_transfer",
		"fee_rate_bank_card_us",
		"fee_rate_bank_card_eu",
		"fee_rate_bank_card_cn",
	}

	const epsilon = 1e-12

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate random non-negative fee rates for all payment types
		paypalRate := r.Float64()
		wechatRate := r.Float64()
		alipayRate := r.Float64()
		checkRate := r.Float64()
		wireTransferRate := r.Float64()
		bankCardUSRate := r.Float64()
		bankCardEURate := r.Float64()
		bankCardCNRate := r.Float64()

		// Build the request body matching handleAdminSaveWithdrawalFees expected format
		reqBody := map[string]float64{
			"paypal_fee_rate":        paypalRate,
			"wechat_fee_rate":        wechatRate,
			"alipay_fee_rate":        alipayRate,
			"check_fee_rate":         checkRate,
			"wire_transfer_fee_rate": wireTransferRate,
			"bank_card_us_fee_rate":  bankCardUSRate,
			"bank_card_eu_fee_rate":  bankCardEURate,
			"bank_card_cn_fee_rate":  bankCardCNRate,
		}
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			t.Logf("seed=%d: failed to marshal request body: %v", seed, err)
			return false
		}

		// Save via handleAdminSaveWithdrawalFees
		req := httptest.NewRequest(http.MethodPost, "/admin/api/settings/withdrawal-fees", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleAdminSaveWithdrawalFees(w, req)

		if w.Code != http.StatusOK {
			t.Logf("seed=%d: save failed with status %d: %s", seed, w.Code, w.Body.String())
			return false
		}

		// Read back from settings table via getSetting and verify
		expectedRates := map[string]float64{
			"fee_rate_paypal":        paypalRate,
			"fee_rate_wechat":        wechatRate,
			"fee_rate_alipay":        alipayRate,
			"fee_rate_check":         checkRate,
			"fee_rate_wire_transfer": wireTransferRate,
			"fee_rate_bank_card_us":  bankCardUSRate,
			"fee_rate_bank_card_eu":  bankCardEURate,
			"fee_rate_bank_card_cn":  bankCardCNRate,
		}

		for _, key := range feeKeys {
			storedStr := getSetting(key)
			if storedStr == "" {
				t.Logf("seed=%d: getSetting(%q) returned empty string", seed, key)
				return false
			}

			storedRate, err := strconv.ParseFloat(storedStr, 64)
			if err != nil {
				t.Logf("seed=%d: failed to parse stored rate for %q: %v (raw=%q)", seed, key, err, storedStr)
				return false
			}

			expected := expectedRates[key]
			// The handler stores using fmt.Sprintf("%g", rate), so we compare
			// the round-tripped value: format with %g then parse back
			expectedStr := fmt.Sprintf("%g", expected)
			expectedRoundTripped, _ := strconv.ParseFloat(expectedStr, 64)

			diff := storedRate - expectedRoundTripped
			if diff > epsilon || diff < -epsilon {
				t.Logf("seed=%d: rate mismatch for %q: stored=%f, expected=%f (raw stored=%q, expected formatted=%q)",
					seed, key, storedRate, expectedRoundTripped, storedStr, expectedStr)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 (手续费率设置往返一致性) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 9: 提现状态筛选正确性
// **Validates: Requirements 13.2**
//
// For any set of withdrawal records and a filter status value, filtering by status
// should return only records with that status and not miss any matching records.
func TestProperty9_WithdrawalStatusFilterCorrectness(t *testing.T) {
	// Set up a temporary SQLite database with all required tables
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	// Create required tables
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_type TEXT NOT NULL,
			auth_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(auth_type, auth_id)
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawal_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			credits_amount REAL NOT NULL,
			cash_rate REAL NOT NULL,
			cash_amount REAL NOT NULL,
			payment_type TEXT DEFAULT '',
			payment_details TEXT DEFAULT '{}',
			fee_rate REAL DEFAULT 0,
			fee_amount REAL DEFAULT 0,
			net_amount REAL DEFAULT 0,
			status TEXT DEFAULT 'paid',
			display_name TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	} {
		if _, err := tmpDB.Exec(stmt); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	// Save the original global db and replace with our temp db
	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	var iterCounter int64

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		iterCounter++

		// Clean up withdrawal_records and users for this iteration
		tmpDB.Exec("DELETE FROM withdrawal_records")
		tmpDB.Exec("DELETE FROM users")

		// Create a user
		authID := fmt.Sprintf("p9_%d_%d", iterCounter, seed)
		res, err := tmpDB.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name) VALUES ('test', ?, ?)",
			authID, "User_"+authID,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Generate a random number of withdrawal records (1 to 20)
		numRecords := r.Intn(20) + 1
		expectedPending := 0
		expectedPaid := 0

		for i := 0; i < numRecords; i++ {
			status := "pending"
			if r.Intn(2) == 0 {
				status = "paid"
			}
			if status == "pending" {
				expectedPending++
			} else {
				expectedPaid++
			}

			creditsAmount := float64(r.Intn(1000) + 1)
			cashRate := 0.1
			cashAmount := creditsAmount * cashRate

			_, err := tmpDB.Exec(
				`INSERT INTO withdrawal_records (user_id, credits_amount, cash_rate, cash_amount,
				 payment_type, payment_details, fee_rate, fee_amount, net_amount, status, display_name)
				 VALUES (?, ?, ?, ?, 'paypal', '{"account":"a","username":"u"}', 0.03, ?, ?, ?, ?)`,
				userID, creditsAmount, cashRate, cashAmount,
				cashAmount*0.03, cashAmount*0.97, status, "User_"+authID,
			)
			if err != nil {
				t.Logf("seed=%d: failed to insert withdrawal record: %v", seed, err)
				return false
			}
		}

		// Test filtering by "pending"
		reqPending := httptest.NewRequest(http.MethodGet, "/admin/api/withdrawals?status=pending", nil)
		wPending := httptest.NewRecorder()
		handleAdminGetWithdrawals(wPending, reqPending)

		if wPending.Code != http.StatusOK {
			t.Logf("seed=%d: pending filter returned status %d", seed, wPending.Code)
			return false
		}

		var pendingResp struct {
			Withdrawals []WithdrawalRequest `json:"withdrawals"`
		}
		if err := json.Unmarshal(wPending.Body.Bytes(), &pendingResp); err != nil {
			t.Logf("seed=%d: failed to parse pending response: %v", seed, err)
			return false
		}

		// Verify all returned records have status "pending"
		for _, wr := range pendingResp.Withdrawals {
			if wr.Status != "pending" {
				t.Logf("seed=%d: pending filter returned record with status=%q (id=%d)", seed, wr.Status, wr.ID)
				return false
			}
		}
		// Verify count matches expected
		if len(pendingResp.Withdrawals) != expectedPending {
			t.Logf("seed=%d: pending filter returned %d records, expected %d", seed, len(pendingResp.Withdrawals), expectedPending)
			return false
		}

		// Test filtering by "paid"
		reqPaid := httptest.NewRequest(http.MethodGet, "/admin/api/withdrawals?status=paid", nil)
		wPaid := httptest.NewRecorder()
		handleAdminGetWithdrawals(wPaid, reqPaid)

		if wPaid.Code != http.StatusOK {
			t.Logf("seed=%d: paid filter returned status %d", seed, wPaid.Code)
			return false
		}

		var paidResp struct {
			Withdrawals []WithdrawalRequest `json:"withdrawals"`
		}
		if err := json.Unmarshal(wPaid.Body.Bytes(), &paidResp); err != nil {
			t.Logf("seed=%d: failed to parse paid response: %v", seed, err)
			return false
		}

		// Verify all returned records have status "paid"
		for _, wr := range paidResp.Withdrawals {
			if wr.Status != "paid" {
				t.Logf("seed=%d: paid filter returned record with status=%q (id=%d)", seed, wr.Status, wr.ID)
				return false
			}
		}
		// Verify count matches expected
		if len(paidResp.Withdrawals) != expectedPaid {
			t.Logf("seed=%d: paid filter returned %d records, expected %d", seed, len(paidResp.Withdrawals), expectedPaid)
			return false
		}

		// Verify that pending + paid = total records
		if len(pendingResp.Withdrawals)+len(paidResp.Withdrawals) != numRecords {
			t.Logf("seed=%d: pending(%d) + paid(%d) != total(%d)",
				seed, len(pendingResp.Withdrawals), len(paidResp.Withdrawals), numRecords)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 (提现状态筛选正确性) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 10: 审核状态流转
// **Validates: Requirements 13.3, 14.4**
//
// For any set of pending withdrawal records, after batch approval,
// selected records should change to "paid" and unselected records should remain "pending".
func TestProperty10_ApprovalStatusTransition(t *testing.T) {
	// Set up a temporary SQLite database with all required tables
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	// Create required tables
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_type TEXT NOT NULL,
			auth_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(auth_type, auth_id)
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawal_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			credits_amount REAL NOT NULL,
			cash_rate REAL NOT NULL,
			cash_amount REAL NOT NULL,
			payment_type TEXT DEFAULT '',
			payment_details TEXT DEFAULT '{}',
			fee_rate REAL DEFAULT 0,
			fee_amount REAL DEFAULT 0,
			net_amount REAL DEFAULT 0,
			status TEXT DEFAULT 'paid',
			display_name TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	} {
		if _, err := tmpDB.Exec(stmt); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	// Save the original global db and replace with our temp db
	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	var iterCounter int64

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		iterCounter++

		// Clean up for this iteration
		tmpDB.Exec("DELETE FROM withdrawal_records")
		tmpDB.Exec("DELETE FROM users")

		// Create a user
		authID := fmt.Sprintf("p10_%d_%d", iterCounter, seed)
		res, err := tmpDB.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name) VALUES ('test', ?, ?)",
			authID, "User_"+authID,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Generate a random number of pending withdrawal records (2 to 20)
		numRecords := r.Intn(19) + 2
		var allIDs []int64

		for i := 0; i < numRecords; i++ {
			creditsAmount := float64(r.Intn(1000) + 1)
			cashRate := 0.1
			cashAmount := creditsAmount * cashRate

			result, err := tmpDB.Exec(
				`INSERT INTO withdrawal_records (user_id, credits_amount, cash_rate, cash_amount,
				 payment_type, payment_details, fee_rate, fee_amount, net_amount, status, display_name)
				 VALUES (?, ?, ?, ?, 'paypal', '{"account":"a","username":"u"}', 0.03, ?, ?, 'pending', ?)`,
				userID, creditsAmount, cashRate, cashAmount,
				cashAmount*0.03, cashAmount*0.97, "User_"+authID,
			)
			if err != nil {
				t.Logf("seed=%d: failed to insert withdrawal record: %v", seed, err)
				return false
			}
			id, _ := result.LastInsertId()
			allIDs = append(allIDs, id)
		}

		// Randomly select a subset of IDs to approve (at least 1, at most all)
		numToApprove := r.Intn(numRecords) + 1
		// Shuffle and pick first numToApprove
		perm := r.Perm(numRecords)
		approveIDSet := make(map[int64]bool)
		var approveIDs []int64
		for i := 0; i < numToApprove; i++ {
			id := allIDs[perm[i]]
			approveIDs = append(approveIDs, id)
			approveIDSet[id] = true
		}

		// Call handleAdminApproveWithdrawals with the selected IDs
		reqBody, _ := json.Marshal(map[string]interface{}{"ids": approveIDs})
		req := httptest.NewRequest(http.MethodPost, "/admin/api/withdrawals/approve", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleAdminApproveWithdrawals(w, req)

		if w.Code != http.StatusOK {
			t.Logf("seed=%d: approve returned status %d, body=%s", seed, w.Code, w.Body.String())
			return false
		}

		// Verify: selected records should be "paid", unselected should remain "pending"
		for _, id := range allIDs {
			var status string
			err := tmpDB.QueryRow("SELECT status FROM withdrawal_records WHERE id = ?", id).Scan(&status)
			if err != nil {
				t.Logf("seed=%d: failed to query record id=%d: %v", seed, id, err)
				return false
			}

			if approveIDSet[id] {
				if status != "paid" {
					t.Logf("seed=%d: approved record id=%d should be 'paid' but got %q", seed, id, status)
					return false
				}
			} else {
				if status != "pending" {
					t.Logf("seed=%d: unapproved record id=%d should be 'pending' but got %q", seed, id, status)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 10 (审核状态流转) failed: %v", err)
	}
}

// Feature: user-payment-settings, Property 11: Excel 导出完整性
// **Validates: Requirements 14.1, 14.2**
//
// For any set of withdrawal records, the exported Excel file should contain
// the same number of data rows as input records, each row should contain all
// required columns (作者名称、收款方式、收款详情、提现金额、手续费率、手续费金额、实付金额),
// and column values should match the source data.
func TestProperty11_ExcelExportCompleteness(t *testing.T) {
	// Set up a temporary SQLite database
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_type TEXT NOT NULL,
			auth_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(auth_type, auth_id)
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawal_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			credits_amount REAL NOT NULL,
			cash_rate REAL NOT NULL,
			cash_amount REAL NOT NULL,
			payment_type TEXT DEFAULT '',
			payment_details TEXT DEFAULT '{}',
			fee_rate REAL DEFAULT 0,
			fee_amount REAL DEFAULT 0,
			net_amount REAL DEFAULT 0,
			status TEXT DEFAULT 'paid',
			display_name TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	} {
		if _, err := tmpDB.Exec(stmt); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	paymentTypes := []string{"paypal", "bank_card", "wechat", "alipay", "check", "wire_transfer", "bank_card_us", "bank_card_eu", "bank_card_cn"}
	typeLabels := map[string]string{
		"paypal": "PayPal", "bank_card": "银行卡", "wechat": "微信", "alipay": "AliPay", "check": "支票",
		"wire_transfer": "电汇", "bank_card_us": "美国银行卡", "bank_card_eu": "欧洲银行卡", "bank_card_cn": "中国银行卡",
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// roundTo2 rounds a float to 2 decimal places to avoid floating-point precision issues
	roundTo2 := func(v float64) float64 {
		return math.Round(v*100) / 100
	}

	var iterCounter int64

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		iterCounter++

		tmpDB.Exec("DELETE FROM withdrawal_records")
		tmpDB.Exec("DELETE FROM users")

		// Create a user
		authID := fmt.Sprintf("p11_%d_%d", iterCounter, seed)
		res, err := tmpDB.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name) VALUES ('test', ?, ?)",
			authID, "Author_"+authID,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Generate 1 to 15 random withdrawal records
		numRecords := r.Intn(15) + 1

		type recordData struct {
			ID             int64
			DisplayName    string
			PaymentType    string
			PaymentDetails string
			CashAmount     float64
			FeeRate        float64
			FeeAmount      float64
			NetAmount      float64
		}
		var records []recordData
		var idStrs []string

		for i := 0; i < numRecords; i++ {
			pt := paymentTypes[r.Intn(len(paymentTypes))]
			displayName := fmt.Sprintf("Author_%s_%d", authID, i)
			creditsAmount := float64(r.Intn(9999) + 1)
			cashRate := 0.1
			cashAmount := roundTo2(creditsAmount * cashRate)
			feeRate := float64(r.Intn(20)) / 100.0 // 0% to 19%
			feeAmount := roundTo2(cashAmount * feeRate)
			netAmount := roundTo2(cashAmount - feeAmount)

			var details string
			switch pt {
			case "paypal", "wechat", "alipay":
				details = fmt.Sprintf(`{"account":"acc_%d","username":"user_%d"}`, i, i)
			case "bank_card":
				details = fmt.Sprintf(`{"bank_name":"Bank_%d","card_number":"6222%04d","account_holder":"Holder_%d"}`, i, i, i)
			case "check":
				details = fmt.Sprintf(`{"address":"Addr_%d"}`, i)
			}

			result, err := tmpDB.Exec(
				`INSERT INTO withdrawal_records (user_id, credits_amount, cash_rate, cash_amount,
				 payment_type, payment_details, fee_rate, fee_amount, net_amount, status, display_name)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
				userID, creditsAmount, cashRate, cashAmount, pt, details, feeRate, feeAmount, netAmount, displayName,
			)
			if err != nil {
				t.Logf("seed=%d: failed to insert record: %v", seed, err)
				return false
			}
			id, _ := result.LastInsertId()
			records = append(records, recordData{
				ID: id, DisplayName: displayName, PaymentType: pt,
				PaymentDetails: details, CashAmount: cashAmount,
				FeeRate: feeRate, FeeAmount: feeAmount, NetAmount: netAmount,
			})
			idStrs = append(idStrs, strconv.FormatInt(id, 10))
		}

		// Call the export handler
		url := "/admin/api/withdrawals/export?ids=" + strings.Join(idStrs, ",")
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		handleAdminExportWithdrawals(w, req)

		if w.Code != http.StatusOK {
			t.Logf("seed=%d: export returned status %d, body=%s", seed, w.Code, w.Body.String())
			return false
		}

		// Parse the Excel file from response body
		xlFile, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
		if err != nil {
			t.Logf("seed=%d: failed to open Excel: %v", seed, err)
			return false
		}
		defer xlFile.Close()

		sheetName := "提现记录"
		rows, err := xlFile.GetRows(sheetName)
		if err != nil {
			t.Logf("seed=%d: failed to get rows from sheet %q: %v", seed, sheetName, err)
			return false
		}

		// First row is header, remaining rows are data
		if len(rows) < 1 {
			t.Logf("seed=%d: Excel has no rows at all", seed)
			return false
		}

		// Verify header row has all required columns
		expectedHeaders := []string{"作者名称", "收款方式", "收款详情", "提现金额", "手续费率", "手续费金额", "实付金额"}
		header := rows[0]
		if len(header) < len(expectedHeaders) {
			t.Logf("seed=%d: header has %d columns, expected %d", seed, len(header), len(expectedHeaders))
			return false
		}
		for i, eh := range expectedHeaders {
			if header[i] != eh {
				t.Logf("seed=%d: header[%d] = %q, expected %q", seed, i, header[i], eh)
				return false
			}
		}

		// Verify data row count matches
		dataRows := rows[1:]
		if len(dataRows) != numRecords {
			t.Logf("seed=%d: got %d data rows, expected %d", seed, len(dataRows), numRecords)
			return false
		}

		// Verify each row's values match source data
		for idx, rec := range records {
			row := dataRows[idx]
			if len(row) < 7 {
				t.Logf("seed=%d row=%d: only %d columns, expected 7", seed, idx, len(row))
				return false
			}

			// Column 0: 作者名称
			if row[0] != rec.DisplayName {
				t.Logf("seed=%d row=%d: DisplayName = %q, expected %q", seed, idx, row[0], rec.DisplayName)
				return false
			}

			// Column 1: 收款方式 (label)
			expectedLabel := typeLabels[rec.PaymentType]
			if row[1] != expectedLabel {
				t.Logf("seed=%d row=%d: PaymentType label = %q, expected %q", seed, idx, row[1], expectedLabel)
				return false
			}

			// Column 2: 收款详情
			if row[2] != rec.PaymentDetails {
				t.Logf("seed=%d row=%d: PaymentDetails = %q, expected %q", seed, idx, row[2], rec.PaymentDetails)
				return false
			}

			// Column 3: 提现金额 (CashAmount)
			parsedCash, err := strconv.ParseFloat(row[3], 64)
			if err != nil {
				t.Logf("seed=%d row=%d: failed to parse CashAmount %q: %v", seed, idx, row[3], err)
				return false
			}
			if fmt.Sprintf("%.2f", parsedCash) != fmt.Sprintf("%.2f", rec.CashAmount) {
				t.Logf("seed=%d row=%d: CashAmount = %.2f, expected %.2f", seed, idx, parsedCash, rec.CashAmount)
				return false
			}

			// Column 4: 手续费率 (formatted as "X.XX%")
			expectedFeeRateStr := fmt.Sprintf("%.2f%%", rec.FeeRate*100)
			if row[4] != expectedFeeRateStr {
				t.Logf("seed=%d row=%d: FeeRate = %q, expected %q", seed, idx, row[4], expectedFeeRateStr)
				return false
			}

			// Column 5: 手续费金额
			parsedFee, err := strconv.ParseFloat(row[5], 64)
			if err != nil {
				t.Logf("seed=%d row=%d: failed to parse FeeAmount %q: %v", seed, idx, row[5], err)
				return false
			}
			if fmt.Sprintf("%.2f", parsedFee) != fmt.Sprintf("%.2f", rec.FeeAmount) {
				t.Logf("seed=%d row=%d: FeeAmount = %.2f, expected %.2f", seed, idx, parsedFee, rec.FeeAmount)
				return false
			}

			// Column 6: 实付金额
			parsedNet, err := strconv.ParseFloat(row[6], 64)
			if err != nil {
				t.Logf("seed=%d row=%d: failed to parse NetAmount %q: %v", seed, idx, row[6], err)
				return false
			}
			if fmt.Sprintf("%.2f", parsedNet) != fmt.Sprintf("%.2f", rec.NetAmount) {
				t.Logf("seed=%d row=%d: NetAmount = %.2f, expected %.2f", seed, idx, parsedNet, rec.NetAmount)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 11 (Excel 导出完整性) failed: %v", err)
	}
}

// Feature: check-payment-enhancement, Property 1: 支票必填字段验证
// **Validates: Requirements 1.2, 1.3, 2.2, 3.2**
//
// For any check payment_details where the 7 required fields (full_legal_name, province,
// city, district, street_address, postal_code, phone) have randomly generated values
// (possibly non-empty strings, empty strings, or whitespace-only strings), the validation
// function should accept the input if and only if all 7 required fields are non-empty
// after trimming. The memo field's presence or value should not affect the validation result.
func TestCheckProperty1_RequiredFieldValidation(t *testing.T) {
	checkRequiredFields := []string{
		"full_legal_name", "province", "city", "district",
		"street_address", "postal_code", "phone",
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// genNonEmptyValue generates a string that is non-empty after trimming.
	genNonEmptyValue := func(r *rand.Rand) string {
		n := r.Intn(10) + 1
		bs := make([]byte, n)
		for i := range bs {
			bs[i] = byte(r.Intn(94) + 33) // printable non-space ASCII
		}
		prefix := ""
		suffix := ""
		if r.Intn(2) == 0 {
			prefix = "  "
		}
		if r.Intn(2) == 0 {
			suffix = "  "
		}
		return prefix + string(bs) + suffix
	}

	// genMaybeEmptyValue generates a string that may be empty, whitespace-only, or non-empty.
	genMaybeEmptyValue := func(r *rand.Rand) string {
		switch r.Intn(3) {
		case 0:
			return "" // empty
		case 1:
			// whitespace-only
			options := []string{" ", "  ", "\t", " \t ", "   "}
			return options[r.Intn(len(options))]
		default:
			return genNonEmptyValue(r) // non-empty after trim
		}
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		details := make(map[string]string)

		// Generate random values for each required field
		for _, field := range checkRequiredFields {
			details[field] = genMaybeEmptyValue(r)
		}

		// Randomly decide whether to include memo, and with what value
		switch r.Intn(3) {
		case 0:
			// no memo field at all
		case 1:
			details["memo"] = "" // empty memo
		case 2:
			details["memo"] = genMaybeEmptyValue(r) // random memo value
		}

		// Compute expected result: valid IFF all required fields are non-empty after trim
		allValid := true
		for _, field := range checkRequiredFields {
			if strings.TrimSpace(details[field]) == "" {
				allValid = false
				break
			}
		}

		detailsJSON, _ := json.Marshal(details)
		errMsg := validatePaymentInfo("check", detailsJSON)

		if allValid && errMsg != "" {
			t.Logf("seed=%d: all required fields valid but rejected: %s, details=%v", seed, errMsg, details)
			return false
		}
		if !allValid && errMsg == "" {
			t.Logf("seed=%d: some required fields invalid but accepted: details=%v", seed, details)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Check Property 1 (支票必填字段验证) failed: %v", err)
	}
}

// Feature: check-payment-enhancement, Property 2: 支票收款信息保存读取往返一致性
// **Validates: Requirements 2.3, 4.5, 5.1, 5.2, 5.3**
//
// For any valid check PaymentInfo object (7 required fields are all non-empty strings,
// memo field is randomly empty or has a value), saving via POST /user/payment-info and
// then reading via GET /user/payment-info should return payment_type "check" and all
// field values (including memo) should match the saved values.
func TestCheckProperty2_SaveReadRoundTrip(t *testing.T) {
	// Set up a temporary SQLite database
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
	defer tmpDB.Close()

	// Create the user_payment_info table
	_, err = tmpDB.Exec(`
		CREATE TABLE IF NOT EXISTS user_payment_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			payment_type TEXT NOT NULL,
			payment_details TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Save the original global db and replace with our temp db
	origDB := db
	db = tmpDB
	defer func() { db = origDB }()

	checkRequiredFields := []string{
		"full_legal_name", "province", "city", "district",
		"street_address", "postal_code", "phone",
	}

	// genNonEmptyString generates a random non-empty printable string (safe for JSON).
	genNonEmptyString := func(r *rand.Rand) string {
		n := r.Intn(20) + 1
		bs := make([]byte, n)
		for i := range bs {
			ch := byte(r.Intn(90) + 33) // ASCII 33-122
			if ch == '\\' || ch == '"' {
				ch = 'x'
			}
			bs[i] = ch
		}
		return string(bs)
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Use a unique user ID per iteration
		userID := r.Int63n(1000000) + 1
		tmpDB.Exec("DELETE FROM user_payment_info WHERE user_id = ?", userID)
		userIDStr := strconv.FormatInt(userID, 10)

		// Build valid check payment details with all 7 required fields
		details := make(map[string]string)
		for _, field := range checkRequiredFields {
			details[field] = genNonEmptyString(r)
		}

		// Randomly include memo: no memo, empty memo, or non-empty memo
		switch r.Intn(3) {
		case 0:
			// no memo field
		case 1:
			details["memo"] = "" // empty memo
		case 2:
			details["memo"] = genNonEmptyString(r) // non-empty memo
		}

		// --- Save via POST /user/payment-info ---
		detailsJSON, _ := json.Marshal(details)
		body := PaymentInfo{
			PaymentType:    "check",
			PaymentDetails: json.RawMessage(detailsJSON),
		}
		bodyBytes, _ := json.Marshal(body)

		reqPost := httptest.NewRequest(http.MethodPost, "/user/payment-info", bytes.NewReader(bodyBytes))
		reqPost.Header.Set("X-User-ID", userIDStr)
		reqPost.Header.Set("Content-Type", "application/json")
		wPost := httptest.NewRecorder()
		handleSavePaymentInfo(wPost, reqPost)

		if wPost.Code != http.StatusOK {
			t.Logf("seed=%d: POST save failed with status %d: %s", seed, wPost.Code, wPost.Body.String())
			return false
		}

		// --- Read back via GET /user/payment-info ---
		reqGet := httptest.NewRequest(http.MethodGet, "/user/payment-info", nil)
		reqGet.Header.Set("X-User-ID", userIDStr)
		wGet := httptest.NewRecorder()
		handleGetPaymentInfo(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Logf("seed=%d: GET read failed with status %d: %s", seed, wGet.Code, wGet.Body.String())
			return false
		}

		var result struct {
			PaymentType    string          `json:"payment_type"`
			PaymentDetails json.RawMessage `json:"payment_details"`
		}
		if err := json.Unmarshal(wGet.Body.Bytes(), &result); err != nil {
			t.Logf("seed=%d: failed to parse GET response: %v", seed, err)
			return false
		}

		// Assert payment_type is "check"
		if result.PaymentType != "check" {
			t.Logf("seed=%d: payment_type mismatch: got %q, want %q", seed, result.PaymentType, "check")
			return false
		}

		// Assert all field values match
		var gotDetails map[string]string
		if err := json.Unmarshal(result.PaymentDetails, &gotDetails); err != nil {
			t.Logf("seed=%d: failed to parse payment_details: %v", seed, err)
			return false
		}

		// Check all saved fields are present and match
		for k, v := range details {
			if gotDetails[k] != v {
				t.Logf("seed=%d: field %q mismatch: got %q, want %q", seed, k, gotDetails[k], v)
				return false
			}
		}

		// Check no extra fields appeared
		for k := range gotDetails {
			if _, exists := details[k]; !exists {
				t.Logf("seed=%d: unexpected field %q in response", seed, k)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Check Property 2 (支票收款信息保存读取往返一致性) failed: %v", err)
	}
}
