package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/wensuper/OmniLLM/hub"
)

// The models reply carries the owner's language choice, so the phone's
// page speaks the Mac's language; not chosen, the device hears "".
func TestModelsReplyCarriesOwnerLanguage(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := d.Models(ctx); err != nil {
		t.Fatal(err)
	}
	if d.Lang() != "" {
		t.Fatalf("language before a choice: %q", d.Lang())
	}
	for _, l := range []string{"zh", "en", ""} {
		if err := e.hub.SetUILang(ctx, l); err != nil {
			t.Fatal(err)
		}
		if _, err := d.Models(ctx); err != nil {
			t.Fatal(err)
		}
		if d.Lang() != l {
			t.Fatalf("language after choosing %q: %q", l, d.Lang())
		}
	}
}

// Not chosen by the owner, the host's own language goes to devices; the
// owner's choice wins over it.
func TestModelsReplyFallsBackToHostLanguage(t *testing.T) {
	e := setup(t, func(o *hub.Options) { o.Lang = "zh" })
	d, _ := e.pairOK()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := d.Models(ctx); err != nil {
		t.Fatal(err)
	}
	if d.Lang() != "zh" {
		t.Fatalf("host language: %q", d.Lang())
	}
	if err := e.hub.SetUILang(ctx, "en"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Models(ctx); err != nil {
		t.Fatal(err)
	}
	if d.Lang() != "en" {
		t.Fatalf("owner's choice: %q", d.Lang())
	}
}
