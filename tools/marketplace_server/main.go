package main

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"marketplace_server/templates"
)

// Global database connection
var db *sql.DB

// Session store (in-memory)
var (
	sessions   = make(map[string]sessionEntry) // sessionID -> entry
	sessionsMu sync.RWMutex
)

type sessionEntry struct {
	AdminID int64
	Expiry  time.Time
}

// Captcha store (in-memory)
var (
	captchas   = make(map[string]captchaEntry) // captchaID -> entry
	captchasMu sync.RWMutex
)

type captchaEntry struct {
	Code   string
	Expiry time.Time
}

// MarketplaceUser 市场用户
type MarketplaceUser struct {
	ID             int64   `json:"id"`
	AuthType       string  `json:"auth_type"`
	AuthID         string  `json:"auth_id"`
	DisplayName    string  `json:"display_name"`
	Email          string  `json:"email"`
	CreditsBalance float64 `json:"credits_balance"`
	CreatedAt      string  `json:"created_at"`
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
	ID              int64            `json:"id"`
	UserID          int64            `json:"user_id"`
	CategoryID      int64            `json:"category_id"`
	CategoryName    string           `json:"category_name"`
	PackName        string           `json:"pack_name"`
	PackDescription string           `json:"pack_description"`
	SourceName      string           `json:"source_name"`
	AuthorName      string           `json:"author_name"`
	ShareMode       string           `json:"share_mode"`
	CreditsPrice    int              `json:"credits_price"`
	ValidDays       int              `json:"valid_days"`
	BillingCycle    string           `json:"billing_cycle"`
	DownloadCount   int              `json:"download_count"`
	Status          string           `json:"status"`
	RejectReason    string           `json:"reject_reason,omitempty"`
	ReviewedBy      *int64           `json:"reviewed_by,omitempty"`
	ReviewedAt      string           `json:"reviewed_at,omitempty"`
	MetaInfo        json.RawMessage  `json:"meta_info"`
	CreatedAt       string           `json:"created_at"`
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
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create users table (new schema with auth_type/auth_id)
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_type TEXT NOT NULL,
			auth_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			credits_balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(auth_type, auth_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	// Migrate users table: rename oauth_provider/oauth_provider_id to auth_type/auth_id
	var usersTableSQL string
	err = database.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&usersTableSQL)
	if err == nil && strings.Contains(usersTableSQL, "oauth_provider") {
		if _, err := database.Exec(`
			CREATE TABLE users_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				auth_type TEXT NOT NULL,
				auth_id TEXT NOT NULL,
				display_name TEXT NOT NULL,
				email TEXT,
				credits_balance REAL DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(auth_type, auth_id)
			)
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to create users_new table: %w", err)
		}
		if _, err := database.Exec(`
			INSERT INTO users_new (id, auth_type, auth_id, display_name, email, credits_balance, created_at)
			SELECT id, oauth_provider, oauth_provider_id, display_name, email, credits_balance, created_at FROM users
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to migrate users data: %w", err)
		}
		if _, err := database.Exec(`DROP TABLE users`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to drop old users table: %w", err)
		}
		if _, err := database.Exec(`ALTER TABLE users_new RENAME TO users`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to rename users_new table: %w", err)
		}
		log.Println("Migrated users table: oauth_provider/oauth_provider_id → auth_type/auth_id")
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
			status TEXT DEFAULT 'pending',
			download_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create pack_listings table: %w", err)
	}

	// Add review-related columns to pack_listings (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN reject_reason TEXT")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN reviewed_by INTEGER REFERENCES admin_credentials(id)")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN reviewed_at DATETIME")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN meta_info TEXT DEFAULT '{}'")

	// Add billing-related columns to pack_listings (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN valid_days INTEGER DEFAULT 0")
	database.Exec("ALTER TABLE pack_listings ADD COLUMN billing_cycle TEXT DEFAULT ''")
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

	// Migrate admin_credentials table: detect old CHECK(id=1) constraint and rebuild
	var adminTableSQL string
	err = database.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='admin_credentials'").Scan(&adminTableSQL)
	if err == nil && strings.Contains(adminTableSQL, "CHECK") {
		// Old table detected, migrate to new schema
		if _, err := database.Exec(`
			CREATE TABLE admin_credentials_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				role TEXT NOT NULL DEFAULT 'regular',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to create admin_credentials_new table: %w", err)
		}
		if _, err := database.Exec(`
			INSERT INTO admin_credentials_new (id, username, password_hash, role, created_at)
			SELECT id, username, password_hash, 'super', created_at FROM admin_credentials
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to migrate admin_credentials data: %w", err)
		}
		if _, err := database.Exec(`DROP TABLE admin_credentials`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to drop old admin_credentials table: %w", err)
		}
		if _, err := database.Exec(`ALTER TABLE admin_credentials_new RENAME TO admin_credentials`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to rename admin_credentials_new table: %w", err)
		}
	} else {
		// Fresh install or already migrated: create new schema
		if _, err := database.Exec(`
			CREATE TABLE IF NOT EXISTS admin_credentials (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				role TEXT NOT NULL DEFAULT 'regular',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`); err != nil {
			database.Close()
			return nil, fmt.Errorf("failed to create admin_credentials table: %w", err)
		}
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
// In production, override via MARKETPLACE_JWT_SECRET environment variable.
var jwtSecret = func() []byte {
	if s := os.Getenv("MARKETPLACE_JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("marketplace-server-jwt-secret-key-2024")
}()

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

// --- Admin Authentication System ---

// isAdminSetup checks if admin credentials have been configured.
func isAdminSetup() bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM admin_credentials").Scan(&count)
	return err == nil && count > 0
}

// hashPassword hashes a password using SHA-256 with a random salt.
func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(h.Sum(nil))
}

// checkPassword verifies a password against a stored hash.
func checkPassword(password, stored string) bool {
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, _ := hex.DecodeString(parts[0])
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil)) == parts[1]
}

