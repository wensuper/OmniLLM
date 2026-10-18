package provider

import (
	"net"
	"testing"
)

func TestPublicIPTable(t *testing.T) {
	for addr, want := range map[string]bool{
		"8.8.8.8": true, "1.1.1.1": true, "2606:4700:4700::1111": true,
		"10.0.0.1": false, "172.16.0.1": false, "192.168.1.1": false, "127.0.0.1": false,
		"169.254.169.254": false, "100.64.0.1": false, "0.0.0.0": false, "::1": false,
		"::ffff:10.0.0.1": false, "::ffff:8.8.8.8": true, "fd00::1": false, "fe80::1": false,
		"2002:a00:1::1": false, "::7f00:1": false, "2001:2::1": false, "192.88.99.1": false, "3fff::1": false, "2001:0:1::1": false, "64:ff9b::a00:1": false,
	} {
		if got := publicIP(net.ParseIP(addr)); got != want {
			t.Errorf("publicIP(%s) = %v, want %v", addr, got, want)
		}
	}
}
