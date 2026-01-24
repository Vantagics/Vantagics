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

// 分析结果管理器接口
export interface IAnalysisResultManager {
  // 数据更新
  updateResults(batch: AnalysisResultBatch): void;
  clearResults(sessionId: string, messageId?: string): void;
  clearAll(): void;
  
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
  
  // 加载状态
  setLoading(loading: boolean, requestId?: string): void;
  isLoading(): boolean;
  
  // 错误处理
  setError(error: string | null): void;
  getError(): string | null;
  
  // 获取完整状态
  getState(): AnalysisResultState;
}
