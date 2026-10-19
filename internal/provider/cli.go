package provider

// Subscription providers: the official Claude Code and Codex programs the
// owner is already logged in to on the hub. The hub runs them as a plain
// text model: no tools, no MCP servers, no settings, hooks or plugins, no
// saved sessions, in an empty scratch folder with a minimal environment.
// The login stays inside the official program (its keychain entry or auth
// file); the hub never reads it (design §5.4). Only people in a
// conversation may use these models (§5.2): the hub forces HumanOnly.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/redact"
)

const (
	ClaudeCode Kind = "claude-code"
	Codex      Kind = "codex"
)

// Subscription reports whether the kind runs an official program logged in
// with the owner's subscription.
func (k Kind) Subscription() bool { return k == ClaudeCode || k == Codex }

// cliName is the program's file name for a kind.
func cliName(k Kind) string {
	if k == ClaudeCode {
		return "claude"
	}
	return "codex"
}

// DisplayName is how people see a subscription provider.
func DisplayName(k Kind) string {
	switch k {
	case ClaudeCode:
		return "Claude (subscription)"
	case Codex:
		return "ChatGPT Codex (subscription)"
	}
	return string(k)
}

// cliModels are the models offered for each program. Claude Code takes
// family aliases that follow its newest models; Codex uses its default.
var cliModels = map[Kind][]string{
	ClaudeCode: {"sonnet", "opus", "haiku"},
	Codex:      {"default"},
}

// cliDirs are where the official installers put the programs; a launchd or
// systemd service has a minimal PATH, so PATH alone does not find them.
func cliDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := filepath.SplitList(os.Getenv("PATH"))
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "bin"), filepath.Join(home, ".claude", "local"),
			filepath.Join(home, ".npm-global", "bin"), filepath.Join(home, ".bun", "bin"), filepath.Join(home, ".volta", "bin"))
	}
	return append(dirs, "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin")
}

// FindCLI looks for the official program of kind. It returns "" if absent.
func FindCLI(k Kind) string {
	if !k.Subscription() {
		return ""
	}
	for _, d := range cliDirs() {
		if d == "" || !filepath.IsAbs(d) {
			continue
		}
		p := filepath.Join(d, cliName(k))
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0o111 != 0 {
			return p
		}
	}
	return ""
}

// CLILoggedIn asks the official program whether it is logged in. It never
// reads the login itself.
func CLILoggedIn(ctx context.Context, k Kind, path string) bool {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "omnillm-cli-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	var cmd *exec.Cmd
	env := cliEnv(path, dir)
	if k == ClaudeCode {
		cmd = exec.CommandContext(ctx, path, "auth", "status")
	} else {
		home, err := prepareCodexHome()
		if err != nil {
			return false
		}
		cmd = exec.CommandContext(ctx, path, "-c", codexFileLogin, "login", "status")
		env = append(env, "CODEX_HOME="+home)
	}
	cmd.Dir, cmd.Env = dir, env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	if k == ClaudeCode {
		var st struct {
			LoggedIn bool `json:"loggedIn"`
		}
		return json.Unmarshal(out, &st) == nil && st.LoggedIn
	}
	return strings.Contains(strings.ToLower(string(out)), "logged in")
}

