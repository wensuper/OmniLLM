package control

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func shortDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "omc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

func TestSocketIsPrivateAndConfirmFlows(t *testing.T) {
	dir := filepath.Join(shortDir(t), "hub")
	sock := filepath.Join(dir, "c.sock")
	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(sock)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode %v", st.Mode().Perm())
	}
	dst, _ := os.Stat(dir)
	if dst.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %v", dst.Mode().Perm())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Serve(ctx, ln, func(ctx context.Context, c *Conn, req Request) error {
		if req.Op == "fail" {
			return errors.New("nope")
		}
		ok, err := c.Ask(ctx, "sure?")
		if err != nil {
			return err
		}
		if ok {
			return c.Send(Reply{Event: "done", Text: "yes " + req.Secret})
		}
		return c.Send(Reply{Event: "done", Text: "no"})
	})
	var got []string
	err = Call(sock, Request{Op: "x", Secret: "s"}, func(r Reply) (bool, bool) {
		got = append(got, r.Event+":"+r.Text)
		return true, r.Event == "confirm"
	})
	if err != nil || strings.Join(got, ",") != "confirm:sure?,done:yes s" {
		t.Fatal(got, err)
	}
	if err := Call(sock, Request{Op: "fail"}, func(Reply) (bool, bool) { return false, false }); err == nil || err.Error() != "nope" {
		t.Fatal(err)
	}
}

func TestLoosePermissionsAreTightened(t *testing.T) {
	dir := filepath.Join(shortDir(t), "hub")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.Chmod(dir, 0o755)
	ln, err := Listen(filepath.Join(dir, "c.sock"))
	if err != nil {
		t.Fatal(err)
	}
	ln.Close()
	st, _ := os.Stat(dir)
	if st.Mode().Perm() != 0o700 {
		t.Fatalf("dir left at %v", st.Mode().Perm())
	}
}

func TestPathTooLong(t *testing.T) {
	_, err := Listen("/" + strings.Repeat("a", 120) + "/c.sock")
	var e *PathTooLongError
	if !errors.As(err, &e) {
		t.Fatalf("want PathTooLongError, got %v", err)
	}
}

func TestNoHub(t *testing.T) {
	if err := Call(filepath.Join(shortDir(t), "none.sock"), Request{Op: "x"}, func(Reply) (bool, bool) { return false, false }); err == nil {
		t.Fatal("expected an error")
	}
}
