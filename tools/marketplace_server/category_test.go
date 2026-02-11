package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Helper to POST JSON to admin categories endpoint
func postCategoryJSON(t *testing.T, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleAdminCategories(rr, req)
	return rr
}

// Helper to PUT JSON to admin categories endpoint with ID
func putCategoryJSON(t *testing.T, id int64, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/categories/%d", id), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleAdminCategories(rr, req)
	return rr
}

// Helper to DELETE admin categories endpoint with ID
func deleteCategoryReq(t *testing.T, id int64) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/categories/%d", id), nil)
	rr := httptest.NewRecorder()
	handleAdminCategories(rr, req)
	return rr
}

// Helper to GET categories list
func listCategories(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	rr := httptest.NewRecorder()
	handleListCategories(rr, req)
	return rr
}

func TestListCategories_ReturnsPresets(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := listCategories(t)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var categories []PackCategory
	json.Unmarshal(rr.Body.Bytes(), &categories)

	if len(categories) != 4 {
		t.Fatalf("expected 4 preset categories, got %d", len(categories))
	}

	expectedNames := map[string]bool{"Shopify": true, "BigCommerce": true, "eBay": true, "Etsy": true}
	for _, cat := range categories {
		if !expectedNames[cat.Name] {
			t.Errorf("unexpected category: %s", cat.Name)
		}
		if !cat.IsPreset {
			t.Errorf("expected is_preset=true for %s", cat.Name)
		}
		if cat.PackCount != 0 {
			t.Errorf("expected pack_count=0 for %s, got %d", cat.Name, cat.PackCount)
		}
	}
}

func TestListCategories_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/categories", nil)
	rr := httptest.NewRecorder()
	handleListCategories(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestCreateCategory_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postCategoryJSON(t, map[string]string{
		"name":        "Custom Analytics",
		"description": "Custom analytics packs",
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var cat PackCategory
	json.Unmarshal(rr.Body.Bytes(), &cat)

	if cat.Name != "Custom Analytics" {
		t.Errorf("expected name='Custom Analytics', got %q", cat.Name)
	}
	if cat.Description != "Custom analytics packs" {
		t.Errorf("expected description='Custom analytics packs', got %q", cat.Description)
	}
	if cat.IsPreset {
		t.Error("expected is_preset=false for user-created category")
	}
	if cat.ID <= 0 {
		t.Error("expected positive ID")
	}
}

func TestCreateCategory_EmptyName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postCategoryJSON(t, map[string]string{
		"name":        "",
		"description": "No name",
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateCategory_DuplicateName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// First creation should succeed
	rr1 := postCategoryJSON(t, map[string]string{"name": "UniqueCategory"})
	if rr1.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", rr1.Code)
	}

	// Second creation with same name should fail
	rr2 := postCategoryJSON(t, map[string]string{"name": "UniqueCategory"})
	if rr2.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate name, got %d", rr2.Code)
	}
}

func TestUpdateCategory_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a category first
	rr := postCategoryJSON(t, map[string]string{"name": "Original", "description": "Original desc"})
	var created PackCategory
	json.Unmarshal(rr.Body.Bytes(), &created)

	// Update it
	rr2 := putCategoryJSON(t, created.ID, map[string]string{
		"name":        "Updated",
		"description": "Updated desc",
	})

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr2.Code, rr2.Body.String())
	}

	var updated PackCategory
	json.Unmarshal(rr2.Body.Bytes(), &updated)

	if updated.Name != "Updated" {
		t.Errorf("expected name='Updated', got %q", updated.Name)
	}
	if updated.Description != "Updated desc" {
		t.Errorf("expected description='Updated desc', got %q", updated.Description)
	}
	if updated.ID != created.ID {
		t.Errorf("expected same ID %d, got %d", created.ID, updated.ID)
	}
}

func TestUpdateCategory_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := putCategoryJSON(t, 99999, map[string]string{"name": "Ghost"})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateCategory_EmptyName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := postCategoryJSON(t, map[string]string{"name": "ToUpdate"})
	var created PackCategory
	json.Unmarshal(rr.Body.Bytes(), &created)

	rr2 := putCategoryJSON(t, created.ID, map[string]string{"name": ""})
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr2.Code)
	}
}

func TestDeleteCategory_Success(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a category
	rr := postCategoryJSON(t, map[string]string{"name": "ToDelete"})
	var created PackCategory
	json.Unmarshal(rr.Body.Bytes(), &created)

	// Delete it
	rr2 := deleteCategoryReq(t, created.ID)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr2.Code, rr2.Body.String())
	}

	// Verify it's gone from the list
	rr3 := listCategories(t)
	var categories []PackCategory
	json.Unmarshal(rr3.Body.Bytes(), &categories)
	for _, cat := range categories {
		if cat.ID == created.ID {
			t.Error("deleted category still appears in list")
		}
	}
}

