package allowlist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckNoAllowlist(t *testing.T) {
	lr := &LoadResult{Loaded: false}
	if err := lr.Check("anything.exe", []string{"whatever"}); err != nil {
		t.Errorf("no allowlist should allow everything, got: %v", err)
	}
}

func TestCheckProgramAllowed(t *testing.T) {
	lr := &LoadResult{
		Loaded: true,
		List: &List{
			Allow: []Rule{
				{Program: "notepad.exe"},
				{Program: "explorer"},
			},
		},
	}

	tests := []struct {
		file    string
		wantErr bool
	}{
		{"notepad.exe", false},
		{"NOTEPAD.EXE", false},
		{"notepad", false},
		{`C:\Windows\System32\notepad.exe`, false},
		{"explorer.exe", false},
		{"explorer", false},
		{"cmd.exe", true},
		{"p4.exe", true},
	}

	for _, tt := range tests {
		err := lr.Check(tt.file, nil)
		if (err != nil) != tt.wantErr {
			t.Errorf("Check(%q): err=%v, wantErr=%v", tt.file, err, tt.wantErr)
		}
	}
}

func TestCheckSubcommands(t *testing.T) {
	lr := &LoadResult{
		Loaded: true,
		List: &List{
			Allow: []Rule{
				{
					Program:  "p4",
					Commands: []string{"edit", "diff", "sync", "submit", "revert", "info"},
				},
			},
		},
	}

	tests := []struct {
		name    string
		file    string
		args    []string
		wantErr bool
	}{
		{
			name: "allowed subcommand",
			file: "p4", args: []string{"edit", "//depot/file.txt"},
			wantErr: false,
		},
		{
			name: "allowed subcommand case insensitive",
			file: "p4.exe", args: []string{"SYNC"},
			wantErr: false,
		},
		{
			name: "flags before subcommand",
			file: "p4", args: []string{"-c", "myclient", "edit", "file.txt"},
			wantErr: false,
		},
		{
			name: "denied subcommand",
			file: "p4", args: []string{"open", "file.txt"},
			wantErr: true,
		},
		{
			name: "denied - no subcommand given",
			file: "p4", args: []string{},
			wantErr: true,
		},
		{
			name: "denied - only flags no subcommand",
			file: "p4", args: []string{"-V"},
			wantErr: true,
		},
		{
			name: "full path match",
			file: `C:\Program Files\Perforce\p4.exe`, args: []string{"info"},
			wantErr: false,
		},
		{
			name: "different program denied",
			file: "git", args: []string{"push"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lr.Check(tt.file, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check(%q, %v): err=%v, wantErr=%v", tt.file, tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestFirstPositionalArg(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"edit", "file.txt"}, "edit"},
		{[]string{"-c", "client", "edit", "file.txt"}, "edit"},
		{[]string{"--port=ssl:host:1666", "sync"}, "sync"},
		{[]string{"-V"}, ""},
		{[]string{}, ""},
		{[]string{"-u", "user", "-c", "client", "diff", "-du"}, "diff"},
	}

	for _, tt := range tests {
		got := firstPositionalArg(tt.args)
		if got != tt.want {
			t.Errorf("firstPositionalArg(%v) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.toml")

	content := `
[[allow]]
program = "p4"
commands = ["edit", "sync"]

[[allow]]
program = "notepad.exe"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	lr, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !lr.Loaded {
		t.Fatal("expected allowlist to be loaded")
	}
	if len(lr.List.Allow) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(lr.List.Allow))
	}

	// p4 edit allowed
	if err := lr.Check("p4", []string{"edit", "file.txt"}); err != nil {
		t.Errorf("p4 edit should be allowed: %v", err)
	}
	// p4 open denied
	if err := lr.Check("p4", []string{"open", "file.txt"}); err == nil {
		t.Error("p4 open should be denied")
	}
	// notepad allowed with any args
	if err := lr.Check("notepad.exe", []string{"C:\\file.txt"}); err != nil {
		t.Errorf("notepad should be allowed: %v", err)
	}
	// git denied
	if err := lr.Check("git", []string{"push"}); err == nil {
		t.Error("git should be denied")
	}
}

func TestLoadMissing(t *testing.T) {
	lr, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if lr.Loaded {
		t.Fatal("expected no allowlist loaded for empty dir")
	}
	// Everything allowed when no file exists
	if err := lr.Check("anything", []string{"whatever"}); err != nil {
		t.Errorf("should allow everything: %v", err)
	}
}
