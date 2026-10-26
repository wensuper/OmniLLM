package provider

// Chats lets a conversation with a subscription program continue the
// program's own saved session (Codex's thread, Claude Code's session)
// instead of sending the whole transcript again every turn (design §13.4).
//
// Nothing changes on the wire: a device still sends the whole conversation.
// The hub recognises a follow-up by a hash of everything before the new
// message (every earlier turn, the last answer), who asked, the program,
// the model and the system text. If anything differs (an earlier message
// edited, an answer regenerated, another model) the turn starts a new
// session from the transcript, as before. So a match is never a guess
// about another conversation.
//
// The sessions hold the conversation on the hub's disk, so they are kept
// only while used: removed after a day without use, when the device is
// removed, and when there are too many. The table here holds only hashes
// and the programs' session ids, never text.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

// Limits on kept sessions.
const (
	ChatIdle      = 24 * time.Hour // removed after this long without use
	chatsPerOwner = 20
	chatsMax      = 100
	// pendingAge: a session recorded when its turn started and never
	// finished (the hub stopped meanwhile) is removed once it is older than
	// any call can run (10 minutes).
	pendingAge = 15 * time.Minute
)

// ChatRecord is one kept session. It holds no conversation text.
type ChatRecord struct {
	Key   string    `json:"key"`   // the hash a follow-up will carry (chatKey), or pendingKey
	Kind  Kind      `json:"kind"`  // which program
	ID    string    `json:"id"`    // the program's session id
	Owner string    `json:"owner"` // "device/<id>" or "host"
	Path  string    `json:"path"`  // the program, to delete Codex sessions
	Used  time.Time `json:"used"`
}

// Chats is the hub's table of kept sessions.
type Chats struct {
	mu   sync.Mutex
	recs map[string]ChatRecord // by Key
	now  func() time.Time
	// saveMu keeps saves in order: each writes the table as it is then.
	saveMu sync.Mutex
	save   func([]ChatRecord)
	// remove deletes a session's files; tests replace it.
	remove func(ChatRecord)
	// forgotten counts Forget / ForgetKind calls, so a turn that was
	// running meanwhile does not keep its session afterwards.
	ownerGen map[string]int
	kindGen  map[Kind]int
}

// NewChats makes an empty table. save is called with the whole table
// after every change, one call at a time; now may be nil.
func NewChats(now func() time.Time, save func([]ChatRecord)) *Chats {
	if now == nil {
		now = time.Now
	}
	if save == nil {
		save = func([]ChatRecord) {}
	}
	return &Chats{recs: map[string]ChatRecord{}, now: now, save: save, remove: removeChat,
		ownerGen: map[string]int{}, kindGen: map[Kind]int{}}
}

// Load puts back a saved table (at start). Expired records are dropped
// by the next Sweep.
func (c *Chats) Load(recs []ChatRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range recs {
		if r.Key != "" && validChatID(r.ID) && r.Kind.Subscription() {
			c.recs[r.Key] = r
		}
	}
}

// persist saves the table as it is now.
func (c *Chats) persist() {
	c.saveMu.Lock()
	defer c.saveMu.Unlock()
	c.mu.Lock()
	recs := c.snapshotLocked()
	c.mu.Unlock()
	c.save(recs)
}

func (c *Chats) snapshotLocked() []ChatRecord {
	out := make([]ChatRecord, 0, len(c.recs))
	for _, r := range c.recs {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Used.Before(out[j].Used) })
	return out
}

// take hands out the session a follow-up continues, removing it from the
// table: a turn holds its session alone, and a turn that fails leaves no
// half-written session behind to be continued.
func (c *Chats) take(key string) (ChatRecord, bool) {
	c.mu.Lock()
	r, ok := c.recs[key]
	if ok {
		delete(c.recs, key)
	}
	c.mu.Unlock()
	if !ok {
		return r, false
	}
	c.persist()
	if c.now().Sub(r.Used) > ChatIdle || !chatExists(r) {
		go c.remove(r)
		return r, false
	}
	return r, true
}

// gens is what a turn remembers at its start (see put).
func (c *Chats) gens(owner string, k Kind) (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ownerGen[owner], c.kindGen[k]
}

// put keeps a session after a turn that ended well, and removes the
// oldest ones over the limits. A turn whose device or program was removed
// while it ran (og, kg out of date) keeps nothing.
func (c *Chats) put(r ChatRecord, og, kg int) {
	r.Used = c.now()
	c.mu.Lock()
	delete(c.recs, pendingKey(r.ID))
	if c.ownerGen[r.Owner] != og || c.kindGen[r.Kind] != kg {
		c.mu.Unlock()
		c.persist()
		go c.remove(r)
		return
	}
	var drop []ChatRecord
	if old, ok := c.recs[r.Key]; ok && old.ID != r.ID {
		// The same point reached twice (a question asked again from the
		// start): the newer session wins, the older one goes.
		drop = append(drop, old)
	}
	c.recs[r.Key] = r
	for {
		x, ok := c.overLocked(r)
		if !ok {
			break
		}
		drop = append(drop, x)
		delete(c.recs, x.Key)
	}
	c.mu.Unlock()
	c.persist()
	for _, x := range drop {
		go c.remove(x)
	}
}

