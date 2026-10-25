package provider

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
)

// CodexEntry is one model Codex offers this account.
type CodexEntry struct {
	Slug string // what -m takes ("gpt-6-astra")
	Name string // for people ("GPT-6-Astra")
	Rank int    // Codex's own order in its picker (lower first)
}

// CodexModels are the models Codex's own picker lists for this account, in
// its order (`codex debug models`, read on this machine; no model is
// called, no usage counted). The first is the one Codex takes when none is
// given. Kept for a few hours; an error when the catalog can't be read.
func CodexModels(ctx context.Context, path string) ([]CodexEntry, error) {
	codexCatalog.mu.Lock()
	defer codexCatalog.mu.Unlock()
	if len(codexCatalog.list) > 0 && time.Since(codexCatalog.at) < codexModelFor {
		return append([]CodexEntry(nil), codexCatalog.list...), nil
	}
	list, err := readCodexCatalog(ctx, path)
	if err != nil {
		return nil, err
	}
	codexCatalog.list, codexCatalog.at = list, time.Now()
	return append([]CodexEntry(nil), list...), nil
}

// CodexModel is the model Codex takes when none is given (the first one
// its picker lists). The hub runs Codex with it named (-m), so the version
// shown is the one that answers.
func CodexModel(ctx context.Context, path string) (slug, name string, err error) {
	list, err := CodexModels(ctx, path)
	if err != nil {
		return "", "", err
	}
	return list[0].Slug, list[0].Name, nil
}

// CodexExtra are the catalog's models besides the first: "default" stands
// for the first (it keeps the id devices and chats already use), the rest
// are offered by their own names. None when the catalog can't be read.
func CodexExtra(ctx context.Context, path string) []string {
	list, err := CodexModels(ctx, path)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range list[1:] {
		out = append(out, e.Slug)
	}
	return out
}

// CodexName names a Codex model for people from the last catalog read
// ("gpt-6-sol" → "GPT-6-Sol"), and its place in Codex's picker (0 when
// not known).
func CodexName(slug string) (string, int) {
	codexCatalog.mu.Lock()
	defer codexCatalog.mu.Unlock()
	for _, e := range codexCatalog.list {
		if e.Slug == slug {
			return e.Name, e.Rank
		}
	}
	return PrettyModel(slug), 0
}

// CodexRankOf is the picker place of the model with this name in the last
// catalog read (0 when not known); it never runs Codex.
func CodexRankOf(name string) int {
	codexCatalog.mu.Lock()
	defer codexCatalog.mu.Unlock()
	for _, e := range codexCatalog.list {
		if e.Name == name {
			return e.Rank
		}
	}
	return 0
}

const codexModelFor = 6 * time.Hour

var codexCatalog struct {
	mu   sync.Mutex
	list []CodexEntry
	at   time.Time
}

func readCodexCatalog(ctx context.Context, path string) ([]CodexEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "omnillm-codex-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	home, err := prepareCodexHome()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, "-c", codexFileLogin, "debug", "models")
	cmd.Dir, cmd.Env = dir, append(cliEnv(path, dir), "CODEX_HOME="+home)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return codexCatalogOf(out)
}

// codexCatalogOf reads the catalog's JSON: the models shown in Codex's own
// picker (visibility "list"), lowest priority first.
func codexCatalogOf(catalog []byte) ([]CodexEntry, error) {
	var c struct {
		Models []struct {
			Slug        string `json:"slug"`
			DisplayName string `json:"display_name"`
			Priority    int    `json:"priority"`
			Visibility  string `json:"visibility"`
		} `json:"models"`
	}
	if err := json.Unmarshal(catalog, &c); err != nil {
		return nil, err
	}
	var out []CodexEntry
	for _, m := range c.Models {
		if m.Visibility != "list" || m.Slug == "" {
			continue
		}
		name := m.DisplayName
		if name == "" {
			name = PrettyModel(m.Slug)
		}
		out = append(out, CodexEntry{Slug: m.Slug, Name: name, Rank: m.Priority})
	}
	if len(out) == 0 {
		return nil, errors.New("codex: no model in its catalog")
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rank < out[j].Rank })
	return out, nil
}

// firstCodexModel is the catalog's first model (kept for its test).
func firstCodexModel(catalog []byte) (string, string, error) {
	list, err := codexCatalogOf(catalog)
	if err != nil {
		return "", "", err
	}
	return list[0].Slug, list[0].Name, nil
}
