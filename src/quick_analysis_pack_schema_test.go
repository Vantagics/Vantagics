package main

import (
	"testing"
)

func TestValidateSchema_FullyCompatible(t *testing.T) {
	source := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
			},
		},
		{
			TableName: "customers",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
			},
		},
	}

	target := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
			},
		},
		{
			TableName: "customers",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
			},
		},
	}

	result := ValidateSchema(source, target)

	if !result.Compatible {
		t.Error("expected Compatible to be true")
	}
	if !result.TableCountMatch {
		t.Error("expected TableCountMatch to be true")
	}
	if result.SourceTableCount != 2 {
		t.Errorf("expected SourceTableCount=2, got %d", result.SourceTableCount)
	}
	if result.TargetTableCount != 2 {
		t.Errorf("expected TargetTableCount=2, got %d", result.TargetTableCount)
	}
	if len(result.MissingTables) != 0 {
		t.Errorf("expected no missing tables, got %v", result.MissingTables)
	}
	if len(result.MissingColumns) != 0 {
		t.Errorf("expected no missing columns, got %v", result.MissingColumns)
	}
}

func TestValidateSchema_MissingTable(t *testing.T) {
	source := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
		{TableName: "products", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
	}

	target := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
	}

	result := ValidateSchema(source, target)

	if result.Compatible {
		t.Error("expected Compatible to be false when table is missing")
	}
	if len(result.MissingTables) != 1 || result.MissingTables[0] != "products" {
		t.Errorf("expected MissingTables=[products], got %v", result.MissingTables)
	}
	if result.SourceTableCount != 2 {
		t.Errorf("expected SourceTableCount=2, got %d", result.SourceTableCount)
	}
	if result.TargetTableCount != 1 {
		t.Errorf("expected TargetTableCount=1, got %d", result.TargetTableCount)
	}
	if result.TableCountMatch {
		t.Error("expected TableCountMatch to be false")
	}
}

func TestValidateSchema_MissingColumns(t *testing.T) {
	source := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
				{Name: "status", Type: "TEXT"},
			},
		},
	}

	target := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
			},
		},
	}

	result := ValidateSchema(source, target)

	// Missing columns are warnings, NOT blockers
	if !result.Compatible {
		t.Error("expected Compatible to be true (missing columns are warnings, not blockers)")
	}
	if len(result.MissingColumns) != 2 {
		t.Errorf("expected 2 missing columns, got %d: %v", len(result.MissingColumns), result.MissingColumns)
	}

	// Verify the missing columns are correct
	missingColNames := map[string]bool{}
	for _, mc := range result.MissingColumns {
		missingColNames[mc.ColumnName] = true
		if mc.TableName != "orders" {
			t.Errorf("expected TableName=orders, got %s", mc.TableName)
		}
	}
	if !missingColNames["amount"] {
		t.Error("expected 'amount' in missing columns")
	}
	if !missingColNames["status"] {
		t.Error("expected 'status' in missing columns")
	}
}

func TestValidateSchema_ExtraTablesAndColumnsIgnored(t *testing.T) {
	source := []PackTableSchema{
		{
			TableName: "orders",
			Columns:   []PackColumnInfo{{Name: "id", Type: "INTEGER"}},
		},
	}

	target := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "extra_col", Type: "TEXT"}, // extra column
			},
		},
		{
			TableName: "extra_table", // extra table
			Columns:   []PackColumnInfo{{Name: "id", Type: "INTEGER"}},
		},
	}

	result := ValidateSchema(source, target)

	if !result.Compatible {
		t.Error("expected Compatible to be true (extra tables/columns should be ignored)")
	}
	if len(result.MissingTables) != 0 {
		t.Errorf("expected no missing tables, got %v", result.MissingTables)
	}
	if len(result.MissingColumns) != 0 {
		t.Errorf("expected no missing columns, got %v", result.MissingColumns)
	}
	if len(result.ExtraTables) != 1 || result.ExtraTables[0] != "extra_table" {
		t.Errorf("expected ExtraTables=[extra_table], got %v", result.ExtraTables)
	}
	if result.TableCountMatch {
		t.Error("expected TableCountMatch to be false (1 source vs 2 target)")
	}
}

func TestValidateSchema_EmptySourceSchema(t *testing.T) {
	source := []PackTableSchema{}
	target := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
	}

	result := ValidateSchema(source, target)

	if !result.Compatible {
		t.Error("expected Compatible to be true (empty source has no requirements)")
	}
	if result.SourceTableCount != 0 {
		t.Errorf("expected SourceTableCount=0, got %d", result.SourceTableCount)
	}
	if result.TargetTableCount != 1 {
		t.Errorf("expected TargetTableCount=1, got %d", result.TargetTableCount)
	}
}

