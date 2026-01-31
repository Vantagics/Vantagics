package agent

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/extrame/xls"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

// SchemaCache holds cached schema information for a data source
type SchemaCache struct {
	Tables    []string                         // List of table names
	Columns   map[string][]string              // tableName -> column names
	Samples   map[string][]map[string]interface{} // tableName -> sample rows (3 rows)
	CachedAt  time.Time
}

const schemaCacheTTL = 5 * time.Minute // Cache expires after 5 minutes

// DataSourceService handles data source operations
type DataSourceService struct {
	dataCacheDir string
	Log          func(string)

	// Schema cache for performance
	schemaCache  map[string]*SchemaCache
	cacheMu      sync.RWMutex
}

// NewDataSourceService creates a new service instance
func NewDataSourceService(dataCacheDir string, logFunc func(string)) *DataSourceService {
	return &DataSourceService{
		dataCacheDir: dataCacheDir,
		Log:          logFunc,
		schemaCache:  make(map[string]*SchemaCache),
	}
}

func (s *DataSourceService) log(msg string) {
	if s.Log != nil {
		s.Log(msg)
	}
}

// getSchemaFromCache returns cached schema if valid, nil otherwise
func (s *DataSourceService) getSchemaFromCache(dataSourceID string) *SchemaCache {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	cache, exists := s.schemaCache[dataSourceID]
	if !exists {
		return nil
	}

	// Check if cache has expired
	if time.Since(cache.CachedAt) > schemaCacheTTL {
		return nil
	}

	return cache
}

// updateSchemaCache updates or creates cache for a data source
func (s *DataSourceService) updateSchemaCache(dataSourceID string, cache *SchemaCache) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	cache.CachedAt = time.Now()
	s.schemaCache[dataSourceID] = cache
}

// InvalidateCache clears cache for a specific data source
func (s *DataSourceService) InvalidateCache(dataSourceID string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	delete(s.schemaCache, dataSourceID)
}

// InvalidateAllCache clears all cached schema data
func (s *DataSourceService) InvalidateAllCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.schemaCache = make(map[string]*SchemaCache)
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

// UpdateAnalysis updates the analysis information for a data source
func (s *DataSourceService) UpdateAnalysis(id string, analysis DataSourceAnalysis) error {
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	found := false
	for i := range sources {
		if sources[i].ID == id {
			sources[i].Analysis = &analysis
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("data source not found")
	}

	return s.SaveDataSources(sources)
}

// AddDataSource adds a new source to the registry
func (s *DataSourceService) AddDataSource(ds DataSource) error {
	s.log(fmt.Sprintf("AddDataSource: %s (Type: %s)", ds.Name, ds.Type))
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
	s.log(fmt.Sprintf("DeleteDataSource: %s", id))

	// Invalidate cache first
	s.InvalidateCache(id)

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

	// Remove data directory if it exists
	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		dirToRemove := filepath.Dir(dbPath)

		if strings.HasPrefix(dirToRemove, s.dataCacheDir) {
			_ = os.RemoveAll(dirToRemove)
		}
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
	s.log(fmt.Sprintf("ImportDataSource: %s (Driver: %s)", name, driverType))
	switch strings.ToLower(driverType) {
	case "excel":
		return s.ImportExcel(name, config.OriginalFile, headerGen)
	case "csv":
		return s.ImportCSV(name, config.OriginalFile, headerGen)
	case "json":
		return s.ImportJSON(name, config.OriginalFile, headerGen)
	case "mysql", "doris":
		return s.ImportRemoteDataSource(name, driverType, config)
	case "shopify":
		return s.ImportShopify(name, config)
	case "bigcommerce":
		return s.ImportBigCommerce(name, config)
	case "ebay":
		return s.ImportEbay(name, config)
	case "etsy":
		return s.ImportEtsy(name, config)
	case "jira":
		return s.ImportJira(name, config)
	case "postgresql":
		return nil, fmt.Errorf("postgresql driver not supported yet")
	default:
		return nil, fmt.Errorf("unsupported driver type: %s", driverType)
	}
}

// ImportRemoteDataSource handles importing or linking to remote databases like MySQL
func (s *DataSourceService) ImportRemoteDataSource(name string, driverType string, config DataSourceConfig) (*DataSource, error) {
	// 1. Validate connection
	if config.Port == "" {
		config.Port = "3306"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", config.User, config.Password, config.Host, config.Port, config.Database)
	
	// Assuming mysql compatible
	remoteDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %v", err)
	}
	defer remoteDB.Close()

	if err := remoteDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	id := uuid.New().String()
	
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      strings.ToLower(driverType),
		CreatedAt: time.Now().UnixMilli(),
		Config:    config,
	}

	// 2. If StoreLocally, import data
	if config.StoreLocally {
		// Create local storage
		relDBDir := filepath.Join("sources", id)
		absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
		if err := os.MkdirAll(absDBDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %v", err)
		}

		dbName := "data.db"
		relDBPath := filepath.Join(relDBDir, dbName)
		absDBPath := filepath.Join(absDBDir, dbName)

		localDB, err := sql.Open("sqlite", absDBPath)
		if err != nil {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("failed to create local database: %v", err)
		}
		defer localDB.Close()

		// Get all tables
		rows, err := remoteDB.Query("SHOW TABLES")
		if err != nil {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("failed to list tables: %v", err)
		}
		defer rows.Close()

		var tables []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				continue
			}
			tables = append(tables, tableName)
		}

		if len(tables) == 0 {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("no tables found in database")
		}

		for _, tableName := range tables {
			if err := s.copyTable(remoteDB, localDB, tableName); err != nil {
				// Log error but continue with other tables? Or fail? 
				// For now, let's fail to ensure integrity
				_ = os.RemoveAll(absDBDir)
				return nil, fmt.Errorf("failed to copy table %s: %v", tableName, err)
			}
		}

		ds.Config.DBPath = relDBPath
	}

	// 3. Save to Registry
	if err := s.AddDataSource(ds); err != nil {
		return nil, err
	}

	return &ds, nil
}

