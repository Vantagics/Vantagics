# 国际化迁移示例：event_aggregator.go

本文档展示如何将 `event_aggregator.go` 中的硬编码字符串迁移到国际化系统。

## 原始代码示例

### 1. 恢复建议函数 (getRecoverySuggestions)

**迁移前**:
```go
case ErrorCodeAnalysisError:
    suggestions = append(suggestions, 
        "请检查您的查询是否清晰明确",
        "尝试简化查询条件",
        "如果问题持续，请刷新页面后重试")
```

**迁移后**:
```go
case ErrorCodeAnalysisError:
    suggestions = append(suggestions, 
        i18n.T("error.recovery.check_query"),
        i18n.T("error.recovery.simplify_query"),
        i18n.T("error.recovery.refresh_retry"))
```

### 2. 用户友好消息函数 (getUserFriendlyMessage)

**迁移前**:
```go
switch errorCode {
case ErrorCodeAnalysisError:
    return "分析过程中发生错误"
case ErrorCodeAnalysisTimeout:
    return "分析超时，请稍后重试"
case ErrorCodeAnalysisCancelled:
    return "分析已取消"
// ...
}
```

**迁移后**:
```go
switch errorCode {
case ErrorCodeAnalysisError:
    return i18n.T("error.analysis_error")
case ErrorCodeAnalysisTimeout:
    return i18n.T("error.analysis_timeout")
case ErrorCodeAnalysisCancelled:
    return i18n.T("error.analysis_cancelled")
// ...
}
```

### 3. 超时消息 (EmitTimeout)

**迁移前**:
```go
func (ea *EventAggregator) EmitTimeout(sessionID, requestID string, duration time.Duration) {
    ea.EmitErrorWithDetails(sessionID, requestID, ErrorCodeAnalysisTimeout, 
        fmt.Sprintf("分析超时（已运行 %v）", duration.Round(time.Second)),
        fmt.Sprintf("Analysis timed out after %v", duration.Round(time.Second)))
}
```

**迁移后**:
```go
func (ea *EventAggregator) EmitTimeout(sessionID, requestID string, duration time.Duration) {
    ea.EmitErrorWithDetails(sessionID, requestID, ErrorCodeAnalysisTimeout, 
        i18n.T("error.analysis_timeout_duration", duration.Round(time.Second)),
        fmt.Sprintf("Analysis timed out after %v", duration.Round(time.Second)))
}
```

### 4. 取消消息 (EmitCancelled)

**迁移前**:
```go
func (ea *EventAggregator) EmitCancelled(sessionID, requestID string) {
    errorInfo := createErrorInfo(ErrorCodeAnalysisCancelled, "分析已取消", "")
    // ...
}
```

**迁移后**:
```go
func (ea *EventAggregator) EmitCancelled(sessionID, requestID string) {
    errorInfo := createErrorInfo(ErrorCodeAnalysisCancelled, i18n.T("error.analysis_cancelled"), "")
    // ...
}
```

## 需要添加的翻译键

### 英文翻译 (translations_en.go)

