package main

// `omnillm hub device set`, `app set` and `model set`: the console's
// editors for the terminal. The terminal checks the flags and passes only
// those that were given; the hub finds the device, app or model by ID or
// name and changes those fields alone.

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/provider"
)

// deviceSetArgs turns `device set <name-or-id> [flags]` into the control
// request's arguments, with only the flags that were given.
func deviceSetArgs(target string, args []string) (map[string]string, error) {
	return setArgs("device", t("请写出设备的名字或 ID。", "Name the device (its name or ID)."), target, args)
}

// appSetArgs is deviceSetArgs for `app set`.
func appSetArgs(target string, args []string) (map[string]string, error) {
	return setArgs("app", t("请写出应用的名字或 ID。", "Name the app (its name or ID)."), target, args)
}

// setArgs reads the flags device set and app set share; key holds the
// device or app as the person named it. A mistake in the options is a
// usageErr saying what it was; the full help follows it.
func setArgs(key, nameIt, target string, args []string) (map[string]string, error) {
	fs := flag.NewFlagSet(key+" set", flag.ContinueOnError)
	fs.String("name", "", "")
	fs.String("paused", "", "true or false")
	fs.String("budget-usd", "", "")
	fs.Bool("no-limit", false, "")
	fs.String("models", "", "")
	fs.Bool("all-models", false, "")
	if err := parseFlags(fs, args); err != nil {
		return nil, err
	}
	a := map[string]string{key: strings.TrimSpace(target)}
	fs.Visit(func(f *flag.Flag) { a[strings.ReplaceAll(f.Name, "-", "_")] = f.Value.String() })
	if a[key] == "" {
		return nil, say(nameIt)
	}
	if len(a) == 1 {
		return nil, say(t("没有要改的。加上 --name、--paused、--budget-usd、--no-limit、--models 或 --all-models。",
			"Nothing to change. Add --name, --paused, --budget-usd, --no-limit, --models or --all-models."))
	}
	if _, _, _, _, err := deviceUpdate(lang(), a); err != nil {
		return nil, err
	}
	return a, nil
}

// splitTarget splits `set iPhone (2) --name x` into the name the person
// typed, unquoted words joined by spaces ("iPhone (2)"), and the flags.
func splitTarget(args []string) (string, []string) {
	i := 0
	for i < len(args) && !strings.HasPrefix(args[i], "-") {
		i++
	}
	return strings.Join(args[:i], " "), args[i:]
}

// modelSetArgs turns `model set <name-or-id> [flags]` into the control
// request's arguments, with only the flags that were given: --name and
// --shown are the console's rename and "show on devices" (--name ""
// restores the source's name); --human-only and the prices change the
// model itself.
func modelSetArgs(target string, args []string) (map[string]string, error) {
	fs := flag.NewFlagSet("model set", flag.ContinueOnError)
	fs.String("name", "", "")
	fs.String("shown", "", "true or false")
	fs.String("human-only", "", "true or false")
	fs.String("price-in", "", "")
	fs.String("price-out", "", "")
	if err := parseFlags(fs, args); err != nil {
		return nil, err
	}
	a := map[string]string{"model": strings.TrimSpace(target)}
	fs.Visit(func(f *flag.Flag) { a[strings.ReplaceAll(f.Name, "-", "_")] = f.Value.String() })
	if a["model"] == "" {
		return nil, say(t("请写出模型的名字或 ID。", "Name the model (its name or ID)."))
	}
	if len(a) == 1 {
		return nil, say(t("没有要改的。加上 --name、--shown、--human-only 或 --price-in 和 --price-out。",
			"Nothing to change. Add --name, --shown, --human-only, or --price-in with --price-out."))
	}
	if _, _, _, err := modelUpdate(lang(), a); err != nil {
		return nil, err
	}
	return a, nil
}

// modelUpdate reads what model set changes: the owner's name and whether
// the model is shown on devices (nil leaves either as it is), and the
// model's own fields. Its sentences are in language l.
func modelUpdate(l string, a map[string]string) (name *string, shown *bool, u hub.ModelUpdate, err error) {
	if v, ok := a["name"]; ok {
		name = &v
	}
	if v, ok := a["shown"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, nil, u, say(tl(l, "--shown 只能是 true 或 false。", "--shown must be true or false."))
		}
		shown = &b
	}
	if v, ok := a["human_only"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, nil, u, say(tl(l, "--human-only 只能是 true 或 false。", "--human-only must be true or false."))
		}
		u.HumanOnly = &b
	}
	pin, inOK := a["price_in"]
	pout, outOK := a["price_out"]
	if inOK || outOK {
		pi, err1 := number(pin)
		po, err2 := number(pout)
		if !inOK || !outOK || err1 != nil || err2 != nil || pi < 0 || po < 0 || pi > 1e6 || po > 1e6 {
			return nil, nil, u, say(tl(l, "请同时给出 --price-in 和 --price-out（美元/百万 token）。", "Give both --price-in and --price-out (USD per million tokens)."))
		}
		u.PriceIn, u.PriceOut = &pi, &po
	}
	return name, shown, u, nil
}

