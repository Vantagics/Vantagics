/**
 * Component Manager for Dashboard Drag-Drop Layout System
 * 
 * This module manages component instances, registration, pagination state,
 * and data updates for the dashboard layout system.
 */

import { LayoutItem } from './LayoutEngine';

// ============================================================================
// INTERFACES AND TYPES
// ============================================================================

/**
 * Component type enumeration
 */
export enum ComponentType {
  METRICS = 'metrics',
  TABLE = 'table',
  IMAGE = 'image',
  INSIGHTS = 'insights',
  FILE_DOWNLOAD = 'file_download',
}

/**
 * Component instance data
 */
export interface ComponentInstance {
  /** Unique instance ID */
  id: string;
  /** Component type */
  type: ComponentType;
  /** Display title */
  title: string;
  /** Layout information */
  layout: LayoutItem;
  /** Whether component has data */
  hasData: boolean;
  /** Component-specific data */
  data?: any;
  /** Pagination state */
  pagination?: PaginationState;
  /** Component configuration */
  config?: ComponentConfig;
}

/**
 * Pagination state for component groups
 */
export interface PaginationState {
  /** Current page index (0-based) */
  currentPage: number;
  /** Total number of pages */
  totalPages: number;
  /** Number of items per page */
  itemsPerPage: number;
  /** Total number of items */
  totalItems: number;
}

/**
 * Component configuration
 */
export interface ComponentConfig {
  /** Minimum dimensions */
  minSize?: { w: number; h: number };
  /** Maximum dimensions */
  maxSize?: { w: number; h: number };
  /** Default dimensions */
  defaultSize: { w: number; h: number };
  /** Whether component supports pagination */
  supportsPagination: boolean;
  /** Component-specific settings */
  settings?: Record<string, any>;
}

/**
 * Component registry entry
 */
export interface ComponentRegistryEntry {
  /** Component type */
  type: ComponentType;
  /** Display name */
  displayName: string;
  /** Component configuration */
  config: ComponentConfig;
  /** Factory function to create component instances */
  factory: (id: string, layout: LayoutItem) => ComponentInstance;
}

/**
 * Component data update event
 */
export interface ComponentDataUpdate {
  /** Component instance ID */
  instanceId: string;
  /** Updated data */
  data: any;
  /** Whether component has data */
  hasData: boolean;
  /** Timestamp of update */
  timestamp: number;
}

// ============================================================================
// DEFAULT CONFIGURATIONS
// ============================================================================

/**
 * Default component configurations
 */
export const DEFAULT_COMPONENT_CONFIGS: Record<ComponentType, ComponentConfig> = {
  [ComponentType.METRICS]: {
    defaultSize: { w: 6, h: 12 },  // 高度从 4 增加到 12 (3倍)
    minSize: { w: 3, h: 6 },       // 最小高度从 2 增加到 6 (3倍)
    maxSize: { w: 12, h: 24 },     // 最大高度从 8 增加到 24 (3倍)
    supportsPagination: true,
  },
  [ComponentType.TABLE]: {
    defaultSize: { w: 12, h: 18 }, // 高度从 6 增加到 18 (3倍)
    minSize: { w: 6, h: 12 },      // 最小高度从 4 增加到 12 (3倍)
    maxSize: { w: 24, h: 36 },     // 最大高度从 12 增加到 36 (3倍)
    supportsPagination: true,
  },
  [ComponentType.IMAGE]: {
    defaultSize: { w: 8, h: 18 },  // 高度从 6 增加到 18 (3倍)
    minSize: { w: 4, h: 9 },       // 最小高度从 3 增加到 9 (3倍)
    maxSize: { w: 16, h: 36 },     // 最大高度从 12 增加到 36 (3倍)
    supportsPagination: true,
  },
  [ComponentType.INSIGHTS]: {
    defaultSize: { w: 10, h: 24 }, // 高度从 8 增加到 24 (3倍)
    minSize: { w: 6, h: 12 },      // 最小高度从 4 增加到 12 (3倍)
    maxSize: { w: 18, h: 48 },     // 最大高度从 16 增加到 48 (3倍)
    supportsPagination: true,
  },
  [ComponentType.FILE_DOWNLOAD]: {
    defaultSize: { w: 8, h: 30 },  // 高度从 10 增加到 30 (3倍)
    minSize: { w: 6, h: 18 },      // 最小高度从 6 增加到 18 (3倍)
    maxSize: { w: 12, h: 48 },     // 最大高度从 16 增加到 48 (3倍)
    supportsPagination: false,
  },
};

