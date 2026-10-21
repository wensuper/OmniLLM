package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

// Two devices with the same name are never guessed between: the person is
// told their IDs.
func TestFindNamedRefusesSharedNames(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	ds := []hub.Device{{ID: "d1", Name: "iPhone"}, {ID: "d2", Name: "iPhone"}, {ID: "d3", Name: "iPad"}}
	_, err := findDevice(ds, "iPhone")
	if err == nil || describe(err) != "Several devices are called “iPhone”. Use the ID: d1, d2" {
		t.Fatalf("%v", err)
	}
	if d, err := findDevice(ds, "d2"); err != nil || d.ID != "d2" {
		t.Fatalf("by ID: %v %v", d, err)
	}
	t.Setenv("OMNILLM_LANG", "zh")
	as := []hub.App{{ID: "a1", Name: "backup"}, {ID: "a2", Name: "backup"}}
	if _, err := findApp(as, "backup"); err == nil || describe(err) != "有几个应用都叫“backup”。请用 ID：a1、a2" {
		t.Fatalf("%v", err)
	}
	// An exact name wins over one that differs only in case.
	if a, err := findApp([]hub.App{{ID: "a1", Name: "Notes"}, {ID: "a2", Name: "notes"}}, "Notes"); err != nil || a.ID != "a1" {
		t.Fatalf("%v %v", a, err)
	}
	// A model by the name the table shows, or the one it goes by.
	ms := []hub.ModelView{{ModelInfo: omnillm.ModelInfo{ID: "claude-code/sonnet", DisplayName: "Claude Sonnet", Latest: true}}}
	for _, q := range []string{"claude-code/sonnet", "Claude Sonnet", "Claude Sonnet（最新版）", "claude sonnet"} {
		if m, err := findModel(ms, q); err != nil || m.ID != "claude-code/sonnet" {
			t.Errorf("%q: %v", q, err)
		}
	}
	ps := []hub.ProviderInfo{{Name: "claude-code", Kind: provider.ClaudeCode}, {Name: "groq", Kind: provider.OpenAICompatible, DisplayName: "groq"}}
	for q, want := range map[string]string{"Claude 订阅": "claude-code", "claude-code": "claude-code", "GROQ": "groq"} {
		if p, err := findProvider(ps, q); err != nil || p.Name != want {
			t.Errorf("%q: %v %v", q, p.Name, err)
		}
	}
}

// Names with spaces need no quotes: the words before the first flag are
// the name.
func TestSplitTarget(t *testing.T) {
	for _, c := range []struct {
		args  []string
		name  string
		flags int
	}{
		{[]string{"iPhone", "(2)", "--paused", "true"}, "iPhone (2)", 2},
		{[]string{"iPhone (2)", "--no-limit"}, "iPhone (2)", 1},
		{[]string{"Work", "laptop"}, "Work laptop", 0},
		{[]string{"--name", "x"}, "", 2},
	} {
		name, flags := splitTarget(c.args)
		if name != c.name || len(flags) != c.flags {
			t.Errorf("%q: %q %q", c.args, name, flags)
		}
	}
	a, err := appSetArgs(splitTarget([]string{"my", "script", "--budget-usd", "2", "--paused", "false"}))
	if err != nil || a["app"] != "my script" || a["budget_usd"] != "2" || a["paused"] != "false" {
		t.Fatalf("%v %v", a, err)
	}
	if _, err := appSetArgs("my script", nil); err == nil {
		t.Fatal("nothing to change accepted")
	}
}

