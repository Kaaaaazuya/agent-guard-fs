// guard パッケージは、すべての操作を次の順で実行する（DESIGN §5）。
// パス検証、ポリシー判定、ファイル操作、監査イベント。ポリシーで拒否された操作は
// ファイルシステムに触れないので、応答から対象の存在有無は分からない。
package guard

import (
	"context"

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
}

// New は判定に engine、操作に root、上限に limits を使う Guard を返す。
func New(engine *policy.Engine, root *filesystem.Root, limits policy.Limits) *Guard {
	return &Guard{engine: engine, root: root, limits: limits}
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
func (g *Guard) Read(ctx context.Context, path string) (ReadResult, error) {
	rel, err := g.authorize(ctx, policy.Read, path)
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
func (g *Guard) Write(ctx context.Context, path string, content []byte) (WriteResult, error) {
	rel, err := g.authorize(ctx, policy.Write, path)
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
func (g *Guard) Delete(ctx context.Context, path string) (string, error) {
	rel, err := g.authorize(ctx, policy.Delete, path)
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
func (g *Guard) authorize(ctx context.Context, action policy.Action, path string) (pathutil.RelPath, error) {
	if err := ctx.Err(); err != nil {
		return pathutil.RelPath{}, err
	}
	rel, err := pathutil.Parse(path)
	if err != nil {
		return pathutil.RelPath{}, err
	}
	// 一時ファイルの名前はポリシーにかかわらず拒否する。名前だけで決まるので存在有無は漏れない。
	if filesystem.IsTempName(rel) {
		return pathutil.RelPath{}, errs.New(errs.PolicyDenied, "the operation is not allowed")
	}
	d := g.engine.Evaluate(action, rel)
	if !d.Allowed {
		denied := errs.New(errs.PolicyDenied, "the operation is not allowed")
		denied.Rule = d.Rule
		return pathutil.RelPath{}, denied
	}
	return rel, nil
}
