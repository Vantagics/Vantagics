# Vantagics

[![Go Version](https://img.shields.io/badge/Go-1.25.5-blue.svg)](https://golang.org/)
[![Wails](https://img.shields.io/badge/Wails-2.11.0-green.svg)](https://wails.io/)
[![React](https://img.shields.io/badge/React-18-blue.svg)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5-blue.svg)](https://www.typescriptlang.org/)

Vantagics 是一款基于人工智能的桌面数据分析平台。用户通过自然语言对话即可完成数据查询、分析和可视化，无需编写 SQL 或代码。

## ✨ 核心特性

### 🤖 AI 智能分析
- 自然语言交互，无需编写 SQL 或代码
- 支持多种 AI 模型（OpenAI、Claude、Gemini、通义千问、智谱 GLM、DeepSeek、MiniMax）
- 智能意图理解和执行规划
- 多轮对话和上下文记忆管理

### 📊 数据源支持
- **本地文件**：Excel、CSV、JSON
- **数据库**：MySQL、PostgreSQL、SQLite、Snowflake、BigQuery
- **电商平台**：Shopify、BigCommerce、eBay、Etsy
- **项目管理**：Jira Cloud、Jira Server

### 📈 数据可视化
- 折线图、柱状图、饼图、散点图、热力图
- 自动选择最合适的图表类型
- 交互式图表（数据钻取、动态筛选）

### 🎯 分析技能系统
- **群组分析**：用户留存率、留存热力图、高价值用户识别
- **销售漏斗分析**：多阶段转化率、流失点识别、渠道效果对比
- 支持自定义技能扩展

### 📦 分析技能包
- 将分析流程保存为可重放的分析包
- 在不同数据源上快速重现分析
- 支持口令加密保护
- Schema 兼容性验证

### 🛒 市场系统
- 分享和下载分析技能包
- 四种定价模式：免费、按次付费、限时、订阅
- SN + Email 自动认证
- 本地使用权限管理
- Credits 积分系统

### 📄 报告导出
- PDF、Excel、PowerPoint、Word
- CSV、SQL、JSON
- 综合报告生成

### 🌐 国际化
- 支持英文和简体中文
- 实时切换，无需重启
- 前后端全面本地化

## 🚀 快速开始

### 系统要求

- **操作系统**：Windows 10/11、macOS 10.15+、Linux
- **内存**：建议 8GB 以上
- **存储**：至少 500MB 可用空间
- **网络**：需要网络连接以使用 AI 分析功能

### 安装

1. 下载最新版本的安装包
2. 运行安装程序
3. 按照向导完成安装

### 首次启动

1. 选择授权模式：
   - **商业模式**：使用序列号激活，自动配置 AI 模型
   - **开源模式**：自行配置 AI 模型 API 密钥

2. 导入数据源：
   - 本地文件：Excel、CSV、JSON
   - 数据库连接：MySQL、PostgreSQL 等
   - 电商平台：Shopify、BigCommerce 等

3. 开始分析：
   - 创建新会话
   - 用自然语言提问
   - 查看分析结果和可视化

详细使用说明请参考 [快速开始指南](doc/QUICK_START.md)。

## 📚 文档

- [产品功能说明](product.md) - 面向用户的完整功能说明
- [功能特征文档](features.md) - 技术特性和架构概览
- [快速开始指南](doc/QUICK_START.md) - 新手入门教程
- [技术架构](doc/tech.md) - 技术栈和模块说明
- [工作原理](doc/principle.md) - AI Agent 工作原理详解
- [市场系统](doc/MARKETPLACE_SYSTEM.md) - 分析技能包市场文档
- [授权服务器](licenseUsage.md) - License 服务器使用文档
- [服务器信息](server.md) - 部署和运维文档

## 🏗️ 技术架构

### 技术栈

- **后端**：Go 1.25.5
- **桌面框架**：Wails 2.11.0
- **前端框架**：React 18 + TypeScript 5
- **样式方案**：Tailwind CSS
- **数据处理**：Python（pandas、numpy、matplotlib、mlxtend）
- **本地数据库**：SQLite
- **AI 编排**：Eino（字节跳动开源）

### 核心模块

```
Vantagics/
├── src/                    # 后端 Go 代码
│   ├── agent/             # AI Agent 核心逻辑
│   ├── database/          # 数据库服务
│   ├── export/            # 报告导出
│   ├── config/            # 配置管理
│   └── frontend/          # 前端 React 代码
├── tools/                 # 配套工具
│   ├── license_server/    # 授权服务器
│   └── marketplace_server/# 市场服务器
├── skills/                # 分析技能
├── doc/                   # 文档
└── deploy/                # 部署脚本
```

## 🔧 开发

### 环境准备

1. 安装 Go 1.25.5+
2. 安装 Node.js 18+
3. 安装 Wails CLI：
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/yourusername/Vantagics.git
cd Vantagics/src

# 安装前端依赖
cd frontend
npm install
cd ..

# 启动开发服务器
wails dev
```

### 构建

```bash
# Windows
cd src
wails build

# macOS
cd src
wails build -platform darwin/universal

# Linux
cd src
wails build -platform linux/amd64
```

### 部署服务器

```bash
# 部署 License 服务器和 Marketplace 服务器
cd tools
bash deploy_all.sh
```

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 [MIT License](LICENSE) 许可证。

## 🙏 致谢

- [Wails](https://wails.io/) - 优秀的 Go 桌面应用框架
- [Eino](https://github.com/cloudwego/eino) - 字节跳动开源的 AI 编排框架
- [React](https://reactjs.org/) - 强大的前端框架
- [Tailwind CSS](https://tailwindcss.com/) - 实用的 CSS 框架

## 📞 联系我们

- 官网：[https://vantagics.com](https://vantagics.com)
- 邮箱：support@vantagics.com
- 问题反馈：[GitHub Issues](https://github.com/yourusername/Vantagics/issues)

---

**让数据分析更智能、更简单**
