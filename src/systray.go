//go:build !darwin

package main

import (
	"context"
	_ "embed"

	"github.com/getlantern/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func runSystray(ctx context.Context) {
	go func() {
		systray.Run(func() {
			systray.SetIcon(icon)
			systray.SetTitle("RapidBI")
			systray.SetTooltip("RapidBI")

			mShow := systray.AddMenuItem("Show", "Show App")
			mHide := systray.AddMenuItem("Hide", "Hide App")
			systray.AddSeparator()
			mQuit := systray.AddMenuItem("Quit", "Quit App")

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
