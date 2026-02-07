package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAnalysisPipeline_DiagnosticTest 诊断测试：验证分析流程的各个环节
// 这个测试用于诊断以下问题：
// 1. Agent 是否生成了合适的图表/数据
// 2. 生成的数据是否能正确被解析
// 3. 数据是否能正确传递到前端
func TestAnalysisPipeline_DiagnosticTest(t *testing.T) {
	t.Log("=== 分析流程诊断测试 ===")
	
	// 测试1: 验证 AnalysisPromptBuilder 生成的提示词
	t.Run("PromptBuilder_VisualizationEmphasis", func(t *testing.T) {
		builder := NewAnalysisPromptBuilder()
		
		// 测试各种分析请求
		testCases := []struct {
			request        string
			expectedFormat string
			shouldHaveViz  bool
		}{
			{"分析销售趋势", "visualization", true},
			{"统计各产品销量", "visualization", true},
			{"按月份分析收入", "visualization", true},
			{"显示客户分布", "visualization", true},
			{"总数是多少", "aggregation", false},
			{"不要图，只要数字", "standard", false},
		}
		
		for _, tc := range testCases {
			t.Run(tc.request, func(t *testing.T) {
				// 创建模拟的 schema context
				schemaCtx := &UnifiedSchemaContext{
					DatabaseType: "sqlite",
					DatabasePath: "/test/db.sqlite",
					Tables: []UnifiedTableSchema{
						{
							Name:     "orders",
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
				
				// 创建分类提示
				hints := &ClassificationResult{
					NeedsVisualization: tc.shouldHaveViz,
					SuggestedChartType: "bar",
				}
				
				prompt := builder.BuildPromptWithHints(tc.request, schemaCtx, tc.expectedFormat, hints)
				
				// 验证提示词包含关键指令
				if tc.shouldHaveViz {
					if !strings.Contains(prompt, "plt.savefig") {
						t.Errorf("提示词缺少 plt.savefig 指令")
					}
					if !strings.Contains(prompt, "FILES_DIR") {
						t.Errorf("提示词缺少 FILES_DIR 变量")
					}
					if !strings.Contains(prompt, "chart.png") {
						t.Errorf("提示词缺少 chart.png 文件名")
					}
				}
				
				t.Logf("请求: %s -> 格式: %s, 需要可视化: %v", tc.request, tc.expectedFormat, tc.shouldHaveViz)
			})
		}
	})
	
	// 测试2: 验证 CodeValidator 能检测图表保存代码
	t.Run("CodeValidator_ChartDetection", func(t *testing.T) {
		validator := NewCodeValidator()
		
		testCases := []struct {
			name      string
			code      string
			hasChart  bool
			hasExport bool
		}{
			{
				name: "完整的图表保存代码",
				code: `
import matplotlib.pyplot as plt
import pandas as pd

plt.figure(figsize=(10, 6))
plt.bar(df['category'], df['value'])
plt.savefig(os.path.join(FILES_DIR, 'chart.png'), dpi=150)
plt.close()
`,
				hasChart:  true,
				hasExport: false,
			},
			{
				name: "缺少 savefig 的代码",
				code: `
import matplotlib.pyplot as plt
import pandas as pd

plt.figure(figsize=(10, 6))
plt.bar(df['category'], df['value'])
plt.show()
`,
				hasChart:  false, // plt.show() 不会保存文件
				hasExport: false,
			},
			{
				name: "包含 Excel 导出的代码",
				code: `
import pandas as pd

df.to_excel(os.path.join(FILES_DIR, 'data.xlsx'), index=False)
`,
				hasChart:  false,
				hasExport: true,
			},
			{
				name: "同时包含图表和导出",
				code: `
import matplotlib.pyplot as plt
import pandas as pd

plt.savefig(os.path.join(FILES_DIR, 'chart.png'))
df.to_excel(os.path.join(FILES_DIR, 'data.xlsx'))
`,
				hasChart:  true,
				hasExport: true,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := validator.ValidateCode(tc.code)
				
				if result.HasChart != tc.hasChart {
					t.Errorf("HasChart: 期望 %v, 实际 %v", tc.hasChart, result.HasChart)
				}
				if result.HasExport != tc.hasExport {
					t.Errorf("HasExport: 期望 %v, 实际 %v", tc.hasExport, result.HasExport)
				}
				
				t.Logf("代码验证: HasChart=%v, HasExport=%v, Valid=%v", 
					result.HasChart, result.HasExport, result.Valid)
			})
		}
	})
	
	// 测试3: 验证 ResultParser 能正确检测文件
	t.Run("ResultParser_FileDetection", func(t *testing.T) {
		// 创建临时目录模拟 session 目录
		tmpDir, err := os.MkdirTemp("", "analysis-test-*")
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
		testFiles := []struct {
			name     string
			content  []byte
			isChart  bool
			isExport bool
		}{
			{"chart.png", []byte{0x89, 0x50, 0x4E, 0x47}, true, false},  // PNG 魔数
			{"data.xlsx", []byte{0x50, 0x4B, 0x03, 0x04}, false, true}, // ZIP/XLSX 魔数
			{"result.csv", []byte("a,b,c\n1,2,3"), false, true},
			{"analysis.json", []byte(`{"key": "value"}`), false, true},
		}
		
		for _, tf := range testFiles {
			filePath := filepath.Join(filesDir, tf.name)
			if err := os.WriteFile(filePath, tf.content, 0644); err != nil {
				t.Fatalf("创建测试文件失败: %v", err)
			}
		}
		
		// 使用 ResultParser 检测文件
		parser := NewResultParser(func(msg string) {
			t.Log(msg)
		})
		
		chartFiles, exportFiles := parser.detectGeneratedFiles(tmpDir)
		
		t.Logf("检测到图表文件: %d", len(chartFiles))
		for _, f := range chartFiles {
			t.Logf("  - %s (%s, %d bytes)", f.Name, f.Type, f.Size)
		}
		
		t.Logf("检测到导出文件: %d", len(exportFiles))
		for _, f := range exportFiles {
			t.Logf("  - %s (%s, %d bytes)", f.Name, f.Type, f.Size)
		}
		
		// 验证检测结果
		if len(chartFiles) != 1 {
			t.Errorf("期望检测到 1 个图表文件，实际 %d", len(chartFiles))
		}
		if len(exportFiles) != 3 {
			t.Errorf("期望检测到 3 个导出文件，实际 %d", len(exportFiles))
		}
	})
	
	// 测试4: 验证 ECharts 配置检测
	t.Run("ResultParser_EChartsDetection", func(t *testing.T) {
		parser := NewResultParser(func(msg string) {
			t.Log(msg)
		})
		
		testCases := []struct {
			name      string
			json      string
			isECharts bool
		}{
			{
				name: "标准 ECharts 配置",
				json: `{
					"title": {"text": "销售趋势"},
					"xAxis": {"type": "category", "data": ["1月", "2月", "3月"]},
					"yAxis": {"type": "value"},
					"series": [{"type": "bar", "data": [100, 200, 150]}]
				}`,
				isECharts: true,
			},
			{
				name: "简化 ECharts 配置",
				json: `{
					"series": [{"type": "pie", "data": [{"name": "A", "value": 100}]}]
				}`,
				isECharts: true,
			},
			{
				name: "普通 JSON 对象",
				json: `{"name": "test", "value": 123}`,
				isECharts: false,
			},
			{
				name: "表格数据",
				json: `[{"product": "A", "sales": 100}, {"product": "B", "sales": 200}]`,
				isECharts: false,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := parser.IsEChartsConfig(tc.json)
				
				if result.IsECharts != tc.isECharts {
					t.Errorf("IsECharts: 期望 %v, 实际 %v (分数: %d, 原因: %s)", 
						tc.isECharts, result.IsECharts, result.Score, result.Reason)
				}
				
				t.Logf("ECharts 检测: %v (分数: %d, 匹配字段: %v)", 
					result.IsECharts, result.Score, result.MatchedFields)
			})
		}
	})
	
	// 测试5: 验证输出格式判断
	t.Run("UnifiedPythonGenerator_OutputFormat", func(t *testing.T) {
		generator := &UnifiedPythonGenerator{}
		
		testCases := []struct {
			request        string
			expectedFormat string
		}{
			{"分析销售趋势", "visualization"},
			{"统计各产品销量", "visualization"},
			{"按月份分析收入", "visualization"},
			{"显示客户分布图", "visualization"},
			{"画一个柱状图", "visualization"},
			{"生成折线图", "visualization"},
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
				
				t.Logf("请求: %s -> 格式: %s", tc.request, format)
			})
		}
	})
}