// maxMicroUSD is the largest daily limit or total: $1,000,000.
const maxMicroUSD = 1_000_000_000_000

// number reads an amount as people type it: "5", "2.50", "$2.50". NaN and
// infinities are not numbers.
func number(v string) (float64, error) {
	v = strings.TrimSpace(v)
	v = strings.TrimSpace(strings.TrimPrefix(v, "$"))
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, errors.New("not a number")
	}
	return f, nil
}

// dollars reads a daily limit (or, with total, the daily total) in US
// dollars as micro-USD: above 0 and at most $1,000,000; its sentences are
// in language l.
func dollars(l, v string, total bool) (int64, error) {
	f, err := number(v)
	switch {
	case err != nil || f <= 0 || f*1e6 < 0.5:
		if total {
			return 0, say(tl(l, "每天总额度要大于 0（美元），或者用 --no-limit。", "The daily total must be above 0 (USD), or use --no-limit."))
		}
		return 0, say(tl(l, "每天额度要大于 0（美元），或者用 --no-limit。", "The daily limit must be above 0 (USD), or use --no-limit."))
	case f > maxMicroUSD/1e6:
		if total {
			return 0, say(tl(l, "每天总额度最多 $1,000,000。", "The daily total can be at most $1,000,000."))
		}
		return 0, say(tl(l, "每天额度最多 $1,000,000。", "The daily limit can be at most $1,000,000."))
	}
	return int64(math.Round(f * 1e6)), nil
}

// dailyTotalArgs reads `daily-total <usd> | --no-limit` as micro-USD (or
// hub.NoLimit).
func dailyTotalArgs(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, usageErr(t("用法：omnillm hub daily-total <美元> | --no-limit，例如 omnillm hub daily-total 20",
			"Usage: omnillm hub daily-total <usd> | --no-limit, e.g. omnillm hub daily-total 20"))
	}
	if args[0] == "--no-limit" {
		return hub.NoLimit, nil
	}
	if strings.HasPrefix(args[0], "--") {
		return 0, usageErr(fmt.Sprintf(t("不认识选项 %s。", "Unknown option %s."), args[0]))
	}
	return dollars(lang(), args[0], true)
}

// deviceUpdate reads the fields to change; nil leaves a field as it is.
// The hub checks again what the terminal sent; its sentences are in
// language l. Models are as the person named them (names or IDs).
func deviceUpdate(l string, a map[string]string) (name *string, budget *int64, paused *bool, models *[]string, err error) {
	if v, ok := a["name"]; ok {
		if strings.TrimSpace(v) == "" {
			return nil, nil, nil, nil, say(tl(l, "名字不能是空的。", "The name can’t be empty."))
		}
		name = &v
	}
	if v, ok := a["paused"]; ok {
		p, err := strconv.ParseBool(v)
		if err != nil {
			return nil, nil, nil, nil, say(tl(l, "--paused 只能是 true 或 false。", "--paused must be true or false."))
		}
		paused = &p
	}
	_, noLimit := a["no_limit"]
	if v, ok := a["budget_usd"]; ok {
		if noLimit {
			return nil, nil, nil, nil, say(tl(l, "--budget-usd 和 --no-limit 只能选一个。", "Use either --budget-usd or --no-limit, not both."))
		}
		b, err := dollars(l, v, false)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		budget = &b
	} else if noLimit && a["no_limit"] != "false" {
		b := hub.NoLimit
		budget = &b
	}
	_, all := a["all_models"]
	if v, ok := a["models"]; ok {
		if all {
			return nil, nil, nil, nil, say(tl(l, "--models 和 --all-models 只能选一个。", "Use either --models or --all-models, not both."))
		}
		var ms []string
		for _, m := range strings.Split(v, ",") {
			if m = strings.TrimSpace(m); m != "" {
				ms = append(ms, m)
			}
		}
		if len(ms) == 0 {
			return nil, nil, nil, nil, say(tl(l, "--models 里没有模型。要开放全部模型，用 --all-models。", "--models lists no model. To allow every model, use --all-models."))
		}
		models = &ms
	} else if all && a["all_models"] != "false" {
		ms := []string{} // empty: every model, including ones added later
		models = &ms
	}
	return name, budget, paused, models, nil
}

