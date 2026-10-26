package console

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/relay"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

const (
	addr      = "127.0.0.1:8770"
	origin    = "http://127.0.0.1:8770"
	testKey   = "sk-CONSOLE-TEST-KEY-0123456789abcdef"
	hubTok    = "console-test-hub-token-0123456789abcdef0123"
	replyText = "console-reply-ok"
)

func init() { provider.AllowLoopbackForTests() }

func fakeUpstream(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testKey {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/models" {
			fmt.Fprint(w, `{"data":[{"id":"m1"}]}`)
			return
		}
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", replyText)
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2}}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

type rig struct {
	h       *hub.Hub
	s       *Server
	handler http.Handler
	relayWS string
	up      *httptest.Server
	cookie  string
	secret  string
}

func newRig(t *testing.T, withRelay bool) *rig {
	t.Helper()
	r := &rig{up: fakeUpstream(t)}
	opts := hub.Options{Secrets: secret.NewMemory(), Store: store.NewMemory()}
	if withRelay {
		rs := httptest.NewServer(relay.New(relay.Options{HubTokenHashes: [][32]byte{sha256.Sum256([]byte(hubTok))}}))
		t.Cleanup(rs.Close)
		r.relayWS = "ws" + strings.TrimPrefix(rs.URL, "http")
		opts.RelayURL, opts.RelayToken = r.relayWS, hubTok
	}
	h, err := hub.New(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Stop(context.Background()) })
	if withRelay {
		for i := 0; i < 200 && !h.RelayUp(); i++ {
			time.Sleep(10 * time.Millisecond)
		}
	}
	s, err := New(h, addr)
	if err != nil {
		t.Fatal(err)
	}
	r.h, r.s, r.handler = h, s, s.Handler()
	return r
}

type req struct {
	method, path, body string
	host, origin       string
	header             bool
	cookie             string
	secret             string
	fetchSite          string
}

func (r *rig) do(q req) *httptest.ResponseRecorder {
	if q.host == "" {
		q.host = addr
	}
	hr := httptest.NewRequest(q.method, "http://"+addr+q.path, strings.NewReader(q.body))
	hr.Host = q.host
	if q.origin != "" {
		hr.Header.Set("Origin", q.origin)
	}
	if q.header {
		hr.Header.Set(headerName, "1")
	}
	if q.cookie != "" {
		hr.Header.Set("Cookie", cookieName+"="+q.cookie)
		if q.secret == "" {
			q.secret = r.secret
		}
	}
	if q.secret != "" && q.secret != "-" {
		hr.Header.Set(secretHeader, q.secret)
	}
	if q.fetchSite != "" {
		hr.Header.Set("Sec-Fetch-Site", q.fetchSite)
	}
	w := httptest.NewRecorder()
	r.handler.ServeHTTP(w, hr)
	return w
}

// login performs a full login and returns the session cookie value.
func (r *rig) login(t *testing.T) string {
	t.Helper()
	u := r.s.LoginURL()
	tok := u[strings.Index(u, "#login=")+len("#login="):]
	w := r.do(req{method: "POST", path: "/console/api/login", body: `{"token":"` + tok + `"}`, origin: origin, header: true})
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	var out map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	r.secret = out["session"]
	for _, c := range w.Result().Cookies() {
		if sessionCookie(c.Name) {
			r.cookie = c.Value
			return c.Value
		}
	}
	t.Fatal("no cookie")
	return ""
}

