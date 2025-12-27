//go:build darwin

package main

import "context"

func runSystray(ctx context.Context) {
	// No-op on macOS to avoid Main Thread conflict with Wails/Cocoa
}
