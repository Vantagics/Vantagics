package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	mrand "math/rand"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
	"license_server/templates"
)

// Default fallback values (used only when environment variables are not set)
const (
	defaultDBPassword       = "sunion123!"
	defaultAdminPassword    = "sunion123"
)

// hashAdminPassword hashes a password using bcrypt (cost 12).
// Format: bcrypt:<bcrypt_hash>
func hashAdminPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatalf("[FATAL] bcrypt.GenerateFromPassword failed: %v", err)
	}
	return "bcrypt:" + string(hash)
}

// checkAdminPassword verifies a password against a stored hash.
// Supports bcrypt (new format: bcrypt:<hash>), salted SHA-256 (legacy: hex_salt:hex_hash), and plaintext (legacy).
func checkAdminPassword(password, stored string) bool {
	// New bcrypt format
	if strings.HasPrefix(stored, "bcrypt:") {
		hash := strings.TrimPrefix(stored, "bcrypt:")
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	// Legacy salted SHA-256 format
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) == 2 && len(parts[0]) == 32 && len(parts[1]) == 64 {
		salt, err := hex.DecodeString(parts[0])
		if err != nil {
			return false
		}
		h := sha256.New()
		h.Write(salt)
		h.Write([]byte(password))
		return hex.EncodeToString(h.Sum(nil)) == parts[1]
	}
	// Legacy plaintext comparison
	return password == stored
}

// getDBPassword returns the database encryption password.
// Priority: environment variable LICENSE_DB_PASSWORD > default fallback.
func getDBPassword() string {
	if pw := os.Getenv("LICENSE_DB_PASSWORD"); pw != "" {
		return pw
	}
	log.Println("[WARN] LICENSE_DB_PASSWORD not set, using default. Set this env var in production!")
	return defaultDBPassword
}

// getDefaultAdminPassword returns the initial admin password.
// Priority: environment variable LICENSE_ADMIN_PASSWORD > default fallback.
func getDefaultAdminPassword() string {
	if pw := os.Getenv("LICENSE_ADMIN_PASSWORD"); pw != "" {
		return pw
	}
	log.Println("[WARN] LICENSE_ADMIN_PASSWORD not set, using default. Change admin password after first login!")
	return defaultAdminPassword
}

// DBPassword and DefaultAdminPassword are kept as variables for backward compatibility.
// They are initialized in init() from environment variables or defaults.
var (
	DBPassword           string
	DefaultAdminPassword string
)

func init() {
	DBPassword = getDBPassword()
	DefaultAdminPassword = getDefaultAdminPassword()
}

// Error codes for API responses (for client-side localization)
const (
	CodeSuccess           = "SUCCESS"
	CodeInvalidRequest    = "INVALID_REQUEST"
	CodeInvalidSN         = "INVALID_SN"
	CodeSNDisabled        = "SN_DISABLED"
	CodeSNExpired         = "SN_EXPIRED"
	CodeEncryptFailed     = "ENCRYPT_FAILED"
	CodeInvalidEmail      = "INVALID_EMAIL"
	CodeEmailBlacklisted  = "EMAIL_BLACKLISTED"
	CodeEmailNotWhitelisted = "EMAIL_NOT_WHITELISTED"
	CodeEmailAlreadyUsed  = "EMAIL_ALREADY_USED"
	CodeNoAvailableSN     = "NO_AVAILABLE_SN"
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	CodeEmailLimitExceeded = "EMAIL_LIMIT_EXCEEDED"
	CodeInternalError     = "INTERNAL_ERROR"
	CodeInvalidValue      = "INVALID_VALUE"
	CodeEmailMismatch     = "EMAIL_MISMATCH"
	CodeInvalidToken      = "INVALID_TOKEN"
	CodeTokenExpired      = "TOKEN_EXPIRED"
)


// LLMGroup holds LLM API group information
type LLMGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SearchGroup holds Search API group information
type SearchGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type LicenseGroup struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	TrustLevel    string `json:"trust_level"`     // "high" (正式) or "low" (试用)
	LLMGroupID    string `json:"llm_group_id"`    // LLM group for official groups
	SearchGroupID string `json:"search_group_id"` // Search group for official groups
}

// ProductType holds product type information for license categorization
type ProductType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// EmailTemplate holds email template information
type EmailTemplate struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	IsPreset  bool   `json:"is_preset"`
	CreatedAt string `json:"created_at"`
}

// LLMConfig holds LLM API configuration
type LLMConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	IsActive  bool   `json:"is_active"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	GroupID   string `json:"group_id"`
}

// SearchConfig holds search engine configuration
type SearchConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	APIKey    string `json:"api_key"`
	IsActive  bool   `json:"is_active"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	GroupID   string `json:"group_id"`
}

// License holds license information
type License struct {
	SN             string    `json:"sn"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	ValidDays      int       `json:"valid_days"`      // Validity period in days (used before activation)
	Description    string    `json:"description"`
	IsActive       bool      `json:"is_active"`
	UsageCount     int       `json:"usage_count"`
	LastUsedAt     time.Time `json:"last_used_at"`
	DailyAnalysis  int       `json:"daily_analysis"`  // Daily analysis limit, 0 = unlimited
	TotalCredits   float64   `json:"total_credits"`   // Credits total, 0 = unlimited in credits mode
	CreditsMode    bool      `json:"credits_mode"`    // true = credits mode, false = daily limit mode
	UsedCredits    float64   `json:"used_credits"`    // Server-tracked used credits
	LLMGroupID     string    `json:"llm_group_id"`    // Bound LLM group
	LicenseGroupID string    `json:"license_group_id"` // License group for organization
	SearchGroupID  string    `json:"search_group_id"` // Bound Search group
	ProductID      int       `json:"product_id"`      // Product type ID, 0 = unclassified
}

// EmailRecord stores email request information
type EmailRecord struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	SN        string    `json:"sn"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	ProductID int       `json:"product_id"`
	APIKeyID  string    `json:"api_key_id,omitempty"`
}

// APIKey stores API key information for third-party integrations
type APIKey struct {
	ID           string     `json:"id"`
	APIKey       string     `json:"api_key"`
	ProductID    int        `json:"product_id"`
	Organization string     `json:"organization"`
	ContactName  string     `json:"contact_name"`
	Description  string     `json:"description"`
	IsActive     bool       `json:"is_active"`
	UsageCount   int        `json:"usage_count"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// ActivationResponse is sent to client
type ActivationResponse struct {
	Success       bool   `json:"success"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	EncryptedData string `json:"encrypted_data,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

// ActivationData contains the decrypted configuration
type ActivationData struct {
	LLMType         string                 `json:"llm_type"`
	LLMBaseURL      string                 `json:"llm_base_url"`
	LLMAPIKey       string                 `json:"llm_api_key"`
	LLMModel        string                 `json:"llm_model"`
	LLMStartDate    string                 `json:"llm_start_date"`
	LLMEndDate      string                 `json:"llm_end_date"`
	SearchType      string                 `json:"search_type"`
	SearchAPIKey    string                 `json:"search_api_key"`
	SearchStartDate string                 `json:"search_start_date"`
	SearchEndDate   string                 `json:"search_end_date"`
	ExpiresAt       string                 `json:"expires_at"`
	ActivatedAt     string                 `json:"activated_at"`
	DailyAnalysis   int                    `json:"daily_analysis"`   // Daily analysis limit
	TotalCredits    float64                `json:"total_credits"`    // Credits total, 0 = unlimited in credits mode
	CreditsMode     bool                   `json:"credits_mode"`     // true = credits mode, false = daily limit mode
	ProductID       int                    `json:"product_id"`       // Product ID
	ProductName     string                 `json:"product_name"`     // Product name
	TrustLevel      string                 `json:"trust_level"`      // "high" or "low"
	RefreshInterval int                    `json:"refresh_interval"` // SN refresh interval in days (1=daily, 30=monthly)
	ExtraInfo       map[string]interface{} `json:"extra_info,omitempty"` // Product-specific extra info
	UsedCredits     float64                `json:"used_credits"`         // Server-tracked used credits
}

// RequestSNResponse is sent to client for SN request
type RequestSNResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	SN      string `json:"sn,omitempty"`
	Code    string `json:"code,omitempty"`
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"from_email"`
	FromName   string `json:"from_name"`
	UseTLS     bool   `json:"use_tls"`
	UseSTARTTLS bool  `json:"use_starttls"`
}

var (
	db         *sql.DB
	dbLock     sync.RWMutex
	dbPath     string
	managePort = 8899
	authPort   = 6699
	useSSL     = false
	sslCert    = ""
	sslKey     = ""

	// Login attempt tracking
	loginAttempts     = make(map[string]*loginAttemptInfo)
	loginAttemptsLock sync.Mutex
)

type loginAttemptInfo struct {
	FailCount      int       // failures in current window (resets after cooldown)
	TotalFailCount int       // total failures ever (never resets, for permanent lock)
	LastFailTime   time.Time
	Locked         bool      // permanently locked after 15 total failures
}

func main() {
	execPath, _ := os.Executable()
	dbPath = filepath.Join(filepath.Dir(execPath), "license_server.db")
	
	initDB()
	loadPorts()
	loadSSLConfig()
	
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go startManageServer()
	go startAuthServer()
	
	<-quit
	log.Println("Shutting down servers...")
	if db != nil {
		db.Close()
	}
	log.Println("Server stopped")
}

func initDB() {
	var err error
	// Use _pragma parameters to ensure every connection from the pool gets the same settings.
	// busy_timeout(5000): wait up to 5s on lock contention instead of failing immediately
	// journal_mode(WAL): allow concurrent readers while one writer is active
	// synchronous(NORMAL): safe with WAL, reduces fsync overhead
	// temp_store(MEMORY): keep temp tables in memory
	// cache_size(-65536): 64MB page cache
	// Note: the database is NOT encrypted (confirmed on production server).
	// _pragma_key/_pragma_cipher_page_size were previously included but had no effect
	// with the modernc.org/sqlite driver (only SQLCipher supports encryption).
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-65536)", dbPath)
	db, err = sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Small pool avoids excessive lock contention on the single database file
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0) // reuse connections indefinitely
	
	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);
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
		CREATE TABLE IF NOT EXISTS llm_groups (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT
		);
		CREATE TABLE IF NOT EXISTS search_groups (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT
		);
		CREATE TABLE IF NOT EXISTS license_groups (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			trust_level TEXT DEFAULT 'low'
		);
		CREATE TABLE IF NOT EXISTS product_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT
		);
		CREATE TABLE IF NOT EXISTS llm_configs (
			id TEXT PRIMARY KEY,
			name TEXT,
			type TEXT,
			base_url TEXT,
			api_key TEXT,
			model TEXT,
			is_active INTEGER DEFAULT 0,
			start_date TEXT,
			end_date TEXT,
			group_id TEXT DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS search_configs (
			id TEXT PRIMARY KEY,
			name TEXT,
			type TEXT,
			api_key TEXT,
			is_active INTEGER DEFAULT 0,
			start_date TEXT,
			end_date TEXT,
			group_id TEXT DEFAULT ''
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
		CREATE TABLE IF NOT EXISTS request_limits (
			ip TEXT,
			date TEXT,
			count INTEGER,
			PRIMARY KEY (ip, date)
		);
		CREATE TABLE IF NOT EXISTS email_whitelist (
			pattern TEXT PRIMARY KEY,
			created_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS email_blacklist (
			pattern TEXT PRIMARY KEY,
			created_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS email_conditions (
			pattern TEXT PRIMARY KEY,
			created_at DATETIME,
			llm_group_id TEXT DEFAULT '',
			search_group_id TEXT DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			api_key TEXT UNIQUE NOT NULL,
			product_id INTEGER DEFAULT 0,
			organization TEXT DEFAULT '',
			contact_name TEXT DEFAULT '',
			description TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1,
			usage_count INTEGER DEFAULT 0,
			created_at DATETIME,
			expires_at DATETIME
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
	
	// Migration: Add new columns if they don't exist
	db.Exec("ALTER TABLE licenses ADD COLUMN llm_group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE licenses ADD COLUMN search_group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE licenses ADD COLUMN license_group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE licenses ADD COLUMN product_id INTEGER DEFAULT 0")
	db.Exec("ALTER TABLE licenses ADD COLUMN valid_days INTEGER DEFAULT 365")
	db.Exec("ALTER TABLE licenses ADD COLUMN total_credits FLOAT DEFAULT 0")
	db.Exec("ALTER TABLE licenses ADD COLUMN credits_mode INTEGER DEFAULT 0")
	db.Exec("ALTER TABLE llm_configs ADD COLUMN group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE search_configs ADD COLUMN group_id TEXT DEFAULT ''")
	// Migration: Add trust_level to license_groups
	db.Exec("ALTER TABLE license_groups ADD COLUMN trust_level TEXT DEFAULT 'low'")
	// Migration: Add llm_group_id and search_group_id to license_groups for official groups
	db.Exec("ALTER TABLE license_groups ADD COLUMN llm_group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE license_groups ADD COLUMN search_group_id TEXT DEFAULT ''")
	// Migration: Add product_id to email_records if not exists
	db.Exec("ALTER TABLE email_records ADD COLUMN product_id INTEGER DEFAULT 0")
	// Migration: Add api_key_id to email_records for tracking API-created bindings
	db.Exec("ALTER TABLE email_records ADD COLUMN api_key_id TEXT DEFAULT ''")
	// Migration: Add sn_type to email_records to distinguish free/oss/commercial bindings.
	// This allows one email to have up to 3 SNs per product (one of each type).
	// SQLite cannot drop the old UNIQUE(email, product_id) constraint, so we rebuild the table.
	db.Exec("ALTER TABLE email_records ADD COLUMN sn_type TEXT DEFAULT 'commercial'")
	// Backfill sn_type for existing records based on license_group_id
	db.Exec(`UPDATE email_records SET sn_type = 'free' WHERE sn IN (SELECT sn FROM licenses WHERE license_group_id LIKE 'free_%')`)
	db.Exec(`UPDATE email_records SET sn_type = 'oss' WHERE sn IN (SELECT sn FROM licenses WHERE license_group_id LIKE 'oss_%')`)
	// Rebuild table to replace UNIQUE(email, product_id) with UNIQUE(email, product_id, sn_type)
	var hasSnType int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('email_records') WHERE name='sn_type'").Scan(&hasSnType)
	if hasSnType > 0 {
		// Check if we need to rebuild the table (old UNIQUE(email, product_id) constraint still present)
		var idxCount int
		db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_email_product_sntype'").Scan(&idxCount)
		if idxCount == 0 {
			tx, txErr := db.Begin()
			if txErr == nil {
				_, err := tx.Exec(`CREATE TABLE email_records_new (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					email TEXT,
					sn TEXT,
					ip TEXT,
					created_at DATETIME,
					product_id INTEGER DEFAULT 0,
					api_key_id TEXT DEFAULT '',
					sn_type TEXT DEFAULT 'commercial',
					UNIQUE(email, product_id, sn_type)
				)`)
				if err != nil {
					tx.Rollback()
					log.Printf("[MIGRATION] Failed to create email_records_new: %v", err)
				} else {
					tx.Exec(`INSERT OR IGNORE INTO email_records_new (id, email, sn, ip, created_at, product_id, api_key_id, sn_type)
						SELECT id, email, sn, ip, created_at, product_id, COALESCE(api_key_id, ''), COALESCE(sn_type, 'commercial') FROM email_records`)
					tx.Exec(`DROP TABLE email_records`)
					tx.Exec(`ALTER TABLE email_records_new RENAME TO email_records`)
					// Mark migration as done (the UNIQUE constraint creates an implicit index, but we add a named one for detection)
					tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_email_product_sntype ON email_records(email, product_id, sn_type)`)
					if err := tx.Commit(); err != nil {
						log.Printf("[MIGRATION] Failed to rebuild email_records table: %v", err)
					} else {
						log.Printf("[MIGRATION] Rebuilt email_records with UNIQUE(email, product_id, sn_type)")
					}
				}
			}
		}
	}
	// Migration: Create email_conditions table if not exists (for existing databases)
	db.Exec(`CREATE TABLE IF NOT EXISTS email_conditions (
		pattern TEXT PRIMARY KEY,
		created_at DATETIME,
		llm_group_id TEXT DEFAULT '',
		search_group_id TEXT DEFAULT ''
	)`)
	// Migration: Create product_extra_info table for product-specific key-value pairs
	db.Exec(`CREATE TABLE IF NOT EXISTS product_extra_info (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL,
		key TEXT NOT NULL,
		value TEXT,
		value_type TEXT DEFAULT 'string',
		UNIQUE(product_id, key)
	)`)
	// Migration: Add used_credits to licenses for tracking credits usage
	db.Exec("ALTER TABLE licenses ADD COLUMN used_credits FLOAT DEFAULT 0")
	// Migration: Create credits_usage_log table for tracking credits usage reports
	db.Exec(`CREATE TABLE IF NOT EXISTS credits_usage_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sn TEXT NOT NULL,
		used_credits FLOAT NOT NULL,
		reported_at DATETIME NOT NULL,
		client_ip TEXT
	)`)
	// Migration: Move whitelist entries with groups to conditions table
	db.Exec(`INSERT OR IGNORE INTO email_conditions (pattern, created_at, llm_group_id, search_group_id)
		SELECT pattern, created_at, llm_group_id, search_group_id FROM email_whitelist 
		WHERE llm_group_id != '' OR search_group_id != ''`)
	
	// Set default admin password if not exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM settings WHERE key='admin_password'").Scan(&count)
	if count == 0 {
		db.Exec("INSERT INTO settings (key, value) VALUES ('admin_password', ?)", DefaultAdminPassword)
		db.Exec("INSERT INTO settings (key, value) VALUES ('admin_username', 'admin')")
		db.Exec("INSERT INTO settings (key, value) VALUES ('manage_port', '8899')")
		db.Exec("INSERT INTO settings (key, value) VALUES ('auth_port', '6699')")
	}

	// Ensure default product type (ID 0) exists for Vantagics
	var defaultProductCount int
	db.QueryRow("SELECT COUNT(*) FROM product_types WHERE id=0").Scan(&defaultProductCount)
	if defaultProductCount == 0 {
		db.Exec("INSERT INTO product_types (id, name, description) VALUES (0, 'Vantagics', 'Intelligent Data Analytics Platform')")
	}
	
	// Set default username if not exists
	var usernameCount int
	db.QueryRow("SELECT COUNT(*) FROM settings WHERE key='admin_username'").Scan(&usernameCount)
	if usernameCount == 0 {
		db.Exec("INSERT INTO settings (key, value) VALUES ('admin_username', 'admin')")
	}
	
	// Set default rate limit settings if not exists
	var limitCount int
	db.QueryRow("SELECT COUNT(*) FROM settings WHERE key='daily_request_limit'").Scan(&limitCount)
	if limitCount == 0 {
		db.Exec("INSERT INTO settings (key, value) VALUES ('daily_request_limit', '5')")
		db.Exec("INSERT INTO settings (key, value) VALUES ('daily_email_limit', '5')")
	}

	// Migration: Create email_templates table for email notification templates
	db.Exec(`CREATE TABLE IF NOT EXISTS email_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		subject TEXT NOT NULL,
		body TEXT NOT NULL,
		is_preset BOOLEAN DEFAULT 0,
		created_at DATETIME
	)`)

	// Migration: Create email_send_tasks table for batch send tasks
	db.Exec(`CREATE TABLE IF NOT EXISTS email_send_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		subject TEXT NOT NULL,
		body TEXT NOT NULL,
		total_count INTEGER,
		sent_count INTEGER DEFAULT 0,
		failed_count INTEGER DEFAULT 0,
		status TEXT DEFAULT 'running',
		created_at DATETIME,
		completed_at DATETIME
	)`)

	// Migration: Create email_send_items table for individual send items
	db.Exec(`CREATE TABLE IF NOT EXISTS email_send_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL,
		email TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		error TEXT,
		sent_at DATETIME
	)`)

	// Migration: Add index on email_send_items.task_id for efficient progress queries
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_send_items_task_id ON email_send_items(task_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_send_items_task_status ON email_send_items(task_id, status)`)

	// Migration: Add product_id to email_send_tasks
	db.Exec("ALTER TABLE email_send_tasks ADD COLUMN product_id INTEGER DEFAULT -1")

	// Insert preset email templates (5 preset templates with HTML body)
	db.Exec(`INSERT OR IGNORE INTO email_templates (name, subject, body, is_preset, created_at)
		SELECT '产品重大更新通知', '{{.ProductName}} 重大更新通知', '<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
<h2 style="color:#1a56db">{{.ProductName}} 重大更新通知</h2>
<p>尊敬的用户（{{.Email}}），您好！</p>
<p>我们很高兴地通知您，<strong>{{.ProductName}}</strong> 已发布重大更新。本次更新包含多项功能改进和性能优化，将为您带来更好的使用体验。</p>
<p><strong>您的序列号：</strong>{{.SN}}</p>
<p>请及时更新至最新版本，以享受全部新功能。如有任何问题，请随时联系我们的技术支持团队。</p>
<p>官网地址：<a href="https://vantagics.com" style="color:#1a56db">https://vantagics.com</a></p>
<p style="color:#666;font-size:12px;margin-top:30px">此邮件由系统自动发送，请勿直接回复。</p>
</div>', 1, datetime('now')
		WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE name='产品重大更新通知' AND is_preset=1)`)

	db.Exec(`INSERT OR IGNORE INTO email_templates (name, subject, body, is_preset, created_at)
		SELECT '服务器临时维护通知', '{{.ProductName}} 服务器临时维护通知', '<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
<h2 style="color:#d97706">{{.ProductName}} 服务器临时维护通知</h2>
<p>尊敬的用户（{{.Email}}），您好！</p>
<p>为了提升服务质量，<strong>{{.ProductName}}</strong> 将进行服务器临时维护。维护期间，部分服务可能暂时不可用。</p>
<p><strong>您的序列号：</strong>{{.SN}}</p>
<p>我们将尽快完成维护工作，届时服务将自动恢复。感谢您的理解与支持。</p>
<p>官网地址：<a href="https://vantagics.com" style="color:#d97706">https://vantagics.com</a></p>
<p style="color:#666;font-size:12px;margin-top:30px">此邮件由系统自动发送，请勿直接回复。</p>
</div>', 1, datetime('now')
		WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE name='服务器临时维护通知' AND is_preset=1)`)

	db.Exec(`INSERT OR IGNORE INTO email_templates (name, subject, body, is_preset, created_at)
		SELECT '新版本发布通知', '{{.ProductName}} 新版本发布通知', '<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
<h2 style="color:#059669">{{.ProductName}} 新版本发布通知</h2>
<p>尊敬的用户（{{.Email}}），您好！</p>
<p><strong>{{.ProductName}}</strong> 新版本已正式发布！新版本带来了诸多改进和新功能，欢迎您下载体验。</p>
<p><strong>您的序列号：</strong>{{.SN}}</p>
<p>请前往官方网站下载最新版本：<a href="https://vantagics.com" style="color:#059669">https://vantagics.com</a>。如需帮助，请联系我们的技术支持团队。</p>
<p style="color:#666;font-size:12px;margin-top:30px">此邮件由系统自动发送，请勿直接回复。</p>
</div>', 1, datetime('now')
		WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE name='新版本发布通知' AND is_preset=1)`)

	db.Exec(`INSERT OR IGNORE INTO email_templates (name, subject, body, is_preset, created_at)
		SELECT '服务到期提醒', '{{.ProductName}} 服务到期提醒', '<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
<h2 style="color:#dc2626">{{.ProductName}} 服务到期提醒</h2>
<p>尊敬的用户（{{.Email}}），您好！</p>
<p>您的 <strong>{{.ProductName}}</strong> 授权即将到期，为避免影响您的正常使用，请及时续费。</p>
<p><strong>您的序列号：</strong>{{.SN}}</p>
<p>如您已完成续费，请忽略此邮件。如需帮助，请访问 <a href="https://vantagics.com" style="color:#dc2626">https://vantagics.com</a> 或联系我们的技术支持团队。</p>
<p style="color:#666;font-size:12px;margin-top:30px">此邮件由系统自动发送，请勿直接回复。</p>
</div>', 1, datetime('now')
		WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE name='服务到期提醒' AND is_preset=1)`)

	db.Exec(`INSERT OR IGNORE INTO email_templates (name, subject, body, is_preset, created_at)
		SELECT '安全公告通知', '{{.ProductName}} 安全公告通知', '<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
<h2 style="color:#7c3aed">{{.ProductName}} 安全公告通知</h2>
<p>尊敬的用户（{{.Email}}），您好！</p>
<p>我们发布了一项关于 <strong>{{.ProductName}}</strong> 的重要安全公告。为保障您的数据安全，请务必关注以下信息并采取相应措施。</p>
<p><strong>您的序列号：</strong>{{.SN}}</p>
<p>建议您尽快更新至最新版本，以获取最新的安全补丁。如有疑问，请访问 <a href="https://vantagics.com" style="color:#7c3aed">https://vantagics.com</a> 或联系我们的技术支持团队。</p>
<p style="color:#666;font-size:12px;margin-top:30px">此邮件由系统自动发送，请勿直接回复。</p>
</div>', 1, datetime('now')
		WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE name='安全公告通知' AND is_preset=1)`)

	// Migration: Update preset templates to include website URL if missing
	db.Exec(`UPDATE email_templates SET body = REPLACE(body,
		'<p>请及时更新至最新版本，以享受全部新功能。如有任何问题，请随时联系我们的技术支持团队。</p>',
		'<p>请及时更新至最新版本，以享受全部新功能。如有任何问题，请随时联系我们的技术支持团队。</p>
<p>官网地址：<a href="https://vantagics.com" style="color:#1a56db">https://vantagics.com</a></p>')
		WHERE name='产品重大更新通知' AND is_preset=1 AND body NOT LIKE '%vantagics.com%'`)

	db.Exec(`UPDATE email_templates SET body = REPLACE(body,
		'<p>我们将尽快完成维护工作，届时服务将自动恢复。感谢您的理解与支持。</p>',
		'<p>我们将尽快完成维护工作，届时服务将自动恢复。感谢您的理解与支持。</p>
<p>官网地址：<a href="https://vantagics.com" style="color:#d97706">https://vantagics.com</a></p>')
		WHERE name='服务器临时维护通知' AND is_preset=1 AND body NOT LIKE '%vantagics.com%'`)

	db.Exec(`UPDATE email_templates SET body = REPLACE(body,
		'<p>请前往官方网站下载最新版本。如需帮助，请联系我们的技术支持团队。</p>',
		'<p>请前往官方网站下载最新版本：<a href="https://vantagics.com" style="color:#059669">https://vantagics.com</a>。如需帮助，请联系我们的技术支持团队。</p>')
		WHERE name='新版本发布通知' AND is_preset=1 AND body NOT LIKE '%vantagics.com%'`)

	db.Exec(`UPDATE email_templates SET body = REPLACE(body,
		'<p>如您已完成续费，请忽略此邮件。如需帮助，请联系我们的技术支持团队。</p>',
		'<p>如您已完成续费，请忽略此邮件。如需帮助，请访问 <a href="https://vantagics.com" style="color:#dc2626">https://vantagics.com</a> 或联系我们的技术支持团队。</p>')
		WHERE name='服务到期提醒' AND is_preset=1 AND body NOT LIKE '%vantagics.com%'`)

	db.Exec(`UPDATE email_templates SET body = REPLACE(body,
		'<p>建议您尽快更新至最新版本，以获取最新的安全补丁。如有疑问，请联系我们的技术支持团队。</p>',
		'<p>建议您尽快更新至最新版本，以获取最新的安全补丁。如有疑问，请访问 <a href="https://vantagics.com" style="color:#7c3aed">https://vantagics.com</a> 或联系我们的技术支持团队。</p>')
		WHERE name='安全公告通知' AND is_preset=1 AND body NOT LIKE '%vantagics.com%'`)
}

func loadPorts() {
	var port string
	if err := db.QueryRow("SELECT value FROM settings WHERE key='manage_port'").Scan(&port); err == nil {
		fmt.Sscanf(port, "%d", &managePort)
	}
	if err := db.QueryRow("SELECT value FROM settings WHERE key='auth_port'").Scan(&port); err == nil {
		fmt.Sscanf(port, "%d", &authPort)
	}
}

func loadSSLConfig() {
	var val string
	if err := db.QueryRow("SELECT value FROM settings WHERE key='use_ssl'").Scan(&val); err == nil {
		useSSL = val == "true" || val == "1"
	}
	if err := db.QueryRow("SELECT value FROM settings WHERE key='ssl_cert'").Scan(&val); err == nil {
		sslCert = val
	}
	if err := db.QueryRow("SELECT value FROM settings WHERE key='ssl_key'").Scan(&val); err == nil {
		sslKey = val
	}
}

// getSMTPConfig retrieves SMTP configuration from database
func getSMTPConfig() SMTPConfig {
	config := SMTPConfig{
		Port:    587,
		UseTLS:  false,
		UseSTARTTLS: true,
	}
	
	if val := getSetting("smtp_enabled"); val == "true" || val == "1" {
		config.Enabled = true
	}
	if val := getSetting("smtp_host"); val != "" {
		config.Host = val
	}
	if val := getSetting("smtp_port"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Port = port
		}
	}
	if val := getSetting("smtp_username"); val != "" {
		config.Username = val
	}
	if val := getSetting("smtp_password"); val != "" {
		config.Password = val
	}
	if val := getSetting("smtp_from_email"); val != "" {
		config.FromEmail = val
	}
	if val := getSetting("smtp_from_name"); val != "" {
		config.FromName = val
	}
	if val := getSetting("smtp_use_tls"); val == "true" || val == "1" {
		config.UseTLS = true
	}
	if val := getSetting("smtp_use_starttls"); val == "true" || val == "1" {
		config.UseSTARTTLS = true
	}
	
	return config
}

// saveSMTPConfig saves SMTP configuration to database
func saveSMTPConfig(config SMTPConfig) {
	setSetting("smtp_enabled", fmt.Sprintf("%v", config.Enabled))
	setSetting("smtp_host", config.Host)
	setSetting("smtp_port", fmt.Sprintf("%d", config.Port))
	setSetting("smtp_username", config.Username)
	setSetting("smtp_password", config.Password)
	setSetting("smtp_from_email", config.FromEmail)
	setSetting("smtp_from_name", config.FromName)
	setSetting("smtp_use_tls", fmt.Sprintf("%v", config.UseTLS))
	setSetting("smtp_use_starttls", fmt.Sprintf("%v", config.UseSTARTTLS))
}

// sendEmail sends an email using SMTP
func sendEmail(to, subject, htmlBody string) error {
	config := getSMTPConfig()
	
	if !config.Enabled {
		log.Printf("[EMAIL] SMTP not enabled, skipping email to %s", to)
		return nil
	}
	
	if config.Host == "" || config.FromEmail == "" {
		return fmt.Errorf("SMTP configuration incomplete")
	}
	
	// Build email headers
	fromHeader := config.FromEmail
	if config.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail)
	}
	
	// Build message with deterministic header order
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)
	
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	
	// Create auth
	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}
	
	// Send email based on TLS configuration
	if config.UseTLS {
		// Direct TLS connection (port 465)
		return sendEmailTLS(config, to, msg.Bytes())
	} else if config.UseSTARTTLS {
		// STARTTLS connection (port 587)
		return sendEmailSTARTTLS(config, to, msg.Bytes(), auth, addr)
	} else {
		// Plain connection (not recommended)
		return smtp.SendMail(addr, auth, config.FromEmail, []string{to}, msg.Bytes())
	}
}

