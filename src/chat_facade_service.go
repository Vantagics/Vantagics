package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	gort "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"vantagics/agent"
	"vantagics/config"
	"vantagics/i18n"

	"github.com/cloudwego/eino/schema"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ChatManager 定义聊天管理接口
type ChatManager interface {
	GetChatHistory() ([]ChatThread, error)
	GetChatHistoryByDataSource(dataSourceID string) ([]ChatThread, error)
	CreateChatThread(dataSourceID, title string) (ChatThread, error)
	DeleteThread(threadID string) error
	SendMessage(threadID, message, userMessageID, requestID string) (string, error)
	SendFreeChatMessage(threadID, message, userMessageID string) (string, error)
	CancelAnalysis() error
	ClearHistory() error
	ClearThreadMessages(threadID string) error
	UpdateThreadTitle(threadID, newTitle string) (string, error)
	CheckSessionNameExists(dataSourceID, sessionName, excludeThreadID string) (bool, error)
	SaveChatHistory(threads []ChatThread) error
	GetSessionFiles(threadID string) ([]SessionFile, error)
	GetSessionFilePath(threadID, fileName string) (string, error)
	OpenSessionFile(threadID, fileName string) error
	DeleteSessionFile(threadID, fileName string) error
	OpenSessionResultsDirectory(threadID string) error
}

// ChatFacadeService 聊天服务门面，封装所有聊天相关的业务逻辑和并发状�
type ChatFacadeService struct {
	ctx             context.Context
	chatService     *ChatService
	configProvider  ConfigProvider
	einoService     *agent.EinoService
	eventAggregator *EventAggregator
	logger          func(string)

	// 并发状态（�App 迁移过来�
	activeThreads       map[string]bool
	activeThreadsMutex  sync.RWMutex
	cancelAnalysisMutex sync.Mutex // Protects both cancelAnalysis and activeThreadID
	cancelAnalysis      bool
	activeThreadID      string
	isChatOpen          bool

	// Dependencies for SendMessage/SendFreeChatMessage
	licenseClient         *agent.LicenseClient
	searchKeywordsManager *agent.SearchKeywordsManager

	// saveChartDataToFile helper (injected from App)
	saveChartDataToFileFn func(threadID, chartType, data string) (string, error)
}

// NewChatFacadeService 创建新的 ChatFacadeService 实例
func NewChatFacadeService(
	chatService *ChatService,
	configProvider ConfigProvider,
	einoService *agent.EinoService,
	eventAggregator *EventAggregator,
	logger func(string),
) *ChatFacadeService {
	return &ChatFacadeService{
		chatService:     chatService,
		configProvider:  configProvider,
		einoService:     einoService,
		eventAggregator: eventAggregator,
		logger:          logger,
		activeThreads:   make(map[string]bool),
		isChatOpen:      false,
	}
}

// Name 返回服务名称
func (c *ChatFacadeService) Name() string {
	return "chat"
}

// Initialize 初始化聊天门面服�
func (c *ChatFacadeService) Initialize(ctx context.Context) error {
	c.ctx = ctx
	if c.chatService == nil {
		return WrapError("chat", "Initialize", fmt.Errorf("chatService dependency is nil"))
	}
	c.log("ChatFacadeService initialized")
	return nil
}

// Shutdown 关闭聊天门面服务，取消所有活跃分�
func (c *ChatFacadeService) Shutdown() error {
	c.cancelAnalysisMutex.Lock()
	c.cancelAnalysis = true
	c.cancelAnalysisMutex.Unlock()
	c.log("ChatFacadeService shutdown")
	return nil
}

// SetContext sets the Wails runtime context
func (c *ChatFacadeService) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// SetLicenseClient sets the license client dependency
func (c *ChatFacadeService) SetLicenseClient(lc *agent.LicenseClient) {
	c.licenseClient = lc
}

// SetSearchKeywordsManager sets the search keywords manager dependency
func (c *ChatFacadeService) SetSearchKeywordsManager(skm *agent.SearchKeywordsManager) {
	c.searchKeywordsManager = skm
}

// SetDataSourceService sets the data source service dependency
func (c *ChatFacadeService) SetDataSourceService(dss *agent.DataSourceService) {
	// DataSourceService is used by some chat operations but not stored as a field yet
	// This is a placeholder for future use when full tool integration is needed
}

// SetSaveChartDataToFileFn sets the chart data save function
func (c *ChatFacadeService) SetSaveChartDataToFileFn(fn func(threadID, chartType, data string) (string, error)) {
	c.saveChartDataToFileFn = fn
}

// SetChatOpen 更新聊天打开状�
func (c *ChatFacadeService) SetChatOpen(isOpen bool) {
	c.isChatOpen = isOpen
}
// SetEinoService updates the EinoService reference (used during reinitializeServices)
func (c *ChatFacadeService) SetEinoService(es *agent.EinoService) {
	c.einoService = es
}

// --- Chat History Methods ---

// GetChatHistory 加载聊天历史
func (c *ChatFacadeService) GetChatHistory() ([]ChatThread, error) {
	if c.chatService == nil {
		return nil, WrapError("chat", "GetChatHistory", fmt.Errorf("chat service not initialized"))
	}
	return c.chatService.LoadThreads()
}

// GetChatHistoryByDataSource 加载特定数据源的聊天历史
func (c *ChatFacadeService) GetChatHistoryByDataSource(dataSourceID string) ([]ChatThread, error) {
	if c.chatService == nil {
		return nil, WrapError("chat", "GetChatHistoryByDataSource", fmt.Errorf("chat service not initialized"))
	}
	return c.chatService.GetThreadsByDataSource(dataSourceID)
}

// CheckSessionNameExists 检查会话名称是否已存在
func (c *ChatFacadeService) CheckSessionNameExists(dataSourceID, sessionName, excludeThreadID string) (bool, error) {
	if c.chatService == nil {
		return false, WrapError("chat", "CheckSessionNameExists", fmt.Errorf("chat service not initialized"))
	}
	return c.chatService.CheckSessionNameExists(dataSourceID, sessionName, excludeThreadID)
}

// SaveChatHistory 保存聊天历史
func (c *ChatFacadeService) SaveChatHistory(threads []ChatThread) error {
	if c.chatService == nil {
		return WrapError("chat", "SaveChatHistory", fmt.Errorf("chat service not initialized"))
	}
	return c.chatService.SaveThreads(threads)
}

// CreateChatThread 创建新的聊天线程
func (c *ChatFacadeService) CreateChatThread(dataSourceID, title string) (ChatThread, error) {
	if c.chatService == nil {
		return ChatThread{}, WrapError("chat", "CreateChatThread", fmt.Errorf("chat service not initialized"))
	}
	thread, err := c.chatService.CreateThread(dataSourceID, title)
	if err != nil {
		return ChatThread{}, err
	}
	return thread, nil
}

// DeleteThread 删除指定的聊天线�
func (c *ChatFacadeService) DeleteThread(threadID string) error {
	if c.chatService == nil {
		return WrapError("chat", "DeleteThread", fmt.Errorf("chat service not initialized"))
	}

	// Check if this thread is currently running analysis
	c.cancelAnalysisMutex.Lock()
	isActiveThread := c.activeThreadID == threadID

	c.activeThreadsMutex.RLock()
	isGenerating := c.activeThreads[threadID]
	c.activeThreadsMutex.RUnlock()

	if isActiveThread && isGenerating {
		c.cancelAnalysis = true
		c.log(fmt.Sprintf("[DELETE-THREAD] Cancelling ongoing analysis for thread: %s", threadID))
	}
	c.cancelAnalysisMutex.Unlock()

	// Wait a moment for cancellation to take effect if needed
	if isActiveThread && isGenerating {
		time.Sleep(100 * time.Millisecond)
		c.log("[DELETE-THREAD] Waited for analysis cancellation")
	}

	// Delete the thread
	err := c.chatService.DeleteThread(threadID)
	if err != nil {
		return err
	}

	// If the deleted thread was active, clear dashboard data
	if isActiveThread {
		c.log(fmt.Sprintf("[DELETE-THREAD] Clearing dashboard data for deleted active thread: %s", threadID))
		if c.eventAggregator != nil {
			c.eventAggregator.Clear(threadID)
		}
	}

	return nil
}

