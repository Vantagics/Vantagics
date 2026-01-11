package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// MemoryTier defines different memory tiers for context management
type MemoryTier int

const (
	ShortTermMemory  MemoryTier = iota // Recent messages, kept as-is
	MidTermMemory                       // Older messages, compressed/summarized
	LongTermMemory                      // Overall session summary
)

// MemoryConfig holds configuration for memory management
type MemoryConfig struct {
	MaxTokens           int // Maximum tokens from config
	ShortTermMessages   int // Number of recent messages to keep in full
	TokenReservePercent int // Percentage of tokens to reserve for response (default 20%)
	EstimatedTokenRatio int // Estimated chars per token (default 4 for English)
}

// MemoryManager manages hierarchical memory for chat sessions
type MemoryManager struct {
	config    MemoryConfig
	chatModel model.ChatModel
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(maxTokens int, chatModel model.ChatModel) *MemoryManager {
	// Cap maxTokens at 100k to leave room for system prompt + response
	// Claude models have 200k context but we need to be conservative
	if maxTokens > 100000 {
		maxTokens = 100000
	}

	return &MemoryManager{
		config: MemoryConfig{
			MaxTokens:           maxTokens,
			ShortTermMessages:   3,  // Keep last 3 messages only (reduced from 6)
			TokenReservePercent: 30, // Reserve 30% for response + system (increased from 25%)
			EstimatedTokenRatio: 3,  // ~3 chars per token (conservative)
		},
		chatModel: chatModel,
	}
}

// SetShortTermMessages sets the number of recent messages to keep
func (m *MemoryManager) SetShortTermMessages(count int) {
	m.config.ShortTermMessages = count
}

// EstimateTokens estimates token count from message content
func (m *MemoryManager) EstimateTokens(messages []*schema.Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
		// Add tool calls content
		for _, tc := range msg.ToolCalls {
			totalChars += len(tc.Function.Arguments)
		}
		// Tool messages can be very large, count them carefully
		if msg.Role == schema.Tool {
			// Tool results might be huge, make sure to count them
			totalChars += len(msg.Content)
		}
	}
	return totalChars / m.config.EstimatedTokenRatio
}

// TruncateToolMessages truncates tool message content to limit size
func (m *MemoryManager) TruncateToolMessages(messages []*schema.Message, maxToolContentChars int) []*schema.Message {
	result := make([]*schema.Message, len(messages))
	for i, msg := range messages {
		// Copy message
		result[i] = &schema.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			Name:             msg.Name,
			ToolCalls:        msg.ToolCalls,
			ToolCallID:       msg.ToolCallID,
			ToolName:         msg.ToolName,
			ResponseMeta:     msg.ResponseMeta,
			ReasoningContent: msg.ReasoningContent,
			Extra:            msg.Extra,
		}

		// Truncate tool message content if too long
		if msg.Role == schema.Tool && len(msg.Content) > maxToolContentChars {
			result[i].Content = msg.Content[:maxToolContentChars] + fmt.Sprintf("\n\n[... Tool output truncated - %d chars omitted]", len(msg.Content)-maxToolContentChars)
		}
	}
	return result
}

// ManageMemory processes message history and returns optimized messages within token limit
func (m *MemoryManager) ManageMemory(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// First, truncate any extremely large tool messages (max 1500 chars per tool message)
	messages = m.TruncateToolMessages(messages, 1500)

	// Calculate available tokens (reserve some for response)
	availableTokens := m.config.MaxTokens * (100 - m.config.TokenReservePercent) / 100
	currentTokens := m.EstimateTokens(messages)

	// If within limit, return as-is
	if currentTokens <= availableTokens {
		return messages, nil
	}

	// Separate system message (if exists) from conversation
	var systemMsg *schema.Message
	conversationMsgs := messages
	if len(messages) > 0 && messages[0].Role == schema.System {
		systemMsg = messages[0]
		conversationMsgs = messages[1:]
	}

	// Apply hierarchical memory management
	optimizedMsgs, err := m.applyHierarchicalMemory(ctx, conversationMsgs, availableTokens)
	if err != nil {
		return nil, err
	}

	// Prepend system message
	if systemMsg != nil {
		optimizedMsgs = append([]*schema.Message{systemMsg}, optimizedMsgs...)
	}

	return optimizedMsgs, nil
}

// applyHierarchicalMemory implements the three-tier memory strategy
func (m *MemoryManager) applyHierarchicalMemory(ctx context.Context, messages []*schema.Message, availableTokens int) ([]*schema.Message, error) {
	if len(messages) <= m.config.ShortTermMessages {
		// All messages fit in short-term, no compression needed
		return messages, nil
	}

	// Split messages into tiers
	splitPoint := len(messages) - m.config.ShortTermMessages
	olderMessages := messages[:splitPoint]
	recentMessages := messages[splitPoint:]

	// Keep recent messages as-is (short-term memory)
	recentTokens := m.EstimateTokens(recentMessages)

	// Check if we need compression
	if recentTokens >= availableTokens {
		// Even recent messages exceed limit, need more aggressive compression
		return m.compressRecent(ctx, recentMessages, availableTokens)
	}

	// Budget for older messages
	budgetForOlder := availableTokens - recentTokens

	// Compress older messages (mid-term + long-term memory)
	compressedOlder, err := m.compressMessages(ctx, olderMessages, budgetForOlder)
	if err != nil {
		// If compression fails, just keep recent messages
		return recentMessages, nil
	}

	// Combine compressed older + recent messages
	result := append(compressedOlder, recentMessages...)
	return result, nil
}