// sendEmailTLS sends email using direct TLS connection (port 465)
func sendEmailTLS(config SMTPConfig, to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	
	tlsConfig := &tls.Config{
		ServerName: config.Host,
	}
	
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %v", err)
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %v", err)
	}
	defer client.Close()
	
	// Auth
	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %v", err)
		}
	}
	
	// Set sender and recipient
	if err := client.Mail(config.FromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %v", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT command failed: %v", err)
	}
	
	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %v", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("SMTP write failed: %v", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("SMTP close failed: %v", err)
	}
	
	return client.Quit()
}

// sendEmailSTARTTLS sends email using STARTTLS (port 587)
func sendEmailSTARTTLS(config SMTPConfig, to string, msg []byte, auth smtp.Auth, addr string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP dial failed: %v", err)
	}
	defer client.Close()
	
	// STARTTLS
	tlsConfig := &tls.Config{
		ServerName: config.Host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %v", err)
	}
	
	// Auth
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %v", err)
		}
	}
	
	// Set sender and recipient
	if err := client.Mail(config.FromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %v", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT command failed: %v", err)
	}
	
	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %v", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("SMTP write failed: %v", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("SMTP close failed: %v", err)
	}
	
	return client.Quit()
}

// getProductName returns the product name for a given product ID
func getProductName(productID int) string {
	if productID == 0 {
		return "Vantagics"
	}
	var name string
	err := db.QueryRow("SELECT name FROM product_types WHERE id = ?", productID).Scan(&name)
	if err != nil || name == "" {
		return "Vantagics"
	}
	return name
}

// getProductInfo returns the product name and description for a given product ID
func getProductInfo(productID int) (string, string) {
	if productID == 0 {
		return "Vantagics", "Intelligent Data Analytics Platform"
	}
	var name, description string
	err := db.QueryRow("SELECT name, COALESCE(description, '') FROM product_types WHERE id = ?", productID).Scan(&name, &description)
	if err != nil || name == "" {
		return "Vantagics", "Intelligent Data Analytics Platform"
	}
	if description == "" {
		description = name
	}
	return name, description
}

