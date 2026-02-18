package main

import (
	"context"
	"fmt"

	"vantagedata/agent"
)

// PythonManager 定义 Python 环境管理接口
type PythonManager interface {
	GetPythonEnvironments() []agent.PythonEnvironment
	ValidatePython(path string) agent.PythonValidationResult
	InstallPythonPackages(pythonPath string, packages []string) error
	CreateVantageDataEnvironment() (string, error)
	CheckVantageDataEnvironmentExists() bool
	DiagnosePythonInstallation() map[string]interface{}
}

// PythonFacadeService Python 环境服务门面，封装所有 Python 相关的业务逻辑
type PythonFacadeService struct {
	ctx           context.Context
	pythonService *agent.PythonService
	logger        func(string)
}

// NewPythonFacadeService 创建新的 PythonFacadeService 实例
func NewPythonFacadeService(
	pythonService *agent.PythonService,
	logger func(string),
) *PythonFacadeService {
	return &PythonFacadeService{
		pythonService: pythonService,
		logger:        logger,
	}
}

// Name 返回服务名称
func (s *PythonFacadeService) Name() string {
	return "python"
}

// Initialize 初始化 Python 门面服务
func (s *PythonFacadeService) Initialize(ctx context.Context) error {
	s.ctx = ctx
	s.log("PythonFacadeService initialized")
	return nil
}

// Shutdown 关闭 Python 门面服务
func (s *PythonFacadeService) Shutdown() error {
	return nil
}

// SetContext 设置 Wails 上下文
func (s *PythonFacadeService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetPythonService 设置 PythonService（用于延迟注入或重新初始化）
func (s *PythonFacadeService) SetPythonService(ps *agent.PythonService) {
	s.pythonService = ps
}

// log 记录日志
func (s *PythonFacadeService) log(msg string) {
	if s.logger != nil {
		s.logger(msg)
	}
}

// --- Python Environment Methods ---

// GetPythonEnvironments 返回检测到的 Python 环境列表
func (s *PythonFacadeService) GetPythonEnvironments() []agent.PythonEnvironment {
	if s.pythonService == nil {
		s.log("[PYTHON] python service not available")
		return nil
	}
	return s.pythonService.ProbePythonEnvironments()
}

// ValidatePython 验证指定路径的 Python 环境
func (s *PythonFacadeService) ValidatePython(path string) agent.PythonValidationResult {
	if s.pythonService == nil {
		return agent.PythonValidationResult{Valid: false, Version: "", MissingPackages: []string{}}
	}
	return s.pythonService.ValidatePythonEnvironment(path)
}

// InstallPythonPackages 为指定 Python 环境安装缺失的包
func (s *PythonFacadeService) InstallPythonPackages(pythonPath string, packages []string) error {
	if s.pythonService == nil {
		return WrapError("python", "InstallPythonPackages", fmt.Errorf("python service not initialized"))
	}
	return s.pythonService.InstallMissingPackages(pythonPath, packages)
}

// CreateVantageDataEnvironment 创建 VantageData 专用虚拟环境
func (s *PythonFacadeService) CreateVantageDataEnvironment() (string, error) {
	if s.pythonService == nil {
		return "", WrapError("python", "CreateVantageDataEnvironment", fmt.Errorf("python service not initialized"))
	}
	return s.pythonService.CreateVantageDataEnvironment()
}

// CheckVantageDataEnvironmentExists 检查 VantageData 环境是否已存在
func (s *PythonFacadeService) CheckVantageDataEnvironmentExists() bool {
	if s.pythonService == nil {
		return false
	}
	return s.pythonService.CheckVantageDataEnvironmentExists()
}

// DiagnosePythonInstallation 提供 Python 安装的详细诊断信息
func (s *PythonFacadeService) DiagnosePythonInstallation() map[string]interface{} {
	if s.pythonService == nil {
		return map[string]interface{}{"error": "python service not initialized"}
	}
	return s.pythonService.DiagnosePythonInstallation()
}
