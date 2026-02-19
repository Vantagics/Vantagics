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
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"marketplace_server/i18n"
	"marketplace_server/templates"
	"github.com/xuri/excelize/v2"

	"golang.org/x/crypto/bcrypt"
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
	DownloadCount   int              `json:"download_count"`
	Status          string           `json:"status"`
	RejectReason    string           `json:"reject_reason,omitempty"`
	ReviewedBy      *int64           `json:"reviewed_by,omitempty"`
	ReviewedAt      string           `json:"reviewed_at,omitempty"`
	MetaInfo        json.RawMessage  `json:"meta_info"`
	CreatedAt       string           `json:"created_at"`
	Purchased       bool             `json:"purchased"`
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

// PaymentInfo represents a user's payment receiving information.
type PaymentInfo struct {
	PaymentType    string          `json:"payment_type"`
	PaymentDetails json.RawMessage `json:"payment_details"`
}

// WithdrawalRequest represents a withdrawal request with fee calculation.
type WithdrawalRequest struct {
	ID             int64   `json:"id"`
	UserID         int64   `json:"user_id"`
	DisplayName    string  `json:"display_name"`
	CreditsAmount  float64 `json:"credits_amount"`
	CashRate       float64 `json:"cash_rate"`
	CashAmount     float64 `json:"cash_amount"`
	PaymentType    string  `json:"payment_type"`
	PaymentDetails string  `json:"payment_details"`
	FeeRate        float64 `json:"fee_rate"`
	FeeAmount      float64 `json:"fee_amount"`
	NetAmount      float64 `json:"net_amount"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
}

// validPaymentTypes is the set of allowed payment type values.
var validPaymentTypes = map[string]bool{
	"paypal":        true,
	"bank_card":     true,
	"wechat":        true,
	"alipay":        true,
	"check":         true,
	"wire_transfer": true,
	"bank_card_us":  true,
	"bank_card_eu":  true,
	"bank_card_cn":  true,
}

// requiredFieldsByPaymentType maps each payment type to its required detail fields.
var requiredFieldsByPaymentType = map[string][]string{
	"paypal":        {"account", "username"},
	"bank_card":     {"bank_name", "card_number", "account_holder"},
	"wechat":        {"account", "username"},
	"alipay":        {"account", "username"},
	"check":         {"full_legal_name", "province", "city", "district", "street_address", "postal_code", "phone"},
	"wire_transfer": {"beneficiary_name", "beneficiary_address", "bank_name", "swift_code", "account_number"},
	"bank_card_us":  {"legal_name", "routing_number", "account_number", "account_type"},
	"bank_card_eu":  {"legal_name", "iban", "bic_swift"},
	"bank_card_cn":  {"real_name", "card_number", "bank_branch"},
}

// validatePaymentInfo validates a payment_type and its corresponding payment_details JSON.
// Returns an error message string if validation fails, or empty string if valid.
func validatePaymentInfo(paymentType string, paymentDetailsJSON json.RawMessage) string {
	if !validPaymentTypes[paymentType] {
		return "invalid payment_type: must be one of paypal, bank_card, wechat, alipay, check, wire_transfer, bank_card_us, bank_card_eu, bank_card_cn"
	}

	var details map[string]string
	if err := json.Unmarshal(paymentDetailsJSON, &details); err != nil {
		return "invalid payment_details: must be a JSON object with string values"
	}

	requiredFields := requiredFieldsByPaymentType[paymentType]
	for _, field := range requiredFields {
		val, ok := details[field]
		if !ok || strings.TrimSpace(val) == "" {
			return fmt.Sprintf("missing or empty required field: %s", field)
		}
	}

	return ""
}


// handleGetPaymentInfo handles GET /user/payment-info.
// Returns the current user's payment receiving information as JSON.
func handleGetPaymentInfo(w http.ResponseWriter, r *http.Request) {
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

	var paymentType string
	var paymentDetailsStr string
	err = db.QueryRow("SELECT payment_type, payment_details FROM user_payment_info WHERE user_id = ?", userID).Scan(&paymentType, &paymentDetailsStr)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"payment_type":    "",
			"payment_details": map[string]interface{}{},
		})
		return
	}
	if err != nil {
		log.Printf("[PAYMENT-INFO] failed to query payment info for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}

	var paymentDetails json.RawMessage
	if err := json.Unmarshal([]byte(paymentDetailsStr), &paymentDetails); err != nil {
		paymentDetails = json.RawMessage(`{}`)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"payment_type":    paymentType,
		"payment_details": paymentDetails,
	})
}

// handleSavePaymentInfo handles POST /user/payment-info.
// Validates and saves the user's payment receiving information.
func handleSavePaymentInfo(w http.ResponseWriter, r *http.Request) {
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

	var info PaymentInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if errMsg := validatePaymentInfo(info.PaymentType, info.PaymentDetails); errMsg != "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	detailsStr := string(info.PaymentDetails)
	_, err = db.Exec(
		`INSERT INTO user_payment_info (user_id, payment_type, payment_details, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id) DO UPDATE SET payment_type=excluded.payment_type, payment_details=excluded.payment_details, updated_at=CURRENT_TIMESTAMP`,
		userID, info.PaymentType, detailsStr,
	)
	if err != nil {
		log.Printf("[PAYMENT-INFO] failed to save payment info for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}

	log.Printf("[PAYMENT-INFO] user %d saved payment info: type=%s", userID, info.PaymentType)
	jsonResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleGetPaymentFeeRate handles GET /user/payment-info/fee-rate?type=paypal
// Returns the fee rate for the given payment type from settings.
func handleGetPaymentFeeRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	paymentType := r.URL.Query().Get("type")
	if !validPaymentTypes[paymentType] {
		jsonResponse(w, http.StatusOK, map[string]float64{"fee_rate": 0})
		return
	}
	feeRateStr := getSetting("fee_rate_" + paymentType)
	feeRate, _ := strconv.ParseFloat(feeRateStr, 64)
	if feeRate < 0 {
		feeRate = 0
	}
	jsonResponse(w, http.StatusOK, map[string]float64{"fee_rate": feeRate})
}

// handleGetAllPaymentFeeRates handles GET /user/payment-info/fee-rates
// Returns the fee rates for all payment types from settings.
func handleGetAllPaymentFeeRates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	allTypes := []string{"paypal", "wechat", "alipay", "check", "wire_transfer", "bank_card_us", "bank_card_eu", "bank_card_cn"}
	rates := make(map[string]float64, len(allTypes))
	for _, pt := range allTypes {
		feeRateStr := getSetting("fee_rate_" + pt)
		feeRate, _ := strconv.ParseFloat(feeRateStr, 64)
		if feeRate < 0 {
			feeRate = 0
		}
		rates[pt] = feeRate
	}
	jsonResponse(w, http.StatusOK, rates)
}

// initDB initializes the SQLite database with WAL mode and creates all required tables.
func initDB(dbPath string) (*sql.DB, error) {
	// Use _pragma parameters to ensure every connection from the pool gets the same settings.
	// WAL mode is database-level (persists), but busy_timeout/synchronous/cache_size are per-connection.
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-65536)"
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite concurrency notes:
	// - WAL allows concurrent readers while one writer is active
	// - busy_timeout(5000) makes writers wait up to 5s instead of failing on lock contention
	// - synchronous(NORMAL) is safe with WAL and reduces fsync overhead
	// - Small pool avoids excessive lock contention on the single database file
	database.SetMaxOpenConns(4)
	database.SetMaxIdleConns(4)
	database.SetConnMaxLifetime(0) // reuse connections indefinitely

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

	// Add encryption_password column for paid pack encryption (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN encryption_password TEXT DEFAULT ''")

	// Add version column for pack replacement tracking (ignore error if already exists)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN version INTEGER DEFAULT 1")

	// Add share_token column for public URLs (prevents sequential ID enumeration)
	database.Exec("ALTER TABLE pack_listings ADD COLUMN share_token TEXT")
	database.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_pack_listings_share_token ON pack_listings(share_token) WHERE share_token IS NOT NULL")
	// Backfill share_token for existing rows that don't have one
	backfillShareTokens(database)

	// Add username and password_hash columns to users table (ignore error if already exists)
	database.Exec("ALTER TABLE users ADD COLUMN username TEXT")
	database.Exec("ALTER TABLE users ADD COLUMN password_hash TEXT")

	// Add is_blocked column to users table (ignore error if already exists)
	database.Exec("ALTER TABLE users ADD COLUMN is_blocked INTEGER DEFAULT 0")
	// Create unique index on username (ALTER TABLE ADD COLUMN does not support UNIQUE in SQLite)
	database.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE username IS NOT NULL")

	// Create user_downloads table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS user_downloads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create user_downloads table: %w", err)
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

	// Add ip_address column to credits_transactions for sales tracking (ignore error if already exists)
	database.Exec("ALTER TABLE credits_transactions ADD COLUMN ip_address TEXT DEFAULT ''")

	// Add ip_address column to user_downloads for buyer region tracking (ignore error if already exists)
	database.Exec("ALTER TABLE user_downloads ADD COLUMN ip_address TEXT DEFAULT ''")

	// Create user_purchased_packs table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS user_purchased_packs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			is_hidden INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id),
			UNIQUE(user_id, listing_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create user_purchased_packs table: %w", err)
	}

	// Create pack_usage_records table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS pack_usage_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			used_count INTEGER NOT NULL DEFAULT 0,
			total_purchased INTEGER NOT NULL DEFAULT 0,
			last_used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (listing_id) REFERENCES pack_listings(id),
			UNIQUE(user_id, listing_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create pack_usage_records table: %w", err)
	}

	// Create pack_usage_log table (deduplication via UNIQUE constraint)
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS pack_usage_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			listing_id INTEGER NOT NULL,
			used_at TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, listing_id, used_at)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create pack_usage_log table: %w", err)
	}

	// Create withdrawal_records table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS withdrawal_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			credits_amount REAL NOT NULL,
			cash_rate REAL NOT NULL,
			cash_amount REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create withdrawal_records table: %w", err)
	}

	// Create user_payment_info table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS user_payment_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			payment_type TEXT NOT NULL,
			payment_details TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create user_payment_info table: %w", err)
	}

	// Add payment/fee/status columns to withdrawal_records (ignore error if already exists)
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN payment_type TEXT DEFAULT ''")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN payment_details TEXT DEFAULT '{}'")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN fee_rate REAL DEFAULT 0")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN fee_amount REAL DEFAULT 0")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN net_amount REAL DEFAULT 0")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN status TEXT DEFAULT 'paid'")
	database.Exec("ALTER TABLE withdrawal_records ADD COLUMN display_name TEXT DEFAULT ''")

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

	// Insert default credit_cash_rate setting (ignore if already exists)
	_, err = database.Exec(
		"INSERT OR IGNORE INTO settings (key, value) VALUES ('credit_cash_rate', '0')",
	)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to insert credit_cash_rate setting: %w", err)
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

	// Add permissions column to admin_credentials (ignore error if already exists)
	database.Exec("ALTER TABLE admin_credentials ADD COLUMN permissions TEXT DEFAULT ''")

	// Create notifications table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			target_type TEXT NOT NULL DEFAULT 'broadcast',
			effective_date DATETIME NOT NULL,
			display_duration_days INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES admin_credentials(id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create notifications table: %w", err)
	}

	// Create notification_targets table
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS notification_targets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			notification_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			FOREIGN KEY (notification_id) REFERENCES notifications(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(notification_id, user_id)
		)
	`); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create notification_targets table: %w", err)
	}

	return database, nil
}

// isNotificationVisible determines whether a notification should be visible
// based on its effective date, display duration, and the current time.
// - If now is before effectiveDate, the notification is not yet active.
// - If durationDays is 0, the notification is permanent (always visible once active).
// - If durationDays > 0, the notification expires after effectiveDate + durationDays.
func isNotificationVisible(effectiveDate time.Time, durationDays int, now time.Time) bool {
	if now.Before(effectiveDate) {
		return false
	}
	if durationDays == 0 {
		return true
	}
	expiryDate := effectiveDate.AddDate(0, 0, durationDays)
	return now.Before(expiryDate)
}

// handleAdminCreateNotification handles POST /api/admin/notifications.
// It creates a new notification (broadcast or targeted).
func handleAdminCreateNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Title               string  `json:"title"`
		Content             string  `json:"content"`
		TargetType          string  `json:"target_type"`
		TargetUserIDs       []int64 `json:"target_user_ids"`
		EffectiveDate       string  `json:"effective_date"`
		DisplayDurationDays int     `json:"display_duration_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "title and content are required"})
		return
	}

	if req.TargetType == "targeted" && len(req.TargetUserIDs) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "target_user_ids required for targeted messages"})
		return
	}

	if req.TargetType == "" {
		req.TargetType = "broadcast"
	}

	var effectiveDate time.Time
	if req.EffectiveDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.EffectiveDate)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid effective_date format"})
			return
		}
		effectiveDate = parsed
	}
	if effectiveDate.IsZero() {
		effectiveDate = time.Now()
	}

	adminID := getSessionAdminID(r)

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO notifications (title, content, target_type, effective_date, display_duration_days, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		req.Title, req.Content, req.TargetType, effectiveDate.Format(time.RFC3339), req.DisplayDurationDays, adminID,
	)
	if err != nil {
		log.Printf("Failed to insert notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	notificationID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get notification ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if req.TargetType == "targeted" {
		for _, userID := range req.TargetUserIDs {
			_, err := tx.Exec(`INSERT OR IGNORE INTO notification_targets (notification_id, user_id) VALUES (?, ?)`,
				notificationID, userID)
			if err != nil {
				log.Printf("Failed to insert notification target: %v", err)
				jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]interface{}{"id": notificationID})
}

// AdminNotificationInfo is the response struct for admin notification list API.
type AdminNotificationInfo struct {
	ID                  int64  `json:"id"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	TargetType          string `json:"target_type"`
	EffectiveDate       string `json:"effective_date"`
	DisplayDurationDays int    `json:"display_duration_days"`
	Status              string `json:"status"`
	CreatedBy           int64  `json:"created_by"`
	CreatedAt           string `json:"created_at"`
	TargetCount         int    `json:"target_count"`
}

// handleAdminListNotifications handles GET /api/admin/notifications.
// It returns all non-deleted notifications ordered by created_at DESC.
// For targeted notifications, it includes the target_count field.
func handleAdminListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := db.Query(`
		SELECT id, title, content, target_type, effective_date, display_duration_days, status, created_by, created_at
		FROM notifications
		WHERE status != 'deleted'
		ORDER BY created_at DESC`)
	if err != nil {
		log.Printf("Failed to query notifications: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	var notifications []AdminNotificationInfo
	for rows.Next() {
		var n AdminNotificationInfo
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.TargetType, &n.EffectiveDate, &n.DisplayDurationDays, &n.Status, &n.CreatedBy, &n.CreatedAt); err != nil {
			log.Printf("Failed to scan notification: %v", err)
			continue
		}
		if n.TargetType == "targeted" {
			db.QueryRow("SELECT COUNT(*) FROM notification_targets WHERE notification_id = ?", n.ID).Scan(&n.TargetCount)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminListNotifications] rows iteration error: %v", err)
	}

	if notifications == nil {
		notifications = []AdminNotificationInfo{}
	}

	jsonResponse(w, http.StatusOK, notifications)
}

