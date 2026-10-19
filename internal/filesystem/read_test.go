//go:build linux || darwin

package filesystem

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// newRoot はテスト用のルートと、その中の作業ディレクトリを返す。ルートの外には
// .env を置き、symlink で指してもルート外のファイルが読めないことを確かめる。
func newRoot(t *testing.T) (*Root, string) {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "root")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".env"), []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root, dir
}

func relOf(t *testing.T, s string) pathutil.RelPath {
	t.Helper()
	rel, err := pathutil.Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", s, err)
	}
	return rel
}

// wantCode は err が code を持つ *errs.Error であることを確かめる。
func wantCode(t *testing.T, err error, code errs.Code) {
	t.Helper()
	if got := errs.CodeOf(err); err == nil || got != code {
		t.Fatalf("error = %v (code %q), want code %q", err, got, code)
	}
}

func TestRead(t *testing.T) {
	root, dir := newRoot(t)
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "a.go"), []byte("package a\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := root.Read(relOf(t, "src/a.go"), 1024)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !bytes.Equal(got, []byte("package a\n")) {
		t.Errorf("Read() = %q", got)
	}
}

func TestReadErrors(t *testing.T) {
	root, dir := newRoot(t)
	write := func(name string, content []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("file", []byte("x"))
	write("big", []byte(strings.Repeat("a", 11)))
	write("bad", []byte{0xff, 0xfe})
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		code errs.Code
	}{
		{"missing", "nope", errs.NotFound},
		{"directory", "sub", errs.IsDirectory},
		{"root directory", ".", errs.IsDirectory},
		{"file as parent", "file/x", errs.NotADirectory},
		{"over the size limit", "big", errs.SizeLimitExceeded},
		{"invalid UTF-8", "bad", errs.UnsupportedContent},
		{"reserved temp name", ".agentguard-tmp-1", errs.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := root.Read(relOf(t, tt.path), 10)
			wantCode(t, err, tt.code)
		})
	}
}

func TestReadSizeLimitBoundary(t *testing.T) {
	root, dir := newRoot(t)
	if err := os.WriteFile(filepath.Join(dir, "exact"), []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := root.Read(relOf(t, "exact"), 10); err != nil {
		t.Errorf("Read(exactly max bytes) error = %v", err)
	}
	_, err := root.Read(relOf(t, "exact"), 9)
	wantCode(t, err, errs.SizeLimitExceeded)
}

func TestReadRefusesSymlinks(t *testing.T) {
	root, dir := newRoot(t)
	outside := filepath.Join(filepath.Dir(dir), ".env")
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	// ルート内の symlink がルート外の .env を指す場合（DESIGN §9.1 の `src/leak -> ../.env`）。
	if err := os.Symlink(outside, filepath.Join(dir, "src", "leak")); err != nil {
		t.Fatal(err)
	}
	// ルート内のファイルを指す symlink も、解決先が安全でも拒否する。
	if err := os.WriteFile(filepath.Join(dir, "real"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	// 途中の要素が symlink（ルート内のディレクトリを指す）でも拒否する。
	if err := os.Symlink("src", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"src/leak", "alias", "link/leak"} {
		t.Run(p, func(t *testing.T) {
			_, err := root.Read(relOf(t, p), 1024)
			wantCode(t, err, errs.UnsupportedFileType)
		})
	}
}

func TestReadRefusesFIFO(t *testing.T) {
	root, dir := newRoot(t)
	if err := unix.Mkfifo(filepath.Join(dir, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}

	// O_NONBLOCK で開くので待たずに返り、種別で拒否する。
	_, err := root.Read(relOf(t, "pipe"), 1024)
	wantCode(t, err, errs.UnsupportedFileType)
}
