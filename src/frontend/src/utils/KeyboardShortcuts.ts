/**
 * Keyboard Shortcuts System for Dashboard Drag-Drop Layout
 * 
 * Provides keyboard shortcuts for common dashboard operations
 */

export interface KeyboardShortcut {
  id: string;
  key: string;
  modifiers: KeyModifier[];
  description: string;
  action: () => void | Promise<void>;
  enabled: boolean;
  context?: string; // Optional context where shortcut is active
}

export enum KeyModifier {
  CTRL = 'ctrl',
  ALT = 'alt',
  SHIFT = 'shift',
  META = 'meta', // Cmd on Mac, Windows key on PC
}

export interface KeyboardEvent {
  key: string;
  ctrlKey: boolean;
  altKey: boolean;
  shiftKey: boolean;
  metaKey: boolean;
  preventDefault: () => void;
  stopPropagation: () => void;
}

/**
 * Keyboard shortcuts manager
 */
export class KeyboardShortcutsManager {
  private shortcuts: Map<string, KeyboardShortcut> = new Map();
  private isEnabled = true;
  private currentContext: string | null = null;
  private listeners: Set<(shortcuts: KeyboardShortcut[]) => void> = new Set();

  constructor() {
    this.initializeDefaultShortcuts();
    this.bindEventListeners();
  }

