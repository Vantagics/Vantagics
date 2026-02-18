package agent

import (
	"testing"
)

// TestIsFinancialDataSource_AllFinancialTypes verifies that all 8 financial
// data source types are correctly identified.
func TestIsFinancialDataSource_AllFinancialTypes(t *testing.T) {
	svc := &DataSourceService{}
	financialTypes := []string{
		"sp_global", "lseg", "pitchbook", "bloomberg",
		"morningstar", "iex_cloud", "alpha_vantage", "quandl",
	}
	for _, dsType := range financialTypes {
		if !svc.IsFinancialDataSource(dsType) {
			t.Errorf("IsFinancialDataSource(%q) = false, want true", dsType)
		}
	}
}

// TestIsFinancialDataSource_CaseInsensitive verifies case-insensitive matching.
func TestIsFinancialDataSource_CaseInsensitive(t *testing.T) {
	svc := &DataSourceService{}
	cases := []string{"SP_GLOBAL", "Lseg", "PitchBook", "BLOOMBERG", "IEX_Cloud"}
	for _, dsType := range cases {
		if !svc.IsFinancialDataSource(dsType) {
			t.Errorf("IsFinancialDataSource(%q) = false, want true", dsType)
		}
	}
}

// TestIsFinancialDataSource_NonFinancialTypes verifies that non-financial types return false.
func TestIsFinancialDataSource_NonFinancialTypes(t *testing.T) {
	svc := &DataSourceService{}
	nonFinancial := []string{"shopify", "bigcommerce", "ebay", "etsy", "jira", "csv", "excel", "snowflake", "bigquery", "", "unknown"}
	for _, dsType := range nonFinancial {
		if svc.IsFinancialDataSource(dsType) {
			t.Errorf("IsFinancialDataSource(%q) = true, want false", dsType)
		}
	}
}

// TestIsRefreshableDataSource_IncludesFinancialTypes verifies that all 8 financial
// data source types are refreshable.
func TestIsRefreshableDataSource_IncludesFinancialTypes(t *testing.T) {
	svc := &DataSourceService{}
	financialTypes := []string{
		"sp_global", "lseg", "pitchbook", "bloomberg",
		"morningstar", "iex_cloud", "alpha_vantage", "quandl",
	}
	for _, dsType := range financialTypes {
		if !svc.IsRefreshableDataSource(dsType) {
			t.Errorf("IsRefreshableDataSource(%q) = false, want true", dsType)
		}
	}
}

// TestIsRefreshableDataSource_StillIncludesExistingTypes verifies that existing
// refreshable types (ecommerce + jira) are not broken.
func TestIsRefreshableDataSource_StillIncludesExistingTypes(t *testing.T) {
	svc := &DataSourceService{}
	existing := []string{"shopify", "bigcommerce", "ebay", "etsy", "jira"}
	for _, dsType := range existing {
		if !svc.IsRefreshableDataSource(dsType) {
			t.Errorf("IsRefreshableDataSource(%q) = false, want true", dsType)
		}
	}
}
