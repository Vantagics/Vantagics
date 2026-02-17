package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"
)

// Feature: marketplace-notification-center, Property 1: 消息可见性计算正确性
// **Validates: Requirements 1.3, 1.4, 7.1, 7.2, 7.3**
//
// For any notification's effective_date, display_duration_days, and current time now:
// - If now < effective_date, the notification is not visible
// - If display_duration_days = 0 and now >= effective_date, the notification is permanently visible
// - If display_duration_days > 0 and now >= effective_date and now < effective_date + display_duration_days, visible
// - If display_duration_days > 0 and now >= effective_date + display_duration_days, not visible
func TestProperty_NotificationVisibility(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Sub-property 1: now < effective_date => not visible
	t.Run("before_effective_date_not_visible", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))

			// Generate a random effective_date
			baseUnix := int64(946684800) // 2000-01-01
			effectiveUnix := baseUnix + rng.Int63n(30*365*24*3600) // up to ~30 years
			effectiveDate := time.Unix(effectiveUnix, 0).UTC()

			// now is strictly before effective_date (1 second to 365 days before)
			offsetSec := rng.Int63n(365*24*3600) + 1
			now := effectiveDate.Add(-time.Duration(offsetSec) * time.Second)

			// duration can be anything (0 = permanent, or positive)
			durationDays := rng.Intn(366) // 0..365

			result := isNotificationVisible(effectiveDate, durationDays, now)
			if result {
				t.Logf("FAIL: now=%v < effectiveDate=%v, durationDays=%d, but got visible=true",
					now, effectiveDate, durationDays)
			}
			return !result
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property violated: %v", err)
		}
	})

	// Sub-property 2: durationDays=0 and now >= effective_date => permanently visible
	t.Run("permanent_duration_always_visible", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))

			baseUnix := int64(946684800)
			effectiveUnix := baseUnix + rng.Int63n(30*365*24*3600)
			effectiveDate := time.Unix(effectiveUnix, 0).UTC()

			// now is at or after effective_date (0 to 10 years after)
			offsetSec := rng.Int63n(10 * 365 * 24 * 3600)
			now := effectiveDate.Add(time.Duration(offsetSec) * time.Second)

			durationDays := 0

			result := isNotificationVisible(effectiveDate, durationDays, now)
			if !result {
				t.Logf("FAIL: now=%v >= effectiveDate=%v, durationDays=0, but got visible=false",
					now, effectiveDate)
			}
			return result
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property violated: %v", err)
		}
	})

	// Sub-property 3: durationDays>0, now in [effective_date, effective_date+duration) => visible
	t.Run("within_duration_visible", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))

			baseUnix := int64(946684800)
			effectiveUnix := baseUnix + rng.Int63n(30*365*24*3600)
			effectiveDate := time.Unix(effectiveUnix, 0).UTC()

			// durationDays between 1 and 365
			durationDays := rng.Intn(365) + 1

			expiryDate := effectiveDate.AddDate(0, 0, durationDays)
			totalDuration := expiryDate.Sub(effectiveDate)

			// now is within [effectiveDate, expiryDate) — random fraction of the range
			if totalDuration <= 0 {
				return true // degenerate case, skip
			}
			offsetNano := rng.Int63n(int64(totalDuration))
			now := effectiveDate.Add(time.Duration(offsetNano))

			result := isNotificationVisible(effectiveDate, durationDays, now)
			if !result {
				t.Logf("FAIL: effectiveDate=%v, durationDays=%d, now=%v (within range), but got visible=false",
					effectiveDate, durationDays, now)
			}
			return result
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property violated: %v", err)
		}
	})

	// Sub-property 4: durationDays>0, now >= effective_date+duration => not visible
	t.Run("past_expiry_not_visible", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))

			baseUnix := int64(946684800)
			effectiveUnix := baseUnix + rng.Int63n(30*365*24*3600)
			effectiveDate := time.Unix(effectiveUnix, 0).UTC()

			// durationDays between 1 and 365
			durationDays := rng.Intn(365) + 1

			expiryDate := effectiveDate.AddDate(0, 0, durationDays)

			// now is at or after expiryDate (0 to 10 years after)
			offsetSec := rng.Int63n(10 * 365 * 24 * 3600)
			now := expiryDate.Add(time.Duration(offsetSec) * time.Second)

			result := isNotificationVisible(effectiveDate, durationDays, now)
			if result {
				t.Logf("FAIL: effectiveDate=%v, durationDays=%d, expiryDate=%v, now=%v (past expiry), but got visible=true",
					effectiveDate, durationDays, expiryDate, now)
			}
			return !result
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property violated: %v", err)
		}
	})
}