func TestDeleteCategory_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	rr := deleteCategoryReq(t, 99999)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteCategory_HasListings(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a category
	rr := postCategoryJSON(t, map[string]string{"name": "WithListings"})
	var created PackCategory
	json.Unmarshal(rr.Body.Bytes(), &created)

	// Create a user and a pack_listing referencing this category
	result, err := db.Exec(
		"INSERT INTO users (oauth_provider, oauth_provider_id, display_name, email) VALUES ('google', 'test-user', 'Test', 'test@test.com')",
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	userID, _ := result.LastInsertId()

	_, err = db.Exec(
		"INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, share_mode) VALUES (?, ?, X'00', 'TestPack', 'free')",
		userID, created.ID,
	)
	if err != nil {
		t.Fatalf("failed to create test listing: %v", err)
	}

	// Try to delete - should be refused
	rr2 := deleteCategoryReq(t, created.ID)
	if rr2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d; body: %s", rr2.Code, rr2.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &resp)
	if resp["error"] != "category_has_listings" {
		t.Errorf("expected error='category_has_listings', got %v", resp["error"])
	}
	if resp["count"].(float64) != 1 {
		t.Errorf("expected count=1, got %v", resp["count"])
	}
}

func TestListCategories_PackCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a category
	rr := postCategoryJSON(t, map[string]string{"name": "CountTest"})
	var created PackCategory
	json.Unmarshal(rr.Body.Bytes(), &created)

	// Create a user
	result, _ := db.Exec(
		"INSERT INTO users (oauth_provider, oauth_provider_id, display_name) VALUES ('google', 'count-user', 'Counter')",
	)
	userID, _ := result.LastInsertId()

	// Add 2 published listings to this category
	for i := 0; i < 2; i++ {
		db.Exec(
			"INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, share_mode, status) VALUES (?, ?, X'00', ?, 'free', 'published')",
			userID, created.ID, fmt.Sprintf("Pack%d", i),
		)
	}
	// Add 1 unpublished listing (should not count)
	db.Exec(
		"INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, share_mode, status) VALUES (?, ?, X'00', 'Unpublished', 'free', 'unpublished')",
		userID, created.ID,
	)

	// List categories and check count
	rr2 := listCategories(t)
	var categories []PackCategory
	json.Unmarshal(rr2.Body.Bytes(), &categories)

	for _, cat := range categories {
		if cat.ID == created.ID {
			if cat.PackCount != 2 {
				t.Errorf("expected pack_count=2 for CountTest, got %d", cat.PackCount)
			}
			return
		}
	}
	t.Error("CountTest category not found in list")
}

func TestAdminCategories_InvalidID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPut, "/api/admin/categories/abc", bytes.NewReader([]byte(`{"name":"test"}`)))
	rr := httptest.NewRecorder()
	handleAdminCategories(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAdminCategories_MethodNotAllowed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// GET on /api/admin/categories (no ID) should be method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/admin/categories", nil)
	rr := httptest.NewRecorder()
	handleAdminCategories(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestCRUD_RoundTrip(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create
	rr := postCategoryJSON(t, map[string]string{"name": "RoundTrip", "description": "Test roundtrip"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rr.Code)
	}
	var created PackCategory
	json.Unmarshal(rr.Body.Bytes(), &created)

	// Read (via list)
	rr2 := listCategories(t)
	var categories []PackCategory
	json.Unmarshal(rr2.Body.Bytes(), &categories)
	found := false
	for _, cat := range categories {
		if cat.ID == created.ID {
			found = true
			if cat.Name != "RoundTrip" || cat.Description != "Test roundtrip" {
				t.Errorf("read back mismatch: got name=%q desc=%q", cat.Name, cat.Description)
			}
		}
	}
	if !found {
		t.Fatal("created category not found in list")
	}

	// Update
	rr3 := putCategoryJSON(t, created.ID, map[string]string{"name": "RoundTrip2", "description": "Updated"})
	if rr3.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d", rr3.Code)
	}

	// Read again
	rr4 := listCategories(t)
	var categories2 []PackCategory
	json.Unmarshal(rr4.Body.Bytes(), &categories2)
	for _, cat := range categories2 {
		if cat.ID == created.ID {
			if cat.Name != "RoundTrip2" || cat.Description != "Updated" {
				t.Errorf("after update: got name=%q desc=%q", cat.Name, cat.Description)
			}
		}
	}

	// Delete
	rr5 := deleteCategoryReq(t, created.ID)
	if rr5.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", rr5.Code)
	}

	// Verify gone
	rr6 := listCategories(t)
	var categories3 []PackCategory
	json.Unmarshal(rr6.Body.Bytes(), &categories3)
	for _, cat := range categories3 {
		if cat.ID == created.ID {
			t.Error("deleted category still in list")
		}
	}
}
