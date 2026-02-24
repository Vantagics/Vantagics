package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// newTestPack creates a QuickAnalysisPack with the given steps for testing.
func newTestPack(steps []PackStep) QuickAnalysisPack {
	return QuickAnalysisPack{
		FileType:      "Vantagics_QuickAnalysisPack",
		FormatVersion: "1.0",
		Metadata: PackMetadata{
			Author:      "test-author",
			CreatedAt:   time.Now().Format(time.RFC3339),
			SourceName:  "test_datasource",
			Description: "integration test pack",
		},
		SchemaRequirements: []PackTableSchema{
			{
				TableName: "orders",
				Columns: []PackColumnInfo{
					{Name: "id", Type: "INTEGER"},
					{Name: "amount", Type: "REAL"},
				},
			},
			{
				TableName: "customers",
				Columns: []PackColumnInfo{
					{Name: "id", Type: "INTEGER"},
					{Name: "name", Type: "TEXT"},
					{Name: "email", Type: "TEXT"},
				},
			},
		},
		ExecutableSteps: steps,
	}
}

func TestIntegration_ExportImportRoundTrip_NoEncryption(t *testing.T) {
	pack := newTestPack([]PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM orders", Description: "get orders"},
		{StepID: 2, StepType: "python_code", Code: "print('hello')", Description: "run python", DependsOn: []int{1}},
	})

	// Marshal to JSON
	jsonData, err := json.Marshal(pack)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Pack to ZIP (no password)
	dir := t.TempDir()
	qapPath := filepath.Join(dir, "test.qap")
	if err := PackToZip(jsonData, qapPath, ""); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	// Verify not encrypted
	encrypted, err := IsEncrypted(qapPath)
	if err != nil {
		t.Fatalf("IsEncrypted: %v", err)
	}
	if encrypted {
		t.Fatal("expected file to NOT be encrypted")
	}

	// Unpack
	got, err := UnpackFromZip(qapPath, "")
	if err != nil {
		t.Fatalf("UnpackFromZip: %v", err)
	}

	// Unmarshal and verify
	var restored QuickAnalysisPack
	if err := json.Unmarshal(got, &restored); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if restored.FileType != pack.FileType {
		t.Errorf("FileType: got %q, want %q", restored.FileType, pack.FileType)
	}
	if restored.FormatVersion != pack.FormatVersion {
		t.Errorf("FormatVersion: got %q, want %q", restored.FormatVersion, pack.FormatVersion)
	}
	if restored.Metadata.Author != pack.Metadata.Author {
		t.Errorf("Author: got %q, want %q", restored.Metadata.Author, pack.Metadata.Author)
	}
	if restored.Metadata.SourceName != pack.Metadata.SourceName {
		t.Errorf("SourceName: got %q, want %q", restored.Metadata.SourceName, pack.Metadata.SourceName)
	}
	if len(restored.SchemaRequirements) != len(pack.SchemaRequirements) {
		t.Errorf("SchemaRequirements count: got %d, want %d", len(restored.SchemaRequirements), len(pack.SchemaRequirements))
	}
	if len(restored.ExecutableSteps) != len(pack.ExecutableSteps) {
		t.Fatalf("ExecutableSteps count: got %d, want %d", len(restored.ExecutableSteps), len(pack.ExecutableSteps))
	}
	for i, step := range restored.ExecutableSteps {
		if step.StepID != pack.ExecutableSteps[i].StepID {
			t.Errorf("step[%d].StepID: got %d, want %d", i, step.StepID, pack.ExecutableSteps[i].StepID)
		}
		if step.StepType != pack.ExecutableSteps[i].StepType {
			t.Errorf("step[%d].StepType: got %q, want %q", i, step.StepType, pack.ExecutableSteps[i].StepType)
		}
		if step.Code != pack.ExecutableSteps[i].Code {
			t.Errorf("step[%d].Code: got %q, want %q", i, step.Code, pack.ExecutableSteps[i].Code)
		}
	}
}

