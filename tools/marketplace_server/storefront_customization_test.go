package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"marketplace_server/templates"
)

// TestValidateLayoutConfig_ValidDefault tests that the default layout config passes validation.
func TestValidateLayoutConfig_ValidDefault(t *testing.T) {
	config := DefaultLayoutConfig()
	jsonStr, err := SerializeLayoutConfig(config)
	if err != nil {
		t.Fatalf("failed to serialize default config: %v", err)
	}
	result := ValidateLayoutConfig(jsonStr)
	if result != "" {
		t.Errorf("expected empty string for valid default config, got: %s", result)
	}
}

// TestValidateLayoutConfig_InvalidJSON tests that invalid JSON is rejected.
func TestValidateLayoutConfig_InvalidJSON(t *testing.T) {
	result := ValidateLayoutConfig("{not valid json")
	if result != "布局配置 JSON 格式无效" {
		t.Errorf("expected '布局配置 JSON 格式无效', got: %s", result)
	}
}

// TestValidateLayoutConfig_EmptySections tests that empty sections array is rejected.
func TestValidateLayoutConfig_EmptySections(t *testing.T) {
	result := ValidateLayoutConfig(`{"sections":[]}`)
	if result != "布局配置必须包含至少一个区块" {
		t.Errorf("expected '布局配置必须包含至少一个区块', got: %s", result)
	}
}

// TestValidateLayoutConfig_UnsupportedSectionType tests that unknown section types are rejected.
func TestValidateLayoutConfig_UnsupportedSectionType(t *testing.T) {
	config := `{"sections":[{"type":"hero","visible":true,"settings":{}},{"type":"unknown_type","visible":true,"settings":{}},{"type":"pack_grid","visible":true,"settings":{}}]}`
	result := ValidateLayoutConfig(config)
	if !strings.Contains(result, "不支持的区块类型") {
		t.Errorf("expected error about unsupported section type, got: %s", result)
	}
}

// TestValidateLayoutConfig_MissingHero tests that missing hero section is rejected.
func TestValidateLayoutConfig_MissingHero(t *testing.T) {
	config := `{"sections":[{"type":"pack_grid","visible":true,"settings":{}}]}`
	result := ValidateLayoutConfig(config)
	if result != "布局配置必须包含 hero 区块" {
		t.Errorf("expected '布局配置必须包含 hero 区块', got: %s", result)
	}
}

// TestValidateLayoutConfig_MissingPackGrid tests that missing pack_grid section is rejected.
func TestValidateLayoutConfig_MissingPackGrid(t *testing.T) {
	config := `{"sections":[{"type":"hero","visible":true,"settings":{}}]}`
	result := ValidateLayoutConfig(config)
	if result != "布局配置必须包含 pack_grid 区块" {
		t.Errorf("expected '布局配置必须包含 pack_grid 区块', got: %s", result)
	}
}

// TestValidateLayoutConfig_MultipleHero tests that multiple hero sections are rejected.
func TestValidateLayoutConfig_MultipleHero(t *testing.T) {
	config := `{"sections":[{"type":"hero","visible":true,"settings":{}},{"type":"hero","visible":true,"settings":{}},{"type":"pack_grid","visible":true,"settings":{}}]}`
	result := ValidateLayoutConfig(config)
	if result != "hero 区块只能有一个" {
		t.Errorf("expected 'hero 区块只能有一个', got: %s", result)
	}
}

// TestValidateLayoutConfig_MultiplePackGrid tests that multiple pack_grid sections are rejected.
func TestValidateLayoutConfig_MultiplePackGrid(t *testing.T) {
	config := `{"sections":[{"type":"hero","visible":true,"settings":{}},{"type":"pack_grid","visible":true,"settings":{}},{"type":"pack_grid","visible":true,"settings":{}}]}`
	result := ValidateLayoutConfig(config)
	if result != "pack_grid 区块只能有一个" {
		t.Errorf("expected 'pack_grid 区块只能有一个', got: %s", result)
	}
}

// TestValidateLayoutConfig_HeroHidden tests that hidden hero is rejected.
func TestValidateLayoutConfig_HeroHidden(t *testing.T) {
	config := `{"sections":[{"type":"hero","visible":false,"settings":{}},{"type":"pack_grid","visible":true,"settings":{}}]}`
	result := ValidateLayoutConfig(config)
	if result != "hero 区块不允许隐藏" {
		t.Errorf("expected 'hero 区块不允许隐藏', got: %s", result)
	}
}

// TestValidateLayoutConfig_PackGridHidden tests that hidden pack_grid is rejected.
func TestValidateLayoutConfig_PackGridHidden(t *testing.T) {
	config := `{"sections":[{"type":"hero","visible":true,"settings":{}},{"type":"pack_grid","visible":false,"settings":{}}]}`
	result := ValidateLayoutConfig(config)
	if result != "pack_grid 区块不允许隐藏" {
		t.Errorf("expected 'pack_grid 区块不允许隐藏', got: %s", result)
	}
}

// TestValidateLayoutConfig_TooManyBanners tests that more than 3 custom_banner sections are rejected.
func TestValidateLayoutConfig_TooManyBanners(t *testing.T) {
	config := `{"sections":[
		{"type":"hero","visible":true,"settings":{}},
		{"type":"pack_grid","visible":true,"settings":{}},
		{"type":"custom_banner","visible":true,"settings":{"text":"a","style":"info"}},
		{"type":"custom_banner","visible":true,"settings":{"text":"b","style":"info"}},
		{"type":"custom_banner","visible":true,"settings":{"text":"c","style":"info"}},
		{"type":"custom_banner","visible":true,"settings":{"text":"d","style":"info"}}
	]}`
	result := ValidateLayoutConfig(config)
	if result != "最多添加 3 个自定义横幅" {
		t.Errorf("expected '最多添加 3 个自定义横幅', got: %s", result)
	}
}

// TestValidateLayoutConfig_BannerTextTooLong tests that banner text over 200 chars is rejected.
func TestValidateLayoutConfig_BannerTextTooLong(t *testing.T) {
	longText := strings.Repeat("测", 201)
	settings, _ := json.Marshal(CustomBannerSettings{Text: longText, Style: "info"})
	config := LayoutConfig{
		Sections: []SectionConfig{
			{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "pack_grid", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "custom_banner", Visible: true, Settings: settings},
		},
	}
	jsonStr, _ := SerializeLayoutConfig(config)
	result := ValidateLayoutConfig(jsonStr)
	if result != "横幅文本不能超过 200 字符" {
		t.Errorf("expected '横幅文本不能超过 200 字符', got: %s", result)
	}
}

// TestValidateLayoutConfig_InvalidBannerStyle tests that invalid banner style is rejected.
func TestValidateLayoutConfig_InvalidBannerStyle(t *testing.T) {
	settings, _ := json.Marshal(CustomBannerSettings{Text: "hello", Style: "danger"})
	config := LayoutConfig{
		Sections: []SectionConfig{
			{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "pack_grid", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "custom_banner", Visible: true, Settings: settings},
		},
	}
	jsonStr, _ := SerializeLayoutConfig(config)
	result := ValidateLayoutConfig(jsonStr)
	if !strings.Contains(result, "不支持的横幅样式") {
		t.Errorf("expected error about unsupported banner style, got: %s", result)
	}
}

// TestValidateLayoutConfig_InvalidPackGridColumns tests that invalid columns value is rejected.
func TestValidateLayoutConfig_InvalidPackGridColumns(t *testing.T) {
	settings, _ := json.Marshal(PackGridSettings{Columns: 5})
	config := LayoutConfig{
		Sections: []SectionConfig{
			{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "pack_grid", Visible: true, Settings: settings},
		},
	}
	jsonStr, _ := SerializeLayoutConfig(config)
	result := ValidateLayoutConfig(jsonStr)
	if result != "列数必须为 1、2 或 3" {
		t.Errorf("expected '列数必须为 1、2 或 3', got: %s", result)
	}
}

