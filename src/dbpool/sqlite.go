package dbpool

import (
	"database/sql"
	"fmt"
	"time"
)

// openSQLite opens a SQLite database with retry logic.
// SQLite uses WAL mode by default for better concurrency, but still needs
// retry logic for SQLITE_BUSY on Windows.
//
// NOTE: To use SQLite, the application must import a SQLite driver
// (e.g., _ "modernc.org/sqlite" or _ "github.com/mattn/go-sqlite3")
// and register it as "sqlite3" or "sqlite".
func (m *DBManager) openSQLite(opts OpenOptions) (*sql.DB, error) {
	maxRetries, baseMs := retryParams(opts)

	connStr := opts.Path
	params := "?_journal_mode=WAL&_busy_timeout=5000"
	if opts.Mode == ModeReadOnly {
		params += "&mode=ro"
	}
	connStr += params

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		// Try "sqlite3" first (mattn driver), then "sqlite" (modernc driver)
		db, err := sql.Open("sqlite3", connStr)
		if err != nil {
			db, err = sql.Open("sqlite", connStr)
		}
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