// Feature: marketplace-notification-center, Property 5: 消息创建验证
// **Validates: Requirements 2.4, 2.5**
//
// For any message creation request, if title is empty or content is empty,
// creation should be rejected with an error; if both title and content are
// non-empty, creation should succeed.
func TestProperty_MessageCreationValidation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	// generateString returns a random string that may be empty, whitespace-only,
	// or a non-empty value. We bias toward empty/whitespace to exercise the
	// validation boundary.
	generateString := func(rng *rand.Rand) string {
		kind := rng.Intn(5)
		switch {
		case kind == 0:
			return "" // empty string
		case kind == 1:
			// whitespace-only (spaces/tabs)
			n := rng.Intn(5) + 1
			ws := make([]byte, n)
			for i := range ws {
				if rng.Intn(2) == 0 {
					ws[i] = ' '
				} else {
					ws[i] = '\t'
				}
			}
			return string(ws)
		default:
			// non-empty string of 1..50 printable chars
			length := rng.Intn(50) + 1
			buf := make([]byte, length)
			for i := range buf {
				buf[i] = byte(rng.Intn(94) + 33) // printable ASCII 33..126
			}
			return string(buf)
		}
	}

	isEffectivelyEmpty := func(s string) bool {
		for _, c := range s {
			if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
				return false
			}
		}
		return true
	}

	const iterations = 200
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < iterations; i++ {
		title := generateString(rng)
		content := generateString(rng)

		rr := postNotificationJSON(t, cookie, map[string]interface{}{
			"title":                 title,
			"content":               content,
			"target_type":           "broadcast",
			"effective_date":        "2025-06-01T00:00:00Z",
			"display_duration_days": 0,
		})

		titleEmpty := isEffectivelyEmpty(title)
		contentEmpty := isEffectivelyEmpty(content)

		if titleEmpty || contentEmpty {
			// Should be rejected
			if rr.Code != 400 {
				t.Fatalf("iteration %d: expected 400 for title=%q content=%q, got %d",
					i, title, content, rr.Code)
			}
		} else {
			// Should succeed
			if rr.Code != 201 {
				t.Fatalf("iteration %d: expected 201 for title=%q content=%q, got %d (body: %s)",
					i, title, content, rr.Code, rr.Body.String())
			}
		}
	}
}

// Feature: marketplace-notification-center, Property 2: 消息状态转换往返一致性
// **Validates: Requirements 3.2, 3.4**
//
// For any active notification, executing disable changes status to disabled,
// then executing enable restores status to active, and message content remains unchanged.
func TestProperty_StatusTransitionRoundtrip(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie, _ := setupAdminWithNotificationsPerm(t)

	// generatePrintableString returns a non-empty random printable ASCII string of length 1..80.
	generatePrintableString := func(rng *rand.Rand) string {
		length := rng.Intn(80) + 1
		buf := make([]byte, length)
		for i := range buf {
			buf[i] = byte(rng.Intn(94) + 33) // printable ASCII 33..126
		}
		return string(buf)
	}

	const iterations = 100
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < iterations; i++ {
		title := generatePrintableString(rng)
		content := generatePrintableString(rng)

		// Step 1: Create a notification
		rr := postNotificationJSON(t, cookie, map[string]interface{}{
			"title":                 title,
			"content":               content,
			"target_type":           "broadcast",
			"effective_date":        "2025-06-01T00:00:00Z",
			"display_duration_days": 0,
		})
		if rr.Code != 201 {
			t.Fatalf("iteration %d: create failed with %d: %s", i, rr.Code, rr.Body.String())
		}

		var createResp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
			t.Fatalf("iteration %d: failed to parse create response: %v", i, err)
		}
		notifID := int64(createResp["id"].(float64))

		// Step 2: Verify initial status is "active"
		var status string
		if err := db.QueryRow("SELECT status FROM notifications WHERE id = ?", notifID).Scan(&status); err != nil {
			t.Fatalf("iteration %d: failed to query status: %v", i, err)
		}
		if status != "active" {
			t.Fatalf("iteration %d: expected initial status 'active', got %q", i, status)
		}

		// Step 3: Disable the notification
		disableReq := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/admin/notifications/%d/disable", notifID), nil)
		disableReq.AddCookie(cookie)
		disableRR := httptest.NewRecorder()
		handleAdminDisableNotification(disableRR, disableReq)
		if disableRR.Code != http.StatusOK {
			t.Fatalf("iteration %d: disable failed with %d: %s", i, disableRR.Code, disableRR.Body.String())
		}

		// Step 4: Verify status is now "disabled"
		if err := db.QueryRow("SELECT status FROM notifications WHERE id = ?", notifID).Scan(&status); err != nil {
			t.Fatalf("iteration %d: failed to query status after disable: %v", i, err)
		}
		if status != "disabled" {
			t.Fatalf("iteration %d: expected status 'disabled' after disable, got %q", i, status)
		}

		// Step 5: Enable the notification
		enableReq := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/admin/notifications/%d/enable", notifID), nil)
		enableReq.AddCookie(cookie)
		enableRR := httptest.NewRecorder()
		handleAdminEnableNotification(enableRR, enableReq)
		if enableRR.Code != http.StatusOK {
			t.Fatalf("iteration %d: enable failed with %d: %s", i, enableRR.Code, enableRR.Body.String())
		}

		// Step 6: Verify status is restored to "active"
		if err := db.QueryRow("SELECT status FROM notifications WHERE id = ?", notifID).Scan(&status); err != nil {
			t.Fatalf("iteration %d: failed to query status after enable: %v", i, err)
		}
		if status != "active" {
			t.Fatalf("iteration %d: expected status 'active' after enable roundtrip, got %q", i, status)
		}

		// Step 7: Verify title and content remain unchanged
		var dbTitle, dbContent string
		if err := db.QueryRow("SELECT title, content FROM notifications WHERE id = ?", notifID).Scan(&dbTitle, &dbContent); err != nil {
			t.Fatalf("iteration %d: failed to query title/content: %v", i, err)
		}
		if dbTitle != title {
			t.Fatalf("iteration %d: title changed after roundtrip: expected %q, got %q", i, title, dbTitle)
		}
		if dbContent != content {
			t.Fatalf("iteration %d: content changed after roundtrip: expected %q, got %q", i, content, dbContent)
		}
	}
}

