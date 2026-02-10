/**
 * Property-Based Tests for PanelWidths Utility Module
 *
 * Uses fast-check to verify universal properties across randomized inputs.
 * Each property test runs a minimum of 100 iterations.
 */

import fc from 'fast-check';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import {
  calculatePanelWidths,
  handleResizeDrag,
  savePanelWidths,
  loadPanelWidths,
  PANEL_CONSTRAINTS,
  type PanelWidths,
} from './PanelWidths';

/**
 * Minimum viable totalWidth = left.min + center.min + right.min = 180 + 400 + 280 = 860
 */
const MIN_TOTAL_WIDTH = PANEL_CONSTRAINTS.left.min + PANEL_CONSTRAINTS.center.min + PANEL_CONSTRAINTS.right.min;

/**
 * Arbitrary that generates a valid PanelWidths object for a given totalWidth.
 * Left and right are constrained to their valid ranges, center is the remainder.
 */
function validPanelWidthsArb(totalWidth: number): fc.Arbitrary<PanelWidths> {
  // Compute the feasible range for left given the totalWidth
  const leftMax = Math.min(
    PANEL_CONSTRAINTS.left.max,
    totalWidth - PANEL_CONSTRAINTS.center.min - PANEL_CONSTRAINTS.right.min
  );
  const leftMin = PANEL_CONSTRAINTS.left.min;

  return fc
    .integer({ min: leftMin, max: Math.max(leftMin, leftMax) })
    .chain((left) => {
      const rightMax = Math.min(
        PANEL_CONSTRAINTS.right.max,
        totalWidth - left - PANEL_CONSTRAINTS.center.min
      );
      const rightMin = PANEL_CONSTRAINTS.right.min;

      return fc
        .integer({ min: rightMin, max: Math.max(rightMin, rightMax) })
        .map((right) => ({
          left,
          center: totalWidth - left - right,
          right,
        }));
    });
}

describe('PanelWidths Property-Based Tests', () => {
  // ─────────────────────────────────────────────────────────────────────────
  // Feature: layout-simplification, Property 1: 面板宽度约束不变量
  // ─────────────────────────────────────────────────────────────────────────
  describe('Property 1: 面板宽度约束不变量', () => {
    /**
     * **Validates: Requirements 3.3**
     *
     * For any totalWidth (≥ 860), leftWidth, and rightWidth,
     * calculatePanelWidths must return widths that:
     *   1. left  >= PANEL_CONSTRAINTS.left.min   (180)
     *   2. center >= PANEL_CONSTRAINTS.center.min (400)
     *   3. right >= PANEL_CONSTRAINTS.right.min   (280)
     *   4. left + center + right === totalWidth
     */
    it('calculatePanelWidths always satisfies min constraints and sums to totalWidth', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: MIN_TOTAL_WIDTH, max: 3840 }), // totalWidth
          fc.integer({ min: 0, max: 1000 }),                // leftWidth (unconstrained input)
          fc.integer({ min: 0, max: 1000 }),                // rightWidth (unconstrained input)
          (totalWidth, leftWidth, rightWidth) => {
            const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

            // Min constraints
            expect(result.left).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.left.min);
            expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
            expect(result.right).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.right.min);

            // Sum invariant
            expect(result.left + result.center + result.right).toBe(totalWidth);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // ─────────────────────────────────────────────────────────────────────────
  // Feature: layout-simplification, Property 2: 拖拽调整大小的正确性
  // ─────────────────────────────────────────────────────────────────────────
  describe('Property 2: 拖拽调整大小的正确性', () => {
    /**
     * **Validates: Requirements 3.1, 3.2**
     *
     * For any handle position ('left' | 'right'), any deltaX,
     * any valid currentWidths, and any totalWidth (≥ 860),
     * handleResizeDrag must return widths that:
     *   1. Satisfy all panel min constraints
     *   2. Sum to totalWidth
     */
    it('handleResizeDrag always satisfies constraints and sums to totalWidth', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: MIN_TOTAL_WIDTH, max: 3840 }).chain((totalWidth) =>
            fc.tuple(
              fc.constant(totalWidth),
              fc.constantFrom<'left' | 'right'>('left', 'right'),
              fc.integer({ min: -500, max: 500 }),           // deltaX
              validPanelWidthsArb(totalWidth)                 // currentWidths
            )
          ),
          ([totalWidth, handlePosition, deltaX, currentWidths]) => {
            const result = handleResizeDrag(handlePosition, deltaX, currentWidths, totalWidth);

            // Min constraints
            expect(result.left).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.left.min);
            expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
            expect(result.right).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.right.min);

            // Sum invariant
            expect(result.left + result.center + result.right).toBe(totalWidth);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // ─────────────────────────────────────────────────────────────────────────
  // Feature: layout-simplification, Property 3: 面板宽度持久化往返一致性
  // ─────────────────────────────────────────────────────────────────────────
  describe('Property 3: 面板宽度持久化往返一致性', () => {
    beforeEach(() => {
      localStorage.clear();
    });

    afterEach(() => {
      localStorage.clear();
    });

    /**
     * **Validates: Requirements 3.4**
     *
     * For any valid PanelWidths and the same totalWidth,
     * savePanelWidths(widths) followed by loadPanelWidths(totalWidth)
     * must return left and right values identical to the original.
     */
    it('save then load round-trip preserves left and right values', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: MIN_TOTAL_WIDTH, max: 3840 }).chain((totalWidth) =>
            fc.tuple(fc.constant(totalWidth), validPanelWidthsArb(totalWidth))
          ),
          ([totalWidth, widths]) => {
            // Clear before each iteration to avoid cross-contamination
            localStorage.clear();

            const saveSuccess = savePanelWidths(widths);
            expect(saveSuccess).toBe(true);

            const loaded = loadPanelWidths(totalWidth);
            expect(loaded).not.toBeNull();
            expect(loaded!.left).toBe(widths.left);
            expect(loaded!.right).toBe(widths.right);
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