// generateSessionID creates a random session ID.
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// createSession creates a new session and returns the session ID.
func createSession(adminID int64) string {
	id := generateSessionID()
	sessionsMu.Lock()
	sessions[id] = sessionEntry{AdminID: adminID, Expiry: time.Now().Add(24 * time.Hour)}
	sessionsMu.Unlock()
	return id
}

// isValidSession checks if a session ID is valid and not expired.
func isValidSession(id string) bool {
	sessionsMu.RLock()
	entry, ok := sessions[id]
	sessionsMu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(entry.Expiry) {
		sessionsMu.Lock()
		delete(sessions, id)
		sessionsMu.Unlock()
		return false
	}
	return true
}

// getSessionFromRequest extracts session ID from cookie.
func getSessionFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("admin_session")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getSessionAdminID returns the Admin_ID associated with the session in the request.
// Returns 0 if no valid session is found or the session has expired.
func getSessionAdminID(r *http.Request) int64 {
	sid := getSessionFromRequest(r)
	if sid == "" {
		return 0
	}
	sessionsMu.RLock()
	entry, ok := sessions[sid]
	sessionsMu.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		return 0
	}
	return entry.AdminID
}

// getAdminRole returns the role ("super" or "regular") for the given admin ID.
// Returns "" if the admin is not found.
func getAdminRole(adminID int64) string {
	var role string
	err := db.QueryRow("SELECT role FROM admin_credentials WHERE id = ?", adminID).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

// --- Captcha Generation (pure Go, no external deps) ---

// generateCaptchaCode creates a random 4-digit code.
func generateCaptchaCode() string {
	digits := "0123456789"
	code := make([]byte, 4)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		code[i] = digits[n.Int64()]
	}
	return string(code)
}

// createCaptcha generates a captcha and stores it, returns captchaID.
func createCaptcha() string {
	id := generateSessionID()[:16]
	code := generateCaptchaCode()
	log.Printf("[CAPTCHA] created id=%s code=%s", id, code)
	captchasMu.Lock()
	captchas[id] = captchaEntry{Code: code, Expiry: time.Now().Add(5 * time.Minute)}
	captchasMu.Unlock()
	return id
}

// verifyCaptcha checks if the captcha answer is correct.
func verifyCaptcha(id, answer string) bool {
	captchasMu.Lock()
	entry, ok := captchas[id]
	delete(captchas, id) // one-time use
	captchasMu.Unlock()
	if !ok || time.Now().After(entry.Expiry) {
		return false
	}
	return strings.EqualFold(entry.Code, answer)
}

// getCaptchaCode returns the code for a captcha ID (for image generation).
func getCaptchaCode(id string) string {
	captchasMu.RLock()
	entry, ok := captchas[id]
	captchasMu.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		return ""
	}
	return entry.Code
}

// drawDigit draws a single digit on the image at position x.
func drawDigit(img *image.RGBA, digit byte, xOff, yOff int, c color.RGBA) {
	// Simple 5x7 bitmap font for digits 0-9
	font := map[byte][]string{
		'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
		'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
		'2': {"01110", "10001", "00010", "00100", "01000", "10000", "11111"},
		'3': {"01110", "10001", "00001", "00110", "00001", "10001", "01110"},
		'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
		'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
		'6': {"01110", "10000", "11110", "10001", "10001", "10001", "01110"},
		'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
		'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
		'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	}
	pattern, ok := font[digit]
	if !ok {
		return
	}
	scale := 4
	for row, line := range pattern {
		for col, ch := range line {
			if ch == '1' {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetRGBA(xOff+col*scale+dx, yOff+row*scale+dy, c)
					}
				}
			}
		}
	}
}

