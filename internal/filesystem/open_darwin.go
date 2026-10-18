package filesystem

import "golang.org/x/sys/unix"

// openBeneath は dirfd からの相対で name を開く。O_NOFOLLOW_ANY は途中のどの要素が symlink でも
// 開かずに ELOOP を返す。検査と open は一つのシステムコールで行われる。
func openBeneath(dirfd int, name string, flags int) (int, error) {
	return unix.Openat(dirfd, name, flags|unix.O_NOFOLLOW_ANY|unix.O_CLOEXEC, 0)
}
