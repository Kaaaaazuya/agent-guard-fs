# AgentGuard FS

English | [日本語](README.ja.md)

AgentGuard FS is an [MCP](https://modelcontextprotocol.io/) server that gives
AI agents local file operations (`read_file`, `write_file`, `list_files`,
`delete_file`) and allows or denies each one according to a policy written in
[CUE](https://cuelang.org/).

> **Status:** pre-release. All four tools work under a policy, but no version
> has been tagged yet. See the [changelog](CHANGELOG.md).

## Design in brief

- **Default deny, deny first.** An operation runs only if an allow rule
  matches and no deny rule does. Deny rules ignore case and Unicode
  normalization differences, so `.ENV` cannot slip past a `.env` deny.
- **One root.** All operations stay inside the directory given with
  `--root`. The policy file must live outside that root.
- **No symlinks.** Any path that contains a symlink, in any element, is
  rejected. Files are opened relative to a directory descriptor with
  `openat2(RESOLVE_BENEATH|RESOLVE_NO_SYMLINKS)` on Linux and
  `O_NOFOLLOW_ANY` on macOS. If the kernel does not support these, the
  server refuses to start.
- **Text only.** Reads and writes are UTF-8 text with size limits.

## Threat model

AgentGuard FS runs with the invoking user's OS permissions. It is not a
sandbox: it is only meaningful when the agent has no other way to touch the
file system (for example, no shell tool). The people who choose the root
directory and write the policy are trusted; the agent is not.

## Requirements

- Linux 5.6 or later, or macOS 11 or later
- Go 1.26 or later to build

## Build

Tasks are defined in `Taskfile.yml` and run with [Task](https://taskfile.dev/) v3.

```sh
task build        # writes bin/agentguard
task              # fmt-check, vet, lint, test, build
task --list       # show all tasks
```

`task lint` needs [golangci-lint](https://golangci-lint.run/) v2.

### Fetching dependencies through Takumi Guard (recommended)

CI fetches Go modules through [Takumi Guard](https://shisho.dev/docs/t/guard/quickstart/golang/),
a proxy that blocks known-malicious modules. To do the same locally:

```sh
export GOPROXY=https://golang.flatt.tech   # no ",direct" fallback
```

## Quick start

Build the binary, then run it against the sample project. The policy file
must stay outside the root you pass to `--root`.

```sh
task build
bin/agentguard --root ./examples/basic/project --policy ./examples/basic/policy.cue
```

The server speaks MCP over standard input and output, so an MCP client starts
it rather than you. The sample policy lets the agent read and write `src/`,
read `docs/` and `go.mod`, and never see `.env` files or `.git/`.

## Documentation

- [Usage](docs/usage.md): build, command-line flags, startup refusals, MCP
  client configuration, and troubleshooting.
- [Policy](docs/policy.md): rule fields, actions, globs, deny-first matching,
  and examples.
- [Security](docs/security.md): threat model, protections, residual risks,
  minimum OS versions, and error codes.

## License

[Apache License 2.0](LICENSE)

The runtime dependencies (CUE, the MCP Go SDK, and their transitive modules)
use the Apache-2.0, MIT, BSD-3-Clause, or ISC licenses, which are compatible
with Apache-2.0.