// TestValidateLayoutConfig_ValidPackGridColumns tests that valid columns values pass.
func TestValidateLayoutConfig_ValidPackGridColumns(t *testing.T) {
	for _, cols := range []int{0, 1, 2, 3} {
		settings, _ := json.Marshal(PackGridSettings{Columns: cols})
		config := LayoutConfig{
			Sections: []SectionConfig{
				{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
				{Type: "pack_grid", Visible: true, Settings: settings},
			},
		}
		jsonStr, _ := SerializeLayoutConfig(config)
		result := ValidateLayoutConfig(jsonStr)
		if result != "" {
			t.Errorf("columns=%d: expected empty string, got: %s", cols, result)
		}
	}
}

// TestValidateLayoutConfig_ValidBannerStyles tests that all valid banner styles pass.
func TestValidateLayoutConfig_ValidBannerStyles(t *testing.T) {
	for _, style := range []string{"info", "success", "warning"} {
		settings, _ := json.Marshal(CustomBannerSettings{Text: "hello", Style: style})
		config := LayoutConfig{
			Sections: []SectionConfig{
				{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
				{Type: "pack_grid", Visible: true, Settings: json.RawMessage("{}")},
				{Type: "custom_banner", Visible: true, Settings: settings},
			},
		}
		jsonStr, _ := SerializeLayoutConfig(config)
		result := ValidateLayoutConfig(jsonStr)
		if result != "" {
			t.Errorf("style=%s: expected empty string, got: %s", style, result)
		}
	}
}

// TestValidateLayoutConfig_ThreeBannersOK tests that exactly 3 custom_banner sections pass.
func TestValidateLayoutConfig_ThreeBannersOK(t *testing.T) {
	config := `{"sections":[
		{"type":"hero","visible":true,"settings":{}},
		{"type":"pack_grid","visible":true,"settings":{}},
		{"type":"custom_banner","visible":true,"settings":{"text":"a","style":"info"}},
		{"type":"custom_banner","visible":true,"settings":{"text":"b","style":"success"}},
		{"type":"custom_banner","visible":true,"settings":{"text":"c","style":"warning"}}
	]}`
	result := ValidateLayoutConfig(config)
	if result != "" {
		t.Errorf("expected empty string for 3 banners, got: %s", result)
	}
}

// Feature: storefront-customization, Property 1: 布局配置序列化 Round-Trip
// **Validates: Requirements 6.7**
//
// For any valid LayoutConfig, serializing to JSON and parsing back produces
// a semantically equivalent struct.
func TestProperty1_LayoutConfigRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		config := generateValidLayoutConfig(rng)

		// Serialize
		jsonStr, err := SerializeLayoutConfig(config)
		if err != nil {
			t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
			return false
		}

		// Verify the serialized config passes validation
		if errMsg := ValidateLayoutConfig(jsonStr); errMsg != "" {
			t.Logf("FAIL: generated config failed validation: %s (json: %s)", errMsg, jsonStr)
			return false
		}

		// Parse back
		parsed, err := ParseLayoutConfig(jsonStr)
		if err != nil {
			t.Logf("FAIL: ParseLayoutConfig failed: %v (json: %s)", err, jsonStr)
			return false
		}

		// Verify semantic equivalence
		if len(parsed.Sections) != len(config.Sections) {
			t.Logf("FAIL: section count mismatch: original=%d, parsed=%d",
				len(config.Sections), len(parsed.Sections))
			return false
		}

		for i, origSection := range config.Sections {
			parsedSection := parsed.Sections[i]

			if origSection.Type != parsedSection.Type {
				t.Logf("FAIL: section[%d] type mismatch: original=%q, parsed=%q",
					i, origSection.Type, parsedSection.Type)
				return false
			}

			if origSection.Visible != parsedSection.Visible {
				t.Logf("FAIL: section[%d] visible mismatch: original=%v, parsed=%v",
					i, origSection.Visible, parsedSection.Visible)
				return false
			}

			// Compare settings by unmarshaling both to generic maps
			var origSettings, parsedSettings interface{}
			if err := json.Unmarshal(origSection.Settings, &origSettings); err != nil {
				t.Logf("FAIL: failed to unmarshal original settings[%d]: %v", i, err)
				return false
			}
			if err := json.Unmarshal(parsedSection.Settings, &parsedSettings); err != nil {
				t.Logf("FAIL: failed to unmarshal parsed settings[%d]: %v", i, err)
				return false
			}
			if !reflect.DeepEqual(origSettings, parsedSettings) {
				t.Logf("FAIL: section[%d] settings mismatch: original=%s, parsed=%s",
					i, string(origSection.Settings), string(parsedSection.Settings))
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 violated: %v", err)
	}
}

// generateValidLayoutConfig creates a random valid LayoutConfig for property testing.
// It always includes exactly one hero and one pack_grid (both visible), and randomly
// adds optional sections (featured, filter_bar, custom_banner up to 3).
func generateValidLayoutConfig(rng *rand.Rand) LayoutConfig {
	var sections []SectionConfig

	// Always include hero (must be visible)
	sections = append(sections, SectionConfig{
		Type:     "hero",
		Visible:  true,
		Settings: json.RawMessage("{}"),
	})

	// Randomly include featured section
	if rng.Intn(2) == 0 {
		sections = append(sections, SectionConfig{
			Type:     "featured",
			Visible:  rng.Intn(2) == 0,
			Settings: json.RawMessage("{}"),
		})
	}

	// Randomly include filter_bar section
	if rng.Intn(2) == 0 {
		sections = append(sections, SectionConfig{
			Type:     "filter_bar",
			Visible:  rng.Intn(2) == 0,
			Settings: json.RawMessage("{}"),
		})
	}

	// Randomly add 0-3 custom_banner sections
	numBanners := rng.Intn(4) // 0, 1, 2, or 3
	validStyles := []string{"info", "success", "warning"}
	for i := 0; i < numBanners; i++ {
		textLen := rng.Intn(201) // 0-200 characters
		textRunes := make([]rune, textLen)
		chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789 你好世界")
		for j := range textRunes {
			textRunes[j] = chars[rng.Intn(len(chars))]
		}
		style := validStyles[rng.Intn(len(validStyles))]
		settings, _ := json.Marshal(CustomBannerSettings{
			Text:  string(textRunes),
			Style: style,
		})
		sections = append(sections, SectionConfig{
			Type:     "custom_banner",
			Visible:  rng.Intn(2) == 0,
			Settings: settings,
		})
	}

	// Always include pack_grid (must be visible) with valid columns
	validColumns := []int{0, 1, 2, 3} // 0 means default (unset)
	columns := validColumns[rng.Intn(len(validColumns))]
	packGridSettings, _ := json.Marshal(PackGridSettings{Columns: columns})
	sections = append(sections, SectionConfig{
		Type:     "pack_grid",
		Visible:  true,
		Settings: packGridSettings,
	})

	// Shuffle the sections to randomize order (but keep hero and pack_grid present)
	rng.Shuffle(len(sections), func(i, j int) {
		sections[i], sections[j] = sections[j], sections[i]
	})

	return LayoutConfig{Sections: sections}
}

// Feature: storefront-customization, Property 3: 验证函数拒绝无效配置
// **Validates: Requirements 1.5, 5.2, 6.1, 6.2, 6.3, 6.4**
//
// For any config with an invalid section type, missing hero, missing pack_grid,
// or more than 3 custom_banner sections, ValidateLayoutConfig returns a non-empty error.
func TestProperty3_ValidationRejectsInvalidConfig(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Category 1: Config with an invalid/unknown section type
	t.Run("InvalidSectionType", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))
			config := generateValidLayoutConfig(rng)

			// Pick a random section and replace its type with an invalid one
			invalidTypes := []string{"unknown", "banner", "sidebar", "footer", "header", "widget", "ad_block"}
			invalidType := invalidTypes[rng.Intn(len(invalidTypes))]
			idx := rng.Intn(len(config.Sections))
			config.Sections[idx].Type = invalidType

			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			result := ValidateLayoutConfig(jsonStr)
			if result == "" {
				t.Logf("FAIL: expected non-empty error for invalid section type %q, got empty string (json: %s)", invalidType, jsonStr)
				return false
			}
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 3 (InvalidSectionType) violated: %v", err)
		}
	})

	// Category 2: Config missing the hero section
	t.Run("MissingHero", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))
			config := generateValidLayoutConfig(rng)

			// Remove all hero sections
			var filtered []SectionConfig
			for _, s := range config.Sections {
				if s.Type != "hero" {
					filtered = append(filtered, s)
				}
			}
			// Ensure we still have at least one section (pack_grid should remain)
			if len(filtered) == 0 {
				filtered = append(filtered, SectionConfig{
					Type:     "pack_grid",
					Visible:  true,
					Settings: json.RawMessage(`{}`),
				})
			}
			config.Sections = filtered

			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			result := ValidateLayoutConfig(jsonStr)
			if result == "" {
				t.Logf("FAIL: expected non-empty error for missing hero, got empty string (json: %s)", jsonStr)
				return false
			}
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 3 (MissingHero) violated: %v", err)
		}
	})

	// Category 3: Config missing the pack_grid section
	t.Run("MissingPackGrid", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))
			config := generateValidLayoutConfig(rng)

			// Remove all pack_grid sections
			var filtered []SectionConfig
			for _, s := range config.Sections {
				if s.Type != "pack_grid" {
					filtered = append(filtered, s)
				}
			}
			// Ensure we still have at least one section (hero should remain)
			if len(filtered) == 0 {
				filtered = append(filtered, SectionConfig{
					Type:     "hero",
					Visible:  true,
					Settings: json.RawMessage(`{}`),
				})
			}
			config.Sections = filtered

			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			result := ValidateLayoutConfig(jsonStr)
			if result == "" {
				t.Logf("FAIL: expected non-empty error for missing pack_grid, got empty string (json: %s)", jsonStr)
				return false
			}
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 3 (MissingPackGrid) violated: %v", err)
		}
	})

	// Category 4: Config with more than 3 custom_banner sections
	t.Run("TooManyBanners", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))

			// Start with hero + pack_grid
			sections := []SectionConfig{
				{Type: "hero", Visible: true, Settings: json.RawMessage(`{}`)},
				{Type: "pack_grid", Visible: true, Settings: json.RawMessage(`{}`)},
			}

			// Add 4-7 custom_banner sections (always > 3)
			numBanners := 4 + rng.Intn(4)
			validStyles := []string{"info", "success", "warning"}
			for i := 0; i < numBanners; i++ {
				textLen := rng.Intn(101) // keep text short enough to be valid
				textRunes := make([]rune, textLen)
				chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789 ")
				for j := range textRunes {
					textRunes[j] = chars[rng.Intn(len(chars))]
				}
				style := validStyles[rng.Intn(len(validStyles))]
				settings, _ := json.Marshal(CustomBannerSettings{
					Text:  string(textRunes),
					Style: style,
				})
				sections = append(sections, SectionConfig{
					Type:     "custom_banner",
					Visible:  rng.Intn(2) == 0,
					Settings: settings,
				})
			}

			// Shuffle to randomize order
			rng.Shuffle(len(sections), func(i, j int) {
				sections[i], sections[j] = sections[j], sections[i]
			})

			config := LayoutConfig{Sections: sections}
			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			result := ValidateLayoutConfig(jsonStr)
			if result == "" {
				t.Logf("FAIL: expected non-empty error for %d custom_banners, got empty string (json: %s)", numBanners, jsonStr)
				return false
			}
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 3 (TooManyBanners) violated: %v", err)
		}
	})
}


