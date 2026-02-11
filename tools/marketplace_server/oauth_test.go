package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// setupTestDB creates a temporary in-memory database for testing.
func setupTestDB(t *testing.T) func() {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "marketplace_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	testDB, err := initDB(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to init test db: %v", err)
	}
	db = testDB

	return func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}
}

func postOAuthJSON(t *testing.T, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/oauth", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleOAuthCallback(rr, req)
	return rr
}

func TestOAuthCallback_InvalidMethod(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth", nil)
	rr := httptest.NewRecorder()
	handleOAuthCallback(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestOAuthCallback_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oauth", bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()
	handleOAuthCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestOAuthCallback_UnsupportedProvider(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postOAuthJSON(t, oauthCallbackRequest{
		Provider:       "twitter",
		ProviderUserID: "user123",
		DisplayName:    "Test User",
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["success"] != false {
		t.Error("expected success=false")
	}
}

func TestOAuthCallback_EmptyProviderUserID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postOAuthJSON(t, oauthCallbackRequest{
		Provider:       "google",
		ProviderUserID: "",
		DisplayName:    "Test User",
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestOAuthCallback_EmptyDisplayName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postOAuthJSON(t, oauthCallbackRequest{
		Provider:       "google",
		ProviderUserID: "user123",
		DisplayName:    "",
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestOAuthCallback_NewUserCreation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postOAuthJSON(t, oauthCallbackRequest{
		Provider:       "google",
		ProviderUserID: "google-user-1",
		DisplayName:    "Alice",
		Email:          "alice@example.com",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty token")
	}

	user := resp["user"].(map[string]interface{})
	if user["oauth_provider"] != "google" {
		t.Errorf("expected oauth_provider=google, got %v", user["oauth_provider"])
	}
	if user["display_name"] != "Alice" {
		t.Errorf("expected display_name=Alice, got %v", user["display_name"])
	}
	if user["email"] != "alice@example.com" {
		t.Errorf("expected email=alice@example.com, got %v", user["email"])
	}
	// Default initial balance is 0
	if user["credits_balance"].(float64) != 0 {
		t.Errorf("expected credits_balance=0, got %v", user["credits_balance"])
	}
}

func TestOAuthCallback_RepeatedLoginReturnsSameUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	body := oauthCallbackRequest{
		Provider:       "apple",
		ProviderUserID: "apple-user-1",
		DisplayName:    "Bob",
		Email:          "bob@example.com",
	}

	// First login
	rr1 := postOAuthJSON(t, body)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first login: expected 200, got %d", rr1.Code)
	}
	var resp1 map[string]interface{}
	json.Unmarshal(rr1.Body.Bytes(), &resp1)
	user1 := resp1["user"].(map[string]interface{})
	userID1 := user1["id"].(float64)

	// Second login with same credentials
	rr2 := postOAuthJSON(t, body)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second login: expected 200, got %d", rr2.Code)
	}
	var resp2 map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &resp2)
	user2 := resp2["user"].(map[string]interface{})
	userID2 := user2["id"].(float64)

	if userID1 != userID2 {
		t.Errorf("expected same user ID, got %v and %v", userID1, userID2)
	}

	// Verify only one user exists in DB
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE oauth_provider = 'apple' AND oauth_provider_id = 'apple-user-1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user record, got %d", count)
	}
}

func TestOAuthCallback_InitialCreditsBalance(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Set initial credits balance to 100
	_, err := db.Exec("UPDATE settings SET value = '100' WHERE key = 'initial_credits_balance'")
	if err != nil {
		t.Fatalf("failed to update settings: %v", err)
	}

	rr := postOAuthJSON(t, oauthCallbackRequest{
		Provider:       "facebook",
		ProviderUserID: "fb-user-1",
		DisplayName:    "Charlie",
		Email:          "charlie@example.com",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	user := resp["user"].(map[string]interface{})

	if user["credits_balance"].(float64) != 100 {
		t.Errorf("expected credits_balance=100, got %v", user["credits_balance"])
	}

	// Verify initial credits transaction was recorded
	var txCount int
	userID := int64(user["id"].(float64))
	db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ? AND transaction_type = 'initial'", userID).Scan(&txCount)
	if txCount != 1 {
		t.Errorf("expected 1 initial credits transaction, got %d", txCount)
	}
}

func TestOAuthCallback_AllProviders(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	providers := []string{"google", "apple", "facebook", "amazon"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			rr := postOAuthJSON(t, oauthCallbackRequest{
				Provider:       provider,
				ProviderUserID: provider + "-user-1",
				DisplayName:    "User " + provider,
				Email:          provider + "@example.com",
			})

			if rr.Code != http.StatusOK {
				t.Errorf("expected 200 for provider %s, got %d; body: %s", provider, rr.Code, rr.Body.String())
			}

			var resp map[string]interface{}
			json.Unmarshal(rr.Body.Bytes(), &resp)
			if resp["success"] != true {
				t.Errorf("expected success=true for provider %s", provider)
			}
		})
	}
}

func TestOAuthCallback_JWTTokenIsValid(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postOAuthJSON(t, oauthCallbackRequest{
		Provider:       "amazon",
		ProviderUserID: "amz-user-1",
		DisplayName:    "Diana",
		Email:          "diana@example.com",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	token := resp["token"].(string)

	// Verify the JWT token can be parsed
	userID, displayName, err := parseJWT(token)
	if err != nil {
		t.Fatalf("failed to parse JWT: %v", err)
	}
	if userID <= 0 {
		t.Errorf("expected positive userID, got %d", userID)
	}
	if displayName != "Diana" {
		t.Errorf("expected displayName=Diana, got %q", displayName)
	}
}

func TestOAuthCallback_NoInitialTransactionWhenBalanceZero(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Default balance is 0, so no transaction should be created
	rr := postOAuthJSON(t, oauthCallbackRequest{
		Provider:       "google",
		ProviderUserID: "google-zero-balance",
		DisplayName:    "Zero",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	user := resp["user"].(map[string]interface{})
	userID := int64(user["id"].(float64))

	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM credits_transactions WHERE user_id = ?", userID).Scan(&txCount)
	if txCount != 0 {
		t.Errorf("expected 0 transactions for zero balance, got %d", txCount)
	}
}

func TestGetSetting_ExistingKey(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	val := getSetting("initial_credits_balance")
	if val != "0" {
		t.Errorf("expected '0', got %q", val)
	}
}

func TestGetSetting_NonExistentKey(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	val := getSetting("nonexistent_key")
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}
