package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// mockService is a test helper that implements the Service interface
type mockService struct {
	name          string
	initErr       error
	shutdownErr   error
	initCalled    bool
	shutdownCalled bool
	initOrder     *[]string // shared slice to track initialization order
	shutdownOrder *[]string // shared slice to track shutdown order
}

func (m *mockService) Name() string { return m.name }

func (m *mockService) Initialize(ctx context.Context) error {
	m.initCalled = true
	if m.initOrder != nil {
		*m.initOrder = append(*m.initOrder, m.name)
	}
	return m.initErr
}

func (m *mockService) Shutdown() error {
	m.shutdownCalled = true
	if m.shutdownOrder != nil {
		*m.shutdownOrder = append(*m.shutdownOrder, m.name)
	}
	return m.shutdownErr
}

func newTestLogger() (func(string), *[]string) {
	var logs []string
	return func(msg string) { logs = append(logs, msg) }, &logs
}

func TestNewServiceRegistry(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	if reg == nil {
		t.Fatal("NewServiceRegistry returned nil")
	}
	if reg.services == nil {
		t.Error("services slice should be initialized")
	}
	if reg.byName == nil {
		t.Error("byName map should be initialized")
	}
}

func TestRegister_Success(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	svc := &mockService{name: "test-service"}
	err := reg.Register(svc)
	if err != nil {
		t.Fatalf("Register returned unexpected error: %v", err)
	}

	got, ok := reg.Get("test-service")
	if !ok {
		t.Fatal("Get should find registered service")
	}
	if got != svc {
		t.Error("Get should return the same service instance")
	}
}

func TestRegister_DuplicateName(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	svc1 := &mockService{name: "dup"}
	svc2 := &mockService{name: "dup"}

	if err := reg.Register(svc1); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := reg.Register(svc2)
	if err == nil {
		t.Fatal("Register should return error for duplicate name")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("error should mention 'already registered', got: %v", err)
	}
}

func TestRegisterCritical_Success(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	svc := &mockService{name: "critical-svc"}
	err := reg.RegisterCritical(svc)
	if err != nil {
		t.Fatalf("RegisterCritical returned unexpected error: %v", err)
	}

	got, ok := reg.Get("critical-svc")
	if !ok {
		t.Fatal("Get should find critical service")
	}
	if got != svc {
		t.Error("Get should return the same service instance")
	}
}

func TestGet_NotFound(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Get should return false for unregistered service")
	}
}

func TestGet_ThreadSafe(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	svc := &mockService{name: "concurrent"}
	_ = reg.Register(svc)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, ok := reg.Get("concurrent")
			if !ok || got != svc {
				t.Errorf("concurrent Get failed")
			}
		}()
	}
	wg.Wait()
}

func TestInitializeAll_Order(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	var initOrder []string
	svc1 := &mockService{name: "first", initOrder: &initOrder}
	svc2 := &mockService{name: "second", initOrder: &initOrder}
	svc3 := &mockService{name: "third", initOrder: &initOrder}

	_ = reg.Register(svc1)
	_ = reg.Register(svc2)
	_ = reg.Register(svc3)

	err := reg.InitializeAll()
	if err != nil {
		t.Fatalf("InitializeAll returned unexpected error: %v", err)
	}

	expected := []string{"first", "second", "third"}
	if len(initOrder) != len(expected) {
		t.Fatalf("expected %d initializations, got %d", len(expected), len(initOrder))
	}
	for i, name := range expected {
		if initOrder[i] != name {
			t.Errorf("init order[%d] = %q, want %q", i, initOrder[i], name)
		}
	}
}

func TestInitializeAll_CriticalFailure(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	var initOrder []string
	svc1 := &mockService{name: "ok-service", initOrder: &initOrder}
	svcCritical := &mockService{name: "critical-fail", initErr: fmt.Errorf("critical init error"), initOrder: &initOrder}
	svc3 := &mockService{name: "after-critical", initOrder: &initOrder}

	_ = reg.Register(svc1)
	_ = reg.RegisterCritical(svcCritical)
	_ = reg.Register(svc3)

	err := reg.InitializeAll()
	if err == nil {
		t.Fatal("InitializeAll should return error when critical service fails")
	}
	if !strings.Contains(err.Error(), "critical-fail") {
		t.Errorf("error should mention the failing service, got: %v", err)
	}

	// svc3 should NOT have been initialized
	if svc3.initCalled {
		t.Error("services after critical failure should not be initialized")
	}
}

