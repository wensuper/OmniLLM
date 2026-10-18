package provider

import "testing"

func FuzzParseEvents(f *testing.F) {
	f.Add(`{"choices":[{"delta":{"content":"x"}}]}`)
	f.Add(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"x"}}`)
	f.Add(`[DONE]`)
	f.Fuzz(func(t *testing.T, data string) {
		s := &stream{}
		_, _, _ = parseOpenAI(s, "", data)
		_, _, _ = parseAnthropic(s, "", data)
		if s.usage.InputTokens < 0 && s.usage.OutputTokens < 0 {
			_ = s
		}
	})
}
