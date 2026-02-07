package main

import (
	"os"
	"path/filepath"
	"testing"

	"vantagedata/agent"
	"vantagedata/logger"
)

// TestGetDataSourceStatistics_EmptyList tests with no data sources
func TestGetDataSourceStatistics_EmptyList(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	app := &App{
		dataSourceService: agent.NewDataSourceService(tmpDir, func(msg string) {}),
		logger:            logger.NewLogger(),
	}

	// Get statistics
	stats, err := app.GetDataSourceStatistics()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify empty statistics
	if stats.TotalCount != 0 {
		t.Errorf("Expected TotalCount=0, got %d", stats.TotalCount)
	}

	if len(stats.BreakdownByType) != 0 {
		t.Errorf("Expected empty BreakdownByType, got %d entries", len(stats.BreakdownByType))
	}

	if len(stats.DataSources) != 0 {
		t.Errorf("Expected empty DataSources, got %d entries", len(stats.DataSources))
	}
}

// TestGetDataSourceStatistics_SingleDataSource tests with one data source
func TestGetDataSourceStatistics_SingleDataSource(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	app := &App{
		dataSourceService: service,
		logger:            logger.NewLogger(),
	}

	// Add a test data source
	testDS := agent.DataSource{
		ID:   "test-id-1",
		Name: "Test MySQL DB",
		Type: "mysql",
	}
	if err := service.AddDataSource(testDS); err != nil {
		t.Fatalf("Failed to add test data source: %v", err)
	}

	// Get statistics
	stats, err := app.GetDataSourceStatistics()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify statistics
	if stats.TotalCount != 1 {
		t.Errorf("Expected TotalCount=1, got %d", stats.TotalCount)
	}

	if len(stats.BreakdownByType) != 1 {
		t.Errorf("Expected 1 entry in BreakdownByType, got %d", len(stats.BreakdownByType))
	}

	if stats.BreakdownByType["mysql"] != 1 {
		t.Errorf("Expected mysql count=1, got %d", stats.BreakdownByType["mysql"])
	}

	if len(stats.DataSources) != 1 {
		t.Errorf("Expected 1 DataSource, got %d", len(stats.DataSources))
	}

	if stats.DataSources[0].ID != "test-id-1" {
		t.Errorf("Expected ID='test-id-1', got '%s'", stats.DataSources[0].ID)
	}

	if stats.DataSources[0].Name != "Test MySQL DB" {
		t.Errorf("Expected Name='Test MySQL DB', got '%s'", stats.DataSources[0].Name)
	}

	if stats.DataSources[0].Type != "mysql" {
		t.Errorf("Expected Type='mysql', got '%s'", stats.DataSources[0].Type)
	}
}

// TestGetDataSourceStatistics_MultipleSameType tests with multiple data sources of same type
func TestGetDataSourceStatistics_MultipleSameType(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	app := &App{
		dataSourceService: service,
		logger:            logger.NewLogger(),
	}

	// Add multiple test data sources of same type
	testDS1 := agent.DataSource{
		ID:   "test-id-1",
		Name: "MySQL DB 1",
		Type: "mysql",
	}
	testDS2 := agent.DataSource{
		ID:   "test-id-2",
		Name: "MySQL DB 2",
		Type: "mysql",
	}
	testDS3 := agent.DataSource{
		ID:   "test-id-3",
		Name: "MySQL DB 3",
		Type: "mysql",
	}

	if err := service.AddDataSource(testDS1); err != nil {
		t.Fatalf("Failed to add test data source 1: %v", err)
	}
	if err := service.AddDataSource(testDS2); err != nil {
		t.Fatalf("Failed to add test data source 2: %v", err)
	}
	if err := service.AddDataSource(testDS3); err != nil {
		t.Fatalf("Failed to add test data source 3: %v", err)
	}

	// Get statistics
	stats, err := app.GetDataSourceStatistics()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify statistics
	if stats.TotalCount != 3 {
		t.Errorf("Expected TotalCount=3, got %d", stats.TotalCount)
	}

	if len(stats.BreakdownByType) != 1 {
		t.Errorf("Expected 1 entry in BreakdownByType, got %d", len(stats.BreakdownByType))
	}

	if stats.BreakdownByType["mysql"] != 3 {
		t.Errorf("Expected mysql count=3, got %d", stats.BreakdownByType["mysql"])
	}

	if len(stats.DataSources) != 3 {
		t.Errorf("Expected 3 DataSources, got %d", len(stats.DataSources))
	}
}