// UpdateThreadTitle 更新线程标题
func (c *ChatFacadeService) UpdateThreadTitle(threadID, newTitle string) (string, error) {
	if c.chatService == nil {
		return "", WrapError("chat", "UpdateThreadTitle", fmt.Errorf("chat service not initialized"))
	}
	return c.chatService.UpdateThreadTitle(threadID, newTitle)
}

// ClearHistory 清除所有聊天历�
func (c *ChatFacadeService) ClearHistory() error {
	if c.chatService == nil {
		return WrapError("chat", "ClearHistory", fmt.Errorf("chat service not initialized"))
	}

	// Check if there's an ongoing analysis and cancel it
	c.cancelAnalysisMutex.Lock()
	c.activeThreadsMutex.RLock()
	hasActiveAnalysis := len(c.activeThreads) > 0
	c.activeThreadsMutex.RUnlock()

	if hasActiveAnalysis {
		c.cancelAnalysis = true
		c.log("[CLEAR-HISTORY] Cancelling ongoing analysis before clearing history")
	}
	c.cancelAnalysisMutex.Unlock()

	if hasActiveAnalysis {
		time.Sleep(100 * time.Millisecond)
		c.log("[CLEAR-HISTORY] Waited for analysis cancellation")
	}

	err := c.chatService.ClearHistory()
	if err != nil {
		return err
	}

	c.log("[CLEAR-HISTORY] Clearing dashboard data after clearing all history")
	if c.eventAggregator != nil {
		c.eventAggregator.Clear("")
	}

	return nil
}

// ClearThreadMessages 清除线程中的所有消息但保留线程本身
func (c *ChatFacadeService) ClearThreadMessages(threadID string) error {
	if c.chatService == nil {
		return WrapError("chat", "ClearThreadMessages", fmt.Errorf("chat service not initialized"))
	}
	return c.chatService.ClearThreadMessages(threadID)
}

// --- Concurrent State Methods ---

// CancelAnalysis 取消当前正在运行的分�
func (c *ChatFacadeService) CancelAnalysis() error {
	c.cancelAnalysisMutex.Lock()

	c.activeThreadsMutex.RLock()
	hasActiveAnalysis := len(c.activeThreads) > 0
	c.activeThreadsMutex.RUnlock()

	if !hasActiveAnalysis {
		c.cancelAnalysisMutex.Unlock()
		return fmt.Errorf("no analysis is currently running")
	}

	c.cancelAnalysis = true
	c.log(fmt.Sprintf("[CANCEL] Analysis cancellation requested for thread: %s", c.activeThreadID))
	c.cancelAnalysisMutex.Unlock()

	// Wait for the analysis to actually stop (with timeout)
	maxWaitTime := 5 * time.Second
	checkInterval := 100 * time.Millisecond
	startTime := time.Now()

	for {
		c.activeThreadsMutex.RLock()
		stillActive := len(c.activeThreads) > 0
		c.activeThreadsMutex.RUnlock()

		if !stillActive {
			c.log("[CANCEL] Analysis successfully cancelled and cleaned up")
			return nil
		}

		if time.Since(startTime) > maxWaitTime {
			c.log("[CANCEL] Timeout waiting for analysis to stop, forcing cleanup")
			c.activeThreadsMutex.Lock()
			for threadID := range c.activeThreads {
				delete(c.activeThreads, threadID)
				c.log(fmt.Sprintf("[CANCEL] Force removed thread from activeThreads: %s", threadID))
			}
			c.activeThreadsMutex.Unlock()
			return nil
		}

		time.Sleep(checkInterval)
	}
}

// IsCancelRequested 检查是否已请求取消分析
func (c *ChatFacadeService) IsCancelRequested() bool {
	c.cancelAnalysisMutex.Lock()
	defer c.cancelAnalysisMutex.Unlock()
	return c.cancelAnalysis
}

// GetActiveThreadID 返回当前活跃的线�ID
func (c *ChatFacadeService) GetActiveThreadID() string {
	c.cancelAnalysisMutex.Lock()
	defer c.cancelAnalysisMutex.Unlock()
	return c.activeThreadID
}

// GetActiveAnalysisCount 返回当前活跃分析会话数量
func (c *ChatFacadeService) GetActiveAnalysisCount() int {
	c.activeThreadsMutex.RLock()
	defer c.activeThreadsMutex.RUnlock()
	return len(c.activeThreads)
}

// CanStartNewAnalysis 检查是否可以启动新的分�
func (c *ChatFacadeService) CanStartNewAnalysis() (bool, string) {
	cfg, _ := c.configProvider.GetConfig()
	maxConcurrent := cfg.MaxConcurrentAnalysis
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	if maxConcurrent > 10 {
		maxConcurrent = 10
	}

	c.activeThreadsMutex.RLock()
	activeCount := len(c.activeThreads)
	c.activeThreadsMutex.RUnlock()

	if activeCount >= maxConcurrent {
		var errorMessage string
		if cfg.Language == "简体中�" {
			errorMessage = i18n.T("analysis.max_concurrent", activeCount, maxConcurrent)
		} else {
			errorMessage = fmt.Sprintf("There are currently %d analysis sessions in progress (max concurrent: %d). Please wait for some analyses to complete before starting a new analysis, or increase the max concurrent analysis limit in settings.", activeCount, maxConcurrent)
		}
		return false, errorMessage
	}

	return true, ""
}

// HasActiveAnalysis 返回是否有活跃的分析
func (c *ChatFacadeService) HasActiveAnalysis() bool {
	c.activeThreadsMutex.RLock()
	defer c.activeThreadsMutex.RUnlock()
	return len(c.activeThreads) > 0
}

// --- Session File Methods ---

// GetSessionFiles 获取线程的会话文件列�
func (c *ChatFacadeService) GetSessionFiles(threadID string) ([]SessionFile, error) {
	if c.chatService == nil {
		return nil, WrapError("chat", "GetSessionFiles", fmt.Errorf("chat service not initialized"))
	}
	return c.chatService.GetSessionFiles(threadID)
}

// GetSessionFilePath 返回会话文件的完整路�
func (c *ChatFacadeService) GetSessionFilePath(threadID, fileName string) (string, error) {
	if c.chatService == nil {
		return "", WrapError("chat", "GetSessionFilePath", fmt.Errorf("chat service not initialized"))
	}

	filesDir := c.chatService.GetSessionFilesDirectory(threadID)
	filePath := filepath.Join(filesDir, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fileName)
	}

	return filePath, nil
}

// OpenSessionFile 在默认应用程序中打开会话文件
func (c *ChatFacadeService) OpenSessionFile(threadID, fileName string) error {
	filePath, err := c.GetSessionFilePath(threadID, fileName)
	if err != nil {
		return err
	}

	runtime.BrowserOpenURL(c.ctx, "file://"+filePath)
	return nil
}

// DeleteSessionFile 删除会话中的特定文件
func (c *ChatFacadeService) DeleteSessionFile(threadID, fileName string) error {
	if c.chatService == nil {
		return WrapError("chat", "DeleteSessionFile", fmt.Errorf("chat service not initialized"))
	}

	filePath, err := c.GetSessionFilePath(threadID, fileName)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		return err
	}

	threads, err := c.chatService.LoadThreads()
	if err != nil {
		return err
	}

	for _, t := range threads {
		if t.ID == threadID {
			var updatedFiles []SessionFile
			for _, f := range t.Files {
				if f.Name != fileName {
					updatedFiles = append(updatedFiles, f)
				}
			}
			t.Files = updatedFiles
			return c.chatService.SaveThreads([]ChatThread{t})
		}
	}

	return fmt.Errorf("thread not found")
}

// AssociateNewFilesWithMessage 将新创建的文件关联到特定消息
func (c *ChatFacadeService) AssociateNewFilesWithMessage(threadID, messageID string, existingFiles map[string]bool) error {
	if c.chatService == nil {
		return WrapError("chat", "AssociateNewFilesWithMessage", fmt.Errorf("chat service not initialized"))
	}

	sessionFiles, err := c.chatService.GetSessionFiles(threadID)
	if err != nil {
		return err
	}

	updated := false
	for i := range sessionFiles {
		if existingFiles[sessionFiles[i].Name] {
			continue
		}
		if sessionFiles[i].MessageID != "" {
			continue
		}
		sessionFiles[i].MessageID = messageID
		updated = true
		c.log(fmt.Sprintf("[SESSION] Associated file '%s' with message %s", sessionFiles[i].Name, messageID))
	}

	if updated {
		threads, err := c.chatService.LoadThreads()
		if err != nil {
			return err
		}
		for _, t := range threads {
			if t.ID == threadID {
				t.Files = sessionFiles
				return c.chatService.SaveThreads([]ChatThread{t})
			}
		}
		return fmt.Errorf("thread not found")
	}

	return nil
}

