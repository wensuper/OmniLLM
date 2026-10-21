// Package control is the owner's local control channel to a running hub:
// a Unix socket inside a 0700 directory, mode 0600, so only the logged-in
// owner can use it (design §10: the console relies on the OS login, no
// extra password). Requests are one JSON line; replies are JSON lines.
package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Request is one command.
type Request struct {
	Op   string            `json:"op"`
	Args map[string]string `json:"args,omitempty"`
	// Secret carries a value that must not appear in argv (API keys).
	Secret string `json:"secret,omitempty"`
	// Lang is the caller's language ("zh" or "en"): the hub writes its
	// replies in it. Empty: the hub's own.
	Lang string `json:"lang,omitempty"`
}

// Reply is one line back. Event "confirm" asks the client a yes/no
// question; the client answers with a Request{Op:"answer"}.
type Reply struct {
	Event string          `json:"event"` // ok | error | info | confirm | done
	Data  json.RawMessage `json:"data,omitempty"`
	Text  string          `json:"text,omitempty"`
}

// Conn is one client connection, used by handlers. The context handed to
// a handler ends when the client goes away, so work it started (a pending
// pairing) does not outlive the command that asked for it.
type Conn struct {
	c       net.Conn
	enc     *json.Encoder
	askMu   sync.Mutex // one question at a time on a connection
	answers chan Request
}

func (c *Conn) Send(r Reply) error { return c.enc.Encode(r) }

// Ask sends a confirm event and waits for the answer.
func (c *Conn) Ask(ctx context.Context, text string) (bool, error) {
	c.askMu.Lock()
	defer c.askMu.Unlock()
	if err := c.Send(Reply{Event: "confirm", Text: text}); err != nil {
		return false, err
	}
	select {
	case q, ok := <-c.answers:
		if !ok {
			return false, errors.New("control: the client went away")
		}
		if q.Op != "answer" {
			return false, errors.New("control: expected an answer")
		}
		return q.Args["yes"] == "true", nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// readLoop reads the client's later lines (answers) and cancels the
// connection's context when the client disconnects.
func readLoop(r *bufio.Reader, out chan<- Request, cancel context.CancelFunc) {
	defer cancel()
	defer close(out)
	for {
		line, err := r.ReadSlice('\n')
		if err != nil {
			return
		}
		var q Request
		if json.Unmarshal(line, &q) != nil {
			return
		}
		select {
		case out <- q:
		default: // an answer nobody asked for
		}
	}
}

// Handler serves one request.
type Handler func(ctx context.Context, c *Conn, req Request) error

// Listen creates the socket at path, checking the directory is private.
func Listen(path string) (net.Listener, error) {
	// macOS caps a Unix socket path at 104 bytes, Linux at 108.
	if len(path) > 100 {
		return nil, &PathTooLongError{Path: path}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	st, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(dir, 0o700); err != nil {
			return nil, fmt.Errorf("control: %s must be private (0700)", dir)
		}
	}
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, err
	}
	return ln, nil
}

// Serve accepts connections until ctx ends.
func Serve(ctx context.Context, ln net.Listener, h Handler) {
	go func() { <-ctx.Done(); ln.Close() }()
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(30 * time.Second))
			r := bufio.NewReaderSize(io.LimitReader(c, 1<<20), 64<<10)
			line, err := r.ReadSlice('\n')
			if err != nil {
				return
			}
			_ = c.SetReadDeadline(time.Time{})
			var req Request
			if err := json.Unmarshal(line, &req); err != nil {
				return
			}
			cctx, cancel := context.WithCancel(ctx)
			defer cancel()
			conn := &Conn{c: c, enc: json.NewEncoder(c), answers: make(chan Request, 1)}
			go readLoop(r, conn.answers, cancel)
			if err := h(cctx, conn, req); err != nil {
				_ = conn.Send(Reply{Event: "error", Text: err.Error()})
			}
		}()
	}
}

// Call connects and sends req; onReply sees every reply and may answer
// confirm events by returning (answer, true).
func Call(path string, req Request, onReply func(Reply) (answer bool, answered bool)) error {
	c, err := net.Dial("unix", path)
	if err != nil {
		return fmt.Errorf("the hub is not running (no control socket at %s)", path)
	}
	defer c.Close()
	if err := json.NewEncoder(c).Encode(req); err != nil {
		return err
	}
	r := bufio.NewReader(c)
	for {
		line, err := r.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var rep Reply
		if err := json.Unmarshal(line, &rep); err != nil {
			return err
		}
		if rep.Event == "error" {
			return errors.New(strings.TrimSpace(rep.Text))
		}
		yes, answered := onReply(rep)
		if rep.Event == "confirm" && answered {
			ans := Request{Op: "answer", Args: map[string]string{"yes": fmt.Sprint(yes)}}
			if err := json.NewEncoder(c).Encode(ans); err != nil {
				return err
			}
		}
	}
}

// PathTooLongError means the data folder is too deep for a Unix socket.
type PathTooLongError struct{ Path string }

func (e *PathTooLongError) Error() string {
	return fmt.Sprintf("control: the data folder path is too long for the control socket (%d characters); set OMNILLM_HOME to a shorter folder, e.g. ~/.omnillm", len(e.Path))
}