func (r *rig) api(t *testing.T, method, path, body string) (int, map[string]any) {
	t.Helper()
	w := r.do(req{method: method, path: path, body: body, origin: origin, header: true, cookie: r.cookie})
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestOnlyLoopback(t *testing.T) {
	if _, err := New(nil, "0.0.0.0:8770"); err == nil {
		t.Fatal("console accepted a non-loopback address")
	}
	if _, err := New(nil, "192.168.1.2:8770"); err == nil {
		t.Fatal("console accepted a LAN address")
	}
}

func TestSecurityHeaders(t *testing.T) {
	r := newRig(t, false)
	w := r.do(req{method: "GET", path: "/console/"})
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	csp := w.Header().Get("Content-Security-Policy")
	for _, must := range []string{"default-src 'none'", "script-src 'self'", "frame-ancestors 'none'", "base-uri 'none'"} {
		if !strings.Contains(csp, must) {
			t.Fatalf("CSP lacks %q: %s", must, csp)
		}
	}
	if strings.Contains(csp, "unsafe") {
		t.Fatalf("CSP allows unsafe sources: %s", csp)
	}
	for k, v := range map[string]string{"X-Frame-Options": "DENY", "X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer", "Cache-Control": "no-store"} {
		if w.Header().Get(k) != v {
			t.Fatalf("%s = %q", k, w.Header().Get(k))
		}
	}
}

func TestHostAndOriginChecks(t *testing.T) {
	r := newRig(t, false)
	sid := r.login(t)
	cases := []struct {
		name string
		q    req
		want int
	}{
		{"rebinding host", req{method: "GET", path: "/console/api/status", host: "evil.example:8770", origin: origin, header: true, cookie: sid}, 421},
		{"other port host", req{method: "GET", path: "/console/api/status", host: "127.0.0.1:9999", origin: origin, header: true, cookie: sid}, 421},
		{"no origin", req{method: "GET", path: "/console/api/status", header: true, cookie: sid}, 403},
		{"foreign origin", req{method: "POST", path: "/console/api/devices/remove", body: `{"id":"x"}`, origin: "http://127.0.0.1:3000", header: true, cookie: sid}, 403},
		{"null origin", req{method: "POST", path: "/console/api/apps/add", body: `{"name":"x"}`, origin: "null", header: true, cookie: sid}, 403},
		{"no custom header", req{method: "POST", path: "/console/api/apps/add", body: `{"name":"x"}`, origin: origin, cookie: sid}, 403},
		{"cross-site fetch", req{method: "POST", path: "/console/api/apps/add", body: `{"name":"x"}`, origin: origin, header: true, cookie: sid, fetchSite: "cross-site"}, 403},
		{"no session", req{method: "GET", path: "/console/api/status", origin: origin, header: true}, 401},
		{"bad session", req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: strings.Repeat("a", 64)}, 401},
		{"cookie without the page secret (another localhost server saw it)", req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: sid, secret: "-"}, 401},
		{"cookie with a wrong page secret", req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: sid, secret: strings.Repeat("b", 64)}, 401},
		{"good", req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: sid}, 200},
		{"browser same-origin GET without Origin", req{method: "GET", path: "/console/api/status", header: true, cookie: sid, fetchSite: "same-origin"}, 200},
		{"POST without Origin even if same-origin", req{method: "POST", path: "/console/api/apps/add", body: `{"name":"x"}`, header: true, cookie: sid, fetchSite: "same-origin"}, 403},
		{"GET without Origin, cross-site", req{method: "GET", path: "/console/api/status", header: true, cookie: sid, fetchSite: "cross-site"}, 403},
		{"localhost name", req{method: "GET", path: "/console/api/status", host: "localhost:8770", origin: "http://localhost:8770", header: true, cookie: sid}, 200},
	}
	for _, c := range cases {
		if w := r.do(c.q); w.Code != c.want {
			t.Fatalf("%s: %d want %d (%s)", c.name, w.Code, c.want, w.Body)
		}
	}
	// No CORS headers are ever sent.
	w := r.do(req{method: "OPTIONS", path: "/console/api/status", origin: "http://evil.example"})
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS header sent")
	}
}