// ============================================================================
// COMPONENT MANAGER CLASS
// ============================================================================

/**
 * Component Manager for handling component instances and registry
 */
export class ComponentManager {
  private registry: Map<ComponentType, ComponentRegistryEntry>;
  private instances: Map<string, ComponentInstance>;
  private paginationStates: Map<ComponentType, PaginationState>;
  private dataUpdateCallbacks: Map<string, (update: ComponentDataUpdate) => void>;

  constructor() {
    this.registry = new Map();
    this.instances = new Map();
    this.paginationStates = new Map();
    this.dataUpdateCallbacks = new Map();
    
    // Initialize with default component types
    this.initializeDefaultComponents();
  }

  // ========================================================================
  // COMPONENT REGISTRY
  // ========================================================================

  /**
   * Registers a component type with the manager
   */
  public registerComponent(entry: ComponentRegistryEntry): void {
    this.registry.set(entry.type, entry);
    
    // Initialize pagination state if component supports pagination
    if (entry.config.supportsPagination) {
      this.paginationStates.set(entry.type, {
        currentPage: 0,
        totalPages: 1,
        itemsPerPage: 1,
        totalItems: 0,
      });
    }
  }

  /**
   * Gets a registered component entry
   */
  public getComponentEntry(type: ComponentType): ComponentRegistryEntry | undefined {
    return this.registry.get(type);
  }

  /**
   * Gets all registered component types
   */
  public getRegisteredTypes(): ComponentType[] {
    return Array.from(this.registry.keys());
  }

  /**
   * Checks if a component type is registered
   */
  public isRegistered(type: ComponentType): boolean {
    return this.registry.has(type);
  }

  // ========================================================================
  // COMPONENT INSTANCE MANAGEMENT
  // ========================================================================

  /**
   * Creates a new component instance
   */
  public createInstance(
    type: ComponentType,
    layout: LayoutItem,
    title?: string
  ): ComponentInstance | null {
    const entry = this.registry.get(type);
    if (!entry) {
      // Component type not registered - return null for graceful handling
      return null;
    }

    // Generate unique ID
    const id = this.generateInstanceId(type);
    
    // Create instance using factory
    const instance = entry.factory(id, layout);
    
    // Set title if provided
    if (title) {
      instance.title = title;
    }

    // Store instance
    this.instances.set(id, instance);

    // Update pagination if needed
    if (entry.config.supportsPagination) {
      this.updatePaginationForType(type);
    }

    return instance;
  }

  /**
   * Gets a component instance by ID
   */
  public getInstance(id: string): ComponentInstance | undefined {
    return this.instances.get(id);
  }

  /**
   * Gets all instances of a specific type
   */
  public getInstancesByType(type: ComponentType): ComponentInstance[] {
    return Array.from(this.instances.values()).filter(
      instance => instance.type === type
    );
  }

  /**
   * Gets all component instances
   */
  public getAllInstances(): ComponentInstance[] {
    return Array.from(this.instances.values());
  }

  /**
   * Removes a component instance
   */
  public removeInstance(id: string): boolean {
    const instance = this.instances.get(id);
    if (!instance) {
      return false;
    }

    this.instances.delete(id);
    
    // Update pagination if needed
    const entry = this.registry.get(instance.type);
    if (entry?.config.supportsPagination) {
      this.updatePaginationForType(instance.type);
    }

    return true;
  }

  /**
   * Updates an instance's layout
   */
  public updateInstanceLayout(id: string, layout: LayoutItem): boolean {
    const instance = this.instances.get(id);
    if (!instance) {
      return false;
    }

    instance.layout = { ...layout };
    return true;
  }

  // ========================================================================
  // PAGINATION MANAGEMENT
  // ========================================================================

