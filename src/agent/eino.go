package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"vantagics/agent/templates"
	"vantagics/config"
	"vantagics/i18n"
)

// getProviderMaxTokens returns the maximum OUTPUT tokens for different providers
// This controls how long the LLM's response can be, NOT the total context window
func getProviderMaxTokens(modelName string, configuredMax int) int {
	// Provider-specific OUTPUT limits based on model names
	// These are conservative limits to ensure complete responses
	providerLimits := map[string]int{
		// OpenAI models - output limits
		"gpt-4":           8192,
		"gpt-4-turbo":     16384,  // Increased for longer outputs
		"gpt-4o":          16384,  // Increased for longer outputs
		"gpt-3.5-turbo":   4096,
		
		// Anthropic models - output limits
		"claude-3":        8192,
		"claude-3-sonnet": 8192,
		"claude-3-opus":   8192,
		"claude-3-haiku":  8192,
		
		// Google Gemini models - output limits
		"gemini-3-pro":         16384,
		"gemini-3-flash":       16384,
		"gemini-2.5-pro":       16384,
		"gemini-2.5-flash":     16384,
		
		// Default fallback
		"default":         8192,
	}
	
	// Find the limit for this model
	limit := providerLimits["default"]
	for model, maxTokens := range providerLimits {
		if strings.Contains(strings.ToLower(modelName), strings.ToLower(model)) {
			limit = maxTokens
			break
		}
	}
	
	// If configured max is set and reasonable, use it
	if configuredMax > 0 && configuredMax <= limit {
		return configuredMax
	}
	
	// Otherwise use the provider's limit
	return limit
}

// normalizeOpenAIBaseURL normalizes the base URL for OpenAI-compatible APIs
// The OpenAI SDK automatically appends /chat/completions, so we need to strip it if present
// This allows users to enter either:
//   - https://api.example.com/v1 (correct)
//   - https://api.example.com/v1/chat/completions (also works after normalization)
func normalizeOpenAIBaseURL(baseURL string) string {
	if baseURL == "" {
		return baseURL
	}
	
	// Remove trailing slash first
	baseURL = strings.TrimSuffix(baseURL, "/")
	
	// Remove /chat/completions suffix if present (SDK will add it back)
	if strings.HasSuffix(baseURL, "/chat/completions") {
		baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
	}
	
	// Also handle case where user might have added just /completions
	if strings.HasSuffix(baseURL, "/completions") {
		baseURL = strings.TrimSuffix(baseURL, "/completions")
	}
	
	return baseURL
}

// EinoService manages Eino-based agents
type EinoService struct {
	ChatModel                  model.ChatModel
	dsService                  *DataSourceService
	cfg                        config.Config
	Logger                     func(string)
	memoryManager              *MemoryManager
	workingContextManager      *WorkingContextManager
	conversationContextManager *ConversationContextManager // For tracking conversation context
	pythonPool                 *PythonPool
	errorKnowledge             *ErrorKnowledge
	skillManager               *templates.SkillManager
	memoryService              *MemoryService // For persistent memory storage
	executionValidator         *ExecutionValidator // For execution plan validation
	combinedPlanner            *CombinedClassifierPlanner // Shared combined classifier+planner (avoids 2 LLM calls)
	sharedSchemaBuilder        *SchemaContextBuilder       // Shared schema builder with cache across requests
}

// TrajectoryStep represents a single step in agent execution
type TrajectoryStep struct {
	StepNumber  int                      `json:"step_number"`
	Timestamp   int64                    `json:"timestamp"`
	Type        string                   `json:"type"` // "model_call" | "tool_call"
	ModelInput  []map[string]interface{} `json:"model_input,omitempty"`
	ModelOutput map[string]interface{}   `json:"model_output,omitempty"`
	ToolName    string                   `json:"tool_name,omitempty"`
	ToolInput   string                   `json:"tool_input,omitempty"`
	ToolOutput  string                   `json:"tool_output,omitempty"`
	ToolCallID  string                   `json:"tool_call_id,omitempty"`
	Error       string                   `json:"error,omitempty"`
}

// AgentTrajectory represents complete execution path for training
type AgentTrajectory struct {
	ThreadID       string           `json:"thread_id"`
	UserRequest    string           `json:"user_request"`
	DataSourceID   string           `json:"data_source_id,omitempty"`
	StartTime      int64            `json:"start_time"`
	EndTime        int64            `json:"end_time"`
	TotalDuration  int64            `json:"total_duration_ms"`
	Steps          []TrajectoryStep `json:"steps"`
	FinalResponse  string           `json:"final_response"`
	Success        bool             `json:"success"`
	ErrorMessage   string           `json:"error_message,omitempty"`
	IterationCount int              `json:"iteration_count"`
	ToolCallCount  int              `json:"tool_call_count"`
}

// NewEinoService creates a new EinoService
func NewEinoService(cfg config.Config, dsService *DataSourceService, memoryService *MemoryService, workingContextManager *WorkingContextManager, logger func(string)) (*EinoService, error) {
	// Validate required configuration
	if cfg.ModelName == "" {
		return nil, fmt.Errorf("model name is required but not configured")
	}
	
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required but not configured for provider %s", cfg.LLMProvider)
	}
	
	if dsService == nil {
		return nil, fmt.Errorf("DataSourceService is required but nil")
	}
	
	if logger != nil {
		logger(fmt.Sprintf("[EINO-INIT] Creating EinoService with provider: %s, model: %s", cfg.LLMProvider, cfg.ModelName))
		if cfg.ProxyConfig != nil {
			logger(fmt.Sprintf("[EINO-INIT] Proxy config: enabled=%v, tested=%v, host=%s, port=%d",
				cfg.ProxyConfig.Enabled, cfg.ProxyConfig.Tested, cfg.ProxyConfig.Host, cfg.ProxyConfig.Port))
		} else {
			logger("[EINO-INIT] Proxy config: nil")
		}
	}
	
	var chatModel model.ChatModel
	var err error

	switch cfg.LLMProvider {
	case "Anthropic":
		if logger != nil {
			logger(fmt.Sprintf("[EINO-INIT] Initializing Anthropic model: %s", cfg.ModelName))
		}
		chatModel, err = NewAnthropicChatModel(context.Background(), &AnthropicConfig{
			APIKey:      cfg.APIKey,
			BaseURL:     cfg.BaseURL,
			Model:       cfg.ModelName,
			MaxTokens:   cfg.MaxTokens,
			ProxyConfig: cfg.ProxyConfig,
		})
	case "Gemini":
		if logger != nil {
			logger(fmt.Sprintf("[EINO-INIT] Initializing Gemini model: %s", cfg.ModelName))
		}
		chatModel, err = NewGeminiChatModel(context.Background(), &GeminiConfig{
			APIKey:      cfg.APIKey,
			BaseURL:     cfg.BaseURL,
			Model:       cfg.ModelName,
			MaxTokens:   cfg.MaxTokens,
			ProxyConfig: cfg.ProxyConfig,
		})
	default:
		// Default to OpenAI (includes "OpenAI", "OpenAI-Compatible", "Claude-Compatible" if using OAI format)
		// Note: "Claude-Compatible" in this project usually means "Use OpenAI client but point to Claude proxy"
		// or "Use Anthropic client". 
		// If LLMService treats Claude-Compatible as Anthropic-format, we should use AnthropicChatModel.
		// Checking llm_service.go: Claude-Compatible uses /v1/messages. So it is Anthropic format.
		if cfg.LLMProvider == "Claude-Compatible" {
			if logger != nil {
				logger(fmt.Sprintf("[EINO-INIT] Initializing Claude-Compatible model: %s", cfg.ModelName))
			}
			chatModel, err = NewAnthropicChatModel(context.Background(), &AnthropicConfig{
				APIKey:      cfg.APIKey,
				BaseURL:     cfg.BaseURL,
				Model:       cfg.ModelName,
				MaxTokens:   cfg.MaxTokens,
				ProxyConfig: cfg.ProxyConfig,
			})
		} else {
			if logger != nil {
				logger(fmt.Sprintf("[EINO-INIT] Initializing OpenAI-Compatible model: %s", cfg.ModelName))
			}
			
			// Validate OpenAI configuration
			if cfg.APIKey == "" {
				return nil, fmt.Errorf("OpenAI API key is empty - please configure your API key")
			}
			
			// Set max tokens for OpenAI with intelligent provider limits
			maxTokens := getProviderMaxTokens(cfg.ModelName, cfg.MaxTokens)
			
			// Normalize BaseURL - OpenAI SDK automatically appends /chat/completions
			// so we need to strip it if user included it in the URL
			normalizedBaseURL := normalizeOpenAIBaseURL(cfg.BaseURL)
			if logger != nil && normalizedBaseURL != cfg.BaseURL {
				logger(fmt.Sprintf("[EINO-INIT] Normalized BaseURL: %s -> %s", cfg.BaseURL, normalizedBaseURL))
			}
			
			innerModel, innerErr := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
				APIKey:     cfg.APIKey,
				BaseURL:    normalizedBaseURL,
				Model:      cfg.ModelName,
				MaxTokens:  &maxTokens, // Use pointer to int
				HTTPClient: NewProxyHTTPClient(300*time.Second, cfg.ProxyConfig),
			})
			if innerErr != nil {
				err = innerErr
			} else {
				// Wrap with error handler for better Gemini compatibility
				chatModel = NewOpenAICompatibleWrapper(innerModel, normalizedBaseURL, logger)
				if logger != nil && strings.Contains(normalizedBaseURL, "generativelanguage.googleapis.com") {
					logger("[EINO-INIT] Detected Gemini OpenAI-compatible endpoint, error handling wrapper enabled")
				}
			}
		}
	}

	if err != nil {
		if logger != nil {
			logger(fmt.Sprintf("[EINO-INIT] Failed to create chat model: %v", err))
		}
		return nil, fmt.Errorf("failed to create eino chat model: %v", err)
	}

	if logger != nil {
		logger(fmt.Sprintf("[EINO-INIT] Chat model created successfully"))
	}

	// Initialize memory manager with config's MaxTokens
	memManager := NewMemoryManager(cfg.MaxTokens, chatModel)

	// Initialize Python pool if Python path is configured
	var pyPool *PythonPool
	if cfg.PythonPath != "" {
		pool, err := NewPythonPool(cfg.PythonPath, 2)
		if err != nil {
			if logger != nil {
				logger(fmt.Sprintf("[WARNING] Failed to create Python pool: %v. Will use fallback execution.", err))
			}
		} else {
			pyPool = pool
			if logger != nil {
				logger("[INFO] Python process pool initialized")
			}
		}
	}

	// Initialize error knowledge system
	errorKnowledge := NewErrorKnowledge(dsService.dataCacheDir, logger)
	if logger != nil {
		logger("[INFO] Error knowledge system initialized")
	}

	// Initialize Skills Manager
	skillsDir := filepath.Join(dsService.dataCacheDir, "..", "skills") // Skills in Vantagics/skills
	skillManager := templates.NewSkillManager(skillsDir, logger)
	if err := skillManager.LoadSkills(); err != nil {
		if logger != nil {
			logger(fmt.Sprintf("[WARNING] Failed to load skills: %v", err))
		}
	}

	// Initialize Execution Validator
	executionValidator := NewExecutionValidator(logger)
	if logger != nil {
		logger("[INFO] Execution Validator initialized")
	}

	// Initialize Conversation Context Manager
	conversationContextManager := NewConversationContextManager()
	if logger != nil {
		logger("[INFO] Conversation Context Manager initialized")
	}

	return &EinoService{
		ChatModel:                  chatModel,
		dsService:                  dsService,
		cfg:                        cfg,
		Logger:                     logger,
		memoryManager:              memManager,
		workingContextManager:      workingContextManager,
		conversationContextManager: conversationContextManager,
		pythonPool:                 pyPool,
		errorKnowledge:             errorKnowledge,
		skillManager:               skillManager,
		memoryService:              memoryService,
		executionValidator:         executionValidator,
		combinedPlanner:            NewCombinedClassifierPlanner(chatModel, logger),
		sharedSchemaBuilder:        NewSchemaContextBuilder(dsService, 10*time.Minute, logger),
	}, nil
}

