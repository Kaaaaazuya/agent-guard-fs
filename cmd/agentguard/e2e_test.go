//go:build linux || darwin

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestStdioProtocolStaysClean は、ビルドしたバイナリを実際に STDIO で動かし、
// 標準出力に MCP のメッセージだけが出ることと、ログが標準エラー出力に出ることを確かめる。
func TestStdioProtocolStaysClean(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary; skipped in -short mode")
	}
	bin := filepath.Join(t.TempDir(), "agentguard")
	//nolint:gosec // builds this package with fixed arguments
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	policy := writePolicy(t, base, "policy.cue")

	// 実際のクライアントと同じく、応答を読んでから次の要求を送り、最後まで stdin を開けておく。
	cmd := exec.Command(bin, "--root", root, "--policy", policy) //nolint:gosec // runs the binary this test just built
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(stdout)
	exchange := func(request string, id float64) {
		t.Helper()
		if _, err := io.WriteString(stdin, request+"\n"); err != nil {
			t.Fatalf("write request: %v", err)
		}
		if id == 0 {
			return
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read response for id %v: %v\nstderr: %s", id, err, stderr.String())
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("stdout line is not JSON: %v: %q", err, line)
		}
		if msg["jsonrpc"] != "2.0" || msg["id"] != id {
			t.Errorf("response = %q, want a JSON-RPC response with id %v", line, id)
		}
		if id == 3 {
			result := msg["result"].(map[string]any)
			if result["isError"] == true {
				t.Errorf("read_file result is an error: %v", result)
			}
		}
	}

	exchange(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`, 1)
	exchange(`{"jsonrpc":"2.0","method":"notifications/initialized"}`, 0)
	exchange(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, 2)
	exchange(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"a.txt"}}}`, 3)

	// 標準出力に残りの行（MCP 以外の出力）がないことを確かめる。stdin を閉じれば終了する。
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	rest, _ := io.ReadAll(reader)
	if len(rest) != 0 {
		t.Errorf("unexpected stdout after the last response: %q", rest)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("agentguard exited with %v\nstderr: %s", err, stderr.String())
	}

	// 監査ログは標準エラー出力に、操作ごとに出ているはず。
	if !strings.Contains(stderr.String(), `"msg":"operation"`) || !strings.Contains(stderr.String(), `"action":"read"`) {
		t.Errorf("stderr = %q, want an audit event for read", stderr.String())
	}
}
