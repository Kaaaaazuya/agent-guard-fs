package policy

import (
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// Decision は操作の判定結果。
type Decision struct {
	// Allowed は、操作を実行してよいかどうか。
	Allowed bool
	// Rule は判定を決めたルールの名前。既定拒否の場合は空。
	Rule string
	// Reason は判定理由を表す短い文。エラー応答と監査ログに使う。
	Reason string
}

// Engine はポリシーに基づいて操作を判定する。NewEngine で作り、以後は変更しない。
type Engine struct {
	deny  []compiledRule
	allow []compiledRule
}

// compiledRule は判定に使う形にしたルール。elems は、ルールの効果に応じて
// 正規化したパターン要素である。
type compiledRule struct {
	name      string
	actions   []Action
	elems     []string
	recursive bool
}

// NewEngine はポリシーから判定器を作る。deny ルールは大文字小文字と Unicode 正規化の
// 違いを無視して照合し、allow ルールは NFC に正規化したうえで大文字小文字を区別して照合する
// （DESIGN §6.3）。
func NewEngine(p *Policy) *Engine {
	e := &Engine{}
	for _, r := range p.rules {
		if r.Effect == Deny {
			e.deny = append(e.deny, compileRule(r, fold(r.Path.String())))
		} else {
			e.allow = append(e.allow, compileRule(r, nfc(r.Path.String())))
		}
	}
	return e
}

func compileRule(r Rule, normalized string) compiledRule {
	return compiledRule{
		name:      r.Name,
		actions:   r.Actions,
		elems:     splitPath(normalized),
		recursive: r.Recursive,
	}
}

// Evaluate は、action をパス path に対して行ってよいかを判定する。
// 一致する deny ルールがあれば拒否し、なければ一致する allow ルールで許可する。
// どちらにも一致しなければ拒否する（既定拒否）。ファイルシステムには触れない。
func (e *Engine) Evaluate(action Action, path pathutil.RelPath) Decision {
	if name, ok := firstMatch(e.deny, action, splitPath(fold(path.String()))); ok {
		return Decision{Rule: name, Reason: "deny rule matched"}
	}
	if name, ok := firstMatch(e.allow, action, splitPath(nfc(path.String()))); ok {
		return Decision{Allowed: true, Rule: name, Reason: "allow rule matched"}
	}
	return Decision{Reason: "no allow rule matched"}
}

// firstMatch は、action を含みパス path に一致する最初のルールの名前を返す。
// ルールは名前順に並んでいるので、結果は決定的になる。
func firstMatch(rules []compiledRule, action Action, path []string) (string, bool) {
	for _, r := range rules {
		if slices.Contains(r.actions, action) && matchRule(r.elems, path, r.recursive) {
			return r.name, true
		}
	}
	return "", false
}

// nfc は s を NFC に正規化する。
func nfc(s string) string { return norm.NFC.String(s) }

// fold は、大文字小文字と Unicode 正規化の違いを無視できる形にする。
func fold(s string) string { return cases.Fold().String(nfc(s)) }

// splitPath は正規化済みのパス文字列を要素列にする。ルート（"."）は空の列になる。
func splitPath(s string) []string {
	if s == "." {
		return nil
	}
	return strings.Split(s, "/")
}
