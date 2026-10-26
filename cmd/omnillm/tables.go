package main

import (
	"encoding/json"
	"fmt"
	"runtime"
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
	// Usable is how many models it can use now (newer hubs only).
	Usable *int `json:",omitempty"`
	// Online and SpentMicroUSD (today) come from newer hubs only.
	Online        *bool  `json:",omitempty"`
	SpentMicroUSD *int64 `json:",omitempty"`
}

// devicesTable is the table `omnillm hub devices` prints: name, on or
// paused, today's spend / daily limit, the models it can use, last seen
// (online, or relative local time), ID.
func devicesTable(data []byte, now time.Time) (string, error) {
	var rows []deviceRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return t("还没有设备。", "No devices yet."), nil
	}
	out := [][]string{{t("名字", "Name"), t("状态", "Status"), t("今天 / 每天额度", "Today / daily limit"), t("能用的模型", "Models"), t("最近在线", "Last seen"), "ID"}}
	for _, r := range rows {
		last := whenText(r.LastSeen, now)
		if r.Online != nil && *r.Online {
			last = t("● 在线", "● Online")
		}
		out = append(out, []string{r.Name, stateText(r.Paused), spendText(r.Budget, r.BudgetMicroUSD, r.SpentMicroUSD), modelsText(r.Models, r.Usable), last, r.ID})
	}
	return table(out), nil
}

// spendText is the "today / daily limit" cell: "$1.25 / $5.00", "$0.00 ·
// no limit"; just the limit from a hub that doesn't send the spend.
func spendText(budget string, budgetMicro, spent *int64) string {
	limit := budget
	if budgetMicro != nil {
		limit = limitText(*budgetMicro)
	}
	if spent == nil {
		return limit
	}
	if budgetMicro != nil && *budgetMicro == hub.NoLimit {
		return fmt.Sprintf(t("%s · 不设上限", "%s · no limit"), usd(*spent))
	}
	return usd(*spent) + " / " + limit
}

// modelsText is the models cell: every model, how many, or none usable
// now (the ones it was given are gone, or are not for it).
func modelsText(models []string, usable *int) string {
	n := len(models)
	if usable != nil {
		n = *usable
	}
	switch {
	case n == 0 && (usable != nil || len(models) > 0):
		return t("没有能用的", "None usable")
	case len(models) == 0 && usable != nil:
		return fmt.Sprintf(t("全部 %d 个", "All %d"), n)
	case len(models) == 0:
		return t("全部", "All")
	}
	return fmt.Sprintf(t("%d 个", "%d"), n)
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
	Usable         *int     `json:",omitempty"` // newer hubs only
	SpentMicroUSD  *int64   `json:",omitempty"` // today
}

// appsTable is the table `omnillm hub apps` prints, laid out as the
// devices table: name, on or paused, today's spend / daily limit, the
// models it can use, ID.
func appsTable(data []byte) (string, error) {
	var rows []appRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return t("还没有应用。", "No apps yet."), nil
	}
	out := [][]string{{t("名字", "Name"), t("状态", "Status"), t("今天 / 每天额度", "Today / daily limit"), t("能用的模型", "Models"), "ID"}}
	for _, r := range rows {
		out = append(out, []string{r.Name, stateText(r.Paused), spendText(r.Budget, r.BudgetMicroUSD, r.SpentMicroUSD), modelsText(r.Models, r.Usable), r.ID})
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
		shown := yes(!m.Hidden)
		if m.SourceOff {
			shown = t("否（来源已关闭）", "No (source off)")
		}
		out = append(out, []string{modelName(m.ModelInfo), sourceLabelIn(lang(), provider.Kind(m.Kind), m.Provider), yes(m.HumanOnly), shown, m.ID})
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
	out := [][]string{{t("来源", "Source"), t("接入方式", "Connected with"), t("状态", "Status"), "ID"}}
	for _, p := range ps {
		how := "—"
		switch {
		case p.Subscription:
			how = t("订阅登录", "Subscription login")
		case p.KeyLast4 != "":
			how = t("API Key …", "API key …") + p.KeyLast4
		case p.Local:
			how = onMachine(t("只在 %s 上", "%s only"))
			how = upperFirst(how) // "This Mac only"
		}
		out = append(out, []string{sourceNameIn(lang(), p), how, problemText(p, now), p.Name})
	}
	return table(out), nil
}