```go
// Error recovery suggestions
"error.recovery.check_query":           "Please check if your query is clear and specific",
"error.recovery.simplify_query":        "Try simplifying the query conditions",
"error.recovery.refresh_retry":         "If the problem persists, please refresh the page and try again",
"error.recovery.reduce_data_range":     "Please try reducing the data range",
"error.recovery.check_network":         "Check if the network connection is stable",
"error.recovery.retry_later":           "Please try again later, the system may be processing other tasks",
"error.recovery.resubmit":              "You can resubmit the analysis request",
"error.recovery.check_data_format":     "Please check if the data format is correct",
"error.recovery.try_different_method":  "Try using a different analysis method",
"error.recovery.contact_support":       "If the problem persists, please contact technical support",
"error.recovery.rephrase_query":        "Please try rephrasing your analysis request",
"error.recovery.use_simpler_query":     "Use simpler query statements",
"error.recovery.check_libraries":       "Required analysis libraries may not be installed",
"error.recovery.check_admin":           "Please contact administrator to check system configuration",
"error.recovery.reduce_batch":          "Try processing data in batches",
"error.recovery.check_datasource":      "Please check if the data source is configured correctly",
"error.recovery.check_table_field":     "Confirm if the table or field name is correct",
"error.recovery.check_deleted":         "Check if the data has been deleted or moved",
"error.recovery.check_data_type":       "Confirm if the data type is correct",
"error.recovery.clean_reimport":        "Try cleaning or reimporting the data",
"error.recovery.adjust_filters":        "Please try adjusting the filter conditions",
"error.recovery.check_data_exists":     "Check if the data source contains the required data",
"error.recovery.add_filters":           "Add more filter conditions",
"error.recovery.consider_pagination":   "Consider pagination or batch queries",
"error.recovery.check_service":         "Confirm if the service is running normally",
"error.recovery.check_permissions":     "You may not have permission to access this resource",
"error.recovery.contact_admin":         "Please contact administrator for appropriate permissions",
"error.recovery.check_account":         "Check your account status",
"error.recovery.resource_busy":         "Resource is being used by other tasks",
"error.recovery.check_path":            "Please check if the resource path is correct",
"error.recovery.confirm_resource":      "Contact administrator to confirm resource status",

// Error messages
"error.analysis_error":                 "An error occurred during analysis",
"error.analysis_timeout":               "Analysis timeout, please try again later",
"error.analysis_timeout_duration":      "Analysis timeout (ran for %v)",
"error.analysis_cancelled":             "Analysis cancelled",
"error.python_execution":               "Code execution failed",
"error.python_syntax":                  "Code syntax error",
"error.python_import":                  "Missing required analysis libraries",
"error.python_memory":                  "Insufficient memory, data may be too large",
"error.data_not_found":                 "Requested data not found",
"error.data_invalid":                   "Invalid data format",
"error.data_empty":                     "Query result is empty",
"error.data_too_large":                 "Data exceeds size limit",
"error.connection_failed":              "Connection failed, please check network",
"error.connection_timeout":             "Connection timeout",
"error.permission_denied":              "Insufficient permissions",
"error.resource_busy":                  "Resource busy, please try again later",
"error.resource_not_found":             "Resource not found",
"error.unknown":                        "Unknown error occurred",
```

### 中文翻译 (translations_zh.go)

```go
// 错误恢复建议
"error.recovery.check_query":           "请检查您的查询是否清晰明确",
"error.recovery.simplify_query":        "尝试简化查询条件",
"error.recovery.refresh_retry":         "如果问题持续，请刷新页面后重试",
"error.recovery.reduce_data_range":     "请尝试简化查询或减少数据范围",
"error.recovery.check_network":         "检查网络连接是否稳定",
"error.recovery.retry_later":           "稍后重试，系统可能正在处理其他任务",
"error.recovery.resubmit":              "您可以重新发起分析请求",
"error.recovery.check_data_format":     "请检查数据格式是否正确",
"error.recovery.try_different_method":  "尝试使用不同的分析方式",
"error.recovery.contact_support":       "如果问题持续，请联系技术支持",
"error.recovery.rephrase_query":        "请尝试重新描述您的分析需求",
"error.recovery.use_simpler_query":     "使用更简单的查询语句",
"error.recovery.check_libraries":       "所需的分析库可能未安装",
"error.recovery.check_admin":           "请联系管理员检查系统配置",
"error.recovery.reduce_batch":          "尝试分批处理数据",
"error.recovery.check_datasource":      "请检查数据源是否已正确配置",
"error.recovery.check_table_field":     "确认查询的表或字段名称是否正确",
"error.recovery.check_deleted":         "检查数据是否已被删除或移动",
"error.recovery.check_data_type":       "确认数据类型是否正确",
"error.recovery.clean_reimport":        "尝试清理或重新导入数据",
"error.recovery.adjust_filters":        "请尝试调整筛选条件",
"error.recovery.check_data_exists":     "检查数据源是否包含所需数据",
"error.recovery.add_filters":           "添加更多筛选条件",
"error.recovery.consider_pagination":   "考虑分页或分批查询",
"error.recovery.check_service":         "确认服务是否正常运行",
"error.recovery.check_permissions":     "您可能没有访问此资源的权限",
"error.recovery.contact_admin":         "请联系管理员获取相应权限",
"error.recovery.check_account":         "检查您的账户状态",
"error.recovery.resource_busy":         "资源正在被其他任务使用",
"error.recovery.check_path":            "请检查资源路径是否正确",
"error.recovery.confirm_resource":      "联系管理员确认资源状态",

// 错误消息
"error.analysis_error":                 "分析过程中发生错误",
"error.analysis_timeout":               "分析超时，请稍后重试",
"error.analysis_timeout_duration":      "分析超时（已运行 %v）",
"error.analysis_cancelled":             "分析已取消",
"error.python_execution":               "代码执行失败",
"error.python_syntax":                  "代码语法错误",
"error.python_import":                  "缺少必要的分析库",
"error.python_memory":                  "内存不足，数据量可能过大",
"error.data_not_found":                 "未找到请求的数据",
"error.data_invalid":                   "数据格式无效",
"error.data_empty":                     "查询结果为空",
"error.data_too_large":                 "数据量超出限制",
"error.connection_failed":              "连接失败，请检查网络",
"error.connection_timeout":             "连接超时",
"error.permission_denied":              "权限不足",
"error.resource_busy":                  "资源繁忙，请稍后重试",
"error.resource_not_found":             "资源未找到",
"error.unknown":                        "发生未知错误",
```

