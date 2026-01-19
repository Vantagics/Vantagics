/**
 * Loading States Manager for Dashboard Components
 * 
 * Manages loading states for data fetching operations across dashboard components
 */

export interface LoadingState {
  isLoading: boolean;
  error: string | null;
  lastUpdated: number | null;
}

export interface ComponentLoadingState extends LoadingState {
  componentId: string;
  componentType: string;
  hasData: boolean;
}

/**
 * Loading states manager
 */
export class LoadingStatesManager {
  private states: Map<string, ComponentLoadingState> = new Map();
  private listeners: Map<string, Set<(state: ComponentLoadingState) => void>> = new Map();
  private globalListeners: Set<(states: Map<string, ComponentLoadingState>) => void> = new Set();

  /**
   * Set loading state for a component
   */
  setLoading(componentId: string, componentType: string, isLoading: boolean): void {
    const currentState = this.states.get(componentId) || {
      componentId,
      componentType,
      isLoading: false,
      error: null,
      lastUpdated: null,
      hasData: false,
    };

    const newState: ComponentLoadingState = {
      ...currentState,
      isLoading,
      error: isLoading ? null : currentState.error, // Clear error when starting to load
    };

    this.states.set(componentId, newState);
    this.notifyListeners(componentId, newState);
    this.notifyGlobalListeners();
  }

  /**
   * Set error state for a component
   */
  setError(componentId: string, error: string): void {
    const currentState = this.states.get(componentId);
    if (!currentState) return;

    const newState: ComponentLoadingState = {
      ...currentState,
      isLoading: false,
      error,
      lastUpdated: Date.now(),
    };

    this.states.set(componentId, newState);
    this.notifyListeners(componentId, newState);
    this.notifyGlobalListeners();
  }

  /**
   * Set success state for a component
   */
  setSuccess(componentId: string, hasData: boolean): void {
    const currentState = this.states.get(componentId);
    if (!currentState) return;

    const newState: ComponentLoadingState = {
      ...currentState,
      isLoading: false,
      error: null,
      hasData,
      lastUpdated: Date.now(),
    };

    this.states.set(componentId, newState);
    this.notifyListeners(componentId, newState);
    this.notifyGlobalListeners();
  }

  /**
   * Get loading state for a component
   */
  getState(componentId: string): ComponentLoadingState | null {
    return this.states.get(componentId) || null;
  }

  /**
   * Get all loading states
   */
  getAllStates(): Map<string, ComponentLoadingState> {
    return new Map(this.states);
  }

  /**
   * Check if any component is loading
   */
  isAnyLoading(): boolean {
    return Array.from(this.states.values()).some(state => state.isLoading);
  }

  /**
   * Check if specific component type is loading
   */
  isTypeLoading(componentType: string): boolean {
    return Array.from(this.states.values()).some(
      state => state.componentType === componentType && state.isLoading
    );
  }

  /**
   * Get loading states by component type
   */
  getStatesByType(componentType: string): ComponentLoadingState[] {
    return Array.from(this.states.values()).filter(
      state => state.componentType === componentType
    );
  }

  /**
   * Clear state for a component
   */
  clearState(componentId: string): void {
    this.states.delete(componentId);
    this.notifyGlobalListeners();
  }

  /**
   * Clear all states
   */
  clearAllStates(): void {
    this.states.clear();
    this.notifyGlobalListeners();
  }

  /**
   * Subscribe to state changes for a specific component
   */
  subscribe(componentId: string, callback: (state: ComponentLoadingState) => void): () => void {
    if (!this.listeners.has(componentId)) {
      this.listeners.set(componentId, new Set());
    }
    
    this.listeners.get(componentId)!.add(callback);

    // Return unsubscribe function
    return () => {
      const componentListeners = this.listeners.get(componentId);
      if (componentListeners) {
        componentListeners.delete(callback);
        if (componentListeners.size === 0) {
          this.listeners.delete(componentId);
        }
      }
    };
  }

  /**
   * Subscribe to global state changes
   */
  subscribeGlobal(callback: (states: Map<string, ComponentLoadingState>) => void): () => void {
    this.globalListeners.add(callback);

    // Return unsubscribe function
    return () => {
      this.globalListeners.delete(callback);
    };
  }

  /**
   * Notify listeners for a specific component
   */
  private notifyListeners(componentId: string, state: ComponentLoadingState): void {
    const componentListeners = this.listeners.get(componentId);
    if (componentListeners) {
      componentListeners.forEach(callback => callback(state));
    }
  }

