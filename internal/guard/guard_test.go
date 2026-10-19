//go:build linux || darwin

package guard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/audit"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/filesystem"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy"
)

// newGuard は rules のポリシーで動く Guard と、そのルートを返す。監査ログは捨てる。
func newGuard(t *testing.T, rules string, maxListEntries int) (*Guard, string) {
	t.Helper()
	return newGuardWithLog(t, rules, maxListEntries, io.Discard)
}

// newGuardWithLog は、監査ログを w に書く Guard と、そのルートを返す。
func newGuardWithLog(t *testing.T, rules string, maxListEntries int, w io.Writer) (*Guard, string) {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "root")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	src := fmt.Sprintf("version: 1\nrules: {\n%s\n}\nlimits: {maxReadBytes: 1024, maxWriteBytes: 1024, maxListEntries: %d}\n", rules, maxListEntries)
	p, err := policy.Load("policy.cue", []byte(src))
	if err != nil {
		t.Fatalf("policy.Load() error = %v", err)
	}
	root, err := filesystem.Open(dir)
	if err != nil {
		t.Fatalf("filesystem.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return New(policy.NewEngine(p), root, p.Limits(), audit.New(w)), dir
}

// snapshot は dir の全ファイルの内容を、相対パスと結び付けて返す。ディレクトリも含む。
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	got := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			got[rel] = "dir"
			return nil
		}
		b, err := os.ReadFile(p) //nolint:gosec // paths come from WalkDir under the test's own directory
		if err != nil {
			return err
		}
		got[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func wantCode(t *testing.T, err error, code errs.Code) {
	t.Helper()
	if got := errs.CodeOf(err); err == nil || got != code {
		t.Fatalf("error = %v (code %q), want code %q", err, got, code)
	}
}

const srcAndEnvRules = `
	"src": { path: "src", actions: ["read", "write", "delete", "list"], effect: "allow", recursive: true }
	"env": { path: "**/.env*", actions: ["read", "write", "delete", "list"], effect: "deny" }
`

func TestAllowedOperations(t *testing.T) {
	g, dir := newGuard(t, srcAndEnvRules, 1000)
	ctx := context.Background()
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}

	w, err := g.Write(ctx, "src/a.go", []byte("package a\n"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if want := (WriteResult{Path: "src/a.go", Bytes: 10, Created: true}); w != want {
		t.Errorf("Write() = %+v, want %+v", w, want)
	}

	r, err := g.Read(ctx, "./src//a.go")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if r.Path != "src/a.go" || string(r.Content) != "package a\n" {
		t.Errorf("Read() = %+v", r)
	}

	p, err := g.Delete(ctx, "src/a.go")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if p != "src/a.go" {
		t.Errorf("Delete() path = %q", p)
	}
}

func TestDeniedOperationsLeaveFilesUnchanged(t *testing.T) {
	g, dir := newGuard(t, srcAndEnvRules, 1000)
	ctx := context.Background()
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)

	_, errRead := g.Read(ctx, ".env")
	_, errWrite := g.Write(ctx, ".env", []byte("pwned"))
	_, errDelete := g.Delete(ctx, ".env")
	for _, err := range []error{errRead, errWrite, errDelete} {
		wantCode(t, err, errs.PolicyDenied)
	}
	if after := snapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Errorf("tree changed after denied operations:\nbefore %v\nafter  %v", before, after)
	}
}

func TestDeniedResponsesDoNotRevealExistence(t *testing.T) {
	g, dir := newGuard(t, srcAndEnvRules, 1000)
	ctx := context.Background()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}

	// .env は存在し、.env.missing は存在しない。どちらも同じ deny ルールに一致する。
	for _, op := range []struct {
		name string
		run  func(string) error
	}{
		{"read", func(p string) error { _, err := g.Read(ctx, p); return err }},
		{"write", func(p string) error { _, err := g.Write(ctx, p, []byte("x")); return err }},
		{"delete", func(p string) error { _, err := g.Delete(ctx, p); return err }},
	} {
		t.Run(op.name, func(t *testing.T) {
			existing := op.run(".env")
			missing := op.run(".env.missing")
			if existing == nil || missing == nil {
				t.Fatalf("errors = %v, %v; want both denied", existing, missing)
			}
			if existing.Error() != missing.Error() {
				t.Errorf("existing = %q, missing = %q; want identical", existing, missing)
			}
		})
	}
}

func TestDefaultDenyForUnlistedPath(t *testing.T) {
	g, dir := newGuard(t, srcAndEnvRules, 1000)
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := g.Read(context.Background(), "notes.txt")
	wantCode(t, err, errs.PolicyDenied)
	var e *errs.Error
	if !errors.As(err, &e) || e.Rule != "" {
		t.Errorf("Rule = %q, want empty for default deny", e.Rule)
	}
}

func TestTempNamesAlwaysDenied(t *testing.T) {
	// 全パスを許可するポリシーでも、一時ファイルの名前は拒否する。
	g, dir := newGuard(t, `"all": { path: "**", actions: ["read", "write", "delete", "list"], effect: "allow", recursive: true }`, 1000)
	name := ".agentguard-tmp-1234"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, errRead := g.Read(ctx, name)
	_, errWrite := g.Write(ctx, name, []byte("y"))
	_, errDelete := g.Delete(ctx, name)
	for _, err := range []error{errRead, errWrite, errDelete} {
		wantCode(t, err, errs.PolicyDenied)
	}
}

func TestCancelledOperationDoesNothing(t *testing.T) {
	g, dir := newGuard(t, srcAndEnvRules, 1000)
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := g.Write(ctx, "src/new.txt", []byte("x")); err == nil {
		t.Fatal("Write(cancelled) error = nil")
	}
	if _, err := os.Lstat(filepath.Join(dir, "src", "new.txt")); !os.IsNotExist(err) {
		t.Errorf("file was created by a cancelled write: %v", err)
	}
}
