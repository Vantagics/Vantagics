package main

import (
	"context"
	"os/exec"
	"runtime"
	"time"

	"github.com/chromedp/chromedp"
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
