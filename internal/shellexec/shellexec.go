// Package shellexec wraps the Windows ShellExecuteExW API.
package shellexec

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

var (
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW  = shell32.NewProc("ShellExecuteExW")
)

// SHELLEXECUTEINFOW matches the Win32 SHELLEXECUTEINFOW structure layout.
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
	hIcon        uintptr // union with hMonitor
	hProcess     windows.Handle
}

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskFlagNoUI       = 0x00000400

	swShowNormal = 1
	swHide       = 0
	swMinimize   = 6
	swMaximize   = 3
)

// Execute runs ShellExecuteExW with the given launch request and returns the result.
func Execute(req *protocol.LaunchRequest) *protocol.LaunchResponse {
	resp := &protocol.LaunchResponse{}

	verb := req.Verb
	if verb == "" {
		verb = "open"
	}

	sei := shellExecuteInfo{
		fMask: seeMaskNoCloseProcess | seeMaskFlagNoUI,
		nShow: mapShow(req.Show),
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))

	verbPtr, _ := windows.UTF16PtrFromString(verb)
	sei.lpVerb = verbPtr

	filePtr, _ := windows.UTF16PtrFromString(req.File)
	sei.lpFile = filePtr

	if len(req.Args) > 0 {
		params := strings.Join(req.Args, " ")
		paramsPtr, _ := windows.UTF16PtrFromString(params)
		sei.lpParameters = paramsPtr
	}

	if req.WorkDir != "" {
		dirPtr, _ := windows.UTF16PtrFromString(req.WorkDir)
		sei.lpDirectory = dirPtr
	}

	ret, _, err := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&sei)))
	if ret == 0 {
		resp.Error = fmt.Sprintf("ShellExecuteEx failed: %v", err)
		resp.ErrCode = int(sei.hInstApp)
		return resp
	}

	if sei.hProcess != 0 {
		resp.PID = int(sei.hProcess) // Process handle, not PID, but useful for identification
		if req.Wait {
			if _, err := windows.WaitForSingleObject(sei.hProcess, windows.INFINITE); err != nil {
				resp.Error = fmt.Sprintf("WaitForSingleObject failed: %v", err)
			} else {
				var exitCode uint32
				if err := windows.GetExitCodeProcess(sei.hProcess, &exitCode); err == nil {
					resp.ExitCode = int(exitCode)
				}
			}
		}
		_ = windows.CloseHandle(sei.hProcess)
	}

	return resp
}

func mapShow(show string) int32 {
	switch show {
	case protocol.ShowMin:
		return swMinimize
	case protocol.ShowMax:
		return swMaximize
	case protocol.ShowHidden:
		return swHide
	default:
		return swShowNormal
	}
}
