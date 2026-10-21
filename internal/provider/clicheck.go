package provider

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/egress"
)

// cliCheck remembers which version of a subscription program passed the
// no-tools check. The flags that turn tools off are only as good as the
// program that reads them, so every new version is checked before a
// device's text reaches it: asked to run a command and read a file, it must
// do neither. A version that fails is not used.
type cliCheck struct {
	mu      sync.Mutex
	passed  string    // version that passed
	failed  string    // version that failed
	checked time.Time // when the version was last read
}

// checkEvery is how often the program's version is read again.
const checkEvery = 10 * time.Minute

// Prepare runs the check now (the hub calls it when a subscription
// connects and at start), so a person's first message does not wait for it.
func (c *Client) Prepare(ctx context.Context) error {
	if !c.cfg.Kind.Subscription() {
		return nil
	}
	return c.checkCLI(ctx)
}

func (c *Client) checkCLI(ctx context.Context) error {
	c.cli.mu.Lock()
	defer c.cli.mu.Unlock()
	if c.cli.passed != "" && time.Since(c.cli.checked) < checkEvery {
		return nil
	}
	ver := cliVersion(ctx, c.cfg.Path)
	c.cli.checked = time.Now()
	switch {
	case ver != "" && ver == c.cli.passed:
		return nil
	case ver != "" && ver == c.cli.failed:
		return c.unsafe()
	}
	if err := c.canary(ctx); err != nil {
		if e, ok := err.(*omnillm.Error); ok && e.Msg == omnillm.MsgCLIUnsafe {
			c.cli.failed = ver
		}
		return err
	}
	c.cli.passed = ver
	if ver == "" {
		c.cli.passed = "?"
	}
	return nil
}

func (c *Client) unsafe() error {
	return &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgCLIUnsafe, Args: []string{string(c.cfg.Kind)}}
}

func cliVersion(ctx context.Context, path string) string {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "omnillm-cli-")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(dir)
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Dir, cmd.Env = dir, cliEnv(path, dir)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// canary asks the program to run a command and read a file outside its
// folder. Either one happening means its tools are not off.
func (c *Client) canary(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "omnillm-canary-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	secret := "CANARY-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	file, marker := filepath.Join(dir, "canary.txt"), filepath.Join(dir, "ran")
	if err := os.WriteFile(file, []byte(secret), 0o600); err != nil {
		return err
	}
	model := cliModels[c.cfg.Kind][len(cliModels[c.cfg.Kind])-1] // the cheapest
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	s, err := c.spawnCLI(ctx, model, []omnillm.Message{{Role: "user", Content: "Use your tools. Run the shell command: touch " + marker +
		" . Then read the file " + file + " and reply with its exact contents. If you have no tools, reply exactly: NO TOOLS"}},
		func(egress.Outcome, time.Duration) {})
	if err != nil {
		return err
	}
	var b strings.Builder
	for {
		ch, err := s.Recv()
		if err != nil {
			s.Close()
			if !errors.Is(err, io.EOF) && b.Len() == 0 {
				return err // not logged in, limit reached: say that instead
			}
			break
		}
		b.WriteString(ch.Text)
	}
	if strings.Contains(b.String(), secret) {
		return c.unsafe()
	}
	if _, err := os.Stat(marker); err == nil {
		return c.unsafe()
	}
	return nil
}
