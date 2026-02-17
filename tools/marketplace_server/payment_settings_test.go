package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupPaymentTestDB creates an in-memory SQLite database with the user_payment_info table
// and swaps the global db variable. Returns a cleanup function.
func setupPaymentTestDB(t *testing.T) func() {
	t.Helper()
	tmpDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open temp db: %v", err)
	}
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
	origDB := db
	db = tmpDB
	return func() {
		db = origDB
		tmpDB.Close()
	}
}

// savePaymentInfo is a helper that POSTs payment info and returns the recorder.
func savePaymentInfo(t *testing.T, userID string, paymentType string, details map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	detailsJSON, _ := json.Marshal(details)
	body := PaymentInfo{
		PaymentType:    paymentType,
		PaymentDetails: json.RawMessage(detailsJSON),
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/user/payment-info", bytes.NewReader(bodyBytes))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleSavePaymentInfo(w, req)
	return w
}

// getPaymentInfo is a helper that GETs payment info and returns the recorder.
func getPaymentInfo(t *testing.T, userID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/user/payment-info", nil)
	req.Header.Set("X-User-ID", userID)
	w := httptest.NewRecorder()
	handleGetPaymentInfo(w, req)
	return w
}

// parsePaymentResponse parses the JSON response into payment_type and payment_details.
func parsePaymentResponse(t *testing.T, body []byte) (string, map[string]string) {
	t.Helper()
	var resp struct {
		PaymentType    string          `json:"payment_type"`
		PaymentDetails json.RawMessage `json:"payment_details"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	var details map[string]string
	if err := json.Unmarshal(resp.PaymentDetails, &details); err != nil {
		details = map[string]string{}
	}
	return resp.PaymentType, details
}

// --- Valid save and read tests for each payment type ---

// Validates: Requirements 2.1, 2.2
func TestPayPal_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "user@paypal.com", "username": "PayPalUser"}
	w := savePaymentInfo(t, "1", "paypal", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "1")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "paypal" {
		t.Errorf("payment_type: got %q, want %q", gotType, "paypal")
	}
	if gotDetails["account"] != "user@paypal.com" {
		t.Errorf("account: got %q, want %q", gotDetails["account"], "user@paypal.com")
	}
	if gotDetails["username"] != "PayPalUser" {
		t.Errorf("username: got %q, want %q", gotDetails["username"], "PayPalUser")
	}
}

// Validates: Requirements 3.1, 3.2
func TestBankCard_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"bank_name":      "Bank of China",
		"card_number":    "6222000000001234",
		"account_holder": "Zhang San",
	}
	w := savePaymentInfo(t, "2", "bank_card", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "2")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "bank_card" {
		t.Errorf("payment_type: got %q, want %q", gotType, "bank_card")
	}
	if gotDetails["bank_name"] != "Bank of China" {
		t.Errorf("bank_name: got %q, want %q", gotDetails["bank_name"], "Bank of China")
	}
	if gotDetails["card_number"] != "6222000000001234" {
		t.Errorf("card_number: got %q, want %q", gotDetails["card_number"], "6222000000001234")
	}
	if gotDetails["account_holder"] != "Zhang San" {
		t.Errorf("account_holder: got %q, want %q", gotDetails["account_holder"], "Zhang San")
	}
}

// Validates: Requirements 4.1, 4.2
func TestWeChat_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "wx_12345", "username": "WeChatUser"}
	w := savePaymentInfo(t, "3", "wechat", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "3")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "wechat" {
		t.Errorf("payment_type: got %q, want %q", gotType, "wechat")
	}
	if gotDetails["account"] != "wx_12345" {
		t.Errorf("account: got %q, want %q", gotDetails["account"], "wx_12345")
	}
	if gotDetails["username"] != "WeChatUser" {
		t.Errorf("username: got %q, want %q", gotDetails["username"], "WeChatUser")
	}
}

// Validates: Requirements 5.1, 5.2
func TestAliPay_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "ali@example.com", "username": "AliPayUser"}
	w := savePaymentInfo(t, "4", "alipay", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "4")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "alipay" {
		t.Errorf("payment_type: got %q, want %q", gotType, "alipay")
	}
	if gotDetails["account"] != "ali@example.com" {
		t.Errorf("account: got %q, want %q", gotDetails["account"], "ali@example.com")
	}
	if gotDetails["username"] != "AliPayUser" {
		t.Errorf("username: got %q, want %q", gotDetails["username"], "AliPayUser")
	}
}

// Validates: Requirements 6.1, 6.2
func TestCheck_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"full_legal_name": "张三",
		"province":        "北京市",
		"city":            "北京市",
		"district":        "朝阳区",
		"street_address":  "建国路88号SOHO现代城A座1201",
		"postal_code":     "100022",
		"phone":           "13800138000",
		"memo":            "货款结算",
	}
	w := savePaymentInfo(t, "5", "check", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "5")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "check" {
		t.Errorf("payment_type: got %q, want %q", gotType, "check")
	}
	for _, field := range []string{"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone", "memo"} {
		if gotDetails[field] != details[field] {
			t.Errorf("%s: got %q, want %q", field, gotDetails[field], details[field])
		}
	}
}


// --- Invalid input tests ---

func TestInvalidPaymentType_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "test", "username": "test"}
	w := savePaymentInfo(t, "10", "bitcoin", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEmptyPaymentType_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "test", "username": "test"}
	w := savePaymentInfo(t, "11", "", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPal_EmptyAccount_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "", "username": "user"}
	w := savePaymentInfo(t, "12", "paypal", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty account, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPal_EmptyUsername_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "user@paypal.com", "username": ""}
	w := savePaymentInfo(t, "13", "paypal", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty username, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPal_WhitespaceOnly_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"account": "   ", "username": "user"}
	w := savePaymentInfo(t, "14", "paypal", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for whitespace-only account, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBankCard_MissingCardNumber_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"bank_name": "ICBC", "account_holder": "Li Si"}
	w := savePaymentInfo(t, "15", "bank_card", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing card_number, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCheck_MissingRequiredField_Rejected(t *testing.T) {
	requiredFields := []string{"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"}
	validDetails := map[string]string{
		"full_legal_name": "张三",
		"province":        "北京市",
		"city":            "北京市",
		"district":        "朝阳区",
		"street_address":  "建国路88号",
		"postal_code":     "100022",
		"phone":           "13800138000",
	}

	for _, field := range requiredFields {
		t.Run("missing_"+field, func(t *testing.T) {
			cleanup := setupPaymentTestDB(t)
			defer cleanup()

			details := make(map[string]string)
			for k, v := range validDetails {
				details[k] = v
			}
			delete(details, field)

			w := savePaymentInfo(t, "16", "check", details)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for missing %s, got %d: %s", field, w.Code, w.Body.String())
			}
		})

		t.Run("empty_"+field, func(t *testing.T) {
			cleanup := setupPaymentTestDB(t)
			defer cleanup()

			details := make(map[string]string)
			for k, v := range validDetails {
				details[k] = v
			}
			details[field] = ""

			w := savePaymentInfo(t, "16", "check", details)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for empty %s, got %d: %s", field, w.Code, w.Body.String())
			}
		})
	}
}

func TestCheck_OldFormatAddressOnly_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{"address": "123 Main Street, Suite 100"}
	w := savePaymentInfo(t, "17", "check", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for old format (address only), got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPaymentInfo_NoRecord_ReturnsEmpty(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	wGet := getPaymentInfo(t, "99")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, _ := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "" {
		t.Errorf("expected empty payment_type for no record, got %q", gotType)
	}
}

// --- New payment type tests ---

func TestWireTransfer_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"beneficiary_name":    "Zhang San",
		"beneficiary_address": "123 Main St, Beijing, 100000, China",
		"bank_name":           "Industrial and Commercial Bank of China",
		"swift_code":          "ICBKCNBJ",
		"account_number":      "1234567890",
	}
	w := savePaymentInfo(t, "20", "wire_transfer", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "20")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "wire_transfer" {
		t.Errorf("payment_type: got %q, want %q", gotType, "wire_transfer")
	}
	if gotDetails["swift_code"] != "ICBKCNBJ" {
		t.Errorf("swift_code: got %q, want %q", gotDetails["swift_code"], "ICBKCNBJ")
	}
	if gotDetails["beneficiary_name"] != "Zhang San" {
		t.Errorf("beneficiary_name: got %q, want %q", gotDetails["beneficiary_name"], "Zhang San")
	}
}

func TestBankCardUS_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"legal_name":     "John Doe",
		"routing_number": "021000021",
		"account_number": "123456789",
		"account_type":   "checking",
	}
	w := savePaymentInfo(t, "21", "bank_card_us", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "21")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "bank_card_us" {
		t.Errorf("payment_type: got %q, want %q", gotType, "bank_card_us")
	}
	if gotDetails["routing_number"] != "021000021" {
		t.Errorf("routing_number: got %q, want %q", gotDetails["routing_number"], "021000021")
	}
	if gotDetails["account_type"] != "checking" {
		t.Errorf("account_type: got %q, want %q", gotDetails["account_type"], "checking")
	}
}

func TestBankCardEU_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"legal_name": "Hans Mueller",
		"iban":       "DE89370400440532013000",
		"bic_swift":  "COBADEFFXXX",
	}
	w := savePaymentInfo(t, "22", "bank_card_eu", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "22")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "bank_card_eu" {
		t.Errorf("payment_type: got %q, want %q", gotType, "bank_card_eu")
	}
	if gotDetails["iban"] != "DE89370400440532013000" {
		t.Errorf("iban: got %q, want %q", gotDetails["iban"], "DE89370400440532013000")
	}
}

func TestBankCardCN_ValidSaveAndRead(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"real_name":   "张三",
		"card_number": "6222000000001234567",
		"bank_branch": "中国银行北京分行朝阳支行",
	}
	w := savePaymentInfo(t, "23", "bank_card_cn", details)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	wGet := getPaymentInfo(t, "23")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
	}
	gotType, gotDetails := parsePaymentResponse(t, wGet.Body.Bytes())
	if gotType != "bank_card_cn" {
		t.Errorf("payment_type: got %q, want %q", gotType, "bank_card_cn")
	}
	if gotDetails["real_name"] != "张三" {
		t.Errorf("real_name: got %q, want %q", gotDetails["real_name"], "张三")
	}
	if gotDetails["bank_branch"] != "中国银行北京分行朝阳支行" {
		t.Errorf("bank_branch: got %q, want %q", gotDetails["bank_branch"], "中国银行北京分行朝阳支行")
	}
}

func TestWireTransfer_MissingSwiftCode_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"beneficiary_name":    "Zhang San",
		"beneficiary_address": "Beijing",
		"bank_name":           "ICBC",
		"account_number":      "123",
	}
	w := savePaymentInfo(t, "30", "wire_transfer", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing swift_code, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBankCardUS_MissingRoutingNumber_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"legal_name":     "John Doe",
		"account_number": "123",
		"account_type":   "checking",
	}
	w := savePaymentInfo(t, "31", "bank_card_us", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing routing_number, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBankCardEU_MissingIBAN_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"legal_name": "Hans Mueller",
		"bic_swift":  "COBADEFFXXX",
	}
	w := savePaymentInfo(t, "32", "bank_card_eu", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing iban, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBankCardCN_MissingBankBranch_Rejected(t *testing.T) {
	cleanup := setupPaymentTestDB(t)
	defer cleanup()

	details := map[string]string{
		"real_name":   "张三",
		"card_number": "6222000000001234567",
	}
	w := savePaymentInfo(t, "33", "bank_card_cn", details)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing bank_branch, got %d: %s", w.Code, w.Body.String())
	}
}
