package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ImportJSON processes a JSON file containing tabular data with schema information
func (s *DataSourceService) ImportJSON(name string, filePath string, headerGen func(string) (string, error)) (*DataSource, error) {
	// 1. Validate file
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// 2. Read and parse JSON
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %v", err)
	}

	// Parse structured format
	var structuredFormat JSONExportFormat
	if err := json.Unmarshal(fileData, &structuredFormat); err != nil {
		return nil, fmt.Errorf("invalid JSON format. Expected format:\n{\n  \"tables\": [\n    {\n      \"name\": \"table_name\",\n      \"columns\": [{\"name\": \"col1\", \"type\": \"TEXT\"}, ...],\n      \"data\": [{...}, ...]\n    }\n  ]\n}\nError: %v", err)
	}

	if len(structuredFormat.Tables) == 0 {
		return nil, fmt.Errorf("JSON file contains no tables. Expected format with 'tables' array")
	}

	s.log(fmt.Sprintf("Importing %d tables from structured JSON", len(structuredFormat.Tables)))

	// Prepare Metadata
	id := uuid.New().String()
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	// Create SQLite DB
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}
	defer db.Close()

	// Process each table
	var mainTableName string
	for i, table := range structuredFormat.Tables {
		if i == 0 {
			mainTableName = table.Name
		}

		// Validate table
		if table.Name == "" {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("table at index %d has no name", i)
		}

		if len(table.Columns) == 0 {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("table '%s' has no columns defined", table.Name)
		}

		// Create table with defined schema
		if err := s.createTableWithSchema(db, table); err != nil {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("failed to create table %s: %v", table.Name, err)
		}

		// Insert data
		if err := s.insertTableData(db, table); err != nil {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("failed to insert data for table %s: %v", table.Name, err)
		}

		s.log(fmt.Sprintf("Imported table '%s' with %d columns and %d rows", table.Name, len(table.Columns), len(table.Data)))
	}

	// Save to Registry
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "json",
		CreatedAt: time.Now().UnixMilli(),
		Config: DataSourceConfig{
			OriginalFile: filePath,
			DBPath:       relDBPath,
			TableName:    mainTableName,
		},
	}

	if err := s.AddDataSource(ds); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, err
	}

	return &ds, nil
}

// createTableWithSchema creates a table with the defined schema
func (s *DataSourceService) createTableWithSchema(db *sql.DB, table JSONTable) error {
	// Build CREATE TABLE statement
	var columns []string
	for _, col := range table.Columns {
		// Map JSON types to SQLite types
		sqliteType := col.Type
		if sqliteType == "" {
			sqliteType = "TEXT"
		}
		columns = append(columns, fmt.Sprintf("`%s` %s", col.Name, sqliteType))
	}

	createSQL := fmt.Sprintf("CREATE TABLE `%s` (%s)", table.Name, strings.Join(columns, ", "))
	_, err := db.Exec(createSQL)
	return err
}

// insertTableData inserts data into a table
func (s *DataSourceService) insertTableData(db *sql.DB, table JSONTable) error {
	if len(table.Data) == 0 {
		return nil // No data to insert
	}

	// Prepare insert statement
	placeholders := make([]string, len(table.Columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	insertSQL := fmt.Sprintf("INSERT INTO `%s` VALUES (%s)", table.Name, strings.Join(placeholders, ","))

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	// Insert each row
	for _, row := range table.Data {
		values := make([]interface{}, len(table.Columns))
		for i, col := range table.Columns {
			values[i] = row[col.Name]
		}

		if _, err := stmt.Exec(values...); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
