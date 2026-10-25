package omnillm

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Message keys. Every text a person may see comes from this table: it says
// what happened and what to do next, without codes or internal names
// (principle 8).
const (
	msgBadModel           = "bad_model"
	msgBadMessages        = "bad_messages"
	msgBadRole            = "bad_role"
	msgTooLarge           = "too_large"
	msgBadMaxTokens       = "bad_max_tokens"
	msgBadTemperature     = "bad_temperature"
	msgBadSchema          = "bad_schema"
	msgBadCaller          = "bad_caller"
	msgInternal           = "internal"
	MsgUnsupportedFormat  = "unsupported_format"
	MsgInvalidToken       = "invalid_token"
	MsgModelNotAllowed    = "model_not_allowed" // a device; older devices know only this key
	MsgModelNotAllowedApp = "model_not_allowed_app"
	MsgHumanOnly          = "human_only"
	MsgSourceOff          = "source_off"
	MsgUnknownModel       = "unknown_model"
	MsgBusyQueue          = "busy_queue"
	MsgBusyUpstream       = "busy_upstream"
	MsgBreakerOpen        = "breaker_open"
	MsgBudgetDevice       = "budget_device" // a device; older devices know only this key
	MsgBudgetApp          = "budget_app"
	MsgBudgetGlobal       = "budget_global"
	MsgBudgetWouldExceed  = "budget_would_exceed"
	MsgContextTooLong     = "context_too_long"
	MsgHubOffline         = "hub_offline"
	MsgUpstream           = "upstream"
	MsgCredentialBad      = "credential_bad"
	MsgCredentialMissing  = "credential_missing"
	MsgRegionChanged      = "region_changed"
	MsgRegionUnsupported  = "region_unsupported"
	MsgDeviceRevoked      = "device_revoked"
	MsgDevicePaused       = "device_paused"
	MsgAppPaused          = "app_paused"
	MsgPairingExpired     = "pairing_expired"
	MsgPairingDenied      = "pairing_denied"
	MsgPairingFailed      = "pairing_failed"
	MsgPairingDamaged     = "pairing_damaged"
	MsgNotPairing         = "not_pairing"
	MsgToolNotAllowed     = "tool_not_allowed"
	MsgNotFound           = "not_found"
	MsgBadBudget          = "bad_budget"
	MsgRelayUnavailable   = "relay_unavailable"
	MsgBrowserRefused     = "browser_refused"
	MsgWrongHost          = "wrong_host"
	MsgNoChatModel        = "no_chat_model"
	MsgLocalToken         = "local_token"
	MsgCLILogin           = "cli_login"
	MsgCLINotFound        = "cli_not_found"
	MsgSubscriptionLimit  = "subscription_limit"
	MsgCLIUnsafe          = "cli_unsafe"
	MsgSubscriptionPace   = "subscription_pace"
	MsgBusyPace           = "busy_pace"
	MsgBadImage           = "bad_image"
	MsgNoVision           = "no_vision"
	MsgBadSource          = "bad_source"
	MsgKeyMalformed       = "key_malformed"
)

