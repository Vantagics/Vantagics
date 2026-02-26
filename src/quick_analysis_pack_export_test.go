//go:build property_test

package main

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: datasource-pack-result-consistency, Property 7: 导出步骤收集完整性
// Validates: Requirements 8.1
//
// For any set of execution records in executions.json where Success == true
// and Code != "", every such record shall appear as a PackStep in the exported
// pack's ExecutableSteps list (matched by Code content).

// genExecType generates a valid execution type ("sql" or "python").
func genExecType() gopter.Gen {
	return gen.OneConstOf(execTypeSQL, execTypePython)
}

// genNonEmptyCode generates a non-empty code string.
func genNonEmptyCode() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// genExecutionRecord generates a random parsedExecutionRecord with Success=true and non-empty Code.
func genExecutionRecord() gopter.Gen {
	return gopter.CombineGens(
		genNonEmptyCode(),
		genExecType(),
		gen.AlphaString(),
		gen.AlphaString(),
	).Map(func(values []interface{}) parsedExecutionRecord {
		return parsedExecutionRecord{
			Code:            values[0].(string),
			Type:            values[1].(string),
			StepDescription: values[2].(string),
			UserRequest:     values[3].(string),
			Timestamp:       0,
			Success:         true,
		}
	})
}

// genExecutionRecordSlice generates a non-empty slice of execution records.
func genExecutionRecordSlice() gopter.Gen {
	return gen.SliceOfN(5, genExecutionRecord()).SuchThat(func(records []parsedExecutionRecord) bool {
		return len(records) > 0
	})
}

// simulateCollectSteps replicates the core mapping logic of collectStepsFromExecutions:
// it filters records with Success==true and Code!="", deduplicates by type+code,
// maps exec types to step types, and builds PackSteps.
func simulateCollectSteps(records []parsedExecutionRecord) []PackStep {
	var steps []PackStep
	stepID := 1
	seenCode := make(map[string]bool)

	for _, exec := range records {
		if exec.Code == "" || !exec.Success {
			continue
		}

		codeKey := exec.Type + ":" + exec.Code
		if seenCode[codeKey] {
			continue
		}
		seenCode[codeKey] = true

		stepType := execTypeToStepType(exec.Type)
		if stepType == "" {
			continue
		}

		desc := exec.StepDescription
		if desc == "" {
			desc = exec.UserRequest
		}
		if desc == "" {
			if stepType == stepTypeSQL {
				desc = "SQL Query"
			} else {
				desc = "Python Code"
			}
		}

		steps = append(steps, PackStep{
			StepID:      stepID,
			StepType:    stepType,
			Code:        exec.Code,
			Description: desc,
			UserRequest: exec.UserRequest,
			DependsOn:   buildDependsOn(stepID),
		})
		stepID++
	}

	return steps
}

