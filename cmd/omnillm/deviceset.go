package main

// `omnillm hub device set`, `app set` and `model set`: the console's
// editors for the terminal. The terminal checks the flags and passes only
// those that were given; the hub finds the device, app or model by ID or
// name and changes those fields alone.

import (
	"flag"
	"io"
	"strconv"
	"strings"

	"github.com/wensuper/OmniLLM/hub"
)

// deviceSetArgs turns `device set <name-or-id> [flags]` into the control
// request's arguments, with only the flags that were given.
func deviceSetArgs(target string, args []string) (map[string]string, error) {
	return setArgs("device", t("用法：omnillm hub device set <名字或ID> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models a,b | --all-models]",
		"Usage: omnillm hub device set <name-or-id> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models a,b | --all-models]"),
		t("请写出设备的名字或 ID。", "Name the device (its name or ID)."), target, args)
}

// appSetArgs is deviceSetArgs for `app set`.
func appSetArgs(target string, args []string) (map[string]string, error) {
	return setArgs("app", t("用法：omnillm hub app set <名字或ID> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models a,b | --all-models]",
		"Usage: omnillm hub app set <name-or-id> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models a,b | --all-models]"),
		t("请写出应用的名字或 ID。", "Name the app (its name or ID)."), target, args)
}

// setArgs reads the flags device set and app set share; key holds the
// device or app as the person named it.
func setArgs(key, usageLine, nameIt, target string, args []string) (map[string]string, error) {
	fs := flag.NewFlagSet(key+" set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.String("name", "", "")
	fs.String("paused", "", "true or false")
	fs.String("budget-usd", "", "")
	fs.Bool("no-limit", false, "")
	fs.String("models", "", "")
	fs.Bool("all-models", false, "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return nil, usageErr(usageLine)
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
	if _, _, _, _, err := deviceUpdate(a); err != nil {
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
	fs.SetOutput(io.Discard)
	fs.String("name", "", "")
	fs.String("shown", "", "true or false")
	fs.String("human-only", "", "true or false")
	fs.String("price-in", "", "")
	fs.String("price-out", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return nil, usageErr(t("用法：omnillm hub model set <名字或ID> [--name 新名字] [--shown true|false] [--human-only true|false] [--price-in X --price-out Y]",
			"Usage: omnillm hub model set <name-or-id> [--name new-name] [--shown true|false] [--human-only true|false] [--price-in X --price-out Y]"))
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
	if _, _, _, err := modelUpdate(a); err != nil {
		return nil, err
	}
	return a, nil
}

// modelUpdate reads what model set changes: the owner's name and whether
// the model is shown on devices (nil leaves either as it is), and the
// model's own fields.
func modelUpdate(a map[string]string) (name *string, shown *bool, u hub.ModelUpdate, err error) {
	if v, ok := a["name"]; ok {
		name = &v
	}
	if v, ok := a["shown"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, nil, u, say(t("--shown 只能是 true 或 false。", "--shown must be true or false."))
		}
		shown = &b
	}
	if v, ok := a["human_only"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, nil, u, say(t("--human-only 只能是 true 或 false。", "--human-only must be true or false."))
		}
		u.HumanOnly = &b
	}
	pin, inOK := a["price_in"]
	pout, outOK := a["price_out"]
	if inOK || outOK {
		pi, err1 := strconv.ParseFloat(pin, 64)
		po, err2 := strconv.ParseFloat(pout, 64)
		if !inOK || !outOK || err1 != nil || err2 != nil || pi < 0 || po < 0 {
			return nil, nil, u, say(t("请同时给出 --price-in 和 --price-out（美元/百万 token）。", "Give both --price-in and --price-out (USD per million tokens)."))
		}
		u.PriceIn, u.PriceOut = &pi, &po
	}
	return name, shown, u, nil
}

// dailyTotalArgs reads `daily-total <usd> | --no-limit` as micro-USD (or
// hub.NoLimit).
func dailyTotalArgs(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, usageErr(t("用法：omnillm hub daily-total <美元> | --no-limit", "Usage: omnillm hub daily-total <usd> | --no-limit"))
	}
	if args[0] == "--no-limit" {
		return hub.NoLimit, nil
	}
	f, err := strconv.ParseFloat(strings.TrimPrefix(args[0], "$"), 64)
	if err != nil || f <= 0 || f > 1e6 {
		return 0, say(t("每天总额度要大于 0（美元），或者用 --no-limit。", "The daily total must be above 0 (USD), or use --no-limit."))
	}
	return int64(f*1e6 + 0.5), nil
}

// deviceUpdate reads the fields to change; nil leaves a field as it is.
// The hub checks again what the terminal sent.
func deviceUpdate(a map[string]string) (name *string, budget *int64, paused *bool, models *[]string, err error) {
	if v, ok := a["name"]; ok {
		if strings.TrimSpace(v) == "" {
			return nil, nil, nil, nil, say(t("名字不能是空的。", "The name can’t be empty."))
		}
		name = &v
	}
	if v, ok := a["paused"]; ok {
		p, err := strconv.ParseBool(v)
		if err != nil {
			return nil, nil, nil, nil, say(t("--paused 只能是 true 或 false。", "--paused must be true or false."))
		}
		paused = &p
	}
	_, noLimit := a["no_limit"]
	if v, ok := a["budget_usd"]; ok {
		if noLimit {
			return nil, nil, nil, nil, say(t("--budget-usd 和 --no-limit 只能选一个。", "Use either --budget-usd or --no-limit, not both."))
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f <= 0 || f > 1e6 {
			return nil, nil, nil, nil, say(t("每天额度要大于 0（美元），或者用 --no-limit。", "The daily limit must be above 0 (USD), or use --no-limit."))
		}
		b := int64(f*1e6 + 0.5)
		budget = &b
	} else if noLimit && a["no_limit"] != "false" {
		b := hub.NoLimit
		budget = &b
	}
	_, all := a["all_models"]
	if v, ok := a["models"]; ok {
		if all {
			return nil, nil, nil, nil, say(t("--models 和 --all-models 只能选一个。", "Use either --models or --all-models, not both."))
		}
		var ms []string
		for _, m := range strings.Split(v, ",") {
			if m = strings.TrimSpace(m); m != "" {
				ms = append(ms, m)
			}
		}
		if len(ms) == 0 {
			return nil, nil, nil, nil, say(t("--models 里没有模型。要开放全部模型，用 --all-models。", "--models lists no model. To allow every model, use --all-models."))
		}
		models = &ms
	} else if all && a["all_models"] != "false" {
		ms := []string{} // empty: every model, including ones added later
		models = &ms
	}
	return name, budget, paused, models, nil
}

// findDevice finds a device by its ID, or by its name (exactly, or ignoring
// case); a name more than one device has is refused with their IDs.
func findDevice(ds []hub.Device, q string) (hub.Device, error) {
	return findNamed(ds, q, func(d hub.Device) (string, []string) { return d.ID, []string{d.Name} },
		t("没有叫“%s”的设备。运行 omnillm hub devices 查看。", "No device is called “%s”. Run omnillm hub devices to see them."),
		t("有几台设备都叫“%s”。请用 ID：%s", "Several devices are called “%s”. Use the ID: %s"))
}

// findApp is findDevice for apps.
func findApp(as []hub.App, q string) (hub.App, error) {
	return findNamed(as, q, func(a hub.App) (string, []string) { return a.ID, []string{a.Name} },
		t("没有叫“%s”的应用。运行 omnillm hub apps 查看。", "No app is called “%s”. Run omnillm hub apps to see them."),
		t("有几个应用都叫“%s”。请用 ID：%s", "Several apps are called “%s”. Use the ID: %s"))
}

// findModel is findDevice for models: by ID, by the name the model goes by
// (the owner's own, or its source's) or as omnillm hub models shows it.
func findModel(ms []hub.ModelView, q string) (hub.ModelView, error) {
	return findNamed(ms, q, func(m hub.ModelView) (string, []string) { return m.ID, []string{m.DisplayName, modelName(m.ModelInfo)} },
		t("没有叫“%s”的模型。运行 omnillm hub models 查看。", "No model is called “%s”. Run omnillm hub models to see them."),
		t("有几个模型都叫“%s”。请用 ID：%s", "Several models are called “%s”. Use the ID: %s"))
}

// findProvider is findDevice for model sources: by the source's own name
// ("ollama", "groq") or as omnillm hub providers shows it ("Claude 订阅").
func findProvider(ps []hub.ProviderInfo, q string) (hub.ProviderInfo, error) {
	return findNamed(ps, q, func(p hub.ProviderInfo) (string, []string) { return p.Name, []string{sourceName(p)} },
		t("没有叫“%s”的模型来源。运行 omnillm hub providers 查看。", "No model source is called “%s”. Run omnillm hub providers to see them."),
		t("有几个模型来源都叫“%s”。请用 ID：%s", "Several model sources are called “%s”. Use the ID: %s"))
}

// findNamed finds the one item whose ID or one of whose names is q: the
// ID first, then the name exactly, then ignoring case. None, or a name
// several items share, is refused; several lists their IDs.
func findNamed[T any](items []T, q string, idNames func(T) (string, []string), none, several string) (T, error) {
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
		return zero, sayf(several, q, strings.Join(ids, t("、", ", ")))
	}
	return zero, sayf(none, q)
}