// cliEnv is the whole environment a program gets: who and where the owner
// is, a PATH holding the program's own folders and the system's, and a
// scratch temp folder. Nothing else of the hub's environment (its master
// key, relay token, proxies) is passed on.
func cliEnv(path, scratch string) []string {
	home, _ := os.UserHomeDir()
	dirs := []string{filepath.Dir(path)}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		dirs = append(dirs, filepath.Dir(real))
	}
	dirs = append(dirs, "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin")
	env := []string{"HOME=" + home, "PATH=" + strings.Join(dirs, ":"), "TMPDIR=" + scratch, "LANG=en_US.UTF-8", "NO_COLOR=1"}
	for _, k := range []string{"USER", "LOGNAME", "SHELL"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func validateCLI(c Config) error {
	if c.Name != string(c.Kind) {
		return errors.New("provider: a subscription provider is named after its kind")
	}
	if c.BaseURL != "" || c.NeedsKey {
		return errors.New("provider: a subscription provider has no URL or key")
	}
	if !filepath.IsAbs(c.Path) || filepath.Base(c.Path) != cliName(c.Kind) {
		return errors.New("provider: bad program path")
	}
	return nil
}

// codexFileLogin keeps Codex on the auth.json login (the linked file), not
// a keychain entry of its own.
const codexFileLogin = `cli_auth_credentials_store="file"`

// codexToolsOff turns off every Codex tool: the shell, code mode, apps,
// plugins, browser and computer use, image generation and memories. As -c
// overrides, a feature a given Codex does not know is ignored.
var codexToolsOff = func() []string {
	var out []string
	for _, f := range []string{"shell_tool", "unified_exec", "code_mode_host", "apps", "plugins", "browser_use",
		"browser_use_external", "computer_use", "in_app_browser", "image_generation", "memories", "tool_suggest",
		"web_search_request", "web_search_cached"} {
		out = append(out, "-c", "features."+f+"=false")
	}
	return append(out, "-c", `web_search="disabled"`)
}()

// cliPrompt turns a chat into the one text the programs take. The system
// text goes to Claude's --system-prompt; Codex has no such flag, so it
// leads the prompt there.
func cliPrompt(k Kind, msgs []omnillm.Message) (system, prompt string) {
	var sys []string
	var turns []omnillm.Message
	for _, m := range msgs {
		if m.Role == "system" {
			sys = append(sys, m.Content)
		} else {
			turns = append(turns, m)
		}
	}
	system = strings.Join(sys, "\n\n")
	if system == "" {
		system = "You are a helpful assistant. Answer the user's message directly."
	}
	var b strings.Builder
	if k == Codex {
		b.WriteString("Instructions:\n" + system + "\n\nYou have no tools. Answer in text only.\n\n")
	}
	if len(turns) == 1 && k != Codex {
		return system, turns[0].Content
	}
	b.WriteString("The conversation so far is below. Reply to the user's last message as the assistant.\n")
	n := 0
	for _, m := range turns {
		who := "User"
		if m.Role == "assistant" {
			who = "Assistant"
		}
		b.WriteString("\n" + who + ":\n" + m.Content + "\n")
		for range m.Images {
			n++
			fmt.Fprintf(&b, "[attached image %d]\n", n)
		}
	}
	return system, b.String()
}

// cliImages lists the conversation's images in order (the transcript
// numbers them the same way).
func cliImages(msgs []omnillm.Message) []omnillm.Image {
	var out []omnillm.Image
	for _, m := range msgs {
		out = append(out, m.Images...)
	}
	return out
}

// claudeInput is Claude Code's stream-json input: one user message with
// the images and the prompt.
func claudeInput(images []omnillm.Image, prompt string) string {
	var parts []map[string]any
	for _, im := range images {
		parts = append(parts, map[string]any{"type": "image", "source": map[string]string{
			"type": "base64", "media_type": im.MIME, "data": base64.StdEncoding.EncodeToString(im.Data)}})
	}
	if strings.TrimSpace(prompt) == "" {
		prompt = "(The user sent the image above without any text.)"
	}
	parts = append(parts, map[string]any{"type": "text", "text": prompt})
	b, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": parts}})
	return string(b) + "\n"
}

var imageExt = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}

// cliStream is a reply read from the program's JSON lines.
type cliStream struct {
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	parent  context.Context
	release egress.Release
	kind    Kind
	sc      *bufio.Scanner
	stderr  *limitedWriter
	scratch string
	sent    int

	mu     sync.Mutex
	usage  omnillm.Usage
	finish string
	once   sync.Once
	err    error // set once finished with a failure
	done   bool
	gotAny bool
	limit  time.Time // a subscription limit's reset time, if reported

	waitOnce sync.Once
	waitErr  error
}