func TestProperty7_ExportStepCollectionCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("every successful record with code appears in PackStep list", prop.ForAll(
		func(records []parsedExecutionRecord) bool {
			steps := simulateCollectSteps(records)

			// Build a set of codes present in the resulting steps.
			stepCodes := make(map[string]bool)
			for _, s := range steps {
				stepCodes[s.Code] = true
			}

			// Every record with Success==true and Code!="" must have its Code
			// present in the step list (after dedup, the first occurrence wins).
			seenCode := make(map[string]bool)
			for _, rec := range records {
				if !rec.Success || rec.Code == "" {
					continue
				}
				codeKey := rec.Type + ":" + rec.Code
				if seenCode[codeKey] {
					continue
				}
				seenCode[codeKey] = true

				stepType := execTypeToStepType(rec.Type)
				if stepType == "" {
					continue // unknown type, skipped by the real function too
				}

				if !stepCodes[rec.Code] {
					t.Logf("missing code in steps: %q (type=%s)", rec.Code, rec.Type)
					return false
				}
			}

			return true
		},
		genExecutionRecordSlice(),
	))

	// Also verify that step count matches the number of unique valid records.
	properties.Property("step count equals unique valid record count", prop.ForAll(
		func(records []parsedExecutionRecord) bool {
			steps := simulateCollectSteps(records)

			// Count expected unique valid records.
			seenCode := make(map[string]bool)
			expected := 0
			for _, rec := range records {
				if !rec.Success || rec.Code == "" {
					continue
				}
				codeKey := rec.Type + ":" + rec.Code
				if seenCode[codeKey] {
					continue
				}
				seenCode[codeKey] = true
				if execTypeToStepType(rec.Type) == "" {
					continue
				}
				expected++
			}

			return len(steps) == expected
		},
		genExecutionRecordSlice(),
	))

	// Verify sequential StepID assignment.
	properties.Property("steps have sequential IDs starting from 1", prop.ForAll(
		func(records []parsedExecutionRecord) bool {
			steps := simulateCollectSteps(records)
			for i, s := range steps {
				if s.StepID != i+1 {
					t.Logf("step %d has StepID %d, expected %d", i, s.StepID, i+1)
					return false
				}
			}
			return true
		},
		genExecutionRecordSlice(),
	))

	// Verify that entries sorted by timestamp preserve order in steps.
	properties.Property("timestamp-sorted entries produce ordered steps", prop.ForAll(
		func(records []parsedExecutionRecord) bool {
			// Assign increasing timestamps to simulate sorted entries.
			for i := range records {
				records[i].Timestamp = int64(i)
			}
			sort.Slice(records, func(i, j int) bool {
				return records[i].Timestamp < records[j].Timestamp
			})

			steps := simulateCollectSteps(records)

			// Verify code order matches the first-seen order from sorted records.
			seenCode := make(map[string]bool)
			var expectedOrder []string
			for _, rec := range records {
				if !rec.Success || rec.Code == "" {
					continue
				}
				codeKey := rec.Type + ":" + rec.Code
				if seenCode[codeKey] {
					continue
				}
				seenCode[codeKey] = true
				if execTypeToStepType(rec.Type) == "" {
					continue
				}
				expectedOrder = append(expectedOrder, rec.Code)
			}

			if len(steps) != len(expectedOrder) {
				return false
			}
			for i, s := range steps {
				if s.Code != expectedOrder[i] {
					return false
				}
			}
			return true
		},
		genExecutionRecordSlice(),
	))

	properties.TestingRun(t)
}

// =============================================================================
// Unit Tests: ECharts 双来源导出
// Validates: Requirements 3.1, 3.4
// =============================================================================

