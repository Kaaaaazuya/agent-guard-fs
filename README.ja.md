# AgentGuard FS

[English](README.md) | 日本語

AgentGuard FS は、AI エージェントにローカルファイル操作（`read_file`、`write_file`、
`list_files`、`delete_file`）を提供し、[CUE](https://cuelang.org/) で書いたポリシーに従って
操作ごとに許可・拒否する [MCP](https://modelcontextprotocol.io/) サーバーです。

> **ステータス:** 開発初期です。サーバーはまだ動作しません。

## 設計の概要

- **デフォルト拒否・拒否優先。** allow ルールに一致し、かつどの deny ルールにも一致しない
  操作だけを実行します。deny ルールは大文字小文字と Unicode 正規化の違いを無視して照合するので、
  `.ENV` で `.env` の deny をすり抜けることはできません。
- **ルートは 1 つ。** すべての操作は `--root` で指定したディレクトリ内に限られます。
  ポリシーファイルはそのルートの外に置く必要があります。
- **symlink は使えません。** パスのどの要素に symlink があっても拒否します。ファイルは
  ディレクトリ記述子からの相対で開き、Linux では `openat2(RESOLVE_BENEATH|RESOLVE_NO_SYMLINKS)`、
  macOS では `O_NOFOLLOW_ANY` を使います。カーネルがこれらに対応していない場合は起動しません。
- **テキストのみ。** 読み書きはサイズ上限付きの UTF-8 テキストに限ります。

## 脅威モデル

AgentGuard FS は起動したユーザーの OS 権限で動作します。サンドボックスではありません。
エージェントがファイルシステムに触れる手段を他に持たない場合（シェルツールがない場合など）に
だけ意味を持ちます。ルートディレクトリを選びポリシーを書く人は信頼し、エージェントは信頼しません。

## 動作環境

- Linux 5.6 以上、または macOS 11 以上
- ビルドには Go 1.26 以上

## ビルド

タスクは `Taskfile.yml` に定義してあり、[Task](https://taskfile.dev/) v3 で実行します。

```sh
task build        # bin/agentguard を作る
task              # fmt-check、vet、lint、test、build
task --list       # タスクの一覧
```

`task lint` には [golangci-lint](https://golangci-lint.run/) v2 が必要です。

### Takumi Guard 経由での依存取得（推奨）

CI は、既知の悪性モジュールを遮断するプロキシ
[Takumi Guard](https://shisho.dev/docs/t/guard/quickstart/golang/) 経由で Go モジュールを取得します。
手元でも同じようにするには次のように設定します。

```sh
export GOPROXY=https://golang.flatt.tech   # ",direct" は付けない
```

## 使い方（予定）

```sh
agentguard --root ./examples/basic/project --policy ./examples/basic/policy.cue
```

## ライセンス

[Apache License 2.0](LICENSE)