// copyTable copies a single table from remote MySQL to local SQLite
func (s *DataSourceService) copyTable(remoteDB *sql.DB, localDB *sql.DB, tableName string) error {
	// Get data
	rows, err := remoteDB.Query(fmt.Sprintf("SELECT * FROM `%s`", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Create table in SQLite
	// We'll treat everything as TEXT or try basic inference? 
	// For simplicity and robustness, TEXT is safest for caching unless we strictly map types.
	// But let's try to be a bit smarter or just use TEXT for now to avoid type mismatch issues between MySQL and SQLite.
	createCols := []string{}
	for _, col := range cols {
		colName := s.sanitizeName(col)
		createCols = append(createCols, fmt.Sprintf("`%s` TEXT", colName))
	}

	safeTableName := s.sanitizeName(tableName)
	createSQL := fmt.Sprintf("CREATE TABLE `%s` (%s)", safeTableName, strings.Join(createCols, ", "))
	if _, err := localDB.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table schema: %v", err)
	}

	// Insert data
	placeholders := make([]string, len(cols))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s` VALUES (%s)", safeTableName, strings.Join(placeholders, ","))
	
	tx, err := localDB.Begin()
	if err != nil {
		return err
	}
	
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			tx.Rollback()
			return err
		}

		// Convert to string/interface safe for SQLite
		insertValues := make([]interface{}, len(cols))
		for i, val := range values {
			if val == nil {
				insertValues[i] = nil
			} else {
				switch v := val.(type) {
				case []byte:
					insertValues[i] = string(v)
				default:
					insertValues[i] = v
				}
			}
		}

		if _, err := stmt.Exec(insertValues...); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// sanitizeName makes a string safe for use as a table or column name
func (s *DataSourceService) sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	
	// Build a new name, preserving Unicode characters (including Chinese)
	// Only replace characters that are problematic for SQL identifiers
	var result strings.Builder
	for _, r := range name {
		// Allow:
		// - ASCII letters and numbers
		// - Underscore
		// - Unicode letters and numbers (including Chinese characters)
		// Replace only: spaces, quotes, backticks, and other SQL special characters
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		} else if r > 127 { // Unicode character (including Chinese)
			// Check if it's a valid Unicode letter or number
			// This preserves Chinese, Japanese, Korean, etc.
			result.WriteRune(r)
		} else if r == ' ' || r == '-' {
			// Replace spaces and hyphens with underscore
			result.WriteRune('_')
		} else {
			// Replace other special characters with underscore
			result.WriteRune('_')
		}
	}
	
	name = result.String()
	if name == "" {
		return "unknown"
	}
	return name
}

// processSheet handles the schema inference and data import for a single sheet
func (s *DataSourceService) processSheet(db *sql.DB, tableName string, rows [][]string, headerGen func(string) (string, error)) error {
	if len(rows) == 0 {
		return fmt.Errorf("no rows to process")
	}

	// Determine the maximum number of columns
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Identify valid columns (columns with meaningful data)
	// A column is considered valid if:
	// 1. It has a non-empty header (if hasHeader is true), OR
	// 2. It has at least one non-empty data value
	hasHeader := s.isHeaderRow(rows[0])
	validColumns := make([]bool, maxCols)
	
	for colIdx := 0; colIdx < maxCols; colIdx++ {
		hasData := false
		
		// Check header row if it exists
		if hasHeader && colIdx < len(rows[0]) {
			header := strings.TrimSpace(rows[0][colIdx])
			if header != "" {
				hasData = true
			}
		}
		
		// Check data rows for non-empty values
		startRow := 0
		if hasHeader {
			startRow = 1
		}
		
		for rowIdx := startRow; rowIdx < len(rows) && rowIdx < startRow+20; rowIdx++ {
			if colIdx < len(rows[rowIdx]) {
				cellValue := strings.TrimSpace(rows[rowIdx][colIdx])
				if cellValue != "" {
					hasData = true
					break
				}
			}
		}
		
		validColumns[colIdx] = hasData
	}
	
	// Filter rows to only include valid columns
	filteredRows := make([][]string, len(rows))
	for rowIdx, row := range rows {
		filteredRow := []string{}
		for colIdx := 0; colIdx < len(row); colIdx++ {
			if colIdx < len(validColumns) && validColumns[colIdx] {
				filteredRow = append(filteredRow, row[colIdx])
			}
		}
		filteredRows[rowIdx] = filteredRow
	}
	
	// Update rows to use filtered data
	rows = filteredRows
	
	// Schema Inference and Table Creation
	var headers []string
	var dataStartRow int

	// Determine column types using the first few data rows
	numCols := 0
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	
	if numCols == 0 {
		return fmt.Errorf("no valid columns found in sheet")
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
// Supports both .xlsx (Excel 2007+) and .xls (Excel 97-2003) formats
func (s *DataSourceService) ImportExcel(name string, filePath string, headerGen func(string) (string, error)) (*DataSource, error) {
	// 1. Validate file
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// 2. Check file extension and route to appropriate handler
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".xls":
		// Use xls library for old Excel format
		return s.importXLS(name, filePath, headerGen)
	case ".xlsx", ".xlsm":
		// Use excelize for new Excel format
		return s.importXLSX(name, filePath, headerGen)
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s。请使用 .xlsx 或 .xls 格式的 Excel 文件", ext)
	}
}

// importXLSX processes .xlsx files using excelize library
func (s *DataSourceService) importXLSX(name string, filePath string, headerGen func(string) (string, error)) (*DataSource, error) {
	// Open Excel
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		// Provide more helpful error message
		if strings.Contains(err.Error(), "unsupported workbook file format") {
			return nil, fmt.Errorf("无法打开 Excel 文件：文件格式不受支持。请确保文件是有效的 .xlsx 格式（Excel 2007 或更高版本）")
		}
		return nil, fmt.Errorf("failed to open excel file: %v", err)
	}
	defer f.Close()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("no sheets found in excel file")
	}

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

	// Process Sheets
	processedSheets := 0
	var mainTableName string
	usedTableNames := make(map[string]bool)

	for _, sheetName := range sheetList {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}
		if len(rows) == 0 {
			continue
		}

		tableName := s.sanitizeName(sheetName)
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
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("no valid data found in any sheet")
	}

	// Save to Registry
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "excel",
		CreatedAt: time.Now().UnixMilli(),
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

// importXLS processes .xls files (Excel 97-2003 format) using extrame/xls library
func (s *DataSourceService) importXLS(name string, filePath string, headerGen func(string) (string, error)) (*DataSource, error) {
	// Open XLS file
	xlsFile, err := xls.Open(filePath, "utf-8")
	if err != nil {
		return nil, fmt.Errorf("无法打开 Excel 文件: %v", err)
	}

	if xlsFile.NumSheets() == 0 {
		return nil, fmt.Errorf("Excel 文件中没有找到工作表")
	}

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

	// Process Sheets
	processedSheets := 0
	var mainTableName string
	usedTableNames := make(map[string]bool)

	for i := 0; i < xlsFile.NumSheets(); i++ {
		sheet := xlsFile.GetSheet(i)
		if sheet == nil {
			continue
		}

		sheetName := sheet.Name
		if sheetName == "" {
			sheetName = fmt.Sprintf("Sheet%d", i+1)
		}

		// Convert xls sheet to rows format
		rows := s.xlsSheetToRows(sheet)
		if len(rows) == 0 {
			continue
		}

		tableName := s.sanitizeName(sheetName)
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
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("Excel 文件中没有找到有效数据")
	}

	// Save to Registry
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "excel",
		CreatedAt: time.Now().UnixMilli(),
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

// xlsSheetToRows converts an xls.WorkSheet to [][]string format
func (s *DataSourceService) xlsSheetToRows(sheet *xls.WorkSheet) [][]string {
	if sheet == nil {
		return nil
	}

	maxRow := int(sheet.MaxRow)
	if maxRow == 0 {
		return nil
	}

	var rows [][]string
	
	for rowIdx := 0; rowIdx <= maxRow; rowIdx++ {
		row := sheet.Row(rowIdx)
		if row == nil {
			continue
		}

		var rowData []string
		lastCol := row.LastCol()
		
		for colIdx := 0; colIdx <= lastCol; colIdx++ {
			cell := row.Col(colIdx)
			rowData = append(rowData, cell)
		}

		// Skip completely empty rows
		hasData := false
		for _, cell := range rowData {
			if strings.TrimSpace(cell) != "" {
				hasData = true
				break
			}
		}
		
		if hasData {
			rows = append(rows, rowData)
		}
	}

	return rows
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
		CreatedAt: time.Now().UnixMilli(),
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
	// Check cache first
	if cache := s.getSchemaFromCache(id); cache != nil && len(cache.Tables) > 0 {
		s.log(fmt.Sprintf("[CACHE HIT] GetDataSourceTables for %s", id))
		return cache.Tables, nil
	}
	s.log(fmt.Sprintf("[CACHE MISS] GetDataSourceTables for %s", id))

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

	var tables []string

	// If DBPath exists, use local SQLite
	if target.Config.DBPath != "" {
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

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			tables = append(tables, name)
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		// Remote DB
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		rows, err := db.Query("SHOW TABLES")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			tables = append(tables, name)
		}
	} else {
		return nil, fmt.Errorf("data source has no local storage and is not a supported remote type")
	}

	// Update cache with tables (initialize cache if needed)
	cache := s.getSchemaFromCache(id)
	if cache == nil {
		cache = &SchemaCache{
			Columns: make(map[string][]string),
			Samples: make(map[string][]map[string]interface{}),
		}
	}
	cache.Tables = tables
	s.updateSchemaCache(id, cache)

	return tables, nil
}

// GetDataSourceTableData returns preview data for a table
func (s *DataSourceService) GetDataSourceTableData(id string, tableName string, limit int) ([]map[string]interface{}, error) {
	// Only cache small sample requests (≤10 rows) to avoid memory bloat
	const sampleCacheLimit = 10
	useCache := limit > 0 && limit <= sampleCacheLimit

	// Check cache first for sample data
	if useCache {
		if cache := s.getSchemaFromCache(id); cache != nil {
			if samples, exists := cache.Samples[tableName]; exists && len(samples) > 0 {
				s.log(fmt.Sprintf("[CACHE HIT] GetDataSourceTableData for %s.%s (limit=%d)", id, tableName, limit))
				// Return min(cached, requested) rows
				if len(samples) >= limit {
					return samples[:limit], nil
				}
				return samples, nil
			}
		}
	}
	s.log(fmt.Sprintf("[CACHE MISS] GetDataSourceTableData for %s.%s (limit=%d)", id, tableName, limit))

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

	var db *sql.DB

	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported data source type for direct query")
	}
	defer db.Close()

	if limit <= 0 {
		limit = 100 // Safe default
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
			if val != nil {
				// Handle []byte for MySQL text columns
				if b, ok := (*val).([]byte); ok {
					rowMap[colName] = string(b)
				} else {
					rowMap[colName] = *val
				}
			} else {
				rowMap[colName] = nil
			}
		}
		data = append(data, rowMap)
	}

	// Update cache with sample data (only for small limits)
	if useCache && len(data) > 0 {
		cache := s.getSchemaFromCache(id)
		if cache == nil {
			cache = &SchemaCache{
				Columns: make(map[string][]string),
				Samples: make(map[string][]map[string]interface{}),
			}
		}
		if cache.Samples == nil {
			cache.Samples = make(map[string][]map[string]interface{})
		}
		cache.Samples[tableName] = data
		s.updateSchemaCache(id, cache)
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

	var db *sql.DB

	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return 0, err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("unsupported data source type for count")
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

// GetDataSourceTableColumns returns the column names for a table
func (s *DataSourceService) GetDataSourceTableColumns(id string, tableName string) ([]string, error) {
	// Check cache first
	if cache := s.getSchemaFromCache(id); cache != nil {
		if cols, exists := cache.Columns[tableName]; exists && len(cols) > 0 {
			s.log(fmt.Sprintf("[CACHE HIT] GetDataSourceTableColumns for %s.%s", id, tableName))
			return cols, nil
		}
	}
	s.log(fmt.Sprintf("[CACHE MISS] GetDataSourceTableColumns for %s.%s", id, tableName))

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

	var db *sql.DB

	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported data source type for columns")
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT 0", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Update cache with columns
	cache := s.getSchemaFromCache(id)
	if cache == nil {
		cache = &SchemaCache{
			Columns: make(map[string][]string),
			Samples: make(map[string][]map[string]interface{}),
		}
	}
	if cache.Columns == nil {
		cache.Columns = make(map[string][]string)
	}
	cache.Columns[tableName] = cols
	s.updateSchemaCache(id, cache)

	return cols, nil
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

	var db *sql.DB
	
	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("data source storage not found")
	}
	defer db.Close()

	// 2. Export each table
	// Create directory named after data source in the target directory (parent of outputPath)
	parentDir := filepath.Dir(outputPath)
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

	var db *sql.DB
	
	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("data source storage not found")
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
	s.log(fmt.Sprintf("ExportToMySQL: Source=%s, Tables=%v, TargetDB=%s", id, tableNames, mysqlConfig.Database))
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

	// Check if source and target are the same (only for remote sources)
	if (target.Type == "mysql" || target.Type == "doris") {
		srcHost := strings.TrimSpace(target.Config.Host)
		if srcHost == "" { srcHost = "localhost" }
		if srcHost == "127.0.0.1" || srcHost == "::1" { srcHost = "localhost" }

		srcPort := strings.TrimSpace(target.Config.Port)
		if srcPort == "" { srcPort = "3306" }
		
		dstHost := strings.TrimSpace(mysqlConfig.Host)
		if dstHost == "" { dstHost = "localhost" }
		if dstHost == "127.0.0.1" || dstHost == "::1" { dstHost = "localhost" }

		dstPort := strings.TrimSpace(mysqlConfig.Port)
		if dstPort == "" { dstPort = "3306" }

		srcDb := strings.TrimSpace(target.Config.Database)
		dstDb := strings.TrimSpace(mysqlConfig.Database)

		if strings.EqualFold(srcHost, dstHost) && 
		   srcPort == dstPort && 
		   strings.EqualFold(srcDb, dstDb) {
			s.log("Error: Export source and target are same")
			return fmt.Errorf("cannot export to the same database as the source")
		}
	}

	var sourceDB *sql.DB

	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		sourceDB, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		sourceDB, err = sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("data source storage not found")
	}
	defer sourceDB.Close()

	// 2. Connect to MySQL and Create Database if not exists
	if mysqlConfig.Port == "" {
		mysqlConfig.Port = "3306"
	}
	
	// First connect without database to create it
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?allowNativePasswords=true", mysqlConfig.User, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port)
	rootDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to mysql server: %v", err)
	}
	defer rootDB.Close()

	if err := rootDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping mysql server: %v", err)
	}

	// Create database
	createDbSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", mysqlConfig.Database)
	s.log(fmt.Sprintf("SQL Exec: %s", createDbSQL))
	_, err = rootDB.Exec(createDbSQL)
	if err != nil {
		return fmt.Errorf("failed to create database %s: %v", mysqlConfig.Database, err)
	}
	rootDB.Close()

	// Now connect to the specific database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", mysqlConfig.User, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.Database)
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to mysql database: %v", err)
	}
	defer mysqlDB.Close()

	if err := mysqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping mysql database: %v", err)
	}

	// 3. Export each table
	for _, tableName := range tableNames {
		s.log(fmt.Sprintf("Exporting table: %s", tableName))
		if err := s.exportSingleTableToMySQL(sourceDB, mysqlDB, tableName); err != nil {
			s.log(fmt.Sprintf("Export failed for table %s: %v", tableName, err))
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
	s.log(fmt.Sprintf("SQL Exec: %s", createSQL))
	if _, err := mysqlDB.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table in mysql: %v", err)
	}

	// Bulk Insert
	placeholders := make([]string, len(cols))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES (%s)", tableName, strings.Join(cols, "`, `"), strings.Join(placeholders, ", "))
	// Don't log full insert SQL with all data, just the template or first batch
	s.log(fmt.Sprintf("SQL Prepare: %s", insertSQL))
	
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

	rowCount := 0
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
		rowCount++
	}
	s.log(fmt.Sprintf("Inserted %d rows into %s", rowCount, tableName))

	return nil
}


// TestMySQLConnection tests the connection to a MySQL server
func (s *DataSourceService) TestMySQLConnection(host, port, user, password string) error {
	if port == "" {
		port = "3306"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?allowNativePasswords=true", user, password, host, port)
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?allowNativePasswords=true", user, password, host, port)
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

// ExecuteSQL runs a SQL query against a data source and returns results
func (s *DataSourceService) ExecuteSQL(id string, query string) ([]map[string]interface{}, error) {
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
		return nil, fmt.Errorf("data source not found: %s", id)
	}

	var db *sql.DB

	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported data source type for direct query")
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		result = append(result, row)
	}

	return result, nil
}


// RenameDataSource renames a data source (checks for duplicate names)
func (s *DataSourceService) RenameDataSource(id string, newName string) error {
	s.log(fmt.Sprintf("RenameDataSource: %s -> %s", id, newName))
	
	if newName == "" {
		return fmt.Errorf("data source name cannot be empty")
	}
	
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	// Check for duplicate name (case-insensitive)
	for _, source := range sources {
		if source.ID != id && strings.EqualFold(source.Name, newName) {
			return fmt.Errorf("data source with name '%s' already exists", newName)
		}
	}

	// Find and rename the data source
	found := false
	for i := range sources {
		if sources[i].ID == id {
			sources[i].Name = newName
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("data source not found")
	}

	// Invalidate cache
	s.InvalidateCache(id)

	return s.SaveDataSources(sources)
}

// DeleteTable removes a table from a data source and updates the schema information
func (s *DataSourceService) DeleteTable(id string, tableName string) error {
	s.log(fmt.Sprintf("DeleteTable: DataSource=%s, Table=%s", id, tableName))
	
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	var target *DataSource
	for i := range sources {
		if sources[i].ID == id {
			target = &sources[i]
			break
		}
	}

	if target == nil {
		return fmt.Errorf("data source not found")
	}

	var db *sql.DB

	// Connect to the database
	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %v", err)
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %v", err)
		}
	} else {
		return fmt.Errorf("unsupported data source type for table deletion")
	}
	defer db.Close()

	// Drop the table
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
	s.log(fmt.Sprintf("SQL Exec: %s", dropSQL))
	if _, err := db.Exec(dropSQL); err != nil {
		return fmt.Errorf("failed to drop table: %v", err)
	}

	// Update the schema information in the data source analysis
	if target.Analysis != nil && target.Analysis.Schema != nil {
		newSchema := []TableSchema{}
		for _, table := range target.Analysis.Schema {
			if table.TableName != tableName {
				newSchema = append(newSchema, table)
			}
		}
		target.Analysis.Schema = newSchema
		
		// Save the updated data source
		if err := s.SaveDataSources(sources); err != nil {
			return fmt.Errorf("failed to update data source schema: %v", err)
		}
	}

	// Invalidate cache
	s.InvalidateCache(id)

	s.log(fmt.Sprintf("Table %s deleted successfully from data source %s", tableName, id))
	return nil
}

// GetTables 获取数据源的所有表名（别名方法）
func (s *DataSourceService) GetTables(id string) ([]string, error) {
	return s.GetDataSourceTables(id)
}

// GetTableColumns 获取表的列信息
func (s *DataSourceService) GetTableColumns(id string, tableName string) ([]ColumnSchema, error) {
	sources, err := s.LoadDataSources()
	if err != nil {
		return nil, err
	}

	var ds *DataSource
	for _, source := range sources {
		if source.ID == id {
			ds = &source
			break
		}
	}

	if ds == nil {
		return nil, fmt.Errorf("data source not found: %s", id)
	}

	var db *sql.DB
	// Determine if this is a local SQLite-backed data source
	// Excel, CSV, JSON types are stored locally in SQLite
	isLocalSQLite := ds.Config.DBPath != "" && (ds.Type == "sqlite" || ds.Type == "excel" || ds.Type == "csv" || ds.Type == "json")
	
	if ds.Type == "mysql" || ds.Type == "postgresql" || ds.Type == "doris" {
		// Remote database - build connection string
		var connStr string
		if ds.Type == "mysql" || ds.Type == "doris" {
			connStr = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true",
				ds.Config.User, ds.Config.Password, ds.Config.Host, ds.Config.Port, ds.Config.Database)
		} else if ds.Type == "postgresql" {
			connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
				ds.Config.Host, ds.Config.Port, ds.Config.User, ds.Config.Password, ds.Config.Database)
		} else {
			return nil, fmt.Errorf("unsupported database type: %s", ds.Type)
		}
		driverName := ds.Type
		if ds.Type == "doris" {
			driverName = "mysql" // Doris uses MySQL protocol
		}
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()
	} else if isLocalSQLite {
		// Local SQLite-backed data source (sqlite, excel, csv, json)
		fullDBPath := filepath.Join(s.dataCacheDir, ds.Config.DBPath)
		db, err = sql.Open("sqlite", fullDBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()
	} else {
		return nil, fmt.Errorf("unsupported database type or missing DBPath: %s", ds.Type)
	}

	// Query column information
	var rows *sql.Rows
	if isLocalSQLite {
		// All local SQLite-backed sources use PRAGMA
		rows, err = db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	} else if ds.Type == "mysql" || ds.Type == "doris" {
		rows, err = db.Query(fmt.Sprintf("DESCRIBE `%s`", tableName))
	} else {
		return nil, fmt.Errorf("unsupported database type: %s", ds.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query table info: %w", err)
	}
	defer rows.Close()

	var columns []ColumnSchema
	if isLocalSQLite {
		// All local SQLite-backed sources (sqlite, excel, csv, json) use PRAGMA format
		for rows.Next() {
			var cid int
			var name, colType string
			var notNull int
			var dfltValue sql.NullString
			var pk int

			err = rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
			if err != nil {
				return nil, err
			}

			columns = append(columns, ColumnSchema{
				Name:     name,
				Type:     colType,
				Nullable: notNull == 0,
			})
		}
	} else if ds.Type == "mysql" || ds.Type == "doris" {
		for rows.Next() {
			var field, colType, null, key, extra string
			var dflt sql.NullString

			err = rows.Scan(&field, &colType, &null, &key, &dflt, &extra)
			if err != nil {
				return nil, err
			}

			columns = append(columns, ColumnSchema{
				Name:     field,
				Type:     colType,
				Nullable: null == "YES",
			})
		}
	}

	return columns, nil
}

// ColumnSchema 列结构信息
type ColumnSchema struct {
	Name     string
	Type     string
	Nullable bool
}

// ExecuteQuery 执行查询并返回结果
func (s *DataSourceService) ExecuteQuery(id string, query string) ([]map[string]any, error) {
	return s.ExecuteSQL(id, query)
}

// GetConnection 获取数据库连接
func (s *DataSourceService) GetConnection(id string) (*sql.DB, error) {
	sources, err := s.LoadDataSources()
	if err != nil {
		return nil, err
	}

	var ds *DataSource
	for _, source := range sources {
		if source.ID == id {
			ds = &source
			break
		}
	}

	if ds == nil {
		return nil, fmt.Errorf("data source not found: %s", id)
	}

	var db *sql.DB
	if ds.Type == "mysql" || ds.Type == "postgresql" || ds.Type == "doris" {
		// Remote database - build connection string
		var connStr string
		if ds.Type == "mysql" {
			connStr = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true",
				ds.Config.User, ds.Config.Password, ds.Config.Host, ds.Config.Port, ds.Config.Database)
		} else if ds.Type == "postgresql" {
			connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
				ds.Config.Host, ds.Config.Port, ds.Config.User, ds.Config.Password, ds.Config.Database)
		} else {
			return nil, fmt.Errorf("unsupported database type: %s", ds.Type)
		}
		db, err = sql.Open(ds.Type, connStr)
	} else {
		dbPath := ds.Config.DBPath
		if dbPath == "" {
			return nil, fmt.Errorf("database path not found in config")
		}
		// Use full path by joining with dataCacheDir, and use "sqlite" driver (modernc.org/sqlite)
		fullDBPath := filepath.Join(s.dataCacheDir, dbPath)
		s.log(fmt.Sprintf("[GetConnection] Opening SQLite database: %s", fullDBPath))
		db, err = sql.Open("sqlite", fullDBPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// CreateOptimizedDatabase 创建优化后的数据库文件
// 返回新数据库的完整路径和相对路径（用于存储在 DBPath 中）
func (s *DataSourceService) CreateOptimizedDatabase(originalSource *DataSource, newName string) (string, error) {
	// 生成新的数据源 ID
	id := uuid.New().String()
	
	// 创建数据源目录：sources/{id}
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	// 数据库文件名为 data.db
	dbName := "data.db"
	absDBPath := filepath.Join(absDBDir, dbName)

	// 创建空数据库
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return "", fmt.Errorf("failed to create database: %w", err)
	}
	db.Close()

	s.log(fmt.Sprintf("[CreateOptimizedDatabase] Created database at: %s", absDBPath))

	return absDBPath, nil
}

// SaveDataSource 保存单个数据源
func (s *DataSourceService) SaveDataSource(ds *DataSource) error {
	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	// 检查是否已存在
	found := false
	for i, source := range sources {
		if source.ID == ds.ID {
			sources[i] = *ds
			found = true
			break
		}
	}

	if !found {
		sources = append(sources, *ds)
	}

	return s.SaveDataSources(sources)
}

// sanitizeFilename 清理文件名
func sanitizeFilename(name string) string {
	// 移除或替换不安全的字符
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	return name
}

// RenameColumn renames a column in a table and updates the schema information
func (s *DataSourceService) RenameColumn(id string, tableName string, oldColumnName string, newColumnName string) error {
	s.log(fmt.Sprintf("RenameColumn: DataSource=%s, Table=%s, OldColumn=%s, NewColumn=%s", id, tableName, oldColumnName, newColumnName))

	// Validate new column name
	if newColumnName == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	// Check for invalid characters in column name
	invalidChars := []string{" ", "'", "\"", ";", "--", "/*", "*/", "\t", "\n", "\r"}
	for _, char := range invalidChars {
		if strings.Contains(newColumnName, char) {
			return fmt.Errorf("column name contains invalid character: %s", char)
		}
	}

	// Check if column name starts with a number
	if len(newColumnName) > 0 && newColumnName[0] >= '0' && newColumnName[0] <= '9' {
		return fmt.Errorf("column name cannot start with a number")
	}

	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	var target *DataSource
	var targetIndex int
	for i := range sources {
		if sources[i].ID == id {
			target = &sources[i]
			targetIndex = i
			break
		}
	}

	if target == nil {
		return fmt.Errorf("data source not found")
	}

	// Check if new column name already exists in the table schema
	if target.Analysis != nil && target.Analysis.Schema != nil {
		for _, table := range target.Analysis.Schema {
			if table.TableName == tableName {
				for _, col := range table.Columns {
					if col == newColumnName && col != oldColumnName {
						return fmt.Errorf("column name '%s' already exists in table '%s'", newColumnName, tableName)
					}
				}
				break
			}
		}
	}

	var db *sql.DB

	// Connect to the database
	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %v", err)
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %v", err)
		}
	} else {
		return fmt.Errorf("unsupported data source type for column rename")
	}
	defer db.Close()

	// Rename the column based on database type
	// Use DBPath to determine if it's SQLite (local file) vs MySQL/Doris (remote)
	isSQLite := target.Config.DBPath != ""
	
	if !isSQLite && (target.Type == "mysql" || target.Type == "doris") {
		// For MySQL/Doris, we need to get the column type first
		var colType string
		query := fmt.Sprintf("SELECT COLUMN_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = '%s' AND COLUMN_NAME = '%s'", tableName, oldColumnName)
		err = db.QueryRow(query).Scan(&colType)
		if err != nil {
			return fmt.Errorf("failed to get column type: %v", err)
		}
		renameSQL := fmt.Sprintf("ALTER TABLE `%s` CHANGE `%s` `%s` %s", tableName, oldColumnName, newColumnName, colType)
		s.log(fmt.Sprintf("SQL Exec: %s", renameSQL))
		if _, err := db.Exec(renameSQL); err != nil {
			return fmt.Errorf("failed to rename column: %v", err)
		}
	} else {
		// For SQLite, use ALTER TABLE RENAME COLUMN (SQLite 3.25.0+)
		renameSQL := fmt.Sprintf("ALTER TABLE `%s` RENAME COLUMN `%s` TO `%s`", tableName, oldColumnName, newColumnName)
		s.log(fmt.Sprintf("SQL Exec: %s", renameSQL))
		if _, err := db.Exec(renameSQL); err != nil {
			return fmt.Errorf("failed to rename column: %v", err)
		}
	}

	// Update the schema information in the data source analysis
	if target.Analysis != nil && target.Analysis.Schema != nil {
		for i, table := range target.Analysis.Schema {
			if table.TableName == tableName {
				for j, col := range table.Columns {
					if col == oldColumnName {
						sources[targetIndex].Analysis.Schema[i].Columns[j] = newColumnName
						break
					}
				}
				break
			}
		}

		// Save the updated data source
		if err := s.SaveDataSources(sources); err != nil {
			return fmt.Errorf("failed to update data source schema: %v", err)
		}
	}

	// Invalidate cache
	s.InvalidateCache(id)

	s.log(fmt.Sprintf("Column %s renamed to %s in table %s of data source %s", oldColumnName, newColumnName, tableName, id))
	return nil
}

// DeleteColumn deletes a column from a table and updates the schema information
func (s *DataSourceService) DeleteColumn(id string, tableName string, columnName string) error {
	s.log(fmt.Sprintf("DeleteColumn: DataSource=%s, Table=%s, Column=%s", id, tableName, columnName))

	if columnName == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	sources, err := s.LoadDataSources()
	if err != nil {
		return err
	}

	var target *DataSource
	var targetIndex int
	for i := range sources {
		if sources[i].ID == id {
			target = &sources[i]
			targetIndex = i
			break
		}
	}

	if target == nil {
		return fmt.Errorf("data source not found")
	}

	// Check if column exists in the table schema
	columnExists := false
	columnCount := 0
	if target.Analysis != nil && target.Analysis.Schema != nil {
		for _, table := range target.Analysis.Schema {
			if table.TableName == tableName {
				columnCount = len(table.Columns)
				for _, col := range table.Columns {
					if col == columnName {
						columnExists = true
						break
					}
				}
				break
			}
		}
	}

	if !columnExists {
		return fmt.Errorf("column '%s' not found in table '%s'", columnName, tableName)
	}

	// Prevent deleting the last column
	if columnCount <= 1 {
		return fmt.Errorf("cannot delete the last column in table '%s'", tableName)
	}

	var db *sql.DB

	// Connect to the database
	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %v", err)
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %v", err)
		}
	} else {
		return fmt.Errorf("unsupported data source type for column deletion")
	}
	defer db.Close()

	// Delete the column based on database type
	isSQLite := target.Config.DBPath != ""

	if !isSQLite && (target.Type == "mysql" || target.Type == "doris") {
		// For MySQL/Doris, use ALTER TABLE DROP COLUMN
		dropSQL := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", tableName, columnName)
		s.log(fmt.Sprintf("SQL Exec: %s", dropSQL))
		if _, err := db.Exec(dropSQL); err != nil {
			return fmt.Errorf("failed to delete column: %v", err)
		}
	} else {
		// For SQLite, we need to recreate the table without the column
		// SQLite doesn't support DROP COLUMN directly (before 3.35.0)
		// We'll use the table recreation approach for compatibility

		// Get all columns except the one to delete
		var remainingColumns []string
		if target.Analysis != nil && target.Analysis.Schema != nil {
			for _, table := range target.Analysis.Schema {
				if table.TableName == tableName {
					for _, col := range table.Columns {
						if col != columnName {
							remainingColumns = append(remainingColumns, fmt.Sprintf("`%s`", col))
						}
					}
					break
				}
			}
		}

		if len(remainingColumns) == 0 {
			return fmt.Errorf("no remaining columns after deletion")
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		// Create temp table with remaining columns
		columnsStr := strings.Join(remainingColumns, ", ")
		tempTableName := tableName + "_temp_delete_col"

		createTempSQL := fmt.Sprintf("CREATE TABLE `%s` AS SELECT %s FROM `%s`", tempTableName, columnsStr, tableName)
		s.log(fmt.Sprintf("SQL Exec: %s", createTempSQL))
		if _, err := tx.Exec(createTempSQL); err != nil {
			return fmt.Errorf("failed to create temp table: %v", err)
		}

		// Drop original table
		dropSQL := fmt.Sprintf("DROP TABLE `%s`", tableName)
		s.log(fmt.Sprintf("SQL Exec: %s", dropSQL))
		if _, err := tx.Exec(dropSQL); err != nil {
			return fmt.Errorf("failed to drop original table: %v", err)
		}

		// Rename temp table to original name
		renameSQL := fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`", tempTableName, tableName)
		s.log(fmt.Sprintf("SQL Exec: %s", renameSQL))
		if _, err := tx.Exec(renameSQL); err != nil {
			return fmt.Errorf("failed to rename temp table: %v", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}
	}

	// Update the schema information in the data source analysis
	if target.Analysis != nil && target.Analysis.Schema != nil {
		for i, table := range target.Analysis.Schema {
			if table.TableName == tableName {
				newColumns := make([]string, 0, len(table.Columns)-1)
				for _, col := range table.Columns {
					if col != columnName {
						newColumns = append(newColumns, col)
					}
				}
				sources[targetIndex].Analysis.Schema[i].Columns = newColumns
				break
			}
		}

		// Save the updated data source
		if err := s.SaveDataSources(sources); err != nil {
			return fmt.Errorf("failed to update data source schema: %v", err)
		}
	}

	// Invalidate cache
	s.InvalidateCache(id)

	s.log(fmt.Sprintf("Column %s deleted from table %s of data source %s", columnName, tableName, id))
	return nil
}

// ColumnInfoWithType represents column metadata with type information
type ColumnInfoWithType struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// GetDataSourceTableColumnsWithTypes returns column names and types for a table
func (s *DataSourceService) GetDataSourceTableColumnsWithTypes(id string, tableName string) ([]ColumnInfoWithType, error) {
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

	var db *sql.DB

	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported data source type for columns")
	}
	defer db.Close()

	var columns []ColumnInfoWithType

	if target.Config.DBPath != "" {
		// SQLite: use PRAGMA table_info
		query := fmt.Sprintf("PRAGMA table_info(`%s`)", tableName)
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				return nil, err
			}
			columns = append(columns, ColumnInfoWithType{
				Name: name,
				Type: colType,
			})
		}
	} else {
		// MySQL: use DESCRIBE
		query := fmt.Sprintf("DESCRIBE `%s`", tableName)
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var field, colType string
			var null, key, extra string
			var dflt interface{}
			if err := rows.Scan(&field, &colType, &null, &key, &dflt, &extra); err != nil {
				return nil, err
			}
			columns = append(columns, ColumnInfoWithType{
				Name: field,
				Type: colType,
			})
		}
	}

	return columns, nil
}