func TestModelSetArgs(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	a, err := modelSetArgs("Claude Sonnet", []string{"--name", "", "--shown", "false"})
	if err != nil {
		t.Fatal(err)
	}
	name, shown, u, err := modelUpdate(a)
	if err != nil || name == nil || *name != "" || shown == nil || *shown || u != (hub.ModelUpdate{}) {
		t.Fatalf("%v %v %v %+v %v", a, name, shown, u, err)
	}
	a, _ = modelSetArgs("x", []string{"--price-in", "3", "--price-out", "15", "--human-only", "true"})
	name, shown, u, err = modelUpdate(a)
	if err != nil || name != nil || shown != nil || *u.PriceIn != 3 || *u.PriceOut != 15 || !*u.HumanOnly {
		t.Fatalf("%v %+v %v", a, u, err)
	}
	for _, bad := range [][]string{{}, {"--price-in", "3"}, {"--price-in", "-1", "--price-out", "1"}, {"--shown", "maybe"}, {"--human-only", "x"}, {"--colour", "red"}} {
		if _, err := modelSetArgs("x", bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestDailyTotalArgs(t *testing.T) {
	for in, want := range map[string]int64{"5": 5_000_000, "$2.5": 2_500_000, "--no-limit": hub.NoLimit} {
		if got, err := dailyTotalArgs([]string{in}); err != nil || got != want {
			t.Errorf("%q: %d %v", in, got, err)
		}
	}
	for _, bad := range [][]string{{}, {"0"}, {"-3"}, {"lots"}, {"5", "--no-limit"}} {
		if _, err := dailyTotalArgs(bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestAppsModelsProvidersTables(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "zh")
	five, none, spent := int64(5_000_000), hub.NoLimit, int64(1_250_000)
	b, _ := json.Marshal([]appRow{{ID: "a1", Name: "backup", BudgetMicroUSD: &five, SpentMicroUSD: &spent}, {ID: "a2", Name: "notes", BudgetMicroUSD: &none, Paused: true}})
	got, err := appsTable(b)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "名字") || !strings.HasSuffix(lines[0], "ID") || !strings.Contains(lines[0], "每天额度") {
		t.Fatal(got)
	}
	if !strings.Contains(lines[1], "开启") || !strings.Contains(lines[1], "$1.25") || !strings.Contains(lines[1], "$5.00") || !strings.HasSuffix(lines[1], "a1") {
		t.Fatal(got)
	}
	if !strings.Contains(lines[2], "已暂停") || !strings.Contains(lines[2], "不设上限") || !strings.HasSuffix(lines[2], "a2") {
		t.Fatal(got)
	}
	if got, _ := appsTable([]byte("null")); got != "还没有应用。" {
		t.Fatal(got)
	}

	b, _ = json.Marshal([]hub.ModelView{
		{ModelInfo: omnillm.ModelInfo{ID: "claude-code/sonnet", DisplayName: "Claude Sonnet", Provider: "claude-code", Kind: "claude-code", Latest: true, HumanOnly: true}},
		{ModelInfo: omnillm.ModelInfo{ID: "codex/default", DisplayName: "ChatGPT (Codex)", Provider: "codex", Kind: "codex", Latest: true, HumanOnly: true}},
		{ModelInfo: omnillm.ModelInfo{ID: "groq/llama", DisplayName: "写作", Provider: "groq", Kind: "openai-compatible", Renamed: true}, Hidden: true},
	})
	got, err = modelsTable(b)
	if err != nil {
		t.Fatal(err)
	}
	lines = strings.Split(got, "\n")
	if len(lines) != 4 || !strings.Contains(lines[0], "来源") || !strings.Contains(lines[0], "只能手动对话") || !strings.Contains(lines[0], "在设备上显示") || !strings.HasSuffix(lines[0], "ID") {
		t.Fatal(got)
	}
	if !strings.HasPrefix(lines[1], "Claude Sonnet（最新版）") || !strings.Contains(lines[1], "Claude 订阅") || !strings.HasSuffix(lines[1], "claude-code/sonnet") {
		t.Fatal(got)
	}
	if !strings.HasPrefix(lines[2], "ChatGPT（Codex · 最新版）") || !strings.Contains(lines[2], "ChatGPT 订阅（Codex）") {
		t.Fatal(got)
	}
	if !strings.HasPrefix(lines[3], "写作") || !strings.Contains(lines[3], "groq") || !strings.Contains(lines[3], "否  ") {
		t.Fatal(got)
	}

	t.Setenv("OMNILLM_LANG", "en")
	now := time.Now()
	b, _ = json.Marshal([]hub.ProviderInfo{
		{Name: "claude-code", Kind: provider.ClaudeCode, Subscription: true, Problem: provider.ProblemLimit, ResetsAt: now.Add(3 * time.Hour).Unix()},
		{Name: "codex", Kind: provider.Codex, Subscription: true, Problem: provider.ProblemLogin},
		{Name: "groq", Kind: provider.OpenAICompatible, DisplayName: "groq", KeyLast4: "1234"},
		{Name: "ollama", Kind: provider.Ollama, Local: true},
	})
	got, err = providersTable(b, now)
	if err != nil {
		t.Fatal(err)
	}
	lines = strings.Split(got, "\n")
	if len(lines) != 5 || !strings.HasPrefix(lines[0], "Source") || !strings.Contains(lines[0], "Connected with") {
		t.Fatal(got)
	}
	for i, want := range [][]string{
		{"Claude subscription", "Subscription login", "Usage limit reached; back in about 3 hours"},
		{"ChatGPT subscription (Codex)", "Signed out: in Terminal, run codex login"},
		{"groq", "API key …1234", "OK"},
		{"Ollama (this ", " only", "OK"},
	} {
		for _, w := range want {
			if !strings.Contains(lines[i+1], w) {
				t.Errorf("line %d: no %q\n%s", i+1, w, got)
			}
		}
	}
}

func TestDevicesTableEndsWithTheID(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	b, _ := json.Marshal([]deviceRow{{ID: "d1", Name: "iPhone"}, {ID: "d2", Name: "iPhone (2)"}})
	got, _ := devicesTable(b, time.Now())
	lines := strings.Split(got, "\n")
	if !strings.HasSuffix(lines[0], "ID") || !strings.HasSuffix(lines[1], "d1") || !strings.HasSuffix(lines[2], "d2") {
		t.Fatal(got)
	}
}

func TestHelpListsTheRoundFiveCommands(t *testing.T) {
	for _, lang := range []string{"zh", "en"} {
		t.Setenv("OMNILLM_LANG", lang)
		all := usageText(true)
		for _, c := range []string{"omnillm hub app set", "omnillm hub provider rm", "omnillm hub daily-total", "--shown true|false", `--name ""`, `omnillm hub revoke "iPhone (2)"`, "[--json]"} {
			if !strings.Contains(all, c) {
				t.Errorf("%s: no %q", lang, c)
			}
		}
	}
	t.Setenv("OMNILLM_LANG", "zh")
	if !strings.Contains(usageText(false), "（--purge 连数据和 API Key 一起删）") {
		t.Error(usageText(false))
	}
	t.Setenv("OMNILLM_LANG", "en")
	if !strings.Contains(usageText(false), "(--purge also deletes data and API keys)") {
		t.Error(usageText(false))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// `hub model set` renames and hides a model the way the console does
// (the owner's preferences, --name "" restoring the source's name), and
// `hub models` shows the result.
func TestModelSetGoesThroughThePreferences(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	t.Setenv("PATH", t.TempDir()) // no subscription programs here
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
	id := ms[0].ID
	dir, err := os.MkdirTemp("/tmp", "omni")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "c.sock")
	ln, err := control.Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	go control.Serve(ctx, ln, hubHandler(h, "", nil))
	run := func(op string, a map[string]string) (string, []byte, error) {
		var text string
		var data []byte
		err := control.Call(sock, control.Request{Op: op, Args: a}, func(r control.Reply) (bool, bool) {
			text, data = r.Text, r.Data
			return false, false
		})
		return text, data, err
	}
	a, err := modelSetArgs(splitTarget([]string{id, "--name", "Writer", "--shown", "false"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := run("model.set", a); err != nil {
		t.Fatal(err)
	}
	vs, _ := h.OwnerModels(ctx)
	if len(vs) != 1 || vs[0].DisplayName != "Writer" || !vs[0].Renamed || !vs[0].Hidden {
		t.Fatalf("%+v", vs)
	}
	_, data, err := run("models", nil)
	if err != nil {
		t.Fatal(err)
	}
	table, _ := modelsTable(data)
	if !strings.Contains(table, "Writer") || !strings.Contains(strings.Split(table, "\n")[1], "No") {
		t.Fatal(table)
	}
	// By its new name; --name "" restores the source's name.
	a, _ = modelSetArgs("Writer", []string{"--name", "", "--shown", "true"})
	if _, _, err := run("model.set", a); err != nil {
		t.Fatal(err)
	}
	vs, _ = h.OwnerModels(ctx)
	if vs[0].Renamed || vs[0].Hidden || vs[0].DisplayName == "Writer" {
		t.Fatalf("%+v", vs)
	}
	// The other new commands, end to end.
	if _, _, err := run("daily_total.set", map[string]string{"micro_usd": "7000000"}); err != nil || h.DailyTotal() != 7_000_000 {
		t.Fatal(err, h.DailyTotal())
	}
	if _, _, err := run("model.set", map[string]string{"model": "nothing", "shown": "true"}); err == nil || !strings.Contains(err.Error(), "No model is called") {
		t.Fatal(err)
	}
	app, _, err := h.AddApp(ctx, "my script", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	a, _ = appSetArgs(splitTarget([]string{"my", "script", "--paused", "true", "--no-limit"}))
	if _, _, err := run("app.set", a); err != nil {
		t.Fatal(err)
	}
	_, data, _ = run("apps", nil)
	if table, _ := appsTable(data); !strings.Contains(table, "Paused") || !strings.Contains(table, "No limit") || !strings.Contains(table, app.ID) {
		t.Fatal(table)
	}
	text, _, err := run("provider.rm", map[string]string{"name": "Ollama (this Mac)"})
	if err != nil && !strings.Contains(err.Error(), "No model source") {
		t.Fatal(err)
	}
	if err != nil { // not a Mac: by its own name
		text, _, err = run("provider.rm", map[string]string{"name": "ollama"})
	}
	if ps, _ := h.Providers(ctx); err != nil || len(ps) != 0 || !strings.HasPrefix(text, "Removed “Ollama") {
		t.Fatal(text, err, ps)
	}
}
