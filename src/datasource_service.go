package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

// DataSource represents a registered data source
type DataSource struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Type      string           `json:"type"` // excel, mysql, postgresql, etc.
	CreatedAt time.Time        `json:"created_at"`
	Config    DataSourceConfig `json:"config"`
}

// MySQLExportConfig holds MySQL export configuration
type MySQLExportConfig struct {
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Database string `json:"database,omitempty"`
}

// DataSourceConfig holds configuration specific to the data source
type DataSourceConfig struct {
	OriginalFile      string             `json:"original_file,omitempty"`
	DBPath            string             `json:"db_path"` // Relative to DataCacheDir
	TableName         string             `json:"table_name"`
	Host              string             `json:"host,omitempty"`
	Port              string             `json:"port,omitempty"`
	User              string             `json:"user,omitempty"`
	Password          string             `json:"password,omitempty"`
	Database          string             `json:"database,omitempty"`
	MySQLExportConfig *MySQLExportConfig `json:"mysql_export_config,omitempty"`
}

// DataSourceService handles data source operations
type DataSourceService struct {
	dataCacheDir string
}

// NewDataSourceService creates a new service instance
func NewDataSourceService(dataCacheDir string) *DataSourceService {
	return &DataSourceService{
		dataCacheDir: dataCacheDir,
	}
}

// getMetadataPath returns the path to datasources.json
func (s *DataSourceService) getMetadataPath() string {
	return filepath.Join(s.dataCacheDir, "datasources.json")
}

// LoadDataSources reads the registry of data sources
func (s *DataSourceService) LoadDataSources() ([]DataSource, error) {
	path := s.getMetadataPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []DataSource{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sources []DataSource
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, err
	}

	return sources, nil
}

// SaveDataSources writes the registry of data sources
func (s *DataSourceService) SaveDataSources(sources []DataSource) error {
	path := s.getMetadataPath()
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddDataSource adds a new source to the registry
func (s *DataSourceService) AddDataSource(ds DataSource) error {
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	for _, source := range sources {
		if strings.EqualFold(source.Name, ds.Name) {
			return fmt.Errorf("data source with name '%s' already exists", ds.Name)
		}
	}

	sources = append(sources, ds)
	return s.SaveDataSources(sources)
}

// DeleteDataSource removes a source and its data
func (s *DataSourceService) DeleteDataSource(id string) error {
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	newSources := []DataSource{}
	var target *DataSource

	for _, ds := range sources {
		if ds.ID == id {
			target = &ds // copy
		} else {
			newSources = append(newSources, ds)
		}
	}

	if target == nil {
		return fmt.Errorf("data source not found")
	}

	// Remove data directory
	dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
	dirToRemove := filepath.Dir(dbPath)

	if strings.HasPrefix(dirToRemove, s.dataCacheDir) {
		_ = os.RemoveAll(dirToRemove)
	}

	return s.SaveDataSources(newSources)
}

// UpdateMySQLExportConfig updates the MySQL export configuration for a data source
func (s *DataSourceService) UpdateMySQLExportConfig(id string, config MySQLExportConfig) error {
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	found := false
	for i := range sources {
		if sources[i].ID == id {
			sources[i].Config.MySQLExportConfig = &config
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("data source not found")
	}

	return s.SaveDataSources(sources)
}

// inferColumnType tries to guess the SQLite type from a string value
func (s *DataSourceService) inferColumnType(val string) string {
	if val == "" {
		return "TEXT"
	}
	if _, err := strconv.ParseInt(val, 10, 64); err == nil {
		return "INTEGER"
	}
	if _, err := strconv.ParseFloat(val, 64); err == nil {
		return "REAL"
	}
	return "TEXT"
}

// isHeaderRow checks if the row is likely a header row
func (s *DataSourceService) isHeaderRow(row []string) bool {
	if len(row) == 0 {
		return false
	}
	for _, cell := range row {
		if cell == "" {
			continue
		}
		// If it's a number, it's likely data, not a header
		if _, err := strconv.ParseFloat(cell, 64); err == nil {
			return false
		}
	}
	return true
}

// ImportDataSource is a generic method to import data from various sources
func (s *DataSourceService) ImportDataSource(name string, driverType string, config DataSourceConfig, headerGen func(string) (string, error)) (*DataSource, error) {
	switch strings.ToLower(driverType) {
	case "excel":
		return s.ImportExcel(name, config.OriginalFile, headerGen)
	case "csv":
		return s.ImportCSV(name, config.OriginalFile, headerGen)
	case "mysql", "postgresql", "doris":
		// For other types, we would typically connect and import.
		// For this implementation, we scaffold the behavior of "reading and writing to sqlite"
		// as requested. In a real scenario, we'd use appropriate drivers.
		return nil, fmt.Errorf("driver type %s import not yet fully implemented, but configured for caching", driverType)
	default:
		return nil, fmt.Errorf("unsupported driver type: %s", driverType)
	}
}

// sanitizeName makes a string safe for use as a table or column name
func (s *DataSourceService) sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)
	if name == "" {
		return "unknown"
	}
	return name
}