// OpenSessionResultsDirectory 在文件浏览器中打开会话结果目录
func (c *ChatFacadeService) OpenSessionResultsDirectory(threadID string) error {
	if c.chatService == nil {
		return WrapError("chat", "OpenSessionResultsDirectory", fmt.Errorf("chat service not initialized"))
	}

	if threadID == "" {
		return fmt.Errorf("thread ID cannot be empty")
	}
	for _, r := range threadID {
		if r < '0' || r > '9' {
			return fmt.Errorf("invalid thread ID format")
		}
	}

	sessionDir := c.chatService.GetSessionDirectory(threadID)

	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return fmt.Errorf("session directory does not exist")
	}

	var cmd *exec.Cmd
	switch gort.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", sessionDir)
		cmd.SysProcAttr = hiddenProcAttr()
	case "darwin":
		cmd = exec.Command("open", sessionDir)
	case "linux":
		cmd = exec.Command("xdg-open", sessionDir)
	default:
		return fmt.Errorf("unsupported platform: %s", gort.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}

	return nil
}

// --- Logging and Error Helpers ---

// LogChatToFile 将聊天内容记录到文件
func (c *ChatFacadeService) LogChatToFile(threadID, role, content string) {
	cfg, _ := c.configProvider.GetConfig()

	logPath := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "chat.log")

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		c.log(fmt.Sprintf("logChatToFile: Failed to create log directory: %v", err))
		return
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.log(fmt.Sprintf("logChatToFile: Failed to open log file %s: %v", logPath, err))
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] [%s]\n%s\n\n--------------------------------------------------\n\n", timestamp, role, content)
}

// SaveErrorToChatThread 将错误消息保存到聊天线程
func (c *ChatFacadeService) SaveErrorToChatThread(threadID, errorCode, message string) {
	if c.chatService == nil || threadID == "" {
		return
	}
	chatMsg := ChatMessage{
		ID:        fmt.Sprintf("error_%d", time.Now().UnixNano()),
		Role:      "assistant",
		Content:   fmt.Sprintf(i18n.T("analysis.error_format"), errorCode, message),
		Timestamp: time.Now().Unix(),
	}
	if err := c.chatService.AddMessage(threadID, chatMsg); err != nil {
		c.log(fmt.Sprintf("[ERROR] Failed to save error message to chat thread %s: %v", threadID, err))
	}
}

