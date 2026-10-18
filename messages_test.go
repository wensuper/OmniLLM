package omnillm

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEveryMessageHasBothLanguagesAndMatchingPlaceholders(t *testing.T) {
	for k, v := range texts {
		if v[0] == "" || v[1] == "" {
			t.Fatalf("%s: missing a language", k)
		}
		if strings.Count(v[0], "%s") != strings.Count(v[1], "%s") {
			t.Fatalf("%s: placeholder count differs", k)
		}
		for _, bad := range []string{"relay", "hub_", "noise", "mailbox", "token hash", "_error"} {
			if strings.Contains(strings.ToLower(v[1]), bad) && k != MsgRelayUnavailable {
				t.Fatalf("%s: internal word %q in text a person sees", k, bad)
			}
		}
	}
}

func TestDescribeNeverPrintsMissing(t *testing.T) {
	for k := range texts {
		for _, lang := range []string{"zh", "en"} {
			s := Describe(lang, &Error{Code: CodeBusy, Msg: k})
			if strings.Contains(s, "MISSING") || strings.Contains(s, "%!") {
				t.Fatalf("%s/%s: %q", k, lang, s)
			}
		}
	}
	if Describe("en", errors.New("raw sk-secret")) != Text("en", msgInternal) {
		t.Fatal("non-library errors must render as the internal message")
	}
}

func TestEveryCodeHasAStatus(t *testing.T) {
	for _, c := range []Code{CodeBadRequest, CodeUnsupportedFormat, CodeInvalidToken, CodeModelNotAllowed, CodeBusy,
		CodeBudgetExceeded, CodeHubOffline, CodeUpstream, CodeCredentialProblem, CodeRegionNotAllowed, CodeNotFound, CodeForbiddenOrigin} {
		if HTTPStatus(c) == 500 {
			t.Fatalf("%s maps to 500", c)
		}
	}
	if HTTPStatus(CodeModelNotAllowed) != 403 || HTTPStatus(CodeBusy) != 429 || HTTPStatus(CodeHubOffline) != 503 || HTTPStatus(CodeInvalidToken) != 401 {
		t.Fatal("status table (design §6) changed")
	}
}

func TestValidate(t *testing.T) {
	ok := Request{Model: "a/b", Messages: []Message{{Role: "user", Content: "hi"}}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	tmp := 3.0
	bad := []Request{
		{},
		{Model: "a/b"},
		{Model: "a/b", Messages: []Message{{Role: "tool", Content: "x"}}},
		{Model: "a/b", Messages: []Message{{Role: "user", Content: strings.Repeat("x", MaxRequestBytes+1)}}},
		{Model: "a/b", Messages: []Message{{Role: "user"}}, MaxTokens: -1},
		{Model: "a/b", Messages: []Message{{Role: "user"}}, Temperature: &tmp},
		{Model: "a/b", Messages: []Message{{Role: "user"}}, JSONSchema: []byte("{")},
		{Model: "a/b", Messages: []Message{{Role: "user"}}, Caller: 7},
		{Model: strings.Repeat("m", 300), Messages: []Message{{Role: "user"}}},
	}
	for i, r := range bad {
		if r.Validate() == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
}

func TestErrorsIs(t *testing.T) {
	e := &Error{Code: CodeBusy, Msg: MsgBusyQueue, RetryAfter: time.Second}
	if !errors.Is(e, ErrBusy) || errors.Is(e, ErrHubOffline) {
		t.Fatal("errors.Is by code")
	}
}

func TestRoundDuration(t *testing.T) {
	if RoundDuration("en", 30*time.Second) != "30 seconds" || RoundDuration("zh", 5*time.Minute) != "5 分钟" || RoundDuration("en", 0) != "1 seconds" {
		t.Fatal(RoundDuration("en", 30*time.Second), RoundDuration("zh", 5*time.Minute))
	}
}
