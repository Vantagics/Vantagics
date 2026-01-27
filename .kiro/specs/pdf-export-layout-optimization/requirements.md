# 需求文档

## 简介

本功能旨在优化仪表盘数据导出时的PDF排版质量，包括字体优化、表格优化、图片/图表显示优化、版面优化和分页优化。目标是生成版面饱满、字体大小合适、分页合理的A4格式PDF报告。

## 术语表

- **PDF_Export_Service**: PDF导出服务，负责将仪表盘数据转换为PDF格式
- **GoPDF_Service**: 使用gopdf库实现的PDF导出服务，支持中文字体
- **Layout_Engine**: 版面布局引擎，负责计算和管理页面元素的位置和尺寸
- **Table_Renderer**: 表格渲染器，负责绘制表格及其样式
- **Chart_Renderer**: 图表渲染器，负责处理图表图片的显示
- **Page_Manager**: 分页管理器，负责智能分页决策
- **Font_Manager**: 字体管理器，负责字体加载和样式管理
- **Content_Area**: 内容区域，指页面除去边距后的可用区域

## 需求

### 需求 1：字体优化

**用户故事：** 作为用户，我希望导出的PDF文档字体清晰、大小统一、层次分明，以便于阅读和理解报告内容。

#### 验收标准

1. THE Font_Manager SHALL 使用微软雅黑作为默认中文字体，Arial作为英文备选字体
2. WHEN 渲染标题时，THE Font_Manager SHALL 使用14-16pt字号，并应用粗体样式
3. WHEN 渲染正文时，THE Font_Manager SHALL 使用10-11pt字号
4. WHEN 渲染表格内容时，THE Font_Manager SHALL 使用9pt字号
5. WHEN 渲染页眉页脚时，THE Font_Manager SHALL 使用8pt字号
6. THE Layout_Engine SHALL 设置行间距为字体大小的1.3-1.5倍
7. IF 指定字体不可用，THEN THE Font_Manager SHALL 按优先级顺序尝试备选字体并记录警告日志

### 需求 2：表格优化

**用户故事：** 作为用户，我希望导出的PDF中表格清晰美观、内容完整显示，以便于查看和分析数据。

#### 验收标准

1. THE Table_Renderer SHALL 根据列内容长度智能计算列宽，确保内容不被截断
2. WHEN 渲染表头时，THE Table_Renderer SHALL 应用深蓝色背景(#4472C4)、白色文字、粗体样式
3. THE Table_Renderer SHALL 对数据行应用斑马纹样式（奇偶行交替背景色）
4. WHEN 单元格内容超过列宽时，THE Table_Renderer SHALL 自动换行显示
5. THE Table_Renderer SHALL 为表格添加细边框以提高可读性
6. WHEN 表格跨页时，THE Table_Renderer SHALL 在新页面顶部重复显示表头
7. THE Table_Renderer SHALL 支持最多8列的表格显示，超出列数时智能选择最重要的列

### 需求 3：图片/图表优化

**用户故事：** 作为用户，我希望导出的PDF中图表清晰、尺寸合适、位置居中，以便于直观理解数据可视化结果。

#### 验收标准

1. THE Chart_Renderer SHALL 将图表水平居中显示在内容区域
2. WHEN 渲染图表时，THE Chart_Renderer SHALL 根据图表原始比例自适应调整尺寸
3. THE Chart_Renderer SHALL 确保图表宽度不超过内容区域宽度的95%
4. THE Chart_Renderer SHALL 确保图表高度不超过页面可用高度的60%
5. WHEN 图表有标题时，THE Chart_Renderer SHALL 在图表上方显示标题文字
6. IF 图表无法在当前页面完整显示，THEN THE Page_Manager SHALL 将图表移至下一页
7. THE Chart_Renderer SHALL 在图表下方保留适当间距（至少8mm）

### 需求 4：版面优化

**用户故事：** 作为用户，我希望导出的PDF版面饱满、内容区域最大化利用，以便在有限页面内展示更多信息。

#### 验收标准

1. THE Layout_Engine SHALL 设置页面左右边距各为15mm
2. THE Layout_Engine SHALL 设置页面上边距为12mm，下边距为15mm（为页脚预留空间）
3. THE Layout_Engine SHALL 计算内容区域宽度为180mm（A4宽度210mm减去左右边距）
4. THE Layout_Engine SHALL 设置段落间距为8-10mm
5. THE Layout_Engine SHALL 设置章节标题与内容之间的间距为6mm
6. WHEN 渲染洞察文本时，THE Layout_Engine SHALL 设置段落首行缩进为0（左对齐）
7. THE Layout_Engine SHALL 确保页面内容填充率不低于75%

### 需求 5：分页优化

**用户故事：** 作为用户，我希望导出的PDF分页合理，避免内容在不恰当位置被分割，以便于连续阅读。

#### 验收标准

1. WHEN 表格需要分页时，THE Page_Manager SHALL 确保至少有3行数据与表头在同一页
2. THE Page_Manager SHALL 避免标题与其后续内容分离（标题后至少跟随2行内容）
3. WHEN 图表无法在当前页完整显示时，THE Page_Manager SHALL 将整个图表移至下一页
4. THE Page_Manager SHALL 在每页底部添加页脚，显示页码和生成时间
5. THE Page_Manager SHALL 在每页顶部添加页眉，显示报告标题
6. IF 当前页剩余空间小于30mm，THEN THE Page_Manager SHALL 开始新页面
7. WHEN 洞察段落需要分页时，THE Page_Manager SHALL 在自然段落边界处分页

### 需求 6：页眉页脚设计

**用户故事：** 作为用户，我希望导出的PDF有专业的页眉页脚，以便于识别报告来源和页面位置。

#### 验收标准

1. THE Layout_Engine SHALL 在每页顶部显示页眉，包含报告标题
2. THE Layout_Engine SHALL 在每页底部显示页脚，包含页码（格式：第X页/共Y页）
3. THE Layout_Engine SHALL 在页脚显示生成时间（格式：YYYY-MM-DD HH:mm:ss）
4. THE Layout_Engine SHALL 在页脚显示系统标识"VantageData 智能分析系统"
5. THE Layout_Engine SHALL 使用浅灰色(#94A3B8)作为页眉页脚文字颜色
6. THE Layout_Engine SHALL 在页眉下方添加细分隔线
7. WHEN 渲染首页时，THE Layout_Engine SHALL 显示完整标题区域而非简化页眉
