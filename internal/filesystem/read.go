//go:build linux || darwin

package filesystem

import (
	"io"
	"unicode/utf8"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// Read は rel の通常ファイルの内容を返す（DESIGN §9.4）。
// 上限は実際の読み取りで強制する。上限を超えたら size_limit_exceeded を返し、
// 内容は返さない。UTF-8 として不正な内容は unsupported_content を返す。
func (r *Root) Read(rel pathutil.RelPath, maxBytes int) ([]byte, error) {
	if err := checkName(rel); err != nil {
		return nil, err
	}
	f, err := r.openFile(rel)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, errs.New(errs.Internal, "the file could not be inspected")
	}
	if err := requireRegular(info); err != nil {
		return nil, err
	}

	// maxBytes+1 まで読み、超えたかどうかを判定する。読み取り中にファイルが伸びた場合も検出できる。
	content, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return nil, errs.New(errs.Internal, "the file could not be read")
	}
	if len(content) > maxBytes {
		return nil, errs.New(errs.SizeLimitExceeded, "the file is larger than the read limit")
	}
	if !utf8.Valid(content) {
		return nil, errs.New(errs.UnsupportedContent, "the file is not valid UTF-8")
	}
	return content, nil
}
