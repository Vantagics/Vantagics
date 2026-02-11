package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Helper to make an authenticated GET request to the balance endpoint
func getBalance(t *testing.T, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/credits/balance", nil)
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	handleGetBalance(rr, req)
	return rr
}

// Helper to make an authenticated POST request to the purchase endpoint
func purchaseCredits(t *testing.T, userID int64, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/credits/purchase", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	handlePurchaseCredits(rr, req)
	return rr
}

// Helper to make an authenticated GET request to the transactions endpoint
func getTransactions(t *testing.T, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/credits/transactions", nil)
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	handleListTransactions(rr, req)
	return rr
}

// --- Balance endpoint tests ---

func TestGetBalance_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 250.5)
	rr := getBalance(t, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["credits_balance"] != 250.5 {
		t.Errorf("expected credits_balance=250.5, got %v", resp["credits_balance"])
	}
}

func TestGetBalance_ZeroBalance(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	rr := getBalance(t, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["credits_balance"] != float64(0) {
		t.Errorf("expected credits_balance=0, got %v", resp["credits_balance"])
	}
}

func TestGetBalance_UserNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := getBalance(t, 99999)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetBalance_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/credits/balance", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	handleGetBalance(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

// --- Purchase endpoint tests ---

func TestPurchaseCredits_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 100)
	rr := purchaseCredits(t, userID, map[string]interface{}{"amount": 50.0})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["credits_balance"] != 150.0 {
		t.Errorf("expected credits_balance=150, got %v", resp["credits_balance"])
	}
	if resp["amount_added"] != 50.0 {
		t.Errorf("expected amount_added=50, got %v", resp["amount_added"])
	}

	// Verify actual DB balance
	var balance float64
	db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if balance != 150 {
		t.Errorf("expected DB balance=150, got %v", balance)
	}
}

func TestPurchaseCredits_ZeroAmount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 100)
	rr := purchaseCredits(t, userID, map[string]interface{}{"amount": 0})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for zero amount, got %d", rr.Code)
	}
}

func TestPurchaseCredits_NegativeAmount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 100)
	rr := purchaseCredits(t, userID, map[string]interface{}{"amount": -50})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative amount, got %d", rr.Code)
	}

	// Verify balance unchanged
	var balance float64
	db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if balance != 100 {
		t.Errorf("expected balance unchanged at 100, got %v", balance)
	}
}

func TestPurchaseCredits_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 100)
	req := httptest.NewRequest(http.MethodPost, "/api/credits/purchase", bytes.NewReader([]byte("not json")))
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	handlePurchaseCredits(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestPurchaseCredits_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/credits/purchase", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	handlePurchaseCredits(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestPurchaseCredits_UserNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := purchaseCredits(t, 99999, map[string]interface{}{"amount": 50})

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestPurchaseCredits_TransactionRecorded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	rr := purchaseCredits(t, userID, map[string]interface{}{"amount": 75.5})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Verify transaction was recorded
	var txType string
	var amount float64
	var description string
	err := db.QueryRow(
		"SELECT transaction_type, amount, description FROM credits_transactions WHERE user_id = ? AND transaction_type = 'purchase'",
		userID,
	).Scan(&txType, &amount, &description)
	if err != nil {
		t.Fatalf("failed to query transaction: %v", err)
	}
	if txType != "purchase" {
		t.Errorf("expected transaction_type='purchase', got %q", txType)
	}
	if amount != 75.5 {
		t.Errorf("expected amount=75.5, got %v", amount)
	}
}

func TestPurchaseCredits_MultiplePurchases(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)

	purchaseCredits(t, userID, map[string]interface{}{"amount": 100})
	purchaseCredits(t, userID, map[string]interface{}{"amount": 50})
	purchaseCredits(t, userID, map[string]interface{}{"amount": 25})

	// Verify final balance
	var balance float64
	db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if balance != 175 {
		t.Errorf("expected balance=175 after 3 purchases, got %v", balance)
	}

	// Verify 3 transactions recorded
	var count int
	db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ? AND transaction_type = 'purchase'", userID).Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 purchase transactions, got %d", count)
	}
}

