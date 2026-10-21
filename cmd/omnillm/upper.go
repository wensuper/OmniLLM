package main

import (
	"unicode"
	"unicode/utf8"
)

// upperFirst capitalises a sentence's first letter for English. Chinese
// text comes back as it is: slicing its first byte would cut a character
// in half.
func upperFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r < unicode.MaxASCII && unicode.IsLower(r) {
		return string(unicode.ToUpper(r)) + s[n:]
	}
	return s
}
