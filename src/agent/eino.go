package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	
	"rapidbi/config"
)

// EinoService manages Eino-based agents
type EinoService struct {
	ChatModel model.ChatModel
	dsService *DataSourceService
	cfg       config.Config
}

// NewEinoService creates a new EinoService
func NewEinoService(cfg config.Config, dsService *DataSourceService) (*EinoService, error) {
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

	return &EinoService{
		ChatModel: chatModel,
		dsService: dsService,
		cfg:       cfg,
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
func (s *EinoService) RunAnalysis(ctx context.Context, history []*schema.Message) (*schema.Message, error) {
	// 1. Initialize Tools
	pyTool := NewPythonExecutorTool(s.cfg)
	dsTool := NewDataSourceContextTool(s.dsService)
	tools := []tool.BaseTool{pyTool, dsTool}

	// 2. Create ToolsNode
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: tools,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tools node: %v", err)
	}

	// 3. Bind Tool Infos to Model
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

	// 4. Build Graph
	g := compose.NewGraph[[]*schema.Message, *schema.Message]()

	err = g.AddChatModelNode("model", s.ChatModel)
	if err != nil {
		return nil, err
	}
	err = g.AddToolsNode("tools", toolsNode)
	if err != nil {
		return nil, err
	}

	err = g.AddEdge(compose.START, "model")
	if err != nil {
		return nil, err
	}

	// Branch: loop back to tools or end
	err = g.AddBranch("model", compose.NewGraphBranch(func(ctx context.Context, msg *schema.Message) (string, error) {
		if len(msg.ToolCalls) > 0 {
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

	// 5. Compile and Run
	runnable, err := g.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %v", err)
	}

	sysMsg := &schema.Message{
		Role:    schema.System,
		Content: "You are RapidBI's advanced data analysis agent. Help the user explore their data. Use tools to access data schema and samples, and execute Python code for analysis. \n\nGuidelines:\n1. Format tables as Markdown tables.\n2. When performing visualization, always save the plot as 'chart.png' in the current directory using `plt.savefig('chart.png')`.\n3. Provide concise natural language summaries alongside your data and charts.",
	}
	input := append([]*schema.Message{sysMsg}, history...)

	return runnable.Invoke(ctx, input)
}
