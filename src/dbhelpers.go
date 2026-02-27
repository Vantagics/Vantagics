package main

import (
	"database/sql"
	"fmt"
)

// QueryRowsFunc is a generic function type for processing database rows.
// It receives a *sql.Rows and returns the processed result and any error.
type QueryRowsFunc[T any] func(*sql.Rows) (T, error)

// QueryRows executes a query and processes the results using the provided function.
// It automatically handles rows.Close() and error checking.
//
// Example usage:
//   tables, err := QueryRows(db, "SELECT table_name FROM information_schema.tables",
//       func(rows *sql.Rows) ([]string, error) {
//           var tables []string
//           for rows.Next() {
//               var name string
//               if err := rows.Scan(&name); err != nil {
//                   return nil, err
//               }
//               tables = append(tables, name)
//           }
//           return tables, rows.Err()
//       })
func QueryRows[T any](db *sql.DB, query string, processFunc QueryRowsFunc[T]) (T, error) {
	var zero T
	
	rows, err := db.Query(query)
	if err != nil {
		return zero, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	result, err := processFunc(rows)
	if err != nil {
		return zero, fmt.Errorf("processing rows failed: %w", err)
	}
	
	if err := rows.Err(); err != nil {
		return zero, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return result, nil
}

// QueryRowsWithArgs executes a parameterized query and processes the results.
//
// Example usage:
//   columns, err := QueryRowsWithArgs(db,
//       "SELECT column_name FROM information_schema.columns WHERE table_name = ?",
//       []interface{}{tableName},
//       func(rows *sql.Rows) ([]string, error) {
//           var columns []string
//           for rows.Next() {
//               var name string
//               if err := rows.Scan(&name); err != nil {
//                   return nil, err
//               }
//               columns = append(columns, name)
//           }
//           return columns, nil
//       })
func QueryRowsWithArgs[T any](db *sql.DB, query string, args []interface{}, processFunc QueryRowsFunc[T]) (T, error) {
	var zero T
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return zero, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	result, err := processFunc(rows)
	if err != nil {
		return zero, fmt.Errorf("processing rows failed: %w", err)
	}
	
	if err := rows.Err(); err != nil {
		return zero, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return result, nil
}

// QuerySingleValue executes a query that returns a single value.
//
// Example usage:
//   var count int
//   err := QuerySingleValue(db, "SELECT COUNT(*) FROM users", &count)
func QuerySingleValue[T any](db *sql.DB, query string, dest *T) error {
	err := db.QueryRow(query).Scan(dest)
	if err != nil {
		return fmt.Errorf("query single value failed: %w", err)
	}
	return nil
}

// QuerySingleValueWithArgs executes a parameterized query that returns a single value.
//
// Example usage:
//   var name string
//   err := QuerySingleValueWithArgs(db, "SELECT name FROM users WHERE id = ?", []interface{}{userID}, &name)
func QuerySingleValueWithArgs[T any](db *sql.DB, query string, args []interface{}, dest *T) error {
	err := db.QueryRow(query, args...).Scan(dest)
	if err != nil {
		return fmt.Errorf("query single value failed: %w", err)
	}
	return nil
}

// QueryMapRows executes a query and returns results as a slice of maps.
// Each map represents a row with column names as keys.
//
// Example usage:
//   results, err := QueryMapRows(db, "SELECT id, name, email FROM users")
func QueryMapRows(db *sql.DB, query string) ([]map[string]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	
	var results []map[string]interface{}
	
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		
		// Create a map for this row
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}
		
		results = append(results, rowMap)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return results, nil
}

// ExecWithRetry executes a statement with automatic retry on transient errors.
// Useful for handling temporary database locks or connection issues.
//
// Example usage:
//   err := ExecWithRetry(db, "INSERT INTO logs (message) VALUES (?)", []interface{}{"test"}, 3)
func ExecWithRetry(db *sql.DB, query string, args []interface{}, maxRetries int) error {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		_, err := db.Exec(query, args...)
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// Check if error is retryable (database locked, connection reset, etc.)
		errStr := err.Error()
		isRetryable := false
		retryableErrors := []string{"database is locked", "connection reset", "broken pipe"}
		for _, retryErr := range retryableErrors {
			if contains(errStr, retryErr) {
				isRetryable = true
				break
			}
		}
		
		if !isRetryable || attempt == maxRetries {
			break
		}
		
		// Wait before retry (exponential backoff could be added here)
		// time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	
	return fmt.Errorf("exec failed after %d attempts: %w", maxRetries+1, lastErr)
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
