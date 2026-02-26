package i18n

import (
	"fmt"
	"sync"
)

// Language represents supported languages
type Language string

const (
	English Language = "English"
	Chinese Language = "简体中文"
)

// Translator provides translation functionality
type Translator struct {
	language     Language
	translations map[Language]map[string]string
	mu           sync.RWMutex
}

var (
	defaultTranslator *Translator
	once              sync.Once
)

// GetTranslator returns the singleton translator instance
func GetTranslator() *Translator {
	once.Do(func() {
		defaultTranslator = &Translator{
			language:     English,
			translations: make(map[Language]map[string]string),
		}
		defaultTranslator.loadTranslations()
	})
	return defaultTranslator
}

// SetLanguage sets the current language
func (t *Translator) SetLanguage(lang Language) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.language = lang
}

// GetLanguage returns the current language
func (t *Translator) GetLanguage() Language {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.language
}

// T translates a key with optional parameters
func (t *Translator) T(key string, params ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	langMap, ok := t.translations[t.language]
	if !ok {
		// Fallback to English if current language not found
		langMap, ok = t.translations[English]
		if !ok {
			return key
		}
	}

	text, ok := langMap[key]
	if !ok {
		// Fallback to English for missing keys in non-English languages
		if t.language != English {
			if enMap, enOk := t.translations[English]; enOk {
				if enText, found := enMap[key]; found {
					if len(params) > 0 {
						return fmt.Sprintf(enText, params...)
					}
					return enText
				}
			}
		}
		return key
	}

	// Replace parameters if provided
	if len(params) > 0 {
		return fmt.Sprintf(text, params...)
	}
	return text
}

// T is a convenience function for translation
func T(key string, params ...interface{}) string {
	return GetTranslator().T(key, params...)
}

// TForLang translates a key for a specific language, useful when generating
// content that must be in a particular language regardless of global setting.
func TForLang(lang Language, key string, params ...interface{}) string {
	return GetTranslator().TForLang(lang, key, params...)
}

// TForLang translates a key for a specific language
func (t *Translator) TForLang(lang Language, key string, params ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	langMap, ok := t.translations[lang]
	if !ok {
		langMap, ok = t.translations[English]
		if !ok {
			return key
		}
	}

	text, ok := langMap[key]
	if !ok {
		if lang != English {
			if enMap, enOk := t.translations[English]; enOk {
				if enText, found := enMap[key]; found {
					if len(params) > 0 {
						return fmt.Sprintf(enText, params...)
					}
					return enText
				}
			}
		}
		return key
	}

	if len(params) > 0 {
		return fmt.Sprintf(text, params...)
	}
	return text
}

// SetLanguage is a convenience function to set language
func SetLanguage(lang Language) {
	GetTranslator().SetLanguage(lang)
}

// GetLanguage is a convenience function to get current language
func GetLanguage() Language {
	return GetTranslator().GetLanguage()
}

// loadTranslations loads all translation strings
func (t *Translator) loadTranslations() {
	t.translations[English] = englishTranslations
	t.translations[Chinese] = chineseTranslations
}
