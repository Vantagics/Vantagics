# 需求文档

## 简介

简化当前三面板布局：移除左侧面板中不再需要的"数据源"区域，并修复面板之间拖拽调整大小的功能。数据源管理已通过弹窗（Modal）实现，左侧面板只需保留"新建会话"按钮和"历史会话"列表。同时，面板之间的 ResizeHandle 拖拽功能存在问题，需要排查并修复。

## 术语表

- **Left_Panel**: 左侧面板组件（LeftPanel），位于三面板布局最左侧，当前包含数据源区域、新建会话按钮和历史会话列表
- **DataSourcesSection**: 数据源区域组件，显示数据源列表和添加按钮，本次需求中将被移除
- **NewSessionButton**: 新建会话按钮组件，用于创建新的分析会话
- **HistoricalSessionsSection**: 历史会话列表组件，按时间倒序显示历史会话
- **ResizeHandle**: 拖拽调整大小的手柄组件，位于面板之间，允许用户通过拖拽改变相邻面板的宽度
- **CenterPanel**: 中间面板组件，显示聊天/对话界面
- **RightPanel**: 右侧面板组件，显示仪表盘、图表和分析结果
- **PanelWidths**: 面板宽度管理工具模块，负责计算、约束和持久化面板宽度

## 需求

### 需求 1：移除左侧面板中的数据源区域

**用户故事：** 作为用户，我希望左侧面板不再显示数据源区域，以便界面更简洁，因为数据源管理已通过弹窗完成。

#### 验收标准

1. WHEN Left_Panel 渲染时, THE Left_Panel SHALL 只显示 NewSessionButton 和 HistoricalSessionsSection 两个子组件
2. WHEN Left_Panel 渲染时, THE Left_Panel SHALL 不再渲染 DataSourcesSection 组件
3. WHEN Left_Panel 初始化时, THE Left_Panel SHALL 不再调用数据源相关的后端接口（GetDataSources）
4. WHEN Left_Panel 初始化时, THE Left_Panel SHALL 不再监听数据源相关的事件（data-source-added、data-source-deleted、data-source-renamed）

### 需求 2：调整左侧面板布局

**用户故事：** 作为用户，我希望移除数据源区域后，左侧面板的布局合理，历史会话列表能充分利用可用空间。

#### 验收标准

1. WHEN DataSourcesSection 被移除后, THE HistoricalSessionsSection SHALL 占据 Left_Panel 中 NewSessionButton 以下的全部剩余空间
2. WHEN Left_Panel 渲染时, THE NewSessionButton SHALL 位于面板顶部
3. WHEN NewSessionButton 不再依赖 selectedDataSourceId 时, THE NewSessionButton SHALL 始终处于可点击状态

### 需求 3：修复面板拖拽调整大小功能

**用户故事：** 作为用户，我希望能通过拖拽面板之间的分隔条来调整各面板的宽度，以便根据需要自定义界面布局。

#### 验收标准

1. WHEN 用户在左侧面板和中间面板之间的 ResizeHandle 上按下鼠标并拖拽时, THE ResizeHandle SHALL 实时更新左侧面板和中间面板的宽度
2. WHEN 用户在中间面板和右侧面板之间的 ResizeHandle 上按下鼠标并拖拽时, THE ResizeHandle SHALL 实时更新中间面板和右侧面板的宽度
3. WHEN 拖拽调整大小时, THE PanelWidths SHALL 确保每个面板的宽度不低于其最小值约束（左侧 180px，中间 400px，右侧 280px）
4. WHEN 拖拽结束时, THE PanelWidths SHALL 将当前面板宽度持久化到 localStorage
5. WHEN 拖拽进行中时, THE ResizeHandle SHALL 显示视觉反馈（高亮颜色变化和光标样式变化）
6. WHEN 鼠标悬停在 ResizeHandle 上时, THE ResizeHandle SHALL 显示悬停状态的视觉反馈

### 需求 4：清理不再使用的代码

**用户故事：** 作为开发者，我希望移除不再使用的数据源相关代码和属性，以保持代码库整洁。

#### 验收标准

1. WHEN DataSourcesSection 从 Left_Panel 中移除后, THE Left_Panel SHALL 移除所有与数据源相关的 props（onDataSourceSelect、onBrowseData、selectedDataSourceId）
2. WHEN Left_Panel 的 props 简化后, THE App 组件 SHALL 更新 Left_Panel 的调用代码以匹配新的 props 接口
3. WHEN 数据源相关代码被清理后, THE Left_Panel SHALL 移除不再使用的状态变量和事件监听器
