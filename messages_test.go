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
		CodeBudgetExceeded, CodeHubOffline, CodeUpstream, CodeCredentialProblem, CodeRegionNotAllowed, CodeNotFound, CodeForbiddenOrigin, CodeDevicePaused} {
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
	for _, c := range []struct {
		lang string
		d    time.Duration
		want string
	}{
		{"en", 0, "about 1 second"},
		{"en", 30 * time.Second, "about 30 seconds"},
		{"zh", 30 * time.Second, "约 30 秒"},
		{"en", 100 * time.Second, "about 2 minutes"},
		{"zh", 20 * time.Minute, "约 20 分钟"},
		{"en", time.Minute + 30*time.Second, "about 2 minutes"},
		{"en", 89 * time.Minute, "about 89 minutes"},
		{"zh", 6 * time.Hour, "约 6 小时"},
		{"en", 3 * time.Hour, "about 3 hours"},
		{"zh-CN", 2*time.Hour + 50*time.Minute, "约 3 小时"},
	} {
		if got := RoundDuration(c.lang, c.d); got != c.want {
			t.Errorf("RoundDuration(%q, %v) = %q, want %q", c.lang, c.d, got, c.want)
		}
	}
}

func TestWaitsAreInTheReadersLanguage(t *testing.T) {
	e := &Error{Code: CodeBusy, Msg: MsgSubscriptionLimit, Args: []string{"claude-code"}, RetryAfter: 6 * time.Hour}
	if got := Describe("zh", e); got != "Claude 订阅的用量到了上限，约 6 小时后恢复。现在可以换一个用 API Key 或你的 Mac 上的模型。" {
		t.Fatal(got)
	}
	if got := Describe("en", e); got != "Your Claude subscription has reached its usage limit. It resets in about 6 hours. Meanwhile, pick a model that uses an API key or runs on your Mac." {
		t.Fatal(got)
	}
	// An older hub wrote the wait in English into Args; RetryAfter wins.
	old := &Error{Code: CodeBusy, Msg: MsgSubscriptionLimit, Args: []string{"Claude (subscription)", "360 minutes"}, RetryAfter: 6 * time.Hour}
	if got := Describe("zh", old); strings.Contains(got, "minutes") || !strings.Contains(got, "约 6 小时") || !strings.HasPrefix(got, "Claude 订阅") {
		t.Fatal(got)
	}
	pace := &Error{Code: CodeBusy, Msg: MsgSubscriptionPace, RetryAfter: 20 * time.Minute}
	if got := Describe("zh", pace); !strings.Contains(got, "约 20 分钟后再试") {
		t.Fatal(got)
	}
	up := &Error{Code: CodeBusy, Msg: MsgBusyUpstream, Args: []string{"openai"}, RetryAfter: 30 * time.Second}
	if got := Describe("en", up); got != "OpenAI asked us to wait. Try again in about 30 seconds." {
		t.Fatal(got)
	}
	if got := Describe("zh", &Error{Code: CodeBusy, Msg: MsgBusyPace, Args: []string{"codex"}, RetryAfter: time.Minute}); !strings.Contains(got, "为了让 ChatGPT 订阅的用量") || !strings.Contains(got, "约 60 秒后再试") {
		t.Fatal(got)
	}
}

func TestSubscriptionNamesAndLoginSteps(t *testing.T) {
	login := &Error{Code: CodeCredentialProblem, Msg: MsgCLILogin, Args: []string{"claude-code"}}
	if got := Describe("zh", login); got != "你的 Mac 上的 Claude Code 没有登录或登录已过期。在 Mac 的终端运行 claude，输入 /login，然后再试。" {
		t.Fatal(got)
	}
	if got := Describe("en", login); got != "Claude Code on your Mac is signed out. In Terminal on your Mac, run claude and type /login, then try again." {
		t.Fatal(got)
	}
	codex := &Error{Code: CodeCredentialProblem, Msg: MsgCLILogin, Args: []string{"codex"}}
	if got := Describe("en", codex); got != "Codex on your Mac is signed out. In Terminal on your Mac, run codex login, then try again." {
		t.Fatal(got)
	}
	// An older hub's display name and program still render well.
	old := &Error{Code: CodeCredentialProblem, Msg: MsgCLILogin, Args: []string{"ChatGPT Codex (subscription)", "codex"}}
	if got := Describe("zh", old); got != "你的 Mac 上的 Codex 没有登录或登录已过期。在 Mac 的终端运行 codex login，然后再试。" {
		t.Fatal(got)
	}
	nf := &Error{Code: CodeBadRequest, Msg: MsgCLINotFound, Args: []string{"claude-code"}}
	if got := Describe("zh", nf); got != "要用 Claude 订阅，你的 Mac 上需要先安装 Claude Code 并登录。" {
		t.Fatal(got)
	}
	if got := Describe("en", &Error{Code: CodeUpstream, Msg: MsgCLIUnsafe, Args: []string{"codex"}}); !strings.Contains(got, "This version of Codex on your Mac") {
		t.Fatal(got)
	}
	// Other names pass through unchanged.
	if got := Describe("en", &Error{Code: CodeCredentialProblem, Msg: MsgCredentialMissing, Args: []string{"my-proxy"}}); !strings.HasPrefix(got, "my-proxy isn’t connected on your Mac.") {
		t.Fatal(got)
	}
}

