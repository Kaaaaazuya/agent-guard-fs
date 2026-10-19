//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

// samplePolicy は docs と README が案内するサンプルのポリシーとサンプルのプロジェクト。
// 文書と実際の挙動が食い違わないように、起動の経路で確かめる。
const (
	samplePolicy  = "../../examples/basic/policy.cue"
	sampleProject = "../../examples/basic/project"
)

// copySample はサンプルのプロジェクトを一時ディレクトリにコピーする。隠しファイルも含める。
func copySample(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(sampleProject, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sampleProject, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		data, err := os.ReadFile(path) //nolint:gosec // paths come from walking the fixed sample directory
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600) //nolint:gosec // target is under the test temp directory
	})
	if err != nil {
		t.Fatalf("copy sample project: %v", err)
	}
	return dst
}

func TestSampleProjectPolicy(t *testing.T) {
	root := copySample(t)
	g, err := startup(root, samplePolicy)
	if err != nil {
		t.Fatalf("startup() error = %v", err)
	}
	defer func() { _ = g.Close() }()
	ctx := context.Background()

	t.Run("src can be read and written", func(t *testing.T) {
		if _, err := g.Read(ctx, "src/main.go"); err != nil {
			t.Errorf("Read(src/main.go) error = %v", err)
		}
		if _, err := g.Write(ctx, "src/extra.go", []byte("package main\n")); err != nil {
			t.Errorf("Write(src/extra.go) error = %v", err)
		}
	})

	denied := []struct {
		name string
		run  func() error
		rule string
	}{
		{"docs is read-only", func() error {
			_, err := g.Write(ctx, "docs/new.md", []byte("x"))
			return err
		}, ""},
		{"env file in root is unreadable", func() error {
			_, err := g.Read(ctx, ".env")
			return err
		}, "env-deny"},
		{"env file in src is unreadable", func() error {
			_, err := g.Read(ctx, "src/.env")
			return err
		}, "env-deny"},
		{"go.mod cannot be written", func() error {
			_, err := g.Write(ctx, "go.mod", []byte("module x\n"))
			return err
		}, "go-mod-deny"},
		{".git cannot be read", func() error {
			_, err := g.Read(ctx, ".git/config")
			return err
		}, "git-deny"},
	}
	for _, tc := range denied {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if errs.CodeOf(err) != errs.PolicyDenied {
				t.Fatalf("error = %v, want policy_denied", err)
			}
			if tc.rule != "" {
				if e, _ := errors.AsType[*errs.Error](err); e.Rule != tc.rule {
					t.Errorf("rule = %q, want %q", e.Rule, tc.rule)
				}
			}
		})
	}

	t.Run("root listing hides env files", func(t *testing.T) {
		res, err := g.List(ctx, ".")
		if err != nil {
			t.Fatalf("List(.) error = %v", err)
		}
		var names []string
		for _, e := range res.Entries {
			names = append(names, e.Name)
		}
		want := []string{"docs", "go.mod", "src"}
		if !slices.Equal(names, want) {
			t.Errorf("entries = %v, want %v", names, want)
		}
	})
}
