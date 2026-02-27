/**
 * Preservation Property Tests for AnalysisResultBridge
 *
 * **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5**
 *
 * These tests capture the baseline behavior of all NON-analysis-session-created
 * event paths in AnalysisResultBridge. They MUST PASS on the current unfixed code,
 * confirming the behavior that must be preserved after the fix.
 *
 * Observation-first methodology: we observe what the unfixed code does for each
 * event, then encode those observations as property-based tests.
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
  restoreResults: vi.fn(() => ({ totalItems: 0, validItems: 0, invalidItems: 0, itemsByType: {}, errors: [] })),
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

vi.mock('../systemLog', () => ({
  createLogger: vi.fn(() => ({
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  })),
}));

// ── Helpers ──────────────────────────────────────────────────────────────────

function simulateEvent(eventName: string, ...args: any[]) {
  const listeners = eventListeners[eventName];
  if (listeners) {
    listeners.forEach((cb) => cb(...args));
  }
}


// ── fast-check Arbitraries ───────────────────────────────────────────────────

/** Arbitrary for AnalysisResultItem */
const arbResultItem = fc.record({
  id: fc.string({ minLength: 1, maxLength: 20 }),
  type: fc.constantFrom('echarts', 'image', 'table', 'csv', 'metric', 'insight', 'file' as const),
  data: fc.oneof(fc.string(), fc.constant({ key: 'value' })),
  metadata: fc.record({
    sessionId: fc.string({ minLength: 1, maxLength: 30 }),
    messageId: fc.string({ minLength: 1, maxLength: 30 }),
    timestamp: fc.nat(),
  }),
  source: fc.constantFrom('realtime', 'completed', 'cached', 'restored' as const),
});

/** Arbitrary for AnalysisResultBatch (analysis-result-update payload) */
const arbUpdatePayload = fc.record({
  sessionId: fc.string({ minLength: 1, maxLength: 30 }),
  messageId: fc.string({ minLength: 1, maxLength: 30 }),
  requestId: fc.string({ minLength: 1, maxLength: 30 }),
  items: fc.array(arbResultItem, { minLength: 0, maxLength: 5 }),
  isComplete: fc.boolean(),
  timestamp: fc.nat(),
});

/** Arbitrary for analysis-result-clear payload */
const arbClearPayload = fc.record({
  sessionId: fc.string({ minLength: 1, maxLength: 30 }),
  messageId: fc.option(fc.string({ minLength: 1, maxLength: 30 }), { nil: undefined }),
});

/** Arbitrary for analysis-result-loading payload */
const arbLoadingPayload = fc.record({
  sessionId: fc.string({ minLength: 1, maxLength: 30 }),
  loading: fc.boolean(),
  requestId: fc.option(fc.string({ minLength: 1, maxLength: 30 }), { nil: undefined }),
});

/** Arbitrary for analysis-cancelled payload */
const arbCancelledPayload = fc.record({
  threadId: fc.string({ minLength: 1, maxLength: 30 }),
  message: fc.option(fc.string({ minLength: 0, maxLength: 50 }), { nil: undefined }),
});

/** Arbitrary for analysis-result-restore payload */
const arbRestorePayload = fc.record({
  sessionId: fc.string({ minLength: 1, maxLength: 30 }),
  messageId: fc.string({ minLength: 1, maxLength: 30 }),
  items: fc.array(arbResultItem, { minLength: 0, maxLength: 5 }),
});

/** Arbitrary for analysis-result-error / analysis-error payload */
const arbErrorPayload = fc.record({
  sessionId: fc.string({ minLength: 1, maxLength: 30 }),
  threadId: fc.option(fc.string({ minLength: 1, maxLength: 30 }), { nil: undefined }),
  requestId: fc.option(fc.string({ minLength: 1, maxLength: 30 }), { nil: undefined }),
  code: fc.option(fc.constantFrom(
    'ANALYSIS_ERROR', 'ANALYSIS_TIMEOUT', 'ANALYSIS_CANCELLED',
    'PYTHON_EXECUTION', 'DATA_NOT_FOUND', 'CONNECTION_FAILED',
  ), { nil: undefined }),
  error: fc.option(fc.string({ minLength: 1, maxLength: 50 }), { nil: undefined }),
  message: fc.option(fc.string({ minLength: 1, maxLength: 50 }), { nil: undefined }),
  details: fc.option(fc.string({ minLength: 1, maxLength: 50 }), { nil: undefined }),
  recoverySuggestions: fc.option(fc.array(fc.string({ minLength: 1, maxLength: 50 }), { minLength: 0, maxLength: 3 }), { nil: undefined }),
  timestamp: fc.option(fc.nat(), { nil: undefined }),
});

