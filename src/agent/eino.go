package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"rapidbi/agent/templates"
	"rapidbi/config"
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

// EinoService manages Eino-based agents
type EinoService struct {
	ChatModel             model.ChatModel
	dsService             *DataSourceService
	cfg                   config.Config
	Logger                func(string)
	memoryManager         *MemoryManager
	workingContextManager *WorkingContextManager
	pythonPool            *PythonPool
	errorKnowledge        *ErrorKnowledge
	skillManager          *templates.SkillManager
	memoryService         *MemoryService // For persistent memory storage
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
			
			chatModel, err = openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
				APIKey:    cfg.APIKey,
				BaseURL:   cfg.BaseURL, // Might need adjustment if empty
				Model:     cfg.ModelName,
				MaxTokens: &maxTokens, // Use pointer to int
				Timeout:   0, // Default
			})
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
	skillsDir := filepath.Join(dsService.dataCacheDir, "..", "skills") // Skills in RapidBI/skills
	skillManager := templates.NewSkillManager(skillsDir, logger)
	if err := skillManager.LoadSkills(); err != nil {
		if logger != nil {
			logger(fmt.Sprintf("[WARNING] Failed to load skills: %v", err))
		}
	}

	return &EinoService{
		ChatModel:             chatModel,
		dsService:             dsService,
		cfg:                   cfg,
		Logger:                logger,
		memoryManager:         memManager,
		workingContextManager: workingContextManager,
		pythonPool:            pyPool,
		errorKnowledge:        errorKnowledge,
		skillManager:          skillManager,
		memoryService:         memoryService,
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

	// Configure memory manager with memory service for this thread
	if s.memoryManager != nil && s.memoryService != nil && threadID != "" {
		s.memoryManager.SetMemoryService(s.memoryService, threadID)
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
	if len(history) > 0 {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Role == schema.User {
				trajectory.UserRequest = escapeForTraining(history[i].Content)
				break
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

	emitProgress(StageInitializing, 5, "progress.initializing_tools", 1, 6)

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
						tables, err := s.dsService.GetDataSourceTables(dsID)
						if err != nil {
							return nil, err
						}
						var result []templates.TableInfo
						for _, tableName := range tables {
							cols, _ := s.dsService.GetDataSourceTableColumns(dsID, tableName)
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
					emitProgress(StageComplete, 100, "progress.analysis_complete", 6, 6)
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

	// 1. Initialize Tools
	startTools := time.Now()
	
	var pyTool *PythonExecutorTool
	if s.pythonPool != nil {
		pyTool = NewPythonExecutorToolWithPool(s.cfg, s.pythonPool)
	} else {
		pyTool = NewPythonExecutorTool(s.cfg)
	}
	// Inject error knowledge into Python tool
	pyTool.SetErrorKnowledge(s.errorKnowledge)
	// Inject execution recorder into Python tool
	if executionRecorder != nil {
		pyTool.SetExecutionRecorder(executionRecorder)
	}
	// Set session directory for file storage
	if sessionDir != "" {
		pyTool.SetSessionDirectory(sessionDir)
		if onFileSaved != nil {
			pyTool.SetFileSavedCallback(onFileSaved)
		}
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[SESSION] Files will be saved to: %s", sessionDir))
		}
	}

	dsTool := NewDataSourceContextTool(s.dsService)
	// Inject working context manager for schema caching
	if s.workingContextManager != nil {
		dsTool.SetWorkingContextManager(s.workingContextManager)
	}
	// Inject SQL collector into datasource tool for schema tracking
	if sqlCollector != nil {
		dsTool.SetSQLCollector(sqlCollector)
	}

	// Create SQL planner for self-correction capability
	sqlPlanner := NewSQLPlanner(s.ChatModel, s.dsService, s.Logger)
	sqlTool := NewSQLExecutorToolWithPlanner(s.dsService, sqlPlanner, s.Logger)
	// Inject error knowledge into SQL tool
	sqlTool.SetErrorKnowledge(s.errorKnowledge)
	// Inject execution recorder into SQL tool
	if executionRecorder != nil {
		sqlTool.SetExecutionRecorder(executionRecorder)
	}
	// Inject SQL collector into SQL tool
	if sqlCollector != nil {
		sqlTool.SetSQLCollector(sqlCollector)
		// Set current user request for context
		if len(history) > 0 {
			for i := len(history) - 1; i >= 0; i-- {
				if history[i].Role == schema.User {
					sqlCollector.SetUserRequest(history[i].Content)
					break
				}
			}
		}
	}

	// Remove PythonPlanner to reduce overhead - LLM can generate Python directly
	// pythonPlanner := NewPythonPlanner(s.ChatModel, s.Logger)
	// pythonPlannerTool := NewPythonPlannerTool(pythonPlanner)

	// Initialize tools list
	tools := []tool.BaseTool{pyTool, dsTool, sqlTool}

	// Add Web Search and Fetch tools with configured search engine and proxy
	activeEngine := s.cfg.GetActiveSearchEngine()
	webSearchTool := NewWebSearchTool(s.Logger, activeEngine, s.cfg.ProxyConfig)
	webFetchTool := NewWebFetchTool(s.Logger, s.cfg.ProxyConfig)
	tools = append(tools, webSearchTool, webFetchTool)
	if s.Logger != nil {
		engineName := "default"
		if activeEngine != nil {
			engineName = activeEngine.Name
		}
		s.Logger(fmt.Sprintf("[WEB-TOOLS] Web search (engine: %s) and fetch tools loaded", engineName))
	}

	// Add MCP tool if services are configured
	mcpTool := NewMCPTool(s.cfg.MCPServices, s.Logger)
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

	emitProgress(StageInitializing, 10, "progress.tools_ready", 1, 6)

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
		if iterationCount == 10 {
			warningMsg := &schema.Message{
				Role:    schema.User,
				Content: "⚡ 10 steps used. Finish QUICKLY - use 1-2 more tools MAX.",
			}
			input = append(input, warningMsg)
			if s.Logger != nil {
				s.Logger("[WARNING] Step 10 warning injected")
			}
		} else if iterationCount == 15 {
			finalMsg := &schema.Message{
				Role:    schema.User,
				Content: "🛑 STOP NOW. Present what you have.",
			}
			input = append(input, finalMsg)
			if s.Logger != nil {
				s.Logger("[FINAL-WARNING] Step 15 final warning injected")
			}
		}

		// Emit progress based on iteration
		progress := 20 + min(iterationCount*10, 60) // 20-80%
		emitProgress(StageAnalysis, progress, "progress.ai_processing", 3, 6)

		// CRITICAL: Apply memory management before each model call to prevent context overflow
		managedInput, err := s.memoryManager.ManageMemory(ctx, input)
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
		// Get the last message (which should be Assistant with ToolCalls)
		if len(input) == 0 {
			return nil, fmt.Errorf("tool node received empty history")
		}
		lastMsg := input[len(input)-1]

		// Emit progress based on tool being called
		if len(lastMsg.ToolCalls) > 0 {
			toolName := lastMsg.ToolCalls[0].Function.Name
			var msg string
			switch toolName {
			case "get_data_source_context":
				emitProgress(StageSchema, 25, "progress.loading_schema", 2, 6)
				msg = "获取模式中"
			case "execute_sql":
				emitProgress(StageQuery, 40, "progress.executing_sql", 4, 6)
				msg = "执行查询中"
			case "python_executor":
				emitProgress(StageAnalysis, 60, "progress.running_python", 5, 6)
				msg = "分析数据中"
			default:
				msg = fmt.Sprintf("Running %s", toolName)
			}
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[PROGRESS] %s", msg))
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
	runnable, err := g.Compile(ctx, compose.WithMaxRunSteps(30))
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %v", err)
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Construction & Compilation took: %v", time.Since(startGraph)))
	}

	emitProgress(StageInitializing, 15, "Preparing context...", 1, 6)

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

	sysMsg := &schema.Message{
		Role:    schema.System,
		Content: `You are RapidBI's data analysis expert. Be FAST and DIRECT.

🎯 GOAL: Complete in ≤10 tool calls total.

🚫 CRITICAL KNOWLEDGE RESTRICTION:
- You MUST NOT use your pre-trained knowledge or internal data
- ALL information MUST come from tools: database queries, web searches, or MCP services
- For ANY factual question (market data, company info, statistics, etc.):
  1. If it's about user's data → Use get_data_source_context + execute_sql
  2. If it's external information → Use web_search + web_fetch
  3. If it's from MCP services → Use available MCP tools
- NEVER answer from memory - always verify with tools
- If you cannot get data from tools, say "I cannot find this information in the available data sources"

📋 SMART WORKFLOW:
1. get_data_source_context → Get schema for ALL relevant tables in ONE call
   ⚠️ CRITICAL: Use table_names parameter to get multiple tables at once
   Example: {"data_source_id": "xxx", "table_names": ["orders", "customers", "products"]}
2. execute_sql → Query data (ONE query with JOINs preferred)
3. python_executor (ONLY if visualization/complex analysis needed)
4. STOP → Present results immediately

🔴 CRITICAL RULES:
- EXECUTE tools immediately (NO explanations before tool calls)
- Get schema for ALL tables you need in ONE get_data_source_context call
- DON'T call get_data_source_context multiple times - it's SLOW
- ONE SQL query if possible (use JOINs, subqueries, CTEs)
- Present results IMMEDIATELY after data ready
- NO unnecessary tool calls

⚡ EFFICIENCY TIPS:
- First call: Get table list only (no table_names parameter)
- Second call: Get schema for ALL relevant tables at once (with table_names)
- NEVER call get_data_source_context more than twice
- If SQL error mentions column → Fix it directly, don't re-fetch schema
- Combine multiple questions into ONE SQL query when possible
- Skip python_executor for simple queries (just show table)

📊 OUTPUT FORMAT:
- For charts: ` + "```json:echarts\n{...}\n```" + `
- For tables: ` + "```json:table\n[...]\n```" + `
- IMPORTANT: Add newline after json:echarts or json:table

🌐 WEB SEARCH TOOLS (Use sparingly - SLOW operation):
- web_search: Search the web for current information, market data, competitor analysis
  ⚠️ WARNING: Takes 60-90 seconds! Use ONLY when external data is essential
  Example: "latest smartphone market share 2026", "Tesla vs BYD sales comparison"
  Returns: JSON array with title, url, snippet for each result
- web_fetch: Fetch and parse web page content (use URLs from search results)
  Returns structured data: title, content, tables, links
- ONLY use when database data is insufficient
- Prefer internal data analysis over web searches

📌 CRITICAL - CITING WEB SOURCES:
When using information from web_search or web_fetch results:
1. ALWAYS include the source URL in your response
2. Format citations as: [Source: URL] or use markdown links [text](URL)
3. Place citations immediately after the information
4. Example: "特斯拉2025年销量为180万辆 [来源: https://example.com/tesla-sales]"
5. For multiple sources, cite each one separately
6. This ensures transparency and allows users to verify information

🇨🇳 LANGUAGE REQUIREMENTS:
- ALL chart titles, axis labels, and legends MUST be in Chinese
- Use descriptive Chinese names for all visualizations (e.g., "销售趋势图", not "Sales Trend")
- ALL Python code comments and print statements should be in Chinese where user-facing

⚠️ You have LIMITED steps - be efficient!` + contextPrompt + workingContextPrompt + mcpToolsPrompt,
	}

	// 7. Apply memory management to history
	startMemory := time.Now()
	managedHistory, err := s.memoryManager.ManageMemory(ctx, history)
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

	input := append([]*schema.Message{sysMsg}, managedHistory...)

	emitProgress(StageAnalysis, 20, "开始分析...", 3, 6)

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

	emitProgress(StageComplete, 100, "分析完成", 6, 6)

	// Return the last message and mark trajectory as successful with escaped content
	if len(finalHistory) > 0 {
		lastMsg := finalHistory[len(finalHistory)-1]
		trajectory.Success = true
		trajectory.FinalResponse = escapeForTraining(lastMsg.Content) // Escape for training visibility
		
		// Extract and store valuable memories (only if analysis was successful)
		// Run asynchronously to not block user response
		if lastMsg.Role == schema.Assistant && lastMsg.Content != "" {
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