// GetTableRowCount returns the number of rows in a table
func (s *DataSourceService) GetTableRowCount(id string, tableName string) (int, error) {
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

	var db *sql.DB

	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return 0, err
		}
	} else if target.Type == "mysql" || target.Type == "doris" {
		cfg := target.Config
		if cfg.Port == "" {
			cfg.Port = "3306"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?allowNativePasswords=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("unsupported data source type")
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", tableName)
	var count int
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ImportShopify imports data from Shopify API
func (s *DataSourceService) ImportShopify(name string, config DataSourceConfig) (*DataSource, error) {
	s.log(fmt.Sprintf("ImportShopify: %s (Store: %s)", name, config.ShopifyStore))

	// Validate configuration
	if config.ShopifyStore == "" || config.ShopifyAccessToken == "" {
		return nil, fmt.Errorf("shopify store URL and access token are required")
	}

	// Normalize store URL - remove protocol and trailing slashes
	store := config.ShopifyStore
	store = strings.TrimPrefix(store, "https://")
	store = strings.TrimPrefix(store, "http://")
	store = strings.TrimSuffix(store, "/")
	
	// Ensure it has .myshopify.com suffix
	if !strings.Contains(store, ".myshopify.com") {
		if !strings.Contains(store, ".") {
			store = store + ".myshopify.com"
		}
	}
	
	s.log(fmt.Sprintf("[SHOPIFY] Normalized store URL: %s", store))
	s.log(fmt.Sprintf("[SHOPIFY] Token length: %d", len(config.ShopifyAccessToken)))

	// Set default API version if not provided
	apiVersion := config.ShopifyAPIVersion
	if apiVersion == "" {
		apiVersion = "2024-01"
	}

	// Create data source ID
	id := uuid.New().String()

	// Create local storage directory
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create SQLite database
	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to create local database: %v", err)
	}
	defer db.Close()

	// Fetch and import Shopify data
	if err := s.fetchShopifyData(db, store, config.ShopifyAccessToken, apiVersion); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to fetch Shopify data: %v", err)
	}

	// Create data source object
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "shopify",
		CreatedAt: time.Now().UnixMilli(),
		Config: DataSourceConfig{
			DBPath:             relDBPath,
			ShopifyStore:       config.ShopifyStore,
			ShopifyAccessToken: config.ShopifyAccessToken,
			ShopifyAPIVersion:  apiVersion,
		},
	}

	// Save to registry
	if err := s.AddDataSource(ds); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, err
	}

	s.log(fmt.Sprintf("Shopify data source imported successfully: %s", name))
	return &ds, nil
}

// fetchShopifyData fetches data from Shopify API and stores it in SQLite
func (s *DataSourceService) fetchShopifyData(db *sql.DB, store, accessToken, apiVersion string) error {
	baseURL := fmt.Sprintf("https://%s/admin/api/%s", store, apiVersion)
	
	s.log(fmt.Sprintf("[SHOPIFY] Starting data fetch from %s (API version: %s)", store, apiVersion))
	s.log(fmt.Sprintf("[SHOPIFY] Token length: %d, prefix: %s...", len(accessToken), accessToken[:min(10, len(accessToken))]))
	
	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Fetch and store different Shopify resources
	// Note: inventory_items requires specific IDs, so we skip it
	// We focus on the most commonly used resources for BI analysis
	resources := []struct {
		name     string
		endpoint string
		key      string
	}{
		{"orders", "/orders.json?status=any&limit=250", "orders"},
		{"products", "/products.json?limit=250", "products"},
		{"customers", "/customers.json?limit=250", "customers"},
		{"collections", "/custom_collections.json?limit=250", "custom_collections"},
		{"smart_collections", "/smart_collections.json?limit=250", "smart_collections"},
	}

	importedCount := 0
	var lastError error
	for _, resource := range resources {
		s.log(fmt.Sprintf("[SHOPIFY] Fetching %s...", resource.name))
		
		if err := s.fetchShopifyResource(client, db, baseURL, resource.endpoint, resource.key, resource.name, accessToken); err != nil {
			s.log(fmt.Sprintf("[SHOPIFY] Warning: Failed to fetch %s: %v", resource.name, err))
			lastError = err
			// Continue with other resources even if one fails
		} else {
			importedCount++
			s.log(fmt.Sprintf("[SHOPIFY] Successfully fetched %s", resource.name))
		}
	}

	if importedCount == 0 {
		errMsg := "failed to import any Shopify data"
		if lastError != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, lastError)
		}
		s.log(fmt.Sprintf("[SHOPIFY] %s", errMsg))
		return fmt.Errorf(errMsg)
	}

	s.log(fmt.Sprintf("[SHOPIFY] Successfully imported %d resource types", importedCount))
	return nil
}

// fetchShopifyResource fetches a specific Shopify resource and stores it in a table
func (s *DataSourceService) fetchShopifyResource(client *http.Client, db *sql.DB, baseURL, endpoint, jsonKey, tableName, accessToken string) error {
	allData := []map[string]interface{}{}
	nextURL := baseURL + endpoint

	s.log(fmt.Sprintf("[SHOPIFY] Fetching %s from %s", tableName, nextURL))

	// Paginate through all results
	for nextURL != "" {
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("X-Shopify-Access-Token", accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			s.log(fmt.Sprintf("[SHOPIFY] Request failed for %s: %v", tableName, err))
			return fmt.Errorf("failed to fetch data: %v", err)
		}

		s.log(fmt.Sprintf("[SHOPIFY] Response status for %s: %d", tableName, resp.StatusCode))

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			s.log(fmt.Sprintf("[SHOPIFY] Error response for %s: %s", tableName, string(body)))
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %v", err)
		}

		// Get pagination link before closing body
		linkHeader := resp.Header.Get("Link")
		resp.Body.Close()

		// Extract data from response
		if data, ok := result[jsonKey].([]interface{}); ok {
			s.log(fmt.Sprintf("[SHOPIFY] Got %d items for %s", len(data), tableName))
			for _, item := range data {
				if itemMap, ok := item.(map[string]interface{}); ok {
					allData = append(allData, itemMap)
				}
			}
		} else {
			s.log(fmt.Sprintf("[SHOPIFY] No '%s' key in response for %s, keys: %v", jsonKey, tableName, getMapKeys(result)))
		}

		// Check for pagination link
		nextURL = s.extractNextLink(linkHeader)
	}

	if len(allData) == 0 {
		s.log(fmt.Sprintf("[SHOPIFY] No data found for %s", tableName))
		return nil
	}

	s.log(fmt.Sprintf("[SHOPIFY] Total %d items for %s, creating table...", len(allData), tableName))
	// Create table and insert data
	return s.createTableFromJSON(db, tableName, allData)
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// extractNextLink extracts the next page URL from Link header
func (s *DataSourceService) extractNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}

	// Parse Link header format: <url>; rel="next"
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}

	return ""
}

// createTableFromJSON creates a SQLite table from JSON data
func (s *DataSourceService) createTableFromJSON(db *sql.DB, tableName string, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Analyze first few rows to determine schema
	columns := make(map[string]interface{})
	for i := 0; i < min(10, len(data)); i++ {
		s.flattenJSON("", data[i], columns)
	}

	// Create table
	var colDefs []string
	var colNames []string
	for colName := range columns {
		colDefs = append(colDefs, fmt.Sprintf("`%s` TEXT", colName))
		colNames = append(colNames, colName)
	}

	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", tableName, strings.Join(colDefs, ", "))
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// Insert data
	quotedColNames := make([]string, len(colNames))
	for i, name := range colNames {
		quotedColNames[i] = fmt.Sprintf("`%s`", name)
	}
	placeholders := make([]string, len(colNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(quotedColNames, ", "),
		strings.Join(placeholders, ", "))

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %v", err)
	}
	defer stmt.Close()

	for _, row := range data {
		flatRow := make(map[string]interface{})
		s.flattenJSON("", row, flatRow)

		values := make([]interface{}, len(colNames))
		for i, colName := range colNames {
			if val, ok := flatRow[colName]; ok {
				values[i] = s.formatValue(val)
			} else {
				values[i] = nil
			}
		}

		if _, err := stmt.Exec(values...); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to insert row: %v", err))
		}
	}

	s.log(fmt.Sprintf("Imported %d rows into %s", len(data), tableName))
	return nil
}

// flattenJSON flattens nested JSON into dot-notation columns
func (s *DataSourceService) flattenJSON(prefix string, data interface{}, result map[string]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "_" + key
			}
			s.flattenJSON(newKey, val, result)
		}
	case []interface{}:
		// Store arrays as JSON strings
		if prefix != "" {
			jsonBytes, _ := json.Marshal(v)
			result[prefix] = string(jsonBytes)
		}
	default:
		if prefix != "" {
			result[prefix] = v
		}
	}
}

// formatValue converts a value to string for SQLite storage
func (s *DataSourceService) formatValue(val interface{}) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes)
	}
}


// ImportBigCommerce imports data from BigCommerce API
func (s *DataSourceService) ImportBigCommerce(name string, config DataSourceConfig) (*DataSource, error) {
	s.log(fmt.Sprintf("ImportBigCommerce: %s (Store: %s)", name, config.BigCommerceStoreHash))

	// Validate configuration
	if config.BigCommerceStoreHash == "" || config.BigCommerceAccessToken == "" {
		return nil, fmt.Errorf("bigcommerce store hash and access token are required")
	}

	// Create data source ID
	id := uuid.New().String()

	// Create local storage directory
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create SQLite database
	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to create local database: %v", err)
	}
	defer db.Close()

	// Fetch and import BigCommerce data
	if err := s.fetchBigCommerceData(db, config.BigCommerceStoreHash, config.BigCommerceAccessToken); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to fetch BigCommerce data: %v", err)
	}

	// Create data source object
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "bigcommerce",
		CreatedAt: time.Now().UnixMilli(),
		Config: DataSourceConfig{
			DBPath:                 relDBPath,
			BigCommerceStoreHash:   config.BigCommerceStoreHash,
			BigCommerceAccessToken: config.BigCommerceAccessToken,
		},
	}

	// Save to registry
	if err := s.AddDataSource(ds); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, err
	}

	s.log(fmt.Sprintf("BigCommerce data source imported successfully: %s", name))
	return &ds, nil
}

