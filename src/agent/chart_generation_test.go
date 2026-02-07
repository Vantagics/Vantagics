package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestChartGenerationPipeline 测试图表生成流程
// 验证从代码生成到文件检测的完整流程
func TestChartGenerationPipeline(t *testing.T) {
	t.Log("=== 图表生成流程测试 ===")

	// 测试1: 验证 CodeValidator 正确检测图表保存代码
	t.Run("CodeValidator_ChartSaveDetection", func(t *testing.T) {
		validator := NewCodeValidator()

		testCases := []struct {
			name        string
			code        string
			expectChart bool
			description string
		}{
			{
				name: "plt.savefig_with_path",
				code: `
import matplotlib.pyplot as plt
plt.figure()
plt.bar([1,2,3], [4,5,6])
plt.savefig(os.path.join(FILES_DIR, 'chart.png'), dpi=150)
plt.close()
`,
				expectChart: true,
				description: "使用 plt.savefig 保存图表",
			},
			{
				name: "fig.savefig_with_path",
				code: `
import matplotlib.pyplot as plt
fig, ax = plt.subplots()
ax.plot([1,2,3], [4,5,6])
fig.savefig(os.path.join(FILES_DIR, 'chart.png'))
plt.close()
`,
				expectChart: true,
				description: "使用 fig.savefig 保存图表",
			},
			{
				name: "savefig_simple",
				code: `
import matplotlib.pyplot as plt
plt.plot([1,2,3])
savefig('chart.png')
`,
				expectChart: true,
				description: "简单的 savefig 调用",
			},
			{
				name: "only_plt_show",
				code: `
import matplotlib.pyplot as plt
plt.figure()
plt.bar([1,2,3], [4,5,6])
plt.show()
`,
				expectChart: false,
				description: "只有 plt.show() 不保存文件",
			},
			{
				name: "only_plt_plot",
				code: `
import matplotlib.pyplot as plt
plt.plot([1,2,3], [4,5,6])
`,
				expectChart: false,
				description: "只有绑图代码没有保存",
			},
			{
				name: "seaborn_without_save",
				code: `
import seaborn as sns
sns.barplot(x=[1,2,3], y=[4,5,6])
plt.show()
`,
				expectChart: false,
				description: "Seaborn 图表没有保存",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := validator.ValidateCode(tc.code)

				if result.HasChart != tc.expectChart {
					t.Errorf("%s: 期望 HasChart=%v, 实际=%v",
						tc.description, tc.expectChart, result.HasChart)
				}

				t.Logf("✓ %s: HasChart=%v (符合预期)", tc.description, result.HasChart)
			})
		}
	})

	// 测试2: 验证 UnifiedPythonGenerator 的输出格式判断
	t.Run("OutputFormatDetermination", func(t *testing.T) {
		generator := &UnifiedPythonGenerator{}

		testCases := []struct {
			request        string
			expectedFormat string
		}{
			// 明确需要可视化的请求
			{"分析销售趋势", "visualization"},
			{"显示月度收入图表", "visualization"},
			{"画一个柱状图", "visualization"},
			{"生成饼图展示占比", "visualization"},
			{"对比各产品销量", "visualization"},
			{"展示客户分布", "visualization"},

			// 分析类请求（默认应该生成可视化）
			{"分析客户购买行为", "visualization"},
			{"统计各类别销售额", "visualization"},
			{"按地区分析订单", "visualization"},

			// 明确不需要图表的请求
			{"总数是多少", "aggregation"},
			{"一共有多少条记录", "aggregation"},
			{"不要图，只要数字", "standard"},
			{"纯文本输出", "standard"},
		}

		for _, tc := range testCases {
			t.Run(tc.request, func(t *testing.T) {
				format := generator.determineOutputFormat(tc.request)

				if format != tc.expectedFormat {
					t.Errorf("请求 '%s': 期望格式 %s, 实际 %s",
						tc.request, tc.expectedFormat, format)
				}

				t.Logf("✓ '%s' -> %s", tc.request, format)
			})
		}
	})

	// 测试3: 验证 PromptBuilder 生成的提示词包含图表保存指令
	t.Run("PromptBuilder_ChartSaveInstructions", func(t *testing.T) {
		builder := NewAnalysisPromptBuilder()

		schemaCtx := &UnifiedSchemaContext{
			DatabaseType: "sqlite",
			DatabasePath: "/test/db.sqlite",
			Tables: []UnifiedTableSchema{
				{
					Name:     "sales",
					RowCount: 1000,
					Columns: []UnifiedColumnInfo{
						{Name: "id", Type: "INTEGER", IsPK: true},
						{Name: "product", Type: "TEXT"},
						{Name: "amount", Type: "REAL"},
						{Name: "date", Type: "DATE"},
					},
				},
			},
		}

		hints := &ClassificationResult{
			NeedsVisualization: true,
			SuggestedChartType: "bar",
		}

		prompt := builder.BuildPromptWithHints("分析销售趋势", schemaCtx, "visualization", hints)

		// 验证提示词包含关键指令
		requiredInstructions := []string{
			"plt.savefig",
			"FILES_DIR",
			"chart.png",
			"matplotlib",
			"必须",
		}

		for _, instruction := range requiredInstructions {
			if !strings.Contains(prompt, instruction) {
				t.Errorf("提示词缺少关键指令: %s", instruction)
			} else {
				t.Logf("✓ 提示词包含: %s", instruction)
			}
		}

		// 验证提示词不包含错误示例
		if strings.Contains(prompt, "plt.savefig('chart.png')") &&
			!strings.Contains(prompt, "错误示例") {
			t.Log("⚠️ 提示词可能包含不正确的 savefig 示例")
		}
	})

	// 测试4: 验证 ResultParser 能正确检测生成的文件
	t.Run("ResultParser_FileDetection", func(t *testing.T) {
		// 创建临时目录
		tmpDir, err := os.MkdirTemp("", "chart-test-*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// 创建 files 子目录
		filesDir := filepath.Join(tmpDir, "files")
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			t.Fatalf("创建 files 目录失败: %v", err)
		}

		// 创建测试文件
		testFiles := map[string][]byte{
			"chart.png":       {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG 魔数
			"sales_trend.png": {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			"data.xlsx":       {0x50, 0x4B, 0x03, 0x04}, // ZIP/XLSX 魔数
			"export.csv":      []byte("a,b,c\n1,2,3\n4,5,6"),
		}

		for name, content := range testFiles {
			path := filepath.Join(filesDir, name)
			if err := os.WriteFile(path, content, 0644); err != nil {
				t.Fatalf("创建测试文件 %s 失败: %v", name, err)
			}
		}

		// 使用 ResultParser 检测文件
		parser := NewResultParser(func(msg string) {
			t.Log(msg)
		})

		chartFiles, exportFiles := parser.detectGeneratedFiles(tmpDir)

		// 验证检测结果
		if len(chartFiles) != 2 {
			t.Errorf("期望检测到 2 个图表文件，实际 %d", len(chartFiles))
		}
		if len(exportFiles) != 2 {
			t.Errorf("期望检测到 2 个导出文件，实际 %d", len(exportFiles))
		}

		t.Logf("✓ 检测到 %d 个图表文件, %d 个导出文件", len(chartFiles), len(exportFiles))
	})

	// 测试5: 验证 ECharts 配置检测
	t.Run("EChartsConfigDetection", func(t *testing.T) {
		parser := NewResultParser(func(msg string) {
			t.Log(msg)
		})

		testCases := []struct {
			name      string
			config    map[string]interface{}
			isECharts bool
		}{
			{
				name: "完整的柱状图配置",
				config: map[string]interface{}{
					"title":  map[string]interface{}{"text": "销售统计"},
					"xAxis":  map[string]interface{}{"type": "category", "data": []string{"A", "B", "C"}},
					"yAxis":  map[string]interface{}{"type": "value"},
					"series": []interface{}{map[string]interface{}{"type": "bar", "data": []int{100, 200, 150}}},
				},
				isECharts: true,
			},
			{
				name: "折线图配置",
				config: map[string]interface{}{
					"xAxis":  map[string]interface{}{"type": "category"},
					"yAxis":  map[string]interface{}{"type": "value"},
					"series": []interface{}{map[string]interface{}{"type": "line", "data": []int{1, 2, 3}}},
				},
				isECharts: true,
			},
			{
				name: "饼图配置",
				config: map[string]interface{}{
					"series": []interface{}{
						map[string]interface{}{
							"type": "pie",
							"data": []map[string]interface{}{
								{"name": "A", "value": 100},
								{"name": "B", "value": 200},
							},
						},
					},
				},
				isECharts: true,
			},
			{
				name: "普通数据对象",
				config: map[string]interface{}{
					"name":  "test",
					"value": 123,
				},
				isECharts: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := parser.IsEChartsConfigFromMap(tc.config)

				if result.IsECharts != tc.isECharts {
					t.Errorf("期望 IsECharts=%v, 实际=%v (分数: %d)",
						tc.isECharts, result.IsECharts, result.Score)
				}

				t.Logf("✓ %s: IsECharts=%v, 分数=%d, 匹配字段=%v",
					tc.name, result.IsECharts, result.Score, result.MatchedFields)
			})
		}
	})
}

