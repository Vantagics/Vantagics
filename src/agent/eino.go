package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	
	"rapidbi/config"
)

// EinoService manages Eino-based agents
type EinoService struct {
	ChatModel model.ChatModel
}

// NewEinoService creates a new EinoService
func NewEinoService(cfg config.Config) (*EinoService, error) {
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

// RunAnalysis executes the agent with full history
func (s *EinoService) RunAnalysis(ctx context.Context, history []*schema.Message) (*schema.Message, error) {
	// Create a chain: Model only (since we preprocess history)
	// Ideally we would have a PromptTemplate node here if we were using templates,
	// but since we have raw messages, we can just pass them.
	chain := compose.NewChain[[]*schema.Message, *schema.Message]()

	// Call Chat Model
	chain.AppendChatModel(s.ChatModel)

	// Compile
	runnable, err := chain.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %v", err)
	}

	// Preprocess: Inject System Prompt
	sysMsg := &schema.Message{
		Role:    schema.System,
		Content: "You are RapidBI's advanced data analysis agent. Help the user explore their data. Use tools if available.",
	}
	input := append([]*schema.Message{sysMsg}, history...)

	// Invoke
	return runnable.Invoke(ctx, input)
}
