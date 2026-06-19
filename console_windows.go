//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var procSetConsoleTitle = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleTitleW")

func setConsoleTitle(title string) {
	ptr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	_, _, _ = procSetConsoleTitle.Call(uintptr(unsafe.Pointer(ptr)))
}
