//go:build linux || darwin

package guard

import (
	"context"
	"testing"
)

func TestCloseReleasesRoot(t *testing.T) {
	g, _ := newGuard(t, srcAndEnvRules, 1000)
	if err := g.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	// 閉じた後の操作は、ルートの記述子が無効なのでエラーになる。
	if _, err := g.Read(context.Background(), "src/a"); err == nil {
		t.Error("Read() after Close succeeded, want error")
	}
}