## 完整迁移后的代码

```go
// getRecoverySuggestions returns recovery suggestions based on error code
func getRecoverySuggestions(errorCode string) []string {
    suggestions := make([]string, 0)
    
    switch errorCode {
    case ErrorCodeAnalysisError:
        suggestions = append(suggestions, 
            i18n.T("error.recovery.check_query"),
            i18n.T("error.recovery.simplify_query"),
            i18n.T("error.recovery.refresh_retry"))
    
    case ErrorCodeAnalysisTimeout:
        suggestions = append(suggestions,
            i18n.T("error.recovery.reduce_data_range"),
            i18n.T("error.recovery.check_network"),
            i18n.T("error.recovery.retry_later"))
    
    case ErrorCodeAnalysisCancelled:
        suggestions = append(suggestions,
            i18n.T("error.recovery.resubmit"),
            i18n.T("error.recovery.refresh_retry"))
    
    case ErrorCodePythonExecution:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_data_format"),
            i18n.T("error.recovery.try_different_method"),
            i18n.T("error.recovery.contact_support"))
    
    case ErrorCodePythonSyntax:
        suggestions = append(suggestions,
            i18n.T("error.recovery.rephrase_query"),
            i18n.T("error.recovery.use_simpler_query"))
    
    case ErrorCodePythonImport:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_libraries"),
            i18n.T("error.recovery.check_admin"),
            i18n.T("error.recovery.try_different_method"))
    
    case ErrorCodePythonMemory:
        suggestions = append(suggestions,
            i18n.T("error.recovery.reduce_data_range"),
            i18n.T("error.recovery.reduce_batch"),
            i18n.T("error.recovery.retry_later"))
    
    case ErrorCodeDataNotFound:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_datasource"),
            i18n.T("error.recovery.check_table_field"),
            i18n.T("error.recovery.check_deleted"))
    
    case ErrorCodeDataInvalid:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_data_format"),
            i18n.T("error.recovery.check_data_type"),
            i18n.T("error.recovery.clean_reimport"))
    
    case ErrorCodeDataEmpty:
        suggestions = append(suggestions,
            i18n.T("error.recovery.adjust_filters"),
            i18n.T("error.recovery.check_data_exists"))
    
    case ErrorCodeDataTooLarge:
        suggestions = append(suggestions,
            i18n.T("error.recovery.reduce_data_range"),
            i18n.T("error.recovery.add_filters"),
            i18n.T("error.recovery.consider_pagination"))
    
    case ErrorCodeConnectionFailed:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_network"),
            i18n.T("error.recovery.check_service"),
            i18n.T("error.recovery.retry_later"))
    
    case ErrorCodeConnectionTimeout:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_network"),
            i18n.T("error.recovery.retry_later"),
            i18n.T("error.recovery.contact_support"))
    
    case ErrorCodePermissionDenied:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_permissions"),
            i18n.T("error.recovery.contact_admin"),
            i18n.T("error.recovery.check_account"))
    
    case ErrorCodeResourceBusy:
        suggestions = append(suggestions,
            i18n.T("error.recovery.resource_busy"),
            i18n.T("error.recovery.retry_later"),
            i18n.T("error.recovery.contact_support"))
    
    case ErrorCodeResourceNotFound:
        suggestions = append(suggestions,
            i18n.T("error.recovery.check_path"),
            i18n.T("error.recovery.check_deleted"),
            i18n.T("error.recovery.confirm_resource"))
    
    default:
        suggestions = append(suggestions,
            i18n.T("error.recovery.retry_later"),
            i18n.T("error.recovery.contact_support"))
    }
    
    return suggestions
}

// getUserFriendlyMessage returns a user-friendly message based on error code
func getUserFriendlyMessage(errorCode, originalMessage string) string {
    // If original message is already user-friendly, use it
    if originalMessage != "" && len([]rune(originalMessage)) > 0 {
        // Check if it's already a localized message (contains non-ASCII)
        for _, r := range originalMessage {
            if r > 127 {
                return originalMessage
            }
        }
    }
    
    // Generate user-friendly message based on error code
    switch errorCode {
    case ErrorCodeAnalysisError:
        return i18n.T("error.analysis_error")
    case ErrorCodeAnalysisTimeout:
        return i18n.T("error.analysis_timeout")
    case ErrorCodeAnalysisCancelled:
        return i18n.T("error.analysis_cancelled")
    case ErrorCodePythonExecution:
        return i18n.T("error.python_execution")
    case ErrorCodePythonSyntax:
        return i18n.T("error.python_syntax")
    case ErrorCodePythonImport:
        return i18n.T("error.python_import")
    case ErrorCodePythonMemory:
        return i18n.T("error.python_memory")
    case ErrorCodeDataNotFound:
        return i18n.T("error.data_not_found")
    case ErrorCodeDataInvalid:
        return i18n.T("error.data_invalid")
    case ErrorCodeDataEmpty:
        return i18n.T("error.data_empty")
    case ErrorCodeDataTooLarge:
        return i18n.T("error.data_too_large")
    case ErrorCodeConnectionFailed:
        return i18n.T("error.connection_failed")
    case ErrorCodeConnectionTimeout:
        return i18n.T("error.connection_timeout")
    case ErrorCodePermissionDenied:
        return i18n.T("error.permission_denied")
    case ErrorCodeResourceBusy:
        return i18n.T("error.resource_busy")
    case ErrorCodeResourceNotFound:
        return i18n.T("error.resource_not_found")
    default:
        if originalMessage != "" {
            return originalMessage
        }
        return i18n.T("error.unknown")
    }
}

// EmitTimeout emits a timeout error event with recovery suggestions
func (ea *EventAggregator) EmitTimeout(sessionID, requestID string, duration time.Duration) {
    ea.EmitErrorWithDetails(sessionID, requestID, ErrorCodeAnalysisTimeout, 
        i18n.T("error.analysis_timeout_duration", duration.Round(time.Second)),
        fmt.Sprintf("Analysis timed out after %v", duration.Round(time.Second)))
}

// EmitCancelled emits a cancellation event with recovery suggestions
func (ea *EventAggregator) EmitCancelled(sessionID, requestID string) {
    // Create error info for cancellation
    errorInfo := createErrorInfo(ErrorCodeAnalysisCancelled, i18n.T("error.analysis_cancelled"), "")
    
    runtime.EventsEmit(ea.ctx, "analysis-cancelled", map[string]interface{}{
        "sessionId":           sessionID,
        "threadId":            sessionID,
        "requestId":           requestID,
        "code":                errorInfo.Code,
        "message":             errorInfo.Message,
        "recoverySuggestions": errorInfo.RecoverySuggestions,
        "timestamp":           errorInfo.Timestamp,
    })
}
```

## 迁移步骤总结

1. **导入 i18n 包**: 在文件顶部添加 `"vantagedata/i18n"`
2. **识别硬编码字符串**: 查找所有中文字符串和英文消息
3. **添加翻译键**: 在 `translations_en.go` 和 `translations_zh.go` 中添加对应的翻译
4. **替换硬编码字符串**: 使用 `i18n.T("key")` 替换硬编码字符串
5. **测试**: 切换语言测试所有消息是否正确显示

## 注意事项

1. **保持键名一致**: 使用有意义的键名，便于维护
2. **参数化消息**: 对于包含动态内容的消息，使用 `%v`、`%s` 等占位符
3. **回退机制**: 如果翻译键不存在，系统会返回键名本身
4. **测试两种语言**: 确保英文和中文翻译都正确
5. **日志记录**: 保留技术性的英文日志，用户消息使用翻译

## 下一步

完成 `event_aggregator.go` 的迁移后，可以按照相同的模式迁移其他文件：

1. `src/agent/result_parser.go` - 结果解析错误消息
2. `tools/license_server/main.go` - 授权服务器消息
3. `src/app*.go` - 应用主程序消息
4. `src/export/*.go` - 导出服务消息
