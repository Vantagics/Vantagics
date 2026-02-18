package main

import "fmt"

// ServiceError 统一的服务错误类型
type ServiceError struct {
	Service   string // 服务名称
	Operation string // 操作名称
	Err       error  // 原始错误
}

// Error 返回格式化的错误信息：[Service.Operation] error message
func (e *ServiceError) Error() string {
	return fmt.Sprintf("[%s.%s] %v", e.Service, e.Operation, e.Err)
}

// Unwrap 返回原始错误，支持 errors.Is/errors.As 链式查询
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// WrapError 创建带服务上下文的错误。如果 err 为 nil，返回 nil。
func WrapError(service, operation string, err error) error {
	if err == nil {
		return nil
	}
	return &ServiceError{Service: service, Operation: operation, Err: err}
}
