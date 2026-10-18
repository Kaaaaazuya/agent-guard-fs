// errs パッケージは、AgentGuard FS が MCP クライアントに返す安定したエラーコードを定義する。
// 各層がこのコードを返すことで、MCP 層は変換するだけでよく、下位層のエラーの意味を
// 推測せずに済む。
package errs

import "errors"

// Code は機械可読な安定したエラーコード。文字列値はツールの公開仕様の一部なので変更しない。
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

// Error は Code を持つエラー。Message はエージェントに表示されうるため、絶対パス・
// ファイル内容・秘密情報を含めない。
type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string {
	return string(e.Code) + ": " + e.Message
}

// New は指定したコードとメッセージの *Error を返す。
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// CodeOf は err のチェーンで最初に見つかった *Error の Code を返す。見つからなければ
// Internal を返し、想定外のエラーを安全側（拒否）に倒す。
func CodeOf(err error) Code {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Code
	}
	return Internal
}
