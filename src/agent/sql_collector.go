package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SchemaContext represents database schema information used in SQL generation
type SchemaContext struct {
	Tables  []string            `json:"tables"`
	Columns map[string][]string `json:"columns"`
}

// SQLPair represents a successful user request and corresponding SQL query
type SQLPair struct {
	Timestamp       int64         `json:"timestamp"`
	UserRequest     string        `json:"user_request"`      // Original user request for reference
	SQLIntent       string        `json:"sql_intent"`         // Specific intent from LLM reasoning
	SchemaContext   SchemaContext `json:"schema_context"`
	SQL             string        `json:"sql"`
	ExecutionTimeMs int64         `json:"execution_time_ms,omitempty"`
}

// SQLCollectionData represents all SQL pairs collected in a session
type SQLCollectionData struct {
	ThreadID       string    `json:"thread_id"`
	DataSourceID   string    `json:"data_source_id"`
	DataSourceName string    `json:"data_source_name"`
	CollectionTime int64     `json:"collection_time"`
	SQLPairs       []SQLPair `json:"sql_pairs"`
}

// SQLCollector collects successful SQL executions for training data
type SQLCollector struct {
	threadID       string
	dataSourceID   string
	dataSourceName string
	sqlPairs       []SQLPair
	schemaContext  SchemaContext
	currentRequest string // Track current user request
	currentIntent  string // Track current SQL intent from LLM reasoning
}

// NewSQLCollector creates a new SQL collector for a session
func NewSQLCollector(threadID, dataSourceID, dataSourceName string) *SQLCollector {
	return &SQLCollector{
		threadID:       threadID,
		dataSourceID:   dataSourceID,
		dataSourceName: dataSourceName,
		sqlPairs:       []SQLPair{},
		schemaContext: SchemaContext{
			Tables:  []string{},
			Columns: make(map[string][]string),
		},
	}
}

// SetUserRequest sets the current user request being processed
func (c *SQLCollector) SetUserRequest(request string) {
	c.currentRequest = request
}

// SetSQLIntent sets the specific intent for the upcoming SQL execution
// This should be called with the LLM's reasoning/thinking before tool call
func (c *SQLCollector) SetSQLIntent(intent string) {
	c.currentIntent = intent
}

// SetSchemaContext sets the schema context for the current analysis
func (c *SQLCollector) SetSchemaContext(context SchemaContext) {
	c.schemaContext = context
}

// AddTable adds a table to the schema context
func (c *SQLCollector) AddTable(tableName string, columns []string) {
	if c.schemaContext.Columns == nil {
		c.schemaContext.Columns = make(map[string][]string)
	}
	
	// Check if table already exists
	exists := false
	for _, t := range c.schemaContext.Tables {
		if t == tableName {
			exists = true
			break
		}
	}
	
	if !exists {
		c.schemaContext.Tables = append(c.schemaContext.Tables, tableName)
	}
	c.schemaContext.Columns[tableName] = columns
}

// RecordSuccessfulSQL records a successful SQL execution
func (c *SQLCollector) RecordSuccessfulSQL(sql string, executionTimeMs int64) {
	if c.currentRequest == "" || sql == "" {
		return // Skip if no current request or empty SQL
	}

	pair := SQLPair{
		Timestamp:       time.Now().Unix(),
		UserRequest:     c.currentRequest,
		SQLIntent:       c.currentIntent,
		SchemaContext:   c.schemaContext,
		SQL:             sql,
		ExecutionTimeMs: executionTimeMs,
	}

	c.sqlPairs = append(c.sqlPairs, pair)
}

// SaveToFile saves collected data to a JSON file in the session directory
func (c *SQLCollector) SaveToFile(sessionDir string) error {
	if len(c.sqlPairs) == 0 {
		// No SQL pairs collected, skip saving
		return nil
	}

	// Create SQL directory if it doesn't exist
	sqlDir := filepath.Join(sessionDir, "sql")
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create sql directory: %v", err)
	}

	// Prepare collection data
	data := SQLCollectionData{
		ThreadID:       c.threadID,
		DataSourceID:   c.dataSourceID,
		DataSourceName: c.dataSourceName,
		CollectionTime: time.Now().Unix(),
		SQLPairs:       c.sqlPairs,
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SQL collection data: %v", err)
	}

	// Write to file
	filePath := filepath.Join(sqlDir, "txt2sql_data.json")
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write SQL collection file: %v", err)
	}

	return nil
}

// GetPairCount returns the number of SQL pairs collected
func (c *SQLCollector) GetPairCount() int {
	return len(c.sqlPairs)
}