// RunAgent is a placeholder for running an Eino graph/chain
func (s *EinoService) RunAgent(ctx context.Context, input string) (string, error) {
	// Example: Simple chain
	// In a real scenario, we would build a graph with tools, memory, etc.
	
	chain := compose.NewChain[*schema.Message, *schema.Message]()
	chain.AppendChatModel(s.ChatModel)
	
	runnable, err := chain.Compile(ctx)
	if err != nil {
		return "", err
	}

	msg := &schema.Message{
		Role:    schema.User,
		Content: input,
	}

	resp, err := runnable.Invoke(ctx, msg)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// Close cleans up resources (Python pool, etc.)
func (s *EinoService) Close() {
	if s.pythonPool != nil {
		s.pythonPool.Close()
		s.pythonPool = nil
		if s.Logger != nil {
			s.Logger("[INFO] Python pool closed")
		}
	}
}

// GetErrorKnowledge returns the error knowledge instance
func (s *EinoService) GetErrorKnowledge() *ErrorKnowledge {
	return s.errorKnowledge
}

// GetSkillManager returns the skill manager instance
func (s *EinoService) GetSkillManager() *templates.SkillManager {
	return s.skillManager
}

// GetConfig returns the configuration
func (s *EinoService) GetConfig() config.Config {
	return s.cfg
}

// ExecutePython executes Python code using the pool if available, falling back to PythonService.
// Returns the output string and any error.
func (s *EinoService) ExecutePython(code string, workDir string) (string, error) {
	if s.pythonPool != nil {
		return s.pythonPool.Execute(code, workDir)
	}
	// Fallback to direct execution
	if s.cfg.PythonPath != "" {
		ps := &PythonService{}
		return ps.ExecuteScript(s.cfg.PythonPath, code)
	}
	return "", fmt.Errorf("Python environment not available")
}

// HasPython returns true if the EinoService has a working Python environment (pool or config path).
func (s *EinoService) HasPython() bool {
	return s.pythonPool != nil || s.cfg.PythonPath != ""
}

// routeFromCombinedResult determines execution path from combined classification result
func (s *EinoService) routeFromCombinedResult(result *CombinedResult, dataSourceID string) ExecutionPath {
	switch result.RequestType {
	case "consultation":
		return PathConsultation
	case "calculation":
		return PathQuick
	case "web_search":
		return PathMultiStep
	case "data_analysis", "visualization", "data_export":
		if dataSourceID != "" {
			return PathUnified
		}
		return PathMultiStep
	default:
		if dataSourceID != "" {
			return PathUnified
		}
		return PathMultiStep
	}
}

// GetExecutionValidator returns the execution validator instance
func (s *EinoService) GetExecutionValidator() *ExecutionValidator {
	return s.executionValidator
}

// RunAnalysis executes the agent with full history and tool support
func (s *EinoService) RunAnalysis(ctx context.Context, history []*schema.Message, dataSourceID, threadID string) (*schema.Message, error) {
	return s.RunAnalysisWithProgress(ctx, history, dataSourceID, threadID, "", "", nil, nil, nil)
}

// RunAnalysisWithProgress executes the agent with progress callbacks
func (s *EinoService) RunAnalysisWithProgress(ctx context.Context, history []*schema.Message, dataSourceID, threadID, sessionDir, userMessageID string, onProgress ProgressCallback, onFileSaved func(fileName, fileType string, fileSize int64), cancelCheck func() bool) (*schema.Message, error) {
	startTotal := time.Now()
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Start RunAnalysis for thread: %s", threadID))
	}

	// Configure memory manager with memory service for this thread (only if memory is enabled)
	if s.cfg.EnableMemory && s.memoryManager != nil && s.memoryService != nil && threadID != "" {
		s.memoryManager.SetMemoryService(s.memoryService, threadID)
		if s.Logger != nil {
			s.Logger("[MEMORY] Memory service configured for thread")
		}
	} else if s.Logger != nil && !s.cfg.EnableMemory {
		s.Logger("[MEMORY] Memory feature disabled in config")
	}

	// Initialize trajectory tracking for training
	trajectory := &AgentTrajectory{
		ThreadID:     threadID,
		DataSourceID: dataSourceID,
		StartTime:    time.Now().UnixMilli(),
		Steps:        []TrajectoryStep{},
		Success:      false,
	}

	// Extract user request from last message with escaping for training visibility
	var lastUserMessage string
	if len(history) > 0 {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Role == schema.User {
				trajectory.UserRequest = escapeForTraining(history[i].Content)
				lastUserMessage = history[i].Content
				break
			}
		}
	}

	// Update conversation context with user message
	if s.conversationContextManager != nil && threadID != "" && lastUserMessage != "" {
		s.conversationContextManager.UpdateFromUserMessage(threadID, lastUserMessage)
		
		// Resolve references in user message (e.g., "å¤©æ°”" -> "åŒ—äº¬çš„å¤©æ°?)
		resolvedMessage := s.conversationContextManager.ResolveReferences(threadID, lastUserMessage)
		if resolvedMessage != lastUserMessage {
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[CONTEXT] Resolved message: %s -> %s", lastUserMessage, resolvedMessage))
			}
			// Update the last user message in history with resolved version
			for i := len(history) - 1; i >= 0; i-- {
				if history[i].Role == schema.User {
					history[i].Content = resolvedMessage
					break
				}
			}
		}
	}

	// Initialize SQL collector for this session
	var sqlCollector *SQLCollector
	if sessionDir != "" && dataSourceID != "" {
		// Get data source name
		var dataSourceName string
		if sources, err := s.dsService.LoadDataSources(); err == nil {
			for _, ds := range sources {
				if ds.ID == dataSourceID {
					dataSourceName = ds.Name
					break
				}
			}
		}
		sqlCollector = NewSQLCollector(threadID, dataSourceID, dataSourceName)
		if s.Logger != nil {
			s.Logger("[SQL-COLLECTOR] Initialized for session")
		}
	}
	
	// Initialize execution recorder for this session
	var executionRecorder *ExecutionRecorder
	if sessionDir != "" && dataSourceID != "" {
		// Get data source name
		var dataSourceName string
		if sources, err := s.dsService.LoadDataSources(); err == nil {
			for _, ds := range sources {
				if ds.ID == dataSourceID {
					dataSourceName = ds.Name
					break
				}
			}
		}
		
		// Extract user request from history
		var userRequest string
		if len(history) > 0 {
			for i := len(history) - 1; i >= 0; i-- {
				if history[i].Role == schema.User {
					userRequest = history[i].Content
					break
				}
			}
		}
		
		executionRecorder = NewExecutionRecorder(sessionDir, dataSourceID, dataSourceName, userRequest, userMessageID, s.Logger)
		if s.Logger != nil {
			s.Logger("[EXECUTION-RECORDER] Initialized for session")
		}
	}

	// Save trajectory and SQL collection data on completion (success or error)
	defer func() {
		// Recover from any panic and record it
		if r := recover(); r != nil {
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[PANIC] Recovered from panic in RunAnalysisWithProgress: %v", r))
			}
			trajectory.Success = false
			trajectory.ErrorMessage = fmt.Sprintf("panic: %v", r)
		}
		
		// Record end time and duration
		trajectory.EndTime = time.Now().UnixMilli()
		trajectory.TotalDuration = trajectory.EndTime - trajectory.StartTime
		// Note: iterationCount is updated in trajectory.IterationCount during execution
		
		if sessionDir != "" {
			s.saveTrajectory(sessionDir, trajectory)
			
			// Save SQL collection data
			if sqlCollector != nil {
				if err := sqlCollector.SaveToFile(sessionDir); err != nil {
					if s.Logger != nil {
						s.Logger(fmt.Sprintf("[SQL-COLLECTOR] Failed to save: %v", err))
					}
				} else if sqlCollector.GetPairCount() > 0 && s.Logger != nil {
					s.Logger(fmt.Sprintf("[SQL-COLLECTOR] Saved %d SQL pairs to file", sqlCollector.GetPairCount()))
				}
			}
			
			// Save execution recorder data
			if executionRecorder != nil {
				if err := executionRecorder.SaveToFile(); err != nil {
					if s.Logger != nil {
						s.Logger(fmt.Sprintf("[EXECUTION-RECORDER] Failed to save: %v", err))
					}
				} else if executionRecorder.GetRecordCount() > 0 && s.Logger != nil {
					s.Logger(fmt.Sprintf("[EXECUTION-RECORDER] Saved %d execution records to file", executionRecorder.GetRecordCount()))
				}
			}
		}
	}()

	// Helper to emit progress
	emitProgress := func(stage string, progress int, message string, step, total int) {
		if onProgress != nil {
			onProgress(NewProgressUpdate(stage, progress, message, step, total))
		}
	}

	// Heartbeat mechanism to keep frontend progress bar alive during long operations
	// This prevents the frontend from timing out and clearing the progress bar
	// while the backend is still processing (e.g., during Python execution or LLM calls)
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	var lastStage string = StageInitializing
	var lastProgress int = 5
	var lastMessage string = "progress.initializing_tools"
	var heartbeatMu sync.Mutex

	// Helper to update heartbeat state
	updateHeartbeatState := func(stage string, progress int, message string) {
		heartbeatMu.Lock()
		lastStage = stage
		lastProgress = progress
		lastMessage = message
		heartbeatMu.Unlock()
	}

	// Start heartbeat goroutine - sends progress updates every 30 seconds
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				heartbeatMu.Lock()
				stage := lastStage
				progress := lastProgress
				message := lastMessage
				heartbeatMu.Unlock()
				// Send heartbeat progress update to keep frontend alive
				if onProgress != nil && stage != StageComplete {
					onProgress(NewProgressUpdate(stage, progress, message, 0, 0))
					if s.Logger != nil {
						s.Logger(fmt.Sprintf("[HEARTBEAT] Sent progress heartbeat: stage=%s, progress=%d", stage, progress))
					}
				}
			}
		}
	}()

	// Ensure heartbeat is stopped when function returns
	defer func() {
		cancelHeartbeat()
		<-heartbeatDone
	}()

	// Wrapper for emitProgress that also updates heartbeat state
	emitProgressWithHeartbeat := func(stage string, progress int, message string, step, total int) {
		updateHeartbeatState(stage, progress, message)
		emitProgress(stage, progress, message, step, total)
	}

	emitProgressWithHeartbeat(StageInitializing, 5, "progress.initializing_tools", 0, 0)

	// Check for template match first (faster path)
	if len(history) > 0 {
		lastMsg := history[len(history)-1]
		if lastMsg.Role == schema.User {
			if template := templates.DetectTemplate(lastMsg.Content); template != nil {
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[TEMPLATE] Detected template: %s", template.Name()))
				}

				// Create executor for template
				executor := &templates.ServiceExecutor{
					SQLExecutor: func(ctx context.Context, dsID, query string) ([]map[string]interface{}, error) {
						return s.dsService.ExecuteSQL(dsID, query)
					},
					PythonExecutor: func(code, workDir string) (string, error) {
						if s.pythonPool != nil {
							return s.pythonPool.Execute(code, workDir)
						}
						// Fallback to service
						ps := &PythonService{}
						return ps.ExecuteScript(s.cfg.PythonPath, code)
					},
					SchemaGetter: func(dsID string) ([]templates.TableInfo, error) {
						tablesWithCols, err := s.dsService.GetTablesWithColumns(dsID)
						if err != nil {
							return nil, err
						}
						var result []templates.TableInfo
						for tableName, cols := range tablesWithCols {
							result = append(result, templates.TableInfo{
								Name:    tableName,
								Columns: cols,
							})
						}
						return result, nil
					},
				}

				// Template progress callback
				templateProgress := func(stage string, progress int, message string, step, total int) {
					emitProgressWithHeartbeat(stage, progress, message, step, total)
				}

				result, err := template.Execute(ctx, executor, dataSourceID, templateProgress)
				if err == nil && result.Success {
					emitProgressWithHeartbeat(StageComplete, 100, "progress.analysis_complete", 0, 0)
					if s.Logger != nil {
						s.Logger(fmt.Sprintf("[TIMING] Template execution took: %v", time.Since(startTotal)))
					}
					return &schema.Message{
						Role:    schema.Assistant,
						Content: result.Output,
					}, nil
				}
				// If template failed, fall through to normal LLM flow
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[TEMPLATE] Template failed, falling back to LLM: %v", err))
				}
			}
		}
	}

	// ðŸš€ Combined Classification + Planning: Single LLM call replaces two separate calls
	// Previously: RequestTypeClassifier (LLM call #1) + AnalysisPlanner (LLM call #2)
	// Now: CombinedClassifierPlanner (single LLM call)
	var combinedResult *CombinedResult
	var classificationResult *ClassificationResult
	var planPrompt string

	if len(history) > 0 {
		lastMsg := history[len(history)-1]
		if lastMsg.Role == schema.User {
			userQuery := lastMsg.Content

			// Get data source info (reused for both classification and planning)
			dataSourceInfo := "No data source"
			var dbPath string
			if dataSourceID != "" {
				if sources, err := s.dsService.LoadDataSources(); err == nil {
					for _, ds := range sources {
						if ds.ID == dataSourceID {
							dbPath = ds.Config.DBPath
							tables, _ := s.dsService.GetDataSourceTables(dataSourceID)
							dataSourceInfo = fmt.Sprintf("Data source: %s, Tables: %s", ds.Name, strings.Join(tables, ", "))
							break
						}
					}
				}
			}

			// Single combined LLM call for classification + planning
			startClassify := time.Now()
			var err error
			combinedResult, err = s.combinedPlanner.ClassifyAndPlan(ctx, userQuery, dataSourceInfo)
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[TIMING] Combined classify+plan took: %v", time.Since(startClassify)))
			}

			if err == nil && combinedResult != nil {
				classificationResult = combinedResult.ToClassificationResult()

				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[COMBINED] type=%s, viz=%v, export=%v, complexity=%s, confidence=%.2f",
						combinedResult.RequestType,
						combinedResult.NeedsVisualization,
						combinedResult.NeedsDataExport,
						combinedResult.Complexity,
						combinedResult.Confidence))
				}

				// Quick path: execute directly without LLM
				if combinedResult.IsQuickPath && combinedResult.QuickPathCode != "" {
					if s.Logger != nil {
						s.Logger("[COMBINED] Executing quick path directly")
					}
					var result string
					var execErr error
					if s.pythonPool != nil {
						result, execErr = s.pythonPool.Execute(combinedResult.QuickPathCode, sessionDir)
					} else {
						ps := &PythonService{}
						result, execErr = ps.ExecuteScript(s.cfg.PythonPath, combinedResult.QuickPathCode)
					}
					if execErr == nil {
						emitProgressWithHeartbeat(StageComplete, 100, "progress.analysis_complete", 0, 0)
						trajectory.Success = true
						trajectory.FinalResponse = result
						trajectory.IterationCount = 1
						trajectory.ToolCallCount = 1
						if s.Logger != nil {
							s.Logger(fmt.Sprintf("[TIMING] Quick path took: %v", time.Since(startTotal)))
						}
						return &schema.Message{Role: schema.Assistant, Content: result}, nil
					}
					if s.Logger != nil {
						s.Logger(fmt.Sprintf("[COMBINED] Quick path failed: %v, continuing", execErr))
					}
				}

				// Unified Python path for data analysis
				path := s.routeFromCombinedResult(combinedResult, dataSourceID)
				if path == PathUnified && dbPath != "" && sessionDir != "" {
					if s.Logger != nil {
						s.Logger("[UNIFIED] Attempting unified Python analysis path")
					}

					metrics := NewAnalysisMetrics(s.Logger)
					generator := NewUnifiedPythonGeneratorWithCache(s.ChatModel, s.dsService, s.sharedSchemaBuilder, s.Logger)
					generator.SetMetrics(metrics)
					if classificationResult != nil {
						generator.SetClassificationHints(classificationResult)
					}

					emitProgressWithHeartbeat(StageAnalysis, 30, "progress.generating_code", 0, 0)
					generatedCode, err := generator.GenerateAnalysisCode(ctx, userQuery, dataSourceID, dbPath, sessionDir)

					if err == nil && generatedCode != nil && generatedCode.Code != "" {
						if s.Logger != nil {
							s.Logger(fmt.Sprintf("[UNIFIED] Code generated, %d SQL queries", len(generatedCode.SQLQueries)))
						}

						safety := NewExecutionSafety(s.Logger)
						safety.SetTimeout(120 * time.Second)
						safetyReport := safety.GenerateSafetyReport(generatedCode.Code)

						if safetyReport.IsSafe {
							for _, warning := range safetyReport.Warnings {
								if s.Logger != nil {
									s.Logger(fmt.Sprintf("[UNIFIED] Safety warning: %s", warning))
								}
							}

							emitProgressWithHeartbeat(StageAnalysis, 60, "progress.running_python", 0, 0)
							execStart := time.Now()
							safeResult := safety.ValidateAndExecute(ctx, generatedCode.Code, func(code string) (string, error) {
								if s.pythonPool != nil {
									return s.pythonPool.Execute(code, sessionDir)
								}
								ps := &PythonService{}
								return ps.ExecuteScript(s.cfg.PythonPath, code)
							})
							metrics.RecordExecution(time.Since(execStart))

							if safeResult.Success {
								parser := NewResultParser(s.Logger)
								parsedResult := parser.ParseOutput(safeResult.Output, sessionDir)
								if onFileSaved != nil {
									for _, f := range parsedResult.ChartFiles {
										onFileSaved(f.Name, f.Type, f.Size)
									}
									for _, f := range parsedResult.ExportFiles {
										onFileSaved(f.Name, f.Type, f.Size)
									}
								}
								metrics.LogSummary()
								emitProgressWithHeartbeat(StageComplete, 100, "progress.analysis_complete", 0, 0)
								trajectory.Success = true
								trajectory.FinalResponse = safeResult.Output
								trajectory.IterationCount = 1
								trajectory.ToolCallCount = 2
								if s.Logger != nil {
									s.Logger(fmt.Sprintf("[TIMING] Unified path took: %v", time.Since(startTotal)))
								}
								return &schema.Message{Role: schema.Assistant, Content: parser.FormatAsText(parsedResult)}, nil
							} else if s.Logger != nil {
								if safeResult.TimedOut {
									s.Logger(fmt.Sprintf("[UNIFIED] Timed out after %v, falling back", safeResult.Duration))
								} else {
									s.Logger(fmt.Sprintf("[UNIFIED] Execution failed: %v, falling back", safeResult.Error))
								}
							}
						} else if s.Logger != nil {
							s.Logger(fmt.Sprintf("[UNIFIED] Safety check failed: %v", safetyReport.Errors))
						}
					} else if s.Logger != nil {
						s.Logger(fmt.Sprintf("[UNIFIED] Code generation failed: %v, falling back", err))
					}
				} else if s.Logger != nil && path != PathUnified {
					s.Logger(fmt.Sprintf("[COMBINED] Routed to %s path, skipping unified", path))
				}

				// Build plan prompt from combined result (no extra LLM call needed)
				plan := combinedResult.ToAnalysisPlan()
				planner := NewAnalysisPlanner(s.ChatModel, s.Logger)
				planPrompt = planner.FormatPlanForPrompt(plan)
			}
		}
	}

	// 1. Initialize Tools (parallelized for speed, selective based on classification)
	startTools := time.Now()
	
	// Determine which tools are needed based on combined classification
	needsWebSearch := combinedResult == nil || combinedResult.NeedsWebSearch
	needsExport := combinedResult == nil || combinedResult.NeedsDataExport
	
	// Use sync.WaitGroup for parallel tool initialization
	var wg sync.WaitGroup
	var pyTool *PythonExecutorTool
	var dsTool *DataSourceContextTool
	var sqlTool *SQLExecutorTool
	var webSearchTool tool.BaseTool // Changed to interface to support multiple search implementations
	var webFetchTool *WebFetchTool  // HTTP-based web content fetcher (no Chrome dependency)
	var mcpTool *MCPTool
	var exportTool *ExportTool
	
	// Initialize Python tool (always needed for analysis)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if s.pythonPool != nil {
			pyTool = NewPythonExecutorToolWithPool(s.cfg, s.pythonPool)
		} else {
			pyTool = NewPythonExecutorTool(s.cfg)
		}
		pyTool.SetErrorKnowledge(s.errorKnowledge)
		if executionRecorder != nil {
			pyTool.SetExecutionRecorder(executionRecorder)
		}
		if sessionDir != "" {
			pyTool.SetSessionDirectory(sessionDir)
			if userMessageID != "" {
				pyTool.SetRequestID(userMessageID)
			}
			if onFileSaved != nil {
				pyTool.SetFileSavedCallback(onFileSaved)
			}
		}
	}()
	
	// Initialize DataSource tool
	wg.Add(1)
	go func() {
		defer wg.Done()
		dsTool = NewDataSourceContextTool(s.dsService)
		if s.workingContextManager != nil {
			dsTool.SetWorkingContextManager(s.workingContextManager)
		}
		if sqlCollector != nil {
			dsTool.SetSQLCollector(sqlCollector)
		}
	}()
	
	// Initialize SQL tool
	wg.Add(1)
	go func() {
		defer wg.Done()
		sqlPlanner := NewSQLPlanner(s.ChatModel, s.dsService, s.Logger)
		sqlTool = NewSQLExecutorToolWithPlanner(s.dsService, sqlPlanner, s.Logger)
		sqlTool.SetErrorKnowledge(s.errorKnowledge)
		if executionRecorder != nil {
			sqlTool.SetExecutionRecorder(executionRecorder)
		}
		if sqlCollector != nil {
			sqlTool.SetSQLCollector(sqlCollector)
			if len(history) > 0 {
				for i := len(history) - 1; i >= 0; i-- {
					if history[i].Role == schema.User {
						sqlCollector.SetUserRequest(history[i].Content)
						break
					}
				}
			}
		}
	}()
	
	// Initialize Web tools (only if needed based on classification)
	if needsWebSearch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Initialize search API configuration
			s.cfg.InitializeSearchAPIs()
			activeAPI := s.cfg.GetActiveSearchAPI()
		
		if activeAPI != nil && activeAPI.Enabled {
			searchTool, err := NewSearchAPITool(s.Logger, activeAPI, s.cfg.ProxyConfig)
			if err != nil {
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[SEARCH-API] Failed to initialize search tool: %v", err))
				}
				// Fallback to nil - will be handled later
				webSearchTool = nil
			} else {
				webSearchTool = searchTool
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[SEARCH-API] Initialized %s search API", activeAPI.Name))
				}
			}
		} else {
			if s.Logger != nil {
				s.Logger("[SEARCH-API] No active search API configured")
			}
			webSearchTool = nil
		}
		
		// Initialize HTTP-based web fetch tool (no Chrome dependency)
		webFetchTool = NewWebFetchTool(s.Logger, s.cfg.ProxyConfig)
		}()
	} else {
		// Always init web fetch for non-search use cases
		webFetchTool = NewWebFetchTool(s.Logger, s.cfg.ProxyConfig)
		if s.Logger != nil {
			s.Logger("[SEARCH-API] Skipped web search init (not needed for this request)")
		}
	}
	
	// Initialize MCP tool
	wg.Add(1)
	go func() {
		defer wg.Done()
		mcpTool = NewMCPTool(s.cfg.MCPServices, s.Logger)
	}()
	
	// Initialize Export tool (only if needed)
	if needsExport {
		wg.Add(1)
		go func() {
			defer wg.Done()
			exportTool = NewExportTool(s.Logger)
			if sessionDir != "" {
				exportTool.SetSessionDirectory(sessionDir)
				if userMessageID != "" {
					exportTool.SetRequestID(userMessageID)
				}
				if onFileSaved != nil {
					exportTool.SetFileSavedCallback(onFileSaved)
				}
			}
		}()
	} else {
		// Always create export tool but skip heavy init
		exportTool = NewExportTool(s.Logger)
		if sessionDir != "" {
			exportTool.SetSessionDirectory(sessionDir)
			if userMessageID != "" {
				exportTool.SetRequestID(userMessageID)
			}
			if onFileSaved != nil {
				exportTool.SetFileSavedCallback(onFileSaved)
			}
		}
	}
	
	// Wait for all tools to initialize
	wg.Wait()
	
	if sessionDir != "" && s.Logger != nil {
		s.Logger(fmt.Sprintf("[SESSION] Files will be saved to: %s", sessionDir))
	}

	// Build tools list - only add search tool if it was successfully initialized
	// Add composite query_and_chart tool for efficient visualization workflows
	queryChartTool := NewQueryAndChartTool(sqlTool, pyTool, s.Logger)
	tools := []tool.BaseTool{pyTool, dsTool, sqlTool, queryChartTool, webFetchTool, exportTool}
	
	if webSearchTool != nil {
		tools = append(tools, webSearchTool)
		if s.Logger != nil {
			activeAPI := s.cfg.GetActiveSearchAPI()
			if activeAPI != nil {
				s.Logger(fmt.Sprintf("[SEARCH-API] %s search tool added to agent", activeAPI.Name))
			}
		}
	}
	
	if s.Logger != nil {
		activeAPI := s.cfg.GetActiveSearchAPI()
		apiName := "none"
		if activeAPI != nil {
			apiName = activeAPI.Name
		}
		s.Logger(fmt.Sprintf("[WEB-TOOLS] Web search API: %s, Web fetch: HTTP-based (no Chrome)", apiName))
	}

	// Add MCP tool if services are configured
	if mcpTool.HasServices() {
		tools = append(tools, mcpTool)
		if s.Logger != nil {
			services := mcpTool.GetAvailableServices()
			s.Logger(fmt.Sprintf("[MCP] Loaded %d MCP service(s): %s", 
				len(services), strings.Join(services, ", ")))
		}
	} else {
		if s.Logger != nil {
			s.Logger("[MCP] No MCP services configured or enabled")
		}
	}

	// 2. Create ToolsNode (Standard Eino ToolsNode takes *Message and returns *Message)
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: tools,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tools node: %v", err)
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Tools Initialization took: %v", time.Since(startTools)))
	}

	// 3. Bind Tool Infos to Model
	startBind := time.Now()
	var toolInfos []*schema.ToolInfo
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		toolInfos = append(toolInfos, info)
	}
	err = s.ChatModel.BindTools(toolInfos)
	if err != nil {
		return nil, fmt.Errorf("failed to bind tools: %v", err)
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Binding Tools took: %v", time.Since(startBind)))
	}

	emitProgressWithHeartbeat(StageInitializing, 10, "progress.tools_ready", 0, 0)

	// 4. Build Graph using Lambda nodes to manage state ([]*schema.Message)
	startGraph := time.Now()
	g := compose.NewGraph[[]*schema.Message, []*schema.Message]()

	// Track iteration count for progress
	iterationCount := 0

	// Extract original user goal for attention refresh
	var originalUserGoal string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == schema.User {
			originalUserGoal = history[i].Content
			if len(originalUserGoal) > 200 {
				originalUserGoal = originalUserGoal[:200] + "..."
			}
			break
		}
	}

	// deduplicateMessages removes duplicate consecutive messages with same role and content
	deduplicateMessages := func(messages []*schema.Message) []*schema.Message {
		if len(messages) <= 1 {
			return messages
		}
		
		result := make([]*schema.Message, 0, len(messages))
		seen := make(map[string]bool)
		duplicateCount := 0
		
		for _, msg := range messages {
			// Create a unique key for this message
			key := fmt.Sprintf("%s:%s", msg.Role, msg.Content)
			
			// For user messages, always check for duplicates
			if msg.Role == schema.User {
				if seen[key] {
					// Skip duplicate user message
					duplicateCount++
					if s.Logger != nil {
						contentPreview := msg.Content
						if len(contentPreview) > 50 {
							contentPreview = contentPreview[:50] + "..."
						}
						s.Logger(fmt.Sprintf("[DEDUP] Filtered duplicate user message: %s", contentPreview))
					}
					continue
				}
				seen[key] = true
			}
			
			result = append(result, msg)
		}
		
		if duplicateCount > 0 && s.Logger != nil {
			s.Logger(fmt.Sprintf("[DEDUP] Removed %d duplicate message(s), %d -> %d messages", 
				duplicateCount, len(messages), len(result)))
		}
		
		return result
	}

	// Calculate max steps from config (used by both graph compile and early warnings)
	maxSteps := s.cfg.MaxAnalysisSteps
	if maxSteps < 10 {
		maxSteps = 25 // default
	} else if maxSteps > 50 {
		maxSteps = 50
	}

	// Define Model Node Wrapper
	modelLambda := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		iterationCount++
		startModel := time.Now()

		// Check for cancellation
		if cancelCheck != nil && cancelCheck() {
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[CANCEL] Analysis cancelled at step %d", iterationCount))
			}
			return nil, fmt.Errorf("analysis cancelled by user")
		}

		// ðŸ”´ CRITICAL: Remove duplicate messages before processing
		input = deduplicateMessages(input)

		// âš?EARLY WARNINGS: Encourage completion before hitting limits
		// Dynamic warnings based on maxSteps and estimated complexity
		// Warnings at ~60%, ~70%, ~80% of max iterations (maxSteps/2 since each round = 2 steps)
		maxIter := maxSteps / 2
		warningStep1 := max(4, maxIter*60/100)  // ~60% of max iterations
		warningStep2 := max(5, maxIter*70/100)  // ~70% of max iterations
		warningStep3 := max(6, maxIter*80/100)  // ~80% of max iterations
		
		// Adjust warning thresholds for complex analyses (push later)
		if combinedResult != nil && combinedResult.EstimatedCalls >= 5 {
			warningStep1 = max(warningStep1, maxIter*70/100)
			warningStep2 = max(warningStep2, maxIter*80/100)
			warningStep3 = max(warningStep3, maxIter*90/100)
		}
		
		if iterationCount == warningStep1 {
			warningMsg := &schema.Message{
				Role:    schema.User,
				Content: "âš?Too many steps used. Wrap up the analysis soon, use at most 2 more tool calls.",
			}
			input = append(input, warningMsg)
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[WARNING] Step %d warning injected", iterationCount))
			}
		} else if iterationCount == warningStep2 {
			warningMsg := &schema.Message{
				Role:    schema.User,
				Content: "âš ï¸ Too many steps. Present results immediately, do not call any more tools.",
			}
			input = append(input, warningMsg)
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[WARNING] Step %d warning injected", iterationCount))
			}
		} else if iterationCount == warningStep3 {
			finalMsg := &schema.Message{
				Role:    schema.User,
				Content: "ðŸ›‘ STOP! Output current results immediately.",
			}
			input = append(input, finalMsg)
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[FINAL-WARNING] Step %d final warning injected", iterationCount))
			}
		}

		// Emit progress based on iteration
		progress := 20 + min(iterationCount*10, 60) // 20-80%
		emitProgressWithHeartbeat(StageAnalysis, progress, "progress.ai_processing", iterationCount, 0)

		// HARD LIMIT: Force termination if iterations exceed safe threshold
		// This prevents runaway resource consumption even if warnings are ignored
		if iterationCount > 12 {
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[HARD-LIMIT] Forcing termination at step %d to prevent resource exhaustion", iterationCount))
			}
			// Return a message asking the model to output results immediately
			forceStopMsg := &schema.Message{
				Role:    schema.Assistant,
				Content: "Analysis step limit reached. Outputting current results now.",
			}
			return append(input, forceStopMsg), nil
		}

		// Apply memory management only if enabled in config
		managedInput := input
		if s.cfg.EnableMemory && s.memoryManager != nil {
			var err error
			managedInput, err = s.memoryManager.ManageMemory(ctx, input)
			if err != nil {
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[WARNING] Memory management failed in graph: %v", err))
				}
				managedInput = input
			}

			// Log token reduction if significant
			if s.Logger != nil && len(input) != len(managedInput) {
				originalTokens := s.memoryManager.EstimateTokens(input)
				managedTokens := s.memoryManager.EstimateTokens(managedInput)
				s.Logger(fmt.Sprintf("[MEMORY-GRAPH] Reduced from %d to %d messages (%d -> %d est. tokens)",
					len(input), len(managedInput), originalTokens, managedTokens))
			}
		} else if s.Logger != nil && !s.cfg.EnableMemory {
			s.Logger("[MEMORY] Memory management disabled by config")
		}

		// Call model with managed history
		resp, err := s.ChatModel.Generate(ctx, managedInput)
		if err != nil {
			// Record error in trajectory
			step := TrajectoryStep{
				StepNumber: len(trajectory.Steps) + 1,
				Timestamp:  time.Now().UnixMilli(),
				Type:       "model_call",
				ModelInput: messagesToMap(managedInput),
				Error:      err.Error(),
			}
			trajectory.Steps = append(trajectory.Steps, step)
			return nil, err
		}

		// Record successful model call in trajectory with escaped content
		step := TrajectoryStep{
			StepNumber:  len(trajectory.Steps) + 1,
			Timestamp:   time.Now().UnixMilli(),
			Type:        "model_call",
			ModelInput:  messagesToMap(managedInput),
			ModelOutput: messageToMap(resp),
		}
		trajectory.Steps = append(trajectory.Steps, step)
		trajectory.IterationCount = iterationCount

		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TIMING] Model Generation step took: %v", time.Since(startModel)))
		}
		// Append response to managed history (use managedInput to avoid duplicates)
		return append(managedInput, resp), nil
	})

	// Define Tool Node Wrapper
	toolsLambda := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		startExec := time.Now()
		
		// Check for cancellation before executing tools
		if cancelCheck != nil && cancelCheck() {
			if s.Logger != nil {
				s.Logger("[CANCEL] Analysis cancelled before tool execution")
			}
			return nil, fmt.Errorf("analysis cancelled by user")
		}
		
		// Get the last message (which should be Assistant with ToolCalls)
		if len(input) == 0 {
			return nil, fmt.Errorf("tool node received empty history")
		}
		lastMsg := input[len(input)-1]

		// Emit progress based on tool being called
		if len(lastMsg.ToolCalls) > 0 {
			toolName := lastMsg.ToolCalls[0].Function.Name
			
			// Use centralized tool-to-progress mapping
			if mapping, ok := ToolProgressMapping[toolName]; ok {
				emitProgressWithHeartbeat(mapping.Stage, mapping.Progress, mapping.Message, 0, 0)
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[PROGRESS] %s â†?%s (%s)", toolName, mapping.Stage, mapping.Message))
				}
			} else {
				emitProgressWithHeartbeat(StageAnalysis, 50, "progress.ai_processing", 0, 0)
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[PROGRESS] Running %s", toolName))
				}
			}
		}

		// Execute tools
		toolResultMsg, err := toolsNode.Invoke(ctx, lastMsg)

		// Record tool calls in trajectory with escaped content for training visibility
		for _, tc := range lastMsg.ToolCalls {
			step := TrajectoryStep{
				StepNumber: len(trajectory.Steps) + 1,
				Timestamp:  time.Now().UnixMilli(),
				Type:       "tool_call",
				ToolName:   tc.Function.Name,
				ToolInput:  escapeForTraining(tc.Function.Arguments),
				ToolCallID: tc.ID,
			}

			if err != nil {
				step.Error = escapeForTraining(err.Error())
			} else if len(toolResultMsg) > 0 {
				// Find matching tool result for this call - record escaped output for training visibility
				for _, resultMsg := range toolResultMsg {
					if resultMsg.ToolCallID == tc.ID {
						step.ToolOutput = escapeForTraining(resultMsg.Content)
						break
					}
				}
			}

			trajectory.Steps = append(trajectory.Steps, step)
			trajectory.ToolCallCount++
		}

		if err != nil {
			// Instead of failing the graph, return error as tool result so LLM can retry
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[TOOL ERROR] %v - returning as message for LLM to handle", err))
			}
			
			// Emit progress indicating a retry is happening
			if len(lastMsg.ToolCalls) > 0 {
				toolName := lastMsg.ToolCalls[0].Function.Name
				if toolName == "execute_sql" {
					emitProgressWithHeartbeat(StageQuery, 35, "progress.correcting_sql", 0, 0)
				} else {
					emitProgressWithHeartbeat(StageAnalysis, 45, "progress.ai_processing", 0, 0)
				}
			}
			
			// Create error messages for each tool call with helpful guidance
			var errorMsgs []*schema.Message
			errStr := err.Error()
			for _, tc := range lastMsg.ToolCalls {
				var helpMsg string
				toolName := tc.Function.Name

				if toolName == "execute_sql" {
					if strings.Contains(errStr, "no such column") || strings.Contains(errStr, "Unknown column") {
						helpMsg = fmt.Sprintf("â?SQL Column Error: %v\n\n", err)
						helpMsg += "ðŸ”§ REQUIRED ACTION:\n"
						helpMsg += "1. Call get_data_source_context to see actual column names\n"
						helpMsg += "2. If using subquery, ensure ALL columns needed by outer query are in subquery's SELECT\n"
						helpMsg += "3. Rewrite and execute the corrected query"
					} else if strings.Contains(errStr, "syntax error") {
						helpMsg = fmt.Sprintf("â?SQL Syntax Error: %v\n\n", err)
						helpMsg += "ðŸ”§ For DuckDB, use: strftime('%Y',col) not YEAR(), col1||col2 not CONCAT()"
					} else {
						helpMsg = fmt.Sprintf("â?SQL Error: %v\n\nðŸ”§ Please fix and retry.", err)
					}
				} else if toolName == "python_executor" {
					helpMsg = fmt.Sprintf("â?Python Error: %v\n\nðŸ”§ Please fix the code and retry.", err)
				} else {
					helpMsg = fmt.Sprintf("â?Tool Error: %v\n\nðŸ”§ Please fix and retry.", err)
				}

				errorMsgs = append(errorMsgs, &schema.Message{
					Role:       schema.Tool,
					Content:    helpMsg,
					ToolCallID: tc.ID,
				})
			}
			if len(errorMsgs) == 0 {
				// Fallback if no tool calls found
				errorMsgs = append(errorMsgs, &schema.Message{
					Role:    schema.Tool,
					Content: fmt.Sprintf("Error: %v\n\nðŸ”´ Please fix the issue and try again.", err),
				})
			}
			return append(input, errorMsgs...), nil
		}
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TIMING] Tools Execution step took: %v", time.Since(startExec)))
		}

		// Emit progress for result processing
		emitProgressWithHeartbeat(StageAnalysis, 65, "progress.processing_results", 0, 0)

		// CRITICAL: Truncate tool output to prevent context overflow
		// Tool outputs (especially SQL results) can be huge
		const maxToolOutputChars = 30000 // Reduced from 50000 to prevent memory bloat during long analyses
		for i, msg := range toolResultMsg {
			if msg.Role == schema.Tool && len(msg.Content) > maxToolOutputChars {
				toolResultMsg[i] = &schema.Message{
					Role:       msg.Role,
					Content:    msg.Content[:maxToolOutputChars] + fmt.Sprintf("\n\n[... Output truncated - %d chars omitted for context limit]", len(msg.Content)-maxToolOutputChars),
					ToolCallID: msg.ToolCallID,
				}
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[MEMORY] Truncated tool output from %d to %d chars", len(msg.Content), maxToolOutputChars))
				}
			}
		}

		// Stream tool output to frontend
		if len(toolResultMsg) > 0 && onProgress != nil {
			for _, msg := range toolResultMsg {
				if msg.Role == schema.Tool && msg.Content != "" {
					// Get tool name from the original call
					toolName := ""
					if len(lastMsg.ToolCalls) > 0 {
						toolName = lastMsg.ToolCalls[0].Function.Name
					}

					// Truncate output for streaming preview (keep full in final response)
					preview := msg.Content
					if len(preview) > 200 {
						preview = preview[:200] + "..."
					}

					onProgress(ProgressUpdate{
						Stage:      "tool_output",
						Progress:   65,
						Message:    "progress.tool_completed",
						Step:       4,
						Total:      6,
						ToolName:   toolName,
						ToolOutput: preview,
					})
				}
			}
		}

		// Append tool result to history
		return append(input, toolResultMsg...), nil
	})

	err = g.AddLambdaNode("model", modelLambda)
	if err != nil {
		return nil, err
	}

	err = g.AddLambdaNode("tools", toolsLambda)
	if err != nil {
		return nil, err
	}

	err = g.AddEdge(compose.START, "model")
	if err != nil {
		return nil, err
	}

	// Branch: loop back to tools or end
	err = g.AddBranch("model", compose.NewGraphBranch(func(ctx context.Context, history []*schema.Message) (string, error) {
		lastMsg := history[len(history)-1]
		if len(lastMsg.ToolCalls) > 0 {
			return "tools", nil
		}
		return compose.END, nil
	}, map[string]bool{"tools": true, compose.END: true}))
	if err != nil {
		return nil, err
	}

	err = g.AddEdge("tools", "model")
	if err != nil {
		return nil, err
	}

	// 5. Compile and Run with configurable max steps
	// Each modelâ†’tools round trip counts as 2 steps
	runnable, err := g.Compile(ctx, compose.WithMaxRunSteps(maxSteps))
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %v", err)
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Construction & Compilation took: %v", time.Since(startGraph)))
	}

	emitProgressWithHeartbeat(StageInitializing, 15, "progress.tools_ready", 0, 0)

	// 6. Build Context Prompt (include table names and column names for data background)
	startContext := time.Now()
	var contextPrompt string
	var dbType string = "duckdb"
	if dataSourceID != "" && s.dsService != nil {
		sources, _ := s.dsService.LoadDataSources()
		for _, ds := range sources {
			if ds.ID == dataSourceID {
				// Determine database type
				if ds.Config.DBPath != "" {
					dbType = "duckdb"
				} else if ds.Type == "mysql" || ds.Type == "doris" {
					dbType = ds.Type
				}

				contextPrompt = fmt.Sprintf("\n\n## Data Source\nName: %s (ID: %s, Type: %s)\n", ds.Name, ds.ID, strings.ToUpper(dbType))
				if ds.Analysis != nil && ds.Analysis.Summary != "" {
					contextPrompt += fmt.Sprintf("Summary: %s\n", ds.Analysis.Summary)
				}

				// Include table names AND column names for better data background
				if ds.Analysis != nil && len(ds.Analysis.Schema) > 0 {
					contextPrompt += "\n### Tables & Columns\n"
					for _, t := range ds.Analysis.Schema {
						contextPrompt += fmt.Sprintf("- **%s**: %s\n", t.TableName, strings.Join(t.Columns, ", "))
					}
				} else {
					// Fallback: try to get tables and columns directly
					tables, err := s.dsService.GetDataSourceTables(dataSourceID)
					if err == nil && len(tables) > 0 {
						contextPrompt += "\n### Tables & Columns\n"
						for _, tbl := range tables {
							cols, err := s.dsService.GetDataSourceTableColumns(dataSourceID, tbl)
							if err == nil && len(cols) > 0 {
								contextPrompt += fmt.Sprintf("- **%s**: %s\n", tbl, strings.Join(cols, ", "))
							} else {
								contextPrompt += fmt.Sprintf("- **%s**\n", tbl)
							}
						}
					}
				}
				contextPrompt += "\nðŸ’¡ Column names above are exact (case-sensitive). For simple queries with obvious columns, you may write SQL directly. Call get_data_source_context only if you need sample data, data types, or relationship info.\n"

				// SQL dialect
				if dbType == "duckdb" {
					contextPrompt += `Dialect: DuckDB (use strftime, ||, no YEAR/MONTH)`
				} else if dbType == "mysql" || dbType == "doris" {
					contextPrompt += `Dialect: MySQL (use YEAR/MONTH, CONCAT)`
				}
				break
			}
		}
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Context Prompt preparation took: %v", time.Since(startContext)))
		if contextPrompt != "" {
			s.Logger(fmt.Sprintf("[CONTEXT] Data context prompt length: %d chars", len(contextPrompt)))
		} else {
			s.Logger("[CONTEXT] WARNING: No data context prompt generated!")
		}
	}

	// Load working context if available for context-aware analysis
	var workingContextPrompt string
	if threadID != "" && s.workingContextManager != nil {
		if ctx := s.workingContextManager.GetContext(threadID); ctx != nil {
			workingContextPrompt = ctx.FormatForPrompt()
			if s.Logger != nil {
				s.Logger("[WORKING-CONTEXT] Loaded context for prompt injection")
			}
		}
	}

	// Load conversation context for better follow-up understanding
	var conversationContextPrompt string
	if threadID != "" && s.conversationContextManager != nil {
		conversationContextPrompt = s.conversationContextManager.GetContextForPrompt(threadID)
		if conversationContextPrompt != "" && s.Logger != nil {
			s.Logger("[CONVERSATION-CONTEXT] Loaded conversation context for prompt injection")
		}
	}

	// Build MCP tools prompt if services are available
	var mcpToolsPrompt string
	if len(s.cfg.MCPServices) > 0 {
		// Filter enabled and tested services
		var availableServices []string
		for _, svc := range s.cfg.MCPServices {
			if svc.Enabled && svc.Tested {
				availableServices = append(availableServices, 
					fmt.Sprintf("  â€?%s: %s", svc.Name, svc.Description))
			}
		}
		
		if len(availableServices) > 0 {
			mcpToolsPrompt = "\n\nðŸ”Œ MCP SERVICES (External capabilities):\n"
			mcpToolsPrompt += strings.Join(availableServices, "\n")
			mcpToolsPrompt += "\n- Use mcp_service tool to call these services"
			mcpToolsPrompt += "\n- Specify service_name, method (GET/POST), and endpoint"
			mcpToolsPrompt += "\n- Useful for accessing external APIs and real-time data"
			
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[MCP-PROMPT] Added %d MCP service(s) to system prompt", len(availableServices)))
			}
		}
	}

	// Add analysis plan to prompt if available
	analysisPlanPrompt := ""
	if planPrompt != "" {
		analysisPlanPrompt = planPrompt
	}

	// Detect user language from the last message to respond in the same language
	languageDirective := detectResponseLanguage(lastUserMessage)
	sysMsg := &schema.Message{
		Role:    schema.System,
		Content: buildAnalysisSystemPrompt() + analysisPlanPrompt + contextPrompt + workingContextPrompt + conversationContextPrompt + mcpToolsPrompt + languageDirective,
	}

	// 7. Apply memory management to history (only if enabled)
	startMemory := time.Now()
	managedHistory := history
	if s.cfg.EnableMemory && s.memoryManager != nil {
		var err error
		managedHistory, err = s.memoryManager.ManageMemory(ctx, history)
		if err != nil {
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[WARNING] Memory management failed: %v, using original history", err))
			}
			managedHistory = history
		}
		if s.Logger != nil {
			originalTokens := s.memoryManager.EstimateTokens(history)
			managedTokens := s.memoryManager.EstimateTokens(managedHistory)
			s.Logger(fmt.Sprintf("[MEMORY] Original: %d msgs (%d est. tokens) -> Managed: %d msgs (%d est. tokens)",
				len(history), originalTokens, len(managedHistory), managedTokens))
			s.Logger(fmt.Sprintf("[TIMING] Memory Management took: %v", time.Since(startMemory)))
		}
	} else if s.Logger != nil {
		s.Logger("[MEMORY] Memory management disabled - using raw history")
	}

	input := append([]*schema.Message{sysMsg}, managedHistory...)

	emitProgressWithHeartbeat(StageAnalysis, 20, "progress.ai_processing", 0, 0)

	startInvoke := time.Now()
	finalHistory, err := runnable.Invoke(ctx, input)
	if err != nil {
		// ALWAYS emit completion progress so frontend progress bar clears properly
		emitProgressWithHeartbeat(StageComplete, 100, "progress.analysis_complete", 0, 0)

		// Mark trajectory as failed
		trajectory.Success = false
		trajectory.ErrorMessage = err.Error()
		return nil, err
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Execution (Invoke) took: %v", time.Since(startInvoke)))
		s.Logger(fmt.Sprintf("[TIMING] Total RunAnalysis took: %v", time.Since(startTotal)))
	}

	emitProgressWithHeartbeat(StageComplete, 100, "progress.analysis_complete", 0, 0)

	// Return the last message and mark trajectory as successful with escaped content
	if len(finalHistory) > 0 {
		lastMsg := finalHistory[len(finalHistory)-1]
		trajectory.Success = true
		trajectory.FinalResponse = escapeForTraining(lastMsg.Content) // Escape for training visibility
		
		// Update conversation context with assistant response
		if s.conversationContextManager != nil && threadID != "" && lastMsg.Role == schema.Assistant {
			// Extract tool used from history
			var lastToolUsed string
			var lastToolResult string
			for i := len(finalHistory) - 1; i >= 0; i-- {
				if finalHistory[i].Role == schema.Assistant && len(finalHistory[i].ToolCalls) > 0 {
					lastToolUsed = finalHistory[i].ToolCalls[0].Function.Name
					break
				}
				if finalHistory[i].Role == schema.Tool {
					lastToolResult = finalHistory[i].Content
				}
			}
			s.conversationContextManager.UpdateFromAssistantResponse(threadID, lastMsg.Content, lastToolUsed, lastToolResult)
			if s.Logger != nil {
				s.Logger("[CONTEXT] Updated conversation context with assistant response")
			}
		}
		
		// Extract and store valuable memories (only if memory is enabled and analysis was successful)
		// Run asynchronously to not block user response
		if s.cfg.EnableMemory && lastMsg.Role == schema.Assistant && lastMsg.Content != "" {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						if s.Logger != nil {
							s.Logger(fmt.Sprintf("[MEMORY] Panic in memory extraction goroutine: %v", r))
						}
					}
				}()
				startMemoryExtraction := time.Now()
				
				// Collect SQL queries and results from history
				var sqlQueries []string
				var dataResults []map[string]interface{}
				var userQuery string
				
				// Extract user query from history
				for i := len(finalHistory) - 1; i >= 0; i-- {
					if finalHistory[i].Role == schema.User {
						userQuery = finalHistory[i].Content
						break
					}
				}
				
				// Extract SQL queries and results from tool calls
				for i, msg := range finalHistory {
					if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
						for _, tc := range msg.ToolCalls {
							if tc.Function.Name == "execute_sql" {
								// Parse arguments to get SQL query
								var args map[string]interface{}
								if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
									if query, ok := args["query"].(string); ok {
										sqlQueries = append(sqlQueries, query)
									}
								}
								
								// Look for corresponding tool result in next messages
								for j := i + 1; j < len(finalHistory); j++ {
									if finalHistory[j].Role == schema.Tool && finalHistory[j].ToolCallID == tc.ID {
										// Try to parse tool result as data
										var result []map[string]interface{}
										if err := json.Unmarshal([]byte(finalHistory[j].Content), &result); err == nil {
											dataResults = append(dataResults, result...)
										}
										break
									}
								}
							}
						}
					}
				}
				
				// Create memory extractor and extract key findings
				if len(sqlQueries) > 0 || userQuery != "" {
					extractor := NewMemoryExtractor(s.ChatModel, s.Logger)
					memories := extractor.ExtractKeyFindings(
						context.Background(), // Use background context for async operation
						userQuery,
						lastMsg.Content,
						sqlQueries,
						dataResults,
					)
					
					if s.Logger != nil && len(memories) > 0 {
						s.Logger(fmt.Sprintf("[MEMORY] Extracted %d valuable memories from analysis", len(memories)))
					}
					
					// Store memories using MemoryService based on tier
					if s.memoryService != nil {
						for _, mem := range memories {
							var err error
							
							// Route to appropriate memory tier
							switch mem.Tier {
							case LongTermTier:
								// Long-term: persistent facts (schemas, rules, data characteristics)
								err = s.memoryService.AddSessionLongTermMemory(threadID, mem.Content)
								if err != nil && s.Logger != nil {
									s.Logger(fmt.Sprintf("[MEMORY] Failed to store long-term memory: %v", err))
								}
							case MidTermTier:
								// Mid-term: compressed summaries (not used here, managed by MemoryManager)
								err = s.memoryService.AddSessionMediumTermMemory(threadID, mem.Content)
								if err != nil && s.Logger != nil {
									s.Logger(fmt.Sprintf("[MEMORY] Failed to store mid-term memory: %v", err))
								}
							case ShortTermTier:
								// Short-term: current context (not persisted, managed by MemoryManager)
								// Skip persistence for short-term memories
								continue
							}
							
							if err == nil && s.Logger != nil {
								s.Logger(fmt.Sprintf("[MEMORY] âœ?Stored [%s] %s: %s", 
									mem.Tier,
									mem.Category,
									mem.Content))
							}
						}
					} else if s.Logger != nil {
						// Log only if memoryService is not available
						for _, mem := range memories {
							s.Logger(fmt.Sprintf("[MEMORY] [%s] %s: %s", 
								mem.Tier,
								mem.Category,
								mem.Content))
						}
					}
					
					if s.Logger != nil {
						s.Logger(fmt.Sprintf("[TIMING] Memory extraction took: %v (async)", time.Since(startMemoryExtraction)))
					}
				}
			}()
		}
		
		return lastMsg, nil
	}

	// No response - mark as failed
	trajectory.Success = false
	trajectory.ErrorMessage = "agent returned empty history"
	return nil, fmt.Errorf("agent returned empty history")
}

