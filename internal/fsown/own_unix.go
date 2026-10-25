//go:build !windows

package fsown

import (
	"os"
	"strconv"
	"strings"
	"sync"
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

// MineOrRoot: owned by this user or by root (a system program). In a
// Linux user namespace (a systemd user service with PrivateTmp, say) root
// and every other user show as the overflow uid ("nobody"): such an owner
// can't be told apart there and is accepted; checks made outside the
// namespace (the installer, the person's own commands) tell them apart.
func MineOrRoot(fi os.FileInfo) bool {
	u, ok := owner(fi)
	if !ok {
		return false
	}
	if u == os.Getuid() || u == 0 {
		return true
	}
	nsOnce.Do(readNS)
	return nsMapped && u == nsOverflow
}

var (
	nsOnce     sync.Once
	nsMapped   bool // in a user namespace that doesn't map every uid
	nsOverflow int
)

func readNS() {
	b, err := os.ReadFile("/proc/self/uid_map")
	if err != nil {
		return
	}
	if f := strings.Fields(string(b)); len(f) == 3 && f[0] == "0" && f[1] == "0" && f[2] == "4294967295" {
		return // the initial namespace: every owner is real
	}
	nsOverflow = 65534
	if b, err := os.ReadFile("/proc/sys/kernel/overflowuid"); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
			nsOverflow = v
		}
	}
	nsMapped = true
}
