package main

// `omnillm hub device set`: the console's device editor for the terminal.
// The terminal checks the flags and passes only those that were given; the
// hub finds the device by ID or name and changes those fields alone.

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
	fs := flag.NewFlagSet("device set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.String("name", "", "")
	fs.String("paused", "", "true or false")
	fs.String("budget-usd", "", "")
	fs.Bool("no-limit", false, "")
	fs.String("models", "", "")
	fs.Bool("all-models", false, "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return nil, say(t("用法：omnillm hub device set <名字或ID> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models a,b | --all-models]",
			"Usage: omnillm hub device set <name-or-id> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models a,b | --all-models]"))
	}
	a := map[string]string{"device": strings.TrimSpace(target)}
	fs.Visit(func(f *flag.Flag) { a[strings.ReplaceAll(f.Name, "-", "_")] = f.Value.String() })
	if a["device"] == "" {
		return nil, say(t("请写出设备的名字或 ID。", "Name the device (its name or ID)."))
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
// case when only one matches).
func findDevice(ds []hub.Device, q string) (hub.Device, error) {
	q = strings.TrimSpace(q)
	for _, d := range ds {
		if d.ID == q || d.Name == q {
			return d, nil
		}
	}
	var found []hub.Device
	for _, d := range ds {
		if strings.EqualFold(d.Name, q) {
			found = append(found, d)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return hub.Device{}, sayf(t("没有叫“%s”的设备。运行 omnillm hub devices 查看。", "No device is called “%s”. Run omnillm hub devices to see them."), q)
	}
	return hub.Device{}, sayf(t("有几台设备都叫“%s”（大小写不同）。请照 omnillm hub devices 里的写法写出名字。",
		"Several devices are called “%s” (in different case). Write the name exactly as omnillm hub devices shows it."), q)
}
