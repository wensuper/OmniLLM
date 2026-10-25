//go:build !windows

package fsown

import (
	"os"
	"os/user"
	"strconv"
	"strings"
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

// MineOrRoot: owned by this user or by root (a system program). Exact
// only outside a user namespace, where every owner is real; the hub's
// systemd user service therefore has no PrivateTmp (which would put it in
// one, with root and other users alike as "nobody").
func MineOrRoot(fi os.FileInfo) bool {
	u, ok := owner(fi)
	return ok && (u == os.Getuid() || u == 0)
}

// PrivateGroup: the file's group is this user's own private group (named
// after the user, as Linux systems make one per user), so its group
// write bit lets no one else in.
func PrivateGroup(fi os.FileInfo) bool {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || int(st.Gid) != os.Getgid() {
		return false
	}
	u, err := user.Current()
	if err != nil {
		return false
	}
	g, err := user.LookupGroupId(strconv.Itoa(int(st.Gid)))
	if err != nil {
		// Not in /etc/group (an LDAP account; release builds read only
		// the files): the private-group convention, gid equal to uid.
		return int(st.Gid) == os.Getuid()
	}
	return g.Name == u.Username && !hasMembers(g.Name)
}

// hasMembers: /etc/group lists members for the group (someone added to
// the user's own group can write what the group can).
func hasMembers(name string) bool {
	b, err := os.ReadFile("/etc/group")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if f := strings.Split(line, ":"); len(f) == 4 && f[0] == name {
			for _, m := range strings.Split(strings.TrimSpace(f[3]), ",") {
				if m != "" && m != name { // the user themself doesn't count
					return true
				}
			}
			return false
		}
	}
	return false
}
