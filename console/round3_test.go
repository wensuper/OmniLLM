package console

import (
	"strings"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
)

// "Check again" asks the hub to connect what is signed in on this Mac and
// names what it connected; with nothing to find it answers an empty list,
// not an error, and every name it gives is a connected source.
func TestDetectProviders(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir) // no claude or codex program here
	t.Setenv("HOME", dir)
	t.Setenv("CODEX_HOME", dir)
	r := newRig(t, false)
	r.login(t)
	code, out := r.api(t, "POST", "/console/api/providers/detect", `{}`)
	if code != 200 {
		t.Fatalf("detect: %d %v", code, out)
	}
	names, ok := out["connected"].([]any)
	if !ok {
		t.Fatalf("no list of what was connected: %v", out)
	}
	_, status := r.api(t, "GET", "/console/api/status", "")
	ps, _ := status["providers"].([]any)
	if len(ps) < len(names) {
		t.Fatalf("connected %v, but sources are %v", names, ps)
	}
	for _, n := range names {
		if strings.Contains(strings.ToLower(n.(string)), "claude") || strings.Contains(strings.ToLower(n.(string)), "codex") {
			t.Fatalf("found a subscription that is not there: %v", names)
		}
	}
	// Only the console's own page may ask.
	w := r.do(req{method: "POST", path: "/console/api/providers/detect", body: `{}`, origin: "http://evil.example", header: true, cookie: r.cookie})
	if w.Code != 403 {
		t.Fatalf("another origin: %d", w.Code)
	}
}

// A pairing the Mac's relay link ended is not "the device left".
func TestPairReasonRelay(t *testing.T) {
	if got := pairReason(&omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable}); got != "relay" {
		t.Fatalf("relay: %q", got)
	}
}

// The console asks nothing in browser windows: every question is in the
// page (dialog foot, a row, or a field in place).
func TestNoBrowserDialogs(t *testing.T) {
	js, _ := assets.ReadFile("assets/app.js")
	for _, bad := range []string{"window.confirm", "window.prompt", "window.alert", "confirm(", "prompt(", "alert("} {
		if strings.Contains(string(js), bad) {
			t.Fatalf("app.js uses %s", bad)
		}
	}
}
