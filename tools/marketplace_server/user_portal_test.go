package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
)

func createTestImage(w, h int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

func testColor() color.RGBA {
	return color.RGBA{50, 50, 120, 255}
}

// TestDrawCharSupportsSpecialCharacters verifies that drawChar can render
// digits 0-9 and special characters +, -, =, ?, space without panicking.
func TestDrawCharSupportsSpecialCharacters(t *testing.T) {
	img := createTestImage(100, 50)
	chars := []byte{'0', '1', '9', '+', '-', '=', '?', ' '}
	for _, ch := range chars {
		drawChar(img, ch, 10, 5, testColor())
	}
}

// TestDrawCharUnknownCharDoesNotPanic verifies unknown chars are silently ignored.
func TestDrawCharUnknownCharDoesNotPanic(t *testing.T) {
	img := createTestImage(100, 50)
	drawChar(img, '@', 10, 5, testColor())
	drawChar(img, '#', 10, 5, testColor())
}

// TestGenerateMathCaptchaImageReturnsValidPNG verifies the function produces
// a valid PNG for a real math captcha.
func TestGenerateMathCaptchaImageReturnsValidPNG(t *testing.T) {
	id := createMathCaptcha()
	data := generateMathCaptchaImage(id)
	if data == nil {
		t.Fatal("expected non-nil image data")
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty image data")
	}
	// Verify it's a valid PNG
	_, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected valid PNG, got decode error: %v", err)
	}
}

// TestGenerateMathCaptchaImageInvalidID returns nil for unknown captcha ID.
func TestGenerateMathCaptchaImageInvalidID(t *testing.T) {
	data := generateMathCaptchaImage("nonexistent-id")
	if data != nil {
		t.Fatal("expected nil for invalid captcha ID")
	}
}

