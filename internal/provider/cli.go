package provider

// Subscription providers: the official Claude Code and Codex programs the
// owner is already logged in to on the hub. The hub runs them as a plain
// text model: no tools, no MCP servers, no settings, hooks or plugins, in
// an empty folder with a minimal environment. A person's conversation keeps
// the program's own session while it is used (chats.go); every other call
// saves nothing.
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
		return "ChatGPT (Codex)"
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

// chatOnly says what this is when the chat brings no system text of its own.
const chatOnly = "You are chatting with a person from their phone or computer. " +
	"In this chat you have no tools: you cannot read or write files, run commands or browse the web; if asked, say so briefly. " +
	"Answer the user's message directly, in the user's language."

// cliPrompt turns a chat into the one text the programs take. The system
// text goes to Claude's --system-prompt; Codex has no such flag, so it
// leads the prompt there.
// noCommand keeps a message that starts with "/" from being read as one
// of the program's commands (the owner's own slash commands or skills): a
// device's text is only ever a message to the model.
func noCommand(s string) string {
	if strings.HasPrefix(strings.TrimLeft(s, " \t\r\n"), "/") {
		return "User message:\n" + s
	}
	return s
}

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
		// Claude Code adds its own "built on the Claude Agent SDK" line
		// and the model then offers to work with files. Say plainly what
		// this is: a chat, from the person's phone or computer, no tools.
		system = "You are a helpful AI assistant. " + chatOnly
		if k == ClaudeCode {
			system = "You are Claude, an AI assistant made by Anthropic. " + chatOnly
		}
	}
	var b strings.Builder
	if k == Codex {
		b.WriteString("Instructions:\n" + system + "\n\nYou have no tools. Answer in text only.\n\n")
	}
	if len(turns) == 1 && k != Codex {
		return system, noCommand(turns[0].Content)
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
	owner   *Client // told how the call ended (Problem)
	alias   string  // the model asked for ("sonnet")
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
	model  string    // the exact model, once the program says
	told   bool      // model passed on

	chat     *chatTurn       // a person's conversation (nil: nothing kept)
	peek     []byte          // a first line read early, for Recv
	reply    strings.Builder // the answer as passed on, for chat
	finished bool            // the program ended the answer itself

	waitOnce sync.Once
	waitErr  error
}

// wait reaps the program once, whoever asks first.
func (s *cliStream) wait() error {
	s.waitOnce.Do(func() { s.waitErr = s.cmd.Wait() })
	return s.waitErr
}

const cliMaxOutput = 4 << 20

func (c *Client) streamCLI(ctx context.Context, model string, req omnillm.Request, chats *Chats, owner string) (omnillm.Stream, error) {
	release, err := c.lim.Acquire(ctx, c.cfg.Name, req.Caller)
	if err != nil {
		c.note(err)
		return nil, err
	}
	if err := c.checkCLI(ctx); err != nil {
		release(egress.ClientError, 0)
		c.note(err)
		return nil, err
	}
	return c.spawnCLI(ctx, model, req.Messages, release, chats, owner)
}

