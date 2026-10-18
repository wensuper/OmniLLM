package omnillm

import (
	"fmt"
	"strings"
	"time"
)

// Message keys. Every text a person may see comes from this table: it says
// what happened and what to do next, without codes or internal names
// (principle 8).
const (
	msgBadModel          = "bad_model"
	msgBadMessages       = "bad_messages"
	msgBadRole           = "bad_role"
	msgTooLarge          = "too_large"
	msgBadMaxTokens      = "bad_max_tokens"
	msgBadTemperature    = "bad_temperature"
	msgBadSchema         = "bad_schema"
	msgBadCaller         = "bad_caller"
	msgInternal          = "internal"
	MsgUnsupportedFormat = "unsupported_format"
	MsgInvalidToken      = "invalid_token"
	MsgModelNotAllowed   = "model_not_allowed"
	MsgHumanOnly         = "human_only"
	MsgUnknownModel      = "unknown_model"
	MsgBusyQueue         = "busy_queue"
	MsgBusyUpstream      = "busy_upstream"
	MsgBreakerOpen       = "breaker_open"
	MsgBudgetDevice      = "budget_device"
	MsgBudgetGlobal      = "budget_global"
	MsgHubOffline        = "hub_offline"
	MsgUpstream          = "upstream"
	MsgCredentialBad     = "credential_bad"
	MsgCredentialMissing = "credential_missing"
	MsgRegionChanged     = "region_changed"
	MsgRegionUnsupported = "region_unsupported"
	MsgDeviceRevoked     = "device_revoked"
	MsgPairingExpired    = "pairing_expired"
	MsgPairingDenied     = "pairing_denied"
	MsgPairingFailed     = "pairing_failed"
	MsgToolNotAllowed    = "tool_not_allowed"
	MsgNotFound          = "not_found"
	MsgRelayUnavailable  = "relay_unavailable"
	MsgBrowserRefused    = "browser_refused"
	MsgWrongHost         = "wrong_host"
)