// findDevice finds a device by its ID, or by its name (exactly, or ignoring
// case); a name more than one device has is refused with their IDs. Its
// sentences are in language l.
func findDevice(l string, ds []hub.Device, q string) (hub.Device, error) {
	return findNamed(l, ds, q, func(d hub.Device) (string, []string) { return d.ID, []string{d.Name} },
		tl(l, "没有叫“%s”的设备。运行 omnillm hub devices 查看。", "No device is called “%s”. Run omnillm hub devices to see them."),
		tl(l, "有几台设备都叫“%s”。请用 ID：%s", "Several devices are called “%s”. Use the ID: %s"))
}

// findApp is findDevice for apps.
func findApp(l string, as []hub.App, q string) (hub.App, error) {
	return findNamed(l, as, q, func(a hub.App) (string, []string) { return a.ID, []string{a.Name} },
		tl(l, "没有叫“%s”的应用。运行 omnillm hub apps 查看。", "No app is called “%s”. Run omnillm hub apps to see them."),
		tl(l, "有几个应用都叫“%s”。请用 ID：%s", "Several apps are called “%s”. Use the ID: %s"))
}

// findModel is findDevice for models: by ID, by the name the model goes by
// (the owner's own, or its source's), as omnillm hub models shows it in
// either language, or as the console lists it ("… · 订阅").
func findModel(l string, ms []hub.ModelView, q string) (hub.ModelView, error) {
	return findNamed(l, ms, q, func(m hub.ModelView) (string, []string) {
		sub := provider.Kind(m.Kind).Subscription()
		var names []string
		for _, ln := range []string{"zh", "en"} {
			for _, n := range []string{m.DisplayName, modelNameIn(ln, m.ModelInfo)} {
				names = append(names, n)
				if sub {
					names = append(names, withDot(n, tl(ln, "订阅", "subscription")))
				}
			}
		}
		return m.ID, names
	},
		tl(l, "没有叫“%s”的模型。运行 omnillm hub models 查看。", "No model is called “%s”. Run omnillm hub models to see them."),
		tl(l, "有几个模型都叫“%s”。请用 ID：%s", "Several models are called “%s”. Use the ID: %s"))
}

// withDot is "a · b" as the console writes it, with no space after a
// full-width bracket ("ChatGPT（Codex · 最新版）· 订阅").
func withDot(a, b string) string {
	if strings.HasSuffix(a, "）") || strings.HasSuffix(a, "」") || strings.HasSuffix(a, "』") || strings.HasSuffix(a, "】") {
		return a + "· " + b
	}
	return a + " · " + b
}

// findProvider is findDevice for model sources: by the source's own name
// ("ollama", "groq") or as omnillm hub providers shows it in either
// language ("Claude 订阅", "Claude subscription").
func findProvider(l string, ps []hub.ProviderInfo, q string) (hub.ProviderInfo, error) {
	return findNamed(l, ps, q, func(p hub.ProviderInfo) (string, []string) {
		names := []string{sourceNameIn("zh", p), sourceNameIn("en", p)}
		switch p.Kind {
		case provider.ClaudeCode:
			names = append(names, "claude") // as in omnillm login claude
		case provider.Codex:
			names = append(names, "chatgpt")
		}
		return p.Name, names
	},
		tl(l, "没有叫“%s”的模型来源。运行 omnillm hub providers 查看。", "No model source is called “%s”. Run omnillm hub providers to see them."),
		tl(l, "有几个模型来源都叫“%s”。请用 ID：%s", "Several model sources are called “%s”. Use the ID: %s"))
}

// findNamed finds the one item whose ID or one of whose names is q: the
// ID first, then the name exactly, then ignoring case. None, or a name
// several items share, is refused; several lists their IDs.
func findNamed[T any](l string, items []T, q string, idNames func(T) (string, []string), none, several string) (T, error) {
	var zero T
	q = strings.TrimSpace(q)
	for _, x := range items {
		if id, _ := idNames(x); id == q {
			return x, nil
		}
	}
	for _, same := range []func(string) bool{
		func(n string) bool { return n == q },
		func(n string) bool { return strings.EqualFold(n, q) },
	} {
		var found []T
		var ids []string
		for _, x := range items {
			id, names := idNames(x)
			for _, n := range names {
				if n != "" && same(n) {
					found, ids = append(found, x), append(ids, id)
					break
				}
			}
		}
		switch len(found) {
		case 0:
			continue
		case 1:
			return found[0], nil
		}
		return zero, sayf(several, q, strings.Join(ids, tl(l, "、", ", ")))
	}
	return zero, sayf(none, q)
}
