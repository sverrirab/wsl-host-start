//go:build windows

// wstart-host is the Windows-side helper binary.
// It is invoked by the WSL-side wstart CLI over stdin/stdout.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sverrirab/wsl-host-start/internal/allowlist"
	"github.com/sverrirab/wsl-host-start/internal/drives"
	"github.com/sverrirab/wsl-host-start/internal/install"
	"github.com/sverrirab/wsl-host-start/internal/protocol"
	"github.com/sverrirab/wsl-host-start/internal/shellexec"
	"github.com/sverrirab/wsl-host-start/internal/signing"
)

var version = "dev"

func main() {
	installMode := flag.Bool("install", false, "Install wstart to %LOCALAPPDATA%\\wstart and create default configs")
	drivesMode := flag.Bool("drives", false, "Enumerate drives and print JSON to stdout")
	launchMode := flag.Bool("launch", false, "Read LaunchRequest from stdin, execute via ShellExecuteEx, print LaunchResponse to stdout")
	execMode := flag.Bool("exec", false, "Read LaunchRequest from stdin, execute with stdio passthrough, exit with child's exit code")
	checkConfig := flag.Bool("check-config", false, "Print active configuration diagnostics and exit")
	signConfig := flag.Bool("sign-config", false, "Re-sign config files after editing (stores key in Windows Registry)")
	verbose := flag.Bool("verbose", false, "Print extra detail in check-config output")
	versionFlag := flag.Bool("version", false, "Print version")
	flag.Parse()

	switch {
	case *versionFlag:
		fmt.Println(version)
	case *installMode:
		if err := install.Run(); err != nil {
			fatal(err)
		}
	case *checkConfig:
		if err := runCheckConfig(*verbose); err != nil {
			fatal(err)
		}
	case *signConfig:
		if err := runSignConfig(); err != nil {
			fatal(err)
		}
	case *drivesMode:
		if err := runDrives(); err != nil {
			fatal(err)
		}
	case *launchMode:
		if err := runLaunch(); err != nil {
			fatal(err)
		}
	case *execMode:
		runExec()
	default:
		flag.Usage()
		os.Exit(1)
	}
}

func runDrives() error {
	resp, err := drives.Enumerate()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func runSignConfig() error {
	dir, err := exeDir()
	if err != nil {
		return err
	}
	if err := signing.SignAllConfigs(dir); err != nil {
		return err
	}
	fmt.Println("Config files signed successfully.")
	return nil
}

// loadAndVerify resolves the exe directory, verifies config signatures,
// and loads the allowlist. This is the shared security gate for launch/exec.
func loadAndVerify() (dir string, al *allowlist.LoadResult, err error) {
	dir, err = exeDir()
	if err != nil {
		return "", nil, err
	}

	// Verify config file signatures.
	// On first run (no key), auto-generate key and sign existing configs.
	if _, found, kerr := signing.LoadKey(); kerr != nil {
		return "", nil, fmt.Errorf("checking signing key: %w", kerr)
	} else if !found {
		if serr := signing.SignAllConfigs(dir); serr != nil {
			return "", nil, fmt.Errorf("initial config signing: %w", serr)
		}
	} else {
		if verr := signing.VerifyOrErr(dir); verr != nil {
			return "", nil, verr
		}
	}

	al, err = allowlist.Load(dir)
	if err != nil {
		return "", nil, fmt.Errorf("loading allowlist: %w", err)
	}
	return dir, al, nil
}

func runLaunch() error {
	var req protocol.LaunchRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return fmt.Errorf("decoding launch request: %w", err)
	}

	_, al, err := loadAndVerify()
	if err != nil {
		return err
	}
	if err := al.Check(req.File, req.Args); err != nil {
		resp := &protocol.LaunchResponse{
			Error:   err.Error(),
			ErrCode: 5, // SE_ERR_ACCESSDENIED
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	}

	resp := shellexec.Execute(&req)

	return json.NewEncoder(os.Stdout).Encode(resp)
}

// runExec executes a command with stdio passthrough (for -wait mode).
// The helper's exit code becomes the child's exit code.
func runExec() {
	dec := json.NewDecoder(os.Stdin)
	var req protocol.LaunchRequest
	if err := dec.Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: decoding request: %v\n", err)
		os.Exit(1)
	}

	_, al, err := loadAndVerify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
		os.Exit(1)
	}
	if err := al.Check(req.File, req.Args); err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
		os.Exit(5) // SE_ERR_ACCESSDENIED
	}

	stdin := io.MultiReader(dec.Buffered(), os.Stdin)
	exitCode, err := shellexec.ExecuteConsole(&req, stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}

func exeDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding executable path: %w", err)
	}
	return filepath.Dir(exePath), nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
	os.Exit(1)
}
