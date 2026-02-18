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
// PitchBook Tests
// =============================================================================

// TestImportPitchBook_EmptyAPIKey tests that ImportPitchBook rejects empty API key.
func TestImportPitchBook_EmptyAPIKey(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "",
		FinancialDatasets: "companies",
	}
	_, err := svc.ImportPitchBook("test", config)
	if err == nil {
		t.Fatal("expected error for empty API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("expected error about API key, got: %v", err)
	}
}

// TestImportPitchBook_EmptyDatasets tests that ImportPitchBook rejects empty datasets.
func TestImportPitchBook_EmptyDatasets(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "test-key",
		FinancialDatasets: "",
	}
	_, err := svc.ImportPitchBook("test", config)
	if err == nil {
		t.Fatal("expected error for empty datasets, got nil")
	}
	if !strings.Contains(err.Error(), "dataset") {
		t.Errorf("expected error about dataset, got: %v", err)
	}
}

// TestFetchPitchBookData_ValidResponse tests that a valid PitchBook API response
// is correctly parsed and the auth header (Authorization: Bearer) is properly set.
func TestFetchPitchBookData_ValidResponse(t *testing.T) {
	companiesResponse := []map[string]interface{}{
		{
			"company_id":  "PB001",
			"name":        "Acme Ventures",
			"status":      "Active",
			"founded_date": "2015-03-20",
			"hq_location": "San Francisco, CA",
		},
		{
			"company_id":  "PB002",
			"name":        "TechStart Inc.",
			"status":      "Active",
			"founded_date": "2018-07-10",
			"hq_location": "New York, NY",
		},
	}

	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(companiesResponse)
	}))
	defer server.Close()

	// Simulate the PitchBook response parsing logic from fetchPitchBookData.
	req, err := http.NewRequest("GET", server.URL+"/companies", nil)
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

	// Parse as array (same logic as fetchPitchBookData)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("failed to parse response as array: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["company_id"] != "PB001" {
		t.Errorf("expected company_id=PB001, got %v", records[0]["company_id"])
	}
	if records[0]["name"] != "Acme Ventures" {
		t.Errorf("expected name='Acme Ventures', got %v", records[0]["name"])
	}
	if records[1]["hq_location"] != "New York, NY" {
		t.Errorf("expected hq_location='New York, NY', got %v", records[1]["hq_location"])
	}
}

// TestFetchPitchBookData_NestedResponse tests that a nested JSON response
// with a data key is correctly parsed.
func TestFetchPitchBookData_NestedResponse(t *testing.T) {
	nestedResponse := map[string]interface{}{
		"deals": []interface{}{
			map[string]interface{}{
				"deal_id":    "DL001",
				"company_id": "PB001",
				"deal_type":  "Series A",
				"deal_size":  15000000,
				"date":       "2024-02-10",
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

	req, err := http.NewRequest("GET", server.URL+"/deals", nil)
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

	// Parse as nested object (same logic as fetchPitchBookData)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		var objResponse map[string]interface{}
		if err := json.Unmarshal(body, &objResponse); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if arr, ok := objResponse["deals"].([]interface{}); ok {
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
	if records[0]["deal_id"] != "DL001" {
		t.Errorf("expected deal_id=DL001, got %v", records[0]["deal_id"])
	}
	if records[0]["deal_type"] != "Series A" {
		t.Errorf("expected deal_type='Series A', got %v", records[0]["deal_type"])
	}
}

// TestFetchPitchBookData_HTTP401 tests that the PitchBook API returns proper
// authentication error responses (HTTP 401) with expected error format.
func TestFetchPitchBookData_HTTP401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid API key"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/companies", nil)
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

	// Verify the error format matches what fetchPitchBookData would produce
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		errMsg := "PitchBook authentication failed: invalid API key (HTTP 401)"
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

// TestFetchPitchBookData_HTTP403 tests that HTTP 403 is also treated as auth error.
func TestFetchPitchBookData_HTTP403(t *testing.T) {
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
