package main

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: paid-pack-encryption, Property 2: 付费包上传后存储为加密状态
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 5.1, 5.2**
//
// For any QAP pack uploaded with per_use or subscription pricing model,
// the stored file_data's pack.json should start with QAPENC magic header,
// and encryption_password should be a non-empty hex string of at least 64 characters.
func TestProperty2_PaidPackStoredEncrypted(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		userID := createTestUser(t)
		catID := getCategoryID(t)

		// Generate random QAP metadata
		author := randomAlphaString(r, 1, 30)
		description := randomAlphaString(r, 0, 100)
		sourceName := randomAlphaString(r, 1, 30)

		qapData := createTestQAPFileWithMetadata(t, author, description, sourceName)

		// Randomly pick per_use or subscription with valid price range
		shareModes := []string{"per_use", "subscription"}
		shareMode := shareModes[r.Intn(len(shareModes))]
		var creditsPrice int
		if shareMode == "per_use" {
			creditsPrice = r.Intn(100) + 1 // 1..100
		} else {
			creditsPrice = r.Intn(901) + 100 // 100..1000
		}

		req := createUploadRequest(t, userID, qapData, map[string]string{
			"category_id":   fmt.Sprintf("%d", catID),
			"share_mode":    shareMode,
			"credits_price": fmt.Sprintf("%d", creditsPrice),
		})
		rr := httptest.NewRecorder()
		handleUploadPack(rr, req)

		if rr.Code != http.StatusCreated {
			t.Logf("seed=%d: expected 201, got %d; body: %s", seed, rr.Code, rr.Body.String())
			return false
		}

		var listing PackListingInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &listing); err != nil {
			t.Logf("seed=%d: failed to unmarshal response: %v", seed, err)
			return false
		}

		// Check encryption_password is non-empty, valid hex, at least 64 chars
		var encPwd string
		if err := db.QueryRow("SELECT encryption_password FROM pack_listings WHERE id = ?", listing.ID).Scan(&encPwd); err != nil {
			t.Logf("seed=%d: failed to query encryption_password: %v", seed, err)
			return false
		}
		if encPwd == "" {
			t.Logf("seed=%d: encryption_password is empty for paid pack (share_mode=%s)", seed, shareMode)
			return false
		}
		if len(encPwd) < 64 {
			t.Logf("seed=%d: encryption_password too short: %d chars", seed, len(encPwd))
			return false
		}
		if _, err := hex.DecodeString(encPwd); err != nil {
			t.Logf("seed=%d: encryption_password is not valid hex: %v", seed, err)
			return false
		}

		// Check stored file_data's pack.json starts with QAPENC
		var storedFileData []byte
		if err := db.QueryRow("SELECT file_data FROM pack_listings WHERE id = ?", listing.ID).Scan(&storedFileData); err != nil {
			t.Logf("seed=%d: failed to query file_data: %v", seed, err)
			return false
		}

		packJSON, err := extractPackJSONFromZip(storedFileData)
		if err != nil {
			t.Logf("seed=%d: failed to extract pack.json from stored ZIP: %v", seed, err)
			return false
		}

		if !strings.HasPrefix(string(packJSON), serverEncryptionMagic) {
			t.Logf("seed=%d: stored pack.json does not start with QAPENC (share_mode=%s)", seed, shareMode)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (付费包上传后存储为加密状态) failed: %v", err)
	}
}

// Feature: paid-pack-encryption, Property 3: 免费包上传后存储为明文状态
// **Validates: Requirements 2.5**
//
// For any QAP pack uploaded with free pricing model,
// the stored file_data's pack.json should NOT start with QAPENC magic header,
// and encryption_password should be empty string.
func TestProperty3_FreePackStoredPlaintext(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		userID := createTestUser(t)
		catID := getCategoryID(t)

		// Generate random QAP metadata
		author := randomAlphaString(r, 1, 30)
		description := randomAlphaString(r, 0, 100)
		sourceName := randomAlphaString(r, 1, 30)

		qapData := createTestQAPFileWithMetadata(t, author, description, sourceName)

		req := createUploadRequest(t, userID, qapData, map[string]string{
			"category_id": fmt.Sprintf("%d", catID),
			"share_mode":  "free",
		})
		rr := httptest.NewRecorder()
		handleUploadPack(rr, req)

		if rr.Code != http.StatusCreated {
			t.Logf("seed=%d: expected 201, got %d; body: %s", seed, rr.Code, rr.Body.String())
			return false
		}

		var listing PackListingInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &listing); err != nil {
			t.Logf("seed=%d: failed to unmarshal response: %v", seed, err)
			return false
		}

		// Check encryption_password is empty
		var encPwd string
		if err := db.QueryRow("SELECT encryption_password FROM pack_listings WHERE id = ?", listing.ID).Scan(&encPwd); err != nil {
			t.Logf("seed=%d: failed to query encryption_password: %v", seed, err)
			return false
		}
		if encPwd != "" {
			t.Logf("seed=%d: encryption_password should be empty for free pack, got %q", seed, encPwd)
			return false
		}

		// Check stored file_data's pack.json does NOT start with QAPENC
		var storedFileData []byte
		if err := db.QueryRow("SELECT file_data FROM pack_listings WHERE id = ?", listing.ID).Scan(&storedFileData); err != nil {
			t.Logf("seed=%d: failed to query file_data: %v", seed, err)
			return false
		}

		packJSON, err := extractPackJSONFromZip(storedFileData)
		if err != nil {
			t.Logf("seed=%d: failed to extract pack.json from stored ZIP: %v", seed, err)
			return false
		}

		if strings.HasPrefix(string(packJSON), serverEncryptionMagic) {
			t.Logf("seed=%d: stored pack.json should NOT start with QAPENC for free pack", seed)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (免费包上传后存储为明文状态) failed: %v", err)
	}
}

// --- Helpers ---

// randomAlphaString generates a random alphanumeric string with length between minLen and maxLen.
func randomAlphaString(r *rand.Rand, minLen, maxLen int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := minLen
	if maxLen > minLen {
		length = r.Intn(maxLen-minLen+1) + minLen
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// extractPackJSONFromZip reads the pack.json entry from a ZIP byte slice.
func extractPackJSONFromZip(zipData []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	for _, f := range zr.File {
		if f.Name == "pack.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open pack.json: %w", err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("read pack.json: %w", err)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("pack.json not found in ZIP")
}