var texts = map[string][2]string{ // zh, en
	msgBadModel:           {"请求里没有指定模型，或者模型名太长。", "The request names no model, or the model name is too long."},
	msgBadMessages:        {"请求里没有消息，或者消息太多。", "The request has no messages, or too many."},
	msgBadRole:            {"消息的角色只能是 system、user 或 assistant。", "A message role must be system, user or assistant."},
	msgTooLarge:           {"请求内容太大（文字加图片上限 5 MB），请缩短或少放几张图再试。", "The request is too large (5 MB of text and images). Shorten it or send fewer images."},
	msgBadMaxTokens:       {"回复长度上限不在允许范围内。", "The reply length limit is out of range."},
	msgBadTemperature:     {"temperature 要在 0 到 2 之间。", "temperature must be between 0 and 2."},
	msgBadSchema:          {"结构化输出的 JSON Schema 不是合法的 JSON。", "The structured-output JSON Schema isn’t valid JSON."},
	msgBadCaller:          {"调用方类型不对。", "Unknown caller type."},
	msgInternal:           {"你的 Mac 上出了点问题。稍后再试；一直这样的话请告诉我们。", "Something went wrong on your Mac. Try again later; if it keeps happening, let us know."},
	MsgUnsupportedFormat:  {"这个模型不支持所要求的结构化输出格式，请改用普通文本。", "This model doesn’t support the requested structured output. Use plain text instead."},
	MsgInvalidToken:       {"令牌缺失、无效或已被移除。在你的 Mac 上重新添加这个应用。", "The token is missing, invalid or removed. Add this app again on your Mac."},
	MsgModelNotAllowed:    {"这台设备不能用这个模型。可以在你的 Mac 上调整它能用的模型。", "This device can’t use this model. You can change its models on your Mac."},
	MsgModelNotAllowedApp: {"这个应用不能用这个模型。可以在你的 Mac 上调整它能用的模型。", "This app can’t use this model. You can change its models on your Mac."},
	MsgSourceOff:          {"这个模型的来源在你的 Mac 上关掉了。在控制台“模型”里把它打开就能用。", "This model’s source is switched off on your Mac. Turn it on under “Models” in the console to use it."},
	MsgHumanOnly:          {"这个模型只能手动对话，脚本和应用不能用，免得账号被当成自动化调用。", "This model is chat-only; scripts and apps can’t use it, so the account isn’t flagged as automated."},
	MsgUnknownModel:       {"你的 Mac 上没有这个模型。可以在 Mac 的控制台“模型”里查看已接入的模型。", "Your Mac has no such model. See the connected models under “Models” in the console on your Mac."},
	MsgBusyQueue:          {"你的 Mac 正忙，过几秒再发一次。", "Your Mac is busy; send again in a few seconds."},
	MsgBusyUpstream:       {"%s 那边正忙，%s后再试。", "%s is busy right now. Try again in %s."},
	MsgBreakerOpen:        {"%s 最近连续出错，已暂停一会儿，%s后自动恢复。", "%s kept failing, so calls are paused. They resume in %s."},
	MsgBudgetDevice:       {"这台设备今天的额度用完了。明天恢复，也可以在你的 Mac 上调高。", "This device’s daily limit is used up. It resets tomorrow, or raise it on your Mac."},
	MsgBudgetApp:          {"这个应用今天的额度用完了。明天恢复，也可以在你的 Mac 上调高。", "This app’s daily limit is used up. It resets tomorrow, or raise it on your Mac."},
	MsgBudgetGlobal:       {"整台 Mac 今天的总额度用完了。明天恢复，也可以在你的 Mac 上调高。", "The daily total for your Mac is used up. It resets tomorrow, or raise it on your Mac."},
	MsgBudgetWouldExceed:  {"这条消息可能超出今天剩下的额度。开个新对话，或在你的电脑上调高额度。", "This message may exceed what’s left of today’s limit. Start a new chat, or raise the limit on your computer."},
	MsgContextTooLong:     {"这段对话太长，%s 装不下了。开一个新对话再问。", "This chat is too long for %s. Start a new chat and ask again."},
	MsgHubOffline:         {"你的电脑现在连不上，可能在睡眠、关机或断网。恢复后会自动连上。", "Your computer can’t be reached right now. It may be asleep, off or offline, and will reconnect when it’s back."},
	MsgUpstream:           {"模型来源出错了，请稍后再试。", "The model source returned an error. Try again later."},
	MsgCredentialBad:      {"%s 的 API Key（末 4 位 %s）不可用：可能已失效或余额不足。请在你的 Mac 上更换。", "The %s API key (ending %s) doesn’t work: it may be revoked or out of credit. Replace it on your Mac."},
	MsgCredentialMissing:  {"你的 Mac 上还没有接入 %s。在 Mac 的控制台“模型”里接入它。", "%s isn’t connected on your Mac. Connect it under “Models” in the console on your Mac."},
	MsgRegionChanged:      {"你的电脑所在地区变了，为防止账号被判异常，已暂停使用模型。请在那台电脑上确认继续。", "Your computer’s region changed. To keep the account safe, models are paused. Confirm on that computer to continue."},
	MsgRegionUnsupported:  {"%s 不支持你的 Mac 现在所在的地区，所以没有发出请求。", "%s doesn’t serve the region your Mac is in now, so nothing was sent."},
	MsgDeviceRevoked:      {"这台设备已从你的电脑上移除。要继续用，在那台电脑上重新添加它。", "This device was removed from your computer. To use it again, add it on that computer again."},
	MsgDevicePaused:       {"这台设备已在你的 Mac 上暂停。需要时在 Mac 的控制台“设备”里重新开启。", "This device is paused on your Mac. Turn it back on under “Devices” in the console on your Mac when needed."},
	MsgAppPaused:          {"这个应用已在你的 Mac 上暂停。在控制台“应用”里重新开启。", "This app is paused on your Mac. Turn it back on under “Apps” in the console."},
	MsgPairingExpired:     {"这个二维码或配对文字已过期或已用过。换一个新的再试（控制台里点“添加设备”，或再运行 omnillm pair）。", "This QR code or pairing text expired or was already used. Get a new one (“Add device” in the console, or run omnillm pair again)."},
	MsgPairingDenied:      {"电脑上没有确认。如果是你本人，换一个新的二维码或配对文字再试。", "It wasn’t confirmed on the computer. If this is you, get a new QR code or pairing text and try again."},
	MsgPairingFailed:      {"这个二维码或配对文字已过期或已用过。换一个新的再试（控制台里点“添加设备”，或再运行 omnillm pair）。", "This QR code or pairing text expired or was already used. Get a new one (“Add device” in the console, or run omnillm pair again)."},
	MsgPairingDamaged:     {"配对文字不完整或被改动了。重新完整复制一次。", "The pairing text is incomplete or altered. Copy it again, in full."},
	MsgNotPairing:         {"这不是 OmniLLM 的配对文字。在控制台的“添加设备”里复制，或在你的服务器上运行 omnillm pair。", "This isn’t an OmniLLM pairing text. Copy it from “Add device” in the console, or run omnillm pair on your server."},
	MsgToolNotAllowed:     {"这个工具不允许从别的设备调用。可以在你的 Mac 上调整。", "This tool may not be called from another device. You can change that on your Mac."},
	MsgNotFound:           {"没有找到。", "Not found."},
	MsgBadBudget:          {"每天额度要大于 0，或者选“不设上限”。", "The daily limit must be above 0, or choose “No limit”."},
	MsgBrowserRefused:     {"本机接口不接受网页发来的请求。请用程序直接调用。", "This local interface doesn’t accept requests from web pages. Call it from a program."},
	MsgWrongHost:          {"请用 127.0.0.1 或 localhost 访问本机接口。", "Reach the local interface as 127.0.0.1 or localhost."},
	MsgNoChatModel:        {"这台 Mac 上的 Ollama 里没有能对话的模型（只有向量 / 重排模型）。在终端运行 ollama pull qwen3 下载一个。", "Ollama on this Mac has no chat model (only embedding or rerank models). Run ollama pull qwen3 in Terminal to get one."},
	MsgLocalToken:         {"本机接口的令牌缺失或不对。令牌在这台设备 OmniLLM 数据文件夹里的 device/local.token 文件中（只有本用户能读）。", "The local token is missing or wrong. It is in device/local.token inside OmniLLM’s data folder on this device (readable only by you)."},
	MsgRelayUnavailable:   {"远程连接暂时不可用。检查你的 Mac 的网络后再试一次。", "Remote access is unavailable right now. Check your Mac’s network, then try again."},
	MsgCLILogin:           {"你的电脑上的 %s 没有登录或登录已过期。在那台电脑的终端里%s，然后再试。", "%s on your computer is signed out. In a terminal on that computer, %s, then try again."},
	MsgCLINotFound:        {"要用 %s，你的 Mac 上需要先安装 %s 并登录。", "To use %s, install %s on your Mac and log in first."},
	MsgCLIUnsafe:          {"你的 Mac 上这个版本的 %s 关不掉工具（能运行命令或读文件），为了安全暂时不用它。请更新 OmniLLM。", "This version of %s on your Mac couldn’t be kept from using tools (running commands or reading files), so OmniLLM won’t use it for now. Update OmniLLM."},
	MsgBadImage:           {"图片不能用：只支持 PNG、JPEG、GIF、WebP，每张不超过 2 MB，每条消息最多 4 张。", "That image can’t be used: PNG, JPEG, GIF or WebP only, up to 2 MB each and 4 per message."},
	MsgNoVision:           {"这个模型看不了图片。换一个能看图的模型，或者只发文字。", "This model can’t read images. Pick another model, or send text only."},
	MsgBusyPace:           {"订阅按一个人聊天的节奏使用，两次请求之间要隔开一点。%s后再试。", "Subscriptions are used at one person’s chatting pace, so requests are spaced out. Try again in %s."},
	MsgSubscriptionPace:   {"这台设备用订阅模型太频繁了，超过了一个人聊天的速度。%s后再试，或者换一个用 API Key 或你的 Mac 上的模型，免得账号被当成自动化调用。", "This device is using the subscription models faster than a person chats. Try again in %s, or pick a model that uses an API key or runs on your Mac, so the account isn’t flagged as automated."},
	MsgBadSource:          {"名字只能用英文字母、数字和 -（不要空格）；接口地址要以 https:// 开头。", "Use letters, digits and - for the name (no spaces), and an address starting with https://."},
	MsgKeyMalformed:       {"这不像完整的 API Key。到服务商网站重新复制一次，再粘贴。", "That doesn’t look like a complete API key. Copy it again from the provider’s site and paste it."},
	MsgSubscriptionLimit:  {"%s 的用量到了上限，%s后恢复。现在可以换一个用 API Key 或你的 Mac 上的模型。", "%s has reached its usage limit. It resets in %s. Meanwhile, pick a model that uses an API key or runs on your Mac."},
}

