package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsNotificationVisible(t *testing.T) {
	base := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("now before effectiveDate returns false", func(t *testing.T) {
		now := base.AddDate(0, 0, -1) // one day before
		if isNotificationVisible(base, 0, now) {
			t.Error("expected false when now is before effectiveDate")
		}
		if isNotificationVisible(base, 7, now) {
			t.Error("expected false when now is before effectiveDate (with duration)")
		}
	})

	t.Run("durationDays=0 and now >= effectiveDate returns true", func(t *testing.T) {
		// exactly at effectiveDate
		if !isNotificationVisible(base, 0, base) {
			t.Error("expected true when now equals effectiveDate and duration is 0")
		}
		// well after effectiveDate
		future := base.AddDate(1, 0, 0)
		if !isNotificationVisible(base, 0, future) {
			t.Error("expected true when now is after effectiveDate and duration is 0")
		}
	})

	t.Run("durationDays=7 and now within range returns true", func(t *testing.T) {
		now := base.AddDate(0, 0, 3) // 3 days after
		if !isNotificationVisible(base, 7, now) {
			t.Error("expected true when now is within display duration")
		}
	})

	t.Run("durationDays=7 and now past expiry returns false", func(t *testing.T) {
		now := base.AddDate(0, 0, 7) // exactly at expiry boundary
		if isNotificationVisible(base, 7, now) {
			t.Error("expected false when now equals expiryDate (not Before)")
		}
		now = base.AddDate(0, 0, 10) // well past expiry
		if isNotificationVisible(base, 7, now) {
			t.Error("expected false when now is past expiryDate")
		}
	})
}


