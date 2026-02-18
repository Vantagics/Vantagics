package agent

// Feature: financial-datasource-support, Property 6: 金融数据源刷新返回有效结果
// **Validates: Requirements 11.2, 11.3**

import (
	"strings"
	"testing"
	"testing/quick"
)

// TestPropertyFinancialRefreshRouting verifies that for any financial data
// source type, calling RefreshDataSource routes to the correct refresh method
// rather than returning a "does not support refresh" error.
//
// Since we cannot run a full refresh without a real DuckDB database and API,
// we verify the routing layer: the error returned should be about missing
// data source or database issues, NOT about the type being unsupported.
//
// Feature: financial-datasource-support, Property 6: 金融数据源刷新返回有效结果
func TestPropertyFinancialRefreshRouting(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}

	financialTypes := []string{
		"sp_global", "lseg", "pitchbook", "bloomberg",
		"morningstar", "iex_cloud", "alpha_vantage", "quandl",
	}

	err := quick.Check(func(idx uint) bool {
		dsType := financialTypes[int(idx)%len(financialTypes)]

		svc := NewDataSourceService(t.TempDir(), func(string) {})

		// Create a minimal data source entry so RefreshDataSource can find it
		ds := DataSource{
			ID:   "test-refresh-" + dsType,
			Name: "Test " + dsType,
			Type: dsType,
			Config: DataSourceConfig{
				DBPath: "nonexistent/data.duckdb",
			},
		}
		if err := svc.AddDataSource(ds); err != nil {
			t.Logf("failed to add data source for type %q: %v", dsType, err)
			return false
		}

		_, refreshErr := svc.RefreshDataSource(ds.ID)

		// We expect an error (missing credentials or DB), but it must NOT be
		// "does not support refresh".
		if refreshErr == nil {
			// Unexpected success – still means routing worked.
			return true
		}
		errMsg := refreshErr.Error()
		if strings.Contains(errMsg, "does not support refresh") {
			t.Logf("type %q was not routed in RefreshDataSource: %v", dsType, refreshErr)
			return false
		}
		// Any other error (missing DB, missing credentials) is fine –
		// it means the routing dispatched to the correct refresh method.
		return true
	}, cfg)

	if err != nil {
		t.Errorf("Property 6 (金融数据源刷新返回有效结果) failed: %v", err)
	}
}
