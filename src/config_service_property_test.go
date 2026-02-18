package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"pgregory.net/rapid"
	"vantagedata/config"
)

// Feature: main-architecture-refactor, Property 7: 配置变更通知完整性
//
// For any set of callback functions registered on ConfigService, when SaveConfig()
// is called to save a new configuration, all registered callbacks should be invoked,
// and each callback should receive the same configuration that was saved.
//
// **Validates: Requirements 4.2**

func TestProperty7_ConfigChangeNotificationCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		callbackCount := rapid.IntRange(1, 20).Draw(t, "callbackCount")

		// Create temp directory for test
		tmpDir, err := os.MkdirTemp("", "config_test_*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cs := NewConfigService(nil)
		cs.SetStorageDir(tmpDir)

		// Initialize the service
		if err := cs.Initialize(context.Background()); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		// Track callback invocations
		var mu sync.Mutex
		invokedCallbacks := make([]int, 0, callbackCount)
		receivedConfigs := make([]config.Config, 0, callbackCount)

		// Register callbacks
		for i := 0; i < callbackCount; i++ {
			idx := i
			cs.OnConfigChanged(func(cfg config.Config) {
				mu.Lock()
				defer mu.Unlock()
				invokedCallbacks = append(invokedCallbacks, idx)
				receivedConfigs = append(receivedConfigs, cfg)
			})
		}

		// Generate a random config
		testCfg := generateRandomConfig(t)
		testCfg.DataCacheDir = tmpDir // Use valid directory

		// Save config
		if err := cs.SaveConfig(testCfg); err != nil {
			t.Fatalf("SaveConfig failed: %v", err)
		}

		// Property: all callbacks were invoked
		mu.Lock()
		defer mu.Unlock()

		if len(invokedCallbacks) != callbackCount {
			t.Fatalf("expected %d callbacks invoked, got %d", callbackCount, len(invokedCallbacks))
		}

		// Property: each callback received the saved config
		for i, cfg := range receivedConfigs {
			if cfg.LLMProvider != testCfg.LLMProvider {
				t.Fatalf("callback %d received wrong LLMProvider: got %q, want %q",
					i, cfg.LLMProvider, testCfg.LLMProvider)
			}
			if cfg.ModelName != testCfg.ModelName {
				t.Fatalf("callback %d received wrong ModelName: got %q, want %q",
					i, cfg.ModelName, testCfg.ModelName)
			}
			if cfg.MaxTokens != testCfg.MaxTokens {
				t.Fatalf("callback %d received wrong MaxTokens: got %d, want %d",
					i, cfg.MaxTokens, testCfg.MaxTokens)
			}
		}
	})
}

// Feature: main-architecture-refactor, Property 8: 配置持久化往返一致性
//
// For any valid Config object, after saving it through ConfigService and then loading
// it back, the resulting configuration should be equivalent to the original, and all
// field values should be within the valid ranges defined by Validate().
//
// **Validates: Requirements 4.3, 4.4**

func TestProperty8_ConfigPersistenceRoundTripConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create temp directory for test
		tmpDir, err := os.MkdirTemp("", "config_test_*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cs := NewConfigService(nil)
		cs.SetStorageDir(tmpDir)

		// Initialize the service
		if err := cs.Initialize(context.Background()); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		// Generate a random config with valid values
		originalCfg := generateRandomConfig(t)
		originalCfg.DataCacheDir = tmpDir // Use valid directory

		// Apply Validate to get expected normalized values
		originalCfg.Validate()

		// Save config
		if err := cs.SaveConfig(originalCfg); err != nil {
			t.Fatalf("SaveConfig failed: %v", err)
		}

		// Load config back
		loadedCfg, err := cs.GetConfig()
		if err != nil {
			t.Fatalf("GetConfig failed: %v", err)
		}

		// Property: loaded config matches original (after validation)
		assertConfigEquivalent(t, originalCfg, loadedCfg)

		// Property: loaded config passes validation (all values in valid ranges)
		assertConfigValid(t, loadedCfg)
	})
}

// TestProperty8_ConfigPersistenceWithInvalidValues tests that invalid values
// are corrected during the round-trip through Validate()
func TestProperty8_ConfigPersistenceWithInvalidValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create temp directory for test
		tmpDir, err := os.MkdirTemp("", "config_test_*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cs := NewConfigService(nil)
		cs.SetStorageDir(tmpDir)

		if err := cs.Initialize(context.Background()); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		// Generate config with potentially invalid values
		cfg := generateConfigWithPotentiallyInvalidValues(t)
		cfg.DataCacheDir = tmpDir

		// Save config (Validate is called internally)
		if err := cs.SaveConfig(cfg); err != nil {
			t.Fatalf("SaveConfig failed: %v", err)
		}

		// Load config back
		loadedCfg, err := cs.GetConfig()
		if err != nil {
			t.Fatalf("GetConfig failed: %v", err)
		}

		// Property: loaded config has all values in valid ranges
		assertConfigValid(t, loadedCfg)
	})
}

