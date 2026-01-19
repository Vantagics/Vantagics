package main

import (
	"context"
	"os/exec"
	"runtime"
	"time"

	"github.com/chromedp/chromedp"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ChromeCheckResult represents the result of Chrome availability check
type ChromeCheckResult struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
	Path      string `json:"path,omitempty"`
}

// CheckChromeAvailability checks if Chrome/Chromium is available for chromedp
func (a *App) CheckChromeAvailability() ChromeCheckResult {
	// Try to create a simple chromedp context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create allocator context
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
	)
	defer allocCancel()

	// Create chromedp context
	cdpCtx, cdpCancel := chromedp.NewContext(allocCtx)
	defer cdpCancel()

	// Try to navigate to a simple page
	var title string
	err := chromedp.Run(cdpCtx,
		chromedp.Navigate("about:blank"),
		chromedp.Title(&title),
	)

	if err != nil {
		// Chrome not available, provide helpful message
		return ChromeCheckResult{
			Available: false,
			Message:   a.getChromeInstallMessage(),
		}
	}

	// Chrome is available
	chromePath := a.findChromePath()
	return ChromeCheckResult{
		Available: true,
		Message:   "Chrome/Chromium is available for web search",
		Path:      chromePath,
	}
}

// CheckChromeOnStartup checks Chrome availability and shows dialog if not available
func (a *App) CheckChromeOnStartup() bool {
	result := a.CheckChromeAvailability()
	
	if !result.Available {
		a.Log("[CHROME-CHECK] Chrome not available, showing install dialog")
		
		// Get config to determine language
		cfg, _ := a.GetConfig()
		language := cfg.Language
		
		// Show dialog with install prompt
		go func() {
			// Wait a bit for UI to be ready
			time.Sleep(500 * time.Millisecond)
			
			var title, message, installButton, cancelButton string
			var downloadURL string
			
			if language == "简体中文" {
				title = "需要安装Chrome浏览器"
				message = "Web搜索功能需要Chrome浏览器支持。\n\n是否前往下载页面？"
				installButton = "前往下载"
				cancelButton = "稍后安装"
				downloadURL = "https://www.google.cn/chrome/"
			} else {
				title = "Chrome Browser Required"
				message = "Web search requires Chrome browser.\n\nWould you like to download it now?"
				installButton = "Download Chrome"
				cancelButton = "Later"
				downloadURL = "https://www.google.com/chrome/"
			}
			
			selection, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
				Type:          wailsRuntime.QuestionDialog,
				Title:         title,
				Message:       message,
				Buttons:       []string{installButton, cancelButton},
				DefaultButton: installButton,
				CancelButton:  cancelButton,
			})
			
			if err == nil && selection == installButton {
				a.Log("[CHROME-CHECK] User chose to download Chrome")
				a.OpenURL(downloadURL)
			} else {
				a.Log("[CHROME-CHECK] User chose to install Chrome later")
			}
		}()
		
		return false
	}
	
	a.Log("[CHROME-CHECK] Chrome is available")
	return true
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		a.Log("[OPEN-URL] Unsupported platform: " + runtime.GOOS)
		return nil
	}
	
	err := cmd.Start()
	if err != nil {
		a.Log("[OPEN-URL] Failed to open URL: " + err.Error())
		return err
	}
	
	a.Log("[OPEN-URL] Opened URL: " + url)
	return nil
}

// getChromeInstallMessage returns platform-specific Chrome installation instructions
func (a *App) getChromeInstallMessage() string {
	cfg, _ := a.GetConfig()
	
	switch runtime.GOOS {
	case "windows":
		if cfg.Language == "简体中文" {
			return "未检测到Chrome浏览器\n\n" +
				"Web搜索功能需要Chrome或Chromium浏览器。\n\n" +
				"安装方法：\n" +
				"1. 下载Chrome: https://www.google.com/chrome/\n" +
				"2. 或下载Chromium: https://www.chromium.org/\n" +
				"3. 安装后重启应用\n\n" +
				"注意：如果已安装Chrome但仍显示此消息，请确保Chrome已正确安装到默认位置。"
		}
		return "Chrome browser not detected\n\n" +
			"Web search requires Chrome or Chromium browser.\n\n" +
			"Installation:\n" +
			"1. Download Chrome: https://www.google.com/chrome/\n" +
			"2. Or Chromium: https://www.chromium.org/\n" +
			"3. Restart the app after installation\n\n" +
			"Note: If Chrome is installed but not detected, ensure it's in the default location."

	case "darwin":
		if cfg.Language == "简体中文" {
			return "未检测到Chrome浏览器\n\n" +
				"Web搜索功能需要Chrome或Chromium浏览器。\n\n" +
				"安装方法：\n" +
				"1. 使用Homebrew: brew install --cask google-chrome\n" +
				"2. 或从官网下载: https://www.google.com/chrome/\n" +
				"3. 安装后重启应用\n\n" +
				"注意：确保Chrome安装在 /Applications/Google Chrome.app"
		}
		return "Chrome browser not detected\n\n" +
			"Web search requires Chrome or Chromium browser.\n\n" +
			"Installation:\n" +
			"1. Using Homebrew: brew install --cask google-chrome\n" +
			"2. Or download from: https://www.google.com/chrome/\n" +
			"3. Restart the app after installation\n\n" +
			"Note: Ensure Chrome is installed at /Applications/Google Chrome.app"

	case "linux":
		if cfg.Language == "简体中文" {
			return "未检测到Chrome浏览器\n\n" +
				"Web搜索功能需要Chrome或Chromium浏览器。\n\n" +
				"安装方法：\n" +
				"Ubuntu/Debian: sudo apt install chromium-browser\n" +
				"Fedora: sudo dnf install chromium\n" +
				"Arch: sudo pacman -S chromium\n\n" +
				"或从官网下载Chrome: https://www.google.com/chrome/\n\n" +
				"安装后重启应用。"
		}
		return "Chrome browser not detected\n\n" +
			"Web search requires Chrome or Chromium browser.\n\n" +
			"Installation:\n" +
			"Ubuntu/Debian: sudo apt install chromium-browser\n" +
			"Fedora: sudo dnf install chromium\n" +
			"Arch: sudo pacman -S chromium\n\n" +
			"Or download Chrome: https://www.google.com/chrome/\n\n" +
			"Restart the app after installation."

	default:
		return "Chrome browser not detected. Please install Chrome or Chromium to use web search features."
	}
}

// findChromePath attempts to find the Chrome executable path
func (a *App) findChromePath() string {
	var paths []string

	switch runtime.GOOS {
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Users\` + getUserName() + `\AppData\Local\Google\Chrome\Application\chrome.exe`,
		}
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "linux":
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	}

	for _, path := range paths {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	return ""
}

// getUserName returns the current username (Windows helper)
func getUserName() string {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "echo %USERNAME%")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			return string(output[:len(output)-2]) // Remove \r\n
		}
	}
	return ""
}
