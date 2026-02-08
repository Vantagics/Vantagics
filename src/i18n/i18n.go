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
		return key
	}

	text, ok := langMap[key]
	if !ok {
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
