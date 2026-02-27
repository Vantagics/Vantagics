/**
 * Analysis Result Bridge
 * 
 * 初始化新的统一事件系统监听器
 * 连接后端事件到 AnalysisResultManager
 */

import {
  AnalysisResultBatch,
  AnalysisResultItem,
  AnalysisErrorPayload,
  EnhancedErrorInfo,
  ErrorCodes,
} from '../types/AnalysisResult';
import { getAnalysisResultManager } from '../managers/AnalysisResultManager';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import { createLogger } from './systemLog';

const logger = createLogger('AnalysisResultBridge');

let bridgeInitialized = false;

/**
 * 初始化分析结果事件监听
 * 
 * @param getCurrentSessionId - 获取当前会话ID的回调
 * @param getCurrentMessageId - 获取当前消息ID的回调
 * @returns 清理函数，用于取消所有事件监听
 */
export function initAnalysisResultBridge(
  getCurrentSessionId: () => string | null,
  getCurrentMessageId: () => string | null
): () => void {
  if (bridgeInitialized) {
    logger.warn('Bridge already initialized');
    return () => {};
  }
  
  const manager = getAnalysisResultManager();
  const unsubscribers: (() => void)[] = [];
  
  // 监听 analysis-result-update 事件
  const unsubscribeUpdate = EventsOn('analysis-result-update', (payload: AnalysisResultBatch) => {
    // 详细记录接收到的数据类型分布
    const typeDistribution: Record<string, number> = {};
    if (payload.items) {
      payload.items.forEach(item => {
        typeDistribution[item.type] = (typeDistribution[item.type] || 0) + 1;
      });
    }
    logger.debug(`[EventReceived] analysis-result-update: session=${payload.sessionId}, message=${payload.messageId}, items=${payload.items?.length || 0}, complete=${payload.isComplete}`);
    logger.debug(`[EventReceived] Item types: ${JSON.stringify(typeDistribution)}`);
    
    // NOTE: Do NOT call switchSession/selectMessage here before updateResults.
    // processBatch already handles auto-switching session/message internally.
    // Calling switchSession/selectMessage here would trigger session-switched/message-selected
    // events that clear dataSourceStatistics in useDashboardData, causing a race condition
    // where startup data source info gets cleared before it can be displayed.
    
    manager.updateResults(payload);
  });
  unsubscribers.push(unsubscribeUpdate);
  
  // 监听 analysis-result-clear 事件
  const unsubscribeClear = EventsOn('analysis-result-clear', (payload: { sessionId: string; messageId?: string }) => {
    logger.debug(`[EventReceived] analysis-result-clear: session=${payload.sessionId}, message=${payload.messageId || 'all'}`);
    manager.clearResults(payload.sessionId, payload.messageId);
  });
  unsubscribers.push(unsubscribeClear);
  
  // 监听 analysis-result-loading 事件
  const unsubscribeLoading = EventsOn('analysis-result-loading', (payload: { sessionId: string; loading: boolean; requestId?: string }) => {
    logger.debug(`[EventReceived] analysis-result-loading: session=${payload.sessionId}, loading=${payload.loading}, requestId=${payload.requestId || 'none'}`);
    manager.setLoading(payload.loading, payload.requestId);
  });
  unsubscribers.push(unsubscribeLoading);
  
  // 监听 analysis-result-error 事件
  // 
  // 增强的错误处理 (Requirement 4.4):
  // 1. 接收后端发送的增强错误信息（包括错误代码和恢复建议）
  // 2. 使用 setErrorWithInfo 方法设置完整的错误信息
  // 3. 如果后端没有提供恢复建议，前端会根据错误代码自动生成
  const unsubscribeError = EventsOn('analysis-result-error', (payload: AnalysisErrorPayload) => {
    logger.warn(`[EventReceived] analysis-result-error: session=${payload.sessionId}, requestId=${payload.requestId || 'none'}, code=${payload.code || 'unknown'}`);
    logger.debug(`[EventReceived] Error message: ${payload.error || payload.message}`);
    
    if (payload.recoverySuggestions && payload.recoverySuggestions.length > 0) {
      logger.debug(`[EventReceived] Recovery suggestions from backend: ${payload.recoverySuggestions.join('; ')}`);
    }
    
    // 创建增强的错误信息
    const errorInfo: EnhancedErrorInfo = {
      code: payload.code || ErrorCodes.ANALYSIS_ERROR,
      message: payload.error || payload.message || '发生未知错误',
      details: payload.details,
      recoverySuggestions: payload.recoverySuggestions || [],
      timestamp: payload.timestamp || Date.now(),
    };
    
    // 使用增强的错误处理方法
    manager.setErrorWithInfo(errorInfo);
  });
  unsubscribers.push(unsubscribeError);
  
  // 监听 analysis-error 事件（兼容旧事件名）
  // 
  // 增强的错误处理 (Requirement 4.4):
  // 与 analysis-result-error 相同的处理逻辑
  const unsubscribeAnalysisError = EventsOn('analysis-error', (payload: AnalysisErrorPayload) => {
    logger.warn(`[EventReceived] analysis-error: session=${payload.sessionId}, requestId=${payload.requestId || 'none'}, code=${payload.code || 'unknown'}`);
    logger.debug(`[EventReceived] Error message: ${payload.error || payload.message}`);
    
    if (payload.recoverySuggestions && payload.recoverySuggestions.length > 0) {
      logger.debug(`[EventReceived] Recovery suggestions from backend: ${payload.recoverySuggestions.join('; ')}`);
    }
    
    // 创建增强的错误信息
    const errorInfo: EnhancedErrorInfo = {
      code: payload.code || ErrorCodes.ANALYSIS_ERROR,
      message: payload.error || payload.message || '发生未知错误',
      details: payload.details,
      recoverySuggestions: payload.recoverySuggestions || [],
      timestamp: payload.timestamp || Date.now(),
    };
    
    // 使用增强的错误处理方法
    manager.setErrorWithInfo(errorInfo);
  });
  unsubscribers.push(unsubscribeAnalysisError);
  
  // 监听 analysis-cancelled 事件，清除仪表盘的加载状态
  const unsubscribeCancelled = EventsOn('analysis-cancelled', (payload: { threadId: string; message?: string }) => {
    logger.debug(`[EventReceived] analysis-cancelled: threadId=${payload.threadId}`);
    manager.setLoading(false);
  });
  unsubscribers.push(unsubscribeCancelled);
  
  // 监听 analysis-result-restore 事件（用于恢复历史数据）
  // 
  // 改进的历史数据恢复逻辑 (Requirement 5.3):
  // 1. 使用 AnalysisResultManager.restoreResults 方法进行恢复
  // 2. 该方法会验证数据完整性并记录详细日志
  // 3. 无结果时显示空状态而非数据源统计 (Requirement 2.4)
  const unsubscribeRestore = EventsOn('analysis-result-restore', (payload: {
    sessionId: string;
    messageId: string;
    items: AnalysisResultItem[];
  }) => {
    logger.warn(`analysis-result-restore received: session=${payload.sessionId}, message=${payload.messageId}, items=${payload.items?.length || 0}`);
    
    // 详细记录每个 item 的类型和数据格式（lightweight - avoid JSON.stringify on large objects）
    if (payload.items) {
      payload.items.forEach((item, i) => {
        const dataType = typeof item.data;
        const dataPreview = dataType === 'string' 
          ? (item.data as string).substring(0, 120) + '...'
          : `(object, keys: ${Object.keys(item.data || {}).slice(0, 5).join(',')})`;
        logger.warn(`analysis-result-restore item[${i}]: type=${item.type}, dataType=${dataType}, preview=${dataPreview}`);
      });
    }
    
    // 使用 restoreResults 方法进行恢复
    const stats = manager.restoreResults(payload.sessionId, payload.messageId, payload.items);
    
    logger.warn(`Historical data restoration completed: valid=${stats.validItems}, invalid=${stats.invalidItems}, total=${stats.totalItems}`);
    
    if (stats.errors.length > 0) {
      stats.errors.forEach((err, i) => {
        logger.warn(`Restoration error[${i}]: ${err}`);
      });
    }
  });
  unsubscribers.push(unsubscribeRestore);
  
  // 监听 analysis-session-created 事件（后端创建分析线程后发出）
  // 
  // Bug Fix (Requirements 2.1, 2.2, 2.3, 2.4):
  // 1. 设置 currentSessionId 使后续分析结果能正确关联到会话
  // 2. 设置加载状态，避免 isUserMessageCancelled 误判
  // 3. 通知 ChatSidebar 切换到新线程并刷新线程列表
  const unsubscribeSessionCreated = EventsOn('analysis-session-created', (payload: { threadId: string; dataSourceId: string }) => {
    logger.debug(`[EventReceived] analysis-session-created: threadId=${payload.threadId}, dataSourceId=${payload.dataSourceId}`);
    
    manager.switchSession(payload.threadId);
    manager.setLoading(true);
    
    EventsEmit('switch-to-session', { threadId: payload.threadId, openChat: true });
    EventsEmit('thread-updated', {});
  });
  unsubscribers.push(unsubscribeSessionCreated);
  
  bridgeInitialized = true;
  logger.info('Analysis result bridge initialized successfully');
  
  // 返回清理函数
  return () => {
    logger.info('Destroying analysis result bridge');
    
    for (const unsubscribe of unsubscribers) {
      unsubscribe();
    }
    
    bridgeInitialized = false;
    logger.info('Analysis result bridge destroyed');
  };
}

/**
 * 检查Bridge是否已初始化
 */
export function isBridgeInitialized(): boolean {
  return bridgeInitialized;
}

/**
 * 重置Bridge状态（用于测试）
 */
export function resetBridge(): void {
  bridgeInitialized = false;
}

export default initAnalysisResultBridge;
