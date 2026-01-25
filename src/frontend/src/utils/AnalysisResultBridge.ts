/**
 * Analysis Result Bridge
 * 
 * 初始化新的统一事件系统监听器
 * 连接后端事件到 AnalysisResultManager
 */

import {
  AnalysisResultBatch,
  AnalysisResultItem,
  ResultSource,
} from '../types/AnalysisResult';
import { getAnalysisResultManager } from '../managers/AnalysisResultManager';
import { EventsOn } from '../../wailsjs/runtime/runtime';
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
    logger.debug(`analysis-result-update: ${payload.items?.length || 0} items, session=${payload.sessionId}`);
    
    // 同步会话和消息ID
    if (payload.sessionId && payload.sessionId !== manager.getCurrentSession()) {
      manager.switchSession(payload.sessionId);
    }
    if (payload.messageId && payload.messageId !== manager.getCurrentMessage()) {
      manager.selectMessage(payload.messageId);
    }
    
    manager.updateResults(payload);
  });
  unsubscribers.push(unsubscribeUpdate);
  
  // 监听 analysis-result-clear 事件
  const unsubscribeClear = EventsOn('analysis-result-clear', (payload: { sessionId: string; messageId?: string }) => {
    logger.debug(`analysis-result-clear: session=${payload.sessionId}`);
    manager.clearResults(payload.sessionId, payload.messageId);
  });
  unsubscribers.push(unsubscribeClear);
  
  // 监听 analysis-result-loading 事件
  const unsubscribeLoading = EventsOn('analysis-result-loading', (payload: { sessionId: string; loading: boolean; requestId?: string }) => {
    logger.debug(`analysis-result-loading: loading=${payload.loading}, requestId=${payload.requestId || 'none'}`);
    manager.setLoading(payload.loading, payload.requestId);
  });
  unsubscribers.push(unsubscribeLoading);
  
  // 监听 analysis-result-error 事件
  const unsubscribeError = EventsOn('analysis-result-error', (payload: { sessionId: string; error: string; requestId?: string }) => {
    logger.warn(`analysis-result-error: ${payload.error}`);
    manager.setError(payload.error);
  });
  unsubscribers.push(unsubscribeError);
  
  // 监听 analysis-result-restore 事件（用于恢复历史数据）
  // 
  // 改进的历史数据恢复逻辑:
  // 1. 先清除当前显示的所有数据 (Requirement 2.1)
  // 2. 确保只显示恢复的数据 (Requirement 2.2)
  // 3. 无结果时显示空状态而非数据源统计 (Requirement 2.4)
  const unsubscribeRestore = EventsOn('analysis-result-restore', (payload: {
    sessionId: string;
    messageId: string;
    items: AnalysisResultItem[];
  }) => {
    logger.info(`analysis-result-restore: session=${payload.sessionId}, message=${payload.messageId}, items=${payload.items?.length || 0}`);
    
    // Step 1: 先清除当前会话的所有数据，确保数据隔离 (Requirement 2.1)
    // 这样可以避免新旧数据混合显示
    const currentSessionId = manager.getCurrentSession();
    if (currentSessionId) {
      logger.debug(`Clearing current session data before restore: ${currentSessionId}`);
      manager.clearResults(currentSessionId);
    }
    
    // Step 2: 切换到目标会话（如果不同）
    if (payload.sessionId !== currentSessionId) {
      logger.debug(`Switching to restore target session: ${payload.sessionId}`);
      manager.switchSession(payload.sessionId);
    }
    
    // Step 3: 清除目标会话的数据（确保干净状态）
    // 即使切换了会话，也要确保目标会话是干净的
    manager.clearResults(payload.sessionId);
    
    // Step 4: 选择目标消息
    if (payload.messageId !== manager.getCurrentMessage()) {
      logger.debug(`Selecting restore target message: ${payload.messageId}`);
      manager.selectMessage(payload.messageId);
    }
    
    // Step 5: 处理恢复的数据
    if (!payload.items || payload.items.length === 0) {
      logger.debug('No items to restore, notifying historical empty result for empty state display');
      // 通知历史请求无结果，以便 useDashboardData 显示空状态而非数据源统计 (Requirement 2.4)
      manager.notifyHistoricalEmptyResult(payload.sessionId, payload.messageId);
      return;
    }
    
    // Step 6: 标记为恢复的数据并更新管理器 (Requirement 2.2)
    const restoredItems: AnalysisResultItem[] = payload.items.map(item => ({
      ...item,
      source: 'restored' as ResultSource,
    }));
    
    // 更新管理器 - 此时仪表盘只会显示恢复的数据
    manager.updateResults({
      sessionId: payload.sessionId,
      messageId: payload.messageId,
      requestId: `restore_${Date.now()}`,
      items: restoredItems,
      isComplete: true,
      timestamp: Date.now(),
    });
    
    logger.info(`Historical data restored successfully: ${restoredItems.length} items`);
  });
  unsubscribers.push(unsubscribeRestore);
  
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