// processSheet handles the schema inference and data import for a single sheet
func (s *DataSourceService) processSheet(db *sql.DB, tableName string, rows [][]string, headerGen func(string) (string, error)) error {
	// Schema Inference and Table Creation
	var headers []string
	var dataStartRow int
	hasHeader := s.isHeaderRow(rows[0])

	// Determine column types using the first few data rows
	numCols := 0
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	colTypes := make([]string, numCols)
	for i := range colTypes {
		colTypes[i] = "TEXT" // Default
	}

	// Sample up to 10 rows for type inference
	sampleRows := 10
	if len(rows) < sampleRows {
		sampleRows = len(rows)
	}

	startSample := 0
	if hasHeader {
		startSample = 1
	}

	for i := 0; i < numCols; i++ {
		currentType := "INTEGER"
		for r := startSample; r < sampleRows; r++ {
			if i >= len(rows[r]) || rows[r][i] == "" {
				continue
			}
			t := s.inferColumnType(rows[r][i])
			if t == "TEXT" {
				currentType = "TEXT"
				break
			}
			if t == "REAL" && currentType == "INTEGER" {
				currentType = "REAL"
			}
		}
		colTypes[i] = currentType
	}

	if hasHeader {
		headers = rows[0]
		dataStartRow = 1
		// Sanitize headers and ensure uniqueness
		usedNames := make(map[string]int)
		for i, h := range headers {
			h = s.sanitizeName(h)
			if h == "" || h == "unknown" {
				h = fmt.Sprintf("col_%d_%s", i, strings.ToLower(colTypes[i]))
			}

			origH := h
			for usedNames[strings.ToLower(h)] > 0 {
				usedNames[origH]++
				h = fmt.Sprintf("%s_%d", origH, usedNames[origH])
			}
			usedNames[strings.ToLower(h)]++
			headers[i] = h
		}
	} else {
		// Try to generate headers using LLM if available
		if headerGen != nil {
			var sb strings.Builder
			sb.WriteString("Based on the following lines of data, suggest field names for each column, output only meaningful English field names separated by commas:\n")

			limit := 5
			if len(rows) < limit {
				limit = len(rows)
			}
			for i := 0; i < limit; i++ {
				// Check for comma in data to decide separator
				sep := ","
				for _, cell := range rows[i] {
					if strings.Contains(cell, ",") {
						sep = "|"
						break
					}
				}
				line := strings.Join(rows[i], sep)
				sb.WriteString(line)
				sb.WriteString("\n")
			}

			if resp, err := headerGen(sb.String()); err == nil && resp != "" {
				newHeaders := strings.Split(resp, ",")
				if len(newHeaders) == numCols {
					headers = make([]string, numCols)
					usedNames := make(map[string]int)
					for i, h := range newHeaders {
						h = s.sanitizeName(strings.TrimSpace(h))
						if h == "" || h == "unknown" {
							h = fmt.Sprintf("field_%d_%s", i+1, strings.ToLower(colTypes[i]))
						}

						origH := h
						for usedNames[strings.ToLower(h)] > 0 {
							usedNames[origH]++
							h = fmt.Sprintf("%s_%d", origH, usedNames[origH])
						}
						usedNames[strings.ToLower(h)]++
						headers[i] = h
					}
				}
			}
		}

		if len(headers) == 0 {
			headers = make([]string, numCols)
			for i := 0; i < numCols; i++ {
				headers[i] = fmt.Sprintf("field_%d_%s", i+1, strings.ToLower(colTypes[i]))
			}
		}
		dataStartRow = 0
	}

	createSQL := fmt.Sprintf("CREATE TABLE `%s` (", tableName)
	placeholders := []string{}

	for i, colName := range headers {
		createSQL += fmt.Sprintf("`%s` %s", colName, colTypes[i])
		if i < len(headers)-1 {
			createSQL += ", "
		}
		placeholders = append(placeholders, "?")
	}
	createSQL += ");"

	_, err := db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %v", tableName, err)
	}

	// Insert Data
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	insertSQL := fmt.Sprintf("INSERT INTO `%s` VALUES (%s)", tableName, strings.Join(placeholders, ","))
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i := dataStartRow; i < len(rows); i++ {
		row := rows[i]
		vals := make([]interface{}, len(headers))
		for j := 0; j < len(headers); j++ {
			if j < len(row) {
				val := row[j]
				if colTypes[j] == "INTEGER" {
					if iv, err := strconv.ParseInt(val, 10, 64); err == nil {
						vals[j] = iv
					} else {
						vals[j] = nil
					}
				} else if colTypes[j] == "REAL" {
					if fv, err := strconv.ParseFloat(val, 64); err == nil {
						vals[j] = fv
					} else {
						vals[j] = nil
					}
				} else {
					vals[j] = val
				}
			} else {
				vals[j] = nil
			}
		}
		_, err = stmt.Exec(vals...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert row %d in sheet %s: %v", i+1, tableName, err)
		}
	}

	return tx.Commit()
}

