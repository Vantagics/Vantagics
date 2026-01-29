/**
 * Accessibility Manager for Dashboard Drag-Drop Layout
 * 
 * Provides accessibility features including ARIA labels, keyboard navigation,
 * screen reader support, and focus management
 */

export interface AccessibilityConfig {
  enableScreenReaderAnnouncements: boolean;
  enableKeyboardNavigation: boolean;
  enableFocusTrapping: boolean;
  enableHighContrast: boolean;
  announceLayoutChanges: boolean;
  announceComponentChanges: boolean;
}

export interface AriaAttributes {
  role?: string;
  label?: string;
  labelledby?: string;
  describedby?: string;
  expanded?: boolean;
  selected?: boolean;
  disabled?: boolean;
  hidden?: boolean;
  live?: 'off' | 'polite' | 'assertive';
  atomic?: boolean;
  relevant?: string;
}

/**
 * Accessibility manager for dashboard
 */
export class AccessibilityManager {
  private config: AccessibilityConfig;
  private announcer: HTMLElement | null = null;
  private focusedElement: HTMLElement | null = null;
  private focusableElements: HTMLElement[] = [];
  private isKeyboardNavigationActive = false;

  constructor(config?: Partial<AccessibilityConfig>) {
    this.config = {
      enableScreenReaderAnnouncements: true,
      enableKeyboardNavigation: true,
      enableFocusTrapping: false,
      enableHighContrast: false,
      announceLayoutChanges: true,
      announceComponentChanges: true,
      ...config,
    };

    this.initializeAccessibility();
  }

  /**
   * Initialize accessibility features
   */
  private initializeAccessibility(): void {
    this.createScreenReaderAnnouncer();
    this.setupKeyboardNavigation();
    this.setupFocusManagement();
    this.applyAccessibilityStyles();
  }

  /**
   * Create screen reader announcer element
   */
  private createScreenReaderAnnouncer(): void {
    if (!this.config.enableScreenReaderAnnouncements) return;

    this.announcer = document.createElement('div');
    this.announcer.setAttribute('aria-live', 'polite');
    this.announcer.setAttribute('aria-atomic', 'true');
    this.announcer.className = 'sr-only';
    this.announcer.style.cssText = `
      position: absolute !important;
      width: 1px !important;
      height: 1px !important;
      padding: 0 !important;
      margin: -1px !important;
      overflow: hidden !important;
      clip: rect(0, 0, 0, 0) !important;
      white-space: nowrap !important;
      border: 0 !important;
    `;

    document.body.appendChild(this.announcer);
  }

  /**
   * Setup keyboard navigation
   */
  private setupKeyboardNavigation(): void {
    if (!this.config.enableKeyboardNavigation) return;

    document.addEventListener('keydown', this.handleKeyboardNavigation);
    document.addEventListener('focusin', this.handleFocusIn);
    document.addEventListener('focusout', this.handleFocusOut);
  }

  /**
   * Setup focus management
   */
  private setupFocusManagement(): void {
    // Detect keyboard navigation
    document.addEventListener('keydown', (event) => {
      if (event.key === 'Tab') {
        this.isKeyboardNavigationActive = true;
        document.body.classList.add('keyboard-navigation-active');
      }
    });

    document.addEventListener('mousedown', () => {
      this.isKeyboardNavigationActive = false;
      document.body.classList.remove('keyboard-navigation-active');
    });
  }

  /**
   * Apply accessibility styles
   */
  private applyAccessibilityStyles(): void {
    if (this.config.enableHighContrast) {
      document.body.classList.add('high-contrast-mode');
    }
  }

  /**
   * Handle keyboard navigation
   */
  private handleKeyboardNavigation = (event: KeyboardEvent): void => {
    if (!this.config.enableKeyboardNavigation) return;

    // Don't interfere with keyboard shortcuts in input fields
    const target = event.target as HTMLElement;
    if (target && (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.contentEditable === 'true'
    )) {
      return;
    }

    switch (event.key) {
      case 'Tab':
        this.handleTabNavigation(event);
        break;
      case 'ArrowUp':
      case 'ArrowDown':
      case 'ArrowLeft':
      case 'ArrowRight':
        this.handleArrowNavigation(event);
        break;
      case 'Enter':
      case ' ':
        this.handleActivation(event);
        break;
      case 'Escape':
        this.handleEscape(event);
        break;
    }
  };

