package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateSchema compares the source schema (from the pack) against the target schema (from the data source).
// The source schema only contains tables and columns actually referenced by the pack's SQL steps,
// so validation checks only the necessary tables and fields, not the entire original data source.
//
// Rules:
// - Missing tables in target → incompatible (blocks import)
// - Missing columns in target → warning (does not block import)
// - Extra tables/columns in target → ignored
func ValidateSchema(sourceSchema []PackTableSchema, targetSchema []PackTableSchema) *SchemaValidationResult {
	result := &SchemaValidationResult{
		SourceTableCount: len(sourceSchema),
		TargetTableCount: len(targetSchema),
		TableCountMatch:  len(sourceSchema) == len(targetSchema),
		MissingTables:    []string{},
		MissingColumns:   []MissingColumnInfo{},
		ExtraTables:      []string{},
	}

	// Build a lookup map for target tables: tableName -> set of column names (case-insensitive)
	targetTableMap := make(map[string]map[string]bool)
	for _, t := range targetSchema {
		colSet := make(map[string]bool)
		for _, c := range t.Columns {
			colSet[strings.ToLower(c.Name)] = true
		}
		targetTableMap[strings.ToLower(t.TableName)] = colSet
	}

	// Build a lookup set for source table names (case-insensitive)
	sourceTableSet := make(map[string]bool)
	for _, t := range sourceSchema {
		sourceTableSet[strings.ToLower(t.TableName)] = true
	}

	// Check each source table exists in target
	for _, srcTable := range sourceSchema {
		targetCols, exists := targetTableMap[strings.ToLower(srcTable.TableName)]
		if !exists {
			result.MissingTables = append(result.MissingTables, srcTable.TableName)
			continue
		}

		// Check each source column exists in the target table (case-insensitive)
		for _, srcCol := range srcTable.Columns {
			if !targetCols[strings.ToLower(srcCol.Name)] {
				result.MissingColumns = append(result.MissingColumns, MissingColumnInfo{
					TableName:  srcTable.TableName,
					ColumnName: srcCol.Name,
				})
			}
		}
	}

	// Identify extra tables in target (not in source)
	for _, t := range targetSchema {
		if !sourceTableSet[strings.ToLower(t.TableName)] {
			result.ExtraTables = append(result.ExtraTables, t.TableName)
		}
	}

	// Compatible = true only when there are NO missing tables
	// Missing columns are warnings, not blockers
	result.Compatible = len(result.MissingTables) == 0

	return result
}

// collectFullSchema retrieves the complete schema information (table names, column names, column types)
// for a data source. It uses the DataSourceService to get tables and column details.
func (a *App) collectFullSchema(dataSourceID string) ([]PackTableSchema, error) {
	tables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables for data source %s: %w", dataSourceID, err)
	}

	var schema []PackTableSchema
	for _, tableName := range tables {
		columns, err := a.dataSourceService.GetTableColumns(dataSourceID, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}

		packCols := make([]PackColumnInfo, len(columns))
		for i, col := range columns {
			packCols[i] = PackColumnInfo{
				Name: col.Name,
				Type: col.Type,
			}
		}

		schema = append(schema, PackTableSchema{
			TableName: tableName,
			Columns:   packCols,
		})
	}

	return schema, nil
}

