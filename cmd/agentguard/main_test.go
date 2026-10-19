//go:build linux || darwin

package main

import (
	"os"
	"path/filepath"
	"testing"
)

const validPolicy = `version: 1
rules: {
	"all": { path: "**", actions: ["read"], effect: "allow", recursive: true }
}
limits: { maxReadBytes: 1024, maxWriteBytes: 1024 }
`

// writePolicy は dir に有効なポリシーを書き、そのパスを返す。
func writePolicy(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(validPolicy), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunRequiresRootAndPolicy(t *testing.T) {
	if code := run(nil); code != 2 {
		t.Errorf("run(no flags) = %d, want 2", code)
	}
	if code := run([]string{"--root", t.TempDir()}); code != 2 {
		t.Errorf("run(no policy) = %d, want 2", code)
	}
}

func TestRunFailsOnInvalidPolicy(t *testing.T) {
	base := t.TempDir()
	bad := filepath.Join(base, "bad.cue")
	if err := os.WriteFile(bad, []byte("version: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{"--root", t.TempDir(), "--policy", bad}); code != 1 {
		t.Errorf("run(invalid policy) = %d, want 1", code)
	}
}

func TestRunRefusesPolicyInsideRoot(t *testing.T) {
	root := t.TempDir()
	policy := writePolicy(t, root, "policy.cue")

	if code := run([]string{"--root", root, "--policy", policy}); code != 1 {
		t.Errorf("run(policy inside root) = %d, want 1", code)
	}
}
