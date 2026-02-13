package main

import (
	"testing"
)

func TestExtractReferencedTables_BasicSQL(t *testing.T) {
	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM orders WHERE id > 10"},
		{StepID: 2, StepType: "sql_query", Code: "SELECT o.id, c.name FROM orders o JOIN customers c ON o.customer_id = c.id"},
	}

	refs := extractReferencedTables(steps)

	if !refs["orders"] {
		t.Error("expected 'orders' to be referenced")
	}
	if !refs["customers"] {
		t.Error("expected 'customers' to be referenced")
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 referenced tables, got %d", len(refs))
	}
}

func TestExtractReferencedTables_CaseInsensitive(t *testing.T) {
	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM Orders"},
		{StepID: 2, StepType: "sql_query", Code: "select * from ORDERS"},
	}

	refs := extractReferencedTables(steps)

	if !refs["orders"] {
		t.Error("expected 'orders' to be referenced (case-insensitive)")
	}
	if len(refs) != 1 {
		t.Errorf("expected 1 unique referenced table, got %d", len(refs))
	}
}

func TestExtractReferencedTables_SkipsPythonSteps(t *testing.T) {
	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM orders"},
		{StepID: 2, StepType: "python_code", Code: "df = pd.read_sql('SELECT * FROM products', conn)"},
	}

	refs := extractReferencedTables(steps)

	if !refs["orders"] {
		t.Error("expected 'orders' to be referenced")
	}
	// Python steps should not be parsed
	if refs["products"] {
		t.Error("did not expect 'products' from python step")
	}
}

func TestExtractReferencedTables_JoinVariants(t *testing.T) {
	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: `
			SELECT a.id, b.name, c.total
			FROM orders a
			LEFT JOIN customers b ON a.cid = b.id
			INNER JOIN payments c ON a.id = c.order_id
		`},
	}

	refs := extractReferencedTables(steps)

	expected := []string{"orders", "customers", "payments"}
	for _, tbl := range expected {
		if !refs[tbl] {
			t.Errorf("expected '%s' to be referenced", tbl)
		}
	}
}

func TestExtractReferencedTables_BacktickQuoted(t *testing.T) {
	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM `order items` WHERE qty > 0"},
	}

	refs := extractReferencedTables(steps)

	if !refs["order items"] {
		t.Error("expected 'order items' to be referenced")
	}
}

func TestExtractReferencedTables_SchemaPrefix(t *testing.T) {
	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM dbo.orders"},
	}

	refs := extractReferencedTables(steps)

	if !refs["orders"] {
		t.Error("expected 'orders' to be referenced (with schema prefix)")
	}
}

func TestExtractReferencedTables_EmptySteps(t *testing.T) {
	refs := extractReferencedTables(nil)
	if len(refs) != 0 {
		t.Errorf("expected 0 referenced tables for nil steps, got %d", len(refs))
	}
}

func TestFilterSchemaByReferencedTables_FiltersCorrectly(t *testing.T) {
	fullSchema := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{{Name: "id", Type: "int"}, {Name: "total", Type: "decimal"}}},
		{TableName: "customers", Columns: []PackColumnInfo{{Name: "id", Type: "int"}, {Name: "name", Type: "varchar"}}},
		{TableName: "products", Columns: []PackColumnInfo{{Name: "id", Type: "int"}, {Name: "sku", Type: "varchar"}}},
		{TableName: "inventory", Columns: []PackColumnInfo{{Name: "id", Type: "int"}}},
	}

	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT * FROM orders JOIN customers ON orders.cid = customers.id"},
	}

	filtered := filterSchemaByReferencedTables(fullSchema, steps)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered tables, got %d", len(filtered))
	}

	tableNames := map[string]bool{}
	for _, t := range filtered {
		tableNames[t.TableName] = true
	}
	if !tableNames["orders"] || !tableNames["customers"] {
		t.Errorf("expected orders and customers, got %v", tableNames)
	}
}

func TestFilterSchemaByReferencedTables_NoSQLSteps_ReturnsFullSchema(t *testing.T) {
	fullSchema := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{{Name: "id", Type: "int"}}},
		{TableName: "products", Columns: []PackColumnInfo{{Name: "id", Type: "int"}}},
	}

	steps := []PackStep{
		{StepID: 1, StepType: "python_code", Code: "print('hello')"},
	}

	filtered := filterSchemaByReferencedTables(fullSchema, steps)

	if len(filtered) != len(fullSchema) {
		t.Errorf("expected full schema (%d tables) for pure Python pack, got %d", len(fullSchema), len(filtered))
	}
}

func TestFilterSchemaByReferencedTables_EmptySteps_ReturnsFullSchema(t *testing.T) {
	fullSchema := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{{Name: "id", Type: "int"}}},
	}

	filtered := filterSchemaByReferencedTables(fullSchema, nil)

	if len(filtered) != len(fullSchema) {
		t.Errorf("expected full schema for nil steps, got %d tables", len(filtered))
	}
}

func TestFilterSchemaByReferencedTables_PreservesColumns(t *testing.T) {
	fullSchema := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{
			{Name: "id", Type: "int"},
			{Name: "total", Type: "decimal"},
			{Name: "status", Type: "varchar"},
		}},
		{TableName: "products", Columns: []PackColumnInfo{{Name: "id", Type: "int"}}},
	}

	steps := []PackStep{
		{StepID: 1, StepType: "sql_query", Code: "SELECT id, total FROM orders"},
	}

	filtered := filterSchemaByReferencedTables(fullSchema, steps)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 table, got %d", len(filtered))
	}
	// All columns of the referenced table should be preserved
	if len(filtered[0].Columns) != 3 {
		t.Errorf("expected 3 columns preserved for orders, got %d", len(filtered[0].Columns))
	}
}
