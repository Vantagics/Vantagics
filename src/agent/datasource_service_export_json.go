package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSONTableColumn represents a column definition
type JSONTableColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// JSONTable represents a table with schema and data
type JSONTable struct {
	Name    string                   `json:"name"`
	Columns []JSONTableColumn        `json:"columns"`
	Data    []map[string]interface{} `json:"data"`
}

// JSONExportFormat represents the export format
type JSONExportFormat struct {
	Tables []JSONTable `json:"tables"`
}

// ExportToJSON exports one or more tables to a JSON file with schema information
func (s *DataSourceService) ExportToJSON(id string, tableNames []string, outputPath string) error {
	if len(tableNames) == 0 {
		return fmt.Errorf("no tables specified for export")
	}

	// 1. Get data source
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

	// 2. Open database connection
	var db *sql.DB
	var isLocalSQLite bool
	
	if target.Config.DBPath != "" {
		dbPath := filepath.Join(s.dataCacheDir, target.Config.DBPath)
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return err
		}
		isLocalSQLite = true
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
		isLocalSQLite = false
	} else {
		return fmt.Errorf("data source storage not found")
	}
	defer db.Close()

	// 3. Export tables with schema and data
	exportFormat := JSONExportFormat{
		Tables: make([]JSONTable, 0, len(tableNames)),
	}

	for _, tableName := range tableNames {
		// Get column information (pass isLocalSQLite flag)
		columns, err := s.getTableColumns(db, tableName, isLocalSQLite)
		if err != nil {
			return fmt.Errorf("failed to get columns for table %s: %v", tableName, err)
		}

		// Get data
		data, err := s.queryTableAsJSON(db, tableName)
		if err != nil {
			return fmt.Errorf("failed to query table %s: %v", tableName, err)
		}

		exportFormat.Tables = append(exportFormat.Tables, JSONTable{
			Name:    tableName,
			Columns: columns,
			Data:    data,
		})
	}

	// 4. Write to file
	jsonData, err := json.MarshalIndent(exportFormat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	s.log(fmt.Sprintf("Exported %d tables to JSON: %s", len(tableNames), outputPath))
	return nil
}

// getTableColumns retrieves column names and types for a table
func (s *DataSourceService) getTableColumns(db *sql.DB, tableName string, isLocalSQLite bool) ([]JSONTableColumn, error) {
	var columns []JSONTableColumn

	if isLocalSQLite {
		// SQLite: use PRAGMA table_info
		query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
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
				continue
			}
			columns = append(columns, JSONTableColumn{
				Name: name,
				Type: colType,
			})
		}
	} else {
		// MySQL/Doris: use SHOW COLUMNS
		query := fmt.Sprintf("SHOW COLUMNS FROM `%s`", tableName)
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var field, colType string
			var null, key, extra sql.NullString
			var defaultVal sql.NullString
			if err := rows.Scan(&field, &colType, &null, &key, &defaultVal, &extra); err != nil {
				continue
			}
			columns = append(columns, JSONTableColumn{
				Name: field,
				Type: colType,
			})
		}
	}

	return columns, nil
}

// queryTableAsJSON queries a table and returns rows as array of objects
func (s *DataSourceService) queryTableAsJSON(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}

	for rows.Next() {
		// Create a slice of interface{}'s to represent each column
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Create map for this row
		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			if val != nil && *val != nil {
				// Handle []byte for text columns
				if b, ok := (*val).([]byte); ok {
					rowMap[colName] = string(b)
				} else {
					rowMap[colName] = *val
				}
			} else {
				rowMap[colName] = nil
			}
		}

		result = append(result, rowMap)
	}

	return result, nil
}
