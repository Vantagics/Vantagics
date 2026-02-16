package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) func() {
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
			product_id INTEGER DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS email_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT,
			sn TEXT,
			ip TEXT,
			created_at DATETIME,
			product_id INTEGER DEFAULT 0,
			UNIQUE(email, product_id)
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	os.Setenv("LICENSE_MARKETPLACE_SECRET", "test-secret-key-for-property-testing")
	return func() {
		db.Close()
		os.Unsetenv("LICENSE_MARKETPLACE_SECRET")
	}
}

// insertLicense inserts a license into the test database.
func insertLicense(t *testing.T, sn string, isActive bool, expiresAt time.Time) {
	t.Helper()
	active := 0
	if isActive {
		active = 1
	}
	_, err := db.Exec("INSERT OR REPLACE INTO licenses (sn, created_at, expires_at, is_active) VALUES (?, ?, ?, ?)",
		sn, time.Now(), expiresAt, active)
	if err != nil {
		t.Fatalf("failed to insert license: %v", err)
	}
}

// insertEmailRecord inserts an email record into the test database.
func insertEmailRecord(t *testing.T, email, sn string) {
	t.Helper()
	_, err := db.Exec("INSERT OR REPLACE INTO email_records (email, sn, ip, created_at) VALUES (?, ?, '127.0.0.1', ?)",
		email, sn, time.Now())
	if err != nil {
		t.Fatalf("failed to insert email record: %v", err)
	}
}

