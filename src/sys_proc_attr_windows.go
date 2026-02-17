//go:build windows

package main

import "syscall"

func hiddenProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW - prevents console window creation entirely
	}
}
