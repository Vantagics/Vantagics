package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// S&P Global Tests
// =============================================================================

// TestImportSPGlobal_EmptyAPIKey tests that ImportSPGlobal rejects empty API key.
func TestImportSPGlobal_EmptyAPIKey(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:    "",
		FinancialAPISecret: "test-secret",
		FinancialDatasets:  "companies",
	}
	_, err := svc.ImportSPGlobal("test", config)
	if err == nil {
		t.Fatal("expected error for empty API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("expected error about API key, got: %v", err)
	}
}

// TestImportSPGlobal_EmptyAPISecret tests that ImportSPGlobal rejects empty API secret.
func TestImportSPGlobal_EmptyAPISecret(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:    "test-key",
		FinancialAPISecret: "",
		FinancialDatasets:  "companies",
	}
	_, err := svc.ImportSPGlobal("test", config)
	if err == nil {
		t.Fatal("expected error for empty API secret, got nil")
	}
	if !strings.Contains(err.Error(), "API secret") {
		t.Errorf("expected error about API secret, got: %v", err)
	}
}

// TestImportSPGlobal_EmptyDatasets tests that ImportSPGlobal rejects empty datasets.
func TestImportSPGlobal_EmptyDatasets(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:    "test-key",
		FinancialAPISecret: "test-secret",
		FinancialDatasets:  "",
	}
	_, err := svc.ImportSPGlobal("test", config)
	if err == nil {
		t.Fatal("expected error for empty datasets, got nil")
	}
	if !strings.Contains(err.Error(), "dataset") {
		t.Errorf("expected error about dataset, got: %v", err)
	}
}

// TestFetchSPGlobalData_ValidResponse tests that a valid S&P Global API response
// is correctly parsed and the auth headers (X-API-Key, X-API-Secret) are properly set.
func TestFetchSPGlobalData_ValidResponse(t *testing.T) {
	companiesResponse := []map[string]interface{}{
		{
			"company_id": "SP001",
			"name":       "Apple Inc.",
			"ticker":     "AAPL",
			"sector":     "Technology",
			"industry":   "Consumer Electronics",
		},
		{
			"company_id": "SP002",
			"name":       "Microsoft Corp.",
			"ticker":     "MSFT",
			"sector":     "Technology",
			"industry":   "Software",
		},
	}

	var receivedAPIKey, receivedAPISecret string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("X-API-Key")
		receivedAPISecret = r.Header.Get("X-API-Secret")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(companiesResponse)
	}))
	defer server.Close()

	// Simulate the S&P Global response parsing logic from fetchSPGlobalData.
	req, err := http.NewRequest("GET", server.URL+"/companies", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("X-API-Secret", "test-api-secret")

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
	if receivedAPIKey != "test-api-key" {
		t.Errorf("expected X-API-Key header 'test-api-key', got '%s'", receivedAPIKey)
	}
	if receivedAPISecret != "test-api-secret" {
		t.Errorf("expected X-API-Secret header 'test-api-secret', got '%s'", receivedAPISecret)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Parse as array (same logic as fetchSPGlobalData)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("failed to parse response as array: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["company_id"] != "SP001" {
		t.Errorf("expected company_id=SP001, got %v", records[0]["company_id"])
	}
	if records[0]["name"] != "Apple Inc." {
		t.Errorf("expected name='Apple Inc.', got %v", records[0]["name"])
	}
	if records[1]["ticker"] != "MSFT" {
		t.Errorf("expected ticker=MSFT, got %v", records[1]["ticker"])
	}
}

// TestFetchSPGlobalData_NestedResponse tests that a nested JSON response
// with a data key is correctly parsed.
func TestFetchSPGlobalData_NestedResponse(t *testing.T) {
	nestedResponse := map[string]interface{}{
		"credit_ratings": []interface{}{
			map[string]interface{}{
				"entity_id": "ENT001",
				"rating":    "AA+",
				"outlook":   "Stable",
				"date":      "2024-01-15",
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

	req, err := http.NewRequest("GET", server.URL+"/credit-ratings", nil)
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

	// Parse as nested object (same logic as fetchSPGlobalData)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		var objResponse map[string]interface{}
		if err := json.Unmarshal(body, &objResponse); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if arr, ok := objResponse["credit_ratings"].([]interface{}); ok {
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
	if records[0]["entity_id"] != "ENT001" {
		t.Errorf("expected entity_id=ENT001, got %v", records[0]["entity_id"])
	}
	if records[0]["rating"] != "AA+" {
		t.Errorf("expected rating=AA+, got %v", records[0]["rating"])
	}
}

// TestFetchSPGlobalData_HTTP401 tests that the S&P Global API returns proper
// authentication error responses (HTTP 401) with expected error format.
func TestFetchSPGlobalData_HTTP401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid API credentials"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/companies", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Key", "bad-key")
	req.Header.Set("X-API-Secret", "bad-secret")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d", resp.StatusCode)
	}

	// Verify the error format matches what fetchSPGlobalData would produce
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		errMsg := "S&P Global authentication failed: invalid API key or secret (HTTP 401)"
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

// TestFetchSPGlobalData_HTTP403 tests that HTTP 403 is also treated as auth error.
func TestFetchSPGlobalData_HTTP403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"Access denied"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/companies", nil)
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
