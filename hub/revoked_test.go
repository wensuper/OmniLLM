package hub_test

import (
	"context"
	"errors"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
)

// A removed device is told so (not "hub offline"), now and after it
// reconnects, so it can forget its pairing.
func TestRemovedDeviceIsTold(t *testing.T) {
	e := setup(t, nil)
	d, sec := e.pairOK()
	ctx := context.Background()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	devs, _ := e.hub.Devices(ctx)
	if err := e.hub.Revoke(ctx, devs[0].ID); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d2, err := device.Open(ctx, device.Options{Secrets: sec})
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	_, err = d2.Models(ctx)
	var oe *omnillm.Error
	if !errors.As(err, &oe) || oe.Msg != omnillm.MsgDeviceRevoked {
		t.Fatalf("removed device got %v", err)
	}
	if _, _, err := ask(ctx, d2, "fake/m1", "hello", omnillm.Interactive); !errors.As(err, &oe) || oe.Msg != omnillm.MsgDeviceRevoked {
		t.Fatalf("chat from a removed device: %v", err)
	}
}

// "Remove this device" on the device removes it on the hub too: it leaves
// the device list, is told it was removed if it comes back, and leaving
// again (a retry after a lost answer) still succeeds.
func TestDeviceLeaves(t *testing.T) {
	e := setup(t, nil)
	d, sec := e.pairOK()
	ctx := context.Background()
	if _, err := d.Models(ctx); err != nil {
		t.Fatal(err)
	}
	if err := d.Leave(ctx); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if devs, _ := e.hub.Devices(ctx); len(devs) != 0 {
		t.Fatalf("still listed after leaving: %+v", devs)
	}
	d.Close()
	d2, err := device.Open(ctx, device.Options{Secrets: sec})
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	var oe *omnillm.Error
	if _, err := d2.Models(ctx); !errors.As(err, &oe) || oe.Msg != omnillm.MsgDeviceRevoked {
		t.Fatalf("after leaving: %v", err)
	}
	if err := d2.Leave(ctx); err != nil {
		t.Fatalf("leaving twice: %v", err)
	}
}