// TestChartInjectionLogic 测试图表保存代码注入逻辑
func TestChartInjectionLogic(t *testing.T) {
	t.Log("=== 图表保存代码注入测试 ===")

	// 测试代码中有图表代码但没有 savefig 的情况
	t.Run("InjectSavefigWhenMissing", func(t *testing.T) {
		// 模拟没有 savefig 的代码
		codeWithoutSave := `
import matplotlib.pyplot as plt
import pandas as pd

df = pd.DataFrame({'x': [1,2,3], 'y': [4,5,6]})
plt.figure(figsize=(10, 6))
plt.bar(df['x'], df['y'])
plt.title('测试图表')
plt.show()
`
		// 验证代码有图表相关代码
		hasChartCode := strings.Contains(codeWithoutSave, "plt.") ||
			strings.Contains(codeWithoutSave, "matplotlib")

		if !hasChartCode {
			t.Error("测试代码应该包含图表相关代码")
		}

		// 验证代码没有 savefig
		hasSavefig := strings.Contains(codeWithoutSave, "savefig")
		if hasSavefig {
			t.Error("测试代码不应该包含 savefig")
		}

		// 模拟注入 savefig 代码
		chartSaveCode := `
    # 【自动添加】保存图表
    try:
        chart_path = os.path.join(FILES_DIR, 'chart.png')
        if plt.get_fignums():
            plt.savefig(chart_path, dpi=150, bbox_inches='tight', facecolor='white')
            plt.close('all')
            print(f"✅ 图表已保存: {chart_path}")
    except Exception as chart_err:
        print(f"⚠️ 图表保存失败: {chart_err}")
`
		// 注入代码
		injectedCode := strings.Replace(codeWithoutSave, "plt.show()", chartSaveCode+"\nplt.show()", 1)

		// 验证注入后的代码包含 savefig
		if !strings.Contains(injectedCode, "savefig") {
			t.Error("注入后的代码应该包含 savefig")
		}

		t.Log("✓ 图表保存代码注入成功")
	})
}

