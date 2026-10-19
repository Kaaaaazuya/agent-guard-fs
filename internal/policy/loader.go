// policy パッケージは、CUE で書かれたポリシーを読み込んで検証し、操作の許可・拒否を判定する。
package policy

import (
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy/schema"
)

//go:embed schema/policy.cue
var schemaSource []byte

// Action はポリシーで扱う操作の種別。スキーマから生成した型をそのまま使う。
type Action = schema.Action

const (
	Read   Action = "read"
	Write  Action = "write"
	Delete Action = "delete"
	List   Action = "list"
)

// Effect はルールが一致したときの効果。
type Effect = schema.Effect

const (
	Allow Effect = "allow"
	Deny  Effect = "deny"
)

// Rule は 1 つのルール。Name はポリシー内のマップのキー。
type Rule struct {
	Name      string
	Path      pathutil.Pattern
	Actions   []Action
	Effect    Effect
	Recursive bool
}

// Limits はサイズと件数の上限。値と既定値は CUE スキーマだけが持つ。
type Limits = schema.Limits

// Policy は検証済みのポリシー。読み込み後は変更しない。
type Policy struct {
	rules  []Rule
	limits Limits
}

// Limits はポリシーの上限値を返す。
func (p *Policy) Limits() Limits { return p.limits }

// RuleCount はルールの数を返す。起動時のログに使う。
func (p *Policy) RuleCount() int { return len(p.rules) }

// LoadFile は path のポリシーファイルを読み込んで検証する。
func LoadFile(path string) (*Policy, error) {
	src, err := os.ReadFile(path) // #nosec G304 -- パスは信頼できる起動引数
	if err != nil {
		return nil, fmt.Errorf("read policy: %w", err)
	}
	return Load(path, src)
}

// Load は src をスキーマの #Policy と unify し、具体値まで検証してから Go の構造体に変換する。
// filename はエラーメッセージの位置表示にだけ使う。
func Load(filename string, src []byte) (*Policy, error) {
	ctx := cuecontext.New()
	schemaValue := ctx.CompileBytes(schemaSource, cue.Filename("schema/policy.cue"))
	if err := schemaValue.Err(); err != nil {
		return nil, fmt.Errorf("compile embedded schema: %w", err)
	}
	user := ctx.CompileBytes(src, cue.Filename(filename))
	if err := user.Err(); err != nil {
		return nil, invalidPolicy(err)
	}
	v := schemaValue.LookupPath(cue.ParsePath("#Policy")).Unify(user)
	if err := v.Validate(cue.Concrete(true)); err != nil {
		return nil, invalidPolicy(err)
	}

	var raw schema.Policy
	if err := v.Decode(&raw); err != nil {
		return nil, invalidPolicy(err)
	}

	p := &Policy{limits: raw.Limits}
	for name, r := range raw.Rules {
		// スキーマでは表しきれない規則（".." やグロブ構文）はここで pathutil により検証する。
		path, err := pathutil.ParsePattern(r.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid policy: rule %q: %w", name, err)
		}
		p.rules = append(p.rules, Rule{
			Name:      name,
			Path:      path,
			Actions:   r.Actions,
			Effect:    r.Effect,
			Recursive: r.Recursive,
		})
	}
	// マップの反復順は不定なので、判定と出力を決定的にするため名前順に並べる。
	slices.SortFunc(p.rules, func(a, b Rule) int { return strings.Compare(a.Name, b.Name) })
	if err := runTests(NewEngine(p), raw.Tests); err != nil {
		return nil, err
	}
	return p, nil
}

// invalidPolicy は CUE のエラーを、ファイル上の位置付きで全件まとめたエラーにする。
func invalidPolicy(err error) error {
	return fmt.Errorf("invalid policy:\n%s", cueerrors.Details(err, nil))
}