// SendMessage 发送消息到 AI 进行分析
func (c *ChatFacadeService) SendMessage(threadID, message, userMessageID, requestID string) (string, error) {
	if c.chatService == nil {
		return "", WrapError("chat", "SendMessage", fmt.Errorf("chat service not initialized"))
	}

	cfg, err := c.configProvider.GetEffectiveConfig()
	if err != nil {
		return "", err
	}

	startTotal := time.Now()

	// Log user message if threadID provided
	if threadID != "" && cfg.DetailedLog {
		c.LogChatToFile(threadID, "USER REQUEST", message)
	}

	// Save user message to thread file BEFORE processing
	if threadID != "" && userMessageID != "" {
		threads, err := c.chatService.LoadThreads()
		if err == nil {
			messageExists := false
			for _, t := range threads {
				if t.ID == threadID {
					for _, m := range t.Messages {
						if m.ID == userMessageID {
							messageExists = true
							c.log(fmt.Sprintf("[CHAT] User message already exists in thread: %s", userMessageID))
							break
						}
					}
					break
				}
			}

			if !messageExists {
				userMsg := ChatMessage{
					ID:        userMessageID,
					Role:      "user",
					Content:   message,
					Timestamp: time.Now().Unix(),
				}
				if err := c.chatService.AddMessage(threadID, userMsg); err != nil {
					c.log(fmt.Sprintf("[ERROR] Failed to save user message: %v", err))
				} else {
					c.log(fmt.Sprintf("[CHAT] Saved user message to thread: %s", userMessageID))
				}
			}
		}
	}

	// Wait for concurrent analysis slot if needed
	cfg, _ = c.configProvider.GetConfig()
	maxConcurrent := cfg.MaxConcurrentAnalysis
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	if maxConcurrent > 10 {
		maxConcurrent = 10
	}

	// Reset cancel flag at the start of new analysis to prevent stale cancel state
	// from previous cancelled analyses affecting this new request
	c.cancelAnalysisMutex.Lock()
	wasCancelled := c.cancelAnalysis
	c.cancelAnalysis = false
	c.cancelAnalysisMutex.Unlock()
	if wasCancelled {
		c.log(fmt.Sprintf("[DEBUG-CANCEL] Cleared stale cancelAnalysis flag (was true) for thread: %s", threadID))
	}

	waitStartTime := time.Now()
	maxWaitTime := 5 * time.Minute
	checkInterval := 500 * time.Millisecond
	notifiedWaiting := false

	c.activeThreadsMutex.RLock()
	activeCount := len(c.activeThreads)
	c.activeThreadsMutex.RUnlock()

	if activeCount >= maxConcurrent {
		c.log(fmt.Sprintf("[CONCURRENT] Need to wait for slot. Active: %d, Max: %d, Thread: %s", activeCount, maxConcurrent, threadID))

		if threadID != "" {
			c.log(fmt.Sprintf("[LOADING-DEBUG] Backend emitting chat-loading true (waiting) for threadId: %s", threadID))
			runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
				"loading":  true,
				"threadId": threadID,
			})
		}

		var waitMessage string
		if cfg.Language == "简体中�" {
			waitMessage = i18n.T("analysis.queue_wait", activeCount, maxConcurrent)
		} else {
			waitMessage = fmt.Sprintf("Waiting in analysis queue... (%d/%d tasks in progress)", activeCount, maxConcurrent)
		}
		runtime.EventsEmit(c.ctx, "analysis-queue-status", map[string]interface{}{
			"threadId": threadID,
			"status":   "waiting",
			"message":  waitMessage,
			"position": activeCount - maxConcurrent + 1,
		})
		notifiedWaiting = true
	}

	for {
		c.activeThreadsMutex.RLock()
		activeCount = len(c.activeThreads)
		c.activeThreadsMutex.RUnlock()

		if activeCount < maxConcurrent {
			if notifiedWaiting {
				c.log(fmt.Sprintf("[CONCURRENT] Slot available after waiting, proceeding with analysis for thread: %s", threadID))
				runtime.EventsEmit(c.ctx, "analysis-queue-status", map[string]interface{}{
					"threadId": threadID,
					"status":   "starting",
					"message":  i18n.T("general.processing"),
				})
			}
			break
		}

		if time.Since(waitStartTime) > maxWaitTime {
			c.log(fmt.Sprintf("[CONCURRENT] Timeout waiting for analysis slot for thread: %s", threadID))
			if threadID != "" {
				runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
					"loading":  false,
					"threadId": threadID,
				})
			}
			var errorMessage string
			if cfg.Language == "简体中�" {
				errorMessage = i18n.T("analysis.queue_timeout", time.Since(waitStartTime).Round(time.Second), activeCount)
			} else {
				errorMessage = fmt.Sprintf("Timeout waiting for analysis queue (waited %v). There are currently %d analysis tasks in progress. Please try again later.", time.Since(waitStartTime).Round(time.Second), activeCount)
			}
			if threadID != "" {
				c.SaveErrorToChatThread(threadID, "QUEUE_TIMEOUT", errorMessage)
			}
			return "", fmt.Errorf("%s", errorMessage)
		}

		if c.IsCancelRequested() {
			c.log(fmt.Sprintf("[CONCURRENT] Cancellation requested while waiting for slot, aborting for thread: %s", threadID))
			if threadID != "" {
				runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
					"loading":  false,
					"threadId": threadID,
				})
				c.SaveErrorToChatThread(threadID, "QUEUE_CANCELLED", i18n.T("analysis.cancelled_msg"))
			}
			return "", fmt.Errorf("analysis cancelled while waiting in queue")
		}

		if notifiedWaiting && int(time.Since(waitStartTime).Seconds())%5 == 0 {
			var waitMessage string
			waitedTime := time.Since(waitStartTime).Round(time.Second)
			if cfg.Language == "简体中�" {
				waitMessage = i18n.T("analysis.queue_wait_elapsed", waitedTime, activeCount, maxConcurrent)
			} else {
				waitMessage = fmt.Sprintf("Waiting in analysis queue... (waited %v, %d/%d tasks in progress)", waitedTime, activeCount, maxConcurrent)
			}
			runtime.EventsEmit(c.ctx, "analysis-queue-status", map[string]interface{}{
				"threadId": threadID,
				"status":   "waiting",
				"message":  waitMessage,
				"position": activeCount - maxConcurrent + 1,
			})
		}

		time.Sleep(checkInterval)
	}

	// Mark this thread as having active analysis
	c.activeThreadsMutex.Lock()
	c.activeThreads[threadID] = true
	c.activeThreadsMutex.Unlock()

	// Check license analysis limit before proceeding
	if c.licenseClient != nil && c.licenseClient.IsActivated() {
		canAnalyze, limitMsg := c.licenseClient.CanAnalyze()
		if !canAnalyze {
			c.activeThreadsMutex.Lock()
			delete(c.activeThreads, threadID)
			c.activeThreadsMutex.Unlock()

			if threadID != "" {
				runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
					"loading":  false,
					"threadId": threadID,
				})
				c.SaveErrorToChatThread(threadID, "LICENSE_LIMIT", limitMsg)
			}

			return "", fmt.Errorf("%s", limitMsg)
		}
		c.log("[LICENSE] Analysis limit check passed, count will be incremented on success")
	}

	// Notify frontend that loading has started
	if threadID != "" {
		c.log(fmt.Sprintf("[LOADING-DEBUG] Backend emitting chat-loading true for threadId: %s", threadID))
		runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
			"loading":  true,
			"threadId": threadID,
		})

		if c.eventAggregator != nil {
			c.log(fmt.Sprintf("[DASHBOARD] Setting loading state for thread: %s, requestId: %s", threadID, requestID))
			c.eventAggregator.SetLoading(threadID, true, requestID)
		}
	}

	defer func() {
		c.activeThreadsMutex.Lock()
		delete(c.activeThreads, threadID)
		c.activeThreadsMutex.Unlock()

		if threadID != "" {
			c.log(fmt.Sprintf("[LOADING-DEBUG] Backend emitting chat-loading false for threadId: %s", threadID))
			runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
				"loading":  false,
				"threadId": threadID,
			})
		}
	}()

	// Set active thread and reset cancel flag
	c.cancelAnalysisMutex.Lock()
	c.activeThreadID = threadID
	c.cancelAnalysis = false
	c.cancelAnalysisMutex.Unlock()

	// Check if we should use Eino (if thread has DataSourceID)
	var useEino bool
	var dataSourceID string
	if threadID != "" && c.einoService != nil {
		startCheck := time.Now()
		threads, _ := c.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID && t.DataSourceID != "" {
				useEino = true
				dataSourceID = t.DataSourceID
				break
			}
		}
		c.log(fmt.Sprintf("[TIMING] Checking Eino eligibility took: %v", time.Since(startCheck)))
	} else if threadID != "" && c.einoService == nil {
		c.log("[ERROR] EinoService is nil - cannot use advanced analysis features")
		if cfg, err := c.configProvider.GetConfig(); err == nil {
			c.log(fmt.Sprintf("[DEBUG] Current config - Provider: %s, Model: %s", cfg.LLMProvider, cfg.ModelName))
		}
	}

	if useEino {
		resp, err := c.runEinoAnalysis(threadID, message, userMessageID, requestID, dataSourceID, cfg)
		if err != nil {
			return "", err
		}
		c.log(fmt.Sprintf("[TIMING] Total SendMessage (Eino) took: %v", time.Since(startTotal)))
		return resp, nil
	}

	// Standard LLM fallback path
	langPrompt := getLangPromptFromMessage(message)
	fullMessage := fmt.Sprintf("%s\n\n(Please answer in %s)", message, langPrompt)

	llm := agent.NewLLMService(cfg, c.log)

	chatStartTime := time.Now()
	resp, err := llm.Chat(c.ctx, fullMessage)

	chatDuration := time.Since(chatStartTime)
	minutes := int(chatDuration.Minutes())
	seconds := int(chatDuration.Seconds()) % 60

	c.log(fmt.Sprintf("[TIMING] LLM Chat (Standard) took: %v", chatDuration))

	if err == nil && resp != "" {
		if !strings.Contains(resp, i18n.T("analysis.timing_check")) {
			timingInfo := fmt.Sprintf(i18n.T("analysis.timing"), minutes, seconds)
			resp = resp + timingInfo
		}
		c.log(fmt.Sprintf("[TIMING] Chat completed in: %dm%ds (%v)", minutes, seconds, chatDuration))
	}

	if threadID != "" && cfg.DetailedLog {
		if err != nil {
			c.LogChatToFile(threadID, "SYSTEM ERROR", fmt.Sprintf("Error: %v", err))
		} else {
			c.LogChatToFile(threadID, "LLM RESPONSE", resp)
		}
	}

	// Increment analysis count on successful completion (standard LLM path)
	if err == nil && c.licenseClient != nil && c.licenseClient.IsActivated() {
		c.licenseClient.IncrementAnalysis()
		c.log("[LICENSE] Analysis count incremented after successful completion (standard LLM)")
	}

	c.log(fmt.Sprintf("[TIMING] Total SendMessage (Standard) took: %v", time.Since(startTotal)))
	return resp, err
}