// fetchBigCommerceData fetches data from BigCommerce API and stores it in SQLite
func (s *DataSourceService) fetchBigCommerceData(db *sql.DB, storeHash, accessToken string) error {
	baseURL := fmt.Sprintf("https://api.bigcommerce.com/stores/%s/v3", storeHash)
	baseURLV2 := fmt.Sprintf("https://api.bigcommerce.com/stores/%s/v2", storeHash)

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Fetch and store different BigCommerce resources
	// V3 API resources
	resourcesV3 := []struct {
		name     string
		endpoint string
		key      string
	}{
		{"products", "/catalog/products?limit=250", "data"},
		{"categories", "/catalog/categories?limit=250", "data"},
		{"brands", "/catalog/brands?limit=250", "data"},
		{"customers", "/customers?limit=250", "data"},
	}

	// V2 API resources (orders use V2)
	resourcesV2 := []struct {
		name     string
		endpoint string
	}{
		{"orders", "/orders?limit=250"},
	}

	importedCount := 0

	// Fetch V3 resources
	for _, resource := range resourcesV3 {
		s.log(fmt.Sprintf("Fetching BigCommerce %s...", resource.name))

		if err := s.fetchBigCommerceResourceV3(client, db, baseURL, resource.endpoint, resource.key, resource.name, accessToken); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to fetch %s: %v", resource.name, err))
		} else {
			importedCount++
		}
	}

	// Fetch V2 resources
	for _, resource := range resourcesV2 {
		s.log(fmt.Sprintf("Fetching BigCommerce %s...", resource.name))

		if err := s.fetchBigCommerceResourceV2(client, db, baseURLV2, resource.endpoint, resource.name, accessToken); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to fetch %s: %v", resource.name, err))
		} else {
			importedCount++
		}
	}

	if importedCount == 0 {
		return fmt.Errorf("failed to import any BigCommerce data, please check your access token permissions")
	}

	return nil
}

// fetchBigCommerceResourceV3 fetches a V3 API resource
func (s *DataSourceService) fetchBigCommerceResourceV3(client *http.Client, db *sql.DB, baseURL, endpoint, jsonKey, tableName, accessToken string) error {
	allData := []map[string]interface{}{}
	nextURL := baseURL + endpoint

	// Paginate through all results
	for nextURL != "" {
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("X-Auth-Token", accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to fetch data: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %v", err)
		}
		resp.Body.Close()

		// Extract data from response
		if data, ok := result[jsonKey].([]interface{}); ok {
			for _, item := range data {
				if itemMap, ok := item.(map[string]interface{}); ok {
					allData = append(allData, itemMap)
				}
			}
		}

		// Check for pagination
		nextURL = ""
		if meta, ok := result["meta"].(map[string]interface{}); ok {
			if pagination, ok := meta["pagination"].(map[string]interface{}); ok {
				if links, ok := pagination["links"].(map[string]interface{}); ok {
					if next, ok := links["next"].(string); ok && next != "" {
						nextURL = next
					}
				}
			}
		}
	}

	if len(allData) == 0 {
		s.log(fmt.Sprintf("No data found for %s", tableName))
		return nil
	}

	return s.createTableFromJSON(db, tableName, allData)
}

// fetchBigCommerceResourceV2 fetches a V2 API resource (for orders)
func (s *DataSourceService) fetchBigCommerceResourceV2(client *http.Client, db *sql.DB, baseURL, endpoint, tableName, accessToken string) error {
	allData := []map[string]interface{}{}
	page := 1

	// Paginate through all results
	for {
		url := fmt.Sprintf("%s%s&page=%d", baseURL, endpoint, page)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("X-Auth-Token", accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to fetch data: %v", err)
		}

		// V2 API returns 204 when no more data
		if resp.StatusCode == http.StatusNoContent {
			resp.Body.Close()
			break
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %v", err)
		}
		resp.Body.Close()

		if len(result) == 0 {
			break
		}

		allData = append(allData, result...)
		page++
	}

	if len(allData) == 0 {
		s.log(fmt.Sprintf("No data found for %s", tableName))
		return nil
	}

	return s.createTableFromJSON(db, tableName, allData)
}


// ImportEbay imports data from eBay APIs (Fulfillment, Finances, Analytics)
func (s *DataSourceService) ImportEbay(name string, config DataSourceConfig) (*DataSource, error) {
	s.log(fmt.Sprintf("ImportEbay: %s", name))

	// Validate configuration
	if config.EbayAccessToken == "" {
		return nil, fmt.Errorf("ebay access token is required")
	}

	// Set default environment
	environment := config.EbayEnvironment
	if environment == "" {
		environment = "production"
	}

	// Create data source ID
	id := uuid.New().String()

	// Create local storage directory
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create SQLite database
	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to create local database: %v", err)
	}
	defer db.Close()

	// Fetch and import eBay data
	if err := s.fetchEbayData(db, config.EbayAccessToken, environment, config); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to fetch eBay data: %v", err)
	}

	// Create data source object
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "ebay",
		CreatedAt: time.Now().UnixMilli(),
		Config: DataSourceConfig{
			DBPath:             relDBPath,
			EbayAccessToken:    config.EbayAccessToken,
			EbayEnvironment:    environment,
			EbayApiFulfillment: config.EbayApiFulfillment,
			EbayApiFinances:    config.EbayApiFinances,
			EbayApiAnalytics:   config.EbayApiAnalytics,
		},
	}

	// Save to registry
	if err := s.AddDataSource(ds); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, err
	}

	s.log(fmt.Sprintf("eBay data source imported successfully: %s", name))
	return &ds, nil
}

// fetchEbayData fetches data from eBay APIs and stores it in SQLite
func (s *DataSourceService) fetchEbayData(db *sql.DB, accessToken, environment string, config DataSourceConfig) error {
	// Determine base URL based on environment
	var baseURL string
	if environment == "sandbox" {
		baseURL = "https://api.sandbox.ebay.com"
	} else {
		baseURL = "https://api.ebay.com"
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	importedCount := 0

	// Fulfillment API - Orders
	if config.EbayApiFulfillment {
		s.log("Fetching eBay Fulfillment API data (orders)...")
		if err := s.fetchEbayFulfillmentData(client, db, baseURL, accessToken); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to fetch Fulfillment data: %v", err))
		} else {
			importedCount++
		}
	}

	// Finances API - Transactions, Payouts
	if config.EbayApiFinances {
		s.log("Fetching eBay Finances API data...")
		if err := s.fetchEbayFinancesData(client, db, baseURL, accessToken); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to fetch Finances data: %v", err))
		} else {
			importedCount++
		}
	}

	// Analytics API - Traffic, Seller Standards
	if config.EbayApiAnalytics {
		s.log("Fetching eBay Analytics API data...")
		if err := s.fetchEbayAnalyticsData(client, db, baseURL, accessToken); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to fetch Analytics data: %v", err))
		} else {
			importedCount++
		}
	}

	if importedCount == 0 {
		return fmt.Errorf("failed to import any eBay data, please check your access token permissions")
	}

	return nil
}

// fetchEbayFulfillmentData fetches order data from Fulfillment API
func (s *DataSourceService) fetchEbayFulfillmentData(client *http.Client, db *sql.DB, baseURL, accessToken string) error {
	allOrders := []map[string]interface{}{}
	offset := 0
	limit := 200

	for {
		url := fmt.Sprintf("%s/sell/fulfillment/v1/order?limit=%d&offset=%d", baseURL, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to fetch data: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %v", err)
		}
		resp.Body.Close()

		// Extract orders
		if orders, ok := result["orders"].([]interface{}); ok {
			for _, order := range orders {
				if orderMap, ok := order.(map[string]interface{}); ok {
					allOrders = append(allOrders, orderMap)
				}
			}

			// Check if there are more pages
			if len(orders) < limit {
				break
			}
			offset += limit
		} else {
			break
		}
	}

	if len(allOrders) == 0 {
		s.log("No orders found in Fulfillment API")
		return nil
	}

	// Create orders table
	if err := s.createTableFromJSON(db, "orders", allOrders); err != nil {
		return fmt.Errorf("failed to create orders table: %v", err)
	}

	// Extract line items into separate table for easier analysis
	allLineItems := []map[string]interface{}{}
	for _, order := range allOrders {
		orderID, _ := order["orderId"].(string)
		if lineItems, ok := order["lineItems"].([]interface{}); ok {
			for _, item := range lineItems {
				if itemMap, ok := item.(map[string]interface{}); ok {
					itemMap["orderId"] = orderID
					allLineItems = append(allLineItems, itemMap)
				}
			}
		}
	}

	if len(allLineItems) > 0 {
		if err := s.createTableFromJSON(db, "order_line_items", allLineItems); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to create line items table: %v", err))
		}
	}

	s.log(fmt.Sprintf("Imported %d orders from Fulfillment API", len(allOrders)))
	return nil
}

// fetchEbayFinancesData fetches financial data from Finances API
func (s *DataSourceService) fetchEbayFinancesData(client *http.Client, db *sql.DB, baseURL, accessToken string) error {
	// Fetch transactions
	allTransactions := []map[string]interface{}{}
	offset := 0
	limit := 200

	for {
		url := fmt.Sprintf("%s/sell/finances/v1/transaction?limit=%d&offset=%d", baseURL, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to fetch data: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			// Some accounts may not have access to Finances API
			if resp.StatusCode == http.StatusForbidden {
				s.log("Finances API access denied - skipping transactions")
				break
			}
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %v", err)
		}
		resp.Body.Close()

		if transactions, ok := result["transactions"].([]interface{}); ok {
			for _, txn := range transactions {
				if txnMap, ok := txn.(map[string]interface{}); ok {
					allTransactions = append(allTransactions, txnMap)
				}
			}

			if len(transactions) < limit {
				break
			}
			offset += limit
		} else {
			break
		}
	}

	if len(allTransactions) > 0 {
		if err := s.createTableFromJSON(db, "transactions", allTransactions); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to create transactions table: %v", err))
		} else {
			s.log(fmt.Sprintf("Imported %d transactions from Finances API", len(allTransactions)))
		}
	}

	// Fetch payouts
	allPayouts := []map[string]interface{}{}
	offset = 0

	for {
		url := fmt.Sprintf("%s/sell/finances/v1/payout?limit=%d&offset=%d", baseURL, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			break
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			break
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			break
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			break
		}
		resp.Body.Close()

		if payouts, ok := result["payouts"].([]interface{}); ok {
			for _, payout := range payouts {
				if payoutMap, ok := payout.(map[string]interface{}); ok {
					allPayouts = append(allPayouts, payoutMap)
				}
			}

			if len(payouts) < limit {
				break
			}
			offset += limit
		} else {
			break
		}
	}

	if len(allPayouts) > 0 {
		if err := s.createTableFromJSON(db, "payouts", allPayouts); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to create payouts table: %v", err))
		} else {
			s.log(fmt.Sprintf("Imported %d payouts from Finances API", len(allPayouts)))
		}
	}

	return nil
}

// fetchEbayAnalyticsData fetches analytics data from Analytics API
func (s *DataSourceService) fetchEbayAnalyticsData(client *http.Client, db *sql.DB, baseURL, accessToken string) error {
	// Fetch traffic report
	trafficURL := fmt.Sprintf("%s/sell/analytics/v1/traffic_report?dimension=DAY&metric=CLICK_THROUGH_RATE&metric=LISTING_IMPRESSION_TOTAL&metric=LISTING_VIEWS_TOTAL", baseURL)

	req, err := http.NewRequest("GET", trafficURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			// Extract dimension metrics
			if dimensionMetrics, ok := result["dimensionMetrics"].([]interface{}); ok {
				trafficData := []map[string]interface{}{}
				for _, dm := range dimensionMetrics {
					if dmMap, ok := dm.(map[string]interface{}); ok {
						trafficData = append(trafficData, dmMap)
					}
				}
				if len(trafficData) > 0 {
					if err := s.createTableFromJSON(db, "traffic_report", trafficData); err != nil {
						s.log(fmt.Sprintf("Warning: Failed to create traffic_report table: %v", err))
					} else {
						s.log(fmt.Sprintf("Imported %d traffic report records", len(trafficData)))
					}
				}
			}
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		s.log(fmt.Sprintf("Traffic report API returned status %d: %s", resp.StatusCode, string(body)))
	}

	// Fetch seller standards profile
	standardsURL := fmt.Sprintf("%s/sell/analytics/v1/seller_standards_profile?program=GLOBAL", baseURL)

	req2, err := http.NewRequest("GET", standardsURL, nil)
	if err != nil {
		return nil // Non-fatal, continue
	}

	req2.Header.Set("Authorization", "Bearer "+accessToken)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept", "application/json")

	resp2, err := client.Do(req2)
	if err != nil {
		return nil
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp2.Body).Decode(&result); err == nil {
			// Store as single-row table
			standardsData := []map[string]interface{}{result}
			if err := s.createTableFromJSON(db, "seller_standards", standardsData); err != nil {
				s.log(fmt.Sprintf("Warning: Failed to create seller_standards table: %v", err))
			} else {
				s.log("Imported seller standards profile")
			}
		}
	}

	return nil
}


// ImportEtsy imports data from Etsy API
func (s *DataSourceService) ImportEtsy(name string, config DataSourceConfig) (*DataSource, error) {
	s.log(fmt.Sprintf("ImportEtsy: %s (Shop: %s)", name, config.EtsyShopId))

	// Validate configuration
	if config.EtsyAccessToken == "" {
		return nil, fmt.Errorf("etsy access token is required")
	}

	// Create data source ID
	id := uuid.New().String()

	// Create local storage directory
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create SQLite database
	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to create local database: %v", err)
	}
	defer db.Close()

	// Auto-detect shop ID if not provided
	shopId := config.EtsyShopId
	if shopId == "" {
		s.log("Auto-detecting Etsy shop ID...")
		detectedShopId, err := s.getEtsyShopId(config.EtsyAccessToken)
		if err != nil {
			_ = os.RemoveAll(absDBDir)
			return nil, fmt.Errorf("failed to detect shop ID: %v", err)
		}
		shopId = detectedShopId
		s.log(fmt.Sprintf("Detected shop ID: %s", shopId))
	}

	// Fetch and import Etsy data
	if err := s.fetchEtsyData(db, shopId, config.EtsyAccessToken); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to fetch Etsy data: %v", err)
	}

	// Create data source object
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "etsy",
		CreatedAt: time.Now().UnixMilli(),
		Config: DataSourceConfig{
			DBPath:          relDBPath,
			EtsyShopId:      shopId,
			EtsyAccessToken: config.EtsyAccessToken,
		},
	}

	// Save to registry
	if err := s.AddDataSource(ds); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, err
	}

	s.log(fmt.Sprintf("Etsy data source imported successfully: %s", name))
	return &ds, nil
}

// fetchEtsyData fetches data from Etsy API and stores it in SQLite
func (s *DataSourceService) fetchEtsyData(db *sql.DB, shopId, accessToken string) error {
	baseURL := "https://openapi.etsy.com/v3"

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	importedCount := 0

	// Fetch shop info
	s.log("Fetching Etsy shop info...")
	if err := s.fetchEtsyShopInfo(client, db, baseURL, shopId, accessToken); err != nil {
		s.log(fmt.Sprintf("Warning: Failed to fetch shop info: %v", err))
	} else {
		importedCount++
	}

	// Fetch listings (products)
	s.log("Fetching Etsy listings...")
	if err := s.fetchEtsyListings(client, db, baseURL, shopId, accessToken); err != nil {
		s.log(fmt.Sprintf("Warning: Failed to fetch listings: %v", err))
	} else {
		importedCount++
	}

	// Fetch receipts (orders)
	s.log("Fetching Etsy receipts (orders)...")
	if err := s.fetchEtsyReceipts(client, db, baseURL, shopId, accessToken); err != nil {
		s.log(fmt.Sprintf("Warning: Failed to fetch receipts: %v", err))
	} else {
		importedCount++
	}

	// Fetch transactions
	s.log("Fetching Etsy transactions...")
	if err := s.fetchEtsyTransactions(client, db, baseURL, shopId, accessToken); err != nil {
		s.log(fmt.Sprintf("Warning: Failed to fetch transactions: %v", err))
	} else {
		importedCount++
	}

	// Fetch reviews
	s.log("Fetching Etsy reviews...")
	if err := s.fetchEtsyReviews(client, db, baseURL, shopId, accessToken); err != nil {
		s.log(fmt.Sprintf("Warning: Failed to fetch reviews: %v", err))
	} else {
		importedCount++
	}

	if importedCount == 0 {
		return fmt.Errorf("failed to import any Etsy data, please check your credentials")
	}

	return nil
}

