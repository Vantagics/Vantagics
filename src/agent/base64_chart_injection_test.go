package agent

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildChartPythonCode_Base64Encoding verifies that buildChartPythonCode
// produces valid Python code using base64 encoding for SQL result JSON.
func TestBuildChartPythonCode_Base64Encoding(t *testing.T) {

	t.Run("BasicSQLResult", func(t *testing.T) {
		sqlResult := `[{"name":"Alice","age":30},{"name":"Bob","age":25}]`
		chartCode := "plt.bar(df['name'], df['age'])"

		result := buildChartPythonCode(sqlResult, chartCode)

		if !strings.Contains(result, "import base64") {
			t.Error("Generated code must import base64")
		}
		if !strings.Contains(result, "base64.b64decode") {
			t.Error("Generated code must use base64.b64decode")
		}
		if !strings.Contains(result, chartCode) {
			t.Error("Generated code must contain the user's chart code")
		}
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("SQLResultWithQuotesAndBackslashes", func(t *testing.T) {
		// This is the exact scenario that broke the old string escaping approach
		sqlResult := `[{"desc":"He said \"hello\"","path":"C:\\Users\\test"}]`
		chartCode := "print(df)"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("SQLResultWithNewlines", func(t *testing.T) {
		sqlResult := `[{"text":"line1\nline2\nline3"}]`
		chartCode := "print(df)"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("SQLResultWithUnicode", func(t *testing.T) {
		sqlResult := `[{"name":"张三","city":"北京"},{"name":"李四","city":"上海"}]`
		chartCode := "plt.title('用户分布')"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("EmptySQLResult", func(t *testing.T) {
		sqlResult := `[]`
		chartCode := "print('empty')"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("DictSQLResult", func(t *testing.T) {
		sqlResult := `{"rows":[{"a":1,"b":2}],"total":1}`
		chartCode := "print(df)"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("LargePayload", func(t *testing.T) {
		// Build a large JSON array
		var rows []map[string]interface{}
		for i := 0; i < 500; i++ {
			rows = append(rows, map[string]interface{}{
				"id":    i,
				"value": float64(i) * 3.14,
				"label": strings.Repeat("data", 20),
			})
		}
		sqlResultBytes, _ := json.Marshal(rows)
		sqlResult := string(sqlResultBytes)
		chartCode := "plt.plot(df['id'], df['value'])"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("SQLResultWithTripleQuotes", func(t *testing.T) {
		// Would break triple-quote embedding
		sqlResult := `[{"text":"contains '''triple''' quotes"}]`
		chartCode := "print(df)"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("SQLResultWithPercentSign", func(t *testing.T) {
		// % signs could interfere with fmt.Sprintf
		sqlResult := `[{"growth":"15%","note":"100% complete"}]`
		chartCode := "print(df)"

		result := buildChartPythonCode(sqlResult, chartCode)
		verifyBase64Payload(t, result, sqlResult)
	})

	t.Run("NoOldEscapingPatterns", func(t *testing.T) {
		sqlResult := `[{"a":1}]`
		chartCode := "print(df)"

		result := buildChartPythonCode(sqlResult, chartCode)

		// Verify the old broken pattern is NOT present
		if strings.Contains(result, `json.loads("`) && !strings.Contains(result, "b64decode") {
			t.Error("Must not use old json.loads with raw string embedding")
		}
	})
}

// verifyBase64Payload extracts the base64 string from generated Python code,
// decodes it, and verifies it matches the original JSON string.
func verifyBase64Payload(t *testing.T, code string, originalJSON string) {
	t.Helper()

	prefix := `base64.b64decode("`
	idx := strings.Index(code, prefix)
	if idx < 0 {
		t.Fatal("Could not find base64.b64decode in generated code")
	}
	start := idx + len(prefix)
	end := strings.Index(code[start:], `"`)
	if end < 0 {
		t.Fatal("Could not find closing quote for base64 string")
	}
	b64str := code[start : start+end]

	decoded, err := base64.StdEncoding.DecodeString(b64str)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	// Verify decoded content matches original JSON
	if string(decoded) != originalJSON {
		t.Errorf("Decoded base64 does not match original JSON.\nOriginal: %s\nDecoded:  %s", originalJSON, string(decoded))
	}

	// Verify it's valid JSON
	var parsed interface{}
	if err := json.Unmarshal(decoded, &parsed); err != nil {
		t.Errorf("Decoded base64 is not valid JSON: %v", err)
	}
}