// ImportExcel processes an Excel file and stores it in SQLite
func (s *DataSourceService) ImportExcel(name string, filePath string, headerGen func(string) (string, error)) (*DataSource, error) {
	// 1. Validate file
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// 2. Open Excel
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %v", err)
	}
	defer f.Close()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("no sheets found in excel file")
	}

	// 3. Prepare Metadata
	id := uuid.New().String()
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	// 4. Create SQLite DB
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}
	defer db.Close()

	// 5. Process Sheets
	processedSheets := 0
	var mainTableName string
	usedTableNames := make(map[string]bool)

	for _, sheetName := range sheetList {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			// Skip sheets we can't read? Or just log? For now skip.
			continue
		}
		if len(rows) == 0 {
			continue
		}

		tableName := s.sanitizeName(sheetName)
		// Ensure unique table name
		originalTableName := tableName
		counter := 1
		for usedTableNames[tableName] {
			tableName = fmt.Sprintf("%s_%d", originalTableName, counter)
			counter++
		}
		usedTableNames[tableName] = true

		if processedSheets == 0 {
			mainTableName = tableName
		}

		if err := s.processSheet(db, tableName, rows, headerGen); err != nil {
			return nil, err
		}
		processedSheets++
	}

	if processedSheets == 0 {
		// Clean up
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("no valid data found in any sheet")
	}

	// 7. Save to Registry
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "excel",
		CreatedAt: time.Now(),
		Config: DataSourceConfig{
			OriginalFile: filePath,
			DBPath:       relDBPath,
			TableName:    mainTableName,
		},
	}

	if err := s.AddDataSource(ds); err != nil {
		return nil, err
	}

	return &ds, nil
}

// ImportCSV processes a single CSV file or a directory of CSV files
func (s *DataSourceService) ImportCSV(name string, path string, headerGen func(string) (string, error)) (*DataSource, error) {
	// 1. Validate path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	var csvFiles []string
	if info.IsDir() {
		// Directory: Find all CSV files
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %v", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
				csvFiles = append(csvFiles, filepath.Join(path, entry.Name()))
			}
		}
	} else {
		// Single File: Just add it
		if strings.HasSuffix(strings.ToLower(info.Name()), ".csv") {
			csvFiles = append(csvFiles, path)
		} else {
			return nil, fmt.Errorf("file is not a csv file: %s", path)
		}
	}

	if len(csvFiles) == 0 {
		return nil, fmt.Errorf("no csv files found")
	}

	// 3. Prepare Metadata
	id := uuid.New().String()
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	// 4. Create SQLite DB
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}
	defer db.Close()

	// 5. Process Files
	processedFiles := 0
	var mainTableName string
	usedTableNames := make(map[string]bool)

	for _, csvPath := range csvFiles {
		// Read CSV file
		f, err := os.Open(csvPath)
		if err != nil {
			continue
		}

		reader := csv.NewReader(f)
		rows, err := reader.ReadAll()
		f.Close()

		if err != nil {
			continue
		}

		if len(rows) == 0 {
			continue
		}

		// Use filename as table name
		baseName := filepath.Base(csvPath)
		ext := filepath.Ext(baseName)
		rawTableName := strings.TrimSuffix(baseName, ext)

		tableName := s.sanitizeName(rawTableName)
		// Ensure unique table name
		originalTableName := tableName
		counter := 1
		for usedTableNames[tableName] {
			tableName = fmt.Sprintf("%s_%d", originalTableName, counter)
			counter++
		}
		usedTableNames[tableName] = true

		if processedFiles == 0 {
			mainTableName = tableName
		}

		if err := s.processSheet(db, tableName, rows, headerGen); err != nil {
			return nil, err
		}
		processedFiles++
	}

	if processedFiles == 0 {
		// Clean up
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("no valid csv data found")
	}

	// 6. Save to Registry
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "csv",
		CreatedAt: time.Now(),
		Config: DataSourceConfig{
			OriginalFile: path,
			DBPath:       relDBPath,
			TableName:    mainTableName,
		},
	}

	if err := s.AddDataSource(ds); err != nil {
		return nil, err
	}

	return &ds, nil
}