// buildAnalysisSystemPrompt builds the main analysis system prompt.
// Instead of detecting and hardcoding a specific language, we instruct the LLM
// to always respond in the same language as the user's message.
// This naturally supports all languages (Chinese, English, Japanese, Korean, French, etc.)
// detectResponseLanguage analyzes the user's message and returns a language directive
// to append to the system prompt. This is placed at the END of the prompt (closest to
// generation) to maximize its influence on Chinese-tuned models that tend to default
// to Chinese even when the user writes in English.
func detectResponseLanguage(userMessage string) string {
	if userMessage == "" {
		return ""
	}

	chineseCount := 0
	japaneseCount := 0
	totalCount := 0
	for _, r := range userMessage {
		if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3400 && r <= 0x4DBF {
			chineseCount++
		}
		// Hiragana + Katakana
		if r >= 0x3040 && r <= 0x309F || r >= 0x30A0 && r <= 0x30FF {
			japaneseCount++
		}
		if r > 32 { // count non-whitespace
			totalCount++
		}
	}

	if totalCount == 0 {
		return ""
	}

	chineseRatio := float64(chineseCount) / float64(totalCount)
	japaneseRatio := float64(japaneseCount) / float64(totalCount)

	if chineseRatio > 0.3 {
		// User is writing in Chinese â€?no extra directive needed,
		// Chinese-tuned models naturally respond in Chinese
		return ""
	}

	if japaneseRatio > 0.1 {
		return "\n\nðŸš¨ **RESPONSE LANGUAGE: You MUST respond in Japanese (æ—¥æœ¬èª?.** The user's message is in Japanese. All output must be in Japanese. Do NOT use Chinese."
	}

	// User is writing in a non-CJK language (likely English).
	// Add a strong directive at the end of the system prompt to override
	// the model's Chinese default. Position matters: end-of-prompt has
	// the strongest influence on generation.
	return "\n\nðŸš¨ **RESPONSE LANGUAGE: You MUST respond in English.** The user's message is in English. All text output â€?analysis, insights, suggestions, chart titles, labels â€?must be in English. Do NOT use Chinese."
}

