//go:build linux || darwin

package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

func TestWriteCreatesAndReplaces(t *testing.T) {
	root, dir := newRoot(t)
	ctx := context.Background()

	created, err := root.Write(ctx, relOf(t, "new.txt"), []byte("one"), 100)
	if err != nil {
		t.Fatalf("Write(new) error = %v", err)
	}
	if !created {
		t.Error("Write(new) created = false, want true")
	}
	if got := readFile(t, filepath.Join(dir, "new.txt")); got != "one" {
		t.Errorf("new file = %q, want %q", got, "one")
	}

	created, err = root.Write(ctx, relOf(t, "new.txt"), []byte("two"), 100)
	if err != nil {
		t.Fatalf("Write(replace) error = %v", err)
	}
	if created {
		t.Error("Write(replace) created = true, want false")
	}
	if got := readFile(t, filepath.Join(dir, "new.txt")); got != "two" {
		t.Errorf("replaced file = %q, want %q", got, "two")
	}
	assertNoTempFiles(t, dir)
}

func TestWritePreservesExistingMode(t *testing.T) {
	root, dir := newRoot(t)
	path := filepath.Join(dir, "script.sh")
	mustWrite(t, path, "old")
	// 新規作成のモードは 0644 なので、0600 が保たれれば既存モードの引き継ぎを確かめられる。
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := root.Write(context.Background(), relOf(t, "script.sh"), []byte("new"), 100); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %o, want 600", got)
	}
}

func TestWriteErrors(t *testing.T) {
	root, dir := newRoot(t)
	ctx := context.Background()
	mustWrite(t, filepath.Join(dir, "file"), "keep")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(dir, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		content string
		max     int
		code    errs.Code
	}{
		{"size limit", "file", "0123456789", 9, errs.SizeLimitExceeded},
		{"missing parent", "missing/x", "x", 100, errs.NotFound},
		{"parent is a file", "file/x", "x", 100, errs.NotADirectory},
		{"target is a directory", "sub", "x", 100, errs.IsDirectory},
		{"target is a FIFO", "pipe", "x", 100, errs.UnsupportedFileType},
		{"reserved temp name", ".agentguard-tmp-1", "x", 100, errs.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := root.Write(ctx, relOf(t, tt.path), []byte(tt.content), tt.max)
			wantCode(t, err, tt.code)
		})
	}
	if got := readFile(t, filepath.Join(dir, "file")); got != "keep" {
		t.Errorf("file = %q after failed writes, want unchanged", got)
	}
	assertNoTempFiles(t, dir)
}

func TestWriteRefusesSymlinks(t *testing.T) {
	root, dir := newRoot(t)
	ctx := context.Background()
	outside := filepath.Join(filepath.Dir(dir), ".env")
	mustWrite(t, outside, "SECRET=1")
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	// src/leak -> ../.env。書き込みで .env を置き換えてはならない。
	if err := os.Symlink(outside, filepath.Join(dir, "src", "leak")); err != nil {
		t.Fatal(err)
	}
	// ルート内のファイルを指す symlink も置き換えない。
	mustWrite(t, filepath.Join(dir, "real"), "real")
	if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	// 親ディレクトリが symlink の場合も拒否する。
	if err := os.Symlink("src", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{"leaf symlink to outside", "src/leak"},
		{"leaf symlink to inside", "alias"},
		{"symlinked parent", "link/new"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := root.Write(ctx, relOf(t, tt.path), []byte("pwned"), 100)
			wantCode(t, err, errs.UnsupportedFileType)
		})
	}
	if got := readFile(t, outside); got != "SECRET=1" {
		t.Errorf(".env outside the root = %q, want unchanged", got)
	}
	if got := readFile(t, filepath.Join(dir, "real")); got != "real" {
		t.Errorf("real = %q, want unchanged", got)
	}
	assertNoTempFiles(t, dir)
}

func TestWriteCancelledLeavesOriginal(t *testing.T) {
	root, dir := newRoot(t)
	mustWrite(t, filepath.Join(dir, "file"), "keep")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := root.Write(ctx, relOf(t, "file"), []byte("new"), 100)
	if err == nil {
		t.Fatal("Write(cancelled) error = nil, want error")
	}
	if got := readFile(t, filepath.Join(dir, "file")); got != "keep" {
		t.Errorf("file = %q after cancelled write, want unchanged", got)
	}
	assertNoTempFiles(t, dir)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // paths are built by the test itself
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), tempPrefix) {
			t.Errorf("temporary file %q was left behind", e.Name())
		}
	}
}
