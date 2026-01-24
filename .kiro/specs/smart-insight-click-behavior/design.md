# Design Document: Smart Insight Click Behavior Optimization

## Overview

本设计文档描述了智能洞察点击行为优化功能的技术实现方案。该功能通过改进状态管理机制，确保用户点击洞察项发起新分析时，仪表盘保持当前显示内容不变，直到新的分析结果返回，从而提供更流畅的用户体验。

### 核心设计原则

1. **状态分离**: 将"当前显示的数据"与"正在加载的请求"分离管理
2. **数据持久性**: 在分析请求处理期间保持现有显示内容
3. **渐进式更新**: 只在新数据到达时更新显示
4. **请求去重**: 确保只处理最新的分析请求

## Architecture

### 当前架构分析

当前系统采用 React + TypeScript 架构，主要组件包括：

- **App.tsx**: 应用主组件，管理全局状态和事件监听
- **DraggableDashboard.tsx**: 仪表盘组件，显示分析结果
- **ChatSidebar.tsx**: 聊天侧边栏，处理用户交互和分析请求

**当前问题**:
1. 点击洞察项时，可能触发仪表盘数据清空
2. 加载状态与显示数据耦合，导致内容闪烁
3. 缺少对旧请求结果的过滤机制

### 改进后的架构

```
┌─────────────────────────────────────────────────────────────┐
│                         App.tsx                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  State Management                                       │ │
│  │  - dashboardData (current displayed)                    │ │
│  │  - pendingRequestId (loading request ID)                │ │
│  │  - isAnalysisLoading (loading flag)                     │ │
│  │  - lastCompletedRequestId (for deduplication)           │ │
│  └────────────────────────────────────────────────────────┘ │
│                           │                                  │
│                           ├──────────────┬──────────────────┤
│                           ▼              ▼                  ▼
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────┐│
│  │ DraggableDashboard│  │   ChatSidebar    │  │ EventBus   ││
│  │                   │  │                  │  │            ││
│  │ Props:            │  │ Emits:           │  │ Events:    ││
│  │ - data (stable)   │  │ - insight-click  │  │ - chat-*   ││
│  │ - isLoading       │  │ - send-message   │  │ - analysis-││
│  │ - requestId       │  │                  │  │   complete ││
│  └──────────────────┘  └──────────────────┘  └────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. State Management (App.tsx)

#### 新增状态变量

```typescript
interface AppState {
  // 当前显示的仪表盘数据（稳定，不因加载而清空）
  dashboardData: main.DashboardData | null;
  
  // 正在处理的请求ID（用于匹配返回结果）
  pendingRequestId: string | null;
  
  // 加载状态标志
  isAnalysisLoading: boolean;
  
  // 最后完成的请求ID（用于去重）
  lastCompletedRequestId: string | null;
  
  // 活动会话ID
  activeSessionId: string | null;
}
```

#### 状态更新逻辑

```typescript
// 点击洞察项时
const handleInsightClick = (insightText: string) => {
  const requestId = generateUniqueId();
  
  // 设置加载状态，但不清空 dashboardData
  setPendingRequestId(requestId);
  setIsAnalysisLoading(true);
  
  // 发送分析请求
  EventsEmit('chat-send-message-in-session', {
    text: insightText,
    threadId: activeSessionId,
    requestId: requestId
  });
};

// 接收分析结果时
const handleAnalysisComplete = (payload: any) => {
  const { requestId, data } = payload;
  
  // 验证请求ID匹配
  if (requestId !== pendingRequestId) {
    console.log('Ignoring stale analysis result');
    return;
  }
  
  // 更新显示数据
  setDashboardData(data);
  setLastCompletedRequestId(requestId);
  setPendingRequestId(null);
  setIsAnalysisLoading(false);
};
```

### 2. Dashboard Component (DraggableDashboard.tsx)

#### Props 接口

```typescript
interface DraggableDashboardProps {
  // 当前显示的数据（稳定）
  data: main.DashboardData | null;
  
  // 图表数据
  activeChart?: ChartData | null;
  
  // 加载状态（用于显示加载指示器）
  isAnalysisLoading?: boolean;
  
  // 当前请求ID（用于显示加载状态）
  loadingRequestId?: string | null;
  
  // 洞察点击回调
  onInsightClick?: (insightText: string) => void;
  