  /**
   * Gets pagination state for a component type
   */
  public getPaginationState(type: ComponentType): PaginationState | null {
    const entry = this.registry.get(type);
    if (!entry?.config.supportsPagination) {
      return null;
    }

    return this.paginationStates.get(type) || null;
  }

  /**
   * Updates pagination state for a component type
   */
  public updatePaginationState(
    type: ComponentType,
    updates: Partial<PaginationState>
  ): boolean {
    const entry = this.registry.get(type);
    if (!entry?.config.supportsPagination) {
      return false;
    }

    const currentState = this.paginationStates.get(type);
    if (!currentState) {
      return false;
    }

    const newState = { ...currentState, ...updates };
    
    // Validate pagination state
    newState.currentPage = Math.max(0, Math.min(newState.currentPage, newState.totalPages - 1));
    newState.totalPages = Math.max(1, newState.totalPages);
    newState.itemsPerPage = Math.max(1, newState.itemsPerPage);
    newState.totalItems = Math.max(0, newState.totalItems);

    this.paginationStates.set(type, newState);
    return true;
  }

  /**
   * Navigates to next page for a component type
   */
  public nextPage(type: ComponentType): boolean {
    const state = this.paginationStates.get(type);
    if (!state || state.currentPage >= state.totalPages - 1) {
      return false;
    }

    return this.updatePaginationState(type, {
      currentPage: state.currentPage + 1,
    });
  }

  /**
   * Navigates to previous page for a component type
   */
  public previousPage(type: ComponentType): boolean {
    const state = this.paginationStates.get(type);
    if (!state || state.currentPage <= 0) {
      return false;
    }

    return this.updatePaginationState(type, {
      currentPage: state.currentPage - 1,
    });
  }

  /**
   * Gets the currently visible instance for a paginated component type
   */
  public getCurrentPageInstance(type: ComponentType): ComponentInstance | null {
    const state = this.paginationStates.get(type);
    if (!state) {
      return null;
    }

    const instances = this.getInstancesByType(type);
    if (instances.length === 0) {
      return null;
    }

    const index = state.currentPage;
    return instances[index] || null;
  }

  // ========================================================================
  // DATA MANAGEMENT
  // ========================================================================

  /**
   * Updates component data and triggers callbacks
   */
  public updateComponentData(
    instanceId: string,
    data: any,
    hasData: boolean = true
  ): boolean {
    const instance = this.instances.get(instanceId);
    if (!instance) {
      return false;
    }

    // Update instance data
    instance.data = data;
    instance.hasData = hasData;

    // Create update event
    const update: ComponentDataUpdate = {
      instanceId,
      data,
      hasData,
      timestamp: Date.now(),
    };

    // Trigger callback if registered
    const callback = this.dataUpdateCallbacks.get(instanceId);
    if (callback) {
      callback(update);
    }

    return true;
  }

  /**
   * Registers a callback for data updates on a specific instance
   */
  public onDataUpdate(
    instanceId: string,
    callback: (update: ComponentDataUpdate) => void
  ): void {
    this.dataUpdateCallbacks.set(instanceId, callback);
  }

  /**
   * Unregisters a data update callback
   */
  public offDataUpdate(instanceId: string): void {
    this.dataUpdateCallbacks.delete(instanceId);
  }

  /**
   * Checks if any instances of a type have data
   */
  public typeHasData(type: ComponentType): boolean {
    const instances = this.getInstancesByType(type);
    return instances.some(instance => instance.hasData);
  }

  /**
   * Gets instances that have data
   */
  public getInstancesWithData(): ComponentInstance[] {
    return Array.from(this.instances.values()).filter(
      instance => instance.hasData
    );
  }

  /**
   * Gets instances that don't have data
   */
  public getInstancesWithoutData(): ComponentInstance[] {
    return Array.from(this.instances.values()).filter(
      instance => !instance.hasData
    );
  }

  // ========================================================================
  // UTILITY METHODS
  // ========================================================================

  /**
   * Generates a unique instance ID
   */
  private generateInstanceId(type: ComponentType): string {
    const timestamp = Date.now();
    const random = Math.random().toString(36).substr(2, 9);
    return `${type}_${timestamp}_${random}`;
  }