// GetDataSourceTables returns all table names for a data source
func (s *DataSourceService) GetDataSourceTables(id string) ([]string, error) {
	sources, err := s.LoadDataSources()
	if err != nil {
		return nil, err
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == id {
			target = &ds
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("data source not found")
	}

	dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, nil
}

// GetDataSourceTableData returns preview data for a table
func (s *DataSourceService) GetDataSourceTableData(id string, tableName string, limit int) ([]map[string]interface{}, error) {
	sources, err := s.LoadDataSources()
	if err != nil {
		return nil, err
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == id {
			target = &ds
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("data source not found")
	}

	dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if limit <= 0 {
		limit = 100 // Safe default if something goes wrong
	}

	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d", tableName, limit)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var data []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			rowMap[colName] = *val
		}
		data = append(data, rowMap)
	}

	return data, nil
}

// GetDataSourceTableCount returns the total number of rows in a table
func (s *DataSourceService) GetDataSourceTableCount(id string, tableName string) (int, error) {
	sources, err := s.LoadDataSources()
	if err != nil {
		return 0, err
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == id {
			target = &ds
			break
		}
	}

	if target == nil {
		return 0, fmt.Errorf("data source not found")
	}

	dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", tableName)
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ExportToCSV exports one or more tables to CSV file(s)
func (s *DataSourceService) ExportToCSV(id string, tableNames []string, outputPath string) error {
	if len(tableNames) == 0 {
		return fmt.Errorf("no tables specified for export")
	}

	// 1. Get Data Source
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == id {
			target = &ds
			break
		}
	}

	if target == nil {
		return fmt.Errorf("data source not found")
	}

	dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// 2. Export each table
	// Create directory named after data source in the target directory (outputPath is now a directory)
	parentDir := outputPath
	dsName := s.sanitizeName(target.Name)
	exportDir := filepath.Join(parentDir, dsName)

	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %v", err)
	}

	for _, tableName := range tableNames {
		// Output file path for this table
		filePath := filepath.Join(exportDir, fmt.Sprintf("%s.csv", tableName))

		// Export this table
		if err := s.exportSingleTableToCSV(db, tableName, filePath); err != nil {
			return fmt.Errorf("failed to export table %s: %v", tableName, err)
		}
	}

	return nil
}

