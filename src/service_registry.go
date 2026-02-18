package main

import (
	"context"
	"fmt"
	"sync"
)

// Service 定义所有服务必须实现的生命周期接口
type Service interface {
	// Name 返回服务名称，用于日志和错误信息
	Name() string
	// Initialize 初始化服务，在所有依赖注入完成后调用
	Initialize(ctx context.Context) error
	// Shutdown 关闭服务，释放资源
	Shutdown() error
}

// serviceEntry 内部使用的服务元数据
type serviceEntry struct {
	service  Service
	name     string
	critical bool // 是否为关键服务（失败则阻止启动）
}

// ServiceRegistry 集中管理所有服务实例
type ServiceRegistry struct {
	ctx      context.Context
	logger   func(string)
	services []serviceEntry      // 按注册顺序存储
	byName   map[string]Service  // 按名称索引
	mu       sync.RWMutex
}

// NewServiceRegistry 创建新的服务注册表
func NewServiceRegistry(ctx context.Context, logger func(string)) *ServiceRegistry {
	return &ServiceRegistry{
		ctx:      ctx,
		logger:   logger,
		services: make([]serviceEntry, 0),
		byName:   make(map[string]Service),
	}
}

// Register 注册一个非关键服务实例。重复名称返回错误。
func (r *ServiceRegistry) Register(svc Service) error {
	return r.register(svc, false)
}

// RegisterCritical 注册一个关键服务实例。关键服务初始化失败将阻止应用启动。
func (r *ServiceRegistry) RegisterCritical(svc Service) error {
	return r.register(svc, true)
}

// register 内部注册方法
func (r *ServiceRegistry) register(svc Service, critical bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := svc.Name()
	if _, exists := r.byName[name]; exists {
		return WrapError("ServiceRegistry", "Register", fmt.Errorf("service %q already registered", name))
	}

	r.services = append(r.services, serviceEntry{
		service:  svc,
		name:     name,
		critical: critical,
	})
	r.byName[name] = svc
	return nil
}

// Get 按名称获取服务（类型断言由调用方负责）。线程安全。
func (r *ServiceRegistry) Get(name string) (Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, ok := r.byName[name]
	return svc, ok
}

// InitializeAll 按注册顺序初始化所有服务。
// 关键服务初始化失败时立即返回错误；非关键服务失败时记录日志并继续。
func (r *ServiceRegistry) InitializeAll() error {
	r.mu.RLock()
	entries := make([]serviceEntry, len(r.services))
	copy(entries, r.services)
	r.mu.RUnlock()

	for _, entry := range entries {
		if err := entry.service.Initialize(r.ctx); err != nil {
			if entry.critical {
				r.logger(fmt.Sprintf("Critical service %q failed to initialize: %v", entry.name, err))
				return WrapError("ServiceRegistry", "InitializeAll", fmt.Errorf("critical service %q failed: %w", entry.name, err))
			}
			r.logger(fmt.Sprintf("Non-critical service %q failed to initialize (degraded): %v", entry.name, err))
		}
	}
	return nil
}

// ShutdownAll 按注册的逆序关闭所有服务。
// 关闭过程中的错误会被记录但不会中断其他服务的关闭。
func (r *ServiceRegistry) ShutdownAll() {
	r.mu.RLock()
	entries := make([]serviceEntry, len(r.services))
	copy(entries, r.services)
	r.mu.RUnlock()

	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if err := entry.service.Shutdown(); err != nil {
			r.logger(fmt.Sprintf("Service %q shutdown error: %v", entry.name, err))
		}
	}
}