// problemText is a source's state: fine, or what to do about it, as the
// console's problem chips say it.
func problemText(p hub.ProviderInfo, now time.Time) string {
	if p.Off {
		return t("已关闭", "Off")
	}
	switch p.Problem {
	case provider.ProblemLogin:
		if p.Kind == provider.Codex || p.Kind == provider.Antigravity {
			return fmt.Sprintf(t("登录已失效：%s运行 omnillm login %s", "Signed out: %s, run omnillm login %s"), inTerminal(), loginWord(p.Kind))
		}
		return fmt.Sprintf(t("登录已失效：%s运行 omnillm login claude", "Signed out: %s, run omnillm login claude"), inTerminal())
	case provider.ProblemLimit:
		if wait := time.Unix(p.ResetsAt, 0).Sub(now); p.ResetsAt > 0 && wait > 0 {
			return fmt.Sprintf(t("用量到上限，%s后恢复", "Usage limit reached; back in %s"), omnillm.RoundDuration(lang(), wait))
		}
		return t("用量到上限，稍后恢复", "Usage limit reached; back soon")
	}
	return t("正常", "OK")
}

// inTerminal is where a command is typed: the Terminal app on a Mac, a
// terminal elsewhere.
func inTerminal() string {
	if runtime.GOOS == "darwin" {
		return t("在终端", "in Terminal")
	}
	return t("在终端里", "in a terminal")
}

// sourceNameIn is a model source as people call it, in language l:
// "Claude 订阅" / "Claude subscription", "Ollama（这台 Mac）" /
// "Ollama (this Mac)".
func sourceNameIn(l string, p hub.ProviderInfo) string {
	name := p.DisplayName
	if name == "" {
		name = p.Name
	}
	return sourceLabelIn(l, p.Kind, name)
}

func sourceLabelIn(l string, k provider.Kind, name string) string {
	switch k {
	case provider.ClaudeCode:
		return tl(l, "Claude 订阅", "Claude subscription")
	case provider.Codex:
		return tl(l, "ChatGPT 订阅（Codex）", "ChatGPT subscription (Codex)")
	case provider.Antigravity:
		return tl(l, "Gemini 订阅（Antigravity）", "Google AI subscription (Antigravity)")
	case provider.Ollama:
		// Only the Ollama app itself is "Ollama"; another server on this
		// machine (llama.cpp, LM Studio) added with its own name is a
		// model on this machine, by that name.
		if name != "" && name != "ollama" {
			return onMachineIn(l, tl(l, "%s 上的模型", "Model on %s")) + tl(l, "（"+name+"）", " ("+name+")")
		}
		return onMachineIn(l, tl(l, "Ollama（%s）", "Ollama (%s)"))
	}
	return name
}

// limitText is a daily limit in the terminal's language.
func limitText(micro int64) string { return limitTextIn(lang(), micro) }

func limitTextIn(l string, micro int64) string {
	if micro == hub.NoLimit {
		return tl(l, "不设上限", "No limit")
	}
	return usd(micro)
}

// modelName is a model's name in the terminal's language: the owner's own
// name as it is; otherwise Codex by the name people know, with the model
// it runs when the hub names one and "latest" when not (as the phone and
// console say it), and a model whose version is not known yet marked as
// the latest.
func modelName(m omnillm.ModelInfo) string { return stripControls(modelNameIn(lang(), m)) } // for the terminal

// modelNameIn is modelName in language l.
func modelNameIn(l string, m omnillm.ModelInfo) string {
	switch {
	case m.Renamed:
		return m.DisplayName
	case m.Kind == "codex" && m.Version != "":
		return fmt.Sprintf(tl(l, "ChatGPT（Codex · %s）", "ChatGPT (Codex · %s)"), m.Version)
	case m.Kind == "codex":
		return tl(l, "ChatGPT（Codex · 最新版）", "ChatGPT (Codex · latest)")
	case m.Latest:
		return m.DisplayName + tl(l, "（最新版）", " (latest)")
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

// usageTable is what `omnillm hub usage --day day` prints, as the console
// shows it: one line per device or app by name, busiest first, removed
// ones folded into one line, the total last.
func usageTable(data []byte, day string) (string, error) {
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
				s.Name = upperFirst(thisMachine())
			} else if s.Name == "" {
				s.Name = s.Who
			}
			rows = append(rows, s)
		}
	}
	if len(rows) == 0 && goneN == 0 {
		switch day {
		case "yesterday":
			return t("昨天没有用过。", "Nothing was used yesterday."), nil
		case "7d":
			return t("最近 7 天没有用过。", "Nothing was used in the last 7 days."), nil
		}
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
		switch day {
		case "yesterday":
			total.Name = t("昨天合计", "Total yesterday")
		case "7d":
			total.Name = t("最近 7 天合计", "Total, last 7 days")
		default:
			total.Name = t("今天合计", "Total today")
		}
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
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) || (r >= 0x3000 && r <= 0x303f) || (r >= 0xff00 && r <= 0xff60) || r >= 0x1f300 {
			n += 2
		} else {
			n++
		}
	}
	return n
}
