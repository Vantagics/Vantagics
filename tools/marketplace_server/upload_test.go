package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createTestQAPFile creates a valid .qap ZIP file with the given metadata for testing.
func createTestQAPFile(t *testing.T, author, description, sourceName string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	content := map[string]interface{}{
		"file_type":      "VantageData_QuickAnalysisPack",
		"format_version": "1.0",
		"metadata": map[string]string{
			"author":      author,
			"created_at":  "2024-01-01T00:00:00Z",
			"source_name": sourceName,
			"description": description,
		},
		"schema_requirements": []interface{}{},
		"executable_steps":    []interface{}{},
	}
	jsonData, _ := json.Marshal(content)

	fw, err := zw.Create("analysis_pack.json")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	fw.Write(jsonData)
	zw.Close()
	return buf.Bytes()
}

// createUploadRequest builds a multipart/form-data request for the upload endpoint.
func createUploadRequest(t *testing.T, userID int64, fileData []byte, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if fileData != nil {
		part, err := writer.CreateFormFile("file", "test.qap")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		io.Copy(part, bytes.NewReader(fileData))
	}

	for k, v := range fields {
		writer.WriteField(k, v)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/packs/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	return req
}

// createTestUser inserts a test user and returns the user ID.
func createTestUser(t *testing.T) int64 {
	t.Helper()
	result, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email) VALUES ('google', 'upload-test-user', 'Uploader', 'up@test.com')",
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

// getCategoryID returns the ID of the first preset category (Shopify).
func getCategoryID(t *testing.T) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow("SELECT id FROM categories WHERE name = 'Shopify'").Scan(&id)
	if err != nil {
		t.Fatalf("failed to get Shopify category ID: %v", err)
	}
	return id
}

func TestUploadPack_Success_Free(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Alice", "Sales analysis pack", "ShopifyStore")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var listing PackListingInfo
	json.Unmarshal(rr.Body.Bytes(), &listing)

	if listing.PackName != "ShopifyStore" {
		t.Errorf("expected pack_name='ShopifyStore', got %q", listing.PackName)
	}
	if listing.PackDescription != "Sales analysis pack" {
		t.Errorf("expected description='Sales analysis pack', got %q", listing.PackDescription)
	}
	if listing.AuthorName != "Alice" {
		t.Errorf("expected author='Alice', got %q", listing.AuthorName)
	}
	if listing.SourceName != "ShopifyStore" {
		t.Errorf("expected source_name='ShopifyStore', got %q", listing.SourceName)
	}
	if listing.ShareMode != "free" {
		t.Errorf("expected share_mode='free', got %q", listing.ShareMode)
	}
	if listing.CreditsPrice != 0 {
		t.Errorf("expected credits_price=0, got %d", listing.CreditsPrice)
	}
	if listing.CategoryName != "Shopify" {
		t.Errorf("expected category_name='Shopify', got %q", listing.CategoryName)
	}
	if listing.UserID != userID {
		t.Errorf("expected user_id=%d, got %d", userID, listing.UserID)
	}
}

func TestUploadPack_Success_Paid(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Bob", "Premium pack", "EtsyShop")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id":   fmt.Sprintf("%d", catID),
		"share_mode":    "per_use",
		"credits_price": "100",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var listing PackListingInfo
	json.Unmarshal(rr.Body.Bytes(), &listing)

	if listing.ShareMode != "per_use" {
		t.Errorf("expected share_mode='per_use', got %q", listing.ShareMode)
	}
	if listing.CreditsPrice != 100 {
		t.Errorf("expected credits_price=100, got %d", listing.CreditsPrice)
	}
}

func TestUploadPack_MissingShareMode(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Test", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestUploadPack_InvalidShareMode(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Test", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "unknown",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestUploadPack_PaidMissingPrice(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Test", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "per_use",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestUploadPack_PaidZeroPrice(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Test", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id":   fmt.Sprintf("%d", catID),
		"share_mode":    "per_use",
		"credits_price": "0",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for zero price, got %d", rr.Code)
	}
}

func TestUploadPack_PaidNegativePrice(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Test", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id":   fmt.Sprintf("%d", catID),
		"share_mode":    "per_use",
		"credits_price": "-5",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative price, got %d", rr.Code)
	}
}

func TestUploadPack_MissingCategoryID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	qapData := createTestQAPFile(t, "Test", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"share_mode": "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestUploadPack_NonexistentCategory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	qapData := createTestQAPFile(t, "Test", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": "99999",
		"share_mode":  "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestUploadPack_InvalidZipFile(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)

	req := createUploadRequest(t, userID, []byte("not a zip file"), map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "invalid_pack_format" {
		t.Errorf("expected error='invalid_pack_format', got %q", resp["error"])
	}
}

func TestUploadPack_ZipWithoutJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)

	// Create a valid ZIP but without analysis_pack.json
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("other_file.txt")
	fw.Write([]byte("hello"))
	zw.Close()

	req := createUploadRequest(t, userID, buf.Bytes(), map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "invalid_pack_format" {
		t.Errorf("expected error='invalid_pack_format', got %q", resp["error"])
	}
}

func TestUploadPack_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/packs/upload", nil)
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestUploadPack_EmptySourceName_FallsBackToUntitled(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Description", "")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var listing PackListingInfo
	json.Unmarshal(rr.Body.Bytes(), &listing)

	if listing.PackName != "Untitled" {
		t.Errorf("expected pack_name='Untitled' for empty source_name, got %q", listing.PackName)
	}
}

func TestUploadPack_StatusIsPending(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var listing PackListingInfo
	json.Unmarshal(rr.Body.Bytes(), &listing)

	// Verify status in DB directly - should be 'pending' after admin review feature
	var status string
	err := db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listing.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to query status: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected status='pending', got %q", status)
	}
}

func TestUploadPack_FileDataStoredInDB(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFile(t, "Author", "Desc", "Source")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id": fmt.Sprintf("%d", catID),
		"share_mode":  "free",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var listing PackListingInfo
	json.Unmarshal(rr.Body.Bytes(), &listing)

	// Verify file_data is stored
	var fileData []byte
	err := db.QueryRow("SELECT file_data FROM pack_listings WHERE id = ?", listing.ID).Scan(&fileData)
	if err != nil {
		t.Fatalf("failed to query file_data: %v", err)
	}
	if len(fileData) == 0 {
		t.Error("expected non-empty file_data in DB")
	}
	if !bytes.Equal(fileData, qapData) {
		t.Error("stored file_data does not match uploaded data")
	}
}
