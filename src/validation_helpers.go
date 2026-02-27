package main

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a validation error with field and message
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateEmail validates an email address format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}
	return nil
}

// ValidateStringLength validates string length constraints
func ValidateStringLength(field, value string, minLen, maxLen int) error {
	length := utf8.RuneCountInString(value)
	if length < minLen {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters", minLen),
		}
	}
	if maxLen > 0 && length > maxLen {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must not exceed %d characters", maxLen),
		}
	}
	return nil
}

// ValidateRequired validates that a field is not empty
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Message: "is required"}
	}
	return nil
}

// ValidateSlug validates a URL-safe slug (alphanumeric, hyphens, underscores)
func ValidateSlug(slug string) error {
	if slug == "" {
		return &ValidationError{Field: "slug", Message: "slug is required"}
	}
	
	// Slug must be 3-50 characters, alphanumeric with hyphens/underscores
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]{3,50}$`, slug)
	if err != nil {
		return fmt.Errorf("regex error: %w", err)
	}
	if !matched {
		return &ValidationError{
			Field:   "slug",
			Message: "must be 3-50 characters (letters, numbers, hyphens, underscores only)",
		}
	}
	
	// Slug cannot start or end with hyphen/underscore
	if strings.HasPrefix(slug, "-") || strings.HasPrefix(slug, "_") ||
		strings.HasSuffix(slug, "-") || strings.HasSuffix(slug, "_") {
		return &ValidationError{
			Field:   "slug",
			Message: "cannot start or end with hyphen or underscore",
		}
	}
	
	return nil
}

// ValidateURL validates a URL format
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return &ValidationError{Field: "url", Message: "URL is required"}
	}
	
	// Must start with http:// or https://
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return &ValidationError{
			Field:   "url",
			Message: "must start with http:// or https://",
		}
	}
	
	return nil
}

// ValidatePositiveInt validates that an integer is positive
func ValidatePositiveInt(field string, value int) error {
	if value <= 0 {
		return &ValidationError{
			Field:   field,
			Message: "must be a positive number",
		}
	}
	return nil
}

// ValidateRange validates that a number is within a range
func ValidateRange(field string, value, min, max int) error {
	if value < min || value > max {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be between %d and %d", min, max),
		}
	}
	return nil
}

// ValidateEnum validates that a value is in a list of allowed values
func ValidateEnum(field, value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return &ValidationError{
		Field:   field,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
	}
}

// SanitizeHTML removes potentially dangerous HTML tags and attributes
// This is a basic implementation - for production use a proper HTML sanitizer library
func SanitizeHTML(input string) string {
	// Remove script tags
	re := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	input = re.ReplaceAllString(input, "")
	
	// Remove event handlers (onclick, onerror, etc.)
	re = regexp.MustCompile(`(?i)\s*on\w+\s*=\s*["'][^"']*["']`)
	input = re.ReplaceAllString(input, "")
	
	// Remove javascript: protocol
	re = regexp.MustCompile(`(?i)javascript:`)
	input = re.ReplaceAllString(input, "")
	
	return input
}

// ValidateFileExtension validates file extension against allowed list
func ValidateFileExtension(filename string, allowedExts []string) error {
	if filename == "" {
		return &ValidationError{Field: "filename", Message: "filename is required"}
	}
	
	// Get extension
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return &ValidationError{Field: "filename", Message: "file must have an extension"}
	}
	
	ext := strings.ToLower(parts[len(parts)-1])
	
	// Check if extension is allowed
	for _, allowed := range allowedExts {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}
	
	return &ValidationError{
		Field:   "filename",
		Message: fmt.Sprintf("file extension must be one of: %s", strings.Join(allowedExts, ", ")),
	}
}

// ValidateFileSize validates file size against maximum
func ValidateFileSize(size, maxSize int64) error {
	if size <= 0 {
		return &ValidationError{Field: "file_size", Message: "file is empty"}
	}
	if size > maxSize {
		return &ValidationError{
			Field:   "file_size",
			Message: fmt.Sprintf("file size exceeds maximum of %d bytes", maxSize),
		}
	}
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return &ValidationError{
			Field:   "password",
			Message: "must be at least 8 characters",
		}
	}
	
	// Check for at least one uppercase, one lowercase, and one digit
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	
	if !hasUpper || !hasLower || !hasDigit {
		return &ValidationError{
			Field:   "password",
			Message: "must contain at least one uppercase letter, one lowercase letter, and one digit",
		}
	}
	
	return nil
}

// ValidateJSONField validates that a string is valid JSON
func ValidateJSONField(field, value string) error {
	if value == "" {
		return nil // Empty is valid (will be treated as null or empty object)
	}
	
	// Try to parse as JSON
	var js interface{}
	if err := json.Unmarshal([]byte(value), &js); err != nil {
		return &ValidationError{
			Field:   field,
			Message: "must be valid JSON",
		}
	}
	
	return nil
}

// Import json package
import "encoding/json"
