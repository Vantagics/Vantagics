# Skill 分析保护机制

## 概述

为了防止在分析进行中修改 Skills 导致 Agent 执行出错，系统实现了分析状态保护机制�?

## 功能说明

当有分析任务正在进行时，系统会阻止以下操作：

1. **禁用 Skill** - 防止正在使用�?Skill 被禁�?
2. **启用 Skill** - 防止在分析过程中改变可用 Skill 集合

## 实现细节

### 后端保护 (Go)

�?`src/app.go` 中的 `EnableSkill` �?`DisableSkill` 方法中添加了分析状态检查：

```go
// Check if analysis is in progress
a.cancelAnalysisMutex.Lock()
isGenerating := a.isChatGenerating
a.cancelAnalysisMutex.Unlock()

if isGenerating {
    return fmt.Errorf("cannot enable/disable skill while analysis is in progress")
}
```

### 前端提示 (TypeScript)

�?`src/frontend/src/components/SkillsManagementPage.tsx` 中添加了友好的错误提示：

```typescript
// Check if error is due to analysis in progress
if (errorMsg.includes('analysis is in progress') || errorMsg.includes('分析正在进行')) {
    setMessage({ 
        type: 'error', 
        text: `无法修改 Skill 状态：当前有分析任务正在进行中。请等待分析完成后再试。` 
    });
}
```

## 用户体验

1. **正常情况**：用户可以自由启�?禁用 Skills
2. **分析进行�?*�?
   - 用户尝试切换 Skill 状态时会收到明确的错误提示
   - 提示信息说明原因并建议等待分析完�?
   - 不会导致系统崩溃或数据不一�?

## 技术要�?

### 状态标�?

- `isChatGenerating`: 布尔标志，表示是否有分析正在进行
- `cancelAnalysisMutex`: 互斥锁，保护并发访问

### 保护范围

- �?启用 Skill (`EnableSkill`)
- �?禁用 Skill (`DisableSkill`)
- ℹ️ 查看 Skills (`ListSkills`) - 不受限制
- ℹ️ 安装�?Skills (`InstallSkillsFromZip`) - 不受限制（新 Skill 默认不会立即被使用）

## 测试

测试文件：`src/agent/skill_service_analysis_protection_test.go`

测试覆盖�?
- �?基本的启�?禁用操作
- �?不存在的 Skill 错误处理
- �?获取已启�?Skills 列表

运行测试�?
```bash
cd src
go test -v -run TestSkillService vantagedata/agent
```

## 相关文件

- `src/app.go` - 主应用逻辑，包含分析状态检�?
- `src/agent/skill_service.go` - Skill 服务实现
- `src/frontend/src/components/SkillsManagementPage.tsx` - 前端 Skill 管理界面
- `src/agent/skill_service_analysis_protection_test.go` - 单元测试

## 未来改进

1. **更细粒度的保�?*�?
   - 可以考虑只保护正在使用的 Skills
   - 允许修改未被当前分析使用�?Skills

2. **状态指�?*�?
   - �?UI 中显示哪�?Skills 正在被使�?
   - 提供更详细的分析状态信�?

3. **队列机制**�?
   - 允许用户预约在分析完成后执行的操�?
   - 自动在分析完成后应用待处理的更改
