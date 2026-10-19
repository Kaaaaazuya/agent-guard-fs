//go:build linux || darwin

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExplain(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		code      int
		wantLines []string
	}{
		{
			name:      "allowed by a rule",
			args:      []string{"read", "src/main.go"},
			wantLines: []string{"allow", "rule:   source-read", "reason: allow rule matched"},
		},
		{
			name:      "denied by a deny rule over a broader allow",
			args:      []string{"read", "src/.env"},
			wantLines: []string{"deny", "rule:   env-deny", "reason: deny rule matched"},
		},
		{
			name:      "denied by default",
			args:      []string{"read", "other/file.txt"},
			wantLines: []string{"deny", "rule:   (none)", "reason: no allow rule matched"},
		},
		{
			name:      "path is normalized before matching",
			args:      []string{"write", "./src//main.go"},
			wantLines: []string{"allow", "path:   src/main.go", "rule:   source-write"},
		},
		{name: "unknown action", args: []string{"exec", "src"}, code: 2},
		{name: "path outside the root", args: []string{"read", "../x"}, code: 2},
		{name: "missing path", args: []string{"read"}, code: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			args := append([]string{"--policy", samplePolicy}, tt.args...)
			if code := runExplain(args, &out); code != tt.code {
				t.Fatalf("runExplain = %d, want %d\n%s", code, tt.code, out.String())
			}
			for _, want := range tt.wantLines {
				if !strings.Contains(out.String(), want+"\n") {
					t.Errorf("output lacks %q:\n%s", want, out.String())
				}
			}
		})
	}
}

func TestExplainFailsOnInvalidPolicy(t *testing.T) {
	var out bytes.Buffer
	policy := writePolicy(t, t.TempDir(), "policy.cue")
	if code := runExplain([]string{"--policy", policy + ".missing", "read", "a"}, &out); code != 1 {
		t.Errorf("runExplain(missing policy) = %d, want 1", code)
	}
}

func TestRunDispatchesExplain(t *testing.T) {
	if code := run([]string{"explain"}); code != 2 {
		t.Errorf("run(explain without args) = %d, want 2", code)
	}
}
