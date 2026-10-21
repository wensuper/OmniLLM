// Package console is the hub's local web console (design §10): first-run
// wizard, pairing with a QR code, devices, apps and usage. It serves only
// on a loopback address and is reached through a one-time login link that
// the owner gets from the running hub over the owner-only control socket
// ("omnillm open"), so the OS login is the only credential.
//
// Every API request must carry the exact Host and Origin of the console and
// a custom header; the session cookie is HttpOnly and SameSite=Strict; the
// Content-Security-Policy forbids inline scripts; text that came from a
// device (its name) is only ever set as text, never as HTML.
package console

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/internal/qrcode"
)

//go:embed assets
var assets embed.FS

// Icon is the OmniLLM mark (SVG), for pages outside the console that
// should carry it too (the page `omnillm open` writes).
func Icon() []byte { b, _ := assets.ReadFile("assets/icon.svg"); return b }

const (
	cookieName   = "omnillm_console"
	headerName   = "X-OmniLLM-Console"
	secretHeader = "X-OmniLLM-Session"
	loginTTL     = 2 * time.Minute
	sessionIdle  = 12 * time.Hour
	sessionMax   = 7 * 24 * time.Hour
	maxBody      = 64 << 10
	maxSessions  = 16
	maxLoginsOut = 8
)

// Server is the console for one hub on one loopback address.
type Server struct {
	hub  *hub.Hub
	addr string // host:port as the browser sees it
	now  func() time.Time

	mu       sync.Mutex
	logins   map[string]time.Time // one-time login tokens → expiry
	sessions map[string]*session
	pairs    map[string]*pairState
}

// session pairs the cookie with a second secret the page keeps in its own
// origin's sessionStorage and sends as a header. Browsers share cookies
// across localhost ports; they do not share sessionStorage, so another
// local web server that sees the cookie still cannot use it.
type session struct {
	secret  string
	created time.Time
	last    time.Time
}

// New makes a console. addr must be the loopback address it is served on.
func New(h *hub.Hub, addr string) (*Server, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return nil, errors.New("console: must be served on 127.0.0.1")
	}
	return &Server{hub: h, addr: addr, now: time.Now, logins: map[string]time.Time{},
		sessions: map[string]*session{}, pairs: map[string]*pairState{}}, nil
}

func token() string { return hex.EncodeToString(e2e.RandomBytes(32)) }

// LoginURL makes a one-time link. Give it only to the owner (the control
// socket does that). The token is in the fragment, so it is never sent in a
// request line, a log or a Referer.
func (s *Server) LoginURL() string {
	t := token()
	s.mu.Lock()
	now := s.now()
	for k, exp := range s.logins {
		if now.After(exp) {
			delete(s.logins, k)
		}
	}
	for len(s.logins) >= maxLoginsOut {
		// Drop the link that expires first, not all of them.
		var oldest string
		for k, exp := range s.logins {
			if oldest == "" || exp.Before(s.logins[oldest]) {
				oldest = k
			}
		}
		delete(s.logins, oldest)
	}
	s.logins[t] = now.Add(loginTTL)
	s.mu.Unlock()
	return "http://" + s.addr + "/console/#login=" + t
}

func (s *Server) origins() []string {
	_, port, _ := net.SplitHostPort(s.addr)
	return []string{"http://" + s.addr, "http://127.0.0.1:" + port, "http://localhost:" + port}
}

func (s *Server) hostOK(h string) bool {
	_, port, _ := net.SplitHostPort(s.addr)
	return h == s.addr || h == "127.0.0.1:"+port || h == "localhost:"+port
}

func (s *Server) originOK(o string) bool {
	for _, x := range s.origins() {
		if subtle.ConstantTimeCompare([]byte(o), []byte(x)) == 1 {
			return true
		}
	}
	return false
}

