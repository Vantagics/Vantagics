/**
 * Error Handler for Dashboard Drag-Drop Layout
 * 
 * Centralized error handling and user-friendly error messages
 */

export interface DashboardError {
  id: string;
  type: ErrorType;
  message: string;
  details?: string;
  componentId?: string;
  timestamp: number;
  severity: ErrorSeverity;
  recoverable: boolean;
}

export enum ErrorType {
  LAYOUT_SAVE_FAILED = 'layout_save_failed',
  LAYOUT_LOAD_FAILED = 'layout_load_failed',
  COMPONENT_DATA_FAILED = 'component_data_failed',
  FILE_DOWNLOAD_FAILED = 'file_download_failed',
  EXPORT_FAILED = 'export_failed',
  DRAG_OPERATION_FAILED = 'drag_operation_failed',
  RESIZE_OPERATION_FAILED = 'resize_operation_failed',
  VALIDATION_ERROR = 'validation_error',
  NETWORK_ERROR = 'network_error',
  PERMISSION_ERROR = 'permission_error',
  UNKNOWN_ERROR = 'unknown_error',
}

export enum ErrorSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

export interface ErrorRecoveryAction {
  label: string;
  action: () => void | Promise<void>;
}

/**
 * Error handler class
 */
export class ErrorHandler {
  private errors: Map<string, DashboardError> = new Map();
  private listeners: Set<(errors: DashboardError[]) => void> = new Set();
  private maxErrors = 50; // Maximum number of errors to keep in memory

  /**
   * Handle an error
   */
  handleError(
    type: ErrorType,
    message: string,
    details?: string,
    componentId?: string,
    severity: ErrorSeverity = ErrorSeverity.MEDIUM
  ): string {
    const error: DashboardError = {
      id: this.generateErrorId(),
      type,
      message,
      details,
      componentId,
      timestamp: Date.now(),
      severity,
      recoverable: this.isRecoverable(type),
    };

    this.errors.set(error.id, error);
    this.trimErrors();
    this.notifyListeners();

    // Log error for debugging in development
    if (process.env.NODE_ENV === 'development') {
      console.error(`Dashboard Error [${type}]:`, message, details);
    }

    return error.id;
  }

  /**
   * Handle a caught exception
   */
  handleException(
    error: unknown,
    context: string,
    componentId?: string
  ): string {
    let message = 'An unexpected error occurred';
    let details = '';
    let type = ErrorType.UNKNOWN_ERROR;
    let severity = ErrorSeverity.MEDIUM;

    if (error instanceof Error) {
      message = error.message;
      details = error.stack || '';
      
      // Classify error based on message
      if (error.message.includes('network') || error.message.includes('fetch')) {
        type = ErrorType.NETWORK_ERROR;
      } else if (error.message.includes('permission') || error.message.includes('unauthorized')) {
        type = ErrorType.PERMISSION_ERROR;
        severity = ErrorSeverity.HIGH;
      }
    } else if (typeof error === 'string') {
      message = error;
    }

    return this.handleError(type, `${context}: ${message}`, details, componentId, severity);
  }

  /**
   * Get user-friendly error message
   */
  getUserFriendlyMessage(errorType: ErrorType, originalMessage?: string): string {
    const messages: Record<ErrorType, string> = {
      [ErrorType.LAYOUT_SAVE_FAILED]: 'Failed to save dashboard layout. Your changes may not be preserved.',
      [ErrorType.LAYOUT_LOAD_FAILED]: 'Failed to load dashboard layout. Using default layout instead.',
      [ErrorType.COMPONENT_DATA_FAILED]: 'Failed to load component data. The component may appear empty.',
      [ErrorType.FILE_DOWNLOAD_FAILED]: 'Failed to download file. Please try again.',
      [ErrorType.EXPORT_FAILED]: 'Failed to export dashboard. Please try again.',
      [ErrorType.DRAG_OPERATION_FAILED]: 'Failed to move component. The component has been returned to its original position.',
      [ErrorType.RESIZE_OPERATION_FAILED]: 'Failed to resize component. The component has been returned to its original size.',
      [ErrorType.VALIDATION_ERROR]: 'Invalid operation. Please check your input and try again.',
      [ErrorType.NETWORK_ERROR]: 'Network connection error. Please check your internet connection and try again.',
      [ErrorType.PERMISSION_ERROR]: 'You do not have permission to perform this action.',
      [ErrorType.UNKNOWN_ERROR]: 'An unexpected error occurred. Please try again.',
    };

    return messages[errorType] || originalMessage || 'An error occurred';
  }

