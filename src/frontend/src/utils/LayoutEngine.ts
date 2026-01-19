/**
 * Layout Engine for Dashboard Drag-Drop Layout System
 * 
 * This module provides grid-based layout calculations, collision detection,
 * and layout compaction for the dashboard drag-drop system.
 */

// ============================================================================
// INTERFACES AND TYPES
// ============================================================================

/**
 * Grid configuration for the layout system
 */
export interface GridConfig {
  /** Number of columns in the grid (24 for responsive design) */
  columns: number;
  /** Height of each row unit in pixels */
  rowHeight: number;
  /** Horizontal and vertical margins between grid items [horizontal, vertical] */
  margin: [number, number];
  /** Container padding [horizontal, vertical] */
  containerPadding: [number, number];
}

/**
 * Layout item representing a component's position and size in the grid
 */
export interface LayoutItem {
  /** Unique component instance ID */
  i: string;
  /** Grid column position (0-based) */
  x: number;
  /** Grid row position (0-based) */
  y: number;
  /** Width in grid units */
  w: number;
  /** Height in grid units */
  h: number;
  /** Minimum width in grid units */
  minW?: number;
  /** Minimum height in grid units */
  minH?: number;
  /** Maximum width in grid units */
  maxW?: number;
  /** Maximum height in grid units */
  maxH?: number;
  /** If true, item cannot be dragged or resized */
  static?: boolean;
}

/**
 * Position coordinates
 */
export interface Position {
  x: number;
  y: number;
}

/**
 * Dimensions
 */
export interface Dimensions {
  width: number;
  height: number;
}

/**
 * Collision detection result
 */
export interface CollisionResult {
  /** Whether any collisions were detected */
  hasCollisions: boolean;
  /** Array of colliding item pairs */
  collisions: Array<{
    item1: LayoutItem;
    item2: LayoutItem;
  }>;
}

/**
 * Layout compaction result
 */
export interface CompactionResult {
  /** Compacted layout items */
  items: LayoutItem[];
  /** Whether any changes were made */
  changed: boolean;
}

// ============================================================================
// DEFAULT CONFIGURATION
// ============================================================================

/**
 * Default grid configuration for 24-column responsive layout
 */
export const DEFAULT_GRID_CONFIG: GridConfig = {
  columns: 24,
  rowHeight: 30,
  margin: [10, 10],
  containerPadding: [10, 10],
};

// ============================================================================
// LAYOUT ENGINE CLASS
// ============================================================================

/**
 * Layout Engine for grid-based positioning and collision detection
 */
export class LayoutEngine {
  private config: GridConfig;

  constructor(config: GridConfig = DEFAULT_GRID_CONFIG) {
    this.config = { ...config };
  }

  /**
   * Updates the grid configuration
   */
  public updateConfig(config: Partial<GridConfig>): void {
    this.config = { ...this.config, ...config };
  }

  /**
   * Gets the current grid configuration
   */
  public getConfig(): GridConfig {
    return { ...this.config };
  }

  // ========================================================================
  // POSITION CALCULATION
  // ========================================================================

  /**
   * Calculates a valid position for an item with collision detection
   * 
   * @param item - The layout item to position
   * @param x - Desired x position
   * @param y - Desired y position
   * @param existingItems - Array of existing items to check for collisions
   * @returns Valid position that doesn't collide with existing items
   */
  public calculatePosition(
    item: LayoutItem,
    x: number,
    y: number,
    existingItems: LayoutItem[] = []
  ): Position {
    // Snap to grid boundaries
    const snappedX = this.snapToGridX(x);
    const snappedY = this.snapToGridY(y);

    // Ensure position is within grid bounds
    const boundedX = Math.max(0, Math.min(snappedX, this.config.columns - item.w));
    const boundedY = Math.max(0, snappedY);

    // Create a test item with the new position
    const testItem: LayoutItem = {
      ...item,
      x: boundedX,
      y: boundedY,
    };

    // Check for collisions with existing items
    const collisions = this.getCollisionsForItem(testItem, existingItems);

    if (collisions.length === 0) {
      // No collisions, return the position
      return { x: boundedX, y: boundedY };
    }

    // Find the next available position
    return this.findNextAvailablePosition(testItem, existingItems);
  }

