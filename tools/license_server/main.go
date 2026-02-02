package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
	"license_server/templates"
)

const (
	// DBPassword is the encryption password for the SQLite database.
	// SECURITY WARNING: In production, this should be loaded from environment
	// variables or a secure configuration file, not hardcoded.
	// Example: DBPassword = os.Getenv("LICENSE_DB_PASSWORD")
	DBPassword    = "sunion123!"
	
	// DefaultAdminPassword is the initial admin password.
	// SECURITY WARNING: Users should be prompted to change this on first login.
	// This default is only for initial setup convenience.
	DefaultAdminPassword = "sunion123"
)

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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ProductType holds product type information for license categorization
type ProductType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
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
	Description    string    `json:"description"`
	IsActive       bool      `json:"is_active"`
	UsageCount     int       `json:"usage_count"`
	LastUsedAt     time.Time `json:"last_used_at"`
	DailyAnalysis  int       `json:"daily_analysis"`  // Daily analysis limit, 0 = unlimited
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
	LLMType         string `json:"llm_type"`
	LLMBaseURL      string `json:"llm_base_url"`
	LLMAPIKey       string `json:"llm_api_key"`
	LLMModel        string `json:"llm_model"`
	LLMStartDate    string `json:"llm_start_date"`
	LLMEndDate      string `json:"llm_end_date"`
	SearchType      string `json:"search_type"`
	SearchAPIKey    string `json:"search_api_key"`
	SearchStartDate string `json:"search_start_date"`
	SearchEndDate   string `json:"search_end_date"`
	ExpiresAt       string `json:"expires_at"`
	ActivatedAt     string `json:"activated_at"`
	DailyAnalysis   int    `json:"daily_analysis"`   // Daily analysis limit
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
)

func main() {
	execPath, _ := os.Executable()
	dbPath = filepath.Join(filepath.Dir(execPath), "license_server.db")
	
	initDB()
	loadPorts()
	loadSSLConfig()
	
	go startManageServer()
	startAuthServer()
}