// Feature: storefront-customization, Property 4: 必需区块不可隐藏
// **Validates: Requirements 2.6, 2.7**
//
// For any layout config where hero or pack_grid has visible=false,
// ValidateLayoutConfig must return a non-empty error.
func TestProperty4_RequiredSectionsNotHideable(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Category 1: hero visible=false, pack_grid visible=true
	t.Run("HeroHidden", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))
			config := generateValidLayoutConfig(rng)

			// Set hero visible=false, ensure pack_grid visible=true
			for i := range config.Sections {
				if config.Sections[i].Type == "hero" {
					config.Sections[i].Visible = false
				}
				if config.Sections[i].Type == "pack_grid" {
					config.Sections[i].Visible = true
				}
			}

			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			result := ValidateLayoutConfig(jsonStr)
			if result == "" {
				t.Logf("FAIL: expected non-empty error for hidden hero, got empty string (json: %s)", jsonStr)
				return false
			}
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 4 (HeroHidden) violated: %v", err)
		}
	})

	// Category 2: pack_grid visible=false, hero visible=true
	t.Run("PackGridHidden", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))
			config := generateValidLayoutConfig(rng)

			// Set pack_grid visible=false, ensure hero visible=true
			for i := range config.Sections {
				if config.Sections[i].Type == "pack_grid" {
					config.Sections[i].Visible = false
				}
				if config.Sections[i].Type == "hero" {
					config.Sections[i].Visible = true
				}
			}

			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			result := ValidateLayoutConfig(jsonStr)
			if result == "" {
				t.Logf("FAIL: expected non-empty error for hidden pack_grid, got empty string (json: %s)", jsonStr)
				return false
			}
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 4 (PackGridHidden) violated: %v", err)
		}
	})

	// Category 3: both hero and pack_grid visible=false
	t.Run("BothHidden", func(t *testing.T) {
		f := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))
			config := generateValidLayoutConfig(rng)

			// Set both hero and pack_grid visible=false
			for i := range config.Sections {
				if config.Sections[i].Type == "hero" || config.Sections[i].Type == "pack_grid" {
					config.Sections[i].Visible = false
				}
			}

			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			result := ValidateLayoutConfig(jsonStr)
			if result == "" {
				t.Logf("FAIL: expected non-empty error for both hidden, got empty string (json: %s)", jsonStr)
				return false
			}
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 4 (BothHidden) violated: %v", err)
		}
	})
}

// Feature: storefront-customization, Property 7: 主题 CSS 变量完整性与正确性
// **Validates: Requirements 3.5, 3.6, 8.4**
//
// For all valid themes, GetThemeCSS returns a string containing all 5 required
// CSS variable names: --primary-color, --primary-hover, --hero-gradient,
// --accent-color, --card-border.
func TestProperty7_ThemeCSSCompleteness(t *testing.T) {
	requiredVars := []string{
		"--primary-color",
		"--primary-hover",
		"--hero-gradient",
		"--accent-color",
		"--card-border",
	}

	for theme := range ValidThemes {
		t.Run(theme, func(t *testing.T) {
			css := GetThemeCSS(theme)
			if css == "" {
				t.Errorf("GetThemeCSS(%q) returned empty string", theme)
				return
			}
			for _, varName := range requiredVars {
				if !strings.Contains(css, varName) {
					t.Errorf("GetThemeCSS(%q) missing CSS variable %q; got: %s", theme, varName, css)
				}
			}
		})
	}
}

