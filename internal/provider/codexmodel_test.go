package provider

import "testing"

func TestFirstCodexModelIsTheTopOfItsOwnPicker(t *testing.T) {
	cat := []byte(`{"models":[
		{"slug":"gpt-reserve","display_name":"GPT-Reserve","priority":1,"visibility":"hide"},
		{"slug":"gpt-6-sol","display_name":"GPT-6-Sol","priority":4,"visibility":"list"},
		{"slug":"gpt-6-astra","display_name":"GPT-6-Astra","priority":2,"visibility":"list"},
		{"slug":"codex-auto-review","display_name":"Codex Auto Review","priority":43,"visibility":"hide"}]}`)
	slug, name, err := firstCodexModel(cat)
	if err != nil || slug != "gpt-6-astra" || name != "GPT-6-Astra" {
		t.Fatalf("%q %q %v", slug, name, err)
	}
	if _, _, err := firstCodexModel([]byte(`{"models":[{"slug":"x","priority":1,"visibility":"hide"}]}`)); err == nil {
		t.Fatal("a catalog with nothing listed gave a model")
	}
	if _, _, err := firstCodexModel([]byte(`not json`)); err == nil {
		t.Fatal("junk gave a model")
	}
}