// slot says what a placeholder holds, so Describe can render it in the
// reader's language instead of passing on whatever the hub wrote.
type slot int

const (
	slotPlain    slot = iota // shown as given
	slotProvider             // a provider name or kind ("claude-code", "openai")
	slotProgram              // the program a subscription runs ("Claude Code")
	slotLogin                // how to log that program in again
	slotWait                 // a wait, taken from RetryAfter
)

// slots lists the placeholders that need rendering, by position. Program
// and login slots are worked out from the provider kind in Args; a wait
// slot comes from RetryAfter (an older hub also wrote it, in English, into
// Args at the same position, which is used only when RetryAfter is 0).
var slots = map[string][]slot{
	MsgBusyUpstream:      {slotProvider, slotWait},
	MsgBreakerOpen:       {slotProvider, slotWait},
	MsgBusyPace:          {slotWait},
	MsgSubscriptionPace:  {slotWait},
	MsgSubscriptionLimit: {slotProvider, slotWait},
	MsgCredentialBad:     {slotProvider, slotPlain},
	MsgCredentialMissing: {slotProvider},
	MsgRegionUnsupported: {slotProvider},
	MsgCLILogin:          {slotProgram, slotLogin},
	MsgCLINotFound:       {slotProvider, slotProgram},
	MsgCLIUnsafe:         {slotProgram},
}