  /**
   * Get recovery actions for an error
   */
  getRecoveryActions(errorId: string): ErrorRecoveryAction[] {
    const error = this.errors.get(errorId);
    if (!error) return [];

    const actions: ErrorRecoveryAction[] = [];

    switch (error.type) {
      case ErrorType.LAYOUT_SAVE_FAILED:
        actions.push({
          label: 'Retry Save',
          action: () => this.retryLayoutSave(error.componentId),
        });
        actions.push({
          label: 'Reset Layout',
          action: () => this.resetLayout(),
        });
        break;

      case ErrorType.LAYOUT_LOAD_FAILED:
        actions.push({
          label: 'Retry Load',
          action: () => this.retryLayoutLoad(),
        });
        actions.push({
          label: 'Use Default Layout',
          action: () => this.useDefaultLayout(),
        });
        break;

      case ErrorType.COMPONENT_DATA_FAILED:
        actions.push({
          label: 'Retry Load Data',
          action: () => this.retryComponentData(error.componentId),
        });
        break;

      case ErrorType.FILE_DOWNLOAD_FAILED:
        actions.push({
          label: 'Retry Download',
          action: () => this.retryFileDownload(error.componentId),
        });
        break;

      case ErrorType.EXPORT_FAILED:
        actions.push({
          label: 'Retry Export',
          action: () => this.retryExport(),
        });
        break;

      case ErrorType.NETWORK_ERROR:
        actions.push({
          label: 'Retry',
          action: () => this.retryLastOperation(error.componentId),
        });
        break;

      default:
        if (error.recoverable) {
          actions.push({
            label: 'Retry',
            action: () => this.retryLastOperation(error.componentId),
          });
        }
        break;
    }

    // Always add dismiss action
    actions.push({
      label: 'Dismiss',
      action: () => this.dismissError(errorId),
    });

    return actions;
  }

  /**
   * Dismiss an error
   */
  dismissError(errorId: string): void {
    this.errors.delete(errorId);
    this.notifyListeners();
  }

  /**
   * Dismiss all errors
   */
  dismissAllErrors(): void {
    this.errors.clear();
    this.notifyListeners();
  }

  /**
   * Get all errors
   */
  getAllErrors(): DashboardError[] {
    return Array.from(this.errors.values()).sort((a, b) => b.timestamp - a.timestamp);
  }

  /**
   * Get errors by component
   */
  getErrorsByComponent(componentId: string): DashboardError[] {
    return this.getAllErrors().filter(error => error.componentId === componentId);
  }

  /**
   * Get errors by severity
   */
  getErrorsBySeverity(severity: ErrorSeverity): DashboardError[] {
    return this.getAllErrors().filter(error => error.severity === severity);
  }

  /**
   * Check if there are any critical errors
   */
  hasCriticalErrors(): boolean {
    return this.getAllErrors().some(error => error.severity === ErrorSeverity.CRITICAL);
  }

  /**
   * Subscribe to error updates
   */
  subscribe(callback: (errors: DashboardError[]) => void): () => void {
    this.listeners.add(callback);
    
    // Return unsubscribe function
    return () => {
      this.listeners.delete(callback);
    };
  }

