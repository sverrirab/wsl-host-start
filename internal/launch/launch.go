// Package launch orchestrates the WSL-side workflow: detect environment,
// load config, translate paths, invoke the Windows helper, and return results.
package launch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sverrirab/wsl-host-start/internal/config"
	"github.com/sverrirab/wsl-host-start/internal/drivecache"
	"github.com/sverrirab/wsl-host-start/internal/interop"
	"github.com/sverrirab/wsl-host-start/internal/pathconv"
	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

// Version is set by the WSL CLI so the launch package can verify
// that the host helper matches.
var Version string

// Options holds the parsed CLI flags.
type Options struct {
	Target  string
	Args    []string
	Verb    string
	WorkDir string
	Show    string
	Wait    bool
	DryRun  bool
	Verbose bool
}

// Result holds the outcome of a launch.
type Result struct {
	ExitCode int
	PID      int
}

// Run executes the full launch workflow.
func Run(opts *Options) (*Result, error) {
	// 1. Detect WSL interop.
	info, err := interop.Detect()
	if err != nil {
		return nil, fmt.Errorf("WSL interop: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "WSL version: %d, distro: %s\n", info.WSLVersion, info.DistroName)
	}

	// 2. Locate helper binary and verify version.
	helperPath, err := findHelper()
	if err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Helper: %s\n", helperPath)
	}
	if err := checkHelperVersion(helperPath, opts.Verbose); err != nil {
		return nil, err
	}

	// 3. Load config from the helper's directory.
	helperDir := filepath.Dir(helperPath)
	cfg, err := config.Load(helperDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// 4. Get drive mappings.
	var drives []protocol.DriveInfo
	if cfg.Drives.AutoDetect {
		cache := drivecache.New(0)
		resp, cacheErr := cache.Get(helperPath)
		if cacheErr != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Drive cache: %v (continuing without aliases)\n", cacheErr)
			}
		} else {
			drives = resp.Drives
			if opts.Verbose {
				for _, d := range drives {
					if d.Type == protocol.DriveSubst {
						fmt.Fprintf(os.Stderr, "Subst: %s: → %s\n", d.Letter, d.Target)
					}
				}
			}
		}
	}

	// 5. Translate paths.
	conv := pathconv.NewConverter(drives, cfg.Drives.Aliases, cfg.Drives.PreferAliases)

	winTarget, err := conv.ToWindows(opts.Target)
	if err != nil {
		return nil, fmt.Errorf("translating target path: %w", err)
	}

	winWorkDir := opts.WorkDir
	if winWorkDir != "" {
		winWorkDir, err = conv.ToWindows(winWorkDir)
		if err != nil {
			return nil, fmt.Errorf("translating working directory: %w", err)
		}
	} else {
		// Default working directory: translate current WSL cwd.
		cwd, _ := os.Getwd()
		winWorkDir, err = conv.ToWindows(cwd)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Could not translate cwd %q: %v\n", cwd, err)
			}
			winWorkDir = ""
		}
	}

	// 6. Build launch request.
	verb := opts.Verb
	if verb == "" {
		verb = cfg.Defaults.Verb
	}
	show := opts.Show
	if show == "" {
		show = cfg.Defaults.Show
	}

	req := protocol.LaunchRequest{
		File:    winTarget,
		Verb:    verb,
		Args:    opts.Args,
		WorkDir: winWorkDir,
		Show:    show,
		Wait:    opts.Wait,
		EnvVars: collectEnvVars(cfg),
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Request: file=%q verb=%q workDir=%q wait=%v\n",
			req.File, req.Verb, req.WorkDir, req.Wait)
		if len(req.Args) > 0 {
			fmt.Fprintf(os.Stderr, "  args: %v\n", req.Args)
		}
		if len(req.EnvVars) > 0 {
			fmt.Fprintf(os.Stderr, "  env: %v\n", req.EnvVars)
		}
	}

	if opts.DryRun {
		data, _ := json.MarshalIndent(req, "", "  ")
		fmt.Println(string(data))
		return &Result{}, nil
	}

	// 7. Invoke helper.
	// Use --exec mode for wait+open: stdio passthrough for console programs.
	// Use --launch mode (ShellExecuteEx) for everything else.
	if opts.Wait && (verb == "open" || verb == "") {
		exitCode, err := invokeHelperExec(helperPath, &req)
		if err != nil {
			return nil, err
		}
		return &Result{ExitCode: exitCode}, nil
	}

	resp, err := invokeHelper(helperPath, &req)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("host helper: %s (code %d)", resp.Error, resp.ErrCode)
	}

	return &Result{
		ExitCode: resp.ExitCode,
		PID:      resp.PID,
	}, nil
}