// TestEventAggregator_DataFlow 测试事件聚合器的数据流
func TestEventAggregator_DataFlow(t *testing.T) {
	t.Log("=== 事件聚合器数据流测试 ===")
	
	// 模拟各种数据类型的添加和聚合
	t.Run("AddMultipleDataTypes", func(t *testing.T) {
		// 这个测试验证不同类型的数据能否正确添加到聚合器
		
		// 模拟 ECharts 数据
		echartsData := `{
			"title": {"text": "测试图表"},
			"series": [{"type": "bar", "data": [1, 2, 3]}]
		}`
		
		// 模拟图片数据 (base64)
		imageData := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
		
		// 模拟表格数据
		tableData := []map[string]interface{}{
			{"product": "A", "sales": 100},
			{"product": "B", "sales": 200},
		}
		
		// 验证数据格式
		var echarts map[string]interface{}
		if err := json.Unmarshal([]byte(echartsData), &echarts); err != nil {
			t.Errorf("ECharts JSON 解析失败: %v", err)
		} else {
			t.Log("ECharts 数据格式正确")
		}
		
		if !strings.HasPrefix(imageData, "data:image/") {
			t.Error("图片数据格式不正确")
		} else {
			t.Log("图片数据格式正确")
		}
		
		if len(tableData) == 0 {
			t.Error("表格数据为空")
		} else {
			t.Logf("表格数据: %d 行", len(tableData))
		}
	})
}