// spawnCLI starts the program for one reply; release is called once when
// it ends. With chats, the conversation keeps the program's session: a
// follow-up of a kept one sends only its new message.
func (c *Client) spawnCLI(ctx context.Context, model string, msgs []omnillm.Message, release egress.Release, chats *Chats, owner string) (*cliStream, error) {
	scratch, err := os.MkdirTemp("", "omnillm-cli-")
	if err != nil {
		release(egress.ClientError, 0)
		return nil, err
	}
	_ = os.Chmod(scratch, 0o700)
	system, prompt := cliPrompt(c.cfg.Kind, msgs)
	images := cliImages(msgs)
	env := cliEnv(c.cfg.Path, scratch)
	dir := scratch
	var turn *chatTurn
	if sys, turns := splitChat(msgs); chats != nil && len(turns) > 0 {
		turn = &chatTurn{chats: chats, owner: owner, kind: c.cfg.Kind, alias: model, system: sys,
			turns: turns, path: c.cfg.Path}
		turn.og, turn.kg = chats.gens(owner, c.cfg.Kind)
		if key, ok := followUpKey(owner, c.cfg.Kind, model, sys, turns); ok {
			if r, ok := chats.take(key); ok {
				turn.id, turn.resumed = r.ID, true
				// The session already holds the conversation: only the
				// new message goes in.
				last := turns[len(turns)-1]
				prompt, images = noCommand(last.Content), last.Images
			}
		}
		if !turn.resumed && c.cfg.Kind == ClaudeCode {
			turn.id = newChatID()
		}
		if turn.id != "" {
			chats.pending(ChatRecord{Kind: c.cfg.Kind, ID: turn.id, Owner: owner, Path: c.cfg.Path})
		}
	}
	fail := func(err error) (*cliStream, error) {
		os.RemoveAll(scratch)
		release(egress.ClientError, 0)
		if turn != nil {
			turn.discard()
		}
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
			"--setting-sources", ""} // no user/project settings, hooks or plugins
		switch {
		case turn == nil:
			args = append(args, "--no-session-persistence") // nothing saved
		case turn.resumed:
			args = append(args, "--resume", turn.id)
		default:
			args = append(args, "--session-id", turn.id)
		}
		args = append(args, "--system-prompt-file="+sp)
		if turn != nil {
			// The hub's own empty folder, the same every turn: Claude Code
			// finds a session by the folder it was started in.
			if dir, err = prepareClaudeChatDir(); err != nil {
				return fail(&omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "claude folder: " + err.Error()})
			}
		}
		if model != "" && model != "default" {
			args = append(args, "--model", model)
		}
		if len(images) > 0 {
			// Images go in as base64 in Claude Code's JSON input, not as
			// files it would need a tool to read.
			args = append(args, "--input-format", "stream-json")
			prompt = claudeInput(images, prompt)
		}
	} else {
		home, err := prepareCodexHome()
		if err != nil {
			// The hub's own Codex folder could not be made: not a login
			// problem the owner could fix with codex login.
			return fail(&omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "codex home: " + err.Error()})
		}
		env = append(env, "CODEX_HOME="+home)
		switch {
		case turn != nil && turn.resumed:
			// resume takes no --sandbox, --color or -C: the sandbox is set
			// as a value, and the folder is the process's own.
			args = append([]string{"-c", codexFileLogin, "exec", "resume", "--json", "--skip-git-repo-check", "--ignore-user-config", "--ignore-rules",
				"-c", `sandbox_mode="read-only"`}, codexToolsOff...)
			if strings.TrimSpace(prompt) == "" {
				prompt = "(The user sent the attached image without any text.)"
			}
		default:
			args = []string{"-c", codexFileLogin, "exec", "--json", "--skip-git-repo-check", "--ignore-user-config", "--ignore-rules",
				"--sandbox", "read-only", "--color", "never", "-C", scratch}
			if turn == nil {
				args = append(args, "--ephemeral") // nothing saved
			}
			args = append(args, codexToolsOff...)
		}
		if model != "" && model != "default" {
			args = append(args, "-m", model)
		}
		// Images as files in the private scratch folder, attached with -i
		// (Codex reads them itself; its tools stay off).
		for i, im := range images {
			f := filepath.Join(scratch, fmt.Sprintf("image-%d%s", i+1, imageExt[im.MIME]))
			if err := os.WriteFile(f, im.Data, 0o600); err != nil {
				return fail(err)
			}
			// One argument: -i takes several values, so a separate "-i f"
			// could swallow the "-" that reads the prompt from stdin.
			args = append(args, "--image="+f)
		}
		if turn != nil && turn.resumed {
			args = append(args, turn.id)
		}
		args = append(args, "-")
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	cmd := exec.CommandContext(cctx, c.cfg.Path, args...)
	cmd.Dir, cmd.Env = dir, env
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
		stderr: stderr, scratch: scratch, owner: c, alias: model, chat: turn}
	// A reply nobody reads to the end or closes still frees its slot when
	// the time limit stops the program.
	go func() {
		<-cctx.Done()
		s.finishWith(egress.ClientError, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "stopped"})
	}()
	if turn != nil && turn.id == "" {
		// Codex names a new session on its first line. Read it now, so a
		// turn stopped at once still knows which session to delete.
		if sc.Scan() {
			first := append([]byte(nil), sc.Bytes()...)
			if text, _ := s.line(first); text != "" || s.chatID() == "" {
				s.peek = first // not the session line: Recv reads it as usual
			}
		}
	}
	return s, nil
}