// wait reaps the program once, whoever asks first.
func (s *cliStream) wait() error {
	s.waitOnce.Do(func() { s.waitErr = s.cmd.Wait() })
	return s.waitErr
}

const cliMaxOutput = 4 << 20

func (c *Client) streamCLI(ctx context.Context, model string, req omnillm.Request) (omnillm.Stream, error) {
	release, err := c.lim.Acquire(ctx, c.cfg.Name, req.Caller)
	if err != nil {
		return nil, err
	}
	if err := c.checkCLI(ctx); err != nil {
		release(egress.ClientError, 0)
		return nil, err
	}
	return c.spawnCLI(ctx, model, req.Messages, release)
}

// spawnCLI starts the program for one reply; release is called once when
// it ends.
func (c *Client) spawnCLI(ctx context.Context, model string, msgs []omnillm.Message, release egress.Release) (*cliStream, error) {
	req := omnillm.Request{Messages: msgs}
	scratch, err := os.MkdirTemp("", "omnillm-cli-")
	if err != nil {
		release(egress.ClientError, 0)
		return nil, err
	}
	_ = os.Chmod(scratch, 0o700)
	system, prompt := cliPrompt(c.cfg.Kind, req.Messages)
	env := cliEnv(c.cfg.Path, scratch)
	fail := func(err error) (*cliStream, error) {
		os.RemoveAll(scratch)
		release(egress.ClientError, 0)
		return nil, err
	}
	var args []string
	if c.cfg.Kind == ClaudeCode {
		// The system text goes in a file in the private scratch folder,
		// not argv, where other local users could read it.
		sp := filepath.Join(scratch, "system.txt")
		if err := os.WriteFile(sp, []byte(system), 0o600); err != nil {
			return fail(err)
		}
		args = []string{"-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages",
			"--tools", "", // no built-in tools
			"--strict-mcp-config",   // no MCP servers
			"--setting-sources", "", // no user/project settings, hooks or plugins
			"--no-session-persistence", // nothing saved in the owner's history
			"--system-prompt-file=" + sp}
		if model != "" && model != "default" {
			args = append(args, "--model", model)
		}
		if imgs := cliImages(req.Messages); len(imgs) > 0 {
			// Images go in as base64 in Claude Code's JSON input, not as
			// files it would need a tool to read.
			args = append(args, "--input-format", "stream-json")
			prompt = claudeInput(imgs, prompt)
		}
	} else {
		home, err := prepareCodexHome()
		if err != nil {
			return fail(&omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCLILogin,
				Args: []string{DisplayName(c.cfg.Kind), cliName(c.cfg.Kind)}, Detail: err.Error()})
		}
		env = append(env, "CODEX_HOME="+home)
		args = append([]string{"-c", codexFileLogin, "exec", "--json", "--skip-git-repo-check", "--ephemeral", "--ignore-user-config", "--ignore-rules",
			"--sandbox", "read-only", "--color", "never", "-C", scratch}, codexToolsOff...)
		if model != "" && model != "default" {
			args = append(args, "-m", model)
		}
		// Images as files in the private scratch folder, attached with -i
		// (Codex reads them itself; its tools stay off).
		for i, im := range cliImages(req.Messages) {
			f := filepath.Join(scratch, fmt.Sprintf("image-%d%s", i+1, imageExt[im.MIME]))
			if err := os.WriteFile(f, im.Data, 0o600); err != nil {
				return fail(err)
			}
			args = append(args, "-i", f)
		}
		args = append(args, "-")
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	cmd := exec.CommandContext(cctx, c.cfg.Path, args...)
	cmd.Dir, cmd.Env = scratch, env
	cmd.Stdin = strings.NewReader(prompt)
	cmd.WaitDelay = 5 * time.Second
	ownGroup(cmd) // a cancel stops its helpers too
	stderr := &limitedWriter{n: 16 << 10}
	cmd.Stderr = stderr
	out, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		cancel()
		return fail(&omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: redact.String(err.Error())})
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 64<<10), 8<<20)
	s := &cliStream{cmd: cmd, cancel: cancel, parent: ctx, release: release, kind: c.cfg.Kind, sc: sc,
		stderr: stderr, scratch: scratch}
	// A reply nobody reads to the end or closes still frees its slot when
	// the time limit stops the program.
	go func() {
		<-cctx.Done()
		s.finishWith(egress.ClientError, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "stopped"})
	}()
	return s, nil
}

