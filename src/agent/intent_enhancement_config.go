package agent

import "rapidbi/config"

// IntentEnhancementConfig is an alias to config.IntentEnhancementConfig
// This allows the agent package to use the config type while maintaining backward compatibility
type IntentEnhancementConfig = config.IntentEnhancementConfig

// DefaultIntentEnhancementConfig returns the default intent enhancement configuration
// Delegates to config.DefaultIntentEnhancementConfig
func DefaultIntentEnhancementConfig() *IntentEnhancementConfig {
	return config.DefaultIntentEnhancementConfig()
}

// DisabledIntentEnhancementConfig returns a configuration with all enhancements disabled
// Delegates to config.DisabledIntentEnhancementConfig
func DisabledIntentEnhancementConfig() *IntentEnhancementConfig {
	return config.DisabledIntentEnhancementConfig()
}