// generateRandomConfig creates a random but valid Config for testing
func generateRandomConfig(t *rapid.T) config.Config {
	providers := []string{"OpenAI", "Anthropic", "Azure", "Ollama", "DeepSeek"}
	languages := []string{"English", "zh-CN", "ja-JP", "ko-KR", "de-DE"}

	return config.Config{
		LLMProvider:             rapid.SampledFrom(providers).Draw(t, "llmProvider"),
		APIKey:                  rapid.StringMatching(`[a-zA-Z0-9]{10,50}`).Draw(t, "apiKey"),
		BaseURL:                 rapid.StringMatching(`https?://[a-z0-9.]+`).Draw(t, "baseURL"),
		ModelName:               rapid.StringMatching(`[a-z0-9-]{3,30}`).Draw(t, "modelName"),
		MaxTokens:               rapid.IntRange(1000, 32000).Draw(t, "maxTokens"),
		DarkMode:                rapid.Bool().Draw(t, "darkMode"),
		EnableMemory:            rapid.Bool().Draw(t, "enableMemory"),
		AutoAnalysisSuggestions: rapid.Bool().Draw(t, "autoAnalysisSuggestions"),
		LocalCache:              rapid.Bool().Draw(t, "localCache"),
		Language:                rapid.SampledFrom(languages).Draw(t, "language"),
		MaxPreviewRows:          rapid.IntRange(10, 5000).Draw(t, "maxPreviewRows"),
		MaxConcurrentAnalysis:   rapid.IntRange(1, 10).Draw(t, "maxConcurrentAnalysis"),
		MaxAnalysisSteps:        rapid.IntRange(10, 50).Draw(t, "maxAnalysisSteps"),
		DetailedLog:             rapid.Bool().Draw(t, "detailedLog"),
		SoundNotification:       rapid.Bool().Draw(t, "soundNotification"),
		LogMaxSizeMB:            rapid.IntRange(1, 500).Draw(t, "logMaxSizeMB"),
		AutoIntentUnderstanding: rapid.Bool().Draw(t, "autoIntentUnderstanding"),
		MCPServices:             []config.MCPService{},
		SearchAPIs:              []config.SearchAPIConfig{},
		IntentEnhancement:       config.DefaultIntentEnhancementConfig(),
	}
}

// generateConfigWithPotentiallyInvalidValues creates a config that may have
// out-of-range values to test Validate() correction
func generateConfigWithPotentiallyInvalidValues(t *rapid.T) config.Config {
	return config.Config{
		LLMProvider:           rapid.String().Draw(t, "llmProvider"), // May be empty
		ModelName:             rapid.String().Draw(t, "modelName"),
		MaxTokens:             rapid.IntRange(-100, 50000).Draw(t, "maxTokens"),       // May be negative
		MaxPreviewRows:        rapid.IntRange(-100, 20000).Draw(t, "maxPreviewRows"),  // May be out of range
		MaxConcurrentAnalysis: rapid.IntRange(-5, 20).Draw(t, "maxConcurrentAnalysis"), // May be out of range
		MaxAnalysisSteps:      rapid.IntRange(-10, 100).Draw(t, "maxAnalysisSteps"),    // May be out of range
		LogMaxSizeMB:          rapid.IntRange(-10, 1000).Draw(t, "logMaxSizeMB"),       // May be negative
		Language:              rapid.String().Draw(t, "language"),                      // May be empty
		MCPServices:           []config.MCPService{},
		IntentEnhancement:     config.DefaultIntentEnhancementConfig(),
	}
}