// handleAdminDisableNotification handles POST /api/admin/notifications/{id}/disable.
// It updates the notification status to "disabled".
func handleAdminDisableNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse notification ID from URL: /api/admin/notifications/{id}/disable
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "disable" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	notificationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM notifications WHERE id = ?", notificationID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("UPDATE notifications SET status='disabled', updated_at=CURRENT_TIMESTAMP WHERE id=?", notificationID)
	if err != nil {
		log.Printf("Failed to disable notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminEnableNotification handles POST /api/admin/notifications/{id}/enable.
// It restores the notification status from "disabled" to "active".
func handleAdminEnableNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse notification ID from URL: /api/admin/notifications/{id}/enable
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "enable" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	notificationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM notifications WHERE id = ?", notificationID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("UPDATE notifications SET status='active', updated_at=CURRENT_TIMESTAMP WHERE id=?", notificationID)
	if err != nil {
		log.Printf("Failed to enable notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleAdminDeleteNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse notification ID from URL: /api/admin/notifications/{id}/delete
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "delete" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	notificationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM notifications WHERE id = ?", notificationID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	_, err = db.Exec("UPDATE notifications SET status='deleted', updated_at=CURRENT_TIMESTAMP WHERE id=?", notificationID)
	if err != nil {
		log.Printf("Failed to delete notification: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleAdminNotificationRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/api/admin/notifications" {
		switch r.Method {
		case http.MethodGet:
			handleAdminListNotifications(w, r)
		case http.MethodPost:
			handleAdminCreateNotification(w, r)
		default:
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}
	// /api/admin/notifications/{id}/disable|enable|delete
	if strings.HasSuffix(path, "/disable") {
		handleAdminDisableNotification(w, r)
		return
	}
	if strings.HasSuffix(path, "/enable") {
		handleAdminEnableNotification(w, r)
		return
	}
	if strings.HasSuffix(path, "/delete") {
		handleAdminDeleteNotification(w, r)
		return
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// NotificationInfo is the response struct for user-facing notification query API.
type NotificationInfo struct {
	ID                  int64  `json:"id"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	TargetType          string `json:"target_type"`
	EffectiveDate       string `json:"effective_date"`
	DisplayDurationDays int    `json:"display_duration_days"`
	CreatedAt           string `json:"created_at"`
}

// handleListNotifications handles GET /api/notifications.
// It returns active, visible notifications for the current user.
// Authenticated users see broadcast + their targeted messages.
// Unauthenticated users see only broadcast messages.
func handleListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID := optionalUserID(r)
	now := time.Now()

	var rows *sql.Rows
	var err error

	if userID > 0 {
		// Authenticated: broadcast + targeted for this user
		rows, err = db.Query(`
			SELECT DISTINCT n.id, n.title, n.content, n.target_type, n.effective_date, n.display_duration_days, n.created_at
			FROM notifications n
			LEFT JOIN notification_targets nt ON n.id = nt.notification_id
			WHERE n.status = 'active'
			  AND n.effective_date <= ?
			  AND (n.target_type = 'broadcast' OR (n.target_type = 'targeted' AND nt.user_id = ?))
			ORDER BY n.created_at DESC`,
			now.Format(time.RFC3339), userID,
		)
	} else {
		// Unauthenticated: broadcast only
		rows, err = db.Query(`
			SELECT id, title, content, target_type, effective_date, display_duration_days, created_at
			FROM notifications
			WHERE status = 'active'
			  AND target_type = 'broadcast'
			  AND effective_date <= ?
			ORDER BY created_at DESC`,
			now.Format(time.RFC3339),
		)
	}

	if err != nil {
		log.Printf("Failed to query notifications: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	var notifications []NotificationInfo
	for rows.Next() {
		var n NotificationInfo
		var effectiveDateStr string
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.TargetType, &effectiveDateStr, &n.DisplayDurationDays, &n.CreatedAt); err != nil {
			log.Printf("Failed to scan notification: %v", err)
			continue
		}
		// Parse effective_date and apply visibility filter
		effectiveDate, err := time.Parse(time.RFC3339, effectiveDateStr)
		if err != nil {
			log.Printf("Failed to parse effective_date for notification %d: %v", n.ID, err)
			continue
		}
		if !isNotificationVisible(effectiveDate, n.DisplayDurationDays, now) {
			continue
		}
		n.EffectiveDate = effectiveDateStr
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleListNotifications] rows iteration error: %v", err)
	}

	if notifications == nil {
		notifications = []NotificationInfo{}
	}

	jsonResponse(w, http.StatusOK, notifications)
}

// upsertUserPurchasedPack inserts or updates a user_purchased_packs record,
// ensuring is_hidden is set to 0 (visible). If the record already exists
// (same user_id + listing_id), it updates is_hidden to 0 and refreshes updated_at.
func upsertUserPurchasedPack(userID int64, listingID int64) error {
	_, err := db.Exec(`
		INSERT INTO user_purchased_packs (user_id, listing_id, is_hidden, updated_at)
		VALUES (?, ?, 0, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, listing_id) DO UPDATE SET is_hidden = 0, updated_at = CURRENT_TIMESTAMP`,
		userID, listingID,
	)
	return err
}

// softDeleteUserPurchasedPack marks the specified (user_id, listing_id) record
// as hidden by setting is_hidden to 1 and refreshing updated_at.
func softDeleteUserPurchasedPack(userID int64, listingID int64) error {
	_, err := db.Exec(`
		UPDATE user_purchased_packs SET is_hidden = 1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND listing_id = ?`,
		userID, listingID,
	)
	return err
}


// generateShareToken creates a cryptographically random URL-safe token (22 chars, ~131 bits of entropy).
func generateShareToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp + random suffix (should never happen)
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// backfillShareTokens assigns share_token to any existing pack_listings rows that lack one.
func backfillShareTokens(database *sql.DB) {
	rows, err := database.Query("SELECT id FROM pack_listings WHERE share_token IS NULL OR share_token = ''")
	if err != nil {
		log.Printf("[BACKFILL] failed to query rows without share_token: %v", err)
		return
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		token := generateShareToken()
		if _, err := database.Exec("UPDATE pack_listings SET share_token = ? WHERE id = ?", token, id); err != nil {
			log.Printf("[BACKFILL] failed to set share_token for id=%d: %v", id, err)
		}
	}
	if len(ids) > 0 {
		log.Printf("[BACKFILL] assigned share_token to %d existing pack_listings", len(ids))
	}
}

// resolveShareToken looks up the listing_id for a given share_token.
func resolveShareToken(token string) (int64, error) {
	var listingID int64
	err := db.QueryRow("SELECT id FROM pack_listings WHERE share_token = ?", token).Scan(&listingID)
	return listingID, err
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
// MUST be set via MARKETPLACE_JWT_SECRET environment variable in production.
var jwtSecret = func() []byte {
	if s := os.Getenv("MARKETPLACE_JWT_SECRET"); s != "" {
		return []byte(s)
	}
	log.Println("[WARN] MARKETPLACE_JWT_SECRET not set, using insecure default. Set this in production!")
	return []byte("marketplace-server-jwt-secret-key-2024-dev-only")
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

// optionalUserID attempts to extract userID from the Authorization header.
// Returns 0 if the header is missing, malformed, or JWT parsing fails.
// This allows public endpoints to optionally identify logged-in users.
func optionalUserID(r *http.Request) int64 {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return 0
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	userID, _, err := parseJWT(token)
	if err != nil {
		return 0
	}
	return userID
}

// getUserPurchasedListingIDs queries user_downloads and credits_transactions
// to return the set of listing IDs that the given user has purchased or downloaded.
func getUserPurchasedListingIDs(userID int64) map[int64]bool {
	purchased := make(map[int64]bool)

	// Build a set of listing IDs that the user has explicitly hidden (soft-deleted)
	hiddenSet := make(map[int64]bool)
	hiddenRows, err := db.Query(
		"SELECT listing_id FROM user_purchased_packs WHERE user_id = ? AND is_hidden = 1",
		userID)
	if err == nil {
		defer hiddenRows.Close()
		for hiddenRows.Next() {
			var lid int64
			if hiddenRows.Scan(&lid) == nil {
				hiddenSet[lid] = true
			}
		}
		if err := hiddenRows.Err(); err != nil {
			log.Printf("[getUserPurchasedListingIDs] hiddenRows iteration error: %v", err)
		}
	}

	// Query user_downloads table
	rows, err := db.Query("SELECT listing_id FROM user_downloads WHERE user_id = ?", userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var lid int64
			if rows.Scan(&lid) == nil {
				if !hiddenSet[lid] {
					purchased[lid] = true
				}
			}
		}
		if err := rows.Err(); err != nil {
			log.Printf("[getUserPurchasedListingIDs] rows iteration error: %v", err)
		}
	}
	// Query credits_transactions table (purchase-type transactions)
	rows2, err := db.Query(
		"SELECT DISTINCT listing_id FROM credits_transactions WHERE user_id = ? AND listing_id IS NOT NULL",
		userID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var lid int64
			if rows2.Scan(&lid) == nil {
				if !hiddenSet[lid] {
					purchased[lid] = true
				}
			}
		}
		if err := rows2.Err(); err != nil {
			log.Printf("[getUserPurchasedListingIDs] rows2 iteration error: %v", err)
		}
	}
	return purchased
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

		// Check if user is blocked
		var isBlocked int
		if err := db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked); err == nil && isBlocked == 1 {
			jsonResponse(w, http.StatusForbidden, map[string]string{"error": "account_blocked", "message": "Your account has been blocked"})
			return
		}

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

// hashPassword hashes a password using bcrypt (cost 12).
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		// bcrypt only fails if password > 72 bytes or cost is invalid — both are programming errors.
		// Do NOT fall back to weaker hashing; fail loudly instead.
		log.Fatalf("[FATAL] bcrypt.GenerateFromPassword failed: %v", err)
	}
	return "bcrypt:" + string(hash)
}

// checkPassword verifies a password against a stored hash.
// Supports both bcrypt (new) and legacy SHA-256 (old) formats.
func checkPassword(password, stored string) bool {
	// New bcrypt format
	if strings.HasPrefix(stored, "bcrypt:") {
		hash := strings.TrimPrefix(stored, "bcrypt:")
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	// Legacy SHA-256 format for backward compatibility
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
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is a critical system issue; fail loudly
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
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

// User session store (independent from admin sessions)
var (
	userSessions   = make(map[string]userSessionEntry)
	userSessionsMu sync.RWMutex
)

type userSessionEntry struct {
	UserID int64
	Expiry time.Time
}

// Login ticket store for one-time ticket-based login (SSO from desktop client)
var (
	loginTickets   = make(map[string]loginTicketEntry)
	loginTicketsMu sync.RWMutex
)

type loginTicketEntry struct {
	UserID int64
	Expiry time.Time
}

// createLoginTicket creates a one-time login ticket for the given user ID.
func createLoginTicket(userID int64) string {
	id := generateSessionID()
	loginTicketsMu.Lock()
	loginTickets[id] = loginTicketEntry{UserID: userID, Expiry: time.Now().Add(5 * time.Minute)}
	loginTicketsMu.Unlock()
	return id
}

// consumeLoginTicket validates and consumes a one-time login ticket. Returns userID or 0.
func consumeLoginTicket(ticket string) int64 {
	loginTicketsMu.Lock()
	defer loginTicketsMu.Unlock()
	entry, ok := loginTickets[ticket]
	if !ok || time.Now().After(entry.Expiry) {
		delete(loginTickets, ticket)
		return 0
	}
	delete(loginTickets, ticket)
	return entry.UserID
}

// createUserSession creates a new user session and returns the session ID.
func createUserSession(userID int64) string {
	id := generateSessionID()
	userSessionsMu.Lock()
	userSessions[id] = userSessionEntry{UserID: userID, Expiry: time.Now().Add(24 * time.Hour)}
	userSessionsMu.Unlock()
	return id
}

// isValidUserSession checks if a user session ID is valid and not expired.
func isValidUserSession(id string) bool {
	userSessionsMu.RLock()
	entry, ok := userSessions[id]
	userSessionsMu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(entry.Expiry) {
		userSessionsMu.Lock()
		delete(userSessions, id)
		userSessionsMu.Unlock()
		return false
	}
	return true
}

// getUserSessionUserID returns the user ID for a valid user session, or 0 if invalid.
func getUserSessionUserID(id string) int64 {
	userSessionsMu.RLock()
	entry, ok := userSessions[id]
	userSessionsMu.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		return 0
	}
	return entry.UserID
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

// allPermissions is the complete list of assignable permission keys.
var allPermissions = []string{"categories", "marketplace", "authors", "review", "settings", "customers", "sales", "notifications"}

// getAdminPermissions returns the permission list for the given admin ID.
// id=1 always gets all permissions. Others get what's stored in the DB.
func getAdminPermissions(adminID int64) []string {
	if adminID == 1 {
		return allPermissions
	}
	var perms string
	err := db.QueryRow("SELECT COALESCE(permissions, '') FROM admin_credentials WHERE id = ?", adminID).Scan(&perms)
	if err != nil || perms == "" {
		return []string{}
	}
	return strings.Split(perms, ",")
}

// hasPermission checks if the given admin has a specific permission.
func hasPermission(adminID int64, perm string) bool {
	if adminID == 1 {
		return true
	}
	perms := getAdminPermissions(adminID)
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// permissionAuth creates a middleware that checks if the admin has the specified permission.
// id=1 always passes. Other admins must have the permission in their permissions list.
func permissionAuth(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
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
			if !hasPermission(adminID, permission) {
				jsonResponse(w, http.StatusForbidden, map[string]string{"error": "permission_denied"})
				return
			}
			r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
			next(w, r)
		}
	}
}

// superAdminOnlyAuth is a middleware that only allows admin with id=1.
func superAdminOnlyAuth(next http.HandlerFunc) http.HandlerFunc {
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
		if adminID != 1 {
			jsonResponse(w, http.StatusForbidden, map[string]string{"error": "permission_denied"})
			return
		}
		r.Header.Set("X-Admin-ID", strconv.FormatInt(adminID, 10))
		next(w, r)
	}
}

// userAuth is the authentication middleware for the user portal.
// It reads the "user_session" cookie, validates the session, and sets X-User-ID header.
// Invalid or missing sessions are redirected to /user/login.
func userAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_session")
		if err != nil || !isValidUserSession(cookie.Value) {
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return
		}
		userID := getUserSessionUserID(cookie.Value)

		// Check if user is blocked
		var isBlocked int
		if err := db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked); err == nil && isBlocked == 1 {
			// Clear session and redirect to login with error
			http.SetCookie(w, makeSessionCookie("user_session", "", -1))
			http.Redirect(w, r, "/user/login?error=blocked", http.StatusFound)
			return
		}

		r.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		next(w, r)
	}
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
	// Also clean up math captcha expression if present
	mathCaptchaExpressionsMu.Lock()
	delete(mathCaptchaExpressions, id)
	mathCaptchaExpressionsMu.Unlock()
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

// Math captcha expression store (in-memory, keyed by captcha ID)
var (
	mathCaptchaExpressions   = make(map[string]string) // captchaID -> expression string
	mathCaptchaExpressionsMu sync.RWMutex
)

// generateMathCaptcha generates a math captcha with two operands (1-20) and + or -.
// Subtraction ensures non-negative result. Returns (expression, answer).
func generateMathCaptcha() (string, string) {
	maxVal := big.NewInt(20)
	na, _ := rand.Int(rand.Reader, maxVal)
	nb, _ := rand.Int(rand.Reader, maxVal)
	a := int(na.Int64()) + 1 // 1-20
	b := int(nb.Int64()) + 1 // 1-20

	// Random operator: 0 = add, 1 = subtract
	nOp, _ := rand.Int(rand.Reader, big.NewInt(2))
	if nOp.Int64() == 0 {
		// Addition
		return fmt.Sprintf("%d + %d = ?", a, b), fmt.Sprintf("%d", a+b)
	}
	// Subtraction: ensure non-negative result
	if a < b {
		a, b = b, a
	}
	return fmt.Sprintf("%d - %d = ?", a, b), fmt.Sprintf("%d", a-b)
}

// createMathCaptcha generates a math captcha, stores the answer in captchas map
// and the expression in mathCaptchaExpressions map. Returns captcha ID.
func createMathCaptcha() string {
	id := generateSessionID()[:16]
	expression, answer := generateMathCaptcha()
	log.Printf("[MATH_CAPTCHA] created id=%s expr=%s answer=%s", id, expression, answer)
	captchasMu.Lock()
	captchas[id] = captchaEntry{Code: answer, Expiry: time.Now().Add(5 * time.Minute)}
	captchasMu.Unlock()
	mathCaptchaExpressionsMu.Lock()
	mathCaptchaExpressions[id] = expression
	mathCaptchaExpressionsMu.Unlock()
	return id
}

// getMathCaptchaExpression returns the expression string for a math captcha ID.
// Returns empty string if the captcha does not exist or has expired.
func getMathCaptchaExpression(id string) string {
	mathCaptchaExpressionsMu.RLock()
	expr, ok := mathCaptchaExpressions[id]
	mathCaptchaExpressionsMu.RUnlock()
	if !ok {
		return ""
	}
	// Also check expiry via the captchas map
	captchasMu.RLock()
	entry, exists := captchas[id]
	captchasMu.RUnlock()
	if !exists || time.Now().After(entry.Expiry) {
		return ""
	}
	return expr
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

// drawChar draws a single character on the image at position (xOff, yOff).
// Supports digits 0-9 and special characters: +, -, =, ?, space.
func drawChar(img *image.RGBA, ch byte, xOff, yOff int, c color.RGBA) {
	// 5x7 bitmap font for special characters
	specialFont := map[byte][]string{
		'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
		'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
		'=': {"00000", "00000", "11111", "00000", "11111", "00000", "00000"},
		'?': {"01110", "10001", "00001", "00110", "00100", "00000", "00100"},
		' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	}

	// Check if it's a digit — delegate to drawDigit
	if ch >= '0' && ch <= '9' {
		drawDigit(img, ch, xOff, yOff, c)
		return
	}

	pattern, ok := specialFont[ch]
	if !ok {
		return
	}
	scale := 4
	for row, line := range pattern {
		for col, bit := range line {
			if bit == '1' {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetRGBA(xOff+col*scale+dx, yOff+row*scale+dy, c)
					}
				}
			}
		}
	}
}

// generateMathCaptchaImage creates a PNG image for the given math captcha ID.
// It renders the math expression (e.g. "12 + 5 = ?") with noise lines and dots.
func generateMathCaptchaImage(id string) []byte {
	expr := getMathCaptchaExpression(id)
	if expr == "" {
		return nil
	}

	// Calculate image width based on expression length
	charWidth := 24 // 5 cols * 4 scale + 4 spacing
	padding := 15
	width := padding*2 + len(expr)*charWidth
	if width < 200 {
		width = 200
	}
	height := 50

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
	for i := 0; i < 150; i++ {
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

	// Draw each character of the expression
	charColors := []color.RGBA{
		{50, 50, 120, 255}, {120, 40, 40, 255}, {40, 100, 50, 255}, {100, 50, 120, 255},
	}
	xPos := padding
	yPos := 8
	colorIdx := 0
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		drawChar(img, ch, xPos, yPos, charColors[colorIdx%len(charColors)])
		if ch != ' ' {
			colorIdx++
		}
		xPos += charWidth
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
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
		Secure:   true,
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
	lang := i18n.DetectLang(r)
	if username == "" || len(username) < 3 {
		errMsg = i18n.T(lang, "admin_username_min3_err")
	} else if password == "" || len(password) < 6 {
		errMsg = i18n.T(lang, "admin_password_min6_err")
	} else if len(password) > 72 {
		errMsg = i18n.T(lang, "admin_password_max72_err")
	} else if password != password2 {
		errMsg = i18n.T(lang, "admin_password_mismatch_err")
	} else if !verifyCaptcha(captchaID, captchaAns) {
		errMsg = i18n.T(lang, "captcha_error_admin")
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
			"Error":     i18n.T(lang, "save_failed"),
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
	lang := i18n.DetectLang(r)
	var adminID int64
	if !verifyCaptcha(captchaID, captchaAns) {
		log.Printf("[LOGIN] captcha verification failed for ID=%q answer=%q", captchaID, captchaAns)
		errMsg = i18n.T(lang, "captcha_error_admin")
	} else {
		var storedHash string
		err := db.QueryRow("SELECT id, password_hash FROM admin_credentials WHERE username = ?", username).Scan(&adminID, &storedHash)
		if err != nil {
			log.Printf("[LOGIN] db query error for username=%q: %v", username, err)
			errMsg = i18n.T(lang, "admin_login_error")
		} else if !checkPassword(password, storedHash) {
			log.Printf("[LOGIN] password check failed for username=%q adminID=%d", username, adminID)
			errMsg = i18n.T(lang, "admin_login_error")
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

// handleUserLogin handles user login (GET: render form, POST: authenticate).
func handleUserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		redirect := r.URL.Query().Get("redirect")
		captchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
			"Redirect":  redirect,
		})
		templates.UserLoginTmpl.Execute(w, data)
		return
	}

	// POST
	lang := i18n.DetectLang(r)
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha_answer"))
	redirect := r.FormValue("redirect")

	log.Printf("[USER-LOGIN] attempt: username=%q, captchaID=%q", username, captchaID)

	errMsg := ""
	var userID int64
	if !verifyCaptcha(captchaID, captchaAns) {
		log.Printf("[USER-LOGIN] captcha verification failed for ID=%q", captchaID)
		errMsg = i18n.T(lang, "captcha_error")
	} else {
		var storedHash string
		err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&userID, &storedHash)
		if err != nil {
			log.Printf("[USER-LOGIN] db query error for username=%q: %v", username, err)
			errMsg = i18n.T(lang, "login_error")
		} else if !checkPassword(password, storedHash) {
			log.Printf("[USER-LOGIN] password check failed for username=%q", username)
			errMsg = i18n.T(lang, "login_error")
		} else {
			log.Printf("[USER-LOGIN] success for username=%q userID=%d", username, userID)
		}
	}

	if errMsg != "" {
		newCaptchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     errMsg,
			"Redirect":  redirect,
		})
		templates.UserLoginTmpl.Execute(w, data)
		return
	}

	sid := createUserSession(userID)
	http.SetCookie(w, makeSessionCookie("user_session", sid, 86400))

	// Redirect to the original page if redirect parameter starts with /pack/
	if strings.HasPrefix(redirect, "/pack/") {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}


// handleUserRegister handles GET/POST /user/register for SN+Email binding registration.
func handleUserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		redirect := r.URL.Query().Get("redirect")
		captchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": captchaID,
			"Error":     "",
			"Redirect":  redirect,
		})
		templates.UserRegisterTmpl.Execute(w, data)
		return
	}

	// POST
	lang := i18n.DetectLang(r)
	email := strings.TrimSpace(r.FormValue("email"))
	sn := strings.TrimSpace(r.FormValue("sn"))
	password := r.FormValue("password")
	password2 := r.FormValue("password2")
	captchaID := r.FormValue("captcha_id")
	captchaAns := strings.TrimSpace(r.FormValue("captcha_answer"))
	redirect := r.FormValue("redirect")

	log.Printf("[USER-REGISTER] attempt: email=%q, sn=%q, captchaID=%q", email, sn, captchaID)

	renderError := func(msg string) {
		newCaptchaID := createMathCaptcha()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"CaptchaID": newCaptchaID,
			"Error":     msg,
			"Redirect":  redirect,
		})
		templates.UserRegisterTmpl.Execute(w, data)
	}

	// Step 1: Verify captcha
	if !verifyCaptcha(captchaID, captchaAns) {
		log.Printf("[USER-REGISTER] captcha verification failed for ID=%q", captchaID)
		renderError(i18n.T(lang, "captcha_error"))
		return
	}

	// Step 2: Validate email format
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") || len(email) > 254 {
		renderError(i18n.T(lang, "invalid_email"))
		return
	}

	// Step 3: Verify password consistency and length
	if password != password2 {
		log.Printf("[USER-REGISTER] password mismatch for email=%q", email)
		renderError(i18n.T(lang, "password_mismatch"))
		return
	}
	if len(password) < 6 {
		log.Printf("[USER-REGISTER] password too short for email=%q", email)
		renderError(i18n.T(lang, "password_min_6"))
		return
	}
	if len(password) > 72 {
		log.Printf("[USER-REGISTER] password too long for email=%q", email)
		renderError(i18n.T(lang, "password_max_72"))
		return
	}

	// Step 4: Call License_Server /api/marketplace-auth to verify SN+Email
	authReqBody, err := json.Marshal(map[string]string{"sn": sn, "email": email})
	if err != nil {
		log.Printf("[USER-REGISTER] failed to marshal auth request: %v", err)
		renderError(i18n.T(lang, "system_error"))
		return
	}

	lsURL := getSetting("license_server_url")
	if lsURL == "" {
		lsURL = licenseServerURL
	}
	authURL := lsURL + "/api/marketplace-auth"
	httpResp, err := http.Post(authURL, "application/json", bytes.NewReader(authReqBody))
	if err != nil {
		log.Printf("[USER-REGISTER] failed to contact license server at %s: %v", authURL, err)
		renderError(i18n.T(lang, "license_server_error"))
		return
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Printf("[USER-REGISTER] failed to read license server response: %v", err)
		renderError(i18n.T(lang, "license_server_error"))
		return
	}

	var authResp struct {
		Success bool   `json:"success"`
		Token   string `json:"token,omitempty"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		log.Printf("[USER-REGISTER] failed to parse license server response: %v", err)
		renderError(i18n.T(lang, "license_server_error"))
		return
	}

	if !authResp.Success {
		msg := authResp.Message
		if msg == "" {
			msg = i18n.T(lang, "sn_email_verify_failed")
		}
		log.Printf("[USER-REGISTER] license server auth failed: code=%s msg=%s", authResp.Code, msg)
		renderError(msg)
		return
	}

	// Step 4: Check SN not already bound
	var existingID int64
	err = db.QueryRow("SELECT id FROM users WHERE auth_type='sn' AND auth_id=?", sn).Scan(&existingID)
	if err == nil {
		log.Printf("[USER-REGISTER] SN already bound: sn=%q existingUserID=%d", sn, existingID)
		renderError(i18n.T(lang, "sn_already_bound"))
		return
	} else if err != sql.ErrNoRows {
		log.Printf("[USER-REGISTER] db error checking SN binding: %v", err)
		renderError(i18n.T(lang, "system_error"))
		return
	}

	// Step 5: Create user (username = email prefix)
	username := email
	if idx := strings.Index(email, "@"); idx > 0 {
		username = email[:idx]
	}

	initialBalanceStr := getSetting("initial_credits_balance")
	var initialBalance float64
	if initialBalanceStr != "" {
		fmt.Sscanf(initialBalanceStr, "%f", &initialBalance)
	}

	result, err := db.Exec(
		"INSERT INTO users (auth_type, auth_id, display_name, email, username, password_hash, credits_balance) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"sn", sn, username, email, username, hashPassword(password), initialBalance,
	)
	if err != nil {
		log.Printf("[USER-REGISTER] failed to create user: %v", err)
		if strings.Contains(err.Error(), "UNIQUE") {
			renderError(i18n.T(lang, "username_already_exists"))
		} else {
			renderError(i18n.T(lang, "create_account_failed"))
		}
		return
	}

	userID, err := result.LastInsertId()
	if err != nil {
		log.Printf("[USER-REGISTER] failed to get last insert ID: %v", err)
		renderError(i18n.T(lang, "create_account_failed"))
		return
	}

	// Record initial credits transaction if balance > 0
	if initialBalance > 0 {
		_, err = db.Exec(
			"INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'initial', ?, 'Initial credits balance')",
			userID, initialBalance,
		)
		if err != nil {
			log.Printf("[USER-REGISTER] failed to record initial credits transaction: %v", err)
		}
	}

	log.Printf("[USER-REGISTER] success: email=%q sn=%q userID=%d username=%q", email, sn, userID, username)

	// Step 6: Create session and redirect
	sid := createUserSession(userID)
	http.SetCookie(w, makeSessionCookie("user_session", sid, 86400))

	// Redirect to the original page if redirect parameter starts with /pack/ (security: only allow /pack/ prefix)
	if strings.HasPrefix(redirect, "/pack/") {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

// PurchasedPackInfo holds info about a purchased/downloaded pack for the user dashboard.
type PurchasedPackInfo struct {
	ListingID      int64
	PackName       string
	CategoryName   string
	ShareMode      string
	CreditsPrice   int
	ValidDays      int
	PurchaseDate   string
	ExpiresAt      string
	UsedCount      int
	TotalPurchased int
	SourceName     string
	AuthorName     string
	DownloadCount  int
	Version        int
}

// BillingRecord holds a single billing/transaction record for the user billing page.
type BillingRecord struct {
	ID              int64
	TransactionType string
	Amount          float64
	PackName        string
	Description     string
	CreatedAt       string
}

// AuthorPackInfo holds info about an author's shared pack with sales data.
type AuthorPackInfo struct {
	ListingID    int64
	PackName     string
	PackDesc     string
	ShareMode    string
	CreditsPrice int
	Status       string
	SoldCount    int
	TotalRevenue float64
	Version      int
	ShareToken   string
}

// AuthorDashboardData holds all author panel data for the user dashboard.
type AuthorDashboardData struct {
	IsAuthor           bool
	AuthorPacks        []AuthorPackInfo
	TotalRevenue       float64
	TotalWithdrawn     float64
	UnwithdrawnCredits float64
	CreditCashRate     float64
	WithdrawalEnabled  bool
	RevenueSplitPct    float64
}

// TopPackInfo holds info about a top-ranked analysis pack for the TOP分析包 tab.
type TopPackInfo struct {
	Rank          int
	ListingID     int64
	PackName      string
	AuthorName    string
	CategoryName  string
	ShareMode     string
	CreditsPrice  int
	DownloadCount int
	TotalRevenue  float64
}

// handleUserDashboard renders the user dashboard page showing account info and purchased packs.
func handleUserDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[USER-DASHBOARD] invalid X-User-ID header: %q", userIDStr)
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Query user info
	var user MarketplaceUser
	err = db.QueryRow("SELECT id, email, credits_balance FROM users WHERE id = ?", userID).Scan(
		&user.ID, &user.Email, &user.CreditsBalance,
	)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query user id=%d: %v", userID, err)
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_data_failed"), http.StatusInternalServerError)
		return
	}

	// Query all purchased/downloaded packs using a UNION approach:
	// 1. From user_purchased_packs (canonical record)
	// 2. From credits_transactions (paid downloads)
	// 3. From user_downloads (all downloads including free)
	allRows, err := db.Query(`
		SELECT pl.id, pl.pack_name, pl.share_mode, pl.credits_price, COALESCE(pl.valid_days, 0),
		       COALESCE(c.name, '') as category_name, COALESCE(src.purchase_date, upp.created_at) as purchase_date,
		       COALESCE(pur.used_count, 0), COALESCE(pur.total_purchased, 0),
		       COALESCE(pl.source_name, ''), COALESCE(pl.author_name, ''), COALESCE(pl.download_count, 0),
		       COALESCE(pl.version, 1)
		FROM user_purchased_packs upp
		JOIN pack_listings pl ON upp.listing_id = pl.id
		LEFT JOIN categories c ON pl.category_id = c.id
		LEFT JOIN pack_usage_records pur ON pur.user_id = upp.user_id AND pur.listing_id = upp.listing_id
		LEFT JOIN (
		    SELECT user_id, listing_id, MIN(created_at) as purchase_date
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription')
		    GROUP BY user_id, listing_id
		    UNION ALL
		    SELECT user_id, listing_id, MIN(downloaded_at) as purchase_date
		    FROM user_downloads
		    GROUP BY user_id, listing_id
		) src ON src.user_id = upp.user_id AND src.listing_id = upp.listing_id
		WHERE upp.user_id = ? AND (upp.is_hidden IS NULL OR upp.is_hidden = 0)
		ORDER BY purchase_date DESC
	`, userID)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query purchased packs for user %d: %v", userID, err)
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_data_failed"), http.StatusInternalServerError)
		return
	}
	defer allRows.Close()

	// Use a map to deduplicate by listing_id
	seenListings := make(map[int64]bool)
	var packs []PurchasedPackInfo

	for allRows.Next() {
		var p PurchasedPackInfo
		var purchaseDateStr string
		if err := allRows.Scan(&p.ListingID, &p.PackName, &p.ShareMode, &p.CreditsPrice, &p.ValidDays, &p.CategoryName, &purchaseDateStr, &p.UsedCount, &p.TotalPurchased, &p.SourceName, &p.AuthorName, &p.DownloadCount, &p.Version); err != nil {
			log.Printf("[USER-DASHBOARD] failed to scan purchased pack row: %v", err)
			continue
		}
		if seenListings[p.ListingID] {
			continue
		}
		seenListings[p.ListingID] = true
		p.PurchaseDate = purchaseDateStr

		// Calculate ExpiresAt for time_limited and subscription packs
		// For subscription packs with valid_days=0, default to 30 days (1 month)
		effectiveDays := p.ValidDays
		if p.ShareMode == "subscription" && effectiveDays == 0 {
			effectiveDays = 30
		}
		if (p.ShareMode == "time_limited" || p.ShareMode == "subscription") && effectiveDays > 0 {
			if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr); err == nil {
				p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
			} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr); err == nil {
				p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
			}
		}

		packs = append(packs, p)
	}
	if err := allRows.Err(); err != nil {
		log.Printf("[handleUserDashboard] allRows iteration error: %v", err)
	}

	// Also query from credits_transactions and user_downloads for packs
	// that may not have a user_purchased_packs record yet (legacy data)
	legacyRows, err := db.Query(`
		SELECT DISTINCT pl.id, pl.pack_name, pl.share_mode, pl.credits_price, COALESCE(pl.valid_days, 0),
		       COALESCE(c.name, '') as category_name, src.purchase_date,
		       COALESCE(pur.used_count, 0), COALESCE(pur.total_purchased, 0),
		       COALESCE(pl.source_name, ''), COALESCE(pl.author_name, ''), COALESCE(pl.download_count, 0),
		       COALESCE(pl.version, 1)
		FROM (
		    SELECT user_id, listing_id, MIN(created_at) as purchase_date FROM credits_transactions
		    WHERE user_id = ? AND transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription') AND listing_id IS NOT NULL
		    GROUP BY user_id, listing_id
		    UNION ALL
		    SELECT user_id, listing_id, MIN(downloaded_at) as purchase_date FROM user_downloads
		    WHERE user_id = ?
		    GROUP BY user_id, listing_id
		) src
		JOIN pack_listings pl ON src.listing_id = pl.id
		LEFT JOIN categories c ON pl.category_id = c.id
		LEFT JOIN user_purchased_packs upp ON upp.user_id = src.user_id AND upp.listing_id = src.listing_id
		LEFT JOIN pack_usage_records pur ON pur.user_id = src.user_id AND pur.listing_id = src.listing_id
		WHERE (upp.id IS NULL OR (upp.is_hidden IS NULL OR upp.is_hidden = 0))
		ORDER BY src.purchase_date DESC
	`, userID, userID)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query legacy packs for user %d: %v", userID, err)
		// Non-fatal: continue with packs from user_purchased_packs
	} else {
		defer legacyRows.Close()
		for legacyRows.Next() {
			var p PurchasedPackInfo
			var purchaseDateStr string
			if err := legacyRows.Scan(&p.ListingID, &p.PackName, &p.ShareMode, &p.CreditsPrice, &p.ValidDays, &p.CategoryName, &purchaseDateStr, &p.UsedCount, &p.TotalPurchased, &p.SourceName, &p.AuthorName, &p.DownloadCount, &p.Version); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan legacy pack row: %v", err)
				continue
			}
			if seenListings[p.ListingID] {
				continue
			}
			seenListings[p.ListingID] = true
			p.PurchaseDate = purchaseDateStr

			// For subscription packs with valid_days=0, default to 30 days (1 month)
			effectiveDays := p.ValidDays
			if p.ShareMode == "subscription" && effectiveDays == 0 {
				effectiveDays = 30
			}
			if (p.ShareMode == "time_limited" || p.ShareMode == "subscription") && effectiveDays > 0 {
				if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr); err == nil {
					p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
				} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr); err == nil {
					p.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format("2006-01-02 15:04:05")
				}
			}

			packs = append(packs, p)
		}
		if err := legacyRows.Err(); err != nil {
			log.Printf("[handleUserDashboard] legacyRows iteration error: %v", err)
		}
	}

	// For ALL subscription packs (both main and legacy), calculate the correct cumulative expiry date.
	// Batch-fetch all renew transactions for this user's subscription packs in a single query
	// to avoid N+1 query problem.
	subListingIDs := make([]int64, 0)
	subIndexMap := make(map[int64][]int) // listing_id -> indices in packs slice
	for i := range packs {
		if packs[i].ShareMode == "subscription" {
			subListingIDs = append(subListingIDs, packs[i].ListingID)
			subIndexMap[packs[i].ListingID] = append(subIndexMap[packs[i].ListingID], i)
		}
	}

	// Batch query all renew transactions for subscription packs
	type renewRecord struct {
		ListingID int64
		CreatedAt string
		Desc      string
	}
	renewsByListing := make(map[int64][]renewRecord)
	if len(subListingIDs) > 0 {
		// Build placeholder string for IN clause
		placeholders := make([]string, len(subListingIDs))
		args := make([]interface{}, 0, len(subListingIDs)+1)
		args = append(args, userID)
		for i, lid := range subListingIDs {
			placeholders[i] = "?"
			args = append(args, lid)
		}
		renewQuery := `SELECT listing_id, created_at, COALESCE(description, '')
			FROM credits_transactions
			WHERE user_id = ? AND listing_id IN (` + strings.Join(placeholders, ",") + `)
			  AND transaction_type = 'renew'
			ORDER BY listing_id, created_at ASC`
		renewRows, err := db.Query(renewQuery, args...)
		if err != nil {
			log.Printf("[USER-DASHBOARD] failed to batch query renew transactions for user %d: %v", userID, err)
		} else {
			defer renewRows.Close()
			for renewRows.Next() {
				var rr renewRecord
				if err := renewRows.Scan(&rr.ListingID, &rr.CreatedAt, &rr.Desc); err != nil {
					continue
				}
				renewsByListing[rr.ListingID] = append(renewsByListing[rr.ListingID], rr)
			}
			if err := renewRows.Err(); err != nil {
				log.Printf("[handleUserDashboard] renewRows iteration error: %v", err)
			}
		}
	}

	// Apply renewal calculations using the batch-fetched data
	for i := range packs {
		if packs[i].ShareMode != "subscription" {
			continue
		}

		// Get original purchase date as the base
		baseTime := time.Time{}
		if t, err := time.Parse("2006-01-02 15:04:05", packs[i].PurchaseDate); err == nil {
			baseTime = t
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", packs[i].PurchaseDate); err == nil {
			baseTime = t
		}
		if baseTime.IsZero() {
			continue
		}

		// Initial subscription period: valid_days from purchase (default 30 days = ~1 month)
		effectiveDays := packs[i].ValidDays
		if effectiveDays == 0 {
			effectiveDays = 30
		}
		currentExpiry := baseTime.AddDate(0, 0, effectiveDays)

		// Apply renew transactions from batch-fetched data
		for _, rr := range renewsByListing[packs[i].ListingID] {
			renewMonths := 1
			if strings.Contains(rr.Desc, "yearly") || strings.Contains(rr.Desc, "14 month") {
				renewMonths = 14
			} else if strings.Contains(rr.Desc, "12 month") {
				renewMonths = 12
			}
			var renewTime time.Time
			if t, err := time.Parse("2006-01-02 15:04:05", rr.CreatedAt); err == nil {
				renewTime = t
			} else if t, err := time.Parse("2006-01-02T15:04:05Z", rr.CreatedAt); err == nil {
				renewTime = t
			}
			if renewTime.IsZero() {
				continue
			}
			if currentExpiry.After(renewTime) {
				currentExpiry = currentExpiry.AddDate(0, renewMonths, 0)
			} else {
				currentExpiry = renewTime.AddDate(0, renewMonths, 0)
			}
			packs[i].PurchaseDate = rr.CreatedAt
		}

		packs[i].ExpiresAt = currentExpiry.Format("2006-01-02 15:04:05")
	}

	// Query password_hash to determine if user has a password set
	var passwordHash sql.NullString
	db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	hasPassword := passwordHash.Valid && passwordHash.String != ""

	// --- Task 3.1: Author role detection + Task 3.3: Author packs ---
	// Combine into a single flow: query author packs directly, derive isAuthor from result.
	var authorData AuthorDashboardData

	// Query author's shared packs with sales data
	splitPctStr := getSetting("revenue_split_publisher_pct")
	splitPct, _ := strconv.ParseFloat(splitPctStr, 64)
	if splitPct <= 0 {
		splitPct = 70 // default 70%
	}
	authorRows, err := db.Query(`
		SELECT pl.id, pl.pack_name, pl.pack_description, pl.share_mode, pl.credits_price, pl.status,
		       COALESCE(sales.sold_count, 0), COALESCE(sales.total_revenue, 0) * ? / 100,
		       COALESCE(pl.version, 1), COALESCE(pl.share_token, '')
		FROM pack_listings pl
		LEFT JOIN (
		    SELECT listing_id, COUNT(*) as sold_count, SUM(ABS(amount)) as total_revenue
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		      AND amount < 0
		    GROUP BY listing_id
		) sales ON sales.listing_id = pl.id
		WHERE pl.user_id = ?
		ORDER BY pl.created_at DESC
	`, splitPct, userID)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query author packs for user %d: %v", userID, err)
	} else {
		defer authorRows.Close()
		for authorRows.Next() {
			var ap AuthorPackInfo
			if err := authorRows.Scan(&ap.ListingID, &ap.PackName, &ap.PackDesc, &ap.ShareMode, &ap.CreditsPrice, &ap.Status, &ap.SoldCount, &ap.TotalRevenue, &ap.Version, &ap.ShareToken); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan author pack row: %v", err)
				continue
			}
			authorData.AuthorPacks = append(authorData.AuthorPacks, ap)
		}
		if err := authorRows.Err(); err != nil {
			log.Printf("[handleUserDashboard] authorRows iteration error: %v", err)
		}
	}
	isAuthor := len(authorData.AuthorPacks) > 0
	authorData.IsAuthor = isAuthor

	if isAuthor {

		// --- Task 3.4: Calculate total revenue, total withdrawn, unwithdrawn credits ---
		var totalRevenue float64
		err = db.QueryRow(`
			SELECT COALESCE(SUM(ABS(ct.amount)), 0)
			FROM credits_transactions ct
			JOIN pack_listings pl ON ct.listing_id = pl.id
			WHERE pl.user_id = ? AND ct.transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
			  AND ct.amount < 0
		`, userID).Scan(&totalRevenue)
		if err != nil {
			log.Printf("[USER-DASHBOARD] failed to query total revenue for user %d: %v", userID, err)
		}

		// Apply revenue split: publisher only gets their configured share (splitPct already loaded above)
		publisherRevenue := totalRevenue * splitPct / 100
		authorData.TotalRevenue = publisherRevenue

		var totalWithdrawn float64
		err = db.QueryRow(`
			SELECT COALESCE(SUM(credits_amount), 0)
			FROM withdrawal_records
			WHERE user_id = ?
		`, userID).Scan(&totalWithdrawn)
		if err != nil {
			log.Printf("[USER-DASHBOARD] failed to query total withdrawn for user %d: %v", userID, err)
		}
		authorData.TotalWithdrawn = totalWithdrawn
		authorData.UnwithdrawnCredits = publisherRevenue - totalWithdrawn
		if authorData.UnwithdrawnCredits < 0 {
			authorData.UnwithdrawnCredits = 0
		}

		// --- Task 3.5: Query credit_cash_rate setting ---
		cashRateStr := getSetting("credit_cash_rate")
		cashRate, _ := strconv.ParseFloat(cashRateStr, 64)
		authorData.CreditCashRate = cashRate
		authorData.WithdrawalEnabled = cashRate > 0
		authorData.RevenueSplitPct = splitPct
	}

	// --- Task 9.1: Query visible notifications for this user ---
	var notifications []NotificationInfo
	now := time.Now()
	notifRows, err := db.Query(`
		SELECT DISTINCT n.id, n.title, n.content, n.target_type, n.effective_date, n.display_duration_days, n.created_at
		FROM notifications n
		LEFT JOIN notification_targets nt ON n.id = nt.notification_id
		WHERE n.status = 'active'
		  AND n.effective_date <= ?
		  AND (n.target_type = 'broadcast' OR (n.target_type = 'targeted' AND nt.user_id = ?))
		ORDER BY n.created_at DESC`,
		now.Format(time.RFC3339), userID,
	)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query notifications for user %d: %v", userID, err)
	} else {
		defer notifRows.Close()
		for notifRows.Next() {
			var n NotificationInfo
			var effectiveDateStr string
			if err := notifRows.Scan(&n.ID, &n.Title, &n.Content, &n.TargetType, &effectiveDateStr, &n.DisplayDurationDays, &n.CreatedAt); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan notification row: %v", err)
				continue
			}
			effectiveDate, err := time.Parse(time.RFC3339, effectiveDateStr)
			if err != nil {
				log.Printf("[USER-DASHBOARD] failed to parse effective_date for notification %d: %v", n.ID, err)
				continue
			}
			if !isNotificationVisible(effectiveDate, n.DisplayDurationDays, now) {
				continue
			}
			n.EffectiveDate = effectiveDateStr
			notifications = append(notifications, n)
		}
		if err := notifRows.Err(); err != nil {
			log.Printf("[handleUserDashboard] notifRows iteration error: %v", err)
		}
	}

	log.Printf("[USER-DASHBOARD] user %d: email=%q, credits=%.0f, packs=%d, hasPassword=%v, isAuthor=%v", userID, user.Email, user.CreditsBalance, len(packs), hasPassword, isAuthor)

	// --- Query top 100 packs by downloads and by revenue for the TOP分析包 tab ---
	// Single query fetching all published packs with revenue data; sort in Go to avoid duplicate subquery.
	var allTopPacks []TopPackInfo
	topRows, err := db.Query(`
		SELECT pl.id, pl.pack_name, COALESCE(pl.author_name, ''), COALESCE(c.name, ''),
		       pl.share_mode, pl.credits_price, pl.download_count,
		       COALESCE(sales.total_revenue, 0)
		FROM pack_listings pl
		LEFT JOIN categories c ON c.id = pl.category_id
		LEFT JOIN (
		    SELECT listing_id, SUM(ABS(amount)) as total_revenue
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		      AND amount < 0
		    GROUP BY listing_id
		) sales ON sales.listing_id = pl.id
		WHERE pl.status = 'published'
		ORDER BY pl.download_count DESC
	`)
	if err != nil {
		log.Printf("[USER-DASHBOARD] failed to query top packs: %v", err)
	} else {
		defer topRows.Close()
		for topRows.Next() {
			var tp TopPackInfo
			if err := topRows.Scan(&tp.ListingID, &tp.PackName, &tp.AuthorName, &tp.CategoryName,
				&tp.ShareMode, &tp.CreditsPrice, &tp.DownloadCount, &tp.TotalRevenue); err != nil {
				log.Printf("[USER-DASHBOARD] failed to scan top pack row: %v", err)
				continue
			}
			allTopPacks = append(allTopPacks, tp)
		}
		if err := topRows.Err(); err != nil {
			log.Printf("[USER-DASHBOARD] topRows iteration error: %v", err)
		}
	}

	// Build top-by-downloads (already sorted by DB query)
	topPacksByDownloads := allTopPacks
	if len(topPacksByDownloads) > 100 {
		topPacksByDownloads = topPacksByDownloads[:100]
	}
	for i := range topPacksByDownloads {
		topPacksByDownloads[i].Rank = i + 1
	}

	// Build top-by-revenue: copy and sort descending by TotalRevenue
	topPacksByRevenue := make([]TopPackInfo, len(allTopPacks))
	copy(topPacksByRevenue, allTopPacks)
	sort.Slice(topPacksByRevenue, func(i, j int) bool {
		return topPacksByRevenue[i].TotalRevenue > topPacksByRevenue[j].TotalRevenue
	})
	if len(topPacksByRevenue) > 100 {
		topPacksByRevenue = topPacksByRevenue[:100]
	}
	for i := range topPacksByRevenue {
		topPacksByRevenue[i].Rank = i + 1
	}

	// --- Task 3.6: Pass author data to template ---
	successMsg := r.URL.Query().Get("success")
	errorMsg := r.URL.Query().Get("error")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	defaultLang := getSetting("default_language")
	if defaultLang == "" {
		defaultLang = "zh-CN"
	}
	templates.UserDashboardTmpl.Execute(w, map[string]interface{}{
		"User":                user,
		"PurchasedPacks":      packs,
		"HasPassword":         hasPassword,
		"AuthorData":          authorData,
		"TopPacksByDownloads": topPacksByDownloads,
		"TopPacksByRevenue":   topPacksByRevenue,
		"Notifications":  notifications,
		"SuccessMsg":     successMsg,
		"ErrorMsg":       errorMsg,
		"DefaultLang":    defaultLang,
	})
}

