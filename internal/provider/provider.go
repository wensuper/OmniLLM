// Package provider talks to model providers. It runs only inside the hub,
// behind the egress limiter, and it is the only code that ever holds a
// credential in memory. Upstream responses are reduced to text, usage and a
// finish reason; upstream headers and error bodies never leave this package.
package provider

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/redact"
)

// Kind names an adapter.
type Kind string

const (
	OpenAI           Kind = "openai"
	Anthropic        Kind = "anthropic"
	Ollama           Kind = "ollama"
	OpenAICompatible Kind = "openai-compatible"
)

// Config is one connected provider.
type Config struct {
	Name    string `json:"name"` // e.g. "anthropic"
	Kind    Kind   `json:"kind"`
	BaseURL string `json:"base_url"`
	// NeedsKey is false for local models.
	NeedsKey bool `json:"needs_key"`
	// Path is the official program a subscription provider runs.
	Path string `json:"path,omitempty"`
}

// Defaults for the built-in providers.
func Default(kind Kind) (Config, error) {
	switch kind {
	case OpenAI:
		return Config{Name: "openai", Kind: OpenAI, BaseURL: "https://api.openai.com/v1", NeedsKey: true}, nil
	case Anthropic:
		return Config{Name: "anthropic", Kind: Anthropic, BaseURL: "https://api.anthropic.com", NeedsKey: true}, nil
	case Ollama:
		return Config{Name: "ollama", Kind: Ollama, BaseURL: "http://127.0.0.1:11434/v1", NeedsKey: false}, nil
	case ClaudeCode, Codex, Antigravity:
		p := FindCLI(kind)
		if p == "" {
			return Config{}, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgCLINotFound, Args: []string{string(kind)}}
		}
		return Config{Name: string(kind), Kind: kind, Path: p}, nil
	}
	return Config{}, errors.New("provider: unknown kind")
}

// Local reports whether calls stay on this machine (a loopback local model).
func (c Config) Local() bool {
	if c.Kind != Ollama {
		return false
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return false
	}
	return isLoopbackHost(u.Hostname())
}

func isLoopbackHost(h string) bool {
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// Validate checks a provider config against SSRF and downgrade risks:
// remote providers must use https on a public host name; only a local model
// may use http, and only on loopback.
func (c Config) Validate() error {
	if c.Name == "" || len(c.Name) > 40 || strings.ContainsAny(c.Name, "/ \t\n") {
		return errors.New("provider: bad name")
	}
	if c.Kind.Subscription() {
		return validateCLI(c)
	}
	if c.Path != "" {
		return errors.New("provider: only a subscription provider runs a program")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("provider: bad base URL")
	}
	host := u.Hostname()
	switch c.Kind {
	case Ollama:
		if !isLoopbackHost(host) {
			return errors.New("provider: a local model must be on this machine (127.0.0.1)")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("provider: bad scheme")
		}
		return nil
	case OpenAI, Anthropic, OpenAICompatible:
	default:
		return errors.New("provider: unknown kind")
	}
	if testLoopback.Load() && testing.Testing() && isLoopbackHost(host) {
		return nil
	}
	if u.Scheme != "https" {
		return errors.New("provider: a remote provider must use https")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !publicIP(ip) {
			return errors.New("provider: address is not public")
		}
	} else if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || !strings.Contains(host, ".") {
		return errors.New("provider: address is not public")
	}
	return nil
}

var (
	blocked4 = cidrs("0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12",
		"192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"224.0.0.0/4", "240.0.0.0/4")
	blocked6 = cidrs("::/96", "::1/128", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32", "3fff::/20",
		"2002::/16", "fc00::/7", "fe80::/10", "fec0::/10", "ff00::/8")
)

func cidrs(list ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(list))
	for _, c := range list {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		out = append(out, n)
	}
	return out
}

// publicIP reports whether ip is a public unicast address. IPv4 (including
// IPv4-mapped IPv6) is checked against IPv4 ranges only; native IPv6
// against IPv6 ranges, including the ones that embed IPv4 (6to4, Teredo,
// NAT64) or are private (ULA, site-local).
func publicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		for _, n := range blocked4 {
			if n.Contains(v4) {
				return false
			}
		}
		return true
	}
	if len(ip) != net.IPv6len {
		return false
	}
	for _, n := range blocked6 {
		if n.Contains(ip) {
			return false
		}
	}
	return true
}

