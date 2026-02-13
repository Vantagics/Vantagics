package main

import (
	"encoding/json"
	"testing"
)

func TestIsToolOutputFailed_SQLErrors(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"empty", "", false},
		{"success", `{"rows": [{"id": 1}], "success": true}`, false},
		{"no such column", `Error: no such column: s.Sales_Count`, true},
		{"no such table", `no such table: missing_table`, true},
		{"syntax error", `SQL logic error: near "SELEC": syntax error`, true},
		{"ambiguous column", `ambiguous column name: id`, true},
		{"near keyword", `near "FROM": syntax error`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isToolOutputFailed(tt.output); got != tt.want {
				t.Errorf("isToolOutputFailed(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestIsToolOutputFailed_PythonErrors(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"traceback", "Traceback (most recent call last):\n  File ...", true},
		{"module not found", "ModuleNotFoundError: No module named 'pandas'", true},
		{"name error", "NameError: name 'df' is not defined", true},
		{"python not configured", "Python path is not configured", true},
		{"exit status 1", "exit status 1", true},
		{"exit status 0", "exit status 0", false},
		{"type error", "TypeError: unsupported operand type", true},
		{"value error", "ValueError: invalid literal", true},
		{"key error", "KeyError: 'missing_key'", true},
		{"import error", "ImportError: cannot import name 'foo'", true},
		{"attribute error", "AttributeError: 'NoneType' object has no attribute 'bar'", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isToolOutputFailed(tt.output); got != tt.want {
				t.Errorf("isToolOutputFailed(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestIsToolOutputFailed_SuccessField(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"success false", `{"success":false, "error": "something"}`, true},
		{"success true", `{"success":true, "data": []}`, false},
		{"escaped success false", `{"success\":false}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isToolOutputFailed(tt.output); got != tt.want {
				t.Errorf("isToolOutputFailed(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestTruncStr(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := truncStr(tt.input, tt.maxLen); got != tt.want {
				t.Errorf("truncStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestBuildDependsOn(t *testing.T) {
	tests := []struct {
		stepID int
		want   []int
	}{
		{1, nil},
		{2, []int{1}},
		{5, []int{4}},
	}
	for _, tt := range tests {
		got := buildDependsOn(tt.stepID)
		if tt.want == nil && got != nil {
			t.Errorf("buildDependsOn(%d) = %v, want nil", tt.stepID, got)
		}
		if tt.want != nil {
			if len(got) != len(tt.want) || got[0] != tt.want[0] {
				t.Errorf("buildDependsOn(%d) = %v, want %v", tt.stepID, got, tt.want)
			}
		}
	}
}

func TestUnescapeToolInput_ValidJSON(t *testing.T) {
	// Simple case: already valid JSON
	input := `{"query": "SELECT * FROM orders"}`
	result := unescapeToolInput(input)
	if result != input {
		t.Errorf("expected unchanged input for valid JSON, got %q", result)
	}
}

func TestUnescapeToolInput_EscapedQuotes(t *testing.T) {
	// Escaped quotes: {\"query\": \"SELECT * FROM orders\"}
	input := `{\"query\": \"SELECT * FROM orders\"}`
	result := unescapeToolInput(input)
	expected := `{"query": "SELECT * FROM orders"}`
	if result != expected {
		t.Errorf("unescapeToolInput(%q) = %q, want %q", input, result, expected)
	}
}

func TestUnescapeToolInput_EscapedNewlines(t *testing.T) {
	// Escaped newlines in code: {\"code\": \"line1\\nline2\"}
	// After unescape, \\n becomes \n (JSON escape for newline), which is valid JSON
	input := `{\"code\": \"line1\\nline2\"}`
	result := unescapeToolInput(input)
	// The result should be valid JSON — verify by parsing
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("unescapeToolInput result is not valid JSON: %v\nresult: %q", err, result)
		return
	}
	code, ok := parsed["code"].(string)
	if !ok {
		t.Error("expected 'code' field in parsed result")
		return
	}
	if code != "line1\nline2" {
		t.Errorf("expected code='line1\\nline2', got %q", code)
	}
}
