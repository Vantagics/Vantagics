package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// ImagePattern represents a detected image pattern with its type and data
type ImagePattern struct {
	Type string // "base64", "markdown", "file_reference"
	Data string // The extracted image data
	Raw  string // The original matched string
}

// ImageDetector provides utilities for detecting and validating image patterns
type ImageDetector struct {
	// Regex patterns for different image formats
	base64Pattern        *regexp.Regexp
	markdownPattern      *regexp.Regexp
	fileReferencePattern *regexp.Regexp
	sandboxPattern       *regexp.Regexp // Pattern for sandbox: paths (OpenAI code interpreter format)
	htmlImgPattern       *regexp.Regexp // Pattern for HTML img tags
	logger               func(string)   // Optional logger function for debug logging
}

// NewImageDetector creates a new ImageDetector with compiled regex patterns
func NewImageDetector() *ImageDetector {
	return &ImageDetector{
		// Pattern for base64 images: data:image/[type];base64,[data]
		// Matches: data:image/png;base64,iVBORw0KGgo...
		// Matches: data:image/jpeg;base64,/9j/4AAQSkZJRg...
		// Matches: data:image/gif;base64,R0lGODlh...
		// Matches: data:image/webp;base64,...
		// Matches: data:image/bmp;base64,...
		// Matches: data:image/tiff;base64,...
		base64Pattern: regexp.MustCompile(
			`data:image/([a-zA-Z0-9+\-\.]+);base64,([A-Za-z0-9+/=]+)`,
		),

		// Pattern for markdown images: ![alt](path)
		// Matches: ![alt text](image.png)
		// Matches: ![](path/to/image.jpg)
		// Matches: ![description](https://example.com/image.png)
		markdownPattern: regexp.MustCompile(
			`!\[([^\]]*)\]\(([^\)]+)\)`,
		),

		// Pattern for file references: files/[filename]
		// Matches: files/chart.png
		// Matches: files/output_123.jpg
		// Matches: file://path/to/image.png
		// Supports: png, jpg, jpeg, gif, webp, svg, bmp, tiff, tif
		fileReferencePattern: regexp.MustCompile(
			`(?:files/|file://)([^\s\)]+\.(png|jpg|jpeg|gif|webp|svg|bmp|tiff|tif))`,
		),

		// Pattern for sandbox paths (OpenAI code interpreter format)
		// Matches: sandbox:/mnt/data/chart.png
		// Matches: sandbox:/mnt/data/output.jpg
		// Supports: png, jpg, jpeg, gif, webp, svg, bmp, tiff, tif
		sandboxPattern: regexp.MustCompile(
			`sandbox:(/[^\s\)]+\.(png|jpg|jpeg|gif|webp|svg|bmp|tiff|tif))`,
		),

		// Pattern for HTML img tags
		// Matches: <img src="image.png">
		// Matches: <img src='path/to/image.jpg' alt="description">
		// Matches: <img alt="text" src="https://example.com/image.webp" />
		// Supports: png, jpg, jpeg, gif, webp, svg, bmp, tiff, tif
		htmlImgPattern: regexp.MustCompile(
			`<img[^>]*\ssrc=["']([^"']+\.(png|jpg|jpeg|gif|webp|svg|bmp|tiff|tif))["'][^>]*>`,
		),

		logger: nil,
	}
}

// SetLogger sets the logger function for debug logging
func (id *ImageDetector) SetLogger(logger func(string)) {
	id.logger = logger
}

// log writes a debug message if logger is set
func (id *ImageDetector) log(message string) {
	if id.logger != nil {
		id.logger(message)
	}
}

// DetectBase64Images finds all base64 image patterns in the given text
// Returns a slice of ImagePattern structs with Type="base64"
func (id *ImageDetector) DetectBase64Images(text string) []ImagePattern {
	var patterns []ImagePattern
	matches := id.base64Pattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			// match[0] is the full match
			// match[1] is the image type (png, jpeg, etc.)
			// match[2] is the base64 data
			patterns = append(patterns, ImagePattern{
				Type: "base64",
				Data: match[0], // Store the full data URL
				Raw:  match[0],
			})
			id.log(fmt.Sprintf("[IMAGE-DETECTOR] Found base64 image: type=%s, data length=%d", match[1], len(match[2])))
		}
	}

	if len(patterns) > 0 {
		id.log(fmt.Sprintf("[IMAGE-DETECTOR] DetectBase64Images: found %d base64 images", len(patterns)))
	}

	return patterns
}