// generateCaptchaImage creates a PNG image for the given captcha ID.
func generateCaptchaImage(id string) []byte {
	code := getCaptchaCode(id)
	if code == "" {
		return nil
	}

	width, height := 160, 50
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Background
	bg := color.RGBA{240, 240, 245, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, bg)
		}
	}

	// Add noise dots
	noiseColors := []color.RGBA{
		{180, 180, 190, 255}, {200, 200, 210, 255}, {160, 170, 180, 255},
	}
	for i := 0; i < 100; i++ {
		nx, _ := rand.Int(rand.Reader, big.NewInt(int64(width)))
		ny, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		ci, _ := rand.Int(rand.Reader, big.NewInt(int64(len(noiseColors))))
		img.SetRGBA(int(nx.Int64()), int(ny.Int64()), noiseColors[ci.Int64()])
	}

	// Add noise lines
	lineColor := color.RGBA{180, 180, 200, 255}
	for l := 0; l < 3; l++ {
		y1r, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y2r, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y1, y2 := int(y1r.Int64()), int(y2r.Int64())
		for x := 0; x < width; x++ {
			y := y1 + (y2-y1)*x/width
			if y >= 0 && y < height {
				img.SetRGBA(x, y, lineColor)
				if y+1 < height {
					img.SetRGBA(x, y+1, lineColor)
				}
			}
		}
	}

	// Draw digits
	digitColors := []color.RGBA{
		{50, 50, 120, 255}, {120, 40, 40, 255}, {40, 100, 50, 255}, {100, 50, 120, 255},
	}
	for i, ch := range code {
		xOff := 15 + i*35
		yOff := 8
		drawDigit(img, byte(ch), xOff, yOff, digitColors[i%len(digitColors)])
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// adminAuth protects admin routes with session-based authentication.
func isAPIRoute(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/")
}

func makeSessionCookie(name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

func adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdminSetup() {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "not_setup"})
			} else {
				http.Redirect(w, r, "/admin/setup", http.StatusFound)
			}
			return
		}
		sid := getSessionFromRequest(r)
		log.Printf("[AUTH] path=%s sid=%q valid=%v", r.URL.Path, sid, isValidSession(sid))
		if !isValidSession(sid) {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		adminID := getSessionAdminID(r)
		if adminID == 0 {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
		next(w, r)
	}
}

func superAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdminSetup() {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "not_setup"})
			} else {
				http.Redirect(w, r, "/admin/setup", http.StatusFound)
			}
			return
		}
		sid := getSessionFromRequest(r)
		if !isValidSession(sid) {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		adminID := getSessionAdminID(r)
		if adminID == 0 {
			if isAPIRoute(r) {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
			}
			return
		}
		role := getAdminRole(adminID)
		if role != "super" {
			jsonResponse(w, http.StatusForbidden, map[string]string{"error": "permission_denied"})
			return
		}
		r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
		next(w, r)
	}
}


// handleAdminSetup handles the first-time admin setup page.
func handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if isAdminSetup() {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		captchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SetupTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
		})
		return
	}

	// POST
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	password2 := r.FormValue("password2")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha"))

	errMsg := ""
	if username == "" || len(username) < 3 {
		errMsg = "用户名至少3个字符"
	} else if password == "" || len(password) < 6 {
		errMsg = "密码至少6个字符"
	} else if password != password2 {
		errMsg = "两次输入的密码不一致"
	} else if !verifyCaptcha(captchaID, captchaAns) {
		errMsg = "验证码错误"
	}

	if errMsg != "" {
		newCaptchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SetupTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     errMsg,
			"Username":  username,
		})
		return
	}

	hash := hashPassword(password)
	result, err := db.Exec("INSERT INTO admin_credentials (username, password_hash, role) VALUES (?, ?, 'super')", username, hash)
	if err != nil {
		log.Printf("Failed to save admin credentials: %v", err)
		newCaptchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SetupTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     "保存失败，请重试",
			"Username":  username,
		})
		return
	}

	adminID, _ := result.LastInsertId()
	// Auto-login after setup
	sid := createSession(adminID)
	http.SetCookie(w, makeSessionCookie("admin_session", sid, 86400))
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

// handleAdminLogin handles the login page.
func handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if !isAdminSetup() {
		http.Redirect(w, r, "/admin/setup", http.StatusFound)
		return
	}

	// Already logged in?
	if isValidSession(getSessionFromRequest(r)) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		captchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
		})
		return
	}

	// POST
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha"))

	log.Printf("[LOGIN] attempt: username=%q, captchaID=%q, captchaAns=%q", username, captchaID, captchaAns)

	errMsg := ""
	var adminID int64
	if !verifyCaptcha(captchaID, captchaAns) {
		log.Printf("[LOGIN] captcha verification failed for ID=%q answer=%q", captchaID, captchaAns)
		errMsg = "验证码错误"
	} else {
		var storedHash string
		err := db.QueryRow("SELECT id, password_hash FROM admin_credentials WHERE username = ?", username).Scan(&adminID, &storedHash)
		if err != nil {
			log.Printf("[LOGIN] db query error for username=%q: %v", username, err)
			errMsg = "用户名或密码错误"
		} else if !checkPassword(password, storedHash) {
			log.Printf("[LOGIN] password check failed for username=%q adminID=%d", username, adminID)
			errMsg = "用户名或密码错误"
		} else {
			log.Printf("[LOGIN] success for username=%q adminID=%d", username, adminID)
		}
	}

	if errMsg != "" {
		newCaptchaID := createCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginTmpl.Execute(w, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     errMsg,
			"Username":  username,
		})
		return
	}

	sid := createSession(adminID)
	http.SetCookie(w, makeSessionCookie("admin_session", sid, 86400))
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

// handleAdminLogout logs out the admin.
func handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	sid := getSessionFromRequest(r)
	if sid != "" {
		sessionsMu.Lock()
		delete(sessions, sid)
		sessionsMu.Unlock()
	}
	http.SetCookie(w, makeSessionCookie("admin_session", "", -1))
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