// runEinoAnalysis 执行 Eino 分析引擎的分析流�
func (c *ChatFacadeService) runEinoAnalysis(threadID, message, userMessageID, requestID, dataSourceID string, cfg config.Config) (resp string, retErr error) {
	// Recover from panics to ensure errors are always saved to the chat thread
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("analysis panic: %v", r)
			c.log(fmt.Sprintf("[PANIC] runEinoAnalysis panicked: %v", r))
			c.handleEinoError(threadID, userMessageID, requestID, panicErr, cfg, 0)
			resp = ""
			retErr = panicErr
		}
	}()
	// Load history
	startHist := time.Now()
	var history []*schema.Message
	threads, _ := c.chatService.LoadThreads()
	for _, t := range threads {
		if t.ID == threadID {
			for _, m := range t.Messages {
				role := schema.User
				if m.Role == "assistant" {
					role = schema.Assistant
				}
				history = append(history, &schema.Message{
					Role:    role,
					Content: m.Content,
				})
			}
			break
		}
	}
	c.log(fmt.Sprintf("[TIMING] Loading history took: %v", time.Since(startHist)))

	history = append(history, &schema.Message{Role: schema.User, Content: message})

	// Create progress callback
	progressCallback := func(update agent.ProgressUpdate) {
		progressWithThread := map[string]interface{}{
			"threadId":    threadID,
			"stage":       update.Stage,
			"progress":    update.Progress,
			"message":     update.Message,
			"step":        update.Step,
			"total":       update.Total,
			"tool_name":   update.ToolName,
			"tool_output": update.ToolOutput,
		}
		runtime.EventsEmit(c.ctx, "analysis-progress", progressWithThread)
	}

	sessionDir := c.chatService.GetSessionDirectory(threadID)

	// Capture existing session files before analysis
	existingFiles := make(map[string]bool)
	if preAnalysisFiles, err := c.chatService.GetSessionFiles(threadID); err == nil {
		for _, file := range preAnalysisFiles {
			existingFiles[file.Name] = true
		}
		c.log(fmt.Sprintf("[CHART] Pre-analysis: %d existing files in session", len(existingFiles)))
	}

	// Create file saved callback
	fileSavedCallback := func(fileName, fileType string, fileSize int64) {
		file := SessionFile{
			Name:      fileName,
			Path:      fmt.Sprintf("files/%s", fileName),
			Type:      fileType,
			Size:      fileSize,
			CreatedAt: time.Now().Unix(),
		}
		if err := c.chatService.AddSessionFile(threadID, file); err != nil {
			c.log(fmt.Sprintf("[ERROR] Failed to register session file: %v", err))
		} else {
			c.log(fmt.Sprintf("[SESSION] Registered file: %s (%s, %d bytes)", fileName, fileType, fileSize))
		}
	}

	if c.einoService == nil {
		c.log("[WARNING] EinoService became nil during request processing")
		return "", WrapError("chat", "runEinoAnalysis", fmt.Errorf("EinoService is not available"))
	}

	c.log(fmt.Sprintf("[EINO-CHECK] EinoService is available, proceeding with analysis for thread: %s, dataSource: %s", threadID, dataSourceID))

	analysisStartTime := time.Now()
	respMsg, err := c.einoService.RunAnalysisWithProgress(c.ctx, history, dataSourceID, threadID, sessionDir, userMessageID, progressCallback, fileSavedCallback, c.IsCancelRequested)
	analysisDuration := time.Since(analysisStartTime)
	minutes := int(analysisDuration.Minutes())
	seconds := int(analysisDuration.Seconds()) % 60

	if err != nil {
		c.log(fmt.Sprintf("[DEBUG-CANCEL] Analysis failed after %v, error: %s, cancelFlag: %v", analysisDuration, err.Error(), c.IsCancelRequested()))
		c.handleEinoError(threadID, userMessageID, requestID, err, cfg, analysisDuration)
		return "", err
	}

	resp = respMsg.Content

	if !strings.Contains(resp, i18n.T("analysis.timing_check")) {
		timingInfo := fmt.Sprintf(i18n.T("analysis.timing"), minutes, seconds)
		resp = resp + timingInfo
		c.log(fmt.Sprintf("[TIMING] Analysis completed in: %dm%ds (%v)", minutes, seconds, analysisDuration))
	}

	if cfg.DetailedLog {
		c.LogChatToFile(threadID, "LLM RESPONSE", resp)
	}

	// Process response: detect images, filter false claims, extract charts
	c.detectAndEmitImages(resp, threadID, userMessageID, requestID)
	resp = c.filterFalseFileClaimsIfECharts(resp)

	chartItems := c.extractChartItems(resp, threadID, userMessageID, requestID, sessionDir, existingFiles)

	var chartData *ChartData
	if len(chartItems) > 0 {
		chartData = &ChartData{Charts: chartItems}
		c.log(fmt.Sprintf("[CHART] Final total charts: %d", len(chartItems)))
	}

	// Attach chart data to user message
	if chartData != nil && threadID != "" {
		if userMessageID != "" {
			c.attachChartToUserMessage(threadID, userMessageID, chartData)
		} else {
			c.log("[WARNING] SendMessage called without userMessageID, falling back to last user message")
			c.attachChartToUserMessage(threadID, "", chartData)
		}
	}

	// Save assistant message
	if threadID != "" {
		totalSecs := analysisDuration.Seconds()
		timingData := map[string]interface{}{
			"total_seconds":           totalSecs,
			"total_minutes":           minutes,
			"total_seconds_remainder": seconds,
			"analysis_type":           "eino_service",
			"timestamp":               analysisStartTime.Add(analysisDuration).Unix(),
			"stages": []map[string]interface{}{
				{"name": "AI 分析", "duration": totalSecs * 0.60, "percentage": 60.0, "description": "LLM 理解需求、生成代码和分析结果"},
				{"name": "SQL 查询", "duration": totalSecs * 0.20, "percentage": 20.0, "description": "数据库查询和数据提取"},
				{"name": "Python 处理", "duration": totalSecs * 0.15, "percentage": 15.0, "description": "数据处理和图表生�"},
				{"name": "其他", "duration": totalSecs * 0.05, "percentage": 5.0, "description": "初始化和后处�"},
			},
		}

		assistantMsg := ChatMessage{
			ID:         strconv.FormatInt(time.Now().UnixNano(), 10),
			Role:       "assistant",
			Content:    resp,
			Timestamp:  time.Now().Unix(),
			ChartData:  chartData,
			TimingData: timingData,
		}

		if err := c.chatService.AddMessage(threadID, assistantMsg); err != nil {
			c.log(fmt.Sprintf("[CHART] Failed to save assistant message: %v", err))
		} else {
			c.log(fmt.Sprintf("[CHART] Saved assistant message with chart_data: %v, timing_data: %v", chartData != nil, timingData != nil))

			if userMessageID != "" {
				if err := c.AssociateNewFilesWithMessage(threadID, userMessageID, existingFiles); err != nil {
					c.log(fmt.Sprintf("[SESSION] Failed to associate files with user message: %v", err))
				}
			}

			// Flush and persist analysis results
			if c.eventAggregator != nil {
				c.eventAggregator.FlushNow(threadID, true)
			}

			if c.eventAggregator != nil && userMessageID != "" {
				allItems := c.eventAggregator.GetAllFlushedItems(threadID)

				typeCount := make(map[string]int)
				for _, item := range allItems {
					typeCount[item.Type]++
				}
				c.log(fmt.Sprintf("[PERSISTENCE] GetAllFlushedItems returned %d items, types: %v", len(allItems), typeCount))

				// Safety net: fill in missing items from chartItems
				hasECharts := typeCount["echarts"] > 0
				hasTable := typeCount["table"] > 0
				if (!hasECharts || !hasTable) && len(chartItems) > 0 {
					for idx, ci := range chartItems {
						if ci.Type == "echarts" && !hasECharts {
							allItems = append(allItems, AnalysisResultItem{
								ID:   fmt.Sprintf("fallback_echarts_%s_%d", userMessageID, idx),
								Type: "echarts",
								Data: ci.Data,
								Metadata: map[string]interface{}{
									"sessionId": threadID,
									"messageId": userMessageID,
									"timestamp": time.Now().UnixMilli(),
								},
								Source: "realtime",
							})
						} else if ci.Type == "table" && !hasTable {
							var tableData []map[string]interface{}
							if json.Unmarshal([]byte(ci.Data), &tableData) == nil {
								allItems = append(allItems, AnalysisResultItem{
									ID:   fmt.Sprintf("fallback_table_%s_%d", userMessageID, idx),
									Type: "table",
									Data: map[string]interface{}{"title": "", "rows": tableData},
									Metadata: map[string]interface{}{
										"sessionId": threadID,
										"messageId": userMessageID,
										"timestamp": time.Now().UnixMilli(),
									},
									Source: "realtime",
								})
							}
						}
					}
				}

				if len(allItems) > 0 {
					if err := c.chatService.SaveAnalysisResults(threadID, userMessageID, allItems); err != nil {
						c.log(fmt.Sprintf("[PERSISTENCE] Failed to save analysis results: %v", err))
					} else {
						c.log(fmt.Sprintf("[PERSISTENCE] Saved %d analysis results to message %s", len(allItems), userMessageID))
					}
				}
				c.eventAggregator.ClearFlushedItems(threadID)
			}

			runtime.EventsEmit(c.ctx, "analysis-completed", map[string]interface{}{
				"threadId":       threadID,
				"userMessageId":  userMessageID,
				"assistantMsgId": assistantMsg.ID,
				"hasChartData":   chartData != nil,
				"requestId":      requestID,
			})

			// Increment analysis count on successful completion
			if c.licenseClient != nil && c.licenseClient.IsActivated() {
				c.licenseClient.IncrementAnalysis()
				c.log("[LICENSE] Analysis count incremented after successful completion")
			}
		}
	}

	return resp, nil
}

