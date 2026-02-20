package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"vantagedata/agent"
	"vantagedata/config"
	"vantagedata/i18n"
)

// LicenseManager 定义许可证管理接口
type LicenseManager interface {
	ActivateLicense(serverURL, sn string) (*ActivationResult, error)
	DeactivateLicense() error
	RefreshLicense() (*ActivationResult, error)
	RequestSN(serverURL, email string) (*RequestSNResult, error)
	RequestFreeSN(serverURL, email string) (*RequestSNResult, error)
	RequestOpenSourceSN(serverURL, email string) (*RequestSNResult, error)
	GetActivationStatus() map[string]interface{}
	LoadSavedActivation(sn string) (*ActivationResult, error)
	CheckLicenseActivationFailed() bool
	GetLicenseActivationError() string
	GetActivatedLLMConfig() *agent.ActivationData
	IsLicenseActivated() bool
	IsPermanentFreeMode() bool
	IsOpenSourceMode() bool
	HasActiveAnalysis() bool
}

// LicenseFacadeService 许可证服务门面，封装所有许可证相关的业务逻辑和并发状态
type LicenseFacadeService struct {
	ctx            context.Context
	configProvider ConfigProvider
	configPersister ConfigPersister
	logger         func(string)

	// License client for activation
	licenseClient *agent.LicenseClient

	// 并发状态（从 App 迁移过来）
	licenseActivationFailed bool
	licenseActivationError  string
	mu                      sync.RWMutex

	// Dependencies injected after construction
	chatFacadeService      *ChatFacadeService
	reinitializeServicesFn func(cfg config.Config)
}

// NewLicenseFacadeService 创建新的 LicenseFacadeService 实例
func NewLicenseFacadeService(
	configProvider ConfigProvider,
	configPersister ConfigPersister,
	logger func(string),
) *LicenseFacadeService {
	return &LicenseFacadeService{
		configProvider:  configProvider,
		configPersister: configPersister,
		logger:          logger,
	}
}

// Name 返回服务名称
func (l *LicenseFacadeService) Name() string {
	return "license"
}

// Initialize 初始化许可证门面服务
func (l *LicenseFacadeService) Initialize(ctx context.Context) error {
	l.ctx = ctx
	l.log("LicenseFacadeService initialized")
	return nil
}

// Shutdown 关闭许可证门面服务
func (l *LicenseFacadeService) Shutdown() error {
	l.log("LicenseFacadeService shutdown")
	return nil
}

// SetContext 设置 Wails 上下文
func (l *LicenseFacadeService) SetContext(ctx context.Context) {
	l.ctx = ctx
}

// SetChatFacadeService 注入聊天门面服务依赖（用于 HasActiveAnalysis 检查）
func (l *LicenseFacadeService) SetChatFacadeService(cfs *ChatFacadeService) {
	l.chatFacadeService = cfs
}

// SetReinitializeServicesFn 注入服务重新初始化回调
func (l *LicenseFacadeService) SetReinitializeServicesFn(fn func(cfg config.Config)) {
	l.reinitializeServicesFn = fn
}

// SetLicenseClient 设置许可证客户端
func (l *LicenseFacadeService) SetLicenseClient(lc *agent.LicenseClient) {
	l.licenseClient = lc
}

// GetLicenseClient 返回许可证客户端实例
func (l *LicenseFacadeService) GetLicenseClient() *agent.LicenseClient {
	return l.licenseClient
}