// Feature: marketplace-user-portal, Property 1: 数学验证码生成约束
// Validates: Requirements 1.2, 3.1
// For any generated math captcha, both operands should be in [1,20],
// operator is + or -, and the result is a non-negative integer.
func TestPropertyMathCaptchaConstraints(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(_ byte) bool {
		expr, ansStr := generateMathCaptcha()

		// Parse expression: expected format "A + B = ?" or "A - B = ?"
		parts := strings.Fields(expr) // ["A", "+/-", "B", "=", "?"]
		if len(parts) != 5 {
			t.Logf("unexpected expression format: %q", expr)
			return false
		}

		a, errA := strconv.Atoi(parts[0])
		op := parts[1]
		b, errB := strconv.Atoi(parts[2])
		if errA != nil || errB != nil {
			t.Logf("failed to parse operands: %q", expr)
			return false
		}

		// Operands must be in [1, 20]
		if a < 1 || a > 20 || b < 1 || b > 20 {
			t.Logf("operand out of range: a=%d b=%d", a, b)
			return false
		}

		// Operator must be + or -
		if op != "+" && op != "-" {
			t.Logf("unexpected operator: %q", op)
			return false
		}

		// Answer must be a non-negative integer
		ans, errAns := strconv.Atoi(ansStr)
		if errAns != nil {
			t.Logf("answer is not a valid integer: %q", ansStr)
			return false
		}
		if ans < 0 {
			t.Logf("negative result: %d", ans)
			return false
		}

		// Verify answer matches the expression
		var expected int
		if op == "+" {
			expected = a + b
		} else {
			expected = a - b
		}
		if ans != expected {
			t.Logf("answer mismatch: %s => got %d, want %d", expr, ans, expected)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Errorf("Property 1 failed: %v", err)
	}
}


// Feature: marketplace-user-portal, Property 3: 验证码一次性使用
// Validates: Requirements 3.3
// For any captcha, after the first verification attempt (whether correct or not),
// a second verification using the same captcha ID should return false.
func TestPropertyCaptchaOneTimeUse(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(_ byte) bool {
		id := createMathCaptcha()
		answer := getCaptchaCode(id)
		if answer == "" {
			t.Log("getCaptchaCode returned empty for fresh captcha")
			return false
		}

		// First verification with correct answer should succeed
		first := verifyCaptcha(id, answer)
		if !first {
			t.Logf("first verifyCaptcha with correct answer should succeed, id=%s", id)
			return false
		}

		// Second verification with the same ID should fail (captcha consumed)
		second := verifyCaptcha(id, answer)
		if second {
			t.Logf("second verifyCaptcha should fail after one-time use, id=%s", id)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Errorf("Property 3 failed: %v", err)
	}
}


// Feature: marketplace-user-portal, Property 10: 密码哈希往返一致性
// Validates: Requirements 7.5
// For any password string, hashPassword followed by checkPassword with the same
// password should return true; checkPassword with a different password should return false.
func TestPropertyPasswordHashRoundTrip(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(password string, other string) bool {
		if len(password) == 0 || len(password) > 72 {
			return true // skip empty or too-long passwords (bcrypt limit is 72 bytes)
		}
		if len(other) > 72 {
			return true
		}

		hashed := hashPassword(password)

		// Same password must verify successfully
		if !checkPassword(password, hashed) {
			t.Logf("checkPassword should return true for same password %q", password)
			return false
		}

		// Different password must fail verification
		if password != other && checkPassword(other, hashed) {
			t.Logf("checkPassword should return false for different password: original=%q other=%q", password, other)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Errorf("Property 10 failed: %v", err)
	}
}

// Feature: marketplace-user-portal, Property 8: 用户会话与管理后台会话隔离
// **Validates: Requirements 6.2**
// For any user ID and admin ID, creating a user session should not create or
// affect admin sessions, and creating an admin session should not create or
// affect user sessions.
func TestPropertySessionIsolation(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(userID uint32, adminID uint32) bool {
		// Use non-zero IDs
		uid := int64(userID%10000) + 1
		aid := int64(adminID%10000) + 1

		// Create a user session
		userSessID := createUserSession(uid)

		// User session must be valid in user session store
		if !isValidUserSession(userSessID) {
			t.Logf("user session %s should be valid in user store", userSessID)
			return false
		}

		// User session must NOT be valid in admin session store
		if isValidSession(userSessID) {
			t.Logf("user session %s should NOT be valid in admin store", userSessID)
			return false
		}

		// User session should return correct user ID
		if getUserSessionUserID(userSessID) != uid {
			t.Logf("getUserSessionUserID returned wrong ID for user session")
			return false
		}

		// Create an admin session
		adminSessID := createSession(aid)

		// Admin session must be valid in admin session store
		if !isValidSession(adminSessID) {
			t.Logf("admin session %s should be valid in admin store", adminSessID)
			return false
		}

		// Admin session must NOT be valid in user session store
		if isValidUserSession(adminSessID) {
			t.Logf("admin session %s should NOT be valid in user store", adminSessID)
			return false
		}

		// Creating admin session should not have invalidated the user session
		if !isValidUserSession(userSessID) {
			t.Logf("user session %s should still be valid after creating admin session", userSessID)
			return false
		}

		// Creating user session should not have invalidated the admin session
		if !isValidSession(adminSessID) {
			t.Logf("admin session %s should still be valid after user session was created", adminSessID)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Errorf("Property 8 failed: %v", err)
	}
}


// Feature: marketplace-user-portal, Property 9: 未认证用户重定向
// **Validates: Requirements 6.3**
// For any /user/ protected path, accessing without a valid user_session
// should return a redirect (302) to /user/login.
func TestPropertyUnauthenticatedRedirect(t *testing.T) {
	// A dummy handler that should never be reached for unauthenticated requests.
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dashboard"))
	})
	protected := userAuth(dummyHandler)

	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(suffix uint16) bool {
		// Generate random path suffixes under /user/
		path := "/user/dashboard/" + strconv.Itoa(int(suffix))
		req := httptest.NewRequest(http.MethodGet, path, nil)
		// No user_session cookie set — request is unauthenticated
		rr := httptest.NewRecorder()

		protected.ServeHTTP(rr, req)

		// Must be a 302 redirect
		if rr.Code != http.StatusFound {
			t.Logf("expected 302 for path %s without session, got %d", path, rr.Code)
			return false
		}

		// Redirect location must be /user/login
		loc := rr.Header().Get("Location")
		if loc != "/user/login" {
			t.Logf("expected redirect to /user/login, got %q for path %s", loc, path)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Errorf("Property 9 failed: %v", err)
	}
}

// Feature: marketplace-user-portal, Property 2: 登录凭据验证正确性
// Validates: Requirements 1.3, 1.4, 1.5
// For any registered user with correct captcha, login with correct credentials
// should succeed (302 redirect), login with wrong password or wrong captcha should fail.
func TestPropertyLoginCredentialVerification(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed uint32) bool {
		// Generate random username and password from seed
		username := fmt.Sprintf("testuser_%d", seed%100000)
		password := fmt.Sprintf("pass_%d_secure", seed)
		wrongPassword := password + "_wrong"

		// Clean up any existing user with this username
		db.Exec("DELETE FROM users WHERE username = ?", username)
		db.Exec("DELETE FROM email_wallets WHERE username = ?", username)

		// Insert user into database
		hashed := hashPassword(password)
		_, insertErr := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-TEST-%d", seed), username, fmt.Sprintf("%s@test.com", username), username, hashed, 0,
		)
		if insertErr != nil {
			t.Logf("failed to insert user: %v", insertErr)
			return false
		}

		// Also set password in email_wallets (email-level, required for login)
		testEmail := fmt.Sprintf("%s@test.com", username)
		ensureWalletExists(testEmail)
		db.Exec("UPDATE email_wallets SET password_hash = ?, username = ? WHERE email = ?", hashed, username, testEmail)

		// --- Test 1: Correct credentials + correct captcha → 302 redirect ---
		captchaID1 := createMathCaptcha()
		captchaAnswer1 := getCaptchaCode(captchaID1)
		if captchaAnswer1 == "" {
			t.Log("getCaptchaCode returned empty for fresh captcha")
			return false
		}

		form1 := fmt.Sprintf("username=%s&password=%s&captcha_id=%s&captcha_answer=%s",
			username, password, captchaID1, captchaAnswer1)
		req1 := httptest.NewRequest(http.MethodPost, "/user/login", strings.NewReader(form1))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr1 := httptest.NewRecorder()
		handleUserLogin(rr1, req1)

		if rr1.Code != http.StatusFound {
			t.Logf("correct credentials: expected 302, got %d (body: %s)", rr1.Code, rr1.Body.String()[:min(200, rr1.Body.Len())])
			return false
		}
		loc1 := rr1.Header().Get("Location")
		if loc1 != "/user/dashboard" {
			t.Logf("correct credentials: expected redirect to /user/dashboard, got %q", loc1)
			return false
		}
		// Verify user_session cookie is set
		cookies1 := rr1.Result().Cookies()
		hasSession := false
		for _, c := range cookies1 {
			if c.Name == "user_session" && c.Value != "" {
				hasSession = true
				break
			}
		}
		if !hasSession {
			t.Log("correct credentials: expected user_session cookie to be set")
			return false
		}

		// --- Test 2: Wrong password + correct captcha → login fails ---
		captchaID2 := createMathCaptcha()
		captchaAnswer2 := getCaptchaCode(captchaID2)

		form2 := fmt.Sprintf("username=%s&password=%s&captcha_id=%s&captcha_answer=%s",
			username, wrongPassword, captchaID2, captchaAnswer2)
		req2 := httptest.NewRequest(http.MethodPost, "/user/login", strings.NewReader(form2))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr2 := httptest.NewRecorder()
		handleUserLogin(rr2, req2)

		if rr2.Code == http.StatusFound {
			t.Log("wrong password: should NOT get 302 redirect")
			return false
		}
		if !strings.Contains(rr2.Body.String(), "用户名或密码错误") {
			t.Log("wrong password: expected error message '用户名或密码错误'")
			return false
		}

		// --- Test 3: Correct credentials + wrong captcha → login fails ---
		captchaID3 := createMathCaptcha()
		// Use a deliberately wrong captcha answer
		wrongCaptchaAnswer := "99999"

		form3 := fmt.Sprintf("username=%s&password=%s&captcha_id=%s&captcha_answer=%s",
			username, password, captchaID3, wrongCaptchaAnswer)
		req3 := httptest.NewRequest(http.MethodPost, "/user/login", strings.NewReader(form3))
		req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr3 := httptest.NewRecorder()
		handleUserLogin(rr3, req3)

		if rr3.Code == http.StatusFound {
			t.Log("wrong captcha: should NOT get 302 redirect")
			return false
		}
		if !strings.Contains(rr3.Body.String(), "验证码错误") {
			t.Log("wrong captcha: expected error message '验证码错误'")
			return false
		}

		// Clean up user for next iteration
		db.Exec("DELETE FROM users WHERE username = ?", username)
		db.Exec("DELETE FROM email_wallets WHERE username = ?", username)

		return true
	}, cfg)
	if err != nil {
		t.Errorf("Property 2 failed: %v", err)
	}
}

