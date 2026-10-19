//go:build linux || darwin

package guard

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// events は、監査ログの各行を JSON として読む。
func events(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("log line is not JSON: %v: %q", err, line)
		}
		out = append(out, e)
	}
	return out
}

func TestAuditRecordsDeniedOperationWithRule(t *testing.T) {
	var buf bytes.Buffer
	g, dir := newGuardWithLog(t, srcAndEnvRules, 1000, &buf)
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _ = g.Read(context.Background(), ".env")

	got := events(t, &buf)
	if len(got) != 1 {
		t.Fatalf("events = %d, want 1: %q", len(got), buf.String())
	}
	e := got[0]
	want := map[string]any{
		"action":   "read",
		"path":     ".env",
		"decision": "deny",
		"rule":     "env",
		"result":   "policy_denied",
	}
	for k, v := range want {
		if e[k] != v {
			t.Errorf("%s = %v, want %v", k, e[k], v)
		}
	}
	if _, ok := e["duration_ms"]; !ok {
		t.Error("duration_ms is missing")
	}
}

func TestAuditDoesNotRecordFileContent(t *testing.T) {
	var buf bytes.Buffer
	g, dir := newGuardWithLog(t, srcAndEnvRules, 1000, &buf)
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	const payload = "body-marker-do-not-log"

	if _, err := g.Write(context.Background(), "src/a.txt", []byte(payload)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := g.Read(context.Background(), "src/a.txt"); err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if strings.Contains(buf.String(), payload) {
		t.Errorf("audit log contains file content: %q", buf.String())
	}
	if n := len(events(t, &buf)); n != 2 {
		t.Errorf("events = %d, want 2", n)
	}
}

func TestAuditRecordsRejectedInputWithoutRawPath(t *testing.T) {
	var buf bytes.Buffer
	g, _ := newGuardWithLog(t, srcAndEnvRules, 1000, &buf)

	// 制御文字を含む入力は検証で止まる。ログには生の入力を出さない。
	_, _ = g.Read(context.Background(), "bad\x07name")

	got := events(t, &buf)
	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
	if got[0]["path"] != "" || got[0]["decision"] != "not_evaluated" || got[0]["result"] != "invalid_argument" {
		t.Errorf("event = %v", got[0])
	}
	if strings.Contains(buf.String(), "\x07") {
		t.Errorf("raw control character reached the log: %q", buf.String())
	}
}

func TestAuditRecordsDefaultDenyWithoutRule(t *testing.T) {
	var buf bytes.Buffer
	g, dir := newGuardWithLog(t, srcAndEnvRules, 1000, &buf)
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _ = g.Read(context.Background(), "notes.txt")

	e := events(t, &buf)[0]
	if e["decision"] != "deny" || e["rule"] != "" {
		t.Errorf("event = %v, want deny with empty rule", e)
	}
}
