/**
 * Tests for UI/UX Polish Features
 * 
 * Tests for animations, visual feedback, loading states, error handling,
 * responsive design, keyboard shortcuts, and accessibility
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { VisualFeedback } from './VisualFeedback';
import { LoadingStatesManager } from './LoadingStates';
import { ErrorHandler, ErrorType, ErrorSeverity } from './ErrorHandler';
import { ResponsiveLayoutManager } from './ResponsiveLayout';
import { KeyboardShortcutsManager, KeyModifier } from './KeyboardShortcuts';
import { AccessibilityManager } from './AccessibilityManager';

describe('UI/UX Polish Features', () => {
  let container: HTMLElement;

  beforeEach(() => {
    // Create test container
    container = document.createElement('div');
    container.id = 'test-container';
    document.body.appendChild(container);
  });

  afterEach(() => {
    // Cleanup
    document.body.removeChild(container);
    vi.clearAllMocks();
  });

  describe('Visual Feedback System', () => {
    let visualFeedback: VisualFeedback;

    beforeEach(() => {
      visualFeedback = new VisualFeedback(container);
    });

    afterEach(() => {
      visualFeedback.cleanup();
    });

    it('should show and hide snap indicator', () => {
      const bounds = { x: 10, y: 20, width: 100, height: 50 };
      
      visualFeedback.showSnapIndicator(bounds);
      
      const indicator = container.querySelector('.grid-snap-indicator') as HTMLElement;
      expect(indicator).toBeTruthy();
      expect(indicator.style.display).toBe('block');
      expect(indicator.style.left).toBe('10px');
      expect(indicator.style.top).toBe('20px');
      expect(indicator.style.width).toBe('100px');
      expect(indicator.style.height).toBe('50px');

      visualFeedback.hideSnapIndicator();
      expect(indicator.style.display).toBe('none');
    });

    it('should show and hide drag preview', () => {
      const element = document.createElement('div');
      element.textContent = 'Test Component';
      element.style.width = '200px';
      element.style.height = '100px';
      
      const position = { x: 50, y: 75 };
      
      visualFeedback.showDragPreview(element, position);
      
      const preview = container.querySelector('.drag-preview') as HTMLElement;
      expect(preview).toBeTruthy();
      expect(preview.style.display).toBe('block');
      expect(preview.style.left).toBe('50px');
      expect(preview.style.top).toBe('75px');

      visualFeedback.hideDragPreview();
      expect(preview.style.display).toBe('none');
    });

    it('should show and hide resize preview', () => {
      const bounds = { x: 30, y: 40, width: 150, height: 80 };
      
      visualFeedback.showResizePreview(bounds);
      
      const preview = container.querySelector('.resize-preview') as HTMLElement;
      expect(preview).toBeTruthy();
      expect(preview.style.display).toBe('block');
      expect(preview.style.left).toBe('30px');
      expect(preview.style.top).toBe('40px');
      expect(preview.style.width).toBe('150px');
      expect(preview.style.height).toBe('80px');

      visualFeedback.hideResizePreview();
      expect(preview.style.display).toBe('none');
    });

    it('should add and remove drag state', () => {
      const element = document.createElement('div');
      container.appendChild(element);
      
      visualFeedback.addDragState(element);
      expect(element.classList.contains('draggable-component--dragging')).toBe(true);
      
      visualFeedback.removeDragState(element);
      expect(element.classList.contains('draggable-component--dragging')).toBe(false);
    });

    it('should show collision warning', () => {
      const element = document.createElement('div');
      container.appendChild(element);
      
      visualFeedback.showCollisionWarning(element);
      expect(element.classList.contains('draggable-component--collision-warning')).toBe(true);
      
      // Should remove warning after animation
      setTimeout(() => {
        expect(element.classList.contains('draggable-component--collision-warning')).toBe(false);
      }, 350);
    });

    it('should animate component addition', () => {
      const element = document.createElement('div');
      container.appendChild(element);
      
      visualFeedback.animateComponentAddition(element);
      expect(element.classList.contains('draggable-component--newly-added')).toBe(true);
      
      setTimeout(() => {
        expect(element.classList.contains('draggable-component--newly-added')).toBe(false);
      }, 350);
    });

    it('should animate component removal', async () => {
      const element = document.createElement('div');
      container.appendChild(element);
      
      const promise = visualFeedback.animateComponentRemoval(element);
      expect(element.classList.contains('draggable-component--removing')).toBe(true);
      
      await promise;
      // Animation should complete
    });
  });

  describe('Loading States Manager', () => {
    let loadingManager: LoadingStatesManager;

    beforeEach(() => {
      loadingManager = new LoadingStatesManager();
    });

    it('should set and get loading state', () => {
      loadingManager.setLoading('comp1', 'metrics', true);
      
      const state = loadingManager.getState('comp1');
      expect(state).toBeTruthy();
      expect(state!.isLoading).toBe(true);
      expect(state!.componentType).toBe('metrics');
      expect(state!.error).toBe(null);
    });

    it('should set error state', () => {
      loadingManager.setLoading('comp1', 'metrics', true);
      loadingManager.setError('comp1', 'Failed to load data');
      
      const state = loadingManager.getState('comp1');
      expect(state!.isLoading).toBe(false);
      expect(state!.error).toBe('Failed to load data');
      expect(state!.lastUpdated).toBeTruthy();
    });

    it('should set success state', () => {
      loadingManager.setLoading('comp1', 'metrics', true);
      loadingManager.setSuccess('comp1', true);
      
      const state = loadingManager.getState('comp1');
      expect(state!.isLoading).toBe(false);
      expect(state!.error).toBe(null);
      expect(state!.hasData).toBe(true);
      expect(state!.lastUpdated).toBeTruthy();
    });

    it('should check if any component is loading', () => {
      expect(loadingManager.isAnyLoading()).toBe(false);
      
      loadingManager.setLoading('comp1', 'metrics', true);
      expect(loadingManager.isAnyLoading()).toBe(true);
      
      loadingManager.setSuccess('comp1', true);
      expect(loadingManager.isAnyLoading()).toBe(false);
    });

    it('should get states by type', () => {
      loadingManager.setLoading('comp1', 'metrics', true);
      loadingManager.setLoading('comp2', 'table', true);
      loadingManager.setLoading('comp3', 'metrics', false);
      
      const metricsStates = loadingManager.getStatesByType('metrics');
      expect(metricsStates).toHaveLength(2);
      expect(metricsStates.every(s => s.componentType === 'metrics')).toBe(true);
    });

    it('should handle loading operation wrapper', async () => {
      const operation = vi.fn().mockResolvedValue('success');
      
      const result = await loadingManager.withLoading('comp1', 'metrics', operation);
      
      expect(result).toBe('success');
      expect(operation).toHaveBeenCalled();
      
      const state = loadingManager.getState('comp1');
      expect(state!.isLoading).toBe(false);
      expect(state!.hasData).toBe(true);
    });

    it('should handle loading operation failure', async () => {
      const operation = vi.fn().mockRejectedValue(new Error('Test error'));
      
      await expect(loadingManager.withLoading('comp1', 'metrics', operation)).rejects.toThrow('Test error');
      
      const state = loadingManager.getState('comp1');
      expect(state!.isLoading).toBe(false);
      expect(state!.error).toBe('Test error');
    });

    it('should batch update states', () => {
      const updates = [
        { componentId: 'comp1', componentType: 'metrics', isLoading: true },
        { componentId: 'comp2', componentType: 'table', error: 'Failed' },
        { componentId: 'comp3', componentType: 'image', hasData: true },
      ];
      
      loadingManager.batchUpdate(updates);
      
      expect(loadingManager.getState('comp1')!.isLoading).toBe(true);
      expect(loadingManager.getState('comp2')!.error).toBe('Failed');
      expect(loadingManager.getState('comp3')!.hasData).toBe(true);
    });

    it('should provide loading statistics', () => {
      // Initialize components first
      loadingManager.setLoading('comp1', 'metrics', true);
      loadingManager.setLoading('comp2', 'table', false);
      loadingManager.setLoading('comp3', 'image', false);
      
      // Set different states
      loadingManager.setError('comp2', 'Error');
      loadingManager.setSuccess('comp3', true);
      
      const stats = loadingManager.getStats();
      expect(stats.total).toBe(3);
      expect(stats.loading).toBe(1);
      expect(stats.error).toBe(1);
      expect(stats.success).toBe(1);
    });
  });

  describe('Error Handler', () => {
    let errorHandler: ErrorHandler;

    beforeEach(() => {
      errorHandler = new ErrorHandler();
    });

    it('should handle errors', () => {
      const errorId = errorHandler.handleError(
        ErrorType.LAYOUT_SAVE_FAILED,
        'Failed to save layout',
        'Network timeout',
        'comp1',
        ErrorSeverity.HIGH
      );
      
      expect(errorId).toBeTruthy();
      
      const errors = errorHandler.getAllErrors();
      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe(ErrorType.LAYOUT_SAVE_FAILED);
      expect(errors[0].message).toBe('Failed to save layout');
      expect(errors[0].severity).toBe(ErrorSeverity.HIGH);
    });

    it('should handle exceptions', () => {
      const error = new Error('Test error');
      const errorId = errorHandler.handleException(error, 'Test context', 'comp1');
      
      expect(errorId).toBeTruthy();
      
      const errors = errorHandler.getAllErrors();
      expect(errors).toHaveLength(1);
      expect(errors[0].message).toContain('Test context: Test error');
    });

    it('should provide user-friendly messages', () => {
      const message = errorHandler.getUserFriendlyMessage(ErrorType.NETWORK_ERROR);
      expect(message).toBe('Network connection error. Please check your internet connection and try again.');
    });

    it('should provide recovery actions', () => {
      const errorId = errorHandler.handleError(ErrorType.LAYOUT_SAVE_FAILED, 'Save failed');
      const actions = errorHandler.getRecoveryActions(errorId);
      
      expect(actions.length).toBeGreaterThan(0);
      expect(actions.some(action => action.label === 'Retry Save')).toBe(true);
      expect(actions.some(action => action.label === 'Dismiss')).toBe(true);
    });

    it('should dismiss errors', () => {
      const errorId = errorHandler.handleError(ErrorType.UNKNOWN_ERROR, 'Test error');
      expect(errorHandler.getAllErrors()).toHaveLength(1);
      
      errorHandler.dismissError(errorId);
      expect(errorHandler.getAllErrors()).toHaveLength(0);
    });

    it('should filter errors by component', () => {
      errorHandler.handleError(ErrorType.COMPONENT_DATA_FAILED, 'Error 1', undefined, 'comp1');
      errorHandler.handleError(ErrorType.COMPONENT_DATA_FAILED, 'Error 2', undefined, 'comp2');
      errorHandler.handleError(ErrorType.COMPONENT_DATA_FAILED, 'Error 3', undefined, 'comp1');
      
      const comp1Errors = errorHandler.getErrorsByComponent('comp1');
      expect(comp1Errors).toHaveLength(2);
    });

    it('should filter errors by severity', () => {
      errorHandler.handleError(ErrorType.UNKNOWN_ERROR, 'Low error', undefined, undefined, ErrorSeverity.LOW);
      errorHandler.handleError(ErrorType.UNKNOWN_ERROR, 'High error', undefined, undefined, ErrorSeverity.HIGH);
      errorHandler.handleError(ErrorType.UNKNOWN_ERROR, 'Critical error', undefined, undefined, ErrorSeverity.CRITICAL);
      
      const criticalErrors = errorHandler.getErrorsBySeverity(ErrorSeverity.CRITICAL);
      expect(criticalErrors).toHaveLength(1);
      expect(criticalErrors[0].message).toBe('Critical error');
    });

    it('should detect critical errors', () => {
      expect(errorHandler.hasCriticalErrors()).toBe(false);
      
      errorHandler.handleError(ErrorType.UNKNOWN_ERROR, 'Critical error', undefined, undefined, ErrorSeverity.CRITICAL);
      expect(errorHandler.hasCriticalErrors()).toBe(true);
    });
  });

  describe('Responsive Layout Manager', () => {
    let responsiveManager: ResponsiveLayoutManager;

    beforeEach(() => {
      responsiveManager = new ResponsiveLayoutManager();
      responsiveManager.setContainer(container);
    });

    afterEach(() => {
      responsiveManager.cleanup();
    });

    it('should detect viewport changes', () => {
      const callback = vi.fn();
      responsiveManager.subscribe(callback);
      
      // Simulate viewport change
      Object.defineProperty(container, 'clientWidth', { value: 800, configurable: true });
      Object.defineProperty(container, 'clientHeight', { value: 600, configurable: true });
      
      // Trigger resize
      window.dispatchEvent(new Event('resize'));
      
      expect(callback).toHaveBeenCalled();
    });

    it('should get breakpoint for width', () => {
      const breakpoint = responsiveManager['getBreakpointForWidth'](800);
      expect(breakpoint.name).toBe('md');
      expect(breakpoint.columns).toBe(12);
    });

    it('should convert layout for different breakpoints', () => {
      const layout = [
        { i: 'comp1', x: 0, y: 0, w: 12, h: 4 },
        { i: 'comp2', x: 12, y: 0, w: 12, h: 4 },
      ];
      
      const convertedLayout = responsiveManager.convertLayoutForBreakpoint(layout, 'xl', 'md');
      
      expect(convertedLayout[0].w).toBe(6); // 12 * (12/24) = 6
      expect(convertedLayout[1].x).toBe(6); // 12 * (12/24) = 6
    });

    it('should get optimal component size', () => {
      // Mock current viewport
      responsiveManager['currentViewport'] = {
        width: 400,
        height: 600,
        breakpoint: { name: 'xs', minWidth: 0, maxWidth: 575, columns: 4, margin: [5, 5], containerPadding: [5, 5], rowHeight: 60 },
        isMobile: true,
        isTablet: false,
        isDesktop: false,
        orientation: 'portrait',
      };
      
      const size = responsiveManager.getOptimalComponentSize('table', { w: 12, h: 6 });
      expect(size.w).toBe(4); // Full width on mobile
      expect(size.h).toBeGreaterThanOrEqual(4);
    });

    it('should disable drag/resize on small screens', () => {
      responsiveManager['currentViewport'] = {
        width: 300,
        height: 600,
        breakpoint: { name: 'xs', minWidth: 0, maxWidth: 575, columns: 4, margin: [5, 5], containerPadding: [5, 5], rowHeight: 60 },
        isMobile: true,
        isTablet: false,
        isDesktop: false,
        orientation: 'portrait',
      };
      
      expect(responsiveManager.shouldDisableDragResize()).toBe(true);
    });

    it('should provide touch-friendly handle sizes', () => {
      responsiveManager['currentViewport'] = {
        width: 400,
        height: 600,
        breakpoint: { name: 'sm', minWidth: 576, maxWidth: 767, columns: 8, margin: [8, 8], containerPadding: [8, 8], rowHeight: 70 },
        isMobile: true,
        isTablet: false,
        isDesktop: false,
        orientation: 'portrait',
      };
      
      expect(responsiveManager.getTouchHandleSize()).toBe(16);
    });
  });

  describe('Keyboard Shortcuts Manager', () => {
    let keyboardManager: KeyboardShortcutsManager;

    beforeEach(() => {
      keyboardManager = new KeyboardShortcutsManager();
    });

    afterEach(() => {
      keyboardManager.cleanup();
    });

    it('should add and remove shortcuts', () => {
      const shortcut = {
        id: 'test-shortcut',
        key: 't',
        modifiers: [KeyModifier.CTRL],
        description: 'Test shortcut',
        action: vi.fn(),
        enabled: true,
      };
      
      keyboardManager.addShortcut(shortcut);
      expect(keyboardManager.getShortcut('test-shortcut')).toBeTruthy();
      
      keyboardManager.removeShortcut('test-shortcut');
      expect(keyboardManager.getShortcut('test-shortcut')).toBe(null);
    });

    it('should enable/disable shortcuts', () => {
      const shortcut = keyboardManager.getShortcut('toggle-edit-mode');
      expect(shortcut!.enabled).toBe(true);
      
      keyboardManager.setShortcutEnabled('toggle-edit-mode', false);
      expect(keyboardManager.getShortcut('toggle-edit-mode')!.enabled).toBe(false);
    });

    it('should handle context switching', () => {
      keyboardManager.setContext('edit-mode');
      
      const contextShortcuts = keyboardManager.getContextShortcuts();
      const editModeShortcuts = contextShortcuts.filter(s => s.context === 'edit-mode');
      
      expect(editModeShortcuts.length).toBeGreaterThan(0);
    });

    it('should format shortcuts for display', () => {
      const shortcut = keyboardManager.getShortcut('toggle-edit-mode');
      const formatted = keyboardManager.formatShortcut(shortcut!);
      
      expect(formatted).toBe('Ctrl + e');
    });

    it('should match keyboard events to shortcuts', () => {
      const mockEvent = {
        key: 'e',
        ctrlKey: true,
        altKey: false,
        shiftKey: false,
        metaKey: false,
        preventDefault: vi.fn(),
        stopPropagation: vi.fn(),
        target: document.body,
      } as any;
      
      const matchingShortcut = keyboardManager['findMatchingShortcut'](mockEvent);
      expect(matchingShortcut?.id).toBe('toggle-edit-mode');
    });

    it('should not trigger shortcuts in input fields', () => {
      const input = document.createElement('input');
      container.appendChild(input);
      
      const mockEvent = {
        key: 'e',
        ctrlKey: true,
        altKey: false,
        shiftKey: false,
        metaKey: false,
        preventDefault: vi.fn(),
        stopPropagation: vi.fn(),
        target: input,
      } as any;
      
      keyboardManager['handleKeyDown'](mockEvent);
      expect(mockEvent.preventDefault).not.toHaveBeenCalled();
    });
  });

  describe('Accessibility Manager', () => {
    let accessibilityManager: AccessibilityManager;

    beforeEach(() => {
      accessibilityManager = new AccessibilityManager();
    });

    afterEach(() => {
      accessibilityManager.cleanup();
    });

    it('should create screen reader announcer', () => {
      const announcer = document.querySelector('[aria-live="polite"]');
      expect(announcer).toBeTruthy();
    });

    it('should announce messages', () => {
      const announcer = document.querySelector('[aria-live="polite"]') as HTMLElement;
      
      accessibilityManager.announce('Test message');
      expect(announcer.textContent).toBe('Test message');
      
      // Should clear after timeout
      setTimeout(() => {
        expect(announcer.textContent).toBe('');
      }, 1100);
    });

    it('should set ARIA attributes', () => {
      const element = document.createElement('div');
      
      accessibilityManager.setAriaAttributes(element, {
        role: 'button',
        label: 'Test button',
        expanded: false,
      });
      
      expect(element.getAttribute('aria-role')).toBe('button');
      expect(element.getAttribute('aria-label')).toBe('Test button');
      expect(element.getAttribute('aria-expanded')).toBe('false');
    });

    it('should make elements focusable', () => {
      const element = document.createElement('div');
      
      accessibilityManager.makeFocusable(element);
      
      expect(element.getAttribute('tabindex')).toBe('0');
      expect(element.getAttribute('role')).toBe('button');
    });

    it('should setup component accessibility', () => {
      const element = document.createElement('div');
      container.appendChild(element);
      
      accessibilityManager.setupComponentAccessibility(element, 'metrics', 'comp1', true);
      
      expect(element.getAttribute('tabindex')).toBe('0');
      expect(element.getAttribute('aria-role')).toBe('img');
      expect(element.getAttribute('aria-label')).toContain('Metrics chart with data');
    });

    it('should setup drag handle accessibility', () => {
      const handle = document.createElement('div');
      container.appendChild(handle);
      
      accessibilityManager.setupDragHandleAccessibility(handle, 'comp1');
      
      expect(handle.getAttribute('tabindex')).toBe('0');
      expect(handle.getAttribute('aria-role')).toBe('button');
      expect(handle.getAttribute('aria-label')).toContain('Drag handle for comp1');
      
      const instructions = handle.querySelector('.sr-only');
      expect(instructions).toBeTruthy();
    });

    it('should announce layout changes', () => {
      const announcer = document.querySelector('[aria-live="polite"]') as HTMLElement;
      
      accessibilityManager.announceLayoutChange('component-moved', 'to new position');
      expect(announcer.textContent).toBe('Component moved. to new position');
    });

    it('should handle high contrast mode', () => {
      accessibilityManager.setHighContrastMode(true);
      expect(document.body.classList.contains('high-contrast-mode')).toBe(true);
      
      accessibilityManager.setHighContrastMode(false);
      expect(document.body.classList.contains('high-contrast-mode')).toBe(false);
    });

    it('should detect keyboard navigation', () => {
      const tabEvent = new KeyboardEvent('keydown', { key: 'Tab' });
      document.dispatchEvent(tabEvent);
      
      expect(document.body.classList.contains('keyboard-navigation-active')).toBe(true);
      
      const mouseEvent = new MouseEvent('mousedown');
      document.dispatchEvent(mouseEvent);
      
      expect(document.body.classList.contains('keyboard-navigation-active')).toBe(false);
    });
  });

  describe('Integration Tests', () => {
    it('should work together for complete UX', () => {
      const visualFeedback = new VisualFeedback(container);
      const loadingManager = new LoadingStatesManager();
      const errorHandler = new ErrorHandler();
      const responsiveManager = new ResponsiveLayoutManager();
      const keyboardManager = new KeyboardShortcutsManager();
      const accessibilityManager = new AccessibilityManager();

      // Simulate a complete interaction flow
      
      // 1. Start loading
      loadingManager.setLoading('comp1', 'metrics', true);
      
      // 2. Show visual feedback
      const element = document.createElement('div');
      container.appendChild(element);
      visualFeedback.showLoadingState(element);
      
      // 3. Handle error
      const errorId = errorHandler.handleError(ErrorType.COMPONENT_DATA_FAILED, 'Failed to load');
      
      // 4. Announce to screen reader
      accessibilityManager.announce('Failed to load component data');
      
      // 5. Update loading state
      loadingManager.setError('comp1', 'Failed to load');
      
      // 6. Show error visual feedback
      visualFeedback.showErrorState(element, 'Failed to load');
      
      // Verify everything works together
      expect(loadingManager.getState('comp1')!.error).toBe('Failed to load');
      expect(errorHandler.getAllErrors()).toHaveLength(1);
      expect(element.querySelector('.error-state')).toBeTruthy();
      
      // Cleanup
      visualFeedback.cleanup();
      responsiveManager.cleanup();
      keyboardManager.cleanup();
      accessibilityManager.cleanup();
    });
  });
});