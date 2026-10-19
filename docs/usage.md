# Usage guide

English | [日本語](usage.ja.md)

This guide covers what you can do with AgentGuard FS on the current `main`
branch. The project is in early development: the policy, the file operations
and the audit log work as Go packages, but the MCP server that an agent
connects to is not wired up yet. Sections are added here as features land.

## What works today

| Area | State |
| --- | --- |
| Building the `agentguard` binary | Works |
| Policy file format and validation | Works |
| Allow/deny decisions for each operation | Works |
| Reading, writing, listing and deleting files under the root | Works inside the Go packages (`internal/guard`) |
| Audit log (one JSON line per operation) | Works inside the Go packages |
| MCP server and the `agentguard` command line | Not implemented. `agentguard` prints `agentguard: not implemented yet` and exits with status 1 |

## How it fits together

When the MCP server is finished, an AI agent will not touch your files
directly. It calls AgentGuard FS's tools, and every call passes through the
same four steps. The steps on the right already work on `main`; the MCP
connection on the left is planned.

```text
 AI agent (MCP client)                AgentGuard FS
 ─────────────────────               ─────────────────────────────────────────
  read_file   ".env"    ──(planned)──▶ 1. check the path    (no "..", no "/x")
  write_file  "src/a.go"              2. ask the policy    (deny first, default deny)
  list_files  "."                     3. touch the file    (inside --root, no symlinks)
  delete_file "tmp.txt" ◀──────────── 4. write an audit line to stderr
```

The planned command line takes the project directory the agent may work in
and a policy file kept outside it:

```sh
# planned: does not run yet
agentguard --root ./my-project --policy ./my-policy.cue
```

## Walkthrough: the sample policy in action

This walkthrough runs [`examples/basic/policy.cue`](../examples/basic/policy.cue)
against a small project. Every result below was produced by the guard layer
on `main`; only the MCP tool names around it are planned.

The project the agent works in:

```text
my-project/
├── .env            # secrets
├── .git/config
├── README.md
├── docs/guide.md
├── go.mod
└── src/
    ├── .env.local  # more secrets
    └── main.go
```

What the agent gets back for each request:

| Agent asks to | Result | Decided by |
| --- | --- | --- |
| List `.` | `docs`, `go.mod`, `src` | `root-list` (allow) |
| List `src` | `main.go` | `source-read` (allow) |
| Read `src/main.go` | The file's text | `source-read` (allow) |
| Write `src/util.go` | Created | `source-write` (allow) |
| Read `.env` | `policy_denied` | `env-deny` (deny) |
| Write `go.mod` | `policy_denied` | `go-mod-deny` (deny) |
| Read `README.md` | `policy_denied` | No allow rule (default deny) |
| Write `docs/guide.md` | `policy_denied` | No allow rule for `write` (default deny) |
| Delete `src/util.go` | `policy_denied` | No allow rule for `delete` (default deny) |
| Read `../etc/passwd` | `invalid_argument` | Rejected before the policy is asked |

Three things to notice:

- **Hidden, not just blocked.** Listing `.` does not show `.env`, `.git` or
  `README.md`, and listing `src` does not show `.env.local`. The agent only
  sees entries it is allowed to list.
- **Denials look the same.** A denied request always returns
  `policy_denied: the operation is not allowed`, whether or not the file
  exists, so the agent cannot probe for hidden files.
- **Everything is logged.** Each request writes one JSON line to stderr,
  never the file's contents. For example, the `.env` read above:

  ```json
  {"time":"…","level":"INFO","msg":"operation","action":"read","path":".env","decision":"deny","rule":"env-deny","result":"policy_denied","duration_ms":0.0017}
  ```

## Set up and build

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

## Run the checks

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

## Write a policy

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

There is no command to check a policy file yet. `go test ./internal/policy` loads the
example policy and checks several of its decisions, so `task test` fails if
the example stops loading or its main decisions change.
