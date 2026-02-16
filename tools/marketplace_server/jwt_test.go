package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateJWT_ReturnsValidToken(t *testing.T) {
	token, err := generateJWT(42, "Alice")
	if err != nil {
		t.Fatalf("generateJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestParseJWT_RoundTrip(t *testing.T) {
	userID := int64(123)
	displayName := "Bob"

	token, err := generateJWT(userID, displayName)
	if err != nil {
		t.Fatalf("generateJWT failed: %v", err)
	}

	gotID, gotName, err := parseJWT(token)
	if err != nil {
		t.Fatalf("parseJWT failed: %v", err)
	}
	if gotID != userID {
		t.Errorf("userID: got %d, want %d", gotID, userID)
	}
	if gotName != displayName {
		t.Errorf("displayName: got %q, want %q", gotName, displayName)
	}
}

func TestParseJWT_InvalidFormat(t *testing.T) {
	_, _, err := parseJWT("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token format")
	}
}

func TestParseJWT_TamperedSignature(t *testing.T) {
	token, _ := generateJWT(1, "Test")
	tampered := token[:len(token)-1] + "X"
	_, _, err := parseJWT(tampered)
	if err == nil {
		t.Fatal("expected error for tampered signature")
	}
}

func TestParseJWT_ExpiredToken(t *testing.T) {
	payload := jwtPayload{
		UserID:      1,
		DisplayName: "Expired",
		Exp:         time.Now().Add(-1 * time.Hour).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64URLEncode(payloadJSON)

	signingInput := jwtHeaderEncoded + "." + payloadEncoded
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	signature := base64URLEncode(mac.Sum(nil))
	token := signingInput + "." + signature

	_, _, err := parseJWT(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if err.Error() != "token_expired" {
		t.Errorf("expected token_expired error, got: %v", err)
	}
}

func TestParseJWT_EmptyString(t *testing.T) {
	_, _, err := parseJWT("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestAuthMiddleware_NoAuthHeader(t *testing.T) {
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/credits/balance", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_InvalidBearerToken(t *testing.T) {
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/credits/balance", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_NoBearerPrefix(t *testing.T) {
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	token, _ := generateJWT(1, "Test")
	req := httptest.NewRequest("GET", "/api/credits/balance", nil)
	req.Header.Set("Authorization", token) // missing "Bearer " prefix
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	var capturedUserID, capturedName string
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Header.Get("X-User-ID")
		capturedName = r.Header.Get("X-Display-Name")
		w.WriteHeader(http.StatusOK)
	})

	token, _ := generateJWT(99, "Charlie")
	req := httptest.NewRequest("GET", "/api/credits/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedUserID != "99" {
		t.Errorf("expected X-User-ID=99, got %q", capturedUserID)
	}
	if capturedName != "Charlie" {
		t.Errorf("expected X-Display-Name=Charlie, got %q", capturedName)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Build an expired token
	payload := jwtPayload{UserID: 1, DisplayName: "Exp", Exp: time.Now().Add(-1 * time.Hour).Unix()}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64URLEncode(payloadJSON)
	signingInput := jwtHeaderEncoded + "." + payloadEncoded
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	signature := base64URLEncode(mac.Sum(nil))
	token := signingInput + "." + signature

	req := httptest.NewRequest("GET", "/api/credits/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// --- optionalUserID tests ---

func TestOptionalUserID_NoHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/packs", nil)
	got := optionalUserID(req)
	if got != 0 {
		t.Errorf("expected 0 for no auth header, got %d", got)
	}
}

func TestOptionalUserID_EmptyHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/packs", nil)
	req.Header.Set("Authorization", "")
	got := optionalUserID(req)
	if got != 0 {
		t.Errorf("expected 0 for empty auth header, got %d", got)
	}
}

func TestOptionalUserID_NoBearerPrefix(t *testing.T) {
	token, _ := generateJWT(42, "Alice")
	req := httptest.NewRequest("GET", "/api/packs", nil)
	req.Header.Set("Authorization", token) // missing "Bearer " prefix
	got := optionalUserID(req)
	if got != 0 {
		t.Errorf("expected 0 for missing Bearer prefix, got %d", got)
	}
}

func TestOptionalUserID_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/packs", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	got := optionalUserID(req)
	if got != 0 {
		t.Errorf("expected 0 for invalid token, got %d", got)
	}
}

func TestOptionalUserID_ExpiredToken(t *testing.T) {
	// Build an expired token
	payload := jwtPayload{UserID: 7, DisplayName: "Expired", Exp: time.Now().Add(-1 * time.Hour).Unix()}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64URLEncode(payloadJSON)
	signingInput := jwtHeaderEncoded + "." + payloadEncoded
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	signature := base64URLEncode(mac.Sum(nil))
	token := signingInput + "." + signature

	req := httptest.NewRequest("GET", "/api/packs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := optionalUserID(req)
	if got != 0 {
		t.Errorf("expected 0 for expired token, got %d", got)
	}
}

func TestOptionalUserID_ValidToken(t *testing.T) {
	token, _ := generateJWT(99, "Charlie")
	req := httptest.NewRequest("GET", "/api/packs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := optionalUserID(req)
	if got != 99 {
		t.Errorf("expected 99 for valid token, got %d", got)
	}
}
