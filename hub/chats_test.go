package hub_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/provider"
)

// addSessionClaude adds a stand-in claude that keeps sessions where the
// real one does (~/.claude/projects/<its folder, dashed>/<id>.jsonl) and
// records each call's arguments.
func addSessionClaude(t *testing.T, e *env) (rec string) {
	t.Helper()
	dir := t.TempDir()
	rec = filepath.Join(dir, "rec")
	script := `#!/bin/sh
if [ "$1" = auth ]; then echo '{"loggedIn":true}'; exit 0; fi
echo "$*" >> ` + rec + `
cat >/dev/null
sess="$HOME/.claude/projects/$(pwd -P | sed 's/[^a-zA-Z0-9]/-/g')"
prev=""
for a in "$@"; do
  if [ "$prev" = --session-id ]; then mkdir -p "$sess"; echo x > "$sess/$a.jsonl"; fi
  if [ "$prev" = --resume ]; then [ -f "$sess/$a.jsonl" ] || exit 1; fi
  prev="$a"
done
echo '{"type":"result","is_error":false,"result":"sub-pong","usage":{"input_tokens":1,"output_tokens":1}}'
`
	p := filepath.Join(dir, "claude")
	if err := os.WriteFile(p, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := e.hub.AddProvider(context.Background(), provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: p}, ""); err != nil {
		t.Fatal(err)
	}
	return rec
}

func chat(ctx context.Context, c interface {
	Complete(context.Context, omnillm.Request) (omnillm.Stream, error)
}, msgs ...omnillm.Message) error {
	s, err := c.Complete(ctx, omnillm.Request{Model: "claude-code/haiku", Messages: msgs, Caller: omnillm.Interactive})
	if err != nil {
		return err
	}
	defer s.Close()
	for {
		if _, err := s.Recv(); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func lastArgs(rec string) string {
	b, _ := os.ReadFile(rec)
	l := strings.Split(strings.TrimSpace(string(b)), "\n")
	return l[len(l)-1]
}

// sessionFiles waits for the kept sessions to settle at n files.
func sessionFiles(t *testing.T, n int) []string {
	t.Helper()
	var m []string
	for i := 0; i < 200; i++ {
		m, _ = filepath.Glob(filepath.Join(os.Getenv("HOME"), ".claude", "projects", "*", "*.jsonl"))
		if len(m) == n {
			return m
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session files: %v, want %d", m, n)
	return nil
}

func TestDeviceChatContinuesUntilTheDeviceIsRemoved(t *testing.T) {
	e := setup(t, nil)
	rec := addSessionClaude(t, e)
	d, _ := e.pairOK()
	ctx := context.Background()
	u := func(s string) omnillm.Message { return omnillm.Message{Role: "user", Content: s} }
	a := func(s string) omnillm.Message { return omnillm.Message{Role: "assistant", Content: s} }

	if err := chat(ctx, d, u("first")); err != nil {
		t.Fatal(err)
	}
	sessionFiles(t, 1)
	if err := chat(ctx, d, u("first"), a("sub-pong"), u("second")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lastArgs(rec), "--resume ") {
		t.Fatalf("the follow-up did not continue its session: %s", lastArgs(rec))
	}
	// The hub itself (NEXA) has its own conversations.
	if err := chat(ctx, e.hub, u("first"), a("sub-pong"), u("second"), a("sub-pong"), u("third")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(lastArgs(rec), "--resume ") {
		t.Fatalf("the hub continued the device's session: %s", lastArgs(rec))
	}
	sessionFiles(t, 2)

	devs, _ := e.hub.Devices(ctx)
	if err := e.hub.Revoke(ctx, devs[0].ID); err != nil {
		t.Fatal(err)
	}
	sessionFiles(t, 1) // the hub's own stays

	if err := e.hub.RemoveProvider(ctx, "claude-code"); err != nil {
		t.Fatal(err)
	}
	sessionFiles(t, 0)
}