  /**
   * Handle tab navigation
   */
  private handleTabNavigation(event: KeyboardEvent): void {
    if (this.config.enableFocusTrapping && this.focusableElements.length > 0) {
      const currentIndex = this.focusableElements.indexOf(document.activeElement as HTMLElement);
      let nextIndex;

      if (event.shiftKey) {
        nextIndex = currentIndex <= 0 ? this.focusableElements.length - 1 : currentIndex - 1;
      } else {
        nextIndex = currentIndex >= this.focusableElements.length - 1 ? 0 : currentIndex + 1;
      }

      event.preventDefault();
      this.focusableElements[nextIndex]?.focus();
    }
  }

  /**
   * Handle arrow key navigation
   */
  private handleArrowNavigation(event: KeyboardEvent): void {
    const target = event.target as HTMLElement;
    if (!target || !target.closest('.draggable-component')) return;

    // Only handle arrow navigation for dashboard components
    const component = target.closest('.draggable-component') as HTMLElement;
    if (!component) return;

    event.preventDefault();
    this.navigateToAdjacentComponent(component, event.key);
  }

  /**
   * Navigate to adjacent component
   */
  private navigateToAdjacentComponent(currentComponent: HTMLElement, direction: string): void {
    const allComponents = Array.from(document.querySelectorAll('.draggable-component')) as HTMLElement[];
    const currentIndex = allComponents.indexOf(currentComponent);
    
    let nextIndex = currentIndex;
    
    switch (direction) {
      case 'ArrowUp':
        nextIndex = Math.max(0, currentIndex - 1);
        break;
      case 'ArrowDown':
        nextIndex = Math.min(allComponents.length - 1, currentIndex + 1);
        break;
      case 'ArrowLeft':
        nextIndex = Math.max(0, currentIndex - 1);
        break;
      case 'ArrowRight':
        nextIndex = Math.min(allComponents.length - 1, currentIndex + 1);
        break;
    }

    if (nextIndex !== currentIndex) {
      allComponents[nextIndex]?.focus();
      this.announceComponentFocus(allComponents[nextIndex]);
    }
  }

  /**
   * Handle activation (Enter/Space)
   */
  private handleActivation(event: KeyboardEvent): void {
    const target = event.target as HTMLElement;
    
    if (target.tagName === 'BUTTON' || target.getAttribute('role') === 'button') {
      event.preventDefault();
      target.click();
    }
  }

  /**
   * Handle escape key
   */
  private handleEscape(event: KeyboardEvent): void {
    // Cancel any ongoing operations
    this.announce('Operation cancelled');
  }

  /**
   * Handle focus in
   */
  private handleFocusIn = (event: FocusEvent): void => {
    this.focusedElement = event.target as HTMLElement;
  };

  /**
   * Handle focus out
   */
  private handleFocusOut = (event: FocusEvent): void => {
    // Focus management logic
  };

  /**
   * Announce message to screen readers
   */
  announce(message: string, priority: 'polite' | 'assertive' = 'polite'): void {
    if (!this.config.enableScreenReaderAnnouncements || !this.announcer) return;

    this.announcer.setAttribute('aria-live', priority);
    this.announcer.textContent = message;

    // Clear after announcement
    setTimeout(() => {
      if (this.announcer) {
        this.announcer.textContent = '';
      }
    }, 1000);
  }

  /**
   * Set ARIA attributes on element
   */
  setAriaAttributes(element: HTMLElement, attributes: AriaAttributes): void {
    Object.entries(attributes).forEach(([key, value]) => {
      if (value !== undefined) {
        if (typeof value === 'boolean') {
          element.setAttribute(`aria-${key}`, value.toString());
        } else {
          element.setAttribute(`aria-${key}`, value);
        }
      }
    });
  }

  /**
   * Make element focusable
   */
  makeFocusable(element: HTMLElement, tabIndex = 0): void {
    element.setAttribute('tabindex', tabIndex.toString());
    
    if (!element.getAttribute('role')) {
      element.setAttribute('role', 'button');
    }
  }