// getEtsyShopId fetches the shop ID for the authenticated user
func (s *DataSourceService) getEtsyShopId(accessToken string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Get user info first
	req, err := http.NewRequest("GET", "https://openapi.etsy.com/v3/application/users/me", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Extract shop_id from user info
	if shopId, ok := result["shop_id"]; ok {
		switch v := shopId.(type) {
		case float64:
			return fmt.Sprintf("%.0f", v), nil
		case string:
			return v, nil
		}
	}

	return "", fmt.Errorf("shop_id not found in user info, user may not have a shop")
}

// fetchEtsyShopInfo fetches shop information
func (s *DataSourceService) fetchEtsyShopInfo(client *http.Client, db *sql.DB, baseURL, shopId, accessToken string) error {
	url := fmt.Sprintf("%s/application/shops/%s", baseURL, shopId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Store as single-row table
	shopData := []map[string]interface{}{result}
	return s.createTableFromJSON(db, "shop", shopData)
}

// fetchEtsyListings fetches all active listings
func (s *DataSourceService) fetchEtsyListings(client *http.Client, db *sql.DB, baseURL, shopId, accessToken string) error {
	allListings := []map[string]interface{}{}
	offset := 0
	limit := 100

	for {
		url := fmt.Sprintf("%s/application/shops/%s/listings?limit=%d&offset=%d&state=active", baseURL, shopId, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		if results, ok := result["results"].([]interface{}); ok {
			for _, item := range results {
				if itemMap, ok := item.(map[string]interface{}); ok {
					allListings = append(allListings, itemMap)
				}
			}

			// Check pagination
			count, _ := result["count"].(float64)
			if offset+limit >= int(count) {
				break
			}
			offset += limit
		} else {
			break
		}
	}

	if len(allListings) == 0 {
		s.log("No listings found")
		return nil
	}

	s.log(fmt.Sprintf("Imported %d listings", len(allListings)))
	return s.createTableFromJSON(db, "listings", allListings)
}

// fetchEtsyReceipts fetches shop receipts (orders)
func (s *DataSourceService) fetchEtsyReceipts(client *http.Client, db *sql.DB, baseURL, shopId, accessToken string) error {
	allReceipts := []map[string]interface{}{}
	offset := 0
	limit := 100

	for {
		url := fmt.Sprintf("%s/application/shops/%s/receipts?limit=%d&offset=%d", baseURL, shopId, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		if results, ok := result["results"].([]interface{}); ok {
			for _, item := range results {
				if itemMap, ok := item.(map[string]interface{}); ok {
					allReceipts = append(allReceipts, itemMap)
				}
			}

			count, _ := result["count"].(float64)
			if offset+limit >= int(count) {
				break
			}
			offset += limit
		} else {
			break
		}
	}

	if len(allReceipts) == 0 {
		s.log("No receipts found")
		return nil
	}

	s.log(fmt.Sprintf("Imported %d receipts", len(allReceipts)))
	return s.createTableFromJSON(db, "receipts", allReceipts)
}

// fetchEtsyTransactions fetches shop transactions
func (s *DataSourceService) fetchEtsyTransactions(client *http.Client, db *sql.DB, baseURL, shopId, accessToken string) error {
	allTransactions := []map[string]interface{}{}
	offset := 0
	limit := 100

	for {
		url := fmt.Sprintf("%s/application/shops/%s/transactions?limit=%d&offset=%d", baseURL, shopId, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		if results, ok := result["results"].([]interface{}); ok {
			for _, item := range results {
				if itemMap, ok := item.(map[string]interface{}); ok {
					allTransactions = append(allTransactions, itemMap)
				}
			}

			count, _ := result["count"].(float64)
			if offset+limit >= int(count) {
				break
			}
			offset += limit
		} else {
			break
		}
	}

	if len(allTransactions) == 0 {
		s.log("No transactions found")
		return nil
	}

	s.log(fmt.Sprintf("Imported %d transactions", len(allTransactions)))
	return s.createTableFromJSON(db, "transactions", allTransactions)
}

// fetchEtsyReviews fetches shop reviews
func (s *DataSourceService) fetchEtsyReviews(client *http.Client, db *sql.DB, baseURL, shopId, accessToken string) error {
	allReviews := []map[string]interface{}{}
	offset := 0
	limit := 100

	for {
		url := fmt.Sprintf("%s/application/shops/%s/reviews?limit=%d&offset=%d", baseURL, shopId, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			// Reviews might not be accessible, skip silently
			resp.Body.Close()
			break
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		if results, ok := result["results"].([]interface{}); ok {
			for _, item := range results {
				if itemMap, ok := item.(map[string]interface{}); ok {
					allReviews = append(allReviews, itemMap)
				}
			}

			count, _ := result["count"].(float64)
			if offset+limit >= int(count) {
				break
			}
			offset += limit
		} else {
			break
		}
	}

	if len(allReviews) == 0 {
		s.log("No reviews found")
		return nil
	}

	s.log(fmt.Sprintf("Imported %d reviews", len(allReviews)))
	return s.createTableFromJSON(db, "reviews", allReviews)
}


// RefreshEcommerceDataSource performs incremental update for e-commerce data sources
// It fetches new data from the API and merges it with existing data
func (s *DataSourceService) RefreshEcommerceDataSource(id string) (*RefreshResult, error) {
	s.log(fmt.Sprintf("RefreshEcommerceDataSource: %s", id))

	// Load data source
	sources, err := s.LoadDataSources()
	if err != nil {
		return nil, fmt.Errorf("failed to load data sources: %v", err)
	}

	var ds *DataSource
	for i := range sources {
		if sources[i].ID == id {
			ds = &sources[i]
			break
		}
	}

	if ds == nil {
		return nil, fmt.Errorf("data source not found: %s", id)
	}

	// Check if it's an e-commerce data source
	switch ds.Type {
	case "shopify":
		return s.refreshShopifyData(ds)
	case "bigcommerce":
		return s.refreshBigCommerceData(ds)
	case "ebay":
		return s.refreshEbayData(ds)
	case "etsy":
		return s.refreshEtsyData(ds)
	default:
		return nil, fmt.Errorf("data source type '%s' does not support incremental refresh", ds.Type)
	}
}

// RefreshResult holds the result of a data refresh operation
type RefreshResult struct {
	DataSourceID   string            `json:"data_source_id"`
	DataSourceName string            `json:"data_source_name"`
	TablesUpdated  map[string]int    `json:"tables_updated"` // table name -> new rows count
	TotalNewRows   int               `json:"total_new_rows"`
	Error          string            `json:"error,omitempty"`
}

// refreshShopifyData performs incremental update for Shopify data source
func (s *DataSourceService) refreshShopifyData(ds *DataSource) (*RefreshResult, error) {
	s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Starting refresh for %s", ds.Name))

	result := &RefreshResult{
		DataSourceID:   ds.ID,
		DataSourceName: ds.Name,
		TablesUpdated:  make(map[string]int),
	}

	// Validate configuration
	if ds.Config.ShopifyStore == "" || ds.Config.ShopifyAccessToken == "" {
		return nil, fmt.Errorf("shopify store URL and access token are required")
	}

	// Normalize store URL
	store := ds.Config.ShopifyStore
	store = strings.TrimPrefix(store, "https://")
	store = strings.TrimPrefix(store, "http://")
	store = strings.TrimSuffix(store, "/")
	if !strings.Contains(store, ".myshopify.com") {
		if !strings.Contains(store, ".") {
			store = store + ".myshopify.com"
		}
	}

	apiVersion := ds.Config.ShopifyAPIVersion
	if apiVersion == "" {
		apiVersion = "2024-01"
	}

	// Open existing database
	absDBPath := filepath.Join(s.dataCacheDir, ds.Config.DBPath)
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	baseURL := fmt.Sprintf("https://%s/admin/api/%s", store, apiVersion)
	client := &http.Client{Timeout: 30 * time.Second}

	// Resources to refresh - focus on frequently updated ones
	resources := []struct {
		name     string
		endpoint string
		key      string
		idField  string
	}{
		{"orders", "/orders.json?status=any&limit=250&order=created_at desc", "orders", "id"},
		{"customers", "/customers.json?limit=250&order=created_at desc", "customers", "id"},
		{"products", "/products.json?limit=250", "products", "id"},
	}

	for _, resource := range resources {
		s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Refreshing %s...", resource.name))
		
		newCount, err := s.refreshShopifyResource(client, db, baseURL, resource.endpoint, resource.key, resource.name, resource.idField, ds.Config.ShopifyAccessToken)
		if err != nil {
			s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Warning: Failed to refresh %s: %v", resource.name, err))
			continue
		}
		
		if newCount > 0 {
			result.TablesUpdated[resource.name] = newCount
			result.TotalNewRows += newCount
			s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Added %d new rows to %s", newCount, resource.name))
		}
	}

	// Invalidate cache after refresh
	s.InvalidateCache(ds.ID)

	s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Completed. Total new rows: %d", result.TotalNewRows))
	return result, nil
}

// refreshShopifyResource fetches new data for a specific resource and inserts only new records
func (s *DataSourceService) refreshShopifyResource(client *http.Client, db *sql.DB, baseURL, endpoint, jsonKey, tableName, idField, accessToken string) (int, error) {
	// Get the maximum ID from existing records to use as a baseline
	// This is more efficient than loading all IDs into memory
	var maxID int64 = 0
	row := db.QueryRow(fmt.Sprintf("SELECT MAX(CAST(`%s` AS INTEGER)) FROM `%s`", idField, tableName))
	row.Scan(&maxID) // Ignore error - table might be empty or have non-numeric IDs
	
	s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Max existing ID in %s: %d", tableName, maxID))

	// Also build a set of existing IDs for accurate duplicate detection
	// (in case IDs are not strictly sequential)
	existingIDs := make(map[int64]bool)
	rows, err := db.Query(fmt.Sprintf("SELECT CAST(`%s` AS INTEGER) FROM `%s`", idField, tableName))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil {
				existingIDs[id] = true
			}
		}
	}
	s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Found %d existing records in %s", len(existingIDs), tableName))

	// Fetch new data
	newData := []map[string]interface{}{}
	nextURL := baseURL + endpoint

	for nextURL != "" {
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("X-Shopify-Access-Token", accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch data: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return 0, fmt.Errorf("failed to decode response: %v", err)
		}

		linkHeader := resp.Header.Get("Link")
		resp.Body.Close()

		foundExisting := false
		if data, ok := result[jsonKey].([]interface{}); ok {
			for _, item := range data {
				if itemMap, ok := item.(map[string]interface{}); ok {
					// Check if this record already exists
					if idVal, ok := itemMap[idField]; ok {
						// Convert to int64 for comparison
						var idInt int64
						switch v := idVal.(type) {
						case float64:
							idInt = int64(v)
						case int64:
							idInt = v
						case int:
							idInt = int64(v)
						default:
							continue // Skip non-numeric IDs
						}
						
						if !existingIDs[idInt] {
							newData = append(newData, itemMap)
						} else {
							foundExisting = true
						}
					}
				}
			}
		}

		nextURL = s.extractNextLink(linkHeader)
		
		// Stop pagination if we've hit existing records (data is sorted by created_at desc)
		// or if we've found enough new records
		if foundExisting || len(newData) >= 500 {
			break
		}
	}

	if len(newData) == 0 {
		return 0, nil
	}

	// Insert new data
	s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Inserting %d new records into %s", len(newData), tableName))
	
	// Get existing columns
	colRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", tableName))
	if err != nil {
		return 0, fmt.Errorf("failed to get table info: %v", err)
	}
	defer colRows.Close()

	var colNames []string
	for colRows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := colRows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
			colNames = append(colNames, name)
		}
	}

	// Prepare insert statement
	quotedColNames := make([]string, len(colNames))
	for i, name := range colNames {
		quotedColNames[i] = fmt.Sprintf("`%s`", name)
	}
	placeholders := make([]string, len(colNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(quotedColNames, ", "),
		strings.Join(placeholders, ", "))

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert: %v", err)
	}
	defer stmt.Close()

	insertedCount := 0
	for _, row := range newData {
		flatRow := make(map[string]interface{})
		s.flattenJSON("", row, flatRow)

		values := make([]interface{}, len(colNames))
		for i, colName := range colNames {
			if val, ok := flatRow[colName]; ok {
				values[i] = s.formatValue(val)
			} else {
				values[i] = nil
			}
		}

		if _, err := stmt.Exec(values...); err != nil {
			s.log(fmt.Sprintf("[SHOPIFY-REFRESH] Warning: Failed to insert row: %v", err))
		} else {
			insertedCount++
		}
	}

	return insertedCount, nil
}

// refreshBigCommerceData performs incremental update for BigCommerce data source
func (s *DataSourceService) refreshBigCommerceData(ds *DataSource) (*RefreshResult, error) {
	s.log(fmt.Sprintf("[BIGCOMMERCE-REFRESH] Starting refresh for %s", ds.Name))

	result := &RefreshResult{
		DataSourceID:   ds.ID,
		DataSourceName: ds.Name,
		TablesUpdated:  make(map[string]int),
	}

	// Validate configuration
	if ds.Config.BigCommerceStoreHash == "" || ds.Config.BigCommerceAccessToken == "" {
		return nil, fmt.Errorf("bigcommerce store hash and access token are required")
	}

	// Open existing database
	absDBPath := filepath.Join(s.dataCacheDir, ds.Config.DBPath)
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	baseURL := fmt.Sprintf("https://api.bigcommerce.com/stores/%s/v3", ds.Config.BigCommerceStoreHash)
	client := &http.Client{Timeout: 30 * time.Second}

	// Resources to refresh
	resources := []struct {
		name     string
		endpoint string
		key      string
		idField  string
	}{
		{"orders", "/orders?sort=date_created:desc&limit=250", "data", "id"},
		{"customers", "/customers?sort=date_created:desc&limit=250", "data", "id"},
		{"products", "/catalog/products?limit=250", "data", "id"},
	}

	for _, resource := range resources {
		s.log(fmt.Sprintf("[BIGCOMMERCE-REFRESH] Refreshing %s...", resource.name))
		
		newCount, err := s.refreshBigCommerceResource(client, db, baseURL, resource.endpoint, resource.key, resource.name, resource.idField, ds.Config.BigCommerceAccessToken)
		if err != nil {
			s.log(fmt.Sprintf("[BIGCOMMERCE-REFRESH] Warning: Failed to refresh %s: %v", resource.name, err))
			continue
		}
		
		if newCount > 0 {
			result.TablesUpdated[resource.name] = newCount
			result.TotalNewRows += newCount
		}
	}

	s.InvalidateCache(ds.ID)
	s.log(fmt.Sprintf("[BIGCOMMERCE-REFRESH] Completed. Total new rows: %d", result.TotalNewRows))
	return result, nil
}

// refreshBigCommerceResource fetches new data for a specific BigCommerce resource
func (s *DataSourceService) refreshBigCommerceResource(client *http.Client, db *sql.DB, baseURL, endpoint, jsonKey, tableName, idField, accessToken string) (int, error) {
	// Get existing IDs using integer comparison
	existingIDs := make(map[int64]bool)
	rows, err := db.Query(fmt.Sprintf("SELECT CAST(`%s` AS INTEGER) FROM `%s`", idField, tableName))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil {
				existingIDs[id] = true
			}
		}
	}

	// Fetch new data
	newData := []map[string]interface{}{}
	url := baseURL + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("X-Auth-Token", accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	if data, ok := result[jsonKey].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if idVal, ok := itemMap[idField]; ok {
					var idInt int64
					switch v := idVal.(type) {
					case float64:
						idInt = int64(v)
					case int64:
						idInt = v
					case int:
						idInt = int64(v)
					default:
						continue
					}
					if !existingIDs[idInt] {
						newData = append(newData, itemMap)
					}
				}
			}
		}
	}

	if len(newData) == 0 {
		return 0, nil
	}

	// Insert new data using existing table schema
	return s.insertNewRecords(db, tableName, newData)
}

// refreshEbayData performs incremental update for eBay data source
func (s *DataSourceService) refreshEbayData(ds *DataSource) (*RefreshResult, error) {
	s.log(fmt.Sprintf("[EBAY-REFRESH] Starting refresh for %s", ds.Name))

	result := &RefreshResult{
		DataSourceID:   ds.ID,
		DataSourceName: ds.Name,
		TablesUpdated:  make(map[string]int),
	}

	if ds.Config.EbayAccessToken == "" {
		return nil, fmt.Errorf("ebay access token is required")
	}

	// Open existing database
	absDBPath := filepath.Join(s.dataCacheDir, ds.Config.DBPath)
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Determine environment
	env := ds.Config.EbayEnvironment
	if env == "" {
		env = "production"
	}

	var baseURL string
	if env == "sandbox" {
		baseURL = "https://api.sandbox.ebay.com"
	} else {
		baseURL = "https://api.ebay.com"
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// Refresh orders (most commonly updated)
	s.log("[EBAY-REFRESH] Refreshing orders...")
	newCount, err := s.refreshEbayOrders(client, db, baseURL, ds.Config.EbayAccessToken)
	if err != nil {
		s.log(fmt.Sprintf("[EBAY-REFRESH] Warning: Failed to refresh orders: %v", err))
	} else if newCount > 0 {
		result.TablesUpdated["orders"] = newCount
		result.TotalNewRows += newCount
	}

	s.InvalidateCache(ds.ID)
	s.log(fmt.Sprintf("[EBAY-REFRESH] Completed. Total new rows: %d", result.TotalNewRows))
	return result, nil
}

// refreshEbayOrders fetches new eBay orders
func (s *DataSourceService) refreshEbayOrders(client *http.Client, db *sql.DB, baseURL, accessToken string) (int, error) {
	// Get existing order IDs (eBay order IDs are strings like "12-34567-89012")
	existingIDs := make(map[string]bool)
	rows, err := db.Query("SELECT `orderId` FROM `orders`")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil {
				existingIDs[strings.TrimSpace(id)] = true
			}
		}
	}

	// Fetch recent orders
	url := baseURL + "/sell/fulfillment/v1/order?limit=50"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	newData := []map[string]interface{}{}
	if orders, ok := result["orders"].([]interface{}); ok {
		for _, item := range orders {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if idVal, ok := itemMap["orderId"]; ok {
					idStr := strings.TrimSpace(fmt.Sprintf("%v", idVal))
					if !existingIDs[idStr] {
						newData = append(newData, itemMap)
					}
				}
			}
		}
	}

	if len(newData) == 0 {
		return 0, nil
	}

	return s.insertNewRecords(db, "orders", newData)
}