func buildAnalysisSystemPrompt() string {
	// Use internationalized system prompt from i18n package
	return i18n.GetAnalysisSystemPrompt()
}


// saveTrajectory saves the trajectory to session directory for training use
func (s *EinoService) saveTrajectory(sessionDir string, trajectory *AgentTrajectory) {
	if sessionDir == "" || trajectory == nil {
		return
	}

	// Finalize trajectory
	trajectory.EndTime = time.Now().UnixMilli()
	trajectory.TotalDuration = trajectory.EndTime - trajectory.StartTime

	// Create trajectory directory
	trajectoryDir := filepath.Join(sessionDir, "trajectory")
	if err := os.MkdirAll(trajectoryDir, 0755); err != nil {
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TRAJECTORY] Failed to create directory: %v", err))
		}
		return
	}

	// Generate filename based on timestamp
	filename := fmt.Sprintf("%d.json", trajectory.StartTime)
	filePath := filepath.Join(trajectoryDir, filename)

	// Create JSON encoder with proper settings for complete data preservation
	file, err := os.Create(filePath)
	if err != nil {
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TRAJECTORY] Failed to create file: %v", err))
		}
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // Preserve HTML characters in content

	// Encode trajectory to JSON with proper escaping
	if err := encoder.Encode(trajectory); err != nil {
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TRAJECTORY] Failed to encode JSON: %v", err))
		}
		return
	}

	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TRAJECTORY] Saved to: %s (%d steps, %d tool calls, %dms)",
			filePath, len(trajectory.Steps), trajectory.ToolCallCount, trajectory.TotalDuration))
		
		// Verify JSON format by attempting to read it back
		if err := s.verifyTrajectoryJSON(filePath); err != nil {
			s.Logger(fmt.Sprintf("[TRAJECTORY] JSON format verification failed: %v", err))
		} else {
			s.Logger("[TRAJECTORY] JSON format verified successfully")
		}
	}
}