// callMarketplaceAuth calls the handleMarketplaceAuth handler and returns the parsed response.
func callMarketplaceAuth(t *testing.T, sn, email string) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"sn": sn, "email": email})
	req := httptest.NewRequest("POST", "/api/marketplace-auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleMarketplaceAuth(w, req)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

// callMarketplaceVerify calls the handleMarketplaceVerify handler and returns the parsed response.
func callMarketplaceVerify(t *testing.T, token string) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"token": token})
	req := httptest.NewRequest("POST", "/api/marketplace-verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleMarketplaceVerify(w, req)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

// signMarketplaceTokenWithExp is a test helper that signs a token with a custom expiration time.
func signMarketplaceTokenWithExp(sn, email, secret string, exp time.Time) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	payload := map[string]interface{}{
		"sn":      sn,
		"email":   email,
		"purpose": "marketplace_auth",
		"exp":     exp.Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + signatureB64, nil
}

// Feature: marketplace-sn-auth-billing, Property 1: SN+Email 认证验证一致性
// For any SN+Email combination, marketplace-auth returns success=true if and only if
// SN is valid, active, not expired, and email matches the SN's email record.
// Validates: Requirements 1.2, 1.3, 1.5, 1.6
func TestProperty1_SNEmailAuthValidationConsistency(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	knownSN := "TEST-AAAA-BBBB"
	knownEmail := "user@example.com"
	insertLicense(t, knownSN, true, time.Now().Add(24*time.Hour))
	insertEmailRecord(t, knownEmail, knownSN)

	disabledSN := "TEST-CCCC-DDDD"
	insertLicense(t, disabledSN, false, time.Now().Add(24*time.Hour))
	insertEmailRecord(t, "disabled@example.com", disabledSN)

	expiredSN := "TEST-EEEE-FFFF"
	insertLicense(t, expiredSN, true, time.Now().Add(-24*time.Hour))
	insertEmailRecord(t, "expired@example.com", expiredSN)

	type testCase struct {
		sn         string
		email      string
		snExists   bool
		isActive   bool
		notExpired bool
		emailMatch bool
	}

	generateCase := func() testCase {
		switch rng.Intn(6) {
		case 0:
			return testCase{knownSN, knownEmail, true, true, true, true}
		case 1:
			return testCase{fmt.Sprintf("FAKE-%04d-%04d", rng.Intn(9999), rng.Intn(9999)), knownEmail, false, false, false, false}
		case 2:
			return testCase{disabledSN, "disabled@example.com", true, false, true, true}
		case 3:
			return testCase{expiredSN, "expired@example.com", true, true, false, true}
		case 4:
			return testCase{knownSN, "wrong@example.com", true, true, true, false}
		default:
			return testCase{fmt.Sprintf("RAND-%04d-%04d", rng.Intn(9999), rng.Intn(9999)), fmt.Sprintf("rand%d@test.com", rng.Intn(10000)), false, false, false, false}
		}
	}

	for i := 0; i < iterations; i++ {
		tc := generateCase()
		resp := callMarketplaceAuth(t, tc.sn, tc.email)
		success, _ := resp["success"].(bool)
		code, _ := resp["code"].(string)
		shouldSucceed := tc.snExists && tc.isActive && tc.notExpired && tc.emailMatch

		if shouldSucceed && !success {
			t.Errorf("iteration %d: expected success for sn=%s email=%s, got failure (code=%s)", i, tc.sn, tc.email, code)
		}
		if !shouldSucceed && success {
			t.Errorf("iteration %d: expected failure for sn=%s email=%s, got success", i, tc.sn, tc.email)
		}
		if !success {
			if tc.snExists && !tc.isActive {
				if code != CodeSNDisabled {
					t.Errorf("iteration %d: disabled SN should return SN_DISABLED, got %s", i, code)
				}
			}
			if tc.snExists && tc.isActive && !tc.notExpired {
				if code != CodeSNExpired {
					t.Errorf("iteration %d: expired SN should return SN_EXPIRED, got %s", i, code)
				}
			}
			if tc.snExists && tc.isActive && tc.notExpired && !tc.emailMatch {
				if code != CodeEmailMismatch {
					t.Errorf("iteration %d: email mismatch should return EMAIL_MISMATCH, got %s", i, code)
				}
			}
		}
	}
}

// Feature: marketplace-sn-auth-billing, Property 2: 令牌签发与验证往返一致性
// For any valid SN+Email pair, a token issued by marketplace-auth, when verified by
// marketplace-verify, returns the same SN and Email values.
// Validates: Requirements 1.4, 1.7, 10.2, 10.3
func TestProperty2_TokenSignVerifyRoundtrip(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100
	domains := []string{"example.com", "test.org", "mail.net", "company.co"}

	for i := 0; i < iterations; i++ {
		sn := fmt.Sprintf("RT-%04d-%04d", rng.Intn(9999), rng.Intn(9999))
		email := fmt.Sprintf("user%d@%s", rng.Intn(100000), domains[rng.Intn(len(domains))])
		upperSN := strings.ToUpper(sn)

		insertLicense(t, upperSN, true, time.Now().Add(24*time.Hour))
		insertEmailRecord(t, email, upperSN)

		authResp := callMarketplaceAuth(t, sn, email)
		success, _ := authResp["success"].(bool)
		if !success {
			t.Errorf("iteration %d: auth failed for valid sn=%s email=%s: %v", i, sn, email, authResp)
			continue
		}
		token, ok := authResp["token"].(string)
		if !ok || token == "" {
			t.Errorf("iteration %d: no token returned for sn=%s email=%s", i, sn, email)
			continue
		}

		verifyResp := callMarketplaceVerify(t, token)
		verifySuccess, _ := verifyResp["success"].(bool)
		if !verifySuccess {
			t.Errorf("iteration %d: verify failed for token from sn=%s email=%s: %v", i, sn, email, verifyResp)
			continue
		}

		returnedSN, _ := verifyResp["sn"].(string)
		returnedEmail, _ := verifyResp["email"].(string)
		if returnedSN != upperSN {
			t.Errorf("iteration %d: SN mismatch: sent=%s, returned=%s", i, upperSN, returnedSN)
		}
		if !strings.EqualFold(returnedEmail, email) {
			t.Errorf("iteration %d: Email mismatch: sent=%s, returned=%s", i, email, returnedEmail)
		}
	}
}

// TestProperty2_TokenVerifyRejectsInvalidSignature verifies tampered tokens are rejected.
func TestProperty2_TokenVerifyRejectsInvalidSignature(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sn := "TAMPER-TEST-0001"
	email := "tamper@example.com"
	insertLicense(t, sn, true, time.Now().Add(24*time.Hour))
	insertEmailRecord(t, email, sn)

	authResp := callMarketplaceAuth(t, sn, email)
	token, _ := authResp["token"].(string)
	if token == "" {
		t.Fatal("failed to get token for tamper test")
	}

	parts := strings.SplitN(token, ".", 3)
	tamperedToken := parts[0] + "." + parts[1] + ".INVALID_SIGNATURE"

	resp := callMarketplaceVerify(t, tamperedToken)
	success, _ := resp["success"].(bool)
	if success {
		t.Error("tampered token should be rejected but was accepted")
	}
	code, _ := resp["code"].(string)
	if code != CodeInvalidToken {
		t.Errorf("expected INVALID_TOKEN for tampered token, got %s", code)
	}
}

// TestProperty2_TokenVerifyRejectsExpiredToken verifies expired tokens are rejected.
func TestProperty2_TokenVerifyRejectsExpiredToken(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	secret := os.Getenv("LICENSE_MARKETPLACE_SECRET")
	token, err := signMarketplaceTokenWithExp("EXPIRE-TEST-0001", "expire@example.com", secret, time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	resp := callMarketplaceVerify(t, token)
	success, _ := resp["success"].(bool)
	if success {
		t.Error("expired token should be rejected but was accepted")
	}
	code, _ := resp["code"].(string)
	if code != CodeTokenExpired {
		t.Errorf("expected TOKEN_EXPIRED for expired token, got %s", code)
	}
}
