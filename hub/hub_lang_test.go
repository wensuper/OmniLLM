package hub

import (
	"context"
	"testing"

	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

func TestUILangIsKept(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemory()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: st})
	if err != nil {
		t.Fatal(err)
	}
	if l := h.UILang(ctx); l != "" {
		t.Fatalf("not chosen yet, got %q", l)
	}
	for _, l := range []string{"zh", "en", ""} {
		if err := h.SetUILang(ctx, l); err != nil {
			t.Fatal(err)
		}
		if got := h.UILang(ctx); got != l {
			t.Fatalf("set %q, got %q", l, got)
		}
	}
	if err := h.SetUILang(ctx, "fr"); err == nil {
		t.Fatal("an unknown language was kept")
	}
}
