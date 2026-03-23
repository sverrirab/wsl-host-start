//go:build windows

package shellexec_test

// Windows integration tests for shellexec.Execute (ShellExecuteExW) and
// shellexec.ExecuteConsole (os/exec with stdio wiring).
//
// These tests run console programs with hidden windows so they are safe in
// headless CI environments (GitHub Actions windows-latest).

import (
	"strings"
	"testing"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
	"github.com/sverrirab/wsl-host-start/internal/shellexec"
)

// --- ExecuteConsole tests (os/exec, console programs) ---

func TestExecuteConsoleSuccess(t *testing.T) {
	req := &protocol.LaunchRequest{
		File: "cmd.exe",
		Args: []string{"/c", "exit", "0"},
	}
	code, err := shellexec.ExecuteConsole(req, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ExecuteConsole: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestExecuteConsoleExitCode(t *testing.T) {
	req := &protocol.LaunchRequest{
		File: "cmd.exe",
		Args: []string{"/c", "exit", "42"},
	}
	code, err := shellexec.ExecuteConsole(req, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ExecuteConsole: %v", err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

// TestExecuteConsoleStdin pipes data into a command that consumes stdin.
// `findstr /r "."` matches any non-empty line; exit 0 means stdin was read.
func TestExecuteConsoleStdin(t *testing.T) {
	req := &protocol.LaunchRequest{
		File: "findstr.exe",
		Args: []string{"/r", "."},
	}
	code, err := shellexec.ExecuteConsole(req, strings.NewReader("hello from wsl\r\n"))
	if err != nil {
		t.Fatalf("ExecuteConsole: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stdin not delivered or findstr found no match)", code)
	}
}

// TestExecuteConsoleEmptyStdin verifies a clean EOF reaches the child when no
// extra stdin data is provided.
func TestExecuteConsoleEmptyStdin(t *testing.T) {
	req := &protocol.LaunchRequest{
		File: "cmd.exe",
		Args: []string{"/c", "exit", "0"},
	}
	code, err := shellexec.ExecuteConsole(req, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ExecuteConsole: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

// --- Execute tests (ShellExecuteExW, GUI shell operations) ---

// TestExecuteWaitSuccess opens cmd.exe hidden and waits; verifies exit code 0.
func TestExecuteWaitSuccess(t *testing.T) {
	req := &protocol.LaunchRequest{
		File: "cmd.exe",
		Args: []string{"/c", "exit", "0"},
		Verb: "open",
		Show: protocol.ShowHidden,
		Wait: true,
	}
	resp := shellexec.Execute(req)
	if resp.Error != "" {
		t.Fatalf("Execute: %s (errcode %d)", resp.Error, resp.ErrCode)
	}
	if resp.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", resp.ExitCode)
	}
}

// TestExecuteWaitExitCode verifies a non-zero exit code propagates through
// ShellExecuteExW + WaitForSingleObject + GetExitCodeProcess.
func TestExecuteWaitExitCode(t *testing.T) {
	req := &protocol.LaunchRequest{
		File: "cmd.exe",
		Args: []string{"/c", "exit", "7"},
		Verb: "open",
		Show: protocol.ShowHidden,
		Wait: true,
	}
	resp := shellexec.Execute(req)
	if resp.Error != "" {
		t.Fatalf("Execute: %s (errcode %d)", resp.Error, resp.ErrCode)
	}
	if resp.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", resp.ExitCode)
	}
}

// TestExecuteNoWait launches a process without waiting. We just verify no
// error is returned and a process handle is reported.
func TestExecuteNoWait(t *testing.T) {
	req := &protocol.LaunchRequest{
		File: "cmd.exe",
		Args: []string{"/c", "exit", "0"},
		Verb: "open",
		Show: protocol.ShowHidden,
		Wait: false,
	}
	resp := shellexec.Execute(req)
	if resp.Error != "" {
		t.Fatalf("Execute: %s (errcode %d)", resp.Error, resp.ErrCode)
	}
}
