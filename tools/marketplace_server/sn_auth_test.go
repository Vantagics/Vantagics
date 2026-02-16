package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockLicenseServer creates a mock License server that responds to /api/marketplace-verify.
// It returns the mock server and a cleanup function.
func mockLicenseServer(t *testing.T, sn, email string) (*httptest.Server, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/marketplace-verify" {
			var req struct {
				Token string `json:"token"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"sn":      sn,
				"email":   email,
			})
			return
		}
		http.NotFound(w, r)
	}))
	oldURL := licenseServerURL
	licenseServerURL = server.URL
	return server, func() {
		server.Close()
		licenseServerURL = oldURL
	}
}

// callSNLogin calls the handleSNLogin handler with a given license_token.
func callSNLogin(t *testing.T, licenseToken string) (int, map[string]interface{}) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"license_token": licenseToken})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sn-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleSNLogin(w, req)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return w.Code, resp
}

// Feature: marketplace-sn-auth-billing, Property 3: 用户登录幂等性
// For any valid SN, logging in twice via sn-login returns the same user_id,
// and the users table contains exactly one record for that SN.
// Validates: Requirements 2.3, 2.4, 2.5
func TestProperty3_UserLoginIdempotency(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100
	domains := []string{"example.com", "test.org", "mail.net", "company.co"}

	for i := 0; i < iterations; i++ {
		cleanup := setupTestDB(t)

		sn := fmt.Sprintf("SN-%04d-%04d", rng.Intn(9999), rng.Intn(9999))
		email := fmt.Sprintf("user%d@%s", rng.Intn(100000), domains[rng.Intn(len(domains))])

		mockServer, mockCleanup := mockLicenseServer(t, sn, email)
		_ = mockServer

		// First login
		code1, resp1 := callSNLogin(t, "mock-token-1")
		if code1 != http.StatusOK {
			t.Errorf("iteration %d: first login failed with status %d: %v", i, code1, resp1)
			mockCleanup()
			cleanup()
			continue
		}
		success1, _ := resp1["success"].(bool)
		if !success1 {
			t.Errorf("iteration %d: first login returned success=false", i)
			mockCleanup()
			cleanup()
			continue
		}
		user1, _ := resp1["user"].(map[string]interface{})
		userID1 := user1["id"]

		// Second login with same SN
		code2, resp2 := callSNLogin(t, "mock-token-2")
		if code2 != http.StatusOK {
			t.Errorf("iteration %d: second login failed with status %d: %v", i, code2, resp2)
			mockCleanup()
			cleanup()
			continue
		}
		success2, _ := resp2["success"].(bool)
		if !success2 {
			t.Errorf("iteration %d: second login returned success=false", i)
			mockCleanup()
			cleanup()
			continue
		}
		user2, _ := resp2["user"].(map[string]interface{})
		userID2 := user2["id"]

		// Property: same user_id returned
		if userID1 != userID2 {
			t.Errorf("iteration %d: user_id mismatch: first=%v, second=%v", i, userID1, userID2)
		}

		// Property: only one record in users table for this SN
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE auth_type='sn' AND auth_id=?", sn).Scan(&count)
		if err != nil {
			t.Errorf("iteration %d: failed to count users: %v", i, err)
		} else if count != 1 {
			t.Errorf("iteration %d: expected 1 user record for SN=%s, got %d", i, sn, count)
		}

		// Verify display_name is email prefix
		displayName, _ := user1["display_name"].(string)
		expectedPrefix := email
		if idx := len(email) - len(email); idx >= 0 {
			for j, c := range email {
				if c == '@' {
					expectedPrefix = email[:j]
					break
				}
			}
		}
		if displayName != expectedPrefix {
			t.Errorf("iteration %d: display_name=%s, expected=%s", i, displayName, expectedPrefix)
		}

		mockCleanup()
		cleanup()
	}
}