// Feature: marketplace-user-portal, Property 4: 绑定注册创建正确的用户记录
// Validates: Requirements 2.3, 7.3
// For any valid email and SN combination, a user created via binding registration
// should have: username == email prefix (before @), password_hash non-empty,
// auth_type == "sn", auth_id == SN, email == full email address.
func TestPropertyBindingRegistrationCreatesCorrectUserRecord(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Start mock License_Server that always returns success for /api/marketplace-auth
	mockLS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/marketplace-auth" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success": true, "token": "mock-token"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockLS.Close()

	// Point license_server_url setting to mock server
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('license_server_url', ?)", mockLS.URL)
	if err != nil {
		t.Fatalf("failed to set license_server_url: %v", err)
	}

	cfg := &quick.Config{MaxCount: 100}
	iteration := 0
	qErr := quick.Check(func(seed uint32) bool {
		iteration++
		// Generate random email and SN from seed
		localPart := fmt.Sprintf("user%d", seed%100000)
		domains := []string{"example.com", "test.org", "mail.net", "company.co"}
		domain := domains[seed%uint32(len(domains))]
		email := fmt.Sprintf("%s@%s", localPart, domain)
		sn := fmt.Sprintf("SN-REG-%d-%d", seed, iteration)
		password := fmt.Sprintf("pass_%d_secure", seed)

		// Clean up any existing user with this SN or username to avoid conflicts
		db.Exec("DELETE FROM users WHERE auth_id = ? OR username = ?", sn, localPart)

		// Create captcha and get the answer
		captchaID := createMathCaptcha()
		captchaAnswer := getCaptchaCode(captchaID)
		if captchaAnswer == "" {
			t.Logf("iteration %d: getCaptchaCode returned empty", iteration)
			return false
		}

		// Submit registration form via httptest
		formData := fmt.Sprintf("email=%s&sn=%s&password=%s&password2=%s&captcha_id=%s&captcha_answer=%s",
			email, sn, password, password, captchaID, captchaAnswer)
		req := httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handleUserRegister(rr, req)

		// Should redirect on success (302)
		if rr.Code != http.StatusFound {
			t.Logf("iteration %d: expected 302, got %d (body snippet: %s)", iteration, rr.Code, rr.Body.String()[:min(200, rr.Body.Len())])
			return false
		}

		// Verify the created user record in the database
		var dbAuthType, dbAuthID, dbEmail string
		err := db.QueryRow(
			"SELECT auth_type, auth_id, email FROM users WHERE auth_id = ? AND auth_type = 'sn'",
			sn,
		).Scan(&dbAuthType, &dbAuthID, &dbEmail)
		if err != nil {
			t.Logf("iteration %d: failed to query user record: %v", iteration, err)
			return false
		}

		// Property checks:
		// 1. auth_type must be "sn"
		if dbAuthType != "sn" {
			t.Logf("iteration %d: auth_type=%q, expected 'sn'", iteration, dbAuthType)
			return false
		}

		// 4. auth_id must be the SN value
		if dbAuthID != sn {
			t.Logf("iteration %d: auth_id=%q, expected=%q", iteration, dbAuthID, sn)
			return false
		}

		// 3. email must be the full email address
		if dbEmail != email {
			t.Logf("iteration %d: email=%q, expected=%q", iteration, dbEmail, email)
			return false
		}

		// 4. password_hash must be set in email_wallets (email-level)
		var walletPwHash, walletUsername sql.NullString
		err = db.QueryRow("SELECT password_hash, username FROM email_wallets WHERE email = ?", email).Scan(&walletPwHash, &walletUsername)
		if err != nil {
			t.Logf("iteration %d: failed to query email_wallets: %v", iteration, err)
			return false
		}
		if !walletPwHash.Valid || walletPwHash.String == "" {
			t.Logf("iteration %d: email_wallets password_hash is empty", iteration)
			return false
		}
		// 5. username in email_wallets == email prefix
		if !walletUsername.Valid || walletUsername.String != localPart {
			t.Logf("iteration %d: email_wallets username=%q, expected=%q", iteration, walletUsername.String, localPart)
			return false
		}

		return true
	}, cfg)
	if qErr != nil {
		t.Errorf("Property 4 failed: %v", qErr)
	}
}