// TestGetDataSourceStatistics_MultipleDifferentTypes tests with multiple data sources of different types
func TestGetDataSourceStatistics_MultipleDifferentTypes(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	app := &App{
		dataSourceService: service,
		logger:            logger.NewLogger(),
	}

	// Add test data sources of different types
	testDS1 := agent.DataSource{
		ID:   "test-id-1",
		Name: "MySQL DB",
		Type: "mysql",
	}
	testDS2 := agent.DataSource{
		ID:   "test-id-2",
		Name: "PostgreSQL DB",
		Type: "postgresql",
	}
	testDS3 := agent.DataSource{
		ID:   "test-id-3",
		Name: "Excel File",
		Type: "excel",
	}
	testDS4 := agent.DataSource{
		ID:   "test-id-4",
		Name: "SQLite DB",
		Type: "sqlite",
	}
	testDS5 := agent.DataSource{
		ID:   "test-id-5",
		Name: "Another MySQL",
		Type: "mysql",
	}

	if err := service.AddDataSource(testDS1); err != nil {
		t.Fatalf("Failed to add test data source 1: %v", err)
	}
	if err := service.AddDataSource(testDS2); err != nil {
		t.Fatalf("Failed to add test data source 2: %v", err)
	}
	if err := service.AddDataSource(testDS3); err != nil {
		t.Fatalf("Failed to add test data source 3: %v", err)
	}
	if err := service.AddDataSource(testDS4); err != nil {
		t.Fatalf("Failed to add test data source 4: %v", err)
	}
	if err := service.AddDataSource(testDS5); err != nil {
		t.Fatalf("Failed to add test data source 5: %v", err)
	}

	// Get statistics
	stats, err := app.GetDataSourceStatistics()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify statistics
	if stats.TotalCount != 5 {
		t.Errorf("Expected TotalCount=5, got %d", stats.TotalCount)
	}

	if len(stats.BreakdownByType) != 4 {
		t.Errorf("Expected 4 entries in BreakdownByType, got %d", len(stats.BreakdownByType))
	}

	if stats.BreakdownByType["mysql"] != 2 {
		t.Errorf("Expected mysql count=2, got %d", stats.BreakdownByType["mysql"])
	}

	if stats.BreakdownByType["postgresql"] != 1 {
		t.Errorf("Expected postgresql count=1, got %d", stats.BreakdownByType["postgresql"])
	}

	if stats.BreakdownByType["excel"] != 1 {
		t.Errorf("Expected excel count=1, got %d", stats.BreakdownByType["excel"])
	}

	if stats.BreakdownByType["sqlite"] != 1 {
		t.Errorf("Expected sqlite count=1, got %d", stats.BreakdownByType["sqlite"])
	}

	if len(stats.DataSources) != 5 {
		t.Errorf("Expected 5 DataSources, got %d", len(stats.DataSources))
	}

	// Verify sum of breakdown equals total count
	sum := 0
	for _, count := range stats.BreakdownByType {
		sum += count
	}
	if sum != stats.TotalCount {
		t.Errorf("Sum of breakdown (%d) does not equal TotalCount (%d)", sum, stats.TotalCount)
	}
}

// TestGetDataSourceStatistics_ServiceNotInitialized tests error handling when service is nil
func TestGetDataSourceStatistics_ServiceNotInitialized(t *testing.T) {
	// Create app without data source service
	app := &App{
		dataSourceService: nil,
		logger:            logger.NewLogger(),
	}

	// Get statistics
	_, err := app.GetDataSourceStatistics()
	if err == nil {
		t.Fatal("Expected error when service not initialized, got nil")
	}

	expectedMsg := "data source service not initialized"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestGetDataSourceStatistics_LoadDataSourcesError tests error handling when LoadDataSources fails
func TestGetDataSourceStatistics_LoadDataSourcesError(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vantagedata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create app with data source service
	service := agent.NewDataSourceService(tmpDir, func(msg string) {})
	app := &App{
		dataSourceService: service,
		logger:            logger.NewLogger(),
	}

	// Create an invalid datasources.json file to trigger error
	datasourcesPath := filepath.Join(tmpDir, "datasources.json")
	if err := os.WriteFile(datasourcesPath, []byte("invalid json content"), 0644); err != nil {
		t.Fatalf("Failed to create invalid datasources.json: %v", err)
	}

	// Get statistics
	_, err = app.GetDataSourceStatistics()
	if err == nil {
		t.Fatal("Expected error when LoadDataSources fails, got nil")
	}

	// Error should be wrapped with "failed to load data sources"
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}
