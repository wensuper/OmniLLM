package provider

import "testing"

func TestPrettyModel(t *testing.T) {
	for in, want := range map[string]string{
		"claude-opus-5-5":            "Claude Opus 5.5",
		"claude-haiku-5-5":           "Claude Haiku 5.5",
		"claude-sonnet-4-5-20250929": "Claude Sonnet 4.5",
		"claude-opus-4-20250514":     "Claude Opus 4",
		"gpt-5.1-codex":              "GPT-5.1 Codex",
		"gpt-5-codex-max":            "GPT-5 Codex Max",
		"qwen3-32b":                  "qwen3-32b",
	} {
		if got := PrettyModel(in); got != want {
			t.Errorf("PrettyModel(%q) = %q, want %q", in, got, want)
		}
	}
}