// exportSingleTableToCSV exports a single table to a CSV file
func (s *DataSourceService) exportSingleTableToCSV(db *sql.DB, tableName string, outputPath string) error {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM `%s`", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Create File
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write Headers
	if err := w.Write(cols); err != nil {
		return err
	}

	// Write Data
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		record := make([]string, len(cols))
		for i, val := range values {
			if val != nil {
				// Convert varied types to string
				switch v := val.(type) {
				case []byte:
					record[i] = string(v)
				default:
					record[i] = fmt.Sprintf("%v", v)
				}
			} else {
				record[i] = ""
			}
		}

		if err := w.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportToSQL exports one or more tables to a SQL file (INSERT statements)
func (s *DataSourceService) ExportToSQL(id string, tableNames []string, outputPath string) error {
	if len(tableNames) == 0 {
		return fmt.Errorf("no tables specified for export")
	}

	// 1. Get Data Source
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == id {
			target = &ds
			break
		}
	}

	if target == nil {
		return fmt.Errorf("data source not found")
	}

	dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// 2. Create File
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 3. Export each table
	for _, tableName := range tableNames {
		if err := s.exportSingleTableToSQL(db, tableName, f); err != nil {
			return fmt.Errorf("failed to export table %s: %v", tableName, err)
		}
		// Add separator between tables
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	return nil
}

// exportSingleTableToSQL exports a single table to a SQL file
func (s *DataSourceService) exportSingleTableToSQL(db *sql.DB, tableName string, f *os.File) error {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM `%s`", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Write Data
	// Basic INSERT statement generation
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}

	colNames := strings.Join(cols, "`, `")
	baseInsert := fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES ", tableName, colNames)

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		vals := []string{}
		for _, val := range values {
			if val == nil {
				vals = append(vals, "NULL")
			} else {
				// Escape strings
				switch v := val.(type) {
				case []byte:
					str := string(v)
					str = strings.ReplaceAll(str, "'", "''") // Simple SQL escaping
					vals = append(vals, fmt.Sprintf("'%s'", str))
				case string:
					str := strings.ReplaceAll(v, "'", "''")
					vals = append(vals, fmt.Sprintf("'%s'", str))
				case int64, float64:
					vals = append(vals, fmt.Sprintf("%v", v))
				default:
					str := fmt.Sprintf("%v", v)
					str = strings.ReplaceAll(str, "'", "''")
					vals = append(vals, fmt.Sprintf("'%s'", str))
				}
			}
		}

		stmt := fmt.Sprintf("%s(%s);\n", baseInsert, strings.Join(vals, ", "))
		if _, err := f.WriteString(stmt); err != nil {
			return err
		}
	}

	return nil
}
		
// ExportToMySQL exports one or more tables to a MySQL database
func (s *DataSourceService) ExportToMySQL(id string, tableNames []string, mysqlConfig DataSourceConfig) error {
	if len(tableNames) == 0 {
		return fmt.Errorf("no tables specified for export")
	}

	// 1. Get Data Source
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == id {
			target = &ds
			break
		}
	}

	if target == nil {
		return fmt.Errorf("data source not found")
	}

	dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
	sqliteDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer sqliteDB.Close()

	// 2. Connect to MySQL
	if mysqlConfig.Port == "" {
		mysqlConfig.Port = "3306"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", mysqlConfig.User, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.Database)
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to mysql: %v", err)
	}
	defer mysqlDB.Close()

	if err := mysqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping mysql: %v", err)
	}

	// 3. Export each table
	for _, tableName := range tableNames {
		if err := s.exportSingleTableToMySQL(sqliteDB, mysqlDB, tableName); err != nil {
			return fmt.Errorf("failed to export table %s: %v", tableName, err)
		}
	}

	return nil
}

// exportSingleTableToMySQL exports a single table to MySQL
func (s *DataSourceService) exportSingleTableToMySQL(sqliteDB *sql.DB, mysqlDB *sql.DB, tableName string) error {
	// Get Column Info
	query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
	rows, err := sqliteDB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Create Table in MySQL
	createCols := []string{}
	for _, col := range cols {
		createCols = append(createCols, fmt.Sprintf("`%s` TEXT", col))
	}
	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", tableName, strings.Join(createCols, ", "))
	if _, err := mysqlDB.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table in mysql: %v", err)
	}

	// Bulk Insert
	placeholders := make([]string, len(cols))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES (%s)", tableName, strings.Join(cols, "`, `"), strings.Join(placeholders, ", "))
	stmt, err := mysqlDB.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %v", err)
	}
	defer stmt.Close()

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}
		// Convert []byte to string for MySQL driver stability with TEXT
		finalValues := make([]interface{}, len(cols))
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				finalValues[i] = string(b)
			} else {
				finalValues[i] = v
			}
		}

		if _, err := stmt.Exec(finalValues...); err != nil {
			return fmt.Errorf("failed to insert row: %v", err)
		}
	}

	return nil
}


// TestMySQLConnection tests the connection to a MySQL server
func (s *DataSourceService) TestMySQLConnection(host, port, user, password string) error {
	if port == "" {
		port = "3306"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to create connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to MySQL server: %v", err)
	}

	return nil
}

// GetMySQLDatabases returns a list of databases from the MySQL server
func (s *DataSourceService) GetMySQLDatabases(host, port, user, password string) ([]string, error) {
	if port == "" {
		port = "3306"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL server: %v", err)
	}

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %v", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, fmt.Errorf("failed to scan database name: %v", err)
		}
		// Filter out system databases
		if dbName != "information_schema" && dbName != "mysql" && dbName != "performance_schema" && dbName != "sys" {
			databases = append(databases, dbName)
		}
	}

	return databases, nil
}