  /**
   * Remove focusability from element
   */
  removeFocusable(element: HTMLElement): void {
    element.removeAttribute('tabindex');
  }

  /**
   * Set up component accessibility
   */
  setupComponentAccessibility(
    element: HTMLElement,
    componentType: string,
    componentId: string,
    hasData: boolean
  ): void {
    // Make component focusable
    this.makeFocusable(element);

    // Set ARIA attributes
    this.setAriaAttributes(element, {
      role: 'region',
      label: `${componentType} component ${componentId}`,
      describedby: `${componentId}-description`,
    });

    // Add component-specific attributes
    switch (componentType) {
      case 'metrics':
        this.setAriaAttributes(element, {
          role: 'img',
          label: hasData ? 'Metrics chart with data' : 'Empty metrics chart',
        });
        break;
      case 'table':
        this.setAriaAttributes(element, {
          role: 'table',
          label: hasData ? 'Data table with content' : 'Empty data table',
        });
        break;
      case 'image':
        this.setAriaAttributes(element, {
          role: 'img',
          label: hasData ? 'Image component with content' : 'Empty image component',
        });
        break;
      case 'insights':
        this.setAriaAttributes(element, {
          role: 'article',
          label: hasData ? 'Insights with content' : 'Empty insights component',
        });
        break;
      case 'file_download':
        this.setAriaAttributes(element, {
          role: 'region',
          label: hasData ? 'File download area with files' : 'Empty file download area',
        });
        break;
    }

    // Add keyboard event handlers
    element.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        this.announceComponentDetails(element, componentType, hasData);
      }
    });
  }

  /**
   * Setup drag handle accessibility
   */
  setupDragHandleAccessibility(handle: HTMLElement, componentId: string): void {
    this.makeFocusable(handle);
    
    this.setAriaAttributes(handle, {
      role: 'button',
      label: `Drag handle for ${componentId}. Use arrow keys to move, Enter to start drag mode`,
      describedby: `${componentId}-drag-instructions`,
    });

    // Add instructions element
    const instructions = document.createElement('div');
    instructions.id = `${componentId}-drag-instructions`;
    instructions.className = 'sr-only';
    instructions.textContent = 'Use Shift + arrow keys to move component. Press Escape to cancel.';
    handle.appendChild(instructions);
  }

  /**
   * Setup resize handle accessibility
   */
  setupResizeHandleAccessibility(handle: HTMLElement, componentId: string, direction: string): void {
    this.makeFocusable(handle);
    
    this.setAriaAttributes(handle, {
      role: 'button',
      label: `Resize handle ${direction} for ${componentId}. Use Ctrl+Shift+arrow keys to resize`,
    });
  }

  /**
   * Setup pagination accessibility
   */
  setupPaginationAccessibility(
    container: HTMLElement,
    componentType: string,
    currentPage: number,
    totalPages: number
  ): void {
    this.setAriaAttributes(container, {
      role: 'navigation',
      label: `${componentType} pagination`,
    });

    // Announce page changes
    this.announce(`Page ${currentPage} of ${totalPages} for ${componentType}`);
  }

  /**
   * Announce layout changes
   */
  announceLayoutChange(changeType: string, details?: string): void {
    if (!this.config.announceLayoutChanges) return;

    let message = '';
    switch (changeType) {
      case 'component-moved':
        message = `Component moved. ${details || ''}`;
        break;
      case 'component-resized':
        message = `Component resized. ${details || ''}`;
        break;
      case 'component-added':
        message = `Component added. ${details || ''}`;
        break;
      case 'component-removed':
        message = `Component removed. ${details || ''}`;
        break;
      case 'layout-compacted':
        message = 'Layout compacted';
        break;
      case 'mode-changed':
        message = `Mode changed to ${details || ''}`;
        break;
      default:
        message = `Layout changed: ${changeType}`;
    }

    this.announce(message);
  }

  /**
   * Announce component focus
   */
  private announceComponentFocus(component: HTMLElement): void {
    const componentType = component.getAttribute('data-component-type') || 'component';
    const componentId = component.getAttribute('data-component-id') || '';
    const hasData = component.getAttribute('data-has-data') === 'true';

    const status = hasData ? 'with data' : 'empty';
    this.announce(`Focused on ${componentType} ${componentId} ${status}`);
  }

  /**
   * Announce component details
   */
  private announceComponentDetails(element: HTMLElement, componentType: string, hasData: boolean): void {
    const status = hasData ? 'contains data' : 'is empty';
    const position = this.getComponentPosition(element);
    
    this.announce(`${componentType} component ${status}. ${position}`);
  }

  /**
   * Get component position description
   */
  private getComponentPosition(element: HTMLElement): string {
    const rect = element.getBoundingClientRect();
    const container = element.closest('.dashboard-container');
    
    if (!container) return '';

    const containerRect = container.getBoundingClientRect();
    const relativeX = Math.round((rect.left - containerRect.left) / containerRect.width * 100);
    const relativeY = Math.round((rect.top - containerRect.top) / containerRect.height * 100);

    return `Located at ${relativeX}% from left, ${relativeY}% from top`;
  }

  /**
   * Set focusable elements for focus trapping
   */
  setFocusableElements(elements: HTMLElement[]): void {
    this.focusableElements = elements;
  }

  /**
   * Enable/disable focus trapping
   */
  setFocusTrapping(enabled: boolean): void {
    this.config.enableFocusTrapping = enabled;
  }

  /**
   * Enable/disable high contrast mode
   */
  setHighContrastMode(enabled: boolean): void {
    this.config.enableHighContrast = enabled;
    
    if (enabled) {
      document.body.classList.add('high-contrast-mode');
    } else {
      document.body.classList.remove('high-contrast-mode');
    }
  }

  /**
   * Get current accessibility config
   */
  getConfig(): AccessibilityConfig {
    return { ...this.config };
  }

  /**
   * Update accessibility config
   */
  updateConfig(config: Partial<AccessibilityConfig>): void {
    this.config = { ...this.config, ...config };
  }

  /**
   * Cleanup accessibility features
   */
  cleanup(): void {
    if (this.announcer) {
      this.announcer.remove();
      this.announcer = null;
    }

    document.removeEventListener('keydown', this.handleKeyboardNavigation);
    document.removeEventListener('focusin', this.handleFocusIn);
    document.removeEventListener('focusout', this.handleFocusOut);

    document.body.classList.remove('keyboard-navigation-active', 'high-contrast-mode');
  }
}

