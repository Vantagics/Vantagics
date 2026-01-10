package agent

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"rapidbi/config"
	"strings"
	"testing"
)

type mockPythonExecutor struct {
	lastScript string
	err        error
	output     string
	onExecute  func(script string)
}

func (m *mockPythonExecutor) ExecuteScript(pythonPath string, script string) (string, error) {
	m.lastScript = script
	if m.onExecute != nil {
		m.onExecute(script)
	}
	return m.output, m.err
}

func TestPythonExecutorTool_InvokableRun(t *testing.T) {
	mockExec := &mockPythonExecutor{
		output: "Test Output",
	}
	cfg := config.Config{
		PythonPath: "python",
	}
	pyTool := &PythonExecutorTool{
		pythonService: mockExec,
		cfg:           cfg,
	}

	input := `{"code": "print('hello')"}`
	ctx := context.Background()
	
	resp, err := pyTool.InvokableRun(ctx, input)
	if err != nil {
		t.Fatalf("InvokableRun failed: %v", err)
	}

	if !strings.Contains(resp, "Test Output") {
		t.Errorf("Expected output 'Test Output', got '%s'", resp)
	}

	if !strings.Contains(mockExec.lastScript, "print('hello')") {
		t.Errorf("Script did not contain code: %s", mockExec.lastScript)
	}
}

func TestPythonExecutorTool_ChartDetection(t *testing.T) {
	// To test chart detection, we need mockExec to actually create a chart.png in the workDir.
	// But tool creates a random workDir.
	// We'll use onExecute to find the workDir from the script.
	
	mockExec := &mockPythonExecutor{
		output: "Chart generated",
	}
	
mockExec.onExecute = func(script string) {
		// Extract workDir from script: os.chdir(r'/tmp/...')
		// This is a bit hacky but works for testing tool logic
		lines := strings.Split(script, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "os.chdir(r'") {
				path := strings.TrimPrefix(line, "os.chdir(r'")
				path = strings.TrimSuffix(path, "')")
				
				// Create a fake chart.png there
				chartPath := filepath.Join(path, "chart.png")
				os.WriteFile(chartPath, []byte("fake-png-data"), 0644)
				break
			}
		}
	}

	cfg := config.Config{
		PythonPath: "python",
	}
	pyTool := &PythonExecutorTool{
		pythonService: mockExec,
		cfg:           cfg,
	}

	input := `{"code": "plt.plot()"}`
	resp, err := pyTool.InvokableRun(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(resp, "data:image/png;base64,") {
		t.Error("Response should contain base64 image link")
	}
	
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("fake-png-data"))
	if !strings.Contains(resp, expectedB64) {
		t.Error("Response should contain encoded fake-png-data")
	}
}
