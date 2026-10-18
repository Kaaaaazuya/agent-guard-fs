package policy

import (
	"testing"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// engineFor は CUE のルール列からポリシーを読み込み、判定器を作る。
func engineFor(t *testing.T, rules string) *Engine {
	t.Helper()
	src := "version: 1\nrules: {\n" + rules + "\n}\nlimits: {maxReadBytes: 1, maxWriteBytes: 1}\n"
	p, err := Load("policy.cue", []byte(src))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return NewEngine(p)
}

func rel(t *testing.T, s string) pathutil.RelPath {
	t.Helper()
	p, err := pathutil.Parse(s)
	if err != nil {
		t.Fatalf("pathutil.Parse(%q): %v", s, err)
	}
	return p
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name    string
		rules   string
		action  Action
		path    string
		allowed bool
		rule    string
	}{
		{
			name:    "allow rule matches",
			rules:   `"src-rw": {path: "src", actions: ["read"], effect: "allow", recursive: true}`,
			action:  Read,
			path:    "src/main.go",
			allowed: true,
			rule:    "src-rw",
		},
		{
			name:   "default deny when nothing matches",
			rules:  `"src-rw": {path: "src", actions: ["read"], effect: "allow", recursive: true}`,
			action: Read,
			path:   "docs/a.md",
		},
		{
			name:   "action not listed in the rule",
			rules:  `"src-w": {path: "src", actions: ["write"], effect: "allow", recursive: true}`,
			action: Read,
			path:   "src/a.go",
		},
		{
			name:    "deny wins over an earlier-named allow",
			rules:   `"a-all": {path: ".", actions: ["read"], effect: "allow", recursive: true}, "z-env": {path: ".env", actions: ["read"], effect: "deny"}`,
			action:  Read,
			path:    ".env",
			allowed: false,
			rule:    "z-env",
		},
		{
			name:   "non-recursive rule does not cover children",
			rules:  `"src": {path: "src", actions: ["read"], effect: "allow"}`,
			action: Read,
			path:   "src/a.go",
		},
		{
			name:    "non-recursive rule covers its own path",
			rules:   `"src": {path: "src", actions: ["read"], effect: "allow"}`,
			action:  Read,
			path:    "src",
			allowed: true,
			rule:    "src",
		},
		{
			name:   "src does not cover src-old",
			rules:  `"src": {path: "src", actions: ["read"], effect: "allow", recursive: true}`,
			action: Read,
			path:   "src-old/a.go",
		},
		{
			name:    "root rule lists only the root when not recursive",
			rules:   `"root": {path: ".", actions: ["list"], effect: "allow"}`,
			action:  List,
			path:    ".",
			allowed: true,
			rule:    "root",
		},
		{
			name:   "root rule does not list a child",
			rules:  `"root": {path: ".", actions: ["list"], effect: "allow"}`,
			action: List,
			path:   "src",
		},
		{
			name:    "double star deny matches .env in a subdirectory",
			rules:   `"all": {path: ".", actions: ["read"], effect: "allow", recursive: true}, "env": {path: "**/.env*", actions: ["read"], effect: "deny"}`,
			action:  Read,
			path:    "src/a/.env.local",
			allowed: false,
			rule:    "env",
		},
		{
			name:    "double star deny does not match a plain name",
			rules:   `"all": {path: ".", actions: ["read"], effect: "allow", recursive: true}, "env": {path: "**/.env*", actions: ["read"], effect: "deny"}`,
			action:  Read,
			path:    "src/env",
			allowed: true,
			rule:    "all",
		},
		{
			name:    "deny ignores case",
			rules:   `"all": {path: ".", actions: ["read"], effect: "allow", recursive: true}, "env": {path: "**/.env*", actions: ["read"], effect: "deny"}`,
			action:  Read,
			path:    ".ENV",
			allowed: false,
			rule:    "env",
		},
		{
			name:    "deny ignores Unicode normalization",
			rules:   `"all": {path: ".", actions: ["read"], effect: "allow", recursive: true}, "cafe": {path: "café.txt", actions: ["read"], effect: "deny"}`,
			action:  Read,
			path:    "café.txt",
			allowed: false,
			rule:    "cafe",
		},
		{
			name:    "allow matches the NFC form of a name written in NFD",
			rules:   `"cafe": {path: "café.txt", actions: ["read"], effect: "allow"}`,
			action:  Read,
			path:    "café.txt",
			allowed: true,
			rule:    "cafe",
		},
		{
			name:   "allow does not ignore case",
			rules:  `"src": {path: "src", actions: ["read"], effect: "allow", recursive: true}`,
			action: Read,
			path:   "SRC/a.go",
		},
		{
			name:    "allow matches a glob",
			rules:   `"go": {path: "src/*.go", actions: ["read"], effect: "allow"}`,
			action:  Read,
			path:    "src/main.go",
			allowed: true,
			rule:    "go",
		},
		{
			name:   "glob does not cross elements",
			rules:  `"go": {path: "src/*.go", actions: ["read"], effect: "allow"}`,
			action: Read,
			path:   "src/a/main.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := engineFor(t, tt.rules)
			got := e.Evaluate(tt.action, rel(t, tt.path))
			if got.Allowed != tt.allowed || got.Rule != tt.rule {
				t.Errorf("Evaluate(%s, %q) = {Allowed: %v, Rule: %q}, want {Allowed: %v, Rule: %q}",
					tt.action, tt.path, got.Allowed, got.Rule, tt.allowed, tt.rule)
			}
			if got.Reason == "" {
				t.Error("Reason is empty")
			}
		})
	}
}
