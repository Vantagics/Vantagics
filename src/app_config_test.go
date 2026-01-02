package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_DataCacheDir_Default(t *testing.T) {
	app := NewApp()
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-default")
	defer os.RemoveAll(tmpDir)
	app.storageDir = tmpDir

	config, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	home, _ := os.UserHomeDir()
	expectedDefault := filepath.Join(home, "RapidBI")
	
	// This will fail because DataCacheDir doesn't exist in Config yet
	if config.DataCacheDir != expectedDefault {
		t.Errorf("Expected default DataCacheDir %s, got %s", expectedDefault, config.DataCacheDir)
	}
}

func TestConfig_DataCacheDir_Validation(t *testing.T) {
	app := NewApp()
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-val")
	defer os.RemoveAll(tmpDir)
	app.storageDir = tmpDir

	config, _ := app.GetConfig()
	
	// Test non-existent directory
	config.DataCacheDir = "/non/existent/path/rapidbi_test_dir"
	err := app.SaveConfig(config)
	if err == nil {
		t.Error("Expected error for non-existent DataCacheDir, got nil")
	}

	// Test existing directory
	validDir := filepath.Join(tmpDir, "valid_dir")
	os.MkdirAll(validDir, 0755)
	config.DataCacheDir = validDir
	err = app.SaveConfig(config)
	if err != nil {
		t.Errorf("Expected no error for existing DataCacheDir, got %v", err)
	}
}

func TestConfig_PythonPath_Persistence(t *testing.T) {
	app := NewApp()
	tmpDir, _ := os.MkdirTemp("", "rapidbi-test-python-persist")
	defer os.RemoveAll(tmpDir)
	app.storageDir = tmpDir

	config, _ := app.GetConfig()
	config.PythonPath = "/usr/bin/python3"
	
	err := app.SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loadedConfig, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if loadedConfig.PythonPath != "/usr/bin/python3" {
		t.Errorf("Expected PythonPath /usr/bin/python3, got %s", loadedConfig.PythonPath)
	}
}
