package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"vantagedata/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// MenuTexts holds localized menu text
type MenuTexts struct {
	File     string
	Settings string
	Exit     string
	Help     string
	About    string
}

// getMenuTexts returns localized menu texts based on language
func getMenuTexts(language string) MenuTexts {
	if language == "简体中文" {
		return MenuTexts{
			File:     "文件",
			Settings: "设置",
			Exit:     "退出",
			Help:     "帮助",
			About:    "关于",
		}
	}
	// Default to English
	return MenuTexts{
		File:     "File",
		Settings: "Settings",
		Exit:     "Exit",
		Help:     "Help",
		About:    "About",
	}
}

// getWindowTitle returns localized window title based on language
func getWindowTitle(language string) string {
	if language == "简体中文" {
		return "观界 - 智能数据分析"
	}
	return "VantageData - Smart Data Analysis"
}

// getSystemLanguage detects the system language and returns appropriate app language
func getSystemLanguage() string {
	// Get system locale from environment variables
	// Check LANG, LC_ALL, LC_MESSAGES in order of priority
	locale := os.Getenv("LC_ALL")
	if locale == "" {
		locale = os.Getenv("LC_MESSAGES")
	}
	if locale == "" {
		locale = os.Getenv("LANG")
	}
	
	// Check if locale starts with "zh" (Chinese)
	locale = strings.ToLower(locale)
	if strings.HasPrefix(locale, "zh") {
		return "简体中文"
	}
	
	// Default to English for all other languages
	return "English"
}

// loadLanguageFromConfig loads the language setting from config file
// Falls back to system language if config is not set
func loadLanguageFromConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return getSystemLanguage()
	}
	configPath := filepath.Join(home, "VantageData", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config file doesn't exist, use system language
		return getSystemLanguage()
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return getSystemLanguage()
	}

	// If language is explicitly set in config, use it
	if cfg.Language == "简体中文" {
		return "简体中文"
	}
	if cfg.Language == "English" {
		return "English"
	}
	
	// Language not set or invalid, use system language
	return getSystemLanguage()
}

// Global application menu for dynamic updates
var appMenu *menu.Menu

// createApplicationMenu creates the application menu with the given language
func createApplicationMenu(app *App, language string) *menu.Menu {
	texts := getMenuTexts(language)

	newMenu := menu.NewMenu()
	if runtime.GOOS == "darwin" {
		// Create custom AppMenu with Settings/Preferences for macOS
		appMenu := newMenu.AddSubmenu("VantageData")
		appMenu.AddText(texts.About, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-about")
		})
		appMenu.AddSeparator()
		appMenu.AddText(texts.Settings+"...", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-settings")
		})
		appMenu.AddSeparator()
		appMenu.AddText("Hide VantageData", keys.CmdOrCtrl("h"), func(_ *menu.CallbackData) {
			wailsRuntime.Hide(app.ctx)
		})
		appMenu.AddSeparator()
		appMenu.AddText(texts.Exit, keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			wailsRuntime.Quit(app.ctx)
		})

		// IMPORTANT: Add EditMenu to enable Cmd+C, Cmd+V, Cmd+X, Cmd+A, Cmd+Z shortcuts on macOS
		newMenu.Append(menu.EditMenu())
	} else {
		// Non-macOS: Keep Settings in File menu
		fileMenu := newMenu.AddSubmenu(texts.File)
		fileMenu.AddText(texts.Settings, keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-settings")
		})
		fileMenu.AddSeparator()
		fileMenu.AddText(texts.Exit, keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			wailsRuntime.Quit(app.ctx)
		})

		// Add Help Menu for non-macOS
		helpMenu := newMenu.AddSubmenu(texts.Help)
		helpMenu.AddText(texts.About, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-about")
		})
	}

	return newMenu
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Load initial language from config
	language := loadLanguageFromConfig()
	windowTitle := getWindowTitle(language)

	// Create initial Application Menu
	appMenu = createApplicationMenu(app, language)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  windowTitle,
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.onBeforeClose,
		Menu:             appMenu,
		// Enable default context menu (right-click) for text inputs on all platforms
		// This enables Cut/Copy/Paste context menu in production builds
		EnableDefaultContextMenu: true,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
			},
			About: &mac.AboutInfo{
				Title:   "VantageData (观界)",
				Message: "See Beyond Data. Master Your Vantage.\n观数据之界，见商业全貌。",
			},
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: true, // Auto-open DevTools for debugging
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
