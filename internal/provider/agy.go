package provider

// Antigravity: Google's official Antigravity CLI (agy), signed in with the
// owner's Google account, gives a Google AI subscription (Gemini, and the
// other models Antigravity offers) to the owner's phones like Claude Code
// and Codex do (cli.go). Google replaced the Gemini CLI with it for
// personal accounts in June 2026.
//
// agy reads everything from its home: settings, rules (~/.gemini/GEMINI.md,
// AGENTS.md), skills, plugins, hooks and MCP servers, and it has many
// tools (shell, files, browser, web search, schedules, subagents) that no
// flag turns off. So the hub runs it with a home of its own, where the
// owner signs in once more (omnillm login gemini) and where nothing of the
// owner's is: no rules, no MCP servers, and a hook that refuses every tool
// call before it runs, plus deny rules for the tools agy names. The no-tools
// canary (clicheck.go) checks each version, as for the other programs.
//
// A message goes in on stdin as agy's stream-json input (never in argv,
// where other local users could read it); the reply comes back as its
// stream-json events. A conversation is kept by agy's conversation id
// (--conversation); a call outside one is deleted when it ends.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wensuper/OmniLLM/internal/fsown"
)

const Antigravity Kind = "antigravity"

// AgyHome is the hub's own home for agy: where the owner signs in for the
// hub (HOME=<it> agy) and where agy keeps that login and its conversations.
func AgyHome() string {
	if d := os.Getenv("OMNILLM_HOME"); d != "" {
		return filepath.Join(d, "agy-home")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "OmniLLM", "agy-home")
}

// agyDenyAll is the hook that refuses every tool call, whatever its name.
const agyDenyAll = `{
  "omnillm-no-tools": {
    "PreToolUse": [
      {"matcher": "*", "hooks": [{"type": "command", "command": "echo '{\"decision\":\"deny\",\"reason\":\"Chat only: no tools.\"}'", "timeout": 5}]}
    ]
  }
}
`

// agyDeny are agy's own deny rules, a second fence behind the hook.
var agyDeny = []string{"command(*)", "read_file(*)", "write_file(*)", "read_url(*)", "execute_url(*)", "mcp(*)"}

// agyMu keeps two calls from rewriting the home's files at once.
var agyMu sync.Mutex

// prepareAgyHome makes the hub's agy home safe to run in: private and
// ours, the deny-all hook and deny rules in place, and no rules, skills,
// plugins or MCP servers. The login (agy's own files there) is kept.
func prepareAgyHome() (string, error) {
	agyMu.Lock()
	defer agyMu.Unlock()
	home := AgyHome()
	if err := os.MkdirAll(home, 0o700); err != nil {
		return "", err
	}
	if fi, err := os.Lstat(home); err != nil || !fi.IsDir() || !fsown.Mine(fi) {
		return "", errors.New("antigravity: its folder isn't this user's own")
	}
	if err := os.Chmod(home, 0o700); err != nil {
		return "", err
	}
	g := filepath.Join(home, ".gemini")
	cli, cfg := filepath.Join(g, "antigravity-cli"), filepath.Join(g, "config")
	for _, d := range []string{g, cli, cfg} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return "", err
		}
		if fi, err := os.Lstat(d); err != nil || !fi.IsDir() {
			return "", errors.New("antigravity: " + d + " isn't a folder")
		}
	}
	// Anything that would give agy the owner's words or more abilities.
	for _, p := range []string{
		filepath.Join(home, "AGENTS.md"), filepath.Join(home, "GEMINI.md"), filepath.Join(home, ".agents"), filepath.Join(home, ".agent"),
		filepath.Join(g, "AGENTS.md"), filepath.Join(g, "GEMINI.md"),
		filepath.Join(cfg, "AGENTS.md"), filepath.Join(cfg, "GEMINI.md"), filepath.Join(cfg, "rules"), filepath.Join(cfg, "skills"),
		filepath.Join(cfg, "plugins"), filepath.Join(cfg, "workflows"),
		filepath.Join(cli, "rules"), filepath.Join(cli, "skills"), filepath.Join(cli, "plugins"), filepath.Join(cli, "mcp"),
		filepath.Join(cli, "hooks.json"), filepath.Join(cli, "mcp_config.json"),
	} {
		_ = os.RemoveAll(p)
	}
	if err := writePrivate(filepath.Join(cfg, "hooks.json"), []byte(agyDenyAll)); err != nil {
		return "", err
	}
	if err := writePrivate(filepath.Join(cfg, "mcp_config.json"), []byte("{\"mcpServers\": {}}\n")); err != nil {
		return "", err
	}
	// Settings: agy rewrites the file itself (trusted folders, its own
	// defaults), so ours are set again each time and its others kept.
	sp := filepath.Join(cli, "settings.json")
	set := map[string]any{}
	if b, err := os.ReadFile(sp); err == nil {
		_ = json.Unmarshal(b, &set)
	}
	set["toolPermission"] = "strict"
	set["allowNonWorkspaceAccess"] = false
	set["enableTerminalSandbox"] = true
	set["disableSlashCommands"] = true
	set["showFeedbackSurvey"] = false
	set["permissions"] = map[string]any{"deny": agyDeny}
	b, _ := json.MarshalIndent(set, "", "  ")
	if err := writePrivate(sp, append(b, '\n')); err != nil {
		return "", err
	}
	return home, nil
}

