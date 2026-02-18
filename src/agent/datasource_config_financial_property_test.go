package agent

// Feature: financial-datasource-support, Property 4: 金融数据源配置序列化往返一致性
// **Validates: Requirements 10.1, 10.3**

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

// generateRandomString produces a random string of length [0, maxLen) from printable ASCII.
func generateRandomString(r *rand.Rand, maxLen int) string {
	n := r.Intn(maxLen)
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(r.Intn(94) + 32) // printable ASCII 32-125
	}
	return string(buf)
}

// Generate implements quick.Generator for DataSourceConfig so that testing/quick
// can produce random instances with populated financial fields.
func (DataSourceConfig) Generate(r *rand.Rand, size int) reflect.Value {
	cfg := DataSourceConfig{
		FinancialProvider:    generateRandomString(r, size),
		FinancialAPIKey:      generateRandomString(r, size),
		FinancialAPISecret:   generateRandomString(r, size),
		FinancialToken:       generateRandomString(r, size),
		FinancialUsername:    generateRandomString(r, size),
		FinancialPassword:    generateRandomString(r, size),
		FinancialDatasets:    generateRandomString(r, size),
		FinancialSymbols:     generateRandomString(r, size),
		FinancialDataType:    generateRandomString(r, size),
		FinancialDatasetCode: generateRandomString(r, size),
		FinancialCertPath:    generateRandomString(r, size),
		FinancialEnvironment: generateRandomString(r, size),
	}
	return reflect.ValueOf(cfg)
}

// TestPropertyFinancialConfigSerializationRoundTrip verifies that for any valid
// DataSourceConfig with random financial field values, serializing to JSON and
// deserializing back produces an equivalent object — all 12 financial fields
// remain unchanged.
//
// Feature: financial-datasource-support, Property 4: 金融数据源配置序列化往返一致性
func TestPropertyFinancialConfigSerializationRoundTrip(t *testing.T) {
	config := &quick.Config{MaxCount: 100}

	err := quick.Check(func(original DataSourceConfig) bool {
		// Serialize to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Logf("Marshal failed: %v", err)
			return false
		}

		// Deserialize back
		var restored DataSourceConfig
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Logf("Unmarshal failed: %v", err)
			return false
		}

		// Assert all 12 financial fields match
		if original.FinancialProvider != restored.FinancialProvider {
			t.Logf("FinancialProvider mismatch: %q vs %q", original.FinancialProvider, restored.FinancialProvider)
			return false
		}
		if original.FinancialAPIKey != restored.FinancialAPIKey {
			t.Logf("FinancialAPIKey mismatch: %q vs %q", original.FinancialAPIKey, restored.FinancialAPIKey)
			return false
		}
		if original.FinancialAPISecret != restored.FinancialAPISecret {
			t.Logf("FinancialAPISecret mismatch: %q vs %q", original.FinancialAPISecret, restored.FinancialAPISecret)
			return false
		}
		if original.FinancialToken != restored.FinancialToken {
			t.Logf("FinancialToken mismatch: %q vs %q", original.FinancialToken, restored.FinancialToken)
			return false
		}
		if original.FinancialUsername != restored.FinancialUsername {
			t.Logf("FinancialUsername mismatch: %q vs %q", original.FinancialUsername, restored.FinancialUsername)
			return false
		}
		if original.FinancialPassword != restored.FinancialPassword {
			t.Logf("FinancialPassword mismatch: %q vs %q", original.FinancialPassword, restored.FinancialPassword)
			return false
		}
		if original.FinancialDatasets != restored.FinancialDatasets {
			t.Logf("FinancialDatasets mismatch: %q vs %q", original.FinancialDatasets, restored.FinancialDatasets)
			return false
		}
		if original.FinancialSymbols != restored.FinancialSymbols {
			t.Logf("FinancialSymbols mismatch: %q vs %q", original.FinancialSymbols, restored.FinancialSymbols)
			return false
		}
		if original.FinancialDataType != restored.FinancialDataType {
			t.Logf("FinancialDataType mismatch: %q vs %q", original.FinancialDataType, restored.FinancialDataType)
			return false
		}
		if original.FinancialDatasetCode != restored.FinancialDatasetCode {
			t.Logf("FinancialDatasetCode mismatch: %q vs %q", original.FinancialDatasetCode, restored.FinancialDatasetCode)
			return false
		}
		if original.FinancialCertPath != restored.FinancialCertPath {
			t.Logf("FinancialCertPath mismatch: %q vs %q", original.FinancialCertPath, restored.FinancialCertPath)
			return false
		}
		if original.FinancialEnvironment != restored.FinancialEnvironment {
			t.Logf("FinancialEnvironment mismatch: %q vs %q", original.FinancialEnvironment, restored.FinancialEnvironment)
			return false
		}

		return true
	}, config)

	if err != nil {
		t.Errorf("Property 4 failed: %v", err)
	}
}
