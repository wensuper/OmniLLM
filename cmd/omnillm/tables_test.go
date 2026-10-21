package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

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
		{"en", time.Time{}, "Never"},
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
}
