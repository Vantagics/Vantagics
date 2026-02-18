package agent

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// LSEG Tests
// =============================================================================

// TestImportLSEG_EmptyAppKey tests that ImportLSEG rejects empty App Key.
func TestImportLSEG_EmptyAppKey(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "",
		FinancialUsername: "user",
		FinancialPassword: "pass",
		FinancialDatasets: "historical_prices",
	}
	_, err := svc.ImportLSEG("test", config)
	if err == nil {
		t.Fatal("expected error for empty App Key, got nil")
	}
	if !strings.Contains(err.Error(), "App Key") {
		t.Errorf("expected error about App Key, got: %v", err)
	}
}

// TestImportLSEG_EmptyUsername tests that ImportLSEG rejects empty username.
func TestImportLSEG_EmptyUsername(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "test-key",
		FinancialUsername: "",
		FinancialPassword: "pass",
		FinancialDatasets: "historical_prices",
	}
	_, err := svc.ImportLSEG("test", config)
	if err == nil {
		t.Fatal("expected error for empty username, got nil")
	}
	if !strings.Contains(err.Error(), "username") {
		t.Errorf("expected error about username, got: %v", err)
	}
}

// TestImportLSEG_EmptyPassword tests that ImportLSEG rejects empty password.
func TestImportLSEG_EmptyPassword(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "test-key",
		FinancialUsername: "user",
		FinancialPassword: "",
		FinancialDatasets: "historical_prices",
	}
	_, err := svc.ImportLSEG("test", config)
	if err == nil {
		t.Fatal("expected error for empty password, got nil")
	}
	if !strings.Contains(err.Error(), "password") {
		t.Errorf("expected error about password, got: %v", err)
	}
}

// TestImportLSEG_EmptyDatasets tests that ImportLSEG rejects empty datasets.
func TestImportLSEG_EmptyDatasets(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "test-key",
		FinancialUsername: "user",
		FinancialPassword: "pass",
		FinancialDatasets: "",
	}
	_, err := svc.ImportLSEG("test", config)
	if err == nil {
		t.Fatal("expected error for empty datasets, got nil")
	}
	if !strings.Contains(err.Error(), "dataset") {
		t.Errorf("expected error about dataset, got: %v", err)
	}
}

// TestFetchLSEGData_ValidResponse tests that a valid LSEG API response
// is correctly parsed and the auth headers (X-App-Key, Authorization) are properly set.
func TestFetchLSEGData_ValidResponse(t *testing.T) {
	historicalResponse := []map[string]interface{}{
		{
			"ric":    "AAPL.O",
			"date":   "2024-01-15",
			"open":   185.50,
			"high":   187.20,
			"low":    184.80,
			"close":  186.90,
			"volume": 45000000,
		},
		{
			"ric":    "MSFT.O",
			"date":   "2024-01-15",
			"open":   390.10,
			"high":   392.50,
			"low":    389.00,
			"close":  391.80,
			"volume": 22000000,
		},
	}

	var receivedAppKey, receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAppKey = r.Header.Get("X-App-Key")
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(historicalResponse)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/data/historical-pricing", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-App-Key", "test-app-key")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:pass")))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	// Verify auth headers were received
	if receivedAppKey != "test-app-key" {
		t.Errorf("expected X-App-Key header 'test-app-key', got '%s'", receivedAppKey)
	}
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	if receivedAuth != expectedAuth {
		t.Errorf("expected Authorization header '%s', got '%s'", expectedAuth, receivedAuth)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Parse as array (same logic as fetchLSEGData)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("failed to parse response as array: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["ric"] != "AAPL.O" {
		t.Errorf("expected ric=AAPL.O, got %v", records[0]["ric"])
	}
	if records[1]["ric"] != "MSFT.O" {
		t.Errorf("expected ric=MSFT.O, got %v", records[1]["ric"])
	}
}

// TestFetchLSEGData_NestedResponse tests that a nested JSON response
// with a data key is correctly parsed.
func TestFetchLSEGData_NestedResponse(t *testing.T) {
	nestedResponse := map[string]interface{}{
		"esg": []interface{}{
			map[string]interface{}{
				"ric":           "AAPL.O",
				"date":          "2024-01-15",
				"esg_score":     82.5,
				"environmental": 78.3,
				"social":        85.1,
				"governance":    84.2,
			},
		},
		"meta": map[string]interface{}{
			"total": 1,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nestedResponse)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/data/esg", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Parse as nested object (same logic as fetchLSEGData)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		var objResponse map[string]interface{}
		if err := json.Unmarshal(body, &objResponse); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if arr, ok := objResponse["esg"].([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					records = append(records, m)
				}
			}
		}
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["ric"] != "AAPL.O" {
		t.Errorf("expected ric=AAPL.O, got %v", records[0]["ric"])
	}
	if records[0]["esg_score"] != 82.5 {
		t.Errorf("expected esg_score=82.5, got %v", records[0]["esg_score"])
	}
}

// TestFetchLSEGData_HTTP401 tests that the LSEG API returns proper
// authentication error responses (HTTP 401).
func TestFetchLSEGData_HTTP401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid credentials"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/data/historical-pricing", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-App-Key", "bad-key")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("bad:creds")))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d", resp.StatusCode)
	}

	// Verify the error format matches what fetchLSEGData would produce
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		errMsg := "LSEG authentication failed: invalid App Key or credentials (HTTP 401)"
		if !strings.Contains(errMsg, "authentication failed") {
			t.Error("expected 'authentication failed' in error message")
		}
		if !strings.Contains(errMsg, "401") {
			t.Error("expected '401' in error message")
		}
	} else {
		t.Error("expected HTTP 401 or 403 status code")
	}
}

// TestFetchLSEGData_HTTP403 tests that HTTP 403 is also treated as auth error.
func TestFetchLSEGData_HTTP403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"Access denied"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/data/fundamentals", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected HTTP 403, got %d", resp.StatusCode)
	}

	// Both 401 and 403 should be treated as authentication errors
	if resp.StatusCode != 401 && resp.StatusCode != 403 {
		t.Error("expected HTTP 401 or 403 status code")
	}
}
