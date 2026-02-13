package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testUserCounter is used to generate unique auth_id values in tests.
var testUserCounter int64

// createTestUserWithBalance inserts a test user with a specific credits balance and returns the user ID.
func createTestUserWithBalance(t *testing.T, balance float64) int64 {
	t.Helper()
	testUserCounter++
	result, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES ('google', ?, 'Downloader', 'dl@test.com', ?)",
		fmt.Sprintf("dl-user-%d-%f", testUserCounter, balance), balance,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

// createTestPackListing inserts a pack listing directly into the DB and returns the listing ID.
func createTestPackListing(t *testing.T, userID, categoryID int64, shareMode string, creditsPrice int, fileData []byte) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status)
		 VALUES (?, ?, ?, 'TestPack', 'A test pack', 'TestSource', 'TestAuthor', ?, ?, 'published')`,
		userID, categoryID, fileData, shareMode, creditsPrice,
	)
	if err != nil {
		t.Fatalf("failed to create test pack listing: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

func makeDownloadRequest(t *testing.T, packID, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	url := fmt.Sprintf("/api/packs/%d/download", packID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	rr := httptest.NewRecorder()
	handleDownloadPack(rr, req)
	return rr
}

func TestDownloadPack_FreePack_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")
	packID := createTestPackListing(t, userID, catID, "free", 0, qapData)

	rr := makeDownloadRequest(t, packID, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	if rr.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("expected Content-Type 'application/octet-stream', got %q", rr.Header().Get("Content-Type"))
	}

	if !bytes.Equal(rr.Body.Bytes(), qapData) {
		t.Error("response body does not match original file data")
	}
}

func TestDownloadPack_FreePack_ZeroBalance(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")
	packID := createTestPackListing(t, userID, catID, "free", 0, qapData)

	rr := makeDownloadRequest(t, packID, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("free pack should succeed even with zero balance, got %d", rr.Code)
	}
}

func TestDownloadPack_PaidPack_SufficientBalance(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 500)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Premium pack", "Source")
	packID := createTestPackListing(t, userID, catID, "paid", 100, qapData)

	rr := makeDownloadRequest(t, packID, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	if !bytes.Equal(rr.Body.Bytes(), qapData) {
		t.Error("response body does not match original file data")
	}

	// Verify balance was deducted
	var balance float64
	err := db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		t.Fatalf("failed to query balance: %v", err)
	}
	if balance != 400 {
		t.Errorf("expected balance=400 after deduction, got %v", balance)
	}
}

func TestDownloadPack_PaidPack_ExactBalance(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 100)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")
	packID := createTestPackListing(t, userID, catID, "paid", 100, qapData)

	rr := makeDownloadRequest(t, packID, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with exact balance, got %d; body: %s", rr.Code, rr.Body.String())
	}

	// Verify balance is now 0
	var balance float64
	db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if balance != 0 {
		t.Errorf("expected balance=0 after exact deduction, got %v", balance)
	}
}

func TestDownloadPack_PaidPack_InsufficientBalance(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 50)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")
	packID := createTestPackListing(t, userID, catID, "paid", 100, qapData)

	rr := makeDownloadRequest(t, packID, userID)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "INSUFFICIENT_CREDITS" {
		t.Errorf("expected error='INSUFFICIENT_CREDITS', got %v", resp["error"])
	}
	if resp["required"] != float64(100) {
		t.Errorf("expected required=100, got %v", resp["required"])
	}
	if resp["balance"] != float64(50) {
		t.Errorf("expected balance=50, got %v", resp["balance"])
	}

	// Verify balance unchanged
	var balance float64
	db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if balance != 50 {
		t.Errorf("expected balance unchanged at 50, got %v", balance)
	}
}

func TestDownloadPack_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 100)

	rr := makeDownloadRequest(t, 99999, userID)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDownloadPack_UnpublishedPack(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 100)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")
	packID := createTestPackListing(t, userID, catID, "free", 0, qapData)

	// Set pack to unpublished
	db.Exec("UPDATE pack_listings SET status = 'unpublished' WHERE id = ?", packID)

	rr := makeDownloadRequest(t, packID, userID)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unpublished pack, got %d", rr.Code)
	}
}

func TestDownloadPack_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/packs/1/download", nil)
	rr := httptest.NewRecorder()
	handleDownloadPack(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestDownloadPack_InvalidPackID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/packs/abc/download", nil)
	req.Header.Set("X-User-ID", "1")
	rr := httptest.NewRecorder()
	handleDownloadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid pack ID, got %d", rr.Code)
	}
}

func TestDownloadPack_PaidPack_TransactionRecorded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 500)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")
	packID := createTestPackListing(t, userID, catID, "paid", 100, qapData)

	rr := makeDownloadRequest(t, packID, userID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Verify transaction was recorded
	var txType string
	var amount float64
	var listingID int64
	err := db.QueryRow(
		"SELECT transaction_type, amount, listing_id FROM credits_transactions WHERE user_id = ? AND listing_id = ?",
		userID, packID,
	).Scan(&txType, &amount, &listingID)
	if err != nil {
		t.Fatalf("failed to query transaction: %v", err)
	}
	if txType != "download" {
		t.Errorf("expected transaction_type='download', got %q", txType)
	}
	if amount != -100 {
		t.Errorf("expected amount=-100, got %v", amount)
	}
}

func TestDownloadPack_DownloadCountIncremented(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")
	packID := createTestPackListing(t, userID, catID, "free", 0, qapData)

	// Download twice
	makeDownloadRequest(t, packID, userID)
	makeDownloadRequest(t, packID, userID)

	var count int
	err := db.QueryRow("SELECT download_count FROM pack_listings WHERE id = ?", packID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query download_count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected download_count=2, got %d", count)
	}
}

func TestDownloadPack_ContentDispositionHeader(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUserWithBalance(t, 0)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "MySource")
	packID := createTestPackListing(t, userID, catID, "free", 0, qapData)

	rr := makeDownloadRequest(t, packID, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	disposition := rr.Header().Get("Content-Disposition")
	expected := `attachment; filename="TestPack.qap"`
	if disposition != expected {
		t.Errorf("expected Content-Disposition=%q, got %q", expected, disposition)
	}
}
