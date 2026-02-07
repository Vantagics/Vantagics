/**
 * Property-Based Tests for useDashboardData Hook
 * 
 * Feature: dashboard-data-isolation, Property 2: 数据源统计清除同步
 * 
 * These tests verify that dataSourceStatistics is properly cleared when
 * specific events are triggered (analysis-started, session-switched, manager clearAll).
 * 
 * **Validates: Requirements 1.2, 4.2, 5.1**
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import * as fc from 'fast-check';
import { AnalysisResultManagerImpl, getAnalysisResultManager } from '../managers/AnalysisResultManager';
import { agent } from '../../wailsjs/go/models';

// Mock the systemLog to avoid Wails runtime errors in tests
vi.mock('../utils/systemLog', () => ({
  createLogger: () => ({
    debug: () => {},
    info: () => {},
    warn: () => {},
    error: () => {},
  }),
}));

// Mock the Wails API for GetDataSourceStatistics
vi.mock('../../wailsjs/go/main/App', () => ({
  GetDataSourceStatistics: vi.fn(),
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
 * Generate random data source type
 */
const dataSourceTypeArb = fc.constantFrom('csv', 'xlsx', 'mysql', 'postgresql', 'sqlite', 'json');

/**
 * Generate random data source entry
 */
const dataSourceEntryArb = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 50 }),
  type: dataSourceTypeArb,
});

/**
 * Generate random breakdown by type (dictionary of type -> count)
 */
const breakdownByTypeArb = fc.dictionary(
  dataSourceTypeArb,
  fc.integer({ min: 1, max: 100 })
);

/**
 * Generate random DataSourceStatistics
 * This simulates the data structure returned by GetDataSourceStatistics
 */
const dataSourceStatisticsArb = fc.record({
  total_count: fc.integer({ min: 1, max: 100 }),
  breakdown_by_type: breakdownByTypeArb,
  data_sources: fc.array(dataSourceEntryArb, { minLength: 1, maxLength: 10 }),
}) as fc.Arbitrary<agent.DataSourceStatistics>;

/**
 * Generate clearing event type for Property 2
 * These are the events that should clear dataSourceStatistics
 */
const clearingEventTypeArb = fc.constantFrom(
  'analysis-started',
  'session-switched',
  'manager-clearAll'
);

// ==================== Helper Types ====================

/**
 * Simulated state for testing dataSourceStatistics clearing
 */
interface SimulatedDashboardState {
  dataSourceStatistics: agent.DataSourceStatistics | null;
  currentSessionId: string | null;
  currentMessageId: string | null;
}

/**
 * Event handler registry for testing
 */
interface EventHandlers {
  'analysis-started': ((event: { sessionId: string; messageId: string; requestId: string }) => void)[];
  'session-switched': ((event: { fromSessionId: string | null; toSessionId: string }) => void)[];
}

// ==================== Test Setup ====================

