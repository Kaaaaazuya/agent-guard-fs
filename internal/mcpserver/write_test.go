//go:build linux || darwin

package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWriteFileToolCreatesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, dir)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "write_file",
		Arguments: map[string]any{"path": "src/a.txt", "content": "one"},
	})
	if err != nil || res.IsError {
		t.Fatalf("write_file(new) = %+v, %v", res, err)
	}
	if got := res.StructuredContent.(map[string]any); got["created"] != true || got["bytes"] != float64(3) {
		t.Errorf("structured = %v, want created true and 3 bytes", got)
	}

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "write_file",
		Arguments: map[string]any{"path": "src/a.txt", "content": "two"},
	})
	if err != nil || res.IsError {
		t.Fatalf("write_file(replace) = %+v, %v", res, err)
	}
	if got := res.StructuredContent.(map[string]any); got["created"] != false {
		t.Errorf("structured = %v, want created false", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "src", "a.txt")); string(b) != "two" { //nolint:gosec // test-owned path
		t.Errorf("file = %q, want two", b)
	}
}

func TestWriteFileDeniedLeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, dir)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "write_file",
		Arguments: map[string]any{"path": ".env", "content": "pwned"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	structured := res.StructuredContent.(map[string]any)
	if !res.IsError || structured["code"] != "policy_denied" || structured["rule"] != "env" {
		t.Errorf("result = %+v, want policy_denied from rule env", structured)
	}
	if b, _ := os.ReadFile(envPath); string(b) != "SECRET=1" { //nolint:gosec // test-owned path
		t.Errorf(".env = %q, want unchanged", b)
	}
}

func TestWriteFileAdvertisesDestructive(t *testing.T) {
	cs := connect(t, t.TempDir())

	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	for _, tool := range list.Tools {
		if tool.Name == "write_file" && (tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint) {
			t.Errorf("write_file annotations = %+v, want destructiveHint", tool.Annotations)
		}
	}
}
