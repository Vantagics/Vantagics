package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"vantagics/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsLogger "github.com/wailsapp/wails/v2/pkg/logger"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// MenuTexts holds localized menu text
type MenuTexts struct {
	File           string
	Settings       string
	Exit           string
	Help           string
	About          string
	Market         string
	PackManager    string
	BrowseMarket   string
	VisitMarket    string
	ProductService string
}

// getMenuTexts returns localized menu texts based on language
func getMenuTexts(language string) MenuTexts {
	if language == "简体中�" {
		return MenuTexts{
			File:           "文件",
			Settings:       "设置",
			Exit:           "退�",
			Help:           "帮助",
			About:          "关于",
			Market:         "市场",
			PackManager:    "分析包管�",
			BrowseMarket:   "分析包市�",
			VisitMarket:    "个人账户",
			ProductService: "产品服务",
		}
	}
	// Default to English
	return MenuTexts{
		File:           "File",
		Settings:       "Settings",
		Exit:           "Exit",
		Help:           "Help",
		About:          "About",
		Market:         "Market",
		PackManager:    "Pack Manager",
		BrowseMarket:   "Browse Market",
		VisitMarket:    "My Account",
		ProductService: "Product Service",
	}
}

// getWindowTitle returns localized window title based on language
func getWindowTitle(language string) string {
	if language == "简体中�" {
		return "万策"
	}
	return "Vantagics"
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
	
	// On macOS, also check AppleLocale and AppleLanguages
	if runtime.GOOS == "darwin" && locale == "" {
		// Try to get locale from defaults command
		// This is more reliable on macOS
		if out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output(); err == nil {
			locale = strings.TrimSpace(string(out))
		}
	}
	
	// On Windows, use native API to get locale (no subprocess needed)
	if runtime.GOOS == "windows" && locale == "" {
		locale = getWindowsLocale()
	}
	
	// Normalize and check locale
	locale = strings.ToLower(locale)
	
	// Check for Chinese variants
	if strings.HasPrefix(locale, "zh") ||
		strings.Contains(locale, "chinese") ||
		strings.Contains(locale, "cn") ||
		strings.Contains(locale, "tw") ||
		strings.Contains(locale, "hk") ||
		strings.Contains(locale, "sg") {
		return "简体中�"
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
	configPath := filepath.Join(home, "Vantagics", "config.json")

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
	if cfg.Language == "简体中�" {
		return "简体中�"
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
		appMenu := newMenu.AddSubmenu("Vantagics")
		appMenu.AddText(texts.About, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-about")
		})
		appMenu.AddSeparator()
		appMenu.AddText(texts.Settings+"...", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-settings")
		})
		appMenu.AddSeparator()
		appMenu.AddText("Hide Vantagics", keys.CmdOrCtrl("h"), func(_ *menu.CallbackData) {
			wailsRuntime.Hide(app.ctx)
		})
		appMenu.AddSeparator()
		appMenu.AddText(texts.Exit, keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			wailsRuntime.Quit(app.ctx)
		})

		// IMPORTANT: Add EditMenu to enable Cmd+C, Cmd+V, Cmd+X, Cmd+A, Cmd+Z shortcuts on macOS
		newMenu.Append(menu.EditMenu())

		// Add Market menu
		marketMenu := newMenu.AddSubmenu(texts.Market)
		marketMenu.AddText(texts.PackManager, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-pack-manager")
		})
		marketMenu.AddText(texts.BrowseMarket, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-market-browse")
		})
		marketMenu.AddText(texts.VisitMarket, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "marketplace-portal-login")
		})

		// Add Help menu for macOS
		helpMenuMac := newMenu.AddSubmenu(texts.Help)
		helpMenuMac.AddText(texts.ProductService, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "service-portal-login")
		})
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

		// Add Market menu
		marketMenu := newMenu.AddSubmenu(texts.Market)
		marketMenu.AddText(texts.PackManager, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-pack-manager")
		})
		marketMenu.AddText(texts.BrowseMarket, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-market-browse")
		})
		marketMenu.AddText(texts.VisitMarket, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "marketplace-portal-login")
		})

		// Add Help Menu for non-macOS
		helpMenu := newMenu.AddSubmenu(texts.Help)
		helpMenu.AddText(texts.About, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "open-about")
		})
		helpMenu.AddText(texts.ProductService, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "service-portal-login")
		})
	}

	return newMenu
}

func main() {
	// Suppress internal log output (e.g. WebView2 init messages) to prevent
	// console window flash on Windows GUI builds
	log.SetOutput(io.Discard)

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
		LogLevel:         wailsLogger.ERROR,
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
				Title:   "Vantagics (万策)",
				Message: "See Beyond. Decide Better.\n于万千数据中，定最优之策�",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
