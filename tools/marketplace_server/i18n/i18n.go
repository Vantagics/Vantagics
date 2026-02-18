package i18n

import (
	"net/http"
	"strings"
)

// Lang represents a supported language.
type Lang string

const (
	ZhCN Lang = "zh-CN"
	EnUS Lang = "en-US"
)

// T returns the translation for the given key in the specified language.
func T(lang Lang, key string) string {
	if m, ok := translations[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	// Fallback to zh-CN
	if m, ok := translations[ZhCN]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}

// DetectLang detects the preferred language from the request.
// Priority: cookie > query param > Accept-Language header > configured default
func DetectLang(r *http.Request) Lang {
	// 1. Cookie
	if c, err := r.Cookie("lang"); err == nil {
		if l := normalizeLang(c.Value); l != "" {
			return l
		}
	}
	// 2. Query param
	if q := r.URL.Query().Get("lang"); q != "" {
		if l := normalizeLang(q); l != "" {
			return l
		}
	}
	// 3. Configured default (skip browser Accept-Language to respect admin setting)
	if DefaultLang != "" {
		return DefaultLang
	}
	return ZhCN
}

// DefaultLang is the system-wide default language, configurable via admin settings.
// Set by main.go from the "default_language" setting. Empty means use ZhCN.
var DefaultLang Lang

// normalizeLang maps various language tags to supported Lang values.
func normalizeLang(s string) Lang {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "en" || strings.HasPrefix(s, "en-") || strings.HasPrefix(s, "en_"):
		return EnUS
	case s == "zh" || strings.HasPrefix(s, "zh-") || strings.HasPrefix(s, "zh_"):
		return ZhCN
	}
	return ""
}

// AllTranslations returns the full translation map for a language (used in JS).
func AllTranslations(lang Lang) map[string]string {
	if m, ok := translations[lang]; ok {
		return m
	}
	return translations[ZhCN]
}
