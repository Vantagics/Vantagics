//go:build !windows

package main

// getWindowsLocale is a no-op on non-Windows platforms.
func getWindowsLocale() string {
	return ""
}
