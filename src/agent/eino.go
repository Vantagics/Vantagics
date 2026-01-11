package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	
	"rapidbi/config"
)

// EinoService manages Eino-based agents
type EinoService struct {
	ChatModel     model.ChatModel
	dsService     *DataSourceService
	cfg           config.Config
	Logger        func(string)
	memoryManager *MemoryManager
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

	return &EinoService{
		ChatModel:     chatModel,
		dsService:     dsService,
		cfg:           cfg,
		Logger:        logger,
		memoryManager: memManager,
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

// RunAnalysis executes the agent with full history and tool support
func (s *EinoService) RunAnalysis(ctx context.Context, history []*schema.Message, dataSourceID, threadID string) (*schema.Message, error) {
	startTotal := time.Now()
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Start RunAnalysis for thread: %s", threadID))
	}

	// 1. Initialize Tools
	startTools := time.Now()
	pyTool := NewPythonExecutorTool(s.cfg)
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

	// 4. Build Graph using Lambda nodes to manage state ([]*schema.Message)
	startGraph := time.Now()
	g := compose.NewGraph[[]*schema.Message, []*schema.Message]()

	// Define Model Node Wrapper
	modelLambda := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		startModel := time.Now()
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

		// Execute tools
		toolResultMsg, err := toolsNode.Invoke(ctx, lastMsg)
		if err != nil {
			return nil, err
		}
		if s.Logger != nil {
			s.Logger(fmt.Sprintf("[TIMING] Tools Execution step took: %v", time.Since(startExec)))
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

				// Add SQL dialect hints
				if dbType == "sqlite" {
					contextPrompt += "\nSQL Dialect: SQLite\n"
					contextPrompt += "Date/Time Functions:\n"
					contextPrompt += "- Extract year: strftime('%Y', date_column)\n"
					contextPrompt += "- Extract month: strftime('%m', date_column)\n"
					contextPrompt += "- Extract day: strftime('%d', date_column)\n"
					contextPrompt += "- Format: strftime('%Y-%m', date_column) for YYYY-MM\n"
					contextPrompt += "- IMPORTANT: Use strftime() for date operations, don't parse dates manually with SUBSTR/INSTR\n"
					contextPrompt += "- String functions: INSTR(str, substr) only takes 2 parameters (not 3)\n"
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
		Content: "You are RapidBI's data analysis agent. Use tools to access schema and execute queries.\n\nTools:\n1. get_data_source_context - Get schema + 3 sample rows\n2. execute_sql - Execute SELECT queries (max 1000 rows)\n3. python_executor - Execute Python (max 80 lines)\n\n⚠️ CRITICAL: Python limited to 80 lines/call. Code >80 lines WILL FAIL.\n\nRFM/Clustering Pattern (MANDATORY):\n1. execute_sql: Get customer data\n2. python_executor: Load + calc scores (60 lines)\n3. python_executor: Segment customers (50 lines)\n4. python_executor: Visualize (50 lines)\n5. python_executor: Summary table (30 lines)\nNEVER do RFM in 1 call.\n\nWorkflow:\n1. get_data_source_context for schema\n2. execute_sql to query data\n3. python_executor - ALWAYS load data:\n   ```python\n   import json, pandas as pd\n   data = json.loads('''<SQL_RESULT>''')\n   df = pd.DataFrame(data)\n   ```\n4. NEVER assume df exists\n\nVisualization:\n- Save plots: plt.savefig('chart.png')\n- For interactive: ```json:echarts {...}```\n\nRules:\n- Max 80 lines per python call\n- Break >80 into multiple calls\n- Each call: import libs, load data, ONE task\n- Use short names (r_score not recency_score)\n- Simple analyses: 1-2 calls. Complex: 4-5 calls min" + contextPrompt,
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

	startInvoke := time.Now()
	finalHistory, err := runnable.Invoke(ctx, input)
	if err != nil {
		return nil, err
	}
	if s.Logger != nil {
		s.Logger(fmt.Sprintf("[TIMING] Graph Execution (Invoke) took: %v", time.Since(startInvoke)))
		s.Logger(fmt.Sprintf("[TIMING] Total RunAnalysis took: %v", time.Since(startTotal)))
	}

	// Return the last message
	if len(finalHistory) > 0 {
		return finalHistory[len(finalHistory)-1], nil
	}
	return nil, fmt.Errorf("agent returned empty history")
}