// setupAdminSession creates an admin user with notifications permission and returns
// a session cookie and the admin ID.
func setupAdminWithNotificationsPerm(t *testing.T) (*http.Cookie, int64) {
	t.Helper()
	result, err := db.Exec(
		"INSERT INTO admin_credentials (username, password_hash, role, permissions) VALUES ('notifadmin', ?, 'admin', 'notifications')",
		hashPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	adminID, _ := result.LastInsertId()
	sessID := createSession(adminID)
	cookie := &http.Cookie{Name: "admin_session", Value: sessID}
	return cookie, adminID
}

func postNotificationJSON(t *testing.T, cookie *http.Cookie, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/notifications", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handleAdminCreateNotification(rr, req)
	return rr
}

func TestCreateNotification_BroadcastSuccess(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	rr := postNotificationJSON(t, cookie, map[string]interface{}{
		"title":                "System Maintenance",
		"content":              "Scheduled downtime this Saturday",
		"target_type":          "broadcast",
		"effective_date":       "2025-01-20T00:00:00Z",
		"display_duration_days": 7,
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["id"] == nil || resp["id"].(float64) <= 0 {
		t.Error("expected positive notification ID")
	}

	// Verify in DB
	notifID := int64(resp["id"].(float64))
	var title, content, targetType, status string
	err := db.QueryRow("SELECT title, content, target_type, status FROM notifications WHERE id = ?", notifID).
		Scan(&title, &content, &targetType, &status)
	if err != nil {
		t.Fatalf("failed to query notification: %v", err)
	}
	if title != "System Maintenance" {
		t.Errorf("expected title='System Maintenance', got %q", title)
	}
	if content != "Scheduled downtime this Saturday" {
		t.Errorf("expected content='Scheduled downtime this Saturday', got %q", content)
	}
	if targetType != "broadcast" {
		t.Errorf("expected target_type='broadcast', got %q", targetType)
	}
	if status != "active" {
		t.Errorf("expected status='active', got %q", status)
	}
}

func TestCreateNotification_TargetedSuccess(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	// Create test users
	r1, _ := db.Exec("INSERT INTO users (auth_type, auth_id, display_name) VALUES ('google', 'u1', 'User1')")
	r2, _ := db.Exec("INSERT INTO users (auth_type, auth_id, display_name) VALUES ('google', 'u2', 'User2')")
	uid1, _ := r1.LastInsertId()
	uid2, _ := r2.LastInsertId()

	rr := postNotificationJSON(t, cookie, map[string]interface{}{
		"title":           "Targeted Message",
		"content":         "This is for you",
		"target_type":     "targeted",
		"target_user_ids": []int64{uid1, uid2},
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	notifID := int64(resp["id"].(float64))

	// Verify notification_targets
	var count int
	db.QueryRow("SELECT COUNT(*) FROM notification_targets WHERE notification_id = ?", notifID).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 notification targets, got %d", count)
	}
}

func TestCreateNotification_EmptyTitle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	rr := postNotificationJSON(t, cookie, map[string]interface{}{
		"title":   "",
		"content": "Some content",
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "title and content are required" {
		t.Errorf("expected 'title and content are required', got %q", resp["error"])
	}
}

func TestCreateNotification_EmptyContent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	rr := postNotificationJSON(t, cookie, map[string]interface{}{
		"title":   "Has Title",
		"content": "",
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateNotification_TargetedWithoutUserIDs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	rr := postNotificationJSON(t, cookie, map[string]interface{}{
		"title":           "Targeted",
		"content":         "Content",
		"target_type":     "targeted",
		"target_user_ids": []int64{},
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "target_user_ids required for targeted messages" {
		t.Errorf("expected 'target_user_ids required for targeted messages', got %q", resp["error"])
	}
}

func TestCreateNotification_DefaultEffectiveDate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	before := time.Now().Add(-time.Second)

	rr := postNotificationJSON(t, cookie, map[string]interface{}{
		"title":       "No Date",
		"content":     "Should default to now",
		"target_type": "broadcast",
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	notifID := int64(resp["id"].(float64))

	var effectiveDateStr string
	db.QueryRow("SELECT effective_date FROM notifications WHERE id = ?", notifID).Scan(&effectiveDateStr)

	effectiveDate, err := time.Parse(time.RFC3339, effectiveDateStr)
	if err != nil {
		t.Fatalf("failed to parse effective_date: %v", err)
	}

	after := time.Now().Add(time.Second)
	if effectiveDate.Before(before) || effectiveDate.After(after) {
		t.Errorf("effective_date %v should be close to now (between %v and %v)", effectiveDate, before, after)
	}
}

func TestCreateNotification_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications", nil)
	rr := httptest.NewRecorder()
	handleAdminCreateNotification(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}


func TestAdminListNotifications_ReturnsNonDeleted(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, adminID := setupAdminWithNotificationsPerm(t)

	// Insert notifications directly: 2 active, 1 disabled, 1 deleted
	db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Active1', 'Content1', 'broadcast', '2025-01-20T00:00:00Z', 7, 'active', ?, '2025-01-20T10:00:00Z', '2025-01-20T10:00:00Z')`, adminID)
	db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Active2', 'Content2', 'broadcast', '2025-01-21T00:00:00Z', 0, 'active', ?, '2025-01-21T10:00:00Z', '2025-01-21T10:00:00Z')`, adminID)
	db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Disabled1', 'Content3', 'broadcast', '2025-01-22T00:00:00Z', 0, 'disabled', ?, '2025-01-22T10:00:00Z', '2025-01-22T10:00:00Z')`, adminID)
	db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Deleted1', 'Content4', 'broadcast', '2025-01-23T00:00:00Z', 0, 'deleted', ?, '2025-01-23T10:00:00Z', '2025-01-23T10:00:00Z')`, adminID)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []AdminNotificationInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should return 3 (active + disabled), not the deleted one
	if len(notifications) != 3 {
		t.Fatalf("expected 3 notifications, got %d", len(notifications))
	}

	// Verify order: created_at DESC → Disabled1, Active2, Active1
	if notifications[0].Title != "Disabled1" {
		t.Errorf("expected first notification title='Disabled1', got %q", notifications[0].Title)
	}
	if notifications[1].Title != "Active2" {
		t.Errorf("expected second notification title='Active2', got %q", notifications[1].Title)
	}
	if notifications[2].Title != "Active1" {
		t.Errorf("expected third notification title='Active1', got %q", notifications[2].Title)
	}

	// Verify deleted notification is excluded
	for _, n := range notifications {
		if n.Title == "Deleted1" {
			t.Error("deleted notification should not appear in list")
		}
	}
}

func TestAdminListNotifications_TargetedIncludesTargetCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, adminID := setupAdminWithNotificationsPerm(t)

	// Create users
	db.Exec("INSERT INTO users (auth_type, auth_id, display_name) VALUES ('google', 'u1', 'User1')")
	db.Exec("INSERT INTO users (auth_type, auth_id, display_name) VALUES ('google', 'u2', 'User2')")
	db.Exec("INSERT INTO users (auth_type, auth_id, display_name) VALUES ('google', 'u3', 'User3')")

	// Insert a targeted notification
	result, _ := db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Targeted', 'For specific users', 'targeted', '2025-01-20T00:00:00Z', 0, 'active', ?, '2025-01-20T10:00:00Z', '2025-01-20T10:00:00Z')`, adminID)
	notifID, _ := result.LastInsertId()

	// Add 3 targets
	db.Exec("INSERT INTO notification_targets (notification_id, user_id) VALUES (?, 1)", notifID)
	db.Exec("INSERT INTO notification_targets (notification_id, user_id) VALUES (?, 2)", notifID)
	db.Exec("INSERT INTO notification_targets (notification_id, user_id) VALUES (?, 3)", notifID)

	// Also insert a broadcast notification
	db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Broadcast', 'For everyone', 'broadcast', '2025-01-21T00:00:00Z', 0, 'active', ?, '2025-01-21T10:00:00Z', '2025-01-21T10:00:00Z')`, adminID)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []AdminNotificationInfo
	json.Unmarshal(rr.Body.Bytes(), &notifications)

	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifications))
	}

	// Find the targeted notification and verify target_count
	for _, n := range notifications {
		if n.TargetType == "targeted" {
			if n.TargetCount != 3 {
				t.Errorf("expected target_count=3 for targeted notification, got %d", n.TargetCount)
			}
		}
		if n.TargetType == "broadcast" {
			if n.TargetCount != 0 {
				t.Errorf("expected target_count=0 for broadcast notification, got %d", n.TargetCount)
			}
		}
	}
}

func TestAdminListNotifications_EmptyList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications", nil)
	rr := httptest.NewRecorder()
	handleAdminListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []AdminNotificationInfo
	json.Unmarshal(rr.Body.Bytes(), &notifications)

	if len(notifications) != 0 {
		t.Errorf("expected empty list, got %d notifications", len(notifications))
	}
}

func TestAdminListNotifications_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/admin/notifications", nil)
	rr := httptest.NewRecorder()
	handleAdminListNotifications(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}


func createTestNotification(t *testing.T, adminID int64, status string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Test Notification', 'Test Content', 'broadcast', '2025-01-20T00:00:00Z', 7, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, status, adminID)
	if err != nil {
		t.Fatalf("failed to create test notification: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

func TestDisableNotification_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, adminID := setupAdminWithNotificationsPerm(t)
	notifID := createTestNotification(t, adminID, "active")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/notifications/%d/disable", notifID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminDisableNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status='ok', got %q", resp["status"])
	}

	// Verify status in DB
	var status string
	db.QueryRow("SELECT status FROM notifications WHERE id = ?", notifID).Scan(&status)
	if status != "disabled" {
		t.Errorf("expected status='disabled', got %q", status)
	}
}

func TestDisableNotification_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/notifications/99999/disable", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminDisableNotification(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "notification not found" {
		t.Errorf("expected error='notification not found', got %q", resp["error"])
	}
}

func TestDisableNotification_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications/1/disable", nil)
	rr := httptest.NewRecorder()
	handleAdminDisableNotification(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestDisableNotification_InvalidID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/admin/notifications/abc/disable", nil)
	rr := httptest.NewRecorder()
	handleAdminDisableNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestDisableNotification_UpdatesTimestamp(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, adminID := setupAdminWithNotificationsPerm(t)

	// Insert with a known old timestamp
	result, err := db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES ('Test', 'Content', 'broadcast', '2025-01-20T00:00:00Z', 7, 'active', ?, '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`, adminID)
	if err != nil {
		t.Fatalf("failed to create notification: %v", err)
	}
	notifID, _ := result.LastInsertId()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/notifications/%d/disable", notifID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminDisableNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var newUpdatedAt string
	db.QueryRow("SELECT updated_at FROM notifications WHERE id = ?", notifID).Scan(&newUpdatedAt)

	if newUpdatedAt == "2025-01-01T00:00:00Z" || newUpdatedAt == "2025-01-01 00:00:00" {
		t.Error("expected updated_at to change after disable operation")
	}
}

func TestEnableNotification_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, adminID := setupAdminWithNotificationsPerm(t)
	notifID := createTestNotification(t, adminID, "disabled")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/notifications/%d/enable", notifID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminEnableNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status='ok', got %q", resp["status"])
	}

	// Verify status in DB
	var status string
	db.QueryRow("SELECT status FROM notifications WHERE id = ?", notifID).Scan(&status)
	if status != "active" {
		t.Errorf("expected status='active', got %q", status)
	}
}

func TestEnableNotification_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/notifications/99999/enable", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminEnableNotification(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "notification not found" {
		t.Errorf("expected error='notification not found', got %q", resp["error"])
	}
}

