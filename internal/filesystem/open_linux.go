package filesystem

import "golang.org/x/sys/unix"

// openBeneath は dirfd からの相対で name を開く。openat2 の RESOLVE_BENEATH は dirfd の外への
// 移動を、RESOLVE_NO_SYMLINKS は途中のすべての symlink を拒否する。検査と open は一つの
// システムコールで行われるので、検査後の差し替えは起きない。
func openBeneath(dirfd int, name string, flags int) (int, error) {
	return unix.Openat2(dirfd, name, &unix.OpenHow{
		Flags:   uint64(flags | unix.O_CLOEXEC),
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS,
	})
}
