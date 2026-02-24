package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vantagedata/agent"
	"vantagedata/logger"
)

// setupAutoDecryptTestApp creates an App with a real DuckDB data source for testing
// the auto-decrypt flow in LoadQuickAnalysisPackByPath.
func setupAutoDecryptTestApp(t *testing.T) (*App, string) {
	t.Helper()
	tmpDir := t.TempDir()

	// Create a DuckDB database with a simple table
	dbDir := filepath.Join(tmpDir, "ds_test-ds-1")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("failed to create db dir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "data.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	_, err = db.Exec("CREATE TABLE orders (id INTEGER, amount REAL)")
	if err != nil {
		db.Close()
		t.Fatalf("failed to create table: %v", err)
	}
	db.Close()

	// Set up DataSourceService and register the data source
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	testDS := agent.DataSource{
		ID:   "test-ds-1",
		Name: "Test DS",
		Type: "sqlite",
		Config: agent.DataSourceConfig{
			DBPath: "ds_test-ds-1/data.duckdb",
		},
	}
	if err := service.AddDataSource(testDS); err != nil {
		t.Fatalf("failed to add data source: %v", err)
	}

	app := &App{
		dataSourceService: service,
		logger:            logger.NewLogger(),
		packPasswords:     make(map[string]string),
	}

	return app, tmpDir
}

// createEncryptedQAP creates an encrypted .qap file with a minimal valid pack and returns its path.
func createEncryptedQAP(t *testing.T, dir string, password string) string {
	t.Helper()
	pack := QuickAnalysisPack{
		FileType:      "Vantagics_QuickAnalysisPack",
		FormatVersion: "1.0",
		Metadata: PackMetadata{
			Author:     "test-author",
			CreatedAt:  time.Now().Format(time.RFC3339),
			SourceName: "test_source",
		},
		SchemaRequirements: []PackTableSchema{
			{
				TableName: "orders",
				Columns: []PackColumnInfo{
					{Name: "id", Type: "INTEGER"},
					{Name: "amount", Type: "REAL"},
				},
			},
		},
		ExecutableSteps: []PackStep{
			{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM orders", Description: "get orders"},
		},
	}

	jsonData, err := json.Marshal(pack)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	qapPath := filepath.Join(dir, "encrypted_test.qap")
	if err := PackToZip(jsonData, qapPath, password); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}
	return qapPath
}

// TestAutoDecrypt_StoredPasswordSuccess tests that when a correct stored password
// exists in packPasswords, LoadQuickAnalysisPackByPath auto-decrypts successfully
// without requiring manual password input.
// Validates: Requirements 4.2, 4.3
func TestAutoDecrypt_StoredPasswordSuccess(t *testing.T) {
	app, tmpDir := setupAutoDecryptTestApp(t)
	password := "test-auto-decrypt-password"

	// Create an encrypted .qap file
	qapPath := createEncryptedQAP(t, tmpDir, password)

	// Store the correct password (simulating marketplace download)
	app.packPasswords[qapPath] = password

	// Load the pack — should auto-decrypt without NeedsPassword
	result, err := app.LoadQuickAnalysisPackByPath(qapPath, "test-ds-1")
	if err != nil {
		t.Fatalf("LoadQuickAnalysisPackByPath failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should NOT need password (auto-decrypted)
	if result.NeedsPassword {
		t.Error("expected NeedsPassword=false when stored password is correct")
	}

	// Should have loaded the pack successfully
	if result.Pack == nil {
		t.Fatal("expected Pack to be non-nil after auto-decrypt")
	}
	if result.Pack.FileType != "Vantagics_QuickAnalysisPack" {
		t.Errorf("FileType: got %q, want %q", result.Pack.FileType, "Vantagics_QuickAnalysisPack")
	}
	if result.Pack.Metadata.Author != "test-author" {
		t.Errorf("Author: got %q, want %q", result.Pack.Metadata.Author, "test-author")
	}
	if len(result.Pack.ExecutableSteps) != 1 {
		t.Errorf("expected 1 step, got %d", len(result.Pack.ExecutableSteps))
	}

	// IsEncrypted should be true (it was encrypted, just auto-decrypted)
	if !result.IsEncrypted {
		t.Error("expected IsEncrypted=true for auto-decrypted pack")
	}
}

// TestAutoDecrypt_WrongStoredPasswordFallback tests that when a stored password
// is wrong, LoadQuickAnalysisPackByPath falls back to NeedsPassword=true
// so the user can manually input the correct password.
// Validates: Requirements 4.2, 4.3
func TestAutoDecrypt_WrongStoredPasswordFallback(t *testing.T) {
	app, tmpDir := setupAutoDecryptTestApp(t)
	correctPassword := "correct-password"
	wrongPassword := "wrong-stored-password"

	// Create an encrypted .qap file with the correct password
	qapPath := createEncryptedQAP(t, tmpDir, correctPassword)

	// Store a WRONG password (simulating corrupted/outdated stored password)
	app.packPasswords[qapPath] = wrongPassword

	// Load the pack — auto-decrypt should fail, fall back to NeedsPassword
	result, err := app.LoadQuickAnalysisPackByPath(qapPath, "test-ds-1")
	if err != nil {
		t.Fatalf("LoadQuickAnalysisPackByPath should not return error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should need password (auto-decrypt failed)
	if !result.NeedsPassword {
		t.Error("expected NeedsPassword=true when stored password is wrong")
	}

	// Should be marked as encrypted
	if !result.IsEncrypted {
		t.Error("expected IsEncrypted=true for encrypted pack")
	}

	// Pack should be nil (not loaded yet, waiting for manual password)
	if result.Pack != nil {
		t.Error("expected Pack=nil when falling back to manual password input")
	}

	// FilePath should be set for the frontend to use with LoadQuickAnalysisPackWithPassword
	if result.FilePath != qapPath {
		t.Errorf("FilePath: got %q, want %q", result.FilePath, qapPath)
	}
}
