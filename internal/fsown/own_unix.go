//go:build !windows

package fsown

import (
	"os"
	"syscall"
)

func owner(fi os.FileInfo) (int, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(st.Uid), true
}

// Mine: owned by the user running this program.
func Mine(fi os.FileInfo) bool {
	u, ok := owner(fi)
	return ok && u == os.Getuid()
}

// MineOrRoot: owned by this user or by root (a system program).
func MineOrRoot(fi os.FileInfo) bool {
	u, ok := owner(fi)
	return ok && (u == os.Getuid() || u == 0)
}
