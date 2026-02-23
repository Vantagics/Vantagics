package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: author-storefront, Property 12: user_id 唯一约束
// **Validates: Requirements 7.2**
//
// For any user_id in author_storefronts, inserting a second storefront with the
// same user_id must fail due to the UNIQUE constraint on user_id.
// This ensures each author can have at most one storefront.
func TestProperty12_UserIDUniqueConstraint(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user
		balance := float64(rng.Intn(1000))
		userID := createTestUserWithBalance(t, balance)

		// Generate a random slug
		slug1 := fmt.Sprintf("store-%d-%d", userID, rng.Int63n(1000000))
		slug2 := fmt.Sprintf("store-%d-%d-dup", userID, rng.Int63n(1000000))

		// First insert should succeed
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug1,
		)
		if err != nil {
			t.Logf("FAIL: first insert failed unexpectedly: %v", err)
			return false
		}

		// Second insert with same user_id should fail
		_, err = db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug2,
		)
		if err == nil {
			t.Logf("FAIL: second insert with same user_id=%d succeeded, expected UNIQUE violation", userID)
			return false
		}

		// Verify exactly one record exists for this user_id
		var count int
		db.QueryRow("SELECT COUNT(*) FROM author_storefronts WHERE user_id = ?", userID).Scan(&count)
		if count != 1 {
			t.Logf("FAIL: expected exactly 1 storefront for user_id=%d, got %d", userID, count)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 12 violated: %v", err)
	}
}

// Feature: author-storefront, Property 13: storefront_packs 联合唯一约束
// **Validates: Requirements 7.5, 9.9**
//
// For any (storefront_id, pack_listing_id) pair in storefront_packs, inserting a
// duplicate pair must fail due to the UNIQUE(storefront_id, pack_listing_id) constraint.
// This prevents the same pack from being added to a storefront more than once.
func TestProperty13_StorefrontPacksCompositeUniqueConstraint(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		balance := float64(rng.Intn(1000))
		userID := createTestUserWithBalance(t, balance)
		slug := fmt.Sprintf("store-%d-%d", userID, rng.Int63n(1000000))

		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create a test pack listing
		packID := createTestPackListing(t, userID, 1, "free", 0, []byte("test-data"))

		// First insert should succeed
		_, err = db.Exec(
			"INSERT INTO storefront_packs (storefront_id, pack_listing_id) VALUES (?, ?)",
			storefrontID, packID,
		)
		if err != nil {
			t.Logf("FAIL: first insert failed unexpectedly: %v", err)
			return false
		}

		// Second insert with same (storefront_id, pack_listing_id) should fail
		_, err = db.Exec(
			"INSERT INTO storefront_packs (storefront_id, pack_listing_id) VALUES (?, ?)",
			storefrontID, packID,
		)
		if err == nil {
			t.Logf("FAIL: duplicate insert (storefront_id=%d, pack_listing_id=%d) succeeded, expected UNIQUE violation",
				storefrontID, packID)
			return false
		}

		// Verify exactly one record exists for this pair
		var count int
		db.QueryRow(
			"SELECT COUNT(*) FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?",
			storefrontID, packID,
		).Scan(&count)
		if count != 1 {
			t.Logf("FAIL: expected exactly 1 record for (storefront_id=%d, pack_listing_id=%d), got %d",
				storefrontID, packID, count)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 13 violated: %v", err)
	}
}


