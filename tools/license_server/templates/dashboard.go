package templates

import (
	"strings"
)

// GetDashboardHTML returns the complete dashboard HTML
func GetDashboardHTML() string {
	// Combine all panel HTML
	content := strings.Join([]string{
		LicensesHTML,
		EmailRecordsHTML,
		EmailFilterHTML,
		ProductTypesHTML,
		LicenseGroupsHTML,
		LLMConfigHTML,
		SearchConfigHTML,
		BackupHTML,
		SettingsHTML,
	}, "\n")
	
	// Combine all scripts
	scripts := strings.Join([]string{
		CommonScripts,
		LicensesScripts,
		EmailRecordsScripts,
		EmailFilterScripts,
		ProductTypesScripts,
		LicenseGroupsScripts,
		LLMConfigScripts,
		SearchConfigScripts,
		BackupScripts,
		SettingsScripts,
		InitScripts,
	}, "\n")
	
	// Replace placeholders in base HTML
	html := BaseHTML
	html = strings.Replace(html, "{{.Content}}", content, 1)
	html = strings.Replace(html, "{{.Scripts}}", scripts, 1)
	
	return html
}