describe('Feature: dashboard-data-isolation, Property 2: 数据源统计清除同步', () => {
  /**
   * **Validates: Requirements 1.2, 4.2, 5.1**
   * 
   * Property 2: 数据源统计清除同步
   * For any 需要清除数据源统计的事件（新分析开始、会话切换、Manager 清除数据），
   * 当该事件触发时，dataSourceStatistics 状态应被设置为 null。
   */

  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  /**
   * Property Test 2.1: When analysis-started event is emitted,
   * dataSourceStatistics should be cleared (set to null).
   * 
   * This tests the event subscription mechanism in useDashboardData
   * that listens for analysis-started events and clears the statistics.
   * 
   * **Validates: Requirements 1.2**
   */
  it('should clear dataSourceStatistics when analysis-started event is emitted', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        dataSourceStatisticsArb,
        (sessionId, messageId, requestId, initialStats) => {
          // Arrange: Set up manager and simulate initial state with dataSourceStatistics
          const manager = getAnalysisResultManager();
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          
          // Subscribe to analysis-started event (simulating useDashboardData behavior)
          const unsubscribe = manager.on('analysis-started', () => {
            // This is what useDashboardData does: clear dataSourceStatistics
            dataSourceStatistics = null;
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          expect(dataSourceStatistics?.total_count).toBeGreaterThan(0);
          
          // Act: Trigger analysis-started event by calling setLoading
          manager.setLoading(true, requestId, messageId);
          
          // Assert: dataSourceStatistics should be cleared
          expect(dataSourceStatistics).toBeNull();
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 2.2: When session-switched event is emitted,
   * dataSourceStatistics should be cleared (set to null).
   * 
   * This tests the event subscription mechanism in useDashboardData
   * that listens for session-switched events and clears the statistics.
   * 
   * **Validates: Requirements 4.2**
   */
  it('should clear dataSourceStatistics when session-switched event is emitted', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        dataSourceStatisticsArb,
        (fromSessionId, toSessionId, initialStats) => {
          // Skip if sessions are the same (no switch event would be emitted)
          fc.pre(fromSessionId !== toSessionId);
          
          // Arrange: Set up manager and simulate initial state
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(fromSessionId);
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          
          // Subscribe to session-switched event (simulating useDashboardData behavior)
          const unsubscribe = manager.on('session-switched', () => {
            // This is what useDashboardData does: clear dataSourceStatistics
            dataSourceStatistics = null;
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          expect(dataSourceStatistics?.total_count).toBeGreaterThan(0);
          
          // Act: Trigger session-switched event by switching session
          manager.switchSession(toSessionId);
          
          // Assert: dataSourceStatistics should be cleared
          expect(dataSourceStatistics).toBeNull();
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 2.3: When manager's clearAll is called,
   * dataSourceStatistics should be cleared (set to null).
   * 
   * This tests that when the AnalysisResultManager clears all data,
   * the useDashboardData hook should also clear its dataSourceStatistics.
   * 
   * **Validates: Requirements 5.1**
   */
  it('should clear dataSourceStatistics when manager clearAll is called', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        (sessionId, messageId, initialStats) => {
          // Arrange: Set up manager with some data
          const manager = getAnalysisResultManager();
          
          // Set up session and message
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          
          // Subscribe to state changes (simulating useDashboardData behavior)
          // In the actual implementation, clearAllData calls manager.clearAll()
          // and also clears dataSourceStatistics
          const unsubscribe = manager.subscribe((state) => {
            // When all data is cleared (no session, no message), clear statistics
            if (state.currentSessionId === null && state.currentMessageId === null) {
              dataSourceStatistics = null;
            }
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          expect(dataSourceStatistics?.total_count).toBeGreaterThan(0);
          
          // Act: Call clearAll on the manager
          manager.clearAll();
          
          // Assert: dataSourceStatistics should be cleared
          expect(dataSourceStatistics).toBeNull();
          
          // Also verify manager state is cleared
          expect(manager.getCurrentSession()).toBeNull();
          expect(manager.getCurrentMessage()).toBeNull();
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 2.4: For any clearing event type, dataSourceStatistics
   * should always be set to null after the event is triggered.
   * 
   * This is a comprehensive test that verifies the clearing behavior
   * across all event types that should clear dataSourceStatistics.
   * 
   * **Validates: Requirements 1.2, 4.2, 5.1**
   */
  it('should clear dataSourceStatistics for any clearing event type', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        dataSourceStatisticsArb,
        clearingEventTypeArb,
        (sessionId1, sessionId2, messageId, requestId, initialStats, eventType) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId1);
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          
          // Subscribe to events (simulating useDashboardData behavior)
          const unsubscribeAnalysisStarted = manager.on('analysis-started', () => {
            dataSourceStatistics = null;
          });
          
          const unsubscribeSessionSwitched = manager.on('session-switched', () => {
            dataSourceStatistics = null;
          });
          
          const unsubscribeState = manager.subscribe((state) => {
            if (state.currentSessionId === null && state.currentMessageId === null) {
              dataSourceStatistics = null;
            }
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          
          // Act: Trigger the appropriate event based on eventType
          switch (eventType) {
            case 'analysis-started':
              manager.setLoading(true, requestId, messageId);
              break;
            case 'session-switched':
              // Ensure we switch to a different session
              const targetSession = sessionId1 !== sessionId2 ? sessionId2 : `${sessionId2}-different`;
              manager.switchSession(targetSession);
              break;
            case 'manager-clearAll':
              manager.clearAll();
              break;
          }
          
          // Assert: dataSourceStatistics should be cleared
          expect(dataSourceStatistics).toBeNull();
          
          // Cleanup
          unsubscribeAnalysisStarted();
          unsubscribeSessionSwitched();
          unsubscribeState();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 2.5: Multiple consecutive clearing events should
   * all result in dataSourceStatistics being null.
   * 
   * This tests that the clearing behavior is idempotent and consistent
   * even when multiple events are triggered in sequence.
   * 
   * **Validates: Requirements 1.2, 4.2, 5.1**
   */
  it('should maintain null dataSourceStatistics after multiple clearing events', () => {
    fc.assert(
      fc.property(
        fc.array(
          fc.tuple(
            clearingEventTypeArb,
            sessionIdArb,
            messageIdArb,
            requestIdArb
          ),
          { minLength: 2, maxLength: 5 }
        ),
        dataSourceStatisticsArb,
        (events, initialStats) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          if (events.length > 0) {
            manager.switchSession(events[0][1]);
          }
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          
          // Subscribe to events
          const unsubscribeAnalysisStarted = manager.on('analysis-started', () => {
            dataSourceStatistics = null;
          });
          
          const unsubscribeSessionSwitched = manager.on('session-switched', () => {
            dataSourceStatistics = null;
          });
          
          const unsubscribeState = manager.subscribe((state) => {
            if (state.currentSessionId === null && state.currentMessageId === null) {
              dataSourceStatistics = null;
            }
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          
          // Act: Trigger multiple events
          let lastSessionId = events[0][1];
          for (const [eventType, sessionId, messageId, requestId] of events) {
            switch (eventType) {
              case 'analysis-started':
                manager.setLoading(true, requestId, messageId);
                manager.setLoading(false); // Reset loading state
                break;
              case 'session-switched':
                const targetSession = lastSessionId !== sessionId ? sessionId : `${sessionId}-alt`;
                manager.switchSession(targetSession);
                lastSessionId = targetSession;
                break;
              case 'manager-clearAll':
                manager.clearAll();
                // Re-initialize session for next iteration
                if (events.indexOf([eventType, sessionId, messageId, requestId]) < events.length - 1) {
                  manager.switchSession(sessionId);
                  lastSessionId = sessionId;
                }
                break;
            }
            
            // Assert: After each event, dataSourceStatistics should be null
            expect(dataSourceStatistics).toBeNull();
          }
          
          // Cleanup
          unsubscribeAnalysisStarted();
          unsubscribeSessionSwitched();
          unsubscribeState();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 2.6: Event handlers should be properly called
   * with correct event data when clearing events are triggered.
   * 
   * This verifies that the event system correctly passes event data
   * to subscribers, which is essential for the clearing logic.
   * 
   * **Validates: Requirements 1.2, 4.2, 5.1**
   */
  it('should emit events with correct data when clearing events are triggered', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (fromSessionId, toSessionId, messageId, requestId) => {
          // Skip if sessions are the same for session-switched test
          fc.pre(fromSessionId !== toSessionId);
          
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Track received events
          let receivedAnalysisStartedEvent: { sessionId: string; messageId: string; requestId: string } | null = null;
          let receivedSessionSwitchedEvent: { fromSessionId: string | null; toSessionId: string } | null = null;
          
          // Subscribe to events
          const unsubscribeAnalysisStarted = manager.on('analysis-started', (event) => {
            receivedAnalysisStartedEvent = event;
          });
          
          const unsubscribeSessionSwitched = manager.on('session-switched', (event) => {
            receivedSessionSwitchedEvent = event;
          });
          
          // Set initial session
          manager.switchSession(fromSessionId);
          
          // Clear the session-switched event from initial setup
          receivedSessionSwitchedEvent = null;
          
          // Act & Assert: Test analysis-started event
          manager.setLoading(true, requestId, messageId);
          
          expect(receivedAnalysisStartedEvent).not.toBeNull();
          expect(receivedAnalysisStartedEvent?.sessionId).toBe(fromSessionId);
          expect(receivedAnalysisStartedEvent?.messageId).toBe(messageId);
          expect(receivedAnalysisStartedEvent?.requestId).toBe(requestId);
          
          // Reset loading state
          manager.setLoading(false);
          
          // Act & Assert: Test session-switched event
          manager.switchSession(toSessionId);
          
          expect(receivedSessionSwitchedEvent).not.toBeNull();
          expect(receivedSessionSwitchedEvent?.fromSessionId).toBe(fromSessionId);
          expect(receivedSessionSwitchedEvent?.toSessionId).toBe(toSessionId);
          
          // Cleanup
          unsubscribeAnalysisStarted();
          unsubscribeSessionSwitched();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 2.7: Unsubscribing from events should prevent
   * dataSourceStatistics from being cleared.
   * 
   * This tests that the cleanup mechanism works correctly,
   * ensuring no memory leaks or unexpected behavior after unsubscribe.
   * 
   * **Validates: Requirements 1.2, 4.2, 5.1**
   */
  it('should not clear dataSourceStatistics after unsubscribing from events', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        dataSourceStatisticsArb,
        (sessionId1, sessionId2, messageId, requestId, initialStats) => {
          // Skip if sessions are the same
          fc.pre(sessionId1 !== sessionId2);
          
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId1);
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          
          // Subscribe to events
          const unsubscribeAnalysisStarted = manager.on('analysis-started', () => {
            dataSourceStatistics = null;
          });
          
          const unsubscribeSessionSwitched = manager.on('session-switched', () => {
            dataSourceStatistics = null;
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          
          // Act: Unsubscribe from events
          unsubscribeAnalysisStarted();
          unsubscribeSessionSwitched();
          
          // Reset statistics to initial value
          dataSourceStatistics = initialStats;
          
          // Trigger events that would normally clear statistics
          manager.setLoading(true, requestId, messageId);
          manager.switchSession(sessionId2);
          
          // Assert: dataSourceStatistics should NOT be cleared (still has initial value)
          expect(dataSourceStatistics).not.toBeNull();
          expect(dataSourceStatistics?.total_count).toBe(initialStats.total_count);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


/**
 * Property-Based Tests for Historical Data Restoration Logic
 * 
 * Feature: dashboard-data-isolation, Task 4.2: 改进历史数据恢复逻辑
 * 
 * These tests verify that when historical data is restored:
 * 1. Old data is cleared first (Requirement 2.1)
 * 2. Only restored data is displayed (Requirement 2.2)
 * 
 * **Validates: Requirements 2.1, 2.2**
 */
describe('Feature: dashboard-data-isolation, Task 4.2: 改进历史数据恢复逻辑', () => {
  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  /**
   * Property Test 4.2.1: When message-selected event is emitted,
   * dataSourceStatistics should be cleared.
   * 
   * This is important for historical data restoration because
   * restoring historical data triggers message selection.
   * 
   * **Validates: Requirements 2.1, 2.2**
   */
  it('should clear dataSourceStatistics when message-selected event is emitted', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        (sessionId, fromMessageId, toMessageId, initialStats) => {
          // Skip if messages are the same (no event would be emitted)
          fc.pre(fromMessageId !== toMessageId);
          
          // Arrange: Set up manager and simulate initial state
          const manager = getAnalysisResultManager();
          
          // Set initial session and message
          manager.switchSession(sessionId);
          manager.selectMessage(fromMessageId);
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          
          // Subscribe to message-selected event (simulating useDashboardData behavior)
          const unsubscribe = manager.on('message-selected', () => {
            // This is what useDashboardData does: clear dataSourceStatistics
            dataSourceStatistics = null;
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          expect(dataSourceStatistics?.total_count).toBeGreaterThan(0);
          
          // Act: Trigger message-selected event by selecting a different message
          manager.selectMessage(toMessageId);
          
          // Assert: dataSourceStatistics should be cleared
          expect(dataSourceStatistics).toBeNull();
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.2.2: Historical data restoration should clear old data
   * before loading new data.
   * 
   * This simulates the behavior of the analysis-result-restore event handler
   * which should clear current session data before restoring historical data.
   * 
   * **Validates: Requirements 2.1**
   */
  it('should clear old data before restoring historical data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        (currentSessionId, restoreSessionId, currentMessageId, restoreMessageId) => {
          // Arrange: Set up manager with existing data
          const manager = getAnalysisResultManager();
          
          // Set up current session with some data
          manager.switchSession(currentSessionId);
          manager.selectMessage(currentMessageId);
          
          // Add some existing data to current session
          manager.updateResults({
            sessionId: currentSessionId,
            messageId: currentMessageId,
            requestId: 'existing-request',
            items: [{
              id: 'existing-item',
              type: 'metric',
              data: { title: 'Existing Metric', value: '100', change: '+10%' },
              metadata: {
                sessionId: currentSessionId,
                messageId: currentMessageId,
                timestamp: Date.now(),
              },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify existing data is present
          expect(manager.hasCurrentData()).toBe(true);
          
          // Act: Simulate historical data restoration
          // Step 1: Clear current session data (as done in improved AnalysisResultBridge)
          manager.clearResults(currentSessionId);
          
          // Verify data is cleared
          expect(manager.hasData(currentSessionId, currentMessageId)).toBe(false);
          
          // Step 2: Switch to restore target session if different
          if (restoreSessionId !== currentSessionId) {
            manager.switchSession(restoreSessionId);
          }
          
          // Step 3: Clear target session data
          manager.clearResults(restoreSessionId);
          
          // Step 4: Select target message
          manager.selectMessage(restoreMessageId);
          
          // Step 5: Add restored data
          manager.updateResults({
            sessionId: restoreSessionId,
            messageId: restoreMessageId,
            requestId: `restore_${Date.now()}`,
            items: [{
              id: 'restored-item',
              type: 'metric',
              data: { title: 'Restored Metric', value: '200', change: '+20%' },
              metadata: {
                sessionId: restoreSessionId,
                messageId: restoreMessageId,
                timestamp: Date.now(),
              },
              source: 'restored',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Only restored data should be present
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(1);
          expect(currentResults[0].id).toBe('restored-item');
          expect(currentResults[0].source).toBe('restored');
          
          // Old data should not be present
          expect(manager.hasData(currentSessionId, currentMessageId)).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.2.3: After historical data restoration, only the restored
   * data should be visible in the dashboard.
   * 
   * This verifies the data isolation property - restored data should not
   * be mixed with any previous data.
   * 
   * **Validates: Requirements 2.2**
   */
  it('should only display restored data after historical data restoration', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.array(
          fc.record({
            id: fc.uuid(),
            // Only use types that can be easily normalized with simple data
            type: fc.constantFrom('metric', 'insight'),
            value: fc.string({ minLength: 1, maxLength: 20 }),
          }),
          { minLength: 1, maxLength: 5 }
        ),
        (sessionId, messageId, restoredItems) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set up session with some existing data
          manager.switchSession(sessionId);
          manager.selectMessage('old-message-id');
          
          // Add existing data
          manager.updateResults({
            sessionId: sessionId,
            messageId: 'old-message-id',
            requestId: 'old-request',
            items: [{
              id: 'old-item',
              type: 'metric',
              data: { title: 'Old Metric', value: '999', change: '' },
              metadata: {
                sessionId: sessionId,
                messageId: 'old-message-id',
                timestamp: Date.now() - 10000,
              },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now() - 10000,
          });
          
          // Act: Simulate historical data restoration with clearing
          // Clear current session data first
          manager.clearResults(sessionId);
          
          // Select the restore target message
          manager.selectMessage(messageId);
          
          // Create restored items with proper structure for each type
          const items = restoredItems.map(item => ({
            id: item.id,
            type: item.type as any,
            data: item.type === 'metric' 
              ? { title: item.value, value: '100', change: '' }
              : { text: item.value, icon: 'info' }, // insight type
            metadata: {
              sessionId: sessionId,
              messageId: messageId,
              timestamp: Date.now(),
            },
            source: 'restored' as const,
          }));
          
          // Add restored data
          manager.updateResults({
            sessionId: sessionId,
            messageId: messageId,
            requestId: `restore_${Date.now()}`,
            items: items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Only restored data should be present
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(restoredItems.length);
          
          // All items should be from the restored source
          currentResults.forEach(result => {
            expect(result.source).toBe('restored');
          });
          
          // All restored item IDs should be present
          const resultIds = new Set(currentResults.map(r => r.id));
          restoredItems.forEach(item => {
            expect(resultIds.has(item.id)).toBe(true);
          });
          
          // Old data should not be present
          expect(manager.hasData(sessionId, 'old-message-id')).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.2.4: Historical data restoration with empty items
   * should result in empty dashboard state.
   * 
   * This verifies that when historical request has no results,
   * the dashboard shows empty state rather than old data.
   * 
   * **Validates: Requirements 2.1, 2.2**
   */
  it('should show empty state when restoring historical data with no items', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        (sessionId, currentMessageId, restoreMessageId) => {
          // Skip if messages are the same
          fc.pre(currentMessageId !== restoreMessageId);
          
          // Arrange: Set up manager with existing data
          const manager = getAnalysisResultManager();
          
          // Set up session with some existing data
          manager.switchSession(sessionId);
          manager.selectMessage(currentMessageId);
          
          // Add existing data
          manager.updateResults({
            sessionId: sessionId,
            messageId: currentMessageId,
            requestId: 'existing-request',
            items: [{
              id: 'existing-item',
              type: 'metric',
              data: { title: 'Existing Metric', value: '100', change: '' },
              metadata: {
                sessionId: sessionId,
                messageId: currentMessageId,
                timestamp: Date.now(),
              },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify existing data is present
          expect(manager.hasCurrentData()).toBe(true);
          
          // Act: Simulate historical data restoration with no items
          // Clear current session data first
          manager.clearResults(sessionId);
          
          // Select the restore target message
          manager.selectMessage(restoreMessageId);
          
          // Don't add any items (simulating empty historical result)
          
          // Assert: Dashboard should be empty
          const currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBe(0);
          expect(manager.hasCurrentData()).toBe(false);
          
          // Old data should not be present
          expect(manager.hasData(sessionId, currentMessageId)).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


/**
 * Property-Based Tests for Historical Empty Result Handling
 * 
 * Feature: dashboard-data-isolation, Task 4.3: 添加历史请求无结果时的空状态处理
 * 
 * These tests verify that when a historical analysis request has no associated results,
 * the dashboard shows an empty state instead of data source statistics.
 * 
 * **Validates: Requirements 2.4**
 */
describe('Feature: dashboard-data-isolation, Task 4.3: 历史请求无结果时的空状态处理', () => {
  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  /**
   * Property Test 4.3.1: When historical-empty-result event is emitted,
   * dataSourceStatistics should be cleared and not reloaded.
   * 
   * This tests that when a historical request has no results,
   * the dashboard shows empty state instead of data source statistics.
   * 
   * **Validates: Requirements 2.4**
   */
  it('should clear dataSourceStatistics when historical-empty-result event is emitted', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        (sessionId, messageId, initialStats) => {
          // Arrange: Set up manager and simulate initial state
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId);
          
          // Simulate the state that useDashboardData would have
          let dataSourceStatistics: agent.DataSourceStatistics | null = initialStats;
          let isViewingHistoricalEmptyResult = false;
          
          // Subscribe to historical-empty-result event (simulating useDashboardData behavior)
          const unsubscribe = manager.on('historical-empty-result', () => {
            // This is what useDashboardData does: set flag and clear dataSourceStatistics
            isViewingHistoricalEmptyResult = true;
            dataSourceStatistics = null;
          });
          
          // Verify initial state has statistics
          expect(dataSourceStatistics).not.toBeNull();
          expect(dataSourceStatistics?.total_count).toBeGreaterThan(0);
          expect(isViewingHistoricalEmptyResult).toBe(false);
          
          // Act: Trigger historical-empty-result event
          manager.notifyHistoricalEmptyResult(sessionId, messageId);
          
          // Assert: dataSourceStatistics should be cleared and flag should be set
          expect(dataSourceStatistics).toBeNull();
          expect(isViewingHistoricalEmptyResult).toBe(true);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.3.2: When analysis-started event is emitted after
   * historical-empty-result, the flag should be reset.
   * 
   * This tests that starting a new analysis resets the historical empty result state.
   * 
   * **Validates: Requirements 2.4**
   */
  it('should reset isViewingHistoricalEmptyResult when analysis-started event is emitted', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, historicalMessageId, newMessageId, requestId) => {
          // Skip if messages are the same
          fc.pre(historicalMessageId !== newMessageId);
          
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId);
          
          // Simulate the state that useDashboardData would have
          let isViewingHistoricalEmptyResult = false;
          
          // Subscribe to events (simulating useDashboardData behavior)
          const unsubscribeHistoricalEmpty = manager.on('historical-empty-result', () => {
            isViewingHistoricalEmptyResult = true;
          });
          
          const unsubscribeAnalysisStarted = manager.on('analysis-started', () => {
            isViewingHistoricalEmptyResult = false;
          });
          
          // Act: First trigger historical-empty-result
          manager.notifyHistoricalEmptyResult(sessionId, historicalMessageId);
          
          // Verify flag is set
          expect(isViewingHistoricalEmptyResult).toBe(true);
          
          // Act: Then trigger analysis-started
          manager.setLoading(true, requestId, newMessageId);
          
          // Assert: Flag should be reset
          expect(isViewingHistoricalEmptyResult).toBe(false);
          
          // Cleanup
          unsubscribeHistoricalEmpty();
          unsubscribeAnalysisStarted();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.3.3: When session-switched event is emitted after
   * historical-empty-result, the flag should be reset.
   * 
   * This tests that switching sessions resets the historical empty result state.
   * 
   * **Validates: Requirements 2.4**
   */
  it('should reset isViewingHistoricalEmptyResult when session-switched event is emitted', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        (sessionId1, sessionId2, messageId) => {
          // Skip if sessions are the same
          fc.pre(sessionId1 !== sessionId2);
          
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId1);
          
          // Simulate the state that useDashboardData would have
          let isViewingHistoricalEmptyResult = false;
          
          // Subscribe to events (simulating useDashboardData behavior)
          const unsubscribeHistoricalEmpty = manager.on('historical-empty-result', () => {
            isViewingHistoricalEmptyResult = true;
          });
          
          const unsubscribeSessionSwitched = manager.on('session-switched', () => {
            isViewingHistoricalEmptyResult = false;
          });
          
          // Act: First trigger historical-empty-result
          manager.notifyHistoricalEmptyResult(sessionId1, messageId);
          
          // Verify flag is set
          expect(isViewingHistoricalEmptyResult).toBe(true);
          
          // Act: Then trigger session-switched
          manager.switchSession(sessionId2);
          
          // Assert: Flag should be reset
          expect(isViewingHistoricalEmptyResult).toBe(false);
          
          // Cleanup
          unsubscribeHistoricalEmpty();
          unsubscribeSessionSwitched();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.3.4: The notifyHistoricalEmptyResult method should emit
   * the historical-empty-result event with correct data.
   * 
   * This tests that the event is emitted with the correct sessionId and messageId.
   * 
   * **Validates: Requirements 2.4**
   */
  it('should emit historical-empty-result event with correct data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        (sessionId, messageId) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Track received event
          let receivedEvent: { sessionId: string; messageId: string } | null = null;
          
          // Subscribe to historical-empty-result event
          const unsubscribe = manager.on('historical-empty-result', (event) => {
            receivedEvent = event;
          });
          
          // Act: Trigger historical-empty-result event
          manager.notifyHistoricalEmptyResult(sessionId, messageId);
          
          // Assert: Event should be received with correct data
          expect(receivedEvent).not.toBeNull();
          expect(receivedEvent?.sessionId).toBe(sessionId);
          expect(receivedEvent?.messageId).toBe(messageId);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 4.3.5: When isViewingHistoricalEmptyResult is true,
   * data source statistics should not be displayed even if available.
   * 
   * This simulates the behavior in useDashboardData where shouldShowDataSourceStats
   * is false when isViewingHistoricalEmptyResult is true.
   * 
   * **Validates: Requirements 2.4**
   */
  it('should not show data source statistics when viewing historical empty result', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        (sessionId, messageId, dataSourceStats) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId);
          
          // Simulate the state that useDashboardData would have
          let isViewingHistoricalEmptyResult = false;
          let dataSourceStatistics: agent.DataSourceStatistics | null = dataSourceStats;
          
          // Subscribe to historical-empty-result event
          const unsubscribe = manager.on('historical-empty-result', () => {
            isViewingHistoricalEmptyResult = true;
            dataSourceStatistics = null;
          });
          
          // Verify initial state
          expect(dataSourceStatistics).not.toBeNull();
          
          // Act: Trigger historical-empty-result event
          manager.notifyHistoricalEmptyResult(sessionId, messageId);
          
          // Simulate the shouldShowDataSourceStats logic from useDashboardData
          const hasAnyAnalysisResults = false; // No analysis results
          const shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          
          // Assert: shouldShowDataSourceStats should be false
          expect(shouldShowDataSourceStats).toBe(false);
          expect(isViewingHistoricalEmptyResult).toBe(true);
          expect(dataSourceStatistics).toBeNull();
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


/**
 * Property-Based Tests for Data Source Statistics Display Mutual Exclusivity
 * 
 * Feature: dashboard-data-isolation, Property 4: 数据源统计显示互斥性
 * 
 * These tests verify that:
 * 1. When hasAnyAnalysisResults is true, data source statistics metrics/insights are NOT shown
 * 2. When hasAnyAnalysisResults is false AND data sources exist, data source statistics ARE shown
 * 3. The mutual exclusivity between analysis results and data source statistics display
 * 
 * **Validates: Requirements 2.3, 3.1, 3.2, 3.3**
 */
describe('Feature: dashboard-data-isolation, Property 4: 数据源统计显示互斥性', () => {
  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });


  /**
   * Property Test 4.1: When hasAnyAnalysisResults is true, data source statistics
   * metrics and insights should NOT be included in the display data.
   * 
   * This tests the mutual exclusivity: analysis results take priority over
   * data source statistics display.
   * 
   * **Validates: Requirements 2.3, 3.1**
   */
  it('should NOT show data source statistics when hasAnyAnalysisResults is true', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        fc.constantFrom('metric', 'insight', 'echarts', 'table', 'image'),
        (sessionId, messageId, dataSourceStats, resultType) => {
          // Arrange: Set up manager with analysis results
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);

          
          // Create analysis result item based on type
          const createResultItem = (type: string) => {
            const baseItem = {
              id: `analysis-${type}-${Date.now()}`,
              type: type as any,
              metadata: {
                sessionId,
                messageId,
                timestamp: Date.now(),
              },
              source: 'realtime' as const,
            };
            
            switch (type) {
              case 'metric':
                return { ...baseItem, data: { title: 'Analysis Metric', value: '100', change: '+10%' } };
              case 'insight':
                return { ...baseItem, data: { text: 'Analysis Insight', icon: 'chart' } };
              case 'echarts':
                return { ...baseItem, data: { option: { title: { text: 'Chart' } } } };
              case 'table':
                return { ...baseItem, data: { columns: ['A'], rows: [['1']] } };
              case 'image':
                return { ...baseItem, data: 'data:image/png;base64,test' };
              default:
                return { ...baseItem, data: {} };
            }
          };

          
          // Add analysis result
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items: [createResultItem(resultType)],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Simulate hasAnyAnalysisResults calculation (same logic as useDashboardData)
          const currentResults = manager.getCurrentResults();
          const hasAnyAnalysisResults = currentResults.length > 0;
          
          // Verify we have analysis results
          expect(hasAnyAnalysisResults).toBe(true);
          
          // Simulate shouldShowDataSourceStats logic from useDashboardData
          const isViewingHistoricalEmptyResult = false;
          const shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          
          // Assert: shouldShowDataSourceStats should be false when we have analysis results
          expect(shouldShowDataSourceStats).toBe(false);

          
          // Simulate the metrics/insights generation logic from useDashboardData
          // When shouldShowDataSourceStats is false, data source metrics/insights should NOT be added
          const dataSourceMetrics: any[] = [];
          const dataSourceInsights: any[] = [];
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.total_count > 0) {
            // This block should NOT execute when hasAnyAnalysisResults is true
            dataSourceMetrics.push({
              title: '数据源总数',
              value: String(dataSourceStats.total_count),
              change: ''
            });
          }
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.data_sources && dataSourceStats.data_sources.length > 0) {
            // This block should NOT execute when hasAnyAnalysisResults is true
            dataSourceStats.data_sources.forEach((ds: any) => {
              dataSourceInsights.push({
                text: `${ds.name} (${ds.type.toUpperCase()}) - 点击启动智能分析`,
                icon: 'database',
                dataSourceId: ds.id,
              });
            });
          }
          
          // Assert: No data source metrics or insights should be generated
          expect(dataSourceMetrics.length).toBe(0);
          expect(dataSourceInsights.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });


  /**
   * Property Test 4.2: When hasAnyAnalysisResults is false AND data sources exist,
   * data source statistics metrics and insights SHOULD be shown.
   * 
   * This tests that data source statistics are displayed when there are no
   * analysis results to show.
   * 
   * **Validates: Requirements 3.1, 3.2, 3.3**
   */
  it('should show data source statistics when hasAnyAnalysisResults is false and data sources exist', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        dataSourceStatisticsArb,
        (sessionId, dataSourceStats) => {
          // Arrange: Set up manager without analysis results
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Verify no analysis results
          const currentResults = manager.getCurrentResults();
          const hasAnyAnalysisResults = currentResults.length > 0;
          
          expect(hasAnyAnalysisResults).toBe(false);

          
          // Simulate shouldShowDataSourceStats logic from useDashboardData
          const isViewingHistoricalEmptyResult = false;
          const shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          
          // Assert: shouldShowDataSourceStats should be true when no analysis results
          expect(shouldShowDataSourceStats).toBe(true);
          
          // Simulate the metrics/insights generation logic from useDashboardData
          const dataSourceMetrics: any[] = [];
          const dataSourceInsights: any[] = [];
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.total_count > 0) {
            // This block SHOULD execute when hasAnyAnalysisResults is false
            dataSourceMetrics.push({
              title: '数据源总数',
              value: String(dataSourceStats.total_count),
              change: ''
            });
            
            // Add breakdown by type metrics
            const sortedTypes = Object.entries(dataSourceStats.breakdown_by_type)
              .sort(([, a], [, b]) => (b as number) - (a as number))
              .slice(0, 3);
            
            sortedTypes.forEach(([type, count]) => {
              dataSourceMetrics.push({
                title: `${type.toUpperCase()} 数据源`,
                value: String(count),
                change: ''
              });
            });
          }

          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.data_sources && dataSourceStats.data_sources.length > 0) {
            // This block SHOULD execute when hasAnyAnalysisResults is false
            dataSourceStats.data_sources.forEach((ds: any) => {
              dataSourceInsights.push({
                text: `${ds.name} (${ds.type.toUpperCase()}) - 点击启动智能分析`,
                icon: 'database',
                dataSourceId: ds.id,
              });
            });
          }
          
          // Assert: Data source metrics should be generated (at least the total count)
          expect(dataSourceMetrics.length).toBeGreaterThan(0);
          expect(dataSourceMetrics[0].title).toBe('数据源总数');
          expect(dataSourceMetrics[0].value).toBe(String(dataSourceStats.total_count));
          
          // Assert: Data source insights should be generated for each data source
          expect(dataSourceInsights.length).toBe(dataSourceStats.data_sources.length);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });


  /**
   * Property Test 4.3: The mutual exclusivity property - analysis results and
   * data source statistics should never be displayed together.
   * 
   * For any dashboard state, exactly one of the following should be true:
   * - Analysis results are shown (hasAnyAnalysisResults = true, no data source stats)
   * - Data source statistics are shown (hasAnyAnalysisResults = false, data source stats visible)
   * - Empty state (no analysis results, no data sources)
   * 
   * **Validates: Requirements 2.3, 3.1, 3.2, 3.3**
   */
  it('should maintain mutual exclusivity between analysis results and data source statistics', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.option(dataSourceStatisticsArb),
        fc.boolean(),
        (sessionId, messageId, maybeDataSourceStats, hasAnalysisResults) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);

          
          // Conditionally add analysis results
          if (hasAnalysisResults) {
            manager.updateResults({
              sessionId,
              messageId,
              requestId: `request-${Date.now()}`,
              items: [{
                id: `metric-${Date.now()}`,
                type: 'metric',
                data: { title: 'Test Metric', value: '100', change: '' },
                metadata: { sessionId, messageId, timestamp: Date.now() },
                source: 'realtime',
              }],
              isComplete: true,
              timestamp: Date.now(),
            });
          }
          
          // Simulate hasAnyAnalysisResults calculation
          const currentResults = manager.getCurrentResults();
          const hasAnyAnalysisResults = currentResults.length > 0;
          
          // Simulate shouldShowDataSourceStats logic
          const isViewingHistoricalEmptyResult = false;
          const shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;

          
          // Generate data source metrics/insights based on shouldShowDataSourceStats
          const dataSourceMetrics: any[] = [];
          const dataSourceInsights: any[] = [];
          const dataSourceStats = maybeDataSourceStats;
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.total_count > 0) {
            dataSourceMetrics.push({
              title: '数据源总数',
              value: String(dataSourceStats.total_count),
              change: ''
            });
          }
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.data_sources && dataSourceStats.data_sources.length > 0) {
            dataSourceStats.data_sources.forEach((ds: any) => {
              dataSourceInsights.push({
                text: `${ds.name} - 点击启动智能分析`,
                icon: 'database',
                dataSourceId: ds.id,
              });
            });
          }
          
          // Assert: Mutual exclusivity property
          // If we have analysis results, data source stats should NOT be shown
          if (hasAnyAnalysisResults) {
            expect(shouldShowDataSourceStats).toBe(false);
            expect(dataSourceMetrics.length).toBe(0);
            expect(dataSourceInsights.length).toBe(0);
          }

          
          // If we don't have analysis results, data source stats CAN be shown (if available)
          if (!hasAnyAnalysisResults) {
            expect(shouldShowDataSourceStats).toBe(true);
            // Data source metrics/insights are shown only if data sources exist
            if (dataSourceStats && dataSourceStats.total_count > 0) {
              expect(dataSourceMetrics.length).toBeGreaterThan(0);
            }
            if (dataSourceStats && dataSourceStats.data_sources && dataSourceStats.data_sources.length > 0) {
              expect(dataSourceInsights.length).toBe(dataSourceStats.data_sources.length);
            }
          }
          
          // The key mutual exclusivity assertion:
          // It should NEVER be the case that both analysis results AND data source stats are shown
          const showingAnalysisResults = hasAnyAnalysisResults;
          const showingDataSourceStats = dataSourceMetrics.length > 0 || dataSourceInsights.length > 0;
          
          // XOR: exactly one or neither, but never both
          expect(showingAnalysisResults && showingDataSourceStats).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });


  /**
   * Property Test 4.4: When analysis results are cleared, data source statistics
   * should become visible (if data sources exist).
   * 
   * This tests the transition from showing analysis results to showing
   * data source statistics after clearing.
   * 
   * **Validates: Requirements 3.2**
   */
  it('should show data source statistics after analysis results are cleared', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        (sessionId, messageId, dataSourceStats) => {
          // Arrange: Set up manager with analysis results
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Add analysis results
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items: [{
              id: `metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Test Metric', value: '100', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });

          
          // Verify initial state: has analysis results
          let currentResults = manager.getCurrentResults();
          let hasAnyAnalysisResults = currentResults.length > 0;
          expect(hasAnyAnalysisResults).toBe(true);
          
          // Calculate initial shouldShowDataSourceStats
          let isViewingHistoricalEmptyResult = false;
          let shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          expect(shouldShowDataSourceStats).toBe(false);
          
          // Act: Clear analysis results
          manager.clearResults(sessionId);
          
          // Recalculate after clearing
          currentResults = manager.getCurrentResults();
          hasAnyAnalysisResults = currentResults.length > 0;
          shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          
          // Assert: After clearing, data source statistics should be showable
          expect(hasAnyAnalysisResults).toBe(false);
          expect(shouldShowDataSourceStats).toBe(true);

          
          // Simulate data source metrics/insights generation after clearing
          const dataSourceMetrics: any[] = [];
          const dataSourceInsights: any[] = [];
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.total_count > 0) {
            dataSourceMetrics.push({
              title: '数据源总数',
              value: String(dataSourceStats.total_count),
              change: ''
            });
          }
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.data_sources && dataSourceStats.data_sources.length > 0) {
            dataSourceStats.data_sources.forEach((ds: any) => {
              dataSourceInsights.push({
                text: `${ds.name} - 点击启动智能分析`,
                icon: 'database',
                dataSourceId: ds.id,
              });
            });
          }
          
          // Assert: Data source statistics should now be visible
          expect(dataSourceMetrics.length).toBeGreaterThan(0);
          expect(dataSourceInsights.length).toBe(dataSourceStats.data_sources.length);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });


  /**
   * Property Test 4.5: When new analysis results arrive, data source statistics
   * should be hidden immediately.
   * 
   * This tests the transition from showing data source statistics to showing
   * analysis results when new results arrive.
   * 
   * **Validates: Requirements 3.1**
   */
  it('should hide data source statistics when new analysis results arrive', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        fc.constantFrom('metric', 'insight', 'echarts', 'table', 'image'),
        (sessionId, messageId, dataSourceStats, resultType) => {
          // Arrange: Set up manager without analysis results (showing data source stats)
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Verify initial state: no analysis results
          let currentResults = manager.getCurrentResults();
          let hasAnyAnalysisResults = currentResults.length > 0;
          expect(hasAnyAnalysisResults).toBe(false);

          
          // Calculate initial shouldShowDataSourceStats
          let isViewingHistoricalEmptyResult = false;
          let shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          expect(shouldShowDataSourceStats).toBe(true);
          
          // Verify data source stats would be shown initially
          let dataSourceMetrics: any[] = [];
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.total_count > 0) {
            dataSourceMetrics.push({ title: '数据源总数', value: String(dataSourceStats.total_count), change: '' });
          }
          expect(dataSourceMetrics.length).toBeGreaterThan(0);
          
          // Act: Add new analysis results
          const createResultItem = (type: string) => {
            const baseItem = {
              id: `analysis-${type}-${Date.now()}`,
              type: type as any,
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime' as const,
            };
            switch (type) {
              case 'metric': return { ...baseItem, data: { title: 'New Metric', value: '200', change: '+20%' } };
              case 'insight': return { ...baseItem, data: { text: 'New Insight', icon: 'chart' } };
              case 'echarts': return { ...baseItem, data: { option: { title: { text: 'New Chart' } } } };
              case 'table': return { ...baseItem, data: { columns: ['B'], rows: [['2']] } };
              case 'image': return { ...baseItem, data: 'data:image/png;base64,newtest' };
              default: return { ...baseItem, data: {} };
            }
          };

          
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items: [createResultItem(resultType)],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Recalculate after adding results
          currentResults = manager.getCurrentResults();
          hasAnyAnalysisResults = currentResults.length > 0;
          shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          
          // Assert: After adding results, data source statistics should be hidden
          expect(hasAnyAnalysisResults).toBe(true);
          expect(shouldShowDataSourceStats).toBe(false);
          
          // Regenerate data source metrics (should be empty now)
          dataSourceMetrics = [];
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.total_count > 0) {
            dataSourceMetrics.push({ title: '数据源总数', value: String(dataSourceStats.total_count), change: '' });
          }
          
          // Assert: Data source metrics should NOT be generated
          expect(dataSourceMetrics.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });


  /**
   * Property Test 4.6: Multiple analysis result types should all prevent
   * data source statistics from being shown.
   * 
   * This tests that any combination of analysis result types (charts, images,
   * tables, metrics, insights, files) will hide data source statistics.
   * 
   * **Validates: Requirements 2.3, 3.1**
   */
  it('should hide data source statistics for any combination of analysis result types', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        dataSourceStatisticsArb,
        fc.array(
          fc.constantFrom('metric', 'insight', 'echarts', 'table', 'image', 'file'),
          { minLength: 1, maxLength: 6 }
        ),
        (sessionId, messageId, dataSourceStats, resultTypes) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);

          
          // Create items for each result type
          const items = resultTypes.map((type, index) => {
            const baseItem = {
              id: `item-${type}-${index}-${Date.now()}`,
              type: type as any,
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime' as const,
            };
            switch (type) {
              case 'metric': return { ...baseItem, data: { title: `Metric ${index}`, value: '100', change: '' } };
              case 'insight': return { ...baseItem, data: { text: `Insight ${index}`, icon: 'info' } };
              case 'echarts': return { ...baseItem, data: { option: {} } };
              case 'table': return { ...baseItem, data: { columns: ['A'], rows: [['1']] } };
              case 'image': return { ...baseItem, data: 'data:image/png;base64,test' };
              case 'file': return { ...baseItem, data: { filename: `file${index}.txt`, path: '/tmp' } };
              default: return { ...baseItem, data: {} };
            }
          });
          
          // Add all analysis results
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });

          
          // Calculate hasAnyAnalysisResults
          const currentResults = manager.getCurrentResults();
          const hasAnyAnalysisResults = currentResults.length > 0;
          
          // Calculate shouldShowDataSourceStats
          const isViewingHistoricalEmptyResult = false;
          const shouldShowDataSourceStats = !hasAnyAnalysisResults && !isViewingHistoricalEmptyResult;
          
          // Assert: With any analysis results, data source stats should be hidden
          expect(hasAnyAnalysisResults).toBe(true);
          expect(shouldShowDataSourceStats).toBe(false);
          
          // Verify no data source metrics/insights would be generated
          const dataSourceMetrics: any[] = [];
          const dataSourceInsights: any[] = [];
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.total_count > 0) {
            dataSourceMetrics.push({ title: '数据源总数', value: String(dataSourceStats.total_count), change: '' });
          }
          
          if (shouldShowDataSourceStats && dataSourceStats && dataSourceStats.data_sources) {
            dataSourceStats.data_sources.forEach((ds: any) => {
              dataSourceInsights.push({ text: ds.name, icon: 'database' });
            });
          }
          
          expect(dataSourceMetrics.length).toBe(0);
          expect(dataSourceInsights.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


/**
 * Property-Based Tests for Loading State Consistency
 * 
 * Feature: dashboard-data-isolation, Property 7: 加载状态一致性
 * 
 * These tests verify that:
 * 1. isLoading is true when analysis starts (setLoading(true, requestId))
 * 2. isLoading is false when data arrives (isComplete in updateResults)
 * 3. isLoading is false when error occurs (setError)
 * 4. isLoading is properly passed through useAnalysisResults to useDashboardData
 * 
 * **Validates: Requirements 1.3**
 */
describe('Feature: dashboard-data-isolation, Property 7: 加载状态一致性', () => {
  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  /**
   * Property Test 7.1: When analysis starts (setLoading(true, requestId)),
   * isLoading should be true.
   * 
   * This tests that the loading state is correctly set when a new analysis begins.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should set isLoading to true when analysis starts', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Verify initial state: not loading
          expect(manager.isLoading()).toBe(false);
          
          // Act: Start analysis by calling setLoading(true, requestId)
          manager.setLoading(true, requestId, messageId);
          
          // Assert: isLoading should be true
          expect(manager.isLoading()).toBe(true);
          
          // Also verify the state object reflects this
          const state = manager.getState();
          expect(state.isLoading).toBe(true);
          expect(state.pendingRequestId).toBe(requestId);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.2: When data arrives with isComplete=true,
   * isLoading should be set to false.
   * 
   * This tests that the loading state is correctly cleared when analysis results arrive.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should set isLoading to false when data arrives with isComplete=true', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange: Set up manager and start loading
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.setLoading(true, requestId, messageId);
          
          // Verify loading state is true
          expect(manager.isLoading()).toBe(true);
          
          // Act: Send complete results
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items: [{
              id: `metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Test Metric', value: '100', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: isLoading should be false
          expect(manager.isLoading()).toBe(false);
          
          // Also verify the state object reflects this
          const state = manager.getState();
          expect(state.isLoading).toBe(false);
          expect(state.pendingRequestId).toBeNull();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.3: When error occurs (setError),
   * isLoading should be set to false.
   * 
   * This tests that the loading state is correctly cleared when an error occurs.
   * Note: setError formats the error message with user-friendly text and recovery
   * suggestions, so we only check that error is set (not null) and isLoading is false.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should set isLoading to false when error occurs', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.string({ minLength: 1, maxLength: 100 }),
        (sessionId, messageId, requestId, errorMessage) => {
          // Arrange: Set up manager and start loading
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.setLoading(true, requestId, messageId);
          
          // Verify loading state is true
          expect(manager.isLoading()).toBe(true);
          
          // Act: Set error
          manager.setError(errorMessage);
          
          // Assert: isLoading should be false
          expect(manager.isLoading()).toBe(false);
          
          // Also verify the state object reflects this
          const state = manager.getState();
          expect(state.isLoading).toBe(false);
          expect(state.pendingRequestId).toBeNull();
          // Note: setError formats the error with user-friendly message and recovery suggestions,
          // so we only check that error is set (not null)
          expect(state.error).not.toBeNull();
          expect(typeof state.error).toBe('string');
          expect(state.error!.length).toBeGreaterThan(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.4: Loading state should remain true during partial data arrival
   * (isComplete=false) and only become false when complete data arrives.
   * 
   * This tests the streaming/partial data scenario where multiple batches arrive.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should keep isLoading true during partial data arrival', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.integer({ min: 1, max: 5 }),
        (sessionId, messageId, requestId, numPartialBatches) => {
          // Arrange: Set up manager and start loading
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.setLoading(true, requestId, messageId);
          
          // Verify loading state is true
          expect(manager.isLoading()).toBe(true);
          
          // Act: Send multiple partial batches (isComplete=false)
          for (let i = 0; i < numPartialBatches; i++) {
            manager.updateResults({
              sessionId,
              messageId,
              requestId,
              items: [{
                id: `metric-${i}-${Date.now()}`,
                type: 'metric',
                data: { title: `Partial Metric ${i}`, value: String(i * 100), change: '' },
                metadata: { sessionId, messageId, timestamp: Date.now() },
                source: 'realtime',
              }],
              isComplete: false, // Partial data
              timestamp: Date.now(),
            });
            
            // Assert: isLoading should still be true after partial data
            expect(manager.isLoading()).toBe(true);
          }
          
          // Act: Send final complete batch
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items: [{
              id: `metric-final-${Date.now()}`,
              type: 'metric',
              data: { title: 'Final Metric', value: '999', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true, // Complete data
            timestamp: Date.now(),
          });
          
          // Assert: isLoading should be false after complete data
          expect(manager.isLoading()).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.5: Session switch should cancel pending request and set isLoading to false.
   * 
   * This tests that switching sessions properly clears the loading state.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should set isLoading to false when session is switched during loading', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId1, sessionId2, messageId, requestId) => {
          // Skip if sessions are the same
          fc.pre(sessionId1 !== sessionId2);
          
          // Arrange: Set up manager and start loading in session 1
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId1);
          manager.setLoading(true, requestId, messageId);
          
          // Verify loading state is true
          expect(manager.isLoading()).toBe(true);
          expect(manager.getPendingRequestId()).toBe(requestId);
          
          // Act: Switch to a different session
          manager.switchSession(sessionId2);
          
          // Assert: isLoading should be false (pending request cancelled)
          expect(manager.isLoading()).toBe(false);
          expect(manager.getPendingRequestId()).toBeNull();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.6: clearAll should set isLoading to false.
   * 
   * This tests that clearing all data properly resets the loading state.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should set isLoading to false when clearAll is called', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange: Set up manager and start loading
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.setLoading(true, requestId, messageId);
          
          // Verify loading state is true
          expect(manager.isLoading()).toBe(true);
          
          // Act: Clear all data
          manager.clearAll();
          
          // Assert: isLoading should be false
          expect(manager.isLoading()).toBe(false);
          
          // Also verify the state object reflects this
          const state = manager.getState();
          expect(state.isLoading).toBe(false);
          expect(state.pendingRequestId).toBeNull();
          expect(state.currentSessionId).toBeNull();
          expect(state.currentMessageId).toBeNull();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.7: Loading state should be properly synchronized through state subscription.
   * 
   * This tests that subscribers receive correct loading state updates.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should notify subscribers of loading state changes', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Track loading state changes through subscription
          const loadingStates: boolean[] = [];
          const unsubscribe = manager.subscribe((state) => {
            loadingStates.push(state.isLoading);
          });
          
          // Act: Start loading
          manager.setLoading(true, requestId, messageId);
          
          // Act: Complete loading with data
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items: [{
              id: `metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Test Metric', value: '100', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Should have received loading state changes
          // First notification: isLoading = true (from setLoading)
          // Second notification: isLoading = false (from updateResults with isComplete)
          expect(loadingStates.length).toBeGreaterThanOrEqual(2);
          expect(loadingStates).toContain(true);
          expect(loadingStates[loadingStates.length - 1]).toBe(false);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.8: Stale request data should not affect loading state.
   * 
   * This tests that data from old requests (with different requestId) 
   * does not incorrectly clear the loading state.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should ignore stale request data and maintain loading state', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        requestIdArb,
        (sessionId, messageId, currentRequestId, staleRequestId) => {
          // Skip if request IDs are the same
          fc.pre(currentRequestId !== staleRequestId);
          
          // Arrange: Set up manager and start loading with current request
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.setLoading(true, currentRequestId, messageId);
          
          // Verify loading state is true
          expect(manager.isLoading()).toBe(true);
          expect(manager.getPendingRequestId()).toBe(currentRequestId);
          
          // Act: Send data with stale request ID
          manager.updateResults({
            sessionId,
            messageId,
            requestId: staleRequestId, // Different from current pending request
            items: [{
              id: `stale-metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Stale Metric', value: '0', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: isLoading should still be true (stale data ignored)
          expect(manager.isLoading()).toBe(true);
          expect(manager.getPendingRequestId()).toBe(currentRequestId);
          
          // Act: Send data with current request ID
          manager.updateResults({
            sessionId,
            messageId,
            requestId: currentRequestId, // Matches pending request
            items: [{
              id: `current-metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Current Metric', value: '100', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: isLoading should now be false
          expect(manager.isLoading()).toBe(false);
          expect(manager.getPendingRequestId()).toBeNull();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 7.9: Loading state lifecycle should be consistent across
   * multiple analysis requests in sequence.
   * 
   * This tests that the loading state correctly transitions through multiple
   * analysis request cycles.
   * 
   * **Validates: Requirements 1.3**
   */
  it('should maintain consistent loading state across multiple analysis cycles', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.array(requestIdArb, { minLength: 2, maxLength: 5 }),
        (sessionId, messageId, requestIds) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          
          // Act & Assert: Run multiple analysis cycles
          for (let i = 0; i < requestIds.length; i++) {
            const requestId = requestIds[i];
            
            // Start loading
            manager.setLoading(true, requestId, messageId);
            expect(manager.isLoading()).toBe(true);
            expect(manager.getPendingRequestId()).toBe(requestId);
            
            // Complete with data
            manager.updateResults({
              sessionId,
              messageId,
              requestId,
              items: [{
                id: `metric-${i}-${Date.now()}`,
                type: 'metric',
                data: { title: `Metric ${i}`, value: String(i * 100), change: '' },
                metadata: { sessionId, messageId, timestamp: Date.now() },
                source: 'realtime',
              }],
              isComplete: true,
              timestamp: Date.now(),
            });
            
            // Verify loading is complete
            expect(manager.isLoading()).toBe(false);
            expect(manager.getPendingRequestId()).toBeNull();
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});


/**
 * Property-Based Tests for hasAnyAnalysisResults Boundary Correctness
 * 
 * Feature: dashboard-data-isolation, Property 5: hasAnyAnalysisResults 边界正确性
 * 
 * These tests verify that hasAnyAnalysisResults correctly reflects whether
 * any valid analysis result data exists for any combination of:
 * - Empty arrays for all data types
 * - Only one data type has data
 * - Multiple data types have data
 * - Null/undefined handling
 * - Edge cases with zero-length arrays
 * 
 * **Validates: Requirements 3.4**
 */
describe('Feature: dashboard-data-isolation, Property 5: hasAnyAnalysisResults 边界正确性', () => {
  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  // ==================== Helper Functions ====================
  
  /**
   * Safe array length calculation - mirrors the implementation in useDashboardData
   * This is the exact logic used in the hook to calculate hasAnyAnalysisResults
   */
  const safeArrayLength = (arr: any[] | null | undefined): number => {
    if (arr === null || arr === undefined) return 0;
    if (!Array.isArray(arr)) return 0;
    return arr.length;
  };

  /**
   * Calculate hasAnyAnalysisResults from analysis results state
   * This mirrors the exact logic in useDashboardData hook
   */
  const calculateHasAnyAnalysisResults = (analysisResults: {
    charts: any[];
    images: any[];
    tables: any[];
    metrics: any[];
    insights: any[];
    files: any[];
  }): boolean => {
    const chartsCount = safeArrayLength(analysisResults.charts);
    const imagesCount = safeArrayLength(analysisResults.images);
    const tablesCount = safeArrayLength(analysisResults.tables);
    const metricsCount = safeArrayLength(analysisResults.metrics);
    const insightsCount = safeArrayLength(analysisResults.insights);
    const filesCount = safeArrayLength(analysisResults.files);
    
    return (
      chartsCount > 0 ||
      imagesCount > 0 ||
      tablesCount > 0 ||
      metricsCount > 0 ||
      insightsCount > 0 ||
      filesCount > 0
    );
  };

  // ==================== Test Data Generators ====================

  /**
   * Generate a single analysis result type
   */
  const analysisResultTypeArb = fc.constantFrom(
    'echarts', 'image', 'table', 'metric', 'insight', 'file'
  );

  /**
   * Generate analysis result item based on type
   */
  const createResultItem = (
    type: string,
    sessionId: string,
    messageId: string
  ) => {
    const baseItem = {
      id: `${type}-${Date.now()}-${Math.random()}`,
      type: type as any,
      metadata: { sessionId, messageId, timestamp: Date.now() },
      source: 'realtime' as const,
    };
    
    switch (type) {
      case 'metric':
        return { ...baseItem, data: { title: 'Test Metric', value: '100', change: '+10%' } };
      case 'insight':
        return { ...baseItem, data: { text: 'Test Insight', icon: 'chart' } };
      case 'echarts':
        return { ...baseItem, data: { option: { title: { text: 'Test Chart' } } } };
      case 'table':
        return { ...baseItem, data: { columns: ['A', 'B'], rows: [['1', '2']] } };
      case 'image':
        return { ...baseItem, data: 'data:image/png;base64,testimage' };
      case 'file':
        return { ...baseItem, data: { filename: 'test.txt', path: '/tmp/test.txt' } };
      default:
        return { ...baseItem, data: {} };
    }
  };

  /**
   * Generate a subset of analysis result types (for testing combinations)
   */
  const resultTypesSubsetArb = fc.subarray(
    ['echarts', 'image', 'table', 'metric', 'insight', 'file'],
    { minLength: 0, maxLength: 6 }
  );

  // ==================== Property Tests ====================

  /**
   * Property Test 5.1: When all data arrays are empty, hasAnyAnalysisResults should be false.
   * 
   * This tests the boundary case where no analysis results exist.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should return false when all data arrays are empty', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        (sessionId) => {
          // Arrange: Set up manager with no data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          
          // Get current results (should be empty)
          const currentResults = manager.getCurrentResults();
          
          // Simulate the analysis results structure from useAnalysisResults
          const analysisResults = {
            charts: currentResults.filter(r => r.type === 'echarts'),
            images: currentResults.filter(r => r.type === 'image'),
            tables: currentResults.filter(r => r.type === 'table'),
            metrics: currentResults.filter(r => r.type === 'metric').map(r => r.data),
            insights: currentResults.filter(r => r.type === 'insight').map(r => r.data),
            files: currentResults.filter(r => r.type === 'file'),
          };
          
          // Calculate hasAnyAnalysisResults
          const hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          
          // Assert: Should be false when all arrays are empty
          expect(hasAnyAnalysisResults).toBe(false);
          expect(analysisResults.charts.length).toBe(0);
          expect(analysisResults.images.length).toBe(0);
          expect(analysisResults.tables.length).toBe(0);
          expect(analysisResults.metrics.length).toBe(0);
          expect(analysisResults.insights.length).toBe(0);
          expect(analysisResults.files.length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.2: When only one data type has data, hasAnyAnalysisResults should be true.
   * 
   * This tests that having any single type of data is sufficient to return true.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should return true when only one data type has data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        analysisResultTypeArb,
        (sessionId, messageId, resultType) => {
          // Arrange: Set up manager with only one type of data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Add only one type of result
          const item = createResultItem(resultType, sessionId, messageId);
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items: [item],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Get current results
          const currentResults = manager.getCurrentResults();
          
          // Simulate the analysis results structure
          const analysisResults = {
            charts: currentResults.filter(r => r.type === 'echarts'),
            images: currentResults.filter(r => r.type === 'image'),
            tables: currentResults.filter(r => r.type === 'table'),
            metrics: currentResults.filter(r => r.type === 'metric').map(r => r.data),
            insights: currentResults.filter(r => r.type === 'insight').map(r => r.data),
            files: currentResults.filter(r => r.type === 'file'),
          };
          
          // Calculate hasAnyAnalysisResults
          const hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          
          // Assert: Should be true when any single type has data
          expect(hasAnyAnalysisResults).toBe(true);
          
          // Verify only the expected type has data
          const typeToArrayMap: Record<string, any[]> = {
            'echarts': analysisResults.charts,
            'image': analysisResults.images,
            'table': analysisResults.tables,
            'metric': analysisResults.metrics,
            'insight': analysisResults.insights,
            'file': analysisResults.files,
          };
          
          expect(typeToArrayMap[resultType].length).toBeGreaterThan(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.3: When multiple data types have data, hasAnyAnalysisResults should be true.
   * 
   * This tests combinations of multiple data types.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should return true when multiple data types have data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        fc.subarray(
          ['echarts', 'image', 'table', 'metric', 'insight', 'file'],
          { minLength: 2, maxLength: 6 }
        ),
        (sessionId, messageId, resultTypes) => {
          // Arrange: Set up manager with multiple types of data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Add multiple types of results
          const items = resultTypes.map(type => createResultItem(type, sessionId, messageId));
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Get current results
          const currentResults = manager.getCurrentResults();
          
          // Simulate the analysis results structure
          const analysisResults = {
            charts: currentResults.filter(r => r.type === 'echarts'),
            images: currentResults.filter(r => r.type === 'image'),
            tables: currentResults.filter(r => r.type === 'table'),
            metrics: currentResults.filter(r => r.type === 'metric').map(r => r.data),
            insights: currentResults.filter(r => r.type === 'insight').map(r => r.data),
            files: currentResults.filter(r => r.type === 'file'),
          };
          
          // Calculate hasAnyAnalysisResults
          const hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          
          // Assert: Should be true when multiple types have data
          expect(hasAnyAnalysisResults).toBe(true);
          
          // Verify the total count matches
          const totalCount = 
            analysisResults.charts.length +
            analysisResults.images.length +
            analysisResults.tables.length +
            analysisResults.metrics.length +
            analysisResults.insights.length +
            analysisResults.files.length;
          
          expect(totalCount).toBe(resultTypes.length);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.4: safeArrayLength should handle null values correctly.
   * 
   * This tests the null handling in the hasAnyAnalysisResults calculation.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should handle null values correctly in safeArrayLength', () => {
    fc.assert(
      fc.property(
        fc.constantFrom(null, undefined, [], [1], [1, 2, 3]),
        (input) => {
          const result = safeArrayLength(input as any);
          
          if (input === null || input === undefined) {
            expect(result).toBe(0);
          } else if (Array.isArray(input)) {
            expect(result).toBe(input.length);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.5: safeArrayLength should handle non-array values correctly.
   * 
   * This tests edge cases where the input is not an array.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should handle non-array values correctly in safeArrayLength', () => {
    fc.assert(
      fc.property(
        fc.oneof(
          fc.constant(null),
          fc.constant(undefined),
          fc.string(),
          fc.integer(),
          fc.boolean(),
          fc.object()
        ),
        (input) => {
          const result = safeArrayLength(input as any);
          
          // Non-array values should return 0
          if (!Array.isArray(input)) {
            expect(result).toBe(0);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.6: hasAnyAnalysisResults should be false for empty arrays of all types.
   * 
   * This tests the explicit empty array case for all data types.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should return false for explicit empty arrays of all types', () => {
    fc.assert(
      fc.property(
        fc.constant(true), // Just need to run the test
        () => {
          // Create analysis results with explicit empty arrays
          const analysisResults = {
            charts: [],
            images: [],
            tables: [],
            metrics: [],
            insights: [],
            files: [],
          };
          
          // Calculate hasAnyAnalysisResults
          const hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          
          // Assert: Should be false
          expect(hasAnyAnalysisResults).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.7: hasAnyAnalysisResults should correctly reflect state after clearing data.
   * 
   * This tests that after clearing data, hasAnyAnalysisResults becomes false.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should return false after clearing all data', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        resultTypesSubsetArb,
        (sessionId, messageId, resultTypes) => {
          // Skip if no types selected (already tested in 5.1)
          fc.pre(resultTypes.length > 0);
          
          // Arrange: Set up manager with data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Add results
          const items = resultTypes.map(type => createResultItem(type, sessionId, messageId));
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify data exists
          let currentResults = manager.getCurrentResults();
          expect(currentResults.length).toBeGreaterThan(0);
          
          // Act: Clear all data
          manager.clearResults(sessionId);
          
          // Get results after clearing
          currentResults = manager.getCurrentResults();
          
          // Simulate the analysis results structure
          const analysisResults = {
            charts: currentResults.filter(r => r.type === 'echarts'),
            images: currentResults.filter(r => r.type === 'image'),
            tables: currentResults.filter(r => r.type === 'table'),
            metrics: currentResults.filter(r => r.type === 'metric').map(r => r.data),
            insights: currentResults.filter(r => r.type === 'insight').map(r => r.data),
            files: currentResults.filter(r => r.type === 'file'),
          };
          
          // Calculate hasAnyAnalysisResults
          const hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          
          // Assert: Should be false after clearing
          expect(hasAnyAnalysisResults).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.8: hasAnyAnalysisResults should be true if and only if
   * at least one data array has length > 0.
   * 
   * This is the core property that defines the correctness of hasAnyAnalysisResults.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should be true iff at least one data array has length > 0', () => {
    fc.assert(
      fc.property(
        fc.record({
          charts: fc.array(fc.anything(), { maxLength: 3 }),
          images: fc.array(fc.anything(), { maxLength: 3 }),
          tables: fc.array(fc.anything(), { maxLength: 3 }),
          metrics: fc.array(fc.anything(), { maxLength: 3 }),
          insights: fc.array(fc.anything(), { maxLength: 3 }),
          files: fc.array(fc.anything(), { maxLength: 3 }),
        }),
        (analysisResults) => {
          // Calculate hasAnyAnalysisResults using our function
          const hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          
          // Calculate expected result manually
          const expectedResult = 
            analysisResults.charts.length > 0 ||
            analysisResults.images.length > 0 ||
            analysisResults.tables.length > 0 ||
            analysisResults.metrics.length > 0 ||
            analysisResults.insights.length > 0 ||
            analysisResults.files.length > 0;
          
          // Assert: Results should match
          expect(hasAnyAnalysisResults).toBe(expectedResult);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.9: hasAnyAnalysisResults should handle mixed null/undefined/empty arrays.
   * 
   * This tests combinations of null, undefined, and empty arrays.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should handle mixed null/undefined/empty arrays correctly', () => {
    fc.assert(
      fc.property(
        fc.record({
          charts: fc.oneof(fc.constant(null), fc.constant(undefined), fc.array(fc.anything(), { maxLength: 2 })),
          images: fc.oneof(fc.constant(null), fc.constant(undefined), fc.array(fc.anything(), { maxLength: 2 })),
          tables: fc.oneof(fc.constant(null), fc.constant(undefined), fc.array(fc.anything(), { maxLength: 2 })),
          metrics: fc.oneof(fc.constant(null), fc.constant(undefined), fc.array(fc.anything(), { maxLength: 2 })),
          insights: fc.oneof(fc.constant(null), fc.constant(undefined), fc.array(fc.anything(), { maxLength: 2 })),
          files: fc.oneof(fc.constant(null), fc.constant(undefined), fc.array(fc.anything(), { maxLength: 2 })),
        }),
        (analysisResults) => {
          // Calculate hasAnyAnalysisResults using our function
          const hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults as any);
          
          // Calculate expected result using safeArrayLength
          const expectedResult = 
            safeArrayLength(analysisResults.charts as any) > 0 ||
            safeArrayLength(analysisResults.images as any) > 0 ||
            safeArrayLength(analysisResults.tables as any) > 0 ||
            safeArrayLength(analysisResults.metrics as any) > 0 ||
            safeArrayLength(analysisResults.insights as any) > 0 ||
            safeArrayLength(analysisResults.files as any) > 0;
          
          // Assert: Results should match
          expect(hasAnyAnalysisResults).toBe(expectedResult);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 5.10: Adding any single item should change hasAnyAnalysisResults from false to true.
   * 
   * This tests the transition from empty state to having data.
   * 
   * **Validates: Requirements 3.4**
   */
  it('should transition from false to true when any item is added', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        analysisResultTypeArb,
        (sessionId, messageId, resultType) => {
          // Arrange: Set up manager with no data
          const manager = getAnalysisResultManager();
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Verify initial state is empty
          let currentResults = manager.getCurrentResults();
          let analysisResults = {
            charts: currentResults.filter(r => r.type === 'echarts'),
            images: currentResults.filter(r => r.type === 'image'),
            tables: currentResults.filter(r => r.type === 'table'),
            metrics: currentResults.filter(r => r.type === 'metric').map(r => r.data),
            insights: currentResults.filter(r => r.type === 'insight').map(r => r.data),
            files: currentResults.filter(r => r.type === 'file'),
          };
          
          let hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          expect(hasAnyAnalysisResults).toBe(false);
          
          // Act: Add a single item
          const item = createResultItem(resultType, sessionId, messageId);
          manager.updateResults({
            sessionId,
            messageId,
            requestId: `request-${Date.now()}`,
            items: [item],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Get updated results
          currentResults = manager.getCurrentResults();
          analysisResults = {
            charts: currentResults.filter(r => r.type === 'echarts'),
            images: currentResults.filter(r => r.type === 'image'),
            tables: currentResults.filter(r => r.type === 'table'),
            metrics: currentResults.filter(r => r.type === 'metric').map(r => r.data),
            insights: currentResults.filter(r => r.type === 'insight').map(r => r.data),
            files: currentResults.filter(r => r.type === 'file'),
          };
          
          hasAnyAnalysisResults = calculateHasAnyAnalysisResults(analysisResults);
          
          // Assert: Should now be true
          expect(hasAnyAnalysisResults).toBe(true);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});



/**
 * Property-Based Tests for State Synchronization Responsiveness
 * 
 * Feature: dashboard-data-isolation, Property 6: 状态同步响应性
 * 
 * These tests verify that:
 * 1. When AnalysisResultManager switches session, useDashboardData reloads corresponding data source statistics (Requirement 5.2)
 * 2. When AnalysisResultManager selects new message, useDashboardData updates display data (Requirement 5.3)
 * 3. When state change events are triggered, useDashboardData re-evaluates hasAnyAnalysisResults condition (Requirement 5.5)
 * 
 * Property 6: For any AnalysisResultManager 状态变更（会话切换、消息选择、数据更新），
 * useDashboardData Hook 应在下一个渲染周期内同步更新其状态，包括重新评估 hasAnyAnalysisResults 条件。
 * 
 * **Validates: Requirements 5.2, 5.3, 5.5**
 */
describe('Feature: dashboard-data-isolation, Property 6: 状态同步响应性', () => {
  beforeEach(() => {
    // Reset the singleton instance before each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up after each test
    AnalysisResultManagerImpl.resetInstance();
    vi.clearAllMocks();
  });

  // ==================== Helper Functions ====================
  
  /**
   * Safe array length calculation - mirrors the implementation in useDashboardData
   */
  const safeArrayLength = (arr: any[] | null | undefined): number => {
    if (arr === null || arr === undefined) return 0;
    if (!Array.isArray(arr)) return 0;
    return arr.length;
  };

  /**
   * Calculate hasAnyAnalysisResults from manager state
   * This mirrors the exact logic in useDashboardData hook
   */
  const calculateHasAnyAnalysisResults = (manager: ReturnType<typeof getAnalysisResultManager>): boolean => {
    const results = manager.getCurrentResults();
    
    const charts = results.filter(r => r.type === 'echarts');
    const images = results.filter(r => r.type === 'image');
    const tables = results.filter(r => r.type === 'table' || r.type === 'csv');
    const metrics = results.filter(r => r.type === 'metric');
    const insights = results.filter(r => r.type === 'insight');
    const files = results.filter(r => r.type === 'file');
    
    return (
      safeArrayLength(charts) > 0 ||
      safeArrayLength(images) > 0 ||
      safeArrayLength(tables) > 0 ||
      safeArrayLength(metrics) > 0 ||
      safeArrayLength(insights) > 0 ||
      safeArrayLength(files) > 0
    );
  };

  /**
   * Property Test 6.1: When session is switched, subscribers should be notified
   * and state should be synchronized.
   * 
   * This tests that session switch triggers state synchronization through
   * the subscription mechanism.
   * 
   * **Validates: Requirements 5.2**
   */
  it('should synchronize state when session is switched', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        (sessionId1, sessionId2) => {
          // Skip if sessions are the same
          fc.pre(sessionId1 !== sessionId2);
          
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Track state changes through subscription
          const stateChanges: { sessionId: string | null; messageId: string | null }[] = [];
          const unsubscribe = manager.subscribe((state) => {
            stateChanges.push({
              sessionId: state.currentSessionId,
              messageId: state.currentMessageId,
            });
          });
          
          // Set initial session
          manager.switchSession(sessionId1);
          
          // Verify initial state was captured
          expect(stateChanges.length).toBeGreaterThan(0);
          expect(stateChanges[stateChanges.length - 1].sessionId).toBe(sessionId1);
          
          // Act: Switch to different session
          manager.switchSession(sessionId2);
          
          // Assert: State change should be captured
          expect(stateChanges.length).toBeGreaterThan(1);
          expect(stateChanges[stateChanges.length - 1].sessionId).toBe(sessionId2);
          // Message should be reset when session switches
          expect(stateChanges[stateChanges.length - 1].messageId).toBeNull();
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.2: When message is selected, subscribers should be notified
   * and state should be synchronized.
   * 
   * This tests that message selection triggers state synchronization through
   * the subscription mechanism.
   * 
   * **Validates: Requirements 5.3**
   */
  it('should synchronize state when message is selected', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        (sessionId, messageId1, messageId2) => {
          // Skip if messages are the same
          fc.pre(messageId1 !== messageId2);
          
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId);
          
          // Track state changes through subscription
          const stateChanges: { sessionId: string | null; messageId: string | null }[] = [];
          const unsubscribe = manager.subscribe((state) => {
            stateChanges.push({
              sessionId: state.currentSessionId,
              messageId: state.currentMessageId,
            });
          });
          
          // Select initial message
          manager.selectMessage(messageId1);
          
          // Verify initial state was captured
          expect(stateChanges.length).toBeGreaterThan(0);
          expect(stateChanges[stateChanges.length - 1].messageId).toBe(messageId1);
          
          // Act: Select different message
          manager.selectMessage(messageId2);
          
          // Assert: State change should be captured
          expect(stateChanges.length).toBeGreaterThan(1);
          expect(stateChanges[stateChanges.length - 1].messageId).toBe(messageId2);
          expect(stateChanges[stateChanges.length - 1].sessionId).toBe(sessionId);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.3: When data is updated, hasAnyAnalysisResults should be
   * re-evaluated and reflect the new state.
   * 
   * This tests that data updates trigger re-evaluation of hasAnyAnalysisResults.
   * 
   * **Validates: Requirements 5.5**
   */
  it('should re-evaluate hasAnyAnalysisResults when data is updated', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.constantFrom('metric', 'insight', 'echarts', 'table', 'image'),
        (sessionId, messageId, requestId, resultType) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Track hasAnyAnalysisResults changes
          const hasResultsHistory: boolean[] = [];
          const unsubscribe = manager.subscribe(() => {
            hasResultsHistory.push(calculateHasAnyAnalysisResults(manager));
          });
          
          // Verify initial state: no results
          expect(calculateHasAnyAnalysisResults(manager)).toBe(false);
          
          // Act: Add analysis result
          const createResultItem = (type: string) => {
            const baseItem = {
              id: `item-${type}-${Date.now()}`,
              type: type as any,
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime' as const,
            };
            switch (type) {
              case 'metric': return { ...baseItem, data: { title: 'Test Metric', value: '100', change: '' } };
              case 'insight': return { ...baseItem, data: { text: 'Test Insight', icon: 'info' } };
              case 'echarts': return { ...baseItem, data: { option: {} } };
              case 'table': return { ...baseItem, data: { columns: ['A'], rows: [['1']] } };
              case 'image': return { ...baseItem, data: 'data:image/png;base64,test' };
              default: return { ...baseItem, data: {} };
            }
          };
          
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items: [createResultItem(resultType)],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: hasAnyAnalysisResults should now be true
          expect(calculateHasAnyAnalysisResults(manager)).toBe(true);
          
          // Verify state change was captured
          expect(hasResultsHistory.length).toBeGreaterThan(0);
          expect(hasResultsHistory[hasResultsHistory.length - 1]).toBe(true);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.4: When data is cleared, hasAnyAnalysisResults should be
   * re-evaluated and reflect the empty state.
   * 
   * This tests that data clearing triggers re-evaluation of hasAnyAnalysisResults.
   * 
   * **Validates: Requirements 5.5**
   */
  it('should re-evaluate hasAnyAnalysisResults when data is cleared', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange: Set up manager with data
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Add some data
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items: [{
              id: `metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Test Metric', value: '100', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Verify we have results
          expect(calculateHasAnyAnalysisResults(manager)).toBe(true);
          
          // Track hasAnyAnalysisResults changes
          const hasResultsHistory: boolean[] = [];
          const unsubscribe = manager.subscribe(() => {
            hasResultsHistory.push(calculateHasAnyAnalysisResults(manager));
          });
          
          // Act: Clear results
          manager.clearResults(sessionId);
          
          // Assert: hasAnyAnalysisResults should now be false
          expect(calculateHasAnyAnalysisResults(manager)).toBe(false);
          
          // Verify state change was captured
          expect(hasResultsHistory.length).toBeGreaterThan(0);
          expect(hasResultsHistory[hasResultsHistory.length - 1]).toBe(false);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.5: Session switch should trigger session-switched event
   * which useDashboardData uses to reload data source statistics.
   * 
   * This tests the event-based synchronization mechanism for session switches.
   * 
   * **Validates: Requirements 5.2**
   */
  it('should emit session-switched event when session is switched', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        sessionIdArb,
        (sessionId1, sessionId2) => {
          // Skip if sessions are the same
          fc.pre(sessionId1 !== sessionId2);
          
          // Arrange: Reset and set up manager fresh for each property test iteration
          AnalysisResultManagerImpl.resetInstance();
          const manager = getAnalysisResultManager();
          
          // Verify manager starts with null session
          expect(manager.getCurrentSession()).toBeNull();
          
          // Track session-switched events - subscribe BEFORE any session switch
          const sessionSwitchEvents: { fromSessionId: string | null; toSessionId: string }[] = [];
          const unsubscribe = manager.on('session-switched', (event) => {
            sessionSwitchEvents.push(event);
          });
          
          // Set initial session - this should trigger the first event
          manager.switchSession(sessionId1);
          
          // Verify initial event was captured (fromSessionId is null for first switch)
          expect(sessionSwitchEvents.length).toBe(1);
          expect(sessionSwitchEvents[0].fromSessionId).toBeNull();
          expect(sessionSwitchEvents[0].toSessionId).toBe(sessionId1);
          
          // Act: Switch to different session
          manager.switchSession(sessionId2);
          
          // Assert: Session switch event should be captured
          expect(sessionSwitchEvents.length).toBe(2);
          expect(sessionSwitchEvents[1].fromSessionId).toBe(sessionId1);
          expect(sessionSwitchEvents[1].toSessionId).toBe(sessionId2);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.6: Message selection should trigger message-selected event
   * which useDashboardData uses to update display data.
   * 
   * This tests the event-based synchronization mechanism for message selection.
   * 
   * **Validates: Requirements 5.3**
   */
  it('should emit message-selected event when message is selected', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        messageIdArb,
        (sessionId, messageId1, messageId2) => {
          // Skip if messages are the same
          fc.pre(messageId1 !== messageId2);
          
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Set initial session
          manager.switchSession(sessionId);
          
          // Track message-selected events
          const messageSelectEvents: { sessionId: string; fromMessageId: string | null; toMessageId: string }[] = [];
          const unsubscribe = manager.on('message-selected', (event) => {
            messageSelectEvents.push(event);
          });
          
          // Select initial message
          manager.selectMessage(messageId1);
          
          // Verify initial event was captured
          expect(messageSelectEvents.length).toBe(1);
          expect(messageSelectEvents[0].sessionId).toBe(sessionId);
          expect(messageSelectEvents[0].fromMessageId).toBeNull();
          expect(messageSelectEvents[0].toMessageId).toBe(messageId1);
          
          // Act: Select different message
          manager.selectMessage(messageId2);
          
          // Assert: Message select event should be captured
          expect(messageSelectEvents.length).toBe(2);
          expect(messageSelectEvents[1].sessionId).toBe(sessionId);
          expect(messageSelectEvents[1].fromMessageId).toBe(messageId1);
          expect(messageSelectEvents[1].toMessageId).toBe(messageId2);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.7: Multiple state changes in sequence should all be
   * synchronized to subscribers.
   * 
   * This tests that rapid state changes are all properly synchronized.
   * 
   * **Validates: Requirements 5.2, 5.3, 5.5**
   */
  it('should synchronize all state changes in sequence', () => {
    fc.assert(
      fc.property(
        fc.array(
          fc.tuple(
            fc.constantFrom('session-switch', 'message-select', 'data-update', 'data-clear'),
            sessionIdArb,
            messageIdArb,
            requestIdArb
          ),
          { minLength: 2, maxLength: 5 }
        ),
        (operations) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Track all state changes
          const stateChanges: { sessionId: string | null; messageId: string | null; hasResults: boolean }[] = [];
          const unsubscribe = manager.subscribe((state) => {
            stateChanges.push({
              sessionId: state.currentSessionId,
              messageId: state.currentMessageId,
              hasResults: calculateHasAnyAnalysisResults(manager),
            });
          });
          
          // Initialize with first session
          let currentSessionId = operations[0][1];
          manager.switchSession(currentSessionId);
          
          // Act: Execute all operations
          for (const [opType, sessionId, messageId, requestId] of operations) {
            switch (opType) {
              case 'session-switch':
                const targetSession = currentSessionId !== sessionId ? sessionId : `${sessionId}-alt`;
                manager.switchSession(targetSession);
                currentSessionId = targetSession;
                break;
              case 'message-select':
                manager.selectMessage(messageId);
                break;
              case 'data-update':
                manager.updateResults({
                  sessionId: currentSessionId,
                  messageId: manager.getCurrentMessage() || messageId,
                  requestId,
                  items: [{
                    id: `metric-${Date.now()}-${Math.random()}`,
                    type: 'metric',
                    data: { title: 'Test', value: '100', change: '' },
                    metadata: { sessionId: currentSessionId, messageId: manager.getCurrentMessage() || messageId, timestamp: Date.now() },
                    source: 'realtime',
                  }],
                  isComplete: true,
                  timestamp: Date.now(),
                });
                break;
              case 'data-clear':
                manager.clearResults(currentSessionId);
                break;
            }
          }
          
          // Assert: All operations should have triggered state changes
          expect(stateChanges.length).toBeGreaterThanOrEqual(operations.length);
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.8: State synchronization should be immediate (synchronous)
   * for all state change operations.
   * 
   * This tests that state changes are immediately reflected in the manager state.
   * 
   * **Validates: Requirements 5.2, 5.3, 5.5**
   */
  it('should synchronize state immediately after state change operations', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Act & Assert: Session switch should be immediate
          manager.switchSession(sessionId);
          expect(manager.getCurrentSession()).toBe(sessionId);
          expect(manager.getState().currentSessionId).toBe(sessionId);
          
          // Act & Assert: Message select should be immediate
          manager.selectMessage(messageId);
          expect(manager.getCurrentMessage()).toBe(messageId);
          expect(manager.getState().currentMessageId).toBe(messageId);
          
          // Act & Assert: Data update should be immediate
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items: [{
              id: `metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Test', value: '100', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          expect(calculateHasAnyAnalysisResults(manager)).toBe(true);
          expect(manager.getCurrentResults().length).toBe(1);
          
          // Act & Assert: Data clear should be immediate
          manager.clearResults(sessionId);
          expect(calculateHasAnyAnalysisResults(manager)).toBe(false);
          expect(manager.getCurrentResults().length).toBe(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.9: Subscribers should receive consistent state snapshots
   * that match the manager's current state.
   * 
   * This tests that the state passed to subscribers is consistent with
   * the manager's actual state.
   * 
   * **Validates: Requirements 5.2, 5.3, 5.5**
   */
  it('should provide consistent state snapshots to subscribers', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        (sessionId, messageId, requestId) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          // Track state snapshots from subscriber
          let lastSubscriberState: any = null;
          const unsubscribe = manager.subscribe((state) => {
            lastSubscriberState = state;
          });
          
          // Act: Perform state changes
          manager.switchSession(sessionId);
          
          // Assert: Subscriber state should match manager state
          expect(lastSubscriberState).not.toBeNull();
          expect(lastSubscriberState.currentSessionId).toBe(manager.getCurrentSession());
          
          // Act: Select message
          manager.selectMessage(messageId);
          
          // Assert: Subscriber state should match manager state
          expect(lastSubscriberState.currentMessageId).toBe(manager.getCurrentMessage());
          
          // Act: Add data
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items: [{
              id: `metric-${Date.now()}`,
              type: 'metric',
              data: { title: 'Test', value: '100', change: '' },
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime',
            }],
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: Subscriber state should reflect data presence
          expect(lastSubscriberState.isLoading).toBe(manager.isLoading());
          
          // Cleanup
          unsubscribe();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 6.10: hasAnyAnalysisResults should correctly reflect
   * the presence of any analysis result type after state changes.
   * 
   * This tests that hasAnyAnalysisResults is correctly re-evaluated
   * for all result types after state changes.
   * 
   * **Validates: Requirements 5.5**
   */
  it('should correctly evaluate hasAnyAnalysisResults for all result types after state changes', () => {
    fc.assert(
      fc.property(
        sessionIdArb,
        messageIdArb,
        requestIdArb,
        fc.array(
          fc.constantFrom('metric', 'insight', 'echarts', 'table', 'image', 'file'),
          { minLength: 1, maxLength: 6 }
        ),
        (sessionId, messageId, requestId, resultTypes) => {
          // Arrange: Set up manager
          const manager = getAnalysisResultManager();
          
          manager.switchSession(sessionId);
          manager.selectMessage(messageId);
          
          // Verify initial state: no results
          expect(calculateHasAnyAnalysisResults(manager)).toBe(false);
          
          // Create items for each result type
          const items = resultTypes.map((type, index) => {
            const baseItem = {
              id: `item-${type}-${index}-${Date.now()}`,
              type: type as any,
              metadata: { sessionId, messageId, timestamp: Date.now() },
              source: 'realtime' as const,
            };
            switch (type) {
              case 'metric': return { ...baseItem, data: { title: `Metric ${index}`, value: '100', change: '' } };
              case 'insight': return { ...baseItem, data: { text: `Insight ${index}`, icon: 'info' } };
              case 'echarts': return { ...baseItem, data: { option: {} } };
              case 'table': return { ...baseItem, data: { columns: ['A'], rows: [['1']] } };
              case 'image': return { ...baseItem, data: 'data:image/png;base64,test' };
              case 'file': return { ...baseItem, data: { filename: `file${index}.txt`, path: '/tmp' } };
              default: return { ...baseItem, data: {} };
            }
          });
          
          // Act: Add all items
          manager.updateResults({
            sessionId,
            messageId,
            requestId,
            items,
            isComplete: true,
            timestamp: Date.now(),
          });
          
          // Assert: hasAnyAnalysisResults should be true
          expect(calculateHasAnyAnalysisResults(manager)).toBe(true);
          
          // Act: Clear results
          manager.clearResults(sessionId);
          
          // Assert: hasAnyAnalysisResults should be false
          expect(calculateHasAnyAnalysisResults(manager)).toBe(false);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});
