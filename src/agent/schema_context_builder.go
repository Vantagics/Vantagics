package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// UnifiedSchemaContext represents the schema information for unified code generation
type UnifiedSchemaContext struct {
	DataSourceID  string                    `json:"data_source_id"`
	DatabasePath  string                    `json:"database_path"`
	DatabaseType  string                    `json:"database_type"`
	Tables        []UnifiedTableSchema      `json:"tables"`
	Relationships []UnifiedTableRelationship `json:"relationships"`
	TokenCount    int                       `json:"token_count"`
}

// UnifiedTableSchema represents a single table's schema for unified analysis
type UnifiedTableSchema struct {
	Name       string                   `json:"name"`
	RowCount   int                      `json:"row_count"`
	Columns    []UnifiedColumnInfo      `json:"columns"`
	SampleData []map[string]interface{} `json:"sample_data"`
}

// UnifiedColumnInfo represents column metadata for unified analysis
type UnifiedColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsPK     bool   `json:"is_pk"`
	IsFK     bool   `json:"is_fk"`
	RefTable string `json:"ref_table,omitempty"`
}

// UnifiedTableRelationship represents a relationship between tables
type UnifiedTableRelationship struct {
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
}

// SchemaContextCache holds cached schema context with TTL
type SchemaContextCache struct {
	mu       sync.RWMutex
	cache    map[string]*schemaCacheEntry
	ttl      time.Duration
}

type schemaCacheEntry struct {
	context   *UnifiedSchemaContext
	cachedAt  time.Time
}

// NewSchemaContextCache creates a new schema context cache
func NewSchemaContextCache(ttl time.Duration) *SchemaContextCache {
	return &SchemaContextCache{
		cache: make(map[string]*schemaCacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves a cached schema context if valid
func (c *SchemaContextCache) Get(dataSourceID string) (*UnifiedSchemaContext, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[dataSourceID]
	if !exists {
		return nil, false
	}

	if time.Since(entry.cachedAt) > c.ttl {
		return nil, false
	}

	return entry.context, true
}

// Set stores a schema context in the cache
func (c *SchemaContextCache) Set(dataSourceID string, ctx *UnifiedSchemaContext) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[dataSourceID] = &schemaCacheEntry{
		context:  ctx,
		cachedAt: time.Now(),
	}
}

// Invalidate removes a specific entry from the cache
func (c *SchemaContextCache) Invalidate(dataSourceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, dataSourceID)
}

// Clear removes all entries from the cache
func (c *SchemaContextCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*schemaCacheEntry)
}

// SchemaContextBuilder builds optimized schema context for code generation
type SchemaContextBuilder struct {
	dsService    *DataSourceService
	cache        *SchemaContextCache
	maxTokens    int
	logger       func(string)
}

// NewSchemaContextBuilder creates a new schema context builder
func NewSchemaContextBuilder(dsService *DataSourceService, cacheTTL time.Duration, logger func(string)) *SchemaContextBuilder {
	return &SchemaContextBuilder{
		dsService: dsService,
		cache:     NewSchemaContextCache(cacheTTL),
		maxTokens: 4000, // Default max tokens for schema context
		logger:    logger,
	}
}

func (b *SchemaContextBuilder) log(msg string) {
	if b.logger != nil {
		b.logger(msg)
	}
}