// limitedWriter keeps the first n bytes of the program's stderr. The exec
// copy goroutine writes while classify may read.
type limitedWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
	n   int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.n > 0 {
		k := min(len(p), l.n)
		l.buf.Write(p[:k])
		l.n -= k
	}
	return len(p), nil
}

func (l *limitedWriter) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func (s *cliStream) Recv() (omnillm.Chunk, error) {
	for {
		s.mu.Lock()
		done, ferr := s.done, s.err
		s.mu.Unlock()
		if done {
			if ferr != nil {
				return omnillm.Chunk{}, ferr
			}
			return omnillm.Chunk{}, io.EOF
		}
		if !s.sc.Scan() {
			return omnillm.Chunk{}, s.end()
		}
		text, final := s.line(s.sc.Bytes())
		if final {
			if text != "" {
				s.sent += len(text)
				return omnillm.Chunk{Text: text}, nil
			}
			continue
		}
		if text != "" {
			s.sent += len(text)
			if s.sent > cliMaxOutput {
				s.finishWith(egress.OK, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "reply too long"})
				continue
			}
			return omnillm.Chunk{Text: text}, nil
		}
	}
}

// line reads one JSON line and returns text to pass on.
func (s *cliStream) line(b []byte) (text string, final bool) {
	var ev struct {
		Type  string `json:"type"`
		Event struct {
			Type  string `json:"type"`
			Delta struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
		} `json:"event"`
		Item struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"item"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		IsError    bool   `json:"is_error"`
		Result     string `json:"result"`
		StopReason string `json:"stop_reason"`
		RateLimit  struct {
			Status   string `json:"status"`
			ResetsAt int64  `json:"resetsAt"`
		} `json:"rate_limit_info"`
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &ev) != nil {
		return "", false
	}
	switch ev.Type {
	case "stream_event": // Claude Code, partial message
		if ev.Event.Type == "content_block_delta" && ev.Event.Delta.Type == "text_delta" {
			s.mu.Lock()
			s.gotAny = true
			s.mu.Unlock()
			return ev.Event.Delta.Text, false
		}
		if ev.Event.Type == "message_delta" && ev.Event.Delta.StopReason != "" {
			s.setFinish(mapStop(ev.Event.Delta.StopReason))
		}
	case "rate_limit_event":
		if st := ev.RateLimit.Status; st != "" && st != "allowed" && st != "allowed_warning" && ev.RateLimit.ResetsAt > 0 {
			s.mu.Lock()
			s.limit = time.Unix(ev.RateLimit.ResetsAt, 0)
			s.mu.Unlock()
		}
	case "result": // Claude Code, end
		s.setUsage(ev.Usage.InputTokens, ev.Usage.OutputTokens)
		if ev.IsError {
			s.fail(ev.Result)
			return "", true
		}
		s.mu.Lock()
		got := s.gotAny
		s.gotAny = s.gotAny || ev.Result != ""
		s.mu.Unlock()
		if !got && ev.Result != "" { // no partial messages: the whole reply
			return ev.Result, true
		}
	case "item.completed": // Codex
		if ev.Item.Type == "agent_message" {
			s.mu.Lock()
			s.gotAny = true
			s.mu.Unlock()
			return ev.Item.Text, false
		}
	case "turn.completed":
		s.setUsage(ev.Usage.InputTokens, ev.Usage.OutputTokens)
		s.setFinish("stop")
	case "turn.failed", "error":
		msg := ev.Message
		if ev.Error.Message != "" {
			msg = ev.Error.Message
		}
		s.fail(msg)
		return "", true
	}
	return "", false
}