// sendSNEmail sends the serial number to the user's email
func sendSNEmail(email, sn string, expiresAt time.Time, productID int) error {
	productName, productDesc := getProductInfo(productID)
	subject := fmt.Sprintf("%s - Your Serial Number", productName)
	
	daysLeft := int(expiresAt.Sub(time.Now()).Hours() / 24)
	expiryDate := expiresAt.Format("2006-01-02")
	
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.5; color: #333; margin: 0; padding: 0; background: #f0f0f0; }
        .container { max-width: 520px; margin: 15px auto; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .header h1 { margin: 0; font-size: 22px; }
        .content { background: white; padding: 20px; }
        .no-reply { background: #fff3cd; color: #856404; padding: 8px 12px; border-radius: 4px; margin-bottom: 12px; font-size: 12px; }
        .sn-box { background: #f8f9fa; border: 2px dashed #667eea; padding: 15px; text-align: center; margin: 15px 0; border-radius: 6px; }
        .sn { font-size: 22px; font-weight: bold; color: #667eea; letter-spacing: 2px; font-family: 'Courier New', monospace; }
        .info { background: #e8f4fd; padding: 12px 15px; border-radius: 6px; margin: 12px 0; font-size: 13px; }
        .info p { margin: 4px 0; }
        .info ol { margin: 6px 0; padding-left: 18px; }
        .info li { margin: 2px 0; }
        .help { font-size: 13px; margin-top: 12px; }
        .footer { background: #f8f9fa; padding: 12px; text-align: center; border-radius: 0 0 8px 8px; border-top: 1px solid #eee; }
        .footer p { margin: 2px 0; font-size: 11px; color: #888; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 %s - Your Serial Number</h1>
        </div>
        <div class="content">
            <div class="no-reply">⚠️ This is an automated message. Please do not reply.</div>
            <p style="margin:0 0 10px 0;">Thank you for requesting a %s serial number:</p>
            <div class="sn-box">
                <div class="sn">%s</div>
            </div>
            <div class="info">
                <p><strong>📅 Valid until:</strong> %s (%d days)</p>
                <p><strong>💡 How to use:</strong> Open %s → Select Commercial Mode → Enter serial number → Activate</p>
            </div>
            <p class="help">Questions? Visit <a href="https://vantagics.com" style="color:#667eea;">vantagics.com</a></p>
        </div>
        <div class="footer">
            <p>© %s - %s</p>
        </div>
    </div>
</body>
</html>
`, productName, productName, sn, expiryDate, daysLeft, productName, productName, productDesc)
	
	return sendEmail(email, subject, htmlBody)
}

func getSetting(key string) string {
	var value string
	db.QueryRow("SELECT value FROM settings WHERE key=?", key).Scan(&value)
	return value
}

func setSetting(key, value string) {
	db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
}

// isValidIdentifier checks if a string is safe to use as a SQL identifier.
// Only allows alphanumeric characters, underscores, and common table/column name patterns.
func isValidIdentifier(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func generateSN() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use math/rand (auto-seeded in Go 1.20+)
		log.Printf("Warning: crypto/rand failed: %v, using fallback", err)
		for i := range b {
			b[i] = charset[mrand.Intn(len(charset))]
		}
	} else {
		for i := range b {
			b[i] = charset[int(b[i])%len(charset)]
		}
	}
	return fmt.Sprintf("%s-%s-%s-%s", string(b[0:4]), string(b[4:8]), string(b[8:12]), string(b[12:16]))
}

func generateShortID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use math/rand (auto-seeded in Go 1.20+)
		log.Printf("Warning: crypto/rand failed: %v, using fallback", err)
		for i := range b {
			b[i] = byte(mrand.Intn(256))
		}
	}
	return hex.EncodeToString(b)
}

func encryptData(data []byte, sn string) (string, error) {
	hash := sha256.Sum256([]byte(sn))
	key := hash[:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func isKeyValidForDate(startDate, endDate, checkDate string) bool {
	if startDate == "" {
		startDate = "1970-01-01"
	}
	if checkDate < startDate {
		return false
	}
	if endDate == "" {
		return true
	}
	return checkDate <= endDate
}

func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	ip := r.RemoteAddr
	// Handle IPv6 addresses like [::1]:port
	if strings.HasPrefix(ip, "[") {
		if bracketIdx := strings.LastIndex(ip, "]"); bracketIdx != -1 {
			return ip[1:bracketIdx]
		}
	}
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}

// ============ Management Server ============

func startManageServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleLogin)
	mux.HandleFunc("/login", handleLoginPost)
	mux.HandleFunc("/dashboard", authMiddleware(handleDashboard))
	mux.HandleFunc("/logout", handleLogout)
	mux.HandleFunc("/api/licenses", authMiddleware(handleLicenses))
	mux.HandleFunc("/api/licenses/create", authMiddleware(handleCreateLicense))
	mux.HandleFunc("/api/licenses/batch-create", authMiddleware(handleBatchCreateLicense))
	mux.HandleFunc("/api/licenses/delete", authMiddleware(handleDeleteLicense))
	mux.HandleFunc("/api/licenses/toggle", authMiddleware(handleToggleLicense))
	mux.HandleFunc("/api/licenses/extend", authMiddleware(handleExtendLicense))
	mux.HandleFunc("/api/licenses/set-daily", authMiddleware(handleSetDailyAnalysis))
	mux.HandleFunc("/api/licenses/set-credits", authMiddleware(handleSetCredits))
	mux.HandleFunc("/api/licenses/set-groups", authMiddleware(handleSetLicenseGroups))
	mux.HandleFunc("/api/licenses/search", authMiddleware(handleSearchLicenses))
	mux.HandleFunc("/api/licenses/delete-unused-by-group", authMiddleware(handleDeleteUnusedByGroup))
	mux.HandleFunc("/api/licenses/purge-disabled", authMiddleware(handlePurgeDisabledLicenses))
	mux.HandleFunc("/api/licenses/force-delete", authMiddleware(handleForceDeleteLicense))
	mux.HandleFunc("/api/llm", authMiddleware(handleLLMConfig))
	mux.HandleFunc("/api/llm-groups", authMiddleware(handleLLMGroups))
	mux.HandleFunc("/api/search", authMiddleware(handleSearchConfig))
	mux.HandleFunc("/api/search-groups", authMiddleware(handleSearchGroups))
	mux.HandleFunc("/api/license-groups", authMiddleware(handleLicenseGroups))
	mux.HandleFunc("/api/license-groups/config", authMiddleware(handleLicenseGroupConfig))
	mux.HandleFunc("/api/product-types", authMiddleware(handleProductTypes))
	mux.HandleFunc("/api/product-extra-info", authMiddleware(handleProductExtraInfo))
	mux.HandleFunc("/api/password", authMiddleware(handleChangePassword))
	mux.HandleFunc("/api/username", authMiddleware(handleChangeUsername))
	mux.HandleFunc("/api/ports", authMiddleware(handleChangePorts))
	mux.HandleFunc("/api/ssl", authMiddleware(handleSSLConfig))
	mux.HandleFunc("/api/smtp", authMiddleware(handleSMTPConfig))
	mux.HandleFunc("/api/smtp/test", authMiddleware(handleSMTPTest))
	mux.HandleFunc("/api/settings/request-limits", authMiddleware(handleRequestLimits))
	mux.HandleFunc("/api/email-records", authMiddleware(handleEmailRecords))
	mux.HandleFunc("/api/email-records/update", authMiddleware(handleUpdateEmailRecord))
	mux.HandleFunc("/api/email-records/manual-request", authMiddleware(handleManualRequest))
	mux.HandleFunc("/api/email-records/manual-bind", authMiddleware(handleManualBind))
	mux.HandleFunc("/api/email-records/clear-by-email", authMiddleware(handleClearEmailRecords))
	mux.HandleFunc("/api/settings/clear-ip-records", authMiddleware(handleClearIPRecords))
	mux.HandleFunc("/api/api-keys", authMiddleware(handleAPIKeys))
	mux.HandleFunc("/api/api-keys/toggle", authMiddleware(handleToggleAPIKey))
	mux.HandleFunc("/api/api-keys/bindings", authMiddleware(handleAPIKeyBindings))
	mux.HandleFunc("/api/api-keys/clear-bindings", authMiddleware(handleClearAPIKeyBindings))
	mux.HandleFunc("/api/email-filter", authMiddleware(handleEmailFilter))
	mux.HandleFunc("/api/whitelist", authMiddleware(handleWhitelist))
	mux.HandleFunc("/api/blacklist", authMiddleware(handleBlacklist))
	mux.HandleFunc("/api/conditions", authMiddleware(handleConditions))
	mux.HandleFunc("/api/backup/settings", authMiddleware(handleBackupSettings))
	mux.HandleFunc("/api/backup/create", authMiddleware(handleBackupCreate))
	mux.HandleFunc("/api/backup/restore", authMiddleware(handleBackupRestore))
	mux.HandleFunc("/api/backup/history", authMiddleware(handleBackupHistory))
	mux.HandleFunc("/api/credits-usage-log", authMiddleware(handleCreditsUsageLog))
	mux.HandleFunc("/api/email-templates", authMiddleware(handleEmailTemplates))
	mux.HandleFunc("/api/email-templates/delete", authMiddleware(handleDeleteEmailTemplate))
	mux.HandleFunc("/api/email-notify/recipients", authMiddleware(handleEmailNotifyRecipients))
	mux.HandleFunc("/api/email-notify/send", authMiddleware(handleEmailNotifySend))
	mux.HandleFunc("/api/email-notify/progress/", authMiddleware(handleEmailNotifyProgress))
	mux.HandleFunc("/api/email-notify/cancel/", authMiddleware(handleEmailNotifyCancel))
	mux.HandleFunc("/api/email-history", authMiddleware(handleEmailHistory))
	mux.HandleFunc("/api/email-history/", authMiddleware(handleEmailHistoryDetail))

	// Wrap mux with security headers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if useSSL {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		mux.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", managePort)
	if useSSL && sslCert != "" && sslKey != "" {
		log.Printf("Management server starting on %s (HTTPS)", addr)
		if err := http.ListenAndServeTLS(addr, sslCert, sslKey, handler); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Management server failed: %v", err)
		}
	} else {
		log.Printf("Management server starting on %s (HTTP)", addr)
		if err := http.ListenAndServe(addr, handler); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Management server failed: %v", err)
		}
	}
}

var sessions = make(map[string]time.Time)
var sessionLock sync.RWMutex

// Captcha storage: maps captchaID -> {answer, expiresAt}
type captchaEntry struct {
	answer    string
	expiresAt time.Time
}
var captchas = make(map[string]captchaEntry)
var captchaLock sync.RWMutex

func init() {
	// Periodic cleanup of expired sessions and captchas every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			
			sessionLock.Lock()
			for token, expiry := range sessions {
				if now.After(expiry) {
					delete(sessions, token)
				}
			}
			sessionLock.Unlock()
			
			captchaLock.Lock()
			for id, entry := range captchas {
				if now.After(entry.expiresAt) {
					delete(captchas, id)
				}
			}
			captchaLock.Unlock()
		}
	}()
}

func createSession() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use math/rand (auto-seeded in Go 1.20+)
		log.Printf("Warning: crypto/rand failed for session token: %v, using fallback", err)
		for i := range b {
			b[i] = byte(mrand.Intn(256))
		}
	}
	token := hex.EncodeToString(b)
	sessionLock.Lock()
	sessions[token] = time.Now().Add(24 * time.Hour)
	sessionLock.Unlock()
	return token
}

func validateSession(token string) bool {
	sessionLock.RLock()
	expiry, exists := sessions[token]
	sessionLock.RUnlock()
	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

// Simple digit font patterns (5x7 pixels each)
var digitPatterns = map[rune][]string{
	'0': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00110", "01000", "10000", "11111"},
	'3': {"01110", "10001", "00001", "00110", "00001", "10001", "01110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'*': {"00000", "10101", "01110", "11111", "01110", "10101", "00000"},
	'/': {"00001", "00010", "00010", "00100", "01000", "01000", "10000"},
	'=': {"00000", "00000", "11111", "00000", "11111", "00000", "00000"},
	'?': {"01110", "10001", "00001", "00110", "00100", "00000", "00100"},
}

// Generate math expression captcha - returns captchaID and base64 image
func generateCaptcha() (string, string) {
	// math/rand is auto-seeded in Go 1.20+
	
	// Generate a math expression: one-digit op two-digit or two-digit op one-digit
	var num1, num2, result int
	var expression string
	
	// Randomly choose operation
	opIndex := mrand.Intn(4)
	
	switch opIndex {
	case 0: // Addition
		num1 = mrand.Intn(9) + 1      // 1-9
		num2 = mrand.Intn(90) + 10    // 10-99
		if mrand.Intn(2) == 0 {
			num1, num2 = num2, num1
		}
		result = num1 + num2
		expression = fmt.Sprintf("%d+%d=?", num1, num2)
	case 1: // Subtraction
		num2 = mrand.Intn(9) + 1      // 1-9
		num1 = mrand.Intn(90) + 10    // 10-99
		result = num1 - num2
		expression = fmt.Sprintf("%d-%d=?", num1, num2)
	case 2: // Multiplication
		num1 = mrand.Intn(9) + 1      // 1-9
		num2 = mrand.Intn(9) + 2      // 2-10
		if mrand.Intn(2) == 0 {
			num1, num2 = num2, num1
		}
		result = num1 * num2
		expression = fmt.Sprintf("%d*%d=?", num1, num2)
	case 3: // Division (ensure clean division)
		num2 = mrand.Intn(8) + 2      // 2-9 (divisor)
		result = mrand.Intn(9) + 2    // 2-10 (quotient)
		num1 = num2 * result          // dividend
		expression = fmt.Sprintf("%d/%d=?", num1, num2)
	}
	
	// Generate captcha ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		// Fallback: use math/rand (auto-seeded in Go 1.20+)
		log.Printf("Warning: crypto/rand failed for captcha ID: %v, using fallback", err)
		for i := range idBytes {
			idBytes[i] = byte(mrand.Intn(256))
		}
	}
	captchaID := hex.EncodeToString(idBytes)
	
	// Store captcha answer with 5 minute expiry
	answer := fmt.Sprintf("%d", result)
	captchaLock.Lock()
	captchas[captchaID] = captchaEntry{answer: answer, expiresAt: time.Now().Add(5 * time.Minute)}
	captchaLock.Unlock()
	
	// Generate image with the expression
	captchaImage := generateCaptchaImage(expression)
	
	return captchaID, captchaImage
}

func generateCaptchaImage(code string) string {
	// Calculate width based on expression length (e.g., "12+34=?" = 7 chars)
	charWidth := 18
	width := len(code)*charWidth + 20
	height := 40
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Background with slight color variation
	bgColor := color.RGBA{
		uint8(240 + mrand.Intn(15)),
		uint8(240 + mrand.Intn(15)),
		uint8(240 + mrand.Intn(15)),
		255,
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bgColor)
		}
	}
	
	// Add noise dots
	for i := 0; i < 100; i++ {
		x := mrand.Intn(width)
		y := mrand.Intn(height)
		c := color.RGBA{
			uint8(mrand.Intn(200)),
			uint8(mrand.Intn(200)),
			uint8(mrand.Intn(200)),
			255,
		}
		img.Set(x, y, c)
	}
	
	// Draw interference lines
	for i := 0; i < 4; i++ {
		lineColor := color.RGBA{
			uint8(100 + mrand.Intn(100)),
			uint8(100 + mrand.Intn(100)),
			uint8(100 + mrand.Intn(100)),
			255,
		}
		x1, y1 := mrand.Intn(width), mrand.Intn(height)
		x2, y2 := mrand.Intn(width), mrand.Intn(height)
		drawLine(img, x1, y1, x2, y2, lineColor)
	}
	
	// Draw characters with random colors and positions
	digitColors := []color.RGBA{
		{180, 50, 50, 255},   // Red
		{50, 50, 180, 255},   // Blue
		{50, 130, 50, 255},   // Green
		{130, 50, 130, 255},  // Purple
		{50, 130, 130, 255},  // Teal
	}
	
	startX := 10
	for i, char := range code {
		digitColor := digitColors[mrand.Intn(len(digitColors))]
		offsetY := mrand.Intn(8) - 4 // Random vertical offset
		drawDigit(img, char, startX+i*charWidth, 8+offsetY, digitColor)
	}
	
	// Encode to base64
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
	steps := int(math.Max(dx, dy))
	if steps == 0 {
		return
	}
	xInc := float64(x2-x1) / float64(steps)
	yInc := float64(y2-y1) / float64(steps)
	x, y := float64(x1), float64(y1)
	for i := 0; i <= steps; i++ {
		if int(x) >= 0 && int(x) < img.Bounds().Dx() && int(y) >= 0 && int(y) < img.Bounds().Dy() {
			img.Set(int(x), int(y), c)
		}
		x += xInc
		y += yInc
	}
}

func drawDigit(img *image.RGBA, digit rune, startX, startY int, c color.RGBA) {
	pattern, ok := digitPatterns[digit]
	if !ok {
		return
	}
	scale := 3 // Scale factor for larger digits
	for row, line := range pattern {
		for col, ch := range line {
			if ch == '1' {
				// Draw scaled pixel
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						x := startX + col*scale + dx
						y := startY + row*scale + dy
						if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
							img.Set(x, y, c)
						}
					}
				}
			}
		}
	}
}

func validateCaptcha(captchaID, answer string) bool {
	captchaLock.RLock()
	entry, exists := captchas[captchaID]
	captchaLock.RUnlock()
	
	if !exists || time.Now().After(entry.expiresAt) {
		return false
	}
	
	// Delete captcha after use (one-time use)
	captchaLock.Lock()
	delete(captchas, captchaID)
	captchaLock.Unlock()
	
	return strings.TrimSpace(answer) == entry.answer
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil || !validateSession(cookie.Value) {
			// For API requests, return JSON error instead of redirect
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "未登录或会话已过期", "error": "未登录或会话已过期"})
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	captchaID, captchaImage := generateCaptcha()
	tmpl := template.Must(template.New("login").Parse(templates.LoginHTML))
	tmpl.Execute(w, map[string]interface{}{
		"CaptchaID":    captchaID,
		"CaptchaImage": captchaImage,
		"ManagePort":   managePort,
		"AuthPort":     authPort,
	})
}

func checkLoginAllowed(ip string) (bool, string) {
	loginAttemptsLock.Lock()
	defer loginAttemptsLock.Unlock()

	info, exists := loginAttempts[ip]
	if !exists {
		return true, ""
	}

	if info.Locked {
		return false, "登录已被永久锁定，请在服务器上使用密码重置工具解锁"
	}

	if info.FailCount >= 5 {
		elapsed := time.Since(info.LastFailTime)
		if elapsed < time.Hour {
			remaining := time.Hour - elapsed
			mins := int(remaining.Minutes()) + 1
			return false, fmt.Sprintf("密码错误次数过多，请在 %d 分钟后重试", mins)
		}
		// Cooldown passed, reset to allow retry but keep accumulating toward permanent lock
		info.FailCount = 0
	}

	return true, ""
}

func recordLoginFailure(ip string) {
	loginAttemptsLock.Lock()
	defer loginAttemptsLock.Unlock()

	info, exists := loginAttempts[ip]
	if !exists {
		info = &loginAttemptInfo{}
		loginAttempts[ip] = info
	}

	info.FailCount++
	info.TotalFailCount++
	info.LastFailTime = time.Now()

	if info.TotalFailCount >= 15 {
		info.Locked = true
		log.Printf("[LOGIN] IP %s permanently locked after %d total failed attempts", ip, info.TotalFailCount)
	} else if info.FailCount >= 5 {
		log.Printf("[LOGIN] IP %s temporarily locked after %d failed attempts (total: %d)", ip, info.FailCount, info.TotalFailCount)
	}
}

func clearLoginFailures(ip string) {
	loginAttemptsLock.Lock()
	defer loginAttemptsLock.Unlock()
	delete(loginAttempts, ip)
}

func resetAllLoginLocks() {
	loginAttemptsLock.Lock()
	defer loginAttemptsLock.Unlock()
	loginAttempts = make(map[string]*loginAttemptInfo)
	log.Printf("[LOGIN] All login locks cleared")
}

func handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	clientIP := getClientIP(r)

	// Check if login is allowed for this IP
	allowed, lockMsg := checkLoginAllowed(clientIP)
	if !allowed {
		newCaptchaID, newCaptchaImage := generateCaptcha()
		tmpl := template.Must(template.New("login").Parse(templates.LoginHTML))
		tmpl.Execute(w, map[string]interface{}{
			"Error":        lockMsg,
			"CaptchaID":    newCaptchaID,
			"CaptchaImage": newCaptchaImage,
			"ManagePort":   managePort,
			"AuthPort":     authPort,
		})
		return
	}
	
	username := r.FormValue("username")
	password := r.FormValue("password")
	captchaID := r.FormValue("captcha_id")
	captchaAnswer := r.FormValue("captcha")
	
	// Validate captcha first
	if !validateCaptcha(captchaID, captchaAnswer) {
		newCaptchaID, newCaptchaImage := generateCaptcha()
		tmpl := template.Must(template.New("login").Parse(templates.LoginHTML))
		tmpl.Execute(w, map[string]interface{}{
			"Error":        "验证码错误",
			"CaptchaID":    newCaptchaID,
			"CaptchaImage": newCaptchaImage,
			"ManagePort":   managePort,
			"AuthPort":     authPort,
		})
		return
	}
	
	validUsername := getSetting("admin_username")
	if validUsername == "" {
		validUsername = "admin"
	}
	validPassword := getSetting("admin_password")
	if username != validUsername || !checkAdminPassword(password, validPassword) {
		recordLoginFailure(clientIP)

		// Check lock status after recording failure
		_, lockMsg := checkLoginAllowed(clientIP)
		errorMsg := "用户名或密码错误"
		if lockMsg != "" {
			errorMsg = lockMsg
		}

		newCaptchaID, newCaptchaImage := generateCaptcha()
		tmpl := template.Must(template.New("login").Parse(templates.LoginHTML))
		tmpl.Execute(w, map[string]interface{}{
			"Error":        errorMsg,
			"CaptchaID":    newCaptchaID,
			"CaptchaImage": newCaptchaImage,
			"ManagePort":   managePort,
			"AuthPort":     authPort,
		})
		return
	}

	// Login success - clear failure records
	clearLoginFailures(clientIP)

	token := createSession()
	http.SetCookie(w, &http.Cookie{Name: "session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: useSSL, MaxAge: 86400})
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := templates.GetDashboardHTML()
	html = strings.Replace(html, "{{.Username}}", template.HTMLEscapeString(getSetting("admin_username")), -1)
	html = strings.Replace(html, "{{.ManagePort}}", strconv.Itoa(managePort), -1)
	html = strings.Replace(html, "{{.AuthPort}}", strconv.Itoa(authPort), -1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}


func handleLicenses(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT sn, created_at, expires_at, description, is_active, usage_count, last_used_at, COALESCE(daily_analysis, 20), COALESCE(license_group_id, ''), COALESCE(llm_group_id, ''), COALESCE(search_group_id, '') FROM licenses ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	licenses := make(map[string]License)
	for rows.Next() {
		var l License
		var lastUsed sql.NullTime
		if err := rows.Scan(&l.SN, &l.CreatedAt, &l.ExpiresAt, &l.Description, &l.IsActive, &l.UsageCount, &lastUsed, &l.DailyAnalysis, &l.LicenseGroupID, &l.LLMGroupID, &l.SearchGroupID); err != nil {
			log.Printf("[handleLicenses] scan error: %v", err)
			continue
		}
		if lastUsed.Valid {
			l.LastUsedAt = lastUsed.Time
		}
		licenses[l.SN] = l
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(licenses)
}

func handleCreateLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Description    string  `json:"description"`
		Days           int     `json:"days"`
		DailyAnalysis  int     `json:"daily_analysis"`
		LLMGroupID     string  `json:"llm_group_id"`
		LicenseGroupID string  `json:"license_group_id"`
		SearchGroupID  string  `json:"search_group_id"`
		ProductID      int     `json:"product_id"`
		TotalCredits   float64 `json:"total_credits"`
		CreditsMode    bool    `json:"credits_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Days <= 0 {
		req.Days = 365
	}
	if req.DailyAnalysis < 0 {
		req.DailyAnalysis = 20
	}
	if req.TotalCredits < 0 {
		req.TotalCredits = 0
	}
	sn := generateSN()
	now := time.Now()
	expires := now.AddDate(0, 0, req.Days)
	
	_, err := db.Exec("INSERT INTO licenses (sn, created_at, expires_at, description, is_active, daily_analysis, license_group_id, llm_group_id, search_group_id, product_id, total_credits, credits_mode) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?)",
		sn, now, expires, req.Description, req.DailyAnalysis, req.LicenseGroupID, req.LLMGroupID, req.SearchGroupID, req.ProductID, req.TotalCredits, req.CreditsMode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	license := License{SN: sn, CreatedAt: now, ExpiresAt: expires, Description: req.Description, IsActive: true, DailyAnalysis: req.DailyAnalysis, LLMGroupID: req.LLMGroupID, SearchGroupID: req.SearchGroupID, ProductID: req.ProductID, TotalCredits: req.TotalCredits, CreditsMode: req.CreditsMode}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(license)
}

func handleDeleteLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SN string `json:"sn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}
	if req.SN == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不能为空"})
		return
	}
	result, err := db.Exec("DELETE FROM licenses WHERE sn=?", req.SN)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不存在"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleToggleLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SN string `json:"sn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}
	if req.SN == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不能为空"})
		return
	}
	db.Exec("UPDATE licenses SET is_active = NOT is_active WHERE sn=?", req.SN)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleBatchCreateLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Description    string  `json:"description"`
		Days           int     `json:"days"`
		Count          int     `json:"count"`
		DailyAnalysis  int     `json:"daily_analysis"`
		LLMGroupID     string  `json:"llm_group_id"`
		LicenseGroupID string  `json:"license_group_id"`
		SearchGroupID  string  `json:"search_group_id"`
		ProductID      int     `json:"product_id"`
		TotalCredits   float64 `json:"total_credits"`
		CreditsMode    bool    `json:"credits_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Days <= 0 {
		req.Days = 365
	}
	if req.Count <= 0 {
		req.Count = 100
	}
	if req.Count > 1000 {
		req.Count = 1000 // Max 1000 at a time
	}
	if req.DailyAnalysis < 0 {
		req.DailyAnalysis = 20
	}
	if req.TotalCredits < 0 {
		req.TotalCredits = 0
	}
	
	now := time.Now()
	// expires_at is NULL until SN is bound to email
	var created []string
	
	for i := 0; i < req.Count; i++ {
		sn := generateSN()
		_, err := db.Exec("INSERT INTO licenses (sn, created_at, expires_at, valid_days, description, is_active, daily_analysis, license_group_id, llm_group_id, search_group_id, product_id, total_credits, credits_mode) VALUES (?, ?, NULL, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?)",
			sn, now, req.Days, req.Description, req.DailyAnalysis, req.LicenseGroupID, req.LLMGroupID, req.SearchGroupID, req.ProductID, req.TotalCredits, req.CreditsMode)
		if err == nil {
			created = append(created, sn)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(created),
		"sns":     created,
	})
}

func handleExtendLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SN   string `json:"sn"`
		Days int    `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Days <= 0 {
		req.Days = 30
	}
	
	// Get current expiry date
	var expiresAt time.Time
	err := db.QueryRow("SELECT expires_at FROM licenses WHERE sn=?", req.SN).Scan(&expiresAt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不存在"})
		return
	}
	
	// Extend from current expiry or from now if already expired
	baseDate := expiresAt
	if time.Now().After(expiresAt) {
		baseDate = time.Now()
	}
	newExpiry := baseDate.AddDate(0, 0, req.Days)
	
	db.Exec("UPDATE licenses SET expires_at=? WHERE sn=?", newExpiry, req.SN)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"new_expiry": newExpiry.Format("2006-01-02"),
	})
}

func handleSetDailyAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SN            string `json:"sn"`
		DailyAnalysis int    `json:"daily_analysis"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.DailyAnalysis < 0 {
		req.DailyAnalysis = 0
	}
	
	result, err := db.Exec("UPDATE licenses SET daily_analysis=? WHERE sn=?", req.DailyAnalysis, req.SN)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不存在"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func handleSetCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SN           string  `json:"sn"`
		TotalCredits float64 `json:"total_credits"`
		CreditsMode  bool    `json:"credits_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.TotalCredits < 0 {
		req.TotalCredits = 0
	}

	result, err := db.Exec("UPDATE licenses SET total_credits=?, credits_mode=? WHERE sn=?", req.TotalCredits, req.CreditsMode, req.SN)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不存在"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}


func handleSetLicenseGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SN             string `json:"sn"`
		ProductID      int    `json:"product_id"`
		LicenseGroupID string `json:"license_group_id"`
		LLMGroupID     string `json:"llm_group_id"`
		SearchGroupID  string `json:"search_group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := db.Exec("UPDATE licenses SET product_id=?, license_group_id=?, llm_group_id=?, search_group_id=? WHERE sn=?", req.ProductID, req.LicenseGroupID, req.LLMGroupID, req.SearchGroupID, req.SN)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不存在"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func handleDeleteUnusedByGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		LicenseGroupID string `json:"license_group_id"`
		LLMGroupID     string `json:"llm_group_id"`
		SearchGroupID  string `json:"search_group_id"`
		DeleteAll      bool   `json:"delete_all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Build WHERE clause for unused licenses (usage_count = 0)
	whereConditions := []string{"usage_count = 0"}
	args := []interface{}{}
	
	// Check if any group filter is specified
	hasFilter := false
	
	if req.LicenseGroupID != "" {
		hasFilter = true
		if req.LicenseGroupID == "none" {
			whereConditions = append(whereConditions, "(license_group_id IS NULL OR license_group_id = '')")
		} else {
			whereConditions = append(whereConditions, "license_group_id = ?")
			args = append(args, req.LicenseGroupID)
		}
	}
	
	if req.LLMGroupID != "" {
		hasFilter = true
		if req.LLMGroupID == "none" {
			whereConditions = append(whereConditions, "(llm_group_id IS NULL OR llm_group_id = '')")
		} else {
			whereConditions = append(whereConditions, "llm_group_id = ?")
			args = append(args, req.LLMGroupID)
		}
	}
	
	if req.SearchGroupID != "" {
		hasFilter = true
		if req.SearchGroupID == "none" {
			whereConditions = append(whereConditions, "(search_group_id IS NULL OR search_group_id = '')")
		} else {
			whereConditions = append(whereConditions, "search_group_id = ?")
			args = append(args, req.SearchGroupID)
		}
	}
	
	// If no filter and delete_all is not explicitly set, reject
	if !hasFilter && !req.DeleteAll {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "必须指定至少一个分组条件，或明确指定删除全部"})
		return
	}
	
	whereClause := " WHERE " + strings.Join(whereConditions, " AND ")
	
	// First, count how many will be deleted
	var count int
	countQuery := "SELECT COUNT(*) FROM licenses" + whereClause
	err := db.QueryRow(countQuery, args...).Scan(&count)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	if count == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "deleted": 0, "message": "没有符合条件的未使用序列号"})
		return
	}
	
	// Delete the licenses
	deleteQuery := "DELETE FROM licenses" + whereClause
	result, err := db.Exec(deleteQuery, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	deleted, _ := result.RowsAffected()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   deleted,
		"deleted": deleted,
		"message": fmt.Sprintf("成功删除 %d 个未使用的序列号", deleted),
	})
}

func handlePurgeDisabledLicenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Count disabled licenses that are NOT bound to any email
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM licenses 
		WHERE is_active = 0 
		AND sn NOT IN (SELECT sn FROM email_records)`).Scan(&count)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	if count == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "deleted": 0, "message": "没有可清除的序列号（已禁用且未绑定邮箱）"})
		return
	}
	
	// Get the SNs to be deleted for logging
	rows, err := db.Query(`SELECT sn FROM licenses 
		WHERE is_active = 0 
		AND sn NOT IN (SELECT sn FROM email_records)`)
	var sns []string
	if err != nil {
		log.Printf("Warning: failed to query SNs for logging: %v", err)
	} else {
		for rows.Next() {
			var sn string
			rows.Scan(&sn)
			sns = append(sns, sn)
		}
		if err := rows.Err(); err != nil {
			log.Printf("Warning: rows iteration error: %v", err)
		}
		rows.Close() // Close before DELETE to avoid lock contention
	}
	
	// Delete disabled licenses that are not bound to email
	result, err := db.Exec(`DELETE FROM licenses 
		WHERE is_active = 0 
		AND sn NOT IN (SELECT sn FROM email_records)`)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	deleted, _ := result.RowsAffected()
	log.Printf("[PURGE] Permanently deleted %d disabled licenses (not bound to email): %v", deleted, sns)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"deleted": deleted,
		"message": fmt.Sprintf("成功清除 %d 个已禁用且未绑定邮箱的序列号", deleted),
	})
}

func handleForceDeleteLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		SN string `json:"sn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求"})
		return
	}
	
	sn := strings.TrimSpace(strings.ToUpper(req.SN))
	if sn == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不能为空"})
		return
	}
	
	// Check if license exists in licenses table
	var licenseExists int
	db.QueryRow("SELECT COUNT(*) FROM licenses WHERE sn = ?", sn).Scan(&licenseExists)
	
	// Check if email record exists
	var emailExists int
	db.QueryRow("SELECT COUNT(*) FROM email_records WHERE sn = ?", sn).Scan(&emailExists)
	
	if licenseExists == 0 && emailExists == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不存在（licenses 表和 email_records 表中都没有找到）"})
		return
	}
	
	var description string
	var isActive bool
	var email string
	
	// Get license info for logging (if exists)
	if licenseExists > 0 {
		db.QueryRow("SELECT description, is_active FROM licenses WHERE sn = ?", sn).Scan(&description, &isActive)
	}
	
	// Get email info (if exists)
	if emailExists > 0 {
		db.QueryRow("SELECT email FROM email_records WHERE sn = ?", sn).Scan(&email)
	}
	
	// Delete from email_records first
	emailResult, _ := db.Exec("DELETE FROM email_records WHERE sn = ?", sn)
	emailDeleted, _ := emailResult.RowsAffected()
	
	// Delete the license (if exists)
	var licenseDeleted int64
	if licenseExists > 0 {
		result, err := db.Exec("DELETE FROM licenses WHERE sn = ?", sn)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		licenseDeleted, _ = result.RowsAffected()
	}
	
	log.Printf("[FORCE-DELETE] License %s forcefully deleted (license existed: %v, was active: %v, description: %s, bound email: %s)", sn, licenseExists > 0, isActive, description, email)
	
	var messages []string
	if licenseDeleted > 0 {
		messages = append(messages, "序列号已删除")
	}
	if emailDeleted > 0 {
		messages = append(messages, fmt.Sprintf("删除了 %d 条邮箱申请记录", emailDeleted))
	}
	if licenseExists == 0 && emailDeleted > 0 {
		messages = append(messages, "（注意：序列号本身不存在于 licenses 表，只清理了孤立的邮箱记录）")
	}
	
	message := strings.Join(messages, "，")
	if message == "" {
		message = "没有找到需要删除的记录"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
	})
}

func handleSearchLicenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	query := r.URL.Query()
	search := strings.ToLower(query.Get("search"))
	llmGroupFilter := query.Get("llm_group")
	searchGroupFilter := query.Get("search_group")
	licenseGroupFilter := query.Get("license_group")
	productFilter := query.Get("product_id")
	hideUsed := query.Get("hide_used") != "false" // Default to hide used (bound to email)
	page, pageSize := 1, 20
	fmt.Sscanf(query.Get("page"), "%d", &page)
	fmt.Sscanf(query.Get("pageSize"), "%d", &pageSize)
	if page < 1 { page = 1 }
	if pageSize < 1 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	
	var total int
	var rows *sql.Rows
	var err error
	
	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	
	// Hide licenses that are bound to email (already used)
	if hideUsed {
		whereConditions = append(whereConditions, "sn NOT IN (SELECT sn FROM email_records)")
	}
	
	if search != "" {
		searchLower := strings.ToLower(search)
		whereConditions = append(whereConditions, "(LOWER(sn) LIKE ? OR LOWER(description) LIKE ?)")
		args = append(args, "%"+searchLower+"%", "%"+searchLower+"%")
	}
	
	if llmGroupFilter != "" {
		if llmGroupFilter == "none" {
			whereConditions = append(whereConditions, "(llm_group_id IS NULL OR llm_group_id = '')")
		} else {
			whereConditions = append(whereConditions, "llm_group_id = ?")
			args = append(args, llmGroupFilter)
		}
	}
	
	if searchGroupFilter != "" {
		if searchGroupFilter == "none" {
			whereConditions = append(whereConditions, "(search_group_id IS NULL OR search_group_id = '')")
		} else {
			whereConditions = append(whereConditions, "search_group_id = ?")
			args = append(args, searchGroupFilter)
		}
	}
	
	if licenseGroupFilter != "" {
		if licenseGroupFilter == "none" {
			whereConditions = append(whereConditions, "(license_group_id IS NULL OR license_group_id = '')")
		} else {
			whereConditions = append(whereConditions, "license_group_id = ?")
			args = append(args, licenseGroupFilter)
		}
	}
	
	if productFilter != "" {
		if productFilter == "0" || productFilter == "none" {
			whereConditions = append(whereConditions, "(product_id IS NULL OR product_id = 0)")
		} else {
			var productID int
			fmt.Sscanf(productFilter, "%d", &productID)
			whereConditions = append(whereConditions, "product_id = ?")
			args = append(args, productID)
		}
	}
	
	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = " WHERE " + strings.Join(whereConditions, " AND ")
	}
	
	// Count total
	countQuery := "SELECT COUNT(*) FROM licenses" + whereClause
	db.QueryRow(countQuery, args...).Scan(&total)
	
	// Get paginated results
	selectQuery := `SELECT sn, created_at, expires_at, COALESCE(valid_days, 365), description, is_active, usage_count, last_used_at, COALESCE(daily_analysis, 20), COALESCE(license_group_id, ''), COALESCE(llm_group_id, ''), COALESCE(search_group_id, ''), COALESCE(product_id, 0), COALESCE(total_credits, 0), COALESCE(credits_mode, 0), COALESCE(used_credits, 0) 
		FROM licenses` + whereClause + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err = db.Query(selectQuery, args...)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var licenses []License
	for rows.Next() {
		var l License
		var lastUsed sql.NullTime
		var expiresAt sql.NullTime
		if err := rows.Scan(&l.SN, &l.CreatedAt, &expiresAt, &l.ValidDays, &l.Description, &l.IsActive, &l.UsageCount, &lastUsed, &l.DailyAnalysis, &l.LicenseGroupID, &l.LLMGroupID, &l.SearchGroupID, &l.ProductID, &l.TotalCredits, &l.CreditsMode, &l.UsedCredits); err != nil {
			log.Printf("[handleSearchLicenses] scan error: %v", err)
			continue
		}
		if lastUsed.Valid {
			l.LastUsedAt = lastUsed.Time
		}
		if expiresAt.Valid {
			l.ExpiresAt = expiresAt.Time
		}
		licenses = append(licenses, l)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
		return
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 { totalPages = 1 }
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"licenses": licenses, "total": total, "page": page, "pageSize": pageSize, "totalPages": totalPages,
	})
}

func handleLLMConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, name, type, base_url, api_key, model, is_active, start_date, end_date, COALESCE(group_id, '') FROM llm_configs")
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var configs []LLMConfig
		for rows.Next() {
			var c LLMConfig
			if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.BaseURL, &c.APIKey, &c.Model, &c.IsActive, &c.StartDate, &c.EndDate, &c.GroupID); err != nil {
				log.Printf("[handleLLMConfig] scan error: %v", err)
				continue
			}
			configs = append(configs, c)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(configs)
		return
	}
	if r.Method == "POST" {
		var c LLMConfig
		json.NewDecoder(r.Body).Decode(&c)
		if c.ID == "" {
			c.ID = generateShortID()
		}
		if c.IsActive {
			db.Exec("UPDATE llm_configs SET is_active=0 WHERE group_id=?", c.GroupID)
		}
		db.Exec(`INSERT OR REPLACE INTO llm_configs (id, name, type, base_url, api_key, model, is_active, start_date, end_date, group_id) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, c.ID, c.Name, c.Type, c.BaseURL, c.APIKey, c.Model, c.IsActive, c.StartDate, c.EndDate, c.GroupID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	if r.Method == "DELETE" {
		var req struct{ ID string `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		db.Exec("DELETE FROM llm_configs WHERE id=?", req.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

func handleSearchConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, name, type, api_key, is_active, start_date, end_date, COALESCE(group_id, '') FROM search_configs")
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var configs []SearchConfig
		for rows.Next() {
			var c SearchConfig
			if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.APIKey, &c.IsActive, &c.StartDate, &c.EndDate, &c.GroupID); err != nil {
				log.Printf("[handleSearchConfig] scan error: %v", err)
				continue
			}
			configs = append(configs, c)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(configs)
		return
	}
	if r.Method == "POST" {
		var c SearchConfig
		json.NewDecoder(r.Body).Decode(&c)
		if c.ID == "" {
			c.ID = generateShortID()
		}
		if c.IsActive {
			db.Exec("UPDATE search_configs SET is_active=0 WHERE group_id=?", c.GroupID)
		}
		db.Exec(`INSERT OR REPLACE INTO search_configs (id, name, type, api_key, is_active, start_date, end_date, group_id) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, c.ID, c.Name, c.Type, c.APIKey, c.IsActive, c.StartDate, c.EndDate, c.GroupID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	if r.Method == "DELETE" {
		var req struct{ ID string `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		db.Exec("DELETE FROM search_configs WHERE id=?", req.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

func handleLLMGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, name, description FROM llm_groups ORDER BY name")
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var groups []LLMGroup
		for rows.Next() {
			var g LLMGroup
			if err := rows.Scan(&g.ID, &g.Name, &g.Description); err != nil {
				log.Printf("[handleLLMGroups] scan error: %v", err)
				continue
			}
			groups = append(groups, g)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
		return
	}
	if r.Method == "POST" {
		var g LLMGroup
		json.NewDecoder(r.Body).Decode(&g)
		if g.ID == "" {
			g.ID = generateShortID()
		}
		db.Exec("INSERT OR REPLACE INTO llm_groups (id, name, description) VALUES (?, ?, ?)", g.ID, g.Name, g.Description)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": g.ID})
		return
	}
	if r.Method == "DELETE" {
		var req struct{ ID string `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		// Clear group_id from configs that use this group
		db.Exec("UPDATE llm_configs SET group_id='' WHERE group_id=?", req.ID)
		db.Exec("UPDATE licenses SET llm_group_id='' WHERE llm_group_id=?", req.ID)
		db.Exec("DELETE FROM llm_groups WHERE id=?", req.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

func handleSearchGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, name, description FROM search_groups ORDER BY name")
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var groups []SearchGroup
		for rows.Next() {
			var g SearchGroup
			if err := rows.Scan(&g.ID, &g.Name, &g.Description); err != nil {
				log.Printf("[handleSearchGroups] scan error: %v", err)
				continue
			}
			groups = append(groups, g)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
		return
	}
	if r.Method == "POST" {
		var g SearchGroup
		json.NewDecoder(r.Body).Decode(&g)
		if g.ID == "" {
			g.ID = generateShortID()
		}
		db.Exec("INSERT OR REPLACE INTO search_groups (id, name, description) VALUES (?, ?, ?)", g.ID, g.Name, g.Description)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": g.ID})
		return
	}
	if r.Method == "DELETE" {
		var req struct{ ID string `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		// Clear group_id from configs that use this group
		db.Exec("UPDATE search_configs SET group_id='' WHERE group_id=?", req.ID)
		db.Exec("UPDATE licenses SET search_group_id='' WHERE search_group_id=?", req.ID)
		db.Exec("DELETE FROM search_groups WHERE id=?", req.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}


func handleLicenseGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, name, description, COALESCE(trust_level, 'low'), COALESCE(llm_group_id, ''), COALESCE(search_group_id, '') FROM license_groups ORDER BY name")
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var groups []LicenseGroup
		for rows.Next() {
			var g LicenseGroup
			if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.TrustLevel, &g.LLMGroupID, &g.SearchGroupID); err != nil {
				log.Printf("[handleLicenseGroups] scan error: %v", err)
				continue
			}
			groups = append(groups, g)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
		return
	}
	if r.Method == "POST" {
		var g LicenseGroup
		json.NewDecoder(r.Body).Decode(&g)
		if g.ID == "" {
			g.ID = generateShortID()
		}
		// User-created groups are always low-trust (trial)
		// Only built-in official groups (created by getOrCreateProductOfficialGroup) can be high-trust
		g.TrustLevel = "low"
		db.Exec("INSERT OR REPLACE INTO license_groups (id, name, description, trust_level) VALUES (?, ?, ?, ?)", g.ID, g.Name, g.Description, g.TrustLevel)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": g.ID})
		return
	}
	if r.Method == "DELETE" {
		var req struct{ ID string `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		
		// Prevent deletion of built-in official groups
		if strings.HasPrefix(req.ID, "official_") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, 
				"error": "内置正式授权组不能删除",
			})
			return
		}
		
		// Check if this group is being used by any licenses
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM licenses WHERE license_group_id=?", req.ID).Scan(&count)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "查询失败"})
			return
		}
		
		if count > 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, 
				"error": fmt.Sprintf("此分组中还有 %d 个序列号，无法删除", count),
			})
			return
		}
		
		// No licenses using this group, safe to delete
		db.Exec("DELETE FROM license_groups WHERE id=?", req.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

// handleLicenseGroupConfig handles configuration of official license groups (LLM/Search groups)
func handleLicenseGroupConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ID            string `json:"id"`
		LLMGroupID    string `json:"llm_group_id"`
		SearchGroupID string `json:"search_group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}
	
	// Only allow configuring official groups
	if !strings.HasPrefix(req.ID, "official_") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "只能配置内置正式授权组"})
		return
	}
	
	// Update the group's LLM and Search group settings
	_, err := db.Exec("UPDATE license_groups SET llm_group_id=?, search_group_id=? WHERE id=?", 
		req.LLMGroupID, req.SearchGroupID, req.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	log.Printf("[LICENSE-GROUP] Updated official group %s: LLM=%s, Search=%s", req.ID, req.LLMGroupID, req.SearchGroupID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleProductTypes manages product types for license categorization
func handleProductTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, name, description FROM product_types ORDER BY id")
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var products []ProductType
		for rows.Next() {
			var p ProductType
			if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
				log.Printf("[handleProductTypes] scan error: %v", err)
				continue
			}
			products = append(products, p)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		if products == nil {
			products = []ProductType{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
		return
	}
	if r.Method == "POST" {
		var p ProductType
		json.NewDecoder(r.Body).Decode(&p)
		
		if p.ID == 0 {
			// Insert new product type
			result, err := db.Exec("INSERT INTO product_types (name, description) VALUES (?, ?)", p.Name, p.Description)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			id, _ := result.LastInsertId()
			p.ID = int(id)
		} else {
			// Update existing product type
			_, err := db.Exec("UPDATE product_types SET name=?, description=? WHERE id=?", p.Name, p.Description, p.ID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": p.ID})
		return
	}
	if r.Method == "DELETE" {
		var req struct{ ID int `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		
		// Check if this product type is being used by any licenses
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM licenses WHERE product_id=?", req.ID).Scan(&count)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "查询失败"})
			return
		}
		
		if count > 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, 
				"error": fmt.Sprintf("此产品类型下还有 %d 个序列号，无法删除", count),
			})
			return
		}
		
		// No licenses using this product type, safe to delete
		db.Exec("DELETE FROM product_types WHERE id=?", req.ID)
		// Also delete extra info for this product
		db.Exec("DELETE FROM product_extra_info WHERE product_id=?", req.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

// getProductExtraInfo retrieves extra info for a product as a map
func getProductExtraInfo(productID int) map[string]interface{} {
	rows, err := db.Query("SELECT key, value, value_type FROM product_extra_info WHERE product_id = ?", productID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	
	var result map[string]interface{}
	for rows.Next() {
		var key, value, valueType string
		rows.Scan(&key, &value, &valueType)
		if result == nil {
			result = make(map[string]interface{})
		}
		if valueType == "number" {
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				// Check if it's an integer
				if f == float64(int64(f)) {
					result[key] = int64(f)
				} else {
					result[key] = f
				}
			} else {
				result[key] = value
			}
		} else {
			result[key] = value
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: rows iteration error: %v", err)
	}
	return result
}

// handleProductExtraInfo manages product extra info key-value pairs
func handleProductExtraInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		productIDStr := r.URL.Query().Get("product_id")
		productID := 0
		fmt.Sscanf(productIDStr, "%d", &productID)
		
		rows, err := db.Query("SELECT id, product_id, key, value, value_type FROM product_extra_info WHERE product_id = ? ORDER BY key", productID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		defer rows.Close()
		
		var items []map[string]interface{}
		for rows.Next() {
			var id, pid int
			var key, value, valueType string
			if err := rows.Scan(&id, &pid, &key, &value, &valueType); err != nil {
				log.Printf("[handleProductExtraInfo] scan error: %v", err)
				continue
			}
			items = append(items, map[string]interface{}{
				"id": id, "product_id": pid, "key": key, "value": value, "value_type": valueType,
			})
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
		return
	}
	
	if r.Method == "POST" {
		var req struct {
			ID        int    `json:"id"`
			ProductID int    `json:"product_id"`
			Key       string `json:"key"`
			Value     string `json:"value"`
			ValueType string `json:"value_type"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		
		if req.Key == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Key 不能为空"})
			return
		}
		if req.ValueType == "" {
			req.ValueType = "string"
		}
		
		if req.ID == 0 {
			// Insert new
			_, err := db.Exec("INSERT OR REPLACE INTO product_extra_info (product_id, key, value, value_type) VALUES (?, ?, ?, ?)",
				req.ProductID, req.Key, req.Value, req.ValueType)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
		} else {
			// Update existing
			_, err := db.Exec("UPDATE product_extra_info SET key=?, value=?, value_type=? WHERE id=?",
				req.Key, req.Value, req.ValueType, req.ID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	
	if r.Method == "DELETE" {
		var req struct{ ID int `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		db.Exec("DELETE FROM product_extra_info WHERE id=?", req.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if !checkAdminPassword(req.OldPassword, getSetting("admin_password")) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "旧密码错误"})
		return
	}
	if len(req.NewPassword) < 6 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "新密码长度不能少于6位"})
		return
	}
	if len(req.NewPassword) > 72 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "新密码长度不能超过72位"})
		return
	}
	setSetting("admin_password", hashAdminPassword(req.NewPassword))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleChangeUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		username := getSetting("admin_username")
		if username == "" {
			username = "admin"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": username})
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "用户名不能为空"})
		return
	}
	setSetting("admin_username", req.Username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleChangePorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ManagePort int `json:"manage_port"`
		AuthPort   int `json:"auth_port"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	setSetting("manage_port", fmt.Sprintf("%d", req.ManagePort))
	setSetting("auth_port", fmt.Sprintf("%d", req.AuthPort))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "端口配置已保存，请重启服务生效"})
}

func handleSSLConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"use_ssl":  useSSL,
			"ssl_cert": sslCert,
			"ssl_key":  sslKey,
		})
		return
	}
	if r.Method == "POST" {
		var req struct {
			UseSSL  bool   `json:"use_ssl"`
			SSLCert string `json:"ssl_cert"`
			SSLKey  string `json:"ssl_key"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		
		// Validate certificate files if SSL is enabled
		if req.UseSSL {
			if req.SSLCert == "" || req.SSLKey == "" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "请指定证书和密钥文件路径"})
				return
			}
			// Check if files exist
			if _, err := os.Stat(req.SSLCert); os.IsNotExist(err) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "证书文件不存在: " + req.SSLCert})
				return
			}
			if _, err := os.Stat(req.SSLKey); os.IsNotExist(err) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "密钥文件不存在: " + req.SSLKey})
				return
			}
		}
		
		if req.UseSSL {
			setSetting("use_ssl", "true")
		} else {
			setSetting("use_ssl", "false")
		}
		setSetting("ssl_cert", req.SSLCert)
		setSetting("ssl_key", req.SSLKey)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "SSL配置已保存，请重启服务生效"})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleSMTPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		config := getSMTPConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
		return
	}
	if r.Method == "POST" {
		var config SMTPConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求数据"})
			return
		}
		
		saveSMTPConfig(config)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "SMTP配置已保存"})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleSMTPTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "请提供测试邮箱地址"})
		return
	}
	
	config := getSMTPConfig()
	if !config.Enabled {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "SMTP未启用"})
		return
	}
	
	// Send test email
	subject := "Vantagics SMTP 测试邮件"
	htmlBody := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: sans-serif; padding: 20px;">
    <h2>🎉 SMTP 配置测试成功！</h2>
    <p>如果您收到这封邮件，说明 SMTP 配置正确。</p>
    <p style="color: #666; font-size: 12px;">此邮件由 Vantagics 授权服务器发送。</p>
</body>
</html>
`
	
	if err := sendEmail(req.Email, subject, htmlBody); err != nil {
		log.Printf("[SMTP-TEST] Failed to send test email to %s: %v", req.Email, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": fmt.Sprintf("发送失败: %v", err)})
		return
	}
	
	log.Printf("[SMTP-TEST] Test email sent successfully to %s", req.Email)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "测试邮件已发送，请检查收件箱"})
}

func handleRequestLimits(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		dailyRequestLimit := getSetting("daily_request_limit")
		dailyEmailLimit := getSetting("daily_email_limit")
		
		reqLimit := 5
		emailLimit := 5
		if dailyRequestLimit != "" {
			fmt.Sscanf(dailyRequestLimit, "%d", &reqLimit)
		}
		if dailyEmailLimit != "" {
			fmt.Sscanf(dailyEmailLimit, "%d", &emailLimit)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"daily_request_limit": reqLimit,
			"daily_email_limit":   emailLimit,
		})
		return
	}
	if r.Method == "POST" {
		var req struct {
			DailyRequestLimit int `json:"daily_request_limit"`
			DailyEmailLimit   int `json:"daily_email_limit"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		
		if req.DailyRequestLimit < 1 {
			req.DailyRequestLimit = 5
		}
		if req.DailyEmailLimit < 1 {
			req.DailyEmailLimit = 5
		}
		
		setSetting("daily_request_limit", fmt.Sprintf("%d", req.DailyRequestLimit))
		setSetting("daily_email_limit", fmt.Sprintf("%d", req.DailyEmailLimit))
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		return
	}
}

func handleEmailRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	search := query.Get("search")
	productFilter := query.Get("product_id") // -1 or empty = all, >= 0 = specific product
	licenseGroupFilter := query.Get("license_group") // empty = all, "none" = no group, other = specific group
	page, pageSize := 1, 20
	fmt.Sscanf(query.Get("page"), "%d", &page)
	fmt.Sscanf(query.Get("pageSize"), "%d", &pageSize)
	if page < 1 { page = 1 }
	if pageSize < 1 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	
	// Build dynamic query
	baseQuery := `SELECT e.id, e.email, e.sn, e.ip, e.created_at, COALESCE(e.product_id, 0) 
		FROM email_records e`
	countQuery := `SELECT COUNT(*) FROM email_records e`
	
	var conditions []string
	var args []interface{}
	
	// Join with licenses table if filtering by license_group
	if licenseGroupFilter != "" {
		baseQuery = `SELECT e.id, e.email, e.sn, e.ip, e.created_at, COALESCE(e.product_id, 0) 
			FROM email_records e LEFT JOIN licenses l ON e.sn = l.sn`
		countQuery = `SELECT COUNT(*) FROM email_records e LEFT JOIN licenses l ON e.sn = l.sn`
		
		if licenseGroupFilter == "none" {
			conditions = append(conditions, "(l.license_group_id IS NULL OR l.license_group_id = '')")
		} else {
			conditions = append(conditions, "l.license_group_id = ?")
			args = append(args, licenseGroupFilter)
		}
	}
	
	// Product filter
	hasProductFilter := productFilter != "" && productFilter != "-1"
	if hasProductFilter {
		productID := 0
		fmt.Sscanf(productFilter, "%d", &productID)
		conditions = append(conditions, "COALESCE(e.product_id, 0) = ?")
		args = append(args, productID)
	}
	
	// Search filter
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		conditions = append(conditions, "(LOWER(e.email) LIKE ? OR LOWER(e.sn) LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}
	
	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Get total count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	db.QueryRow(countQuery+whereClause, countArgs...).Scan(&total)
	
	// Get records with pagination
	finalQuery := baseQuery + whereClause + " ORDER BY e.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	
	rows, err := db.Query(finalQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var records []EmailRecord
	for rows.Next() {
		var r EmailRecord
		if err := rows.Scan(&r.ID, &r.Email, &r.SN, &r.IP, &r.CreatedAt, &r.ProductID); err != nil {
			log.Printf("[handleEmailRecords] scan error: %v", err)
			continue
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
		return
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 { totalPages = 1 }
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records": records, "total": total, "page": page, "pageSize": pageSize, "totalPages": totalPages,
	})
}

// handleClearEmailRecords clears all email records for a given email address
func handleClearEmailRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "请输入有效的邮箱地址"})
		return
	}

	// Count records first
	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_records WHERE email=?", email).Scan(&count)
	if count == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": fmt.Sprintf("未找到邮箱 %s 的申请记录", email)})
		return
	}

	// Delete all email records for this email
	result, err := db.Exec("DELETE FROM email_records WHERE email=?", email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	deleted, _ := result.RowsAffected()
	log.Printf("[CLEAR-EMAIL] Cleared %d email records for %s", deleted, email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("已清除邮箱 %s 的 %d 条申请记录", email, deleted),
		"deleted": deleted,
	})
}

// handleClearIPRecords clears all SN request rate limit records for a given IP address
func handleClearIPRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}

	ip := strings.TrimSpace(req.IP)
	if ip == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "请输入有效的IP地址"})
		return
	}

	// Count records first
	var count int
	db.QueryRow("SELECT COUNT(*) FROM request_limits WHERE ip=?", ip).Scan(&count)
	if count == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": fmt.Sprintf("未找到IP %s 的请求记录", ip)})
		return
	}

	// Delete all request limit records for this IP
	result, err := db.Exec("DELETE FROM request_limits WHERE ip=?", ip)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	deleted, _ := result.RowsAffected()
	log.Printf("[CLEAR-IP] Cleared %d request limit records for IP %s", deleted, ip)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("已清除IP %s 的 %d 条请求记录", ip, deleted),
		"deleted": deleted,
	})
}

// handleUpdateEmailRecord handles updating the license associated with an email record
func handleUpdateEmailRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		SN            string `json:"sn"`
		ExpiresAt     string `json:"expires_at"`
		LLMGroupID    string `json:"llm_group_id"`
		SearchGroupID string `json:"search_group_id"`
		IsActive      bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}
	
	// Check if SN exists
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM licenses WHERE sn=?", req.SN).Scan(&count); err != nil || count == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "序列号不存在"})
		return
	}
	
	// Parse expires_at
	expiresAt, err := time.Parse("2006-01-02", req.ExpiresAt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的日期格式"})
		return
	}
	
	// Update license
	_, err = db.Exec(`UPDATE licenses SET 
		expires_at=?, 
		llm_group_id=?, 
		search_group_id=?, 
		is_active=? 
		WHERE sn=?`,
		expiresAt, req.LLMGroupID, req.SearchGroupID, req.IsActive, req.SN)
	
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// getOrCreateProductOfficialGroup gets or creates the built-in high-trust official group for a product
func getOrCreateProductOfficialGroup(productID int) string {
	productName := getProductName(productID)
	groupID := fmt.Sprintf("official_%d", productID)
	groupName := fmt.Sprintf("%s 正式授权", productName)
	
	// Use INSERT OR IGNORE to avoid race conditions
	_, err := db.Exec("INSERT OR IGNORE INTO license_groups (id, name, description, trust_level) VALUES (?, ?, ?, 'high')",
		groupID, groupName, fmt.Sprintf("%s 产品内置高可信正式授权组", productName))
	if err != nil {
		log.Printf("[LICENSE-GROUP] Failed to create official group for product %d: %v", productID, err)
		return ""
	}
	
	return groupID
}

// getOrCreateProductTrialGroup gets or creates the built-in low-trust trial group for a product
func getOrCreateProductTrialGroup(productID int) string {
	productName := getProductName(productID)
	groupID := fmt.Sprintf("trial_%d", productID)
	groupName := fmt.Sprintf("%s 试用授权", productName)
	
	// Use INSERT OR IGNORE to avoid race conditions
	_, err := db.Exec("INSERT OR IGNORE INTO license_groups (id, name, description, trust_level) VALUES (?, ?, ?, 'low')",
		groupID, groupName, fmt.Sprintf("%s 产品内置低可信试用授权组", productName))
	if err != nil {
		log.Printf("[LICENSE-GROUP] Failed to create trial group for product %d: %v", productID, err)
		return ""
	}
	
	return groupID
}

func getOrCreateProductFreeGroup(productID int) string {
	productName := getProductName(productID)
	groupID := fmt.Sprintf("free_%d", productID)
	groupName := fmt.Sprintf("%s 永久免费授权", productName)

	// Use INSERT OR IGNORE to avoid race conditions
	_, err := db.Exec("INSERT OR IGNORE INTO license_groups (id, name, description, trust_level) VALUES (?, ?, ?, 'permanent_free')",
		groupID, groupName, fmt.Sprintf("%s 产品内置永久免费授权组", productName))
	if err != nil {
		log.Printf("[LICENSE-GROUP] Failed to create free group for product %d: %v", productID, err)
		return ""
	}

	return groupID
}

func getOrCreateProductOpenSourceGroup(productID int) string {
	productName := getProductName(productID)
	groupID := fmt.Sprintf("oss_%d", productID)
	groupName := fmt.Sprintf("%s 开源授权", productName)

	// Use INSERT OR IGNORE to avoid race conditions
	// trust_level = "open_source": similar to permanent_free but for open source users
	_, err := db.Exec("INSERT OR IGNORE INTO license_groups (id, name, description, trust_level) VALUES (?, ?, ?, 'open_source')",
		groupID, groupName, fmt.Sprintf("%s 产品开源授权组", productName))
	if err != nil {
		log.Printf("[LICENSE-GROUP] Failed to create open source group for product %d: %v", productID, err)
		return ""
	}

	return groupID
}


// handleManualBind handles manual SN binding from admin panel (creates new high-trust official license)
func handleManualBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Email         string `json:"email"`
		ProductID     int    `json:"product_id"`
		Days          int    `json:"days"`
		LLMGroupID    string `json:"llm_group_id"`
		SearchGroupID string `json:"search_group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "无效的请求格式"})
		return
	}
	
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "无效的邮箱地址"})
		return
	}
	
	if req.Days <= 0 {
		req.Days = 365
	}
	
	// Check if email already has a license for this product
	var existingSN string
	err := db.QueryRow("SELECT sn FROM email_records WHERE email = ? AND product_id = ? AND sn_type = 'commercial'", email, req.ProductID).Scan(&existingSN)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, 
			"message": fmt.Sprintf("该邮箱已绑定商业序列号 %s，请先删除旧记录", existingSN),
		})
		return
	}
	
	// Get or create the official high-trust group for this product (manual bind = official)
	licenseGroupID := getOrCreateProductOfficialGroup(req.ProductID)
	if licenseGroupID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "创建授权组失败"})
		return
	}
	
	// Generate new SN
	sn := generateSN()
	now := time.Now()
	expiresAt := now.AddDate(0, 0, req.Days)
	
	// Create the license with high-trust group, unlimited analysis
	_, err = db.Exec(`INSERT INTO licenses (sn, created_at, expires_at, valid_days, description, is_active, 
		daily_analysis, license_group_id, llm_group_id, search_group_id, product_id) 
		VALUES (?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?)`,
		sn, now, expiresAt, req.Days, fmt.Sprintf("手工绑定: %s", email), 
		licenseGroupID, req.LLMGroupID, req.SearchGroupID, req.ProductID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "创建序列号失败: " + err.Error()})
		return
	}
	
	// Create email record
	_, err = db.Exec("INSERT INTO email_records (email, sn, ip, created_at, product_id, sn_type) VALUES (?, ?, ?, ?, ?, 'commercial')",
		email, sn, "admin-manual-bind", now, req.ProductID)
	if err != nil {
		// Rollback: delete the license
		db.Exec("DELETE FROM licenses WHERE sn = ?", sn)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "创建邮箱记录失败: " + err.Error()})
		return
	}
	
	log.Printf("[MANUAL-BIND] Created high-trust license %s for email %s (Product: %d, Days: %d, Group: %s)", 
		sn, email, req.ProductID, req.Days, licenseGroupID)
	
	// Send email notification
	go func() {
		if err := sendSNEmail(email, sn, expiresAt, req.ProductID); err != nil {
			log.Printf("[EMAIL] Failed to send SN email to %s: %v", email, err)
		} else {
			log.Printf("[EMAIL] SN email sent successfully to %s", email)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"sn":      sn,
		"message": fmt.Sprintf("高可信正式授权，有效期 %d 天", req.Days),
	})
}

