# Export Package

这个包提供了 PDF、Excel、PowerPoint 和 Word 导出功能，用于导出仪表盘数据和分析结果。

## 依赖库 (VantageOffice)

本包使用 VantageOffice 系列纯 Go 库替代了之前的第三方依赖：

| 功能 | 库 | 替代 |
|------|-----|------|
| Excel 导出 | [GoExcel](https://github.com/VantageDataChat/GoExcel) | excelize/v2 |
| PPT 导出 | [GoPPT](https://github.com/VantageDataChat/GoPPT) | gooxml + unioffice |
| Word 导出 | [GoWord](https://github.com/VantageDataChat/GoWord) | 新增功能 |
| PDF 导出 | maroto v2 + gopdf | 保持不变 |

VantageOffice 系列库的优势：
- 纯 Go 实现，零外部依赖
- MIT 开源协议
- 高测试覆盖率 (93%~100%)
- 完整的中文支持

## 功能特性

### PDF 导出 (使用 maroto v2 + gopdf)

- 无需 Chrome 浏览器依赖
- 支持中文字体
- 专业的报告布局
- 支持图表、表格、指标卡片

### Excel 导出 (使用 GoExcel)

- 原生 Excel 格式 (.xlsx)
- 支持多个工作表
- 自动列宽调整
- 表头样式美化
- 冻结首行
- 完整的元数据

### PowerPoint 导出 (使用 GoPPT)

- 原生 PowerPoint 格式 (.pptx)
- 专业的幻灯片布局
- 支持图表图片、表格、指标卡片
- 自动分页和排版
- 精美的视觉设计

### Word 导出 (使用 GoWord) - 新增

- 原生 Word 格式 (.docx)
- 支持标题、段落、表格
- Markdown 格式解析
- 专业的报告布局
