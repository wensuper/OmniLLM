package provider

import (
	"regexp"
	"strings"
)

var claudeID = regexp.MustCompile(`^claude-(opus|sonnet|haiku|fable)-(\d+)(?:-(\d{1,2}))?(?:-\d{8})?$`)

// PrettyModel names an exact model id for people, with its version:
// "claude-opus-5-5" -> "Claude Opus 5.5", "claude-haiku-4-5-20251001" ->
// "Claude Haiku 4.5", "gpt-5.1-codex" -> "GPT-5.1 Codex". Ids it does not
// know come back as they are.
func PrettyModel(id string) string {
	if m := claudeID.FindStringSubmatch(id); m != nil {
		v := m[2]
		if m[3] != "" {
			v += "." + m[3]
		}
		return "Claude " + strings.ToUpper(m[1][:1]) + m[1][1:] + " " + v
	}
	if rest, ok := strings.CutPrefix(id, "gpt-"); ok && rest != "" {
		parts := strings.Split(rest, "-")
		out := "GPT-" + parts[0]
		for _, w := range parts[1:] {
			out += " " + strings.ToUpper(w[:1]) + w[1:]
		}
		return out
	}
	return id
}

// modelName names the model a call went to, for a sentence such as "this
// chat is too long for Claude Sonnet". Subscription aliases follow the
// newest model, so they are named by family.
func modelName(k Kind, upstream string) string {
	switch {
	case k == Codex:
		return "ChatGPT"
	case k == ClaudeCode && upstream != "" && upstream != "default":
		return "Claude " + strings.ToUpper(upstream[:1]) + upstream[1:]
	case upstream == "":
		return string(k)
	}
	return PrettyModel(upstream)
}

// tooLong recognises a provider saying the conversation does not fit the
// model's context: OpenAI's "maximum context length" and
// "context_length_exceeded", Anthropic's and Claude Code's "prompt is too
// long", Codex's "exceeds the context window".
func tooLong(lower string) bool {
	// OpenAI, Anthropic, vLLM, Ollama; llama.cpp's llama-server ("exceeds
	// the available context size", exceed_context_size_error); LM Studio
	// ("context size has been exceeded").
	for _, w := range []string{"context_length", "context length", "context window", "maximum context", "too long", "too many tokens",
		"context size", "exceed_context_size"} {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}
