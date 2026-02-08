package i18n

import (
	"vantagedata/config"
)

// SyncLanguageFromConfig synchronizes language setting from application config
// This should be called when the application starts or when config changes
func SyncLanguageFromConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}

	var lang Language
	switch cfg.Language {
	case "简体中文":
		lang = Chinese
	case "English":
		lang = English
	default:
		// Default to English if not set or invalid
		lang = English
	}

	SetLanguage(lang)
}

// GetLanguageString returns the language as a string compatible with frontend
func GetLanguageString() string {
	lang := GetLanguage()
	return string(lang)
}

// ParseLanguage converts a string to Language type
func ParseLanguage(langStr string) Language {
	switch langStr {
	case "简体中文":
		return Chinese
	case "English":
		return English
	default:
		return English
	}
}