// assertConfigEquivalent checks that two configs are equivalent for key fields
func assertConfigEquivalent(t *rapid.T, expected, actual config.Config) {
	if actual.LLMProvider != expected.LLMProvider {
		t.Fatalf("LLMProvider mismatch: got %q, want %q", actual.LLMProvider, expected.LLMProvider)
	}
	if actual.ModelName != expected.ModelName {
		t.Fatalf("ModelName mismatch: got %q, want %q", actual.ModelName, expected.ModelName)
	}
	if actual.MaxTokens != expected.MaxTokens {
		t.Fatalf("MaxTokens mismatch: got %d, want %d", actual.MaxTokens, expected.MaxTokens)
	}
	if actual.Language != expected.Language {
		t.Fatalf("Language mismatch: got %q, want %q", actual.Language, expected.Language)
	}
	if actual.MaxPreviewRows != expected.MaxPreviewRows {
		t.Fatalf("MaxPreviewRows mismatch: got %d, want %d", actual.MaxPreviewRows, expected.MaxPreviewRows)
	}
	if actual.MaxConcurrentAnalysis != expected.MaxConcurrentAnalysis {
		t.Fatalf("MaxConcurrentAnalysis mismatch: got %d, want %d", actual.MaxConcurrentAnalysis, expected.MaxConcurrentAnalysis)
	}
	if actual.MaxAnalysisSteps != expected.MaxAnalysisSteps {
		t.Fatalf("MaxAnalysisSteps mismatch: got %d, want %d", actual.MaxAnalysisSteps, expected.MaxAnalysisSteps)
	}
	if actual.DarkMode != expected.DarkMode {
		t.Fatalf("DarkMode mismatch: got %v, want %v", actual.DarkMode, expected.DarkMode)
	}
	if actual.EnableMemory != expected.EnableMemory {
		t.Fatalf("EnableMemory mismatch: got %v, want %v", actual.EnableMemory, expected.EnableMemory)
	}
	if actual.LocalCache != expected.LocalCache {
		t.Fatalf("LocalCache mismatch: got %v, want %v", actual.LocalCache, expected.LocalCache)
	}
}

// assertConfigValid checks that a config has all values within valid ranges
func assertConfigValid(t *rapid.T, cfg config.Config) {
	// MaxTokens: must be positive
	if cfg.MaxTokens <= 0 {
		t.Fatalf("MaxTokens should be positive after validation, got %d", cfg.MaxTokens)
	}

	// MaxPreviewRows: 1-10000
	if cfg.MaxPreviewRows <= 0 || cfg.MaxPreviewRows > 10000 {
		t.Fatalf("MaxPreviewRows should be 1-10000 after validation, got %d", cfg.MaxPreviewRows)
	}

	// MaxConcurrentAnalysis: 1-10
	if cfg.MaxConcurrentAnalysis < 1 || cfg.MaxConcurrentAnalysis > 10 {
		t.Fatalf("MaxConcurrentAnalysis should be 1-10 after validation, got %d", cfg.MaxConcurrentAnalysis)
	}

	// MaxAnalysisSteps: 10-50
	if cfg.MaxAnalysisSteps < 10 || cfg.MaxAnalysisSteps > 50 {
		t.Fatalf("MaxAnalysisSteps should be 10-50 after validation, got %d", cfg.MaxAnalysisSteps)
	}

	// LogMaxSizeMB: at least 1
	if cfg.LogMaxSizeMB < 1 {
		t.Fatalf("LogMaxSizeMB should be at least 1 after validation, got %d", cfg.LogMaxSizeMB)
	}

	// Language: should not be empty
	if cfg.Language == "" {
		t.Fatalf("Language should not be empty after validation")
	}

	// LLMProvider: should not be empty
	if cfg.LLMProvider == "" {
		t.Fatalf("LLMProvider should not be empty after validation")
	}

	// PanelRightRatio: 0-1
	if cfg.PanelRightRatio < 0 || cfg.PanelRightRatio > 1 {
		t.Fatalf("PanelRightRatio should be 0-1 after validation, got %f", cfg.PanelRightRatio)
	}
}

// TestConfigService_FileNotExist tests that GetConfig returns defaults when file doesn't exist
func TestConfigService_FileNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cs := NewConfigService(nil)
	cs.SetStorageDir(tmpDir)

	if err := cs.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// GetConfig should return defaults when file doesn't exist
	cfg, err := cs.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	// Check default values
	if cfg.LLMProvider != "OpenAI" {
		t.Errorf("expected default LLMProvider 'OpenAI', got %q", cfg.LLMProvider)
	}
	if cfg.ModelName != "gpt-4o" {
		t.Errorf("expected default ModelName 'gpt-4o', got %q", cfg.ModelName)
	}
	if cfg.MaxTokens != 8192 {
		t.Errorf("expected default MaxTokens 8192, got %d", cfg.MaxTokens)
	}
	if cfg.Language != "English" {
		t.Errorf("expected default Language 'English', got %q", cfg.Language)
	}
}

// TestConfigService_InvalidDataCacheDir tests that SaveConfig fails with invalid directory
func TestConfigService_InvalidDataCacheDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cs := NewConfigService(nil)
	cs.SetStorageDir(tmpDir)

	if err := cs.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	cfg := config.Config{
		LLMProvider:  "OpenAI",
		DataCacheDir: filepath.Join(tmpDir, "nonexistent", "path"),
	}

	err = cs.SaveConfig(cfg)
	if err == nil {
		t.Fatalf("SaveConfig should fail with non-existent DataCacheDir")
	}
}