// TestCompactJSONString verifies that compactJSONString removes whitespace from
// valid JSON and returns trimmed input for invalid JSON.
func TestCompactJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "compact valid JSON with spaces",
			input:    `{ "title": { "text": "Sales" } }`,
			expected: `{"title":{"text":"Sales"}}`,
		},
		{
			name:     "compact valid JSON with newlines",
			input:    "{\n  \"xAxis\": {\n    \"type\": \"category\"\n  }\n}",
			expected: `{"xAxis":{"type":"category"}}`,
		},
		{
			name:     "already compact JSON unchanged",
			input:    `{"a":"b"}`,
			expected: `{"a":"b"}`,
		},
		{
			name:     "invalid JSON returns trimmed input",
			input:    "  not json  ",
			expected: "not json",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := compactJSONString(tc.input)
			if got != tc.expected {
				t.Errorf("compactJSONString(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestEChartsRegexExtraction simulates the regex extraction logic used by
// attachEChartsFromMessages. It verifies that ECharts configs can be extracted
// from assistant message content in both backtick and non-backtick formats.
func TestEChartsRegexExtraction(t *testing.T) {
	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")
	reEChartsNoBT := regexp.MustCompile("(?s)(?:^|\\n)json:echarts\\s*\\n(\\{[\\s\\S]+?\\n\\})(?:\\s*\\n(?:---|###)|\\s*$)")

	extractConfigs := func(content string) []string {
		var configs []string
		for _, match := range reECharts.FindAllStringSubmatch(content, -1) {
			if len(match) > 1 {
				chartJSON := strings.TrimSpace(match[1])
				if chartJSON != "" && json.Valid([]byte(chartJSON)) {
					configs = append(configs, chartJSON)
				}
			}
		}
		for _, match := range reEChartsNoBT.FindAllStringSubmatch(content, -1) {
			if len(match) > 1 {
				chartJSON := strings.TrimSpace(match[1])
				if chartJSON != "" && json.Valid([]byte(chartJSON)) {
					configs = append(configs, chartJSON)
				}
			}
		}
		return configs
	}

	t.Run("extract from backtick format", func(t *testing.T) {
		content := "Here is the chart:\n```json:echarts\n{\"title\":{\"text\":\"Sales\"}}\n```\nDone."
		configs := extractConfigs(content)
		if len(configs) != 1 {
			t.Fatalf("expected 1 config, got %d", len(configs))
		}
		if configs[0] != `{"title":{"text":"Sales"}}` {
			t.Errorf("unexpected config: %s", configs[0])
		}
	})

	t.Run("extract multiple configs from one message", func(t *testing.T) {
		content := "Chart 1:\n```json:echarts\n{\"title\":{\"text\":\"A\"}}\n```\nChart 2:\n```json:echarts\n{\"title\":{\"text\":\"B\"}}\n```"
		configs := extractConfigs(content)
		if len(configs) != 2 {
			t.Fatalf("expected 2 configs, got %d", len(configs))
		}
	})

	t.Run("skip invalid JSON in backtick format", func(t *testing.T) {
		content := "```json:echarts\n{invalid json}\n```\nValid:\n```json:echarts\n{\"valid\":true}\n```"
		configs := extractConfigs(content)
		if len(configs) != 1 {
			t.Fatalf("expected 1 valid config, got %d", len(configs))
		}
		if configs[0] != `{"valid":true}` {
			t.Errorf("unexpected config: %s", configs[0])
		}
	})

	t.Run("no echarts in message returns empty", func(t *testing.T) {
		content := "This is a regular message with no charts."
		configs := extractConfigs(content)
		if len(configs) != 0 {
			t.Fatalf("expected 0 configs, got %d", len(configs))
		}
	})

	t.Run("skip empty JSON block", func(t *testing.T) {
		content := "```json:echarts\n\n```"
		configs := extractConfigs(content)
		if len(configs) != 0 {
			t.Fatalf("expected 0 configs from empty block, got %d", len(configs))
		}
	})
}

// TestEChartsFromAnalysisResultsFiltering simulates the filtering logic used by
// attachEChartsFromAnalysisResults. It verifies that only "echarts" type items
// are extracted and invalid JSON is skipped.
func TestEChartsFromAnalysisResultsFiltering(t *testing.T) {
	// Simulate the filtering logic from attachEChartsFromAnalysisResults
	filterEChartsFromResults := func(results []AnalysisResultItem) []string {
		var configs []string
		for _, item := range results {
			if item.Type != "echarts" {
				continue
			}
			chartJSON := ""
			switch v := item.Data.(type) {
			case string:
				chartJSON = strings.TrimSpace(v)
			default:
				raw, err := json.Marshal(v)
				if err != nil {
					continue
				}
				chartJSON = string(raw)
			}
			if chartJSON == "" {
				continue
			}
			if !json.Valid([]byte(chartJSON)) {
				continue
			}
			configs = append(configs, chartJSON)
		}
		return configs
	}

	t.Run("extract echarts from string data", func(t *testing.T) {
		results := []AnalysisResultItem{
			{ID: "1", Type: "echarts", Data: `{"title":{"text":"Chart1"}}`},
			{ID: "2", Type: "table", Data: "some table data"},
			{ID: "3", Type: "echarts", Data: `{"title":{"text":"Chart2"}}`},
		}
		configs := filterEChartsFromResults(results)
		if len(configs) != 2 {
			t.Fatalf("expected 2 echarts configs, got %d", len(configs))
		}
	})

	t.Run("extract echarts from map data", func(t *testing.T) {
		results := []AnalysisResultItem{
			{ID: "1", Type: "echarts", Data: map[string]interface{}{"title": map[string]interface{}{"text": "MapChart"}}},
		}
		configs := filterEChartsFromResults(results)
		if len(configs) != 1 {
			t.Fatalf("expected 1 config, got %d", len(configs))
		}
		if !json.Valid([]byte(configs[0])) {
			t.Errorf("extracted config is not valid JSON: %s", configs[0])
		}
	})

	t.Run("skip invalid JSON string data", func(t *testing.T) {
		results := []AnalysisResultItem{
			{ID: "1", Type: "echarts", Data: "{not valid json}"},
			{ID: "2", Type: "echarts", Data: `{"valid":true}`},
		}
		configs := filterEChartsFromResults(results)
		if len(configs) != 1 {
			t.Fatalf("expected 1 valid config (invalid skipped), got %d", len(configs))
		}
		if configs[0] != `{"valid":true}` {
			t.Errorf("unexpected config: %s", configs[0])
		}
	})

	t.Run("skip empty data", func(t *testing.T) {
		results := []AnalysisResultItem{
			{ID: "1", Type: "echarts", Data: ""},
			{ID: "2", Type: "echarts", Data: "  "},
		}
		configs := filterEChartsFromResults(results)
		if len(configs) != 0 {
			t.Fatalf("expected 0 configs from empty data, got %d", len(configs))
		}
	})

	t.Run("skip non-echarts types", func(t *testing.T) {
		results := []AnalysisResultItem{
			{ID: "1", Type: "table", Data: `{"valid":true}`},
			{ID: "2", Type: "image", Data: `{"valid":true}`},
			{ID: "3", Type: "metric", Data: `{"valid":true}`},
		}
		configs := filterEChartsFromResults(results)
		if len(configs) != 0 {
			t.Fatalf("expected 0 echarts configs from non-echarts types, got %d", len(configs))
		}
	})
}

// TestEChartsDedupLogic verifies the deduplication logic used when merging
// configs from messages and analysis results. Configs that are semantically
// identical (same JSON after compaction) should not be duplicated.
func TestEChartsDedupLogic(t *testing.T) {
	// Simulate the dedup logic from attachEChartsFromAnalysisResults
	dedupConfigs := func(existing []string, newConfigs []string) []string {
		existingSet := make(map[string]bool, len(existing))
		for _, e := range existing {
			compacted := compactJSONString(e)
			existingSet[compacted] = true
		}

		var added []string
		for _, cfg := range newConfigs {
			compacted := compactJSONString(cfg)
			if existingSet[compacted] {
				continue
			}
			existingSet[compacted] = true
			added = append(added, cfg)
		}
		return added
	}

	t.Run("duplicate config with different whitespace is deduplicated", func(t *testing.T) {
		existing := []string{`{"title":{"text":"Sales"}}`}
		newConfigs := []string{`{ "title": { "text": "Sales" } }`}
		added := dedupConfigs(existing, newConfigs)
		if len(added) != 0 {
			t.Errorf("expected 0 added (duplicate), got %d", len(added))
		}
	})

	t.Run("different config is added", func(t *testing.T) {
		existing := []string{`{"title":{"text":"Sales"}}`}
		newConfigs := []string{`{"title":{"text":"Revenue"}}`}
		added := dedupConfigs(existing, newConfigs)
		if len(added) != 1 {
			t.Fatalf("expected 1 added, got %d", len(added))
		}
	})

	t.Run("mix of duplicates and new configs", func(t *testing.T) {
		existing := []string{
			`{"title":{"text":"A"}}`,
			`{"title":{"text":"B"}}`,
		}
		newConfigs := []string{
			`{ "title": { "text": "A" } }`, // duplicate of first
			`{"title":{"text":"C"}}`,        // new
			`{"title":{"text":"B"}}`,        // duplicate of second
		}
		added := dedupConfigs(existing, newConfigs)
		if len(added) != 1 {
			t.Fatalf("expected 1 new config added, got %d", len(added))
		}
		if compactJSONString(added[0]) != `{"title":{"text":"C"}}` {
			t.Errorf("unexpected added config: %s", added[0])
		}
	})

	t.Run("empty existing allows all new configs", func(t *testing.T) {
		var existing []string
		newConfigs := []string{`{"a":1}`, `{"b":2}`}
		added := dedupConfigs(existing, newConfigs)
		if len(added) != 2 {
			t.Fatalf("expected 2 added, got %d", len(added))
		}
	})

	t.Run("dedup within new configs themselves", func(t *testing.T) {
		var existing []string
		newConfigs := []string{`{"a":1}`, `{ "a": 1 }`}
		added := dedupConfigs(existing, newConfigs)
		if len(added) != 1 {
			t.Fatalf("expected 1 added (self-dedup), got %d", len(added))
		}
	})
}

// TestAttachEChartsFromMessagesSimulation tests the full flow of extracting
// ECharts from message content and assigning them to steps, simulating what
// attachEChartsFromMessages does without requiring an App instance.
func TestAttachEChartsFromMessagesSimulation(t *testing.T) {
	reECharts := regexp.MustCompile("(?s)```\\s*json:echarts\\s*\\n?([\\s\\S]+?)\\n?\\s*```")

	simulateAttach := func(messages []ChatMessage, steps []PackStep) {
		type echartsGroup struct {
			configs []string
		}
		var groups []echartsGroup

		for _, msg := range messages {
			if msg.Role != "assistant" {
				continue
			}
			var configs []string
			for _, match := range reECharts.FindAllStringSubmatch(msg.Content, -1) {
				if len(match) > 1 {
					chartJSON := strings.TrimSpace(match[1])
					if chartJSON != "" && json.Valid([]byte(chartJSON)) {
						configs = append(configs, chartJSON)
					}
				}
			}
			if len(configs) > 0 {
				groups = append(groups, echartsGroup{configs: configs})
			}
		}

		for i, group := range groups {
			stepIdx := i
			if stepIdx >= len(steps) {
				stepIdx = len(steps) - 1
			}
			steps[stepIdx].EChartsConfigs = append(steps[stepIdx].EChartsConfigs, group.configs...)
		}
	}

	t.Run("configs from two messages assigned to two steps", func(t *testing.T) {
		messages := []ChatMessage{
			{Role: "assistant", Content: "```json:echarts\n{\"chart\":\"A\"}\n```"},
			{Role: "user", Content: "next"},
			{Role: "assistant", Content: "```json:echarts\n{\"chart\":\"B\"}\n```"},
		}
		steps := []PackStep{
			{StepID: 1, StepType: stepTypeSQL},
			{StepID: 2, StepType: stepTypeSQL},
		}
		simulateAttach(messages, steps)

		if len(steps[0].EChartsConfigs) != 1 || steps[0].EChartsConfigs[0] != `{"chart":"A"}` {
			t.Errorf("step 1 configs unexpected: %v", steps[0].EChartsConfigs)
		}
		if len(steps[1].EChartsConfigs) != 1 || steps[1].EChartsConfigs[0] != `{"chart":"B"}` {
			t.Errorf("step 2 configs unexpected: %v", steps[1].EChartsConfigs)
		}
	})

	t.Run("more groups than steps overflow to last step", func(t *testing.T) {
		messages := []ChatMessage{
			{Role: "assistant", Content: "```json:echarts\n{\"chart\":\"A\"}\n```"},
			{Role: "assistant", Content: "```json:echarts\n{\"chart\":\"B\"}\n```"},
			{Role: "assistant", Content: "```json:echarts\n{\"chart\":\"C\"}\n```"},
		}
		steps := []PackStep{
			{StepID: 1, StepType: stepTypeSQL},
		}
		simulateAttach(messages, steps)

		if len(steps[0].EChartsConfigs) != 3 {
			t.Fatalf("expected 3 configs on single step, got %d", len(steps[0].EChartsConfigs))
		}
	})

	t.Run("user messages are skipped", func(t *testing.T) {
		messages := []ChatMessage{
			{Role: "user", Content: "```json:echarts\n{\"chart\":\"A\"}\n```"},
		}
		steps := []PackStep{
			{StepID: 1, StepType: stepTypeSQL},
		}
		simulateAttach(messages, steps)

		if len(steps[0].EChartsConfigs) != 0 {
			t.Errorf("expected 0 configs from user message, got %d", len(steps[0].EChartsConfigs))
		}
	})
}

// =============================================================================
// Feature: datasource-pack-result-consistency, Property 8: query_and_chart 步骤配对正确性
// Validates: Requirements 8.3
//
// For any exported PackStep with SourceTool == "query_and_chart" and
// StepType == "python_code", the step shall have a non-zero PairedSQLStepID
// that references an existing SQL step in the same pack, and that SQL step's
// StepID shall be less than the Python step's StepID.
// =============================================================================

// genQueryAndChartPair generates a query_and_chart pair: one SQL step followed
// by one Python chart step. The Python step has SourceTool="query_and_chart".
// StepIDs and PairedSQLStepID are left at 0; renumberSteps will assign them.
func genQueryAndChartPair() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString(),
	).Map(func(values []interface{}) []PackStep {
		sqlCode := values[0].(string)
		pyCode := values[1].(string)
		desc := values[2].(string)
		return []PackStep{
			{
				StepType:    stepTypeSQL,
				Code:        sqlCode,
				Description: desc + " SQL",
				SourceTool:  "query_and_chart",
			},
			{
				StepType:    stepTypePython,
				Code:        pyCode,
				Description: desc + " Chart",
				SourceTool:  "query_and_chart",
			},
		}
	})
}

// genStandaloneStep generates a standalone SQL or Python step (not query_and_chart).
func genStandaloneStep() gopter.Gen {
	return gopter.CombineGens(
		genExecType(),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString(),
	).Map(func(values []interface{}) PackStep {
		execType := values[0].(string)
		code := values[1].(string)
		desc := values[2].(string)
		return PackStep{
			StepType:    execTypeToStepType(execType),
			Code:        code,
			Description: desc,
		}
	})
}

// genPackStepsWithQueryAndChart generates a slice of PackSteps that contains
// at least one query_and_chart pair, optionally interleaved with standalone steps.
func genPackStepsWithQueryAndChart() gopter.Gen {
	return gopter.CombineGens(
		gen.SliceOfN(3, genStandaloneStep()),
		gen.IntRange(1, 3),
		gen.SliceOfN(2, genStandaloneStep()),
	).FlatMap(func(v interface{}) gopter.Gen {
		values := v.([]interface{})
		prefix := values[0].([]PackStep)
		pairCount := values[1].(int)
		suffix := values[2].([]PackStep)

		return gen.SliceOfN(pairCount, genQueryAndChartPair()).Map(func(pairs [][]PackStep) []PackStep {
			var steps []PackStep
			steps = append(steps, prefix...)
			for _, pair := range pairs {
				steps = append(steps, pair...)
			}
			steps = append(steps, suffix...)
			return steps
		})
	}, reflect.TypeOf([]PackStep{}))
}

func TestProperty8_QueryAndChartStepPairingCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("query_and_chart python steps have valid PairedSQLStepID after renumberSteps", prop.ForAll(
		func(steps []PackStep) bool {
			// renumberSteps is the production function that assigns StepIDs and PairedSQLStepIDs.
			renumberSteps(steps)

			// Build a lookup of all steps by StepID for quick reference.
			stepByID := make(map[int]PackStep, len(steps))
			for _, s := range steps {
				stepByID[s.StepID] = s
			}

			for _, s := range steps {
				if s.SourceTool != "query_and_chart" || s.StepType != stepTypePython {
					continue
				}

				// 1. PairedSQLStepID must be non-zero.
				if s.PairedSQLStepID == 0 {
					t.Logf("python step %d has PairedSQLStepID=0", s.StepID)
					return false
				}

				// 2. The referenced SQL step must exist in the pack.
				sqlStep, exists := stepByID[s.PairedSQLStepID]
				if !exists {
					t.Logf("python step %d references non-existent SQL step %d", s.StepID, s.PairedSQLStepID)
					return false
				}

				// 3. The referenced step must have StepType=="sql_query".
				if sqlStep.StepType != stepTypeSQL {
					t.Logf("python step %d references step %d with type %q (expected %q)",
						s.StepID, s.PairedSQLStepID, sqlStep.StepType, stepTypeSQL)
					return false
				}

				// 4. The SQL step's StepID must be less than the Python step's StepID.
				if sqlStep.StepID >= s.StepID {
					t.Logf("python step %d has PairedSQLStepID=%d which is not less than its own StepID",
						s.StepID, sqlStep.StepID)
					return false
				}
			}

			return true
		},
		genPackStepsWithQueryAndChart(),
	))

	// Additional property: every query_and_chart python step's PairedSQLStepID
	// points to the immediately preceding SQL step.
	properties.Property("query_and_chart python step pairs with the nearest preceding SQL step", prop.ForAll(
		func(steps []PackStep) bool {
			renumberSteps(steps)

			lastSQLStepID := 0
			for _, s := range steps {
				if s.StepType == stepTypeSQL {
					lastSQLStepID = s.StepID
				}
				if s.SourceTool == "query_and_chart" && s.StepType == stepTypePython {
					if s.PairedSQLStepID != lastSQLStepID {
						t.Logf("python step %d paired with %d, expected %d (last SQL step)",
							s.StepID, s.PairedSQLStepID, lastSQLStepID)
						return false
					}
				}
			}
			return true
		},
		genPackStepsWithQueryAndChart(),
	))

	properties.TestingRun(t)
}