var texts = map[string][2]string{ // zh, en
	msgBadModel:          {"请求里没有指定模型，或者模型名太长。", "The request names no model, or the model name is too long."},
	msgBadMessages:       {"请求里没有消息，或者消息太多。", "The request has no messages, or too many."},
	msgBadRole:           {"消息的角色只能是 system、user 或 assistant。", "A message role must be system, user or assistant."},
	msgTooLarge:          {"请求内容太大（上限 4 MB），请缩短后再试。", "The request is too large (4 MB limit). Shorten it and try again."},
	msgBadMaxTokens:      {"回复长度上限不在允许范围内。", "The reply length limit is out of range."},
	msgBadTemperature:    {"temperature 要在 0 到 2 之间。", "temperature must be between 0 and 2."},
	msgBadSchema:         {"结构化输出的 JSON Schema 不是合法的 JSON。", "The structured-output JSON Schema is not valid JSON."},
	msgBadCaller:         {"调用方类型不对。", "Unknown caller type."},
	msgInternal:          {"出了点内部问题，请稍后再试；反复出现请告诉我们。", "Something went wrong inside. Try again later; tell us if it keeps happening."},
	MsgUnsupportedFormat: {"这个模型不支持所要求的结构化输出格式，请改用普通文本。", "This model does not support the requested structured output. Use plain text instead."},
	MsgInvalidToken:      {"令牌缺失、无效或已被移除。请在主机上重新添加这个应用。", "The token is missing, invalid or removed. Add this app again on the host."},
	MsgModelNotAllowed:   {"这个设备或应用不允许使用这个模型。可以在主机上调整它能用的模型。", "This device or app may not use this model. You can change its models on the host."},
	MsgHumanOnly:         {"这个模型只能人手动对话时使用，后台程序不能用，以免账号因自动化使用被封。", "This model is for people in a conversation only, not background programs, so the account is not banned for automated use."},
	MsgUnknownModel:      {"主机上没有这个模型。可以在主机上查看已接入的模型。", "The host has no such model. Check the connected models on the host."},
	MsgBusyQueue:         {"主机正忙，排队已满，请稍后自动重试。", "The host is busy and its queue is full. It will be retried shortly."},
	MsgBusyUpstream:      {"%s 让我们稍等，大约 %s 后再试。", "%s asked us to wait. Try again in about %s."},
	MsgBreakerOpen:       {"%s 最近连续出错，已暂停一会儿，大约 %s 后自动恢复。", "%s kept failing, so calls are paused. They resume in about %s."},
	MsgBudgetDevice:      {"今天的额度用完了，明天恢复，或在主机上调高。", "Today's budget is used up. It resets tomorrow, or raise it on the host."},
	MsgBudgetGlobal:      {"主机今天的总额度用完了，明天恢复，或在主机上调高。", "The host's total budget for today is used up. It resets tomorrow, or raise it on the host."},
	MsgHubOffline:        {"主机现在连不上，可能在睡眠或断网。唤醒后会自动恢复。", "The host can't be reached right now. It may be asleep or offline, and will reconnect when it wakes."},
	MsgUpstream:          {"模型服务出错了，请稍后再试。", "The model service returned an error. Try again later."},
	MsgCredentialBad:     {"%s 的 Key（末 4 位 %s）不可用：可能已失效或余额不足。请在主机上更换。", "The %s key (ending %s) does not work: it may be revoked or out of credit. Replace it on the host."},
	MsgCredentialMissing: {"主机上还没有接入 %s。请先在主机上添加它的 API Key。", "%s is not connected on the host yet. Add its API key on the host first."},
	MsgRegionChanged:     {"主机所在地区变了，为防止账号被判异常，已暂停调用。请在主机上确认继续。", "The host's region changed. To keep the account safe, calls are paused. Confirm on the host to continue."},
	MsgRegionUnsupported: {"%s 不支持主机现在所在的地区，所以没有发出请求。", "%s does not serve the host's current region, so nothing was sent."},
	MsgDeviceRevoked:     {"这台设备已被移除。如需继续使用，请在主机上重新添加。", "This device was removed. Add it again on the host to keep using it."},
	MsgPairingExpired:    {"这个配对码已过期或已用过，请在主机上重新生成。", "This pairing code has expired or was already used. Make a new one on the host."},
	MsgPairingDenied:     {"主机上没有确认这台设备，配对已取消。", "The host did not confirm this device, so pairing was cancelled."},
	MsgPairingFailed:     {"配对没有成功，请在主机上重新生成配对码再试。", "Pairing failed. Make a new pairing code on the host and try again."},
	MsgToolNotAllowed:    {"这个工具不允许从别的设备调用。可以在主机上调整。", "This tool may not be called from another device. You can change that on the host."},
	MsgNotFound:          {"没有找到。", "Not found."},
	MsgBrowserRefused:    {"本机接口不接受网页发来的请求。请用程序直接调用。", "This local interface does not accept requests from web pages. Call it from a program."},
	MsgWrongHost:         {"请用 127.0.0.1 或 localhost 访问本机接口。", "Reach the local interface as 127.0.0.1 or localhost."},
	MsgRelayUnavailable:  {"连接服务暂时不可用，稍后会自动重连。", "The connection service is unavailable. It will reconnect automatically."},
}

// Text renders a message key in lang ("zh…" or anything else for English),
// filling %s placeholders from args.
func Text(lang, key string, args ...any) string {
	t, ok := texts[key]
	if !ok {
		t = texts[msgInternal]
		args = nil
	}
	s := t[1]
	if strings.HasPrefix(strings.ToLower(lang), "zh") {
		s = t[0]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
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
	return Text(lang, key, e.args()...)
}

// args returns the placeholder values, padded so a message never renders
// "%!s(MISSING)".
func (e *Error) args() []any {
	n := strings.Count(texts[e.Msg][1], "%s")
	if n == 0 {
		return nil
	}
	out := make([]any, n)
	for i := range out {
		out[i] = "?"
		if i < len(e.Args) {
			out[i] = e.Args[i]
		}
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
	}
	return msgInternal
}

// RoundDuration renders a wait for people ("30 秒" / "30 seconds").
func RoundDuration(lang string, d time.Duration) string {
	zh := strings.HasPrefix(strings.ToLower(lang), "zh")
	s := int(d.Round(time.Second) / time.Second)
	if s < 1 {
		s = 1
	}
	if s < 120 {
		if zh {
			return fmt.Sprintf("%d 秒", s)
		}
		return fmt.Sprintf("%d seconds", s)
	}
	m := (s + 59) / 60
	if zh {
		return fmt.Sprintf("%d 分钟", m)
	}
	return fmt.Sprintf("%d minutes", m)
}
