//go:build linux || darwin

package filesystem

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// newFileMode は新規作成するファイルのモード。umask はカーネルが適用する。
const newFileMode = 0o644

// Write は rel の内容を content で置き換え、無ければ作成する（DESIGN §9.3）。
// 新規に作成した場合は created が true になる。
// 親ディレクトリは作らない。途中で失敗またはキャンセルされた場合は一時ファイルを消し、
// 元のファイルは変更しない。既存ファイルのモードは引き継ぐ。
func (r *Root) Write(ctx context.Context, rel pathutil.RelPath, content []byte, maxBytes int) (created bool, err error) {
	if err := checkName(rel); err != nil {
		return false, err
	}
	if len(content) > maxBytes {
		return false, errs.New(errs.SizeLimitExceeded, "the content is larger than the write limit")
	}

	parentFD, err := r.openParent(rel)
	if err != nil {
		return false, err
	}
	defer func() { _ = unix.Close(parentFD) }()
	name := baseName(rel)

	mode, exists, err := existingMode(parentFD, name)
	if err != nil {
		return false, err
	}

	tmp, err := createTemp(parentFD, mode)
	if err != nil {
		return false, err
	}
	// 置き換えが成功するまで、一時ファイルを残さない。
	replaced := false
	defer func() {
		if !replaced {
			_ = unix.Unlinkat(parentFD, tmp.name, 0)
		}
	}()
	defer func() { _ = tmp.file.Close() }()

	if _, err := tmp.file.Write(content); err != nil {
		return false, errs.New(errs.Internal, "the file could not be written")
	}
	if err := tmp.file.Sync(); err != nil {
		return false, errs.New(errs.Internal, "the file could not be synced")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := unix.Renameat(parentFD, tmp.name, parentFD, name); err != nil {
		return false, errs.New(errs.Internal, "the file could not be replaced")
	}
	replaced = true

	// 親ディレクトリの同期はベストエフォート。失敗しても書き込みは完了している。
	_ = unix.Fsync(parentFD)
	return !exists, nil
}

// openParent は rel の親ディレクトリを symlink を拒否して開き、その記述子を返す。
// ルート直下のファイルの親はルート自身である。
func (r *Root) openParent(rel pathutil.RelPath) (int, error) {
	fd, err := openBeneath(r.fd, path.Dir(rel.String()), unix.O_RDONLY|unix.O_DIRECTORY)
	if err != nil {
		return -1, mapOpenError(err)
	}
	return fd, nil
}

// existingMode は name が既存の通常ファイルならそのモードと true を返す。無ければ新規作成用の
// モードと false を返す。種別の確認は fstatat(AT_SYMLINK_NOFOLLOW) で行い、symlink を置き換えない。
func existingMode(parentFD int, name string) (os.FileMode, bool, error) {
	var st unix.Stat_t
	err := unix.Fstatat(parentFD, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	switch {
	case errors.Is(err, unix.ENOENT):
		return newFileMode, false, nil
	case err != nil:
		return 0, false, errs.New(errs.Internal, "the file could not be inspected")
	case st.Mode&unix.S_IFMT == unix.S_IFDIR:
		return 0, false, errs.New(errs.IsDirectory, "the path is a directory")
	case st.Mode&unix.S_IFMT != unix.S_IFREG:
		return 0, false, errs.New(errs.UnsupportedFileType, "only regular files are replaced")
	default:
		return os.FileMode(st.Mode & 0o777), true, nil
	}
}

type tempFile struct {
	name string
	file *os.File
}

// createTemp は親ディレクトリに O_EXCL で一時ファイルを作り、モードを設定する。
// 既存ファイルのモードは umask に関係なく引き継ぐため、作成後に fchmod する。
func createTemp(parentFD int, mode os.FileMode) (tempFile, error) {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return tempFile{}, errs.New(errs.Internal, "a temporary name could not be generated")
	}
	name := tempPrefix + hex.EncodeToString(suffix[:])

	fd, err := unix.Openat(parentFD, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC, newFileMode)
	if err != nil {
		return tempFile{}, errs.New(errs.Internal, "the temporary file could not be created")
	}
	f := os.NewFile(uintptr(fd), name)
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		_ = unix.Unlinkat(parentFD, name, 0)
		return tempFile{}, errs.New(errs.Internal, "the temporary file mode could not be set")
	}
	return tempFile{name: name, file: f}, nil
}
