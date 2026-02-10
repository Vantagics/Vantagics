package main

import (
	"testing"
)

// TestExtractMarkdownTablesFromText tests the extraction of markdown tables from text
func TestExtractMarkdownTablesFromText(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCount  int
		expectedTitles []string
	}{
		{
			name: "Simple table without title",
			input: `Some text before

| Column1 | Column2 |
|---------|---------|
| Value1  | Value2  |
| Value3  | Value4  |

Some text after`,
			expectedCount:  1,
			expectedTitles: []string{""},
		},
		{
			name: "Table with markdown header title",
			input: `### 品类销售贡献分析

| 品类 | 销售额 | 占比 |
|------|--------|------|
| 饮料 | $111,968.18 | 27% |
| 乳制品 | $77,511.07 | 18% |`,
			expectedCount:  1,
			expectedTitles: []string{"品类销售贡献分析"},
		},
		{
			name: "Table with bold title",
			input: `**TOP3热销产品**

| 产品名称 | 品类 | 销售额 |
|----------|------|--------|
| Côte de Blaye | 饮料 | $67,324.25 |`,
			expectedCount:  1,
			expectedTitles: []string{"TOP3热销产品"},
		},
		{
			name: "Multiple tables with different title formats",
			input: `## 销售分析报告

### 品类销售贡献分析

| 品类 | 销售额 |
|------|--------|
| 饮料 | $111,968 |

**TOP3热销产品**

| 产品 | 销售额 |
|------|--------|
| 产品A | $67,324 |`,
			expectedCount:  2,
			expectedTitles: []string{"品类销售贡献分析", "TOP3热销产品"},
		},
		{
			name: "Table with numbered bold title",
			input: `1. **季度销售趋势**

| 季度 | 销售额 |
|------|--------|
| Q1 | $100,000 |`,
			expectedCount:  1,
			expectedTitles: []string{"季度销售趋势"},
		},
		{
			name: "No tables in text",
			input: `This is just some text without any tables.
It has multiple lines but no markdown tables.`,
			expectedCount:  0,
			expectedTitles: []string{},
		},
		{
			name: "Table with trailing space in separator",
			input: `### 📈 关键绩效指标对比

| 销售人员 | 总销售额 | 订单数 |
|---------|---------|--------| 
| Margaret | $232,891 | 156 |
| Janet | $202,813 | 127 |`,
			expectedCount:  1,
			expectedTitles: []string{"📈 关键绩效指标对比"},
		},
		{
			name: "Real world employee performance table",
			input: `### 📈 关键绩效指标对比

| 销售人员 | 总销售额 | 订单数 | 客户数 | 客户价值 | 复购率 | 平均订单额 |
|---------|---------|--------|--------|----------|--------|-----------| 
| **Margaret Peacock** | $232,891 | 156 | 75 | $3,105 | 2.08 | $1,493 |
| **Janet Leverling** | $202,813 | 127 | 63 | $3,219 | 2.02 | $1,597 |`,
			expectedCount:  1,
			expectedTitles: []string{"📈 关键绩效指标对比"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables := extractMarkdownTablesFromText(tt.input)

			if len(tables) != tt.expectedCount {
				t.Errorf("Expected %d tables, got %d", tt.expectedCount, len(tables))
				return
			}

			for i, table := range tables {
				if i < len(tt.expectedTitles) {
					if table.Title != tt.expectedTitles[i] {
						t.Errorf("Table %d: expected title %q, got %q", i, tt.expectedTitles[i], table.Title)
					}
				}
			}
		})
	}
}

// TestExtractTableTitle tests the extraction of table titles from preceding lines
func TestExtractTableTitle(t *testing.T) {
	tests := []struct {
		name          string
		lines         []string
		tableStartIdx int
		expectedTitle string
	}{
		{
			name: "Markdown header title",
			lines: []string{
				"### 品类销售贡献分析",
				"",
				"| 品类 | 销售额 |",
				"|------|--------|",
			},
			tableStartIdx: 2,
			expectedTitle: "品类销售贡献分析",
		},
		{
			name: "Bold title",
			lines: []string{
				"**TOP3热销产品**",
				"",
				"| 产品 | 销售额 |",
				"|------|--------|",
			},
			tableStartIdx: 2,
			expectedTitle: "TOP3热销产品",
		},
		{
			name: "Bold title with description",
			lines: []string{
				"**销售分析**：按品类统计",
				"",
				"| 品类 | 销售额 |",
				"|------|--------|",
			},
			tableStartIdx: 2,
			expectedTitle: "销售分析",
		},
		{
			name: "No title",
			lines: []string{
				"| 品类 | 销售额 |",
				"|------|--------|",
			},
			tableStartIdx: 0,
			expectedTitle: "",
		},
		{
			name: "Numbered bold title",
			lines: []string{
				"1. **季度销售趋势**",
				"",
				"| 季度 | 销售额 |",
				"|------|--------|",
			},
			tableStartIdx: 2,
			expectedTitle: "季度销售趋势",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := extractTableTitle(tt.lines, tt.tableStartIdx)
			if title != tt.expectedTitle {
				t.Errorf("Expected title %q, got %q", tt.expectedTitle, title)
			}
		})
	}
}
