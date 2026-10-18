package guard

import (
	"context"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/filesystem"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy"
)

// ListResult は list の成功結果。Entries には list を許可されたエントリだけを入れ、
// 上限を超えた分は捨てて Truncated で伝える。
type ListResult struct {
	Path      string
	Entries   []filesystem.Entry
	Truncated bool
}

// List は path の直下のエントリを返す。path に list が許可されていなければ拒否する。
// 各エントリは、list を許可されている場合だけ返す。例えば deny された src/.env は含めない。
func (g *Guard) List(ctx context.Context, path string) (ListResult, error) {
	rel, err := g.authorize(ctx, policy.List, path)
	if err != nil {
		return ListResult{}, err
	}
	entries, err := g.root.List(rel)
	if err != nil {
		return ListResult{}, err
	}

	visible := make([]filesystem.Entry, 0, len(entries))
	for _, e := range entries {
		child, err := childPath(rel, e.Name)
		if err != nil {
			// エージェントが指定できない名前（":" や制御文字を含むなど）は、読むこともできないので隠す。
			continue
		}
		if g.engine.Evaluate(policy.List, child).Allowed {
			visible = append(visible, e)
		}
	}

	truncated := false
	if limit := g.limits.MaxListEntries; len(visible) > limit {
		visible = visible[:limit]
		truncated = true
	}
	return ListResult{Path: rel.String(), Entries: visible, Truncated: truncated}, nil
}

// childPath は dir の直下の name を、検証済みのパスとして返す。
func childPath(dir pathutil.RelPath, name string) (pathutil.RelPath, error) {
	if dir.IsRoot() {
		return pathutil.Parse(name)
	}
	return pathutil.Parse(dir.String() + "/" + name)
}
