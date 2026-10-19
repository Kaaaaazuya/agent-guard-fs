//go:build linux || darwin

package guard

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/filesystem"
)

func TestListHidesDeniedEntries(t *testing.T) {
	// src の直下は読み書きできるが、どの階層の .env も list を拒否する。
	g, dir := newGuard(t, srcAndEnvRules, 1000)
	src := filepath.Join(dir, "src")
	if err := os.Mkdir(src, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.go", ".env", ".env.local"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := g.List(context.Background(), "src")
	if err != nil {
		t.Fatalf("List(src) error = %v", err)
	}
	want := []filesystem.Entry{{Name: "main.go", Type: "file"}}
	if !reflect.DeepEqual(got.Entries, want) {
		t.Errorf("List(src) entries = %v, want %v", got.Entries, want)
	}
	if got.Truncated {
		t.Error("List(src) truncated = true, want false")
	}
}

func TestListRequiresListOnDirectory(t *testing.T) {
	g, dir := newGuard(t, `
	"read-only": { path: "docs", actions: ["read"], effect: "allow", recursive: true }
`, 1000)
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o750); err != nil {
		t.Fatal(err)
	}

	_, err := g.List(context.Background(), "docs")
	wantCode(t, err, errs.PolicyDenied)
}

func TestListTruncatesAfterFiltering(t *testing.T) {
	g, dir := newGuard(t, srcAndEnvRules, 2)
	src := filepath.Join(dir, "src")
	if err := os.Mkdir(src, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "c", ".env"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := g.List(context.Background(), "src")
	if err != nil {
		t.Fatalf("List(src) error = %v", err)
	}
	// .env は除外してから件数を数えるので、残りの a と b だけが返り、c は切り捨てられる。
	want := []filesystem.Entry{{Name: "a", Type: "file"}, {Name: "b", Type: "file"}}
	if !reflect.DeepEqual(got.Entries, want) || !got.Truncated {
		t.Errorf("List(src) = %+v, want entries %v and truncated", got, want)
	}
}

func TestListRootHidesEnvAtAnyDepth(t *testing.T) {
	// 直下のエントリは、それぞれ list の判定を受ける。.env は deny ルールで隠れる。
	g, dir := newGuard(t, `
	"all": { path: "**", actions: ["list"], effect: "allow", recursive: true }
	"env": { path: "**/.env*", actions: ["list"], effect: "deny" }
`, 1000)
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := g.List(context.Background(), ".")
	if err != nil {
		t.Fatalf("List(.) error = %v", err)
	}
	want := []filesystem.Entry{{Name: "README", Type: "file"}}
	if !reflect.DeepEqual(got.Entries, want) {
		t.Errorf("List(.) entries = %v, want %v", got.Entries, want)
	}
}
