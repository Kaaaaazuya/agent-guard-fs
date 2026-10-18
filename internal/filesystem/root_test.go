//go:build linux || darwin

package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRoot(t *testing.T) {
	dir := t.TempDir()

	root, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dir, err)
	}
	if err := root.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestOpenRootRejectsNonDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if root, err := Open(file); err == nil {
		root.Close()
		t.Fatal("Open(file) succeeded, want error")
	}
}

func TestOpenRootRejectsMissingDirectory(t *testing.T) {
	if root, err := Open(filepath.Join(t.TempDir(), "missing")); err == nil {
		root.Close()
		t.Fatal("Open(missing) succeeded, want error")
	}
}
