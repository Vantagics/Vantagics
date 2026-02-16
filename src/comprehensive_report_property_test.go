package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

func generateNonEmptyString(r *rand.Rand, maxLen int) string {
	n := r.Intn(maxLen) + 1
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(r.Intn(94) + 32)
	}
	return string(buf)
}

// Feature: quick-analysis-report, Property 2: Pack 元数据包含
// **Validates: Requirements 1.7**
func TestPropertyPackMetadataInclusion(t *testing.T) {
	config := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		author := generateNonEmptyString(r, 50)
		description := generateNonEmptyString(r, 100)
		sourceName := generateNonEmptyString(r, 50)
		meta := &PackMetadata{Author: author, Description: description, SourceName: sourceName, CreatedAt: time.Now().Format(time.RFC3339)}
		dsName := generateNonEmptyString(r, 30)
		sessionName := generateNonEmptyString(r, 30)
		contents := []string{generateNonEmptyString(r, 200)}
		result := buildComprehensiveSummary(dsName, sessionName, contents, meta)
		return strings.Contains(result, author) && strings.Contains(result, description) && strings.Contains(result, sourceName)
	}
	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 2 failed: %v", err)
	}
}

// Feature: quick-analysis-report, Property 3: 摘要构建完整性
// **Validates: Requirements 2.1**
func TestPropertySummaryBuildCompleteness(t *testing.T) {
	config := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		dsName := generateNonEmptyString(r, 50)
		sessionName := generateNonEmptyString(r, 50)
		numContents := r.Intn(10) + 1
		contents := make([]string, numContents)
		for i := range contents {
			contents[i] = generateNonEmptyString(r, 200)
		}
		result := buildComprehensiveSummary(dsName, sessionName, contents, nil)
		if !strings.Contains(result, dsName) || !strings.Contains(result, sessionName) {
			return false
		}
		for _, c := range contents {
			if !strings.Contains(result, c) {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 3 failed: %v", err)
	}
}

func generateAnalysisContentItem(r *rand.Rand) string {
	contentTypes := []string{"request", "response", "echarts_response", "table", "insight", "metric", "csv"}
	ct := contentTypes[r.Intn(len(contentTypes))]
	switch ct {
	case "request":
		return fmt.Sprintf("### Analysis Request\n%s", generateNonEmptyString(r, 100))
	case "response":
		return fmt.Sprintf("### Analysis Result\n%s", generateNonEmptyString(r, 200))
	case "echarts_response":
		return fmt.Sprintf("### Analysis Result\n%s [图表]", generateNonEmptyString(r, 150))
	case "table":
		return fmt.Sprintf("### Data Table\n%s: %s (%d rows)", generateNonEmptyString(r, 30), generateNonEmptyString(r, 50), r.Intn(100)+1)
	case "insight":
		return fmt.Sprintf("### Insight\n%s", generateNonEmptyString(r, 100))
	case "metric":
		return fmt.Sprintf("### Key Metric\n%s: %s", generateNonEmptyString(r, 30), generateNonEmptyString(r, 20))
	case "csv":
		return fmt.Sprintf("### Data Table\n%s", generateNonEmptyString(r, 150))
	default:
		return generateNonEmptyString(r, 100)
	}
}

// Feature: quick-analysis-report, Property 1: 数据收集完整性
// **Validates: Requirements 1.2, 1.3, 1.4, 1.5, 1.6**
func TestPropertyDataCollectionCompleteness(t *testing.T) {
	config := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		dsName := generateNonEmptyString(r, 50)
		sessionName := generateNonEmptyString(r, 50)
		numContents := r.Intn(20) + 1
		contents := make([]string, numContents)
		for i := range contents {
			contents[i] = generateAnalysisContentItem(r)
		}
		result := buildComprehensiveSummary(dsName, sessionName, contents, nil)
		if !strings.Contains(result, dsName) || !strings.Contains(result, sessionName) {
			return false
		}
		for _, c := range contents {
			if !strings.Contains(result, c) {
				return false
			}
		}
		meta := &PackMetadata{Author: generateNonEmptyString(r, 30), Description: generateNonEmptyString(r, 60), SourceName: generateNonEmptyString(r, 30), CreatedAt: time.Now().Format(time.RFC3339)}
		resultWithMeta := buildComprehensiveSummary(dsName, sessionName, contents, meta)
		for _, c := range contents {
			if !strings.Contains(resultWithMeta, c) {
				return false
			}
		}
		return strings.Contains(resultWithMeta, meta.Author) && strings.Contains(resultWithMeta, meta.Description) && strings.Contains(resultWithMeta, meta.SourceName)
	}
	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1 failed: %v", err)
	}
}

// Feature: quick-analysis-report, Property 7: 内容哈希一致性
// **Validates: Requirements 3.2**
func TestPropertyContentHashConsistency(t *testing.T) {
	config := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
	t.Run("deterministic", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			numContents := r.Intn(10) + 1
			contents := make([]string, numContents)
			for i := range contents {
				contents[i] = generateNonEmptyString(r, 200)
			}
			tableCount := r.Intn(50)
			return computeAnalysisHash(contents, tableCount) == computeAnalysisHash(contents, tableCount)
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 7 determinism failed: %v", err)
		}
	})
	t.Run("different_contents_different_hash", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			numA := r.Intn(10) + 1
			contentsA := make([]string, numA)
			for i := range contentsA {
				contentsA[i] = generateNonEmptyString(r, 200)
			}
			numB := r.Intn(10) + 1
			contentsB := make([]string, numB)
			for i := range contentsB {
				contentsB[i] = generateNonEmptyString(r, 200)
			}
			if slicesEqual(contentsA, contentsB) {
				contentsB = append(contentsB, "extra_unique_element")
			}
			tableCount := r.Intn(50)
			return computeAnalysisHash(contentsA, tableCount) != computeAnalysisHash(contentsB, tableCount)
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 7 different-contents discrimination failed: %v", err)
		}
	})
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Feature: quick-analysis-report, Property 8: 导出文件名格式
// **Validates: Requirements 4.4**
func TestPropertyExportFileNameFormat(t *testing.T) {
	illegalChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	config := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
	t.Run("no_illegal_chars_in_output", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			length := r.Intn(100) + 1
			buf := make([]byte, length)
			candidates := "abcdefghijklmnopqrstuvwxyz0123456789 /\\:*?\"<>|_-.()"
			for i := range buf {
				buf[i] = candidates[r.Intn(len(candidates))]
			}
			input := string(buf)
			result := sanitizeFileName(input)
			for _, ch := range illegalChars {
				if strings.Contains(result, ch) {
					t.Logf("seed=%d: output %q contains illegal char %q (input=%q)", seed, result, ch, input)
					return false
				}
			}
			return true
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 8 (no illegal chars) failed: %v", err)
		}
	})
	t.Run("identity_for_clean_input", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			safeChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-.()"
			length := r.Intn(50) + 1
			buf := make([]byte, length)
			for i := range buf {
				buf[i] = safeChars[r.Intn(len(safeChars))]
			}
			input := string(buf)
			result := sanitizeFileName(input)
			if result != input {
				t.Logf("seed=%d: clean input %q was modified to %q", seed, input, result)
				return false
			}
			return true
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 8 (identity for clean input) failed: %v", err)
		}
	})
}
