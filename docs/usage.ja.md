# 使い方ガイド

[English](usage.md) | 日本語

このガイドでは、現在の `main` ブランチの AgentGuard FS でできることを説明します。
プロジェクトは開発初期です。ポリシー、ファイル操作、監査ログは Go パッケージとして動きますが、
エージェントが接続する MCP サーバーはまだつながっていません。機能が入り次第、このガイドに節を追加します。

## 現在できること

| 項目 | 状態 |
| --- | --- |
| `agentguard` バイナリのビルド | 動作する |
| ポリシーファイルの書式と検証 | 動作する |
| 操作ごとの許可・拒否の判定 | 動作する |
| ルート配下のファイルの読み取り・書き込み・一覧・削除 | Go パッケージ（`internal/guard`）内で動作する |
| 監査ログ（操作ごとに JSON 1 行） | Go パッケージ内で動作する |
| MCP サーバーと `agentguard` のコマンドライン | 未実装。`agentguard` は `agentguard: not implemented yet` を出力し、終了ステータス 1 で終わる |

## 全体の流れ

MCP サーバーが完成すると、AI エージェントはファイルに直接触れず、AgentGuard FS のツールを
呼び出すようになります。どの呼び出しも同じ 4 つの段階を通ります。右側の段階は `main` で
すでに動いており、左側の MCP 接続は予定です。

```text
 AI エージェント（MCP クライアント）    AgentGuard FS
 ─────────────────────               ─────────────────────────────────────────
  read_file   ".env"    ──（予定）───▶ 1. パスを検証        （".." や "/x" は不可）
  write_file  "src/a.go"              2. ポリシーに問い合わせ（拒否優先・デフォルト拒否）
  list_files  "."                     3. ファイルを操作      （--root の中だけ、symlink 不可）
  delete_file "tmp.txt" ◀──────────── 4. 監査ログを stderr に 1 行出力
```

予定しているコマンドラインでは、エージェントに作業させるプロジェクトのディレクトリと、
その外に置いたポリシーファイルを指定します。

```sh
# 予定: まだ動きません
agentguard --root ./my-project --policy ./my-policy.cue
```

## 実例: サンプルポリシーの動き

[`examples/basic/policy.cue`](../examples/basic/policy.cue) を小さなプロジェクトに適用した例です。
以下の結果はすべて `main` の guard 層で実際に得たものです。周りの MCP ツール名だけが予定です。

エージェントが作業するプロジェクト:

```text
my-project/
├── .env            # 秘密情報
├── .git/config
├── README.md
├── docs/guide.md
├── go.mod
└── src/
    ├── .env.local  # これも秘密情報
    └── main.go
```

エージェントの要求ごとの結果:

| エージェントの要求 | 結果 | 決めたルール |
| --- | --- | --- |
| `.` を一覧 | `docs`、`go.mod`、`src` | `root-list`（allow） |
| `src` を一覧 | `main.go` | `source-read`（allow） |
| `src/main.go` を読む | ファイルの内容 | `source-read`（allow） |
| `src/util.go` に書く | 新規作成 | `source-write`（allow） |
| `.env` を読む | `policy_denied` | `env-deny`（deny） |
| `go.mod` に書く | `policy_denied` | `go-mod-deny`（deny） |
| `README.md` を読む | `policy_denied` | allow ルールなし（デフォルト拒否） |
| `docs/guide.md` に書く | `policy_denied` | `write` の allow ルールなし（デフォルト拒否） |
| `src/util.go` を削除 | `policy_denied` | `delete` の allow ルールなし（デフォルト拒否） |
| `../etc/passwd` を読む | `invalid_argument` | ポリシーに問い合わせる前に拒否 |

注目してほしい点が 3 つあります。

- **拒否したものは見せない。** `.` の一覧には `.env`、`.git`、`README.md` が出ず、`src` の一覧にも
  `.env.local` は出ません。エージェントには一覧を許可されたエントリだけが見えます。
- **拒否の応答はどれも同じ。** 拒否された要求は、ファイルがあってもなくても常に
  `policy_denied: the operation is not allowed` を返すので、エージェントは隠れたファイルの
  有無を探れません。
- **すべて記録する。** 要求ごとに JSON を 1 行 stderr に出します。ファイルの内容は出しません。
  たとえば上の `.env` の読み取りは次のようになります。

  ```json
  {"time":"…","level":"INFO","msg":"operation","action":"read","path":".env","decision":"deny","rule":"env-deny","result":"policy_denied","duration_ms":0.0017}
  ```

## 準備とビルド

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

## チェックを実行する

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

## ポリシーを書く

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

ポリシーファイルを検査するコマンドはまだありません。`go test ./internal/policy` がサンプルポリシーを
読み込み、主な判定を確かめているので、読み込めなくなったり主な判定が変わったりすると `task test` が失敗します。
