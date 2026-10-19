// mcpserver パッケージは、guard の操作を MCP ツールとして公開する。引数の変換、結果の表現、
// エラーコードの MCP 表現だけを担う。判定と安全性の検査はすべて guard に任せる（DESIGN §5）。
package mcpserver

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/guard"
)

// Version は広告するサーバーのバージョン。
const Version = "0.1.0"

// Server は guard の操作を MCP ツールとして登録した MCP サーバー。
type Server struct {
	guard *guard.Guard
}

// New は g の操作を MCP ツールとして公開するサーバーを返す。
// ServerCapabilities は空にし、使わない logging を広告しない。
func New(g *guard.Guard) *mcp.Server {
	s := &Server{guard: g}
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "agentguard", Version: Version},
		&mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{}},
	)
	srv.AddTool(readFileTool(), s.readFile)
	srv.AddTool(writeFileTool(), s.writeFile)
	srv.AddTool(listFilesTool(), s.listFiles)
	return srv
}

// decode は呼び出しの引数を v に読む。未知のフィールドと型の違いは拒否する。
func decode(req *mcp.CallToolRequest, v any) error {
	dec := json.NewDecoder(bytes.NewReader(req.Params.Arguments))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// boolPtr は、ツールアノテーションのポインタ型のフラグを作る。
func boolPtr(b bool) *bool { return &b }

// invalidArguments は引数が読めなかったときの結果を返す。
func invalidArguments() *mcp.CallToolResult {
	return errorResult(errs.New(errs.InvalidArgument, "the arguments are invalid"))
}

// errorResult は err を isError の結果にする。構造化内容には、エラーコードと判定したルール名だけを入れる。
// 絶対パスや生の入力は含めない。
func errorResult(err error) *mcp.CallToolResult {
	var e *errs.Error
	if !errors.As(err, &e) {
		// 想定外のエラーは内部の詳細を出さず、安全側の内部エラーにする。
		return &mcp.CallToolResult{
			IsError:           true,
			Content:           []mcp.Content{&mcp.TextContent{Text: string(errs.Internal) + ": the operation failed"}},
			StructuredContent: map[string]string{"code": string(errs.Internal)},
		}
	}
	content := map[string]string{"code": string(e.Code)}
	if e.Rule != "" {
		content["rule"] = e.Rule
	}
	return &mcp.CallToolResult{
		IsError:           true,
		Content:           []mcp.Content{&mcp.TextContent{Text: e.Error()}},
		StructuredContent: content,
	}
}
