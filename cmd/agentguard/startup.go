package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/audit"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/filesystem"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/guard"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy"
)

// errPolicyInRoot は、ポリシーの実体パスがルートの内側にあるときの起動拒否。
// エージェントがポリシーを書き換えて次回の起動で緩める経路を断つ（DESIGN §9.1）。
var errPolicyInRoot = errors.New("the policy file must be outside the root")

// startup は起動の順に検査する。ポリシーを読んで検証し、ルートを開き、
// ポリシーの実体パスがルートの外にあることを確かめる。すべて通ったら guard を返す。
// 途中で失敗したら、開いたものはすべて閉じる。
func startup(rootDir, policyFile string) (*guard.Guard, error) {
	p, err := policy.LoadFile(policyFile)
	if err != nil {
		return nil, err
	}
	root, err := filesystem.Open(rootDir)
	if err != nil {
		return nil, err
	}
	if inside, err := policyInRoot(rootDir, policyFile); err != nil {
		_ = root.Close()
		return nil, err
	} else if inside {
		_ = root.Close()
		return nil, errPolicyInRoot
	}
	return guard.New(policy.NewEngine(p), root, p.Limits(), audit.NewStderr()), nil
}

// policyInRoot は、symlink を解決した後のポリシーの実体パスがルートの内側にあるかを返す。
// ルート自身も実体パスで比べ、ルートの外からの symlink でも内側の実体を指すものは拒否する。
func policyInRoot(rootDir, policyFile string) (bool, error) {
	realRoot, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		return false, fmt.Errorf("resolve root: %w", err)
	}
	realPolicy, err := filepath.EvalSymlinks(policyFile)
	if err != nil {
		return false, fmt.Errorf("resolve policy: %w", err)
	}
	rel, err := filepath.Rel(realRoot, realPolicy)
	if err != nil {
		return false, nil
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}
