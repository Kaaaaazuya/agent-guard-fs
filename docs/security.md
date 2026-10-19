# Security

English | [日本語](security.ja.md)

This page describes what AgentGuard FS protects against, what it does not,
and the limits you need to know before relying on it. Read it together with
[policy.md](policy.md), which explains how decisions are made.

## What it is for

AgentGuard FS limits the file operations an AI agent can perform through MCP.
It is useful when the agent's only way to touch files is this server. If the
agent also has a shell, an editor with its own file access, or any other tool
that reads or writes the same directory, the policy is bypassed. In that case
AgentGuard FS does not protect the files.

It is not a sandbox. It runs with the same OS permissions as the user who
starts it, and it does not replace containers, separate accounts, or file
permissions.

## Trust assumptions

- The person who starts the server chooses the root and writes the policy.
  They are trusted.
- The agent is not trusted. It may send any request, including malicious
  paths.
- The policy file is owned and maintained by a trusted person and kept outside
  the root.

## What it protects against

| Threat | Protection |
|---|---|
| Reading or writing outside the root with `..`, absolute paths, or `\` | Paths are validated and rejected before policy evaluation. Files are then opened relative to a directory descriptor of the root. |
| Following a symlink inside the root to a protected file, such as `src/leak -> ../.env` | Any symlink in any element of a path is refused. The kernel does the check and the open in one call: `openat2` with `RESOLVE_BENEATH \| RESOLVE_NO_SYMLINKS` on Linux, `openat` with `O_NOFOLLOW_ANY` on macOS. |
| Swapping a path for a symlink between the check and the open | The check and the open are one system call. Writes and deletes operate on the parent directory descriptor with `openat`, `renameat`, and `unlinkat`, not on a path that is resolved again. |
| Case or Unicode variants of a protected name, such as `.ENV` or a decomposed `é` | Deny rules compare with NFC and Unicode case folding. |
| Editing the policy to loosen it, then restarting | Startup refuses a policy whose resolved real path is inside the root, including through an outside symlink. |
| Partial or corrupt writes | Writes go to a temporary file in the same directory, are synced, and are renamed over the target. Cancelled or failed writes remove the temporary file. |
| Huge reads or writes exhausting memory or disk | Size limits (`maxReadBytes`, `maxWriteBytes`) are enforced in the I/O layer, and reads stop one byte past the limit. |
| Special files (devices, FIFOs, sockets) and symlinks as operation targets | Only regular files and directories are operated on. Others return `unsupported_file_type`. |
| Agent-supplied names that collide with the server's own temporary files | Names of the form `.agentguard-tmp-*` are refused for every action and never listed. |
| Logs that leak secrets | Audit records hold time, action, relative path, decision, rule name, result, and duration. They never hold file content. Error messages omit absolute paths and content. |
| Protocol corruption | Standard output is used only for MCP messages. Logs go to standard error. |

## Fail-closed behavior

- A policy that fails to load or validate stops startup. There is no fallback
  policy.
- If the kernel cannot open a directory without following symlinks, the
  server refuses to start. It does not switch to a weaker open.
- Any unexpected error inside the guard is mapped to `internal_error` and the
  operation is not performed.
- A path whose policy evaluation fails is denied.

## Residual risks

These are known limits. They are documented here, not hidden.

### Hard links

A hard link inside the root points to the same file as another name. Paths
cannot tell them apart. If a protected file has a hard link with an allowed
name, reading the allowed name returns the protected content. The server does
not detect hard links in the MVP.

Mitigation: keep protected files out of the root, or make sure no hard links
to them exist inside it.

### Bind mounts and special file systems

A bind mount or a bind-mounted path inside the root can expose files that the
policy never sees through the root path. Access through `/proc` and similar
special file systems is not blocked, except that special file types are
refused as operation targets.

Mitigation: do not put bind mounts inside the root, and do not run the server
where `/proc` or similar paths can reach protected data through the root.

### Other tools the agent can use

As described above, any other file access by the agent bypasses the policy.
This is the most important limit. The server only protects what the agent can
reach through it.

### Runs with the user's permissions

The server can read and write anything the user can, within the root and the
policy. A misconfigured allow rule grants access that the OS would otherwise
not. Review allow rules as carefully as you would review a permission grant.

### Single policy, read once

The policy is read at startup and never reloaded. Changing it requires a
restart, which also means a running server keeps its old policy until stopped.

## Minimum platform

| Platform | Minimum | Mechanism |
|---|---|---|
| Linux | kernel 5.6 | `openat2` with `RESOLVE_NO_SYMLINKS` |
| macOS | 11 | `openat` with `O_NOFOLLOW_ANY` |
| Windows | not supported | |

The probe runs at startup. A kernel or seccomp profile that rejects these
calls causes a startup refusal with a message that names the requirement. See
[usage.md](usage.md#troubleshooting).

## Audit logging

Audit records are JSON lines on standard error, one per operation. The MVP has
no file option. Many MCP clients keep or discard standard error differently,
so you may need to capture it yourself. A `--audit-log <file>` option is
future work.

## Error codes

Tool errors carry one of these stable codes in their structured content.

| Code | When |
|---|---|
| `invalid_argument` | Missing or wrong-typed argument, or a path that fails validation. |
| `policy_denied` | The policy denies the operation. The `rule` field names the deny rule, or is empty for default-deny. Returned whether or not the file exists. |
| `not_found` | Allowed by policy, but the target does not exist. |
| `size_limit_exceeded` | Read or write over the configured size limit. |
| `unsupported_file_type` | Symlink, device, FIFO, socket, or other non-regular target. |
| `is_directory` | A file operation targeted a directory. |
| `not_a_directory` | `list_files` targeted a non-directory, or a parent is not a directory. |
| `unsupported_content` | The file is not valid UTF-8. |
| `permission_denied` | Allowed by policy, but the OS refused the operation. |
| `internal_error` | Unexpected failure. The operation was not performed. |

## Reporting a problem

Report suspected bypasses by opening an issue on the GitHub repository. Include
the policy, the request, and the observed result, with any file contents
removed.
