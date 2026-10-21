package main

import "unicode"

// upperFirst capitalises a sentence's first letter for English. Chinese
// text comes back as it is: slicing its first byte would cut a character
// in half.
func upperFirst(s string) string {
	for i, r := range s {
		if r < unicode.MaxASCII && unicode.IsLower(r) {
			return string(unicode.ToUpper(r)) + s[i+1:]
		}
		return s
	}
	return s
}
