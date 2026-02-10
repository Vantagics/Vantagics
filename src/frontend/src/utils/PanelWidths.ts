/**
 * PanelWidths Utility Module
 * 
 * Manages panel width calculations, constraints, and persistence for the three-panel layout.
 * Implements Requirements: 1.4, 9.1, 9.2, 9.4, 9.5
 */

/**
 * Interface for panel width configuration
 */
export interface PanelWidths {
  left: number;
  center: number;  // Calculated as remaining space
  right: number;
}

/**
 * Panel width constraints
 */
export const PANEL_CONSTRAINTS = {
  left: {
    min: 180,
    max: 400,
    default: 256
  },
  center: {
    min: 300,
    max: Infinity  // Fills remaining space
  },
  right: {
    min: 280,
    max: Infinity,
    default: 384  // Fallback only; getDefaultPanelWidths uses 1:2 ratio
  }
} as const;

/**
 * LocalStorage key for persisting panel widths (legacy fallback)
 */
const STORAGE_KEY = 'panelWidths';

/**
 * LocalStorage key for persisting sidebar width (legacy fallback)
 */
const SIDEBAR_STORAGE_KEY = 'sidebarWidth';

/**
 * Calculate panel widths with constraint enforcement
 * 
 * @param totalWidth - Total available width for all panels
 * @param leftWidth - Desired width for left panel
 * @param rightWidth - Desired width for right panel
 * @returns Constrained panel widths that sum to totalWidth
 * 
 * Validates: Requirements 1.4, 9.4, 9.5
 */
export function calculatePanelWidths(
  totalWidth: number,
  leftWidth: number,
  rightWidth: number
): PanelWidths {
  // Enforce constraints on left panel
  const constrainedLeft = Math.max(
    PANEL_CONSTRAINTS.left.min,
    Math.min(PANEL_CONSTRAINTS.left.max, leftWidth)
  );
  
  // Enforce constraints on right panel
  const constrainedRight = Math.max(
    PANEL_CONSTRAINTS.right.min,
    Math.min(PANEL_CONSTRAINTS.right.max, rightWidth)
  );
  
  // Calculate center width (remaining space)
  let centerWidth = totalWidth - constrainedLeft - constrainedRight;
  
  // Ensure center meets minimum requirement
  if (centerWidth < PANEL_CONSTRAINTS.center.min) {
    // Try to reduce right panel to accommodate center minimum
    const adjustedRight = Math.max(
      PANEL_CONSTRAINTS.right.min,
      totalWidth - constrainedLeft - PANEL_CONSTRAINTS.center.min
    );
    
    centerWidth = totalWidth - constrainedLeft - adjustedRight;
    
    // If center is still too small, try reducing left panel as well
    if (centerWidth < PANEL_CONSTRAINTS.center.min) {
      const adjustedLeft = Math.max(
        PANEL_CONSTRAINTS.left.min,
        totalWidth - PANEL_CONSTRAINTS.center.min - adjustedRight
      );
      
      return {
        left: adjustedLeft,
        center: totalWidth - adjustedLeft - adjustedRight,
        right: adjustedRight
      };
    }
    
    return {
      left: constrainedLeft,
      center: centerWidth,
      right: adjustedRight
    };
  }
  
  return {
    left: constrainedLeft,
    center: centerWidth,
    right: constrainedRight
  };
}

/**
 * Handle resize drag operations
 * 
 * @param handlePosition - Which resize handle is being dragged ('left' or 'right')
 * @param deltaX - Horizontal movement in pixels (positive = right, negative = left)
 * @param currentWidths - Current panel widths
 * @param totalWidth - Total available width
 * @returns New panel widths after applying the drag delta
 * 
 * Validates: Requirements 1.9
 */
