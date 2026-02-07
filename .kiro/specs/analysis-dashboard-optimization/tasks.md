# Implementation Plan: Analysis Dashboard Optimization

## Overview

本实现计划将分析逻辑与仪表盘数据显示优化功能分解为可执行的编码任务。主要改进包括：增强分析提示词以优先生成可视化结果、优化数据传递链路确保完整性、以及修改仪表盘默认布局为全宽显示。

## Tasks

- [x] 1. 增强 AnalysisPromptBuilder 的可视化指令
  - [x] 1.1 修改 BuildPromptWithHints 方法，添加更强的可视化强调
    - 在"分析要求"部分添加⭐⭐⭐标记
    - 添加图表类型推荐逻辑
    - 添加完整的代码示例
    - _Requirements: 1.1, 1.5, 1.6, 2.1, 2.3, 2.4_
  
  - [x] 1.2 添加无分类提示时的默认可视化鼓励
    - 当 hints 为 nil 时，仍然添加可视化建议
    - _Requirements: 2.5_
  
  - [x] 1.3 编写 AnalysisPromptBuilder 属性测试
    - **Property 1: Prompt Content Completeness**
    - **Property 2: Classification Hints Affect Prompt**
    - **Validates: Requirements 1.1, 1.5, 1.6, 2.1, 2.2, 2.3, 2.4, 2.5**

- [x] 2. Checkpoint - 确保提示词增强测试通过
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. 优化 EventAggregator 数据验证和日志
  - [x] 3.1 增强 ValidateItem 方法的验证日志
    - 添加更详细的数据类型检查
    - 记录所有添加的数据项
    - _Requirements: 3.1, 3.2, 3.3, 3.5_
  
  - [x] 3.2 优化 AddItem 方法的数据完整性检查
    - 确保空 ID 时记录警告但继续处理
    - _Requirements: 3.5_
  
  - [x] 3.3 编写 EventAggregator 属性测试
    - **Property 3: EventAggregator Data Capture**
    - **Property 5: Graceful Degradation on Empty IDs**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.5**

- [x] 4. 优化响应解析逻辑
  - [x] 4.1 确保解析器提取所有 json:echarts 代码块
    - 使用 FindAllStringSubmatch 替代 FindStringSubmatch
    - _Requirements: 4.1, 4.6_
  
  - [x] 4.2 确保解析器提取所有 base64 图片
    - 使用 FindAllStringSubmatch 替代 FindStringSubmatch
    - _Requirements: 4.2, 4.6_
  
  - [x] 4.3 确保解析器提取所有 json:table 代码块
    - 使用 FindAllStringSubmatch 替代 FindStringSubmatch
    - _Requirements: 4.3, 4.6_
  
  - [x] 4.4 优化 JSON 解析错误日志
    - 确保错误日志包含最多前 500 字符
    - _Requirements: 4.5_
  
  - [x] 4.5 编写响应解析属性测试
    - **Property 6: Response Parsing Completeness**
    - **Property 7: JSON Error Logging**
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.5, 4.6**

- [x] 5. Checkpoint - 确保后端数据处理测试通过
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. 修改 DraggableDashboard 默认布局为全宽
  - [x] 6.1 修改 defaultLayout 配置
    - 将所有 Layout_Item 的 w 值设为 100
    - 调整 y 值实现垂直堆叠布局
    - _Requirements: 5.1, 5.2_
  
  - [x] 6.2 确保布局加载时保留保存的宽度值
    - 验证 LoadLayout 正确恢复宽度
    - _Requirements: 5.3_
  
  - [x] 6.3 编写布局配置属性测试
    - **Property 8: Layout Configuration Integrity**
    - **Property 9: Layout Persistence Round-Trip**
    - **Validates: Requirements 5.2, 5.3, 6.1, 6.2, 6.5**

- [x] 7. 优化 AnalysisResultManager 数据处理
  - [x] 7.1 确保数据存储保留正确的 sessionId 和 messageId
    - 验证 updateResults 方法正确处理 ID
    - _Requirements: 3.4_
  
  - [x] 7.2 确保会话切换时清除旧数据
    - 验证 switchSession 方法清除旧会话数据
    - _Requirements: 7.5_
  
  - [x] 7.3 编写 AnalysisResultManager 属性测试
    - **Property 4: AnalysisResultManager Data Storage**
    - **Property 11: Session Switching Clears Data**
    - **Validates: Requirements 3.4, 7.5**

- [x] 8. 优化错误处理和数据类型支持
  - [x] 8.1 确保错误事件包含恢复建议
    - 验证 EmitError 方法包含 recoverySuggestions
    - _Requirements: 7.4_
  
  - [x] 8.2 确保数据类型正确处理
    - 验证 metric、insight、file 数据正确规范化
    - _Requirements: 8.4, 8.5, 8.6_
  
  - [x] 8.3 编写错误处理和数据类型属性测试
    - **Property 10: Error Events Include Recovery Suggestions**
    - **Property 12: Data Type Processing**
    - **Validates: Requirements 7.4, 8.4, 8.5, 8.6**

- [x] 9. Final Checkpoint - 确保所有测试通过
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All tasks are required for comprehensive implementation
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
