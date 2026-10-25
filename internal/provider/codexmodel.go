package provider

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"
)

// CodexModel is the model Codex takes for this account when none is
// given: the first one its own catalog lists (`codex debug models`, read
// on this machine; no model is called, no usage counted). The hub passes
// it with -m, so the version the phone shows is the one that answers.
// Kept for a few hours; "" and an error when the catalog can't be read
// (then Codex is run without -m, as before, and no version is shown).
func CodexModel(ctx context.Context, path string) (slug, name string, err error) {
	codexModel.mu.Lock()
	defer codexModel.mu.Unlock()
	if codexModel.slug != "" && time.Since(codexModel.at) < codexModelFor {
		return codexModel.slug, codexModel.name, nil
	}
	slug, name, err = readCodexModel(ctx, path)
	if err != nil {
		return "", "", err
	}
	codexModel.slug, codexModel.name, codexModel.at = slug, name, time.Now()
	return slug, name, nil
}

const codexModelFor = 6 * time.Hour

var codexModel struct {
	mu         sync.Mutex
	slug, name string
	at         time.Time
}

func readCodexModel(ctx context.Context, path string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "omnillm-codex-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(dir)
	home, err := prepareCodexHome()
	if err != nil {
		return "", "", err
	}
	cmd := exec.CommandContext(ctx, path, "-c", codexFileLogin, "debug", "models")
	cmd.Dir, cmd.Env = dir, append(cliEnv(path, dir), "CODEX_HOME="+home)
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	return firstCodexModel(out)
}

// firstCodexModel picks, from the catalog's JSON, the model listed first
// (the lowest priority among the ones shown in Codex's own picker).
func firstCodexModel(catalog []byte) (string, string, error) {
	var c struct {
		Models []struct {
			Slug        string `json:"slug"`
			DisplayName string `json:"display_name"`
			Priority    int    `json:"priority"`
			Visibility  string `json:"visibility"`
		} `json:"models"`
	}
	if err := json.Unmarshal(catalog, &c); err != nil {
		return "", "", err
	}
	best := -1
	for i, m := range c.Models {
		if m.Visibility != "list" || m.Slug == "" {
			continue
		}
		if best < 0 || m.Priority < c.Models[best].Priority {
			best = i
		}
	}
	if best < 0 {
		return "", "", errors.New("codex: no model in its catalog")
	}
	m := c.Models[best]
	if m.DisplayName == "" {
		m.DisplayName = PrettyModel(m.Slug)
	}
	return m.Slug, m.DisplayName, nil
}