  // 其他现有props...
}
```

#### 渲染逻辑

```typescript
const DraggableDashboard: React.FC<DraggableDashboardProps> = ({
  data,
  isAnalysisLoading,
  onInsightClick,
  ...otherProps
}) => {
  // 渲染当前数据（不受加载状态影响）
  const renderContent = () => {
    if (!data) {
      return <EmptyState />;
    }
    
    return (
      <>
        {/* 显示加载指示器（覆盖层） */}
        {isAnalysisLoading && (
          <LoadingOverlay message="正在分析..." />
        )}
        
        {/* 显示当前数据 */}
        <MetricsSection metrics={data.metrics} />
        <InsightsSection 
          insights={data.insights}
          onInsightClick={onInsightClick}
        />
        <ChartsSection charts={data.charts} />
      </>
    );
  };
  
  return (
    <div className="dashboard-container">
      {renderContent()}
    </div>
  );
};
```

### 3. Event Communication

#### 事件定义

```typescript
// 洞察点击事件
interface InsightClickEvent {
  text: string;
  threadId: string;
  requestId: string;
  timestamp: number;
}

// 分析完成事件
interface AnalysisCompleteEvent {
  requestId: string;
  threadId: string;
  data: main.DashboardData;
  chartData?: ChartData;
  timestamp: number;
}

// 加载状态事件
interface LoadingStateEvent {
  loading: boolean;
  requestId: string | null;
  threadId: string;
}
```

#### 事件流程

```
User Click Insight
       │
       ▼
[DraggableDashboard]
  onInsightClick()
       │
       ▼
[App.tsx]
  handleInsightClick()
  - Generate requestId
  - Set loading state
  - Keep dashboardData unchanged
       │
       ▼
EventsEmit('chat-send-message-in-session')
       │
       ▼
[Backend Processing]
       │
       ▼
EventsOn('analysis-completed')
       │
       ▼
[App.tsx]
  handleAnalysisComplete()
  - Verify requestId
  - Update dashboardData
  - Clear loading state
       │
       ▼
[DraggableDashboard]
  Re-render with new data
```

## Data Models

### DashboardState

```typescript
interface DashboardState {
  // 当前显示的数据（持久化）
  displayData: {
    metrics: MetricCard[];
    insights: SmartInsight[];
    charts: ChartData[];
    tables: TableData[];
    files: SessionFile[];
  };
  
  // 请求状态
  requestState: {
    pending: string | null;      // 当前待处理的请求ID
    lastCompleted: string | null; // 最后完成的请求ID
    isLoading: boolean;           // 加载标志
  };
  
  // 会话信息
  sessionInfo: {
    activeThreadId: string | null;
    selectedMessageId: string | null;
  };
}
```

### RequestMetadata

```typescript
interface RequestMetadata {
  requestId: string;
  threadId: string;
  timestamp: number;
  source: 'insight-click' | 'user-message' | 'auto-analysis';
  insightText?: string;
}
```

## Correctness Properties

*属性是一种特征或行为，应该在系统的所有有效执行中保持为真——本质上是关于系统应该做什么的形式化陈述。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### Property 1: Dashboard Data Persistence During Loading

*For any* dashboard state with existing data, when an insight click triggers a new analysis request, the dashboard SHALL continue displaying the current data until new results arrive.

**Validates: Requirements 1.1, 1.2, 4.2**

### Property 2: Loading State Lifecycle

*For any* analysis request, the loading state SHALL be set to active when the request is initiated and SHALL be set to inactive when the request completes or fails.

**Validates: Requirements 2.2, 2.3, 2.4**

### Property 3: Request ID Matching

*For any* analysis result received, the system SHALL only update the dashboard if the result's requestId matches the current pendingRequestId, otherwise the result SHALL be ignored.

**Validates: Requirements 4.3, 4.4**

### Property 4: Request Cancellation

*For any* sequence of insight clicks, when a new insight is clicked while a previous request is processing, the system SHALL cancel the previous request and only process the most recent one.

**Validates: Requirements 5.3, 5.4**

### Property 5: Event Communication Integrity

*For any* insight click event, the system SHALL emit an event with the insight content and a unique requestId, and the event SHALL be received by the chat area to initiate the analysis request.

**Validates: Requirements 6.1, 6.2, 6.3**

### Property 6: State Separation

*For any* system state, the current displayed dashboard data and the pending request state SHALL be maintained as separate state variables that can be updated independently.

**Validates: Requirements 4.1**

### Property 7: Dashboard Update on Completion

*For any* valid analysis result (with matching requestId), when the result is received, the dashboard SHALL update to display the new content.

**Validates: Requirements 1.3**

## Error Handling

### 错误场景处理

#### 1. 请求超时

```typescript
const REQUEST_TIMEOUT = 30000; // 30秒

