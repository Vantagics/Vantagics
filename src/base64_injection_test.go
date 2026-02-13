package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildDataFrameInjection_Base64Encoding verifies that buildDataFrameInjection
// produces valid Python code using base64 encoding for all types of JSON data.
func TestBuildDataFrameInjection_Base64Encoding(t *testing.T) {
	app := &App{}

	t.Run("SimpleListResult", func(t *testing.T) {
		sqlResult := []map[string]interface{}{
			{"name": "Alice", "age": 30},
			{"name": "Bob", "age": 25},
		}
		chartCode := "plt.bar(df['name'], df['age'])"

		result := app.buildDataFrameInjection(sqlResult, chartCode)

		// Must contain base64 import
		if !strings.Contains(result, "import base64") {
			t.Error("Generated code must import base64")
		}
		// Must use base64.b64decode
		if !strings.Contains(result, "base64.b64decode") {
			t.Error("Generated code must use base64.b64decode")
		}
		// Must NOT use old-style json.loads("...") with raw JSON
		if strings.Contains(result, `json.loads("[{`) || strings.Contains(result, `json.loads("[{`) {
			t.Error("Generated code must NOT embed raw JSON in string literals")
		}
		// Must contain the chart code
		if !strings.Contains(result, chartCode) {
			t.Error("Generated code must contain the user's chart code")
		}
		// Verify the embedded base64 decodes to valid JSON
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("JSONWithSpecialCharacters", func(t *testing.T) {
		// This is the exact case that broke the old escaping approach:
		// JSON values containing quotes, backslashes, newlines
		sqlResult := []map[string]interface{}{
			{"description": `He said "hello" and left`},
			{"path": `C:\Users\test\file.txt`},
			{"note": "line1\nline2\ttab"},
		}
		chartCode := "print(df)"

		result := app.buildDataFrameInjection(sqlResult, chartCode)

		// The old approach would produce broken Python here.
		// With base64, the special characters are safely encoded.
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("JSONWithUnicodeCharacters", func(t *testing.T) {
		sqlResult := []map[string]interface{}{
			{"name": "张三", "city": "北京"},
			{"name": "李四", "city": "上海"},
		}
		chartCode := "plt.title('用户分布')"

		result := app.buildDataFrameInjection(sqlResult, chartCode)
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("EmptyResult", func(t *testing.T) {
		sqlResult := []map[string]interface{}{}
		chartCode := "print('no data')"

		result := app.buildDataFrameInjection(sqlResult, chartCode)

		if !strings.Contains(result, "base64.b64decode") {
			t.Error("Even empty results should use base64 encoding")
		}
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("LargeDataset", func(t *testing.T) {
		// Simulate a large dataset with many rows
		sqlResult := make([]map[string]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			sqlResult[i] = map[string]interface{}{
				"id":    i,
				"value": float64(i) * 1.5,
				"label": strings.Repeat("x", 100),
			}
		}
		chartCode := "plt.plot(df['id'], df['value'])"

		result := app.buildDataFrameInjection(sqlResult, chartCode)
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("NestedJSONValues", func(t *testing.T) {
		// JSON with nested structures (edge case)
		sqlResult := []map[string]interface{}{
			{"data": map[string]interface{}{"nested": "value", "count": 42}},
		}
		chartCode := "print(df)"

		result := app.buildDataFrameInjection(sqlResult, chartCode)
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("JSONWithTripleQuotes", func(t *testing.T) {
		// This would break the old triple-quote approach used in templates
		sqlResult := []map[string]interface{}{
			{"text": "contains '''triple quotes''' inside"},
			{"code": "x = '''multiline\nstring'''"},
		}
		chartCode := "print(df)"

		result := app.buildDataFrameInjection(sqlResult, chartCode)
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("JSONWithBackslashQuoteCombinations", func(t *testing.T) {
		// The exact pattern that caused the original bug:
		// When JSON contains \" sequences, the old ReplaceAll chain would corrupt them
		sqlResult := []map[string]interface{}{
			{"regex": `\d+\.\d+`},
			{"path": `C:\new\test`},
			{"escaped": `value with \"quotes\" inside`},
		}
		chartCode := "print(df)"

		result := app.buildDataFrameInjection(sqlResult, chartCode)
		verifyBase64InCode(t, result, sqlResult)
	})

	t.Run("NilResult", func(t *testing.T) {
		// When sqlResult is nil, json.Marshal produces "null"
		chartCode := "print('fallback')"
		result := app.buildDataFrameInjection(nil, chartCode)

		// Should still produce valid code with base64
		if !strings.Contains(result, "base64.b64decode") {
			t.Error("Nil result should still use base64 encoding")
		}
		if !strings.Contains(result, chartCode) {
			t.Error("Chart code must be preserved")
		}
	})

	t.Run("DictResult", func(t *testing.T) {
		// ExecuteSQL returns []map, but the interface{} could be a dict
		sqlResult := map[string]interface{}{
			"rows": []map[string]interface{}{
				{"a": 1, "b": 2},
			},
		}
		chartCode := "print(df)"

		result := app.buildDataFrameInjection(sqlResult, chartCode)
		verifyBase64InCode(t, result, sqlResult)
	})
}

// verifyBase64InCode extracts the base64 string from generated Python code,
// decodes it, and verifies it matches the original data.
func verifyBase64InCode(t *testing.T, code string, originalData interface{}) {
	t.Helper()

	// Extract the base64 string from: base64.b64decode("XXXXX")
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

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(b64str)
	if err != nil {
		t.Fatalf("Failed to decode base64 string: %v", err)
	}

	// Verify it's valid JSON
	var decodedData interface{}
	if err := json.Unmarshal(decoded, &decodedData); err != nil {
		t.Fatalf("Decoded base64 is not valid JSON: %v\nDecoded string: %s", err, string(decoded))
	}

	// Verify it matches the original data by re-marshaling both
	originalJSON, _ := json.Marshal(originalData)
	decodedJSON, _ := json.Marshal(decodedData)

	if string(originalJSON) != string(decodedJSON) {
		t.Errorf("Decoded data does not match original.\nOriginal: %s\nDecoded:  %s", originalJSON, decodedJSON)
	}
}
