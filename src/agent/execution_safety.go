package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ExecutionSafety provides safety mechanisms for code execution
type ExecutionSafety struct {
	codeValidator *CodeValidator
	logger        func(string)
	timeout       time.Duration
}

// SafeExecutionResult represents the result of a safe execution
type SafeExecutionResult struct {
	Success      bool          `json:"success"`
	Output       string        `json:"output"`
	Error        string        `json:"error"`
	TimedOut     bool          `json:"timed_out"`
	Blocked      bool          `json:"blocked"`
	BlockReason  string        `json:"block_reason"`
	Duration     time.Duration `json:"duration"`
}

// NewExecutionSafety creates a new execution safety wrapper
func NewExecutionSafety(logger func(string)) *ExecutionSafety {
	return &ExecutionSafety{
		codeValidator: NewCodeValidator(),
		logger:        logger,
		timeout:       120 * time.Second, // Default 2 minute timeout
	}
}

func (s *ExecutionSafety) log(msg string) {
	if s.logger != nil {
		s.logger(msg)
	}
}

// SetTimeout sets the execution timeout
func (s *ExecutionSafety) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
}

// ValidateAndExecute validates code safety and executes with timeout
func (s *ExecutionSafety) ValidateAndExecute(
	ctx context.Context,
	code string,
	executor func(code string) (string, error),
) *SafeExecutionResult {
	startTime := time.Now()
	result := &SafeExecutionResult{
		Success: false,
	}

	// 1. Validate code safety
	s.log("[SAFETY] Validating code safety...")
	validation := s.codeValidator.ValidateCode(code)
	
	if !validation.Valid {
		result.Blocked = true
		result.BlockReason = fmt.Sprintf("Code validation failed: %v", validation.Errors)
		result.Duration = time.Since(startTime)
		s.log(fmt.Sprintf("[SAFETY] Code blocked: %s", result.BlockReason))
		return result
	}

	// Log warnings
	for _, warning := range validation.Warnings {
		s.log(fmt.Sprintf("[SAFETY] Warning: %s", warning))
	}

	// 2. Execute with timeout
	s.log(fmt.Sprintf("[SAFETY] Executing code with %v timeout...", s.timeout))
	
	// Create a context with timeout
	execCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Channel for execution result
	resultChan := make(chan struct {
		output string
		err    error
	}, 1)

	// Execute in goroutine
	go func() {
		output, err := executor(code)
		resultChan <- struct {
			output string
			err    error
		}{output, err}
	}()

	// Wait for result or timeout
	select {
	case execResult := <-resultChan:
		result.Duration = time.Since(startTime)
		if execResult.err != nil {
			result.Error = execResult.err.Error()
			result.Output = execResult.output
			s.log(fmt.Sprintf("[SAFETY] Execution failed: %s", result.Error))
		} else {
			result.Success = true
			result.Output = execResult.output
			s.log(fmt.Sprintf("[SAFETY] Execution succeeded in %v", result.Duration))
		}

	case <-execCtx.Done():
		result.TimedOut = true
		result.Duration = time.Since(startTime)
		result.Error = fmt.Sprintf("Execution timed out after %v", s.timeout)
		s.log(fmt.Sprintf("[SAFETY] Execution timed out after %v", s.timeout))
	}

	return result
}

// ValidateSQL validates SQL queries for safety
func (s *ExecutionSafety) ValidateSQL(queries []string) (bool, []string) {
	var errors []string
	
	err := s.codeValidator.ValidateSQLQueries(queries)
	if err != nil {
		errors = append(errors, err.Error())
	}
	
	return len(errors) == 0, errors
}

// CheckResourceLimits checks if the code might exceed resource limits
func (s *ExecutionSafety) CheckResourceLimits(code string) []string {
	var warnings []string
	codeLower := strings.ToLower(code)

	// Check for potentially memory-intensive operations
	if strings.Contains(codeLower, "read_csv") && !strings.Contains(codeLower, "chunksize") {
		warnings = append(warnings, "Large CSV read without chunking may cause memory issues")
	}

	// Check for infinite loops
	if strings.Contains(codeLower, "while true") && !strings.Contains(codeLower, "break") {
		warnings = append(warnings, "Potential infinite loop detected")
	}

	// Check for recursive functions without base case
	if strings.Contains(codeLower, "def ") && strings.Contains(codeLower, "return ") {
		// This is a simple heuristic, not foolproof
		if strings.Count(codeLower, "def ") > 0 {
			funcName := extractFunctionName(code)
			if funcName != "" && strings.Count(codeLower, funcName+"(") > 1 {
				warnings = append(warnings, "Recursive function detected - ensure base case exists")
			}
		}
	}

	// Check for large data operations
	if strings.Contains(codeLower, "cross_join") || strings.Contains(codeLower, "cartesian") {
		warnings = append(warnings, "Cartesian product may cause memory issues with large datasets")
	}

	return warnings
}

// extractFunctionName extracts the first function name from code
func extractFunctionName(code string) string {
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") {
			// Extract function name
			parts := strings.Split(trimmed[4:], "(")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}
	}
	return ""
}

// SafetyReport generates a safety report for the code
type SafetyReport struct {
	IsSafe       bool     `json:"is_safe"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
	SQLQueries   []string `json:"sql_queries"`
	HasChart     bool     `json:"has_chart"`
	HasExport    bool     `json:"has_export"`
	ResourceRisk string   `json:"resource_risk"` // "low", "medium", "high"
}

// GenerateSafetyReport generates a comprehensive safety report
func (s *ExecutionSafety) GenerateSafetyReport(code string) *SafetyReport {
	report := &SafetyReport{
		IsSafe:       true,
		Errors:       []string{},
		Warnings:     []string{},
		SQLQueries:   []string{},
		ResourceRisk: "low",
	}

	// Validate code
	validation := s.codeValidator.ValidateCode(code)
	report.Errors = validation.Errors
	report.Warnings = validation.Warnings
	report.SQLQueries = validation.SQLQueries
	report.HasChart = validation.HasChart
	report.HasExport = validation.HasExport

	if !validation.Valid {
		report.IsSafe = false
	}

	// Check resource limits
	resourceWarnings := s.CheckResourceLimits(code)
	report.Warnings = append(report.Warnings, resourceWarnings...)

	// Determine resource risk level
	if len(resourceWarnings) > 2 {
		report.ResourceRisk = "high"
	} else if len(resourceWarnings) > 0 {
		report.ResourceRisk = "medium"
	}

	return report
}

// truncateSafetyString truncates a string to maxLen characters
func truncateSafetyString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