  /**
   * Initialize default dashboard shortcuts
   */
  private initializeDefaultShortcuts(): void {
    // Layout operations
    this.addShortcut({
      id: 'toggle-edit-mode',
      key: 'e',
      modifiers: [KeyModifier.CTRL],
      description: 'Toggle edit mode',
      action: () => this.triggerAction('toggle-edit-mode'),
      enabled: true,
    });

    this.addShortcut({
      id: 'toggle-lock',
      key: 'l',
      modifiers: [KeyModifier.CTRL],
      description: 'Toggle layout lock',
      action: () => this.triggerAction('toggle-lock'),
      enabled: true,
    });

    this.addShortcut({
      id: 'save-layout',
      key: 's',
      modifiers: [KeyModifier.CTRL],
      description: 'Save layout',
      action: () => this.triggerAction('save-layout'),
      enabled: true,
    });

    this.addShortcut({
      id: 'compact-layout',
      key: 'k',
      modifiers: [KeyModifier.CTRL],
      description: 'Compact layout',
      action: () => this.triggerAction('compact-layout'),
      enabled: true,
    });

    // Component operations
    this.addShortcut({
      id: 'add-metrics',
      key: '1',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Add metrics component',
      action: () => this.triggerAction('add-component', 'metrics'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'add-table',
      key: '2',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Add table component',
      action: () => this.triggerAction('add-component', 'table'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'add-image',
      key: '3',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Add image component',
      action: () => this.triggerAction('add-component', 'image'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'add-insights',
      key: '4',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Add insights component',
      action: () => this.triggerAction('add-component', 'insights'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'add-files',
      key: '5',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Add file download component',
      action: () => this.triggerAction('add-component', 'file_download'),
      enabled: true,
      context: 'edit-mode',
    });

    // Navigation
    this.addShortcut({
      id: 'focus-next-component',
      key: 'Tab',
      modifiers: [],
      description: 'Focus next component',
      action: () => this.triggerAction('focus-next-component'),
      enabled: true,
    });

    this.addShortcut({
      id: 'focus-previous-component',
      key: 'Tab',
      modifiers: [KeyModifier.SHIFT],
      description: 'Focus previous component',
      action: () => this.triggerAction('focus-previous-component'),
      enabled: true,
    });

    // Pagination
    this.addShortcut({
      id: 'next-page',
      key: 'ArrowRight',
      modifiers: [KeyModifier.CTRL],
      description: 'Next page',
      action: () => this.triggerAction('next-page'),
      enabled: true,
    });

    this.addShortcut({
      id: 'previous-page',
      key: 'ArrowLeft',
      modifiers: [KeyModifier.CTRL],
      description: 'Previous page',
      action: () => this.triggerAction('previous-page'),
      enabled: true,
    });

    // Component movement (when focused)
    this.addShortcut({
      id: 'move-component-up',
      key: 'ArrowUp',
      modifiers: [KeyModifier.SHIFT],
      description: 'Move focused component up',
      action: () => this.triggerAction('move-component', 'up'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'move-component-down',
      key: 'ArrowDown',
      modifiers: [KeyModifier.SHIFT],
      description: 'Move focused component down',
      action: () => this.triggerAction('move-component', 'down'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'move-component-left',
      key: 'ArrowLeft',
      modifiers: [KeyModifier.SHIFT],
      description: 'Move focused component left',
      action: () => this.triggerAction('move-component', 'left'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'move-component-right',
      key: 'ArrowRight',
      modifiers: [KeyModifier.SHIFT],
      description: 'Move focused component right',
      action: () => this.triggerAction('move-component', 'right'),
      enabled: true,
      context: 'edit-mode',
    });

    // Component resizing (when focused)
    this.addShortcut({
      id: 'resize-component-wider',
      key: 'ArrowRight',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Make focused component wider',
      action: () => this.triggerAction('resize-component', 'wider'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'resize-component-narrower',
      key: 'ArrowLeft',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Make focused component narrower',
      action: () => this.triggerAction('resize-component', 'narrower'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'resize-component-taller',
      key: 'ArrowDown',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Make focused component taller',
      action: () => this.triggerAction('resize-component', 'taller'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'resize-component-shorter',
      key: 'ArrowUp',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Make focused component shorter',
      action: () => this.triggerAction('resize-component', 'shorter'),
      enabled: true,
      context: 'edit-mode',
    });

    // Delete component
    this.addShortcut({
      id: 'delete-component',
      key: 'Delete',
      modifiers: [],
      description: 'Delete focused component',
      action: () => this.triggerAction('delete-component'),
      enabled: true,
      context: 'edit-mode',
    });

    this.addShortcut({
      id: 'delete-component-backspace',
      key: 'Backspace',
      modifiers: [],
      description: 'Delete focused component',
      action: () => this.triggerAction('delete-component'),
      enabled: true,
      context: 'edit-mode',
    });

    // Export
    this.addShortcut({
      id: 'export-dashboard',
      key: 'e',
      modifiers: [KeyModifier.CTRL, KeyModifier.SHIFT],
      description: 'Export dashboard',
      action: () => this.triggerAction('export-dashboard'),
      enabled: true,
    });

    // Help
    this.addShortcut({
      id: 'show-help',
      key: '?',
      modifiers: [KeyModifier.SHIFT],
      description: 'Show keyboard shortcuts help',
      action: () => this.triggerAction('show-help'),
      enabled: true,
    });

    // Escape to cancel operations
    this.addShortcut({
      id: 'cancel-operation',
      key: 'Escape',
      modifiers: [],
      description: 'Cancel current operation',
      action: () => this.triggerAction('cancel-operation'),
      enabled: true,
    });
  }

  /**
   * Bind keyboard event listeners
   */
  private bindEventListeners(): void {
    if (typeof window !== 'undefined') {
      document.addEventListener('keydown', this.handleKeyDown);
    }
  }

  /**
   * Handle keydown events
   */
  private handleKeyDown = (event: KeyboardEvent): void => {
    if (!this.isEnabled) return;

    // Don't trigger shortcuts when typing in inputs
    const target = event.target as HTMLElement;
    if (target && (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.contentEditable === 'true'
    )) {
      return;
    }

    const matchingShortcut = this.findMatchingShortcut(event);
    if (matchingShortcut && this.isShortcutEnabled(matchingShortcut)) {
      event.preventDefault();
      event.stopPropagation();
      matchingShortcut.action();
    }
  };

  /**
   * Find matching shortcut for keyboard event
   */
  private findMatchingShortcut(event: KeyboardEvent): KeyboardShortcut | null {
    for (const shortcut of this.shortcuts.values()) {
      if (this.isEventMatchingShortcut(event, shortcut)) {
        return shortcut;
      }
    }
    return null;
  }

  /**
   * Check if keyboard event matches shortcut
   */
  private isEventMatchingShortcut(event: KeyboardEvent, shortcut: KeyboardShortcut): boolean {
    // Check key
    if (event.key !== shortcut.key) {
      return false;
    }

    // Check modifiers
    const hasCtrl = shortcut.modifiers.includes(KeyModifier.CTRL);
    const hasAlt = shortcut.modifiers.includes(KeyModifier.ALT);
    const hasShift = shortcut.modifiers.includes(KeyModifier.SHIFT);
    const hasMeta = shortcut.modifiers.includes(KeyModifier.META);

    return (
      event.ctrlKey === hasCtrl &&
      event.altKey === hasAlt &&
      event.shiftKey === hasShift &&
      event.metaKey === hasMeta
    );
  }

  /**
   * Check if shortcut is enabled in current context
   */
  private isShortcutEnabled(shortcut: KeyboardShortcut): boolean {
    if (!shortcut.enabled) {
      return false;
    }

    if (shortcut.context && shortcut.context !== this.currentContext) {
      return false;
    }

    return true;
  }

  /**
   * Add a keyboard shortcut
   */
  addShortcut(shortcut: KeyboardShortcut): void {
    this.shortcuts.set(shortcut.id, shortcut);
    this.notifyListeners();
  }

  /**
   * Remove a keyboard shortcut
   */
  removeShortcut(id: string): void {
    this.shortcuts.delete(id);
    this.notifyListeners();
  }

  /**
   * Enable/disable a shortcut
   */
  setShortcutEnabled(id: string, enabled: boolean): void {
    const shortcut = this.shortcuts.get(id);
    if (shortcut) {
      shortcut.enabled = enabled;
      this.notifyListeners();
    }
  }

  /**
   * Enable/disable all shortcuts
   */
  setEnabled(enabled: boolean): void {
    this.isEnabled = enabled;
  }

  /**
   * Set current context
   */
  setContext(context: string | null): void {
    this.currentContext = context;
  }

  /**
   * Get all shortcuts
   */
  getAllShortcuts(): KeyboardShortcut[] {
    return Array.from(this.shortcuts.values());
  }

  /**
   * Get shortcuts for current context
   */
  getContextShortcuts(): KeyboardShortcut[] {
    return this.getAllShortcuts().filter(shortcut => 
      !shortcut.context || shortcut.context === this.currentContext
    );
  }

  /**
   * Get shortcut by ID
   */
  getShortcut(id: string): KeyboardShortcut | null {
    return this.shortcuts.get(id) || null;
  }

  /**
   * Format shortcut for display
   */
  formatShortcut(shortcut: KeyboardShortcut): string {
    const modifiers = shortcut.modifiers.map(mod => {
      switch (mod) {
        case KeyModifier.CTRL: return 'Ctrl';
        case KeyModifier.ALT: return 'Alt';
        case KeyModifier.SHIFT: return 'Shift';
        case KeyModifier.META: return navigator.platform.includes('Mac') ? 'Cmd' : 'Win';
        default: return mod;
      }
    });

    const key = shortcut.key === ' ' ? 'Space' : shortcut.key;
    
    return [...modifiers, key].join(' + ');
  }

  /**
   * Subscribe to shortcut changes
   */
  subscribe(callback: (shortcuts: KeyboardShortcut[]) => void): () => void {
    this.listeners.add(callback);
    
    // Return unsubscribe function
    return () => {
      this.listeners.delete(callback);
    };
  }

  /**
   * Notify listeners of shortcut changes
   */
  private notifyListeners(): void {
    const shortcuts = this.getAllShortcuts();
    this.listeners.forEach(callback => callback(shortcuts));
  }

  /**
   * Trigger action (to be overridden by implementation)
   */
  private triggerAction(action: string, ...args: any[]): void {
    // This would be implemented by the dashboard component
    // Action triggered: action with args
  }

  /**
   * Set action handler
   */
  setActionHandler(handler: (action: string, ...args: any[]) => void): void {
    this.triggerAction = handler;
  }

  /**
   * Cleanup
   */
  cleanup(): void {
    if (typeof window !== 'undefined') {
      document.removeEventListener('keydown', this.handleKeyDown);
    }
    this.listeners.clear();
  }
}

/**
 * Global keyboard shortcuts manager instance
 */
let globalKeyboardManager: KeyboardShortcutsManager | null = null;

/**
 * Get or create global keyboard shortcuts manager
 */
export function getKeyboardShortcutsManager(): KeyboardShortcutsManager {
  if (!globalKeyboardManager) {
    globalKeyboardManager = new KeyboardShortcutsManager();
  }
  return globalKeyboardManager;
}

/**
 * React hook for keyboard shortcuts
 */
export function useKeyboardShortcuts() {
  const manager = getKeyboardShortcutsManager();
  const [shortcuts, setShortcuts] = React.useState<KeyboardShortcut[]>(
    manager.getAllShortcuts()
  );

  React.useEffect(() => {
    const unsubscribe = manager.subscribe(setShortcuts);
    return unsubscribe;
  }, [manager]);

  const addShortcut = React.useCallback((shortcut: KeyboardShortcut) => {
    manager.addShortcut(shortcut);
  }, [manager]);

  const removeShortcut = React.useCallback((id: string) => {
    manager.removeShortcut(id);
  }, [manager]);

  const setShortcutEnabled = React.useCallback((id: string, enabled: boolean) => {
    manager.setShortcutEnabled(id, enabled);
  }, [manager]);

  const setEnabled = React.useCallback((enabled: boolean) => {
    manager.setEnabled(enabled);
  }, [manager]);

  const setContext = React.useCallback((context: string | null) => {
    manager.setContext(context);
  }, [manager]);

  const setActionHandler = React.useCallback((handler: (action: string, ...args: any[]) => void) => {
    manager.setActionHandler(handler);
  }, [manager]);

  const formatShortcut = React.useCallback((shortcut: KeyboardShortcut) => {
    return manager.formatShortcut(shortcut);
  }, [manager]);

  return {
    shortcuts,
    addShortcut,
    removeShortcut,
    setShortcutEnabled,
    setEnabled,
    setContext,
    setActionHandler,
    formatShortcut,
    getContextShortcuts: () => manager.getContextShortcuts(),
  };
}

// Add React import for hooks
import React from 'react';