export function handleResizeDrag(
  handlePosition: 'left' | 'right',
  deltaX: number,
  currentWidths: PanelWidths,
  totalWidth: number
): PanelWidths {
  if (handlePosition === 'left') {
    // Dragging between left and center panels
    // Moving right increases left panel, decreases center
    const newLeftWidth = currentWidths.left + deltaX;
    return calculatePanelWidths(
      totalWidth,
      newLeftWidth,
      currentWidths.right
    );
  } else {
    // Dragging between center and right panels
    // Moving right decreases right panel, increases center
    const newRightWidth = currentWidths.right - deltaX;
    return calculatePanelWidths(
      totalWidth,
      currentWidths.left,
      newRightWidth
    );
  }
}

/**
 * Save panel widths to localStorage
 * 
 * @param widths - Panel widths to persist
 * @returns true if save was successful, false otherwise
 * 
 * Validates: Requirements 9.1
 */
export function savePanelWidths(widths: PanelWidths): boolean {
  try {
    const data = {
      left: widths.left,
      right: widths.right
      // Note: center is calculated, so we don't persist it
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
    return true;
  } catch (error) {
    console.warn('Failed to save panel widths to localStorage:', error);
    return false;
  }
}

/**
 * Load panel widths from localStorage
 * 
 * @param totalWidth - Total available width for calculating center panel
 * @returns Loaded panel widths, or null if no saved widths exist
 * 
 * Validates: Requirements 9.2
 */
export function loadPanelWidths(totalWidth: number): PanelWidths | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return null;
    }
    
    const data = JSON.parse(stored);
    
    // Validate that we have the required fields
    if (typeof data.left !== 'number' || typeof data.right !== 'number') {
      console.warn('Invalid panel widths data in localStorage');
      return null;
    }
    
    // Recalculate with current total width to ensure constraints are met
    return calculatePanelWidths(totalWidth, data.left, data.right);
  } catch (error) {
    console.warn('Failed to load panel widths from localStorage:', error);
    return null;
  }
}

/**
 * Get default panel widths using 1:2 ratio (center:right)
 * 
 * @param totalWidth - Total available width (excluding sidebar)
 * @returns Default panel widths with center being ~1/3 and right ~2/3
 * 
 * Validates: Requirements 9.3
 */
export function getDefaultPanelWidths(totalWidth: number): PanelWidths {
  // 1:2 ratio means center gets 1/3, right gets 2/3
  // totalWidth is already the space for center+right (sidebar excluded)
  const rightWidth = Math.round(totalWidth * 2 / 3);
  const centerWidth = totalWidth - rightWidth;
  return {
    left: 0,
    center: Math.max(PANEL_CONSTRAINTS.center.min, centerWidth),
    right: Math.max(PANEL_CONSTRAINTS.right.min, rightWidth)
  };
}

/**
 * Clear saved panel widths from localStorage
 * 
 * @returns true if clear was successful, false otherwise
 */
export function clearPanelWidths(): boolean {
  try {
    localStorage.removeItem(STORAGE_KEY);
    return true;
  } catch (error) {
    console.warn('Failed to clear panel widths from localStorage:', error);
    return false;
  }
}

/**
 * Save sidebar width to localStorage
 */
export function saveSidebarWidth(width: number): boolean {
  try {
    localStorage.setItem(SIDEBAR_STORAGE_KEY, JSON.stringify(width));
    return true;
  } catch (error) {
    console.warn('Failed to save sidebar width to localStorage:', error);
    return false;
  }
}

/**
 * Load sidebar width from localStorage
 * 
 * @returns Saved sidebar width, or null if none exists
 */
export function loadSidebarWidth(): number | null {
  try {
    const stored = localStorage.getItem(SIDEBAR_STORAGE_KEY);
    if (!stored) return null;
    const width = JSON.parse(stored);
    if (typeof width !== 'number' || width < PANEL_CONSTRAINTS.left.min || width > PANEL_CONSTRAINTS.left.max) {
      return null;
    }
    return width;
  } catch (error) {
    console.warn('Failed to load sidebar width from localStorage:', error);
    return null;
  }
}