/**
 * Global accessibility manager instance
 */
let globalAccessibilityManager: AccessibilityManager | null = null;

/**
 * Get or create global accessibility manager
 */
export function getAccessibilityManager(config?: Partial<AccessibilityConfig>): AccessibilityManager {
  if (!globalAccessibilityManager) {
    globalAccessibilityManager = new AccessibilityManager(config);
  }
  return globalAccessibilityManager;
}

/**
 * React hook for accessibility
 */
export function useAccessibility(config?: Partial<AccessibilityConfig>) {
  const manager = getAccessibilityManager(config);

  const announce = React.useCallback((message: string, priority?: 'polite' | 'assertive') => {
    manager.announce(message, priority);
  }, [manager]);

  const setAriaAttributes = React.useCallback((element: HTMLElement, attributes: AriaAttributes) => {
    manager.setAriaAttributes(element, attributes);
  }, [manager]);

  const makeFocusable = React.useCallback((element: HTMLElement, tabIndex?: number) => {
    manager.makeFocusable(element, tabIndex);
  }, [manager]);

  const setupComponentAccessibility = React.useCallback((
    element: HTMLElement,
    componentType: string,
    componentId: string,
    hasData: boolean
  ) => {
    manager.setupComponentAccessibility(element, componentType, componentId, hasData);
  }, [manager]);

  const announceLayoutChange = React.useCallback((changeType: string, details?: string) => {
    manager.announceLayoutChange(changeType, details);
  }, [manager]);

  const setHighContrastMode = React.useCallback((enabled: boolean) => {
    manager.setHighContrastMode(enabled);
  }, [manager]);

  return {
    announce,
    setAriaAttributes,
    makeFocusable,
    setupComponentAccessibility,
    announceLayoutChange,
    setHighContrastMode,
    config: manager.getConfig(),
  };
}

// Add React import for hooks
import React from 'react';