// Feature: marketplace-notification-center, Property 6: 权限控制强制执行
// **Validates: Requirements 8.3**
//
// For any admin without the "notifications" permission, calling any notification
// management API (create, list, disable, enable, delete) should return 403 status code.
func TestProperty_PermissionControlEnforcement(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// The wrapped handler that includes the permissionAuth("notifications") middleware,
	// exactly as registered in main().
	wrappedHandler := permissionAuth("notifications")(handleAdminNotificationRoutes)

	// Create a placeholder super admin (ID=1) so that test admins get ID > 1.
	// Admin ID=1 bypasses all permission checks, so we must avoid it.
	_, err := db.Exec(
		"INSERT INTO admin_credentials (username, password_hash, role, permissions) VALUES ('superadmin', ?, 'admin', 'all')",
		hashPassword("superpass"),
	)
	if err != nil {
		t.Fatalf("failed to create placeholder super admin: %v", err)
	}

	// Available non-notifications permissions to randomly assign.
	nonNotifPerms := []string{"packs", "review", "categories", "customers", "billing"}

	const iterations = 100
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < iterations; i++ {
		// Generate a random permission set that does NOT include "notifications".
		numPerms := rng.Intn(len(nonNotifPerms)) + 1
		perm := rand.Perm(len(nonNotifPerms))
		selectedPerms := make([]string, numPerms)
		for j := 0; j < numPerms; j++ {
			selectedPerms[j] = nonNotifPerms[perm[j]]
		}
		permStr := ""
		for j, p := range selectedPerms {
			if j > 0 {
				permStr += ","
			}
			permStr += p
		}

		// Generate a random username to avoid collisions.
		username := fmt.Sprintf("noperm_admin_%d_%d", i, rng.Int63())

		// Create admin with non-notifications permissions (ID will be > 1).
		result, err := db.Exec(
			"INSERT INTO admin_credentials (username, password_hash, role, permissions) VALUES (?, ?, 'admin', ?)",
			username, hashPassword("testpass"), permStr,
		)
		if err != nil {
			t.Fatalf("iteration %d: failed to create admin: %v", i, err)
		}
		adminID, _ := result.LastInsertId()
		if adminID == 1 {
			t.Fatalf("iteration %d: admin ID must not be 1 (super admin bypasses permission checks)", i)
		}

		sessID := createSession(adminID)
		cookie := &http.Cookie{Name: "admin_session", Value: sessID}

		// Define all 5 admin notification endpoints to test.
		endpoints := []struct {
			method string
			path   string
		}{
			{http.MethodPost, "/api/admin/notifications"},                // create
			{http.MethodGet, "/api/admin/notifications"},                 // list
			{http.MethodPost, "/api/admin/notifications/1/disable"},      // disable
			{http.MethodPost, "/api/admin/notifications/1/enable"},       // enable
			{http.MethodPost, "/api/admin/notifications/1/delete"},       // delete
		}

		for _, ep := range endpoints {
			var req *http.Request
			if ep.method == http.MethodPost && ep.path == "/api/admin/notifications" {
				// Create endpoint needs a JSON body
				body := []byte(`{"title":"test","content":"test","target_type":"broadcast"}`)
				req = httptest.NewRequest(ep.method, ep.path, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			wrappedHandler(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("iteration %d: %s %s with perms=%q: expected 403, got %d (body: %s)",
					i, ep.method, ep.path, permStr, rr.Code, rr.Body.String())
			}
		}
	}
}

// Feature: marketplace-notification-center, Property 3: 已认证用户消息查询正确性
// **Validates: Requirements 4.2, 4.3**
//
// For any authenticated user and any message set (containing broadcast messages and
// targeted messages for different users), the query result should only contain:
// (a) all visible broadcast messages, and (b) visible targeted messages for this user.
// It should NOT contain targeted messages for other users.
func TestProperty_AuthenticatedUserQuery(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, adminID := setupAdminWithNotificationsPerm(t)

	const iterations = 100
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < iterations; i++ {
		// Clean up notifications and targets from previous iteration
		db.Exec("DELETE FROM notification_targets")
		db.Exec("DELETE FROM notifications")

		// Create the authenticated test user
		authUserID := createTestUserWithName(t,
			fmt.Sprintf("auth_%d_%d", i, rng.Int63()),
			fmt.Sprintf("AuthUser_%d", i),
		)

		// Create 1..3 other users
		numOtherUsers := rng.Intn(3) + 1
		otherUserIDs := make([]int64, numOtherUsers)
		for j := 0; j < numOtherUsers; j++ {
			otherUserIDs[j] = createTestUserWithName(t,
				fmt.Sprintf("other_%d_%d_%d", i, j, rng.Int63()),
				fmt.Sprintf("OtherUser_%d_%d", i, j),
			)
		}

		// Use a past effective_date so all notifications are currently visible
		now := time.Now()
		pastDate := now.Add(-24 * time.Hour).Format(time.RFC3339)

		// Generate a random mix of notifications:
		// - broadcast messages
		// - targeted messages for the auth user
		// - targeted messages for other users
		numBroadcast := rng.Intn(4) + 1       // 1..4
		numTargetedSelf := rng.Intn(4)         // 0..3
		numTargetedOther := rng.Intn(4) + 1    // 1..4 (at least 1 to test exclusion)

		expectedTitles := make(map[string]bool)

		// Create broadcast notifications
		for b := 0; b < numBroadcast; b++ {
			title := fmt.Sprintf("Broadcast_%d_%d", i, b)
			insertNotification(t, adminID, title, "broadcast content", "broadcast", pastDate, 0, "active")
			expectedTitles[title] = true
		}

		// Create targeted notifications for the authenticated user
		for s := 0; s < numTargetedSelf; s++ {
			title := fmt.Sprintf("TargetSelf_%d_%d", i, s)
			nID := insertNotification(t, adminID, title, "targeted self content", "targeted", pastDate, 0, "active")
			addNotificationTarget(t, nID, authUserID)
			expectedTitles[title] = true
		}

		// Create targeted notifications for other users (should NOT appear)
		excludedTitles := make(map[string]bool)
		for o := 0; o < numTargetedOther; o++ {
			title := fmt.Sprintf("TargetOther_%d_%d", i, o)
			nID := insertNotification(t, adminID, title, "targeted other content", "targeted", pastDate, 0, "active")
			targetUser := otherUserIDs[rng.Intn(len(otherUserIDs))]
			addNotificationTarget(t, nID, targetUser)
			excludedTitles[title] = true
		}

		// Generate JWT for the authenticated user
		token, err := generateJWT(authUserID, fmt.Sprintf("AuthUser_%d", i))
		if err != nil {
			t.Fatalf("iteration %d: failed to generate JWT: %v", i, err)
		}

		// Query as authenticated user
		req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		handleListNotifications(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("iteration %d: expected 200, got %d; body: %s", i, rr.Code, rr.Body.String())
		}

		var notifications []NotificationInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &notifications); err != nil {
			t.Fatalf("iteration %d: failed to unmarshal response: %v", i, err)
		}

		// Collect returned titles
		returnedTitles := make(map[string]bool)
		for _, n := range notifications {
			returnedTitles[n.Title] = true
		}

		// Verify all expected titles are present
		for title := range expectedTitles {
			if !returnedTitles[title] {
				t.Fatalf("iteration %d: expected title %q in results but not found (broadcast=%d, targetedSelf=%d, targetedOther=%d, returned=%d)",
					i, title, numBroadcast, numTargetedSelf, numTargetedOther, len(notifications))
			}
		}

		// Verify no excluded titles are present
		for title := range excludedTitles {
			if returnedTitles[title] {
				t.Fatalf("iteration %d: title %q targeted to other user should NOT appear in results",
					i, title)
			}
		}

		// Verify count matches expected
		expectedCount := numBroadcast + numTargetedSelf
		if len(notifications) != expectedCount {
			t.Fatalf("iteration %d: expected %d notifications, got %d (broadcast=%d, targetedSelf=%d, targetedOther=%d)",
				i, expectedCount, len(notifications), numBroadcast, numTargetedSelf, numTargetedOther)
		}
	}
}

// Feature: marketplace-notification-center, Property 4: 未认证用户仅可见广播消息
// **Validates: Requirements 4.5**
//
// For any message set (containing broadcast and targeted messages), an unauthenticated
// user's query result should only contain visible broadcast messages and should NOT
// contain any targeted messages.
func TestProperty_UnauthenticatedUserBroadcastOnly(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, adminID := setupAdminWithNotificationsPerm(t)

	const iterations = 100
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < iterations; i++ {
		// Clean up from previous iteration
		db.Exec("DELETE FROM notification_targets")
		db.Exec("DELETE FROM notifications")

		// Create some users to be targets of targeted messages
		numUsers := rng.Intn(3) + 1
		userIDs := make([]int64, numUsers)
		for j := 0; j < numUsers; j++ {
			userIDs[j] = createTestUserWithName(t,
				fmt.Sprintf("unauth_user_%d_%d_%d", i, j, rng.Int63()),
				fmt.Sprintf("User_%d_%d", i, j),
			)
		}

		// Use a past effective_date so all notifications are currently visible
		now := time.Now()
		pastDate := now.Add(-24 * time.Hour).Format(time.RFC3339)

		// Generate a random mix of broadcast and targeted notifications
		numBroadcast := rng.Intn(4) + 1    // 1..4
		numTargeted := rng.Intn(4) + 1     // 1..4 (at least 1 to test exclusion)

		broadcastTitles := make(map[string]bool)

		// Create broadcast notifications
		for b := 0; b < numBroadcast; b++ {
			title := fmt.Sprintf("Broadcast_%d_%d", i, b)
			insertNotification(t, adminID, title, "broadcast content", "broadcast", pastDate, 0, "active")
			broadcastTitles[title] = true
		}

		// Create targeted notifications (should NOT appear for unauthenticated users)
		targetedTitles := make(map[string]bool)
		for tg := 0; tg < numTargeted; tg++ {
			title := fmt.Sprintf("Targeted_%d_%d", i, tg)
			nID := insertNotification(t, adminID, title, "targeted content", "targeted", pastDate, 0, "active")
			targetUser := userIDs[rng.Intn(len(userIDs))]
			addNotificationTarget(t, nID, targetUser)
			targetedTitles[title] = true
		}

		// Query WITHOUT Authorization header (unauthenticated)
		req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
		rr := httptest.NewRecorder()
		handleListNotifications(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("iteration %d: expected 200, got %d; body: %s", i, rr.Code, rr.Body.String())
		}

		var notifications []NotificationInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &notifications); err != nil {
			t.Fatalf("iteration %d: failed to unmarshal response: %v", i, err)
		}

		// Collect returned titles
		returnedTitles := make(map[string]bool)
		for _, n := range notifications {
			returnedTitles[n.Title] = true
		}

		// Verify all broadcast titles are present
		for title := range broadcastTitles {
			if !returnedTitles[title] {
				t.Fatalf("iteration %d: expected broadcast title %q in results but not found", i, title)
			}
		}

		// Verify NO targeted titles are present
		for title := range targetedTitles {
			if returnedTitles[title] {
				t.Fatalf("iteration %d: targeted title %q should NOT appear for unauthenticated user", i, title)
			}
		}

		// Verify count matches only broadcast notifications
		if len(notifications) != numBroadcast {
			t.Fatalf("iteration %d: expected %d broadcast notifications, got %d (targeted=%d)",
				i, numBroadcast, len(notifications), numTargeted)
		}
	}
}