// Feature: storefront-customization, Property 8: 无效主题回退到默认
// **Validates: Requirements 3.7**
//
// For all strings that are NOT in ValidThemes, GetThemeCSS should return
// the same result as GetThemeCSS("default").
func TestProperty8_InvalidThemeFallback(t *testing.T) {
	defaultCSS := GetThemeCSS("default")

	f := func(theme string) bool {
		if ValidThemes[theme] {
			// Skip valid themes — we only test invalid ones
			return true
		}
		got := GetThemeCSS(theme)
		return got == defaultCSS
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Values: func(values []reflect.Value, rng *rand.Rand) {
			// Generate random strings that are guaranteed NOT to be valid themes
			const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-!@#$%^&* "
			for {
				length := rng.Intn(20) + 1
				b := make([]byte, length)
				for i := range b {
					b[i] = chars[rng.Intn(len(chars))]
				}
				s := string(b)
				if !ValidThemes[s] {
					values[0] = reflect.ValueOf(s)
					return
				}
			}
		},
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 8 failed: %v", err)
	}
}

// Feature: storefront-customization, Property 2: 布局保存 API Round-Trip
// **Validates: Requirements 2.3, 4.2, 5.5**
//
// For any valid layout config JSON string, saving it via POST /user/storefront/layout
// and then reading it back from the database produces a semantically equivalent config.
func TestProperty2_LayoutSaveAPIRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		cleanup := setupTestDB(t)
		defer cleanup()

		rng := rand.New(rand.NewSource(seed))

		// Create a test user and storefront
		userID := createTestUserWithBalance(t, 0)
		slug := fmt.Sprintf("layout-rt-%d-%d", userID, rng.Int63n(1000000))
		_, err := db.Exec(
			"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
			userID, slug,
		)
		if err != nil {
			t.Logf("FAIL: failed to create storefront: %v", err)
			return false
		}

		// Generate a random valid layout config
		config := generateValidLayoutConfig(rng)
		jsonStr, err := SerializeLayoutConfig(config)
		if err != nil {
			t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
			return false
		}

		// Verify the generated config is valid
		if errMsg := ValidateLayoutConfig(jsonStr); errMsg != "" {
			t.Logf("FAIL: generated config failed validation: %s", errMsg)
			return false
		}

		// POST the layout config to the save handler
		form := url.Values{}
		form.Set("layout_config", jsonStr)
		req := httptest.NewRequest(http.MethodPost, "/user/storefront/layout", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		rr := httptest.NewRecorder()
		handleStorefrontSaveLayout(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("FAIL: save layout returned status %d, body: %s", rr.Code, rr.Body.String())
			return false
		}

		// Verify the response is {"ok": true}
		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Logf("FAIL: failed to parse response JSON: %v", err)
			return false
		}
		if resp["ok"] != true {
			t.Logf("FAIL: expected ok=true, got: %v (error: %v)", resp["ok"], resp["error"])
			return false
		}

		// Read back from the database
		var storedLayoutConfig string
		err = db.QueryRow(
			"SELECT layout_config FROM author_storefronts WHERE user_id = ?",
			userID,
		).Scan(&storedLayoutConfig)
		if err != nil {
			t.Logf("FAIL: failed to read layout_config from DB: %v", err)
			return false
		}

		// Parse the stored config
		storedConfig, err := ParseLayoutConfig(storedLayoutConfig)
		if err != nil {
			t.Logf("FAIL: failed to parse stored layout_config: %v (raw: %s)", err, storedLayoutConfig)
			return false
		}

		// Verify semantic equivalence: same number of sections
		if len(storedConfig.Sections) != len(config.Sections) {
			t.Logf("FAIL: section count mismatch: original=%d, stored=%d",
				len(config.Sections), len(storedConfig.Sections))
			return false
		}

		// Verify each section matches
		for i, origSection := range config.Sections {
			storedSection := storedConfig.Sections[i]

			if origSection.Type != storedSection.Type {
				t.Logf("FAIL: section[%d] type mismatch: original=%q, stored=%q",
					i, origSection.Type, storedSection.Type)
				return false
			}

			if origSection.Visible != storedSection.Visible {
				t.Logf("FAIL: section[%d] visible mismatch: original=%v, stored=%v",
					i, origSection.Visible, storedSection.Visible)
				return false
			}

			// Compare settings by unmarshaling both to generic maps
			var origSettings, storedSettings interface{}
			if err := json.Unmarshal(origSection.Settings, &origSettings); err != nil {
				t.Logf("FAIL: failed to unmarshal original settings[%d]: %v", i, err)
				return false
			}
			if err := json.Unmarshal(storedSection.Settings, &storedSettings); err != nil {
				t.Logf("FAIL: failed to unmarshal stored settings[%d]: %v", i, err)
				return false
			}
			if !reflect.DeepEqual(origSettings, storedSettings) {
				t.Logf("FAIL: section[%d] settings mismatch: original=%s, stored=%s",
					i, string(origSection.Settings), string(storedSection.Settings))
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 violated: %v", err)
	}
}

