package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: paid-pack-encryption, Property 6: API 列表响应不暴露加密密码
// **Validates: Requirements 5.3**
//
// For any pack listing's API list or detail response (including admin interfaces),
// the returned JSON should NOT contain an encryption_password field.
func TestProperty6_APIListResponseNoEncryptionPassword(t *testing.T) {
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

		// Randomly pick a pricing model (including paid ones that have encryption_password)
		allModes := []string{"free", "per_use", "subscription"}
		shareMode := allModes[r.Intn(len(allModes))]

		var creditsPrice int
		switch shareMode {
		case "per_use":
			creditsPrice = r.Intn(100) + 1
		case "subscription":
			creditsPrice = r.Intn(901) + 100
		default:
			creditsPrice = 0
		}

		// Upload the pack
		uploaderID := createTestUser(t)
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

		// Check upload response itself does not contain encryption_password
		uploadBody := uploadRR.Body.String()
		if strings.Contains(uploadBody, "encryption_password") {
			t.Logf("seed=%d: upload response contains 'encryption_password' (share_mode=%s)", seed, shareMode)
			return false
		}

		var listing PackListingInfo
		if err := json.Unmarshal(uploadRR.Body.Bytes(), &listing); err != nil {
			t.Logf("seed=%d: failed to unmarshal upload response: %v", seed, err)
			return false
		}

		// Publish the pack so it appears in list APIs
		if _, err := db.Exec("UPDATE pack_listings SET status = 'published' WHERE id = ?", listing.ID); err != nil {
			t.Logf("seed=%d: failed to publish pack: %v", seed, err)
			return false
		}

		// Check 1: handleListPacks response does not contain encryption_password
		listReq := httptest.NewRequest(http.MethodGet, "/api/packs", nil)
		listRR := httptest.NewRecorder()
		handleListPacks(listRR, listReq)

		if listRR.Code != http.StatusOK {
			t.Logf("seed=%d: list packs expected 200, got %d; body: %s", seed, listRR.Code, listRR.Body.String())
			return false
		}

		listBody := listRR.Body.String()
		if strings.Contains(listBody, "encryption_password") {
			t.Logf("seed=%d: handleListPacks response contains 'encryption_password' (share_mode=%s)", seed, shareMode)
			return false
		}

		// Check 2: handleAdminMarketplaceList response does not contain encryption_password
		adminListReq := httptest.NewRequest(http.MethodGet, "/admin/api/marketplace", nil)
		adminListRR := httptest.NewRecorder()
		handleAdminMarketplaceList(adminListRR, adminListReq)

		if adminListRR.Code != http.StatusOK {
			t.Logf("seed=%d: admin list expected 200, got %d; body: %s", seed, adminListRR.Code, adminListRR.Body.String())
			return false
		}

		adminListBody := adminListRR.Body.String()
		if strings.Contains(adminListBody, "encryption_password") {
			t.Logf("seed=%d: handleAdminMarketplaceList response contains 'encryption_password' (share_mode=%s)", seed, shareMode)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 (API 列表响应不暴露加密密码) failed: %v", err)
	}
}
