package main

import (
	"net"

	"golang.org/x/sys/unix"
)

// socketPeer is the process at the other end of a Unix socket (0 when not
// known): the hub answering on its control socket.
func socketPeer(c *net.UnixConn) int {
	raw, err := c.SyscallConn()
	if err != nil {
		return 0
	}
	pid := 0
	_ = raw.Control(func(fd uintptr) {
		if v, err := unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID); err == nil {
			pid = v
		}
	})
	return pid
}
