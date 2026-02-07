/**
 * Analysis Result Manager
 * 
 * 集中状态管理器，作为所有分析结果数据的单一数据源
 */

import {
  AnalysisResultItem,
  AnalysisResultType,
  AnalysisResultBatch,
  AnalysisResultState,
  IAnalysisResultManager,
  StateChangeCallback,
  ResultSource,
  ResultMetadata,
  AnalysisResultEvents,
  AnalysisResultEventCallback,
  RestoreResultStats,
  EnhancedErrorInfo,
  ErrorCodes,
} from '../types/AnalysisResult';
import { DataNormalizer } from '../utils/DataNormalizer';
import { createLogger } from '../utils/systemLog';

const logger = createLogger('AnalysisResultManager');

/**
 * 生成唯一ID
 */
function generateId(): string {
  return `${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * 根据错误代码获取恢复建议
 * 
 * Validates: Requirement 4.4 - 错误时显示友好的错误信息
 */
function getRecoverySuggestions(errorCode: string): string[] {
  const suggestions: string[] = [];
  
  switch (errorCode) {
    case ErrorCodes.ANALYSIS_ERROR:
      suggestions.push(
        '请检查您的查询是否清晰明确',
        '尝试简化查询条件',
        '如果问题持续，请刷新页面后重试'
      );
      break;
    
    case ErrorCodes.ANALYSIS_TIMEOUT:
      suggestions.push(
        '请尝试简化查询或减少数据范围',
        '检查网络连接是否稳定',
        '稍后重试，系统可能正在处理其他任务'
      );
      break;
    
    case ErrorCodes.ANALYSIS_CANCELLED:
      suggestions.push(
        '您可以重新发起分析请求',
        '如果是误操作，请再次提交相同的查询'
      );
      break;
    
    case ErrorCodes.PYTHON_EXECUTION:
      suggestions.push(
        '请检查数据格式是否正确',
        '尝试使用不同的分析方式',
        '如果问题持续，请联系技术支持'
      );
      break;
    
    case ErrorCodes.PYTHON_SYNTAX:
      suggestions.push(
        '系统生成的代码存在语法问题',
        '请尝试重新描述您的分析需求',
        '使用更简单的查询语句'
      );
      break;
    
    case ErrorCodes.PYTHON_IMPORT:
      suggestions.push(
        '所需的分析库可能未安装',
        '请联系管理员检查系统配置',
        '尝试使用其他分析方法'
      );
      break;
    
    case ErrorCodes.PYTHON_MEMORY:
      suggestions.push(
        '数据量可能过大，请减少查询范围',
        '尝试分批处理数据',
        '稍后重试，系统可能正在释放资源'
      );
      break;
    
    case ErrorCodes.DATA_NOT_FOUND:
      suggestions.push(
        '请检查数据源是否已正确配置',
        '确认查询的表或字段名称是否正确',
        '检查数据是否已被删除或移动'
      );
      break;
    
    case ErrorCodes.DATA_INVALID:
      suggestions.push(
        '请检查数据格式是否符合要求',
        '确认数据类型是否正确',
        '尝试清理或重新导入数据'
      );
      break;
    
    case ErrorCodes.DATA_EMPTY:
      suggestions.push(
        '当前查询条件下没有数据',
        '请尝试调整筛选条件',
        '检查数据源是否包含所需数据'
      );
      break;
    
    case ErrorCodes.DATA_TOO_LARGE:
      suggestions.push(
        '请减少查询的数据范围',
        '添加更多筛选条件',
        '考虑分页或分批查询'
      );
      break;
    
    case ErrorCodes.CONNECTION_FAILED:
      suggestions.push(
        '请检查网络连接',
        '确认服务是否正常运行',
        '稍后重试'
      );
      break;
    
    case ErrorCodes.CONNECTION_TIMEOUT:
      suggestions.push(
        '网络连接超时，请检查网络状态',
        '服务可能繁忙，请稍后重试',
        '如果问题持续，请联系技术支持'
      );
      break;
    
    case ErrorCodes.PERMISSION_DENIED:
      suggestions.push(
        '您可能没有访问此资源的权限',
        '请联系管理员获取相应权限',
        '检查您的账户状态'
      );
      break;
    
    case ErrorCodes.RESOURCE_BUSY:
      suggestions.push(
        '资源正在被其他任务使用',
        '请稍后重试',
        '如果问题持续，请联系技术支持'
      );
      break;
    
    case ErrorCodes.RESOURCE_NOT_FOUND:
      suggestions.push(
        '请检查资源路径是否正确',
        '确认资源是否已被删除',
        '联系管理员确认资源状态'
      );
      break;
    
    default:
      suggestions.push(
        '请稍后重试',
        '如果问题持续，请联系技术支持'
      );
  }
  
  return suggestions;
}

/**
 * 根据错误代码获取用户友好的错误消息
 * 
 * Validates: Requirement 4.4 - 错误时显示友好的错误信息
 */
function getUserFriendlyMessage(errorCode: string, originalMessage?: string): string {
  // If original message is already user-friendly (Chinese), use it
  if (originalMessage && originalMessage.length > 0) {
    // Check if it's already a Chinese message
    const hasChinese = /[\u4e00-\u9fff]/.test(originalMessage);
    if (hasChinese) {
      return originalMessage;
    }
  }
  
  // Generate user-friendly message based on error code
  switch (errorCode) {
    case ErrorCodes.ANALYSIS_ERROR:
      return '分析过程中发生错误';
    case ErrorCodes.ANALYSIS_TIMEOUT:
      return '分析超时，请稍后重试';
    case ErrorCodes.ANALYSIS_CANCELLED:
      return '分析已取消';
    case ErrorCodes.PYTHON_EXECUTION:
      return '代码执行失败';
    case ErrorCodes.PYTHON_SYNTAX:
      return '代码语法错误';
    case ErrorCodes.PYTHON_IMPORT:
      return '缺少必要的分析库';
    case ErrorCodes.PYTHON_MEMORY:
      return '内存不足，数据量可能过大';
    case ErrorCodes.DATA_NOT_FOUND:
      return '未找到请求的数据';
    case ErrorCodes.DATA_INVALID:
      return '数据格式无效';
    case ErrorCodes.DATA_EMPTY:
      return '查询结果为空';
    case ErrorCodes.DATA_TOO_LARGE:
      return '数据量超出限制';
    case ErrorCodes.CONNECTION_FAILED:
      return '连接失败，请检查网络';
    case ErrorCodes.CONNECTION_TIMEOUT:
      return '连接超时';
    case ErrorCodes.PERMISSION_DENIED:
      return '权限不足';
    case ErrorCodes.RESOURCE_BUSY:
      return '资源繁忙，请稍后重试';
    case ErrorCodes.RESOURCE_NOT_FOUND:
      return '资源未找到';
    default:
      return originalMessage || '发生未知错误';
  }
}

/**
 * 创建增强的错误信息
 * 
 * Validates: Requirement 4.4 - 错误时显示友好的错误信息
 */
function createEnhancedErrorInfo(
  errorCode: string,
  errorMessage?: string,
  details?: string,
  existingSuggestions?: string[]
): EnhancedErrorInfo {
  return {
    code: errorCode,
    message: getUserFriendlyMessage(errorCode, errorMessage),
    details: details,
    recoverySuggestions: existingSuggestions && existingSuggestions.length > 0 
      ? existingSuggestions 
      : getRecoverySuggestions(errorCode),
    timestamp: Date.now(),
  };
}

/**
 * 格式化错误信息为显示字符串
 * 
 * Validates: Requirement 4.4 - 错误时显示友好的错误信息
 */
function formatErrorForDisplay(errorInfo: EnhancedErrorInfo): string {
  let displayMessage = errorInfo.message;
  
  // Add recovery suggestions if available
  if (errorInfo.recoverySuggestions && errorInfo.recoverySuggestions.length > 0) {
    displayMessage += '\n\n建议：\n';
    errorInfo.recoverySuggestions.forEach((suggestion, index) => {
      displayMessage += `${index + 1}. ${suggestion}\n`;
    });
  }
  
  return displayMessage;
}

/**
 * 历史请求空结果事件数据
 * 
 * 当历史分析请求没有关联的分析结果时触发
 * 用于通知 useDashboardData 显示空状态而非数据源统计
 */
export interface HistoricalEmptyResultEvent {
  sessionId: string;
  messageId: string;
}

/**
 * AnalysisResultManager 单例类
 * 
 * 负责：
 * 1. 统一管理所有分析结果数据
 * 2. 按 sessionId -> messageId 组织数据
 * 3. 提供数据查询和更新接口
 * 4. 管理加载状态和错误状态
 * 5. 支持状态订阅机制
 */
class AnalysisResultManagerImpl implements IAnalysisResultManager {
  private static instance: AnalysisResultManagerImpl | null = null;
  
  private state: AnalysisResultState;
  private subscribers: Set<StateChangeCallback>;
  private updateQueue: AnalysisResultBatch[];
  private isProcessingQueue: boolean;
  
  // 事件监听器存储
  private eventListeners: Map<keyof AnalysisResultEvents, Set<AnalysisResultEventCallback<any>>>;
  
  // 增强的错误信息存储 (Requirement 4.4)
  private errorInfo: EnhancedErrorInfo | null;
  
  private constructor() {
    this.state = {
      currentSessionId: null,
      currentMessageId: null,
      isLoading: false,
      pendingRequestId: null,
      error: null,
      data: new Map(),
    };
    this.subscribers = new Set();
    this.updateQueue = [];
    this.isProcessingQueue = false;
    this.eventListeners = new Map();
    this.errorInfo = null;
  }
  
  /**
   * 获取单例实例
   */
  static getInstance(): AnalysisResultManagerImpl {
    if (!AnalysisResultManagerImpl.instance) {
      AnalysisResultManagerImpl.instance = new AnalysisResultManagerImpl();
    }
    return AnalysisResultManagerImpl.instance;
  }
  
  /**
   * 重置实例（用于测试）
   */
  static resetInstance(): void {
    AnalysisResultManagerImpl.instance = null;
  }
  
  // ==================== 数据更新 ====================
  
  /**
   * 更新分析结果（支持批量）
   */
  updateResults(batch: AnalysisResultBatch): void {
    logger.debug(`[updateResults] Queuing batch: session=${batch.sessionId}, message=${batch.messageId}, items=${batch.items?.length || 0}, queueSize=${this.updateQueue.length + 1}`);
    
    // 添加到队列
    this.updateQueue.push(batch);
    
    // 处理队列
    this.processQueue();
  }
  
  /**
   * 处理更新队列（顺序处理，防止并发问题）
   */
  private async processQueue(): Promise<void> {
    if (this.isProcessingQueue) {
      return;
    }
    
    this.isProcessingQueue = true;
    
    while (this.updateQueue.length > 0) {
      const batch = this.updateQueue.shift()!;
      this.processBatch(batch);
    }
    
    this.isProcessingQueue = false;
  }
  
  /**
   * 处理单个批次
   */
  private processBatch(batch: AnalysisResultBatch): void {
    const { sessionId, messageId, items, isComplete, requestId } = batch;
    
    logger.warn(`[processBatch] Processing batch: session=${sessionId}, message=${messageId}, items=${items.length}, complete=${isComplete}`);
    
    // 记录每个项目的类型
    for (const item of items) {
      logger.warn(`[processBatch] Item: id=${item.id}, type=${item.type}, dataType=${typeof item.data}`);
    }
    
    // 检查requestId是否匹配（如果有pendingRequestId）
    if (this.state.pendingRequestId && requestId !== this.state.pendingRequestId) {
      logger.warn(`[processBatch] Ignoring stale batch: received=${requestId}, expected=${this.state.pendingRequestId}`);
      return;
    }
    
    // 自动更新当前会话和消息（确保数据能被正确获取）
    // 只有当收到新数据时才更新，避免覆盖用户手动选择
    if (sessionId && items.length > 0) {
      if (this.state.currentSessionId !== sessionId) {
        logger.debug(`Auto-switching session: ${this.state.currentSessionId} -> ${sessionId}`);
        this.state.currentSessionId = sessionId;
      }
      if (this.state.currentMessageId !== messageId) {
        logger.debug(`Auto-selecting message: ${this.state.currentMessageId} -> ${messageId}`);
        this.state.currentMessageId = messageId;
      }
    }
    
    // 获取或创建session数据
    if (!this.state.data.has(sessionId)) {
      this.state.data.set(sessionId, new Map());
    }
    const sessionData = this.state.data.get(sessionId)!;
    
    // 当新的 messageId 数据到达时，清除该 session 下的旧 message 数据
    // 这样仪表盘会在干净的状态下展示新的分析结果
    if (!sessionData.has(messageId) && sessionData.size > 0) {
      logger.debug(`Clearing old message data for session ${sessionId} before adding new message ${messageId}`);
      sessionData.clear();
    }
    
    // 获取或创建message数据
    if (!sessionData.has(messageId)) {
      sessionData.set(messageId, []);
    }
    const messageData = sessionData.get(messageId)!;
    
    // 规范化并添加数据项
    for (const item of items) {
      const normalizedResult = DataNormalizer.normalize(item.type, item.data);
      
      if (normalizedResult.success) {
        // 检查是否已存在相同ID的项
        const existingIndex = messageData.findIndex(existing => existing.id === item.id);
        
        const normalizedItem: AnalysisResultItem = {
          ...item,
          data: normalizedResult.data,
        };
        
        if (existingIndex >= 0) {
          // 更新现有项
          messageData[existingIndex] = normalizedItem;
          logger.debug(`Updated existing item: ${item.id}, type=${item.type}`);
        } else {
          // 添加新项
          messageData.push(normalizedItem);
          logger.debug(`Added new item: ${item.id}, type=${item.type}`);
        }
      } else {
        logger.warn(`Failed to normalize item: ${item.id}, type=${item.type}, error=${normalizedResult.error}`);
      }
    }
    
    // 如果是完整结果，清除加载状态
    if (isComplete) {
      this.state.isLoading = false;
      this.state.pendingRequestId = null;
      this.state.error = null;
    }
    
    // 通知订阅者
    this.notifySubscribers();
  }
  
  /**
   * 清除分析结果
   * 
   * 支持两种模式：
   * 1. 清除特定消息的数据（提供 messageId）
   * 2. 清除整个会话的数据（不提供 messageId）
   * 
   * Validates: Requirements 5.1, 5.2
   */
  clearResults(sessionId: string, messageId?: string): void {
    if (messageId) {
      // 清除特定消息的数据
      const sessionData = this.state.data.get(sessionId);
      if (sessionData) {
        const existingItems = sessionData.get(messageId);
        const itemCount = existingItems?.length || 0;
        sessionData.delete(messageId);
        logger.info(`[clearResults] Cleared ${itemCount} items for message: ${messageId} (session: ${sessionId})`);
      } else {
        logger.debug(`[clearResults] No session data found for session: ${sessionId}`);
      }
    } else {
      // 清除整个会话的数据
      const sessionData = this.state.data.get(sessionId);
      if (sessionData) {
        const messageCount = sessionData.size;
        let totalItemCount = 0;
        sessionData.forEach((items) => {
          totalItemCount += items.length;
        });
        this.state.data.delete(sessionId);
        logger.info(`[clearResults] Cleared entire session: ${sessionId} (${messageCount} messages, ${totalItemCount} items)`);
      } else {
        logger.debug(`[clearResults] No session data found for session: ${sessionId}`);
      }
    }
    
    this.notifySubscribers();
  }
  
  /**
   * 清除所有数据
   * 
   * 完全重置管理器状态，包括：
   * - 所有会话和消息数据
   * - 当前会话和消息选择
   * - 加载状态和错误状态
   * 
   * Validates: Requirements 5.1, 5.2
   */
  clearAll(): void {
    // 记录清除前的状态
    const sessionCount = this.state.data.size;
    let totalMessageCount = 0;
    let totalItemCount = 0;
    this.state.data.forEach((sessionData) => {
      totalMessageCount += sessionData.size;
      sessionData.forEach((items) => {
        totalItemCount += items.length;
      });
    });
    
    logger.info(`[clearAll] Starting full clear: ${sessionCount} sessions, ${totalMessageCount} messages, ${totalItemCount} items`);
    
    // 清除所有数据
    this.state.data.clear();
    this.state.currentSessionId = null;
    this.state.currentMessageId = null;
    this.state.isLoading = false;
    this.state.pendingRequestId = null;
    this.state.error = null;
    this.errorInfo = null; // 清除增强的错误信息
    
    logger.info(`[clearAll] Full clear completed: all state reset to initial values`);
    logger.debug(`[clearAll] State: currentSessionId=null, currentMessageId=null, isLoading=false`);
    
    this.notifySubscribers();
  }
  
  // ==================== 历史数据恢复 ====================
  
  /**
   * 恢复历史分析结果
   * 
   * 用于从持久化存储恢复历史消息的分析结果。
   * 该方法会：
   * 1. 验证每个数据项的完整性和有效性
   * 2. 规范化数据格式
   * 3. 记录详细的恢复日志
   * 4. 触发 data-restored 事件
   * 
   * Validates: Requirement 5.3 - 历史消息的结果能正确恢复
   * 
   * @param sessionId - 会话ID
   * @param messageId - 消息ID
   * @param items - 要恢复的分析结果项
   * @returns 恢复结果统计
   */
  restoreResults(sessionId: string, messageId: string, items: AnalysisResultItem[]): RestoreResultStats {
    logger.info(`[restoreResults] Starting restoration: session=${sessionId}, message=${messageId}, items=${items?.length || 0}`);
    
    // 初始化统计
    const stats: RestoreResultStats = {
      totalItems: items?.length || 0,
      validItems: 0,
      invalidItems: 0,
      itemsByType: {},
      errors: [],
    };
    
    // 处理空数据情况
    if (!items || items.length === 0) {
      logger.info(`[restoreResults] No items to restore, notifying empty result`);
      this.notifyHistoricalEmptyResult(sessionId, messageId);
      return stats;
    }
    
    // 验证并规范化数据项
    const validItems: AnalysisResultItem[] = [];
    
    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      
      // 详细记录原始数据类型
      const originalDataType = typeof item.data;
      const originalDataPreview = originalDataType === 'string' 
        ? (item.data as string).substring(0, 100) + '...'
        : JSON.stringify(item.data).substring(0, 100) + '...';
      logger.warn(`[restoreResults] Item ${i}: type=${item.type}, originalDataType=${originalDataType}, preview=${originalDataPreview}`);
      
      const validationResult = this.validateRestoreItem(item, i);
      
      if (validationResult.valid) {
        // 规范化数据
        logger.warn(`[restoreResults] Item ${i}: calling DataNormalizer.normalize(${item.type}, ${originalDataType})`);
        const normalizedResult = DataNormalizer.normalize(item.type, item.data);
        
        if (normalizedResult.success) {
          // 记录规范化后的数据类型
          const normalizedDataType = typeof normalizedResult.data;
          logger.warn(`[restoreResults] Item ${i}: normalized successfully, resultDataType=${normalizedDataType}`);
          
          // 创建恢复的数据项，标记来源为 'restored'
          const restoredItem: AnalysisResultItem = {
            ...item,
            id: item.id || generateId(),
            data: normalizedResult.data,
            source: 'restored' as ResultSource,
            metadata: {
              ...item.metadata,
              sessionId,
              messageId,
              timestamp: item.metadata?.timestamp || Date.now(),
            },
          };
          
          validItems.push(restoredItem);
          stats.validItems++;
          
          // 统计类型分布
          stats.itemsByType[item.type] = (stats.itemsByType[item.type] || 0) + 1;
          
          logger.debug(`[restoreResults] Item ${i} validated and normalized: id=${restoredItem.id}, type=${item.type}`);
        } else {
          stats.invalidItems++;
          const error = `Item ${i} normalization failed: ${normalizedResult.error}`;
          stats.errors.push(error);
          logger.warn(`[restoreResults] ${error}`);
        }
      } else {
        stats.invalidItems++;
        const error = `Item ${i} validation failed: ${validationResult.errors.join(', ')}`;
        stats.errors.push(error);
        logger.warn(`[restoreResults] ${error}`);
      }
    }
    
    // 记录恢复统计
    logger.info(`[restoreResults] Validation complete: valid=${stats.validItems}, invalid=${stats.invalidItems}`);
    logger.debug(`[restoreResults] Items by type: ${JSON.stringify(stats.itemsByType)}`);
    
    if (stats.errors.length > 0) {
      logger.warn(`[restoreResults] Errors encountered: ${stats.errors.length}`);
      stats.errors.forEach((err, idx) => {
        logger.debug(`[restoreResults] Error ${idx + 1}: ${err}`);
      });
    }
    
    // 如果没有有效数据项，通知空结果
    if (validItems.length === 0) {
      logger.info(`[restoreResults] No valid items after validation, notifying empty result`);
      this.notifyHistoricalEmptyResult(sessionId, messageId);
      return stats;
    }
    
    // 清除当前会话的旧数据，确保数据隔离
    if (this.state.currentSessionId && this.state.data.has(this.state.currentSessionId)) {
      const oldSessionData = this.state.data.get(this.state.currentSessionId)!;
      const clearedCount = oldSessionData.size;
      oldSessionData.clear();
      logger.debug(`[restoreResults] Cleared ${clearedCount} old messages from current session`);
    }
    
    // 更新当前会话和消息
    this.state.currentSessionId = sessionId;
    this.state.currentMessageId = messageId;
    
    // 获取或创建session数据
    if (!this.state.data.has(sessionId)) {
      this.state.data.set(sessionId, new Map());
    }
    const sessionData = this.state.data.get(sessionId)!;
    
    // 清除目标会话的旧数据
    sessionData.clear();
    
    // 设置恢复的数据
    sessionData.set(messageId, validItems);
    
    // 清除加载状态和错误
    this.state.isLoading = false;
    this.state.pendingRequestId = null;
    this.state.error = null;
    
    // 触发 data-restored 事件
    this.emit('data-restored', {
      sessionId,
      messageId,
      itemCount: stats.totalItems,
      validCount: stats.validItems,
      invalidCount: stats.invalidItems,
      itemsByType: stats.itemsByType,
    });
    
    logger.info(`[restoreResults] Restoration completed successfully: ${validItems.length} items restored`);
    logger.debug(`[restoreResults] Final state: session=${sessionId}, message=${messageId}, items=${validItems.length}`);
    
    // 通知订阅者
    this.notifySubscribers();
    
    return stats;
  }
  
  /**
   * 验证恢复数据项的完整性
   * 
   * 检查数据项是否包含必要的字段和有效的数据类型
   * 
   * @param item - 要验证的数据项
   * @param index - 数据项索引（用于日志）
   * @returns 验证结果
   */
  private validateRestoreItem(item: AnalysisResultItem, index: number): { valid: boolean; errors: string[] } {
    const errors: string[] = [];
    
    // 检查基本结构
    if (!item) {
      errors.push('Item is null or undefined');
      return { valid: false, errors };
    }
    
    // 检查类型字段
    if (!item.type) {
      errors.push('Missing type field');
    } else {
      const validTypes: AnalysisResultType[] = ['echarts', 'image', 'table', 'csv', 'metric', 'insight', 'file'];
      if (!validTypes.includes(item.type)) {
        errors.push(`Invalid type: ${item.type}`);
      }
    }
    
    // 检查数据字段
    if (item.data === undefined || item.data === null) {
      errors.push('Missing or null data field');
    }
    
    // 类型特定验证
    if (item.type && item.data !== undefined && item.data !== null) {
      switch (item.type) {
        case 'echarts':
          // ECharts data can be either an object or a JSON string
          // DataNormalizer.normalizeECharts handles both formats
          if (typeof item.data !== 'object' && typeof item.data !== 'string') {
            errors.push('ECharts data must be an object or JSON string');
          }
          break;
        case 'image':
          if (typeof item.data !== 'string' || item.data.length === 0) {
            errors.push('Image data must be a non-empty string');
          }
          break;
        case 'table':
          // Table data can be an array, object, or JSON string
          // DataNormalizer.normalizeTable handles all formats
          if (!Array.isArray(item.data) && typeof item.data !== 'object' && typeof item.data !== 'string') {
            errors.push('Table data must be an array, object, or JSON string');
          }
          break;
        case 'metric':
          if (typeof item.data !== 'object') {
            errors.push('Metric data must be an object');
          }
          break;
        case 'insight':
          if (typeof item.data !== 'object' && typeof item.data !== 'string') {
            errors.push('Insight data must be an object or string');
          }
          break;
        case 'file':
          if (typeof item.data !== 'object') {
            errors.push('File data must be an object');
          }
          break;
      }
    }
    
    const valid = errors.length === 0;
    
    if (!valid) {
      logger.debug(`[validateRestoreItem] Item ${index} validation failed: ${errors.join(', ')}`);
    }
    
    return { valid, errors };
  }
  
  // ==================== 数据查询 ====================
  
  /**
   * 获取指定会话和消息的所有结果
   */
  getResults(sessionId: string, messageId: string): AnalysisResultItem[] {
    const sessionData = this.state.data.get(sessionId);
    if (!sessionData) return [];
    
    return sessionData.get(messageId) || [];
  }
  
  /**
   * 获取指定类型的结果
   */
  getResultsByType(sessionId: string, messageId: string, type: AnalysisResultType): AnalysisResultItem[] {
    const results = this.getResults(sessionId, messageId);
    return results.filter(item => item.type === type);
  }
  
  /**
   * 检查是否有数据
   */
  hasData(sessionId: string, messageId: string, type?: AnalysisResultType): boolean {
    const results = this.getResults(sessionId, messageId);
    
    if (type) {
      return results.some(item => item.type === type);
    }
    
    return results.length > 0;
  }
  
  /**
   * 获取当前会话和消息的所有结果
   */
  getCurrentResults(): AnalysisResultItem[] {
    if (!this.state.currentSessionId || !this.state.currentMessageId) {
      return [];
    }
    return this.getResults(this.state.currentSessionId, this.state.currentMessageId);
  }
  
  /**
   * 获取当前会话和消息的指定类型结果
   */
  getCurrentResultsByType(type: AnalysisResultType): AnalysisResultItem[] {
    if (!this.state.currentSessionId || !this.state.currentMessageId) {
      return [];
    }
    return this.getResultsByType(this.state.currentSessionId, this.state.currentMessageId, type);
  }
  
  /**
   * 检查当前会话和消息是否有数据
   */
  hasCurrentData(type?: AnalysisResultType): boolean {
    if (!this.state.currentSessionId || !this.state.currentMessageId) {
      return false;
    }
    return this.hasData(this.state.currentSessionId, this.state.currentMessageId, type);
  }
  
  // ==================== 会话管理 ====================
  
  /**
   * 切换会话
   * 
   * 当切换到不同会话时：
   * 1. 清除旧会话的所有数据（确保数据隔离）
   * 2. 取消当前的pending请求
   * 3. 重置消息选择和错误状态
   * 4. 触发 session-switched 事件
   * 
   * Validates: Requirements 5.1, 5.2
   */
  switchSession(sessionId: string): void {
    if (this.state.currentSessionId === sessionId) {
      logger.debug(`[switchSession] Session unchanged, skipping: ${sessionId}`);
      return;
    }
    
    const fromSessionId = this.state.currentSessionId;
    const fromMessageId = this.state.currentMessageId;
    
    logger.info(`[switchSession] Starting session switch: ${fromSessionId || 'null'} -> ${sessionId}`);
    
    // 记录切换前的状态
    const oldSessionDataSize = fromSessionId ? (this.state.data.get(fromSessionId)?.size || 0) : 0;
    logger.debug(`[switchSession] Old session data: ${oldSessionDataSize} messages`);
    
    // 清除旧会话的数据（确保数据隔离 - Requirement 5.1）
    if (fromSessionId && this.state.data.has(fromSessionId)) {
      const oldSessionData = this.state.data.get(fromSessionId)!;
      const clearedMessageCount = oldSessionData.size;
      let clearedItemCount = 0;
      oldSessionData.forEach((items) => {
        clearedItemCount += items.length;
      });
      
      oldSessionData.clear();
      logger.info(`[switchSession] Cleared old session data: ${clearedMessageCount} messages, ${clearedItemCount} items`);
    }
    
    // 取消当前的pending请求
    if (this.state.pendingRequestId) {
      logger.info(`[switchSession] Canceling pending request: ${this.state.pendingRequestId}`);
      this.state.pendingRequestId = null;
      this.state.isLoading = false;
    }
    
    // 更新状态
    this.state.currentSessionId = sessionId;
    this.state.currentMessageId = null; // 重置消息选择
    this.state.error = null;
    
    logger.debug(`[switchSession] State updated: currentSessionId=${sessionId}, currentMessageId=null`);
    
    // 触发 session-switched 事件
    this.emit('session-switched', {
      fromSessionId,
      toSessionId: sessionId,
    });
    
    logger.info(`[switchSession] Session switch completed: ${fromSessionId || 'null'} -> ${sessionId}`);
    
    this.notifySubscribers();
  }
  
  /**
   * 获取当前会话ID
   */
  getCurrentSession(): string | null {
    return this.state.currentSessionId;
  }
  
  /**
   * 选择消息
   * 
   * 当切换到新消息时：
   * 1. 清除当前会话下的所有旧消息数据（确保数据隔离 - Requirement 5.2）
   * 2. 保留新消息的已有数据（如果有）
   * 3. 触发 message-selected 事件
   * 
   * 这确保仪表盘只显示当前选中消息的分析结果，不会与其他消息的结果混淆
   * 
   * Validates: Requirements 5.1, 5.2
   */
  selectMessage(messageId: string): void {
    if (this.state.currentMessageId === messageId) {
      logger.debug(`[selectMessage] Message unchanged, skipping: ${messageId}`);
      return;
    }
    
    const fromMessageId = this.state.currentMessageId;
    const sessionId = this.state.currentSessionId || '';
    
    logger.info(`[selectMessage] Starting message selection: ${fromMessageId || 'null'} -> ${messageId} (session: ${sessionId})`);
    
    // 清除当前会话下的所有旧消息数据（除了新选中的消息）
    // 这样仪表盘会在干净的状态下等待新数据加载（Requirement 5.2）
    if (this.state.currentSessionId) {
      const sessionData = this.state.data.get(this.state.currentSessionId);
      if (sessionData) {
        // 记录清除前的状态
        const oldMessageCount = sessionData.size;
        let oldItemCount = 0;
        sessionData.forEach((items, msgId) => {
          if (msgId !== messageId) {
            oldItemCount += items.length;
          }
        });
        
        logger.debug(`[selectMessage] Session has ${oldMessageCount} messages before clearing`);
        
        // 保存新消息的数据（如果有）
        const newMessageData = sessionData.get(messageId);
        const preservedItemCount = newMessageData?.length || 0;
        
        // 清除所有旧数据
        sessionData.clear();
        logger.info(`[selectMessage] Cleared ${oldMessageCount} messages, ${oldItemCount} items from session ${this.state.currentSessionId}`);
        
        // 恢复新消息的数据（如果有）
        if (newMessageData && newMessageData.length > 0) {
          sessionData.set(messageId, newMessageData);
          logger.info(`[selectMessage] Restored ${preservedItemCount} items for message ${messageId}`);
        } else {
          logger.debug(`[selectMessage] No existing data for message ${messageId}, dashboard will be empty until new data arrives`);
        }
      } else {
        logger.debug(`[selectMessage] No session data found for session ${this.state.currentSessionId}`);
      }
    } else {
      logger.debug(`[selectMessage] No current session, skipping data clearing`);
    }
    
    // 更新当前消息
    this.state.currentMessageId = messageId;
    
    logger.debug(`[selectMessage] State updated: currentMessageId=${messageId}`);
    
    // 触发 message-selected 事件
    this.emit('message-selected', {
      sessionId,
      fromMessageId,
      toMessageId: messageId,
    });
    
    logger.info(`[selectMessage] Message selection completed: ${fromMessageId || 'null'} -> ${messageId}`);
    
    this.notifySubscribers();
  }
  
  /**
   * 获取当前消息ID
   */
  getCurrentMessage(): string | null {
    return this.state.currentMessageId;
  }
  
  // ==================== 状态订阅 ====================
  
  /**
   * 订阅状态变更
   */
  subscribe(callback: StateChangeCallback): () => void {
    this.subscribers.add(callback);
    logger.debug(`[subscribe] New subscriber added, total subscribers: ${this.subscribers.size}`);
    
    // 返回取消订阅函数
    return () => {
      this.subscribers.delete(callback);
      logger.debug(`[subscribe] Subscriber removed, remaining subscribers: ${this.subscribers.size}`);
    };
  }
  
  /**
   * 通知所有订阅者
   */
  private notifySubscribers(): void {
    const stateCopy = this.getState();
    logger.debug(`[notifySubscribers] Notifying ${this.subscribers.size} subscribers`);
    
    let successCount = 0;
    let errorCount = 0;
    
    this.subscribers.forEach(callback => {
      try {
        callback(stateCopy);
        successCount++;
      } catch (error) {
        errorCount++;
        logger.error(`[notifySubscribers] Subscriber callback error: ${error}`);
      }
    });
    
    if (errorCount > 0) {
      logger.warn(`[notifySubscribers] Completed with ${errorCount} errors out of ${this.subscribers.size} subscribers`);
    } else {
      logger.debug(`[notifySubscribers] Successfully notified ${successCount} subscribers`);
    }
  }
  
  // ==================== 事件订阅 ====================
  
  /**
   * 订阅特定事件
   */
  on<K extends keyof AnalysisResultEvents>(
    event: K,
    callback: AnalysisResultEventCallback<K>
  ): () => void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, new Set());
    }
    
    const listeners = this.eventListeners.get(event)!;
    listeners.add(callback);
    
    logger.debug(`Event listener added for: ${event}`);
    
    // 返回取消订阅函数
    return () => {
      listeners.delete(callback);
      logger.debug(`Event listener removed for: ${event}`);
    };
  }
  
  /**
   * 触发事件
   */
  private emit<K extends keyof AnalysisResultEvents>(
    event: K,
    data: AnalysisResultEvents[K]
  ): void {
    const listeners = this.eventListeners.get(event);
    if (!listeners || listeners.size === 0) {
      logger.debug(`No listeners for event: ${event}`);
      return;
    }
    
    logger.debug(`Emitting event: ${event}, data=${JSON.stringify(data)}`);
    
    listeners.forEach(callback => {
      try {
        callback(data);
      } catch (error) {
        logger.error(`Event callback error for ${event}: ${error}`);
      }
    });
  }
  
  // ==================== 加载状态 ====================
  
  /**
   * 设置加载状态
   * 
   * 当 loading 为 true 时，会触发 analysis-started 事件
   * 事件携带 sessionId、messageId、requestId
   */
  setLoading(loading: boolean, requestId?: string, messageId?: string): void {
    const prevLoading = this.state.isLoading;
    this.state.isLoading = loading;
    
    logger.debug(`[setLoading] Loading state changed: ${prevLoading} -> ${loading}, requestId=${requestId || 'none'}, messageId=${messageId || 'none'}`);
    
    if (loading && requestId) {
      this.state.pendingRequestId = requestId;
      
      // 如果提供了 messageId，更新当前消息
      if (messageId) {
        this.state.currentMessageId = messageId;
      }
      
      // 触发 analysis-started 事件
      const sessionId = this.state.currentSessionId || '';
      const currentMessageId = messageId || this.state.currentMessageId || '';
      
      this.emit('analysis-started', {
        sessionId,
        messageId: currentMessageId,
        requestId,
      });
      
      logger.info(`[setLoading] Analysis started: session=${sessionId}, message=${currentMessageId}, request=${requestId}`);
    } else if (!loading) {
      const clearedRequestId = this.state.pendingRequestId;
      this.state.pendingRequestId = null;
      logger.info(`[setLoading] Loading cleared, previous requestId=${clearedRequestId || 'none'}`);
    }
    
    this.notifySubscribers();
  }
  
  /**
   * 检查是否正在加载
   */
  isLoading(): boolean {
    return this.state.isLoading;
  }
  
  /**
   * 获取pending请求ID
   */
  getPendingRequestId(): string | null {
    return this.state.pendingRequestId;
  }
  
  // ==================== 错误处理 ====================
  
  /**
   * 设置错误（简单字符串版本）
   * 
   * 此方法会自动创建增强的错误信息，包括恢复建议
   * 
   * Validates: Requirement 4.4 - 错误时显示友好的错误信息
   */
  setError(error: string | null): void {
    const prevError = this.state.error;
    
    if (error) {
      // 创建增强的错误信息，使用默认错误代码
      this.errorInfo = createEnhancedErrorInfo(
        ErrorCodes.ANALYSIS_ERROR,
        error
      );
      this.state.error = formatErrorForDisplay(this.errorInfo);
      this.state.isLoading = false;
      this.state.pendingRequestId = null;
      
      logger.warn(`[setError] Error set: "${error}"`);
      logger.debug(`[setError] Enhanced error info: code=${this.errorInfo.code}, suggestions=${this.errorInfo.recoverySuggestions.length}`);
    } else {
      this.errorInfo = null;
      this.state.error = null;
      
      if (prevError) {
        logger.info(`[setError] Error cleared, previous error: "${prevError}"`);
      }
    }
    
    this.notifySubscribers();
  }
  
  /**
   * 设置增强的错误信息
   * 
   * 此方法接受完整的错误信息对象，包括错误代码、消息、详情和恢复建议
   * 通常由 AnalysisResultBridge 在接收到后端错误事件时调用
   * 
   * Validates: Requirement 4.4 - 错误时显示友好的错误信息
   */
  setErrorWithInfo(errorInfo: EnhancedErrorInfo | null): void {
    const prevError = this.state.error;
    
    if (errorInfo) {
      // 如果没有恢复建议，根据错误代码生成
      if (!errorInfo.recoverySuggestions || errorInfo.recoverySuggestions.length === 0) {
        errorInfo = createEnhancedErrorInfo(
          errorInfo.code,
          errorInfo.message,
          errorInfo.details,
          []
        );
      }
      
      this.errorInfo = errorInfo;
      this.state.error = formatErrorForDisplay(errorInfo);
      this.state.isLoading = false;
      this.state.pendingRequestId = null;
      
      logger.warn(`[setErrorWithInfo] Error set: code=${errorInfo.code}, message="${errorInfo.message}"`);
      logger.debug(`[setErrorWithInfo] Recovery suggestions: ${errorInfo.recoverySuggestions.join('; ')}`);
      
      if (errorInfo.details) {
        logger.debug(`[setErrorWithInfo] Error details: ${errorInfo.details}`);
      }
    } else {
      this.errorInfo = null;
      this.state.error = null;
      
      if (prevError) {
        logger.info(`[setErrorWithInfo] Error cleared, previous error: "${prevError}"`);
      }
    }
    
    this.notifySubscribers();
  }
  
  /**
   * 获取错误消息字符串
   */
  getError(): string | null {
    return this.state.error;
  }
  
  /**
   * 获取增强的错误信息
   * 
   * 返回完整的错误信息对象，包括错误代码、消息、详情和恢复建议
   * 
   * Validates: Requirement 4.4 - 错误时显示友好的错误信息
   */
  getErrorInfo(): EnhancedErrorInfo | null {
    return this.errorInfo;
  }
  
  // ==================== 历史请求空结果处理 ====================
  
  /**
   * 通知历史请求无结果
   * 
   * 当历史分析请求没有关联的分析结果时调用
   * 触发 historical-empty-result 事件，通知 useDashboardData 显示空状态而非数据源统计
   * 
   * Validates: Requirement 2.4
   */
  notifyHistoricalEmptyResult(sessionId: string, messageId: string): void {
    logger.info(`Historical request has no results: session=${sessionId}, message=${messageId}`);
    
    // 触发 historical-empty-result 事件
    this.emit('historical-empty-result', {
      sessionId,
      messageId,
    });
  }
  
  // ==================== 状态获取 ====================
  
  /**
   * 获取完整状态（深拷贝）
   */
  getState(): AnalysisResultState {
    // 创建data的深拷贝
    const dataCopy = new Map<string, Map<string, AnalysisResultItem[]>>();
    this.state.data.forEach((sessionData, sessionId) => {
      const sessionCopy = new Map<string, AnalysisResultItem[]>();
      sessionData.forEach((items, messageId) => {
        sessionCopy.set(messageId, [...items]);
      });
      dataCopy.set(sessionId, sessionCopy);
    });
    
    return {
      ...this.state,
      data: dataCopy,
    };
  }
  
  // ==================== 辅助方法 ====================
  
  /**
   * 从原始数据创建AnalysisResultItem
   */
  createItem(
    type: AnalysisResultType,
    data: any,
    metadata: Partial<ResultMetadata>,
    source: ResultSource = 'realtime'
  ): AnalysisResultItem | null {
    const normalizedResult = DataNormalizer.normalize(type, data);
    
    if (!normalizedResult.success) {
      logger.warn(`Failed to create item: ${normalizedResult.error}`);
      return null;
    }
    
    return {
      id: generateId(),
      type,
      data: normalizedResult.data,
      metadata: {
        sessionId: metadata.sessionId || '',
        messageId: metadata.messageId || '',
        timestamp: metadata.timestamp || Date.now(),
        requestId: metadata.requestId,
        fileName: metadata.fileName,
        mimeType: metadata.mimeType,
      },
      source,
    };
  }
  
  /**
   * 批量添加原始数据
   */
  addRawResults(
    sessionId: string,
    messageId: string,
    requestId: string,
    rawItems: Array<{ type: AnalysisResultType; data: any; fileName?: string }>,
    isComplete: boolean = false
  ): void {
    const items: AnalysisResultItem[] = [];
    
    for (const raw of rawItems) {
      const item = this.createItem(raw.type, raw.data, {
        sessionId,
        messageId,
        requestId,
        fileName: raw.fileName,
      });
      
      if (item) {
        items.push(item);
      }
    }
    
    if (items.length > 0) {
      this.updateResults({
        sessionId,
        messageId,
        requestId,
        items,
        isComplete,
        timestamp: Date.now(),
      });
    }
  }
}

// 导出单例获取函数
export function getAnalysisResultManager(): IAnalysisResultManager {
  return AnalysisResultManagerImpl.getInstance();
}

// 导出类（用于测试）
export { AnalysisResultManagerImpl };

// 默认导出
export default getAnalysisResultManager;
