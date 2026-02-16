package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"vantagedata/config"
	"vantagedata/logger"
)

// Feature: purchased-pack-display-fix, Property 3: 下载文件落入 QAP 目录且临时文件被清理
// **Validates: Requirements 2.1, 2.2**
//
// For any successful marketplace download operation, the returned file path should be
// under {DataCacheDir}/qap/ directory, the file should exist at that path, and no
// temporary file should remain in os.TempDir() with the expected temp filename pattern.
func TestProperty3_DownloadFileLandsInQAPDirectoryAndTempCleaned(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate random .qap file content (1 byte to 4KB)
		contentSize := r.Intn(4096) + 1
		fakeQAPContent := make([]byte, contentSize)
		for i := range fakeQAPContent {
			fakeQAPContent[i] = byte(r.Intn(256))
		}

		// Generate a random listing ID (1-99999)
		listingID := int64(r.Intn(99999) + 1)

		// Set up a mock HTTP server that returns the fake .qap file
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Verify the request path matches expected pattern
			if req.Header.Get("Authorization") == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write(fakeQAPContent)
		}))
		defer mockServer.Close()

		// Create a temporary DataCacheDir for this test iteration
		dataCacheDir, err := os.MkdirTemp("", "pbt_datacache_*")
		if err != nil {
			t.Logf("seed=%d: failed to create temp data cache dir: %v", seed, err)
			return false
		}
		defer os.RemoveAll(dataCacheDir)

		// Create a temporary storage dir for config
		storageDir, err := os.MkdirTemp("", "pbt_storage_*")
		if err != nil {
			t.Logf("seed=%d: failed to create temp storage dir: %v", seed, err)
			return false
		}
		defer os.RemoveAll(storageDir)

		// Write a config file with the DataCacheDir set
		cfgData := config.Config{
			DataCacheDir: dataCacheDir,
		}
		cfgJSON, _ := json.Marshal(cfgData)
		if err := os.WriteFile(filepath.Join(storageDir, "config.json"), cfgJSON, 0644); err != nil {
			t.Logf("seed=%d: failed to write config file: %v", seed, err)
			return false
		}

		// Initialize a logger for the App
		logDir, err := os.MkdirTemp("", "pbt_logdir_*")
		if err != nil {
			t.Logf("seed=%d: failed to create temp log dir: %v", seed, err)
			return false
		}
		defer os.RemoveAll(logDir)

		lg := logger.NewLogger()
		if err := lg.Init(logDir); err != nil {
			t.Logf("seed=%d: failed to init logger: %v", seed, err)
			return false
		}
		defer lg.Close()

		// Initialize the App with the mock marketplace client
		app := &App{
			storageDir: storageDir,
			logger:     lg,
			marketplaceClient: &MarketplaceClient{
				ServerURL: mockServer.URL,
				Token:     "test-token-for-pbt",
				client:    mockServer.Client(),
			},
		}

		// Call DownloadMarketplacePack
		resultPath, err := app.DownloadMarketplacePack(listingID)
		if err != nil {
			t.Logf("seed=%d: DownloadMarketplacePack returned error: %v", seed, err)
			return false
		}

		// Property check 1: The returned file path should be under {DataCacheDir}/qap/
		qapDir := filepath.Join(dataCacheDir, "qap")
		if !strings.HasPrefix(resultPath, qapDir) {
			t.Logf("seed=%d: returned path %q is not under qap dir %q", seed, resultPath, qapDir)
			return false
		}

		// Property check 2: The file should exist at the returned path
		info, err := os.Stat(resultPath)
		if err != nil {
			t.Logf("seed=%d: file does not exist at returned path %q: %v", seed, resultPath, err)
			return false
		}
		if info.Size() != int64(contentSize) {
			t.Logf("seed=%d: file size mismatch: expected %d, got %d", seed, contentSize, info.Size())
			return false
		}

		// Property check 3: No temp file should remain in os.TempDir() with the expected pattern
		tmpDir := os.TempDir()
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Logf("seed=%d: failed to read temp dir: %v", seed, err)
			return false
		}
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "marketplace_pack_") && strings.HasSuffix(name, ".qap") {
				// Check if this temp file belongs to our listing ID
				if strings.Contains(name, fmt.Sprintf("marketplace_pack_%d_", listingID)) {
					t.Logf("seed=%d: temp file %q still exists in %s", seed, name, tmpDir)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (下载文件落入 QAP 目录且临时文件被清理) failed: %v", err)
	}
}