// overLocked picks the oldest session to drop while the owner or the
// table is over its limit; never the one just kept.
func (c *Chats) overLocked(keep ChatRecord) (ChatRecord, bool) {
	var recs []ChatRecord // kept ones, oldest first; pending ones do not count
	for _, x := range c.snapshotLocked() {
		if !isPending(x) {
			recs = append(recs, x)
		}
	}
	mine := 0
	for _, x := range recs {
		if x.Owner == keep.Owner {
			mine++
		}
	}
	for _, x := range recs {
		if x.Key == keep.Key {
			continue
		}
		if mine > chatsPerOwner && x.Owner == keep.Owner || mine <= chatsPerOwner && len(recs) > chatsMax {
			return x, true
		}
	}
	return ChatRecord{}, false
}

// pending records a session as soon as its id is known, before the turn
// ends, so one the hub never finished (it stopped meanwhile) is still
// found and removed by Sweep. Nothing outside this table is ever deleted.
func (c *Chats) pending(r ChatRecord) {
	if !validChatID(r.ID) {
		return
	}
	r.Key, r.Used = pendingKey(r.ID), c.now()
	c.mu.Lock()
	c.recs[r.Key] = r
	c.mu.Unlock()
	c.persist()
}

func pendingKey(id string) string { return "pending:" + id }

func isPending(r ChatRecord) bool { return strings.HasPrefix(r.Key, "pending:") }

// discard removes a session that will not be continued.
func (c *Chats) discard(r ChatRecord) {
	if !validChatID(r.ID) {
		return
	}
	c.mu.Lock()
	_, was := c.recs[pendingKey(r.ID)]
	delete(c.recs, pendingKey(r.ID))
	c.mu.Unlock()
	if was {
		c.persist()
	}
	go c.remove(r)
}

// Forget removes every session of one owner (a removed device).
func (c *Chats) Forget(owner string) {
	c.mu.Lock()
	c.ownerGen[owner]++
	c.mu.Unlock()
	c.dropWhere(func(r ChatRecord) bool { return r.Owner == owner })
}

// ForgetKind removes every session of one program (its source removed).
func (c *Chats) ForgetKind(k Kind) {
	c.mu.Lock()
	c.kindGen[k]++
	c.mu.Unlock()
	c.dropWhere(func(r ChatRecord) bool { return r.Kind == k })
}

func (c *Chats) dropWhere(match func(ChatRecord) bool) {
	c.mu.Lock()
	var drop []ChatRecord
	for k, r := range c.recs {
		if match(r) {
			drop = append(drop, r)
			delete(c.recs, k)
		}
	}
	c.mu.Unlock()
	if len(drop) == 0 {
		return
	}
	c.persist()
	// In the background: Codex deletes each session itself, and removing a
	// device must not wait for that.
	go func() {
		for _, r := range drop {
			c.remove(r)
		}
	}()
}

// Sweep removes sessions unused for ChatIdle and turns that never
// finished. The hub runs it at start and every hour.
func (c *Chats) Sweep() {
	now := c.now()
	c.dropWhere(func(r ChatRecord) bool {
		if isPending(r) {
			return now.Sub(r.Used) > pendingAge
		}
		return now.Sub(r.Used) > ChatIdle
	})
}

// RemoveChats deletes the given sessions' files now and waits for it, and
// the hub's empty Claude Code folder (uninstall: the hub is stopped).
func RemoveChats(recs []ChatRecord) {
	for _, r := range recs {
		if validChatID(r.ID) && r.Kind.Subscription() {
			removeChat(r)
		}
	}
	_ = os.RemoveAll(claudeChatDir())
}

// Len is how many sessions are kept.
func (c *Chats) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.recs)
}

// chatTurn is what a turn needs to keep its session afterwards.
type chatTurn struct {
	chats   *Chats
	owner   string
	kind    Kind
	alias   string
	system  string
	turns   []omnillm.Message // the request's turns, ending with the new message
	id      string            // the program's session id (Codex: from its first line)
	path    string
	resumed bool
	og, kg  int // Chats.gens at the start
}

// keep stores the session under the key the next turn will carry.
func (t *chatTurn) keep(reply string) {
	if !validChatID(t.id) {
		return
	}
	history := append(append([]omnillm.Message{}, t.turns...), omnillm.Message{Role: "assistant", Content: reply})
	r := ChatRecord{Kind: t.kind, ID: t.id, Owner: t.owner, Path: t.path,
		Key: chatKey(t.owner, t.kind, t.alias, t.system, history)}
	if !chatExists(r) {
		t.discard() // not where expected: the next turn starts afresh
		return
	}
	t.chats.put(r, t.og, t.kg)
}

func (t *chatTurn) discard() {
	t.chats.discard(ChatRecord{Kind: t.kind, ID: t.id, Owner: t.owner, Path: t.path})
}

// splitChat separates the system text from the turns.
func splitChat(msgs []omnillm.Message) (system string, turns []omnillm.Message) {
	var sys []string
	for _, m := range msgs {
		if m.Role == "system" {
			sys = append(sys, m.Content)
		} else {
			turns = append(turns, m)
		}
	}
	return strings.Join(sys, "\n\n"), turns
}

