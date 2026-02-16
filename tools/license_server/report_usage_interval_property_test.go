package main

// Feature: credits-upload-interval-fix, Property 3: 服务器端间隔强制执行
// **Validates: Requirements 4.1, 4.2, 4.3, 4.4**

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"

	_ "modernc.org/sqlite"
)

// setupReportUsageTestDB creates an in-memory SQLite DB with licenses and credits_usage_log tables.
// It sets the global db variable used by handleReportUsage.
func setupReportUsageTestDB(t *testing.T) func() {
	t.Helper()
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS licenses (
			sn TEXT PRIMARY KEY,
			created_at DATETIME,
			expires_at DATETIME,
			valid_days INTEGER DEFAULT 365,
			description TEXT,
			is_active INTEGER DEFAULT 1,
			usage_count INTEGER DEFAULT 0,
			last_used_at DATETIME,
			daily_analysis INTEGER DEFAULT 20,
			llm_group_id TEXT DEFAULT '',
			search_group_id TEXT DEFAULT '',
			product_id INTEGER DEFAULT 0,
			used_credits FLOAT DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS credits_usage_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sn TEXT NOT NULL,
			used_credits FLOAT NOT NULL,
			reported_at DATETIME NOT NULL,
			client_ip TEXT
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	return func() { db.Close() }
}

// generateTestSN creates a random SN string for testing using the existing generateSN function.
func generateTestSN() string {
	return generateSN()
}

// callReportUsage sends a POST request to handleReportUsage and returns the parsed JSON response.
func callReportUsage(t *testing.T, sn string, usedCredits float64) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"sn":           sn,
		"used_credits": usedCredits,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/report-usage", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleReportUsage(w, req)
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v (body: %s)", err, w.Body.String())
	}
	return resp
}

// countUsageLogs returns the number of credits_usage_log records for a given SN.
func countUsageLogs(t *testing.T, sn string) int {
	t.Helper()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM credits_usage_log WHERE sn=?", sn).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count usage logs: %v", err)
	}
	return count
}

// insertTestLicense inserts a license record for testing.
func insertTestLicense(t *testing.T, sn string) {
	t.Helper()
	_, err := db.Exec("INSERT OR REPLACE INTO licenses (sn, created_at, expires_at, is_active, used_credits) VALUES (?, datetime('now'), datetime('now', '+365 days'), 1, 0)", sn)
	if err != nil {
		t.Fatalf("failed to insert test license: %v", err)
	}
}

// insertUsageLog inserts a credits_usage_log record with a specific reported_at time.
func insertUsageLog(t *testing.T, sn string, usedCredits float64, reportedAt time.Time) {
	t.Helper()
	_, err := db.Exec("INSERT INTO credits_usage_log (sn, used_credits, reported_at, client_ip) VALUES (?, ?, ?, '127.0.0.1')",
		sn, usedCredits, reportedAt.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("failed to insert usage log: %v", err)
	}
}

// TestProperty3_ServerIntervalEnforcement_FirstReport verifies that when a SN has no prior
// usage logs (first report), the server accepts the report and inserts a log record.
// Sub-property 1: First report => success, log inserted.
func TestProperty3_ServerIntervalEnforcement_FirstReport(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupReportUsageTestDB(t)
		defer cleanup()

		sn := generateTestSN()
		rng := rand.New(rand.NewSource(seed))
		credits := float64(rng.Intn(10000)+1) / 100.0 // random positive credits

		insertTestLicense(t, sn)

		// No prior usage logs — first report
		resp := callReportUsage(t, sn, credits)

		success, _ := resp["success"].(bool)
		code, hasCode := resp["code"]
		if !success {
			t.Logf("FAIL: first report should succeed, got success=false for SN=%s", sn)
			return false
		}
		if hasCode && code == "THROTTLED" {
			t.Logf("FAIL: first report should not be THROTTLED for SN=%s", sn)
			return false
		}

		logCount := countUsageLogs(t, sn)
		if logCount != 1 {
			t.Logf("FAIL: expected 1 log record after first report, got %d for SN=%s", logCount, sn)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (sub-property 1: first report) failed: %v", err)
	}
}

