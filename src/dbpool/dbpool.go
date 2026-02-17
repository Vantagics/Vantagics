// Package dbpool provides a unified database connection manager that abstracts
// away engine-specific details (DuckDB, SQLite, MySQL, etc.) and handles
// retry logic, connection pooling, and file-lock contention on Windows.
//
// All code that needs a *sql.DB should go through DBManager instead of calling
// sql.Open directly. This gives us a single place to:
//   - switch between DuckDB and SQLite
//   - add retry/backoff for file-lock contention
//   - enforce connection pool settings
//   - handle WAL checkpoint for DuckDB
package dbpool

import (
	"database/sql"
	"fmt"
)

// Engine identifies the database engine to use.
type Engine string

const (
	EngineDuckDB Engine = "duckdb"
	EngineSQLite Engine = "sqlite"
	EngineMySQL  Engine = "mysql"
)

// AccessMode controls whether the connection is read-only or read-write.
type AccessMode int

const (
	ModeReadWrite AccessMode = iota
	ModeReadOnly
)

// OpenOptions configures how a database connection is opened.
type OpenOptions struct {
	// Engine to use. Defaults to EngineDuckDB if empty.
	Engine Engine
	// Path is the file path for file-based engines (DuckDB, SQLite).
	// For MySQL, this is the DSN string.
	Path string
	// Mode controls read-only vs read-write access.
	Mode AccessMode
	// MaxRetries overrides the default retry count (0 = use default).
	MaxRetries int
	// RetryBaseMs overrides the base retry interval in milliseconds (0 = use default).
	RetryBaseMs int
}

// Logger is a simple logging function signature.
type Logger func(string)

// DBManager is the central connection manager.
type DBManager struct {
	logger Logger
	engine Engine // default engine for the application
}

// New creates a new DBManager with the given default engine and logger.
func New(defaultEngine Engine, logger Logger) *DBManager {
	if logger == nil {
		logger = func(string) {}
	}
	return &DBManager{
		engine: defaultEngine,
		logger: logger,
	}
}

// DefaultEngine returns the manager's default engine.
func (m *DBManager) DefaultEngine() Engine {
	return m.engine
}

// Open opens a database connection with the given options.
// It applies retry logic for file-based engines to handle lock contention.
func (m *DBManager) Open(opts OpenOptions) (*sql.DB, error) {
	eng := opts.Engine
	if eng == "" {
		eng = m.engine
	}

	switch eng {
	case EngineDuckDB:
		return m.openDuckDB(opts)
	case EngineSQLite:
		return m.openSQLite(opts)
	case EngineMySQL:
		return m.openMySQL(opts)
	default:
		return nil, fmt.Errorf("dbpool: unsupported engine %q", eng)
	}
}

// OpenReadOnly is a convenience wrapper for read-only access.
func (m *DBManager) OpenReadOnly(path string) (*sql.DB, error) {
	return m.Open(OpenOptions{Path: path, Mode: ModeReadOnly})
}

// OpenWritable is a convenience wrapper for read-write access.
func (m *DBManager) OpenWritable(path string) (*sql.DB, error) {
	return m.Open(OpenOptions{Path: path, Mode: ModeReadWrite})
}

// OpenNew opens a brand-new database file (for imports). No retry needed since
// the file doesn't exist yet and can't be locked.
func (m *DBManager) OpenNew(path string) (*sql.DB, error) {
	return m.Open(OpenOptions{Path: path, Mode: ModeReadWrite, MaxRetries: 1})
}

// configurePool sets connection pool parameters that ensure file locks are
// released immediately on Close(). This is critical on Windows.
func configurePool(db *sql.DB) {
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(1)
}

// retryParams returns (maxRetries, baseMs) from opts or defaults.
func retryParams(opts OpenOptions) (int, int) {
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 8
	}
	baseMs := opts.RetryBaseMs
	if baseMs <= 0 {
		baseMs = 400
	}
	return maxRetries, baseMs
}
