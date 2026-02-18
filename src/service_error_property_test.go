package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: main-architecture-refactor, Property 9: 服务错误格式一致性
//
// For any service name and operation name combination, a ServiceError created via
// WrapError() should have Error() containing both names, and Unwrap() should return
// the original error.
//
// **Validates: Requirements 5.1, 5.2, 5.3**

// TestProperty9_ServiceErrorFormatConsistency verifies that for any random service name
// and operation name, the ServiceError format is consistent and Unwrap works correctly.
func TestProperty9_ServiceErrorFormatConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.String().Draw(t, "service")
		operation := rapid.String().Draw(t, "operation")
		errMsg := rapid.String().Draw(t, "errMsg")

		original := fmt.Errorf("%s", errMsg)
		wrapped := WrapError(service, operation, original)

		if wrapped == nil {
			t.Fatal("WrapError with non-nil error should return non-nil")
		}

		errStr := wrapped.Error()

		// Property: Error() string contains the service name
		if service != "" && !strings.Contains(errStr, service) {
			t.Fatalf("Error() %q should contain service name %q", errStr, service)
		}

		// Property: Error() string contains the operation name
		if operation != "" && !strings.Contains(errStr, operation) {
			t.Fatalf("Error() %q should contain operation name %q", errStr, operation)
		}

		// Property: Unwrap() returns the original error
		var se *ServiceError
		if !errors.As(wrapped, &se) {
			t.Fatal("wrapped error should be *ServiceError")
		}
		if se.Unwrap() != original {
			t.Fatal("Unwrap() should return the original error")
		}

		// Property: Error() format matches [Service.Operation] error message
		expected := fmt.Sprintf("[%s.%s] %s", service, operation, errMsg)
		if errStr != expected {
			t.Fatalf("Error() = %q, want %q", errStr, expected)
		}
	})
}

// TestProperty9b_WrapErrorNilReturnsNil verifies that WrapError with nil error always returns nil,
// regardless of service and operation names.
func TestProperty9b_WrapErrorNilReturnsNil(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.String().Draw(t, "service")
		operation := rapid.String().Draw(t, "operation")

		result := WrapError(service, operation, nil)
		if result != nil {
			t.Fatalf("WrapError(%q, %q, nil) should return nil, got %v", service, operation, result)
		}
	})
}

// TestProperty9c_ServiceErrorFieldsPreserved verifies that ServiceError preserves
// the Service, Operation, and Err fields exactly as provided.
func TestProperty9c_ServiceErrorFieldsPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.String().Draw(t, "service")
		operation := rapid.String().Draw(t, "operation")
		errMsg := rapid.String().Draw(t, "errMsg")

		original := fmt.Errorf("%s", errMsg)
		wrapped := WrapError(service, operation, original)

		var se *ServiceError
		if !errors.As(wrapped, &se) {
			t.Fatal("wrapped error should be *ServiceError")
		}

		if se.Service != service {
			t.Fatalf("Service = %q, want %q", se.Service, service)
		}
		if se.Operation != operation {
			t.Fatalf("Operation = %q, want %q", se.Operation, operation)
		}
		if se.Err != original {
			t.Fatal("Err should be the original error")
		}
	})
}
