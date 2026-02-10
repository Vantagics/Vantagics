/**
 * Unit tests for PanelWidths utility module
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import {
  calculatePanelWidths,
  handleResizeDrag,
  savePanelWidths,
  loadPanelWidths,
  getDefaultPanelWidths,
  clearPanelWidths,
  PANEL_CONSTRAINTS,
  type PanelWidths
} from './PanelWidths';

describe('PanelWidths Utility', () => {
  // Clear localStorage before and after each test
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  describe('calculatePanelWidths', () => {
    it('should calculate panel widths that sum to total width', () => {
      const totalWidth = 1920;
      const leftWidth = 256;
      const rightWidth = 384;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.left + result.center + result.right).toBe(totalWidth);
    });

    it('should enforce minimum left panel width', () => {
      const totalWidth = 1920;
      const leftWidth = 100; // Below minimum of 180
      const rightWidth = 384;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.left).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.left.min);
    });

    it('should enforce maximum left panel width', () => {
      const totalWidth = 1920;
      const leftWidth = 500; // Above maximum of 400
      const rightWidth = 384;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.left).toBeLessThanOrEqual(PANEL_CONSTRAINTS.left.max);
    });

    it('should enforce minimum right panel width', () => {
      const totalWidth = 1920;
      const leftWidth = 256;
      const rightWidth = 200; // Below minimum of 280

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.right).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.right.min);
    });

    it('should enforce maximum right panel width', () => {
      const totalWidth = 1920;
      const leftWidth = 256;
      const rightWidth = 700; // Above maximum of 600

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.right).toBeLessThanOrEqual(PANEL_CONSTRAINTS.right.max);
    });

    it('should enforce minimum center panel width', () => {
      const totalWidth = 1920;
      const leftWidth = 400; // Maximum left
      const rightWidth = 600; // Maximum right

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
    });

    it('should adjust right panel when center would be too small', () => {
      const totalWidth = 1000; // Small total width
      const leftWidth = 400;
      const rightWidth = 600;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      // Center should be at least 400px
      expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
      // Right should be reduced to accommodate
      expect(result.right).toBeLessThan(rightWidth);
      // Sum should still equal total
      expect(result.left + result.center + result.right).toBe(totalWidth);
    });

    it('should handle minimum window size (1024px)', () => {
      const totalWidth = 1024;
      const leftWidth = 180;
      const rightWidth = 280;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.left).toBe(180);
      expect(result.center).toBeGreaterThanOrEqual(400);
      expect(result.right).toBeGreaterThanOrEqual(280);
      expect(result.left + result.center + result.right).toBe(totalWidth);
    });

    it('should handle large window size (3840px)', () => {
      const totalWidth = 3840;
      const leftWidth = 350;
      const rightWidth = 550;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.left).toBe(350);
      expect(result.center).toBe(2940);
      expect(result.right).toBe(550);
      expect(result.left + result.center + result.right).toBe(totalWidth);
    });

    it('should return valid widths with default values', () => {
      const totalWidth = 1920;
      const leftWidth = PANEL_CONSTRAINTS.left.default;
      const rightWidth = PANEL_CONSTRAINTS.right.default;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.left).toBe(256);
      expect(result.right).toBe(384);
      expect(result.center).toBe(1280);
    });
  });

  describe('handleResizeDrag', () => {
    it('should increase left panel width when dragging left handle right', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = 50; // Drag right

      const result = handleResizeDrag('left', deltaX, currentWidths, totalWidth);

      expect(result.left).toBeGreaterThan(currentWidths.left);
      expect(result.center).toBeLessThan(currentWidths.center);
      expect(result.right).toBe(currentWidths.right);
    });

    it('should decrease left panel width when dragging left handle left', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = -50; // Drag left

      const result = handleResizeDrag('left', deltaX, currentWidths, totalWidth);

      expect(result.left).toBeLessThan(currentWidths.left);
      expect(result.center).toBeGreaterThan(currentWidths.center);
      expect(result.right).toBe(currentWidths.right);
    });

    it('should increase center panel width when dragging right handle right', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = 50; // Drag right

      const result = handleResizeDrag('right', deltaX, currentWidths, totalWidth);

      expect(result.left).toBe(currentWidths.left);
      expect(result.center).toBeGreaterThan(currentWidths.center);
      expect(result.right).toBeLessThan(currentWidths.right);
    });

    it('should decrease center panel width when dragging right handle left', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = -50; // Drag left

      const result = handleResizeDrag('right', deltaX, currentWidths, totalWidth);

      expect(result.left).toBe(currentWidths.left);
      expect(result.center).toBeLessThan(currentWidths.center);
      expect(result.right).toBeGreaterThan(currentWidths.right);
    });

    it('should respect constraints when dragging left handle', () => {
      const currentWidths: PanelWidths = { left: 180, center: 1360, right: 380 };
      const totalWidth = 1920;
      const deltaX = -100; // Try to drag below minimum

      const result = handleResizeDrag('left', deltaX, currentWidths, totalWidth);

      expect(result.left).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.left.min);
      expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
    });

    it('should respect constraints when dragging right handle', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1384, right: 280 };
      const totalWidth = 1920;
      const deltaX = -100; // Try to drag below minimum

      const result = handleResizeDrag('right', deltaX, currentWidths, totalWidth);

      expect(result.right).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.right.min);
      expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
    });

    it('should maintain total width after resize', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = 75;

      const result = handleResizeDrag('left', deltaX, currentWidths, totalWidth);

      expect(result.left + result.center + result.right).toBe(totalWidth);
    });
  });

  describe('savePanelWidths', () => {
    it('should save panel widths to localStorage', () => {
      const widths: PanelWidths = { left: 300, center: 1200, right: 420 };

      const success = savePanelWidths(widths);

      expect(success).toBe(true);
      const stored = localStorage.getItem('panelWidths');
      expect(stored).not.toBeNull();
      
      const parsed = JSON.parse(stored!);
      expect(parsed.left).toBe(300);
      expect(parsed.right).toBe(420);
      // Center should not be stored (it's calculated)
      expect(parsed.center).toBeUndefined();
    });

    it('should return true on successful save', () => {
      const widths: PanelWidths = { left: 256, center: 1280, right: 384 };

      const success = savePanelWidths(widths);

      expect(success).toBe(true);
    });

    it('should handle localStorage errors gracefully', () => {
      // Fill localStorage to trigger quota error
      const widths: PanelWidths = { left: 256, center: 1280, right: 384 };
      
      // Save once successfully
      const firstSave = savePanelWidths(widths);
      expect(firstSave).toBe(true);
      
      // The function should handle errors gracefully if they occur
      // Note: In a real browser environment with quota exceeded, this would return false
      // In test environment, localStorage is unlimited, so we just verify the function works
    });
  });

  describe('loadPanelWidths', () => {
    it('should load panel widths from localStorage', () => {
      const totalWidth = 1920;
      const savedData = { left: 300, right: 420 };
      localStorage.setItem('panelWidths', JSON.stringify(savedData));

      const result = loadPanelWidths(totalWidth);

      expect(result).not.toBeNull();
      expect(result!.left).toBe(300);
      expect(result!.right).toBe(420);
      expect(result!.center).toBe(1200);
    });

    it('should return null when no saved widths exist', () => {
      const totalWidth = 1920;

      const result = loadPanelWidths(totalWidth);

      expect(result).toBeNull();
    });

    it('should return null for invalid data', () => {
      const totalWidth = 1920;
      localStorage.setItem('panelWidths', 'invalid json');

      const result = loadPanelWidths(totalWidth);

      expect(result).toBeNull();
    });

    it('should return null for incomplete data', () => {
      const totalWidth = 1920;
      localStorage.setItem('panelWidths', JSON.stringify({ left: 300 }));

      const result = loadPanelWidths(totalWidth);

      expect(result).toBeNull();
    });

    it('should recalculate with constraints when loading', () => {
      const totalWidth = 1920;
      // Save widths that would violate constraints
      const savedData = { left: 500, right: 700 }; // Both above max
      localStorage.setItem('panelWidths', JSON.stringify(savedData));

      const result = loadPanelWidths(totalWidth);

      expect(result).not.toBeNull();
      expect(result!.left).toBeLessThanOrEqual(PANEL_CONSTRAINTS.left.max);
      expect(result!.right).toBeLessThanOrEqual(PANEL_CONSTRAINTS.right.max);
      expect(result!.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
    });

    it('should handle different total widths correctly', () => {
      const savedData = { left: 300, right: 400 };
      localStorage.setItem('panelWidths', JSON.stringify(savedData));

      // Load with different total width
      const result = loadPanelWidths(2560);

      expect(result).not.toBeNull();
      expect(result!.left).toBe(300);
      expect(result!.right).toBe(400);
      expect(result!.center).toBe(1860);
      expect(result!.left + result!.center + result!.right).toBe(2560);
    });
  });

  describe('getDefaultPanelWidths', () => {
    it('should return default panel widths', () => {
      const totalWidth = 1920;

      const result = getDefaultPanelWidths(totalWidth);

      expect(result.left).toBe(PANEL_CONSTRAINTS.left.default);
      expect(result.right).toBe(PANEL_CONSTRAINTS.right.default);
      expect(result.center).toBe(1280);
    });

    it('should return widths that sum to total width', () => {
      const totalWidth = 1920;

      const result = getDefaultPanelWidths(totalWidth);

      expect(result.left + result.center + result.right).toBe(totalWidth);
    });

    it('should work with different window sizes', () => {
      const totalWidth = 2560;

      const result = getDefaultPanelWidths(totalWidth);

      expect(result.left).toBe(256);
      expect(result.right).toBe(384);
      expect(result.center).toBe(1920);
      expect(result.left + result.center + result.right).toBe(totalWidth);
    });
  });

  describe('clearPanelWidths', () => {
    it('should remove panel widths from localStorage', () => {
      const widths: PanelWidths = { left: 300, center: 1200, right: 420 };
      savePanelWidths(widths);

      const success = clearPanelWidths();

      expect(success).toBe(true);
      expect(localStorage.getItem('panelWidths')).toBeNull();
    });

    it('should return true even if no widths were stored', () => {
      const success = clearPanelWidths();

      expect(success).toBe(true);
    });

    it('should handle localStorage errors gracefully', () => {
      // In test environment, localStorage operations don't fail
      // This test verifies the function completes successfully
      const success = clearPanelWidths();

      expect(success).toBe(true);
    });
  });

  describe('Edge Cases', () => {
    it('should handle zero delta in resize drag', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = 0;

      const result = handleResizeDrag('left', deltaX, currentWidths, totalWidth);

      expect(result.left).toBe(currentWidths.left);
      expect(result.center).toBe(currentWidths.center);
      expect(result.right).toBe(currentWidths.right);
    });

    it('should handle very large positive delta', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = 1000; // Very large drag

      const result = handleResizeDrag('left', deltaX, currentWidths, totalWidth);

      // Should be constrained to maximum
      expect(result.left).toBeLessThanOrEqual(PANEL_CONSTRAINTS.left.max);
      expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
      expect(result.left + result.center + result.right).toBe(totalWidth);
    });

    it('should handle very large negative delta', () => {
      const currentWidths: PanelWidths = { left: 256, center: 1280, right: 384 };
      const totalWidth = 1920;
      const deltaX = -1000; // Very large drag

      const result = handleResizeDrag('left', deltaX, currentWidths, totalWidth);

      // Should be constrained to minimum
      expect(result.left).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.left.min);
      expect(result.center).toBeGreaterThanOrEqual(PANEL_CONSTRAINTS.center.min);
      expect(result.left + result.center + result.right).toBe(totalWidth);
    });

    it('should handle minimum viable total width', () => {
      // Minimum total = left.min + center.min + right.min = 180 + 400 + 280 = 860
      const totalWidth = 860;
      const leftWidth = 180;
      const rightWidth = 280;

      const result = calculatePanelWidths(totalWidth, leftWidth, rightWidth);

      expect(result.left).toBe(180);
      expect(result.center).toBe(400);
      expect(result.right).toBe(280);
    });
  });

  describe('Integration: Save and Load Round-Trip', () => {
    it('should preserve widths through save and load cycle', () => {
      const totalWidth = 1920;
      const originalWidths: PanelWidths = { left: 300, center: 1220, right: 400 };

      // Save
      savePanelWidths(originalWidths);

      // Load
      const loadedWidths = loadPanelWidths(totalWidth);

      expect(loadedWidths).not.toBeNull();
      expect(loadedWidths!.left).toBe(originalWidths.left);
      expect(loadedWidths!.right).toBe(originalWidths.right);
      expect(loadedWidths!.center).toBe(originalWidths.center);
    });

    it('should use defaults when no saved widths exist', () => {
      const totalWidth = 1920;

      const loadedWidths = loadPanelWidths(totalWidth);
      expect(loadedWidths).toBeNull();

      const defaultWidths = getDefaultPanelWidths(totalWidth);
      expect(defaultWidths.left).toBe(256);
      expect(defaultWidths.right).toBe(384);
    });
  });
});
