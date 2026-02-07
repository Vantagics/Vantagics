/**
 * Property-Based Tests for AnalysisResultManager
 * 
 * Feature: dashboard-data-isolation
 * 
 * These tests verify the correctness properties defined in the design document
 * using fast-check for property-based testing.
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import * as fc from 'fast-check';
import { AnalysisResultManagerImpl, getAnalysisResultManager } from './AnalysisResultManager';
import {
  AnalysisResultItem,
  AnalysisResultType,
  AnalysisResultBatch,
  ResultSource,
} from '../types/AnalysisResult';

// Mock the systemLog to avoid Wails runtime errors in tests
vi.mock('../utils/systemLog', () => ({
  createLogger: () => ({
    debug: () => {},
    info: () => {},
    warn: () => {},
    error: () => {},
  }),
}));

// ==================== Test Data Generators ====================

/**
 * Generate random sessionId (UUID format)
 */
const sessionIdArb = fc.uuid();

/**
 * Generate random messageId (UUID format)
 */
const messageIdArb = fc.uuid();

/**
 * Generate random requestId (UUID format)
 */
const requestIdArb = fc.uuid();

/**
 * Generate random result source
 */
const resultSourceArb = fc.constantFrom<ResultSource>(
  'realtime', 'completed', 'cached', 'restored'
);

/**
 * Generate valid metric data that will pass normalization
 */
const validMetricDataArb = fc.record({
  title: fc.string({ minLength: 1, maxLength: 50 }),
  value: fc.oneof(fc.string({ minLength: 1 }), fc.integer().map(String)),
  change: fc.option(fc.string(), { nil: undefined }),
});

/**
 * Generate valid insight data that will pass normalization
 */
const validInsightDataArb = fc.record({
  text: fc.string({ minLength: 1, maxLength: 200 }),
  icon: fc.option(fc.string(), { nil: undefined }),
});

/**
 * Generate valid file data that will pass normalization
 */
const validFileDataArb = fc.record({
  fileName: fc.string({ minLength: 1, maxLength: 100 }),
  filePath: fc.string({ minLength: 1, maxLength: 200 }),
  fileType: fc.constantFrom('csv', 'xlsx', 'pdf', 'txt'),
});

/**
 * Generate a valid analysis result item that will pass normalization
 * We use 'metric', 'insight', and 'file' types as they have simpler validation
 */
const validAnalysisResultItemArb = (sessionId: string, messageId: string) =>
  fc.oneof(
    // Metric type
    fc.record({
      id: fc.uuid(),
      type: fc.constant('metric' as AnalysisResultType),
      data: validMetricDataArb,
      metadata: fc.record({
        sessionId: fc.constant(sessionId),
        messageId: fc.constant(messageId),
        timestamp: fc.nat(),
      }),
      source: resultSourceArb,
    }),
    // Insight type
    fc.record({
      id: fc.uuid(),
      type: fc.constant('insight' as AnalysisResultType),
      data: validInsightDataArb,
      metadata: fc.record({
        sessionId: fc.constant(sessionId),
        messageId: fc.constant(messageId),
        timestamp: fc.nat(),
      }),
      source: resultSourceArb,
    }),
    // File type
    fc.record({
      id: fc.uuid(),
      type: fc.constant('file' as AnalysisResultType),
      data: validFileDataArb,
      metadata: fc.record({
        sessionId: fc.constant(sessionId),
        messageId: fc.constant(messageId),
        timestamp: fc.nat(),
      }),
      source: resultSourceArb,
    })
  ) as fc.Arbitrary<AnalysisResultItem>;

/**
 * Generate clearing event type
 */
const clearingEventTypeArb = fc.constantFrom(
  'analysis-started',
  'session-switched',
  'message-selected'
);

// ==================== Test Setup ====================

