package templates

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"golang.org/x/net/html"
)

// StorefrontInfo represents the storefront data structure used in templates
type StorefrontInfo struct {
	ID              int64
	UserID          int64
	StoreName       string
	StoreSlug       string
	Description     string
	HasLogo         bool
	LogoContentType string
	AutoAddEnabled  bool
	StoreLayout     string
	CreatedAt       string
	UpdatedAt       string
}

// SectionConfig represents a section configuration
type SectionConfig struct {
	Type    string
	Visible bool
}

// CustomBannerSettings represents custom banner settings
type CustomBannerSettings struct {
	Text  string
	Style string
}

// CustomProduct represents a custom product
type CustomProduct struct {
	ID          int64
	ProductName string
	Description string
	PriceUSD    float64
	ProductType string
}

// StorefrontPackInfo represents pack information
type StorefrontPackInfo struct {
	ListingID     int64
	PackName      string
	PackDesc      string
	ShareToken    string
	ShareMode     string
	CreditsPrice  int
	DownloadCount int
	CategoryName  string
	HasLogo       bool
}

// StorefrontPageData represents the template data structure
type StorefrontPageData struct {
	Storefront         StorefrontInfo
	FeaturedPacks      []StorefrontPackInfo
	Packs              []StorefrontPackInfo
	PurchasedIDs       map[int64]bool
	IsLoggedIn         bool
	CurrentUserID      int64
	DefaultLang        string
	Filter             string
	Sort               string
	SearchQuery        string
	Categories         []string
	CategoryFilter     string
	DownloadURLWindows string
	DownloadURLMacOS   string
	Sections           []SectionConfig
	ThemeCSS           string
	PackGridColumns    int
	BannerData         map[int]CustomBannerSettings
	HeroLayout         string
	IsPreviewMode      bool
	CustomProducts     []CustomProduct
	FeaturedVisible    bool
	SupportApproved    bool
	ServicePortalURL   string
}

// createTestData creates a StorefrontPageData with the given store name
func genStorefrontData() gopter.Gen {
	return gen.Frequency(
		map[int]gopter.Gen{
			// Normal length names (2-30 characters) - most common
			50: gen.AlphaString().SuchThat(func(s string) bool {
				runes := []rune(s)
				return len(runes) >= 2 && len(runes) <= 30
			}).Map(func(name string) StorefrontPageData {
				return createTestData(name)
			}),
			// Chinese names
			20: gen.OneConstOf(
				"技术分析工作室",
				"数据科学实验室",
				"AI研究中心",
				"云计算服务平台",
				"开发者工具箱",
			).Map(func(name string) StorefrontPageData {
				return createTestData(name)
			}),
			// Mixed language names
			10: gen.OneConstOf(
				"Tech工作室",
				"Data分析Lab",
				"AI研究室",
				"开发者Studio",
			).Map(func(name string) StorefrontPageData {
				return createTestData(name)
			}),
			// Names with special characters
			8: gen.OneConstOf(
				"A&B工作室",
				"Tech<Data>",
				"分析\"专家\"",
				"Code & Design",
			).Map(func(name string) StorefrontPageData {
				return createTestData(name)
			}),
			// Very short names (single character)
			5: gen.OneConstOf("A", "技", "1", "店").Map(func(name string) StorefrontPageData {
				return createTestData(name)
			}),
			// Names near the 30 character limit
			4: gen.Const("这是一个非常非常长的店铺名称用于测试文本换行功能").Map(func(name string) StorefrontPageData {
				return createTestData(name)
			}),
			// Empty string (should display default "小铺")
			3: gen.Const("").Map(func(name string) StorefrontPageData {
				return createTestData(name)
			}),
		},
	)
}

// createTestData creates a StorefrontPageData with the given store name
func createTestData(storeName string) StorefrontPageData {
	return StorefrontPageData{
		Storefront: StorefrontInfo{
			ID:        1,
			UserID:    1,
			StoreName: storeName,
			StoreSlug: "test-store",
		},
		DefaultLang:     "zh-CN",
		ThemeCSS:        "",
		IsLoggedIn:      false,
		IsPreviewMode:   false,
		Sections:        []SectionConfig{},
		Packs:           []StorefrontPackInfo{},
		FeaturedPacks:   []StorefrontPackInfo{},
		PurchasedIDs:    make(map[int64]bool),
		Categories:      []string{},
		BannerData:      make(map[int]CustomBannerSettings),
		CustomProducts:  []CustomProduct{},
		PackGridColumns: 2,
		HeroLayout:      "default",
		FeaturedVisible: false,
		SupportApproved: false,
	}
}

