package agent

import (
	"fmt"
	"sync"
	"time"
)

// AnalysisMetrics tracks performance metrics for analysis requests
type AnalysisMetrics struct {
	mu              sync.Mutex
	schemaFetchTime time.Duration
	codeGenTime     time.Duration
	executionTime   time.Duration
	llmCallCount    int
	toolCallCount   int
	startTime       time.Time
	logger          func(string)
}

// MetricsSummary represents aggregated metrics
type MetricsSummary struct {
	TotalDuration    time.Duration `json:"total_duration"`
	SchemaFetchTime  time.Duration `json:"schema_fetch_time"`
	CodeGenTime      time.Duration `json:"code_gen_time"`
	ExecutionTime    time.Duration `json:"execution_time"`
	LLMCallCount     int           `json:"llm_call_count"`
	ToolCallCount    int           `json:"tool_call_count"`
	BaselineEstimate time.Duration `json:"baseline_estimate"`
	Improvement      float64       `json:"improvement_percent"`
}

// NewAnalysisMetrics creates a new metrics collector
func NewAnalysisMetrics(logger func(string)) *AnalysisMetrics {
	return &AnalysisMetrics{
		startTime: time.Now(),
		logger:    logger,
	}
}

func (m *AnalysisMetrics) log(msg string) {
	if m.logger != nil {
		m.logger(msg)
	}
}

// RecordSchemaFetch records schema fetch timing
func (m *AnalysisMetrics) RecordSchemaFetch(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schemaFetchTime = duration
	m.toolCallCount++
	m.log(fmt.Sprintf("[METRICS] Schema fetch: %v", duration))
}

// RecordCodeGeneration records code generation timing
func (m *AnalysisMetrics) RecordCodeGeneration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codeGenTime = duration
	m.log(fmt.Sprintf("[METRICS] Code generation: %v", duration))
}

// RecordExecution records Python execution timing
func (m *AnalysisMetrics) RecordExecution(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executionTime = duration
	m.toolCallCount++
	m.log(fmt.Sprintf("[METRICS] Execution: %v", duration))
}

// IncrementLLMCalls increments the LLM call counter
func (m *AnalysisMetrics) IncrementLLMCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.llmCallCount++
}

// IncrementToolCalls increments the tool call counter
func (m *AnalysisMetrics) IncrementToolCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolCallCount++
}

// GetSummary returns a summary of all metrics
func (m *AnalysisMetrics) GetSummary() *MetricsSummary {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalDuration := time.Since(m.startTime)
	
	// Estimate baseline (traditional multi-step approach)
	// Typically: 3-4 LLM calls, each ~2-3 seconds
	baselineEstimate := time.Duration(m.llmCallCount*3) * 3 * time.Second
	if baselineEstimate == 0 {
		baselineEstimate = 9 * time.Second // Default estimate for 3 LLM calls
	}

	// Calculate improvement
	var improvement float64
	if baselineEstimate > 0 {
		improvement = float64(baselineEstimate-totalDuration) / float64(baselineEstimate) * 100
		if improvement < 0 {
			improvement = 0
		}
	}

	return &MetricsSummary{
		TotalDuration:    totalDuration,
		SchemaFetchTime:  m.schemaFetchTime,
		CodeGenTime:      m.codeGenTime,
		ExecutionTime:    m.executionTime,
		LLMCallCount:     m.llmCallCount,
		ToolCallCount:    m.toolCallCount,
		BaselineEstimate: baselineEstimate,
		Improvement:      improvement,
	}
}

// LogSummary logs a summary of the metrics
func (m *AnalysisMetrics) LogSummary() {
	summary := m.GetSummary()
	
	m.log("=== 分析性能指标 ===")
	m.log(fmt.Sprintf("总耗时: %v", summary.TotalDuration))
	m.log(fmt.Sprintf("Schema获取: %v", summary.SchemaFetchTime))
	m.log(fmt.Sprintf("代码生成: %v", summary.CodeGenTime))
	m.log(fmt.Sprintf("代码执行: %v", summary.ExecutionTime))
	m.log(fmt.Sprintf("LLM调用次数: %d", summary.LLMCallCount))
	m.log(fmt.Sprintf("工具调用次数: %d", summary.ToolCallCount))
	m.log(fmt.Sprintf("基线估计: %v", summary.BaselineEstimate))
	m.log(fmt.Sprintf("性能提升: %.1f%%", summary.Improvement))
}

// CheckTimeout checks if the operation has exceeded the timeout
func (m *AnalysisMetrics) CheckTimeout(timeout time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	elapsed := time.Since(m.startTime)
	if elapsed > timeout {
		m.log(fmt.Sprintf("[METRICS] WARNING: Operation exceeded timeout (%v > %v)", elapsed, timeout))
		return true
	}
	return false
}

// Reset resets all metrics
func (m *AnalysisMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.schemaFetchTime = 0
	m.codeGenTime = 0
	m.executionTime = 0
	m.llmCallCount = 0
	m.toolCallCount = 0
	m.startTime = time.Now()
}