// Feature: marketplace-user-portal, Property 5: 重复 SN 绑定拒绝
// **Validates: Requirements 2.7**
func TestPropertyDuplicateSNBindingRejection(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Start mock License_Server that always returns success for /api/marketplace-auth
	mockLS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/marketplace-auth" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success": true, "token": "mock-token"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockLS.Close()

	// Point license_server_url setting to mock server
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('license_server_url', ?)", mockLS.URL)
	if err != nil {
		t.Fatalf("failed to set license_server_url: %v", err)
	}

	cfg := &quick.Config{MaxCount: 100}
	iteration := 0
	qErr := quick.Check(func(seed uint32) bool {
		iteration++
		// Generate a unique SN for this iteration
		sn := fmt.Sprintf("SN-DUP-%d-%d", seed, iteration)

		// First registration: create a user with this SN
		email1 := fmt.Sprintf("first%d_%d@example.com", seed, iteration)
		localPart1 := fmt.Sprintf("first%d_%d", seed, iteration)
		password1 := fmt.Sprintf("pass1_%d_secure", seed)

		// Clean up any existing user with this SN or usernames to avoid conflicts
		db.Exec("DELETE FROM users WHERE auth_id = ? OR username = ? OR username = ?", sn, localPart1, fmt.Sprintf("second%d_%d", seed, iteration))

		// First registration should succeed
		captchaID1 := createMathCaptcha()
		captchaAnswer1 := getCaptchaCode(captchaID1)
		if captchaAnswer1 == "" {
			t.Logf("iteration %d: getCaptchaCode returned empty for first registration", iteration)
			return false
		}

		formData1 := fmt.Sprintf("email=%s&sn=%s&password=%s&password2=%s&captcha_id=%s&captcha_answer=%s",
			email1, sn, password1, password1, captchaID1, captchaAnswer1)
		req1 := httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(formData1))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr1 := httptest.NewRecorder()
		handleUserRegister(rr1, req1)

		if rr1.Code != http.StatusFound {
			t.Logf("iteration %d: first registration expected 302, got %d (body: %s)", iteration, rr1.Code, rr1.Body.String()[:min(200, rr1.Body.Len())])
			return false
		}

		// Second registration with SAME SN but different email should be rejected
		email2 := fmt.Sprintf("second%d_%d@example.com", seed, iteration)
		password2 := fmt.Sprintf("pass2_%d_secure", seed)

		captchaID2 := createMathCaptcha()
		captchaAnswer2 := getCaptchaCode(captchaID2)
		if captchaAnswer2 == "" {
			t.Logf("iteration %d: getCaptchaCode returned empty for second registration", iteration)
			return false
		}

		formData2 := fmt.Sprintf("email=%s&sn=%s&password=%s&password2=%s&captcha_id=%s&captcha_answer=%s",
			email2, sn, password2, password2, captchaID2, captchaAnswer2)
		req2 := httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(formData2))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr2 := httptest.NewRecorder()
		handleUserRegister(rr2, req2)

		// Second registration should NOT redirect (should show error page)
		if rr2.Code == http.StatusFound {
			t.Logf("iteration %d: second registration should not succeed (got 302 redirect)", iteration)
			return false
		}

		// Response body should contain the duplicate SN error message
		body := rr2.Body.String()
		if !strings.Contains(body, "该序列号已绑定账号") {
			t.Logf("iteration %d: expected error '该序列号已绑定账号' in body, got: %s", iteration, body[:min(300, len(body))])
			return false
		}

		// Verify no new user was created: count of users with this SN should still be 1
		var userCount int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE auth_type='sn' AND auth_id=?", sn).Scan(&userCount)
		if err != nil {
			t.Logf("iteration %d: failed to count users: %v", iteration, err)
			return false
		}
		if userCount != 1 {
			t.Logf("iteration %d: expected 1 user with SN %q, got %d", iteration, sn, userCount)
			return false
		}

		return true
	}, cfg)
	if qErr != nil {
		t.Errorf("Property 5 failed: %v", qErr)
	}
}

