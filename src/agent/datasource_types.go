package agent

import "time"

// DataSource represents a registered data source
type DataSource struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Type      string              `json:"type"` // excel, mysql, postgresql, etc.
	CreatedAt time.Time           `json:"created_at"`
	Config    DataSourceConfig    `json:"config"`
	Analysis  *DataSourceAnalysis `json:"analysis,omitempty"`
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
	MySQLExportConfig *MySQLExportConfig `json:"mysql_export_config,omitempty"`
}
