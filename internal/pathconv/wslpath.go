package pathconv

import (
	"fmt"
	"os/exec"
	"strings"
)

// runWslpath calls the wslpath utility to translate a WSL path to a Windows path.
func runWslpath(wslPath string) (string, error) {
	cmd := exec.Command("wslpath", "-w", wslPath)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wslpath -w %q failed: %w", wslPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}
