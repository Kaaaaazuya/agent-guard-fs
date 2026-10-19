# 使い方

[English](usage.md) | 日本語

このページでは、インストール、ビルド、起動、MCP クライアントへの接続を説明します。
ポリシーの書き方は [policy.ja.md](policy.ja.md)、脅威モデルは [security.ja.md](security.ja.md) を参照してください。

## インストールとビルド

要件:

- Linux 5.6 以上、または macOS 11 以上。Windows は対象外です。
- Go 1.26 以上。
- [Task](https://taskfile.dev/) v3（任意。以下のコマンドは Task を使います）。
- [golangci-lint](https://golangci-lint.run/) v2（`task lint` を実行する場合のみ）。

バイナリをビルドします。

```sh
task build        # bin/agentguard を出力する
```

Task を使わない場合:

```sh
go build -trimpath -o bin/agentguard ./cmd/agentguard
```

CI と同じ Takumi Guard プロキシ経由で依存を取得する場合:

```sh
export GOPROXY=https://golang.flatt.tech   # ",direct" のフォールバックは付けない
```

すべてのチェック（フォーマット、生成コード、vet、lint、レース検出付きのテスト、ビルド）を実行します。

```sh
task
```

ポリシーを読み込むための Go の型は、`internal/policy/schema/policy.cue` の CUE スキーマから生成します。スキーマを変えたら、型を再生成し、結果は別のコミットにしてください。

```sh
task generate
```

CI が実行する `task generate-check` は、生成されたファイルが古いと失敗します。生成に使う `cue` は、`go.mod` の `cuelang.org/go` と同じバージョン（`Taskfile.yml` の `CUE_VERSION`）です。モジュールを更新するときは、両方をそろえてください。

## コマンドライン

```sh
agentguard --root <dir> --policy <file>
```

| フラグ | 必須 | 意味 |
|---|---|---|
| `--root` | はい | サーバーがアクセスしてよい唯一のディレクトリ。ルートを定義するのはこのフラグだけで、ポリシーに `root` フィールドはありません。 |
| `--policy` | はい | CUE のポリシーファイル。ルートの外に置く必要があります。 |

サーバーは標準入出力で MCP を話します。ログと監査記録は標準エラー出力に出します。
標準出力には MCP 以外を書きません。

終了コード:

| コード | 意味 |
|---|---|
| 0 | 中断シグナル（Ctrl-C や SIGTERM）などで正常に停止した。 |
| 1 | 起動または実行時の失敗。理由を標準エラー出力に出す。 |
| 2 | 引数が不正。使い方の行を標準エラー出力に出す。 |

### 判定の確認

```sh
agentguard explain --policy <file> <read|write|delete|list> <path>
```

ポリシーが 1 つの操作をどう判定し、どのルールが決めたかを表示します。
ポリシーはサーバーと同じ方法で読み込むので、検証や `tests` に失敗するポリシーは拒否されますが、ルートは開かず、サーバーも起動しません。答えは標準出力に出ます。

```text
$ agentguard explain --policy examples/basic/policy.cue read src/.ENV
deny
action: read
path:   src/.ENV
rule:   env-deny
reason: deny rule matched
```

1 行目は `allow` か `deny` です。どの allow ルールにも一致しなかった場合（デフォルト拒否）は、`rule` が `(none)` になります。パスはリクエストと同じ規則で正規化され、`./src//a` は `src/a` と表示されます。判定できたときは拒否でも終了コードが 0 で、ポリシーを読み込めなければ 1、引数が不正、またはサーバーが拒否するパスなら 2 です。

## 起動の順序と起動拒否

起動時は次の順に検査し、最初に失敗した時点で止まります。

1. ポリシーを読み込んで検証する。未知のフィールド、不正な action や effect、範囲外の上限値、失敗する `tests` などがあれば起動しない。deny ルールに必ず覆われる allow ルールは、[警告](policy.ja.md#効かないルール)を出すだけで起動する。
2. ルートディレクトリを開き、カーネルが symlink をたどらずに開けるかを確かめる（後述）。
3. ルートとポリシーのパスをどちらも symlink 解決してから、ポリシーがルートの内側にないかを確かめる。内側なら `the policy file must be outside the root` で起動を拒否する。この検査は symlink をたどるので、ルートの外からの symlink でも内側を指していれば拒否する。
4. 標準入出力で MCP サーバーを開始する。

ポリシーは起動時に一度だけ読みます。変更するにはサーバーを止めて起動し直してください。
再読み込みはありません。

### 拒否: カーネルが symlink をたどらずに開けない

Linux では `openat2(2)` の `RESOLVE_NO_SYMLINKS` が必要です。macOS では `O_NOFOLLOW_ANY` が必要です。
確認に失敗すると、サーバーは次のメッセージで終了します。

```
the root cannot be opened without following symlinks (this platform needs openat2 on Linux or O_NOFOLLOW_ANY on macOS): ...
```

対処は [トラブルシューティング](#トラブルシューティング) を参照してください。サーバーは安全性の低い方式に自動で切り替えません。

## MCP クライアントの設定

絶対パスを使ってください。MCP クライアントは自分の作業ディレクトリからサーバーを起動するため、
相対パスは別の場所を指すことがあります。

クライアントの設定例（ファイル名とキー名はクライアントによって異なります）:

```json
{
  "mcpServers": {
    "agentguard": {
      "command": "/home/you/agent-guard-fs/bin/agentguard",
      "args": [
        "--root", "/home/you/work/my-project",
        "--policy", "/home/you/.config/agentguard/policy.cue"
      ]
    }
  }
}
```

この構成には次の 2 点が必要です。

- エージェントからポリシーファイルが見えないようにする。エージェントのファイルツールが届かない場所に置き、ルートの外に置きます。
- ほかのファイルツール（シェルなど）を外すか制限する。ルートを読み書きできる別のツールがあると、ポリシーは迂回されます。

リポジトリのサンプルを試す場合の例:

```json
{
  "mcpServers": {
    "agentguard-sample": {
      "command": "/absolute/path/to/agent-guard-fs/bin/agentguard",
      "args": [
        "--root", "/absolute/path/to/agent-guard-fs/examples/basic/project",
        "--policy", "/absolute/path/to/agent-guard-fs/examples/basic/policy.cue"
      ]
    }
  }
}
```

リポジトリ内のサンプルをそのまま使うと、書き込みでリポジトリのファイルが変わります。
書き込みを試すなら、先に `examples/basic/project` を別の場所にコピーしてください。

## ツール

| ツール | 入力 | 結果 |
|---|---|---|
| `read_file` | `path` | ファイル内容をテキストで返す。構造化出力: `path`、`size`。 |
| `write_file` | `path`、`content` | ファイルを作成または置換する。構造化出力: `path`、`bytes`、`created`。 |
| `list_files` | `path` | 直下の子のみ。名前と種別（`file`、`dir`、`symlink`、`other`）を返す。構造化出力: `path`、`entries`、`truncated`。 |
| `delete_file` | `path` | 通常ファイルを 1 つ削除する。構造化出力: `path`。 |

パスはルートからの相対パス（例: `src/main.go`）です。ルート自身は `.` です。

エラーは `isError: true` のツール結果として返り、構造化出力に `code`、ポリシーによる拒否では `rule` が入ります。
コードの一覧は [security.ja.md](security.ja.md#エラーコード) にあります。エラーの文面には絶対パス、
ファイル内容、秘密情報を含めません。

書き込みはファイル全体を置き換えます。親ディレクトリが既に存在している必要があり、サーバーはディレクトリを作りません。
書き込みは原子的です。同じディレクトリの一時ファイルに書いてから、対象の上に rename します。

## トラブルシューティング

**`the root cannot be opened without following symlinks ...`**

カーネルが `openat2` または `O_NOFOLLOW_ANY` を拒否しています。よくある原因:

- Linux のカーネルが 5.6 より古い。
- seccomp プロファイルが `openat2` を禁止しているコンテナ実行環境。古い Docker の既定プロファイルや、一部の強化されたサンドボックスがこうなります。そのコンテナの外でサーバーを動かすか、`openat2` を許可するプロファイルを使ってください。
- macOS が 11 より古い。

この拒否を避けるために、より単純な open を使うようコードを書き換えないでください。symlink の保護がなくなります。

**`the policy file must be outside the root`**

ポリシーファイルをルートの外に移すか、`--policy` にルートの外のコピーを指定してください。
symlink では回避できません。検査は解決後の実体パスで行います。

**期待したファイルが `policy_denied` になる**

拒否ルールは許可ルールより優先されます。`**/.env*` や再帰的な `.git` の deny のように、
一致する deny があれば、allow も一致していても操作は拒否されます。エラーの `rule` フィールドが判定したルールの名前です。
既定は拒否なので、どの allow にも一致しないパスも拒否されることを覚えておいてください。

**`size_limit_exceeded`**

ポリシーの `limits.maxReadBytes` または `limits.maxWriteBytes` を上げて、再起動してください。上限は 64 MiB です。

**`unsupported_content`**

ファイルが有効な UTF-8 ではありません。サーバーはテキストだけを扱います。

**クライアントに何も出ない、または接続できない**

`--root` と `--policy` を付けて手で実行し、標準エラー出力の起動メッセージを確認してください。
標準出力には MCP のメッセージだけを流す必要があるので、バイナリを包むスクリプトに `echo` などを入れないでください。

## 監査ログ

操作ごとに、時刻、操作、相対パス、判定、ルール名、結果、所要時間を含む JSON 1 行を標準エラー出力に書きます。
ファイル内容は記録しません。標準エラー出力を保存しない MCP クライアントもあるため、監査記録が失われることがあります。
`--audit-log <file>` オプションは後のリリースで追加する予定です。
