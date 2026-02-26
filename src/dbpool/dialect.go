package dbpool

import (
	"fmt"
	"strings"
)

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
// Internal quotes are escaped by doubling them.
func (d *Dialect) QuoteIdent(name string) string {
	switch d.Engine {
	case EngineMySQL:
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	default:
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
}

// ListTablesQuery returns the SQL to list user tables.
func (d *Dialect) ListTablesQuery() string {
	switch d.Engine {
	case EngineDuckDB:
		return "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
	case EngineSQLite:
		return "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	default:
		return "SHOW TABLES"
	}
}

// DescribeColumnsQuery returns the SQL to describe columns for a table.
// For DuckDB, the returned query uses a ? placeholder �caller must pass
// tableName as a query parameter: db.Query(sql, tableName).
// For SQLite/MySQL, the table name is quoted directly in the SQL string.
func (d *Dialect) DescribeColumnsQuery(tableName string) string {
	qi := d.QuoteIdent(tableName)
	switch d.Engine {
	case EngineDuckDB:
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
	case EngineDuckDB, EngineSQLite:
		// PRAGMA table_info works in both DuckDB and SQLite
		return fmt.Sprintf(`PRAGMA table_info(%s)`, qi)
	case EngineMySQL:
		return fmt.Sprintf("DESCRIBE %s", qi)
	default:
		return fmt.Sprintf("DESCRIBE %s", qi)
	}
}
