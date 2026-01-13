package agent

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// PythonWorker represents a single warm Python process
type PythonWorker struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	ready    bool
	lastUsed time.Time
	mu       sync.Mutex
}

// PythonPool manages a pool of warm Python processes
type PythonPool struct {
	workers    []*PythonWorker
	available  chan *PythonWorker
	pythonPath string
	poolSize   int
	mu         sync.Mutex
	closed     bool
}

// WorkerRequest is sent to the Python worker
type WorkerRequest struct {
	Code    string `json:"code"`
	WorkDir string `json:"workdir,omitempty"`
}

// WorkerResponse is received from the Python worker
type WorkerResponse struct {
	Status string `json:"status"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

// Python worker script that reads JSON requests from stdin
const pythonWorkerScript = `
import sys
import os
import json
import base64
import traceback
import io
from contextlib import redirect_stdout, redirect_stderr

# Pre-import common libraries
try:
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
    # Configure Chinese font support
    plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'DejaVu Sans', 'Arial Unicode MS']
    plt.rcParams['axes.unicode_minus'] = False  # Fix minus sign display
except:
    pass

try:
    import pandas as pd
except:
    pass

try:
    import numpy as np
except:
    pass

def execute_code(code, workdir=None):
    """Execute code and capture output"""
    # Change to work directory if specified
    original_dir = os.getcwd()
    if workdir:
        try:
            os.chdir(workdir)
        except:
            pass

    stdout_capture = io.StringIO()
    stderr_capture = io.StringIO()

    try:
        with redirect_stdout(stdout_capture), redirect_stderr(stderr_capture):
            exec(code, {'__name__': '__main__'})
        output = stdout_capture.getvalue() + stderr_capture.getvalue()
        return {"status": "ok", "output": output, "error": ""}
    except Exception as e:
        error_msg = traceback.format_exc()
        return {"status": "error", "output": stdout_capture.getvalue(), "error": error_msg}
    finally:
        # Restore original directory
        try:
            os.chdir(original_dir)
        except:
            pass

# Worker loop
while True:
    try:
        line = sys.stdin.readline()
        if not line:
            break

        request = json.loads(line.strip())
        code = base64.b64decode(request['code']).decode('utf-8')
        workdir = request.get('workdir', '')

        result = execute_code(code, workdir)
        print(json.dumps(result), flush=True)
    except Exception as e:
        print(json.dumps({"status": "error", "output": "", "error": str(e)}), flush=True)
