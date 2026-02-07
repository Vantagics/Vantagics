package agent

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
)

// **Validates: Requirements 1.2**
// 属性 5: 图片检测覆盖性
// 对于任意包含图片引用的文本（base64、markdown、文件引用、sandbox 路径），
// ImageDetector 应该能够检测到所有图片引用。

// TestImageDetector_Property_Base64Detection tests that all generated base64 images are detected
// **Validates: Requirements 1.2**
func TestImageDetector_Property_Base64Detection(t *testing.T) {
	detector := NewImageDetector()

	// Property: For any valid base64 image pattern, DetectBase64Images should find it
	property := func(imageType string, dataLength uint8) bool {
		// Constrain imageType to valid image types
		validTypes := []string{"png", "jpeg", "gif", "webp", "bmp", "tiff"}
		imageType = validTypes[int(dataLength)%len(validTypes)]

		// Generate valid base64 data (use a reasonable length)
		base64Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		dataLen := int(dataLength)%64 + 4 // At least 4 chars, max 67
		// Ensure length is multiple of 4 for valid base64
		dataLen = (dataLen / 4) * 4
		if dataLen < 4 {
			dataLen = 4
		}

		var base64Data strings.Builder
		for i := 0; i < dataLen; i++ {
			base64Data.WriteByte(base64Chars[rand.Intn(len(base64Chars))])
		}

		// Create the base64 image pattern
		pattern := fmt.Sprintf("data:image/%s;base64,%s", imageType, base64Data.String())

		// Embed in some surrounding text
		text := fmt.Sprintf("Here is an image: %s and some more text", pattern)

		// Detect images
		detected := detector.DetectBase64Images(text)

		// Property: Should detect exactly 1 image
		if len(detected) != 1 {
			t.Logf("Expected 1 base64 image, got %d for pattern: %s", len(detected), pattern)
			return false
		}

		// Property: The detected pattern should contain our data
		if !strings.Contains(detected[0].Data, imageType) {
			t.Logf("Detected data doesn't contain image type: %s", imageType)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_MarkdownDetection tests that all generated markdown images are detected
// **Validates: Requirements 1.2**
func TestImageDetector_Property_MarkdownDetection(t *testing.T) {
	detector := NewImageDetector()

	// Property: For any valid markdown image pattern, DetectMarkdownImages should find it
	property := func(altText string, pathSeed uint16) bool {
		// Sanitize alt text (remove characters that would break markdown)
		altText = strings.ReplaceAll(altText, "[", "")
		altText = strings.ReplaceAll(altText, "]", "")
		altText = strings.ReplaceAll(altText, "(", "")
		altText = strings.ReplaceAll(altText, ")", "")
		if len(altText) > 20 {
			altText = altText[:20]
		}

		// Generate a valid image path
		extensions := []string{"png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "tiff"}
		ext := extensions[int(pathSeed)%len(extensions)]
		path := fmt.Sprintf("images/chart_%d.%s", pathSeed, ext)

		// Create the markdown image pattern
		pattern := fmt.Sprintf("![%s](%s)", altText, path)

		// Embed in some surrounding text
		text := fmt.Sprintf("Check this image: %s for details", pattern)

		// Detect images
		detected := detector.DetectMarkdownImages(text)

		// Property: Should detect exactly 1 image
		if len(detected) != 1 {
			t.Logf("Expected 1 markdown image, got %d for pattern: %s", len(detected), pattern)
			return false
		}

		// Property: The detected path should match
		if detected[0].Data != path {
			t.Logf("Expected path %s, got %s", path, detected[0].Data)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_FileReferenceDetection tests that all generated file references are detected
// **Validates: Requirements 1.2**
func TestImageDetector_Property_FileReferenceDetection(t *testing.T) {
	detector := NewImageDetector()

	// Property: For any valid file reference pattern, DetectFileReferences should find it
	property := func(filenameSeed uint16) bool {
		// Generate a valid filename
		extensions := []string{"png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "tiff", "tif"}
		ext := extensions[int(filenameSeed)%len(extensions)]
		filename := fmt.Sprintf("output_%d.%s", filenameSeed, ext)

		// Test both files/ and file:// prefixes
		prefixes := []string{"files/", "file://"}
		prefix := prefixes[int(filenameSeed/256)%len(prefixes)]

		// Create the file reference pattern
		pattern := prefix + filename

		// Embed in some surrounding text
		text := fmt.Sprintf("Generated file: %s is ready", pattern)

		// Detect file references
		detected := detector.DetectFileReferences(text)

		// Property: Should detect exactly 1 file reference
		if len(detected) != 1 {
			t.Logf("Expected 1 file reference, got %d for pattern: %s", len(detected), pattern)
			return false
		}

		// Property: The detected filename should match
		if detected[0].Data != filename {
			t.Logf("Expected filename %s, got %s", filename, detected[0].Data)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_SandboxPathDetection tests that all generated sandbox paths are detected
// **Validates: Requirements 1.2**
func TestImageDetector_Property_SandboxPathDetection(t *testing.T) {
	detector := NewImageDetector()

	// Property: For any valid sandbox path pattern, DetectSandboxPaths should find it
	property := func(filenameSeed uint16) bool {
		// Generate a valid filename
		extensions := []string{"png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "tiff", "tif"}
		ext := extensions[int(filenameSeed)%len(extensions)]
		filename := fmt.Sprintf("chart_%d.%s", filenameSeed, ext)

		// Create the sandbox path pattern (OpenAI code interpreter format)
		pattern := fmt.Sprintf("sandbox:/mnt/data/%s", filename)

		// Embed in some surrounding text
		text := fmt.Sprintf("Image saved to: %s successfully", pattern)

		// Detect sandbox paths
		detected := detector.DetectSandboxPaths(text)

		// Property: Should detect exactly 1 sandbox path
		if len(detected) != 1 {
			t.Logf("Expected 1 sandbox path, got %d for pattern: %s", len(detected), pattern)
			return false
		}

		// Property: The detected filename should match
		if detected[0].Data != filename {
			t.Logf("Expected filename %s, got %s", filename, detected[0].Data)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_HTMLImgDetection tests that all generated HTML img tags are detected
// **Validates: Requirements 1.2**
func TestImageDetector_Property_HTMLImgDetection(t *testing.T) {
	detector := NewImageDetector()

	// Property: For any valid HTML img tag, DetectHTMLImgTags should find it
	property := func(srcSeed uint16, hasAlt bool) bool {
		// Generate a valid image src
		extensions := []string{"png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "tiff", "tif"}
		ext := extensions[int(srcSeed)%len(extensions)]
		src := fmt.Sprintf("images/photo_%d.%s", srcSeed, ext)

		// Create the HTML img tag pattern
		var pattern string
		if hasAlt {
			pattern = fmt.Sprintf(`<img src="%s" alt="description">`, src)
		} else {
			pattern = fmt.Sprintf(`<img src="%s">`, src)
		}

		// Embed in some surrounding text
		text := fmt.Sprintf("Here is the image: %s in the document", pattern)

		// Detect HTML img tags
		detected := detector.DetectHTMLImgTags(text)

		// Property: Should detect exactly 1 HTML img tag
		if len(detected) != 1 {
			t.Logf("Expected 1 HTML img tag, got %d for pattern: %s", len(detected), pattern)
			return false
		}

		// Property: The detected src should match
		if detected[0].Data != src {
			t.Logf("Expected src %s, got %s", src, detected[0].Data)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}


// TestImageDetector_Property_DetectAllImages_Completeness tests that DetectAllImages finds all image types
// **Validates: Requirements 1.2**
func TestImageDetector_Property_DetectAllImages_Completeness(t *testing.T) {
	detector := NewImageDetector()

	// Property: DetectAllImages should find all types of images in a mixed text
	property := func(seed uint16) bool {
		extensions := []string{"png", "jpg", "jpeg", "gif", "webp"}
		ext := extensions[int(seed)%len(extensions)]

		// Create one of each type
		base64Pattern := fmt.Sprintf("data:image/%s;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk", ext)
		markdownPattern := fmt.Sprintf("![chart](images/chart_%d.%s)", seed, ext)
		fileRefPattern := fmt.Sprintf("files/output_%d.%s", seed, ext)
		sandboxPattern := fmt.Sprintf("sandbox:/mnt/data/result_%d.%s", seed, ext)
		htmlImgPattern := fmt.Sprintf(`<img src="photos/image_%d.%s">`, seed, ext)

		// Combine all patterns in a text
		text := fmt.Sprintf(`
Analysis Results:
1. Base64 image: %s
2. Markdown image: %s
3. File reference: %s
4. Sandbox path: %s
5. HTML image: %s
`, base64Pattern, markdownPattern, fileRefPattern, sandboxPattern, htmlImgPattern)

		// Detect all images
		detected := detector.DetectAllImages(text)

		// Property: Should detect exactly 5 images (one of each type)
		if len(detected) != 5 {
			t.Logf("Expected 5 images, got %d", len(detected))
			for i, d := range detected {
				t.Logf("  [%d] Type=%s, Data=%s", i, d.Type, d.Data)
			}
			return false
		}

		// Property: Should have one of each type
		typeCount := make(map[string]int)
		for _, d := range detected {
			typeCount[d.Type]++
		}

		expectedTypes := []string{"base64", "markdown", "file_reference", "sandbox", "html_img"}
		for _, expectedType := range expectedTypes {
			if typeCount[expectedType] != 1 {
				t.Logf("Expected 1 %s image, got %d", expectedType, typeCount[expectedType])
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_MultipleImagesOfSameType tests detection of multiple images of the same type
// **Validates: Requirements 1.2**
func TestImageDetector_Property_MultipleImagesOfSameType(t *testing.T) {
	detector := NewImageDetector()

	// Property: Should detect all images when multiple of the same type exist
	property := func(count uint8) bool {
		// Constrain count to reasonable range (1-10)
		numImages := int(count)%10 + 1

		// Generate multiple markdown images
		var patterns []string
		for i := 0; i < numImages; i++ {
			pattern := fmt.Sprintf("![chart%d](images/chart_%d.png)", i, i)
			patterns = append(patterns, pattern)
		}

		// Combine all patterns in a text
		text := "Images: " + strings.Join(patterns, " | ")

		// Detect markdown images
		detected := detector.DetectMarkdownImages(text)

		// Property: Should detect exactly numImages images
		if len(detected) != numImages {
			t.Logf("Expected %d markdown images, got %d", numImages, len(detected))
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_NoFalsePositives tests that non-image text doesn't produce false positives
// **Validates: Requirements 1.2**
func TestImageDetector_Property_NoFalsePositives(t *testing.T) {
	detector := NewImageDetector()

	// Property: Text without image patterns should not produce any detections
	property := func(text string) bool {
		// Filter out any text that might accidentally contain image patterns
		// by removing common image-related substrings
		text = strings.ReplaceAll(text, "data:", "")
		text = strings.ReplaceAll(text, "base64", "")
		text = strings.ReplaceAll(text, "![", "")
		text = strings.ReplaceAll(text, "](", "")
		text = strings.ReplaceAll(text, "files/", "")
		text = strings.ReplaceAll(text, "file://", "")
		text = strings.ReplaceAll(text, "sandbox:", "")
		text = strings.ReplaceAll(text, "<img", "")
		text = strings.ReplaceAll(text, ".png", "")
		text = strings.ReplaceAll(text, ".jpg", "")
		text = strings.ReplaceAll(text, ".jpeg", "")
		text = strings.ReplaceAll(text, ".gif", "")
		text = strings.ReplaceAll(text, ".webp", "")
		text = strings.ReplaceAll(text, ".svg", "")
		text = strings.ReplaceAll(text, ".bmp", "")
		text = strings.ReplaceAll(text, ".tiff", "")
		text = strings.ReplaceAll(text, ".tif", "")

		// Detect all images
		detected := detector.DetectAllImages(text)

		// Property: Should detect 0 images
		if len(detected) != 0 {
			t.Logf("Expected 0 images in sanitized text, got %d", len(detected))
			for i, d := range detected {
				t.Logf("  [%d] Type=%s, Raw=%s", i, d.Type, d.Raw)
			}
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_CountMatchesDetect tests that CountImages matches DetectAllImages length
// **Validates: Requirements 1.2**
func TestImageDetector_Property_CountMatchesDetect(t *testing.T) {
	detector := NewImageDetector()

	// Property: CountImages should return the same count as len(DetectAllImages)
	property := func(seed uint16) bool {
		// Generate a random number of images (0-5)
		numImages := int(seed) % 6

		var patterns []string
		for i := 0; i < numImages; i++ {
			pattern := fmt.Sprintf("![image%d](path/to/image_%d.png)", i, i)
			patterns = append(patterns, pattern)
		}

		text := "Content: " + strings.Join(patterns, " ")

		// Get count and detected list
		count := detector.CountImages(text)
		detected := detector.DetectAllImages(text)

		// Property: Count should match length of detected
		if count != len(detected) {
			t.Logf("CountImages returned %d, but DetectAllImages found %d", count, len(detected))
			return false
		}

		// Property: Count should match expected number
		if count != numImages {
			t.Logf("Expected %d images, got count=%d", numImages, count)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestImageDetector_Property_HasImagesConsistency tests that HasImages is consistent with CountImages
// **Validates: Requirements 1.2**
func TestImageDetector_Property_HasImagesConsistency(t *testing.T) {
	detector := NewImageDetector()

	// Property: HasImages should return true iff CountImages > 0
	property := func(seed uint16) bool {
		// Generate a random number of images (0-3)
		numImages := int(seed) % 4

		var patterns []string
		for i := 0; i < numImages; i++ {
			pattern := fmt.Sprintf("files/output_%d.png", i)
			patterns = append(patterns, pattern)
		}

		text := "Files: " + strings.Join(patterns, ", ")

		// Get hasImages and count
		hasImages := detector.HasImages(text)
		count := detector.CountImages(text)

		// Property: HasImages should be true iff count > 0
		expectedHasImages := count > 0
		if hasImages != expectedHasImages {
			t.Logf("HasImages=%v but count=%d (expected HasImages=%v)", hasImages, count, expectedHasImages)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}
