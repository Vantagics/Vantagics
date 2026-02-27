package main

import "fmt"

// WrapOperationError wraps an error with a consistent "failed to {operation}: %w" format.
// This helper reduces code duplication across the codebase.
//
// Example:
//   err := someOperation()
//   if err != nil {
//       return WrapOperationError("load config", err)
//   }
func WrapOperationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to %s: %w", operation, err)
}

// WrapOperationErrorf wraps an error with additional context using format string.
//
// Example:
//   err := someOperation(id)
//   if err != nil {
//       return WrapOperationErrorf("load user %s", err, id)
//   }
func WrapOperationErrorf(format string, err error, args ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("failed to %s: %w", msg, err)
}
