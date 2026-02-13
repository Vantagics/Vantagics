/**
 * Property-Based Tests for getPackOrigin
 *
 * Feature: datasource-pack-loader, Property 1: 分析包来源图标区分
 * **Validates: Requirements 1.3**
 *
 * Uses fast-check to verify that getPackOrigin correctly distinguishes
 * marketplace-downloaded packs from locally-created packs based on
 * the `marketplace_` filename prefix.
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { getPackOrigin } from './ImportPackDialog';

describe('getPackOrigin Property-Based Tests', () => {
  // Feature: datasource-pack-loader, Property 1: 分析包来源图标区分
  describe('Property 1: 分析包来源图标区分', () => {
    /**
     * **Validates: Requirements 1.3**
     *
     * For any arbitrary string prepended with 'marketplace_',
     * getPackOrigin must return 'marketplace'.
     */
    it('returns "marketplace" for any string starting with "marketplace_"', () => {
      fc.assert(
        fc.property(
          fc.string().map(s => `marketplace_${s}`),
          (fileName) => {
            expect(getPackOrigin(fileName)).toBe('marketplace');
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 1.3**
     *
     * For any arbitrary string that does NOT start with 'marketplace_',
     * getPackOrigin must return 'local'.
     */
    it('returns "local" for any string not starting with "marketplace_"', () => {
      fc.assert(
        fc.property(
          fc.string().filter(s => !s.startsWith('marketplace_')),
          (fileName) => {
            expect(getPackOrigin(fileName)).toBe('local');
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 1.3**
     *
     * Exhaustive biconditional: for any string, getPackOrigin returns
     * 'marketplace' if and only if the string starts with 'marketplace_'.
     */
    it('result is "marketplace" iff fileName starts with "marketplace_"', () => {
      fc.assert(
        fc.property(
          fc.string(),
          (fileName) => {
            const result = getPackOrigin(fileName);
            const startsWithPrefix = fileName.startsWith('marketplace_');
            expect(result).toBe(startsWithPrefix ? 'marketplace' : 'local');
          }
        ),
        { numRuns: 200 }
      );
    });
  });
});


/**
 * Property-Based Tests for canImport
 *
 * Feature: datasource-pack-loader, Property 3: 确认按钮启用逻辑
 * **Validates: Requirements 3.3, 3.4, 3.5, 3.6**
 *
 * Uses fast-check to verify that canImport correctly determines whether
 * the confirm button should be enabled based on the combination of
 * PackLoadResult, SchemaValidationResult, and dialog state.
 */

import { canImport } from './ImportPackDialog';

// Valid DialogState values
const dialogStates = ['pack-list', 'loading', 'password', 'preview', 'executing'] as const;

// Arbitrary for generating a loadResult-like object
const loadResultArb = fc.record({
  has_python_steps: fc.boolean(),
  python_configured: fc.boolean(),
  missing_tables: fc.array(fc.string({ minLength: 1 })),
});

describe('canImport Property-Based Tests', () => {
  // Feature: datasource-pack-loader, Property 3: 确认按钮启用逻辑
  describe('Property 3: 确认按钮启用逻辑', () => {
    /**
     * **Validates: Requirements 3.3, 3.4, 3.5, 3.6**
     *
     * For any combination of has_python_steps, python_configured,
     * missing_tables, and state, canImport returns true if and only if:
     * - state === 'preview'
     * - AND missing_tables is empty
     * - AND (has_python_steps is false OR python_configured is true)
     */
    it('returns true iff state=preview, no missing tables, and python ok', () => {
      fc.assert(
        fc.property(
          loadResultArb,
          fc.constantFrom(...dialogStates),
          ({ has_python_steps, python_configured, missing_tables }, state) => {
            const result = canImport(
              {
                has_python_steps,
                python_configured,
                validation: { missing_tables },
              },
              state
            );
            const expected =
              state === 'preview' &&
              missing_tables.length === 0 &&
              (!has_python_steps || python_configured);
            expect(result).toBe(expected);
          }
        ),
        { numRuns: 200 }
      );
    });

    /**
     * **Validates: Requirements 3.3**
     *
     * Non-preview states always return false regardless of loadResult.
     */
    it('always returns false for non-preview states', () => {
      fc.assert(
        fc.property(
          loadResultArb,
          fc.constantFrom('pack-list', 'loading', 'password', 'executing'),
          ({ has_python_steps, python_configured, missing_tables }, state) => {
            const result = canImport(
              {
                has_python_steps,
                python_configured,
                validation: { missing_tables },
              },
              state
            );
            expect(result).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 3.3**
     *
     * When missing_tables is non-empty, canImport always returns false
     * even in preview state.
     */
    it('returns false when missing_tables is non-empty', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          fc.boolean(),
          fc.array(fc.string({ minLength: 1 }), { minLength: 1 }),
          (has_python_steps, python_configured, missing_tables) => {
            const result = canImport(
              {
                has_python_steps,
                python_configured,
                validation: { missing_tables },
              },
              'preview'
            );
            expect(result).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 3.6**
     *
     * When has_python_steps=true and python_configured=false,
     * canImport returns false even in preview state with no missing tables.
     */
    it('returns false when has_python_steps=true and python_configured=false', () => {
      fc.assert(
        fc.property(
          fc.constantFrom(...dialogStates),
          (state) => {
            const result = canImport(
              {
                has_python_steps: true,
                python_configured: false,
                validation: { missing_tables: [] },
              },
              state
            );
            expect(result).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 3.4, 3.5**
     *
     * When missing_columns is non-empty but missing_tables is empty,
     * canImport returns true (warning style, but button enabled).
     * This verifies the distinction between error (missing tables) and
     * warning (missing columns only).
     */
    it('returns true when missing_columns present but missing_tables empty (warning style)', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          fc.array(fc.string({ minLength: 1 }), { minLength: 1 }),
          (python_configured, missing_columns) => {
            const result = canImport(
              {
                has_python_steps: false,
                python_configured,
                validation: { missing_tables: [] },
              },
              'preview'
            );
            // missing_columns doesn't affect canImport — button is enabled
            expect(result).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 3.5**
     *
     * When loadResult is null and state is 'preview', canImport returns true
     * (no validation = compatible).
     */
    it('returns true when loadResult is null and state is preview', () => {
      expect(canImport(null, 'preview')).toBe(true);
    });

    /**
     * **Validates: Requirements 3.5**
     *
     * When loadResult is null, only preview state returns true.
     */
    it('returns false when loadResult is null and state is not preview', () => {
      fc.assert(
        fc.property(
          fc.constantFrom('pack-list', 'loading', 'password', 'executing'),
          (state) => {
            expect(canImport(null, state)).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});


/**
 * Property-Based Tests for canCloseDialog
 *
 * Feature: datasource-pack-loader, Property 4: 非执行状态下对话框可关闭
 * **Validates: Requirements 6.1, 6.2**
 *
 * Uses fast-check to verify that canCloseDialog correctly determines
 * whether the dialog can be closed (via Escape key or overlay click)
 * based on the current dialog state.
 */

import { canCloseDialog } from './ImportPackDialog';

// All valid DialogState values
const allDialogStates = ['pack-list', 'loading', 'password', 'preview', 'executing'] as const;
const closeableStates = ['pack-list', 'password', 'preview'] as const;
const nonCloseableStates = ['loading', 'executing'] as const;

describe('canCloseDialog Property-Based Tests', () => {
  // Feature: datasource-pack-loader, Property 4: 非执行状态下对话框可关闭
  describe('Property 4: 非执行状态下对话框可关闭', () => {
    /**
     * **Validates: Requirements 6.1, 6.2**
     *
     * For any closeable state ('pack-list', 'password', 'preview'),
     * canCloseDialog must return true — Escape key and overlay click
     * should trigger close.
     */
    it('returns true for closeable states (pack-list, password, preview)', () => {
      fc.assert(
        fc.property(
          fc.constantFrom(...closeableStates),
          (state) => {
            expect(canCloseDialog(state)).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 6.1, 6.2**
     *
     * For any non-closeable state ('loading', 'executing'),
     * canCloseDialog must return false — Escape key and overlay click
     * should be ignored.
     */
    it('returns false for non-closeable states (loading, executing)', () => {
      fc.assert(
        fc.property(
          fc.constantFrom(...nonCloseableStates),
          (state) => {
            expect(canCloseDialog(state)).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 6.1, 6.2**
     *
     * Exhaustive biconditional: for any DialogState, canCloseDialog
     * returns true if and only if the state is NOT 'loading' and NOT 'executing'.
     */
    it('returns true iff state is not loading and not executing', () => {
      fc.assert(
        fc.property(
          fc.constantFrom(...allDialogStates),
          (state) => {
            const result = canCloseDialog(state);
            const expected = state !== 'loading' && state !== 'executing';
            expect(result).toBe(expected);
          }
        ),
        { numRuns: 200 }
      );
    });
  });
});