// RefreshDrives forces a drive cache refresh and prints the results.
func RefreshDrives() error {
	helperPath, err := findHelper()
	if err != nil {
		return err
	}

	cache := drivecache.New(0)
	resp, err := cache.Refresh(helperPath)
	if err != nil {
		return err
	}

	for _, d := range resp.Drives {
		line := fmt.Sprintf("%s: [%s]", d.Letter, d.Type)
		if d.Target != "" {
			line += fmt.Sprintf(" → %s", d.Target)
		}
		if d.Label != "" {
			line += fmt.Sprintf(" (%s)", d.Label)
		}
		fmt.Println(line)
	}
	return nil
}

// checkHelperVersion invokes the helper with --version and compares against
// the WSL CLI version. Returns an error if they don't match.
func checkHelperVersion(helperPath string, verbose bool) error {
	if Version == "" || Version == "dev" {
		return nil // Skip check for dev builds.
	}

	out, err := exec.Command(helperPath, "--version").Output()
	if err != nil {
		return fmt.Errorf("querying host helper version: %w", err)
	}
	hostVersion := strings.TrimSpace(string(out))

	if verbose {
		fmt.Fprintf(os.Stderr, "Version: wstart=%s, wstart-host=%s\n", Version, hostVersion)
	}

	if hostVersion != Version {
		return fmt.Errorf("version mismatch: wstart %s, wstart-host.exe %s\n"+
			"Run install-host.ps1 and install-wsl.sh to update both binaries", Version, hostVersion)
	}
	return nil
}

func findHelper() (string, error) {
	// 1. $WSTART_HOST_PATH environment variable.
	if p := os.Getenv("WSTART_HOST_PATH"); p != "" {
		return p, nil
	}

	// 2. Well-known locations via wslpath translation.
	candidates := helperCandidates()
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf(
		"wstart-host.exe not found. Run install-host.ps1 or set $WSTART_HOST_PATH.\n"+
			"Searched: %s", strings.Join(candidates, ", "),
	)
}

func helperCandidates() []string {
	var candidates []string

	// Try to resolve %LOCALAPPDATA% via cmd.exe.
	out, err := exec.Command("cmd.exe", "/C", "echo", "%LOCALAPPDATA%").Output()
	if err == nil {
		winPath := strings.TrimSpace(strings.ReplaceAll(string(out), "\r", ""))
		if winPath != "" && winPath != "%LOCALAPPDATA%" {
			wslOut, err := exec.Command("wslpath", "-u", winPath).Output()
			if err == nil {
				localAppData := strings.TrimSpace(string(wslOut))
				candidates = append(candidates, filepath.Join(localAppData, "wstart", "wstart-host.exe"))
			}
		}
	}

	// Also check relative to the wstart binary itself (for portable installs).
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "wstart-host.exe"))
	}

	return candidates
}

func collectEnvVars(cfg *config.Config) map[string]string {
	if len(cfg.Env.Forward) == 0 {
		return nil
	}

	blocked := make(map[string]bool)
	for _, b := range cfg.Env.Block {
		blocked[strings.ToUpper(b)] = true
	}

	vars := make(map[string]string)
	for _, name := range cfg.Env.Forward {
		if blocked[strings.ToUpper(name)] {
			continue
		}
		if val, ok := os.LookupEnv(name); ok {
			vars[name] = val
		}
	}

	if len(vars) == 0 {
		return nil
	}
	return vars
}

// invokeHelperExec runs the helper in --exec mode with stdio passthrough.
// Program output flows directly to the terminal. Returns the child's exit code.
func invokeHelperExec(helperPath string, req *protocol.LaunchRequest) (int, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return 1, fmt.Errorf("encoding request: %w", err)
	}

	cmd := exec.Command(helperPath, "--exec")
	cmd.Stdin = bytes.NewReader(reqData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("helper exec failed: %w", err)
	}
	return 0, nil
}

func invokeHelper(helperPath string, req *protocol.LaunchRequest) (*protocol.LaunchResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	cmd := exec.Command(helperPath, "--launch")
	cmd.Stdin = bytes.NewReader(reqData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("helper failed: %s", stderrStr)
		}
		return nil, fmt.Errorf("helper failed: %w", err)
	}

	var resp protocol.LaunchResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w (raw: %s)", err, stdout.String())
	}

	return &resp, nil
}
