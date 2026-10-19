// Package relay is the connection service between hubs and devices. It is
// untrusted by design (principle 2): it routes end-to-end ciphertext by
// random mailbox ID, keeps nothing on disk, logs no payloads or mailbox IDs,
// and cannot read or forge anything that passes through it.
//
// What it does see, stated plainly: IP addresses, connection times, message
// sizes and timing, and which mailboxes are online.
package relay

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/wensuper/OmniLLM/internal/relayproto"
)

// Options configures a relay.
type Options struct {
	// HubTokenHashes lists sha256(token) of hubs allowed to register. Empty
	// with Open false means no hub can connect.
	HubTokenHashes [][32]byte
	// Open lets any hub with a token of at least 32 bytes register (the
	// public relay). Each distinct token is a separate hub.
	Open bool
	// MaxHubs bounds concurrent hubs (default 10000).
	MaxHubs int
	// MaxHubsPerIP bounds concurrent hubs from one address (default 5).
	MaxHubsPerIP int
	// MaxClaims bounds mailboxes across all hubs (default 200000).
	MaxClaims int
	// MessagesPerSecond bounds each connection (default 200).
	MessagesPerSecond int
	// Logf receives operational events without IDs or payloads. Optional.
	Logf func(format string, args ...any)
	// TrustProxy uses the last X-Forwarded-For entry (the one the trusted
	// reverse proxy appended) for per-IP limits.
	TrustProxy bool
	// WebOrigins are the web client origins whose pages may open a device
	// mailbox (nil: DefaultWebOrigins). Hub sockets never accept a browser.
	WebOrigins []string
}

// DefaultWebOrigins is where the project publishes its web client.
var DefaultWebOrigins = []string{"https://wensuper.github.io"}

// Browser devices cannot set headers on a WebSocket, so they offer two
// subprotocols: WebProtocol, and WebAuthPrefix + the mailbox secret
// (base64url). They cannot read an HTTP status either: a refused browser is
// accepted and closed with CloseUnavailable, an admitted one first gets the
// text frame "ready".
const (
	WebProtocol   = "omnillm.v1"
	WebAuthPrefix = "omnillm.auth."
)

// Server is an http.Handler.
type Server struct {
	opts Options

	mu        sync.Mutex
	hubs      map[[32]byte]*hubConn
	hubsPerIP map[string]int
	pending   int
	claims    int
	mailboxes map[relayproto.ID]*mailbox
	failures  map[string]*bucket // per-IP failed device auth

	mux *http.ServeMux
}

type hubConn struct {
	id     [32]byte
	ip     string
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
	claims map[relayproto.ID]bool
	dead   bool
}

type mailbox struct {
	hub        *hubConn
	secretHash [32]byte
	dev        *devConn
	gen        uint64
}

type devConn struct {
	conn   *websocket.Conn
	cancel context.CancelFunc
	out    chan outItem
	gen    uint64
}

// outItem is a message for the device, or (close set) the instruction to
// close after everything queued before it has been written.
type outItem struct {
	data  []byte
	close bool
	code  websocket.StatusCode
	why   string
}

// closeAfterQueued closes a device connection once the messages already
// queued for it are written (a "pairing denied" must reach the device
// before its mailbox is closed). If the queue is full it closes at once.
func (d *devConn) closeAfterQueued(code websocket.StatusCode, why string) {
	select {
	case d.out <- outItem{close: true, code: code, why: why}:
	default:
		closeLater(d.conn, code, why)
	}
}

// New makes a relay.
func New(opts Options) *Server {
	if opts.MaxHubs <= 0 {
		opts.MaxHubs = 10000
	}
	if opts.MaxHubsPerIP <= 0 {
		opts.MaxHubsPerIP = 5
	}
	if opts.MaxClaims <= 0 {
		opts.MaxClaims = 200000
	}
	if opts.MessagesPerSecond <= 0 {
		opts.MessagesPerSecond = 200
	}
	s := &Server{
		opts:      opts,
		hubs:      map[[32]byte]*hubConn{},
		hubsPerIP: map[string]int{},
		mailboxes: map[relayproto.ID]*mailbox{},
		failures:  map[string]*bucket{},
		mux:       http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /v1/hub", s.serveHub)
	s.mux.HandleFunc("GET /v1/mailbox/{id}", s.serveDevice)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	s.mux.ServeHTTP(w, r)
}

func (s *Server) logf(format string, args ...any) {
	if s.opts.Logf != nil {
		s.opts.Logf(format, args...)
	}
}

func bearer(r *http.Request) ([]byte, bool) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, false
	}
	t := strings.TrimSpace(h[len("Bearer "):])
	if t == "" || len(t) > 512 {
		return nil, false
	}
	return []byte(t), true
}