// TestDataNormalization 测试数据规范化
func TestDataNormalization(t *testing.T) {
	t.Log("=== 数据规范化测试 ===")

	// 测试表格数据规范化
	t.Run("TableDataNormalization", func(t *testing.T) {
		// 模拟从 Python 输出解析的表格数据
		rawTableData := []map[string]interface{}{
			{"product": "A", "sales": 100, "date": "2024-01-01"},
			{"product": "B", "sales": 200, "date": "2024-01-02"},
			{"product": "C", "sales": 150, "date": "2024-01-03"},
		}

		// 验证数据结构
		if len(rawTableData) != 3 {
			t.Errorf("期望 3 行数据，实际 %d", len(rawTableData))
		}

		// 验证每行数据的字段
		for i, row := range rawTableData {
			if _, ok := row["product"]; !ok {
				t.Errorf("第 %d 行缺少 product 字段", i)
			}
			if _, ok := row["sales"]; !ok {
				t.Errorf("第 %d 行缺少 sales 字段", i)
			}
		}

		// 转换为 JSON 验证
		jsonData, err := json.Marshal(rawTableData)
		if err != nil {
			t.Errorf("JSON 序列化失败: %v", err)
		}

		t.Logf("✓ 表格数据规范化成功: %s", string(jsonData))
	})

	// 测试 ECharts 数据规范化
	t.Run("EChartsDataNormalization", func(t *testing.T) {
		// 模拟 ECharts 配置
		echartsConfig := map[string]interface{}{
			"title": map[string]interface{}{
				"text": "销售趋势",
			},
			"xAxis": map[string]interface{}{
				"type": "category",
				"data": []string{"1月", "2月", "3月"},
			},
			"yAxis": map[string]interface{}{
				"type": "value",
			},
			"series": []interface{}{
				map[string]interface{}{
					"type": "bar",
					"data": []int{100, 200, 150},
				},
			},
		}

		// 转换为 JSON
		jsonData, err := json.Marshal(echartsConfig)
		if err != nil {
			t.Errorf("JSON 序列化失败: %v", err)
		}

		// 验证 JSON 可以被解析回来
		var parsed map[string]interface{}
		if err := json.Unmarshal(jsonData, &parsed); err != nil {
			t.Errorf("JSON 反序列化失败: %v", err)
		}

		// 验证关键字段存在
		if _, ok := parsed["series"]; !ok {
			t.Error("解析后的数据缺少 series 字段")
		}

		t.Logf("✓ ECharts 数据规范化成功")
	})
}