func initDB() {
	var err error
	dsn := fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096", dbPath, DBPassword)
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	
	// Enable WAL mode for better concurrent read/write performance
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		log.Printf("Warning: Failed to enable WAL mode: %v", err)
	} else {
		log.Println("SQLite WAL mode enabled")
	}
	
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
			description TEXT
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
			email TEXT UNIQUE,
			sn TEXT,
			ip TEXT,
			created_at DATETIME
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
	`)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
	
	// Migration: Add new columns if they don't exist
	db.Exec("ALTER TABLE licenses ADD COLUMN llm_group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE licenses ADD COLUMN search_group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE licenses ADD COLUMN license_group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE licenses ADD COLUMN product_id INTEGER DEFAULT 0")
	db.Exec("ALTER TABLE llm_configs ADD COLUMN group_id TEXT DEFAULT ''")
	db.Exec("ALTER TABLE search_configs ADD COLUMN group_id TEXT DEFAULT ''")
	// Migration: Create email_conditions table if not exists (for existing databases)
	db.Exec(`CREATE TABLE IF NOT EXISTS email_conditions (
		pattern TEXT PRIMARY KEY,
		created_at DATETIME,
		llm_group_id TEXT DEFAULT '',
		search_group_id TEXT DEFAULT ''
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
	
	headers := make(map[string]string)
	headers["From"] = fromHeader
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	
	// Build message
	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
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

// sendSNEmail sends the serial number to the user's email
func sendSNEmail(email, sn string, expiresAt time.Time) error {
	subject := "VantageData - Your Serial Number"
	
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
            <h1>🎉 VantageData - Your Serial Number</h1>
        </div>
        <div class="content">
            <div class="no-reply">⚠️ This is an automated message. Please do not reply.</div>
            <p style="margin:0 0 10px 0;">Thank you for requesting a VantageData serial number:</p>
            <div class="sn-box">
                <div class="sn">%s</div>
            </div>
            <div class="info">
                <p><strong>📅 Valid until:</strong> %s (%d days)</p>
                <p><strong>💡 How to use:</strong> Open VantageData → Select Commercial Mode → Enter serial number → Activate</p>
            </div>
            <p class="help">Questions? Visit <a href="https://vantagedata.chat" style="color:#667eea;">vantagedata.chat</a></p>
        </div>
        <div class="footer">
            <p>© VantageData - Intelligent Data Analytics Platform</p>
        </div>
    </div>
</body>
</html>
`, sn, expiryDate, daysLeft)
	
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

func generateSN() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use math/rand with time seed if crypto/rand fails
		log.Printf("Warning: crypto/rand failed: %v, using fallback", err)
		mrand.Seed(time.Now().UnixNano())
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
		// Fallback: use math/rand with time seed if crypto/rand fails
		log.Printf("Warning: crypto/rand failed: %v, using fallback", err)
		mrand.Seed(time.Now().UnixNano())
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
	mux.HandleFunc("/api/product-types", authMiddleware(handleProductTypes))
	mux.HandleFunc("/api/password", authMiddleware(handleChangePassword))
	mux.HandleFunc("/api/username", authMiddleware(handleChangeUsername))
	mux.HandleFunc("/api/ports", authMiddleware(handleChangePorts))
	mux.HandleFunc("/api/ssl", authMiddleware(handleSSLConfig))
	mux.HandleFunc("/api/smtp", authMiddleware(handleSMTPConfig))
	mux.HandleFunc("/api/smtp/test", authMiddleware(handleSMTPTest))
	mux.HandleFunc("/api/settings/request-limits", authMiddleware(handleRequestLimits))
	mux.HandleFunc("/api/email-records", authMiddleware(handleEmailRecords))
	mux.HandleFunc("/api/email-records/update", authMiddleware(handleUpdateEmailRecord))
	mux.HandleFunc("/api/email-filter", authMiddleware(handleEmailFilter))
	mux.HandleFunc("/api/whitelist", authMiddleware(handleWhitelist))
	mux.HandleFunc("/api/blacklist", authMiddleware(handleBlacklist))
	mux.HandleFunc("/api/conditions", authMiddleware(handleConditions))

	addr := fmt.Sprintf(":%d", managePort)
	if useSSL && sslCert != "" && sslKey != "" {
		log.Printf("Management server starting on %s (HTTPS)", addr)
		if err := http.ListenAndServeTLS(addr, sslCert, sslKey, mux); err != nil {
			log.Fatalf("Management server failed: %v", err)
		}
	} else {
		log.Printf("Management server starting on %s (HTTP)", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Management server failed: %v", err)
		}
	}
}

var sessions = make(map[string]time.Time)
var sessionLock sync.RWMutex

// Captcha storage
var captchas = make(map[string]string)
var captchaLock sync.RWMutex

func createSession() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use math/rand with time seed if crypto/rand fails
		log.Printf("Warning: crypto/rand failed for session token: %v, using fallback", err)
		mrand.Seed(time.Now().UnixNano())
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
	mrand.Seed(time.Now().UnixNano())
	
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
		// Fallback: use math/rand with time seed if crypto/rand fails
		log.Printf("Warning: crypto/rand failed for captcha ID: %v, using fallback", err)
		mrand.Seed(time.Now().UnixNano())
		for i := range idBytes {
			idBytes[i] = byte(mrand.Intn(256))
		}
	}
	captchaID := hex.EncodeToString(idBytes)
	
	// Store captcha answer with 5 minute expiry
	answer := fmt.Sprintf("%d", result)
	captchaLock.Lock()
	captchas[captchaID] = answer
	captchaLock.Unlock()
	
	// Clean old captchas periodically
	go func() {
		time.Sleep(5 * time.Minute)
		captchaLock.Lock()
		delete(captchas, captchaID)
		captchaLock.Unlock()
	}()
	
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
	correctAnswer, exists := captchas[captchaID]
	captchaLock.RUnlock()
	
	if !exists {
		return false
	}
	
	// Delete captcha after use
	captchaLock.Lock()
	delete(captchas, captchaID)
	captchaLock.Unlock()
	
	return strings.TrimSpace(answer) == correctAnswer
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil || !validateSession(cookie.Value) {
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

func handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
	if username != validUsername || password != validPassword {
		newCaptchaID, newCaptchaImage := generateCaptcha()
		tmpl := template.Must(template.New("login").Parse(templates.LoginHTML))
		tmpl.Execute(w, map[string]interface{}{
			"Error":        "用户名或密码错误",
			"CaptchaID":    newCaptchaID,
			"CaptchaImage": newCaptchaImage,
			"ManagePort":   managePort,
			"AuthPort":     authPort,
		})
		return
	}
	token := createSession()
	http.SetCookie(w, &http.Cookie{Name: "session", Value: token, Path: "/", HttpOnly: true, MaxAge: 86400})
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ManagePort int
		AuthPort   int
		Username   string
	}{ManagePort: managePort, AuthPort: authPort, Username: getSetting("admin_username")}
	tmpl := template.Must(template.New("dashboard").Parse(templates.GetDashboardHTML()))
	tmpl.Execute(w, data)
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
		rows.Scan(&l.SN, &l.CreatedAt, &l.ExpiresAt, &l.Description, &l.IsActive, &l.UsageCount, &lastUsed, &l.DailyAnalysis, &l.LicenseGroupID, &l.LLMGroupID, &l.SearchGroupID)
		if lastUsed.Valid {
			l.LastUsedAt = lastUsed.Time
		}
		licenses[l.SN] = l
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
		Description    string `json:"description"`
		Days           int    `json:"days"`
		DailyAnalysis  int    `json:"daily_analysis"`
		LLMGroupID     string `json:"llm_group_id"`
		LicenseGroupID string `json:"license_group_id"`
		SearchGroupID  string `json:"search_group_id"`
		ProductID      int    `json:"product_id"`
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
	sn := generateSN()
	now := time.Now()
	expires := now.AddDate(0, 0, req.Days)
	
	_, err := db.Exec("INSERT INTO licenses (sn, created_at, expires_at, description, is_active, daily_analysis, license_group_id, llm_group_id, search_group_id, product_id) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?)",
		sn, now, expires, req.Description, req.DailyAnalysis, req.LicenseGroupID, req.LLMGroupID, req.SearchGroupID, req.ProductID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	license := License{SN: sn, CreatedAt: now, ExpiresAt: expires, Description: req.Description, IsActive: true, DailyAnalysis: req.DailyAnalysis, LLMGroupID: req.LLMGroupID, SearchGroupID: req.SearchGroupID, ProductID: req.ProductID}
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
	json.NewDecoder(r.Body).Decode(&req)
	db.Exec("DELETE FROM licenses WHERE sn=?", req.SN)
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
	json.NewDecoder(r.Body).Decode(&req)
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
		Description    string `json:"description"`
		Days           int    `json:"days"`
		Count          int    `json:"count"`
		DailyAnalysis  int    `json:"daily_analysis"`
		LLMGroupID     string `json:"llm_group_id"`
		LicenseGroupID string `json:"license_group_id"`
		SearchGroupID  string `json:"search_group_id"`
		ProductID      int    `json:"product_id"`
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
	
	now := time.Now()
	expires := now.AddDate(0, 0, req.Days)
	var created []string
	
	for i := 0; i < req.Count; i++ {
		sn := generateSN()
		_, err := db.Exec("INSERT INTO licenses (sn, created_at, expires_at, description, is_active, daily_analysis, license_group_id, llm_group_id, search_group_id, product_id) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?)",
			sn, now, expires, req.Description, req.DailyAnalysis, req.LicenseGroupID, req.LLMGroupID, req.SearchGroupID, req.ProductID)
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
	rows, _ := db.Query(`SELECT sn FROM licenses 
		WHERE is_active = 0 
		AND sn NOT IN (SELECT sn FROM email_records)`)
	var sns []string
	for rows.Next() {
		var sn string
		rows.Scan(&sn)
		sns = append(sns, sn)
	}
	rows.Close()
	
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
	selectQuery := `SELECT sn, created_at, expires_at, description, is_active, usage_count, last_used_at, COALESCE(daily_analysis, 20), COALESCE(license_group_id, ''), COALESCE(llm_group_id, ''), COALESCE(search_group_id, ''), COALESCE(product_id, 0) 
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
		rows.Scan(&l.SN, &l.CreatedAt, &l.ExpiresAt, &l.Description, &l.IsActive, &l.UsageCount, &lastUsed, &l.DailyAnalysis, &l.LicenseGroupID, &l.LLMGroupID, &l.SearchGroupID, &l.ProductID)
		if lastUsed.Valid {
			l.LastUsedAt = lastUsed.Time
		}
		licenses = append(licenses, l)
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
		rows, _ := db.Query("SELECT id, name, type, base_url, api_key, model, is_active, start_date, end_date, COALESCE(group_id, '') FROM llm_configs")
		defer rows.Close()
		var configs []LLMConfig
		for rows.Next() {
			var c LLMConfig
			rows.Scan(&c.ID, &c.Name, &c.Type, &c.BaseURL, &c.APIKey, &c.Model, &c.IsActive, &c.StartDate, &c.EndDate, &c.GroupID)
			configs = append(configs, c)
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
		rows, _ := db.Query("SELECT id, name, type, api_key, is_active, start_date, end_date, COALESCE(group_id, '') FROM search_configs")
		defer rows.Close()
		var configs []SearchConfig
		for rows.Next() {
			var c SearchConfig
			rows.Scan(&c.ID, &c.Name, &c.Type, &c.APIKey, &c.IsActive, &c.StartDate, &c.EndDate, &c.GroupID)
			configs = append(configs, c)
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
		rows, _ := db.Query("SELECT id, name, description FROM llm_groups ORDER BY name")
		defer rows.Close()
		var groups []LLMGroup
		for rows.Next() {
			var g LLMGroup
			rows.Scan(&g.ID, &g.Name, &g.Description)
			groups = append(groups, g)
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
		rows, _ := db.Query("SELECT id, name, description FROM search_groups ORDER BY name")
		defer rows.Close()
		var groups []SearchGroup
		for rows.Next() {
			var g SearchGroup
			rows.Scan(&g.ID, &g.Name, &g.Description)
			groups = append(groups, g)
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
		rows, _ := db.Query("SELECT id, name, description FROM license_groups ORDER BY name")
		defer rows.Close()
		var groups []LicenseGroup
		for rows.Next() {
			var g LicenseGroup
			rows.Scan(&g.ID, &g.Name, &g.Description)
			groups = append(groups, g)
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
		db.Exec("INSERT OR REPLACE INTO license_groups (id, name, description) VALUES (?, ?, ?)", g.ID, g.Name, g.Description)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": g.ID})
		return
	}
	if r.Method == "DELETE" {
		var req struct{ ID string `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req)
		
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

// handleProductTypes manages product types for license categorization
func handleProductTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, _ := db.Query("SELECT id, name, description FROM product_types ORDER BY id")
		defer rows.Close()
		var products []ProductType
		for rows.Next() {
			var p ProductType
			rows.Scan(&p.ID, &p.Name, &p.Description)
			products = append(products, p)
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
	if req.OldPassword != getSetting("admin_password") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "旧密码错误"})
		return
	}
	setSetting("admin_password", req.NewPassword)
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
	subject := "VantageData SMTP 测试邮件"
	htmlBody := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: sans-serif; padding: 20px;">
    <h2>🎉 SMTP 配置测试成功！</h2>
    <p>如果您收到这封邮件，说明 SMTP 配置正确。</p>
    <p style="color: #666; font-size: 12px;">此邮件由 VantageData 授权服务器发送。</p>
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
	page, pageSize := 1, 20
	fmt.Sscanf(query.Get("page"), "%d", &page)
	fmt.Sscanf(query.Get("pageSize"), "%d", &pageSize)
	if page < 1 { page = 1 }
	if pageSize < 1 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	
	var total int
	var rows *sql.Rows
	var err error
	
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		db.QueryRow("SELECT COUNT(*) FROM email_records WHERE LOWER(email) LIKE ? OR LOWER(sn) LIKE ?", 
			searchPattern, searchPattern).Scan(&total)
		rows, err = db.Query(`SELECT id, email, sn, ip, created_at FROM email_records 
			WHERE LOWER(email) LIKE ? OR LOWER(sn) LIKE ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			searchPattern, searchPattern, pageSize, (page-1)*pageSize)
	} else {
		db.QueryRow("SELECT COUNT(*) FROM email_records").Scan(&total)
		rows, err = db.Query("SELECT id, email, sn, ip, created_at FROM email_records ORDER BY created_at DESC LIMIT ? OFFSET ?",
			pageSize, (page-1)*pageSize)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var records []EmailRecord
	for rows.Next() {
		var r EmailRecord
		rows.Scan(&r.ID, &r.Email, &r.SN, &r.IP, &r.CreatedAt)
		records = append(records, r)
	}
	
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 { totalPages = 1 }
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records": records, "total": total, "page": page, "pageSize": pageSize, "totalPages": totalPages,
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
// Returns: allowed, reason, llmGroupID, searchGroupID
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
		rows, _ := db.Query("SELECT pattern FROM email_blacklist")
		defer rows.Close()
		for rows.Next() {
			var pattern string
			rows.Scan(&pattern)
			if matchEmailPattern(email, pattern) {
				return false, CodeEmailBlacklisted, "您的邮箱已被限制申请", "", ""
			}
		}
	}
	
	// Step 2: If whitelist is enabled, must match whitelist
	if whitelistEnabled {
		rows, _ := db.Query("SELECT pattern FROM email_whitelist")
		defer rows.Close()
		matched := false
		for rows.Next() {
			var pattern string
			rows.Scan(&pattern)
			if matchEmailPattern(email, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			return false, CodeEmailNotWhitelisted, "您的邮箱不在白名单中", "", ""
		}
	}
	
	// Step 3: Check conditions list for group bindings (only if enabled)
	if conditionsEnabled {
		rows, _ := db.Query("SELECT pattern, COALESCE(llm_group_id, ''), COALESCE(search_group_id, '') FROM email_conditions")
		defer rows.Close()
		for rows.Next() {
			var pattern, llmGroupID, searchGroupID string
			rows.Scan(&pattern, &llmGroupID, &searchGroupID)
			if matchEmailPattern(email, pattern) {
				return true, "", "", llmGroupID, searchGroupID // Condition match = allow with group bindings
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

// ============ Auth Server ============

func startAuthServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/activate", handleActivate)
	mux.HandleFunc("/request-sn", handleRequestSN)
	mux.HandleFunc("/health", handleHealth)

	addr := fmt.Sprintf(":%d", authPort)
	if useSSL && sslCert != "" && sslKey != "" {
		log.Printf("Auth server starting on %s (HTTPS)", addr)
		if err := http.ListenAndServeTLS(addr, sslCert, sslKey, mux); err != nil {
			log.Fatalf("Auth server failed: %v", err)
		}
	} else {
		log.Printf("Auth server starting on %s (HTTP)", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Auth server failed: %v", err)
		}
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
	err := db.QueryRow("SELECT sn, created_at, expires_at, description, is_active, usage_count, last_used_at, COALESCE(daily_analysis, 20), COALESCE(license_group_id, ''), COALESCE(llm_group_id, ''), COALESCE(search_group_id, '') FROM licenses WHERE sn=?", sn).
		Scan(&license.SN, &license.CreatedAt, &license.ExpiresAt, &license.Description, &license.IsActive, &license.UsageCount, &lastUsed, &license.DailyAnalysis, &license.LicenseGroupID, &license.LLMGroupID, &license.SearchGroupID)
	
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
	if time.Now().After(license.ExpiresAt) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ActivationResponse{Success: false, Code: CodeSNExpired, Message: "序列号已过期"})
		return
	}
	
	// Update usage
	db.Exec("UPDATE licenses SET usage_count=usage_count+1, last_used_at=? WHERE sn=?", time.Now(), sn)
	
	// Get best LLM config for the license's group (or all if no group specified)
	today := time.Now().Format("2006-01-02")
	var bestLLM *LLMConfig
	var llmQuery string
	var llmArgs []interface{}
	if license.LLMGroupID != "" {
		llmQuery = "SELECT id, name, type, base_url, api_key, model, is_active, start_date, end_date, COALESCE(group_id, '') FROM llm_configs WHERE group_id=?"
		llmArgs = []interface{}{license.LLMGroupID}
	} else {
		llmQuery = "SELECT id, name, type, base_url, api_key, model, is_active, start_date, end_date, COALESCE(group_id, '') FROM llm_configs"
	}
	rows, _ := db.Query(llmQuery, llmArgs...)
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
	rows.Close()
	
	// Get best Search config for the license's group (or all if no group specified)
	var bestSearch *SearchConfig
	var searchQuery string
	var searchArgs []interface{}
	if license.SearchGroupID != "" {
		searchQuery = "SELECT id, name, type, api_key, is_active, start_date, end_date, COALESCE(group_id, '') FROM search_configs WHERE group_id=?"
		searchArgs = []interface{}{license.SearchGroupID}
	} else {
		searchQuery = "SELECT id, name, type, api_key, is_active, start_date, end_date, COALESCE(group_id, '') FROM search_configs"
	}
	rows, _ = db.Query(searchQuery, searchArgs...)
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
	rows.Close()
	
	// Build activation data
	activationData := ActivationData{
		ExpiresAt:     license.ExpiresAt.Format(time.RFC3339),
		ActivatedAt:   time.Now().Format(time.RFC3339),
		DailyAnalysis: license.DailyAnalysis,
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
	
	// Check if email already has SN for this product
	var existingSN string
	var existingProductID int
	if err := db.QueryRow(`SELECT e.sn, COALESCE(l.product_id, 0) FROM email_records e 
		LEFT JOIN licenses l ON e.sn = l.sn WHERE e.email=?`, email).Scan(&existingSN, &existingProductID); err == nil {
		// Check if the SN still exists in licenses table
		var snExists int
		db.QueryRow("SELECT COUNT(*) FROM licenses WHERE sn=?", existingSN).Scan(&snExists)
		if snExists > 0 {
			// If same product, return existing SN; if different product, allow new request
			if existingProductID == productID {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeEmailAlreadyUsed, Message: "您已申请过序列号", SN: existingSN})
				return
			}
			// Different product, continue to allocate new SN
			log.Printf("[REQUEST-SN] Email %s has SN for product %d, requesting for product %d", email, existingProductID, productID)
		} else {
			// SN was deleted, remove the old email record and continue to generate new SN
			db.Exec("DELETE FROM email_records WHERE email=?", email)
			log.Printf("[REQUEST-SN] Old SN %s for email %s was deleted, generating new one", existingSN, email)
		}
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
	// 5. Is active and not expired
	// 6. Has not been used (usage_count = 0)
	now := time.Now()
	var sn string
	var query string
	var args []interface{}
	
	// Base condition: product_id must match
	productCondition := "(product_id IS NULL OR product_id = 0)"
	if productID > 0 {
		productCondition = "product_id = ?"
	}
	
	if llmGroupID != "" && searchGroupID != "" {
		if productID > 0 {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + ` AND llm_group_id = ? AND search_group_id = ?
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{productID, llmGroupID, searchGroupID, now}
		} else {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + ` AND llm_group_id = ? AND search_group_id = ?
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{llmGroupID, searchGroupID, now}
		}
	} else if llmGroupID != "" {
		if productID > 0 {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + ` AND llm_group_id = ?
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{productID, llmGroupID, now}
		} else {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + ` AND llm_group_id = ?
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{llmGroupID, now}
		}
	} else if searchGroupID != "" {
		if productID > 0 {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + ` AND search_group_id = ?
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{productID, searchGroupID, now}
		} else {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + ` AND search_group_id = ?
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{searchGroupID, now}
		}
	} else {
		// No group specified, find any available SN with matching product_id
		if productID > 0 {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + `
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{productID, now}
		} else {
			query = `SELECT sn FROM licenses 
				WHERE ` + productCondition + `
				AND is_active = 1 AND expires_at > ? AND usage_count = 0
				AND sn NOT IN (SELECT sn FROM email_records)
				ORDER BY created_at ASC LIMIT 1`
			args = []interface{}{now}
		}
	}
	
	err := db.QueryRow(query, args...).Scan(&sn)
	if err != nil {
		log.Printf("[REQUEST-SN] No available SN found for email %s (Product: %d, LLM Group: %s, Search Group: %s): %v", email, productID, llmGroupID, searchGroupID, err)
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
	
	// Bind the SN to the email
	db.Exec("INSERT INTO email_records (email, sn, ip, created_at) VALUES (?, ?, ?, ?)", email, sn, clientIP, now)
	
	// Update the license description to include the email
	db.Exec("UPDATE licenses SET description = ? WHERE sn = ?", fmt.Sprintf("Email申请: %s", email), sn)
	
	log.Printf("[REQUEST-SN] SN allocated for email %s from IP %s: %s (Product: %d, LLM Group: %s, Search Group: %s)", email, clientIP, sn, productID, llmGroupID, searchGroupID)
	
	// Get the expiry date of the allocated SN
	var expiresAt time.Time
	db.QueryRow("SELECT expires_at FROM licenses WHERE sn = ?", sn).Scan(&expiresAt)
	daysLeft := int(expiresAt.Sub(now).Hours() / 24)
	
	// Send email with SN (async, don't block response)
	go func() {
		if err := sendSNEmail(email, sn, expiresAt); err != nil {
			log.Printf("[EMAIL] Failed to send SN email to %s: %v", email, err)
		} else {
			log.Printf("[EMAIL] SN email sent successfully to %s", email)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RequestSNResponse{Success: true, Code: CodeSuccess, Message: fmt.Sprintf("序列号分配成功，有效期还剩 %d 天", daysLeft), SN: sn})
}