// handleManualRequest handles manual SN request from admin panel
func handleManualRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Email     string `json:"email"`
		ProductID int    `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "无效的请求格式"})
		return
	}
	
	email := strings.ToLower(strings.TrimSpace(req.Email))
	productID := req.ProductID
	
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "请输入有效的邮箱地址"})
		return
	}
	
	// Check email whitelist/blacklist and get group bindings (same as normal request)
	allowed, _, reason, llmGroupID, searchGroupID := isEmailAllowedWithGroups(email)
	if !allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": reason})
		return
	}
	
	// Check if email already has SN for this product
	var existingSN string
	if err := db.QueryRow(`SELECT e.sn FROM email_records e 
		JOIN licenses l ON e.sn = l.sn 
		WHERE e.email=? AND e.product_id=? AND e.sn_type='commercial'`, email, productID).Scan(&existingSN); err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "该邮箱已申请过此产品的商业序列号: " + existingSN})
		return
	}
	
	// Check if there's an orphaned commercial record (SN was deleted) for this email+product
	var orphanedSN string
	if err := db.QueryRow(`SELECT sn FROM email_records WHERE email=? AND product_id=? AND sn_type='commercial'`, email, productID).Scan(&orphanedSN); err == nil {
		db.Exec("DELETE FROM email_records WHERE email=? AND product_id=? AND sn_type='commercial'", email, productID)
		log.Printf("[MANUAL-REQUEST] Old SN %s for email %s product %d was deleted, generating new one", orphanedSN, email, productID)
	}
	
	// Find an available SN (same logic as handleRequestSN but without rate limiting)
	now := time.Now()
	var sn string
	var validDays int
	var hasExpiresAt bool
	
	query, args := buildAvailableSNQuery(productID, llmGroupID, searchGroupID, now, false, false)
	
	var nullableExpiresAt sql.NullTime
	err := db.QueryRow(query, args...).Scan(&sn, &validDays, &nullableExpiresAt)
	if err != nil {
		log.Printf("[MANUAL-REQUEST] No available SN found for email %s (Product: %d, LLM Group: %s, Search Group: %s): %v", email, productID, llmGroupID, searchGroupID, err)
		var msg string
		if productID > 0 || llmGroupID != "" || searchGroupID != "" {
			msg = fmt.Sprintf("暂无匹配的可用序列号（产品ID: %d, LLM分组: %s, 搜索分组: %s）", productID, llmGroupID, searchGroupID)
		} else {
			msg = "暂无可用序列号"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": msg})
		return
	}
	
	// Determine expiry date:
	// - If expires_at is NULL (new SN), calculate from now + valid_days
	// - If expires_at is set (old SN), use existing expiry date
	var expiresAt time.Time
	if nullableExpiresAt.Valid {
		expiresAt = nullableExpiresAt.Time
		hasExpiresAt = true
	} else {
		expiresAt = now.AddDate(0, 0, validDays)
		hasExpiresAt = false
	}
	
	// Bind the SN to the email
	db.Exec("INSERT INTO email_records (email, sn, ip, created_at, product_id, sn_type) VALUES (?, ?, ?, ?, ?, 'commercial')", email, sn, "manual", now, productID)
	
	// Update the license: set expires_at (only if not already set) and description
	if hasExpiresAt {
		// Old SN with existing expiry, just update description
		db.Exec("UPDATE licenses SET description = ? WHERE sn = ?", fmt.Sprintf("手工申请: %s", email), sn)
	} else {
		// New SN, set expiry date
		db.Exec("UPDATE licenses SET expires_at = ?, description = ? WHERE sn = ?", expiresAt, fmt.Sprintf("手工申请: %s", email), sn)
	}
	
	// Calculate days left
	daysLeft := int(expiresAt.Sub(now).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}
	
	log.Printf("[MANUAL-REQUEST] SN allocated for email %s: %s (Product: %d, LLM Group: %s, Search Group: %s, HasExpiry: %v)", email, sn, productID, llmGroupID, searchGroupID, hasExpiresAt)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, 
		"sn": sn, 
		"message": fmt.Sprintf("有效期 %d 天", daysLeft),
	})
}

// ============ API Key Handlers ============

// generateAPIKey generates a sk- prefixed 64 character API key
func generateAPIKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 61) // 64 - 3 (for "sk-")
	if _, err := rand.Read(b); err != nil {
		log.Printf("Warning: crypto/rand failed: %v, using fallback", err)
		for i := range b {
			b[i] = charset[mrand.Intn(len(charset))]
		}
	} else {
		for i := range b {
			b[i] = charset[int(b[i])%len(charset)]
		}
	}
	return "sk-" + string(b)
}

func handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query(`SELECT id, api_key, product_id, organization, contact_name, description, 
			is_active, usage_count, created_at, expires_at FROM api_keys ORDER BY created_at DESC`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		
		var keys []APIKey
		for rows.Next() {
			var k APIKey
			var expiresAt sql.NullTime
			rows.Scan(&k.ID, &k.APIKey, &k.ProductID, &k.Organization, &k.ContactName, 
				&k.Description, &k.IsActive, &k.UsageCount, &k.CreatedAt, &expiresAt)
			if expiresAt.Valid {
				k.ExpiresAt = &expiresAt.Time
			}
			keys = append(keys, k)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)
		return
	}
	
	if r.Method == "POST" {
		var req struct {
			ID           string `json:"id"`
			ProductID    int    `json:"product_id"`
			Organization string `json:"organization"`
			ContactName  string `json:"contact_name"`
			Description  string `json:"description"`
			ExpiresAt    string `json:"expires_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
			return
		}
		
		now := time.Now()
		var expiresAt *time.Time
		if req.ExpiresAt != "" {
			t, err := time.Parse("2006-01-02", req.ExpiresAt)
			if err == nil {
				// Set to end of day
				t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				expiresAt = &t
			}
		}
		
		if req.ID == "" {
			// Create new API key
			apiKey := generateAPIKey()
			id := generateShortID()
			
			_, err := db.Exec(`INSERT INTO api_keys (id, api_key, product_id, organization, contact_name, 
				description, is_active, usage_count, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, 1, 0, ?, ?)`,
				id, apiKey, req.ProductID, req.Organization, req.ContactName, req.Description, now, expiresAt)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			
			log.Printf("[API-KEY] Created new API key %s for product %d (org: %s)", apiKey[:20]+"...", req.ProductID, req.Organization)
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "api_key": apiKey})
		} else {
			// Update existing API key (only organization, contact_name, description, expires_at)
			_, err := db.Exec(`UPDATE api_keys SET organization=?, contact_name=?, description=?, expires_at=? WHERE id=?`,
				req.Organization, req.ContactName, req.Description, expiresAt, req.ID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		}
		return
	}
	
	if r.Method == "DELETE" {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
			return
		}
		
		// Check if API key has been used
		var usageCount int
		db.QueryRow("SELECT usage_count FROM api_keys WHERE id=?", req.ID).Scan(&usageCount)
		if usageCount > 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, 
				"error": fmt.Sprintf("此 API Key 已使用 %d 次，不能删除，只能禁用", usageCount),
			})
			return
		}
		
		db.Exec("DELETE FROM api_keys WHERE id=?", req.ID)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		return
	}
}

func handleToggleAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}
	
	db.Exec("UPDATE api_keys SET is_active = NOT is_active WHERE id=?", req.ID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleAPIKeyBindings returns all email records created by a specific API key
func handleAPIKeyBindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	keyID := r.URL.Query().Get("id")
	if keyID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "缺少 API Key ID"})
		return
	}
	
	rows, err := db.Query(`SELECT id, email, sn, ip, created_at, product_id FROM email_records 
		WHERE api_key_id = ? ORDER BY created_at DESC`, keyID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	defer rows.Close()
	
	var records []EmailRecord
	for rows.Next() {
		var r EmailRecord
		rows.Scan(&r.ID, &r.Email, &r.SN, &r.IP, &r.CreatedAt, &r.ProductID)
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: rows iteration error: %v", err)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "records": records})
}

// handleClearAPIKeyBindings deletes all email records and licenses created by a specific API key
func handleClearAPIKeyBindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求格式"})
		return
	}
	
	if req.ID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "缺少 API Key ID"})
		return
	}
	
	// Get all SNs created by this API key
	rows, err := db.Query("SELECT sn FROM email_records WHERE api_key_id = ?", req.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	
	var sns []string
	for rows.Next() {
		var sn string
		rows.Scan(&sn)
		sns = append(sns, sn)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: rows iteration error: %v", err)
	}
	rows.Close()
	
	deletedCount := len(sns)
	
	// Delete all licenses created by this API key
	for _, sn := range sns {
		db.Exec("DELETE FROM licenses WHERE sn = ?", sn)
	}
	
	// Delete all email records created by this API key
	db.Exec("DELETE FROM email_records WHERE api_key_id = ?", req.ID)
	
	// Reset usage count for this API key
	db.Exec("UPDATE api_keys SET usage_count = 0 WHERE id = ?", req.ID)
	
	log.Printf("[API-KEY] Cleared %d bindings for API key %s", deletedCount, req.ID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "deleted": deletedCount})
}

