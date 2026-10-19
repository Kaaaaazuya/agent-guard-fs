# Policy

English | [日本語](policy.ja.md)

The policy is a [CUE](https://cuelang.org/) file that lists the operations the
agent may perform on the root directory. It is loaded and validated once at
startup. A policy that fails validation stops the server from starting.

The working example is [`examples/basic/policy.cue`](../examples/basic/policy.cue),
which applies to [`examples/basic/project`](../examples/basic/project).

## Top-level structure

```cue
package config

version: 1

limits: {
	maxReadBytes:  1048576
	maxWriteBytes: 1048576
	// maxListEntries is optional. Default: 1000.
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

- `version` must be `1`.
- `limits` sets the size caps in bytes. `maxReadBytes` and `maxWriteBytes`
  are required; each must be between 1 and 67108864 (64 MiB).
  `maxListEntries` is optional, defaults to 1000, and must be between 1 and
  100000. A `list` result larger than that is cut off in name order and
  marked `truncated: true`.
- `rules` is a map. The key is the rule name. Names must be unique, and the
  name appears in audit records and in `policy_denied` errors.

Unknown fields are rejected at every level (top level, `limits`, and each
rule).

## Rule fields

| Field | Required | Values | Meaning |
|---|---|---|---|
| `path` | yes | a relative path with optional globs | The paths the rule covers. |
| `actions` | yes | `"read"`, `"write"`, `"delete"`, `"list"` | Operations the rule applies to. No duplicates. |
| `effect` | yes | `"allow"` or `"deny"` | What happens when the rule matches. |
| `recursive` | no, default `false` | `true` or `false` | Whether the rule also covers paths below `path`. |

### Actions

- `read`: read a file's content.
- `write`: create a file or replace it entirely. There is no append or partial
  update.
- `delete`: delete a regular file. Directories and symlinks cannot be deleted.
- `list`: list the direct children of a directory. `list` is not recursive.

## How a decision is made

For each operation the server checks, in order:

1. The path is validated and normalized (see [Paths](#paths)). A bad path is
   `invalid_argument`.
2. If any `deny` rule matches the path and the action, the operation is
   denied with `policy_denied`.
3. If no `deny` rule matches and an `allow` rule matches, the operation
   continues.
4. Otherwise the operation is denied with `policy_denied`. This is the
   default-deny rule.
5. Only after the policy allows the operation does the file system layer run
   it. A missing file then returns `not_found`.

Because the policy is checked before the file system, a denied path returns
`policy_denied` whether or not the file exists. An agent cannot tell from the
error whether a protected file is there.

Deny always wins. With several matching rules, a single matching deny blocks
the operation. There is no priority order between allow rules and no way to
override a deny from another rule.

## Paths

Rule paths and request paths follow the same rules.

- Use `/` as the separator. `\` and `:` are rejected on every OS, so one
  policy means the same thing everywhere.
- No empty paths, leading `/`, NUL, or control characters (including newline
  and tab).
- `..` is rejected in any position, even if it would stay inside the root.
- `./a`, `a//b`, and a trailing `/` are normalized and accepted.
- The root itself is `.`.
- Paths must be valid UTF-8.

Rule paths also accept glob characters, described below. The characters `[`,
`]`, `{`, and `}` are rejected in rule paths.

## Globs

- `*` matches any run of characters inside one path element. It does not
  match `/`.
- `?` matches exactly one character other than `/`.
- `**` is valid only as a whole path element. It matches zero or more path
  elements. So `**/.env*` matches `.env` in any directory, and `src/**`
  matches `src` itself as well as everything under it.
- Matching works on whole path elements. `src` does not match `src-old`.

`recursive` is the other way to cover a subtree. These two rules cover the
same paths:

```cue
"a": { path: "src", actions: ["read"], effect: "allow", recursive: true }
"b": { path: "src/**", actions: ["read"], effect: "allow" }
```

`recursive: false` (the default) matches only the path itself. So a
non-recursive rule on `.` allows listing the root but says nothing about the
files inside it. Use `recursive: true` or a `**` glob to cover what is below.

## Case and Unicode

- **Deny rules ignore case and Unicode normalization.** Both the rule and the
  request are normalized to NFC and folded with Unicode case folding before
  comparison. `.ENV` therefore matches a `.env` deny, even on a
  case-sensitive file system.
- **Allow rules are case-sensitive.** Both sides are normalized to NFC, but
  letters must match exactly. An allow on `README.md` does not allow
  `readme.md`.

Because deny is the strict side, a mismatch between case or normalization can
only make the server refuse more, never less.

## Examples

These come from `examples/basic/policy.cue`, with the reason for each.

```cue
// Read and write the whole src tree.
"source-read":  { path: "src", actions: ["read", "list"], effect: "allow", recursive: true }
"source-write": { path: "src", actions: ["write"], effect: "allow", recursive: true }

// docs is read-only.
"docs-read": { path: "docs", actions: ["read", "list"], effect: "allow", recursive: true }

// The root can be listed, but its children are checked one by one.
"root-list": { path: ".", actions: ["list"], effect: "allow" }

// go.mod can be read, not changed.
"go-mod-read": { path: "go.mod", actions: ["read", "list"], effect: "allow" }
"go-mod-deny": { path: "go.mod", actions: ["write", "delete"], effect: "deny" }

// Every .env and .env.* file at any depth is hidden from every action.
"env-deny": { path: "**/.env*", actions: ["read", "write", "delete", "list"], effect: "deny" }

// .git is hidden completely. Changing hooks or config there can run code.
"git-deny": { path: ".git", actions: ["read", "write", "delete", "list"], effect: "deny", recursive: true }
```

Some of these combine. `env-deny` hides `src/.env` even though `source-read`
allows everything under `src`, because deny wins.

### Listing a directory

`list_files` on a directory needs a `list` allow on that directory. Each child
is then checked on its own, and children that are denied or not allowed are
left out of the result rather than causing an error. In the sample, listing
the root returns `docs`, `go.mod`, and `src`, but not `.env`.

If nothing is allowed below a directory, its children do not appear in
`list_files`, even though the directory itself is listed.

## Sharing values between rules

The policy file is ordinary CUE, so a value that several rules need can be
written once. Fields that start with `_` are hidden: CUE accepts them but
they are not part of the policy. The sample builds its deny rules from one
map:

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

This produces the same `env-deny` and `git-deny` rules as writing them out.
Adding a path to `_hidden` denies every action on it. The result is still
checked against the schema, so a rule built this way must be valid like any
other.

## Tests

A policy can state the decisions its author expects. Each test names a path,
an action, and the result (`"allow"` or `"deny"`):

```cue
tests: [
	{path: "src/main.go", action: "read", want: "allow"},
	{path: "src/.env", action: "read", want: "deny"},
]
```

When the policy is loaded, every test runs against the same decision logic
the server uses: deny first, then allow, then default deny. If any result
differs from `want`, the server does not start and the error lists every
test that failed:

```text
policy tests failed:
tests[1]: read src/.env: want deny, got allow (rule "src-read")
```

- `tests` is optional. Without it nothing extra runs.
- `path` is a concrete path with the same rules as a request path, not a glob.
- Tests only check the policy. They cannot grant or remove access.

Write a test for each deny rule, and for one path next to it that must stay
allowed. A later edit that loosens a rule then fails at startup instead of
when the agent reads a secret.

## Rules that never take effect

Deny wins over allow, so an allow rule is useless when a deny rule already
covers everything it allows. The server and `agentguard explain` print a
warning to standard error and carry on:

```text
agentguard: warning: allow rule "env-read" never takes effect: everything it allows is denied by "env-deny"
```

The check is exact but not complete. It only looks at allow rules whose
`path` has no `*` or `?`, and it reports a rule only when every action it
grants is denied. For a `recursive` allow rule, the covering deny rule must
also reach everything below (`recursive: true`, or a path ending in `**`). A
rule that is not reported can still be useless.

## Writing a policy

- Start with the smallest set of allows. Everything else is denied by default.
- Add a deny for each secret or tool-state path that an allow might cover:
  at least `**/.env*`, and `.git` if the root is a repository.
- Keep the policy file outside the root and owned by someone you trust. The
  agent must not be able to change it.
- Add [tests](#tests) for what must stay denied. To see which rule decides a
  path, run [`agentguard explain`](usage.md#explaining-a-decision) on it.
  Check a change end to end by starting the server and calling the tools, or
  by running the sample test in `cmd/agentguard`.

## Validation errors

The server reports a clear message at startup for:

- a missing or unknown field,
- `version` other than `1`,
- an `action` or `effect` that is not one of the listed values,
- duplicate actions in one rule,
- a rule `path` that is empty, absolute, contains `..`, `\`, `:`, a control
  character, or `[`, `]`, `{`, `}`,
- a size limit outside its range,
- a `tests` entry with an unknown field, action, or `want`, or a path that is
  not a valid request path,
- a test whose `want` differs from the actual decision.

The policy is validated in CUE and again in Go. A value that passes one check
but fails the other is refused.

## Related

- [usage.md](usage.md): running the server and connecting a client.
- [security.md](security.md): what the policy does and does not protect.
