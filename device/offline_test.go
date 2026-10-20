package device

import (
	"context"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

// A hub that answered a moment ago gets several tries (a relay restart);
// one that has been away is reported after one quick retry.
func TestWhileOffline(t *testing.T) {
	ctx := context.Background()
	off := func(n *int) func() error {
		return func() error { *n++; return &omnillm.Error{Code: omnillm.CodeHubOffline} }
	}
	var d Device
	n := 0
	start := time.Now()
	_ = d.whileOffline(ctx, off(&n))
	if n != 2 || time.Since(start) > 2*time.Second {
		t.Fatalf("away hub: %d tries in %s", n, time.Since(start))
	}
	d.lastOK.Store(time.Now().UnixNano())
	n = 0
	tries := 0
	_ = d.whileOffline(ctx, func() error {
		tries++
		if tries < 3 {
			n++
			return &omnillm.Error{Code: omnillm.CodeHubOffline}
		}
		return nil
	})
	if tries != 3 {
		t.Fatalf("blip: %d tries", tries)
	}
}
