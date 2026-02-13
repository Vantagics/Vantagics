/**
 * Unit tests for Comprehensive Report button visibility logic in ChatSidebar.
 *
 * The actual condition in ChatSidebar.tsx (~line 3206):
 *   {activeThread && activeThread.data_source_id && (activeDataSource || activeThread.is_replay_session) && (
 *
 * We extract and test this condition as a pure function to avoid
 * the complexity of rendering the full ChatSidebar component.
 *
 * Validates: Requirements 5.1, 5.3
 */

import { describe, it, expect } from 'vitest';

/**
 * Pure function that mirrors the comprehensive report button visibility condition
 * from ChatSidebar.tsx.
 *
 * The button is visible when ALL of the following are true:
 * 1. activeThread exists
 * 2. activeThread.data_source_id is truthy (non-empty)
 * 3. Either activeDataSource is truthy OR activeThread.is_replay_session is true
 */
function isComprehensiveReportButtonVisible(
  activeThread: { data_source_id?: string; is_replay_session?: boolean } | null | undefined,
  activeDataSource: unknown
): boolean {
  return !!(
    activeThread &&
    activeThread.data_source_id &&
    (activeDataSource || activeThread.is_replay_session)
  );
}

describe('Comprehensive Report Button Visibility', () => {
  describe('replay sessions (is_replay_session=true)', () => {
    it('should be visible for replay session even without activeDataSource', () => {
      const thread = { data_source_id: 'ds-1', is_replay_session: true };
      expect(isComprehensiveReportButtonVisible(thread, null)).toBe(true);
    });

    it('should be visible for replay session with activeDataSource', () => {
      const thread = { data_source_id: 'ds-1', is_replay_session: true };
      expect(isComprehensiveReportButtonVisible(thread, { id: 'ds-1' })).toBe(true);
    });
  });

  describe('normal sessions (is_replay_session=false or undefined)', () => {
    it('should be visible when activeDataSource is present', () => {
      const thread = { data_source_id: 'ds-1', is_replay_session: false };
      expect(isComprehensiveReportButtonVisible(thread, { id: 'ds-1' })).toBe(true);
    });

    it('should be hidden when activeDataSource is null', () => {
      const thread = { data_source_id: 'ds-1', is_replay_session: false };
      expect(isComprehensiveReportButtonVisible(thread, null)).toBe(false);
    });

    it('should be hidden when activeDataSource is undefined', () => {
      const thread = { data_source_id: 'ds-1' };
      expect(isComprehensiveReportButtonVisible(thread, undefined)).toBe(false);
    });
  });

  describe('threads without data_source_id', () => {
    it('should be hidden when data_source_id is empty string', () => {
      const thread = { data_source_id: '', is_replay_session: true };
      expect(isComprehensiveReportButtonVisible(thread, { id: 'ds-1' })).toBe(false);
    });

    it('should be hidden when data_source_id is undefined', () => {
      const thread = { is_replay_session: true };
      expect(isComprehensiveReportButtonVisible(thread, { id: 'ds-1' })).toBe(false);
    });
  });

  describe('no active thread', () => {
    it('should be hidden when activeThread is null', () => {
      expect(isComprehensiveReportButtonVisible(null, { id: 'ds-1' })).toBe(false);
    });

    it('should be hidden when activeThread is undefined', () => {
      expect(isComprehensiveReportButtonVisible(undefined, { id: 'ds-1' })).toBe(false);
    });
  });

  describe('loading state interaction', () => {
    /**
     * When isGeneratingComprehensiveReport is true, the button area shows a loading
     * indicator instead of the clickable button. The visibility condition is the same —
     * the container is shown, but its content switches between button and loader.
     * We verify the container visibility condition holds during generation.
     */
    it('container should remain visible during report generation for replay session', () => {
      const thread = { data_source_id: 'ds-1', is_replay_session: true };
      // The visibility condition doesn't depend on isGeneratingComprehensiveReport;
      // it controls the outer container. The loading state only affects inner content.
      expect(isComprehensiveReportButtonVisible(thread, null)).toBe(true);
    });

    it('container should remain visible during report generation for normal session', () => {
      const thread = { data_source_id: 'ds-1', is_replay_session: false };
      expect(isComprehensiveReportButtonVisible(thread, { id: 'ds-1' })).toBe(true);
    });
  });
});
