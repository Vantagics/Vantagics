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

	"vantagedata/agent/templates"
	"vantagedata/config"
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
	
	if logger != nil {
		logger(fmt.Sprintf("[EINO-INIT] Creating EinoService with provider: %s, model: %s", cfg.LLMProvider, cfg.ModelName))
	}
	
	var chatModel model.ChatModel
	var err error

	switch cfg.LLMProvider {
	case "Anthropic":
		if logger != nil {
			logger(fmt.Sprintf("[EINO-INIT] Initializing Anthropic model: %s", cfg.ModelName))
		}
		chatModel, err = NewAnthropicChatModel(context.Background(), &AnthropicConfig{
			APIKey:    cfg.APIKey,
			BaseURL:   cfg.BaseURL,
			Model:     cfg.ModelName,
			MaxTokens: cfg.MaxTokens,
		})
	case "Gemini":
		if logger != nil {
			logger(fmt.Sprintf("[EINO-INIT] Initializing Gemini model: %s", cfg.ModelName))
		}
		chatModel, err = NewGeminiChatModel(context.Background(), &GeminiConfig{
			APIKey:    cfg.APIKey,
			BaseURL:   cfg.BaseURL,
			Model:     cfg.ModelName,
			MaxTokens: cfg.MaxTokens,
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
				APIKey:    cfg.APIKey,
				BaseURL:   cfg.BaseURL,
				Model:     cfg.ModelName,
				MaxTokens: cfg.MaxTokens,
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
				APIKey:    cfg.APIKey,
				BaseURL:   normalizedBaseURL,
				Model:     cfg.ModelName,
				MaxTokens: &maxTokens, // Use pointer to int
				Timeout:   0, // Default
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
	skillsDir := filepath.Join(dsService.dataCacheDir, "..", "skills") // Skills in VantageData/skills
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
		
		// Resolve references in user message (e.g., "天气" -> "北京的天气")
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

	emitProgress(StageInitializing, 5, "progress.initializing_tools", 0, 0)

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
					emitProgress(stage, progress, message, step, total)
				}

				result, err := template.Execute(ctx, executor, dataSourceID, templateProgress)
				if err == nil && result.Success {
					emitProgress(StageComplete, 100, "progress.analysis_complete", 0, 0)
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

	// 🚀 Combined Classification + Planning: Single LLM call replaces two separate calls
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
			dataSourceInfo := "无数据源"
			var dbPath string
			if dataSourceID != "" {
				if sources, err := s.dsService.LoadDataSources(); err == nil {
					for _, ds := range sources {
						if ds.ID == dataSourceID {
							dbPath = ds.Config.DBPath
							tables, _ := s.dsService.GetDataSourceTables(dataSourceID)
							dataSourceInfo = fmt.Sprintf("数据源: %s, 表: %s", ds.Name, strings.Join(tables, ", "))
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
						emitProgress(StageComplete, 100, "progress.analysis_complete", 0, 0)
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

					emitProgress(StageAnalysis, 30, "progress.generating_code", 0, 0)
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

							emitProgress(StageAnalysis, 60, "progress.running_python", 0, 0)
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
								emitProgress(StageComplete, 100, "progress.analysis_complete", 0, 0)
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
			searchTool, err := NewSearchAPITool(s.Logger, activeAPI)
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
	tools := []tool.BaseTool{pyTool, dsTool, sqlTool, webFetchTool, exportTool}
	
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

	emitProgress(StageInitializing, 10, "progress.tools_ready", 0, 0)

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

		// 🔴 CRITICAL: Remove duplicate messages before processing
		input = deduplicateMessages(input)

		// ⚡ EARLY WARNINGS: Encourage completion before hitting limits
		// Dynamic warnings based on estimated complexity
		warningStep1 := 6  // Default first warning at step 6
		warningStep2 := 8  // Default second warning at step 8
		warningStep3 := 10 // Default final warning at step 10
		
		// Adjust warning thresholds for complex analyses
		if combinedResult != nil && combinedResult.EstimatedCalls >= 5 {
			warningStep1 = 8
			warningStep2 = 10
			warningStep3 = 12
		}
		
		if iterationCount == warningStep1 {
			warningMsg := &schema.Message{
				Role:    schema.User,
				Content: "⚡ 已用较多步骤。尽快完成分析，最多再用2次工具。",
			}
			input = append(input, warningMsg)
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[WARNING] Step %d warning injected", iterationCount))
			}
		} else if iterationCount == warningStep2 {
			warningMsg := &schema.Message{
				Role:    schema.User,
				Content: "⚠️ 步骤较多。立即呈现结果,不要再调用工具。",
			}
			input = append(input, warningMsg)
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[WARNING] Step %d warning injected", iterationCount))
			}
		} else if iterationCount == warningStep3 {
			finalMsg := &schema.Message{
				Role:    schema.User,
				Content: "🛑 停止! 立即输出当前结果。",
			}
			input = append(input, finalMsg)
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[FINAL-WARNING] Step %d final warning injected", iterationCount))
			}
		}

		// Emit progress based on iteration
		progress := 20 + min(iterationCount*10, 60) // 20-80%
		emitProgress(StageAnalysis, progress, "progress.ai_processing", iterationCount, 0)

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
				emitProgress(mapping.Stage, mapping.Progress, mapping.Message, 0, 0)
				if s.Logger != nil {
					s.Logger(fmt.Sprintf("[PROGRESS] %s → %s (%s)", toolName, mapping.Stage, mapping.Message))
				}
			} else {
				emitProgress(StageAnalysis, 50, "progress.ai_processing", 0, 0)
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
					emitProgress(StageQuery, 35, "progress.correcting_sql", 0, 0)
				} else {
					emitProgress(StageAnalysis, 45, "progress.ai_processing", 0, 0)
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
						helpMsg = fmt.Sprintf("❌ SQL Column Error: %v\n\n", err)
						helpMsg += "🔧 REQUIRED ACTION:\n"
						helpMsg += "1. Call get_data_source_context to see actual column names\n"
						helpMsg += "2. If using subquery, ensure ALL columns needed by outer query are in subquery's SELECT\n"
						helpMsg += "3. Rewrite and execute the corrected query"
					} else if strings.Contains(errStr, "syntax error") {
						helpMsg = fmt.Sprintf("❌ SQL Syntax Error: %v\n\n", err)
						helpMsg += "🔧 For SQLite, use: strftime('%Y',col) not YEAR(), col1||col2 not CONCAT()"
					} else {
						helpMsg = fmt.Sprintf("❌ SQL Error: %v\n\n🔧 Please fix and retry.", err)
					}
				} else if toolName == "python_executor" {
					helpMsg = fmt.Sprintf("❌ Python Error: %v\n\n🔧 Please fix the code and retry.", err)
				} else {
					helpMsg = fmt.Sprintf("❌ Tool Error: %v\n\n🔧 Please fix and retry.", err)
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
					Content: fmt.Sprintf("Error: %v\n\n🔴 Please fix the issue and try again.", err),
				})
			}
			return append(input, errorMsgs...), nil
		}
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TIMING] Tools Execution step took: %v", time.Since(startExec)))
		}

		// Emit progress for result processing
		emitProgress(StageAnalysis, 65, "progress.processing_results", 0, 0)

		// CRITICAL: Truncate tool output to prevent context overflow
		// Tool outputs (especially SQL results) can be huge
		const maxToolOutputChars = 50000 // Very high limit to prevent truncation of important data
		for i, msg := range toolResultMsg {
			if msg.Role == schema.Tool && len(msg.Content) > maxToolOutputChars {
				toolResultMsg[i] = &schema.Message{
					Role:       msg.Role,
					Content:    msg.Content[:maxToolOutputChars] + fmt.Sprintf("\n\n[... Output truncated - %d chars omitted for context limit]", len(msg.Content)-maxToolOutputChars),
					ToolCallID: msg.ToolCallID,
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

	// 5. Compile and Run with reduced max steps for better efficiency
	runnable, err := g.Compile(ctx, compose.WithMaxRunSteps(20))
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %v", err)
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Construction & Compilation took: %v", time.Since(startGraph)))
	}

	emitProgress(StageInitializing, 15, "progress.tools_ready", 0, 0)

	// 6. Build Context Prompt (minimal - only table names, let tool provide details)
	startContext := time.Now()
	var contextPrompt string
	var dbType string = "sqlite"
	if dataSourceID != "" && s.dsService != nil {
		sources, _ := s.dsService.LoadDataSources()
		for _, ds := range sources {
			if ds.ID == dataSourceID {
				// Determine database type
				if ds.Config.DBPath != "" {
					dbType = "sqlite"
				} else if ds.Type == "mysql" || ds.Type == "doris" {
					dbType = ds.Type
				}

				contextPrompt = fmt.Sprintf("\n\nData: %s (ID: %s, Type: %s)\n", ds.Name, ds.ID, strings.ToUpper(dbType))
				if ds.Analysis != nil && ds.Analysis.Summary != "" {
					contextPrompt += fmt.Sprintf("Summary: %s\n", ds.Analysis.Summary)
				}

				// Only send table names, not full schema
				if ds.Analysis != nil && len(ds.Analysis.Schema) > 0 {
					var tableNames []string
					for _, t := range ds.Analysis.Schema {
						tableNames = append(tableNames, t.TableName)
					}
					contextPrompt += fmt.Sprintf("Tables: %s\n", strings.Join(tableNames, ", "))
					contextPrompt += "⚠️ Call get_data_source_context for columns.\n"
				}

				// SQL dialect
				if dbType == "sqlite" {
					contextPrompt += `Dialect: SQLite (use strftime, ||, no YEAR/MONTH)`
				} else if dbType == "mysql" || dbType == "doris" {
					contextPrompt += `Dialect: MySQL (use YEAR/MONTH, CONCAT)`
				}
				break
			}
		}
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Context Prompt preparation took: %v", time.Since(startContext)))
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
					fmt.Sprintf("  • %s: %s", svc.Name, svc.Description))
			}
		}
		
		if len(availableServices) > 0 {
			mcpToolsPrompt = "\n\n🔌 MCP SERVICES (External capabilities):\n"
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

	sysMsg := &schema.Message{
		Role:    schema.System,
		Content: `VantageData数据分析专家。快速、直接、可视化优先。

🎯 目标: 高质量分析产出（图表+数据+洞察）

📊 **可视化方式（二选一）**:

**方式1: ECharts（推荐，无需执行代码）**
- 直接在回复中输出 ` + "```json:echarts\n{...}\n```" + `
- 前端会自动渲染图表
- 适合：交互式图表、快速展示
- 🚫 **ECharts绝对不会生成任何文件！** 不要说"已生成xxx.pdf"或"已保存xxx.png"
- ⚠️ **ECharts配置必须是纯JSON格式！** 不要使用JavaScript函数（如function(params){...}）。formatter请使用字符串模板（如"{b}: {c}"），不要用function。

**方式2: Python matplotlib（需要执行代码才能生成文件）**
- 必须调用python_executor工具执行代码
- 使用FILES_DIR变量保存文件
- 适合：需要导出PDF/PNG文件时
- ✅ 只有python_executor执行成功后，文件才真正存在

🚨🚨🚨 **严禁虚假文件声明（最重要规则）** 🚨🚨🚨
- **ECharts = 前端渲染 = 无文件生成** → 绝对不能说"图表已生成: xxx.pdf"
- **只有调用python_executor并执行成功后，才能声称文件已生成**
- **违规示例（绝对禁止）**:
  - ❌ "图表文件已生成: analysis.pdf (32KB)" ← 如果没调用python_executor，这是虚假声明
  - ❌ "✅ 散点图: scatter.pdf (28KB)" ← 如果只用了ECharts，这是虚假声明
- **正确示例**:
  - ✅ 使用ECharts时: "以下是交互式图表:" + json:echarts代码块（不提及任何文件）
  - ✅ 使用matplotlib时: 先调用python_executor，执行成功后才说"文件已保存"

⚡ 快速路径(跳过搜索,直接用python_executor):
- 时间/日期查询 → datetime.now().strftime("%Y年%m月%d日 %H:%M:%S")
- 数学计算 → 直接计算
- 单位换算 → 直接换算

🔧 **工具调用规范（严格遵守）**:

**工具依赖链（数据分析场景）:**
get_data_source_context → execute_sql → python_executor/ECharts → export_data

**规则:**
1. **先schema后SQL**: 必须先调用get_data_source_context获取列名和数据类型，再写SQL
2. **SQL结果传递**: execute_sql返回JSON数据，在python_executor中用json.loads()加载
3. **不要猜测列名**: 列名大小写敏感，必须从schema中获取准确名称
4. **一次获取足够schema**: 用table_names参数一次获取所有需要的表，避免多次调用
5. **工具错误处理**: SQL报错时根据错误信息修正后重试，不要放弃

📋 数据分析标准流程:
1. get_data_source_context → 获取schema（含列名、类型、样例数据、SQL方言提示）
2. execute_sql → 用正确的列名和语法查询数据
3. 可视化：ECharts(直接输出,无文件) 或 python_executor(生成文件)
4. 呈现结果(图表+洞察+数据表)

📤 数据导出规则:
- ⭐ 数据表格导出 → Excel格式(export_data, format="excel")
- 可视化报告 → PDF格式(需要python_executor)
- 演示文稿 → PPT格式

🔴 关键规则:
- **分析请求必须有可视化** - ECharts或matplotlib
- **ECharts不生成文件，不要声称生成了文件**
- 立即执行工具(不要先解释)
- get_data_source_context最多调用2次
- SQL错误时直接修复

🐍 **Python万能工具（当现有工具不够用时）**:
- 如果现有agent工具（execute_sql、web_search、export_data等）无法完成用户需求，**主动使用python_executor编写Python脚本来解决**
- Python可以做到几乎任何事情：数据处理、文件操作、API调用、文本分析、数学建模、格式转换等
- 示例场景：
  - 需要复杂数据转换/清洗 → 用pandas编写处理脚本
  - 需要调用外部API → 用requests库
  - 需要文本处理/正则匹配 → 用re/string操作
  - 需要统计建模/机器学习 → 用scipy/sklearn
  - 需要文件格式转换 → 用相应Python库
- **不要因为没有专门的工具就放弃任务，用Python编写解决方案！**

📊 输出格式:
- ECharts图表: ` + "```json:echarts\n{...}\n```" + ` (仅前端渲染，无文件，必须纯JSON，禁止function)
- 表格: ` + "```json:table\n[...]\n```" + `
- 图片会自动检测并显示

🌐 网络搜索(仅用于外部信息):
- web_search: 新闻、股价、天气等实时外部数据
- web_fetch: 获取网页内容
- ⚠️ 不要用搜索查时间/计算/本地可完成的任务
- 引用来源: [来源: URL]

🇨🇳 语言: 图表标题/标签必须用中文

📈 分析产出要求:
- 数据分析 → 必须包含: 图表(ECharts或matplotlib) + 关键洞察 + 数据摘要
- 简单问题(时间/计算) → 直接返回结果
- 不要只返回纯文字分析，要有可视化支撑

💡 **建议输出（重要）**:
- 每次数据分析完成后，在回复末尾添加"**建议**"或"**进一步分析建议**"小节
- 用编号列表(1. 2. 3.)列出3-5条后续分析建议
- 建议应具体、可操作，帮助用户深入探索数据

⚠️ 高效执行，但不要牺牲分析质量!` + analysisPlanPrompt + contextPrompt + workingContextPrompt + conversationContextPrompt + mcpToolsPrompt,
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

	emitProgress(StageAnalysis, 20, "progress.ai_processing", 0, 0)

	startInvoke := time.Now()
	finalHistory, err := runnable.Invoke(ctx, input)
	if err != nil {
		// Mark trajectory as failed
		trajectory.Success = false
		trajectory.ErrorMessage = err.Error()
		return nil, err
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Execution (Invoke) took: %v", time.Since(startInvoke)))
		s.Logger(fmt.Sprintf("[TIMING] Total RunAnalysis took: %v", time.Since(startTotal)))
	}

	emitProgress(StageComplete, 100, "progress.analysis_complete", 0, 0)

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
								s.Logger(fmt.Sprintf("[MEMORY] ✓ Stored [%s] %s: %s", 
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
	// Replace actual characters with their escaped representations for training visibility
	content = strings.ReplaceAll(content, "\n", "\\n")
	content = strings.ReplaceAll(content, "\r", "\\r")
	content = strings.ReplaceAll(content, "\t", "\\t")
	content = strings.ReplaceAll(content, "\"", "\\\"")
	content = strings.ReplaceAll(content, "\\", "\\\\")
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
