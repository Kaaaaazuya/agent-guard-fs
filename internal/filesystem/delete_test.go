//go:build linux || darwin

package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

func TestDelete(t *testing.T) {
	root, dir := newRoot(t)
	mustWrite(t, filepath.Join(dir, "gone"), "x")

	if err := root.Delete(relOf(t, "gone")); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "gone")); !os.IsNotExist(err) {
		t.Errorf("file still exists after Delete: %v", err)
	}
}

func TestDeleteErrors(t *testing.T) {
	root, dir := newRoot(t)
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(dir, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, ".agentguard-tmp-keep"), "x")

	tests := []struct {
		name string
		path string
		code errs.Code
	}{
		{"missing", "nope", errs.NotFound},
		{"directory", "sub", errs.IsDirectory},
		{"root directory", ".", errs.IsDirectory},
		{"FIFO", "pipe", errs.UnsupportedFileType},
		{"reserved temp name", ".agentguard-tmp-keep", errs.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantCode(t, root.Delete(relOf(t, tt.path)), tt.code)
		})
	}
	for _, kept := range []string{"sub", "pipe", ".agentguard-tmp-keep"} {
		if _, err := os.Lstat(filepath.Join(dir, kept)); err != nil {
			t.Errorf("%s was removed by a refused delete: %v", kept, err)
		}
	}
}

func TestDeleteRefusesSymlinks(t *testing.T) {
	root, dir := newRoot(t)
	outside := filepath.Join(filepath.Dir(dir), ".env")
	mustWrite(t, outside, "SECRET=1")
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "src", "leak")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("src", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	wantCode(t, root.Delete(relOf(t, "src/leak")), errs.UnsupportedFileType)
	wantCode(t, root.Delete(relOf(t, "link/leak")), errs.UnsupportedFileType)
	if got := readFile(t, outside); got != "SECRET=1" {
		t.Errorf(".env outside the root = %q, want unchanged", got)
	}
}
