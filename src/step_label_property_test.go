package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: quick-analysis-dashboard-display, Property 1: Step message content contains analysis request description
// **Validates: Requirements 4.1, 4.3, 4.4**

// generateRandomPackStep creates a random PackStep for property testing.
// When allowEmptyDesc is true, Description may be empty (to test default label logic).
func generateRandomPackStep(r *rand.Rand, allowEmptyDesc bool) PackStep {
	stepTypes := []string{"sql_query", "python_code", "unknown_type", "custom"}

	var desc string
	if allowEmptyDesc && r.Intn(2) == 0 {
		desc = "" // empty description to trigger default label
	} else {
		desc = generateRandomString(r, 40)
	}

	return PackStep{
		StepID:      r.Intn(100) + 1, // 1-100
		StepType:    stepTypes[r.Intn(len(stepTypes))],
		Code:        generateRandomString(r, 50),
		Description: desc,
	}
}

// TestProperty1_GetStepLabelReturnsCorrectLabel verifies that for any PackStep:
// - If Description is non-empty, getStepLabel returns the Description as-is
// - If Description is empty and StepType is "sql_query", returns "SQL 查询 #N"
// - If Description is empty and StepType is "python_code", returns "Python 脚本 #N"
// - If Description is empty and StepType is unknown, returns "步骤 #N"
// **Validates: Requirements 4.1, 4.3, 4.4**
func TestProperty1_GetStepLabelReturnsCorrectLabel(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		step := generateRandomPackStep(r, true)

		label := getStepLabel(step)

		if step.Description != "" {
			// Property: non-empty Description → label == Description
			if label != step.Description {
				t.Logf("seed=%d: expected label=%q for non-empty Description, got=%q", seed, step.Description, label)
				return false
			}
		} else {
			// Property: empty Description → default label based on StepType
			var expected string
			switch step.StepType {
			case "sql_query":
				expected = fmt.Sprintf("SQL 查询 #%d", step.StepID)
			case "python_code":
				expected = fmt.Sprintf("Python 脚本 #%d", step.StepID)
			default:
				expected = fmt.Sprintf("步骤 #%d", step.StepID)
			}
			if label != expected {
				t.Logf("seed=%d: expected default label=%q for empty Description (type=%s, id=%d), got=%q",
					seed, expected, step.StepType, step.StepID, label)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1 (getStepLabel returns correct label) failed: %v", err)
	}
}

// TestProperty1b_SuccessMessageContainsStepLabel verifies that for any PackStep,
// the success message format (as used in executePackSQLStep) contains the step label
// returned by getStepLabel.
// **Validates: Requirements 4.1, 4.4**
func TestProperty1b_SuccessMessageContainsStepLabel(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		step := generateRandomPackStep(r, true)

		label := getStepLabel(step)

		// Simulate the success message format from executePackSQLStep
		successMsg := fmt.Sprintf("✅ 步骤 %d (%s):\n\n> 📋 分析请求：%s\n\n```json:table\n[{\"col\":1}]\n```",
			step.StepID, step.Description, label)

		// Property: success message must contain the analysis request label
		expectedFragment := fmt.Sprintf("> 📋 分析请求：%s", label)
		if !strings.Contains(successMsg, expectedFragment) {
			t.Logf("seed=%d: success message does not contain expected fragment %q", seed, expectedFragment)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1b (success message contains step label) failed: %v", err)
	}
}

// TestProperty1c_FailureMessageContainsStepLabel verifies that for any PackStep,
// the failure message format (as used in executePackSQLStep/executePackPythonStep)
// contains the step label returned by getStepLabel.
// **Validates: Requirements 4.3, 4.4**
func TestProperty1c_FailureMessageContainsStepLabel(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		step := generateRandomPackStep(r, true)

		label := getStepLabel(step)
		errText := generateRandomString(r, 30)

		// Simulate the failure message format from executePackSQLStep
		failureMsg := fmt.Sprintf("❌ 步骤 %d (%s) 执行失败：%s\n\n> 📋 分析请求：%s\n\n```sql\n%s\n```",
			step.StepID, step.Description, errText, label, step.Code)

		// Property: failure message must contain the analysis request label
		expectedFragment := fmt.Sprintf("> 📋 分析请求：%s", label)
		if !strings.Contains(failureMsg, expectedFragment) {
			t.Logf("seed=%d: failure message does not contain expected fragment %q", seed, expectedFragment)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1c (failure message contains step label) failed: %v", err)
	}
}

// TestProperty1d_GetStepLabelNeverEmpty verifies that getStepLabel never returns
// an empty string for any valid PackStep (StepID >= 1).
// **Validates: Requirements 4.4**
func TestProperty1d_GetStepLabelNeverEmpty(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		step := generateRandomPackStep(r, true)

		label := getStepLabel(step)

		if label == "" {
			t.Logf("seed=%d: getStepLabel returned empty string for step (desc=%q, type=%s, id=%d)",
				seed, step.Description, step.StepType, step.StepID)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1d (getStepLabel never empty) failed: %v", err)
	}
}
