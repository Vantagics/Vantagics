package agent

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

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
