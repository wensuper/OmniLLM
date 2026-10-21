package provider

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/egress"
)

// chatHome points every folder the chats touch into the test's own.
func chatHome(t *testing.T) {
	t.Helper()
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("OMNILLM_HOME", filepath.Join(d, "omnillm"))
	src := filepath.Join(d, "codex-src")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "auth.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", src)
}

// fakeClaudeChat is a stand-in claude that keeps sessions where the real
// one does: --session-id creates the file, --resume needs it. Each call's
// arguments and input are appended to the record.
func fakeClaudeChat(t *testing.T, failResume bool) (*Client, string) {
	t.Helper()
	dir := t.TempDir()
	rec := filepath.Join(dir, "rec")
	fail := ":"
	if failResume {
		fail = "exit 3"
	}
	script := `#!/bin/sh
if [ "$1" = auth ]; then echo '{"loggedIn":true}'; exit 0; fi
echo "ARGS $* PWD $(pwd)" >> ` + rec + `
in=$(cat); echo "IN $in" >> ` + rec + `
sess="$HOME/.claude/projects/$(pwd -P | sed 's/[^a-zA-Z0-9]/-/g')"
prev=""
for a in "$@"; do
  if [ "$prev" = --session-id ]; then mkdir -p "$sess"; echo x > "$sess/$a.jsonl"; fi
  if [ "$prev" = --resume ]; then [ -f "$sess/$a.jsonl" ] || { echo "No conversation found" >&2; exit 1; }; ` + fail + `; fi
  prev="$a"
done
cat <<'X'
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"po"}}}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"ng"}}}
{"type":"result","is_error":false,"result":"pong","usage":{"input_tokens":3,"output_tokens":2}}
X
`
	p := filepath.Join(dir, "claude")
	if err := os.WriteFile(p, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{Name: "claude-code", Kind: ClaudeCode, Path: p}, nil, egress.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	c.cli.passed, c.cli.checked = "test", time.Now()
	return c, rec
}

func chatAsk(t *testing.T, c *Client, chats *Chats, owner string, msgs ...omnillm.Message) (string, error) {
	t.Helper()
	s, err := c.StreamChat(context.Background(), "haiku", omnillm.Request{Caller: omnillm.Interactive, Messages: msgs}, Credential{}, chats, owner)
	if err != nil {
		return "", err
	}
	defer s.Close()
	var b strings.Builder
	for {
		ch, err := s.Recv()
		if err == io.EOF {
			return b.String(), nil
		}
		if err != nil {
			return b.String(), err
		}
		b.WriteString(ch.Text)
	}
}

// settled waits until the table holds n kept (not pending) sessions:
// keeping happens once the program has exited.
func settled(t *testing.T, chats *Chats, n int) []ChatRecord {
	t.Helper()
	for i := 0; i < 200; i++ {
		chats.mu.Lock()
		var kept []ChatRecord
		pend := 0
		for _, r := range chats.recs {
			if isPending(r) {
				pend++
			} else {
				kept = append(kept, r)
			}
		}
		chats.mu.Unlock()
		if len(kept) == n && pend == 0 {
			return kept
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("table never settled at %d kept", n)
	return nil
}

// claudeFiles waits until n Claude session files exist.
func claudeFiles(t *testing.T, n int) []string {
	t.Helper()
	var m []string
	for i := 0; i < 200; i++ {
		m, _ = filepath.Glob(filepath.Join(os.Getenv("HOME"), ".claude", "projects", "*", "*.jsonl"))
		if len(m) == n {
			return m
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session files %v, want %d", m, n)
	return nil
}

func lastCall(t *testing.T, rec string) (args, in string) {
	t.Helper()
	b, _ := os.ReadFile(rec)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "ARGS ") {
			args = l
		} else if strings.HasPrefix(l, "IN ") {
			in = strings.TrimPrefix(l, "IN ")
		}
	}
	return args, in
}

func u(s string) omnillm.Message { return omnillm.Message{Role: "user", Content: s} }
func a(s string) omnillm.Message { return omnillm.Message{Role: "assistant", Content: s} }

func TestClaudeChatContinuesItsSession(t *testing.T) {
	chatHome(t)
	c, rec := fakeClaudeChat(t, false)
	var saved [][]ChatRecord
	var mu sync.Mutex
	chats := NewChats(nil, func(r []ChatRecord) { mu.Lock(); saved = append(saved, r); mu.Unlock() })

	if got, err := chatAsk(t, c, chats, "device/A", u("first")); err != nil || got != "pong" {
		t.Fatalf("turn 1: %q %v", got, err)
	}
	args, _ := lastCall(t, rec)
	if !strings.Contains(args, "--session-id ") || strings.Contains(args, "--no-session-persistence") {
		t.Fatalf("turn 1 should start a kept session: %s", args)
	}
	if real, _ := filepath.EvalSymlinks(claudeChatDir()); !strings.Contains(args, "PWD "+real) {
		t.Fatalf("turn 1 should run in the hub's chat folder: %s", args)
	}
	kept := settled(t, chats, 1)
	id := kept[0].ID

	// The follow-up sends only its new message into the same session.
	if _, err := chatAsk(t, c, chats, "device/A", u("first"), a("pong"), u("second")); err != nil {
		t.Fatal(err)
	}
	args, in := lastCall(t, rec)
	if !strings.Contains(args, "--resume "+id) || in != "second" {
		t.Fatalf("turn 2 should resume %s with only the new message: %s / %q", id, args, in)
	}
	settled(t, chats, 1)

	// An earlier message changed, the end the same: a new session from
	// the whole transcript.
	if _, err := chatAsk(t, c, chats, "device/A", u("first!"), a("pong"), u("second"), a("pong"), u("third")); err != nil {
		t.Fatal(err)
	}
	args, in = lastCall(t, rec)
	if strings.Contains(args, "--resume") || !strings.Contains(in, "The conversation so far") {
		t.Fatalf("an edited conversation must start afresh: %s / %q", args, in)
	}
	settled(t, chats, 2)

	// Another device never continues device A's conversation.
	if _, err := chatAsk(t, c, chats, "device/B", u("first"), a("pong"), u("second"), a("pong"), u("third")); err != nil {
		t.Fatal(err)
	}
	if args, _ = lastCall(t, rec); strings.Contains(args, "--resume") {
		t.Fatalf("device B resumed device A's session: %s", args)
	}
	settled(t, chats, 3)

	// The saved table holds ids and hashes only.
	mu.Lock()
	last := saved[len(saved)-1]
	mu.Unlock()
	for _, r := range last {
		if strings.Contains(r.Key, "first") || strings.Contains(r.Key, "pong") {
			t.Fatalf("text in the saved table: %+v", r)
		}
	}

	// A removed device's sessions go, files included.
	chats.Forget("device/B")
	settled(t, chats, 2)
	claudeFiles(t, 2)
}

func TestClaudeChatFailedTurnLeavesNothingToContinue(t *testing.T) {
	chatHome(t)
	c, rec := fakeClaudeChat(t, true)
	chats := NewChats(nil, nil)
	if _, err := chatAsk(t, c, chats, "host", u("first")); err != nil {
		t.Fatal(err)
	}
	id := settled(t, chats, 1)[0].ID
	if _, err := chatAsk(t, c, chats, "host", u("first"), a("pong"), u("second")); err == nil {
		t.Fatal("the resumed turn should fail")
	}
	settled(t, chats, 0)
	claudeFiles(t, 0)
	_ = id
	// The retry starts afresh from the transcript.
	if _, err := chatAsk(t, c, chats, "host", u("first"), a("pong"), u("second")); err != nil {
		t.Fatal(err)
	}
	if args, _ := lastCall(t, rec); strings.Contains(args, "--resume") {
		t.Fatalf("retry resumed a discarded session: %s", args)
	}
}

func TestClaudeChatSessionFileGoneStartsAfresh(t *testing.T) {
	chatHome(t)
	c, rec := fakeClaudeChat(t, false)
	chats := NewChats(nil, nil)
	if _, err := chatAsk(t, c, chats, "host", u("first")); err != nil {
		t.Fatal(err)
	}
	id := settled(t, chats, 1)[0].ID
	for _, f := range claudeSessionFiles(id) {
		os.Remove(f)
	}
	if _, err := chatAsk(t, c, chats, "host", u("first"), a("pong"), u("second")); err != nil {
		t.Fatalf("a missing session file must not fail the turn: %v", err)
	}
	if args, _ := lastCall(t, rec); strings.Contains(args, "--resume") {
		t.Fatalf("resumed a session whose file is gone: %s", args)
	}
}

func TestChatsSweepAndLimits(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	var removed []string
	var mu sync.Mutex
	chats := NewChats(func() time.Time { return now }, nil)
	chats.remove = func(r ChatRecord) { mu.Lock(); removed = append(removed, r.ID); mu.Unlock() }
	id := func(i int) string {
		return "00000000-0000-4000-8000-" + strings.Repeat("0", 8) + string(rune('a'+i/16)) + string("0123456789abcdef"[i%16]) + "00"
	}

	// Per owner: the oldest goes when a 21st is kept.
	for i := 0; i < chatsPerOwner+1; i++ {
		now = now.Add(time.Minute)
		chats.put(ChatRecord{Key: "k" + id(i), Kind: ClaudeCode, ID: id(i), Owner: "device/A"}, 0, 0)
	}
	if chats.Len() != chatsPerOwner {
		t.Fatalf("kept %d for one device", chats.Len())
	}
	// A pending one (a turn still running) is never the one dropped.
	chats.pending(ChatRecord{Kind: Codex, ID: id(40), Owner: "device/A"})
	now = now.Add(time.Minute)
	chats.put(ChatRecord{Key: "k" + id(41), Kind: ClaudeCode, ID: id(41), Owner: "device/A"}, 0, 0)
	chats.mu.Lock()
	_, stillPending := chats.recs[pendingKey(id(40))]
	chats.mu.Unlock()
	if !stillPending {
		t.Fatal("a running turn's session was evicted")
	}
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	sort.Strings(removed)
	if len(removed) != 2 || removed[0] != id(0) || removed[1] != id(1) {
		t.Fatalf("evicted %v", removed)
	}
	removed = nil
	mu.Unlock()

	// The same point reached by a newer session: the older one goes.
	now = now.Add(time.Minute)
	chats.put(ChatRecord{Key: "k" + id(41), Kind: ClaudeCode, ID: id(42), Owner: "device/A"}, 0, 0)
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if len(removed) != 1 || removed[0] != id(41) {
		t.Fatalf("replaced session not removed: %v", removed)
	}
	removed = nil
	mu.Unlock()

	// A device removed while its turn ran: that turn keeps nothing.
	og, kg := chats.gens("device/C", ClaudeCode)
	chats.Forget("device/C")
	chats.put(ChatRecord{Key: "kc", Kind: ClaudeCode, ID: id(43), Owner: "device/C"}, og, kg)
	time.Sleep(50 * time.Millisecond)
	chats.mu.Lock()
	_, kept := chats.recs["kc"]
	chats.mu.Unlock()
	mu.Lock()
	if kept || len(removed) != 1 || removed[0] != id(43) {
		t.Fatalf("a removed device's late turn: kept %v, removed %v", kept, removed)
	}
	removed = nil
	mu.Unlock()

	// A day later everything unused goes; the pending one after 15 minutes.
	now = now.Add(ChatIdle + time.Minute)
	chats.Sweep()
	if chats.Len() != 0 {
		t.Fatalf("after a day: %d left", chats.Len())
	}

	// A saved table with junk loads only well-formed records.
	chats.Load([]ChatRecord{{Key: "x", Kind: ClaudeCode, ID: "../../etc"}, {Key: "y", Kind: "openai", ID: id(1)}, {Key: "z", Kind: Codex, ID: id(2)}})
	if chats.Len() != 1 {
		t.Fatalf("loaded %d", chats.Len())
	}
}

func TestChatKeyFollowsTheConversation(t *testing.T) {
	img := omnillm.Message{Role: "user", Content: "look", Images: []omnillm.Image{{MIME: "image/png", Data: []byte{1, 2, 3}}}}
	history := []omnillm.Message{u("hi"), a("hello"), img, a("a cat")}
	k, ok := followUpKey("device/A", ClaudeCode, "sonnet", "", append(history, u("more")))
	if !ok || k != chatKey("device/A", ClaudeCode, "sonnet", "", history) {
		t.Fatal("follow-up key differs from the key kept after the turn")
	}
	// Later the web client names that picture instead of sending it: still
	// the same conversation.
	later := []omnillm.Message{u("hi"), a("hello"), {Role: "user", Content: "look\n[1 earlier image(s)]"}, a("a cat"), u("more"), a("ok")}
	k1 := chatKey("device/A", ClaudeCode, "sonnet", "", append(append([]omnillm.Message{}, history...), u("more"), a("ok")))
	if k2, _ := followUpKey("device/A", ClaudeCode, "sonnet", "", append(later, u("next"))); k2 != k1 {
		t.Fatal("an image named instead of sent breaks the conversation")
	}
	other := omnillm.Message{Role: "user", Content: "look", Images: []omnillm.Image{{MIME: "image/png", Data: []byte{1, 2, 4}}}}
	for name, k2 := range map[string]string{
		"owner":   chatKey("device/B", ClaudeCode, "sonnet", "", history),
		"model":   chatKey("device/A", ClaudeCode, "opus", "", history),
		"kind":    chatKey("device/A", Codex, "sonnet", "", history),
		"system":  chatKey("device/A", ClaudeCode, "sonnet", "be brief", history),
		"image":   chatKey("device/A", ClaudeCode, "sonnet", "", []omnillm.Message{u("hi"), a("hello"), other, a("a cat")}),
		"reply":   chatKey("device/A", ClaudeCode, "sonnet", "", []omnillm.Message{u("hi"), a("hello"), img, a("a dog")}),
		"earlier": chatKey("device/A", ClaudeCode, "sonnet", "", []omnillm.Message{u("hi!"), a("hello"), img, a("a cat")}),
		"answer":  chatKey("device/A", ClaudeCode, "sonnet", "", []omnillm.Message{u("hi"), a("hey"), img, a("a cat")}),
		"shorter": chatKey("device/A", ClaudeCode, "sonnet", "", history[2:]),
	} {
		if k2 == k {
			t.Errorf("%s does not change the key", name)
		}
	}
	if _, ok := followUpKey("h", ClaudeCode, "", "", []omnillm.Message{u("x")}); ok {
		t.Error("a first message is not a follow-up")
	}
	if !validChatID(newChatID()) || validChatID("01A15C95-c87a-7563-8013-c6631b41a089") || validChatID("../x") {
		t.Error("chat id check")
	}
}

// fakeCodexChat is a stand-in codex that keeps rollout files where the
// real one does and deletes them on "delete --force".
func fakeCodexChat(t *testing.T) (*Client, string) {
	t.Helper()
	dir := t.TempDir()
	rec := filepath.Join(dir, "rec")
	script := `#!/bin/sh
echo "ARGS $*" >> ` + rec + `
case "$*" in *"login status"*) echo "Logged in using ChatGPT"; exit 0;; esac
case "$*" in *"delete --force"*) for a in "$@"; do last="$a"; done; rm -f "$CODEX_HOME"/sessions/*/*/*/rollout-*-"$last".jsonl; exit 0;; esac
in=$(cat); echo "IN $in" >> ` + rec + `
id=11111111-2222-4333-8444-555555555555
case "$*" in *"exec resume"*) for a in "$@"; do [ "$a" = - ] && break; id="$a"; done;; esac
mkdir -p "$CODEX_HOME/sessions/2026/10/21"; echo x > "$CODEX_HOME/sessions/2026/10/21/rollout-2026-10-21T11-51-22-$id.jsonl"
echo '{"type":"thread.started","thread_id":"'$id'"}'
echo '{"type":"item.completed","item":{"type":"agent_message","text":"pong"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":3,"output_tokens":1}}'
`
	p := filepath.Join(dir, "codex")
	if err := os.WriteFile(p, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{Name: "codex", Kind: Codex, Path: p}, nil, egress.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	c.cli.passed, c.cli.checked = "test", time.Now()
	return c, rec
}

func TestCodexChatContinuesItsThread(t *testing.T) {
	chatHome(t)
	c, rec := fakeCodexChat(t)
	chats := NewChats(nil, nil)
	if _, err := chatAsk(t, c, chats, "device/A", u("first")); err != nil {
		t.Fatal(err)
	}
	args, _ := lastCall(t, rec)
	if strings.Contains(args, "--ephemeral") || !strings.Contains(args, "--sandbox read-only") {
		t.Fatalf("turn 1: %s", args)
	}
	kept := settled(t, chats, 1)
	if kept[0].ID != "11111111-2222-4333-8444-555555555555" {
		t.Fatalf("thread id: %+v", kept[0])
	}
	if _, err := chatAsk(t, c, chats, "device/A", u("first"), a("pong"), u("second")); err != nil {
		t.Fatal(err)
	}
	args, in := lastCall(t, rec)
	if !strings.Contains(args, "exec resume") || !strings.Contains(args, `sandbox_mode="read-only"`) ||
		!strings.Contains(args, "features.shell_tool=false") || !strings.HasSuffix(args, kept[0].ID+" -") || in != "second" {
		t.Fatalf("turn 2: %s / %q", args, in)
	}
	settled(t, chats, 1)
	chats.Forget("device/A")
	for i := 0; i < 200 && len(codexSessionFiles(kept[0].ID)) > 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if files := codexSessionFiles(kept[0].ID); len(files) != 0 {
		t.Fatalf("Codex session left after Forget: %v", files)
	}
	if args, _ := lastCall(t, rec); !strings.Contains(args, "delete --force "+kept[0].ID) {
		t.Fatalf("Codex was not asked to delete its session: %s", args)
	}
}

func TestBackgroundCallsKeepNothing(t *testing.T) {
	chatHome(t)
	for _, k := range []Kind{ClaudeCode, Codex} {
		var c *Client
		var rec string
		if k == ClaudeCode {
			c, rec = fakeClaudeChat(t, false)
		} else {
			c, rec = fakeCodexChat(t)
		}
		chats := NewChats(nil, nil)
		s, err := c.StreamChat(context.Background(), "haiku", omnillm.Request{Caller: omnillm.Background, Messages: []omnillm.Message{u("x")}}, Credential{}, chats, "host")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := collect(t, s); err != nil {
			t.Fatal(err)
		}
		s.Close()
		args, _ := lastCall(t, rec)
		if !strings.Contains(args, "--ephemeral") && !strings.Contains(args, "--no-session-persistence") {
			t.Fatalf("%s: a background call kept a session: %q", k, args)
		}
		time.Sleep(50 * time.Millisecond)
		if chats.Len() != 0 {
			t.Fatalf("%s: a background call is in the table", k)
		}
	}
}

func TestCodexEarlyStopStillDeletesItsSession(t *testing.T) {
	chatHome(t)
	c, _ := fakeCodexChat(t)
	chats := NewChats(nil, nil)
	s, err := c.StreamChat(context.Background(), "default", omnillm.Request{Caller: omnillm.Interactive, Messages: []omnillm.Message{u("x")}}, Credential{}, chats, "host")
	if err != nil {
		t.Fatal(err)
	}
	s.Close() // before reading anything
	for i := 0; i < 200 && len(codexSessionFiles("11111111-2222-4333-8444-555555555555")) > 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if f := codexSessionFiles("11111111-2222-4333-8444-555555555555"); len(f) > 0 {
		t.Fatalf("a turn stopped at once left its session: %v", f)
	}
	if chats.Len() != 0 {
		t.Fatal("a stopped turn is in the table")
	}
}