// followUpKey is the key a request carries if it continues a kept
// session: everything before its new message, ending with an answer.
func followUpKey(owner string, k Kind, alias, system string, turns []omnillm.Message) (string, bool) {
	n := len(turns)
	if n < 3 || turns[n-1].Role != "user" || turns[n-2].Role != "assistant" || turns[n-3].Role != "user" {
		return "", false
	}
	return chatKey(owner, k, alias, system, turns[:n-1]), true
}

// imageNote is how the web client names the images of older turns it no
// longer sends ("[2 earlier image(s)]"); hashed as if they were absent.
var imageNote = regexp.MustCompile(`\n?\[\d+ earlier image\(s\)\]$`)

// chatKey hashes a point in a conversation: who, which program and model,
// the system text, and every turn up to the last answer. The user message
// just before that answer counts with its images (the device sends them
// with the next turn too); older ones count by their text only, because
// the web client stops sending their images.
func chatKey(owner string, k Kind, alias, system string, history []omnillm.Message) string {
	h := sha256.New()
	field := func(h hash.Hash, s string) {
		_ = binary.Write(h, binary.BigEndian, uint64(len(s)))
		h.Write([]byte(s))
	}
	field(h, "omnillm-chat-2")
	field(h, owner)
	field(h, string(k))
	field(h, alias)
	field(h, system)
	_ = binary.Write(h, binary.BigEndian, uint64(len(history)))
	for i, m := range history {
		field(h, m.Role)
		if i == len(history)-2 {
			field(h, m.Content)
			for _, im := range m.Images {
				sum := sha256.Sum256(im.Data)
				field(h, im.MIME)
				field(h, string(sum[:]))
			}
			continue
		}
		field(h, imageNote.ReplaceAllString(m.Content, ""))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// newChatID is a random UUID (Claude Code takes the session id it is given).
func newChatID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// validChatID accepts only a UUID: the id becomes a file name and an
// argument.
func validChatID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
				return false
			}
		}
	}
	return true
}

// claudeChatDir is the folder Claude Code runs in for conversations. It
// stays the same, because Claude Code finds a session by its folder, and
// it is empty and the hub's own, so these sessions sit apart from the
// owner's projects.
func claudeChatDir() string {
	return filepath.Join(filepath.Dir(codexHubHome()), "claude-chats")
}

// claudeSessionFiles finds a Claude Code session by its id, which is a
// random UUID: ~/.claude/projects/<the folder it ran in>/<id>.jsonl. Found,
// not computed, so a change in how Claude names that folder does not lose
// sessions.
func claudeSessionFiles(id string) []string {
	home, _ := os.UserHomeDir()
	m, _ := filepath.Glob(filepath.Join(home, ".claude", "projects", "*", id+".jsonl"))
	return m
}

func prepareClaudeChatDir() (string, error) {
	d := claudeChatDir()
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", err
	}
	_ = os.Chmod(d, 0o700)
	return d, nil
}

// codexSessionFiles finds a Codex session's file(s) in the hub's home.
func codexSessionFiles(id string) []string {
	m, _ := filepath.Glob(filepath.Join(codexHubHome(), "sessions", "*", "*", "*", "rollout-*-"+id+".jsonl"))
	return m
}

// chatExists says whether the session's file is still there.
func chatExists(r ChatRecord) bool {
	if !validChatID(r.ID) {
		return false
	}
	switch r.Kind {
	case ClaudeCode:
		return len(claudeSessionFiles(r.ID)) > 0
	case Antigravity:
		return len(agyConversationFiles(r.ID)) > 0
	}
	return len(codexSessionFiles(r.ID)) > 0
}

// removeChat deletes a session's files. Codex also keeps the session in
// its own database, so Codex deletes it; its files go even if that fails.
func removeChat(r ChatRecord) {
	if !validChatID(r.ID) {
		return
	}
	if r.Kind == Antigravity {
		removeAgyConversation(r.ID)
		return
	}
	if r.Kind == ClaudeCode {
		for _, f := range claudeSessionFiles(r.ID) {
			_ = os.Remove(f)
			_ = os.RemoveAll(strings.TrimSuffix(f, ".jsonl"))
		}
		return
	}
	path := r.Path
	if path == "" {
		path = FindCLI(Codex)
	}
	if base := filepath.Base(path); path != "" && (base == "codex" || strings.HasPrefix(base, "codex-")) && TrustedCLI(path) {
		if home, err := prepareCodexHome(); err == nil {
			if dir, err := os.MkdirTemp("", "omnillm-cli-"); err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				cmd := exec.CommandContext(ctx, path, "-c", codexFileLogin, "delete", "--force", r.ID)
				cmd.Dir, cmd.Env = dir, append(cliEnv(path, dir), "CODEX_HOME="+home)
				_ = cmd.Run()
				cancel()
				os.RemoveAll(dir)
			}
		}
	}
	for _, f := range codexSessionFiles(r.ID) {
		_ = os.Remove(f)
	}
}