// handleBindLicenseAPI handles the public API for binding licenses via API key
func handleBindLicenseAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		APIKey string `json:"api_key"`
		Email  string `json:"email"`
		Days   int    `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, 
			"code": "INVALID_REQUEST",
			"message": "无效的请求格式",
		})
		return
	}
	
	// Validate API key
	var keyID string
	var productID int
	var isActive bool
	var expiresAt sql.NullTime
	err := db.QueryRow(`SELECT id, product_id, is_active, expires_at FROM api_keys WHERE api_key=?`, req.APIKey).
		Scan(&keyID, &productID, &isActive, &expiresAt)
	if err != nil {
		log.Printf("[BIND-API] Invalid API key attempted: %s", req.APIKey[:min(20, len(req.APIKey))]+"...")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "INVALID_API_KEY",
			"message": "无效的 API Key",
		})
		return
	}
	
	if !isActive {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "API_KEY_DISABLED",
			"message": "API Key 已被禁用",
		})
		return
	}
	
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "API_KEY_EXPIRED",
			"message": "API Key 已过期",
		})
		return
	}
	
	// Validate email
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "INVALID_EMAIL",
			"message": "无效的邮箱地址",
		})
		return
	}
	
	// Validate days
	if req.Days <= 0 {
		req.Days = 365
	}
	
	// Check if email already has a license for this product
	var existingSN string
	err = db.QueryRow("SELECT sn FROM email_records WHERE email=? AND product_id=? AND sn_type='commercial'", email, productID).Scan(&existingSN)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "EMAIL_ALREADY_BOUND",
			"message": fmt.Sprintf("该邮箱已绑定商业序列号 %s", existingSN),
		})
		return
	}
	
	// Get or create the official high-trust group for this product
	licenseGroupID := getOrCreateProductOfficialGroup(productID)
	if licenseGroupID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "INTERNAL_ERROR",
			"message": "创建授权组失败",
		})
		return
	}
	
	// Get LLM and Search group from the official group
	var llmGroupID, searchGroupID string
	db.QueryRow("SELECT COALESCE(llm_group_id, ''), COALESCE(search_group_id, '') FROM license_groups WHERE id=?", 
		licenseGroupID).Scan(&llmGroupID, &searchGroupID)
	
	// Generate new SN
	sn := generateSN()
	now := time.Now()
	expiresAtTime := now.AddDate(0, 0, req.Days)
	
	// Use transaction to ensure license creation and email binding are atomic
	tx, txErr := db.Begin()
	if txErr != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "INTERNAL_ERROR",
			"message": "系统错误，请稍后重试",
		})
		return
	}
	
	// Create the license with high-trust group, unlimited analysis
	_, err = tx.Exec(`INSERT INTO licenses (sn, created_at, expires_at, valid_days, description, is_active, 
		daily_analysis, license_group_id, llm_group_id, search_group_id, product_id) 
		VALUES (?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?)`,
		sn, now, expiresAtTime, req.Days, fmt.Sprintf("API绑定: %s", email), 
		licenseGroupID, llmGroupID, searchGroupID, productID)
	if err != nil {
		tx.Rollback()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "INTERNAL_ERROR",
			"message": "创建序列号失败: " + err.Error(),
		})
		return
	}
	
	// Create email record with api_key_id for tracking
	_, err = tx.Exec("INSERT INTO email_records (email, sn, ip, created_at, product_id, api_key_id, sn_type) VALUES (?, ?, ?, ?, ?, ?, 'commercial')",
		email, sn, "api-bind", now, productID, keyID)
	if err != nil {
		tx.Rollback()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "INTERNAL_ERROR",
			"message": "创建邮箱记录失败: " + err.Error(),
		})
		return
	}
	
	// Increment API key usage count
	_, _ = tx.Exec("UPDATE api_keys SET usage_count = usage_count + 1 WHERE id=?", keyID)
	
	if txErr = tx.Commit(); txErr != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"code": "INTERNAL_ERROR",
			"message": "系统错误，请稍后重试",
		})
		return
	}
	
	log.Printf("[BIND-API] Created license %s for email %s via API key (Product: %d, Days: %d)", 
		sn, email, productID, req.Days)
	
	// Send email notification
	go func() {
		if err := sendSNEmail(email, sn, expiresAtTime, productID); err != nil {
			log.Printf("[EMAIL] Failed to send SN email to %s: %v", email, err)
		} else {
			log.Printf("[EMAIL] SN email sent successfully to %s", email)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"code": "SUCCESS",
		"sn": sn,
		"expires_at": expiresAtTime.Format("2006-01-02"),
		"message": fmt.Sprintf("正式授权创建成功，有效期 %d 天", req.Days),
	})
}

// ============ Email Filter Handlers ============

func handleEmailFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		whitelistEnabled := getSetting("whitelist_enabled") == "true"
		blacklistEnabled := getSetting("blacklist_enabled")
		if blacklistEnabled == "" {
			blacklistEnabled = "true" // Default to blacklist mode
		}
		conditionsEnabled := getSetting("conditions_enabled") == "true"
		dailyRequestLimit := getSetting("daily_request_limit")
		if dailyRequestLimit == "" {
			dailyRequestLimit = "5"
		}
		dailyEmailLimit := getSetting("daily_email_limit")
		if dailyEmailLimit == "" {
			dailyEmailLimit = "5"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"whitelist_enabled":   whitelistEnabled,
			"blacklist_enabled":   blacklistEnabled == "true",
			"conditions_enabled":  conditionsEnabled,
			"daily_request_limit": dailyRequestLimit,
			"daily_email_limit":   dailyEmailLimit,
		})
		return
	}
	if r.Method == "POST" {
		var req struct {
			WhitelistEnabled  bool   `json:"whitelist_enabled"`
			BlacklistEnabled  bool   `json:"blacklist_enabled"`
			ConditionsEnabled bool   `json:"conditions_enabled"`
			DailyRequestLimit string `json:"daily_request_limit"`
			DailyEmailLimit   string `json:"daily_email_limit"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.WhitelistEnabled {
			setSetting("whitelist_enabled", "true")
		} else {
			setSetting("whitelist_enabled", "false")
		}
		if req.BlacklistEnabled {
			setSetting("blacklist_enabled", "true")
		} else {
			setSetting("blacklist_enabled", "false")
		}
		if req.ConditionsEnabled {
			setSetting("conditions_enabled", "true")
		} else {
			setSetting("conditions_enabled", "false")
		}
		if req.DailyRequestLimit != "" {
			setSetting("daily_request_limit", req.DailyRequestLimit)
		}
		if req.DailyEmailLimit != "" {
			setSetting("daily_email_limit", req.DailyEmailLimit)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

func handleWhitelist(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT pattern, created_at FROM email_whitelist ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var patterns []map[string]interface{}
		for rows.Next() {
			var pattern string
			var createdAt time.Time
			rows.Scan(&pattern, &createdAt)
			patterns = append(patterns, map[string]interface{}{
				"pattern":    pattern,
				"created_at": createdAt,
			})
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(patterns)
		return
	}
	if r.Method == "POST" {
		var req struct {
			Pattern string `json:"pattern"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		pattern := strings.ToLower(strings.TrimSpace(req.Pattern))
		if pattern == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "模式不能为空"})
			return
		}
		_, err := db.Exec("INSERT OR REPLACE INTO email_whitelist (pattern, created_at) VALUES (?, ?)", 
			pattern, time.Now())
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	if r.Method == "DELETE" {
		var req struct {
			Pattern string `json:"pattern"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		db.Exec("DELETE FROM email_whitelist WHERE pattern=?", req.Pattern)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

func handleBlacklist(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT pattern, created_at FROM email_blacklist ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var patterns []map[string]interface{}
		for rows.Next() {
			var pattern string
			var createdAt time.Time
			rows.Scan(&pattern, &createdAt)
			patterns = append(patterns, map[string]interface{}{
				"pattern":    pattern,
				"created_at": createdAt,
			})
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(patterns)
		return
	}
	if r.Method == "POST" {
		var req struct {
			Pattern string `json:"pattern"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		pattern := strings.ToLower(strings.TrimSpace(req.Pattern))
		if pattern == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "模式不能为空"})
			return
		}
		_, err := db.Exec("INSERT OR REPLACE INTO email_blacklist (pattern, created_at) VALUES (?, ?)", pattern, time.Now())
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	if r.Method == "DELETE" {
		var req struct {
			Pattern string `json:"pattern"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		db.Exec("DELETE FROM email_blacklist WHERE pattern=?", req.Pattern)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

func handleConditions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT pattern, created_at, COALESCE(llm_group_id, ''), COALESCE(search_group_id, '') FROM email_conditions ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var patterns []map[string]interface{}
		for rows.Next() {
			var pattern, llmGroupID, searchGroupID string
			var createdAt time.Time
			rows.Scan(&pattern, &createdAt, &llmGroupID, &searchGroupID)
			patterns = append(patterns, map[string]interface{}{
				"pattern":         pattern,
				"created_at":      createdAt,
				"llm_group_id":    llmGroupID,
				"search_group_id": searchGroupID,
			})
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(patterns)
		return
	}
	if r.Method == "POST" {
		var req struct {
			Pattern       string `json:"pattern"`
			LLMGroupID    string `json:"llm_group_id"`
			SearchGroupID string `json:"search_group_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		pattern := strings.ToLower(strings.TrimSpace(req.Pattern))
		if pattern == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "模式不能为空"})
			return
		}
		_, err := db.Exec("INSERT OR REPLACE INTO email_conditions (pattern, created_at, llm_group_id, search_group_id) VALUES (?, ?, ?, ?)", 
			pattern, time.Now(), req.LLMGroupID, req.SearchGroupID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	if r.Method == "DELETE" {
		var req struct {
			Pattern string `json:"pattern"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		db.Exec("DELETE FROM email_conditions WHERE pattern=?", req.Pattern)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
}

// isEmailAllowed checks if an email is allowed based on whitelist/blacklist settings
// Logic: Default blacklist mode. If both enabled, whitelist takes precedence (whitelist match = allow, even if blacklisted)
func isEmailAllowed(email string) (bool, string) {
	allowed, _, reason, _, _ := isEmailAllowedWithGroups(email)
	return allowed, reason
}

// isEmailAllowedWithGroups checks if an email is allowed and returns group bindings
// Returns: allowed, code, reason, llmGroupID, searchGroupID
// Logic:
// 1. Check blacklist first - if match, deny
// 2. Check whitelist (if enabled) - if enabled and no match, deny
// 3. Check conditions list (if enabled) - if match, return group bindings
// 4. If passed all checks, allow with no group binding
func isEmailAllowedWithGroups(email string) (bool, string, string, string, string) {
	email = strings.ToLower(email)
	
	whitelistEnabled := getSetting("whitelist_enabled") == "true"
	blacklistEnabled := getSetting("blacklist_enabled")
	if blacklistEnabled == "" {
		blacklistEnabled = "true" // Default to blacklist mode
	}
	blacklistOn := blacklistEnabled == "true"
	conditionsEnabled := getSetting("conditions_enabled") == "true"
	
	// Step 1: Check blacklist first (blacklist always takes precedence when enabled)
	if blacklistOn {
		rows, err := db.Query("SELECT pattern FROM email_blacklist")
		if err != nil {
			log.Printf("[EMAIL-FILTER] failed to query blacklist: %v", err)
		} else {
			blacklisted := false
			for rows.Next() {
				var pattern string
				rows.Scan(&pattern)
				if matchEmailPattern(email, pattern) {
					blacklisted = true
					break
				}
			}
			if rowErr := rows.Err(); rowErr != nil {
				log.Printf("Warning: rows iteration error: %v", rowErr)
			}
			rows.Close()
			if blacklisted {
				return false, CodeEmailBlacklisted, "您的邮箱已被限制申请", "", ""
			}
		}
	}
	
	// Step 2: If whitelist is enabled, must match whitelist
	if whitelistEnabled {
		rows, err := db.Query("SELECT pattern FROM email_whitelist")
		if err != nil {
			log.Printf("[EMAIL-FILTER] failed to query whitelist: %v", err)
		} else {
			matched := false
			for rows.Next() {
				var pattern string
				rows.Scan(&pattern)
				if matchEmailPattern(email, pattern) {
					matched = true
					break
				}
			}
			if rowErr := rows.Err(); rowErr != nil {
				log.Printf("Warning: rows iteration error: %v", rowErr)
			}
			rows.Close()
			if !matched {
				return false, CodeEmailNotWhitelisted, "您的邮箱不在白名单中", "", ""
			}
		}
	}
	
	// Step 3: Check conditions list for group bindings (only if enabled)
	if conditionsEnabled {
		rows, err := db.Query("SELECT pattern, COALESCE(llm_group_id, ''), COALESCE(search_group_id, '') FROM email_conditions")
		if err != nil {
			log.Printf("[EMAIL-FILTER] failed to query conditions: %v", err)
		} else {
			var matchedLLMGroupID, matchedSearchGroupID string
			conditionMatched := false
			for rows.Next() {
				var pattern, llmGroupID, searchGroupID string
				rows.Scan(&pattern, &llmGroupID, &searchGroupID)
				if matchEmailPattern(email, pattern) {
					matchedLLMGroupID = llmGroupID
					matchedSearchGroupID = searchGroupID
					conditionMatched = true
					break
				}
			}
			if rowErr := rows.Err(); rowErr != nil {
				log.Printf("Warning: rows iteration error: %v", rowErr)
			}
			rows.Close()
			if conditionMatched {
				return true, "", "", matchedLLMGroupID, matchedSearchGroupID
			}
		}
	}
	
	// Step 4: Passed all checks, allow with no group binding
	return true, "", "", "", ""
}

// matchEmailPattern checks if email matches a pattern
// Pattern can be: full email (user@domain.com) or domain (@domain.com)
func matchEmailPattern(email, pattern string) bool {
	pattern = strings.ToLower(pattern)
	email = strings.ToLower(email)
	
	if strings.HasPrefix(pattern, "@") {
		// Domain pattern: @domain.com matches user@domain.com
		return strings.HasSuffix(email, pattern)
	}
	// Full email match
	return email == pattern
}

// buildAvailableSNQuery builds a SQL query to find an available SN based on product, LLM group,
// search group, and conditions settings. Returns the query string and args.
// If excludeGrouped is true and conditions are enabled, it first tries to exclude SNs with group bindings.
// If excludeFreeOSS is true, SNs belonging to free or open-source license groups are excluded
// (so that trial SN requests don't accidentally pick up a free/oss SN).
func buildAvailableSNQuery(productID int, llmGroupID, searchGroupID string, now time.Time, excludeGrouped bool, excludeFreeOSS bool) (string, []interface{}) {
	productCondition := "(product_id IS NULL OR product_id = 0)"
	if productID > 0 {
		productCondition = "product_id = ?"
	}
	baseCondition := "is_active = 1 AND (expires_at IS NULL OR expires_at > ?) AND usage_count = 0 AND sn NOT IN (SELECT sn FROM email_records)"
	orderClause := " ORDER BY expires_at IS NULL DESC, created_at ASC LIMIT 1"

	var conditions []string
	var args []interface{}

	conditions = append(conditions, productCondition)
	if productID > 0 {
		args = append(args, productID)
	}

	if llmGroupID != "" {
		conditions = append(conditions, "llm_group_id = ?")
		args = append(args, llmGroupID)
	}
	if searchGroupID != "" {
		conditions = append(conditions, "search_group_id = ?")
		args = append(args, searchGroupID)
	}

	// When no group specified and conditions enabled, exclude grouped SNs
	groupExclusion := ""
	if llmGroupID == "" && searchGroupID == "" && excludeGrouped {
		groupExclusion = " AND (llm_group_id IS NULL OR llm_group_id = '') AND (search_group_id IS NULL OR search_group_id = '')"
	}

	// Exclude free and open-source group SNs (for trial SN requests)
	freeOSSExclusion := ""
	if excludeFreeOSS {
		freeOSSExclusion = " AND (license_group_id IS NULL OR (license_group_id NOT LIKE 'free_%' AND license_group_id NOT LIKE 'oss_%'))"
	}

	conditions = append(conditions, baseCondition)
	args = append(args, now)

	query := "SELECT sn, COALESCE(valid_days, 365), expires_at FROM licenses WHERE " +
		strings.Join(conditions, " AND ") + groupExclusion + freeOSSExclusion + orderClause

	return query, args
}

// ============ Auth Server ============

func startAuthServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/activate", handleActivate)
	mux.HandleFunc("/request-sn", handleRequestSN)
	mux.HandleFunc("/request-free-sn", handleRequestFreeSN)
	mux.HandleFunc("/request-oss-sn", handleRequestOpenSourceSN)
	mux.HandleFunc("/api/bind-license", handleBindLicenseAPI)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/report-usage", handleReportUsage)
	mux.HandleFunc("/api/marketplace-auth", handleMarketplaceAuth)
	mux.HandleFunc("/api/marketplace-verify", handleMarketplaceVerify)

	// Wrap with request body size limit (1MB) to prevent abuse
	handler := http.MaxBytesHandler(mux, 1<<20)

	addr := fmt.Sprintf(":%d", authPort)
	if useSSL && sslCert != "" && sslKey != "" {
		log.Printf("Auth server starting on %s (HTTPS)", addr)
		if err := http.ListenAndServeTLS(addr, sslCert, sslKey, handler); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Auth server failed: %v", err)
		}
	} else {
		log.Printf("Auth server starting on %s (HTTP)", addr)
		if err := http.ListenAndServe(addr, handler); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Auth server failed: %v", err)
		}
	}
}

func getMarketplaceSecret() string {
	return os.Getenv("LICENSE_MARKETPLACE_SECRET")
}

func handleMarketplaceAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidRequest, "message": "Method not allowed",
		})
		return
	}

	var req struct {
		SN    string `json:"sn"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidRequest, "message": "无效的请求格式",
		})
		return
	}

	sn := strings.ToUpper(strings.TrimSpace(req.SN))
	email := strings.TrimSpace(req.Email)

	if sn == "" || email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidRequest, "message": "SN 和 Email 不能为空",
		})
		return
	}

	// Verify SN exists, is active, and not expired
	var license License
	err := db.QueryRow("SELECT sn, is_active, expires_at FROM licenses WHERE sn=?", sn).
		Scan(&license.SN, &license.IsActive, &license.ExpiresAt)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidSN, "message": "序列号无效",
		})
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInternalError, "message": "内部错误",
		})
		return
	}
	if !license.IsActive {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeSNDisabled, "message": "序列号已被禁用",
		})
		return
	}
	if time.Now().After(license.ExpiresAt) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeSNExpired, "message": "序列号已过期",
		})
		return
	}

	// Verify email matches the SN in email_records
	var recordEmail string
	err = db.QueryRow("SELECT email FROM email_records WHERE sn=?", sn).Scan(&recordEmail)
	if err != nil || !strings.EqualFold(recordEmail, email) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeEmailMismatch, "message": "邮箱与序列号不匹配",
		})
		return
	}

	// Sign JWT token with HMAC-SHA256
	secret := getMarketplaceSecret()
	if secret == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInternalError, "message": "服务器未配置 marketplace 签名密钥",
		})
		return
	}

	token, err := signMarketplaceToken(sn, email, secret)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInternalError, "message": "令牌签发失败",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, "token": token,
	})
}

// signMarketplaceToken creates a JWT-like token: base64url(header).base64url(payload).base64url(signature)
func signMarketplaceToken(sn, email, secret string) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"sn":      sn,
		"email":   email,
		"purpose": "marketplace_auth",
		"exp":     time.Now().Add(5 * time.Minute).Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64, nil
}

func handleMarketplaceVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidRequest, "message": "无效的请求格式",
		})
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidRequest, "message": "token 不能为空",
		})
		return
	}

	// Split token into 3 parts: header.payload.signature
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidToken, "message": "令牌格式无效",
		})
		return
	}

	secret := getMarketplaceSecret()
	if secret == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInternalError, "message": "服务器未配置 marketplace 签名密钥",
		})
		return
	}

	// Recompute HMAC-SHA256 of header.payload and compare with signature
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(expectedSig, actualSig) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidToken, "message": "令牌签名无效",
		})
		return
	}

	// Decode payload and check expiration
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidToken, "message": "令牌载荷解码失败",
		})
		return
	}

	var payload struct {
		SN    string  `json:"sn"`
		Email string  `json:"email"`
		Exp   float64 `json:"exp"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeInvalidToken, "message": "令牌载荷解析失败",
		})
		return
	}

	// Check expiration
	if time.Now().Unix() > int64(payload.Exp) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "code": CodeTokenExpired, "message": "令牌已过期",
		})
		return
	}

	// Success - return sn and email
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"sn":      payload.SN,
		"email":   payload.Email,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleReportUsage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SN          string  `json:"sn"`
		UsedCredits float64 `json:"used_credits"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "code": CodeInvalidRequest})
		return
	}

	// Validate SN exists in licenses table
	sn := strings.ToUpper(strings.ReplaceAll(req.SN, " ", ""))
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM licenses WHERE sn=?", sn).Scan(&exists)
	if err != nil || exists == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "code": CodeInvalidSN})
		return
	}

	// Validate used_credits >= 0
	if req.UsedCredits < 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "code": CodeInvalidValue})
		return
	}

	// 间隔校验：检查该 SN 最近一次上报时间
	var lastReportedAt sql.NullString
	err = db.QueryRow(
		"SELECT reported_at FROM credits_usage_log WHERE sn=? ORDER BY reported_at DESC LIMIT 1",
		sn,
	).Scan(&lastReportedAt)

	if err == nil && lastReportedAt.Valid {
		var parsedTime time.Time
		var parsed bool
		if t, parseErr := time.Parse("2006-01-02 15:04:05", lastReportedAt.String); parseErr == nil {
			parsedTime = t
			parsed = true
		} else if t, parseErr2 := time.Parse(time.RFC3339, lastReportedAt.String); parseErr2 == nil {
			parsedTime = t.UTC()
			parsed = true
		} else {
			log.Printf("Failed to parse reported_at for SN %s: %v", sn, parseErr)
		}
		if parsed && time.Now().UTC().Sub(parsedTime) < time.Hour {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"code":    "THROTTLED",
			})
			return
		}
	} else if err != nil && err != sql.ErrNoRows {
		log.Printf("Failed to query last report time for SN %s: %v", sn, err)
	}
	// If sql.ErrNoRows or parse error, continue with normal processing (first report or error recovery)

	// Insert into credits_usage_log
	clientIP := getClientIP(r)
	_, err = db.Exec("INSERT INTO credits_usage_log (sn, used_credits, reported_at, client_ip) VALUES (?, ?, datetime('now'), ?)",
		sn, req.UsedCredits, clientIP)
	if err != nil {
		log.Printf("Failed to insert credits usage log: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "code": CodeInternalError})
		return
	}

	// Update licenses: used_credits = MAX(used_credits, reported_value)
	_, err = db.Exec("UPDATE licenses SET used_credits = MAX(used_credits, ?) WHERE sn = ?",
		req.UsedCredits, sn)
	if err != nil {
		log.Printf("Failed to update license used_credits: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "code": CodeInternalError})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}


func handleActivate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct{ SN string `json:"sn"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivationResponse{Success: false, Code: CodeInvalidRequest, Message: "无效的请求格式"})
		return
	}
	
	sn := strings.ToUpper(strings.ReplaceAll(req.SN, " ", ""))
	
	var license License
	var lastUsed sql.NullTime
	var usedCredits float64
	err := db.QueryRow("SELECT sn, created_at, expires_at, description, is_active, usage_count, last_used_at, COALESCE(daily_analysis, 20), COALESCE(license_group_id, ''), COALESCE(llm_group_id, ''), COALESCE(search_group_id, ''), COALESCE(product_id, 0), COALESCE(total_credits, 0), COALESCE(credits_mode, 0), COALESCE(used_credits, 0) FROM licenses WHERE sn=?", sn).
		Scan(&license.SN, &license.CreatedAt, &license.ExpiresAt, &license.Description, &license.IsActive, &license.UsageCount, &lastUsed, &license.DailyAnalysis, &license.LicenseGroupID, &license.LLMGroupID, &license.SearchGroupID, &license.ProductID, &license.TotalCredits, &license.CreditsMode, &usedCredits)
	
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivationResponse{Success: false, Code: CodeInvalidSN, Message: "序列号无效"})
		return
	}
	if !license.IsActive {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivationResponse{Success: false, Code: CodeSNDisabled, Message: "序列号已被禁用"})
		return
	}

	// Get trust level from license group early so we can skip expiration check for permanent_free
	trustLevel := "low"
	refreshInterval := 1 // Default: daily refresh for low trust
	if license.LicenseGroupID != "" {
		var groupTrustLevel string
		err := db.QueryRow("SELECT COALESCE(trust_level, 'low') FROM license_groups WHERE id=?", license.LicenseGroupID).Scan(&groupTrustLevel)
		if err == nil && groupTrustLevel != "" {
			trustLevel = groupTrustLevel
		}
	}

	// Skip expiration check for permanent_free and open_source SN
	if trustLevel != "permanent_free" && trustLevel != "open_source" && time.Now().After(license.ExpiresAt) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivationResponse{Success: false, Code: CodeSNExpired, Message: "序列号已过期"})
		return
	}
	
	// Update usage
	db.Exec("UPDATE licenses SET usage_count=usage_count+1, last_used_at=? WHERE sn=?", time.Now(), sn)

	var bestLLM *LLMConfig
	var bestSearch *SearchConfig

	// For permanent_free and open_source, skip LLM and Search config queries, return empty values
	if trustLevel != "permanent_free" && trustLevel != "open_source" {
		// Get best LLM config for the license's group (or all if no group specified)
		today := time.Now().Format("2006-01-02")
		var llmQuery string
		var llmArgs []interface{}
		if license.LLMGroupID != "" {
			llmQuery = "SELECT id, name, type, base_url, api_key, model, is_active, start_date, end_date, COALESCE(group_id, '') FROM llm_configs WHERE group_id=?"
			llmArgs = []interface{}{license.LLMGroupID}
		} else {
			llmQuery = "SELECT id, name, type, base_url, api_key, model, is_active, start_date, end_date, COALESCE(group_id, '') FROM llm_configs"
		}
		rows, err := db.Query(llmQuery, llmArgs...)
		if err != nil {
			log.Printf("[ACTIVATE] failed to query LLM configs: %v", err)
		} else {
			for rows.Next() {
				var c LLMConfig
				rows.Scan(&c.ID, &c.Name, &c.Type, &c.BaseURL, &c.APIKey, &c.Model, &c.IsActive, &c.StartDate, &c.EndDate, &c.GroupID)
				if !isKeyValidForDate(c.StartDate, c.EndDate, today) {
					continue
				}
				// Priority: latest start_date, then is_active
				if bestLLM == nil || c.StartDate > bestLLM.StartDate || (c.IsActive && !bestLLM.IsActive && c.StartDate == bestLLM.StartDate) {
					bestLLM = &c
				}
			}
			if err := rows.Err(); err != nil {
				log.Printf("Warning: rows iteration error: %v", err)
			}
			rows.Close()
		}

		// Get best Search config for the license's group (or all if no group specified)
		var searchQuery string
		var searchArgs []interface{}
		if license.SearchGroupID != "" {
			searchQuery = "SELECT id, name, type, api_key, is_active, start_date, end_date, COALESCE(group_id, '') FROM search_configs WHERE group_id=?"
			searchArgs = []interface{}{license.SearchGroupID}
		} else {
			searchQuery = "SELECT id, name, type, api_key, is_active, start_date, end_date, COALESCE(group_id, '') FROM search_configs"
		}
		rows, err = db.Query(searchQuery, searchArgs...)
		if err != nil {
			log.Printf("[ACTIVATE] failed to query search configs: %v", err)
		} else {
			for rows.Next() {
				var c SearchConfig
				rows.Scan(&c.ID, &c.Name, &c.Type, &c.APIKey, &c.IsActive, &c.StartDate, &c.EndDate, &c.GroupID)
				if !isKeyValidForDate(c.StartDate, c.EndDate, today) {
					continue
				}
				// Priority: latest start_date, then is_active
				if bestSearch == nil || c.StartDate > bestSearch.StartDate || (c.IsActive && !bestSearch.IsActive && c.StartDate == bestSearch.StartDate) {
					bestSearch = &c
				}
			}
			if err := rows.Err(); err != nil {
				log.Printf("Warning: rows iteration error: %v", err)
			}
			rows.Close()
		}
	}

	// Set refresh interval based on trust level
	if trustLevel == "permanent_free" || trustLevel == "open_source" {
		refreshInterval = 365 // Yearly refresh for permanent free and open source
	} else if trustLevel == "high" {
		refreshInterval = 30 // Monthly refresh for high trust (正式)
	} else {
		refreshInterval = 1 // Daily refresh for low trust (试用)
	}
	
	// Build activation data
	productName := getProductName(license.ProductID)
	extraInfo := getProductExtraInfo(license.ProductID)
	
	activationData := ActivationData{
		ExpiresAt:       license.ExpiresAt.Format(time.RFC3339),
		ActivatedAt:     time.Now().Format(time.RFC3339),
		DailyAnalysis:   license.DailyAnalysis,
		ProductID:       license.ProductID,
		ProductName:     productName,
		TrustLevel:      trustLevel,
		RefreshInterval: refreshInterval,
		ExtraInfo:       extraInfo,
		TotalCredits:    license.TotalCredits,
		CreditsMode:     license.CreditsMode,
		UsedCredits:     usedCredits,
	}
	if bestLLM != nil {
		activationData.LLMType = bestLLM.Type
		activationData.LLMBaseURL = bestLLM.BaseURL
		activationData.LLMAPIKey = bestLLM.APIKey
		activationData.LLMModel = bestLLM.Model
		activationData.LLMStartDate = bestLLM.StartDate
		activationData.LLMEndDate = bestLLM.EndDate
	}
	if bestSearch != nil {
		activationData.SearchType = bestSearch.Type
		activationData.SearchAPIKey = bestSearch.APIKey
		activationData.SearchStartDate = bestSearch.StartDate
		activationData.SearchEndDate = bestSearch.EndDate
	}
	
	dataJSON, _ := json.Marshal(activationData)
	encryptedData, err := encryptData(dataJSON, sn)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivationResponse{Success: false, Code: CodeEncryptFailed, Message: "加密失败"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ActivationResponse{
		Success: true, Code: CodeSuccess, Message: "激活成功", EncryptedData: encryptedData, ExpiresAt: license.ExpiresAt.Format("2006-01-02"),
	})
}

