package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ErrorRecord represents a recorded error and its solution
type ErrorRecord struct {
	ID           string `json:"id"`
	Timestamp    int64  `json:"timestamp"`    // Unix timestamp in milliseconds
	ErrorType    string `json:"error_type"`    // "sql", "python", "schema", "timeout"
	ErrorMessage string `json:"error_message"` // Original error
	Context      string    `json:"context"`       // What was being attempted
	Solution     string    `json:"solution"`      // How it was resolved
	Successful   bool      `json:"successful"`    // Was the solution effective?
	Tags         []string  `json:"tags"`          // For similarity matching
}

// UnmarshalJSON implements custom unmarshaling to handle both new (int64) and old (time.Time string) formats
func (er *ErrorRecord) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with Timestamp as interface{} to handle both formats
	type Alias ErrorRecord
	aux := &struct {
		Timestamp interface{} `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(er),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle Timestamp field conversion
	switch v := aux.Timestamp.(type) {
	case float64:
		// Already int64 (JSON numbers are float64)
		er.Timestamp = int64(v)
	case string:
		// Old time.Time format - parse and convert to Unix milliseconds
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			er.Timestamp = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", v); err == nil {
			er.Timestamp = t.UnixMilli()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z07:00", v); err == nil {
			er.Timestamp = t.UnixMilli()
		} else {
			// If parsing fails, use current time
			er.Timestamp = time.Now().UnixMilli()
		}
	case nil:
		// No Timestamp field - use current time
		er.Timestamp = time.Now().UnixMilli()
	default:
		er.Timestamp = time.Now().UnixMilli()
	}

	return nil
}

// ErrorKnowledge manages the error knowledge base
type ErrorKnowledge struct {
	records     []ErrorRecord
	storagePath string
	mu          sync.RWMutex
	logger      func(string)
}

// NewErrorKnowledge creates a new error knowledge manager
func NewErrorKnowledge(dataDir string, logger func(string)) *ErrorKnowledge {
	ek := &ErrorKnowledge{
		records:     make([]ErrorRecord, 0),
		storagePath: filepath.Join(dataDir, "error_knowledge.json"),
		logger:      logger,
	}
	ek.load()
	return ek
}

// load reads existing error records from disk
func (ek *ErrorKnowledge) load() {
	ek.mu.Lock()
	defer ek.mu.Unlock()

	data, err := os.ReadFile(ek.storagePath)
	if err != nil {
		// File doesn't exist yet, start fresh
		return
	}

	if err := json.Unmarshal(data, &ek.records); err != nil {
		if ek.logger != nil {
			ek.logger(fmt.Sprintf("[ERROR-KNOWLEDGE] Failed to load: %v", err))
		}
	}
}

// save writes error records to disk
func (ek *ErrorKnowledge) save() {
	data, err := json.MarshalIndent(ek.records, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(ek.storagePath, data, 0644)
}

// RecordError adds a new error and solution to the knowledge base
func (ek *ErrorKnowledge) RecordError(errorType, errorMsg, context, solution string, successful bool) {
	ek.mu.Lock()
	defer ek.mu.Unlock()

	record := ErrorRecord{
		ID:           fmt.Sprintf("err_%d", time.Now().UnixNano()),
		Timestamp:    time.Now().UnixMilli(),
		ErrorType:    errorType,
		ErrorMessage: truncateString(errorMsg, 500),
		Context:      truncateString(context, 200),
		Solution:     truncateString(solution, 500),
		Successful:   successful,
		Tags:         extractErrorTags(errorMsg),
	}

	ek.records = append(ek.records, record)

	// Keep only last 100 records to prevent bloat
	if len(ek.records) > 100 {
		ek.records = ek.records[len(ek.records)-100:]
	}

	ek.save()

	if ek.logger != nil {
		ek.logger(fmt.Sprintf("[ERROR-KNOWLEDGE] Recorded: %s → %s (success: %v)",
			errorType, truncateString(solution, 50), successful))
	}
}

// FindSimilarErrors finds past errors that match the current error
func (ek *ErrorKnowledge) FindSimilarErrors(errorType, errorMsg string, limit int) []ErrorRecord {
	ek.mu.RLock()
	defer ek.mu.RUnlock()

	var matches []ErrorRecord
	currentTags := extractErrorTags(errorMsg)

	for i := len(ek.records) - 1; i >= 0 && len(matches) < limit; i-- {
		record := ek.records[i]

		// Must be same error type
		if record.ErrorType != errorType {
			continue
		}

		// Must have been successfully resolved
		if !record.Successful {
			continue
		}

		// Check for tag overlap (similarity)
		if hasTagOverlap(record.Tags, currentTags) {
			matches = append(matches, record)
		}
	}

	return matches
}

// GetRecentErrors returns recent error records for context
func (ek *ErrorKnowledge) GetRecentErrors(limit int) []ErrorRecord {
	ek.mu.RLock()
	defer ek.mu.RUnlock()

	if len(ek.records) <= limit {
		return ek.records
	}
	return ek.records[len(ek.records)-limit:]
}

// FormatHintsForLLM formats similar errors as hints for the LLM
func (ek *ErrorKnowledge) FormatHintsForLLM(errorType, errorMsg string) string {
	similar := ek.FindSimilarErrors(errorType, errorMsg, 3)
	if len(similar) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n💡 **Historical Solutions (from past errors):**\n")

	for i, record := range similar {
		sb.WriteString(fmt.Sprintf("%d. **Error:** %s\n", i+1, truncateString(record.ErrorMessage, 100)))
		sb.WriteString(fmt.Sprintf("   **Solution:** %s\n", record.Solution))
	}

	sb.WriteString("\n⚠️ Consider these proven solutions before attempting a fix.\n")
	return sb.String()
}

// extractErrorTags extracts keywords from error message for similarity matching
func extractErrorTags(errorMsg string) []string {
	tags := make([]string, 0)
	lowerMsg := strings.ToLower(errorMsg)

	// SQL error patterns
	sqlPatterns := map[string]string{
		"no such column":     "column_not_found",
		"unknown column":     "column_not_found",
		"no such table":      "table_not_found",
		"syntax error":       "syntax_error",
		"ambiguous column":   "ambiguous_column",
		"division by zero":   "division_zero",
		"null":               "null_handling",
		"type mismatch":      "type_error",
		"year(":              "date_function",
		"month(":             "date_function",
		"strftime":           "date_function",
		"concat":             "string_concat",
		"group by":           "aggregation",
		"subquery":           "subquery",
	}

	// Python error patterns
	pythonPatterns := map[string]string{
		"keyerror":           "key_error",
		"typeerror":          "type_error",
		"valueerror":         "value_error",
		"indexerror":         "index_error",
		"importerror":        "import_error",
		"modulenotfounderror": "module_not_found",
		"filenotfounderror":  "file_not_found",
		"zerodivisionerror":  "division_zero",
		"attributeerror":     "attribute_error",
	}

	for pattern, tag := range sqlPatterns {
		if strings.Contains(lowerMsg, pattern) {
			tags = append(tags, tag)
		}
	}

	for pattern, tag := range pythonPatterns {
		if strings.Contains(lowerMsg, pattern) {
			tags = append(tags, tag)
		}
	}

	return tags
}

// hasTagOverlap checks if two tag sets have any common elements
func hasTagOverlap(tags1, tags2 []string) bool {
	tagSet := make(map[string]bool)
	for _, t := range tags1 {
		tagSet[t] = true
	}
	for _, t := range tags2 {
		if tagSet[t] {
			return true
		}
	}
	return false
}

// GetErrorSummary returns a summary of error knowledge for debugging
func (ek *ErrorKnowledge) GetErrorSummary() string {
	ek.mu.RLock()
	defer ek.mu.RUnlock()

	if len(ek.records) == 0 {
		return "No errors recorded yet."
	}

	typeCounts := make(map[string]int)
	successCount := 0

	for _, r := range ek.records {
		typeCounts[r.ErrorType]++
		if r.Successful {
			successCount++
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 Error Knowledge Base: %d records\n", len(ek.records)))
	sb.WriteString(fmt.Sprintf("✅ Successfully resolved: %d (%.0f%%)\n", successCount, float64(successCount)/float64(len(ek.records))*100))
	sb.WriteString("By type:\n")
	for t, c := range typeCounts {
		sb.WriteString(fmt.Sprintf("  - %s: %d\n", t, c))
	}

	return sb.String()
}
