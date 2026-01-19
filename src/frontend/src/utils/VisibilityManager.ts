import { ComponentInstance } from './ComponentManager';
import { ComponentType } from '../components/PaginationControl';

export interface VisibilityState {
  isVisible: boolean;
  reason: 'has_data' | 'no_data_edit_mode' | 'no_data_locked_mode' | 'group_hidden';
}

export interface GroupVisibilityState {
  hasVisibleInstances: boolean;
  totalInstances: number;
  visibleInstances: number;
  shouldShowPagination: boolean;
}

export class VisibilityManager {
  private componentVisibility: Map<string, VisibilityState> = new Map();
  private groupVisibility: Map<ComponentType, GroupVisibilityState> = new Map();

  /**
   * Check if a component should be visible based on data availability and mode
   */
  checkComponentVisibility(
    instance: ComponentInstance,
    isEditMode: boolean,
    isLocked: boolean
  ): VisibilityState {
    // In edit mode, all components are visible (with empty states if no data)
    if (isEditMode) {
      return {
        isVisible: true,
        reason: instance.hasData ? 'has_data' : 'no_data_edit_mode'
      };
    }

    // In locked mode, only show components with data
    if (isLocked) {
      if (instance.hasData) {
        return {
          isVisible: true,
          reason: 'has_data'
        };
      } else {
        return {
          isVisible: false,
          reason: 'no_data_locked_mode'
        };
      }
    }

    // Default view mode - show components with data
    return {
      isVisible: instance.hasData,
      reason: instance.hasData ? 'has_data' : 'no_data_locked_mode'
    };
  }

  /**
   * Update visibility state for a component
   */
  updateComponentVisibility(
    instanceId: string,
    instance: ComponentInstance,
    isEditMode: boolean,
    isLocked: boolean
  ): VisibilityState {
    const visibility = this.checkComponentVisibility(instance, isEditMode, isLocked);
    this.componentVisibility.set(instanceId, visibility);
    return visibility;
  }

  /**
   * Get visibility state for a component
   */
  getComponentVisibility(instanceId: string): VisibilityState | undefined {
    return this.componentVisibility.get(instanceId);
  }

  /**
   * Check group visibility for a component type
   */
  checkGroupVisibility(
    componentType: ComponentType,
    instances: ComponentInstance[],
    isEditMode: boolean,
    isLocked: boolean
  ): GroupVisibilityState {
    const visibleInstances = instances.filter(instance => {
      const visibility = this.checkComponentVisibility(instance, isEditMode, isLocked);
      return visibility.isVisible;
    });

    const groupState: GroupVisibilityState = {
      hasVisibleInstances: visibleInstances.length > 0,
      totalInstances: instances.length,
      visibleInstances: visibleInstances.length,
      shouldShowPagination: visibleInstances.length > 1
    };

    this.groupVisibility.set(componentType, groupState);
    return groupState;
  }

  /**
   * Update group visibility for a component type
   */
  updateGroupVisibility(
    componentType: ComponentType,
    instances: ComponentInstance[],
    isEditMode: boolean,
    isLocked: boolean
  ): GroupVisibilityState {
    return this.checkGroupVisibility(componentType, instances, isEditMode, isLocked);
  }

  /**
   * Get group visibility state for a component type
   */
  getGroupVisibility(componentType: ComponentType): GroupVisibilityState | undefined {
    return this.groupVisibility.get(componentType);
  }

  /**
   * Batch update visibility for all components
   */
  updateAllVisibility(
    instancesByType: Map<ComponentType, ComponentInstance[]>,
    isEditMode: boolean,
    isLocked: boolean
  ): Map<ComponentType, GroupVisibilityState> {
    const results = new Map<ComponentType, GroupVisibilityState>();

    instancesByType.forEach((instances, componentType) => {
      // Update individual component visibility
      instances.forEach(instance => {
        this.updateComponentVisibility(instance.id, instance, isEditMode, isLocked);
      });

      // Update group visibility
      const groupState = this.updateGroupVisibility(componentType, instances, isEditMode, isLocked);
      results.set(componentType, groupState);
    });

    return results;
  }

  /**
   * Get all visible components
   */
  getVisibleComponents(): string[] {
    const visibleComponents: string[] = [];
    
    this.componentVisibility.forEach((visibility, instanceId) => {
      if (visibility.isVisible) {
        visibleComponents.push(instanceId);
      }
    });

    return visibleComponents;
  }

  /**
   * Get all hidden components
   */
  getHiddenComponents(): string[] {
    const hiddenComponents: string[] = [];
    
    this.componentVisibility.forEach((visibility, instanceId) => {
      if (!visibility.isVisible) {
        hiddenComponents.push(instanceId);
      }
    });

    return hiddenComponents;
  }

  /**
   * Get visibility statistics
   */
  getVisibilityStats(): {
    totalComponents: number;
    visibleComponents: number;
    hiddenComponents: number;
    groupsWithVisibleComponents: number;
    totalGroups: number;
  } {
    const visibleComponents = this.getVisibleComponents();
    const hiddenComponents = this.getHiddenComponents();
    const groupsWithVisible = Array.from(this.groupVisibility.values())
      .filter(group => group.hasVisibleInstances).length;

    return {
      totalComponents: this.componentVisibility.size,
      visibleComponents: visibleComponents.length,
      hiddenComponents: hiddenComponents.length,
      groupsWithVisibleComponents: groupsWithVisible,
      totalGroups: this.groupVisibility.size
    };
  }

  /**
   * Clear all visibility state
   */
  clear(): void {
    this.componentVisibility.clear();
    this.groupVisibility.clear();
  }

  /**
   * Check if any components are visible
   */
  hasVisibleComponents(): boolean {
    return Array.from(this.componentVisibility.values())
      .some(visibility => visibility.isVisible);
  }

  /**
   * Check if a component type has any visible instances
   */
  hasVisibleInstancesOfType(componentType: ComponentType): boolean {
    const groupState = this.groupVisibility.get(componentType);
    return groupState?.hasVisibleInstances ?? false;
  }

  /**
   * Get components that should show empty state indicators in edit mode
   */
  getComponentsWithEmptyState(): string[] {
    const emptyStateComponents: string[] = [];
    
    this.componentVisibility.forEach((visibility, instanceId) => {
      if (visibility.isVisible && visibility.reason === 'no_data_edit_mode') {
        emptyStateComponents.push(instanceId);
      }
    });

    return emptyStateComponents;
  }
}

export default VisibilityManager;