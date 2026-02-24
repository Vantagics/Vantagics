package main

import (
	"context"
	"fmt"

	"vantagics/agent"
)

// PythonManager 定义 Python 环境管理接口
type PythonManager interface {
	GetPythonEnvironments() []agent.PythonEnvironment
	ValidatePython(path string) agent.PythonValidationResult
	InstallPythonPackages(pythonPath string, packages []string) error
	CreateVantagicsEnvironment() (string, error)
	CheckVantagicsEnvironmentExists() bool
	DiagnosePythonInstallation() map[string]interface{}
}

// PythonFacadeService Python 环境服务门面，封装所�?Python 相关的业务逻辑
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

// Initialize 初始�?Python 门面服务
func (s *PythonFacadeService) Initialize(ctx context.Context) error {
	s.ctx = ctx
	s.log("PythonFacadeService initialized")
	return nil
}

// Shutdown 关闭 Python 门面服务
func (s *PythonFacadeService) Shutdown() error {
	return nil
}

// SetContext 设置 Wails 上下�?
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

// GetPythonEnvironments 返回检测到�?Python 环境列表
func (s *PythonFacadeService) GetPythonEnvironments() []agent.PythonEnvironment {
	if s.pythonService == nil {
		s.log("[PYTHON] python service not available")
		return nil
	}
	return s.pythonService.ProbePythonEnvironments()
}

// ValidatePython 验证指定路径�?Python 环境
func (s *PythonFacadeService) ValidatePython(path string) agent.PythonValidationResult {
	if s.pythonService == nil {
		return agent.PythonValidationResult{Valid: false, Version: "", MissingPackages: []string{}}
	}
	return s.pythonService.ValidatePythonEnvironment(path)
}

// InstallPythonPackages 为指�?Python 环境安装缺失的包
func (s *PythonFacadeService) InstallPythonPackages(pythonPath string, packages []string) error {
	if s.pythonService == nil {
		return WrapError("python", "InstallPythonPackages", fmt.Errorf("python service not initialized"))
	}
	return s.pythonService.InstallMissingPackages(pythonPath, packages)
}

// CreateVantagicsEnvironment 创建 Vantagics 专用虚拟环境
func (s *PythonFacadeService) CreateVantagicsEnvironment() (string, error) {
	if s.pythonService == nil {
		return "", WrapError("python", "CreateVantagicsEnvironment", fmt.Errorf("python service not initialized"))
	}
	return s.pythonService.CreateVantagicsEnvironment()
}

// CheckVantagicsEnvironmentExists 检�?Vantagics 环境是否已存�?
func (s *PythonFacadeService) CheckVantagicsEnvironmentExists() bool {
	if s.pythonService == nil {
		return false
	}
	return s.pythonService.CheckVantagicsEnvironmentExists()
}

// DiagnosePythonInstallation 提供 Python 安装的详细诊断信�?
func (s *PythonFacadeService) DiagnosePythonInstallation() map[string]interface{} {
	if s.pythonService == nil {
		return map[string]interface{}{"error": "python service not initialized"}
	}
	return s.pythonService.DiagnosePythonInstallation()
}

// SetupUvEnvironment 创建 uv 虚拟环境并安装必要的�?
func (s *PythonFacadeService) SetupUvEnvironment() (string, error) {
	if s.pythonService == nil {
		return "", WrapError("python", "SetupUvEnvironment", fmt.Errorf("python service not initialized"))
	}
	s.log("[PYTHON] Setting up uv virtual environment...")
	pythonPath, err := s.pythonService.SetupUvEnvironment()
	if err != nil {
		s.log(fmt.Sprintf("[PYTHON] uv environment setup failed: %v", err))
		return "", err
	}
	s.log(fmt.Sprintf("[PYTHON] uv environment ready: %s", pythonPath))
	return pythonPath, nil
}

// GetUvEnvironmentStatus 获取 uv 环境状�?
func (s *PythonFacadeService) GetUvEnvironmentStatus() agent.UvEnvironmentStatus {
	if s.pythonService == nil {
		return agent.UvEnvironmentStatus{}
	}
	return s.pythonService.GetUvEnvironmentStatus()
}
