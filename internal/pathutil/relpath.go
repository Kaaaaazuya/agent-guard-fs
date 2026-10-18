// pathutil パッケージは、ルートからの相対パスを検証・正規化する（DESIGN §6.4）。
// policy と filesystem はここで定義する型だけを受け取るので、未検証の文字列パスが
// パッケージ境界を越えることはない。
package pathutil

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

// RelPath は検証・正規化済みのルートからの相対パス。要素の区切りは "/" で、
// ルート自身は "." で表す。ゼロ値は無効なので Parse で生成する。
type RelPath struct {
	p string
}

// Parse は s を検証し、正規化した形で返す。空文字列、絶対パス、".." を含むパス
// （ルート内に収まる場合も含む）、"\" と ":"、NUL などの制御文字、不正な UTF-8 を拒否する。
// "./a"、"a//b"、末尾の "/" は正規化する。
func Parse(s string) (RelPath, error) {
	elems, err := split(s, checkLiteral)
	if err != nil {
		return RelPath{}, err
	}
	return RelPath{p: join(elems)}, nil
}

// String は正規化済みのパスを返す。
func (r RelPath) String() string { return r.p }

// IsRoot は r がルート自身かどうかを返す。
func (r RelPath) IsRoot() bool { return r.p == "." }

// split は s を検証し、"." と空の要素を除いた要素の列を返す。残った各要素は check で検証する。
func split(s string, check func(string) error) ([]string, error) {
	if s == "" {
		return nil, invalid("path is empty")
	}
	if !utf8.ValidString(s) {
		return nil, invalid("path is not valid UTF-8")
	}
	if strings.HasPrefix(s, "/") {
		return nil, invalid("path must be relative to the root")
	}
	for _, r := range s {
		switch {
		case r == '\\' || r == ':':
			return nil, invalid(`path must not contain "\" or ":"`)
		case unicode.IsControl(r):
			return nil, invalid("path must not contain control characters")
		}
	}
	var elems []string
	for e := range strings.SplitSeq(s, "/") {
		switch e {
		case "", ".":
			continue
		case "..":
			return nil, invalid(`path must not contain ".."`)
		}
		if err := check(e); err != nil {
			return nil, err
		}
		elems = append(elems, e)
	}
	return elems, nil
}

// checkLiteral はどの要素も受け付ける。入力パスでは "*" や "?" も普通のファイル名の文字である。
func checkLiteral(string) error { return nil }

func join(elems []string) string {
	if len(elems) == 0 {
		return "."
	}
	return strings.Join(elems, "/")
}

func invalid(msg string) error {
	return errs.New(errs.InvalidArgument, msg)
}
