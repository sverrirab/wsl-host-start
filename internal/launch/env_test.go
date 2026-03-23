package launch_test

import (
	"testing"

	"github.com/sverrirab/wsl-host-start/internal/config"
	"github.com/sverrirab/wsl-host-start/internal/launch"
)

func TestCollectEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		forward []string
		block   []string
		setEnv  map[string]string // vars to set before the call
		want    map[string]string // nil means expect nil return
	}{
		{
			name:    "blocked var filtered out",
			forward: []string{"P4PASSWD", "P4PORT"},
			block:   []string{"P4PASSWD"},
			setEnv:  map[string]string{"P4PASSWD": "secret", "P4PORT": "ssl:host:1666"},
			want:    map[string]string{"P4PORT": "ssl:host:1666"},
		},
		{
			name:    "block is case insensitive",
			forward: []string{"p4passwd"},
			block:   []string{"P4PASSWD"},
			setEnv:  map[string]string{"p4passwd": "secret"},
			want:    nil,
		},
		{
			name:    "forward empty returns nil",
			forward: []string{},
			block:   []string{"P4PASSWD"},
			setEnv:  map[string]string{"P4PASSWD": "secret"},
			want:    nil,
		},
		{
			name:    "var in forward but not set in env",
			forward: []string{"MISSING_VAR"},
			block:   []string{},
			setEnv:  map[string]string{},
			want:    nil,
		},
		{
			name:    "block does not affect non-forwarded vars",
			forward: []string{},
			block:   []string{"SECRET"},
			setEnv:  map[string]string{"SECRET": "value"},
			want:    nil,
		},
		{
			name:    "same var in forward and block - block wins",
			forward: []string{"P4TICKETS"},
			block:   []string{"P4TICKETS"},
			setEnv:  map[string]string{"P4TICKETS": "/tmp/tickets"},
			want:    nil,
		},
		{
			name:    "multiple forwarded vars with partial block",
			forward: []string{"P4PORT", "P4CLIENT", "P4PASSWD", "P4USER"},
			block:   []string{"P4PASSWD", "P4TICKETS"},
			setEnv:  map[string]string{"P4PORT": "ssl:host:1666", "P4CLIENT": "myclient", "P4PASSWD": "secret", "P4USER": "joe"},
			want:    map[string]string{"P4PORT": "ssl:host:1666", "P4CLIENT": "myclient", "P4USER": "joe"},
		},
		{
			name:    "all forwarded vars are blocked",
			forward: []string{"P4PASSWD", "P4TRUST"},
			block:   []string{"P4PASSWD", "P4TRUST"},
			setEnv:  map[string]string{"P4PASSWD": "secret", "P4TRUST": "/tmp/trust"},
			want:    nil,
		},
		{
			name:    "no block list - all forwarded vars pass through",
			forward: []string{"P4PORT", "P4CLIENT"},
			block:   []string{},
			setEnv:  map[string]string{"P4PORT": "ssl:host:1666", "P4CLIENT": "myclient"},
			want:    map[string]string{"P4PORT": "ssl:host:1666", "P4CLIENT": "myclient"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars for this test case.
			for k, v := range tt.setEnv {
				t.Setenv(k, v)
			}

			cfg := &config.Config{
				Env: config.EnvConfig{
					Forward: tt.forward,
					Block:   tt.block,
				},
			}

			got := launch.CollectEnvVars(cfg)

			if tt.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("expected %v, got nil", tt.want)
			}

			if len(got) != len(tt.want) {
				t.Errorf("length mismatch: got %d, want %d\n  got:  %v\n  want: %v", len(got), len(tt.want), got, tt.want)
			}

			for k, wantV := range tt.want {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("missing key %q (want %q)", k, wantV)
				} else if gotV != wantV {
					t.Errorf("key %q: got %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}
