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

	// Add classification hints if available
	if hints != nil {
		sb.WriteString("## 分析要求（基于请求理解）\n")
		if hints.NeedsVisualization {
			sb.WriteString("- ⭐ **必须生成可视化图表** (chart.png)\n")
		}
		if hints.NeedsDataExport {
			sb.WriteString("- ⭐ **需要导出数据文件** (Excel/CSV)\n")
		}
		if len(hints.SuggestedOutputs) > 0 {
			sb.WriteString(fmt.Sprintf("- 建议输出: %s\n", strings.Join(hints.SuggestedOutputs, ", ")))
		}
		if hints.Reasoning != "" {
			sb.WriteString(fmt.Sprintf("- 分析原因: %s\n", hints.Reasoning))
		}
		sb.WriteString("\n")
	}

	// Database info section
	sb.WriteString("## 数据库信息\n")
	sb.WriteString(fmt.Sprintf("- 数据库类型: %s\n", schemaContext.DatabaseType))
	sb.WriteString("- 数据库路径: {DB_PATH} (运行时注入)\n")
	sb.WriteString("- 会话目录: {SESSION_DIR} (运行时注入)\n\n")

	// Schema section
	sb.WriteString("## 数据库Schema\n")
	sb.WriteString(b.formatSchemaForPrompt(schemaContext))
	sb.WriteString("\n")

	// Code requirements section
	sb.WriteString("## 代码要求\n")
	sb.WriteString("1. 代码必须完整可执行，不需要任何修改\n")
	sb.WriteString("2. 使用sqlite3连接数据库，pandas处理数据\n")
	
	// Visualization requirement based on hints
	if hints != nil && hints.NeedsVisualization {
		sb.WriteString("3. **必须生成可视化图表**，使用matplotlib/seaborn，保存为chart.png到SESSION_DIR\n")
	} else {
		sb.WriteString("3. **必须生成可视化图表**，使用matplotlib/seaborn，保存为chart.png到SESSION_DIR\n")
	}
	
	sb.WriteString("4. 所有输出使用中文（图表标题、标签、洞察）\n")
	sb.WriteString("5. 包含完整的错误处理（try-except-finally）\n")
	sb.WriteString("6. 在finally块中关闭数据库连接\n")
	sb.WriteString("7. 使用print输出分析结果和关键洞察\n")
	sb.WriteString("8. 数据库路径使用变量DB_PATH，会话目录使用变量SESSION_DIR\n")
	sb.WriteString("9. 图表配置: plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei']\n")
	
	// Data export requirement based on hints
	if hints != nil && hints.NeedsDataExport {
		sb.WriteString("10. **导出数据到Excel文件**: 使用df.to_excel()保存到SESSION_DIR\n")
	}
	sb.WriteString("\n")

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

# 数据库路径和会话目录（运行时注入）
DB_PATH = "{DB_PATH}"
SESSION_DIR = "{SESSION_DIR}"

def main():
    conn = None
    try:
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

# 数据库路径和会话目录（运行时注入）
DB_PATH = "{DB_PATH}"
SESSION_DIR = "{SESSION_DIR}"

def main():
    conn = None
    try:
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
        
        # 4. 创建可视化
        plt.figure(figsize=(10, 6))
        # ... 绑定数据到图表 ...
        plt.title('图表标题', fontsize=14)
        plt.xlabel('X轴标签')
        plt.ylabel('Y轴标签')
        plt.tight_layout()
        
        # 5. 保存图表
        chart_path = os.path.join(SESSION_DIR, 'chart.png')
        plt.savefig(chart_path, dpi=150, bbox_inches='tight')
        plt.close()
        
        # 6. 输出结果
        print("=== 分析结果 ===")
        print(df.to_string())
        print(f"\n图表已保存: {chart_path}")
        
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

const aggregationCodeTemplate = `import sqlite3
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib
matplotlib.use('Agg')
plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei', 'DejaVu Sans']
plt.rcParams['axes.unicode_minus'] = False
import os

# 数据库路径和会话目录（运行时注入）
DB_PATH = "{DB_PATH}"
SESSION_DIR = "{SESSION_DIR}"

def main():
    conn = None
    try:
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
plt.savefig(os.path.join(SESSION_DIR, 'chart.png'), dpi=150)
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
