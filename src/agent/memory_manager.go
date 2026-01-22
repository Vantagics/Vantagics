package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
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
	config         MemoryConfig
	chatModel      model.ChatModel
	memoryService  *MemoryService // Optional: for persisting mid-term summaries
	currentThreadID string         // Current thread ID for memory persistence
}

// NewMemoryManager creates a new memory manager
// maxTokens here represents the OUTPUT token limit from config, NOT the context window
// We need to determine the actual context window size based on the model
func NewMemoryManager(maxTokens int, chatModel model.ChatModel) *MemoryManager {
	// Determine context window size based on common model limits
	// This is separate from output token limit (maxTokens parameter)
	contextWindowSize := 128000 // Default to 128k for modern models
	
	// If maxTokens suggests a smaller model, adjust context window
	if maxTokens > 0 && maxTokens < 8192 {
		contextWindowSize = 32000 // Smaller model, use 32k context
	}
	
	// Reserve tokens for output - use the configured maxTokens or default to 8192
	outputReserve := maxTokens
	if outputReserve <= 0 {
		outputReserve = 8192 // Default output reserve
	}
	
	// Available tokens for input context = context window - output reserve
	availableForInput := contextWindowSize - outputReserve
	if availableForInput < 10000 {
		// Safety check: ensure at least 10k tokens for input
		availableForInput = 10000
	}

	return &MemoryManager{
		config: MemoryConfig{
			MaxTokens:           availableForInput, // Use available input tokens, not output limit
			ShortTermMessages:   5,                 // Keep more recent messages
			TokenReservePercent: 20,                // Reserve 20% as safety buffer
			EstimatedTokenRatio: 3,                 // ~3 chars per token (conservative)
		},
		chatModel: chatModel,
	}
}

// SetMemoryService sets the memory service for persisting mid-term summaries
func (m *MemoryManager) SetMemoryService(service *MemoryService, threadID string) {
	m.memoryService = service
	m.currentThreadID = threadID
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

	// Use the full available input tokens (output is reserved separately in API call)
	targetTokens := m.config.MaxTokens * (100 - m.config.TokenReservePercent) / 100

	// First pass: moderate truncation of tool messages (max 10000 chars each)
	messages = m.TruncateToolMessages(messages, 10000)
	currentTokens := m.EstimateTokens(messages)

	// If still too large, more aggressive truncation (5000 chars)
	if currentTokens > targetTokens {
		messages = m.TruncateToolMessages(messages, 5000)
		currentTokens = m.EstimateTokens(messages)
	}

	// If STILL too large, strip tool messages entirely from older messages
	if currentTokens > targetTokens {
		messages = m.stripOldToolMessages(messages)
		currentTokens = m.EstimateTokens(messages)
	}

	// If within limit, return as-is
	if currentTokens <= targetTokens {
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
	optimizedMsgs, err := m.applyHierarchicalMemory(ctx, conversationMsgs, targetTokens)
	if err != nil {
		return nil, err
	}

	// Prepend system message
	if systemMsg != nil {
		optimizedMsgs = append([]*schema.Message{systemMsg}, optimizedMsgs...)
	}

	// Final safety check - if still too large, keep only last 2 user/assistant pairs
	finalTokens := m.EstimateTokens(optimizedMsgs)
	if finalTokens > targetTokens {
		optimizedMsgs = m.keepOnlyRecent(optimizedMsgs, 4) // Keep last 4 messages max
	}

	return optimizedMsgs, nil
}

// stripOldToolMessages removes tool messages from history except the most recent ones
// CRITICAL: Maintains message sequence integrity - assistant+tool pairs are atomic
func (m *MemoryManager) stripOldToolMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) <= 6 {
		return messages
	}

	// Keep last 6 messages intact, remove assistant+tool pairs from older ones
	result := make([]*schema.Message, 0, len(messages))
	cutoff := len(messages) - 6

	// Track indices to skip (tool messages whose assistant was removed)
	skipIndices := make(map[int]bool)

	// First pass: identify assistant messages with tool_calls that should be removed
	for i := 0; i < cutoff; i++ {
		msg := messages[i]
		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			// Mark this assistant and its subsequent tool messages for removal
			skipIndices[i] = true
			// Find and mark corresponding tool result messages
			for j := i + 1; j < len(messages); j++ {
				if messages[j].Role == schema.Tool {
					// Check if this tool message belongs to this assistant
					skipIndices[j] = true
				} else if messages[j].Role == schema.Assistant {
					// Stop at next assistant message
					break
				}
			}
		}
	}

	// Second pass: build result, skipping marked messages
	for i, msg := range messages {
		if skipIndices[i] {
			// Skip this message entirely to maintain sequence integrity
			continue
		}
		result = append(result, msg)
	}

	return result
}

