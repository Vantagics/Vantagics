package agent

import (
	"fmt"
	"sync"
)

// ExecutionMetrics tracks execution statistics
type ExecutionMetrics struct {
	PlannedCalls   int
	ActualCalls    int
	PlannedSteps   []string
	ActualSteps    []string
	DeviationScore float64 // 0.0 = perfect match, 1.0 = completely different
	Warnings       []string
}

// ExecutionValidator validates execution plans and tracks deviations
type ExecutionValidator struct {
	logger         func(string)
	plannedCalls   int
	actualCalls    int
	plannedSteps   []string
	actualSteps    []string
	warnings       []string
	mu             sync.RWMutex
}

// NewExecutionValidator creates a new execution validator
func NewExecutionValidator(logger func(string)) *ExecutionValidator {
	return &ExecutionValidator{
		logger:       logger,
		plannedSteps: make([]string, 0),
		actualSteps:  make([]string, 0),
		warnings:     make([]string, 0),
	}
}

// ValidatePlan validates that a plan is consistent with request type
func (v *ExecutionValidator) ValidatePlan(plan *AnalysisPlan, requestType RequestType) (*AnalysisPlan, []string) {
	if plan == nil {
		return nil, []string{"plan is nil"}
	}

	warnings := make([]string, 0)

	// Validate consultation requests don't have SQL steps
	if requestType == RequestTypeConsultation {
		for i, step := range plan.Steps {
			if step.Tool == "execute_sql" {
				warning := fmt.Sprintf("Consultation request should not have SQL step at position %d", i)
				warnings = append(warnings, warning)
				if v.logger != nil {
					v.logger(fmt.Sprintf("[VALIDATOR] %s", warning))
				}
				// Remove the SQL step
				plan.Steps = append(plan.Steps[:i], plan.Steps[i+1:]...)
				plan.EstimatedCalls--
			}
		}
	}

	// Validate multi-step analysis has checkpoints
	if requestType == RequestTypeMultiStepAnalysis {
		if len(plan.Checkpoints) == 0 {
			warning := "Multi-step analysis should have at least one checkpoint"
			warnings = append(warnings, warning)
			if v.logger != nil {
				v.logger(fmt.Sprintf("[VALIDATOR] %s", warning))
			}
			// Add checkpoints at reasonable intervals
			if len(plan.Steps) > 2 {
				plan.Checkpoints = append(plan.Checkpoints, len(plan.Steps)/2)
			}
		}
	}

	// Validate tool names are exact
	validTools := map[string]bool{
		"get_data_source_context": true,
		"execute_sql":             true,
		"python_executor":         true,
		"web_search":              true,
		"web_fetch":               true,
	}

	for i, step := range plan.Steps {
		if !validTools[step.Tool] {
			warning := fmt.Sprintf("Invalid tool name at step %d: %s", i, step.Tool)
			warnings = append(warnings, warning)
			if v.logger != nil {
				v.logger(fmt.Sprintf("[VALIDATOR] %s", warning))
			}
		}

		// Validate schema level for get_data_source_context
		if step.Tool == "get_data_source_context" && step.SchemaLevel == "" {
			warning := fmt.Sprintf("Step %d (get_data_source_context) missing schema level", i)
			warnings = append(warnings, warning)
			if v.logger != nil {
				v.logger(fmt.Sprintf("[VALIDATOR] %s", warning))
			}
			step.SchemaLevel = string(SchemaLevelBasic)
		}

		// Validate query type for execute_sql
		if step.Tool == "execute_sql" && step.QueryType == "" {
			warning := fmt.Sprintf("Step %d (execute_sql) missing query type", i)
			warnings = append(warnings, warning)
			if v.logger != nil {
				v.logger(fmt.Sprintf("[VALIDATOR] %s", warning))
			}
			step.QueryType = "general"
		}
	}

	v.mu.Lock()
	v.plannedSteps = make([]string, len(plan.Steps))
	v.plannedCalls = len(plan.Steps)
	for i, step := range plan.Steps {
		v.plannedSteps[i] = step.Tool
	}
	v.mu.Unlock()

	if v.logger != nil {
		v.logger(fmt.Sprintf("[VALIDATOR] Plan validated: %d steps, %d warnings", len(plan.Steps), len(warnings)))
	}

	return plan, warnings
}

