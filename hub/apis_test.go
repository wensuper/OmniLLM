package hub_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/localapi"
	"github.com/wensuper/OmniLLM/secret"
)

func call(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "http://127.0.0.1:1"+path, strings.NewReader(body))
	r.Host = "127.0.0.1:1"
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

const body = `{"model":"fake/m1","messages":[{"role":"user","content":"hello"}]}`

func TestHubLocalAPIWithAppTokens(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	app, tok, _ := e.hub.AddApp(ctx, "script", []string{"fake/m1"}, 0)
	h := localapi.Handler(localapi.HubBackend{Hub: e.hub})
	if w := call(t, h, "GET", "/v1/models", tok, ""); w.Code != 200 || strings.Contains(w.Body.String(), "fake/m2") || !strings.Contains(w.Body.String(), `"locality":"remote"`) {
		t.Fatalf("models: %d %s", w.Code, w.Body)
	}
	if w := call(t, h, "POST", "/v1/chat/completions", tok, body); w.Code != 200 || !strings.Contains(w.Body.String(), sentinelReply) {
		t.Fatalf("chat: %d %s", w.Code, w.Body)
	}
	if w := call(t, h, "POST", "/v1/chat/completions", tok, strings.Replace(body, "m1", "m2", 1)); w.Code != 403 || !strings.Contains(w.Body.String(), "model_not_allowed") {
		t.Fatalf("model outside the token's list: %d %s", w.Code, w.Body)
	}
	if err := e.hub.RevokeApp(ctx, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := call(t, h, "POST", "/v1/chat/completions", tok, body); w.Code != 401 {
		t.Fatalf("revoked token still works: %d", w.Code)
	}
}

func TestConnectLocalAPI(t *testing.T) {
	e := setup(t, nil)
	_, sec := e.pairOK()
	ctx := context.Background()
	d, _ := device.Open(ctx, device.Options{Secrets: sec})
	defer d.Close()
	_ = e.hub.SetModel(ctx, "fake/m2", hubModelHumanOnly())
	h := localapi.Handler(localapi.NewDeviceBackend(d, "omni_local_right"))
	if w := call(t, h, "POST", "/v1/chat/completions", "omni_local_wrong", body); w.Code != 401 {
		t.Fatalf("wrong local token: %d", w.Code)
	}
	if w := call(t, h, "POST", "/v1/chat/completions", "", body); w.Code != 401 {
		t.Fatalf("no token: %d", w.Code)
	}
	if w := call(t, h, "POST", "/v1/chat/completions", "omni_local_right", body); w.Code != 200 {
		t.Fatalf("right token: %d %s", w.Code, w.Body)
	}
	// Calls through connect are Background: a human-only model refuses.
	if w := call(t, h, "POST", "/v1/chat/completions", "omni_local_right", strings.Replace(body, "m1", "m2", 1)); w.Code != 403 {
		t.Fatalf("connect reached a human-only model: %d %s", w.Code, w.Body)
	}
	if w := call(t, h, "GET", "/v1/models", "omni_local_right", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"locality":"remote"`) {
		t.Fatalf("models via connect: %s", w.Body)
	}
	if !device.Paired(ctx, sec) {
		t.Fatal("paired")
	}
	if err := device.Forget(ctx, sec); err != nil || device.Paired(ctx, sec) {
		t.Fatal("forget")
	}
	if _, err := device.Open(ctx, device.Options{Secrets: sec}); err == nil {
		t.Fatal("opened a forgotten pairing")
	}
	_ = secret.NewMemory
}

func TestRemoveProviderDeletesTheKey(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	if err := e.hub.RemoveProvider(ctx, "fake"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.hubSec.Get(ctx, "provider/fake/key"); err == nil {
		t.Fatal("key left behind")
	}
	if ms, _ := e.hub.Models(ctx); len(ms) != 0 {
		t.Fatal("models left behind", ms)
	}
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("removed provider still answers: %v", err)
	}
	if err := e.hub.RemoveModel(ctx, "fake/m1"); err != nil {
		t.Fatal(err)
	}
	if len(e.hub.PublicKey()) != 32 {
		t.Fatal("public key")
	}
	if a, err := e.hub.Audit(ctx, ""); err != nil || a == nil {
		_ = a
	}
}

func TestSwitchedOffSourceServesNoOneAndKeepsItsKey(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	if err := e.hub.SetProviderOff(ctx, "fake", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("a switched-off source answered: %v", err)
	}
	if _, err := e.hubSec.Get(ctx, "provider/fake/key"); err != nil {
		t.Fatal("switching off deleted the key")
	}
	ps, _ := e.hub.Providers(ctx)
	if len(ps) != 1 || !ps[0].Off {
		t.Fatalf("providers don't say it is off: %+v", ps)
	}
	if vs, _ := e.hub.OwnerModels(ctx); len(vs) == 0 || !vs[0].SourceOff {
		t.Fatalf("the owner's list doesn't mark its models: %+v", vs)
	}
	if err := e.hub.SetProviderOff(ctx, "fake", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("on again, it still refuses: %v", err)
	}
	if err := e.hub.SetProviderOff(ctx, "nosuch", true); err == nil {
		t.Fatal("switched off a source that isn't connected")
	}
}