func TestMessagesUseTheGlossary(t *testing.T) {
	for k, v := range texts {
		for _, bad := range []string{"主机", "连接服务", "Codex"} {
			if strings.Contains(v[0], bad) {
				t.Errorf("%s zh: %q", k, bad)
			}
		}
		for _, bad := range []string{"the host", "The host", "budget", "Codex"} {
			if strings.Contains(v[1], bad) {
				t.Errorf("%s en: %q", k, bad)
			}
		}
		if strings.Contains(v[0], " Key") && !strings.Contains(v[0], "API Key") {
			t.Errorf("%s zh: Key without API", k)
		}
	}
}

func TestNewMessages(t *testing.T) {
	ctx := &Error{Code: CodeBadRequest, Msg: MsgContextTooLong, Args: []string{"Claude Sonnet"}}
	if got := Describe("zh", ctx); got != "这段对话太长，Claude Sonnet 装不下了。开一个新对话再问。" {
		t.Fatal(got)
	}
	if got := Describe("en", ctx); got != "This chat is too long for Claude Sonnet. Start a new chat and ask again." {
		t.Fatal(got)
	}
	if got := Describe("en", &Error{Code: CodeBudgetExceeded, Msg: MsgBudgetWouldExceed}); !strings.HasPrefix(got, "This message may exceed what’s left of today’s limit.") {
		t.Fatal(got)
	}
	if got := Describe("zh", &Error{Code: CodeModelNotAllowed, Msg: MsgHumanOnly}); !strings.Contains(got, "只能手动对话") {
		t.Fatal(got)
	}
	if got := Describe("en", &Error{Code: CodeBusy, Msg: MsgSubscriptionPace, RetryAfter: time.Minute}); !strings.HasSuffix(got, "so the account isn’t flagged as automated.") {
		t.Fatal(got)
	}
}

func TestEnglishUsesCurlyQuotes(t *testing.T) {
	for k, v := range texts {
		if strings.ContainsAny(v[1], "'\"") {
			t.Errorf("%s en: straight quote in %q", k, v[1])
		}
	}
}

func TestDeviceAndAppMessagesAreApart(t *testing.T) {
	for _, c := range []struct{ key, zh, en string }{
		{MsgBudgetDevice, "这台设备今天的额度用完了。明天恢复，也可以在你的 Mac 上调高。", "This device’s daily limit is used up. It resets tomorrow, or raise it on your Mac."},
		{MsgBudgetApp, "这个应用今天的额度用完了。明天恢复，也可以在你的 Mac 上调高。", "This app’s daily limit is used up. It resets tomorrow, or raise it on your Mac."},
		{MsgModelNotAllowed, "这台设备不能用这个模型。可以在你的 Mac 上调整它能用的模型。", "This device can’t use this model. You can change its models on your Mac."},
		{MsgModelNotAllowedApp, "这个应用不能用这个模型。可以在你的 Mac 上调整它能用的模型。", "This app can’t use this model. You can change its models on your Mac."},
	} {
		if got := Text("zh", c.key); got != c.zh {
			t.Errorf("%s zh: %q", c.key, got)
		}
		if got := Text("en", c.key); got != c.en {
			t.Errorf("%s en: %q", c.key, got)
		}
	}
	// Older devices know only the device keys, which stay as they were named.
	if MsgBudgetDevice != "budget_device" || MsgModelNotAllowed != "model_not_allowed" {
		t.Fatal("the device keys were renamed")
	}
	if got := Text("en", MsgPairingExpired); !strings.HasPrefix(got, "This QR code") || !strings.Contains(got, "“Add device”") {
		t.Fatal(got)
	}
	for _, k := range []string{MsgSubscriptionPace, MsgSubscriptionLimit} {
		if s := Text("zh", k); !strings.Contains(s, "你的 Mac 上的模型") || strings.Contains(s, "本机") {
			t.Errorf("%s zh: %q", k, s)
		}
	}
}
