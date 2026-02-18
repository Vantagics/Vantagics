package main

import (
	"context"
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// Feature: main-architecture-refactor, Property 1: 服务注册与检索一致性
//
// For any set of services implementing the Service interface, after registering them
// into a ServiceRegistry, Get(name) should retrieve each registered service instance,
// and the returned instance should be identical to the one registered.
//
// **Validates: Requirements 2.1, 7.3**

func TestProperty1_ServiceRegistrationRetrievalConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 50).Draw(t, "serviceCount")

		logger, _ := newTestLogger()
		reg := NewServiceRegistry(context.Background(), logger)

		// Generate unique service names and register them
		services := make([]*mockService, count)
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("svc-%d", i)
			svc := &mockService{name: name}
			services[i] = svc

			err := reg.Register(svc)
			if err != nil {
				t.Fatalf("Register(%q) failed: %v", name, err)
			}
		}

		// Property: every registered service is retrievable by name and is the exact same instance
		for i, svc := range services {
			name := fmt.Sprintf("svc-%d", i)
			got, ok := reg.Get(name)
			if !ok {
				t.Fatalf("Get(%q) returned false, expected service to be found", name)
			}
			if got != svc {
				t.Fatalf("Get(%q) returned different instance", name)
			}
		}

		// Property: unregistered names are not found
		_, ok := reg.Get(fmt.Sprintf("svc-%d", count))
		if ok {
			t.Fatalf("Get for unregistered name should return false")
		}
	})
}

// Feature: main-architecture-refactor, Property 2: 初始化依赖顺序
//
// For any list of services registered in order, after calling InitializeAll(),
// each service's Initialize() should be called in the same order as registration
// (i.e., dependencies registered first are initialized first).
//
// **Validates: Requirements 2.2**

func TestProperty2_InitializationDependencyOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 50).Draw(t, "serviceCount")

		logger, _ := newTestLogger()
		reg := NewServiceRegistry(context.Background(), logger)

		var initOrder []string
		names := make([]string, count)

		for i := 0; i < count; i++ {
			name := fmt.Sprintf("svc-%d", i)
			names[i] = name
			svc := &mockService{name: name, initOrder: &initOrder}
			if err := reg.Register(svc); err != nil {
				t.Fatalf("Register(%q) failed: %v", name, err)
			}
		}

		err := reg.InitializeAll()
		if err != nil {
			t.Fatalf("InitializeAll failed: %v", err)
		}

		// Property: initialization order matches registration order
		if len(initOrder) != count {
			t.Fatalf("expected %d initializations, got %d", count, len(initOrder))
		}
		for i, name := range names {
			if initOrder[i] != name {
				t.Fatalf("init order[%d] = %q, want %q", i, initOrder[i], name)
			}
		}
	})
}

// Feature: main-architecture-refactor, Property 3: 非关键服务故障优雅降级
//
// For any registry containing a mix of critical and non-critical services, when a
// non-critical service's Initialize() returns an error, InitializeAll() should continue
// initializing remaining services and return nil. When a critical service's Initialize()
// returns an error, InitializeAll() should return an error.
//
// **Validates: Requirements 2.4**

func TestProperty3_NonCriticalServiceGracefulDegradation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(2, 30).Draw(t, "serviceCount")
		// Pick a random subset of indices to be non-critical failures
		failCount := rapid.IntRange(1, count).Draw(t, "failCount")

		logger, _ := newTestLogger()
		reg := NewServiceRegistry(context.Background(), logger)

		var initOrder []string
		failIndices := make(map[int]bool)

		// Select unique fail indices
		for len(failIndices) < failCount {
			idx := rapid.IntRange(0, count-1).Draw(t, fmt.Sprintf("failIdx-%d", len(failIndices)))
			failIndices[idx] = true
		}

		for i := 0; i < count; i++ {
			name := fmt.Sprintf("svc-%d", i)
			svc := &mockService{name: name, initOrder: &initOrder}
			if failIndices[i] {
				svc.initErr = fmt.Errorf("init error for %s", name)
			}
			// All registered as non-critical
			if err := reg.Register(svc); err != nil {
				t.Fatalf("Register(%q) failed: %v", name, err)
			}
		}

		// Property: InitializeAll succeeds when only non-critical services fail
		err := reg.InitializeAll()
		if err != nil {
			t.Fatalf("InitializeAll should succeed with only non-critical failures, got: %v", err)
		}

		// Property: all services were attempted (including failing ones)
		if len(initOrder) != count {
			t.Fatalf("expected %d init attempts, got %d", count, len(initOrder))
		}
	})

	// Sub-property: critical service failure causes InitializeAll to return error
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(2, 30).Draw(t, "serviceCount")
		criticalIdx := rapid.IntRange(0, count-1).Draw(t, "criticalIdx")

		logger, _ := newTestLogger()
		reg := NewServiceRegistry(context.Background(), logger)

		var initOrder []string

		for i := 0; i < count; i++ {
			name := fmt.Sprintf("svc-%d", i)
			svc := &mockService{name: name, initOrder: &initOrder}
			if i == criticalIdx {
				svc.initErr = fmt.Errorf("critical init error")
				if err := reg.RegisterCritical(svc); err != nil {
					t.Fatalf("RegisterCritical(%q) failed: %v", name, err)
				}
			} else {
				if err := reg.Register(svc); err != nil {
					t.Fatalf("Register(%q) failed: %v", name, err)
				}
			}
		}

		// Property: InitializeAll returns error when a critical service fails
		err := reg.InitializeAll()
		if err == nil {
			t.Fatalf("InitializeAll should return error when critical service fails")
		}

		// Property: services after the critical failure are not initialized
		if len(initOrder) != criticalIdx+1 {
			t.Fatalf("expected %d init attempts (up to and including critical), got %d", criticalIdx+1, len(initOrder))
		}
	})
}

// Feature: main-architecture-refactor, Property 4: 关闭逆序
//
// For any list of registered services, calling ShutdownAll() should invoke each
// service's Shutdown() in the exact reverse order of registration.
//
// **Validates: Requirements 2.5**

func TestProperty4_ShutdownReverseOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 50).Draw(t, "serviceCount")

		logger, _ := newTestLogger()
		reg := NewServiceRegistry(context.Background(), logger)

		var shutdownOrder []string
		names := make([]string, count)

		for i := 0; i < count; i++ {
			name := fmt.Sprintf("svc-%d", i)
			names[i] = name
			svc := &mockService{name: name, shutdownOrder: &shutdownOrder}
			if err := reg.Register(svc); err != nil {
				t.Fatalf("Register(%q) failed: %v", name, err)
			}
		}

		reg.ShutdownAll()

		// Property: shutdown order is exact reverse of registration order
		if len(shutdownOrder) != count {
			t.Fatalf("expected %d shutdowns, got %d", count, len(shutdownOrder))
		}
		for i := 0; i < count; i++ {
			expected := names[count-1-i]
			if shutdownOrder[i] != expected {
				t.Fatalf("shutdown order[%d] = %q, want %q (reverse of registration)", i, shutdownOrder[i], expected)
			}
		}
	})
}