// handleUserBilling renders the billing page showing all transaction records for the user.
func handleUserBilling(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[USER-BILLING] invalid X-User-ID header: %q", userIDStr)
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	rows, err := db.Query(`
		SELECT ct.id, ct.transaction_type, ct.amount, ct.description, ct.created_at,
		       COALESCE(pl.pack_name, '') as pack_name
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		WHERE ct.user_id = ?
		ORDER BY ct.created_at DESC
	`, userID)
	if err != nil {
		log.Printf("[USER-BILLING] failed to query transactions for user %d: %v", userID, err)
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_billing_failed"), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var records []BillingRecord
	for rows.Next() {
		var rec BillingRecord
		var desc sql.NullString
		if err := rows.Scan(&rec.ID, &rec.TransactionType, &rec.Amount, &desc, &rec.CreatedAt, &rec.PackName); err != nil {
			log.Printf("[USER-BILLING] failed to scan transaction row: %v", err)
			continue
		}
		if desc.Valid {
			rec.Description = desc.String
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleUserBilling] rows iteration error: %v", err)
	}

	log.Printf("[USER-BILLING] user %d: %d transaction records", userID, len(records))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.UserBillingTmpl.Execute(w, struct{ Records []BillingRecord }{Records: records})
}

// handleUserRenewPerUse handles per_use pack renewal from the user portal.
// POST /user/pack/renew-uses
// Form params: listing_id, quantity
func handleUserRenewPerUse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	quantityStr := r.FormValue("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		http.Redirect(w, r, "/user/?error=invalid_quantity", http.StatusFound)
		return
	}

	// Query pack listing info and verify share_mode
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		listingID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/user/?error=pack_not_found", http.StatusFound)
		return
	} else if err != nil {
		log.Printf("[USER-RENEW-USES] failed to query pack listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if shareMode != "per_use" {
		http.Redirect(w, r, "/user/?error=not_per_use", http.StatusFound)
		return
	}

	totalCost := creditsPrice * quantity

	// Check user balance
	var balance float64
	err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to query balance for user %d: %v", userID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if balance < float64(totalCost) {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	// Transaction: deduct credits + record transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to begin transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"UPDATE users SET credits_balance = credits_balance - ? WHERE id = ? AND credits_balance >= ?",
		totalCost, userID, totalCost,
	)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to deduct credits: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
		 VALUES (?, 'purchase_uses', ?, ?, ?)`,
		userID, -float64(totalCost), listingID, fmt.Sprintf("Purchase %d additional uses: %s", quantity, packName),
	)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to record transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[USER-RENEW-USES] failed to commit transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	// Record/restore user purchased pack
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[USER-RENEW-USES] failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
	}

	// Sync pack_usage_records.total_purchased after successful renewal
	_, err = db.Exec(
		`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
		userID, listingID,
	)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to insert pack_usage_records (user=%d, listing=%d): %v", userID, listingID, err)
	}
	_, err = db.Exec(
		`UPDATE pack_usage_records SET total_purchased = total_purchased + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
		quantity, userID, listingID,
	)
	if err != nil {
		log.Printf("[USER-RENEW-USES] failed to update total_purchased (user=%d, listing=%d): %v", userID, listingID, err)
	}

	log.Printf("[USER-RENEW-USES] user %d renewed %d uses for pack %d (%s), cost=%d", userID, quantity, listingID, packName, totalCost)
	http.Redirect(w, r, "/user/?success=renew_uses", http.StatusFound)
}

// handleUserRenewSubscription handles subscription pack renewal from the user portal.
// POST /user/pack/renew-subscription
// Form params: listing_id, months (1 or 12)
func handleUserRenewSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	monthsStr := r.FormValue("months")
	months, err := strconv.Atoi(monthsStr)
	if err != nil || (months != 1 && months != 12) {
		http.Redirect(w, r, "/user/?error=invalid_months", http.StatusFound)
		return
	}

	// Query pack listing info and verify share_mode
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'`,
		listingID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/user/?error=pack_not_found", http.StatusFound)
		return
	} else if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to query pack listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if shareMode != "subscription" {
		http.Redirect(w, r, "/user/?error=not_subscription", http.StatusFound)
		return
	}

	// Calculate cost: monthly = credits_price * months, yearly = credits_price * 12 (grants 14 months)
	totalCost := creditsPrice * months

	// Check user balance
	var balance float64
	err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to query balance for user %d: %v", userID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if balance < float64(totalCost) {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	// Transaction: deduct credits + record transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to begin transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"UPDATE users SET credits_balance = credits_balance - ? WHERE id = ? AND credits_balance >= ?",
		totalCost, userID, totalCost,
	)
	if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to deduct credits: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Redirect(w, r, "/user/?error=insufficient_credits", http.StatusFound)
		return
	}

	// Build description based on months
	var description string
	if months == 12 {
		description = fmt.Sprintf("Renew subscription (yearly, 14 months): %s", packName)
	} else {
		description = fmt.Sprintf("Renew subscription (%d month): %s", months, packName)
	}

	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description)
		 VALUES (?, 'renew', ?, ?, ?)`,
		userID, -float64(totalCost), listingID, description,
	)
	if err != nil {
		log.Printf("[USER-RENEW-SUB] failed to record transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[USER-RENEW-SUB] failed to commit transaction: %v", err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	// Record/restore user purchased pack
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[USER-RENEW-SUB] failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
	}

	log.Printf("[USER-RENEW-SUB] user %d renewed subscription for pack %d (%s), months=%d, cost=%d", userID, listingID, packName, months, totalCost)
	http.Redirect(w, r, "/user/?success=renew_subscription", http.StatusFound)
}

// handleSoftDeletePack handles soft-deleting a purchased pack.
// POST /user/pack/delete
// Form params: listing_id
func handleSoftDeletePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	if err := softDeleteUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[USER-SOFT-DELETE] failed to soft delete pack (user=%d, listing=%d): %v", userID, listingID, err)
		http.Redirect(w, r, "/user/?error=delete_failed", http.StatusFound)
		return
	}

	log.Printf("[USER-SOFT-DELETE] user %d soft-deleted pack %d", userID, listingID)
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}


// handleUserLogout logs out the user by deleting the session and clearing the cookie.
func handleUserLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_session")
	if err == nil && cookie.Value != "" {
		userSessionsMu.Lock()
		delete(userSessions, cookie.Value)
		userSessionsMu.Unlock()
	}
	http.SetCookie(w, makeSessionCookie("user_session", "", -1))
	http.Redirect(w, r, "/user/login", http.StatusFound)
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

// handleUserCaptchaImage serves a math captcha image for the user portal.
func handleUserCaptchaImage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	data := generateMathCaptchaImage(id)
	if data == nil {
		http.Error(w, "captcha expired", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(data)
}

// handleUserCaptchaRefresh creates a new math captcha and returns JSON with captcha_id.
func handleUserCaptchaRefresh(w http.ResponseWriter, r *http.Request) {
	id := createMathCaptcha()
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
	return "https://license.vantagics.com"
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

	// Limit request body size to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max

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
		"success":      true,
		"token":        token,
		"login_ticket": createLoginTicket(user.ID),
		"user": map[string]interface{}{
			"id":              user.ID,
			"display_name":    user.DisplayName,
			"email":           user.Email,
			"credits_balance": user.CreditsBalance,
		},
	})
}


// handleTicketLogin handles GET /user/ticket-login?ticket=xxx.
// It consumes a one-time login ticket, creates a user session, and redirects.
// If the user has no password set, redirects to the set-password page.
func handleTicketLogin(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	userID := consumeLoginTicket(ticket)
	if userID == 0 {
		log.Printf("[TICKET-LOGIN] invalid or expired ticket")
		http.Redirect(w, r, "/user/login?error=ticket_expired", http.StatusFound)
		return
	}

	// Check if user is blocked
	var isBlocked int
	if err := db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked); err == nil && isBlocked == 1 {
		http.Redirect(w, r, "/user/login?error=blocked", http.StatusFound)
		return
	}

	// Create session
	sid := createUserSession(userID)
	http.SetCookie(w, makeSessionCookie("user_session", sid, 86400))

	// Check if user has a password set
	var passwordHash sql.NullString
	err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("[TICKET-LOGIN] failed to check password for user %d: %v", userID, err)
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
		return
	}

	if !passwordHash.Valid || passwordHash.String == "" {
		// First time: redirect to set-password page
		log.Printf("[TICKET-LOGIN] user %d has no password, redirecting to set-password", userID)
		http.Redirect(w, r, "/user/set-password", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

// handleUserChangePassword handles GET/POST /user/change-password.
// Allows users who already have a password to change it.
// Users without a password are redirected to /user/set-password.
func handleUserChangePassword(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Query user info
	var passwordHash sql.NullString
	var email string
	err = db.QueryRow("SELECT email, password_hash FROM users WHERE id = ?", userID).Scan(&email, &passwordHash)
	if err != nil {
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_failed"), http.StatusInternalServerError)
		return
	}

	// If user has no password, redirect to set-password page (Requirement 3.7)
	if !passwordHash.Valid || passwordHash.String == "" {
		http.Redirect(w, r, "/user/set-password", http.StatusFound)
		return
	}

	lang := i18n.DetectLang(r)

	renderForm := func(errMsg, successMsg string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"Email":   email,
			"Error":   errMsg,
			"Success": successMsg,
		})
		templates.UserChangePasswordTmpl.Execute(w, data)
	}

	if r.Method == http.MethodGet {
		renderForm("", "")
		return
	}

	// POST: change password
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate current password (Requirement 3.4)
	if !checkPassword(currentPassword, passwordHash.String) {
		renderForm(i18n.T(lang, "invalid_old_password"), "")
		return
	}

	// Validate new password length (Requirement 3.5)
	if len(newPassword) < 6 {
		renderForm(i18n.T(lang, "password_min_6"), "")
		return
	}
	if len(newPassword) > 72 {
		renderForm(i18n.T(lang, "password_max_72"), "")
		return
	}

	// Validate password confirmation (Requirement 3.6)
	if newPassword != confirmPassword {
		renderForm(i18n.T(lang, "password_mismatch"), "")
		return
	}

	// Update password (Requirement 3.3)
	hashed := hashPassword(newPassword)
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hashed, userID)
	if err != nil {
		log.Printf("[CHANGE-PASSWORD] failed to update password for user %d: %v", userID, err)
		renderForm(i18n.T(lang, "change_password_failed"), "")
		return
	}

	log.Printf("[CHANGE-PASSWORD] user %d changed password successfully", userID)
	renderForm("", i18n.T(lang, "change_password_success"))
}

// handleUserSetPassword handles GET/POST /user/set-password.
// Shows a form for users to set their password (first-time login via SSO).
func handleUserSetPassword(w http.ResponseWriter, r *http.Request) {
	// Get user from session (protected by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Check if user already has a password
	var passwordHash sql.NullString
	var email string
	err = db.QueryRow("SELECT email, password_hash FROM users WHERE id = ?", userID).Scan(&email, &passwordHash)
	if err != nil {
		http.Error(w, i18n.T(i18n.DetectLang(r), "load_failed"), http.StatusInternalServerError)
		return
	}

	if passwordHash.Valid && passwordHash.String != "" {
		// Already has password, go to dashboard
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"Email": email,
			"Error": "",
		})
		templates.UserSetPasswordTmpl.Execute(w, data)
		return
	}

	// POST: set password
	lang := i18n.DetectLang(r)
	password := r.FormValue("password")
	password2 := r.FormValue("password2")

	errMsg := ""
	if len(password) < 6 {
		errMsg = i18n.T(lang, "password_min_6")
	} else if len(password) > 72 {
		errMsg = i18n.T(lang, "password_max_72")
	} else if password != password2 {
		errMsg = i18n.T(lang, "password_mismatch")
	}

	if errMsg != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := i18n.TemplateData(r)
		i18n.MergeTemplateData(data, map[string]interface{}{
			"Email": email,
			"Error": errMsg,
		})
		templates.UserSetPasswordTmpl.Execute(w, data)
		return
	}

	// Set password and username (use email prefix as username)
	hashed := hashPassword(password)
	username := email
	if idx := strings.Index(email, "@"); idx > 0 {
		username = email[:idx]
	}

	_, err = db.Exec("UPDATE users SET password_hash = ?, username = ? WHERE id = ? AND (password_hash IS NULL OR password_hash = '')",
		hashed, username, userID)
	if err != nil {
		log.Printf("[SET-PASSWORD] failed to update password for user %d: %v", userID, err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		errData := i18n.TemplateData(r)
		i18n.MergeTemplateData(errData, map[string]interface{}{
			"Email": email,
			"Error": i18n.T(lang, "set_password_failed"),
		})
		templates.UserSetPasswordTmpl.Execute(w, errData)
		return
	}

	log.Printf("[SET-PASSWORD] user %d set password successfully, username=%q", userID, username)
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
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
	if err := rows.Err(); err != nil {
		log.Printf("[handleListCategories] rows iteration error: %v", err)
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
		PackName    string `json:"pack_name"`
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
func validatePricingParams(shareMode string, creditsPrice int) string {
	switch shareMode {
	case "free":
		return ""
	case "per_use":
		if creditsPrice < 1 || creditsPrice > 100 {
			return "credits_price must be between 1 and 100 for per_use mode"
		}
		return ""
	case "subscription":
		if creditsPrice < 100 || creditsPrice > 1000 {
			return "credits_price must be between 100 and 1000 for subscription mode"
		}
		return ""
	default:
		return "share_mode must be 'free', 'per_use', or 'subscription'"
	}
}

// handleGetListingID handles GET /api/packs/listing-id?pack_name={name}.
// Returns the listing_id for a published pack owned by the authenticated user.
func handleGetListingID(w http.ResponseWriter, r *http.Request) {
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

	packName := r.URL.Query().Get("pack_name")
	if packName == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack_name is required"})
		return
	}

	var listingID int64
	var shareToken sql.NullString
	err = db.QueryRow(
		"SELECT id, share_token FROM pack_listings WHERE pack_name = ? AND user_id = ? AND status = 'published'",
		packName, userID,
	).Scan(&listingID, &shareToken)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query listing ID: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"listing_id":  listingID,
		"share_token": shareToken.String,
	})
}

// handleGetPackDetail handles GET /api/packs/{listing_id}/detail.
// Returns full pack detail JSON for a published pack. Public access, no auth required.
func handleGetPackDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse listing_id from URL path: /api/packs/{listing_id}/detail
	path := strings.TrimPrefix(r.URL.Path, "/api/packs/")
	path = strings.TrimSuffix(path, "/detail")
	listingID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid listing_id"})
		return
	}

	var detail struct {
		ListingID       int64  `json:"listing_id"`
		PackName        string `json:"pack_name"`
		PackDescription string `json:"pack_description"`
		SourceName      string `json:"source_name"`
		AuthorName      string `json:"author_name"`
		ShareMode       string `json:"share_mode"`
		CreditsPrice    int    `json:"credits_price"`
		DownloadCount   int    `json:"download_count"`
		CategoryName    string `json:"category_name"`
	}

	err = db.QueryRow(`
		SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''), COALESCE(pl.source_name, ''),
		       COALESCE(pl.author_name, ''), pl.share_mode, pl.credits_price, pl.download_count,
		       COALESCE(c.name, '')
		FROM pack_listings pl
		LEFT JOIN categories c ON pl.category_id = c.id
		WHERE pl.id = ? AND pl.status = 'published'`,
		listingID,
	).Scan(&detail.ListingID, &detail.PackName, &detail.PackDescription, &detail.SourceName,
		&detail.AuthorName, &detail.ShareMode, &detail.CreditsPrice, &detail.DownloadCount,
		&detail.CategoryName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found or not published"})
		return
	}
	if err != nil {
		log.Printf("Failed to query pack detail: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, detail)
}

// handlePackDetailPage handles GET /pack/{share_token}.
// Renders the server-side pack detail HTML page.
// Optionally checks user login status via user_session cookie (not enforced).
func handlePackDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse share_token from URL path: /pack/{share_token}
	shareToken := strings.TrimPrefix(r.URL.Path, "/pack/")
	shareToken = strings.TrimSuffix(shareToken, "/")

	// Resolve share_token to listing_id
	listingID, err := resolveShareToken(shareToken)
	if err != nil || listingID <= 0 {
		lang := i18n.DetectLang(r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.PackDetailTmpl.Execute(w, map[string]interface{}{
			"ListingID":    int64(0),
			"ShareToken":   "",
			"PackName":     "",
			"PackDescription": "",
			"SourceName":   "",
			"AuthorName":   "",
			"ShareMode":    "",
			"CreditsPrice": 0,
			"DownloadCount": 0,
			"CategoryName": "",
			"IsLoggedIn":   false,
			"HasPurchased": false,
			"Error":        i18n.T(lang, "invalid_pack_link"),
			"MonthOptions": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		})
		return
	}

	// Query pack detail from database
	var packName, packDesc, sourceName, authorName, shareMode, categoryName string
	var creditsPrice, downloadCount int
	err = db.QueryRow(`
		SELECT pl.pack_name, COALESCE(pl.pack_description, ''), COALESCE(pl.source_name, ''),
		       COALESCE(pl.author_name, ''), pl.share_mode, pl.credits_price, pl.download_count,
		       COALESCE(c.name, '')
		FROM pack_listings pl
		LEFT JOIN categories c ON pl.category_id = c.id
		WHERE pl.id = ? AND pl.status = 'published'`,
		listingID,
	).Scan(&packName, &packDesc, &sourceName, &authorName, &shareMode, &creditsPrice, &downloadCount, &categoryName)
	if err == sql.ErrNoRows {
		lang := i18n.DetectLang(r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.PackDetailTmpl.Execute(w, map[string]interface{}{
			"ListingID":    listingID,
			"ShareToken":   shareToken,
			"PackName":     "",
			"PackDescription": "",
			"SourceName":   "",
			"AuthorName":   "",
			"ShareMode":    "",
			"CreditsPrice": 0,
			"DownloadCount": 0,
			"CategoryName": "",
			"IsLoggedIn":   false,
			"HasPurchased": false,
			"Error":        i18n.T(lang, "pack_not_found"),
			"MonthOptions": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		})
		return
	}
	if err != nil {
		log.Printf("[PACK-DETAIL-PAGE] failed to query pack id=%d: %v", listingID, err)
		http.Error(w, i18n.T(i18n.DetectLang(r), "server_internal_error"), http.StatusInternalServerError)
		return
	}

	// Optionally check user login status (read user_session cookie, not enforced)
	isLoggedIn := false
	hasPurchased := false
	var userID int64
	cookie, cookieErr := r.Cookie("user_session")
	if cookieErr == nil && isValidUserSession(cookie.Value) {
		userID = getUserSessionUserID(cookie.Value)
		if userID > 0 {
			isLoggedIn = true

			// Check if user already purchased this pack
			var count int
			err = db.QueryRow(
				"SELECT COUNT(*) FROM user_purchased_packs WHERE user_id = ? AND listing_id = ? AND (is_hidden IS NULL OR is_hidden = 0)",
				userID, listingID,
			).Scan(&count)
			if err == nil && count > 0 {
				hasPurchased = true
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.PackDetailTmpl.Execute(w, map[string]interface{}{
		"ListingID":       listingID,
		"ShareToken":      shareToken,
		"PackName":        packName,
		"PackDescription": packDesc,
		"SourceName":      sourceName,
		"AuthorName":      authorName,
		"ShareMode":       shareMode,
		"CreditsPrice":    creditsPrice,
		"DownloadCount":   downloadCount,
		"CategoryName":    categoryName,
		"IsLoggedIn":      isLoggedIn,
		"HasPurchased":    hasPurchased,
		"Error":           "",
		"MonthOptions":    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	})
}

// handleClaimFreePack handles POST /pack/{share_token}/claim.
// Protected by userAuth middleware. Only allows claiming free packs (share_mode='free').
// Creates a purchase record via upsertUserPurchasedPack and records a download in user_downloads.
func handleClaimFreePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[CLAIM-FREE-PACK] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse share_token from URL path: /pack/{share_token}/claim
	path := strings.TrimPrefix(r.URL.Path, "/pack/")
	path = strings.TrimSuffix(path, "/claim")
	path = strings.TrimSuffix(path, "/")
	shareToken := path

	// Resolve share_token to listing_id
	listingID, err := resolveShareToken(shareToken)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_listing_id"})
		return
	}

	// Verify the pack exists, is published, and share_mode='free'
	var shareMode string
	err = db.QueryRow(
		"SELECT share_mode FROM pack_listings WHERE id = ? AND status = 'published'",
		listingID,
	).Scan(&shareMode)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack_not_found"})
		return
	}
	if err != nil {
		log.Printf("[CLAIM-FREE-PACK] failed to query pack id=%d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if shareMode != "free" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack_not_free"})
		return
	}

	// Create/update purchase record
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[CLAIM-FREE-PACK] failed to upsert purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Record download in user_downloads table (with buyer IP for region tracking)
	_, err = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, listingID, getClientIP(r))
	if err != nil {
		log.Printf("[CLAIM-FREE-PACK] failed to record download (user=%d, listing=%d): %v", userID, listingID, err)
		// Non-critical: purchase record already created, so we still return success
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

func handlePurchaseFromDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	// Get user_id from X-User-ID header (set by userAuth middleware)
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] invalid X-User-ID header: %q", userIDStr)
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse share_token from URL path: /pack/{share_token}/purchase
	path := strings.TrimPrefix(r.URL.Path, "/pack/")
	path = strings.TrimSuffix(path, "/purchase")
	path = strings.TrimSuffix(path, "/")
	shareToken := path

	// Resolve share_token to listing_id
	listingID, err := resolveShareToken(shareToken)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_listing_id"})
		return
	}

	// Verify the pack exists, is published, and share_mode is NOT 'free'
	var shareMode string
	var creditsPrice int
	var packName string
	err = db.QueryRow(
		"SELECT share_mode, credits_price, pack_name FROM pack_listings WHERE id = ? AND status = 'published'",
		listingID,
	).Scan(&shareMode, &creditsPrice, &packName)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack_not_found"})
		return
	}
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to query pack id=%d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if shareMode == "free" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack_is_free"})
		return
	}

	// Parse JSON body
	var reqBody struct {
		Quantity int `json:"quantity"`
		Months   int `json:"months"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request_body"})
		return
	}

	// Calculate total cost based on share_mode
	var totalCost int
	switch shareMode {
	case "per_use":
		if reqBody.Quantity <= 0 {
			reqBody.Quantity = 1
		}
		totalCost = creditsPrice * reqBody.Quantity
	case "subscription":
		if reqBody.Months <= 0 {
			reqBody.Months = 1
		}
		if reqBody.Months > 12 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "months_must_be_1_to_12"})
			return
		}
		totalCost = creditsPrice * reqBody.Months
	default:
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "unsupported_share_mode"})
		return
	}

	// Check user's credits balance
	var balance float64
	err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to query balance for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if balance < float64(totalCost) {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"insufficient_balance": true,
			"balance":              balance,
		})
		return
	}

	// Use a database transaction for atomic credits deduction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to begin transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer tx.Rollback()

	// Deduct credits atomically
	result, err := tx.Exec(
		"UPDATE users SET credits_balance = credits_balance - ? WHERE id = ? AND credits_balance >= ?",
		totalCost, userID, totalCost,
	)
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to deduct credits: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"insufficient_balance": true,
			"balance":              balance,
		})
		return
	}

	// Record credits transaction
	var description string
	if shareMode == "per_use" {
		description = fmt.Sprintf("Purchase %d uses from detail: %s", reqBody.Quantity, packName)
	} else {
		description = fmt.Sprintf("Purchase %d month(s) subscription from detail: %s", reqBody.Months, packName)
	}
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
		 VALUES (?, 'purchase', ?, ?, ?, ?)`,
		userID, -float64(totalCost), listingID, description, getClientIP(r),
	)
	if err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to record transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to commit transaction: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// Create/update user purchased pack record
	if err := upsertUserPurchasedPack(userID, listingID); err != nil {
		log.Printf("[PURCHASE-FROM-DETAIL] failed to upsert purchased pack (user=%d, listing=%d): %v", userID, listingID, err)
	}

	// For per_use packs, sync pack_usage_records.total_purchased
	if shareMode == "per_use" {
		_, err = db.Exec(
			`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
			userID, listingID,
		)
		if err != nil {
			log.Printf("[PURCHASE-FROM-DETAIL] failed to insert pack_usage_records (user=%d, listing=%d): %v", userID, listingID, err)
		}
		_, err = db.Exec(
			`UPDATE pack_usage_records SET total_purchased = total_purchased + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
			reqBody.Quantity, userID, listingID,
		)
		if err != nil {
			log.Printf("[PURCHASE-FROM-DETAIL] failed to update total_purchased (user=%d, listing=%d): %v", userID, listingID, err)
		}
	}

	log.Printf("[PURCHASE-FROM-DETAIL] user %d purchased pack %d (%s), mode=%s, cost=%d", userID, listingID, packName, shareMode, totalCost)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"credits_deducted": totalCost,
	})
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
	if shareMode != "free" && shareMode != "per_use" && shareMode != "subscription" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "share_mode must be 'free', 'per_use', or 'subscription'"})
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

	// Validate pricing parameters
	if errMsg := validatePricingParams(shareMode, creditsPrice); errMsg != "" {
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
	var isEncrypted bool

	// First, try to read metadata.json (always unencrypted, written by PackToZip)
	for _, f := range zipReader.File {
		if f.Name == "metadata.json" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			// metadata.json contains only the metadata object directly
			var meta struct {
				PackName    string `json:"pack_name"`
				Author      string `json:"author"`
				CreatedAt   string `json:"created_at"`
				SourceName  string `json:"source_name"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(jsonData, &meta); err == nil {
				qapContent.Metadata = meta
				foundJSON = true
			}
			break
		}
	}

	// Then, try to read pack.json for full content (schema_requirements, etc.)
	for _, f := range zipReader.File {
		if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
			rc, err := f.Open()
			if err != nil {
				if !foundJSON {
					jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
					return
				}
				break
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				if !foundJSON {
					jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
					return
				}
				break
			}
			// Check if pack.json is encrypted (starts with "QAPENC" magic header)
			if len(jsonData) >= 6 && string(jsonData[:6]) == "QAPENC" {
				isEncrypted = true
				// Encrypted pack — metadata already read from metadata.json above
				break
			}
			var fullContent qapFileContent
			if err := json.Unmarshal(jsonData, &fullContent); err != nil {
				if !foundJSON {
					jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
					return
				}
				break
			}
			// Full content parsed successfully — use it (overrides metadata.json)
			qapContent = fullContent
			foundJSON = true
			break
		}
	}
	_ = isEncrypted

	if !foundJSON {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	// Encrypt paid packs (per_use or subscription)
	var encryptionPassword string
	if shareMode == "per_use" || shareMode == "subscription" {
		// Reject pre-encrypted packs — server must control encryption
		if isEncrypted {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "paid packs must not be pre-encrypted"})
			return
		}
	}

	// Use pack_name from metadata, fall back to source_name, then "Untitled"
	packName := qapContent.Metadata.PackName
	if packName == "" {
		packName = qapContent.Metadata.SourceName
	}
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

	// Insert pack_listing record (with original fileData to get listingID first)
	shareToken := generateShareToken()
	result, err := db.Exec(
		`INSERT INTO pack_listings (user_id, category_id, file_data, pack_name, pack_description, source_name, author_name, share_mode, credits_price, status, meta_info, encryption_password, share_token)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)`,
		userID, categoryID, fileData, packName, qapContent.Metadata.Description,
		qapContent.Metadata.SourceName, qapContent.Metadata.Author, shareMode, creditsPrice, metaInfoJSON, encryptionPassword, shareToken,
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

	// Inject listing_id into the .qap file, then encrypt if paid, and UPDATE file_data
	{
		// Step 1: Inject listing_id into pack.json and metadata.json
		injectedData, err := injectListingIDIntoQAP(fileData, listingID)
		if err != nil {
			log.Printf("Failed to inject listing_id into QAP: %v", err)
			// Non-fatal: the pack will work without listing_id (client has fallback)
		} else {
			fileData = injectedData
		}

		// Step 2: Encrypt if paid pack
		if shareMode == "per_use" || shareMode == "subscription" {
			// Extract pack.json bytes from the (now listing_id-injected) ZIP
			var packJSONBytes []byte
			zr2, _ := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
			for _, f := range zr2.File {
				if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
					rc, err := f.Open()
					if err != nil {
						break
					}
					packJSONBytes, err = io.ReadAll(rc)
					rc.Close()
					if err != nil {
						packJSONBytes = nil
					}
					break
				}
			}
			if packJSONBytes != nil {
				pwd, err := generateSecurePassword()
				if err == nil {
					encryptedData, err := serverEncryptPackJSON(packJSONBytes, pwd)
					if err == nil {
						newFileData, err := repackZipWithEncryptedData(fileData, encryptedData)
						if err == nil {
							fileData = newFileData
							encryptionPassword = pwd
						} else {
							log.Printf("Failed to repack ZIP: %v", err)
						}
					} else {
						log.Printf("Failed to encrypt pack.json: %v", err)
					}
				} else {
					log.Printf("Failed to generate encryption password: %v", err)
				}
			}
		}

		// Step 3: UPDATE file_data and encryption_password with the final version
		_, err = db.Exec(
			`UPDATE pack_listings SET file_data = ?, encryption_password = ? WHERE id = ?`,
			fileData, encryptionPassword, listingID,
		)
		if err != nil {
			log.Printf("Failed to update file_data with listing_id: %v", err)
		}
	}

	// Read back the created listing
	var listing PackListingInfo
	var metaInfoReadBack sql.NullString
	err = db.QueryRow(
		`SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		        pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.status, pl.meta_info, pl.created_at
		 FROM pack_listings pl
		 JOIN categories c ON c.id = pl.category_id
		 WHERE pl.id = ?`, listingID,
	).Scan(&listing.ID, &listing.UserID, &listing.CategoryID, &listing.CategoryName,
		&listing.PackName, &listing.PackDescription, &listing.SourceName, &listing.AuthorName,
		&listing.ShareMode, &listing.CreditsPrice, &listing.DownloadCount, &listing.Status, &metaInfoReadBack, &listing.CreatedAt)
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

func handleReplacePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse multipart form (max 500MB)
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

	// Parse listing_id (the published pack to replace)
	listingIDStr := r.FormValue("listing_id")
	if listingIDStr == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "listing_id is required"})
		return
	}
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid listing_id"})
		return
	}

	// Verify listing exists, belongs to current user, and is published
	var ownerID int64
	var currentStatus string
	var currentVersion int
	var categoryID int64
	var shareMode string
	var creditsPrice int
	err = db.QueryRow(
		`SELECT user_id, status, COALESCE(version, 1), category_id, share_mode, credits_price FROM pack_listings WHERE id = ?`,
		listingID,
	).Scan(&ownerID, &currentStatus, &currentVersion, &categoryID, &shareMode, &creditsPrice)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing not found"})
		return
	}
	if err != nil {
		log.Printf("[REPLACE-PACK] failed to query listing %d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if ownerID != userID {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "not your listing"})
		return
	}
	if currentStatus != "published" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "can only replace published packs"})
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
		log.Printf("[REPLACE-PACK] failed to read uploaded file: %v", err)
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
	var isEncrypted bool

	for _, f := range zipReader.File {
		if f.Name == "metadata.json" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			var meta struct {
				PackName    string `json:"pack_name"`
				Author      string `json:"author"`
				CreatedAt   string `json:"created_at"`
				SourceName  string `json:"source_name"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(jsonData, &meta); err == nil {
				qapContent.Metadata = meta
				foundJSON = true
			}
			break
		}
	}

	for _, f := range zipReader.File {
		if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
			rc, err := f.Open()
			if err != nil {
				break
			}
			jsonData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				break
			}
			if len(jsonData) >= 6 && string(jsonData[:6]) == "QAPENC" {
				isEncrypted = true
				break
			}
			var fullContent qapFileContent
			if err := json.Unmarshal(jsonData, &fullContent); err == nil {
				qapContent = fullContent
				foundJSON = true
			}
			break
		}
	}
	_ = isEncrypted

	if !foundJSON {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_pack_format"})
		return
	}

	// Inject listing_id into the .qap file before encryption
	var encryptionPassword string
	if !isEncrypted {
		injectedData, err := injectListingIDIntoQAP(fileData, listingID)
		if err != nil {
			log.Printf("[REPLACE-PACK] Failed to inject listing_id: %v", err)
		} else {
			fileData = injectedData
		}
	}

	// Encrypt paid packs
	if shareMode == "per_use" || shareMode == "subscription" {
		if isEncrypted {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "paid packs must not be pre-encrypted"})
			return
		}
		var packJSONBytes []byte
		zr2, _ := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
		for _, f := range zr2.File {
			if f.Name == "pack.json" || f.Name == "analysis_pack.json" {
				rc, err := f.Open()
				if err != nil {
					break
				}
				packJSONBytes, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					packJSONBytes = nil
				}
				break
			}
		}
		if packJSONBytes == nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		pwd, err := generateSecurePassword()
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		encryptedData, err := serverEncryptPackJSON(packJSONBytes, pwd)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		newFileData, err := repackZipWithEncryptedData(fileData, encryptedData)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		fileData = newFileData
		encryptionPassword = pwd
	}

	// Extract meta info
	metaInfo := PackMetaInfo{Tables: []PackMetaTable{}}
	for _, sr := range qapContent.SchemaRequirements {
		table := PackMetaTable{TableName: sr.TableName, Columns: []string{}}
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

	packName := qapContent.Metadata.PackName
	if packName == "" {
		packName = qapContent.Metadata.SourceName
	}
	if packName == "" {
		packName = "Untitled"
	}

	newVersion := currentVersion + 1

	// Update the listing: replace file_data, update metadata, bump version, reset to pending
	_, err = db.Exec(`
		UPDATE pack_listings
		SET file_data = ?, pack_name = ?, pack_description = ?, source_name = ?, author_name = ?,
		    meta_info = ?, encryption_password = ?, version = ?,
		    status = 'pending', reviewed_by = NULL, reviewed_at = NULL, reject_reason = NULL
		WHERE id = ? AND user_id = ?
	`, fileData, packName, qapContent.Metadata.Description, qapContent.Metadata.SourceName,
		qapContent.Metadata.Author, metaInfoJSON, encryptionPassword, newVersion, listingID, userID)
	if err != nil {
		log.Printf("[REPLACE-PACK] failed to update listing %d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	log.Printf("[REPLACE-PACK] user %d replaced listing %d, version %d -> %d", userID, listingID, currentVersion, newVersion)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"listing_id":  listingID,
		"new_version": newVersion,
		"status":      "pending",
	})
}

func handleReportPackUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Get user ID from auth middleware
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse request body
	var req struct {
		ListingID int64  `json:"listing_id"`
		UsedAt    string `json:"used_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Validate fields
	if req.ListingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "listing_id must be positive"})
		return
	}
	if req.UsedAt == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "used_at is required"})
		return
	}
	if _, err := time.Parse(time.RFC3339, req.UsedAt); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "used_at must be valid RFC3339 format"})
		return
	}

	// Verify listing exists and is per_use
	var shareMode string
	err = db.QueryRow(
		`SELECT share_mode FROM pack_listings WHERE id = ?`,
		req.ListingID,
	).Scan(&shareMode)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if shareMode != "per_use" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack is not per_use type"})
		return
	}

	// INSERT OR IGNORE into pack_usage_log for idempotent dedup
	result, err := db.Exec(
		`INSERT OR IGNORE INTO pack_usage_log (user_id, listing_id, used_at) VALUES (?, ?, ?)`,
		userID, req.ListingID, req.UsedAt,
	)
	if err != nil {
		log.Printf("Failed to insert usage log: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// New usage record — ensure pack_usage_records row exists, then increment
		_, err = db.Exec(
			`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
			userID, req.ListingID,
		)
		if err != nil {
			log.Printf("Failed to init pack_usage_records: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		_, err = db.Exec(
			`UPDATE pack_usage_records SET used_count = used_count + 1, last_used_at = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
			req.UsedAt, userID, req.ListingID,
		)
		if err != nil {
			log.Printf("Failed to update pack_usage_records: %v", err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}
	// If rowsAffected == 0, it's a duplicate — no increment needed

	// Query current counts to return
	var usedCount, totalPurchased int
	err = db.QueryRow(
		`SELECT used_count, total_purchased FROM pack_usage_records WHERE user_id = ? AND listing_id = ?`,
		userID, req.ListingID,
	).Scan(&usedCount, &totalPurchased)
	if err != nil {
		log.Printf("Failed to query pack_usage_records: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	remaining := totalPurchased - usedCount
	if remaining < 0 {
		remaining = 0
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"used_count":      usedCount,
		"total_purchased": totalPurchased,
		"remaining_uses":  remaining,
		"exhausted":       usedCount >= totalPurchased,
	})
}

// handleGetMyLicenses handles GET /api/packs/my-licenses.
// Returns the authenticated user's usage license info for all purchased packs.
// The client uses this to sync its local UsageLicenseStore with the server's authoritative data.
func handleGetMyLicenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	if userID == 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	type LicenseJSON struct {
		ListingID          int64  `json:"listing_id"`
		PackName           string `json:"pack_name"`
		PricingModel       string `json:"pricing_model"`
		RemainingUses      int    `json:"remaining_uses"`
		TotalUses          int    `json:"total_uses"`
		ExpiresAt          string `json:"expires_at"`
		SubscriptionMonths int    `json:"subscription_months"`
	}

	rows, err := db.Query(`
		SELECT pl.id, pl.pack_name, pl.share_mode, pl.credits_price, COALESCE(pl.valid_days, 0),
		       COALESCE(pur.used_count, 0), COALESCE(pur.total_purchased, 0),
		       COALESCE(src.purchase_date, upp.created_at) as purchase_date
		FROM user_purchased_packs upp
		JOIN pack_listings pl ON upp.listing_id = pl.id
		LEFT JOIN pack_usage_records pur ON pur.user_id = upp.user_id AND pur.listing_id = upp.listing_id
		LEFT JOIN (
		    SELECT user_id, listing_id, MIN(created_at) as purchase_date
		    FROM credits_transactions
		    WHERE transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription')
		    GROUP BY user_id, listing_id
		) src ON src.user_id = upp.user_id AND src.listing_id = upp.listing_id
		WHERE upp.user_id = ? AND (upp.is_hidden IS NULL OR upp.is_hidden = 0)
		ORDER BY upp.updated_at DESC
	`, userID)
	if err != nil {
		log.Printf("[handleGetMyLicenses] query error for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	licenses := []LicenseJSON{}
	for rows.Next() {
		var listingID int64
		var packName, shareMode string
		var creditsPrice, validDays, usedCount, totalPurchased int
		var purchaseDateStr string
		if err := rows.Scan(&listingID, &packName, &shareMode, &creditsPrice, &validDays, &usedCount, &totalPurchased, &purchaseDateStr); err != nil {
			log.Printf("[handleGetMyLicenses] scan error: %v", err)
			continue
		}

		lic := LicenseJSON{
			ListingID:    listingID,
			PackName:     packName,
			PricingModel: shareMode,
		}

		switch shareMode {
		case "per_use":
			lic.TotalUses = totalPurchased
			remaining := totalPurchased - usedCount
			if remaining < 0 {
				remaining = 0
			}
			lic.RemainingUses = remaining
		case "subscription":
			effectiveDays := validDays
			if effectiveDays == 0 {
				effectiveDays = 30
			}
			if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr); err == nil {
				lic.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format(time.RFC3339)
			} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr); err == nil {
				lic.ExpiresAt = t.AddDate(0, 0, effectiveDays).Format(time.RFC3339)
			}
			lic.SubscriptionMonths = validDays / 30
		}

		licenses = append(licenses, lic)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleGetMyLicenses] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"licenses": licenses})
}

