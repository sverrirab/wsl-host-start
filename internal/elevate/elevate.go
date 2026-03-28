//go:build windows

// Package elevate provides UAC elevation detection and self-elevation
// for Windows binaries.
package elevate

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32             = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW = shell32.NewProc("ShellExecuteExW")
)

const tokenElevation = 20 // TokenElevation info class

type tokenElevationStruct struct {
	TokenIsElevated uint32
}

// IsElevated returns true if the current process is running with admin privileges.
func IsElevated() bool {
	var token windows.Token
	process := windows.CurrentProcess()
	err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	var elevation tokenElevationStruct
	var size uint32
	err = windows.GetTokenInformation(token, tokenElevation, (*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &size)
	if err != nil {
		return false
	}
	return elevation.TokenIsElevated != 0
}

// RunElevated re-launches the current executable with the given arguments
// using the "runas" verb (triggering a UAC prompt). It does not wait for
// the elevated process to finish — the current process should exit after
// calling this.
func RunElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}

	argsStr := strings.Join(args, " ")

	verbPtr, _ := windows.UTF16PtrFromString("runas")
	exePtr, _ := windows.UTF16PtrFromString(exe)
	argsPtr, _ := windows.UTF16PtrFromString(argsStr)
	dirPtr, _ := windows.UTF16PtrFromString("")

	// SHELLEXECUTEINFOW structure
	type shellExecuteInfo struct {
		cbSize       uint32
		fMask        uint32
		hwnd         uintptr
		lpVerb       *uint16
		lpFile       *uint16
		lpParameters *uint16
		lpDirectory  *uint16
		nShow        int32
		hInstApp     uintptr
		lpIDList     uintptr
		lpClass      *uint16
		hkeyClass    uintptr
		dwHotKey     uint32
		hIcon        uintptr
		hProcess     uintptr
	}

	sei := shellExecuteInfo{
		lpVerb:       verbPtr,
		lpFile:       exePtr,
		lpParameters: argsPtr,
		lpDirectory:  dirPtr,
		nShow:        1, // SW_SHOWNORMAL
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))
	sei.fMask = 0x00000040 // SEE_MASK_NOCLOSEPROCESS

	ret, _, err := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&sei)))
	if ret == 0 {
		return fmt.Errorf("elevation failed: %w", err)
	}

	// Wait for the elevated process to finish so the user sees its output.
	if sei.hProcess != 0 {
		_, _ = windows.WaitForSingleObject(windows.Handle(sei.hProcess), windows.INFINITE)
		_ = windows.CloseHandle(windows.Handle(sei.hProcess))
	}

	return nil
}

// RequireElevation checks if the process is elevated. If not, it re-launches
// with UAC elevation and returns true (meaning the caller should exit).
// If already elevated, returns false (caller should continue).
func RequireElevation(args []string) (bool, error) {
	if IsElevated() {
		return false, nil
	}

	if err := RunElevated(args); err != nil {
		return false, fmt.Errorf("failed to elevate: %w", err)
	}
	return true, nil
}
