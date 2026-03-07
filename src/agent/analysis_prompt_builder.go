package agent

import (
	"fmt"
	"strings"
)

// AnalysisPromptBuilder builds prompts for unified code generation
type AnalysisPromptBuilder struct {
	templates map[string]*CodeTemplate
}

// CodeTemplate represents a code template for specific analysis types
type CodeTemplate struct {
	Name        string
	Description string
	Structure   string
	Examples    []string
}

// NewAnalysisPromptBuilder creates a new prompt builder with default templates
func NewAnalysisPromptBuilder() *AnalysisPromptBuilder {
	builder := &AnalysisPromptBuilder{
		templates: make(map[string]*CodeTemplate),
	}
	builder.initDefaultTemplates()
	return builder
}

// initDefaultTemplates initializes the default code templates
func (b *AnalysisPromptBuilder) initDefaultTemplates() {
	// Standard analysis template
	b.templates["standard"] = &CodeTemplate{
		Name:        "standard",
		Description: "标准数据分析模板",
		Structure:   standardCodeTemplate,
		Examples:    []string{standardExample},
	}

	// Visualization template
	b.templates["visualization"] = &CodeTemplate{
		Name:        "visualization",
		Description: "数据可视化分析模�",
		Structure:   visualizationCodeTemplate,
		Examples:    []string{visualizationExample},
	}

	// Aggregation template
	b.templates["aggregation"] = &CodeTemplate{
		Name:        "aggregation",
		Description: "数据聚合分析模板",
		Structure:   aggregationCodeTemplate,
		Examples:    []string{aggregationExample},
	}
}

// BuildPrompt constructs the complete prompt for code generation
func (b *AnalysisPromptBuilder) BuildPrompt(userRequest string, schemaContext *UnifiedSchemaContext, outputFormat string) string {
	return b.BuildPromptWithHints(userRequest, schemaContext, outputFormat, nil)
}

