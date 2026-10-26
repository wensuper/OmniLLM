package omnillm

import (
	"errors"
	"regexp"
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
	space := &Error{Code: CodeBusy, Msg: MsgSubscriptionPace, RetryAfter: 20 * time.Minute}
	if got := Describe("zh", space); !strings.Contains(got, "约 20 分钟后再试") {
		t.Fatal(got)
	}
	up := &Error{Code: CodeBusy, Msg: MsgBusyUpstream, Args: []string{"openai"}, RetryAfter: 30 * time.Second}
	if got := Describe("en", up); got != "OpenAI is busy right now. Try again in about 30 seconds." {
		t.Fatal(got)
	}
	if got := Describe("zh", up); got != "OpenAI 那边正忙，约 30 秒后再试。" {
		t.Fatal(got)
	}
	pace := &Error{Code: CodeBusy, Msg: MsgBusyPace, Args: []string{"codex"}, RetryAfter: time.Minute}
	if got := Describe("zh", pace); got != "订阅按一个人聊天的节奏使用，两次请求之间要隔开一点。约 60 秒后再试。" {
		t.Fatal(got)
	}
	if got := Describe("en", pace); got != "Subscriptions are used at one person’s chatting pace, so requests are spaced out. Try again in about 60 seconds." {
		t.Fatal(got)
	}
	// The subscription is still sent first; it never fills the wait.
	if got := Describe("en", &Error{Code: CodeBusy, Msg: MsgBusyPace, Args: []string{"codex", "1 minute"}}); !strings.HasSuffix(got, "Try again in 1 minute.") {
		t.Fatal(got)
	}
}

func TestSubscriptionNamesAndLoginSteps(t *testing.T) {
	login := &Error{Code: CodeCredentialProblem, Msg: MsgCLILogin, Args: []string{"claude-code"}}
	if got := Describe("zh", login); got != "你的电脑上的 Claude Code 没有登录或登录已过期。在那台电脑的终端里运行 omnillm login claude，然后再试。" {
		t.Fatal(got)
	}
	if got := Describe("en", login); got != "Claude Code on your computer is signed out. In a terminal on that computer, run omnillm login claude, then try again." {
		t.Fatal(got)
	}
	codex := &Error{Code: CodeCredentialProblem, Msg: MsgCLILogin, Args: []string{"codex"}}
	if got := Describe("en", codex); got != "Codex on your computer is signed out. In a terminal on that computer, run omnillm login chatgpt, then try again." {
		t.Fatal(got)
	}
	// An older hub's display name and program still render well.
	old := &Error{Code: CodeCredentialProblem, Msg: MsgCLILogin, Args: []string{"ChatGPT Codex (subscription)", "codex"}}
	if got := Describe("zh", old); got != "你的电脑上的 Codex 没有登录或登录已过期。在那台电脑的终端里运行 omnillm login chatgpt，然后再试。" {
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

func TestSourceAndKeyMessages(t *testing.T) {
	for _, c := range []struct{ key, zh, en string }{
		{MsgBadSource, "名字只能用英文字母、数字和 -（不要空格）；接口地址要以 https:// 开头。", "Use letters, digits and - for the name (no spaces), and an address starting with https://."},
		{MsgKeyMalformed, "这不像完整的 API Key。到服务商网站重新复制一次，再粘贴。", "That doesn’t look like a complete API key. Copy it again from the provider’s site and paste it."},
		{MsgNoChatModel, "这台 Mac 上的 Ollama 还没有能聊天的模型。想用的话，在终端运行 ollama pull qwen3。", "Ollama on this Mac has no chat model yet. To use it, run ollama pull qwen3 in Terminal."},
		{MsgPairingExpired, "这个二维码或配对文字已过期或已用过。换一个新的再试（控制台里点“添加设备”，或再运行 omnillm pair）。", "This QR code or pairing text expired or was already used. Get a new one (“Add device” in the console, or run omnillm pair again)."},
		{MsgPairingFailed, "这个二维码或配对文字已过期或已用过。换一个新的再试（控制台里点“添加设备”，或再运行 omnillm pair）。", "This QR code or pairing text expired or was already used. Get a new one (“Add device” in the console, or run omnillm pair again)."},
	} {
		if got := Text("zh", c.key); got != c.zh {
			t.Errorf("%s zh: %q", c.key, got)
		}
		if got := Text("en", c.key); got != c.en {
			t.Errorf("%s en: %q", c.key, got)
		}
	}
	if s := Text("zh", MsgRegionChanged); !strings.Contains(s, "已暂停使用模型") {
		t.Fatal(s)
	}
}

// zh: a space between a Latin word (or a placeholder) and Chinese; en:
// contractions, as people write.
var (
	latinThenHan = regexp.MustCompile(`[A-Za-z0-9%][\p{Han}]|[\p{Han}][A-Za-z0-9]`)
	uncontracted = regexp.MustCompile(`\b(does not|do not|is not|are not|cannot|could not|did not|was not|will not|would not|has not|have not)\b`)
)

func TestSpacingAndContractions(t *testing.T) {
	for k, v := range texts {
		// "%s后" is left: the waits it takes end in Chinese ("约 3 小时").
		zh := strings.ReplaceAll(v[0], "%s后", "")
		if m := latinThenHan.FindString(zh); m != "" {
			t.Errorf("%s zh: no space in %q (%q)", k, m, v[0])
		}
		if m := uncontracted.FindString(v[1]); m != "" {
			t.Errorf("%s en: %q", k, m)
		}
	}
}

// A hub that isn't a Mac is "your computer" in the program's messages.
func TestForHub(t *testing.T) {
	en := Text("en", MsgBudgetDevice)
	if s := ForHub("en", "darwin", en); s != en {
		t.Fatal(s)
	}
	if s := ForHub("en", "linux", en); strings.Contains(s, "Mac") || !strings.Contains(s, "your computer") {
		t.Fatal(s)
	}
	if s := ForHub("zh", "", Text("zh", MsgBudgetDevice)); strings.Contains(s, "Mac") || strings.Contains(s, "电脑 ") || !strings.Contains(s, "你的电脑上") {
		t.Fatal(s)
	}
}
