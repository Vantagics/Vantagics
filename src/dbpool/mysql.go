package dbpool

import (
	"database/sql"
	"fmt"
	"time"
)

// openMySQL opens a MySQL (or MySQL-compatible like Doris) connection with retry.
// NOTE: The caller's application must import the MySQL driver
// (e.g., _ "github.com/go-sql-driver/mysql").
func (m *DBManager) openMySQL(opts OpenOptions) (*sql.DB, error) {
	maxRetries, baseMs := retryParams(opts)

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		db, err := sql.Open("mysql", opts.Path)
		if err == nil {
			err = db.Ping()
			if err != nil {
				db.Close()
			}
		}

		if err != nil {
			lastErr = err
			m.logger(fmt.Sprintf("[dbpool] MySQL attempt %d/%d failed: %v", i+1, maxRetries, err))
			if maxRetries > 1 {
				time.Sleep(time.Duration(baseMs*(i+1)) * time.Millisecond)
			}
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("dbpool: failed to open MySQL after %d retries: %w", maxRetries, lastErr)
}
