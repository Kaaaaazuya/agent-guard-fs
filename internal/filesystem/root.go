//go:build linux || darwin

package filesystem

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Root は起動時に開いたルートディレクトリの記述子。
type Root struct {
	fd int
}

// Open は dir をルートとして開き、symlink を拒否するための open 方式が使えることを確認する。
// 使えない環境では、より弱い方式に切り替えず error を返す（fail closed）。
// dir 自体は CLI で指定された信頼できるパスなので、dir の途中の symlink は許容する。
func Open(dir string) (*Root, error) {
	fd, err := unix.Openat(unix.AT_FDCWD, dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open root %q: %w", dir, err)
	}
	if err := probe(fd); err != nil {
		unix.Close(fd)
		return nil, err
	}
	return &Root{fd: fd}, nil
}

// Close はルートの記述子を閉じる。
func (r *Root) Close() error {
	return unix.Close(r.fd)
}

// probe は、ルート自身を symlink 拒否の方式で開けるかを確認する。
func probe(fd int) error {
	f, err := openBeneath(fd, ".", unix.O_RDONLY|unix.O_DIRECTORY)
	if err != nil {
		return fmt.Errorf("the root cannot be opened without following symlinks (this platform needs openat2 on Linux or O_NOFOLLOW_ANY on macOS): %w", err)
	}
	return unix.Close(f)
}
