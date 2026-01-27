/**
 * Visual Feedback System for Dashboard Drag-Drop Layout
 * 
 * Provides visual feedback during drag, resize, and other interactions
 */

export interface Position {
  x: number;
  y: number;
}

export interface Size {
  width: number;
  height: number;
}

export interface Bounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

/**
 * Visual feedback manager for dashboard interactions
 */
export class VisualFeedback {
  private container: HTMLElement | null = null;
  private snapIndicator: HTMLElement | null = null;
  private dragPreview: HTMLElement | null = null;
  private resizePreview: HTMLElement | null = null;

  constructor(containerElement?: HTMLElement) {
    this.container = containerElement || document.body;
    this.initializeElements();
  }

  /**
   * Initialize visual feedback elements
   */
  private initializeElements(): void {
    // Create snap indicator
    this.snapIndicator = document.createElement('div');
    this.snapIndicator.className = 'grid-snap-indicator';
    this.snapIndicator.style.display = 'none';
    this.container?.appendChild(this.snapIndicator);

    // Create drag preview
    this.dragPreview = document.createElement('div');
    this.dragPreview.className = 'drag-preview';
    this.dragPreview.style.display = 'none';
    this.dragPreview.style.position = 'absolute';
    this.dragPreview.style.pointerEvents = 'none';
    this.dragPreview.style.zIndex = '1001';
    this.container?.appendChild(this.dragPreview);

    // Create resize preview
    this.resizePreview = document.createElement('div');
    this.resizePreview.className = 'resize-preview';
    this.resizePreview.style.display = 'none';
    this.resizePreview.style.position = 'absolute';
    this.resizePreview.style.pointerEvents = 'none';
    this.resizePreview.style.zIndex = '1001';
    this.resizePreview.style.border = '2px dashed rgba(59, 130, 246, 0.8)';
    this.resizePreview.style.backgroundColor = 'rgba(59, 130, 246, 0.1)';
    this.container?.appendChild(this.resizePreview);
  }

  /**
   * Show grid snap indicator
   */
  showSnapIndicator(bounds: Bounds): void {
    if (!this.snapIndicator) return;

    this.snapIndicator.style.display = 'block';
    this.snapIndicator.style.left = `${bounds.x}px`;
    this.snapIndicator.style.top = `${bounds.y}px`;
    this.snapIndicator.style.width = `${bounds.width}px`;
    this.snapIndicator.style.height = `${bounds.height}px`;
  }

  /**
   * Hide grid snap indicator
   */
  hideSnapIndicator(): void {
    if (this.snapIndicator) {
      this.snapIndicator.style.display = 'none';
    }
  }

  /**
   * Show drag preview
   */
  showDragPreview(element: HTMLElement, position: Position): void {
    if (!this.dragPreview) return;

    // Clone the element for preview
    const clone = element.cloneNode(true) as HTMLElement;
    clone.style.opacity = '0.8';
    clone.style.transform = 'rotate(2deg)';
    clone.style.width = `${element.offsetWidth}px`;
    clone.style.height = `${element.offsetHeight}px`;

    this.dragPreview.innerHTML = '';
    this.dragPreview.appendChild(clone);
    this.dragPreview.style.display = 'block';
    this.dragPreview.style.left = `${position.x}px`;
    this.dragPreview.style.top = `${position.y}px`;
  }

  /**
   * Update drag preview position
   */
  updateDragPreview(position: Position): void {
    if (!this.dragPreview) return;

    this.dragPreview.style.left = `${position.x}px`;
    this.dragPreview.style.top = `${position.y}px`;
  }

  /**
   * Hide drag preview
   */
  hideDragPreview(): void {
    if (this.dragPreview) {
      this.dragPreview.style.display = 'none';
      this.dragPreview.innerHTML = '';
    }
  }

  /**
   * Show resize preview
   */
  showResizePreview(bounds: Bounds): void {
    if (!this.resizePreview) return;

    this.resizePreview.style.display = 'block';
    this.resizePreview.style.left = `${bounds.x}px`;
    this.resizePreview.style.top = `${bounds.y}px`;
    this.resizePreview.style.width = `${bounds.width}px`;
    this.resizePreview.style.height = `${bounds.height}px`;
  }

  /**
   * Update resize preview
   */
  updateResizePreview(bounds: Bounds): void {
    if (!this.resizePreview) return;

    this.resizePreview.style.left = `${bounds.x}px`;
    this.resizePreview.style.top = `${bounds.y}px`;
    this.resizePreview.style.width = `${bounds.width}px`;
    this.resizePreview.style.height = `${bounds.height}px`;
  }

  /**
   * Hide resize preview
   */
  hideResizePreview(): void {
    if (this.resizePreview) {
      this.resizePreview.style.display = 'none';
    }
  }

  /**
   * Show collision warning
   */
  showCollisionWarning(element: HTMLElement): void {
    element.classList.add('draggable-component--collision-warning');
    
    // Remove warning after animation
    setTimeout(() => {
      element.classList.remove('draggable-component--collision-warning');
    }, 300);
  }

