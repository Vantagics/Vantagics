# 后端国际化 (i18n) 使用指南

## 概述

本包提供了后端 Go 代码的国际化支持，支持英文和简体中文两种语言。

## 使用方法

### 1. 基本使用

```go
import "vantagics/i18n"

// 简单翻译
message := i18n.T("datasource.import_success")

// 带参数的翻译
message := i18n.T("datasource.import_failed", err.Error())
```

### 2. 设置语言

```go
// 设置为中文
i18n.SetLanguage(i18n.Chinese)

// 设置为英文
i18n.SetLanguage(i18n.English)

// 获取当前语言
currentLang := i18n.GetLanguage()
```

### 3. 在 HTTP 响应中使用

```go
func handleImport(w http.ResponseWriter, r *http.Request) {
    // 从配置或请求头获取语言设置
    lang := getLanguageFromConfig() // 或从 r.Header.Get("Accept-Language")
    i18n.SetLanguage(lang)
    
    err := importDataSource()
    if err != nil {
        response := map[string]interface{}{
            "error": i18n.T("datasource.import_failed", err.Error()),
        }
        json.NewEncoder(w).Encode(response)
        return
    }
    
    response := map[string]interface{}{
        "message": i18n.T("datasource.import_success"),
    }
    json.NewEncoder(w).Encode(response)
}
```

### 4. 在错误处理中使用

```go
func processData() error {
    if data == nil {
        return fmt.Errorf(i18n.T("general.invalid_input"))
    }
    
    if err := validateData(data); err != nil {
        return fmt.Errorf(i18n.T("datasource.connection_failed", err.Error()))
    }
    
    return nil
}
```

## 翻译键命名规范

翻译键采用分类命名方式：

- `license.*` - 授权相关
- `datasource.*` - 数据源操作
- `analysis.*` - 分析操作
- `file.*` - 文件操作
- `db.*` - 数据库操作
- `skills.*` - Skills管理
- `python.*` - Python环境
- `config.*` - 配置
- `auth.*` - 认证授权
- `general.*` - 通用消息
- `export.*` - 导出操作
- `mcp.*` - MCP服务
- `search.*` - 搜索API
- `session.*` - 会话管理
- `table.*` - 表操作
- `dashboard.*` - 仪表盘操作

## 添加新的翻译

### 1. 在 `translations_en.go` 中添加英文翻译：

```go
var englishTranslations = map[string]string{
    // ... existing translations
    "myfeature.success": "Operation completed successfully",
    "myfeature.failed":  "Operation failed: %s",
}
```

### 2. 在 `translations_zh.go` 中添加中文翻译：

```go
var chineseTranslations = map[string]string{
    // ... existing translations
    "myfeature.success": "操作成功完成",
    "myfeature.failed":  "操作失败：%s",
}
```

## 与前端同步

前端使用 TypeScript 的 i18n 系统，位于 `src/frontend/src/i18n.ts`。

为了保持前后端一致性：

1. 后端错误消息应使用翻译键返回给前端
2. 前端根据用户语言设置显示对应的翻译
3. 错误代码可以使用统一的键名（如 `INVALID_SN`），前端通过 `license_error_INVALID_SN` 查找翻译

## 最佳实践

1. **始终使用翻译键**：不要在代码中硬编码用户可见的字符串
2. **参数化消息**：使用 `%s`、`%d` 等占位符传递动态内容
3. **保持键名一致**：前后端使用相同或相似的键名
4. **及时更新**：添加新功能时同时添加翻译
5. **测试两种语言**：确保两种语言的翻译都正确且完整

## 线程安全

`Translator` 使用读写锁保证线程安全，可以在并发环境中安全使用。

## 性能考虑

- 翻译在启动时加载到内存，查找速度快
- 使用单例模式，避免重复初始化
- 读操作使用读锁，不会阻塞其他读操作