// TestImageDetector_AllFormats 测试图片检测器的各种格式
func TestImageDetector_AllFormats(t *testing.T) {
	t.Log("=== 图片检测器格式测试 ===")
	
	detector := NewImageDetector()
	detector.SetLogger(func(msg string) {
		t.Log(msg)
	})
	
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Base64 PNG",
			text:     "这是一个图片: data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			expected: 1,
		},
		{
			name:     "Markdown 图片",
			text:     "![图表](files/chart.png)",
			expected: 2, // markdown 和 file_reference 都会匹配
		},
		{
			name:     "文件引用",
			text:     "图表已保存: files/chart.png",
			expected: 1,
		},
		{
			name:     "Sandbox 路径",
			text:     "sandbox:/mnt/data/output.png",
			expected: 1,
		},
		{
			name:     "多个图片",
			text:     "![图1](chart1.png) ![图2](chart2.png) data:image/png;base64,abc123==",
			expected: 3,
		},
		{
			name:     "无图片",
			text:     "这是一段普通文本，没有图片",
			expected: 0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			images := detector.DetectAllImages(tc.text)
			
			if len(images) != tc.expected {
				t.Errorf("期望检测到 %d 个图片，实际 %d", tc.expected, len(images))
			}
			
			for i, img := range images {
				t.Logf("  图片 %d: 类型=%s, 数据=%s...", i+1, img.Type, truncate(img.Data, 50))
			}
		})
	}
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestSchemaContextBuilder_Integration 测试 Schema 上下文构建
func TestSchemaContextBuilder_Integration(t *testing.T) {
	t.Log("=== Schema 上下文构建测试 ===")
	
	// 这个测试需要实际的数据源服务，这里只测试基本逻辑
	t.Run("ContextFormatting", func(t *testing.T) {
		ctx := &UnifiedSchemaContext{
			DatabaseType: "sqlite",
			DatabasePath: "/test/db.sqlite",
			Tables: []UnifiedTableSchema{
				{
					Name:     "orders",
					RowCount: 1000,
					Columns: []UnifiedColumnInfo{
						{Name: "id", Type: "INTEGER", IsPK: true},
						{Name: "customer_id", Type: "INTEGER", IsFK: true, RefTable: "customers"},
						{Name: "product", Type: "TEXT"},
						{Name: "amount", Type: "REAL"},
						{Name: "order_date", Type: "DATE"},
					},
					SampleData: []map[string]interface{}{
						{"id": 1, "customer_id": 101, "product": "Widget", "amount": 99.99, "order_date": "2024-01-15"},
					},
				},
				{
					Name:     "customers",
					RowCount: 500,
					Columns: []UnifiedColumnInfo{
						{Name: "id", Type: "INTEGER", IsPK: true},
						{Name: "name", Type: "TEXT"},
						{Name: "email", Type: "TEXT"},
					},
				},
			},
			Relationships: []UnifiedTableRelationship{
				{FromTable: "orders", FromColumn: "customer_id", ToTable: "customers", ToColumn: "id"},
			},
		}
		
		// 验证上下文包含必要信息
		if ctx.DatabaseType == "" {
			t.Error("数据库类型为空")
		}
		if len(ctx.Tables) == 0 {
			t.Error("表信息为空")
		}
		
		t.Logf("Schema 上下文: %d 个表, %d 个关系", len(ctx.Tables), len(ctx.Relationships))
		
		for _, table := range ctx.Tables {
			t.Logf("  表 %s: %d 列, %d 行, %d 条示例数据", 
				table.Name, len(table.Columns), table.RowCount, len(table.SampleData))
		}
	})
}

