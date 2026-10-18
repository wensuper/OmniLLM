package e2e

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// pipe is an in-memory MsgConn pair. tap, when set, sees (and may alter)
// every message going a→b, standing in for a malicious relay.
type pipe struct {
	in     chan []byte
	out    chan []byte
	tap    func([]byte) []byte
	closed chan struct{}
	once   sync.Once
}

func newPipe() (*pipe, *pipe) {
	ab, ba := make(chan []byte, 64), make(chan []byte, 64)
	c := make(chan struct{})
	return &pipe{in: ba, out: ab, closed: c}, &pipe{in: ab, out: ba, closed: c}
}

func (p *pipe) ReadMessage(ctx context.Context) ([]byte, error) {
	select {
	case b := <-p.in:
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.closed:
		return nil, errors.New("closed")
	}
}

func (p *pipe) WriteMessage(ctx context.Context, b []byte) error {
	if p.tap != nil {
		b = p.tap(append([]byte{}, b...))
		if b == nil {
			return nil
		}
	}
	select {
	case p.out <- append([]byte{}, b...):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *pipe) Close() error { p.once.Do(func() { close(p.closed) }); return nil }

func mustKey(t *testing.T) KeyPair {
	t.Helper()
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

type result struct {
	s   *Session
	err error
}

func handshake(t *testing.T, dev, hub *pipe, devKey, hubKey KeyPair, dialHubPub, dialPSK, acceptPSK []byte, auth Authorizer) (*Session, *Session, error, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mb := []byte("mailbox-0123456789")
	ch := make(chan result, 1)
	go func() {
		s, err := Accept(ctx, hub, mb, hubKey, acceptPSK, auth)
		ch <- result{s, err}
	}()
	ds, derr := Dial(ctx, dev, mb, devKey, dialHubPub, dialPSK)
	if derr != nil {
		dev.Close()
	}
	r := <-ch
	return ds, r.s, derr, r.err
}

func TestHandshakeAndRoundTrip(t *testing.T) {
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	ds, hs, derr, herr := handshake(t, dev, hub, dk, hk, hk.Public, nil, nil, func(p []byte) error {
		if !KeyEqual(p, dk.Public) {
			return errors.New("unknown device")
		}
		return nil
	})
	if derr != nil || herr != nil {
		t.Fatalf("handshake: %v %v", derr, herr)
	}
	if !KeyEqual(hs.PeerStatic(), dk.Public) {
		t.Fatal("hub sees wrong device key")
	}
	if SAS(ds.ChannelBinding()) != SAS(hs.ChannelBinding()) {
		t.Fatal("SAS differs between ends")
	}
	ctx := context.Background()
	for _, size := range []int{0, 1, 1000, 1019, 1020, 5000, 65000, 70000, 300000} {
		msg := bytes.Repeat([]byte{byte(size)}, size)
		if err := ds.Send(ctx, msg); err != nil {
			t.Fatal(err)
		}
		got, err := hs.Recv(ctx)
		if err != nil || !bytes.Equal(got, msg) {
			t.Fatalf("size %d: err=%v equal=%v", size, err, bytes.Equal(got, msg))
		}
		if err := hs.Send(ctx, msg); err != nil {
			t.Fatal(err)
		}
		got, err = ds.Recv(ctx)
		if err != nil || !bytes.Equal(got, msg) {
			t.Fatalf("reverse size %d: %v", size, err)
		}
	}
}

func TestWrongHubKeyFails(t *testing.T) {
	dev, hub := newPipe()
	dk, hk, other := mustKey(t), mustKey(t), mustKey(t)
	_, _, derr, herr := handshake(t, dev, hub, dk, hk, other.Public, nil, nil, nil)
	if derr == nil && herr == nil {
		t.Fatal("handshake with the wrong hub key must fail")
	}
}

func TestUnknownDeviceRejected(t *testing.T) {
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	_, _, _, herr := handshake(t, dev, hub, dk, hk, hk.Public, nil, nil, func([]byte) error { return errors.New("no") })
	if herr == nil {
		t.Fatal("hub must reject an unknown device key")
	}
}

func TestWrongPairingSecretFails(t *testing.T) {
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	good, bad := RandomBytes(32), RandomBytes(32)
	ds, hs, derr, herr := handshake(t, dev, hub, dk, hk, hk.Public, bad, good, nil)
	if derr == nil {
		// IKpsk2 mixes the psk into message 2, so the device must fail there.
		t.Fatal("device must fail with the wrong pairing secret")
	}
	if herr == nil && hs != nil {
		// The hub may complete its side, but it must never decrypt anything.
		_ = ds
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		if _, err := hs.Recv(ctx); err == nil {
			t.Fatal("hub decrypted a message from a device with the wrong secret")
		}
	}
}

func TestPairingWithRightSecret(t *testing.T) {
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	psk := RandomBytes(32)
	ds, hs, derr, herr := handshake(t, dev, hub, dk, hk, hk.Public, psk, psk, nil)
	if derr != nil || herr != nil {
		t.Fatal(derr, herr)
	}
	if err := ds.Send(context.Background(), []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if got, err := hs.Recv(context.Background()); err != nil || string(got) != "hello" {
		t.Fatal(err)
	}
}

func TestPSKMismatchDirectionHubWithoutPSK(t *testing.T) {
	// A device holding a pairing secret talking to a normal mailbox (and the
	// reverse) must not complete.
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	_, _, derr, herr := handshake(t, dev, hub, dk, hk, hk.Public, RandomBytes(32), nil, nil)
	if derr == nil && herr == nil {
		t.Fatal("psk on one side only must fail")
	}
}

func TestTamperReplayReorderAreFatal(t *testing.T) {
	for _, mode := range []string{"flip", "replay", "drop", "truncate"} {
		t.Run(mode, func(t *testing.T) {
			dev, hub := newPipe()
			dk, hk := mustKey(t), mustKey(t)
			ds, hs, derr, herr := handshake(t, dev, hub, dk, hk, hk.Public, nil, nil, nil)
			if derr != nil || herr != nil {
				t.Fatal(derr, herr)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var last []byte
			n := 0
			dev.tap = func(b []byte) []byte {
				n++
				switch mode {
				case "flip":
					b[len(b)/2] ^= 1
				case "replay":
					if n == 2 {
						return last
					}
				case "drop":
					if n == 1 {
						last = b
						return nil
					}
				case "truncate":
					return b[:len(b)-1]
				}
				last = b
				return b
			}
			_ = ds.Send(ctx, []byte("one"))
			if mode == "replay" || mode == "drop" {
				_ = ds.Send(ctx, []byte("two"))
			}
			if mode == "replay" {
				if _, err := hs.Recv(ctx); err != nil {
					t.Fatalf("first message should decrypt: %v", err)
				}
			}
			if _, err := hs.Recv(ctx); err == nil {
				t.Fatalf("%s: tampered stream must not decrypt", mode)
			}
		})
	}
}

func TestPaddingHidesSmallSizes(t *testing.T) {
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	ds, _, derr, herr := handshake(t, dev, hub, dk, hk, hk.Public, nil, nil, nil)
	if derr != nil || herr != nil {
		t.Fatal(derr, herr)
	}
	var sizes []int
	dev.tap = func(b []byte) []byte { sizes = append(sizes, len(b)); return b }
	for _, m := range []string{"a", "hello world", string(bytes.Repeat([]byte("x"), 900))} {
		_ = ds.Send(context.Background(), []byte(m))
	}
	for _, s := range sizes {
		if s != 1024+16 {
			t.Fatalf("small messages must all be one bucket, got %v", sizes)
		}
	}
}

func TestOversizedMessageRefused(t *testing.T) {
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	ds, _, _, _ := handshake(t, dev, hub, dk, hk, hk.Public, nil, nil, nil)
	if err := ds.Send(context.Background(), make([]byte, MaxMessage+1)); err == nil {
		t.Fatal("must refuse a message over the limit")
	}
}

func TestBadHandshakeInputs(t *testing.T) {
	hk := mustKey(t)
	for _, msg := range [][]byte{nil, {1}, make([]byte, 31), make([]byte, 2000), bytes.Repeat([]byte{0}, 96)} {
		dev, hub := newPipe()
		go func() { _ = dev.WriteMessage(context.Background(), msg) }()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if _, err := Accept(ctx, hub, []byte("m"), hk, nil, nil); err == nil {
			t.Fatalf("accepted garbage handshake of %d bytes", len(msg))
		}
		cancel()
	}
}

func TestLowOrderKeyRejected(t *testing.T) {
	dev, hub := newPipe()
	dk, hk := mustKey(t), mustKey(t)
	zero := make([]byte, 32)
	_, _, derr, _ := handshake(t, dev, hub, dk, hk, zero, nil, nil, nil)
	if derr == nil {
		t.Fatal("an all-zero hub key must be rejected")
	}
}

func TestSASFormat(t *testing.T) {
	for i := 0; i < 100; i++ {
		s := SAS(RandomBytes(32))
		if len(s) != 6 {
			t.Fatalf("SAS %q", s)
		}
	}
}
