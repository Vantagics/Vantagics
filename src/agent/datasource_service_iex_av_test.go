package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// IEX Cloud Tests
// =============================================================================

// TestFetchIEXCloudEndpoint_ValidQuote tests fetching a single-object quote response.
// IEX Cloud /quote endpoint returns a single JSON object, not an array.
func TestFetchIEXCloudEndpoint_ValidQuote(t *testing.T) {
	quoteResponse := map[string]interface{}{
		"symbol":        "AAPL",
		"latestPrice":   150.25,
		"change":        1.5,
		"changePercent": 0.01,
		"volume":        50000000,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(quoteResponse)
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	result, err := svc.fetchIEXCloudEndpoint(client, server.URL+"/stock/AAPL/quote?token=test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0]["symbol"] != "AAPL" {
		t.Errorf("expected symbol=AAPL, got %v", result[0]["symbol"])
	}
}

// TestFetchIEXCloudEndpoint_ValidChartArray tests fetching an array response (chart endpoint).
func TestFetchIEXCloudEndpoint_ValidChartArray(t *testing.T) {
	chartResponse := []map[string]interface{}{
		{"date": "2024-01-01", "open": 148.0, "high": 151.0, "low": 147.5, "close": 150.0, "volume": 40000000},
		{"date": "2024-01-02", "open": 150.0, "high": 152.0, "low": 149.0, "close": 151.5, "volume": 35000000},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chartResponse)
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	result, err := svc.fetchIEXCloudEndpoint(client, server.URL+"/stock/AAPL/chart?token=test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if result[0]["date"] != "2024-01-01" {
		t.Errorf("expected date=2024-01-01, got %v", result[0]["date"])
	}
}

// TestFetchIEXCloudEndpoint_HTTP401 tests authentication error handling (401).
func TestFetchIEXCloudEndpoint_HTTP401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	_, err := svc.fetchIEXCloudEndpoint(client, server.URL+"/stock/AAPL/quote?token=bad")
	if err == nil {
		t.Fatal("expected error for HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected 'authentication failed' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

// TestFetchIEXCloudEndpoint_HTTP403 tests authentication error handling (403).
func TestFetchIEXCloudEndpoint_HTTP403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	_, err := svc.fetchIEXCloudEndpoint(client, server.URL+"/stock/AAPL/quote?token=expired")
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected 'authentication failed' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected '403' in error, got: %v", err)
	}
}

// =============================================================================
// Alpha Vantage Tests
// =============================================================================

// TestFetchAlphaVantageEndpoint_ValidTimeSeries tests a valid time series response.
func TestFetchAlphaVantageEndpoint_ValidTimeSeries(t *testing.T) {
	avResponse := map[string]interface{}{
		"Meta Data": map[string]interface{}{
			"1. Information": "Daily Prices",
			"2. Symbol":      "MSFT",
		},
		"Time Series (Daily)": map[string]interface{}{
			"2024-01-02": map[string]interface{}{
				"1. open": "370.00", "2. high": "375.00",
				"3. low": "368.00", "4. close": "374.50", "5. volume": "25000000",
			},
			"2024-01-03": map[string]interface{}{
				"1. open": "374.50", "2. high": "378.00",
				"3. low": "373.00", "4. close": "377.00", "5. volume": "22000000",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(avResponse)
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	url := fmt.Sprintf("%s?function=TIME_SERIES_DAILY&symbol=MSFT&apikey=test", server.URL)
	result, err := svc.fetchAlphaVantageEndpoint(client, url, "time_series", "MSFT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	// Verify rows have expected fields
	for _, row := range result {
		if row["symbol"] != "MSFT" {
			t.Errorf("expected symbol=MSFT, got %v", row["symbol"])
		}
		if _, ok := row["date"]; !ok {
			t.Error("expected 'date' field in row")
		}
	}
}

// TestFetchAlphaVantageEndpoint_HTTP401 tests authentication error handling.
func TestFetchAlphaVantageEndpoint_HTTP401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	url := fmt.Sprintf("%s?function=TIME_SERIES_DAILY&symbol=MSFT&apikey=bad", server.URL)
	_, err := svc.fetchAlphaVantageEndpoint(client, url, "time_series", "MSFT")
	if err == nil {
		t.Fatal("expected error for HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected 'authentication failed' in error, got: %v", err)
	}
}

// TestFetchAlphaVantageEndpoint_HTTP429_Retry tests rate limit handling with retry.
// The server returns 429 on the first two attempts, then succeeds on the third.
func TestFetchAlphaVantageEndpoint_HTTP429_Retry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping retry test in short mode (involves sleep)")
	}

	attempt := 0
	avResponse := map[string]interface{}{
		"Meta Data": map[string]interface{}{
			"1. Information": "Daily Prices",
			"2. Symbol":      "GOOG",
		},
		"Time Series (Daily)": map[string]interface{}{
			"2024-01-02": map[string]interface{}{
				"1. open": "140.00", "2. high": "142.00",
				"3. low": "139.00", "4. close": "141.50", "5. volume": "15000000",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(avResponse)
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	url := fmt.Sprintf("%s?function=TIME_SERIES_DAILY&symbol=GOOG&apikey=test", server.URL)
	result, err := svc.fetchAlphaVantageEndpoint(client, url, "time_series", "GOOG")
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if attempt < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempt)
	}
}

// TestFetchAlphaVantageEndpoint_InBodyRateLimit tests the "Note" field rate limit
// that Alpha Vantage returns with HTTP 200 but a rate limit message in the body.
func TestFetchAlphaVantageEndpoint_InBodyRateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping retry test in short mode (involves sleep)")
	}

	// Alpha Vantage returns 200 with a "Note" field containing rate limit info
	rateLimitResponse := map[string]interface{}{
		"Note": "Thank you for using Alpha Vantage! Our standard API call frequency is 5 calls per minute and 500 calls per day.",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rateLimitResponse)
	}))
	defer server.Close()

	svc := &DataSourceService{}
	client := server.Client()

	url := fmt.Sprintf("%s?function=TIME_SERIES_DAILY&symbol=AAPL&apikey=test", server.URL)
	_, err := svc.fetchAlphaVantageEndpoint(client, url, "time_series", "AAPL")
	if err == nil {
		t.Fatal("expected error for in-body rate limit, got nil")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected 'rate limit' in error, got: %v", err)
	}
}

