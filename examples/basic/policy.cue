// examples/basic/project 用のサンプルポリシー。
// ポリシーファイルはルートの外に置く必要があるため、project/ の外にある。
package config

version: 1

limits: {
	maxReadBytes:  1048576
	maxWriteBytes: 1048576
}

_allActions: ["read", "write", "delete", "list"]

_hidden: {
	// どの階層の .env と .env.* も拒否する。
	"env-deny": path: "**/.env*"

	// .git/ の hooks や config の書き換えはコード実行につながるため拒否する。
	"git-deny": {
		path:      ".git"
		recursive: true
	}
}

rules: {
	"root-list": {
		path: "."
		actions: ["list"]
		effect: "allow"
	}

	// src/ は読み書きできる。
	"source-read": {
		path: "src"
		actions: ["read", "list"]
		effect:    "allow"
		recursive: true
	}

	"source-write": {
		path: "src"
		actions: ["write"]
		effect:    "allow"
		recursive: true
	}

	// docs/ は読み取りのみ。
	"docs-read": {
		path: "docs"
		actions: ["read", "list"]
		effect:    "allow"
		recursive: true
	}

	"go-mod-read": {
		path: "go.mod"
		actions: ["read", "list"]
		effect: "allow"
	}

	// 隠すパスは _hidden にまとめ、すべての操作を拒否する。
	for name, h in _hidden {
		(name): h & {
			actions: _allActions
			effect:  "deny"
		}
	}

	"go-mod-deny": {
		path: "go.mod"
		actions: ["write", "delete"]
		effect: "deny"
	}
}

// 期待する判定。ポリシーを読み込むときに実行され、食い違うと起動しない。
tests: [
	{path: "src/main.go", action: "read", want: "allow"},
	{path: "src/main.go", action: "write", want: "allow"},
	{path: "src/.env", action: "read", want: "deny"},
	{path: "src/.ENV", action: "read", want: "deny"},
	{path: ".git/config", action: "write", want: "deny"},
	{path: "docs/README.md", action: "read", want: "allow"},
	{path: "docs/README.md", action: "write", want: "deny"},
	{path: "go.mod", action: "read", want: "allow"},
	{path: "go.mod", action: "delete", want: "deny"},
	{path: "other/file.txt", action: "read", want: "deny"},
]