// keepOnlyRecent keeps only the most recent messages
// CRITICAL: Ensures message sequence integrity for tool calls
func (m *MemoryManager) keepOnlyRecent(messages []*schema.Message, maxCount int) []*schema.Message {
	// Always keep system message if present
	var systemMsg *schema.Message
	conversation := messages

	if len(messages) > 0 && messages[0].Role == schema.System {
		systemMsg = messages[0]
		conversation = messages[1:]
	}

	// Keep only last maxCount messages from conversation
	if len(conversation) > maxCount {
		startIdx := len(conversation) - maxCount

		// CRITICAL: Ensure we don't start in the middle of an assistant+tool sequence
		// If the first message we're keeping is a Tool message, we need to include
		// the preceding Assistant message with tool_calls
		for startIdx > 0 && conversation[startIdx].Role == schema.Tool {
			startIdx--
			// Also need to find the assistant message that made this tool call
			for startIdx > 0 && conversation[startIdx].Role != schema.Assistant {
				startIdx--
			}
		}

		conversation = conversation[startIdx:]
	}

	if systemMsg != nil {
		return append([]*schema.Message{systemMsg}, conversation...)
	}
	return conversation
}

// ensureSafeSplitPoint adjusts the split point to avoid breaking assistant+tool sequences
// Returns a safe split point that doesn't separate tool calls from their results
func (m *MemoryManager) ensureSafeSplitPoint(messages []*schema.Message, proposedSplit int) int {
	if proposedSplit <= 0 || proposedSplit >= len(messages) {
		return proposedSplit
	}

	// If the message at split point is a Tool message, we need to move the split
	// to after all tool results for the current tool call sequence
	splitIdx := proposedSplit
	for splitIdx < len(messages) && messages[splitIdx].Role == schema.Tool {
		splitIdx++
	}

	return splitIdx
}

// applyHierarchicalMemory implements the three-tier memory strategy
// CRITICAL: Maintains message sequence integrity for tool calls
func (m *MemoryManager) applyHierarchicalMemory(ctx context.Context, messages []*schema.Message, availableTokens int) ([]*schema.Message, error) {
	if len(messages) <= m.config.ShortTermMessages {
		// All messages fit in short-term, no compression needed
		return messages, nil
	}

	// Split messages into tiers - ensure split doesn't break tool sequences
	proposedSplit := len(messages) - m.config.ShortTermMessages
	splitPoint := m.ensureSafeSplitPoint(messages, proposedSplit)

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
// The summary is also persisted to mid-term memory if MemoryService is available
func (m *MemoryManager) compressMessages(ctx context.Context, messages []*schema.Message, targetTokens int) ([]*schema.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Simple compression: create a summary message with key information extracted
	summary := m.extractKeyInformation(messages, targetTokens)

	// Persist to mid-term memory if service is available
	if m.memoryService != nil && m.currentThreadID != "" {
		// Store the compressed summary in mid-term memory
		err := m.memoryService.AddSessionMediumTermMemory(m.currentThreadID, summary)
		if err != nil {
			// Log error but don't fail the compression
			// (we could add a logger here if needed)
		}
	}

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
// CRITICAL: Maintains message sequence integrity for tool calls
func (m *MemoryManager) compressRecent(ctx context.Context, messages []*schema.Message, targetTokens int) ([]*schema.Message, error) {
	// Keep only the last 2-3 messages if possible
	minMessages := 3
	if len(messages) <= minMessages {
		// Can't compress further, return as-is and let the API handle it
		return messages, nil
	}

	// Keep last few messages, compress the rest
	keepCount := minMessages
	proposedSplit := len(messages) - keepCount

	// Ensure split point doesn't break tool sequences
	splitPoint := m.ensureSafeSplitPoint(messages, proposedSplit)

	toCompress := messages[:splitPoint]
	toKeep := messages[splitPoint:]

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
