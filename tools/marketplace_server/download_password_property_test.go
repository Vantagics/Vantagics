package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"
)

// Feature: paid-pack-encryption, Property 4: 下载响应中的密码头与定价模型一致
// **Validates: Requirements 3.1, 3.2**
//
// For any paid pack (per_use or subscription) download request, the HTTP response
// should contain X-Encryption-Password header with a value matching the database-stored
// password. For any free pack download request, the response should NOT contain
// X-Encryption-Password header.
func TestProperty4_DownloadPasswordHeaderMatchesPricingModel(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		catID := getCategoryID(t)

		// Generate random QAP metadata
		author := randomAlphaString(r, 1, 30)
		description := randomAlphaString(r, 0, 100)
		sourceName := randomAlphaString(r, 1, 30)

		qapData := createTestQAPFileWithMetadata(t, author, description, sourceName)

		// Randomly pick a pricing model: free, per_use, or subscription
		allModes := []string{"free", "per_use", "subscription"}
		shareMode := allModes[r.Intn(len(allModes))]

		var creditsPrice int
		switch shareMode {
		case "per_use":
			creditsPrice = r.Intn(100) + 1 // 1..100
		case "subscription":
			creditsPrice = r.Intn(901) + 100 // 100..1000
		default:
			creditsPrice = 0
		}

		// Create uploader user
		uploaderID := createTestUser(t)

		// Upload the pack via handleUploadPack (to get proper server-side encryption)
		fields := map[string]string{
			"category_id": fmt.Sprintf("%d", catID),
			"share_mode":  shareMode,
		}
		if creditsPrice > 0 {
			fields["credits_price"] = fmt.Sprintf("%d", creditsPrice)
		}

		uploadReq := createUploadRequest(t, uploaderID, qapData, fields)
		uploadRR := httptest.NewRecorder()
		handleUploadPack(uploadRR, uploadReq)

		if uploadRR.Code != http.StatusCreated {
			t.Logf("seed=%d: upload expected 201, got %d; body: %s", seed, uploadRR.Code, uploadRR.Body.String())
			return false
		}

		var listing PackListingInfo
		if err := json.Unmarshal(uploadRR.Body.Bytes(), &listing); err != nil {
			t.Logf("seed=%d: failed to unmarshal upload response: %v", seed, err)
			return false
		}

		// Publish the pack so it can be downloaded
		if _, err := db.Exec("UPDATE pack_listings SET status = 'published' WHERE id = ?", listing.ID); err != nil {
			t.Logf("seed=%d: failed to publish pack: %v", seed, err)
			return false
		}

		// Read the stored encryption_password from DB
		var storedPassword string
		if err := db.QueryRow("SELECT encryption_password FROM pack_listings WHERE id = ?", listing.ID).Scan(&storedPassword); err != nil {
			t.Logf("seed=%d: failed to query encryption_password: %v", seed, err)
			return false
		}

		// Create a downloader user with sufficient balance
		downloaderID := createTestUserWithBalance(t, 10000)

		// Download via handleDownloadPack
		downloadRR := makeDownloadRequest(t, listing.ID, downloaderID)

		if downloadRR.Code != http.StatusOK {
			t.Logf("seed=%d: download expected 200, got %d; body: %s", seed, downloadRR.Code, downloadRR.Body.String())
			return false
		}

		gotHeader := downloadRR.Header().Get("X-Encryption-Password")

		switch shareMode {
		case "per_use", "subscription":
			// Paid pack: header must be present and match stored password
			if gotHeader == "" {
				t.Logf("seed=%d: paid pack (share_mode=%s) missing X-Encryption-Password header", seed, shareMode)
				return false
			}
			if gotHeader != storedPassword {
				t.Logf("seed=%d: paid pack (share_mode=%s) X-Encryption-Password mismatch: header=%q, stored=%q", seed, shareMode, gotHeader, storedPassword)
				return false
			}
		case "free":
			// Free pack: header must NOT be present
			if gotHeader != "" {
				t.Logf("seed=%d: free pack should NOT have X-Encryption-Password header, got %q", seed, gotHeader)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 (下载响应中的密码头与定价模型一致) failed: %v", err)
	}
}
