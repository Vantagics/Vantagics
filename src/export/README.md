# Export Package

这个包提供了 PDF、Excel 和 PowerPoint 导出功能，用于导出仪表盘数据和分析结果。

## 功能特性

### PDF 导出 (使用 maroto v2)

- ✅ 无需 Chrome 浏览器依赖
- ✅ 更快的生成速度
- ✅ 更小的二进制文件大小
- ✅ 支持中文字体
- ✅ 专业的报告布局
- ✅ 支持图表、表格、指标卡片

### Excel 导出 (使用 excelize v2)

- ✅ 原生 Excel 格式 (.xlsx)
- ✅ 支持多个工作表
- ✅ 自动列宽调整
- ✅ 表头样式美化
- ✅ 自动筛选功能
- ✅ 冻结首行
- ✅ 完整的元数据

### PowerPoint 导出 (使用 unioffice)

- ✅ 原生 PowerPoint 格式 (.pptx)
- ✅ 专业的幻灯片布局
- ✅ 支持图表、表格、指标卡片
- ✅ 自动分页和排版
- ✅ 精美的视觉设计
- ✅ 完整的演示文稿结构

## 使用方法

### PDF 导出

```go
import "rapidbi/export"

// 创建 PDF 导出服务
pdfService := export.NewPDFExportService()

// 准备数据
data := export.DashboardData{
    UserRequest: "分析销售数据",
    Metrics: []export.MetricData{
        {Title: "总销售额", Value: "¥1,234,567", Change: "+15.3%"},
    },
    Insights: []string{
        "销售额环比上月增长15.3%",
    },
    ChartImages: []string{
        "data:image/png;base64,...", // base64 编码的图片
    },
    TableData: &export.TableData{
        Columns: []export.TableColumn{
            {Title: "产品名称", DataType: "string"},
            {Title: "销售额", DataType: "number"},
        },
        Data: [][]interface{}{
            {"产品A", 50000},
            {"产品B", 38000},
        },
    },
}

// 生成 PDF
pdfBytes, err := pdfService.ExportDashboardToPDF(data)
if err != nil {
    log.Fatal(err)
}

// 保存文件
os.WriteFile("report.pdf", pdfBytes, 0644)
```

### Excel 导出

```go
import "rapidbi/export"

// 创建 Excel 导出服务
excelService := export.NewExcelExportService()

// 准备表格数据
tableData := &export.TableData{
    Columns: []export.TableColumn{
        {Title: "日期", DataType: "string"},
        {Title: "销售额", DataType: "number"},
    },
    Data: [][]interface{}{
        {"2024-01-01", 12500},
        {"2024-01-02", 13200},
    },
}

// 生成 Excel (单个工作表)
excelBytes, err := excelService.ExportTableToExcel(tableData, "销售数据")
if err != nil {
    log.Fatal(err)
}

// 保存文件
os.WriteFile("data.xlsx", excelBytes, 0644)
```

### PowerPoint 导出

```go
import "rapidbi/export"

// 创建 PPT 导出服务
pptService := export.NewPPTExportService()

// 准备数据（与PDF相同的数据结构）
data := export.DashboardData{
    UserRequest: "分析销售数据",
    Metrics: []export.MetricData{
        {Title: "总销售额", Value: "¥1,234,567", Change: "+15.3%"},
        {Title: "订单数量", Value: "3,456", Change: "+8.7%"},
    },
    Insights: []string{
        "销售额环比上月增长15.3%",
        "新客户占比达到34%",
    },
    ChartImages: []string{
        "data:image/png;base64,...",
    },
    TableData: &export.TableData{
        Columns: []export.TableColumn{
            {Title: "产品名称", DataType: "string"},
            {Title: "销售额", DataType: "number"},
        },
        Data: [][]interface{}{
            {"产品A", 50000},
            {"产品B", 38000},
        },
    },
}

// 生成 PPT
pptBytes, err := pptService.ExportDashboardToPPT(data)
if err != nil {
    log.Fatal(err)
}

// 保存文件
os.WriteFile("presentation.pptx", pptBytes, 0644)
```

### 多工作表 Excel 导出

```go
// 准备多个表格
tables := map[string]*export.TableData{
    "销售数据": salesTable,
    "客户数据": customerTable,
    "产品数据": productTable,
}

// 生成包含多个工作表的 Excel
excelBytes, err := excelService.ExportMultipleTablesToExcel(tables)
if err != nil {
    log.Fatal(err)
}

os.WriteFile("report.xlsx", excelBytes, 0644)
```

## 在 App 中集成

