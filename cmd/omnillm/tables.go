package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/hub"
)

// deviceRow is one device as the hub sends it to `omnillm hub devices`.
// The terminal renders it in its own language and time zone.
type deviceRow struct {
	ID             string
	Name           string
	Budget         string // as text, from an older hub
	BudgetMicroUSD *int64 `json:",omitempty"`
	Paused         bool   `json:",omitempty"`
	Created        time.Time
	LastSeen       time.Time `json:",omitempty"`
	Models         []string  `json:",omitempty"`
	// Online and SpentMicroUSD (today) come from newer hubs only.
	Online        *bool  `json:",omitempty"`
	SpentMicroUSD *int64 `json:",omitempty"`
}

// devicesTable is the table `omnillm hub devices` prints: name, on or
// paused, today's spend / daily limit, last seen (online, or relative
// local time).
func devicesTable(data []byte, now time.Time) (string, error) {
	var rows []deviceRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return t("还没有设备。", "No devices yet."), nil
	}
	out := [][]string{{t("名字", "Name"), t("状态", "Status"), t("今天 / 每天额度", "Today / daily limit"), t("最近在线", "Last seen")}}
	for _, r := range rows {
		state := t("开启", "On")
		if r.Paused {
			state = t("已暂停", "Paused")
		}
		limit := r.Budget
		if r.BudgetMicroUSD != nil {
			limit = limitText(*r.BudgetMicroUSD)
		}
		if r.SpentMicroUSD != nil {
			if r.BudgetMicroUSD != nil && *r.BudgetMicroUSD == hub.NoLimit {
				limit = fmt.Sprintf(t("%s · 不设上限", "%s · no limit"), usd(*r.SpentMicroUSD))
			} else {
				limit = usd(*r.SpentMicroUSD) + " / " + limit
			}
		}
		last := whenText(r.LastSeen, now)
		if r.Online != nil && *r.Online {
			last = t("● 在线", "● Online")
		}
		out = append(out, []string{r.Name, state, limit, last})
	}
	return table(out), nil
}

func limitText(micro int64) string {
	if micro == hub.NoLimit {
		return t("不设上限", "No limit")
	}
	return usd(micro)
}

// modelName is a model's name in the terminal's language: the owner's own
// name as it is; otherwise Codex by the name people know (with the model
// it runs, when the hub names one), and a model whose version is not known
// yet marked as the latest.
func modelName(m omnillm.ModelInfo) string {
	switch {
	case m.Renamed:
		return m.DisplayName
	case m.Kind == "codex" && m.Version != "":
		return fmt.Sprintf(t("ChatGPT（Codex · %s）", "ChatGPT (Codex · %s)"), m.Version)
	case m.Kind == "codex" && m.Latest:
		return t("ChatGPT（Codex）", "ChatGPT (Codex)") // its version is never known; same as the phone and console
	case m.Kind == "codex":
		return t("ChatGPT（Codex）", "ChatGPT (Codex)")
	case m.Latest:
		return m.DisplayName + t("（最新版）", " (latest)")
	}
	return m.DisplayName
}

// thousands writes n with thousands separators: 12,345.
func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "-" + s
	}
	return s
}

// whenText is a time as people say it: "今天 09:33" / "Today 9:33 AM",
// "昨天 21:10" / "Yesterday 9:10 PM", "10月18日" / "Oct 18".
func whenText(at, now time.Time) string {
	if at.IsZero() {
		return t("还没用过", "Not yet")
	}
	at = at.In(now.Location())
	day := func(x time.Time) time.Time { y, m, d := x.Date(); return time.Date(y, m, d, 0, 0, 0, 0, x.Location()) }
	switch day(now).Sub(day(at)) {
	case 0:
		return t("今天 "+at.Format("15:04"), "Today "+at.Format("3:04 PM"))
	case 24 * time.Hour:
		return t("昨天 "+at.Format("15:04"), "Yesterday "+at.Format("3:04 PM"))
	}
	if at.Year() != now.Year() {
		return t(fmt.Sprintf("%d年%d月%d日", at.Year(), at.Month(), at.Day()), at.Format("Jan 2, 2006"))
	}
	return t(fmt.Sprintf("%d月%d日", at.Month(), at.Day()), at.Format("Jan 2"))
}

// usageTable is what `omnillm hub usage` prints, as the console shows it:
// one line per device or app by name, busiest first, removed ones folded
// into one line, the total last.
func usageTable(data []byte) (string, error) {
	var spends []hub.Spend
	if err := json.Unmarshal(data, &spends); err != nil {
		return "", err
	}
	var rows []hub.Spend
	var total *hub.Spend
	gone, goneN := hub.Spend{}, 0
	for i, s := range spends {
		switch {
		case s.Who == "_total":
			total = &spends[i]
		case s.Removed:
			goneN++
			gone.Name = s.Name
			gone.Calls += s.Calls
			gone.MicroUSD += s.MicroUSD
			gone.InputTokens += s.InputTokens
			gone.OutputTokens += s.OutputTokens
		default:
			if s.Who == "host" {
				s.Name = t("这台 Mac", "This Mac")
			} else if s.Name == "" {
				s.Name = s.Who
			}
			rows = append(rows, s)
		}
	}
	if len(rows) == 0 && goneN == 0 {
		return t("今天还没有用过。", "Nothing used yet today."), nil
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Calls > rows[j].Calls })
	if goneN == 1 && gone.Name != "" {
		gone.Name = fmt.Sprintf(t("%s（已移除）", "%s (removed)"), gone.Name)
		rows = append(rows, gone)
	} else if goneN > 0 {
		gone.Name = fmt.Sprintf(t("已移除的设备和应用（%d 个）", "Removed devices and apps (%d)"), goneN)
		rows = append(rows, gone)
	}
	if total != nil {
		total.Name = t("今天合计", "Total today")
		rows = append(rows, *total)
	}
	out := [][]string{{t("名字", "Name"), t("次数", "Calls"), t("花费", "Spend"), t("Token（发出 / 收到）", "Tokens (in / out)")}}
	for _, s := range rows {
		out = append(out, []string{s.Name, thousands(s.Calls), usd(s.MicroUSD), thousands(s.InputTokens) + " / " + thousands(s.OutputTokens)})
	}
	return table(out), nil
}

// table lays rows out in columns, counting a CJK character as two.
func table(rows [][]string) string {
	var widths []int
	for _, r := range rows {
		for i, c := range r {
			if i >= len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], width(c))
		}
	}
	var b strings.Builder
	for _, r := range rows {
		for i, c := range r {
			b.WriteString(c)
			if i < len(r)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-width(c)+2))
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func width(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || (r >= 0x3000 && r <= 0x303f) || (r >= 0xff00 && r <= 0xff60) {
			n += 2
		} else {
			n++
		}
	}
	return n
}
