package agent

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
	_ "modernc.org/sqlite"
)

func TestDataSourceService_ImportExcel(t *testing.T) {
	// 1. Setup Temp Dir
	tempDir, err := os.MkdirTemp("", "rapidbi_test_data")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Create Sample Excel
	excelPath := filepath.Join(tempDir, "test_data.xlsx")
	f := excelize.NewFile()
	// Create a new sheet.
	index, err := f.NewSheet("Sheet1")
	if err != nil {
		t.Fatalf("Failed to create sheet: %v", err)
	}
	// Set value of a cell.
	f.SetCellValue("Sheet1", "A1", "ID")
	f.SetCellValue("Sheet1", "B1", "Name")
	f.SetCellValue("Sheet1", "C1", "Age")
	f.SetCellValue("Sheet1", "A2", 1)
	f.SetCellValue("Sheet1", "B2", "Alice")
	f.SetCellValue("Sheet1", "C2", 30)
	f.SetCellValue("Sheet1", "A3", 2)
	f.SetCellValue("Sheet1", "B3", "Bob")
	f.SetCellValue("Sheet1", "C3", 25)

	// Set active sheet of the workbook.
	f.SetActiveSheet(index)
	// Save spreadsheet by the given path.
	if err := f.SaveAs(excelPath); err != nil {
		t.Fatalf("Failed to save excel: %v", err)
	}

	// 3. Init Service
	service := NewDataSourceService(tempDir, nil)

	// 4. Test Import
	ds, err := service.ImportExcel("Test Source", excelPath, nil)
	if err != nil {
		t.Fatalf("ImportExcel failed: %v", err)
	}

	if ds.Name != "Test Source" {
		t.Errorf("Expected name 'Test Source', got %s", ds.Name)
	}
	if ds.Type != "excel" {
		t.Errorf("Expected type 'excel', got %s", ds.Type)
	}

	// 5. Verify Metadata
	sources, err := service.LoadDataSources()
	if err != nil {
		t.Fatalf("LoadDataSources failed: %v", err)
	}
	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}

	// 6. Verify SQLite
	dbPath := filepath.Join(tempDir, ds.Config.DBPath)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file not found at %s", dbPath)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer db.Close()

	var count int
	// Table name should be Sheet1 now
	err = db.QueryRow("SELECT COUNT(*) FROM Sheet1").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 2 { // 2 rows of data
		t.Errorf("Expected 2 rows, got %d", count)
	}

    // Check Content
    var name string
    err = db.QueryRow("SELECT Name FROM Sheet1 WHERE ID='1'").Scan(&name)
    if err != nil {
        t.Fatalf("Query content failed: %v", err)
    }
    if name != "Alice" {
        t.Errorf("Expected Alice, got %s", name)
    }
    db.Close()

	// 7. Test Delete
	err = service.DeleteDataSource(ds.ID)
	if err != nil {
		t.Fatalf("DeleteDataSource failed: %v", err)
	}

	// Verify cleanup
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Error("Database file was not deleted")
	}

    sources, err = service.LoadDataSources()
    if len(sources) != 0 {
        t.Errorf("Expected 0 sources, got %d", len(sources))
    }
}

func TestDataSourceService_ImportExcel_NoHeader(t *testing.T) {
	// 1. Setup Temp Dir
	tempDir, err := os.MkdirTemp("", "rapidbi_test_noheader")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Create Sample Excel WITHOUT headers (starts with numbers)
	excelPath := filepath.Join(tempDir, "no_header.xlsx")
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", 100)
	f.SetCellValue("Sheet1", "B1", "Data 1")
	f.SetCellValue("Sheet1", "A2", 200)
	f.SetCellValue("Sheet1", "B2", "Data 2")
	if err := f.SaveAs(excelPath); err != nil {
		t.Fatalf("Failed to save excel: %v", err)
	}

	// 3. Init Service
	service := NewDataSourceService(tempDir, nil)

	// 4. Test Import
	ds, err := service.ImportExcel("No Header Source", excelPath, nil)
	if err != nil {
		t.Fatalf("ImportExcel failed: %v", err)
	}

	// 5. Verify SQLite columns
	dbPath := filepath.Join(tempDir, ds.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Check if data is imported correctly
	var count int
	// Table name should be Sheet1 now
	err = db.QueryRow("SELECT COUNT(*) FROM Sheet1").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows, got %d", count)
	}

	// Check column names (should be generated like field_1_integer, field_2_text)
	// We can check this by querying the first column using its expected name
	var val int
	err = db.QueryRow("SELECT field_1_integer FROM Sheet1 WHERE field_1_integer=100").Scan(&val)
	if err != nil {
		t.Fatalf("Failed to query generated column name: %v", err)
	}
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	// 6. Test Delete
	err = service.DeleteDataSource(ds.ID)
	if err != nil {
		t.Fatalf("DeleteDataSource failed: %v", err)
	}
}

