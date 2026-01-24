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
    
    logger.debug(`Processing batch: session=${sessionId}, message=${messageId}, items=${items.length}, complete=${isComplete}`);
    
    // 检查requestId是否匹配（如果有pendingRequestId）
    if (this.state.pendingRequestId && requestId !== this.state.pendingRequestId) {
      logger.info(`Ignoring stale batch: received=${requestId}, expected=${this.state.pendingRequestId}`);
      return;
    }
    
    // 获取或创建session数据
    if (!this.state.data.has(sessionId)) {
      this.state.data.set(sessionId, new Map());
    }
    const sessionData = this.state.data.get(sessionId)!;
    
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
   */
  clearResults(sessionId: string, messageId?: string): void {
    if (messageId) {
      // 清除特定消息的数据
      const sessionData = this.state.data.get(sessionId);
      if (sessionData) {
        sessionData.delete(messageId);
        logger.debug(`Cleared results for message: ${messageId}`);
      }
    } else {
      // 清除整个会话的数据
      this.state.data.delete(sessionId);
      logger.debug(`Cleared results for session: ${sessionId}`);
    }
    
    this.notifySubscribers();
  }
  
  /**
   * 清除所有数据
   */
  clearAll(): void {
    this.state.data.clear();
    this.state.currentSessionId = null;
    this.state.currentMessageId = null;
    this.state.isLoading = false;
    this.state.pendingRequestId = null;
    this.state.error = null;
    
    logger.debug('Cleared all results');
    this.notifySubscribers();
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
   */
  switchSession(sessionId: string): void {
    if (this.state.currentSessionId === sessionId) {
      return;
    }
    
    logger.debug(`Switching session: ${this.state.currentSessionId} -> ${sessionId}`);
    
    // 取消当前的pending请求
    if (this.state.pendingRequestId) {
      logger.info(`Canceling pending request due to session switch: ${this.state.pendingRequestId}`);
      this.state.pendingRequestId = null;
      this.state.isLoading = false;
    }
    
    this.state.currentSessionId = sessionId;
    this.state.currentMessageId = null; // 重置消息选择
    this.state.error = null;
    
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
   */
  selectMessage(messageId: string): void {
    if (this.state.currentMessageId === messageId) {
      return;
    }
    
    logger.debug(`Selecting message: ${this.state.currentMessageId} -> ${messageId}`);
    
    this.state.currentMessageId = messageId;
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
    
    // 返回取消订阅函数
    return () => {
      this.subscribers.delete(callback);
    };
  }
  
  /**
   * 通知所有订阅者
   */
  private notifySubscribers(): void {
    const stateCopy = this.getState();
    this.subscribers.forEach(callback => {
      try {
        callback(stateCopy);
      } catch (error) {
        logger.error(`Subscriber callback error: ${error}`);
      }
    });
  }
  
  // ==================== 加载状态 ====================
  
  /**
   * 设置加载状态
   */
  setLoading(loading: boolean, requestId?: string): void {
    this.state.isLoading = loading;
    
    if (loading && requestId) {
      this.state.pendingRequestId = requestId;
    } else if (!loading) {
      this.state.pendingRequestId = null;
    }
    
    logger.debug(`Loading state: ${loading}, requestId: ${requestId || 'none'}`);
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
   * 设置错误
   */
  setError(error: string | null): void {
    this.state.error = error;
    
    if (error) {
      this.state.isLoading = false;
      this.state.pendingRequestId = null;
      logger.warn(`Error set: ${error}`);
    }
    
    this.notifySubscribers();
  }
  
  /**
   * 获取错误
   */
  getError(): string | null {
    return this.state.error;
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