// handleCaptchaImage serves the captcha image.
func handleCaptchaImage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	data := generateCaptchaImage(id)
	if data == nil {
		http.Error(w, "captcha expired", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(data)
}

// handleCaptchaRefresh creates a new captcha and returns its ID as JSON.
func handleCaptchaRefresh(w http.ResponseWriter, r *http.Request) {
	id := createCaptcha()
	jsonResponse(w, http.StatusOK, map[string]string{"captcha_id": id})
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
		"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE auth_type = ? AND auth_id = ?",
		req.Provider, req.ProviderUserID,
	).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)

	if err == sql.ErrNoRows {
		// First-time login: create new user with initial credits
		initialBalanceStr := getSetting("initial_credits_balance")
		var initialBalance float64
		if initialBalanceStr != "" {
			fmt.Sscanf(initialBalanceStr, "%f", &initialBalance)
		}

		result, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
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
			"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE id = ?",
			userID,
		).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)
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

// licenseServerURL returns the License server base URL from environment variable.
var licenseServerURL = func() string {
	if u := os.Getenv("LICENSE_SERVER_URL"); u != "" {
		return u
	}
	return "https://license.vantagedata.chat"
}()

// snLoginRequest represents the JSON body for SN login.
type snLoginRequest struct {
	LicenseToken string `json:"license_token"`
}

// licenseVerifyRequest is the request body sent to License server's /api/marketplace-verify.
type licenseVerifyRequest struct {
	Token string `json:"token"`
}

