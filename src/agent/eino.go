package agent

import (
	"context"
	"fmt"
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
	ChatModel     model.ChatModel
	dsService     *DataSourceService
	cfg           config.Config
	Logger        func(string)
	memoryManager *MemoryManager
	pythonPool    *PythonPool
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

	return &EinoService{
		ChatModel:     chatModel,
		dsService:     dsService,
		cfg:           cfg,
		Logger:        logger,
		memoryManager: memManager,
		pythonPool:    pyPool,
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

// RunAnalysis executes the agent with full history and tool support
func (s *EinoService) RunAnalysis(ctx context.Context, history []*schema.Message, dataSourceID, threadID string) (*schema.Message, error) {
	return s.RunAnalysisWithProgress(ctx, history, dataSourceID, threadID, nil)
}

// RunAnalysisWithProgress executes the agent with progress callbacks
func (s *EinoService) RunAnalysisWithProgress(ctx context.Context, history []*schema.Message, dataSourceID, threadID string, onProgress ProgressCallback) (*schema.Message, error) {
	startTotal := time.Now()
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Start RunAnalysis for thread: %s", threadID))
	}

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
	dsTool := NewDataSourceContextTool(s.dsService)
	sqlTool := NewSQLExecutorTool(s.dsService)
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

	// Define Model Node Wrapper
	modelLambda := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		iterationCount++
		startModel := time.Now()

		// Emit progress based on iteration
		progress := 20 + min(iterationCount*10, 60) // 20-80%
		emitProgress(StageAnalysis, progress, fmt.Sprintf("AI processing (step %d)...", iterationCount), 3, 6)

		// Call model with full history
		resp, err := s.ChatModel.Generate(ctx, input)
		if err != nil {
			return nil, err
		}
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TIMING] Model Generation step took: %v", time.Since(startModel)))
		}
		// Append response to history
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
				emitProgress(StageQuery, 40, "Executing SQL query...", 3, 6)
				msg = "Executing query"
			case "python_executor":
				emitProgress(StageAnalysis, 60, "Running Python analysis...", 4, 6)
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
		if err != nil {
			return nil, err
		}
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TIMING] Tools Execution step took: %v", time.Since(startExec)))
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
					if len(preview) > 500 {
						preview = preview[:500] + "... (more)"
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

	// 5. Compile and Run with increased max steps
	runnable, err := g.Compile(ctx, compose.WithMaxRunSteps(50))
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %v", err)
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Construction & Compilation took: %v", time.Since(startGraph)))
	}

	emitProgress(StageInitializing, 15, "Preparing context...", 1, 6)

	// 6. Build Context Prompt (with length control)
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

				contextPrompt = fmt.Sprintf("\n\nCurrent Data Source Context (ID: %s, Name: %s, Type: %s):\n", ds.ID, ds.Name, dbType)
				if ds.Analysis != nil {
					contextPrompt += fmt.Sprintf("Summary: %s\n", ds.Analysis.Summary)
					contextPrompt += "Schema Overview:\n"
					for _, t := range ds.Analysis.Schema {
						contextPrompt += fmt.Sprintf("- Table: %s, Columns: %v\n", t.TableName, t.Columns)
					}
				}

				// Add SQL dialect hints based on database type
				contextPrompt += fmt.Sprintf("\n⚠️ SQL Dialect: %s\n", strings.ToUpper(dbType))
				if dbType == "sqlite" {
					contextPrompt += `IMPORTANT - SQLite Syntax Rules:
• Date: strftime('%Y', col), strftime('%m', col), strftime('%d', col)
• Date format: strftime('%Y-%m', col) for YYYY-MM
• Concat: col1 || ' ' || col2 (NOT CONCAT())
• INSTR(str, substr) - only 2 params!
• COALESCE() instead of IFNULL()
• SUBSTR(str, start, len)
• Current: date('now'), datetime('now')
• NO YEAR(), MONTH(), DAY() functions!
`
				} else if dbType == "mysql" || dbType == "doris" {
					contextPrompt += `IMPORTANT - MySQL/Doris Syntax Rules:
• Date: YEAR(col), MONTH(col), DAY(col)
• Date format: DATE_FORMAT(col, '%Y-%m')
• Concat: CONCAT(col1, ' ', col2)
• IFNULL(a, b) or COALESCE(a, b)
• SUBSTRING(str, start, len)
• Current: NOW(), CURDATE()
• GROUP_CONCAT() for string aggregation
`
				}

				// Truncate context if too long (reserve ~2000 tokens = 8000 chars)
				maxContextChars := 8000
				if len(contextPrompt) > maxContextChars {
					contextPrompt = s.memoryManager.TruncateDataContext(contextPrompt, maxContextChars)
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
		Content: "You are RapidBI's data analysis agent. Use tools to access schema and execute queries.\n\nTools:\n1. get_data_source_context - Get schema + 3 sample rows\n2. execute_sql - Execute SELECT queries (max 1000 rows)\n3. python_executor - Execute Python (max 80 lines)\n\n⚠️ CRITICAL: Python limited to 80 lines/call. Code >80 lines WILL FAIL.\n\n🔴 MANDATORY: ALWAYS call get_data_source_context FIRST before any execute_sql.\nNEVER assume column names exist. If you get \"no such column\" error, you MUST:\n1. Call get_data_source_context to see actual schema\n2. Rewrite query using ONLY the columns shown\n3. Do NOT retry with the same wrong column names\n\n🔴 FOR RFM/CLUSTERING - DO EXACTLY THIS:\nStep 1: get_data_source_context\nStep 2: execute_sql to get data\nStep 3: python_executor with ONLY this code (copy exactly):\n```python\nimport json\nimport pandas as pd\ndata = json.loads('''PASTE_SQL_RESULT_HERE''')\ndf = pd.DataFrame(data)\nref_date = df['OrderDate'].max()\nrfm = df.groupby('CustomerID').agg({\n    'OrderDate': lambda x: (ref_date - x.max()).days,\n    'OrderID': 'count',\n    'TotalAmount': 'sum'\n}).rename(columns={'OrderDate':'R','OrderID':'F','TotalAmount':'M'})\nprint(rfm.describe())\nprint(f'\\nTotal: {len(rfm)} customers')\n```\nSTOP. Wait for result.\n\nStep 4 (segmentation) - Use duplicates='drop' for qcut:\n```python\nrfm['R_Score'] = pd.qcut(rfm['R'], q=5, labels=[5,4,3,2,1], duplicates='drop')\nrfm['F_Score'] = pd.qcut(rfm['F'], q=5, labels=[1,2,3,4,5], duplicates='drop')\nrfm['M_Score'] = pd.qcut(rfm['M'], q=5, labels=[1,2,3,4,5], duplicates='drop')\n```\nStep 5: Visualize (50 lines)\nStep 6: Summary (30 lines)\n\nWorkflow:\n1. get_data_source_context for schema\n2. execute_sql to query\n3. python_executor - load data:\n   ```python\n   import json, pandas as pd\n   data = json.loads('''<SQL>''')\n   df = pd.DataFrame(data)\n   ```\n\nViz: plt.savefig('chart.png') or ```json:echarts {...}```\n\nRules:\n- Max 80 lines/call\n- ONE step at a time for RFM\n- Use duplicates='drop' with qcut\n- Short names" + contextPrompt,
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
		return nil, err
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Execution (Invoke) took: %v", time.Since(startInvoke)))
		s.Logger(fmt.Sprintf("[TIMING] Total RunAnalysis took: %v", time.Since(startTotal)))
	}

	emitProgress(StageComplete, 100, "Analysis complete", 6, 6)

	// Return the last message
	if len(finalHistory) > 0 {
		return finalHistory[len(finalHistory)-1], nil
	}
	return nil, fmt.Errorf("agent returned empty history")
}