// sameOrigin: the request comes from the console's own page. Browsers send
// Origin on POST but not on a same-origin GET; for those, the browser's own
// Sec-Fetch-Site header (which a page cannot set) must say same-origin.
func (s *Server) sameOrigin(r *http.Request) bool {
	if o := r.Header.Get("Origin"); o != "" {
		return s.originOK(o)
	}
	return r.Method == http.MethodGet && r.Header.Get("Sec-Fetch-Site") == "same-origin"
}

// Handler serves the console.
func (s *Server) Handler() http.Handler {
	static, _ := fs.Sub(assets, "assets")
	files := http.FileServer(http.FS(static))
	mux := http.NewServeMux()
	mux.Handle("GET /console/", http.StripPrefix("/console/", files))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/console/", http.StatusFound) })
	mux.HandleFunc("POST /console/api/login", s.login)
	api := func(pattern string, f func(http.ResponseWriter, *http.Request)) {
		mux.HandleFunc(pattern, s.authed(f))
	}
	api("GET /console/api/status", s.status)
	api("GET /console/api/relay", s.relay)
	api("GET /console/api/ollama", s.ollama)
	api("POST /console/api/providers/add", s.addProvider)
	api("POST /console/api/providers/remove", s.removeProvider)
	api("POST /console/api/providers/detect", s.detectProviders)
	api("POST /console/api/models/set", s.setModel)
	api("POST /console/api/devices/remove", s.removeDevice)
	api("POST /console/api/devices/update", s.updateDevice)
	api("POST /console/api/apps/add", s.addApp)
	api("POST /console/api/apps/remove", s.removeApp)
	api("POST /console/api/apps/update", s.updateApp)
	api("POST /console/api/daily-total", s.setDailyTotal)
	api("POST /console/api/pair/start", s.pairStart)
	api("GET /console/api/pair/state", s.pairGet)
	api("POST /console/api/pair/decide", s.pairDecide)
	api("POST /console/api/pair/cancel", s.pairCancel)
	api("POST /console/api/test", s.test)
	api("POST /console/api/region/confirm", s.regionConfirm)
	api("POST /console/api/logout", s.logout)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; "+
			"connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Cache-Control", "no-store")
		if !s.hostOK(r.Host) {
			http.Error(w, "use 127.0.0.1", http.StatusMisdirectedRequest)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/console/api/") {
			// API: same origin only, and a header no other site can set
			// without a CORS preflight (which is never answered).
			if !s.sameOrigin(r) || r.Header.Get(headerName) != "1" {
				writeErr(w, r, http.StatusForbidden, &omnillm.Error{Code: omnillm.CodeForbiddenOrigin, Msg: omnillm.MsgBrowserRefused})
				return
			}
			if sfs := r.Header.Get("Sec-Fetch-Site"); sfs != "" && sfs != "same-origin" {
				writeErr(w, r, http.StatusForbidden, &omnillm.Error{Code: omnillm.CodeForbiddenOrigin, Msg: omnillm.MsgBrowserRefused})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		}
		mux.ServeHTTP(w, r)
	})
}

// Serve listens on addr (loopback only) until ctx ends.
func (s *Server) Serve(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	return s.ServeListener(ctx, ln)
}

// Listen binds the console on addr, or, if that port is taken by another
// program, on the next free one of the following 20 ports. The console's
// address (and every login link) follows the port actually bound, so a
// link never points at someone else's server.
func Listen(addr string) (net.Listener, error) {
	host, portS, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return nil, errors.New("console: must be served on 127.0.0.1")
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return nil, err
	}
	var last error
	for p := port; p <= port+20 && p < 65536; p++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(p)))
		if err == nil {
			return ln, nil
		}
		last = err
	}
	return nil, last
}