// licenseVerifyResponse is the response from License server's /api/marketplace-verify.
type licenseVerifyResponse struct {
	Success bool   `json:"success"`
	SN      string `json:"sn"`
	Email   string `json:"email"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// handleSNLogin handles POST /api/auth/sn-login.
// It verifies the license_token with the License server, then finds or creates a marketplace user.
func handleSNLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"success": false,
			"error":   "method not allowed",
		})
		return
	}

	var req snLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request body",
		})
		return
	}

	if req.LicenseToken == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "license_token is required",
		})
		return
	}

	// Step 1: Verify the license token with the License server
	verifyReqBody, err := json.Marshal(licenseVerifyRequest{Token: req.LicenseToken})
	if err != nil {
		log.Printf("Failed to marshal verify request: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "internal_error",
		})
		return
	}

	verifyURL := licenseServerURL + "/api/marketplace-verify"
	httpResp, err := http.Post(verifyURL, "application/json", bytes.NewReader(verifyReqBody))
	if err != nil {
		log.Printf("Failed to contact license server at %s: %v", verifyURL, err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{
			"success": false,
			"error":   "LICENSE_SERVER_ERROR",
			"message": "Failed to contact license server",
		})
		return
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Printf("Failed to read license server response: %v", err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{
			"success": false,
			"error":   "LICENSE_SERVER_ERROR",
			"message": "Failed to read license server response",
		})
		return
	}

	var verifyResp licenseVerifyResponse
	if err := json.Unmarshal(respBody, &verifyResp); err != nil {
		log.Printf("Failed to parse license server response: %v", err)
		jsonResponse(w, http.StatusBadGateway, map[string]interface{}{
			"success": false,
			"error":   "LICENSE_SERVER_ERROR",
			"message": "Invalid license server response",
		})
		return
	}

	if !verifyResp.Success {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "AUTH_FAILED",
			"message": verifyResp.Message,
			"code":    verifyResp.Code,
		})
		return
	}

	sn := verifyResp.SN
	email := verifyResp.Email

	if sn == "" {
		jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "AUTH_FAILED",
			"message": "License server returned empty SN",
		})
		return
	}

	// Step 2: Find or create marketplace user using SN as auth_id
	var user MarketplaceUser
	err = db.QueryRow(
		"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE auth_type = ? AND auth_id = ?",
		"sn", sn,
	).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)

	if err == sql.ErrNoRows {
		// New user: create with display_name = email prefix
		displayName := email
		if idx := strings.Index(email, "@"); idx > 0 {
			displayName = email[:idx]
		}

		initialBalanceStr := getSetting("initial_credits_balance")
		var initialBalance float64
		if initialBalanceStr != "" {
			fmt.Sscanf(initialBalanceStr, "%f", &initialBalance)
		}

		result, err := db.Exec(
			"INSERT INTO users (auth_type, auth_id, display_name, email, credits_balance) VALUES (?, ?, ?, ?, ?)",
			"sn", sn, displayName, email, initialBalance,
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
			}
		}

		// Read back the created user
		err = db.QueryRow(
			"SELECT id, auth_type, auth_id, display_name, email, credits_balance, created_at FROM users WHERE id = ?",
			userID,
		).Scan(&user.ID, &user.AuthType, &user.AuthID, &user.DisplayName, &user.Email, &user.CreditsBalance, &user.CreatedAt)
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

	// Step 3: Generate marketplace JWT
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
		"user": map[string]interface{}{
			"id":              user.ID,
			"display_name":    user.DisplayName,
			"email":           user.Email,
			"credits_balance": user.CreditsBalance,
		},
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

	jsonResponse(w, http.StatusOK, map[string]interface{}{"categories": categories})
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
	// Check if category is a preset (not deletable)
	var isPreset int
	err := db.QueryRow("SELECT is_preset FROM categories WHERE id = ?", categoryID).Scan(&isPreset)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to check category: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if isPreset == 1 {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "preset categories cannot be deleted"})
		return
	}

	// Check for associated pack_listings
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pack_listings WHERE category_id = ?", categoryID).Scan(&count)
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

// schemaColumn represents a column in a schema requirement.
type schemaColumn struct {
	Name string `json:"name"`
}

// schemaRequirement represents a table and its columns in schema_requirements.
type schemaRequirement struct {
	TableName string         `json:"table_name"`
	Columns   []schemaColumn `json:"columns"`
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
	SchemaRequirements []schemaRequirement `json:"schema_requirements"`
}

// PackMetaInfo represents extracted meta information from a QAP file.
type PackMetaInfo struct {
	Tables []PackMetaTable `json:"tables"`
}

// PackMetaTable represents a table and its column names in pack meta info.
type PackMetaTable struct {
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
}

// validatePricingParams validates the pricing parameters based on the share_mode (pricing model).
// Returns an error message string if validation fails, or empty string if valid.
func validatePricingParams(shareMode string, creditsPrice int, validDays int, billingCycle string) string {
	switch shareMode {
	case "free":
		return ""
	case "per_use":
		if creditsPrice <= 0 {
			return "credits_price must be a positive integer for per_use mode"
		}
		return ""
	case "time_limited":
		if creditsPrice <= 0 {
			return "credits_price must be a positive integer for time_limited mode"
		}
		if validDays <= 0 {
			return "valid_days must be a positive integer for time_limited mode"
		}
		return ""
	case "subscription":
		if creditsPrice <= 0 {
			return "credits_price must be a positive integer for subscription mode"
		}
		if billingCycle != "monthly" && billingCycle != "yearly" {
			return "billing_cycle must be 'monthly' or 'yearly' for subscription mode"
		}
		return ""
	default:
		return "share_mode must be 'free', 'per_use', 'time_limited', or 'subscription'"
	}
}

// handleUploadPack handles POST /api/packs/upload.
// Accepts a multipart form with a .qap file and sharing settings.
func handleUploadPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse multipart form (max 500MB to match client limit)
	if err := r.ParseMultipartForm(500 << 20); err != nil {
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

	// Validate share_mode (pricing model)
	shareMode := r.FormValue("share_mode")
	if shareMode != "free" && shareMode != "per_use" && shareMode != "time_limited" && shareMode != "subscription" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "share_mode must be 'free', 'per_use', 'time_limited', or 'subscription'"})
		return
	}

	// Parse credits_price
	var creditsPrice int
	creditsPriceStr := r.FormValue("credits_price")
	if creditsPriceStr != "" {
		creditsPrice, err = strconv.Atoi(creditsPriceStr)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "credits_price must be a valid integer"})
			return
		}
	}

	// Parse valid_days
	var validDays int
	validDaysStr := r.FormValue("valid_days")
	if validDaysStr != "" {
		validDays, err = strconv.Atoi(validDaysStr)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "valid_days must be a valid integer"})
			return
		}
	}

	// Parse billing_cycle
	billingCycle := r.FormValue("billing_cycle")

	// Validate pricing parameters
	if errMsg := validatePricingParams(shareMode, creditsPrice, validDays, billingCycle); errMsg != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
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

	fileData, err := io.ReadAll(io.LimitReader(file, 500*1024*1024+1))
	if err != nil {
		log.Printf("Failed to read uploaded file: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if int64(len(fileData)) > 500*1024*1024 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "file_too_large"})
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

	// Extract meta info from schema_requirements
	metaInfo := PackMetaInfo{Tables: []PackMetaTable{}}
	for _, sr := range qapContent.SchemaRequirements {
		table := PackMetaTable{
			TableName: sr.TableName,
			Columns:   []string{},
		}
		for _, col := range sr.Columns {
			table.Columns = append(table.Columns, col.Name)
		}
		metaInfo.Tables = append(metaInfo.Tables, table)
	}

	metaInfoJSON := "{}"
	if len(metaInfo.Tables) > 0 {
		if b, err := json.Marshal(metaInfo); err == nil {
			metaInfoJSON = string(b)
		}
	}

	// Insert pack_listing record
	result, err := db.Exec(
		`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, valid_days, billing_cycle, status, meta_info)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
		userID, categoryID, fileData, packName, qapContent.Metadata.Description,
		qapContent.Metadata.SourceName, qapContent.Metadata.Author, shareMode, creditsPrice, validDays, billingCycle, metaInfoJSON,
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
	var metaInfoReadBack sql.NullString
	err = db.QueryRow(
		`SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		        pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.valid_days, pl.billing_cycle, pl.download_count, pl.status, pl.meta_info, pl.created_at
		 FROM pack_listings pl
		 JOIN categories c ON c.id = pl.category_id
		 WHERE pl.id = ?`, listingID,
	).Scan(&listing.ID, &listing.UserID, &listing.CategoryID, &listing.CategoryName,
		&listing.PackName, &listing.PackDescription, &listing.SourceName, &listing.AuthorName,
		&listing.ShareMode, &listing.CreditsPrice, &listing.ValidDays, &listing.BillingCycle, &listing.DownloadCount, &listing.Status, &metaInfoReadBack, &listing.CreatedAt)
	if err != nil {
		log.Printf("Failed to read back listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if metaInfoReadBack.Valid && metaInfoReadBack.String != "" {
		listing.MetaInfo = json.RawMessage(metaInfoReadBack.String)
	} else {
		listing.MetaInfo = json.RawMessage("{}")
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
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.valid_days, pl.billing_cycle, pl.download_count, pl.meta_info, pl.created_at
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
		var desc, sourceName, authorName, metaInfoStr sql.NullString
		if err := rows.Scan(&l.ID, &l.UserID, &l.CategoryID, &l.CategoryName,
			&l.PackName, &desc, &sourceName, &authorName,
			&l.ShareMode, &l.CreditsPrice, &l.ValidDays, &l.BillingCycle, &l.DownloadCount, &metaInfoStr, &l.CreatedAt); err != nil {
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
		if metaInfoStr.Valid && metaInfoStr.String != "" {
			l.MetaInfo = json.RawMessage(metaInfoStr.String)
		} else {
			l.MetaInfo = json.RawMessage("{}")
		}
		listings = append(listings, l)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"packs": listings})
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
	var validDays int
	var billingCycle string
	var fileData []byte
	var packName string
	var metaInfoStr sql.NullString
	err = db.QueryRow(
		`SELECT share_mode, credits_price, valid_days, billing_cycle, file_data, pack_name, meta_info FROM pack_listings WHERE id = ? AND status = 'published'`,
		packID,
	).Scan(&shareMode, &creditsPrice, &validDays, &billingCycle, &fileData, &packName, &metaInfoStr)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Handle billing based on pricing model
	switch shareMode {
	case "free":
		// Free pack: just increment download count, no credits deduction
		_, err = db.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
		if err != nil {
			log.Printf("Failed to increment download count: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

	case "per_use", "time_limited", "subscription":
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
				"error":    "INSUFFICIENT_CREDITS",
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
				"error":    "INSUFFICIENT_CREDITS",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		// Record credits transaction
		_, err = tx.Exec(
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
			 VALUES (?, 'download', ?, ?, ?)`,
			userID, -float64(creditsPrice), packID, fmt.Sprintf("Download pack: %s", packName),
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

		// Build X-Usage-License header based on pricing model
		usageLicense := map[string]interface{}{
			"listing_id":    packID,
			"pricing_model": shareMode,
			"remaining_uses": 0,
			"expires_at":    "",
			"billing_cycle": "",
		}

		now := time.Now().UTC()
		switch shareMode {
		case "per_use":
			usageLicense["remaining_uses"] = 1
		case "time_limited":
			expiresAt := now.AddDate(0, 0, validDays)
			usageLicense["expires_at"] = expiresAt.Format(time.RFC3339)
		case "subscription":
			var cycleDays int
			switch billingCycle {
			case "monthly":
				cycleDays = 30
			case "yearly":
				cycleDays = 365
			}
			expiresAt := now.AddDate(0, 0, cycleDays)
			usageLicense["expires_at"] = expiresAt.Format(time.RFC3339)
			usageLicense["billing_cycle"] = billingCycle
		}

		licenseJSON, err := json.Marshal(usageLicense)
		if err != nil {
			log.Printf("Failed to marshal usage license: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		w.Header().Set("X-Usage-License", string(licenseJSON))

	default:
		// Legacy "paid" mode or unknown: treat as paid with basic deduction
		var balance float64
		err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
		if err != nil {
			log.Printf("Failed to query user balance: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		if balance < float64(creditsPrice) {
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "INSUFFICIENT_CREDITS",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Failed to begin transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		defer tx.Rollback()

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
			jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
				"error":    "INSUFFICIENT_CREDITS",
				"required": creditsPrice,
				"balance":  balance,
			})
			return
		}

		_, err = tx.Exec(
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
			 VALUES (?, 'download', ?, ?, ?)`,
			userID, -float64(creditsPrice), packID, fmt.Sprintf("Download pack: %s", packName),
		)
		if err != nil {
			log.Printf("Failed to record transaction: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

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
	}

	// Return file data as binary response with meta_info header
	metaInfoValue := "{}"
	if metaInfoStr.Valid && metaInfoStr.String != "" {
		metaInfoValue = metaInfoStr.String
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.qap"`, packName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
	w.Header().Set("X-Meta-Info", metaInfoValue)
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)
}

// handlePurchaseAdditionalUses handles POST /api/packs/{id}/purchase-uses
// Deducts credits for additional uses of a per_use pack.
func handlePurchaseAdditionalUses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract pack ID from URL: /api/packs/{id}/purchase-uses
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/purchase-uses")
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

	// Parse request body for quantity
	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default quantity is 1 if body is empty or invalid
		req.Quantity = 1
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	// Look up the pack listing
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		packID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Verify share_mode is per_use
	if shareMode != "per_use" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack is not per_use type"})
		return
	}

	totalCost := creditsPrice * req.Quantity

	// Check user's credits balance
	var balance float64
	err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		log.Printf("Failed to query user balance: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if balance < float64(totalCost) {
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":    "INSUFFICIENT_CREDITS",
			"required": totalCost,
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

	// Deduct credits
	result, err := tx.Exec(
		"UPDATE users SET credits_balance = credits_balance - ? WHERE id = ? AND credits_balance >= ?",
		totalCost, userID, totalCost,
	)
	if err != nil {
		log.Printf("Failed to deduct credits: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":    "INSUFFICIENT_CREDITS",
			"required": totalCost,
			"balance":  balance,
		})
		return
	}

	// Record credits transaction
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
		 VALUES (?, 'purchase_uses', ?, ?, ?)`,
		userID, -float64(totalCost), packID, fmt.Sprintf("Purchase %d additional uses: %s", req.Quantity, packName),
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

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"remaining_uses":  req.Quantity,
		"credits_deducted": totalCost,
	})
}

// handleRenewSubscription handles POST /api/packs/{id}/renew
// Deducts one billing cycle of credits and returns new expiration date.
func handleRenewSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Extract pack ID from URL: /api/packs/{id}/renew
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/renew")
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

	// Look up the pack listing
	var shareMode string
	var creditsPrice int
	var billingCycle string
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, billing_cycle, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		packID,
	).Scan(&shareMode, &creditsPrice, &billingCycle, &packName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Verify share_mode is subscription
	if shareMode != "subscription" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack is not subscription type"})
		return
	}

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
			"error":    "INSUFFICIENT_CREDITS",
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

	// Deduct credits
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
		jsonResponse(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":    "INSUFFICIENT_CREDITS",
			"required": creditsPrice,
			"balance":  balance,
		})
		return
	}

	// Calculate new expires_at based on billing_cycle
	now := time.Now().UTC()
	var cycleDays int
	switch billingCycle {
	case "monthly":
		cycleDays = 30
	case "yearly":
		cycleDays = 365
	default:
		cycleDays = 30 // fallback to monthly
	}
	expiresAt := now.AddDate(0, 0, cycleDays)

	// Record credits transaction
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
		 VALUES (?, 'renew', ?, ?, ?)`,
		userID, -float64(creditsPrice), packID, fmt.Sprintf("Renew subscription (%s): %s", billingCycle, packName),
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

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"expires_at":      expiresAt.Format(time.RFC3339),
		"credits_deducted": creditsPrice,
	})
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
		 FROM credits_transactions WHERE user_id = ? ORDER BY created_at DESC, id DESC`,
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
// handleAdminManagement dispatches GET and POST requests for /api/admin/admins.
func handleAdminManagement(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleAdminList(w, r)
	case http.MethodPost:
		handleCreateAdmin(w, r)
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleAdminList returns all admins with id, username, role, created_at.
// GET /api/admin/admins
func handleAdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, username, role, created_at FROM admin_credentials ORDER BY id")
	if err != nil {
		log.Printf("Failed to query admins: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	type adminInfo struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}

	var admins []adminInfo
	for rows.Next() {
		var a adminInfo
		if err := rows.Scan(&a.ID, &a.Username, &a.Role, &a.CreatedAt); err != nil {
			log.Printf("Failed to scan admin row: %v", err)
			continue
		}
		admins = append(admins, a)
	}
	if admins == nil {
		admins = []adminInfo{}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"admins": admins})
}

// handleCreateAdmin creates a new admin with role="regular".
// POST /api/admin/admins
func handleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	if len(req.Username) < 3 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "username must be at least 3 characters"})
		return
	}
	if len(req.Password) < 6 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}

	// Check username uniqueness
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM admin_credentials WHERE username = ?", req.Username).Scan(&count)
	if err != nil {
		log.Printf("Failed to check username: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if count > 0 {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "username_already_exists"})
		return
	}

	passwordHash := hashPassword(req.Password)
	result, err := db.Exec("INSERT INTO admin_credentials (username, password_hash, role) VALUES (?, ?, 'regular')", req.Username, passwordHash)
	if err != nil {
		log.Printf("Failed to create admin: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	newID, _ := result.LastInsertId()
	var createdAt string
	db.QueryRow("SELECT created_at FROM admin_credentials WHERE id = ?", newID).Scan(&createdAt)

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":         newID,
		"username":   req.Username,
		"role":       "regular",
		"created_at": createdAt,
	})
}

// handleUpdateProfile allows an admin to update their own username and/or password.
// PUT /api/admin/profile
func handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil || adminID == 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Username    string `json:"username"`
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	// Get current admin info
	var currentUsername, currentPasswordHash string
	err = db.QueryRow("SELECT username, password_hash FROM admin_credentials WHERE id = ?", adminID).Scan(&currentUsername, &currentPasswordHash)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Update username if provided and different
	if req.Username != "" && req.Username != currentUsername {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM admin_credentials WHERE username = ? AND id != ?", req.Username, adminID).Scan(&count)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if count > 0 {
			jsonResponse(w, http.StatusConflict, map[string]string{"error": "username_already_exists"})
			return
		}
		_, err = db.Exec("UPDATE admin_credentials SET username = ? WHERE id = ?", req.Username, adminID)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	// Update password if new_password provided
	if req.NewPassword != "" {
		if !checkPassword(req.OldPassword, currentPasswordHash) {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_old_password"})
			return
		}
		newHash := hashPassword(req.NewPassword)
		_, err = db.Exec("UPDATE admin_credentials SET password_hash = ? WHERE id = ?", newHash, adminID)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handlePendingList returns all pack listings with status='pending'.
// GET /api/admin/review/pending
func handlePendingList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.valid_days, pl.billing_cycle, pl.download_count, pl.status, pl.meta_info, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.status = 'pending'
		ORDER BY pl.created_at ASC`)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	var listings []PackListingInfo
	for rows.Next() {
		var p PackListingInfo
		var categoryName, desc, sourceName, authorName, metaInfoStr sql.NullString
		err := rows.Scan(&p.ID, &p.UserID, &p.CategoryID, &categoryName, &p.PackName, &desc,
			&sourceName, &authorName, &p.ShareMode, &p.CreditsPrice, &p.ValidDays, &p.BillingCycle, &p.DownloadCount, &p.Status, &metaInfoStr, &p.CreatedAt)
		if err != nil {
			log.Printf("Failed to scan pending listing: %v", err)
			continue
		}
		if categoryName.Valid {
			p.CategoryName = categoryName.String
		}
		if desc.Valid {
			p.PackDescription = desc.String
		}
		if sourceName.Valid {
			p.SourceName = sourceName.String
		}
		if authorName.Valid {
			p.AuthorName = authorName.String
		}
		if metaInfoStr.Valid && metaInfoStr.String != "" {
			p.MetaInfo = json.RawMessage(metaInfoStr.String)
		} else {
			p.MetaInfo = json.RawMessage("{}")
		}
		listings = append(listings, p)
	}
	if listings == nil {
		listings = []PackListingInfo{}
	}
	jsonResponse(w, http.StatusOK, listings)
}

