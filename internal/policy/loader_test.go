package policy

import (
	"strings"
	"testing"
)

const validPolicy = `
package config

version: 1
limits: {
	maxReadBytes:  1048576
	maxWriteBytes: 1048576
}
rules: {
	"src-rw": {
		path: "src"
		actions: ["read", "write", "list"]
		effect: "allow"
		recursive: true
	}
	"env-deny": {
		path: "**/.env*"
		actions: ["read", "write", "delete", "list"]
		effect: "deny"
	}
}
`

func TestLoadValid(t *testing.T) {
	p, err := Load("policy.cue", []byte(validPolicy))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := p.Limits(); got != (Limits{MaxReadBytes: 1048576, MaxWriteBytes: 1048576, MaxListEntries: 1000}) {
		t.Errorf("Limits() = %+v", got)
	}
	if p.RuleCount() != 2 {
		t.Fatalf("RuleCount() = %d, want 2", p.RuleCount())
	}
	// 名前順に並ぶ。
	env, src := p.rules[0], p.rules[1]
	if env.Name != "env-deny" || src.Name != "src-rw" {
		t.Fatalf("rule order = %q, %q", env.Name, src.Name)
	}
	if env.Effect != Deny || env.Recursive {
		t.Errorf("env-deny = %+v, want deny with recursive defaulting to false", env)
	}
	if src.Effect != Allow || !src.Recursive || len(src.Actions) != 3 {
		t.Errorf("src-rw = %+v", src)
	}
}

func TestLoadRejects(t *testing.T) {
	const limits = `limits: {maxReadBytes: 1, maxWriteBytes: 1}`
	const rule = `path: "src", actions: ["read"], effect: "allow"`
	tests := []struct {
		name, src string
	}{
		{"syntax error", `version: `},
		{"missing version", `rules: {}, ` + limits},
		{"wrong version", `version: 2, rules: {}, ` + limits},
		{"missing rules", `version: 1, ` + limits},
		{"missing limits", `version: 1, rules: {}`},
		{"unknown top-level field", `version: 1, rules: {}, extra: true, ` + limits},
		{"root field", `version: 1, rules: {}, root: "/", ` + limits},
		{"unknown rule field", `version: 1, rules: r: {` + rule + `, mode: "x"}, ` + limits},
		{"unknown limits field", `version: 1, rules: {}, limits: {maxReadBytes: 1, maxWriteBytes: 1, x: 1}`},
		{"missing rule path", `version: 1, rules: r: {actions: ["read"], effect: "allow"}, ` + limits},
		{"invalid action", `version: 1, rules: r: {path: "src", actions: ["exec"], effect: "allow"}, ` + limits},
		{"invalid effect", `version: 1, rules: r: {path: "src", actions: ["read"], effect: "maybe"}, ` + limits},
		{"empty actions", `version: 1, rules: r: {path: "src", actions: [], effect: "allow"}, ` + limits},
		{"duplicate actions", `version: 1, rules: r: {path: "src", actions: ["read", "read"], effect: "allow"}, ` + limits},
		{"empty path", `version: 1, rules: r: {path: "", actions: ["read"], effect: "allow"}, ` + limits},
		{"absolute path", `version: 1, rules: r: {path: "/etc", actions: ["read"], effect: "allow"}, ` + limits},
		{"backslash path", `version: 1, rules: r: {path: "a\\b", actions: ["read"], effect: "allow"}, ` + limits},
		{"colon path", `version: 1, rules: r: {path: "a:b", actions: ["read"], effect: "allow"}, ` + limits},
		{"NUL path", `version: 1, rules: r: {path: "a\u0000b", actions: ["read"], effect: "allow"}, ` + limits},
		{"zero read limit", `version: 1, rules: {}, limits: {maxReadBytes: 0, maxWriteBytes: 1}`},
		{"read limit too large", `version: 1, rules: {}, limits: {maxReadBytes: 67108865, maxWriteBytes: 1}`},
		{"negative write limit", `version: 1, rules: {}, limits: {maxReadBytes: 1, maxWriteBytes: -1}`},
		{"list limit too large", `version: 1, rules: {}, limits: {maxReadBytes: 1, maxWriteBytes: 1, maxListEntries: 100001}`},
		{"non-concrete value", `version: 1, rules: r: {path: string, actions: ["read"], effect: "allow"}, ` + limits},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load("policy.cue", []byte(tt.src))
			if err == nil {
				t.Fatal("Load succeeded, want error")
			}
			if !strings.Contains(err.Error(), "invalid policy") {
				t.Errorf("error = %v, want an invalid policy error", err)
			}
		})
	}
}

func TestLoadErrorShowsPosition(t *testing.T) {
	_, err := Load("my-policy.cue", []byte("version: 1\nrules: {}\nextra: 1\nlimits: {maxReadBytes: 1, maxWriteBytes: 1}\n"))
	if err == nil || !strings.Contains(err.Error(), "my-policy.cue:3") {
		t.Errorf("error = %v, want it to point at my-policy.cue:3", err)
	}
}
