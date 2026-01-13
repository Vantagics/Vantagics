package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AnalysisRecorder records analysis steps for later replay
type AnalysisRecorder struct {
	recording *AnalysisRecording
	mu        sync.Mutex
	enabled   bool
	stepCount int
}

// NewAnalysisRecorder creates a new analysis recorder
func NewAnalysisRecorder(sourceID, sourceName string, schema []ReplayTableSchema) *AnalysisRecorder {
	return &AnalysisRecorder{
		recording: &AnalysisRecording{
			RecordingID:     fmt.Sprintf("rec_%d", time.Now().Unix()),
			CreatedAt:       time.Now(),
			SourceID:        sourceID,
			SourceName:      sourceName,
			SourceSchema:    schema,
			Steps:           []AnalysisStep{},
			LLMConversation: []ConversationTurn{},
		},
		enabled:   true,
		stepCount: 0,
	}
}

// RecordStep records an analysis step
func (r *AnalysisRecorder) RecordStep(toolName, description, input, output, chartType, chartData string) {
	if !r.enabled {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.stepCount++
	step := AnalysisStep{
		StepID:      r.stepCount,
		Timestamp:   time.Now(),
		ToolName:    toolName,
		Description: description,
		Input:       input,
		Output:      output,
		ChartType:   chartType,
		ChartData:   chartData,
	}

	r.recording.Steps = append(r.recording.Steps, step)
}

// RecordConversation records an LLM conversation turn
func (r *AnalysisRecorder) RecordConversation(role, content string) {
	if !r.enabled {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	turn := ConversationTurn{
		Role:    role,
		Content: content,
	}

	r.recording.LLMConversation = append(r.recording.LLMConversation, turn)
}

// SetMetadata sets recording metadata
func (r *AnalysisRecorder) SetMetadata(title, description string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.recording.Title = title
	r.recording.Description = description
}

// Enable enables recording
func (r *AnalysisRecorder) Enable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = true
}

// Disable disables recording
func (r *AnalysisRecorder) Disable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = false
}

// GetRecording returns the current recording
func (r *AnalysisRecorder) GetRecording() *AnalysisRecording {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recording
}

// SaveRecording saves the recording to a file
func (r *AnalysisRecorder) SaveRecording(dirPath string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("recording_%s.json", r.recording.RecordingID)
	filePath := filepath.Join(dirPath, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(r.recording, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal recording: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

// LoadRecording loads a recording from a file
func LoadRecording(filePath string) (*AnalysisRecording, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var recording AnalysisRecording
	if err := json.Unmarshal(data, &recording); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recording: %w", err)
	}

	return &recording, nil
}
