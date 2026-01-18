package agent

import (
	"encoding/json"
	"time"
)

// DataSource represents a registered data source
type DataSource struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Type      string              `json:"type"` // excel, mysql, postgresql, etc.
	CreatedAt int64               `json:"created_at"` // Unix timestamp in milliseconds
	Config    DataSourceConfig    `json:"config"`
	Analysis  *DataSourceAnalysis `json:"analysis,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to handle both new (int64) and old (time.Time string) formats
func (ds *DataSource) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with CreatedAt as interface{} to handle both formats
	type Alias DataSource
	aux := &struct {
		CreatedAt interface{} `json:"created_at"`
		*Alias
	}{
		Alias: (*Alias)(ds),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle CreatedAt field conversion
	switch v := aux.CreatedAt.(type) {
	case float64:
		// Already int64 (JSON numbers are float64)
		ds.CreatedAt = int64(v)
	case int64:
		// Direct int64
		ds.CreatedAt = v
	case int:
		// Convert int to int64
		ds.CreatedAt = int64(v)
	case string:
		// Old time.Time format - parse and convert to Unix milliseconds
		if v == "" {
			ds.CreatedAt = time.Now().UnixMilli()
		} else if t, err := time.Parse(time.RFC3339, v); err == nil {
			ds.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", v); err == nil {
			ds.CreatedAt = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z07:00", v); err == nil {
			ds.CreatedAt = t.UnixMilli()
		} else {
			// If parsing fails, use current time
			ds.CreatedAt = time.Now().UnixMilli()
		}
	case nil:
		// No CreatedAt field - use current time
		ds.CreatedAt = time.Now().UnixMilli()
	default:
		// Try to convert to string and parse, or use current time as fallback
		ds.CreatedAt = time.Now().UnixMilli()
	}

	return nil
}

// DataSourceAnalysis holds the AI-generated analysis of the data source
type DataSourceAnalysis struct {
	Summary string        `json:"summary"`
	Schema  []TableSchema `json:"schema"`
}

// TableSchema represents the schema of a table
type TableSchema struct {
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
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
	StoreLocally      bool               `json:"store_locally"`
	Optimized         bool               `json:"optimized"` // Whether the database has been optimized
	MySQLExportConfig *MySQLExportConfig `json:"mysql_export_config,omitempty"`
}
