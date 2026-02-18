package i18n

import "net/http"

// TemplateData returns common template data with translations for the given request.
// It detects the language from the request and returns a map with:
//   - "T": the full translation map for the detected language
//   - "HtmlLang": the HTML lang attribute value (e.g. "zh-CN" or "en")
//   - "Lang": the detected Lang value
func TemplateData(r *http.Request) map[string]interface{} {
	lang := DetectLang(r)
	htmlLang := "zh-CN"
	if lang == EnUS {
		htmlLang = "en"
	}
	return map[string]interface{}{
		"T":        AllTranslations(lang),
		"HtmlLang": htmlLang,
		"Lang":     lang,
	}
}

// MergeTemplateData merges additional key-value pairs into the template data map.
func MergeTemplateData(base map[string]interface{}, extra map[string]interface{}) map[string]interface{} {
	for k, v := range extra {
		base[k] = v
	}
	return base
}