func handleRequestSN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct{ 
		Email     string `json:"email"` 
		ProductID int    `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInvalidRequest, Message: "无效的请求格式"})
		return
	}
	
	email := strings.ToLower(strings.TrimSpace(req.Email))
	productID := req.ProductID // 0 = default/unclassified
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInvalidEmail, Message: "请输入有效的邮箱地址"})
		return
	}
	
	// Check email whitelist/blacklist and get group bindings
	allowed, code, reason, llmGroupID, searchGroupID := isEmailAllowedWithGroups(email)
	if !allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: code, Message: reason})
		return
	}
	
	// Check if email already has a commercial/trial SN for this product
	var existingSN string
	if err := db.QueryRow(`SELECT e.sn FROM email_records e 
		JOIN licenses l ON e.sn = l.sn 
		WHERE e.email=? AND e.product_id=? AND e.sn_type='commercial'`, email, productID).Scan(&existingSN); err == nil {
		// Email already has a commercial SN for this product, return it
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeEmailAlreadyUsed, Message: "您已申请过该产品的序列号", SN: existingSN})
		return
	}
	
	// Check if there's an orphaned commercial record (SN was deleted) for this email+product
	var orphanedSN string
	if err := db.QueryRow(`SELECT sn FROM email_records WHERE email=? AND product_id=? AND sn_type='commercial'`, email, productID).Scan(&orphanedSN); err == nil {
		// SN was deleted, remove the old email record
		db.Exec("DELETE FROM email_records WHERE email=? AND product_id=? AND sn_type='commercial'", email, productID)
		log.Printf("[REQUEST-SN] Old SN %s for email %s product %d was deleted, generating new one", orphanedSN, email, productID)
	}
	
	// Get rate limit settings
	dailyRequestLimitStr := getSetting("daily_request_limit")
	if dailyRequestLimitStr == "" {
		dailyRequestLimitStr = "5"
	}
	dailyRequestLimit := 5
	fmt.Sscanf(dailyRequestLimitStr, "%d", &dailyRequestLimit)
	
	dailyEmailLimitStr := getSetting("daily_email_limit")
	if dailyEmailLimitStr == "" {
		dailyEmailLimitStr = "5"
	}
	dailyEmailLimit := 5
	fmt.Sscanf(dailyEmailLimitStr, "%d", &dailyEmailLimit)
	
	// Check rate limit
	clientIP := getClientIP(r)
	today := time.Now().Format("2006-01-02")
	
	// Check daily request count for this IP
	var count int
	db.QueryRow("SELECT count FROM request_limits WHERE ip=? AND date=?", clientIP, today).Scan(&count)
	if count >= dailyRequestLimit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeRateLimitExceeded, Message: fmt.Sprintf("今日申请次数已达上限（%d次），请明天再试", dailyRequestLimit)})
		return
	}
	
	// Check unique email count for this IP today
	var emailCount int
	db.QueryRow("SELECT COUNT(DISTINCT email) FROM email_records WHERE ip=? AND DATE(created_at)=?", clientIP, today).Scan(&emailCount)
	if emailCount >= dailyEmailLimit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeEmailLimitExceeded, Message: fmt.Sprintf("同一IP今日使用不同邮箱申请次数已达上限（%d个），请明天再试", dailyEmailLimit)})
		return
	}
	
	db.Exec("INSERT OR REPLACE INTO request_limits (ip, date, count) VALUES (?, ?, ?)", clientIP, today, count+1)
	
	// Find an available SN from existing licenses that:
	// 1. Has matching product_id
	// 2. Has matching LLM group (or no group if llmGroupID is empty)
	// 3. Has matching Search group (or no group if searchGroupID is empty)
	// 4. Is not already bound to an email
	// 5. Is active
	// 6. Has not been used (usage_count = 0)
	// 7. Either not activated (expires_at IS NULL) or not expired (expires_at > now)
	now := time.Now()
	var sn string
	var validDays int
	var hasExpiresAt bool
	
	conditionsEnabled := getSetting("conditions_enabled") == "true"
	excludeGrouped := conditionsEnabled && llmGroupID == "" && searchGroupID == ""
	
	query, args := buildAvailableSNQuery(productID, llmGroupID, searchGroupID, now, excludeGrouped, true)
	
	var nullableExpiresAt sql.NullTime
	err := db.QueryRow(query, args...).Scan(&sn, &validDays, &nullableExpiresAt)
	
	// If no ungrouped SNs available and we were excluding grouped ones, fall back to any available SN
	if err != nil && excludeGrouped {
		log.Printf("[REQUEST-SN] No ungrouped SNs available, falling back to any available SN for email %s", email)
		query, args = buildAvailableSNQuery(productID, llmGroupID, searchGroupID, now, false, true)
		err = db.QueryRow(query, args...).Scan(&sn, &validDays, &nullableExpiresAt)
	}
	
	if err != nil {
		// Diagnostic logging: count available SNs at each filter stage
		var totalCount, activeCount, unexpiredCount, unusedCount, unboundCount int
		db.QueryRow("SELECT COUNT(*) FROM licenses").Scan(&totalCount)
		db.QueryRow("SELECT COUNT(*) FROM licenses WHERE is_active = 1").Scan(&activeCount)
		db.QueryRow("SELECT COUNT(*) FROM licenses WHERE is_active = 1 AND (expires_at IS NULL OR expires_at > ?)", now).Scan(&unexpiredCount)
		db.QueryRow("SELECT COUNT(*) FROM licenses WHERE is_active = 1 AND (expires_at IS NULL OR expires_at > ?) AND usage_count = 0", now).Scan(&unusedCount)
		db.QueryRow("SELECT COUNT(*) FROM licenses WHERE is_active = 1 AND (expires_at IS NULL OR expires_at > ?) AND usage_count = 0 AND sn NOT IN (SELECT sn FROM email_records)", now).Scan(&unboundCount)
		log.Printf("[REQUEST-SN] Diagnostic - Total: %d, Active: %d, Unexpired: %d, Unused(usage_count=0): %d, Unbound: %d",
			totalCount, activeCount, unexpiredCount, unusedCount, unboundCount)
		log.Printf("[REQUEST-SN] No available SN found for email %s (Product: %d, LLM Group: %s, Search Group: %s): %v", email, productID, llmGroupID, searchGroupID, err)
		log.Printf("[REQUEST-SN] Query: %s", query)
		var msg string
		if productID > 0 || llmGroupID != "" || searchGroupID != "" {
			msg = fmt.Sprintf("暂无匹配的可用序列号（产品ID: %d, LLM分组: %s, 搜索分组: %s），请联系管理员", productID, llmGroupID, searchGroupID)
		} else {
			msg = "暂无可用序列号，请联系管理员"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeNoAvailableSN, Message: msg})
		return
	}
	
	// Determine expiry date:
	// - If expires_at is NULL (new SN), calculate from now + valid_days
	// - If expires_at is set (old SN), use existing expiry date
	var expiresAt time.Time
	if nullableExpiresAt.Valid {
		expiresAt = nullableExpiresAt.Time
		hasExpiresAt = true
	} else {
		expiresAt = now.AddDate(0, 0, validDays)
		hasExpiresAt = false
	}
	
	// Bind the SN to the email and update license in a transaction
	tx, txErr := db.Begin()
	if txErr != nil {
		log.Printf("[REQUEST-SN] Failed to begin transaction: %v", txErr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "系统错误，请稍后重试"})
		return
	}
	
	_, txErr = tx.Exec("INSERT INTO email_records (email, sn, ip, created_at, product_id, sn_type) VALUES (?, ?, ?, ?, ?, 'commercial')", email, sn, clientIP, now, productID)
	if txErr != nil {
		tx.Rollback()
		log.Printf("[REQUEST-SN] Failed to bind SN to email %s: %v", email, txErr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "绑定邮箱失败"})
		return
	}
	
	// Update the license: set expires_at (only if not already set) and description
	if hasExpiresAt {
		// Old SN with existing expiry, just update description
		_, txErr = tx.Exec("UPDATE licenses SET description = ? WHERE sn = ?", fmt.Sprintf("Email申请: %s", email), sn)
	} else {
		// New SN, set expiry date
		_, txErr = tx.Exec("UPDATE licenses SET expires_at = ?, description = ? WHERE sn = ?", expiresAt, fmt.Sprintf("Email申请: %s", email), sn)
	}
	if txErr != nil {
		tx.Rollback()
		log.Printf("[REQUEST-SN] Failed to update license for email %s: %v", email, txErr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "更新序列号失败"})
		return
	}
	
	if txErr = tx.Commit(); txErr != nil {
		log.Printf("[REQUEST-SN] Failed to commit transaction: %v", txErr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "系统错误，请稍后重试"})
		return
	}
	
	log.Printf("[REQUEST-SN] SN allocated for email %s from IP %s: %s (Product: %d, LLM Group: %s, Search Group: %s, HasExpiry: %v)", email, clientIP, sn, productID, llmGroupID, searchGroupID, hasExpiresAt)
	
	// Calculate days left
	daysLeft := int(expiresAt.Sub(now).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}
	
	// Send email with SN (async, don't block response)
	go func() {
		if err := sendSNEmail(email, sn, expiresAt, productID); err != nil {
			log.Printf("[EMAIL] Failed to send SN email to %s: %v", email, err)
		} else {
			log.Printf("[EMAIL] SN email sent successfully to %s", email)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeSuccess, Message: fmt.Sprintf("序列号分配成功，有效期 %d 天", daysLeft), SN: sn})
}

func handleRequestFreeSN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email     string `json:"email"`
		ProductID int    `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInvalidRequest, Message: "无效的请求格式"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	productID := req.ProductID
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInvalidEmail, Message: "请输入有效的邮箱地址"})
		return
	}

	// Check email whitelist/blacklist (reuse existing logic, ignore group bindings)
	allowed, code, reason, _, _ := isEmailAllowedWithGroups(email)
	if !allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: code, Message: reason})
		return
	}

	// Idempotency: if the same email+product already has a free SN, return it
	freeGroupID := fmt.Sprintf("free_%d", productID)
	var existingSN string
	if err := db.QueryRow(`SELECT e.sn FROM email_records e
		JOIN licenses l ON e.sn = l.sn
		WHERE e.email = ? AND e.product_id = ? AND e.sn_type = 'free' AND l.license_group_id = ?`,
		email, productID, freeGroupID).Scan(&existingSN); err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeSuccess, Message: "免费序列号已存在", SN: existingSN})
		return
	}

	// Check if there's an orphaned free record (SN was deleted) for this email+product
	var oldSN string
	if err := db.QueryRow(`SELECT sn FROM email_records WHERE email = ? AND product_id = ? AND sn_type = 'free'`,
		email, productID).Scan(&oldSN); err == nil {
		db.Exec("DELETE FROM email_records WHERE email = ? AND product_id = ? AND sn_type = 'free'", email, productID)
		log.Printf("[REQUEST-FREE-SN] Removed orphaned free record for %s (old SN: %s)", email, oldSN)
	}

	// Rate limiting (reuse existing logic)
	dailyRequestLimitStr := getSetting("daily_request_limit")
	if dailyRequestLimitStr == "" {
		dailyRequestLimitStr = "5"
	}
	dailyRequestLimit := 5
	fmt.Sscanf(dailyRequestLimitStr, "%d", &dailyRequestLimit)

	dailyEmailLimitStr := getSetting("daily_email_limit")
	if dailyEmailLimitStr == "" {
		dailyEmailLimitStr = "5"
	}
	dailyEmailLimit := 5
	fmt.Sscanf(dailyEmailLimitStr, "%d", &dailyEmailLimit)

	clientIP := getClientIP(r)
	today := time.Now().Format("2006-01-02")

	var count int
	db.QueryRow("SELECT count FROM request_limits WHERE ip=? AND date=?", clientIP, today).Scan(&count)
	if count >= dailyRequestLimit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeRateLimitExceeded, Message: fmt.Sprintf("今日申请次数已达上限（%d次），请明天再试", dailyRequestLimit)})
		return
	}

	var emailCount int
	db.QueryRow("SELECT COUNT(DISTINCT email) FROM email_records WHERE ip=? AND DATE(created_at)=?", clientIP, today).Scan(&emailCount)
	if emailCount >= dailyEmailLimit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeEmailLimitExceeded, Message: fmt.Sprintf("同一IP今日使用不同邮箱申请次数已达上限（%d个），请明天再试", dailyEmailLimit)})
		return
	}

	db.Exec("INSERT OR REPLACE INTO request_limits (ip, date, count) VALUES (?, ?, ?)", clientIP, today, count+1)

	// Ensure the free group exists
	groupID := getOrCreateProductFreeGroup(productID)
	if groupID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建免费授权组失败"})
		return
	}

	// Generate a new free SN
	sn := generateSN()
	now := time.Now()
	validDays := 36500
	expiresAt := now.AddDate(0, 0, validDays)

	// Use transaction to ensure license creation and email binding are atomic
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[REQUEST-FREE-SN] Failed to begin transaction: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建免费序列号失败"})
		return
	}

	_, err = tx.Exec(
		"INSERT INTO licenses (sn, created_at, expires_at, valid_days, description, is_active, daily_analysis, license_group_id, llm_group_id, search_group_id, product_id, total_credits, credits_mode) VALUES (?, ?, ?, ?, ?, 1, 0, ?, '', '', ?, 0, 0)",
		sn, now, expiresAt, validDays, fmt.Sprintf("永久免费: %s", email), groupID, productID)
	if err != nil {
		tx.Rollback()
		log.Printf("[REQUEST-FREE-SN] Failed to create free license for email %s: %v", email, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建免费序列号失败"})
		return
	}

	// Bind SN to email
	_, err = tx.Exec("INSERT INTO email_records (email, sn, ip, created_at, product_id, sn_type) VALUES (?, ?, ?, ?, ?, 'free')",
		email, sn, clientIP, now, productID)
	if err != nil {
		tx.Rollback()
		log.Printf("[REQUEST-FREE-SN] Failed to bind free SN to email %s: %v", email, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "绑定邮箱失败"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[REQUEST-FREE-SN] Failed to commit transaction: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建免费序列号失败"})
		return
	}

	log.Printf("[REQUEST-FREE-SN] Free SN created for email %s from IP %s: %s (Product: %d)", email, clientIP, sn, productID)

	// Send confirmation email (async)
	go func() {
		if err := sendSNEmail(email, sn, expiresAt, productID); err != nil {
			log.Printf("[EMAIL] Failed to send free SN email to %s: %v", email, err)
		} else {
			log.Printf("[EMAIL] Free SN email sent successfully to %s", email)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeSuccess, Message: "免费序列号创建成功", SN: sn})
}

func handleRequestOpenSourceSN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email     string `json:"email"`
		ProductID int    `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInvalidRequest, Message: "无效的请求格式"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	productID := req.ProductID
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInvalidEmail, Message: "请输入有效的邮箱地址"})
		return
	}

	// Check email whitelist/blacklist (reuse existing logic, ignore group bindings)
	allowed, code, reason, _, _ := isEmailAllowedWithGroups(email)
	if !allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: code, Message: reason})
		return
	}

	// Idempotency: if the same email+product already has an open source SN, return it
	ossGroupID := fmt.Sprintf("oss_%d", productID)
	var existingSN string
	if err := db.QueryRow(`SELECT e.sn FROM email_records e
		JOIN licenses l ON e.sn = l.sn
		WHERE e.email = ? AND e.product_id = ? AND e.sn_type = 'oss' AND l.license_group_id = ?`,
		email, productID, ossGroupID).Scan(&existingSN); err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeSuccess, Message: "开源授权序列号已存在", SN: existingSN})
		return
	}

	// Check if there's an orphaned oss record (SN was deleted) for this email+product
	var oldSN string
	if err := db.QueryRow(`SELECT sn FROM email_records WHERE email = ? AND product_id = ? AND sn_type = 'oss'`,
		email, productID).Scan(&oldSN); err == nil {
		db.Exec("DELETE FROM email_records WHERE email = ? AND product_id = ? AND sn_type = 'oss'", email, productID)
		log.Printf("[REQUEST-OSS-SN] Removed orphaned oss record for %s (old SN: %s)", email, oldSN)
	}

	// Rate limiting (reuse existing logic)
	dailyRequestLimitStr := getSetting("daily_request_limit")
	if dailyRequestLimitStr == "" {
		dailyRequestLimitStr = "5"
	}
	dailyRequestLimit := 5
	fmt.Sscanf(dailyRequestLimitStr, "%d", &dailyRequestLimit)

	dailyEmailLimitStr := getSetting("daily_email_limit")
	if dailyEmailLimitStr == "" {
		dailyEmailLimitStr = "5"
	}
	dailyEmailLimit := 5
	fmt.Sscanf(dailyEmailLimitStr, "%d", &dailyEmailLimit)

	clientIP := getClientIP(r)
	today := time.Now().Format("2006-01-02")

	var count int
	db.QueryRow("SELECT count FROM request_limits WHERE ip=? AND date=?", clientIP, today).Scan(&count)
	if count >= dailyRequestLimit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeRateLimitExceeded, Message: fmt.Sprintf("今日申请次数已达上限（%d次），请明天再试", dailyRequestLimit)})
		return
	}

	var emailCount int
	db.QueryRow("SELECT COUNT(DISTINCT email) FROM email_records WHERE ip=? AND DATE(created_at)=?", clientIP, today).Scan(&emailCount)
	if emailCount >= dailyEmailLimit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeEmailLimitExceeded, Message: fmt.Sprintf("同一IP今日使用不同邮箱申请次数已达上限（%d个），请明天再试", dailyEmailLimit)})
		return
	}

	db.Exec("INSERT OR REPLACE INTO request_limits (ip, date, count) VALUES (?, ?, ?)", clientIP, today, count+1)

	// Ensure the open source group exists
	groupID := getOrCreateProductOpenSourceGroup(productID)
	if groupID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建开源授权组失败"})
		return
	}

	// Generate a new open source SN
	sn := generateSN()
	now := time.Now()
	validDays := 36500
	expiresAt := now.AddDate(0, 0, validDays)

	// Use transaction to ensure license creation and email binding are atomic
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[REQUEST-OSS-SN] Failed to begin transaction: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建开源授权序列号失败"})
		return
	}

	_, err = tx.Exec(
		"INSERT INTO licenses (sn, created_at, expires_at, valid_days, description, is_active, daily_analysis, license_group_id, llm_group_id, search_group_id, product_id, total_credits, credits_mode) VALUES (?, ?, ?, ?, ?, 1, 0, ?, '', '', ?, 0, 0)",
		sn, now, expiresAt, validDays, fmt.Sprintf("开源授权: %s", email), groupID, productID)
	if err != nil {
		tx.Rollback()
		log.Printf("[REQUEST-OSS-SN] Failed to create open source license for email %s: %v", email, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建开源授权序列号失败"})
		return
	}

	// Bind SN to email
	_, err = tx.Exec("INSERT INTO email_records (email, sn, ip, created_at, product_id, sn_type) VALUES (?, ?, ?, ?, ?, 'oss')",
		email, sn, clientIP, now, productID)
	if err != nil {
		tx.Rollback()
		log.Printf("[REQUEST-OSS-SN] Failed to bind open source SN to email %s: %v", email, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "绑定邮箱失败"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[REQUEST-OSS-SN] Failed to commit transaction: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestSNResponse{Success: false, Code: CodeInternalError, Message: "创建开源授权序列号失败"})
		return
	}

	log.Printf("[REQUEST-OSS-SN] Open source SN created for email %s from IP %s: %s (Product: %d)", email, clientIP, sn, productID)

	// Send confirmation email (async)
	go func() {
		if err := sendSNEmail(email, sn, expiresAt, productID); err != nil {
			log.Printf("[EMAIL] Failed to send open source SN email to %s: %v", email, err)
		} else {
			log.Printf("[EMAIL] Open source SN email sent successfully to %s", email)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeSuccess, Message: "开源授权序列号创建成功", SN: sn})
}

// ============ Backup and Restore Handlers ============

// BackupInfo contains metadata about a backup
type BackupInfo struct {
	Type         string         `json:"type"`          // "full" or "incremental"
	Version      string         `json:"version"`       // Backup format version
	Domain       string         `json:"domain"`        // Server domain/identifier
	CreatedAt    string         `json:"created_at"`    // Backup creation time
	RecordCounts map[string]int `json:"record_counts"` // Count of records per table
}

// BackupData contains the complete backup data
type BackupData struct {
	BackupInfo       BackupInfo                `json:"backup_info"`
	Settings         map[string]string         `json:"settings,omitempty"`
	Licenses         []map[string]interface{}  `json:"licenses,omitempty"`
	LLMGroups        []map[string]interface{}  `json:"llm_groups,omitempty"`
	SearchGroups     []map[string]interface{}  `json:"search_groups,omitempty"`
	LicenseGroups    []map[string]interface{}  `json:"license_groups,omitempty"`
	ProductTypes     []map[string]interface{}  `json:"product_types,omitempty"`
	ProductExtraInfo []map[string]interface{}  `json:"product_extra_info,omitempty"`
	LLMConfigs       []map[string]interface{}  `json:"llm_configs,omitempty"`
	SearchConfigs    []map[string]interface{}  `json:"search_configs,omitempty"`
	EmailRecords     []map[string]interface{}  `json:"email_records,omitempty"`
	EmailWhitelist   []map[string]interface{}  `json:"email_whitelist,omitempty"`
	EmailBlacklist   []map[string]interface{}  `json:"email_blacklist,omitempty"`
	EmailConditions  []map[string]interface{}  `json:"email_conditions,omitempty"`
}

// BackupHistory stores backup history record
type BackupHistory struct {
	Time        string `json:"time"`
	Type        string `json:"type"`
	RecordCount int    `json:"record_count"`
	Filename    string `json:"filename"`
}

func handleBackupSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		domain := getSetting("backup_domain")
		lastBackupTime := getSetting("last_backup_time")
		lastBackupType := getSetting("last_backup_type")
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domain":           domain,
			"last_backup_time": lastBackupTime,
			"last_backup_type": lastBackupType,
		})
		return
	}
	
	if r.Method == "POST" {
		var req struct {
			Domain string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		setSetting("backup_domain", req.Domain)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleBackupCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Type   string `json:"type"`   // "full" or "incremental"
		Domain string `json:"domain"` // Server identifier
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if req.Type != "full" && req.Type != "incremental" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "无效的备份类型，必须是 full 或 incremental",
		})
		return
	}
	
	// Get last backup time for incremental backup
	lastBackupTime := getSetting("last_backup_time")
	var lastBackupTimestamp time.Time
	if lastBackupTime != "" {
		lastBackupTimestamp, _ = time.Parse("2006-01-02 15:04:05", lastBackupTime)
	}
	
	now := time.Now()
	backupData := BackupData{
		BackupInfo: BackupInfo{
			Type:         req.Type,
			Version:      "1.0",
			Domain:       req.Domain,
			CreatedAt:    now.Format("2006-01-02 15:04:05"),
			RecordCounts: make(map[string]int),
		},
	}
	
	totalRecords := 0
	
	// Backup based on type
	if req.Type == "full" {
		backupData.Settings = backupAllSettings()
		backupData.Licenses = backupTable("licenses", "", nil)
		backupData.LLMGroups = backupTable("llm_groups", "", nil)
		backupData.SearchGroups = backupTable("search_groups", "", nil)
		backupData.LicenseGroups = backupTable("license_groups", "", nil)
		backupData.ProductTypes = backupTable("product_types", "", nil)
		backupData.ProductExtraInfo = backupTable("product_extra_info", "", nil)
		backupData.LLMConfigs = backupTable("llm_configs", "", nil)
		backupData.SearchConfigs = backupTable("search_configs", "", nil)
		backupData.EmailRecords = backupTable("email_records", "", nil)
		backupData.EmailWhitelist = backupTable("email_whitelist", "", nil)
		backupData.EmailBlacklist = backupTable("email_blacklist", "", nil)
		backupData.EmailConditions = backupTable("email_conditions", "", nil)
	} else {
		// Incremental: only backup records created/modified after last backup
		timeCondition := "created_at > ?"
		backupData.Settings = backupAllSettings() // Always include settings
		backupData.Licenses = backupTable("licenses", timeCondition, lastBackupTimestamp)
		backupData.LLMGroups = backupTable("llm_groups", "", nil) // Groups don't have timestamps
		backupData.SearchGroups = backupTable("search_groups", "", nil)
		backupData.LicenseGroups = backupTable("license_groups", "", nil)
		backupData.ProductTypes = backupTable("product_types", "", nil)
		backupData.ProductExtraInfo = backupTable("product_extra_info", "", nil)
		backupData.LLMConfigs = backupTable("llm_configs", "", nil)
		backupData.SearchConfigs = backupTable("search_configs", "", nil)
		backupData.EmailRecords = backupTable("email_records", timeCondition, lastBackupTimestamp)
		backupData.EmailWhitelist = backupTable("email_whitelist", timeCondition, lastBackupTimestamp)
		backupData.EmailBlacklist = backupTable("email_blacklist", timeCondition, lastBackupTimestamp)
		backupData.EmailConditions = backupTable("email_conditions", timeCondition, lastBackupTimestamp)
	}
	
	// Calculate record counts
	backupData.BackupInfo.RecordCounts["settings"] = len(backupData.Settings)
	backupData.BackupInfo.RecordCounts["licenses"] = len(backupData.Licenses)
	backupData.BackupInfo.RecordCounts["llm_groups"] = len(backupData.LLMGroups)
	backupData.BackupInfo.RecordCounts["search_groups"] = len(backupData.SearchGroups)
	backupData.BackupInfo.RecordCounts["license_groups"] = len(backupData.LicenseGroups)
	backupData.BackupInfo.RecordCounts["product_types"] = len(backupData.ProductTypes)
	backupData.BackupInfo.RecordCounts["product_extra_info"] = len(backupData.ProductExtraInfo)
	backupData.BackupInfo.RecordCounts["llm_configs"] = len(backupData.LLMConfigs)
	backupData.BackupInfo.RecordCounts["search_configs"] = len(backupData.SearchConfigs)
	backupData.BackupInfo.RecordCounts["email_records"] = len(backupData.EmailRecords)
	backupData.BackupInfo.RecordCounts["email_whitelist"] = len(backupData.EmailWhitelist)
	backupData.BackupInfo.RecordCounts["email_blacklist"] = len(backupData.EmailBlacklist)
	backupData.BackupInfo.RecordCounts["email_conditions"] = len(backupData.EmailConditions)
	
	for _, count := range backupData.BackupInfo.RecordCounts {
		totalRecords += count
	}
	
	// Generate filename: backup_<domain>_<type>_<date>.json
	safeDomain := strings.ReplaceAll(req.Domain, ".", "_")
	safeDomain = strings.ReplaceAll(safeDomain, "/", "_")
	safeDomain = strings.ReplaceAll(safeDomain, ":", "_")
	filename := fmt.Sprintf("backup_%s_%s_%s.json", safeDomain, req.Type, now.Format("20060102_150405"))
	
	// Update last backup time
	setSetting("last_backup_time", now.Format("2006-01-02 15:04:05"))
	setSetting("last_backup_type", req.Type)
	
	// Save backup history
	saveBackupHistory(BackupHistory{
		Time:        now.Format("2006-01-02 15:04:05"),
		Type:        req.Type,
		RecordCount: totalRecords,
		Filename:    filename,
	})
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"filename":     filename,
		"record_count": totalRecords,
		"data":         backupData,
	})
}

func backupAllSettings() map[string]string {
	settings := make(map[string]string)
	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		return settings
	}
	defer rows.Close()
	
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err == nil {
			// Skip sensitive settings and backup-related settings
			if key != "admin_password" && !strings.HasPrefix(key, "backup_") && !strings.HasPrefix(key, "last_backup") {
				settings[key] = value
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: rows iteration error: %v", err)
	}
	return settings
}

func backupTable(tableName, condition string, conditionArg interface{}) []map[string]interface{} {
	// Validate table name to prevent SQL injection (allow only alphanumeric and underscore)
	if !isValidIdentifier(tableName) {
		log.Printf("[BACKUP] Invalid table name: %s", tableName)
		return nil
	}
	
	var rows *sql.Rows
	var err error
	
	if condition != "" && conditionArg != nil {
		rows, err = db.Query(fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s", tableName, condition), conditionArg)
	} else {
		rows, err = db.Query(fmt.Sprintf("SELECT * FROM \"%s\"", tableName))
	}
	
	if err != nil {
		log.Printf("[BACKUP] Error querying table %s: %v", tableName, err)
		return nil
	}
	defer rows.Close()
	
	columns, err := rows.Columns()
	if err != nil {
		return nil
	}
	
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: rows iteration error: %v", err)
	}
	return result
}

func handleBackupRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Limit request body to 100MB to prevent DoS
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)
	
	var req struct {
		Type string     `json:"type"` // "full" or "incremental"
		Data BackupData `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "解析请求失败: " + err.Error(),
		})
		return
	}
	
	// Validate type match
	if req.Type != req.Data.BackupInfo.Type {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("恢复类型(%s)与备份文件类型(%s)不匹配", req.Type, req.Data.BackupInfo.Type),
		})
		return
	}
	
	dbLock.Lock()
	defer dbLock.Unlock()
	
	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "开始事务失败: " + err.Error(),
		})
		return
	}
	
	var restoredCount int
	
	if req.Type == "full" {
		// Full restore: delete all existing data first
		tables := []string{
			"licenses", "llm_groups", "search_groups", "license_groups",
			"product_types", "product_extra_info", "llm_configs", "search_configs",
			"email_records", "email_whitelist", "email_blacklist", "email_conditions",
		}
		for _, table := range tables {
			if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
				tx.Rollback()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("清空表 %s 失败: %v", table, err),
				})
				return
			}
		}
		// Also clear non-sensitive settings (keep admin_password)
		tx.Exec("DELETE FROM settings WHERE key != 'admin_password'")
	}
	
	// Restore settings
	for key, value := range req.Data.Settings {
		if key != "admin_password" {
			tx.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
			restoredCount++
		}
	}
	
	// Restore tables
	restoredCount += restoreTable(tx, "licenses", req.Data.Licenses, req.Type == "incremental")
	restoredCount += restoreTable(tx, "llm_groups", req.Data.LLMGroups, req.Type == "incremental")
	restoredCount += restoreTable(tx, "search_groups", req.Data.SearchGroups, req.Type == "incremental")
	restoredCount += restoreTable(tx, "license_groups", req.Data.LicenseGroups, req.Type == "incremental")
	restoredCount += restoreTable(tx, "product_types", req.Data.ProductTypes, req.Type == "incremental")
	restoredCount += restoreTable(tx, "product_extra_info", req.Data.ProductExtraInfo, req.Type == "incremental")
	restoredCount += restoreTable(tx, "llm_configs", req.Data.LLMConfigs, req.Type == "incremental")
	restoredCount += restoreTable(tx, "search_configs", req.Data.SearchConfigs, req.Type == "incremental")
	restoredCount += restoreTable(tx, "email_records", req.Data.EmailRecords, req.Type == "incremental")
	restoredCount += restoreTable(tx, "email_whitelist", req.Data.EmailWhitelist, req.Type == "incremental")
	restoredCount += restoreTable(tx, "email_blacklist", req.Data.EmailBlacklist, req.Type == "incremental")
	restoredCount += restoreTable(tx, "email_conditions", req.Data.EmailConditions, req.Type == "incremental")
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "提交事务失败: " + err.Error(),
		})
		return
	}
	
	// Reload ports in case they changed
	loadPorts()
	loadSSLConfig()
	
	typeLabel := "完全"
	if req.Type == "incremental" {
		typeLabel = "增量"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("%s恢复完成，共恢复 %d 条记录", typeLabel, restoredCount),
	})
}

