package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

// A device paired with the same name while the owner was still looking at
// another pairing's question does not leave two "Test Phone"s: the name is
// numbered again right before the second one is stored, and the device is
// told the name it got.
func TestPairingNameRecheckedBeforeStoring(t *testing.T) {
	e := setup(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	asked := make(chan string, 1)
	release := make(chan struct{})
	p, err := e.hub.Pair(ctx, hub.PairOptions{Confirm: func(ctx context.Context, r hub.PairRequest) (bool, error) {
		asked <- r.DeviceName
		select {
		case <-release:
			return true, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		info device.Info
		err  error
	}
	slow := make(chan result, 1)
	go func() {
		info, err := device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "Test Phone", nil)
		slow <- result{info, err}
	}()
	select {
	case n := <-asked:
		if n != "Test Phone" {
			t.Fatalf("proposed name %q", n)
		}
	case <-ctx.Done():
		t.Fatal("owner was never asked")
	}

	// Meanwhile a second phone with the same name is paired all the way.
	if _, _, err := e.pair(func(a, b string) bool { return a == b }, hub.PairOptions{}); err != nil {
		t.Fatalf("second pairing: %v", err)
	}
	close(release)
	r := <-slow
	if r.err != nil {
		t.Fatalf("first pairing: %v", r.err)
	}
	if r.info.Name != "Test Phone (2)" {
		t.Fatalf("device was told %q", r.info.Name)
	}
	devs, err := e.hub.Devices(ctx)
	if err != nil || len(devs) != 2 {
		t.Fatalf("devices %+v %v", devs, err)
	}
	if devs[0].Name == devs[1].Name {
		t.Fatalf("two devices share the name %q", devs[0].Name)
	}
}

// Two device records that share a name from before names were unique (or
// from two hubs' records merged) are told apart once: the later one is
// renamed and stays renamed; the last-seen time is not written into it.
func TestDevicesRenamesDuplicatesApart(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	now := time.Now().UTC()
	for i, id := range []string{"aaaa", "bbbb", "cccc"} {
		name := "iPhone"
		if id == "cccc" {
			name = "iPad"
		}
		d := hub.Device{ID: id, Name: name, Created: now.Add(time.Duration(i) * time.Minute)}
		if err := store.PutJSON(ctx, e.hubSt, "device/"+id, d); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.PutJSON(ctx, e.hubSt, "devicelast/bbbb", now); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"aaaa": "iPhone", "bbbb": "iPhone (2)", "cccc": "iPad"}
	for round := 0; round < 2; round++ {
		devs, err := e.hub.Devices(ctx)
		if err != nil || len(devs) != 3 {
			t.Fatalf("devices %+v %v", devs, err)
		}
		for _, d := range devs {
			if d.Name != want[d.ID] {
				t.Fatalf("round %d: %s is %q, want %q", round, d.ID, d.Name, want[d.ID])
			}
		}
	}
	var rec hub.Device
	if err := store.GetJSON(ctx, e.hubSt, "device/bbbb", &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Name != "iPhone (2)" || !rec.LastSeen.IsZero() {
		t.Fatalf("stored record %+v", rec)
	}
}