// handleGetPurchasedPacks handles GET /api/packs/purchased.
// Returns the authenticated user's purchased packs (excluding hidden ones) as JSON.
func handleGetPurchasedPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	if userID == 0 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	type PurchasedPackJSON struct {
		ListingID       int64  `json:"listing_id"`
		PackName        string `json:"pack_name"`
		PackDescription string `json:"pack_description"`
		SourceName      string `json:"source_name"`
		AuthorName      string `json:"author_name"`
		ShareMode       string `json:"share_mode"`
		CreditsPrice    int    `json:"credits_price"`
		CreatedAt       string `json:"created_at"`
	}

	rows, err := db.Query(`
		SELECT pl.id, pl.pack_name, COALESCE(pl.pack_description, ''), COALESCE(pl.source_name, ''),
		       COALESCE(u.display_name, u.email, '') as author_name,
		       pl.share_mode, pl.credits_price, pl.created_at
		FROM user_purchased_packs upp
		JOIN pack_listings pl ON upp.listing_id = pl.id
		LEFT JOIN users u ON pl.user_id = u.id
		WHERE upp.user_id = ? AND (upp.is_hidden IS NULL OR upp.is_hidden = 0)
		ORDER BY upp.updated_at DESC
	`, userID)
	if err != nil {
		log.Printf("[handleGetPurchasedPacks] query error for user %d: %v", userID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	packs := []PurchasedPackJSON{}
	for rows.Next() {
		var p PurchasedPackJSON
		if err := rows.Scan(&p.ListingID, &p.PackName, &p.PackDescription, &p.SourceName, &p.AuthorName, &p.ShareMode, &p.CreditsPrice, &p.CreatedAt); err != nil {
			log.Printf("[handleGetPurchasedPacks] scan error: %v", err)
			continue
		}
		packs = append(packs, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleGetPurchasedPacks] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"packs": packs})
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
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.meta_info, pl.created_at
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
			&l.ShareMode, &l.CreditsPrice, &l.DownloadCount, &metaInfoStr, &l.CreatedAt); err != nil {
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
	if err := rows.Err(); err != nil {
		log.Printf("[handleListPacks] rows iteration error: %v", err)
	}

	// Optionally resolve user identity from Authorization header
	userID := optionalUserID(r)
	if userID > 0 {
		purchasedSet := getUserPurchasedListingIDs(userID)
		for i := range listings {
			listings[i].Purchased = purchasedSet[listings[i].ID]
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"packs": listings})
}

// handleDownloadPack handles GET /api/packs/{id}/download.
// Free packs return file data directly. Paid packs check credits balance,
// deduct if sufficient, and return file data; otherwise return 402.

