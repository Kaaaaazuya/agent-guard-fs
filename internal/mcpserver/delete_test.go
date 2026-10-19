//go:build linux || darwin

package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDeleteFileTool(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "src", "old.txt")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, dir)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_file",
		Arguments: map[string]any{"path": "src/old.txt"},
	})
	if err != nil || res.IsError {
		t.Fatalf("delete_file = %+v, %v", res, err)
	}
	if got := res.StructuredContent.(map[string]any)["path"]; got != "src/old.txt" {
		t.Errorf("path = %v", got)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("file still exists after delete_file: %v", err)
	}
}

func TestDeleteFileDeniedLeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, dir)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_file",
		Arguments: map[string]any{"path": ".env"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if structured := res.StructuredContent.(map[string]any); !res.IsError || structured["code"] != "policy_denied" {
		t.Errorf("result = %+v, want policy_denied", structured)
	}
	if _, err := os.Lstat(envPath); err != nil {
		t.Errorf(".env was removed by a denied delete: %v", err)
	}
}