// Feature: storefront-customization, Property 9: 布局 API 响应正确性
// **Validates: Requirements 6.5, 7.3, 7.5**
//
// For any layout config JSON string, POST /user/storefront/layout should:
// - If config is valid: return {"ok": true} and update DB layout_config
// - If config is invalid: return {"ok": false, "error": "..."} and DB layout_config remains unchanged
func TestProperty9_LayoutAPIResponseCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Sub-test 1: Valid configs should return ok:true and update DB
	t.Run("ValidConfigAccepted", func(t *testing.T) {
		f := func(seed int64) bool {
			cleanup := setupTestDB(t)
			defer cleanup()

			rng := rand.New(rand.NewSource(seed))

			// Create a test user and storefront
			userID := createTestUserWithBalance(t, 0)
			slug := fmt.Sprintf("p9-valid-%d-%d", userID, rng.Int63n(1000000))
			_, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug) VALUES (?, ?)",
				userID, slug,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}

			// Generate a random valid layout config
			config := generateValidLayoutConfig(rng)
			jsonStr, err := SerializeLayoutConfig(config)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			// Verify the generated config is actually valid
			if errMsg := ValidateLayoutConfig(jsonStr); errMsg != "" {
				t.Logf("FAIL: generated config failed validation: %s", errMsg)
				return false
			}

			// POST the layout config
			form := url.Values{}
			form.Set("layout_config", jsonStr)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/layout", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSaveLayout(rr, req)

			// Parse response
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response JSON: %v (body: %s)", err, rr.Body.String())
				return false
			}

			// Verify response has ok:true
			if resp["ok"] != true {
				t.Logf("FAIL: expected ok=true for valid config, got: %v (error: %v)", resp["ok"], resp["error"])
				return false
			}

			// Verify database was updated
			var storedLayoutConfig string
			err = db.QueryRow(
				"SELECT layout_config FROM author_storefronts WHERE user_id = ?",
				userID,
			).Scan(&storedLayoutConfig)
			if err != nil {
				t.Logf("FAIL: failed to read layout_config from DB: %v", err)
				return false
			}

			// Parse stored config and verify semantic equivalence
			storedConfig, err := ParseLayoutConfig(storedLayoutConfig)
			if err != nil {
				t.Logf("FAIL: failed to parse stored layout_config: %v", err)
				return false
			}

			if len(storedConfig.Sections) != len(config.Sections) {
				t.Logf("FAIL: section count mismatch: original=%d, stored=%d",
					len(config.Sections), len(storedConfig.Sections))
				return false
			}

			for i, origSection := range config.Sections {
				storedSection := storedConfig.Sections[i]
				if origSection.Type != storedSection.Type || origSection.Visible != storedSection.Visible {
					t.Logf("FAIL: section[%d] mismatch: orig={%s,%v} stored={%s,%v}",
						i, origSection.Type, origSection.Visible, storedSection.Type, storedSection.Visible)
					return false
				}
			}

			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 9 (ValidConfigAccepted) violated: %v", err)
		}
	})

	// Sub-test 2: Invalid configs (missing hero) should return ok:false and DB unchanged
	t.Run("InvalidConfigMissingHero", func(t *testing.T) {
		f := func(seed int64) bool {
			cleanup := setupTestDB(t)
			defer cleanup()

			rng := rand.New(rand.NewSource(seed))

			// Create a test user and storefront with an initial valid layout
			userID := createTestUserWithBalance(t, 0)
			slug := fmt.Sprintf("p9-nohero-%d-%d", userID, rng.Int63n(1000000))

			initialConfig := generateValidLayoutConfig(rng)
			initialJSON, err := SerializeLayoutConfig(initialConfig)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}
			// Ensure initial config is valid
			if errMsg := ValidateLayoutConfig(initialJSON); errMsg != "" {
				t.Logf("FAIL: initial config failed validation: %s", errMsg)
				return false
			}

			_, err = db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, layout_config) VALUES (?, ?, ?)",
				userID, slug, initialJSON,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}

			// Generate an invalid config by removing hero
			invalidConfig := generateValidLayoutConfig(rng)
			var filtered []SectionConfig
			for _, s := range invalidConfig.Sections {
				if s.Type != "hero" {
					filtered = append(filtered, s)
				}
			}
			if len(filtered) == 0 {
				filtered = append(filtered, SectionConfig{
					Type: "pack_grid", Visible: true, Settings: json.RawMessage(`{}`),
				})
			}
			invalidConfig.Sections = filtered

			invalidJSON, err := SerializeLayoutConfig(invalidConfig)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			// POST the invalid config
			form := url.Values{}
			form.Set("layout_config", invalidJSON)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/layout", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSaveLayout(rr, req)

			// Parse response
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response JSON: %v", err)
				return false
			}

			// Verify response has ok:false with non-empty error
			if resp["ok"] != false {
				t.Logf("FAIL: expected ok=false for invalid config (missing hero), got: %v", resp["ok"])
				return false
			}
			errStr, _ := resp["error"].(string)
			if errStr == "" {
				t.Logf("FAIL: expected non-empty error for invalid config, got empty")
				return false
			}

			// Verify database was NOT changed
			var storedLayoutConfig string
			err = db.QueryRow(
				"SELECT layout_config FROM author_storefronts WHERE user_id = ?",
				userID,
			).Scan(&storedLayoutConfig)
			if err != nil {
				t.Logf("FAIL: failed to read layout_config from DB: %v", err)
				return false
			}
			if storedLayoutConfig != initialJSON {
				t.Logf("FAIL: DB layout_config changed after invalid request: before=%s, after=%s",
					initialJSON, storedLayoutConfig)
				return false
			}

			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 9 (InvalidConfigMissingHero) violated: %v", err)
		}
	})

	// Sub-test 3: Invalid configs (invalid section type) should return ok:false and DB unchanged
	t.Run("InvalidConfigBadSectionType", func(t *testing.T) {
		f := func(seed int64) bool {
			cleanup := setupTestDB(t)
			defer cleanup()

			rng := rand.New(rand.NewSource(seed))

			// Create a test user and storefront with an initial valid layout
			userID := createTestUserWithBalance(t, 0)
			slug := fmt.Sprintf("p9-badtype-%d-%d", userID, rng.Int63n(1000000))

			initialConfig := generateValidLayoutConfig(rng)
			initialJSON, err := SerializeLayoutConfig(initialConfig)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}
			if errMsg := ValidateLayoutConfig(initialJSON); errMsg != "" {
				t.Logf("FAIL: initial config failed validation: %s", errMsg)
				return false
			}

			_, err = db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, layout_config) VALUES (?, ?, ?)",
				userID, slug, initialJSON,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}

			// Generate an invalid config by injecting an invalid section type
			invalidConfig := generateValidLayoutConfig(rng)
			invalidTypes := []string{"unknown", "sidebar", "footer", "widget", "ad_block"}
			invalidType := invalidTypes[rng.Intn(len(invalidTypes))]
			idx := rng.Intn(len(invalidConfig.Sections))
			invalidConfig.Sections[idx].Type = invalidType

			invalidJSON, err := SerializeLayoutConfig(invalidConfig)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			// POST the invalid config
			form := url.Values{}
			form.Set("layout_config", invalidJSON)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/layout", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSaveLayout(rr, req)

			// Parse response
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response JSON: %v", err)
				return false
			}

			// Verify response has ok:false with non-empty error
			if resp["ok"] != false {
				t.Logf("FAIL: expected ok=false for invalid section type %q, got: %v", invalidType, resp["ok"])
				return false
			}
			errStr, _ := resp["error"].(string)
			if errStr == "" {
				t.Logf("FAIL: expected non-empty error for invalid section type, got empty")
				return false
			}

			// Verify database was NOT changed
			var storedLayoutConfig string
			err = db.QueryRow(
				"SELECT layout_config FROM author_storefronts WHERE user_id = ?",
				userID,
			).Scan(&storedLayoutConfig)
			if err != nil {
				t.Logf("FAIL: failed to read layout_config from DB: %v", err)
				return false
			}
			if storedLayoutConfig != initialJSON {
				t.Logf("FAIL: DB layout_config changed after invalid request: before=%s, after=%s",
					initialJSON, storedLayoutConfig)
				return false
			}

			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 9 (InvalidConfigBadSectionType) violated: %v", err)
		}
	})

	// Sub-test 4: Invalid configs (too many custom_banners) should return ok:false and DB unchanged
	t.Run("InvalidConfigTooManyBanners", func(t *testing.T) {
		f := func(seed int64) bool {
			cleanup := setupTestDB(t)
			defer cleanup()

			rng := rand.New(rand.NewSource(seed))

			// Create a test user and storefront with an initial valid layout
			userID := createTestUserWithBalance(t, 0)
			slug := fmt.Sprintf("p9-banners-%d-%d", userID, rng.Int63n(1000000))

			initialConfig := generateValidLayoutConfig(rng)
			initialJSON, err := SerializeLayoutConfig(initialConfig)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}
			if errMsg := ValidateLayoutConfig(initialJSON); errMsg != "" {
				t.Logf("FAIL: initial config failed validation: %s", errMsg)
				return false
			}

			_, err = db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, layout_config) VALUES (?, ?, ?)",
				userID, slug, initialJSON,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}

			// Generate an invalid config with 4+ custom_banners
			sections := []SectionConfig{
				{Type: "hero", Visible: true, Settings: json.RawMessage(`{}`)},
				{Type: "pack_grid", Visible: true, Settings: json.RawMessage(`{}`)},
			}
			numBanners := 4 + rng.Intn(4) // 4-7 banners
			validStyles := []string{"info", "success", "warning"}
			for i := 0; i < numBanners; i++ {
				style := validStyles[rng.Intn(len(validStyles))]
				settings, _ := json.Marshal(CustomBannerSettings{Text: "test banner", Style: style})
				sections = append(sections, SectionConfig{
					Type: "custom_banner", Visible: true, Settings: settings,
				})
			}
			rng.Shuffle(len(sections), func(i, j int) {
				sections[i], sections[j] = sections[j], sections[i]
			})
			invalidConfig := LayoutConfig{Sections: sections}

			invalidJSON, err := SerializeLayoutConfig(invalidConfig)
			if err != nil {
				t.Logf("FAIL: SerializeLayoutConfig failed: %v", err)
				return false
			}

			// POST the invalid config
			form := url.Values{}
			form.Set("layout_config", invalidJSON)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/layout", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSaveLayout(rr, req)

			// Parse response
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response JSON: %v", err)
				return false
			}

			// Verify response has ok:false with non-empty error
			if resp["ok"] != false {
				t.Logf("FAIL: expected ok=false for too many banners (%d), got: %v", numBanners, resp["ok"])
				return false
			}
			errStr, _ := resp["error"].(string)
			if errStr == "" {
				t.Logf("FAIL: expected non-empty error for too many banners, got empty")
				return false
			}

			// Verify database was NOT changed
			var storedLayoutConfig string
			err = db.QueryRow(
				"SELECT layout_config FROM author_storefronts WHERE user_id = ?",
				userID,
			).Scan(&storedLayoutConfig)
			if err != nil {
				t.Logf("FAIL: failed to read layout_config from DB: %v", err)
				return false
			}
			if storedLayoutConfig != initialJSON {
				t.Logf("FAIL: DB layout_config changed after invalid request: before=%s, after=%s",
					initialJSON, storedLayoutConfig)
				return false
			}

			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 9 (InvalidConfigTooManyBanners) violated: %v", err)
		}
	})
}

