// Package shellexec wraps Windows process execution APIs.
// Execute() uses ShellExecuteExW for GUI operations (open, runas, edit, etc.).
// ExecuteConsole() uses CreateProcess (via os/exec) for console programs with stdio passthrough.
package shellexec

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

var (
	shell32             = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW = shell32.NewProc("ShellExecuteExW")

	kernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procSearchPathW   = kernel32.NewProc("SearchPathW")
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

// resolveCommand searches for a bare command name in PATH using PATHEXT
// extensions, mimicking how cmd.exe resolves commands for "start".
func resolveCommand(name string) string {
	// If it already contains a path separator or drive letter, leave it alone.
	if strings.ContainsAny(name, `\/`) || (len(name) >= 2 && name[1] == ':') {
		return name
	}

	// If it already has an extension that exists on disk, return as-is.
	if filepath.Ext(name) != "" {
		if fullPath, err := searchPath(name); err == nil {
			return fullPath
		}
		return name
	}

	// Try each PATHEXT extension.
	pathext := os.Getenv("PATHEXT")
	if pathext == "" {
		pathext = ".COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC"
	}
	for _, ext := range strings.Split(pathext, ";") {
		if fullPath, err := searchPath(name + ext); err == nil {
			return fullPath
		}
	}

	return name
}

// searchPath uses the Windows SearchPathW API to find a file in PATH.
func searchPath(name string) (string, error) {
	namePtr, _ := windows.UTF16PtrFromString(name)
	buf := make([]uint16, windows.MAX_PATH)
	n, _, err := procSearchPathW.Call(
		0, // lpPath = NULL → use default search order (system dirs + PATH)
		uintptr(unsafe.Pointer(namePtr)),
		0, // lpExtension = NULL
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&buf[0])),
		0, // lpFilePart = NULL
	)
	if n == 0 {
		return "", fmt.Errorf("SearchPath: %w", err)
	}
	return windows.UTF16ToString(buf[:n]), nil
}

// Execute runs ShellExecuteExW with the given launch request and returns the result.
func Execute(req *protocol.LaunchRequest) *protocol.LaunchResponse {
	resp := &protocol.LaunchResponse{}

	verb := req.Verb
	if verb == "" {
		verb = "open"
	}

	// Resolve bare command names against PATH + PATHEXT.
	file := resolveCommand(req.File)

	sei := shellExecuteInfo{
		fMask: seeMaskNoCloseProcess | seeMaskFlagNoUI,
		nShow: mapShow(req.Show),
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))

	verbPtr, _ := windows.UTF16PtrFromString(verb)
	sei.lpVerb = verbPtr

	filePtr, _ := windows.UTF16PtrFromString(file)
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

// ExecuteConsole runs a command with stdio connected to the current console.
// This is used for -wait mode where program output needs to flow to the terminal.
// Returns the child process exit code.
func ExecuteConsole(req *protocol.LaunchRequest) (int, error) {
	file := resolveCommand(req.File)

	cmd := exec.Command(file, req.Args...)
	cmd.Dir = req.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Apply forwarded env vars on top of current environment.
	if len(req.EnvVars) > 0 {
		cmd.Env = os.Environ()
		for k, v := range req.EnvVars {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
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
