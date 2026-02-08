# VantageData 多语言国际化系统

## 概述

VantageData 实现了完整的多语言国际化支持，覆盖前端用户界面和后端服务消息。系统支持英文 (English) 和简体中文两种语言，并提供了灵活的扩展机制以支持更多语言。

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        用户界面                              │
│                    (React/TypeScript)                        │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  useLanguage() Hook                                   │  │
│  │  - t(key, ...params)                                  │  │
│  │  - language: Language                                 │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                  │
│                           ▼                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  i18n.ts                                              │  │
│  │  - translations: Record<Language, Record<string>>     │  │
│  │  - 1000+ 翻译键                                        │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ HTTP/WebSocket
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      后端服务 (Go)                           │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  i18n Package                                         │  │
│  │  - T(key, ...params)                                  │  │
│  │  - SetLanguage(lang)                                  │  │
│  │  - SyncLanguageFromConfig(cfg)                        │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                  │
│                           ▼                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Translation Maps                                     │  │
│  │  - englishTranslations: map[string]string             │  │
│  │  - chineseTranslations: map[string]string             │  │
│  │  - 150+ 翻译键                                         │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Config File  │
                    │  language: XX │
                    └──────────────┘
```

## 核心特性

### 1. 双语支持
- **英文 (English)**: 默认语言，适用于国际用户
- **简体中文**: 完整的中文本地化支持

### 2. 参数化翻译
支持动态内容插入，避免字符串拼接：

**前端**:
```typescript
t('skills_install_success_count', 5, 'skill1, skill2')
// 输出: "Successfully installed 5 Skills: skill1, skill2"
```

**后端**:
```go
i18n.T("license.sn_deleted", 10)
// 输出: "Successfully deleted 10 unused serial numbers"
```

### 3. 实时语言切换
用户可以在设置中切换语言，无需重启应用：

```typescript
// 前端自动响应语言变化
const { language, t } = useLanguage();

// 后端同步语言设置
i18n.SyncLanguageFromConfig(config)
```

### 4. 线程安全
后端使用读写锁保证并发安全：

```go
type Translator struct {
    language     Language
    translations map[Language]map[string]string
    mu           sync.RWMutex  // 读写锁
}
```

### 5. 统一的错误消息
前后端使用一致的错误代码和消息：

```typescript
// 前端
'license_error_INVALID_SN': 'Invalid serial number'

// 后端
"license.invalid_sn": "Invalid serial number"
```

## 文件结构

```
vantagedata/
├── src/
│   ├── i18n/                          # 后端国际化包
│   │   ├── i18n.go                    # 核心翻译器
│   │   ├── translations_en.go         # 英文翻译
│   │   ├── translations_zh.go         # 中文翻译
│   │   ├── sync.go                    # 配置同步
│   │   └── README.md                  # 使用文档
│   │
│   └── frontend/
│       └── src/
│           ├── i18n.ts                # 前端国际化
│           └── i18n_additions.ts      # 额外翻译
│
├── doc/
│   ├── I18N_OVERVIEW.md               # 系统概述 (本文档)
│   ├── I18N_MIGRATION_GUIDE.md        # 迁移指南
│   └── I18N_IMPLEMENTATION_PLAN.md    # 实施计划
│
└── tools/
    ├── find_hardcoded_strings.sh      # 查找脚本 (Linux/Mac)
    └── find_hardcoded_strings.bat     # 查找脚本 (Windows)
```

## 翻译覆盖范围

### 前端 (1000+ 翻译键)

#### 核心功能
- 应用菜单和导航
- 数据源管理
- 聊天和分析
- 仪表盘和可视化
- 设置和配置

#### 用户交互
- 按钮和标签
- 对话框和提示
- 表单验证
- 错误和成功消息
- 工具提示和帮助文本

#### 特定功能
- 授权和激活
- Python 环境管理
- Skills 管理
- MCP 服务配置
- 搜索 API 配置
- 导出功能

### 后端 (150+ 翻译键)

#### 服务消息
- 授权服务器 (30+ 条)
- 数据源操作 (15+ 条)
- 分析引擎 (15+ 条)
- 文件操作 (10+ 条)
- 数据库操作 (12+ 条)

#### 功能模块
- Skills 管理 (10+ 条)
- Python 环境 (7+ 条)
- 配置管理 (5+ 条)
- 认证授权 (6+ 条)
- 导出服务 (10+ 条)

#### 通用消息
- 成功/失败消息
- 错误处理
- 状态提示
- 验证消息

## 使用示例

### 前端使用

#### 基本翻译
```typescript
import { useLanguage } from '../i18n';

