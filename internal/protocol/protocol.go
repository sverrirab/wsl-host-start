// Package protocol defines the JSON types shared between the WSL CLI and Windows helper.
package protocol

// LaunchRequest is sent from the WSL CLI to the Windows helper over stdin.
type LaunchRequest struct {
	File    string            `json:"file"`
	Verb    string            `json:"verb"`
	Args    []string          `json:"args,omitempty"`
	WorkDir string            `json:"workDir,omitempty"`
	Show    string            `json:"show"`
	Wait    bool              `json:"wait"`
	EnvVars map[string]string `json:"envVars,omitempty"`
}

// LaunchResponse is returned from the Windows helper to the WSL CLI over stdout.
type LaunchResponse struct {
	ExitCode int    `json:"exitCode"`
	PID      int    `json:"pid"`
	Error    string `json:"error,omitempty"`
	ErrCode  int    `json:"errCode,omitempty"`
}

// DriveInfo describes a single Windows drive letter.
type DriveInfo struct {
	Letter string `json:"letter"`
	Type   string `json:"type"`
	Target string `json:"target,omitempty"`
	Label  string `json:"label,omitempty"`
}

// DrivesResponse is returned by the Windows helper in --drives mode.
type DrivesResponse struct {
	Drives       []DriveInfo `json:"drives"`
	Username     string      `json:"username"`
	LocalAppData string      `json:"localAppData"`
}

// Show mode constants matching the JSON "show" field.
const (
	ShowNormal = "normal"
	ShowMin    = "min"
	ShowMax    = "max"
	ShowHidden = "hidden"
)

// Drive type constants matching the JSON "type" field.
const (
	DriveFixed     = "fixed"
	DriveNetwork   = "network"
	DriveSubst     = "subst"
	DriveRemovable = "removable"
	DriveCDROM     = "cdrom"
	DriveRAMDisk   = "ramdisk"
	DriveUnknown   = "unknown"
)