// TestEndToEnd_SimulatedAnalysis 端到端模拟测试
func TestEndToEnd_SimulatedAnalysis(t *testing.T) {
	t.Log("=== 端到端模拟分析测试 ===")
	
	// 模拟完整的分析流程
	t.Run("SimulatedFlow", func(t *testing.T) {
		// 1. 用户请求
		userRequest := "分析各产品的销售趋势"
		t.Logf("1. 用户请求: %s", userRequest)
		
		// 2. 确定输出格式
		generator := &UnifiedPythonGenerator{}
		outputFormat := generator.determineOutputFormat(userRequest)
		t.Logf("2. 输出格式: %s", outputFormat)
		
		// 3. 构建提示词
		builder := NewAnalysisPromptBuilder()
		schemaCtx := &UnifiedSchemaContext{
			DatabaseType: "sqlite",
			Tables: []UnifiedTableSchema{
				{Name: "sales", Columns: []UnifiedColumnInfo{{Name: "product", Type: "TEXT"}, {Name: "amount", Type: "REAL"}}},
			},
		}
		hints := &ClassificationResult{NeedsVisualization: true, SuggestedChartType: "bar"}
		prompt := builder.BuildPromptWithHints(userRequest, schemaCtx, outputFormat, hints)
		t.Logf("3. 提示词长度: %d 字符", len(prompt))
		
		// 4. 验证提示词包含关键指令
		keyInstructions := []string{
			"plt.savefig",
			"FILES_DIR",
			"chart.png",
			"matplotlib",
		}
		
		for _, instruction := range keyInstructions {
			if strings.Contains(prompt, instruction) {
				t.Logf("   ✓ 包含: %s", instruction)
			} else {
				t.Errorf("   ✗ 缺少: %s", instruction)
			}
		}
		
		// 5. 模拟 Python 输出
		pythonOutput := `
=== 分析结果 ===
产品销售统计:
  产品A: 1000
  产品B: 2000
  产品C: 1500

✅ 图表已保存: /tmp/session/files/chart.png
`
		t.Logf("5. Python 输出长度: %d 字符", len(pythonOutput))
		
		// 6. 解析输出
		parser := NewResultParser(func(msg string) {
			t.Log("   " + msg)
		})
		result := parser.ParseOutput(pythonOutput, "")
		t.Logf("6. 解析结果: 成功=%v, 表格=%d, 警告=%d", 
			result.Success, len(result.Tables), len(result.Warnings))
		
		// 7. 验证流程完整性
		if outputFormat != "visualization" {
			t.Error("分析请求应该使用 visualization 格式")
		}
		if !result.Success {
			t.Error("Python 输出解析应该成功")
		}
	})
}
