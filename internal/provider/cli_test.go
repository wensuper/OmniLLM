package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/egress"
)

// fakeCLI writes a stand-in "claude" that prints body for any prompt and
// records its environment and arguments.
func fakeCLI(t *testing.T, body string) (*Client, string) {
	t.Helper()
	dir := t.TempDir()
	rec := filepath.Join(dir, "rec")
	script := "#!/bin/sh\nif [ \"$1\" = auth ]; then echo '{\"loggedIn\":true}'; exit 0; fi\n" +
		"env > " + rec + ".env\necho \"$@\" > " + rec + ".args\ncat > " + rec + ".stdin\n" + body + "\n"
	p := filepath.Join(dir, "claude")
	if err := os.WriteFile(p, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{Name: "claude-code", Kind: ClaudeCode, Path: p}, nil, egress.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	c.cli.passed, c.cli.checked = "test", time.Now() // the canary has its own tests
	return c, rec
}

const okLines = `cat <<'X'
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"po"}}}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"ng"}}}
{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}
{"type":"result","is_error":false,"result":"pong","usage":{"input_tokens":3,"output_tokens":2}}
X`

func ask(c *Client, msgs ...omnillm.Message) (omnillm.Stream, error) {
	if len(msgs) == 0 {
		msgs = []omnillm.Message{{Role: "user", Content: "ping"}}
	}
	return c.Stream(context.Background(), "haiku", omnillm.Request{Caller: omnillm.Interactive, Messages: msgs}, Credential{})
}

func TestCLIStreamsAndKeepsEnvMinimal(t *testing.T) {
	t.Setenv("OMNILLM_MASTER_KEY", "SENTINEL-MASTER")
	t.Setenv("OMNILLM_RELAY_TOKEN", "SENTINEL-RELAY")
	t.Setenv("HTTPS_PROXY", "http://evil.example")
	c, rec := fakeCLI(t, okLines)
	s, err := ask(c, omnillm.Message{Role: "system", Content: "Be brief."}, omnillm.Message{Role: "user", Content: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	text, err := collect(t, s)
	if err != nil || text != "pong" {
		t.Fatalf("%q %v", text, err)
	}
	if u := s.Usage(); u.InputTokens != 3 || u.OutputTokens != 2 || s.FinishReason() != "stop" {
		t.Fatalf("%+v %s", u, s.FinishReason())
	}
	env, _ := os.ReadFile(rec + ".env")
	for _, bad := range []string{"SENTINEL", "evil.example"} {
		if strings.Contains(string(env), bad) {
			t.Fatalf("the program saw %s", bad)
		}
	}
	args, _ := os.ReadFile(rec + ".args")
	for _, want := range []string{"--tools  --strict-mcp-config --setting-sources  --no-session-persistence", "--system-prompt-file=", "--model haiku"} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("args %q lack %q", args, want)
		}
	}
	if in, _ := os.ReadFile(rec + ".stdin"); string(in) != "ping" {
		t.Fatalf("stdin %q", in)
	}
}

func TestCLINotLoggedIn(t *testing.T) {
	c, _ := fakeCLI(t, `echo '{"type":"result","is_error":true,"result":"Invalid API key · Please run /login"}'; exit 1`)
	s, _ := ask(c)
	_, err := collect(t, s)
	var e *omnillm.Error
	if !errors.As(err, &e) || e.Msg != omnillm.MsgCLILogin || strings.Contains(omnillm.Describe("en", err), "Invalid API key") {
		t.Fatalf("%v", err)
	}
}

func TestCLISubscriptionLimitPausesProvider(t *testing.T) {
	reset := time.Now().Add(2 * time.Hour).Unix()
	c, _ := fakeCLI(t, `echo '{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":`+itoa(reset)+`}}'
echo '{"type":"result","is_error":true,"result":"Claude AI usage limit reached"}'; exit 1`)
	s, _ := ask(c)
	_, err := collect(t, s)
	var e *omnillm.Error
	if !errors.As(err, &e) || e.Msg != omnillm.MsgSubscriptionLimit || e.RetryAfter < time.Hour {
		t.Fatalf("%v %+v", err, e)
	}
	// The wait is written in the reader's language, not fixed in English.
	if zh := omnillm.Describe("zh", err); !strings.Contains(zh, "Claude 订阅") || !strings.Contains(zh, "约 2 小时") || strings.Contains(zh, "minutes") {
		t.Fatal(zh)
	}
	// The provider is paused: the next call does not run the program.
	if _, err := ask(c); err == nil {
		t.Fatal("a call went out while the subscription limit is reached")
	}
}

func TestCLICloseStopsTheProgram(t *testing.T) {
	c, rec := fakeCLI(t, `echo '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"a"}}}'
sleep 30; echo late > `+"\"$0.late\"")
	_ = rec
	s, err := ask(c)
	if err != nil {
		t.Fatal(err)
	}
	if ch, err := s.Recv(); err != nil || ch.Text != "a" {
		t.Fatal(ch, err)
	}
	start := time.Now()
	s.Close()
	// The slot is free again right away (one at a time otherwise).
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rel, err := c.lim.Acquire(ctx, "claude-code", omnillm.Interactive)
	if err != nil {
		t.Fatalf("slot not freed: %v", err)
	}
	rel(egress.OK, 0)
	if time.Since(start) > 3*time.Second {
		t.Fatal("close was slow")
	}
}

func TestCLIConfigValidation(t *testing.T) {
	for _, c := range []Config{
		{Name: "claude-code", Kind: ClaudeCode, Path: "claude"},
		{Name: "claude-code", Kind: ClaudeCode, Path: "/bin/sh"},
		{Name: "mine", Kind: ClaudeCode, Path: "/x/claude"},
		{Name: "claude-code", Kind: ClaudeCode, Path: "/x/claude", BaseURL: "https://a.example"},
		{Name: "openai", Kind: OpenAI, BaseURL: "https://api.openai.com/v1", Path: "/x/claude"},
	} {
		if c.Validate() == nil {
			t.Errorf("accepted %+v", c)
		}
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// A program that obeys "run a command" is refused, and stays refused for
// that version.
func TestCLICanaryRefusesToolUse(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 9.9.9; exit 0; fi\n" +
		"prompt=$(cat)\nf=$(echo \"$prompt\" | sed -n 's/.*read the file \\([^ ]*\\) .*/\\1/p')\n" +
		"m=$(echo \"$prompt\" | sed -n 's/.*touch \\([^ ]*\\) .*/\\1/p')\n[ -n \"$m\" ] && touch \"$m\"\n" +
		"echo '{\"type\":\"result\",\"is_error\":false,\"result\":\"done\"}'\n"
	os.WriteFile(p, []byte(script), 0o700)
	c, _ := New(Config{Name: "claude-code", Kind: ClaudeCode, Path: p}, nil, egress.New(nil))
	_, err := ask(c)
	var e *omnillm.Error
	if !errors.As(err, &e) || e.Msg != omnillm.MsgCLIUnsafe {
		t.Fatalf("a program that ran a command was used: %v", err)
	}
	if err := c.Prepare(context.Background()); err == nil {
		t.Fatal("the failed version passed later")
	}
}

// Without a system text of its own, a chat says what it is: no tools, the
// person's language (Claude Code otherwise calls itself an agent that works
// with files).
func TestCLIPromptDefaultSystem(t *testing.T) {
	sys, _ := cliPrompt(ClaudeCode, []omnillm.Message{{Role: "user", Content: "hi"}})
	if !strings.Contains(sys, "Claude") || !strings.Contains(sys, "no tools") {
		t.Fatalf("claude default: %q", sys)
	}
	sys, _ = cliPrompt(Codex, []omnillm.Message{{Role: "user", Content: "hi"}})
	if strings.Contains(sys, "Claude") || !strings.Contains(sys, "no tools") {
		t.Fatalf("codex default: %q", sys)
	}
	sys, _ = cliPrompt(ClaudeCode, []omnillm.Message{{Role: "system", Content: "Be terse."}, {Role: "user", Content: "hi"}})
	if sys != "Be terse." {
		t.Fatalf("own system text kept: %q", sys)
	}
}

func TestCLIProblemIsKeptUntilASuccess(t *testing.T) {
	dir := t.TempDir()
	mode := filepath.Join(dir, "mode")
	os.WriteFile(mode, []byte("login"), 0o600)
	c, _ := fakeCLI(t, `if [ "$(cat `+mode+`)" = login ]; then echo '{"type":"result","is_error":true,"result":"Please run /login"}'; exit 1; fi
`+okLines)
	if p, _ := c.Problem(); p != "" {
		t.Fatalf("problem before any call: %q", p)
	}
	s, _ := ask(c)
	collect(t, s)
	if p, _ := c.Problem(); p != ProblemLogin {
		t.Fatalf("after a signed-out reply: %q", p)
	}
	os.WriteFile(mode, []byte("ok"), 0o600)
	s, _ = ask(c)
	if text, err := collect(t, s); err != nil || text != "pong" {
		t.Fatal(text, err)
	}
	if p, _ := c.Problem(); p != "" {
		t.Fatalf("a success did not clear the problem: %q", p)
	}
}

func TestCLILimitIsAProblemWithItsReset(t *testing.T) {
	reset := time.Now().Add(3 * time.Hour).Unix()
	c, _ := fakeCLI(t, `echo '{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":`+itoa(reset)+`}}'
echo '{"type":"result","is_error":true,"result":"usage limit reached"}'; exit 1`)
	s, _ := ask(c)
	collect(t, s)
	p, at := c.Problem()
	if p != ProblemLimit || at.Before(time.Now().Add(2*time.Hour)) {
		t.Fatalf("%q %v", p, at)
	}
	// Refused while cooling down: still the limit, in the same words.
	_, err := ask(c)
	if e := omnillm.AsError(err); e.Msg != omnillm.MsgSubscriptionLimit {
		t.Fatalf("cooldown says %v", err)
	}
}

func TestCLIPromptTooLong(t *testing.T) {
	c, _ := fakeCLI(t, `echo '{"type":"result","is_error":true,"result":"Prompt is too long: 200429 tokens > 200000 maximum"}'; exit 1`)
	s, _ := ask(c)
	_, err := collect(t, s)
	e := omnillm.AsError(err)
	if e.Msg != omnillm.MsgContextTooLong {
		t.Fatalf("%v", err)
	}
	if got := omnillm.Describe("en", err); got != "This chat is too long for Claude Haiku. Start a new chat and ask again." {
		t.Fatal(got)
	}
	// Not a limit: the next call goes out.
	if _, err := ask(c); err != nil {
		t.Fatalf("too long paused the source: %v", err)
	}
}

func TestCodexHomeFailureIsNotALogin(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir()) // no auth.json there
	p := filepath.Join(t.TempDir(), "codex")
	os.WriteFile(p, []byte("#!/bin/sh\necho should-not-run\n"), 0o700)
	c, err := New(Config{Name: "codex", Kind: Codex, Path: p}, nil, egress.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	c.cli.passed, c.cli.checked = "test", time.Now()
	_, err = c.Stream(context.Background(), "default", omnillm.Request{Caller: omnillm.Interactive,
		Messages: []omnillm.Message{{Role: "user", Content: "hi"}}}, Credential{})
	if e := omnillm.AsError(err); e.Msg != omnillm.MsgUpstream {
		t.Fatalf("%v", err)
	}
	if p, _ := c.Problem(); p != "" {
		t.Fatalf("problem %q", p)
	}
}