var testLoopback atomic.Bool

// AllowLoopbackForTests lets tests point a remote-kind provider at a fake
// server on 127.0.0.1. It does nothing outside "go test".
func AllowLoopbackForTests() {
	if testing.Testing() {
		testLoopback.Store(true)
	}
}

// GuardedTransport is the standalone default: it refuses to connect a remote
// provider to a non-public address even if DNS says so (DNS rebinding), and
// lets local models reach loopback only. Hosts that pass their own transport
// (NEXA's egress) take this responsibility.
func GuardedTransport() *http.Transport {
	d := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		local := isLoopbackHost(host)
		for _, ip := range ips {
			if local && !ip.IP.IsLoopback() {
				return nil, errors.New("provider: local name resolved off this machine")
			}
			if !local && !publicIP(ip.IP) {
				return nil, errors.New("provider: refusing a non-public address")
			}
		}
		if len(ips) == 0 {
			return nil, errors.New("provider: no address")
		}
		return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	t.ResponseHeaderTimeout = 2 * time.Minute
	t.MaxResponseHeaderBytes = 64 << 10
	return t
}

// Client sends requests for one provider.
type Client struct {
	cfg  Config
	http *http.Client
	lim  *egress.Limiter
	idle time.Duration // longest gap between stream lines (default 2 min)
	cli  cliCheck      // subscription programs: the no-tools check

	healthMu sync.Mutex
	problem  string    // "", ProblemLogin or ProblemLimit
	until    time.Time // when a limit resets
}

// Problems a subscription source can be in, for the console.
const (
	ProblemLogin = "login" // the program is signed out
	ProblemLimit = "limit" // the subscription's usage limit is reached
)

// Problem reports what made the last call fail, if that is something the
// owner can act on (a sign-in, a limit and when it resets). The next
// successful call clears it; a limit also clears once its time is past.
func (c *Client) Problem() (problem string, resets time.Time) {
	c.healthMu.Lock()
	defer c.healthMu.Unlock()
	if c.problem == ProblemLimit && !time.Now().Before(c.until) {
		return "", time.Time{}
	}
	return c.problem, c.until
}

// SignedOut records what a sign-in check found: signed out sets the login
// problem at once (before any call fails); signed in clears it. A limit is
// left as it is.
func (c *Client) SignedOut(out bool) {
	c.healthMu.Lock()
	defer c.healthMu.Unlock()
	if out {
		c.problem, c.until = ProblemLogin, time.Time{}
	} else if c.problem == ProblemLogin {
		c.problem = ""
	}
}

// note records how a call ended: nil is a success; a login or limit error
// sets the problem; anything else leaves it as it was.
func (c *Client) note(err error) {
	c.healthMu.Lock()
	defer c.healthMu.Unlock()
	if err == nil {
		c.problem, c.until = "", time.Time{}
		return
	}
	var e *omnillm.Error
	if !errors.As(err, &e) {
		return
	}
	switch e.Msg {
	case omnillm.MsgCLILogin:
		c.problem, c.until = ProblemLogin, time.Time{}
	case omnillm.MsgSubscriptionLimit:
		c.problem, c.until = ProblemLimit, time.Now().Add(e.RetryAfter)
	}
}

