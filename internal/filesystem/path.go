//go:build linux || darwin

package filesystem

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// tempPrefix は書き込み途中の一時ファイルの名前の接頭辞（DESIGN §9.3）。
// この名前のパスは、ポリシーにかかわらず全操作で拒否し、list にも出さない。
const tempPrefix = ".agentguard-tmp-"

// checkName は、最後の要素が一時ファイルの名前でないことを確認する。
func checkName(rel pathutil.RelPath) error {
	if strings.HasPrefix(baseName(rel), tempPrefix) {
		return errs.New(errs.InvalidArgument, "the name is reserved for temporary files")
	}
	return nil
}

// baseName は rel の最後の要素を返す。ルート自身は "." を返す。
func baseName(rel pathutil.RelPath) string {
	s := rel.String()
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// openFile は rel を symlink を拒否して開く。FIFO で止まらないよう O_NONBLOCK を付け、
// 種別の確認は開いた記述子に対して呼び出し側で行う。
func (r *Root) openFile(rel pathutil.RelPath) (*os.File, error) {
	fd, err := openBeneath(r.fd, rel.String(), unix.O_RDONLY|unix.O_NONBLOCK)
	if err != nil {
		return nil, mapOpenError(err)
	}
	return os.NewFile(uintptr(fd), rel.String()), nil
}

// requireRegular は開いた記述子が通常ファイルであることを確認する。
// ディレクトリは is_directory、それ以外（デバイス、FIFO、ソケット等）は unsupported_file_type。
func requireRegular(info os.FileInfo) error {
	switch {
	case info.Mode().IsRegular():
		return nil
	case info.IsDir():
		return errs.New(errs.IsDirectory, "the path is a directory")
	default:
		return errs.New(errs.UnsupportedFileType, "only regular files and directories are supported")
	}
}

// mapOpenError は open 系システムコールのエラーを安定したエラーコードに変換する。
// 元のエラーの文字列は含めない（パスを漏らさないため）。
func mapOpenError(err error) error {
	switch {
	case errors.Is(err, unix.ENOENT):
		return errs.New(errs.NotFound, "the path does not exist")
	case errors.Is(err, unix.ELOOP):
		return errs.New(errs.UnsupportedFileType, "symbolic links are not followed")
	case errors.Is(err, unix.ENOTDIR):
		return errs.New(errs.NotADirectory, "a path element is not a directory")
	case errors.Is(err, unix.EACCES), errors.Is(err, unix.EPERM):
		return errs.New(errs.PermissionDenied, "permission denied")
	case errors.Is(err, unix.EXDEV):
		return errs.New(errs.PermissionDenied, "the path is outside the root")
	default:
		return errs.New(errs.Internal, "the file could not be opened")
	}
}
