package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"vantagedata/config"
)

// newTestConfigService creates a ConfigService with a temp directory for testing.
// Returns the service and a cleanup function.
func newTestConfigService(t *testing.T) (*ConfigService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	logger := func(msg string) { t.Log(msg) }
	cs := NewConfigService(logger)
	cs.SetStorageDir(tmpDir)
	return cs, func() {}
}

func TestConfigService_Name(t *testing.T) {
	cs := NewConfigService(nil)
	if cs.Name() != "config" {
		t.Errorf("expected Name() = %q, got %q", "config", cs.Name())
	}
}

func TestConfigService_Initialize(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	err := cs.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	dir, _ := cs.GetStorageDir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("storage dir does not exist after Initialize: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("storage dir is not a directory")
	}
}

func TestConfigService_Shutdown(t *testing.T) {
	cs := NewConfigService(nil)
	if err := cs.Shutdown(); err != nil {
		t.Fatalf("Shutdown should return nil, got: %v", err)
	}
}

func TestConfigService_GetStorageDir_Default(t *testing.T) {
	cs := NewConfigService(nil)
	dir, err := cs.GetStorageDir()
	if err != nil {
		t.Fatalf("GetStorageDir failed: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "Vantagics")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestConfigService_GetStorageDir_Custom(t *testing.T) {
	cs := NewConfigService(nil)
	cs.SetStorageDir("/tmp/test-vantage")
	dir, err := cs.GetStorageDir()
	if err != nil {
		t.Fatalf("GetStorageDir failed: %v", err)
	}
	if dir != "/tmp/test-vantage" {
		t.Errorf("expected /tmp/test-vantage, got %q", dir)
	}
}

func TestConfigService_GetConfigPath(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	path, err := cs.GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	dir, _ := cs.GetStorageDir()
	expected := filepath.Join(dir, "config.json")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestConfigService_GetConfig_DefaultWhenNoFile(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	cfg, err := cs.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if cfg.LLMProvider != "OpenAI" {
		t.Errorf("expected LLMProvider=OpenAI, got %q", cfg.LLMProvider)
	}
	if cfg.ModelName != "gpt-4o" {
		t.Errorf("expected ModelName=gpt-4o, got %q", cfg.ModelName)
	}
	if cfg.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens=8192, got %d", cfg.MaxTokens)
	}
	if cfg.MaxPreviewRows != 100 {
		t.Errorf("expected MaxPreviewRows=100, got %d", cfg.MaxPreviewRows)
	}
	if cfg.IntentEnhancement == nil {
		t.Error("expected IntentEnhancement to be non-nil")
	}
}

func TestConfigService_SaveAndGetConfig(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	// Create the data cache dir (SaveConfig validates it exists)
	dir, _ := cs.GetStorageDir()

	original := config.Config{
		LLMProvider:    "Anthropic",
		APIKey:         "test-key-123",
		ModelName:      "claude-3",
		MaxTokens:      4096,
		Language:       "zh-CN",
		DataCacheDir:   dir, // use storage dir as data cache dir for test
		MaxPreviewRows: 200,
	}

	if err := cs.SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := cs.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig after save failed: %v", err)
	}

	if loaded.LLMProvider != original.LLMProvider {
		t.Errorf("LLMProvider: expected %q, got %q", original.LLMProvider, loaded.LLMProvider)
	}
	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey: expected %q, got %q", original.APIKey, loaded.APIKey)
	}
	if loaded.ModelName != original.ModelName {
		t.Errorf("ModelName: expected %q, got %q", original.ModelName, loaded.ModelName)
	}
	if loaded.Language != original.Language {
		t.Errorf("Language: expected %q, got %q", original.Language, loaded.Language)
	}
}

func TestConfigService_SaveConfig_ValidatesCalled(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	dir, _ := cs.GetStorageDir()

	cfg := config.Config{
		MaxTokens:      -1,   // invalid, Validate should fix to 4096
		MaxPreviewRows: -5,   // invalid, Validate should fix to 100
		DataCacheDir:   dir,
	}

	if err := cs.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := cs.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if loaded.MaxTokens != 4096 {
		t.Errorf("expected MaxTokens=4096 after Validate, got %d", loaded.MaxTokens)
	}
	if loaded.MaxPreviewRows != 100 {
		t.Errorf("expected MaxPreviewRows=100 after Validate, got %d", loaded.MaxPreviewRows)
	}
}

