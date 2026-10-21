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