// Feature: storefront-customization, Property 10: 主题 API 响应正确性
// **Validates: Requirements 3.4, 7.4, 7.6**
func TestProperty10_ThemeAPIResponseCorrectness(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	validThemesList := []string{"default", "ocean", "sunset", "forest", "minimal"}

	// Sub-test 1: Valid themes should return ok:true and update DB
	t.Run("ValidThemeAccepted", func(t *testing.T) {
		f := func(seed int64) bool {
			cleanup := setupTestDB(t)
			defer cleanup()

			rng := rand.New(rand.NewSource(seed))

			// Create a test user and storefront
			userID := createTestUserWithBalance(t, 0)
			slug := fmt.Sprintf("p10-valid-%d-%d", userID, rng.Int63n(1000000))
			_, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, theme) VALUES (?, ?, 'default')",
				userID, slug,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}

			// Pick a random valid theme
			theme := validThemesList[rng.Intn(len(validThemesList))]

			// POST the theme
			form := url.Values{}
			form.Set("theme", theme)
			req := httptest.NewRequest(http.MethodPost, "/user/storefront/theme", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
			rr := httptest.NewRecorder()
			handleStorefrontSaveTheme(rr, req)

			// Parse response
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Logf("FAIL: failed to parse response JSON: %v (body: %s)", err, rr.Body.String())
				return false
			}

			// Verify response has ok:true
			if resp["ok"] != true {
				t.Logf("FAIL: expected ok=true for valid theme %q, got: %v (error: %v)", theme, resp["ok"], resp["error"])
				return false
			}

			// Verify database was updated
			var storedTheme string
			err = db.QueryRow(
				"SELECT theme FROM author_storefronts WHERE user_id = ?",
				userID,
			).Scan(&storedTheme)
			if err != nil {
				t.Logf("FAIL: failed to read theme from DB: %v", err)
				return false
			}
			if storedTheme != theme {
				t.Logf("FAIL: DB theme mismatch: expected=%q, stored=%q", theme, storedTheme)
				return false
			}

			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 10 (ValidThemeAccepted) violated: %v", err)
		}
	})

	// Sub-test 2: Invalid themes should return ok:false and DB unchanged
	t.Run("InvalidThemeRejected", func(t *testing.T) {
		f := func(seed int64) bool {
			cleanup := setupTestDB(t)
			defer cleanup()

			rng := rand.New(rand.NewSource(seed))

			// Create a test user and storefront with a known initial theme
			userID := createTestUserWithBalance(t, 0)
			slug := fmt.Sprintf("p10-invalid-%d-%d", userID, rng.Int63n(1000000))
			initialTheme := validThemesList[rng.Intn(len(validThemesList))]
			_, err := db.Exec(
				"INSERT INTO author_storefronts (user_id, store_slug, theme) VALUES (?, ?, ?)",
				userID, slug, initialTheme,
			)
			if err != nil {
				t.Logf("FAIL: failed to create storefront: %v", err)
				return false
			}

			// Generate a random invalid theme string (not in ValidThemes)
			invalidChars := "abcdefghijklmnopqrstuvwxyz0123456789_"
			for attempts := 0; attempts < 20; attempts++ {
				length := 1 + rng.Intn(15)
				var sb strings.Builder
				for i := 0; i < length; i++ {
					sb.WriteByte(invalidChars[rng.Intn(len(invalidChars))])
				}
				candidate := sb.String()
				if !ValidThemes[candidate] {
					// Found an invalid theme, use it
					form := url.Values{}
					form.Set("theme", candidate)
					req := httptest.NewRequest(http.MethodPost, "/user/storefront/theme", strings.NewReader(form.Encode()))
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
					rr := httptest.NewRecorder()
					handleStorefrontSaveTheme(rr, req)

					// Parse response
					var resp map[string]interface{}
					if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
						t.Logf("FAIL: failed to parse response JSON: %v (body: %s)", err, rr.Body.String())
						return false
					}

					// Verify response has ok:false with correct error
					if resp["ok"] != false {
						t.Logf("FAIL: expected ok=false for invalid theme %q, got: %v", candidate, resp["ok"])
						return false
					}
					errStr, _ := resp["error"].(string)
					if errStr != "不支持的主题" {
						t.Logf("FAIL: expected error '不支持的主题' for invalid theme %q, got: %q", candidate, errStr)
						return false
					}

					// Verify database was NOT changed
					var storedTheme string
					err = db.QueryRow(
						"SELECT theme FROM author_storefronts WHERE user_id = ?",
						userID,
					).Scan(&storedTheme)
					if err != nil {
						t.Logf("FAIL: failed to read theme from DB: %v", err)
						return false
					}
					if storedTheme != initialTheme {
						t.Logf("FAIL: DB theme changed after invalid request: before=%q, after=%q", initialTheme, storedTheme)
						return false
					}

					return true
				}
			}
			// Extremely unlikely: couldn't generate an invalid theme in 20 attempts
			t.Logf("WARN: could not generate invalid theme string, skipping")
			return true
		}
		if err := quick.Check(f, cfg); err != nil {
			t.Errorf("Property 10 (InvalidThemeRejected) violated: %v", err)
		}
	})
}