  /**
   * Add drag state to element
   */
  addDragState(element: HTMLElement): void {
    element.classList.add('draggable-component--dragging');
    this.updateCursor('grabbing');
  }

  /**
   * Remove drag state from element
   */
  removeDragState(element: HTMLElement): void {
    element.classList.remove('draggable-component--dragging');
    this.updateCursor('default');
  }

  /**
   * Add resize state to element
   */
  addResizeState(element: HTMLElement): void {
    element.classList.add('draggable-component--resizing');
  }

  /**
   * Remove resize state from element
   */
  removeResizeState(element: HTMLElement): void {
    element.classList.remove('draggable-component--resizing');
  }

  /**
   * Update cursor style
   */
  private updateCursor(cursor: string): void {
    if (this.container) {
      this.container.style.cursor = cursor;
    }
  }

  /**
   * Show component addition animation
   */
  animateComponentAddition(element: HTMLElement): void {
    element.classList.add('draggable-component--newly-added');
    
    // Remove class after animation
    setTimeout(() => {
      element.classList.remove('draggable-component--newly-added');
    }, 300);
  }

  /**
   * Show component removal animation
   */
  animateComponentRemoval(element: HTMLElement): Promise<void> {
    return new Promise((resolve) => {
      element.classList.add('draggable-component--removing');
      
      // Wait for animation to complete
      setTimeout(() => {
        resolve();
      }, 200);
    });
  }

  /**
   * Show layout compaction animation
   */
  animateLayoutCompaction(containerElement: HTMLElement): void {
    containerElement.classList.add('dashboard-container--compacting');
    
    // Remove class after animation
    setTimeout(() => {
      containerElement.classList.remove('dashboard-container--compacting');
    }, 400);
  }

  /**
   * Create loading skeleton
   */
  createLoadingSkeleton(bounds: Bounds): HTMLElement {
    const skeleton = document.createElement('div');
    skeleton.className = 'loading-skeleton';
    skeleton.style.position = 'absolute';
    skeleton.style.left = `${bounds.x}px`;
    skeleton.style.top = `${bounds.y}px`;
    skeleton.style.width = `${bounds.width}px`;
    skeleton.style.height = `${bounds.height}px`;
    skeleton.style.borderRadius = '4px';
    
    return skeleton;
  }

  /**
   * Show loading state
   */
  showLoadingState(element: HTMLElement): void {
    const spinner = document.createElement('div');
    spinner.className = 'loading-spinner';
    spinner.innerHTML = '⟳';
    spinner.style.position = 'absolute';
    spinner.style.top = '50%';
    spinner.style.left = '50%';
    spinner.style.transform = 'translate(-50%, -50%)';
    spinner.style.fontSize = '24px';
    spinner.style.color = 'rgba(59, 130, 246, 0.8)';
    
    element.appendChild(spinner);
    element.classList.add('loading-state');
  }

  /**
   * Hide loading state
   */
  hideLoadingState(element: HTMLElement): void {
    const spinner = element.querySelector('.loading-spinner');
    if (spinner) {
      spinner.remove();
    }
    element.classList.remove('loading-state');
  }

  /**
   * Show error state
   */
  showErrorState(element: HTMLElement, message: string): void {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-state';
    errorDiv.style.position = 'absolute';
    errorDiv.style.top = '50%';
    errorDiv.style.left = '50%';
    errorDiv.style.transform = 'translate(-50%, -50%)';
    errorDiv.style.padding = '8px 12px';
    errorDiv.style.backgroundColor = 'rgba(239, 68, 68, 0.1)';
    errorDiv.style.border = '1px solid rgba(239, 68, 68, 0.3)';
    errorDiv.style.borderRadius = '4px';
    errorDiv.style.color = 'rgba(239, 68, 68, 0.9)';
    errorDiv.style.fontSize = '12px';
    errorDiv.style.textAlign = 'center';
    errorDiv.textContent = message;
    
    element.appendChild(errorDiv);
  }

  /**
   * Hide error state
   */
  hideErrorState(element: HTMLElement): void {
    const errorDiv = element.querySelector('.error-state');
    if (errorDiv) {
      errorDiv.remove();
    }
  }

  /**
   * Cleanup visual feedback elements
   */
  cleanup(): void {
    if (this.snapIndicator) {
      this.snapIndicator.remove();
      this.snapIndicator = null;
    }
    
    if (this.dragPreview) {
      this.dragPreview.remove();
      this.dragPreview = null;
    }
    
    if (this.resizePreview) {
      this.resizePreview.remove();
      this.resizePreview = null;
    }
  }
}

/**
 * Global visual feedback instance
 */
let globalVisualFeedback: VisualFeedback | null = null;

/**
 * Get or create global visual feedback instance
 */
export function getVisualFeedback(container?: HTMLElement): VisualFeedback {
  if (!globalVisualFeedback) {
    globalVisualFeedback = new VisualFeedback(container);
  }
  return globalVisualFeedback;
}

/**
 * Cleanup global visual feedback instance
 */
export function cleanupVisualFeedback(): void {
  if (globalVisualFeedback) {
    globalVisualFeedback.cleanup();
    globalVisualFeedback = null;
  }
}