func TestEnableNotification_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications/1/enable", nil)
	rr := httptest.NewRecorder()
	handleAdminEnableNotification(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestDeleteNotification_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, adminID := setupAdminWithNotificationsPerm(t)
	notifID := createTestNotification(t, adminID, "active")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/notifications/%d/delete", notifID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminDeleteNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status='ok', got %q", resp["status"])
	}

	// Verify status in DB
	var status string
	db.QueryRow("SELECT status FROM notifications WHERE id = ?", notifID).Scan(&status)
	if status != "deleted" {
		t.Errorf("expected status='deleted', got %q", status)
	}
}

func TestDeleteNotification_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/notifications/99999/delete", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleAdminDeleteNotification(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "notification not found" {
		t.Errorf("expected error='notification not found', got %q", resp["error"])
	}
}

func TestDeleteNotification_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications/1/delete", nil)
	rr := httptest.NewRecorder()
	handleAdminDeleteNotification(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}


// --- handleListNotifications tests ---

// insertNotification is a helper that inserts a notification directly into the DB.
func insertNotification(t *testing.T, adminID int64, title, content, targetType, effectiveDate string, durationDays int, status string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		title, content, targetType, effectiveDate, durationDays, status, adminID)
	if err != nil {
		t.Fatalf("failed to insert notification: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

func addNotificationTarget(t *testing.T, notificationID, userID int64) {
	t.Helper()
	_, err := db.Exec("INSERT INTO notification_targets (notification_id, user_id) VALUES (?, ?)", notificationID, userID)
	if err != nil {
		t.Fatalf("failed to insert notification target: %v", err)
	}
}

func createTestUserWithName(t *testing.T, authID, displayName string) int64 {
	t.Helper()
	result, err := db.Exec("INSERT INTO users (auth_type, auth_id, display_name) VALUES ('google', ?, ?)", authID, displayName)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

func TestListNotifications_UnauthenticatedSeesBroadcastOnly(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, adminID := setupAdminWithNotificationsPerm(t)
	userID := createTestUserWithName(t, "u1", "User1")

	now := time.Now()
	pastDate := now.Add(-24 * time.Hour).Format(time.RFC3339)

	// Active broadcast
	insertNotification(t, adminID, "Broadcast1", "Content1", "broadcast", pastDate, 0, "active")
	// Active targeted for userID
	nTargeted := insertNotification(t, adminID, "Targeted1", "Content2", "targeted", pastDate, 0, "active")
	addNotificationTarget(t, nTargeted, userID)

	// Unauthenticated request (no Authorization header)
	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	rr := httptest.NewRecorder()
	handleListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []NotificationInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification (broadcast only), got %d", len(notifications))
	}
	if notifications[0].Title != "Broadcast1" {
		t.Errorf("expected title='Broadcast1', got %q", notifications[0].Title)
	}
}

func TestListNotifications_AuthenticatedSeesBroadcastAndTargeted(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, adminID := setupAdminWithNotificationsPerm(t)
	userID := createTestUserWithName(t, "u1", "User1")
	otherUserID := createTestUserWithName(t, "u2", "User2")

	now := time.Now()
	pastDate := now.Add(-24 * time.Hour).Format(time.RFC3339)

	// Active broadcast
	insertNotification(t, adminID, "Broadcast1", "Content1", "broadcast", pastDate, 0, "active")
	// Active targeted for userID
	nTargeted := insertNotification(t, adminID, "ForUser1", "Content2", "targeted", pastDate, 0, "active")
	addNotificationTarget(t, nTargeted, userID)
	// Active targeted for otherUserID (should NOT appear)
	nOther := insertNotification(t, adminID, "ForUser2", "Content3", "targeted", pastDate, 0, "active")
	addNotificationTarget(t, nOther, otherUserID)

	// Generate JWT for userID
	token, err := generateJWT(userID, "User1")
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handleListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []NotificationInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications (broadcast + targeted for user), got %d", len(notifications))
	}

	titles := map[string]bool{}
	for _, n := range notifications {
		titles[n.Title] = true
	}
	if !titles["Broadcast1"] {
		t.Error("expected Broadcast1 in results")
	}
	if !titles["ForUser1"] {
		t.Error("expected ForUser1 in results")
	}
	if titles["ForUser2"] {
		t.Error("ForUser2 should NOT appear for this user")
	}
}

func TestListNotifications_ExpiredMessagesExcluded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, adminID := setupAdminWithNotificationsPerm(t)

	now := time.Now()
	// Notification that expired 2 days ago (effective 10 days ago, duration 7 days)
	expiredDate := now.Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	insertNotification(t, adminID, "Expired", "Old content", "broadcast", expiredDate, 7, "active")

	// Notification still active (effective 2 days ago, duration 7 days)
	recentDate := now.Add(-2 * 24 * time.Hour).Format(time.RFC3339)
	insertNotification(t, adminID, "StillActive", "Fresh content", "broadcast", recentDate, 7, "active")

	// Permanent notification (duration 0)
	insertNotification(t, adminID, "Permanent", "Always visible", "broadcast", expiredDate, 0, "active")

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	rr := httptest.NewRecorder()
	handleListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []NotificationInfo
	json.Unmarshal(rr.Body.Bytes(), &notifications)

	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications (StillActive + Permanent), got %d", len(notifications))
	}

	titles := map[string]bool{}
	for _, n := range notifications {
		titles[n.Title] = true
	}
	if titles["Expired"] {
		t.Error("expired notification should not appear")
	}
	if !titles["StillActive"] {
		t.Error("expected StillActive in results")
	}
	if !titles["Permanent"] {
		t.Error("expected Permanent in results")
	}
}