// DetectMarkdownImages finds all markdown image patterns in the given text
// Returns a slice of ImagePattern structs with Type="markdown"
func (id *ImageDetector) DetectMarkdownImages(text string) []ImagePattern {
	var patterns []ImagePattern
	matches := id.markdownPattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			// match[0] is the full match: ![alt](path)
			// match[1] is the alt text
			// match[2] is the image path
			patterns = append(patterns, ImagePattern{
				Type: "markdown",
				Data: match[2], // Store the path
				Raw:  match[0],
			})
			id.log(fmt.Sprintf("[IMAGE-DETECTOR] Found markdown image: alt='%s', path='%s'", match[1], match[2]))
		}
	}

	if len(patterns) > 0 {
		id.log(fmt.Sprintf("[IMAGE-DETECTOR] DetectMarkdownImages: found %d markdown images", len(patterns)))
	}

	return patterns
}

// DetectFileReferences finds all file reference patterns in the given text
// Returns a slice of ImagePattern structs with Type="file_reference"
func (id *ImageDetector) DetectFileReferences(text string) []ImagePattern {
	var patterns []ImagePattern
	matches := id.fileReferencePattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			// match[0] is the full match: files/filename or file:///path
			// match[1] is the filename
			patterns = append(patterns, ImagePattern{
				Type: "file_reference",
				Data: match[1], // Store just the filename
				Raw:  match[0],
			})
			id.log(fmt.Sprintf("[IMAGE-DETECTOR] Found file reference: '%s'", match[1]))
		}
	}

	if len(patterns) > 0 {
		id.log(fmt.Sprintf("[IMAGE-DETECTOR] DetectFileReferences: found %d file references", len(patterns)))
	}

	return patterns
}

// DetectSandboxPaths finds all sandbox: path patterns in the given text
// These are generated by OpenAI code interpreter and need to be converted
// Returns a slice of ImagePattern structs with Type="sandbox"
func (id *ImageDetector) DetectSandboxPaths(text string) []ImagePattern {
	var patterns []ImagePattern
	matches := id.sandboxPattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			// match[0] is the full match: sandbox:/mnt/data/filename.png
			// match[1] is the path: /mnt/data/filename.png
			// Extract just the filename from the path
			path := match[1]
			parts := strings.Split(path, "/")
			filename := parts[len(parts)-1]
			
			patterns = append(patterns, ImagePattern{
				Type: "sandbox",
				Data: filename, // Store just the filename
				Raw:  match[0],
			})
			id.log(fmt.Sprintf("[IMAGE-DETECTOR] Found sandbox path: '%s' -> filename='%s'", path, filename))
		}
	}

	if len(patterns) > 0 {
		id.log(fmt.Sprintf("[IMAGE-DETECTOR] DetectSandboxPaths: found %d sandbox paths", len(patterns)))
	}

	return patterns
}

// DetectHTMLImgTags finds all HTML img tag patterns in the given text
// Returns a slice of ImagePattern structs with Type="html_img"
func (id *ImageDetector) DetectHTMLImgTags(text string) []ImagePattern {
	var patterns []ImagePattern
	matches := id.htmlImgPattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			// match[0] is the full match: <img src="path/to/image.png" ...>
			// match[1] is the src value: path/to/image.png
			patterns = append(patterns, ImagePattern{
				Type: "html_img",
				Data: match[1], // Store the src path
				Raw:  match[0],
			})
			id.log(fmt.Sprintf("[IMAGE-DETECTOR] Found HTML img tag: src='%s'", match[1]))
		}
	}

	if len(patterns) > 0 {
		id.log(fmt.Sprintf("[IMAGE-DETECTOR] DetectHTMLImgTags: found %d HTML img tags", len(patterns)))
	}

	return patterns
}

// DetectAllImages finds all image patterns in the given text
// Returns a slice of all detected ImagePattern structs
func (id *ImageDetector) DetectAllImages(text string) []ImagePattern {
	var allPatterns []ImagePattern

	id.log(fmt.Sprintf("[IMAGE-DETECTOR] DetectAllImages: scanning text of length %d", len(text)))

	// Detect base64 images
	allPatterns = append(allPatterns, id.DetectBase64Images(text)...)

	// Detect markdown images
	allPatterns = append(allPatterns, id.DetectMarkdownImages(text)...)

	// Detect file references
	allPatterns = append(allPatterns, id.DetectFileReferences(text)...)

	// Detect sandbox paths (OpenAI code interpreter format)
	allPatterns = append(allPatterns, id.DetectSandboxPaths(text)...)

	// Detect HTML img tags
	allPatterns = append(allPatterns, id.DetectHTMLImgTags(text)...)

	id.log(fmt.Sprintf("[IMAGE-DETECTOR] DetectAllImages: total images found = %d", len(allPatterns)))

	return allPatterns
}

