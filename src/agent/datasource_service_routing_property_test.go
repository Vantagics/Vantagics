package agent

// Feature: financial-datasource-support, Property 1: 金融数据源类型路由正确性
// **Validates: Requirements 1.3**
//
// Feature: financial-datasource-support, Property 7: 必填字段验证
// **Validates: Requirements 12.1, 12.2**

import (
	"strings"
	"testing"
	"testing/quick"
)

// allFinancialDriverTypes is the canonical list of 8 financial driver types.
var allFinancialDriverTypes = []string{
	"sp_global", "lseg", "pitchbook", "bloomberg",
	"morningstar", "iex_cloud", "alpha_vantage", "quandl",
}

// --------------------------------------------------------------------------
// Property 1: 金融数据源类型路由正确性
// For any financial data source driver type, calling ImportDataSource should
// route to the corresponding Import method rather than returning
// "unsupported driver type" error.
// --------------------------------------------------------------------------

// TestPropertyFinancialDriverTypeRouting verifies that every financial driver
// type is recognised by ImportDataSource (i.e. does NOT produce an
// "unsupported driver type" error). The error returned should be about
// missing config fields, not about an unknown type.
//
// Feature: financial-datasource-support, Property 1: 金融数据源类型路由正确性
func TestPropertyFinancialDriverTypeRouting(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}

	err := quick.Check(func(idx uint) bool {
		// Pick a financial type deterministically from the random index.
		driverType := allFinancialDriverTypes[int(idx)%len(allFinancialDriverTypes)]

		svc := &DataSourceService{}
		_, importErr := svc.ImportDataSource("test", driverType, DataSourceConfig{}, nil)

		// We expect an error (missing credentials), but it must NOT be
		// "unsupported driver type".
		if importErr == nil {
			// Unexpected success with empty config – still means routing worked.
			return true
		}
		if strings.Contains(importErr.Error(), "unsupported driver type") {
			t.Logf("driver type %q was not routed: %v", driverType, importErr)
			return false
		}
		return true
	}, cfg)

	if err != nil {
		t.Errorf("Property 1 (金融数据源类型路由正确性) failed: %v", err)
	}
}

// --------------------------------------------------------------------------
// Property 7: 必填字段验证
// For any financial data source type, when required API credential fields are
// empty, the import operation should be rejected with a clear error message.
// --------------------------------------------------------------------------

// TestPropertyRequiredFieldValidation verifies that for every financial data
// source type, calling the corresponding Import method with empty credential
// fields returns an error (i.e. the import is rejected at validation time).
//
// The test randomly selects a financial type on each iteration and confirms
// that an all-empty DataSourceConfig is rejected by the Import method.
//
// Feature: financial-datasource-support, Property 7: 必填字段验证
func TestPropertyRequiredFieldValidation(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}

	err := quick.Check(func(idx uint) bool {
		driverType := allFinancialDriverTypes[int(idx)%len(allFinancialDriverTypes)]

		// All credential fields are empty strings — the Import methods must
		// reject this before attempting any I/O or API calls.
		config := DataSourceConfig{}

		svc := &DataSourceService{}

		var importErr error
		switch driverType {
		case "sp_global":
			_, importErr = svc.ImportSPGlobal("test", config)
		case "lseg":
			_, importErr = svc.ImportLSEG("test", config)
		case "pitchbook":
			_, importErr = svc.ImportPitchBook("test", config)
		case "bloomberg":
			_, importErr = svc.ImportBloomberg("test", config)
		case "morningstar":
			_, importErr = svc.ImportMorningstar("test", config)
		case "iex_cloud":
			_, importErr = svc.ImportIEXCloud("test", config)
		case "alpha_vantage":
			_, importErr = svc.ImportAlphaVantage("test", config)
		case "quandl":
			_, importErr = svc.ImportQuandl("test", config)
		}

		if importErr == nil {
			t.Logf("driver type %q accepted empty credentials without error", driverType)
			return false
		}
		return true
	}, cfg)

	if err != nil {
		t.Errorf("Property 7 (必填字段验证) failed: %v", err)
	}
}
