package dbpool

import (
	"database/sql"
	"fmt"
	"time"
)

// sqliteDriverName caches the detected SQLite driver name so we only probe once.
var sqliteDriverName string

// detectSQLiteDriver probes which SQLite driver is registered.
// sql.Open is lazy so it won't fail for an unknown driver until Ping.
// We try each known driver name with an in-memory DB to find the right one.
func detectSQLiteDriver() string {
	if sqliteDriverName != "" {
		return sqliteDriverName
	}
	for _, name := range []string{"sqlite3", "sqlite"} {
		db, err := sql.Open(name, ":memory:")
		if err != nil {
			continue
		}
		if err := db.Ping(); err == nil {
			db.Close()
			sqliteDriverName = name
			return name
		}
		db.Close()
	}
	return ""
}

// openSQLite opens a SQLite database with retry logic.
// SQLite uses WAL mode by default for better concurrency, but still needs
// retry logic for SQLITE_BUSY on Windows.
//
// NOTE: The caller's application must import a SQLite driver
// (e.g., _ "modernc.org/sqlite" or _ "github.com/mattn/go-sqlite3").
func (m *DBManager) openSQLite(opts OpenOptions) (*sql.DB, error) {
	driverName := detectSQLiteDriver()
	if driverName == "" {
		return nil, fmt.Errorf("dbpool: no SQLite driver registered (import modernc.org/sqlite or github.com/mattn/go-sqlite3)")
	}

	maxRetries, baseMs := retryParams(opts)

	connStr := opts.Path
	params := "?_journal_mode=WAL&_busy_timeout=5000"
	if opts.Mode == ModeReadOnly {
		params += "&mode=ro"
	}
	connStr += params

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		db, err := sql.Open(driverName, connStr)
		if err != nil {
			lastErr = err
			m.logger(fmt.Sprintf("[dbpool] SQLite open attempt %d/%d failed: %v", i+1, maxRetries, err))
			if maxRetries > 1 {
				time.Sleep(time.Duration(baseMs*(i+1)) * time.Millisecond)
			}
			continue
		}

		configurePool(db)

		if err := db.Ping(); err != nil {
			db.Close()
			lastErr = err
			m.logger(fmt.Sprintf("[dbpool] SQLite ping attempt %d/%d failed: %v", i+1, maxRetries, err))
			if maxRetries > 1 {
				time.Sleep(time.Duration(baseMs*(i+1)) * time.Millisecond)
			}
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("dbpool: failed to open SQLite %q after %d retries: %w", opts.Path, maxRetries, lastErr)
}