// Feature: storefront-customization, Property 5: 隐藏区块不渲染
// **Validates: Requirements 2.5, 8.6**
//
// For any valid layout config with sections set to visible=false,
// the rendered HTML should NOT contain data-section-type markers for hidden sections,
// and SHOULD contain data-section-type markers for visible sections.
// Since custom_banner can appear multiple times, we count expected visible markers
// per section type and compare with actual occurrences in the HTML.
func TestProperty5_HiddenSectionsNotRendered(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate a valid layout config that has at least one hidden optional section.
		config := generateValidLayoutConfigWithHidden(rng)

		// Build BannerData map for custom_banner sections
		bannerData := make(map[int]CustomBannerSettings)
		for i, sec := range config.Sections {
			if sec.Type == "custom_banner" {
				var bs CustomBannerSettings
				if err := json.Unmarshal(sec.Settings, &bs); err == nil {
					bannerData[i] = bs
				}
			}
		}

		// Determine PackGridColumns
		packGridColumns := 2
		for _, sec := range config.Sections {
			if sec.Type == "pack_grid" {
				var pgs PackGridSettings
				if err := json.Unmarshal(sec.Settings, &pgs); err == nil && pgs.Columns >= 1 && pgs.Columns <= 3 {
					packGridColumns = pgs.Columns
				}
			}
		}

		// Build StorefrontPageData
		pageData := StorefrontPageData{
			Storefront: StorefrontInfo{
				StoreName: "TestStore",
				StoreSlug: "test-store",
			},
			Packs:           []StorefrontPackInfo{},
			PurchasedIDs:    map[int64]bool{},
			DefaultLang:     "zh-CN",
			Sections:        config.Sections,
			ThemeCSS:        GetThemeCSS("default"),
			PackGridColumns: packGridColumns,
			BannerData:      bannerData,
		}

		// Render the template
		var buf bytes.Buffer
		if err := templates.StorefrontTmpl.Execute(&buf, pageData); err != nil {
			t.Logf("FAIL: template execution error: %v", err)
			return false
		}
		html := buf.String()

		// Count expected visible markers per section type.
		// For custom_banner, only visible ones with non-empty text actually render a marker.
		// For featured, it only renders if FeaturedPacks is non-empty (we have none, so 0).
		// For hero, filter_bar, pack_grid: always render when visible.
		expectedMarkerCount := make(map[string]int)
		for i, sec := range config.Sections {
			if !sec.Visible {
				continue
			}
			switch sec.Type {
			case "hero", "filter_bar", "pack_grid":
				expectedMarkerCount[sec.Type]++
			case "featured":
				// featured only renders if FeaturedPacks is non-empty; we have none
			case "custom_banner":
				if bd, ok := bannerData[i]; ok && bd.Text != "" {
					expectedMarkerCount["custom_banner"]++
				}
			}
		}

		// Count actual marker occurrences in the HTML
		for _, sectionType := range []string{"hero", "featured", "filter_bar", "pack_grid", "custom_banner"} {
			marker := fmt.Sprintf(`data-section-type="%s"`, sectionType)
			actualCount := strings.Count(html, marker)
			expected := expectedMarkerCount[sectionType]

			if actualCount != expected {
				t.Logf("FAIL: section type %q: expected %d markers, found %d in HTML", sectionType, expected, actualCount)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (HiddenSectionsNotRendered) violated: %v", err)
	}
}

// generateValidLayoutConfigWithHidden generates a valid layout config that ensures
// at least one optional section (featured, filter_bar, or custom_banner) is hidden.
func generateValidLayoutConfigWithHidden(rng *rand.Rand) LayoutConfig {
	var sections []SectionConfig

	// Always include hero (must be visible)
	sections = append(sections, SectionConfig{
		Type:     "hero",
		Visible:  true,
		Settings: json.RawMessage("{}"),
	})

	// Track which optional sections we add
	type optionalSection struct {
		index   int
		canHide bool
	}
	var optionals []int

	// Include featured section
	featuredVisible := rng.Intn(2) == 0
	sections = append(sections, SectionConfig{
		Type:     "featured",
		Visible:  featuredVisible,
		Settings: json.RawMessage("{}"),
	})
	optionals = append(optionals, len(sections)-1)

	// Include filter_bar section
	filterVisible := rng.Intn(2) == 0
	sections = append(sections, SectionConfig{
		Type:     "filter_bar",
		Visible:  filterVisible,
		Settings: json.RawMessage("{}"),
	})
	optionals = append(optionals, len(sections)-1)

	// Add 1-3 custom_banner sections
	numBanners := 1 + rng.Intn(3) // 1, 2, or 3
	validStyles := []string{"info", "success", "warning"}
	for i := 0; i < numBanners; i++ {
		textLen := 1 + rng.Intn(50) // 1-50 characters, non-empty for rendering
		textRunes := make([]rune, textLen)
		chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
		for j := range textRunes {
			textRunes[j] = chars[rng.Intn(len(chars))]
		}
		style := validStyles[rng.Intn(len(validStyles))]
		settings, _ := json.Marshal(CustomBannerSettings{
			Text:  string(textRunes),
			Style: style,
		})
		bannerVisible := rng.Intn(2) == 0
		sections = append(sections, SectionConfig{
			Type:     "custom_banner",
			Visible:  bannerVisible,
			Settings: settings,
		})
		optionals = append(optionals, len(sections)-1)
	}

	// Ensure at least one optional section is hidden
	anyHidden := false
	for _, idx := range optionals {
		if !sections[idx].Visible {
			anyHidden = true
			break
		}
	}
	if !anyHidden && len(optionals) > 0 {
		// Force one random optional section to be hidden
		hideIdx := optionals[rng.Intn(len(optionals))]
		sections[hideIdx].Visible = false
	}

	// Always include pack_grid (must be visible) with valid columns
	validColumns := []int{1, 2, 3}
	columns := validColumns[rng.Intn(len(validColumns))]
	packGridSettings, _ := json.Marshal(PackGridSettings{Columns: columns})
	sections = append(sections, SectionConfig{
		Type:     "pack_grid",
		Visible:  true,
		Settings: packGridSettings,
	})

	// Shuffle the sections to randomize order
	rng.Shuffle(len(sections), func(i, j int) {
		sections[i], sections[j] = sections[j], sections[i]
	})

	return LayoutConfig{Sections: sections}
}

// Feature: storefront-customization, Property 6: 区块按配置顺序渲染
// **Validates: Requirements 2.8, 8.2, 8.3**
//
// For any valid layout config, the rendered HTML should have visible sections
// appear in the same order as the sections array.
func TestProperty6_SectionsRenderedInOrder(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate a valid layout config with randomized section order
		config := generateValidLayoutConfigForOrder(rng)

		// Build BannerData map for custom_banner sections
		bannerData := make(map[int]CustomBannerSettings)
		for i, sec := range config.Sections {
			if sec.Type == "custom_banner" {
				var bs CustomBannerSettings
				if err := json.Unmarshal(sec.Settings, &bs); err == nil {
					bannerData[i] = bs
				}
			}
		}

		// Determine PackGridColumns
		packGridColumns := 2
		for _, sec := range config.Sections {
			if sec.Type == "pack_grid" {
				var pgs PackGridSettings
				if err := json.Unmarshal(sec.Settings, &pgs); err == nil && pgs.Columns >= 1 && pgs.Columns <= 3 {
					packGridColumns = pgs.Columns
				}
			}
		}

		// Build StorefrontPageData
		pageData := StorefrontPageData{
			Storefront: StorefrontInfo{
				StoreName: "TestStore",
				StoreSlug: "test-store",
			},
			Packs:           []StorefrontPackInfo{},
			PurchasedIDs:    map[int64]bool{},
			DefaultLang:     "zh-CN",
			Sections:        config.Sections,
			ThemeCSS:        GetThemeCSS("default"),
			PackGridColumns: packGridColumns,
			BannerData:      bannerData,
		}

		// Render the template
		var buf bytes.Buffer
		if err := templates.StorefrontTmpl.Execute(&buf, pageData); err != nil {
			t.Logf("FAIL: template execution error: %v", err)
			return false
		}
		html := buf.String()

		// Collect the expected order of visible section markers.
		// For featured: only renders if FeaturedPacks is non-empty (we have none, so skip).
		// For custom_banner: only renders if text is non-empty.
		// For hero, filter_bar, pack_grid: always render when visible.
		var expectedOrder []string
		for i, sec := range config.Sections {
			if !sec.Visible {
				continue
			}
			switch sec.Type {
			case "hero", "filter_bar", "pack_grid":
				expectedOrder = append(expectedOrder, sec.Type)
			case "featured":
				// featured only renders if FeaturedPacks is non-empty; we have none
			case "custom_banner":
				if bd, ok := bannerData[i]; ok && bd.Text != "" {
					expectedOrder = append(expectedOrder, "custom_banner")
				}
			}
		}

		// Find positions of each expected marker in the HTML (in order)
		lastPos := -1
		for _, sectionType := range expectedOrder {
			marker := fmt.Sprintf(`data-section-type="%s"`, sectionType)
			// Find the next occurrence of this marker after lastPos
			searchFrom := lastPos + 1
			pos := strings.Index(html[searchFrom:], marker)
			if pos == -1 {
				t.Logf("FAIL: marker %q not found after position %d", marker, lastPos)
				return false
			}
			absolutePos := searchFrom + pos
			if absolutePos <= lastPos {
				t.Logf("FAIL: marker %q at position %d is not after previous position %d", marker, absolutePos, lastPos)
				return false
			}
			lastPos = absolutePos
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 6 (SectionsRenderedInOrder) violated: %v", err)
	}
}

// generateValidLayoutConfigForOrder generates a valid layout config with all section types
// present and visible (except random custom_banner visibility), with randomized ordering.
// This ensures we have multiple visible sections to verify ordering.
func generateValidLayoutConfigForOrder(rng *rand.Rand) LayoutConfig {
	var sections []SectionConfig

	// Always include hero (must be visible)
	sections = append(sections, SectionConfig{
		Type:     "hero",
		Visible:  true,
		Settings: json.RawMessage("{}"),
	})

	// Always include featured (visible)
	sections = append(sections, SectionConfig{
		Type:     "featured",
		Visible:  true,
		Settings: json.RawMessage("{}"),
	})

	// Always include filter_bar (visible)
	sections = append(sections, SectionConfig{
		Type:     "filter_bar",
		Visible:  true,
		Settings: json.RawMessage("{}"),
	})

	// Add 1-3 custom_banner sections with non-empty text to ensure they render
	numBanners := 1 + rng.Intn(3) // 1, 2, or 3
	validStyles := []string{"info", "success", "warning"}
	for i := 0; i < numBanners; i++ {
		textLen := 1 + rng.Intn(50) // 1-50 characters, always non-empty
		textRunes := make([]rune, textLen)
		chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
		for j := range textRunes {
			textRunes[j] = chars[rng.Intn(len(chars))]
		}
		style := validStyles[rng.Intn(len(validStyles))]
		settings, _ := json.Marshal(CustomBannerSettings{
			Text:  string(textRunes),
			Style: style,
		})
		sections = append(sections, SectionConfig{
			Type:     "custom_banner",
			Visible:  true,
			Settings: settings,
		})
	}

	// Always include pack_grid (must be visible) with valid columns
	validColumns := []int{1, 2, 3}
	columns := validColumns[rng.Intn(len(validColumns))]
	packGridSettings, _ := json.Marshal(PackGridSettings{Columns: columns})
	sections = append(sections, SectionConfig{
		Type:     "pack_grid",
		Visible:  true,
		Settings: packGridSettings,
	})

	// Shuffle the sections to randomize order - this is the key for Property 6
	rng.Shuffle(len(sections), func(i, j int) {
		sections[i], sections[j] = sections[j], sections[i]
	})

	return LayoutConfig{Sections: sections}
}

// Feature: storefront-customization, Property 11: 分析包网格列数反映在 CSS 中
// **Validates: Requirements 4.3, 4.4**
func TestProperty11_PackGridColumnsCSS(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Pick one of 4 cases: columns = 1, 2, 3, or undefined (0 means unset)
		choices := []int{0, 1, 2, 3}
		columns := choices[rng.Intn(len(choices))]

		// Build pack_grid settings
		var packGridSettingsJSON json.RawMessage
		if columns == 0 {
			// Undefined / empty settings — should default to 2
			packGridSettingsJSON = json.RawMessage("{}")
		} else {
			s, _ := json.Marshal(PackGridSettings{Columns: columns})
			packGridSettingsJSON = s
		}

		// Determine the effective columns for PackGridColumns field
		effectiveColumns := 2
		if columns >= 1 && columns <= 3 {
			effectiveColumns = columns
		}

		// Build a minimal valid layout with hero + pack_grid
		sections := []SectionConfig{
			{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "pack_grid", Visible: true, Settings: packGridSettingsJSON},
		}

		// Provide at least one pack so the pack-list grid div is rendered
		// (template only renders grid-template-columns when Packs is non-empty)
		dummyPacks := []StorefrontPackInfo{
			{PackName: "TestPack", ShareMode: "free", ShareToken: "tok1"},
		}

		pageData := StorefrontPageData{
			Storefront: StorefrontInfo{
				StoreName: "TestStore",
				StoreSlug: "test-store",
			},
			Packs:           dummyPacks,
			PurchasedIDs:    map[int64]bool{},
			DefaultLang:     "zh-CN",
			Sections:        sections,
			ThemeCSS:        GetThemeCSS("default"),
			PackGridColumns: effectiveColumns,
			BannerData:      map[int]CustomBannerSettings{},
		}

		// Render the template
		var buf bytes.Buffer
		if err := templates.StorefrontTmpl.Execute(&buf, pageData); err != nil {
			t.Logf("FAIL: template execution error: %v", err)
			return false
		}
		html := buf.String()

		// Verify grid-template-columns matches the effective column count
		expectedCSS := fmt.Sprintf("grid-template-columns: repeat(%d, 1fr)", effectiveColumns)
		if !strings.Contains(html, expectedCSS) {
			t.Logf("FAIL: columns=%d (effective=%d), expected %q in HTML but not found", columns, effectiveColumns, expectedCSS)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 11 (PackGridColumnsCSS) violated: %v", err)
	}
}

// Feature: storefront-customization, Property 12: 预览模式仅对小铺作者生效
// **Validates: Requirements 9.3, 9.4**
func TestProperty12_PreviewModeAuthorOnly(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate a random store name
		storeNames := []string{"TestStore", "我的小铺", "DataShop", "分析工坊", "Pack市场"}
		storeName := storeNames[rng.Intn(len(storeNames))]
		storeSlug := fmt.Sprintf("store-%d", rng.Int63())

		// Build a minimal valid layout with hero + pack_grid
		sections := []SectionConfig{
			{Type: "hero", Visible: true, Settings: json.RawMessage("{}")},
			{Type: "pack_grid", Visible: true, Settings: json.RawMessage(`{"columns":2}`)},
		}

		// Case 1: Author visits with preview=1 → IsPreviewMode=true → should contain "预览模式"
		authorPageData := StorefrontPageData{
			Storefront: StorefrontInfo{
				StoreName: storeName,
				StoreSlug: storeSlug,
			},
			Packs:           []StorefrontPackInfo{},
			PurchasedIDs:    map[int64]bool{},
			DefaultLang:     "zh-CN",
			Sections:        sections,
			ThemeCSS:        GetThemeCSS("default"),
			PackGridColumns: 2,
			BannerData:      map[int]CustomBannerSettings{},
			IsPreviewMode:   true,
		}

		var buf1 bytes.Buffer
		if err := templates.StorefrontTmpl.Execute(&buf1, authorPageData); err != nil {
			t.Logf("FAIL: template execution error (author preview): %v", err)
			return false
		}
		authorHTML := buf1.String()

		if !strings.Contains(authorHTML, "预览模式") {
			t.Logf("FAIL: author with preview=1 should see '预览模式' but it was not found (store=%s)", storeName)
			return false
		}

		// Case 2: Non-author visits with preview=1 → IsPreviewMode=false → should NOT contain "预览模式"
		nonAuthorPageData := StorefrontPageData{
			Storefront: StorefrontInfo{
				StoreName: storeName,
				StoreSlug: storeSlug,
			},
			Packs:           []StorefrontPackInfo{},
			PurchasedIDs:    map[int64]bool{},
			DefaultLang:     "zh-CN",
			Sections:        sections,
			ThemeCSS:        GetThemeCSS("default"),
			PackGridColumns: 2,
			BannerData:      map[int]CustomBannerSettings{},
			IsPreviewMode:   false,
		}

		var buf2 bytes.Buffer
		if err := templates.StorefrontTmpl.Execute(&buf2, nonAuthorPageData); err != nil {
			t.Logf("FAIL: template execution error (non-author): %v", err)
			return false
		}
		nonAuthorHTML := buf2.String()

		if strings.Contains(nonAuthorHTML, "预览模式") {
			t.Logf("FAIL: non-author with preview=1 should NOT see '预览模式' but it was found (store=%s)", storeName)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 12 (PreviewModeAuthorOnly) violated: %v", err)
	}
}
