// guard パッケージは、すべての操作を次の順で実行する（DESIGN §5）。
// パス検証、ポリシー判定、ファイル操作、監査イベント。ポリシーで拒否された操作は
// ファイルシステムに触れないので、応答から対象の存在有無は分からない。
package guard

import (
	"context"
	"time"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/audit"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/filesystem"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy"
)

// Guard はポリシーとルートを組み合わせて操作を実行する。起動後は変更しない。
type Guard struct {
	engine *policy.Engine
	root   *filesystem.Root
	limits policy.Limits
	audit  *audit.Logger
}

// New は判定に engine、操作に root、上限に limits、監査の出力に log を使う Guard を返す。
func New(engine *policy.Engine, root *filesystem.Root, limits policy.Limits, log *audit.Logger) *Guard {
	return &Guard{engine: engine, root: root, limits: limits, audit: log}
}

// ReadResult は read の成功結果。
type ReadResult struct {
	Path    string
	Content []byte
}

// WriteResult は write の成功結果。
type WriteResult struct {
	Path    string
	Bytes   int
	Created bool
}

// Read は path のファイル内容を読む。
func (g *Guard) Read(ctx context.Context, path string) (res ReadResult, err error) {
	a := g.begin(policy.Read)
	defer func() { g.finish(a, err) }()

	rel, err := g.authorize(ctx, a, path)
	if err != nil {
		return ReadResult{}, err
	}
	content, err := g.root.Read(rel, int(g.limits.MaxReadBytes))
	if err != nil {
		return ReadResult{}, err
	}
	return ReadResult{Path: rel.String(), Content: content}, nil
}

// Write は path の内容を content で置き換え、無ければ作成する。
func (g *Guard) Write(ctx context.Context, path string, content []byte) (res WriteResult, err error) {
	a := g.begin(policy.Write)
	defer func() { g.finish(a, err) }()

	rel, err := g.authorize(ctx, a, path)
	if err != nil {
		return WriteResult{}, err
	}
	created, err := g.root.Write(ctx, rel, content, int(g.limits.MaxWriteBytes))
	if err != nil {
		return WriteResult{}, err
	}
	return WriteResult{Path: rel.String(), Bytes: len(content), Created: created}, nil
}

// Delete は path の通常ファイルを削除する。
func (g *Guard) Delete(ctx context.Context, path string) (deleted string, err error) {
	a := g.begin(policy.Delete)
	defer func() { g.finish(a, err) }()

	rel, err := g.authorize(ctx, a, path)
	if err != nil {
		return "", err
	}
	if err := g.root.Delete(rel); err != nil {
		return "", err
	}
	return rel.String(), nil
}

// authorize は path を検証し、action を許可してよいかを判定する。拒否されたら、
// ファイルシステムに触れる前に policy_denied を返す。応答の文言は対象の存在有無によらず同じにする。
// 判定の結果は a に記録する。
func (g *Guard) authorize(ctx context.Context, a *attempt, path string) (pathutil.RelPath, error) {
	if err := ctx.Err(); err != nil {
		return pathutil.RelPath{}, err
	}
	rel, err := pathutil.Parse(path)
	if err != nil {
		return pathutil.RelPath{}, err
	}
	a.path = rel.String()

	// 一時ファイルの名前はポリシーにかかわらず拒否する。名前だけで決まるので存在有無は漏れない。
	if filesystem.IsTempName(rel) {
		a.decision = audit.Deny
		return pathutil.RelPath{}, errs.New(errs.PolicyDenied, "the operation is not allowed")
	}
	d := g.engine.Evaluate(a.action, rel)
	a.rule = d.Rule
	if !d.Allowed {
		a.decision = audit.Deny
		denied := errs.New(errs.PolicyDenied, "the operation is not allowed")
		denied.Rule = d.Rule
		return pathutil.RelPath{}, denied
	}
	a.decision = audit.Allow
	return rel, nil
}

// attempt は 1 回の操作の監査イベントを組み立てる途中の状態。
type attempt struct {
	start    time.Time
	action   policy.Action
	path     string
	decision string
	rule     string
}

// begin は action の操作を始める。判定の前に終わった操作は、判定を未評価として記録する。
func (g *Guard) begin(action policy.Action) *attempt {
	return &attempt{start: time.Now(), action: action, decision: audit.NotEvaluated}
}

// finish は操作の結果 err を監査イベントとして記録する。
func (g *Guard) finish(a *attempt, err error) {
	result := "ok"
	if err != nil {
		result = string(errs.CodeOf(err))
	}
	g.audit.Log(audit.Event{
		Action:   string(a.action),
		Path:     a.path,
		Decision: a.decision,
		Rule:     a.rule,
		Result:   result,
		Duration: time.Since(a.start),
	})
}
