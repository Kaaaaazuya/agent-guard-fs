// agentguard コマンドは、CUE の allow/deny ポリシーで制御したローカルファイル操作を
// 提供する MCP サーバーである。標準入出力は MCP プロトコル専用で、ログは標準エラー出力に出す。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/mcpserver"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run は main とテストから呼ぶエントリーポイント。起動に失敗したら終了コードを 1 にし、
// 引数が不正なら 2 を返す。
func run(args []string) int {
	fset := flag.NewFlagSet("agentguard", flag.ContinueOnError)
	fset.SetOutput(os.Stderr)
	rootDir := fset.String("root", "", "the only directory the server may access (required)")
	policyFile := fset.String("policy", "", "the CUE policy file, which must be outside the root (required)")
	if err := fset.Parse(args); err != nil {
		return 2
	}
	if *rootDir == "" || *policyFile == "" {
		fmt.Fprintln(os.Stderr, "usage: agentguard --root <dir> --policy <file>")
		return 2
	}

	g, err := startup(*rootDir, *policyFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "agentguard:", err)
		return 1
	}
	defer func() { _ = g.Close() }()

	// 終了シグナルを受けたら、進行中の操作にキャンセルを伝えて止める。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcpserver.New(g).Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, "agentguard:", err)
		return 1
	}
	return 0
}