// IsValidBase64Image validates if a string is a valid base64 image data URL
func (id *ImageDetector) IsValidBase64Image(data string) bool {
	if !strings.HasPrefix(data, "data:image/") {
		return false
	}

	if !strings.Contains(data, ";base64,") {
		return false
	}

	// Extract the base64 part
	parts := strings.Split(data, ";base64,")
	if len(parts) != 2 {
		return false
	}

	base64Data := parts[1]

	// Validate base64 format (should only contain valid base64 characters)
	// Base64 uses A-Z, a-z, 0-9, +, /, and = for padding
	validBase64 := regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	if !validBase64.MatchString(base64Data) {
		return false
	}

	// Check that length is valid for base64 (should be multiple of 4 with padding)
	if len(base64Data)%4 != 0 {
		return false
	}

	return true
}

// IsValidMarkdownImage validates if a string is a valid markdown image reference
func (id *ImageDetector) IsValidMarkdownImage(data string) bool {
	// Check if it matches the markdown pattern
	return id.markdownPattern.MatchString(data)
}

// IsValidFileReference validates if a string is a valid file reference
func (id *ImageDetector) IsValidFileReference(data string) bool {
	// Check if it matches the file reference pattern
	return id.fileReferencePattern.MatchString(data)
}

// ExtractImageType extracts the MIME type from a base64 image data URL
// Returns the MIME type (e.g., "image/png") or empty string if not found
func (id *ImageDetector) ExtractImageType(base64Data string) string {
	if !strings.HasPrefix(base64Data, "data:") {
		return ""
	}

	// Extract the part between "data:" and ";base64"
	parts := strings.Split(base64Data, ";base64,")
	if len(parts) != 2 {
		return ""
	}

	mimeType := strings.TrimPrefix(parts[0], "data:")
	return mimeType
}

// ExtractBase64Data extracts just the base64 string from a data URL
// Returns the base64 string without the "data:image/...;base64," prefix
func (id *ImageDetector) ExtractBase64Data(base64DataURL string) string {
	if !strings.Contains(base64DataURL, ";base64,") {
		return ""
	}

	parts := strings.Split(base64DataURL, ";base64,")
	if len(parts) != 2 {
		return ""
	}

	return parts[1]
}

// NormalizeImagePath normalizes a file path to a consistent format (just the filename)
// Handles various path formats:
// - "files/filename", "file:///path/to/filename", "sandbox:/mnt/data/filename"
// - Windows paths: "C:\path\to\file.png", "D:\images\chart.jpg"
// - Unix paths: "/home/user/images/chart.png", "/tmp/output.jpg"
// - Relative paths: "./images/chart.png", "../output/result.jpg", "images/chart.png"
// - Mixed separators: "path/to\file.png"
func (id *ImageDetector) NormalizeImagePath(path string) string {
	if path == "" {
		return ""
	}

	// Remove sandbox: prefix if present (OpenAI code interpreter format)
	if strings.HasPrefix(path, "sandbox:") {
		path = strings.TrimPrefix(path, "sandbox:")
	}

	// Remove file:// or file:/// prefix if present
	if strings.HasPrefix(path, "file:///") {
		path = strings.TrimPrefix(path, "file:///")
	} else if strings.HasPrefix(path, "file://") {
		path = strings.TrimPrefix(path, "file://")
	}

	// Normalize path separators: convert Windows backslashes to forward slashes
	path = strings.ReplaceAll(path, "\\", "/")

	// Handle Windows drive letters (e.g., "C:/path/to/file" -> "path/to/file")
	// After backslash normalization, Windows paths look like "C:/path/to/file"
	if len(path) >= 2 && path[1] == ':' {
		// Check if it's a valid drive letter (A-Z or a-z)
		driveLetter := path[0]
		if (driveLetter >= 'A' && driveLetter <= 'Z') || (driveLetter >= 'a' && driveLetter <= 'z') {
			// Remove drive letter and colon, e.g., "C:/path" -> "/path"
			path = path[2:]
		}
	}

	// Remove leading "./" for relative paths
	for strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	}

	// Handle "../" by removing it (we just want the filename anyway)
	for strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
	}

	// Remove leading slashes (Unix absolute paths)
	path = strings.TrimLeft(path, "/")

	// Extract just the filename from the path
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		// Get the last non-empty part
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" {
				return parts[i]
			}
		}
	}

	return path
}

// CountImages returns the total number of images detected in the text
func (id *ImageDetector) CountImages(text string) int {
	return len(id.DetectAllImages(text))
}

// HasImages returns true if any images are detected in the text
func (id *ImageDetector) HasImages(text string) bool {
	return id.CountImages(text) > 0
}