  /**
   * Finds the next available position for an item that doesn't collide
   */
  private findNextAvailablePosition(
    item: LayoutItem,
    existingItems: LayoutItem[]
  ): Position {
    const maxY = this.getMaxY(existingItems) + 10; // Add some buffer

    // Try positions row by row
    for (let y = item.y; y <= maxY; y++) {
      for (let x = 0; x <= this.config.columns - item.w; x++) {
        const testItem: LayoutItem = { ...item, x, y };
        const collisions = this.getCollisionsForItem(testItem, existingItems);
        
        if (collisions.length === 0) {
          return { x, y };
        }
      }
    }

    // If no position found, place at the bottom
    return { x: 0, y: maxY + 1 };
  }

  /**
   * Gets the maximum Y position of all items
   */
  private getMaxY(items: LayoutItem[]): number {
    if (items.length === 0) return 0;
    return Math.max(...items.map(item => item.y + item.h));
  }

  // ========================================================================
  // GRID SNAPPING
  // ========================================================================

  /**
   * Snaps positions and dimensions to grid boundaries
   * 
   * @param width - Width in pixels
   * @param height - Height in pixels
   * @returns Dimensions snapped to grid units
   */
  public snapToGrid(width: number, height: number): Dimensions {
    const gridWidth = this.snapToGridWidth(width);
    const gridHeight = this.snapToGridHeight(height);

    return {
      width: gridWidth,
      height: gridHeight,
    };
  }

  /**
   * Snaps a pixel width to grid units
   */
  private snapToGridWidth(width: number): number {
    const columnWidth = this.getColumnWidth();
    const gridUnits = Math.round(width / columnWidth);
    return Math.max(1, gridUnits); // Minimum 1 grid unit
  }

  /**
   * Snaps a pixel height to grid units
   */
  private snapToGridHeight(height: number): number {
    const gridUnits = Math.round(height / this.config.rowHeight);
    return Math.max(1, gridUnits); // Minimum 1 grid unit
  }

  /**
   * Snaps an x coordinate to grid column
   */
  private snapToGridX(x: number): number {
    const columnWidth = this.getColumnWidth();
    return Math.round(x / columnWidth);
  }

  /**
   * Snaps a y coordinate to grid row
   */
  private snapToGridY(y: number): number {
    return Math.round(y / this.config.rowHeight);
  }

  /**
   * Calculates the width of a single column
   */
  private getColumnWidth(): number {
    // This would typically be calculated based on container width
    // For now, we'll use a default value
    return 40; // pixels per column
  }

  // ========================================================================
  // COLLISION DETECTION
  // ========================================================================

  /**
   * Detects collisions between layout items
   * 
   * @param items - Array of layout items to check
   * @returns Collision detection result
   */
  public detectCollisions(items: LayoutItem[]): CollisionResult {
    const collisions: Array<{ item1: LayoutItem; item2: LayoutItem }> = [];

    for (let i = 0; i < items.length; i++) {
      for (let j = i + 1; j < items.length; j++) {
        const item1 = items[i];
        const item2 = items[j];

        if (this.itemsCollide(item1, item2)) {
          collisions.push({ item1, item2 });
        }
      }
    }

    return {
      hasCollisions: collisions.length > 0,
      collisions,
    };
  }

  /**
   * Checks if two items collide
   */
  private itemsCollide(item1: LayoutItem, item2: LayoutItem): boolean {
    // Items don't collide if they don't overlap in either dimension
    const noXOverlap = item1.x + item1.w <= item2.x || item2.x + item2.w <= item1.x;
    const noYOverlap = item1.y + item1.h <= item2.y || item2.y + item2.h <= item1.y;

    return !(noXOverlap || noYOverlap);
  }