`

// NewPythonPool creates a new pool with the specified size
func NewPythonPool(pythonPath string, size int) (*PythonPool, error) {
	if size <= 0 {
		size = 2 // Default pool size
	}

	pool := &PythonPool{
		workers:    make([]*PythonWorker, 0, size),
		available:  make(chan *PythonWorker, size),
		pythonPath: pythonPath,
		poolSize:   size,
	}

	// Start initial workers
	for i := 0; i < size; i++ {
		worker, err := pool.startWorker()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to start worker %d: %v", i, err)
		}
		pool.workers = append(pool.workers, worker)
		pool.available <- worker
	}

	// Start background cleanup goroutine
	go pool.maintenance()

	return pool, nil
}

// startWorker creates and starts a new Python worker process
func (p *PythonPool) startWorker() (*PythonWorker, error) {
	cmd := exec.Command(p.pythonPath, "-c", pythonWorkerScript)

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	// Set UTF-8 encoding
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start worker: %v", err)
	}

	worker := &PythonWorker{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   bufio.NewReader(stdout),
		ready:    true,
		lastUsed: time.Now(),
	}

	return worker, nil
}

// Execute runs Python code using a pooled worker
func (p *PythonPool) Execute(code string, workDir string) (string, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return "", fmt.Errorf("pool is closed")
	}
	p.mu.Unlock()

	// Get an available worker with timeout
	var worker *PythonWorker
	select {
	case worker = <-p.available:
		// Got a worker
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("timeout waiting for available worker")
	}

	defer func() {
		// Return worker to pool if still valid
		worker.mu.Lock()
		if worker.ready {
			worker.lastUsed = time.Now()
			worker.mu.Unlock()
			p.available <- worker
		} else {
			worker.mu.Unlock()
			// Worker is dead, try to create a new one
			go p.replaceWorker(worker)
		}
	}()

	return worker.execute(code, workDir)
}

// execute runs code on a specific worker
func (w *PythonWorker) execute(code string, workDir string) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.ready {
		return "", fmt.Errorf("worker not ready")
	}

	// Encode code to base64
	encodedCode := base64.StdEncoding.EncodeToString([]byte(code))
	request := WorkerRequest{Code: encodedCode, WorkDir: workDir}

	reqBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send request
	_, err = fmt.Fprintf(w.stdin, "%s\n", reqBytes)
	if err != nil {
		w.ready = false
		return "", fmt.Errorf("failed to send request: %v", err)
	}

	// Read response with timeout
	responseChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		line, err := w.stdout.ReadString('\n')
		if err != nil {
			errorChan <- err
			return
		}
		responseChan <- line
	}()

	select {
	case line := <-responseChan:
		var response WorkerResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			return "", fmt.Errorf("failed to parse response: %v", err)
		}

		if response.Status == "error" {
			return response.Output + "\n" + response.Error, fmt.Errorf("python error: %s", response.Error)
		}

		return response.Output, nil

	case err := <-errorChan:
		w.ready = false
		return "", fmt.Errorf("failed to read response: %v", err)

	case <-time.After(120 * time.Second):
		// Kill the hung process
		w.ready = false
		if w.cmd.Process != nil {
			w.cmd.Process.Kill()
		}
		return "", fmt.Errorf("execution timeout")
	}
}

// replaceWorker replaces a dead worker with a new one
func (p *PythonPool) replaceWorker(oldWorker *PythonWorker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	// Clean up old worker
	if oldWorker.cmd.Process != nil {
		oldWorker.cmd.Process.Kill()
	}

	// Create new worker
	newWorker, err := p.startWorker()
	if err != nil {
		// Log error but don't fail
		fmt.Printf("Failed to replace worker: %v\n", err)
		return
	}

	// Replace in workers slice
	for i, w := range p.workers {
		if w == oldWorker {
			p.workers[i] = newWorker
			break
		}
	}

	// Add to available channel
	p.available <- newWorker
}

// maintenance runs periodic cleanup
func (p *PythonPool) maintenance() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()

		// Could add worker health checks here
	}
}

// Close shuts down all workers
func (p *PythonPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	// Close all workers
	for _, worker := range p.workers {
		worker.mu.Lock()
		worker.ready = false
		if worker.stdin != nil {
			worker.stdin.Close()
		}
		if worker.cmd.Process != nil {
			worker.cmd.Process.Kill()
		}
		worker.mu.Unlock()
	}

	// Drain available channel
	close(p.available)
	for range p.available {
		// Drain
	}
}

// PooledPythonExecutor wraps a pool for use as PythonExecutor interface
type PooledPythonExecutor struct {
	pool *PythonPool
}

// NewPooledPythonExecutor creates a new pooled executor
func NewPooledPythonExecutor(pythonPath string) (*PooledPythonExecutor, error) {
	pool, err := NewPythonPool(pythonPath, 2)
	if err != nil {
		return nil, err
	}
	return &PooledPythonExecutor{pool: pool}, nil
}

// ExecuteScript implements PythonExecutor interface using the pool
func (e *PooledPythonExecutor) ExecuteScript(pythonPath string, script string) (string, error) {
	return e.pool.Execute(script, "")
}

// Close shuts down the executor's pool
func (e *PooledPythonExecutor) Close() {
	if e.pool != nil {
		e.pool.Close()
	}
}
