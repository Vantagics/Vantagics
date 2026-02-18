package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestServiceError_Error(t *testing.T) {
	original := fmt.Errorf("connection refused")
	se := &ServiceError{
		Service:   "ChatService",
		Operation: "SendMessage",
		Err:       original,
	}

	got := se.Error()
	expected := "[ChatService.SendMessage] connection refused"
	if got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestServiceError_ErrorFormat(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		operation string
		err       error
		want      string
	}{
		{
			name:      "basic error",
			service:   "Config",
			operation: "Load",
			err:       fmt.Errorf("file not found"),
			want:      "[Config.Load] file not found",
		},
		{
			name:      "empty service name",
			service:   "",
			operation: "Save",
			err:       fmt.Errorf("disk full"),
			want:      "[.Save] disk full",
		},
		{
			name:      "empty operation name",
			service:   "Export",
			operation: "",
			err:       fmt.Errorf("timeout"),
			want:      "[Export.] timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &ServiceError{Service: tt.service, Operation: tt.operation, Err: tt.err}
			if got := se.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	original := fmt.Errorf("original error")
	se := &ServiceError{
		Service:   "Test",
		Operation: "Op",
		Err:       original,
	}

	if unwrapped := se.Unwrap(); unwrapped != original {
		t.Errorf("Unwrap() returned different error: got %v, want %v", unwrapped, original)
	}
}

func TestServiceError_ErrorsIs(t *testing.T) {
	sentinel := fmt.Errorf("sentinel error")
	se := WrapError("Svc", "Op", sentinel)

	if !errors.Is(se, sentinel) {
		t.Error("errors.Is should find the wrapped sentinel error")
	}
}

func TestServiceError_ErrorsAs(t *testing.T) {
	original := fmt.Errorf("some error")
	wrapped := WrapError("MySvc", "MyOp", original)

	var se *ServiceError
	if !errors.As(wrapped, &se) {
		t.Fatal("errors.As should find *ServiceError")
	}
	if se.Service != "MySvc" {
		t.Errorf("Service = %q, want %q", se.Service, "MySvc")
	}
	if se.Operation != "MyOp" {
		t.Errorf("Operation = %q, want %q", se.Operation, "MyOp")
	}
}

func TestWrapError_NilError(t *testing.T) {
	result := WrapError("Svc", "Op", nil)
	if result != nil {
		t.Errorf("WrapError with nil err should return nil, got %v", result)
	}
}

func TestWrapError_NonNilError(t *testing.T) {
	original := fmt.Errorf("something failed")
	result := WrapError("DataSource", "Import", original)

	if result == nil {
		t.Fatal("WrapError with non-nil err should return non-nil")
	}

	se, ok := result.(*ServiceError)
	if !ok {
		t.Fatal("WrapError should return *ServiceError")
	}
	if se.Service != "DataSource" {
		t.Errorf("Service = %q, want %q", se.Service, "DataSource")
	}
	if se.Operation != "Import" {
		t.Errorf("Operation = %q, want %q", se.Operation, "Import")
	}
	if se.Err != original {
		t.Error("Err should be the original error")
	}

	// Verify the formatted message contains service and operation
	msg := result.Error()
	if !strings.Contains(msg, "DataSource") || !strings.Contains(msg, "Import") {
		t.Errorf("Error message should contain service and operation: %q", msg)
	}
}
