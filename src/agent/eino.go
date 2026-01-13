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

// EinoService manages Eino-based agents
type EinoService struct {
	ChatModel       model.ChatModel
	dsService       *DataSourceService
	cfg             config.Config
	Logger          func(string)
	memoryManager   *MemoryManager
	pythonPool      *PythonPool
	errorKnowledge  *ErrorKnowledge
	skillManager    *templates.SkillManager
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
func NewEinoService(cfg config.Config, dsService *DataSourceService, logger func(string)) (*EinoService, error) {
	var chatModel model.ChatModel
	var err error

	switch cfg.LLMProvider {
	case "Anthropic":
		chatModel, err = NewAnthropicChatModel(context.Background(), &AnthropicConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.ModelName,
		})
	default:
		// Default to OpenAI (includes "OpenAI", "OpenAI-Compatible", "Claude-Compatible" if using OAI format)
		// Note: "Claude-Compatible" in this project usually means "Use OpenAI client but point to Claude proxy"
		// or "Use Anthropic client". 
		// If LLMService treats Claude-Compatible as Anthropic-format, we should use AnthropicChatModel.
		// Checking llm_service.go: Claude-Compatible uses /v1/messages. So it is Anthropic format.
		if cfg.LLMProvider == "Claude-Compatible" {
			chatModel, err = NewAnthropicChatModel(context.Background(), &AnthropicConfig{
				APIKey:  cfg.APIKey,
				BaseURL: cfg.BaseURL,
				Model:   cfg.ModelName,
			})
		} else {
			chatModel, err = openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
				APIKey:  cfg.APIKey,
				BaseURL: cfg.BaseURL, // Might need adjustment if empty
				Model:   cfg.ModelName,
				Timeout: 0, // Default
			})
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create eino chat model: %v", err)
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
		ChatModel:      chatModel,
		dsService:      dsService,
		cfg:            cfg,
		Logger:         logger,
		memoryManager:  memManager,
		pythonPool:     pyPool,
		errorKnowledge: errorKnowledge,
		skillManager:   skillManager,
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

// RunAnalysis executes the agent with full history and tool support
func (s *EinoService) RunAnalysis(ctx context.Context, history []*schema.Message, dataSourceID, threadID string) (*schema.Message, error) {
	return s.RunAnalysisWithProgress(ctx, history, dataSourceID, threadID, "", nil, nil, nil)
}

// RunAnalysisWithProgress executes the agent with progress callbacks
func (s *EinoService) RunAnalysisWithProgress(ctx context.Context, history []*schema.Message, dataSourceID, threadID, sessionDir string, onProgress ProgressCallback, onFileSaved func(fileName, fileType string, fileSize int64), cancelCheck func() bool) (*schema.Message, error) {
	startTotal := time.Now()
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Start RunAnalysis for thread: %s", threadID))
	}

	// Initialize trajectory tracking for training
	trajectory := &AgentTrajectory{
		ThreadID:     threadID,
		DataSourceID: dataSourceID,
		StartTime:    time.Now().UnixMilli(),
		Steps:        []TrajectoryStep{},
		Success:      false,
	}

	// Extract user request from last message
	if len(history) > 0 {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Role == schema.User {
				trajectory.UserRequest = history[i].Content
				break
			}
		}
	}

	// Save trajectory on completion (success or error)
	defer func() {
		if sessionDir != "" {
			s.saveTrajectory(sessionDir, trajectory)
		}
	}()

	// Helper to emit progress
	emitProgress := func(stage string, progress int, message string, step, total int) {
		if onProgress != nil {
			onProgress(NewProgressUpdate(stage, progress, message, step, total))
		}
	}

	emitProgress(StageInitializing, 5, "Initializing analysis tools...", 1, 6)

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
					emitProgress(StageComplete, 100, "Analysis complete", 6, 6)
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

	// Create SQL planner for self-correction capability
	sqlPlanner := NewSQLPlanner(s.ChatModel, s.dsService, s.Logger)
	sqlTool := NewSQLExecutorToolWithPlanner(s.dsService, sqlPlanner, s.Logger)
	// Inject error knowledge into SQL tool
	sqlTool.SetErrorKnowledge(s.errorKnowledge)

	// Remove PythonPlanner to reduce overhead - LLM can generate Python directly
	// pythonPlanner := NewPythonPlanner(s.ChatModel, s.Logger)
	// pythonPlannerTool := NewPythonPlannerTool(pythonPlanner)

	tools := []tool.BaseTool{pyTool, dsTool, sqlTool}

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

	emitProgress(StageInitializing, 10, "Tools ready, building analysis graph...", 1, 6)

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
		emitProgress(StageAnalysis, progress, fmt.Sprintf("AI processing (step %d)...", iterationCount), 3, 6)

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

		// Record successful model call in trajectory
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
		// Append response to history (use original input to preserve full context for tools)
		return append(input, resp), nil
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
				emitProgress(StageSchema, 25, "Loading database schema...", 2, 6)
				msg = "Fetching schema"
			case "execute_sql":
				emitProgress(StageQuery, 40, "Executing SQL query...", 4, 6)
				msg = "Executing query"
			case "python_executor":
				emitProgress(StageAnalysis, 60, "Running Python analysis...", 5, 6)
				msg = "Analyzing data"
			default:
				msg = fmt.Sprintf("Running %s", toolName)
			}
			if s.Logger != nil {
				s.Logger(fmt.Sprintf("[PROGRESS] %s", msg))
			}
		}

		// Execute tools
		toolResultMsg, err := toolsNode.Invoke(ctx, lastMsg)

		// Record tool calls in trajectory
		for _, tc := range lastMsg.ToolCalls {
			step := TrajectoryStep{
				StepNumber: len(trajectory.Steps) + 1,
				Timestamp:  time.Now().UnixMilli(),
				Type:       "tool_call",
				ToolName:   tc.Function.Name,
				ToolInput:  tc.Function.Arguments,
				ToolCallID: tc.ID,
			}

			if err != nil {
				step.Error = err.Error()
			} else if len(toolResultMsg) > 0 {
				// Find matching tool result for this call
				for _, resultMsg := range toolResultMsg {
					if resultMsg.ToolCallID == tc.ID {
						step.ToolOutput = truncateString(resultMsg.Content, 1000)
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
		const maxToolOutputChars = 2000
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
						Message:    fmt.Sprintf("Tool %s completed", toolName),
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

	sysMsg := &schema.Message{
		Role:    schema.System,
		Content: `You are RapidBI's data analysis expert. Be FAST and DIRECT.

🎯 GOAL: Complete in ≤10 tool calls total.

📋 WORKFLOW (EXECUTE IMMEDIATELY):
1. get_data_source_context → Get columns
2. execute_sql → Query data
3. python_executor (if needed) → Analyze/visualize
4. STOP → Present results

🔴 CRITICAL:
- EXECUTE tools immediately (not explain first)
- ONE SQL query if possible (use JOINs, not multiple queries)
- Present results IMMEDIATELY after data ready
- NO explanatory text before tool calls

📊 OUTPUT: Use ` + "```json:echarts{...}```" + ` or ` + "```json:table[...]```" + `

⚠️ You have LIMITED steps - be efficient!` + contextPrompt,
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

	emitProgress(StageAnalysis, 20, "Starting analysis...", 3, 6)

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

	emitProgress(StageComplete, 100, "Analysis complete", 6, 6)

	// Return the last message and mark trajectory as successful
	if len(finalHistory) > 0 {
		lastMsg := finalHistory[len(finalHistory)-1]
		trajectory.Success = true
		trajectory.FinalResponse = truncateString(lastMsg.Content, 2000)
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

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(trajectory, "", "  ")
	if err != nil {
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TRAJECTORY] Failed to marshal: %v", err))
		}
		return
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TRAJECTORY] Failed to write file: %v", err))
		}
		return
	}

	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TRAJECTORY] Saved to: %s (%d steps, %d tool calls, %dms)",
			filePath, len(trajectory.Steps), trajectory.ToolCallCount, trajectory.TotalDuration))
	}
}

// messagesToMap converts messages to simplified map representation for trajectory
func messagesToMap(msgs []*schema.Message) []map[string]interface{} {
	var result []map[string]interface{}
	for _, msg := range msgs {
		result = append(result, messageToMap(msg))
	}
	return result
}

// messageToMap converts a single message to map with truncated content
func messageToMap(msg *schema.Message) map[string]interface{} {
	m := map[string]interface{}{
		"role": string(msg.Role),
	}

	// Truncate long content to avoid huge trajectory files
	if len(msg.Content) > 500 {
		m["content"] = msg.Content[:500] + "... [truncated]"
	} else {
		m["content"] = msg.Content
	}

	// Add tool calls if present
	if len(msg.ToolCalls) > 0 {
		var calls []string
		for _, tc := range msg.ToolCalls {
			calls = append(calls, tc.Function.Name)
		}
		m["tool_calls"] = calls
	}

	return m
}
