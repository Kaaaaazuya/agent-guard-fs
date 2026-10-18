//go:build linux || darwin

package filesystem

import (
	"os"
	"slices"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
	"github.com/Kaaaaazuya/agent-guard-fs/internal/pathutil"
)

// Entry は直下の要素の名前と種別。種別は "file"、"directory"、"symlink"、"other" のいずれか。
type Entry struct {
	Name string
	Type string
}

// List は rel の直下の要素を名前順に返す。symlink はたどらず、"symlink" として報告する。
// 一時ファイルは含めない（DESIGN §9.3）。ポリシーによる絞り込みと件数の打ち切りは呼び出し側で行う。
func (r *Root) List(rel pathutil.RelPath) ([]Entry, error) {
	fd, err := openBeneath(r.fd, rel.String(), unix.O_RDONLY|unix.O_DIRECTORY)
	if err != nil {
		return nil, mapOpenError(err)
	}
	dir := os.NewFile(uintptr(fd), rel.String())
	defer func() { _ = dir.Close() }()

	dirents, err := dir.ReadDir(-1)
	if err != nil {
		return nil, errs.New(errs.Internal, "the directory could not be read")
	}

	entries := make([]Entry, 0, len(dirents))
	for _, d := range dirents {
		if strings.HasPrefix(d.Name(), tempPrefix) {
			continue
		}
		entries = append(entries, Entry{Name: d.Name(), Type: entryType(d.Type())})
	}
	slices.SortFunc(entries, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
	return entries, nil
}

// entryType は DirEntry の種別を公開用の文字列に変換する。
func entryType(mode os.FileMode) string {
	switch {
	case mode.IsRegular():
		return "file"
	case mode.IsDir():
		return "directory"
	case mode&os.ModeSymlink != 0:
		return "symlink"
	default:
		return "other"
	}
}