const handleInsightClick = (insightText: string) => {
  const requestId = generateUniqueId();
  setPendingRequestId(requestId);
  setIsAnalysisLoading(true);
  
  // 设置超时处理
  const timeoutId = setTimeout(() => {
    if (pendingRequestId === requestId) {
      // 超时，清除加载状态但保留数据
      setIsAnalysisLoading(false);
      setPendingRequestId(null);
      
      // 显示错误提示
      showToast('分析请求超时，请重试', 'error');
    }
  }, REQUEST_TIMEOUT);
  
  // 存储超时ID以便取消
  setRequestTimeout(requestId, timeoutId);
  
  // 发送请求...
};
```

#### 2. 请求失败

```typescript
const handleAnalysisError = (payload: any) => {
  const { requestId, error } = payload;
  
  // 验证是否是当前请求
  if (requestId !== pendingRequestId) {
    return;
  }
  
  // 清除加载状态，保留现有数据
  setIsAnalysisLoading(false);
  setPendingRequestId(null);
  
  // 显示错误信息
  showToast(`分析失败: ${error}`, 'error');
  
  // 数据保持不变
};
```

#### 3. 网络断开

```typescript
const handleNetworkError = () => {
  // 清除所有待处理请求
  setPendingRequestId(null);
  setIsAnalysisLoading(false);
  
  // 保留现有显示数据
  // dashboardData 不变
  
  // 显示网络错误提示
  showToast('网络连接已断开', 'warning');
};
```

#### 4. 会话切换

```typescript
const handleSessionSwitch = (newSessionId: string) => {
  // 取消当前会话的待处理请求
  if (pendingRequestId && activeSessionId !== newSessionId) {
    setPendingRequestId(null);
    setIsAnalysisLoading(false);
  }
  
  // 切换会话
  setActiveSessionId(newSessionId);
  
  // 加载新会话的数据
  loadSessionData(newSessionId);
};
```

### 错误恢复策略

1. **自动重试**: 对于临时性错误（如网络超时），提供自动重试机制
2. **手动重试**: 为用户提供重试按钮
3. **降级显示**: 在错误情况下保持现有内容显示
4. **错误日志**: 记录所有错误以便调试

## Testing Strategy

### 单元测试

#### 测试范围

1. **状态管理测试**
   - 测试 `handleInsightClick` 正确设置加载状态
   - 测试 `handleAnalysisComplete` 正确更新数据
   - 测试请求ID验证逻辑
   - 测试状态分离（displayData vs requestState）

2. **组件渲染测试**
   - 测试 DraggableDashboard 在加载时保持内容显示
   - 测试加载指示器正确显示/隐藏
   - 测试洞察点击事件正确触发

3. **事件通信测试**
   - 测试事件正确发送和接收
   - 测试事件数据结构完整性
   - 测试事件处理器正确调用

#### 示例测试用例

```typescript
describe('Smart Insight Click Behavior', () => {
  it('should maintain dashboard data during loading', () => {
    const initialData = createMockDashboardData();
    const { getByTestId } = render(<App initialData={initialData} />);
    
    // 点击洞察项
    const insight = getByTestId('insight-item-0');
    fireEvent.click(insight);
    
    // 验证数据仍然显示
    expect(getByTestId('metrics-section')).toBeInTheDocument();
    expect(getByTestId('insights-section')).toBeInTheDocument();
    
    // 验证加载指示器显示
    expect(getByTestId('loading-overlay')).toBeInTheDocument();
  });
  
  it('should ignore stale analysis results', () => {
    const { rerender } = render(<App />);
    
    // 发起第一个请求
    const requestId1 = 'req-1';
    act(() => {
      handleInsightClick('insight 1', requestId1);
    });
    
    // 发起第二个请求
    const requestId2 = 'req-2';
    act(() => {
      handleInsightClick('insight 2', requestId2);
    });
    
    // 第一个请求的结果返回（应该被忽略）
    act(() => {
      handleAnalysisComplete({
        requestId: requestId1,
        data: mockData1
      });
    });
    
    // 验证数据未更新
    expect(dashboardData).not.toEqual(mockData1);
    
    // 第二个请求的结果返回（应该被接受）
    act(() => {
      handleAnalysisComplete({
        requestId: requestId2,
        data: mockData2
      });
    });
    
    // 验证数据已更新
    expect(dashboardData).toEqual(mockData2);
  });
});
```

### 属性测试

#### 测试配置

- **测试库**: fast-check (TypeScript/JavaScript property-based testing)
- **迭代次数**: 最少 100 次
- **标签格式**: `Feature: smart-insight-click-behavior, Property {N}: {property_text}`

#### 属性测试实现

```typescript
import fc from 'fast-check';

