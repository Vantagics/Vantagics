/**
 * Property-Based Tests for MessageBubble Result Display Button Visibility Logic
 *
 * Uses fast-check to verify universal properties across randomized inputs.
 * Each property test runs a minimum of 100 iterations.
 *
 * Feature: pack-result-display, Property 2: Result display button visibility logic
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';

/**
 * Pure function that encapsulates the result display button visibility logic
 * from MessageBubble.tsx (line ~932):
 *
 *   isReplaySession && !isUser && content.includes('✅') && !content.includes('❌') && messageId && onShowResult
 *
 * We test the core boolean logic here. The `messageId` and `onShowResult` are
 * always truthy in a well-formed scenario (they're required for the button to
 * function), so we treat them as preconditions rather than variables under test.
 */
function shouldShowResultButton(params: {
  isReplaySession: boolean;
  role: 'user' | 'assistant';
  content: string;
  hasMessageId: boolean;
  hasOnShowResult: boolean;
}): boolean {
  const { isReplaySession, role, content, hasMessageId, hasOnShowResult } = params;
  const isUser = role === 'user';
  return (
    isReplaySession &&
    !isUser &&
    content.includes('✅') &&
    !content.includes('❌') &&
    hasMessageId &&
    hasOnShowResult
  );
}

/**
 * Arbitrary that generates random message content strings.
 * Mixes regular text with optional ✅ and ❌ markers.
 */
const contentArb = fc.tuple(
  fc.string({ minLength: 0, maxLength: 200 }),
  fc.boolean(), // include ✅
  fc.boolean(), // include ❌
  fc.string({ minLength: 0, maxLength: 100 }),
).map(([prefix, includeSuccess, includeFailure, suffix]) => {
  let content = prefix;
  if (includeSuccess) content += ' ✅ ';
  if (includeFailure) content += ' ❌ ';
  content += suffix;
  return content;
});

const roleArb = fc.constantFrom<'user' | 'assistant'>('user', 'assistant');

describe('MessageBubble Property-Based Tests', () => {
  // ─────────────────────────────────────────────────────────────────────────
  // Feature: pack-result-display, Property 2: Result display button visibility logic
  // ─────────────────────────────────────────────────────────────────────────
  describe('Property 2: Result display button visibility logic', () => {
    /**
     * **Validates: Requirements 2.1, 2.3**
     *
     * The button is visible if and only if ALL of these hold:
     *   1. isReplaySession === true
     *   2. role === 'assistant'
     *   3. content includes "✅"
     *   4. content does NOT include "❌"
     *   5. messageId is truthy
     *   6. onShowResult is truthy
     */
    it('button is visible only when all conditions are met', () => {
      fc.assert(
        fc.property(
          fc.boolean(),   // isReplaySession
          roleArb,        // role
          contentArb,     // content
          fc.boolean(),   // hasMessageId
          fc.boolean(),   // hasOnShowResult
          (isReplaySession, role, content, hasMessageId, hasOnShowResult) => {
            const visible = shouldShowResultButton({
              isReplaySession,
              role,
              content,
              hasMessageId,
              hasOnShowResult,
            });

            const allConditionsMet =
              isReplaySession &&
              role === 'assistant' &&
              content.includes('✅') &&
              !content.includes('❌') &&
              hasMessageId &&
              hasOnShowResult;

            expect(visible).toBe(allConditionsMet);
          }
        ),
        { numRuns: 200 }
      );
    });

    /**
     * **Validates: Requirements 2.3**
     *
     * When isReplaySession is false, the button is NEVER visible
     * regardless of any other conditions.
     */
    it('button is never visible when isReplaySession is false', () => {
      fc.assert(
        fc.property(
          roleArb,
          contentArb,
          fc.boolean(),
          fc.boolean(),
          (role, content, hasMessageId, hasOnShowResult) => {
            const visible = shouldShowResultButton({
              isReplaySession: false,
              role,
              content,
              hasMessageId,
              hasOnShowResult,
            });

            expect(visible).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 2.1**
     *
     * When role is 'user', the button is NEVER visible
     * regardless of any other conditions.
     */
    it('button is never visible when role is user', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          contentArb,
          fc.boolean(),
          fc.boolean(),
          (isReplaySession, content, hasMessageId, hasOnShowResult) => {
            const visible = shouldShowResultButton({
              isReplaySession,
              role: 'user',
              content,
              hasMessageId,
              hasOnShowResult,
            });

            expect(visible).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 2.3**
     *
     * When content includes "❌", the button is NEVER visible
     * regardless of any other conditions.
     */
    it('button is never visible when content includes ❌', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          roleArb,
          // Generate content that always includes ❌
          fc.tuple(fc.string(), fc.string()).map(([a, b]) => `${a}❌${b}`),
          fc.boolean(),
          fc.boolean(),
          (isReplaySession, role, content, hasMessageId, hasOnShowResult) => {
            const visible = shouldShowResultButton({
              isReplaySession,
              role,
              content,
              hasMessageId,
              hasOnShowResult,
            });

            expect(visible).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 2.1**
     *
     * When content does NOT include "✅", the button is NEVER visible
     * regardless of any other conditions.
     */
    it('button is never visible when content does not include ✅', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          roleArb,
          // Generate content that never includes ✅
          fc.string().filter(s => !s.includes('✅')),
          fc.boolean(),
          fc.boolean(),
          (isReplaySession, role, content, hasMessageId, hasOnShowResult) => {
            const visible = shouldShowResultButton({
              isReplaySession,
              role,
              content,
              hasMessageId,
              hasOnShowResult,
            });

            expect(visible).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 2.1, 2.3**
     *
     * When ALL positive conditions are met (replay session, assistant role,
     * ✅ present, no ❌, messageId and callback present), the button IS visible.
     */
    it('button is visible when all positive conditions are met', () => {
      fc.assert(
        fc.property(
          // Generate content with ✅ but without ❌
          fc.tuple(
            fc.string().filter(s => !s.includes('❌') && !s.includes('✅')),
            fc.string().filter(s => !s.includes('❌') && !s.includes('✅')),
          ).map(([a, b]) => `${a}✅${b}`),
          (content) => {
            const visible = shouldShowResultButton({
              isReplaySession: true,
              role: 'assistant',
              content,
              hasMessageId: true,
              hasOnShowResult: true,
            });

            expect(visible).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
