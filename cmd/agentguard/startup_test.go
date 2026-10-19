//go:build linux || darwin

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy"
)

func TestStartupAcceptsPolicyOutsideRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	policy := writePolicy(t, base, "policy.cue")

	g, err := startup(root, policy)
	if err != nil {
		t.Fatalf("startup() error = %v", err)
	}
	if err := g.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestStartupRefusesPolicyInsideRoot(t *testing.T) {
	root := t.TempDir()
	policy := writePolicy(t, root, "policy.cue")

	if _, err := startup(root, policy); !errors.Is(err, errPolicyInRoot) {
		t.Errorf("startup() error = %v, want errPolicyInRoot", err)
	}
}

func TestStartupRefusesOutsideSymlinkToPolicyInsideRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	inside := writePolicy(t, root, "policy.cue")
	// ルートの外にある名前が、ルートの内側の実体を指している。実体で判定して拒否する。
	link := filepath.Join(base, "link.cue")
	if err := os.Symlink(inside, link); err != nil {
		t.Fatal(err)
	}

	if _, err := startup(root, link); !errors.Is(err, errPolicyInRoot) {
		t.Errorf("startup() error = %v, want errPolicyInRoot", err)
	}
}

func TestStartupAcceptsInsideSymlinkToPolicyOutsideRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	outside := writePolicy(t, base, "policy.cue")
	// ルートの内側の名前が、外側のポリシーを指している。実体はルートの外なので起動できる。
	link := filepath.Join(root, "policy.cue")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	g, err := startup(root, link)
	if err != nil {
		t.Fatalf("startup() error = %v", err)
	}
	_ = g.Close()
}

func TestWarnShadowed(t *testing.T) {
	const src = `version: 1
limits: {maxReadBytes: 1, maxWriteBytes: 1}
rules: {
	"env-read": {path: ".env", actions: ["read"], effect: "allow"}
	"env-deny": {path: "**/.env*", actions: ["read"], effect: "deny"}
	"src-read": {path: "src", actions: ["read"], effect: "allow", recursive: true}
}
`
	p, err := policy.Load("policy.cue", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	warnShadowed(&out, policy.NewEngine(p))
	want := `agentguard: warning: allow rule "env-read" never takes effect: everything it allows is denied by "env-deny"` + "\n"
	if out.String() != want {
		t.Errorf("warnShadowed wrote %q, want %q", out.String(), want)
	}
}

func TestWarnShadowedIsQuietForTheSample(t *testing.T) {
	p, err := policy.LoadFile(samplePolicy)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	warnShadowed(&out, policy.NewEngine(p))
	if out.Len() != 0 {
		t.Errorf("warnShadowed wrote %q for the sample policy", out.String())
	}
}
