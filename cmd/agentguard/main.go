// agentguard コマンドは、CUE の allow/deny ポリシーで制御したローカルファイル操作を
// 提供する MCP サーバーである。
package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run は main とテストから呼ぶエントリーポイント。まだ何も組み込んでいないため、
// ポリシーなしで動作させず、常に起動を拒否する。
func run(args []string) int {
	_ = args
	fmt.Fprintln(os.Stderr, "agentguard: not implemented yet")
	return 1
}