function MyComponent() {
    const { t } = useLanguage();
    
    return (
        <div>
            <h1>{t('welcome_back')}</h1>
            <button>{t('start_new_analysis')}</button>
        </div>
    );
}
```

#### 参数化翻译
```typescript
const { t } = useLanguage();

// 单个参数
const message = t('delete_data_source_message', dataSourceName);

// 多个参数
const status = t('tables_selected', selectedCount, totalCount);
```

#### 错误处理
```typescript
try {
    await importDataSource();
    setToast({ 
        type: 'success', 
        message: t('datasource_import_success') 
    });
} catch (err) {
    setToast({ 
        type: 'error', 
        message: t('datasource_import_failed', err.message) 
    });
}
```

### 后端使用

#### 基本翻译
```go
import "vantagedata/i18n"

func handleRequest() error {
    if err := validateInput(); err != nil {
        return fmt.Errorf(i18n.T("general.invalid_input"))
    }
    return nil
}
```

#### 参数化翻译
```go
// 单个参数
message := i18n.T("datasource.connection_failed", err.Error())

// 多个参数
message := i18n.T("license.sn_deleted", deletedCount)
```

#### HTTP 响应
```go
func handleImport(w http.ResponseWriter, r *http.Request) {
    // 从配置同步语言
    i18n.SyncLanguageFromConfig(config)
    
    if err := importData(); err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": i18n.T("datasource.import_failed", err.Error()),
        })
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": i18n.T("datasource.import_success"),
    })
}
```

## 语言切换流程

```
用户在设置中选择语言
        │
        ▼
前端更新配置
        │
        ▼
保存到配置文件
        │
        ▼
触发 config-updated 事件
        │
        ├─────────────────┬─────────────────┐
        ▼                 ▼                 ▼
   前端 Hook          后端 API         其他组件
   重新渲染          同步语言          更新显示
```

## 添加新语言

### 1. 定义语言类型

**前端** (`i18n.ts`):
```typescript
export type Language = 'English' | '简体中文' | '日本語';
```

**后端** (`i18n.go`):
```go
const (
    English  Language = "English"
    Chinese  Language = "简体中文"
    Japanese Language = "日本語"
)
```

### 2. 添加翻译

**前端**:
```typescript
export const translations: Record<Language, Record<string, string>> = {
    'English': { /* ... */ },
    '简体中文': { /* ... */ },
    '日本語': {
        'welcome_back': 'おかえりなさい',
        // ...
    }
};
```

**后端**:
```go
// translations_ja.go
var japaneseTranslations = map[string]string{
    "general.success": "成功",
    // ...
}