// TestProperty3_ServerIntervalEnforcement_WithinOneHour verifies that when a SN has a prior
// usage log within the last hour, the server returns THROTTLED and does not insert a new log.
// Sub-property 2: Report within 1 hour => THROTTLED, no new log.
func TestProperty3_ServerIntervalEnforcement_WithinOneHour(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupReportUsageTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))
		sn := generateTestSN()
		credits := float64(rng.Intn(10000)+1) / 100.0

		insertTestLicense(t, sn)

		// Insert a prior usage log within the last hour (1 to 59 minutes ago)
		minutesAgo := rng.Intn(59) + 1 // 1..59 minutes
		priorTime := time.Now().Add(-time.Duration(minutesAgo) * time.Minute)
		insertUsageLog(t, sn, credits, priorTime)

		// Second report should be throttled
		resp := callReportUsage(t, sn, credits+1)

		success, _ := resp["success"].(bool)
		code, _ := resp["code"].(string)
		if !success {
			t.Logf("FAIL: THROTTLED response should have success=true, got false for SN=%s", sn)
			return false
		}
		if code != "THROTTLED" {
			t.Logf("FAIL: expected code=THROTTLED, got %q for SN=%s (minutesAgo=%d)", code, sn, minutesAgo)
			return false
		}

		logCount := countUsageLogs(t, sn)
		if logCount != 1 {
			t.Logf("FAIL: expected still 1 log record (no new insert), got %d for SN=%s", logCount, sn)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (sub-property 2: within one hour) failed: %v", err)
	}
}

// TestProperty3_ServerIntervalEnforcement_AfterOneHour verifies that when a SN has a prior
// usage log older than 1 hour, the server accepts the report and inserts a new log record.
// Sub-property 3: Report after 1+ hours => success, new log inserted.
func TestProperty3_ServerIntervalEnforcement_AfterOneHour(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupReportUsageTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))
		sn := generateTestSN()
		credits := float64(rng.Intn(10000)+1) / 100.0

		insertTestLicense(t, sn)

		// Insert a prior usage log more than 1 hour ago (61 to 1440 minutes = 1h1m to 24h)
		minutesAgo := rng.Intn(1380) + 61 // 61..1440 minutes
		priorTime := time.Now().Add(-time.Duration(minutesAgo) * time.Minute)
		insertUsageLog(t, sn, credits, priorTime)

		// Second report should succeed normally
		resp := callReportUsage(t, sn, credits+1)

		success, _ := resp["success"].(bool)
		code, hasCode := resp["code"]
		if !success {
			t.Logf("FAIL: report after 1+ hours should succeed, got success=false for SN=%s", sn)
			return false
		}
		if hasCode && code == "THROTTLED" {
			t.Logf("FAIL: report after 1+ hours should not be THROTTLED for SN=%s (minutesAgo=%d)", sn, minutesAgo)
			return false
		}

		logCount := countUsageLogs(t, sn)
		if logCount != 2 {
			t.Logf("FAIL: expected 2 log records after second report, got %d for SN=%s", logCount, sn)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (sub-property 3: after one hour) failed: %v", err)
	}
}

// TestProperty3_ServerIntervalEnforcement_ThrottledPreservesSuccessTrue is a focused check
// that THROTTLED responses always use success:true (Requirement 4.4).
func TestProperty3_ServerIntervalEnforcement_ThrottledPreservesSuccessTrue(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupReportUsageTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))
		sn := generateTestSN()
		credits := float64(rng.Intn(10000)+1) / 100.0

		insertTestLicense(t, sn)

		// Insert a very recent usage log (just now)
		insertUsageLog(t, sn, credits, time.Now())

		resp := callReportUsage(t, sn, credits+1)

		success, _ := resp["success"].(bool)
		code, _ := resp["code"].(string)

		// Requirement 4.4: THROTTLED must use success: true
		if code == "THROTTLED" && !success {
			t.Logf("FAIL: THROTTLED response must have success=true for SN=%s", sn)
			return false
		}
		// Should be throttled since we just inserted a log
		if code != "THROTTLED" {
			t.Logf("FAIL: expected THROTTLED for immediate re-report, got code=%q for SN=%s", code, sn)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 3 (THROTTLED preserves success:true) failed: %v", err)
	}
}