func TestListNotifications_FutureMessagesExcluded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, adminID := setupAdminWithNotificationsPerm(t)

	now := time.Now()
	futureDate := now.Add(48 * time.Hour).Format(time.RFC3339)
	pastDate := now.Add(-24 * time.Hour).Format(time.RFC3339)

	// Future notification (not yet effective)
	insertNotification(t, adminID, "Future", "Not yet", "broadcast", futureDate, 0, "active")
	// Past notification (already effective)
	insertNotification(t, adminID, "Current", "Active now", "broadcast", pastDate, 0, "active")

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	rr := httptest.NewRecorder()
	handleListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []NotificationInfo
	json.Unmarshal(rr.Body.Bytes(), &notifications)

	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification (Current only), got %d", len(notifications))
	}
	if notifications[0].Title != "Current" {
		t.Errorf("expected title='Current', got %q", notifications[0].Title)
	}
}

func TestListNotifications_DisabledMessagesExcluded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, adminID := setupAdminWithNotificationsPerm(t)

	now := time.Now()
	pastDate := now.Add(-24 * time.Hour).Format(time.RFC3339)

	// Disabled notification
	insertNotification(t, adminID, "Disabled", "Should not show", "broadcast", pastDate, 0, "disabled")
	// Deleted notification
	insertNotification(t, adminID, "Deleted", "Should not show", "broadcast", pastDate, 0, "deleted")
	// Active notification
	insertNotification(t, adminID, "Active", "Should show", "broadcast", pastDate, 0, "active")

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	rr := httptest.NewRecorder()
	handleListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []NotificationInfo
	json.Unmarshal(rr.Body.Bytes(), &notifications)

	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification (Active only), got %d", len(notifications))
	}
	if notifications[0].Title != "Active" {
		t.Errorf("expected title='Active', got %q", notifications[0].Title)
	}
}

func TestListNotifications_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/notifications", nil)
	rr := httptest.NewRecorder()
	handleListNotifications(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestListNotifications_EmptyResult(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	rr := httptest.NewRecorder()
	handleListNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var notifications []NotificationInfo
	json.Unmarshal(rr.Body.Bytes(), &notifications)

	if len(notifications) != 0 {
		t.Errorf("expected empty list, got %d notifications", len(notifications))
	}
}
