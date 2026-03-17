package launch

// Integration tests for the stdin/stdout/stderr pipe chain between the WSL
// CLI and the Windows helper. These tests use the "test subprocess" pattern:
// the test binary re-invokes itself as a fake helper controlled by the
// WSTART_TEST_HELPER environment variable, so the full JSON-framing +
// io.MultiReader protocol is exercised without requiring WSL or Windows.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

// TestMain intercepts subprocess invocations before the test framework parses
// flags. When WSTART_TEST_HELPER is set the binary acts as a fake
// wstart-host --exec: it reads a JSON LaunchRequest from stdin (mirroring
// runExec in cmd/wstart-host/main.go) then executes the requested action.
func TestMain(m *testing.M) {
	if mode := os.Getenv("WSTART_TEST_HELPER"); mode != "" {
		fakeHelper(mode)
		// fakeHelper always exits; this line is never reached.
	}
	os.Exit(m.Run())
}

// fakeHelper mirrors the protocol of cmd/wstart-host runExec:
//  1. Decode exactly one JSON LaunchRequest from stdin.
//  2. Recover any bytes the decoder buffered past the JSON object.
//  3. Perform the action dictated by mode.
func fakeHelper(mode string) {
	dec := json.NewDecoder(os.Stdin)
	var req protocol.LaunchRequest
	if err := dec.Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "fakeHelper: decode:", err)
		os.Exit(1)
	}
	remaining := io.MultiReader(dec.Buffered(), os.Stdin)

	switch mode {
	case "echo-stdin":
		if _, err := io.Copy(os.Stdout, remaining); err != nil {
			fmt.Fprintln(os.Stderr, "fakeHelper: copy:", err)
			os.Exit(1)
		}
	case "echo-stderr":
		if _, err := io.Copy(os.Stderr, remaining); err != nil {
			fmt.Fprintln(os.Stderr, "fakeHelper: copy:", err)
			os.Exit(1)
		}
	case "exit-42":
		os.Exit(42)
	case "echo-file":
		fmt.Fprint(os.Stdout, req.File)
	default:
		fmt.Fprintln(os.Stderr, "fakeHelper: unknown mode:", mode)
		os.Exit(1)
	}
	os.Exit(0)
}

// testBinary returns the path to the currently running test binary, which
// doubles as the fake helper when WSTART_TEST_HELPER is set.
func testBinary(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal("os.Executable:", err)
	}
	return exe
}

// testReq returns a minimal LaunchRequest sufficient for fakeHelper.
func testReq() *protocol.LaunchRequest {
	return &protocol.LaunchRequest{File: "test-target"}
}

func TestExecWithIOStdinForwarding(t *testing.T) {
	t.Setenv("WSTART_TEST_HELPER", "echo-stdin")

	want := "hello from wsl stdin"
	var stdout, stderr bytes.Buffer

	code, err := execWithIO(testBinary(t), testReq(), strings.NewReader(want), &stdout, &stderr)
	if err != nil {
		t.Fatalf("execWithIO: %v (stderr: %s)", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if got := stdout.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

// TestExecWithIOLargeStdin verifies that bytes are not lost across the
// json.Decoder's internal read-ahead buffer when stdin data follows the JSON.
func TestExecWithIOLargeStdin(t *testing.T) {
	t.Setenv("WSTART_TEST_HELPER", "echo-stdin")

	want := strings.Repeat("abcdefgh", 8*1024) // 64 KiB
	var stdout, stderr bytes.Buffer

	code, err := execWithIO(testBinary(t), testReq(), strings.NewReader(want), &stdout, &stderr)
	if err != nil {
		t.Fatalf("execWithIO: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if got := stdout.String(); got != want {
		t.Errorf("large stdin: got %d bytes, want %d bytes", len(got), len(want))
	}
}

func TestExecWithIOEmptyStdin(t *testing.T) {
	t.Setenv("WSTART_TEST_HELPER", "echo-stdin")

	var stdout, stderr bytes.Buffer

	code, err := execWithIO(testBinary(t), testReq(), strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("execWithIO: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
}

func TestExecWithIOStderr(t *testing.T) {
	t.Setenv("WSTART_TEST_HELPER", "echo-stderr")

	want := "error output from child"
	var stdout, stderr bytes.Buffer

	code, err := execWithIO(testBinary(t), testReq(), strings.NewReader(want), &stdout, &stderr)
	if err != nil {
		t.Fatalf("execWithIO: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty, got %q", stdout.String())
	}
	if got := stderr.String(); got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestExecWithIOExitCode(t *testing.T) {
	t.Setenv("WSTART_TEST_HELPER", "exit-42")

	code, err := execWithIO(testBinary(t), testReq(), strings.NewReader(""), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("execWithIO: %v", err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

// TestExecWithIORequestFields verifies the JSON request is decoded correctly
// by the helper (fakeHelper echoes req.File back on stdout via "echo-file").
func TestExecWithIORequestFields(t *testing.T) {
	t.Setenv("WSTART_TEST_HELPER", "echo-file")

	req := &protocol.LaunchRequest{
		File: "notepad.exe",
		Args: []string{"foo.txt"},
		Verb: "open",
		Wait: true,
	}
	var stdout, stderr bytes.Buffer

	code, err := execWithIO(testBinary(t), req, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("execWithIO: %v (stderr: %s)", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if got := stdout.String(); got != req.File {
		t.Errorf("decoded File = %q, want %q", got, req.File)
	}
}
