# Changelog

All notable changes to this project are documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Project skeleton: Go module, `agentguard` command stub, Taskfile, CI on
  Linux and macOS with Go 1.26 and 1.27, golangci-lint and govulncheck.
- CUE policy with `read`, `write`, `delete`, and `list` rules, `recursive`
  paths, `*`, `**`, and `?` globs, deny-first and default-deny decisions,
  and case- and Unicode-insensitive deny matching.
- Filesystem layer that opens every path relative to the root descriptor
  without following symlinks (`openat2` on Linux, `O_NOFOLLOW_ANY` on macOS),
  with atomic writes, size limits, and UTF-8 checks.
- Guard layer that evaluates the policy before each operation, filters
  `list` results per entry, and records one audit event per operation.
- MCP server over STDIO with `read_file`, `write_file`, `list_files`, and
  `delete_file`, and structured errors with stable codes.
- `agentguard` command with `--root` and `--policy`. The policy must be
  outside the root, and startup fails closed if the kernel cannot open
  without following symlinks.
- Sample project and policy in `examples/basic`, covered by a test.
- Policy `tests`: expected decisions that run when the policy is loaded and
  stop startup if any of them differs.
- Documentation for usage, policy, and security, in English and Japanese.