// Feature: marketplace-user-portal, Property 6: 注册表单密码验证
// **Validates: Requirements 2.5, 2.6**
// Sub-property A: For any two different strings as password and password2,
// registration should be rejected with "两次密码不一致".
// Sub-property B: For any string with length < 6 as password (with matching password2),
// registration should be rejected with "密码至少6个字符".
func TestPropertyRegistrationPasswordValidation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Start mock License_Server (should never be reached for password validation failures)
	mockLS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/marketplace-auth" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success": true, "token": "mock-token"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockLS.Close()

	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('license_server_url', ?)", mockLS.URL)
	if err != nil {
		t.Fatalf("failed to set license_server_url: %v", err)
	}

	// Sub-property A: password != password2 → "两次密码不一致"
	t.Run("PasswordMismatch", func(t *testing.T) {
		cfg := &quick.Config{MaxCount: 100}
		qErr := quick.Check(func(seed uint32) bool {
			// Generate two different passwords from seed
			pw1 := fmt.Sprintf("pass_%d_alpha", seed)
			pw2 := fmt.Sprintf("pass_%d_beta", seed)
			// Ensure they are actually different
			if pw1 == pw2 {
				pw2 = pw2 + "_x"
			}

			captchaID := createMathCaptcha()
			captchaAnswer := getCaptchaCode(captchaID)
			if captchaAnswer == "" {
				t.Log("getCaptchaCode returned empty for fresh captcha")
				return false
			}

			formData := fmt.Sprintf("email=test%d@example.com&sn=SN-PWV-%d&password=%s&password2=%s&captcha_id=%s&captcha_answer=%s",
				seed, seed, pw1, pw2, captchaID, captchaAnswer)
			req := httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(formData))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			handleUserRegister(rr, req)

			// Should NOT redirect (password mismatch error)
			if rr.Code == http.StatusFound {
				t.Logf("password mismatch should not succeed (got 302), pw1=%q pw2=%q", pw1, pw2)
				return false
			}

			body := rr.Body.String()
			if !strings.Contains(body, "两次密码不一致") {
				t.Logf("expected '两次密码不一致' in body, got: %s", body[:min(300, len(body))])
				return false
			}

			return true
		}, cfg)
		if qErr != nil {
			t.Errorf("Property 6 sub-property A (password mismatch) failed: %v", qErr)
		}
	})

	// Sub-property B: len(password) < 6 with matching password2 → "密码至少6个字符"
	t.Run("PasswordTooShort", func(t *testing.T) {
		cfg := &quick.Config{MaxCount: 100}
		qErr := quick.Check(func(seed uint32) bool {
			// Generate a short password (1-5 characters)
			length := int(seed%5) + 1 // 1 to 5
			shortPw := ""
			for i := 0; i < length; i++ {
				shortPw += string(rune('a' + (int(seed)+i)%26))
			}

			captchaID := createMathCaptcha()
			captchaAnswer := getCaptchaCode(captchaID)
			if captchaAnswer == "" {
				t.Log("getCaptchaCode returned empty for fresh captcha")
				return false
			}

			formData := fmt.Sprintf("email=short%d@example.com&sn=SN-SHORT-%d&password=%s&password2=%s&captcha_id=%s&captcha_answer=%s",
				seed, seed, shortPw, shortPw, captchaID, captchaAnswer)
			req := httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(formData))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			handleUserRegister(rr, req)

			// Should NOT redirect (password too short error)
			if rr.Code == http.StatusFound {
				t.Logf("short password should not succeed (got 302), pw=%q len=%d", shortPw, len(shortPw))
				return false
			}

			body := rr.Body.String()
			if !strings.Contains(body, "密码至少6个字符") {
				t.Logf("expected '密码至少6个字符' in body for pw=%q (len=%d), got: %s", shortPw, len(shortPw), body[:min(300, len(body))])
				return false
			}

			return true
		}, cfg)
		if qErr != nil {
			t.Errorf("Property 6 sub-property B (password too short) failed: %v", qErr)
		}
	})
}