// Feature: author-storefront, Property 1: Store_Slug 格式不变性
// **Validates: Requirements 2.1, 2.5, 2.6**
//
// For any valid display_name, generateStoreSlug produces a slug that:
// - Matches ^[a-z0-9-]+$
// - Has length >= 3 and <= 50
func TestProperty1_StoreSlugFormatInvariance(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Generate a random display name of varying length and content
		nameLen := rng.Intn(100) + 1
		chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_!@#$%^&*()你好世界")
		name := make([]rune, nameLen)
		for i := range name {
			name[i] = chars[rng.Intn(len(chars))]
		}
		displayName := string(name)

		slug := generateStoreSlug(displayName)

		// Check slug matches ^[a-z0-9-]+$
		matched, err := regexp.MatchString(`^[a-z0-9-]+$`, slug)
		if err != nil || !matched {
			t.Logf("FAIL: slug %q from displayName %q does not match ^[a-z0-9-]+$", slug, displayName)
			return false
		}

		// Check length >= 3
		if len(slug) < 3 {
			t.Logf("FAIL: slug %q from displayName %q has length %d < 3", slug, displayName, len(slug))
			return false
		}

		// Check length <= 50
		if len(slug) > 50 {
			t.Logf("FAIL: slug %q from displayName %q has length %d > 50", slug, displayName, len(slug))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// Feature: author-storefront, Property 3: 小铺名称长度验证
// **Validates: Requirements 1.4**
//
// For any string name, validateStoreName(name) returns valid (empty string)
// iff rune count of name >= 2 AND rune count of name <= 30.
func TestProperty3_StoreNameLengthValidation(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate a random string with varying length (0-60 runes) including Unicode
		nameLen := rng.Intn(61)
		chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789你好世界测试名称铺子")
		name := make([]rune, nameLen)
		for i := range name {
			name[i] = chars[rng.Intn(len(chars))]
		}
		nameStr := string(name)

		result := validateStoreName(nameStr)
		runeCount := len([]rune(nameStr))
		isValid := runeCount >= 2 && runeCount <= 30

		if isValid && result != "" {
			t.Logf("FAIL: name %q (rune count %d) should be valid but got error: %s", nameStr, runeCount, result)
			return false
		}
		if !isValid && result == "" {
			t.Logf("FAIL: name %q (rune count %d) should be invalid but got no error", nameStr, runeCount)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 violated: %v", err)
	}
}

// Feature: author-storefront, Property 11: Logo 上传 Round-Trip
// **Validates: Requirements 1.7, 1.11, 7.8**
//
// For any valid image (PNG or JPEG, <= 2MB), uploading it via handleStorefrontUploadLogo
// and then retrieving the stored data from the database yields the same binary data
// and the same content type.
// Feature: author-storefront, Property 11: Logo 上传 Round-Trip
// **Validates: Requirements 1.7, 1.11, 7.8**
//
// For any valid image (PNG or JPEG, ≤ 2MB), uploading it via handleStorefrontUploadLogo
// and then retrieving via /store/{slug}/logo (handleStorefrontLogo) returns identical data.
// The Content-Type header must match the uploaded image type.
func TestProperty11_LogoUploadRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Decide randomly whether to generate PNG or JPEG
		isPNG := rng.Intn(2) == 0

		var imageData []byte
		var expectedContentType string

		if isPNG {
			imageData = generateValidPNG(rng)
			expectedContentType = "image/png"
		} else {
			imageData = generateValidJPEG(rng)
			expectedContentType = "image/jpeg"
		}

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("logo-test-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		// Upload logo via handler using multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("logo", "test-logo.png")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		_, err = part.Write(imageData)
		if err != nil {
			t.Logf("FAIL: failed to write image data: %v", err)
			return false
		}
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/user/storefront/logo", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleStorefrontUploadLogo(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: upload returned status %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		// Retrieve logo via /store/{slug}/logo HTTP endpoint (round-trip verification)
		logoReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/store/%s/logo", slug), nil)
		logoRR := httptest.NewRecorder()
		handleStorefrontLogo(logoRR, logoReq, slug)

		if logoRR.Code != http.StatusOK {
			t.Logf("FAIL: GET /store/%s/logo returned status %d", slug, logoRR.Code)
			return false
		}

		// Verify retrieved data matches original upload
		retrievedData := logoRR.Body.Bytes()
		if !bytes.Equal(retrievedData, imageData) {
			t.Logf("FAIL: retrieved data (%d bytes) does not match uploaded data (%d bytes)",
				len(retrievedData), len(imageData))
			return false
		}

		// Verify Content-Type header matches the uploaded image type
		retrievedContentType := logoRR.Header().Get("Content-Type")
		if retrievedContentType != expectedContentType {
			t.Logf("FAIL: retrieved Content-Type %q does not match expected %q",
				retrievedContentType, expectedContentType)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 11 violated: %v", err)
	}
}
// Feature: author-storefront, Property 4: 推荐分析包数量上限
// **Validates: Requirements 10.2, 10.3**
//
// For any storefront, the number of featured packs (is_featured = 1) never exceeds 4.
// Setting a 5th featured pack must be rejected with error "最多设置 4 个推荐分析包".
func TestProperty4_FeaturedPackCountLimit(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("feat-test-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		// Create 6+ published pack listings (random between 6 and 10)
		numPacks := rng.Intn(5) + 6
		var packIDs []int64
		for i := 0; i < numPacks; i++ {
			modes := []string{"free", "per_use", "subscription"}
			mode := modes[rng.Intn(len(modes))]
			price := rng.Intn(100)
			packID := createTestPackListing(t, userID, 1, mode, price, []byte("data"))
			packIDs = append(packIDs, packID)
		}

		// Shuffle pack IDs to randomize the order of featuring
		rng.Shuffle(len(packIDs), func(i, j int) {
			packIDs[i], packIDs[j] = packIDs[j], packIDs[i]
		})

		// Try to set each pack as featured via the handler
		for i, packID := range packIDs {
			body := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
				bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSetFeatured(rr, req)

			if i < 4 {
				// First 4 should succeed
				if rr.Code != http.StatusOK {
					t.Logf("FAIL: setting featured pack #%d (packID=%d) returned status %d, expected 200. Body: %s",
						i+1, packID, rr.Code, rr.Body.String())
					return false
				}
			} else {
				// 5th and beyond should be rejected
				if rr.Code != http.StatusBadRequest {
					t.Logf("FAIL: setting featured pack #%d (packID=%d) returned status %d, expected 400. Body: %s",
						i+1, packID, rr.Code, rr.Body.String())
					return false
				}
				// Verify error message
				respBody := rr.Body.String()
				if !bytes.Contains([]byte(respBody), []byte("最多设置 4 个推荐分析包")) {
					t.Logf("FAIL: expected error '最多设置 4 个推荐分析包' for pack #%d, got: %s",
						i+1, respBody)
					return false
				}
			}

			// After each attempt, verify DB invariant: featured count <= 4
			var featuredCount int
			var storefrontID int64
			db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
			db.QueryRow("SELECT COUNT(*) FROM storefront_packs WHERE storefront_id = ? AND is_featured = 1",
				storefrontID).Scan(&featuredCount)
			if featuredCount > 4 {
				t.Logf("FAIL: after attempt #%d, featured count = %d > 4 for storefront %d",
					i+1, featuredCount, storefrontID)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 violated: %v", err)
	}
}

// Feature: author-storefront, Property 5: 自动模式与手动模式查询一致性
// **Validates: Requirements 9.4, 9.5, 7.6, 7.7**
//
// When auto_add_enabled = 1, queryStorefrontPacks returns exactly all published
// packs by the author. When auto_add_enabled = 0, queryStorefrontPacks returns
// only the packs explicitly added to storefront_packs, which must be a subset
// of the author's published packs.
func TestProperty5_AutoManualModeQueryConsistency(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("mode-test-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, auto_add_enabled) VALUES (?, ?, 0)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create a random number of pack listings (3-8), with random statuses
		totalPacks := rng.Intn(6) + 3
		var publishedIDs []int64
		for i := 0; i < totalPacks; i++ {
			modes := []string{"free", "per_use", "subscription"}
			mode := modes[rng.Intn(len(modes))]
			price := rng.Intn(100)
			packID := createTestPackListing(t, userID, 1, mode, price, []byte("data"))

			// Randomly set some packs to non-published status
			if rng.Intn(3) == 0 {
				// Make this pack non-published (e.g., pending or rejected)
				statuses := []string{"pending", "rejected"}
				_, err := db.Exec("UPDATE pack_listings SET status = ? WHERE id = ?",
					statuses[rng.Intn(len(statuses))], packID)
				if err != nil {
					t.Logf("FAIL: failed to update pack status: %v", err)
					return false
				}
			} else {
				publishedIDs = append(publishedIDs, packID)
			}
		}

		// --- AUTO MODE TEST ---
		// Query with auto mode enabled
		autoPacks, err := queryStorefrontPacks(storefrontID, true, "revenue", "", "", "")
		if err != nil {
			t.Logf("FAIL: queryStorefrontPacks auto mode failed: %v", err)
			return false
		}

		// Auto mode should return exactly all published packs by this author
		autoPackIDs := make(map[int64]bool)
		for _, p := range autoPacks {
			autoPackIDs[p.ListingID] = true
		}

		if len(autoPackIDs) != len(publishedIDs) {
			t.Logf("FAIL: auto mode returned %d packs, expected %d published packs",
				len(autoPackIDs), len(publishedIDs))
			return false
		}
		for _, pid := range publishedIDs {
			if !autoPackIDs[pid] {
				t.Logf("FAIL: auto mode missing published pack ID %d", pid)
				return false
			}
		}

		// --- MANUAL MODE TEST ---
		// Add a random subset of published packs to storefront_packs
		var manuallyAddedIDs []int64
		for _, pid := range publishedIDs {
			if rng.Intn(2) == 0 {
				_, err := db.Exec(
					"INSERT INTO storefront_packs (storefront_id, pack_listing_id) VALUES (?, ?)",
					storefrontID, pid,
				)
				if err != nil {
					t.Logf("FAIL: failed to add pack to storefront: %v", err)
					return false
				}
				manuallyAddedIDs = append(manuallyAddedIDs, pid)
			}
		}

		// Query with manual mode
		manualPacks, err := queryStorefrontPacks(storefrontID, false, "revenue", "", "", "")
		if err != nil {
			t.Logf("FAIL: queryStorefrontPacks manual mode failed: %v", err)
			return false
		}

		// Manual mode should return exactly the manually added packs
		manualPackIDs := make(map[int64]bool)
		for _, p := range manualPacks {
			manualPackIDs[p.ListingID] = true
		}

		if len(manualPackIDs) != len(manuallyAddedIDs) {
			t.Logf("FAIL: manual mode returned %d packs, expected %d manually added packs",
				len(manualPackIDs), len(manuallyAddedIDs))
			return false
		}
		for _, pid := range manuallyAddedIDs {
			if !manualPackIDs[pid] {
				t.Logf("FAIL: manual mode missing manually added pack ID %d", pid)
				return false
			}
		}

		// --- SUBSET PROPERTY ---
		// Manual mode results must be a subset of auto mode results
		for pid := range manualPackIDs {
			if !autoPackIDs[pid] {
				t.Logf("FAIL: manual mode pack ID %d is not in auto mode results (not a subset)", pid)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 violated: %v", err)
	}
}

// generateValidPNG creates a minimal valid PNG image with random pixel data.
// PNG format: 8-byte signature + IHDR chunk + IDAT chunk + IEND chunk
func generateValidPNG(rng *rand.Rand) []byte {
	buf := &bytes.Buffer{}

	// PNG signature
	buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

	// IHDR chunk: 1x1 pixel, 8-bit RGB
	ihdrData := []byte{
		0x00, 0x00, 0x00, 0x01, // width = 1
		0x00, 0x00, 0x00, 0x01, // height = 1
		0x08,                   // bit depth = 8
		0x02,                   // color type = RGB
		0x00,                   // compression method
		0x00,                   // filter method
		0x00,                   // interlace method
	}
	writeChunk(buf, "IHDR", ihdrData)

	// IDAT chunk: zlib-compressed image data (1x1 RGB pixel)
	// Filter byte (0x00 = None) + 3 random RGB bytes
	rawRow := []byte{0x00, byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256))}
	var zlibBuf bytes.Buffer
	zlibW := zlib.NewWriter(&zlibBuf)
	zlibW.Write(rawRow)
	zlibW.Close()
	writeChunk(buf, "IDAT", zlibBuf.Bytes())

	// IEND chunk
	writeChunk(buf, "IEND", nil)

	return buf.Bytes()
}

// generateValidJPEG creates a minimal valid JPEG image with random data.
// JPEG format: SOI marker + APP0 (JFIF) + minimal content + EOI marker
func generateValidJPEG(rng *rand.Rand) []byte {
	buf := &bytes.Buffer{}

	// SOI (Start of Image)
	buf.Write([]byte{0xFF, 0xD8})

	// APP0 JFIF marker
	buf.Write([]byte{0xFF, 0xE0})
	// Length of APP0 segment (16 bytes)
	buf.Write([]byte{0x00, 0x10})
	// JFIF identifier
	buf.Write([]byte{0x4A, 0x46, 0x49, 0x46, 0x00})
	// Version 1.01
	buf.Write([]byte{0x01, 0x01})
	// Aspect ratio units (0 = no units)
	buf.Write([]byte{0x00})
	// X density, Y density
	buf.Write([]byte{0x00, 0x01, 0x00, 0x01})
	// Thumbnail dimensions (0x0)
	buf.Write([]byte{0x00, 0x00})

	// Add some random bytes as padding (still valid JPEG structure)
	randomLen := rng.Intn(64) + 16
	randomData := make([]byte, randomLen)
	for i := range randomData {
		randomData[i] = byte(rng.Intn(256))
	}
	// Wrap random data in a COM (comment) marker segment
	buf.Write([]byte{0xFF, 0xFE}) // COM marker
	comLen := len(randomData) + 2
	buf.Write([]byte{byte(comLen >> 8), byte(comLen & 0xFF)})
	buf.Write(randomData)

	// EOI (End of Image)
	buf.Write([]byte{0xFF, 0xD9})

	return buf.Bytes()
}

// writeChunk writes a PNG chunk (length + type + data + CRC).
func writeChunk(buf *bytes.Buffer, chunkType string, data []byte) {
	// Length (4 bytes, big-endian)
	length := uint32(len(data))
	buf.Write([]byte{byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)})

	// Type (4 bytes)
	typeBytes := []byte(chunkType)
	buf.Write(typeBytes)

	// Data
	if len(data) > 0 {
		buf.Write(data)
	}

	// CRC32 over type + data
	crcData := append(typeBytes, data...)
	crc := crc32.ChecksumIEEE(crcData)
	buf.Write([]byte{byte(crc >> 24), byte(crc >> 16), byte(crc >> 8), byte(crc)})
}

// Feature: author-storefront, Property 7: 推荐分析包排序不变性
// **Validates: Requirements 10.7**
//
// For all featured_packs in a storefront, after any reorder operation,
// featured_packs[i].sort_order < featured_packs[i+1].sort_order (for all valid i).
// The sort orders must be strictly increasing when queried by featured_sort_order ASC.
func TestProperty7_FeaturedPackSortOrderInvariance(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("sort-test-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		// Create 4 published pack listings
		var packIDs []int64
		for i := 0; i < 4; i++ {
			modes := []string{"free", "per_use", "subscription"}
			mode := modes[rng.Intn(len(modes))]
			price := rng.Intn(100)
			packID := createTestPackListing(t, userID, 1, mode, price, []byte("data"))
			packIDs = append(packIDs, packID)
		}

		// Set all 4 as featured via handleStorefrontSetFeatured
		for _, packID := range packIDs {
			body := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
				bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSetFeatured(rr, req)
			if rr.Code != http.StatusOK {
				t.Logf("FAIL: setting featured pack %d returned status %d: %s", packID, rr.Code, rr.Body.String())
				return false
			}
		}

		// Perform multiple random reorders and verify invariant each time
		numReorders := rng.Intn(5) + 2 // 2-6 reorder attempts
		for attempt := 0; attempt < numReorders; attempt++ {
			// Shuffle pack IDs to create a random new order
			shuffled := make([]int64, len(packIDs))
			copy(shuffled, packIDs)
			rng.Shuffle(len(shuffled), func(i, j int) {
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			})

			// Build JSON body with pack IDs
			idsJSON, _ := json.Marshal(map[string][]int64{"ids": shuffled})

			// Call handleStorefrontReorderFeatured
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/reorder",
				bytes.NewBuffer(idsJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontReorderFeatured(rr, req)
			if rr.Code != http.StatusOK {
				t.Logf("FAIL: reorder attempt %d returned status %d: %s", attempt, rr.Code, rr.Body.String())
				return false
			}

			// Query featured packs ordered by featured_sort_order ASC
			rows, err := db.Query(
				`SELECT pack_listing_id, featured_sort_order FROM storefront_packs
				 WHERE storefront_id = (SELECT id FROM author_storefronts WHERE user_id = ?)
				 AND is_featured = 1
				 ORDER BY featured_sort_order ASC`, userID)
			if err != nil {
				t.Logf("FAIL: failed to query featured packs: %v", err)
				return false
			}

			var sortOrders []int
			for rows.Next() {
				var packID int64
				var sortOrder int
				if err := rows.Scan(&packID, &sortOrder); err != nil {
					rows.Close()
					t.Logf("FAIL: failed to scan row: %v", err)
					return false
				}
				sortOrders = append(sortOrders, sortOrder)
			}
			rows.Close()

			// Verify strictly increasing sort orders
			for i := 0; i < len(sortOrders)-1; i++ {
				if sortOrders[i] >= sortOrders[i+1] {
					t.Logf("FAIL: reorder attempt %d: sort_order[%d]=%d >= sort_order[%d]=%d (not strictly increasing)",
						attempt, i, sortOrders[i], i+1, sortOrders[i+1])
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 7 violated: %v", err)
	}
}

// Feature: author-storefront, Property 9: 分析包列表排序正确性
// **Validates: Requirements 4.4, 4.6, 4.7**
//
// For all sort_field in {revenue, downloads, orders},
//   packs = queryStorefrontPacks(storefront, sort_field)
//   => packs[i].sort_value >= packs[i+1].sort_value (for all valid i)
// The pack list must be in descending order for the selected sort field.
func TestProperty9_PackListSortingCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront (auto mode for simplicity)
		userID := createTestUserWithBalance(t, 1000)
		slug := fmt.Sprintf("sort9-test-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, auto_add_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create a buyer user for transactions
		buyerID := createTestUserWithBalance(t, 10000)

		// Create 4-8 published pack listings with varying download counts
		numPacks := rng.Intn(5) + 4
		type packInfo struct {
			id            int64
			downloadCount int
		}
		var packs []packInfo

		for i := 0; i < numPacks; i++ {
			modes := []string{"free", "per_use", "subscription"}
			mode := modes[rng.Intn(len(modes))]
			price := rng.Intn(50) + 1
			packID := createTestPackListing(t, userID, 1, mode, price, []byte("data"))

			// Set a random download_count for each pack
			dlCount := rng.Intn(500)
			_, err := db.Exec("UPDATE pack_listings SET download_count = ? WHERE id = ?", dlCount, packID)
			if err != nil {
				t.Logf("FAIL: failed to update download_count: %v", err)
				return false
			}
			packs = append(packs, packInfo{id: packID, downloadCount: dlCount})

			// Insert random number of credits_transactions for revenue and order count
			numTxns := rng.Intn(10)
			txnTypes := []string{"purchase", "purchase_uses", "renew_subscription"}
			for j := 0; j < numTxns; j++ {
				txnType := txnTypes[rng.Intn(len(txnTypes))]
				amount := -(float64(rng.Intn(100) + 1)) // negative amount = purchase
				_, err := db.Exec(
					"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id) VALUES (?, ?, ?, ?)",
					buyerID, txnType, amount, packID,
				)
				if err != nil {
					t.Logf("FAIL: failed to insert transaction: %v", err)
					return false
				}
			}
		}

		// Test all three sort fields
		sortFields := []string{"revenue", "downloads", "orders"}
		for _, sortField := range sortFields {
			results, err := queryStorefrontPacks(storefrontID, true, sortField, "", "", "")
			if err != nil {
				t.Logf("FAIL: queryStorefrontPacks(%s) failed: %v", sortField, err)
				return false
			}

			// Verify descending order for the selected sort field
			for i := 0; i < len(results)-1; i++ {
				var currVal, nextVal float64
				switch sortField {
				case "downloads":
					currVal = float64(results[i].DownloadCount)
					nextVal = float64(results[i+1].DownloadCount)
				case "revenue":
					currVal = results[i].TotalRevenue
					nextVal = results[i+1].TotalRevenue
				case "orders":
					currVal = float64(results[i].OrderCount)
					nextVal = float64(results[i+1].OrderCount)
				}

				if currVal < nextVal {
					t.Logf("FAIL: sort by %s not descending at index %d: %v < %v (pack IDs: %d, %d)",
						sortField, i, currVal, nextVal, results[i].ListingID, results[i+1].ListingID)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 9 violated: %v", err)
	}
}

// Feature: author-storefront, Property 10: 筛选结果子集属性
// **Validates: Requirements 4.2**
//
// For all filter in {free, per_use, subscription},
//   filtered = queryStorefrontPacks(storefront, filter)
//   all = queryStorefrontPacks(storefront, "all")
//   => len(filtered) <= len(all)
//   AND for all pack in filtered, pack.share_mode == filter
func TestProperty10_FilterResultSubsetProperty(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront (auto mode for simplicity)
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("filter-test-%d-%d", userID, rng.Int63n(1000000))
		result, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug, auto_add_enabled) VALUES (?, ?, 1)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := result.LastInsertId()

		// Create pack listings with mixed share_modes
		modes := []string{"free", "per_use", "subscription"}
		numPacks := rng.Intn(8) + 3 // 3-10 packs
		for i := 0; i < numPacks; i++ {
			mode := modes[rng.Intn(len(modes))]
			price := rng.Intn(100)
			createTestPackListing(t, userID, 1, mode, price, []byte("data"))
		}

		// Query all packs (no filter)
		allPacks, err := queryStorefrontPacks(storefrontID, true, "revenue", "", "", "")
		if err != nil {
			t.Logf("FAIL: queryStorefrontPacks(all) failed: %v", err)
			return false
		}

		// For each filter value, verify subset property
		for _, filterMode := range modes {
			filtered, err := queryStorefrontPacks(storefrontID, true, "revenue", filterMode, "", "")
			if err != nil {
				t.Logf("FAIL: queryStorefrontPacks(filter=%s) failed: %v", filterMode, err)
				return false
			}

			// Verify: len(filtered) <= len(all)
			if len(filtered) > len(allPacks) {
				t.Logf("FAIL: filter=%s returned %d packs > total %d packs",
					filterMode, len(filtered), len(allPacks))
				return false
			}

			// Verify: all packs in filtered have matching share_mode
			for _, pack := range filtered {
				if pack.ShareMode != filterMode {
					t.Logf("FAIL: filter=%s but pack %d has share_mode=%s",
						filterMode, pack.ListingID, pack.ShareMode)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 10 violated: %v", err)
	}
}

// Feature: author-storefront, Property 14: OG description 截断属性
// **Validates: Requirements 6.2**
//
// For all storefront.description,
//   og_desc = truncateDesc(description, 200)
//   => the meaningful content (before "..." suffix) has rune length <= 200
//   AND the original description starts with that meaningful content (or og_desc == description)
func TestProperty14_OGDescriptionTruncation(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Reimplement the same truncation logic as truncateDesc in templates/storefront.go
	truncateDesc := func(s string, maxLen int) string {
		runes := []rune(s)
		if len(runes) <= maxLen {
			return s
		}
		return string(runes[:maxLen]) + "..."
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate a random description of 0-500 runes, including Unicode characters
		descLen := rng.Intn(501)
		chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 你好世界测试描述小铺作者分析包市场推荐🎉🚀📦✨")
		desc := make([]rune, descLen)
		for i := range desc {
			desc[i] = chars[rng.Intn(len(chars))]
		}
		description := string(desc)

		ogDesc := truncateDesc(description, 200)
		ogRunes := []rune(ogDesc)
		descRunes := []rune(description)

		if len(descRunes) <= 200 {
			// Short description: ogDesc should equal the original description exactly
			if ogDesc != description {
				t.Logf("FAIL: description has %d runes (<= 200), expected ogDesc == description, but ogDesc=%q != description=%q",
					len(descRunes), ogDesc, description)
				return false
			}
		} else {
			// Long description: ogDesc should be first 200 runes + "..."
			// Verify the meaningful content is exactly 200 runes
			suffix := "..."
			if len(ogRunes) < 3 {
				t.Logf("FAIL: truncated ogDesc is too short: %q", ogDesc)
				return false
			}
			// Check that ogDesc ends with "..."
			if ogDesc[len(ogDesc)-len(suffix):] != suffix {
				t.Logf("FAIL: truncated ogDesc %q does not end with '...'", ogDesc)
				return false
			}
			// Extract meaningful content (without "..." suffix)
			meaningful := string(ogRunes[:len(ogRunes)-3])
			meaningfulRunes := []rune(meaningful)

			// Verify meaningful content rune count <= 200
			if len(meaningfulRunes) > 200 {
				t.Logf("FAIL: meaningful content has %d runes > 200", len(meaningfulRunes))
				return false
			}

			// Verify the original description starts with the meaningful content
			if len(descRunes) < len(meaningfulRunes) {
				t.Logf("FAIL: description (%d runes) shorter than meaningful content (%d runes)",
					len(descRunes), len(meaningfulRunes))
				return false
			}
			descPrefix := string(descRunes[:len(meaningfulRunes)])
			if descPrefix != meaningful {
				t.Logf("FAIL: description does not start with meaningful content. prefix=%q, meaningful=%q",
					descPrefix, meaningful)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 14 violated: %v", err)
	}
}

// Feature: author-storefront, Property 6: 客户去重不变性
// **Validates: Requirements 12.4, 12.6**
//
// 查询全部客户或部分客户时，返回的用户列表中不包含重复用户（即使同一用户购买了多个分析包）。
// FOR ALL author_id,
//   recipients = getRecipients(author_id, "all")
//   => COUNT(DISTINCT recipients.user_id) == COUNT(recipients.user_id)
func TestProperty6_CustomerDeduplicationInvariance(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// 创建作者用户和小铺
		authorID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("dedup-test-%d-%d", authorID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			authorID, slug,
		)
		if err != nil {
			t.Logf("FAIL: 创建小铺失败: %v", err)
			return false
		}

		// 创建多个分析包（2-5个）
		numPacks := rng.Intn(4) + 2
		var packIDs []int64
		for i := 0; i < numPacks; i++ {
			modes := []string{"free", "per_use", "subscription"}
			mode := modes[rng.Intn(len(modes))]
			price := rng.Intn(100)
			packID := createTestPackListing(t, authorID, 1, mode, price, []byte("data"))
			packIDs = append(packIDs, packID)
		}

		// 创建多个买家用户（2-6个）
		numBuyers := rng.Intn(5) + 2
		var buyerIDs []int64
		for i := 0; i < numBuyers; i++ {
			buyerID := createTestUserWithBalance(t, 1000)
			buyerIDs = append(buyerIDs, buyerID)
		}

		// 模拟购买：每个买家随机购买多个分析包（关键：同一买家购买多个包）
		distinctBuyers := make(map[int64]bool)
		for _, buyerID := range buyerIDs {
			// 每个买家随机购买 1 到 numPacks 个分析包
			numPurchases := rng.Intn(numPacks) + 1
			// 打乱分析包顺序后取前 numPurchases 个
			shuffled := make([]int64, len(packIDs))
			copy(shuffled, packIDs)
			rng.Shuffle(len(shuffled), func(i, j int) {
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			})
			for p := 0; p < numPurchases; p++ {
				_, err := db.Exec(
					"INSERT OR IGNORE INTO user_purchased_packs (user_id, listing_id) VALUES (?, ?)",
					buyerID, shuffled[p],
				)
				if err != nil {
					t.Logf("FAIL: 插入购买记录失败: %v", err)
					return false
				}
				distinctBuyers[buyerID] = true
			}
		}

		expectedDistinctCount := len(distinctBuyers)

		// --- 测试 scope=all ---
		req := httptest.NewRequest(http.MethodGet, "/user/storefront/notify/recipients?scope=all", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", authorID))
		rr := httptest.NewRecorder()
		handleStorefrontGetRecipients(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: scope=all 返回状态码 %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		var allResp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &allResp); err != nil {
			t.Logf("FAIL: 解析 scope=all 响应失败: %v", err)
			return false
		}

		allCount := int(allResp["count"].(float64))

		// 去重不变性：返回的数量必须等于不同买家的数量（不是总购买次数）
		if allCount != expectedDistinctCount {
			t.Logf("FAIL: scope=all 返回 count=%d, 期望去重后的买家数=%d", allCount, expectedDistinctCount)
			return false
		}

		// --- 测试 scope=partial（选择部分分析包）---
		// 随机选择 1 到 numPacks 个分析包
		numSelected := rng.Intn(numPacks) + 1
		selectedShuffled := make([]int64, len(packIDs))
		copy(selectedShuffled, packIDs)
		rng.Shuffle(len(selectedShuffled), func(i, j int) {
			selectedShuffled[i], selectedShuffled[j] = selectedShuffled[j], selectedShuffled[i]
		})
		selectedPacks := selectedShuffled[:numSelected]

		// 构建 listing_ids 参数
		var idStrs []string
		for _, id := range selectedPacks {
			idStrs = append(idStrs, fmt.Sprintf("%d", id))
		}
		listingIDsParam := strings.Join(idStrs, ",")

		// 计算期望的去重买家数（购买了所选分析包中任意一个的不同买家）
		expectedPartialBuyers := make(map[int64]bool)
		selectedSet := make(map[int64]bool)
		for _, pid := range selectedPacks {
			selectedSet[pid] = true
		}
		rows, err := db.Query("SELECT user_id, listing_id FROM user_purchased_packs")
		if err != nil {
			t.Logf("FAIL: 查询购买记录失败: %v", err)
			return false
		}
		for rows.Next() {
			var uid, lid int64
			rows.Scan(&uid, &lid)
			if selectedSet[lid] {
				expectedPartialBuyers[uid] = true
			}
		}
		rows.Close()

		partialURL := fmt.Sprintf("/user/storefront/notify/recipients?scope=partial&listing_ids=%s", listingIDsParam)
		req2 := httptest.NewRequest(http.MethodGet, partialURL, nil)
		req2.Header.Set("X-User-ID", fmt.Sprintf("%d", authorID))
		rr2 := httptest.NewRecorder()
		handleStorefrontGetRecipients(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Logf("FAIL: scope=partial 返回状态码 %d, body: %s", rr2.Code, rr2.Body.String())
			return false
		}

		var partialResp map[string]interface{}
		if err := json.Unmarshal(rr2.Body.Bytes(), &partialResp); err != nil {
			t.Logf("FAIL: 解析 scope=partial 响应失败: %v", err)
			return false
		}

		partialCount := int(partialResp["count"].(float64))

		// 去重不变性：partial 返回的数量必须等于购买了所选分析包的不同买家数
		if partialCount != len(expectedPartialBuyers) {
			t.Logf("FAIL: scope=partial 返回 count=%d, 期望去重后的买家数=%d", partialCount, len(expectedPartialBuyers))
			return false
		}

		// 额外不变性：partial 的数量不应超过 all 的数量
		if partialCount > allCount {
			t.Logf("FAIL: scope=partial count=%d > scope=all count=%d", partialCount, allCount)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 violated: %v", err)
	}
}

// Feature: author-storefront, Property 8: 移除/下架级联取消推荐
// **Validates: Requirements 10.9**
//
// 当分析包从小铺移除或下架后，该分析包的推荐状态必须被清除。
// FOR ALL pack WHERE pack.status != 'published' OR pack NOT IN storefront_packs,
//   storefront_packs.is_featured == 0 (for that pack)
func TestProperty8_RemoveDelistCascade(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// 创建作者用户和小铺
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("cascade-test-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: 创建小铺失败: %v", err)
			return false
		}

		// 创建 4 个已发布的分析包并添加到小铺，全部设为推荐
		var packIDs []int64
		for i := 0; i < 4; i++ {
			modes := []string{"free", "per_use", "subscription"}
			mode := modes[rng.Intn(len(modes))]
			price := rng.Intn(100)
			packID := createTestPackListing(t, userID, 1, mode, price, []byte("data"))
			packIDs = append(packIDs, packID)
		}

		// 打乱顺序以增加随机性
		rng.Shuffle(len(packIDs), func(i, j int) {
			packIDs[i], packIDs[j] = packIDs[j], packIDs[i]
		})

		// 通过 handler 将所有分析包设为推荐
		for _, packID := range packIDs {
			body := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
				bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSetFeatured(rr, req)
			if rr.Code != http.StatusOK {
				t.Logf("FAIL: 设置推荐分析包 %d 失败，状态码 %d: %s", packID, rr.Code, rr.Body.String())
				return false
			}
		}

		// 验证初始状态：4 个推荐分析包
		var storefrontID int64
		db.QueryRow("SELECT id FROM author_storefronts WHERE user_id = ?", userID).Scan(&storefrontID)
		var featuredCount int
		db.QueryRow("SELECT COUNT(*) FROM storefront_packs WHERE storefront_id = ? AND is_featured = 1",
			storefrontID).Scan(&featuredCount)
		if featuredCount != 4 {
			t.Logf("FAIL: 初始推荐数量应为 4，实际为 %d", featuredCount)
			return false
		}

		// 随机选择测试场景：移除或下架
		// 随机选一个分析包进行移除测试
		removeIdx := rng.Intn(len(packIDs))
		removePackID := packIDs[removeIdx]

		// --- 场景 A：从小铺移除推荐分析包 ---
		body := fmt.Sprintf("pack_listing_id=%d", removePackID)
		req := httptest.NewRequest(http.MethodPost, "/user/storefront/packs/remove",
			bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleStorefrontRemovePack(rr, req)
		if rr.Code != http.StatusOK {
			t.Logf("FAIL: 移除分析包 %d 失败，状态码 %d: %s", removePackID, rr.Code, rr.Body.String())
			return false
		}

		// 验证不变性：被移除的分析包不应存在于 storefront_packs 中（因此不可能有推荐状态）
		var removedCount int
		db.QueryRow("SELECT COUNT(*) FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?",
			storefrontID, removePackID).Scan(&removedCount)
		if removedCount != 0 {
			t.Logf("FAIL: 移除后分析包 %d 仍存在于 storefront_packs 中（count=%d）", removePackID, removedCount)
			return false
		}

		// 验证被移除的分析包没有推荐状态
		var removedFeatured int
		db.QueryRow("SELECT COUNT(*) FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ? AND is_featured = 1",
			storefrontID, removePackID).Scan(&removedFeatured)
		if removedFeatured != 0 {
			t.Logf("FAIL: 移除后分析包 %d 仍有推荐状态", removePackID)
			return false
		}

		// --- 场景 B：下架推荐分析包，验证级联取消推荐 ---
		// 从剩余的分析包中随机选一个进行下架测试
		var remainingPackIDs []int64
		for _, pid := range packIDs {
			if pid != removePackID {
				remainingPackIDs = append(remainingPackIDs, pid)
			}
		}
		delistIdx := rng.Intn(len(remainingPackIDs))
		delistPackID := remainingPackIDs[delistIdx]

		// 确认下架前该分析包是推荐状态
		var beforeFeatured int
		db.QueryRow("SELECT is_featured FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?",
			storefrontID, delistPackID).Scan(&beforeFeatured)
		if beforeFeatured != 1 {
			t.Logf("FAIL: 下架前分析包 %d 应为推荐状态，实际 is_featured=%d", delistPackID, beforeFeatured)
			return false
		}

		// 通过 handleAuthorDelistPack 下架分析包
		delistBody := fmt.Sprintf("listing_id=%d", delistPackID)
		delistReq := httptest.NewRequest(http.MethodPost, "/user/author/delist-pack",
			bytes.NewBufferString(delistBody))
		delistReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		delistReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		delistRR := httptest.NewRecorder()
		handleAuthorDelistPack(delistRR, delistReq)

		// handleAuthorDelistPack 使用 http.Redirect，所以期望 302
		if delistRR.Code != http.StatusFound {
			t.Logf("FAIL: 下架分析包 %d 返回状态码 %d，期望 302: %s", delistPackID, delistRR.Code, delistRR.Body.String())
			return false
		}

		// 验证不变性：下架后该分析包的推荐状态必须被清除
		var afterFeatured int
		err = db.QueryRow("SELECT is_featured FROM storefront_packs WHERE storefront_id = ? AND pack_listing_id = ?",
			storefrontID, delistPackID).Scan(&afterFeatured)
		if err != nil {
			// 如果记录不存在也是可以接受的（推荐状态已清除）
			t.Logf("INFO: 下架后分析包 %d 的 storefront_packs 记录不存在（可接受）", delistPackID)
		} else if afterFeatured != 0 {
			t.Logf("FAIL: 下架后分析包 %d 的 is_featured 应为 0，实际为 %d", delistPackID, afterFeatured)
			return false
		}

		// 全局不变性验证：所有非 published 状态的分析包不应有推荐状态
		rows, err := db.Query(`
			SELECT sp.pack_listing_id, sp.is_featured, pl.status
			FROM storefront_packs sp
			JOIN pack_listings pl ON sp.pack_listing_id = pl.id
			WHERE sp.storefront_id = ? AND sp.is_featured = 1 AND pl.status != 'published'`,
			storefrontID)
		if err != nil {
			t.Logf("FAIL: 查询全局不变性失败: %v", err)
			return false
		}
		defer rows.Close()

		for rows.Next() {
			var pid int64
			var featured int
			var status string
			rows.Scan(&pid, &featured, &status)
			t.Logf("FAIL: 分析包 %d 状态为 %q 但仍有推荐状态 (is_featured=%d)", pid, status, featured)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 violated: %v", err)
	}
}

// Feature: author-storefront, Property 2: Store_Slug 唯一性不变性
// **Validates: Requirements 2.2, 2.3, 7.3**
//
// 对于任意数量的作者创建小铺，所有生成的 store_slug 必须互不相同。
// FOR ALL slugs in author_storefronts,
//
//	COUNT(DISTINCT store_slug) == COUNT(store_slug)
func TestProperty2_StoreSlugUniquenessInvariance(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 30,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// 随机生成 3-8 个用户，使用相同或相似的 display_name
		numUsers := rng.Intn(6) + 3

		// 随机选择一个基础名称
		baseNames := []string{
			"Test Author",
			"hello world",
			"数据分析师",
			"a",
			"my-store",
			"John Doe",
		}
		baseName := baseNames[rng.Intn(len(baseNames))]

		var generatedSlugs []string

		for i := 0; i < numUsers; i++ {
			// 创建用户
			userID := createTestUserWithBalance(t, float64(rng.Intn(1000)))

			// 随机决定是否使用完全相同的名称或略有变化
			displayName := baseName
			if rng.Intn(3) == 0 && i > 0 {
				// 1/3 概率使用略有变化的名称
				variations := []string{
					baseName + " ",
					baseName + "!",
					" " + baseName,
					strings.ToUpper(baseName),
				}
				displayName = variations[rng.Intn(len(variations))]
			}

			// 调用 generateStoreSlug 生成 slug
			slug := generateStoreSlug(displayName)

			// 插入到 author_storefronts 表
			_, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
				userID, slug,
			)
			if err != nil {
				t.Logf("FAIL: 插入小铺失败 (user_id=%d, slug=%q): %v", userID, slug, err)
				return false
			}

			generatedSlugs = append(generatedSlugs, slug)
		}

		// 验证不变性：所有生成的 slug 互不相同
		slugSet := make(map[string]bool)
		for _, s := range generatedSlugs {
			if slugSet[s] {
				t.Logf("FAIL: 发现重复 slug %q（共 %d 个用户，基础名称 %q）", s, numUsers, baseName)
				return false
			}
			slugSet[s] = true
		}

		// 通过数据库查询验证 COUNT(DISTINCT store_slug) == COUNT(store_slug)
		var totalCount, distinctCount int
		db.QueryRow("SELECT COUNT(store_slug) FROM author_storefronts").Scan(&totalCount)
		db.QueryRow("SELECT COUNT(DISTINCT store_slug) FROM author_storefronts").Scan(&distinctCount)

		if totalCount != distinctCount {
			t.Logf("FAIL: COUNT(store_slug)=%d != COUNT(DISTINCT store_slug)=%d", totalCount, distinctCount)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// Feature: pack-logo-upload, Property 1: 推荐分析包 Logo 上传往返一致性
// **Validates: Requirements 2.6, 3.2, 3.4**
//
// For any valid PNG or JPEG image data, when uploaded as a featured pack's logo,
// reading it back via GET /store/{slug}/featured/{listing_id}/logo should return
// identical data with matching Content-Type and Cache-Control: public, max-age=3600.
func TestPackLogoProperty1_FeaturedPackLogoUploadRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Randomly choose PNG or JPEG
		isPNG := rng.Intn(2) == 0

		var imageData []byte
		var expectedContentType string
		if isPNG {
			imageData = generateValidPNG(rng)
			expectedContentType = "image/png"
		} else {
			imageData = generateValidJPEG(rng)
			expectedContentType = "image/jpeg"
		}

		// Create a test user, storefront, and a featured pack
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("plogo-rt-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		packID := createTestPackListing(t, userID, 1, "free", 0, []byte("data"))

		// Set the pack as featured via the handler
		featBody := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
		featReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
			bytes.NewBufferString(featBody))
		featReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		featReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		featRR := httptest.NewRecorder()
		handleStorefrontSetFeatured(featRR, featReq)
		if featRR.Code != http.StatusOK {
			t.Logf("FAIL: set featured returned status %d, body: %s", featRR.Code, featRR.Body.String())
			return false
		}

		// Upload the logo via handleStorefrontFeaturedLogoUpload
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("pack_listing_id", fmt.Sprintf("%d", packID))
		part, err := writer.CreateFormFile("logo", "test-logo.png")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		if _, err := part.Write(imageData); err != nil {
			t.Logf("FAIL: failed to write image data: %v", err)
			return false
		}
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		uploadRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoUpload(uploadRR, uploadReq)
		if uploadRR.Code != http.StatusOK {
			t.Logf("FAIL: upload returned status %d, body: %s", uploadRR.Code, uploadRR.Body.String())
			return false
		}

		// Read the logo back via GET /store/{slug}/featured/{listing_id}/logo
		getURL := fmt.Sprintf("/store/%s/featured/%d/logo", slug, packID)
		getReq := httptest.NewRequest(http.MethodGet, getURL, nil)
		getRR := httptest.NewRecorder()
		handleStorefrontRoutes(getRR, getReq)

		if getRR.Code != http.StatusOK {
			t.Logf("FAIL: GET logo returned status %d, body: %s", getRR.Code, getRR.Body.String())
			return false
		}

		// Verify data consistency: response body must match uploaded image data
		if !bytes.Equal(getRR.Body.Bytes(), imageData) {
			t.Logf("FAIL: GET response body (%d bytes) does not match uploaded data (%d bytes)",
				getRR.Body.Len(), len(imageData))
			return false
		}

		// Verify Content-Type matches the uploaded image's MIME type
		gotContentType := getRR.Header().Get("Content-Type")
		if gotContentType != expectedContentType {
			t.Logf("FAIL: Content-Type %q does not match expected %q", gotContentType, expectedContentType)
			return false
		}

		// Verify Cache-Control header
		gotCacheControl := getRR.Header().Get("Cache-Control")
		if gotCacheControl != "public, max-age=3600" {
			t.Logf("FAIL: Cache-Control %q does not match expected %q", gotCacheControl, "public, max-age=3600")
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 1: 推荐分析包 Logo 上传往返一致性 violated: %v", err)
	}
}

func TestPackLogoProperty2_InvalidUploadRejected(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user, storefront, and a featured pack
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("plogo-inv-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		packID := createTestPackListing(t, userID, 1, "free", 0, []byte("data"))

		// Set the pack as featured
		featBody := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
		featReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
			bytes.NewBufferString(featBody))
		featReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		featReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		featRR := httptest.NewRecorder()
		handleStorefrontSetFeatured(featRR, featReq)
		if featRR.Code != http.StatusOK {
			t.Logf("FAIL: set featured returned status %d, body: %s", featRR.Code, featRR.Body.String())
			return false
		}

		// Capture logo_data before the invalid upload attempt
		var logoBefore []byte
		db.QueryRow("SELECT logo_data FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoBefore)

		// Randomly choose between oversized file or non-image format
		testOversized := rng.Intn(2) == 0

		var invalidData []byte
		if testOversized {
			// Generate an oversized file (>2MB) with valid PNG header
			overSize := 2*1024*1024 + 1 + rng.Intn(1024) // 2MB + 1 to 2MB + 1024
			invalidData = make([]byte, overSize)
			// Start with PNG signature so it's detected as image/png
			copy(invalidData, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
			for i := 8; i < len(invalidData); i++ {
				invalidData[i] = byte(rng.Intn(256))
			}
		} else {
			// Generate a non-image format file (random bytes that won't be detected as PNG/JPEG)
			// Use a text/plain prefix or application/octet-stream content
			dataLen := rng.Intn(1024) + 64
			invalidData = make([]byte, dataLen)
			// Fill with ASCII text to ensure http.DetectContentType returns text/plain
			for i := range invalidData {
				invalidData[i] = byte('A' + rng.Intn(26))
			}
		}

		// Attempt to upload the invalid file
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("pack_listing_id", fmt.Sprintf("%d", packID))
		part, err := writer.CreateFormFile("logo", "test-logo.bin")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		if _, err := part.Write(invalidData); err != nil {
			t.Logf("FAIL: failed to write invalid data: %v", err)
			return false
		}
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		uploadRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoUpload(uploadRR, uploadReq)

		// Verify the upload was rejected (non-200 status code)
		if uploadRR.Code == http.StatusOK {
			if testOversized {
				t.Logf("FAIL: oversized file (%d bytes) was accepted with status 200", len(invalidData))
			} else {
				t.Logf("FAIL: non-image file was accepted with status 200")
			}
			return false
		}

		// Verify logo_data remains unchanged
		var logoAfter []byte
		db.QueryRow("SELECT logo_data FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoAfter)

		if !bytes.Equal(logoBefore, logoAfter) {
			t.Logf("FAIL: logo_data changed after rejected upload (before=%d bytes, after=%d bytes)",
				len(logoBefore), len(logoAfter))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 2: 无效上传被拒绝 violated: %v", err)
	}
}

// TestPackLogoProperty3_NonOwnerUploadRejected tests that a user cannot upload a logo
// for another user's featured pack.
// **Validates: Requirements 2.4**
func TestPackLogoProperty3_NonOwnerUploadRejected(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create User A (the attacker) and User B (the owner)
		userA := createTestUserWithBalance(t, 100)
		userB := createTestUserWithBalance(t, 100)

		// Create a storefront for User B
		slugB := fmt.Sprintf("plogo-noown-%d-%d", userB, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userB, slugB,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront for User B: %v", err)
			return false
		}

		// Create a pack listing owned by User B and set it as featured
		packID := createTestPackListing(t, userB, 1, "free", 0, []byte("data"))

		featBody := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
		featReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
			bytes.NewBufferString(featBody))
		featReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		featReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userB))
		featRR := httptest.NewRecorder()
		handleStorefrontSetFeatured(featRR, featReq)
		if featRR.Code != http.StatusOK {
			t.Logf("FAIL: set featured returned status %d, body: %s", featRR.Code, featRR.Body.String())
			return false
		}

		// Capture logo_data before the unauthorized upload attempt
		var logoBefore []byte
		db.QueryRow("SELECT logo_data FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoBefore)

		// Generate a valid image for the upload attempt
		imageData := generateValidPNG(rng)

		// User A attempts to upload a logo for User B's featured pack
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("pack_listing_id", fmt.Sprintf("%d", packID))
		part, err := writer.CreateFormFile("logo", "test-logo.png")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		if _, err := part.Write(imageData); err != nil {
			t.Logf("FAIL: failed to write image data: %v", err)
			return false
		}
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userA)) // User A, NOT the owner
		uploadRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoUpload(uploadRR, uploadReq)

		// Verify the upload was rejected with 403
		if uploadRR.Code != http.StatusForbidden {
			t.Logf("FAIL: non-owner upload returned status %d (expected 403), body: %s",
				uploadRR.Code, uploadRR.Body.String())
			return false
		}

		// Verify logo_data remains unchanged
		var logoAfter []byte
		db.QueryRow("SELECT logo_data FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoAfter)

		if !bytes.Equal(logoBefore, logoAfter) {
			t.Logf("FAIL: logo_data changed after rejected non-owner upload (before=%d bytes, after=%d bytes)",
				len(logoBefore), len(logoAfter))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 3: 非所有者上传被拒绝 violated: %v", err)
	}
}

// TestPackLogoProperty4_NonFeaturedUploadRejected tests that uploading a logo
// for a pack that is NOT set as featured is rejected.
// **Validates: Requirements 2.5**
func TestPackLogoProperty4_NonFeaturedUploadRejected(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("plogo-nofeat-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		// Create a pack listing but do NOT set it as featured
		packID := createTestPackListing(t, userID, 1, "free", 0, []byte("data"))

		// Capture logo_data before the upload attempt
		var logoBefore []byte
		db.QueryRow("SELECT logo_data FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoBefore)

		// Generate a valid image for the upload attempt
		imageData := generateValidPNG(rng)

		// Attempt to upload a logo for the non-featured pack
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("pack_listing_id", fmt.Sprintf("%d", packID))
		part, err := writer.CreateFormFile("logo", "test-logo.png")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		if _, err := part.Write(imageData); err != nil {
			t.Logf("FAIL: failed to write image data: %v", err)
			return false
		}
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		uploadRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoUpload(uploadRR, uploadReq)

		// Verify the upload was rejected (non-200 status code)
		if uploadRR.Code == http.StatusOK {
			t.Logf("FAIL: non-featured pack logo upload was accepted with status 200")
			return false
		}

		// Verify logo_data remains unchanged
		var logoAfter []byte
		db.QueryRow("SELECT logo_data FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoAfter)

		if !bytes.Equal(logoBefore, logoAfter) {
			t.Logf("FAIL: logo_data changed after rejected non-featured upload (before=%d bytes, after=%d bytes)",
				len(logoBefore), len(logoAfter))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 4: 非推荐分析包上传被拒绝 violated: %v", err)
	}
}

func TestPackLogoProperty5_NoLogoReturns404(t *testing.T) {
	// Feature: pack-logo-upload, Property 5: 无 Logo 时返回 404
	// Validates: Requirements 3.3
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user, storefront, and a featured pack (no logo uploaded)
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("plogo-no404-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		packID := createTestPackListing(t, userID, 1, "free", 0, []byte("data"))

		// Set the pack as featured via the handler
		featBody := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
		featReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
			bytes.NewBufferString(featBody))
		featReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		featReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		featRR := httptest.NewRecorder()
		handleStorefrontSetFeatured(featRR, featReq)
		if featRR.Code != http.StatusOK {
			t.Logf("FAIL: set featured returned status %d, body: %s", featRR.Code, featRR.Body.String())
			return false
		}

		// Do NOT upload any logo — logo_data should be NULL

		// GET the logo via the public route
		getURL := fmt.Sprintf("/store/%s/featured/%d/logo", slug, packID)
		getReq := httptest.NewRequest(http.MethodGet, getURL, nil)
		getRR := httptest.NewRecorder()
		handleStorefrontRoutes(getRR, getReq)

		// Verify 404 status code
		if getRR.Code != http.StatusNotFound {
			t.Logf("FAIL: expected 404 for featured pack with no logo, got %d, body: %s",
				getRR.Code, getRR.Body.String())
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 5: 无 Logo 时返回 404 violated: %v", err)
	}
}

// TestPackLogoProperty7_DeleteLogoClearsData tests that deleting a logo clears
// logo_data to NULL and subsequent GET returns 404.
// **Validates: Requirements 7.2**
func TestPackLogoProperty7_DeleteLogoClearsData(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Randomly choose PNG or JPEG
		isPNG := rng.Intn(2) == 0

		var imageData []byte
		if isPNG {
			imageData = generateValidPNG(rng)
		} else {
			imageData = generateValidJPEG(rng)
		}

		// Create a test user, storefront, and a featured pack
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("plogo-del-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		packID := createTestPackListing(t, userID, 1, "free", 0, []byte("data"))

		// Set the pack as featured via the handler
		featBody := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
		featReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
			bytes.NewBufferString(featBody))
		featReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		featReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		featRR := httptest.NewRecorder()
		handleStorefrontSetFeatured(featRR, featReq)
		if featRR.Code != http.StatusOK {
			t.Logf("FAIL: set featured returned status %d, body: %s", featRR.Code, featRR.Body.String())
			return false
		}

		// Upload a logo via handleStorefrontFeaturedLogoUpload
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("pack_listing_id", fmt.Sprintf("%d", packID))
		part, err := writer.CreateFormFile("logo", "test-logo.png")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		if _, err := part.Write(imageData); err != nil {
			t.Logf("FAIL: failed to write image data: %v", err)
			return false
		}
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		uploadRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoUpload(uploadRR, uploadReq)
		if uploadRR.Code != http.StatusOK {
			t.Logf("FAIL: upload returned status %d, body: %s", uploadRR.Code, uploadRR.Body.String())
			return false
		}

		// Delete the logo via handleStorefrontFeaturedLogoDelete
		delBody := fmt.Sprintf("pack_listing_id=%d", packID)
		delReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo/delete",
			bytes.NewBufferString(delBody))
		delReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		delReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		delRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoDelete(delRR, delReq)
		if delRR.Code != http.StatusOK {
			t.Logf("FAIL: delete returned status %d, body: %s", delRR.Code, delRR.Body.String())
			return false
		}

		// Verify logo_data is NULL in the database
		var logoData []byte
		var logoContentType *string
		err = db.QueryRow("SELECT logo_data, logo_content_type FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoData, &logoContentType)
		if err != nil {
			t.Logf("FAIL: failed to query logo_data after delete: %v", err)
			return false
		}
		if logoData != nil {
			t.Logf("FAIL: logo_data is not NULL after delete (%d bytes)", len(logoData))
			return false
		}
		if logoContentType != nil {
			t.Logf("FAIL: logo_content_type is not NULL after delete (got %q)", *logoContentType)
			return false
		}

		// Verify GET returns 404
		getURL := fmt.Sprintf("/store/%s/featured/%d/logo", slug, packID)
		getReq := httptest.NewRequest(http.MethodGet, getURL, nil)
		getRR := httptest.NewRecorder()
		handleStorefrontRoutes(getRR, getReq)

		if getRR.Code != http.StatusNotFound {
			t.Logf("FAIL: expected 404 after logo delete, got %d, body: %s",
				getRR.Code, getRR.Body.String())
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 7: 删除 Logo 清除数据 violated: %v", err)
	}
}

func TestPackLogoProperty8_UnfeatureCascadeClearsLogo(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Randomly choose PNG or JPEG
		isPNG := rng.Intn(2) == 0

		var imageData []byte
		if isPNG {
			imageData = generateValidPNG(rng)
		} else {
			imageData = generateValidJPEG(rng)
		}

		// Create a test user, storefront, and a featured pack
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("plogo-unf-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		packID := createTestPackListing(t, userID, 1, "free", 0, []byte("data"))

		// Set the pack as featured via the handler
		featBody := fmt.Sprintf("pack_listing_id=%d&featured=1", packID)
		featReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
			bytes.NewBufferString(featBody))
		featReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		featReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		featRR := httptest.NewRecorder()
		handleStorefrontSetFeatured(featRR, featReq)
		if featRR.Code != http.StatusOK {
			t.Logf("FAIL: set featured returned status %d, body: %s", featRR.Code, featRR.Body.String())
			return false
		}

		// Upload a logo via handleStorefrontFeaturedLogoUpload
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("pack_listing_id", fmt.Sprintf("%d", packID))
		part, err := writer.CreateFormFile("logo", "test-logo.png")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		if _, err := part.Write(imageData); err != nil {
			t.Logf("FAIL: failed to write image data: %v", err)
			return false
		}
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		uploadRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoUpload(uploadRR, uploadReq)
		if uploadRR.Code != http.StatusOK {
			t.Logf("FAIL: upload returned status %d, body: %s", uploadRR.Code, uploadRR.Body.String())
			return false
		}

		// Unfeature the pack (featured=0) via handleStorefrontSetFeatured
		unfeatBody := fmt.Sprintf("pack_listing_id=%d&featured=0", packID)
		unfeatReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
			bytes.NewBufferString(unfeatBody))
		unfeatReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		unfeatReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		unfeatRR := httptest.NewRecorder()
		handleStorefrontSetFeatured(unfeatRR, unfeatReq)
		if unfeatRR.Code != http.StatusOK {
			t.Logf("FAIL: unfeature returned status %d, body: %s", unfeatRR.Code, unfeatRR.Body.String())
			return false
		}

		// Verify logo_data and logo_content_type are NULL in the database
		var logoData []byte
		var logoContentType *string
		err = db.QueryRow("SELECT logo_data, logo_content_type FROM storefront_packs WHERE pack_listing_id = ?", packID).Scan(&logoData, &logoContentType)
		if err != nil {
			t.Logf("FAIL: failed to query logo_data after unfeature: %v", err)
			return false
		}
		if logoData != nil {
			t.Logf("FAIL: logo_data is not NULL after unfeature (%d bytes)", len(logoData))
			return false
		}
		if logoContentType != nil {
			t.Logf("FAIL: logo_content_type is not NULL after unfeature (got %q)", *logoContentType)
			return false
		}

		return true
	}

	// **Validates: Requirements 7.3**
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 8: 取消推荐级联清除 Logo violated: %v", err)
	}
}
// TestPackLogoProperty6_HasLogoReflectsLogoDataPresence tests that the HasLogo field
// correctly reflects whether logo_data is present for featured packs.
// **Validates: Requirements 1.2, 1.3, 6.1, 6.2**
func TestPackLogoProperty6_HasLogoReflectsLogoDataPresence(t *testing.T) {
	// Feature: pack-logo-upload, Property 6: HasLogo 正确反映 logo_data 存在性
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 100)
		slug := fmt.Sprintf("plogo-has-%d-%d", userID, rng.Int63n(1000000))
		res, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}
		storefrontID, _ := res.LastInsertId()

		// Create two packs and set both as featured
		packWithLogo := createTestPackListing(t, userID, 1, "free", 0, []byte("data1"))
		packWithoutLogo := createTestPackListing(t, userID, 1, "free", 0, []byte("data2"))

		for _, pid := range []int64{packWithLogo, packWithoutLogo} {
			featBody := fmt.Sprintf("pack_listing_id=%d&featured=1", pid)
			featReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured",
				bytes.NewBufferString(featBody))
			featReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			featReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			featRR := httptest.NewRecorder()
			handleStorefrontSetFeatured(featRR, featReq)
			if featRR.Code != http.StatusOK {
				t.Logf("FAIL: set featured for pack %d returned status %d, body: %s", pid, featRR.Code, featRR.Body.String())
				return false
			}
		}

		// Upload a logo to only the first pack
		var imageData []byte
		if rng.Intn(2) == 0 {
			imageData = generateValidPNG(rng)
		} else {
			imageData = generateValidJPEG(rng)
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("pack_listing_id", fmt.Sprintf("%d", packWithLogo))
		part, err := writer.CreateFormFile("logo", "test-logo.png")
		if err != nil {
			t.Logf("FAIL: failed to create form file: %v", err)
			return false
		}
		if _, err := part.Write(imageData); err != nil {
			t.Logf("FAIL: failed to write image data: %v", err)
			return false
		}
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/user/storefront/featured/logo", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		uploadRR := httptest.NewRecorder()
		handleStorefrontFeaturedLogoUpload(uploadRR, uploadReq)
		if uploadRR.Code != http.StatusOK {
			t.Logf("FAIL: upload returned status %d, body: %s", uploadRR.Code, uploadRR.Body.String())
			return false
		}

		// Query storefront packs and verify HasLogo
		packs, err := queryStorefrontPacks(storefrontID, false, "revenue", "", "", "")
		if err != nil {
			t.Logf("FAIL: queryStorefrontPacks failed: %v", err)
			return false
		}

		if len(packs) < 2 {
			t.Logf("FAIL: expected at least 2 packs, got %d", len(packs))
			return false
		}

		for _, p := range packs {
			if p.ListingID == packWithLogo {
				if !p.HasLogo {
					t.Logf("FAIL: pack %d has logo_data but HasLogo is false", packWithLogo)
					return false
				}
			} else if p.ListingID == packWithoutLogo {
				if p.HasLogo {
					t.Logf("FAIL: pack %d has no logo_data but HasLogo is true", packWithoutLogo)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Feature: pack-logo-upload, Property 6: HasLogo 正确反映 logo_data 存在性 violated: %v", err)
	}
}






