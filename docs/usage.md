# Usage guide

English | [日本語](usage.ja.md)

This guide covers what you can do with AgentGuard FS on the current `main`
branch. The project is in early development: the policy format and its
validation work, but the MCP server itself does not run yet. Sections are
added here as features land.

## What works today

| Area | State |
| --- | --- |
| Building the `agentguard` binary | Works |
| Policy file format and validation | Works (used from Go tests; no command-line check yet) |
| Rule path syntax and normalization | Works |
| MCP server and the file tools | Not implemented. `agentguard` prints `agentguard: not implemented yet` and exits with status 1 |

## 1. Set up and build

You need:

- Linux 5.6 or later, or macOS 11 or later
- Go 1.26 or later
- [Task](https://taskfile.dev/) v3
- [golangci-lint](https://golangci-lint.run/) v2, only for `task lint`

```sh
git clone https://github.com/Kaaaaazuya/agent-guard-fs.git
cd agent-guard-fs

# Optional but recommended: fetch modules through Takumi Guard, as CI does.
export GOPROXY=https://golang.flatt.tech   # no ",direct" fallback

task build          # writes bin/agentguard
./bin/agentguard    # prints "agentguard: not implemented yet", exit status 1
```

## 2. Run the checks

| Command | What it does |
| --- | --- |
| `task` | Runs `fmt-check`, `vet`, `lint`, `test` and `build` in order |
| `task fmt` | Formats Go code with `gofmt -w` |
| `task fmt-check` | Fails if any Go file needs `gofmt` |
| `task vet` | Runs `go vet ./...` |
| `task lint` | Runs `golangci-lint run` |
| `task test` | Runs `go test -race ./...` |
| `task vuln` | Runs `govulncheck` |
| `task build` | Builds `bin/agentguard` |
| `task clean` | Removes `bin/` |

Run `task --list` to see the same list from the Taskfile.

## 3. Write a policy

A policy is a single CUE file. Start from
[`examples/basic/policy.cue`](../examples/basic/policy.cue).

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

The file is checked against a schema built into the binary, so you cannot
replace or loosen the schema from outside. Fields not listed below are
rejected.

### Top-level fields

| Field | Required | Value |
| --- | --- | --- |
| `version` | Yes | Must be `1` |
| `limits` | Yes | See [Limits](#limits) |
| `rules` | Yes | A map from rule name to rule. The name is any string you choose and appears in error messages |

### Limits

| Field | Required | Range | Default |
| --- | --- | --- | --- |
| `maxReadBytes` | Yes | 1 to 67108864 (64 MiB) | None |
| `maxWriteBytes` | Yes | 1 to 67108864 (64 MiB) | None |
| `maxListEntries` | No | 1 to 100000 | 1000 |

### Rules

| Field | Required | Value |
| --- | --- | --- |
| `path` | Yes | A path relative to the root, see [Rule paths](#rule-paths) |
| `actions` | Yes | A non-empty list of `"read"`, `"write"`, `"delete"`, `"list"`, with no duplicates |
| `effect` | Yes | `"allow"` or `"deny"` |
| `recursive` | No | `true` or `false` (default `false`) |

### Rule paths

A rule path is relative to the root directory and uses `/` as the separator.

- Allowed: `.` (the root itself), plain names such as `src/main.go`, and the
  globs `*`, `?` and `**`.
- `**` must be a whole path element: `**/.env` is fine, `src/a**` is not.
- Rejected: an empty path, a leading `/`, any `..` element (even one that
  stays inside the root), `\`, `:`, control characters such as NUL, invalid
  UTF-8, and the glob characters `[`, `]`, `{`, `}`.
- `./`, repeated `/` and a trailing `/` are removed, so `./src//` is the same
  as `src`.

### Where to put the policy file

Keep the policy file outside the directory the agent will work in, so the
agent can never edit its own policy.

### When a policy is invalid

An invalid policy is rejected with an error that points at the file and line.
For example, an unknown action gives (shortened):

```text
invalid policy:
#Policy.rules.r.actions.0: conflicting values "read" and "exec":
    my-policy.cue:3:35
    ...
```

and a path that climbs out with `..`:

```text
invalid policy: rule "r": invalid_argument: path must not contain ".."
```

There is no command to check a policy file yet. The example policy is loaded
by `go test ./internal/policy`, so `task test` fails if it ever becomes
invalid.
