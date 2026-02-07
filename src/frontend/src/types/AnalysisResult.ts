/**
 * Analysis Result Types
 * 
 * 统一的分析结果数据模型，用于规范化所有分析结果类型
 */

// 分析结果类型枚举
export type AnalysisResultType = 
  | 'echarts'    // ECharts 图表配置对象
  | 'image'      // Base64 图片数据
  | 'table'      // 表格数据数组
  | 'csv'        // CSV 数据（转换为表格格式）
  | 'metric'     // 关键指标
  | 'insight'    // 智能洞察
  | 'file';      // 可下载文件

// 数据来源类型
export type ResultSource = 
  | 'realtime'      // 实时流式更新
  | 'completed'     // 分析完成后的完整数据
  | 'cached'        // 从缓存加载
  | 'restored';     // 从持久化存储恢复

// 结果元数据
export interface ResultMetadata {
  sessionId: string;             // 会话ID
  messageId: string;             // 消息ID
  timestamp: number;             // 时间戳
  requestId?: string;            // 请求ID（用于匹配）
  fileName?: string;             // 文件名（如适用）
  mimeType?: string;             // MIME类型（图片/文件）
}

// 统一的分析结果数据项
export interface AnalysisResultItem {
  id: string;                    // 唯一标识符
  type: AnalysisResultType;      // 数据类型
  data: any;                     // 规范化后的数据
  metadata: ResultMetadata;      // 元数据
  source: ResultSource;          // 数据来源
}

// 批量分析结果（单次事件传递）
export interface AnalysisResultBatch {
  sessionId: string;
  messageId: string;
  requestId: string;
  items: AnalysisResultItem[];
  isComplete: boolean;           // 是否为完整结果
  timestamp: number;
}

// 管理器内部状态
export interface AnalysisResultState {
  currentSessionId: string | null;
  currentMessageId: string | null;
  isLoading: boolean;
  pendingRequestId: string | null;
  error: string | null;
  // 按 sessionId -> messageId -> items 组织的数据
  data: Map<string, Map<string, AnalysisResultItem[]>>;
}

// 规范化后的表格数据格式
export interface NormalizedTableData {
  title?: string;
  columns: string[];
  rows: Record<string, any>[];
}

// 规范化后的指标数据格式
export interface NormalizedMetricData {
  title: string;
  value: string;
  change?: string;
  unit?: string;
}

// 规范化后的洞察数据格式
export interface NormalizedInsightData {
  text: string;
  icon?: string;
  dataSourceId?: string;
  sourceName?: string;
}

// 规范化后的文件数据格式
export interface NormalizedFileData {
  fileName: string;
  filePath: string;
  fileType: string;
  size?: number;
  preview?: string;  // base64预览图
}

// 规范化结果
export interface NormalizedResult<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

// 验证结果
export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

// 状态变更回调类型
export type StateChangeCallback = (state: AnalysisResultState) => void;

// 分析结果事件类型
export interface AnalysisResultEvents {
  'analysis-started': { sessionId: string; messageId: string; requestId: string };
  'session-switched': { fromSessionId: string | null; toSessionId: string };
  'message-selected': { sessionId: string; fromMessageId: string | null; toMessageId: string };
  // 历史请求无结果事件 - 当历史分析请求没有关联的分析结果时触发
  // 用于通知 useDashboardData 显示空状态而非数据源统计 (Requirement 2.4)
  'historical-empty-result': { sessionId: string; messageId: string };
  // 数据恢复完成事件 - 当历史数据恢复完成时触发 (Requirement 5.3)
  'data-restored': { 
    sessionId: string; 
    messageId: string; 
    itemCount: number; 
    validCount: number;
    invalidCount: number;
    itemsByType: Record<string, number>;
  };
}

// 恢复结果统计
export interface RestoreResultStats {
  totalItems: number;
  validItems: number;
  invalidItems: number;
  itemsByType: Record<string, number>;
  errors: string[];
}

