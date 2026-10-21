package main

import (
	"errors"
	"strings"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
)

// Sentences written for people are shown as they are, never behind
// "something went wrong inside"; unexpected errors keep their cause.
func TestDescribe(t *testing.T) {
	if got := describe(sayf("already paired (%s)", "/x")); got != "already paired (/x)" {
		t.Fatalf("say: %q", got)
	}
	got := describe(errors.New("dial tcp: refused"))
	if !strings.Contains(got, "dial tcp: refused") || got == "dial tcp: refused" {
		t.Fatalf("internal: %q", got)
	}
	e := &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable, Detail: "x"}
	if got := describe(e); got != omnillm.Describe(lang(), e) {
		t.Fatalf("library error: %q", got)
	}
}

func TestHelpShortAndAll(t *testing.T) {
	for _, lang := range []string{"zh", "en"} {
		t.Setenv("OMNILLM_LANG", lang)
		short, all := usageText(false), usageText(true)
		for _, c := range []string{"omnillm install", "omnillm open", "omnillm status", "omnillm update", "omnillm uninstall [--purge]", "omnillm tray", "omnillm version", "omnillm help --all"} {
			if !strings.Contains(short, c) {
				t.Errorf("%s short: no %q", lang, c)
			}
		}
		if strings.Contains(short, "omnillm hub ") || strings.Contains(short, "omnillm relay ") {
			t.Errorf("%s short lists admin commands", lang)
		}
		for _, c := range []string{"omnillm tray", "omnillm version", "omnillm uninstall", "omnillm hub revoke", "omnillm hub app rm", "omnillm relay token"} {
			if !strings.Contains(all, c) {
				t.Errorf("%s all: no %q", lang, c)
			}
		}
		if strings.Contains(short+all, "连接服务") || strings.Contains(all, "<设备ID>") || strings.Contains(all, "<device-id>") {
			t.Errorf("%s: old wording", lang)
		}
	}
}

func TestConnectedText(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	if got := connectedText("groq", 1); got != "Connected groq: 1 model available." {
		t.Fatal(got)
	}
	if got := connectedText("groq", 3); got != "Connected groq: 3 models available." {
		t.Fatal(got)
	}
	t.Setenv("OMNILLM_LANG", "zh")
	if got := connectedText("groq", 1); got != "已接入 groq，1 个模型可用。" {
		t.Fatal(got)
	}
}
