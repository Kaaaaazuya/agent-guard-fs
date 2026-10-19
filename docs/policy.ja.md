# ポリシー

[English](policy.md) | 日本語

ポリシーは、エージェントがルートディレクトリに対して実行してよい操作を並べた [CUE](https://cuelang.org/) ファイルです。
起動時に一度だけ読み込んで検証します。検証に失敗したポリシーでは、サーバーは起動しません。

動作例は [`examples/basic/policy.cue`](../examples/basic/policy.cue) で、
[`examples/basic/project`](../examples/basic/project) に適用します。

## トップレベルの構造

```cue
package config

version: 1

limits: {
	maxReadBytes:  1048576
	maxWriteBytes: 1048576
	// maxListEntries は省略可能。既定値は 1000。
}

rules: {
	"source-read": {
		path: "src"
		actions: ["read", "list"]
		effect:    "allow"
		recursive: true
	}
}
```

- `version` は `1` でなければなりません。
- `limits` はサイズ上限をバイト単位で指定します。`maxReadBytes` と `maxWriteBytes` は必須で、
  1 から 67108864（64 MiB）の範囲です。`maxListEntries` は省略可能で、既定値は 1000、
  範囲は 1 から 100000 です。`list` の結果がこの件数を超えると、名前順に打ち切って `truncated: true` を付けます。
- `rules` はマップです。キーがルール名になります。名前は重複できず、監査記録と `policy_denied` のエラーに現れます。

未知のフィールドは、トップレベル、`limits`、各ルールのすべてで拒否されます。

## ルールのフィールド

| フィールド | 必須 | 値 | 意味 |
|---|---|---|---|
| `path` | はい | グロブを含められる相対パス | ルールが対象とするパス。 |
| `actions` | はい | `"read"`、`"write"`、`"delete"`、`"list"` | ルールを適用する操作。重複は不可。 |
| `effect` | はい | `"allow"` または `"deny"` | ルールが一致したときの結果。 |
| `recursive` | いいえ（既定は `false`） | `true` または `false` | `path` の配下にも適用するかどうか。 |

### 操作

- `read`: ファイル内容を読む。
- `write`: ファイルを作成するか、全体を置き換える。追記や部分更新はありません。
- `delete`: 通常ファイルを削除する。ディレクトリと symlink は削除できません。
- `list`: ディレクトリの直下の子を一覧する。`list` は再帰しません。

## 判定の手順

サーバーは操作ごとに次の順で判定します。

1. パスを検証して正規化する（[パス](#パス) を参照）。不正なパスは `invalid_argument`。
2. パスと操作に一致する `deny` ルールが 1 つでもあれば、`policy_denied` で拒否する。
3. `deny` に一致せず、一致する `allow` ルールがあれば、操作を続ける。
4. それ以外は `policy_denied` で拒否する。これがデフォルト拒否です。
5. ポリシーが許可した操作だけがファイルシステム層で実行されます。ファイルがなければ、ここで `not_found` になります。

ポリシー判定はファイルシステムより先に行うため、拒否されるパスはファイルがあるかどうかに関係なく `policy_denied` を返します。
エージェントは、保護対象のファイルが存在するかどうかをエラーから推測できません。

拒否は常に優先されます。複数のルールに一致しても、1 つでも一致する deny があれば操作は止まります。
allow 同士に優先順位はなく、別のルールの deny を上書きする方法もありません。

## パス

ルールのパスとリクエストのパスには同じ規則を適用します。

- 区切り文字は `/` だけです。`\` と `:` は OS によらず拒否します。どの OS でも同じポリシーの意味になるようにするためです。
- 空のパス、先頭の `/`、NUL、制御文字（改行やタブを含む）は使えません。
- `..` は、ルート内に収まる場合でも、どの位置でも拒否します。
- `./a`、`a//b`、末尾の `/` は正規化して受け付けます。
- ルート自身は `.` です。
- パスは有効な UTF-8 でなければなりません。

ルールのパスにはグロブも使えます（後述）。ルールのパスでは `[`、`]`、`{`、`}` を使えず、拒否されます。

## グロブ

- `*` は 1 つのパス要素の中の任意の文字列に一致します。`/` には一致しません。
- `?` は `/` 以外の任意の 1 文字に一致します。
- `**` はパス要素全体として書いた場合だけ有効です。0 個以上のパス要素に一致します。
  したがって `**/.env*` はどのディレクトリの `.env` にも一致し、`src/**` は `src` 自身と、その配下のすべてに一致します。
- 一致はパス要素単位で行います。`src` は `src-old` に一致しません。

配下をまとめて指定する方法は `recursive` もあります。次の 2 つのルールは同じパスを対象にします。

```cue
"a": { path: "src", actions: ["read"], effect: "allow", recursive: true }
"b": { path: "src/**", actions: ["read"], effect: "allow" }
```

`recursive: false`（既定）は、`path` に一致するパスそのものだけを対象にします。
たとえば `.` に対する非再帰のルールは、ルートを list できるようにしますが、その中のファイルについては何も言いません。
配下も対象にするには `recursive: true` か `**` を使ってください。

## 大文字小文字と Unicode

- **deny ルールは大文字小文字と Unicode 正規化の違いを無視します。** ルールとリクエストの両方を NFC に正規化し、
  Unicode の case folding をかけてから比べます。したがって、大文字小文字を区別するファイルシステムでも、
  `.ENV` は `.env` の deny に一致します。
- **allow ルールは大文字小文字を区別します。** 両方を NFC に正規化しますが、文字は完全一致で比べます。
  `README.md` への allow は `readme.md` を許可しません。

deny 側は厳しい方向に倒れるので、大文字小文字や正規化の違いがあっても、拒否が増えることはあっても減ることはありません。

## 例

以下は `examples/basic/policy.cue` からの抜粋で、各行の理由を添えています。

```cue
// src ツリー全体を読み書きできる。
"source-read":  { path: "src", actions: ["read", "list"], effect: "allow", recursive: true }
"source-write": { path: "src", actions: ["write"], effect: "allow", recursive: true }

// docs は読み取りのみ。
"docs-read": { path: "docs", actions: ["read", "list"], effect: "allow", recursive: true }

// ルートは一覧できるが、子は 1 つずつ判定される。
"root-list": { path: ".", actions: ["list"], effect: "allow" }

// go.mod は読めるが変更できない。
"go-mod-read": { path: "go.mod", actions: ["read", "list"], effect: "allow" }
"go-mod-deny": { path: "go.mod", actions: ["write", "delete"], effect: "deny" }

// どの階層の .env と .env.* も、すべての操作から隠す。
"env-deny": { path: "**/.env*", actions: ["read", "write", "delete", "list"], effect: "deny" }

// .git は完全に隠す。hooks や config を書き換えるとコードが実行されうるため。
"git-deny": { path: ".git", actions: ["read", "write", "delete", "list"], effect: "deny", recursive: true }
```

これらは組み合わさります。`source-read` が `src` 以下すべてを許可していても、`env-deny` は `src/.env` を隠します。deny が優先されるためです。

### ディレクトリの一覧

`list_files` でディレクトリを一覧するには、そのディレクトリに対する `list` の allow が必要です。
子はそれぞれ個別に判定され、拒否されたものや許可されていないものは、エラーにせず結果から除きます。
サンプルでルートを一覧すると `docs`、`go.mod`、`src` が返り、`.env` は返りません。

ディレクトリの下に何も許可されていなければ、そのディレクトリ自体を一覧できても、子は `list_files` に現れません。

## ルール間で値を共有する

ポリシーファイルは通常の CUE なので、複数のルールで使う値は 1 か所に書けます。`_` で始まるフィールドは隠しフィールドで、CUE は受け付けますが、ポリシーの一部にはなりません。サンプルでは、deny ルールを 1 つのマップから作っています。

```cue
_allActions: ["read", "write", "delete", "list"]

_hidden: {
	"env-deny": path: "**/.env*"
	"git-deny": {
		path:      ".git"
		recursive: true
	}
}

rules: {
	for name, h in _hidden {
		(name): h & {
			actions: _allActions
			effect:  "deny"
		}
	}
}
```

これは `env-deny` と `git-deny` を直接書いた場合と同じルールになります。`_hidden` にパスを足すと、そのパスへのすべての操作が拒否されます。結果はスキーマで検証されるので、この方法で作ったルールも、ほかのルールと同じ条件を満たす必要があります。

## ポリシーの書き方

- 許可は必要最小限にします。それ以外はデフォルトで拒否されます。
- allow が覆いうる秘密や状態のパスには、必ず deny を付けます。少なくとも `**/.env*` と、ルートがリポジトリなら `.git` です。
- ポリシーファイルはルートの外に置き、信頼できる人が管理してください。エージェントがポリシーを変更できてはいけません。
- 変更の確認は、サーバーを起動してツールを呼ぶか、`cmd/agentguard` のサンプルテストを実行して行います。

## 検証エラー

サーバーは次の場合に、起動時に明確なメッセージを出します。

- フィールドが欠けている、または未知のフィールドがある。
- `version` が `1` 以外。
- `action` または `effect` が定義された値ではない。
- 1 つのルールで同じ action が重複している。
- ルールの `path` が空、絶対パス、`..` を含む、`\`、`:`、制御文字、または `[`、`]`、`{`、`}` を含む。
- サイズ上限が範囲外。

ポリシーは CUE と Go の両方で検証します。片方を通っても、もう片方で失敗した値は拒否されます。

## 関連

- [usage.ja.md](usage.ja.md): サーバーの起動とクライアントの接続。
- [security.ja.md](security.ja.md): ポリシーが守るものと守らないもの。