// sanitizeDownloadFilename removes characters that could cause HTTP header injection
// or filesystem issues in Content-Disposition filenames.
func sanitizeDownloadFilename(name string) string {
	// Remove characters that are dangerous in HTTP headers and filenames
	replacer := strings.NewReplacer(
		`"`, "",
		"\r", "",
		"\n", "",
		"\\", "_",
		"/", "_",
	)
	return replacer.Replace(name)
}

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

	// Look up the pack listing.
	// For re-downloads by users who already purchased, allow any status (including delisted).
	// First try published, then check if user has a purchase record for non-published packs.
	var shareMode string
	var creditsPrice int
	var fileData []byte
	var packName string
	var metaInfoStr sql.NullString
	var encryptionPassword string
	var packStatus string
	err = db.QueryRow(
		`SELECT share_mode, credits_price, file_data, pack_name, meta_info, encryption_password, status FROM pack_listings WHERE id = ?`,
		packID,
	).Scan(&shareMode, &creditsPrice, &fileData, &packName, &metaInfoStr, &encryptionPassword, &packStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query pack listing: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	// If pack is not published, only allow re-download for users who already purchased it
	if packStatus != "published" {
		var purchaseCount int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM user_purchased_packs WHERE user_id = ? AND listing_id = ? AND (is_hidden IS NULL OR is_hidden = 0)`,
			userID, packID,
		).Scan(&purchaseCount)
		if err != nil || purchaseCount == 0 {
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "pack not found"})
			return
		}
		// User has purchased this pack before — allow re-download without charging
		// Record download and return file data directly
		_, _ = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, packID, getClientIP(r))

		metaInfoValue := "{}"
		if metaInfoStr.Valid && metaInfoStr.String != "" {
			metaInfoValue = metaInfoStr.String
		}
		if encryptionPassword != "" {
			w.Header().Set("X-Encryption-Password", encryptionPassword)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.qap"`, sanitizeDownloadFilename(packName)))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
		w.Header().Set("X-Meta-Info", metaInfoValue)
		w.WriteHeader(http.StatusOK)
		w.Write(fileData)
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

	case "per_use", "subscription":
		// For subscription packs, check if user already has an active (non-expired) subscription.
		// If so, allow re-download without charging credits.
		if shareMode == "subscription" {
			hasActiveSubscription := false

			// Get the original purchase date from credits_transactions (include 'renew' as it may be the first record)
			var purchaseDateStr sql.NullString
			err = db.QueryRow(`
				SELECT MIN(created_at) FROM credits_transactions
				WHERE user_id = ? AND listing_id = ? AND transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew_subscription', 'renew')
			`, userID, packID).Scan(&purchaseDateStr)

			// Fallback: if no credits_transactions record, try user_purchased_packs.created_at
			if (err != nil || !purchaseDateStr.Valid || purchaseDateStr.String == "") {
				var uppDate sql.NullString
				_ = db.QueryRow(`SELECT created_at FROM user_purchased_packs WHERE user_id = ? AND listing_id = ?`, userID, packID).Scan(&uppDate)
				if uppDate.Valid && uppDate.String != "" {
					purchaseDateStr = uppDate
					err = nil
				}
			}

			if err == nil && purchaseDateStr.Valid && purchaseDateStr.String != "" {
				// Parse purchase date
				var baseTime time.Time
				if t, err := time.Parse("2006-01-02 15:04:05", purchaseDateStr.String); err == nil {
					baseTime = t
				} else if t, err := time.Parse("2006-01-02T15:04:05Z", purchaseDateStr.String); err == nil {
					baseTime = t
				}

				if !baseTime.IsZero() {
					// Get valid_days from pack listing (default 30 for subscription)
					var validDays int
					db.QueryRow("SELECT COALESCE(valid_days, 0) FROM pack_listings WHERE id = ?", packID).Scan(&validDays)
					if validDays == 0 {
						validDays = 30
					}
					currentExpiry := baseTime.AddDate(0, 0, validDays)

					// Apply all renew transactions to calculate cumulative expiry
					renewRows, err := db.Query(`
						SELECT created_at, COALESCE(description, '')
						FROM credits_transactions
						WHERE user_id = ? AND listing_id = ? AND transaction_type = 'renew'
						ORDER BY created_at ASC
					`, userID, packID)
					if err == nil {
						for renewRows.Next() {
							var renewDateStr, desc string
							if err := renewRows.Scan(&renewDateStr, &desc); err != nil {
								continue
							}
							renewMonths := 1
							if strings.Contains(desc, "yearly") || strings.Contains(desc, "14 month") {
								renewMonths = 14
							} else if strings.Contains(desc, "12 month") {
								renewMonths = 12
							}
							var renewTime time.Time
							if t, err := time.Parse("2006-01-02 15:04:05", renewDateStr); err == nil {
								renewTime = t
							} else if t, err := time.Parse("2006-01-02T15:04:05Z", renewDateStr); err == nil {
								renewTime = t
							}
							if renewTime.IsZero() {
								continue
							}
							if currentExpiry.After(renewTime) {
								currentExpiry = currentExpiry.AddDate(0, renewMonths, 0)
							} else {
								currentExpiry = renewTime.AddDate(0, renewMonths, 0)
							}
						}
						renewRows.Close()
					}

					// Check if subscription is still active
					if time.Now().UTC().Before(currentExpiry) {
						hasActiveSubscription = true
						log.Printf("[DOWNLOAD] User %d has active subscription for pack %d (expires: %s), allowing free re-download",
							userID, packID, currentExpiry.Format("2006-01-02 15:04:05"))
					}
				}
			}

			if hasActiveSubscription {
				// Active subscription: allow download without charging, just increment download count
				_, _ = db.Exec("UPDATE pack_listings SET download_count = download_count + 1 WHERE id = ?", packID)
				_, _ = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, packID, getClientIP(r))
				if err := upsertUserPurchasedPack(userID, packID); err != nil {
					log.Printf("Failed to upsert user purchased pack: %v", err)
				}

				metaInfoValue := "{}"
				if metaInfoStr.Valid && metaInfoStr.String != "" {
					metaInfoValue = metaInfoStr.String
				}
				if encryptionPassword != "" {
					w.Header().Set("X-Encryption-Password", encryptionPassword)
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.qap"`, sanitizeDownloadFilename(packName)))
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
				w.Header().Set("X-Meta-Info", metaInfoValue)
				w.WriteHeader(http.StatusOK)
				w.Write(fileData)
				return
			}
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
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
			 VALUES (?, 'download', ?, ?, ?, ?)`,
			userID, -float64(creditsPrice), packID, fmt.Sprintf("Download pack: %s", packName), getClientIP(r),
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
			"listing_id":          packID,
			"pack_name":           packName,
			"pricing_model":       shareMode,
			"remaining_uses":      0,
			"total_uses":          0,
			"expires_at":          "",
			"subscription_months": 0,
		}

		now := time.Now().UTC()
		switch shareMode {
		case "per_use":
			usageLicense["remaining_uses"] = 1
			usageLicense["total_uses"] = 1
		case "subscription":
			expiresAt := now.AddDate(0, 1, 0)
			usageLicense["expires_at"] = expiresAt.Format(time.RFC3339)
			usageLicense["subscription_months"] = 1
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
			`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
			 VALUES (?, 'download', ?, ?, ?, ?)`,
			userID, -float64(creditsPrice), packID, fmt.Sprintf("Download pack: %s", packName), getClientIP(r),
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

	// Record download in user_downloads table with buyer IP (non-critical, ignore errors)
	_, err = db.Exec("INSERT INTO user_downloads (user_id, listing_id, ip_address) VALUES (?, ?, ?)", userID, packID, getClientIP(r))
	if err != nil {
		log.Printf("Failed to record user download: %v", err)
	}

	// Record/restore user purchased pack (non-critical for download, used for display)
	if err := upsertUserPurchasedPack(userID, packID); err != nil {
		log.Printf("Failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, packID, err)
	}

	// Return file data as binary response with meta_info header
	metaInfoValue := "{}"
	if metaInfoStr.Valid && metaInfoStr.String != "" {
		metaInfoValue = metaInfoStr.String
	}
	if encryptionPassword != "" {
		w.Header().Set("X-Encryption-Password", encryptionPassword)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.qap"`, sanitizeDownloadFilename(packName)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
	w.Header().Set("X-Meta-Info", metaInfoValue)
	w.WriteHeader(http.StatusOK)
	w.Write(fileData)

	// For per_use packs, initialize pack_usage_records on first download (non-critical)
	if shareMode == "per_use" {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 1)`,
			userID, packID,
		)
		if err != nil {
			log.Printf("Failed to initialize pack_usage_records for per_use pack (user=%d, listing=%d): %v", userID, packID, err)
		}
	}
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
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
		 VALUES (?, 'purchase_uses', ?, ?, ?, ?)`,
		userID, -float64(totalCost), packID, fmt.Sprintf("Purchase %d additional uses: %s", req.Quantity, packName), getClientIP(r),
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

	// Record/restore user purchased pack (non-critical, used for display)
	if err := upsertUserPurchasedPack(userID, packID); err != nil {
		log.Printf("Failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, packID, err)
	}

	// Sync pack_usage_records.total_purchased after successful purchase
	_, err = db.Exec(
		`INSERT OR IGNORE INTO pack_usage_records (user_id, listing_id, used_count, total_purchased) VALUES (?, ?, 0, 0)`,
		userID, packID,
	)
	if err != nil {
		log.Printf("[PURCHASE-USES] failed to insert pack_usage_records (user=%d, listing=%d): %v", userID, packID, err)
	}
	_, err = db.Exec(
		`UPDATE pack_usage_records SET total_purchased = total_purchased + ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND listing_id = ?`,
		req.Quantity, userID, packID,
	)
	if err != nil {
		log.Printf("[PURCHASE-USES] failed to update total_purchased (user=%d, listing=%d): %v", userID, packID, err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"remaining_uses":  req.Quantity,
		"credits_deducted": totalCost,
	})
}

// handleRenewSubscription handles POST /api/packs/{id}/renew
// Accepts JSON body with "months" field (1 for monthly, 12 for yearly with bonus).
// Yearly: pays 12 months credits, gets 14 months duration.
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

	// Parse request body for months
	var reqBody struct {
		Months int `json:"months"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&reqBody)
	}
	if reqBody.Months == 0 {
		reqBody.Months = 1 // default to monthly
	}
	if reqBody.Months < 1 || reqBody.Months > 36 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "months must be between 1 and 36"})
		return
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

	// Verify share_mode is subscription
	if shareMode != "subscription" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "pack is not subscription type"})
		return
	}

	// Calculate total cost and granted months
	totalCost := creditsPrice * reqBody.Months
	grantedMonths := reqBody.Months
	// Yearly bonus: for every 12 months purchased, grant 2 bonus months
	yearlyBlocks := reqBody.Months / 12
	if yearlyBlocks > 0 {
		grantedMonths = reqBody.Months + yearlyBlocks*2
	}

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

	// Calculate new expires_at: extend from current expiry if still valid, otherwise from now
	now := time.Now().UTC()
	baseTime := now

	// Query the current effective expiry for this user+pack from the latest renew or original purchase
	var currentExpiresAt time.Time
	var latestTxDate string
	var latestDesc string
	err = db.QueryRow(`
		SELECT created_at, COALESCE(description, '')
		FROM credits_transactions
		WHERE user_id = ? AND listing_id = ? AND transaction_type = 'renew'
		ORDER BY created_at DESC LIMIT 1
	`, userID, packID).Scan(&latestTxDate, &latestDesc)
	if err == nil && latestTxDate != "" {
		// Determine previous renewal months from description
		prevMonths := 1
		if strings.Contains(latestDesc, "yearly") || strings.Contains(latestDesc, "14 month") {
			prevMonths = 14
		} else if strings.Contains(latestDesc, "12 month") {
			prevMonths = 12
		}
		if t, parseErr := time.Parse("2006-01-02 15:04:05", latestTxDate); parseErr == nil {
			currentExpiresAt = t.AddDate(0, prevMonths, 0)
		} else if t, parseErr := time.Parse("2006-01-02T15:04:05Z", latestTxDate); parseErr == nil {
			currentExpiresAt = t.AddDate(0, prevMonths, 0)
		}
	}
	// If no renew record, check original purchase/download date
	if currentExpiresAt.IsZero() {
		var purchaseDate string
		_ = db.QueryRow(`
			SELECT created_at FROM credits_transactions
			WHERE user_id = ? AND listing_id = ? AND transaction_type IN ('purchase', 'download')
			ORDER BY created_at ASC LIMIT 1
		`, userID, packID).Scan(&purchaseDate)
		if purchaseDate != "" {
			if t, parseErr := time.Parse("2006-01-02 15:04:05", purchaseDate); parseErr == nil {
				currentExpiresAt = t.AddDate(0, 1, 0) // initial subscription = 1 month
			} else if t, parseErr := time.Parse("2006-01-02T15:04:05Z", purchaseDate); parseErr == nil {
				currentExpiresAt = t.AddDate(0, 1, 0)
			}
		}
	}

	// If current subscription is still valid, extend from its expiry; otherwise extend from now
	if !currentExpiresAt.IsZero() && currentExpiresAt.After(now) {
		baseTime = currentExpiresAt
	}
	expiresAt := baseTime.AddDate(0, grantedMonths, 0)

	// Record credits transaction
	desc := fmt.Sprintf("Renew subscription (%d months): %s", reqBody.Months, packName)
	if reqBody.Months == 12 {
		desc = fmt.Sprintf("Renew subscription (yearly, pay 12 get 14 months): %s", packName)
	}
	_, err = tx.Exec(
		`INSERT INTO credits_transactions (user_id, transaction_type, amount, listing_id, description, ip_address)
		 VALUES (?, 'renew', ?, ?, ?, ?)`,
		userID, -float64(totalCost), packID, desc, getClientIP(r),
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

	// Record/restore user purchased pack (non-critical, used for display)
	if err := upsertUserPurchasedPack(userID, packID); err != nil {
		log.Printf("Failed to upsert user purchased pack (user=%d, listing=%d): %v", userID, packID, err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":             true,
		"expires_at":          expiresAt.Format(time.RFC3339),
		"subscription_months": grantedMonths,
		"credits_deducted":    totalCost,
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
	if err := rows.Err(); err != nil {
		log.Printf("[handleListTransactions] rows iteration error: %v", err)
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

// handleAdminList returns all admins with id, username, role, permissions, created_at.
// GET /api/admin/admins
func handleAdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, username, role, COALESCE(permissions, ''), created_at FROM admin_credentials ORDER BY id")
	if err != nil {
		log.Printf("Failed to query admins: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	type adminInfo struct {
		ID          int64    `json:"id"`
		Username    string   `json:"username"`
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
		CreatedAt   string   `json:"created_at"`
	}

	var admins []adminInfo
	for rows.Next() {
		var a adminInfo
		var permsStr string
		if err := rows.Scan(&a.ID, &a.Username, &a.Role, &permsStr, &a.CreatedAt); err != nil {
			log.Printf("Failed to scan admin row: %v", err)
			continue
		}
		if a.ID == 1 {
			a.Permissions = allPermissions
		} else if permsStr != "" {
			a.Permissions = strings.Split(permsStr, ",")
		} else {
			a.Permissions = []string{}
		}
		admins = append(admins, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminList] rows iteration error: %v", err)
	}
	if admins == nil {
		admins = []adminInfo{}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"admins": admins})
}

// handleCreateAdmin creates a new admin with specified permissions.
// POST /api/admin/admins
func handleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string   `json:"username"`
		Password    string   `json:"password"`
		Permissions []string `json:"permissions"`
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

	// Validate permissions
	validPerms := map[string]bool{"categories": true, "marketplace": true, "authors": true, "review": true, "settings": true, "customers": true}
	var filteredPerms []string
	for _, p := range req.Permissions {
		if validPerms[p] {
			filteredPerms = append(filteredPerms, p)
		}
	}
	permsStr := strings.Join(filteredPerms, ",")

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
	result, err := db.Exec("INSERT INTO admin_credentials (username, password_hash, role, permissions) VALUES (?, ?, 'regular', ?)", req.Username, passwordHash, permsStr)
	if err != nil {
		log.Printf("Failed to create admin: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	newID, _ := result.LastInsertId()
	var createdAt string
	db.QueryRow("SELECT created_at FROM admin_credentials WHERE id = ?", newID).Scan(&createdAt)

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":          newID,
		"username":    req.Username,
		"role":        "regular",
		"permissions": filteredPerms,
		"created_at":  createdAt,
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
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.status, pl.meta_info, pl.created_at
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
			&sourceName, &authorName, &p.ShareMode, &p.CreditsPrice, &p.DownloadCount, &p.Status, &metaInfoStr, &p.CreatedAt)
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
	if err := rows.Err(); err != nil {
		log.Printf("[handlePendingList] rows iteration error: %v", err)
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

	creditCashRate := getSetting("credit_cash_rate")
	if creditCashRate == "" {
		creditCashRate = "0"
	}

	revenueSplitPublisherPct := getSetting("revenue_split_publisher_pct")
	if revenueSplitPublisherPct == "" {
		revenueSplitPublisherPct = "70"
	}
	pubPctVal, _ := strconv.ParseFloat(revenueSplitPublisherPct, 64)
	revenueSplitPlatformPct := strconv.FormatFloat(100-pubPctVal, 'f', -1, 64)

	// Query withdrawal fee rates from settings
	feeRatePaypal := getSetting("fee_rate_paypal")
	if feeRatePaypal == "" {
		feeRatePaypal = "0"
	}
	feeRateWechat := getSetting("fee_rate_wechat")
	if feeRateWechat == "" {
		feeRateWechat = "0"
	}
	feeRateAlipay := getSetting("fee_rate_alipay")
	if feeRateAlipay == "" {
		feeRateAlipay = "0"
	}
	feeRateCheck := getSetting("fee_rate_check")
	if feeRateCheck == "" {
		feeRateCheck = "0"
	}
	feeRateWireTransfer := getSetting("fee_rate_wire_transfer")
	if feeRateWireTransfer == "" {
		feeRateWireTransfer = "0"
	}
	feeRateBankCardUS := getSetting("fee_rate_bank_card_us")
	if feeRateBankCardUS == "" {
		feeRateBankCardUS = "0"
	}
	feeRateBankCardEU := getSetting("fee_rate_bank_card_eu")
	if feeRateBankCardEU == "" {
		feeRateBankCardEU = "0"
	}
	feeRateBankCardCN := getSetting("fee_rate_bank_card_cn")
	if feeRateBankCardCN == "" {
		feeRateBankCardCN = "0"
	}

	// Get admin info from session
	adminID := getSessionAdminID(r)
	permissions := getAdminPermissions(adminID)

	// Convert permissions to JSON for JS consumption (use template.JS to avoid HTML escaping)
	permsJSON, _ := json.Marshal(permissions)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	adminTmpl.Execute(w, map[string]interface{}{
		"InitialCredits":  initialCredits,
		"CreditCashRate":  creditCashRate,
		"FeeRatePaypal":   feeRatePaypal,
		"FeeRateWechat":   feeRateWechat,
		"FeeRateAlipay":   feeRateAlipay,
		"FeeRateCheck":         feeRateCheck,
		"FeeRateWireTransfer": feeRateWireTransfer,
		"FeeRateBankCardUS":   feeRateBankCardUS,
		"FeeRateBankCardEU":   feeRateBankCardEU,
		"FeeRateBankCardCN":          feeRateBankCardCN,
		"RevenueSplitPublisherPct":   revenueSplitPublisherPct,
		"RevenueSplitPlatformPct":    revenueSplitPlatformPct,
		"AdminID":                    adminID,
		"PermissionsJSON":            template.JS(string(permsJSON)),
		"DefaultLang":                getSetting("default_language"),
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

// handleSetDefaultLanguage updates the default_language setting.
// POST /admin/api/settings/default-language
func handleSetDefaultLanguage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	value := r.FormValue("value")
	if value != "zh-CN" && value != "en-US" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be zh-CN or en-US"})
		return
	}
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('default_language', ?)", value)
	if err != nil {
		log.Printf("Failed to update default_language: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	// Update runtime default
	if value == "en-US" {
		i18n.DefaultLang = i18n.EnUS
	} else {
		i18n.DefaultLang = i18n.ZhCN
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

func handleSetCreditCashRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}

	// Validate that value is a non-negative number
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil || rate < 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "value must be a non-negative number"})
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('credit_cash_rate', ?)", value)
	if err != nil {
		log.Printf("Failed to update credit_cash_rate: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "value": value})
}

// handleAdminSaveRevenueSplit saves the publisher revenue split percentage.
// POST /admin/api/settings/revenue-split
func handleAdminSaveRevenueSplit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		PublisherPct float64 `json:"publisher_pct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.PublisherPct < 0 || req.PublisherPct > 100 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "publisher_pct must be between 0 and 100"})
		return
	}

	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "revenue_split_publisher_pct", fmt.Sprintf("%g", req.PublisherPct))
	if err != nil {
		log.Printf("Failed to save revenue_split_publisher_pct: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	log.Printf("[ADMIN] revenue split updated: publisher=%.0f%%, platform=%.0f%%", req.PublisherPct, 100-req.PublisherPct)
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleAdminSaveWithdrawalFees saves withdrawal fee rates for each payment type.
// POST /admin/api/settings/withdrawal-fees
func handleAdminSaveWithdrawalFees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		PaypalFeeRate       float64 `json:"paypal_fee_rate"`
		WechatFeeRate       float64 `json:"wechat_fee_rate"`
		AlipayFeeRate       float64 `json:"alipay_fee_rate"`
		CheckFeeRate        float64 `json:"check_fee_rate"`
		WireTransferFeeRate float64 `json:"wire_transfer_fee_rate"`
		BankCardUSFeeRate   float64 `json:"bank_card_us_fee_rate"`
		BankCardEUFeeRate   float64 `json:"bank_card_eu_fee_rate"`
		BankCardCNFeeRate   float64 `json:"bank_card_cn_fee_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	feeRates := map[string]float64{
		"fee_rate_paypal":        req.PaypalFeeRate,
		"fee_rate_wechat":        req.WechatFeeRate,
		"fee_rate_alipay":        req.AlipayFeeRate,
		"fee_rate_check":         req.CheckFeeRate,
		"fee_rate_wire_transfer": req.WireTransferFeeRate,
		"fee_rate_bank_card_us":  req.BankCardUSFeeRate,
		"fee_rate_bank_card_eu":  req.BankCardEUFeeRate,
		"fee_rate_bank_card_cn":  req.BankCardCNFeeRate,
	}

	for key, rate := range feeRates {
		if rate < 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": key + " must be non-negative"})
			return
		}
	}

	for key, rate := range feeRates {
		_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, fmt.Sprintf("%g", rate))
		if err != nil {
			log.Printf("Failed to save %s: %v", key, err)
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleAdminGetWithdrawals returns a list of withdrawal records, optionally filtered by status.
// GET /admin/api/withdrawals?status=pending|paid
func handleAdminGetWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	statusFilter := r.URL.Query().Get("status")
	authorFilter := r.URL.Query().Get("author")
	var rows *sql.Rows
	var err error

	query := `SELECT id, user_id, display_name, credits_amount, cash_rate, cash_amount,
	       payment_type, payment_details, fee_rate, fee_amount, net_amount, status, created_at
	FROM withdrawal_records`
	var conditions []string
	var args []interface{}

	if statusFilter == "pending" || statusFilter == "paid" {
		conditions = append(conditions, "status = ?")
		args = append(args, statusFilter)
	}
	if authorFilter != "" {
		conditions = append(conditions, "display_name LIKE ?")
		args = append(args, "%"+authorFilter+"%")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"
	rows, err = db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query withdrawal records: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	withdrawals := []WithdrawalRequest{}
	for rows.Next() {
		var wr WithdrawalRequest
		if err := rows.Scan(&wr.ID, &wr.UserID, &wr.DisplayName, &wr.CreditsAmount, &wr.CashRate, &wr.CashAmount,
			&wr.PaymentType, &wr.PaymentDetails, &wr.FeeRate, &wr.FeeAmount, &wr.NetAmount, &wr.Status, &wr.CreatedAt); err != nil {
			log.Printf("Failed to scan withdrawal record: %v", err)
			continue
		}
		withdrawals = append(withdrawals, wr)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminGetWithdrawals] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"withdrawals": withdrawals})
}

// handleAdminApproveWithdrawals batch-approves withdrawal records by setting status from pending to paid.
// POST /admin/api/withdrawals/approve
func handleAdminApproveWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "ids is required"})
		return
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(req.IDs))
	args := make([]interface{}, len(req.IDs))
	for i, id := range req.IDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("UPDATE withdrawal_records SET status = 'paid' WHERE id IN (%s) AND status = 'pending'",
		strings.Join(placeholders, ","))

	result, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to approve withdrawals: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	updated, _ := result.RowsAffected()
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "updated": updated})
}

// handleAdminExportWithdrawals exports selected withdrawal records as an Excel file.
// GET /admin/api/withdrawals/export?ids=1,2,3
func handleAdminExportWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	idsParam := r.URL.Query().Get("ids")
	if strings.TrimSpace(idsParam) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "ids parameter is required"})
		return
	}

	idStrs := strings.Split(idsParam, ",")
	var ids []interface{}
	var placeholders []string
	for _, s := range idStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid id: " + s})
			return
		}
		ids = append(ids, id)
		placeholders = append(placeholders, "?")
	}

	if len(ids) == 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "no valid ids provided"})
		return
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, display_name, credits_amount, cash_rate, cash_amount,
		       payment_type, payment_details, fee_rate, fee_amount, net_amount, status, created_at
		FROM withdrawal_records
		WHERE id IN (%s)
		ORDER BY id`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, ids...)
	if err != nil {
		log.Printf("Failed to query withdrawal records for export: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	defer rows.Close()

	var withdrawals []WithdrawalRequest
	for rows.Next() {
		var wr WithdrawalRequest
		if err := rows.Scan(&wr.ID, &wr.UserID, &wr.DisplayName, &wr.CreditsAmount, &wr.CashRate, &wr.CashAmount,
			&wr.PaymentType, &wr.PaymentDetails, &wr.FeeRate, &wr.FeeAmount, &wr.NetAmount, &wr.Status, &wr.CreatedAt); err != nil {
			log.Printf("Failed to scan withdrawal record for export: %v", err)
			continue
		}
		withdrawals = append(withdrawals, wr)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminExportWithdrawals] rows iteration error: %v", err)
	}

	// Create Excel file
	lang := i18n.DetectLang(r)
	f := excelize.NewFile()
	defer f.Close()
	sheetName := i18n.T(lang, "excel_withdraw_sheet")
	f.SetSheetName("Sheet1", sheetName)

	// Write header row
	headers := []string{i18n.T(lang, "excel_author_name"), i18n.T(lang, "excel_payment_method"), i18n.T(lang, "excel_payment_detail"), i18n.T(lang, "excel_withdraw_amount"), i18n.T(lang, "excel_fee_rate"), i18n.T(lang, "excel_fee_amount"), i18n.T(lang, "excel_net_amount")}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// Payment type labels
	typeLabels := map[string]string{
		"paypal": "PayPal", "wechat": i18n.T(lang, "excel_wechat"), "alipay": "AliPay", "check": i18n.T(lang, "excel_check"),
	}

	// Write data rows
	for rowIdx, wr := range withdrawals {
		row := rowIdx + 2
		typeLabel := typeLabels[wr.PaymentType]
		if typeLabel == "" {
			typeLabel = wr.PaymentType
		}
		feeRatePercent := fmt.Sprintf("%.2f%%", wr.FeeRate*100)

		cells := []interface{}{wr.DisplayName, typeLabel, wr.PaymentDetails, wr.CashAmount, feeRatePercent, wr.FeeAmount, wr.NetAmount}
		for i, val := range cells {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("Failed to write Excel file: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate Excel file"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="withdrawals_export.xlsx"`)
	w.Write(buf.Bytes())
}




// cash_amount = credits_amount × cash_rate
// fee_amount = cash_amount × fee_rate / 100
// net_amount = cash_amount - fee_amount
func calculateWithdrawalFee(creditsAmount, cashRate, feeRate float64) (cashAmount, feeAmount, netAmount float64) {
	cashAmount = creditsAmount * cashRate
	feeAmount = cashAmount * feeRate / 100
	netAmount = cashAmount - feeAmount
	return
}

// handleAuthorWithdraw processes author credit withdrawal requests.
// POST /user/author/withdraw
// Supports both form submit (redirect) and AJAX (JSON response).
func handleAuthorWithdraw(w http.ResponseWriter, r *http.Request) {
	isAjax := r.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
		strings.Contains(r.Header.Get("Accept"), "application/json")

	withdrawError := func(code string, msg string) {
		if isAjax {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": code, "message": msg})
		} else {
			http.Redirect(w, r, "/user/?error="+code, http.StatusFound)
		}
	}
	withdrawSuccess := func() {
		if isAjax {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
		} else {
			http.Redirect(w, r, "/user/?success=withdraw", http.StatusFound)
		}
	}

	if r.Method != http.MethodPost {
		log.Printf("[AUTHOR-WITHDRAW] rejected: method=%s (expected POST)", r.Method)
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] rejected: invalid user ID %q", userIDStr)
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	log.Printf("[AUTHOR-WITHDRAW] user %d: starting withdrawal request (isAjax=%v)", userID, isAjax)

	lang := i18n.DetectLang(r)

	// Payment info pre-check: user must have payment info set before withdrawing
	var paymentType, paymentDetailsStr string
	err = db.QueryRow("SELECT payment_type, payment_details FROM user_payment_info WHERE user_id = ?", userID).Scan(&paymentType, &paymentDetailsStr)
	if err == sql.ErrNoRows {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - no payment info", userID)
		withdrawError("no_payment_info", i18n.T(lang, "withdraw_no_payment"))
		return
	}
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query payment info for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	// Get user's display_name
	var displayName string
	err = db.QueryRow("SELECT display_name FROM users WHERE id = ?", userID).Scan(&displayName)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query display_name for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	// Verify user is an author (has at least one pack listing)
	var authorPackCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pack_listings WHERE user_id = ?", userID).Scan(&authorPackCount)
	if err != nil || authorPackCount == 0 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - not author (count=%d, err=%v)", userID, authorPackCount, err)
		withdrawError("not_author", i18n.T(lang, "withdraw_not_author"))
		return
	}

	// Parse and validate credits_amount
	creditsAmountStr := r.FormValue("credits_amount")
	creditsAmount, err := strconv.ParseFloat(creditsAmountStr, 64)
	if err != nil || creditsAmount <= 0 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - invalid amount %q (err=%v)", userID, creditsAmountStr, err)
		withdrawError("invalid_withdraw_amount", i18n.T(lang, "withdraw_invalid_amount"))
		return
	}

	// Query credit_cash_rate from settings
	cashRateStr := getSetting("credit_cash_rate")
	cashRate, _ := strconv.ParseFloat(cashRateStr, 64)
	if cashRate <= 0 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - withdraw disabled (cashRate=%s)", userID, cashRateStr)
		withdrawError("withdraw_disabled", i18n.T(lang, "withdraw_not_open"))
		return
	}

	// Read fee rate for the user's payment type from settings (default to 0 if not found)
	feeRateStr := getSetting("fee_rate_" + paymentType)
	feeRate, _ := strconv.ParseFloat(feeRateStr, 64)
	if feeRate < 0 {
		feeRate = 0
	}

	// Calculate unwithdrawn credits: total revenue minus total withdrawn (with revenue split)
	// Must match the dashboard query exactly: purchase, download, purchase_uses, renew with amount < 0
	var totalRevenue float64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(ABS(ct.amount)), 0)
		FROM credits_transactions ct
		JOIN pack_listings pl ON ct.listing_id = pl.id
		WHERE pl.user_id = ? AND ct.transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		  AND ct.amount < 0
	`, userID).Scan(&totalRevenue)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query total revenue for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	// Apply revenue split: publisher only gets their configured share
	splitPctStr := getSetting("revenue_split_publisher_pct")
	splitPct, _ := strconv.ParseFloat(splitPctStr, 64)
	if splitPct <= 0 {
		splitPct = 70 // default 70%
	}
	publisherRevenue := totalRevenue * splitPct / 100

	var totalWithdrawn float64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(credits_amount), 0)
		FROM withdrawal_records
		WHERE user_id = ?
	`, userID).Scan(&totalWithdrawn)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to query total withdrawn for user %d: %v", userID, err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	unwithdrawn := publisherRevenue - totalWithdrawn
	if unwithdrawn < 0 {
		unwithdrawn = 0
	}

	log.Printf("[AUTHOR-WITHDRAW] user %d: amount=%.2f, totalRevenue=%.2f, splitPct=%.0f, publisherRevenue=%.2f, totalWithdrawn=%.2f, unwithdrawn=%.2f",
		userID, creditsAmount, totalRevenue, splitPct, publisherRevenue, totalWithdrawn, unwithdrawn)

	// Verify credits_amount does not exceed unwithdrawn
	if creditsAmount > unwithdrawn {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - amount %.2f exceeds unwithdrawn %.2f", userID, creditsAmount, unwithdrawn)
		withdrawError("withdraw_exceeds_balance", i18n.T(lang, "withdraw_exceeds"))
		return
	}

	// Calculate cash_amount, fee_amount, net_amount using calculateWithdrawalFee
	cashAmount, feeAmount, netAmount := calculateWithdrawalFee(creditsAmount, cashRate, feeRate)

	log.Printf("[AUTHOR-WITHDRAW] user %d: cashRate=%.4f, feeRate=%.2f, cashAmount=%.2f, feeAmount=%.2f, netAmount=%.2f",
		userID, cashRate, feeRate, cashAmount, feeAmount, netAmount)

	// Minimum withdrawal: net_amount must be at least 100 元
	if netAmount < 100 {
		log.Printf("[AUTHOR-WITHDRAW] user %d: rejected - netAmount %.2f < 100", userID, netAmount)
		withdrawError("withdraw_below_minimum", i18n.T(lang, "withdraw_below_min_net"))
		return
	}

	// Transaction: INSERT withdrawal_records + UPDATE credits_balance
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to begin transaction: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}
	defer tx.Rollback()

	// Store fee_rate as decimal (e.g. 0.03 for 3%) for consistency with admin frontend display
	feeRateDecimal := feeRate / 100

	_, err = tx.Exec(
		`INSERT INTO withdrawal_records (user_id, credits_amount, cash_rate, cash_amount, payment_type, payment_details, fee_rate, fee_amount, net_amount, status, display_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
		userID, creditsAmount, cashRate, cashAmount, paymentType, paymentDetailsStr, feeRateDecimal, feeAmount, netAmount, displayName,
	)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to insert withdrawal record: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	_, err = tx.Exec(
		`UPDATE users SET credits_balance = credits_balance - ? WHERE id = ?`,
		creditsAmount, userID,
	)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to update credits_balance: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[AUTHOR-WITHDRAW] failed to commit transaction: %v", err)
		withdrawError("internal", i18n.T(lang, "system_error"))
		return
	}

	log.Printf("[AUTHOR-WITHDRAW] user %d withdrew %.2f credits (cash=%.2f, fee=%.2f, net=%.2f, rate=%.4f, feeRate=%.4f, paymentType=%s)", userID, creditsAmount, cashAmount, feeAmount, netAmount, cashRate, feeRate, paymentType)
	withdrawSuccess()
}

// WithdrawalRecord holds a single withdrawal record for the withdrawal records page.
type WithdrawalRecord struct {
	ID            int64
	CreditsAmount float64
	CashRate      float64
	CashAmount    float64
	CreatedAt     string
}

// handleAuthorWithdrawRecords renders the withdrawal records page for authors.
// GET /user/author/withdrawals
// Supports JSON response (Accept: application/json) for modal display.
func handleAuthorWithdrawRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	isAjax := strings.Contains(r.Header.Get("Accept"), "application/json")

	rows, err := db.Query(`
		SELECT id, credits_amount, cash_rate, cash_amount, fee_rate, fee_amount, net_amount, status, created_at
		FROM withdrawal_records
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		log.Printf("[AUTHOR-WITHDRAWALS] failed to query withdrawal records for user %d: %v", userID, err)
		if isAjax {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		} else {
			http.Error(w, i18n.T(i18n.DetectLang(r), "load_withdraw_records_failed"), http.StatusInternalServerError)
		}
		return
	}
	defer rows.Close()

	type jsonRecord struct {
		ID            int64   `json:"id"`
		CreditsAmount float64 `json:"credits_amount"`
		CashRate      float64 `json:"cash_rate"`
		CashAmount    float64 `json:"cash_amount"`
		FeeRate       float64 `json:"fee_rate"`
		FeeAmount     float64 `json:"fee_amount"`
		NetAmount     float64 `json:"net_amount"`
		Status        string  `json:"status"`
		CreatedAt     string  `json:"created_at"`
	}

	var records []WithdrawalRecord
	var jsonRecords []jsonRecord
	var totalCash float64
	for rows.Next() {
		var jr jsonRecord
		if err := rows.Scan(&jr.ID, &jr.CreditsAmount, &jr.CashRate, &jr.CashAmount, &jr.FeeRate, &jr.FeeAmount, &jr.NetAmount, &jr.Status, &jr.CreatedAt); err != nil {
			log.Printf("[AUTHOR-WITHDRAWALS] failed to scan withdrawal record row: %v", err)
			continue
		}
		totalCash += jr.CashAmount
		jsonRecords = append(jsonRecords, jr)
		records = append(records, WithdrawalRecord{
			ID: jr.ID, CreditsAmount: jr.CreditsAmount, CashRate: jr.CashRate,
			CashAmount: jr.CashAmount, CreatedAt: jr.CreatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAuthorWithdrawRecords] rows iteration error: %v", err)
	}

	log.Printf("[AUTHOR-WITHDRAWALS] user %d: %d withdrawal records, total cash=%.2f", userID, len(records), totalCash)

	if isAjax {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"records":    jsonRecords,
			"total_cash": totalCash,
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.UserWithdrawalRecordsTmpl.Execute(w, struct {
		Records   []WithdrawalRecord
		TotalCash float64
	}{Records: records, TotalCash: totalCash})
}

// handleAuthorEditPack processes author pack metadata edit requests.
// POST /user/author/edit-pack
func handleAuthorEditPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	// Parse form values
	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	packName := r.FormValue("pack_name")
	packDescription := r.FormValue("pack_description")
	shareMode := r.FormValue("share_mode")
	creditsPriceStr := r.FormValue("credits_price")
	creditsPrice, err := strconv.Atoi(creditsPriceStr)
	if err != nil {
		creditsPrice = 0
	}

	// Verify listing belongs to current user
	var ownerID int64
	err = db.QueryRow("SELECT user_id FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID)
	if err != nil {
		log.Printf("[AUTHOR-EDIT-PACK] listing %d not found: %v", listingID, err)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if ownerID != userID {
		log.Printf("[AUTHOR-EDIT-PACK] user %d attempted to edit listing %d owned by user %d", userID, listingID, ownerID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Validate share_mode and pricing using existing validatePricingParams
	if errMsg := validatePricingParams(shareMode, creditsPrice); errMsg != "" {
		log.Printf("[AUTHOR-EDIT-PACK] pricing validation failed for listing %d: %s", listingID, errMsg)
		http.Redirect(w, r, "/user/?error=invalid_pricing", http.StatusFound)
		return
	}

	// For free mode, force credits_price to 0
	if shareMode == "free" {
		creditsPrice = 0
	}

	// Update pack_listings: set new metadata and reset review status
	_, err = db.Exec(`
		UPDATE pack_listings
		SET pack_name = ?, pack_description = ?, share_mode = ?, credits_price = ?,
		    status = 'pending', reviewed_by = NULL, reviewed_at = NULL
		WHERE id = ? AND user_id = ?
	`, packName, packDescription, shareMode, creditsPrice, listingID, userID)
	if err != nil {
		log.Printf("[AUTHOR-EDIT-PACK] failed to update listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	log.Printf("[AUTHOR-EDIT-PACK] user %d updated listing %d: name=%s mode=%s price=%d", userID, listingID, packName, shareMode, creditsPrice)
	http.Redirect(w, r, "/user/", http.StatusFound)
}

// handleAuthorDeletePack allows an author to delete their own rejected pack listing.
// POST /user/author/delete-pack
func handleAuthorDeletePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	// Verify listing belongs to current user and is rejected
	var ownerID int64
	var status string
	err = db.QueryRow("SELECT user_id, status FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID, &status)
	if err != nil {
		log.Printf("[AUTHOR-DELETE-PACK] listing %d not found: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=not_found", http.StatusFound)
		return
	}
	if ownerID != userID {
		log.Printf("[AUTHOR-DELETE-PACK] user %d attempted to delete listing %d owned by user %d", userID, listingID, ownerID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if status != "rejected" {
		log.Printf("[AUTHOR-DELETE-PACK] user %d attempted to delete listing %d with status %q (only rejected allowed)", userID, listingID, status)
		http.Redirect(w, r, "/user/?error=not_rejected", http.StatusFound)
		return
	}

	// Delete the pack listing
	_, err = db.Exec("DELETE FROM pack_listings WHERE id = ? AND user_id = ? AND status = 'rejected'", listingID, userID)
	if err != nil {
		log.Printf("[AUTHOR-DELETE-PACK] failed to delete listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=delete_failed", http.StatusFound)
		return
	}

	log.Printf("[AUTHOR-DELETE-PACK] user %d deleted rejected listing %d", userID, listingID)
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

// handleAuthorDelistPack allows an author to delist their own published pack listing.
// POST /user/author/delist-pack
func handleAuthorDelistPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user/", http.StatusFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}

	listingIDStr := r.FormValue("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		http.Redirect(w, r, "/user/?error=invalid_listing", http.StatusFound)
		return
	}

	// Verify listing belongs to current user and is published
	var ownerID int64
	var status string
	err = db.QueryRow("SELECT user_id, status FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID, &status)
	if err != nil {
		log.Printf("[AUTHOR-DELIST-PACK] listing %d not found: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=not_found", http.StatusFound)
		return
	}
	if ownerID != userID {
		log.Printf("[AUTHOR-DELIST-PACK] user %d attempted to delist listing %d owned by user %d", userID, listingID, ownerID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if status != "published" {
		log.Printf("[AUTHOR-DELIST-PACK] user %d attempted to delist listing %d with status %q (only published allowed)", userID, listingID, status)
		http.Redirect(w, r, "/user/?error=not_published", http.StatusFound)
		return
	}

	// Update the pack listing status to delisted
	_, err = db.Exec("UPDATE pack_listings SET status = 'delisted' WHERE id = ? AND user_id = ? AND status = 'published'", listingID, userID)
	if err != nil {
		log.Printf("[AUTHOR-DELIST-PACK] failed to delist listing %d: %v", listingID, err)
		http.Redirect(w, r, "/user/?error=internal", http.StatusFound)
		return
	}

	log.Printf("[AUTHOR-DELIST-PACK] user %d delisted listing %d", userID, listingID)
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}


// handleAuthorPackPurchases returns JSON with purchase details for a specific pack listing.
// GET /user/author/pack-purchases?listing_id=123
func handleAuthorPackPurchases(w http.ResponseWriter, r *http.Request) {
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

	listingIDStr := r.URL.Query().Get("listing_id")
	listingID, err := strconv.ParseInt(listingIDStr, 10, 64)
	if err != nil || listingID <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid listing_id"})
		return
	}

	// Verify listing belongs to current user
	var ownerID int64
	err = db.QueryRow("SELECT user_id FROM pack_listings WHERE id = ?", listingID).Scan(&ownerID)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing not found"})
		return
	}
	if ownerID != userID {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	// Get revenue split percentage
	splitPctStr := getSetting("revenue_split_publisher_pct")
	splitPct, _ := strconv.ParseFloat(splitPctStr, 64)
	if splitPct <= 0 {
		splitPct = 70
	}

	// Query purchase transactions for this listing
	rows, err := db.Query(`
		SELECT ct.id, COALESCE(u.email, u.display_name, 'unknown') as buyer,
		       ABS(ct.amount) as amount, ct.created_at, COALESCE(ct.description, '')
		FROM credits_transactions ct
		LEFT JOIN users u ON ct.user_id = u.id
		WHERE ct.listing_id = ? AND ct.transaction_type IN ('purchase', 'download', 'purchase_uses', 'renew')
		  AND ct.amount < 0
		ORDER BY ct.created_at DESC
	`, listingID)
	if err != nil {
		log.Printf("[AUTHOR-PACK-PURCHASES] failed to query purchases for listing %d: %v", listingID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	defer rows.Close()

	type PurchaseDetail struct {
		ID            int64   `json:"id"`
		Buyer         string  `json:"buyer"`
		Amount        float64 `json:"amount"`
		AuthorEarning float64 `json:"author_earning"`
		CreatedAt     string  `json:"created_at"`
		Description   string  `json:"description"`
	}

	var purchases []PurchaseDetail
	for rows.Next() {
		var p PurchaseDetail
		if err := rows.Scan(&p.ID, &p.Buyer, &p.Amount, &p.CreatedAt, &p.Description); err != nil {
			log.Printf("[AUTHOR-PACK-PURCHASES] failed to scan row: %v", err)
			continue
		}
		p.AuthorEarning = p.Amount * splitPct / 100
		purchases = append(purchases, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[AUTHOR-PACK-PURCHASES] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"purchases": purchases,
		"split_pct": splitPct,
	})
}

// handleAdminMarketplaceList lists published packs for admin marketplace management.
// GET /api/admin/marketplace?category_id=&share_mode=&sort=downloads&order=desc
func handleAdminMarketplaceList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	statusParam := r.URL.Query().Get("status")
	if statusParam == "" {
		statusParam = "published"
	}
	if statusParam != "published" && statusParam != "delisted" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_status", "message": "status must be published or delisted"})
		return
	}

	query := `
		SELECT pl.id, pl.user_id, pl.category_id, c.name, pl.pack_name, pl.pack_description,
		       pl.source_name, pl.author_name, pl.share_mode, pl.credits_price, pl.download_count, pl.status, pl.meta_info, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.status = ?`
	args := []interface{}{statusParam}

	// Filter by category
	if catID := r.URL.Query().Get("category_id"); catID != "" {
		id, err := strconv.ParseInt(catID, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid category_id"})
			return
		}
		query += " AND pl.category_id = ?"
		args = append(args, id)
	}

	// Filter by share_mode
	if mode := r.URL.Query().Get("share_mode"); mode != "" {
		query += " AND pl.share_mode = ?"
		args = append(args, mode)
	}

	// Sort
	sortField := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}
	switch sortField {
	case "downloads":
		query += " ORDER BY pl.download_count " + order
	case "price":
		query += " ORDER BY pl.credits_price " + order
	case "name":
		query += " ORDER BY pl.pack_name " + order
	default:
		query += " ORDER BY pl.download_count DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query marketplace listings: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	listings := []PackListingInfo{}
	for rows.Next() {
		var l PackListingInfo
		var desc, sourceName, authorName, metaInfoStr sql.NullString
		if err := rows.Scan(&l.ID, &l.UserID, &l.CategoryID, &l.CategoryName,
			&l.PackName, &desc, &sourceName, &authorName,
			&l.ShareMode, &l.CreditsPrice, &l.DownloadCount, &l.Status, &metaInfoStr, &l.CreatedAt); err != nil {
			log.Printf("Failed to scan marketplace listing: %v", err)
			continue
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
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminMarketplaceList] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"packs": listings})
}

// handleAdminDelistPack delists a published pack (sets status to 'delisted').
// POST /api/admin/marketplace/{id}/delist
func handleAdminDelistPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse listing ID from URL: /api/admin/marketplace/{id}/delist
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/marketplace/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "delist" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	listingID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "published" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "can_only_delist_published"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	_, err = db.Exec("UPDATE pack_listings SET status='delisted', reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=?",
		adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminRelistPack relists a delisted pack (sets status back to 'published').
// POST /api/admin/marketplace/{id}/relist
func handleAdminRelistPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse listing ID from URL: /api/admin/marketplace/{id}/relist
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/marketplace/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "relist" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	listingID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var currentStatus string
	err = db.QueryRow("SELECT status FROM pack_listings WHERE id = ?", listingID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "listing_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	if currentStatus != "delisted" {
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "can_only_relist_delisted"})
		return
	}

	adminIDStr := r.Header.Get("X-Admin-ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	_, err = db.Exec("UPDATE pack_listings SET status='published', reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=?",
		adminID, listingID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}


// AuthorInfo represents author statistics for admin management.
type AuthorInfo struct {
	UserID         int64   `json:"user_id"`
	DisplayName    string  `json:"display_name"`
	Email          string  `json:"email"`
	TotalPacks     int     `json:"total_packs"`
	TotalDownloads int     `json:"total_downloads"`
	TotalRevenue   float64 `json:"total_revenue"`
	YearDownloads  int     `json:"year_downloads"`
	YearRevenue    float64 `json:"year_revenue"`
	MonthDownloads int     `json:"month_downloads"`
	MonthRevenue   float64 `json:"month_revenue"`
	CreatedAt      string  `json:"created_at"`
}

// handleAdminAuthorList lists authors with sales statistics.
// GET /api/admin/authors?sort=total_downloads&order=desc&email=
func handleAdminAuthorList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	now := time.Now()
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	query := `
		SELECT u.id, u.display_name, COALESCE(u.email, ''),
		       COUNT(pl.id) as total_packs,
		       COALESCE(SUM(pl.download_count), 0) as total_downloads,
		       COALESCE(SUM(pl.download_count * pl.credits_price), 0) as total_revenue,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count ELSE 0 END), 0) as year_downloads,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count * pl.credits_price ELSE 0 END), 0) as year_revenue,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count ELSE 0 END), 0) as month_downloads,
		       COALESCE(SUM(CASE WHEN pl.created_at >= ? THEN pl.download_count * pl.credits_price ELSE 0 END), 0) as month_revenue,
		       u.created_at
		FROM users u
		INNER JOIN pack_listings pl ON pl.user_id = u.id AND pl.status IN ('published', 'delisted')
	`
	args := []interface{}{yearStart, yearStart, monthStart, monthStart}

	// Filter by email
	if email := r.URL.Query().Get("email"); email != "" {
		query += " AND u.email LIKE ?"
		args = append(args, "%"+email+"%")
	}

	query += " GROUP BY u.id"

	// Sort
	sortField := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}
	switch sortField {
	case "total_packs":
		query += " ORDER BY total_packs " + order
	case "year_downloads":
		query += " ORDER BY year_downloads " + order
	case "year_revenue":
		query += " ORDER BY year_revenue " + order
	case "month_downloads":
		query += " ORDER BY month_downloads " + order
	case "month_revenue":
		query += " ORDER BY month_revenue " + order
	default:
		query += " ORDER BY total_downloads " + order
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query authors: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	authors := []AuthorInfo{}
	for rows.Next() {
		var a AuthorInfo
		if err := rows.Scan(&a.UserID, &a.DisplayName, &a.Email,
			&a.TotalPacks, &a.TotalDownloads, &a.TotalRevenue,
			&a.YearDownloads, &a.YearRevenue, &a.MonthDownloads, &a.MonthRevenue,
			&a.CreatedAt); err != nil {
			log.Printf("Failed to scan author: %v", err)
			continue
		}
		authors = append(authors, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminAuthorList] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"authors": authors})
}

// AuthorPackDetail represents per-pack sales detail for an author.
type AuthorPackDetail struct {
	PackID        int64   `json:"pack_id"`
	PackName      string  `json:"pack_name"`
	CategoryName  string  `json:"category_name"`
	ShareMode     string  `json:"share_mode"`
	CreditsPrice  int     `json:"credits_price"`
	DownloadCount int     `json:"download_count"`
	TotalRevenue  float64 `json:"total_revenue"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
}