// verifyTrajectoryJSON verifies that the saved trajectory file is valid JSON
func (s *EinoService) verifyTrajectoryJSON(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var trajectory AgentTrajectory
	if err := decoder.Decode(&trajectory); err != nil {
		return fmt.Errorf("JSON decode failed: %v", err)
	}

	// Additional verification: check if final_response can be extracted
	if trajectory.FinalResponse != "" {
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TRAJECTORY] Final response length: %d chars (escaped)", len(trajectory.FinalResponse)))
			// Log first 100 chars to verify escaped content is preserved correctly
			preview := trajectory.FinalResponse
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			s.Logger(fmt.Sprintf("[TRAJECTORY] Final response preview (escaped): %s", preview))
		}
	}

	return nil
}

// messagesToMap converts messages to simplified map representation for trajectory
func messagesToMap(msgs []*schema.Message) []map[string]interface{} {
	var result []map[string]interface{}
	for _, msg := range msgs {
		result = append(result, messageToMap(msg))
	}
	return result
}

// escapeForTraining converts content to escaped format for better training visibility
func escapeForTraining(content string) string {
	// IMPORTANT: Backslash must be escaped FIRST to avoid double-escaping
	// the backslashes introduced by subsequent replacements.
	content = strings.ReplaceAll(content, "\\", "\\\\")
	content = strings.ReplaceAll(content, "\n", "\\n")
	content = strings.ReplaceAll(content, "\r", "\\r")
	content = strings.ReplaceAll(content, "\t", "\\t")
	content = strings.ReplaceAll(content, "\"", "\\\"")
	return content
}

// messageToMap converts a single message to map with escaped content for training visibility
func messageToMap(msg *schema.Message) map[string]interface{} {
	m := map[string]interface{}{
		"role": string(msg.Role),
	}

	// Escape content for training visibility - show actual escape sequences
	m["content"] = escapeForTraining(msg.Content)

	// Add complete tool calls information if present
	if len(msg.ToolCalls) > 0 {
		var toolCalls []map[string]interface{}
		for _, tc := range msg.ToolCalls {
			toolCall := map[string]interface{}{
				"id":        tc.ID,
				"name":      tc.Function.Name,
				"arguments": escapeForTraining(tc.Function.Arguments),
			}
			toolCalls = append(toolCalls, toolCall)
		}
		m["tool_calls"] = toolCalls
	}

	// Add tool call ID if this is a tool response
	if msg.ToolCallID != "" {
		m["tool_call_id"] = msg.ToolCallID
	}

	// Add tool name if present
	if msg.ToolName != "" {
		m["tool_name"] = msg.ToolName
	}

	return m
}