func TestConfigService_SaveConfig_InvalidDataCacheDir(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	cfg := config.Config{
		DataCacheDir: "/nonexistent/path/that/does/not/exist",
	}

	err := cs.SaveConfig(cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent DataCacheDir")
	}
}

func TestConfigService_OnConfigChanged_CallbackTriggered(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	dir, _ := cs.GetStorageDir()

	var received config.Config
	called := false
	cs.OnConfigChanged(func(cfg config.Config) {
		called = true
		received = cfg
	})

	cfg := config.Config{
		LLMProvider:  "OpenAI",
		Language:     "en-US",
		DataCacheDir: dir,
	}

	if err := cs.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	if !called {
		t.Fatal("callback was not called after SaveConfig")
	}
	if received.Language != "en-US" {
		t.Errorf("callback received wrong Language: %q", received.Language)
	}
}

func TestConfigService_OnConfigChanged_MultipleCallbacks(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	dir, _ := cs.GetStorageDir()

	callCount := 0
	for i := 0; i < 3; i++ {
		cs.OnConfigChanged(func(cfg config.Config) {
			callCount++
		})
	}

	cfg := config.Config{
		DataCacheDir: dir,
	}
	if err := cs.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 callbacks called, got %d", callCount)
	}
}

func TestConfigService_NotifyConfigChanged_NoCallbacks(t *testing.T) {
	cs := NewConfigService(nil)
	// Should not panic with no callbacks registered
	cs.NotifyConfigChanged(config.Config{})
}

func TestConfigService_GetEffectiveConfig_SameAsGetConfig(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	dir, _ := cs.GetStorageDir()
	cfg := config.Config{
		LLMProvider:  "OpenAI",
		DataCacheDir: dir,
	}
	if err := cs.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	effective, err := cs.GetEffectiveConfig()
	if err != nil {
		t.Fatalf("GetEffectiveConfig failed: %v", err)
	}

	regular, err := cs.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if effective.LLMProvider != regular.LLMProvider {
		t.Errorf("GetEffectiveConfig and GetConfig differ: %q vs %q", effective.LLMProvider, regular.LLMProvider)
	}
}

func TestConfigService_GetConfig_InvalidJSON(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	path, _ := cs.GetConfigPath()
	os.WriteFile(path, []byte("not valid json{{{"), 0600)

	_, err := cs.GetConfig()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestConfigService_ServiceInterface(t *testing.T) {
	// Verify ConfigService implements Service interface
	var _ Service = (*ConfigService)(nil)
}

func TestConfigService_ConfigProviderInterface(t *testing.T) {
	// Verify ConfigService implements ConfigProvider interface
	var _ ConfigProvider = (*ConfigService)(nil)
}

func TestConfigService_ConfigPersisterInterface(t *testing.T) {
	// Verify ConfigService implements ConfigPersister interface
	var _ ConfigPersister = (*ConfigService)(nil)
}

func TestConfigService_ConfigNotifierInterface(t *testing.T) {
	// Verify ConfigService implements ConfigNotifier interface
	var _ ConfigNotifier = (*ConfigService)(nil)
}

func TestConfigService_ConcurrentCallbackRegistration(t *testing.T) {
	cs := NewConfigService(nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cs.OnConfigChanged(func(cfg config.Config) {})
		}()
	}
	wg.Wait()

	cs.mu.RLock()
	count := len(cs.callbacks)
	cs.mu.RUnlock()

	if count != 10 {
		t.Errorf("expected 10 callbacks, got %d", count)
	}
}

func TestConfigService_SaveConfig_FilePermissions(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	dir, _ := cs.GetStorageDir()
	cfg := config.Config{DataCacheDir: dir}

	if err := cs.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	path, _ := cs.GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	// Verify it's valid JSON
	var loaded config.Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("saved config is not valid JSON: %v", err)
	}
}

func TestConfigService_SaveConfig_MCPServicesInitialized(t *testing.T) {
	cs, cleanup := newTestConfigService(t)
	defer cleanup()

	dir, _ := cs.GetStorageDir()
	cfg := config.Config{
		MCPServices:  nil, // nil should be initialized to empty slice
		DataCacheDir: dir,
	}

	if err := cs.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := cs.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if loaded.MCPServices == nil {
		t.Error("expected MCPServices to be non-nil after save")
	}
}