describe('Property-Based Tests: Smart Insight Click Behavior', () => {
  // Property 1: Dashboard Data Persistence During Loading
  it('Feature: smart-insight-click-behavior, Property 1: Dashboard data persists during loading', () => {
    fc.assert(
      fc.property(
        fc.record({
          metrics: fc.array(fc.record({
            title: fc.string(),
            value: fc.string(),
            change: fc.string()
          })),
          insights: fc.array(fc.record({
            text: fc.string(),
            icon: fc.string()
          }))
        }),
        fc.string(),
        (initialData, insightText) => {
          // Setup: Create dashboard with initial data
          const dashboard = createDashboard(initialData);
          const dataBefore = dashboard.getData();
          
          // Action: Click insight
          dashboard.handleInsightClick(insightText);
          
          // Assert: Data should remain unchanged during loading
          const dataAfter = dashboard.getData();
          expect(dataAfter).toEqual(dataBefore);
          expect(dashboard.isLoading()).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });
  
  // Property 3: Request ID Matching
  it('Feature: smart-insight-click-behavior, Property 3: Only matching requestId updates dashboard', () => {
    fc.assert(
      fc.property(
        fc.string(),
        fc.string(),
        fc.record({
          metrics: fc.array(fc.anything()),
          insights: fc.array(fc.anything())
        }),
        (currentRequestId, staleRequestId, newData) => {
          fc.pre(currentRequestId !== staleRequestId); // Ensure IDs are different
          
          // Setup
          const dashboard = createDashboard();
          dashboard.setPendingRequestId(currentRequestId);
          const dataBefore = dashboard.getData();
          
          // Action: Receive result with stale ID
          dashboard.handleAnalysisComplete({
            requestId: staleRequestId,
            data: newData
          });
          
          // Assert: Data should not change
          expect(dashboard.getData()).toEqual(dataBefore);
          
          // Action: Receive result with matching ID
          dashboard.handleAnalysisComplete({
            requestId: currentRequestId,
            data: newData
          });
          
          // Assert: Data should update
          expect(dashboard.getData()).toEqual(newData);
        }
      ),
      { numRuns: 100 }
    );
  });
  
  // Property 4: Request Cancellation
  it('Feature: smart-insight-click-behavior, Property 4: Only most recent request is processed', () => {
    fc.assert(
      fc.property(
        fc.array(fc.string(), { minLength: 2, maxLength: 5 }),
        (insightTexts) => {
          // Setup
          const dashboard = createDashboard();
          const requestIds: string[] = [];
          
          // Action: Click multiple insights rapidly
          insightTexts.forEach(text => {
            const requestId = dashboard.handleInsightClick(text);
            requestIds.push(requestId);
          });
          
          // Assert: Only the last requestId should be pending
          const lastRequestId = requestIds[requestIds.length - 1];
          expect(dashboard.getPendingRequestId()).toBe(lastRequestId);
          
          // Action: Complete all requests
          requestIds.forEach((id, index) => {
            const data = { value: `data-${index}` };
            dashboard.handleAnalysisComplete({ requestId: id, data });
          });
          
          // Assert: Only the last request's data should be displayed
          expect(dashboard.getData().value).toBe(`data-${requestIds.length - 1}`);
        }
      ),
      { numRuns: 100 }
    );
  });
});
```

### 集成测试

#### 端到端测试场景

1. **完整流程测试**
   - 用户点击洞察项
   - 验证仪表盘内容保持不变
   - 验证聊天区域显示请求消息
   - 等待分析完成
   - 验证仪表盘更新为新内容

2. **多次点击测试**
   - 快速点击多个洞察项
   - 验证只有最后一个请求被处理
   - 验证仪表盘显示最后一个请求的结果

3. **错误恢复测试**
   - 模拟请求失败
   - 验证仪表盘保持原有内容
   - 验证错误提示显示
   - 验证可以重新发起请求

### 测试覆盖率目标

- **单元测试覆盖率**: ≥ 85%
- **属性测试覆盖率**: 所有核心属性
- **集成测试覆盖率**: 所有主要用户流程

## Implementation Notes

### 关键实现要点

1. **避免数据清空**: 在任何情况下都不要在加载开始时清空 `dashboardData`
2. **请求ID生成**: 使用 UUID 或时间戳+随机数确保唯一性
3. **事件命名一致性**: 使用统一的事件命名约定（如 `chat-send-message-in-session`）
4. **状态更新原子性**: 使用 React 的函数式更新确保状态一致性
5. **内存泄漏防护**: 及时清理超时定时器和事件监听器

### 性能优化

1. **避免不必要的重渲染**: 使用 `React.memo` 和 `useMemo` 优化组件
2. **事件去抖**: 对快速连续的点击事件进行去抖处理
3. **数据缓存**: 缓存会话数据避免重复加载

### 兼容性考虑

1. **向后兼容**: 保持对现有事件格式的支持
2. **渐进式迁移**: 可以逐步迁移到新的状态管理方式
3. **降级方案**: 在不支持新特性的环境中提供降级体验
