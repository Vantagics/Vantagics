package dbpool

import (
	"database/sql"
	"fmt"
	"time"
)

// openDuckDB opens a DuckDB database with retry logic and WAL handling.
// NOTE: The caller's application must import the DuckDB driver
// (e.g., _ "github.com/marcboeker/go-duckdb").
func (m *DBManager) openDuckDB(opts OpenOptions) (*sql.DB, error) {
	maxRetries, baseMs := retryParams(opts)

	connStr := opts.Path
	if opts.Mode == ModeReadOnly {
		connStr += "?access_mode=read_only"
	}

	walCheckpointed := false
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		db, err := sql.Open("duckdb", connStr)
		if err == nil {
			configurePool(db)
			err = db.Ping()
			if err != nil {
				db.Close()
			}
		}

		if err != nil {
			lastErr = err
			m.logger(fmt.Sprintf("[dbpool] DuckDB attempt %d/%d failed: %v", i+1, maxRetries, err))

			// For read-only mode, try checkpointing the WAL on first failure
			if opts.Mode == ModeReadOnly && !walCheckpointed {
				walCheckpointed = true
				m.checkpointWAL(opts.Path)
			}

			if maxRetries > 1 {
				time.Sleep(time.Duration(baseMs*(i+1)) * time.Millisecond)
			}
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("dbpool: failed to open DuckDB %q after %d retries: %w", opts.Path, maxRetries, lastErr)
}

// checkpointWAL attempts to flush the DuckDB WAL file by opening in write mode
// and executing CHECKPOINT. This is needed because DuckDB cannot open a database
// with an un-checkpointed WAL in read-only mode.
func (m *DBManager) checkpointWAL(dbPath string) {
	m.logger(fmt.Sprintf("[dbpool] Attempting WAL checkpoint for: %s", dbPath))
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		m.logger(fmt.Sprintf("[dbpool] WAL checkpoint: failed to open in write mode: %v", err))
		return
	}
	configurePool(db)
	defer db.Close()

	if _, err := db.Exec("CHECKPOINT"); err != nil {
		m.logger(fmt.Sprintf("[dbpool] WAL checkpoint failed: %v", err))
	} else {
		m.logger("[dbpool] WAL checkpoint: success")
	}
}