// ServeListener serves on an already bound listener until ctx ends.
func (s *Server) ServeListener(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: time.Minute,
		WriteTimeout: 2 * time.Minute, IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 32 << 10}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
			return
		}
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ---------- auth ----------

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.originOK(r.Header.Get("Origin")) || r.Header.Get(headerName) != "1" {
		writeErr(w, r, http.StatusForbidden, &omnillm.Error{Code: omnillm.CodeForbiddenOrigin, Msg: omnillm.MsgBrowserRefused})
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Token) != 64 {
		writeErr(w, r, http.StatusUnauthorized, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: msgLoginExpired})
		return
	}
	s.mu.Lock()
	now := s.now()
	ok := false
	for t, exp := range s.logins {
		if subtle.ConstantTimeCompare([]byte(t), []byte(in.Token)) == 1 && now.Before(exp) {
			ok = true
			delete(s.logins, t) // one use only
		}
	}
	var sid, sec string
	if ok {
		s.pruneSessionsLocked(now)
		for len(s.sessions) >= maxSessions {
			var oldest string
			for k, v := range s.sessions {
				if oldest == "" || v.last.Before(s.sessions[oldest].last) {
					oldest = k
				}
			}
			delete(s.sessions, oldest)
		}
		sid, sec = token(), token()
		s.sessions[sid] = &session{secret: sec, created: now, last: now}
	}
	s.mu.Unlock()
	if !ok {
		writeErr(w, r, http.StatusUnauthorized, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: msgLoginExpired})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: sid, Path: "/console", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, map[string]string{"session": sec})
}

func (s *Server) pruneSessionsLocked(now time.Time) {
	for k, v := range s.sessions {
		if now.Sub(v.last) >= sessionIdle || now.Sub(v.created) >= sessionMax {
			delete(s.sessions, k)
		}
	}
}

func (s *Server) authed(f func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sec := r.Header.Get(secretHeader)
		ok := false
		// Any cookie with our name (another local server may have set one
		// of its own) is tried; one must match together with the secret.
		for _, c := range r.Cookies() {
			if c.Name != cookieName || len(c.Value) != 64 || len(sec) != 64 {
				continue
			}
			s.mu.Lock()
			now := s.now()
			s.pruneSessionsLocked(now)
			for sid, ss := range s.sessions {
				if subtle.ConstantTimeCompare([]byte(sid), []byte(c.Value)) == 1 &&
					subtle.ConstantTimeCompare([]byte(ss.secret), []byte(sec)) == 1 {
					ok = true
					ss.last = now
				}
			}
			s.mu.Unlock()
		}
		if !ok {
			writeErr(w, r, http.StatusUnauthorized, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: msgLoginExpired})
			return
		}
		f(w, r)
	}
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/console", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, map[string]bool{"ok": true})
}

// ---------- helpers ----------

const msgLoginExpired = "console_login_expired"

func lang(r *http.Request) string {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("X-OmniLLM-Lang")), "zh") {
		return "zh"
	}
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Accept-Language")), "zh") {
		return "zh"
	}
	return "en"
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, r *http.Request, status int, err error) {
	e := omnillm.AsError(err)
	msg := omnillm.Describe(lang(r), e)
	if e.Msg == msgLoginExpired {
		msg = map[string]string{"zh": "登录已失效。点菜单栏里的 OmniLLM 图标，选“打开控制台”，或者在终端运行 omnillm open。",
			"en": "This login has expired. Click the OmniLLM icon in the menu bar and choose “Open the console”, or run omnillm open in Terminal."}[lang(r)]
	}
	if status == 0 {
		status = omnillm.HTTPStatus(e.Code)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": e.Code, "message": msg}})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, r, http.StatusBadRequest, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_messages"})
		return false
	}
	return true
}