// Error code constants for common error types
// These codes help categorize errors and provide appropriate recovery suggestions
export const ErrorCodes = {
  // Analysis errors
  ANALYSIS_ERROR: 'ANALYSIS_ERROR',
  ANALYSIS_TIMEOUT: 'ANALYSIS_TIMEOUT',
  ANALYSIS_CANCELLED: 'ANALYSIS_CANCELLED',
  
  // Python execution errors
  PYTHON_EXECUTION: 'PYTHON_EXECUTION',
  PYTHON_SYNTAX: 'PYTHON_SYNTAX',
  PYTHON_IMPORT: 'PYTHON_IMPORT',
  PYTHON_MEMORY: 'PYTHON_MEMORY',
  
  // Data errors
  DATA_NOT_FOUND: 'DATA_NOT_FOUND',
  DATA_INVALID: 'DATA_INVALID',
  DATA_EMPTY: 'DATA_EMPTY',
  DATA_TOO_LARGE: 'DATA_TOO_LARGE',
  
  // Connection errors
  CONNECTION_FAILED: 'CONNECTION_FAILED',
  CONNECTION_TIMEOUT: 'CONNECTION_TIMEOUT',
  
  // Permission errors
  PERMISSION_DENIED: 'PERMISSION_DENIED',
  
  // Resource errors
  RESOURCE_BUSY: 'RESOURCE_BUSY',
  RESOURCE_NOT_FOUND: 'RESOURCE_NOT_FOUND',
} as const;

export type ErrorCode = typeof ErrorCodes[keyof typeof ErrorCodes];

// Enhanced error information with recovery suggestions
export interface EnhancedErrorInfo {
  code: ErrorCode | string;
  message: string;
  details?: string;
  recoverySuggestions: string[];
  timestamp: number;
}

// Error event payload from backend
export interface AnalysisErrorPayload {
  sessionId: string;
  threadId?: string;
  requestId?: string;
  code?: string;
  error?: string;
  message?: string;
  details?: string;
  recoverySuggestions?: string[];
  timestamp?: number;
}

// 事件回调类型
export type AnalysisResultEventCallback<K extends keyof AnalysisResultEvents> = 
  (data: AnalysisResultEvents[K]) => void;

// 分析结果管理器接口
export interface IAnalysisResultManager {
  // 数据更新
  updateResults(batch: AnalysisResultBatch): void;
  clearResults(sessionId: string, messageId?: string): void;
  clearAll(): void;
  
  // 历史数据恢复 (Requirement 5.3)
  restoreResults(sessionId: string, messageId: string, items: AnalysisResultItem[]): RestoreResultStats;
  
  // 数据查询
  getResults(sessionId: string, messageId: string): AnalysisResultItem[];
  getResultsByType(sessionId: string, messageId: string, type: AnalysisResultType): AnalysisResultItem[];
  hasData(sessionId: string, messageId: string, type?: AnalysisResultType): boolean;
  
  // 当前会话数据快捷方法
  getCurrentResults(): AnalysisResultItem[];
  getCurrentResultsByType(type: AnalysisResultType): AnalysisResultItem[];
  hasCurrentData(type?: AnalysisResultType): boolean;
  
  // 会话管理
  switchSession(sessionId: string): void;
  getCurrentSession(): string | null;
  selectMessage(messageId: string): void;
  getCurrentMessage(): string | null;
  
  // 状态订阅
  subscribe(callback: StateChangeCallback): () => void;
  
  // 事件订阅
  on<K extends keyof AnalysisResultEvents>(
    event: K, 
    callback: AnalysisResultEventCallback<K>
  ): () => void;
  
  // 加载状态
  setLoading(loading: boolean, requestId?: string, messageId?: string): void;
  isLoading(): boolean;
  
  // 错误处理 (Requirement 4.4)
  setError(error: string | null): void;
  setErrorWithInfo(errorInfo: EnhancedErrorInfo | null): void;
  getError(): string | null;
  getErrorInfo(): EnhancedErrorInfo | null;
  
  // 获取完整状态
  getState(): AnalysisResultState;
  
  // 历史请求空结果通知 (Requirement 2.4)
  // 当历史分析请求没有关联的分析结果时调用
  notifyHistoricalEmptyResult(sessionId: string, messageId: string): void;
}
