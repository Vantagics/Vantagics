package main

import (
	"fmt"
)

// ValidateSchema compares the source schema (from the pack) against the target schema (from the data source).
// It returns a SchemaValidationResult indicating compatibility.
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

	// Build a lookup map for target tables: tableName -> set of column names
	targetTableMap := make(map[string]map[string]bool)
	for _, t := range targetSchema {
		colSet := make(map[string]bool)
		for _, c := range t.Columns {
			colSet[c.Name] = true
		}
		targetTableMap[t.TableName] = colSet
	}

	// Build a lookup set for source table names
	sourceTableSet := make(map[string]bool)
	for _, t := range sourceSchema {
		sourceTableSet[t.TableName] = true
	}

	// Check each source table exists in target
	for _, srcTable := range sourceSchema {
		targetCols, exists := targetTableMap[srcTable.TableName]
		if !exists {
			result.MissingTables = append(result.MissingTables, srcTable.TableName)
			continue
		}

		// Check each source column exists in the target table
		for _, srcCol := range srcTable.Columns {
			if !targetCols[srcCol.Name] {
				result.MissingColumns = append(result.MissingColumns, MissingColumnInfo{
					TableName:  srcTable.TableName,
					ColumnName: srcCol.Name,
				})
			}
		}
	}

	// Identify extra tables in target (not in source)
	for _, t := range targetSchema {
		if !sourceTableSet[t.TableName] {
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