// BuildContext builds schema context for the given data source
func (b *SchemaContextBuilder) BuildContext(ctx context.Context, dataSourceID string, userRequest string) (*UnifiedSchemaContext, error) {
	// Check cache first
	if cached, ok := b.cache.Get(dataSourceID); ok {
		b.log(fmt.Sprintf("[SCHEMA_BUILDER] Cache hit for data source: %s", dataSourceID))
		return cached, nil
	}
	b.log(fmt.Sprintf("[SCHEMA_BUILDER] Cache miss for data source: %s", dataSourceID))

	// Get data source info
	sources, err := b.dsService.LoadDataSources()
	if err != nil {
		return nil, fmt.Errorf("failed to load data sources: %w", err)
	}

	var target *DataSource
	for _, ds := range sources {
		if ds.ID == dataSourceID {
			target = &ds
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("data source not found: %s", dataSourceID)
	}

	// Get database path
	dbPath := ""
	if target.Config.DBPath != "" {
		dbPath = target.Config.DBPath
	}

	// Get all tables
	tables, err := b.dsService.GetDataSourceTables(dataSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Prioritize tables based on user request
	prioritizedTables := b.PrioritizeTables(tables, userRequest, 10)

	// Build table schemas
	var tableSchemas []UnifiedTableSchema
	for _, tableName := range prioritizedTables {
		schema, err := b.buildTableSchema(dataSourceID, tableName)
		if err != nil {
			b.log(fmt.Sprintf("[SCHEMA_BUILDER] Warning: failed to build schema for table %s: %v", tableName, err))
			continue
		}
		tableSchemas = append(tableSchemas, *schema)
	}

	// Detect relationships
	relationships := b.detectRelationships(tableSchemas)

	// Build schema context
	schemaCtx := &UnifiedSchemaContext{
		DataSourceID:  dataSourceID,
		DatabasePath:  dbPath,
		DatabaseType:  target.Type,
		Tables:        tableSchemas,
		Relationships: relationships,
		TokenCount:    b.estimateTokenCount(tableSchemas),
	}

	// Cache the result
	b.cache.Set(dataSourceID, schemaCtx)

	return schemaCtx, nil
}

// buildTableSchema builds schema for a single table
func (b *SchemaContextBuilder) buildTableSchema(dataSourceID string, tableName string) (*UnifiedTableSchema, error) {
	// Get columns with types
	columns, err := b.dsService.GetDataSourceTableColumnsWithTypes(dataSourceID, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Get sample data (3 rows)
	sampleData, err := b.dsService.GetDataSourceTableData(dataSourceID, tableName, 3)
	if err != nil {
		b.log(fmt.Sprintf("[SCHEMA_BUILDER] Warning: failed to get sample data for %s: %v", tableName, err))
		sampleData = nil
	}

	// Get row count
	rowCount, err := b.dsService.GetTableRowCount(dataSourceID, tableName)
	if err != nil {
		b.log(fmt.Sprintf("[SCHEMA_BUILDER] Warning: failed to get row count for %s: %v", tableName, err))
		rowCount = 0
	}

	// Build column info
	var columnInfos []UnifiedColumnInfo
	for _, col := range columns {
		colInfo := UnifiedColumnInfo{
			Name: col.Name,
			Type: col.Type,
			IsPK: b.isPrimaryKey(col.Name),
			IsFK: b.isForeignKey(col.Name),
		}
		if colInfo.IsFK {
			colInfo.RefTable = b.inferRefTable(col.Name)
		}
		columnInfos = append(columnInfos, colInfo)
	}

	return &UnifiedTableSchema{
		Name:       tableName,
		RowCount:   rowCount,
		Columns:    columnInfos,
		SampleData: sampleData,
	}, nil
}

// PrioritizeTables selects most relevant tables based on user request
func (b *SchemaContextBuilder) PrioritizeTables(tables []string, userRequest string, maxTables int) []string {
	if len(tables) <= maxTables {
		return tables
	}

	// Score tables based on relevance to user request
	type tableScore struct {
		name  string
		score int
	}

	var scores []tableScore
	requestLower := strings.ToLower(userRequest)

	for _, table := range tables {
		score := 0
		tableLower := strings.ToLower(table)

		// Direct mention in request
		if strings.Contains(requestLower, tableLower) {
			score += 100
		}

		// Partial match
		words := strings.Fields(requestLower)
		for _, word := range words {
			if len(word) > 2 && strings.Contains(tableLower, word) {
				score += 20
			}
		}

		// Common important table patterns
		importantPatterns := []string{"order", "订单", "sales", "销售", "customer", "客户", "product", "产品", "user", "用户"}
		for _, pattern := range importantPatterns {
			if strings.Contains(tableLower, pattern) {
				score += 10
			}
		}

		scores = append(scores, tableScore{name: table, score: score})
	}

	// Sort by score descending
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Return top tables
	result := make([]string, 0, maxTables)
	for i := 0; i < len(scores) && i < maxTables; i++ {
		result = append(result, scores[i].name)
	}

	return result
}

// detectRelationships detects relationships between tables
func (b *SchemaContextBuilder) detectRelationships(tables []UnifiedTableSchema) []UnifiedTableRelationship {
	var relationships []UnifiedTableRelationship

	// Build a map of table names for quick lookup
	tableNames := make(map[string]bool)
	for _, t := range tables {
		tableNames[strings.ToLower(t.Name)] = true
	}

	// Look for foreign key patterns
	for _, table := range tables {
		for _, col := range table.Columns {
			colLower := strings.ToLower(col.Name)

			// Pattern: table_id or tableId
			if strings.HasSuffix(colLower, "_id") || strings.HasSuffix(colLower, "id") {
				refTable := strings.TrimSuffix(colLower, "_id")
				if refTable == colLower {
					refTable = strings.TrimSuffix(colLower, "id")
				}

				// Check if referenced table exists
				if tableNames[refTable] || tableNames[refTable+"s"] {
					actualRefTable := refTable
					if tableNames[refTable+"s"] {
						actualRefTable = refTable + "s"
					}

					relationships = append(relationships, UnifiedTableRelationship{
						FromTable:  table.Name,
						FromColumn: col.Name,
						ToTable:    actualRefTable,
						ToColumn:   "id",
					})
				}
			}
		}
	}

	return relationships
}

// isPrimaryKey checks if a column is likely a primary key
func (b *SchemaContextBuilder) isPrimaryKey(colName string) bool {
	colLower := strings.ToLower(colName)
	return colLower == "id" || colLower == "pk" || strings.HasSuffix(colLower, "_id") && !strings.Contains(colLower, "_")
}

// isForeignKey checks if a column is likely a foreign key
func (b *SchemaContextBuilder) isForeignKey(colName string) bool {
	colLower := strings.ToLower(colName)
	return (strings.HasSuffix(colLower, "_id") || strings.HasSuffix(colLower, "id")) && colLower != "id"
}

// inferRefTable infers the referenced table from a foreign key column name
func (b *SchemaContextBuilder) inferRefTable(colName string) string {
	colLower := strings.ToLower(colName)
	if strings.HasSuffix(colLower, "_id") {
		return strings.TrimSuffix(colLower, "_id")
	}
	if strings.HasSuffix(colLower, "id") {
		return strings.TrimSuffix(colLower, "id")
	}
	return ""
}

// estimateTokenCount estimates the token count for the schema
func (b *SchemaContextBuilder) estimateTokenCount(tables []UnifiedTableSchema) int {
	// Rough estimation: ~4 characters per token
	totalChars := 0

	for _, table := range tables {
		totalChars += len(table.Name) + 20 // Table name + overhead

		for _, col := range table.Columns {
			totalChars += len(col.Name) + len(col.Type) + 10
		}

		// Sample data estimation
		for _, row := range table.SampleData {
			for k, v := range row {
				totalChars += len(k) + len(fmt.Sprintf("%v", v)) + 5
			}
		}
	}

	return totalChars / 4
}

// FormatForPrompt formats the schema context for LLM prompt
func (b *SchemaContextBuilder) FormatForPrompt(ctx *UnifiedSchemaContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("数据库类型: %s\n", ctx.DatabaseType))
	sb.WriteString(fmt.Sprintf("数据库路径: %s\n\n", ctx.DatabasePath))

	for _, table := range ctx.Tables {
		sb.WriteString(fmt.Sprintf("表名: %s (约 %d 行)\n", table.Name, table.RowCount))
		sb.WriteString("字段:\n")

		for _, col := range table.Columns {
			flags := ""
			if col.IsPK {
				flags += " [主键]"
			}
			if col.IsFK {
				flags += fmt.Sprintf(" [外键->%s]", col.RefTable)
			}
			sb.WriteString(fmt.Sprintf("  - %s (%s)%s\n", col.Name, col.Type, flags))
		}

		if len(table.SampleData) > 0 {
			sb.WriteString("示例数据:\n")
			for i, row := range table.SampleData {
				if i >= 2 { // Limit to 2 sample rows in prompt
					break
				}
				sb.WriteString("  ")
				first := true
				for k, v := range row {
					if !first {
						sb.WriteString(", ")
					}
					sb.WriteString(fmt.Sprintf("%s=%v", k, v))
					first = false
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(ctx.Relationships) > 0 {
		sb.WriteString("表关系:\n")
		for _, rel := range ctx.Relationships {
			sb.WriteString(fmt.Sprintf("  - %s.%s -> %s.%s\n", rel.FromTable, rel.FromColumn, rel.ToTable, rel.ToColumn))
		}
	}

	return sb.String()
}

// InvalidateCache invalidates the cache for a specific data source
func (b *SchemaContextBuilder) InvalidateCache(dataSourceID string) {
	b.cache.Invalidate(dataSourceID)
}

// ClearCache clears all cached schema contexts
func (b *SchemaContextBuilder) ClearCache() {
	b.cache.Clear()
}
