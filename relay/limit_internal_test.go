package relay

import "testing"

func TestLimitKeyGroupsIPv6By64(t *testing.T) {
	if limitKey("2001:db8:1:2:aaaa::1") != limitKey("2001:db8:1:2:bbbb::9") {
		t.Fatal("two addresses in one /64 counted apart")
	}
	if limitKey("2001:db8:1:2::1") == limitKey("2001:db8:1:3::1") {
		t.Fatal("two /64s counted together")
	}
	if limitKey("203.0.113.7") != "203.0.113.7" {
		t.Fatal("IPv4 changed")
	}
}
