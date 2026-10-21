package main

import "testing"

func TestUpperFirstLeavesChineseWhole(t *testing.T) {
	for in, want := range map[string]string{
		"this Mac only":  "This Mac only",
		"只在这台 Mac 上":     "只在这台 Mac 上",
		"这台 Mac 所在的地区变了": "这台 Mac 所在的地区变了",
		"":               "",
		"Already":        "Already",
	} {
		if got := upperFirst(in); got != want {
			t.Errorf("upperFirst(%q) = %q, want %q", in, got, want)
		}
	}
}