// refreshEtsyData performs incremental update for Etsy data source
func (s *DataSourceService) refreshEtsyData(ds *DataSource) (*RefreshResult, error) {
	s.log(fmt.Sprintf("[ETSY-REFRESH] Starting refresh for %s", ds.Name))

	result := &RefreshResult{
		DataSourceID:   ds.ID,
		DataSourceName: ds.Name,
		TablesUpdated:  make(map[string]int),
	}

	if ds.Config.EtsyAccessToken == "" || ds.Config.EtsyShopId == "" {
		return nil, fmt.Errorf("etsy access token and shop ID are required")
	}

	// Open existing database
	absDBPath := filepath.Join(s.dataCacheDir, ds.Config.DBPath)
	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	baseURL := "https://openapi.etsy.com/v3"
	client := &http.Client{Timeout: 30 * time.Second}

	// Refresh receipts (orders)
	s.log("[ETSY-REFRESH] Refreshing receipts...")
	newCount, err := s.refreshEtsyReceipts(client, db, baseURL, ds.Config.EtsyShopId, ds.Config.EtsyAccessToken)
	if err != nil {
		s.log(fmt.Sprintf("[ETSY-REFRESH] Warning: Failed to refresh receipts: %v", err))
	} else if newCount > 0 {
		result.TablesUpdated["receipts"] = newCount
		result.TotalNewRows += newCount
	}

	s.InvalidateCache(ds.ID)
	s.log(fmt.Sprintf("[ETSY-REFRESH] Completed. Total new rows: %d", result.TotalNewRows))
	return result, nil
}

// refreshEtsyReceipts fetches new Etsy receipts
func (s *DataSourceService) refreshEtsyReceipts(client *http.Client, db *sql.DB, baseURL, shopId, accessToken string) (int, error) {
	// Get existing receipt IDs using integer comparison
	existingIDs := make(map[int64]bool)
	rows, err := db.Query("SELECT CAST(`receipt_id` AS INTEGER) FROM `receipts`")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil {
				existingIDs[id] = true
			}
		}
	}

	// Fetch recent receipts
	url := fmt.Sprintf("%s/application/shops/%s/receipts?limit=25", baseURL, shopId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", accessToken) // Etsy uses API key in header
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	newData := []map[string]interface{}{}
	if results, ok := result["results"].([]interface{}); ok {
		for _, item := range results {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if idVal, ok := itemMap["receipt_id"]; ok {
					var idInt int64
					switch v := idVal.(type) {
					case float64:
						idInt = int64(v)
					case int64:
						idInt = v
					case int:
						idInt = int64(v)
					default:
						continue
					}
					if !existingIDs[idInt] {
						newData = append(newData, itemMap)
					}
				}
			}
		}
	}

	if len(newData) == 0 {
		return 0, nil
	}

	return s.insertNewRecords(db, "receipts", newData)
}

// insertNewRecords inserts new records into an existing table
func (s *DataSourceService) insertNewRecords(db *sql.DB, tableName string, newData []map[string]interface{}) (int, error) {
	if len(newData) == 0 {
		return 0, nil
	}

	// Get existing columns
	colRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", tableName))
	if err != nil {
		return 0, fmt.Errorf("failed to get table info: %v", err)
	}
	defer colRows.Close()

	var colNames []string
	for colRows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := colRows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
			colNames = append(colNames, name)
		}
	}

	if len(colNames) == 0 {
		return 0, fmt.Errorf("table %s has no columns", tableName)
	}

	// Prepare insert statement
	quotedColNames := make([]string, len(colNames))
	for i, name := range colNames {
		quotedColNames[i] = fmt.Sprintf("`%s`", name)
	}
	placeholders := make([]string, len(colNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(quotedColNames, ", "),
		strings.Join(placeholders, ", "))

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert: %v", err)
	}
	defer stmt.Close()

	insertedCount := 0
	for _, row := range newData {
		flatRow := make(map[string]interface{})
		s.flattenJSON("", row, flatRow)

		values := make([]interface{}, len(colNames))
		for i, colName := range colNames {
			if val, ok := flatRow[colName]; ok {
				values[i] = s.formatValue(val)
			} else {
				values[i] = nil
			}
		}

		if _, err := stmt.Exec(values...); err != nil {
			s.log(fmt.Sprintf("Warning: Failed to insert row: %v", err))
		} else {
			insertedCount++
		}
	}

	return insertedCount, nil
}

// IsEcommerceDataSource checks if a data source is an e-commerce type
func (s *DataSourceService) IsEcommerceDataSource(dsType string) bool {
	switch strings.ToLower(dsType) {
	case "shopify", "bigcommerce", "ebay", "etsy":
		return true
	default:
		return false
	}
}

// IsRefreshableDataSource checks if a data source type supports incremental refresh
func (s *DataSourceService) IsRefreshableDataSource(dsType string) bool {
	switch strings.ToLower(dsType) {
	case "shopify", "bigcommerce", "ebay", "etsy", "jira":
		return true
	default:
		return false
	}
}

// JiraProject represents a Jira project
type JiraProject struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	ID   string `json:"id"`
}

// GetJiraProjects fetches available projects from Jira
func (s *DataSourceService) GetJiraProjects(instanceType, baseUrl, username, apiToken string) ([]JiraProject, error) {
	// Normalize base URL
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	if !strings.HasPrefix(baseUrl, "http://") && !strings.HasPrefix(baseUrl, "https://") {
		baseUrl = "https://" + baseUrl
	}

	if instanceType == "" {
		instanceType = "cloud"
	}

	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("%s/rest/api/2/project", baseUrl)

	resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Jira: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		if instanceType == "cloud" {
			return nil, fmt.Errorf("authentication failed: Please verify your email and API token")
		}
		return nil, fmt.Errorf("authentication failed: Please verify your username and password")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jira API error (%d): %s", resp.StatusCode, string(body))
	}

	var projects []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	result := make([]JiraProject, 0, len(projects))
	for _, p := range projects {
		if proj, ok := p.(map[string]interface{}); ok {
			result = append(result, JiraProject{
				Key:  fmt.Sprintf("%v", proj["key"]),
				Name: fmt.Sprintf("%v", proj["name"]),
				ID:   fmt.Sprintf("%v", proj["id"]),
			})
		}
	}

	return result, nil
}

// RefreshDataSource performs incremental update for supported data sources
func (s *DataSourceService) RefreshDataSource(id string) (*RefreshResult, error) {
	sources, err := s.LoadDataSources()
	if err != nil {
		return nil, err
	}

	var ds *DataSource
	for i := range sources {
		if sources[i].ID == id {
			ds = &sources[i]
			break
		}
	}

	if ds == nil {
		return nil, fmt.Errorf("data source not found")
	}

	switch strings.ToLower(ds.Type) {
	case "shopify":
		return s.refreshShopifyData(ds)
	case "bigcommerce":
		return s.refreshBigCommerceData(ds)
	case "ebay":
		return s.refreshEbayData(ds)
	case "etsy":
		return s.refreshEtsyData(ds)
	case "jira":
		return s.refreshJiraData(ds)
	default:
		return nil, fmt.Errorf("data source type '%s' does not support refresh", ds.Type)
	}
}

// refreshJiraData performs incremental update for Jira data source
func (s *DataSourceService) refreshJiraData(ds *DataSource) (*RefreshResult, error) {
	s.log(fmt.Sprintf("[JIRA] Starting refresh for data source: %s", ds.Name))

	result := &RefreshResult{
		DataSourceID:   ds.ID,
		DataSourceName: ds.Name,
		TablesUpdated:  make(map[string]int),
		TotalNewRows:   0,
	}

	// Get database path
	if ds.Config.DBPath == "" {
		return nil, fmt.Errorf("data source has no local database")
	}

	dbPath := filepath.Join(s.dataCacheDir, ds.Config.DBPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Normalize base URL
	baseURL := strings.TrimSuffix(ds.Config.JiraBaseUrl, "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	instanceType := ds.Config.JiraInstanceType
	if instanceType == "" {
		instanceType = "cloud"
	}

	client := &http.Client{Timeout: 60 * time.Second}

	// Fetch custom field definitions for proper field mapping
	s.log("[JIRA] Fetching custom field definitions...")
	customFields := s.fetchJiraCustomFields(client, baseURL, ds.Config.JiraUsername, ds.Config.JiraApiToken, instanceType)
	s.log(fmt.Sprintf("[JIRA] Found %d custom fields", len(customFields)))

	// Get the latest issue update time from existing data
	var lastUpdated string
	err = db.QueryRow("SELECT MAX(updated) FROM issues WHERE updated IS NOT NULL").Scan(&lastUpdated)
	if err != nil || lastUpdated == "" {
		s.log("[JIRA] No existing issues found, will fetch all")
		lastUpdated = ""
	} else {
		s.log(fmt.Sprintf("[JIRA] Last updated issue: %s", lastUpdated))
	}

	// Build JQL to fetch only updated issues
	jql := "ORDER BY updated DESC"
	if ds.Config.JiraProjectKey != "" {
		jql = fmt.Sprintf("project = %s ORDER BY updated DESC", ds.Config.JiraProjectKey)
	}
	if lastUpdated != "" {
		// Fetch issues updated after the last known update
		// Add a small buffer to avoid missing issues
		if ds.Config.JiraProjectKey != "" {
			jql = fmt.Sprintf("project = %s AND updated > '%s' ORDER BY updated DESC", ds.Config.JiraProjectKey, lastUpdated[:10])
		} else {
			jql = fmt.Sprintf("updated > '%s' ORDER BY updated DESC", lastUpdated[:10])
		}
	}

	// Fetch updated issues with custom field support
	newIssues, err := s.fetchJiraIssuesForRefresh(client, baseURL, ds.Config.JiraUsername, ds.Config.JiraApiToken, instanceType, jql, customFields)
	if err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch issues: %v", err))
	} else if len(newIssues) > 0 {
		// Update or insert issues
		count, err := s.upsertJiraIssues(db, newIssues)
		if err != nil {
			s.log(fmt.Sprintf("[JIRA] Warning: Failed to upsert issues: %v", err))
		} else {
			result.TablesUpdated["issues"] = count
			result.TotalNewRows += count
			s.log(fmt.Sprintf("[JIRA] Updated %d issues", count))
		}
	}

	// Refresh worklogs
	s.log("[JIRA] Refreshing worklogs...")
	if err := s.refreshJiraWorklogs(client, db, baseURL, ds.Config.JiraUsername, ds.Config.JiraApiToken, instanceType, ds.Config.JiraProjectKey); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to refresh worklogs: %v", err))
	} else {
		s.log("[JIRA] Successfully refreshed worklogs")
	}

	// Refresh projects (full refresh as they don't change often)
	if err := s.refreshJiraProjects(client, db, baseURL, ds.Config.JiraUsername, ds.Config.JiraApiToken, instanceType); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to refresh projects: %v", err))
	}

	// Refresh sprints
	if err := s.refreshJiraSprints(client, db, baseURL, ds.Config.JiraUsername, ds.Config.JiraApiToken, instanceType); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to refresh sprints: %v", err))
	}

	s.log(fmt.Sprintf("[JIRA] Refresh completed. Total new/updated rows: %d", result.TotalNewRows))
	return result, nil
}

// fetchJiraIssuesForRefresh fetches issues for refresh operation with custom field support
func (s *DataSourceService) fetchJiraIssuesForRefresh(client *http.Client, baseURL, username, apiToken, instanceType, jql string, customFields map[string]JiraCustomField) ([]map[string]interface{}, error) {
	allIssues := []map[string]interface{}{}
	startAt := 0
	maxResults := 100

	encodedJQL := strings.ReplaceAll(jql, " ", "%20")
	encodedJQL = strings.ReplaceAll(encodedJQL, "=", "%3D")
	encodedJQL = strings.ReplaceAll(encodedJQL, ">", "%3E")
	encodedJQL = strings.ReplaceAll(encodedJQL, "'", "%27")

	for {
		url := fmt.Sprintf("%s/rest/api/2/search?jql=%s&startAt=%d&maxResults=%d",
			baseURL, encodedJQL, startAt, maxResults)

		resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if issues, ok := result["issues"].([]interface{}); ok {
			for _, issue := range issues {
				if issueMap, ok := issue.(map[string]interface{}); ok {
					flatIssue := s.flattenJiraIssueWithCustomFields(issueMap, customFields)
					allIssues = append(allIssues, flatIssue)
				}
			}
		}

		total := int(result["total"].(float64))
		startAt += maxResults
		if startAt >= total {
			break
		}
	}

	return allIssues, nil
}

// upsertJiraIssues updates or inserts issues into the database
func (s *DataSourceService) upsertJiraIssues(db *sql.DB, issues []map[string]interface{}) (int, error) {
	if len(issues) == 0 {
		return 0, nil
	}

	// Get existing columns
	rows, err := db.Query("PRAGMA table_info(issues)")
	if err != nil {
		return 0, err
	}
	existingCols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
		existingCols[name] = true
	}
	rows.Close()

	// Collect all columns from new data
	allCols := make(map[string]bool)
	for _, issue := range issues {
		for col := range issue {
			allCols[col] = true
		}
	}

	// Add missing columns
	for col := range allCols {
		if !existingCols[col] {
			_, err := db.Exec(fmt.Sprintf("ALTER TABLE issues ADD COLUMN `%s` TEXT", col))
			if err != nil {
				s.log(fmt.Sprintf("[JIRA] Warning: Failed to add column %s: %v", col, err))
			}
		}
	}

	// Upsert issues using key as unique identifier
	updatedCount := 0
	for _, issue := range issues {
		key, ok := issue["key"].(string)
		if !ok || key == "" {
			continue
		}

		// Check if issue exists
		var existingKey string
		err := db.QueryRow("SELECT key FROM issues WHERE key = ?", key).Scan(&existingKey)
		
		if err == sql.ErrNoRows {
			// Insert new issue
			cols := []string{}
			placeholders := []string{}
			values := []interface{}{}
			for col, val := range issue {
				cols = append(cols, fmt.Sprintf("`%s`", col))
				placeholders = append(placeholders, "?")
				values = append(values, s.formatValue(val))
			}
			insertSQL := fmt.Sprintf("INSERT INTO issues (%s) VALUES (%s)", 
				strings.Join(cols, ","), strings.Join(placeholders, ","))
			_, err := db.Exec(insertSQL, values...)
			if err != nil {
				s.log(fmt.Sprintf("[JIRA] Warning: Failed to insert issue %s: %v", key, err))
			} else {
				updatedCount++
			}
		} else if err == nil {
			// Update existing issue
			setClauses := []string{}
			values := []interface{}{}
			for col, val := range issue {
				if col != "key" {
					setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", col))
					values = append(values, s.formatValue(val))
				}
			}
			values = append(values, key)
			updateSQL := fmt.Sprintf("UPDATE issues SET %s WHERE key = ?", strings.Join(setClauses, ", "))
			_, err := db.Exec(updateSQL, values...)
			if err != nil {
				s.log(fmt.Sprintf("[JIRA] Warning: Failed to update issue %s: %v", key, err))
			} else {
				updatedCount++
			}
		}
	}

	return updatedCount, nil
}

// refreshJiraProjects refreshes the projects table
func (s *DataSourceService) refreshJiraProjects(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType string) error {
	url := fmt.Sprintf("%s/rest/api/2/project", baseURL)

	resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var projects []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return err
	}

	// Drop and recreate projects table
	db.Exec("DROP TABLE IF EXISTS projects")

	projectMaps := []map[string]interface{}{}
	for _, p := range projects {
		if proj, ok := p.(map[string]interface{}); ok {
			flatProj := map[string]interface{}{
				"id":   proj["id"],
				"key":  proj["key"],
				"name": proj["name"],
			}
			if lead, ok := proj["lead"].(map[string]interface{}); ok {
				flatProj["lead"] = lead["displayName"]
			}
			if projectType, ok := proj["projectTypeKey"].(string); ok {
				flatProj["project_type"] = projectType
			}
			projectMaps = append(projectMaps, flatProj)
		}
	}

	if len(projectMaps) > 0 {
		return s.createTableFromJSON(db, "projects", projectMaps)
	}
	return nil
}

