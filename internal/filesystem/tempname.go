package filesystem

import (
	"path"
	"strings"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// tempPrefix は書き込み途中の一時ファイルの名前の接頭辞（DESIGN §9.3）。
const tempPrefix = ".agentguard-tmp-"

// IsTempName は rel の最後の要素が一時ファイルの名前かどうかを返す。この名前のパスは、
// ポリシーにかかわらず全操作で拒否し、list にも出さない。
func IsTempName(rel pathutil.RelPath) bool {
	return strings.HasPrefix(path.Base(rel.String()), tempPrefix)
}