  /**
   * Notify global listeners
   */
  private notifyGlobalListeners(): void {
    this.globalListeners.forEach(callback => callback(this.getAllStates()));
  }

  /**
   * Create a loading operation wrapper
   */
  async withLoading<T>(
    componentId: string,
    componentType: string,
    operation: () => Promise<T>
  ): Promise<T> {
    this.setLoading(componentId, componentType, true);

    try {
      const result = await operation();
      this.setSuccess(componentId, true);
      return result;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      this.setError(componentId, errorMessage);
      throw error;
    }
  }

  /**
   * Batch update multiple component states
   */
  batchUpdate(updates: Array<{
    componentId: string;
    componentType: string;
    isLoading?: boolean;
    error?: string | null;
    hasData?: boolean;
  }>): void {
    updates.forEach(update => {
      const currentState = this.states.get(update.componentId) || {
        componentId: update.componentId,
        componentType: update.componentType,
        isLoading: false,
        error: null,
        lastUpdated: null,
        hasData: false,
      };

      const newState: ComponentLoadingState = {
        ...currentState,
        ...update,
        lastUpdated: Date.now(),
      };

      this.states.set(update.componentId, newState);
      this.notifyListeners(update.componentId, newState);
    });

    this.notifyGlobalListeners();
  }

  /**
   * Get loading statistics
   */
  getStats(): {
    total: number;
    loading: number;
    error: number;
    success: number;
    byType: Record<string, { total: number; loading: number; error: number; success: number }>;
  } {
    const states = Array.from(this.states.values());
    const byType: Record<string, { total: number; loading: number; error: number; success: number }> = {};

    states.forEach(state => {
      if (!byType[state.componentType]) {
        byType[state.componentType] = { total: 0, loading: 0, error: 0, success: 0 };
      }

      byType[state.componentType].total++;

      if (state.isLoading) {
        byType[state.componentType].loading++;
      } else if (state.error) {
        byType[state.componentType].error++;
      } else {
        byType[state.componentType].success++;
      }
    });

    return {
      total: states.length,
      loading: states.filter(s => s.isLoading).length,
      error: states.filter(s => s.error !== null).length,
      success: states.filter(s => !s.isLoading && !s.error).length,
      byType,
    };
  }
}

/**
 * Global loading states manager instance
 */
let globalLoadingStatesManager: LoadingStatesManager | null = null;

/**
 * Get or create global loading states manager
 */
export function getLoadingStatesManager(): LoadingStatesManager {
  if (!globalLoadingStatesManager) {
    globalLoadingStatesManager = new LoadingStatesManager();
  }
  return globalLoadingStatesManager;
}

/**
 * React hook for component loading state
 */
export function useComponentLoadingState(componentId: string, componentType: string) {
  const manager = getLoadingStatesManager();
  const [state, setState] = React.useState<ComponentLoadingState | null>(
    manager.getState(componentId)
  );

  React.useEffect(() => {
    const unsubscribe = manager.subscribe(componentId, setState);
    return unsubscribe;
  }, [componentId, manager]);

  const setLoading = React.useCallback((isLoading: boolean) => {
    manager.setLoading(componentId, componentType, isLoading);
  }, [componentId, componentType, manager]);

  const setError = React.useCallback((error: string) => {
    manager.setError(componentId, error);
  }, [componentId, manager]);

  const setSuccess = React.useCallback((hasData: boolean) => {
    manager.setSuccess(componentId, hasData);
  }, [componentId, manager]);

  const withLoading = React.useCallback(async <T>(operation: () => Promise<T>): Promise<T> => {
    return manager.withLoading(componentId, componentType, operation);
  }, [componentId, componentType, manager]);

  return {
    state,
    setLoading,
    setError,
    setSuccess,
    withLoading,
  };
}

/**
 * React hook for global loading states
 */
export function useGlobalLoadingStates() {
  const manager = getLoadingStatesManager();
  const [states, setStates] = React.useState<Map<string, ComponentLoadingState>>(
    manager.getAllStates()
  );

  React.useEffect(() => {
    const unsubscribe = manager.subscribeGlobal(setStates);
    return unsubscribe;
  }, [manager]);

  const stats = React.useMemo(() => manager.getStats(), [states, manager]);

  return {
    states,
    stats,
    isAnyLoading: manager.isAnyLoading(),
    isTypeLoading: (type: string) => manager.isTypeLoading(type),
  };
}

// Add React import for hooks
import React from 'react';