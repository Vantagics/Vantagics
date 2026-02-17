//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocaleName = kernel32.NewProc("GetUserDefaultLocaleName")
)

// getWindowsLocale returns the user's default locale name using Windows API directly,
// avoiding the need to spawn a PowerShell process which can cause console window flash.
func getWindowsLocale() string {
	buf := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH = 85
	r, _, _ := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
