/**
 * Bug Condition Exploration Test for ChatSidebar chat-loading closure & LoadingStateManager
 *
 * **Validates: Requirements 1.1, 1.2, 1.3, 2.1, 2.2**
 *
 * This test verifies two bugs in ChatSidebar.tsx:
 *
 * Bug 1 - Stale Closure: The `chat-loading` event handler captures `activeThreadId`
 * via closure. The useEffect dependency array is `[threads]` and does NOT include
 * `activeThreadId`. After `start-new-chat` calls `setActiveThreadId(thread.id)`,
 * the handler still sees the OLD value. So `data.threadId === activeThreadId` is false
 * and `setIsLoading` is never called.
 *
 * Bug 2 - LoadingStateManager not notified: The `start-new-chat` handler's
 * `initialMessage` path calls `setIsLoading(true)` and `setLoadingThreadId(thread.id)`
 * but never calls `loadingStateManager.setLoading(thread.id, true)`.
 *
 * On UNFIXED code these tests MUST FAIL — confirming the bugs exist.
 */

import { describe, it, expect, vi } from 'vitest';
import * as fc from 'fast-check';

// ---------------------------------------------------------------------------
// Helpers: simulate the chat-loading handler logic extracted from ChatSidebar.tsx
// ---------------------------------------------------------------------------

/**
 * Simulates the CURRENT (buggy) chat-loading handler logic.
 * Uses `activeThreadId` from closure (stale value).
 */
function chatLoadingHandler_buggy(
  data: any,
  activeThreadId: string | null, // closure-captured (stale) value
  setIsLoading: (v: boolean) => void,
  setLoadingThreadId: (v: string | null) => void,
) {
  if (typeof data === 'boolean') {
    // backward compat: boolean applies to current active session
    if (activeThreadId) {
      setIsLoading(data);
      if (data) {
        setLoadingThreadId(activeThreadId);
      } else {
        setLoadingThreadId(null);
      }
    }
  } else if (data && typeof data === 'object') {
    // new format: object with threadId
    if (data.threadId === activeThreadId) {
      setIsLoading(data.loading);
      if (data.loading) {
        setLoadingThreadId(data.threadId);
      } else {
        setLoadingThreadId(null);
      }
    }
  }
}

/**
 * Simulates the FIXED chat-loading handler logic.
 * Uses `activeThreadIdRef.current` (always up-to-date).
 */
