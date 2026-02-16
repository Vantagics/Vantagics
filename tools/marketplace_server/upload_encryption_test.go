package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createTestQAPFileWithMetadata creates a .qap ZIP with both pack.json and metadata.json.
func createTestQAPFileWithMetadata(t *testing.T, author, description, sourceName string) []byte {
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

	fw, _ := zw.Create("pack.json")
	fw.Write(jsonData)

	// Also write metadata.json
	meta := map[string]string{
		"author":      author,
		"created_at":  "2024-01-01T00:00:00Z",
		"source_name": sourceName,
		"description": description,
	}
	metaData, _ := json.Marshal(meta)
	mw, _ := zw.Create("metadata.json")
	mw.Write(metaData)

	zw.Close()
	return buf.Bytes()
}

// createPreEncryptedQAPFile creates a .qap ZIP where pack.json starts with QAPENC magic header.
func createPreEncryptedQAPFile(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Write a fake encrypted pack.json (starts with QAPENC)
	fw, _ := zw.Create("pack.json")
	fw.Write([]byte("QAPENC" + "fake-encrypted-data-here"))

	// Write metadata.json so the handler can still parse metadata
	meta := map[string]string{
		"author":      "TestAuthor",
		"created_at":  "2024-01-01T00:00:00Z",
		"source_name": "TestSource",
		"description": "Pre-encrypted test",
	}
	metaData, _ := json.Marshal(meta)
	mw, _ := zw.Create("metadata.json")
	mw.Write(metaData)

	zw.Close()
	return buf.Bytes()
}

func TestUploadPack_PaidPreEncrypted_Rejected(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createPreEncryptedQAPFile(t)

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id":   fmt.Sprintf("%d", catID),
		"share_mode":    "per_use",
		"credits_price": "50",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for pre-encrypted paid pack, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "paid packs must not be pre-encrypted" {
		t.Errorf("expected error='paid packs must not be pre-encrypted', got %q", resp["error"])
	}
}

func TestUploadPack_PaidEncryptsFileData(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFileWithMetadata(t, "Alice", "Premium pack", "PremiumStore")

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

	// Verify encryption_password is stored and non-empty
	var encPwd string
	err := db.QueryRow("SELECT encryption_password FROM pack_listings WHERE id = ?", listing.ID).Scan(&encPwd)
	if err != nil {
		t.Fatalf("failed to query encryption_password: %v", err)
	}
	if encPwd == "" {
		t.Error("expected non-empty encryption_password for paid pack")
	}
	if len(encPwd) < 64 {
		t.Errorf("expected encryption_password >= 64 chars, got %d", len(encPwd))
	}

	// Verify file_data contains encrypted pack.json (starts with QAPENC)
	var storedFileData []byte
	err = db.QueryRow("SELECT file_data FROM pack_listings WHERE id = ?", listing.ID).Scan(&storedFileData)
	if err != nil {
		t.Fatalf("failed to query file_data: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(storedFileData), int64(len(storedFileData)))
	if err != nil {
		t.Fatalf("stored file_data is not a valid ZIP: %v", err)
	}

	for _, f := range zr.File {
		if f.Name == "pack.json" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			if len(data) < 6 || string(data[:6]) != "QAPENC" {
				t.Error("expected pack.json in stored ZIP to start with QAPENC magic header")
			}
			break
		}
	}
}

func TestUploadPack_FreeNoEncryption(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFileWithMetadata(t, "Bob", "Free pack", "FreeStore")

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

	// Verify encryption_password is empty for free pack
	var encPwd string
	err := db.QueryRow("SELECT encryption_password FROM pack_listings WHERE id = ?", listing.ID).Scan(&encPwd)
	if err != nil {
		t.Fatalf("failed to query encryption_password: %v", err)
	}
	if encPwd != "" {
		t.Errorf("expected empty encryption_password for free pack, got %q", encPwd)
	}

	// Verify file_data is NOT encrypted (pack.json should not start with QAPENC)
	var storedFileData []byte
	err = db.QueryRow("SELECT file_data FROM pack_listings WHERE id = ?", listing.ID).Scan(&storedFileData)
	if err != nil {
		t.Fatalf("failed to query file_data: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(storedFileData), int64(len(storedFileData)))
	if err != nil {
		t.Fatalf("stored file_data is not a valid ZIP: %v", err)
	}

	for _, f := range zr.File {
		if f.Name == "pack.json" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			if len(data) >= 6 && string(data[:6]) == "QAPENC" {
				t.Error("expected pack.json in free pack to NOT be encrypted")
			}
			break
		}
	}
}

func TestUploadPack_SubscriptionEncryptsFileData(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	userID := createTestUser(t)
	catID := getCategoryID(t)
	qapData := createTestQAPFileWithMetadata(t, "Carol", "Sub pack", "SubStore")

	req := createUploadRequest(t, userID, qapData, map[string]string{
		"category_id":   fmt.Sprintf("%d", catID),
		"share_mode":    "subscription",
		"credits_price": "200",
	})
	rr := httptest.NewRecorder()
	handleUploadPack(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var listing PackListingInfo
	json.Unmarshal(rr.Body.Bytes(), &listing)

	// Verify encryption_password is stored
	var encPwd string
	err := db.QueryRow("SELECT encryption_password FROM pack_listings WHERE id = ?", listing.ID).Scan(&encPwd)
	if err != nil {
		t.Fatalf("failed to query encryption_password: %v", err)
	}
	if encPwd == "" {
		t.Error("expected non-empty encryption_password for subscription pack")
	}
	if len(encPwd) < 64 {
		t.Errorf("expected encryption_password >= 64 chars, got %d", len(encPwd))
	}
}