在 `app_dashboard_export.go` 中已经集成了新的导出功能：

### PDF 导出

```go
// 优先使用 maroto，失败时回退到 chromedp
func (a *App) ExportDashboardToPDF(data DashboardExportData) error
```

### Excel 导出

```go
// 导出单个表格
func (a *App) ExportTableToExcel(tableData *TableData, sheetName string) error

// 导出仪表盘数据
func (a *App) ExportDashboardToExcel(data DashboardExportData) error
```

### PowerPoint 导出

```go
// 导出仪表盘为PPT
func (a *App) ExportDashboardToPPT(data DashboardExportData) error
```

## 前端调用

在前端可以通过 Wails 绑定调用：

```typescript
import { ExportDashboardToPDF, ExportDashboardToExcel, ExportDashboardToPPT } from '../../wailsjs/go/main/App';

// 导出 PDF
await ExportDashboardToPDF(dashboardData);

// 导出 Excel
await ExportDashboardToExcel(dashboardData);

// 导出 PPT
await ExportDashboardToPPT(dashboardData);
```

## 依赖库

- **maroto v2** (`github.com/johnfercher/maroto/v2`): PDF 生成
- **excelize v2** (`github.com/xuri/excelize/v2`): Excel 生成
- **unioffice** (`github.com/unidoc/unioffice`): PowerPoint 生成

## 优势对比

### PDF 导出

| 特性 | maroto (新) | chromedp (旧) |
|------|------------|--------------|
| Chrome 依赖 | ❌ 不需要 | ✅ 需要 |
| 生成速度 | ⚡ 快 | 🐌 慢 |
| 二进制大小 | 📦 小 | 📦 大 |
| 跨平台 | ✅ 完全支持 | ⚠️ 需要 Chrome |
| 自定义布局 | ✅ 灵活 | ⚠️ 受限于 HTML/CSS |

### Excel 导出

| 特性 | excelize (新) | CSV (旧) |
|------|--------------|----------|
| 格式 | .xlsx | .csv |
| 多工作表 | ✅ 支持 | ❌ 不支持 |
| 样式 | ✅ 丰富 | ❌ 无 |
| 公式 | ✅ 支持 | ❌ 不支持 |
| 筛选 | ✅ 自动 | ❌ 无 |
| 冻结窗格 | ✅ 支持 | ❌ 不支持 |

### PowerPoint 导出

| 特性 | unioffice | 手动制作 |
|------|-----------|---------|
| 自动化 | ✅ 完全自动 | ❌ 手动 |
| 一致性 | ✅ 完美一致 | ⚠️ 可能不一致 |
| 效率 | ⚡ 秒级生成 | 🐌 分钟级 |
| 模板 | ✅ 可定制 | ⚠️ 需要设计 |
| 批量生成 | ✅ 支持 | ❌ 困难 |

## 测试

运行测试：

```bash
cd src
go test -v ./export/
```

测试会生成示例文件：
- `test_dashboard.pdf` - PDF 报告示例
- `test_table.xlsx` - 单表 Excel 示例
- `test_multi_tables.xlsx` - 多表 Excel 示例
- `test_dashboard.pptx` - PowerPoint 演示文稿示例

## 注意事项

1. **图片格式**: 所有导出都支持 PNG 格式的 base64 图片
2. **表格大小**: 
   - PDF 中表格限制为 50 行、6 列
   - PPT 中表格限制为 10 行、6 列
   - Excel 无限制
3. **幻灯片数量**: PPT 会根据内容自动生成多张幻灯片
4. **中文支持**: 所有导出格式都完全支持中文
5. **文件大小**: 
   - Excel 文件通常最小
   - PDF 文件适中
   - PPT 文件可能较大（包含图片时）

## PPT 幻灯片结构

生成的 PowerPoint 包含以下幻灯片：

1. **标题页** - 显示报告标题和用户请求
2. **关键指标页** - 以卡片形式展示指标（最多6个）
3. **智能洞察页** - 以项目符号列表展示洞察（最多8条）
4. **图表页** - 每个图表一张幻灯片
5. **数据表格页** - 展示表格数据（最多10行6列）
6. **结束页** - 感谢页面

## 未来改进

- [ ] 支持更多图片格式 (JPEG, GIF, SVG)
- [ ] PDF 添加页眉页脚
- [ ] Excel 添加图表
- [ ] PPT 添加动画效果
- [ ] 支持自定义样式主题
- [ ] 批量导出功能
- [ ] 导出进度回调
- [ ] PPT 添加备注页
- [ ] 支持自定义幻灯片模板
