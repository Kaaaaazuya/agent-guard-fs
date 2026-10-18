// Package errs defines the stable error codes that AgentGuard FS reports to
// MCP clients. Every layer returns these codes so that the MCP layer only
// has to translate them, never to guess what a lower-level error means.
package errs

import "errors"

// Code is a stable, machine-readable error code. Its string value is part
// of the public tool contract and must not change.
type Code string

const (
	InvalidArgument     Code = "invalid_argument"
	PolicyDenied        Code = "policy_denied"
	NotFound            Code = "not_found"
	SizeLimitExceeded   Code = "size_limit_exceeded"
	UnsupportedFileType Code = "unsupported_file_type"
	IsDirectory         Code = "is_directory"
	NotADirectory       Code = "not_a_directory"
	UnsupportedContent  Code = "unsupported_content"
	PermissionDenied    Code = "permission_denied"
	Internal            Code = "internal_error"
)

// Error is an error carrying a Code. Message must not contain absolute
// paths, file contents, or secrets, because it may be shown to the agent.
type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string {
	return string(e.Code) + ": " + e.Message
}

// New returns an *Error with the given code and message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// CodeOf returns the Code of the first *Error in err's chain, or Internal
// if there is none, so that unexpected errors fail closed.
func CodeOf(err error) Code {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Code
	}
	return Internal
}