// ActivateLicense activates the application with a license server
func (l *LicenseFacadeService) ActivateLicense(serverURL, sn string) (*ActivationResult, error) {
	if l.licenseClient == nil {
		l.licenseClient = agent.NewLicenseClient(l.log)
	}

	result, err := l.licenseClient.Activate(serverURL, sn)
	if err != nil {
		return &ActivationResult{
			Success: false,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}, nil
	}

	if !result.Success {
		return &ActivationResult{
			Success: false,
			Code:    result.Code,
			Message: result.Message,
		}, nil
	}

	// Save encrypted activation data to local storage
	if err := l.licenseClient.SaveActivationData(); err != nil {
		l.log(fmt.Sprintf("[LICENSE] Warning: Failed to save activation data: %v", err))
	}

	// Save extra info to config file
	if result.Data != nil && result.Data.ExtraInfo != nil && len(result.Data.ExtraInfo) > 0 {
		cfg, err := l.configProvider.GetConfig()
		if err == nil {
			cfg.LicenseExtraInfo = result.Data.ExtraInfo
			if saveErr := l.configPersister.SaveConfig(cfg); saveErr != nil {
				l.log(fmt.Sprintf("[LICENSE] Warning: Failed to save extra info to config: %v", saveErr))
			} else {
				l.log(fmt.Sprintf("[LICENSE] Saved %d extra info items to config", len(result.Data.ExtraInfo)))
			}
		}
	}

	// Reinitialize services with the new license configuration
	cfg, _ := l.configProvider.GetConfig()
	if l.reinitializeServicesFn != nil {
		l.reinitializeServicesFn(cfg)
	}

	return &ActivationResult{
		Success:   true,
		Code:      "SUCCESS",
		Message:   "激活成功",
		ExpiresAt: result.Data.ExpiresAt,
	}, nil
}

