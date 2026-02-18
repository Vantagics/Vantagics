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
	// Shopify API configuration
	ShopifyStore       string `json:"shopify_store,omitempty"`
	ShopifyAccessToken string `json:"shopify_access_token,omitempty"`
	ShopifyAPIVersion  string `json:"shopify_api_version,omitempty"`
	// BigCommerce API configuration
	BigCommerceStoreHash    string `json:"bigcommerce_store_hash,omitempty"`
	BigCommerceAccessToken  string `json:"bigcommerce_access_token,omitempty"`
	// eBay API configuration
	EbayAccessToken    string `json:"ebay_access_token,omitempty"`
	EbayEnvironment    string `json:"ebay_environment,omitempty"` // "production" or "sandbox"
	EbayApiFulfillment bool   `json:"ebay_api_fulfillment,omitempty"`
	EbayApiFinances    bool   `json:"ebay_api_finances,omitempty"`
	EbayApiAnalytics   bool   `json:"ebay_api_analytics,omitempty"`
	// Etsy API configuration
	EtsyShopId      string `json:"etsy_shop_id,omitempty"`      // Auto-detected if not provided
	EtsyAccessToken string `json:"etsy_access_token,omitempty"`
	// Jira configuration (Cloud and Server/Data Center)
	JiraInstanceType string `json:"jira_instance_type,omitempty"` // "cloud" or "server"
	JiraBaseUrl      string `json:"jira_base_url,omitempty"`
	JiraUsername     string `json:"jira_username,omitempty"` // Email for Cloud, username for Server
	JiraApiToken     string `json:"jira_api_token,omitempty"` // API token for Cloud, password for Server
	JiraProjectKey   string `json:"jira_project_key,omitempty"` // Optional: specific project to import
	// Snowflake configuration
	SnowflakeAccount   string `json:"snowflake_account,omitempty"`   // Account identifier (e.g., xy12345.us-east-1)
	SnowflakeUser      string `json:"snowflake_user,omitempty"`
	SnowflakePassword  string `json:"snowflake_password,omitempty"`
	SnowflakeWarehouse string `json:"snowflake_warehouse,omitempty"` // Optional: compute warehouse
	SnowflakeDatabase  string `json:"snowflake_database,omitempty"`  // Optional: default database
	SnowflakeSchema    string `json:"snowflake_schema,omitempty"`    // Optional: default schema
	SnowflakeRole      string `json:"snowflake_role,omitempty"`      // Optional: role to use
	// BigQuery configuration
	BigQueryProjectID   string `json:"bigquery_project_id,omitempty"`   // GCP Project ID
	BigQueryDatasetID   string `json:"bigquery_dataset_id,omitempty"`   // Optional: specific dataset
	BigQueryCredentials string `json:"bigquery_credentials,omitempty"`  // JSON service account key
	// Financial data source configuration
	FinancialProvider    string `json:"financial_provider,omitempty"`     // Financial data provider identifier
	FinancialAPIKey      string `json:"financial_api_key,omitempty"`      // API Key
	FinancialAPISecret   string `json:"financial_api_secret,omitempty"`   // API Secret
	FinancialToken       string `json:"financial_token,omitempty"`        // OAuth Token or Access Token
	FinancialUsername    string `json:"financial_username,omitempty"`     // Username (required by LSEG etc.)
	FinancialPassword    string `json:"financial_password,omitempty"`     // Password (required by LSEG etc.)
	FinancialDatasets    string `json:"financial_datasets,omitempty"`     // Selected datasets, comma-separated
	FinancialSymbols     string `json:"financial_symbols,omitempty"`      // Stock/currency symbols, comma-separated
	FinancialDataType    string `json:"financial_data_type,omitempty"`    // Data type selection (e.g. Alpha Vantage time series type)
	FinancialDatasetCode string `json:"financial_dataset_code,omitempty"` // Quandl dataset code (e.g. WIKI/AAPL)
	FinancialCertPath    string `json:"financial_cert_path,omitempty"`    // Bloomberg certificate path
	FinancialEnvironment string `json:"financial_environment,omitempty"`  // Environment selection (sandbox/production)
}

// DataSourceStatistics holds aggregated statistics about data sources
type DataSourceStatistics struct {
	TotalCount      int                  `json:"total_count"`
	BreakdownByType map[string]int       `json:"breakdown_by_type"`
	DataSources     []DataSourceSummary  `json:"data_sources"`
}

// DataSourceSummary provides minimal info for selection UI
type DataSourceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
