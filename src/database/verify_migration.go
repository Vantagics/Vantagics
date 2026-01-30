// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"vantagedata/database"
)

// This is a standalone verification script to demonstrate the migration system
// Run with: go run verify_migration.go

func main() {
	// Create temporary directory for demonstration
	tempDir, err := os.MkdirTemp("", "vantagedata-verify-*")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("Using temporary directory: %s\n\n", tempDir)

	// Initialize database
	fmt.Println("Initializing database...")
	db, err := database.InitDB(tempDir)
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	dbPath := filepath.Join(tempDir, "vantagedata.db")
	fmt.Printf("Database created at: %s\n\n", dbPath)

	// Verify schema_migrations table
	fmt.Println("Checking schema_migrations table...")
	rows, err := db.Query("SELECT version, description, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		fmt.Printf("Failed to query migrations: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Println("\nApplied Migrations:")
	fmt.Println("-------------------")
	for rows.Next() {
		var version int
		var description, appliedAt string
		if err := rows.Scan(&version, &description, &appliedAt); err != nil {
			fmt.Printf("Failed to scan row: %v\n", err)
			continue
		}
		fmt.Printf("Version %d: %s (applied at %s)\n", version, description, appliedAt)
	}

	// Verify layout_configs table
	fmt.Println("\nChecking layout_configs table...")
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='layout_configs'").Scan(&count)
	if err != nil {
		fmt.Printf("Failed to check table: %v\n", err)
		os.Exit(1)
	}
	if count == 1 {
		fmt.Println("✓ layout_configs table exists")
	} else {
		fmt.Println("✗ layout_configs table not found")
		os.Exit(1)
	}

	// Verify table schema
	fmt.Println("\nTable Schema:")
	fmt.Println("-------------")
	schemaRows, err := db.Query("PRAGMA table_info(layout_configs)")
	if err != nil {
		fmt.Printf("Failed to query schema: %v\n", err)
		os.Exit(1)
	}
	defer schemaRows.Close()

	for schemaRows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := schemaRows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			fmt.Printf("Failed to scan schema: %v\n", err)
			continue
		}

		pkStr := ""
		if pk == 1 {
			pkStr = " (PRIMARY KEY)"
		}
		notNullStr := ""
		if notNull == 1 {
			notNullStr = " NOT NULL"
		}
		defaultStr := ""
		if dfltValue.Valid {
			defaultStr = fmt.Sprintf(" DEFAULT %s", dfltValue.String)
		}

		fmt.Printf("  %s: %s%s%s%s\n", name, colType, notNullStr, defaultStr, pkStr)
	}

	// Verify index
	fmt.Println("\nChecking indexes...")
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_layout_user'").Scan(&count)
	if err != nil {
		fmt.Printf("Failed to check index: %v\n", err)
		os.Exit(1)
	}
	if count == 1 {
		fmt.Println("✓ idx_layout_user index exists")
	} else {
		fmt.Println("✗ idx_layout_user index not found")
		os.Exit(1)
	}

	// Test inserting a record
	fmt.Println("\nTesting data insertion...")
	_, err = db.Exec(`
		INSERT INTO layout_configs (id, user_id, is_locked, layout_data)
		VALUES ('test-id', 'test-user', 0, '{"items":[]}')
	`)
	if err != nil {
		fmt.Printf("Failed to insert test record: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Successfully inserted test record")

	// Verify the record
	var id, userId, layoutData string
	var isLocked bool
	err = db.QueryRow("SELECT id, user_id, is_locked, layout_data FROM layout_configs WHERE id = 'test-id'").
		Scan(&id, &userId, &isLocked, &layoutData)
	if err != nil {
		fmt.Printf("Failed to query test record: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Retrieved record: id=%s, user_id=%s, is_locked=%v, layout_data=%s\n", id, userId, isLocked, layoutData)

	// Test UNIQUE constraint
	fmt.Println("\nTesting UNIQUE constraint on user_id...")
	_, err = db.Exec(`
		INSERT INTO layout_configs (id, user_id, is_locked, layout_data)
		VALUES ('test-id-2', 'test-user', 0, '{"items":[]}')
	`)
	if err != nil {
		fmt.Println("✓ UNIQUE constraint working correctly (duplicate user_id rejected)")
	} else {
		fmt.Println("✗ UNIQUE constraint not working (duplicate user_id accepted)")
		os.Exit(1)
	}

	fmt.Println("\n✓ All verification checks passed!")
	fmt.Println("\nMigration system is working correctly.")
}
