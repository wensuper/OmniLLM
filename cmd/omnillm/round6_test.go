package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/control"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

// testHub is a hub with one local model behind a control socket, as
// `omnillm hub run` serves it.
func testHub(t *testing.T) (*hub.Hub, string, string) {
	t.Helper()
	t.Setenv("PATH", t.TempDir()) // no subscription programs here
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	tr := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"qwen3"}]}`)), Request: r}, nil
	})
	h, err := hub.New(ctx, hub.Options{Secrets: secret.NewMemory(), Store: store.NewMemory(), Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := provider.Default(provider.Ollama)
	ms, err := h.AddProvider(ctx, cfg, "")
	if err != nil || len(ms) != 1 {
		t.Fatal(ms, err)
	}
	dir, err := os.MkdirTemp("/tmp", "omni")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "c.sock")
	ln, err := control.Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	go control.Serve(ctx, ln, hubHandler(h, "", nil))
	return h, sock, ms[0].ID
}

// run sends one request and returns the last reply's text and data.
func run(sock string, req control.Request) (string, []byte, error) {
	var text string
	var data []byte
	err := control.Call(sock, req, func(r control.Reply) (bool, bool) {
		text, data = r.Text, r.Data
		return false, false
	})
	return text, data, err
}

// The hub answers in the language the command was typed in, not the one
// it was installed with.
func TestHubAnswersInTheCallersLanguage(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en") // the hub's own
	h, sock, _ := testHub(t)
	ctx := context.Background()
	if _, _, err := h.AddApp(ctx, "notes", nil, 0); err != nil {
		t.Fatal(err)
	}
	_, _, err := run(sock, control.Request{Op: "model.set", Lang: "zh", Args: map[string]string{"model": "nothing", "shown": "true"}})
	if err == nil || err.Error() != "没有叫“nothing”的模型。运行 omnillm hub models 查看。" {
		t.Fatal(err)
	}
	text, _, err := run(sock, control.Request{Op: "app.set", Lang: "zh", Args: map[string]string{"app": "notes", "paused": "true"}})
	if err != nil || text != "已更新“notes”。" {
		t.Fatal(text, err)
	}
	text, _, err = run(sock, control.Request{Op: "daily_total.set", Lang: "zh", Args: map[string]string{"micro_usd": "20000000"}})
	if err != nil || !strings.Contains(text, "每天的总额度：$20.00。") || strings.Contains(text, "Daily") {
		t.Fatal(text, err)
	}
	// An older terminal sends no language: the hub's own.
	text, _, err = run(sock, control.Request{Op: "app.set", Args: map[string]string{"app": "notes", "paused": "false"}})
	if err != nil || text != "Updated “notes”." {
		t.Fatal(text, err)
	}
	// The limit the hub writes into a row is in the caller's language.
	if _, _, err := run(sock, control.Request{Op: "app.set", Args: map[string]string{"app": "notes", "no_limit": "true"}}); err != nil {
		t.Fatal(err)
	}
	for lang, want := range map[string]string{"zh": "不设上限", "en": "No limit"} {
		_, data, _ := run(sock, control.Request{Op: "apps", Lang: lang})
		var rows []appRow
		if json.Unmarshal(data, &rows) != nil || len(rows) != 1 || rows[0].Budget != want {
			t.Fatalf("%s: %s", lang, data)
		}
	}
	// ctl sends the terminal's language.
	t.Setenv("OMNILLM_LANG", "zh")
	if err := ctl(sock, control.Request{Op: "revoke", Args: map[string]string{"id": "Pixel"}}, func(control.Reply) (bool, bool) { return false, false }); err == nil ||
		err.Error() != "没有叫“Pixel”的设备。运行 omnillm hub devices 查看。" {
		t.Fatal(err)
	}
}

// --models takes names as well as IDs; a name nobody has is refused.
func TestModelsByNameInDeviceAndAppSet(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	h, sock, id := testHub(t)
	ctx := context.Background()
	if err := h.SetModelPref(ctx, id, ptr("Writer"), nil); err != nil {
		t.Fatal(err)
	}
	app, _, err := h.AddApp(ctx, "notes", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	a, err := appSetArgs(splitTarget([]string{"notes", "--models", "writer, " + id}))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(sock, control.Request{Op: "app.set", Args: a}); err != nil {
		t.Fatal(err)
	}
	as, _ := h.Apps(ctx)
	if len(as) != 1 || as[0].ID != app.ID || len(as[0].Models) != 1 || as[0].Models[0] != id {
		t.Fatalf("%+v", as)
	}
	a, _ = appSetArgs("notes", []string{"--models", "Poet"})
	for lang, want := range map[string]string{
		"zh": "没有叫“Poet”的模型。运行 omnillm hub models 查看。",
		"en": "No model is called “Poet”. Run omnillm hub models to see them.",
	} {
		if _, _, err := run(sock, control.Request{Op: "app.set", Lang: lang, Args: a}); err == nil || err.Error() != want {
			t.Errorf("%s: %v", lang, err)
		}
	}
	// The apps table now says how many it can use.
	_, data, _ := run(sock, control.Request{Op: "apps"})
	if table, _ := appsTable(data); !strings.Contains(strings.Split(table, "\n")[1], "  1  ") {
		t.Fatal(table)
	}
	// Help names the form.
	for _, lang := range []string{"zh", "en"} {
		t.Setenv("OMNILLM_LANG", lang)
		want := map[string]string{"zh": "[--models 名字或ID,… | --all-models]", "en": "[--models name-or-id,… | --all-models]"}[lang]
		if strings.Count(usageText(true), want) != 2 {
			t.Errorf("%s: help has no %q", lang, want)
		}
	}
}

func ptr[T any](v T) *T { return &v }

// A model is found by the name the terminal shows in either language and
// as the console lists it ("… · 订阅"); a source by either language's name.
func TestFindModelAndSourceInBothLanguages(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	ms := []hub.ModelView{
		{ModelInfo: omnillm.ModelInfo{ID: "claude-code/sonnet", DisplayName: "Claude Sonnet", Kind: "claude-code", Latest: true}},
		{ModelInfo: omnillm.ModelInfo{ID: "codex/default", DisplayName: "ChatGPT (Codex)", Kind: "codex"}},
	}
	for q, want := range map[string]string{
		"Claude Sonnet（最新版）":             "claude-code/sonnet",
		"Claude Sonnet (latest)":         "claude-code/sonnet",
		"Claude Sonnet · 订阅":             "claude-code/sonnet",
		"Claude Sonnet（最新版）· 订阅":         "claude-code/sonnet",
		"claude sonnet · subscription":   "claude-code/sonnet",
		"ChatGPT（Codex · 最新版）":           "codex/default",
		"ChatGPT（Codex · 最新版）· 订阅":       "codex/default",
		"ChatGPT (Codex · latest)":       "codex/default",
		"ChatGPT (Codex) · subscription": "codex/default",
	} {
		if m, err := findModel("en", ms, q); err != nil || m.ID != want {
			t.Errorf("%q: %v %v", q, m.ID, err)
		}
	}
	ps := []hub.ProviderInfo{{Name: "claude-code", Kind: provider.ClaudeCode}, {Name: "codex", Kind: provider.Codex}}
	for q, want := range map[string]string{"Claude 订阅": "claude-code", "Claude subscription": "claude-code", "ChatGPT 订阅（Codex）": "codex", "chatgpt subscription (codex)": "codex"} {
		if p, err := findProvider("zh", ps, q); err != nil || p.Name != want {
			t.Errorf("%q: %v %v", q, p.Name, err)
		}
	}
	if _, err := findProvider("zh", ps, "groq"); err == nil || err.Error() != "没有叫“groq”的模型来源。运行 omnillm hub providers 查看。" {
		t.Fatal(err)
	}
}

// A mistake in the options is said in one line before the full help, and
// pair and provider add go the same way (never Go's "flag provided but
// not defined").
func TestOptionMistakesSayWhatWasWrong(t *testing.T) {
	for _, c := range []struct {
		lang string
		err  func() error
		want string
	}{
		{"zh", func() error { _, err := deviceSetArgs("iPhone", []string{"--colour", "red"}); return err }, "不认识选项 --colour。"},
		{"en", func() error { _, err := deviceSetArgs("iPhone", []string{"--colour=red"}); return err }, "Unknown option --colour."},
		{"en", func() error { _, err := appSetArgs("x", []string{"-budget", "3"}); return err }, "Unknown option -budget."},
		{"zh", func() error { _, err := modelSetArgs("x", []string{"--name"}); return err }, "选项 --name 后面要写一个值。"},
		{"en", func() error { _, err := deviceSetArgs("x", []string{"--no-limit=maybe"}); return err }, "The value of option --no-limit isn’t valid."},
		{"zh", func() error { _, err := pairArgs([]string{"--budget", "5"}); return err }, "不认识选项 --budget。"},
		{"en", func() error { _, err := pairArgs([]string{"--name", "x", "extra"}); return err }, "Unexpected “extra”."},
		{"zh", func() error { _, err := dailyTotalArgs([]string{"--unlimited"}); return err }, "不认识选项 --unlimited。"},
		{"en", func() error {
			fs := flag.NewFlagSet("provider add", flag.ContinueOnError)
			fs.String("name", "", "")
			return parseFlags(fs, []string{"--key", "sk"})
		}, "Unknown option --key."},
	} {
		t.Setenv("OMNILLM_LANG", c.lang)
		err := c.err()
		var u usageErr
		if !errors.As(err, &u) || describe(err) != c.want {
			t.Errorf("%s: %v, want %q", c.lang, err, c.want)
		}
	}
	// -h is a request for the help, not a mistake.
	if _, err := pairArgs([]string{"-h"}); !errors.Is(err, errHelp) {
		t.Fatal(err)
	}
	// The values pair takes.
	t.Setenv("OMNILLM_LANG", "en")
	a, err := pairArgs([]string{"--name", "Work", "--budget-usd", "$2.50", "--models", "Writer", "--background-only"})
	if err != nil || a["budget_usd"] != "2.5" || a["models"] != "Writer" || a["background_only"] != "true" {
		t.Fatal(a, err)
	}
	if _, err := pairArgs([]string{"--budget-usd", "NaN"}); err == nil {
		t.Fatal("NaN accepted")
	}
}

// Amounts: a leading "$" everywhere; NaN, infinities, zero and more than
// $1,000,000 refused with a sentence that says which.
func TestAmounts(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	for in, want := range map[string]int64{"5": 5_000_000, "$2.5": 2_500_000, " $ 3 ": 3_000_000, "1000000": 1_000_000_000_000, "0.004": 4_000} {
		if got, err := dollars("en", in, false); err != nil || got != want {
			t.Errorf("%q: %d %v", in, got, err)
		}
	}
	for _, bad := range []string{"NaN", "nan", "Inf", "+Inf", "-inf", "0", "-1", "lots", "$", ""} {
		if _, err := dollars("en", bad, false); err == nil || describe(err) != "The daily limit must be above 0 (USD), or use --no-limit." {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if _, err := dollars("en", "1000000.01", false); err == nil || describe(err) != "The daily limit can be at most $1,000,000." {
		t.Fatal(err)
	}
	if _, err := dollars("zh", "1e9", false); err == nil || describe(err) != "每天额度最多 $1,000,000。" {
		t.Fatal(err)
	}
	if _, err := dollars("zh", "1e9", true); err == nil || describe(err) != "每天总额度最多 $1,000,000。" {
		t.Fatal(err)
	}
	a, err := deviceSetArgs("iPhone", []string{"--budget-usd", "$7"})
	if err != nil || a["budget_usd"] != "$7" {
		t.Fatal(a, err)
	}
	if _, b, _, _, err := deviceUpdate("en", a); err != nil || *b != 7_000_000 {
		t.Fatal(err)
	}
	for _, bad := range [][]string{{"--budget-usd", "NaN"}, {"--budget-usd", "Inf"}, {"--budget-usd", "2000000"}} {
		if _, err := deviceSetArgs("iPhone", bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	for _, bad := range []string{"NaN", "Inf", "1e7"} {
		if _, err := dailyTotalArgs([]string{bad}); err == nil {
			t.Errorf("daily-total %s accepted", bad)
		}
	}
	if _, err := modelSetArgs("x", []string{"--price-in", "NaN", "--price-out", "1"}); err == nil {
		t.Error("NaN price accepted")
	}
	if a, err := modelSetArgs("x", []string{"--price-in", "$3", "--price-out", "$15"}); err != nil {
		t.Error(err)
	} else if _, _, u, err := modelUpdate("en", a); err != nil || *u.PriceIn != 3 || *u.PriceOut != 15 {
		t.Error(u, err)
	}
	// The help shows a real amount.
	for _, lang := range []string{"zh", "en"} {
		t.Setenv("OMNILLM_LANG", lang)
		if !strings.Contains(usageText(true), "omnillm hub daily-total 20") || strings.Contains(usageText(true), "{daily}") {
			t.Errorf("%s: %s", lang, usageText(true))
		}
	}
}

// omnillm status names each source that needs the owner, as the console's
// chips do, instead of "All set".
func TestStatusNamesSourcesThatNeedAttention(t *testing.T) {
	st := Status{Running: true, Providers: 2, RelayConfigured: true, RelayUp: true, Devices: 1, Problems: []SourceProblem{
		{Name: "claude-code", Kind: provider.ClaudeCode, Problem: provider.ProblemLogin},
		{Name: "codex", Kind: provider.Codex, Problem: provider.ProblemLimit},
	}}
	term := map[string]string{"zh": "在终端里", "en": "in a terminal"}
	if runtime.GOOS == "darwin" {
		term = map[string]string{"zh": "在终端", "en": "in Terminal"}
	}
	t.Setenv("OMNILLM_LANG", "zh")
	got := nextStep(st)
	want := "需要处理：Claude 订阅 — 登录已失效：" + term["zh"] + "运行 claude，再输入 /login\n需要处理：ChatGPT 订阅（Codex） — 用量到上限，稍后恢复"
	if got != want {
		t.Fatalf("%q\nwant %q", got, want)
	}
	t.Setenv("OMNILLM_LANG", "en")
	got = nextStep(st)
	want = "Needs attention: Claude subscription — Signed out: " + term["en"] + ", run claude, then type /login\nNeeds attention: ChatGPT subscription (Codex) — Usage limit reached; back soon"
	if got != want {
		t.Fatalf("%q\nwant %q", got, want)
	}
	st.Problems = nil
	if got := nextStep(st); !strings.HasPrefix(got, "All set.") {
		t.Fatal(got)
	}
	// The hub sends them with its status.
	_, sock, _ := testHub(t)
	_, data, err := run(sock, control.Request{Op: "status"})
	var s Status
	if err != nil || json.Unmarshal(data, &s) != nil || s.Providers != 1 || len(s.Problems) != 0 {
		t.Fatalf("%s %v", data, err)
	}
}

// hub usage --day: today, yesterday or the last 7 days, said so in the
// table.
func TestUsageByDay(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	_, sock, _ := testHub(t)
	for _, day := range []string{"today", "yesterday", "7d"} {
		if _, _, err := run(sock, control.Request{Op: "usage", Args: map[string]string{"day": day}}); err != nil {
			t.Errorf("%s: %v", day, err)
		}
	}
	if _, _, err := run(sock, control.Request{Op: "usage", Lang: "zh", Args: map[string]string{"day": "monday"}}); err == nil || !strings.Contains(err.Error(), "--day 只能是 today") {
		t.Fatal(err)
	}
	b, _ := json.Marshal([]hub.Spend{{Who: "_total", Calls: 3}, {Who: "host", Calls: 3}})
	for _, c := range []struct{ lang, day, total, empty string }{
		{"zh", "today", "今天合计", "今天还没有用过。"},
		{"zh", "yesterday", "昨天合计", "昨天没有用过。"},
		{"zh", "7d", "最近 7 天合计", "最近 7 天没有用过。"},
		{"en", "yesterday", "Total yesterday", "Nothing was used yesterday."},
		{"en", "7d", "Total, last 7 days", "Nothing was used in the last 7 days."},
	} {
		t.Setenv("OMNILLM_LANG", c.lang)
		got, _ := usageTable(b, c.day)
		lines := strings.Split(got, "\n")
		if len(lines) != 3 || !strings.HasPrefix(lines[2], c.total) || !strings.HasPrefix(lines[1], upperFirst(thisMachine())) {
			t.Errorf("%s %s:\n%s", c.lang, c.day, got)
		}
		if got, _ := usageTable([]byte("null"), c.day); got != c.empty {
			t.Errorf("%s %s: %q", c.lang, c.day, got)
		}
	}
	t.Setenv("OMNILLM_LANG", "zh")
	if !strings.Contains(usageText(true), "看哪一天：今天、昨天或最近 7 天") || !strings.Contains(usageText(true), "--day today|yesterday|7d") {
		t.Fatal(usageText(true))
	}
}

// The devices and apps tables have the same layout, with the models each
// can use: all, how many, or none usable.
func TestDevicesAndAppsTablesShowTheirModels(t *testing.T) {
	now := time.Now()
	five, none, spent := int64(5_000_000), hub.NoLimit, int64(1_250_000)
	zero, three := 0, 3
	devices, _ := json.Marshal([]deviceRow{
		{ID: "d1", Name: "iPhone", BudgetMicroUSD: &five, SpentMicroUSD: &spent, LastSeen: now},
		{ID: "d2", Name: "工作电脑", BudgetMicroUSD: &none, Models: []string{"a", "b", "c"}, Usable: &three},
		{ID: "d3", Name: "iPad", BudgetMicroUSD: &five, Models: []string{"gone"}, Usable: &zero},
	})
	apps, _ := json.Marshal([]appRow{
		{ID: "a1", Name: "backup", BudgetMicroUSD: &five, SpentMicroUSD: &spent},
		{ID: "a2", Name: "notes", BudgetMicroUSD: &none, Paused: true, Models: []string{"a"}, Usable: &zero},
	})
	for _, c := range []struct {
		lang                       string
		head                       []string
		all, three, none, spentCol string
	}{
		{"zh", []string{"名字", "状态", "今天 / 每天额度", "能用的模型"}, "全部", "3 个", "没有能用的", "$1.25 / $5.00"},
		{"en", []string{"Name", "Status", "Today / daily limit", "Models"}, "All", "3", "None usable", "$1.25 / $5.00"},
	} {
		t.Setenv("OMNILLM_LANG", c.lang)
		d, _ := devicesTable(devices, now)
		a, _ := appsTable(apps)
		dl, al := strings.Split(d, "\n"), strings.Split(a, "\n")
		if len(dl) != 4 || len(al) != 3 {
			t.Fatalf("%s:\n%s\n%s", c.lang, d, a)
		}
		for _, h := range c.head {
			if !strings.Contains(dl[0], h) || !strings.Contains(al[0], h) {
				t.Errorf("%s: no %q\n%s\n%s", c.lang, h, d, a)
			}
		}
		// One "today / daily limit" column, as on the devices table.
		if strings.Count(al[0], t2(c.lang, "每天额度", "daily limit")) != 1 || !strings.HasSuffix(al[0], "ID") {
			t.Errorf("%s: %s", c.lang, al[0])
		}
		if !strings.Contains(dl[1], c.spentCol) || !strings.Contains(dl[1], "  "+c.all+" ") || !strings.Contains(dl[2], "  "+c.three+" ") || !strings.Contains(dl[3], "  "+c.none+" ") {
			t.Errorf("%s:\n%s", c.lang, d)
		}
		if !strings.Contains(al[1], c.spentCol) || !strings.Contains(al[1], "  "+c.all+" ") || !strings.Contains(al[2], "  "+c.none+" ") {
			t.Errorf("%s:\n%s", c.lang, a)
		}
		// Columns line up, a Chinese character counting as two.
		col := strings.Index(dl[0], t2(c.lang, "能用的模型", "Models"))
		if width(dl[0][:col]) != width(dl[1][:strings.Index(dl[1], c.all+" ")]) {
			t.Errorf("%s: columns do not line up\n%s", c.lang, d)
		}
	}
}

func t2(lang, z, e string) string { return tl(lang, z, e) }

// usableModels counts what a device or app can really use.
func TestUsableModels(t *testing.T) {
	ms := []hub.ModelView{
		{ModelInfo: omnillm.ModelInfo{ID: "a"}},
		{ModelInfo: omnillm.ModelInfo{ID: "b", HumanOnly: true}},
		{ModelInfo: omnillm.ModelInfo{ID: "c"}, Hidden: true},
	}
	for _, c := range []struct {
		allowed            []string
		device, background bool
		want               int
	}{
		{nil, true, false, 2}, // a phone: not the hidden one
		{nil, true, true, 1},  // a server: no chat-only ones either
		{nil, false, true, 2}, // an app: hiding is about devices
		{[]string{"b", "x"}, true, false, 1},
		{[]string{"b"}, false, true, 0}, // an app given a chat-only model only
		{[]string{"gone"}, true, false, 0},
	} {
		if got := usableModels(ms, c.allowed, c.device, c.background); got != c.want {
			t.Errorf("%+v: %d", c, got)
		}
	}
}