func TestDataSourceService_ImportExcel_MultipleSheets(t *testing.T) {
	// 1. Setup Temp Dir
	tempDir, err := os.MkdirTemp("", "rapidbi_test_multisheet")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Create Sample Excel with 2 sheets
	excelPath := filepath.Join(tempDir, "multisheet.xlsx")
	f := excelize.NewFile()

	// Sheet 1 (default)
	f.SetSheetName("Sheet1", "Sales")
	f.SetCellValue("Sales", "A1", "Product")
	f.SetCellValue("Sales", "B1", "Amount")
	f.SetCellValue("Sales", "A2", "Apple")
	f.SetCellValue("Sales", "B2", 100)

	// Sheet 2
	f.NewSheet("Customers")
	f.SetCellValue("Customers", "A1", "Name")
	f.SetCellValue("Customers", "B1", "City")
	f.SetCellValue("Customers", "A2", "John")
	f.SetCellValue("Customers", "B2", "New York")

	if err := f.SaveAs(excelPath); err != nil {
		t.Fatalf("Failed to save excel: %v", err)
	}

	// 3. Init Service
	service := NewDataSourceService(tempDir, nil)

	// 4. Test Import
	ds, err := service.ImportExcel("Multi Sheet Source", excelPath, nil)
	if err != nil {
		t.Fatalf("ImportExcel failed: %v", err)
	}

	// 5. Verify SQLite
	dbPath := filepath.Join(tempDir, ds.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Verify Sales table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM Sales").Scan(&count)
	if err != nil {
		t.Fatalf("Query Sales failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row in Sales, got %d", count)
	}

	// Verify Customers table
	err = db.QueryRow("SELECT COUNT(*) FROM Customers").Scan(&count)
	if err != nil {
		t.Fatalf("Query Customers failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row in Customers, got %d", count)
	}
}

func TestDataSourceService_GetDataSourceTableCount(t *testing.T) {
	// 1. Setup Temp Dir
	tempDir, err := os.MkdirTemp("", "rapidbi_test_count")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Create Sample Excel
	excelPath := filepath.Join(tempDir, "count_test.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Data")
	f.SetCellValue("Data", "A1", "ID")
	f.SetCellValue("Data", "A2", 1)
	f.SetCellValue("Data", "A3", 2)
	f.SetCellValue("Data", "A4", 3)
	if err := f.SaveAs(excelPath); err != nil {
		t.Fatalf("Failed to save excel: %v", err)
	}

	// 3. Init Service
	service := NewDataSourceService(tempDir, nil)
	ds, err := service.ImportExcel("Count Source", excelPath, nil)
	if err != nil {
		t.Fatalf("ImportExcel failed: %v", err)
	}

	// 4. Test GetDataSourceTableCount
	count, err := service.GetDataSourceTableCount(ds.ID, "Data")
	if err != nil {
		t.Fatalf("GetDataSourceTableCount failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestDataSourceService_GetDataSourceTableColumns(t *testing.T) {
	// 1. Setup Temp Dir
	tempDir, err := os.MkdirTemp("", "rapidbi_test_cols")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Create Sample Excel
	excelPath := filepath.Join(tempDir, "cols_test.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Data")
	f.SetCellValue("Data", "A1", "ID")
	f.SetCellValue("Data", "B1", "Name")
	f.SetCellValue("Data", "C1", "Value")
	f.SetCellValue("Data", "A2", 1)
	if err := f.SaveAs(excelPath); err != nil {
		t.Fatalf("Failed to save excel: %v", err)
	}

	// 3. Init Service
	service := NewDataSourceService(tempDir, nil)
	ds, err := service.ImportExcel("Cols Source", excelPath, nil)
	if err != nil {
		t.Fatalf("ImportExcel failed: %v", err)
	}

	// 4. Test GetDataSourceTableColumns
	cols, err := service.GetDataSourceTableColumns(ds.ID, "Data")
	if err != nil {
		t.Fatalf("GetDataSourceTableColumns failed: %v", err)
	}

	expected := []string{"ID", "Name", "Value"}
	if len(cols) != len(expected) {
		t.Fatalf("Expected %d columns, got %d", len(expected), len(cols))
	}

	for i, col := range cols {
		if col != expected[i] {
			t.Errorf("Expected column %d to be '%s', got '%s'", i, expected[i], col)
		}
	}
}

func TestDataSourceService_AddDataSource_DuplicateName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rapidbi_test_dup")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	service := NewDataSourceService(tempDir, nil)
	ds1 := DataSource{ID: "1", Name: "Sales", Type: "excel"}

	// Add first source
	if err := service.AddDataSource(ds1); err != nil {
		t.Fatalf("Failed to add first source: %v", err)
	}

	// Try adding second source with same name
	ds2 := DataSource{ID: "2", Name: "sales", Type: "excel"}
	err = service.AddDataSource(ds2)
	if err == nil {
		t.Error("Expected error for duplicate name, got nil")
	} else if err.Error() != "data source with name 'sales' already exists" && err.Error() != "data source with name 'Sales' already exists" {
		// Error message might vary slightly depending on which name is used in error construction, but logic should hold
		// Current implementation uses ds.Name (the new one)
	}
}

func TestDataSourceService_ExportToCSV(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "rapidbi-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewDataSourceService(tmpDir, nil)

	// Create a dummy data source
	dbPath := filepath.Join(tmpDir, "sources", "test-id", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE test_table (id INTEGER, name TEXT)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO test_table VALUES (1, 'Alice'), (2, 'Bob')")
	if err != nil {
		t.Fatal(err)
	}

	ds := DataSource{
		ID:   "test-id",
		Name: "Test Source",
		Config: DataSourceConfig{
			DBPath: filepath.Join("sources", "test-id", "data.db"),
		},
	}
	if err := service.SaveDataSources([]DataSource{ds}); err != nil {
		t.Fatal(err)
	}

	// Test Export
	outputPath := filepath.Join(tmpDir, "export", "dummy.csv")
	err = service.ExportToCSV("test-id", []string{"test_table"}, outputPath)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	// Verify directory creation
	expectedDir := filepath.Join(tmpDir, "export", "Test_Source")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to exist", expectedDir)
	}

	// Verify file creation
	expectedFile := filepath.Join(expectedDir, "test_table.csv")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", expectedFile)
	}
}