func TestLoginIsOneTimeAndExpires(t *testing.T) {
	r := newRig(t, false)
	u := r.s.LoginURL()
	if !strings.HasPrefix(u, "http://127.0.0.1:8770/console/#login=") {
		t.Fatalf("login link must carry the token in the fragment: %s", u)
	}
	tok := u[strings.Index(u, "#login=")+7:]
	body := `{"token":"` + tok + `"}`
	w := r.do(req{method: "POST", path: "/console/api/login", body: body, origin: origin, header: true})
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var c *http.Cookie
	for _, x := range w.Result().Cookies() {
		if sessionCookie(x.Name) {
			c = x
		}
	}
	if c == nil || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/console" || len(c.Value) != 64 {
		t.Fatalf("cookie %+v", c)
	}
	if w := r.do(req{method: "POST", path: "/console/api/login", body: body, origin: origin, header: true}); w.Code != 401 {
		t.Fatal("login token worked twice")
	}
	// Expiry.
	u2 := r.s.LoginURL()
	r.s.now = func() time.Time { return time.Now().Add(loginTTL + time.Second) }
	tok2 := u2[strings.Index(u2, "#login=")+7:]
	if w := r.do(req{method: "POST", path: "/console/api/login", body: `{"token":"` + tok2 + `"}`, origin: origin, header: true}); w.Code != 401 {
		t.Fatal("expired login token accepted")
	}
	// Session idle expiry.
	r.s.now = func() time.Time { return time.Now().Add(sessionIdle + time.Minute) }
	if w := r.do(req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: c.Value}); w.Code != 401 {
		t.Fatal("idle session still valid")
	}
	// Login from a foreign origin is refused even with a valid token.
	r.s.now = time.Now
	u3 := r.s.LoginURL()
	tok3 := u3[strings.Index(u3, "#login=")+7:]
	if w := r.do(req{method: "POST", path: "/console/api/login", body: `{"token":"` + tok3 + `"}`, origin: "http://evil.example", header: true}); w.Code != 403 {
		t.Fatal("login from a foreign origin")
	}
}

func TestLogout(t *testing.T) {
	r := newRig(t, false)
	r.login(t)
	if code, _ := r.api(t, "POST", "/console/api/logout", `{}`); code != 200 {
		t.Fatal(code)
	}
	if code, _ := r.api(t, "GET", "/console/api/status", ""); code != 401 {
		t.Fatal("session survived logout")
	}
}

func TestAssetsHaveNoInlineCodeOrHTMLInsertion(t *testing.T) {
	html, _ := assets.ReadFile("assets/index.html")
	js, _ := assets.ReadFile("assets/app.js")
	if regexp.MustCompile(`<script(?:\s[^>]*)?>\s*[^<\s]`).Match(html) {
		t.Fatal("inline script in index.html")
	}
	if regexp.MustCompile(`(?i)\son[a-z]+\s*=`).Match(html) {
		t.Fatal("inline event handler in index.html")
	}
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function"} {
		if strings.Contains(string(js), bad) {
			t.Fatalf("app.js uses %s", bad)
		}
	}
	if strings.Contains(string(js), "localStorage.setItem(\"omnillm.key") || strings.Contains(string(js), "localStorage.setItem(\"omnillm.session") {
		t.Fatal("app.js keeps secrets in persistent browser storage")
	}
}

func TestWizardFlows(t *testing.T) {
	r := newRig(t, false)
	r.login(t)
	// Step 1: connect a provider (key checked first).
	code, out := r.api(t, "POST", "/console/api/providers/add", `{"kind":"openai-compatible","name":"fake","base_url":"`+r.up.URL+`","key":"sk-WRONG-key-000000000000000000"}`)
	if code == 200 {
		t.Fatal("a wrong key was accepted")
	}
	if msg, _ := json.Marshal(out); strings.Contains(string(msg), "WRONG") {
		t.Fatal("the key came back in the error")
	}
	code, out = r.api(t, "POST", "/console/api/providers/add", `{"kind":"openai-compatible","name":"fake","base_url":"`+r.up.URL+`","key":"`+testKey+`"}`)
	if code != 200 {
		t.Fatalf("add provider: %d %v", code, out)
	}
	// Status never contains the key.
	w := r.do(req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: r.cookie})
	if strings.Contains(w.Body.String(), testKey) || !strings.Contains(w.Body.String(), `"key_last4":"cdef"`) {
		t.Fatalf("status: %s", w.Body)
	}
	// Step 3: try it.
	code, out = r.api(t, "POST", "/console/api/test", `{"model":"fake/m1","text":"你好"}`)
	if code != 200 || out["reply"] != replyText {
		t.Fatalf("test: %d %v", code, out)
	}
	// Models: human-only toggle.
	if code, _ := r.api(t, "POST", "/console/api/models/set", `{"id":"fake/m1","human_only":true}`); code != 200 {
		t.Fatal(code)
	}
	// Apps: token shown once, never in status.
	code, out = r.api(t, "POST", "/console/api/apps/add", `{"name":"script"}`)
	tok, _ := out["token"].(string)
	if code != 200 || !strings.HasPrefix(tok, "omni_app_") {
		t.Fatalf("app: %d %v", code, out)
	}
	w = r.do(req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: r.cookie})
	if strings.Contains(w.Body.String(), tok) {
		t.Fatal("app token shown again in status")
	}
	// Pairing without a relay explains itself.
	code, _ = r.api(t, "POST", "/console/api/pair/start", `{}`)
	if code == 200 {
		t.Fatal("pairing started without a relay")
	}
	// Remove the provider: its key is gone.
	if code, _ := r.api(t, "POST", "/console/api/providers/remove", `{"id":"fake"}`); code != 200 {
		t.Fatal(code)
	}
	w = r.do(req{method: "GET", path: "/console/api/status", origin: origin, header: true, cookie: r.cookie})
	if strings.Contains(w.Body.String(), "cdef") {
		t.Fatal("removed provider still listed")
	}
}