// skipArgs is how many leading Args a text no longer shows: the busy-pace
// text stopped naming the subscription, while the hub still sends it first
// (and an older hub its wait second).
var skipArgs = map[string]int{MsgBusyPace: 1}

// providerNames are how people know a provider, zh and en. Subscription
// programs are named by the subscription the person pays for. The old
// display names are here too, for errors from an older hub.
var providerNames = map[string][2]string{
	"claude-code":                  {"Claude 订阅", "your Claude subscription"},
	"codex":                        {"ChatGPT 订阅", "your ChatGPT subscription"},
	"Claude (subscription)":        {"Claude 订阅", "your Claude subscription"},
	"ChatGPT Codex (subscription)": {"ChatGPT 订阅", "your ChatGPT subscription"},
	"openai":                       {"OpenAI", "OpenAI"},
	"anthropic":                    {"Anthropic", "Anthropic"},
	"ollama":                       {"Ollama", "Ollama"},
}

// subscriptionOf finds which subscription program args are about:
// "claude", "codex" or "".
func subscriptionOf(args []string) string {
	for _, a := range args {
		switch a {
		case "claude-code", "claude", "Claude (subscription)":
			return "claude"
		case "codex", "ChatGPT Codex (subscription)":
			return "codex"
		}
	}
	return ""
}

func isZh(lang string) bool { return strings.HasPrefix(strings.ToLower(lang), "zh") }

func pick(lang, zh, en string) string {
	if isZh(lang) {
		return zh
	}
	return en
}

