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
	// Initialize OpenAI Chat Model
	// Note: Eino's OpenAI component configuration might differ slightly from our internal config
	// We need to map our config to Eino's expectations.
	
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL, // Might need adjustment if empty
		Model:   cfg.ModelName,
		Timeout: 0, // Default
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create eino openai model: %v", err)
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