// RequestSN requests a serial number from the license server
func (l *LicenseFacadeService) RequestSN(serverURL, email string) (*RequestSNResult, error) {
	if l.licenseClient == nil {
		l.licenseClient = agent.NewLicenseClient(l.log)
	}

	result, err := l.licenseClient.RequestSN(serverURL, email)
	if err != nil {
		return &RequestSNResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &RequestSNResult{
		Success: result.Success,
		Message: result.Message,
		SN:      result.SN,
		Code:    result.Code,
	}, nil
}

// RequestFreeSN requests a permanent free serial number from the license server
func (l *LicenseFacadeService) RequestFreeSN(serverURL, email string) (*RequestSNResult, error) {
	if l.licenseClient == nil {
		l.licenseClient = agent.NewLicenseClient(l.log)
	}

	result, err := l.licenseClient.RequestFreeSN(serverURL, email)
	if err != nil {
		return &RequestSNResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &RequestSNResult{
		Success: result.Success,
		Message: result.Message,
		SN:      result.SN,
		Code:    result.Code,
	}, nil
}

// RequestOpenSourceSN requests an open source serial number from the license server
func (l *LicenseFacadeService) RequestOpenSourceSN(serverURL, email string) (*RequestSNResult, error) {
	if l.licenseClient == nil {
		l.licenseClient = agent.NewLicenseClient(l.log)
	}

	result, err := l.licenseClient.RequestOpenSourceSN(serverURL, email)
	if err != nil {
		return &RequestSNResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &RequestSNResult{
		Success: result.Success,
		Message: result.Message,
		SN:      result.SN,
		Code:    result.Code,
	}, nil
}

// GetActivationStatus returns the current activation status
func (l *LicenseFacadeService) GetActivationStatus() map[string]interface{} {
	l.mu.RLock()
	failed := l.licenseActivationFailed
	errorMsg := l.licenseActivationError
	l.mu.RUnlock()

	// Check if activation failed during startup
	if failed {
		return map[string]interface{}{
			"activated":         false,
			"activation_failed": true,
			"error_message":     errorMsg,
		}
	}

	if l.licenseClient == nil || !l.licenseClient.IsActivated() {
		return map[string]interface{}{
			"activated": false,
		}
	}

	data := l.licenseClient.GetData()
	count, limit, date := l.licenseClient.GetAnalysisStatus()
	totalCredits, usedCredits, isCreditsMode := l.licenseClient.GetCreditsStatus()
	cfg, _ := l.configProvider.GetConfig()

	return map[string]interface{}{
		"activated":            true,
		"expires_at":           data.ExpiresAt,
		"has_llm":              data.LLMAPIKey != "",
		"has_search":           data.SearchAPIKey != "",
		"llm_type":             data.LLMType,
		"search_type":          data.SearchType,
		"sn":                   l.licenseClient.GetSN(),
		"server_url":           l.licenseClient.GetServerURL(),
		"daily_analysis_limit": limit,
		"daily_analysis_count": count,
		"daily_analysis_date":  date,
		"trust_level":          data.TrustLevel,
		"refresh_interval":     data.RefreshInterval,
		"total_credits":        totalCredits,
		"used_credits":         usedCredits,
		"credits_mode":         isCreditsMode,
		"email":                cfg.LicenseEmail,
		"is_permanent_free":    data.TrustLevel == "permanent_free",
		"is_open_source":       data.TrustLevel == "open_source",
	}
}

// CheckLicenseActivationFailed returns true if license activation failed during startup
func (l *LicenseFacadeService) CheckLicenseActivationFailed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.licenseActivationFailed
}

// GetLicenseActivationError returns the license activation error message
func (l *LicenseFacadeService) GetLicenseActivationError() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.licenseActivationError
}

// LoadSavedActivation attempts to load saved activation data from local storage
func (l *LicenseFacadeService) LoadSavedActivation(sn string) (*ActivationResult, error) {
	if l.licenseClient == nil {
		l.licenseClient = agent.NewLicenseClient(l.log)
	}

	err := l.licenseClient.LoadActivationData(sn)
	if err != nil {
		return &ActivationResult{
			Success: false,
			Code:    "LOAD_FAILED",
			Message: err.Error(),
		}, nil
	}

	data := l.licenseClient.GetData()
	return &ActivationResult{
		Success:   true,
		Code:      "SUCCESS",
		Message:   "从本地加载激活数据成功",
		ExpiresAt: data.ExpiresAt,
	}, nil
}

// GetActivatedLLMConfig returns the LLM config from activation (for internal use)
func (l *LicenseFacadeService) GetActivatedLLMConfig() *agent.ActivationData {
	if l.licenseClient == nil || !l.licenseClient.IsActivated() {
		return nil
	}
	return l.licenseClient.GetData()
}

// IsPermanentFreeMode returns true if the current activation has trust_level "permanent_free"
func (l *LicenseFacadeService) IsPermanentFreeMode() bool {
	if l.licenseClient == nil || !l.licenseClient.IsActivated() {
		return false
	}
	data := l.licenseClient.GetData()
	return data != nil && data.TrustLevel == "permanent_free"
}

// IsOpenSourceMode returns true if the current activation has trust_level "open_source"
func (l *LicenseFacadeService) IsOpenSourceMode() bool {
	if l.licenseClient == nil || !l.licenseClient.IsActivated() {
		return false
	}
	data := l.licenseClient.GetData()
	return data != nil && data.TrustLevel == "open_source"
}

// IsLicenseActivated returns true if license is activated
func (l *LicenseFacadeService) IsLicenseActivated() bool {
	return l.licenseClient != nil && l.licenseClient.IsActivated()
}

// HasActiveAnalysis checks if there are any active analysis sessions
func (l *LicenseFacadeService) HasActiveAnalysis() bool {
	if l.chatFacadeService == nil {
		return false
	}
	return l.chatFacadeService.HasActiveAnalysis()
}

// DeactivateLicense clears the activation
func (l *LicenseFacadeService) DeactivateLicense() error {
	// Check if there are active analysis sessions
	if l.HasActiveAnalysis() {
		cfg, _ := l.configProvider.GetConfig()
		if cfg.Language == "简体中文" {
			return fmt.Errorf("当前有正在进行的分析任务，无法切换模式")
		}
		return fmt.Errorf("cannot switch mode while analysis is in progress")
	}

	if l.licenseClient != nil {
		l.licenseClient.ClearSavedData()
	}

	// Clear license info from config
	cfg, err := l.configProvider.GetConfig()
	if err == nil {
		cfg.LicenseExtraInfo = nil
		cfg.LicenseSN = ""
		cfg.LicenseServerURL = ""
		cfg.LicenseEmail = ""
		if saveErr := l.configPersister.SaveConfig(cfg); saveErr != nil {
			l.log(fmt.Sprintf("[LICENSE] Warning: Failed to clear license info from config: %v", saveErr))
		} else {
			l.log("[LICENSE] Cleared license info from config")
		}
	}

	// Reset activation failed flag
	l.mu.Lock()
	l.licenseActivationFailed = false
	l.licenseActivationError = ""
	l.mu.Unlock()

	return nil
}

// RefreshLicense refreshes the license from server using stored SN
func (l *LicenseFacadeService) RefreshLicense() (*ActivationResult, error) {
	if l.licenseClient == nil || !l.licenseClient.IsActivated() {
		return &ActivationResult{
			Success: false,
			Code:    "NOT_ACTIVATED",
			Message: "未激活，无法刷新",
		}, nil
	}

	sn := l.licenseClient.GetSN()
	if sn == "" {
		return &ActivationResult{
			Success: false,
			Code:    "NO_SN",
			Message: "未找到序列号",
		}, nil
	}

	serverURL := l.licenseClient.GetServerURL()
	if serverURL == "" {
		// Try from config
		cfg, _ := l.configProvider.GetConfig()
		serverURL = cfg.LicenseServerURL
	}
	if serverURL == "" {
		return &ActivationResult{
			Success: false,
			Code:    "NO_SERVER",
			Message: "未找到授权服务器地址",
		}, nil
	}

	l.log(fmt.Sprintf("[LICENSE] Refreshing license with SN: %s, Server: %s", sn, serverURL))

	// Re-activate with the same SN
	result, err := l.licenseClient.Activate(serverURL, sn)
	if err != nil {
		l.log(fmt.Sprintf("[LICENSE] Refresh failed: %v", err))
		return &ActivationResult{
			Success: false,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("刷新失败: %v", err),
		}, nil
	}

	if !result.Success {
		l.log(fmt.Sprintf("[LICENSE] Refresh failed: %s (code: %s)", result.Message, result.Code))

		// Check if the license was disabled, deleted, or invalidated on server
		// In these cases, switch to open source mode
		if result.Code == "INVALID_SN" || result.Code == "SN_EXPIRED" || result.Code == "SN_DISABLED" {
			l.log(fmt.Sprintf("[LICENSE] License is no longer valid (code: %s), switching to open source mode", result.Code))

			// Clear license data
			if err := l.licenseClient.ClearSavedData(); err != nil {
				l.log(fmt.Sprintf("[LICENSE] Warning: Failed to clear saved license data: %v", err))
			}
			l.licenseClient.Clear()

			// Clear license info from config
			cfg, _ := l.configProvider.GetConfig()
			cfg.LicenseSN = ""
			cfg.LicenseServerURL = ""
			l.configPersister.SaveConfig(cfg)

			// Reinitialize services with user's own config (open source mode)
			if l.reinitializeServicesFn != nil {
				l.reinitializeServicesFn(cfg)
			}

			// Return with switched_to_oss flag
			var message string
			switch result.Code {
			case "INVALID_SN":
				message = "序列号无效，已切换到开源软件模式。请使用您自己的 LLM API 配置。"
			case "SN_EXPIRED":
				message = "序列号已过期，已切换到开源软件模式。请使用您自己的 LLM API 配置。"
			case "SN_DISABLED":
				message = "序列号已被禁用，已切换到开源软件模式。请使用您自己的 LLM API 配置。"
			default:
				message = "授权已失效，已切换到开源软件模式。请使用您自己的 LLM API 配置。"
			}

			return &ActivationResult{
				Success:       false,
				Code:          result.Code,
				Message:       message,
				SwitchedToOSS: true,
			}, nil
		}

		return &ActivationResult{
			Success: false,
			Code:    result.Code,
			Message: fmt.Sprintf("刷新失败: %s", result.Message),
		}, nil
	}

	// Save updated activation data
	if err := l.licenseClient.SaveActivationData(); err != nil {
		l.log(fmt.Sprintf("[LICENSE] Warning: Failed to save refreshed data: %v", err))
	}

	// Reinitialize services with updated config
	cfg, _ := l.configProvider.GetConfig()
	if l.reinitializeServicesFn != nil {
		l.reinitializeServicesFn(cfg)
	}

	l.log(fmt.Sprintf("[LICENSE] License refreshed successfully, expires: %s", result.Data.ExpiresAt))

	return &ActivationResult{
		Success:   true,
		Code:      "SUCCESS",
		Message:   "授权刷新成功",
		ExpiresAt: result.Data.ExpiresAt,
	}, nil
}

// tryLicenseActivationWithRetry attempts license activation with exponential backoff retry.
// operation should be "Activation" or "Refresh" for logging purposes.
func (l *LicenseFacadeService) tryLicenseActivationWithRetry(cfg *config.Config, operation string) {
	const maxRetries = 10
	const maxBackoff = 60 * time.Second

	success := false
	var lastErr error
	var serverRejected bool
	var rejectionCode string

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			backoff := time.Duration(1<<retry) * time.Second
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			l.log(fmt.Sprintf("[STARTUP] %s retry %d/%d in %v...", operation, retry+1, maxRetries, backoff))
			time.Sleep(backoff)
		}
		result, err := l.licenseClient.Activate(cfg.LicenseServerURL, cfg.LicenseSN)
		if err != nil {
			lastErr = err
			l.log(fmt.Sprintf("[STARTUP] %s attempt %d failed: %v", operation, retry+1, err))
			continue
		}
		if !result.Success {
			lastErr = fmt.Errorf("%s", result.Message)
			l.log(fmt.Sprintf("[STARTUP] %s attempt %d rejected: %s (code: %s)", operation, retry+1, result.Message, result.Code))
			if result.Code == "INVALID_SN" || result.Code == "SN_EXPIRED" || result.Code == "SN_DISABLED" {
				serverRejected = true
				rejectionCode = result.Code
				break
			}
			continue
		}
		success = true
		l.log(fmt.Sprintf("[STARTUP] License %s successful", strings.ToLower(operation)))
		if saveErr := l.licenseClient.SaveActivationData(); saveErr != nil {
			l.log(fmt.Sprintf("[STARTUP] Warning: failed to save %s data: %v", strings.ToLower(operation), saveErr))
		}
		break
	}

	if !success {
		if serverRejected {
			l.log(fmt.Sprintf("[STARTUP] License rejected by server (code: %s), switching to open source mode", rejectionCode))
			l.licenseClient.ClearSavedData()
			l.licenseClient.Clear()
			cfg.LicenseSN = ""
			cfg.LicenseServerURL = ""
			l.configPersister.SaveConfig(*cfg)
		} else {
			l.log(fmt.Sprintf("[STARTUP] FATAL: License %s failed after %d retries: %v", strings.ToLower(operation), maxRetries, lastErr))
			l.mu.Lock()
			l.licenseActivationFailed = true
			i18nKey := "app.license_activation_failed"
			if operation == "Refresh" {
				i18nKey = "app.license_refresh_failed"
			}
			l.licenseActivationError = i18n.T(i18nKey, lastErr)
			l.mu.Unlock()
		}
	}
}

// log 记录日志
func (l *LicenseFacadeService) log(msg string) {
	if l.logger != nil {
		l.logger(msg)
	}
}
