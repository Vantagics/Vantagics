package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

// GetMigrations returns all database migrations in order
func GetMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Create layout_configs table",
			Up: `
				CREATE TABLE IF NOT EXISTS layout_configs (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL,
					is_locked BOOLEAN DEFAULT FALSE,
					layout_data TEXT NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(user_id)
				);
				
				CREATE INDEX IF NOT EXISTS idx_layout_user ON layout_configs(user_id);
			`,
			Down: `
				DROP INDEX IF EXISTS idx_layout_user;
				DROP TABLE IF EXISTS layout_configs;
			`,
		},
	}
}

// InitDB initializes the application database and runs migrations
func InitDB(dataDir string) (*sql.DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Database file path
	dbPath := filepath.Join(dataDir, "vantagedata.db")

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// createMigrationsTable creates the schema_migrations table to track applied migrations
func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(query)
	return err
}

// runMigrations applies all pending migrations
func runMigrations(db *sql.DB) error {
	migrations := GetMigrations()

	for _, migration := range migrations {
		// Check if migration has already been applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migration.Version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status for version %d: %w", migration.Version, err)
		}

		if count > 0 {
			// Migration already applied, skip
			continue
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.Up); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d (%s): %w", migration.Version, migration.Description, err)
		}

		// Record migration
		if _, err := tx.Exec("INSERT INTO schema_migrations (version, description) VALUES (?, ?)", migration.Version, migration.Description); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Description)
	}

	return nil
}

// RollbackMigration rolls back a specific migration
func RollbackMigration(db *sql.DB, version int) error {
	migrations := GetMigrations()

	// Find the migration
	var targetMigration *Migration
	for _, m := range migrations {
		if m.Version == version {
			targetMigration = &m
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration version %d not found", version)
	}

	// Check if migration has been applied
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("migration %d has not been applied", version)
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute rollback
	if _, err := tx.Exec(targetMigration.Down); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to rollback migration %d: %w", version, err)
	}

	// Remove migration record
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	fmt.Printf("Rolled back migration %d: %s\n", version, targetMigration.Description)
	return nil
}
