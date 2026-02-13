package main

import (
	"testing"
)

func TestExtractStepDescriptionFromContent_AnalysisRequestLine(t *testing.T) {
	content := "✅ 步骤 1 (查询销售数据):\n\n> 📋 分析请求：查询2024年Q1销售数据\n\n```json:table\n[]\n```"
	got := extractStepDescriptionFromContent(content)
	if got != "查询2024年Q1销售数据" {
		t.Errorf("expected '查询2024年Q1销售数据', got '%s'", got)
	}
}

func TestExtractStepDescriptionFromContent_StepHeaderFallback(t *testing.T) {
	// No 📋 line, should fall back to step header
	content := "✅ 步骤 2 (分析用户行为):\n\n```json:table\n[]\n```"
	got := extractStepDescriptionFromContent(content)
	if got != "分析用户行为" {
		t.Errorf("expected '分析用户行为', got '%s'", got)
	}
}

func TestExtractStepDescriptionFromContent_EmptyContent(t *testing.T) {
	got := extractStepDescriptionFromContent("")
	if got != "" {
		t.Errorf("expected empty string, got '%s'", got)
	}
}

func TestExtractStepDescriptionFromContent_NoMatch(t *testing.T) {
	content := "Some random content without step info"
	got := extractStepDescriptionFromContent(content)
	if got != "" {
		t.Errorf("expected empty string, got '%s'", got)
	}
}

func TestExtractStepDescriptionFromContent_AnalysisRequestPriority(t *testing.T) {
	// Both formats present — 📋 line should take priority
	content := "✅ 步骤 3 (旧描述):\n\n> 📋 分析请求：新的分析请求描述\n\n```json:table\n[]\n```"
	got := extractStepDescriptionFromContent(content)
	if got != "新的分析请求描述" {
		t.Errorf("expected '新的分析请求描述', got '%s'", got)
	}
}
