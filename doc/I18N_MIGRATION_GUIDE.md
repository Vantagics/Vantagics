# 多语言国际化迁移指南

## 概述

本指南说明如何将现有的硬编码字符串迁移到国际化系统。

## 架构

### 前端 (TypeScript/React)

- **位置**: `src/frontend/src/i18n.ts`
- **Hook**: `useLanguage()`
- **函数**: `t(key, ...params)`
- **支持语言**: English, 简体中文

### 后端 (Go)

- **位置**: `src/i18n/`
- **包**: `vantagedata/i18n`
- **函数**: `i18n.T(key, ...params)`
- **支持语言**: English, Chinese

## 前端迁移步骤

### 1. 导入 Hook

```typescript
import { useLanguage } from '../i18n';

function MyComponent() {
    const { t } = useLanguage();
    // ...
}
```

### 2. 替换硬编码字符串

**迁移前:**
```typescript
setError('未选择文件或安装失败');
alert('Delete failed: ' + err);
```

**迁移后:**
```typescript
setError(t('no_file_selected'));
alert(t('delete_failed') + ': ' + err);
```

### 3. 使用参数化翻译

**迁移前:**
```typescript
setMessage({ 
    type: 'success', 
    text: `成功安装 ${installed.length} 个Skills: ${installed.join(', ')}` 
});
```

**迁移后:**
```typescript
setMessage({ 
    type: 'success', 
    text: t('skills_install_success_count', installed.length, installed.join(', '))
});
```

### 4. 添加新的翻译键

在 `src/frontend/src/i18n.ts` 中添加：

```typescript
export const translations: Record<Language, Record<string, string>> = {
    'English': {
        // ... existing translations
        'my_new_key': 'My new message',
        'my_param_key': 'Value is {0}',
    },
    '简体中文': {
        // ... existing translations
        'my_new_key': '我的新消息',
        'my_param_key': '值为 {0}',
    }
};
```

## 后端迁移步骤

### 1. 导入包

```go
import "vantagedata/i18n"
```

### 2. 替换硬编码字符串

**迁移前:**
```go
return fmt.Errorf("SMTP configuration incomplete")
return fmt.Errorf("成功删除 %d 个未使用的序列号", deleted)
```

**迁移后:**
```go
return fmt.Errorf(i18n.T("license.smtp_incomplete"))
return fmt.Errorf(i18n.T("license.sn_deleted", deleted))
```

### 3. 在 HTTP 响应中使用

**迁移前:**
```go
json.NewEncoder(w).Encode(map[string]interface{}{
    "error": "此分组中还有序列号，无法删除",
})
```

**迁移后:**
```go
json.NewEncoder(w).Encode(map[string]interface{}{
    "error": i18n.T("license.group_has_sn", count),
})
```

### 4. 同步语言设置

在应用启动时或配置更改时：

```go
import (
    "vantagedata/config"
    "vantagedata/i18n"
)

func initializeApp() {
    cfg := config.LoadConfig()
    i18n.SyncLanguageFromConfig(cfg)
}
```

### 5. 添加新的翻译键

在 `src/i18n/translations_en.go` 中：
```go
var englishTranslations = map[string]string{
    // ... existing translations
    "myfeature.success": "Operation successful",
    "myfeature.failed":  "Operation failed: %s",
}
```

在 `src/i18n/translations_zh.go` 中：
```go
var chineseTranslations = map[string]string{
    // ... existing translations
    "myfeature.success": "操作成功",
    "myfeature.failed":  "操作失败：%s",
}
```

## 常见模式

### 错误消息

**前端:**
```typescript
try {
    await someOperation();
    setToast({ type: 'success', message: t('operation_success') });
} catch (err) {
    setToast({ type: 'error', message: t('operation_failed') + ': ' + err });
}
```

**后端:**
```go
if err := someOperation(); err != nil {
    return fmt.Errorf(i18n.T("operation.failed", err.Error()))
}
return nil
```

### 确认对话框

**前端:**
```typescript
const confirmed = await showConfirmDialog({
    title: t('delete_confirmation'),
    message: t('delete_warning', itemName),
    confirmText: t('delete'),
    cancelText: t('cancel')
});
```

### 状态消息

**前端:**
```typescript
const statusMessages = {
    idle: t('status_idle'),
    loading: t('status_loading'),
    processing: t('status_processing'),
    complete: t('status_complete'),
    error: t('status_error')
};
```

## 翻译键命名规范

### 前端

