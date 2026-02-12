// Package interop detects the WSL interop environment.
package interop

import (
	"fmt"
	"os"
	"strings"
)

// Info holds detected WSL environment information.
type Info struct {
	WSLVersion int    // 1 or 2
	DistroName string // e.g. "Ubuntu"
	InteropOK  bool   // whether Windows interop is functional
}

// Detect checks the WSL environment and returns info or an actionable error.
func Detect() (*Info, error) {
	info := &Info{}

	// Check if we're running in WSL at all.
	procVersion, err := os.ReadFile("/proc/version")
	if err != nil {
		return nil, fmt.Errorf("cannot read /proc/version: not running in Linux/WSL")
	}
	versionStr := string(procVersion)
	if !strings.Contains(strings.ToLower(versionStr), "microsoft") {
		return nil, fmt.Errorf("not running in WSL (no 'microsoft' in /proc/version)")
	}

	// Detect WSL version.
	if strings.Contains(strings.ToLower(versionStr), "wsl2") {
		info.WSLVersion = 2
	} else {
		// WSL1 kernel versions contain "Microsoft" but not "WSL2".
		// Also check for the WSL2 VM indicator.
		if _, err := os.Stat("/run/WSL"); err == nil {
			info.WSLVersion = 2
		} else {
			info.WSLVersion = 1
		}
	}

	// Get distro name.
	info.DistroName = os.Getenv("WSL_DISTRO_NAME")
	if info.DistroName == "" {
		return nil, fmt.Errorf("$WSL_DISTRO_NAME is not set; cannot determine WSL distribution")
	}

	// Check if interop is enabled.
	if err := checkInteropEnabled(); err != nil {
		return nil, err
	}

	// Check the interop socket.
	if err := checkInteropSocket(); err != nil {
		return nil, err
	}

	info.InteropOK = true
	return info, nil
}

func checkInteropEnabled() error {
	// Check binfmt_misc registration.
	for _, path := range []string{
		"/proc/sys/fs/binfmt_misc/WSLInterop",
		"/proc/sys/fs/binfmt_misc/WSLInterop-late",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "enabled") {
			return nil
		}
	}
	return fmt.Errorf(
		"WSL interop is not enabled.\n" +
			"Enable it in /etc/wsl.conf:\n" +
			"  [interop]\n" +
			"  enabled = true\n" +
			"Then restart WSL: wsl --shutdown",
	)
}

func checkInteropSocket() error {
	sock := os.Getenv("WSL_INTEROP")
	if sock == "" {
		return fmt.Errorf(
			"$WSL_INTEROP is not set. This can happen with systemd.\n" +
				"Try: ls /run/WSL/*_interop\n" +
				"And export WSL_INTEROP to the correct socket path.",
		)
	}
	if _, err := os.Stat(sock); err != nil {
		return fmt.Errorf(
			"$WSL_INTEROP socket %q does not exist.\n"+
				"Try restarting your WSL session or running: wsl --shutdown",
			sock,
		)
	}
	return nil
}
