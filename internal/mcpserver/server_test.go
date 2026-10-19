//go:build linux || darwin

package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/audit"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/filesystem"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/guard"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy"
)

const testRules = `
	"src": { path: "src", actions: ["read", "write", "delete", "list"], effect: "allow", recursive: true }
	"env": { path: "**/.env*", actions: ["read", "write", "delete", "list"], effect: "deny" }
`

// connect は testRules のポリシーと dir をルートにした MCP サーバーに、in-memory のクライアントをつなぐ。
func connect(t *testing.T, dir string) *mcp.ClientSession {
	t.Helper()
	src := "version: 1\nrules: {\n" + testRules + "\n}\nlimits: {maxReadBytes: 1024, maxWriteBytes: 1024, maxListEntries: 1000}\n"
	p, err := policy.Load("policy.cue", []byte(src))
	if err != nil {
		t.Fatalf("policy.Load() error = %v", err)
	}
	root, err := filesystem.Open(dir)
	if err != nil {
		t.Fatalf("filesystem.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	g := guard.New(policy.NewEngine(p), root, p.Limits(), audit.New(io.Discard))

	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := New(g).Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server Connect() error = %v", err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestReadFileTool(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "a.go"), []byte("package a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, dir)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_file",
		Arguments: map[string]any{"path": "src/a.go"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if res.IsError {
		t.Fatalf("CallTool() isError = true: %+v", res.Content)
	}
	if got := textOf(t, res); got != "package a\n" {
		t.Errorf("text = %q", got)
	}
	structured := res.StructuredContent.(map[string]any)
	if structured["path"] != "src/a.go" || structured["size"] != float64(10) {
		t.Errorf("structured = %v", structured)
	}
}

func TestReadFileDeniedReturnsCodeAndRule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, dir)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_file",
		Arguments: map[string]any{"path": ".env"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !res.IsError {
		t.Fatal("isError = false, want true")
	}
	structured := res.StructuredContent.(map[string]any)
	if structured["code"] != "policy_denied" || structured["rule"] != "env" {
		t.Errorf("structured = %v, want code policy_denied and rule env", structured)
	}
	if strings.Contains(textOf(t, res), dir) {
		t.Errorf("error text contains the absolute root: %q", textOf(t, res))
	}
}

func TestReadFileRejectsBadArguments(t *testing.T) {
	cs := connect(t, t.TempDir())

	for name, args := range map[string]map[string]any{
		"unknown field": {"path": "a", "extra": true},
		"wrong type":    {"path": 3},
		"empty path":    {"path": ""},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "read_file", Arguments: args})
			if err != nil {
				t.Fatalf("CallTool() error = %v", err)
			}
			if !res.IsError {
				t.Fatal("isError = false, want true")
			}
		})
	}
}

func TestReadFileAdvertisesReadOnly(t *testing.T) {
	cs := connect(t, t.TempDir())

	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	for _, tool := range list.Tools {
		if tool.Name == "read_file" && (tool.Annotations == nil || !tool.Annotations.ReadOnlyHint) {
			t.Errorf("read_file annotations = %+v, want readOnlyHint", tool.Annotations)
		}
	}
}

// textOf は結果の最初のテキスト内容を返す。
func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("no content")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] = %T, want *mcp.TextContent", res.Content[0])
	}
	return text.Text
}

func TestServerDoesNotAdvertiseLogging(t *testing.T) {
	cs := connect(t, t.TempDir())

	// logging は非推奨のフィールドなので、JSON の形で広告の有無を確かめる。
	b, err := json.Marshal(cs.InitializeResult().Capabilities)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"logging"`) {
		t.Errorf("capabilities = %s, want logging not advertised", b)
	}
}
