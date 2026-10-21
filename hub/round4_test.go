package hub_test

import (
	"context"
	"testing"
	"time"
)

// The models reply carries the device's name as the Mac has it now, so a
// rename on the Mac reaches the device the next time it asks.
func TestModelsReplyCarriesDeviceName(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := d.Models(ctx); err != nil {
		t.Fatal(err)
	}
	if d.Name() != "Test Phone" {
		t.Fatalf("name after pairing: %q", d.Name())
	}
	devs, err := e.hub.Devices(ctx)
	if err != nil || len(devs) != 1 {
		t.Fatalf("devices %v %v", devs, err)
	}
	name := "Kitchen iPad"
	if _, err := e.hub.UpdateDevice(ctx, devs[0].ID, &name, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Models(ctx); err != nil {
		t.Fatal(err)
	}
	if d.Name() != name {
		t.Fatalf("name after the rename: %q", d.Name())
	}
}

// Saving an app's other settings keeps a model on its list that became
// chat-only meanwhile (the owner didn't switch it off), while a chat-only
// model can still not be newly given to an app (using one is refused at
// every call, see complete.go).
func TestAppKeepsModelsItCannotUseNow(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	a, _, err := e.hub.AddApp(ctx, "script", []string{"fake/m1"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.hub.SetModel(ctx, "fake/m1", hubModelHumanOnly()); err != nil {
		t.Fatal(err)
	}
	name, ms := "script-2", []string{"fake/m1"}
	got, err := e.hub.UpdateApp(ctx, a.ID, &name, nil, nil, &ms)
	if err != nil || len(got.Models) != 1 || got.Models[0] != "fake/m1" {
		t.Fatalf("kept list: %+v %v", got.Models, err)
	}
	if err := e.hub.SetModel(ctx, "fake/m2", hubModelHumanOnly()); err != nil {
		t.Fatal(err)
	}
	ms = []string{"fake/m1", "fake/m2"}
	if _, err := e.hub.UpdateApp(ctx, a.ID, nil, nil, nil, &ms); err == nil {
		t.Fatal("a chat-only model was newly given to an app")
	}
}