func TestInitializeAll_NonCriticalFailure(t *testing.T) {
	logger, logs := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	var initOrder []string
	svc1 := &mockService{name: "before", initOrder: &initOrder}
	svcNonCrit := &mockService{name: "non-critical-fail", initErr: fmt.Errorf("non-critical error"), initOrder: &initOrder}
	svc3 := &mockService{name: "after", initOrder: &initOrder}

	_ = reg.Register(svc1)
	_ = reg.Register(svcNonCrit) // non-critical by default
	_ = reg.Register(svc3)

	err := reg.InitializeAll()
	if err != nil {
		t.Fatalf("InitializeAll should succeed when only non-critical services fail, got: %v", err)
	}

	// All services should have been attempted
	if len(initOrder) != 3 {
		t.Errorf("expected 3 init attempts, got %d", len(initOrder))
	}

	// svc3 should still be initialized
	if !svc3.initCalled {
		t.Error("services after non-critical failure should still be initialized")
	}

	// Error should be logged
	found := false
	for _, log := range *logs {
		if strings.Contains(log, "non-critical-fail") && strings.Contains(log, "degraded") {
			found = true
			break
		}
	}
	if !found {
		t.Error("non-critical failure should be logged")
	}
}

func TestShutdownAll_ReverseOrder(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	var shutdownOrder []string
	svc1 := &mockService{name: "first", shutdownOrder: &shutdownOrder}
	svc2 := &mockService{name: "second", shutdownOrder: &shutdownOrder}
	svc3 := &mockService{name: "third", shutdownOrder: &shutdownOrder}

	_ = reg.Register(svc1)
	_ = reg.Register(svc2)
	_ = reg.Register(svc3)

	reg.ShutdownAll()

	expected := []string{"third", "second", "first"}
	if len(shutdownOrder) != len(expected) {
		t.Fatalf("expected %d shutdowns, got %d", len(expected), len(shutdownOrder))
	}
	for i, name := range expected {
		if shutdownOrder[i] != name {
			t.Errorf("shutdown order[%d] = %q, want %q", i, shutdownOrder[i], name)
		}
	}
}

func TestShutdownAll_ContinuesOnError(t *testing.T) {
	logger, logs := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	var shutdownOrder []string
	svc1 := &mockService{name: "first", shutdownOrder: &shutdownOrder}
	svc2 := &mockService{name: "failing", shutdownErr: fmt.Errorf("shutdown error"), shutdownOrder: &shutdownOrder}
	svc3 := &mockService{name: "third", shutdownOrder: &shutdownOrder}

	_ = reg.Register(svc1)
	_ = reg.Register(svc2)
	_ = reg.Register(svc3)

	reg.ShutdownAll()

	// All services should have been shut down despite the error
	if len(shutdownOrder) != 3 {
		t.Fatalf("expected 3 shutdowns, got %d", len(shutdownOrder))
	}

	// Error should be logged
	found := false
	for _, log := range *logs {
		if strings.Contains(log, "failing") && strings.Contains(log, "shutdown error") {
			found = true
			break
		}
	}
	if !found {
		t.Error("shutdown error should be logged")
	}
}

func TestShutdownAll_EmptyRegistry(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	// Should not panic
	reg.ShutdownAll()
}

func TestInitializeAll_EmptyRegistry(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	err := reg.InitializeAll()
	if err != nil {
		t.Fatalf("InitializeAll on empty registry should succeed, got: %v", err)
	}
}

func TestRegister_MixedCriticalAndNonCritical(t *testing.T) {
	logger, _ := newTestLogger()
	reg := NewServiceRegistry(context.Background(), logger)

	svc1 := &mockService{name: "critical-a"}
	svc2 := &mockService{name: "non-critical-b"}
	svc3 := &mockService{name: "critical-c"}

	_ = reg.RegisterCritical(svc1)
	_ = reg.Register(svc2)
	_ = reg.RegisterCritical(svc3)

	// All should be retrievable
	for _, name := range []string{"critical-a", "non-critical-b", "critical-c"} {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("Get(%q) should find the service", name)
		}
	}
}