func TestValidateSchema_EmptyTargetSchema(t *testing.T) {
	source := []PackTableSchema{
		{TableName: "orders", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
	}
	target := []PackTableSchema{}

	result := ValidateSchema(source, target)

	if result.Compatible {
		t.Error("expected Compatible to be false (target has no tables)")
	}
	if len(result.MissingTables) != 1 || result.MissingTables[0] != "orders" {
		t.Errorf("expected MissingTables=[orders], got %v", result.MissingTables)
	}
}

func TestValidateSchema_BothEmpty(t *testing.T) {
	result := ValidateSchema([]PackTableSchema{}, []PackTableSchema{})

	if !result.Compatible {
		t.Error("expected Compatible to be true (both empty)")
	}
	if !result.TableCountMatch {
		t.Error("expected TableCountMatch to be true (both 0)")
	}
	if result.SourceTableCount != 0 || result.TargetTableCount != 0 {
		t.Error("expected both counts to be 0")
	}
}

func TestValidateSchema_MixedMissingTablesAndColumns(t *testing.T) {
	source := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "total", Type: "REAL"},
			},
		},
		{
			TableName: "products",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
			},
		},
		{
			TableName: "categories",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
			},
		},
	}

	target := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				// "total" is missing
			},
		},
		// "products" table is missing entirely
		{
			TableName: "categories",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
			},
		},
	}

	result := ValidateSchema(source, target)

	if result.Compatible {
		t.Error("expected Compatible to be false (products table is missing)")
	}
	if len(result.MissingTables) != 1 || result.MissingTables[0] != "products" {
		t.Errorf("expected MissingTables=[products], got %v", result.MissingTables)
	}
	if len(result.MissingColumns) != 1 {
		t.Errorf("expected 1 missing column, got %d", len(result.MissingColumns))
	}
	if len(result.MissingColumns) == 1 {
		if result.MissingColumns[0].TableName != "orders" || result.MissingColumns[0].ColumnName != "total" {
			t.Errorf("expected missing column orders.total, got %s.%s",
				result.MissingColumns[0].TableName, result.MissingColumns[0].ColumnName)
		}
	}
}

func TestValidateSchema_AllTablesMissing(t *testing.T) {
	source := []PackTableSchema{
		{TableName: "a", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
		{TableName: "b", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
	}
	target := []PackTableSchema{
		{TableName: "x", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
		{TableName: "y", Columns: []PackColumnInfo{{Name: "id", Type: "INTEGER"}}},
	}

	result := ValidateSchema(source, target)

	if result.Compatible {
		t.Error("expected Compatible to be false")
	}
	if len(result.MissingTables) != 2 {
		t.Errorf("expected 2 missing tables, got %d", len(result.MissingTables))
	}
	if len(result.ExtraTables) != 2 {
		t.Errorf("expected 2 extra tables, got %d", len(result.ExtraTables))
	}
}

func TestValidateSchema_CaseInsensitive(t *testing.T) {
	source := []PackTableSchema{
		{
			TableName: "Orders",
			Columns: []PackColumnInfo{
				{Name: "ID", Type: "INTEGER"},
				{Name: "Amount", Type: "REAL"},
			},
		},
	}

	target := []PackTableSchema{
		{
			TableName: "orders",
			Columns: []PackColumnInfo{
				{Name: "id", Type: "INTEGER"},
				{Name: "amount", Type: "REAL"},
			},
		},
	}

	result := ValidateSchema(source, target)

	if !result.Compatible {
		t.Error("expected Compatible to be true (case-insensitive matching)")
	}
	if len(result.MissingTables) != 0 {
		t.Errorf("expected no missing tables, got %v", result.MissingTables)
	}
	if len(result.MissingColumns) != 0 {
		t.Errorf("expected no missing columns, got %v", result.MissingColumns)
	}
}

func TestValidateSchema_CaseInsensitiveMixedCase(t *testing.T) {
	source := []PackTableSchema{
		{
			TableName: "CUSTOMER_ORDERS",
			Columns: []PackColumnInfo{
				{Name: "Order_ID", Type: "INTEGER"},
				{Name: "CUSTOMER_NAME", Type: "TEXT"},
				{Name: "total_amount", Type: "REAL"},
			},
		},
	}

	target := []PackTableSchema{
		{
			TableName: "customer_orders",
			Columns: []PackColumnInfo{
				{Name: "order_id", Type: "INTEGER"},
				{Name: "customer_name", Type: "TEXT"},
				{Name: "Total_Amount", Type: "REAL"},
			},
		},
	}

	result := ValidateSchema(source, target)

	if !result.Compatible {
		t.Error("expected Compatible to be true (case-insensitive matching)")
	}
	if len(result.MissingColumns) != 0 {
		t.Errorf("expected no missing columns, got %v", result.MissingColumns)
	}
}
