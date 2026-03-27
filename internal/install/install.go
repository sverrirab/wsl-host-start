//go:build windows

// Package install handles self-installation of wstart-host.exe and the
// WSL companion binary to %LOCALAPPDATA%\wstart\.
package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sverrirab/wsl-host-start/internal/signing"
)

// InstallDir returns the target installation directory.
func InstallDir() (string, error) {
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		return "", fmt.Errorf("%%ProgramFiles%% is not set")
	}
	return filepath.Join(programFiles, "wstart"), nil
}

// IsRunningFromInstallDir reports whether the current executable is already
// in the install directory.
func IsRunningFromInstallDir() bool {
	dir, err := InstallDir()
	if err != nil {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe, _ = filepath.EvalSymlinks(exe)
	return strings.EqualFold(filepath.Dir(exe), dir)
}

// Run performs the full installation:
//  1. Create install directory
//  2. Copy binaries (host + WSL)
//  3. Create default config files
//  4. Sign config files
//  5. Print WSL setup instructions
func Run() error {
	dir, err := InstallDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	// Copy binaries.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	srcDir := filepath.Dir(exePath)

	if !IsRunningFromInstallDir() {
		fmt.Printf("Installing to %s\n", dir)
		if err := copyFile(exePath, filepath.Join(dir, "wstart-host.exe")); err != nil {
			return fmt.Errorf("copying wstart-host.exe: %w", err)
		}
		fmt.Println("  Installed wstart-host.exe")
	} else {
		fmt.Printf("Already running from %s\n", dir)
	}

	// Copy WSL binary if present next to us.
	wslSrc := filepath.Join(srcDir, "wstart")
	wslDst := filepath.Join(dir, "wstart")
	if _, err := os.Stat(wslSrc); err == nil {
		if err := copyFile(wslSrc, wslDst); err != nil {
			return fmt.Errorf("copying wstart: %w", err)
		}
		fmt.Println("  Installed wstart (WSL binary)")
	} else if !IsRunningFromInstallDir() {
		fmt.Println("  WARNING: wstart (WSL binary) not found next to wstart-host.exe")
		fmt.Println("  You can copy it manually later.")
	}

	// Create default config files.
	if c, err := createIfMissing(filepath.Join(dir, "config.toml"), defaultConfig); err != nil {
		return err
	} else if c {
		fmt.Println("  Created config.toml")
	}
	if c, err := createIfMissing(filepath.Join(dir, "allowlist.toml"), defaultAllowlist); err != nil {
		return err
	} else if c {
		fmt.Println("  Created allowlist.toml")
	}

	// Sign config files.
	signed, err := signing.SignAllConfigs(dir)
	if err != nil {
		return fmt.Errorf("signing config files: %w", err)
	}
	if len(signed) > 0 {
		for _, f := range signed {
			fmt.Printf("  Signed %s\n", filepath.Base(f))
		}
	}

	// Print WSL instructions.
	fmt.Println()
	printWSLInstructions(dir)

	// Print config instructions.
	fmt.Println()
	printConfigGuide(dir)

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func createIfMissing(path, content string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("creating %s: %w", filepath.Base(path), err)
	}
	return true, nil
}

// winPathToWSL converts a Windows path like C:\Users\bob\AppData to /mnt/c/Users/bob/AppData.
func winPathToWSL(winPath string) string {
	if len(winPath) < 3 || winPath[1] != ':' {
		return winPath
	}
	drive := strings.ToLower(string(winPath[0]))
	rest := strings.ReplaceAll(winPath[2:], `\`, "/")
	return "/mnt/" + drive + rest
}

func printWSLInstructions(dir string) {
	wslPath := winPathToWSL(dir)
	wslBinary := wslPath + "/wstart"

	fmt.Println("--- WSL Setup ---")
	fmt.Println("Run these commands inside your WSL session:")
	fmt.Println()
	fmt.Printf("  mkdir -p ~/.local/bin\n")
	fmt.Printf("  ln -sf \"%s\" ~/.local/bin/wstart\n", wslBinary)
	fmt.Println()
	fmt.Println("If ~/.local/bin is not in your PATH, add to ~/.bashrc or ~/.zshrc:")
	fmt.Println(`  export PATH="$HOME/.local/bin:$PATH"`)
	fmt.Println()
	fmt.Println("Then test with:")
	fmt.Println("  wstart .")
}

func printConfigGuide(dir string) {
	fmt.Println("--- Configuration ---")
	fmt.Println()
	fmt.Printf("Config files are in: %s\n", dir)
	fmt.Println()
	fmt.Println("  config.toml      Drive mappings, env forwarding, default verb/show")
	fmt.Println("  allowlist.toml   Restrict which programs can be launched")
	fmt.Println()
	fmt.Println("After editing any config file, re-sign it:")
	fmt.Printf("  wstart-host.exe --sign-config\n")
	fmt.Println()
	fmt.Println("Config files are signed to prevent tampering from WSL.")
	fmt.Println("If signatures are invalid, wstart will refuse to launch programs.")
	fmt.Println()
	fmt.Println("To check your current configuration:")
	fmt.Printf("  wstart-host.exe --check-config\n")
	fmt.Printf("  wstart-host.exe --check-config --verbose    (show drive and default details)\n")
}