func mapStop(r string) string {
	switch r {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	}
	return r
}

func (s *cliStream) setUsage(in, out int) {
	s.mu.Lock()
	if in > 0 {
		s.usage.InputTokens = in
	}
	if out > 0 {
		s.usage.OutputTokens = out
	}
	s.mu.Unlock()
}

func (s *cliStream) setFinish(f string) {
	s.mu.Lock()
	s.finish = f
	s.mu.Unlock()
}

// fail records a failure the program reported; the text is classified, not
// passed on (it may quote the account).
func (s *cliStream) fail(text string) {
	e, o, wait := s.classify(text)
	s.finishWith2(o, wait, e)
}

func (s *cliStream) classify(text string) (*omnillm.Error, egress.Outcome, time.Duration) {
	lower := strings.ToLower(text + " " + s.stderr.String())
	name := DisplayName(s.kind)
	s.mu.Lock()
	limit := s.limit
	s.mu.Unlock()
	switch {
	case !limit.IsZero() || strings.Contains(lower, "usage limit") || strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "limit reached") || strings.Contains(lower, "429") || strings.Contains(lower, "too many requests"):
		wait := 15 * time.Minute
		if !limit.IsZero() {
			wait = time.Until(limit)
		}
		wait = max(min(wait, 6*time.Hour), time.Minute)
		return &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgSubscriptionLimit,
			Args: []string{name, omnillm.RoundDuration("en", wait)}, RetryAfter: wait, Detail: string(s.kind) + ": limit"}, egress.RateLimited, wait
	case strings.Contains(lower, "login") || strings.Contains(lower, "log in") || strings.Contains(lower, "logged in") ||
		strings.Contains(lower, "authenticat") || strings.Contains(lower, "401") || strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "oauth") || strings.Contains(lower, "token expired"):
		return &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCLILogin,
			Args: []string{name, cliName(s.kind)}, Detail: string(s.kind) + ": login"}, egress.ClientError, 0
	}
	return &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: string(s.kind) + ": " + redact.String(tail(text))}, egress.Failed, 0
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 300 {
		return s[len(s)-300:]
	}
	return s
}

// end runs when stdout closed or could not be read: wait for the program
// and settle. A read error (an over-long line) stops the program first, or
// it would block writing and hold the slot.
func (s *cliStream) end() error {
	scanErr := s.sc.Err()
	if scanErr != nil {
		s.cancel()
	}
	werr := s.wait()
	if scanErr != nil && werr == nil {
		werr = scanErr
	}
	s.mu.Lock()
	done, ferr, got := s.done, s.err, s.gotAny
	s.mu.Unlock()
	if done {
		if ferr != nil {
			return ferr
		}
		return io.EOF
	}
	switch {
	case s.parent.Err() != nil:
		s.finishWith(egress.ClientError, s.parent.Err())
	case werr != nil || !got:
		msg := ""
		if werr != nil {
			msg = werr.Error()
		}
		e, o, wait := s.classify(msg)
		s.finishWith2(o, wait, e)
	default:
		if s.FinishReason() == "" {
			s.setFinish("stop")
		}
		s.finishWith(egress.OK, nil)
		return io.EOF
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *cliStream) finishWith(o egress.Outcome, err error) { s.finishWith2(o, 0, err) }

func (s *cliStream) finishWith2(o egress.Outcome, wait time.Duration, err error) {
	s.once.Do(func() {
		s.mu.Lock()
		s.done, s.err = true, err
		s.mu.Unlock()
		s.release(o, wait)
		s.cancel()
		go func() {
			_ = s.wait()
			os.RemoveAll(s.scratch)
		}()
	})
}

func (s *cliStream) Usage() omnillm.Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage
}

func (s *cliStream) FinishReason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finish
}

// Close stops the program (a cancelled device) and frees the slot.
func (s *cliStream) Close() error {
	o := egress.OK
	if s.parent.Err() != nil {
		o = egress.ClientError
	}
	s.finishWith(o, nil)
	return nil
}
