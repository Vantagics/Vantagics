package agent

// Feature: financial-datasource-support, Property 5: 所有金融数据源类型均可刷新
// **Validates: Requirements 11.1**

import (
	"math/rand"
	"testing"
	"testing/quick"
)

// financialDataSourceTypes is the canonical list of all 8 financial data source driver types.
var financialDataSourceTypes = []string{
	"sp_global", "lseg", "pitchbook", "bloomberg",
	"morningstar", "iex_cloud", "alpha_vantage", "quandl",
}

// randomFinancialType picks a random financial data source type from the canonical list.
func randomFinancialType(r *rand.Rand) string {
	return financialDataSourceTypes[r.Intn(len(financialDataSourceTypes))]
}

// TestPropertyAllFinancialTypesAreRefreshable verifies that for any randomly
// selected financial data source driver type, IsRefreshableDataSource returns true.
//
// Feature: financial-datasource-support, Property 5: 所有金融数据源类型均可刷新
func TestPropertyAllFinancialTypesAreRefreshable(t *testing.T) {
	svc := &DataSourceService{}
	config := &quick.Config{MaxCount: 100}

	// Property: randomly selecting from the 8 financial types always yields refreshable = true
	err := quick.Check(func(idx uint) bool {
		dsType := financialDataSourceTypes[int(idx)%len(financialDataSourceTypes)]
		return svc.IsRefreshableDataSource(dsType)
	}, config)

	if err != nil {
		t.Errorf("Property 5 failed – some financial type is not refreshable: %v", err)
	}
}

// TestPropertyFinancialImpliesRefreshable verifies that for any randomly selected
// financial data source type, if IsFinancialDataSource returns true then
// IsRefreshableDataSource must also return true.
//
// Feature: financial-datasource-support, Property 5: 所有金融数据源类型均可刷新
func TestPropertyFinancialImpliesRefreshable(t *testing.T) {
	svc := &DataSourceService{}
	config := &quick.Config{MaxCount: 100}

	err := quick.Check(func(idx uint) bool {
		dsType := financialDataSourceTypes[int(idx)%len(financialDataSourceTypes)]
		if svc.IsFinancialDataSource(dsType) {
			return svc.IsRefreshableDataSource(dsType)
		}
		// If not financial, the implication is vacuously true
		return true
	}, config)

	if err != nil {
		t.Errorf("Property 5 (implication) failed – a financial type is not refreshable: %v", err)
	}
}
