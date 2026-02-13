package main

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"vantagedata/logger"
)

func newTestAppWithAggregator() *App {
	ctx := context.Background()
	ea := NewEventAggregator(ctx)
	return &App{
		eventAggregator: ea,
		logger:          logger.NewLogger(),
	}
}

// TestDetectAndSendPythonECharts_WithBacktickFence tests ECharts detection with ```json:echarts blocks.
func TestDetectAndSendPythonECharts_WithBacktickFence(t *testing.T) {
	app := newTestAppWithAggregator()

	output := "Some text\n```json:echarts\n{\"title\":{\"text\":\"Test\"},\"series\":[{\"type\":\"bar\",\"data\":[1,2,3]}]}\n```\nMore text"

	results := app.detectAndSendPythonECharts("thread1", "msg1", output, "")

	if len(results) != 1 {
		t.Fatalf("expected 1 ECharts result, got %d", len(results))
	}
	if results[0].Type != "echarts" {
		t.Errorf("expected type 'echarts', got '%s'", results[0].Type)
	}
}

// TestDetectAndSendPythonECharts_MultipleBlocks tests detection of multiple ECharts blocks.
func TestDetectAndSendPythonECharts_MultipleBlocks(t *testing.T) {
	app := newTestAppWithAggregator()

	output := "```json:echarts\n{\"title\":{\"text\":\"Chart1\"}}\n```\nSome text\n```json:echarts\n{\"title\":{\"text\":\"Chart2\"}}\n```"

	results := app.detectAndSendPythonECharts("thread1", "msg1", output, "")

	if len(results) != 2 {
		t.Fatalf("expected 2 ECharts results, got %d", len(results))
	}
}

// TestDetectAndSendPythonECharts_InvalidJSON tests that invalid JSON is skipped.
func TestDetectAndSendPythonECharts_InvalidJSON(t *testing.T) {
	app := newTestAppWithAggregator()

	output := "```json:echarts\nnot-valid-json\n```"

	results := app.detectAndSendPythonECharts("thread1", "msg1", output, "")

	if len(results) != 0 {
		t.Fatalf("expected 0 results for invalid JSON, got %d", len(results))
	}
}

// TestDetectAndSendPythonECharts_NoECharts tests output with no ECharts markers.
func TestDetectAndSendPythonECharts_NoECharts(t *testing.T) {
	app := newTestAppWithAggregator()

	output := "Just some regular Python output\nNo charts here"

	results := app.detectAndSendPythonECharts("thread1", "msg1", output, "")

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// TestDetectAndSendPythonChartFiles_WithImages tests image file detection from workDir.
func TestDetectAndSendPythonChartFiles_WithImages(t *testing.T) {
	app := newTestAppWithAggregator()

	// Create temp dir with test image files
	workDir := t.TempDir()
	testImageData := []byte("fake-png-data")
	os.WriteFile(filepath.Join(workDir, "chart.png"), testImageData, 0644)
	os.WriteFile(filepath.Join(workDir, "output.jpg"), testImageData, 0644)
	os.WriteFile(filepath.Join(workDir, "script.py"), []byte("print('hello')"), 0644) // non-image

	results := app.detectAndSendPythonChartFiles("thread1", "msg1", workDir, "")

	if len(results) != 2 {
		t.Fatalf("expected 2 image results, got %d", len(results))
	}
	for _, r := range results {
		if r.Type != "image" {
			t.Errorf("expected type 'image', got '%s'", r.Type)
		}
		// Verify base64 encoding
		data, ok := r.Data.(string)
		if !ok {
			t.Fatal("expected Data to be string")
		}
		if len(data) == 0 {
			t.Error("expected non-empty base64 data")
		}
		// Should start with data:image/png;base64,
		expectedPrefix := "data:image/png;base64,"
		if len(data) < len(expectedPrefix) || data[:len(expectedPrefix)] != expectedPrefix {
			t.Errorf("expected data to start with '%s'", expectedPrefix)
		}
		// Verify the base64 content decodes correctly
		b64Part := data[len(expectedPrefix):]
		decoded, err := base64.StdEncoding.DecodeString(b64Part)
		if err != nil {
			t.Errorf("failed to decode base64: %v", err)
		}
		if string(decoded) != string(testImageData) {
			t.Errorf("decoded data mismatch")
		}
	}
}

// TestDetectAndSendPythonChartFiles_EmptyDir tests with no image files.
func TestDetectAndSendPythonChartFiles_EmptyDir(t *testing.T) {
	app := newTestAppWithAggregator()

	workDir := t.TempDir()

	results := app.detectAndSendPythonChartFiles("thread1", "msg1", workDir, "")

	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty dir, got %d", len(results))
	}
}

// TestDetectAndSendPythonChartFiles_NonexistentDir tests with a nonexistent directory.
func TestDetectAndSendPythonChartFiles_NonexistentDir(t *testing.T) {
	app := newTestAppWithAggregator()

	results := app.detectAndSendPythonChartFiles("thread1", "msg1", "/nonexistent/path", "")

	if results != nil {
		t.Fatalf("expected nil results for nonexistent dir, got %d items", len(results))
	}
}