func (s *Server) hubAllowed(token []byte) ([32]byte, bool) {
	h := sha256.Sum256(token)
	if s.opts.Open {
		return h, len(token) >= 32
	}
	ok := 0
	for _, a := range s.opts.HubTokenHashes {
		ok |= subtle.ConstantTimeCompare(a[:], h[:])
	}
	return h, ok == 1
}

func accept(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	// No browser may open these sockets.
	if r.Header.Get("Origin") != "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, errors.New("origin")
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled})
	if err != nil {
		return nil, err
	}
	c.SetReadLimit(relayproto.MaxMessage)
	return c, nil
}

// closeLater closes a connection without holding any lock: a close
// handshake can take seconds. The connection's read loop then ends on its
// own, so the peer still receives the close code.
func closeLater(c *websocket.Conn, code websocket.StatusCode, reason string) {
	go c.Close(code, reason)
}

func (s *Server) serveHub(w http.ResponseWriter, r *http.Request) {
	token, ok := bearer(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := s.hubAllowed(token)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ip := s.clientIP(r)
	// Take the per-IP slot before the upgrade, so a burst of parallel
	// connections cannot all pass the check.
	s.mu.Lock()
	if s.hubs[id] == nil && (len(s.hubs)+s.pending >= s.opts.MaxHubs || s.hubsPerIP[ip] >= s.opts.MaxHubsPerIP) {
		s.mu.Unlock()
		http.Error(w, "busy", http.StatusServiceUnavailable)
		return
	}
	s.hubsPerIP[ip]++
	s.pending++
	s.mu.Unlock()
	c, err := accept(w, r)
	if err != nil {
		s.mu.Lock()
		s.pending--
		if s.hubsPerIP[ip]--; s.hubsPerIP[ip] <= 0 {
			delete(s.hubsPerIP, ip)
		}
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	h := &hubConn{id: id, ip: ip, conn: c, ctx: ctx, cancel: cancel, claims: map[relayproto.ID]bool{}}

	s.mu.Lock()
	if old := s.hubs[id]; old != nil {
		// One hub online per token: the newest wins.
		s.dropHubLocked(old, websocket.StatusCode(relayproto.CloseReplaced), "replaced")
	}
	s.hubs[id] = h
	s.pending--
	n := len(s.hubs)
	s.mu.Unlock()
	s.logf("relay: hub connected (%d online)", n)

	defer func() {
		s.mu.Lock()
		s.dropHubLocked(h, websocket.StatusNormalClosure, "bye")
		s.mu.Unlock()
		cancel()
		s.logf("relay: hub disconnected")
	}()
	go keepalive(ctx, c)
	lim := newBucket(s.opts.MessagesPerSecond)
	for {
		typ, b, err := c.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageBinary || !lim.take() {
			closeLater(c, websocket.StatusPolicyViolation, "policy")
			return
		}
		t, mid, payload, err := relayproto.Decode(b)
		if err != nil {
			closeLater(c, websocket.StatusPolicyViolation, "bad frame")
			return
		}
		s.handleHubFrame(h, t, mid, payload)
	}
}

func (s *Server) handleHubFrame(h *hubConn, t byte, mid relayproto.ID, payload []byte) {
	switch t {
	case relayproto.Claim:
		if len(payload) != 32 {
			return
		}
		var sh [32]byte
		copy(sh[:], payload)
		s.mu.Lock()
		if h.dead {
			s.mu.Unlock()
			return
		}
		mb := s.mailboxes[mid]
		switch {
		case mb != nil && mb.hub != h:
			s.mu.Unlock()
			_ = s.sendHub(h, relayproto.ClaimFail, mid, nil)
			return
		case mb == nil && (len(h.claims) >= relayproto.MaxClaimsPerHub || s.claims >= s.opts.MaxClaims):
			s.mu.Unlock()
			_ = s.sendHub(h, relayproto.ClaimFail, mid, nil)
			return
		case mb == nil:
			s.mailboxes[mid] = &mailbox{hub: h, secretHash: sh}
			h.claims[mid] = true
			s.claims++
		default:
			mb.secretHash = sh
		}
		s.mu.Unlock()
	case relayproto.Release, relayproto.Kick:
		s.mu.Lock()
		mb := s.mailboxes[mid]
		if mb == nil || mb.hub != h {
			s.mu.Unlock()
			return
		}
		dev := mb.dev
		mb.dev = nil
		if t == relayproto.Release {
			delete(s.mailboxes, mid)
			delete(h.claims, mid)
			s.claims--
		}
		s.mu.Unlock()
		if dev != nil {
			dev.closeAfterQueued(websocket.StatusCode(relayproto.CloseKicked), "closed")
		}
	case relayproto.Data:
		s.mu.Lock()
		mb := s.mailboxes[mid]
		var dev *devConn
		if mb != nil && mb.hub == h {
			dev = mb.dev
		}
		s.mu.Unlock()
		if dev == nil {
			return
		}
		// Each device has its own queue: one stalled device never blocks
		// the hub's other mailboxes. A device that cannot keep up is cut.
		select {
		case dev.out <- outItem{data: payload}:
		default:
			closeLater(dev.conn, websocket.StatusPolicyViolation, "slow")
		}
	}
}

// dropHubLocked forgets a hub and every device on its mailboxes. Network
// closes happen outside the lock.
func (s *Server) dropHubLocked(h *hubConn, code websocket.StatusCode, reason string) {
	if h.dead {
		return
	}
	h.dead = true
	for mid := range h.claims {
		if mb := s.mailboxes[mid]; mb != nil && mb.hub == h {
			if mb.dev != nil {
				closeLater(mb.dev.conn, websocket.StatusCode(relayproto.CloseUnavailable), "unavailable")
			}
			delete(s.mailboxes, mid)
			s.claims--
		}
	}
	h.claims = map[relayproto.ID]bool{}
	if s.hubs[h.id] == h {
		delete(s.hubs, h.id)
	}
	if s.hubsPerIP[h.ip]--; s.hubsPerIP[h.ip] <= 0 {
		delete(s.hubsPerIP, h.ip)
	}
	closeLater(h.conn, code, reason)
	h.cancel()
}

func (s *Server) sendHub(h *hubConn, t byte, mid relayproto.ID, payload []byte) error {
	ctx, cancel := context.WithTimeout(h.ctx, 15*time.Second)
	defer cancel()
	return h.conn.Write(ctx, websocket.MessageBinary, relayproto.Encode(t, mid, payload))
}

func (s *Server) clientIP(r *http.Request) string {
	if s.opts.TrustProxy {
		if f := r.Header.Get("X-Forwarded-For"); f != "" {
			parts := strings.Split(f, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// tooManyFailures limits wrong-secret guessing per IP.
func (s *Server) tooManyFailures(ip string, record bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.failures[ip]
	if b == nil {
		if !record {
			return false
		}
		if len(s.failures) > 100000 {
			// Drop buckets that have refilled; keep the active ones.
			for k, v := range s.failures {
				v.refill()
				if v.tokens >= v.burst {
					delete(s.failures, k)
				}
			}
		}
		b = &bucket{rate: 0.2, burst: 10, tokens: 10, last: time.Now()}
		s.failures[ip] = b
	}
	if !record {
		b.refill()
		return b.tokens < 1
	}
	return !b.take()
}

func genBytes(g uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, g)
	return b
}

func (s *Server) serveDevice(w http.ResponseWriter, r *http.Request) {
	web, webAuth := webProtocols(r)
	if web && !s.webOriginAllowed(r.Header.Get("Origin")) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	// refuse gives the one "unavailable" answer, in the form the client
	// can read.
	refuse := func(status int) {
		if !web {
			http.Error(w, "unavailable", status)
			return
		}
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{WebProtocol},
			CompressionMode: websocket.CompressionDisabled, OriginPatterns: s.originHosts()})
		if err == nil {
			_ = c.Close(websocket.StatusCode(relayproto.CloseUnavailable), "unavailable")
		}
	}
	ip := s.clientIP(r)
	if s.tooManyFailures(ip, false) {
		refuse(http.StatusTooManyRequests)
		return
	}
	mid, err := relayproto.ParseID(r.PathValue("id"))
	enc, ok := bearer(r)
	if web {
		enc, ok = []byte(webAuth), webAuth != ""
	}
	var secretB []byte
	if ok {
		secretB, err = base64.RawURLEncoding.DecodeString(string(enc))
		ok = err == nil && len(secretB) == 32
	}
	if err != nil || !ok {
		s.tooManyFailures(ip, true)
		refuse(http.StatusNotFound)
		return
	}
	sh := relayproto.SecretHash(secretB)
	s.mu.Lock()
	mb := s.mailboxes[mid]
	valid := mb != nil && subtle.ConstantTimeCompare(mb.secretHash[:], sh[:]) == 1
	s.mu.Unlock()
	if !valid {
		s.tooManyFailures(ip, true)
		// Same answer for unknown mailbox, wrong secret and hub offline.
		refuse(http.StatusNotFound)
		return
	}
	var c *websocket.Conn
	if web {
		c, err = websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{WebProtocol},
			CompressionMode: websocket.CompressionDisabled, OriginPatterns: s.originHosts()})
		if err == nil {
			c.SetReadLimit(relayproto.MaxMessage)
		}
	} else {
		c, err = accept(w, r)
	}
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	d := &devConn{conn: c, cancel: cancel, out: make(chan outItem, 128)}

	s.mu.Lock()
	mb = s.mailboxes[mid]
	if mb == nil || subtle.ConstantTimeCompare(mb.secretHash[:], sh[:]) != 1 || mb.hub.dead {
		s.mu.Unlock()
		closeLater(c, websocket.StatusCode(relayproto.CloseUnavailable), "unavailable")
		cancel()
		return
	}
	old := mb.dev
	mb.gen++
	d.gen = mb.gen
	mb.dev = d
	h := mb.hub
	s.mu.Unlock()
	if old != nil {
		closeLater(old.conn, websocket.StatusCode(relayproto.CloseReplaced), "replaced")
	}
	if web {
		wctx, wc := context.WithTimeout(ctx, 10*time.Second)
		err := c.Write(wctx, websocket.MessageText, []byte("ready"))
		wc()
		if err != nil {
			s.mu.Lock()
			if mb := s.mailboxes[mid]; mb != nil && mb.dev == d {
				mb.dev = nil
			}
			s.mu.Unlock()
			cancel()
			c.CloseNow()
			return
		}
	}
	// Open and Closed carry the attachment's generation, so the hub ignores
	// a Closed for an older connection that arrives after a newer Open.
	if err := s.sendHub(h, relayproto.Open, mid, genBytes(d.gen)); err != nil {
		s.mu.Lock()
		if mb := s.mailboxes[mid]; mb != nil && mb.dev == d {
			mb.dev = nil
		}
		s.mu.Unlock()
		closeLater(c, websocket.StatusCode(relayproto.CloseUnavailable), "unavailable")
		cancel()
		return
	}
	defer func() {
		s.mu.Lock()
		if mb := s.mailboxes[mid]; mb != nil && mb.dev == d {
			mb.dev = nil
		}
		s.mu.Unlock()
		_ = s.sendHub(h, relayproto.Closed, mid, genBytes(d.gen))
		cancel()
		closeLater(c, websocket.StatusNormalClosure, "bye")
	}()
	go func() { // hub → device
		for {
			select {
			case it := <-d.out:
				if it.close {
					closeLater(c, it.code, it.why)
					return
				}
				wctx, wc := context.WithTimeout(ctx, 15*time.Second)
				err := c.Write(wctx, websocket.MessageBinary, it.data)
				wc()
				if err != nil {
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	go keepalive(ctx, c)
	lim := newBucket(s.opts.MessagesPerSecond)
	for {
		typ, b, err := c.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageBinary || !lim.take() || len(b) > 65535 {
			closeLater(c, websocket.StatusPolicyViolation, "policy")
			return
		}
		s.mu.Lock()
		mb := s.mailboxes[mid]
		ok := mb != nil && mb.dev == d
		s.mu.Unlock()
		if !ok {
			return
		}
		if err := s.sendHub(h, relayproto.Data, mid, b); err != nil {
			return
		}
	}
}

func keepalive(ctx context.Context, c *websocket.Conn) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			err := c.Ping(pctx)
			cancel()
			if err != nil {
				closeLater(c, websocket.StatusGoingAway, "timeout")
				return
			}
		}
	}
}

// bucket is a token bucket.
type bucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newBucket(perSec int) *bucket {
	return &bucket{rate: float64(perSec), burst: float64(perSec) * 2, tokens: float64(perSec) * 2, last: time.Now()}
}

func (b *bucket) refill() {
	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
}

func (b *bucket) take() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