// handleAdminAuthorDetail returns per-pack sales detail for a specific author.
// GET /api/admin/authors/{id}
func handleAdminAuthorDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse user ID from URL: /api/admin/authors/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/authors/")
	userID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	// Get author info
	var displayName, email string
	err = db.QueryRow("SELECT display_name, COALESCE(email, '') FROM users WHERE id = ?", userID).Scan(&displayName, &email)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Get per-pack details
	rows, err := db.Query(`
		SELECT pl.id, pl.pack_name, c.name, pl.share_mode, pl.credits_price, pl.download_count,
		       pl.download_count * pl.credits_price as total_revenue, pl.status, pl.created_at
		FROM pack_listings pl
		JOIN categories c ON c.id = pl.category_id
		WHERE pl.user_id = ? AND pl.status IN ('published', 'delisted')
		ORDER BY pl.download_count DESC`, userID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	packs := []AuthorPackDetail{}
	for rows.Next() {
		var p AuthorPackDetail
		if err := rows.Scan(&p.PackID, &p.PackName, &p.CategoryName, &p.ShareMode,
			&p.CreditsPrice, &p.DownloadCount, &p.TotalRevenue, &p.Status, &p.CreatedAt); err != nil {
			log.Printf("Failed to scan author pack detail: %v", err)
			continue
		}
		packs = append(packs, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminAuthorDetail] rows iteration error: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user_id":      userID,
		"display_name": displayName,
		"email":        email,
		"packs":        packs,
	})
}

// handleAdminMarketplaceRoutes dispatches marketplace admin API requests.
func handleAdminMarketplaceRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/marketplace")
	if path == "" || path == "/" {
		handleAdminMarketplaceList(w, r)
		return
	}
	// /api/admin/marketplace/{id}/delist
	if strings.HasSuffix(path, "/delist") {
		handleAdminDelistPack(w, r)
		return
	}
	// /api/admin/marketplace/{id}/relist
	if strings.HasSuffix(path, "/relist") {
		handleAdminRelistPack(w, r)
		return
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// handleAdminAuthorRoutes dispatches author admin API requests.
func handleAdminAuthorRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/authors")
	if path == "" || path == "/" {
		handleAdminAuthorList(w, r)
		return
	}
	// /api/admin/authors/{id}
	handleAdminAuthorDetail(w, r)
}

// --- Customer Management ---

// handleAdminCustomerList lists all marketplace users (customers).
// GET /api/admin/customers?search=&sort=created_at&order=desc
func handleAdminCustomerList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query := `
		SELECT u.id, u.auth_type, u.auth_id, u.display_name, COALESCE(u.email, ''),
		       u.credits_balance, COALESCE(u.is_blocked, 0), u.created_at,
		       COUNT(DISTINCT ud.listing_id) as download_count,
		       COALESCE(SUM(CASE WHEN ct.transaction_type = 'download' THEN ABS(ct.amount) ELSE 0 END), 0) as total_spent
		FROM users u
		LEFT JOIN user_downloads ud ON ud.user_id = u.id
		LEFT JOIN credits_transactions ct ON ct.user_id = u.id
		GROUP BY u.id`

	args := []interface{}{}

	// Search by email or display_name
	if search := r.URL.Query().Get("search"); search != "" {
		query = `
		SELECT u.id, u.auth_type, u.auth_id, u.display_name, COALESCE(u.email, ''),
		       u.credits_balance, COALESCE(u.is_blocked, 0), u.created_at,
		       COUNT(DISTINCT ud.listing_id) as download_count,
		       COALESCE(SUM(CASE WHEN ct.transaction_type = 'download' THEN ABS(ct.amount) ELSE 0 END), 0) as total_spent
		FROM users u
		LEFT JOIN user_downloads ud ON ud.user_id = u.id
		LEFT JOIN credits_transactions ct ON ct.user_id = u.id
		WHERE u.email LIKE ? OR u.display_name LIKE ? OR u.auth_id LIKE ?
		GROUP BY u.id`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	// Sort
	sortField := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}
	switch sortField {
	case "credits":
		query += " ORDER BY u.credits_balance " + order
	case "downloads":
		query += " ORDER BY download_count " + order
	case "spent":
		query += " ORDER BY total_spent " + order
	case "name":
		query += " ORDER BY u.display_name " + order
	default:
		query += " ORDER BY u.created_at " + order
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query customers: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type CustomerInfo struct {
		ID             int64   `json:"id"`
		AuthType       string  `json:"auth_type"`
		AuthID         string  `json:"auth_id"`
		DisplayName    string  `json:"display_name"`
		Email          string  `json:"email"`
		CreditsBalance float64 `json:"credits_balance"`
		IsBlocked      bool    `json:"is_blocked"`
		CreatedAt      string  `json:"created_at"`
		DownloadCount  int     `json:"download_count"`
		TotalSpent     float64 `json:"total_spent"`
	}

	customers := []CustomerInfo{}
	for rows.Next() {
		var c CustomerInfo
		var blocked int
		if err := rows.Scan(&c.ID, &c.AuthType, &c.AuthID, &c.DisplayName, &c.Email,
			&c.CreditsBalance, &blocked, &c.CreatedAt, &c.DownloadCount, &c.TotalSpent); err != nil {
			log.Printf("Failed to scan customer: %v", err)
			continue
		}
		c.IsBlocked = blocked == 1
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminCustomerList] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"customers": customers})
}

// handleAdminCustomerTopup adds credits to a customer's balance.
// POST /api/admin/customers/{id}/topup  body: {"amount": 100, "reason": "manual topup"}
func handleAdminCustomerTopup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse user ID from URL: /api/admin/customers/{id}/topup
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "topup" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if req.Amount <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}

	// Check user exists
	var currentBalance float64
	err = db.QueryRow("SELECT credits_balance FROM users WHERE id = ?", userID).Scan(&currentBalance)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Update balance
	newBalance := currentBalance + req.Amount
	_, err = db.Exec("UPDATE users SET credits_balance = ? WHERE id = ?", newBalance, userID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Record transaction
	desc := "Admin topup"
	if req.Reason != "" {
		desc = "Admin topup: " + req.Reason
	}
	_, err = db.Exec("INSERT INTO credits_transactions (user_id, transaction_type, amount, description) VALUES (?, 'admin_topup', ?, ?)",
		userID, req.Amount, desc)
	if err != nil {
		log.Printf("Failed to record topup transaction: %v", err)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"new_balance": newBalance,
	})
}

