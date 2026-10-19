//go:build linux || darwin

package filesystem

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

func TestList(t *testing.T) {
	root, dir := newRoot(t)
	mustWrite(t, filepath.Join(dir, "b.txt"), "b")
	mustWrite(t, filepath.Join(dir, "a.txt"), "a")
	mustWrite(t, filepath.Join(dir, ".agentguard-tmp-123"), "tmp")
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../..", filepath.Join(dir, "up")); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(dir, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := root.List(relOf(t, "."))
	if err != nil {
		t.Fatalf("List(.) error = %v", err)
	}
	want := []Entry{
		{Name: "a.txt", Type: "file"},
		{Name: "b.txt", Type: "file"},
		{Name: "pipe", Type: "other"},
		{Name: "src", Type: "dir"},
		{Name: "up", Type: "symlink"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List(.) = %v, want %v", got, want)
	}
}

func TestListErrors(t *testing.T) {
	root, dir := newRoot(t)
	mustWrite(t, filepath.Join(dir, "file"), "x")
	if err := os.Symlink("src", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		code errs.Code
	}{
		{"missing", "nope", errs.NotFound},
		{"regular file", "file", errs.NotADirectory},
		{"symlink to directory", "link", errs.UnsupportedFileType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := root.List(relOf(t, tt.path))
			wantCode(t, err, tt.code)
		})
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
