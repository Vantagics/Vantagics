/**
 * Bug Condition Exploration Test for AnalysisResultBridge
 *
 * **Validates: Requirements 1.1, 2.1**
 *
 * This test verifies that `initAnalysisResultBridge` registers a listener
 * for the `analysis-session-created` event and correctly calls
 * `manager.switchSession(threadId)` and `manager.setLoading(true)`.
 *
 * On UNFIXED code this test MUST FAIL — confirming the bug exists:
 * AnalysisResultBridge does not listen for `analysis-session-created`,
 * so `currentSessionId` remains null after the event fires.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as fc from 'fast-check';

// ── Mocks ────────────────────────────────────────────────────────────────────

// Capture every EventsOn registration so we can simulate events later.
const eventListeners: Record<string, ((...args: any[]) => void)[]> = {};

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn((eventName: string, callback: (...args: any[]) => void) => {
    if (!eventListeners[eventName]) {
      eventListeners[eventName] = [];
    }
    eventListeners[eventName].push(callback);
    // Return an unsubscribe function
    return () => {
      const idx = eventListeners[eventName]?.indexOf(callback);
      if (idx !== undefined && idx >= 0) {
        eventListeners[eventName].splice(idx, 1);
      }
    };
  }),
  EventsEmit: vi.fn(),
}));

// Spy manager returned by getAnalysisResultManager
const mockManager = {
  switchSession: vi.fn(),
  setLoading: vi.fn(),
  updateResults: vi.fn(),
  clearResults: vi.fn(),
  clearAll: vi.fn(),
  restoreResults: vi.fn(),
  setErrorWithInfo: vi.fn(),
  setError: vi.fn(),
  getError: vi.fn(),
  getErrorInfo: vi.fn(),
  getResults: vi.fn(),
  getResultsByType: vi.fn(),
  hasData: vi.fn(),
  getCurrentResults: vi.fn(),
  getCurrentResultsByType: vi.fn(),
  hasCurrentData: vi.fn(),
  getCurrentSession: vi.fn(() => null),
  selectMessage: vi.fn(),
  getCurrentMessage: vi.fn(() => null),
  subscribe: vi.fn(() => () => {}),
  on: vi.fn(() => () => {}),
  isLoading: vi.fn(() => false),
  getState: vi.fn(),
  notifyHistoricalEmptyResult: vi.fn(),
  getPendingRequestId: vi.fn(() => null),
};

vi.mock('../../managers/AnalysisResultManager', () => ({
  getAnalysisResultManager: vi.fn(() => mockManager),
}));

// Mock the logger so it doesn't try to call real system log
vi.mock('../systemLog', () => ({
  createLogger: vi.fn(() => ({
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  })),
}));

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Simulate a Wails event by invoking all registered callbacks for that name. */
function simulateEvent(eventName: string, ...args: any[]) {
  const listeners = eventListeners[eventName];
  if (listeners) {
    listeners.forEach((cb) => cb(...args));
  }
}

// ── Tests ────────────────────────────────────────────────────────────────────

describe('AnalysisResultBridge – bug condition exploration', () => {
  beforeEach(() => {
    // Clear captured listeners and mock call history
    for (const key of Object.keys(eventListeners)) {
      delete eventListeners[key];
    }
    vi.clearAllMocks();
  });

  afterEach(async () => {
    // Reset the bridge's internal `bridgeInitialized` flag so each test starts fresh
    const { resetBridge } = await import('../AnalysisResultBridge');
    resetBridge();
  });

  it('should register a listener for analysis-session-created and call switchSession', async () => {
    const { initAnalysisResultBridge } = await import('../AnalysisResultBridge');

    // Initialize the bridge
    initAnalysisResultBridge(
      () => null,
      () => null,
    );

    // Simulate the backend emitting analysis-session-created
    simulateEvent('analysis-session-created', {
      threadId: 'thread-123',
      dataSourceId: 'ds-456',
    });

    // On unfixed code this WILL FAIL because no listener is registered
    expect(mockManager.switchSession).toHaveBeenCalledWith('thread-123');
  });

  it('should call manager.setLoading(true) when analysis-session-created fires', async () => {
    const { initAnalysisResultBridge } = await import('../AnalysisResultBridge');

    initAnalysisResultBridge(
      () => null,
      () => null,
    );

    simulateEvent('analysis-session-created', {
      threadId: 'thread-789',
      dataSourceId: 'ds-012',
    });

    // On unfixed code this WILL FAIL because no listener is registered
    expect(mockManager.setLoading).toHaveBeenCalledWith(true);
  });

  it('should have analysis-session-created in registered event listeners', async () => {
    const { initAnalysisResultBridge } = await import('../AnalysisResultBridge');

    initAnalysisResultBridge(
      () => null,
      () => null,
    );

    // On unfixed code this WILL FAIL — no listener registered for this event
    expect(eventListeners['analysis-session-created']).toBeDefined();
    expect(eventListeners['analysis-session-created'].length).toBeGreaterThan(0);
  });

  // ── Property-Based Test ──────────────────────────────────────────────────

  it('property: switchSession is always called with the generated threadId for any valid payload', async () => {
    /**
     * **Validates: Requirements 1.1, 2.1**
     *
     * For ANY valid analysis-session-created payload, switchSession must be
     * called with the exact threadId from the payload.
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(
        fc.record({
          threadId: fc.uuid(),
          dataSourceId: fc.uuid(),
        }),
        (payload) => {
          // Reset state for each generated case
          resetBridge();
          for (const key of Object.keys(eventListeners)) {
            delete eventListeners[key];
          }
          vi.clearAllMocks();

          // Initialize bridge
          initAnalysisResultBridge(
            () => null,
            () => null,
          );

          // Simulate event with generated payload
          simulateEvent('analysis-session-created', payload);

          // switchSession MUST be called with the exact threadId
          // On unfixed code this WILL FAIL for every generated payload
          expect(mockManager.switchSession).toHaveBeenCalledWith(payload.threadId);
        },
      ),
      { numRuns: 50 },
    );
  });
});
