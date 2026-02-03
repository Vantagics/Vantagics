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
		Description: "数据可视化分析模板",
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

// BuildPromptWithHints constructs the prompt with LLM classification hints
func (b *AnalysisPromptBuilder) BuildPromptWithHints(userRequest string, schemaContext *UnifiedSchemaContext, outputFormat string, hints *ClassificationResult) string {
	var sb strings.Builder

	sb.WriteString("你是一个数据分析专家。请根据用户请求生成完整的Python分析代码。\n\n")

	// User request section
	sb.WriteString("## 用户请求\n")
	sb.WriteString(userRequest)
	sb.WriteString("\n\n")

	// Add classification hints if available - with stronger emphasis on visualization
	if hints != nil {
		sb.WriteString("## 分析要求（基于请求理解）\n")
		if hints.NeedsVisualization {
			sb.WriteString("- ⭐⭐⭐ **【必须】生成可视化图表** - 这是核心要求！\n")
			sb.WriteString("  - 使用 matplotlib/seaborn 创建图表\n")
			sb.WriteString("  - 必须调用 plt.savefig() 保存图表到 FILES_DIR\n")
			sb.WriteString("  - 图表文件名: chart.png\n")
			if hints.SuggestedChartType != "" {
				chartTypeDesc := map[string]string{
					"line":        "折线图 (plt.plot) - 适合展示趋势变化",
					"bar":         "柱状图 (plt.bar) - 适合分类对比",
					"pie":         "饼图 (plt.pie) - 适合展示占比分布",
					"grouped_bar": "分组柱状图 - 适合多维度对比",
					"scatter":     "散点图 (plt.scatter) - 适合相关性分析",
					"heatmap":     "热力图 (sns.heatmap) - 适合矩阵数据",
				}
				if desc, ok := chartTypeDesc[hints.SuggestedChartType]; ok {
					sb.WriteString(fmt.Sprintf("  - 推荐图表类型: %s\n", desc))
				}
			}
		}
		if hints.NeedsDataExport {
			sb.WriteString("- ⭐ **必须导出数据文件** - 使用df.to_excel()保存到SESSION_DIR\n")
		}
		if len(hints.SuggestedOutputs) > 0 {
			sb.WriteString(fmt.Sprintf("- 建议输出: %s\n", strings.Join(hints.SuggestedOutputs, ", ")))
		}
		if hints.Reasoning != "" {
			sb.WriteString(fmt.Sprintf("- 分析原因: %s\n", hints.Reasoning))
		}
		sb.WriteString("\n")
	} else {
		// Even without hints, encourage visualization for analysis requests
		sb.WriteString("## 分析要求\n")
		sb.WriteString("- ⭐ **建议生成可视化图表** - 图表能更直观地展示分析结果\n")
		sb.WriteString("- 使用 plt.savefig() 保存图表到 FILES_DIR/chart.png\n\n")
	}

	// Database info section
	sb.WriteString("## 数据库信息\n")
	sb.WriteString(fmt.Sprintf("- 数据库类型: %s\n", schemaContext.DatabaseType))
	sb.WriteString("- 数据库路径: {DB_PATH} (运行时注入)\n")
	sb.WriteString("- 文件保存目录: {FILES_DIR} (运行时注入，所有生成的文件必须保存到此目录)\n\n")

	// Schema section
	sb.WriteString("## 数据库Schema\n")
	sb.WriteString(b.formatSchemaForPrompt(schemaContext))
	sb.WriteString("\n")

	// Code requirements section - 更强调文件保存
	sb.WriteString("## 代码要求（必须严格遵守）\n")
	sb.WriteString("1. 代码必须完整可执行，不需要任何修改\n")
	sb.WriteString("2. 使用sqlite3连接数据库，pandas处理数据\n")
	sb.WriteString("3. **⭐⭐⭐ 图表必须实际保存**: \n")
	sb.WriteString("   ```python\n")
	sb.WriteString("   # 必须包含以下代码来保存图表\n")
	sb.WriteString("   chart_path = os.path.join(FILES_DIR, 'chart.png')\n")
	sb.WriteString("   plt.savefig(chart_path, dpi=150, bbox_inches='tight', facecolor='white')\n")
	sb.WriteString("   plt.close()\n")
	sb.WriteString("   print(f'✅ 图表已保存: {chart_path}')\n")
	sb.WriteString("   ```\n")
	sb.WriteString("4. 所有输出使用中文（图表标题、标签、洞察）\n")
	sb.WriteString("5. 包含完整的错误处理（try-except-finally）\n")
	sb.WriteString("6. 在finally块中关闭数据库连接\n")
	sb.WriteString("7. 使用print输出分析结果和关键洞察\n")
	sb.WriteString("8. 数据库路径使用变量DB_PATH，文件保存目录使用变量FILES_DIR\n")
	sb.WriteString("9. 图表配置: plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei']\n")
	sb.WriteString("10. **图表美化**: 使用合适的颜色、标题、标签，确保图表清晰易读\n")
	
	// Data export requirement based on hints
	if hints != nil && hints.NeedsDataExport {
		sb.WriteString("10. **数据必须实际导出**: 使用以下代码保存Excel:\n")
		sb.WriteString("    ```python\n")
		sb.WriteString("    export_path = os.path.join(FILES_DIR, 'analysis_data.xlsx')\n")
		sb.WriteString("    df.to_excel(export_path, index=False, sheet_name='数据')\n")
		sb.WriteString("    print(f'✅ 数据已导出: {export_path}')\n")
		sb.WriteString("    ```\n")
	}
	sb.WriteString("\n")

	// Critical warning about file generation
	sb.WriteString("## ⚠️ 重要警告 - 文件保存\n")
	sb.WriteString("**必须使用 FILES_DIR 变量保存所有文件，不要使用其他路径！**\n\n")
	sb.WriteString("正确示例:\n")
	sb.WriteString("```python\n")
	sb.WriteString("# 在代码开头定义（会被自动替换为实际路径）\n")
	sb.WriteString("FILES_DIR = \"{FILES_DIR}\"\n")
	sb.WriteString("os.makedirs(FILES_DIR, exist_ok=True)  # 确保目录存在\n\n")
	sb.WriteString("# 保存图表\n")
	sb.WriteString("chart_path = os.path.join(FILES_DIR, 'chart.png')\n")
	sb.WriteString("plt.savefig(chart_path, dpi=150, bbox_inches='tight')\n")
	sb.WriteString("print(f'✅ 图表已保存: {chart_path}')\n\n")
	sb.WriteString("# 保存Excel\n")
	sb.WriteString("excel_path = os.path.join(FILES_DIR, 'data.xlsx')\n")
	sb.WriteString("df.to_excel(excel_path, index=False)\n")
	sb.WriteString("print(f'✅ Excel已保存: {excel_path}')\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**错误示例（不要这样做）:**\n")
	sb.WriteString("- ❌ `plt.savefig('chart.png')` - 没有使用 FILES_DIR\n")
	sb.WriteString("- ❌ `plt.savefig('/tmp/chart.png')` - 使用了硬编码路径\n")
	sb.WriteString("- ❌ `df.to_excel('data.xlsx')` - 没有使用 FILES_DIR\n\n")

	// Output format section
	sb.WriteString("## 输出格式\n")
	sb.WriteString("只输出Python代码，不要其他解释。代码用```python和```包裹。\n\n")

	// Template section - always use visualization template for analysis
	template := b.GetTemplate(outputFormat)
	if template == nil || outputFormat == "standard" {
		// Default to visualization for analysis requests
		template = b.templates["visualization"]
	}
	if template != nil {
		sb.WriteString("## 代码结构参考\n")
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
		sb.WriteString(fmt.Sprintf("### 表: %s", table.Name))
		if table.RowCount > 0 {
			sb.WriteString(fmt.Sprintf(" (约 %d 行)", table.RowCount))
		}
		sb.WriteString("\n")

		sb.WriteString("字段:\n")
		for _, col := range table.Columns {
			sb.WriteString(fmt.Sprintf("- %s (%s)", col.Name, col.Type))
			if col.IsPK {
				sb.WriteString(" [主键]")
			}
			if col.IsFK && col.RefTable != "" {
				sb.WriteString(fmt.Sprintf(" [外键->%s]", col.RefTable))
			}
			sb.WriteString("\n")
		}

		if len(table.SampleData) > 0 {
			sb.WriteString("\n示例数据:\n")
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
		sb.WriteString("### 表关系\n")
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
const standardCodeTemplate = `import sqlite3
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib
matplotlib.use('Agg')
plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei', 'DejaVu Sans']
plt.rcParams['axes.unicode_minus'] = False
import os

# 数据库路径和文件保存目录（运行时注入）
DB_PATH = "{DB_PATH}"
FILES_DIR = "{FILES_DIR}"

def main():
    conn = None
    try:
        # 确保文件目录存在
        os.makedirs(FILES_DIR, exist_ok=True)
        
        # 1. 连接数据库
        conn = sqlite3.connect(DB_PATH)
        
        # 2. 执行SQL查询
        sql = """
        SELECT ...
        FROM ...
        WHERE ...
        """
        df = pd.read_sql_query(sql, conn)
        
        # 3. 数据处理
        # ... 数据清洗、转换、计算 ...
        
        # 4. 输出结果
        print("=== 分析结果 ===")
        print(df.to_string())
        
    except sqlite3.Error as e:
        print(f"数据库错误: {e}")
    except pd.errors.EmptyDataError:
        print("查询结果为空")
    except Exception as e:
        print(f"分析错误: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    main()
`

const visualizationCodeTemplate = `import sqlite3
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib
matplotlib.use('Agg')
plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei', 'DejaVu Sans']
plt.rcParams['axes.unicode_minus'] = False
import seaborn as sns
import os

# 数据库路径和文件保存目录（运行时注入）
DB_PATH = "{DB_PATH}"
FILES_DIR = "{FILES_DIR}"

def main():
    conn = None
    try:
        # 确保文件目录存在
        os.makedirs(FILES_DIR, exist_ok=True)
        
        # 1. 连接数据库
        conn = sqlite3.connect(DB_PATH)
        
        # 2. 执行SQL查询
        sql = """
        SELECT ...
        FROM ...
        GROUP BY ...
        ORDER BY ...
        """
        df = pd.read_sql_query(sql, conn)
        
        # 3. 数据处理
        # ... 数据清洗、转换、计算 ...
        
        # 4. 【必须】创建可视化图表
        fig, ax = plt.subplots(figsize=(10, 6))
        
        # 选择合适的图表类型：
        # - 时间趋势: plt.plot() 折线图
        # - 分类对比: plt.bar() 柱状图
        # - 占比分布: plt.pie() 饼图
        # - 多维对比: 分组柱状图
        
        # 示例：柱状图
        # ax.bar(df['category'], df['value'], color='steelblue')
        
        # 示例：折线图
        # ax.plot(df['date'], df['value'], marker='o', linewidth=2, color='steelblue')
        
        # 示例：饼图
        # ax.pie(df['value'], labels=df['category'], autopct='%1.1f%%')
        
        # 图表美化
        ax.set_title('图表标题', fontsize=14, fontweight='bold')
        ax.set_xlabel('X轴标签', fontsize=12)
        ax.set_ylabel('Y轴标签', fontsize=12)
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        
        # 5. 【必须】保存图表到FILES_DIR
        chart_path = os.path.join(FILES_DIR, 'chart.png')
        plt.savefig(chart_path, dpi=150, bbox_inches='tight', facecolor='white')
        plt.close()
        print(f"✅ 图表已保存: {chart_path}")
        
        # 6. 【可选】导出数据到Excel
        # export_path = os.path.join(FILES_DIR, 'data_export.xlsx')
        # df.to_excel(export_path, index=False, sheet_name='分析数据')
        # print(f"✅ 数据已导出: {export_path}")
        
        # 7. 输出分析结果和洞察
        print("\\n=== 分析结果 ===")
        print(df.to_string(index=False))
        
        print("\\n=== 关键洞察 ===")
        # 输出数据洞察，例如：
        # print(f"- 最高值: {df['value'].max()}")
        # print(f"- 最低值: {df['value'].min()}")
        # print(f"- 平均值: {df['value'].mean():.2f}")
        
    except sqlite3.Error as e:
        print(f"数据库错误: {e}")
    except pd.errors.EmptyDataError:
        print("查询结果为空")
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

const aggregationCodeTemplate = `import sqlite3
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib
matplotlib.use('Agg')
plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei', 'DejaVu Sans']
plt.rcParams['axes.unicode_minus'] = False
import os

# 数据库路径和文件保存目录（运行时注入）
DB_PATH = "{DB_PATH}"
FILES_DIR = "{FILES_DIR}"

def main():
    conn = None
    try:
        # 确保文件目录存在
        os.makedirs(FILES_DIR, exist_ok=True)
        
        # 1. 连接数据库
        conn = sqlite3.connect(DB_PATH)
        
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
        df = pd.read_sql_query(sql, conn)
        
        # 3. 计算汇总统计
        total = df['total'].sum()
        avg = df['average'].mean()
        
        # 4. 输出结果
        print("=== 聚合分析结果 ===")
        print(f"总计: {total:,.2f}")
        print(f"平均: {avg:,.2f}")
        print("\n详细数据:")
        print(df.to_string())
        
    except sqlite3.Error as e:
        print(f"数据库错误: {e}")
    except pd.errors.EmptyDataError:
        print("查询结果为空")
    except Exception as e:
        print(f"分析错误: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    main()
`

// Example code snippets
const standardExample = `# 示例：查询销售数据
sql = """
SELECT product_name, SUM(quantity) as total_qty, SUM(amount) as total_amount
FROM orders
WHERE order_date >= '2024-01-01'
GROUP BY product_name
ORDER BY total_amount DESC
LIMIT 10
"""
df = pd.read_sql_query(sql, conn)
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
df = pd.read_sql_query(sql, conn)

plt.figure(figsize=(12, 6))
plt.plot(df['month'], df['total'], marker='o', linewidth=2)
plt.title('月度销售趋势', fontsize=14)
plt.xlabel('月份')
plt.ylabel('销售额')
plt.xticks(rotation=45)
plt.grid(True, alpha=0.3)
plt.tight_layout()
plt.savefig(os.path.join(FILES_DIR, 'chart.png'), dpi=150)
print(f"✅ 图表已保存: {os.path.join(FILES_DIR, 'chart.png')}")
`

const excelExportExample = `# 示例：导出数据到Excel
sql = """
SELECT customer_name, order_date, product_name, quantity, amount
FROM orders o
JOIN customers c ON o.customer_id = c.id
JOIN products p ON o.product_id = p.id
ORDER BY order_date DESC
"""
df = pd.read_sql_query(sql, conn)

# 保存到Excel文件
export_path = os.path.join(FILES_DIR, 'order_details.xlsx')
df.to_excel(export_path, index=False, sheet_name='订单明细')
print(f"✅ 数据已导出到Excel: {export_path}")
print(f"共导出 {len(df)} 条记录")
`

const aggregationExample = `# 示例：客户分析
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
df = pd.read_sql_query(sql, conn)
print(f"活跃客户数: {len(df)}")
print(f"总消费: {df['total_spent'].sum():,.2f}")
`
