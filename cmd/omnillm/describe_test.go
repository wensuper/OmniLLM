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
