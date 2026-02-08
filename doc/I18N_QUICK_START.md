# 国际化快速开始指南

## 5分钟快速上手

本指南帮助你快速开始使用 VantageData 的国际化系统。

## 前端使用

### 1. 导入 Hook

```typescript
import { useLanguage } from '../i18n';
```

### 2. 在组件中使用

```typescript
function MyComponent() {
    const { t, language } = useLanguage();
    
    return (
        <div>
            <h1>{t('welcome_back')}</h1>
            <button>{t('save')}</button>
            <p>{t('tables_selected', 2, 5)}</p>
        </div>
    );
}
```

### 3. 常用翻译键

```typescript
// 按钮
t('save')           // 保存 / Save
t('cancel')         // 取消 / Cancel
t('confirm')        // 确认 / Confirm
t('delete')         // 删除 / Delete
t('edit')           // 编辑 / Edit

// 状态
t('loading')        // 加载中 / Loading
t('processing')     // 处理中 / Processing
t('success')        // 成功 / Success
t('error')          // 错误 / Error

// 消息
t('operation_success')  // 操作成功 / Operation successful
t('operation_failed')   // 操作失败 / Operation failed
```

## 后端使用

### 1. 导入包

```go
import "vantagedata/i18n"
```

### 2. 使用翻译

```go
// 简单翻译
message := i18n.T("datasource.import_success")

// 带参数的翻译
message := i18n.T("datasource.import_failed", err.Error())

// 在错误中使用
return fmt.Errorf(i18n.T("general.invalid_input"))
```

### 3. 常用翻译键

```go
// 成功消息
i18n.T("general.success")
i18n.T("datasource.import_success")
i18n.T("analysis.export_success")

// 错误消息
i18n.T("general.failed")
i18n.T("datasource.import_failed", err.Error())
i18n.T("analysis.execution_failed")

// 数据库操作
i18n.T("db.connection_failed", err.Error())
i18n.T("db.query_error", err.Error())
```

## 添加新翻译

### 前端 (i18n.ts)

```typescript
export const translations = {
    'English': {
        'my_new_key': 'My new message',
        'my_param_key': 'Value is {0}',
    },
    '简体中文': {
        'my_new_key': '我的新消息',
        'my_param_key': '值为 {0}',
    }
};
```

### 后端 (translations_en.go 和 translations_zh.go)

```go
// translations_en.go
var englishTranslations = map[string]string{
    "myfeature.success": "Operation successful",
    "myfeature.failed":  "Operation failed: %s",
}

// translations_zh.go
var chineseTranslations = map[string]string{
    "myfeature.success": "操作成功",
    "myfeature.failed":  "操作失败：%s",
}
```

## 语言切换

### 前端

语言切换由用户在设置中完成，组件会自动响应：

```typescript
const { language } = useLanguage();
console.log('Current language:', language); // 'English' or '简体中文'
```

### 后端

语言设置在应用启动和配置保存时自动同步：

```go
// 应用启动时
i18n.SyncLanguageFromConfig(&cfg)

// 获取当前语言
currentLang := i18n.GetLanguage()
```

## 常见模式

### 错误处理

**前端**:
```typescript
try {
    await someOperation();
    setToast({ 
        type: 'success', 
        message: t('operation_success') 
    });
} catch (err) {
    setToast({ 
        type: 'error', 
        message: t('operation_failed') + ': ' + err 
    });
}
```

**后端**:
```go
if err := someOperation(); err != nil {
    return fmt.Errorf(i18n.T("operation.failed", err.Error()))
}
```

### 确认对话框

```typescript
const confirmed = await showConfirmDialog({
    title: t('delete_confirmation'),
    message: t('delete_warning', itemName),
    confirmText: t('delete'),
    cancelText: t('cancel')
});
```

### 表单验证

```typescript
if (!name) {
    setError(t('data_source_name_required'));
    return;
}

if (name.length > 50) {
    setError(t('name_too_long', 50));
    return;
}
```

## 命名规范

### 前端

- 使用下划线: `category_action_detail`
- 示例: `datasource_import_success`

### 后端

- 使用点分隔: `category.action`
- 示例: `datasource.import_success`

## 参数化

### 前端

使用 `{0}`, `{1}` 等占位符：

```typescript
// 翻译定义
'tables_selected': '{0} of {1} table(s) selected'

// 使用
t('tables_selected', 2, 5)
// 输出: "2 of 5 table(s) selected"
```

### 后端

使用 `%s`, `%d`, `%v` 等格式化符：

```go
// 翻译定义
"license.sn_deleted": "Successfully deleted %d unused serial numbers"

// 使用
i18n.T("license.sn_deleted", 10)
// 输出: "Successfully deleted 10 unused serial numbers"
```

## 测试

### 切换语言测试

1. 打开应用设置
2. 切换语言（English ↔ 简体中文）
3. 检查所有界面文本是否正确显示
4. 触发各种操作，检查消息是否正确

### 参数化测试

```typescript
// 测试不同参数
console.log(t('tables_selected', 0, 5));  // "0 of 5 table(s) selected"
console.log(t('tables_selected', 1, 5));  // "1 of 5 table(s) selected"
console.log(t('tables_selected', 5, 5));  // "5 of 5 table(s) selected"
```

## 常见问题

### Q: 翻译键不存在会怎样？

A: 系统会返回键名本身，便于调试：

```typescript
t('non_existent_key')  // 返回: "non_existent_key"
```

### Q: 如何处理复数形式？

A: 使用不同的键或条件判断：

```typescript
const key = count === 1 ? 'item_singular' : 'item_plural';
const message = t(key, count);
```

### Q: 如何处理长文本？

A: 将长文本分段，或使用组件化：

```typescript
<p>
    {t('message_part1')} 
    <strong>{t('message_part2')}</strong>
    {t('message_part3')}
</p>
```

### Q: 翻译文件太大怎么办？

A: 按模块拆分，使用动态导入（高级用法）。

## 下一步

- 阅读 [完整迁移指南](./I18N_MIGRATION_GUIDE.md)
- 查看 [迁移示例](./I18N_MIGRATION_EXAMPLE.md)
- 了解 [系统架构](./I18N_OVERVIEW.md)
- 查看 [实施计划](./I18N_IMPLEMENTATION_PLAN.md)

## 获取帮助

如有问题，请查看：
- [常见问题](./I18N_MIGRATION_GUIDE.md#常见问题)
- [最佳实践](./I18N_OVERVIEW.md#最佳实践)
- [故障排除](./I18N_OVERVIEW.md#故障排除)
