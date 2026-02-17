package dbpool

import "fmt"

// Dialect provides engine-specific SQL fragments so callers don't need to
// know which engine is in use.
type Dialect struct {
	Engine Engine
}

// NewDialect creates a Dialect for the given engine.
func NewDialect(engine Engine) *Dialect {
	return &Dialect{Engine: engine}
}

// QuoteIdent returns a properly quoted SQL identifier.
// DuckDB/SQLite use double quotes; MySQL uses backticks.
func (d *Dialect) QuoteIdent(name string) string {
	switch d.Engine {
	case EngineMySQL:
		return "`" + name + "`"
	default:
		return `"` + name + `"`
	}
}

// ListTablesQuery returns the SQL to list user tables.
func (d *Dialect) ListTablesQuery() string {
	switch d.Engine {
	case EngineDuckDB:
		return "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
	case EngineSQLite:
		return "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	case EngineMySQL:
		return "SHOW TABLES"
	default:
		return "SHOW TABLES"
	}
}

// DescribeColumnsQuery returns the SQL to describe columns for a table.
// For DuckDB, use a parameterized query with db.Query(sql, tableName).
// For SQLite/MySQL, the table name is quoted in the SQL directly.
func (d *Dialect) DescribeColumnsQuery(tableName string) string {
	qi := d.QuoteIdent(tableName)
	switch d.Engine {
	case EngineDuckDB:
		// Use ? placeholder — caller must pass tableName as a query parameter.
		return "SELECT column_name, data_type, CASE WHEN is_nullable = 'YES' THEN 1 ELSE 0 END AS nullable " +
			"FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position"
	case EngineSQLite:
		return fmt.Sprintf("PRAGMA table_info(%s)", qi)
	case EngineMySQL:
		return fmt.Sprintf("DESCRIBE %s", qi)
	default:
		return fmt.Sprintf("DESCRIBE %s", qi)
	}
}

// ListIndexesQuery returns the SQL to list existing indexes.
func (d *Dialect) ListIndexesQuery() string {
	switch d.Engine {
	case EngineDuckDB:
		return "SELECT index_name FROM duckdb_indexes()"
	case EngineSQLite:
		return "SELECT name FROM sqlite_master WHERE type='index'"
	case EngineMySQL:
		return "" // MySQL indexes are per-table, use SHOW INDEX FROM <table>
	default:
		return ""
	}
}

// TableInfoQuery returns the SQL to get column info for a table (used for
// schema introspection during data append/update operations).
func (d *Dialect) TableInfoQuery(tableName string) string {
	qi := d.QuoteIdent(tableName)
	switch d.Engine {
	case EngineDuckDB:
		// PRAGMA table_info works in DuckDB too
		return fmt.Sprintf(`PRAGMA table_info(%s)`, qi)
	case EngineSQLite:
		return fmt.Sprintf(`PRAGMA table_info(%s)`, qi)
	case EngineMySQL:
		return fmt.Sprintf("DESCRIBE %s", qi)
	default:
		return fmt.Sprintf("DESCRIBE %s", qi)
	}
}