// New makes a client. rt is the host's transport (nil: GuardedTransport).
func New(cfg Config, rt http.RoundTripper, lim *egress.Limiter) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if rt == nil {
		rt = GuardedTransport()
	}
	return &Client{cfg: cfg, lim: lim, idle: 2 * time.Minute, http: &http.Client{
		Transport: rt,
		// Never follow redirects: a redirect could carry the key elsewhere.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (c *Client) Config() Config { return c.cfg }

// Credential is held only for the duration of a call.
type Credential struct {
	Key string
}

// stream is the common streamed reply. Recv runs on one goroutine; Close,
// Usage and FinishReason may be called from another (a cancelled device),
// so shared fields are guarded and finishing happens exactly once.
type stream struct {
	body    io.ReadCloser
	sc      *bufio.Scanner
	parse   func(event, data string) (text string, done bool, err error)
	event   string
	release egress.Release
	idle    time.Duration
	cancel  context.CancelFunc
	timer   atomic.Pointer[time.Timer]
	parent  context.Context // the caller's context
	inRecv  atomic.Bool
	idled   atomic.Bool

	mu     sync.Mutex
	usage  omnillm.Usage
	finish string
	done   atomic.Bool
	once   sync.Once
}

func (s *stream) setUsage(in, out int) {
	s.mu.Lock()
	if in > 0 {
		s.usage.InputTokens = in
	}
	if out > 0 {
		s.usage.OutputTokens = out
	}
	s.mu.Unlock()
}

func (s *stream) setFinish(f string) {
	s.mu.Lock()
	s.finish = f
	s.mu.Unlock()
}

func (s *stream) Recv() (omnillm.Chunk, error) {
	for {
		if s.done.Load() {
			if s.idled.Load() {
				return omnillm.Chunk{}, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "upstream idle timeout"}
			}
			return omnillm.Chunk{}, io.EOF
		}
		if t := s.timer.Load(); t != nil {
			t.Reset(s.idle)
		}
		s.inRecv.Store(true)
		more := s.sc.Scan()
		s.inRecv.Store(false)
		if !more {
			err := s.sc.Err()
			if s.idled.Load() {
				return omnillm.Chunk{}, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "upstream idle timeout"}
			}
			if s.done.Load() {
				return omnillm.Chunk{}, io.EOF
			}
			s.finishCall(s.failure())
			if err == nil {
				err = io.ErrUnexpectedEOF
			}
			return omnillm.Chunk{}, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: redact.String(err.Error())}
		}
		line := s.sc.Text()
		switch {
		case line == "":
			s.event = ""
			continue
		case strings.HasPrefix(line, "event:"):
			s.event = strings.TrimSpace(line[6:])
			continue
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(line[5:])
			text, done, err := s.parse(s.event, data)
			if err != nil {
				s.finishCall(egress.Failed)
				return omnillm.Chunk{}, err
			}
			if done {
				s.finishCall(egress.OK)
				if text != "" {
					return omnillm.Chunk{Text: text}, nil
				}
				return omnillm.Chunk{}, io.EOF
			}
			if text != "" {
				return omnillm.Chunk{Text: text}, nil
			}
		}
	}
}

// failure is how a broken stream counts against the provider: a caller
// that cancelled is not the provider's fault and must not open the breaker.
func (s *stream) failure() egress.Outcome {
	if s.parent.Err() != nil {
		return egress.ClientError
	}
	return egress.Failed
}

// onIdle fires when no line arrived for idle: an upstream stall if someone
// was reading, an abandoned stream if not. Either way the slot is freed.
func (s *stream) onIdle() {
	o := egress.ClientError
	if s.inRecv.Load() {
		o = egress.Failed
	}
	// Only a stream the timer actually ended reports a timeout; one that
	// had just finished normally keeps its clean end.
	s.once.Do(func() {
		s.idled.Store(true)
		s.done.Store(true)
		s.release(o, 0)
		s.body.Close()
		s.cancel()
	})
}

func (s *stream) finishCall(o egress.Outcome) {
	s.once.Do(func() {
		s.done.Store(true)
		if t := s.timer.Load(); t != nil {
			t.Stop()
		}
		s.release(o, 0)
		s.body.Close()
		s.cancel()
	})
}

func (s *stream) Usage() omnillm.Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage
}

func (s *stream) FinishReason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finish
}

func (s *stream) Close() error {
	s.finishCall(egress.OK)
	return nil
}

// StreamChat is Stream for a person's conversation: a subscription program
// continues its kept session when the request follows one (chats.go).
// owner names who asks ("device/<id>", "host").
func (c *Client) StreamChat(ctx context.Context, upstreamModel string, req omnillm.Request, cred Credential, chats *Chats, owner string) (omnillm.Stream, error) {
	if c.cfg.Kind.Subscription() && chats != nil && req.Caller == omnillm.Interactive {
		return c.streamCLI(ctx, upstreamModel, req, chats, owner)
	}
	return c.Stream(ctx, upstreamModel, req, cred)
}

