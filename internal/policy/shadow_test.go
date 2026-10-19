package policy

import (
	"slices"
	"testing"
)

func TestShadowed(t *testing.T) {
	tests := []struct {
		name  string
		rules string
		want  []Shadowed
	}{
		{
			name: "allow on exactly the denied path",
			rules: `"a": {path: ".env", actions: ["read"], effect: "allow"}
"d": {path: "**/.env*", actions: ["read", "write"], effect: "deny"}`,
			want: []Shadowed{{Rule: "a", By: []string{"d"}}},
		},
		{
			name: "deny ignores case, so it covers a differently spelled allow",
			rules: `"a": {path: "Secret.TXT", actions: ["read"], effect: "allow"}
"d": {path: "secret.txt", actions: ["read"], effect: "deny"}`,
			want: []Shadowed{{Rule: "a", By: []string{"d"}}},
		},
		{
			name: "actions split across two deny rules",
			rules: `"a": {path: "x", actions: ["read", "write"], effect: "allow"}
"d1": {path: "x", actions: ["read"], effect: "deny"}
"d2": {path: "x", actions: ["write"], effect: "deny"}`,
			want: []Shadowed{{Rule: "a", By: []string{"d1", "d2"}}},
		},
		{
			name: "recursive deny covers a recursive allow below it",
			rules: `"a": {path: ".git/hooks", actions: ["read"], effect: "allow", recursive: true}
"d": {path: ".git", actions: ["read"], effect: "deny", recursive: true}`,
			want: []Shadowed{{Rule: "a", By: []string{"d"}}},
		},
		{
			name: "deny ending in ** reaches below like recursive",
			rules: `"a": {path: "tmp/x", actions: ["read"], effect: "allow", recursive: true}
"d": {path: "tmp/**", actions: ["read"], effect: "deny"}`,
			want: []Shadowed{{Rule: "a", By: []string{"d"}}},
		},
		{
			name: "one action is not denied",
			rules: `"a": {path: "x", actions: ["read", "write"], effect: "allow"}
"d": {path: "x", actions: ["read"], effect: "deny"}`,
		},
		{
			name: "non-recursive deny does not cover a recursive allow",
			rules: `"a": {path: "x", actions: ["read"], effect: "allow", recursive: true}
"d": {path: "x", actions: ["read"], effect: "deny"}`,
		},
		{
			name: "deny covers only part of a glob allow, which is not checked",
			rules: `"a": {path: "src/*", actions: ["read"], effect: "allow"}
"d": {path: "src/*", actions: ["read"], effect: "deny"}`,
		},
		{
			name: "root allow stays reachable under a narrower deny",
			rules: `"a": {path: ".", actions: ["list"], effect: "allow"}
"d": {path: ".git", actions: ["list"], effect: "deny", recursive: true}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engineFor(t, tt.rules).Shadowed()
			if !slices.EqualFunc(got, tt.want, func(a, b Shadowed) bool {
				return a.Rule == b.Rule && slices.Equal(a.By, b.By)
			}) {
				t.Errorf("Shadowed() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestShadowedMessage(t *testing.T) {
	s := Shadowed{Rule: "a", By: []string{"d1", "d2"}}
	want := `allow rule "a" never takes effect: everything it allows is denied by "d1", "d2"`
	if got := s.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestExamplePolicyHasNoShadowedRule(t *testing.T) {
	p, err := LoadFile("../../examples/basic/policy.cue")
	if err != nil {
		t.Fatal(err)
	}
	if got := NewEngine(p).Shadowed(); len(got) != 0 {
		t.Errorf("Shadowed() = %+v, want none", got)
	}
}