// collectTargetSchemaForPack retrieves schema information only for the tables
// that the pack's SchemaRequirements reference. This avoids fetching column details
// for every table in the target data source when only a few are needed.
// Tables that exist in the target are returned with their columns; tables that
// don't exist are omitted (ValidateSchema will detect them as missing).
func (a *App) collectTargetSchemaForPack(dataSourceID string, requiredTables []PackTableSchema) ([]PackTableSchema, error) {
	// Get the list of all table names in the target (cheap operation)
	allTables, err := a.dataSourceService.GetDataSourceTables(dataSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables for data source %s: %w", dataSourceID, err)
	}

	// Build a case-insensitive lookup of available tables
	availableTableMap := make(map[string]string, len(allTables)) // lowercase -> original name
	for _, t := range allTables {
		availableTableMap[strings.ToLower(t)] = t
	}

	var schema []PackTableSchema
	for _, reqTable := range requiredTables {
		originalName, exists := availableTableMap[strings.ToLower(reqTable.TableName)]
		if !exists {
			// Table doesn't exist in target — skip it here,
			// ValidateSchema will report it as missing
			continue
		}

		columns, err := a.dataSourceService.GetTableColumns(dataSourceID, originalName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", originalName, err)
		}

		packCols := make([]PackColumnInfo, len(columns))
		for i, col := range columns {
			packCols[i] = PackColumnInfo{
				Name: col.Name,
				Type: col.Type,
			}
		}

		schema = append(schema, PackTableSchema{
			TableName: originalName,
			Columns:   packCols,
		})
	}

	return schema, nil
}

// Precompiled regex patterns for extracting table references from SQL.
// Compiled once at package level for performance.
var sqlTableRefPatterns []*regexp.Regexp

func init() {
	tnp := tableNamePattern()
	sqlTableRefPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bFROM\s+` + tnp),
		regexp.MustCompile(`(?i)\bJOIN\s+` + tnp),
		regexp.MustCompile(`(?i)\bINTO\s+` + tnp),
		regexp.MustCompile(`(?i)\bUPDATE\s+` + tnp),
	}
}

// extractReferencedTables parses SQL code from pack steps and extracts the table names
// actually referenced via FROM, JOIN, INTO, and UPDATE clauses.
// Returns a set of table names (lowercased) that the pack's SQL steps depend on.
func extractReferencedTables(steps []PackStep) map[string]bool {
	referenced := make(map[string]bool)

	for _, step := range steps {
		if step.StepType != stepTypeSQL {
			continue
		}
		sql := step.Code
		for _, re := range sqlTableRefPatterns {
			matches := re.FindAllStringSubmatch(sql, -1)
			for _, m := range matches {
				// The table name is in the last non-empty capture group
				tableName := ""
				for i := len(m) - 1; i >= 1; i-- {
					if m[i] != "" {
						tableName = m[i]
						break
					}
				}
				if tableName != "" {
					// Strip backticks, brackets, or double quotes
					tableName = strings.Trim(tableName, "`[]\"")
					referenced[strings.ToLower(tableName)] = true
				}
			}
		}
	}

	return referenced
}

// tableNamePattern returns a regex pattern that matches a SQL table name,
// optionally prefixed with a schema name (schema.table).
// It handles backtick-quoted, bracket-quoted, double-quote-quoted, and unquoted identifiers.
func tableNamePattern() string {
	// An identifier: unquoted word, or `backtick`, or [bracket], or "double-quoted"
	ident := `(?:` +
		"`" + `([^` + "`" + `]+)` + "`" + // backtick-quoted
		`|\[([^\]]+)\]` + // bracket-quoted
		`|"([^"]+)"` + // double-quote-quoted
		`|(\w+)` + // unquoted
		`)`
	// Optional schema prefix: schema.
	return `(?:\w+\.)?` + ident
}

// filterSchemaByReferencedTables filters a full schema to only include tables
// that are actually referenced by the pack's SQL steps.
// If no SQL steps exist (e.g., pure Python pack), returns the full schema unchanged.
func filterSchemaByReferencedTables(fullSchema []PackTableSchema, steps []PackStep) []PackTableSchema {
	referenced := extractReferencedTables(steps)

	// If no tables were extracted (no SQL steps, or parsing found nothing),
	// return the full schema as a safe fallback
	if len(referenced) == 0 {
		return fullSchema
	}

	var filtered []PackTableSchema
	for _, table := range fullSchema {
		if referenced[strings.ToLower(table.TableName)] {
			filtered = append(filtered, table)
		}
	}

	// Safety: if filtering removed everything but we had SQL steps,
	// return full schema (parsing may have missed something)
	if len(filtered) == 0 {
		return fullSchema
	}

	return filtered
}
