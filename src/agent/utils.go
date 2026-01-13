package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// SanitizeFilename removes invalid characters from a filename
func SanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	re := regexp.MustCompile(`[<>:"/\\|?*]`) // Corrected escaping for backslash
	safe := re.ReplaceAllString(name, "_")
	safe = strings.TrimSpace(safe)
	if safe == "" {
		return "unnamed"
	}
	// Truncate to reasonable length
	if len(safe) > 50 {
		return safe[:50]
	}
	return safe
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... [%d chars truncated]", len(s)-maxLen)
}
