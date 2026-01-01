package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProbePythonEnvironments(t *testing.T) {
	service := NewPythonService()
	envs := service.ProbePythonEnvironments()

	// We expect at least one system python (assuming dev environment has python)
	if len(envs) == 0 {
		t.Log("Warning: No Python environments found. This might be expected in some CI/sandbox environments, but unusual for a dev machine.")
	} else {
		t.Logf("Found %d Python environments", len(envs))
		for _, env := range envs {
			t.Logf("Path: %s, Version: %s, Type: %s", env.Path, env.Version, env.Type)
		}
	}
}

func TestValidatePythonEnvironment(t *testing.T) {
	service := NewPythonService()
	
	// Create a dummy file to test "invalid python"
	tmpDir, _ := os.MkdirTemp("", "test-python")
	defer os.RemoveAll(tmpDir)
	dummyPath := filepath.Join(tmpDir, "python")
	os.WriteFile(dummyPath, []byte("not python"), 0755)

	result := service.ValidatePythonEnvironment(dummyPath)
	if result.Valid {
		t.Error("Expected dummy file to be invalid python")
	}

	// Test non-existent
	result = service.ValidatePythonEnvironment("/non/existent/path")
	if result.Valid {
		t.Error("Expected non-existent path to be invalid")
	}
}