// =============================================================================
// Import Validation Tests
// =============================================================================

// TestImportIEXCloud_EmptyToken tests that ImportIEXCloud rejects empty token.
func TestImportIEXCloud_EmptyToken(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialToken:   "",
		FinancialSymbols: "AAPL",
	}
	_, err := svc.ImportIEXCloud("test", config)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	if !strings.Contains(err.Error(), "token") && !strings.Contains(err.Error(), "Token") {
		t.Errorf("expected error about token, got: %v", err)
	}
}

// TestImportIEXCloud_EmptySymbols tests that ImportIEXCloud rejects empty symbols.
func TestImportIEXCloud_EmptySymbols(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialToken:   "test-token",
		FinancialSymbols: "",
	}
	_, err := svc.ImportIEXCloud("test", config)
	if err == nil {
		t.Fatal("expected error for empty symbols, got nil")
	}
	if !strings.Contains(err.Error(), "symbol") {
		t.Errorf("expected error about symbol, got: %v", err)
	}
}

// TestImportAlphaVantage_EmptyAPIKey tests that ImportAlphaVantage rejects empty API key.
func TestImportAlphaVantage_EmptyAPIKey(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "",
		FinancialSymbols:  "AAPL",
		FinancialDataType: "time_series",
	}
	_, err := svc.ImportAlphaVantage("test", config)
	if err == nil {
		t.Fatal("expected error for empty API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("expected error about API key, got: %v", err)
	}
}

// TestImportAlphaVantage_EmptySymbols tests that ImportAlphaVantage rejects empty symbols.
func TestImportAlphaVantage_EmptySymbols(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "test-key",
		FinancialSymbols:  "",
		FinancialDataType: "time_series",
	}
	_, err := svc.ImportAlphaVantage("test", config)
	if err == nil {
		t.Fatal("expected error for empty symbols, got nil")
	}
	if !strings.Contains(err.Error(), "symbol") {
		t.Errorf("expected error about symbol, got: %v", err)
	}
}

// TestImportAlphaVantage_EmptyDataType tests that ImportAlphaVantage rejects empty data type.
func TestImportAlphaVantage_EmptyDataType(t *testing.T) {
	svc := &DataSourceService{}
	config := DataSourceConfig{
		FinancialAPIKey:   "test-key",
		FinancialSymbols:  "AAPL",
		FinancialDataType: "",
	}
	_, err := svc.ImportAlphaVantage("test", config)
	if err == nil {
		t.Fatal("expected error for empty data type, got nil")
	}
	if !strings.Contains(err.Error(), "data type") {
		t.Errorf("expected error about data type, got: %v", err)
	}
}