func TestIntegration_ExportImportRoundTrip_WithEncryption(t *testing.T) {
	pack := newTestPack([]PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT count(*) FROM customers", Description: "count customers"},
	})

	jsonData, err := json.Marshal(pack)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	dir := t.TempDir()
	qapPath := filepath.Join(dir, "encrypted.qap")
	password := "my-s3cret-p@ss"

	// Pack with encryption
	if err := PackToZip(jsonData, qapPath, password); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	// Verify encrypted
	encrypted, err := IsEncrypted(qapPath)
	if err != nil {
		t.Fatalf("IsEncrypted: %v", err)
	}
	if !encrypted {
		t.Fatal("expected file to be encrypted")
	}

	// Unpack with correct password
	got, err := UnpackFromZip(qapPath, password)
	if err != nil {
		t.Fatalf("UnpackFromZip with correct password: %v", err)
	}

	var restored QuickAnalysisPack
	if err := json.Unmarshal(got, &restored); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if restored.Metadata.Author != pack.Metadata.Author {
		t.Errorf("Author mismatch after decrypt: got %q, want %q", restored.Metadata.Author, pack.Metadata.Author)
	}
	if len(restored.ExecutableSteps) != 1 {
		t.Errorf("expected 1 step, got %d", len(restored.ExecutableSteps))
	}

	// Wrong password → ErrWrongPassword
	_, err = UnpackFromZip(qapPath, "wrong-password")
	if !errors.Is(err, ErrWrongPassword) {
		t.Errorf("expected ErrWrongPassword, got: %v", err)
	}

	// Empty password → ErrPasswordRequired
	_, err = UnpackFromZip(qapPath, "")
	if !errors.Is(err, ErrPasswordRequired) {
		t.Errorf("expected ErrPasswordRequired, got: %v", err)
	}
}

func TestIntegration_SchemaIncompatibility(t *testing.T) {
	sourceSchema := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
				{Name: "status", Type: "TEXT"},
			},
		},
		{
			TableName: "customers",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
			},
		},
		{
			TableName: "products",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "price", Type: "REAL"},
			},
		},
	}

	// Case 1: target missing 1 table → incompatible
	targetMissingTable := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
				{Name: "status", Type: "TEXT"},
			},
		},
		{
			TableName: "customers",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
			},
		},
		// "products" table is missing
	}

	result := ValidateSchema(sourceSchema, targetMissingTable)
	if result.Compatible {
		t.Error("case 1: expected Compatible=false when table is missing")
	}
	if len(result.MissingTables) != 1 || result.MissingTables[0] != "products" {
		t.Errorf("case 1: expected MissingTables=[products], got %v", result.MissingTables)
	}

	// Case 2: all tables present but missing some columns → compatible with warnings
	targetMissingCols := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				// "amount" and "status" missing
			},
		},
		{
			TableName: "customers",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
			},
		},
		{
			TableName: "products",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				// "price" missing
			},
		},
	}

	result = ValidateSchema(sourceSchema, targetMissingCols)
	if !result.Compatible {
		t.Error("case 2: expected Compatible=true (missing columns are warnings)")
	}
	if len(result.MissingColumns) != 3 {
		t.Errorf("case 2: expected 3 missing columns, got %d: %v", len(result.MissingColumns), result.MissingColumns)
	}
	missingCols := make(map[string]string)
	for _, mc := range result.MissingColumns {
		missingCols[mc.TableName+"."+mc.ColumnName] = ""
	}
	for _, expected := range []string{"orders.amount", "orders.status", "products.price"} {
		if _, ok := missingCols[expected]; !ok {
			t.Errorf("case 2: expected missing column %s", expected)
		}
	}

	// Case 3: target is a superset → fully compatible, no missing items
	targetSuperset := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
				{Name: "status", Type: "TEXT"},
				{Name: "extra_col", Type: "TEXT"},
			},
		},
		{
			TableName: "customers",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
				{Name: "bonus_field", Type: "INTEGER"},
			},
		},
		{
			TableName: "products",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "price", Type: "REAL"},
			},
		},
		{
			TableName: "extra_table",
			Columns:   []PackColumnInfo{{Name: "id", Type: "INTEGER"}},
		},
	}

	result = ValidateSchema(sourceSchema, targetSuperset)
	if !result.Compatible {
		t.Error("case 3: expected Compatible=true for superset target")
	}
	if len(result.MissingTables) != 0 {
		t.Errorf("case 3: expected no missing tables, got %v", result.MissingTables)
	}
	if len(result.MissingColumns) != 0 {
		t.Errorf("case 3: expected no missing columns, got %v", result.MissingColumns)
	}
}