// --- Transactions endpoint tests ---

func TestListTransactions_Empty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	rr := getTransactions(t, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	txns := resp["transactions"].([]interface{})
	if len(txns) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(txns))
	}
}

func TestListTransactions_AfterPurchase(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	purchaseCredits(t, userID, map[string]interface{}{"amount": 100})

	rr := getTransactions(t, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	txns := resp["transactions"].([]interface{})
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}

	tx := txns[0].(map[string]interface{})
	if tx["transaction_type"] != "purchase" {
		t.Errorf("expected transaction_type='purchase', got %v", tx["transaction_type"])
	}
	if tx["amount"] != 100.0 {
		t.Errorf("expected amount=100, got %v", tx["amount"])
	}
	if tx["user_id"] != float64(userID) {
		t.Errorf("expected user_id=%d, got %v", userID, tx["user_id"])
	}
}

func TestListTransactions_OrderedByCreatedAtDesc(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	purchaseCredits(t, userID, map[string]interface{}{"amount": 10})
	purchaseCredits(t, userID, map[string]interface{}{"amount": 20})
	purchaseCredits(t, userID, map[string]interface{}{"amount": 30})

	rr := getTransactions(t, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	txns := resp["transactions"].([]interface{})
	if len(txns) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(txns))
	}

	// Most recent (30) should be first
	first := txns[0].(map[string]interface{})
	last := txns[2].(map[string]interface{})
	if first["amount"] != 30.0 {
		t.Errorf("expected first transaction amount=30, got %v", first["amount"])
	}
	if last["amount"] != 10.0 {
		t.Errorf("expected last transaction amount=10, got %v", last["amount"])
	}
}

func TestListTransactions_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/credits/transactions", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	handleListTransactions(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestListTransactions_IsolatedPerUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user1 := createTestUserWithBalance(t, 0)
	user2 := createTestUserWithBalance(t, 0)

	purchaseCredits(t, user1, map[string]interface{}{"amount": 100})
	purchaseCredits(t, user1, map[string]interface{}{"amount": 200})
	purchaseCredits(t, user2, map[string]interface{}{"amount": 50})

	// User 1 should see 2 transactions
	rr1 := getTransactions(t, user1)
	var resp1 map[string]interface{}
	json.Unmarshal(rr1.Body.Bytes(), &resp1)
	txns1 := resp1["transactions"].([]interface{})
	if len(txns1) != 2 {
		t.Errorf("expected 2 transactions for user1, got %d", len(txns1))
	}

	// User 2 should see 1 transaction
	rr2 := getTransactions(t, user2)
	var resp2 map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &resp2)
	txns2 := resp2["transactions"].([]interface{})
	if len(txns2) != 1 {
		t.Errorf("expected 1 transaction for user2, got %d", len(txns2))
	}
}

// --- Integration: purchase + balance + transactions ---

func TestCredits_PurchaseUpdatesBalanceAndTransactions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 50)

	// Purchase credits
	rr := purchaseCredits(t, userID, map[string]interface{}{"amount": 200})
	if rr.Code != http.StatusOK {
		t.Fatalf("purchase failed: %d; body: %s", rr.Code, rr.Body.String())
	}

	// Check balance via API
	balRR := getBalance(t, userID)
	var balResp map[string]interface{}
	json.Unmarshal(balRR.Body.Bytes(), &balResp)
	if balResp["credits_balance"] != 250.0 {
		t.Errorf("expected balance=250, got %v", balResp["credits_balance"])
	}

	// Check transactions via API
	txRR := getTransactions(t, userID)
	var txResp map[string]interface{}
	json.Unmarshal(txRR.Body.Bytes(), &txResp)
	txns := txResp["transactions"].([]interface{})
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
	tx := txns[0].(map[string]interface{})
	if tx["transaction_type"] != "purchase" {
		t.Errorf("expected 'purchase', got %v", tx["transaction_type"])
	}
	if tx["amount"] != 200.0 {
		t.Errorf("expected amount=200, got %v", tx["amount"])
	}
}
