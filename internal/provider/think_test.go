package provider

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// A thinking model's reasoning reaches Chunk.Think, apart from the answer,
// whichever way its provider streams it.
func TestReasoningIsKeptApartFromTheAnswer(t *testing.T) {
	s := &stream{}
	for _, data := range []string{
		`{"choices":[{"delta":{"content":"","reasoning":"Ollama thinks. "}}]}`,
		`{"choices":[{"delta":{"content":"","reasoning_content":"DeepSeek thinks."}}]}`,
	} {
		if text, _, err := parseOpenAI(s, "", data); err != nil || text != "" {
			t.Fatalf("parseOpenAI(%s) = %q, %v; reasoning is not answer", data, text, err)
		}
	}
	if got := s.think.take(); got != "Ollama thinks. DeepSeek thinks." {
		t.Fatalf("OpenAI-compatible reasoning = %q", got)
	}
	if text, _, _ := parseOpenAI(s, "", `{"choices":[{"delta":{"content":"2"}}]}`); text != "2" || s.think.take() != "" {
		t.Fatal("an answer delta carried reasoning")
	}

	if text, _, _ := parseAnthropic(s, "", `{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"Claude thinks."}}`); text != "" {
		t.Fatalf("Anthropic thinking came out as answer %q", text)
	}
	if got := s.think.take(); got != "Claude thinks." {
		t.Fatalf("Anthropic thinking = %q", got)
	}

	c := &cliStream{}
	if text, _ := c.line([]byte(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"Claude Code thinks."}}}`)); text != "" {
		t.Fatalf("Claude Code thinking came out as answer %q", text)
	}
	if text, _ := c.line([]byte(`{"type":"item.completed","item":{"type":"reasoning","text":"Codex thought."}}`)); text != "" {
		t.Fatalf("Codex reasoning came out as answer %q", text)
	}
	if got := c.think.take(); got != "Claude Code thinks.Codex thought.\n\n" {
		t.Fatalf("CLI reasoning = %q", got)
	}
}

// Reasoning past maxThink is dropped, not held in memory or sent, and the
// cut never splits a character.
func TestReasoningIsCapped(t *testing.T) {
	var th thinking
	th.add(strings.Repeat("x", maxThink-3))
	th.add("abcdef")
	th.add("more")
	if got := th.take(); len(got) != maxThink || !strings.HasSuffix(got, "abc") {
		t.Fatalf("kept %d bytes ending %q; want %d ending abc", len(got), got[len(got)-3:], maxThink)
	}
	var zh thinking
	zh.add(strings.Repeat("x", maxThink-4))
	zh.add("思考") // 6 bytes: only the first character fits
	zh.add("x")
	if got := zh.take(); !utf8.ValidString(got) || !strings.HasSuffix(got, "x思") {
		t.Fatalf("cut inside a character or kept the wrong end: %q", got[len(got)-8:])
	}
}

// A server that puts something other than text in "reasoning" still gets
// its answer through.
func TestOddReasoningDoesNotBreakTheAnswer(t *testing.T) {
	s := &stream{}
	text, _, err := parseOpenAI(s, "", `{"choices":[{"delta":{"content":"ok","reasoning":{"summary":"x"},"reasoning_content":null}}]}`)
	if err != nil || text != "ok" || s.think.take() != "" {
		t.Fatalf("got %q, %v", text, err)
	}
}