// ---------- API ----------

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ps, _ := s.hub.Providers(ctx)
	if len(ps) == 0 {
		// Logged in to Claude Code or Codex since the hub started: connect
		// it now, so the first step is already done.
		for _, cfg := range s.hub.DetectSubscriptions(ctx) {
			_, _ = s.hub.AddProvider(ctx, cfg, "")
		}
		ps, _ = s.hub.Providers(ctx)
	}
	ms, _ := s.hub.OwnerModels(ctx)
	devs, _ := s.hub.Devices(ctx)
	apps, _ := s.hub.Apps(ctx)
	usage, _ := s.hub.Usage(ctx, "")
	// The Usage tab's range: today, yesterday or the last 7 days, per
	// device or app and per model.
	first, last := s.hub.Day(0), s.hub.Day(0)
	switch r.URL.Query().Get("range") {
	case "yesterday":
		first, last = s.hub.Day(1), s.hub.Day(1)
	case "week":
		first = s.hub.Day(6)
	}
	rangeUsage, _ := s.hub.UsageBetween(ctx, first, last)
	modelUsage, _ := s.hub.ModelUsageBetween(ctx, first, last)
	confirmed, seen, paused := s.hub.RegionStatus()
	type dev struct {
		ID, Name       string
		Created        time.Time
		LastSeen       time.Time
		BudgetMicroUSD int64
		BackgroundOnly bool
		Models         []string
		Online, Paused bool
	}
	var ds []dev
	for _, d := range devs {
		ds = append(ds, dev{d.ID, d.Name, d.Created, d.LastSeen, d.Budget, d.BackgroundOnly, orEmpty(d.Models), d.Online, d.Paused})
	}
	type app struct {
		ID, Name       string
		Created        time.Time
		BudgetMicroUSD int64
		Paused         bool
		Models         []string
	}
	var as []app
	for _, a := range apps {
		as = append(as, app{a.ID, a.Name, a.Created, a.Budget, a.Paused, orEmpty(a.Models)})
	}
	// Each model with its group for the edit dialog's model list:
	// subscriptions, API keys, or this Mac.
	kinds := map[string]provider.Kind{}
	for _, p := range ps {
		kinds[p.Name] = p.Kind
	}
	type modelRow struct {
		hub.ModelView
		Group string `json:"group"`
	}
	mrs := []modelRow{}
	for _, m := range ms {
		g := "key"
		switch {
		case kinds[m.Provider].Subscription():
			g = "sub"
		case m.Locality == omnillm.Local:
			g = "local"
		}
		mrs = append(mrs, modelRow{m, g})
	}
	// Empty lists are [] (not null) for the page.
	if ps == nil {
		ps = []hub.ProviderInfo{}
	}
	if ds == nil {
		ds = []dev{}
	}
	if as == nil {
		as = []app{}
	}
	if usage == nil {
		usage = []hub.Spend{}
	}
	if rangeUsage == nil {
		rangeUsage = []hub.Spend{}
	}
	if modelUsage == nil {
		modelUsage = []hub.Spend{}
	}
	writeJSON(w, map[string]any{
		"providers": ps, "models": mrs, "devices": ds, "apps": as, "usage": usage,
		"range_usage": rangeUsage, "model_usage": modelUsage,
		"relay_up": s.hub.RelayUp(), "relay_configured": s.hub.RelayConfigured(),
		"daily_total_micro_usd": s.hub.DailyTotal(),
		"region":                map[string]any{"confirmed": confirmed, "seen": seen, "paused": paused},
	})
}

// relay is the cheap part of status, polled while the hub (re)connects.
func (s *Server) relay(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"relay_up": s.hub.RelayUp(), "relay_configured": s.hub.RelayConfigured()})
}

func (s *Server) ollama(w http.ResponseWriter, r *http.Request) {
	ids, err := s.hub.DetectOllama(r.Context())
	writeJSON(w, map[string]any{"available": err == nil, "models": ids})
}

func (s *Server) addProvider(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		Key     string `json:"key"`
	}
	if !decode(w, r, &in) {
		return
	}
	var cfg provider.Config
	var err error
	if in.Kind == string(provider.OpenAICompatible) {
		cfg = provider.Config{Kind: provider.OpenAICompatible, NeedsKey: true}
	} else if cfg, err = provider.Default(provider.Kind(in.Kind)); err != nil {
		writeErr(w, r, http.StatusBadRequest, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_model"})
		return
	}
	if in.Name != "" {
		cfg.Name = in.Name
	}
	if in.BaseURL != "" {
		cfg.BaseURL = in.BaseURL
	}
	ms, err := s.hub.AddProvider(r.Context(), cfg, in.Key)
	in.Key = ""
	if err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]any{"models": ms})
}

