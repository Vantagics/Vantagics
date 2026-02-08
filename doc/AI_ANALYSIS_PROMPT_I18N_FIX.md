# AI分析提示词国际化修复

## 问题描述

在用户界面中，AI助手生成的分析建议存在文字渲染问题。这些建议的提示词是硬编码的英文，导致在中文环境下显示不一致。

## 修复内容

### 1. 添加国际化的分析系统提示词

**文件**: `src/i18n/prompts.go`

添加了`analysisSystemPrompts`映射，包含英文和中文两个版本的完整系统提示词：

```go
var analysisSystemPrompts = map[Language]string{
    English: `VantageData Data Analysis Expert. Fast, direct, visualization-first.
    
    🌐 **LANGUAGE RULE (CRITICAL)**: You MUST respond in English...
    
    [完整的英文提示词]
    `,
    
    Chinese: `VantageData 数据分析专家。快速、直接、可视化优先。
    
    🌐 **语言规则（关键）**：你必须用中文回复...
    
    [完整的中文提示词]
    `,
}
```

添加了`GetAnalysisSystemPrompt()`函数：

```go
func GetAnalysisSystemPrompt() string {
    lang := GetLanguage()
    if prompt, ok := analysisSystemPrompts[lang]; ok {
        return prompt
    }
    return analysisSystemPrompts[English]
}
```

### 2. 修改eino.go使用国际化提示词

**文件**: `src/agent/eino.go`

1. 添加了i18n包的导入：
```go
import (
    // ... 其他导入
    "vantagedata/i18n"
)
```

2. 简化了`buildAnalysisSystemPrompt()`函数：
```go
func buildAnalysisSystemPrompt() string {
    // Use internationalized system prompt from i18n package
    return i18n.GetAnalysisSystemPrompt()
}
```

## 提示词内容

系统提示词包含以下关键部分：

1. **语言规则**：明确要求AI使用与用户相同的语言
2. **可视化方法**：ECharts和Python matplotlib两种方式
3. **工具使用规则**：数据分析工作流程
4. **输出格式**：图表、表格、图像的格式要求
5. **分析输出要求**：必须包含图表、洞察和数据摘要
6. **建议输出**：每次分析后提供3-5个后续分析建议

## 影响范围

- AI助手生成的所有分析建议
- 图表标题和标签
- 数据洞察文本
- 后续分析建议

## 测试建议

1. **英文环境测试**：
   - 切换到英文界面
   - 进行数据分析
   - 验证AI建议全部为英文

2. **中文环境测试**：
   - 切换到中文界面
   - 进行数据分析
   - 验证AI建议全部为中文

3. **语言切换测试**：
   - 从英文切换到中文
   - 从中文切换到英文
   - 验证AI建议正确切换

## 相关文档

- [国际化系统概述](./I18N_OVERVIEW.md)
- [国际化实施计划](./I18N_IMPLEMENTATION_PLAN.md)
- [报告国际化分析](./REPORT_I18N_ANALYSIS.md)

---

**修复日期**: 2026-02-08
**状态**: 已完成