// compressMessages summarizes a batch of messages into a condensed form using simple text extraction
func (m *MemoryManager) compressMessages(ctx context.Context, messages []*schema.Message, targetTokens int) ([]*schema.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Simple compression: create a summary message with key information extracted
	summary := m.extractKeyInformation(messages, targetTokens)

	// Create a summary message
	summaryMessage := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("[Previous Conversation Summary - %d messages compressed]\n%s\n[End of Summary]", len(messages), summary),
	}

	return []*schema.Message{summaryMessage}, nil
}

// extractKeyInformation extracts key information from messages without calling LLM
func (m *MemoryManager) extractKeyInformation(messages []*schema.Message, targetTokens int) string {
	var sb strings.Builder

	// Count user questions and assistant key findings
	userQuestions := []string{}
	assistantFindings := []string{}

	for _, msg := range messages {
		if msg.Role == schema.User {
			// Extract user questions (truncate if too long)
			content := strings.TrimSpace(msg.Content)
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			if content != "" && !strings.HasPrefix(content, "[Previous Conversation Summary") {
				userQuestions = append(userQuestions, content)
			}
		} else if msg.Role == schema.Assistant {
			// Extract key phrases from assistant (first 300 chars)
			content := strings.TrimSpace(msg.Content)
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			if content != "" {
				assistantFindings = append(assistantFindings, content)
			}
		}
	}

	// Build summary
	if len(userQuestions) > 0 {
		sb.WriteString("User asked about:\n")
		// Keep only last 5 questions
		start := 0
		if len(userQuestions) > 5 {
			start = len(userQuestions) - 5
			sb.WriteString(fmt.Sprintf("... [%d earlier questions omitted] ...\n", start))
		}
		for i := start; i < len(userQuestions); i++ {
			sb.WriteString(fmt.Sprintf("- %s\n", userQuestions[i]))
		}
		sb.WriteString("\n")
	}

	if len(assistantFindings) > 0 {
		sb.WriteString("Key findings:\n")
		// Keep only last 3 findings
		start := 0
		if len(assistantFindings) > 3 {
			start = len(assistantFindings) - 3
			sb.WriteString(fmt.Sprintf("... [%d earlier findings omitted] ...\n", start))
		}
		for i := start; i < len(assistantFindings); i++ {
			sb.WriteString(fmt.Sprintf("- %s\n", assistantFindings[i]))
		}
	}

	summary := sb.String()

	// Final truncation to fit target tokens
	maxChars := targetTokens * m.config.EstimatedTokenRatio
	if len(summary) > maxChars {
		summary = summary[:maxChars] + "\n... [Summary truncated to fit context]"
	}

	return summary
}

// compressRecent aggressively compresses even recent messages when needed
func (m *MemoryManager) compressRecent(ctx context.Context, messages []*schema.Message, targetTokens int) ([]*schema.Message, error) {
	// Keep only the last 2-3 messages if possible
	minMessages := 3
	if len(messages) <= minMessages {
		// Can't compress further, return as-is and let the API handle it
		return messages, nil
	}

	// Keep last few messages, compress the rest
	keepCount := minMessages
	toCompress := messages[:len(messages)-keepCount]
	toKeep := messages[len(messages)-keepCount:]

	budgetForCompressed := targetTokens - m.EstimateTokens(toKeep)
	if budgetForCompressed <= 0 {
		// Not enough budget, just keep the minimal messages
		return toKeep, nil
	}

	compressed, err := m.compressMessages(ctx, toCompress, budgetForCompressed)
	if err != nil {
		return toKeep, nil
	}

	return append(compressed, toKeep...), nil
}

// formatMessagesForSummary converts messages to readable text for summary generation
func (m *MemoryManager) formatMessagesForSummary(messages []*schema.Message) string {
	var sb strings.Builder
	for i, msg := range messages {
		roleName := "User"
		if msg.Role == schema.Assistant {
			roleName = "Assistant"
		} else if msg.Role == schema.System {
			roleName = "System"
		} else if msg.Role == schema.Tool {
			roleName = "Tool"
		}

		sb.WriteString(fmt.Sprintf("%d. %s: ", i+1, roleName))

		// Include message content
		if msg.Content != "" {
			// Truncate very long content
			content := msg.Content
			if len(content) > 1000 {
				content = content[:1000] + "... [truncated]"
			}
			sb.WriteString(content)
			sb.WriteString("\n")
		}

		// Include tool call information (simplified)
		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  [Tools used: ")
			toolNames := []string{}
			for _, tc := range msg.ToolCalls {
				toolNames = append(toolNames, tc.Function.Name)
			}
			sb.WriteString(strings.Join(toolNames, ", "))
			sb.WriteString("]\n")
		}

		sb.WriteString("\n")
	}
	return sb.String()
}

// TruncateDataContext truncates data source context to fit within limits
func (m *MemoryManager) TruncateDataContext(contextPrompt string, maxChars int) string {
	if len(contextPrompt) <= maxChars {
		return contextPrompt
	}

	// Preserve header and truncate schema details
	lines := strings.Split(contextPrompt, "\n")
	var result strings.Builder
	currentLen := 0

	for _, line := range lines {
		if currentLen+len(line) > maxChars {
			result.WriteString("\n... [Schema details truncated for length] ...")
			break
		}
		result.WriteString(line)
		result.WriteString("\n")
		currentLen += len(line) + 1
	}

	return result.String()
}

// SerializeMemoryState exports memory state for persistence
func (m *MemoryManager) SerializeMemoryState(messages []*schema.Message) (string, error) {
	data, err := json.Marshal(messages)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializeMemoryState imports memory state from persistence
func (m *MemoryManager) DeserializeMemoryState(data string) ([]*schema.Message, error) {
	var messages []*schema.Message
	err := json.Unmarshal([]byte(data), &messages)
	if err != nil {
		return nil, err
	}
	return messages, nil
}
