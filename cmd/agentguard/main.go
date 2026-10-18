// Command agentguard is an MCP server that exposes local file operations
// gated by a CUE allow/deny policy.
package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run is the entry point used by main and tests. Nothing is wired up yet,
// so it always refuses to start rather than serving without a policy.
func run(args []string) int {
	_ = args
	fmt.Fprintln(os.Stderr, "agentguard: not implemented yet")
	return 1
}