// i18n.go
func (t *Translator) loadTranslations() {
    t.translations[English] = englishTranslations
    t.translations[Chinese] = chineseTranslations
    t.translations[Japanese] = japaneseTranslations
}
```

### 3. 更新配置

在配置文件中添加新语言选项。

## 性能考虑

### 前端
- 翻译在组件挂载时加载
- 使用 React Hook 避免重复渲染
- 翻译键查找时间复杂度 O(1)

### 后端
- 翻译在启动时加载到内存
- 使用单例模式避免重复初始化
- 读写锁保证并发安全
- 查找时间复杂度 O(1)

### 优化建议
1. 避免在循环中频繁调用翻译函数
2. 缓存常用的翻译结果
3. 使用参数化翻译而不是字符串拼接
4. 按需加载大型翻译文件

## 测试策略

### 单元测试
```typescript
// 前端
describe('i18n', () => {
    it('should translate keys correctly', () => {
        const { t } = useLanguage();
        expect(t('welcome_back')).toBe('Welcome back!');
    });
    
    it('should handle parameters', () => {
        const { t } = useLanguage();
        expect(t('tables_selected', 2, 5)).toBe('2 of 5 table(s) selected');
    });
});
```

```go
// 后端
func TestTranslation(t *testing.T) {
    i18n.SetLanguage(i18n.English)
    result := i18n.T("general.success")
    assert.Equal(t, "Operation successful", result)
}
```

### 集成测试
- 测试语言切换功能
- 测试前后端语言同步
- 测试所有翻译键存在
- 测试参数化翻译

### 手动测试
- 切换语言检查所有界面
- 验证错误消息显示
- 检查翻译质量和一致性
- 测试边界情况

## 最佳实践

### 开发规范
1. **始终使用翻译键**: 不要硬编码用户可见的字符串
2. **保持键名一致**: 前后端使用相似的命名
3. **参数化动态内容**: 使用占位符而不是拼接
4. **及时添加翻译**: 新功能开发时同步添加
5. **测试两种语言**: 确保翻译完整且正确

### 命名规范
- 前端: `category_action_detail` (下划线分隔)
- 后端: `category.action` (点分隔)
- 使用有意义的名称
- 按功能模块分组

### 翻译质量
- 由母语者审查翻译
- 保持术语一致性
- 考虑文化差异
- 避免直译
- 使用自然的表达

## 故障排除

### 问题: 翻译键未找到
**症状**: 显示翻译键而不是翻译文本
**解决**: 
1. 检查键名拼写
2. 确认翻译已添加到两种语言
3. 重启应用重新加载翻译

### 问题: 语言切换不生效
**症状**: 切换语言后界面未更新
**解决**:
1. 检查配置是否正确保存
2. 确认事件监听器已注册
3. 检查组件是否使用 `useLanguage` Hook

### 问题: 参数化翻译显示错误
**症状**: 占位符未被替换
**解决**:
1. 检查参数数量是否匹配
2. 确认占位符格式正确 (`{0}` 或 `%s`)
3. 验证参数类型

### 问题: 性能下降
**症状**: 翻译查找缓慢
**解决**:
1. 检查是否在循环中频繁调用
2. 考虑缓存翻译结果
3. 优化翻译文件结构

## 维护和更新

### 日常维护
- 新功能开发时添加翻译
- 定期审查翻译质量
- 收集用户反馈
- 修复翻译错误

### 定期审查
- 每季度检查翻译完整性
- 每半年审查翻译质量
- 每年评估新语言需求

### 工具支持
- 使用自动化脚本查找硬编码字符串
- 维护翻译管理工具
- 改进开发者体验

## 相关资源

### 文档
- [迁移指南](./I18N_MIGRATION_GUIDE.md)
- [实施计划](./I18N_IMPLEMENTATION_PLAN.md)
- [后端使用指南](../src/i18n/README.md)

### 工具
- `tools/find_hardcoded_strings.sh` - 查找硬编码字符串
- `tools/find_hardcoded_strings.bat` - Windows 版本

### 外部资源
- [React i18n 最佳实践](https://react.i18next.com/)
- [Go 国际化](https://github.com/nicksnyder/go-i18n)
- [Unicode CLDR](http://cldr.unicode.org/)

## 贡献指南

### 添加翻译
1. 在对应的翻译文件中添加键值对
2. 确保两种语言都有翻译
3. 遵循命名规范
4. 提交 Pull Request

### 报告问题
1. 描述问题和重现步骤
2. 提供相关代码片段
3. 说明预期行为
4. 附上截图（如适用）

### 改进建议
1. 描述改进目标
2. 说明实施方案
3. 评估影响范围
4. 讨论可行性

## 许可证

本国际化系统遵循 VantageData 项目的许可证。

## 联系方式

如有问题或建议，请联系：
- 技术支持: [待填写]
- 翻译团队: [待填写]
- 项目负责人: [待填写]