// handleAdminCustomerToggleBlock blocks or unblocks a customer.
// POST /api/admin/customers/{id}/toggle-block
func handleAdminCustomerToggleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse user ID from URL: /api/admin/customers/{id}/toggle-block
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "toggle-block" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	// Get current blocked status
	var isBlocked int
	err = db.QueryRow("SELECT COALESCE(is_blocked, 0) FROM users WHERE id = ?", userID).Scan(&isBlocked)
	if err == sql.ErrNoRows {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
		return
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	newBlocked := 0
	if isBlocked == 0 {
		newBlocked = 1
	}

	_, err = db.Exec("UPDATE users SET is_blocked = ? WHERE id = ?", newBlocked, userID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	status := "unblocked"
	if newBlocked == 1 {
		status = "blocked"
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": status})
}

// handleAdminCustomerTransactions returns credits transaction history for a customer.
// GET /api/admin/customers/{id}/transactions
func handleAdminCustomerTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse user ID from URL: /api/admin/customers/{id}/transactions
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "transactions" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_path"})
		return
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	rows, err := db.Query(`
		SELECT id, transaction_type, amount, COALESCE(listing_id, 0), COALESCE(description, ''), created_at
		FROM credits_transactions WHERE user_id = ? ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type TxInfo struct {
		ID              int64   `json:"id"`
		TransactionType string  `json:"transaction_type"`
		Amount          float64 `json:"amount"`
		ListingID       int64   `json:"listing_id,omitempty"`
		Description     string  `json:"description"`
		CreatedAt       string  `json:"created_at"`
	}

	txns := []TxInfo{}
	for rows.Next() {
		var t TxInfo
		if err := rows.Scan(&t.ID, &t.TransactionType, &t.Amount, &t.ListingID, &t.Description, &t.CreatedAt); err != nil {
			continue
		}
		txns = append(txns, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminCustomerTransactions] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"transactions": txns})
}

// handleAdminCustomerRoutes dispatches customer admin API requests.
func handleAdminCustomerRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/customers")
	if path == "" || path == "/" {
		handleAdminCustomerList(w, r)
		return
	}
	if strings.HasSuffix(path, "/topup") {
		handleAdminCustomerTopup(w, r)
		return
	}
	if strings.HasSuffix(path, "/toggle-block") {
		handleAdminCustomerToggleBlock(w, r)
		return
	}
	if strings.HasSuffix(path, "/transactions") {
		handleAdminCustomerTransactions(w, r)
		return
	}
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// handleAdminSalesRoutes dispatches sales management API requests.
func handleAdminSalesRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/sales")
	switch {
	case path == "" || path == "/":
		handleAdminSalesList(w, r)
	case path == "/authors":
		handleAdminSalesAuthors(w, r)
	case path == "/export":
		handleAdminSalesExport(w, r)
	default:
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	}
}

// buildSalesWhereClause builds the WHERE clause and args for sales queries based on filters.
func buildSalesWhereClause(r *http.Request) (string, []interface{}) {
	where := "WHERE ct.transaction_type IN ('purchase', 'purchase_uses', 'renew', 'download') AND ct.listing_id IS NOT NULL"
	var args []interface{}

	if catID := r.URL.Query().Get("category_id"); catID != "" {
		where += " AND pl.category_id = ?"
		args = append(args, catID)
	}
	if authorID := r.URL.Query().Get("author_id"); authorID != "" {
		where += " AND pl.user_id = ?"
		args = append(args, authorID)
	}
	if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
		where += " AND ct.created_at >= ?"
		args = append(args, dateFrom+" 00:00:00")
	}
	if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
		where += " AND ct.created_at <= ?"
		args = append(args, dateTo+" 23:59:59")
	}
	return where, args
}

// handleAdminSalesList returns sales orders with summary statistics.
// GET /api/admin/sales?category_id=&author_id=&date_from=&date_to=
func handleAdminSalesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	where, args := buildSalesWhereClause(r)

	// Parse pagination params
	page := 1
	pageSize := 100
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 500 {
			pageSize = v
		}
	}
	offset := (page - 1) * pageSize

	// Query summary stats (always full, no pagination)
	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(SUM(ABS(ct.amount)), 0),
		       COUNT(DISTINCT ct.user_id), COUNT(DISTINCT pl.user_id)
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		%s`, where)

	var totalOrders int
	var totalCredits float64
	var totalUsers, totalAuthors int
	err := db.QueryRow(summaryQuery, args...).Scan(&totalOrders, &totalCredits, &totalUsers, &totalAuthors)
	if err != nil {
		log.Printf("[handleAdminSalesList] summary query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}

	// Query order details with pagination
	orderQuery := fmt.Sprintf(`
		SELECT ct.id, ct.user_id, COALESCE(buyer.display_name, ''), COALESCE(buyer.email, ''),
		       COALESCE(pl.pack_name, ''), COALESCE(cat.name, ''),
		       COALESCE(author.display_name, ''), ct.amount, ct.transaction_type,
		       COALESCE(ct.ip_address, ''), ct.created_at
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users buyer ON ct.user_id = buyer.id
		LEFT JOIN users author ON pl.user_id = author.id
		LEFT JOIN categories cat ON pl.category_id = cat.id
		%s
		ORDER BY ct.created_at DESC
		LIMIT ? OFFSET ?`, where)

	paginatedArgs := append(append([]interface{}{}, args...), pageSize, offset)
	rows, err := db.Query(orderQuery, paginatedArgs...)
	if err != nil {
		log.Printf("[handleAdminSalesList] order query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type SalesOrder struct {
		ID              int64   `json:"id"`
		BuyerID         int64   `json:"buyer_id"`
		BuyerName       string  `json:"buyer_name"`
		BuyerEmail      string  `json:"buyer_email"`
		PackName        string  `json:"pack_name"`
		CategoryName    string  `json:"category_name"`
		AuthorName      string  `json:"author_name"`
		Amount          float64 `json:"amount"`
		TransactionType string  `json:"transaction_type"`
		BuyerIP         string  `json:"buyer_ip"`
		CreatedAt       string  `json:"created_at"`
	}

	orders := []SalesOrder{}
	for rows.Next() {
		var o SalesOrder
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.BuyerName, &o.BuyerEmail,
			&o.PackName, &o.CategoryName, &o.AuthorName, &o.Amount, &o.TransactionType,
			&o.BuyerIP, &o.CreatedAt); err != nil {
			log.Printf("[handleAdminSalesList] scan error: %v", err)
			continue
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminSalesList] rows iteration error: %v", err)
	}

	totalPages := (totalOrders + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total_orders":  totalOrders,
		"total_credits": totalCredits,
		"total_users":   totalUsers,
		"total_authors": totalAuthors,
		"orders":        orders,
		"page":          page,
		"page_size":     pageSize,
		"total_pages":   totalPages,
	})
}


// handleAdminSalesAuthors returns a list of authors who have published packs (for filter dropdown).
// GET /api/admin/sales/authors
func handleAdminSalesAuthors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := db.Query(`
		SELECT DISTINCT u.id, u.display_name
		FROM users u
		INNER JOIN pack_listings pl ON pl.user_id = u.id
		ORDER BY u.display_name`)
	if err != nil {
		log.Printf("[handleAdminSalesAuthors] query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer rows.Close()

	type AuthorOption struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	authors := []AuthorOption{}
	for rows.Next() {
		var a AuthorOption
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			continue
		}
		authors = append(authors, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[handleAdminSalesAuthors] rows iteration error: %v", err)
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"authors": authors})
}

// handleAdminSalesExport exports sales data as an Excel file with 3 sheets:
// 1. 订单详表 (Order Details) - every transaction with buyer info, IP, amount
// 2. 用户汇总 (User Summary) - aggregated by buyer
// 3. 作者汇总 (Author Summary) - aggregated by author
// GET /api/admin/sales/export?category_id=&author_id=&date_from=&date_to=
func handleAdminSalesExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	where, args := buildSalesWhereClause(r)

	// --- Sheet 1: Order Details ---
	orderQuery := fmt.Sprintf(`
		SELECT ct.id, COALESCE(buyer.display_name, ''), COALESCE(buyer.email, ''),
		       COALESCE(pl.pack_name, ''), COALESCE(cat.name, ''),
		       COALESCE(author.display_name, ''), ABS(ct.amount), ct.transaction_type,
		       COALESCE(ct.ip_address, ''), ct.created_at
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users buyer ON ct.user_id = buyer.id
		LEFT JOIN users author ON pl.user_id = author.id
		LEFT JOIN categories cat ON pl.category_id = cat.id
		%s
		ORDER BY ct.created_at DESC`, where)

	orderRows, err := db.Query(orderQuery, args...)
	if err != nil {
		log.Printf("[handleAdminSalesExport] order query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer orderRows.Close()

	type OrderRow struct {
		ID              int64
		BuyerName       string
		BuyerEmail      string
		PackName        string
		CategoryName    string
		AuthorName      string
		Amount          float64
		TransactionType string
		BuyerIP         string
		CreatedAt       string
	}
	var orders []OrderRow
	for orderRows.Next() {
		var o OrderRow
		if err := orderRows.Scan(&o.ID, &o.BuyerName, &o.BuyerEmail, &o.PackName, &o.CategoryName,
			&o.AuthorName, &o.Amount, &o.TransactionType, &o.BuyerIP, &o.CreatedAt); err != nil {
			continue
		}
		orders = append(orders, o)
	}
	if err := orderRows.Err(); err != nil {
		log.Printf("[handleAdminSalesExport] order rows iteration error: %v", err)
	}

	// --- Sheet 2: User Summary ---
	userQuery := fmt.Sprintf(`
		SELECT buyer.id, COALESCE(buyer.display_name, ''), COALESCE(buyer.email, ''),
		       COUNT(*), COALESCE(SUM(ABS(ct.amount)), 0)
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users buyer ON ct.user_id = buyer.id
		%s
		GROUP BY buyer.id
		ORDER BY COALESCE(SUM(ABS(ct.amount)), 0) DESC`, where)

	userRows, err := db.Query(userQuery, args...)
	if err != nil {
		log.Printf("[handleAdminSalesExport] user query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer userRows.Close()

	type UserRow struct {
		ID           int64
		DisplayName  string
		Email        string
		OrderCount   int
		TotalCredits float64
	}
	var users []UserRow
	for userRows.Next() {
		var u UserRow
		if err := userRows.Scan(&u.ID, &u.DisplayName, &u.Email, &u.OrderCount, &u.TotalCredits); err != nil {
			continue
		}
		users = append(users, u)
	}
	if err := userRows.Err(); err != nil {
		log.Printf("[handleAdminSalesExport] user rows iteration error: %v", err)
	}

	// --- Sheet 3: Author Summary ---
	authorQuery := fmt.Sprintf(`
		SELECT author.id, COALESCE(author.display_name, ''), COALESCE(author.email, ''),
		       COUNT(*), COALESCE(SUM(ABS(ct.amount)), 0),
		       COUNT(DISTINCT pl.id)
		FROM credits_transactions ct
		LEFT JOIN pack_listings pl ON ct.listing_id = pl.id
		LEFT JOIN users author ON pl.user_id = author.id
		%s
		GROUP BY author.id
		ORDER BY COALESCE(SUM(ABS(ct.amount)), 0) DESC`, where)

	authorRows, err := db.Query(authorQuery, args...)
	if err != nil {
		log.Printf("[handleAdminSalesExport] author query error: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "database_error"})
		return
	}
	defer authorRows.Close()

	type AuthorRow struct {
		ID           int64
		DisplayName  string
		Email        string
		OrderCount   int
		TotalCredits float64
		PackCount    int
	}
	var authors []AuthorRow
	for authorRows.Next() {
		var a AuthorRow
		if err := authorRows.Scan(&a.ID, &a.DisplayName, &a.Email, &a.OrderCount, &a.TotalCredits, &a.PackCount); err != nil {
			continue
		}
		authors = append(authors, a)
	}
	if err := authorRows.Err(); err != nil {
		log.Printf("[handleAdminSalesExport] author rows iteration error: %v", err)
	}

	// Build Excel file
	lang := i18n.DetectLang(r)
	f := excelize.NewFile()
	defer f.Close()

	typeLabels := map[string]string{
		"purchase": i18n.T(lang, "excel_tx_purchase"), "purchase_uses": i18n.T(lang, "excel_tx_purchase_uses"), "renew": i18n.T(lang, "excel_tx_renew"), "download": i18n.T(lang, "excel_tx_free_claim"),
	}

	// Sheet 1: 订单详表
	sheet1 := i18n.T(lang, "excel_order_sheet")
	f.SetSheetName("Sheet1", sheet1)
	orderHeaders := []string{i18n.T(lang, "excel_order_id"), i18n.T(lang, "excel_buyer_name"), i18n.T(lang, "excel_buyer_email"), i18n.T(lang, "excel_pack_name"), i18n.T(lang, "excel_category"), i18n.T(lang, "excel_author"), i18n.T(lang, "excel_amount_credits"), i18n.T(lang, "excel_tx_type"), i18n.T(lang, "excel_buyer_ip"), i18n.T(lang, "excel_time")}
	for i, h := range orderHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet1, cell, h)
	}
	for rowIdx, o := range orders {
		row := rowIdx + 2
		tl := typeLabels[o.TransactionType]
		if tl == "" {
			tl = o.TransactionType
		}
		vals := []interface{}{o.ID, o.BuyerName, o.BuyerEmail, o.PackName, o.CategoryName, o.AuthorName, o.Amount, tl, o.BuyerIP, o.CreatedAt}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet1, cell, val)
		}
	}

	// Sheet 2: 用户汇总
	sheet2 := i18n.T(lang, "excel_user_sheet")
	f.NewSheet(sheet2)
	userHeaders := []string{i18n.T(lang, "excel_user_id"), i18n.T(lang, "excel_name"), i18n.T(lang, "excel_email"), i18n.T(lang, "excel_order_count"), i18n.T(lang, "excel_total_spent")}
	for i, h := range userHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet2, cell, h)
	}
	for rowIdx, u := range users {
		row := rowIdx + 2
		vals := []interface{}{u.ID, u.DisplayName, u.Email, u.OrderCount, u.TotalCredits}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet2, cell, val)
		}
	}

	// Sheet 3: 作者汇总
	sheet3 := i18n.T(lang, "excel_author_sheet")
	f.NewSheet(sheet3)
	authorHeaders := []string{i18n.T(lang, "excel_author_id"), i18n.T(lang, "excel_name"), i18n.T(lang, "excel_email"), i18n.T(lang, "excel_order_count"), i18n.T(lang, "excel_total_sales"), i18n.T(lang, "excel_pack_count")}
	for i, h := range authorHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet3, cell, h)
	}
	for rowIdx, a := range authors {
		row := rowIdx + 2
		vals := []interface{}{a.ID, a.DisplayName, a.Email, a.OrderCount, a.TotalCredits, a.PackCount}
		for i, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet3, cell, val)
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("[handleAdminSalesExport] failed to write Excel: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate Excel file"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="sales_export.xlsx"`)
	w.Write(buf.Bytes())
}

// handleTranslationsAPI returns all translations for the detected language as JSON.
func handleTranslationsAPI(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLang(r)
	translations := i18n.AllTranslations(lang)
	// Cache per-user (language depends on cookie), not shared
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Vary", "Cookie")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lang":         string(lang),
		"translations": translations,
	})
}

// handleSetLang sets the language cookie and redirects back.
func handleSetLang(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = r.FormValue("lang")
	}
	if lang != "zh-CN" && lang != "en-US" {
		lang = "zh-CN"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Path:     "/",
		MaxAge:   365 * 24 * 3600,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	// Use redirect query param first, then Referer header (validated to be same-origin)
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = r.Header.Get("Referer")
	}
	// Security: only allow relative paths to prevent open redirect
	if redirect == "" || !strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") {
		// Parse Referer to extract path if it's an absolute URL on the same host
		if redirect != "" && strings.Contains(redirect, "://") {
			if u, err := url.Parse(redirect); err == nil && u.Host == r.Host {
				redirect = u.RequestURI()
			} else {
				redirect = "/"
			}
		} else {
			redirect = "/"
		}
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

// securityHeaders adds standard security headers to all responses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
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

	// Load default language setting
	if dl := getSetting("default_language"); dl == "en-US" {
		i18n.DefaultLang = i18n.EnUS
	} else {
		i18n.DefaultLang = i18n.ZhCN
	}

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
			// Clean up expired user sessions
			userSessionsMu.Lock()
			for id, entry := range userSessions {
				if now.After(entry.Expiry) {
					delete(userSessions, id)
				}
			}
			userSessionsMu.Unlock()
			// Clean up expired math captcha expressions
			mathCaptchaExpressionsMu.Lock()
			for id := range mathCaptchaExpressions {
				captchasMu.RLock()
				_, exists := captchas[id]
				captchasMu.RUnlock()
				if !exists {
					delete(mathCaptchaExpressions, id)
				}
			}
			mathCaptchaExpressionsMu.Unlock()
			// Clean up expired login tickets
			loginTicketsMu.Lock()
			for id, entry := range loginTickets {
				if now.After(entry.Expiry) {
					delete(loginTickets, id)
				}
			}
			loginTicketsMu.Unlock()
		}
	}()

	// i18n routes
	http.HandleFunc("/api/translations", handleTranslationsAPI)
	http.HandleFunc("/set-lang", handleSetLang)

	// Auth routes
	http.HandleFunc("/api/auth/sn-login", handleSNLogin)
	http.HandleFunc("/api/auth/oauth", handleOAuthCallback) // kept for backward compatibility

	// Category routes (listing is public, admin requires auth)
	http.HandleFunc("/api/categories", handleListCategories)
	http.HandleFunc("/api/admin/categories", permissionAuth("categories")(handleAdminCategories))
	http.HandleFunc("/api/admin/categories/", permissionAuth("categories")(handleAdminCategories))

	// Pack routes (upload and download require auth, listing is public)
	http.HandleFunc("/api/packs/upload", authMiddleware(handleUploadPack))
	http.HandleFunc("/api/packs/replace", authMiddleware(handleReplacePack))
	http.HandleFunc("/api/packs/report-usage", authMiddleware(handleReportPackUsage))
	http.HandleFunc("/api/packs/listing-id", authMiddleware(handleGetListingID))
	http.HandleFunc("/api/packs/purchased", authMiddleware(handleGetPurchasedPacks))
	http.HandleFunc("/api/packs/my-licenses", authMiddleware(handleGetMyLicenses))
	http.HandleFunc("/api/packs", handleListPacks)
	http.HandleFunc("/api/packs/", func(w http.ResponseWriter, r *http.Request) {
		// Dispatch based on URL suffix
		switch {
		case strings.HasSuffix(r.URL.Path, "/detail"):
			// Pack detail API is public, no auth required
			handleGetPackDetail(w, r)
		case strings.HasSuffix(r.URL.Path, "/purchase-uses"):
			authMiddleware(handlePurchaseAdditionalUses)(w, r)
		case strings.HasSuffix(r.URL.Path, "/renew"):
			authMiddleware(handleRenewSubscription)(w, r)
		default:
			authMiddleware(handleDownloadPack)(w, r)
		}
	})

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

	// Admin management API routes (super admin id=1 only)
	http.HandleFunc("/api/admin/admins", superAdminOnlyAuth(handleAdminManagement))
	http.HandleFunc("/api/admin/profile", adminAuth(handleUpdateProfile))

	// Marketplace management API routes (permission-based)
	http.HandleFunc("/api/admin/marketplace", permissionAuth("marketplace")(handleAdminMarketplaceRoutes))
	http.HandleFunc("/api/admin/marketplace/", permissionAuth("marketplace")(handleAdminMarketplaceRoutes))

	// Author management API routes (permission-based)
	http.HandleFunc("/api/admin/authors", permissionAuth("authors")(handleAdminAuthorRoutes))
	http.HandleFunc("/api/admin/authors/", permissionAuth("authors")(handleAdminAuthorRoutes))

	// Customer management API routes (permission-based)
	http.HandleFunc("/api/admin/customers", permissionAuth("customers")(handleAdminCustomerRoutes))
	http.HandleFunc("/api/admin/customers/", permissionAuth("customers")(handleAdminCustomerRoutes))

	// User notification query API (public, optional JWT auth)
	http.HandleFunc("/api/notifications", handleListNotifications)

	// Notification management API routes (permission-based)
	http.HandleFunc("/api/admin/notifications", permissionAuth("notifications")(handleAdminNotificationRoutes))
	http.HandleFunc("/api/admin/notifications/", permissionAuth("notifications")(handleAdminNotificationRoutes))

	// Review API routes (permission-based)
	http.HandleFunc("/api/admin/review/", permissionAuth("review")(handleReviewRoutes))

	// Sales management API routes (permission-based)
	http.HandleFunc("/api/admin/sales", permissionAuth("sales")(handleAdminSalesRoutes))
	http.HandleFunc("/api/admin/sales/", permissionAuth("sales")(handleAdminSalesRoutes))

	// Admin routes (protected by session auth)
	http.HandleFunc("/admin/settings/initial-credits", permissionAuth("settings")(handleSetInitialCredits))
	http.HandleFunc("/admin/settings/credit-cash-rate", permissionAuth("settings")(handleSetCreditCashRate))
	http.HandleFunc("/admin/api/settings/revenue-split", permissionAuth("settings")(handleAdminSaveRevenueSplit))
	http.HandleFunc("/admin/api/settings/withdrawal-fees", permissionAuth("settings")(handleAdminSaveWithdrawalFees))
	http.HandleFunc("/admin/api/settings/default-language", permissionAuth("settings")(handleSetDefaultLanguage))
	http.HandleFunc("/admin/api/withdrawals/export", permissionAuth("settings")(handleAdminExportWithdrawals))
	http.HandleFunc("/admin/api/withdrawals/approve", permissionAuth("settings")(handleAdminApproveWithdrawals))
	http.HandleFunc("/admin/api/withdrawals", permissionAuth("settings")(handleAdminGetWithdrawals))
	http.HandleFunc("/admin/", adminAuth(handleAdminDashboard))

	// User portal routes
	http.HandleFunc("/user/login", handleUserLogin)
	http.HandleFunc("/user/register", handleUserRegister)
	http.HandleFunc("/user/logout", handleUserLogout)
	http.HandleFunc("/user/ticket-login", handleTicketLogin)
	http.HandleFunc("/user/change-password", userAuth(handleUserChangePassword))
	http.HandleFunc("/user/set-password", userAuth(handleUserSetPassword))
	http.HandleFunc("/user/captcha", handleUserCaptchaImage)
	http.HandleFunc("/user/captcha/refresh", handleUserCaptchaRefresh)
	http.HandleFunc("/user/billing", userAuth(handleUserBilling))
	http.HandleFunc("/user/pack/renew-uses", userAuth(handleUserRenewPerUse))
	http.HandleFunc("/user/pack/renew-subscription", userAuth(handleUserRenewSubscription))
	http.HandleFunc("/user/pack/delete", userAuth(handleSoftDeletePack))
	http.HandleFunc("/user/payment-info", userAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetPaymentInfo(w, r)
		case http.MethodPost:
			handleSavePaymentInfo(w, r)
		default:
			jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))
	http.HandleFunc("/user/payment-info/fee-rate", userAuth(handleGetPaymentFeeRate))
	http.HandleFunc("/user/payment-info/fee-rates", userAuth(handleGetAllPaymentFeeRates))
	http.HandleFunc("/user/author/withdraw", userAuth(handleAuthorWithdraw))
	http.HandleFunc("/user/author/withdrawals", userAuth(handleAuthorWithdrawRecords))
	http.HandleFunc("/user/author/edit-pack", userAuth(handleAuthorEditPack))
	http.HandleFunc("/user/author/delete-pack", userAuth(handleAuthorDeletePack))
	http.HandleFunc("/user/author/delist-pack", userAuth(handleAuthorDelistPack))
	http.HandleFunc("/user/author/pack-purchases", userAuth(handleAuthorPackPurchases))
	http.HandleFunc("/user/", userAuth(handleUserDashboard))

	// Pack detail page route (catches /pack/*)
	http.HandleFunc("/pack/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/claim"):
			userAuth(handleClaimFreePack)(w, r)
		case strings.HasSuffix(r.URL.Path, "/purchase"):
			userAuth(handlePurchaseFromDetail)(w, r)
		default:
			handlePackDetailPage(w, r)
		}
	})

	// Root redirect to user portal
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/user/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Marketplace server starting on %s", addr)

	// Wrap with security headers middleware
	handler := securityHeaders(http.DefaultServeMux)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
