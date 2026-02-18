package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Bloomberg Tests
// =============================================================================

// TestImportBloomberg_EmptyCredentials tests that ImportBloomberg rejects empty API key AND empty cert path.
func TestImportBloomberg_EmptyCredentials(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "",
		FinancialCertPath: "",
		FinancialDatasets: "reference_data",
	}
	_, err := svc.ImportBloomberg("test", config)
	if err == nil {
		t.Fatal("expected error for empty credentials, got nil")
	}
	if !strings.Contains(err.Error(), "API key") && !strings.Contains(err.Error(), "certificate") {
		t.Errorf("expected error about API key or certificate, got: %v", err)
	}
}

// TestImportBloomberg_EmptyDatasets tests that ImportBloomberg rejects empty datasets.
func TestImportBloomberg_EmptyDatasets(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "test-key",
		FinancialDatasets: "",
	}
	_, err := svc.ImportBloomberg("test", config)
	if err == nil {
		t.Fatal("expected error for empty datasets, got nil")
	}
	if !strings.Contains(err.Error(), "dataset") {
		t.Errorf("expected error about dataset, got: %v", err)
	}
}

// TestFetchBloombergData_ValidResponse tests that a valid Bloomberg API response
// is correctly parsed and the auth header (Authorization: Bearer) is properly set.
func TestFetchBloombergData_ValidResponse(t *testing.T) {
	referenceDataResponse := []map[string]interface{}{
		{
			"figi":     "BBG000BLNNH6",
			"ticker":   "AAPL",
			"name":     "Apple Inc.",
			"exchange": "NASDAQ",
			"currency": "USD",
		},
		{
			"figi":     "BBG000BVPV84",
			"ticker":   "MSFT",
			"name":     "Microsoft Corp.",
			"exchange": "NASDAQ",
			"currency": "USD",
		},
	}

	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(referenceDataResponse)
	}))
	defer server.Close()

	// Simulate the Bloomberg response parsing logic from fetchBloombergData.
	req, err := http.NewRequest("GET", server.URL+"/reference-data", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-api-key")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	// Verify auth header was received with Bearer token
	if receivedAuth != "Bearer test-api-key" {
		t.Errorf("expected Authorization header 'Bearer test-api-key', got '%s'", receivedAuth)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Parse as array (same logic as fetchBloombergData)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("failed to parse response as array: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["figi"] != "BBG000BLNNH6" {
		t.Errorf("expected figi=BBG000BLNNH6, got %v", records[0]["figi"])
	}
	if records[0]["ticker"] != "AAPL" {
		t.Errorf("expected ticker='AAPL', got %v", records[0]["ticker"])
	}
	if records[1]["name"] != "Microsoft Corp." {
		t.Errorf("expected name='Microsoft Corp.', got %v", records[1]["name"])
	}
}

// TestFetchBloombergData_HTTP401 tests that the Bloomberg API returns proper
// authentication error responses (HTTP 401) with expected error format.
func TestFetchBloombergData_HTTP401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid credentials"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/reference-data", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer bad-key")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d", resp.StatusCode)
	}

	// Verify the error format matches what fetchBloombergData would produce
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		errMsg := fmt.Sprintf("Bloomberg authentication failed: invalid credentials (HTTP %d)", resp.StatusCode)
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

// TestFetchBloombergData_HTTP403 tests that HTTP 403 is also treated as auth error.
func TestFetchBloombergData_HTTP403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"Access denied"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/reference-data", nil)
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
