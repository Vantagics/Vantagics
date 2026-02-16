package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: purchased-pack-display-fix, Property 4: 密码修改往返一致性
// **Validates: Requirements 3.3**
//
// For any user with a set password and any valid new password (length >= 6),
// after changing the password using the correct old password, checkPassword
// with the new password should return true, and checkPassword with the old
// password should return false.
func TestProperty4_PasswordChangeRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Generate a random old password (length 6-20)
		oldPassLen := r.Intn(15) + 6
		oldPassword := randomString(r, oldPassLen)

		// Generate a random new password (length 6-20), ensure it differs from old
		newPassLen := r.Intn(15) + 6
		newPassword := randomString(r, newPassLen)
		for newPassword == oldPassword {
			newPassword = randomString(r, newPassLen)
		}

		// Create a user with the old password
		username := fmt.Sprintf("pwchange_user_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword(oldPassword)
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"email", fmt.Sprintf("EMAIL-%d", seed), username, email, username, hashed, 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// POST to handleUserChangePassword with correct old password and new password
		form := url.Values{}
		form.Set("current_password", oldPassword)
		form.Set("new_password", newPassword)
		form.Set("confirm_password", newPassword)

		req := httptest.NewRequest(http.MethodPost, "/user/change-password", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleUserChangePassword(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("seed=%d: expected 200, got %d", seed, rr.Code)
			return false
		}

		// Verify response contains success message
		body := rr.Body.String()
		if !strings.Contains(body, "密码修改成功") {
			t.Logf("seed=%d: response does not contain success message; body snippet: %s", seed, body[:min(300, len(body))])
			return false
		}

		// Query the updated password_hash from the database
		var updatedHash string
		err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&updatedHash)
		if err != nil {
			t.Logf("seed=%d: failed to query updated password_hash: %v", seed, err)
			return false
		}

		// checkPassword with new password should return true
		if !checkPassword(newPassword, updatedHash) {
			t.Logf("seed=%d: checkPassword(newPassword, updatedHash) returned false", seed)
			return false
		}

		// checkPassword with old password should return false
		if checkPassword(oldPassword, updatedHash) {
			t.Logf("seed=%d: checkPassword(oldPassword, updatedHash) returned true (should be false)", seed)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 4 (密码修改往返一致性) failed: %v", err)
	}
}

// randomString generates a random alphanumeric string of the given length.
func randomString(r *rand.Rand, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// Feature: purchased-pack-display-fix, Property 5: 错误旧密码被拒绝
// **Validates: Requirements 3.4**
//
// For any user and any string different from the current password used as the
// old password, the change-password operation should be rejected and return
// an error message "当前密码错误". The password_hash in the database must
// remain unchanged.
func TestProperty5_WrongOldPasswordRejected(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		cleanup := setupTestDB(t)
		defer cleanup()

		// Generate a random actual password (length 6-20)
		actualPassLen := r.Intn(15) + 6
		actualPassword := randomString(r, actualPassLen)

		// Generate a random wrong password (length 1-20), ensure it differs from actual
		wrongPassLen := r.Intn(20) + 1
		wrongPassword := randomString(r, wrongPassLen)
		for wrongPassword == actualPassword {
			wrongPassword = randomString(r, wrongPassLen)
		}

		// Create a user with the actual password
		username := fmt.Sprintf("wrongpw_user_%d", seed)
		email := fmt.Sprintf("%s@test.com", username)
		hashed := hashPassword(actualPassword)
		res, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"email", fmt.Sprintf("EMAIL-%d", seed), username, email, username, hashed, 0.0,
		)
		if err != nil {
			t.Logf("seed=%d: failed to create user: %v", seed, err)
			return false
		}
		userID, _ := res.LastInsertId()

		// Record the original password_hash
		var originalHash string
		err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&originalHash)
		if err != nil {
			t.Logf("seed=%d: failed to query original password_hash: %v", seed, err)
			return false
		}

		// Generate a random new password for the form (length 6-20)
		newPassLen := r.Intn(15) + 6
		newPassword := randomString(r, newPassLen)

		// POST to handleUserChangePassword with the WRONG old password
		form := url.Values{}
		form.Set("current_password", wrongPassword)
		form.Set("new_password", newPassword)
		form.Set("confirm_password", newPassword)

		req := httptest.NewRequest(http.MethodPost, "/user/change-password", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleUserChangePassword(rr, req)

		// Verify response contains error message "当前密码错误"
		body := rr.Body.String()
		if !strings.Contains(body, "当前密码错误") {
			t.Logf("seed=%d: response does not contain '当前密码错误'; body snippet: %s", seed, body[:min(300, len(body))])
			return false
		}

		// Verify the password_hash in the database has NOT changed
		var currentHash string
		err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
		if err != nil {
			t.Logf("seed=%d: failed to query current password_hash: %v", seed, err)
			return false
		}

		if currentHash != originalHash {
			t.Logf("seed=%d: password_hash changed after wrong old password attempt", seed)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (错误旧密码被拒绝) failed: %v", err)
	}
}


