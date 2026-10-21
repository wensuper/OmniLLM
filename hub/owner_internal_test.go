package hub

import (
	"context"
	"errors"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

// The model list a device is answered with: refused while it is paused
// (the phone polls it to see when it is on again), without hidden models,
// with the owner's names.
func TestModelsForDevice(t *testing.T) {
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory()})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"x/a", "x/b"} {
		if err := h.putModel(ctx, Model{ID: id, Provider: "x", Upstream: id[2:], DisplayName: id[2:]}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.modelsFor(ctx, Device{Paused: true}); !errors.Is(err, omnillm.ErrDevicePaused) {
		t.Fatalf("a paused device got its models: %v", err)
	}
	show, name := false, "Mine"
	if err := h.SetModelPref(ctx, "x/a", nil, &show); err != nil {
		t.Fatal(err)
	}
	if err := h.SetModelPref(ctx, "x/b", &name, nil); err != nil {
		t.Fatal(err)
	}
	ms, err := h.modelsFor(ctx, Device{})
	if err != nil || len(ms) != 1 || ms[0].ID != "x/b" || ms[0].DisplayName != "Mine" {
		t.Fatalf("device list %+v %v", ms, err)
	}
}