func TestPairingThroughTheConsole(t *testing.T) {
	r := newRig(t, true)
	r.login(t)
	if code, out := r.api(t, "POST", "/console/api/providers/add", `{"kind":"openai-compatible","name":"fake","base_url":"`+r.up.URL+`","key":"`+testKey+`"}`); code != 200 {
		t.Fatal(code, out)
	}
	code, out := r.api(t, "POST", "/console/api/pair/start", `{"name":"Phone"}`)
	if code != 200 {
		t.Fatalf("pair start: %d %v", code, out)
	}
	id, _ := out["id"].(string)
	pairText, _ := out["code"].(string)
	qr, _ := out["qr"].(string)
	if !strings.HasPrefix(qr, "data:image/svg+xml;base64,") || !strings.HasPrefix(pairText, "omnillm3-") {
		t.Fatalf("pairing payload %v", out)
	}
	// The page a computer's browser opens: the QR link's origin, without
	// the pairing secret.
	if web, _ := out["web"].(string); web != strings.TrimSuffix(hub.DefaultWebAppURL, "/") {
		t.Fatalf("web %q", web)
	}
	// "Copy link": the same link the QR code holds, secret in the fragment.
	if link, _ := out["link"].(string); !strings.HasPrefix(link, strings.TrimSuffix(hub.DefaultWebAppURL, "/")+"/#pair=") {
		t.Fatalf("link %q", link)
	}
	sas := make(chan string, 1)
	devErr := make(chan error, 1)
	sec := secret.NewMemory()
	go func() {
		_, err := device.Pair(context.Background(), device.Options{Secrets: sec}, pairText, "Phone", func(s string) { sas <- s })
		devErr <- err
	}()
	devCode := <-sas
	var st map[string]any
	for i := 0; i < 100; i++ {
		_, st = r.api(t, "GET", "/console/api/pair/state?id="+id, "")
		if st["state"] == "confirm" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if st["state"] != "confirm" || st["code"] != devCode {
		t.Fatalf("console shows %v, device shows %s", st, devCode)
	}
	// A decision for an unknown pairing is refused.
	if code, _ := r.api(t, "POST", "/console/api/pair/decide", `{"id":"nope","yes":true}`); code != 404 {
		t.Fatal(code)
	}
	if code, _ := r.api(t, "POST", "/console/api/pair/decide", `{"id":"`+id+`","yes":true,"name":"Living room"}`); code != 200 {
		t.Fatal(code)
	}
	if err := <-devErr; err != nil {
		t.Fatalf("device: %v", err)
	}
	for i := 0; i < 100; i++ {
		_, st = r.api(t, "GET", "/console/api/pair/state?id="+id, "")
		if st["state"] == "done" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if st["state"] != "done" {
		t.Fatalf("state %v", st)
	}
	_, status := r.api(t, "GET", "/console/api/status", "")
	if devs, _ := status["devices"].([]any); len(devs) != 1 {
		t.Fatalf("devices %v", status["devices"])
	}
	dev0 := status["devices"].([]any)[0].(map[string]any)
	devID := dev0["ID"].(string)
	// The name the owner typed while confirming.
	if dev0["Name"] != "Living room" || dev0["Online"] != false {
		t.Fatalf("device %v", dev0)
	}
	// Models it may use: one, an unknown one (refused), then all again.
	if code, out := r.api(t, "POST", "/console/api/devices/update", `{"id":"`+devID+`","models":["fake/m1"]}`); code != 200 || fmt.Sprint(out["models"]) != "[fake/m1]" {
		t.Fatal(code, out)
	}
	if code, _ := r.api(t, "POST", "/console/api/devices/update", `{"id":"`+devID+`","models":["fake/zzz"]}`); code == 200 {
		t.Fatal("an unknown model was accepted")
	}
	if code, out := r.api(t, "POST", "/console/api/devices/update", `{"id":"`+devID+`","models":[],"paused":true}`); code != 200 || fmt.Sprint(out["models"]) != "[]" || out["paused"] != true {
		t.Fatal(code, out)
	}
	_, status = r.api(t, "GET", "/console/api/status?range=week", "")
	if d := status["devices"].([]any)[0].(map[string]any); d["Paused"] != true {
		t.Fatalf("pause not in status: %v", d)
	}
	if ms := status["models"].([]any); len(ms) == 0 || ms[0].(map[string]any)["group"] != "key" {
		t.Fatalf("models without their group: %v", ms)
	}
	if _, ok := status["model_usage"].([]any); !ok {
		t.Fatalf("no model usage: %v", status["model_usage"])
	}
	// Rename it and change its daily limit: a number, no limit, and a bad one.
	if code, out := r.api(t, "POST", "/console/api/devices/update", `{"id":"`+devID+`","name":"  Kid's iPad ","budget_usd":2.5}`); code != 200 || out["name"] != "Kid's iPad" || out["budget_micro_usd"] != 2.5e6 {
		t.Fatal(code, out)
	}
	if code, out := r.api(t, "POST", "/console/api/devices/update", `{"id":"`+devID+`","no_limit":true}`); code != 200 || out["budget_micro_usd"] != float64(-1) || out["name"] != "Kid's iPad" {
		t.Fatal(code, out)
	}
	if code, _ := r.api(t, "POST", "/console/api/devices/update", `{"id":"`+devID+`","budget_usd":0}`); code == 200 {
		t.Fatal("a zero limit was accepted")
	}
	if code, _ := r.api(t, "POST", "/console/api/devices/update", `{"id":"nope","name":"x"}`); code == 200 {
		t.Fatal("an unknown device was updated")
	}
	// Remove it from the console.
	if code, _ := r.api(t, "POST", "/console/api/devices/remove", `{"id":"`+devID+`"}`); code != 200 {
		t.Fatal(code)
	}
}

func TestDeniedPairingFromConsole(t *testing.T) {
	r := newRig(t, true)
	r.login(t)
	_, out := r.api(t, "POST", "/console/api/pair/start", `{}`)
	id, _ := out["id"].(string)
	pairText, _ := out["code"].(string)
	done := make(chan error, 1)
	go func() {
		_, err := device.Pair(context.Background(), device.Options{Secrets: secret.NewMemory()}, pairText, "Evil", nil)
		done <- err
	}()
	for i := 0; i < 100; i++ {
		if _, st := r.api(t, "GET", "/console/api/pair/state?id="+id, ""); st["state"] == "confirm" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	r.api(t, "POST", "/console/api/pair/decide", `{"id":"`+id+`","yes":false}`)
	if err := <-done; err == nil {
		t.Fatal("a refused device paired")
	}
	// The page learns it was refused (not that it failed), to say so and
	// offer a new QR code.
	var st map[string]any
	for i := 0; i < 100; i++ {
		if _, st = r.api(t, "GET", "/console/api/pair/state?id="+id, ""); st["state"] == "failed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if st["state"] != "failed" || st["reason"] != "denied" {
		t.Fatalf("refused pairing shows %v", st)
	}
	if devs, _ := r.h.Devices(context.Background()); len(devs) != 0 {
		t.Fatal("refused device saved")
	}
}

func TestBodyLimit(t *testing.T) {
	r := newRig(t, false)
	r.login(t)
	big := `{"name":"` + strings.Repeat("x", maxBody+10) + `"}`
	if code, _ := r.api(t, "POST", "/console/api/apps/add", big); code != 400 {
		t.Fatalf("oversized body: %d", code)
	}
	_ = io.Discard
}

func TestListenSkipsATakenPort(t *testing.T) {
	busy, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	ln, err := Listen(busy.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if ln.Addr().String() == busy.Addr().String() {
		t.Fatal("bound a port another program holds")
	}
	if _, err := Listen("0.0.0.0:8770"); err == nil {
		t.Fatal("listened beyond loopback")
	}
	// The login link follows the port actually bound.
	s, _ := New(nil, ln.Addr().String())
	if !strings.HasPrefix(s.LoginURL(), "http://"+ln.Addr().String()+"/console/#login=") {
		t.Fatal(s.LoginURL())
	}
}

// Opening "add device" again and again (closing the dialog in between)
// must not use up the hub's pairing slots: a new pairing ends the old one.
func TestRepeatedPairStartsDoNotFillSlots(t *testing.T) {
	r := newRig(t, true)
	r.login(t)
	var first string
	for i := 0; i < 12; i++ {
		code, out := r.api(t, "POST", "/console/api/pair/start", `{}`)
		if code != 200 {
			t.Fatalf("start %d: %d %v", i, code, out)
		}
		if i == 0 {
			first, _ = out["id"].(string)
		}
	}
	var st map[string]any
	for i := 0; i < 100; i++ {
		_, st = r.api(t, "GET", "/console/api/pair/state?id="+first, "")
		if st["state"] == "failed" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the replaced pairing is still %v", st["state"])
}

func TestClosingAddDeviceEndsThePairing(t *testing.T) {
	r := newRig(t, true)
	r.login(t)
	code, out := r.api(t, "POST", "/console/api/pair/start", `{}`)
	if code != 200 {
		t.Fatal(code, out)
	}
	if code, _ := r.api(t, "POST", "/console/api/pair/cancel", `{"id":"`+out["id"].(string)+`"}`); code != 200 {
		t.Fatal(code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, out["code"].(string), "Late", func(string) {}); err == nil {
		t.Fatal("a closed pairing still paired a device")
	}
}

// Every string the page shows exists in Chinese and in English.
func TestEveryStringInBothLanguages(t *testing.T) {
	js, _ := assets.ReadFile("assets/app.js")
	html, _ := assets.ReadFile("assets/index.html")
	src := string(js)
	zhAt, enAt := strings.Index(src, "\n  zh: {"), strings.Index(src, "\n  en: {")
	end := strings.Index(src[enAt:], "\n  },") + enAt
	if zhAt < 0 || enAt < zhAt || end < enAt {
		t.Fatal("string table not found")
	}
	keyRe := regexp.MustCompile(`(?:^|[\s,{])([A-Za-z0-9]+): "`)
	keys := func(block string) map[string]bool {
		m := map[string]bool{}
		for _, k := range keyRe.FindAllStringSubmatch(block, -1) {
			m[k[1]] = true
		}
		return m
	}
	zh, en := keys(src[zhAt:enAt]), keys(src[enAt:end])
	for k := range zh {
		if !en[k] {
			t.Errorf("%s has no English", k)
		}
	}
	for k := range en {
		if !zh[k] {
			t.Errorf("%s has no Chinese", k)
		}
	}
	// Keys used by the page itself: data-t / data-tip in the HTML, t("…") in the script.
	used := regexp.MustCompile(`data-(?:t|tip)="([A-Za-z0-9]+)"`).FindAllStringSubmatch(string(html), -1)
	used = append(used, regexp.MustCompile(`\bt\("([A-Za-z0-9]+)"`).FindAllStringSubmatch(src, -1)...)
	for _, k := range used {
		if !zh[k[1]] || !en[k[1]] {
			t.Errorf("%s is shown but has no text", k[1])
		}
	}
}

// A device that went away while the owner was asked is not "refused".
func TestPairReasonDeviceLeft(t *testing.T) {
	if got := pairReason(hub.ErrPairingDeviceLeft); got != "left" {
		t.Fatalf("device left: %q", got)
	}
	if got := pairReason(&omnillm.Error{Code: omnillm.CodePairingDenied, Msg: omnillm.MsgPairingDenied}); got != "denied" {
		t.Fatalf("denied: %q", got)
	}
}

// The console's 中文 / English is the hub's choice, which the menu bar reads
// too; one made elsewhere shows in the status the page polls.
func TestLanguageIsSharedWithTheMenuBar(t *testing.T) {
	r := newRig(t, false)
	sid := r.login(t)
	w := r.do(req{method: "POST", path: "/console/api/lang", body: `{"lang":"zh"}`, origin: origin, header: true, cookie: sid})
	if w.Code != 200 {
		t.Fatalf("set: %d %s", w.Code, w.Body)
	}
	if l := r.h.UILang(context.Background()); l != "zh" {
		t.Fatalf("hub keeps %q", l)
	}
	if err := r.h.SetUILang(context.Background(), "en"); err != nil { // the menu bar's switch
		t.Fatal(err)
	}
	for _, path := range []string{"/console/api/status", "/console/api/relay"} {
		w = r.do(req{method: "GET", path: path, origin: origin, header: true, cookie: sid})
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		if out["lang"] != "en" {
			t.Fatalf("%s: lang %v", path, out["lang"])
		}
	}
	w = r.do(req{method: "POST", path: "/console/api/lang", body: `{"lang":"fr"}`, origin: origin, header: true, cookie: sid})
	if w.Code != 400 {
		t.Fatalf("unknown language accepted: %d", w.Code)
	}
}

// The first step's subscription cards: one entry for Claude and one for
// ChatGPT, and without the host's hook the page is told to show the
// command instead of a button that would do nothing.
func TestSubscriptionCards(t *testing.T) {
	// No real program is found or asked (and nothing is connected with
	// the owner's real subscription): an empty home and PATH.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "/usr/bin:/bin")
	r := newRig(t, false)
	sid := r.login(t)
	w := r.do(req{method: "GET", path: "/console/api/subs", origin: origin, header: true, cookie: sid})
	var out struct {
		Subs []struct {
			Kind string `json:"kind"`
		} `json:"subs"`
		CanOpen bool `json:"can_open"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if len(out.Subs) != 3 || out.Subs[0].Kind != "claude-code" || out.Subs[1].Kind != "codex" || out.Subs[2].Kind != "antigravity" || out.CanOpen {
		t.Fatalf("%+v", out)
	}
	w = r.do(req{method: "POST", path: "/console/api/subs/login", body: `{"kind":"claude-code"}`, origin: origin, header: true, cookie: sid})
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("no hook: %d", w.Code)
	}
	var opened string
	r.s.OpenLogin = func(kind string) error { opened = kind; return nil }
	w = r.do(req{method: "POST", path: "/console/api/subs/login", body: `{"kind":"codex"}`, origin: origin, header: true, cookie: sid})
	if w.Code != 200 || opened != "codex" {
		t.Fatalf("hook: %d %q", w.Code, opened)
	}
	w = r.do(req{method: "POST", path: "/console/api/subs/login", body: `{"kind":"../etc"}`, origin: origin, header: true, cookie: sid})
	if w.Code != 400 {
		t.Fatalf("unknown kind: %d", w.Code)
	}
}

// A second console tab (the menu bar opens one each time) leaves the first
// one signed in: each session has a cookie of its own.
func TestTwoTabsStaySignedIn(t *testing.T) {
	r := newRig(t, false)
	login := func() (*http.Cookie, string) {
		u := r.s.LoginURL()
		tok := u[strings.Index(u, "#login=")+len("#login="):]
		w := r.do(req{method: "POST", path: "/console/api/login", body: `{"token":"` + tok + `"}`, origin: origin, header: true})
		var out map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		for _, c := range w.Result().Cookies() {
			if sessionCookie(c.Name) {
				return c, out["session"]
			}
		}
		t.Fatal("no cookie")
		return nil, ""
	}
	c1, s1 := login()
	c2, _ := login()
	if c1.Name == c2.Name {
		t.Fatal("both tabs got the same cookie name")
	}
	// The browser now sends both; the first tab's secret still works.
	hr := httptest.NewRequest("GET", "http://"+addr+"/console/api/status", nil)
	hr.Host = addr
	hr.Header.Set(headerName, "1")
	hr.Header.Set("Sec-Fetch-Site", "same-origin")
	hr.Header.Set("Cookie", c1.Name+"="+c1.Value+"; "+c2.Name+"="+c2.Value)
	hr.Header.Set(secretHeader, s1)
	w := httptest.NewRecorder()
	r.handler.ServeHTTP(w, hr)
	if w.Code != 200 {
		t.Fatalf("first tab signed out: %d", w.Code)
	}
}