// TrackExecution records actual tool calls during execution
func (v *ExecutionValidator) TrackExecution(toolName string, stepNum int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.actualSteps = append(v.actualSteps, toolName)
	v.actualCalls++

	if v.logger != nil {
		v.logger(fmt.Sprintf("[VALIDATOR] Tracked execution: step %d, tool %s (total: %d)", stepNum, toolName, v.actualCalls))
	}
}

// GetMetrics returns execution metrics after completion
func (v *ExecutionValidator) GetMetrics() *ExecutionMetrics {
	v.mu.RLock()
	defer v.mu.RUnlock()

	metrics := &ExecutionMetrics{
		PlannedCalls:   v.plannedCalls,
		ActualCalls:    v.actualCalls,
		PlannedSteps:   make([]string, len(v.plannedSteps)),
		ActualSteps:    make([]string, len(v.actualSteps)),
		Warnings:       make([]string, len(v.warnings)),
	}

	copy(metrics.PlannedSteps, v.plannedSteps)
	copy(metrics.ActualSteps, v.actualSteps)
	copy(metrics.Warnings, v.warnings)

	// Calculate deviation score
	metrics.DeviationScore = v.calculateDeviationScore()

	return metrics
}

// calculateDeviationScore calculates how much the actual execution deviated from the plan
func (v *ExecutionValidator) calculateDeviationScore() float64 {
	if v.plannedCalls == 0 {
		return 0.0
	}

	// Simple deviation calculation: difference in call count / planned calls
	callDiff := float64(v.actualCalls - v.plannedCalls)
	if callDiff < 0 {
		callDiff = -callDiff
	}

	callDeviation := callDiff / float64(v.plannedCalls)

	// Step order deviation: count mismatches in step sequence
	stepDeviation := 0.0
	minLen := len(v.plannedSteps)
	if len(v.actualSteps) < minLen {
		minLen = len(v.actualSteps)
	}

	if minLen > 0 {
		mismatches := 0
		for i := 0; i < minLen; i++ {
			if v.plannedSteps[i] != v.actualSteps[i] {
				mismatches++
			}
		}
		stepDeviation = float64(mismatches) / float64(minLen)
	}

	// Combined deviation score (average of call and step deviations)
	totalDeviation := (callDeviation + stepDeviation) / 2.0

	// Cap at 1.0
	if totalDeviation > 1.0 {
		totalDeviation = 1.0
	}

	return totalDeviation
}

// LogDeviations logs warnings if execution deviated significantly from plan
func (v *ExecutionValidator) LogDeviations() {
	v.mu.Lock()
	defer v.mu.Unlock()

	metrics := &ExecutionMetrics{
		PlannedCalls:   v.plannedCalls,
		ActualCalls:    v.actualCalls,
		PlannedSteps:   v.plannedSteps,
		ActualSteps:    v.actualSteps,
	}
	metrics.DeviationScore = v.calculateDeviationScore()

	if v.logger != nil {
		v.logger(fmt.Sprintf("[VALIDATOR] Execution metrics: planned=%d, actual=%d, deviation=%.2f",
			metrics.PlannedCalls, metrics.ActualCalls, metrics.DeviationScore))
	}

	// Log warning if deviation exceeds threshold
	if metrics.DeviationScore > 0.5 {
		warning := fmt.Sprintf("Execution deviated significantly from plan (score: %.2f)", metrics.DeviationScore)
		v.warnings = append(v.warnings, warning)

		if v.logger != nil {
			v.logger(fmt.Sprintf("[VALIDATOR] ⚠️ %s", warning))
			v.logger(fmt.Sprintf("[VALIDATOR] Planned steps: %v", metrics.PlannedSteps))
			v.logger(fmt.Sprintf("[VALIDATOR] Actual steps: %v", metrics.ActualSteps))
		}
	}
}

// AddWarning adds a warning message
func (v *ExecutionValidator) AddWarning(warning string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.warnings = append(v.warnings, warning)

	if v.logger != nil {
		v.logger(fmt.Sprintf("[VALIDATOR] Warning: %s", warning))
	}
}

// Reset resets the validator for a new execution
func (v *ExecutionValidator) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.plannedCalls = 0
	v.actualCalls = 0
	v.plannedSteps = make([]string, 0)
	v.actualSteps = make([]string, 0)
	v.warnings = make([]string, 0)

	if v.logger != nil {
		v.logger("[VALIDATOR] Reset for new execution")
	}
}