// refreshJiraWorklogs refreshes the worklogs table with recent entries
func (s *DataSourceService) refreshJiraWorklogs(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType, projectKey string) error {
	// Get the latest worklog update time from existing data
	var lastUpdated string
	err := db.QueryRow("SELECT MAX(updated) FROM worklogs WHERE updated IS NOT NULL").Scan(&lastUpdated)
	if err != nil || lastUpdated == "" {
		s.log("[JIRA] No existing worklogs found, fetching recent worklogs")
	}

	// Build JQL to fetch issues with recent worklogs
	jql := "worklogDate >= -30d ORDER BY updated DESC"
	if projectKey != "" {
		jql = fmt.Sprintf("project = %s AND worklogDate >= -30d ORDER BY updated DESC", projectKey)
	}

	encodedJQL := strings.ReplaceAll(jql, " ", "%20")
	encodedJQL = strings.ReplaceAll(encodedJQL, "=", "%3D")
	encodedJQL = strings.ReplaceAll(encodedJQL, ">", "%3E")
	encodedJQL = strings.ReplaceAll(encodedJQL, "-", "%2D")

	url := fmt.Sprintf("%s/rest/api/2/search?jql=%s&maxResults=100&fields=key,worklog", baseURL, encodedJQL)
	resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	allWorklogs := []map[string]interface{}{}

	if issues, ok := result["issues"].([]interface{}); ok {
		for _, issue := range issues {
			if issueMap, ok := issue.(map[string]interface{}); ok {
				issueKey := fmt.Sprintf("%v", issueMap["key"])

				if fields, ok := issueMap["fields"].(map[string]interface{}); ok {
					if worklog, ok := fields["worklog"].(map[string]interface{}); ok {
						if worklogs, ok := worklog["worklogs"].([]interface{}); ok {
							for _, wl := range worklogs {
								if wlMap, ok := wl.(map[string]interface{}); ok {
									flatWorklog := map[string]interface{}{
										"id":                 wlMap["id"],
										"issue_key":          issueKey,
										"started":            wlMap["started"],
										"time_spent":         wlMap["timeSpent"],
										"time_spent_seconds": wlMap["timeSpentSeconds"],
										"comment":            wlMap["comment"],
										"created":            wlMap["created"],
										"updated":            wlMap["updated"],
									}

									if author, ok := wlMap["author"].(map[string]interface{}); ok {
										flatWorklog["author"] = author["displayName"]
										flatWorklog["author_email"] = author["emailAddress"]
										if accountId, ok := author["accountId"].(string); ok {
											flatWorklog["author_id"] = accountId
										} else if name, ok := author["name"].(string); ok {
											flatWorklog["author_id"] = name
										}
									}

									if updateAuthor, ok := wlMap["updateAuthor"].(map[string]interface{}); ok {
										flatWorklog["update_author"] = updateAuthor["displayName"]
									}

									allWorklogs = append(allWorklogs, flatWorklog)
								}
							}
						}
					}
				}
			}
		}
	}

	if len(allWorklogs) == 0 {
		s.log("[JIRA] No worklogs found to refresh")
		return nil
	}

	// Upsert worklogs
	count, err := s.upsertJiraWorklogs(db, allWorklogs)
	if err != nil {
		return err
	}
	s.log(fmt.Sprintf("[JIRA] Refreshed %d worklogs", count))
	return nil
}

// upsertJiraWorklogs updates or inserts worklogs into the database
func (s *DataSourceService) upsertJiraWorklogs(db *sql.DB, worklogs []map[string]interface{}) (int, error) {
	if len(worklogs) == 0 {
		return 0, nil
	}

	// Check if worklogs table exists
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='worklogs'").Scan(&tableName)
	if err == sql.ErrNoRows {
		// Create table if it doesn't exist
		return len(worklogs), s.createTableFromJSON(db, "worklogs", worklogs)
	}

	// Get existing columns
	rows, err := db.Query("PRAGMA table_info(worklogs)")
	if err != nil {
		return 0, err
	}
	existingCols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
		existingCols[name] = true
	}
	rows.Close()

	// Collect all columns from new data
	allCols := make(map[string]bool)
	for _, wl := range worklogs {
		for col := range wl {
			allCols[col] = true
		}
	}

	// Add missing columns
	for col := range allCols {
		if !existingCols[col] {
			_, err := db.Exec(fmt.Sprintf("ALTER TABLE worklogs ADD COLUMN `%s` TEXT", col))
			if err != nil {
				s.log(fmt.Sprintf("[JIRA] Warning: Failed to add column %s: %v", col, err))
			}
		}
	}

	// Upsert worklogs using id as unique identifier
	updatedCount := 0
	for _, wl := range worklogs {
		id, ok := wl["id"].(string)
		if !ok {
			if idFloat, ok := wl["id"].(float64); ok {
				id = fmt.Sprintf("%.0f", idFloat)
			} else {
				continue
			}
		}

		// Check if worklog exists
		var existingID string
		err := db.QueryRow("SELECT id FROM worklogs WHERE id = ?", id).Scan(&existingID)

		if err == sql.ErrNoRows {
			// Insert new worklog
			cols := []string{}
			placeholders := []string{}
			values := []interface{}{}
			for col, val := range wl {
				cols = append(cols, fmt.Sprintf("`%s`", col))
				placeholders = append(placeholders, "?")
				values = append(values, s.formatValue(val))
			}
			insertSQL := fmt.Sprintf("INSERT INTO worklogs (%s) VALUES (%s)",
				strings.Join(cols, ","), strings.Join(placeholders, ","))
			_, err := db.Exec(insertSQL, values...)
			if err != nil {
				s.log(fmt.Sprintf("[JIRA] Warning: Failed to insert worklog %s: %v", id, err))
			} else {
				updatedCount++
			}
		} else if err == nil {
			// Update existing worklog
			setClauses := []string{}
			values := []interface{}{}
			for col, val := range wl {
				if col != "id" {
					setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", col))
					values = append(values, s.formatValue(val))
				}
			}
			values = append(values, id)
			updateSQL := fmt.Sprintf("UPDATE worklogs SET %s WHERE id = ?", strings.Join(setClauses, ", "))
			_, err := db.Exec(updateSQL, values...)
			if err != nil {
				s.log(fmt.Sprintf("[JIRA] Warning: Failed to update worklog %s: %v", id, err))
			} else {
				updatedCount++
			}
		}
	}

	return updatedCount, nil
}

// refreshJiraSprints refreshes the sprints table
func (s *DataSourceService) refreshJiraSprints(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType string) error {
	boardsURL := fmt.Sprintf("%s/rest/agile/1.0/board", baseURL)

	resp, err := s.makeJiraRequest(client, "GET", boardsURL, username, apiToken, instanceType)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		resp.Body.Close()
		return nil // Agile API not available
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var boardsResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&boardsResult); err != nil {
		resp.Body.Close()
		return err
	}
	resp.Body.Close()

	boards, ok := boardsResult["values"].([]interface{})
	if !ok || len(boards) == 0 {
		return nil
	}

	// Drop and recreate sprints table
	db.Exec("DROP TABLE IF EXISTS sprints")

	allSprints := []map[string]interface{}{}
	seenSprints := make(map[string]bool)

	for _, b := range boards {
		board, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		boardID := fmt.Sprintf("%v", board["id"])
		boardName := fmt.Sprintf("%v", board["name"])
		boardType := ""
		if bt, ok := board["type"].(string); ok {
			boardType = bt
		}

		if boardType != "" && boardType != "scrum" {
			continue
		}

		sprintsURL := fmt.Sprintf("%s/rest/agile/1.0/board/%s/sprint", baseURL, boardID)
		sprintResp, err := s.makeJiraRequest(client, "GET", sprintsURL, username, apiToken, instanceType)
		if err != nil || sprintResp.StatusCode != http.StatusOK {
			if sprintResp != nil {
				sprintResp.Body.Close()
			}
			continue
		}

		var sprintsResult map[string]interface{}
		if err := json.NewDecoder(sprintResp.Body).Decode(&sprintsResult); err != nil {
			sprintResp.Body.Close()
			continue
		}
		sprintResp.Body.Close()

		sprints, ok := sprintsResult["values"].([]interface{})
		if !ok {
			continue
		}

		for _, sp := range sprints {
			sprint, ok := sp.(map[string]interface{})
			if !ok {
				continue
			}

			sprintID := fmt.Sprintf("%v", sprint["id"])
			if seenSprints[sprintID] {
				continue
			}
			seenSprints[sprintID] = true

			flatSprint := map[string]interface{}{
				"id":              sprint["id"],
				"name":            sprint["name"],
				"state":           sprint["state"],
				"start_date":      sprint["startDate"],
				"end_date":        sprint["endDate"],
				"complete_date":   sprint["completeDate"],
				"board_id":        boardID,
				"board_name":      boardName,
				"origin_board_id": sprint["originBoardId"],
				"goal":            sprint["goal"],
			}
			allSprints = append(allSprints, flatSprint)
		}
	}

	if len(allSprints) > 0 {
		return s.createTableFromJSON(db, "sprints", allSprints)
	}
	return nil
}

// ImportJira imports data from Jira REST API (Cloud or Server/Data Center)
func (s *DataSourceService) ImportJira(name string, config DataSourceConfig) (*DataSource, error) {
	s.log(fmt.Sprintf("ImportJira: %s (Instance: %s, Type: %s)", name, config.JiraBaseUrl, config.JiraInstanceType))

	// Validate configuration
	if config.JiraBaseUrl == "" {
		return nil, fmt.Errorf("Jira base URL is required")
	}
	if config.JiraUsername == "" {
		return nil, fmt.Errorf("Jira username/email is required")
	}
	if config.JiraApiToken == "" {
		return nil, fmt.Errorf("Jira API token/password is required")
	}

	// Normalize base URL
	baseURL := strings.TrimSuffix(config.JiraBaseUrl, "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	// Determine instance type (default to cloud)
	instanceType := config.JiraInstanceType
	if instanceType == "" {
		instanceType = "cloud"
	}

	s.log(fmt.Sprintf("[JIRA] Base URL: %s, Instance Type: %s", baseURL, instanceType))

	// Create data source ID
	id := uuid.New().String()

	// Create local storage directory
	relDBDir := filepath.Join("sources", id)
	absDBDir := filepath.Join(s.dataCacheDir, relDBDir)
	if err := os.MkdirAll(absDBDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create SQLite database
	dbName := "data.db"
	relDBPath := filepath.Join(relDBDir, dbName)
	absDBPath := filepath.Join(absDBDir, dbName)

	db, err := sql.Open("sqlite", absDBPath)
	if err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to create local database: %v", err)
	}
	defer db.Close()

	// Fetch and import Jira data
	if err := s.fetchJiraData(db, baseURL, config.JiraUsername, config.JiraApiToken, instanceType, config.JiraProjectKey); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, fmt.Errorf("failed to fetch Jira data: %v", err)
	}

	// Create data source object
	ds := DataSource{
		ID:        id,
		Name:      name,
		Type:      "jira",
		CreatedAt: time.Now().UnixMilli(),
		Config: DataSourceConfig{
			DBPath:           relDBPath,
			JiraInstanceType: instanceType,
			JiraBaseUrl:      config.JiraBaseUrl,
			JiraUsername:     config.JiraUsername,
			JiraApiToken:     config.JiraApiToken,
			JiraProjectKey:   config.JiraProjectKey,
		},
	}

	// Save to registry
	if err := s.AddDataSource(ds); err != nil {
		_ = os.RemoveAll(absDBDir)
		return nil, err
	}

	s.log(fmt.Sprintf("Jira data source imported successfully: %s", name))
	return &ds, nil
}

// fetchJiraData fetches data from Jira REST API and stores it in SQLite
func (s *DataSourceService) fetchJiraData(db *sql.DB, baseURL, username, apiToken, instanceType, projectKey string) error {
	s.log(fmt.Sprintf("[JIRA] Starting data fetch from %s", baseURL))

	// Create HTTP client
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Build JQL query
	jql := "ORDER BY created DESC"
	if projectKey != "" {
		jql = fmt.Sprintf("project = %s ORDER BY created DESC", projectKey)
	}

	importedCount := 0
	var lastError error

	// First, fetch custom field definitions to map field IDs to names
	s.log("[JIRA] Fetching custom field definitions...")
	customFields := s.fetchJiraCustomFields(client, baseURL, username, apiToken, instanceType)
	s.log(fmt.Sprintf("[JIRA] Found %d custom fields", len(customFields)))

	// Fetch Issues (with custom field mapping)
	s.log("[JIRA] Fetching issues...")
	if err := s.fetchJiraIssuesWithCustomFields(client, db, baseURL, username, apiToken, instanceType, jql, customFields); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch issues: %v", err))
		lastError = err
	} else {
		importedCount++
		s.log("[JIRA] Successfully fetched issues")
	}

	// Fetch Worklogs
	s.log("[JIRA] Fetching worklogs...")
	if err := s.fetchJiraWorklogs(client, db, baseURL, username, apiToken, instanceType, projectKey); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch worklogs: %v", err))
		// Don't set lastError for worklogs as it may not be critical
	} else {
		importedCount++
		s.log("[JIRA] Successfully fetched worklogs")
	}

	// Fetch Projects
	s.log("[JIRA] Fetching projects...")
	if err := s.fetchJiraProjects(client, db, baseURL, username, apiToken, instanceType); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch projects: %v", err))
		lastError = err
	} else {
		importedCount++
		s.log("[JIRA] Successfully fetched projects")
	}

	// Fetch Users (if accessible)
	s.log("[JIRA] Fetching users...")
	if err := s.fetchJiraUsers(client, db, baseURL, username, apiToken, instanceType); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch users: %v", err))
		// Don't set lastError for users as it may require admin permissions
	} else {
		importedCount++
		s.log("[JIRA] Successfully fetched users")
	}

	// Fetch Sprints (if Jira Software)
	s.log("[JIRA] Fetching sprints...")
	if err := s.fetchJiraSprints(client, db, baseURL, username, apiToken, instanceType); err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch sprints: %v", err))
		// Don't set lastError for sprints as it requires Jira Software
	} else {
		importedCount++
		s.log("[JIRA] Successfully fetched sprints")
	}

	if importedCount == 0 {
		errMsg := "failed to import any Jira data"
		if lastError != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, lastError)
		}
		s.log(fmt.Sprintf("[JIRA] %s", errMsg))
		return fmt.Errorf(errMsg)
	}

	s.log(fmt.Sprintf("[JIRA] Successfully imported %d resource types", importedCount))
	return nil
}

// JiraCustomField represents a custom field definition
type JiraCustomField struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// fetchJiraCustomFields fetches custom field definitions from Jira
func (s *DataSourceService) fetchJiraCustomFields(client *http.Client, baseURL, username, apiToken, instanceType string) map[string]JiraCustomField {
	result := make(map[string]JiraCustomField)

	url := fmt.Sprintf("%s/rest/api/2/field", baseURL)
	resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
	if err != nil {
		s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch custom fields: %v", err))
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result
	}

	var fields []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return result
	}

	for _, f := range fields {
		if field, ok := f.(map[string]interface{}); ok {
			id := fmt.Sprintf("%v", field["id"])
			name := fmt.Sprintf("%v", field["name"])
			custom, _ := field["custom"].(bool)
			
			if custom && strings.HasPrefix(id, "customfield_") {
				fieldType := ""
				if schema, ok := field["schema"].(map[string]interface{}); ok {
					fieldType = fmt.Sprintf("%v", schema["type"])
				}
				result[id] = JiraCustomField{
					ID:   id,
					Name: name,
					Type: fieldType,
				}
			}
		}
	}

	return result
}

// fetchJiraIssuesWithCustomFields fetches issues with proper custom field handling
func (s *DataSourceService) fetchJiraIssuesWithCustomFields(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType, jql string, customFields map[string]JiraCustomField) error {
	allIssues := []map[string]interface{}{}
	startAt := 0
	maxResults := 100

	encodedJQL := strings.ReplaceAll(jql, " ", "%20")
	encodedJQL = strings.ReplaceAll(encodedJQL, "=", "%3D")

	for {
		url := fmt.Sprintf("%s/rest/api/2/search?jql=%s&startAt=%d&maxResults=%d&expand=changelog",
			baseURL, encodedJQL, startAt, maxResults)

		resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
		if err != nil {
			return fmt.Errorf("request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 401 {
				if instanceType == "cloud" {
					return fmt.Errorf("authentication failed (401): Please verify your email and API token")
				}
				return fmt.Errorf("authentication failed (401): Please verify your username and password")
			}
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %v", err)
		}
		resp.Body.Close()

		if issues, ok := result["issues"].([]interface{}); ok {
			for _, issue := range issues {
				if issueMap, ok := issue.(map[string]interface{}); ok {
					flatIssue := s.flattenJiraIssueWithCustomFields(issueMap, customFields)
					allIssues = append(allIssues, flatIssue)
				}
			}
		}

		total := int(result["total"].(float64))
		startAt += maxResults
		if startAt >= total {
			break
		}
		s.log(fmt.Sprintf("[JIRA] Fetched %d/%d issues...", startAt, total))
	}

	if len(allIssues) == 0 {
		s.log("[JIRA] No issues found")
		return nil
	}

	s.log(fmt.Sprintf("[JIRA] Total %d issues fetched, creating table...", len(allIssues)))
	return s.createTableFromJSON(db, "issues", allIssues)
}

