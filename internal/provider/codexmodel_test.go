package provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/wensuper/OmniLLM"
)

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

func TestCodexRunsTheCatalogsFirstModelByName(t *testing.T) {
	chatHome(t)
	c, rec := fakeCodexChat(t)
	// The same stand-in, now with a catalog.
	b, _ := os.ReadFile(c.cfg.Path)
	s := strings.Replace(string(b), `case "$*" in *"login status"*)`, `case "$*" in *"debug models"*) echo '{"models":[{"slug":"gpt-6-sol","display_name":"GPT-6-Sol","priority":4,"visibility":"list"},{"slug":"gpt-6-astra","display_name":"GPT-6-Astra","priority":2,"visibility":"list"}]}'; exit 0;; esac
case "$*" in *"login status"*)`, 1)
	if err := os.WriteFile(c.cfg.Path, []byte(s), 0o700); err != nil {
		t.Fatal(err)
	}
	codexModel.mu.Lock()
	codexModel.slug = ""
	codexModel.mu.Unlock()
	st, err := c.Stream(context.Background(), "default", omnillm.Request{Model: "codex/default", Caller: omnillm.Interactive, MaxTokens: 16,
		Messages: []omnillm.Message{{Role: "user", Content: "ping"}}}, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	model := ""
	for {
		ch, err := st.Recv()
		if ch.Model != "" && model == "" {
			model = ch.Model
		}
		if err != nil {
			break
		}
	}
	if model != "GPT-6-Astra" {
		t.Fatalf("the reply named %q", model)
	}
	got, _ := os.ReadFile(rec)
	if !strings.Contains(string(got), "-m gpt-6-astra") {
		t.Fatalf("not run with -m gpt-6-astra:\n%s", got)
	}
}
