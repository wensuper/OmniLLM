package hub_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/provider"
)

func addFakeClaude(t *testing.T, e *env) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "claude")
	script := "#!/bin/sh\nif [ \"$1\" = auth ]; then echo '{\"loggedIn\":true}'; exit 0; fi\ncat >/dev/null\n" +
		`echo '{"type":"result","is_error":false,"result":"sub-pong","usage":{"input_tokens":1,"output_tokens":1}}'` + "\n"
	if err := os.WriteFile(p, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	ms, err := e.hub.AddProvider(context.Background(), provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: p}, "")
	if err != nil || len(ms) != 3 {
		t.Fatal(ms, err)
	}
	for _, m := range ms {
		if !m.HumanOnly {
			t.Fatalf("%s is not human-only", m.ID)
		}
	}
}

func TestSubscriptionIsForPeopleOnly(t *testing.T) {
	e := setup(t, nil)
	addFakeClaude(t, e)
	d, _ := e.pairOK()
	ctx := context.Background()
	if text, _, err := ask(ctx, d, "claude-code/haiku", "hi", omnillm.Interactive); err != nil || text != "sub-pong" {
		t.Fatalf("person: %q %v", text, err)
	}
	_, _, err := ask(ctx, d, "claude-code/haiku", "hi", omnillm.Background)
	var oe *omnillm.Error
	if !errors.As(err, &oe) || oe.Msg != omnillm.MsgHumanOnly {
		t.Fatalf("background got %v", err)
	}
	// The owner cannot lift it.
	no := false
	if err := e.hub.SetModel(ctx, "claude-code/haiku", hub.ModelUpdate{HumanOnly: &no}); err == nil {
		t.Fatal("human-only was turned off for a subscription")
	}
}

func TestSubscriptionPacePerDevice(t *testing.T) {
	var p hub.SubscriptionPaceForTest
	now := time.Now()
	for i := 0; i < 60; i++ {
		if err := p.Allow("phone", now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("call %d refused: %v", i, err)
		}
	}
	err := p.Allow("phone", now.Add(61*time.Second))
	var oe *omnillm.Error
	if !errors.As(err, &oe) || oe.Msg != omnillm.MsgSubscriptionPace || oe.RetryAfter <= 0 {
		t.Fatalf("61st call in an hour: %v", err)
	}
	if err := p.Allow("tablet", now.Add(61*time.Second)); err != nil {
		t.Fatal("another device was held back")
	}
	if err := p.Allow("phone", now.Add(time.Hour+time.Second)); err != nil {
		t.Fatalf("an hour later: %v", err)
	}
}
