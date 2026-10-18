// Package redact removes credential-shaped strings from text before it
// reaches any log (principle 3). It is a last line of defence: credentials
// are never put into log calls in the first place.
package redact

import "regexp"

var patterns = []*regexp.Regexp{
	// Authorization-style headers and "key: value" pairs.
	regexp.MustCompile(`(?i)(authorization|x-api-key|api[-_]?key|cookie|set-cookie|bearer|token|secret|password)(["']?\s*[:=]\s*["']?|\s+)[^\s"',;]+`),
	// Provider key shapes.
	regexp.MustCompile(`\bsk-[A-Za-z0-9_\-]{8,}`),
	regexp.MustCompile(`\bomni_[a-z]+_[A-Za-z0-9_\-]{8,}`),
	// JWTs.
	regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}`),
	// Long opaque base64/hex runs (pairing secrets, keys).
	regexp.MustCompile(`\b[A-Za-z0-9_\-]{40,}\b`),
}

// String returns s with anything credential-shaped replaced by [redacted].
func String(s string) string {
	for i, p := range patterns {
		if i == 0 {
			s = p.ReplaceAllString(s, "${1}${2}[redacted]")
			continue
		}
		s = p.ReplaceAllString(s, "[redacted]")
	}
	return s
}

// Last4 returns the last four characters for display ("…a1b2"), or "????"
// when the value is too short to show any of it safely.
func Last4(secret string) string {
	if len(secret) < 24 {
		return "????"
	}
	return secret[len(secret)-4:]
}
