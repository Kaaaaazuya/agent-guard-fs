package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLogWritesOneJSONLinePerEvent(t *testing.T) {
	var buf bytes.Buffer
	New(&buf).Log(Event{
		Action:   "write",
		Path:     "src/a.go",
		Decision: "deny",
		Rule:     "env-deny",
		Result:   "policy_denied",
		Duration: 1500 * time.Microsecond,
	})

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1: %q", len(lines), buf.String())
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("line is not JSON: %v", err)
	}
	want := map[string]any{
		"action":      "write",
		"path":        "src/a.go",
		"decision":    "deny",
		"rule":        "env-deny",
		"result":      "policy_denied",
		"duration_ms": 1.5,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	if _, ok := got["time"]; !ok {
		t.Error("time field is missing")
	}
}

func TestLogEscapesPathAsOneLine(t *testing.T) {
	// 検証済みの相対パスに改行は含まれないが、ログが複数行にならないことを確かめる。
	var buf bytes.Buffer
	New(&buf).Log(Event{Action: "read", Path: "a\nb", Decision: "allow", Result: "ok"})

	if n := strings.Count(buf.String(), "\n"); n != 1 {
		t.Errorf("newline count = %d, want 1: %q", n, buf.String())
	}
}

func TestNewStderrNeverWritesToStdout(t *testing.T) {
	origOut, origErr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = origOut, origErr }()

	NewStderr().Log(Event{Action: "list", Path: ".", Decision: "allow", Result: "ok"})

	_ = outW.Close()
	_ = errW.Close()
	stdout, _ := io.ReadAll(outR)
	stderr, _ := io.ReadAll(errR)
	if len(stdout) != 0 {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
	if !strings.Contains(string(stderr), `"action":"list"`) {
		t.Errorf("stderr = %q, want the event", stderr)
	}
}