// Feature: purchased-pack-display-fix, Property 6: 密码验证规则
// **Validates: Requirements 3.5, 3.6**
//
// For any new password string shorter than 6 characters, the change-password
// operation should be rejected with "密码至少6个字符".
// For any two different strings as new password and confirm password, the
// change-password operation should be rejected with "两次密码不一致".
func TestProperty6_PasswordValidationRules(t *testing.T) {
	t.Run("ShortPasswordRejected", func(t *testing.T) {
		cfg := &quick.Config{
			MaxCount: 100,
			Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		}

		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))

			cleanup := setupTestDB(t)
			defer cleanup()

			// Generate a random actual password (length 6-20) for the user
			actualPassLen := r.Intn(15) + 6
			actualPassword := randomString(r, actualPassLen)

			// Create a user with the actual password
			username := fmt.Sprintf("shortpw_user_%d", seed)
			email := fmt.Sprintf("%s@test.com", username)
			hashed := hashPassword(actualPassword)
			res, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
				"email", fmt.Sprintf("EMAIL-SHORT-%d", seed), username, email, username, hashed, 0.0,
			)
			if err != nil {
				t.Logf("seed=%d: failed to create user: %v", seed, err)
				return false
			}
			userID, _ := res.LastInsertId()

			// Record the original password_hash
			var originalHash string
			err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&originalHash)
			if err != nil {
				t.Logf("seed=%d: failed to query original password_hash: %v", seed, err)
				return false
			}

			// Generate a short new password (length 1-5)
			shortLen := r.Intn(5) + 1
			shortPassword := randomString(r, shortLen)

			// POST to handleUserChangePassword with correct old password but short new password
			form := url.Values{}
			form.Set("current_password", actualPassword)
			form.Set("new_password", shortPassword)
			form.Set("confirm_password", shortPassword)

			req := httptest.NewRequest(http.MethodPost, "/user/change-password", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleUserChangePassword(rr, req)

			// Verify response contains error message "密码至少6个字符"
			body := rr.Body.String()
			if !strings.Contains(body, "密码至少6个字符") {
				t.Logf("seed=%d: response does not contain '密码至少6个字符'; body snippet: %s", seed, body[:min(300, len(body))])
				return false
			}

			// Verify the password_hash in the database has NOT changed
			var currentHash string
			err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
			if err != nil {
				t.Logf("seed=%d: failed to query current password_hash: %v", seed, err)
				return false
			}

			if currentHash != originalHash {
				t.Logf("seed=%d: password_hash changed after short password attempt", seed)
				return false
			}

			return true
		}

		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 6a (短密码被拒绝) failed: %v", err)
		}
	})

	t.Run("MismatchedPasswordsRejected", func(t *testing.T) {
		cfg := &quick.Config{
			MaxCount: 100,
			Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		}

		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))

			cleanup := setupTestDB(t)
			defer cleanup()

			// Generate a random actual password (length 6-20) for the user
			actualPassLen := r.Intn(15) + 6
			actualPassword := randomString(r, actualPassLen)

			// Create a user with the actual password
			username := fmt.Sprintf("mismatch_user_%d", seed)
			email := fmt.Sprintf("%s@test.com", username)
			hashed := hashPassword(actualPassword)
			res, err := db.Exec(
				"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
				"email", fmt.Sprintf("EMAIL-MISMATCH-%d", seed), username, email, username, hashed, 0.0,
			)
			if err != nil {
				t.Logf("seed=%d: failed to create user: %v", seed, err)
				return false
			}
			userID, _ := res.LastInsertId()

			// Record the original password_hash
			var originalHash string
			err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&originalHash)
			if err != nil {
				t.Logf("seed=%d: failed to query original password_hash: %v", seed, err)
				return false
			}

			// Generate two different valid passwords (length 6-20)
			newPassLen := r.Intn(15) + 6
			newPassword := randomString(r, newPassLen)

			confirmPassLen := r.Intn(15) + 6
			confirmPassword := randomString(r, confirmPassLen)
			for confirmPassword == newPassword {
				confirmPassword = randomString(r, confirmPassLen)
			}

			// POST to handleUserChangePassword with correct old password but mismatched new/confirm
			form := url.Values{}
			form.Set("current_password", actualPassword)
			form.Set("new_password", newPassword)
			form.Set("confirm_password", confirmPassword)

			req := httptest.NewRequest(http.MethodPost, "/user/change-password", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleUserChangePassword(rr, req)

			// Verify response contains error message "两次密码不一致"
			body := rr.Body.String()
			if !strings.Contains(body, "两次密码不一致") {
				t.Logf("seed=%d: response does not contain '两次密码不一致'; body snippet: %s", seed, body[:min(300, len(body))])
				return false
			}

			// Verify the password_hash in the database has NOT changed
			var currentHash string
			err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
			if err != nil {
				t.Logf("seed=%d: failed to query current password_hash: %v", seed, err)
				return false
			}

			if currentHash != originalHash {
				t.Logf("seed=%d: password_hash changed after mismatched password attempt", seed)
				return false
			}

			return true
		}

		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 6b (密码不一致被拒绝) failed: %v", err)
		}
	})
}