// writePrivate replaces p with data, 0600, never through a link.
func writePrivate(p string, data []byte) error {
	_ = os.Remove(p)
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// agyEnv is cliEnv with the hub's agy home.
func agyEnv(path, scratch, home string) []string {
	return append(cliEnv(path, scratch), "HOME="+home, "AGY_CLI_DISABLE_AUTO_UPDATE=1", "AGY_CLI_HIDE_ACCOUNT_INFO=1")
}

// AgyEntry is one model Antigravity offers this account.
type AgyEntry struct {
	Slug string // what --model takes ("gemini-3.8-flash-high")
	Name string // for people ("Gemini 3.8 Flash (High)")
	Rank int    // its place in agy's list, from 1
}

// AgyModels are the models `agy models` lists for the signed-in account,
// in its order (no model is called). The first is the one used when none
// is chosen. Kept for a few hours; an error when signed out.
func AgyModels(ctx context.Context, path string) ([]AgyEntry, error) {
	agyCatalog.mu.Lock()
	defer agyCatalog.mu.Unlock()
	if len(agyCatalog.list) > 0 && time.Since(agyCatalog.at) < codexModelFor {
		return append([]AgyEntry(nil), agyCatalog.list...), nil
	}
	list, err := readAgyModels(ctx, path)
	if err != nil {
		return nil, err
	}
	agyCatalog.list, agyCatalog.at = list, time.Now()
	return append([]AgyEntry(nil), list...), nil
}

var agyCatalog struct {
	mu   sync.Mutex
	list []AgyEntry
	at   time.Time
}

// AgyName names an Antigravity model for people from the last list read,
// and its place in it (0 when not known).
func AgyName(slug string) (string, int) {
	agyCatalog.mu.Lock()
	defer agyCatalog.mu.Unlock()
	for _, e := range agyCatalog.list {
		if e.Slug == slug {
			return e.Name, e.Rank
		}
	}
	return PrettyModel(slug), 0
}

// AgyDisplay names a model id of an Antigravity source for people:
// "default" is the first model listed. It never runs agy ("" before the
// first list was read).
func AgyDisplay(upstream string) (string, int) {
	if upstream == "default" || upstream == "" {
		agyCatalog.mu.Lock()
		defer agyCatalog.mu.Unlock()
		if len(agyCatalog.list) == 0 {
			return "", 0
		}
		return agyCatalog.list[0].Name, 1
	}
	return AgyName(upstream)
}

// readAgyModels runs `agy models` in the hub's home. It also says whether
// agy is signed in there: signed out, it lists nothing.
func readAgyModels(ctx context.Context, path string) ([]AgyEntry, error) {
	if !TrustedCLI(path) {
		return nil, errUntrustedCLI
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	home, err := prepareAgyHome()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "omnillm-cli-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	cmd := exec.CommandContext(ctx, path, "models")
	cmd.Dir, cmd.Env = dir, agyEnv(path, dir, home)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return agyModelsOf(out)
}

// agyModelsOf reads `agy models`: one "slug<TAB>name" line per model.
func agyModelsOf(out []byte) ([]AgyEntry, error) {
	var list []AgyEntry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.SplitN(strings.TrimSpace(sc.Text()), "\t", 2)
		if len(f) != 2 {
			continue
		}
		slug, name := strings.TrimSpace(f[0]), strings.TrimSpace(f[1])
		if slug == "" || len(slug) > 100 || strings.ContainsAny(slug, " /\\") {
			continue
		}
		if name == "" {
			name = PrettyModel(slug)
		}
		list = append(list, AgyEntry{Slug: slug, Name: name, Rank: len(list) + 1})
	}
	if len(list) == 0 {
		return nil, errors.New("antigravity: no model listed (signed out?)")
	}
	return list, nil
}

// agyExtra are the listed models besides the first ("default" stands for
// the first). None when signed out.
func agyExtra(ctx context.Context, path string) []string {
	list, err := AgyModels(ctx, path)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range list[1:] {
		out = append(out, e.Slug)
	}
	return out
}

// agyInput is the one stream-json message agy reads on stdin.
func agyInput(prompt string) string {
	b, _ := json.Marshal(map[string]any{"event": "user", "message": map[string]any{"role": "user", "content": prompt}})
	return string(b) + "\n"
}

// agyArgs are the flags for one reply. id: the conversation to go on with.
func agyArgs(model, id string) []string {
	// --print= with no prompt: the message comes on stdin.
	args := []string{"--print=", "--input-format", "stream-json", "--output-format", "stream-json", "--disable-slash-commands"}
	if model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	if id != "" {
		args = append(args, "--conversation", id)
	}
	return args
}

// agyConversationFiles are where agy keeps a conversation in the hub's home.
func agyConversationFiles(id string) []string {
	c := filepath.Join(AgyHome(), ".gemini", "antigravity-cli")
	var out []string
	for _, p := range []string{filepath.Join(c, "conversations", id+".db"), filepath.Join(c, "conversations", id+".db-wal"),
		filepath.Join(c, "conversations", id+".db-shm"), filepath.Join(c, "brain", id), filepath.Join(c, "annotations", id),
		filepath.Join(c, "implicit", id)} {
		if _, err := os.Lstat(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func removeAgyConversation(id string) {
	if !validChatID(id) {
		return
	}
	for _, p := range agyConversationFiles(id) {
		_ = os.RemoveAll(p)
	}
}

// agyLine reads one of agy's stream-json events.
func (s *cliStream) agyLine(b []byte) (text string, final bool) {
	var ev struct {
		Event          string `json:"event"`
		ConversationID string `json:"conversation_id"`
		Init           struct {
			Model string `json:"model"`
		} `json:"init"`
		Step struct {
			StepType  string `json:"step_type"`
			State     string `json:"state"`
			TextDelta string `json:"text_delta"`
		} `json:"step_update"`
		Result struct {
			ConversationID string `json:"conversation_id"`
			Status         string `json:"status"`
			Response       string `json:"response"`
			Error          string `json:"error"`
			Usage          struct {
				InputTokens    int `json:"input_tokens"`
				OutputTokens   int `json:"output_tokens"`
				ThinkingTokens int `json:"thinking_tokens"`
				CacheRead      int `json:"cache_read_tokens"`
			} `json:"usage"`
		} `json:"result"`
	}
	if json.Unmarshal(b, &ev) != nil {
		return "", false
	}
	switch ev.Event {
	case "init":
		s.agyConversation(ev.ConversationID)
		if ev.Init.Model != "" {
			name, _ := AgyName(ev.Init.Model)
			s.mu.Lock()
			if s.model == "" {
				s.model = name
			}
			s.mu.Unlock()
		}
	case "step_update":
		if ev.Step.StepType == "agent_response" && ev.Step.TextDelta != "" {
			s.mu.Lock()
			s.gotAny = true
			s.mu.Unlock()
			return ev.Step.TextDelta, false
		}
	case "result":
		s.agyConversation(ev.Result.ConversationID)
		u := ev.Result.Usage
		s.setUsage(u.InputTokens+u.CacheRead, u.OutputTokens+u.ThinkingTokens)
		if ev.Result.Status != "SUCCESS" {
			s.fail(ev.Result.Error)
			return "", true
		}
		s.setFinish("stop")
		s.mu.Lock()
		got := s.gotAny
		s.gotAny = s.gotAny || ev.Result.Response != ""
		s.mu.Unlock()
		if !got && ev.Result.Response != "" {
			return ev.Result.Response, true
		}
		return "", true
	}
	return "", false
}

// agyConversation notes the conversation agy uses: kept with a person's
// chat, otherwise deleted when the call ends.
func (s *cliStream) agyConversation(id string) {
	if !validChatID(id) {
		return
	}
	s.mu.Lock()
	if s.agyID == "" {
		s.agyID = id
	}
	t := s.chat
	fresh := t != nil && !t.resumed && t.id == ""
	if fresh {
		t.id = id
	}
	s.mu.Unlock()
	if fresh {
		t.chats.pending(ChatRecord{Kind: t.kind, ID: id, Owner: t.owner, Path: t.path})
	}
}