// detectProviders is "Check again": every subscription program logged in
// on this Mac (even one removed before: asking is adding it again) and a
// running Ollama are connected now. connected lists their kinds (the page
// names them in its language); failed, what was found but not connected,
// each with the hub's sentence in the page's language.
func (s *Server) detectProviders(w http.ResponseWriter, r *http.Request) {
	kinds, failures, err := s.hub.DetectNow(r.Context())
	if err != nil {
		writeErr(w, r, 0, err)
		return
	}
	failed := []map[string]string{}
	for _, f := range failures {
		failed = append(failed, map[string]string{"kind": f.Kind, "message": omnillm.Describe(lang(r), f.Err)})
	}
	writeJSON(w, map[string]any{"connected": orEmpty(kinds), "failed": failed})
}

type idBody struct {
	ID string `json:"id"`
}

func (s *Server) removeProvider(w http.ResponseWriter, r *http.Request) {
	var in idBody
	if !decode(w, r, &in) {
		return
	}
	if err := s.hub.RemoveProvider(r.Context(), in.ID); err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) setModel(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID          string   `json:"id"`
		DisplayName *string  `json:"display_name"`
		PriceIn     *float64 `json:"price_in"`
		PriceOut    *float64 `json:"price_out"`
		HumanOnly   *bool    `json:"human_only"`
		// Name is the owner's own name for it ("" goes back to the
		// source's); Shown: offered to devices.
		Name  *string `json:"name"`
		Shown *bool   `json:"shown"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.DisplayName != nil || in.PriceIn != nil || in.PriceOut != nil || in.HumanOnly != nil {
		if err := s.hub.SetModel(r.Context(), in.ID, hub.ModelUpdate{DisplayName: in.DisplayName, PriceIn: in.PriceIn, PriceOut: in.PriceOut, HumanOnly: in.HumanOnly}); err != nil {
			writeErr(w, r, 0, err)
			return
		}
	}
	if in.Name != nil || in.Shown != nil {
		if err := s.hub.SetModelPref(r.Context(), in.ID, in.Name, in.Shown); err != nil {
			writeErr(w, r, 0, err)
			return
		}
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) removeDevice(w http.ResponseWriter, r *http.Request) {
	var in idBody
	if !decode(w, r, &in) {
		return
	}
	if err := s.hub.Revoke(r.Context(), in.ID); err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// updateBody renames a device or app, changes its daily limit, or pauses
// or resumes it; no_limit leaves only the hub's daily total.
type updateBody struct {
	ID        string   `json:"id"`
	Name      *string  `json:"name"`
	BudgetUSD *float64 `json:"budget_usd"`
	NoLimit   bool     `json:"no_limit"`
	Paused    *bool    `json:"paused"`
	// Models it may use; [] is all (models added later too), absent
	// leaves them as they are.
	Models *[]string `json:"models"`
}

func (in updateBody) budget() *int64 {
	var b int64
	switch {
	case in.NoLimit:
		b = hub.NoLimit
	case in.BudgetUSD != nil:
		b = max(int64(math.Round(*in.BudgetUSD*1e6)), 0) // 0 is refused with a sentence
	default:
		return nil
	}
	return &b
}

func (s *Server) updateDevice(w http.ResponseWriter, r *http.Request) {
	var in updateBody
	if !decode(w, r, &in) {
		return
	}
	d, err := s.hub.UpdateDevice(r.Context(), in.ID, in.Name, in.budget(), in.Paused, in.Models)
	if err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]any{"name": d.Name, "budget_micro_usd": d.Budget, "paused": d.Paused, "models": orEmpty(d.Models)})
}

// orEmpty: a model list for the page, [] (all models) rather than null.
func orEmpty(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

func (s *Server) setDailyTotal(w http.ResponseWriter, r *http.Request) {
	var in updateBody
	if !decode(w, r, &in) {
		return
	}
	b := in.budget()
	if b == nil {
		writeErr(w, r, http.StatusBadRequest, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadBudget})
		return
	}
	if err := s.hub.SetDailyTotal(r.Context(), *b); err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]any{"daily_total_micro_usd": *b})
}

func (s *Server) updateApp(w http.ResponseWriter, r *http.Request) {
	var in updateBody
	if !decode(w, r, &in) {
		return
	}
	a, err := s.hub.UpdateApp(r.Context(), in.ID, in.Name, in.budget(), in.Paused, in.Models)
	if err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]any{"name": a.Name, "budget_micro_usd": a.Budget, "paused": a.Paused, "models": orEmpty(a.Models)})
}

func (s *Server) addApp(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	a, tok, err := s.hub.AddApp(r.Context(), in.Name, nil, 0)
	if err != nil {
		writeErr(w, r, 0, err)
		return
	}
	// The token is shown once; it is never stored or shown again.
	writeJSON(w, map[string]any{"id": a.ID, "name": a.Name, "token": tok})
}

func (s *Server) removeApp(w http.ResponseWriter, r *http.Request) {
	var in idBody
	if !decode(w, r, &in) {
		return
	}
	if err := s.hub.RevokeApp(r.Context(), in.ID); err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) regionConfirm(w http.ResponseWriter, r *http.Request) {
	if err := s.hub.ConfirmRegion(r.Context()); err != nil {
		writeErr(w, r, 0, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) test(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Model string `json:"model"`
		Text  string `json:"text"`
	}
	if !decode(w, r, &in) {
		return
	}
	if r := []rune(in.Text); len(r) > 1000 {
		in.Text = string(r[:1000])
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	st, err := s.hub.Complete(ctx, omnillm.Request{Model: in.Model, Caller: omnillm.Interactive, MaxTokens: 300,
		Messages: []omnillm.Message{{Role: "user", Content: in.Text}}})
	if err != nil {
		writeErr(w, r, 0, err)
		return
	}
	defer st.Close()
	var b strings.Builder
	for {
		c, err := st.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeErr(w, r, 0, err)
			return
		}
		b.WriteString(c.Text)
	}
	writeJSON(w, map[string]any{"reply": b.String(), "usage": st.Usage()})
}

// ---------- pairing from the console ----------

type pairState struct {
	mu      sync.Mutex
	state   string // waiting | confirm | done | failed
	name    string
	sas     string
	device  string
	errMsg  error
	decide  chan decision
	expires time.Time
	seen    time.Time // last time the page asked about it
	cancel  func()
}

// decision is the owner's answer, with the name they gave the device.
type decision struct {
	yes  bool
	name string
}

// pairIdle: a console pairing nobody has looked at for this long is
// cancelled (the page was closed), so it does not hold a pairing slot.
const pairIdle = 90 * time.Second

func (s *Server) pairStart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name           string  `json:"name"`
		BudgetUSD      float64 `json:"budget_usd"`
		BackgroundOnly bool    `json:"background_only"`
	}
	if !decode(w, r, &in) {
		return
	}
	ps := &pairState{state: "waiting", decide: make(chan decision, 1)}
	// The pairing outlives this request: it waits for the device and then
	// for the owner's decision in the console.
	ctx := context.WithoutCancel(r.Context())
	p, err := s.hub.Pair(ctx, hub.PairOptions{Name: in.Name, Budget: int64(in.BudgetUSD * 1e6), BackgroundOnly: in.BackgroundOnly,
		ConfirmName: func(ctx context.Context, req hub.PairRequest) (bool, string, error) {
			ps.mu.Lock()
			ps.state, ps.name, ps.sas = "confirm", req.DeviceName, req.Code
			ps.mu.Unlock()
			select {
			case d := <-ps.decide:
				return d.yes, d.name, nil
			case <-ctx.Done():
				return false, "", ctx.Err()
			}
		}})
	if err != nil {
		writeErr(w, r, 0, err)
		return
	}
	ps.expires, ps.seen, ps.cancel = p.Expires, time.Now(), p.Cancel
	id := token()[:32]
	s.mu.Lock()
	for k, v := range s.pairs {
		// One "add device" at a time: starting a new one ends the old one.
		v.cancel()
		if s.now().After(v.expires.Add(time.Minute)) {
			delete(s.pairs, k)
		}
	}
	s.pairs[id] = ps
	s.mu.Unlock()
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for range t.C {
			ps.mu.Lock()
			open := ps.state == "waiting" || ps.state == "confirm"
			idle := time.Since(ps.seen) > pairIdle
			ps.mu.Unlock()
			if !open {
				return
			}
			if idle {
				p.Cancel()
				return
			}
		}
	}()
	go func() {
		d, err := p.Wait(ctx)
		ps.mu.Lock()
		defer ps.mu.Unlock()
		if err != nil {
			ps.state, ps.errMsg = "failed", err
			return
		}
		ps.state, ps.device, ps.name = "done", d.ID, d.Name
	}()
	svg, _ := qrcode.SVG(p.Link)
	writeJSON(w, map[string]any{
		"id": id, "code": p.Code, "expires": p.Expires,
		"qr": "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)),
	})
}

// pairCancel ends an "add device" the owner closed: the code stops
// working at once instead of when it expires.
func (s *Server) pairCancel(w http.ResponseWriter, r *http.Request) {
	var in idBody
	if !decode(w, r, &in) {
		return
	}
	s.mu.Lock()
	ps := s.pairs[in.ID]
	delete(s.pairs, in.ID)
	s.mu.Unlock()
	if ps != nil && ps.cancel != nil {
		ps.cancel()
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) pairGet(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	ps := s.pairs[r.URL.Query().Get("id")]
	s.mu.Unlock()
	if ps == nil {
		writeErr(w, r, http.StatusNotFound, &omnillm.Error{Code: omnillm.CodeNotFound, Msg: omnillm.MsgPairingExpired})
		return
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.seen = time.Now()
	out := map[string]any{"state": ps.state, "device_name": ps.name, "code": ps.sas, "device_id": ps.device}
	if ps.errMsg != nil {
		out["message"] = omnillm.Describe(lang(r), ps.errMsg)
		out["reason"] = pairReason(ps.errMsg)
	}
	writeJSON(w, out)
}

// pairReason tells the page why a pairing ended, so it can say "refused"
// or "expired" in its own words and offer a new QR code: denied (the owner
// said no), left (the device cancelled or lost its connection while the
// owner was asked), relay (this Mac lost remote access; the message says
// so), expired (the code ran out or was cancelled), or failed.
func pairReason(err error) string {
	e := omnillm.AsError(err)
	switch {
	case errors.Is(err, hub.ErrPairingDeviceLeft):
		return "left"
	case e.Msg == omnillm.MsgRelayUnavailable:
		return "relay"
	case e.Code == omnillm.CodePairingDenied:
		return "denied"
	case e.Msg == omnillm.MsgPairingExpired, errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "expired"
	}
	return "failed"
}

func (s *Server) pairDecide(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID   string `json:"id"`
		Yes  bool   `json:"yes"`
		Name string `json:"name"` // the owner's name for the device; "" keeps its own
	}
	if !decode(w, r, &in) {
		return
	}
	s.mu.Lock()
	ps := s.pairs[in.ID]
	s.mu.Unlock()
	if ps == nil {
		writeErr(w, r, http.StatusNotFound, &omnillm.Error{Code: omnillm.CodeNotFound, Msg: omnillm.MsgPairingExpired})
		return
	}
	ps.mu.Lock()
	waiting := ps.state == "confirm"
	ps.mu.Unlock()
	if !waiting {
		writeErr(w, r, http.StatusConflict, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgPairingExpired})
		return
	}
	select {
	case ps.decide <- decision{in.Yes, in.Name}:
	default:
	}
	writeJSON(w, map[string]bool{"ok": true})
}