func (s *cliStream) chatID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.chat.id
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
		// The exact model, as soon as the program has said it.
		s.mu.Lock()
		m := ""
		if s.model != "" && !s.told {
			m, s.told = s.model, true
		}
		s.mu.Unlock()
		if m != "" {
			return omnillm.Chunk{Model: m}, nil
		}
		var b []byte
		if s.peek != nil {
			b, s.peek = s.peek, nil
		} else if s.sc.Scan() {
			b = s.sc.Bytes()
		} else {
			return omnillm.Chunk{}, s.end()
		}
		text, final := s.line(b)
		if text != "" && s.chat != nil {
			s.mu.Lock()
			s.reply.WriteString(text)
			s.mu.Unlock()
		}
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
		Type      string `json:"type"`
		Model     string `json:"model"`      // Claude Code's system/init
		SessionID string `json:"session_id"` // the same: the session it really uses
		Event     struct {
			Type    string `json:"type"`
			Message struct {
				Model string `json:"model"`
			} `json:"message"`
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
			// Claude Code sends most of a prompt (its own instructions,
			// the earlier turns) through the prompt cache, and counts
			// those tokens apart from input_tokens.
			CacheRead     int `json:"cache_read_input_tokens"`
			CacheCreation int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
		IsError    bool   `json:"is_error"`
		Result     string `json:"result"`
		StopReason string `json:"stop_reason"`
		RateLimit  struct {
			Status   string `json:"status"`
			ResetsAt int64  `json:"resetsAt"`
		} `json:"rate_limit_info"`
		Message  string `json:"message"`
		ThreadID string `json:"thread_id"` // Codex's thread.started
		Error    struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &ev) != nil {
		return "", false
	}
	if m := ev.Model + ev.Event.Message.Model; m != "" && ev.Type != "result" {
		s.mu.Lock()
		if s.model == "" {
			s.model = m
		}
		s.mu.Unlock()
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
		// All of what was sent counts as input, cached or not; otherwise
		// "Tokens (in)" shows next to nothing.
		s.setUsage(ev.Usage.InputTokens+ev.Usage.CacheRead+ev.Usage.CacheCreation, ev.Usage.OutputTokens)
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
	case "system": // Claude Code, first: the session it uses (the one it was given)
		if t := s.chat; t != nil && t.kind == ClaudeCode && validChatID(ev.SessionID) {
			s.mu.Lock()
			was := t.id
			moved := was != ev.SessionID
			if moved {
				t.id = ev.SessionID
			}
			s.mu.Unlock()
			if moved {
				t.chats.pending(ChatRecord{Kind: t.kind, ID: ev.SessionID, Owner: t.owner, Path: t.path})
				t.chats.discard(ChatRecord{Kind: t.kind, ID: was, Owner: t.owner, Path: t.path})
			}
		}
	case "thread.started": // Codex, first: the session id
		if t := s.chat; t != nil && !t.resumed && validChatID(ev.ThreadID) {
			s.mu.Lock()
			fresh := t.id == ""
			if fresh {
				t.id = ev.ThreadID
			}
			s.mu.Unlock()
			if fresh {
				t.chats.pending(ChatRecord{Kind: t.kind, ID: t.id, Owner: t.owner, Path: t.path})
			}
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
	// Args carry the kind; Describe names it in the reader's language.
	name := string(s.kind)
	s.mu.Lock()
	limit := s.limit
	s.mu.Unlock()
	switch {
	case tooLong(lower):
		// Before the limit check: "prompt is too long: 200429 tokens"
		// holds a "429" too.
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgContextTooLong, Args: []string{modelName(s.kind, s.alias)},
			Detail: string(s.kind) + ": too long"}, egress.ClientError, 0
	case !limit.IsZero() || strings.Contains(lower, "usage limit") || strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "limit reached") || strings.Contains(lower, "429") || strings.Contains(lower, "too many requests"):
		wait := 15 * time.Minute
		if !limit.IsZero() {
			wait = time.Until(limit)
		}
		wait = max(min(wait, 6*time.Hour), time.Minute)
		return &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgSubscriptionLimit,
			Args: []string{name}, RetryAfter: wait, Detail: string(s.kind) + ": limit"}, egress.LimitReached, wait
	case strings.Contains(lower, "login") || strings.Contains(lower, "log in") || strings.Contains(lower, "logged in") ||
		strings.Contains(lower, "authenticat") || strings.Contains(lower, "401") || strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "oauth") || strings.Contains(lower, "token expired"):
		return &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCLILogin,
			Args: []string{name}, Detail: string(s.kind) + ": login"}, egress.ClientError, 0
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
		s.mu.Lock()
		s.finished = true
		s.mu.Unlock()
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
		got := s.gotAny
		s.mu.Unlock()
		s.release(o, wait)
		if s.owner != nil && (err != nil || (o == egress.OK && got)) {
			s.owner.note(err)
		}
		s.cancel()
		var turn *chatTurn
		s.mu.Lock()
		keep, reply := s.finished && err == nil, s.reply.String()
		if s.chat != nil {
			t := *s.chat // with the id as known now (Codex sets it while reading)
			turn = &t
		}
		s.mu.Unlock()
		if keep && turn != nil {
			// The answer ended by itself, so end() has already waited for
			// the program: keep the session before the caller hears the
			// end, or a quick next message would miss it.
			turn.keep(reply)
			turn = nil
		}
		go func() {
			_ = s.wait()
			os.RemoveAll(s.scratch)
			// After the program has exited: its session file is complete,
			// or no longer written to.
			if turn != nil {
				turn.discard()
			}
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
