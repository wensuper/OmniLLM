//go:build !linux && !darwin

package main

import "net"

// socketPeer: not known on this system.
func socketPeer(*net.UnixConn) int { return 0 }
