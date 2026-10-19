//go:build linux || darwin

package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListFilesToolHidesDeniedEntries(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.Mkdir(src, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.go", ".env"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cs := connect(t, dir)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_files",
		Arguments: map[string]any{"path": "src"},
	})
	if err != nil || res.IsError {
		t.Fatalf("list_files = %+v, %v", res, err)
	}
	structured := res.StructuredContent.(map[string]any)
	entries := structured["entries"].([]any)
	if len(entries) != 1 || entries[0].(map[string]any)["name"] != "main.go" {
		t.Errorf("entries = %v, want only main.go", entries)
	}
	if structured["truncated"] != false {
		t.Errorf("truncated = %v, want false", structured["truncated"])
	}
	if got := textOf(t, res); got != "main.go (file)" {
		t.Errorf("text = %q, want one line per entry", got)
	}
}

func TestListFilesToolEmptyDirectoryHasEmptyArray(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, dir)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_files",
		Arguments: map[string]any{"path": "src"},
	})
	if err != nil || res.IsError {
		t.Fatalf("list_files = %+v, %v", res, err)
	}
	if entries := res.StructuredContent.(map[string]any)["entries"]; entries == nil {
		t.Errorf("entries = nil, want an empty array")
	}
}