// Stream sends req (already validated, model already mapped to upstream
// name) and returns the reply stream. The egress slot is held until the
// stream ends.
func (c *Client) Stream(ctx context.Context, upstreamModel string, req omnillm.Request, cred Credential) (omnillm.Stream, error) {
	if c.cfg.Kind.Subscription() {
		return c.streamCLI(ctx, upstreamModel, req, nil, "")
	}
	if c.cfg.NeedsKey && cred.Key == "" {
		return nil, &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCredentialMissing, Args: []string{c.cfg.Name}}
	}
	httpReq, parse, err := c.build(ctx, upstreamModel, req, cred)
	if err != nil {
		return nil, err
	}
	release, err := c.lim.Acquire(ctx, c.cfg.Name, req.Caller)
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	httpReq = httpReq.WithContext(cctx)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		cancel()
		if ctx.Err() != nil {
			release(egress.ClientError, 0)
			return nil, ctx.Err()
		}
		release(egress.Failed, 0)
		return nil, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: redact.String(err.Error())}
	}
	if resp.StatusCode != http.StatusOK {
		e, outcome, wait := c.classify(resp, cred, req, upstreamModel)
		resp.Body.Close()
		cancel()
		release(outcome, wait)
		return nil, e
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	s := &stream{body: resp.Body, sc: sc, release: release, idle: c.idle, cancel: cancel, parent: ctx}
	s.parse = func(ev, data string) (string, bool, error) { return parse(s, ev, data) }
	s.timer.Store(time.AfterFunc(s.idle, s.onIdle))
	return s, nil
}

// classify turns a non-200 response into a fixed error. The body is read
// (bounded) only to recognise "unsupported response_format"; it is never
// returned or logged.
func (c *Client) classify(resp *http.Response, cred Credential, req omnillm.Request, model string) (*omnillm.Error, egress.Outcome, time.Duration) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
	lower := strings.ToLower(string(body))
	detail := fmt.Sprintf("%s: status %d", c.cfg.Name, resp.StatusCode)
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		wait := retryAfter(resp.Header.Get("Retry-After"))
		if wait <= 0 {
			wait = 30 * time.Second
		}
		if wait > time.Hour {
			wait = time.Hour
		}
		return &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyUpstream,
			Args: []string{c.cfg.Name}, RetryAfter: wait, Detail: detail}, egress.RateLimited, wait
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusPaymentRequired:
		return &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCredentialBad,
			Args: []string{c.cfg.Name, redact.Last4(cred.Key)}, Detail: detail}, egress.ClientError, 0
	case (resp.StatusCode == 400 || resp.StatusCode == 422) && (len(req.JSONSchema) > 0 || req.JSONObject) &&
		(strings.Contains(lower, "response_format") || strings.Contains(lower, "json_schema") || strings.Contains(lower, "json_object") || strings.Contains(lower, "structured")):
		return &omnillm.Error{Code: omnillm.CodeUnsupportedFormat, Msg: omnillm.MsgUnsupportedFormat, Detail: detail}, egress.ClientError, 0
	case (resp.StatusCode == 400 || resp.StatusCode == 413 || resp.StatusCode == 422) && tooLong(lower):
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgContextTooLong, Args: []string{modelName(c.cfg.Kind, model)},
			Detail: detail}, egress.ClientError, 0
	case (resp.StatusCode == 400 || resp.StatusCode == 422) && req.HasImages() &&
		(strings.Contains(lower, "image") || strings.Contains(lower, "vision") || strings.Contains(lower, "multimodal")):
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgNoVision, Detail: detail}, egress.ClientError, 0
	case resp.StatusCode == http.StatusNotFound && strings.Contains(lower, "model"):
		return &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgUnknownModel, Detail: detail}, egress.ClientError, 0
	case resp.StatusCode >= 500:
		return &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: detail}, egress.Failed, 0
	default:
		return &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: detail}, egress.ClientError, 0
	}
}

func retryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		if n > 86400 {
			n = 86400
		}
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return time.Until(t)
	}
	return 0
}
