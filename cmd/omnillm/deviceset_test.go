package main

import (
	"strings"
	"testing"

	"github.com/wensuper/OmniLLM/hub"
)

func TestDeviceSetArgs(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	a, err := deviceSetArgs("iPhone", []string{"--name", "Work phone", "--paused", "true", "--budget-usd", "2.5", "--models", "openai/gpt-5, ollama/qwen3"})
	if err != nil {
		t.Fatal(err)
	}
	name, budget, paused, models, err := deviceUpdate(lang(), a)
	if err != nil || *name != "Work phone" || *budget != 2_500_000 || !*paused || len(*models) != 2 || (*models)[1] != "ollama/qwen3" {
		t.Fatalf("%v %v", a, err)
	}
	// Only what was given changes.
	a, _ = deviceSetArgs("iPhone", []string{"--no-limit"})
	name, budget, paused, models, err = deviceUpdate(lang(), a)
	if err != nil || name != nil || paused != nil || models != nil || *budget != hub.NoLimit {
		t.Fatalf("%v %v", a, err)
	}
	a, _ = deviceSetArgs("iPhone", []string{"--all-models", "--paused=false"})
	_, budget, paused, models, err = deviceUpdate(lang(), a)
	if err != nil || budget != nil || *paused || models == nil || len(*models) != 0 {
		t.Fatalf("%v %v", a, err)
	}
	for _, bad := range [][]string{
		{},
		{"--budget-usd", "3", "--no-limit"},
		{"--models", "a/b", "--all-models"},
		{"--budget-usd", "0"},
		{"--budget-usd", "lots"},
		{"--paused", "maybe"},
		{"--name", " "},
		{"--models", " , "},
		{"--colour", "red"},
		{"--name", "x", "extra"},
	} {
		if _, err := deviceSetArgs("iPhone", bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	if _, err := deviceSetArgs(" ", []string{"--paused", "true"}); err == nil {
		t.Error("no device named")
	}
}

func TestFindDevice(t *testing.T) {
	ds := []hub.Device{{ID: "d1", Name: "iPhone"}, {ID: "d2", Name: "iPad"}, {ID: "d3", Name: "ipad"}}
	for q, want := range map[string]string{"d1": "d1", "iPhone": "d1", "iphone": "d1", "iPad": "d2", "ipad": "d3"} {
		if d, err := findDevice(lang(), ds, q); err != nil || d.ID != want {
			t.Errorf("%q: %v %v", q, d.ID, err)
		}
	}
	if _, err := findDevice(lang(), ds, "IPAD"); err == nil {
		t.Error("ambiguous name accepted")
	}
	if _, err := findDevice(lang(), ds, "Pixel"); err == nil {
		t.Error("unknown name accepted")
	}
}

func TestFindApp(t *testing.T) {
	as := []hub.App{{ID: "a1", Name: "backup"}, {ID: "a2", Name: "Notes"}, {ID: "a3", Name: "notes"}}
	for q, want := range map[string]string{"a1": "a1", "backup": "a1", "BACKUP": "a1", " Notes ": "a2", "notes": "a3"} {
		if a, err := findApp(lang(), as, q); err != nil || a.ID != want {
			t.Errorf("%q: %v %v", q, a.ID, err)
		}
	}
	if _, err := findApp(lang(), as, "NOTES"); err == nil {
		t.Error("ambiguous name accepted")
	}
	if _, err := findApp(lang(), as, "x"); err == nil || !strings.Contains(describe(err), "omnillm hub apps") {
		t.Errorf("unknown name: %v", err)
	}
}
