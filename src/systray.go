//go:build !darwin

package main

import (
	"context"
	_ "embed"

	"github.com/getlantern/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

// getTrayText returns localized text for system tray based on language
func getTrayText(language string) map[string]string {
	if language == "简体中文" {
		return map[string]string{
			"show":        "显示",
			"show_tip":    "显示应用程序",
			"hide":        "隐藏",
			"hide_tip":    "隐藏应用程序",
			"quit":        "退出",
			"quit_tip":    "退出应用程序",
			"tooltip":     "RapidBI - 智能数据分析",
		}
	}
	// Default to English
	return map[string]string{
		"show":        "Show",
		"show_tip":    "Show App",
		"hide":        "Hide",
		"hide_tip":    "Hide App",
		"quit":        "Quit",
		"quit_tip":    "Quit App",
		"tooltip":     "RapidBI - Smart Data Analysis",
	}
}

func runSystray(ctx context.Context) {
	go func() {
		systray.Run(func() {
			systray.SetIcon(trayIcon)
			systray.SetTitle("RapidBI")
			
			// Get initial language from config
			app := ctx.Value("app").(*App)
			config, err := app.GetConfig()
			language := "English"
			if err == nil && config.Language != "" {
				language = config.Language
			}
			
			texts := getTrayText(language)
			systray.SetTooltip(texts["tooltip"])

			mShow := systray.AddMenuItem(texts["show"], texts["show_tip"])
			mHide := systray.AddMenuItem(texts["hide"], texts["hide_tip"])
			systray.AddSeparator()
			mQuit := systray.AddMenuItem(texts["quit"], texts["quit_tip"])

			// Listen for config updates to change language
			go func() {
				// Subscribe to config-updated events
				wailsRuntime.EventsOn(ctx, "config-updated", func(optionalData ...interface{}) {
					config, err := app.GetConfig()
					if err == nil {
						newTexts := getTrayText(config.Language)
						systray.SetTooltip(newTexts["tooltip"])
						mShow.SetTitle(newTexts["show"])
						mShow.SetTooltip(newTexts["show_tip"])
						mHide.SetTitle(newTexts["hide"])
						mHide.SetTooltip(newTexts["hide_tip"])
						mQuit.SetTitle(newTexts["quit"])
						mQuit.SetTooltip(newTexts["quit_tip"])
					}
				})
			}()

			go func() {
				for {
					select {
					case <-mShow.ClickedCh:
						wailsRuntime.WindowShow(ctx)
					case <-mHide.ClickedCh:
						wailsRuntime.WindowHide(ctx)
					case <-mQuit.ClickedCh:
						systray.Quit()
						wailsRuntime.Quit(ctx)
					}
				}
			}()
		}, func() {
			// Cleanup
		})
	}()
}
