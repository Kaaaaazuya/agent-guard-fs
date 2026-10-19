package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/policy"
)

const explainUsage = "usage: agentguard explain --policy <file> <read|write|delete|list> <path>"

// runExplain は、ポリシーが action と path をどう判定し、どのルールがそれを決めたかを stdout に出す。
// ルートには触れないので、ポリシーを書き換えながら繰り返し確かめられる。
// 判定が拒否でも終了コードは 0 で、読み込みや引数の誤りだけが 1 と 2 になる。
func runExplain(args []string, stdout io.Writer) int {
	fset := flag.NewFlagSet("agentguard explain", flag.ContinueOnError)
	fset.SetOutput(os.Stderr)
	policyFile := fset.String("policy", "", "the CUE policy file (required)")
	if err := fset.Parse(args); err != nil {
		return 2
	}
	if *policyFile == "" || fset.NArg() != 2 {
		fmt.Fprintln(os.Stderr, explainUsage)
		return 2
	}
	action, ok := parseAction(fset.Arg(0))
	if !ok {
		fmt.Fprintf(os.Stderr, "agentguard: unknown action %q\n%s\n", fset.Arg(0), explainUsage)
		return 2
	}
	path, err := pathutil.Parse(fset.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "agentguard:", err)
		return 2
	}

	p, err := policy.LoadFile(*policyFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "agentguard:", err)
		return 1
	}
	d := policy.NewEngine(p).Evaluate(action, path)

	verdict, rule := "deny", d.Rule
	if d.Allowed {
		verdict = "allow"
	}
	if rule == "" {
		rule = "(none)"
	}
	_, _ = fmt.Fprintf(stdout, "%s\naction: %s\npath:   %s\nrule:   %s\nreason: %s\n", verdict, action, path, rule, d.Reason)
	return 0
}

func parseAction(s string) (policy.Action, bool) {
	switch a := policy.Action(s); a {
	case policy.Read, policy.Write, policy.Delete, policy.List:
		return a, true
	}
	return "", false
}