// ── Tests ────────────────────────────────────────────────────────────────────

describe('AnalysisResultBridge – preservation properties', () => {
  beforeEach(() => {
    for (const key of Object.keys(eventListeners)) {
      delete eventListeners[key];
    }
    vi.clearAllMocks();
  });

  afterEach(async () => {
    const { resetBridge } = await import('../AnalysisResultBridge');
    resetBridge();
  });

  // ── Property: analysis-result-update → updateResults ───────────────────

  it('property: for all analysis-result-update payloads, updateResults is called with the exact payload', async () => {
    /**
     * **Validates: Requirements 3.1, 3.2**
     *
     * For ANY analysis-result-update payload, the bridge must call
     * manager.updateResults with the exact same payload object.
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(arbUpdatePayload, (payload) => {
        resetBridge();
        for (const key of Object.keys(eventListeners)) {
          delete eventListeners[key];
        }
        vi.clearAllMocks();

        initAnalysisResultBridge(() => null, () => null);
        simulateEvent('analysis-result-update', payload);

        expect(mockManager.updateResults).toHaveBeenCalledTimes(1);
        expect(mockManager.updateResults).toHaveBeenCalledWith(payload);
      }),
      { numRuns: 50 },
    );
  });

  // ── Property: analysis-result-clear → clearResults ─────────────────────

  it('property: for all analysis-result-clear payloads, clearResults is called with exact sessionId and messageId', async () => {
    /**
     * **Validates: Requirements 3.1, 3.2**
     *
     * For ANY analysis-result-clear payload, the bridge must call
     * manager.clearResults with the exact sessionId and messageId.
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(arbClearPayload, (payload) => {
        resetBridge();
        for (const key of Object.keys(eventListeners)) {
          delete eventListeners[key];
        }
        vi.clearAllMocks();

        initAnalysisResultBridge(() => null, () => null);
        simulateEvent('analysis-result-clear', payload);

        expect(mockManager.clearResults).toHaveBeenCalledTimes(1);
        expect(mockManager.clearResults).toHaveBeenCalledWith(payload.sessionId, payload.messageId);
      }),
      { numRuns: 50 },
    );
  });

  // ── Property: analysis-cancelled → setLoading(false) ───────────────────

  it('property: for all analysis-cancelled payloads, setLoading(false) is called', async () => {
    /**
     * **Validates: Requirements 3.4**
     *
     * For ANY analysis-cancelled payload, the bridge must call
     * manager.setLoading(false).
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(arbCancelledPayload, (payload) => {
        resetBridge();
        for (const key of Object.keys(eventListeners)) {
          delete eventListeners[key];
        }
        vi.clearAllMocks();

        initAnalysisResultBridge(() => null, () => null);
        simulateEvent('analysis-cancelled', payload);

        expect(mockManager.setLoading).toHaveBeenCalledTimes(1);
        expect(mockManager.setLoading).toHaveBeenCalledWith(false);
      }),
      { numRuns: 50 },
    );
  });

  // ── Property: analysis-result-loading → setLoading ─────────────────────

  it('property: for all analysis-result-loading payloads, setLoading is called with exact loading and requestId', async () => {
    /**
     * **Validates: Requirements 3.1, 3.2**
     *
     * For ANY analysis-result-loading payload, the bridge must call
     * manager.setLoading with the exact loading boolean and requestId.
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(arbLoadingPayload, (payload) => {
        resetBridge();
        for (const key of Object.keys(eventListeners)) {
          delete eventListeners[key];
        }
        vi.clearAllMocks();

        initAnalysisResultBridge(() => null, () => null);
        simulateEvent('analysis-result-loading', payload);

        expect(mockManager.setLoading).toHaveBeenCalledTimes(1);
        expect(mockManager.setLoading).toHaveBeenCalledWith(payload.loading, payload.requestId);
      }),
      { numRuns: 50 },
    );
  });

  // ── Property: analysis-result-restore → restoreResults ─────────────────

  it('property: for all analysis-result-restore payloads, restoreResults is called with exact sessionId, messageId, items', async () => {
    /**
     * **Validates: Requirements 3.3**
     *
     * For ANY analysis-result-restore payload, the bridge must call
     * manager.restoreResults with the exact sessionId, messageId, and items.
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(arbRestorePayload, (payload) => {
        resetBridge();
        for (const key of Object.keys(eventListeners)) {
          delete eventListeners[key];
        }
        vi.clearAllMocks();

        initAnalysisResultBridge(() => null, () => null);
        simulateEvent('analysis-result-restore', payload);

        expect(mockManager.restoreResults).toHaveBeenCalledTimes(1);
        expect(mockManager.restoreResults).toHaveBeenCalledWith(
          payload.sessionId,
          payload.messageId,
          payload.items,
        );
      }),
      { numRuns: 50 },
    );
  });

  // ── Property: analysis-result-error → setErrorWithInfo ─────────────────

  it('property: for all analysis-result-error payloads, setErrorWithInfo is called with correct errorInfo', async () => {
    /**
     * **Validates: Requirements 3.5**
     *
     * For ANY analysis-result-error payload, the bridge must call
     * manager.setErrorWithInfo with an EnhancedErrorInfo object that
     * preserves the code, message, details, recoverySuggestions, and timestamp.
     *
     * Note: The bridge uses `payload.timestamp || Date.now()`, so falsy
     * timestamps (0, undefined) are replaced with Date.now().
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(arbErrorPayload, (payload) => {
        resetBridge();
        for (const key of Object.keys(eventListeners)) {
          delete eventListeners[key];
        }
        vi.clearAllMocks();

        initAnalysisResultBridge(() => null, () => null);
        simulateEvent('analysis-result-error', payload);

        expect(mockManager.setErrorWithInfo).toHaveBeenCalledTimes(1);

        const calledWith = mockManager.setErrorWithInfo.mock.calls[0][0];
        expect(calledWith.code).toBe(payload.code || 'ANALYSIS_ERROR');
        expect(calledWith.message).toBe(payload.error || payload.message || '发生未知错误');
        expect(calledWith.details).toBe(payload.details);
        expect(calledWith.recoverySuggestions).toEqual(payload.recoverySuggestions || []);
        // Bridge uses `payload.timestamp || Date.now()` — falsy values (0, undefined) become Date.now()
        if (payload.timestamp) {
          expect(calledWith.timestamp).toBe(payload.timestamp);
        } else {
          expect(typeof calledWith.timestamp).toBe('number');
          expect(calledWith.timestamp).toBeGreaterThan(0);
        }
      }),
      { numRuns: 50 },
    );
  });

  // ── Property: analysis-error (legacy) → setErrorWithInfo ───────────────

  it('property: for all analysis-error payloads, setErrorWithInfo is called with correct errorInfo', async () => {
    /**
     * **Validates: Requirements 3.5**
     *
     * For ANY analysis-error payload (legacy event name), the bridge must call
     * manager.setErrorWithInfo with the same EnhancedErrorInfo structure.
     */
    const { initAnalysisResultBridge, resetBridge } = await import('../AnalysisResultBridge');

    fc.assert(
      fc.property(arbErrorPayload, (payload) => {
        resetBridge();
        for (const key of Object.keys(eventListeners)) {
          delete eventListeners[key];
        }
        vi.clearAllMocks();

        initAnalysisResultBridge(() => null, () => null);
        simulateEvent('analysis-error', payload);

        expect(mockManager.setErrorWithInfo).toHaveBeenCalledTimes(1);

        const calledWith = mockManager.setErrorWithInfo.mock.calls[0][0];
        expect(calledWith.code).toBe(payload.code || 'ANALYSIS_ERROR');
        expect(calledWith.message).toBe(payload.error || payload.message || '发生未知错误');
        expect(calledWith.details).toBe(payload.details);
        expect(calledWith.recoverySuggestions).toEqual(payload.recoverySuggestions || []);
        // Bridge uses `payload.timestamp || Date.now()` — falsy values (0, undefined) become Date.now()
        if (payload.timestamp) {
          expect(calledWith.timestamp).toBe(payload.timestamp);
        } else {
          expect(typeof calledWith.timestamp).toBe('number');
          expect(calledWith.timestamp).toBeGreaterThan(0);
        }
      }),
      { numRuns: 50 },
    );
  });
});