// findElementByClass searches for an HTML element with the specified class
func findElementByClass(n *html.Node, className string) *html.Node {
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, className) {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findElementByClass(c, className); result != nil {
			return result
		}
	}
	return nil
}

// extractTextContent extracts all text content from an HTML node
func extractTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(extractTextContent(c))
	}
	return strings.TrimSpace(text.String())
}

// containsHTMLEscapedChars checks if the HTML contains properly escaped special characters
func containsHTMLEscapedChars(htmlStr string, originalText string) bool {
	// Check if special characters in original text are properly escaped in HTML
	if strings.Contains(originalText, "<") && !strings.Contains(htmlStr, "&lt;") {
		return false
	}
	if strings.Contains(originalText, ">") && !strings.Contains(htmlStr, "&gt;") {
		return false
	}
	if strings.Contains(originalText, "&") && !strings.Contains(htmlStr, "&amp;") {
		return false
	}
	if strings.Contains(originalText, "\"") && !strings.Contains(htmlStr, "&quot;") && !strings.Contains(htmlStr, "&#34;") {
		return false
	}
	return true
}

// **Validates: Requirements 1.1, 3.1, 3.4**
// Property 1: 店铺名称模板渲染
// For any valid storefront data, when rendering the storefront page template,
// the output HTML should contain a store name display block (class="store-name-header"),
// and the text content within that block should exactly match the input store name.
func TestProperty1_StorefrontNameRendering(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	properties.Property("storefront name is rendered correctly in HTML", prop.ForAll(
		func(data StorefrontPageData) bool {
			// Render the template
			var buf bytes.Buffer
			err := StorefrontTmpl.Execute(&buf, data)
			if err != nil {
				t.Logf("Template execution failed: %v", err)
				return false
			}

			htmlOutput := buf.String()

			// Parse the HTML
			doc, err := html.Parse(strings.NewReader(htmlOutput))
			if err != nil {
				t.Logf("HTML parsing failed: %v", err)
				return false
			}

			// Find the store-name-header element
			storeNameHeader := findElementByClass(doc, "store-name-header")
			if storeNameHeader == nil {
				t.Logf("store-name-header element not found in HTML")
				return false
			}

			// Extract the text content from the store-name-header
			actualText := extractTextContent(storeNameHeader)

			// Determine expected text (empty name should display "小铺")
			expectedText := data.Storefront.StoreName
			if expectedText == "" {
				expectedText = "小铺"
			}

			// Verify the text matches
			if actualText != expectedText {
				t.Logf("Store name mismatch: expected '%s', got '%s'", expectedText, actualText)
				return false
			}

			return true
		},
		genStorefrontData(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test edge cases specifically
func TestStorefrontNameEdgeCases(t *testing.T) {
	testCases := []struct {
		name          string
		storeName     string
		expectedText  string
		description   string
	}{
		{
			name:         "Empty store name",
			storeName:    "",
			expectedText: "小铺",
			description:  "Empty store name should display default text '小铺'",
		},
		{
			name:         "Single character",
			storeName:    "A",
			expectedText: "A",
			description:  "Single character store name should display correctly",
		},
		{
			name:         "Chinese single character",
			storeName:    "店",
			expectedText: "店",
			description:  "Single Chinese character should display correctly",
		},
		{
			name:         "Long store name",
			storeName:    "这是一个非常非常长的店铺名称用于测试文本换行功能",
			expectedText: "这是一个非常非常长的店铺名称用于测试文本换行功能",
			description:  "Long store name should display completely without truncation",
		},
		{
			name:         "HTML special characters",
			storeName:    "A&B <测试> \"引号\"",
			expectedText: "A&B <测试> \"引号\"",
			description:  "HTML special characters should be properly escaped and displayed",
		},
		{
			name:         "Mixed languages",
			storeName:    "日本語テスト 한국어 English",
			expectedText: "日本語テスト 한국어 English",
			description:  "Mixed language text should display correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := StorefrontPageData{
				Storefront: StorefrontInfo{
					ID:        1,
					UserID:    1,
					StoreName: tc.storeName,
					StoreSlug: "test-store",
				},
				DefaultLang:     "zh-CN",
				ThemeCSS:        "",
				IsLoggedIn:      false,
				IsPreviewMode:   false,
				Sections:        []SectionConfig{},
				Packs:           []StorefrontPackInfo{},
				FeaturedPacks:   []StorefrontPackInfo{},
				PurchasedIDs:    make(map[int64]bool),
				Categories:      []string{},
				BannerData:      make(map[int]CustomBannerSettings),
				CustomProducts:  []CustomProduct{},
				PackGridColumns: 2,
				HeroLayout:      "default",
				FeaturedVisible: false,
				SupportApproved: false,
			}

			// Render the template
			var buf bytes.Buffer
			err := StorefrontTmpl.Execute(&buf, data)
			if err != nil {
				t.Fatalf("Template execution failed: %v", err)
			}

			htmlOutput := buf.String()

			// Parse the HTML
			doc, err := html.Parse(strings.NewReader(htmlOutput))
			if err != nil {
				t.Fatalf("HTML parsing failed: %v", err)
			}

			// Find the store-name-header element
			storeNameHeader := findElementByClass(doc, "store-name-header")
			if storeNameHeader == nil {
				t.Fatal("store-name-header element not found in HTML")
			}

			// Extract the text content
			actualText := extractTextContent(storeNameHeader)

			// Verify the text matches
			if actualText != tc.expectedText {
				t.Errorf("%s: expected '%s', got '%s'", tc.description, tc.expectedText, actualText)
			}

			// For names with special characters, verify they are properly escaped in HTML
			if strings.ContainsAny(tc.storeName, "<>&\"") {
				if !containsHTMLEscapedChars(htmlOutput, tc.storeName) {
					t.Errorf("HTML special characters not properly escaped for store name: %s", tc.storeName)
				}
			}
		})
	}
}

// genStoreNameWithHTMLChars generates store names containing HTML special characters
func genStoreNameWithHTMLChars() gopter.Gen {
	return gen.Frequency(
		map[int]gopter.Gen{
			// Names with < and > characters
			25: gen.OneConstOf(
				"A<B>C",
				"<script>alert('test')</script>",
				"测试<标签>内容",
				"<div>店铺</div>",
				"Shop<Name>",
			),
			// Names with & character
			24: gen.OneConstOf(
				"A&B",
				"Rock & Roll",
				"技术&设计",
				"Code&Data",
				"A&B&C",
			),
			// Names with quotes
			23: gen.OneConstOf(
				"\"引号店铺\"",
				"Shop \"Name\"",
				"'单引号'",
				"Mixed \"double\" and 'single'",
			),
			// Names with multiple special characters
			18: gen.OneConstOf(
				"A&B <测试> \"引号\"",
				"<div>A&B</div>",
				"\"Shop\" & <Store>",
				"'Test' & \"Demo\"",
			),
			// Names with apostrophes (common in names)
			6: gen.OneConstOf(
				"John's Shop",
				"L'Atelier",
				"O'Brien's Store",
			),
			// Edge case: only special characters
			3: gen.OneConstOf(
				"<>",
				"&",
				"\"\"",
				"<&>",
			),
			// Mixed with Chinese/Japanese/Korean
			1: gen.OneConstOf(
				"技术<工作室>",
				"日本語\"テスト\"",
				"한국어&English",
			),
		},
	)
}

// genStoreNameWithMultilingual generates store names with multilingual text
func genStoreNameWithMultilingual() gopter.Gen {
	return gen.OneConstOf(
		// Chinese
		"技术分析工作室",
		"数据科学实验室",
		"人工智能研究中心",
		// Japanese
		"日本語テストショップ",
		"データ分析ラボ",
		"技術研究所",
		// Korean
		"한국어 테스트 상점",
		"데이터 분석 연구소",
		"기술 작업실",
		// Mixed Chinese + English
		"Tech技术工作室",
		"Data数据Lab",
		"AI人工智能Center",
		// Mixed Japanese + English
		"Techテストショップ",
		"Dataデータ分析",
		"AIラボLab",
		// Mixed Korean + English
		"Tech한국어Shop",
		"Data데이터Store",
		"AI연구소Lab",
		// All three + English
		"Tech技术テスト한국어",
		"日本語中文Korean",
		"多语言MultilingualStore",
		// With special Unicode characters
		"Café☕店铺",
		"🎨Art工作室",
		"Music🎵ショップ",
	)
}

// **Validates: Requirements 3.2**
// Property 2: HTML 特殊字符转义
// For any store name containing HTML special characters (such as <, >, &, ", ')
// or multilingual text, the rendered HTML should properly escape these characters
// so they are displayed as text content rather than being interpreted as HTML markup.
func TestProperty2_HTMLSpecialCharacterEscaping(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	properties.Property("HTML special characters are properly escaped", prop.ForAll(
		func(storeName string) bool {
			data := createTestData(storeName)

			// Render the template
			var buf bytes.Buffer
			err := StorefrontTmpl.Execute(&buf, data)
			if err != nil {
				t.Logf("Template execution failed: %v", err)
				return false
			}

			htmlOutput := buf.String()

			// Parse the HTML to ensure it's valid
			doc, err := html.Parse(strings.NewReader(htmlOutput))
			if err != nil {
				t.Logf("HTML parsing failed (indicates improper escaping): %v", err)
				return false
			}

			// Find the store-name-header element
			storeNameHeader := findElementByClass(doc, "store-name-header")
			if storeNameHeader == nil {
				t.Logf("store-name-header element not found in HTML")
				return false
			}

			// Extract the text content (this automatically decodes HTML entities)
			actualText := extractTextContent(storeNameHeader)

			// The extracted text should match the original store name exactly
			// because HTML entities should be decoded back to original characters
			if actualText != storeName {
				t.Logf("Text content mismatch: expected '%s', got '%s'", storeName, actualText)
				return false
			}

			// Verify that special characters are escaped in the raw HTML
			// Find the store-name-title section in raw HTML
			titleStart := strings.Index(htmlOutput, `class="store-name-title"`)
			if titleStart == -1 {
				t.Logf("store-name-title class not found in HTML")
				return false
			}

			// Extract a portion of HTML around the title for checking escaping
			titleSection := htmlOutput[titleStart : titleStart+min(500, len(htmlOutput)-titleStart)]

			// Check that special characters in the original name are properly escaped in HTML
			if strings.Contains(storeName, "<") {
				if !strings.Contains(titleSection, "&lt;") && !strings.Contains(titleSection, "&#60;") {
					t.Logf("'<' character not properly escaped in HTML")
					return false
				}
			}

			if strings.Contains(storeName, ">") {
				if !strings.Contains(titleSection, "&gt;") && !strings.Contains(titleSection, "&#62;") {
					t.Logf("'>' character not properly escaped in HTML")
					return false
				}
			}

			if strings.Contains(storeName, "&") {
				// Count & in original name
				ampCount := strings.Count(storeName, "&")
				// Count escaped versions in HTML (should have at least as many)
				escapedCount := strings.Count(titleSection, "&amp;") + strings.Count(titleSection, "&#38;")
				if escapedCount < ampCount {
					t.Logf("'&' character not properly escaped in HTML (expected at least %d, found %d)", ampCount, escapedCount)
					return false
				}
			}

			return true
		},
		genStoreNameWithHTMLChars(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Validates: Requirements 3.2**
// Property 2 (Multilingual): 多语言文本正确显示
// For any store name containing multilingual text (Chinese, Japanese, Korean, English mixed),
// the rendered HTML should correctly display the text with proper UTF-8 encoding.
func TestProperty2_MultilingualTextDisplay(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	properties.Property("multilingual text is displayed correctly", prop.ForAll(
		func(storeName string) bool {
			data := createTestData(storeName)

			// Render the template
			var buf bytes.Buffer
			err := StorefrontTmpl.Execute(&buf, data)
			if err != nil {
				t.Logf("Template execution failed: %v", err)
				return false
			}

			htmlOutput := buf.String()

			// Parse the HTML
			doc, err := html.Parse(strings.NewReader(htmlOutput))
			if err != nil {
				t.Logf("HTML parsing failed: %v", err)
				return false
			}

			// Find the store-name-header element
			storeNameHeader := findElementByClass(doc, "store-name-header")
			if storeNameHeader == nil {
				t.Logf("store-name-header element not found in HTML")
				return false
			}

			// Extract the text content
			actualText := extractTextContent(storeNameHeader)

			// The text should match exactly (UTF-8 should be preserved)
			if actualText != storeName {
				t.Logf("Multilingual text mismatch: expected '%s', got '%s'", storeName, actualText)
				return false
			}

			// Verify that the HTML output is valid UTF-8
			if !isValidUTF8(htmlOutput) {
				t.Logf("HTML output is not valid UTF-8")
				return false
			}

			return true
		},
		genStoreNameWithMultilingual(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// isValidUTF8 checks if a string is valid UTF-8
func isValidUTF8(s string) bool {
	// In Go, strings are always valid UTF-8 or the conversion would fail
	// This is a simple check that the string doesn't contain invalid sequences
	for _, r := range s {
		if r == '\uFFFD' {
			// Unicode replacement character indicates invalid UTF-8
			return false
		}
	}
	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Test specific HTML escaping edge cases
func TestHTMLEscapingEdgeCases(t *testing.T) {
	testCases := []struct {
		name         string
		storeName    string
		description  string
		checkEscaped map[string][]string // original char -> possible escaped forms
	}{
		{
			name:        "Script tag injection attempt",
			storeName:   "<script>alert('xss')</script>",
			description: "Script tags should be escaped to prevent XSS",
			checkEscaped: map[string][]string{
				"<": {"&lt;", "&#60;"},
				">": {"&gt;", "&#62;"},
			},
		},
		{
			name:        "HTML div tag",
			storeName:   "<div>店铺名称</div>",
			description: "HTML tags should be escaped",
			checkEscaped: map[string][]string{
				"<": {"&lt;", "&#60;"},
				">": {"&gt;", "&#62;"},
			},
		},
		{
			name:        "Ampersand in name",
			storeName:   "Rock & Roll Shop",
			description: "Ampersands should be escaped",
			checkEscaped: map[string][]string{
				"&": {"&amp;", "&#38;"},
			},
		},
		{
			name:        "Double quotes",
			storeName:   "The \"Best\" Shop",
			description: "Double quotes should be escaped",
			checkEscaped: map[string][]string{
				"\"": {"&quot;", "&#34;"},
			},
		},
		{
			name:        "Single quotes",
			storeName:   "John's Shop",
			description: "Single quotes should be escaped",
			checkEscaped: map[string][]string{
				"'": {"&#39;", "&apos;"},
			},
		},
		{
			name:        "Multiple special characters",
			storeName:   "A&B <测试> \"引号\"",
			description: "Multiple special characters should all be escaped",
			checkEscaped: map[string][]string{
				"&": {"&amp;", "&#38;"},
				"<": {"&lt;", "&#60;"},
				">": {"&gt;", "&#62;"},
				"\"": {"&quot;", "&#34;"},
			},
		},
		{
			name:        "Only special characters",
			storeName:   "<>&\"'",
			description: "String with only special characters should be fully escaped",
			checkEscaped: map[string][]string{
				"<": {"&lt;", "&#60;"},
				">": {"&gt;", "&#62;"},
				"&": {"&amp;", "&#38;"},
				"\"": {"&quot;", "&#34;"},
				"'": {"&#39;", "&apos;"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := createTestData(tc.storeName)

			// Render the template
			var buf bytes.Buffer
			err := StorefrontTmpl.Execute(&buf, data)
			if err != nil {
				t.Fatalf("Template execution failed: %v", err)
			}

			htmlOutput := buf.String()

			// Parse the HTML to ensure it's valid (proper escaping prevents parsing errors)
			doc, err := html.Parse(strings.NewReader(htmlOutput))
			if err != nil {
				t.Fatalf("HTML parsing failed (indicates improper escaping): %v", err)
			}

			// Find the store-name-header element
			storeNameHeader := findElementByClass(doc, "store-name-header")
			if storeNameHeader == nil {
				t.Fatal("store-name-header element not found in HTML")
			}

			// Extract the text content (HTML entities should be decoded)
			actualText := extractTextContent(storeNameHeader)

			// Verify the text matches the original (entities decoded correctly)
			if actualText != tc.storeName {
				t.Errorf("%s: text content mismatch: expected '%s', got '%s'", tc.description, tc.storeName, actualText)
			}

			// Find the store-name-title section in raw HTML
			titleStart := strings.Index(htmlOutput, `class="store-name-title"`)
			if titleStart == -1 {
				t.Fatal("store-name-title class not found in HTML")
			}

			// Extract a portion of HTML around the title
			titleSection := htmlOutput[titleStart : titleStart+min(500, len(htmlOutput)-titleStart)]

			// Check that each special character is properly escaped in the raw HTML
			for char, escapedForms := range tc.checkEscaped {
				if strings.Contains(tc.storeName, char) {
					found := false
					for _, escapedForm := range escapedForms {
						if strings.Contains(titleSection, escapedForm) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("%s: character '%s' not properly escaped (expected one of %v)", tc.description, char, escapedForms)
					}
				}
			}
		})
	}
}

// Test multilingual text edge cases
func TestMultilingualTextEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		storeName   string
		description string
	}{
		{
			name:        "Pure Chinese",
			storeName:   "技术分析工作室",
			description: "Pure Chinese text should display correctly",
		},
		{
			name:        "Pure Japanese",
			storeName:   "日本語テストショップ",
			description: "Pure Japanese text (Hiragana, Katakana, Kanji) should display correctly",
		},
		{
			name:        "Pure Korean",
			storeName:   "한국어 테스트 상점",
			description: "Pure Korean text should display correctly",
		},
		{
			name:        "Chinese + English",
			storeName:   "Tech技术工作室",
			description: "Mixed Chinese and English should display correctly",
		},
		{
			name:        "Japanese + English",
			storeName:   "Dataデータ分析Lab",
			description: "Mixed Japanese and English should display correctly",
		},
		{
			name:        "Korean + English",
			storeName:   "AI연구소Laboratory",
			description: "Mixed Korean and English should display correctly",
		},
		{
			name:        "All CJK languages mixed",
			storeName:   "技术テスト한국어Shop",
			description: "Mixed Chinese, Japanese, Korean, and English should display correctly",
		},
		{
			name:        "Unicode emoji",
			storeName:   "Café☕店铺🎨",
			description: "Unicode characters including emoji should display correctly",
		},
		{
			name:        "Special Unicode characters",
			storeName:   "Résumé™ Café® 店铺©",
			description: "Special Unicode characters (accents, symbols) should display correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := createTestData(tc.storeName)

			// Render the template
			var buf bytes.Buffer
			err := StorefrontTmpl.Execute(&buf, data)
			if err != nil {
				t.Fatalf("Template execution failed: %v", err)
			}

			htmlOutput := buf.String()

			// Verify the output is valid UTF-8
			if !isValidUTF8(htmlOutput) {
				t.Errorf("%s: HTML output is not valid UTF-8", tc.description)
			}

			// Parse the HTML
			doc, err := html.Parse(strings.NewReader(htmlOutput))
			if err != nil {
				t.Fatalf("HTML parsing failed: %v", err)
			}

			// Find the store-name-header element
			storeNameHeader := findElementByClass(doc, "store-name-header")
			if storeNameHeader == nil {
				t.Fatal("store-name-header element not found in HTML")
			}

			// Extract the text content
			actualText := extractTextContent(storeNameHeader)

			// Verify the text matches exactly
			if actualText != tc.storeName {
				t.Errorf("%s: expected '%s', got '%s'", tc.description, tc.storeName, actualText)
			}

			// Verify that the original characters are preserved in the HTML
			// (they should not be unnecessarily entity-encoded)
			if !strings.Contains(htmlOutput, tc.storeName) {
				// Some characters might be escaped, so check if the decoded version matches
				t.Logf("%s: original text not found directly in HTML (may be entity-encoded)", tc.description)
			}
		})
	}
}

// Test that the store-name-header element has the correct structure
func TestStorefrontNameHeaderStructure(t *testing.T) {
	data := StorefrontPageData{
		Storefront: StorefrontInfo{
			ID:        1,
			UserID:    1,
			StoreName: "测试店铺",
			StoreSlug: "test-store",
		},
		DefaultLang:     "zh-CN",
		ThemeCSS:        "",
		IsLoggedIn:      false,
		IsPreviewMode:   false,
		Sections:        []SectionConfig{},
		Packs:           []StorefrontPackInfo{},
		FeaturedPacks:   []StorefrontPackInfo{},
		PurchasedIDs:    make(map[int64]bool),
		Categories:      []string{},
		BannerData:      make(map[int]CustomBannerSettings),
		CustomProducts:  []CustomProduct{},
		PackGridColumns: 2,
		HeroLayout:      "default",
		FeaturedVisible: false,
		SupportApproved: false,
	}

	// Render the template
	var buf bytes.Buffer
	err := StorefrontTmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	htmlOutput := buf.String()

	// Parse the HTML
	doc, err := html.Parse(strings.NewReader(htmlOutput))
	if err != nil {
		t.Fatalf("HTML parsing failed: %v", err)
	}

	// Find the store-name-header element
	storeNameHeader := findElementByClass(doc, "store-name-header")
	if storeNameHeader == nil {
		t.Fatal("store-name-header element not found in HTML")
	}

	// Verify it's a div element
	if storeNameHeader.Data != "div" {
		t.Errorf("store-name-header should be a div element, got: %s", storeNameHeader.Data)
	}

	// Find the h1 element with class store-name-title inside
	var h1Found bool
	for c := storeNameHeader.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "h1" {
			for _, attr := range c.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "store-name-title") {
					h1Found = true
					break
				}
			}
		}
	}

	if !h1Found {
		t.Error("store-name-header should contain an h1 element with class store-name-title")
	}
}
