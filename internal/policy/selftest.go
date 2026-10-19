package policy

import (
	"fmt"
	"strings"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy/schema"
)

// runTests は、tests を判定器で実行する。期待と食い違うものがあれば、全件をまとめたエラーを返す。
// 期待が外れるポリシーで起動させないための検査で、判定そのものには影響しない。
func runTests(e *Engine, tests []schema.Test) error {
	var failures []string
	for i, tc := range tests {
		path, err := pathutil.Parse(tc.Path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("tests[%d]: %v", i, err))
			continue
		}
		d := e.Evaluate(tc.Action, path)
		if d.Allowed == (tc.Want == Allow) {
			continue
		}
		failures = append(failures, fmt.Sprintf("tests[%d]: %s %s: want %s, got %s", i, tc.Action, path, tc.Want, describe(d)))
	}
	if len(failures) > 0 {
		return fmt.Errorf("policy tests failed:\n%s", strings.Join(failures, "\n"))
	}
	return nil
}

// describe は判定を、効果と決めたルールの名前で表す。
func describe(d Decision) string {
	effect := Deny
	if d.Allowed {
		effect = Allow
	}
	if d.Rule == "" {
		return fmt.Sprintf("%s (%s)", effect, d.Reason)
	}
	return fmt.Sprintf("%s (rule %q)", effect, d.Rule)
}