// handleEinoError 处理 Eino 分析错误
func (c *ChatFacadeService) handleEinoError(threadID, userMessageID, requestID string, err error, cfg config.Config, analysisDuration time.Duration) {
	errStr := err.Error()
	var errorCode string
	var userFriendlyMessage string

	if strings.Contains(errStr, "cancelled by user") || strings.Contains(errStr, "cancelled while waiting") {
		errorCode = "CANCELLED"
		userFriendlyMessage = i18n.T("error.analysis_cancelled")
		c.log(fmt.Sprintf("[CANCEL] Analysis cancelled for thread: %s", threadID))
	} else {
		switch {
		case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "Timeout"):
			errorCode = "ANALYSIS_TIMEOUT"
			minutes := int(analysisDuration.Minutes())
			seconds := int(analysisDuration.Seconds()) % 60
			userFriendlyMessage = i18n.T("analysis.timeout_detail", minutes, seconds)
		case strings.Contains(errStr, "context canceled") || strings.Contains(errStr, "context deadline exceeded"):
			errorCode = "ANALYSIS_TIMEOUT"
			userFriendlyMessage = i18n.T("analysis.timeout_request")
		case strings.Contains(errStr, "connection") || strings.Contains(errStr, "network"):
			errorCode = "NETWORK_ERROR"
			userFriendlyMessage = i18n.T("analysis.network_error_msg")
		case strings.Contains(errStr, "database") || strings.Contains(errStr, "sqlite") || strings.Contains(errStr, "SQL"):
			errorCode = "DATABASE_ERROR"
			userFriendlyMessage = i18n.T("analysis.database_error_msg")
		case strings.Contains(errStr, "Python") || strings.Contains(errStr, "python"):
			errorCode = "PYTHON_ERROR"
			userFriendlyMessage = i18n.T("analysis.python_error_msg")
		case strings.Contains(errStr, "LLM") || strings.Contains(errStr, "API") || strings.Contains(errStr, "model"):
			errorCode = "LLM_ERROR"
			userFriendlyMessage = i18n.T("analysis.llm_error_msg")
		default:
			errorCode = "ANALYSIS_ERROR"
			userFriendlyMessage = i18n.T("analysis.error_detail", errStr)
		}
		c.log(fmt.Sprintf("[ERROR] Analysis error for thread %s: code=%s, message=%s", threadID, errorCode, errStr))
	}

	// Save error message to database BEFORE emitting events
	if threadID != "" {
		var chatErrorMsg string
		if errorCode == "CANCELLED" {
			chatErrorMsg = i18n.T("analysis.cancelled_msg")
		} else {
			chatErrorMsg = fmt.Sprintf(i18n.T("analysis.error_with_detail"), errorCode, userFriendlyMessage, errStr)
		}
		errChatMsg := ChatMessage{
			ID:        fmt.Sprintf("error_%d", time.Now().UnixNano()),
			Role:      "assistant",
			Content:   chatErrorMsg,
			Timestamp: time.Now().Unix(),
		}
		if addErr := c.chatService.AddMessage(threadID, errChatMsg); addErr != nil {
			c.log(fmt.Sprintf("[ERROR] Failed to save error message to chat: %v", addErr))
		} else {
			// Notify frontend that thread data changed so it reloads from DB
			// This prevents the frontend's stale SaveChatHistory from overwriting our error message
			runtime.EventsEmit(c.ctx, "thread-updated", threadID)
		}
	}

	// Emit progress complete event
	progressMessage := "progress.analysis_cancelled"
	if errorCode != "CANCELLED" {
		progressMessage = "progress.analysis_error"
	}
	runtime.EventsEmit(c.ctx, "analysis-progress", map[string]interface{}{
		"threadId": threadID,
		"stage":    "complete",
		"progress": 100,
		"message":  progressMessage,
		"step":     0,
		"total":    0,
	})

	if errorCode == "CANCELLED" {
		if c.eventAggregator != nil {
			c.eventAggregator.EmitCancelled(threadID, requestID)
			c.eventAggregator.SetLoading(threadID, false, requestID)
		} else {
			runtime.EventsEmit(c.ctx, "analysis-cancelled", map[string]interface{}{
				"threadId":  threadID,
				"requestId": requestID,
				"message":   i18n.T("error.analysis_cancelled"),
				"timestamp": time.Now().UnixMilli(),
			})
			runtime.EventsEmit(c.ctx, "analysis-result-loading", map[string]interface{}{
				"sessionId": threadID,
				"loading":   false,
				"requestId": requestID,
			})
		}
	} else {
		if c.eventAggregator != nil {
			c.eventAggregator.EmitErrorWithCode(threadID, requestID, errorCode, userFriendlyMessage)
			c.eventAggregator.SetLoading(threadID, false, requestID)
		} else {
			runtime.EventsEmit(c.ctx, "analysis-error", map[string]interface{}{
				"threadId":  threadID,
				"sessionId": threadID,
				"requestId": requestID,
				"code":      errorCode,
				"error":     userFriendlyMessage,
				"message":   userFriendlyMessage,
				"timestamp": time.Now().UnixMilli(),
			})
			runtime.EventsEmit(c.ctx, "analysis-result-loading", map[string]interface{}{
				"sessionId": threadID,
				"loading":   false,
				"requestId": requestID,
			})
		}
	}

	runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
		"loading":  false,
		"threadId": threadID,
	})
}