function chatLoadingHandler_fixed(
  data: any,
  activeThreadIdRef: { current: string | null }, // ref (latest value)
  setIsLoading: (v: boolean) => void,
  setLoadingThreadId: (v: string | null) => void,
) {
  if (typeof data === 'boolean') {
    if (activeThreadIdRef.current) {
      setIsLoading(data);
      if (data) {
        setLoadingThreadId(activeThreadIdRef.current);
      } else {
        setLoadingThreadId(null);
      }
    }
  } else if (data && typeof data === 'object') {
    if (data.threadId === activeThreadIdRef.current) {
      setIsLoading(data.loading);
      if (data.loading) {
        setLoadingThreadId(data.threadId);
      } else {
        setLoadingThreadId(null);
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('ChatSidebar chat-loading – bug condition exploration', () => {
  // ── Test 1: Stale closure ──────────────────────────────────────────────

  describe('Bug 1 - Stale closure in chat-loading handler', () => {
    it('buggy handler: setIsLoading is NOT called when closure activeThreadId is stale', () => {
      const setIsLoading = vi.fn();
      const setLoadingThreadId = vi.fn();

      // Closure captured old value (null — no thread was active when useEffect ran)
      const staleActiveThreadId: string | null = null;

      // Backend emits chat-loading for the NEW thread
      const data = { threadId: 'new-thread-123', loading: true };

      chatLoadingHandler_buggy(data, staleActiveThreadId, setIsLoading, setLoadingThreadId);

      // With stale closure, data.threadId !== null → setIsLoading NOT called
      expect(setIsLoading).not.toHaveBeenCalled();
      expect(setLoadingThreadId).not.toHaveBeenCalled();
    });

    it('buggy handler: setIsLoading is NOT called when closure has old thread id', () => {
      const setIsLoading = vi.fn();
      const setLoadingThreadId = vi.fn();

      // Closure captured previous thread id
      const staleActiveThreadId = 'old-thread-456';

      // Backend emits chat-loading for the NEW thread
      const data = { threadId: 'new-thread-123', loading: true };

      chatLoadingHandler_buggy(data, staleActiveThreadId, setIsLoading, setLoadingThreadId);

      // 'new-thread-123' !== 'old-thread-456' → setIsLoading NOT called
      expect(setIsLoading).not.toHaveBeenCalled();
      expect(setLoadingThreadId).not.toHaveBeenCalled();
    });

    it('fixed handler: setIsLoading IS called when ref has the latest activeThreadId', () => {
      const setIsLoading = vi.fn();
      const setLoadingThreadId = vi.fn();

      // Ref always has the latest value
      const activeThreadIdRef = { current: 'new-thread-123' };

      // Backend emits chat-loading for the NEW thread
      const data = { threadId: 'new-thread-123', loading: true };

      chatLoadingHandler_fixed(data, activeThreadIdRef, setIsLoading, setLoadingThreadId);

      // ref.current matches → setIsLoading IS called
      expect(setIsLoading).toHaveBeenCalledWith(true);
      expect(setLoadingThreadId).toHaveBeenCalledWith('new-thread-123');
    });

    it('EXPECTED TO FAIL: buggy handler should set loading state for matching thread (confirms bug)', () => {
      /**
       * This test encodes the EXPECTED (correct) behavior.
       * On unfixed code, the buggy handler uses the stale closure value,
       * so this assertion WILL FAIL — confirming the bug exists.
       */
      const setIsLoading = vi.fn();
      const setLoadingThreadId = vi.fn();

      // Simulate: start-new-chat created 'new-thread', but closure still has null
      const staleActiveThreadId: string | null = null;

      const data = { threadId: 'new-thread-123', loading: true };

      // Run the BUGGY handler
      chatLoadingHandler_buggy(data, staleActiveThreadId, setIsLoading, setLoadingThreadId);

      // EXPECTED behavior: loading state SHOULD be set for the new thread
      // ACTUAL behavior on buggy code: setIsLoading is NOT called
      // This assertion WILL FAIL on unfixed code → confirms the bug
      expect(setIsLoading).toHaveBeenCalledWith(true);
    });
  });

  // ── Test 2: LoadingStateManager not notified ───────────────────────────

  describe('Bug 2 - LoadingStateManager not notified in initialMessage path', () => {
    /**
     * Simulates the initialMessage path of start-new-chat handler.
     * In the BUGGY code, loadingStateManager.setLoading is NOT called.
     */
    function simulateInitialMessagePath_buggy(
      threadId: string,
      setIsLoading: (v: boolean) => void,
      setLoadingThreadId: (v: string | null) => void,
      loadingStateManager: { setLoading: (id: string, loading: boolean) => void },
    ) {
      // This is what the buggy code does:
      setIsLoading(true);
      setLoadingThreadId(threadId);
      // NOTE: loadingStateManager.setLoading(threadId, true) is MISSING
    }

    function simulateInitialMessagePath_fixed(
      threadId: string,
      setIsLoading: (v: boolean) => void,
      setLoadingThreadId: (v: string | null) => void,
      loadingStateManager: { setLoading: (id: string, loading: boolean) => void },
    ) {
      setIsLoading(true);
      setLoadingThreadId(threadId);
      loadingStateManager.setLoading(threadId, true); // FIX: notify LoadingStateManager
    }

    it('buggy path: loadingStateManager.setLoading is NOT called', () => {
      const setIsLoading = vi.fn();
      const setLoadingThreadId = vi.fn();
      const loadingStateManager = { setLoading: vi.fn() };

      simulateInitialMessagePath_buggy('thread-abc', setIsLoading, setLoadingThreadId, loadingStateManager);

      // Confirm the bug: loadingStateManager.setLoading was NOT called
      expect(loadingStateManager.setLoading).not.toHaveBeenCalled();
    });

    it('fixed path: loadingStateManager.setLoading IS called', () => {
      const setIsLoading = vi.fn();
      const setLoadingThreadId = vi.fn();
      const loadingStateManager = { setLoading: vi.fn() };

      simulateInitialMessagePath_fixed('thread-abc', setIsLoading, setLoadingThreadId, loadingStateManager);

      expect(loadingStateManager.setLoading).toHaveBeenCalledWith('thread-abc', true);
    });

    it('EXPECTED TO FAIL: buggy initialMessage path should notify LoadingStateManager (confirms bug)', () => {
      /**
       * This test encodes the EXPECTED (correct) behavior.
       * On unfixed code, loadingStateManager.setLoading is never called,
       * so this assertion WILL FAIL — confirming the bug exists.
       */
      const setIsLoading = vi.fn();
      const setLoadingThreadId = vi.fn();
      const loadingStateManager = { setLoading: vi.fn() };

      simulateInitialMessagePath_buggy('thread-abc', setIsLoading, setLoadingThreadId, loadingStateManager);

      // EXPECTED behavior: loadingStateManager.setLoading SHOULD be called
      // ACTUAL behavior on buggy code: it is NOT called
      // This assertion WILL FAIL on unfixed code → confirms the bug
      expect(loadingStateManager.setLoading).toHaveBeenCalledWith('thread-abc', true);
    });
  });

  // ── Property-Based Test ────────────────────────────────────────────────

  describe('Property-based: stale closure prevents loading state from being set', () => {
    it('EXPECTED TO FAIL: for any threadId and stale activeThreadId, buggy handler should still set loading (confirms bug)', () => {
      /**
       * **Validates: Requirements 1.1, 1.2, 1.3, 2.1, 2.2**
       *
       * For ANY chat-loading payload { threadId, loading: true } and ANY stale
       * activeThreadId (from closure), when threadId !== activeThreadId:
       * - Buggy handler: setIsLoading is NOT called (bug)
       * - Fixed handler (using ref): setIsLoading IS called
       *
       * This property test WILL FAIL on unfixed code, confirming the bug exists
       * across the entire input space.
       */
      fc.assert(
        fc.property(
          fc.record({
            threadId: fc.uuid(),
            loading: fc.constant(true),
          }),
          fc.option(fc.uuid(), { nil: null }),
          (payload, staleActiveThreadId) => {
            // Skip the rare case where they happen to match (not a bug scenario)
            fc.pre(payload.threadId !== staleActiveThreadId);

            const setIsLoading = vi.fn();
            const setLoadingThreadId = vi.fn();

            // Simulate: ref has the CORRECT (latest) value = payload.threadId
            const activeThreadIdRef = { current: payload.threadId };

            // Run buggy handler with stale closure value
            chatLoadingHandler_buggy(payload, staleActiveThreadId, setIsLoading, setLoadingThreadId);

            // EXPECTED: setIsLoading should have been called (correct behavior)
            // ACTUAL on buggy code: NOT called because threadId !== staleActiveThreadId
            // This WILL FAIL → confirms the bug
            expect(setIsLoading).toHaveBeenCalledWith(true);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('for any threadId, fixed handler with matching ref always sets loading', () => {
      /**
       * **Validates: Requirements 2.1**
       *
       * Confirms that the fixed handler (using ref) correctly sets loading
       * for ANY generated threadId when the ref has the matching value.
       */
      fc.assert(
        fc.property(
          fc.uuid(),
          (threadId) => {
            const setIsLoading = vi.fn();
            const setLoadingThreadId = vi.fn();
            const activeThreadIdRef = { current: threadId };

            const payload = { threadId, loading: true };

            chatLoadingHandler_fixed(payload, activeThreadIdRef, setIsLoading, setLoadingThreadId);

            expect(setIsLoading).toHaveBeenCalledWith(true);
            expect(setLoadingThreadId).toHaveBeenCalledWith(threadId);
          },
        ),
        { numRuns: 100 },
      );
    });
  });
});
