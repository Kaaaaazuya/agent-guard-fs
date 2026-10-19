//go:build linux || darwin

package filesystem

import (
	"errors"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// Delete は rel の通常ファイルを削除する。ディレクトリ、symlink、デバイス等は削除しない。
// 削除は親ディレクトリの記述子に対する unlinkat で行う（DESIGN §9.1）。
func (r *Root) Delete(rel pathutil.RelPath) error {
	if err := checkName(rel); err != nil {
		return err
	}
	parentFD, err := r.openParent(rel)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parentFD) }()
	name := baseName(rel)

	// symlink は、リンク先ではなくリンク自体の種別で判定する。
	var st unix.Stat_t
	if err := unix.Fstatat(parentFD, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return errs.New(errs.NotFound, "the path does not exist")
		}
		return errs.New(errs.Internal, "the file could not be inspected")
	}
	switch st.Mode & unix.S_IFMT {
	case unix.S_IFREG:
	case unix.S_IFDIR:
		return errs.New(errs.IsDirectory, "directories are not deleted")
	default:
		return errs.New(errs.UnsupportedFileType, "only regular files are deleted")
	}

	// AT_REMOVEDIR を付けないので、検査後にディレクトリへ置き換えられても消さずに失敗する。
	if err := unix.Unlinkat(parentFD, name, 0); err != nil {
		return errs.New(errs.Internal, "the file could not be deleted")
	}
	return nil
}
