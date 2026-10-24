package relay

import (
	"fmt"
	"testing"
	"time"

	"github.com/wensuper/OmniLLM/internal/relayproto"
)

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

// The wrong-secret table stops growing at its cap: a flood of new
// addresses is not recorded instead of filling memory.
func TestFailureTableIsCapped(t *testing.T) {
	s := New(Options{})
	for i := 0; i < maxFailureBuckets; i++ {
		b := &bucket{rate: 0.2, burst: 10, tokens: 0, last: time.Now()}
		s.failures[fmt.Sprintf("10.%d.%d.%d", i>>16, (i>>8)&255, i&255)] = b
	}
	if s.tooManyFailures("192.0.2.1", true) {
		t.Fatal("a new address was refused because the table is full")
	}
	if len(s.failures) > maxFailureBuckets {
		t.Fatalf("table grew past its cap: %d", len(s.failures))
	}
}

// Frames waiting for one device are counted in bytes as they are queued.
func TestDeviceQueueCountsBytes(t *testing.T) {
	s := New(Options{})
	h := &hubConn{claims: map[relayproto.ID]bool{}}
	var mid relayproto.ID
	d := &devConn{out: make(chan outItem, 128), cancel: func() {}}
	s.mailboxes[mid] = &mailbox{hub: h, dev: d}
	s.handleHubFrame(h, relayproto.Data, mid, make([]byte, 1000))
	s.handleHubFrame(h, relayproto.Data, mid, make([]byte, 500))
	if got := d.queued.Load(); got != 1500 {
		t.Fatalf("queued %d bytes, want 1500", got)
	}
}
