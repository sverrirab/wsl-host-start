// wstart-host is the Windows-side helper binary.
// It is invoked by the WSL-side wstart CLI over stdin/stdout.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sverrirab/wsl-host-start/internal/allowlist"
	"github.com/sverrirab/wsl-host-start/internal/drives"
	"github.com/sverrirab/wsl-host-start/internal/protocol"
	"github.com/sverrirab/wsl-host-start/internal/shellexec"
)

var version = "dev"

func main() {
	drivesMode := flag.Bool("drives", false, "Enumerate drives and print JSON to stdout")
	launchMode := flag.Bool("launch", false, "Read LaunchRequest from stdin, execute via ShellExecuteEx, print LaunchResponse to stdout")
	execMode := flag.Bool("exec", false, "Read LaunchRequest from stdin, execute with stdio passthrough, exit with child's exit code")
	versionFlag := flag.Bool("version", false, "Print version")
	flag.Parse()

	switch {
	case *versionFlag:
		fmt.Println(version)
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

func runLaunch() error {
	var req protocol.LaunchRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return fmt.Errorf("decoding launch request: %w", err)
	}

	// Load and enforce allowlist from the directory containing this binary.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	al, err := allowlist.Load(filepath.Dir(exePath))
	if err != nil {
		return fmt.Errorf("loading allowlist: %w", err)
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
	var req protocol.LaunchRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: decoding request: %v\n", err)
		os.Exit(1)
	}

	// Load and enforce allowlist.
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: finding executable path: %v\n", err)
		os.Exit(1)
	}
	al, err := allowlist.Load(filepath.Dir(exePath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: loading allowlist: %v\n", err)
		os.Exit(1)
	}
	if err := al.Check(req.File, req.Args); err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
		os.Exit(5) // SE_ERR_ACCESSDENIED
	}

	exitCode, err := shellexec.ExecuteConsole(&req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "wstart-host: %v\n", err)
	os.Exit(1)
}