// Feature: marketplace-user-portal, Property 7: 个人中心分析包显示正确性
// **Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5, 4.6**
// For any user and their purchased pack set, the dashboard page should display all
// purchased packs, and each pack should show the correct billing type label based on
// its share_mode (free→"免费", per_use→"按次付费", time_limited→"限时", subscription→"订阅").
func TestPropertyDashboardPackDisplay(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	shareModes := []string{"free", "per_use", "time_limited", "subscription"}
	expectedLabels := map[string]string{
		"free":         "免费",
		"per_use":      "按次付费",
		"time_limited": "限时",
		"subscription": "订阅",
	}

	// Create a category for packs
	_, err := db.Exec("INSERT INTO categories (name, description) VALUES ('TestCat', 'test category')")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}
	var categoryID int64
	db.QueryRow("SELECT id FROM categories WHERE name = 'TestCat'").Scan(&categoryID)

	cfg := &quick.Config{MaxCount: 50}
	qErr := quick.Check(func(seed uint32) bool {
		// Clean up from previous iteration
		db.Exec("DELETE FROM user_downloads")
		db.Exec("DELETE FROM credits_transactions WHERE transaction_type = 'purchase'")
		db.Exec("DELETE FROM pack_listings WHERE pack_name LIKE 'DashPack_%'")
		db.Exec("DELETE FROM users WHERE username LIKE 'dashuser_%'")

		// Create a test user
		username := fmt.Sprintf("dashuser_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword("password123")
		res, insertErr := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"sn", fmt.Sprintf("SN-DASH-%d", seed), username, email, username, hashed, 100.0,
		)
		if insertErr != nil {
			t.Logf("failed to insert user: %v", insertErr)
			return false
		}
		userID, _ := res.LastInsertId()

		// Also set password in email_wallets (email-level)
		ensureWalletExists(email)
		db.Exec("UPDATE email_wallets SET password_hash = ?, username = ? WHERE email = ?", hashed, username, email)

		// Determine which share_modes to include based on seed bits
		// Use at least 1 mode, up to all 4
		includedModes := []string{}
		for i, mode := range shareModes {
			if seed&(1<<uint(i)) != 0 {
				includedModes = append(includedModes, mode)
			}
		}
		if len(includedModes) == 0 {
			// Ensure at least one mode is included
			includedModes = append(includedModes, shareModes[seed%4])
		}

		// Create pack_listings and corresponding purchase/download records
		createdPacks := make(map[string]string) // packName -> shareMode
		for idx, mode := range includedModes {
			packName := fmt.Sprintf("DashPack_%d_%d", seed, idx)
			price := 0
			if mode != "free" {
				price = 10
			}
			validDays := 0
			if mode == "time_limited" || mode == "subscription" {
				validDays = 30
			}

			listRes, lErr := db.Exec(
				"INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, valid_days) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', ?)",
				userID, categoryID, []byte("fake-zip"), packName, "desc", "source", "author", mode, price, validDays,
			)
			if lErr != nil {
				t.Logf("failed to insert pack_listing: %v", lErr)
				return false
			}
			listingID, _ := listRes.LastInsertId()

			if mode == "free" {
				// Free packs use user_downloads
				_, dErr := db.Exec(
					"INSERT INTO user_downloads (user_id, listing_id, downloaded_at) VALUES (?, ?, datetime('now'))",
					userID, listingID,
				)
				if dErr != nil {
					t.Logf("failed to insert user_download: %v", dErr)
					return false
				}
			} else {
				// Paid packs use credits_transactions
				_, tErr := db.Exec(
					"INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, created_at) VALUES (?, 'purchase', ?, ?, ?, datetime('now'))",
					userID, -float64(price), listingID, fmt.Sprintf("购买 %s", packName),
				)
				if tErr != nil {
					t.Logf("failed to insert credits_transaction: %v", tErr)
					return false
				}
			}

			createdPacks[packName] = mode
		}

		// Create a valid user session
		sessionID := createUserSession(userID)

		// Call handleUserDashboard via httptest
		req := httptest.NewRequest(http.MethodGet, "/user/dashboard", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		req.AddCookie(&http.Cookie{Name: "user_session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleUserDashboard(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String()[:min(300, rr.Body.Len())])
			return false
		}

		body := rr.Body.String()

		// Verify all packs appear in the response
		for packName, mode := range createdPacks {
			if !strings.Contains(body, packName) {
				t.Logf("pack %q (mode=%s) not found in dashboard HTML", packName, mode)
				return false
			}
			// Verify the correct label appears
			label := expectedLabels[mode]
			if !strings.Contains(body, label) {
				t.Logf("expected label %q for mode %q not found in dashboard HTML", label, mode)
				return false
			}
		}

		return true
	}, cfg)
	if qErr != nil {
		t.Errorf("Property 7 failed: %v", qErr)
	}
}

// Feature: marketplace-user-portal, Property 11: SN 自动登录保持空密码字段
// Validates: Requirements 7.4
// For any user created via SN auto-login (handleSNLogin), the username and
// password_hash fields should remain NULL (empty).
func TestPropertySNAutoLoginKeepsEmptyPassword(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	domains := []string{"example.com", "test.org", "mail.net", "company.co"}

	iteration := 0
	qErr := quick.Check(func(seed uint32) bool {
		iteration++

		cleanup := setupTestDB(t)
		defer cleanup()

		sn := fmt.Sprintf("SN-AUTO-%d-%d", seed, iteration)
		email := fmt.Sprintf("auto%d_%d@%s", seed, iteration, domains[seed%uint32(len(domains))])

		// Start mock License_Server that responds to /api/marketplace-verify
		mockLS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/marketplace-verify" {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"success": true, "sn": %q, "email": %q}`, sn, email)
				return
			}
			http.NotFound(w, r)
		}))
		defer mockLS.Close()

		oldURL := licenseServerURL
		licenseServerURL = mockLS.URL
		defer func() { licenseServerURL = oldURL }()

		// Call handleSNLogin to create a user via SN auto-login
		body := fmt.Sprintf(`{"license_token": "mock-token-%d"}`, seed)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/sn-login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handleSNLogin(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("iteration %d: handleSNLogin returned %d, expected 200", iteration, rr.Code)
			return false
		}

		// Query the created user record and check username and password_hash are NULL
		var usernameNull, passwordHashNull sql.NullString
		err := db.QueryRow(
			"SELECT username, password_hash FROM users WHERE auth_type = 'sn' AND auth_id = ?", sn,
		).Scan(&usernameNull, &passwordHashNull)
		if err != nil {
			t.Logf("iteration %d: failed to query user: %v", iteration, err)
			return false
		}

		// username should be NULL (not valid)
		if usernameNull.Valid && usernameNull.String != "" {
			t.Logf("iteration %d: username should be NULL/empty, got %q", iteration, usernameNull.String)
			return false
		}

		// password_hash should be NULL (not valid)
		if passwordHashNull.Valid && passwordHashNull.String != "" {
			t.Logf("iteration %d: password_hash should be NULL/empty, got %q", iteration, passwordHashNull.String)
			return false
		}

		return true
	}, cfg)
	if qErr != nil {
		t.Errorf("Property 11 failed: %v", qErr)
	}
}
