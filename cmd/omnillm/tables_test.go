package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/hub"
)

func TestWhenText(t *testing.T) {
	now := time.Date(2026, 10, 21, 15, 0, 0, 0, time.Local)
	for _, c := range []struct {
		lang string
		at   time.Time
		want string
	}{
		{"zh", now.Add(-5*time.Hour - 27*time.Minute), "今天 09:33"},
		{"en", now.Add(-5*time.Hour - 27*time.Minute), "Today 9:33 AM"},
		{"zh", time.Date(2026, 10, 20, 21, 10, 0, 0, time.Local), "昨天 21:10"},
		{"en", time.Date(2026, 10, 20, 21, 10, 0, 0, time.Local), "Yesterday 9:10 PM"},
		{"zh", time.Date(2026, 10, 18, 8, 0, 0, 0, time.Local), "10月18日"},
		{"en", time.Date(2026, 10, 18, 8, 0, 0, 0, time.Local), "Oct 18"},
		{"en", time.Time{}, "Not yet"},
		{"zh", time.Time{}, "还没用过"},
	} {
		t.Setenv("OMNILLM_LANG", c.lang)
		if got := whenText(c.at, now); got != c.want {
			t.Errorf("%s %v: %q, want %q", c.lang, c.at, got, c.want)
		}
	}
}

func TestDevicesTable(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "zh")
	now := time.Now()
	five, none := int64(5_000_000), hub.NoLimit
	b, _ := json.Marshal([]deviceRow{
		{ID: "a", Name: "iPhone", BudgetMicroUSD: &five, LastSeen: now},
		{ID: "b", Name: "工作电脑", BudgetMicroUSD: &none, Paused: true},
	})
	got, err := devicesTable(b, now)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "名字") || !strings.Contains(lines[0], "每天额度") || !strings.Contains(lines[0], "最近使用") {
		t.Fatal(got)
	}
	if !strings.Contains(lines[1], "$5.00") || !strings.Contains(lines[1], "今天 ") || strings.Contains(got, `"ID"`) {
		t.Fatal(got)
	}
	if !strings.Contains(lines[2], "已暂停") || !strings.Contains(lines[2], "不设上限") || !strings.Contains(lines[2], "还没用过") {
		t.Fatal(got)
	}
	// Columns line up, a Chinese character counting as two.
	if width(lines[1][:strings.Index(lines[1], "$")]) != width(lines[2][:strings.Index(lines[2], "不设")]) {
		t.Fatalf("columns do not line up:\n%s", got)
	}
	if got, _ := devicesTable([]byte("null"), now); got != "还没有设备。" {
		t.Fatal(got)
	}
}

func TestUsageTable(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	b, _ := json.Marshal([]hub.Spend{
		{Who: "_total", MicroUSD: 3_000_000, Calls: 9},
		{Who: "device/x", Name: "Old phone", Removed: true, Calls: 1},
		{Who: "device/y", Name: "Gone tablet", Removed: true, Calls: 1},
		{Who: "host", Calls: 2},
		{Who: "device/a", Name: "iPhone", Calls: 5, MicroUSD: 2_000_000},
	})
	got, err := usageTable(b)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(got, "\n")
	want := []string{"Name", "iPhone", "This Mac", "Removed devices and apps (2)", "Total today"}
	if len(lines) != len(want) {
		t.Fatal(got)
	}
	for i, w := range want {
		if !strings.HasPrefix(lines[i], w) {
			t.Fatalf("line %d: %q\n%s", i, lines[i], got)
		}
	}
	if strings.Contains(got, "device/") {
		t.Fatal("an id is shown:\n" + got)
	}
	if !strings.Contains(lines[0], "Calls") || !strings.Contains(lines[0], "Tokens (in / out)") {
		t.Fatal(lines[0])
	}
}

func TestUsageTableGroupsThousands(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "zh")
	b, _ := json.Marshal([]hub.Spend{{Who: "device/a", Name: "iPhone", Calls: 1234, MicroUSD: 1_234_567_890, InputTokens: 1_234_567, OutputTokens: 89}})
	got, err := usageTable(b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "次数") || !strings.Contains(got, "Token（发出 / 收到）") || !strings.Contains(got, "1,234  ") ||
		!strings.Contains(got, "$1,234.57") || !strings.Contains(got, "1,234,567 / 89") {
		t.Fatal(got)
	}
}

func TestDevicesTableWithOnlineAndSpend(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	now := time.Now()
	five, none, spent, zero := int64(5_000_000), hub.NoLimit, int64(1_250_000), int64(0)
	on, off := true, false
	b, _ := json.Marshal([]deviceRow{
		{ID: "a", Name: "iPhone", BudgetMicroUSD: &five, SpentMicroUSD: &spent, Online: &on, LastSeen: now},
		{ID: "b", Name: "Laptop", BudgetMicroUSD: &none, SpentMicroUSD: &zero, Online: &off, Paused: true},
	})
	got, err := devicesTable(b, now)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[0], "Status") || !strings.Contains(lines[0], "Today / daily limit") {
		t.Fatal(got)
	}
	if !strings.Contains(lines[1], " On ") || !strings.Contains(lines[1], "$1.25 / $5.00") || !strings.Contains(lines[1], "● Online") {
		t.Fatal(got)
	}
	if !strings.Contains(lines[2], "Paused") || !strings.Contains(lines[2], "$0.00 · no limit") || !strings.Contains(lines[2], "Not yet") {
		t.Fatal(got)
	}
}

func TestModelNameInTheTerminalsLanguage(t *testing.T) {
	for _, c := range []struct {
		lang string
		m    omnillm.ModelInfo
		want string
	}{
		{"zh", omnillm.ModelInfo{DisplayName: "Claude Sonnet", Latest: true, Kind: "claude-code"}, "Claude Sonnet（最新版）"},
		{"en", omnillm.ModelInfo{DisplayName: "Claude Sonnet", Latest: true, Kind: "claude-code"}, "Claude Sonnet (latest)"},
		{"en", omnillm.ModelInfo{DisplayName: "Claude Opus 5.5", Kind: "claude-code"}, "Claude Opus 5.5"},
		{"zh", omnillm.ModelInfo{DisplayName: "ChatGPT (Codex)", Kind: "codex"}, "ChatGPT（Codex）"},
		{"en", omnillm.ModelInfo{DisplayName: "ChatGPT (Codex)", Kind: "codex"}, "ChatGPT (Codex)"},
		{"zh", omnillm.ModelInfo{DisplayName: "我的 GPT", Kind: "codex", Renamed: true}, "我的 GPT"},
		{"zh", omnillm.ModelInfo{DisplayName: "写作", Latest: true, Renamed: true}, "写作"},
		{"en", omnillm.ModelInfo{DisplayName: "gpt-5", Kind: "openai"}, "gpt-5"},
	} {
		t.Setenv("OMNILLM_LANG", c.lang)
		if got := modelName(c.m); got != c.want {
			t.Errorf("%s %+v: %q, want %q", c.lang, c.m, got, c.want)
		}
	}
}

func TestMoney(t *testing.T) {
	for in, want := range map[int64]string{0: "$0.00", 5_000_000: "$5.00", 4_999: "$0.00", 5_000: "$0.01", 1_234_567_890: "$1,234.57", -1_500_000: "-$1.50"} {
		if got := usd(in); got != want {
			t.Errorf("usd(%d) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567", -1000: "-1,000"} {
		if got := thousands(in); got != want {
			t.Errorf("thousands(%d) = %q, want %q", in, got, want)
		}
	}
}
