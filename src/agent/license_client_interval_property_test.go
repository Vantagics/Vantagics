package agent

// Feature: credits-upload-interval-fix, Property 1: 客户端间隔守卫正确性
// **Validates: Requirements 1.1, 1.2, 2.1, 2.3, 3.2**

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"
)

// TestPropertyClientIntervalGuard verifies that ReportUsage() skips HTTP requests
// when lastReportAt is non-zero and time.Since(lastReportAt) < 1 hour,
// and executes requests when lastReportAt is zero or interval >= 1 hour.
func TestPropertyClientIntervalGuard(t *testing.T) {
	// Property: For any duration in [0, 2h), when lastReportAt is non-zero:
	//   - duration < 1h => ReportUsage skips (no HTTP request, returns nil)
	//   - duration >= 1h => ReportUsage executes (HTTP request sent)
	// When lastReportAt is zero (first report), always execute.

	config := &quick.Config{MaxCount: 100}

	// Sub-property 1: Non-zero lastReportAt within 1 hour => skip
	t.Run("NonZeroWithinOneHour_Skips", func(t *testing.T) {
		err := quick.Check(func(minutesRaw uint16) bool {
			// Generate duration in [1, 59] minutes (strictly within 1 hour, non-zero lastReportAt)
			minutes := int(minutesRaw%59) + 1
			duration := time.Duration(minutes) * time.Minute

			var requestCount int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt64(&requestCount, 1)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
			}))
			defer server.Close()

			client := &LicenseClient{
				serverURL: server.URL,
				sn:        "test-sn-001",
				data:      &ActivationData{UsedCredits: 10.0, CreditsMode: true},
				log:       func(s string) {},
			}
			// Set lastReportAt to `duration` ago (within 1 hour)
			client.lastReportAt = time.Now().Add(-duration)

			err := client.ReportUsage()

			// Should skip: no error, no HTTP request
			if err != nil {
				t.Logf("Expected nil error for duration=%v, got: %v", duration, err)
				return false
			}
			if atomic.LoadInt64(&requestCount) != 0 {
				t.Logf("Expected 0 requests for duration=%v, got: %d", duration, requestCount)
				return false
			}
			return true
		}, config)
		if err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// Sub-property 2: Non-zero lastReportAt beyond 1 hour => execute
	t.Run("NonZeroBeyondOneHour_Executes", func(t *testing.T) {
		err := quick.Check(func(extraMinutesRaw uint16) bool {
			// Generate duration in [60, 180] minutes (>= 1 hour)
			extraMinutes := int(extraMinutesRaw%121) + 60
			duration := time.Duration(extraMinutes) * time.Minute

			var requestCount int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt64(&requestCount, 1)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
			}))
			defer server.Close()

			client := &LicenseClient{
				serverURL: server.URL,
				sn:        "test-sn-002",
				data:      &ActivationData{UsedCredits: 5.0, CreditsMode: true},
				log:       func(s string) {},
			}
			// Set lastReportAt to `duration` ago (>= 1 hour)
			client.lastReportAt = time.Now().Add(-duration)

			err := client.ReportUsage()

			// Should execute: no error, HTTP request sent
			if err != nil {
				t.Logf("Expected nil error for duration=%v, got: %v", duration, err)
				return false
			}
			if atomic.LoadInt64(&requestCount) != 1 {
				t.Logf("Expected 1 request for duration=%v, got: %d", duration, requestCount)
				return false
			}
			return true
		}, config)
		if err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// Sub-property 3: Zero lastReportAt (first report) => always execute
	t.Run("ZeroLastReportAt_AlwaysExecutes", func(t *testing.T) {
		err := quick.Check(func(creditsRaw uint16) bool {
			credits := float64(creditsRaw%1000) + 1.0

			var requestCount int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt64(&requestCount, 1)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
			}))
			defer server.Close()

			client := &LicenseClient{
				serverURL: server.URL,
				sn:        "test-sn-003",
				data:      &ActivationData{UsedCredits: credits, CreditsMode: true},
				log:       func(s string) {},
			}
			// lastReportAt is zero value (first report)

			err := client.ReportUsage()

			// Should execute: no error, HTTP request sent
			if err != nil {
				t.Logf("Expected nil error for first report with credits=%.1f, got: %v", credits, err)
				return false
			}
			if atomic.LoadInt64(&requestCount) != 1 {
				t.Logf("Expected 1 request for first report, got: %d", requestCount)
				return false
			}
			return true
		}, config)
		if err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// Sub-property 4: Boundary - exactly 1 hour should execute (>= 1 hour)
	t.Run("ExactlyOneHour_Executes", func(t *testing.T) {
		var requestCount int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		}))
		defer server.Close()

		client := &LicenseClient{
			serverURL: server.URL,
			sn:        "test-sn-004",
			data:      &ActivationData{UsedCredits: 10.0, CreditsMode: true},
			log:       func(s string) {},
		}
		// Set lastReportAt to exactly 1 hour ago (plus a small buffer for execution time)
		client.lastReportAt = time.Now().Add(-time.Hour - time.Second)

		err := client.ReportUsage()
		if err != nil {
			t.Errorf("Expected nil error at 1h boundary, got: %v", err)
		}
		if atomic.LoadInt64(&requestCount) != 1 {
			t.Errorf("Expected 1 request at 1h boundary, got: %d", requestCount)
		}
	})
}