func restoreTable(tx *sql.Tx, tableName string, data []map[string]interface{}, merge bool) int {
	if len(data) == 0 {
		return 0
	}
	
	// Validate table name to prevent SQL injection
	if !isValidIdentifier(tableName) {
		log.Printf("[RESTORE] Invalid table name: %s", tableName)
		return 0
	}
	
	count := 0
	for _, row := range data {
		columns := make([]string, 0, len(row))
		placeholders := make([]string, 0, len(row))
		values := make([]interface{}, 0, len(row))
		
		for col, val := range row {
			// Validate column name
			if !isValidIdentifier(col) {
				log.Printf("[RESTORE] Skipping invalid column name: %s", col)
				continue
			}
			columns = append(columns, fmt.Sprintf("\"%s\"", col))
			placeholders = append(placeholders, "?")
			values = append(values, val)
		}
		
		if len(columns) == 0 {
			continue
		}
		
		var query string
		if merge {
			query = fmt.Sprintf("INSERT OR REPLACE INTO \"%s\" (%s) VALUES (%s)",
				tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
		} else {
			query = fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s)",
				tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
		}
		
		if _, err := tx.Exec(query, values...); err != nil {
			log.Printf("[RESTORE] Error inserting into %s: %v", tableName, err)
			continue
		}
		count++
	}
	return count
}

func handleBackupHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	historyJSON := getSetting("backup_history")
	var history []BackupHistory
	if historyJSON != "" {
		json.Unmarshal([]byte(historyJSON), &history)
	}
	
	// Return last 20 records
	if len(history) > 20 {
		history = history[len(history)-20:]
	}
	
	// Reverse to show newest first
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
	})
}

func saveBackupHistory(record BackupHistory) {
	historyJSON := getSetting("backup_history")
	var history []BackupHistory
	if historyJSON != "" {
		json.Unmarshal([]byte(historyJSON), &history)
	}
	
	history = append(history, record)
	
	// Keep only last 50 records
	if len(history) > 50 {
		history = history[len(history)-50:]
	}
	
	newHistoryJSON, _ := json.Marshal(history)
	setSetting("backup_history", string(newHistoryJSON))
}

// handleCreditsUsageLog returns the credits usage log for a given SN with pagination
func handleCreditsUsageLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sn := r.URL.Query().Get("sn")
	if sn == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"records": []interface{}{}, "total": 0, "page": 1, "pageSize": 20, "totalPages": 0})
		return
	}

	// Parse pagination params
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get total count
	var total int
	db.QueryRow("SELECT COUNT(*) FROM credits_usage_log WHERE sn=?", sn).Scan(&total)

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Get license info for summary
	var totalCredits, usedCredits float64
	var creditsMode int
	db.QueryRow("SELECT COALESCE(total_credits, 0), COALESCE(used_credits, 0), COALESCE(credits_mode, 0) FROM licenses WHERE sn=?", sn).Scan(&totalCredits, &usedCredits, &creditsMode)

	// Query with pagination
	offset := (page - 1) * pageSize
	rows, err := db.Query("SELECT sn, used_credits, reported_at, COALESCE(client_ip, '') FROM credits_usage_log WHERE sn=? ORDER BY reported_at DESC LIMIT ? OFFSET ?", sn, pageSize, offset)
	if err != nil {
		log.Printf("Failed to query credits usage log: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"records": []interface{}{}, "total": 0, "page": page, "pageSize": pageSize, "totalPages": 0})
		return
	}
	defer rows.Close()

	type UsageLogEntry struct {
		SN          string  `json:"sn"`
		UsedCredits float64 `json:"used_credits"`
		ReportedAt  string  `json:"reported_at"`
		ClientIP    string  `json:"client_ip"`
	}

	var logs []UsageLogEntry
	for rows.Next() {
		var entry UsageLogEntry
		if err := rows.Scan(&entry.SN, &entry.UsedCredits, &entry.ReportedAt, &entry.ClientIP); err != nil {
			continue
		}
		logs = append(logs, entry)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: rows iteration error: %v", err)
	}

	if logs == nil {
		logs = []UsageLogEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records":       logs,
		"total":         total,
		"page":          page,
		"pageSize":      pageSize,
		"totalPages":    totalPages,
		"total_credits": totalCredits,
		"used_credits":  usedCredits,
		"credits_mode":  creditsMode == 1,
	})
}

// handleEmailTemplates handles GET and POST for email templates
func handleEmailTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query("SELECT id, name, subject, body, is_preset, created_at FROM email_templates ORDER BY is_preset DESC, id ASC")
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var templates []EmailTemplate
		for rows.Next() {
			var t EmailTemplate
			var isPreset int
			if err := rows.Scan(&t.ID, &t.Name, &t.Subject, &t.Body, &isPreset, &t.CreatedAt); err != nil {
				continue
			}
			t.IsPreset = isPreset == 1
			templates = append(templates, t)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("database iteration error: %v", err), http.StatusInternalServerError)
			return
		}
		if templates == nil {
			templates = []EmailTemplate{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
		return
	}
	if r.Method == "POST" {
		var t EmailTemplate
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求数据"})
			return
		}
		if t.Name == "" || t.Subject == "" || t.Body == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "模板名称、标题和内容不能为空"})
			return
		}
		if t.ID == 0 {
			// Create new custom template
			result, err := db.Exec("INSERT INTO email_templates (name, subject, body, is_preset, created_at) VALUES (?, ?, ?, 0, datetime('now'))", t.Name, t.Subject, t.Body)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			id, _ := result.LastInsertId()
			t.ID = int(id)
		} else {
			// Update existing template - only allow updating non-preset templates
			var isPreset int
			err := db.QueryRow("SELECT is_preset FROM email_templates WHERE id=?", t.ID).Scan(&isPreset)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "模板不存在"})
				return
			}
			if isPreset == 1 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "预置模板不可修改"})
				return
			}
			_, err = db.Exec("UPDATE email_templates SET name=?, subject=?, body=? WHERE id=? AND is_preset=0", t.Name, t.Subject, t.Body, t.ID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": t.ID})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleDeleteEmailTemplate handles DELETE for email templates
func handleDeleteEmailTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求数据"})
		return
	}
	if req.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "模板 ID 不能为空"})
		return
	}
	// Check if template exists and is not preset
	var isPreset int
	err := db.QueryRow("SELECT is_preset FROM email_templates WHERE id=?", req.ID).Scan(&isPreset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "模板不存在"})
		return
	}
	if isPreset == 1 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "预置模板不可删除"})
		return
	}
	_, err = db.Exec("DELETE FROM email_templates WHERE id=? AND is_preset=0", req.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
// TemplateVars holds the variables available for email template rendering.
type TemplateVars struct {
	ProductName string
	Email       string
	SN          string
}

// renderEmailTemplate renders an email template string by replacing
// {{.ProductName}}, {{.Email}}, {{.SN}} with the provided values.
// Uses simple string replacement to avoid issues with HTML content
// that may contain characters conflicting with Go template syntax.
func renderEmailTemplate(tmplStr string, vars TemplateVars) (string, error) {
	r := strings.NewReplacer(
		"{{.ProductName}}", vars.ProductName,
		"{{.Email}}", vars.Email,
		"{{.SN}}", vars.SN,
	)
	return r.Replace(tmplStr), nil
}

func handleEmailNotifyRecipients(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	productID := query.Get("product_id")
	search := query.Get("search")

	var rows *sql.Rows
	var err error

	if productID != "" {
		pid := 0
		fmt.Sscanf(productID, "%d", &pid)
		rows, err = db.Query("SELECT DISTINCT email FROM email_records WHERE product_id = ?", pid)
	} else if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		rows, err = db.Query("SELECT DISTINCT email FROM email_records WHERE LOWER(email) LIKE ? LIMIT 100", searchPattern)
	} else {
		rows, err = db.Query("SELECT DISTINCT email FROM email_records LIMIT 500")
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			continue
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	if emails == nil {
		emails = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"emails": emails,
		"count":  len(emails),
	})
}

// handleEmailNotifySend creates a new batch send task.
// POST /api/email-notify/send
// Body: {"subject": "...", "body": "...", "emails": ["a@b.com", ...]}
func handleEmailNotifySend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Subject   string   `json:"subject"`
		Body      string   `json:"body"`
		Emails    []string `json:"emails"`
		ProductID *int     `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "无效的请求数据"})
		return
	}

	// Resolve product_id: nil means not specified (-1), otherwise use the value
	taskProductID := -1
	if req.ProductID != nil {
		taskProductID = *req.ProductID
	}

	if req.Subject == "" || req.Body == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "邮件标题和内容不能为空"})
		return
	}

	if len(req.Emails) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "收件人列表不能为空"})
		return
	}

	// Check if another task is already running
	if sendQueue.IsRunning() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "已有发送任务正在执行，请等待完成或取消后再试"})
		return
	}

	// Create the send task and items in a single transaction
	tx, err := db.Begin()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "创建发送任务失败: " + err.Error()})
		return
	}

	result, err := tx.Exec(
		"INSERT INTO email_send_tasks (subject, body, total_count, sent_count, failed_count, status, product_id, created_at) VALUES (?, ?, ?, 0, 0, 'running', ?, datetime('now'))",
		req.Subject, req.Body, len(req.Emails), taskProductID,
	)
	if err != nil {
		tx.Rollback()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "创建发送任务失败: " + err.Error()})
		return
	}

	taskID, _ := result.LastInsertId()

	// Insert all recipients as send_items with status='pending'
	stmt, err := tx.Prepare("INSERT INTO email_send_items (task_id, email, status) VALUES (?, ?, 'pending')")
	if err != nil {
		tx.Rollback()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "创建发送明细失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	for _, email := range req.Emails {
		if _, err := stmt.Exec(taskID, email); err != nil {
			tx.Rollback()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "写入收件人失败: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "提交事务失败: " + err.Error()})
		return
	}

	// Start the send queue
	sendQueue.Start(taskID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"taskId":     taskID,
		"totalCount": len(req.Emails),
		"status":     "running",
	})
}

// handleEmailNotifyProgress returns the progress of a send task.
// GET /api/email-notify/progress/:taskId
func handleEmailNotifyProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract taskId from URL path
	taskIdStr := strings.TrimPrefix(r.URL.Path, "/api/email-notify/progress/")
	if taskIdStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "缺少任务ID"})
		return
	}

	taskID := int64(0)
	fmt.Sscanf(taskIdStr, "%d", &taskID)
	if taskID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "无效的任务ID"})
		return
	}

	// Query task info
	var status string
	var totalCount int
	err := db.QueryRow(
		"SELECT status, total_count FROM email_send_tasks WHERE id=?",
		taskID,
	).Scan(&status, &totalCount)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "任务不存在"})
		return
	}

	// Count all statuses from send_items in a single query for efficiency
	var sentCount, failedCount, pendingCount, cancelledCount int
	countRows, countErr := db.Query(
		"SELECT status, COUNT(*) FROM email_send_items WHERE task_id=? GROUP BY status",
		taskID,
	)
	if countErr == nil {
		defer countRows.Close()
		for countRows.Next() {
			var st string
			var cnt int
			if countRows.Scan(&st, &cnt) == nil {
				switch st {
				case "sent":
					sentCount = cnt
				case "failed":
					failedCount = cnt
				case "pending":
					pendingCount = cnt
				case "cancelled":
					cancelledCount = cnt
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"taskId":    taskID,
		"status":    status,
		"total":     totalCount,
		"sent":      sentCount,
		"failed":    failedCount,
		"pending":   pendingCount,
		"cancelled": cancelledCount,
	})
}

// handleEmailNotifyCancel cancels a running send task.
// POST /api/email-notify/cancel/:taskId
func handleEmailNotifyCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract taskId from URL path
	taskIdStr := strings.TrimPrefix(r.URL.Path, "/api/email-notify/cancel/")
	if taskIdStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "缺少任务ID"})
		return
	}

	taskID := int64(0)
	fmt.Sscanf(taskIdStr, "%d", &taskID)
	if taskID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "无效的任务ID"})
		return
	}

	// Check task exists and is running
	var status string
	err := db.QueryRow("SELECT status FROM email_send_tasks WHERE id=?", taskID).Scan(&status)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "任务不存在"})
		return
	}

	if status != "running" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "任务不在运行状态，无法取消"})
		return
	}

	// Cancel the send queue
	sendQueue.Cancel()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "发送任务已取消",
	})
}

// SendQueue manages rate-limited email sending with a background goroutine.
// It processes email_send_items in batches of up to 5, with 10-second intervals.
type SendQueue struct {
	mu         sync.Mutex
	activeTask *int64         // current active task ID, nil means idle
	cancel     chan struct{}  // signal channel to cancel the current task
}

// global send queue instance
var sendQueue = &SendQueue{}

// Start launches a background goroutine to process the send task identified by taskID.
// It fetches pending items from email_send_items in batches of 5, sends each email,
// and updates the item status accordingly. Batches are separated by 10-second intervals.
func (q *SendQueue) Start(taskID int64) {
	q.mu.Lock()
	if q.activeTask != nil {
		q.mu.Unlock()
		log.Printf("[SendQueue] already running task %d, ignoring Start(%d)", *q.activeTask, taskID)
		return
	}
	id := taskID
	q.activeTask = &id
	q.cancel = make(chan struct{})
	q.mu.Unlock()

	go q.runLoop(taskID)
}

// Cancel stops the currently running send task. All remaining pending items
// are marked as 'cancelled'. Already sent items are not affected.
func (q *SendQueue) Cancel() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.activeTask == nil {
		return
	}

	// Signal the goroutine to stop
	select {
	case <-q.cancel:
		// already closed
	default:
		close(q.cancel)
	}
}

// IsRunning returns true if a send task is currently being processed.
func (q *SendQueue) IsRunning() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.activeTask != nil
}

// runLoop is the core send loop executed in a background goroutine.
func (q *SendQueue) runLoop(taskID int64) {
	defer q.finish(taskID)

	// Recover from panics to ensure task state is always cleaned up
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SendQueue] panic in task %d: %v", taskID, r)
			q.markTaskStatus(taskID, "completed")
		}
	}()

	// Fetch the task's subject, body, and product_id for sending
	var subject, body string
	var taskProductID int
	err := db.QueryRow("SELECT subject, body, COALESCE(product_id, -1) FROM email_send_tasks WHERE id=?", taskID).Scan(&subject, &body, &taskProductID)
	if err != nil {
		log.Printf("[SendQueue] failed to load task %d: %v", taskID, err)
		q.markTaskStatus(taskID, "completed")
		return
	}

	for {
		// Check for cancellation before each batch
		select {
		case <-q.cancel:
			q.cancelPendingItems(taskID)
			q.markTaskStatus(taskID, "cancelled")
			return
		default:
		}

		// Fetch next batch of pending items (max 5)
		rows, err := db.Query(
			"SELECT id, email FROM email_send_items WHERE task_id=? AND status='pending' LIMIT 5",
			taskID,
		)
		if err != nil {
			log.Printf("[SendQueue] query error for task %d: %v", taskID, err)
			break
		}

		type sendItem struct {
			ID    int64
			Email string
		}
		var items []sendItem
		for rows.Next() {
			var item sendItem
			if err := rows.Scan(&item.ID, &item.Email); err != nil {
				continue
			}
			items = append(items, item)
		}
		if rowErr := rows.Err(); rowErr != nil {
			log.Printf("[SendQueue] rows iteration error for task %d: %v", taskID, rowErr)
		}
		rows.Close()

		// No more pending items — we're done
		if len(items) == 0 {
			break
		}

		// Process each item in the batch
		for _, item := range items {
			// Check cancellation between individual sends
			select {
			case <-q.cancel:
				q.cancelPendingItems(taskID)
				q.markTaskStatus(taskID, "cancelled")
				return
			default:
			}

			// Look up SN and product for this recipient to render template variables
			var sn string
			var productID int
			if taskProductID >= 0 {
				// Product explicitly specified by the sender
				productID = taskProductID
				db.QueryRow("SELECT COALESCE(sn,'') FROM email_records WHERE email=? AND product_id=? LIMIT 1", item.Email, productID).Scan(&sn)
			} else {
				// Fallback: look up from email_records
				err := db.QueryRow("SELECT COALESCE(sn,''), COALESCE(product_id,0) FROM email_records WHERE email=? LIMIT 1", item.Email).Scan(&sn, &productID)
				if err != nil {
					sn = ""
					productID = 0
				}
			}
			productName := getProductName(productID)

			vars := TemplateVars{
				ProductName: productName,
				Email:       item.Email,
				SN:          sn,
			}

			renderedSubject, _ := renderEmailTemplate(subject, vars)
			renderedBody, _ := renderEmailTemplate(body, vars)

			err = sendEmail(item.Email, renderedSubject, renderedBody)
			if err != nil {
				// Mark as failed with error message
				db.Exec(
					"UPDATE email_send_items SET status='failed', error=? WHERE id=?",
					err.Error(), item.ID,
				)
				db.Exec(
					"UPDATE email_send_tasks SET failed_count = failed_count + 1 WHERE id=?",
					taskID,
				)
				log.Printf("[SendQueue] task %d: failed to send to %s: %v", taskID, item.Email, err)
			} else {
				// Mark as sent with timestamp
				db.Exec(
					"UPDATE email_send_items SET status='sent', sent_at=datetime('now') WHERE id=?",
					item.ID,
				)
				db.Exec(
					"UPDATE email_send_tasks SET sent_count = sent_count + 1 WHERE id=?",
					taskID,
				)
			}
		}

		// Wait 10 seconds between batches, or exit on cancel
		select {
		case <-q.cancel:
			q.cancelPendingItems(taskID)
			q.markTaskStatus(taskID, "cancelled")
			return
		case <-time.After(10 * time.Second):
		}
	}

	// All items processed — mark task as completed
	q.markTaskStatus(taskID, "completed")
}

// cancelPendingItems marks all remaining pending items for a task as cancelled.
func (q *SendQueue) cancelPendingItems(taskID int64) {
	_, err := db.Exec(
		"UPDATE email_send_items SET status='cancelled' WHERE task_id=? AND status='pending'",
		taskID,
	)
	if err != nil {
		log.Printf("[SendQueue] failed to cancel pending items for task %d: %v", taskID, err)
	}
}

// markTaskStatus updates the task's status and sets completed_at timestamp.
func (q *SendQueue) markTaskStatus(taskID int64, status string) {
	_, err := db.Exec(
		"UPDATE email_send_tasks SET status=?, completed_at=datetime('now') WHERE id=?",
		status, taskID,
	)
	if err != nil {
		log.Printf("[SendQueue] failed to update task %d status to %s: %v", taskID, status, err)
	}
}

// finish clears the active task state so the queue can accept new tasks.
func (q *SendQueue) finish(taskID int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.activeTask != nil && *q.activeTask == taskID {
		q.activeTask = nil
		q.cancel = nil
	}
}

// handleEmailHistory returns a paginated list of email send tasks, ordered by created_at DESC.
// GET /api/email-history?page=1&pageSize=20
func handleEmailHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	page, pageSize := 1, 20
	fmt.Sscanf(query.Get("page"), "%d", &page)
	fmt.Sscanf(query.Get("pageSize"), "%d", &pageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Get total count
	var total int
	db.QueryRow("SELECT COUNT(*) FROM email_send_tasks").Scan(&total)

	// Get paginated tasks ordered by created_at DESC
	rows, err := db.Query(
		"SELECT id, subject, total_count, sent_count, failed_count, status, created_at, completed_at, COALESCE(product_id, -1) FROM email_send_tasks ORDER BY created_at DESC LIMIT ? OFFSET ?",
		pageSize, (page-1)*pageSize,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "查询发送历史失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var tasks []map[string]interface{}
	for rows.Next() {
		var id int64
		var subject, status string
		var totalCount, sentCount, failedCount int
		var createdAt string
		var completedAt sql.NullString
		var productID int
		if err := rows.Scan(&id, &subject, &totalCount, &sentCount, &failedCount, &status, &createdAt, &completedAt, &productID); err != nil {
			continue
		}
		// Resolve template variables in subject for display
		pName := getProductName(productID)
		if productID < 0 {
			pName = getProductName(0)
		}
		subject = strings.ReplaceAll(subject, "{{.ProductName}}", pName)
		task := map[string]interface{}{
			"id":           id,
			"subject":      subject,
			"total_count":  totalCount,
			"sent_count":   sentCount,
			"failed_count": failedCount,
			"status":       status,
			"created_at":   createdAt,
		}
		if completedAt.Valid {
			task["completed_at"] = completedAt.String
		} else {
			task["completed_at"] = nil
		}
		tasks = append(tasks, task)
	}
	if tasks == nil {
		tasks = []map[string]interface{}{}
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":      tasks,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
	})
}

// handleEmailHistoryDetail returns the detail of a specific send task including all recipient statuses.
// GET /api/email-history/<taskId>
func handleEmailHistoryDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract taskId from URL path
	taskIdStr := strings.TrimPrefix(r.URL.Path, "/api/email-history/")
	if taskIdStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "缺少任务ID"})
		return
	}

	taskID := int64(0)
	fmt.Sscanf(taskIdStr, "%d", &taskID)
	if taskID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "无效的任务ID"})
		return
	}

	// Query task info
	var subject, status, createdAt string
	var totalCount, sentCount, failedCount int
	var completedAt sql.NullString
	var productID int
	err := db.QueryRow(
		"SELECT subject, total_count, sent_count, failed_count, status, created_at, completed_at, COALESCE(product_id, -1) FROM email_send_tasks WHERE id=?",
		taskID,
	).Scan(&subject, &totalCount, &sentCount, &failedCount, &status, &createdAt, &completedAt, &productID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "任务不存在"})
		return
	}

	// Resolve template variables in subject for display
	pName := getProductName(productID)
	if productID < 0 {
		pName = getProductName(0)
	}
	subject = strings.ReplaceAll(subject, "{{.ProductName}}", pName)

	// Query all send items for this task
	rows, err := db.Query(
		"SELECT id, email, status, error, sent_at FROM email_send_items WHERE task_id=? ORDER BY id ASC",
		taskID,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "查询发送明细失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var itemID int64
		var email, itemStatus string
		var itemError sql.NullString
		var sentAt sql.NullString
		if err := rows.Scan(&itemID, &email, &itemStatus, &itemError, &sentAt); err != nil {
			continue
		}
		item := map[string]interface{}{
			"id":     itemID,
			"email":  email,
			"status": itemStatus,
		}
		if itemError.Valid {
			item["error"] = itemError.String
		} else {
			item["error"] = nil
		}
		if sentAt.Valid {
			item["sent_at"] = sentAt.String
		} else {
			item["sent_at"] = nil
		}
		items = append(items, item)
	}
	if items == nil {
		items = []map[string]interface{}{}
	}

	taskInfo := map[string]interface{}{
		"id":           taskID,
		"subject":      subject,
		"total_count":  totalCount,
		"sent_count":   sentCount,
		"failed_count": failedCount,
		"status":       status,
		"created_at":   createdAt,
	}
	if completedAt.Valid {
		taskInfo["completed_at"] = completedAt.String
	} else {
		taskInfo["completed_at"] = nil
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task":  taskInfo,
		"items": items,
	})
}
