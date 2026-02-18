package main

import (
	"testing"

	"vantagedata/config"
	"vantagedata/logger"
)

// TestApp_NewApp_ConfigServiceInitialized verifies that NewApp creates an App
// with a non-nil configService field.
func TestApp_NewApp_ConfigServiceInitialized(t *testing.T) {
	app := NewApp()
	if app.configService == nil {
		t.Fatal("expected configService to be non-nil after NewApp()")
	}
	if app.configService.Name() != "config" {
		t.Errorf("expected configService.Name() = %q, got %q", "config", app.configService.Name())
	}
}

// TestApp_GetConfig_DelegatesToConfigService verifies that App.GetConfig
// delegates to configService and returns a valid config.
func TestApp_GetConfig_DelegatesToConfigService(t *testing.T) {
	tmpDir := t.TempDir()
	l := logger.NewLogger()
	cs := NewConfigService(l.Log)
	cs.SetStorageDir(tmpDir)

	app := &App{
		configService: cs,
		logger:        l,
	}

	cfg, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	// Should return defaults since no config file exists
	if cfg.LLMProvider != "OpenAI" {
		t.Errorf("expected LLMProvider=OpenAI, got %q", cfg.LLMProvider)
	}
	if cfg.ModelName != "gpt-4o" {
		t.Errorf("expected ModelName=gpt-4o, got %q", cfg.ModelName)
	}
}

// TestApp_SaveConfig_DelegatesToConfigService verifies that App.SaveConfig
// delegates persistence to configService. We test this indirectly by saving
// via configService directly and verifying App.GetConfig reads it back,
// since App.SaveConfig has Wails runtime dependencies (updateWindowTitle etc.)
// that can't run in unit tests.
func TestApp_SaveConfig_DelegatesToConfigService(t *testing.T) {
	tmpDir := t.TempDir()
	l := logger.NewLogger()
	cs := NewConfigService(l.Log)
	cs.SetStorageDir(tmpDir)

	app := &App{
		configService: cs,
		logger:        l,
	}

	// Save via configService directly (simulating what App.SaveConfig delegates to)
	cfg := config.Config{
		LLMProvider:  "Anthropic",
		ModelName:    "claude-3",
		Language:     "zh-CN",
		DataCacheDir: tmpDir,
	}

	if err := cs.SaveConfig(cfg); err != nil {
		t.Fatalf("configService.SaveConfig failed: %v", err)
	}

	// Verify App.GetConfig reads back the config via configService delegation
	loaded, err := app.GetConfig()
	if err != nil {
		t.Fatalf("App.GetConfig failed: %v", err)
	}

	if loaded.LLMProvider != "Anthropic" {
		t.Errorf("expected LLMProvider=Anthropic, got %q", loaded.LLMProvider)
	}
	if loaded.ModelName != "claude-3" {
		t.Errorf("expected ModelName=claude-3, got %q", loaded.ModelName)
	}
}

// TestApp_GetEffectiveConfig_DelegatesToConfigService verifies that
// App.GetEffectiveConfig delegates to configService via GetConfig.
func TestApp_GetEffectiveConfig_DelegatesToConfigService(t *testing.T) {
	tmpDir := t.TempDir()
	l := logger.NewLogger()
	cs := NewConfigService(l.Log)
	cs.SetStorageDir(tmpDir)

	app := &App{
		configService: cs,
		logger:        l,
	}

	cfg, err := app.GetEffectiveConfig()
	if err != nil {
		t.Fatalf("GetEffectiveConfig failed: %v", err)
	}

	// Without license client, effective config should match base config
	if cfg.LLMProvider != "OpenAI" {
		t.Errorf("expected LLMProvider=OpenAI, got %q", cfg.LLMProvider)
	}
}

// TestApp_GetConfig_NilConfigService verifies graceful fallback when
// configService is nil (defensive case).
func TestApp_GetConfig_NilConfigService(t *testing.T) {
	app := &App{
		configService: nil,
		logger:        logger.NewLogger(),
	}

	cfg, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig with nil configService should not fail: %v", err)
	}

	// Should return a config (either defaults or from existing file) without panicking
	// The key assertion is that it doesn't panic and returns no error
	if cfg.MaxPreviewRows <= 0 {
		t.Errorf("expected MaxPreviewRows > 0, got %d", cfg.MaxPreviewRows)
	}
}

// TestApp_GetStorageDir_DelegatesToConfigService verifies that getStorageDir
// delegates to configService when available.
func TestApp_GetStorageDir_DelegatesToConfigService(t *testing.T) {
	tmpDir := t.TempDir()
	cs := NewConfigService(nil)
	cs.SetStorageDir(tmpDir)

	app := &App{
		configService: cs,
		logger:        logger.NewLogger(),
	}

	dir, err := app.getStorageDir()
	if err != nil {
		t.Fatalf("getStorageDir failed: %v", err)
	}

	if dir != tmpDir {
		t.Errorf("expected %q, got %q", tmpDir, dir)
	}
}

// TestApp_GetConfigPath_DelegatesToConfigService verifies that getConfigPath
// delegates to configService when available.
func TestApp_GetConfigPath_DelegatesToConfigService(t *testing.T) {
	tmpDir := t.TempDir()
	cs := NewConfigService(nil)
	cs.SetStorageDir(tmpDir)

	app := &App{
		configService: cs,
		logger:        logger.NewLogger(),
	}

	path, err := app.getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath failed: %v", err)
	}

	expected, _ := cs.GetConfigPath()
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}
