# 使い方ガイド

[English](usage.md) | 日本語

このガイドでは、現在の `main` ブランチの AgentGuard FS でできることを説明します。
プロジェクトは開発初期で、ポリシーの書式と検証は動きますが、MCP サーバー本体はまだ動作しません。
機能が入り次第、このガイドに節を追加します。

## 現在できること

| 項目 | 状態 |
| --- | --- |
| `agentguard` バイナリのビルド | 動作する |
| ポリシーファイルの書式と検証 | 動作する（Go のテストから使用。コマンドでの検査はまだない） |
| ルールパスの構文と正規化 | 動作する |
| MCP サーバーとファイル操作ツール | 未実装。`agentguard` は `agentguard: not implemented yet` を出力し、終了ステータス 1 で終わる |

## 1. 準備とビルド

必要なもの:

- Linux 5.6 以上、または macOS 11 以上
- Go 1.26 以上
- [Task](https://taskfile.dev/) v3
- [golangci-lint](https://golangci-lint.run/) v2（`task lint` を使う場合のみ）

```sh
git clone https://github.com/Kaaaaazuya/agent-guard-fs.git
cd agent-guard-fs

# 任意ですが推奨: CI と同じく Takumi Guard 経由でモジュールを取得する
export GOPROXY=https://golang.flatt.tech   # ",direct" は付けない

task build          # bin/agentguard を作る
./bin/agentguard    # "agentguard: not implemented yet" を出力し、終了ステータス 1
```

## 2. チェックを実行する

| コマンド | 内容 |
| --- | --- |
| `task` | `fmt-check`、`vet`、`lint`、`test`、`build` を順に実行 |
| `task fmt` | `gofmt -w` で Go コードを整形 |
| `task fmt-check` | `gofmt` が必要なファイルがあれば失敗 |
| `task vet` | `go vet ./...` を実行 |
| `task lint` | `golangci-lint run` を実行 |
| `task test` | `go test -race ./...` を実行 |
| `task vuln` | `govulncheck` を実行 |
| `task build` | `bin/agentguard` をビルド |
| `task clean` | `bin/` を削除 |

同じ一覧は `task --list` でも確認できます。

## 3. ポリシーを書く

ポリシーは 1 つの CUE ファイルです。
[`examples/basic/policy.cue`](../examples/basic/policy.cue) をもとに書き始めてください。

```cue
package config

version: 1

limits: {
	maxReadBytes:  1048576
	maxWriteBytes: 1048576
}

rules: {
	"source-read": {
		path: "src"
		actions: ["read", "list"]
		effect:    "allow"
		recursive: true
	}

	"env-deny": {
		path: "**/.env*"
		actions: ["read", "write", "delete", "list"]
		effect: "deny"
	}
}
```

ファイルはバイナリに埋め込まれたスキーマで検証されるため、外部からスキーマを差し替えたり
緩めたりすることはできません。以下に挙げていないフィールドはエラーになります。

### トップレベルのフィールド

| フィールド | 必須 | 値 |
| --- | --- | --- |
| `version` | はい | `1` 固定 |
| `limits` | はい | [上限値](#上限値) を参照 |
| `rules` | はい | ルール名からルールへのマップ。名前は自由に付けられ、エラーメッセージに表示される |

### 上限値

| フィールド | 必須 | 範囲 | 既定値 |
| --- | --- | --- | --- |
| `maxReadBytes` | はい | 1〜67108864（64 MiB） | なし |
| `maxWriteBytes` | はい | 1〜67108864（64 MiB） | なし |
| `maxListEntries` | いいえ | 1〜100000 | 1000 |

### ルール

| フィールド | 必須 | 値 |
| --- | --- | --- |
| `path` | はい | ルートからの相対パス。[ルールパス](#ルールパス) を参照 |
| `actions` | はい | `"read"`、`"write"`、`"delete"`、`"list"` のうち 1 つ以上。重複は不可 |
| `effect` | はい | `"allow"` または `"deny"` |
| `recursive` | いいえ | `true` または `false`（既定値 `false`） |

### ルールパス

ルールパスはルートディレクトリからの相対パスで、区切りは `/` です。

- 書けるもの: `.`（ルート自身）、`src/main.go` のような通常の名前、グロブ `*`、`?`、`**`。
- `**` はパス要素全体としてのみ使えます。`**/.env` は可、`src/a**` は不可です。
- 拒否されるもの: 空のパス、先頭の `/`、`..` の要素（ルート内に収まる場合も含む）、`\`、`:`、
  NUL などの制御文字、不正な UTF-8、グロブ文字 `[`、`]`、`{`、`}`。
- `./`、連続した `/`、末尾の `/` は取り除かれるので、`./src//` は `src` と同じです。

### ポリシーファイルの置き場所

エージェントが自分のポリシーを書き換えられないよう、ポリシーファイルはエージェントが作業する
ディレクトリの外に置いてください。

### ポリシーが不正なとき

不正なポリシーは、ファイル名と行を示すエラーで拒否されます。たとえば、存在しない action を
書いた場合は次のようになります（抜粋）。

```text
invalid policy:
#Policy.rules.r.actions.0: conflicting values "read" and "exec":
    my-policy.cue:3:35
    ...
```

`..` でルートの外に出るパスの場合は次のとおりです。

```text
invalid policy: rule "r": invalid_argument: path must not contain ".."
```

ポリシーファイルを検査するコマンドはまだありません。サンプルポリシーは
`go test ./internal/policy` で読み込んでいるので、不正になれば `task test` が失敗します。
