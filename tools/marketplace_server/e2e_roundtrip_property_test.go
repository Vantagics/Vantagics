package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"
)

// Feature: paid-pack-encryption, Property 5: 付费包上传-下载-解密端到端往返
// **Validates: Requirements 2.2, 3.1, 3.3**
//
// For any valid plaintext QAP pack uploaded with a paid pricing model,
// downloading the encrypted file and decrypting it with the password from
// the X-Encryption-Password response header should yield the original pack.json content.
func TestProperty5_PaidPackE2ERoundTrip(t *testing.T) {
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

		// Create the QAP file
		qapData := createTestQAPFileWithMetadata(t, author, description, sourceName)

		// Extract original pack.json content before upload
		originalPackJSON, err := extractPackJSONFromZip(qapData)
		if err != nil {
			t.Logf("seed=%d: failed to extract original pack.json: %v", seed, err)
			return false
		}

		// Randomly pick per_use or subscription
		shareModes := []string{"per_use", "subscription"}
		shareMode := shareModes[r.Intn(len(shareModes))]
		var creditsPrice int
		if shareMode == "per_use" {
			creditsPrice = r.Intn(100) + 1 // 1..100
		} else {
			creditsPrice = r.Intn(901) + 100 // 100..1000
		}

		// Step 1: Upload as paid pack
		uploaderID := createTestUser(t)
		fields := map[string]string{
			"category_id":   fmt.Sprintf("%d", catID),
			"share_mode":    shareMode,
			"credits_price": fmt.Sprintf("%d", creditsPrice),
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

		// Step 2: Publish the pack
		if _, err := db.Exec("UPDATE pack_listings SET status = 'published' WHERE id = ?", listing.ID); err != nil {
			t.Logf("seed=%d: failed to publish pack: %v", seed, err)
			return false
		}

		// Step 3: Download via handleDownloadPack
		downloaderID := createTestUserWithBalance(t, 100000)
		downloadRR := makeDownloadRequest(t, listing.ID, downloaderID)

		if downloadRR.Code != http.StatusOK {
			t.Logf("seed=%d: download expected 200, got %d; body: %s", seed, downloadRR.Code, downloadRR.Body.String())
			return false
		}

		// Step 4: Extract X-Encryption-Password header
		password := downloadRR.Header().Get("X-Encryption-Password")
		if password == "" {
			t.Logf("seed=%d: paid pack (share_mode=%s) missing X-Encryption-Password header", seed, shareMode)
			return false
		}

		// Step 5: Extract pack.json from the downloaded ZIP
		downloadedData := downloadRR.Body.Bytes()
		encryptedPackJSON, err := extractPackJSONFromZip(downloadedData)
		if err != nil {
			t.Logf("seed=%d: failed to extract pack.json from downloaded ZIP: %v", seed, err)
			return false
		}

		// Step 6: Decrypt pack.json using the password
		decryptedPackJSON, err := testDecryptData(encryptedPackJSON, password)
		if err != nil {
			t.Logf("seed=%d: failed to decrypt pack.json with provided password: %v", seed, err)
			return false
		}

		// Step 7: Compare decrypted content with the original pack.json content.
		// The server injects listing_id into pack.json during upload, so we need
		// to compare after adding listing_id to the original.
		var originalMap map[string]interface{}
		var decryptedMap map[string]interface{}
		if err := json.Unmarshal(originalPackJSON, &originalMap); err != nil {
			t.Logf("seed=%d: failed to parse original pack.json: %v", seed, err)
			return false
		}
		if err := json.Unmarshal(decryptedPackJSON, &decryptedMap); err != nil {
			t.Logf("seed=%d: failed to parse decrypted pack.json: %v", seed, err)
			return false
		}
		// Inject the expected listing_id into the original for comparison
		if meta, ok := originalMap["metadata"].(map[string]interface{}); ok {
			meta["listing_id"] = float64(listing.ID)
		}
		origNorm, _ := json.Marshal(originalMap)
		decNorm, _ := json.Marshal(decryptedMap)
		if !bytes.Equal(origNorm, decNorm) {
			t.Logf("seed=%d: decrypted pack.json does not match original (after listing_id injection)\n  original: %s\n  decrypted: %s", seed, origNorm, decNorm)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (付费包上传-下载-解密端到端往返) failed: %v", err)
	}
}