// flattenJiraIssueWithCustomFields flattens a Jira issue with proper custom field naming
func (s *DataSourceService) flattenJiraIssueWithCustomFields(issue map[string]interface{}, customFields map[string]JiraCustomField) map[string]interface{} {
	flat := make(map[string]interface{})

	// Basic fields
	flat["key"] = issue["key"]
	flat["id"] = issue["id"]

	if fields, ok := issue["fields"].(map[string]interface{}); ok {
		// Standard fields
		flat["summary"] = fields["summary"]
		flat["description"] = fields["description"]
		flat["created"] = fields["created"]
		flat["updated"] = fields["updated"]
		flat["resolutiondate"] = fields["resolutiondate"]
		flat["duedate"] = fields["duedate"]

		// Status
		if status, ok := fields["status"].(map[string]interface{}); ok {
			flat["status"] = status["name"]
			if cat, ok := status["statusCategory"].(map[string]interface{}); ok {
				flat["status_category"] = cat["name"]
			}
		}

		// Priority
		if priority, ok := fields["priority"].(map[string]interface{}); ok {
			flat["priority"] = priority["name"]
		}

		// Issue Type
		if issueType, ok := fields["issuetype"].(map[string]interface{}); ok {
			flat["issue_type"] = issueType["name"]
		}

		// Project
		if project, ok := fields["project"].(map[string]interface{}); ok {
			flat["project_key"] = project["key"]
			flat["project_name"] = project["name"]
		}

		// Assignee
		if assignee, ok := fields["assignee"].(map[string]interface{}); ok {
			flat["assignee"] = assignee["displayName"]
			flat["assignee_email"] = assignee["emailAddress"]
			if accountId, ok := assignee["accountId"].(string); ok {
				flat["assignee_id"] = accountId
			} else if name, ok := assignee["name"].(string); ok {
				flat["assignee_id"] = name
			}
		}

		// Reporter
		if reporter, ok := fields["reporter"].(map[string]interface{}); ok {
			flat["reporter"] = reporter["displayName"]
			flat["reporter_email"] = reporter["emailAddress"]
		}

		// Creator
		if creator, ok := fields["creator"].(map[string]interface{}); ok {
			flat["creator"] = creator["displayName"]
		}

		// Resolution
		if resolution, ok := fields["resolution"].(map[string]interface{}); ok {
			flat["resolution"] = resolution["name"]
		}

		// Labels
		if labels, ok := fields["labels"].([]interface{}); ok && len(labels) > 0 {
			labelStrs := make([]string, len(labels))
			for i, l := range labels {
				labelStrs[i] = fmt.Sprintf("%v", l)
			}
			flat["labels"] = strings.Join(labelStrs, ",")
		}

		// Components
		if components, ok := fields["components"].([]interface{}); ok && len(components) > 0 {
			compNames := []string{}
			for _, c := range components {
				if comp, ok := c.(map[string]interface{}); ok {
					compNames = append(compNames, fmt.Sprintf("%v", comp["name"]))
				}
			}
			flat["components"] = strings.Join(compNames, ",")
		}

		// Fix Versions
		if fixVersions, ok := fields["fixVersions"].([]interface{}); ok && len(fixVersions) > 0 {
			versionNames := []string{}
			for _, v := range fixVersions {
				if ver, ok := v.(map[string]interface{}); ok {
					versionNames = append(versionNames, fmt.Sprintf("%v", ver["name"]))
				}
			}
			flat["fix_versions"] = strings.Join(versionNames, ",")
		}

		// Affected Versions
		if versions, ok := fields["versions"].([]interface{}); ok && len(versions) > 0 {
			versionNames := []string{}
			for _, v := range versions {
				if ver, ok := v.(map[string]interface{}); ok {
					versionNames = append(versionNames, fmt.Sprintf("%v", ver["name"]))
				}
			}
			flat["affected_versions"] = strings.Join(versionNames, ",")
		}

		// Time tracking
		if timeTracking, ok := fields["timetracking"].(map[string]interface{}); ok {
			flat["original_estimate"] = timeTracking["originalEstimate"]
			flat["remaining_estimate"] = timeTracking["remainingEstimate"]
			flat["time_spent"] = timeTracking["timeSpent"]
			flat["original_estimate_seconds"] = timeTracking["originalEstimateSeconds"]
			flat["remaining_estimate_seconds"] = timeTracking["remainingEstimateSeconds"]
			flat["time_spent_seconds"] = timeTracking["timeSpentSeconds"]
		}

		// Worklog count
		if worklog, ok := fields["worklog"].(map[string]interface{}); ok {
			if total, ok := worklog["total"].(float64); ok {
				flat["worklog_count"] = int(total)
			}
		}

		// Comment count
		if comment, ok := fields["comment"].(map[string]interface{}); ok {
			if total, ok := comment["total"].(float64); ok {
				flat["comment_count"] = int(total)
			}
		}

		// Subtasks count
		if subtasks, ok := fields["subtasks"].([]interface{}); ok {
			flat["subtask_count"] = len(subtasks)
		}

		// Parent (for subtasks)
		if parent, ok := fields["parent"].(map[string]interface{}); ok {
			flat["parent_key"] = parent["key"]
		}

		// Environment
		if env, ok := fields["environment"].(string); ok {
			flat["environment"] = env
		}

		// Process custom fields with proper naming
		for fieldID, fieldDef := range customFields {
			if val, exists := fields[fieldID]; exists && val != nil {
				// Use sanitized field name instead of ID
				fieldName := s.sanitizeName(fieldDef.Name)
				
				// Handle different field types
				switch v := val.(type) {
				case []interface{}:
					// Array fields (like Sprint, multi-select)
					if len(v) > 0 {
						values := []string{}
						for _, item := range v {
							if itemMap, ok := item.(map[string]interface{}); ok {
								if name, ok := itemMap["name"].(string); ok {
									values = append(values, name)
								} else if value, ok := itemMap["value"].(string); ok {
									values = append(values, value)
								}
							} else {
								values = append(values, fmt.Sprintf("%v", item))
							}
						}
						flat[fieldName] = strings.Join(values, ",")
					}
				case map[string]interface{}:
					// Object fields (like single select, user picker)
					if name, ok := v["name"].(string); ok {
						flat[fieldName] = name
					} else if value, ok := v["value"].(string); ok {
						flat[fieldName] = value
					} else if displayName, ok := v["displayName"].(string); ok {
						flat[fieldName] = displayName
					}
				case string:
					flat[fieldName] = v
				case float64:
					flat[fieldName] = v
				case bool:
					flat[fieldName] = v
				default:
					flat[fieldName] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	return flat
}

// fetchJiraWorklogs fetches worklog entries from Jira
func (s *DataSourceService) fetchJiraWorklogs(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType, projectKey string) error {
	// First get all issue keys to fetch worklogs from
	jql := "worklogDate >= -30d ORDER BY updated DESC" // Last 30 days of worklogs
	if projectKey != "" {
		jql = fmt.Sprintf("project = %s AND worklogDate >= -30d ORDER BY updated DESC", projectKey)
	}

	encodedJQL := strings.ReplaceAll(jql, " ", "%20")
	encodedJQL = strings.ReplaceAll(encodedJQL, "=", "%3D")
	encodedJQL = strings.ReplaceAll(encodedJQL, ">", "%3E")
	encodedJQL = strings.ReplaceAll(encodedJQL, "-", "%2D")

	// Get issues with worklogs
	url := fmt.Sprintf("%s/rest/api/2/search?jql=%s&maxResults=100&fields=key,worklog", baseURL, encodedJQL)
	resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	allWorklogs := []map[string]interface{}{}

	if issues, ok := result["issues"].([]interface{}); ok {
		for _, issue := range issues {
			if issueMap, ok := issue.(map[string]interface{}); ok {
				issueKey := fmt.Sprintf("%v", issueMap["key"])
				
				if fields, ok := issueMap["fields"].(map[string]interface{}); ok {
					if worklog, ok := fields["worklog"].(map[string]interface{}); ok {
						if worklogs, ok := worklog["worklogs"].([]interface{}); ok {
							for _, wl := range worklogs {
								if wlMap, ok := wl.(map[string]interface{}); ok {
									flatWorklog := map[string]interface{}{
										"id":           wlMap["id"],
										"issue_key":    issueKey,
										"started":      wlMap["started"],
										"time_spent":   wlMap["timeSpent"],
										"time_spent_seconds": wlMap["timeSpentSeconds"],
										"comment":      wlMap["comment"],
										"created":      wlMap["created"],
										"updated":      wlMap["updated"],
									}
									
									// Author
									if author, ok := wlMap["author"].(map[string]interface{}); ok {
										flatWorklog["author"] = author["displayName"]
										flatWorklog["author_email"] = author["emailAddress"]
										if accountId, ok := author["accountId"].(string); ok {
											flatWorklog["author_id"] = accountId
										} else if name, ok := author["name"].(string); ok {
											flatWorklog["author_id"] = name
										}
									}
									
									// Update author
									if updateAuthor, ok := wlMap["updateAuthor"].(map[string]interface{}); ok {
										flatWorklog["update_author"] = updateAuthor["displayName"]
									}
									
									allWorklogs = append(allWorklogs, flatWorklog)
								}
							}
						}
					}
				}
			}
		}
	}

	if len(allWorklogs) == 0 {
		s.log("[JIRA] No worklogs found")
		return nil
	}

	s.log(fmt.Sprintf("[JIRA] Total %d worklogs fetched, creating table...", len(allWorklogs)))
	return s.createTableFromJSON(db, "worklogs", allWorklogs)
}

// makeJiraRequest creates and executes a Jira API request with proper authentication
func (s *DataSourceService) makeJiraRequest(client *http.Client, method, url, username, apiToken, instanceType string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Set authentication based on instance type
	// Cloud uses email + API token with Basic Auth
	// Server/Data Center uses username + password with Basic Auth
	req.SetBasicAuth(username, apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return client.Do(req)
}

// fetchJiraProjects fetches projects from Jira
func (s *DataSourceService) fetchJiraProjects(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType string) error {
	url := fmt.Sprintf("%s/rest/api/2/project", baseURL)

	resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var projects []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if len(projects) == 0 {
		s.log("[JIRA] No projects found")
		return nil
	}

	// Convert to map format
	projectMaps := []map[string]interface{}{}
	for _, p := range projects {
		if proj, ok := p.(map[string]interface{}); ok {
			flatProj := map[string]interface{}{
				"id":   proj["id"],
				"key":  proj["key"],
				"name": proj["name"],
			}
			if lead, ok := proj["lead"].(map[string]interface{}); ok {
				flatProj["lead"] = lead["displayName"]
			}
			if projectType, ok := proj["projectTypeKey"].(string); ok {
				flatProj["project_type"] = projectType
			}
			projectMaps = append(projectMaps, flatProj)
		}
	}

	s.log(fmt.Sprintf("[JIRA] Total %d projects fetched, creating table...", len(projectMaps)))
	return s.createTableFromJSON(db, "projects", projectMaps)
}

// fetchJiraUsers fetches users from Jira (requires admin permissions on some instances)
func (s *DataSourceService) fetchJiraUsers(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType string) error {
	// Cloud and Server have different user search endpoints
	var url string
	if instanceType == "cloud" {
		// Jira Cloud: uses /rest/api/3/users/search (API v3 for better user data)
		// Falls back to /rest/api/2/users/search if v3 fails
		url = fmt.Sprintf("%s/rest/api/3/users/search?maxResults=1000", baseURL)
	} else {
		// Jira Server/Data Center: uses different endpoint with username query
		// The "." matches all users
		url = fmt.Sprintf("%s/rest/api/2/user/search?username=.&maxResults=1000&includeInactive=true", baseURL)
	}

	resp, err := s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// If Cloud API v3 fails, try v2
	if resp.StatusCode != http.StatusOK && instanceType == "cloud" {
		s.log("[JIRA] Cloud API v3 failed, trying v2...")
		url = fmt.Sprintf("%s/rest/api/2/users/search?maxResults=1000", baseURL)
		resp, err = s.makeJiraRequest(client, "GET", url, username, apiToken, instanceType)
		if err != nil {
			return fmt.Errorf("request failed: %v", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var users []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if len(users) == 0 {
		s.log("[JIRA] No users found")
		return nil
	}

	// Convert to map format - handle differences between Cloud and Server
	userMaps := []map[string]interface{}{}
	for _, u := range users {
		if user, ok := u.(map[string]interface{}); ok {
			flatUser := map[string]interface{}{}
			
			if instanceType == "cloud" {
				// Cloud uses accountId
				flatUser["account_id"] = user["accountId"]
				flatUser["display_name"] = user["displayName"]
				flatUser["email_address"] = user["emailAddress"]
				flatUser["active"] = user["active"]
				if accountType, ok := user["accountType"].(string); ok {
					flatUser["account_type"] = accountType
				}
			} else {
				// Server uses key/name
				flatUser["user_key"] = user["key"]
				flatUser["username"] = user["name"]
				flatUser["display_name"] = user["displayName"]
				flatUser["email_address"] = user["emailAddress"]
				flatUser["active"] = user["active"]
			}
			userMaps = append(userMaps, flatUser)
		}
	}

	s.log(fmt.Sprintf("[JIRA] Total %d users fetched, creating table...", len(userMaps)))
	return s.createTableFromJSON(db, "users", userMaps)
}

// fetchJiraSprints fetches sprints from Jira Software boards
// Note: This requires Jira Software (not just Jira Core) and the Agile REST API
func (s *DataSourceService) fetchJiraSprints(client *http.Client, db *sql.DB, baseURL, username, apiToken, instanceType string) error {
	// First, get all boards
	// The Agile API endpoint is the same for both Cloud and Server
	boardsURL := fmt.Sprintf("%s/rest/agile/1.0/board", baseURL)

	resp, err := s.makeJiraRequest(client, "GET", boardsURL, username, apiToken, instanceType)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}

	if resp.StatusCode == 404 {
		resp.Body.Close()
		// Agile API not available - likely Jira Core without Software
		s.log("[JIRA] Agile API not available (Jira Software may not be installed)")
		return fmt.Errorf("Agile API not available - Jira Software may not be installed")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var boardsResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&boardsResult); err != nil {
		resp.Body.Close()
		return fmt.Errorf("failed to decode boards response: %v", err)
	}
	resp.Body.Close()

	boards, ok := boardsResult["values"].([]interface{})
	if !ok || len(boards) == 0 {
		s.log("[JIRA] No boards found")
		return nil
	}

	// Fetch sprints from each board
	allSprints := []map[string]interface{}{}
	seenSprints := make(map[string]bool)

	for _, b := range boards {
		board, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		boardID := fmt.Sprintf("%v", board["id"])
		boardName := fmt.Sprintf("%v", board["name"])
		boardType := ""
		if bt, ok := board["type"].(string); ok {
			boardType = bt
		}

		// Only scrum boards have sprints
		if boardType != "" && boardType != "scrum" {
			continue
		}

		// Get sprints for this board
		sprintsURL := fmt.Sprintf("%s/rest/agile/1.0/board/%s/sprint", baseURL, boardID)
		sprintResp, err := s.makeJiraRequest(client, "GET", sprintsURL, username, apiToken, instanceType)
		if err != nil {
			s.log(fmt.Sprintf("[JIRA] Warning: Failed to fetch sprints for board %s: %v", boardName, err))
			continue
		}

		if sprintResp.StatusCode != http.StatusOK {
			sprintResp.Body.Close()
			continue
		}

		var sprintsResult map[string]interface{}
		if err := json.NewDecoder(sprintResp.Body).Decode(&sprintsResult); err != nil {
			sprintResp.Body.Close()
			continue
		}
		sprintResp.Body.Close()

		sprints, ok := sprintsResult["values"].([]interface{})
		if !ok {
			continue
		}

		for _, sp := range sprints {
			sprint, ok := sp.(map[string]interface{})
			if !ok {
				continue
			}

			sprintID := fmt.Sprintf("%v", sprint["id"])
			if seenSprints[sprintID] {
				continue
			}
			seenSprints[sprintID] = true

			flatSprint := map[string]interface{}{
				"id":              sprint["id"],
				"name":            sprint["name"],
				"state":           sprint["state"],
				"start_date":      sprint["startDate"],
				"end_date":        sprint["endDate"],
				"complete_date":   sprint["completeDate"],
				"board_id":        boardID,
				"board_name":      boardName,
				"origin_board_id": sprint["originBoardId"],
				"goal":            sprint["goal"],
			}
			allSprints = append(allSprints, flatSprint)
		}
	}

	if len(allSprints) == 0 {
		s.log("[JIRA] No sprints found")
		return nil
	}

	s.log(fmt.Sprintf("[JIRA] Total %d sprints fetched, creating table...", len(allSprints)))
	return s.createTableFromJSON(db, "sprints", allSprints)
}
