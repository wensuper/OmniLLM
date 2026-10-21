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
	"github.com/wensuper/OmniLLM/internal/provider"
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
	out := [][]string{{t("名字", "Name"), t("状态", "Status"), t("今天 / 每天额度", "Today / daily limit"), t("最近在线", "Last seen"), "ID"}}
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
		out = append(out, []string{r.Name, state, limit, last, r.ID})
	}
	return table(out), nil
}

// stateText is a device's or app's switch: on or paused.
func stateText(paused bool) string {
	if paused {
		return t("已暂停", "Paused")
	}
	return t("开启", "On")
}

// appRow is one app as the hub sends it to `omnillm hub apps`.
type appRow struct {
	ID             string
	Name           string
	Budget         string   // as text, from an older hub
	BudgetMicroUSD *int64   `json:",omitempty"`
	Paused         bool     `json:",omitempty"`
	Models         []string `json:",omitempty"`
	SpentMicroUSD  *int64   `json:",omitempty"` // today
}

// appsTable is the table `omnillm hub apps` prints: name, on or paused,
// today's spend, daily limit, ID.
func appsTable(data []byte) (string, error) {
	var rows []appRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return t("还没有应用。", "No apps yet."), nil
	}
	out := [][]string{{t("名字", "Name"), t("状态", "Status"), t("今天", "Today"), t("每天额度", "Daily limit"), "ID"}}
	for _, r := range rows {
		limit, spent := r.Budget, "—"
		if r.BudgetMicroUSD != nil {
			limit = limitText(*r.BudgetMicroUSD)
		}
		if r.SpentMicroUSD != nil {
			spent = usd(*r.SpentMicroUSD)
		}
		out = append(out, []string{r.Name, stateText(r.Paused), spent, limit, r.ID})
	}
	return table(out), nil
}

// modelsTable is the table `omnillm hub models` prints, with the owner's
// names and choices as the console shows them: name, source, chat only,
// shown on devices, ID.
func modelsTable(data []byte) (string, error) {
	var ms []hub.ModelView
	if err := json.Unmarshal(data, &ms); err != nil {
		return "", err
	}
	if len(ms) == 0 {
		return t("还没有模型。运行 omnillm open，在“模型”里接入。", "No models yet. Run omnillm open and connect one under “Models”."), nil
	}
	yes := func(b bool) string {
		if b {
			return t("是", "Yes")
		}
		return t("否", "No")
	}
	out := [][]string{{t("名字", "Name"), t("来源", "Source"), t("只能手动对话", "Chat only"), t("在设备上显示", "Shown on devices"), "ID"}}
	for _, m := range ms {
		out = append(out, []string{modelName(m.ModelInfo), sourceLabel(provider.Kind(m.Kind), m.Provider), yes(m.HumanOnly), yes(!m.Hidden), m.ID})
	}
	return table(out), nil
}

// providersTable is the table `omnillm hub providers` prints: the source
// as people call it, how it is connected, and whether it works now.
func providersTable(data []byte, now time.Time) (string, error) {
	var ps []hub.ProviderInfo
	if err := json.Unmarshal(data, &ps); err != nil {
		return "", err
	}
	if len(ps) == 0 {
		return t("还没有模型来源。运行 omnillm open，在“模型”里接入。", "No model sources yet. Run omnillm open and connect one under “Models”."), nil
	}
	out := [][]string{{t("来源", "Source"), t("接入方式", "Connected with"), t("状态", "Status")}}
	for _, p := range ps {
		how := "—"
		switch {
		case p.Subscription:
			how = t("订阅登录", "Subscription login")
		case p.KeyLast4 != "":
			how = t("API Key …", "API key …") + p.KeyLast4
		case p.Local:
			how = onMachine(t("只在 %s 上", "%s only"))
			how = strings.ToUpper(how[:1]) + how[1:] // "This Mac only"
		}
		out = append(out, []string{sourceName(p), how, problemText(p, now)})
	}
	return table(out), nil
}

// problemText is a source's state: fine, or what to do about it.
func problemText(p hub.ProviderInfo, now time.Time) string {
	switch p.Problem {
	case provider.ProblemLogin:
		if p.Kind == provider.Codex {
			return t("登录已失效：在终端运行 codex login", "Signed out: in Terminal, run codex login")
		}
		return t("登录已失效：在终端运行 claude，再输入 /login", "Signed out: in Terminal, run claude, then type /login")
	case provider.ProblemLimit:
		if wait := time.Unix(p.ResetsAt, 0).Sub(now); p.ResetsAt > 0 && wait > 0 {
			return fmt.Sprintf(t("用量到上限，%s后恢复", "Usage limit reached; back in %s"), omnillm.RoundDuration(lang(), wait))
		}
		return t("用量到上限，稍后恢复", "Usage limit reached; back soon")
	}
	return t("正常", "OK")
}

// sourceName is a model source as people call it: "Claude 订阅" /
// "Claude subscription", "Ollama（这台 Mac）" / "Ollama (this Mac)".
func sourceName(p hub.ProviderInfo) string {
	name := p.DisplayName
	if name == "" {
		name = p.Name
	}
	return sourceLabel(p.Kind, name)
}

func sourceLabel(k provider.Kind, name string) string {
	switch k {
	case provider.ClaudeCode:
		return t("Claude 订阅", "Claude subscription")
	case provider.Codex:
		return t("ChatGPT 订阅（Codex）", "ChatGPT subscription (Codex)")
	case provider.Ollama:
		return onMachine(t("Ollama（%s）", "Ollama (%s)"))
	}
	return name
}

func limitText(micro int64) string {
	if micro == hub.NoLimit {
		return t("不设上限", "No limit")
	}
	return usd(micro)
}

// modelName is a model's name in the terminal's language: the owner's own
// name as it is; otherwise Codex by the name people know, with the model
// it runs when the hub names one and "latest" when not (as the phone and
// console say it), and a model whose version is not known yet marked as
// the latest.
func modelName(m omnillm.ModelInfo) string {
	switch {
	case m.Renamed:
		return m.DisplayName
	case m.Kind == "codex" && m.Version != "":
		return fmt.Sprintf(t("ChatGPT（Codex · %s）", "ChatGPT (Codex · %s)"), m.Version)
	case m.Kind == "codex":
		return t("ChatGPT（Codex · 最新版）", "ChatGPT (Codex · latest)")
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