// render returns the value for one slot in lang.
func (e *Error) render(lang string, s slot, i int) string {
	raw := "?"
	if i < len(e.Args) && e.Args[i] != "" {
		raw = e.Args[i]
	}
	switch s {
	case slotProvider:
		if n, ok := providerNames[raw]; ok {
			return pick(lang, n[0], n[1])
		}
	case slotProgram:
		switch subscriptionOf(e.Args) {
		case "claude":
			return "Claude Code"
		case "codex":
			return "Codex"
		}
	case slotLogin:
		switch subscriptionOf(e.Args) {
		case "claude":
			return pick(lang, "运行 omnillm login claude", "run omnillm login claude")
		case "codex":
			return pick(lang, "运行 omnillm login chatgpt", "run omnillm login chatgpt")
		}
	case slotWait:
		if e.RetryAfter > 0 {
			return RoundDuration(lang, e.RetryAfter)
		}
	}
	return raw
}

// hanSpace is a space between two Chinese characters, left when a
// provider name such as "Claude 订阅" fills "%s 的".
var hanSpace = regexp.MustCompile(`(\p{Han}) (\p{Han})`)

// Text renders a message key in lang ("zh…" or anything else for English),
// filling %s placeholders from args.
func Text(lang, key string, args ...any) string {
	t, ok := texts[key]
	if !ok {
		t = texts[msgInternal]
		args = nil
	}
	s := pick(lang, t[0], t[1])
	if len(args) == 0 {
		return s
	}
	out := fmt.Sprintf(s, args...)
	if isZh(lang) {
		return hanSpace.ReplaceAllString(out, "$1$2")
	}
	if strings.HasPrefix(s, "%s") && strings.HasPrefix(out, "your ") {
		out = "Y" + out[1:] // "your Claude subscription …" opens a sentence
	}
	return out
}

// Describe is the person-facing sentence for err in lang.
func Describe(lang string, err error) string {
	e := AsError(err)
	if e == nil {
		return ""
	}
	key := e.Msg
	if key == "" {
		key = defaultMsg(e.Code)
	}
	return Text(lang, key, e.args(lang, key)...)
}

// args returns the placeholder values rendered for lang, padded so a
// message never renders "%!s(MISSING)".
func (e *Error) args(lang, key string) []any {
	n := strings.Count(texts[key][1], "%s")
	if n == 0 {
		return nil
	}
	kinds := slots[key]
	out := make([]any, n)
	for i := range out {
		s := slotPlain
		if i < len(kinds) {
			s = kinds[i]
		}
		out[i] = e.render(lang, s, i+skipArgs[key])
	}
	return out
}

func defaultMsg(c Code) string {
	switch c {
	case CodeUnsupportedFormat:
		return MsgUnsupportedFormat
	case CodeInvalidToken:
		return MsgInvalidToken
	case CodeModelNotAllowed:
		return MsgModelNotAllowed
	case CodeBusy:
		return MsgBusyQueue
	case CodeBudgetExceeded:
		return MsgBudgetDevice
	case CodeHubOffline:
		return MsgHubOffline
	case CodeUpstream:
		return MsgUpstream
	case CodePairingDenied:
		return MsgPairingDenied
	case CodePairingFailed:
		return MsgPairingFailed
	case CodeToolNotAllowed:
		return MsgToolNotAllowed
	case CodeNotFound:
		return MsgNotFound
	case CodeDevicePaused:
		return MsgDevicePaused
	}
	return msgInternal
}

// RoundDuration renders a wait for people, rounded to what a person plans
// by ("约 30 秒" / "about 30 seconds", "约 20 分钟", "约 3 小时").
func RoundDuration(lang string, d time.Duration) string {
	zh := isZh(lang)
	s := int(d.Round(time.Second) / time.Second)
	if s < 1 {
		s = 1
	}
	n, zu, en := s, "秒", "second"
	switch {
	case s >= 90*60:
		n, zu, en = (s+1800)/3600, "小时", "hour"
	case s >= 90:
		n, zu, en = (s+59)/60, "分钟", "minute"
	}
	if zh {
		return fmt.Sprintf("约 %d %s", n, zu)
	}
	if n != 1 {
		en += "s"
	}
	return fmt.Sprintf("about %d %s", n, en)
}

// ForHub says "your computer" where a message says "your Mac", unless the
// hub is a Mac (hubOS "darwin"; "" while not known counts as not one).
func ForHub(lang, hubOS, s string) string {
	if hubOS == "darwin" {
		return s
	}
	if isZh(lang) {
		return zhMac.ReplaceAllString(s, "电脑")
	}
	return enMac.ReplaceAllString(s, "computer")
}

var (
	zhMac = regexp.MustCompile(` ?Mac ?`)
	enMac = regexp.MustCompile(`\bMac\b`)
)
