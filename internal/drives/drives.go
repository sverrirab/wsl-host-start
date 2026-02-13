//go:build windows

// Package drives enumerates Windows drive letters and their types using Win32 APIs.
package drives

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procGetLogicalDrives = kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW    = kernel32.NewProc("GetDriveTypeW")
	procQueryDosDeviceW  = kernel32.NewProc("QueryDosDeviceW")
)

const (
	driveUnknown   = 0
	driveNoRootDir = 1
	driveRemovable = 2
	driveFixed     = 3
	driveRemote    = 4
	driveCDROM     = 5
	driveRAMDisk   = 6
)

// Enumerate returns information about all active drive letters.
func Enumerate() (*protocol.DrivesResponse, error) {
	mask, _, err := procGetLogicalDrives.Call()
	if mask == 0 {
		return nil, fmt.Errorf("GetLogicalDrives failed: %w", err)
	}

	var drives []protocol.DriveInfo
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		info, enumErr := enumerateDrive(letter)
		if enumErr != nil {
			continue
		}
		drives = append(drives, info)
	}

	return &protocol.DrivesResponse{
		Drives:       drives,
		Username:     os.Getenv("USERNAME"),
		LocalAppData: os.Getenv("LOCALAPPDATA"),
	}, nil
}

func enumerateDrive(letter string) (protocol.DriveInfo, error) {
	rootPath := letter + ":\\"
	rootPathPtr, _ := windows.UTF16PtrFromString(rootPath)

	driveType, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootPathPtr)))

	info := protocol.DriveInfo{
		Letter: letter,
		Type:   classifyDriveType(uint32(driveType)),
	}

	// Use QueryDosDevice to detect subst drives and resolve targets.
	deviceName := letter + ":"
	deviceNamePtr, _ := windows.UTF16PtrFromString(deviceName)
	buf := make([]uint16, 1024)
	n, _, err := procQueryDosDeviceW.Call(
		uintptr(unsafe.Pointer(deviceNamePtr)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if n > 0 {
		target := windows.UTF16ToString(buf[:n])
		if strings.HasPrefix(target, `\??\`) {
			// This is a subst drive. The real path is after the \??\ prefix.
			info.Type = protocol.DriveSubst
			info.Target = target[4:]
		} else if info.Type == protocol.DriveNetwork {
			info.Target = target
		}
	} else if err != nil {
		// Non-fatal: we just won't have target info.
		_ = err
	}

	// Get volume label.
	info.Label = getVolumeLabel(rootPath)

	return info, nil
}

func getVolumeLabel(rootPath string) string {
	rootPathPtr, _ := windows.UTF16PtrFromString(rootPath)
	labelBuf := make([]uint16, 256)
	err := windows.GetVolumeInformation(
		rootPathPtr,
		&labelBuf[0],
		uint32(len(labelBuf)),
		nil, nil, nil, nil, 0,
	)
	if err != nil {
		return ""
	}
	return windows.UTF16ToString(labelBuf)
}

func classifyDriveType(t uint32) string {
	switch t {
	case driveFixed:
		return protocol.DriveFixed
	case driveRemote:
		return protocol.DriveNetwork
	case driveRemovable:
		return protocol.DriveRemovable
	case driveCDROM:
		return protocol.DriveCDROM
	case driveRAMDisk:
		return protocol.DriveRAMDisk
	default:
		return protocol.DriveUnknown
	}
}