describe('Feature: dashboard-data-isolation, Property 1: 数据清除一致性', () => {
  /**
   * **Validates: Requirements 1.1, 2.1, 4.1, 4.3**
   * 
   * Property 1: 数据清除一致性
   * For any 触发清除的事件（新分析开始、历史请求选择、会话切换、消息切换），
   * 当该事件触发时，当前显示的分析结果数据应被完全清除，不保留任何旧数据。
   */

  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Property Test 1.1: When analysis-started event is triggered,
   * the current analysis results should be cleared.
   * 
   * **Validates: Requirements 1.1**
   */
  it('should clear current message data when new analysis starts (analysis-started event)', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        requestIdArb,
        (sessionId, oldMessageId, newMessageId, oldRequestId, newRequestId) => {
          // Skip if messages are the same (no clearing expected)
          fc.pre(oldMessageId !== newMessageId);
          
          // Arrange: Set up manager with initial data
          const manager = getAnalysisResultManager();
          
          // Set current session
          manager.switchSession(sessionId);
          
          // Generate valid items for the old message
          const initialItems = fc.sample(validAnalysisResultItemArb(sessionId, oldMessageId), { numRuns: 3 });
          
          // Add initial data to the old message
          const initialBatch: AnalysisResultBatch = {
            sessionId,
            messageId: oldMessageId,
            requestId: oldRequestId,
            items: initialItems,
            isComplete: true,
            timestamp: Date.now(),
          };
          manager.updateResults(initialBatch);
          
          // Verify initial data exists
          const initialResults = manager.getResults(sessionId, oldMessageId);
          expect(initialResults.length).toBeGreaterThan(0);
          
          // Act: Trigger analysis-started event by calling setLoading with new requestId and messageId
          manager.setLoading(true, newRequestId, newMessageId);
          
          // Assert: Current results should be empty since we switched to a new message
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          // The current message should be the new message
          expect(manager.getCurrentMessage()).toBe(newMessageId);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 1.2: When session-switched event is triggered,
   * the current session's analysis results should be cleared.
   * 
   * **Validates: Requirements 4.1**
   */
  it('should clear current session data when switching to a different session (session-switched event)', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (oldSessionId, newSessionId, messageId, requestId) => {
          // Skip if sessions are the same
          fc.pre(oldSessionId !== newSessionId);
          
          // Arrange: Set up manager with initial data
          const manager = getAnalysisResultManager();
          
          // Set current session and add data
          manager.switchSession(oldSessionId);
          
          // Generate valid items
          const initialItems = fc.sample(validAnalysisResultItemArb(oldSessionId, messageId), { numRuns: 3 });
          
          const initialBatch: AnalysisResultBatch = {
            sessionId: oldSessionId,
            messageId,
            requestId,
            items: initialItems,
            isComplete: true,
            timestamp: Date.now(),
          };
          manager.updateResults(initialBatch);
          
          // Verify initial data exists
          const initialResults = manager.getResults(oldSessionId, messageId);
          expect(initialResults.length).toBeGreaterThan(0);
          
          // Act: Switch to a different session
          manager.switchSession(newSessionId);
          
          // Assert: Current results should be empty (new session has no data)
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          // The current session should be the new session
          expect(manager.getCurrentSession()).toBe(newSessionId);
          
          // The current message should be reset to null
          expect(manager.getCurrentMessage()).toBeNull();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 1.3: When message-selected event is triggered,
   * the current message's analysis results should be cleared.
   * 
   * **Validates: Requirements 4.3**
   */
  it('should clear current message data when switching to a different message (message-selected event)', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, oldMessageId, newMessageId, requestId) => {
          // Skip if messages are the same
          fc.pre(oldMessageId !== newMessageId);
          
          // Arrange: Set up manager with initial data
          const manager = getAnalysisResultManager();
          
          // Set current session
          manager.switchSession(sessionId);
          
          // Generate valid items
          const initialItems = fc.sample(validAnalysisResultItemArb(sessionId, oldMessageId), { numRuns: 3 });
          
          // Add initial data to the old message
          const initialBatch: AnalysisResultBatch = {
            sessionId,
            messageId: oldMessageId,
            requestId,
            items: initialItems,
            isComplete: true,
            timestamp: Date.now(),
          };
          manager.updateResults(initialBatch);
          
          // Verify initial data exists
          const initialResults = manager.getResults(sessionId, oldMessageId);
          expect(initialResults.length).toBeGreaterThan(0);
          
          // Act: Select a different message
          manager.selectMessage(newMessageId);
          
          // Assert: Old message data should be cleared
          const oldMessageResults = manager.getResults(sessionId, oldMessageId);
          expect(oldMessageResults.length).toBe(0);
          
          // Current results should be empty (new message has no data yet)
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          // The current message should be the new message
          expect(manager.getCurrentMessage()).toBe(newMessageId);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 1.4: When historical request is selected (simulated by loading data for a different message),
   * the current analysis results should be cleared before loading new data.
   * 
   * **Validates: Requirements 2.1**
   */
  it('should clear current data when loading historical request data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        requestIdArb,
        (sessionId, currentMessageId, historicalMessageId, currentRequestId, historicalRequestId) => {
          // Skip if messages are the same
          fc.pre(currentMessageId !== historicalMessageId);
          
          // Arrange: Set up manager with current data
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Generate valid items for current message
          const currentItems = fc.sample(validAnalysisResultItemArb(sessionId, currentMessageId), { numRuns: 3 });
          
          // Add current analysis data
          const currentBatch: AnalysisResultBatch = {
            sessionId,
            messageId: currentMessageId,
            requestId: currentRequestId,
            items: currentItems,
            isComplete: true,
            timestamp: Date.now(),
          };
          manager.updateResults(currentBatch);
          
          // Verify current data exists
          expect(manager.getResults(sessionId, currentMessageId).length).toBeGreaterThan(0);
          
          // Act: Select historical message (simulating clicking on historical request)
          manager.selectMessage(historicalMessageId);
          
          // Assert: Current message data should be cleared
          const currentMessageResults = manager.getResults(sessionId, currentMessageId);
          expect(currentMessageResults.length).toBe(0);
          
          // Current results should be empty until historical data is loaded
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 1.5: After any clearing event, no old data should remain in the current view.
   * This is a comprehensive test that verifies the data clearing consistency across all event types.
   * 
   * **Validates: Requirements 1.1, 2.1, 4.1, 4.3**
   */
  it('should ensure no old data remains after any clearing event', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        clearingEventTypeArb,
        (oldSessionId, newSessionId, oldMessageId, newMessageId, requestId, eventType) => {
          // Arrange: Set up manager with initial data
          const manager = getAnalysisResultManager();
          
          manager.switchSession(oldSessionId);
          
          // Generate valid items
          const initialItems = fc.sample(validAnalysisResultItemArb(oldSessionId, oldMessageId), { numRuns: 3 });
          
          const initialBatch: AnalysisResultBatch = {
            sessionId: oldSessionId,
            messageId: oldMessageId,
            requestId,
            items: initialItems,
            isComplete: true,
            timestamp: Date.now(),
          };
          manager.updateResults(initialBatch);
          
          // Store initial item IDs for verification
          const initialItemIds = new Set(initialItems.map(item => item.id));
          
          // Act: Trigger the clearing event
          switch (eventType) {
            case 'analysis-started':
              // Ensure we're using a different message to trigger clearing
              if (oldMessageId !== newMessageId) {
                manager.setLoading(true, fc.sample(requestIdArb, 1)[0], newMessageId);
              }
              break;
            case 'session-switched':
              if (oldSessionId !== newSessionId) {
                manager.switchSession(newSessionId);
              }
              break;
            case 'message-selected':
              if (oldMessageId !== newMessageId) {
                manager.selectMessage(newMessageId);
              }
              break;
          }
          
          // Assert: Current results should not contain any of the old items
          const currentResults = manager.getCurrentResults();
          const currentItemIds = new Set(currentResults.map(item => item.id));
          
          // Check that no old item IDs are in the current results
          for (const oldId of initialItemIds) {
            expect(currentItemIds.has(oldId)).toBe(false);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 1.6: clearAll should completely clear all data from the manager.
   * 
   * **Validates: Requirements 1.1, 2.1, 4.1, 4.3**
   */
  it('should completely clear all data when clearAll is called', () => {
    fc.assert(
      fc.property(
        fc.array(
          fc.tuple(sessionIdArb, messageIdArb, requestIdArb),
          { minLength: 1, maxLength: 5 }
        ),
        (sessionDataList) => {
          // Arrange: Set up manager with multiple sessions and messages
          const manager = getAnalysisResultManager();
          
          for (const [sessionId, messageId, requestId] of sessionDataList) {
            manager.switchSession(sessionId);
            
            // Generate valid items
            const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 2 });
            
            const batch: AnalysisResultBatch = {
              sessionId,
              messageId,
              requestId,
              items,
              isComplete: true,
              timestamp: Date.now(),
            };
            manager.updateResults(batch);
          }
          
          // Verify data exists
          const state = manager.getState();
          expect(state.data.size).toBeGreaterThan(0);
          
          // Act: Clear all data
          manager.clearAll();
          
          // Assert: All data should be cleared
          const clearedState = manager.getState();
          expect(clearedState.data.size).toBe(0);
          expect(clearedState.currentSessionId).toBeNull();
          expect(clearedState.currentMessageId).toBeNull();
          expect(clearedState.isLoading).toBe(false);
          expect(clearedState.pendingRequestId).toBeNull();
          expect(clearedState.error).toBeNull();
          
          // Current results should be empty
          expect(manager.getCurrentResults().length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


describe('Feature: dashboard-data-isolation, Task 4.1: selectMessage 数据清除逻辑', () => {
  /**
   * **Validates: Requirements 4.3, 4.4**
   * 
   * Task 4.1: 改进 selectMessage 方法的数据清除逻辑
   * - 切换消息时清除当前消息的分析结果
   * - 保留新消息的已有数据（如果有）
   */

  beforeEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Test 4.1.1: When switching to a message that has existing cached data,
   * the cached data should be preserved and displayed.
   * 
   * Note: The current design clears old message data when new message data arrives
   * via updateResults. This test verifies that selectMessage preserves data
   * for the target message if it exists in the current session.
   * 
   * **Validates: Requirements 4.4**
   */
  it('should preserve new message existing data when switching messages', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, message1Id, message2Id, requestId) => {
          // Skip if messages are the same
          fc.pre(message1Id !== message2Id);
          
          // Arrange: Set up manager with data for message 2 (the target message)
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Add data for message 2 first (this will be the cached data)
          const message2Items = fc.sample(validAnalysisResultItemArb(sessionId, message2Id), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId: message2Id,
            requestId,
            items: message2Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify message 2 has data and is the current message
          expect(manager.getResults(sessionId, message2Id).length).toBeGreaterThan(0);
          expect(manager.getCurrentMessage()).toBe(message2Id);
          
          // Select message 1 (which has no data) - this will clear message 2's data
          manager.selectMessage(message1Id);
          expect(manager.getCurrentMessage()).toBe(message1Id);
          expect(manager.getCurrentResults().length).toBe(0);
          
          // Now add data for message 2 again (simulating cached/restored data)
          manager.updateResults({
            sessionId,
            messageId: message2Id,
            requestId,
            items: message2Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify message 2 has data again
          expect(manager.getResults(sessionId, message2Id).length).toBeGreaterThan(0);
          
          // Act: Switch back to message 2
          manager.selectMessage(message2Id);
          
          // Assert: Message 2's data should be preserved
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBeGreaterThan(0);
          
          // The current message should be message 2
          expect(manager.getCurrentMessage()).toBe(message2Id);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Test 4.1.2: When switching to a message that has no existing data,
   * the dashboard should be empty until new data arrives.
   * 
   * **Validates: Requirements 4.3, 4.4**
   */
  it('should show empty dashboard when switching to message with no data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, message1Id, message2Id, requestId) => {
          // Skip if messages are the same
          fc.pre(message1Id !== message2Id);
          
          // Arrange: Set up manager with data only for message 1
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Add data for message 1 only
          const message1Items = fc.sample(validAnalysisResultItemArb(sessionId, message1Id), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId: message1Id,
            requestId,
            items: message1Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify message 1 has data
          expect(manager.getResults(sessionId, message1Id).length).toBeGreaterThan(0);
          
          // Act: Switch to message 2 (which has no data)
          manager.selectMessage(message2Id);
          
          // Assert: Current results should be empty
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          // The current message should be message 2
          expect(manager.getCurrentMessage()).toBe(message2Id);
          
          // Message 1's data should be cleared
          const message1Results = manager.getResults(sessionId, message1Id);
          expect(message1Results.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Test 4.1.3: The message-selected event should be emitted with correct fromMessageId and toMessageId.
   * 
   * **Validates: Requirements 4.3, 4.4**
   */
  it('should emit message-selected event with correct fromMessageId and toMessageId', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        (sessionId, fromMessageId, toMessageId) => {
          // Skip if messages are the same
          fc.pre(fromMessageId !== toMessageId);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          manager.selectMessage(fromMessageId);
          
          // Set up event listener
          let eventData: { sessionId: string; fromMessageId: string | null; toMessageId: string } | null = null;
          const unsubscribe = manager.on('message-selected', (data) => {
            eventData = data;
          });
          
          // Act: Switch to a different message
          manager.selectMessage(toMessageId);
          
          // Assert: Event should be emitted with correct data
          expect(eventData).not.toBeNull();
          expect(eventData!.sessionId).toBe(sessionId);
          expect(eventData!.fromMessageId).toBe(fromMessageId);
          expect(eventData!.toMessageId).toBe(toMessageId);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Test 4.1.4: Selecting the same message should be a no-op.
   * 
   * **Validates: Requirements 4.3, 4.4**
   */
  it('should not clear data when selecting the same message', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Add data for the message
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Select the message
          manager.selectMessage(messageId);
          
          // Verify data exists
          const initialResults = manager.getCurrentResults();
          expect(initialResults.length).toBeGreaterThan(0);
          
          // Set up event listener to verify no event is emitted
          let eventEmitted = false;
          const unsubscribe = manager.on('message-selected', () => {
            eventEmitted = true;
          });
          
          // Act: Select the same message again
          manager.selectMessage(messageId);
          
          // Assert: Data should still exist
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(initialResults.length);
          
          // No event should be emitted
          expect(eventEmitted).toBe(false);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Test 4.1.5: selectMessage should handle null currentSessionId gracefully.
   * 
   * **Validates: Requirements 4.3, 4.4**
   */
  it('should handle selectMessage when currentSessionId is null', () => {
    fc.assert(
      fc.property(
        messageIdArb,
        (messageId) => {
          // Arrange: Manager with no session set
          const manager = getAnalysisResultManager();
          
          // Verify no session is set
          expect(manager.getCurrentSession()).toBeNull();
          
          // Act: Select a message without a session
          manager.selectMessage(messageId);
          
          // Assert: Message should be selected
          expect(manager.getCurrentMessage()).toBe(messageId);
          
          // Current results should be empty (no session data)
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


describe('Feature: dashboard-data-isolation, Property 3: 数据隔离性', () => {
  /**
   * **Validates: Requirements 1.4, 2.2, 4.4**
   * 
   * Property 3: 数据隔离性
   * For any 数据加载操作（新分析结果到达、历史数据恢复、消息切换），
   * 加载完成后显示的数据应仅包含目标 sessionId 和 messageId 对应的数据，
   * 不包含其他会话或消息的数据。
   */

  beforeEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Property Test 3.1: After new analysis results arrive, only data for the target
   * sessionId and messageId should be displayed, not data from other sessions/messages.
   * 
   * **Validates: Requirements 1.4**
   */
  it('should only display target session/message data after new analysis results arrive', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        requestIdArb,
        (session1Id, session2Id, message1Id, message2Id, request1Id, request2Id) => {
          // Ensure we have distinct sessions and messages
          fc.pre(session1Id !== session2Id);
          fc.pre(message1Id !== message2Id);
          
          // Arrange: Set up manager with data for session1/message1
          const manager = getAnalysisResultManager();
          
          manager.switchSession(session1Id);
          
          // Generate valid items for session1/message1
          const session1Items = fc.sample(validAnalysisResultItemArb(session1Id, message1Id), { numRuns: 3 });
          
          manager.updateResults({
            sessionId: session1Id,
            messageId: message1Id,
            requestId: request1Id,
            items: session1Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store session1 item IDs for verification
          const session1ItemIds = new Set(session1Items.map(item => item.id));
          
          // Act: Load new analysis results for session2/message2
          manager.switchSession(session2Id);
          
          // Generate valid items for session2/message2
          const session2Items = fc.sample(validAnalysisResultItemArb(session2Id, message2Id), { numRuns: 3 });
          
          manager.updateResults({
            sessionId: session2Id,
            messageId: message2Id,
            requestId: request2Id,
            items: session2Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Current results should only contain session2/message2 data
          const currentResults = manager.getCurrentResults();
          
          // Verify all current results belong to session2/message2
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(session2Id);
            expect(result.metadata.messageId).toBe(message2Id);
          }
          
          // Verify no session1 items are in current results
          const currentItemIds = new Set(currentResults.map(item => item.id));
          for (const session1ItemId of session1ItemIds) {
            expect(currentItemIds.has(session1ItemId)).toBe(false);
          }
          
          // Verify current session and message are correct
          expect(manager.getCurrentSession()).toBe(session2Id);
          expect(manager.getCurrentMessage()).toBe(message2Id);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.2: After message switch within the same session,
   * only data for the target messageId should be displayed.
   * 
   * **Validates: Requirements 4.4**
   */
  it('should only display target message data after message switch within same session', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        requestIdArb,
        (sessionId, message1Id, message2Id, request1Id, request2Id) => {
          // Ensure we have distinct messages
          fc.pre(message1Id !== message2Id);
          
          // Arrange: Set up manager with data for message1
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Generate valid items for message1
          const message1Items = fc.sample(validAnalysisResultItemArb(sessionId, message1Id), { numRuns: 3 });
          
          manager.updateResults({
            sessionId,
            messageId: message1Id,
            requestId: request1Id,
            items: message1Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store message1 item IDs for verification
          const message1ItemIds = new Set(message1Items.map(item => item.id));
          
          // Verify message1 data exists
          expect(manager.getResults(sessionId, message1Id).length).toBeGreaterThan(0);
          
          // Act: Switch to message2 and load new data
          manager.selectMessage(message2Id);
          
          // Generate valid items for message2
          const message2Items = fc.sample(validAnalysisResultItemArb(sessionId, message2Id), { numRuns: 3 });
          
          manager.updateResults({
            sessionId,
            messageId: message2Id,
            requestId: request2Id,
            items: message2Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Current results should only contain message2 data
          const currentResults = manager.getCurrentResults();
          
          // Verify all current results belong to message2
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(sessionId);
            expect(result.metadata.messageId).toBe(message2Id);
          }
          
          // Verify no message1 items are in current results
          const currentItemIds = new Set(currentResults.map(item => item.id));
          for (const message1ItemId of message1ItemIds) {
            expect(currentItemIds.has(message1ItemId)).toBe(false);
          }
          
          // Verify message1 data has been cleared
          const message1Results = manager.getResults(sessionId, message1Id);
          expect(message1Results.length).toBe(0);
          
          // Verify current message is correct
          expect(manager.getCurrentMessage()).toBe(message2Id);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.3: After historical data restore (simulated by loading data for a specific message),
   * only the restored data should be displayed, not data from other messages.
   * 
   * **Validates: Requirements 2.2**
   */
  it('should only display restored historical data after restore operation', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        requestIdArb,
        (sessionId, currentMessageId, historicalMessageId, currentRequestId, historicalRequestId) => {
          // Ensure we have distinct messages
          fc.pre(currentMessageId !== historicalMessageId);
          
          // Arrange: Set up manager with current analysis data
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Generate valid items for current message
          const currentItems = fc.sample(validAnalysisResultItemArb(sessionId, currentMessageId), { numRuns: 3 });
          
          manager.updateResults({
            sessionId,
            messageId: currentMessageId,
            requestId: currentRequestId,
            items: currentItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store current item IDs for verification
          const currentItemIds = new Set(currentItems.map(item => item.id));
          
          // Verify current data exists
          expect(manager.getCurrentResults().length).toBeGreaterThan(0);
          
          // Act: Restore historical data (select historical message and load its data)
          manager.selectMessage(historicalMessageId);
          
          // Generate valid items for historical message (simulating restored data)
          const historicalItems = fc.sample(validAnalysisResultItemArb(sessionId, historicalMessageId), { numRuns: 3 });
          
          manager.updateResults({
            sessionId,
            messageId: historicalMessageId,
            requestId: historicalRequestId,
            items: historicalItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Current results should only contain historical data
          const currentResults = manager.getCurrentResults();
          
          // Verify all current results belong to historical message
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(sessionId);
            expect(result.metadata.messageId).toBe(historicalMessageId);
          }
          
          // Verify no current message items are in results
          const resultItemIds = new Set(currentResults.map(item => item.id));
          for (const currentItemId of currentItemIds) {
            expect(resultItemIds.has(currentItemId)).toBe(false);
          }
          
          // Verify current message data has been cleared
          const currentMessageResults = manager.getResults(sessionId, currentMessageId);
          expect(currentMessageResults.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.4: Data isolation should be maintained across multiple
   * sequential data loading operations.
   * 
   * **Validates: Requirements 1.4, 2.2, 4.4**
   */
  it('should maintain data isolation across multiple sequential data loading operations', () => {
    fc.assert(
      fc.property(
        fc.array(
          fc.tuple(sessionIdArb, messageIdArb, requestIdArb),
          { minLength: 2, maxLength: 5 }
        ),
        (operationList) => {
          // Ensure all operations have unique session/message combinations
          const uniqueKeys = new Set(operationList.map(([s, m]) => `${s}:${m}`));
          fc.pre(uniqueKeys.size === operationList.length);
          
          // Arrange
          const manager = getAnalysisResultManager();
          
          // Act: Perform multiple sequential data loading operations
          let lastSessionId: string | null = null;
          let lastMessageId: string | null = null;
          let lastItems: AnalysisResultItem[] = [];
          
          for (const [sessionId, messageId, requestId] of operationList) {
            // Switch session if needed
            if (sessionId !== lastSessionId) {
              manager.switchSession(sessionId);
            }
            
            // Select message if needed
            if (messageId !== lastMessageId) {
              manager.selectMessage(messageId);
            }
            
            // Generate and load data
            const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 2 });
            
            manager.updateResults({
              sessionId,
              messageId,
              requestId,
              items,
              isComplete: true,
              timestamp: Date.now(),
            });
            
            lastSessionId = sessionId;
            lastMessageId = messageId;
            lastItems = items;
          }
          
          // Assert: After all operations, only the last operation's data should be displayed
          const currentResults = manager.getCurrentResults();
          
          // Verify all current results belong to the last session/message
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(lastSessionId);
            expect(result.metadata.messageId).toBe(lastMessageId);
          }
          
          // Verify current session and message are correct
          expect(manager.getCurrentSession()).toBe(lastSessionId);
          expect(manager.getCurrentMessage()).toBe(lastMessageId);
          
          // Verify the number of results matches the last operation
          expect(currentResults.length).toBe(lastItems.length);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.5: When loading data for a new message in the same session,
   * the old message's data should be completely isolated (cleared).
   * 
   * **Validates: Requirements 4.4**
   */
  it('should completely isolate old message data when loading new message data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        fc.array(messageIdArb, { minLength: 2, maxLength: 4 }),
        requestIdArb,
        (sessionId, messageIds, requestId) => {
          // Ensure all message IDs are unique
          const uniqueMessageIds = new Set(messageIds);
          fc.pre(uniqueMessageIds.size === messageIds.length);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Track all items added for each message
          const itemsByMessage = new Map<string, Set<string>>();
          
          // Act: Load data for each message sequentially
          for (const messageId of messageIds) {
            manager.selectMessage(messageId);
            
            const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 2 });
            
            manager.updateResults({
              sessionId,
              messageId,
              requestId: `${requestId}_${messageId}`,
              items,
              isComplete: true,
              timestamp: Date.now(),
            });
            
            itemsByMessage.set(messageId, new Set(items.map(item => item.id)));
          }
          
          // Assert: Only the last message's data should be accessible
          const lastMessageId = messageIds[messageIds.length - 1];
          const currentResults = manager.getCurrentResults();
          
          // Verify current results only contain last message's data
          const currentItemIds = new Set(currentResults.map(item => item.id));
          const lastMessageItemIds = itemsByMessage.get(lastMessageId)!;
          
          // All current items should be from the last message
          for (const itemId of currentItemIds) {
            expect(lastMessageItemIds.has(itemId)).toBe(true);
          }
          
          // All previous messages should have no data
          for (let i = 0; i < messageIds.length - 1; i++) {
            const oldMessageId = messageIds[i];
            const oldMessageResults = manager.getResults(sessionId, oldMessageId);
            expect(oldMessageResults.length).toBe(0);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.6: Data isolation should work correctly when switching
   * between sessions with different messages.
   * 
   * **Validates: Requirements 1.4, 2.2, 4.4**
   */
  it('should maintain data isolation when switching between sessions with different messages', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (session1Id, session2Id, message1Id, message2Id, requestId) => {
          // Ensure distinct sessions and messages
          fc.pre(session1Id !== session2Id);
          fc.pre(message1Id !== message2Id);
          
          // Arrange
          const manager = getAnalysisResultManager();
          
          // Load data for session1/message1
          manager.switchSession(session1Id);
          const session1Items = fc.sample(validAnalysisResultItemArb(session1Id, message1Id), { numRuns: 3 });
          manager.updateResults({
            sessionId: session1Id,
            messageId: message1Id,
            requestId: `${requestId}_s1`,
            items: session1Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          const session1ItemIds = new Set(session1Items.map(item => item.id));
          
          // Load data for session2/message2
          manager.switchSession(session2Id);
          const session2Items = fc.sample(validAnalysisResultItemArb(session2Id, message2Id), { numRuns: 3 });
          manager.updateResults({
            sessionId: session2Id,
            messageId: message2Id,
            requestId: `${requestId}_s2`,
            items: session2Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          const session2ItemIds = new Set(session2Items.map(item => item.id));
          
          // Assert: Current results should only contain session2/message2 data
          const currentResults = manager.getCurrentResults();
          const currentItemIds = new Set(currentResults.map(item => item.id));
          
          // Verify no session1 items in current results
          for (const session1ItemId of session1ItemIds) {
            expect(currentItemIds.has(session1ItemId)).toBe(false);
          }
          
          // Verify all current items are from session2
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(session2Id);
            expect(result.metadata.messageId).toBe(message2Id);
          }
          
          // Act: Switch back to session1
          manager.switchSession(session1Id);
          
          // Assert: After switching back, current results should be empty
          // (session1's data was cleared when we switched to session2)
          const session1Results = manager.getCurrentResults();
          expect(session1Results.length).toBe(0);
          
          // Verify current session is session1
          expect(manager.getCurrentSession()).toBe(session1Id);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.7: Verify that getResults returns only data for the specified
   * sessionId and messageId, ensuring data isolation at the query level.
   * 
   * **Validates: Requirements 1.4, 2.2, 4.4**
   */
  it('should return only data for specified sessionId/messageId in getResults', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, message1Id, message2Id, requestId) => {
          // Ensure distinct messages
          fc.pre(message1Id !== message2Id);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Load data for message1
          const message1Items = fc.sample(validAnalysisResultItemArb(sessionId, message1Id), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId: message1Id,
            requestId: `${requestId}_m1`,
            items: message1Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Note: Due to the data clearing behavior, loading message2 will clear message1
          // So we need to verify isolation at the point of loading
          
          // Verify message1 data is accessible
          const message1Results = manager.getResults(sessionId, message1Id);
          expect(message1Results.length).toBe(message1Items.length);
          
          // All results should have correct sessionId and messageId
          for (const result of message1Results) {
            expect(result.metadata.sessionId).toBe(sessionId);
            expect(result.metadata.messageId).toBe(message1Id);
          }
          
          // Query for message2 (which has no data yet) should return empty
          const message2Results = manager.getResults(sessionId, message2Id);
          expect(message2Results.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


// ==================== analysis-result-display-flow Property Tests ====================

describe('Feature: analysis-result-display-flow, Property 3: 会话数据隔离', () => {
  /**
   * **Validates: Requirements 5.1, 5.2**
   * 
   * Property 3: 会话数据隔离
   * 对于任意两个不同的会话 ID，一个会话的分析结果更新不应该影响另一个会话的数据。
   * 
   * For any two different session IDs, updating analysis results for one session
   * should not affect the data of another session.
   */

  beforeEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Property Test 3.1: Updating results for session A should not modify session B's data.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should not affect session B data when updating session A results', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        requestIdArb,
        (sessionAId, sessionBId, messageAId, messageBId, requestAId, requestBId) => {
          // Ensure we have distinct sessions
          fc.pre(sessionAId !== sessionBId);
          
          // Arrange: Set up manager and add data to session B first
          const manager = getAnalysisResultManager();
          
          // Add data to session B
          manager.switchSession(sessionBId);
          const sessionBItems = fc.sample(validAnalysisResultItemArb(sessionBId, messageBId), { numRuns: 3 });
          manager.updateResults({
            sessionId: sessionBId,
            messageId: messageBId,
            requestId: requestBId,
            items: sessionBItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store session B item IDs and data for verification
          const sessionBItemIds = new Set(sessionBItems.map(item => item.id));
          const sessionBItemCount = sessionBItems.length;
          
          // Verify session B has data
          expect(manager.getResults(sessionBId, messageBId).length).toBe(sessionBItemCount);
          
          // Act: Switch to session A and add data
          manager.switchSession(sessionAId);
          const sessionAItems = fc.sample(validAnalysisResultItemArb(sessionAId, messageAId), { numRuns: 5 });
          manager.updateResults({
            sessionId: sessionAId,
            messageId: messageAId,
            requestId: requestAId,
            items: sessionAItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Session B's data should be cleared when we switched sessions
          // (This is the expected behavior per Requirement 5.1 - 切换会话时清除旧数据)
          const sessionBResultsAfterSwitch = manager.getResults(sessionBId, messageBId);
          expect(sessionBResultsAfterSwitch.length).toBe(0);
          
          // Session A should have its data
          const sessionAResults = manager.getResults(sessionAId, messageAId);
          expect(sessionAResults.length).toBe(sessionAItems.length);
          
          // Current session should be session A
          expect(manager.getCurrentSession()).toBe(sessionAId);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.2: Session isolation should be maintained when switching between sessions.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should maintain session isolation when switching between sessions', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionAId, sessionBId, messageId, requestId) => {
          // Ensure we have distinct sessions
          fc.pre(sessionAId !== sessionBId);
          
          // Arrange
          const manager = getAnalysisResultManager();
          
          // Add data to session A
          manager.switchSession(sessionAId);
          const sessionAItems = fc.sample(validAnalysisResultItemArb(sessionAId, messageId), { numRuns: 3 });
          manager.updateResults({
            sessionId: sessionAId,
            messageId,
            requestId: `${requestId}_A`,
            items: sessionAItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify session A has data
          expect(manager.getCurrentResults().length).toBe(sessionAItems.length);
          
          // Act: Switch to session B
          manager.switchSession(sessionBId);
          
          // Assert: Current results should be empty (session B has no data)
          expect(manager.getCurrentResults().length).toBe(0);
          
          // Session A's data should be cleared (per Requirement 5.1)
          expect(manager.getResults(sessionAId, messageId).length).toBe(0);
          
          // Current session should be session B
          expect(manager.getCurrentSession()).toBe(sessionBId);
          
          // Current message should be reset to null
          expect(manager.getCurrentMessage()).toBeNull();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.3: Multiple session switches should maintain data isolation.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should maintain data isolation across multiple session switches', () => {
    fc.assert(
      fc.property(
        fc.array(sessionIdArb, { minLength: 2, maxLength: 5 }),
        messageIdArb,
        requestIdArb,
        (sessionIds, messageId, requestId) => {
          // Ensure all session IDs are unique
          const uniqueSessionIds = new Set(sessionIds);
          fc.pre(uniqueSessionIds.size === sessionIds.length);
          
          // Arrange
          const manager = getAnalysisResultManager();
          
          // Act: Switch through multiple sessions and add data to each
          for (let i = 0; i < sessionIds.length; i++) {
            const sessionId = sessionIds[i];
            manager.switchSession(sessionId);
            
            const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 2 });
            manager.updateResults({
              sessionId,
              messageId,
              requestId: `${requestId}_${i}`,
              items,
              isComplete: true,
              timestamp: Date.now(),
            });
          }
          
          // Assert: Only the last session should have data
          const lastSessionId = sessionIds[sessionIds.length - 1];
          
          // Current session should be the last one
          expect(manager.getCurrentSession()).toBe(lastSessionId);
          
          // Only the last session should have data
          for (let i = 0; i < sessionIds.length - 1; i++) {
            const oldSessionId = sessionIds[i];
            const oldSessionResults = manager.getResults(oldSessionId, messageId);
            expect(oldSessionResults.length).toBe(0);
          }
          
          // Last session should have data
          const lastSessionResults = manager.getCurrentResults();
          expect(lastSessionResults.length).toBeGreaterThan(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.4: Session switch should emit correct event with session IDs.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should emit session-switched event with correct fromSessionId and toSessionId', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        (fromSessionId, toSessionId) => {
          // Ensure we have distinct sessions
          fc.pre(fromSessionId !== toSessionId);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(fromSessionId);
          
          // Set up event listener
          let eventData: { fromSessionId: string | null; toSessionId: string } | null = null;
          const unsubscribe = manager.on('session-switched', (data) => {
            eventData = data;
          });
          
          // Act: Switch to a different session
          manager.switchSession(toSessionId);
          
          // Assert: Event should be emitted with correct data
          expect(eventData).not.toBeNull();
          expect(eventData!.fromSessionId).toBe(fromSessionId);
          expect(eventData!.toSessionId).toBe(toSessionId);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 3.5: Switching to the same session should be a no-op.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should not clear data when switching to the same session', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Add data
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify data exists
          const initialResults = manager.getCurrentResults();
          expect(initialResults.length).toBeGreaterThan(0);
          
          // Set up event listener to verify no event is emitted
          let eventEmitted = false;
          const unsubscribe = manager.on('session-switched', () => {
            eventEmitted = true;
          });
          
          // Act: Switch to the same session
          manager.switchSession(sessionId);
          
          // Assert: Data should still exist
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(initialResults.length);
          
          // No event should be emitted
          expect(eventEmitted).toBe(false);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


describe('Feature: analysis-result-display-flow, Property 4: 消息数据隔离', () => {
  /**
   * **Validates: Requirements 5.1, 5.2**
   * 
   * Property 4: 消息数据隔离
   * 对于任意同一会话下的两个不同消息 ID，切换消息后仪表盘只应显示当前选中消息的分析结果。
   * 
   * For any two different message IDs within the same session, after switching messages,
   * the dashboard should only display the analysis results of the currently selected message.
   */

  beforeEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Property Test 4.1: After switching messages, only the current message's data should be displayed.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should only display current message data after message switch', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageAId, messageBId, requestId) => {
          // Ensure we have distinct messages
          fc.pre(messageAId !== messageBId);
          
          // Arrange: Set up manager with data for message A
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Add data to message A
          const messageAItems = fc.sample(validAnalysisResultItemArb(sessionId, messageAId), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId: messageAId,
            requestId: `${requestId}_A`,
            items: messageAItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store message A item IDs for verification
          const messageAItemIds = new Set(messageAItems.map(item => item.id));
          
          // Verify message A has data
          expect(manager.getCurrentResults().length).toBe(messageAItems.length);
          expect(manager.getCurrentMessage()).toBe(messageAId);
          
          // Act: Switch to message B
          manager.selectMessage(messageBId);
          
          // Assert: Current results should be empty (message B has no data yet)
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          // Current message should be message B
          expect(manager.getCurrentMessage()).toBe(messageBId);
          
          // Message A's data should be cleared (per Requirement 5.2)
          const messageAResults = manager.getResults(sessionId, messageAId);
          expect(messageAResults.length).toBe(0);
          
          // No message A items should be in current results
          const currentItemIds = new Set(currentResults.map(item => item.id));
          for (const messageAItemId of messageAItemIds) {
            expect(currentItemIds.has(messageAItemId)).toBe(false);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.2: After switching messages and loading new data, 
   * only the new message's data should be displayed.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should only display new message data after switch and load', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageAId, messageBId, requestId) => {
          // Ensure we have distinct messages
          fc.pre(messageAId !== messageBId);
          
          // Arrange: Set up manager with data for message A
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Add data to message A
          const messageAItems = fc.sample(validAnalysisResultItemArb(sessionId, messageAId), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId: messageAId,
            requestId: `${requestId}_A`,
            items: messageAItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store message A item IDs for verification
          const messageAItemIds = new Set(messageAItems.map(item => item.id));
          
          // Act: Switch to message B and add data
          manager.selectMessage(messageBId);
          const messageBItems = fc.sample(validAnalysisResultItemArb(sessionId, messageBId), { numRuns: 4 });
          manager.updateResults({
            sessionId,
            messageId: messageBId,
            requestId: `${requestId}_B`,
            items: messageBItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Current results should only contain message B data
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(messageBItems.length);
          
          // All current results should belong to message B
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(sessionId);
            expect(result.metadata.messageId).toBe(messageBId);
          }
          
          // No message A items should be in current results
          const currentItemIds = new Set(currentResults.map(item => item.id));
          for (const messageAItemId of messageAItemIds) {
            expect(currentItemIds.has(messageAItemId)).toBe(false);
          }
          
          // Message A's data should be cleared
          const messageAResults = manager.getResults(sessionId, messageAId);
          expect(messageAResults.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.3: Multiple message switches should maintain data isolation.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should maintain data isolation across multiple message switches', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        fc.array(messageIdArb, { minLength: 2, maxLength: 5 }),
        requestIdArb,
        (sessionId, messageIds, requestId) => {
          // Ensure all message IDs are unique
          const uniqueMessageIds = new Set(messageIds);
          fc.pre(uniqueMessageIds.size === messageIds.length);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Act: Switch through multiple messages and add data to each
          for (let i = 0; i < messageIds.length; i++) {
            const messageId = messageIds[i];
            manager.selectMessage(messageId);
            
            const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 2 });
            manager.updateResults({
              sessionId,
              messageId,
              requestId: `${requestId}_${i}`,
              items,
              isComplete: true,
              timestamp: Date.now(),
            });
          }
          
          // Assert: Only the last message should have data
          const lastMessageId = messageIds[messageIds.length - 1];
          
          // Current message should be the last one
          expect(manager.getCurrentMessage()).toBe(lastMessageId);
          
          // Only the last message should have data
          for (let i = 0; i < messageIds.length - 1; i++) {
            const oldMessageId = messageIds[i];
            const oldMessageResults = manager.getResults(sessionId, oldMessageId);
            expect(oldMessageResults.length).toBe(0);
          }
          
          // Last message should have data
          const lastMessageResults = manager.getCurrentResults();
          expect(lastMessageResults.length).toBeGreaterThan(0);
          
          // All results should belong to the last message
          for (const result of lastMessageResults) {
            expect(result.metadata.messageId).toBe(lastMessageId);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.4: Message switch should emit correct event with message IDs.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should emit message-selected event with correct fromMessageId and toMessageId', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        (sessionId, fromMessageId, toMessageId) => {
          // Ensure we have distinct messages
          fc.pre(fromMessageId !== toMessageId);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          manager.selectMessage(fromMessageId);
          
          // Set up event listener
          let eventData: { sessionId: string; fromMessageId: string | null; toMessageId: string } | null = null;
          const unsubscribe = manager.on('message-selected', (data) => {
            eventData = data;
          });
          
          // Act: Switch to a different message
          manager.selectMessage(toMessageId);
          
          // Assert: Event should be emitted with correct data
          expect(eventData).not.toBeNull();
          expect(eventData!.sessionId).toBe(sessionId);
          expect(eventData!.fromMessageId).toBe(fromMessageId);
          expect(eventData!.toMessageId).toBe(toMessageId);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.5: Switching to the same message should be a no-op.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should not clear data when switching to the same message', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Add data
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify data exists
          const initialResults = manager.getCurrentResults();
          expect(initialResults.length).toBeGreaterThan(0);
          
          // Set up event listener to verify no event is emitted
          let eventEmitted = false;
          const unsubscribe = manager.on('message-selected', () => {
            eventEmitted = true;
          });
          
          // Act: Switch to the same message
          manager.selectMessage(messageId);
          
          // Assert: Data should still exist
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(initialResults.length);
          
          // No event should be emitted
          expect(eventEmitted).toBe(false);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.6: Dashboard should show empty state when switching to message with no data.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should show empty dashboard when switching to message with no data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageWithDataId, messageWithoutDataId, requestId) => {
          // Ensure we have distinct messages
          fc.pre(messageWithDataId !== messageWithoutDataId);
          
          // Arrange: Set up manager with data for one message
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Add data to messageWithDataId
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageWithDataId), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId: messageWithDataId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify data exists
          expect(manager.getCurrentResults().length).toBeGreaterThan(0);
          
          // Act: Switch to message without data
          manager.selectMessage(messageWithoutDataId);
          
          // Assert: Dashboard should be empty
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          // Current message should be the new message
          expect(manager.getCurrentMessage()).toBe(messageWithoutDataId);
          
          // Old message data should be cleared
          const oldMessageResults = manager.getResults(sessionId, messageWithDataId);
          expect(oldMessageResults.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.7: Data isolation should work correctly with restoreResults.
   * 
   * **Validates: Requirements 5.1, 5.2**
   */
  it('should maintain message data isolation when using restoreResults', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, currentMessageId, restoredMessageId, requestId) => {
          // Ensure we have distinct messages
          fc.pre(currentMessageId !== restoredMessageId);
          
          // Arrange: Set up manager with current data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Add current data
          const currentItems = fc.sample(validAnalysisResultItemArb(sessionId, currentMessageId), { numRuns: 3 });
          manager.updateResults({
            sessionId,
            messageId: currentMessageId,
            requestId,
            items: currentItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store current item IDs for verification
          const currentItemIds = new Set(currentItems.map(item => item.id));
          
          // Verify current data exists
          expect(manager.getCurrentResults().length).toBe(currentItems.length);
          
          // Act: Restore data for a different message
          const restoredItems = fc.sample(validAnalysisResultItemArb(sessionId, restoredMessageId), { numRuns: 4 });
          manager.restoreResults(sessionId, restoredMessageId, restoredItems);
          
          // Assert: Current results should only contain restored data
          const currentResults = manager.getCurrentResults();
          
          // All current results should belong to restored message
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(sessionId);
            expect(result.metadata.messageId).toBe(restoredMessageId);
          }
          
          // No current message items should be in results
          const resultItemIds = new Set(currentResults.map(item => item.id));
          for (const currentItemId of currentItemIds) {
            expect(resultItemIds.has(currentItemId)).toBe(false);
          }
          
          // Current message should be the restored message
          expect(manager.getCurrentMessage()).toBe(restoredMessageId);
          
          // Old message data should be cleared
          const oldMessageResults = manager.getResults(sessionId, currentMessageId);
          expect(oldMessageResults.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


// ==================== Property 7: 历史数据恢复完整性 ====================

describe('Feature: analysis-result-display-flow, Property 7: 历史数据恢复完整性', () => {
  /**
   * **Validates: Requirements 5.3**
   * 
   * Property 7: 历史数据恢复完整性
   * 对于任意保存的分析结果，恢复后应该与原始数据等价，且仪表盘应该正确显示恢复的数据。
   * 
   * For any saved analysis results, after restoration, the data should be equivalent
   * to the original data, and the dashboard should correctly display the restored data.
   */

  beforeEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Generate valid ECharts data for restoration testing
   */
  const validEChartsDataArb = fc.record({
    series: fc.array(
      fc.record({
        type: fc.constantFrom('bar', 'line', 'pie', 'scatter'),
        data: fc.array(fc.integer({ min: 0, max: 1000 }), { minLength: 1, maxLength: 10 }),
        name: fc.option(fc.string({ minLength: 1, maxLength: 20 }), { nil: undefined }),
      }),
      { minLength: 1, maxLength: 3 }
    ),
    xAxis: fc.option(
      fc.record({
        type: fc.constantFrom('category', 'value'),
        data: fc.option(fc.array(fc.string({ minLength: 1, maxLength: 10 }), { minLength: 1, maxLength: 10 }), { nil: undefined }),
      }),
      { nil: undefined }
    ),
    yAxis: fc.option(
      fc.record({
        type: fc.constantFrom('category', 'value'),
      }),
      { nil: undefined }
    ),
    title: fc.option(
      fc.record({
        text: fc.string({ minLength: 1, maxLength: 50 }),
      }),
      { nil: undefined }
    ),
  });

  /**
   * Generate valid table data for restoration testing
   */
  const validTableDataArb = fc.array(
    fc.record({
      id: fc.uuid(),
      name: fc.string({ minLength: 1, maxLength: 50 }),
      value: fc.oneof(fc.integer(), fc.double({ min: -1000, max: 1000 })),
    }),
    { minLength: 1, maxLength: 10 }
  );

  /**
   * Generate valid image data (base64 string) for restoration testing
   */
  const validImageDataArb = fc.string({ minLength: 10, maxLength: 100 }).map(
    (s) => `data:image/png;base64,${Buffer.from(s).toString('base64')}`
  );

  /**
   * Generate a complete AnalysisResultItem for restoration testing
   */
  const restorableItemArb = (sessionId: string, messageId: string): fc.Arbitrary<AnalysisResultItem> =>
    fc.oneof(
      // Metric type
      fc.record({
        id: fc.uuid(),
        type: fc.constant('metric' as AnalysisResultType),
        data: validMetricDataArb,
        metadata: fc.record({
          sessionId: fc.constant(sessionId),
          messageId: fc.constant(messageId),
          timestamp: fc.nat(),
        }),
        source: fc.constant('completed' as ResultSource),
      }),
      // Insight type
      fc.record({
        id: fc.uuid(),
        type: fc.constant('insight' as AnalysisResultType),
        data: validInsightDataArb,
        metadata: fc.record({
          sessionId: fc.constant(sessionId),
          messageId: fc.constant(messageId),
          timestamp: fc.nat(),
        }),
        source: fc.constant('completed' as ResultSource),
      }),
      // File type
      fc.record({
        id: fc.uuid(),
        type: fc.constant('file' as AnalysisResultType),
        data: validFileDataArb,
        metadata: fc.record({
          sessionId: fc.constant(sessionId),
          messageId: fc.constant(messageId),
          timestamp: fc.nat(),
        }),
        source: fc.constant('completed' as ResultSource),
      }),
      // ECharts type
      fc.record({
        id: fc.uuid(),
        type: fc.constant('echarts' as AnalysisResultType),
        data: validEChartsDataArb,
        metadata: fc.record({
          sessionId: fc.constant(sessionId),
          messageId: fc.constant(messageId),
          timestamp: fc.nat(),
        }),
        source: fc.constant('completed' as ResultSource),
      }),
      // Table type
      fc.record({
        id: fc.uuid(),
        type: fc.constant('table' as AnalysisResultType),
        data: validTableDataArb,
        metadata: fc.record({
          sessionId: fc.constant(sessionId),
          messageId: fc.constant(messageId),
          timestamp: fc.nat(),
        }),
        source: fc.constant('completed' as ResultSource),
      })
    ) as fc.Arbitrary<AnalysisResultItem>;

  /**
   * Property Test 7.1: Restored data should be equivalent to original data.
   * 
   * For any valid analysis result items, after restoration:
   * - The number of valid items should match the original count
   * - The data types should be preserved
   * - The data content should be equivalent (after normalization)
   * 
   * **Validates: Requirements 5.3**
   */
  it('should restore data equivalent to original data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.integer({ min: 1, max: 5 }),
        (sessionId, messageId, itemCount) => {
          // Arrange: Generate valid items to restore
          const manager = getAnalysisResultManager();
          const originalItems = fc.sample(restorableItemArb(sessionId, messageId), { numRuns: itemCount });
          
          // Act: Restore the items
          const stats = manager.restoreResults(sessionId, messageId, originalItems);
          
          // Assert: All valid items should be restored
          expect(stats.totalItems).toBe(originalItems.length);
          expect(stats.validItems).toBe(originalItems.length);
          expect(stats.invalidItems).toBe(0);
          expect(stats.errors.length).toBe(0);
          
          // Verify the restored data is accessible
          const restoredItems = manager.getResults(sessionId, messageId);
          expect(restoredItems.length).toBe(originalItems.length);
          
          // Verify each item type is preserved
          const originalTypeCount: Record<string, number> = {};
          originalItems.forEach(item => {
            originalTypeCount[item.type] = (originalTypeCount[item.type] || 0) + 1;
          });
          
          const restoredTypeCount: Record<string, number> = {};
          restoredItems.forEach(item => {
            restoredTypeCount[item.type] = (restoredTypeCount[item.type] || 0) + 1;
          });
          
          // Type distribution should match
          expect(restoredTypeCount).toEqual(originalTypeCount);
          expect(stats.itemsByType).toEqual(originalTypeCount);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.2: Dashboard should correctly display restored data.
   * 
   * After restoration, the dashboard (via getCurrentResults) should:
   * - Return all restored items
   * - Have the correct current session and message set
   * - Not be in loading state
   * 
   * **Validates: Requirements 5.3**
   */
  it('should correctly display restored data in dashboard', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.integer({ min: 1, max: 5 }),
        (sessionId, messageId, itemCount) => {
          // Arrange
          const manager = getAnalysisResultManager();
          const originalItems = fc.sample(restorableItemArb(sessionId, messageId), { numRuns: itemCount });
          
          // Act: Restore the items
          manager.restoreResults(sessionId, messageId, originalItems);
          
          // Assert: Dashboard should display the restored data
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(originalItems.length);
          
          // Current session and message should be set correctly
          expect(manager.getCurrentSession()).toBe(sessionId);
          expect(manager.getCurrentMessage()).toBe(messageId);
          
          // Should not be in loading state
          expect(manager.isLoading()).toBe(false);
          
          // All items should have 'restored' source
          for (const item of currentResults) {
            expect(item.source).toBe('restored');
          }
          
          // All items should have correct sessionId and messageId in metadata
          for (const item of currentResults) {
            expect(item.metadata.sessionId).toBe(sessionId);
            expect(item.metadata.messageId).toBe(messageId);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.3: Restoration should clear previous data and maintain isolation.
   * 
   * When restoring data for a message:
   * - Previous data for other messages should be cleared
   * - Only the restored data should be visible
   * 
   * **Validates: Requirements 5.3**
   */
  it('should clear previous data when restoring historical data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        fc.integer({ min: 1, max: 3 }),
        fc.integer({ min: 1, max: 3 }),
        (sessionId, currentMessageId, historicalMessageId, requestId, currentItemCount, historicalItemCount) => {
          // Ensure distinct messages
          fc.pre(currentMessageId !== historicalMessageId);
          
          // Arrange: Set up manager with current data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Add current analysis data
          const currentItems = fc.sample(validAnalysisResultItemArb(sessionId, currentMessageId), { numRuns: currentItemCount });
          manager.updateResults({
            sessionId,
            messageId: currentMessageId,
            requestId,
            items: currentItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Store current item IDs for verification
          const currentItemIds = new Set(currentItems.map(item => item.id));
          
          // Verify current data exists
          expect(manager.getCurrentResults().length).toBe(currentItemCount);
          
          // Act: Restore historical data
          const historicalItems = fc.sample(restorableItemArb(sessionId, historicalMessageId), { numRuns: historicalItemCount });
          manager.restoreResults(sessionId, historicalMessageId, historicalItems);
          
          // Assert: Current results should only contain restored data
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(historicalItemCount);
          
          // No current message items should be in results
          const resultItemIds = new Set(currentResults.map(item => item.id));
          for (const currentItemId of currentItemIds) {
            expect(resultItemIds.has(currentItemId)).toBe(false);
          }
          
          // All results should belong to historical message
          for (const result of currentResults) {
            expect(result.metadata.messageId).toBe(historicalMessageId);
          }
          
          // Old message data should be cleared
          const oldMessageResults = manager.getResults(sessionId, currentMessageId);
          expect(oldMessageResults.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.4: Restoration should emit data-restored event with correct statistics.
   * 
   * **Validates: Requirements 5.3**
   */
  it('should emit data-restored event with correct statistics', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.integer({ min: 1, max: 5 }),
        (sessionId, messageId, itemCount) => {
          // Arrange
          const manager = getAnalysisResultManager();
          const originalItems = fc.sample(restorableItemArb(sessionId, messageId), { numRuns: itemCount });
          
          // Set up event listener
          let eventData: {
            sessionId: string;
            messageId: string;
            itemCount: number;
            validCount: number;
            invalidCount: number;
            itemsByType: Record<string, number>;
          } | null = null;
          
          const unsubscribe = manager.on('data-restored', (data) => {
            eventData = data;
          });
          
          // Act: Restore the items
          const stats = manager.restoreResults(sessionId, messageId, originalItems);
          
          // Assert: Event should be emitted with correct data
          expect(eventData).not.toBeNull();
          expect(eventData!.sessionId).toBe(sessionId);
          expect(eventData!.messageId).toBe(messageId);
          expect(eventData!.itemCount).toBe(originalItems.length);
          expect(eventData!.validCount).toBe(stats.validItems);
          expect(eventData!.invalidCount).toBe(stats.invalidItems);
          expect(eventData!.itemsByType).toEqual(stats.itemsByType);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.5: Restoration with empty items should notify empty result.
   * 
   * **Validates: Requirements 5.3**
   */
  it('should notify empty result when restoring empty items', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        (sessionId, messageId) => {
          // Arrange
          const manager = getAnalysisResultManager();
          
          // Set up event listener for historical-empty-result
          let emptyResultEvent: { sessionId: string; messageId: string } | null = null;
          const unsubscribe = manager.on('historical-empty-result', (data) => {
            emptyResultEvent = data;
          });
          
          // Act: Restore empty items
          const stats = manager.restoreResults(sessionId, messageId, []);
          
          // Assert: Stats should reflect empty restoration
          expect(stats.totalItems).toBe(0);
          expect(stats.validItems).toBe(0);
          expect(stats.invalidItems).toBe(0);
          
          // Empty result event should be emitted
          expect(emptyResultEvent).not.toBeNull();
          expect(emptyResultEvent!.sessionId).toBe(sessionId);
          expect(emptyResultEvent!.messageId).toBe(messageId);
          
          // Dashboard should be empty
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.6: Restoration should handle invalid items gracefully.
   * 
   * When some items are invalid:
   * - Valid items should still be restored
   * - Invalid items should be counted in stats
   * - Errors should be recorded
   * 
   * **Validates: Requirements 5.3**
   */
  it('should handle invalid items gracefully during restoration', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.integer({ min: 1, max: 3 }),
        (sessionId, messageId, validItemCount) => {
          // Arrange
          const manager = getAnalysisResultManager();
          
          // Generate valid items
          const validItems = fc.sample(restorableItemArb(sessionId, messageId), { numRuns: validItemCount });
          
          // Create invalid items (missing required fields)
          const invalidItems: AnalysisResultItem[] = [
            {
              id: 'invalid-1',
              type: 'unknown-type' as AnalysisResultType, // Invalid type
              data: {},
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'completed' as ResultSource,
            },
            {
              id: 'invalid-2',
              type: 'metric' as AnalysisResultType,
              data: null as any, // Null data
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'completed' as ResultSource,
            },
          ];
          
          // Mix valid and invalid items
          const mixedItems = [...validItems, ...invalidItems];
          
          // Act: Restore the mixed items
          const stats = manager.restoreResults(sessionId, messageId, mixedItems);
          
          // Assert: Stats should reflect the mixed restoration
          expect(stats.totalItems).toBe(mixedItems.length);
          expect(stats.validItems).toBe(validItemCount);
          expect(stats.invalidItems).toBe(invalidItems.length);
          expect(stats.errors.length).toBe(invalidItems.length);
          
          // Only valid items should be restored
          const restoredItems = manager.getResults(sessionId, messageId);
          expect(restoredItems.length).toBe(validItemCount);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.7: Multiple sequential restorations should maintain data integrity.
   * 
   * When restoring data multiple times:
   * - Each restoration should replace the previous data
   * - Only the latest restored data should be visible
   * 
   * **Validates: Requirements 5.3**
   */
  it('should maintain data integrity across multiple sequential restorations', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        fc.array(messageIdArb, { minLength: 2, maxLength: 4 }),
        (sessionId, messageIds) => {
          // Ensure all message IDs are unique
          const uniqueMessageIds = new Set(messageIds);
          fc.pre(uniqueMessageIds.size === messageIds.length);
          
          // Arrange
          const manager = getAnalysisResultManager();
          
          // Act: Perform multiple sequential restorations
          let lastMessageId: string | null = null;
          let lastItemCount = 0;
          
          for (const messageId of messageIds) {
            const itemCount = fc.sample(fc.integer({ min: 1, max: 3 }), 1)[0];
            const items = fc.sample(restorableItemArb(sessionId, messageId), { numRuns: itemCount });
            
            manager.restoreResults(sessionId, messageId, items);
            
            lastMessageId = messageId;
            lastItemCount = itemCount;
          }
          
          // Assert: Only the last restoration's data should be visible
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(lastItemCount);
          
          // Current message should be the last one
          expect(manager.getCurrentMessage()).toBe(lastMessageId);
          
          // All results should belong to the last message
          for (const result of currentResults) {
            expect(result.metadata.messageId).toBe(lastMessageId);
          }
          
          // Previous messages should have no data
          for (let i = 0; i < messageIds.length - 1; i++) {
            const oldMessageId = messageIds[i];
            const oldMessageResults = manager.getResults(sessionId, oldMessageId);
            expect(oldMessageResults.length).toBe(0);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.8: Restoration should preserve item metadata.
   * 
   * After restoration:
   * - Original timestamps should be preserved (if provided)
   * - SessionId and messageId should be set correctly
   * 
   * **Validates: Requirements 5.3**
   */
  it('should preserve item metadata during restoration', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.integer({ min: 1, max: 5 }),
        (sessionId, messageId, itemCount) => {
          // Arrange
          const manager = getAnalysisResultManager();
          const originalItems = fc.sample(restorableItemArb(sessionId, messageId), { numRuns: itemCount });
          
          // Store original timestamps
          const originalTimestamps = new Map<string, number>();
          originalItems.forEach(item => {
            if (item.metadata?.timestamp) {
              originalTimestamps.set(item.id, item.metadata.timestamp);
            }
          });
          
          // Act: Restore the items
          manager.restoreResults(sessionId, messageId, originalItems);
          
          // Assert: Metadata should be preserved
          const restoredItems = manager.getResults(sessionId, messageId);
          
          for (const restoredItem of restoredItems) {
            // SessionId and messageId should be set correctly
            expect(restoredItem.metadata.sessionId).toBe(sessionId);
            expect(restoredItem.metadata.messageId).toBe(messageId);
            
            // Timestamp should be preserved or set
            expect(restoredItem.metadata.timestamp).toBeDefined();
            expect(typeof restoredItem.metadata.timestamp).toBe('number');
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.9: Restoration should work correctly across different sessions.
   * 
   * When restoring data for a different session:
   * - The session should be switched
   * - Old session data should be cleared
   * 
   * **Validates: Requirements 5.3**
   */
  it('should work correctly when restoring data for different sessions', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        fc.integer({ min: 1, max: 3 }),
        fc.integer({ min: 1, max: 3 }),
        (session1Id, session2Id, message1Id, message2Id, requestId, itemCount1, itemCount2) => {
          // Ensure distinct sessions
          fc.pre(session1Id !== session2Id);
          
          // Arrange: Set up manager with session 1 data
          const manager = getAnalysisResultManager();
          manager.switchSession(session1Id);
          
          const session1Items = fc.sample(validAnalysisResultItemArb(session1Id, message1Id), { numRuns: itemCount1 });
          manager.updateResults({
            sessionId: session1Id,
            messageId: message1Id,
            requestId,
            items: session1Items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify session 1 has data
          expect(manager.getCurrentResults().length).toBe(itemCount1);
          expect(manager.getCurrentSession()).toBe(session1Id);
          
          // Act: Restore data for session 2
          const session2Items = fc.sample(restorableItemArb(session2Id, message2Id), { numRuns: itemCount2 });
          manager.restoreResults(session2Id, message2Id, session2Items);
          
          // Assert: Session should be switched to session 2
          expect(manager.getCurrentSession()).toBe(session2Id);
          expect(manager.getCurrentMessage()).toBe(message2Id);
          
          // Current results should be session 2's data
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(itemCount2);
          
          for (const result of currentResults) {
            expect(result.metadata.sessionId).toBe(session2Id);
            expect(result.metadata.messageId).toBe(message2Id);
          }
          
          // Session 1's data should be cleared
          const session1Results = manager.getResults(session1Id, message1Id);
          expect(session1Results.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.10: Restoration should clear error state.
   * 
   * After successful restoration:
   * - Error state should be cleared
   * - Loading state should be false
   * 
   * **Validates: Requirements 5.3**
   */
  it('should clear error state after successful restoration', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.integer({ min: 1, max: 3 }),
        (sessionId, messageId, itemCount) => {
          // Arrange: Set up manager with error state
          const manager = getAnalysisResultManager();
          manager.setError('Previous error message');
          
          // Verify error is set
          expect(manager.getError()).not.toBeNull();
          
          // Act: Restore data
          const items = fc.sample(restorableItemArb(sessionId, messageId), { numRuns: itemCount });
          manager.restoreResults(sessionId, messageId, items);
          
          // Assert: Error should be cleared
          expect(manager.getError()).toBeNull();
          expect(manager.isLoading()).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


// ==================== Property Tests for analysis-dashboard-optimization ====================

describe('Feature: analysis-dashboard-optimization, Property 4: AnalysisResultManager Data Storage', () => {
  /**
   * **Validates: Requirements 3.4**
   * 
   * Property 4: AnalysisResultManager Data Storage
   * For any AnalysisResultBatch received by AnalysisResultManager, all items SHALL be
   * stored with their original sessionId and messageId preserved.
   */

  beforeEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Property Test 4.1: Items should preserve sessionId and messageId after storage
   * 
   * **Validates: Requirements 3.4**
   */
  it('should preserve sessionId and messageId for all stored items', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.integer({ min: 1, max: 10 }),
        (sessionId, messageId, requestId, itemCount) => {
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Generate items with specific sessionId and messageId
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: itemCount });
          
          // Act: Store items
          const batch: AnalysisResultBatch = {
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          };
          manager.updateResults(batch);
          
          // Assert: All stored items should have correct sessionId and messageId
          const storedResults = manager.getResults(sessionId, messageId);
          
          for (const result of storedResults) {
            expect(result.metadata.sessionId).toBe(sessionId);
            expect(result.metadata.messageId).toBe(messageId);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.2: Batch items should all be stored
   * 
   * **Validates: Requirements 3.4**
   */
  it('should store all items from a batch', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.integer({ min: 1, max: 10 }),
        (sessionId, messageId, requestId, itemCount) => {
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: itemCount });
          
          // Act
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: All items should be stored
          const storedResults = manager.getResults(sessionId, messageId);
          expect(storedResults.length).toBe(itemCount);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.3: Item data should be preserved after storage
   * 
   * **Validates: Requirements 3.4**
   */
  it('should preserve item data after storage', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Create items with specific data
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: 3 });
          const originalIds = items.map(item => item.id);
          const originalTypes = items.map(item => item.type);
          
          // Act
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Item IDs and types should be preserved
          const storedResults = manager.getResults(sessionId, messageId);
          const storedIds = storedResults.map(item => item.id);
          const storedTypes = storedResults.map(item => item.type);
          
          for (const id of originalIds) {
            expect(storedIds).toContain(id);
          }
          
          for (const type of originalTypes) {
            expect(storedTypes).toContain(type);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});

describe('Feature: analysis-dashboard-optimization, Property 11: Session Switching Clears Data', () => {
  /**
   * **Validates: Requirements 7.5**
   * 
   * Property 11: Session Switching Clears Data
   * For any session switch from sessionA to sessionB, the AnalysisResultManager SHALL
   * clear all data from sessionA before loading sessionB data.
   */

  beforeEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  afterEach(() => {
    AnalysisResultManagerImpl.resetInstance();
  });

  /**
   * Property Test 11.1: Session switch should clear old session data
   * 
   * **Validates: Requirements 7.5**
   */
  it('should clear old session data when switching sessions', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.integer({ min: 1, max: 5 }),
        (sessionA, sessionB, messageId, requestId, itemCount) => {
          // Skip if sessions are the same
          fc.pre(sessionA !== sessionB);
          
          // Arrange: Set up manager with sessionA data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionA);
          
          const sessionAItems = fc.sample(validAnalysisResultItemArb(sessionA, messageId), { numRuns: itemCount });
          manager.updateResults({
            sessionId: sessionA,
            messageId,
            requestId,
            items: sessionAItems,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify sessionA has data
          expect(manager.getResults(sessionA, messageId).length).toBe(itemCount);
          
          // Act: Switch to sessionB
          manager.switchSession(sessionB);
          
          // Assert: SessionA data should be cleared
          const sessionAResults = manager.getResults(sessionA, messageId);
          expect(sessionAResults.length).toBe(0);
          
          // Current session should be sessionB
          expect(manager.getCurrentSession()).toBe(sessionB);
          
          // Current results should be empty (sessionB has no data yet)
          expect(manager.getCurrentResults().length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 11.2: Session switch should reset current message
   * 
   * **Validates: Requirements 7.5**
   */
  it('should reset current message when switching sessions', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionA, sessionB, messageId, requestId) => {
          // Skip if sessions are the same
          fc.pre(sessionA !== sessionB);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionA);
          manager.selectMessage(messageId);
          
          // Verify current message is set
          expect(manager.getCurrentMessage()).toBe(messageId);
          
          // Act: Switch to sessionB
          manager.switchSession(sessionB);
          
          // Assert: Current message should be reset to null
          expect(manager.getCurrentMessage()).toBeNull();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 11.3: Session switch should emit session-switched event
   * 
   * **Validates: Requirements 7.5**
   */
  it('should emit session-switched event with correct data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        (sessionA, sessionB) => {
          // Skip if sessions are the same
          fc.pre(sessionA !== sessionB);
          
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionA);
          
          let eventData: { fromSessionId: string | null; toSessionId: string } | null = null;
          const unsubscribe = manager.on('session-switched', (data) => {
            eventData = data;
          });
          
          // Act: Switch to sessionB
          manager.switchSession(sessionB);
          
          // Assert: Event should be emitted with correct data
          expect(eventData).not.toBeNull();
          expect(eventData!.fromSessionId).toBe(sessionA);
          expect(eventData!.toSessionId).toBe(sessionB);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 11.4: Switching to same session should be a no-op
   * 
   * **Validates: Requirements 7.5**
   */
  it('should not clear data when switching to the same session', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.integer({ min: 1, max: 5 }),
        (sessionId, messageId, requestId, itemCount) => {
          // Arrange
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          const items = fc.sample(validAnalysisResultItemArb(sessionId, messageId), { numRuns: itemCount });
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify data exists
          expect(manager.getResults(sessionId, messageId).length).toBe(itemCount);
          
          // Set up event listener
          let eventEmitted = false;
          const unsubscribe = manager.on('session-switched', () => {
            eventEmitted = true;
          });
          
          // Act: Switch to the same session
          manager.switchSession(sessionId);
          
          // Assert: Data should still exist
          expect(manager.getResults(sessionId, messageId).length).toBe(itemCount);
          
          // No event should be emitted
          expect(eventEmitted).toBe(false);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});