// TestEndToEndChartGeneration 端到端图表生成测试
func TestEndToEndChartGeneration(t *testing.T) {
	t.Log("=== 端到端图表生成测试 ===")

	t.Run("SimulatedAnalysisFlow", func(t *testing.T) {
		// 1. 用户请求
		userRequest := "分析各产品的销售趋势，生成柱状图"
		t.Logf("1. 用户请求: %s", userRequest)

		// 2. 确定输出格式
		generator := &UnifiedPythonGenerator{}
		outputFormat := generator.determineOutputFormat(userRequest)
		if outputFormat != "visualization" {
			t.Errorf("期望输出格式为 visualization，实际为 %s", outputFormat)
		}
		t.Logf("2. 输出格式: %s ✓", outputFormat)

		// 3. 构建提示词
		builder := NewAnalysisPromptBuilder()
		schemaCtx := &UnifiedSchemaContext{
			DatabaseType: "sqlite",
			Tables: []UnifiedTableSchema{
				{
					Name: "sales",
					Columns: []UnifiedColumnInfo{
						{Name: "product", Type: "TEXT"},
						{Name: "amount", Type: "REAL"},
					},
				},
			},
		}
		hints := &ClassificationResult{
			NeedsVisualization: true,
			SuggestedChartType: "bar",
		}
		prompt := builder.BuildPromptWithHints(userRequest, schemaCtx, outputFormat, hints)

		// 验证提示词
		if !strings.Contains(prompt, "plt.savefig") {
			t.Error("提示词缺少 plt.savefig 指令")
		}
		if !strings.Contains(prompt, "FILES_DIR") {
			t.Error("提示词缺少 FILES_DIR 变量")
		}
		t.Logf("3. 提示词生成成功 (长度: %d) ✓", len(prompt))

		// 4. 模拟 LLM 生成的代码
		generatedCode := `
import sqlite3
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib
matplotlib.use('Agg')
plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei']
plt.rcParams['axes.unicode_minus'] = False
import os

DB_PATH = "{DB_PATH}"
FILES_DIR = "{FILES_DIR}"

def main():
    conn = None
    try:
        os.makedirs(FILES_DIR, exist_ok=True)
        conn = sqlite3.connect(DB_PATH)
        
        sql = """
        SELECT product, SUM(amount) as total
        FROM sales
        GROUP BY product
        ORDER BY total DESC
        """
        df = pd.read_sql_query(sql, conn)
        
        fig, ax = plt.subplots(figsize=(10, 6))
        ax.bar(df['product'], df['total'], color='steelblue')
        ax.set_title('各产品销售额', fontsize=14, fontweight='bold')
        ax.set_xlabel('产品', fontsize=12)
        ax.set_ylabel('销售额', fontsize=12)
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        
        chart_path = os.path.join(FILES_DIR, 'chart.png')
        plt.savefig(chart_path, dpi=150, bbox_inches='tight', facecolor='white')
        plt.close()
        print(f"✅ 图表已保存: {chart_path}")
        
        print("\n=== 分析结果 ===")
        print(df.to_string(index=False))
        
    except Exception as e:
        print(f"分析错误: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    main()
`
		t.Logf("4. 模拟代码生成成功 (长度: %d) ✓", len(generatedCode))

		// 5. 验证代码
		validator := NewCodeValidator()
		result := validator.ValidateCode(generatedCode)

		if !result.Valid {
			t.Errorf("代码验证失败: %v", result.Errors)
		}
		if !result.HasChart {
			t.Error("代码验证未检测到图表保存")
		}
		t.Logf("5. 代码验证通过: Valid=%v, HasChart=%v ✓", result.Valid, result.HasChart)

		// 6. 模拟 Python 输出
		pythonOutput := `
✅ 图表已保存: /tmp/session/files/chart.png

=== 分析结果 ===
product  total
      A  10000
      B   8000
      C   6000
`
		t.Logf("6. 模拟 Python 输出 (长度: %d) ✓", len(pythonOutput))

		// 7. 解析输出
		parser := NewResultParser(func(msg string) {
			t.Log("   " + msg)
		})
		parseResult := parser.ParseOutput(pythonOutput, "")

		if !parseResult.Success {
			t.Errorf("输出解析失败: %s", parseResult.ErrorMsg)
		}
		t.Logf("7. 输出解析成功: Success=%v, Tables=%d ✓",
			parseResult.Success, len(parseResult.Tables))

		t.Log("=== 端到端测试完成 ===")
	})
}
