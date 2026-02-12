package pathconv

import (
	"testing"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

func TestApplyAlias(t *testing.T) {
	drives := []protocol.DriveInfo{
		{Letter: "P", Type: protocol.DriveSubst, Target: `C:\dev\workspace`},
		{Letter: "Z", Type: protocol.DriveSubst, Target: `D:\assets\shared`},
	}
	configAliases := map[string]string{
		"Y": `C:\dev\workspace\deep\nested`, // More specific than P:
	}

	conv := NewConverter(drives, configAliases, true)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "subst match with remainder",
			input: `C:\dev\workspace\project\file.txt`,
			want:  `P:\project\file.txt`,
		},
		{
			name:  "subst exact match",
			input: `C:\dev\workspace`,
			want:  `P:`,
		},
		{
			name:  "longer config alias wins",
			input: `C:\dev\workspace\deep\nested\thing`,
			want:  `Y:\thing`,
		},
		{
			name:  "no match passes through",
			input: `C:\other\path`,
			want:  `C:\other\path`,
		},
		{
			name:  "case insensitive match",
			input: `c:\DEV\WORKSPACE\project`,
			want:  `P:\project`,
		},
		{
			name:  "different drive subst",
			input: `D:\assets\shared\textures`,
			want:  `Z:\textures`,
		},
		{
			name:  "partial directory name no match",
			input: `C:\dev\workspace2\file.txt`,
			want:  `C:\dev\workspace2\file.txt`,
		},
		{
			name:  "forward slashes normalized",
			input: `C:/dev/workspace/project`,
			want:  `P:\project`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := conv.applyAlias(tt.input)
			if got != tt.want {
				t.Errorf("applyAlias(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToWindowsURLPassthrough(t *testing.T) {
	conv := NewConverter(nil, nil, true)

	urls := []string{
		"http://example.com",
		"https://google.com/search?q=test",
		"HTTP://UPPERCASE.COM",
	}

	for _, url := range urls {
		got, err := conv.ToWindows(url)
		if err != nil {
			t.Errorf("ToWindows(%q) error: %v", url, err)
			continue
		}
		if got != url {
			t.Errorf("ToWindows(%q) = %q, want passthrough", url, got)
		}
	}
}
