package main

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"marketplace_server/templates"
)

// Global database connection
var db *sql.DB

// MarketplaceUser 市场用户
type MarketplaceUser struct {
	ID              int64   `json:"id"`
	OAuthProvider   string  `json:"oauth_provider"`
	OAuthProviderID string  `json:"oauth_provider_id"`
	DisplayName     string  `json:"display_name"`
	Email           string  `json:"email"`
	CreditsBalance  float64 `json:"credits_balance"`
	CreatedAt       string  `json:"created_at"`
}

// PackCategory 分析包分类
type PackCategory struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPreset    bool   `json:"is_preset"`
	PackCount   int    `json:"pack_count"`
}

// PackListingInfo 分析包列表信息（不含文件数据）
type PackListingInfo struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	CategoryID      int64  `json:"category_id"`
	CategoryName    string `json:"category_name"`
	PackName        string `json:"pack_name"`
	PackDescription string `json:"pack_description"`
	SourceName      string `json:"source_name"`
	AuthorName      string `json:"author_name"`
	ShareMode       string `json:"share_mode"`
	CreditsPrice    int    `json:"credits_price"`
	DownloadCount   int    `json:"download_count"`
	CreatedAt       string `json:"created_at"`
}

// CreditsTransaction Credits 交易记录
type CreditsTransaction struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"user_id"`
	TransactionType string  `json:"transaction_type"`
	Amount          float64 `json:"amount"`
	ListingID       *int64  `json:"listing_id,omitempty"`
	Description     string  `json:"description"`
	CreatedAt       string  `json:"created_at"`
}

// initDB initializes the SQLite database with WAL mode and creates all required tables.
func initDB(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create users table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			oauth_provider TEXT NOT NULL,
			oauth_provider_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(oauth_provider, oauth_provider_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	// Create categories table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			is_preset INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create categories table: %w", err)
	}

	// Create pack_listings table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS pack_listings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			file_data BLOB NOT NULL,
			pack_name TEXT NOT NULL,
			pack_description TEXT,
			source_name TEXT,
			author_name TEXT,
			share_mode TEXT NOT NULL,
			credits_price INTEGER DEFAULT 0,
			status TEXT DEFAULT 'published',
			download_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create pack_listings table: %w", err)
	}

	// Create credits_transactions table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS credits_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			transaction_type TEXT NOT NULL,
			amount REAL NOT NULL,
			listing_id INTEGER,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create credits_transactions table: %w", err)
	}

	// Create settings table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create settings table: %w", err)
	}

	// Insert preset e-commerce categories (ignore if already exist)
	presetCategories := []struct {
		Name        string
		Description string
	}{
		{"Shopify", "Shopify e-commerce platform analysis packs"},
		{"BigCommerce", "BigCommerce e-commerce platform analysis packs"},
		{"eBay", "eBay marketplace analysis packs"},
		{"Etsy", "Etsy marketplace analysis packs"},
	}
	for _, cat := range presetCategories {
		_, err := database.Exec(
			"INSERT OR IGNORE INTO categories (name, description, is_preset) VALUES (?, ?, 1)",
			cat.Name, cat.Description,
		)
		if err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to insert preset category %s: %w", cat.Name, err)
		}
	}

	// Insert default settings (ignore if already exist)
	_, err = database.Exec(
		"INSERT OR IGNORE INTO settings (key, value) VALUES ('initial_credits_balance', '0')",
	)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to insert default settings: %w", err)
	}

	return database, nil
}

// jsonResponse writes a JSON response with the given status code.
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Placeholder handler for unimplemented endpoints
func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusNotImplemented, map[string]string{
		"error": "not_implemented",
	})
}

// jwtSecret is the HMAC-SHA256 signing key for JWT tokens.
var jwtSecret = []byte("marketplace-server-jwt-secret-key-2024")

// jwtHeader is the pre-encoded JWT header for HS256.
var jwtHeaderEncoded = base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

// base64URLEncode encodes data using base64url (no padding).
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode decodes a base64url-encoded string (no padding).
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// jwtPayload represents the JWT claims.
type jwtPayload struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	Exp         int64  `json:"exp"`
}

// generateJWT creates a JWT token with userID and displayName claims, valid for 24 hours.
func generateJWT(userID int64, displayName string) (string, error) {
	payload := jwtPayload{
		UserID:      userID,
		DisplayName: displayName,
		Exp:         time.Now().Add(24 * time.Hour).Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT payload: %w", err)
	}
	payloadEncoded := base64URLEncode(payloadJSON)

	signingInput := jwtHeaderEncoded + "." + payloadEncoded
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	signature := base64URLEncode(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// parseJWT validates a JWT token and returns the userID, displayName, and any error.
func parseJWT(tokenString string) (int64, string, error) {
	parts := strings.SplitN(tokenString, ".", 3)
	if len(parts) != 3 {
		return 0, "", fmt.Errorf("invalid token format")
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	expectedSig := base64URLEncode(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return 0, "", fmt.Errorf("invalid token signature")
	}

	// Decode payload
	payloadJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return 0, "", fmt.Errorf("failed to decode payload: %w", err)
	}
	var payload jwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return 0, "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > payload.Exp {
		return 0, "", fmt.Errorf("token_expired")
	}

	return payload.UserID, payload.DisplayName, nil
}

// authMiddleware validates the JWT token from the Authorization header.
// Returns 401 for missing/invalid tokens.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		userID, displayName, err := parseJWT(token)
		if err != nil {
			if err.Error() == "token_expired" {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "token_expired"})
			} else {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return
		}

		// Store user info in request headers for downstream handlers
		r.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		r.Header.Set("X-Display-Name", displayName)
		next(w, r)
	}
}

// getSetting reads a value from the settings table by key.
func getSetting(key string) string {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

// oauthCallbackRequest represents the JSON body for OAuth callback.
type oauthCallbackRequest struct {
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
}

// validOAuthProviders is the set of supported OAuth providers.
var validOAuthProviders = map[string]bool{
	"google":   true,
	"apple":    true,
	"facebook": true,
	"amazon":   true,
}

// handleOAuthCallback handles POST /api/auth/oauth.
// It validates the OAuth provider, creates new users on first login (with initial credits),
// returns existing users on repeat login, and issues a JWT token.
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"success": false,
			"error":   "method not allowed",
		})
		return
	}

	var req oauthCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request body",
		})
		return
	}

	// Validate provider
	if !validOAuthProviders[req.Provider] {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported oauth provider: %s", req.Provider),
		})
		return
	}

	// Validate required fields
	if req.ProviderUserID == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "provider_user_id is required",
		})
		return
	}
	if req.DisplayName == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "display_name is required",
		})
		return
	}

	// Check if user already exists
	var user MarketplaceUser
	err := db.QueryRow(
		"SELECT id, oauth_provider, oauth_provider_id, display_name, email, credits_balance, created_at FROM users WHERE oauth_provider = ? AND oauth_provider_id = ?",
		req.Provider, req.ProviderUserID,
	).Scan(&user.ID, &user.OAuthProvider, &user.OAuthProviderID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)

	if err == sql.ErrNoRows {
		// First-time login: create new user with initial credits
		initialBalanceStr := getSetting("initial_credits_balance")
		var initialBalance float64
		if initialBalanceStr != "" {
			fmt.Sscanf(initialBalanceStr, "%f", &initialBalance)
		}

		result, err := db.Exec(
			"INSERT INTO users (oauth_provider, oauth_provider_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			req.Provider, req.ProviderUserID, req.DisplayName, req.Email, initialBalance,
		)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Failed to get last insert ID: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}

		// Record initial credits transaction if balance > 0
		if initialBalance > 0 {
			_, err = db.Exec(
				"INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'initial', ?, 'Initial credits balance')",
				userID, initialBalance,
			)
			if err != nil {
				log.Printf("Failed to record initial credits transaction: %v", err)
				// Non-fatal: user was created, just log the error
			}
		}

		// Read back the created user
		err = db.QueryRow(
			"SELECT id, oauth_provider, oauth_provider_id, display_name, email, credits_balance, created_at FROM users WHERE id = ?",
			userID,
		).Scan(&user.ID, &user.OAuthProvider, &user.OAuthProviderID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)
		if err != nil {
			log.Printf("Failed to read back created user: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "internal_error",
			})
			return
		}
	} else if err != nil {
		log.Printf("Failed to query user: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	// Generate JWT token
	token, err := generateJWT(user.ID, user.DisplayName)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
		"user":    user,
	})
}

// handleListCategories handles GET /api/categories.
// Returns all categories with pack_count (number of published pack_listings per category).
func handleListCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := db.Query(`
		SELECT c.id, c.name, c.description, c.is_preset,
			COUNT(CASE WHEN pl.status = 'published' THEN 1 END) AS pack_count
		FROM categories c
		LEFT JOIN pack_listings pl ON pl.category_id = c.id
		GROUP BY c.id
		ORDER BY c.id
	`)
	if err != nil {
		log.Printf("Failed to query categories: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	categories := []PackCategory{}
	for rows.Next() {
		var cat PackCategory
		var isPreset int
		var desc sql.NullString
		if err := rows.Scan(&cat.ID, &cat.Name, &desc, &isPreset, &cat.PackCount); err != nil {
			log.Printf("Failed to scan category: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		cat.IsPreset = isPreset == 1
		if desc.Valid {
			cat.Description = desc.String
		}
		categories = append(categories, cat)
	}

	jsonResponse(w, http.StatusOK, categories)
}

// handleCreateCategory handles POST /api/admin/categories.
// Creates a new category with the given name and description.
func handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	result, err := db.Exec(
		"INSERT INTO categories (name, description, is_preset) VALUES (?, ?, 0)",
		req.Name, req.Description,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			jsonResponse(w, http.StatusConflict, map[string]string{"error": "category name already exists"})
			return
		}
		log.Printf("Failed to create category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get last insert ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	cat := PackCategory{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		IsPreset:    false,
		PackCount:   0,
	}
	jsonResponse(w, http.StatusCreated, cat)
}

// handleUpdateCategory handles PUT /api/admin/categories/{id}.
// Updates the name and description of an existing category.
func handleUpdateCategory(w http.ResponseWriter, r *http.Request, categoryID int64) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	result, err := db.Exec(
		"UPDATE categories SET name = ?, description = ? WHERE id = ?",
		req.Name, req.Description, categoryID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			jsonResponse(w, http.StatusConflict, map[string]string{"error": "category name already exists"})
			return
		}
		log.Printf("Failed to update category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	cat := PackCategory{
		ID:          categoryID,
		Name:        req.Name,
		Description: req.Description,
	}
	// Read back is_preset
	var isPreset int
	db.QueryRow("SELECT is_preset FROM categories WHERE id = ?", categoryID).Scan(&isPreset)
	cat.IsPreset = isPreset == 1

	jsonResponse(w, http.StatusOK, cat)
}

// handleDeleteCategory handles DELETE /api/admin/categories/{id}.
// Refuses deletion if the category has associated pack_listings.
func handleDeleteCategory(w http.ResponseWriter, r *http.Request, categoryID int64) {
	// Check for associated pack_listings
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM pack_listings WHERE category_id = ?", categoryID).Scan(&count)
	if err != nil {
		log.Printf("Failed to count pack listings: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if count > 0 {
		jsonResponse(w, http.StatusConflict, map[string]interface{}{
			"error": "category_has_listings",
			"count": count,
		})
		return
	}

	result, err := db.Exec("DELETE FROM categories WHERE id = ?", categoryID)
	if err != nil {
		log.Printf("Failed to delete category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleAdminCategories dispatches admin category requests based on HTTP method and URL path.
func handleAdminCategories(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/admin/categories/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/categories")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		// No ID in path: only POST is valid
		if r.Method == http.MethodPost {
			handleCreateCategory(w, r)
			return
		}
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse category ID
	categoryID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category id"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		handleUpdateCategory(w, r, categoryID)
	case http.MethodDelete:
		handleDeleteCategory(w, r, categoryID)
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// qapFileContent represents the JSON structure inside a .qap ZIP file.
type qapFileContent struct {
	FileType      string `json:"file_type"`
	FormatVersion string `json:"format_version"`
	Metadata      struct {
		Author      string `json:"author"`
		CreatedAt   string `json:"created_at"`
		SourceName  string `json:"source_name"`
		Description string `json:"description"`
	} `json:"metadata"`
}

// handleUploadPack handles POST /api/packs/upload.
// Accepts a multipart form with a .qap file and sharing settings.
func handleUploadPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Validate share_mode
	shareMode := r.FormValue("share_mode")
	if shareMode != "free" && shareMode != "paid" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "share_mode must be 'free' or 'paid'"})
		return
	}

	// Validate credits_price for paid mode
	var creditsPrice int
	if shareMode == "paid" {
		creditsPriceStr := r.FormValue("credits_price")
		if creditsPriceStr == "" {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "credits_price is required for paid share_mode"})
			return
		}
		creditsPrice, err = strconv.Atoi(creditsPriceStr)
		if err != nil || creditsPrice <= 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "credits_price must be a positive integer"})
			return
		}
	}

	// Validate category_id
	categoryIDStr := r.FormValue("category_id")
	if categoryIDStr == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "category_id is required"})
		return
	}
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category_id"})
		return
	}

	// Check category exists
	var categoryName string
	err = db.QueryRow("SELECT name FROM categories WHERE id = ?", categoryID).Scan(&categoryName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "category not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Read uploaded file
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Failed to read uploaded file: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Parse .qap file as ZIP and extract metadata
	zipReader, err := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	var qapContent qapFileContent
	var foundJSON bool
	for _, f := range zipReader.File {
		if f.Name == "analysis_pack.json" {
			rc, err := f.Open()
			if err != nil {
				jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
				return
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
				return
			}
			if err := json.Unmarshal(jsonData, &qapContent); err != nil {
				jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
				return
			}
			foundJSON = true
			break
		}
	}

	if !foundJSON {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	// Use source_name as pack_name, fall back to "Untitled"
	packName := qapContent.Metadata.SourceName
	if packName == "" {
		packName = "Untitled"
	}

	// Insert pack_listing record
	result, err := db.Exec(
		`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published')`,
		userID, categoryID, fileData, packName, qapContent.Metadata.Description,
		qapContent.Metadata.SourceName, qapContent.Metadata.Author, shareMode, creditsPrice,
	)
	if err != nil {
		log.Printf("Failed to insert pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	listingID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get last insert ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Read back the created listing
	var listing PackListingInfo
	err = db.QueryRow(
		`SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		        pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.created_at
		 FROM pack_listings pl
		 JOIN categories c ON c.id = pl.category_id
		 WHERE pl.id = ?`, listingID,
	).Scan(&listing.ID, &listing.UserID, &listing.CategoryID, &listing.CategoryName,
		&listing.PackName, &listing.PackDescription, &listing.SourceName, &listing.AuthorName,
		&listing.ShareMode, &listing.CreditsPrice, &listing.DownloadCount, &listing.CreatedAt)
	if err != nil {
		log.Printf("Failed to read back listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusCreated, listing)
}
// handleListPacks handles GET /api/packs.
// Returns a list of published PackListingInfo (without file_data).
// Supports optional category_id query parameter for filtering.
func handleListPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query := `
		SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.status = 'published'`
	var args []interface{}

	categoryIDStr := r.URL.Query().Get("category_id")
	if categoryIDStr != "" {
		categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category_id"})
			return
		}
		query += " AND pl.category_id = ?"
		args = append(args, categoryID)
	}

	query += " ORDER BY pl.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query pack listings: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	listings := []PackListingInfo{}
	for rows.Next() {
		var l PackListingInfo
		var desc, sourceName, authorName sql.NullString
		if err := rows.Scan(&l.ID, &l.UserID, &l.CategoryID, &l.CategoryName,
			&l.PackName, &desc, &sourceName, &authorName,
			&l.ShareMode, &l.CreditsPrice, &l.DownloadCount, &l.CreatedAt); err != nil {
			log.Printf("Failed to scan pack listing: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if desc.Valid {
			l.PackDescription = desc.String
		}
		if sourceName.Valid {
			l.SourceName = sourceName.String
		}
		if authorName.Valid {
			l.AuthorName = authorName.String
		}
		listings = append(listings, l)
	}

	jsonResponse(w, http.StatusOK, listings)
}

// handleDownloadPack handles GET /api/packs/{id}/download.
// Free packs return file data directly. Paid packs check credits balance,
// deduct if sufficient, and return file data; otherwise return 402.
func handleDownloadPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract pack ID from URL: /api/packs/{id}/download
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/download")
	packID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid pack id"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Look up the pack listing (must be published)
	var shareMode string
	var creditsPrice int
	var fileData []byte
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, file_data, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		packID,
	).Scan(&shareMode, &creditsPrice, &fileData, &packName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if shareMode == "paid" {
		// Check user's credits balance
		var balance float64
		err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
		if err != nil {
			log.Printf("Failed to query user balance: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		if balance < float64(creditsPrice) {
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "insufficient_credits",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		// Use a database transaction for atomic credits deduction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Failed to begin transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		defer tx.Rollback()

		// Deduct credits from user balance
		result, err := tx.Exec(
			"UPDATE users SET credits_balance = credits_balance - ? WHERE id = ? AND credits_balance >= ?",
			creditsPrice, userID, creditsPrice,
		)
		if err != nil {
			log.Printf("Failed to deduct credits: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// Race condition: balance changed between check and update
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "insufficient_credits",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		// Record credits transaction
		_, err = tx.Exec(
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
			 VALUES (?, 'consume', ?, ?, ?)`,
			userID, float64(creditsPrice), packID, fmt.Sprintf("Download pack: %s", packName),
		)
		if err != nil {
			log.Printf("Failed to record transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		// Increment download count
		_, err = tx.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
		if err != nil {
			log.Printf("Failed to increment download count: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	} else {
		// Free pack: just increment download count
		_, err = db.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
		if err != nil {
			log.Printf("Failed to increment download count: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	// Return file data as binary response
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.qap"`, packName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)
}

// handleGetBalance returns the authenticated user's current credits balance.
// GET /api/credits/balance
func handleGetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var balance float64
	err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query user balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"credits_balance": balance,
	})
}

// purchaseRequest represents the JSON body for a credits purchase.
type purchaseRequest struct {
	Amount float64 `json:"amount"`
}

// handlePurchaseCredits processes a credits purchase, increasing the user's balance.
// POST /api/credits/purchase
func handlePurchaseCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req purchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Amount <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}

	// Use a database transaction for atomic balance update
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	// Verify user exists
	var currentBalance float64
	err = tx.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&currentBalance)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query user balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Update balance
	_, err = tx.Exec("UPDATE users SET credits_balance = credits_balance + ? WHERE id = ?", req.Amount, userID)
	if err != nil {
		log.Printf("Failed to update credits balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, description)
		 VALUES (?, 'purchase', ?, ?)`,
		userID, req.Amount, fmt.Sprintf("Purchase %.2f credits", req.Amount),
	)
	if err != nil {
		log.Printf("Failed to record transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	newBalance := currentBalance + req.Amount
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"credits_balance": newBalance,
		"amount_added":    req.Amount,
	})
}

// handleListTransactions returns the authenticated user's credits transaction history.
// GET /api/credits/transactions
func handleListTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	rows, err := db.Query(
		`SELECT id, user_id, transaction_type, amount, listing_id, description, created_at
		 FROM credits_transactions WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		log.Printf("Failed to query transactions: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	transactions := []CreditsTransaction{}
	for rows.Next() {
		var t CreditsTransaction
		var listingID sql.NullInt64
		err := rows.Scan(&t.ID, &t.UserID, &t.TransactionType, &t.Amount, &listingID, &t.Description, &t.CreatedAt)
		if err != nil {
			log.Printf("Failed to scan transaction row: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if listingID.Valid {
			t.ListingID = &listingID.Int64
		}
		transactions = append(transactions, t)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
	})
}
// adminTmpl is the parsed admin panel HTML template.
var adminTmpl = template.Must(template.New("admin").Parse(templates.AdminHTML))

// handleAdminDashboard renders the admin panel HTML page.
// GET /admin/
func handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	initialCredits := getSetting("initial_credits_balance")
	if initialCredits == "" {
		initialCredits = "0"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	adminTmpl.Execute(w, map[string]interface{}{
		"InitialCredits": initialCredits,
	})
}

// handleSetInitialCredits updates the initial_credits_balance setting.
// POST /admin/settings/initial-credits
func handleSetInitialCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}

	// Validate that value is a non-negative integer
	credits, err := strconv.Atoi(value)
	if err != nil || credits < 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be a non-negative integer"})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('initial_credits_balance', ?)", value)
	if err != nil {
		log.Printf("Failed to update initial_credits_balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

func main() {
	port := flag.Int("port", 8090, "Server port")
	dbPath := flag.String("db", "marketplace.db", "SQLite database path")
	flag.Parse()

	var err error
	db, err = initDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Auth routes
	http.HandleFunc("/api/auth/oauth", handleOAuthCallback)

	// Category routes (listing is public)
	http.HandleFunc("/api/categories", handleListCategories)
	http.HandleFunc("/api/admin/categories", handleAdminCategories)
	http.HandleFunc("/api/admin/categories/", handleAdminCategories)

	// Pack routes (upload and download require auth, listing is public)
	http.HandleFunc("/api/packs/upload", authMiddleware(handleUploadPack))
	http.HandleFunc("/api/packs", handleListPacks)
	http.HandleFunc("/api/packs/", authMiddleware(handleDownloadPack))

	// Credits routes (all require auth)
	http.HandleFunc("/api/credits/balance", authMiddleware(handleGetBalance))
	http.HandleFunc("/api/credits/purchase", authMiddleware(handlePurchaseCredits))
	http.HandleFunc("/api/credits/transactions", authMiddleware(handleListTransactions))

	// Admin routes
	http.HandleFunc("/admin/settings/initial-credits", handleSetInitialCredits)
	http.HandleFunc("/admin/", handleAdminDashboard)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Marketplace server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
