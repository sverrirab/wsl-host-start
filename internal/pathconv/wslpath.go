package pathconv

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runWslpath calls the wslpath utility to translate a WSL path to a Windows path.
func runWslpath(wslPath string) (string, error) {
	cmd := exec.Command("wslpath", "-w", wslPath)
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("wslpath not found: not running in WSL?")
		}
		return "", fmt.Errorf("wslpath -w %q failed: %w", wslPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}
