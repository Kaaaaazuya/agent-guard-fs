# Usage

English | [日本語](usage.ja.md)

This page covers installing, building, starting, and connecting AgentGuard FS
to an MCP client. For the policy language, see [policy.md](policy.md). For the
threat model, see [security.md](security.md).

## Install and build

Requirements:

- Linux 5.6 or later, or macOS 11 or later. Windows is not supported.
- Go 1.26 or later.
- [Task](https://taskfile.dev/) v3 (optional, but the commands below use it).
- [golangci-lint](https://golangci-lint.run/) v2, only if you run `task lint`.

Build the binary:

```sh
task build        # writes bin/agentguard
```

Without Task:

```sh
go build -trimpath -o bin/agentguard ./cmd/agentguard
```

To fetch modules through the Takumi Guard proxy the CI uses:

```sh
export GOPROXY=https://golang.flatt.tech   # no ",direct" fallback
```

Run every check (format, generated code, vet, lint, tests with the race
detector, build):

```sh
task
```

The Go types that decode a policy are generated from the CUE schema in
`internal/policy/schema/policy.cue`. After changing the schema, regenerate them
and commit the result in a separate commit:

```sh
task generate
```

`task generate-check`, which CI runs, fails if the generated file is out of
date. The generator is `cue` at the same version as `cuelang.org/go` in
`go.mod` (`CUE_VERSION` in `Taskfile.yml`); keep the two in step when you
update the module.

## Command line

```sh
agentguard --root <dir> --policy <file>
```

| Flag | Required | Meaning |
|---|---|---|
| `--root` | yes | The only directory the server may access. This is the only place the root is defined; the policy has no `root` field. |
| `--policy` | yes | The CUE policy file. It must be outside the root. |

The server speaks MCP over standard input and output. Logs and audit records go
to standard error. Nothing else is written to standard output.

Exit codes:

| Code | Meaning |
|---|---|
| 0 | The server ran and stopped normally, for example after an interrupt or SIGTERM. |
| 1 | Startup or runtime failure. The reason is printed to standard error. |
| 2 | Invalid arguments. The usage line is printed to standard error. |

### Explaining a decision

```sh
agentguard explain --policy <file> <read|write|delete|list> <path>
```

This prints how the policy decides one operation and which rule decided it.
It loads the policy the same way the server does, so a policy that fails
validation or its own `tests` is refused, but it never opens a root and does
not start a server. Standard output carries the answer:

```text
$ agentguard explain --policy examples/basic/policy.cue read src/.ENV
deny
action: read
path:   src/.ENV
rule:   env-deny
reason: deny rule matched
```

The first line is `allow` or `deny`. `rule` is `(none)` when nothing allowed
the operation (default deny). The path is normalized as it is for a request,
so `./src//a` is shown as `src/a`. The exit code is 0 whenever a decision was
made, including a denial; it is 1 if the policy cannot be loaded and 2 for
invalid arguments or a path the server would reject.

## Startup order and refusals

At startup the server runs these steps in order and stops at the first failure:

1. Load and validate the policy. A bad policy (unknown field, invalid action or
   effect, out-of-range limit, a failing `tests` entry, and so on) refuses to
   start. An allow rule that a deny rule always overrides only prints a
   [warning](policy.md#rules-that-never-take-effect).
2. Open the root directory and check that the kernel can open files without
   following symlinks (see below).
3. Resolve symlinks in both the root and the policy path, then check that the
   policy file is not inside the root. Inside means the server refuses to
   start with `the policy file must be outside the root`. This check follows
   symlinks, so an outside link that points into the root is refused too.
4. Start the MCP server on standard input and output.

The server reads the policy once at startup. To change the policy, stop the
server and start it again. There is no reload.

### Refused: the kernel cannot open without following symlinks

On Linux the server needs `openat2(2)` with `RESOLVE_NO_SYMLINKS`. On macOS it
needs `O_NOFOLLOW_ANY`. If the probe fails, the server exits with:

```
the root cannot be opened without following symlinks (this platform needs openat2 on Linux or O_NOFOLLOW_ANY on macOS): ...
```

See [Troubleshooting](#troubleshooting). The server never falls back to a less
strict method.

## Connect an MCP client

Use absolute paths. MCP clients start the server from their own working
directory, so relative paths may point somewhere else.

A typical client entry (the exact file and key names depend on the client):

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

Two rules for this configuration:

- Keep the agent away from the policy file. Store it where the agent's file
  tools cannot reach it, and keep it outside the root.
- Remove or restrict every other file tool the agent has (for example, a shell
  tool). Another tool that can read or write the root bypasses the policy.

To try the sample project, use the paths in the repository:

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

Running the sample in place changes the files in the repository, so copy
`examples/basic/project` somewhere else first if you plan to write to it.

## Tools

| Tool | Input | Result |
|---|---|---|
| `read_file` | `path` | File content as text. Structured: `path`, `size`. |
| `write_file` | `path`, `content` | Creates or replaces the file. Structured: `path`, `bytes`, `created`. |
| `list_files` | `path` | Direct children only: name and type (`file`, `dir`, `symlink`, `other`). Structured: `path`, `entries`, `truncated`. |
| `delete_file` | `path` | Deletes one regular file. Structured: `path`. |

Paths are relative to the root, such as `src/main.go`. The root itself is `.`.

Errors come back as a tool result with `isError: true` and a structured
`code` and, for policy denials, a `rule` name. Codes are listed in
[security.md](security.md#error-codes). Error text never contains absolute
paths, file content, or secrets.

Writes replace the whole file. The parent directory must already exist; the
server does not create directories. Writes are atomic: content goes to a
temporary file in the same directory and is renamed over the target.

## Troubleshooting

**`the root cannot be opened without following symlinks ...`**

The kernel refused `openat2` or `O_NOFOLLOW_ANY`. Common causes:

- Linux kernel older than 5.6.
- A container runtime whose seccomp profile blocks `openat2`. Older Docker
  default profiles do this, and so do some hardened sandboxes. Run the server
  outside that container, or use a profile that allows `openat2`.
- macOS older than 11.

Do not work around this by editing the code to use a plainer open. That would
remove the symlink protection.

**`the policy file must be outside the root`**

Move the policy file out of the root directory, or point `--policy` at a copy
that lives outside it. A symlink does not help: the check uses the resolved
path.

**`policy_denied` for a file you expected to allow**

Deny rules win over allow rules. A matching deny, such as `**/.env*` or a
recursive `.git` deny, blocks the operation even when an allow rule also matches. The `rule` field in the error names the rule that decided. Remember that
the default is deny: a path with no matching allow rule is denied too.

**`size_limit_exceeded`**

Raise `limits.maxReadBytes` or `limits.maxWriteBytes` in the policy and restart.
The maximum is 64 MiB.

**`unsupported_content`**

The file is not valid UTF-8. The server handles text only.

**The client shows no output or fails to connect**

Run the command by hand with `--root` and `--policy` to see the startup
message on standard error. Standard output must carry only MCP messages, so do
not add `echo` or other prints to a wrapper script around the binary.

## Audit log

Each operation writes one JSON line to standard error with the time, action,
relative path, decision, rule name, result, and duration. File content is
never logged. Some MCP clients discard standard error, so the audit trail can
be lost. A `--audit-log <file>` option is planned for a later release.