// extractChartItems 从响应中提取所有图表项
func (c *ChatFacadeService) extractChartItems(resp, threadID, userMessageID, requestID, sessionDir string, existingFiles map[string]bool) []ChartItem {
	var chartItems []ChartItem

	// 1. ECharts JSON
	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	allEChartsMatches := reECharts.FindAllStringSubmatch(resp, -1)
	reEChartsNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:echarts\\s*\\n(\\{[\\s\\S]+?\\n\\})(?:\\s*\\n(?:---|###)|\\s*$)")
	allEChartsNoBTMatches := reEChartsNoBT.FindAllStringSubmatch(resp, -1)
	allEChartsMatches = append(allEChartsMatches, allEChartsNoBTMatches...)

	for matchIdx, matchECharts := range allEChartsMatches {
		if len(matchECharts) > 1 {
			jsonStr := strings.TrimSpace(matchECharts[1])
			var testJSON map[string]interface{}
			parsedJSON := jsonStr
			parseErr := json.Unmarshal([]byte(jsonStr), &testJSON)
			if parseErr != nil {
				c.log(fmt.Sprintf("[CHART] Initial JSON parse failed for echarts #%d: %v, attempting to clean", matchIdx+1, parseErr))
				cleanedJSON := cleanEChartsJSON(jsonStr)
				if cleanErr := json.Unmarshal([]byte(cleanedJSON), &testJSON); cleanErr == nil {
					parsedJSON = cleanedJSON
					parseErr = nil
				}
			}
			if parseErr == nil {
				chartDataStr := parsedJSON
				if c.saveChartDataToFileFn != nil {
					if fileRef, saveErr := c.saveChartDataToFileFn(threadID, "echarts", parsedJSON); saveErr == nil && fileRef != "" {
						chartDataStr = fileRef
					}
				}
				chartItems = append(chartItems, ChartItem{Type: "echarts", Data: chartDataStr})
				if c.eventAggregator != nil {
					c.eventAggregator.AddECharts(threadID, userMessageID, requestID, parsedJSON)
				}
			}
		}
	}

	// 2. Markdown Image (Base64)
	reImage := regexp.MustCompile(`!\[.*?\]\((data:image\/.*?;base64,.*?)\)`)
	allImageMatches := reImage.FindAllStringSubmatch(resp, -1)
	for _, matchImage := range allImageMatches {
		if len(matchImage) > 1 {
			chartItems = append(chartItems, ChartItem{Type: "image", Data: matchImage[1]})
			if c.eventAggregator != nil {
				c.eventAggregator.AddImage(threadID, userMessageID, requestID, matchImage[1], "")
			}
		}
	}

	// 3. Check for saved chart files
	if threadID != "" {
		sessionFiles, err := c.chatService.GetSessionFiles(threadID)
		if err == nil {
			for _, file := range sessionFiles {
				if existingFiles[file.Name] {
					continue
				}
				if file.Type == "image" && (file.Name == "chart.png" || strings.HasPrefix(file.Name, "chart")) {
					filePath := filepath.Join(sessionDir, "files", file.Name)
					if imageData, err := os.ReadFile(filePath); err == nil {
						base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
						chartItems = append(chartItems, ChartItem{Type: "image", Data: base64Data})
					}
				}
			}
		}
	}

	// 4. Dashboard Data Update
	reDashboard := regexp.MustCompile("(?s)```\\s*json:dashboard\\s*\\n([\\s\\S]+?)\\n\\s*```")
	matchDashboard := reDashboard.FindStringSubmatch(resp)
	if len(matchDashboard) > 1 {
		jsonStr := strings.TrimSpace(matchDashboard[1])
		var data DashboardData
		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
			if c.eventAggregator != nil {
				for _, metric := range data.Metrics {
					c.eventAggregator.AddMetric(threadID, userMessageID, requestID, metric)
				}
				for _, insight := range data.Insights {
					c.eventAggregator.AddInsight(threadID, userMessageID, requestID, insight)
				}
			}
		}
	}

	// 5. Table Data
	reTable := regexp.MustCompile("(?s)```\\s*json:table\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	allTableMatches := reTable.FindAllStringSubmatchIndex(resp, -1)
	reTableNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:table\\s*\\n((?:\\{[\\s\\S]+?\\n\\}|\\[[\\s\\S]+?\\n\\]))(?:\\s*\\n(?:---|###)|\\s*$)")
	allTableNoBTMatches := reTableNoBT.FindAllStringSubmatchIndex(resp, -1)
	allTableMatches = append(allTableMatches, allTableNoBTMatches...)

	for _, matchIndices := range allTableMatches {
		if len(matchIndices) >= 4 {
			fullMatchStart := matchIndices[0]
			jsonContent := strings.TrimSpace(resp[matchIndices[2]:matchIndices[3]])

			tableTitle := ""
			if fullMatchStart > 0 {
				textBefore := resp[:fullMatchStart]
				lastNewline := strings.LastIndex(textBefore, "\n")
				if lastNewline >= 0 {
					lineBeforeCodeBlock := strings.TrimSpace(textBefore[lastNewline+1:])
					tableTitle = strings.TrimLeft(lineBeforeCodeBlock, "#*- ")
					tableTitle = strings.TrimRight(tableTitle, ":�")
					tableTitle = strings.TrimSpace(tableTitle)
					if strings.HasPrefix(tableTitle, "{") || strings.HasPrefix(tableTitle, "[") || strings.HasPrefix(tableTitle, "```") {
						tableTitle = ""
					}
				}
			}

			var tableData []map[string]interface{}
			var parseErr error
			var columnsOrder []string
			if parseErr = json.Unmarshal([]byte(jsonContent), &tableData); parseErr != nil {
				var colDataFormat struct {
					Columns []string        `json:"columns"`
					Data    [][]interface{} `json:"data"`
				}
				if err := json.Unmarshal([]byte(jsonContent), &colDataFormat); err == nil && len(colDataFormat.Columns) > 0 && len(colDataFormat.Data) > 0 {
					columnsOrder = colDataFormat.Columns
					tableData = make([]map[string]interface{}, 0, len(colDataFormat.Data))
					for _, row := range colDataFormat.Data {
						rowMap := make(map[string]interface{})
						for i, val := range row {
							if i < len(colDataFormat.Columns) {
								rowMap[colDataFormat.Columns[i]] = val
							}
						}
						tableData = append(tableData, rowMap)
					}
					parseErr = nil
				} else {
					var arrayData [][]interface{}
					if err := json.Unmarshal([]byte(jsonContent), &arrayData); err == nil && len(arrayData) > 1 {
						headers := make([]string, len(arrayData[0]))
						for i, h := range arrayData[0] {
							headers[i] = fmt.Sprintf("%v", h)
						}
						columnsOrder = headers
						tableData = make([]map[string]interface{}, 0, len(arrayData)-1)
						for _, row := range arrayData[1:] {
							rowMap := make(map[string]interface{})
							for i, val := range row {
								if i < len(headers) {
									rowMap[headers[i]] = val
								}
							}
							tableData = append(tableData, rowMap)
						}
						parseErr = nil
					}
				}
			} else {
				columnsOrder = extractJSONObjectKeysOrdered(jsonContent)
			}

			if parseErr == nil && len(tableData) > 0 {
				tableDataWithTitle := map[string]interface{}{
					"title":   tableTitle,
					"columns": columnsOrder,
					"rows":    tableData,
				}

				tableDataJSON, _ := json.Marshal(tableData)
				tableDataStr := string(tableDataJSON)

				if c.saveChartDataToFileFn != nil {
					if fileRef, saveErr := c.saveChartDataToFileFn(threadID, "table", tableDataStr); saveErr == nil && fileRef != "" {
						tableDataStr = fileRef
					}
				}

				chartItems = append(chartItems, ChartItem{Type: "table", Data: tableDataStr})
				if c.eventAggregator != nil {
					c.eventAggregator.AddTable(threadID, userMessageID, requestID, tableDataWithTitle)
				}
			}
		}
	}

	// 6. CSV Download Link
	reCSV := regexp.MustCompile(`\[.*?\]\((data:text/csv;base64,[A-Za-z0-9+/=]+)\)`)
	allCSVMatches := reCSV.FindAllStringSubmatch(resp, -1)
	for _, matchCSV := range allCSVMatches {
		if len(matchCSV) > 1 {
			chartItems = append(chartItems, ChartItem{Type: "csv", Data: matchCSV[1]})
			if c.eventAggregator != nil {
				c.eventAggregator.AddCSV(threadID, userMessageID, requestID, matchCSV[1], "")
			}
		}
	}

	// 7. Markdown tables
	mdTables := extractMarkdownTablesFromText(resp)
	for _, mdTable := range mdTables {
		tableDataWithTitle := map[string]interface{}{
			"title":   mdTable.Title,
			"columns": mdTable.Columns,
			"rows":    mdTable.Rows,
		}

		tableDataJSON, _ := json.Marshal(mdTable.Rows)
		tableDataStr := string(tableDataJSON)

		if c.saveChartDataToFileFn != nil {
			if fileRef, saveErr := c.saveChartDataToFileFn(threadID, "table", tableDataStr); saveErr == nil && fileRef != "" {
				tableDataStr = fileRef
			}
		}

		chartItems = append(chartItems, ChartItem{Type: "table", Data: tableDataStr})
		if c.eventAggregator != nil {
			c.eventAggregator.AddTable(threadID, userMessageID, requestID, tableDataWithTitle)
		}
	}

	return chartItems
}

// attachChartToUserMessage 将图表数据附加到特定用户消息
func (c *ChatFacadeService) attachChartToUserMessage(threadID, messageID string, chartData *ChartData) {
	if c.chatService == nil {
		return
	}

	threads, err := c.chatService.LoadThreads()
	if err != nil {
		c.log(fmt.Sprintf("[CHART] Failed to load threads for chart attachment: %v", err))
		return
	}

	for _, t := range threads {
		if t.ID == threadID {
			for i := len(t.Messages) - 1; i >= 0; i-- {
				msg := &t.Messages[i]
				if msg.Role == "user" {
					if messageID == "" || msg.ID == messageID {
						msg.ChartData = chartData
						if err := c.chatService.SaveThreads([]ChatThread{t}); err != nil {
							c.log(fmt.Sprintf("[CHART] Failed to save chart data: %v", err))
						}
						return
					}
				}
			}
			break
		}
	}
}

// detectAndEmitImages 检测并发送响应中的图�
func (c *ChatFacadeService) detectAndEmitImages(response, threadID, userMessageID, requestID string) {
	if c.chatService == nil || threadID == "" {
		return
	}

	// Check for saved image files in session directory
	sessionFiles, err := c.chatService.GetSessionFiles(threadID)
	if err != nil {
		return
	}

	sessionDir := c.chatService.GetSessionDirectory(threadID)
	for _, file := range sessionFiles {
		if file.Type == "image" {
			filePath := filepath.Join(sessionDir, "files", file.Name)
			if imageData, err := os.ReadFile(filePath); err == nil {
				base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)
				if c.eventAggregator != nil {
					c.eventAggregator.AddImage(threadID, userMessageID, requestID, base64Data, file.Name)
				}
			}
		}
	}
}

// filterFalseFileClaimsIfECharts 过滤 ECharts 响应中的虚假文件声明
func (c *ChatFacadeService) filterFalseFileClaimsIfECharts(response string) string {
	// Check if response contains ECharts data
	if !strings.Contains(response, "json:echarts") {
		return response
	}

	// Filter out false file generation claims
	lines := strings.Split(response, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip lines that claim file generation but are likely false
		if strings.Contains(trimmed, "已保�") && strings.Contains(trimmed, ".png") && strings.Contains(response, "json:echarts") {
			continue
		}
		if strings.Contains(trimmed, "saved to") && strings.Contains(trimmed, ".png") && strings.Contains(response, "json:echarts") {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}

// SendFreeChatMessage 发送自由聊天消息（无数据源上下文）
func (c *ChatFacadeService) SendFreeChatMessage(threadID, message, userMessageID string) (string, error) {
	if c.chatService == nil {
		return "", WrapError("chat", "SendFreeChatMessage", fmt.Errorf("chat service not initialized"))
	}

	cfg, err := c.configProvider.GetEffectiveConfig()
	if err != nil {
		return "", err
	}

	startTotal := time.Now()

	if threadID != "" && cfg.DetailedLog {
		c.LogChatToFile(threadID, "FREE CHAT USER", message)
	}

	// Save user message to thread file BEFORE processing
	if threadID != "" && userMessageID != "" {
		threads, err := c.chatService.LoadThreads()
		if err == nil {
			messageExists := false
			for _, t := range threads {
				if t.ID == threadID {
					for _, m := range t.Messages {
						if m.ID == userMessageID {
							messageExists = true
							break
						}
					}
					break
				}
			}

			if !messageExists {
				userMsg := ChatMessage{
					ID:        userMessageID,
					Role:      "user",
					Content:   message,
					Timestamp: time.Now().Unix(),
				}
				if err := c.chatService.AddMessage(threadID, userMsg); err != nil {
					c.log(fmt.Sprintf("[ERROR] Failed to save free chat user message: %v", err))
				}
			}
		}
	}

	// Build conversation history for context
	var historyContext strings.Builder
	if threadID != "" {
		threads, _ := c.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				startIdx := 0
				if len(t.Messages) > 10 {
					startIdx = len(t.Messages) - 10
				}
				for _, m := range t.Messages[startIdx:] {
					if m.Role == "user" {
						historyContext.WriteString(fmt.Sprintf("User: %s\n", m.Content))
					} else if m.Role == "assistant" {
						content := m.Content
						if len(content) > 500 {
							content = content[:500] + "..."
						}
						historyContext.WriteString(fmt.Sprintf("Assistant: %s\n", content))
					}
				}
				break
			}
		}
	}

	// Detect if tools are needed
	needsTools := false
	if c.searchKeywordsManager != nil {
		needsSearch, _ := c.searchKeywordsManager.DetectSearchNeed(message)
		needsTools = needsSearch
	}

	// Check history for analysis context
	historyStr := historyContext.String()
	historyHasAnalysisContext := strings.Contains(historyStr, "分析") ||
		strings.Contains(historyStr, "数据�") ||
		strings.Contains(historyStr, "analyze") ||
		strings.Contains(historyStr, "data source") ||
		strings.Contains(historyStr, "start_datasource_analysis")
	needsTools = needsTools || historyHasAnalysisContext

	langPrompt := getLangPromptFromMessage(message)
	var fullMessage string
	if historyContext.Len() > 0 {
		fullMessage = fmt.Sprintf("Previous conversation:\n%s\nUser: %s\n\n(Please answer in %s)", historyContext.String(), message, langPrompt)
	} else {
		fullMessage = fmt.Sprintf("%s\n\n(Please answer in %s)", message, langPrompt)
	}

	assistantMsgID := fmt.Sprintf("assistant_%d", time.Now().UnixNano())

	if threadID != "" {
		runtime.EventsEmit(c.ctx, "free-chat-stream-start", map[string]interface{}{
			"threadId":  threadID,
			"messageId": assistantMsgID,
		})
	}

	chatStartTime := time.Now()

	onChunk := func(content string) {
		if threadID != "" {
			runtime.EventsEmit(c.ctx, "free-chat-stream-chunk", map[string]interface{}{
				"threadId":  threadID,
				"messageId": assistantMsgID,
				"content":   content,
			})
		}
	}

	var resp string

	if needsTools && c.einoService != nil {
		c.log("[FREE-CHAT] Tool router detected tool need, using agent with tools")

		if threadID != "" {
			runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
				"loading":  true,
				"threadId": threadID,
			})
			runtime.EventsEmit(c.ctx, "free-chat-search-status", map[string]interface{}{
				"threadId":  threadID,
				"messageId": assistantMsgID,
				"searching": true,
			})
		}

		resp, err = c.runFreeChatWithTools(c.ctx, message, historyContext.String(), langPrompt, onChunk, cfg)

		if err != nil {
			c.log(fmt.Sprintf("[FREE-CHAT] Tool-based chat failed: %v, falling back to streaming chat", err))
			llm := agent.NewLLMService(cfg, c.log)
			resp, err = llm.ChatStream(c.ctx, fullMessage, onChunk)
		}

		if threadID != "" {
			runtime.EventsEmit(c.ctx, "free-chat-search-status", map[string]interface{}{
				"threadId":  threadID,
				"messageId": assistantMsgID,
				"searching": false,
			})
			runtime.EventsEmit(c.ctx, "chat-loading", map[string]interface{}{
				"loading":  false,
				"threadId": threadID,
			})
		}
	} else {
		c.log("[FREE-CHAT] No search keyword detected, using streaming LLM chat")
		llm := agent.NewLLMService(cfg, c.log)
		resp, err = llm.ChatStream(c.ctx, fullMessage, onChunk)
	}

	chatDuration := time.Since(chatStartTime)

	if threadID != "" {
		runtime.EventsEmit(c.ctx, "free-chat-stream-end", map[string]interface{}{
			"threadId":  threadID,
			"messageId": assistantMsgID,
		})
	}

	if err != nil {
		if threadID != "" && cfg.DetailedLog {
			c.LogChatToFile(threadID, "FREE CHAT ERROR", fmt.Sprintf("Error: %v", err))
		}
		return "", err
	}

	if threadID != "" && resp != "" {
		assistantMsg := ChatMessage{
			ID:        assistantMsgID,
			Role:      "assistant",
			Content:   resp,
			Timestamp: time.Now().Unix(),
		}
		if err := c.chatService.AddMessage(threadID, assistantMsg); err != nil {
			c.log(fmt.Sprintf("[ERROR] Failed to save free chat assistant message: %v", err))
		}
		runtime.EventsEmit(c.ctx, "thread-updated", threadID)
	}

	if threadID != "" && cfg.DetailedLog {
		c.LogChatToFile(threadID, "FREE CHAT RESPONSE", resp)
	}

	c.log(fmt.Sprintf("[FREE-CHAT] Completed in %v", chatDuration))
	c.log(fmt.Sprintf("[TIMING] Total SendFreeChatMessage took: %v", time.Since(startTotal)))

	return resp, nil
}

// runFreeChatWithTools 使用工具运行自由聊天（搜索、获取等�
// 注意：此方法�App.runFreeChatWithTools 的简化版本，
// 完整的工具集成将�App 门面委托时通过 App 上下文提�
func (c *ChatFacadeService) runFreeChatWithTools(ctx context.Context, userMessage, historyContext, langPrompt string, onChunk func(string), cfg config.Config) (string, error) {
	if c.einoService == nil {
		return "", fmt.Errorf("eino service not available for tool-based chat")
	}

	// Build messages
	var fullMessage string
	if historyContext != "" {
		fullMessage = fmt.Sprintf("Previous conversation:\n%s\nUser: %s\n\n(Please answer in %s)", historyContext, userMessage, langPrompt)
	} else {
		fullMessage = fmt.Sprintf("%s\n\n(Please answer in %s)", userMessage, langPrompt)
	}

	// Fallback to simple streaming chat
	// The full tool integration (web search, fetch, time, location, start_datasource_analysis)
	// requires access to dataSourceService and other App-level dependencies.
	// When App delegates to ChatFacadeService, it can inject these via the runFreeChatWithToolsFn.
	llm := agent.NewLLMService(cfg, c.log)
	return llm.ChatStream(ctx, fullMessage, onChunk)
}

// --- Internal Helpers ---

// log 内部日志辅助方法
func (c *ChatFacadeService) log(msg string) {
	if c.logger != nil {
		c.logger(msg)
	}
}

// getLangPromptFromMessage 从消息内容检测语言并返回语言提示
// 这是一个包级别的辅助函数，�ChatFacadeService 使用
func getLangPromptFromMessage(message string) string {
	return "the same language as the user's message"
}