  /**
   * Updates pagination state based on current instances
   */
  private updatePaginationForType(type: ComponentType): void {
    const instances = this.getInstancesByType(type);
    const totalItems = instances.length;
    const itemsPerPage = 1; // One instance per page for dashboard layout
    const totalPages = Math.max(1, totalItems);

    this.updatePaginationState(type, {
      totalItems,
      totalPages,
      itemsPerPage,
    });
  }

  /**
   * Initializes default component types
   */
  private initializeDefaultComponents(): void {
    // Register metrics component
    this.registerComponent({
      type: ComponentType.METRICS,
      displayName: 'Key Metrics',
      config: DEFAULT_COMPONENT_CONFIGS[ComponentType.METRICS],
      factory: (id: string, layout: LayoutItem) => ({
        id,
        type: ComponentType.METRICS,
        title: 'Key Metrics',
        layout,
        hasData: false,
        pagination: {
          currentPage: 0,
          totalPages: 1,
          itemsPerPage: 1,
          totalItems: 0,
        },
      }),
    });

    // Register table component
    this.registerComponent({
      type: ComponentType.TABLE,
      displayName: 'Data Table',
      config: DEFAULT_COMPONENT_CONFIGS[ComponentType.TABLE],
      factory: (id: string, layout: LayoutItem) => ({
        id,
        type: ComponentType.TABLE,
        title: 'Data Table',
        layout,
        hasData: false,
        pagination: {
          currentPage: 0,
          totalPages: 1,
          itemsPerPage: 1,
          totalItems: 0,
        },
      }),
    });

    // Register image component
    this.registerComponent({
      type: ComponentType.IMAGE,
      displayName: 'Image Display',
      config: DEFAULT_COMPONENT_CONFIGS[ComponentType.IMAGE],
      factory: (id: string, layout: LayoutItem) => ({
        id,
        type: ComponentType.IMAGE,
        title: 'Image Display',
        layout,
        hasData: false,
        pagination: {
          currentPage: 0,
          totalPages: 1,
          itemsPerPage: 1,
          totalItems: 0,
        },
      }),
    });

    // Register insights component
    this.registerComponent({
      type: ComponentType.INSIGHTS,
      displayName: 'Smart Insights',
      config: DEFAULT_COMPONENT_CONFIGS[ComponentType.INSIGHTS],
      factory: (id: string, layout: LayoutItem) => ({
        id,
        type: ComponentType.INSIGHTS,
        title: 'Smart Insights',
        layout,
        hasData: false,
        pagination: {
          currentPage: 0,
          totalPages: 1,
          itemsPerPage: 1,
          totalItems: 0,
        },
      }),
    });

    // Register file download component
    this.registerComponent({
      type: ComponentType.FILE_DOWNLOAD,
      displayName: 'File Downloads',
      config: DEFAULT_COMPONENT_CONFIGS[ComponentType.FILE_DOWNLOAD],
      factory: (id: string, layout: LayoutItem) => ({
        id,
        type: ComponentType.FILE_DOWNLOAD,
        title: 'File Downloads',
        layout,
        hasData: false,
        // No pagination for file download component
      }),
    });
  }

  /**
   * Clears all instances and resets state
   */
  public clear(): void {
    this.instances.clear();
    this.dataUpdateCallbacks.clear();
    
    // Reset pagination states
    for (const type of this.registry.keys()) {
      const entry = this.registry.get(type);
      if (entry?.config.supportsPagination) {
        this.paginationStates.set(type, {
          currentPage: 0,
          totalPages: 1,
          itemsPerPage: 1,
          totalItems: 0,
        });
      }
    }
  }

  /**
   * Gets component statistics
   */
  public getStats(): {
    totalInstances: number;
    instancesByType: Record<string, number>;
    instancesWithData: number;
    instancesWithoutData: number;
  } {
    const instances = this.getAllInstances();
    const instancesByType: Record<string, number> = {};
    
    for (const type of this.getRegisteredTypes()) {
      instancesByType[type] = this.getInstancesByType(type).length;
    }

    return {
      totalInstances: instances.length,
      instancesByType,
      instancesWithData: this.getInstancesWithData().length,
      instancesWithoutData: this.getInstancesWithoutData().length,
    };
  }
}

// ============================================================================
// EXPORTS
// ============================================================================

export default ComponentManager;