// BuildPromptWithHints constructs the prompt with LLM classification hints.
// Uses a single English prompt with a "match user's language" instruction,
// so the LLM naturally adapts chart labels and print output to the user's language.
func (b *AnalysisPromptBuilder) BuildPromptWithHints(userRequest string, schemaContext *UnifiedSchemaContext, outputFormat string, hints *ClassificationResult) string {
	var sb strings.Builder

	sb.WriteString("You are a data analysis expert. Generate complete Python analysis code based on the user request.\n\n")

	sb.WriteString("## User Request\n")
	sb.WriteString(userRequest)
	sb.WriteString("\n\n")

	// Add classification hints if available
	if hints != nil {
		sb.WriteString("## Analysis Requirements\n")
		if hints.NeedsVisualization {
			sb.WriteString("- ⭐⭐�**[MUST] Generate visualization chart** - core requirement!\n")
			sb.WriteString("  - Use ECharts only\n")
			sb.WriteString("  - Print a valid json:echarts block directly in stdout\n")
			sb.WriteString("  - Do NOT save chart images such as chart.png\n")
			if hints.SuggestedChartType != "" {
				chartTypeDesc := map[string]string{
					"line":        "Line chart - for trends",
					"bar":         "Bar chart - for category comparison",
					"pie":         "Pie chart - for proportions",
					"grouped_bar": "Grouped bar chart - for multi-dimensional comparison",
					"scatter":     "Scatter plot - for correlation analysis",
					"heatmap":     "Heatmap - for matrix data",
				}
				if desc, ok := chartTypeDesc[hints.SuggestedChartType]; ok {
					sb.WriteString(fmt.Sprintf("  - Recommended chart type: %s\n", desc))
				}
			}
		}
		if hints.NeedsDataExport {
			sb.WriteString("- �**Must export data file** - use df.to_excel() to save to FILES_DIR\n")
		}
		if len(hints.SuggestedOutputs) > 0 {
			sb.WriteString(fmt.Sprintf("- Suggested outputs: %s\n", strings.Join(hints.SuggestedOutputs, ", ")))
		}
		if hints.Reasoning != "" {
			sb.WriteString(fmt.Sprintf("- Reasoning: %s\n", hints.Reasoning))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("## Analysis Requirements\n")
		sb.WriteString("- �**Recommend generating visualization charts**\n")
		sb.WriteString("- Use ECharts only and print a json:echarts block\n\n")
	}

	// Database info section
	sb.WriteString("## Database Info\n")
	sb.WriteString(fmt.Sprintf("- Database type: %s\n", schemaContext.DatabaseType))
	sb.WriteString("- Database path: {DB_PATH} (injected at runtime)\n")
	sb.WriteString("- File save directory: {FILES_DIR} (injected at runtime)\n\n")

	// Schema section
	sb.WriteString("## Database Schema\n")
	sb.WriteString(b.formatSchemaForPrompt(schemaContext))
	sb.WriteString("\n")

	// Code requirements section
	sb.WriteString("## Code Requirements (strict)\n")
	sb.WriteString("1. Code must be complete and executable without modifications\n")
	sb.WriteString("2. Use duckdb for database, pandas for data processing\n")
	sb.WriteString("3. **Visualization must use ECharts only**:\n")
	sb.WriteString("   ```python\n")
	sb.WriteString("   import json\n")
	sb.WriteString("   print('```json:echarts')\n")
	sb.WriteString("   print(json.dumps(echarts_option, ensure_ascii=False))\n")
	sb.WriteString("   print('```')\n")
	sb.WriteString("   ```\n")
	sb.WriteString("   - Do NOT use matplotlib/seaborn and do NOT save chart images\n")
	sb.WriteString("4. **LANGUAGE**: All user-facing output (chart titles, labels, print statements, insights) MUST be in the SAME language as the user's request above\n")
	sb.WriteString("5. Include complete error handling (try-except-finally)\n")
	sb.WriteString("6. Close database connection in finally block\n")
	sb.WriteString("7. Use print to output analysis results and key insights\n")
	sb.WriteString("8. Use DB_PATH for database path, FILES_DIR for file save directory\n")
	sb.WriteString("9. ECharts options must be valid JSON, no JavaScript functions\n")
	sb.WriteString("10. **Chart styling**: Use appropriate colors, titles, labels\n")

	// Data export requirement
	if hints != nil && hints.NeedsDataExport {
		sb.WriteString("11. **Data export**:\n")
		sb.WriteString("    ```python\n")
		sb.WriteString("    export_path = os.path.join(FILES_DIR, 'analysis_data.xlsx')\n")
		sb.WriteString("    df.to_excel(export_path, index=False)\n")
		sb.WriteString("    ```\n")
	}
	sb.WriteString("\n")

	// File saving warning
	sb.WriteString("## ⚠️ File Saving\n")
	sb.WriteString("**Must use FILES_DIR variable for all files. No other paths!**\n\n")

	// Output format
	sb.WriteString("## Output Format\n")
	sb.WriteString("Output only Python code, no explanations. Wrap in ```python and ```.\n\n")

	// Template section
	template := b.GetTemplate(outputFormat)
	if template == nil || outputFormat == "standard" {
		template = b.templates["visualization"]
	}
	if template != nil {
		sb.WriteString("## Code Structure Reference\n")
		sb.WriteString("```python\n")
		sb.WriteString(template.Structure)
		sb.WriteString("\n```\n")
	}

	return sb.String()
}


// formatSchemaForPrompt formats schema context for the prompt
func (b *AnalysisPromptBuilder) formatSchemaForPrompt(ctx *UnifiedSchemaContext) string {
	var sb strings.Builder

	for _, table := range ctx.Tables {
		sb.WriteString(fmt.Sprintf("### Table: %s", table.Name))
		if table.RowCount > 0 {
			sb.WriteString(fmt.Sprintf(" (~%d rows)", table.RowCount))
		}
		sb.WriteString("\n")

		sb.WriteString("Columns:\n")
		for _, col := range table.Columns {
			sb.WriteString(fmt.Sprintf("- %s (%s)", col.Name, col.Type))
			if col.IsPK {
				sb.WriteString(" [PK]")
			}
			if col.IsFK && col.RefTable != "" {
				sb.WriteString(fmt.Sprintf(" [FK->%s]", col.RefTable))
			}
			sb.WriteString("\n")
		}

		if len(table.SampleData) > 0 {
			sb.WriteString("\nSample data:\n")
			for i, row := range table.SampleData {
				if i >= 2 { // Limit to 2 rows
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
		sb.WriteString("### Relationships\n")
		for _, rel := range ctx.Relationships {
			sb.WriteString(fmt.Sprintf("- %s.%s -> %s.%s\n", rel.FromTable, rel.FromColumn, rel.ToTable, rel.ToColumn))
		}
	}

	return sb.String()
}

// GetTemplate returns the appropriate template for the analysis type
func (b *AnalysisPromptBuilder) GetTemplate(analysisType string) *CodeTemplate {
	if template, ok := b.templates[analysisType]; ok {
		return template
	}
	return b.templates["standard"]
}

// AddTemplate adds a custom template
func (b *AnalysisPromptBuilder) AddTemplate(name string, template *CodeTemplate) {
	b.templates[name] = template
}

// Code templates
const standardCodeTemplate = `import duckdb
import pandas as pd
import os

# 数据库路径和文件保存目录（运行时注入�
DB_PATH = "{DB_PATH}"
FILES_DIR = "{FILES_DIR}"

def main():
    conn = None
    try:
        # 确保文件目录存在
        os.makedirs(FILES_DIR, exist_ok=True)
        
        # 1. 连接数据�
        conn = duckdb.connect(DB_PATH, read_only=True)
        
        # 2. 执行SQL查询
        sql = """
        SELECT ...
        FROM ...
        WHERE ...
        """
        # DuckDB directly supports pandas
        df = conn.execute(sql).df()
        
        # 3. 数据处理
        # ... 数据清洗、转换、计�...
        
        # 4. 输出结果
        print("=== 分析结果 ===")
        print(df.to_string())
        
    except Exception as e:
        print(f"分析错误: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    main()
`

const visualizationCodeTemplate = `import duckdb
import pandas as pd
import json
import os

# 数据库路径和文件保存目录（运行时注入�
DB_PATH = "{DB_PATH}"
FILES_DIR = "{FILES_DIR}"

def main():
    conn = None
    try:
        # 确保文件目录存在
        os.makedirs(FILES_DIR, exist_ok=True)
        
        # 1. 连接数据�
        conn = duckdb.connect(DB_PATH, read_only=True)
        
        # 2. 执行SQL查询
        sql = """
        SELECT ...
        FROM ...
        GROUP BY ...
        ORDER BY ...
        """
        # DuckDB directly supports pandas
        df = conn.execute(sql).df()
        
        # 3. 数据处理
        # ... 数据清洗、转换、计�...
        
        # 4. 【必须】创建可视化图表
        echarts_option = {
            "title": {"text": "图表标题"},
            "tooltip": {"trigger": "axis"},
            "xAxis": {"type": "category", "data": df.iloc[:, 0].astype(str).tolist()},
            "yAxis": {"type": "value"},
            "series": [{
                "type": "bar",
                "data": df.iloc[:, 1].tolist()
            }]
        }
        
        # 选择合适的图表类型�
        # - 时间趋势: plt.plot() 折线�
        # - 分类对比: plt.bar() 柱状�
        # - 占比分布: plt.pie() 饼图
        # - 多维对比: 分组柱状�
        
        # 示例：柱状图
        # 例如：根据数据选择 bar / line / pie 等 ECharts series 类型
        
        # 示例：折线图
        # 例如：series = [{"type": "line", "data": ...}]
        
        # 示例：饼�
        # 例如：series = [{"type": "pie", "data": ...}]
        
        # 图表美化
        # 可继续补充 title / legend / axisLabel / color 等 ECharts 配置
        print("json:echarts")
        print(json.dumps(echarts_option, ensure_ascii=False))
        
        # 5. 【必须】保存图表到FILES_DIR
        # 不要保存 chart.png，直接输出 json:echarts
        print(f"�图表已保�" {chart_path}")
        
        # 6. 【可选】导出数据到Excel
        # export_path = os.path.join(FILES_DIR, 'data_export.xlsx')
        # df.to_excel(export_path, index=False, sheet_name='分析数据')
        # print(f"�数据已导�" {export_path}")
        
        # 7. 输出分析结果和洞�
        print("\\n=== 分析结果 ===")
        print(df.to_string(index=False))
        
        print("\\n=== 关键洞察 ===")
        # 输出数据洞察，例如：
        # print(f"- 最高�" {df['value'].max()}")
        # print(f"- 最低�" {df['value'].min()}")
        # print(f"- 平均�" {df['value'].mean():.2f}")
        
    except Exception as e:
        print(f"分析错误: {e}")
        import traceback
        traceback.print_exc()
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    main()
`

const aggregationCodeTemplate = `import duckdb
import pandas as pd
import os

# 数据库路径和文件保存目录（运行时注入�
DB_PATH = "{DB_PATH}"
FILES_DIR = "{FILES_DIR}"

def main():
    conn = None
    try:
        # 确保文件目录存在
        os.makedirs(FILES_DIR, exist_ok=True)
        
        # 1. 连接数据�
        conn = duckdb.connect(DB_PATH, read_only=True)
        
        # 2. 执行聚合查询
        sql = """
        SELECT 
            dimension_column,
            COUNT(*) as count,
            SUM(value_column) as total,
            AVG(value_column) as average
        FROM table_name
        GROUP BY dimension_column
        ORDER BY total DESC
        """
        # DuckDB directly supports pandas
        df = conn.execute(sql).df()
        
        # 3. 计算汇总统�
        total = df['total'].sum()
        avg = df['average'].mean()
        
        # 4. 输出结果
        print("=== 聚合分析结果 ===")
        print(f"总计: {total:,.2f}")
        print(f"平均: {avg:,.2f}")
        print("\n详细数据:")
        print(df.to_string())
        
    except Exception as e:
        print(f"分析错误: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    main()
`

// Example code snippets
const standardExample = `# 示例：查询销售数�
sql = """
SELECT product_name, SUM(quantity) as total_qty, SUM(amount) as total_amount
FROM orders
WHERE order_date >= '2024-01-01'
GROUP BY product_name
ORDER BY total_amount DESC
LIMIT 10
"""
df = conn.execute(sql).df()
print("=== 销售排行榜 ===")
print(df.to_string(index=False))
`

const visualizationExample = `# 示例：销售趋势图
sql = """
SELECT strftime('%Y-%m', order_date) as month, SUM(amount) as total
FROM orders
GROUP BY month
ORDER BY month
"""
df = conn.execute(sql).df()

plt.figure(figsize=(12, 6))
plt.plot(df['month'], df['total'], marker='o', linewidth=2)
plt.title('月度销售趋�, fontsize=14)
plt.xlabel('月份')
plt.ylabel('销售额')
plt.xticks(rotation=45)
plt.grid(True, alpha=0.3)
plt.tight_layout()
plt.savefig(os.path.join(FILES_DIR, 'chart.png'), dpi=150)
print(f"�图表已保�" {os.path.join(FILES_DIR, 'chart.png')}")
`

const excelExportExample = `# 示例：导出数据到Excel
sql = """
SELECT customer_name, order_date, product_name, quantity, amount
FROM orders o
JOIN customers c ON o.customer_id = c.id
JOIN products p ON o.product_id = p.id
ORDER BY order_date DESC
"""
df = conn.execute(sql).df()

# 保存到Excel文件
export_path = os.path.join(FILES_DIR, 'order_details.xlsx')
df.to_excel(export_path, index=False, sheet_name='订单明细')
print(f"�数据已导出到Excel: {export_path}")
print(f"共导�{len(df)} 条记�")
`

const aggregationExample = `# 示例：客户分�
sql = """
SELECT 
    customer_id,
    COUNT(*) as order_count,
    SUM(amount) as total_spent,
    AVG(amount) as avg_order
FROM orders
GROUP BY customer_id
HAVING order_count >= 3
ORDER BY total_spent DESC
"""
df = conn.execute(sql).df()
print(f"活跃客户�" {len(df)}")
print(f"总消�" {df['total_spent'].sum():,.2f}")
`
