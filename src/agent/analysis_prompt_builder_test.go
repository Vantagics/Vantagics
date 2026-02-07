package agent

import (
	"strings"
	"testing"
	"testing/quick"
)

// **Validates: Requirements 1.1, 1.5, 1.6, 2.1, 2.2, 2.3, 2.4, 2.5**
// Property 1: Prompt Content Completeness
// Property 2: Classification Hints Affect Prompt

// TestAnalysisPromptBuilder_Property_PromptContentCompleteness tests that generated prompts contain all required sections
// **Validates: Requirements 1.1, 1.5, 1.6, 2.1, 2.3, 2.4, 2.5**
func TestAnalysisPromptBuilder_Property_PromptContentCompleteness(t *testing.T) {
	// Property: For any user request and schema context, the generated prompt SHALL contain:
	// - A "分析要求" section with ⭐⭐⭐ markers for visualization
	// - plt.savefig() instructions for chart saving
	// - FILES_DIR and os.path.join() usage instructions
	// - A "重要警告" section about file saving
	// - Code examples showing correct chart saving patterns

	property := func(seed uint16) bool {
		builder := NewAnalysisPromptBuilder()

		// Generate test data based on seed
		userRequest := generateUserRequest(seed)
		schemaContext := generateSchemaContext(seed)

		// Test with hints that require visualization
		hints := &ClassificationResult{
			NeedsVisualization: true,
			SuggestedChartType: getChartType(seed),
			NeedsDataExport:    seed%2 == 0,
			SuggestedOutputs:   []string{"chart", "table"},
			Reasoning:          "Test reasoning",
		}

		prompt := builder.BuildPromptWithHints(userRequest, schemaContext, "visualization", hints)

		// Property 1: Must contain "分析要求" section with ⭐⭐⭐ markers
		if !strings.Contains(prompt, "分析要求") {
			t.Logf("Missing '分析要求' section")
			return false
		}
		if !strings.Contains(prompt, "⭐⭐⭐") {
			t.Logf("Missing ⭐⭐⭐ markers for visualization emphasis")
			return false
		}

		// Property 2: Must contain plt.savefig() instructions
		if !strings.Contains(prompt, "plt.savefig") {
			t.Logf("Missing plt.savefig() instructions")
			return false
		}

		// Property 3: Must contain FILES_DIR usage instructions
		if !strings.Contains(prompt, "FILES_DIR") {
			t.Logf("Missing FILES_DIR instructions")
			return false
		}

		// Property 4: Must contain os.path.join() usage
		if !strings.Contains(prompt, "os.path.join") {
			t.Logf("Missing os.path.join() usage")
			return false
		}

		// Property 5: Must contain "重要警告" section
		if !strings.Contains(prompt, "重要警告") {
			t.Logf("Missing '重要警告' section")
			return false
		}

		// Property 6: Must contain code examples with chart saving pattern
		if !strings.Contains(prompt, "chart_path") {
			t.Logf("Missing chart_path code example")
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestAnalysisPromptBuilder_Property_ClassificationHintsAffectPrompt tests that hints affect prompt content
// **Validates: Requirements 2.2**
func TestAnalysisPromptBuilder_Property_ClassificationHintsAffectPrompt(t *testing.T) {
	// Property: For any classification result with NeedsVisualization=true,
	// the generated prompt SHALL include chart type recommendations based on SuggestedChartType.

	property := func(seed uint8) bool {
		builder := NewAnalysisPromptBuilder()

		userRequest := "分析销售数据"
		schemaContext := &UnifiedSchemaContext{
			DatabaseType: "sqlite",
			Tables: []UnifiedTableSchema{
				{Name: "sales", Columns: []UnifiedColumnInfo{{Name: "amount", Type: "REAL"}}},
			},
		}

		chartTypes := []string{"line", "bar", "pie", "grouped_bar", "scatter", "heatmap"}
		chartType := chartTypes[int(seed)%len(chartTypes)]

		hints := &ClassificationResult{
			NeedsVisualization: true,
			SuggestedChartType: chartType,
		}

		prompt := builder.BuildPromptWithHints(userRequest, schemaContext, "visualization", hints)

		// Property: When NeedsVisualization=true and SuggestedChartType is set,
		// the prompt should contain chart type recommendation
		expectedDescriptions := map[string]string{
			"line":        "折线图",
			"bar":         "柱状图",
			"pie":         "饼图",
			"grouped_bar": "分组柱状图",
			"scatter":     "散点图",
			"heatmap":     "热力图",
		}

		if desc, ok := expectedDescriptions[chartType]; ok {
			if !strings.Contains(prompt, desc) {
				t.Logf("Missing chart type description '%s' for type '%s'", desc, chartType)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestAnalysisPromptBuilder_Property_DefaultVisualizationEncouragement tests that prompts without hints still encourage visualization
// **Validates: Requirements 2.5**
func TestAnalysisPromptBuilder_Property_DefaultVisualizationEncouragement(t *testing.T) {
	// Property: When hints are nil, the prompt should still contain visualization suggestions

	property := func(seed uint16) bool {
		builder := NewAnalysisPromptBuilder()

		userRequest := generateUserRequest(seed)
		schemaContext := generateSchemaContext(seed)

		// Test WITHOUT hints (nil)
		prompt := builder.BuildPromptWithHints(userRequest, schemaContext, "standard", nil)

		// Property: Even without hints, should encourage visualization
		if !strings.Contains(prompt, "建议生成可视化图表") {
			t.Logf("Missing visualization encouragement when hints are nil")
			return false
		}

		// Property: Should still contain plt.savefig instructions
		if !strings.Contains(prompt, "plt.savefig") {
			t.Logf("Missing plt.savefig instructions when hints are nil")
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestAnalysisPromptBuilder_Property_DataExportInstructions tests that data export hints affect prompt
// **Validates: Requirements 2.3, 2.4**
func TestAnalysisPromptBuilder_Property_DataExportInstructions(t *testing.T) {
	// Property: When NeedsDataExport=true, the prompt should contain Excel export instructions

	property := func(seed uint8) bool {
		builder := NewAnalysisPromptBuilder()

		userRequest := "导出销售数据"
		schemaContext := &UnifiedSchemaContext{
			DatabaseType: "sqlite",
			Tables: []UnifiedTableSchema{
				{Name: "sales", Columns: []UnifiedColumnInfo{{Name: "amount", Type: "REAL"}}},
			},
		}

		hints := &ClassificationResult{
			NeedsVisualization: false,
			NeedsDataExport:    true,
		}

		prompt := builder.BuildPromptWithHints(userRequest, schemaContext, "standard", hints)

		// Property: When NeedsDataExport=true, should contain Excel export instructions
		if !strings.Contains(prompt, "to_excel") {
			t.Logf("Missing to_excel instructions when NeedsDataExport=true")
			return false
		}

		// Property: Should contain export path example
		if !strings.Contains(prompt, "export_path") || !strings.Contains(prompt, "xlsx") {
			t.Logf("Missing export path example")
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestAnalysisPromptBuilder_Property_SchemaContextIncluded tests that schema context is properly included
// **Validates: Requirements 1.1**
func TestAnalysisPromptBuilder_Property_SchemaContextIncluded(t *testing.T) {
	// Property: The generated prompt should include all table names and column information from schema context

	property := func(seed uint16) bool {
		builder := NewAnalysisPromptBuilder()

		// Generate schema with multiple tables
		numTables := int(seed%5) + 1
		tables := make([]UnifiedTableSchema, numTables)
		for i := 0; i < numTables; i++ {
			tableName := generateTableName(uint16(i) + seed)
			tables[i] = UnifiedTableSchema{
				Name: tableName,
				Columns: []UnifiedColumnInfo{
					{Name: "id", Type: "INTEGER", IsPK: true},
					{Name: "value", Type: "REAL"},
				},
				RowCount: int(seed) * (i + 1),
			}
		}

		schemaContext := &UnifiedSchemaContext{
			DatabaseType: "sqlite",
			Tables:       tables,
		}

		prompt := builder.BuildPromptWithHints("分析数据", schemaContext, "standard", nil)

		// Property: All table names should be in the prompt
		for _, table := range tables {
			if !strings.Contains(prompt, table.Name) {
				t.Logf("Missing table name '%s' in prompt", table.Name)
				return false
			}
		}

		// Property: Should contain "数据库Schema" section
		if !strings.Contains(prompt, "数据库Schema") {
			t.Logf("Missing '数据库Schema' section")
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Helper functions for generating test data

func generateUserRequest(seed uint16) string {
	requests := []string{
		"分析销售数据",
		"查看用户增长趋势",
		"统计订单数量",
		"计算收入分布",
		"分析客户行为",
	}
	return requests[int(seed)%len(requests)]
}

func generateSchemaContext(seed uint16) *UnifiedSchemaContext {
	return &UnifiedSchemaContext{
		DatabaseType: "sqlite",
		Tables: []UnifiedTableSchema{
			{
				Name: generateTableName(seed),
				Columns: []UnifiedColumnInfo{
					{Name: "id", Type: "INTEGER", IsPK: true},
					{Name: "name", Type: "TEXT"},
					{Name: "value", Type: "REAL"},
					{Name: "created_at", Type: "TEXT"},
				},
				RowCount: int(seed) * 100,
			},
		},
	}
}

func generateTableName(seed uint16) string {
	names := []string{"sales", "orders", "customers", "products", "transactions"}
	return names[int(seed)%len(names)]
}

func getChartType(seed uint16) string {
	types := []string{"line", "bar", "pie", "grouped_bar", "scatter", "heatmap"}
	return types[int(seed)%len(types)]
}