- 使用下划线分隔: `my_translation_key`
- 按功能分组: `datasource_*`, `analysis_*`, `export_*`
- 参数使用 `{0}`, `{1}` 等占位符

### 后端

- 使用点分隔: `category.action`
- 按模块分组: `license.*`, `datasource.*`, `analysis.*`
- 参数使用 `%s`, `%d` 等格式化符

## 优先级

### 高优先级（用户直接可见）

1. 错误消息和警告
2. 成功提示
3. 按钮和标签文本
4. 对话框标题和内容
5. 表单验证消息

### 中优先级（间接可见）

1. 日志消息（如果显示给用户）
2. 工具提示
3. 占位符文本
4. 帮助文本

### 低优先级（内部使用）

1. 调试日志
2. 开发者消息
3. 内部错误代码

## 测试清单

- [ ] 所有用户可见的字符串都已翻译
- [ ] 两种语言的翻译都已添加
- [ ] 参数化消息正确工作
- [ ] 语言切换功能正常
- [ ] 前后端语言设置同步
- [ ] 错误消息在两种语言下都清晰易懂
- [ ] 没有遗漏的硬编码字符串

## 工具和脚本

### 查找硬编码字符串

**前端 (TypeScript):**
```bash
# 查找可能的硬编码中文字符串
grep -r "[\u4e00-\u9fa5]" src/frontend/src/components --include="*.tsx" --include="*.ts"

# 查找可能的硬编码英文字符串（在字符串字面量中）
grep -r "setError\|setMessage\|alert\|confirm" src/frontend/src/components --include="*.tsx"
```

**后端 (Go):**
```bash
# 查找可能的硬编码中文字符串
grep -r "[\u4e00-\u9fa5]" src --include="*.go"

# 查找 fmt.Errorf 和 fmt.Sprintf 调用
grep -r "fmt\.Errorf\|fmt\.Sprintf" src --include="*.go"
```

### 验证翻译完整性

创建脚本检查所有翻译键在两种语言中都存在：

```typescript
// check-translations.ts
import { translations } from './src/frontend/src/i18n';

const englishKeys = Object.keys(translations['English']);
const chineseKeys = Object.keys(translations['简体中文']);

const missingInChinese = englishKeys.filter(k => !chineseKeys.includes(k));
const missingInEnglish = chineseKeys.filter(k => !englishKeys.includes(k));

if (missingInChinese.length > 0) {
    console.log('Missing in Chinese:', missingInChinese);
}
if (missingInEnglish.length > 0) {
    console.log('Missing in English:', missingInEnglish);
}
```

## 最佳实践

1. **始终使用翻译键**: 不要在代码中硬编码用户可见的字符串
2. **保持键名一致**: 前后端使用相似的键名便于维护
3. **参数化动态内容**: 使用占位符而不是字符串拼接
4. **及时更新翻译**: 添加新功能时同时添加翻译
5. **测试两种语言**: 确保两种语言的翻译都正确且完整
6. **使用有意义的键名**: 键名应该清晰表达其用途
7. **分组管理**: 按功能模块组织翻译键
8. **文档化特殊情况**: 对于复杂的翻译逻辑添加注释

## 常见问题

### Q: 如何处理复数形式？

**A:** 目前系统不支持自动复数处理，需要使用不同的键：

```typescript
// 英文
'item_count_singular': '{0} item',
'item_count_plural': '{0} items',

// 使用
const key = count === 1 ? 'item_count_singular' : 'item_count_plural';
const message = t(key, count);
```

### Q: 如何处理性别相关的翻译？

**A:** 使用中性表达或提供多个键：

```typescript
'welcome_message': 'Welcome, {0}!', // 中性
```

### Q: 翻译文件太大怎么办？

**A:** 考虑按模块拆分翻译文件，使用动态导入：

```typescript
// 延迟加载特定模块的翻译
const moduleTranslations = await import('./translations/module-name');
```

### Q: 如何处理 HTML 内容？

**A:** 使用 React 的 `dangerouslySetInnerHTML` 或组件化：

```typescript
// 方法1: 纯文本
<p>{t('my_message')}</p>

// 方法2: 组件化
<p>
    {t('message_part1')} 
    <strong>{t('message_part2')}</strong>
    {t('message_part3')}
</p>
```

## 参考资源

- [React i18n 最佳实践](https://react.i18next.com/)
- [Go 国际化库](https://github.com/nicksnyder/go-i18n)
- [Unicode CLDR](http://cldr.unicode.org/)