func TestIntegration_CompletePackLifecycle(t *testing.T) {
	// Create a pack with mixed step types
	pack := newTestPack([]PackStep{
		{StepID: 1, StepType: "sql_query", Code: "CREATE TABLE tmp AS SELECT * FROM orders", Description: "create temp table"},
		{StepID: 2, StepType: "python_code", Code: "import pandas as pd\ndf = pd.DataFrame({'a':[1,2]})", Description: "python processing"},
		{StepID: 3, StepType: "sql_query", Code: "SELECT count(*) FROM tmp", Description: "count rows", DependsOn: []int{1}},
		{StepID: 4, StepType: "python_code", Code: "result = 42", Description: "compute result", DependsOn: []int{2}},
	})

	// Pack to ZIP with encryption
	jsonData, err := json.Marshal(pack)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	dir := t.TempDir()
	qapPath := filepath.Join(dir, "lifecycle.qap")
	password := "lifecycle-pass"

	if err := PackToZip(jsonData, qapPath, password); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	// Unpack and verify
	got, err := UnpackFromZip(qapPath, password)
	if err != nil {
		t.Fatalf("UnpackFromZip: %v", err)
	}

	var restored QuickAnalysisPack
	if err := json.Unmarshal(got, &restored); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// Verify metadata
	if restored.Metadata.Author != "test-author" {
		t.Errorf("Author: got %q, want %q", restored.Metadata.Author, "test-author")
	}
	if restored.Metadata.SourceName != "test_datasource" {
		t.Errorf("SourceName: got %q, want %q", restored.Metadata.SourceName, "test_datasource")
	}
	if restored.FileType != "Vantagics_QuickAnalysisPack" {
		t.Errorf("FileType: got %q", restored.FileType)
	}
	if restored.FormatVersion != "1.0" {
		t.Errorf("FormatVersion: got %q", restored.FormatVersion)
	}

	// Verify step count and types
	if len(restored.ExecutableSteps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(restored.ExecutableSteps))
	}

	expectedTypes := []string{"sql_query", "python_code", "sql_query", "python_code"}
	for i, step := range restored.ExecutableSteps {
		if step.StepType != expectedTypes[i] {
			t.Errorf("step[%d].StepType: got %q, want %q", i, step.StepType, expectedTypes[i])
		}
		if step.StepID != i+1 {
			t.Errorf("step[%d].StepID: got %d, want %d", i, step.StepID, i+1)
		}
	}

	// Verify step order preserved (DependsOn)
	if len(restored.ExecutableSteps[2].DependsOn) != 1 || restored.ExecutableSteps[2].DependsOn[0] != 1 {
		t.Errorf("step[2].DependsOn: got %v, want [1]", restored.ExecutableSteps[2].DependsOn)
	}
	if len(restored.ExecutableSteps[3].DependsOn) != 1 || restored.ExecutableSteps[3].DependsOn[0] != 2 {
		t.Errorf("step[3].DependsOn: got %v, want [2]", restored.ExecutableSteps[3].DependsOn)
	}

	// Validate schema against a compatible target
	compatibleTarget := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
				{Name: "extra", Type: "TEXT"},
			},
		},
		{
			TableName: "customers",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
				{Name: "email", Type: "TEXT"},
			},
		},
	}

	result := ValidateSchema(restored.SchemaRequirements, compatibleTarget)
	if !result.Compatible {
		t.Errorf("expected compatible with matching target, got MissingTables=%v", result.MissingTables)
	}

	// Validate schema against an incompatible target (missing "customers" table)
	incompatibleTarget := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
			},
		},
	}

	result = ValidateSchema(restored.SchemaRequirements, incompatibleTarget)
	if result.Compatible {
		t.Error("expected incompatible when customers table is missing")
	}
	if len(result.MissingTables) != 1 || result.MissingTables[0] != "customers" {
		t.Errorf("expected MissingTables=[customers], got %v", result.MissingTables)
	}
}
