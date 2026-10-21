package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/store"
)

// A duplicate is numbered against every other device's name, later ones
// too: with "iPhone", "iPhone" and an owner-chosen "iPhone (2)", the second
// "iPhone" becomes "iPhone (3)" and "iPhone (2)" keeps its name.
func TestDevicesNumberingKeepsAChosenName(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	now := time.Now().UTC()
	for i, x := range [][2]string{{"aaaa", "iPhone"}, {"bbbb", "iPhone"}, {"cccc", "iPhone (2)"}} {
		d := hub.Device{ID: x[0], Name: x[1], Created: now.Add(time.Duration(i) * time.Minute)}
		if err := store.PutJSON(ctx, e.hubSt, "device/"+x[0], d); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]string{"aaaa": "iPhone", "bbbb": "iPhone (3)", "cccc": "iPhone (2)"}
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
	for id, name := range want {
		var rec hub.Device
		if err := store.GetJSON(ctx, e.hubSt, "device/"+id, &rec); err != nil || rec.Name != name {
			t.Fatalf("stored %s: %+v %v", id, rec, err)
		}
	}
}
