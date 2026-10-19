package policy

import (
	"strings"
	"testing"
)

const testsPolicyHead = `
version: 1
limits: {maxReadBytes: 1, maxWriteBytes: 1}
rules: {
	"src-read": {path: "src", actions: ["read"], effect: "allow", recursive: true}
	"env-deny": {path: "**/.env*", actions: ["read"], effect: "deny"}
}
`

func TestLoadRunsTests(t *testing.T) {
	src := testsPolicyHead + `tests: [
	{path: "src/main.go", action: "read", want: "allow"},
	{path: "src/.env", action: "read", want: "deny"},
	{path: "docs/a.md", action: "read", want: "deny"},
]`
	if _, err := Load("policy.cue", []byte(src)); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func TestLoadReportsEveryFailedTest(t *testing.T) {
	src := testsPolicyHead + `tests: [
	{path: "src/main.go", action: "read", want: "allow"},
	{path: "src/.env", action: "read", want: "allow"},
	{path: "docs/a.md", action: "read", want: "allow"},
	{path: "src/main.go", action: "read", want: "deny"},
]`
	_, err := Load("policy.cue", []byte(src))
	if err == nil {
		t.Fatal("Load succeeded for a policy whose tests fail")
	}
	for _, want := range []string{
		`tests[1]: read src/.env: want allow, got deny (rule "env-deny")`,
		`tests[2]: read docs/a.md: want allow, got deny (no allow rule matched)`,
		`tests[3]: read src/main.go: want deny, got allow (rule "src-read")`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not contain %q:\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), "tests[0]") {
		t.Errorf("error mentions a passing test:\n%v", err)
	}
}

func TestLoadRejectsBadTests(t *testing.T) {
	tests := []struct {
		name, tests string
	}{
		{"unknown action", `{path: "src", action: "exec", want: "allow"}`},
		{"unknown want", `{path: "src", action: "read", want: "maybe"}`},
		{"missing want", `{path: "src", action: "read"}`},
		{"unknown field", `{path: "src", action: "read", want: "allow", note: "x"}`},
		{"empty path", `{path: "", action: "read", want: "allow"}`},
		{"parent path", `{path: "../x", action: "read", want: "deny"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Load("policy.cue", []byte(testsPolicyHead+"tests: ["+tt.tests+"]")); err == nil {
				t.Error("Load succeeded")
			}
		})
	}
}