  /**
   * Generate unique error ID
   */
  private generateErrorId(): string {
    return `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Check if error type is recoverable
   */
  private isRecoverable(type: ErrorType): boolean {
    const recoverableTypes = [
      ErrorType.LAYOUT_SAVE_FAILED,
      ErrorType.LAYOUT_LOAD_FAILED,
      ErrorType.COMPONENT_DATA_FAILED,
      ErrorType.FILE_DOWNLOAD_FAILED,
      ErrorType.EXPORT_FAILED,
      ErrorType.NETWORK_ERROR,
    ];

    return recoverableTypes.includes(type);
  }

  /**
   * Trim errors to maximum limit
   */
  private trimErrors(): void {
    const errors = this.getAllErrors();
    if (errors.length > this.maxErrors) {
      const toRemove = errors.slice(this.maxErrors);
      toRemove.forEach(error => this.errors.delete(error.id));
    }
  }

  /**
   * Notify listeners of error changes
   */
  private notifyListeners(): void {
    const errors = this.getAllErrors();
    this.listeners.forEach(callback => callback(errors));
  }

  // Recovery action implementations
  private async retryLayoutSave(componentId?: string): Promise<void> {
    // Implementation would depend on the actual save mechanism
    // Retrying layout save for component: componentId
  }

  private async retryLayoutLoad(): Promise<void> {
    // Retrying layout load
  }

  private async resetLayout(): Promise<void> {
    // Resetting layout to default
  }

  private async useDefaultLayout(): Promise<void> {
    // Using default layout
  }

  private async retryComponentData(componentId?: string): Promise<void> {
    // Retrying component data load for: componentId
  }

  private async retryFileDownload(componentId?: string): Promise<void> {
    // Retrying file download for component: componentId
  }

  private async retryExport(): Promise<void> {
    // Retrying dashboard export
  }

  private async retryLastOperation(componentId?: string): Promise<void> {
    // Retrying last operation for component: componentId
  }
}

/**
 * Global error handler instance
 */
let globalErrorHandler: ErrorHandler | null = null;

/**
 * Get or create global error handler
 */
export function getErrorHandler(): ErrorHandler {
  if (!globalErrorHandler) {
    globalErrorHandler = new ErrorHandler();
  }
  return globalErrorHandler;
}

/**
 * Convenience function to handle errors
 */
export function handleError(
  type: ErrorType,
  message: string,
  details?: string,
  componentId?: string,
  severity: ErrorSeverity = ErrorSeverity.MEDIUM
): string {
  return getErrorHandler().handleError(type, message, details, componentId, severity);
}

/**
 * Convenience function to handle exceptions
 */
export function handleException(
  error: unknown,
  context: string,
  componentId?: string
): string {
  return getErrorHandler().handleException(error, context, componentId);
}

/**
 * React hook for error handling
 */
export function useErrorHandler() {
  const errorHandler = getErrorHandler();
  const [errors, setErrors] = React.useState<DashboardError[]>(errorHandler.getAllErrors());

  React.useEffect(() => {
    const unsubscribe = errorHandler.subscribe(setErrors);
    return unsubscribe;
  }, [errorHandler]);

  const handleError = React.useCallback((
    type: ErrorType,
    message: string,
    details?: string,
    componentId?: string,
    severity: ErrorSeverity = ErrorSeverity.MEDIUM
  ) => {
    return errorHandler.handleError(type, message, details, componentId, severity);
  }, [errorHandler]);

  const handleException = React.useCallback((
    error: unknown,
    context: string,
    componentId?: string
  ) => {
    return errorHandler.handleException(error, context, componentId);
  }, [errorHandler]);

  const dismissError = React.useCallback((errorId: string) => {
    errorHandler.dismissError(errorId);
  }, [errorHandler]);

  const dismissAllErrors = React.useCallback(() => {
    errorHandler.dismissAllErrors();
  }, [errorHandler]);

  const getRecoveryActions = React.useCallback((errorId: string) => {
    return errorHandler.getRecoveryActions(errorId);
  }, [errorHandler]);

  return {
    errors,
    handleError,
    handleException,
    dismissError,
    dismissAllErrors,
    getRecoveryActions,
    hasCriticalErrors: errorHandler.hasCriticalErrors(),
  };
}

// Add React import for hooks
import React from 'react';