// handleApproveReview approves a pending pack listing.
// POST /api/admin/review/{id}/approve
func handleApproveReview(w http.ResponseWriter, r *http.Request, listingID int64) {
	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	// Check current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "pending" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "invalid_review_status"})
		return
	}

	_, err = db.Exec("UPDATE pack_listings SET status='published', reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'",
		adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleRejectReview rejects a pending pack listing with a reason.
// POST /api/admin/review/{id}/reject
func handleRejectReview(w http.ResponseWriter, r *http.Request, listingID int64) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if strings.TrimSpace(body.Reason) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "reject_reason_required"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	// Check current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "pending" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "invalid_review_status"})
		return
	}

	_, err = db.Exec("UPDATE pack_listings SET status='rejected', reject_reason=?, reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'",
		body.Reason, adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReviewRoutes dispatches review API requests.
func handleReviewRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/review/")
	if path == "pending" && r.Method == http.MethodGet {
		handlePendingList(w, r)
		return
	}
	// Parse: {id}/approve or {id}/reject
	parts := strings.Split(path, "/")
	if len(parts) == 2 {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
			return
		}
		switch parts[1] {
		case "approve":
			if r.Method == http.MethodPost {
				handleApproveReview(w, r, id)
				return
			}
		case "reject":
			if r.Method == http.MethodPost {
				handleRejectReview(w, r, id)
				return
			}
		}
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
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

	// Get admin info from session
	adminID := getSessionAdminID(r)
	role := getAdminRole(adminID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	adminTmpl.Execute(w, map[string]interface{}{
		"InitialCredits": initialCredits,
		"Role":           role,
		"AdminID":        adminID,
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
	port := flag.Int("port", 8088, "Server port")
	dbPath := flag.String("db", "marketplace.db", "SQLite database path")
	flag.Parse()

	var err error
	db, err = initDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Start background goroutine to clean up expired sessions and captchas
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			sessionsMu.Lock()
			for id, entry := range sessions {
				if now.After(entry.Expiry) {
					delete(sessions, id)
				}
			}
			sessionsMu.Unlock()
			captchasMu.Lock()
			for id, entry := range captchas {
				if now.After(entry.Expiry) {
					delete(captchas, id)
				}
			}
			captchasMu.Unlock()
		}
	}()

	// Auth routes
	http.HandleFunc("/api/auth/sn-login", handleSNLogin)
	http.HandleFunc("/api/auth/oauth", handleOAuthCallback) // kept for backward compatibility

	// Category routes (listing is public, admin requires auth)
	http.HandleFunc("/api/categories", handleListCategories)
	http.HandleFunc("/api/admin/categories", superAdminAuth(handleAdminCategories))
	http.HandleFunc("/api/admin/categories/", superAdminAuth(handleAdminCategories))

	// Pack routes (upload and download require auth, listing is public)
	http.HandleFunc("/api/packs/upload", authMiddleware(handleUploadPack))
	http.HandleFunc("/api/packs", handleListPacks)
	http.HandleFunc("/api/packs/", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Dispatch based on URL suffix
		switch {
		case strings.HasSuffix(r.URL.Path, "/purchase-uses"):
			handlePurchaseAdditionalUses(w, r)
		case strings.HasSuffix(r.URL.Path, "/renew"):
			handleRenewSubscription(w, r)
		default:
			handleDownloadPack(w, r)
		}
	}))

	// Credits routes (all require auth)
	http.HandleFunc("/api/credits/balance", authMiddleware(handleGetBalance))
	http.HandleFunc("/api/credits/purchase", authMiddleware(handlePurchaseCredits))
	http.HandleFunc("/api/credits/transactions", authMiddleware(handleListTransactions))

	// Admin auth routes (public)
	http.HandleFunc("/admin/setup", handleAdminSetup)
	http.HandleFunc("/admin/login", handleAdminLogin)
	http.HandleFunc("/admin/logout", handleAdminLogout)
	http.HandleFunc("/admin/captcha", handleCaptchaImage)
	http.HandleFunc("/admin/captcha/refresh", handleCaptchaRefresh)

	// Admin management API routes
	http.HandleFunc("/api/admin/admins", superAdminAuth(handleAdminManagement))
	http.HandleFunc("/api/admin/profile", adminAuth(handleUpdateProfile))

	// Review API routes
	http.HandleFunc("/api/admin/review/", adminAuth(handleReviewRoutes))

	// Admin routes (protected by session auth)
	http.HandleFunc("/admin/settings/initial-credits", superAdminAuth(handleSetInitialCredits))
	http.HandleFunc("/admin/", adminAuth(handleAdminDashboard))

	// Root redirect to admin
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/admin/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Marketplace server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