  /**
   * Gets all items that collide with a specific item
   */
  private getCollisionsForItem(item: LayoutItem, items: LayoutItem[]): LayoutItem[] {
    return items.filter(otherItem => 
      otherItem.i !== item.i && this.itemsCollide(item, otherItem)
    );
  }

  // ========================================================================
  // LAYOUT COMPACTION
  // ========================================================================

  /**
   * Compacts the layout by moving items up to fill gaps
   * 
   * @param items - Array of layout items to compact
   * @returns Compacted layout result
   */
  public compactLayout(items: LayoutItem[]): CompactionResult {
    if (items.length === 0) {
      return { items: [], changed: false };
    }

    // Create a copy of items to avoid mutating the original
    const compactedItems = items.map(item => ({ ...item }));
    let changed = false;

    // Sort items by y position, then by x position
    compactedItems.sort((a, b) => {
      if (a.y !== b.y) return a.y - b.y;
      return a.x - b.x;
    });

    // Compact each item by moving it up as much as possible
    for (let i = 0; i < compactedItems.length; i++) {
      const item = compactedItems[i];
      
      // Skip static items
      if (item.static) continue;

      const originalY = item.y;
      const newY = this.findHighestValidPosition(item, compactedItems.slice(0, i));
      
      if (newY < originalY) {
        item.y = newY;
        changed = true;
      }
    }

    return {
      items: compactedItems,
      changed,
    };
  }

  /**
   * Finds the highest valid Y position for an item
   */
  private findHighestValidPosition(item: LayoutItem, existingItems: LayoutItem[]): number {
    // Start from y=0 and work down until we find a valid position
    for (let y = 0; y <= item.y; y++) {
      const testItem: LayoutItem = { ...item, y };
      const collisions = this.getCollisionsForItem(testItem, existingItems);
      
      if (collisions.length === 0) {
        return y;
      }
    }

    // If no valid position found above current position, return current
    return item.y;
  }

  // ========================================================================
  // UTILITY METHODS
  // ========================================================================

  /**
   * Validates that an item's dimensions are within constraints
   */
  public validateItemConstraints(item: LayoutItem): LayoutItem {
    const validatedItem = { ...item };

    // Apply minimum constraints
    if (item.minW !== undefined) {
      validatedItem.w = Math.max(validatedItem.w, item.minW);
    }
    if (item.minH !== undefined) {
      validatedItem.h = Math.max(validatedItem.h, item.minH);
    }

    // Apply maximum constraints
    if (item.maxW !== undefined) {
      validatedItem.w = Math.min(validatedItem.w, item.maxW);
    }
    if (item.maxH !== undefined) {
      validatedItem.h = Math.min(validatedItem.h, item.maxH);
    }

    // Ensure item fits within grid bounds
    validatedItem.w = Math.min(validatedItem.w, this.config.columns);
    validatedItem.x = Math.max(0, Math.min(validatedItem.x, this.config.columns - validatedItem.w));
    validatedItem.y = Math.max(0, validatedItem.y);

    return validatedItem;
  }

  /**
   * Converts grid coordinates to pixel coordinates
   */
  public gridToPixel(gridX: number, gridY: number): Position {
    const columnWidth = this.getColumnWidth();
    
    return {
      x: gridX * columnWidth + this.config.containerPadding[0],
      y: gridY * this.config.rowHeight + this.config.containerPadding[1],
    };
  }

  /**
   * Converts pixel coordinates to grid coordinates
   */
  public pixelToGrid(pixelX: number, pixelY: number): Position {
    const columnWidth = this.getColumnWidth();
    
    return {
      x: Math.round((pixelX - this.config.containerPadding[0]) / columnWidth),
      y: Math.round((pixelY - this.config.containerPadding[1]) / this.config.rowHeight),
    };
  }

  /**
   * Gets the total height of the layout in pixels
   */
  public getLayoutHeight(items: LayoutItem[]): number {
    if (items.length === 0) return 0;
    
    const maxY = this.getMaxY(items);
    return maxY * this.config.rowHeight + this.config.containerPadding[1] * 2;
  }
}

// ============================================================================
// EXPORTS
// ============================================================================

export default LayoutEngine;